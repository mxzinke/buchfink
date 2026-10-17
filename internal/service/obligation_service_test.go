package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// obligationClock ist der Stichtag der Kalendertests.
var obligationClock = time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)

func (e *testEnv) obligations(t *testing.T) *ObligationService {
	t.Helper()
	svc := NewObligationService(
		repository.NewSettingsRepository(e.db),
		repository.NewObligationRepository(e.db),
		repository.NewAuditRepository(e.db),
	)
	svc.SetVatSources(e.vatReturns(t), repository.NewVatReturnRepository(e.db))
	svc.SetZMSources(e.zmReturns(t), repository.NewZMReturnRepository(e.db))
	svc.SetCommitments(repository.NewFestschreibungRepository(e.db))
	svc.SetFiscalYearSource(repository.NewFiscalYearRepository(e.db))
	svc.SetSizeSource(e.statements(t))
	svc.SetJournalSource(e.journalRepo)
	svc.SetClock(func() time.Time { return obligationClock })
	return svc
}

func deadlineByKey(list []domain.Deadline, key string) (domain.Deadline, bool) {
	for _, d := range list {
		if d.Key == key {
			return d, true
		}
	}
	return domain.Deadline{}, false
}

func obligationByKey(list []domain.Obligation, key string) (domain.Obligation, bool) {
	for _, o := range list {
		if o.Key == key {
			return o, true
		}
	}
	return domain.Obligation{}, false
}

func listObligations(t *testing.T, svc *ObligationService, q ObligationQuery) []domain.Obligation {
	t.Helper()
	list, err := svc.List(context.Background(), q)
	if err != nil {
		t.Fatalf("Pflichten: %v", err)
	}
	return list
}

var (
	appCtx   = actor.WithChannel(context.Background(), actor.ChannelUI)
	agentCtx = func() context.Context {
		c, _ := actor.Agent("claude-desktop")
		return actor.WithChannel(context.Background(), c)
	}()
)

// Die Voranmeldungen folgen dem Zeitraum aus den Unternehmensdaten.
func TestObligationsPerVatPeriodType(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	quarterly, err := env.obligations(t).Deadlines(ctx, 2026)
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
	if !strings.Contains(q1.Reference, "§ 18 Abs. 1") {
		t.Errorf("Norm = %q", q1.Reference)
	}
	if _, ok := deadlineByKey(quarterly, "ustva.2026-01"); ok {
		t.Error("bei vierteljährlicher Abgabe gibt es keine Monatstermine")
	}

	env.setSettings(t, func(c *domain.CompanySettings) { c.VatPeriod = "month" })
	monthly, err := env.obligations(t).Deadlines(ctx, 2026)
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

// § 108 Abs. 3 AO: der 10. Mai 2026 ist ein Sonntag, die Frist läuft am Montag
// ab; der 10. Mai 2029 ist Christi Himmelfahrt.
func TestObligationMovesToTheNextWorkday(t *testing.T) {
	env := newTestEnv(t)
	env.setSettings(t, func(c *domain.CompanySettings) { c.VatPeriod = "month" })
	svc := env.obligations(t)

	list := listObligations(t, svc, ObligationQuery{Kinds: []domain.ObligationKind{domain.ObligationVatReturn},
		From: "2026-05-01", To: "2029-06-30"})
	april, ok := obligationByKey(list, "ustva.2026-04")
	if !ok {
		t.Fatal("die Voranmeldung für April fehlt")
	}
	if april.StatutoryDate != "2026-05-10" || april.DueDate != "2026-05-11" || april.ShiftReason != "Sonntag" {
		t.Errorf("April 2026: gesetzlich %s, fällig %s (%s)", april.StatutoryDate, april.DueDate, april.ShiftReason)
	}
	april29, ok := obligationByKey(list, "ustva.2029-04")
	if !ok {
		t.Fatal("die Voranmeldung für April 2029 fehlt")
	}
	if april29.DueDate != "2029-05-11" || april29.ShiftReason != "Christi Himmelfahrt" {
		t.Errorf("April 2029 fällig am %s (%s), erwartet 2029-05-11 nach Christi Himmelfahrt",
			april29.DueDate, april29.ShiftReason)
	}
}

// Ein Landesfeiertag verschiebt nur im Land am Sitz: der 01.11.2028 ist in
// Bayern Allerheiligen, in Sachsen ein Werktag. Die Monatsfrist ab dem
// 01.10.2028 endet am 01.11.2028 (§ 188 Abs. 2 BGB).
func TestStateHolidayShiftsOnlyInItsState(t *testing.T) {
	for state, want := range map[string]string{"BY": "2028-11-02", "SN": "2028-11-01"} {
		env := newTestEnv(t)
		env.setSettings(t, func(c *domain.CompanySettings) {
			c.FederalState, c.FoundedOn = state, "2028-10-01"
			c.TaxNumber = map[string]string{"SN": "201/123/12340", "BY": "181/815/08155"}[state]
		})
		o, err := env.obligations(t).Get(context.Background(), "steuerliche-erfassung.gruendung", "")
		if err != nil {
			t.Fatalf("%s: %v", state, err)
		}
		if o.StatutoryDate != "2028-11-01" || o.DueDate != want {
			t.Errorf("%s: gesetzlich %s, fällig %s — erwartet %s", state, o.StatutoryDate, o.DueDate, want)
		}
	}
}

// Die Dauerfristverlängerung verschiebt jede Voranmeldung um einen Monat und
// bringt beim Monatszahler die Sondervorauszahlung mit (§ 47 Abs. 1 UStDV).
func TestObligationsWithPermanentExtension(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.setSettings(t, func(c *domain.CompanySettings) { c.VatPeriod = "month" })

	without, err := env.obligations(t).Deadlines(ctx, 2026)
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
	with, err := env.obligations(t).Deadlines(ctx, 2026)
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
	if prepayment.DueDate != "2026-02-10" || !prepayment.Confirmable {
		t.Errorf("Sondervorauszahlung fällig am %s (bestätigbar %v), erwartet 2026-02-10",
			prepayment.DueDate, prepayment.Confirmable)
	}

	// Die Zahlung auf 3830 erledigt sie.
	env.post(t, "2026-02-09", "3830", "1800", 100000)
	paid, err := env.obligations(t).Get(ctx, "sondervorauszahlung.2026", "")
	if err != nil {
		t.Fatalf("Sondervorauszahlung: %v", err)
	}
	if paid.Status != domain.ObligationDone || paid.Proof == nil || paid.Proof.Kind != domain.EvidenceBooking ||
		paid.Proof.Date != "2026-02-09" {
		t.Errorf("nach der Zahlung: %s %+v", paid.Status, paid.Proof)
	}
}

// Der Vierteljahreszahler bekommt die Dauerfristverlängerung ohne
// Sondervorauszahlung; die Verschiebung bleibt (§ 46 UStDV).
func TestQuarterlyFilerHasNoSpecialPrepayment(t *testing.T) {
	env := newTestEnv(t)
	env.setSettings(t, func(c *domain.CompanySettings) {
		c.VatPeriod = "quarter"
		c.PermanentExtension = true
		c.SpecialPrepayment = 100000
	})
	deadlines, err := env.obligations(t).Deadlines(context.Background(), 2026)
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
		t.Errorf("Q1 mit Dauerfrist fällig am %s, erwartet 2026-05-11", q1.DueDate)
	}
}

