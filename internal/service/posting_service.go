package service

import (
	"context"
	"fmt"
	"sort"

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
	Currency       string              `json:"currency,omitempty"`

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
	lines, contact, receipt, err := s.buildIncomingLines(ctx, req)
	if err != nil {
		return nil, err
	}

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
		Currency:           req.Currency,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
		Entertainment:      req.Entertainment,
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
	lines, contact, receipt, err := s.buildIncomingLines(ctx, req)
	if err != nil {
		return nil, err
	}
	preview := s.preview(ctx, lines)
	if notice := eInvoiceNotice(contact, receipt, req.TaxTreatment, req.DocumentDate, preview.Gross); notice != nil {
		preview.Warnings = append(preview.Warnings, *notice)
	}
	return preview, nil
}

// buildIncomingLines produces the journal lines of an Eingangsbeleg. It is the
// single implementation both the booking and its preview run through, so the two
// cannot disagree.
func (s *PostingService) buildIncomingLines(ctx context.Context, req ReceiptRequest) ([]domain.JournalLine, *domain.Contact, *domain.Receipt, error) {
	if len(req.Positions) == 0 {
		return nil, nil, nil, fmt.Errorf("der Beleg hat keine Positionen")
	}
	// Kein stiller Vorgabewert: ein steuerfreier, ein nullbesteuerter und ein
	// dem Reverse-Charge unterliegender Einkauf sehen im Betrag gleich aus und
	// werden verschieden gebucht. Wo der Steuerfall fehlt — etwa weil der
	// Rechnungsdatensatz Kategorien mischt —, ist er zu wählen, nicht zu raten.
	if req.TaxTreatment == "" {
		return nil, nil, nil, fmt.Errorf(
			"der Steuerfall fehlt. Er ist anzugeben und lässt sich nicht aus den Beträgen erschließen")
	}

	contact, err := s.contactRepo.FindByID(ctx, req.ContactID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Lieferant konnte nicht geladen werden: %w", err)
	}
	if contact.Type != domain.ContactTypeVendor {
		return nil, nil, nil, fmt.Errorf("%s ist als Kunde angelegt und kann keinen Eingangsbeleg stellen", contact.Name)
	}
	if err := validateIncomingTreatment(req.TaxTreatment, contact); err != nil {
		return nil, nil, nil, err
	}

	receipt, err := s.loadBookableReceipt(ctx, req.ReceiptID)
	if err != nil {
		return nil, nil, nil, err
	}

	if err := s.validateAdvanceRequest(req); err != nil {
		return nil, nil, nil, err
	}

	var lines []domain.JournalLine

	// 1. Aufwands- bzw. Anschaffungszeilen aus den fachlichen Gruppen.
	netByRate := map[domain.TaxRate]domain.Cents{}
	needsEntertainmentRecord := false
	for i, p := range req.Positions {
		if p.Net <= 0 {
			return nil, nil, nil, fmt.Errorf("Position %d: der Nettobetrag muss größer als null sein", i+1)
		}
		// Die geleistete Anzahlung geht nicht durch die fachlichen Gruppen: sie
		// ist kein Aufwand, sondern ein Posten des Vermögens, und welcher,
		// entscheidet die Verwendung und nicht die Art der Leistung.
		if req.AdvanceTarget != "" {
			account, err := accounting.VendorAdvanceAccountFor(req.AdvanceTarget)
			if err != nil {
				return nil, nil, nil, err
			}
			text := p.Text
			if text == "" {
				text = "Geleistete Anzahlung"
			}
			lines = append(lines, domain.JournalLine{
				Side: domain.SideDebit, Account: account, Amount: p.Net, Text: text,
			})
			netByRate[p.TaxRate] += p.Net
			continue
		}
		positionLines, quota, err := s.expenseLines(p, req.TaxTreatment, req.DocumentDate)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("Position %d: %w", i+1, err)
		}
		lines = append(lines, positionLines...)
		if quota == accounting.QuotaEntertainment {
			needsEntertainmentRecord = true
		}
		// Die Bemessungsgrundlage bleibt der volle Nettobetrag, auch wo der
		// Aufwand geteilt wird: § 15 Abs. 1a Satz 2 UStG nimmt Bewirtungs-
		// aufwendungen vom Vorsteuerausschluss ausdrücklich aus.
		netByRate[p.TaxRate] += p.Net
	}

	if needsEntertainmentRecord {
		if req.Entertainment == nil {
			return nil, nil, nil, fmt.Errorf(
				"zu einer Bewirtung gehören Ort, Tag, Teilnehmer und Anlass (§ 4 Abs. 5 Satz 1 Nr. 2 EStG). Ohne diese Aufzeichnung ist der Abzug auch für die abziehbaren 70 %% verloren")
		}
		if err := req.Entertainment.Validate(); err != nil {
			return nil, nil, nil, err
		}
	}

	// 2. Die abgesetzten Anzahlungen. Sie stehen vor den Steuerzeilen, weil sie
	// deren Bemessungsgrundlage mindern: die Vorsteuer auf den angezahlten Teil
	// ist mit der Zahlung schon gezogen worden, und ein zweites Mal gäbe es sie
	// nicht.
	deductions, err := s.advanceDeductionLines(ctx, req, netByRate)
	if err != nil {
		return nil, nil, nil, err
	}
	lines = append(lines, deductions...)

	// 3. Steuerzeilen, einmal je Steuersatzgruppe gerundet.
	taxLines, err := s.taxLines(domain.DirectionIncoming, req.TaxTreatment, netByRate)
	if err != nil {
		return nil, nil, nil, err
	}
	lines = append(lines, taxLines...)

	// 4. Ist der Beleg einer Rückstellung zugeordnet, tritt das
	// Rückstellungskonto an die Stelle des Aufwands — bis zur Höhe ihres
	// Bestands.
	if req.ProvisionID != 0 {
		lines, err = s.applyProvision(ctx, req.ProvisionID, lines)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	// 5. Gegenzeile: was tatsächlich an den Lieferanten zu zahlen ist.
	settlementLine, err := s.settlementLine(lines, req.Settlement, req.PaymentAccount, contact)
	if err != nil {
		return nil, nil, nil, err
	}
	lines = append(lines, settlementLine)

	return lines, contact, receipt, nil
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
		net += p.Net
		if p.TaxRate > rate {
			rate = p.TaxRate
		}
	}
	for _, l := range entry.Lines {
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
		Lines:    out,
		Net:      net,
		Tax:      gross - net,
		Gross:    gross,
		Balanced: debit == credit,
	}
}

