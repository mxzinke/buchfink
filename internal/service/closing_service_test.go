package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// closing wires the Jahresabschluss on the shared test environment. Der
// JournalService bekommt dabei die Geschäftsjahre und die Festschreibungen: erst
// mit ihnen kann er eine Buchung in ein festgestelltes Jahr abweisen.
func (e *testEnv) closing(t *testing.T) *ClosingService {
	t.Helper()
	e.journal.SetFiscalYearRepo(repository.NewFiscalYearRepository(e.db))
	e.journal.SetFestschreibungRepo(repository.NewFestschreibungRepository(e.db))
	return NewClosingService(
		repository.NewFiscalYearRepository(e.db),
		e.journalRepo,
		repository.NewAccountRepository(e.db),
		e.contactRepo,
		repository.NewPaymentAllocationRepository(e.db),
		repository.NewSettingsRepository(e.db),
		repository.NewFestschreibungRepository(e.db),
		repository.NewAuditRepository(e.db),
		e.journal,
		e.fiscalYear,
	)
}

// datedEntry ist eine ausgeglichene Buchung an einem bestimmten Tag.
func datedEntry(date, debit, credit string, amount domain.Cents) *domain.JournalEntry {
	return &domain.JournalEntry{
		BookingDate:     date,
		DocumentDate:    date,
		ServiceDateFrom: date,
		ServiceDateTo:   date,
		Description:     "Testbuchung " + date,
		Source:          domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: debit, Amount: amount},
			{Side: domain.SideCredit, Account: credit, Amount: amount},
		},
	}
}

// balances liefert die Salden eines Geschäftsjahres in Soll-Richtung.
func balances(t *testing.T, e *testEnv, year int) map[string]domain.Cents {
	t.Helper()
	turnovers, err := e.journalRepo.AccountTurnovers(context.Background(), year)
	if err != nil {
		t.Fatalf("Salden des Jahres %d: %v", year, err)
	}
	out := map[string]domain.Cents{}
	for account, turnover := range turnovers {
		if b := turnover.Debit - turnover.Credit; b != 0 {
			out[account] = b
		}
	}
	return out
}

func rowFor(preview *CarryForwardPreview, account string) *CarryForwardRow {
	for i := range preview.Rows {
		if preview.Rows[i].Account == account {
			return &preview.Rows[i]
		}
	}
	return nil
}

// assertOpeningMatchesClosing prüft die Bilanzidentität über alle Bilanzkonten
// und nicht nur über die beiden, um die es im jeweiligen Fall geht: jedes Konto
// des neuen Jahres muss auf dem Schlusssaldo des alten stehen, das Ergebniskonto
// zusätzlich um das Jahresergebnis erhöht (§ 252 Abs. 1 Nr. 1 HGB). Die
// Vortragskonten der Klasse 9 bleiben außen vor — sie sind das Gegenkonto des
// Vortrags und stehen im alten Jahr naturgemäß anders als im neuen.
func assertOpeningMatchesClosing(
	t *testing.T, e *testEnv, fromYear, toYear int, netIncome domain.Cents, resultAccount string,
) {
	t.Helper()
	chart, err := e.journal.Chart(context.Background())
	if err != nil {
		t.Fatalf("Kontenplan: %v", err)
	}

	old := balances(t, e, fromYear)
	fresh := balances(t, e, toYear)
	accounts := map[string]bool{}
	for account := range old {
		accounts[account] = true
	}
	for account := range fresh {
		accounts[account] = true
	}

	checked := 0
	for account := range accounts {
		if domain.IsCarryForwardAccount(account) {
			continue
		}
		if !domain.IsLedgerAccount(account) {
			acc, ok := chart.Lookup(account)
			if !ok || acc.StatementType != "Bilanz" {
				continue
			}
		}
		want := old[account]
		// Das Jahresergebnis steht im Haben, in Soll-Richtung also negativ.
		if account == resultAccount {
			want -= netIncome
		}
		if fresh[account] != want {
			t.Errorf("Konto %s steht im Geschäftsjahr %d auf %s €, erwartet %s €",
				account, toYear, fresh[account], want)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("es wurde kein einziges Bilanzkonto geprüft")
	}
}

// openingEntry liefert die (nicht stornierte) Vortragsbuchung eines Jahres.
func openingEntry(t *testing.T, e *testEnv, year int) domain.JournalEntry {
	t.Helper()
	for _, entry := range entriesOf(t, e, year) {
		if entry.Source == domain.EntrySourceOpening && entry.Kind != domain.EntryKindReversal {
			return entry
		}
	}
	t.Fatalf("im Geschäftsjahr %d gibt es keine Vortragsbuchung", year)
	return domain.JournalEntry{}
}

// --- Saldenvortrag --------------------------------------------------------

// Die Eröffnungsbilanz des neuen Jahres muss der Schlussbilanz des alten
// entsprechen (§ 252 Abs. 1 Nr. 1 HGB), und das Jahresergebnis muss auf dem
// Gewinnvortrag landen.
func TestCarryForwardReproducesClosingBalances(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	if _, err := env.journal.Post(ctx, datedEntry("2026-02-01", domain.AccountBank, "4400", 1000000)); err != nil {
		t.Fatalf("Erlösbuchung: %v", err)
	}
	if _, err := env.journal.Post(ctx, datedEntry("2026-03-01", "6300", domain.AccountBank, 400000)); err != nil {
		t.Fatalf("Aufwandsbuchung: %v", err)
	}

	preview, err := closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortragsvorschau: %v", err)
	}
	if !preview.IsBalanced {
		t.Fatalf("die Salden gehen um %s € nicht auf", preview.BalanceDifference)
	}
	if preview.NetIncome != 600000 {
		t.Errorf("Jahresergebnis = %s €, erwartet 6.000,00", preview.NetIncome)
	}
	if preview.ResultAccount != domain.AccountGewinnvortrag {
		t.Errorf("Ergebniskonto = %s, erwartet %s", preview.ResultAccount, domain.AccountGewinnvortrag)
	}
	if preview.AlreadyCarried {
		t.Error("vor dem ersten Lauf darf nichts vorgetragen sein")
	}
	if row := rowFor(preview, domain.AccountBank); row == nil || row.ClosingBalance != 600000 {
		t.Errorf("Zeile Bank fehlt oder zeigt den falschen Schlusssaldo: %+v", row)
	}
	// Die GuV-Konten werden nicht vorgetragen.
	if row := rowFor(preview, "4400"); row != nil {
		t.Errorf("ein Erlöskonto gehört nicht in den Saldenvortrag: %+v", row)
	}

	created, err := closing.CarryForward(ctx, 2027)
	if err != nil {
		t.Fatalf("Saldenvortrag buchen: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("erwartet 1 Vortragsbuchung (nur Sachkonten), erhalten %d", len(created))
	}
	if created[0].Source != domain.EntrySourceOpening {
		t.Errorf("der Saldenvortrag muss als Eröffnungsbuchung erfasst sein, ist aber %q", created[0].Source)
	}
	if created[0].BookingDate != "2027-01-01" {
		t.Errorf("Buchungsdatum = %s, erwartet den ersten Tag des neuen Jahres", created[0].BookingDate)
	}
	assertLines(t, &created[0], []bookedLine{
		{domain.SideDebit, domain.AccountBank, 600000},
		{domain.SideCredit, domain.AccountGewinnvortrag, 600000},
	})

	// Die Bestandskonten stehen im neuen Jahr auf den Schlusssalden des alten.
	old := balances(t, env, 2026)
	fresh := balances(t, env, 2027)
	if fresh[domain.AccountBank] != old[domain.AccountBank] {
		t.Errorf("Bank im neuen Jahr = %s €, Schlusssaldo des Vorjahres = %s €",
			fresh[domain.AccountBank], old[domain.AccountBank])
	}
	if fresh[domain.AccountGewinnvortrag] != -600000 {
		t.Errorf("Gewinnvortrag = %s €, erwartet 6.000,00 im Haben", fresh[domain.AccountGewinnvortrag])
	}
	assertOpeningMatchesClosing(t, env, 2026, 2027, 600000, domain.AccountGewinnvortrag)

	// Die Summen- und Saldenliste des neuen Jahres muss aufgehen.
	env.accounting.SetFiscalYear(2027)
	susa, err := env.accounting.GetSuSaOverview(ctx)
	if err != nil {
		t.Fatalf("Summen- und Saldenliste 2027: %v", err)
	}
	if !susa.IsBalanced {
		t.Errorf("die SuSa des neuen Jahres geht um %s € nicht auf", susa.Difference)
	}
}

// Ein Verlust geht auf den Verlustvortrag, nicht auf den Gewinnvortrag.
func TestCarryForwardBooksLossToVerlustvortrag(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	if _, err := env.journal.Post(ctx, datedEntry("2026-02-01", "6300", domain.AccountBank, 250000)); err != nil {
		t.Fatalf("Aufwandsbuchung: %v", err)
	}

	preview, err := closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortragsvorschau: %v", err)
	}
	if preview.NetIncome != -250000 {
		t.Fatalf("Jahresergebnis = %s €, erwartet -2.500,00", preview.NetIncome)
	}
	if preview.ResultAccount != domain.AccountVerlustvortrag {
		t.Errorf("Ergebniskonto = %s, erwartet %s", preview.ResultAccount, domain.AccountVerlustvortrag)
	}

	created, err := closing.CarryForward(ctx, 2027)
	if err != nil {
		t.Fatalf("Saldenvortrag buchen: %v", err)
	}
	assertLines(t, &created[0], []bookedLine{
		{domain.SideCredit, domain.AccountBank, 250000},
		{domain.SideDebit, domain.AccountVerlustvortrag, 250000},
	})
}

