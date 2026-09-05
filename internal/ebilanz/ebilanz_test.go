package ebilanz

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

func minimalSettings() *domain.CompanySettings {
	return &domain.CompanySettings{
		CompanyName: "Muster GmbH",
		LegalForm:   "GmbH",
		TaxNumber:   "12/345/67890",
		Seat:        "München",

		RegisterCourt:  "Amtsgericht München",
		RegisterNumber: "HRB 123456",
		FiscalYear:     2026,
	}
}

// accountsWith baut Konten des echten SKR04-Katalogs mit vorgegebenen Salden.
// Positiv heißt Sollsaldo.
func accountsWith(t *testing.T, balances map[string]domain.Cents) []domain.Account {
	t.Helper()
	chart := accounting.NewChart(accounting.DefaultSKR04Accounts())
	out := make([]domain.Account, 0, len(balances))
	for number, value := range balances {
		acc, ok := chart.Lookup(number)
		if !ok {
			t.Fatalf("Konto %s steht nicht im SKR04-Katalog", number)
		}
		if value >= 0 {
			acc.DebitSum = value
		} else {
			acc.CreditSum = -value
		}
		out = append(out, acc)
	}
	return out
}

// sampleInput ist ein kleiner, aufgehender Abschluss mit Vorjahr.
func sampleInput(t *testing.T, accounts []domain.Account) InstanceInput {
	t.Helper()
	prior := accountsWith(t, map[string]domain.Cents{
		"1800": 2_500_000,
		"2900": -2_500_000,
	})
	stmt, err := accounting.BuildStatement(accounts, prior, domain.DepthFull)
	if err != nil {
		t.Fatalf("Bilanz: %v", err)
	}
	return InstanceInput{
		Settings: minimalSettings(), Statement: stmt, Accounts: accounts,
		FiscalYear: 2026,
		StartDate:  "2026-01-01", EndDate: "2026-12-31",
		PriorStartDate: "2025-01-01", PriorEndDate: "2025-12-31",
	}
}

// Die Instanz muss wohlgeformtes XML sein, die Bilanzsumme, das Jahresergebnis,
// beide Kontexte und den Kontennachweis enthalten.
func TestInstanceCarriesTheStatement(t *testing.T) {
	accounts := accountsWith(t, map[string]domain.Cents{
		"1800": 10_000_000,
		"2900": -2_500_000,
		"4400": -7_500_000,
	})
	out, report, err := GenerateEBilanzXBRL(sampleInput(t, accounts))
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !report.CanExport {
		t.Fatalf("der Zuordnungsbericht blockiert: %v", report.Blocking)
	}

	if err := xml.Unmarshal([]byte(out), new(struct{ XMLName xml.Name })); err != nil {
		t.Errorf("die Instanz ist kein wohlgeformtes XML: %v", err)
	}

	for _, want := range []string{
		// Die Bilanzsumme selbst, auf beiden Seiten und mit Vorjahr. Sie steht in
		// keiner Zeile der Gliederung; ohne eigenes Element fehlte sie in der
		// Instanz, und ein Wert einer einzelnen Position, der zufällig gleich hoch
		// ist, ist kein Nachweis dafür, dass sie enthalten wäre.
		`<de-gaap-ci:bs.ass contextRef="ctx_instant" unitRef="EUR" decimals="2">100000.00</de-gaap-ci:bs.ass>`,
		`<de-gaap-ci:bs.ass contextRef="ctx_instant_prior" unitRef="EUR" decimals="2">25000.00</de-gaap-ci:bs.ass>`,
		`<de-gaap-ci:bs.eqLiab contextRef="ctx_instant" unitRef="EUR" decimals="2">100000.00</de-gaap-ci:bs.eqLiab>`,
		`<de-gaap-ci:bs.eqLiab contextRef="ctx_instant_prior" unitRef="EUR" decimals="2">25000.00</de-gaap-ci:bs.eqLiab>`,
		// Das Umlaufvermögen als Gliederungsposition daneben.
		`<de-gaap-ci:bs.ass.currAss contextRef="ctx_instant" unitRef="EUR" decimals="2">100000.00</de-gaap-ci:bs.ass.currAss>`,
		// Jahresergebnis in Bilanz und Staffel.
		`<de-gaap-ci:bs.eqLiab.equity.netIncome contextRef="ctx_instant" unitRef="EUR" decimals="2">75000.00</de-gaap-ci:bs.eqLiab.equity.netIncome>`,
		`<de-gaap-ci:is.netIncome contextRef="ctx_duration" unitRef="EUR" decimals="2">75000.00</de-gaap-ci:is.netIncome>`,
		// Vorjahr als zweiter Kontext.
		`contextRef="ctx_instant_prior"`,
		`contextRef="ctx_duration_prior"`,
		// Kontennachweis mit Position aus der Gliederung.
		"de-gaap-ci:accountAuditProof",
		"<de-gaap-ci:accountNumber>1800</de-gaap-ci:accountNumber>",
		"<de-gaap-ci:accountPosition>Kassenbestand, Bundesbankguthaben, Guthaben bei Kreditinstituten und Schecks</de-gaap-ci:accountPosition>",
		// Pflichtangaben des § 264 Abs. 1a HGB.
		"<de-gcd:genInfo.company.id.register.number contextRef=\"ctx_duration\">HRB 123456</de-gcd:genInfo.company.id.register.number>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("die Instanz enthält %q nicht", want)
		}
	}

	// Der stille Auffang ist entfallen.
	if strings.Contains(out, "bs.other") {
		t.Error("die Instanz enthält noch die Auffangposition bs.other")
	}
}