// Die Frist zur Festschreibung folgt der Nachfrist aus den Einstellungen, und
// der Prüflauf meldet denselben Tag. Sie verschiebt sich nicht: sie ist keine
// Frist der AO.
func TestCommitObligationFollowsTheGraceSetting(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	base, err := env.obligations(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	january, ok := deadlineByKey(base, "festschreibung.2026-01")
	if !ok {
		t.Fatal("die Festschreibung des Januars fehlt")
	}
	if january.DueDate != "2026-02-28" {
		t.Errorf("ohne Nachfrist fällig am %s, erwartet 2026-02-28 (ein Sonnabend, unverschoben)", january.DueDate)
	}

	env.setSettings(t, func(c *domain.CompanySettings) { c.CommitGraceDays = 10 })
	extended, err := env.obligations(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	january, _ = deadlineByKey(extended, "festschreibung.2026-01")
	if january.DueDate != "2026-03-10" {
		t.Errorf("mit zehn Tagen Nachfrist fällig am %s, erwartet 2026-03-10", january.DueDate)
	}

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
		found = found || f.Rule == domain.CheckRuleCommitOverdue
	}
	if !found {
		t.Error("einen Tag nach der Nachfrist meldet der Prüflauf die Festschreibung als überfällig")
	}
}

// „Erledigt" ergibt sich aus den Daten: eine übermittelte Voranmeldung mit
// ihrem Transferticket, ein festgeschriebener Monat.
func TestObligationDoneFollowsFromTheData(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-02-10", 100000)

	before, err := env.obligations(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	if q1, _ := deadlineByKey(before, "ustva.2026-Q1"); q1.IsDone || q1.Status != string(domain.ObligationOverdue) {
		t.Errorf("ohne übermittelte Anmeldung ist Q1 überfällig, erhalten %s", q1.Status)
	}
	if january, _ := deadlineByKey(before, "festschreibung.2026-01"); january.IsDone {
		t.Error("ohne Festschreibung ist der Januar offen")
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

	svc := env.obligations(t)
	q1, err := svc.Get(ctx, "ustva.2026-Q1", "")
	if err != nil {
		t.Fatalf("Q1: %v", err)
	}
	if q1.Status != domain.ObligationDone || q1.Proof == nil || q1.Proof.Date != "2026-04-09" ||
		q1.Proof.Reference != "TT-1" || !q1.Proof.Automatic {
		t.Errorf("die übermittelte Anmeldung: %s %+v", q1.Status, q1.Proof)
	}
	after, err := svc.Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	for key, done := range map[string]bool{
		"festschreibung.2026-01": true, "festschreibung.2026-03": true, "festschreibung.2026-04": false,
	} {
		if d, _ := deadlineByKey(after, key); d.IsDone != done {
			t.Errorf("%s erledigt = %v, erwartet %v", key, d.IsDone, done)
		}
	}
}

// Was Buchfink nicht sieht, bestätigt der Mensch mit Nachweis. Die Bestätigung
// steht beim Mandanten und im Protokoll.
func TestConfirmationIsStoredWithEvidence(t *testing.T) {
	env := newTestEnv(t)
	svc := env.obligations(t)

	annual, err := svc.Get(appCtx, "ust-erklaerung.2026", "")
	if err != nil {
		t.Fatalf("Umsatzsteuer-Jahreserklärung: %v", err)
	}
	if annual.Status != domain.ObligationOpen || annual.DueDate != "2027-08-02" || annual.StatutoryDate != "2027-07-31" {
		t.Errorf("Jahreserklärung 2026: %s, fällig %s (gesetzlich %s) — der 31.07.2027 ist ein Sonnabend",
			annual.Status, annual.DueDate, annual.StatutoryDate)
	}

	if _, err := svc.Confirm(appCtx, "ust-erklaerung.2026", ConfirmInput{
		Date: "2026-10-01", Kind: domain.EvidenceTransferTicket}); err == nil {
		t.Error("ein Transferticket ohne Nummer ist kein Nachweis")
	}
	if _, err := svc.Confirm(appCtx, "ust-erklaerung.2026", ConfirmInput{Date: "2026-10-18"}); err == nil {
		t.Error("ein Nachweis aus der Zukunft wird abgewiesen")
	}
	if _, err := svc.Confirm(appCtx, "ust-erklaerung.2026", ConfirmInput{
		Date: "2026-10-01", Kind: domain.EvidenceTransferTicket, Reference: "et-2026-ust", Note: "über Viking"}); err != nil {
		t.Fatalf("bestätigen: %v", err)
	}

	again, err := env.obligations(t).Get(appCtx, "ust-erklaerung.2026", "")
	if err != nil {
		t.Fatalf("erneut lesen: %v", err)
	}
	if again.Status != domain.ObligationDone || again.Proof == nil || again.Proof.Reference != "et-2026-ust" ||
		again.Proof.Automatic || again.Proof.Channel != "ui" {
		t.Errorf("nach der Bestätigung: %s %+v", again.Status, again.Proof)
	}
	entries, err := repository.NewAuditRepository(env.db).FindFiltered(context.Background(), 10,
		domain.AuditFilter{EntityType: "OBLIGATION", EntityID: "ust-erklaerung.2026"})
	if err != nil || len(entries) != 1 || !strings.Contains(entries[0].Details, "et-2026-ust") {
		t.Errorf("Protokoll: %+v, %v", entries, err)
	}
	if _, err := svc.Confirm(appCtx, "ust-erklaerung.2026", ConfirmInput{Date: "2026-10-01"}); err == nil {
		t.Error("eine erledigte Pflicht wird nicht ein zweites Mal bestätigt")
	}

	// Zurücknehmen braucht einen Grund und öffnet die Pflicht wieder.
	if _, err := svc.Reopen(appCtx, "ust-erklaerung.2026", " "); err == nil {
		t.Error("ohne Grund wird nichts zurückgenommen")
	}
	reopened, err := svc.Reopen(appCtx, "ust-erklaerung.2026", "falsches Jahr bestätigt")
	if err != nil {
		t.Fatalf("zurücknehmen: %v", err)
	}
	if reopened.Status != domain.ObligationOpen || reopened.Proof != nil {
		t.Errorf("nach dem Zurücknehmen: %s %+v", reopened.Status, reopened.Proof)
	}
}

// Die Zusammenfassende Meldung steht nur im Kalender, wenn es sonstige
// Leistungen ins übrige Gemeinschaftsgebiet gibt.
func TestZMObligationOnlyWithIntraCommunityTurnover(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	list, err := env.obligations(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	if _, ok := deadlineByKey(list, "zm.2026-Q1"); ok {
		t.Error("ohne ig. Umsätze gibt es keine Meldung")
	}

	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euService(t, customer.ID, "2026-02-10", 500000)

	list, err = env.obligations(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	zm, ok := deadlineByKey(list, "zm.2026-Q1")
	if !ok {
		t.Fatal("mit einer Leistung ins übrige Gemeinschaftsgebiet gehört die Meldung in den Kalender")
	}
	if zm.DueDate != "2026-04-27" {
		t.Errorf("ZM Q1 fällig am %s, erwartet 2026-04-27 (der 25. ist ein Sonnabend)", zm.DueDate)
	}
	if zm.Reference != "§ 18a Abs. 2 UStG" {
		t.Errorf("Norm = %q", zm.Reference)
	}
}

// Bestätigen und Entfallen gibt es nur, wo der Stand nicht aus den Daten kommt.
func TestManualDecisionsOnlyWhereTheDataCannotTell(t *testing.T) {
	env := newTestEnv(t)
	env.setSettings(t, func(c *domain.CompanySettings) {
		c.VatPeriod, c.PermanentExtension = "month", true
	})
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euService(t, customer.ID, "2026-02-10", 500000)
	svc := env.obligations(t)

	for _, key := range []string{"ustva.2026-01", "zm.2026-Q1", "festschreibung.2026-01", "aufstellung.2026"} {
		if _, err := svc.Confirm(appCtx, key, ConfirmInput{Date: "2026-05-04"}); err == nil ||
			!strings.Contains(err.Error(), "ergibt sich aus den Daten") {
			t.Errorf("%s ließ sich bestätigen: %v", key, err)
		}
		if _, err := svc.Waive(appCtx, key, "nicht nötig"); err == nil {
			t.Errorf("%s ließ sich als entfallen kennzeichnen", key)
		}
	}
	for _, key := range []string{"sondervorauszahlung.2026", "ust-erklaerung.2026", "e-bilanz.2026"} {
		if _, err := svc.Confirm(appCtx, key, ConfirmInput{Date: "2026-05-04"}); err != nil {
			t.Errorf("%s muss sich bestätigen lassen: %v", key, err)
		}
	}
	if _, err := svc.Confirm(appCtx, "ust-erklaerung.1999", ConfirmInput{Date: "2026-05-04"}); err == nil {
		t.Error("eine Pflicht, die es nicht gibt, lässt sich nicht bestätigen")
	}
}

// Ein Agent meldet „vorbereitet", nie „erledigt"; über die Fälligkeit hinaus
// bleibt die Pflicht überfällig, mit der Vorbereitung daneben.
func TestAgentMayOnlyPrepare(t *testing.T) {
	env := newTestEnv(t)
	svc := env.obligations(t)

	prepared, err := svc.MarkPrepared(agentCtx, "kst-erklaerung.2026", "Überleitung liegt vor")
	if err != nil {
		t.Fatalf("vorbereitet melden: %v", err)
	}
	if prepared.Status != domain.ObligationPrepared || prepared.Preparation.Channel != "agent:claude-desktop" ||
		prepared.Preparation.Note != "Überleitung liegt vor" {
		t.Errorf("vorbereitet: %s %+v", prepared.Status, prepared.Preparation)
	}
	for name, call := range map[string]func() error{
		"bestätigen": func() error {
			_, err := svc.Confirm(agentCtx, "kst-erklaerung.2026", ConfirmInput{Date: "2026-10-01"})
			return err
		},
		"entfallen":    func() error { _, err := svc.Waive(agentCtx, "kst-erklaerung.2026", "x"); return err },
		"zurücknehmen": func() error { _, err := svc.Reopen(agentCtx, "kst-erklaerung.2026", "x"); return err },
	} {
		if err := call(); err == nil || !strings.Contains(err.Error(), "vorbehalten") {
			t.Errorf("ein Agent darf nicht %s: %v", name, err)
		}
	}

	system := actor.WithChannel(context.Background(), actor.ChannelSystem)
	if _, err := svc.MarkPrepared(system, "gewst-erklaerung.2026", ""); err == nil {
		t.Error("ein Hintergrundlauf meldet nichts als vorbereitet")
	}

	late, err := svc.Get(agentCtx, "kst-erklaerung.2026", "2027-08-03")
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}
	if late.Status != domain.ObligationOverdue || late.Preparation == nil {
		t.Errorf("nach der Frist: %s %+v", late.Status, late.Preparation)
	}

	if _, err := svc.Confirm(appCtx, "kst-erklaerung.2026", ConfirmInput{Date: "2026-10-01"}); err != nil {
		t.Fatalf("bestätigen: %v", err)
	}
	if _, err := svc.MarkPrepared(agentCtx, "kst-erklaerung.2026", ""); err == nil {
		t.Error("eine erledigte Pflicht lässt sich nicht mehr vorbereitet melden")
	}
}

func TestWaiveNeedsAReason(t *testing.T) {
	env := newTestEnv(t)
	env.setSettings(t, func(c *domain.CompanySettings) { c.FoundedOn = "2026-03-15" })
	svc := env.obligations(t)
	if _, err := svc.Waive(appCtx, "gewerbeanmeldung.gruendung", ""); err == nil {
		t.Error("ohne Grund entfällt nichts")
	}
	waived, err := svc.Waive(appCtx, "gewerbeanmeldung.gruendung", "rein freiberufliche Softwareentwicklung")
	if err != nil {
		t.Fatalf("entfallen: %v", err)
	}
	if waived.Status != domain.ObligationWaived || waived.WaivedReason != "rein freiberufliche Softwareentwicklung" {
		t.Errorf("entfallen: %s %q", waived.Status, waived.WaivedReason)
	}
	deadlines, err := svc.Deadlines(context.Background(), 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	if d, _ := deadlineByKey(deadlines, "gewerbeanmeldung.gruendung"); !d.IsDone || d.Confirmable {
		t.Errorf("auf der Fristenseite: %+v", d)
	}
}

// Die Abgabefristen der Jahreserklärungen: § 149 Abs. 2 und 3 AO mit den
// Übergangsfristen des Art. 97 § 36 EGAO.
func TestAnnualReturnDeadlines(t *testing.T) {
	cases := []struct {
		year    int
		advisor bool
		want    string
	}{
		{2020, false, "2021-10-31"},
		{2020, true, "2022-08-31"},
		{2022, false, "2023-09-30"},
		{2022, true, "2024-07-31"},
		{2023, false, "2024-08-31"},
		{2023, true, "2025-05-31"},
		{2024, false, "2025-07-31"},
		{2024, true, "2026-04-30"},
		{2025, false, "2026-07-31"},
		{2025, true, "2027-02-28"},
		{2026, false, "2027-07-31"},
		{2026, true, "2028-02-29"},
	}
	for _, c := range cases {
		if got := annualReturnDeadline(c.year, c.advisor).Format(domain.DateLayout); got != c.want {
			t.Errorf("%d (Berater %v) = %s, erwartet %s", c.year, c.advisor, got, c.want)
		}
	}
}

// Ein Steuerberater verlängert die Frist nur für die Erklärungen, die er
// erstellt, und ab dem Jahr seiner Beauftragung.
func TestTaxAdvisorExtendsOnlyHisDeclarations(t *testing.T) {
	env := newTestEnv(t)
	env.setSettings(t, func(c *domain.CompanySettings) {
		c.TaxAdvisor = domain.TaxAdvisor{Engaged: true, SinceYear: 2026,
			Declarations: []string{domain.DeclarationCorporateTax, domain.DeclarationEBilanz}}
	})
	svc := env.obligations(t)
	list := listObligations(t, svc, ObligationQuery{FiscalYear: 2026})

	kst, _ := obligationByKey(list, "kst-erklaerung.2026")
	if kst.DueDate != "2028-02-29" || kst.Responsibility != domain.ResponsibilityTaxAdvisor {
		t.Errorf("KSt 2026 mit Berater: fällig %s, %s", kst.DueDate, kst.Responsibility)
	}
	if !containsHint(kst.Hints, "§ 149 Abs. 4 AO") || !containsHint(kst.Hints, "§ 152 Abs. 2 Nr. 1 AO") {
		t.Errorf("Hinweise zur Vorabanforderung und zum Verspätungszuschlag fehlen: %v", kst.Hints)
	}
	gewst, _ := obligationByKey(list, "gewst-erklaerung.2026")
	if gewst.DueDate != "2027-08-02" || gewst.Responsibility != domain.ResponsibilityAgentPrepares {
		t.Errorf("GewSt 2026 ohne Berater: fällig %s, %s", gewst.DueDate, gewst.Responsibility)
	}
	ebilanz, _ := obligationByKey(list, "e-bilanz.2026")
	if ebilanz.DueDate != kst.DueDate || ebilanz.Responsibility != domain.ResponsibilityTaxAdvisor {
		t.Errorf("E-Bilanz 2026 folgt der KSt: fällig %s, %s", ebilanz.DueDate, ebilanz.Responsibility)
	}
	kst25, err := svc.Get(context.Background(), "kst-erklaerung.2025", "")
	if err != nil {
		t.Fatalf("KSt 2025: %v", err)
	}
	if kst25.DueDate != "2026-07-31" || kst25.Status != domain.ObligationOverdue {
		t.Errorf("KSt 2025 vor der Beauftragung: fällig %s, %s", kst25.DueDate, kst25.Status)
	}
}

func containsHint(hints []string, part string) bool {
	for _, h := range hints {
		if strings.Contains(h, part) {
			return true
		}
	}
	return false
}

// stubYearList liefert feste Geschäftsjahre.
type stubYearList []domain.FiscalYear

func (s stubYearList) FindAll(context.Context) ([]domain.FiscalYear, error) { return s, nil }

// stubSizes liefert eine feste Größenklasse.
type stubSizes struct{ class domain.SizeClassKind }

func (s stubSizes) SizeClassFor(_ context.Context, year int) (*domain.SizeClass, error) {
	return &domain.SizeClass{Year: year, Class: s.class}, nil
}

// Aufstellung, Feststellung, Ergebnisverwendung und Offenlegung hängen am
// Stichtag und an der Größenklasse und verschieben sich nicht; ihr Stand
// folgt der Statuskette des Jahresabschlusses.
func TestAnnualStatementObligations(t *testing.T) {
	years := stubYearList{
		{Year: 2025, StartDate: "2025-01-01", EndDate: "2025-12-31", Status: domain.FiscalYearOpen},
		{Year: 2026, StartDate: "2026-01-01", EndDate: "2026-12-31", Status: domain.FiscalYearOpen},
		{Year: 2027, StartDate: "2027-01-01", EndDate: "2027-12-31", Status: domain.FiscalYearOpen},
	}
	build := func(class domain.SizeClassKind, decision *domain.Appropriation, ys stubYearList) *ObligationService {
		svc := NewObligationService(nil, nil, nil)
		svc.SetFiscalYearSource(ys)
		svc.SetSizeSource(stubSizes{class: class})
		svc.SetAppropriationSource(stubAppropriationSource{decision: decision})
		svc.SetClock(func() time.Time { return obligationClock })
		return svc
	}

	small := listObligations(t, build(domain.SizeSmall, nil, years), ObligationQuery{FiscalYear: 2025})
	for key, want := range map[string]string{
		"abschlussbuchungen.2025": "2026-06-30",
		"aufstellung.2025":        "2026-06-30",
		"feststellung.2025":       "2026-11-30",
		"ergebnisverwendung.2025": "2026-11-30",
		"offenlegung.2025":        "2026-12-31",
		"e-bilanz.2025":           "2026-07-31",
	} {
		o, ok := obligationByKey(small, key)
		if !ok {
			t.Errorf("%s fehlt", key)
			continue
		}
		if o.DueDate != want {
			t.Errorf("%s fällig am %s, erwartet %s", key, o.DueDate, want)
		}
	}
	if o, _ := obligationByKey(small, "aufstellung.2025"); o.Status != domain.ObligationOverdue ||
		!containsHint(o.Hints, "ordnungsgemäßen Geschäftsgang") {
		t.Errorf("Aufstellung 2025 der kleinen Gesellschaft: %s %v", o.Status, o.Hints)
	}

	medium := listObligations(t, build(domain.SizeMedium, nil, years), ObligationQuery{FiscalYear: 2025})
	if o, _ := obligationByKey(medium, "aufstellung.2025"); o.DueDate != "2026-03-31" {
		t.Errorf("Aufstellung der mittelgroßen Gesellschaft fällig am %s, erwartet 2026-03-31", o.DueDate)
	}
	if o, _ := obligationByKey(medium, "feststellung.2025"); o.DueDate != "2026-08-31" {
		t.Errorf("Feststellung der mittelgroßen Gesellschaft fällig am %s, erwartet 2026-08-31", o.DueDate)
	}

	// Der 31.12.2028 ist ein Sonntag; die Offenlegungsfrist bleibt dort.
	later := listObligations(t, build(domain.SizeSmall, nil, years), ObligationQuery{FiscalYear: 2027})
	if o, _ := obligationByKey(later, "offenlegung.2027"); o.DueDate != "2028-12-31" || o.ShiftReason != "" {
		t.Errorf("Offenlegung 2027: fällig %s (%s), erwartet 2028-12-31 unverschoben", o.DueDate, o.ShiftReason)
	}

	// Die Statuskette erledigt eine Pflicht nach der anderen.
	done := stubYearList{{Year: 2025, StartDate: "2025-01-01", EndDate: "2025-12-31",
		Status: domain.FiscalYearAdopted, PreparedOn: "2026-05-20", AdoptedOn: "2026-06-15", AdoptionNote: "Beschluss vom 15.06."}}
	decided := &domain.Appropriation{Year: 2025, DecisionDate: "2026-06-15", Text: "Vortrag auf neue Rechnung"}
	chain := listObligations(t, build(domain.SizeSmall, decided, done), ObligationQuery{FiscalYear: 2025})
	for key, status := range map[string]domain.ObligationStatus{
		"aufstellung.2025":        domain.ObligationDone,
		"abschlussbuchungen.2025": domain.ObligationDone,
		"feststellung.2025":       domain.ObligationDone,
		"ergebnisverwendung.2025": domain.ObligationDone,
		"offenlegung.2025":        domain.ObligationOpen,
	} {
		if o, _ := obligationByKey(chain, key); o.Status != status {
			t.Errorf("%s: %s, erwartet %s", key, o.Status, status)
		}
	}
	if o, _ := obligationByKey(chain, "feststellung.2025"); o.Proof == nil || o.Proof.Reference != "Beschluss vom 15.06." {
		t.Errorf("Nachweis der Feststellung: %+v", o.Proof)
	}
	if _, ok := obligationByKey(chain, "kapitalertragsteuer.2025"); ok {
		t.Error("ohne Ausschüttung gibt es keine Kapitalertragsteuer")
	}

	// Festgestellt, aber ohne Beschluss über die Ergebnisverwendung.
	open := listObligations(t, build(domain.SizeSmall, nil, done), ObligationQuery{FiscalYear: 2025})
	if o, _ := obligationByKey(open, "ergebnisverwendung.2025"); o.Status != domain.ObligationOpen {
		t.Errorf("Ergebnisverwendung ohne Beschluss: %s", o.Status)
	}
}

// Eine Ausschüttung löst die Kapitalertragsteuer aus: anzumelden und
// abzuführen im Zeitpunkt des Zuflusses, ohne Auszahlungstag am Tag nach dem
// Beschluss.
func TestWithholdingTaxOnDistribution(t *testing.T) {
	svc := NewObligationService(nil, nil, nil)
	svc.SetFiscalYearSource(stubYearList{{Year: 2025, StartDate: "2025-01-01", EndDate: "2025-12-31",
		Status: domain.FiscalYearAdopted, PreparedOn: "2026-04-01", AdoptedOn: "2026-05-20"}})
	svc.SetSizeSource(stubSizes{class: domain.SizeSmall})
	svc.SetAppropriationSource(stubAppropriationSource{decision: &domain.Appropriation{
		Year: 2025, DecisionDate: "2026-05-20", Distribution: 1_000_000,
		WithholdingTax: 250_000, SolidarityOnWithholding: 13_750,
	}})
	svc.SetClock(func() time.Time { return obligationClock })

	o, err := svc.Get(context.Background(), "kapitalertragsteuer.2025", "")
	if err != nil {
		t.Fatalf("Kapitalertragsteuer: %v", err)
	}
	if o.DueDate != "2026-05-21" || o.Status != domain.ObligationOverdue || o.Channel != domain.ChannelElster {
		t.Errorf("Kapitalertragsteuer: fällig %s, %s, %s", o.DueDate, o.Status, o.Channel)
	}
	if !containsHint(o.Hints, "2.500,00") || !containsHint(o.Hints, "§ 44 Abs. 2 Satz 1 EStG") {
		t.Errorf("Hinweise: %v", o.Hints)
	}
}

// Vorauszahlungen stehen im Kalender, sobald gezahlt oder ein Bescheid
// erfasst ist; erledigt sind sie mit der gebuchten Zahlung. Zahltermine
// verschieben sich nicht.
func TestPrepaymentsFollowBookingsAndNotices(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.obligations(t)

	none := listObligations(t, svc, ObligationQuery{FiscalYear: 2026,
		Kinds: []domain.ObligationKind{domain.ObligationCorporatePrepay, domain.ObligationTradePrepay}})
	if len(none) != 0 {
		t.Fatalf("ohne Bescheid und Zahlung keine Vorauszahlung: %+v", none)
	}

	env.post(t, "2026-03-10", "7600", "1800", 250_000)
	kst := listObligations(t, svc, ObligationQuery{FiscalYear: 2026,
		Kinds: []domain.ObligationKind{domain.ObligationCorporatePrepay}})
	if len(kst) != 4 {
		t.Fatalf("vier Termine der Körperschaftsteuer erwartet: %+v", kst)
	}
	for key, status := range map[string]domain.ObligationStatus{
		"kst-vorauszahlung.2026-Q1": domain.ObligationDone,
		"kst-vorauszahlung.2026-Q2": domain.ObligationOverdue,
		"kst-vorauszahlung.2026-Q3": domain.ObligationOverdue,
		"kst-vorauszahlung.2026-Q4": domain.ObligationOpen,
	} {
		if o, _ := obligationByKey(kst, key); o.Status != status {
			t.Errorf("%s: %s, erwartet %s", key, o.Status, status)
		}
	}
	if q4, _ := obligationByKey(kst, "kst-vorauszahlung.2026-Q4"); q4.DueDate != "2026-12-10" {
		t.Errorf("Q4 fällig am %s", q4.DueDate)
	}

	if _, err := svc.RecordEvent(ctx, domain.EventTradePrepaymentNotice, "2026-04-01", "Bescheid der Stadt"); err != nil {
		t.Fatalf("Bescheid erfassen: %v", err)
	}
	trade := listObligations(t, svc, ObligationQuery{FiscalYear: 2026,
		Kinds: []domain.ObligationKind{domain.ObligationTradePrepay}})
	if _, ok := obligationByKey(trade, "gewst-vorauszahlung.2026-Q1"); ok {
		t.Error("der Termin vor dem Bescheid gehört nicht in den Kalender")
	}
	// Der 15.08.2026 ist ein Sonnabend; der Zahltermin bleibt.
	if q3, ok := obligationByKey(trade, "gewst-vorauszahlung.2026-Q3"); !ok || q3.DueDate != "2026-08-15" {
		t.Errorf("GewSt Q3: %+v", q3)
	}
	if len(trade) != 3 {
		t.Errorf("drei Termine nach dem Bescheid erwartet: %d", len(trade))
	}
}

// Die Lohnsteuer-Anmeldung steht im Kalender, sobald ein Lohnjournal gebucht
// ist, und ist mit dem gebuchten Lohnjournal des Zeitraums erledigt.
func TestPayrollTaxFollowsThePayrollJournal(t *testing.T) {
	env := newTestEnv(t)
	svc := env.obligations(t)
	payroll := func() []domain.Obligation {
		return listObligations(t, svc, ObligationQuery{FiscalYear: 2026,
			Kinds: []domain.ObligationKind{domain.ObligationPayrollTax}})
	}
	if got := payroll(); len(got) != 0 {
		t.Fatalf("ohne Lohn keine Anmeldung: %+v", got)
	}

	env.post(t, "2026-01-31", "6020", "3730", 40_000)
	env.post(t, "2026-02-28", "6020", "3730", 40_000)
	early := payroll()
	if len(early) != 2 {
		t.Fatalf("zwei Anmeldungen erwartet (Januar, Februar): %+v", early)
	}
	january, _ := obligationByKey(early, "lohnsteuer-anmeldung.2026-01")
	if january.Status != domain.ObligationDone || january.DueDate != "2026-02-10" ||
		january.Channel != domain.ChannelPayrollProvider {
		t.Errorf("Januar: %s fällig %s über %s", january.Status, january.DueDate, january.Channel)
	}
	if !containsHint(january.Hints, "Kalendermonat") {
		t.Errorf("Anmeldungszeitraum fehlt in den Hinweisen: %v", january.Hints)
	}

	// Lohn im September: die Monate dazwischen fehlen, der Oktober läuft.
	env.post(t, "2026-09-30", "6020", "3730", 40_000)
	later := payroll()
	for key, status := range map[string]domain.ObligationStatus{
		"lohnsteuer-anmeldung.2026-03": domain.ObligationOverdue,
		"lohnsteuer-anmeldung.2026-08": domain.ObligationOverdue,
		"lohnsteuer-anmeldung.2026-09": domain.ObligationDone,
		"lohnsteuer-anmeldung.2026-10": domain.ObligationOpen,
	} {
		if o, _ := obligationByKey(later, key); o.Status != status {
			t.Errorf("%s: %s, erwartet %s", key, o.Status, status)
		}
	}
	if _, ok := obligationByKey(later, "lohnsteuer-anmeldung.2026-11"); ok {
		t.Error("der November hat noch nicht begonnen")
	}
}

// § 41a Abs. 2 EStG: Kalenderjahr bis 1.080 €, Vierteljahr bis 5.000 €.
func TestPayrollPeriodTypeFromThePriorYear(t *testing.T) {
	run := &obligationRun{start: "2020-01-01"}
	cases := []struct {
		sum  domain.Cents
		want domain.VatPeriodType
	}{
		{108_000, domain.VatPeriodYear},
		{108_001, domain.VatPeriodQuarter},
		{500_000, domain.VatPeriodQuarter},
		{500_001, domain.VatPeriodMonth},
	}
	for _, c := range cases {
		got, _ := run.payrollPeriodType(2026, []ledgerPosting{{date: "2025-06-30", amount: c.sum}})
		if got != c.want {
			t.Errorf("%s € im Vorjahr: %s, erwartet %s", c.sum, got, c.want)
		}
	}
	// Bestand die Gesellschaft im Vorjahr nur vier Monate, wird umgerechnet:
	// 400 € in vier Monaten sind 1.200 € im Jahr.
	partial := &obligationRun{start: "2025-09-01"}
	if got, _ := partial.payrollPeriodType(2026, []ledgerPosting{{date: "2025-12-31", amount: 40_000}}); got != domain.VatPeriodQuarter {
		t.Errorf("umgerechnet: %s, erwartet Vierteljahr", got)
	}
}

// Ein Gesellschafterwechsel löst Gesellschafterliste und Transparenzregister
// aus; eine Änderung bei den wirtschaftlich Berechtigten nur das Register.
func TestEventObligations(t *testing.T) {
	env := newTestEnv(t)
	svc := env.obligations(t)
	ctx := context.Background()
	if _, err := svc.RecordEvent(ctx, "kapitalerhoehung", "2026-05-01", ""); err == nil {
		t.Error("ein unbekannter Anlass wird abgewiesen")
	}
	change, err := svc.RecordEvent(ctx, domain.EventShareholderChange, "2026-05-04", "Anteilskauf Bauer")
	if err != nil {
		t.Fatalf("Anlass: %v", err)
	}
	if _, err := svc.RecordEvent(ctx, domain.EventBeneficialOwnerChange, "2026-11-02", ""); err != nil {
		t.Fatalf("Anlass: %v", err)
	}
	list := listObligations(t, svc, ObligationQuery{FiscalYear: 2026,
		Kinds: []domain.ObligationKind{domain.ObligationShareholderList, domain.ObligationTransparency}})
	if len(list) != 3 {
		t.Fatalf("drei Pflichten erwartet: %+v", list)
	}
	shareholders, ok := obligationByKey(list, "gesellschafterliste.ereignis-1")
	if !ok || change.ID != 1 {
		t.Fatalf("Gesellschafterliste fehlt: %+v", list)
	}
	if !shareholders.Immediate || shareholders.DueDate != "2026-05-04" || shareholders.Status != domain.ObligationOverdue ||
		!containsHint(shareholders.Hints, "Anteilskauf Bauer") {
		t.Errorf("Gesellschafterliste: %+v", shareholders)
	}
	register, _ := obligationByKey(list, "transparenzregister.ereignis-1")
	if !containsHint(register.Hints, "25 %") {
		t.Errorf("Transparenzregister ohne Hinweis auf die wirtschaftlich Berechtigten: %v", register.Hints)
	}
}

// Die Gründungspflichten stehen nur im Kalender einer Gesellschaft, die mit
// Buchfink gegründet wurde.
func TestFoundingObligations(t *testing.T) {
	env := newTestEnv(t)
	env.setSettings(t, func(c *domain.CompanySettings) {
		c.FoundedOn, c.RegisteredOn, c.FederalState = "2026-03-15", "2026-04-20", "BY"
		c.TaxNumber = "181/815/08155"
	})
	list := listObligations(t, env.obligations(t), ObligationQuery{FiscalYear: 2026, Kinds: []domain.ObligationKind{
		domain.ObligationTaxRegistration, domain.ObligationTradeRegistration, domain.ObligationTransparency}})
	for key, due := range map[string]string{
		"steuerliche-erfassung.gruendung": "2026-04-15",
		// Der 15.03.2026 ist ein Sonntag; die Gewerbeanzeige ist keine Frist der AO.
		"gewerbeanmeldung.gruendung":    "2026-03-15",
		"transparenzregister.gruendung": "2026-04-20",
	} {
		o, ok := obligationByKey(list, key)
		if !ok {
			t.Errorf("%s fehlt", key)
			continue
		}
		if o.DueDate != due {
			t.Errorf("%s fällig am %s, erwartet %s", key, o.DueDate, due)
		}
	}
	if _, ok := obligationByKey(list, "ustva.2026-Q1"); ok {
		t.Error("die Auswahl nach Pflichtart gilt")
	}

	old := newTestEnv(t)
	old.setSettings(t, func(c *domain.CompanySettings) { c.FoundedOn = "2010-05-12" })
	if err := old.closing(t).EnsureFiscalYears(context.Background(), 2026); err != nil {
		t.Fatalf("Geschäftsjahre: %v", err)
	}
	none := listObligations(t, old.obligations(t), ObligationQuery{Kinds: []domain.ObligationKind{
		domain.ObligationTaxRegistration, domain.ObligationTradeRegistration}})
	if len(none) != 0 {
		t.Errorf("eine 2010 gegründete Gesellschaft hat keine offenen Gründungspflichten: %+v", none)
	}
}

// Die Gesellschafterliste in den Unternehmensdaten zu ändern meldet den
// Anlass; die erste Erfassung ist keine Veränderung.
func TestShareholderChangeInTheSettingsRecordsTheEvent(t *testing.T) {
	env := newTestEnv(t)
	obligations := env.obligations(t)
	settings := NewSettingsService(repository.NewSettingsRepository(env.db), repository.NewAuditRepository(env.db))
	settings.SetEventRecorder(obligations)
	update := func(holders ...domain.CompanyShareholder) {
		t.Helper()
		cfg, err := settings.GetCompanySettings(appCtx)
		if err != nil {
			t.Fatal(err)
		}
		cfg.ShareCapital = 0
		for _, h := range holders {
			cfg.ShareCapital += h.ShareCapital
		}
		cfg.Shareholders = holders
		if err := settings.UpdateCompanySettings(appCtx, cfg); err != nil {
			t.Fatalf("Unternehmensdaten: %v", err)
		}
	}
	count := func() int {
		return len(listObligations(t, obligations, ObligationQuery{
			Kinds: []domain.ObligationKind{domain.ObligationShareholderList}}))
	}
	update(domain.CompanyShareholder{Name: "Anna Bauer", ShareCapital: 2_500_000})
	if n := count(); n != 0 {
		t.Fatalf("die erste Erfassung löst %d Pflichten aus", n)
	}
	update(domain.CompanyShareholder{Name: "Anna Bauer", ShareCapital: 2_500_000})
	if n := count(); n != 0 {
		t.Fatalf("eine unveränderte Liste löst %d Pflichten aus", n)
	}
	update(domain.CompanyShareholder{Name: "Anna Bauer", ShareCapital: 1_500_000},
		domain.CompanyShareholder{Name: "Ben Conrad", ShareCapital: 1_000_000})
	if n := count(); n != 1 {
		t.Fatalf("der Gesellschafterwechsel löst %d Gesellschafterlisten aus, erwartet eine", n)
	}
}

// Die Liste filtert nach Fälligkeit und Stand; Next liefert, was noch etwas
// verlangt, die früheste Frist zuerst.
func TestListFiltersAndNext(t *testing.T) {
	env := newTestEnv(t)
	svc := env.obligations(t)

	window := listObligations(t, svc, ObligationQuery{From: "2026-11-01", To: "2026-11-30"})
	for _, o := range window {
		if o.DueDate < "2026-11-01" || o.DueDate > "2026-11-30" {
			t.Errorf("%s fällig am %s liegt außerhalb", o.Key, o.DueDate)
		}
	}
	if _, ok := obligationByKey(window, "festschreibung.2026-10"); !ok {
		t.Errorf("die Festschreibung des Oktobers ist im November fällig: %+v", window)
	}

	overdue := listObligations(t, svc, ObligationQuery{FiscalYear: 2026,
		Statuses: []domain.ObligationStatus{domain.ObligationOverdue}})
	for _, o := range overdue {
		if o.Status != domain.ObligationOverdue {
			t.Errorf("%s steht auf %s", o.Key, o.Status)
		}
	}
	if _, err := svc.List(context.Background(), ObligationQuery{Statuses: []domain.ObligationStatus{"fertig"}}); err == nil {
		t.Error("ein unbekannter Stand wird abgewiesen")
	}
	if _, err := svc.List(context.Background(), ObligationQuery{Kinds: []domain.ObligationKind{"lohnsteuer"}}); err == nil {
		t.Error("eine unbekannte Pflichtart wird abgewiesen")
	}

	next, err := svc.Next(context.Background(), "", 5)
	if err != nil {
		t.Fatalf("nächste Pflichten: %v", err)
	}
	if len(next) != 5 {
		t.Fatalf("fünf Pflichten erwartet, erhalten %d", len(next))
	}
	for i := range next {
		if next[i].Status.Closed() {
			t.Errorf("%s ist schon %s", next[i].Key, next[i].Status)
		}
		if i > 0 && next[i].DueDate < next[i-1].DueDate {
			t.Errorf("Reihenfolge: %s vor %s", next[i-1].DueDate, next[i].DueDate)
		}
	}
}

// Jede Pflichtart des Katalogs verweist auf ihren Anker in der Skill-Referenz.
func TestCatalogCarriesSkillReferences(t *testing.T) {
	seen := map[domain.ObligationKind]bool{}
	for _, typ := range ObligationTypes() {
		if seen[typ.Kind] {
			t.Errorf("%s steht zweimal im Katalog", typ.Kind)
		}
		seen[typ.Kind] = true
		if typ.SkillRef != "buchfink-grundlagen/references/pflichten.md#"+string(typ.Kind) {
			t.Errorf("%s verweist auf %q", typ.Kind, typ.SkillRef)
		}
		if typ.Basis == "" || typ.Title == "" || typ.Channel == "" || typ.Responsibility == "" || typ.Evidence == "" {
			t.Errorf("%s ist unvollständig: %+v", typ.Kind, typ)
		}
		if strings.Contains(typ.Title, "§") {
			t.Errorf("der Titel von %s nennt eine Norm", typ.Kind)
		}
	}
	if len(seen) != 21 {
		t.Errorf("%d Pflichtarten im Katalog, erwartet 21", len(seen))
	}
}
