package accounting

import (
	"fmt"
	"strings"
	"sync"

	"github.com/buchfink/buchfink/internal/domain"
)

// taxKeyRule ist, was ein Steuerschlüssel an seiner Zeile verlangt.
type taxKeyRule struct {
	accounts []string
	// rate ist der Satz, aus dem der Steuerbetrag folgt. Null bei Schlüsseln
	// ohne Bemessungsgrundlage (§ 14c, § 15a).
	rate domain.TaxRate
}

var (
	taxKeyRulesOnce sync.Once
	taxKeyRules     map[string]taxKeyRule
)

// rulesForTaxKeys baut die Regeln aus dem Katalog: jeder Schlüssel, den der
// Steuerrechner erzeugen kann, dazu die Skontokorrektur jedes Schlüssels mit
// Satz und die beiden Schlüssel ohne Steuerfall.
func rulesForTaxKeys() map[string]taxKeyRule {
	taxKeyRulesOnce.Do(func() {
		rules := map[string]taxKeyRule{}
		for _, info := range TaxKeyCatalog() {
			switch info.Key {
			case TaxKeyUnlawful:
				rules[info.Key] = taxKeyRule{accounts: []string{domain.AccountUmsatzsteuer14c}}
			case TaxKeyInputTaxCorrection:
				rules[info.Key] = taxKeyRule{accounts: []string{
					InputTaxCorrectionDeductibleMovable, InputTaxCorrectionRepayableMovable,
					InputTaxCorrectionDeductibleImmovable, InputTaxCorrectionRepayableImmovable,
				}}
			default:
				rule := taxKeyRule{accounts: []string{info.Account}, rate: info.Rate}
				rules[info.Key] = rule
				rules[TaxKeySkontoPrefix+info.Key] = rule
			}
		}
		taxKeyRules = rules
	})
	return taxKeyRules
}

// IsKnownTaxKey meldet, ob Buchfink den Steuerschlüssel kennt.
func IsKnownTaxKey(key string) bool {
	_, ok := rulesForTaxKeys()[key]
	return ok
}

// CheckTaxLine prüft eine Zeile mit Steuerschlüssel gegen den Katalog: der
// Schlüssel existiert, steht auf seinem Konto, und der Steuerbetrag passt zur
// Bemessungsgrundlage. Ohne diese Prüfung liefe ein unbekannter Schlüssel still
// an der Voranmeldung vorbei, und ein falscher Satz stünde dort unbemerkt.
func CheckTaxLine(l domain.JournalLine) error {
	if l.TaxKey == "" {
		return nil
	}
	rule, ok := rulesForTaxKeys()[l.TaxKey]
	if !ok {
		return fmt.Errorf("unbekannter Steuerschlüssel %q; er käme in keiner Kennziffer der Voranmeldung an", l.TaxKey)
	}
	if !containsAccount(rule.accounts, l.Account) {
		return fmt.Errorf("der Steuerschlüssel %s gehört auf Konto %s, nicht auf %s",
			l.TaxKey, strings.Join(rule.accounts, " oder "), l.Account)
	}
	if rule.rate == domain.TaxRateNone {
		return nil
	}
	if l.TaxBase == 0 {
		return fmt.Errorf("der Steuerschlüssel %s braucht die Bemessungsgrundlage; ohne sie fehlt der Umsatz in der Voranmeldung",
			l.TaxKey)
	}
	if (l.TaxBase < 0) != (l.Amount < 0) {
		return fmt.Errorf("Steuerbetrag %s € und Bemessungsgrundlage %s € haben verschiedene Vorzeichen",
			l.Amount, l.TaxBase)
	}
	expected := rule.rate.Tax(l.TaxBase.Abs())
	if diff := (l.Amount.Abs() - expected).Abs(); diff > taxTolerance(expected) {
		return fmt.Errorf(
			"%s € Steuer passen nicht zu %s € Bemessungsgrundlage beim Satz %s (erwartet etwa %s €)",
			l.Amount.Abs(), l.TaxBase.Abs(), rule.rate.Label(), expected)
	}
	return nil
}

// taxTolerance lässt Rundungen zu, wie sie aus Positionsrundung, Skonto oder
// fremden Rechnungen entstehen: ein Prozent der Steuer, mindestens fünf Cent.
// Ein falscher Satz oder der Bruttobetrag als Grundlage liegen weit darüber.
func taxTolerance(expected domain.Cents) domain.Cents {
	if t := expected / 100; t > 5 {
		return t
	}
	return 5
}

func containsAccount(accounts []string, account string) bool {
	for _, a := range accounts {
		if a == account {
			return true
		}
	}
	return false
}
