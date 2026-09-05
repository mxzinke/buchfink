package service

import (
	"context"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Die Ausbuchung eines uneinbringlichen Postens.
//
// Sie läuft nicht über Settle, und das ist kein Zufall: Settle ist der
// Zahlungsweg und verlangt ein Zahlungsmittel und einen Betrag, der bewegt
// wurde. Bei einer Ausbuchung fließt nichts. Sie über ein Zahlungsmittelkonto
// mit dem Betrag null zu schleusen hieße, eine Zahlung zu behaupten, die es
// nicht gab.
//
// Zwei Dinge geschehen: der Forderungsverlust wird als Aufwand erfasst, und die
// Umsatzsteuer wird berichtigt. Die Berichtigung ist keine Kür — § 17 Abs. 2
// Nr. 1 i. V. m. Abs. 1 UStG verlangt sie, sobald das Entgelt uneinbringlich
// geworden ist, und zwar im Zeitraum der Uneinbringlichkeit (Abs. 1 Satz 8).

// WriteOffRequest bucht einen offenen Posten als uneinbringlich aus.
type WriteOffRequest struct {
	OpenItemEntryID uint `json:"openItemEntryId"`
	// Amount ist der auszubuchende Bruttobetrag. Null heißt: der ganze offene
	// Betrag.
	Amount domain.Cents `json:"amount"`
	Date   string       `json:"date"`
	// Reason ist Pflicht. Eine Ausbuchung ohne Begründung ist von einer
	// vergessenen Forderung nicht zu unterscheiden, und die Betriebsprüfung
	// fragt nach genau dieser Unterscheidung.
	Reason string `json:"reason"`
}

// WriteOffOpenItem bucht eine uneinbringliche Forderung aus.
func (s *PaymentService) WriteOffOpenItem(ctx context.Context, req WriteOffRequest) (*domain.JournalEntry, error) {
	if req.Reason == "" {
		return nil, fmt.Errorf(
			"zu jeder Ausbuchung gehört eine Begründung: woran die Forderung gescheitert ist, entscheidet, " +
				"ob sie uneinbringlich im Sinne des § 17 Abs. 2 Nr. 1 UStG ist")
	}
	if req.Date == "" {
		req.Date = time.Now().Format("2006-01-02")
	}

	openItems, err := s.OpenItems(ctx)
	if err != nil {
		return nil, err
	}
	var item *domain.OpenItem
	for i := range openItems {
		if openItems[i].EntryID == req.OpenItemEntryID {
			item = &openItems[i]
			break
		}
	}
	if item == nil {
		return nil, fmt.Errorf("die Buchung %d hat keinen offenen Posten", req.OpenItemEntryID)
	}
	if item.ContactType != domain.ContactTypeCustomer {
		return nil, fmt.Errorf(
			"%s ist eine Verbindlichkeit. Ihr Wegfall ist ein Erlass oder eine Verjährung und wird als "+
				"Ertrag gebucht, nicht als Forderungsverlust", item.DocumentNumber)
	}
	amount := req.Amount
	if amount == 0 {
		amount = item.OpenAmount
	}
	if amount <= 0 || amount > item.OpenAmount {
		return nil, fmt.Errorf(
			"%s ist mit %s € offen, ausgebucht werden sollen aber %s €",
			item.DocumentNumber, item.OpenAmount, amount)
	}
	if item.TaxTreatment == "" {
		return nil, fmt.Errorf(
			"der Steuerfall von %s lässt sich nicht bestimmen; ohne ihn wäre die Steuerkorrektur nach "+
				"§ 17 Abs. 2 Nr. 1 UStG nicht sauber zu buchen", item.DocumentNumber)
	}

	// Nur beim steuerpflichtigen Inlandsumsatz steckt die Steuer im offenen
	// Betrag; bei einer steuerfreien Lieferung ist der ganze Betrag Entgelt.
	rate := domain.TaxRateNone
	if item.TaxTreatment == domain.TaxTreatmentDomestic {
		rate = item.TaxRate
	}
	net := rate.NetFromGross(amount)

	account, err := accounting.WriteOffAccountFor(rate)
	if err != nil {
		return nil, err
	}

	lines := []domain.JournalLine{{
		Side: domain.SideDebit, Account: account, Amount: net,
		Text: "Forderungsverlust " + item.DocumentNumber,
	}}

	// Die Steuerkorrektur kommt aus der Steuerautomatik und nicht aus einer
	// zweiten Kontentabelle: so folgt sie dem Steuerfall des Belegs und landet
	// bei § 13b nicht auf dem falschen Konto.
	legs, err := s.journalSvc.TaxResolver().Resolve(
		domain.DirectionOutgoing, item.TaxTreatment, item.TaxRate, net)
	if err != nil {
		return nil, fmt.Errorf("die Steuerkorrektur der Ausbuchung ließ sich nicht auflösen: %w", err)
	}
	// Die Steuerkorrektur trägt die Rundungsdifferenz.
	//
	// Ausgeglichen wird das Personenkonto mit dem tatsächlich offenen Betrag;
	// Netto und Steuer müssen ihn deshalb genau treffen. Aus dem gerundeten
	// Netto gerechnet tun sie das nicht: eine Teilausbuchung über drei Cent
	// ergäbe 0,03 € Aufwand und 0,01 € Steuer — vier Cent im Soll gegen drei im
	// Haben, und die Buchung ginge nicht durch.
	legs = absorbRounding(legs, amount-net)
	for _, leg := range legs {
		if leg.Amount == 0 {
			continue
		}
		line := taxLegLine(leg)
		// Gegenseite der ursprünglichen Steuerzeile: was die Rechnung im Haben
		// geschuldet hat, wird im Soll zurückgenommen. Der Steuerschlüssel
		// bleibt derselbe — die Voranmeldung muss die Minderung sehen, und ein
		// eigener Schlüssel fiele aus ihrer Auswertung heraus.
		line.Side = leg.Side.Opposite()
		line.Text = "Steuerkorrektur Forderungsverlust (§ 17 Abs. 2 Nr. 1 UStG)"
		lines = append(lines, line)
	}

	contactID := item.ContactID
	lines = append(lines, domain.JournalLine{
		Side:      item.SettleSide(),
		Account:   item.LedgerAccount,
		ContactID: &contactID,
		Amount:    amount,
		Text:      item.DocumentNumber,
	})

	entry, err := s.journalSvc.Post(ctx, &domain.JournalEntry{
		BookingDate:     req.Date,
		DocumentDate:    req.Date,
		ServiceDateFrom: req.Date,
		ServiceDateTo:   req.Date,
		Description: fmt.Sprintf("Ausbuchung %s (%s): %s",
			item.DocumentNumber, item.ContactName, req.Reason),
		// Quelle „Zahlung": die Buchung schließt einen offenen Posten und
		// begründet keinen neuen. Ohne das erschiene sie selbst wieder in der
		// Liste der offenen Posten.
		Source:             domain.EntrySourcePayment,
		DocumentNumber:     item.DocumentNumber,
		TaxTreatment:       item.TaxTreatment,
		ContactID:          &contactID,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	})
	if err != nil {
		return nil, err
	}

	if err := s.allocationRepo.Create(ctx, []domain.PaymentAllocation{{
		OpenItemEntryID:  item.EntryID,
		PaymentEntryID:   entry.ID,
		ContactID:        item.ContactID,
		SettledAmount:    amount,
		CashAmount:       0,
		DifferenceKind:   domain.DifferenceWriteoff,
		DifferenceAmount: amount,
	}}); err != nil {
		return nil, fmt.Errorf("die Ausbuchung wurde gebucht, ließ sich aber dem Posten nicht zuordnen: %w", err)
	}
	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, domain.AuditActionUpdate, "OPEN_ITEM",
			fmt.Sprintf("%d", item.EntryID), fmt.Sprintf(
				"Offener Posten %s (%s) über %s € als uneinbringlich ausgebucht (%s): %s",
				item.DocumentNumber, item.ContactName, amount, entry.EntryNumber, req.Reason))
	}
	return entry, nil
}

// absorbRounding setzt den Steuerbetrag auf die Differenz zwischen Brutto und
// Netto.
//
// Nur wo es genau ein Steuerbein gibt — der steuerpflichtige Inlandsumsatz.
// Beim Reverse-Charge stehen sich zwei Beine gegenüber, die sich aufheben; dort
// gibt es keine Differenz zu verteilen, und eine einseitige Anpassung machte
// aus der ausgeglichenen Buchung eine unausgeglichene.
func absorbRounding(legs []domain.TaxLeg, wanted domain.Cents) []domain.TaxLeg {
	index, count := -1, 0
	for i := range legs {
		if legs[i].Amount != 0 {
			index, count = i, count+1
		}
	}
	if count != 1 {
		return legs
	}
	legs[index].Amount = wanted
	return legs
}
