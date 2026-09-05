package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// SettlementKind separates *when* something is paid from *how* it is paid.
//
// The concept originally called an immediately settled receipt "bar", which
// silently implied the bank account. Cash is Kasse (1600) and has its own rules;
// a card payment is a different account again. Payment timing and payment means
// are therefore two independent inputs.
type SettlementKind string

const (
	// SettlementOpen books against the partner's Personenkonto and leaves an
	// open item.
	SettlementOpen SettlementKind = "open"
	// SettlementPaid books directly against a liquid account.
	SettlementPaid SettlementKind = "paid"
)

// ReceiptPosition is one net amount of an incoming document, classified by a
// fachliche Gruppe (or a directly chosen account) and a VAT rate.
type ReceiptPosition struct {
	PostingGroup string         `json:"postingGroup"`
	Account      string         `json:"account,omitempty"` // direkte Kontowahl statt Gruppe
	Net          domain.Cents   `json:"net"`
	TaxRate      domain.TaxRate `json:"taxRate"`
	Text         string         `json:"text,omitempty"`

	// InputTaxShare ist der abziehbare Anteil der Vorsteuer in Promille — der
	// Vorsteuerschlüssel der gemischten Nutzung.
	//
	// Null heißt „voll abziehbar": der Regelfall braucht keine Eingabe. Wo der
	// Anteil kleiner ist, wird nur er in der Voranmeldung geltend gemacht
	// (§ 15 Abs. 4 UStG), und die nicht abziehbare Vorsteuer gehört zum Aufwand
	// (§ 9b Abs. 1 EStG) — sie verschwindet nicht, sie wechselt die Zeile.
	InputTaxShare int `json:"inputTaxShare,omitempty"`
	// InputTaxShareReason ist bei einem Anteil unter 1000 Pflicht. Die Aufteilung
	// ist eine sachgerechte Schätzung des Unternehmers (§ 15 Abs. 4 Satz 2 UStG);
	// eine Schätzung ohne Maßstab ist keine.
	InputTaxShareReason string `json:"inputTaxShareReason,omitempty"`

	// Gift ist die Aufzeichnung nach § 4 Abs. 7 EStG, wo die Gruppe sie verlangt.
	Gift *GiftInput `json:"gift,omitempty"`
}

// GiftInput ist der Empfänger eines Geschenks, wie ihn die Maske übergibt.
type GiftInput struct {
	// ContactID benennt den Empfänger, wo er ein erfasster Geschäftspartner ist.
	ContactID uint `json:"contactId,omitempty"`
	// Name ist der Empfänger als Freitext — für den, der nicht in der
	// Kontaktliste steht.
	Name     string `json:"name,omitempty"`
	Occasion string `json:"occasion,omitempty"`
}

// ReceiptRequest is the complete input for booking an incoming document.
//
// These are the eight facts a booking cannot be derived without: direction (implied
// by the type), partner, Belegdatum, Leistungszeitraum, net amounts per VAT
// rate, Steuerfall, fachliche Gruppe and how it is settled.
type ReceiptRequest struct {
	ContactID       uint   `json:"contactId"`
	BookingDate     string `json:"bookingDate"`
	DocumentDate    string `json:"documentDate"`
	ServiceDateFrom string `json:"serviceDateFrom"`
	ServiceDateTo   string `json:"serviceDateTo"`

	// ReceiptID names the Beleg this booking belongs to. It replaces the former
	// pair of document hash and path: a Beleg is several files, and the
	// Belegnummer comes from the Beleg rather than being typed in freehand.
	ReceiptID uint `json:"receiptId"`

	Description    string              `json:"description"`
	TaxTreatment   domain.TaxTreatment `json:"taxTreatment"`
	Positions      []ReceiptPosition   `json:"positions"`
	Settlement     SettlementKind      `json:"settlement"`
	PaymentAccount string              `json:"paymentAccount,omitempty"`

	// Currency ist die Währung des Belegs. Leer oder „EUR" heißt: es wird in der
	// Buchwährung abgerechnet und nichts umgerechnet.
	//
	// Steht hier eine andere Währung, sind die Nettobeträge der Positionen die
	// der Fremdwährung — der Anwender tippt ab, was auf der Rechnung steht.
	// Buchfink holt den EZB-Referenzkurs des Belegdatums, rechnet um und hält
	// Kurs, Quelle und Kurstag an der Buchung fest. Ohne Kurs wird nicht
	// gebucht: ein geratener Kurs von 1,0 bucht einen Dollarbetrag als
	// Eurobetrag und fällt niemandem auf.
	Currency string `json:"currency,omitempty"`

	// ForeignAmount ist die Endsumme des Belegs in der Fremdwährung — die
	// Kontrollsumme zu den Positionen.
	//
	// Sie ist freiwillig und wird, wo sie steht, gegen die Summe der Positionen
	// gehalten. Ein Zahlendreher in einer Position ergibt sonst eine Buchung,
	// die in sich aufgeht und mit der Rechnung daneben nicht übereinstimmt.
	ForeignAmount domain.Cents `json:"foreignAmount,omitempty"`

	// Entertainment carries the record § 4 Abs. 5 Satz 1 Nr. 2 EStG requires
	// whenever entertainment expenses are booked. Without it the deduction is
	// lost even for the deductible 70 %, so the booking is refused rather than
	// written incomplete.
	Entertainment *domain.EntertainmentDetail `json:"entertainment,omitempty"`

	// ProvisionID ordnet den Beleg einer Rückstellung zu.
	//
	// Dann bucht Buchfink nicht gegen den Aufwand, sondern gegen das
	// Rückstellungskonto: die Verpflichtung, für die zurückgestellt wurde, wird
	// erfüllt, und der Aufwand ist im Vorjahr schon entstanden. Ein zweites Mal
	// Aufwand zu buchen wäre die häufigste Art, eine Rückstellung falsch
	// abzuwickeln. Was die Rückstellung nicht deckt, bleibt Aufwand.
	ProvisionID uint `json:"provisionId,omitempty"`

	// AdvanceTarget kennzeichnet den Beleg als geleistete Anzahlung und sagt,
	// wofür angezahlt wurde.
	//
	// Der Beleg wird dann nicht als Aufwand gebucht, sondern auf das Konto der
	// geleisteten Anzahlungen: bezahlt ist etwas, geliefert nichts. Und er wird
	// erst mit der Zahlung gebucht — der Vorsteuerabzug aus einer
	// Anzahlungsrechnung setzt nach § 15 Abs. 1 Satz 1 Nr. 1 Satz 3 UStG neben
	// der Rechnung die Entrichtung des Entgelts voraus.
	AdvanceTarget accounting.AdvanceTarget `json:"advanceTarget,omitempty"`

	// OverrideReason übersteuert einen blockierenden Befund der
	// Rechnungsprüfung — mit Grund und nur mit Grund.
	//
	// Die Pflichtangaben der §§ 14, 14a UStG sind Voraussetzung des
	// Vorsteuerabzugs. Fehlt eine, weist Buchfink die Buchung zurück; der Weg
	// daran vorbei bleibt trotzdem offen, weil eine Rechnung nach der
	// Rechtsprechung rückwirkend berichtigt werden kann und weil ein Programm,
	// das die eigene Einschätzung des Anwenders gar nicht zulässt, umgangen
	// statt befolgt wird. Der Grund steht danach am Beleg und im Protokoll.
	OverrideReason string `json:"overrideReason,omitempty"`

	// SettledAdvanceIDs sind die geleisteten Anzahlungen, die dieser Beleg
	// absetzt — die Schlussrechnung des Lieferanten.
	//
	// Ohne sie stünde die Anzahlung weiter im Vermögen und die Vorsteuer würde
	// ein zweites Mal gezogen: die Schlussrechnung weist den Gesamtbetrag aus,
	// die Steuer auf den angezahlten Teil ist aber schon abgezogen.
	SettledAdvanceIDs []uint `json:"settledAdvanceIds,omitempty"`
}

// PostingPreview is what a booking would look like, computed without writing it.
//
// It exists so the user interface never re-implements the tax rules. A second
// computation in the frontend is a second truth that drifts the moment a
// Steuerfall is added — and the one in the frontend is the one nobody tests.
type PostingPreview struct {
	Lines []domain.JournalLine `json:"lines"`
	// Net is the sum of the expense or revenue lines, Gross what is actually
	// paid or received, and Tax the difference between them. Under Reverse
	// Charge the two coincide and Tax is zero, which is exactly right: the
	// supplier is paid the net amount. The tax that arises there is visible in
	// Lines, where the two legs cancel out.
	Net      domain.Cents `json:"net"`
	Tax      domain.Cents `json:"tax"`
	Gross    domain.Cents `json:"gross"`
	Balanced bool         `json:"balanced"`

	// Warnings are notes the user should see before booking. They never block.
	// They are computed on demand rather than stored: a conserved legal
	// assessment goes stale, and every input it depends on is still there.
	Warnings []PostingWarning `json:"warnings,omitempty"`

	// SmallAmountLimit ist die Bruttogrenze der Kleinbetragsrechnung am
	// Rechnungsdatum (§ 33 UStDV), null außerhalb der Ausgangsrechnung.
	//
	// Sie steht in der Vorschau, weil der Rechnungsdialog die Option sonst
	// anbietet und erst das Ausstellen sie zurückweist. Die Grenze ist datiert
	// (150 Euro bis 2016, seither 250) und darf im Frontend nicht als zweite,
	// undatierte Zahl liegen: sie kommt mit demselben Aufruf, der auch den
	// Bruttobetrag liefert, gegen den sie zu vergleichen ist.
	SmallAmountLimit domain.Cents `json:"smallAmountLimit,omitempty"`

	// InputTaxFindings sind die blockierenden Befunde der Rechnungsprüfung. Sie
	// stehen neben den Warnungen und nicht in ihnen: eine Warnung zeigt, ein
	// Befund hält an. Leer statt nil, damit die Maske sie ohne Umweg liest.
	InputTaxFindings []InputTaxFinding `json:"inputTaxFindings"`

	// Conversion ist die Umrechnung eines Fremdwährungsbelegs: der verwendete
	// Tageskurs mit Quelle und Kurstag, der Durchschnittskurs der Umsatzsteuer
	// und die Differenz zwischen beiden. Nil bei einem Beleg in Euro.
	//
	// Sie steht in der Vorschau, weil der Anwender den Kurs sehen muss, bevor er
	// bucht — und weil er sonst nicht merkt, dass Buchfink den Kurs eines
	// früheren Handelstages als Näherung genommen hat.
	Conversion *Conversion `json:"conversion,omitempty"`
}

