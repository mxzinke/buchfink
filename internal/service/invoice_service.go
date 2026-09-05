package service

import (
	"context"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/invoice"
)

// InvoiceService manages outgoing invoices, their booking and the ZUGFeRD /
// Typst rendering.
type InvoiceService struct {
	invoiceRepo  domain.InvoiceRepository
	contactRepo  domain.ContactRepository
	settingsRepo domain.SettingsRepository
	numberRepo   domain.NumberRangeRepository
	postingSvc   *PostingService
	auditRepo    domain.AuditRepository
	receiptSvc   *ReceiptService
	renderer     *invoice.Renderer

	// Die Verdrahtung aus Welle 5b. Sie ist optional: ohne sie stellt der
	// Dienst weiter Rechnungen aus, nur ohne Transaktionsklammer,
	// Lückenbericht und Rechnungsverbund. Ein Pflichtparameter im
	// Konstruktor hätte jeden Aufrufer erfasst, auch die, die nichts davon
	// brauchen.
	txRunner       domain.TxRunner
	gapRepo        domain.NumberGapRepository
	groupRepo      domain.InvoiceGroupRepository
	fiscalYearRepo domain.FiscalYearRepository
	bankRepo       domain.BankRepository

	// vatIDs ist die Bestätigungsabfrage der USt-IdNr. (Welle 5c). Ebenso
	// optional: ohne sie stellt der Dienst aus wie zuvor.
	vatIDs vatIDConfirmer
}

// InvoiceRegistry bündelt, was das Rechnungswesen der Welle 5b zusätzlich
// braucht.
type InvoiceRegistry struct {
	// Tx klammert Nummernvergabe, Rechnung und Buchung zu einer Transaktion.
	Tx domain.TxRunner
	// Gaps hält die Gründe verbrauchter, aber nicht belegter Nummern fest.
	Gaps domain.NumberGapRepository
	// Groups ist der Rechnungsverbund aus Abschlägen und Schlussrechnung.
	Groups domain.InvoiceGroupRepository
	// FiscalYears liefert den Vorjahresumsatz für § 27 Abs. 38 UStG.
	FiscalYears domain.FiscalYearRepository
	// Bank ist der Kontoauszug. Die Vereinnahmung einer Anzahlung kommt in der
	// Praxis aus dem Bankimport, und ohne ihn bliebe der Umsatz dort nach der
	// Buchung als offen stehen — bis ihn jemand ein zweites Mal zuordnet.
	Bank domain.BankRepository
}

// SetRegistry wires the Welle-5b collaborators.
func (s *InvoiceService) SetRegistry(r InvoiceRegistry) {
	s.txRunner = r.Tx
	s.gapRepo = r.Gaps
	s.groupRepo = r.Groups
	s.fiscalYearRepo = r.FiscalYears
	s.bankRepo = r.Bank
}

// SetDocumentPipeline wires in what an issued invoice needs to become a Beleg of
// its own: the renderer that produces the hybrid PDF and the Beleg service that
// files and seals it.
func (s *InvoiceService) SetDocumentPipeline(receiptSvc *ReceiptService, renderer *invoice.Renderer) {
	s.receiptSvc = receiptSvc
	s.renderer = renderer
}

// NewInvoiceService creates the invoice service.
func NewInvoiceService(
	invoiceRepo domain.InvoiceRepository,
	contactRepo domain.ContactRepository,
	settingsRepo domain.SettingsRepository,
	numberRepo domain.NumberRangeRepository,
	postingSvc *PostingService,
	auditRepo domain.AuditRepository,
) *InvoiceService {
	return &InvoiceService{
		invoiceRepo:  invoiceRepo,
		contactRepo:  contactRepo,
		settingsRepo: settingsRepo,
		numberRepo:   numberRepo,
		postingSvc:   postingSvc,
		auditRepo:    auditRepo,
	}
}

// GetInvoices returns the invoices of a fiscal year.
func (s *InvoiceService) GetInvoices(ctx context.Context, fiscalYear int) ([]domain.Invoice, error) {
	return s.invoiceRepo.FindAll(ctx, fiscalYear)
}

// Issue finalises an invoice: it assigns the consecutive invoice number, books
// the receivable and stores the document.
//
// Issuing and booking are one step, and seit dieser Welle auch eine
// Transaktion. § 14 Abs. 4 Nr. 4 UStG verlangt eine einmalige, fortlaufende
// Nummer, und GoBD Rz. 42 verlangt sie lückenlos — vorher vergab Buchfink die
// Nummer in einer eigenen Transaktion und buchte danach, und eine
// fehlgeschlagene Buchung ließ eine verbrauchte Nummer ohne Rechnung zurück.
//
// Das Dokument entsteht danach und außerhalb der Transaktion: es liegt als
// Datei auf der Platte, und eine Datei kann keine Datenbanktransaktion
// zurücknehmen. Scheitert es, bleibt die Rechnung mit Nummer und Buchung im
// Zustand „Dokument fehlt" stehen und lässt sich nachholen — die Nummer ist
// dann nicht verloren.
func (s *InvoiceService) Issue(ctx context.Context, inv *domain.Invoice) error {
	// Über diesen Weg kommt eine ganze `domain.Invoice` aus der Oberfläche,
	// samt der Felder, die zu anderen Rechnungsarten gehören. Sie werden hier
	// verworfen — die Prüfung der Art übernimmt ensureKind.
	dropForeignKindFields(inv)
	return s.issueAs(ctx, inv, domain.InvoiceKindInvoice, nil)
}

// issueAs ist Issue für eine bestimmte Rechnungsart, mit einem Nebenschritt
// innerhalb der Nummernklammer.
//
// Die Art ist ein Parameter des Weges und keine Angabe des Aufrufers: jede Art
// außer der gewöhnlichen Rechnung hat eigene Invarianten, und die erzwingt
// jeweils nur ihr eigener Weg (siehe ensureKind).
//
// Der Nebenschritt ist alles, was zur Rechnung gehört, ohne eine Buchung zu
// sein — bei der Abschlagsrechnung der offene Posten des Verbunds. Er läuft
// innerhalb der Transaktion, weil er sonst hinter der Dokumenterzeugung stünde:
// scheitert das Dokument, gäbe es die Rechnung mit Nummer, aber ohne ihren
// offenen Posten, und keine Wiederholung holte ihn nach.
func (s *InvoiceService) issueAs(
	ctx context.Context, inv *domain.Invoice, kind domain.InvoiceKind, within func(context.Context) error,
) error {
	if inv.Status.IsIssued() || inv.JournalEntryID != nil {
		return fmt.Errorf("Rechnung %s ist bereits ausgestellt und gebucht", inv.InvoiceNumber)
	}

	contact, err := s.prepareForIssue(ctx, inv, kind)
	if err != nil {
		return err
	}
	if err := s.numberAndBook(ctx, inv, contact, within); err != nil {
		return err
	}
	if err := s.attachDocument(ctx, inv, contact); err != nil {
		return err
	}

	s.logIssued(ctx, inv)
	return nil
}

