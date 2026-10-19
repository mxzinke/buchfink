package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/ebilanz"
	"github.com/buchfink/buchfink/internal/repository"
)

// openingEnv verdrahtet die Gründung mit Gliederung und Satz, wie es die
// Anwendung tut.
func openingEnv(t *testing.T, env *testEnv) *FoundationService {
	t.Helper()
	svc := env.foundations(t)
	wireOpening(t, env, svc)
	return svc
}

// wireOpening gibt der Gründung Sitz, Gliederung und Satz.
func wireOpening(t *testing.T, env *testEnv, svc *FoundationService) {
	t.Helper()
	// Der Sitz ist Pflichtangabe im Kopf (§ 264 Abs. 1a Nr. 2 HGB). Die
	// Testumgebung setzt ihn nicht — hier steht er, damit die Prüfung auf
	// fehlende Angaben nicht jede Aufstellung beanstandet.
	settings := repository.NewSettingsRepository(env.db)
	cfg, err := settings.GetCompanySettings(context.Background())
	if err != nil {
		t.Fatalf("Unternehmensdaten: %v", err)
	}
	cfg.Seat = "München"
	if err := settings.UpdateCompanySettings(context.Background(), cfg); err != nil {
		t.Fatalf("Sitz setzen: %v", err)
	}

	svc.SetStatementSource(env.statements(t))
	// Der echte Satz: die Eröffnungsbilanz entsteht als PDF, und ein
	// Attrappen-Renderer prüfte nur die Verkabelung.
	svc.SetRenderer(sharedRenderer())
}

// Die Eröffnungsbilanz steht auf den Tag der Beurkundung (§ 242 Abs. 1 HGB) und
// zeigt, was an diesem Tag in den Büchern steht: die Zeichnung des
// Stammkapitals und die geleistete Einlage.
func TestOpeningBalanceIsDrawnOnTheNotarizationDay(t *testing.T) {
	env := newTestEnv(t)
	svc := openingEnv(t, env)
	ctx := context.Background()

	env.saveFoundation(t, svc, gmbhFoundation()) // Beurkundung 15.01.2026
	if _, err := svc.BookPostings(ctx); err != nil {
		t.Fatalf("Gründungsbuchungen: %v", err)
	}
	// Die Notarrechnung kommt danach und gehört nicht in die Eröffnungsbilanz.
	env.book(t, "2026-01-20", "Notarkosten Gründung", "6825", "1800", 300_000)

	sheet, err := svc.OpeningBalance(ctx)
	if err != nil {
		t.Fatalf("Eröffnungsbilanz: %v", err)
	}
	if sheet.AsOf != "2026-01-15" {
		t.Errorf("Stichtag %q, erwartet den Beurkundungstag 2026-01-15", sheet.AsOf)
	}
	if !sheet.Balances {
		t.Errorf("die Eröffnungsbilanz geht nicht auf: Aktiva %s €, Passiva %s €", sheet.Assets, sheet.Equity)
	}
	// Bank 12.500 plus ausstehende Einlage 12.500 gegen Stammkapital 25.000.
	if sheet.Assets != 2_500_000 {
		t.Errorf("Aktiva %s €, erwartet 25.000,00", sheet.Assets)
	}
	if len(sheet.Findings) != 0 {
		t.Errorf("unerwartete Befunde: %v", sheet.Findings)
	}

	// Der Kopf trägt die Firma mit dem Zusatz und die Rechtsgrundlage.
	if !strings.Contains(sheet.Header.FirmName, "i. G.") {
		t.Errorf("Firma %q, erwartet den Zusatz i. G.", sheet.Header.FirmName)
	}
	if sheet.Header.Reference != "§ 242 Abs. 1 HGB" {
		t.Errorf("Rechtsgrundlage %q", sheet.Header.Reference)
	}
	if sheet.Header.ClosingDate != "2026-01-15" {
		t.Errorf("Stichtag im Kopf %q, erwartet den Beurkundungstag", sheet.Header.ClosingDate)
	}
}

// Ohne Gründungsbuchung sagt die Aufstellung, was fehlt, statt eine leere
// Bilanz auszuweisen.
func TestOpeningBalanceNamesTheMissingPosting(t *testing.T) {
	env := newTestEnv(t)
	svc := openingEnv(t, env)
	ctx := context.Background()
	env.saveFoundation(t, svc, gmbhFoundation())

	sheet, err := svc.OpeningBalance(ctx)
	if err != nil {
		t.Fatalf("Eröffnungsbilanz: %v", err)
	}
	if len(sheet.Findings) == 0 {
		t.Fatal("ohne jede Buchung muss die Aufstellung einen Befund nennen")
	}
	if !strings.Contains(strings.Join(sheet.Findings, " "), "Zeichnung des Stammkapitals") {
		t.Errorf("der Befund nennt nicht, was zu tun ist: %v", sheet.Findings)
	}
}

