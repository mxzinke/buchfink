package ebilanz

import (
	"regexp"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Alle bebuchbaren Konten in einer Instanz: jede Rechenregel, jedes Mussfeld
// und die Darstellungshierarchie halten. Befunde bleiben nur für Positionen,
// die in der Einheitsbilanz steuerlich unzulässig sind.
func TestAllAccountsInOneInstance(t *testing.T) {
	for _, appropriated := range []bool{false, true} {
		for _, year := range []string{"2025", "2027"} {
			balances := map[string]domain.Cents{}
			var sum domain.Cents
			for _, acc := range bookableAccounts(t) {
				if withoutPosition[acc.Number] || acc.Number == "1800" {
					continue
				}
				key, _ := accounting.StatementKeyForAccount(withBalance(acc, 100))
				if key == "passiva.A.VI" && !appropriated {
					continue
				}
				debit := domain.Cents(100)
				if strings.HasPrefix(key, "passiva.") || strings.HasPrefix(key, "guv.") && !accounting.IsExpensePosition(key) {
					debit = -100
				}
				balances[acc.Number] = debit
				sum += debit
			}
			balances["1800"] = -sum
			res, err := Generate(exampleInput(t, balances, year))
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range res.Report.Findings {
				if f.Severity == SeverityBlocking && f.Code != "steuerlich-unzulaessig" {
					t.Errorf("%s (Ergebnisverwendung %v) %s: %s %s %v", year, appropriated, f.Code, f.Message, f.Element, f.Accounts)
				}
			}
			// Selbst geschaffene immaterielle Vermögensgegenstände (0143) sind
			// steuerlich nicht ansetzbar (§ 5 Abs. 2 EStG).
			if !hasFinding(res.Report, "steuerlich-unzulaessig") || len(res.Report.Rows) < 1000 {
				t.Errorf("%s: %d Konten, Befund zu steuerlich unzulässigen Positionen fehlt", year, len(res.Report.Rows))
			}
		}
	}
}

// Ab Fassung 6.10 bekommt jede werthaltige Blattposition ohne Konto ein Tupel
// ohne Konteninformationen (Leitfaden 6.10 B.12.1): hier der
// Jahresüberschuss der Bilanz, den kein Konto trägt.
func TestVersion610ReportsDerivedPositionsInTheAccountProof(t *testing.T) {
	in := exampleInput(t, exampleBalances, "2026")
	in.Version = "6.10"
	res := generateOK(t, in)
	text := string(res.Instance)
	if !strings.Contains(text, "de-gaap-ci-2026-04-01-shell-fiscal.xsd") {
		t.Fatal("Instanz verweist nicht auf das Einstiegsschema 6.10")
	}
	want := `<de-gaap-ci:detailedInformation.accountbalances.positionName contextRef="D-AKTJAHR">de-gaap-ci:bs.eqLiab.equity.netIncome</de-gaap-ci:detailedInformation.accountbalances.positionName>`
	if !strings.Contains(text, want) {
		t.Error("kein Tupel für den Jahresüberschuss der Bilanz")
	}
	if n := strings.Count(text, `notPossible contextRef="D-AKTJAHR">true<`); n != 1 {
		t.Errorf("%d Tupel ohne Konteninformationen, erwartet 1", n)
	}
	if n := strings.Count(text, `notPossible contextRef="D-AKTJAHR">false<`); n != len(exampleBalances) {
		t.Errorf("%d Tupel mit Konteninformationen, erwartet %d", n, len(exampleBalances))
	}
}

// Ergebnisverwendung (§ 268 Abs. 1 HGB): Gewinnvortrag 4.000 und
// Vorabausschüttung 5.000 machen aus dem Jahresüberschuss von 20.000 einen
// Bilanzgewinn von 19.000 (Leitfaden B.9.1.1, B.14.3.2.3.4.2).
func TestAppropriationShowsTheBalanceSheetProfit(t *testing.T) {
	balances := withChanges(exampleBalances, map[string]domain.Cents{
		"2970": -400000, "7790": 500000, "1800": -100000,
	})
	in := exampleInput(t, balances, "2025")
	priorEquity, priorPaidIn := domain.Cents(2900000), domain.Cents(2500000)
	in.PriorEquity, in.PriorPaidIn = &priorEquity, &priorPaidIn
	res := generateOK(t, in)
	f := facts(t, res.Instance)
	want := map[string]string{
		"incomeUse.gainLoss":                                 "19000.00",
		"incomeUse.gainLoss.netIncome":                       "20000.00",
		"incomeUse.gainLoss.retainedEarningsPrevYear":        "4000.00",
		"incomeUse.gainLoss.dividensPlanned":                 "5000.00",
		"bs.eqLiab.equity.profitLoss":                        "19000.00",
		"bs.eqLiab.equity.netIncome":                         "NIL",
		"bs.eqLiab.equity.retainedEarnings":                  "NIL",
		"bs.eqLiab.equity":                                   "44000.00",
		"genInfo.report.id.incomeStatementendswithBalProfit": "true",
		"genInfo.report.id.reportElement.reportElements.EV":  "Ergebnisverwendung",
		// Betriebsvermögensvergleich: 44.000 − 29.000 + Ausschüttung 5.000.
		"BVV.profitLoss":                   "20000.00",
		"BVV.profitLoss.withdrawalDistrib": "5000.00",
	}
	for name, v := range want {
		got := f[name]
		if strings.HasPrefix(name, "genInfo.") {
			got = gcdValue(string(res.Instance), name)
		}
		if got != v {
			t.Errorf("%s = %q, erwartet %q", name, got, v)
		}
	}
}

// GuV nach MicroBilG (§ 275 Abs. 5 HGB): die Bestandserhöhung geht zu den
// sonstigen Erträgen, die Instanz verweist auf das MicroBilG-Schema.
func TestMicroBilGStatementRoutesTheInventoryChange(t *testing.T) {
	balances := withChanges(exampleBalances, map[string]domain.Cents{"4800": -200000, "1110-1139": 200000})
	in := exampleInput(t, balances, "2025")
	in.MicroBilG = true
	res := generateOK(t, in)
	f := facts(t, res.Instance)
	want := map[string]string{
		"ismi.netIncome": "22000.00",
		"ismi.netIncome.regular.operatingTC.grossTradingProfit.totalOutput.inventoryIncrease": "2000.00",
		"ismi.netIncome.otherOpRevenue":                               "2000.00",
		"ismi.netIncome.netSales":                                     "50000.00",
		"bs.eqLiab.equity.netIncome":                                  "22000.00",
		"genInfo.report.id.reportElement.reportElements.GuVMicroBilG": "GuV nach MicroBilG",
	}
	for name, v := range want {
		got := f[name]
		if strings.HasPrefix(name, "genInfo.") {
			got = gcdValue(string(res.Instance), name)
		}
		if got != v {
			t.Errorf("%s = %q, erwartet %q", name, got, v)
		}
	}
	if gcdValue(string(res.Instance), "genInfo.report.id.reportElement.reportElements.GuV") != "NIL" {
		t.Error("GuV und GuV nach MicroBilG zugleich angekündigt")
	}
	if !strings.Contains(string(res.Instance), "shell-fiscal-microbilg.xsd") {
		t.Error("Instanz verweist nicht auf das MicroBilG-Schema")
	}
}

func TestMicroBilGNeedsAMicroCompany(t *testing.T) {
	in := exampleInput(t, exampleBalances, "2025")
	in.MicroBilG = true
	in.SizeClass = domain.SizeSmall
	res, err := Generate(in)
	if err != nil {
		t.Fatal(err)
	}
	if res.Report.CanExport || !hasFinding(res.Report, "microbilg") {
		t.Fatalf("GuV nach MicroBilG für eine kleine GmbH nicht abgewiesen: %+v", res.Report.Findings)
	}
}

// Eröffnungsbilanz (§ 242 Abs. 1 HGB): Bilanz ohne GuV und BVV, Beginn des
// Wirtschaftsjahres gleich Stichtag (Leitfaden B.14.3.1.6, B.14.3.1.10).
func TestOpeningBalanceSheet(t *testing.T) {
	in := exampleInput(t, map[string]domain.Cents{"1800": 2500000, "2900": -2500000}, "2025")
	in.Kind, in.ClosingDate, in.FiscalYearBegin = KindOpening, "2025-03-15", "2025-03-15"
	in.Statement = nil
	res := generateOK(t, in)
	f := facts(t, res.Instance)
	want := map[string]string{
		"bs.ass": "25000.00", "bs.eqLiab.equity.subscribed.corp": "25000.00",
		"genInfo.report.period.fiscalYearBegin":              "2025-03-15",
		"genInfo.report.period.balSheetClosingDate":          "2025-03-15",
		"genInfo.report.id.reportElement.reportElements.GuV": "NIL",
		"genInfo.report.id.reportElement.reportElements.BVV": "NIL",
		"genInfo.report.id.statementType.statementType.EB":   "Eröffnungsbilanz",
	}
	for name, v := range want {
		got := f[name]
		if strings.HasPrefix(name, "genInfo.") {
			got = gcdValue(string(res.Instance), name)
		}
		if got != v {
			t.Errorf("%s = %q, erwartet %q", name, got, v)
		}
	}
	if _, ok := f["is.netIncome"]; ok {
		t.Error("Eröffnungsbilanz mit GuV")
	}
}

// Handelsbilanz mit Überleitung: die Drohverlustrückstellung von 2.000 ist
// steuerlich unzulässig (§ 5 Abs. 4a EStG). Die Überleitung nimmt sie heraus,
// der Ausgleichsposten trägt die Differenz, und der steuerliche Gewinn ist um
// 2.000 höher (Leitfaden B.15, B.14.3.2.3.4.1).
func TestTransferRemovesAPendingLossProvision(t *testing.T) {
	balances := withChanges(exampleBalances, map[string]domain.Cents{"3092": -200000, "6300": 200000})
	in := exampleInput(t, balances, "2025")
	in.Adjustments = []Adjustment{{Reason: "Drohverlustrückstellung § 5 Abs. 4a EStG", Account: "3092", IncomeAccount: "6300", Current: 200000}}
	res := generateOK(t, in)
	text := string(res.Instance)
	if res.Report.AccountingStandard != "HGBM" || !strings.Contains(text, "accountingStandard.HGBM") {
		t.Fatal("Handelsbilanz nicht als HGBM gekennzeichnet")
	}
	f := facts(t, res.Instance)
	if f["is.netIncome"] != "18000.00" || f["BVV.profitLoss"] != "20000.00" || f["BVV.profitLoss.assetsCurrentYear"] != "45000.00" {
		t.Errorf("JÜ %s, BVV-Gewinn %s, BV %s; erwartet 18000.00, 20000.00, 45000.00",
			f["is.netIncome"], f["BVV.profitLoss"], f["BVV.profitLoss.assetsCurrentYear"])
	}
	for _, s := range []string{
		"de-gaap-ci:bs.eqLiab.accruals.other.imminentLoss.other</de-gaap-ci:hbst.transfer.bsEqLiab.name>",
		"de-gaap-ci:bs.eqLiab.equity.netIncome.taxBalanceGenerally.transferDiffTaxAccounts</de-gaap-ci:hbst.transfer.bsEqLiab.name>",
		"de-gaap-ci:is.netIncome.regular.operatingTC.otherCost.miscellaneous</de-gaap-ci:hbst.transfer.isChangeNetIncome.name>",
		"de-gaap-ci:is.netIncome</de-gaap-ci:hbst.transfer.isChangeNetIncome.name>",
	} {
		if !strings.Contains(text, s) {
			t.Errorf("Überleitung ohne %s", s)
		}
	}
	// Die Passivseite gleicht sich innerhalb des Tupels aus.
	if strings.Contains(text, "de-gaap-ci:bs.eqLiab</de-gaap-ci:hbst.transfer.bsEqLiab.name>") {
		t.Error("Summe Passiva ohne Änderung übergeleitet")
	}
	if got := strings.Count(text, "<de-gaap-ci:hbst.transfer>"); got != 1 {
		t.Errorf("%d Überleitungstupel, erwartet 1", got)
	}
}

// Sonderabschreibung nach § 7g Abs. 5 EStG: steuerlich 6.000 weniger
// Anlagevermögen, Aufwand auf der Position der Sonderabschreibungen.
func TestTransferCarriesASpecialDepreciation(t *testing.T) {
	in := exampleInput(t, exampleBalances, "2025")
	in.Adjustments = []Adjustment{{Reason: "§ 7g Abs. 5 EStG", Account: "0520", IncomeAccount: "6242", Current: -600000}}
	res := generateOK(t, in)
	text := string(res.Instance)
	for _, s := range []string{
		"de-gaap-ci:bs.ass.fixAss.tan.otherEquipm.passengerCars</de-gaap-ci:hbst.transfer.bsAss.name>",
		"de-gaap-ci:bs.ass</de-gaap-ci:hbst.transfer.bsAss.name>",
		"de-gaap-ci:is.netIncome.regular.operatingTC.deprAmort.fixAss.specific.impairment</de-gaap-ci:hbst.transfer.isChangeNetIncome.name>",
	} {
		if !strings.Contains(text, s) {
			t.Errorf("Überleitung ohne %s", s)
		}
	}
	f := facts(t, res.Instance)
	if f["BVV.profitLoss"] != "14000.00" {
		t.Errorf("steuerlicher Gewinn %s, erwartet 14000.00", f["BVV.profitLoss"])
	}
}

func hasFinding(r *MappingReport, code string) bool {
	for _, f := range r.Findings {
		if f.Code == code && f.Severity == SeverityBlocking {
			return true
		}
	}
	return false
}

// Befunde, die den Export verhindern: fehlende Steuernummer, eine steuerlich
// unzulässige Position ohne Überleitung, ein Konto der Personengesellschaft.
func TestBlockingFindingsNameElementAndAccounts(t *testing.T) {
	balances := withChanges(exampleBalances, map[string]domain.Cents{"3092": -200000, "6300": 200000, "2000": -100, "1800": 100})
	in := exampleInput(t, balances, "2025")
	in.Settings.TaxNumberElster = ""
	res, err := Generate(in)
	if err != nil {
		t.Fatal(err)
	}
	if res.Report.CanExport || res.Instance != nil {
		t.Fatal("Export trotz Befunden")
	}
	for _, code := range []string{"steuernummer", "steuerlich-unzulaessig", "konto"} {
		if !hasFinding(res.Report, code) {
			t.Errorf("Befund %s fehlt: %+v", code, res.Report.Findings)
		}
	}
	for _, f := range res.Report.Findings {
		if f.Code == "steuerlich-unzulaessig" && (f.Element != pAcc+"other.imminentLoss.other" || len(f.Accounts) != 1 || f.Accounts[0] != "3092") {
			t.Errorf("Befund nennt %s %v, erwartet imminentLoss.other und 3092", f.Element, f.Accounts)
		}
	}
	if err := res.Report.BlockingError(); err == nil || !strings.Contains(err.Error(), "3092") {
		t.Errorf("Fehlermeldung nennt das Konto nicht: %v", err)
	}
}

// Die Rechenprüfung findet einen Cent Abweichung.
func TestCalculationCheckIsExactToTheCent(t *testing.T) {
	tax, err := LoadTaxonomy("6.9")
	if err != nil {
		t.Fatal(err)
	}
	ds := newDataset(tax)
	ds.add(pCash+"bank", 100)
	ds.add(pCash+"cash", 50)
	ds.rollUp(RoleBalanceSheet, "bs.ass")
	if gaps := ds.inconsistencies(RoleBalanceSheet); len(gaps) != 0 {
		t.Fatalf("Abweichung ohne Grund: %+v", gaps)
	}
	ds.values["bs.ass.currAss.cashEquiv"]++
	gaps := ds.inconsistencies(RoleBalanceSheet)
	if len(gaps) != 2 {
		t.Fatalf("%d Abweichungen, erwartet 2 (die Position und ihre Oberposition): %+v", len(gaps), gaps)
	}
}

func TestEnvelopeCarriesTheClosingDate(t *testing.T) {
	res := generateOK(t, exampleInput(t, exampleBalances, "2025"))
	out, err := Envelope(res.Instance, "2025-12-31")
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, "<ebilanz:stichtag>20251231</ebilanz:stichtag>") || strings.Count(text, "<?xml") != 1 ||
		!strings.Contains(text, "<xbrli:xbrl") {
		t.Errorf("Hülle unvollständig:\n%s", text[:300])
	}
	facts(t, out)
}

