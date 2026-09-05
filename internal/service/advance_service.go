package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Anzahlungen und der Rechnungsverbund.
//
// Der Fall verlässt den gewöhnlichen Rechnungsweg an zwei Stellen. Erstens
// entsteht die Steuer mit der Vereinnahmung statt mit der Leistung
// (§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG) — auch bei Sollversteuerung, und
// deshalb wird die Abschlagsrechnung beim Ausstellen nicht gebucht. Zweitens
// muss die Schlussrechnung die berechneten und vereinnahmten Teilentgelte samt
// ihrer Steuer absetzen (§ 14 Abs. 5 Satz 2 UStG); wer das vergisst, weist die
// Steuer zweimal aus und schuldet den Mehrbetrag.
//
// Daraus folgt die Konstruktion: die Schlussrechnung ist nicht erstellbar, ohne
// dass die Verrechnung geschieht. Nicht als Warnung — als Ablauf.

// AdvanceGroupRequest legt einen Rechnungsverbund an.
type AdvanceGroupRequest struct {
	ContactID uint           `json:"contactId"`
	Title     string         `json:"title"`
	TotalNet  domain.Cents   `json:"totalNet"`
	TaxRate   domain.TaxRate `json:"taxRate"`
}

// AdvanceInvoiceRequest stellt eine Abschlagsrechnung in einem Verbund aus.
type AdvanceInvoiceRequest struct {
	GroupID     uint         `json:"groupId"`
	Date        string       `json:"date"`
	Description string       `json:"description"`
	Net         domain.Cents `json:"net"`
	// PaymentReceivedAt ist der Zeitpunkt der Vereinnahmung, sofern er beim
	// Ausstellen feststeht (§ 14 Abs. 4 Nr. 6 UStG). Vor dem Geldeingang tut er
	// das nicht, und dann bleibt er leer.
	PaymentReceivedAt string `json:"paymentReceivedAt,omitempty"`
}

// SettleAdvanceRequest bucht den Zahlungseingang auf eine Abschlagsrechnung.
type SettleAdvanceRequest struct {
	AdvanceID uint `json:"advanceId"`
	// BankTxID ist der Bankumsatz, aus dem die Vereinnahmung stammt.
	//
	// Er ist der Regelfall und nicht die Ausnahme: das Geld auf eine
	// Abschlagsrechnung kommt über den Kontoauszug. Mit ihm kommen Konto und
	// Datum aus dem Auszug statt aus der Tastatur, und der Umsatz wird als
	// zugeordnet vermerkt — ohne ihn stünde er weiter als offen im Import und
	// würde ein zweites Mal gebucht.
	BankTxID       *uint  `json:"bankTxId,omitempty"`
	PaymentDate    string `json:"paymentDate"`
	PaymentAccount string `json:"paymentAccount"`
}

// FinalInvoiceRequest stellt die Schlussrechnung eines Verbunds aus.
type FinalInvoiceRequest struct {
	GroupID         uint                 `json:"groupId"`
	Date            string               `json:"date"`
	ServiceDateFrom string               `json:"serviceDateFrom"`
	ServiceDateTo   string               `json:"serviceDateTo"`
	Items           []domain.InvoiceItem `json:"items"`
	Terms           domain.PaymentTerms  `json:"terms"`
}

// GetInvoiceGroups liefert die Rechnungsverbünde eines Geschäftsjahres mit
// ihrem Fortschritt.
func (s *InvoiceService) GetInvoiceGroups(ctx context.Context, fiscalYear int) ([]domain.InvoiceGroup, error) {
	if s.groupRepo == nil {
		return []domain.InvoiceGroup{}, nil
	}
	groups, err := s.groupRepo.FindAll(ctx, fiscalYear)
	if err != nil {
		return nil, err
	}
	if groups == nil {
		groups = []domain.InvoiceGroup{}
	}
	return groups, nil
}

// CreateInvoiceGroup legt einen Rechnungsverbund an.
func (s *InvoiceService) CreateInvoiceGroup(ctx context.Context, req AdvanceGroupRequest) (*domain.InvoiceGroup, error) {
	if s.groupRepo == nil {
		return nil, fmt.Errorf("der Rechnungsverbund ist nicht eingerichtet")
	}
	if req.Title == "" {
		return nil, fmt.Errorf("der Rechnungsverbund braucht eine Bezeichnung")
	}
	if req.TotalNet <= 0 {
		return nil, fmt.Errorf("der vereinbarte Gesamtbetrag muss größer als null sein")
	}
	contact, err := s.contactRepo.FindByID(ctx, req.ContactID)
	if err != nil {
		return nil, fmt.Errorf("Auftraggeber konnte nicht geladen werden: %w", err)
	}
	if contact.Type != domain.ContactTypeCustomer {
		return nil, fmt.Errorf("%s ist als Lieferant angelegt und kann keinen Auftrag erteilen", contact.Name)
	}
	if _, err := accounting.AdvanceAccountFor(req.TaxRate); err != nil {
		return nil, err
	}

	group := &domain.InvoiceGroup{
		FiscalYear: s.postingSvc.journalSvc.FiscalYear(),
		ContactID:  req.ContactID,
		Title:      req.Title,
		TotalNet:   req.TotalNet,
		TaxRate:    req.TaxRate,
	}
	if err := s.groupRepo.Save(ctx, group); err != nil {
		return nil, err
	}
	// Der Verbund wird protokolliert wie jeder fachliche Schritt: Gesamtbetrag
	// und Steuersatz sind die Bemessungsgrundlage der Schlussrechnung und die
	// Obergrenze der Abschläge — wer sie später anders vorfindet, muss sehen
	// können, womit begonnen wurde.
	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, domain.AuditActionCreate, "INVOICE_GROUP", fmt.Sprintf("%d", group.ID),
			fmt.Sprintf("Rechnungsverbund %q für %s angelegt: Gesamtbetrag %s € netto, %s",
				group.Title, contact.Name, group.TotalNet, group.TaxRate.Label()))
	}
	return s.groupRepo.FindByID(ctx, group.ID)
}

