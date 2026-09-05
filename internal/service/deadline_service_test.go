package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func (e *testEnv) deadlines(t *testing.T) *DeadlineService {
	t.Helper()
	return NewDeadlineService(
		e.vatReturns(t),
		e.zmReturns(t),
		repository.NewSettingsRepository(e.db),
		repository.NewFestschreibungRepository(e.db),
		repository.NewDeadlineRepository(e.db),
		repository.NewAuditRepository(e.db),
		e.fiscalYear,
	)
}

func deadlineByKey(list []domain.Deadline, key string) (domain.Deadline, bool) {
	for _, d := range list {
		if d.Key == key {
			return d, true
		}
	}
	return domain.Deadline{}, false
}

// Die Fristen kommen aus dem Backend und tragen ihre Norm. Vorher rechnete die
// Ansicht sie selbst.
func TestDeadlinesPerPeriodType(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	quarterly, err := env.deadlines(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	q1, ok := deadlineByKey(quarterly, "ustva.2026-Q1")
	if !ok {
		t.Fatalf("die Voranmeldung für Q1 fehlt")
	}
	if q1.DueDate != "2026-04-10" {
		t.Errorf("Q1 fällig am %s, erwartet 2026-04-10", q1.DueDate)
	}
	if q1.Reference != "§ 18 Abs. 1 UStG" {
		t.Errorf("Norm = %q", q1.Reference)
	}
	if _, ok := deadlineByKey(quarterly, "ustva.2026-01"); ok {
		t.Error("bei vierteljährlicher Abgabe gibt es keine Monatstermine")
	}

	env.setSettings(t, func(c *domain.CompanySettings) { c.VatPeriod = "month" })
	monthly, err := env.deadlines(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	january, ok := deadlineByKey(monthly, "ustva.2026-01")
	if !ok {
		t.Fatalf("die Voranmeldung für Januar fehlt")
	}
	if january.DueDate != "2026-02-10" {
		t.Errorf("Januar fällig am %s, erwartet 2026-02-10", january.DueDate)
	}
}

// Die Wochenendregel des § 108 Abs. 3 AO: der 10. Mai 2026 ist ein Sonntag, die
// Frist läuft am Montag ab. Feiertage bleiben bewusst außen vor — sie sind
// Landesrecht.
func TestDeadlineMovesOffWeekend(t *testing.T) {
	env := newTestEnv(t)
	env.setSettings(t, func(c *domain.CompanySettings) { c.VatPeriod = "month" })

	list, err := env.deadlines(t).Deadlines(context.Background(), 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	april, ok := deadlineByKey(list, "ustva.2026-04")
	if !ok {
		t.Fatal("die Voranmeldung für April fehlt")
	}
	if april.DueDate != "2026-05-11" {
		t.Errorf("April fällig am %s, erwartet 2026-05-11 (Montag nach dem 10.)", april.DueDate)
	}
}

// Die Dauerfristverlängerung verschiebt jeden Termin um einen Monat und bringt
// beim Monatszahler die Sondervorauszahlung mit — ihm wird sie nur gegen diese
// Vorauszahlung gewährt (§ 47 Abs. 1 UStDV).
func TestDeadlinesWithPermanentExtension(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.setSettings(t, func(c *domain.CompanySettings) { c.VatPeriod = "month" })

	without, err := env.deadlines(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	if _, ok := deadlineByKey(without, "sondervorauszahlung.2026"); ok {
		t.Error("ohne Dauerfristverlängerung gibt es keine Sondervorauszahlung")
	}

	env.setSettings(t, func(c *domain.CompanySettings) {
		c.PermanentExtension = true
		c.SpecialPrepayment = 100000
	})
	with, err := env.deadlines(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	january, _ := deadlineByKey(with, "ustva.2026-01")
	if january.DueDate != "2026-03-10" {
		t.Errorf("Januar mit Dauerfrist fällig am %s, erwartet 2026-03-10", january.DueDate)
	}
	prepayment, ok := deadlineByKey(with, "sondervorauszahlung.2026")
	if !ok {
		t.Fatal("die Sondervorauszahlung fehlt")
	}
	if prepayment.DueDate != "2026-02-10" {
		t.Errorf("Sondervorauszahlung fällig am %s, erwartet 2026-02-10", prepayment.DueDate)
	}
}

// Der Vierteljahreszahler bekommt die Dauerfristverlängerung ohne
// Sondervorauszahlung.
//
// § 47 Abs. 1 UStDV verlangt sie nur von den Unternehmern, die monatlich
// voranmelden; § 46 UStDV gewährt die Verlängerung dem Quartalszahler ohne
// Gegenleistung. Ein Termin „Sondervorauszahlung anmelden und zahlen" auf seiner
// Fristenliste wäre eine erfundene Pflicht — die Verschiebung der Voranmeldung
// selbst bleibt.
func TestQuarterlyFilerHasNoSpecialPrepaymentDeadline(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.setSettings(t, func(c *domain.CompanySettings) {
		c.VatPeriod = "quarter"
		c.PermanentExtension = true
		c.SpecialPrepayment = 100000
	})

	deadlines, err := env.deadlines(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	if _, ok := deadlineByKey(deadlines, "sondervorauszahlung.2026"); ok {
		t.Error("der Vierteljahreszahler schuldet keine Sondervorauszahlung (§ 47 Abs. 1 UStDV)")
	}
	q1, ok := deadlineByKey(deadlines, "ustva.2026-Q1")
	if !ok {
		t.Fatal("die Voranmeldung für Q1 fehlt")
	}
	if q1.DueDate != "2026-05-11" {
		// Der 10. Mai 2026 ist ein Sonntag.
		t.Errorf("Q1 mit Dauerfrist fällig am %s, erwartet 2026-05-11 — die Verlängerung gilt auch ohne "+
			"Sondervorauszahlung (§ 46 UStDV)", q1.DueDate)
	}
}

// Die Frist zur Festschreibung folgt der Nachfrist aus den Einstellungen.
//
// Fristenliste und Prüflauf müssen denselben Tag nennen: sonst führt die Liste
// einen Monat als offen, den der Prüfbericht unter commit_overdue noch gar nicht
// anmahnt.
func TestCommitDeadlineFollowsTheGraceSetting(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	base, err := env.deadlines(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	january, ok := deadlineByKey(base, "festschreibung.2026-01")
	if !ok {
		t.Fatal("die Festschreibung des Januars fehlt")
	}
	if january.DueDate != "2026-02-28" {
		t.Errorf("ohne Nachfrist fällig am %s, erwartet 2026-02-28", january.DueDate)
	}

	env.setSettings(t, func(c *domain.CompanySettings) { c.CommitGraceDays = 10 })
	extended, err := env.deadlines(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	january, _ = deadlineByKey(extended, "festschreibung.2026-01")
	if january.DueDate != "2026-03-10" {
		t.Errorf("mit zehn Tagen Nachfrist fällig am %s, erwartet 2026-03-10", january.DueDate)
	}

	// Derselbe Tag entscheidet im Prüflauf über commit_overdue.
	svc := env.checksOn(t, "2026-03-10")
	run, err := svc.Preview(ctx, CheckRequest{CutoffDate: "2026-03-10"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	for _, f := range run.Findings {
		if f.Rule == domain.CheckRuleCommitOverdue {
			t.Errorf("am Fälligkeitstag ist die Festschreibung noch nicht überfällig: %s", f.Message)
		}
	}
	run, err = svc.Preview(ctx, CheckRequest{CutoffDate: "2026-03-11"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	found := false
	for _, f := range run.Findings {
		if f.Rule == domain.CheckRuleCommitOverdue {
			found = true
		}
	}
	if !found {
		t.Error("einen Tag nach der Nachfrist meldet der Prüflauf die Festschreibung als überfällig")
	}
}

// „Erledigt" ergibt sich aus den Daten: eine übermittelte Voranmeldung ist
// abgegeben, ein festgeschriebener Monat ist festgeschrieben. Ein Haken daneben
// könnte nur widersprechen.
func TestDeadlineDoneFollowsFromTheData(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-02-10", 100000)

	before, err := env.deadlines(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	q1, _ := deadlineByKey(before, "ustva.2026-Q1")
	if q1.IsDone {
		t.Error("ohne übermittelte Anmeldung ist der Termin offen")
	}
	january, _ := deadlineByKey(before, "festschreibung.2026-01")
	if january.IsDone {
		t.Error("ohne Festschreibung ist der Termin offen")
	}

	env.commitUntil(t, "2026-03-31")
	vat := env.vatReturns(t)
	saved, err := vat.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Anmeldung: %v", err)
	}
	if _, err := vat.ConfirmSubmitted(ctx, saved.ID, "2026-04-09", "TT-1", ""); err != nil {
		t.Fatalf("bestätigen: %v", err)
	}

	after, err := env.deadlines(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	q1, _ = deadlineByKey(after, "ustva.2026-Q1")
	if !q1.IsDone || q1.DoneOn != "2026-04-09" {
		t.Errorf("die übermittelte Anmeldung ist erledigt am %q (erledigt=%v), erwartet 2026-04-09", q1.DoneOn, q1.IsDone)
	}
	january, _ = deadlineByKey(after, "festschreibung.2026-01")
	if !january.IsDone {
		t.Error("der festgeschriebene Januar ist erledigt")
	}
	march, _ := deadlineByKey(after, "festschreibung.2026-03")
	if !march.IsDone {
		t.Error("der März ist bis zum 31.03. festgeschrieben und damit erledigt")
	}
	april, _ := deadlineByKey(after, "festschreibung.2026-04")
	if april.IsDone {
		t.Error("der April ist nicht festgeschrieben")
	}
}

// Was Buchfink nicht sieht, bleibt ein Haken von Hand — aber in der Datenbank
// und nicht im Browser.
func TestManualDeadlineMarkIsStored(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.deadlines(t)

	list, err := svc.Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	annual, ok := deadlineByKey(list, "ust.jahreserklaerung.2026")
	if !ok {
		t.Fatal("die Umsatzsteuer-Jahreserklärung fehlt")
	}
	if annual.IsDone {
		t.Error("sie ist zunächst offen")
	}
	if annual.DueDate != "2027-08-02" {
		// Der 31. Juli 2027 ist ein Samstag.
		t.Errorf("Jahreserklärung fällig am %s, erwartet 2027-08-02", annual.DueDate)
	}

	if err := svc.MarkDone(ctx, "ust.jahreserklaerung.2026", "2027-05-04"); err != nil {
		t.Fatalf("abhaken: %v", err)
	}

	// Ein neuer Dienst auf derselben Datenbank sieht den Haken — er steht nicht
	// im Speicher, sondern beim Mandanten.
	list, err = env.deadlines(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	annual, _ = deadlineByKey(list, "ust.jahreserklaerung.2026")
	if !annual.IsDone || annual.DoneOn != "2027-05-04" {
		t.Errorf("der Haken = %v am %q, erwartet erledigt am 2027-05-04", annual.IsDone, annual.DoneOn)
	}
}

// Die Zusammenfassende Meldung erscheint nur, wenn es innergemeinschaftliche
// Umsätze gibt — ein Termin, der nichts verlangt, gehört nicht in die Liste.
func TestZMDeadlineOnlyWithIntraCommunityTurnover(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	list, err := env.deadlines(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	if _, ok := deadlineByKey(list, "zm.2026-Q1"); ok {
		t.Error("ohne ig. Umsätze gibt es keine Meldung")
	}

	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euInvoice(t, customer.ID, "2026-02-10", 500000, domain.TaxTreatmentIntraCommunitySupply)

	list, err = env.deadlines(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	zm, ok := deadlineByKey(list, "zm.2026-Q1")
	if !ok {
		t.Fatal("mit ig. Lieferung gehört die Meldung in die Liste")
	}
	if zm.DueDate != "2026-04-27" {
		t.Errorf("ZM Q1 fällig am %s, erwartet 2026-04-27", zm.DueDate)
	}
	if zm.Reference != "§ 18a Abs. 1 UStG" {
		t.Errorf("Norm = %q", zm.Reference)
	}
}
