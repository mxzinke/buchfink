package invoice

import (
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// IncomingTaxTreatment maps the UNTDID 5305 category code of a received invoice
// to the Steuerfall it produces *for the recipient*.
//
// This is the inversion of vatCategoryCode, and it is not symmetric. The code in
// the document is written from the issuer's perspective: "K" says the supplier
// made a tax-exempt intra-community supply, which on this side is an
// innergemeinschaftlicher Erwerb with acquisition tax and matching input tax.
// Taking the code at face value books half the transaction and leaves the VAT
// return short.
//
// The result is a proposal the user confirms. Where no honest mapping exists the
// function says so rather than picking something plausible.
func IncomingTaxTreatment(categoryCode string) (domain.TaxTreatment, error) {
	switch strings.ToUpper(strings.TrimSpace(categoryCode)) {
	case "S":
		// Regelbesteuerung: der Lieferant hat Steuer ausgewiesen.
		return domain.TaxTreatmentDomestic, nil
	case "AE":
		// Steuerschuldnerschaft des Leistungsempfängers — das sind wir.
		return domain.TaxTreatmentReverseCharge, nil
	case "K":
		// Beim Lieferanten eine steuerfreie innergemeinschaftliche Lieferung,
		// bei uns ein innergemeinschaftlicher Erwerb.
		return domain.TaxTreatmentIntraCommunityAcquisition, nil
	case "Z":
		// Nullsteuersatz — steuerpflichtig zum Satz null, nicht steuerfrei.
		return domain.TaxTreatmentZeroRated, nil
	case "E":
		return domain.TaxTreatmentExempt, nil
	case "O":
		return domain.TaxTreatmentNotTaxable, nil
	case "G":
		// Ausfuhr beim Lieferanten heißt Einfuhr bei uns, mit
		// Einfuhrumsatzsteuer aus dem Zollbescheid statt aus dieser Rechnung.
		// Buchfink bildet den Fall nicht ab, und ihn als "steuerfrei" zu buchen
		// wäre falsch.
		return "", fmt.Errorf("der Kategoriecode G steht für eine Ausfuhr des Lieferanten. Für den Empfänger ist das eine Einfuhr, die Buchfink noch nicht abbildet — die Einfuhrumsatzsteuer steht im Zollbescheid, nicht in dieser Rechnung")
	case "":
		return "", fmt.Errorf("der Rechnungsdatensatz nennt keinen Steuerkategoriecode")
	default:
		return "", fmt.Errorf("der Steuerkategoriecode %q ist Buchfink nicht bekannt", categoryCode)
	}
}

// TaxRateFromPercent turns a CII rate ("19.00") into Buchfink's basis points.
func TaxRateFromPercent(percent string) (domain.TaxRate, error) {
	percent = strings.TrimSpace(percent)
	if percent == "" {
		return domain.TaxRateNone, nil
	}
	cents, err := domain.ParseCents(percent)
	if err != nil {
		return 0, fmt.Errorf("der Steuersatz %q ist unlesbar", percent)
	}
	// Cents und Basispunkte sind beide Hundertstel — "19.00" wird zu 1900.
	rate := domain.TaxRate(cents)
	for _, valid := range domain.ValidTaxRates() {
		if rate == valid {
			return rate, nil
		}
	}
	return 0, fmt.Errorf("der Steuersatz %s %% ist in Deutschland nicht vorgesehen", percent)
}