// logIssued schreibt den Protokolleintrag über eine ausgestellte Rechnung.
//
// Er steht an einer Stelle, weil es drei Wege in die Ausstellung gibt —
// gewöhnliche Rechnung, Schlussrechnung, Korrekturdokument. Jeder von ihnen
// braucht denselben Eintrag; drei Fassungen davon wären zwei zu viel, und eine
// davon fehlte.
func (s *InvoiceService) logIssued(ctx context.Context, inv *domain.Invoice) {
	if s.auditRepo == nil {
		return
	}
	recipient := inv.ContactName
	if recipient == "" {
		recipient = "Barverkauf"
	}
	_ = s.auditRepo.Log(ctx, domain.AuditActionCreate, "INVOICE", fmt.Sprintf("%d", inv.ID),
		fmt.Sprintf("%s %s über %s %s an %s ausgestellt",
			inv.ResolvedKind().Label(), inv.InvoiceNumber, inv.GrossAmount, inv.Currency, recipient))
}

// prepareForIssue fills the defaults and runs every check that can be answered
// before a number is spent.
//
// Alles, was hier scheitert, kostet keine Nummer. Das ist der Grund für den
// Schnitt: eine Rechnung, die an ihren eigenen Stammdaten scheitert, darf den
// Nummernkreis nicht weiterzählen.
func (s *InvoiceService) prepareForIssue(
	ctx context.Context, inv *domain.Invoice, allowed ...domain.InvoiceKind,
) (*domain.Contact, error) {
	if err := ensureKind(inv, allowed...); err != nil {
		return nil, err
	}
	// Leere Listen statt nil: die Rechnung geht als JSON an die Oberfläche
	// zurück, und dort wird sie ohne Umweg gelesen.
	inv.EnsureLists()
	if inv.Date == "" {
		inv.Date = time.Now().Format("2006-01-02")
	}
	if inv.ServiceDateFrom == "" {
		inv.ServiceDateFrom = inv.Date
	}
	if inv.ServiceDateTo == "" {
		inv.ServiceDateTo = inv.ServiceDateFrom
	}
	if inv.Currency == "" {
		inv.Currency = "EUR"
	}
	if inv.TaxTreatment == "" {
		inv.TaxTreatment = domain.TaxTreatmentDomestic
	}
	if inv.FiscalYear == 0 {
		inv.FiscalYear = domain.GetFiscalYearForDate(inv.Date, s.fiscalYearStartMonth(ctx))
	}

	var contact *domain.Contact
	if inv.ContactID != 0 {
		var err error
		contact, err = s.contactRepo.FindByID(ctx, inv.ContactID)
		if err != nil {
			return nil, fmt.Errorf("Rechnungsempfänger konnte nicht geladen werden: %w", err)
		}
		if contact.Type != domain.ContactTypeCustomer {
			return nil, fmt.Errorf("%s ist als Lieferant angelegt und kann keine Ausgangsrechnung erhalten", contact.Name)
		}
		inv.ContactName = contact.Name
		if inv.EInvoiceProfile == "" {
			inv.EInvoiceProfile = contact.ResolvedEInvoiceProfile()
		}
	} else if !inv.SmallAmount {
		// Ohne Empfänger geht nur die Kleinbetragsrechnung (§ 33 UStDV); alles
		// andere braucht ein Personenkonto, gegen das die Forderung läuft.
		return nil, fmt.Errorf("Rechnungsempfänger fehlt")
	}
	// Die Kleinbetragsrechnung ohne Empfänger geht als PDF hinaus und nicht als
	// E-Rechnung.
	//
	// EN 16931 verlangt den Namen des Erwerbers (BR-07); § 33 UStDV erlässt ihn.
	// Einen Namen zu erfinden, um die Norm zu bedienen, hieße eine Pflichtangabe
	// zu behaupten, die es nicht gibt — der strukturierte Datensatz wäre falsch,
	// und zwar maschinenlesbar falsch. Der Ausweg steht im Gesetz selbst:
	// Kleinbetragsrechnungen sind von der Pflicht zur E-Rechnung ausgenommen
	// (§ 33 UStDV, BMF vom 15.10.2024, Rz. 21).
	if inv.ContactID == 0 && inv.SmallAmount {
		inv.EInvoiceProfile = domain.EInvoiceProfilePDFOnly
	}
	if inv.EInvoiceProfile == "" {
		inv.EInvoiceProfile = domain.EInvoiceProfileZUGFeRD
	}
	// Eine mitgelieferte Rechnungsnummer wird nicht übernommen: sie kommt aus
	// dem Nummernkreis und aus nichts sonst (§ 14 Abs. 4 Nr. 4 UStG). Vorher
	// ließ numberAndBook eine vorbelegte Nummer stehen und übersprang die
	// Vergabe — der Nummernkreis wäre über die Oberfläche umgehbar gewesen.
	if inv.InvoiceNumber != "" {
		return nil, fmt.Errorf(
			"die Rechnungsnummer wird beim Ausstellen aus dem Nummernkreis vergeben; %q ist vorbelegt",
			inv.InvoiceNumber)
	}

	if inv.DueDate == "" {
		days := inv.Terms.DueDays
		if days <= 0 && contact != nil {
			days = contact.PaymentTermsDays
		}
		if days <= 0 {
			days = 14
		}
		if t, err := time.Parse("2006-01-02", inv.Date); err == nil {
			inv.DueDate = t.AddDate(0, 0, days).Format("2006-01-02")
		} else {
			inv.DueDate = inv.Date
		}
	}

	for i := range inv.Items {
		if inv.Items[i].Position == 0 {
			inv.Items[i].Position = i + 1
		}
	}
	inv.Recalculate()

	if err := inv.Validate(); err != nil {
		return nil, err
	}
	if contact != nil {
		if err := s.validateTaxTreatment(inv, contact); err != nil {
			return nil, err
		}
		// Die Bestätigung der USt-IdNr. steht hinter der Stammdatenprüfung und
		// vor der Nummernvergabe: sie ist die letzte Frage, die noch mit einem
		// Nein beantwortet werden darf, ohne eine Nummer zu verbrauchen.
		if err := s.ensureVatIDConfirmed(ctx, inv, contact); err != nil {
			return nil, err
		}
	}
	// Erst die Stammdaten, dann der Steuerausweis: passt der Steuerfall gar
	// nicht zum Empfänger, ist das die Auskunft, die weiterhilft — der Hinweis
	// auf § 14c käme dann zu einem Steuerfall, den der Anwender ohnehin ändern
	// muss.
	if err := ensureNoUnlawfulTax(inv); err != nil {
		return nil, err
	}
	if err := s.validateSmallAmount(inv); err != nil {
		return nil, err
	}

	seller, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("Unternehmensdaten konnten nicht geladen werden: %w", err)
	}
	if err := inv.ValidateParties(seller.TaxNumber, seller.VatID, contact); err != nil {
		return nil, err
	}
	if err := s.validateProfile(ctx, inv, contact); err != nil {
		return nil, err
	}
	if err := validateRuleset(inv, seller, contact); err != nil {
		return nil, err
	}
	return contact, nil
}

