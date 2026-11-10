package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Drei Projekte am 31.12.:
//
//	Shop:   Fremdleistung 8.000 €, eigene Stunden 4.000 €, Gemeinkosten 1.500 €,
//	        Verwaltung 500 € ohne Wahlrecht → Herstellungskosten 13.500 €;
//	        Auftrag 30.000 €, Restkosten 10.000 € → kein Abschlag; Anzahlung
//	        10.000 € auf 3272 offen abgesetzt.
//	App:    Kosten 20.000 € plus Verwaltung 2.000 € mit Wahlrecht = 22.000 €;
//	        Auftrag 25.000 €, Restkosten 8.000 € → beizulegender Wert 17.000 €,
//	        Abschlag 5.000 €; Anzahlung 20.000 € höchstens bis 17.000 € absetzbar.
//	Portal: Kosten 6.000 €; Auftrag 5.000 €, Restkosten 7.000 € → Wert 0 €,
//	        drohender Verlust über den Bestand hinaus 2.000 €.
func TestWorkInProgressIsValuedAtCostWithoutLoss(t *testing.T) {
	result, err := CalculateWorkInProgress([]WorkInProgressProject{
		{Name: "Shop", DirectCosts: 1_200_000, Overheads: 150_000, AdministrativeCosts: 50_000,
			ContractValue: 3_000_000, CostsToComplete: 1_000_000, AdvancesReceived: 1_000_000},
		{Name: "App", DirectCosts: 2_000_000, AdministrativeCosts: 200_000, IncludeAdministrative: true,
			ContractValue: 2_500_000, CostsToComplete: 800_000, AdvancesReceived: 2_000_000,
			AdvanceAccount: AccountErhalteneAnzahlungen19},
		{Name: "Portal", DirectCosts: 600_000, ContractValue: 500_000, CostsToComplete: 700_000},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]WorkInProgressValuation{}
	for _, p := range result.Projects {
		byName[p.Name] = p
	}
	if p := byName["Shop"]; p.ProductionCost != 1_350_000 || p.Value != 1_350_000 || p.AdvanceOffset != 1_000_000 {
		t.Errorf("Shop: %+v", p)
	}
	if p := byName["App"]; p.ProductionCost != 2_200_000 || p.Value != 1_700_000 || p.WriteDown != 500_000 ||
		p.AdvanceOffset != 1_700_000 {
		t.Errorf("App: %+v", p)
	}
	if p := byName["Portal"]; p.Value != 0 || p.WriteDown != 600_000 || p.ExpectedLoss != 200_000 {
		t.Errorf("Portal: %+v", p)
	}
	if result.Total != 3_050_000 || result.ExpectedLosses != 200_000 ||
		result.AdvanceOffsets[AccountErhalteneAnzahlungen19] != 2_700_000 {
		t.Errorf("Bestand %s €, drohende Verluste %s €, abgesetzt %v",
			result.Total, result.ExpectedLosses, result.AdvanceOffsets)
	}

	// Ohne Absetzung bleiben die Anzahlungen unter den Verbindlichkeiten.
	kept, err := CalculateWorkInProgress([]WorkInProgressProject{
		{Name: "Shop", DirectCosts: 100_000, AdvancesReceived: 50_000},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(kept.AdvanceOffsets) != 0 || kept.Projects[0].AdvanceOffset != 0 {
		t.Errorf("Anzahlungen abgesetzt, obwohl passiviert: %+v", kept)
	}
	if _, err := CalculateWorkInProgress([]WorkInProgressProject{
		{Name: "Shop", DirectCosts: 100_000, AdvancesReceived: 50_000, AdvanceAccount: "1800"},
	}, true); err == nil {
		t.Error("ein Bankkonto ist kein Anzahlungskonto")
	}
	if _, err := CalculateWorkInProgress([]WorkInProgressProject{{Name: " ", DirectCosts: 1}}, false); err == nil {
		t.Error("ein Projekt ohne Namen wurde bewertet")
	}
}

func TestWorkInProgressAccountsExist(t *testing.T) {
	chart := NewChart(DefaultSKR04Accounts())
	for _, account := range []string{AccountUnfertigeLeistungen, AccountBestandsveraenderungUL,
		AccountAnzahlungenVonVorraetenOffen} {
		if _, ok := chart.Lookup(account); !ok {
			t.Errorf("Konto %s steht nicht im SKR04-Katalog", account)
		}
	}
}

// Die offen abgesetzte Anzahlung steht mit Habensaldo unter den Vorräten und
// mindert sie (§ 268 Abs. 5 Satz 2 HGB); das ist kein Befund zum Vorzeichen.
func TestAdvancesDeductedFromInventoryReduceTheInventory(t *testing.T) {
	stmt, err := BuildStatement(accountsWith(t, map[string]domain.Cents{
		"2900": -2_500_000,
		"1800": 2_150_000,
		"1080": 1_350_000,
		"1190": -1_000_000,
	}), nil, domain.DepthFull)
	if err != nil {
		t.Fatal(err)
	}
	if got := lineAmount(t, stmt, "aktiva.B.I.anzahlungen"); got != -1_000_000 {
		t.Errorf("abgesetzte Anzahlungen %s € — erwartet -10.000,00 €", got)
	}
	if got := lineAmount(t, stmt, "aktiva.B.I"); got != 350_000 {
		t.Errorf("Vorräte %s € — erwartet 3.500,00 €", got)
	}
	if got := lineAmount(t, stmt, "passiva.C.3"); got != 0 {
		t.Errorf("die abgesetzte Anzahlung steht zusätzlich unter den Verbindlichkeiten: %s €", got)
	}
	if len(stmt.Assignment.WrongSign) != 0 {
		t.Errorf("Befunde zum Vorzeichen: %+v", stmt.Assignment.WrongSign)
	}
	if stmt.BalanceSheetTotal != 2_500_000 {
		t.Errorf("Bilanzsumme %s € — erwartet 25.000,00 €", stmt.BalanceSheetTotal)
	}
}