// Auf dem Ergebniskonto steht in der Spalte „Schlusssaldo" nicht der Schlusssaldo
// des Vorjahres, sondern er zuzüglich des Jahresergebnisses. Genau dafür trägt
// die Zeile IncludesNetIncome: die Ansicht kann den Zusatz nur benennen, wenn
// die Vorschau ihn ausweist — und nur auf dieser einen Zeile.
func TestCarryForwardMarksTheRowThatCarriesTheNetIncome(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	// Ein Gewinnvortrag aus früheren Jahren steht schon auf dem Ergebniskonto …
	if _, err := env.journal.Post(ctx,
		datedEntry("2026-01-05", domain.AccountBank, domain.AccountGewinnvortrag, 500000)); err != nil {
		t.Fatalf("Vortragsbestand: %v", err)
	}
	// … und das Jahr erwirtschaftet zusätzlich einen Gewinn.
	if _, err := env.journal.Post(ctx, datedEntry("2026-02-01", domain.AccountBank, "4400", 600000)); err != nil {
		t.Fatalf("Erlösbuchung: %v", err)
	}

	preview, err := closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortragsvorschau: %v", err)
	}
	if preview.NetIncome != 600000 {
		t.Fatalf("Jahresergebnis = %s €, erwartet 6.000,00", preview.NetIncome)
	}
	if preview.ResultAccount != domain.AccountGewinnvortrag {
		t.Fatalf("Ergebniskonto = %s, erwartet %s", preview.ResultAccount, domain.AccountGewinnvortrag)
	}

	row := rowFor(preview, domain.AccountGewinnvortrag)
	if row == nil {
		t.Fatalf("die Zeile des Ergebniskontos fehlt: %+v", preview.Rows)
	}
	if !row.IncludesNetIncome {
		t.Error("die Zeile des Ergebniskontos weist das Jahresergebnis nicht aus")
	}
	// Der Schlusssaldo des Kontos im Vorjahr ist -5.000,00 (Haben); mit dem
	// Jahresergebnis von 6.000,00 im Haben trägt die Zeile -11.000,00.
	if row.ClosingBalance != -1100000 {
		t.Errorf("Zeile des Ergebniskontos = %s €, erwartet -11.000,00 (Saldo -5.000,00 zuzüglich Ergebnis)",
			row.ClosingBalance)
	}
	if balance := balances(t, env, 2026)[domain.AccountGewinnvortrag]; balance != -500000 {
		t.Errorf("Schlusssaldo des Ergebniskontos im Vorjahr = %s €, erwartet -5.000,00", balance)
	}

	for _, other := range preview.Rows {
		if other.Account != domain.AccountGewinnvortrag && other.IncludesNetIncome {
			t.Errorf("Konto %s weist das Jahresergebnis aus, obwohl es nicht das Ergebniskonto ist", other.Account)
		}
	}
}

// Ohne Jahresergebnis trägt keine Zeile den Zusatz: ein Hinweis auf ein Ergebnis
// von null wäre eine Aussage über nichts.
func TestCarryForwardMarksNoRowWithoutNetIncome(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	if _, err := env.journal.Post(ctx,
		datedEntry("2026-01-05", domain.AccountBank, domain.AccountGewinnvortrag, 500000)); err != nil {
		t.Fatalf("Vortragsbestand: %v", err)
	}

	preview, err := closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortragsvorschau: %v", err)
	}
	if preview.NetIncome != 0 {
		t.Fatalf("Jahresergebnis = %s €, erwartet 0,00", preview.NetIncome)
	}
	for _, row := range preview.Rows {
		if row.IncludesNetIncome {
			t.Errorf("Konto %s weist ein Jahresergebnis aus, obwohl es keines gibt", row.Account)
		}
	}
}