// otherWayForKind nennt den Weg, über den eine Rechnungsart entsteht.
//
// Er steht in der Meldung, weil die Antwort auf „so nicht" die Frage „wie
// dann?" nach sich zieht — und weil der Rechnungsdialog genau eine Art-Auswahl
// anbietet: wer dort „Abschlag" wählt, soll den Verbund genannt bekommen und
// nicht nur ein Nein.
var otherWayForKind = map[domain.InvoiceKind]string{
	domain.InvoiceKindAdvance: "Eine Abschlagsrechnung entsteht im Rechnungsverbund. Nur dort bekommt sie " +
		"ihren offenen Posten, zählt gegen den vereinbarten Gesamtbetrag und wird in der Schlussrechnung " +
		"abgesetzt (§ 14 Abs. 5 Satz 2 UStG)",
	domain.InvoiceKindFinal: "Eine Schlussrechnung entsteht über den Rechnungsverbund. Nur dort setzt sie " +
		"die vereinnahmten Anzahlungen ab; ohne die Absetzung wäre die Steuer zweimal ausgewiesen " +
		"(§ 14c Abs. 1 UStG)",
	domain.InvoiceKindCorrection: "Eine Rechnungskorrektur entsteht über „Rechnung berichtigen“. Nur dort " +
		"wird die berichtigte Rechnung storniert und der Bezug auf sie gesetzt (BG-3)",
	domain.InvoiceKindCancellation: "Eine Stornorechnung entsteht über „Rechnung stornieren“. Nur dort wird " +
		"die Ursprungsbuchung zurückgenommen und die stornierte Rechnung als storniert gekennzeichnet",
}

// ensureKind hält eine Rechnung auf der Art fest, für die der eingeschlagene
// Weg ihre Invarianten erzwingt.
//
// Der Grund ist der öffentliche Weg: über die Bridge kommt eine ganze
// `domain.Invoice` aus der Oberfläche, und in ihr steht auch die Art. Ohne
// diese Prüfung ließe sich mit `Kind = advance` eine nummerierte
// Abschlagsrechnung ohne offenen Posten und ohne Verbund ausstellen — nie
// vereinnahmbar, nie absetzbar, nicht in der OP-Liste —, und mit `Kind = final`
// samt PrepaidAmount ein Dokument, das BT-113 und einen geminderten Zahlbetrag
// trägt, während die volle Forderung ohne Auflösung der Anzahlungen gebucht
// wird: Dokument und Buchung sagten Verschiedenes, und die Steuer wäre zweimal
// ausgewiesen (§ 14c Abs. 1 UStG). Mit `Kind = correction` schließlich stünde
// ein Bezug auf eine Rechnung auf dem Dokument, die niemand storniert hat.
//
// Erlaubt sind mehrere Arten, weil die berichtigte Schlussrechnung als
// Rechnungskorrektur (Typcode 384) über den Weg der Schlussrechnung geht: sie
// ist ein Korrekturdokument und muss trotzdem die Anzahlungen absetzen.
func ensureKind(inv *domain.Invoice, allowed ...domain.InvoiceKind) error {
	if len(allowed) == 0 {
		return fmt.Errorf("für diesen Weg ist keine Rechnungsart vorgesehen")
	}
	if inv.Kind == "" {
		inv.Kind = allowed[0]
	}
	for _, kind := range allowed {
		if inv.Kind == kind {
			return nil
		}
	}
	hint := otherWayForKind[inv.Kind]
	if hint == "" {
		hint = fmt.Sprintf("Die Art %q gibt es beim Ausstellen nicht", inv.Kind)
	}
	return fmt.Errorf("%s lässt sich hier nicht ausstellen. %s", inv.Kind.Label(), hint)
}

// dropForeignKindFields verwirft die Angaben, die einer anderen Rechnungsart
// gehören als der, die entsteht.
//
// Sie steht an den beiden Stellen, an denen eine ganze Rechnung aus der
// Oberfläche kommt (Issue und CorrectInvoice). Verworfen und nicht bloß
// übergangen: gespeichert würden sie sonst mit, und die Rechnung trüge später
// einen Verbund, eine Anzahlungssumme oder einen Bezug, den es zu ihr nicht
// gibt. Was der Verbundweg selbst braucht, setzt er danach — die Zuordnung zum
// Verbund, die abgesetzten Anzahlungen und ihre Bezüge kommen aus dem Verbund
// und nicht aus der Maske.
func dropForeignKindFields(inv *domain.Invoice) {
	inv.GroupID = nil
	inv.PrepaidAmount = 0
	// Leer und nicht nil: die Liste geht als JSON an die Oberfläche.
	inv.PrecedingRefs = []domain.InvoiceReference{}
	// Der Verweis auf das eigene Storno entsteht beim Stornieren und nirgends
	// sonst; der Bezug auf die berichtigte Rechnung beim Berichtigen.
	inv.CancelledByInvoiceID = nil
	inv.CorrectsInvoiceID = nil
	inv.CorrectsInvoiceNumber = ""
	inv.CorrectsInvoiceDate = ""
}

// placeholderInvoiceNumber steht in der Prüfung an der Stelle der noch nicht
// vergebenen Nummer.
//
// Sie wird gebraucht, weil BT-1 belegt sein muss (BR-02), und sie darf es sein,
// weil keine Regel der EN 16931 und keine der deutschen Ausprägung den Inhalt
// der Nummer bewertet — nur ihr Vorhandensein.
const placeholderInvoiceNumber = "ENTWURF"

// validateRuleset prüft die Rechnung gegen EN 16931 und, bei einer XRechnung,
// gegen die deutsche Ausprägung — bevor eine Nummer verbraucht ist.
//
// Vorher lief die Prüfung erst beim Erzeugen des Dokuments, also hinter der
// Transaktion aus Nummernvergabe und Buchung. Ein Regelverstoß — eine
// XRechnung ohne Ansprechpartner (BR-DE-2 ff.), eine Rechnung ohne
// Bankverbindung (BG-16) — hinterließ dann eine gebuchte Rechnung mit
// verbrauchter Nummer im Zustand „Dokument fehlt". War der Verstoß nicht durch
// Stammdaten heilbar, blieb nur das Storno mit einer zweiten Nummer.
//
// Geprüft wird derselbe Datensatz, der später hinausgeht: gebaut, geschrieben,
// wieder gelesen und gegen das Regelwerk gehalten. Nur die Nummer ist eine
// andere, und sie ändert an keiner Regel etwas.
func validateRuleset(inv *domain.Invoice, seller *domain.CompanySettings, contact *domain.Contact) error {
	profile := inv.EInvoiceProfile
	if profile == "" || profile == domain.EInvoiceProfilePDFOnly {
		return nil
	}
	// Die Kleinbetragsrechnung ohne Empfänger geht als reines PDF hinaus; sie
	// hat keinen strukturierten Datensatz, der zu prüfen wäre (§ 33 UStDV).
	if inv.SmallAmount && contact == nil {
		return nil
	}
	draft := *inv
	draft.InvoiceNumber = placeholderInvoiceNumber
	_, _, err := invoice.RenderInvoiceXML(&draft, seller, contact, profile)
	return err
}

