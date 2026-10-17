package accounting

import (
	"sort"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Mindest-Istversteuerung im gewöhnlichen Rechnungs- und Zahlungsweg.
//
// Wird das Entgelt vor der Leistung vereinnahmt, entsteht die Steuer insoweit
// mit Ablauf des Voranmeldungszeitraums der Vereinnahmung (§ 13 Abs. 1 Nr. 1
// Buchst. a Satz 4 UStG, für § 13b Abs. 4 Satz 2 UStG). Spiegelbildlich ist die
// Vorsteuer auf eine Zahlung vor der Leistung abziehbar, sobald Rechnung und
// Zahlung vorliegen (§ 15 Abs. 1 Satz 1 Nr. 1 Satz 3, Nr. 4 Satz 2 UStG). Das
// Jahresabo, im Januar berechnet und bezahlt, gehört damit in den Januar und
// nicht in den Dezember, in dem die Leistung endet.
//
// Der Anzahlungsweg (Quelle „advance") bucht die Vereinnahmung selbst; hier geht
// es um die Rechnung, die vor dem Leistungsende bezahlt wird.

// vatPayment ist eine Zahlung, die einen offenen Posten ganz oder teilweise
// ausgleicht.
type vatPayment struct {
	date string
	// amount ist der ausgeglichene Betrag des Postens; ein Skonto zählt mit,
	// weil seine Steuerkorrektur am Zahlungstag gebucht ist.
	amount domain.Cents
	// entryID ist die Zahlungsbuchung. Sie steht im Drill-down der Kennzahl,
	// weil erst sie die Steuer in diesen Zeitraum gezogen hat.
	entryID uint
}

// prepayments sind die Zahlungen je Buchung mit offenem Posten.
type prepayments struct {
	gross    map[uint]domain.Cents
	payments map[uint][]vatPayment
	// writeOffs sind die Buchungen, mit denen eine Forderung als uneinbringlich
	// ausgebucht wurde (§ 17 Abs. 2 Nr. 1 UStG).
	writeOffs map[uint]bool
	// originals ordnet einer Generalumkehr ihre Ursprungsbuchung zu.
	originals map[uint]uint
}

func indexPrepayments(entries []domain.JournalEntry, allocations []domain.PaymentAllocation) prepayments {
	out := prepayments{
		gross:     map[uint]domain.Cents{},
		payments:  map[uint][]vatPayment{},
		writeOffs: map[uint]bool{},
		originals: map[uint]uint{},
	}
	byID := make(map[uint]*domain.JournalEntry, len(entries))
	reversed := map[uint]bool{}
	for i := range entries {
		e := &entries[i]
		byID[e.ID] = e
		if e.Kind == domain.EntryKindReversal && e.ReversalOfID != nil {
			reversed[*e.ReversalOfID] = true
			out.originals[e.ID] = *e.ReversalOfID
		}
	}

	for _, a := range allocations {
		if a.DifferenceKind == domain.DifferenceWriteoff {
			out.writeOffs[a.PaymentEntryID] = true
			continue
		}
		payment := byID[a.PaymentEntryID]
		if payment == nil || reversed[payment.ID] || a.SettledAmount <= 0 {
			continue
		}
		out.payments[a.OpenItemEntryID] = append(out.payments[a.OpenItemEntryID], vatPayment{
			date: firstNonEmpty(payment.BookingDate, payment.DocumentDate), amount: a.SettledAmount,
			entryID: payment.ID,
		})
	}

	for i := range entries {
		e := &entries[i]
		if !prepaymentCandidate(e) {
			continue
		}
		var ledger, settled domain.Cents
		hasLedger := false
		for _, l := range e.Lines {
			switch {
			case domain.IsLedgerAccount(l.Account):
				if !hasLedger {
					ledger, hasLedger = l.Amount.Abs(), true
				}
			case domain.SettlesAtBooking(l):
				settled += l.Amount.Abs()
			}
		}
		switch {
		case hasLedger:
			out.gross[e.ID] = ledger
		case settled != 0:
			// Ohne Personenkonto ist der Beleg mit der Buchung bezahlt: über die
			// Bank, die Firmenkreditkarte oder als Auslage.
			out.gross[e.ID] = settled
			out.payments[e.ID] = append(out.payments[e.ID], vatPayment{date: e.BookingDate, amount: settled})
		}
	}
	for id := range out.payments {
		sort.SliceStable(out.payments[id], func(i, j int) bool {
			return out.payments[id][i].date < out.payments[id][j].date
		})
	}
	return out
}

// prepaymentCandidate meldet, ob eine Buchung einen Umsatz trägt, der vor der
// Leistung bezahlt sein kann. Zahlungen, Anzahlungen, Vorträge und
// Abschlussbuchungen tragen keinen.
func prepaymentCandidate(e *domain.JournalEntry) bool {
	if e.Kind == domain.EntryKindReversal {
		return false
	}
	switch e.Source {
	case domain.EntrySourcePayment, domain.EntrySourceAdvance, domain.EntrySourceOpening,
		domain.EntrySourceClosing, domain.EntrySourceDepreciation:
		return false
	}
	return true
}

// paymentsOf liefert Bruttobetrag und Zahlungen, nach denen die Zeilen einer
// Buchung verteilt werden. Die Generalumkehr eines Belegs folgt den Zahlungen
// ihrer Ursprungsbuchung, damit sie ihn in denselben Zeiträumen aufhebt. Die
// Rücknahme einer Ausgangsrechnung wirkt dagegen im Zeitraum der Berichtigung
// und wird nicht verteilt.
func (p prepayments) paymentsOf(e *domain.JournalEntry) (domain.Cents, []vatPayment) {
	id := e.ID
	if e.Kind == domain.EntryKindReversal {
		if e.Source == domain.EntrySourceInvoice {
			return 0, nil
		}
		original, ok := p.originals[e.ID]
		if !ok {
			return 0, nil
		}
		id = original
	}
	gross := p.gross[id]
	if gross <= 0 {
		return 0, nil
	}
	return gross, p.payments[id]
}

// isWriteOff meldet, ob eine Buchung eine Forderung ausbucht oder eine solche
// Ausbuchung zurücknimmt.
func (p prepayments) isWriteOff(e *domain.JournalEntry) bool {
	if p.writeOffs[e.ID] {
		return true
	}
	if original, ok := p.originals[e.ID]; ok {
		return p.writeOffs[original]
	}
	return false
}

// arising ist ein Tag, bis zu dem die Steuer für einen Teil der Bewegung
// entstanden ist: covered ist der kumulierte Anteil in Einheiten der Summe,
// über die verteilt wird. trigger ist die Buchung, die den Teil vorzieht, 0
// beim Jahresende des § 13b Abs. 3 UStG.
type arising struct {
	date    string
	covered domain.Cents
	trigger uint
}

// paymentArisings sind die Zahlungen vor der Leistung als Tage, an denen die
// Steuer insoweit entsteht oder abziehbar wird. earlier liefert zu einem
// Zahlungstag dieses Datum; nil heißt, eine Zahlung zieht nichts vor.
func paymentArisings(gross domain.Cents, payments []vatPayment, earlier func(string) string) []arising {
	if gross <= 0 || earlier == nil {
		return nil
	}
	out := make([]arising, 0, len(payments))
	var covered domain.Cents
	for _, pay := range payments {
		if covered >= gross {
			break
		}
		covered += min(pay.amount, gross-covered)
		if date := earlier(pay.date); date != "" {
			out = append(out, arising{date: date, covered: covered, trigger: pay.entryID})
		}
	}
	return out
}

// splitByArising verteilt eine Bewegung auf die Tage, an denen die Steuer für
// einen Teil früher entsteht als am regulären Datum; was kein Tag vorzieht,
// bleibt dort. Aus mehreren Quellen (Zahlungen, Jahresenden) gilt je Tag der
// größte bis dahin erreichte Anteil.
//
// Gerechnet wird über die aufgelaufenen Anteile, damit sich die Teile ohne
// Rundungsrest zur ursprünglichen Bewegung ergänzen.
func splitByArising(m vatMovement, total domain.Cents, events []arising) []vatMovement {
	if total <= 0 || len(events) == 0 {
		return []vatMovement{m}
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].date < events[j].date })
	out := make([]vatMovement, 0, len(events)+1)
	var covered, baseDone, taxDone domain.Cents
	for _, ev := range events {
		if ev.date >= m.date {
			break
		}
		if ev.covered <= covered {
			continue
		}
		covered = min(ev.covered, total)
		base := domain.MulRound(m.base, int64(covered), int64(total))
		tax := domain.MulRound(m.tax, int64(covered), int64(total))
		part := m
		part.date, part.trigger = ev.date, ev.trigger
		part.base, part.tax = base-baseDone, tax-taxDone
		baseDone, taxDone = base, tax
		out = append(out, part)
	}
	rest := m
	rest.base, rest.tax = m.base-baseDone, m.tax-taxDone
	if rest.base != 0 || rest.tax != 0 || len(out) == 0 {
		out = append(out, rest)
	}
	return out
}

