package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// foundations builds the Gründungsdienst on the shared test environment.
func (e *testEnv) foundations(t *testing.T) *FoundationService {
	t.Helper()
	return NewFoundationService(
		repository.NewFoundationRepository(e.db),
		repository.NewAccountRepository(e.db),
		e.journalRepo,
		repository.NewSettingsRepository(e.db),
		e.journal,
		repository.NewAuditRepository(e.db),
		e.fiscalYear,
	)
}

// gmbh is the standard case: 25.000 €, zwei Gesellschafter im Verhältnis 60/40,
// je die Hälfte ihrer Einlage geleistet.
func gmbhFoundation() *domain.Foundation {
	return &domain.Foundation{
		NotarizedOn:  "2026-01-15",
		ShareCapital: 2_500_000,
		Shareholders: []domain.Shareholder{
			{Name: "Anna Bauer", ShareCapital: 1_500_000, PaidIn: 750_000, Kind: domain.ContributionCash},
			{Name: "Ben Conrad", ShareCapital: 1_000_000, PaidIn: 500_000, Kind: domain.ContributionCash},
		},
	}
}

func (e *testEnv) saveFoundation(t *testing.T, svc *FoundationService, f *domain.Foundation) *domain.Foundation {
	t.Helper()
	saved, err := svc.Save(context.Background(), f)
	if err != nil {
		t.Fatalf("Gründung konnte nicht gespeichert werden: %v", err)
	}
	return saved
}

// book writes a plain two-line entry, so a test can move the Reinvermögen.
func (e *testEnv) book(t *testing.T, date, text, debit, credit string, amount domain.Cents) {
	t.Helper()
	_, err := e.journal.Post(context.Background(), &domain.JournalEntry{
		BookingDate: date, DocumentDate: date, ServiceDateFrom: date, ServiceDateTo: date,
		Description: text, Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: debit, Amount: amount},
			{Side: domain.SideCredit, Account: credit, Amount: amount},
		},
	})
	if err != nil {
		t.Fatalf("Buchung %q: %v", text, err)
	}
}

func TestSaveFoundationRejectsMismatchedShares(t *testing.T) {
	env := newTestEnv(t)
	svc := env.foundations(t)

	f := gmbhFoundation()
	f.Shareholders[1].ShareCapital = 900_000 // Summe 24.000 statt 25.000

	_, err := svc.Save(context.Background(), f)
	if err == nil {
		t.Fatal("eine Gesellschafterliste, die das Stammkapital verfehlt, muss abgelehnt werden")
	}
	if !strings.Contains(err.Error(), "übereinstimmen") {
		t.Errorf("die Meldung nennt den Grund nicht: %v", err)
	}
}

func TestSaveFoundationRejectsCapitalBelowMinimum(t *testing.T) {
	env := newTestEnv(t)
	svc := env.foundations(t)

	f := gmbhFoundation()
	f.ShareCapital = 1_000_000
	f.Shareholders = []domain.Shareholder{{Name: "Anna Bauer", ShareCapital: 1_000_000, Kind: domain.ContributionCash}}

	if _, err := svc.Save(context.Background(), f); err == nil {
		t.Fatal("eine GmbH mit 10.000 € Stammkapital verstößt gegen § 5 Abs. 1 GmbHG")
	}
}

func TestSaveFoundationRejectsSacheinlageAtUG(t *testing.T) {
	env := newTestEnv(t)
	settings := repository.NewSettingsRepository(env.db)
	cfg, _ := settings.GetCompanySettings(context.Background())
	cfg.LegalForm = "UG (haftungsbeschränkt)"
	if err := settings.UpdateCompanySettings(context.Background(), cfg); err != nil {
		t.Fatalf("Rechtsform konnte nicht gesetzt werden: %v", err)
	}
	svc := env.foundations(t)

	f := &domain.Foundation{
		NotarizedOn:  "2026-01-15",
		ShareCapital: 100_000,
		Shareholders: []domain.Shareholder{
			{Name: "Anna Bauer", ShareCapital: 100_000, PaidIn: 100_000, Kind: domain.ContributionInKind},
		},
	}
	_, err := svc.Save(context.Background(), f)
	if err == nil {
		t.Fatal("§ 5a Abs. 2 Satz 2 GmbHG schließt Sacheinlagen bei der UG aus")
	}
	if !strings.Contains(err.Error(), "5a") {
		t.Errorf("die Meldung nennt die Fundstelle nicht: %v", err)
	}
}

