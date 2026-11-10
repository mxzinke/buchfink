package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// -------------------------------------------------------------------------
// Die Konten der § 4 Abs. 5 EStG-Kategorien
// -------------------------------------------------------------------------

// Der Bericht und der Gruppenkatalog müssen dieselben Konten nennen. Zwei
// Listen, die auseinanderlaufen, ergäben einen Bericht, in dem eine Buchung
// nicht vorkommt.
func TestNonDeductibleCategoriesMatchGroups(t *testing.T) {
	inCatalog := map[string]bool{}
	for _, g := range PostingGroups(domain.DirectionIncoming) {
		if g.Account != "" {
			inCatalog[g.Account] = true
		}
		if g.NonDeductibleAccount != "" {
			inCatalog[g.NonDeductibleAccount] = true
		}
	}
	for _, c := range NonDeductibleCategories() {
		for _, account := range []string{c.DeductibleAccount, c.NonDeductibleAccount} {
			if account == "" {
				continue
			}
			if !inCatalog[account] {
				t.Errorf("die Kategorie %q nennt das Konto %s, das keine Buchungsgruppe erreicht",
					c.Label, account)
			}
			if _, ok := NonDeductibleCategoryForAccount(account); !ok {
				t.Errorf("das Konto %s findet seine Kategorie nicht zurück", account)
			}
		}
	}
}

// Die Konten der beschränkt abziehbaren Betriebsausgaben sind nur über ihre
// Gruppe erreichbar — sonst ließe sich an der Aufzeichnungspflicht des
// § 4 Abs. 7 EStG vorbeibuchen.
func TestAccountsRequiringGroupCoversTheLimitedCategories(t *testing.T) {
	blocked := AccountsRequiringGroup()
	for _, account := range []string{"6610", "6620", "6625", "6640", "6644", "6645"} {
		if _, ok := blocked[account]; !ok {
			t.Errorf("das Konto %s muss über seine Gruppe erreichbar sein und nicht frei", account)
		}
	}
	// Ein gewöhnliches Aufwandskonto bleibt frei wählbar: der Notausgang für
	// Fälle, die der Katalog nicht kennt, muss offen bleiben.
	if _, ok := blocked["6815"]; ok {
		t.Error("das Konto 6815 (Bürobedarf) darf frei wählbar bleiben")
	}
}
