package ebilanz

import (
	"encoding/xml"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Beispiel: eine GmbH im Wirtschaftsjahr 2025 mit Pkw, Forderungen samt
// Einzelwertberichtigung, Bank, Umsatzsteuer und Körperschaftsteuer.
//
//	Erlöse 50.000 − Waren 10.000 − Gehälter 8.000 − AfA 5.000 − EWB 1.000
//	− Miete 3.000 − KSt 3.000 = Jahresüberschuss 20.000
//	Aktiva: Pkw 30.000 + Forderungen 11.900 − EWB 1.000 + Vorsteuer 1.140
//	        + Bank 21.410 = 63.450
//	Passiva: Stammkapital 25.000 + JÜ 20.000 + KSt-Rückstellung 3.000
//	        + Lieferanten 5.950 + Umsatzsteuer 9.500 = 63.450
var exampleBalances = map[string]domain.Cents{
	"0520": 3000000, "1200": 1190000, "1246": -100000, "1406": 114000, "1800": 2141000,
	"2900": -2500000, "3040": -300000, "3300": -595000, "3806": -950000,
	"4400-4409": -5000000, "5400-5409": 1000000, "6020": 800000, "6222": 500000,
	"6923": 100000, "6310": 300000, "7600": 300000,
}

func exampleAccounts(t *testing.T, balances map[string]domain.Cents) []domain.Account {
	t.Helper()
	catalog := map[string]domain.Account{}
	for _, acc := range accounting.DefaultSKR04Accounts() {
		catalog[acc.Number] = acc
	}
	var out []domain.Account
	for number, debit := range balances {
		acc, ok := catalog[number]
		if !ok {
			t.Fatalf("Konto %s fehlt im Katalog", number)
		}
		out = append(out, withBalance(acc, debit))
	}
	return out
}

func exampleSettings() *domain.CompanySettings {
	return &domain.CompanySettings{
		CompanyName: "Muster", LegalForm: domain.LegalFormGmbH, TaxNumberElster: "5192050001265",
		Street: "Musterstraße", HouseNumber: "1", PostalCode: "40210", City: "Düsseldorf", CountryCode: "DE",
		Shareholders: []domain.CompanyShareholder{{Name: "Anna Muster", ShareCapital: 1500000}, {Name: "Bernd Muster", ShareCapital: 1000000}},
	}
}

func exampleInput(t *testing.T, balances map[string]domain.Cents, year string) Input {
	t.Helper()
	accounts := exampleAccounts(t, balances)
	stmt, err := accounting.BuildStatement(accounts, nil, domain.DepthFull)
	if err != nil {
		t.Fatal(err)
	}
	zero := domain.Cents(0)
	return Input{
		Settings: exampleSettings(), Statement: stmt, Accounts: accounts, Kind: KindAnnual,
		FiscalYearBegin: year + "-01-01", FiscalYearEnd: year + "-12-31",
		SizeClass: domain.SizeMicro, PriorEquity: &zero, PriorPaidIn: &zero,
	}
}

func generateOK(t *testing.T, in Input) *Result {
	t.Helper()
	res, err := Generate(in)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Report.CanExport {
		for _, f := range res.Report.Findings {
			t.Logf("%s %s: %s (%s) %v", f.Severity, f.Code, f.Message, f.Element, f.Accounts)
		}
		t.Fatalf("Export blockiert")
	}
	if dir := os.Getenv("EBILANZ_WRITE_DIR"); dir != "" {
		name := strings.ReplaceAll(t.Name(), "/", "_") + "-" + res.Version.Version + ".xml"
		if err := os.WriteFile(dir+"/"+name, res.Instance, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return res
}

// facts liest die Werte der Instanz (ohne Tupel) als Name → Text; NIL als „NIL".
func facts(t *testing.T, instance []byte) map[string]string {
	t.Helper()
	var doc struct {
		Items []struct {
			XMLName xml.Name
			Nil     string `xml:"http://www.w3.org/2001/XMLSchema-instance nil,attr"`
			Value   string `xml:",chardata"`
		} `xml:",any"`
	}
	if err := xml.Unmarshal(instance, &doc); err != nil {
		t.Fatalf("Instanz ist kein wohlgeformtes XML: %v", err)
	}
	out := map[string]string{}
	for _, it := range doc.Items {
		v := strings.TrimSpace(it.Value)
		if it.Nil == "true" {
			v = "NIL"
		}
		out[it.XMLName.Local] = v
	}
	return out
}

func TestExampleInstanceCarriesTheStatement(t *testing.T) {
	res := generateOK(t, exampleInput(t, exampleBalances, "2025"))
	if res.Version.Version != "6.9" {
		t.Fatalf("Fassung %s, erwartet 6.9", res.Version.Version)
	}
	f := facts(t, res.Instance)
	want := map[string]string{
		"bs.ass":    "63450.00",
		"bs.eqLiab": "63450.00",
		"bs.ass.fixAss.tan.otherEquipm.passengerCars": "30000.00",
		// Forderungen nach Einzelwertberichtigung: 11.900 − 1.000.
		"bs.ass.currAss.receiv.trade": "10900.00",
		"bs.eqLiab.equity":            "45000.00",
		"bs.eqLiab.equity.netIncome":  "20000.00",
		"is.netIncome":                "20000.00",
		"is.netIncome.regular.operatingTC.grossTradingProfit.totalOutput.netSales": "50000.00",
		"is.netIncome.tax.kst":             "3000.00",
		"BVV.profitLoss":                   "20000.00",
		"BVV.profitLoss.assetsCurrentYear": "45000.00",
		"BVV.profitLoss.contribution":      "25000.00",
		// Mussfelder ohne Konto gehen leer (Leitfaden B.14.2.3).
		"is.netIncome.regular.operatingTC.otherCost.energyCost": "NIL",
		"bs.eqLiab.equity.retainedEarnings":                     "NIL",
		"genInfo.report.period.fiscalYearEnd":                   "2025-12-31",
		"genInfo.report.id.incomeStatementendswithBalProfit":    "false",
		"genInfo.company.id.location.country":                   "Deutschland",
	}
	for name, v := range want {
		if f[name] != v {
			t.Errorf("%s = %q, erwartet %q", name, f[name], v)
		}
	}
	text := string(res.Instance)
	for _, s := range []string{
		`<xbrli:identifier scheme="http://www.rzf-nrw.de/Steuernummer">5192050001265</xbrli:identifier>`,
		`de-gcd-2025-04-01-shell.xsd`, `de-gaap-ci-2025-04-01-shell-fiscal.xsd`,
		`<de-gcd:genInfo.company.id.idNo.type.companyId.BF4 contextRef="D-AKTJAHR">5192</de-gcd:genInfo.company.id.idNo.type.companyId.BF4>`,
		`<de-gcd:genInfo.company.id.legalStatus.legalStatus.GMBH contextRef="D-AKTJAHR">`,
		`<de-gcd:genInfo.company.id.shareholder.ShareDivideKey.numerator contextRef="D-AKTJAHR">3</de-gcd:genInfo.company.id.shareholder.ShareDivideKey.numerator>`,
		`<de-gcd:genInfo.report.id.accountingStandard.accountingStandard.HAOE contextRef="D-AKTJAHR">`,
	} {
		if !strings.Contains(text, s) {
			t.Errorf("Instanz enthält nicht %s", s)
		}
	}
	// Kontennachweis: je Konto ein Tupel mit dem Saldo in Richtung der
	// Position; die Einzelwertberichtigung mindert die Forderungen.
	if n := strings.Count(text, "<de-gaap-ci:detailedInformation.accountBalances>"); n != len(exampleBalances) {
		t.Errorf("%d Tupel im Kontennachweis, erwartet %d", n, len(exampleBalances))
	}
	ewb := regexp.MustCompile(`(?s)accountNumber contextRef="D-AKTJAHR">1246</[^>]+>\s*<[^>]+>[^<]*</[^>]+>\s*<de-gaap-ci:detailedInformation.accountbalances.amount[^>]*>(-?[0-9.]+)<`)
	if m := ewb.FindStringSubmatch(text); m == nil || m[1] != "-1000.00" {
		t.Errorf("Kontennachweis 1246: %v, erwartet -1000.00", m)
	}
}

func withChanges(base map[string]domain.Cents, changes map[string]domain.Cents) map[string]domain.Cents {
	out := map[string]domain.Cents{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range changes {
		out[k] += v
		if out[k] == 0 {
			delete(out, k)
		}
	}
	return out
}