func TestFoundationDoesNotApplyToPersonengesellschaft(t *testing.T) {
	env := newTestEnv(t)
	settings := repository.NewSettingsRepository(env.db)
	cfg, _ := settings.GetCompanySettings(context.Background())
	cfg.LegalForm = "GmbH & Co. KG"
	if err := settings.UpdateCompanySettings(context.Background(), cfg); err != nil {
		t.Fatalf("Rechtsform konnte nicht gesetzt werden: %v", err)
	}
	svc := env.foundations(t)

	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if state.Applies {
		t.Error("bei einer Personengesellschaft gibt es keinen Gründungsweg")
	}
	if _, err := svc.Save(context.Background(), gmbhFoundation()); err == nil {
		t.Error("eine Gründung darf dort nicht erfasst werden können")
	}
}

func TestAnmeldungCheckAgainstParagraph7(t *testing.T) {
	env := newTestEnv(t)
	svc := env.foundations(t)
	env.saveFoundation(t, svc, gmbhFoundation())

	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	// 12.500 € geleistet, verlangt sind 12.500 € (Hälfte des Mindeststammkapitals).
	if !state.Anmeldung.IsSatisfied {
		t.Errorf("die Anmeldung sollte möglich sein, Befunde: %v", state.Anmeldung.Findings)
	}
	if state.Anmeldung.RequiredPaidIn != 1_250_000 {
		t.Errorf("verlangt %s €, erwartet 12.500,00", state.Anmeldung.RequiredPaidIn)
	}

	// Ein Gesellschafter zahlt weniger als ein Viertel seines Anteils: der
	// Gesamtbetrag reicht dann zwar, die Viertelregel je Anteil aber nicht.
	f := gmbhFoundation()
	f.Shareholders[0].PaidIn = 1_250_000
	f.Shareholders[1].PaidIn = 0
	env.saveFoundation(t, svc, f)

	state, err = svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if state.Anmeldung.IsSatisfied {
		t.Error("ohne ein Viertel auf jeden Geschäftsanteil ist die Anmeldung unzulässig")
	}
	if len(state.Anmeldung.Findings) == 0 {
		t.Error("der Befund nennt nicht, was fehlt")
	}
}

func TestUnterbilanzIsZeroWhenNothingWasSpent(t *testing.T) {
	env := newTestEnv(t)
	svc := env.foundations(t)
	env.saveFoundation(t, svc, gmbhFoundation())

	if _, err := svc.BookPostings(context.Background()); err != nil {
		t.Fatalf("Gründungsbuchungen: %v", err)
	}

	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	u := state.Unterbilanz
	// Bank 12.500 plus offene Einlageforderung 12.500 ergeben genau das
	// Stammkapital. Die noch nicht geleistete Einlage ist keine Unterbilanz —
	// sie wird geschuldet, aber als Einlage, nicht als Vorbelastungshaftung.
	if u.NetAssets != 2_500_000 {
		t.Errorf("Reinvermögen %s €, erwartet 25.000,00", u.NetAssets)
	}
	if u.Amount != 0 {
		t.Errorf("Unterbilanz %s €, erwartet 0", u.Amount)
	}
	if u.IsFinal {
		t.Error("solange die Eintragung fehlt, ist die Zahl vorläufig")
	}
}

