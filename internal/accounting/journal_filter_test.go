package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Journalfilter grenzt ein und summiert die gefilterte Menge (PRF-01 K3).
func TestJournalFilterNarrowsAndSums(t *testing.T) {
	withReceipt := uint(4)
	entries := []domain.JournalEntry{
		{
			ID: 1, EntryNumber: "2026-000001", BookingDate: "2026-01-15",
			Description: "Büromaterial", Actor: "anna", ReceiptID: &withReceipt,
			Lines: []domain.JournalLine{
				{Position: 1, Side: domain.SideDebit, Account: "6815", Amount: 10_000},
				{Position: 2, Side: domain.SideDebit, Account: domain.AccountVorsteuer19,
					Amount: 1_900, TaxKey: "VST19", TaxBase: 10_000},
				{Position: 3, Side: domain.SideCredit, Account: domain.AccountBank, Amount: 11_900},
			},
		},
		{
			ID: 2, EntryNumber: "2026-000002", BookingDate: "2026-02-10",
			Description: "Miete", Actor: "bernd", Channel: "agent:claude-desktop",
			Lines: []domain.JournalLine{
				{Position: 1, Side: domain.SideDebit, Account: "6310", Amount: 100_000},
				{Position: 2, Side: domain.SideCredit, Account: domain.AccountBank, Amount: 100_000},
			},
		},
		{
			ID: 3, EntryNumber: "2026-000003", BookingDate: "2026-03-01",
			Description: "Büromaterial", Actor: "anna",
			Lines: []domain.JournalLine{
				{Position: 1, Side: domain.SideDebit, Account: "6815", Amount: 5_000},
				{Position: 2, Side: domain.SideCredit, Account: domain.AccountGeldtransit, Amount: 5_000},
			},
		},
	}

	// Ohne Einschränkung: jede Zeile, und Soll gleich Haben.
	all := FilterJournal(entries, JournalFilter{}, nil)
	if all.RowCount != 7 || all.EntryCount != 3 {
		t.Errorf("%d Zeilen aus %d Buchungen — erwartet 7 aus 3", all.RowCount, all.EntryCount)
	}
	if all.TotalDebit != all.TotalCredit || all.Balance != 0 {
		t.Errorf("die ungefilterte Menge muss ausgeglichen sein: %s / %s",
			all.TotalDebit, all.TotalCredit)
	}

	// Nach Konto: beide Buchungen auf 6815, zusammen 150,00 €.
	byAccount := FilterJournal(entries, JournalFilter{Account: "6815"}, nil)
	if byAccount.RowCount != 2 || byAccount.TotalDebit != 15_000 {
		t.Errorf("Konto 6815: %d Zeilen, Summe %s € — erwartet 2 und 150,00 €",
			byAccount.RowCount, byAccount.TotalDebit)
	}

	// Nach Bearbeiter.
	byActor := FilterJournal(entries, JournalFilter{Actor: "bernd"}, nil)
	if byActor.EntryCount != 1 || byActor.TotalDebit != 100_000 {
		t.Errorf("Bearbeiter bernd: %d Buchungen, Summe %s €", byActor.EntryCount, byActor.TotalDebit)
	}

	// Nach Erfassungsweg.
	byChannel := FilterJournal(entries, JournalFilter{Channel: "agent:claude-desktop"}, nil)
	if byChannel.EntryCount != 1 || byChannel.Rows[0].Channel != "agent:claude-desktop" {
		t.Errorf("Erfassungsweg Agent: %d Buchungen, Zeilen %+v", byChannel.EntryCount, byChannel.Rows)
	}

	// Nach Betrag.
	from := domain.Cents(50_000)
	byAmount := FilterJournal(entries, JournalFilter{AmountFrom: &from}, nil)
	if byAmount.RowCount != 2 {
		t.Errorf("ab 500,00 €: %d Zeilen — erwartet die beiden Zeilen der Miete",
			byAmount.RowCount)
	}

	// Nach Steuerschlüssel.
	byTaxKey := FilterJournal(entries, JournalFilter{TaxKey: "VST19"}, nil)
	if byTaxKey.RowCount != 1 || byTaxKey.TotalDebit != 1_900 {
		t.Errorf("VST19: %d Zeilen, Summe %s €", byTaxKey.RowCount, byTaxKey.TotalDebit)
	}

	// Beleg vorhanden ja/nein.
	no := false
	withoutReceipt := FilterJournal(entries, JournalFilter{HasReceipt: &no}, nil)
	if withoutReceipt.EntryCount != 2 {
		t.Errorf("%d Buchungen ohne Beleg — erwartet zwei", withoutReceipt.EntryCount)
	}
	yes := true
	withReceiptOnly := FilterJournal(entries, JournalFilter{HasReceipt: &yes}, nil)
	if withReceiptOnly.EntryCount != 1 {
		t.Errorf("%d Buchungen mit Beleg — erwartet eine", withReceiptOnly.EntryCount)
	}

	// Gegenkonto: die Buchung muss das Konto enthalten, die Zeile darauf steht
	// nicht im Ergebnis.
	counter := FilterJournal(entries,
		JournalFilter{Account: "6815", CounterAccount: domain.AccountGeldtransit}, nil)
	if counter.RowCount != 1 || counter.Rows[0].EntryNumber != "2026-000003" {
		t.Errorf("Gegenkonto Geldtransit: %+v", counter.Rows)
	}

	// Zeitraum und Textsuche.
	period := FilterJournal(entries, JournalFilter{From: "2026-02-01", To: "2026-02-28"}, nil)
	if period.EntryCount != 1 {
		t.Errorf("Februar: %d Buchungen — erwartet eine", period.EntryCount)
	}
	text := FilterJournal(entries, JournalFilter{Text: "büromaterial"}, nil)
	if text.EntryCount != 2 {
		t.Errorf("Textsuche: %d Buchungen — erwartet zwei", text.EntryCount)
	}
	if !(JournalFilter{}).IsEmpty() {
		t.Error("der leere Filter schränkt nichts ein")
	}
}