// IssueAdvanceInvoice stellt eine Abschlagsrechnung aus.
//
// Sie bekommt Nummer und Dokument wie jede Rechnung — § 14 Abs. 5 Satz 1 UStG
// erklärt die Absätze 1 bis 4 für sinngemäß anwendbar —, aber keine Buchung:
// die Steuer entsteht erst mit der Vereinnahmung. Der offene Posten entsteht
// trotzdem, als eigene Quelle neben dem Journal.
func (s *InvoiceService) IssueAdvanceInvoice(ctx context.Context, req AdvanceInvoiceRequest) (*domain.Invoice, error) {
	if s.groupRepo == nil {
		return nil, fmt.Errorf("der Rechnungsverbund ist nicht eingerichtet")
	}
	group, err := s.groupRepo.FindByID(ctx, req.GroupID)
	if err != nil {
		return nil, fmt.Errorf("Rechnungsverbund konnte nicht geladen werden: %w", err)
	}
	if req.Net <= 0 {
		return nil, fmt.Errorf("der Abschlagsbetrag muss größer als null sein")
	}
	if err := group.EnsureAdvanceFits(req.Net); err != nil {
		return nil, err
	}

	description := req.Description
	if description == "" {
		description = "Abschlag auf " + group.Title
	}
	inv := &domain.Invoice{
		ContactID:         group.ContactID,
		Date:              req.Date,
		TaxTreatment:      domain.TaxTreatmentDomestic,
		Kind:              domain.InvoiceKindAdvance,
		GroupID:           &group.ID,
		PaymentReceivedAt: req.PaymentReceivedAt,
		Items: []domain.InvoiceItem{{
			Description:   description,
			QuantityMilli: 1000,
			Unit:          domain.UnitCodeDefault,
			UnitPrice:     req.Net,
			TaxRate:       group.TaxRate,
		}},
	}
	// Der offene Posten entsteht innerhalb der Nummernklammer und nicht hinter
	// der Dokumenterzeugung.
	//
	// Scheitert das Dokument, bleibt die Abschlagsrechnung mit Nummer im Zustand
	// „Dokument fehlt" stehen und lässt sich nachholen — der offene Posten muss
	// dann schon da sein. Stünde er dahinter, gäbe es die Rechnung ohne ihn:
	// sie fehlte in der OP-Liste, ließe sich nicht vereinnahmen und zählte
	// weder gegen den Gesamtbetrag noch in die Verrechnung der Schlussrechnung.
	advance := &domain.AdvanceItem{
		GroupID:   group.ID,
		ContactID: group.ContactID,
		TaxRate:   group.TaxRate,
	}
	err = s.issueWith(ctx, inv, func(ctx context.Context) error {
		return s.saveAdvanceItem(ctx, advance, inv)
	})
	if err != nil {
		return nil, err
	}
	return inv, nil
}

