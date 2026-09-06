package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Der Blick auf den Abschlussstand legt kein Geschäftsjahr an.
//
// Die Abschlussansicht wird beim Blättern durch die Jahre aufgerufen. Legte sie
// dabei an, stünde jedes angesehene Jahr danach in der Auswahl — angelegt hat es
// niemand, und der Saldenvortrag hinge an einem Jahr, das es nicht geben sollte.
func TestClosingStateForCreatesNoFiscalYear(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	state, err := closing.ClosingStateFor(ctx, 2031)
	if err != nil {
		t.Fatalf("Abschlussstand: %v", err)
	}
	if state.Year != 2031 || state.FiscalYear.Year != 2031 {
		t.Errorf("der Abschlussstand muss das angefragte Jahr beschreiben: %+v", state.FiscalYear)
	}

	years, err := closing.FiscalYears(ctx)
	if err != nil {
		t.Fatalf("Geschäftsjahre: %v", err)
	}
	for _, fy := range years {
		if fy.Year == 2031 {
			t.Fatalf("das bloße Ansehen hat das Geschäftsjahr 2031 angelegt: %+v", fy)
		}
	}
}

// Die Generalumkehr einer Eröffnungsbuchung bleibt im Geschäftsjahr der
// Ursprungsbuchung.
//
// Das vorgegebene Datum gibt es nur für den Korrekturvortrag. Landete die
// Rücknahme in einem anderen Jahr, stünde der Vortrag dort doppelt und hier
// eine Umkehr ohne Gegenstück.
func TestReverseOnKeepsTheOpeningEntryInItsFiscalYear(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	if _, err := env.journal.Post(ctx, datedEntry("2026-02-01", domain.AccountBank, "4400", 500000)); err != nil {
		t.Fatalf("Erlösbuchung: %v", err)
	}
	created, err := closing.CarryForward(ctx, 2027)
	if err != nil {
		t.Fatalf("Saldenvortrag: %v", err)
	}
	if len(created) == 0 {
		t.Fatal("der Saldenvortrag hat nichts gebucht")
	}
	opening := created[0]

	// Ein Datum aus einem anderen Geschäftsjahr wird abgewiesen …
	if _, err := env.journal.ReverseOn(ctx, opening.ID, "Korrekturvortrag", "2028-01-01"); err == nil {
		t.Error("die Umkehr einer Eröffnungsbuchung darf nicht in ein anderes Geschäftsjahr datiert werden")
	} else if !strings.Contains(err.Error(), "2028") {
		t.Errorf("die Meldung muss das falsche Jahr benennen, lautet aber: %v", err)
	}

	// … ein Datum im Jahr der Ursprungsbuchung geht durch.
	reversal, err := env.journal.ReverseOn(ctx, opening.ID, "Korrekturvortrag", "2027-01-01")
	if err != nil {
		t.Fatalf("Korrekturvortrag: %v", err)
	}
	if reversal.FiscalYear != opening.FiscalYear {
		t.Errorf("die Umkehr steht im Geschäftsjahr %d, die Ursprungsbuchung in %d",
			reversal.FiscalYear, opening.FiscalYear)
	}
}

// Ein Geschäftsjahr lässt sich nur als Folgejahr des zuletzt erfassten anlegen.
//
// Eine Lücke ließe einen Zeitraum ohne Geschäftsjahr: die Buchungen darin
// gehörten zu keinem Abschluss, und der Saldenvortrag fände keinen Anschluss.
func TestCreateFiscalYearRejectsAGap(t *testing.T) {
	env := newTestEnv(t)
	closing := env.closing(t)
	ctx := context.Background()

	if _, err := closing.CreateFiscalYear(ctx, 2026); err != nil {
		t.Fatalf("erstes Geschäftsjahr: %v", err)
	}
	if _, err := closing.CreateFiscalYear(ctx, 2029); err == nil {
		t.Error("ein Sprung von 2026 auf 2029 lässt drei Jahre ohne Geschäftsjahr")
	} else if !strings.Contains(err.Error(), "2027") {
		t.Errorf("die Meldung muss das anschließende Jahr nennen, lautet aber: %v", err)
	}
	if _, err := closing.CreateFiscalYear(ctx, 2027); err != nil {
		t.Errorf("das Folgejahr muss sich anlegen lassen: %v", err)
	}
}