// Ein zweiter Lauf ohne Änderung bucht nichts; nach einer weiteren Buchung im
// Vorjahr zeigt der Stand eine Differenz, und der Korrekturvortrag stellt die
// Salden wieder her, ohne sie zu verdoppeln.
func TestCarryForwardIsRepeatableWithoutDoubling(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	if _, err := env.journal.Post(ctx, datedEntry("2026-02-01", domain.AccountBank, "4400", 1000000)); err != nil {
		t.Fatalf("Erlösbuchung: %v", err)
	}
	if _, err := closing.CarryForward(ctx, 2027); err != nil {
		t.Fatalf("erster Saldenvortrag: %v", err)
	}

	before := len(entriesOf(t, env, 2027))

	// Zweiter Lauf ohne Änderung: nichts zu tun, und vor allem nichts zu buchen.
	if _, err := closing.CarryForward(ctx, 2027); err == nil {
		t.Error("ein zweiter Lauf ohne Änderung darf nicht noch einmal buchen")
	} else if !strings.Contains(err.Error(), "aktuell") {
		t.Errorf("die Meldung sollte den aktuellen Stand benennen, lautet aber: %v", err)
	}
	if after := len(entriesOf(t, env, 2027)); after != before {
		t.Errorf("der zweite Lauf hat %d Buchungen erzeugt", after-before)
	}

	state, err := closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortragsstand: %v", err)
	}
	if !state.AlreadyCarried || state.NeedsCorrection {
		t.Errorf("nach dem Vortrag muss der Stand aktuell sein: vorgetragen=%v, Korrektur nötig=%v",
			state.AlreadyCarried, state.NeedsCorrection)
	}

	// Eine weitere Buchung im Vorjahr macht den Vortrag überholt.
	if _, err := env.journal.Post(ctx, datedEntry("2026-11-01", domain.AccountBank, "4400", 500000)); err != nil {
		t.Fatalf("Nachbuchung: %v", err)
	}
	state, err = closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortragsstand: %v", err)
	}
	if !state.NeedsCorrection || !state.HasDifference() {
		t.Fatal("nach einer Buchung im Vorjahr muss der Stand eine Differenz zeigen")
	}
	row := rowFor(state, domain.AccountBank)
	if row == nil || row.Difference != 500000 {
		t.Errorf("die Differenz auf dem Bankkonto = %+v, erwartet 5.000,00", row)
	}

	if _, err := closing.CarryForward(ctx, 2027); err != nil {
		t.Fatalf("Korrekturvortrag: %v", err)
	}

	old := balances(t, env, 2026)
	fresh := balances(t, env, 2027)
	if fresh[domain.AccountBank] != old[domain.AccountBank] {
		t.Errorf("nach dem Korrekturvortrag steht die Bank auf %s €, der Schlusssaldo ist %s €",
			fresh[domain.AccountBank], old[domain.AccountBank])
	}
	if fresh[domain.AccountGewinnvortrag] != -1500000 {
		t.Errorf("Gewinnvortrag = %s €, erwartet 15.000,00 im Haben (keine Verdopplung)",
			fresh[domain.AccountGewinnvortrag])
	}
	assertOpeningMatchesClosing(t, env, 2026, 2027, 1500000, domain.AccountGewinnvortrag)

	state, err = closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortragsstand: %v", err)
	}
	if state.NeedsCorrection {
		t.Error("nach dem Korrekturvortrag darf keine Differenz mehr offen sein")
	}
}

// Der Korrekturvortrag storniert per Generalumkehr in demselben Jahr — sonst
// trüge das neue Jahr den alten Vortrag weiter.
func TestCorrectionReversesInsideTheTargetYear(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	if _, err := env.journal.Post(ctx, datedEntry("2026-02-01", domain.AccountBank, "4400", 100000)); err != nil {
		t.Fatalf("Erlösbuchung: %v", err)
	}
	if _, err := closing.CarryForward(ctx, 2027); err != nil {
		t.Fatalf("erster Saldenvortrag: %v", err)
	}
	if _, err := env.journal.Post(ctx, datedEntry("2026-12-01", domain.AccountBank, "4400", 100000)); err != nil {
		t.Fatalf("Nachbuchung: %v", err)
	}
	if _, err := closing.CarryForward(ctx, 2027); err != nil {
		t.Fatalf("Korrekturvortrag: %v", err)
	}

	var reversals int
	for _, entry := range entriesOf(t, env, 2027) {
		if entry.Kind == domain.EntryKindReversal {
			reversals++
			if entry.BookingDate != "2027-01-01" {
				t.Errorf("die Generalumkehr des Vortrags steht auf %s, erwartet den Jahresbeginn",
					entry.BookingDate)
			}
		}
	}
	if reversals != 1 {
		t.Errorf("erwartet 1 Generalumkehr im Zieljahr, erhalten %d", reversals)
	}
}

// Ist der Jahresbeginn schon festgeschrieben, rückt der Korrekturvortrag auf den
// ersten offenen Tag und sagt es im Buchungstext.
func TestCarryForwardMovesBehindACommittedPeriod(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	if _, err := env.journal.Post(ctx, datedEntry("2026-02-01", domain.AccountBank, "4400", 100000)); err != nil {
		t.Fatalf("Erlösbuchung: %v", err)
	}
	festschreibung := repository.NewFestschreibungRepository(env.db)
	if err := festschreibung.Create(ctx, &domain.Festschreibung{
		FiscalYear: 2027, PeriodType: "month", PeriodLabel: "Januar 2027",
		CutoffDate: "2027-01-31", ChainHead: domain.GenesisHash,
	}); err != nil {
		t.Fatalf("Festschreibung: %v", err)
	}

	preview, err := closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortragsvorschau: %v", err)
	}
	if !preview.Deferred || preview.BookingDate != "2027-02-01" {
		t.Fatalf("Buchungsdatum = %s (verschoben: %v), erwartet den 2027-02-01",
			preview.BookingDate, preview.Deferred)
	}

	created, err := closing.CarryForward(ctx, 2027)
	if err != nil {
		t.Fatalf("Saldenvortrag buchen: %v", err)
	}
	if created[0].BookingDate != "2027-02-01" {
		t.Errorf("Buchungsdatum = %s, erwartet 2027-02-01", created[0].BookingDate)
	}
	if !strings.Contains(created[0].Description, "festgeschrieben") {
		t.Errorf("der Buchungstext muss die Verschiebung nennen, lautet aber: %q", created[0].Description)
	}
}

// Der Korrekturvortrag hinter einer Festschreibung: Generalumkehr und
// Neuvortrag stehen beide auf dem ersten offenen Tag, und die Eröffnungsbilanz
// stimmt danach über alle Bilanzkonten.
func TestCorrectionBehindACommittedPeriod(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	if _, err := env.journal.Post(ctx, datedEntry("2026-02-01", domain.AccountBank, "4400", 1000000)); err != nil {
		t.Fatalf("Erlösbuchung: %v", err)
	}
	if _, err := closing.CarryForward(ctx, 2027); err != nil {
		t.Fatalf("erster Saldenvortrag: %v", err)
	}

	// Im neuen Jahr ist inzwischen weitergebucht und der Januar festgeschrieben.
	festschreibung := repository.NewFestschreibungRepository(env.db)
	if err := festschreibung.Create(ctx, &domain.Festschreibung{
		FiscalYear: 2027, PeriodType: "month", PeriodLabel: "Januar 2027",
		CutoffDate: "2027-01-31", ChainHead: domain.GenesisHash,
	}); err != nil {
		t.Fatalf("Festschreibung: %v", err)
	}
	// Und im Vorjahr kommt eine Buchung nach.
	if _, err := env.journal.Post(ctx, datedEntry("2026-12-20", domain.AccountBank, "4400", 500000)); err != nil {
		t.Fatalf("Nachbuchung: %v", err)
	}

	state, err := closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortragsstand: %v", err)
	}
	if !state.NeedsCorrection || state.BookingDate != "2027-02-01" || !state.Deferred {
		t.Fatalf("Vortragsstand = Korrektur nötig %v, Datum %s (verschoben %v), erwartet den 2027-02-01",
			state.NeedsCorrection, state.BookingDate, state.Deferred)
	}

	created, err := closing.CarryForward(ctx, 2027)
	if err != nil {
		t.Fatalf("Korrekturvortrag: %v", err)
	}
	if len(created) != 1 || created[0].BookingDate != "2027-02-01" {
		t.Fatalf("der Korrekturvortrag steht auf %+v, erwartet eine Buchung zum 2027-02-01", created)
	}
	if !strings.Contains(created[0].Description, "Korrekturvortrag") {
		t.Errorf("der Buchungstext muss den Korrekturvortrag benennen, lautet aber: %q", created[0].Description)
	}

	var reversals int
	for _, entry := range entriesOf(t, env, 2027) {
		if entry.Kind != domain.EntryKindReversal {
			continue
		}
		reversals++
		if entry.BookingDate != "2027-02-01" {
			t.Errorf("die Generalumkehr steht auf %s, erwartet den ersten offenen Tag", entry.BookingDate)
		}
	}
	if reversals != 1 {
		t.Errorf("erwartet 1 Generalumkehr im Zieljahr, erhalten %d", reversals)
	}

	assertOpeningMatchesClosing(t, env, 2026, 2027, 1500000, domain.AccountGewinnvortrag)
}