// Registergericht und -nummer fehlen vor der Eintragung notwendigerweise. Nach
// ihnen zu fragen hieße, etwas zu verlangen, das es noch nicht geben kann.
func TestOpeningBalanceDoesNotAskForTheRegisterBeforeRegistration(t *testing.T) {
	env := newTestEnv(t)
	svc := openingEnv(t, env)
	ctx := context.Background()
	env.saveFoundation(t, svc, gmbhFoundation())
	if _, err := svc.BookPostings(ctx); err != nil {
		t.Fatalf("Gründungsbuchungen: %v", err)
	}

	sheet, err := svc.OpeningBalance(ctx)
	if err != nil {
		t.Fatalf("Eröffnungsbilanz: %v", err)
	}
	for _, finding := range sheet.Findings {
		if strings.Contains(finding, "Register") {
			t.Errorf("vor der Eintragung darf das Register kein Mangel sein: %q", finding)
		}
	}
}

// Eine Bilanz, die nicht aufgeht, wird nicht gesetzt.
type unbalancedOpeningSource struct{ StatementAtSource }

func (s unbalancedOpeningSource) StatementAt(ctx context.Context, cutoff string, depth domain.StatementDepth) (*domain.Statement, error) {
	stmt, err := s.StatementAtSource.StatementAt(ctx, cutoff, depth)
	if err != nil {
		return nil, err
	}
	stmt.Assets = []domain.StatementLine{{Level: 1, Amount: 100_000}}
	stmt.Liabilities = []domain.StatementLine{}
	return stmt, nil
}

func TestOpeningBalancePDFRefusesAnUnbalancedSheet(t *testing.T) {
	env := newTestEnv(t)
	svc := openingEnv(t, env)
	ctx := context.Background()
	env.saveFoundation(t, svc, gmbhFoundation())
	// Eine Aufwandsbuchung bleibt über das Jahresergebnis bilanziell
	// ausgeglichen. Nur ein tatsächlich unausgeglichener Bericht prüft die Sperre.
	svc.SetStatementSource(unbalancedOpeningSource{env.statements(t)})

	if _, err := svc.OpeningBalancePDF(ctx); err == nil {
		t.Fatal("eine Bilanz, die nicht aufgeht, darf nicht gesetzt werden")
	}
}

// Die Eröffnungsbilanz entsteht als PDF auf den Beurkundungstag.
func TestOpeningBalancePDFIsSetOnTheNotarizationDay(t *testing.T) {
	env := newTestEnv(t)
	svc := openingEnv(t, env)
	ctx := context.Background()

	env.saveFoundation(t, svc, gmbhFoundation())
	if _, err := svc.BookPostings(ctx); err != nil {
		t.Fatalf("Gründungsbuchungen: %v", err)
	}

	file, err := svc.OpeningBalancePDF(ctx)
	if err != nil {
		t.Fatalf("Eröffnungsbilanz setzen: %v", err)
	}
	if file.FileName != "Eroeffnungsbilanz_2026-01-15.pdf" {
		t.Errorf("Dateiname %q, erwartet den Beurkundungstag", file.FileName)
	}
	if len(file.Data) < 4 || !strings.HasPrefix(string(file.Data[:4]), "%PDF") {
		t.Error("die Eröffnungsbilanz ist kein PDF")
	}
}

// openingEBilanz erzeugt die E-Bilanz der Eröffnungsbilanz wie die App und
// ebilanz_exportieren und liest Stichtag und Bilanzsummen aus der Instanz.
func openingEBilanz(t *testing.T, env *testEnv, year int) (closing, assets, equity string) {
	t.Helper()
	svc := NewEBilanzService(env.statements(t), repository.NewSettingsRepository(env.db),
		repository.NewAuditRepository(env.db))
	res, err := svc.Generate(context.Background(), year, EBilanzOptions{Kind: ebilanz.KindOpening})
	if err != nil {
		t.Fatalf("E-Bilanz der Eröffnungsbilanz: %v", err)
	}
	if err := res.Report.BlockingError(); err != nil {
		t.Fatalf("E-Bilanz der Eröffnungsbilanz blockiert: %v", err)
	}
	fact := func(element string) string {
		m := regexp.MustCompile(`<de-[\w-]+:` + regexp.QuoteMeta(element) + ` [^>]*>([^<]*)<`).FindSubmatch(res.Instance)
		if m == nil {
			t.Fatalf("die Instanz enthält %s nicht", element)
		}
		return string(m[1])
	}
	return fact("genInfo.report.period.balSheetClosingDate"), fact("bs.ass"), fact("bs.eqLiab")
}

// xbrlAmount schreibt einen Betrag wie die Instanz: Euro mit Punkt.
func xbrlAmount(c domain.Cents) string { return fmt.Sprintf("%d.%02d", c/100, c%100) }

