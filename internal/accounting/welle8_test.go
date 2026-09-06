package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Steuersatz folgt dem Leistungsdatum (UNV-03 K2).
//
// Die Senkung des Zweiten Corona-Steuerhilfegesetzes galt für Umsätze vom
// 1. Juli bis zum 31. Dezember 2020. Eine im Jahr 2026 nacherfasste Rechnung
// über eine Leistung aus dieser Zeit trägt 16 %, und ein Programm, das dort
// 19 % vorschlägt, führt auf eine Buchung, die weder zur Rechnung noch zur
// Voranmeldung passt.
func TestTaxRateFollowsTheServiceDate(t *testing.T) {
	cases := []struct {
		date              string
		standard, reduced domain.TaxRate
	}{
		{"2019-12-31", 1900, 700},
		{"2020-06-30", 1900, 700},
		{"2020-07-01", 1600, 500},
		{"2020-12-31", 1600, 500},
		{"2021-01-01", 1900, 700},
		{"2026-03-01", 1900, 700},
	}
	for _, c := range cases {
		standard, err := TaxRateFor(c.date, false)
		if err != nil {
			t.Fatalf("%s: %v", c.date, err)
		}
		if standard != c.standard {
			t.Errorf("%s: Regelsatz %s, erwartet %s", c.date, standard.Label(), c.standard.Label())
		}
		reduced, err := TaxRateFor(c.date, true)
		if err != nil {
			t.Fatalf("%s: %v", c.date, err)
		}
		if reduced != c.reduced {
			t.Errorf("%s: ermäßigter Satz %s, erwartet %s",
				c.date, reduced.Label(), c.reduced.Label())
		}
	}

	// Vor der geführten Zeit wird nicht geraten.
	if _, err := TaxRateFor("2006-12-31", false); err == nil {
		t.Error("für einen Tag ohne hinterlegten Satz darf kein Satz herauskommen")
	}
	if _, err := TaxRateFor("", false); err == nil {
		t.Error("ohne Datum lässt sich kein Satz bestimmen")
	}
}

// Die sonstigen Leistungen bleiben vierteljährlich, auch wenn die Lieferungen
// monatlich gemeldet werden (§ 18a Abs. 1 Satz 3 UStG, UST-04 K3).
func TestZMServicesStayQuarterlyWhileGoodsAreMonthly(t *testing.T) {
	movements := []ZMMovement{
		// Lieferungen über der Grenze: sie zwingen zur monatlichen Meldung.
		{EntryID: 1, Date: "2026-01-15", Kind: domain.ZMKindSupply, ContactID: 1, Amount: 6_000_000},
		// Sonstige Leistungen in jedem Monat des ersten Quartals.
		{EntryID: 2, Date: "2026-01-20", Kind: domain.ZMKindService, ContactID: 1, Amount: 100_000},
		{EntryID: 3, Date: "2026-02-20", Kind: domain.ZMKindService, ContactID: 1, Amount: 200_000},
		{EntryID: 4, Date: "2026-03-20", Kind: domain.ZMKindService, ContactID: 1, Amount: 300_000},
	}
	periods := ZMPeriodsOfYear(2026, movements)
	if len(periods) == 0 || periods[0].Type != domain.VatPeriodMonth {
		t.Fatalf("über der Grenze wird monatlich gemeldet, erhalten %v", periods)
	}

	recipient := func(uint) ZMRecipient {
		return ZMRecipient{Name: "Client SARL", CountryCode: "FR", VatID: "FR12345678901", IsEU: true}
	}
	byPeriod := map[string][]domain.ZMLine{}
	for _, p := range periods[:3] {
		lines, _ := ZMLines(p, movements, recipient)
		byPeriod[p.Key] = lines
	}

	serviceAmount := func(lines []domain.ZMLine) domain.Cents {
		var total domain.Cents
		for _, l := range lines {
			if l.Kind == domain.ZMKindService {
				total += l.Amount
			}
		}
		return total
	}
	if got := serviceAmount(byPeriod["2026-01"]); got != 0 {
		t.Errorf("Januar meldet %s € sonstige Leistungen — sie gehören in den letzten "+
			"Monat des Quartals (§ 18a Abs. 1 Satz 3 UStG)", got)
	}
	if got := serviceAmount(byPeriod["2026-02"]); got != 0 {
		t.Errorf("Februar meldet %s € sonstige Leistungen — erwartet keine", got)
	}
	if got := serviceAmount(byPeriod["2026-03"]); got != 600_000 {
		t.Errorf("März meldet %s € sonstige Leistungen — erwartet die 6.000,00 € des "+
			"ganzen Quartals", got)
	}
	// Die Lieferung bleibt in ihrem Monat: nur die sonstigen Leistungen wandern.
	var supply domain.Cents
	for _, l := range byPeriod["2026-01"] {
		if l.Kind == domain.ZMKindSupply {
			supply += l.Amount
		}
	}
	if supply != 6_000_000 {
		t.Errorf("Januar meldet %s € Lieferungen — erwartet 60.000,00 €", supply)
	}
}