// smallAmountExcludedByLaw nennt die Steuerfälle, die § 33 Satz 2 UStDV
// ausdrücklich von der Kleinbetragsrechnung ausnimmt.
//
// Die Vorschrift zählt drei auf: den Fernverkauf (§ 3c UStG), die
// innergemeinschaftliche Lieferung (§ 6a UStG) und die Steuerschuldnerschaft
// des Leistungsempfängers (§ 13b UStG). Der Fernverkauf fehlt hier, weil er in
// Buchfink kein eigener Steuerfall ist; die beiden anderen sind es. Der Grund
// der Ausnahme ist derselbe: mit den verkürzten Angaben fehlten gerade die
// Angaben, an denen die Rechtsfolge hängt — die USt-IdNr. des Empfängers und
// der Hinweis auf die Steuerschuldnerschaft.
var smallAmountExcludedByLaw = map[domain.TaxTreatment]string{
	domain.TaxTreatmentIntraCommunitySupply: "die innergemeinschaftliche Lieferung (§ 6a UStG)",
	domain.TaxTreatmentReverseChargeSupply:  "die Steuerschuldnerschaft des Leistungsempfängers (§ 13b UStG)",
}

// validateSmallAmount enforces the two conditions of § 33 UStDV.
//
// Die Grenze ist datiert (250 € seit 2017, davor 150 €) und wird deshalb aus
// den Steuerparametern zum Rechnungsdatum gelesen.
//
// Beim Steuerfall trennt die Prüfung zwei Dinge, die vorher eine Meldung waren.
// Zwei Fälle verbietet das Gesetz (§ 33 Satz 2 UStDV, siehe
// smallAmountExcludedByLaw). Die übrigen — Ausfuhr, steuerfreier Umsatz,
// nicht steuerbarer Umsatz, Nullsteuersatz — wären nach dem Wortlaut zulässig;
// Buchfink bietet sie trotzdem nicht an, und die Meldung sagt das als eigene
// Entscheidung statt sich auf eine Norm zu berufen, die sie nicht deckt. Der
// Grund ist der verkürzte Datensatz: er kennt Bruttobetrag und Steuersatz, aber
// keinen Empfänger und keinen Platz für den Hinweis auf die Steuerbefreiung,
// den § 33 Satz 1 Nr. 4 UStDV in diesen Fällen an die Stelle des Steuersatzes
// setzt.
func (s *InvoiceService) validateSmallAmount(inv *domain.Invoice) error {
	if !inv.SmallAmount {
		return nil
	}
	if excluded, ok := smallAmountExcludedByLaw[inv.TaxTreatment]; ok {
		return fmt.Errorf(
			"für %s ist die Kleinbetragsrechnung nicht zulässig (§ 33 Satz 2 UStDV). "+
				"Die Rechnung braucht die vollständigen Angaben", excluded)
	}
	if inv.TaxTreatment != domain.TaxTreatmentDomestic {
		return fmt.Errorf(
			"Buchfink bietet die Kleinbetragsrechnung nur für den steuerpflichtigen Inlandsumsatz an; "+
				"beim Steuerfall %q braucht die Rechnung die vollständigen Angaben einschließlich des "+
				"Hinweises auf die Steuerbefreiung", inv.TaxTreatment)
	}
	params, err := accounting.TaxParametersFor(inv.Date)
	if err != nil {
		return err
	}
	if inv.GrossAmount > params.SmallAmountInvoiceLimit {
		return fmt.Errorf(
			"die Rechnung über %s € übersteigt die Grenze der Kleinbetragsrechnung von %s € (§ 33 UStDV)",
			inv.GrossAmount, params.SmallAmountInvoiceLimit)
	}
	return nil
}

// validateProfile checks whether the chosen output format may be used.
//
// Zwei Fragen: darf eine sonstige Rechnung überhaupt noch hinausgehen
// (§ 27 Abs. 38 UStG, mit dem Vorjahresumsatz als Bedingung ab 2027), und liegt
// bei einer XRechnung die Leitweg-ID vor, ohne die sie im Behördennetz nicht
// zugestellt werden kann (BR-DE-15).
func (s *InvoiceService) validateProfile(ctx context.Context, inv *domain.Invoice, contact *domain.Contact) error {
	// Zuerst: ist das Profil überhaupt eines, das Buchfink erzeugt? Ein
	// unbekanntes fiel bisher durch diesen switch und wurde vom Renderer still
	// als ZUGFeRD behandelt — die Rechnung ginge in einem anderen Format hinaus
	// als dem, das an ihr steht.
	if err := inv.EInvoiceProfile.Validate(); err != nil {
		return err
	}
	switch inv.EInvoiceProfile {
	case domain.EInvoiceProfileXRechnungCII:
		if contact == nil || contact.LeitwegID == "" {
			name := "der Empfänger"
			if contact != nil {
				name = contact.Name
			}
			return fmt.Errorf(
				"eine XRechnung braucht die Leitweg-ID des Empfängers (BT-10); bei %s ist keine hinterlegt", name)
		}
	case domain.EInvoiceProfilePDFOnly:
		// Die Kleinbetragsrechnung ohne Empfänger bleibt zulässig, auch nach dem
		// Ende der Übergangsfrist: § 33 UStDV nimmt sie von der Pflicht zur
		// E-Rechnung aus, und ohne Erwerber ließe sich der strukturierte
		// Datensatz gar nicht normgerecht erzeugen (BR-07).
		if inv.SmallAmount && contact == nil {
			return nil
		}
		if accounting.EInvoiceIssueTransitionFor(inv.Date, s.priorYearRevenue(ctx, inv.FiscalYear)) ==
			accounting.EInvoiceTransitionExpired {
			return fmt.Errorf(
				"zum %s darf keine sonstige Rechnung ohne strukturierten Datensatz mehr ausgestellt werden "+
					"(§ 27 Abs. 38 UStG). Bitte ein E-Rechnungsformat für %s wählen",
				domain.GermanDate(inv.Date), inv.ContactName)
		}
	}
	return nil
}

// priorYearRevenue liest den Gesamtumsatz des Vorjahres. Null heißt „nicht
// erfasst" und lässt die Übergangsregel greifen — siehe
// accounting.EInvoiceIssueTransitionFor.
func (s *InvoiceService) priorYearRevenue(ctx context.Context, fiscalYear int) domain.Cents {
	if s.fiscalYearRepo == nil {
		return 0
	}
	fy, err := s.fiscalYearRepo.FindByYear(ctx, fiscalYear)
	if err != nil || fy == nil {
		return 0
	}
	return fy.PriorYearRevenue
}

