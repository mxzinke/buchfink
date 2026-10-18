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
// Payment timing and payment means are two independent inputs: a receipt paid
// on the spot names the bank account it was paid from.
type SettlementKind string

const (
	// SettlementOpen books against the partner's Personenkonto and leaves an
	// open item.
	SettlementOpen SettlementKind = "open"
	// SettlementPaid books directly against a liquid account.
	SettlementPaid SettlementKind = "paid"
	// SettlementCreditCard ist die Zahlung mit der Firmenkreditkarte: der
	// Betrag steht auf 3610, bis die Monatsabrechnung vom Bankkonto abgebucht
	// wird.
	SettlementCreditCard SettlementKind = "credit_card"
	// SettlementShareholderExpense ist die Auslage: ein Gesellschafter oder
	// Geschäftsführer hat privat bezahlt und bekommt den Betrag erstattet.
	// Aussteller bleibt der Lieferant, Gläubiger der Erstattung ist er.
	SettlementShareholderExpense SettlementKind = "shareholder_expense"
)

// paidAtReceipt meldet die Zahlungswege, bei denen das Entgelt mit dem Beleg
// entrichtet ist.
func (k SettlementKind) paidAtReceipt() bool {
	return k == SettlementPaid || k == SettlementCreditCard || k == SettlementShareholderExpense
}

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
	// CreditorContactID ist bei einer Auslage der Gesellschafter oder
	// Geschäftsführer, dem der Betrag zu erstatten ist.
	CreditorContactID uint `json:"creditorContactId,omitempty"`

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
	// Dann bucht Buchfink gegen das Rückstellungskonto: die Verpflichtung, für
	// die zurückgestellt wurde, wird erfüllt, und der Aufwand ist im Vorjahr
	// schon entstanden. Ein zweites Mal Aufwand zu buchen wäre die häufigste
	// Art, eine Rückstellung falsch abzuwickeln. Was die Rückstellung nicht
	// deckt, bleibt Aufwand.
	ProvisionID uint `json:"provisionId,omitempty"`

	// AdvanceTarget kennzeichnet den Beleg als geleistete Anzahlung und sagt,
	// wofür angezahlt wurde.
	//
	// Der Beleg wird dann auf das Konto der geleisteten Anzahlungen gebucht:
	// bezahlt ist etwas, geliefert nichts. Und er wird
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

	// ServiceProof ist der Leistungsnachweis, den die Maske mit der Buchung
	// mitschickt, ServiceProofAt sein Datum (leer heißt heute) — RECH-08.
	//
	// Ab der eingestellten Grenze verlangt der Belegweg den Vermerk, bevor er
	// bucht. Ohne dieses Feld müsste der Buchungsdialog zweimal rufen
	// (SaveServiceProof, dann Buchen) und die zweite Hälfte des Vorgangs an der
	// Oberfläche zusammenhalten — mit dem Ergebnis, dass ein abgebrochener
	// Vorgang einen Vermerk hinterlässt, aber keine Buchung. Geschrieben wird
	// der Vermerk über den Belegdienst, damit er dieselbe Prüfung und denselben
	// Protokolleintrag bekommt wie der nachgetragene, und innerhalb der
	// Transaktion der Buchung: eine gescheiterte Buchung lässt keinen Vermerk
	// zurück.
	ServiceProof   string `json:"serviceProof,omitempty"`
	ServiceProofAt string `json:"serviceProofAt,omitempty"`
}

// PostingPreview is what a booking would look like, computed without writing it.
//
// It exists so the user interface never re-implements the tax rules. A second
// computation in the frontend is a second truth that drifts the moment a
// Steuerfall is added — and the one in the frontend is the one nobody tests.
type PostingPreview struct {
	IssuingFindings []string             `json:"issuingFindings,omitempty"`
	Lines           []domain.JournalLine `json:"lines"`
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
	// gifts ist die Geschenkkartei. Ohne sie kennt der Belegweg die Freigrenze
	// des § 4 Abs. 5 Satz 1 Nr. 1 EStG nicht.
	gifts giftRegister
	// currency ist der Kursdienst. Ohne ihn ist ein Beleg in Fremdwährung nicht
	// buchbar.
	currency currencyConverter

	// settingsRepo liefert die Nachweisgrenze des Leistungsnachweises
	// (RECH-08). Ohne ihn kennt der Belegweg keine Grenze und bucht wie zuvor —
	// gesetzt wird er deshalb überall dort, wo es Einstellungen gibt.
	settingsRepo domain.SettingsRepository
}