// Wurde die Vortragsbuchung außerhalb des Zieljahres storniert — der Storno aus
// der Buchungsansicht trägt den Tag seiner Erstellung —, dann wirkt die
// Generalumkehr im Jahr ihres Datums, und die vorgetragenen Werte stehen im
// Zieljahr weiter. Ein erneuter Lauf darf sie deshalb nicht ein zweites Mal
// buchen: die Eröffnungsbilanz wäre doppelt so hoch wie die Schlussbilanz.
func TestCarryForwardDoesNotDoubleAfterAnOutOfYearReversal(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	if _, err := env.journal.Post(ctx, datedEntry("2026-02-01", domain.AccountBank, "4400", 1000000)); err != nil {
		t.Fatalf("Erlösbuchung: %v", err)
	}
	if _, err := closing.CarryForward(ctx, 2027); err != nil {
		t.Fatalf("Saldenvortrag: %v", err)
	}

	sv := openingEntry(t, env, 2027)
	if _, err := env.journal.ReverseOn(ctx, sv.ID, "Storno aus der Buchungsansicht", "2028-03-01"); err != nil {
		t.Fatalf("Storno im Folgejahr: %v", err)
	}

	state, err := closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortragsstand: %v", err)
	}
	if !state.AlreadyCarried {
		t.Error("die Vortragswerte stehen weiterhin im Zieljahr; der Stand muss sie als vorgetragen ausweisen")
	}
	if state.NeedsCorrection {
		t.Errorf("die Salden stimmen mit dem Vorjahr überein; eine Korrektur ist nicht nötig: %+v", state.Rows)
	}

	before := len(entriesOf(t, env, 2027))
	if _, err := closing.CarryForward(ctx, 2027); err == nil {
		t.Fatal("der stehen gebliebene Vortrag darf nicht ein zweites Mal gebucht werden")
	} else if !strings.Contains(err.Error(), "aktuell") {
		t.Errorf("die Meldung sollte den aktuellen Stand benennen, lautet aber: %v", err)
	}
	if after := len(entriesOf(t, env, 2027)); after != before {
		t.Errorf("der zweite Lauf hat %d Buchungen erzeugt", after-before)
	}
	old := balances(t, env, 2026)
	fresh := balances(t, env, 2027)
	if fresh[domain.AccountBank] != old[domain.AccountBank] {
		t.Errorf("Bank im Zieljahr = %s €, Schlusssaldo des Vorjahres = %s €",
			fresh[domain.AccountBank], old[domain.AccountBank])
	}

	// Und eine Nachbuchung im Vorjahr macht daraus keinen Korrekturvortrag: der
	// alte Vortrag lässt sich nicht mehr zurücknehmen, also wird abgelehnt statt
	// verdoppelt.
	if _, err := env.journal.Post(ctx, datedEntry("2026-11-01", domain.AccountBank, "4400", 500000)); err != nil {
		t.Fatalf("Nachbuchung: %v", err)
	}
	_, err = closing.CarryForward(ctx, 2027)
	if err == nil {
		t.Fatal("ein Korrekturvortrag ohne zurücknehmbaren Altvortrag darf nicht gebucht werden")
	}
	if !strings.Contains(err.Error(), "verdoppeln") {
		t.Errorf("die Meldung sollte die Verdopplung benennen, lautet aber: %v", err)
	}
	if after := len(entriesOf(t, env, 2027)); after != before {
		t.Errorf("der abgelehnte Lauf hat %d Buchungen erzeugt", after-before)
	}
}

// Fehlt schon dem Vorjahr sein Vortrag, geht die Bilanzidentität nicht auf — die
// offenen Posten laufen jahresübergreifend, die Sachkontensalden nicht. Die
// Meldung muss dann auf den übersprungenen Vortrag zeigen.
func TestCarryForwardNamesTheSkippedPriorYear(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	vendor := env.vendor(t, "Lieferant", "DE", "")
	invoice := datedEntry("2025-11-15", "6300", vendor.LedgerAccount, 100000)
	invoice.DocumentNumber = "ER-2025-1"
	invoice.ContactID = &vendor.ID
	invoice.Lines[1].ContactID = &vendor.ID
	if _, err := env.journal.Post(ctx, invoice); err != nil {
		t.Fatalf("Eingangsrechnung 2025: %v", err)
	}

	// 2025 → 2026 wurde nie vorgetragen: der offene Posten steht zum 31.12.2026
	// da, in den Sachkontensalden des Jahres 2026 steht er nicht.
	preview, err := closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortragsvorschau: %v", err)
	}
	if preview.IsBalanced || !preview.PriorYearNotCarried {
		t.Fatalf("erwartet eine Differenz und den Hinweis auf das Vorjahr: ausgeglichen %v, Vorjahr ohne Vortrag %v",
			preview.IsBalanced, preview.PriorYearNotCarried)
	}

	_, err = closing.CarryForward(ctx, 2027)
	if err == nil {
		t.Fatal("ein Jahr, dessen Salden nicht aufgehen, darf nicht vorgetragen werden")
	}
	if !strings.Contains(err.Error(), "2025 → 2026") {
		t.Errorf("die Meldung muss den fehlenden Vortrag 2025 → 2026 benennen, lautet aber: %v", err)
	}

	// Und dem Hinweis zu folgen hilft: erst 2026, dann 2027.
	if _, err := closing.CarryForward(ctx, 2026); err != nil {
		t.Fatalf("Saldenvortrag 2025 → 2026: %v", err)
	}
	if _, err := closing.CarryForward(ctx, 2027); err != nil {
		t.Fatalf("Saldenvortrag 2026 → 2027: %v", err)
	}
	if b := balances(t, env, 2027)[vendor.LedgerAccount]; b != -100000 {
		t.Errorf("das Kreditorenkonto steht im Jahr 2027 auf %s €, erwartet 1.000,00 im Haben", b)
	}
}