// numberAndBook spends the number, writes the invoice and appends the booking —
// all three in one transaction.
func (s *InvoiceService) numberAndBook(
	ctx context.Context, inv *domain.Invoice, contact *domain.Contact, within func(context.Context) error,
) error {
	format := s.numberFormat(ctx)
	var allocated int64

	err := s.runInTx(ctx, func(ctx context.Context) error {
		seq, err := s.numberRepo.Allocate(ctx, domain.NumberRangeInvoice, inv.FiscalYear)
		if err != nil {
			return fmt.Errorf("Rechnungsnummer konnte nicht vergeben werden: %w", err)
		}
		allocated = seq
		inv.InvoiceNumber = domain.FormatInvoiceNumberWith(format, inv.FiscalYear, seq)
		inv.Status = domain.InvoiceStatusPendingDocument

		// Die Abschlagsrechnung wird beim Ausstellen nicht gebucht: die Steuer
		// entsteht erst mit der Vereinnahmung (§ 13 Abs. 1 Nr. 1 Buchst. a
		// Satz 4 UStG). Sie bekommt trotzdem Nummer und Datensatz — sie ist
		// eine vollwertige Rechnung nach § 14 Abs. 5 Satz 1 UStG.
		if !inv.ResolvedKind().BooksOnIssue() {
			if err := s.invoiceRepo.Save(ctx, inv); err != nil {
				return err
			}
			return runWithin(ctx, within)
		}
		if err := s.invoiceRepo.Save(ctx, inv); err != nil {
			return err
		}
		// Ohne Empfänger ist es der Barverkauf einer Kleinbetragsrechnung: er
		// bucht gegen das Zahlungsmittel statt gegen ein Personenkonto, das es
		// nicht gibt (§ 33 UStDV lässt den Empfänger weg).
		var entry *domain.JournalEntry
		if contact == nil {
			entry, err = s.postingSvc.PostCashSale(ctx, inv, inv.PaymentAccount)
		} else {
			entry, err = s.postingSvc.PostOutgoingInvoice(ctx, inv, contact)
		}
		if err != nil {
			return err
		}
		inv.JournalEntryID = &entry.ID
		if err := s.invoiceRepo.Save(ctx, inv); err != nil {
			return err
		}
		return runWithin(ctx, within)
	})
	if err != nil {
		s.recordNumberGap(ctx, inv, allocated, err)
		// Die Transaktion ist zurückgerollt: die Rechnung gibt es nicht, und die
		// Nummer ist wieder frei. Die im Speicher gesetzten Schlüssel müssen
		// mit — sonst schriebe ein zweiter Versuch mit GORMs Save ein UPDATE auf
		// eine Zeile, die es nicht gibt, und meldete Erfolg.
		resetAfterRollback(inv, allocated)
		return err
	}
	return nil
}

// runWithin ruft den Nebenschritt der Nummernklammer auf, sofern es einen gibt.
func runWithin(ctx context.Context, within func(context.Context) error) error {
	if within == nil {
		return nil
	}
	return within(ctx)
}

// resetAfterRollback nimmt zurück, was die zurückgerollte Transaktion im
// Speicher hinterlassen hat.
func resetAfterRollback(inv *domain.Invoice, allocated int64) {
	inv.ID = 0
	for i := range inv.Items {
		inv.Items[i].ID = 0
		inv.Items[i].InvoiceID = 0
	}
	if allocated != 0 {
		inv.InvoiceNumber = ""
	}
	inv.JournalEntryID = nil
	inv.ReceiptID = nil
	inv.Status = domain.InvoiceStatusDraft
}

// recordNumberGap vermerkt eine Nummer, die verbraucht wurde, ohne dass eine
// Rechnung sie trägt.
//
// Geprüft wird, ob sie das wirklich ist: hat der Rollback den Zähler
// zurückgesetzt, gibt es keine Lücke und nichts zu vermerken. Der Vermerk
// entsteht also nur dort, wo er stimmt — ein Lückenbericht, der Lücken
// behauptet, die keine sind, wird nicht gelesen.
func (s *InvoiceService) recordNumberGap(ctx context.Context, inv *domain.Invoice, allocated int64, cause error) {
	if allocated == 0 || s.gapRepo == nil {
		return
	}
	next, err := s.numberRepo.Peek(ctx, domain.NumberRangeInvoice, inv.FiscalYear)
	if err != nil || next <= allocated {
		return // der Rollback hat die Nummer zurückgegeben
	}
	if existing, err := s.invoiceRepo.FindByNumber(ctx, inv.InvoiceNumber); err == nil && existing != nil {
		return
	}
	_ = s.gapRepo.Record(ctx, &domain.NumberGap{
		Key:        domain.NumberRangeInvoice,
		FiscalYear: inv.FiscalYear,
		Sequence:   allocated,
		Number:     inv.InvoiceNumber,
		Reason:     domain.NumberGapAborted,
		Detail:     cause.Error(),
		RecordedAt: time.Now().Format(time.RFC3339),
	})
}

// runInTx runs fn inside a transaction where one is wired, and plainly where it
// is not.
//
// Ohne Transaktionsläufer läuft alles wie vorher — die Rechnungen entstehen,
// nur ohne die Klammer. Das ist der Preis dafür, dass der Dienst auch dort
// benutzbar bleibt, wo er ohne vollständige Verdrahtung aufgebaut wird.
func (s *InvoiceService) runInTx(ctx context.Context, fn func(context.Context) error) error {
	if s.txRunner == nil {
		return fn(ctx)
	}
	return s.txRunner.RunInTx(ctx, fn)
}