// SettleAdvance bucht den Zahlungseingang auf eine Abschlagsrechnung.
//
// Erst hier wird gebucht, und erst hier entsteht die Steuer: SOLL Bank an
// HABEN „Erhaltene, versteuerte Anzahlungen" und HABEN Umsatzsteuer. Der
// Voranmeldungszeitraum folgt dem Zahlungsdatum — siehe
// accounting.VatPeriodFor.
func (s *InvoiceService) SettleAdvance(ctx context.Context, req SettleAdvanceRequest) (*domain.AdvanceItem, error) {
	if s.groupRepo == nil {
		return nil, fmt.Errorf("der Rechnungsverbund ist nicht eingerichtet")
	}
	inv, err := s.invoiceRepo.FindByID(ctx, req.AdvanceID)
	if err != nil {
		return nil, err
	}
	advance, err := s.groupRepo.FindAdvanceByInvoice(ctx, inv.ID)
	if err != nil {
		return nil, err
	}
	if advance == nil {
		return nil, fmt.Errorf("zu Rechnung %s ist kein Abschlag erfasst", inv.InvoiceNumber)
	}
	if advance.Cancelled {
		return nil, fmt.Errorf("die Abschlagsrechnung %s ist storniert", advance.InvoiceNumber)
	}
	if advance.Settled() {
		return nil, fmt.Errorf("die Abschlagsrechnung %s ist am %s bereits vereinnahmt",
			advance.InvoiceNumber, domain.GermanDate(advance.SettledAt))
	}
	// Nach der Schlussrechnung gibt es keinen Abschlag mehr zu vereinnahmen:
	// die Schlussrechnung fordert den Gesamtbetrag abzüglich der bis dahin
	// vereinnahmten Anzahlungen. Eine danach gebuchte Anzahlung stünde auf 3272
	// und würde nie wieder aufgelöst — und ihre Steuer wäre ein zweites Mal
	// angemeldet.
	group, err := s.groupRepo.FindByID(ctx, advance.GroupID)
	if err != nil {
		return nil, fmt.Errorf("Rechnungsverbund konnte nicht geladen werden: %w", err)
	}
	if group.Closed {
		return nil, fmt.Errorf(
			"der Rechnungsverbund %q ist mit der Schlussrechnung abgeschlossen; die Zahlung gehört auf "+
				"die Schlussrechnung und nicht mehr auf den Abschlag %s", group.Title, advance.InvoiceNumber)
	}
	// Kommt die Zahlung aus dem Kontoauszug, entscheidet er über Konto, Datum
	// und Betrag: keines davon ist dann noch eine Eingabe, und keines kann sich
	// vertippen.
	bankTx, err := s.bankTxForSettlement(ctx, req.BankTxID, advance)
	if err != nil {
		return nil, err
	}
	if bankTx != nil {
		req.PaymentAccount = bankTx.LedgerAccount
		req.PaymentDate = bankTx.BookingDate
	}
	if req.PaymentDate == "" {
		return nil, fmt.Errorf("ohne Zahlungsdatum lässt sich die Anzahlung keinem Voranmeldungszeitraum zuordnen")
	}
	account := req.PaymentAccount
	if account == "" {
		account = domain.AccountBank
	}

	contact, err := s.contactRepo.FindByID(ctx, advance.ContactID)
	if err != nil {
		return nil, fmt.Errorf("Auftraggeber konnte nicht geladen werden: %w", err)
	}
	// Buchung, Vermerk und die Zuordnung des Bankumsatzes gehören in eine
	// Transaktion.
	//
	// Scheiterte der Vermerk hinter der Buchung, stünde die Vereinnahmung im
	// Journal, während der Abschlag weiter als offen gälte — und der zweite
	// Versuch buchte sie ein zweites Mal, mit der Steuer. Für den Bankumsatz
	// gilt dasselbe von der anderen Seite: bliebe er unzugeordnet, böte ihn der
	// Import weiter an.
	var entry *domain.JournalEntry
	if err := s.runInTx(ctx, func(ctx context.Context) error {
		posted, err := s.postingSvc.PostAdvanceSettlement(
			ctx, advance, contact, req.PaymentDate, account, req.BankTxID)
		if err != nil {
			return err
		}
		entry = posted
		advance.SettledAt = req.PaymentDate
		advance.SettlementEntryID = &posted.ID
		if err := s.groupRepo.SaveAdvance(ctx, advance); err != nil {
			return err
		}
		if bankTx == nil {
			return nil
		}
		if err := s.bankRepo.SetMatchStatus(ctx, bankTx.ID, domain.MatchStatusMatched); err != nil {
			return fmt.Errorf("der Bankumsatz konnte nicht als zugeordnet markiert werden: %w", err)
		}
		return nil
	}); err != nil {
		advance.SettledAt = ""
		advance.SettlementEntryID = nil
		return nil, err
	}
	// Erst jetzt bekommt der Beleg der Abschlagsrechnung seine Buchung. Bis
	// hierhin war er abgelegt und ungebucht — richtig, denn es gab nichts zu
	// buchen; ab hier gibt es sie, und ohne das Siegel meldete der Prüflauf
	// einen abgelegten, nicht gebuchten Beleg (GoBD Rz. 47) und sperrte die
	// Festschreibung.
	if err := s.sealInvoiceReceipt(ctx, inv, entry.ID); err != nil {
		return nil, err
	}
	// Der Vereinnahmungszeitpunkt wird hier nicht mehr an die Rechnung
	// geschrieben.
	//
	// Er stünde sonst am Datensatz einer Rechnung, deren Dokument längst als
	// Beleg abgelegt ist und ihn nicht trägt: ein erneutes Rendern ergäbe ein
	// anderes XML als das archivierte, und nach einer Rückzahlung bliebe er
	// stehen und erschiene auf dem Stornodokument. § 14 Abs. 4 Nr. 6 UStG
	// verlangt die Angabe ohnehin nur, wenn sie beim Ausstellen feststeht —
	// dann kommt sie aus AdvanceInvoiceRequest. Geführt wird die Vereinnahmung
	// am Abschlag (AdvanceItem.SettledAt), und von dort zeigt sie die
	// Oberfläche an.
	s.log(ctx, domain.AuditActionCreate, inv, fmt.Sprintf(
		"Anzahlung zu %s über %s € am %s vereinnahmt und als %s gebucht",
		advance.InvoiceNumber, advance.GrossAmount, domain.GermanDate(req.PaymentDate), entry.EntryNumber))
	return advance, nil
}