// gcdValue liest einen Wert der Stammdaten, auch innerhalb von Tupeln: den
// Text, „NIL" oder leer, wenn das Element fehlt.
func gcdValue(instance, name string) string {
	re := regexp.MustCompile(`<de-gcd:` + regexp.QuoteMeta(name) + `( [^>]*)?>([^<]*)<`)
	m := re.FindStringSubmatch(instance)
	if m == nil {
		return ""
	}
	if strings.Contains(m[1], `xsi:nil="true"`) {
		return "NIL"
	}
	return m[2]
}

// Ohne Gesellschafterliste und Hausnummer gehen die Mussfelder leer, das Tupel
// des Gesellschafters bleibt nach seinem Schema vollständig.
func TestMissingShareholdersAndHouseNumberGoAsNil(t *testing.T) {
	in := exampleInput(t, exampleBalances, "2025")
	in.Settings.Shareholders, in.Settings.HouseNumber = nil, ""
	res := generateOK(t, in)
	text := string(res.Instance)
	for _, name := range []string{
		"genInfo.company.id.shareholder.name",
		"genInfo.company.id.shareholder.ShareDivideKey.numerator",
		"genInfo.company.id.shareholder.ShareDivideKey.denominator",
		"genInfo.company.id.location.houseNo",
	} {
		if got := gcdValue(text, name); got != "NIL" {
			t.Errorf("%s = %q, erwartet NIL", name, got)
		}
	}
}