// SetSettingsSource koppelt die Unternehmenseinstellungen an den Belegweg. Aus
// ihnen kommt die Grenze, ab der ein Eingangsbeleg einen Leistungsnachweis braucht.
func (s *PostingService) SetSettingsSource(repo domain.SettingsRepository) {
	s.settingsRepo = repo
}

func (s *PostingService) runInTx(ctx context.Context, fn func(context.Context) error) error {
	return s.journalSvc.InTx(ctx, fn)
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

// PostIncomingReceipt plant und bucht einen Eingangsbeleg samt Belegsiegel.
// Die Planung liegt vor der Transaktion; Kurse werden im Kern vor Write geholt.
func (s *PostingService) PostIncomingReceipt(ctx context.Context, req ReceiptRequest) (*domain.JournalEntry, error) {
	plan, err := s.planIncomingReceipt(ctx, req)
	if err != nil {
		return nil, err
	}
	return s.bookIncomingReceipt(ctx, plan)
}

// PreviewIncomingReceipt zeigt denselben Plan, den die Buchung verwendet.
func (s *PostingService) PreviewIncomingReceipt(ctx context.Context, req ReceiptRequest) (*PostingPreview, error) {
	plan, err := s.planIncomingReceipt(ctx, req)
	if err != nil {
		return nil, err
	}
	return plan.preview, nil
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
	if !req.Settlement.paidAtReceipt() {
		return fmt.Errorf(
			"eine Anzahlungsrechnung wird mit der Zahlung gebucht, nicht mit ihrem Eingang: " +
				"der Vorsteuerabzug setzt nach § 15 Abs. 1 Satz 1 Nr. 1 Satz 3 UStG die Entrichtung des " +
				"Entgelts voraus. Sie wird mit dem Zahlungstag und dem Zahlungsmittel gebucht")
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
		// Die Gegenzeile hat den tatsächlich gezahlten Betrag; sie ist die
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
		// Der Ausgangsbeleg entsteht vor der Buchung (siehe
		// InvoiceService.issueDocument); die Buchung verweist auf ihn.
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

// taxLinesForShare baut die Steuerzeilen mit zwei Bemessungsgrundlagen
// (UST-07 K2).
//
// Beim steuerpflichtigen Inlandsumsatz fallen die beiden zusammen: es gibt eine
// Steuerzeile, und der Vorsteuerschlüssel mindert sie. Beim § 13b-Umsatz und
// beim innergemeinschaftlichen Erwerb entstehen zwei Zeilen aus derselben
// Bemessungsgrundlage, und der Schlüssel wirkt nur auf eine davon: die
// geschuldete Steuer schuldet der Leistungsempfänger in voller Höhe (§ 13b
// Abs. 5 UStG, § 1a UStG — Kennziffern 46/47 bzw. 89/93), abziehbar ist sie nur
// zum betrieblichen Anteil (§ 15 Abs. 1 Satz 1 Nr. 3 und 4, Abs. 4 UStG —
// Kennziffern 67 bzw. 61). Wer beide Zeilen kürzte, meldete einen Erwerb, den es
// nicht gab; wer keine kürzte, zöge Vorsteuer, die ihm nicht zusteht.
//
// Der nicht abziehbare Teil ist damit endgültig Aufwand (§ 9b Abs. 1 EStG); er
// steht schon in den Aufwandszeilen (siehe nonDeductibleTaxByPosition).
func (s *PostingService) taxLinesForShare(
	dir domain.Direction, treatment domain.TaxTreatment,
	deductibleByRate, owedByRate map[domain.TaxRate]domain.Cents, fx *fxContext,
) ([]domain.JournalLine, error) {
	owed, err := s.taxLinesInCurrency(dir, treatment, owedByRate, fx)
	if err != nil {
		return nil, err
	}
	if sameBases(deductibleByRate, owedByRate) {
		return owed, nil
	}
	reduced, err := s.taxLinesInCurrency(dir, treatment, deductibleByRate, fx)
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]domain.JournalLine, len(reduced))
	for _, l := range reduced {
		if l.TaxKey != "" {
			byKey[l.TaxKey] = l
		}
	}

	out := make([]domain.JournalLine, 0, len(owed))
	for _, l := range owed {
		if !isInputTaxKey(l.TaxKey) {
			out = append(out, l)
			continue
		}
		match, ok := byKey[l.TaxKey]
		if !ok || match.Amount == 0 {
			// Kein abziehbarer Anteil: die Vorsteuerzeile entfällt ganz. Eine
			// Zeile über null wäre ein Steuerkonto ohne Betrag im Journal.
			continue
		}
		l.Amount, l.TaxBase, l.ForeignAmount = match.Amount, match.TaxBase, match.ForeignAmount
		out = append(out, l)
	}
	return out, nil
}

// isInputTaxKey meldet, ob ein Steuerschlüssel eine Vorsteuerzeile kennzeichnet.
//
// Erkennbar am Schlüssel und nicht am Konto: der Schlüssel entscheidet über die
// Kennziffer, und genau die Kennziffern der Vorsteuer (61, 66, 67) sind es, die
// der Vorsteuerschlüssel kürzt.
func isInputTaxKey(key string) bool {
	return strings.HasPrefix(key, "VST") || strings.HasSuffix(key, "_VST")
}

// sameBases meldet, ob zwei Bemessungsgrundlagen übereinstimmen.
func sameBases(a, b map[domain.TaxRate]domain.Cents) bool {
	for rate, value := range a {
		if b[rate] != value {
			return false
		}
	}
	for rate, value := range b {
		if a[rate] != value {
			return false
		}
	}
	return true
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
// Anlagenbuchhaltung bucht laufende Erträge nach derselben Regel; eine zweite
// Fassung davon wäre die, die beim nächsten Steuerfall vergessen wird.
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
			return domain.JournalLine{}, fmt.Errorf("bei sofortiger Zahlung muss das Bankkonto angegeben werden")
		}
		if !isLiquidAccount(paymentAccount) {
			return domain.JournalLine{}, fmt.Errorf("Konto %s ist kein Zahlungsmittelkonto (Bank 1800 bis 1850 oder Geldtransit 1460)", paymentAccount)
		}
		return domain.JournalLine{Side: side, Account: paymentAccount, Amount: difference}, nil
	case SettlementCreditCard:
		return domain.JournalLine{
			Side: side, Account: domain.AccountKreditkartenabrechnung, Amount: difference, Text: "Firmenkreditkarte",
		}, nil
	case SettlementShareholderExpense:
		// Konto und Gläubiger setzt expenseCreditor.
		return domain.JournalLine{Side: side, Amount: difference}, nil
	default:
		return domain.JournalLine{}, fmt.Errorf("unbekannte Zahlungsart %q", kind)
	}
}