// bankTxForSettlement holt den Bankumsatz, aus dem eine Anzahlung vereinnahmt
// wird, und prüft ihn gegen den Abschlag.
//
// Drei Prüfungen, drei Fehler, die sie verhindern. Ein bereits zugeordneter
// Umsatz wäre die zweite Buchung desselben Geldes — derselbe Schutz, den
// PaymentService.Settle über MatchStatusMatched hat. Eine Auszahlung ist keine
// Vereinnahmung, und ohne die Prüfung buchte sie eine Anzahlung samt
// Umsatzsteuer aus einer Überweisung, die hinausgegangen ist. Und der Betrag
// muss stimmen: der Abschlag kennt nur ganz oder gar nicht vereinnahmt
// (AdvanceItem.SettledAt), eine Teilzahlung darf deshalb nicht als volle
// Vereinnahmung durchgehen — sie brächte die Steuer des ganzen Abschlags in
// einen Zeitraum, in dem nur ein Teil zugeflossen ist.
func (s *InvoiceService) bankTxForSettlement(
	ctx context.Context, bankTxID *uint, advance *domain.AdvanceItem,
) (*domain.BankTransaction, error) {
	if bankTxID == nil {
		return nil, nil
	}
	if s.bankRepo == nil {
		return nil, fmt.Errorf("der Bankimport ist nicht eingerichtet")
	}
	tx, err := s.bankRepo.FindByID(ctx, *bankTxID)
	if err != nil {
		return nil, fmt.Errorf("Bankumsatz %d wurde nicht gefunden: %w", *bankTxID, err)
	}
	if tx.MatchStatus == domain.MatchStatusMatched {
		return nil, fmt.Errorf("Bankumsatz %d ist bereits zugeordnet", tx.ID)
	}
	if tx.Amount <= 0 {
		return nil, fmt.Errorf(
			"der Bankumsatz %d über %s € ist eine Auszahlung; eine Anzahlung wird mit dem Zahlungseingang "+
				"vereinnahmt", tx.ID, tx.Amount)
	}
	if tx.Amount != advance.GrossAmount {
		return nil, fmt.Errorf(
			"der Bankumsatz lautet über %s €, die Abschlagsrechnung %s über %s €. Eine Anzahlung gilt erst "+
				"mit dem vollen Betrag als vereinnahmt",
			tx.Amount, advance.InvoiceNumber, advance.GrossAmount)
	}
	return tx, nil
}

// sealInvoiceReceipt versiegelt den Beleg einer Rechnung auf ihre Buchung.
//
// Der Weg ist derselbe wie in attachDocument, nur nachgelagert: bei einer
// Abschlagsrechnung entsteht die Buchung erst mit der Vereinnahmung, und bis
// dahin gibt es nichts, worauf zu versiegeln wäre.
func (s *InvoiceService) sealInvoiceReceipt(ctx context.Context, inv *domain.Invoice, entryID uint) error {
	if s.receiptSvc == nil || inv.ReceiptID == nil {
		return nil
	}
	if err := s.receiptSvc.Seal(ctx, *inv.ReceiptID, entryID); err != nil {
		return fmt.Errorf(
			"die Buchung zu %s steht, der Beleg konnte aber nicht auf sie versiegelt werden: %w",
			inv.InvoiceNumber, err)
	}
	return nil
}

// RefundAdvanceRequest zahlt eine vereinnahmte Anzahlung zurück.
type RefundAdvanceRequest struct {
	AdvanceID      uint   `json:"advanceId"`
	RefundDate     string `json:"refundDate"`
	PaymentAccount string `json:"paymentAccount"`
	Reason         string `json:"reason"`
}

