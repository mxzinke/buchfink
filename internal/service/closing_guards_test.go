package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// -------------------------------------------------------------------------
// Abzinsung ohne gepflegte Zinstabelle
// -------------------------------------------------------------------------

// Der Normalfall einer frischen Installation: die Zinstabelle der Deutschen
// Bundesbank ist leer, weil sie gepflegt und nicht ausgeliefert wird.
//
// Entscheidung 3 sagt für diesen Fall: „fehlt der Satz für den Stichtag, wird
// nicht abgezinst und ein Befund erzeugt". Ein Fehler wäre das Gegenteil davon —
// er nähme dem Anwender die Rückstellung ganz, statt ihm zu sagen, was fehlt.
func TestProvisionWithoutAnyRateTableStillPreviewsAndBooks(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	preview, err := m.provisions.Preview(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionUncertainLiability,
		Text: "Rückbauverpflichtung", Amount: 1_000_000, ExpectedOn: "2029-12-30",
		Reason: "Mietvertrag § 12: Rückbau der Einbauten bei Auszug",
	})
	if err != nil {
		t.Fatalf("Vorschau auf leerer Zinstabelle: %v", err)
	}
	if preview.Discounted {
		t.Error("ohne jeden hinterlegten Satz darf nicht abgezinst werden")
	}
	if preview.Amount != 1_000_000 {
		t.Errorf("gebucht würden %s € — erwartet den vollen Erfüllungsbetrag", preview.Amount)
	}
	if len(preview.Findings) == 0 {
		t.Error("die leere Zinstabelle muss als Befund in der Vorschau stehen")
	}

	provision, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionUncertainLiability,
		Text: "Rückbauverpflichtung", Amount: 1_000_000, ExpectedOn: "2029-12-30",
		Reason: "Mietvertrag § 12: Rückbau der Einbauten bei Auszug",
	})
	if err != nil {
		t.Fatalf("Bildung auf leerer Zinstabelle: %v", err)
	}
	if provision.Balance() != 1_000_000 {
		t.Errorf("Bestand %s € — erwartet den vollen Erfüllungsbetrag", provision.Balance())
	}

	// Und der Prüflauf sagt dasselbe: der fehlende Satz ist ein Befund, kein
	// verschluckter Fehler.
	findings, err := m.provisions.DiscountFindings(ctx, 2026)
	if err != nil {
		t.Fatalf("Befunde der Rückstellungen: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("die fehlende Zinstabelle muss einen Befund erzeugen")
	}
	if findings[0].Rule != domain.CheckRuleProvisionDiscount {
		t.Errorf("Regel %q — erwartet %q", findings[0].Rule, domain.CheckRuleProvisionDiscount)
	}

	// Auch die Sätze selbst lassen sich lesen: leere Tabelle, leere Liste.
	rates, err := m.provisions.DiscountRates(ctx, "")
	if err != nil {
		t.Fatalf("Zinssätze lesen: %v", err)
	}
	if len(rates) != 0 {
		t.Errorf("erwartet keine Sätze, bekommen %d", len(rates))
	}
}

// -------------------------------------------------------------------------
// Schutz der Steuerkonten
// -------------------------------------------------------------------------