// PostOutgoingInvoice books an Ausgangsrechnung as a receivable.
func (s *PostingService) PostOutgoingInvoice(ctx context.Context, inv *domain.Invoice, contact *domain.Contact) (*domain.JournalEntry, error) {
	lines, err := s.buildOutgoingLines(inv, contact)
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
	lines, err := s.outgoingContentLines(inv)
	if err != nil {
		return nil, err
	}
	settlement, err := settlementLineFor(lines, SettlementPaid, account, nil)
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
		Description:        fmt.Sprintf("Kleinbetragsrechnung %s (Barverkauf)", inv.InvoiceNumber),
		ReceiptID:          inv.ReceiptID,
		Source:             domain.EntrySourceInvoice,
		DocumentNumber:     inv.InvoiceNumber,
		TaxTreatment:       inv.TaxTreatment,
		Currency:           inv.Currency,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	})
}

// PreviewCashSale computes the booking of a Kleinbetragsrechnung without a
// recipient.
func (s *PostingService) PreviewCashSale(ctx context.Context, inv *domain.Invoice, account string) (*PostingPreview, error) {
	if account == "" {
		account = domain.AccountKasse
	}
	lines, err := s.outgoingContentLines(inv)
	if err != nil {
		return nil, err
	}
	settlement, err := settlementLineFor(lines, SettlementPaid, account, nil)
	if err != nil {
		return nil, err
	}
	return s.preview(ctx, append(lines, settlement)), nil
}

// PreviewOutgoingInvoice computes the booking of an Ausgangsrechnung without
// writing it. The invoice form shows this instead of doing the arithmetic again.
func (s *PostingService) PreviewOutgoingInvoice(ctx context.Context, inv *domain.Invoice, contact *domain.Contact) (*PostingPreview, error) {
	lines, err := s.buildOutgoingLines(inv, contact)
	if err != nil {
		return nil, err
	}
	return s.preview(ctx, lines), nil
}

// buildOutgoingLines produces the journal lines of an Ausgangsrechnung, shared by
// the booking and its preview.
func (s *PostingService) buildOutgoingLines(inv *domain.Invoice, contact *domain.Contact) ([]domain.JournalLine, error) {
	lines, err := s.outgoingContentLines(inv)
	if err != nil {
		return nil, err
	}
	// An issued invoice is always an open item; the payment is a later,
	// separate business transaction.
	settlementLine, err := s.settlementLine(lines, SettlementOpen, "", contact)
	if err != nil {
		return nil, err
	}
	return append(lines, settlementLine), nil
}