// Der Vortrag ist ein Vorgang und kein Bündel einzelner Buchungen: geht eine von
// ihnen nicht durch, wird gar nichts geschrieben. Sonst bliebe das Zieljahr mit
// zurückgenommenem Altvortrag und halbem Neuvortrag zurück — mit einer
// Eröffnungsbilanz also, die es so nie gab.
func TestCarryForwardWritesNothingWhenOnePostingFails(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	vendor := env.vendor(t, "Lieferant", "DE", "")
	env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)
	if _, err := closing.CarryForward(ctx, 2027); err != nil {
		t.Fatalf("erster Saldenvortrag: %v", err)
	}
	before := entriesOf(t, env, 2027)

	// Eine Nachbuchung im Vorjahr macht den Korrekturvortrag nötig …
	if _, err := env.journal.Post(ctx, datedEntry("2026-12-01", domain.AccountBank, "4400", 50000)); err != nil {
		t.Fatalf("Nachbuchung: %v", err)
	}
	// … und der Geschäftspartner, auf dessen Personenkonto der offene Posten
	// steht, ist inzwischen gelöscht: die Kreditorenbuchung geht nicht durch.
	if err := env.contactRepo.Delete(ctx, vendor.ID); err != nil {
		t.Fatalf("Geschäftspartner löschen: %v", err)
	}

	if _, err := closing.CarryForward(ctx, 2027); err == nil {
		t.Fatal("der Korrekturvortrag darf nicht durchgehen, wenn eine seiner Buchungen scheitert")
	}

	after := entriesOf(t, env, 2027)
	if len(after) != len(before) {
		t.Errorf("der gescheiterte Lauf hat %d Buchungen hinterlassen", len(after)-len(before))
	}
	for _, entry := range after {
		if entry.Kind == domain.EntryKindReversal {
			t.Error("der bestehende Vortrag darf nicht zurückgenommen sein, solange der neue nicht steht")
		}
	}

	state, err := closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortragsstand: %v", err)
	}
	if !state.AlreadyCarried || state.Irreversible {
		t.Errorf("der bestehende Vortrag muss unversehrt bleiben: vorgetragen %v, nicht korrigierbar %v",
			state.AlreadyCarried, state.Irreversible)
	}
}

// Gehen Aktiva, Passiva und Jahresergebnis nicht zusammen, wird der Vortrag
// abgelehnt statt den Fehler ins neue Jahr zu tragen.
func TestCarryForwardRefusesAnUnbalancedYear(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	customer := env.customer(t, "Kundin ohne Zuordnung", "DE", "")
	// Eine Buchung auf ein Personenkonto ohne Geschäftspartner an der Buchung:
	// der Saldo steht auf dem Konto, ein offener Posten entsteht daraus nicht.
	entry := datedEntry("2026-02-01", customer.LedgerAccount, "4400", 119000)
	if _, err := env.journal.Post(ctx, entry); err != nil {
		t.Fatalf("Buchung auf das Personenkonto: %v", err)
	}

	preview, err := closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortragsvorschau: %v", err)
	}
	if preview.IsBalanced {
		t.Fatal("die Salden dürfen nicht als ausgeglichen gelten")
	}

	_, err = closing.CarryForward(ctx, 2027)
	if err == nil {
		t.Fatal("ein Jahr, dessen Salden nicht aufgehen, darf nicht vorgetragen werden")
	}
	if !strings.Contains(err.Error(), preview.BalanceDifference.String()) {
		t.Errorf("die Meldung muss die Differenz benennen, lautet aber: %v", err)
	}
	if len(entriesOf(t, env, 2027)) != 0 {
		t.Error("es darf keine Vortragsbuchung entstanden sein")
	}
}

// --- Personenkonten -------------------------------------------------------

// Je offenem Posten eine Zeile, keine Dubletten in der OP-Liste, und die Zahlung
// im neuen Jahr gleicht den alten Posten aus.
func TestCarryForwardOpenItemsPerDocument(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	payments := env.payments(t)
	ctx := context.Background()

	vendor := env.vendor(t, "Lieferant", "DE", "")
	invoice := env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)

	before, err := payments.OpenItems(ctx)
	if err != nil {
		t.Fatalf("offene Posten vor dem Vortrag: %v", err)
	}
	if len(before) != 1 {
		t.Fatalf("erwartet 1 offener Posten, erhalten %d", len(before))
	}

	preview, err := closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortragsvorschau: %v", err)
	}
	if !preview.IsBalanced {
		t.Fatalf("die Salden gehen um %s € nicht auf", preview.BalanceDifference)
	}
	row := rowFor(preview, vendor.LedgerAccount)
	if row == nil {
		t.Fatalf("die Vorschau kennt das Kreditorenkonto %s nicht", vendor.LedgerAccount)
	}
	if row.Kind != CarryForwardKreditor || row.OpenItems != 1 || row.ClosingBalance != -119000 {
		t.Errorf("Kreditorenzeile = %+v, erwartet einen offenen Posten über 1.190,00 im Haben", row)
	}

	created, err := closing.CarryForward(ctx, 2027)
	if err != nil {
		t.Fatalf("Saldenvortrag buchen: %v", err)
	}
	if len(created) != 2 {
		t.Fatalf("erwartet 2 Vortragsbuchungen (Sachkonten und Kreditoren), erhalten %d", len(created))
	}

	kreditoren := created[1]
	assertLines(t, &kreditoren, []bookedLine{
		{domain.SideCredit, vendor.LedgerAccount, 119000},
		{domain.SideDebit, domain.AccountSaldenvortraegeKreditoren, 119000},
	})
	var ledgerText string
	for _, line := range kreditoren.Lines {
		if line.Account == vendor.LedgerAccount {
			if line.ContactID == nil || *line.ContactID != vendor.ID {
				t.Errorf("die Vortragszeile muss auf den Geschäftspartner verweisen: %+v", line)
			}
			ledgerText = line.Text
		}
	}
	if !strings.Contains(ledgerText, "vom 01.03.2026") {
		t.Errorf("die Vortragszeile muss Belegnummer und Belegdatum tragen, lautet aber %q", ledgerText)
	}

	// Die OP-Liste des neuen Jahres zeigt denselben Posten, nicht zwei.
	payments.SetFiscalYear(2027)
	after, err := payments.OpenItems(ctx)
	if err != nil {
		t.Fatalf("offene Posten nach dem Vortrag: %v", err)
	}
	if len(after) != 1 {
		t.Fatalf("nach dem Vortrag muss es weiterhin genau einen offenen Posten geben, es sind %d", len(after))
	}
	if after[0].EntryID != invoice.ID {
		t.Errorf("der offene Posten muss die Rechnung von 2026 bleiben, ist aber Buchung %d", after[0].EntryID)
	}
	if after[0].OpenAmount != before[0].OpenAmount {
		t.Errorf("offener Betrag = %s €, vor dem Vortrag %s €", after[0].OpenAmount, before[0].OpenAmount)
	}

	// Und die Zahlung im neuen Jahr gleicht ihn aus.
	if _, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2027-02-15",
		Allocations:    []AllocationRequest{{OpenItemEntryID: invoice.ID, SettledAmount: 119000}},
	}); err != nil {
		t.Fatalf("Zahlung im neuen Jahr: %v", err)
	}
	if items, _ := payments.OpenItems(ctx); len(items) != 0 {
		t.Errorf("nach der Zahlung darf kein offener Posten bleiben, es sind %d", len(items))
	}
	if b := balances(t, env, 2027)[vendor.LedgerAccount]; b != 0 {
		t.Errorf("das Kreditorenkonto steht nach der Zahlung auf %s €, erwartet null", b)
	}
}

