package accounting

import (
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Paarprüfungen des Kennziffernblatts.
//
// Sie halten fest, was ELSTER bei der Übermittlung beanstanden würde oder was
// auf eine Fehlbuchung hinweist: eine Bemessungsgrundlage ohne Steuer (oder
// umgekehrt), eine Steuer, die mehr als 1 € neben Grundlage mal Satz liegt, und
// eine Vorsteuer aus § 13b oder Erwerb, die die geschuldete Steuer übersteigt.
// Sie sperren nichts; die Bestätigung liegt beim Menschen.

// vatDeviationTolerance ist die Abweichung je Zeile, ab der die gebuchte
// Steuer als auffällig gilt.
const vatDeviationTolerance domain.Cents = 100

// vatRangeCodes sind die Zeilen ohne festen Satz, deren Steuer zwischen 7 %
// und 19 % der Grundlage liegen muss.
var vatRangeCodes = map[string]bool{VatCodeReverseChargeEU: true, "73": true, VatCodeReverseChargeOther: true}

// vatReturnFindings prüft ein fertiges Blatt. unrounded sind die
// Bemessungsgrundlagen vor der Abrundung auf volle Euro.
func vatReturnFindings(ret *domain.VatReturn, unrounded map[string]domain.Cents) []string {
	out := make([]string, 0)
	for _, spec := range vatCodes {
		if !spec.hasBase || !spec.hasTax {
			continue
		}
		line, ok := ret.Line(spec.code)
		if !ok {
			continue
		}
		taxCode := spec.code
		if spec.taxCode != "" {
			taxCode = spec.taxCode
		}
		base := unrounded[spec.code]
		switch {
		case base != 0 && line.Tax == 0:
			out = append(out, fmt.Sprintf(
				"Kz %s hat eine Bemessungsgrundlage von %s €, aber keinen Steuerbetrag (Kz %s). Der Vordruck verlangt beides.",
				spec.code, base, taxCode))
			continue
		case base == 0 && line.Tax != 0:
			out = append(out, fmt.Sprintf(
				"Kz %s hat einen Steuerbetrag von %s €, aber keine Bemessungsgrundlage (Kz %s). Der Vordruck verlangt beides.",
				taxCode, line.Tax, spec.code))
			continue
		case base == 0:
			continue
		}
		if spec.rate != domain.TaxRateNone {
			if diff := (line.Tax - line.ExpectedTax).Abs(); diff > vatDeviationTolerance {
				out = append(out, fmt.Sprintf(
					"Kz %s: die gebuchte Steuer von %s € weicht um %s € von %s der Bemessungsgrundlage ab (%s €).",
					spec.code, line.Tax, diff, spec.rate.Label(), line.ExpectedTax))
			}
			continue
		}
		if !vatRangeCodes[spec.code] {
			continue
		}
		low := domain.TaxRateReduced.Tax(base.Abs()) - vatDeviationTolerance
		high := domain.TaxRateStandard.Tax(base.Abs()) + vatDeviationTolerance
		if tax := line.Tax.Abs(); tax < low || tax > high {
			out = append(out, fmt.Sprintf(
				"Kz %s: der Steuerbetrag von %s € passt zu keinem Steuersatz zwischen 7 %% und 19 %% auf %s € (Kz %s).",
				taxCode, line.Tax, base, spec.code))
		}
	}

	// Die Vorsteuer aus § 13b und aus dem Erwerb entsteht aus derselben
	// Bemessungsgrundlage wie die geschuldete Steuer und kann sie nicht
	// übersteigen (§ 15 Abs. 1 Satz 1 Nr. 3 und 4 UStG).
	pairs := []struct {
		input string
		owed  []string
		label string
	}{
		{VatCodeInputTaxReverse, []string{VatCodeReverseChargeEU, "73", VatCodeReverseChargeOther},
			"die geschuldete Steuer nach § 13b UStG (Kz 47, 74, 85)"},
		{VatCodeInputTaxAcquisition, []string{VatCodeAcquisition19, VatCodeAcquisition7, "95", "94"},
			"die Erwerbsteuer (Kz 89, 93, 98, 96)"},
	}
	for _, p := range pairs {
		var owed domain.Cents
		for _, code := range p.owed {
			owed += ret.Tax(code)
		}
		if input := ret.Tax(p.input); input > owed+vatDeviationTolerance {
			out = append(out, fmt.Sprintf(
				"Kz %s: die Vorsteuer von %s € übersteigt %s von %s €.", p.input, input, p.label, owed))
		}
	}
	return out
}