// numberFormat liest die Systematik des Nummernkreises aus den Einstellungen.
func (s *InvoiceService) numberFormat(ctx context.Context) string {
	if s.settingsRepo == nil {
		return domain.DefaultInvoiceNumberFormat
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || cfg == nil {
		return domain.DefaultInvoiceNumberFormat
	}
	if domain.ValidateInvoiceNumberFormat(cfg.InvoiceNumberFormat) != nil {
		return domain.DefaultInvoiceNumberFormat
	}
	return cfg.InvoiceNumberFormat
}

// attachDocument erzeugt das Rechnungsdokument und legt es als Ausgangsbeleg ab.
//
// The Beleg carries the PDF as the received form and the XML as the structured
// part on a hybrid document; bei einer XRechnung ist das XML das Original und
// das PDF die erzeugte Darstellung. Beide Rollen zählen: das PDF ist, was der
// Empfänger liest, das XML ist, woran sein Vorsteuerabzug hängt.
//
// GoBD Rz. 76 Abs. 2 would allow skipping the archived PDF entirely, since
// Buchfink can reproduce an identical Mehrstück from the data at any time. It is
// archived anyway: a stored document is more tangible to the user than a promise
// to re-render one.
func (s *InvoiceService) attachDocument(ctx context.Context, inv *domain.Invoice, contact *domain.Contact) error {
	if s.receiptSvc == nil || s.renderer == nil {
		// Ohne Beleg-Pipeline ist die Rechnung mit der Buchung fertig. Der Kern
		// ist die Buchung; das Dokument kommt aus einer Schicht darüber.
		inv.Status = domain.InvoiceStatusIssued
		return s.invoiceRepo.Save(ctx, inv)
	}

	seller, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return fmt.Errorf("Unternehmensdaten konnten nicht geladen werden: %w", err)
	}

	files, xml, validation, err := s.renderDocument(ctx, inv, seller, contact)
	if err != nil {
		return fmt.Errorf(
			"die Rechnung %s ist ausgestellt und gebucht, das Dokument konnte aber nicht erzeugt werden: %w. "+
				"Die Nummer bleibt vergeben; das Dokument lässt sich nachholen", inv.InvoiceNumber, err)
	}
	inv.ZUGFeRDXML = xml

	receipt, err := s.receiptSvc.File(ctx, FileReceiptRequest{
		Direction:     domain.DirectionOutgoing,
		FiscalYear:    inv.FiscalYear,
		ReceiptNumber: inv.InvoiceNumber,
		ReceivedAt:    inv.Date,
		ReceivedVia:   domain.ReceivedViaSelfIssued,
		Files:         files,
	})
	if err != nil {
		return fmt.Errorf(
			"die Rechnung %s ist ausgestellt und gebucht, ihr Beleg konnte aber nicht abgelegt werden: %w",
			inv.InvoiceNumber, err)
	}

	// Der Verweis auf den Beleg wird sofort festgehalten, vor allem, was danach
	// noch scheitern kann.
	//
	// Vorher stand er hinter Validierungsbericht und Speichern: scheiterte einer
	// von beiden, gab es den Beleg mit der Rechnungsnummer bereits, während die
	// Rechnung ihn nicht kannte — und „Dokument erneut erzeugen" legte einen
	// zweiten Beleg unter derselben Nummer an. Ein zweiter Beleg zu einem
	// Vorgang ist genau das, was die Belegablage nicht haben darf.
	inv.ReceiptID = &receipt.ID
	if err := s.invoiceRepo.Save(ctx, inv); err != nil {
		return err
	}
	return s.completeDocument(ctx, inv, receipt.ID, validation)
}

// completeDocument bringt eine Rechnung mit abgelegtem Beleg zu Ende:
// Validierungsbericht, Zustand „ausgestellt" und das Siegel auf die Buchung.
//
// Es ist ein eigener Schritt, weil er wiederholbar sein muss. Scheitert hier
// etwas, steht die Rechnung weiter auf „Dokument fehlt" — mit ihrem Beleg —,
// und „Dokument erneut erzeugen" setzt genau hier wieder auf, statt ein zweites
// Dokument zu erzeugen.
func (s *InvoiceService) completeDocument(
	ctx context.Context, inv *domain.Invoice, receiptID uint, validation domain.ReceiptValidation,
) error {
	// Der Validierungsbericht gehört an den Beleg, und zwar auch bei der eigenen
	// Rechnung. Bisher entstand er nur beim Eingang — und damit ließ sich Jahre
	// später nicht mehr belegen, gegen welches Regelwerk die eigene Rechnung
	// geprüft worden war.
	if validation.At != "" {
		if err := s.receiptSvc.SaveValidation(ctx, receiptID, validation); err != nil {
			return fmt.Errorf("der Validierungsbericht zu %s konnte nicht gespeichert werden: %w",
				inv.InvoiceNumber, err)
		}
	}

	inv.Status = domain.InvoiceStatusIssued
	if err := s.invoiceRepo.Save(ctx, inv); err != nil {
		return err
	}

	// Die Verbindung zwischen Buchung und Beleg läuft jetzt vom Beleg zur
	// Buchung und nicht mehr umgekehrt.
	//
	// Das ist kein Verzicht, sondern die Folge der Reihenfolge: der Beleg-Hash
	// steht im Buchungshash und damit in der Kette (accounting.EntryHash), und
	// ein nachträglicher Eintrag an der Buchung bräche sie. Da das Dokument
	// hinter der Transaktion entsteht, kann die Buchung ihn nicht mehr tragen.
	// Der Nachweis bleibt vollständig: der Beleg wird auf die Buchung
	// versiegelt, und der Prüflauf zählt beide Richtungen (siehe
	// entriesWithReceipt).
	if inv.JournalEntryID != nil {
		if err := s.receiptSvc.Seal(ctx, receiptID, *inv.JournalEntryID); err != nil {
			return fmt.Errorf(
				"die Rechnung %s wurde gebucht, der Beleg konnte aber nicht versiegelt werden: %w",
				inv.InvoiceNumber, err)
		}
	}
	return nil
}

// documentValidation liefert den Validierungsbericht einer Rechnung, ohne das
// Dokument noch einmal zu erzeugen.
//
// Gebraucht wird er, wenn der Beleg schon liegt und nur der Bericht daran fehlt.
// Das PDF dafür ein zweites Mal zu setzen, kostete Sekunden und ergäbe dieselbe
// Auskunft: geprüft wird der strukturierte Datensatz.
func documentValidation(
	inv *domain.Invoice, seller *domain.CompanySettings, contact *domain.Contact,
) (domain.ReceiptValidation, error) {
	profile := outputProfile(inv, contact)
	if profile == domain.EInvoiceProfilePDFOnly {
		return domain.ReceiptValidation{}, nil
	}
	_, validation, err := invoice.RenderInvoiceXML(inv, seller, contact, profile)
	return validation, err
}

// outputProfile ist das Format, in dem das Dokument tatsächlich entsteht.
//
// Zwei Regeln, und beide gehören an eine Stelle: ohne Angabe gilt ZUGFeRD, und
// die Kleinbetragsrechnung ohne Empfänger geht als reines PDF hinaus, auch wenn
// am Kontakt etwas anderes steht — ohne Erwerber gäbe es keinen normgerechten
// Datensatz (BR-07), und § 33 UStDV nimmt sie von der E-Rechnungspflicht aus.
func outputProfile(inv *domain.Invoice, contact *domain.Contact) domain.EInvoiceProfile {
	if inv.SmallAmount && contact == nil {
		return domain.EInvoiceProfilePDFOnly
	}
	if inv.EInvoiceProfile == "" {
		return domain.EInvoiceProfileZUGFeRD
	}
	return inv.EInvoiceProfile
}

