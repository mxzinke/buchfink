package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Jede Kontonummer des Anlagenkatalogs muss im ausgelieferten DATEV SKR04
// vorhanden und bebuchbar sein. Eine erfundene oder aus dem SKR03 übernommene
// Nummer fiele sonst erst beim Buchen auf — im Zweifel Monate später.
func TestAssetAccountsExistInSKR04(t *testing.T) {
	chart := chartForTest(t)

	for _, entry := range AssetAccounts("") {
		if err := chart.EnsurePostable(entry.Number); err != nil {
			t.Errorf("Anlagekonto %s (%s): %v", entry.Number, entry.Name, err)
			continue
		}
		resolved, _ := chart.Lookup(entry.Number)
		if resolved.Kontenklasse != 0 {
			t.Errorf("Anlagekonto %s liegt in Kontenklasse %d — das Anlagevermögen steht in Klasse 0",
				entry.Number, resolved.Kontenklasse)
		}
		if resolved.Type != domain.AccountTypeAsset {
			t.Errorf("Anlagekonto %s %q ist %s, erwartet ein Bestandskonto der Aktiva",
				entry.Number, resolved.Name, resolved.Type)
		}

		if entry.DepreciationAccount == "" {
			if entry.Depreciable {
				t.Errorf("Anlagekonto %s ist abnutzbar, hat aber kein Aufwandskonto", entry.Number)
			}
			continue
		}
		if err := chart.EnsurePostable(entry.DepreciationAccount); err != nil {
			t.Errorf("AfA-Konto %s zu %s: %v", entry.DepreciationAccount, entry.Number, err)
			continue
		}
		if afa, _ := chart.Lookup(entry.DepreciationAccount); afa.Type != domain.AccountTypeExpense {
			t.Errorf("AfA-Konto %s %q ist %s, erwartet ein Aufwandskonto",
				entry.DepreciationAccount, afa.Name, afa.Type)
		}
	}
}

// Die Konten für außerplanmäßige Abschreibung, Zuschreibung und Abgang werden
// nicht eingegeben, sondern aus Klasse und Ergebnis abgeleitet. Damit hängt
// alles an dieser Ableitung — sie wird deshalb über jede Kombination geprüft.
func TestDerivedAssetAccountsExistInSKR04(t *testing.T) {
	chart := chartForTest(t)
	classes := []domain.AssetClass{
		domain.AssetClassIntangible, domain.AssetClassTangible, domain.AssetClassFinancial,
	}

	for _, class := range classes {
		for _, permanent := range []bool{true, false} {
			for _, privileged := range []bool{true, false} {
				account, err := ImpairmentAccount(class, "0630", permanent, privileged)
				if err != nil {
					// Für Sach- und immaterielle Anlagen ist die nicht dauernde
					// Wertminderung kein zulässiger Grund — die Ablehnung ist das
					// erwartete Ergebnis, kein Fehler.
					if permanent {
						t.Errorf("außerplanmäßige Abschreibung %s (dauernd): %v", class, err)
					}
					continue
				}
				if err := chart.EnsurePostable(account); err != nil {
					t.Errorf("Konto der außerplanmäßigen Abschreibung %s (%s): %v", account, class, err)
				}
			}
		}

		for _, privileged := range []bool{true, false} {
			account, err := WriteUpAccount(class, "0630", privileged)
			if err != nil {
				t.Errorf("Zuschreibungskonto %s: %v", class, err)
				continue
			}
			if err := chart.EnsurePostable(account); err != nil {
				t.Errorf("Zuschreibungskonto %s (%s): %v", account, class, err)
			}
		}

		for _, treatment := range domain.AllTaxTreatments() {
			for _, gain := range []bool{true, false} {
				for _, privileged := range []bool{true, false} {
					accounts, err := DisposalAccountsFor(class, treatment, gain, privileged)
					if err != nil {
						t.Errorf("Abgangskonten %s: %v", class, err)
						continue
					}
					if err := chart.EnsurePostable(accounts.Revenue); err != nil {
						t.Errorf("Erlöskonto %s (%s, %s): %v", accounts.Revenue, class, treatment, err)
					}
					if err := chart.EnsurePostable(accounts.BookValue); err != nil {
						t.Errorf("Restbuchwertkonto %s (%s): %v", accounts.BookValue, class, err)
					}
				}
			}
		}
	}
}

// Auf den Geschäfts- oder Firmenwert darf nicht zugeschrieben werden.
func TestGoodwillCannotBeWrittenUp(t *testing.T) {
	if _, err := WriteUpAccount(domain.AssetClassIntangible, GoodwillAccount, false); err == nil {
		t.Fatal("§ 253 Abs. 5 Satz 2 HGB verbietet die Zuschreibung auf den Geschäfts- oder Firmenwert")
	}
}
