package domain

import "testing"

// Die Umkehr der Perspektive ist der Schritt, der am leichtesten falsch geht
// und die größte Folge hat. Dass er hier geprüft werden kann, ohne dass
// irgendwo eine Datei, ein Format oder ein Parser vorkommt, ist der Grund, warum
// er hier steht und nicht im Leser.
func TestCategoryCodeIsTurnedAroundForTheRecipient(t *testing.T) {
	cases := map[string]TaxTreatment{
		"S":  TaxTreatmentDomestic,
		"AE": TaxTreatmentReverseCharge,
		"K":  TaxTreatmentIntraCommunityAcquisition,
		"Z":  TaxTreatmentZeroRated,
		"E":  TaxTreatmentExempt,
		"O":  TaxTreatmentNotTaxable,
		// Schreibweise und Leerraum kommen aus fremden Systemen.
		" s ": TaxTreatmentDomestic,
		"ae":  TaxTreatmentReverseCharge,
	}
	for code, want := range cases {
		got, err := TaxTreatmentForIncomingCategory(code)
		if err != nil {
			t.Errorf("%q: %v", code, err)
			continue
		}
		if got != want {
			t.Errorf("%q ergibt %q, erwartet %q", code, got, want)
		}
	}
}

// "K" ist beim Lieferanten eine steuerfreie innergemeinschaftliche Lieferung.
// Wer den Code eins zu eins übernimmt, bucht den halben Vorgang: der Erwerb
// braucht Erwerbsteuer und die gegenläufige Vorsteuer.
func TestIntraCommunitySupplyBecomesAnAcquisition(t *testing.T) {
	got, err := TaxTreatmentForIncomingCategory("K")
	if err != nil {
		t.Fatal(err)
	}
	if got == TaxTreatmentIntraCommunitySupply {
		t.Fatal("der Code wurde ungedreht übernommen — das bucht den halben Vorgang")
	}
	if got != TaxTreatmentIntraCommunityAcquisition {
		t.Errorf("= %q", got)
	}
}

// Wo es keine ehrliche Zuordnung gibt, wird das gesagt statt etwas Plausibles
// gewählt. Eine Ausfuhr des Lieferanten ist bei uns eine Einfuhr, und die
// Einfuhrumsatzsteuer steht im Zollbescheid — nicht in dieser Rechnung.
func TestUnmappableCategoriesAreNamed(t *testing.T) {
	for _, code := range []string{"G", "L", "M", "B", "", "XX"} {
		got, err := TaxTreatmentForIncomingCategory(code)
		if err == nil {
			t.Errorf("%q hätte keine Zuordnung ergeben dürfen, ergab aber %q", code, got)
		}
		if err != nil && len(err.Error()) < 30 {
			t.Errorf("%q: die Meldung erklärt den Fall nicht: %v", code, err)
		}
	}
}

// Was keine gewöhnliche Rechnung ist, wird nicht als eine gebucht.
func TestOnlyOrdinaryInvoicesAreBookable(t *testing.T) {
	if !EInvoiceKindInvoice.Bookable() {
		t.Error("eine Rechnung ist buchbar")
	}
	for _, kind := range []EInvoiceKind{
		EInvoiceKindCreditNote, EInvoiceKindCorrection, EInvoiceKindPrepayment,
		EInvoiceKindSelfBilled, EInvoiceKindPartialFinal, EInvoiceKindOther,
		EInvoiceKindUnknown,
	} {
		if kind.Bookable() {
			t.Errorf("%s ist noch nicht buchbar", kind.Label())
		}
		if kind.Label() == "" {
			t.Errorf("%s hat keine Bezeichnung", kind)
		}
	}
}