// stubSizeClass ist eine feste Größenklasse. Der Prüflauf soll auf die
// Ankündigung reagieren, nicht die Beurteilung noch einmal rechnen — die steht
// in den Tests des Pakets accounting.
type stubSizeClass struct{ class *domain.SizeClass }

func (s stubSizeClass) SizeClassFor(context.Context, int) (*domain.SizeClass, error) {
	return s.class, nil
}

// Der sich abzeichnende Größenklassenwechsel steht im Abschluss-Prüflauf.
//
// Gemeldet wird ab dem zweiten Stichtag, der die abweichende Klasse ergibt: ein
// einzelner Ausreißer ist ein gutes oder schlechtes Jahr, zwei sind eine
// Ankündigung.
func TestSizeClassChangeAppearsInTheYearlyCheckRun(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	change := &domain.SizeClass{
		Year: 2026, Class: domain.SizeSmall,
		PendingChange: &domain.SizeClassChange{
			From: domain.SizeSmall, To: domain.SizeMedium, Occurrences: 2,
			Note: "Der Abschlussstichtag 31.12.2026 ergibt Mittelgroße Kapitalgesellschaft.",
		},
	}

	checks := env.checks(t)
	checks.SetSizeClassSource(stubSizeClass{class: change})

	run, err := checks.Preview(ctx, CheckRequest{CutoffDate: "2026-12-31", PeriodType: "year"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	found := false
	for _, finding := range run.Findings {
		if finding.Rule != domain.CheckRuleSizeClassChange {
			continue
		}
		found = true
		if finding.Severity != domain.CheckWarning {
			t.Errorf("die Ankündigung blockiert nicht: %q", finding.Severity)
		}
		if !strings.Contains(finding.Message, "Mittelgroße") {
			t.Errorf("der Befund nennt die kommende Klasse nicht: %q", finding.Message)
		}
	}
	if !found {
		t.Errorf("die Ankündigung fehlt im Prüfbericht: %+v", run.Findings)
	}

	// Ein einzelner abweichender Stichtag genügt nicht.
	change.PendingChange.Occurrences = 1
	single, err := checks.Preview(ctx, CheckRequest{CutoffDate: "2026-12-31", PeriodType: "year"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	for _, finding := range single.Findings {
		if finding.Rule == domain.CheckRuleSizeClassChange {
			t.Error("ein einzelner abweichender Stichtag ist noch keine Ankündigung")
		}
	}

	// Und der Monatslauf beurteilt den Abschluss nicht.
	change.PendingChange.Occurrences = 2
	monthly, err := checks.Preview(ctx, CheckRequest{CutoffDate: "2026-01-31", PeriodType: "month"})
	if err != nil {
		t.Fatalf("Monatslauf: %v", err)
	}
	for _, finding := range monthly.Findings {
		if finding.Rule == domain.CheckRuleSizeClassChange {
			t.Error("der Monatslauf soll die Größenklasse nicht beurteilen")
		}
	}
}

// Der Prüflauf beanstandet den fehlenden Leistungsnachweis erst ab dem Tag, an
// dem die Grenze gesetzt wurde.
//
// Eine Festlegung des internen Kontrollsystems gilt ab dem Tag, an dem sie
// getroffen wurde. Rückwirkend angewandt meldete der erste Abschluss danach
// jeden Altbeleg — aus Zeiträumen, die längst festgeschrieben sind, und für
// eine Prüfung, die damals niemand verlangt hat.
func TestServiceProofFindingStartsAtTheDayTheThresholdWasSet(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	settings := repository.NewSettingsRepository(env.db)

	old := env.fileWithAmount(t, "alt.pdf", 500_000, "2026-02-01")
	env.setReceivedAt(t, old.ID, "2026-02-01")
	fresh := env.fileWithAmount(t, "neu.pdf", 500_000, "2026-03-10")
	env.setReceivedAt(t, fresh.ID, "2026-03-10")

	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Unternehmensdaten: %v", err)
	}
	cfg.InvoiceCheckThreshold = 100_000
	cfg.InvoiceCheckSince = "2026-03-01"
	if err := settings.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatalf("Grenze setzen: %v", err)
	}

	run := runChecks(t, env.checksOn(t, "2026-04-01"), "2026-03-31")
	found := findingsFor(run, domain.CheckRuleServiceProofMissing)
	if len(found) != 1 {
		t.Fatalf("erwartet einen Befund (nur der Beleg ab dem Stichtag), erhalten %d: %+v", len(found), found)
	}
	if found[0].ObjectID != fmt.Sprintf("%d", fresh.ID) {
		t.Errorf("der Befund zeigt auf Beleg %s, erwartet %d", found[0].ObjectID, fresh.ID)
	}

	// Der Tag wird nicht verschoben, wenn die Grenze später geändert wird: die
	// Belege davor sind dadurch nicht nachträglich mangelhaft geworden.
	again, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Unternehmensdaten: %v", err)
	}
	if again.InvoiceCheckSince != "2026-03-01" {
		t.Fatalf("der Stichtag lautet %q, erwartet 2026-03-01", again.InvoiceCheckSince)
	}
	again.InvoiceCheckThreshold = 200_000
	again.InvoiceCheckSince = ""
	if err := settings.UpdateCompanySettings(ctx, again); err != nil {
		t.Fatalf("Grenze ändern: %v", err)
	}
	third, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Unternehmensdaten: %v", err)
	}
	if third.InvoiceCheckSince != "2026-03-01" {
		t.Errorf("der Stichtag wurde auf %q verschoben, erwartet 2026-03-01", third.InvoiceCheckSince)
	}
}

