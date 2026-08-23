package einvoice

import "testing"

// Zwei ZUGFeRD-Profile sind keine E-Rechnung im Sinne des Gesetzes. Wer aus
// ihnen bucht, zieht Vorsteuer aus einem Dokument, das rechtlich keine Rechnung
// ist.
func TestUnusableProfilesAreRefused(t *testing.T) {
	for _, profile := range []string{ProfileMinimum, ProfileBasicWL} {
		inv := &Invoice{SpecificationID: profile}
		if err := inv.EnsureUsableProfile(); err == nil {
			t.Errorf("%s hätte abgewiesen werden müssen", profile)
		}
	}
	// Dieselben zwei Profile werden auch unter der ZUGFeRD-Kennung ausgestellt.
	// Ein Wächter, der nur eine Schreibweise kennt, lässt genau die Dokumente
	// durch, die er aufhalten soll.
	for _, profile := range []string{
		"urn:zugferd.de:2p0:minimum", "urn:zugferd.de:2p0:basicwl",
		"urn:zugferd.de:2p1:minimum", "urn:zugferd.de:2p1:basicwl",
		"URN:FACTUR-X.EU:1P0:MINIMUM",
	} {
		inv := &Invoice{SpecificationID: profile}
		if err := inv.EnsureUsableProfile(); err == nil {
			t.Errorf("%s hätte abgewiesen werden müssen", profile)
		}
	}
	for _, profile := range []string{ProfileEN16931, ProfileBasic, ProfileExtended, ProfileXRechnung} {
		inv := &Invoice{SpecificationID: profile}
		if err := inv.EnsureUsableProfile(); err != nil {
			t.Errorf("%s ist nutzbar, wurde aber abgewiesen: %v", profile, err)
		}
	}
}

// Der Rechnungstyp entscheidet über das Vorzeichen der Buchung. Eine Gutschrift
// trägt positive Beträge und sagt nur hier, was sie ist.
func TestDocumentKindFollowsTheTypeCode(t *testing.T) {
	cases := map[string]DocumentKind{
		"380": KindInvoice,
		"381": KindCreditNote,
		"384": KindCorrection,
		"386": KindPrepayment,
		"389": KindSelfBilled,
		"877": KindPartialFinal,
		"325": KindOther,   // Proforma — in UNTDID 1001, aber kein Fall für uns
		"999": KindUnknown, // nicht in der Codeliste
		"":    KindUnknown,
	}
	for code, want := range cases {
		got := (&Invoice{TypeCode: code}).Kind()
		if got != want {
			t.Errorf("Typ %q = %q, erwartet %q", code, got, want)
		}
		if got.Label() == "" {
			t.Errorf("Typ %q hat keine Bezeichnung", code)
		}
	}
}

func TestXRechnungIsRecognised(t *testing.T) {
	if !(&Invoice{SpecificationID: ProfileXRechnung}).IsXRechnung() {
		t.Error("das XRechnung-Profil wird nicht erkannt")
	}
	if (&Invoice{SpecificationID: ProfileEN16931}).IsXRechnung() {
		t.Error("reines EN 16931 ist keine XRechnung")
	}
}