// Die Ausnahme vom Steuerkonten-Schutz gilt der Umsatzsteuer-Jahresverrechnung
// und sonst nichts. Entscheidung 5: „Die Steuerkonten dürfen danach nur noch
// über die Automatik bebucht werden (unverändert)."
func TestClosingEntryMayNotWriteTaxAccountsOutsideTheSettlement(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	entry := &domain.JournalEntry{
		BookingDate: "2026-12-31", DocumentDate: "2026-12-31",
		ServiceDateFrom: "2026-12-31", ServiceDateTo: "2026-12-31",
		Description: "Abschlussbuchung", Source: domain.EntrySourceClosing,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountVorsteuer19, Amount: 60_000},
			{Side: domain.SideCredit, Account: "3070", Amount: 60_000},
		},
	}
	if _, err := env.journal.Post(ctx, entry); err == nil {
		t.Fatal("eine Abschlussbuchung ohne Steuerschlüssel darf kein Steuerkonto bebuchen")
	} else if !strings.Contains(err.Error(), "Steuerautomatik") {
		t.Errorf("die Meldung nennt den Grund nicht: %v", err)
	}

	// Die Belegnummer allein trägt die Ausnahme nicht: eine Handbuchung mit der
	// Belegnummer der Verrechnung bleibt eine Handbuchung.
	entry.DocumentNumber = VatSettlementReference(2026)
	entry.Source = domain.EntrySourceManual
	if _, err := env.journal.Post(ctx, entry); err == nil {
		t.Fatal("eine Handbuchung darf kein Steuerkonto bebuchen, auch nicht unter der Belegnummer der Verrechnung")
	}

	// Die Verrechnung selbst geht durch — sie ist die eine Ausnahme, und sie
	// entsteht nur im Dienst: die Bridge normiert die Quelle jeder Handbuchung
	// auf manual.
	entry.Source = domain.EntrySourceClosing
	if _, err := env.journal.Post(ctx, entry); err != nil {
		t.Fatalf("die Umsatzsteuer-Jahresverrechnung muss durchgehen: %v", err)
	}
}

// Ein anwendergewähltes Konto der Rückstellung darf kein Steuerkonto sein: die
// Voranmeldung lässt Abschlussbuchungen aus, die Jahresverrechnung fegte den so
// entstandenen Saldo aber mit — Journal und Jahreserklärung liefen auseinander.
func TestProvisionMayNotUseATaxAccount(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	if _, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionClosingCosts, Text: "Jahresabschluss 2026",
		Amount: 60_000, ExpectedOn: "2027-06-30", Reason: "Angebot der Kanzlei",
		ExpenseAccount: domain.AccountVorsteuer19,
	}); err == nil {
		t.Fatal("eine Rückstellung darf nicht auf ein Steuerkonto gebucht werden")
	}

	after := balances(t, env, 2026)
	if after[domain.AccountVorsteuer19] != 0 {
		t.Errorf("auf %s steht ein Saldo von %s €, obwohl nichts gebucht werden durfte",
			domain.AccountVorsteuer19, after[domain.AccountVorsteuer19])
	}
}

// -------------------------------------------------------------------------
// Stornierte Rückstellung
// -------------------------------------------------------------------------

// Nach dem Storno der Bildung steht im Journal kein Bestand mehr. Ihn trotzdem
// aufzulösen erzeugte einen Ertrag aus der Auflösung von etwas, das es nicht
// gibt — die Lesepfade rechnen den Storno längst heraus, die Schreibpfade
// müssen es auch.
func TestReversedProvisionCannotBeReleasedOrConsumed(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	provision, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionClosingCosts, Text: "Jahresabschluss 2026",
		Amount: 300_000, ExpectedOn: "2027-06-30", Reason: "Angebot der Kanzlei",
	})
	if err != nil {
		t.Fatalf("Rückstellung bilden: %v", err)
	}
	entryID := provision.Movements[0].JournalEntryID
	if entryID == nil {
		t.Fatal("die Bildung trägt keine Buchung")
	}
	if _, err := env.journal.Reverse(ctx, *entryID, "doppelt erfasst"); err != nil {
		t.Fatalf("Bildung stornieren: %v", err)
	}

	if _, err := m.provisions.BookRelease(ctx, ProvisionChangeRequest{
		ProvisionID: provision.ID, Amount: 100_000, Date: "2027-03-31",
		Reason: "Die Kanzlei hat weniger berechnet",
	}); err == nil {
		t.Error("eine stornierte Rückstellung darf nicht aufgelöst werden")
	}

	if _, err := m.provisions.BookConsumption(ctx, ProvisionChangeRequest{
		ProvisionID: provision.ID, Amount: 100_000, Date: "2027-03-31",
		PaymentAccount: domain.AccountBank, Reason: "Rechnung der Kanzlei",
	}); err == nil {
		t.Error("eine stornierte Rückstellung darf nicht verbraucht werden")
	}

	if _, _, err := m.provisions.ConsumptionSplit(ctx, provision.ID, 100_000); err == nil {
		t.Error("der Belegweg darf nicht gegen eine stornierte Rückstellung buchen")
	}
}