// expenseCreditor setzt die Gegenzeile einer Auslage: die Erstattungsschuld
// gegenüber dem Gesellschafter (3640) oder dem Geschäftsführer, der kein
// Gesellschafter ist (3500), mit ihm als Kontakt. Der Lieferant bleibt
// Aussteller des Belegs und Kontakt der Buchung.
func (s *PostingService) expenseCreditor(
	ctx context.Context, req ReceiptRequest, supplier *domain.Contact, line *domain.JournalLine,
) error {
	if req.CreditorContactID == 0 {
		return fmt.Errorf("bei einer Auslage fehlt der Gesellschafter oder Geschäftsführer, der bezahlt hat")
	}
	if req.CreditorContactID == supplier.ID {
		return fmt.Errorf(
			"%s ist Aussteller des Belegs und kann nicht zugleich die Auslage erstattet bekommen; "+
				"eine Rechnung des Gesellschafters wird offen gebucht", supplier.Name)
	}
	creditor, err := s.contactRepo.FindByID(ctx, req.CreditorContactID)
	if err != nil {
		return fmt.Errorf("der Gläubiger der Auslage wurde nicht gefunden: %w", err)
	}
	if err := ensureNotBlocked(creditor); err != nil {
		return err
	}
	if s.settingsRepo == nil {
		return fmt.Errorf("ohne Unternehmensdaten lässt sich der Gläubiger einer Auslage nicht zuordnen")
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return fmt.Errorf("die Unternehmensdaten ließen sich nicht lesen: %w", err)
	}
	role := companyRoleOf(cfg, creditor)
	switch {
	case role.shareholder:
		line.Account = domain.AccountVerbindlichkeitenGesellschafter
	case role.director:
		line.Account = domain.AccountSonstigeVerbindlichkeiten
	default:
		return fmt.Errorf(
			"%s steht weder in der Gesellschafterliste noch in der Geschäftsführung der Unternehmensdaten. "+
				"Eine Auslage erstattet Buchfink Gesellschaftern und Geschäftsführern; die Auslage eines "+
				"Arbeitnehmers läuft über die Lohnabrechnung", creditor.Name)
	}
	cid := creditor.ID
	line.ContactID = &cid
	line.Text = "Auslage von " + creditor.LegalName()
	return nil
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
	case domain.TaxTreatmentReverseChargeEU:
		// § 13b Abs. 1 UStG setzt einen im übrigen Gemeinschaftsgebiet
		// ansässigen Unternehmer voraus; die USt-IdNr. belegt die
		// Unternehmereigenschaft dort.
		if !contact.IsEUCounterparty() {
			return fmt.Errorf(
				"§ 13b Abs. 1 UStG gilt nur für die sonstige Leistung eines Unternehmers aus einem anderen EU-Staat, "+
					"%s ist in %q erfasst. Für einen Lieferanten aus dem Drittland gilt § 13b Abs. 2 UStG",
				contact.Name, contact.CountryCode)
		}
		if contact.VatID == "" {
			return fmt.Errorf(
				"für § 13b UStG braucht der ausländische Lieferant %s eine USt-IdNr., die seine Unternehmereigenschaft belegt",
				contact.Name)
		}
	case domain.TaxTreatmentReverseCharge:
		// Bei einer inländischen Bauleistung (Abs. 2 Nr. 4) hängt die
		// Steuerschuld am Leistungsempfänger (Abs. 5); eine fehlende USt-IdNr.
		// des inländischen Lieferanten blockiert den Fall deshalb nicht.
		if contact.IsEUCounterparty() && contact.VatID == "" {
			return fmt.Errorf(
				"für § 13b UStG braucht der ausländische Lieferant %s eine USt-IdNr., die seine Unternehmereigenschaft belegt",
				contact.Name)
		}
	}
	return nil
}