// Eine Eröffnungsbuchung auf ein Personenkonto begründet keinen offenen Posten.
//
// Der Posten ist und bleibt die Rechnung, aus der er entstanden ist; gegen sie
// läuft die Zahlung, und an ihr hängen Fälligkeit und Belegnummer. Zählte der
// Vortrag noch einmal mit, stünde jede Forderung nach dem Jahreswechsel doppelt
// in der OP-Liste — einmal als Rechnung und einmal als ihr eigener Vortrag.
func TestOpenItemsIgnoreOpeningEntries(t *testing.T) {
	env := newTestEnv(t)
	payments := env.payments(t)
	ctx := context.Background()

	customer := env.customer(t, "Kundin", "DE", "")
	opening := datedEntry("2026-01-01", customer.LedgerAccount, domain.AccountSaldenvortraegeDebitoren, 119000)
	opening.Source = domain.EntrySourceOpening
	opening.ContactID = &customer.ID
	opening.DocumentNumber = "SV 2026"
	opening.Lines[0].ContactID = &customer.ID
	if _, err := env.journal.Post(ctx, opening); err != nil {
		t.Fatalf("Eröffnungsbuchung: %v", err)
	}

	items, err := payments.OpenItems(ctx)
	if err != nil {
		t.Fatalf("offene Posten: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("eine Eröffnungsbuchung ist kein offener Posten, es sind aber %d: %+v", len(items), items)
	}
}

// Eine Rechnung, die im alten Jahr schon bezahlt war, wird nicht vorgetragen.
func TestCarryForwardSkipsItemsSettledBeforeTheCutoff(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	payments := env.payments(t)
	ctx := context.Background()

	vendor := env.vendor(t, "Lieferant", "DE", "")
	invoice := env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)
	if _, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-04-01",
		Allocations:    []AllocationRequest{{OpenItemEntryID: invoice.ID, SettledAmount: 119000}},
	}); err != nil {
		t.Fatalf("Zahlung im alten Jahr: %v", err)
	}

	preview, err := closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortragsvorschau: %v", err)
	}
	if row := rowFor(preview, vendor.LedgerAccount); row != nil {
		t.Errorf("ein bereits ausgeglichener Posten gehört nicht in den Vortrag: %+v", row)
	}
	if !preview.IsBalanced {
		t.Errorf("die Salden gehen um %s € nicht auf", preview.BalanceDifference)
	}
}

// Eine Dezemberrechnung, die erst im neuen Jahr bezahlt wird, war am
// Bilanzstichtag offen und gehört in die Eröffnungsbilanz — auch wenn die
// Zahlung vor dem Vortrag gebucht wurde.
func TestCarryForwardUsesTheBalanceSheetDate(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	payments := env.payments(t)
	ctx := context.Background()

	vendor := env.vendor(t, "Lieferant", "DE", "")
	invoice := env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)

	payments.SetFiscalYear(2027)
	if _, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2027-01-20",
		Allocations:    []AllocationRequest{{OpenItemEntryID: invoice.ID, SettledAmount: 119000}},
	}); err != nil {
		t.Fatalf("Zahlung im neuen Jahr: %v", err)
	}

	preview, err := closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortragsvorschau: %v", err)
	}
	row := rowFor(preview, vendor.LedgerAccount)
	if row == nil || row.ClosingBalance != -119000 {
		t.Fatalf("der am 31.12. offene Posten fehlt im Vortrag: %+v", row)
	}
	if !preview.IsBalanced {
		t.Errorf("die Salden gehen um %s € nicht auf", preview.BalanceDifference)
	}

	if _, err := closing.CarryForward(ctx, 2027); err != nil {
		t.Fatalf("Saldenvortrag buchen: %v", err)
	}
	if b := balances(t, env, 2027)[vendor.LedgerAccount]; b != 0 {
		t.Errorf("Vortrag und Zahlung müssen sich aufheben, das Konto steht auf %s €", b)
	}
}

// Die Steuerkonten sind Bilanzkonten und werden vorgetragen — aber ihr Vortrag
// ist kein Umsatz und keine Vorsteuer des neuen Jahres.
func TestCarryForwardKeepsTaxAccountsOutOfTheVatReturn(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	vendor := env.vendor(t, "Lieferant", "DE", "")
	env.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)

	if _, err := closing.CarryForward(ctx, 2027); err != nil {
		t.Fatalf("Saldenvortrag buchen: %v", err)
	}
	if b := balances(t, env, 2027)[domain.AccountVorsteuer19]; b != 19000 {
		t.Errorf("die Vorsteuer steht im neuen Jahr auf %s €, erwartet 190,00 im Soll", b)
	}

	vat := NewVatService(env.journalRepo, 2027)
	summary, err := vat.Summary(ctx, "", "")
	if err != nil {
		t.Fatalf("Umsatzsteuer-Auswertung 2027: %v", err)
	}
	if summary.InputTax != 0 {
		t.Errorf("die Voranmeldung 2027 weist %s € Vorsteuer aus; der Saldenvortrag ist kein Vorsteuerabzug",
			summary.InputTax)
	}
}

// --- Abschlussstand -------------------------------------------------------

func TestFiscalYearStatusFlow(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	if _, err := env.journal.Post(ctx, datedEntry("2026-02-01", domain.AccountBank, "4400", 100000)); err != nil {
		t.Fatalf("Erlösbuchung: %v", err)
	}

	// Ein Schritt darf nicht übersprungen werden.
	if _, err := closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearAdopted, "2027-06-30", "Beschluss"); err == nil {
		t.Error("ein Abschluss kann nicht festgestellt werden, bevor er aufgestellt ist")
	}

	// Aufstellen geht erst nach dem Ende des Geschäftsjahres.
	if _, err := closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearPrepared, "2026-06-30", ""); err == nil {
		t.Error("ein Abschluss lässt sich nicht mitten im Geschäftsjahr aufstellen")
	}
	if _, err := closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearPrepared, "2027-03-31", ""); err != nil {
		t.Fatalf("Aufstellung: %v", err)
	}

	// Ohne Jahres-Festschreibung keine Feststellung.
	_, err := closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearAdopted, "2027-06-30", "Beschluss vom 30.06.2027")
	if err == nil {
		t.Fatal("ohne Jahres-Festschreibung darf nicht festgestellt werden")
	}
	if !strings.Contains(err.Error(), "Festschreibung") {
		t.Errorf("die Meldung sollte die fehlende Festschreibung benennen, lautet aber: %v", err)
	}

	festschreibung := repository.NewFestschreibungRepository(env.db)
	if err := festschreibung.Create(ctx, &domain.Festschreibung{
		FiscalYear: 2026, PeriodType: "year", PeriodLabel: "Geschäftsjahr 2026",
		CutoffDate: "2026-12-31", ChainHead: domain.GenesisHash,
	}); err != nil {
		t.Fatalf("Festschreibung: %v", err)
	}

	fy, err := closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearAdopted, "2027-06-30", "Beschluss vom 30.06.2027")
	if err != nil {
		t.Fatalf("Feststellung: %v", err)
	}
	if !fy.IsAdopted() || fy.AdoptionNote != "Beschluss vom 30.06.2027" {
		t.Errorf("das festgestellte Jahr = %+v", fy)
	}

	// Ab hier nimmt das Jahr keine Buchung mehr auf.
	_, err = env.journal.Post(ctx, datedEntry("2026-12-15", domain.AccountBank, "4400", 5000))
	if err == nil {
		t.Fatal("in ein festgestelltes Jahr darf nicht gebucht werden")
	}
	if !strings.Contains(err.Error(), "festgestellt") || !strings.Contains(err.Error(), "zurück") {
		t.Errorf("die Meldung sollte die Feststellung und die Rücksetzung benennen, lautet aber: %v", err)
	}

	// Im offenen Folgejahr bleibt das Buchen möglich.
	if _, err := env.journal.Post(ctx, datedEntry("2027-01-15", domain.AccountBank, "4400", 5000)); err != nil {
		t.Errorf("das Folgejahr ist offen und muss buchbar bleiben: %v", err)
	}
}