// -------------------------------------------------------------------------
// Übersprungene Schritte im Prüfbericht
// -------------------------------------------------------------------------

// Ein übersprungener Baustein ist eine Aussage und kein Versehen — und sie
// gehört in den Bericht, den der Prüfer liest.
func TestSkippedClosingStepAppearsInTheCheckRun(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	checks := env.checks(t)
	checks.SetClosingStepSource(m.steps)

	if _, err := m.steps.SkipStep(ctx, 2026, domain.ClosingStepProvisions,
		"Es liegen keine ungewissen Verbindlichkeiten vor"); err != nil {
		t.Fatalf("Schritt überspringen: %v", err)
	}

	run, err := checks.Preview(ctx, CheckRequest{CutoffDate: "2026-12-31", PeriodType: "year"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	found := false
	for _, finding := range run.Findings {
		if finding.Rule != domain.CheckRuleClosingStepSkipped {
			continue
		}
		found = true
		if finding.Severity != domain.CheckWarning {
			t.Errorf("ein übersprungener Schritt blockiert nicht: %q", finding.Severity)
		}
		if !strings.Contains(finding.Message, "ungewissen Verbindlichkeiten") {
			t.Errorf("der Befund nennt den Grund nicht: %q", finding.Message)
		}
	}
	if !found {
		t.Errorf("der übersprungene Schritt fehlt im Prüfbericht: %+v", run.Findings)
	}

	// Der Monatslauf ist nicht der Ort dafür: der Abschluss wird zum Jahresende
	// beurteilt.
	monthly, err := checks.Preview(ctx, CheckRequest{CutoffDate: "2026-01-31", PeriodType: "month"})
	if err != nil {
		t.Fatalf("Monatslauf: %v", err)
	}
	for _, finding := range monthly.Findings {
		if finding.Rule == domain.CheckRuleClosingStepSkipped {
			t.Error("der Monatslauf soll die Abschlussschritte nicht beurteilen")
		}
	}
}

// -------------------------------------------------------------------------
// Leere Listen statt null
// -------------------------------------------------------------------------

// Der leere Mandant ist der Fall, der die Oberfläche trifft: ein frisch
// angelegtes Geschäftsjahr, in dem noch nichts gebucht ist.
func TestClosingModulesReturnEmptyListsNotNull(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	steps, err := m.steps.Steps(ctx, 2026)
	if err != nil {
		t.Fatalf("Schrittliste: %v", err)
	}
	assertNoNilSlices(t, "Schrittliste", steps)

	proposal, err := m.accruals.Propose(ctx, 2026)
	if err != nil {
		t.Fatalf("Abgrenzungsvorschlag: %v", err)
	}
	assertNoNilSlices(t, "Abgrenzungsvorschlag", proposal)

	accruals, err := m.accruals.List(ctx, 2026)
	if err != nil {
		t.Fatalf("Abgrenzungen: %v", err)
	}
	assertNoNilSlices(t, "Abgrenzungen", accruals)

	report, err := m.accruals.Report(ctx, "2026-12-31")
	if err != nil {
		t.Fatalf("Abgrenzungsbericht: %v", err)
	}
	assertNoNilSlices(t, "Abgrenzungsbericht", report)

	provisions, err := m.provisions.List(ctx, 2026)
	if err != nil {
		t.Fatalf("Rückstellungen: %v", err)
	}
	assertNoNilSlices(t, "Rückstellungen", provisions)

	mirror, err := m.provisions.Mirror(ctx, 2026)
	if err != nil {
		t.Fatalf("Rückstellungsspiegel: %v", err)
	}
	assertNoNilSlices(t, "Rückstellungsspiegel", mirror)

	rates, err := m.provisions.DiscountRates(ctx, "")
	if err != nil {
		t.Fatalf("Zinssätze: %v", err)
	}
	assertNoNilSlices(t, "Zinssätze", rates)

	months, err := m.provisions.DiscountRateMonths(ctx)
	if err != nil {
		t.Fatalf("Zinsmonate: %v", err)
	}
	assertNoNilSlices(t, "Zinsmonate", months)

	inventory, err := m.bookings.InventoryAccounts(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorratskonten: %v", err)
	}
	assertNoNilSlices(t, "Vorratskonten", inventory)

	settlement, err := m.bookings.PreviewVatSettlement(ctx, 2026)
	if err != nil {
		t.Fatalf("Umsatzsteuer-Verrechnung: %v", err)
	}
	assertNoNilSlices(t, "Umsatzsteuer-Verrechnung", settlement)

	taxProvision, err := m.bookings.PreviewTaxProvision(ctx, 2026)
	if err != nil {
		t.Fatalf("Steuerrückstellung: %v", err)
	}
	assertNoNilSlices(t, "Steuerrückstellung", taxProvision)

	register, err := m.register.Register(ctx, 2026)
	if err != nil {
		t.Fatalf("Verzeichnis nach § 5 Abs. 1 Satz 2 EStG: %v", err)
	}
	assertNoNilSlices(t, "Verzeichnis", register)

	reconciliation, err := m.register.Reconcile(ctx, 2026)
	if err != nil {
		t.Fatalf("Überleitungsrechnung: %v", err)
	}
	assertNoNilSlices(t, "Überleitungsrechnung", reconciliation)

	notes, err := m.appropriation.NotesTexts(ctx, 2026)
	if err != nil {
		t.Fatalf("Anhangtexte: %v", err)
	}
	assertNoNilSlices(t, "Anhangtexte", notes)

	// Gegenprobe: der Test prüft nur etwas, solange ein nil-Slice auch
	// wirklich als `null` in der Ausgabe steht.
	raw, err := json.Marshal(struct {
		Rows []string `json:"rows"`
	}{})
	if err != nil {
		t.Fatalf("Gegenprobe: %v", err)
	}
	if !strings.Contains(string(raw), `"rows":null`) {
		t.Errorf("die Gegenprobe trägt nicht mehr: %s", raw)
	}
}

// Die Vorschauen der Abschlussbausteine gehen ebenso an die Oberfläche wie die
// Bestandslisten. Sie werden gezeigt, bevor gebucht wird — und im leeren Jahr
// ist jede von ihnen leer.
func TestClosingPreviewsReturnEmptyListsNotNull(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	inventory, err := m.bookings.PreviewInventory(ctx, InventoryRequest{
		Account: "1140", FiscalYear: 2026,
	})
	if err != nil {
		t.Fatalf("Vorschau Inventurwert: %v", err)
	}
	assertNoNilSlices(t, "Vorschau Inventurwert", inventory)

	appropriation, err := m.appropriation.PreviewAppropriation(ctx, 2026, AppropriationRequest{})
	if err != nil {
		t.Fatalf("Vorschau Ergebnisverwendung: %v", err)
	}
	assertNoNilSlices(t, "Vorschau Ergebnisverwendung", appropriation)

	// Der Vortrag bucht die Auflösung der Rechnungsabgrenzung mit; die Vorschau
	// nennt sie, und die Ansicht läuft über die Liste.
	carry, err := m.closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortragsvorschau: %v", err)
	}
	assertNoNilSlices(t, "Vortragsvorschau", carry)

	releases, err := m.accruals.PendingReleases(ctx, 2027)
	if err != nil {
		t.Fatalf("fällige Auflösungen: %v", err)
	}
	assertNoNilSlices(t, "fällige Auflösungen", releases)

	state, err := m.closing.ClosingStateFor(ctx, 2026)
	if err != nil {
		t.Fatalf("Abschlusszustand: %v", err)
	}
	assertNoNilSlices(t, "Abschlusszustand", state)

	skipped, err := m.steps.SkippedSteps(ctx, 2026)
	if err != nil {
		t.Fatalf("übersprungene Schritte: %v", err)
	}
	assertNoNilSlices(t, "übersprungene Schritte", skipped)
}