// RefundAdvance bucht die Rückzahlung einer vereinnahmten Anzahlung.
//
// Sie ist der Weg, der vor dem Storno einer bezahlten Abschlagsrechnung steht.
// Die Steuer einer Anzahlung entsteht mit der Vereinnahmung; sie zu berichtigen
// setzt nach § 17 Abs. 2 Nr. 2 i. V. m. Abs. 1 UStG voraus, dass das Entgelt
// zurückgezahlt worden ist (Abschn. 17.1 Abs. 7 UStAE). Ohne die Rückzahlung
// bleibt die Steuer geschuldet, und ein Storno, das sie stillschweigend
// auflöste, wäre eine unberechtigte Minderung.
//
// Gebucht wird das Gegenteil der Vereinnahmung: SOLL Anzahlungskonto und SOLL
// Umsatzsteuer an HABEN Zahlungsmittel — im Zeitraum der Rückzahlung
// (§ 17 Abs. 1 Satz 8 UStG).
func (s *InvoiceService) RefundAdvance(ctx context.Context, req RefundAdvanceRequest) (*domain.AdvanceItem, error) {
	if s.groupRepo == nil {
		return nil, fmt.Errorf("der Rechnungsverbund ist nicht eingerichtet")
	}
	if req.Reason == "" {
		return nil, fmt.Errorf("zu jeder Rückzahlung gehört eine Begründung")
	}
	inv, err := s.invoiceRepo.FindByID(ctx, req.AdvanceID)
	if err != nil {
		return nil, err
	}
	advance, err := s.groupRepo.FindAdvanceByInvoice(ctx, inv.ID)
	if err != nil {
		return nil, err
	}
	if advance == nil {
		return nil, fmt.Errorf("zu Rechnung %s ist kein Abschlag erfasst", inv.InvoiceNumber)
	}
	if !advance.Settled() {
		return nil, fmt.Errorf(
			"die Abschlagsrechnung %s ist nicht vereinnahmt; es gibt nichts zurückzuzahlen",
			advance.InvoiceNumber)
	}
	if advance.SettledInFinal {
		return nil, fmt.Errorf(
			"die Anzahlung zu %s ist in der Schlussrechnung verrechnet und kann nicht mehr zurückgezahlt werden",
			advance.InvoiceNumber)
	}
	if req.RefundDate == "" {
		return nil, fmt.Errorf("ohne Datum lässt sich die Rückzahlung keinem Voranmeldungszeitraum zuordnen")
	}
	account := req.PaymentAccount
	if account == "" {
		account = domain.AccountBank
	}

	contact, err := s.contactRepo.FindByID(ctx, advance.ContactID)
	if err != nil {
		return nil, fmt.Errorf("Auftraggeber konnte nicht geladen werden: %w", err)
	}
	// Buchung und Vermerk gehören auch hier in eine Transaktion: bliebe der
	// Vermerk aus, gälte die Anzahlung weiter als vereinnahmt, und der zweite
	// Versuch buchte die Rückzahlung samt Steuerkorrektur ein zweites Mal.
	settledAt, entryID := advance.SettledAt, advance.SettlementEntryID
	if err := s.runInTx(ctx, func(ctx context.Context) error {
		if _, err := s.postingSvc.PostAdvanceRefund(
			ctx, advance, contact, req.RefundDate, account, req.Reason); err != nil {
			return err
		}
		// Die Anzahlung ist damit nicht mehr vereinnahmt: sie fällt aus der
		// Verrechnung der Schlussrechnung heraus und steht wieder als offener
		// Posten, bis die Abschlagsrechnung storniert ist.
		advance.SettledAt = ""
		advance.SettlementEntryID = nil
		return s.groupRepo.SaveAdvance(ctx, advance)
	}); err != nil {
		advance.SettledAt, advance.SettlementEntryID = settledAt, entryID
		return nil, err
	}
	s.log(ctx, domain.AuditActionUpdate, inv, fmt.Sprintf(
		"Anzahlung zu %s über %s € am %s zurückgezahlt: %s",
		advance.InvoiceNumber, advance.GrossAmount, domain.GermanDate(req.RefundDate), req.Reason))
	return advance, nil
}

// IssueFinalInvoice stellt die Schlussrechnung eines Verbunds aus.
//
// Sie rechnet die Gesamtleistung ab und setzt die berechneten *und*
// vereinnahmten Anzahlungen ab. Nicht die vom Anwender ausgewählten: der Verbund
// entscheidet, welche abzusetzen sind, und § 14 Abs. 5 Satz 2 UStG lässt dabei
// keine Wahl.
func (s *InvoiceService) IssueFinalInvoice(ctx context.Context, req FinalInvoiceRequest) (*domain.Invoice, error) {
	if s.groupRepo == nil {
		return nil, fmt.Errorf("der Rechnungsverbund ist nicht eingerichtet")
	}
	group, err := s.groupRepo.FindByID(ctx, req.GroupID)
	if err != nil {
		return nil, fmt.Errorf("Rechnungsverbund konnte nicht geladen werden: %w", err)
	}
	if group.Closed {
		return nil, fmt.Errorf("der Rechnungsverbund %q hat bereits eine Schlussrechnung", group.Title)
	}
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("die Schlussrechnung hat keine Positionen")
	}
	// Ein offener Abschlag darf die Schlussrechnung nicht überleben. Er wird
	// nicht abgesetzt — vereinnahmt ist nichts —, bliebe aber als offener Posten
	// neben der Schlussrechnung stehen: derselbe Betrag würde zweimal
	// gefordert. Die Entscheidung, ob er noch bezahlt oder storniert wird,
	// gehört vor die Schlussrechnung und nicht hinter sie.
	if open := openAdvancesOf(group); len(open) > 0 {
		return nil, fmt.Errorf(
			"der Verbund %q hat noch offene Abschlagsrechnungen: %s. "+
				"Vereinnahme sie oder storniere sie, bevor die Schlussrechnung entsteht — sonst stünden "+
				"sie als offene Posten neben ihr", group.Title, strings.Join(open, ", "))
	}

	advances := group.DeductibleAdvances()
	var prepaid domain.Cents
	refs := make([]domain.InvoiceReference, 0, len(advances))
	for i := range advances {
		prepaid += advances[i].GrossAmount
		refs = append(refs, domain.InvoiceReference{
			Number: advances[i].InvoiceNumber,
			Date:   advances[i].InvoiceDate,
		})
	}

	inv := &domain.Invoice{
		ContactID:       group.ContactID,
		Date:            req.Date,
		ServiceDateFrom: req.ServiceDateFrom,
		ServiceDateTo:   req.ServiceDateTo,
		TaxTreatment:    domain.TaxTreatmentDomestic,
		Kind:            domain.InvoiceKindFinal,
		GroupID:         &group.ID,
		Terms:           req.Terms,
		Items:           req.Items,
		PrecedingRefs:   refs,
		PrepaidAmount:   prepaid,
	}
	// Der Abschluss des Verbunds gehört in dieselbe Transaktion wie Nummer und
	// Buchung der Schlussrechnung.
	//
	// Stünde er dahinter, ließe ein gescheitertes Dokument den Verbund offen,
	// obwohl Gesamterlös und Auflösung der Anzahlungen gebucht sind: der
	// nächste Versuch „Schlussrechnung ausstellen" käme durch und buchte
	// Gesamterlös und volle Umsatzsteuer ein zweites Mal (§ 14c Abs. 1 UStG),
	// während 3272 ins Soll rutschte. Genau die Invariante „die Schlussrechnung
	// setzt alle Anzahlungen ab; danach keine weitere" wäre über den normalen
	// Wiederholungsweg verletzt.
	if err := s.issueFinal(ctx, inv, advances, func(ctx context.Context) error {
		for i := range advances {
			advances[i].SettledInFinal = true
			if err := s.groupRepo.SaveAdvance(ctx, &advances[i]); err != nil {
				return err
			}
		}
		group.Closed = true
		group.FinalInvoiceID = &inv.ID
		return s.groupRepo.Save(ctx, group)
	}); err != nil {
		return nil, err
	}
	return inv, nil
}