func TestReopenNeedsAReasonAndIsLogged(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()
	auditRepo := repository.NewAuditRepository(env.db)

	festschreibung := repository.NewFestschreibungRepository(env.db)
	if err := festschreibung.Create(ctx, &domain.Festschreibung{
		FiscalYear: 2026, PeriodType: "year", PeriodLabel: "Geschäftsjahr 2026",
		CutoffDate: "2026-12-31", ChainHead: domain.GenesisHash,
	}); err != nil {
		t.Fatalf("Festschreibung: %v", err)
	}
	if _, err := closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearPrepared, "2027-03-31", ""); err != nil {
		t.Fatalf("Aufstellung: %v", err)
	}
	if _, err := closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearAdopted, "2027-06-30", "Beschluss"); err != nil {
		t.Fatalf("Feststellung: %v", err)
	}

	if _, err := closing.ReopenFiscalYear(ctx, 2026, "   "); err == nil {
		t.Error("eine Rücksetzung ohne Grund darf nicht durchgehen")
	}

	fy, err := closing.ReopenFiscalYear(ctx, 2026, "Nachträgliche Rechnung des Steuerberaters")
	if err != nil {
		t.Fatalf("Rücksetzung: %v", err)
	}
	if fy.Status != domain.FiscalYearPrepared || fy.AdoptedOn != "" {
		t.Errorf("nach der Rücksetzung = %+v, erwartet den Stand „aufgestellt\" ohne Feststellungsdatum", fy)
	}

	// Die Festschreibung bleibt unberührt: sie ist der Nachweis über die
	// Buchungen und wird durch eine spätere Entscheidung nicht falsch.
	records, err := festschreibung.FindByFiscalYear(ctx, 2026)
	if err != nil || len(records) != 1 {
		t.Errorf("die Festschreibung muss bestehen bleiben, gefunden: %d (%v)", len(records), err)
	}

	logs, err := auditRepo.FindAll(ctx, 100)
	if err != nil {
		t.Fatalf("Protokoll: %v", err)
	}
	var found bool
	for _, log := range logs {
		if log.EntityType == "FISCAL_YEAR" && strings.Contains(log.Details, "Nachträgliche Rechnung") {
			found = true
		}
	}
	if !found {
		t.Error("die Rücksetzung muss mit ihrem Grund im Protokoll stehen")
	}

	// Und das Jahr nimmt wieder Buchungen auf.
	if _, err := env.journal.Post(ctx, datedEntry("2027-01-05", domain.AccountBank, "4400", 5000)); err != nil {
		t.Errorf("nach der Rücksetzung muss wieder gebucht werden können: %v", err)
	}
}

// --- Geschäftsjahre -------------------------------------------------------

// Bestehende Datenbanken kennen das Geschäftsjahr nur als Zahl an der Buchung.
func TestEnsureFiscalYearsCreatesEntitiesForBookedYears(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	if _, err := env.journal.Post(ctx, datedEntry("2025-05-01", domain.AccountBank, "4400", 100000)); err != nil {
		t.Fatalf("Buchung 2025: %v", err)
	}
	if _, err := env.journal.Post(ctx, datedEntry("2026-05-01", domain.AccountBank, "4400", 100000)); err != nil {
		t.Fatalf("Buchung 2026: %v", err)
	}

	if err := closing.EnsureFiscalYears(ctx); err != nil {
		t.Fatalf("Geschäftsjahre anlegen: %v", err)
	}

	years, err := closing.FiscalYears(ctx)
	if err != nil {
		t.Fatalf("Geschäftsjahre lesen: %v", err)
	}
	if len(years) != 2 {
		t.Fatalf("erwartet 2 Geschäftsjahre, erhalten %d: %+v", len(years), years)
	}
	if years[0].Year != 2025 || years[0].StartDate != "2025-01-01" || years[0].EndDate != "2025-12-31" {
		t.Errorf("Geschäftsjahr 2025 = %+v", years[0])
	}
	if years[1].Year != 2026 || years[1].Status != domain.FiscalYearOpen {
		t.Errorf("Geschäftsjahr 2026 = %+v", years[1])
	}
	for _, fy := range years {
		if fy.IsShort {
			t.Errorf("das Kalenderjahr %d ist kein Rumpfgeschäftsjahr", fy.Year)
		}
	}

	// Ein zweiter Lauf ändert nichts.
	if err := closing.EnsureFiscalYears(ctx); err != nil {
		t.Fatalf("zweiter Lauf: %v", err)
	}
	if again, _ := closing.FiscalYears(ctx); len(again) != 2 {
		t.Errorf("der zweite Lauf hat %d Geschäftsjahre hinterlassen", len(again))
	}
}

// Ein Geschäftsjahr, das nebenbei entsteht — der Blick auf den Abschlussstand
// legt es an —, gehört genauso ins Protokoll wie eines aus der Jahresanlage
// (Entscheidung 8: alle Aktionen mit EntityType FISCAL_YEAR).
func TestFiscalYearCreatedOnTheFlyIsLogged(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()
	auditRepo := repository.NewAuditRepository(env.db)

	if _, err := closing.ClosingStateFor(ctx, 2026); err != nil {
		t.Fatalf("Abschlussstand: %v", err)
	}

	logs, err := auditRepo.FindAll(ctx, 100)
	if err != nil {
		t.Fatalf("Protokoll: %v", err)
	}
	var found bool
	for _, log := range logs {
		if log.EntityType == "FISCAL_YEAR" && log.EntityID == "2026" && strings.Contains(log.Details, "angelegt") {
			found = true
		}
	}
	if !found {
		t.Error("das angelegte Geschäftsjahr 2026 muss im Protokoll stehen")
	}
}

// Das Gründungsjahr beginnt mit der Beurkundung und ist ein Rumpfgeschäftsjahr.
func TestEnsureFiscalYearsStartsFoundingYearAtNotarization(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	foundations := repository.NewFoundationRepository(env.db)
	if err := foundations.Save(ctx, &domain.Foundation{
		NotarizedOn: "2026-03-15", ShareCapital: 2500000,
	}); err != nil {
		t.Fatalf("Gründung: %v", err)
	}
	closing.SetFoundationRepo(foundations)

	if err := closing.EnsureFiscalYears(ctx); err != nil {
		t.Fatalf("Geschäftsjahre anlegen: %v", err)
	}
	years, err := closing.FiscalYears(ctx)
	if err != nil || len(years) != 1 {
		t.Fatalf("erwartet ein Geschäftsjahr, erhalten %d (%v)", len(years), err)
	}
	if years[0].StartDate != "2026-03-15" || years[0].EndDate != "2026-12-31" {
		t.Errorf("Gründungsjahr = %s bis %s, erwartet 2026-03-15 bis 2026-12-31",
			years[0].StartDate, years[0].EndDate)
	}
	if !years[0].IsShort {
		t.Error("das Gründungsjahr ist ein Rumpfgeschäftsjahr")
	}
}