// Bei vierteljährlicher Meldung ändert sich nichts: dort stehen ohnehin beide
// Arten im selben Zeitraum.
func TestZMServicesUnchangedOnQuarterlyPeriods(t *testing.T) {
	movements := []ZMMovement{
		{EntryID: 1, Date: "2026-01-20", Kind: domain.ZMKindService, ContactID: 1, Amount: 100_000},
	}
	recipient := func(uint) ZMRecipient {
		return ZMRecipient{Name: "Client SARL", CountryCode: "FR", VatID: "FR12345678901", IsEU: true}
	}
	lines, _ := ZMLines(quarterPeriod(2026, 1), movements, recipient)
	if len(lines) != 1 || lines[0].Amount != 100_000 {
		t.Errorf("das Quartal meldet die sonstige Leistung nicht: %+v", lines)
	}
}

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
			Description: "Miete", Actor: "bernd",
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
				{Position: 2, Side: domain.SideCredit, Account: domain.AccountKasse, Amount: 5_000},
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
		JournalFilter{Account: "6815", CounterAccount: domain.AccountKasse}, nil)
	if counter.RowCount != 1 || counter.Rows[0].EntryNumber != "2026-000003" {
		t.Errorf("Gegenkonto Kasse: %+v", counter.Rows)
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

// Die Fristentabelle kommt aus der Ressource und rechnet die Verkürzung des
// Vierten Bürokratieentlastungsgesetzes richtig (ARC-01 K4).
func TestRetentionRulesComeFromTheResource(t *testing.T) {
	rules := LoadedRetentionRules()
	if rules.Version == "" || rules.Source == "" {
		t.Fatal("die Fristentabelle nennt weder ihre Fassung noch ihre Quelle")
	}
	// Der Gültigkeitsbeginn gehört zur Tabelle wie zu afa_rules.json: ohne ihn
	// sagt die Anzeige „Quelle: Gesetzesstand" nicht, welcher Stand gilt.
	if rules.ValidFrom != "2025-01-01" {
		t.Errorf("Gültigkeitsbeginn %q — erwartet den 1.1.2025, den Tag der Verkürzung",
			rules.ValidFrom)
	}
	if len(rules.Classes) != 3 {
		t.Fatalf("%d Klassen — erwartet Bücher, Belege und Handelsbriefe", len(rules.Classes))
	}

	// Ein Beleg aus 2013: die alte Zehnjahresfrist war am 1.1.2025 abgelaufen,
	// es bleibt bei ihr.
	old := RetentionFor(domain.RetentionKindReceiptInvoice, 2013)
	if old.Years != 10 || old.RetentionEnd != "2023-12-31" {
		t.Errorf("Beleg aus 2013: %d Jahre bis %s — erwartet 10 Jahre bis 2023-12-31",
			old.Years, old.RetentionEnd)
	}
	// Ein Beleg aus 2016: die alte Frist lief am 1.1.2025 noch, die kürzere
	// greift.
	running := RetentionFor(domain.RetentionKindReceiptInvoice, 2016)
	if running.Years != 8 {
		t.Errorf("Beleg aus 2016: %d Jahre — die Verkürzung wirkt auf laufende Fristen",
			running.Years)
	}
	books := RetentionFor(domain.RetentionKindJournal, 2026)
	if books.Years != 10 || books.Class != domain.RetentionClassBooks {
		t.Errorf("Handelsbücher: %d Jahre, Klasse %q", books.Years, books.Class)
	}
	letters := RetentionFor(domain.RetentionKindReceiptLetter, 2026)
	if letters.Years != 6 {
		t.Errorf("Handelsbrief: %d Jahre — erwartet sechs", letters.Years)
	}
	if RetentionForClass(domain.RetentionClassBooks, 2026).RetentionEnd != "2036-12-31" {
		t.Error("die ausdrücklich gewählte Klasse rechnet anders als die abgeleitete")
	}
}