// issueFinal ist Issue mit der Buchung der Schlussrechnung: dieselbe Vorbereitung
// und dieselbe Nummernklammer, nur eine andere Buchung.
func (s *InvoiceService) issueFinal(
	ctx context.Context, inv *domain.Invoice, advances []domain.AdvanceItem, within func(context.Context) error,
) error {
	contact, err := s.prepareForIssue(ctx, inv)
	if err != nil {
		return err
	}
	if inv.PrepaidAmount > inv.GrossAmount {
		return fmt.Errorf(
			"die verrechneten Anzahlungen von %s € übersteigen den Rechnungsbetrag von %s €",
			inv.PrepaidAmount, inv.GrossAmount)
	}

	format := s.numberFormat(ctx)
	var allocated int64
	err = s.runInTx(ctx, func(ctx context.Context) error {
		seq, err := s.numberRepo.Allocate(ctx, domain.NumberRangeInvoice, inv.FiscalYear)
		if err != nil {
			return fmt.Errorf("Rechnungsnummer konnte nicht vergeben werden: %w", err)
		}
		allocated = seq
		inv.InvoiceNumber = domain.FormatInvoiceNumberWith(format, inv.FiscalYear, seq)
		inv.Status = domain.InvoiceStatusPendingDocument
		if err := s.invoiceRepo.Save(ctx, inv); err != nil {
			return err
		}
		entry, err := s.postingSvc.PostFinalInvoice(ctx, inv, contact, advances)
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
		resetAfterRollback(inv, allocated)
		return err
	}
	if err := s.attachDocument(ctx, inv, contact); err != nil {
		return err
	}
	s.logIssued(ctx, inv)
	return nil
}

// OpenAdvanceItems liefert die ausgestellten, noch nicht vereinnahmten
// Abschläge als offene Posten.
//
// Sie sind die zweite Quelle der OP-Liste. Die erste leitet den offenen Posten
// aus der Buchung ab — und eine Abschlagsrechnung hat vor der Zahlung keine.
// Ohne diese Quelle stünde eine gestellte Abschlagsrechnung nirgends, und
// niemand mahnte sie an.
func (s *InvoiceService) OpenAdvanceItems(ctx context.Context) ([]domain.OpenItem, error) {
	out := make([]domain.OpenItem, 0)
	if s.groupRepo == nil {
		return out, nil
	}
	advances, err := s.groupRepo.FindOpenAdvances(ctx)
	if err != nil {
		return nil, err
	}
	for i := range advances {
		a := &advances[i]
		contact, err := s.contactRepo.FindByID(ctx, a.ContactID)
		if err != nil {
			continue
		}
		out = append(out, domain.OpenItem{
			// Die Herkunft steht am Posten und nicht in einer zweiten Liste:
			// der Abschlag hat keine Buchung und damit keine EntryID, und der
			// Zahlungsausgleich muss ihn über SettleAdvance führen statt über
			// Settle, das ohne offenen Posten im Journal nichts fände.
			Source:           domain.OpenItemSourceAdvance,
			AdvanceInvoiceID: a.InvoiceID,
			ContactID:        a.ContactID,
			ContactName:      contact.Name,
			ContactType:      contact.Type,
			LedgerAccount:    contact.LedgerAccount,
			DocumentNumber:   a.InvoiceNumber,
			DocumentDate:     a.InvoiceDate,
			GrossAmount:      a.GrossAmount,
			OpenAmount:       a.GrossAmount,
			TaxRate:          a.TaxRate,
			TaxTreatment:     domain.TaxTreatmentDomestic,
		})
	}
	return out, nil
}

// openAdvancesOf nennt die Abschlagsrechnungen eines Verbunds, die weder
// vereinnahmt noch storniert sind.
func openAdvancesOf(group *domain.InvoiceGroup) []string {
	out := make([]string, 0)
	for i := range group.Advances {
		a := &group.Advances[i]
		if !a.Cancelled && !a.Settled() {
			out = append(out, a.InvoiceNumber)
		}
	}
	return out
}

// ensureFinalCancellable weist das Storno einer Schlussrechnung zurück, deren
// Verbund sich nicht wieder öffnen lässt.
//
// Ohne den Verbund bliebe die Generalumkehr auf halbem Weg stehen: sie nähme die
// Auflösung der Anzahlungen zurück (3272 und 3806 stehen wieder im Haben),
// während die Abschläge weiter als „in der Schlussrechnung verrechnet" gälten.
// Sie wären damit nie wieder absetzbar, und eine neue Schlussrechnung
// scheiterte am geschlossenen Verbund.
func (s *InvoiceService) ensureFinalCancellable(ctx context.Context, inv *domain.Invoice) error {
	if inv.ResolvedKind() != domain.InvoiceKindFinal {
		return nil
	}
	if s.groupRepo == nil || inv.GroupID == nil {
		return fmt.Errorf(
			"die Schlussrechnung %s gehört zu keinem erreichbaren Rechnungsverbund und kann nicht storniert "+
				"werden; ohne ihn ließen sich die verrechneten Anzahlungen nicht wieder absetzen",
			inv.InvoiceNumber)
	}
	if _, err := s.groupRepo.FindByID(ctx, *inv.GroupID); err != nil {
		return fmt.Errorf("der Rechnungsverbund zu %s konnte nicht geladen werden: %w", inv.InvoiceNumber, err)
	}
	return nil
}

// reopenFinalGroup öffnet den Verbund, dessen Schlussrechnung storniert wird.
//
// Die Anzahlungen werden wieder als unverrechnet geführt, und der Verbund
// nimmt wieder eine Schlussrechnung an. Beides gehört zusammen und in dieselbe
// Transaktion wie die Generalumkehr: sie ist es, die die Auflösung von
// 3272/3806 zurücknimmt.
func (s *InvoiceService) reopenFinalGroup(ctx context.Context, inv *domain.Invoice) error {
	if inv.ResolvedKind() != domain.InvoiceKindFinal || s.groupRepo == nil || inv.GroupID == nil {
		return nil
	}
	group, err := s.groupRepo.FindByID(ctx, *inv.GroupID)
	if err != nil {
		return err
	}
	// Nur die Schlussrechnung dieses Verbunds öffnet ihn wieder. Ein
	// Korrekturdokument, das ihn nie geschlossen hat, lässt ihn, wie er ist.
	if group.FinalInvoiceID == nil || *group.FinalInvoiceID != inv.ID {
		return nil
	}
	for i := range group.Advances {
		a := &group.Advances[i]
		if !a.SettledInFinal {
			continue
		}
		a.SettledInFinal = false
		if err := s.groupRepo.SaveAdvance(ctx, a); err != nil {
			return err
		}
	}
	group.Closed = false
	group.FinalInvoiceID = nil
	return s.groupRepo.Save(ctx, group)
}

// issueFinalReplacement stellt die berichtigte Schlussrechnung eines wieder
// geöffneten Verbunds aus.
//
// Sie geht denselben Weg wie die erste (issueFinal) und setzt dieselben
// Anzahlungen ab; sie trägt nur zusätzlich den Bezug auf die berichtigte
// Rechnung (BG-3) und den Typcode 384.
func (s *InvoiceService) issueFinalReplacement(
	ctx context.Context, replacement *domain.Invoice, original *domain.Invoice,
) error {
	if s.groupRepo == nil || original.GroupID == nil {
		return fmt.Errorf("der Rechnungsverbund zu %s ist nicht erreichbar", original.InvoiceNumber)
	}
	group, err := s.groupRepo.FindByID(ctx, *original.GroupID)
	if err != nil {
		return fmt.Errorf("Rechnungsverbund konnte nicht geladen werden: %w", err)
	}
	if group.Closed {
		return fmt.Errorf("der Rechnungsverbund %q hat bereits eine Schlussrechnung", group.Title)
	}

	advances := group.DeductibleAdvances()
	var prepaid domain.Cents
	refs := make([]domain.InvoiceReference, 0, len(advances)+1)
	for i := range advances {
		prepaid += advances[i].GrossAmount
		refs = append(refs, domain.InvoiceReference{
			Number: advances[i].InvoiceNumber,
			Date:   advances[i].InvoiceDate,
		})
	}
	replacement.GroupID = &group.ID
	replacement.PrecedingRefs = refs
	replacement.PrepaidAmount = prepaid

	return s.issueFinal(ctx, replacement, advances, func(ctx context.Context) error {
		for i := range advances {
			advances[i].SettledInFinal = true
			if err := s.groupRepo.SaveAdvance(ctx, &advances[i]); err != nil {
				return err
			}
		}
		group.Closed = true
		group.FinalInvoiceID = &replacement.ID
		return s.groupRepo.Save(ctx, group)
	})
}

// replacementGroupOf liefert den Verbund, in dem die berichtigte
// Abschlagsrechnung entstehen soll.
//
// Sie steht getrennt vom Ausstellen, weil CorrectInvoice sie vor dem Storno
// braucht: was hier scheitert, soll die Ursprungsrechnung nicht schon
// storniert vorfinden.
func (s *InvoiceService) replacementGroupOf(
	ctx context.Context, original *domain.Invoice,
) (*domain.InvoiceGroup, error) {
	if s.groupRepo == nil || original.GroupID == nil {
		return nil, fmt.Errorf(
			"die Abschlagsrechnung %s gehört zu keinem erreichbaren Rechnungsverbund; ohne ihn ließe sich "+
				"die berichtigte Abschlagsrechnung nicht in denselben Vorgang stellen",
			original.InvoiceNumber)
	}
	group, err := s.groupRepo.FindByID(ctx, *original.GroupID)
	if err != nil {
		return nil, fmt.Errorf("Rechnungsverbund konnte nicht geladen werden: %w", err)
	}
	return group, nil
}

// issueAdvanceReplacement stellt die berichtigte Abschlagsrechnung im selben
// Verbund aus.
//
// Sie geht denselben Weg wie die erste (IssueAdvanceInvoice): keine Buchung —
// die Steuer entsteht mit der Vereinnahmung —, dafür der offene Posten der
// Quelle „Abschlag" innerhalb der Nummernklammer. Sie trägt zusätzlich den
// Bezug auf die stornierte Rechnung (BG-3), den CorrectInvoice gesetzt hat.
//
// Der Steuersatz kommt aus dem Verbund und muss zu den Positionen passen. Er
// entscheidet über das Anzahlungskonto (3272 oder 3260) und über die Steuer der
// Vereinnahmung; ein abweichender Satz auf dem Dokument ergäbe eine Buchung,
// die etwas anderes sagt als die Rechnung.
func (s *InvoiceService) issueAdvanceReplacement(
	ctx context.Context, replacement *domain.Invoice, original *domain.Invoice,
) error {
	group, err := s.replacementGroupOf(ctx, original)
	if err != nil {
		return err
	}
	replacement.Kind = domain.InvoiceKindAdvance
	replacement.GroupID = &group.ID
	replacement.TaxTreatment = domain.TaxTreatmentDomestic
	for i := range replacement.Items {
		if replacement.Items[i].TaxRate != group.TaxRate {
			return fmt.Errorf(
				"der Rechnungsverbund %q ist mit %s vereinbart; die berichtigte Abschlagsrechnung weist in "+
					"Position %d %s aus", group.Title, group.TaxRate.Label(), i+1,
				replacement.Items[i].TaxRate.Label())
		}
	}
	replacement.Recalculate()
	if replacement.NetAmount <= 0 {
		return fmt.Errorf("der Abschlagsbetrag muss größer als null sein")
	}
	// Die stornierte Abschlagsrechnung ist zu diesem Zeitpunkt aus der
	// Verrechnung genommen (AdvanceItem.Cancelled) und zählt nicht mehr gegen
	// den Gesamtbetrag: geprüft wird die berichtigte gegen die verbleibenden.
	if err := group.EnsureAdvanceFits(replacement.NetAmount); err != nil {
		return err
	}

	advance := &domain.AdvanceItem{
		GroupID:   group.ID,
		ContactID: group.ContactID,
		TaxRate:   group.TaxRate,
	}
	return s.issueWith(ctx, replacement, func(ctx context.Context) error {
		return s.saveAdvanceItem(ctx, advance, replacement)
	})
}

// saveAdvanceItem schreibt den offenen Posten einer Abschlagsrechnung fort.
//
// Er entsteht innerhalb der Nummernklammer und nicht hinter der
// Dokumenterzeugung — die Begründung steht an IssueAdvanceInvoice.
func (s *InvoiceService) saveAdvanceItem(
	ctx context.Context, advance *domain.AdvanceItem, inv *domain.Invoice,
) error {
	advance.InvoiceID = inv.ID
	advance.InvoiceNumber = inv.InvoiceNumber
	advance.InvoiceDate = inv.Date
	advance.NetAmount = inv.NetAmount
	advance.TaxAmount = inv.TaxAmount
	advance.GrossAmount = inv.GrossAmount
	if err := s.groupRepo.SaveAdvance(ctx, advance); err != nil {
		return fmt.Errorf("die Abschlagsrechnung ließ sich dem Verbund nicht zuordnen: %w", err)
	}
	return nil
}

// markAdvanceCancelled nimmt einen stornierten Abschlag aus der Verrechnung.
func (s *InvoiceService) markAdvanceCancelled(ctx context.Context, invoiceID uint) error {
	if s.groupRepo == nil {
		return nil
	}
	advance, err := s.groupRepo.FindAdvanceByInvoice(ctx, invoiceID)
	if err != nil || advance == nil {
		return err
	}
	if advance.Cancelled {
		return nil
	}
	advance.Cancelled = true
	return s.groupRepo.SaveAdvance(ctx, advance)
}