// Das Folgejahr beginnt am Tag nach dem Ende des Vorjahres — auch nach einem
// Rumpfgeschäftsjahr.
func TestCreateFiscalYearFollowsThePreviousPeriod(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	foundations := repository.NewFoundationRepository(env.db)
	if err := foundations.Save(ctx, &domain.Foundation{
		NotarizedOn: "2026-03-15", ShareCapital: 2500000,
	}); err != nil {
		t.Fatalf("Gründung: %v", err)
	}
	closing.SetFoundationRepo(foundations)
	if err := closing.EnsureFiscalYears(ctx); err != nil {
		t.Fatalf("Geschäftsjahre anlegen: %v", err)
	}

	fy, err := closing.CreateFiscalYear(ctx, 2027)
	if err != nil {
		t.Fatalf("Geschäftsjahr 2027 anlegen: %v", err)
	}
	if fy.StartDate != "2027-01-01" || fy.EndDate != "2027-12-31" {
		t.Errorf("Geschäftsjahr 2027 = %s bis %s, erwartet das volle Kalenderjahr", fy.StartDate, fy.EndDate)
	}
	if fy.IsShort {
		t.Error("das Folgejahr eines Rumpfjahres ist selbst kein Rumpfjahr")
	}

	// Zweimal anlegen ändert nichts.
	again, err := closing.CreateFiscalYear(ctx, 2027)
	if err != nil {
		t.Fatalf("zweiter Aufruf: %v", err)
	}
	if again.StartDate != fy.StartDate {
		t.Errorf("das bestehende Geschäftsjahr wurde überschrieben: %+v", again)
	}
}

// Der Abschlussstand fasst zusammen, was die Ansicht braucht.
func TestClosingStateReportsCommitmentAndCarryForward(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	if _, err := env.journal.Post(ctx, datedEntry("2026-02-01", domain.AccountBank, "4400", 1000000)); err != nil {
		t.Fatalf("Erlösbuchung: %v", err)
	}

	state, err := closing.ClosingStateFor(ctx, 2026)
	if err != nil {
		t.Fatalf("Abschlussstand: %v", err)
	}
	if state.NetIncome != 1000000 {
		t.Errorf("Jahresergebnis = %s €, erwartet 10.000,00", state.NetIncome)
	}
	if state.HasYearCommitment {
		t.Error("ohne Festschreibung darf keine gemeldet werden")
	}
	if state.CarriedForward {
		t.Error("vor dem Vortrag darf keiner gemeldet werden")
	}
	if state.NextStatus != domain.FiscalYearPrepared || !state.CanAdopt {
		t.Errorf("nächster Schritt = %q (möglich: %v), erwartet die Aufstellung", state.NextStatus, state.CanAdopt)
	}

	// Das Ansehen darf kein Geschäftsjahr anlegen: sonst stünde das Folgejahr
	// in der Auswahl, ohne dass es jemand eröffnet hätte.
	years, err := closing.FiscalYears(ctx)
	if err != nil {
		t.Fatalf("Geschäftsjahre: %v", err)
	}
	if len(years) != 1 || years[0].Year != 2026 {
		t.Errorf("nach dem Blick auf den Abschlussstand sind %d Geschäftsjahre erfasst: %+v", len(years), years)
	}

	if _, err := closing.CarryForward(ctx, 2027); err != nil {
		t.Fatalf("Saldenvortrag: %v", err)
	}
	state, err = closing.ClosingStateFor(ctx, 2026)
	if err != nil {
		t.Fatalf("Abschlussstand: %v", err)
	}
	if !state.CarriedForward || !state.CarryForwardCurrent {
		t.Errorf("nach dem Vortrag = vorgetragen %v, aktuell %v", state.CarriedForward, state.CarryForwardCurrent)
	}
	if state.CarriedForwardAt == nil || time.Since(*state.CarriedForwardAt) > time.Hour {
		t.Errorf("der Zeitpunkt des Vortrags fehlt oder ist unplausibel: %v", state.CarriedForwardAt)
	}

	// Eine Nachbuchung im Vorjahr macht den Vortrag überholt.
	if _, err := env.journal.Post(ctx, datedEntry("2026-12-01", domain.AccountBank, "4400", 100000)); err != nil {
		t.Fatalf("Nachbuchung: %v", err)
	}
	state, err = closing.ClosingStateFor(ctx, 2026)
	if err != nil {
		t.Fatalf("Abschlussstand: %v", err)
	}
	if !state.CarriedForward || state.CarryForwardCurrent {
		t.Errorf("nach der Nachbuchung = vorgetragen %v, aktuell %v (erwartet: vorgetragen, nicht aktuell)",
			state.CarriedForward, state.CarryForwardCurrent)
	}
}

// entriesOf liefert die Buchungen eines Geschäftsjahres.
func entriesOf(t *testing.T, e *testEnv, year int) []domain.JournalEntry {
	t.Helper()
	entries, err := e.journalRepo.FindAll(context.Background(), year)
	if err != nil {
		t.Fatalf("Buchungen des Jahres %d: %v", year, err)
	}
	return entries
}

// Die Arbeitnehmerzahl bestimmt über die Größenklasse die Gliederungstiefe und
// den Umfang der Offenlegung. Ein festgestellter Abschluss darf sie deshalb
// nicht mehr annehmen — sonst stünde derselbe Abschluss morgen anders da.
func TestAverageEmployeesRefusedAfterAdoption(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	festschreibung := repository.NewFestschreibungRepository(env.db)
	if err := festschreibung.Create(ctx, &domain.Festschreibung{
		FiscalYear: 2026, PeriodType: "year", PeriodLabel: "Geschäftsjahr 2026",
		CutoffDate: "2026-12-31", ChainHead: domain.GenesisHash,
	}); err != nil {
		t.Fatalf("Festschreibung: %v", err)
	}
	if _, err := closing.SetAverageEmployees(ctx, 2026, 12); err != nil {
		t.Fatalf("Arbeitnehmerzahl im offenen Jahr: %v", err)
	}
	if _, err := closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearPrepared, "2027-03-31", ""); err != nil {
		t.Fatalf("Aufstellung: %v", err)
	}
	// Aufgestellt heißt noch nicht beschlossen: bis zur Feststellung bleibt die
	// Zahl änderbar.
	if _, err := closing.SetAverageEmployees(ctx, 2026, 13); err != nil {
		t.Fatalf("Arbeitnehmerzahl im aufgestellten Jahr: %v", err)
	}
	if _, err := closing.SetFiscalYearStatus(ctx, 2026, domain.FiscalYearAdopted, "2027-06-30", "Beschluss"); err != nil {
		t.Fatalf("Feststellung: %v", err)
	}

	_, err := closing.SetAverageEmployees(ctx, 2026, 400)
	if err == nil {
		t.Fatal("ein festgestelltes Geschäftsjahr darf die Arbeitnehmerzahl nicht mehr annehmen")
	}
	// Die Meldung nennt Ursache und nächsten Schritt (§15.3).
	for _, want := range []string{"Festgestellt", "Feststellung zurück"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("die Fehlermeldung nennt %q nicht: %v", want, err)
		}
	}
	fy, err := closing.YearOf(ctx, 2026)
	if err != nil {
		t.Fatalf("Geschäftsjahr: %v", err)
	}
	if fy.AverageEmployees != 13 {
		t.Errorf("die Arbeitnehmerzahl steht auf %d, erwartet den Stand 13 vor der Feststellung",
			fy.AverageEmployees)
	}

	// Nach der Rücksetzung ist sie wieder zu erfassen — das ist der Weg, den
	// die Fehlermeldung nennt.
	if _, err := closing.ReopenFiscalYear(ctx, 2026, "Korrektur der Arbeitnehmerzahl"); err != nil {
		t.Fatalf("Rücksetzung: %v", err)
	}
	if _, err := closing.SetAverageEmployees(ctx, 2026, 400); err != nil {
		t.Errorf("nach der Rücksetzung muss die Arbeitnehmerzahl wieder änderbar sein: %v", err)
	}
}