// Scheitert die Abschlussbuchung, bleibt kein unversiegelter Eigenbeleg zurück.
//
// Der Eigenbeleg entsteht vor der Buchung — sein Hash geht in den Hash der
// Buchung ein, nachträglich gesetzt wäre er nicht mehr gedeckt. Scheitert die
// Buchung danach, läge im Belegspeicher ein Beleg, der auf nichts verweist: der
// Prüflauf meldete ihn dauerhaft als ungebucht, und niemand wüsste, wozu er
// gehört. Gelöscht wird er nicht — die GoBD kennen kein Löschen, sondern das
// Verwerfen mit Grund.
func TestFailedClosingBookingLeavesNoOpenSelfIssuedReceipt(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	m.rate(t, "2026-12", 3, 15_000)

	before, err := env.receipts.List(ctx, domain.ReceiptStatusFiled)
	if err != nil {
		t.Fatalf("Belege lesen: %v", err)
	}

	// Der Zeitraum ist festgeschrieben: die Buchung zum 31.12. geht nicht mehr
	// durch, der Eigenbeleg ist zu diesem Zeitpunkt aber schon abgelegt.
	env.commitUntil(t, "2026-12-31")

	if _, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionUncertainLiability,
		Text: "Rückbauverpflichtung", Amount: 1_000_000, ExpectedOn: "2029-12-30",
		Reason: "Mietvertrag § 12: Rückbau der Einbauten bei Auszug",
	}); err == nil {
		t.Fatal("in einen festgeschriebenen Zeitraum darf nicht gebucht werden")
	}

	after, err := env.receipts.List(ctx, domain.ReceiptStatusFiled)
	if err != nil {
		t.Fatalf("Belege lesen: %v", err)
	}
	if len(after) != len(before) {
		t.Errorf("nach der gescheiterten Buchung liegen %d offene Belege statt %d: %+v",
			len(after), len(before), after)
	}

	// Der Beleg ist nicht verschwunden, sondern verworfen — mit Grund.
	all, err := env.receipts.List(ctx, "")
	if err != nil {
		t.Fatalf("Belege lesen: %v", err)
	}
	discarded := 0
	for _, r := range all {
		if r.Status != domain.ReceiptStatusDiscarded {
			continue
		}
		discarded++
		if r.DiscardReason == "" {
			t.Errorf("der Beleg %s wurde ohne Begründung verworfen", r.ReceiptNumber)
		}
	}
	if discarded != 1 {
		t.Errorf("erwartet einen verworfenen Eigenbeleg, gefunden %d", discarded)
	}
}