// outgoingContentLines sind Erlös- und Steuerzeilen einer Ausgangsrechnung —
// alles außer der Gegenzeile.
//
// Getrennt, weil die Schlussrechnung dieselben Zeilen braucht und zwischen
// ihnen und der Forderung noch die Auflösung der Anzahlungen einfügt: die
// Gegenzeile ergibt sich dort erst aus dem, was nach der Verrechnung übrig
// bleibt.
func (s *PostingService) outgoingContentLines(inv *domain.Invoice) ([]domain.JournalLine, error) {
	group, err := accounting.LookupPostingGroup("erloese")
	if err != nil {
		return nil, err
	}
	if len(inv.Items) == 0 {
		return nil, fmt.Errorf("die Rechnung hat keine Positionen")
	}
	treatment := inv.TaxTreatment
	if treatment == "" {
		treatment = domain.TaxTreatmentDomestic
	}

	var lines []domain.JournalLine
	netByRate := map[domain.TaxRate]domain.Cents{}

	for i := range inv.Items {
		item := &inv.Items[i]
		g := group
		if item.PostingGroup != "" {
			g, err = accounting.LookupPostingGroup(item.PostingGroup)
			if err != nil {
				return nil, fmt.Errorf("Position %d: %w", i+1, err)
			}
			if g.Direction != domain.DirectionOutgoing {
				return nil, fmt.Errorf("Position %d: %q ist keine Ertragsgruppe", i+1, g.Label)
			}
		}
		lines = append(lines, domain.JournalLine{
			Side:    domain.SideCredit,
			Account: g.ResolveAccount(treatment, item.TaxRate),
			Amount:  item.TotalNet(),
			Text:    item.Description,
		})
		netByRate[item.TaxRate] += item.TotalNet()
	}

	taxLines, err := s.taxLines(domain.DirectionOutgoing, treatment, netByRate)
	if err != nil {
		return nil, err
	}
	return append(lines, taxLines...), nil
}

// expenseLines turns one position into its expense lines.
//
// Most positions produce exactly one. An expense that is only partly deductible
// produces two: the deductible share on the group's account and the rest on the
// non-deductible one. Booking both to one account would leave the Steuerbilanz
// wrong, and splitting it later is not possible — the information which part was
// which is gone by then.
func (s *PostingService) expenseLines(p ReceiptPosition, treatment domain.TaxTreatment, documentDate string) ([]domain.JournalLine, accounting.DeductibleQuota, error) {
	if p.Account != "" {
		// A directly chosen account still has to pass the journal's checks; this
		// is the escape hatch for cases the group catalog does not cover.
		return []domain.JournalLine{{
			Side: domain.SideDebit, Account: p.Account, Amount: p.Net, Text: p.Text,
		}}, "", nil
	}
	if p.PostingGroup == "" {
		return nil, "", fmt.Errorf("weder eine Buchungsgruppe noch ein Konto angegeben")
	}
	group, err := accounting.LookupPostingGroup(p.PostingGroup)
	if err != nil {
		return nil, "", err
	}
	if group.Direction != domain.DirectionIncoming {
		return nil, "", fmt.Errorf("%q passt nicht zur Belegrichtung", group.Label)
	}

	account := group.ResolveAccount(treatment, p.TaxRate)
	if group.NonDeductibleAccount == "" {
		return []domain.JournalLine{{
			Side: domain.SideDebit, Account: account, Amount: p.Net, Text: p.Text,
		}}, "", nil
	}

	params, err := accounting.TaxParametersFor(documentDate)
	if err != nil {
		return nil, "", err
	}
	permille := deductiblePermille(group.DeductibleQuota, params)
	if permille <= 0 || permille >= 1000 {
		return nil, "", fmt.Errorf("für %q ist kein abziehbarer Anteil hinterlegt", group.Label)
	}

	deductible := domain.MulRound(p.Net, permille, 1000)
	// Der Rest ergibt sich als Differenz und wird nicht ein zweites Mal
	// gerundet — sonst summierten sich die beiden Zeilen an einem Cent vorbei.
	nonDeductible := p.Net - deductible

	lines := []domain.JournalLine{
		{Side: domain.SideDebit, Account: account, Amount: deductible, Text: p.Text},
	}
	if nonDeductible > 0 {
		lines = append(lines, domain.JournalLine{
			Side: domain.SideDebit, Account: group.NonDeductibleAccount,
			Amount: nonDeductible, Text: p.Text,
		})
	}
	return lines, group.DeductibleQuota, nil
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
	lines, err := s.outgoingContentLines(inv)
	if err != nil {
		return nil, err
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