// renderDocument produces the files of the outgoing Beleg in the invoice's
// format.
func (s *InvoiceService) renderDocument(
	ctx context.Context, inv *domain.Invoice, seller *domain.CompanySettings, contact *domain.Contact,
) ([]NewFile, string, domain.ReceiptValidation, error) {
	profile := outputProfile(inv, contact)
	if profile == domain.EInvoiceProfilePDFOnly {
		pdf, err := s.renderer.RenderDocumentPDF(ctx,
			invoice.GeneratePlainTypstTemplate(inv, seller, contact), inv.InvoiceNumber)
		if err != nil {
			return nil, "", domain.ReceiptValidation{}, err
		}
		return []NewFile{
			{Role: domain.ReceiptRoleOriginal, FileName: inv.InvoiceNumber + ".pdf", Content: pdf},
		}, "", domain.ReceiptValidation{}, nil
	}

	xml, validation, err := invoice.RenderInvoiceXML(inv, seller, contact, profile)
	if err != nil {
		return nil, "", validation, err
	}
	pdf, err := s.renderer.RenderInvoicePDF(ctx, inv, seller, contact, xml)
	if err != nil {
		return nil, "", validation, err
	}

	if profile == domain.EInvoiceProfileXRechnungCII {
		// Bei der XRechnung ist das XML das Original: es ist die Rechnung, und
		// das PDF ist die von Buchfink erzeugte Darstellung dazu. Dieselbe Datei
		// steht unter beiden Rollen, weil sie beides ist — die empfangene Form
		// und der strukturierte Teil.
		return []NewFile{
			{Role: domain.ReceiptRoleOriginal, FileName: inv.InvoiceNumber + ".xml", Content: []byte(xml)},
			{Role: domain.ReceiptRoleStructured, FileName: inv.InvoiceNumber + ".xml", Content: []byte(xml)},
			{Role: domain.ReceiptRoleRendering, FileName: inv.InvoiceNumber + ".pdf", Content: pdf, Derived: true},
		}, xml, validation, nil
	}

	return []NewFile{
		{Role: domain.ReceiptRoleOriginal, FileName: inv.InvoiceNumber + ".pdf", Content: pdf},
		{Role: domain.ReceiptRoleStructured, FileName: "factur-x.xml", Content: []byte(xml), Derived: true},
	}, xml, validation, nil
}

// ensureNoUnlawfulTax weist eine Rechnung zurück, deren Positionen einen
// Steuersatz tragen, obwohl der Steuerfall keine Steuer entstehen lässt.
//
// Zurückgewiesen und nicht stillschweigend berichtigt: Buchfink nahm den Satz
// bisher selbst aus der Position, und heraus kam eine Rechnung ohne Steuer,
// obwohl der Anwender 19 % erfasst hatte. Wer den Steuerfall falsch gewählt hat
// — eine steuerpflichtige Inlandsleistung als innergemeinschaftliche Lieferung
// etwa —, bekäme so eine Rechnung ohne Steuerausweis und merkte es nicht. Die
// Summenrechnung entscheidet die Frage nicht; sie ist eine Frage an den
// Anwender, welche der beiden Angaben stimmt.
//
// Die Norm dahinter ist § 14c UStG: eine ausgewiesene Steuer ohne Steuerpflicht
// wird trotzdem geschuldet, und ihre Berichtigung setzt die Zustimmung des
// Finanzamts voraus (§ 14c Abs. 2 Sätze 3 bis 5 UStG). Deshalb wird der Fehler
// beim Ausstellen verhindert und nicht danach geheilt.
func ensureNoUnlawfulTax(inv *domain.Invoice) error {
	if inv.TaxTreatment.MayShowTax() {
		return nil
	}
	for i := range inv.Items {
		item := &inv.Items[i]
		if item.TaxRate == domain.TaxRateNone {
			continue
		}
		return fmt.Errorf(
			"Position %d ist mit %s erfasst, der Steuerfall %q lässt aber keine Umsatzsteuer entstehen. "+
				"Ein ausgewiesener Steuerbetrag wird nach § 14c UStG trotzdem geschuldet – wähle entweder "+
				"den steuerpflichtigen Inlandsumsatz oder nimm den Steuersatz aus der Position",
			i+1, item.TaxRate.Label(), inv.TaxTreatment)
	}
	return nil
}

// vatIDConfirmer ist der Ausschnitt der Bestätigungsabfrage, den der
// Rechnungsweg braucht.
type vatIDConfirmer interface {
	EnsureConfirmed(ctx context.Context, contact *domain.Contact, overrideReason string) error
}

// SetVatIDConfirmer koppelt die Bestätigungsabfrage an den Rechnungsweg. Ohne
// sie stellt Buchfink wie bisher aus — mit ihr erst nach einer Bestätigung.
func (s *InvoiceService) SetVatIDConfirmer(c vatIDConfirmer) { s.vatIDs = c }

// ensureVatIDConfirmed hält eine steuerfreie Lieferung an, solange die USt-IdNr.
// des Empfängers nicht bestätigt ist.
//
// Nur für die beiden Steuerfälle, bei denen die Nummer des Empfängers
// materielle Voraussetzung ist: die innergemeinschaftliche Lieferung
// (§ 6a Abs. 1 Satz 1 Nr. 4 UStG) und die Verlagerung der Steuerschuld auf einen
// Empfänger im übrigen Gemeinschaftsgebiet. Bei einer Inlandsrechnung ist die
// USt-IdNr. des Kunden keine Voraussetzung von irgendetwas, und eine Abfrage
// dort wäre ein Netzaufruf ohne Zweck.
func (s *InvoiceService) ensureVatIDConfirmed(
	ctx context.Context, inv *domain.Invoice, contact *domain.Contact,
) error {
	if s.vatIDs == nil {
		return nil
	}
	switch inv.TaxTreatment {
	case domain.TaxTreatmentIntraCommunitySupply:
	case domain.TaxTreatmentReverseChargeSupply:
		if !contact.IsEUCounterparty() {
			return nil
		}
	default:
		return nil
	}
	return s.vatIDs.EnsureConfirmed(ctx, contact, inv.VatIDOverrideReason)
}

// validateTaxTreatment blocks the combinations that would produce a formally
// wrong invoice — the ones a supplier only finds out about during an audit.
func (s *InvoiceService) validateTaxTreatment(inv *domain.Invoice, contact *domain.Contact) error {
	switch inv.TaxTreatment {
	case domain.TaxTreatmentIntraCommunitySupply:
		if !contact.IsEUCounterparty() {
			return fmt.Errorf("eine innergemeinschaftliche Lieferung setzt einen Empfänger in einem anderen EU-Land voraus, %s ist in %q erfasst", contact.Name, contact.CountryCode)
		}
		if contact.VatID == "" {
			return fmt.Errorf("für eine innergemeinschaftliche Lieferung braucht %s eine USt-IdNr. (§ 6a Abs. 1 Nr. 4 UStG)", contact.Name)
		}
	case domain.TaxTreatmentReverseChargeSupply:
		if contact.VatID == "" {
			return fmt.Errorf("für eine Rechnung nach § 13b UStG braucht %s eine USt-IdNr.", contact.Name)
		}
	case domain.TaxTreatmentExport:
		if contact.IsEUCounterparty() || contact.CountryCode == "DE" || contact.CountryCode == "" {
			return fmt.Errorf("eine Ausfuhrlieferung setzt einen Empfänger außerhalb der EU voraus, %s ist in %q erfasst", contact.Name, contact.CountryCode)
		}
	}
	return nil
}