// warningReverseChargeEU ist der Hinweis auf § 13b Abs. 1 bei einem
// EU-Lieferanten, für den Abs. 2 gewählt wurde.
const warningReverseChargeEU = "reverse_charge_eu_supplier"

// reverseChargeParagraphWarning weist darauf hin, dass für die sonstige
// Leistung eines EU-Unternehmers § 13b Abs. 1 gilt. Abs. 2 bleibt für dessen
// Werklieferung richtig, deshalb sperrt der Hinweis nicht.
func reverseChargeParagraphWarning(treatment domain.TaxTreatment, contact *domain.Contact) *PostingWarning {
	if treatment != domain.TaxTreatmentReverseCharge || contact == nil || !contact.IsEUCounterparty() {
		return nil
	}
	return &PostingWarning{
		Code:     warningReverseChargeEU,
		Severity: "warning",
		Title:    "Lieferant aus einem anderen EU-Staat",
		Detail: fmt.Sprintf(
			"%s sitzt in %s. Ist die Leistung eine sonstige Leistung nach § 3a Abs. 2 UStG (etwa Software, "+
				"Werbung, Beratung), gilt § 13b Abs. 1 UStG: Kennzahl 46/47, Steuer im Zeitraum der Leistung. "+
				"§ 13b Abs. 2 (Kennzahl 84/85) passt nur für eine Werklieferung oder eine Leistung, deren Ort "+
				"sich nicht nach § 3a Abs. 2 UStG richtet.",
			contact.Name, contact.CountryCode),
	}
}

// PostAdvanceSettlement bucht die Vereinnahmung einer Anzahlung.
//
// SOLL Zahlungsmittel an HABEN „Erhaltene, versteuerte Anzahlungen" (netto) und
// HABEN Umsatzsteuer. Hier — und nicht beim Ausstellen der Abschlagsrechnung —
// entsteht die Steuer: § 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG lässt sie mit
// Ablauf des Voranmeldungszeitraums entstehen, in dem das Teilentgelt
// vereinnahmt worden ist, und das gilt auch bei Sollversteuerung.
//
// Die Buchung hat die Quelle „advance". Daran erkennt die Periodenzuordnung
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
	receiptID *uint,
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
		ReceiptID:          receiptID,
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
// Die Steuerzeile der Auflösung hat denselben Steuerschlüssel wie die der
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
				"Dieser Fall ist in Buchfink nicht abgebildet — die Schlussrechnung ist in Euro auszustellen",
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
		ReceiptID:          inv.ReceiptID,
		TaxTreatment:       inv.TaxTreatment,
		ContactID:          &contact.ID,
		Currency:           inv.Currency,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	})
}