// Die Namensräume und der Vorbehalt stammen aus der Ressource, nicht aus dem
// Code.
func TestNamespacesAndCaveatComeFromTheResource(t *testing.T) {
	tax, err := LoadTaxonomy()
	if err != nil {
		t.Fatalf("Taxonomie: %v", err)
	}
	accounts := accountsWith(t, map[string]domain.Cents{"1800": 2_500_000, "2900": -2_500_000})
	out, _, err := GenerateEBilanzXBRL(sampleInput(t, accounts))
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	for prefix, uri := range tax.Namespaces {
		if !strings.Contains(out, `xmlns:`+prefix+`="`+uri+`"`) {
			t.Errorf("der Namensraum %s=%q fehlt in der Instanz", prefix, uri)
		}
	}
	if !strings.Contains(out, "esteuer.de") {
		t.Error("die Instanz enthält den Vorbehalt zur amtlichen Taxonomie nicht")
	}
	if !strings.Contains(out, tax.Version) {
		t.Errorf("die Instanz nennt die Taxonomie-Fassung %s nicht", tax.Version)
	}
}

// Ein blockierender Befund verhindert die Erzeugung — die Instanz entsteht gar
// nicht erst.
func TestBlockingFindingPreventsTheExport(t *testing.T) {
	accounts := accountsWith(t, map[string]domain.Cents{"1800": -1_000_000})
	accounts = append(accounts, domain.Account{
		Number: "1234", Name: "Konto ohne Position", Type: domain.AccountTypeAsset,
		PositionID: "bilanz.gibt_es_nicht", DebitSum: 1_000_000,
	})
	stmt, err := accounting.BuildStatement(accounts, nil, domain.DepthFull)
	if err != nil {
		t.Fatalf("Bilanz: %v", err)
	}

	out, report, err := GenerateEBilanzXBRL(InstanceInput{
		Settings: minimalSettings(), Statement: stmt, Accounts: accounts,
		FiscalYear: 2026, StartDate: "2026-01-01", EndDate: "2026-12-31",
	})
	if err == nil {
		t.Fatal("ein Konto ohne Zuordnung muss den Export verhindern")
	}
	if out != "" {
		t.Error("trotz blockierendem Befund ist eine Instanz entstanden")
	}
	if report == nil || report.CanExport {
		t.Fatal("der Zuordnungsbericht meldet keinen blockierenden Befund")
	}
	if len(report.Blocking) != 1 || report.Blocking[0].Account != "1234" {
		t.Fatalf("der Bericht blockiert wegen %v, erwartet das Konto 1234", report.Blocking)
	}
	if !strings.Contains(err.Error(), "1234") {
		t.Errorf("die Fehlermeldung nennt das Konto nicht: %v", err)
	}
}

// Der Zuordnungsbericht führt jedes Konto mit Saldo mit Position, Element und
// dem Vermerk, ob der Elementname geprüft ist.
func TestMappingReportListsEveryAccountWithBalance(t *testing.T) {
	accounts := accountsWith(t, map[string]domain.Cents{
		"1800": 10_000_000,
		"2900": -2_500_000,
		"4400": -7_500_000,
		"1600": 0, // ohne Saldo: gehört nicht in den Nachweis
	})
	stmt, err := accounting.BuildStatement(accounts, nil, domain.DepthFull)
	if err != nil {
		t.Fatalf("Bilanz: %v", err)
	}
	report, err := BuildMappingReport(2026, stmt, accounts)
	if err != nil {
		t.Fatalf("Zuordnungsbericht: %v", err)
	}
	if len(report.Rows) != 3 {
		t.Fatalf("der Bericht führt %d Konten, erwartet die drei mit Saldo", len(report.Rows))
	}
	byAccount := map[string]MappingRow{}
	for _, row := range report.Rows {
		byAccount[row.Account] = row
	}
	if got := byAccount["4400"].Element; got != "de-gaap-ci:is.netSales" {
		t.Errorf("die Erlöse zeigen auf %q, erwartet de-gaap-ci:is.netSales", got)
	}
	if byAccount["2900"].PositionKey != "passiva.A.I" {
		t.Errorf("das gezeichnete Kapital steht auf %q, erwartet passiva.A.I", byAccount["2900"].PositionKey)
	}
	// Solange die Elementnamen nicht gegen die amtliche Taxonomie geprüft sind,
	// muss der Bericht das sagen.
	if report.Unverified != len(report.Rows) {
		t.Errorf("%d von %d Elementen gelten als ungeprüft — erwartet alle",
			report.Unverified, len(report.Rows))
	}
}