// Preview computes what an invoice would book, without issuing it.
//
// It applies the same defaults Issue does, so the numbers the form shows are the
// numbers that will be booked. The invoice form used to compute net, tax and
// gross itself, in a second implementation of the rounding rules — this replaces
// it, because two implementations of a tax computation are one too many.
func (s *InvoiceService) Preview(ctx context.Context, inv *domain.Invoice) (*PostingPreview, error) {
	// Ohne Empfänger gibt es eine Vorschau nur für die Kleinbetragsrechnung —
	// und für sie muss es sie geben: sonst hätte der Barverkauf als einziger
	// Rechnungsfall keine.
	var contact *domain.Contact
	if inv.ContactID != 0 {
		var err error
		contact, err = s.contactRepo.FindByID(ctx, inv.ContactID)
		if err != nil {
			return nil, fmt.Errorf("Rechnungsempfänger konnte nicht geladen werden: %w", err)
		}
		if contact.Type != domain.ContactTypeCustomer {
			return nil, fmt.Errorf("%s ist als Lieferant angelegt und kann keine Ausgangsrechnung erhalten", contact.Name)
		}
	} else if !inv.SmallAmount {
		return nil, fmt.Errorf("Rechnungsempfänger fehlt")
	}

	draft := *inv
	if draft.TaxTreatment == "" {
		draft.TaxTreatment = domain.TaxTreatmentDomestic
	}
	for i := range draft.Items {
		if draft.Items[i].Position == 0 {
			draft.Items[i].Position = i + 1
		}
	}
	if contact != nil {
		if err := s.validateTaxTreatment(&draft, contact); err != nil {
			return nil, err
		}
	}
	// Dieselbe Prüfung wie beim Ausstellen: der § 14c-Fehler soll in der Maske
	// auffallen und nicht erst am Knopf „Ausstellen".
	if err := ensureNoUnlawfulTax(&draft); err != nil {
		return nil, err
	}
	var preview *PostingPreview
	var err error
	if contact == nil {
		preview, err = s.postingSvc.PreviewCashSale(ctx, &draft, draft.PaymentAccount)
	} else {
		preview, err = s.postingSvc.PreviewOutgoingInvoice(ctx, &draft, contact)
	}
	if err != nil {
		return nil, err
	}
	// Die Grenze der Kleinbetragsrechnung reist mit der Vorschau, damit die
	// Maske die Option sperren kann, statt sie anzubieten und den Anwender bis
	// zur Fehlermeldung des Ausstellens laufen zu lassen (§ 33 UStDV). Sie ist
	// datiert und gehört deshalb hierher und nicht in eine Tabelle im Frontend.
	if params, perr := accounting.TaxParametersFor(draft.Date); perr == nil {
		preview.SmallAmountLimit = params.SmallAmountInvoiceLimit
	}
	return preview, nil
}

// Cancel reverses an issued invoice by Generalumkehr — ohne Dokument für den
// Empfänger.
//
// Der Weg des Anwenders ist CancelWithDocument, und die Bridge führt allein
// dorthin: eine stornierte Rechnung ist beim Empfänger in der Welt, und die
// Rücknahme muss ihn erreichen (§ 14 Abs. 4 Nr. 4, § 17 Abs. 1 UStG). Diese
// Fassung bleibt der schmale Weg für Prüfungen, die allein die Buchung
// betreffen; sie ist keine Alternative zum Storno mit Dokument.
func (s *InvoiceService) Cancel(ctx context.Context, invoiceID uint, reason string) error {
	inv, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return err
	}
	if inv.Status == domain.InvoiceStatusCancelled {
		return fmt.Errorf("Rechnung %s ist bereits storniert", inv.InvoiceNumber)
	}
	if inv.JournalEntryID == nil {
		return fmt.Errorf("Rechnung %s ist nicht gebucht und kann nicht storniert werden", inv.InvoiceNumber)
	}

	if _, err := s.postingSvc.journalSvc.Reverse(ctx, *inv.JournalEntryID, reason); err != nil {
		return err
	}
	return s.invoiceRepo.UpdateStatus(ctx, invoiceID, domain.InvoiceStatusCancelled)
}

// GenerateZUGFeRDAndTypst produces the structured record and the Typst source
// of an invoice.
//
// Erzeugt wird, was die Rechnung ist, und nicht immer eine ZUGFeRD-Datei: das
// Profil steht an der Rechnung (Entscheidung 4). Vorher rechnete diese Vorschau
// jede Rechnung in ZUGFeRD um — bei einer XRechnung zeigte sie damit einen
// anderen Datensatz (BT-24) als den abgelegten, und bei einer sonstigen
// Rechnung ein XML, das es zum Beleg gar nicht gibt.
func (s *InvoiceService) GenerateZUGFeRDAndTypst(ctx context.Context, invoiceID uint) (xml string, typst string, err error) {
	inv, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return "", "", fmt.Errorf("Rechnung %d wurde nicht gefunden: %w", invoiceID, err)
	}

	seller, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return "", "", fmt.Errorf("Unternehmensdaten konnten nicht geladen werden: %w", err)
	}

	// Die Kleinbetragsrechnung ohne erfassten Kunden hat keinen Empfänger, und
	// das ist kein Fehler (§ 33 UStDV). Jede andere Rechnung hat einen; ist er
	// nicht mehr auffindbar, bleibt der Name, unter dem sie ausgestellt wurde.
	var buyer *domain.Contact
	if inv.ContactID != 0 {
		buyer, err = s.contactRepo.FindByID(ctx, inv.ContactID)
		if err != nil || buyer == nil {
			buyer = &domain.Contact{Name: inv.ContactName, CountryCode: "DE"}
		}
	}

	profile := outputProfile(inv, buyer)
	if profile == domain.EInvoiceProfilePDFOnly {
		return "", invoice.GeneratePlainTypstTemplate(inv, seller, buyer), nil
	}

	xml, _, err = invoice.RenderInvoiceXML(inv, seller, buyer, profile)
	if err != nil {
		return "", "", fmt.Errorf("der Rechnungsdatensatz konnte nicht erzeugt werden: %w", err)
	}

	typst = invoice.GenerateTypstTemplate(inv, seller, buyer)
	return xml, typst, nil
}

func (s *InvoiceService) fiscalYearStartMonth(ctx context.Context) int {
	if s.settingsRepo == nil {
		return 1
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || cfg == nil || cfg.FiscalYearStartMonth <= 0 {
		return 1
	}
	return cfg.FiscalYearStartMonth
}
