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

	// Die Konten, die sich aus dem Anlagekonto ableiten: Sonderabschreibung,
	// Erhaltungsaufwand und laufender Ertrag. Jede dieser Ableitungen führt zu
	// genau einer Nummer, und jede muss bebuchbar sein.
	for _, entry := range AssetAccounts("") {
		if account, err := SpecialDepreciationAccount(entry.Class, entry.Number); err == nil {
			if err := chart.EnsurePostable(account); err != nil {
				t.Errorf("Konto der Sonderabschreibung %s zu %s: %v", account, entry.Number, err)
			}
		}
		if account, err := MaintenanceAccount(entry.Class, entry.Number); err == nil {
			if err := chart.EnsurePostable(account); err != nil {
				t.Errorf("Konto des Erhaltungsaufwands %s zu %s: %v", account, entry.Number, err)
			}
		}
		if account, err := AssetIncomeAccount(entry.Class, entry.Number); err == nil {
			if err := chart.EnsurePostable(account); err != nil {
				t.Errorf("Ertragskonto %s zu %s: %v", account, entry.Number, err)
			}
		}
	}
	for _, account := range []string{CurrencyLossAccount, CurrencyGainAccount} {
		if err := chart.EnsurePostable(account); err != nil {
			t.Errorf("Konto der Währungsumrechnung %s: %v", account, err)
		}
	}
}

// § 7g Abs. 5 EStG begünstigt nur bewegliche Wirtschaftsgüter. Ein Gebäude ist
// eine Sachanlage wie eine Maschine — die Sonderabschreibung bekommt es trotzdem
// nicht.
func TestSpecialDepreciationIsOnlyForMovableAssets(t *testing.T) {
	if _, err := SpecialDepreciationAccount(domain.AssetClassTangible, "0240"); err == nil {
		t.Error("für Geschäftsbauten darf es keine Sonderabschreibung nach § 7g Abs. 5 EStG geben")
	}
	if _, err := SpecialDepreciationAccount(domain.AssetClassFinancial, "0820"); err == nil {
		t.Error("für eine Beteiligung darf es keine Sonderabschreibung geben")
	}
	account, err := SpecialDepreciationAccount(domain.AssetClassTangible, "0520")
	if err != nil {
		t.Fatalf("Pkw: %v", err)
	}
	if account != "6242" {
		t.Errorf("Sonderabschreibung auf einen Pkw läuft auf %s — erwartet 6242 (Fahrzeuge)", account)
	}
	if account, err := SpecialDepreciationAccount(domain.AssetClassTangible, "0440"); err != nil || account != "6241" {
		t.Errorf("Sonderabschreibung auf eine Maschine: %s, %v — erwartet 6241", account, err)
	}
}

// Auf den Geschäfts- oder Firmenwert darf nicht zugeschrieben werden.
func TestGoodwillCannotBeWrittenUp(t *testing.T) {
	if _, err := WriteUpAccount(domain.AssetClassIntangible, GoodwillAccount, false); err == nil {
		t.Fatal("§ 253 Abs. 5 Satz 2 HGB verbietet die Zuschreibung auf den Geschäfts- oder Firmenwert")
	}
}

// Die Begründungspflicht wird abgeleitet und nicht gepflegt: genau die Konten
// mit dem Vorschlag aus dem BMF-Schreiben vom 22.02.2022 tragen sie.
//
// Der Test hält die Ableitung fest, weil an ihr zwei Seiten hängen: die Maske
// blendet das Begründungsfeld danach ein, und der Dienst verlangt es danach.
// Liefen sie auseinander, verlangte das Backend eine Angabe, für die es kein
// Feld gibt — genau die Sackgasse, die zu vermeiden war.
func TestUsefulLifeReasonFlagFollowsTheDigitalProposal(t *testing.T) {
	var flagged, digital []string
	for _, a := range AssetAccounts("") {
		if a.UsefulLifeReasonRequired {
			flagged = append(flagged, a.Number)
		}
		if a.UsefulLifeSource == UsefulLifeSourceDigital {
			digital = append(digital, a.Number)
		}
	}
	if len(digital) == 0 {
		t.Fatal("kein Konto trägt den Vorschlag des BMF-Schreibens — der Test prüfte nichts")
	}
	if len(flagged) != len(digital) {
		t.Fatalf("gekennzeichnet %v, Vorschlag an %v — die Ableitung stimmt nicht", flagged, digital)
	}
	for i := range digital {
		if flagged[i] != digital[i] {
			t.Errorf("Konto %s ist nicht gekennzeichnet", digital[i])
		}
	}

	// Ein Konto mit einem Erfahrungswert aus der AfA-Tabelle bleibt frei: die
	// Tabelle ist keine Wahlrechtsausübung.
	car, ok := LookupAssetAccount("0520")
	if !ok {
		t.Fatal("das Fahrzeugkonto 0520 fehlt im Katalog")
	}
	if car.DefaultUsefulLifeMonths == 0 {
		t.Fatal("0520 trägt keinen Vorschlag — der Test prüfte nichts")
	}
	if car.UsefulLifeReasonRequired {
		t.Error("für einen Erfahrungswert der AfA-Tabelle wird keine Begründung verlangt")
	}
}