// longTermYearEnds wendet § 13b Abs. 3 UStG an: für eine sonstige Leistung,
// die dauerhaft über mehr als ein Jahr erbracht wird, entsteht die Steuer nach
// § 13b Abs. 1 und Abs. 2 Nr. 1 UStG spätestens mit Ablauf jedes
// Kalenderjahres, in dem sie erbracht wird. Der bis zum Jahresende erbrachte
// Teil ist zeitanteilig nach Tagen bemessen. Die Summe ist total; ist sie
// nicht bekannt, sind es die Tage der Leistung.
func longTermYearEnds(entry *domain.JournalEntry, total domain.Cents) (domain.Cents, []arising) {
	from, errFrom := time.Parse("2006-01-02", entry.ServiceDateFrom)
	to, errTo := time.Parse("2006-01-02", entry.ServiceDateTo)
	if errFrom != nil || errTo != nil || to.Before(from.AddDate(1, 0, 0)) {
		return total, nil
	}
	days := int64(to.Sub(from).Hours()/24) + 1
	if total <= 0 {
		total = domain.Cents(days)
	}
	var out []arising
	for year := from.Year(); year < to.Year(); year++ {
		end := time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC)
		rendered := int64(end.Sub(from).Hours()/24) + 1
		out = append(out, arising{date: end.Format("2006-01-02"),
			covered: domain.MulRound(total, rendered, days)})
	}
	return total, out
}