func TestUnterbilanzGrowsWithGruendungsaufwand(t *testing.T) {
	env := newTestEnv(t)
	svc := env.foundations(t)
	env.saveFoundation(t, svc, gmbhFoundation())
	if _, err := svc.BookPostings(context.Background()); err != nil {
		t.Fatalf("Gründungsbuchungen: %v", err)
	}

	// Notarrechnung, aus der Bank bezahlt. § 248 Abs. 1 Nr. 1 HGB verbietet die
	// Aktivierung, der Betrag mindert das Reinvermögen also sofort.
	env.book(t, "2026-01-20", "Notarkosten Gründung", "6825", "1800", 300_000)

	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	u := state.Unterbilanz
	if u.NetAssets != 2_200_000 {
		t.Errorf("Reinvermögen %s €, erwartet 22.000,00", u.NetAssets)
	}
	if u.Shortfall != 300_000 || u.Amount != 300_000 {
		t.Errorf("Unterdeckung %s €, Haftung %s € — erwartet je 3.000,00", u.Shortfall, u.Amount)
	}

	// 60/40 auf zwei Gesellschafter.
	if len(u.Shares) != 2 {
		t.Fatalf("%d Anteile, erwartet 2", len(u.Shares))
	}
	if u.Shares[0].Amount != 180_000 || u.Shares[1].Amount != 120_000 {
		t.Errorf("Aufteilung %s / %s €, erwartet 1.800,00 / 1.200,00",
			u.Shares[0].Amount, u.Shares[1].Amount)
	}
	var sum domain.Cents
	for _, s := range u.Shares {
		sum += s.Amount
	}
	if sum != u.Amount {
		t.Errorf("die Anteile ergeben %s €, die Haftung beträgt %s €", sum, u.Amount)
	}
}

func TestUnterbilanzIsReducedByTheSatzungsklausel(t *testing.T) {
	env := newTestEnv(t)
	svc := env.foundations(t)
	f := gmbhFoundation()
	f.FoundationCostCap = 250_000 // 2.500 € Gründungsaufwand laut Satzung
	env.saveFoundation(t, svc, f)
	if _, err := svc.BookPostings(context.Background()); err != nil {
		t.Fatalf("Gründungsbuchungen: %v", err)
	}
	env.book(t, "2026-01-20", "Notarkosten Gründung", "6825", "1800", 300_000)

	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	u := state.Unterbilanz
	if u.Shortfall != 300_000 {
		t.Errorf("Unterdeckung %s €, erwartet 3.000,00", u.Shortfall)
	}
	if u.Covered != 250_000 {
		t.Errorf("gedeckt %s €, erwartet 2.500,00", u.Covered)
	}
	if u.Amount != 50_000 {
		t.Errorf("Haftung %s €, erwartet 500,00", u.Amount)
	}
}

func TestUnterbilanzCountsDebts(t *testing.T) {
	env := newTestEnv(t)
	svc := env.foundations(t)
	env.saveFoundation(t, svc, gmbhFoundation())
	if _, err := svc.BookPostings(context.Background()); err != nil {
		t.Fatalf("Gründungsbuchungen: %v", err)
	}

	// Eine offene Notarrechnung belastet genauso wie eine bezahlte: sie ist eine
	// Schuld der Vorgesellschaft, gleich ob das Geld schon geflossen ist.
	env.book(t, "2026-01-20", "Notarrechnung offen", "6825", "3500", 300_000)

	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if state.Unterbilanz.Liabilities != 300_000 {
		t.Errorf("Schulden %s €, erwartet 3.000,00", state.Unterbilanz.Liabilities)
	}
	if state.Unterbilanz.Amount != 300_000 {
		t.Errorf("Haftung %s €, erwartet 3.000,00", state.Unterbilanz.Amount)
	}
}