// PostingService turns business documents into journal entries using the
// deterministic Gruppe → Konten mapping.
type PostingService struct {
	journalSvc  *JournalService
	contactRepo domain.ContactRepository
	taxResolver domain.TaxResolver
	receiptSvc  *ReceiptService
	// provisions ist optional: ohne sie kennt der Belegweg keine
	// Rückstellungen und bucht wie zuvor gegen den Aufwand.
	provisions ProvisionConsumer
	// vendorAdvances ist ebenso optional: ohne sie kennt der Belegweg keine
	// geleisteten Anzahlungen.
	vendorAdvances domain.VendorAdvanceRepository
	// txRunner klammert die Buchung mit dem Vermerk der Anzahlungen. Fehlt er,
	// läuft alles wie zuvor, nur ohne die Klammer.
	txRunner domain.TxRunner
	// gifts ist die Geschenkkartei. Ohne sie kennt der Belegweg die Freigrenze
	// des § 4 Abs. 5 Satz 1 Nr. 1 EStG nicht.
	gifts giftRegister
	// fiscalStartMonth ist der Beginn des Geschäftsjahres. Null heißt Januar.
	//
	// Er wird über einen Leser geführt und nicht als Zahl gehalten: der Anwender
	// kann den Beginn des Geschäftsjahres ändern, und eine einmal beim
	// Einrichten gelesene Zahl rechnete danach bis zum Neustart mit dem alten
	// Wirtschaftsjahr.
	fiscalStartMonth  int
	fiscalStartReader func() int

	// currency ist der Kursdienst. Ohne ihn ist ein Beleg in Fremdwährung nicht
	// buchbar.
	currency currencyConverter
}

// SetTxRunner koppelt die Transaktionsklammer an den Belegweg.
func (s *PostingService) SetTxRunner(r domain.TxRunner) { s.txRunner = r }

// runInTx runs fn inside a transaction where one is wired, and plainly where it
// is not.
func (s *PostingService) runInTx(ctx context.Context, fn func(context.Context) error) error {
	if s.txRunner == nil {
		return fn(ctx)
	}
	return s.txRunner.RunInTx(ctx, fn)
}

// SetReceiptService wires in the Beleg service. Without it a booking cannot
// reference a Beleg, which is what the manual journal entry path relies on.
func (s *PostingService) SetReceiptService(receiptSvc *ReceiptService) { s.receiptSvc = receiptSvc }

// ProvisionConsumer ist der Ausschnitt der Rückstellungen, den der Belegweg
// braucht: wie viel eine Rückstellung von einer Rechnung deckt, und der
// Vermerk, dass sie verbraucht wurde.
type ProvisionConsumer interface {
	ConsumptionSplit(ctx context.Context, provisionID uint, net domain.Cents) (string, domain.Cents, error)
	RecordConsumption(ctx context.Context, provisionID uint, amount domain.Cents, date string, entryID uint, reason string) error
}

// SetProvisionConsumer koppelt die Rückstellungen an den Belegweg.
func (s *PostingService) SetProvisionConsumer(c ProvisionConsumer) { s.provisions = c }

// SetVendorAdvances koppelt die geleisteten Anzahlungen an den Belegweg. Ohne
// sie bucht der Belegweg wie zuvor, nur ohne den Anzahlungsfall.
func (s *PostingService) SetVendorAdvances(r domain.VendorAdvanceRepository) {
	s.vendorAdvances = r
}

// OpenVendorAdvances liefert die noch nicht verrechneten geleisteten
// Anzahlungen eines Lieferanten.
func (s *PostingService) OpenVendorAdvances(ctx context.Context, contactID uint) ([]domain.VendorAdvance, error) {
	if s.vendorAdvances == nil {
		return []domain.VendorAdvance{}, nil
	}
	return s.vendorAdvances.FindOpen(ctx, contactID)
}

// NewPostingService creates the posting service.
func NewPostingService(journalSvc *JournalService, contactRepo domain.ContactRepository) *PostingService {
	return &PostingService{
		journalSvc:  journalSvc,
		contactRepo: contactRepo,
		taxResolver: journalSvc.TaxResolver(),
	}
}