// Das PDF zum Unterzeichnen und die E-Bilanz der Eröffnungsbilanz zeigen
// denselben Stichtag und dieselben Summen. Was nach dem Stichtag gebucht ist,
// steht in keinem von beiden.
func TestOpeningBalancePDFAndEBilanzShowTheSameBalance(t *testing.T) {
	env := newTestEnv(t)
	svc := openingEnv(t, env)
	ctx := context.Background()
	withElsterTaxNumber(t, env)

	env.saveFoundation(t, svc, gmbhFoundation()) // Beurkundung 15.01.2026
	if _, err := svc.BookPostings(ctx); err != nil {
		t.Fatalf("Gründungsbuchungen: %v", err)
	}
	env.book(t, "2026-01-15", "Einzahlung Anna Bauer", "1800", "1298", 750_000)
	env.book(t, "2026-01-20", "Notarkosten Gründung", "6825", "1800", 300_000)

	sheet, err := svc.OpeningBalance(ctx)
	if err != nil {
		t.Fatalf("Eröffnungsbilanz: %v", err)
	}
	file, err := svc.OpeningBalancePDF(ctx)
	if err != nil {
		t.Fatalf("Eröffnungsbilanz setzen: %v", err)
	}
	closing, assets, equity := openingEBilanz(t, env, 2026)

	if sheet.AsOf != "2026-01-15" || closing != sheet.AsOf || file.FileName != "Eroeffnungsbilanz_"+closing+".pdf" {
		t.Errorf("Stichtag: Aufstellung %s, PDF %s, E-Bilanz %s", sheet.AsOf, file.FileName, closing)
	}
	if assets != xbrlAmount(sheet.Assets) || equity != xbrlAmount(sheet.Equity) {
		t.Errorf("Summen: PDF Aktiva %s, Passiva %s; E-Bilanz Aktiva %s, Passiva %s",
			xbrlAmount(sheet.Assets), xbrlAmount(sheet.Equity), assets, equity)
	}
	if sheet.Assets != 2_500_000 {
		t.Errorf("Aktiva %s €, erwartet 25.000,00 ohne die Notarkosten", sheet.Assets)
	}
}

// Der Tag der Beurkundung hat eine Quelle: das Gründungsdatum der
// Unternehmensdaten. Die Gründung schreibt es, die Stammdaten berichtigen es,
// und Geschäftsjahr, Gründungsbuchung, Eröffnungsbilanz als PDF und E-Bilanz
// sowie die Pflichten stehen danach auf demselben Tag.
func TestFoundingDateIsTheSameForYearOpeningBalanceAndObligations(t *testing.T) {
	env := newTestEnv(t)
	closing, svc := alignEnv(t, env)
	wireOpening(t, env, svc)
	ctx := context.Background()
	withElsterTaxNumber(t, env)
	if err := closing.EnsureFiscalYears(ctx, 2026); err != nil {
		t.Fatalf("Geschäftsjahre anlegen: %v", err)
	}

	env.saveFoundationOn(t, svc, "2026-03-10", gmbhFoundation())
	settings := NewSettingsService(repository.NewSettingsRepository(env.db), repository.NewAuditRepository(env.db))
	settings.SetFoundingYearAligner(closing)
	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Unternehmensdaten: %v", err)
	}
	if cfg.FoundedOn != "2026-03-10" {
		t.Fatalf("Gründungsdatum %q, erwartet den Beurkundungstag der Gründung", cfg.FoundedOn)
	}
	cfg.FoundedOn = "2026-03-12"
	if err := settings.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatalf("Gründungsdatum berichtigen: %v", err)
	}

	preview, err := svc.PreviewPostings(ctx)
	if err != nil || len(preview.Postings) == 0 {
		t.Fatalf("Gründungsbuchungen: %+v (%v)", preview, err)
	}
	if _, err := svc.BookPostings(ctx); err != nil {
		t.Fatalf("Gründungsbuchungen: %v", err)
	}
	fy, err := closing.PeriodOf(ctx, 2026)
	if err != nil {
		t.Fatalf("Geschäftsjahr 2026: %v", err)
	}
	sheet, err := svc.OpeningBalance(ctx)
	if err != nil {
		t.Fatalf("Eröffnungsbilanz: %v", err)
	}
	file, err := svc.OpeningBalancePDF(ctx)
	if err != nil {
		t.Fatalf("Eröffnungsbilanz setzen: %v", err)
	}
	ebilanzDay, _, _ := openingEBilanz(t, env, 2026)
	obligations := env.obligations(t)
	var obligationDays []string
	for _, key := range []string{"eroeffnungsbilanz.gruendung", "e-bilanz-eroeffnung.gruendung"} {
		o, err := obligations.Get(ctx, key, "2026-10-17")
		if err != nil {
			t.Fatalf("Pflicht %s: %v", key, err)
		}
		obligationDays = append(obligationDays, o.PeriodFrom)
	}

	const want = "2026-03-12"
	for label, got := range map[string]string{
		"Beginn des Geschäftsjahres": fy.StartDate,
		"Gründungsbuchung":           preview.Postings[0].Date,
		"Eröffnungsbilanz":           sheet.AsOf,
		"PDF":                        strings.TrimSuffix(strings.TrimPrefix(file.FileName, "Eroeffnungsbilanz_"), ".pdf"),
		"E-Bilanz":                   ebilanzDay,
		"Pflicht Eröffnungsbilanz":   obligationDays[0],
		"Pflicht E-Bilanz":           obligationDays[1],
	} {
		if got != want {
			t.Errorf("%s steht auf %q, erwartet %s", label, got, want)
		}
	}
}
