package ebilanz

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func minimalSettings() *domain.CompanySettings {
	return &domain.CompanySettings{
		CompanyName: "Muster GmbH",
		LegalForm:   "GmbH",
		TaxNumber:   "12/345/67890",
		FiscalYear:  2026,
	}
}

// Der Anlagenspiegel ist Bestandteil des Anhangs (§ 284 Abs. 3 HGB). Steht er
// nicht in der Instanz, zeigt die E-Bilanz einen Buchwert, ohne zu zeigen,
// woraus er entstanden ist.
func TestAnlagenspiegelReachesTheInstance(t *testing.T) {
	spiegel := &domain.Anlagenspiegel{
		FiscalYear: 2026,
		Rows: []domain.AnlagenspiegelRow{{
			Class: domain.AssetClassTangible, Account: "0440", AccountName: "Maschinen",
			CostOpening: 1_000_000, Additions: 200_000, Disposals: 50_000, Transfers: 0,
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

	out, err := GenerateEBilanzXBRL(minimalSettings(), nil, &domain.FinancialSummary{}, spiegel)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	for _, want := range []string{
		"fixedAssetsMovement",
		"fixedAssetsMovementSubtotal",
		"fixedAssetsMovementTotal",
		"de-gaap-ci:bs.ass.fixAss.tan.techPlant",
		"<de-gaap-ci:histCost.end unitRef=\"EUR\" decimals=\"2\">11500.00</de-gaap-ci:histCost.end>",
		"<de-gaap-ci:netBookValue.end unitRef=\"EUR\" decimals=\"2\">6900.00</de-gaap-ci:netBookValue.end>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("die Instanz enthält %q nicht", want)
		}
	}
	if err := xml.Unmarshal([]byte(out), new(struct{ XMLName xml.Name })); err != nil {
		t.Errorf("die Instanz ist kein wohlgeformtes XML: %v", err)
	}
}

// Ohne Anlagevermögen bleibt der Block weg — eine leere Aufstellung wäre eine
// Aussage, die niemand getroffen hat.
func TestEmptyAnlagenspiegelIsOmitted(t *testing.T) {
	out, err := GenerateEBilanzXBRL(minimalSettings(), nil, &domain.FinancialSummary{}, nil)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if strings.Contains(out, "fixedAssetsMovement") {
		t.Error("ohne Anlagenspiegel darf kein Anlagenspiegel-Block entstehen")
	}
	if err := xml.Unmarshal([]byte(out), new(struct{ XMLName xml.Name })); err != nil {
		t.Errorf("die Instanz ist kein wohlgeformtes XML: %v", err)
	}
}

// Jedes Konto des Anlagenkatalogs braucht eine Taxonomieposition. Ohne sie
// landete es auf einer Sammelposition und wäre im Nachweis nicht mehr zu finden.
func TestFixedAssetAccountsAreMapped(t *testing.T) {
	for _, account := range []string{
		"0110", "0135", "0150", "0170",
		"0215", "0240", "0420", "0440", "0520", "0540", "0630", "0670", "0675", "0700",
		"0800", "0810", "0820", "0900", "0920", "0930", "0940", "0980", "0990",
	} {
		position, ok := skr04ToXBRL[account]
		if !ok {
			t.Errorf("Anlagekonto %s hat keine Taxonomieposition", account)
			continue
		}
		if !strings.HasPrefix(position, "de-gaap-ci:bs.ass.fixAss.") {
			t.Errorf("Anlagekonto %s zeigt auf %q — erwartet eine Position des Anlagevermögens",
				account, position)
		}
	}
}