// Die Taxonomie-Ressource muss zur Gliederung passen: jede Position des
// Gesetzes braucht ein Element, sonst fiele die Lücke erst beim Export auf.
func TestTaxonomyCoversTheStatementTemplate(t *testing.T) {
	if missing := StatementCoverage(); len(missing) > 0 {
		t.Errorf("für diese Gliederungspositionen fehlt ein Taxonomie-Element: %v", missing)
	}
}

// Der Anlagenspiegel gehört in die Instanz: die Bilanz zeigt einen Buchwert,
// erst der Spiegel zeigt, woraus er entstanden ist (§ 284 Abs. 3 HGB).
func TestAnlagenspiegelReachesTheInstance(t *testing.T) {
	accounts := accountsWith(t, map[string]domain.Cents{
		"0440": 6_900_000,
		"2900": -6_900_000,
	})
	in := sampleInput(t, accounts)
	in.Anlagenspiegel = &domain.Anlagenspiegel{
		FiscalYear: 2026,
		Rows: []domain.AnlagenspiegelRow{{
			Class: domain.AssetClassTangible, Account: "0440", AccountName: "Maschinen",
			CostOpening: 1_000_000, Additions: 200_000, Disposals: 50_000,
			CostClosing: 1_150_000, DepreciationOpening: 250_000, DepreciationYear: 230_000,
			DepreciationDisposal: 20_000, DepreciationClosing: 460_000,
			BookValueOpening: 750_000, BookValueClosing: 690_000,
		}},
		ClassTotals: []domain.AnlagenspiegelRow{{
			Class: domain.AssetClassTangible, AccountName: "Sachanlagen",
			CostClosing: 1_150_000, BookValueClosing: 690_000,
		}},
		Totals: domain.AnlagenspiegelRow{
			AccountName: "Anlagevermögen gesamt",
			CostClosing: 1_150_000, BookValueClosing: 690_000,
		},
	}

	out, _, err := GenerateEBilanzXBRL(in)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	for _, want := range []string{
		"fixedAssetsMovement",
		"fixedAssetsMovementSubtotal",
		"fixedAssetsMovementTotal",
		"de-gaap-ci:bs.ass.fixAss.tan.techPlant",
		`<de-gaap-ci:histCost.end contextRef="ctx_duration" unitRef="EUR" decimals="2">11500.00</de-gaap-ci:histCost.end>`,
		`<de-gaap-ci:netBookValue.end contextRef="ctx_duration" unitRef="EUR" decimals="2">6900.00</de-gaap-ci:netBookValue.end>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("die Instanz enthält %q nicht", want)
		}
	}
}

// Ohne Anlagevermögen bleibt der Block weg — eine leere Aufstellung wäre eine
// Aussage, die niemand getroffen hat.
func TestEmptyAnlagenspiegelIsOmitted(t *testing.T) {
	accounts := accountsWith(t, map[string]domain.Cents{"1800": 2_500_000, "2900": -2_500_000})
	out, _, err := GenerateEBilanzXBRL(sampleInput(t, accounts))
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if strings.Contains(out, "fixedAssetsMovement") {
		t.Error("ohne Anlagenspiegel darf kein Anlagenspiegel-Block entstehen")
	}
}

// Aufwendungen stehen in der Gliederung als negativer Beitrag zum Ergebnis, in
// der Taxonomie als positiver Betrag. Ein Materialaufwand darf in der Instanz
// deshalb nicht mit Minus erscheinen, und das Jahresergebnis bleibt davon
// unberührt.
func TestExpensesAreReportedPositive(t *testing.T) {
	accounts := accountsWith(t, map[string]domain.Cents{
		"1800": 7_500_000,
		"2900": -2_500_000,
		"4400": -7_500_000,
		"5200": 2_500_000, // Wareneingang: Aufwand, Sollsaldo
	})
	out, _, err := GenerateEBilanzXBRL(sampleInput(t, accounts))
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	for _, want := range []string{
		`<de-gaap-ci:is.materialCost contextRef="ctx_duration" unitRef="EUR" decimals="2">25000.00</de-gaap-ci:is.materialCost>`,
		`<de-gaap-ci:is.netIncome contextRef="ctx_duration" unitRef="EUR" decimals="2">50000.00</de-gaap-ci:is.netIncome>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("die Instanz enthält %q nicht", want)
		}
	}
	if strings.Contains(out, `>-25000.00</de-gaap-ci:is.materialCost>`) {
		t.Error("der Materialaufwand steht mit negativem Vorzeichen in der Instanz")
	}
}
