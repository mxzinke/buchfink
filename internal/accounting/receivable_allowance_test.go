package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// allowanceItems sind die offenen Posten zum 31.12.2026:
//
//	A  11.900 € brutto, 19 %, fällig 15.08.2026 → 138 Tage überfällig, netto 10.000 €
//	B   5.950 € brutto, 19 %, fällig 15.11.2026 →  46 Tage, netto 5.000 €
//	C  20.000 € innergemeinschaftlich, fällig 31.01.2027 → nicht fällig, netto 20.000 €
//	D   2.380 € brutto, 19 %, fällig 31.03.2028 → Restlaufzeit über ein Jahr, netto 2.000 €
//	F   1.070 € brutto, 7 %, fällig 10.10.2026 →  82 Tage, netto 1.000 €
//	V   eine Verbindlichkeit, die nicht berichtigt wird
func allowanceItems() []domain.OpenItem {
	customer := func(id uint, open domain.Cents, due string, rate domain.TaxRate, treatment domain.TaxTreatment) domain.OpenItem {
		return domain.OpenItem{
			Source: domain.OpenItemSourceJournal, EntryID: id, EntryNumber: string(rune('A' + id - 1)),
			ContactType: domain.ContactTypeCustomer, ContactName: "Kunde", DueDate: due,
			GrossAmount: open, OpenAmount: open, TaxRate: rate, TaxTreatment: treatment,
		}
	}
	return []domain.OpenItem{
		customer(1, 1_190_000, "2026-08-15", domain.TaxRateStandard, domain.TaxTreatmentDomestic),
		customer(2, 595_000, "2026-11-15", domain.TaxRateStandard, domain.TaxTreatmentDomestic),
		customer(3, 2_000_000, "2027-01-31", 0, domain.TaxTreatmentReverseChargeSupply),
		customer(4, 238_000, "2028-03-31", domain.TaxRateStandard, domain.TaxTreatmentDomestic),
		customer(6, 107_000, "2026-10-10", domain.TaxRateReduced, domain.TaxTreatmentDomestic),
		{EntryID: 9, ContactType: domain.ContactTypeVendor, OpenAmount: 500_000, DueDate: "2026-01-01"},
	}
}

// Einzelwertberichtigung 50 % auf A (5.000 €) und 20 % für das Band 61 bis 90
// Tage auf F (200 €); Pauschalwertberichtigung 2 % auf B und C (25.000 € →
// 500 €) und auf D mit Restlaufzeit über einem Jahr (2.000 € → 40 €).
func TestAllowanceBooksNetAmountsByMaturity(t *testing.T) {
	result, err := CalculateAllowance(AllowanceInput{
		Cutoff: "2026-12-31", Items: allowanceItems(),
		Specific:        []SpecificAllowance{{EntryID: 1, Percent: 50, Reason: "Mahnbescheid ohne Zahlung"}},
		BandPercents:    map[domain.AgingBucketKey]int64{domain.AgingUpTo90: 20},
		GeneralPermille: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 5 {
		t.Fatalf("%d Forderungen bewertet, erwartet 5", len(result.Items))
	}
	if result.SpecificShort != 520_000 || result.SpecificLong != 0 {
		t.Errorf("Einzelwertberichtigung %s € und %s €", result.SpecificShort, result.SpecificLong)
	}
	if result.GeneralBase != 2_700_000 || result.GeneralShort != 50_000 || result.GeneralLong != 4_000 {
		t.Errorf("Pauschalwertberichtigung auf %s €: %s € und %s €",
			result.GeneralBase, result.GeneralShort, result.GeneralLong)
	}
	first := result.Items[0]
	if first.EntryID != 1 || first.OverdueDays != 138 || first.Band != domain.AgingOver90 ||
		!first.Candidate || first.NetAmount != 1_000_000 || first.Amount != 500_000 {
		t.Errorf("älteste Forderung: %+v", first)
	}
	targets := result.Targets()
	if targets[AccountEWBShortTerm] != 520_000 || targets[AccountPWBLongTerm] != 4_000 {
		t.Errorf("Sollbestände: %v", targets)
	}
}

// Eine Einzelwertberichtigung gibt es nur für einen offenen Posten, und der
// Ausfall liegt zwischen 0 und 100 %.
func TestAllowanceRefusesUnknownItemsAndPercentages(t *testing.T) {
	if _, err := CalculateAllowance(AllowanceInput{
		Cutoff: "2026-12-31", Items: allowanceItems(),
		Specific: []SpecificAllowance{{EntryID: 42, Percent: 10}},
	}); err == nil {
		t.Error("eine Buchung ohne offenen Posten wurde berichtigt")
	}
	if _, err := CalculateAllowance(AllowanceInput{
		Cutoff: "2026-12-31", Items: allowanceItems(),
		Specific: []SpecificAllowance{{EntryID: 1, Percent: 120}},
	}); err == nil {
		t.Error("ein Ausfall über 100 % wurde angenommen")
	}
	if got := LossRatePermille(30_000, 10_000_000); got != 3 {
		t.Errorf("Ausfallquote %d ‰ — erwartet 3 ‰", got)
	}
	if got := LossRatePermille(30_000, 0); got != 0 {
		t.Errorf("ohne Umsatz gibt es keine Quote, nicht %d ‰", got)
	}
}

func TestAllowanceAccountsExist(t *testing.T) {
	chart := NewChart(DefaultSKR04Accounts())
	accounts := append(AllowanceAccounts(), AccountEWBExpense, AccountPWBExpense, AccountEWBIncome, AccountPWBIncome)
	for _, account := range append(accounts, ReceivableLossAccounts()...) {
		if _, ok := chart.Lookup(account); !ok {
			t.Errorf("Konto %s steht nicht im SKR04-Katalog", account)
		}
	}
}