func TestRegisterEndsTheVorgesellschaft(t *testing.T) {
	env := newTestEnv(t)
	svc := env.foundations(t)
	env.saveFoundation(t, svc, gmbhFoundation())
	if _, err := svc.BookPostings(context.Background()); err != nil {
		t.Fatalf("Gründungsbuchungen: %v", err)
	}
	env.book(t, "2026-01-20", "Notarkosten Gründung", "6825", "1800", 300_000)

	// Eine Buchung nach dem Eintragungstag darf die festgestellte Unterbilanz
	// nicht mehr verändern.
	env.book(t, "2026-03-01", "Miete März", "6310", "1800", 100_000)

	if _, err := svc.Register(context.Background(), "2026-02-10", "Amtsgericht München", "HRB 123456"); err != nil {
		t.Fatalf("Eintragung: %v", err)
	}

	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if state.Stage != domain.FoundationStageEingetragen {
		t.Errorf("Phase %q, erwartet eingetragen", state.Stage)
	}
	u := state.Unterbilanz
	if !u.IsFinal || u.AsOf != "2026-02-10" {
		t.Errorf("Stichtag %q (endgültig: %v), erwartet den 2026-02-10 als endgültig", u.AsOf, u.IsFinal)
	}
	if u.Amount != 300_000 {
		t.Errorf("Haftung %s €, erwartet 3.000,00 — die Miete vom März zählt nicht mehr mit", u.Amount)
	}

	// Die Anmeldung zum Handelsregister ist mit der Eintragung erledigt.
	for _, d := range state.Duties {
		if d.Key == "handelsregister" && !d.IsDone {
			t.Error("die Anmeldung steht nach der Eintragung noch als offen")
		}
	}

	if _, err := svc.Register(context.Background(), "2026-03-01", "AG München", "HRB 2"); err == nil {
		t.Error("eine zweite Eintragung muss abgelehnt werden")
	}
}

func TestRegisterRejectsDateBeforeBeurkundung(t *testing.T) {
	env := newTestEnv(t)
	svc := env.foundations(t)
	env.saveFoundation(t, svc, gmbhFoundation())

	if _, err := svc.Register(context.Background(), "2026-01-01", "AG München", "HRB 1"); err == nil {
		t.Fatal("die Eintragung kann nicht vor der Beurkundung liegen")
	}
}

func TestPostingsAreOfferedOnceAndUseTheOpeningSource(t *testing.T) {
	env := newTestEnv(t)
	svc := env.foundations(t)
	env.saveFoundation(t, svc, gmbhFoundation())

	preview, err := svc.PreviewPostings(context.Background())
	if err != nil {
		t.Fatalf("PreviewPostings: %v", err)
	}
	// Zeichnung plus zwei Einzahlungen.
	if len(preview.Postings) != 3 {
		t.Fatalf("%d Buchungsvorschläge, erwartet 3", len(preview.Postings))
	}
	if preview.Postings[0].Lines[0].Account != domain.AccountAusstehendeEinlagenGeford ||
		preview.Postings[0].Lines[1].Account != domain.AccountGezeichnetesKapital {
		t.Error("die Zeichnung läuft über 1298 an 2900")
	}

	created, err := svc.BookPostings(context.Background())
	if err != nil {
		t.Fatalf("BookPostings: %v", err)
	}
	for _, e := range created {
		if e.Source != domain.EntrySourceOpening {
			t.Errorf("Buchung %s trägt die Quelle %q, erwartet opening", e.EntryNumber, e.Source)
		}
	}

	if _, err := svc.BookPostings(context.Background()); err == nil {
		t.Error("die Gründungsbuchungen dürfen nur einmal entstehen")
	}
	preview, err = svc.PreviewPostings(context.Background())
	if err != nil {
		t.Fatalf("PreviewPostings nach dem Buchen: %v", err)
	}
	if !preview.AlreadyBooked {
		t.Error("die Vorschau meldet nicht, dass bereits gebucht wurde")
	}
}

