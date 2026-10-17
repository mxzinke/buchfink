package accounting

import (
	"fmt"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestPartnershipAccountsAreNotPostableForCorporations(t *testing.T) {
	chart := NewChart(DefaultSKR04Accounts())
	for _, number := range []string{"2000", "2100", "2180", "2500", "1317", "3645", "0060"} {
		for _, form := range []string{"", domain.LegalFormGmbH, domain.LegalFormUG} {
			err := chart.EnsurePostableFor(number, form)
			if err == nil || !strings.Contains(err.Error(), "verdeckte Gewinnausschüttung") {
				t.Errorf("Konto %s bei %q: erwartet die Sperre mit Hinweis auf den richtigen Weg, Fehler: %v",
					number, form, err)
			}
		}
	}
	for _, number := range []string{"1307", "3640", "2900", "2920", "1800", "9000"} {
		if err := chart.EnsurePostableFor(number, domain.LegalFormGmbH); err != nil {
			t.Errorf("Konto %s gehört zur GmbH und muss bebuchbar bleiben: %v", number, err)
		}
	}
}

// Die Bereiche zeigen auf Konten, deren Bezeichnung sie als Konten der
// Einzelunternehmen und Personengesellschaften ausweist.
func TestPartnershipAccountRangesStartOnPartnershipAccounts(t *testing.T) {
	chart := NewChart(DefaultSKR04Accounts())
	markers := []string{"Vollhafter", "Teilhafter", "persönlich haftende", "Kommandit", "Festkapital",
		"Gesellschafter", "Privat", "Kapital", "Einzahlungsverpflichtung", "Rücklagen", "Entnahmen"}
	for _, r := range partnershipAccountRanges {
		number := fmt.Sprintf("%04d", r[0])
		account, ok := chart.Lookup(number)
		if !ok {
			t.Errorf("Bereichsbeginn %s fehlt im SKR04", number)
			continue
		}
		found := false
		for _, m := range markers {
			if strings.Contains(account.Name, m) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Konto %s (%s) sieht nicht nach einem Konto der Personengesellschaft aus", number, account.Name)
		}
	}
}