// PostIncomingReceipt books an Eingangsbeleg and seals its Beleg.
//
// The order is deliberate: the Beleg is checked for bookability first, then the
// journal writes, then the Beleg is sealed. Sealing belongs *behind* the journal
// write — if the booking fails the Beleg has to stay open so it can be corrected
// and booked again, whereas the other order would leave an unchangeable Beleg
// with no booking.
func (s *PostingService) PostIncomingReceipt(ctx context.Context, req ReceiptRequest) (*domain.JournalEntry, error) {
	built, err := s.buildIncomingLines(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := built.blockingError(req); err != nil {
		return nil, err
	}
	lines, contact, receipt := built.lines, built.contact, built.receipt

	description := req.Description
	if description == "" {
		description = fmt.Sprintf("Eingangsbeleg %s, %s", receipt.ReceiptNumber, contact.Name)
	}

	// Die Quelle „advance" ist keine Etikettierung, sondern die Bedingung der
	// Periodenzuordnung: accounting.VatPeriodFor legt die Steuer einer
	// Anzahlung in den Zeitraum der Zahlung. Ohne sie entschiede der
	// Leistungszeitraum — und geleistet ist bei einer Anzahlung noch nichts.
	source := domain.EntrySourceReceipt
	if req.AdvanceTarget != "" {
		source = domain.EntrySourceAdvance
		if req.Description == "" {
			description = fmt.Sprintf("Geleistete Anzahlung %s an %s", receipt.ReceiptNumber, contact.Name)
		}
	}

	entry := &domain.JournalEntry{
		BookingDate:        req.BookingDate,
		DocumentDate:       req.DocumentDate,
		ServiceDateFrom:    req.ServiceDateFrom,
		ServiceDateTo:      req.ServiceDateTo,
		Description:        description,
		Source:             source,
		DocumentNumber:     receipt.ReceiptNumber,
		TaxTreatment:       req.TaxTreatment,
		ReceiptID:          &receipt.ID,
		ReceiptHash:        receipt.ReceiptHash,
		ContactID:          &contact.ID,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
		Entertainment:      req.Entertainment,
		Gifts:              built.gifts,
	}
	// Währung, Kurs, Quelle und Kurstag gehören zusammen an den Kopf. Die
	// Währung allein — so stand es hier vorher — ließ den Kurs auf 1,000000
	// stehen: eine Buchung, die USD behauptet und Euro rechnet.
	if built.fx != nil {
		built.fx.head(entry)
	}

	// Buchung und der Vermerk der Anzahlungen gehören in eine Transaktion.
	//
	// Der Vermerk ist es, der eine geleistete Anzahlung als verrechnet ausweist
	// und sie aus der Liste der offenen nimmt. Bliebe er hinter der Buchung
	// aus, stünde die Schlussrechnung des Lieferanten im Journal, während die
	// Anzahlung weiter als offen gälte — und der zweite Versuch setzte sie ein
	// zweites Mal ab.
	var created *domain.JournalEntry
	if err := s.runInTx(ctx, func(ctx context.Context) error {
		posted, err := s.journalSvc.Post(ctx, entry)
		if err != nil {
			return err
		}
		created = posted

		// Die geleistete Anzahlung wird vermerkt, sobald die Zahlung gebucht
		// ist: die Schlussrechnung des Lieferanten muss sie absetzen können,
		// und ohne diesen Vermerk stünde sie nur als Saldo auf 1180 — ohne die
		// Angabe, aus welchem Vorgang er stammt.
		if req.AdvanceTarget != "" {
			if err := s.recordVendorAdvance(ctx, req, receipt, posted); err != nil {
				return fmt.Errorf("die Anzahlung ließ sich nicht vermerken: %w", err)
			}
		}
		for _, id := range req.SettledAdvanceIDs {
			advance, err := s.vendorAdvances.FindByID(ctx, id)
			if err != nil {
				return err
			}
			advance.SettledByEntryID = &posted.ID
			if err := s.vendorAdvances.Save(ctx, advance); err != nil {
				return fmt.Errorf("die Anzahlung %s bleibt als offen vermerkt: %w", advance.DocumentNumber, err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// Der Verbrauch wird in der Rückstellungskartei vermerkt, sobald die
	// Buchung steht: der Rückstellungsspiegel liest ihn dort und nicht aus dem
	// Journal, weil er die Bewegungsart braucht und nicht nur das Konto.
	if req.ProvisionID != 0 && s.provisions != nil {
		var consumed domain.Cents
		for _, l := range created.Lines {
			if l.Side == domain.SideDebit && l.Text == "Verbrauch der Rückstellung" {
				consumed += l.Amount
			}
		}
		if consumed > 0 {
			if err := s.provisions.RecordConsumption(
				ctx, req.ProvisionID, consumed, req.BookingDate, created.ID,
				fmt.Sprintf("Rechnung %s", receipt.ReceiptNumber),
			); err != nil {
				return created, fmt.Errorf(
					"die Buchung %s wurde geschrieben, der Verbrauch der Rückstellung aber nicht "+
						"vermerkt: %w", created.EntryNumber, err)
			}
		}
	}

	// Der Grund der Übersteuerung gehört an den Beleg und ins Protokoll — vor
	// der Versiegelung, weil ein versiegelter Beleg nichts mehr aufnimmt.
	if strings.TrimSpace(req.OverrideReason) != "" && len(built.findings) > 0 {
		if err := s.receiptSvc.SaveInputTaxOverride(ctx, receipt.ID, req.OverrideReason); err != nil {
			return created, fmt.Errorf(
				"die Buchung %s wurde geschrieben, der Grund der Übersteuerung aber nicht am Beleg "+
					"vermerkt: %w", created.EntryNumber, err)
		}
	}

	if err := s.receiptSvc.Seal(ctx, receipt.ID, created.ID); err != nil {
		// The booking stands; only the seal is missing. Saying so beats
		// pretending the booking failed, and ReceiptService repairs the state
		// when the Beleg is read again.
		return created, fmt.Errorf(
			"die Buchung %s wurde geschrieben, der Beleg %s konnte aber nicht versiegelt werden: %w",
			created.EntryNumber, receipt.ReceiptNumber, err)
	}
	return created, nil
}

// PreviewIncomingReceipt computes the booking of an Eingangsbeleg without
// writing it.
func (s *PostingService) PreviewIncomingReceipt(ctx context.Context, req ReceiptRequest) (*PostingPreview, error) {
	built, err := s.buildIncomingLines(ctx, req)
	if err != nil {
		return nil, err
	}
	preview := s.preview(ctx, built.lines)
	if notice := eInvoiceNotice(
		built.contact, built.receipt, req.TaxTreatment, req.DocumentDate, preview.Gross); notice != nil {
		preview.Warnings = append(preview.Warnings, *notice)
	}
	preview.Warnings = append(preview.Warnings, built.warnings...)
	preview.InputTaxFindings = built.findings
	if built.fx != nil {
		preview.Conversion = built.fx.conv
	}
	return preview, nil
}

// buildIncomingLines produces the journal lines of an Eingangsbeleg. It is the
// single implementation both the booking and its preview run through, so the two
// cannot disagree.
func (s *PostingService) buildIncomingLines(ctx context.Context, req ReceiptRequest) (*incomingLines, error) {
	if len(req.Positions) == 0 {
		return nil, fmt.Errorf("der Beleg hat keine Positionen")
	}
	// Kein stiller Vorgabewert: ein steuerfreier, ein nullbesteuerter und ein
	// dem Reverse-Charge unterliegender Einkauf sehen im Betrag gleich aus und
	// werden verschieden gebucht. Wo der Steuerfall fehlt — etwa weil der
	// Rechnungsdatensatz Kategorien mischt —, ist er zu wählen, nicht zu raten.
	if req.TaxTreatment == "" {
		return nil, fmt.Errorf(
			"der Steuerfall fehlt. Er ist anzugeben und lässt sich nicht aus den Beträgen erschließen")
	}

	contact, err := s.contactRepo.FindByID(ctx, req.ContactID)
	if err != nil {
		return nil, fmt.Errorf("Lieferant konnte nicht geladen werden: %w", err)
	}
	if contact.Type != domain.ContactTypeVendor {
		return nil, fmt.Errorf("%s ist als Kunde angelegt und kann keinen Eingangsbeleg stellen", contact.Name)
	}
	if err := validateIncomingTreatment(req.TaxTreatment, contact); err != nil {
		return nil, err
	}

	receipt, err := s.loadBookableReceipt(ctx, req.ReceiptID)
	if err != nil {
		return nil, err
	}

	if err := s.validateAdvanceRequest(req); err != nil {
		return nil, err
	}

	// Die Geschenke des Wirtschaftsjahres, einmal gelesen: die Freigrenze des
	// § 4 Abs. 5 Satz 1 Nr. 1 EStG läuft je Empfänger über das Jahr, und ohne
	// den bisherigen Stand ließe sich nicht sagen, ob dieses Geschenk sie reißt.
	giftTotals, err := s.giftTotals(ctx, req)
	if err != nil {
		return nil, err
	}

	// Die Umrechnung steht vor allem anderen: ab hier rechnet der ganze
	// Belegweg in Euro. Die Freigrenze für Geschenke, die 70/30-Aufteilung der
	// Bewirtung und jede Wertgrenze sind Eurobeträge, und sie gegen einen
	// Dollarbetrag zu halten wäre die stillste Art, sie falsch anzuwenden.
	fx, err := s.prepareCurrency(ctx, req)
	if err != nil {
		return nil, err
	}
	euroNet := make([]domain.Cents, len(req.Positions))
	for i := range req.Positions {
		euroNet[i] = req.Positions[i].Net
	}
	if fx != nil {
		foreign := make([]domain.Cents, len(req.Positions))
		for i := range req.Positions {
			foreign[i] = req.Positions[i].Net
		}
		euroNet = fx.splitToEuro(foreign)
	}

	out := &incomingLines{
		contact:  contact,
		receipt:  receipt,
		fx:       fx,
		gifts:    make([]domain.GiftRecord, 0, 1),
		warnings: make([]PostingWarning, 0, 2),
	}
	var lines []domain.JournalLine

	// 1. Aufwands- bzw. Anschaffungszeilen aus den fachlichen Gruppen.
	//
	// In zwei Durchgängen: erst stehen Konto, abziehbarer Vorsteueranteil und
	// die Aufzeichnung zum Geschenk je Position fest, dann entstehen die Zeilen.
	// Der Umweg ist die nicht abziehbare Vorsteuer eines geteilten Abzugs — sie
	// wird je Steuersatzgruppe einmal gerundet wie die Steuerzeile selbst, und
	// dafür müssen alle Positionen der Gruppe bekannt sein
	// (siehe nonDeductibleTaxByPosition).
	netByRate := map[domain.TaxRate]domain.Cents{}
	needsEntertainmentRecord := false
	prepared := make([]preparedPosition, 0, len(req.Positions))
	for i, p := range req.Positions {
		if p.Net <= 0 {
			return nil, fmt.Errorf("Position %d: der Nettobetrag muss größer als null sein", i+1)
		}
		foreignNet := domain.Cents(0)
		if fx != nil {
			foreignNet, p.Net = p.Net, euroNet[i]
			if p.Net <= 0 {
				return nil, fmt.Errorf(
					"Position %d: %s %s ergeben zum Kurs %s keinen Betrag über null",
					i+1, foreignNet, fx.currency, fx.conv.Rate.Source)
			}
		}
		// Die geleistete Anzahlung geht nicht durch die fachlichen Gruppen: sie
		// ist kein Aufwand, sondern ein Posten des Vermögens, und welcher,
		// entscheidet die Verwendung und nicht die Art der Leistung.
		if req.AdvanceTarget != "" {
			prepared = append(prepared, preparedPosition{
				position: p, foreignNet: foreignNet, advance: true})
			continue
		}
		resolved, err := s.resolvePosition(ctx, req, p, giftTotals)
		if err != nil {
			return nil, fmt.Errorf("Position %d: %w", i+1, err)
		}
		if resolved.gift != nil {
			out.gifts = append(out.gifts, *resolved.gift)
			giftTotals[resolved.gift.RecipientKey()] += resolved.gift.NetAmount
		}
		out.warnings = append(out.warnings, resolved.warnings...)
		prepared = append(prepared, preparedPosition{
			position: p, foreignNet: foreignNet, resolved: resolved})
	}

	nonDeductibleTax := nonDeductibleTaxByPosition(prepared)
	for i, pp := range prepared {
		if pp.advance {
			account, err := accounting.VendorAdvanceAccountFor(req.AdvanceTarget)
			if err != nil {
				return nil, err
			}
			text := pp.position.Text
			if text == "" {
				text = "Geleistete Anzahlung"
			}
			lines = append(lines, domain.JournalLine{
				Side: domain.SideDebit, Account: account, Amount: pp.position.Net,
				ForeignAmount: pp.foreignNet, Text: text,
			})
			netByRate[pp.position.TaxRate] += pp.position.Net
			continue
		}
		positionLines, quota, taxableNet, err := s.expenseLines(
			pp.resolved, nonDeductibleTax[i], req.TaxTreatment, req.DocumentDate)
		if err != nil {
			return nil, fmt.Errorf("Position %d: %w", i+1, err)
		}
		spreadForeign(positionLines, pp.foreignNet)
		lines = append(lines, positionLines...)
		if quota == accounting.QuotaEntertainment {
			needsEntertainmentRecord = true
		}
		// In die Bemessungsgrundlage geht der abziehbare Teil. Beim ungeteilten
		// Vorsteuerabzug ist das der volle Nettobetrag — auch dort, wo der
		// *Aufwand* geteilt wird: § 15 Abs. 1a Satz 2 UStG nimmt
		// Bewirtungsaufwendungen vom Vorsteuerausschluss ausdrücklich aus. Nur
		// ein Vorsteuerschlüssel oder ein Ausschluss nach § 15 Abs. 1a UStG
		// mindert sie.
		netByRate[pp.position.TaxRate] += taxableNet
	}

	if needsEntertainmentRecord {
		if req.Entertainment == nil {
			return nil, fmt.Errorf(
				"zu einer Bewirtung gehören Ort, Tag, Teilnehmer und Anlass (§ 4 Abs. 5 Satz 1 Nr. 2 EStG). Ohne diese Aufzeichnung ist der Abzug auch für die abziehbaren 70 %% verloren")
		}
		if err := req.Entertainment.Validate(); err != nil {
			return nil, err
		}
	}

	// 2. Die abgesetzten Anzahlungen. Sie stehen vor den Steuerzeilen, weil sie
	// deren Bemessungsgrundlage mindern: die Vorsteuer auf den angezahlten Teil
	// ist mit der Zahlung schon gezogen worden, und ein zweites Mal gäbe es sie
	// nicht.
	deductions, err := s.advanceDeductionLines(ctx, req, netByRate)
	if err != nil {
		return nil, err
	}
	lines = append(lines, deductions...)

	// 3. Steuerzeilen, einmal je Steuersatzgruppe gerundet. In Fremdwährung
	// kommt die Kursdifferenz zwischen Tages- und Umsatzsteuerkurs hinzu.
	taxLines, err := s.taxLinesInCurrency(domain.DirectionIncoming, req.TaxTreatment, netByRate, fx)
	if err != nil {
		return nil, err
	}
	lines = append(lines, taxLines...)

	// 4. Ist der Beleg einer Rückstellung zugeordnet, tritt das
	// Rückstellungskonto an die Stelle des Aufwands — bis zur Höhe ihres
	// Bestands.
	if req.ProvisionID != 0 {
		lines, err = s.applyProvision(ctx, req.ProvisionID, lines)
		if err != nil {
			return nil, err
		}
	}

	// 5. Gegenzeile: was tatsächlich an den Lieferanten zu zahlen ist.
	settlementLine, err := s.settlementLine(lines, req.Settlement, req.PaymentAccount, contact)
	if err != nil {
		return nil, err
	}
	if fx != nil {
		settlementLine.ForeignAmount = fx.toForeign(settlementLine.Amount)
	}
	lines = append(lines, settlementLine)

	// Zuletzt die Rechnungsprüfung: sie braucht die fertigen Zeilen, weil erst
	// aus ihnen hervorgeht, ob überhaupt Vorsteuer gezogen wird. Ein Beleg ohne
	// Vorsteuerabzug hat keine Pflichtangaben zu erfüllen, für die er gesperrt
	// werden müsste.
	findings, err := s.inputTaxFindings(ctx, req, contact, receipt, lines)
	if err != nil {
		return nil, err
	}
	out.findings = findings
	out.lines = lines
	return out, nil
}

// validateAdvanceRequest prüft die beiden Bedingungen des Anzahlungsfalls auf
// der Eingangsseite.
//
// Die erste ist die Zahlung: § 15 Abs. 1 Satz 1 Nr. 1 Satz 3 UStG lässt den
// Vorsteuerabzug aus einer Anzahlungsrechnung erst zu, wenn das Entgelt
// entrichtet ist. Eine offene Anzahlungsrechnung ist deshalb nichts zu buchen —
// geliefert ist nichts, gezahlt ist nichts, und die Vorsteuer gibt es noch
// nicht. Die zweite ist die Trennung der Wege: ein Beleg ist entweder eine
// Anzahlung oder ihre Schlussrechnung.
func (s *PostingService) validateAdvanceRequest(req ReceiptRequest) error {
	if req.AdvanceTarget == "" && len(req.SettledAdvanceIDs) == 0 {
		return nil
	}
	if s.vendorAdvances == nil {
		return fmt.Errorf("die geleisteten Anzahlungen sind nicht eingerichtet")
	}
	if req.AdvanceTarget != "" && len(req.SettledAdvanceIDs) > 0 {
		return fmt.Errorf(
			"ein Beleg ist entweder eine Anzahlung oder die Schlussrechnung, die sie absetzt — nicht beides")
	}
	if req.AdvanceTarget == "" {
		return nil
	}
	if _, err := accounting.VendorAdvanceAccountFor(req.AdvanceTarget); err != nil {
		return err
	}
	if req.Settlement != SettlementPaid {
		return fmt.Errorf(
			"eine Anzahlungsrechnung wird mit der Zahlung gebucht, nicht mit ihrem Eingang: " +
				"der Vorsteuerabzug setzt nach § 15 Abs. 1 Satz 1 Nr. 1 Satz 3 UStG die Entrichtung des " +
				"Entgelts voraus. Buche sie mit dem Zahlungstag und dem Zahlungsmittel")
	}
	if req.ProvisionID != 0 {
		return fmt.Errorf("eine geleistete Anzahlung verbraucht keine Rückstellung: sie ist kein Aufwand")
	}
	return nil
}

// advanceDeductionLines setzt die geleisteten Anzahlungen ab, die dieser Beleg
// verrechnet, und mindert dabei die Bemessungsgrundlage der Vorsteuer.
//
// Die Schlussrechnung des Lieferanten weist den Gesamtbetrag aus. Abzuziehen
// ist davon, was schon gezahlt und dessen Vorsteuer schon gezogen wurde —
// andernfalls stünde die Anzahlung doppelt im Vermögen und die Vorsteuer
// zweimal in der Voranmeldung.
func (s *PostingService) advanceDeductionLines(
	ctx context.Context, req ReceiptRequest, netByRate map[domain.TaxRate]domain.Cents,
) ([]domain.JournalLine, error) {
	if len(req.SettledAdvanceIDs) == 0 {
		return nil, nil
	}
	advances, err := s.loadSettledAdvances(ctx, req)
	if err != nil {
		return nil, err
	}

	var lines []domain.JournalLine
	for i := range advances {
		a := &advances[i]
		if netByRate[a.TaxRate] < a.NetAmount {
			return nil, fmt.Errorf(
				"die Anzahlung %s über %s € netto übersteigt, was die Schlussrechnung zu diesem Steuersatz "+
					"abrechnet (%s €)", a.DocumentNumber, a.NetAmount, netByRate[a.TaxRate])
		}
		netByRate[a.TaxRate] -= a.NetAmount
		lines = append(lines, domain.JournalLine{
			Side: domain.SideCredit, Account: a.Account, Amount: a.NetAmount,
			Text: "Verrechnete Anzahlung " + a.DocumentNumber,
		})
	}
	return lines, nil
}

// loadSettledAdvances lädt die abzusetzenden Anzahlungen und prüft, dass sie zu
// diesem Lieferanten gehören und noch offen sind.
func (s *PostingService) loadSettledAdvances(ctx context.Context, req ReceiptRequest) ([]domain.VendorAdvance, error) {
	out := make([]domain.VendorAdvance, 0, len(req.SettledAdvanceIDs))
	seen := map[uint]bool{}
	for _, id := range req.SettledAdvanceIDs {
		if seen[id] {
			return nil, fmt.Errorf("die Anzahlung %d ist zweimal angegeben", id)
		}
		seen[id] = true
		advance, err := s.vendorAdvances.FindByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("die geleistete Anzahlung %d wurde nicht gefunden: %w", id, err)
		}
		if advance.ContactID != req.ContactID {
			return nil, fmt.Errorf(
				"die Anzahlung %s gehört zu einem anderen Lieferanten", advance.DocumentNumber)
		}
		if advance.Settled() {
			return nil, fmt.Errorf(
				"die Anzahlung %s ist bereits mit einer Schlussrechnung verrechnet", advance.DocumentNumber)
		}
		out = append(out, *advance)
	}
	return out, nil
}

// recordVendorAdvance hält die gezahlte Anzahlung fest, damit die
// Schlussrechnung sie absetzen kann.
func (s *PostingService) recordVendorAdvance(
	ctx context.Context, req ReceiptRequest, receipt *domain.Receipt, entry *domain.JournalEntry,
) error {
	account, err := accounting.VendorAdvanceAccountFor(req.AdvanceTarget)
	if err != nil {
		return err
	}
	var net, gross domain.Cents
	rate := domain.TaxRateNone
	for _, p := range req.Positions {
		if p.TaxRate > rate {
			rate = p.TaxRate
		}
	}
	for _, l := range entry.Lines {
		// Der Nettobetrag kommt aus der Buchung und nicht aus der Anfrage: bei
		// einem Beleg in Fremdwährung lauten deren Positionen auf die
		// Fremdwährung, und die Kartei der geleisteten Anzahlungen führt Euro.
		// Aus beidem eine Summe zu bilden hieße, Dollar und Euro zu addieren.
		if l.Side == domain.SideDebit && l.Account == account {
			net += l.Amount
		}
		// Die Gegenzeile trägt den tatsächlich gezahlten Betrag; sie ist die
		// einzige Zeile auf einem Zahlungsmittelkonto.
		if l.Side == domain.SideCredit && isLiquidAccount(l.Account) {
			gross += l.Amount
		}
	}
	return s.vendorAdvances.Save(ctx, &domain.VendorAdvance{
		ContactID:      req.ContactID,
		ReceiptID:      receipt.ID,
		EntryID:        entry.ID,
		DocumentNumber: receipt.ReceiptNumber,
		Account:        account,
		Target:         string(req.AdvanceTarget),
		NetAmount:      net,
		TaxAmount:      gross - net,
		GrossAmount:    gross,
		TaxRate:        rate,
		PaidAt:         req.BookingDate,
	})
}

// applyProvision ersetzt Aufwandszeilen durch eine Zeile auf dem
// Rückstellungskonto.
//
// Aufgezehrt wird in der Reihenfolge der Positionen, und der Rest bleibt
// stehen: übersteigt die Rechnung die Rückstellung, ist der Mehrbetrag Aufwand
// des laufenden Jahres. Die Steuerzeilen bleiben unberührt — die Vorsteuer
// entsteht mit der Rechnung und hat mit der Rückstellung nichts zu tun.
func (s *PostingService) applyProvision(
	ctx context.Context, provisionID uint, lines []domain.JournalLine,
) ([]domain.JournalLine, error) {
	if s.provisions == nil {
		return nil, fmt.Errorf("die Rückstellungen sind nicht angebunden")
	}
	var net domain.Cents
	for _, l := range lines {
		if l.Side == domain.SideDebit && !s.taxResolver.IsTaxAccount(l.Account) {
			net += l.Amount
		}
	}
	account, covered, err := s.provisions.ConsumptionSplit(ctx, provisionID, net)
	if err != nil {
		return nil, err
	}

	out := make([]domain.JournalLine, 0, len(lines)+1)
	out = append(out, domain.JournalLine{
		Side: domain.SideDebit, Account: account, Amount: covered,
		Text: "Verbrauch der Rückstellung",
	})
	remaining := covered
	for _, l := range lines {
		if l.Side != domain.SideDebit || s.taxResolver.IsTaxAccount(l.Account) || remaining <= 0 {
			out = append(out, l)
			continue
		}
		if l.Amount <= remaining {
			remaining -= l.Amount
			continue
		}
		l.Amount -= remaining
		remaining = 0
		out = append(out, l)
	}
	return out, nil
}

// loadBookableReceipt fetches the Beleg and runs the bookability check — the
// second of the two checks, the one filing must not impose.
//
// A Beleg is required, not optional. "Keine Buchung ohne Beleg" is the rule the
// whole flow is built on; where no external document exists the user files an
// Eigenbeleg. Leaving the reference optional would reintroduce the freehand
// Belegfeld this change set removed.
func (s *PostingService) loadBookableReceipt(ctx context.Context, receiptID uint) (*domain.Receipt, error) {
	if receiptID == 0 {
		return nil, fmt.Errorf("zu jedem Eingangsbeleg gehört ein abgelegter Beleg. Liegt kein Dokument des Lieferanten vor, ist ein Eigenbeleg abzulegen")
	}
	if s.receiptSvc == nil {
		return nil, fmt.Errorf("die Belegverwaltung ist nicht eingerichtet")
	}
	receipt, err := s.receiptSvc.Get(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	if receipt.Direction != domain.DirectionIncoming {
		return nil, fmt.Errorf("Beleg %s ist ein Ausgangsbeleg und passt nicht zu einem Eingangsbeleg", receipt.ReceiptNumber)
	}
	if receipt.Status == domain.ReceiptStatusSealed {
		return nil, fmt.Errorf("Beleg %s ist bereits gebucht", receipt.ReceiptNumber)
	}
	if err := receipt.ValidateBookable(); err != nil {
		return nil, err
	}
	return receipt, nil
}

// preview turns a set of lines into the numbers a user checks against the paper
// invoice, with account names filled in for display.
func (s *PostingService) preview(ctx context.Context, lines []domain.JournalLine) *PostingPreview {
	out := make([]domain.JournalLine, len(lines))
	copy(out, lines)

	if chart, err := s.journalSvc.Chart(ctx); err == nil {
		for i := range out {
			out[i].AccountName = chart.Name(out[i].Account)
		}
	}
	for i := range out {
		out[i].Position = i + 1
	}

	var debit, credit, net, gross domain.Cents
	for i := range out {
		l := &out[i]
		if l.Side == domain.SideDebit {
			debit += l.Amount
		} else {
			credit += l.Amount
		}
	}
	// The settlement line is the last one and carries what is actually paid or
	// received; everything before it that is not a tax line is the net amount.
	if len(out) > 0 {
		gross = out[len(out)-1].Amount
		for i := 0; i < len(out)-1; i++ {
			if out[i].TaxKey == "" {
				net += out[i].Amount
			}
		}
	}

	return &PostingPreview{
		Lines:            out,
		Net:              net,
		Tax:              gross - net,
		Gross:            gross,
		Balanced:         debit == credit,
		InputTaxFindings: make([]InputTaxFinding, 0),
	}
}

// PostOutgoingInvoice books an Ausgangsrechnung as a receivable.
func (s *PostingService) PostOutgoingInvoice(ctx context.Context, inv *domain.Invoice, contact *domain.Contact) (*domain.JournalEntry, error) {
	lines, fx, err := s.buildOutgoingLines(ctx, inv, contact)
	if err != nil {
		return nil, err
	}

	entry := &domain.JournalEntry{
		FiscalYear:      inv.FiscalYear,
		BookingDate:     inv.Date,
		DocumentDate:    inv.Date,
		ServiceDateFrom: inv.ServiceDateFrom,
		ServiceDateTo:   inv.ServiceDateTo,
		Description:     fmt.Sprintf("Rechnung %s an %s", inv.InvoiceNumber, contact.Name),
		// Der Ausgangsbeleg entsteht vor der Buchung (siehe InvoiceService.Issue)
		// und wird hier an ihr vermerkt. Die Gegenrichtung — der Beleg zeigt nach
		// dem Versiegeln auf die Buchung — genügt nicht: der Prüflauf fragt die
		// Buchung, ob sie einen Beleg hat, und ohne diese Zeile hätte jede
		// Ausgangsrechnung keinen.
		ReceiptID:          inv.ReceiptID,
		Source:             domain.EntrySourceInvoice,
		DocumentNumber:     inv.InvoiceNumber,
		TaxTreatment:       inv.TaxTreatment,
		ContactID:          &contact.ID,
		Currency:           inv.Currency,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	}
	if fx != nil {
		fx.head(entry)
	}

	return s.journalSvc.Post(ctx, entry)
}

// PostCashSale bucht eine Rechnung ohne Empfänger gegen das Zahlungsmittel.
//
// Der Fall ist die Kleinbetragsrechnung des § 33 UStDV, die ohne
// Leistungsempfänger auskommt. Ohne Empfänger gibt es kein Personenkonto — und
// es gibt auch nichts, was darauf stehen könnte: der Barverkauf ist bezahlt,
// wenn die Rechnung entsteht. Ein Sammel-Debitor wäre die Alternative gewesen;
// er trüge einen offenen Posten, den niemand ausgleicht, und die OP-Liste
// verlöre ihre Aussage.
func (s *PostingService) PostCashSale(ctx context.Context, inv *domain.Invoice, account string) (*domain.JournalEntry, error) {
	if account == "" {
		account = domain.AccountKasse
	}
	lines, fx, err := s.outgoingContentLines(ctx, inv)
	if err != nil {
		return nil, err
	}
	settlement, err := settlementLineFor(lines, SettlementPaid, account, nil)
	if err != nil {
		return nil, err
	}
	if fx != nil {
		settlement.ForeignAmount = fx.toForeign(settlement.Amount)
	}
	lines = append(lines, settlement)

	entry := &domain.JournalEntry{
		FiscalYear:         inv.FiscalYear,
		BookingDate:        inv.Date,
		DocumentDate:       inv.Date,
		ServiceDateFrom:    inv.ServiceDateFrom,
		ServiceDateTo:      inv.ServiceDateTo,
		Description:        fmt.Sprintf("Kleinbetragsrechnung %s (Barverkauf)", inv.InvoiceNumber),
		ReceiptID:          inv.ReceiptID,
		Source:             domain.EntrySourceInvoice,
		DocumentNumber:     inv.InvoiceNumber,
		TaxTreatment:       inv.TaxTreatment,
		Currency:           inv.Currency,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	}
	if fx != nil {
		fx.head(entry)
	}
	return s.journalSvc.Post(ctx, entry)
}

// PreviewCashSale computes the booking of a Kleinbetragsrechnung without a
// recipient.
func (s *PostingService) PreviewCashSale(ctx context.Context, inv *domain.Invoice, account string) (*PostingPreview, error) {
	if account == "" {
		account = domain.AccountKasse
	}
	lines, fx, err := s.outgoingContentLines(ctx, inv)
	if err != nil {
		return nil, err
	}
	settlement, err := settlementLineFor(lines, SettlementPaid, account, nil)
	if err != nil {
		return nil, err
	}
	if fx != nil {
		settlement.ForeignAmount = fx.toForeign(settlement.Amount)
	}
	preview := s.preview(ctx, append(lines, settlement))
	if fx != nil {
		preview.Conversion = fx.conv
	}
	return preview, nil
}

// PreviewOutgoingInvoice computes the booking of an Ausgangsrechnung without
// writing it. The invoice form shows this instead of doing the arithmetic again.
func (s *PostingService) PreviewOutgoingInvoice(ctx context.Context, inv *domain.Invoice, contact *domain.Contact) (*PostingPreview, error) {
	lines, fx, err := s.buildOutgoingLines(ctx, inv, contact)
	if err != nil {
		return nil, err
	}
	preview := s.preview(ctx, lines)
	if fx != nil {
		preview.Conversion = fx.conv
	}
	return preview, nil
}

// buildOutgoingLines produces the journal lines of an Ausgangsrechnung, shared by
// the booking and its preview.
func (s *PostingService) buildOutgoingLines(
	ctx context.Context, inv *domain.Invoice, contact *domain.Contact,
) ([]domain.JournalLine, *fxContext, error) {
	lines, fx, err := s.outgoingContentLines(ctx, inv)
	if err != nil {
		return nil, nil, err
	}
	// An issued invoice is always an open item; the payment is a later,
	// separate business transaction.
	settlementLine, err := s.settlementLine(lines, SettlementOpen, "", contact)
	if err != nil {
		return nil, nil, err
	}
	if fx != nil {
		settlementLine.ForeignAmount = fx.toForeign(settlementLine.Amount)
	}
	return append(lines, settlementLine), fx, nil
}

// outgoingContentLines sind Erlös- und Steuerzeilen einer Ausgangsrechnung —
// alles außer der Gegenzeile.
//
// Getrennt, weil die Schlussrechnung dieselben Zeilen braucht und zwischen
// ihnen und der Forderung noch die Auflösung der Anzahlungen einfügt: die
// Gegenzeile ergibt sich dort erst aus dem, was nach der Verrechnung übrig
// bleibt.
func (s *PostingService) outgoingContentLines(
	ctx context.Context, inv *domain.Invoice,
) ([]domain.JournalLine, *fxContext, error) {
	group, err := accounting.LookupPostingGroup("erloese")
	if err != nil {
		return nil, nil, err
	}
	if len(inv.Items) == 0 {
		return nil, nil, fmt.Errorf("die Rechnung hat keine Positionen")
	}
	treatment := inv.TaxTreatment
	if treatment == "" {
		treatment = domain.TaxTreatmentDomestic
	}

	// Eine Rechnung in Fremdwährung lautet in ihren Positionen auf diese
	// Währung. Gebucht wird in Euro, zum EZB-Referenzkurs des Rechnungsdatums —
	// ohne Kurs gar nicht. Die frühere Fassung reichte die Währung an den
	// Buchungskopf durch und ließ den Kurs auf 1,000000 stehen.
	fx, err := s.prepareInvoiceCurrency(ctx, inv)
	if err != nil {
		return nil, nil, err
	}
	amounts := make([]domain.Cents, len(inv.Items))
	for i := range inv.Items {
		amounts[i] = inv.Items[i].TotalNet()
	}
	foreign := append([]domain.Cents(nil), amounts...)
	if fx != nil {
		amounts = fx.splitToEuro(foreign)
	}

	var lines []domain.JournalLine
	netByRate := map[domain.TaxRate]domain.Cents{}

	for i := range inv.Items {
		item := &inv.Items[i]
		g := group
		if item.PostingGroup != "" {
			g, err = accounting.LookupPostingGroup(item.PostingGroup)
			if err != nil {
				return nil, nil, fmt.Errorf("Position %d: %w", i+1, err)
			}
			if g.Direction != domain.DirectionOutgoing {
				return nil, nil, fmt.Errorf("Position %d: %q ist keine Ertragsgruppe", i+1, g.Label)
			}
		}
		line := domain.JournalLine{
			Side:    domain.SideCredit,
			Account: g.ResolveAccount(treatment, item.TaxRate),
			Amount:  amounts[i],
			Text:    item.Description,
		}
		if fx != nil {
			line.ForeignAmount = foreign[i]
		}
		lines = append(lines, line)
		netByRate[item.TaxRate] += amounts[i]
	}

	taxLines, err := s.taxLinesInCurrency(domain.DirectionOutgoing, treatment, netByRate, fx)
	if err != nil {
		return nil, nil, err
	}
	return append(lines, taxLines...), fx, nil
}

// prepareInvoiceCurrency baut die Umrechnung einer Ausgangsrechnung.
func (s *PostingService) prepareInvoiceCurrency(
	ctx context.Context, inv *domain.Invoice,
) (*fxContext, error) {
	code := strings.ToUpper(strings.TrimSpace(inv.Currency))
	if code == "" || code == "EUR" {
		return nil, nil
	}
	var total domain.Cents
	for i := range inv.Items {
		total += inv.Items[i].TotalNet()
	}
	if total <= 0 {
		return nil, fmt.Errorf("die Rechnung hat einen Gesamtbetrag von null")
	}
	if s.currency == nil {
		return nil, fmt.Errorf(
			"der Kursdienst ist nicht eingerichtet. Eine Rechnung über %s ließe sich nur mit einem "+
				"geratenen Kurs buchen, und geraten wird keiner", code)
	}
	conv, err := s.currency.Convert(ctx, code, inv.Date, total)
	if err != nil {
		return nil, fmt.Errorf("für die Rechnung %s in %s: %w", inv.InvoiceNumber, code, err)
	}
	return &fxContext{currency: code, conv: conv}, nil
}

// expenseLines turns one position into its expense lines.
//
// Most positions produce exactly one. An expense that is only partly deductible
// produces two: the deductible share on the group's account and the rest on the
// non-deductible one. Booking both to one account would leave the Steuerbilanz
// wrong, and splitting it later is not possible — the information which part was
// which is gone by then.
func (s *PostingService) expenseLines(
	r resolvedPosition, nonDeductibleTax domain.Cents,
	treatment domain.TaxTreatment, documentDate string,
) ([]domain.JournalLine, accounting.DeductibleQuota, domain.Cents, error) {
	p := r.position

	// Was von der Vorsteuer nicht abziehbar ist, gehört zum Aufwand
	// (§ 9b Abs. 1 EStG). Der Betrag kommt von außen: er wird je
	// Steuersatzgruppe gerechnet und nicht je Position — siehe
	// nonDeductibleTaxByPosition.
	taxableNet := taxableNetOf(r)
	// Der Anteil steht nur an der Zeile, wo er nicht der volle ist. Der
	// Ausschluss des § 15 Abs. 1a UStG bekommt seinen eigenen Wert: null
	// Promille abziehbar ist etwas anderes als „kein Vorsteuerschlüssel", und
	// die Zahl null bedeutet an dieser Stelle das Zweite.
	share := 0
	switch {
	case r.permille == 0:
		share = domain.InputTaxExcluded
	case r.permille < 1000:
		share = int(r.permille)
	}
	text := appendText(p.Text, r.note)

	if r.account != "" {
		// Ein unmittelbar gewähltes Konto durchläuft weiter die Prüfungen des
		// Journals; es ist der Notausgang für Fälle, die der Gruppenkatalog nicht
		// abdeckt — und für das Geschenk über der Freigrenze, dessen Konto der
		// Belegweg selbst gesetzt hat.
		return []domain.JournalLine{{
			Side: domain.SideDebit, Account: r.account, Amount: p.Net + nonDeductibleTax,
			InputTaxShare: share, Text: text,
		}}, "", taxableNet, nil
	}
	if !r.hasGroup {
		return nil, "", 0, fmt.Errorf("weder eine Buchungsgruppe noch ein Konto angegeben")
	}
	group := r.group

	account := group.ResolveAccount(treatment, p.TaxRate)
	if group.NonDeductibleAccount == "" || group.DeductibleQuota == "" {
		return []domain.JournalLine{{
			Side: domain.SideDebit, Account: account, Amount: p.Net + nonDeductibleTax,
			InputTaxShare: share, Text: text,
		}}, "", taxableNet, nil
	}

	params, err := accounting.TaxParametersFor(documentDate)
	if err != nil {
		return nil, "", 0, err
	}
	permille := deductiblePermille(group.DeductibleQuota, params)
	if permille <= 0 || permille >= 1000 {
		return nil, "", 0, fmt.Errorf("für %q ist kein abziehbarer Anteil hinterlegt", group.Label)
	}

	base := p.Net + nonDeductibleTax
	deductible := domain.MulRound(base, permille, 1000)
	// Der Rest ergibt sich als Differenz und wird nicht ein zweites Mal
	// gerundet — sonst summierten sich die beiden Zeilen an einem Cent vorbei.
	nonDeductible := base - deductible

	lines := []domain.JournalLine{
		{Side: domain.SideDebit, Account: account, Amount: deductible,
			InputTaxShare: share, Text: text},
	}
	if nonDeductible > 0 {
		lines = append(lines, domain.JournalLine{
			Side: domain.SideDebit, Account: group.NonDeductibleAccount,
			Amount: nonDeductible, InputTaxShare: share, Text: text,
		})
	}
	return lines, group.DeductibleQuota, taxableNet, nil
}

// preparedPosition ist eine Belegposition, nachdem Umrechnung, Konto und
// abziehbarer Vorsteueranteil feststehen und bevor ihre Zeilen entstehen.
type preparedPosition struct {
	// position trägt den Nettobetrag bereits in Euro; foreignNet den Betrag der
	// Fremdwährung, aus dem er entstanden ist.
	position   ReceiptPosition
	foreignNet domain.Cents
	// advance kennzeichnet die Position eines Anzahlungsbelegs. Sie geht nicht
	// durch die fachlichen Gruppen und trägt keinen geteilten Vorsteuerabzug.
	advance  bool
	resolved resolvedPosition
}

// taxableNetOf liefert den Teil des Nettobetrags, auf den Vorsteuer gezogen
// wird — bei ungeteiltem Abzug der ganze.
func taxableNetOf(r resolvedPosition) domain.Cents {
	if r.permille >= 1000 {
		return r.position.Net
	}
	return domain.MulRound(r.position.Net, r.permille, 1000)
}

// nonDeductibleTaxByPosition verteilt die nicht abziehbare Vorsteuer des
// geteilten Abzugs auf die Positionen.
//
// Gerundet wird einmal je Steuersatzgruppe — genau wie die Steuerzeile, die aus
// derselben Gruppe entsteht (§ 9b Abs. 1 EStG macht den nicht abziehbaren Teil
// zum Aufwand, sagt aber nichts über die Reihenfolge der Rundung). Wird je
// Position gerundet, gehen Sollzeilen und Steuerzeile an unterschiedlichen Cents
// vorbei: die Summe der Buchung weicht dann um ein bis zwei Cent vom
// Bruttobetrag der Rechnung ab, die Gegenzeile gleicht das still aus, und die
// Verbindlichkeit gegenüber dem Lieferanten stimmt mit seiner Rechnung nicht
// mehr überein. Der Rest der Gruppe liegt auf ihrer letzten geteilten Position —
// dieselbe Regel wie beim Ausgleich der Umrechnung.
func nonDeductibleTaxByPosition(prepared []preparedPosition) []domain.Cents {
	out := make([]domain.Cents, len(prepared))
	netByRate := map[domain.TaxRate]domain.Cents{}
	taxableByRate := map[domain.TaxRate]domain.Cents{}
	assignedByRate := map[domain.TaxRate]domain.Cents{}
	lastSplit := map[domain.TaxRate]int{}

	for i, pp := range prepared {
		if pp.advance {
			continue
		}
		rate := pp.position.TaxRate
		taxable := taxableNetOf(pp.resolved)
		netByRate[rate] += pp.position.Net
		taxableByRate[rate] += taxable
		if pp.resolved.permille >= 1000 {
			continue
		}
		// Die Aufteilung je Position ist der Vorschlag; sie hält die Verteilung
		// nah an den Beträgen, aus denen sie stammt.
		out[i] = rate.Tax(pp.position.Net) - rate.Tax(taxable)
		assignedByRate[rate] += out[i]
		lastSplit[rate] = i
	}

	for rate, idx := range lastSplit {
		// Was die Gruppe insgesamt trägt: die Differenz der beiden einmal
		// gerundeten Steuerbeträge. Der Unterschied zur Summe der Vorschläge
		// liegt auf der letzten geteilten Position.
		total := rate.Tax(netByRate[rate]) - rate.Tax(taxableByRate[rate])
		out[idx] += total - assignedByRate[rate]
	}
	return out
}

// appendText hängt einen Zusatz an den Buchungstext.
func appendText(text, addition string) string {
	switch {
	case addition == "":
		return text
	case text == "":
		return addition
	default:
		return text + " — " + addition
	}
}

func deductiblePermille(quota accounting.DeductibleQuota, params accounting.TaxParameters) int64 {
	switch quota {
	case accounting.QuotaEntertainment:
		return params.EntertainmentDeductiblePermille
	default:
		return 0
	}
}

// taxLines produces the tax legs, rounding once per rate group.
func (s *PostingService) taxLines(dir domain.Direction, treatment domain.TaxTreatment, netByRate map[domain.TaxRate]domain.Cents) ([]domain.JournalLine, error) {
	rates := make([]domain.TaxRate, 0, len(netByRate))
	for r := range netByRate {
		rates = append(rates, r)
	}
	sort.Slice(rates, func(i, j int) bool { return rates[i] < rates[j] })

	var lines []domain.JournalLine
	for _, rate := range rates {
		legs, err := s.taxResolver.Resolve(dir, treatment, rate, netByRate[rate])
		if err != nil {
			return nil, err
		}
		for _, leg := range legs {
			if leg.Amount == 0 {
				continue
			}
			lines = append(lines, taxLegLine(leg))
		}
	}
	return lines, nil
}

// taxLegLine turns a Steuerbein into the journal line that carries it.
//
// The renaming between the two types — Base to TaxBase, Key to TaxKey — is
// written once. A second copy is where a field added to TaxLeg reaches one
// booking path and silently misses the other.
func taxLegLine(leg domain.TaxLeg) domain.JournalLine {
	return domain.JournalLine{
		Side:    leg.Side,
		Account: leg.Account,
		Amount:  leg.Amount,
		TaxKey:  leg.Key,
		TaxBase: leg.Base,
	}
}

// settlementLine closes the entry against the partner's Personenkonto or a
// liquid account.
//
// Its amount is whatever is needed to balance the entry, which makes it correct
// for every Steuerfall without a special case: a domestic purchase settles at
// net plus input tax, a Reverse-Charge purchase settles at net, because there
// the input tax leg is cancelled out by the output tax leg.
func (s *PostingService) settlementLine(content []domain.JournalLine, kind SettlementKind, paymentAccount string, contact *domain.Contact) (domain.JournalLine, error) {
	return settlementLineFor(content, kind, paymentAccount, contact)
}

// settlementLineFor is the same computation without a service around it. Die
// Anlagenbuchhaltung bucht Erhaltungsaufwand und laufende Erträge nach derselben
// Regel; eine zweite Fassung davon wäre die, die beim nächsten Steuerfall
// vergessen wird.
func settlementLineFor(content []domain.JournalLine, kind SettlementKind, paymentAccount string, contact *domain.Contact) (domain.JournalLine, error) {
	var debit, credit domain.Cents
	for _, l := range content {
		if l.Side == domain.SideDebit {
			debit += l.Amount
		} else {
			credit += l.Amount
		}
	}

	difference := debit - credit
	if difference == 0 {
		return domain.JournalLine{}, fmt.Errorf("der Beleg hat einen Gesamtbetrag von null")
	}

	side := domain.SideCredit
	if difference < 0 {
		side = domain.SideDebit
		difference = -difference
	}

	switch kind {
	case SettlementOpen:
		return domain.JournalLine{
			Side:      side,
			Account:   contact.LedgerAccount,
			ContactID: &contact.ID,
			Amount:    difference,
		}, nil
	case SettlementPaid:
		if paymentAccount == "" {
			return domain.JournalLine{}, fmt.Errorf("bei sofortiger Zahlung muss das Zahlungsmittel angegeben werden (Kasse, Bank, Kreditkarte)")
		}
		if !isLiquidAccount(paymentAccount) {
			return domain.JournalLine{}, fmt.Errorf("Konto %s ist kein Zahlungsmittelkonto", paymentAccount)
		}
		return domain.JournalLine{Side: side, Account: paymentAccount, Amount: difference}, nil
	default:
		return domain.JournalLine{}, fmt.Errorf("unbekannte Zahlungsart %q", kind)
	}
}

func isLiquidAccount(account string) bool {
	for _, a := range domain.LiquidAccounts() {
		if a == account {
			return true
		}
	}
	return false
}

// validateIncomingTreatment blocks Steuerfälle the partner's master data does
// not support. Reverse Charge without a VAT identification number on file is the
// single most common source of a wrong Umsatzsteuer-Voranmeldung.
func validateIncomingTreatment(treatment domain.TaxTreatment, contact *domain.Contact) error {
	switch treatment {
	case domain.TaxTreatmentIntraCommunityAcquisition:
		if !contact.IsEUCounterparty() {
			return fmt.Errorf("ein innergemeinschaftlicher Erwerb setzt einen Lieferanten in einem anderen EU-Land voraus, %s ist in %q erfasst", contact.Name, contact.CountryCode)
		}
		if contact.VatID == "" {
			return fmt.Errorf("für einen innergemeinschaftlichen Erwerb braucht %s eine USt-IdNr.", contact.Name)
		}
	case domain.TaxTreatmentReverseCharge:
		// § 13b UStG hat zwei getrennte Fälle. Bei einer sonstigen Leistung eines
		// im übrigen Gemeinschaftsgebiet ansässigen Unternehmers (Abs. 1) belegt
		// die USt-IdNr. die Unternehmereigenschaft im Ausland. Bei einer
		// inländischen Bauleistung (Abs. 2 Nr. 4) hängt die Steuerschuld dagegen
		// am *Leistungsempfänger* – er muss selbst nachhaltig Bauleistungen
		// erbringen (Abs. 5). Das steht nicht in den Lieferantenstammdaten, also
		// darf eine fehlende USt-IdNr. des inländischen Lieferanten den Fall auch
		// nicht blockieren.
		if contact.IsEUCounterparty() && contact.VatID == "" {
			return fmt.Errorf(
				"für § 13b UStG braucht der ausländische Lieferant %s eine USt-IdNr., die seine Unternehmereigenschaft belegt",
				contact.Name)
		}
	}
	return nil
}

// PostAdvanceSettlement bucht die Vereinnahmung einer Anzahlung.
//
// SOLL Zahlungsmittel an HABEN „Erhaltene, versteuerte Anzahlungen" (netto) und
// HABEN Umsatzsteuer. Hier — und nicht beim Ausstellen der Abschlagsrechnung —
// entsteht die Steuer: § 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG lässt sie mit
// Ablauf des Voranmeldungszeitraums entstehen, in dem das Teilentgelt
// vereinnahmt worden ist, und das gilt auch bei Sollversteuerung.
//
// Die Buchung trägt die Quelle „advance". Daran erkennt die Periodenzuordnung
// den Fall (accounting.VatPeriodFor): sonst entschiede der Leistungszeitraum,
// und der liegt bei einer Anzahlung noch in der Zukunft.
//
// bankTxID ist der Bankumsatz, aus dem das Geld stammt, sofern die
// Vereinnahmung aus dem Kontoauszug kommt. Er gehört an die Buchung wie bei
// jeder anderen Zahlung: ohne ihn ließe sich die Buchung dem Umsatz nicht mehr
// zuordnen, und der Import wüsste nicht, dass er erledigt ist.
func (s *PostingService) PostAdvanceSettlement(
	ctx context.Context,
	advance *domain.AdvanceItem,
	contact *domain.Contact,
	paymentDate string,
	paymentAccount string,
	bankTxID *uint,
) (*domain.JournalEntry, error) {
	account, err := accounting.AdvanceAccountFor(advance.TaxRate)
	if err != nil {
		return nil, err
	}

	lines := []domain.JournalLine{{
		Side: domain.SideCredit, Account: account, Amount: advance.NetAmount,
		Text: "Erhaltene Anzahlung " + advance.InvoiceNumber,
	}}
	taxLines, err := s.taxLines(domain.DirectionOutgoing, domain.TaxTreatmentDomestic,
		map[domain.TaxRate]domain.Cents{advance.TaxRate: advance.NetAmount})
	if err != nil {
		return nil, err
	}
	lines = append(lines, taxLines...)

	settlement, err := settlementLineFor(lines, SettlementPaid, paymentAccount, contact)
	if err != nil {
		return nil, err
	}
	lines = append(lines, settlement)

	return s.journalSvc.Post(ctx, &domain.JournalEntry{
		BookingDate:  paymentDate,
		DocumentDate: paymentDate,
		// Leistungsdatum und Zahlungsdatum fallen hier zusammen: maßgeblich ist
		// die Vereinnahmung, und die Zeile darf keinen Zeitraum behaupten, in
		// dem noch nichts geleistet wurde.
		ServiceDateFrom: paymentDate,
		ServiceDateTo:   paymentDate,
		Description: fmt.Sprintf("Anzahlung %s von %s vereinnahmt",
			advance.InvoiceNumber, contact.Name),
		Source:             domain.EntrySourceAdvance,
		DocumentNumber:     advance.InvoiceNumber,
		TaxTreatment:       domain.TaxTreatmentDomestic,
		ContactID:          &contact.ID,
		BankTxID:           bankTxID,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	})
}

// PostAdvanceRefund bucht die Rückzahlung einer vereinnahmten Anzahlung.
//
// Sie ist das Gegenstück zu PostAdvanceSettlement und keine Generalumkehr: eine
// Generalumkehr nähme die Buchung im Zeitraum ihrer Entstehung zurück, die
// Rückzahlung ist aber ein eigener Vorgang mit eigenem Datum. § 17 Abs. 1
// Satz 8 UStG legt die Berichtigung in den Zeitraum, in dem sich die
// Bemessungsgrundlage geändert hat — und das ist der Tag, an dem das Geld
// zurückgeflossen ist.
func (s *PostingService) PostAdvanceRefund(
	ctx context.Context,
	advance *domain.AdvanceItem,
	contact *domain.Contact,
	refundDate string,
	paymentAccount string,
	reason string,
) (*domain.JournalEntry, error) {
	account, err := accounting.AdvanceAccountFor(advance.TaxRate)
	if err != nil {
		return nil, err
	}

	lines := []domain.JournalLine{{
		Side: domain.SideDebit, Account: account, Amount: advance.NetAmount,
		Text: "Rückzahlung Anzahlung " + advance.InvoiceNumber,
	}}
	legs, err := s.taxResolver.Resolve(
		domain.DirectionOutgoing, domain.TaxTreatmentDomestic, advance.TaxRate, advance.NetAmount)
	if err != nil {
		return nil, err
	}
	for _, leg := range legs {
		if leg.Amount == 0 {
			continue
		}
		line := taxLegLine(leg)
		// Gegenseite der Vereinnahmung, derselbe Steuerschlüssel: die Minderung
		// muss in derselben Zeile der Voranmeldung erscheinen wie die Steuer,
		// die sie zurücknimmt.
		line.Side = leg.Side.Opposite()
		line.Text = "Steuerkorrektur Rückzahlung (§ 17 Abs. 2 Nr. 2 UStG)"
		lines = append(lines, line)
	}

	settlement, err := settlementLineFor(lines, SettlementPaid, paymentAccount, contact)
	if err != nil {
		return nil, err
	}
	lines = append(lines, settlement)

	return s.journalSvc.Post(ctx, &domain.JournalEntry{
		BookingDate:     refundDate,
		DocumentDate:    refundDate,
		ServiceDateFrom: refundDate,
		ServiceDateTo:   refundDate,
		Description: fmt.Sprintf("Anzahlung %s an %s zurückgezahlt: %s",
			advance.InvoiceNumber, contact.Name, reason),
		Source:             domain.EntrySourceAdvance,
		DocumentNumber:     advance.InvoiceNumber,
		TaxTreatment:       domain.TaxTreatmentDomestic,
		ContactID:          &contact.ID,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	})
}

// PostFinalInvoice bucht die Schlussrechnung samt Auflösung der Anzahlungen.
//
// Der Gesamtbetrag wird als Erlös und Umsatzsteuer erfasst, und die bereits
// vereinnahmten Anzahlungen werden aufgelöst: SOLL Anzahlungskonto (netto) und
// SOLL Umsatzsteuer (die Steuer der Anzahlungen). Als Forderung bleibt der
// Restbetrag stehen.
//
// Die Steuerzeile der Auflösung trägt denselben Steuerschlüssel wie die der
// Rechnung, nur auf der Gegenseite. Damit meldet die Voranmeldung des
// Leistungszeitraums genau die Differenz — die Steuer der Anzahlungen ist im
// Zeitraum ihrer Vereinnahmung schon angemeldet worden, und sie ein zweites Mal
// zu melden wäre der doppelte Ausweis, den § 14 Abs. 5 Satz 2 UStG verhindern
// will.
func (s *PostingService) PostFinalInvoice(
	ctx context.Context,
	inv *domain.Invoice,
	contact *domain.Contact,
	advances []domain.AdvanceItem,
) (*domain.JournalEntry, error) {
	lines, fx, err := s.outgoingContentLines(ctx, inv)
	if err != nil {
		return nil, err
	}
	if fx != nil {
		// Die vereinnahmten Anzahlungen stehen in Euro in der Kartei. Sie gegen
		// eine Schlussrechnung in Fremdwährung zu verrechnen hieße, zwei Beträge
		// unterschiedlicher Währung voneinander abzuziehen — Buchfink bildet das
		// nicht ab und sagt es, statt eine Zahl zu erfinden.
		return nil, fmt.Errorf(
			"die Schlussrechnung %s lautet auf %s, die verrechneten Anzahlungen stehen in Euro. "+
				"Dieser Fall ist in Buchfink nicht abgebildet — stelle die Schlussrechnung in Euro aus",
			inv.InvoiceNumber, fx.currency)
	}

	netByRate := map[domain.TaxRate]domain.Cents{}
	for i := range advances {
		netByRate[advances[i].TaxRate] += advances[i].NetAmount
	}
	rates := make([]domain.TaxRate, 0, len(netByRate))
	for r := range netByRate {
		rates = append(rates, r)
	}
	sort.Slice(rates, func(i, j int) bool { return rates[i] < rates[j] })

	for _, rate := range rates {
		net := netByRate[rate]
		account, err := accounting.AdvanceAccountFor(rate)
		if err != nil {
			return nil, err
		}
		lines = append(lines, domain.JournalLine{
			Side: domain.SideDebit, Account: account, Amount: net,
			Text: "Verrechnete Anzahlungen",
		})
		legs, err := s.taxResolver.Resolve(domain.DirectionOutgoing, domain.TaxTreatmentDomestic, rate, net)
		if err != nil {
			return nil, err
		}
		for _, leg := range legs {
			if leg.Amount == 0 {
				continue
			}
			line := taxLegLine(leg)
			line.Side = leg.Side.Opposite()
			line.Text = "Steuer der verrechneten Anzahlungen (§ 14 Abs. 5 Satz 2 UStG)"
			lines = append(lines, line)
		}
	}

	settlement, err := s.settlementLine(lines, SettlementOpen, "", contact)
	if err != nil {
		return nil, err
	}
	lines = append(lines, settlement)

	return s.journalSvc.Post(ctx, &domain.JournalEntry{
		FiscalYear:         inv.FiscalYear,
		BookingDate:        inv.Date,
		DocumentDate:       inv.Date,
		ServiceDateFrom:    inv.ServiceDateFrom,
		ServiceDateTo:      inv.ServiceDateTo,
		Description:        fmt.Sprintf("Schlussrechnung %s an %s", inv.InvoiceNumber, contact.Name),
		Source:             domain.EntrySourceInvoice,
		DocumentNumber:     inv.InvoiceNumber,
		TaxTreatment:       inv.TaxTreatment,
		ContactID:          &contact.ID,
		Currency:           inv.Currency,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	})
}