func TestPostingsSkipSacheinlage(t *testing.T) {
	env := newTestEnv(t)
	svc := env.foundations(t)
	f := gmbhFoundation()
	f.Shareholders[1].Kind = domain.ContributionInKind
	f.Shareholders[1].PaidIn = 1_000_000
	env.saveFoundation(t, svc, f)

	preview, err := svc.PreviewPostings(context.Background())
	if err != nil {
		t.Fatalf("PreviewPostings: %v", err)
	}
	if len(preview.Skipped) != 1 {
		t.Fatalf("%d übergangene Einlagen, erwartet 1", len(preview.Skipped))
	}
	// Zeichnung plus die eine Bareinlage.
	if len(preview.Postings) != 2 {
		t.Errorf("%d Buchungsvorschläge, erwartet 2", len(preview.Postings))
	}
}

func TestCompleteDutyIsPersistedWithItsDate(t *testing.T) {
	env := newTestEnv(t)
	svc := env.foundations(t)
	env.saveFoundation(t, svc, gmbhFoundation())

	if err := svc.CompleteDuty(context.Background(), "fragebogen", "2026-02-01", "über Mein ELSTER"); err != nil {
		t.Fatalf("CompleteDuty: %v", err)
	}
	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	found := false
	for _, d := range state.Duties {
		if d.Key == "fragebogen" {
			found = true
			if !d.IsDone || d.DoneOn != "2026-02-01" {
				t.Errorf("der Fragebogen steht als %v am %q", d.IsDone, d.DoneOn)
			}
		}
	}
	if !found {
		t.Fatal("der Fragebogen fehlt in der Fristenliste")
	}

	// Ein zweites Erledigen mit anderem Datum ersetzt das erste, statt eine
	// zweite Antwort daneben zu legen.
	if err := svc.CompleteDuty(context.Background(), "fragebogen", "2026-02-05", ""); err != nil {
		t.Fatalf("CompleteDuty erneut: %v", err)
	}
	tasks, err := repository.NewFoundationRepository(env.db).Tasks(context.Background(), 1)
	if err != nil {
		t.Fatalf("Tasks: %v", err)
	}
	count := 0
	for _, task := range tasks {
		if task.Key == "fragebogen" {
			count++
			if task.DoneOn != "2026-02-05" {
				t.Errorf("Datum %q, erwartet 2026-02-05", task.DoneOn)
			}
		}
	}
	if count != 1 {
		t.Errorf("%d Einträge zum Fragebogen, erwartet 1", count)
	}

	if err := svc.CompleteDuty(context.Background(), "fragebogen", "", ""); err != nil {
		t.Fatalf("CompleteDuty zurücknehmen: %v", err)
	}
	state, _ = svc.GetState(context.Background())
	for _, d := range state.Duties {
		if d.Key == "fragebogen" && d.IsDone {
			t.Error("der zurückgenommene Fragebogen steht weiter als erledigt")
		}
	}
}

func TestSaveFoundationReplacesShareholderList(t *testing.T) {
	env := newTestEnv(t)
	svc := env.foundations(t)
	env.saveFoundation(t, svc, gmbhFoundation())

	// Ein Gesellschafter übernimmt alles.
	f := &domain.Foundation{
		NotarizedOn:  "2026-01-15",
		ShareCapital: 2_500_000,
		Shareholders: []domain.Shareholder{
			{Name: "Anna Bauer", ShareCapital: 2_500_000, PaidIn: 1_250_000, Kind: domain.ContributionCash},
		},
	}
	env.saveFoundation(t, svc, f)

	state, err := svc.GetState(context.Background())
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if len(state.Foundation.Shareholders) != 1 {
		t.Errorf("%d Gesellschafter, erwartet 1 — die alte Liste blieb stehen",
			len(state.Foundation.Shareholders))
	}
	if state.Foundation.Shareholders[0].Name != "Anna Bauer" {
		t.Errorf("Gesellschafter %q", state.Foundation.Shareholders[0].Name)
	}
}
