package zugferd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/einvoice"
)

func lineInvoice() *einvoice.Invoice {
	return &einvoice.Invoice{
		SpecificationID: einvoice.ProfileEN16931,
		Lines:           []einvoice.Line{{ID: "1", Item: einvoice.Item{Name: "Beratung"}}},
		VATBreakdown:    []einvoice.VATBreakdown{{TypeCode: "VAT", CategoryCode: "S"}},
	}
}

// Die Kennung wird in mehreren Schreibweisen ausgeliefert — über die Versionen
// hinweg und teils schlicht falsch. Alle müssen ankommen, sonst weist Buchfink
// eine gültige Rechnung wegen eines Satzzeichens ab.
func TestProfileIsRecognisedInEverySpelling(t *testing.T) {
	cases := map[string]Profile{
		"urn:factur-x.eu:1p0:minimum":                                           ProfileMinimum,
		"urn:zugferd.de:2p0:minimum":                                            ProfileMinimum,
		"urn:factur-x.eu:1p0:basicwl":                                           ProfileBasicWL,
		"urn:cen.eu:en16931:2017#compliant#urn:factur-x.eu:1p0:basic":           ProfileBasic,
		"urn:cen.eu:en16931:2017#compliant#urn:zugferd.de:2p0:basic":            ProfileBasic,
		"urn:cen.eu:en16931:2017:compliant:factur-x.eu:1p0:basic":               ProfileBasic,
		"urn:cen.eu:en16931:2017":                                               ProfileEN16931,
		"urn:cen.eu:en16931:2017#conformant#urn:factur-x.eu:1p0:extended":       ProfileExtended,
		"urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0": ProfileXRechnung,
		"urn:etwas:ganz:anderes":                                                ProfileUnknown,
	}
	for id, want := range cases {
		if got := ProfileOf(&einvoice.Invoice{SpecificationID: id}); got != want {
			t.Errorf("%s = %q, erwartet %q", id, got, want)
		}
	}
}

// Zwei Profile sind keine Rechnung im Sinne des Gesetzes. Wer daraus bucht,
// zieht Vorsteuer aus einem Begleitsatz.
func TestOnlyCompleteProfilesAreInvoices(t *testing.T) {
	for _, p := range []Profile{ProfileMinimum, ProfileBasicWL} {
		if p.IsInvoice() {
			t.Errorf("%s ist keine vollständige Rechnung", p.Label())
		}
	}
	for _, p := range []Profile{ProfileBasic, ProfileEN16931, ProfileExtended, ProfileXRechnung} {
		if !p.IsInvoice() {
			t.Errorf("%s ist eine vollständige Rechnung", p.Label())
		}
	}
	if !ProfileExtended.AtLeast(ProfileBasic) || ProfileBasic.AtLeast(ProfileExtended) {
		t.Error("die Rangfolge der Profile stimmt nicht")
	}
}

// Ein Dokument, das MINIMUM nennt und Positionen mitschickt, widerspricht sich
// selbst — und ein Empfänger, der dem Profil glaubt, übersieht die Hälfte.
func TestContentMustFitTheDeclaredProfile(t *testing.T) {
	inv := lineInvoice()
	inv.SpecificationID = ProfileMinimum.Identifier()

	result := einvoice.ValidateWith(inv, Ruleset())
	if len(result.ByRule(RuleProfileFits)) == 0 {
		t.Error("Positionen in einem MINIMUM-Beleg müssen auffallen")
	}
	if len(result.ByRule(RuleProfileIsInvoice)) == 0 {
		t.Error("MINIMUM muss als unvollständige Rechnung gemeldet werden")
	}

	// Anlagen kann erst EN 16931 tragen.
	basic := lineInvoice()
	basic.SpecificationID = ProfileBasic.Identifier()
	basic.SupportingDocs = []einvoice.SupportingDocument{{Reference: "A1"}}
	if len(einvoice.ValidateWith(basic, Ruleset()).ByRule(RuleProfileFits)) == 0 {
		t.Error("Anlagen in einem BASIC-Beleg müssen auffallen")
	}

	// Ein passendes Dokument bleibt still — jedenfalls was das Profil angeht.
	// Ob es sonst vollständig ist, sagt die Norm und nicht diese Schicht.
	ok := lineInvoice()
	for _, f := range einvoice.ValidateWith(ok, Ruleset()).Findings {
		if strings.HasPrefix(f.Rule, "ZF-") {
			t.Errorf("%s: %s", f.Rule, f.Message)
		}
	}
}

// Welches Profil der Inhalt verlangt, ist die Frage beim Erzeugen: ein zu
// kleines Profil scheitert beim Empfänger, ein zu großes schränkt ein, wer die
// Rechnung lesen kann.
func TestMinimumProfileFollowsTheContent(t *testing.T) {
	empty := &einvoice.Invoice{}
	if got := MinimumProfileFor(empty); got != ProfileMinimum {
		t.Errorf("ohne Inhalt = %q", got)
	}
	withTax := &einvoice.Invoice{VATBreakdown: []einvoice.VATBreakdown{{CategoryCode: "S"}}}
	if got := MinimumProfileFor(withTax); got != ProfileBasicWL {
		t.Errorf("mit Steueraufschlüsselung = %q", got)
	}
	if got := MinimumProfileFor(lineInvoice()); got != ProfileBasic {
		t.Errorf("mit Positionen = %q", got)
	}
	withDocs := lineInvoice()
	withDocs.SupportingDocs = []einvoice.SupportingDocument{{Reference: "A1"}}
	if got := MinimumProfileFor(withDocs); got != ProfileEN16931 {
		t.Errorf("mit Unterlagen = %q", got)
	}
}

// Der Rundlauf über die Kennung: was wir für ein Profil schreiben, muss auch
// wieder als dieses Profil gelesen werden.
func TestIdentifiersRoundTrip(t *testing.T) {
	for _, p := range []Profile{ProfileMinimum, ProfileBasicWL, ProfileBasic,
		ProfileEN16931, ProfileExtended, ProfileXRechnung} {
		// ZUGFeRD 1 fehlt hier mit Absicht: Buchfink schreibt es nicht.
		id := p.Identifier()
		if id == "" {
			t.Errorf("%s hat keine Kennung", p.Label())
			continue
		}
		if got := ProfileOf(&einvoice.Invoice{SpecificationID: id}); got != p {
			t.Errorf("%s → %q → %q", p.Label(), id, got)
		}
	}
}

// Eine hybride Rechnung darf neben dem Datensatz weitere Dateien tragen. Aus
// der falschen zu buchen hieße, aus einem Dokument zu buchen, das nicht die
// Rechnung ist.
func TestInvoiceAttachmentIsToldApartFromEnclosures(t *testing.T) {
	for _, name := range []string{"factur-x.xml", "FACTUR-X.XML", "zugferd-invoice.xml", "ZUGFeRD-invoice.xml"} {
		if !IsInvoiceAttachment(name) {
			t.Errorf("%q ist der Rechnungsdatensatz", name)
		}
	}
	for _, name := range []string{"stundenzettel.pdf", "anlage.xml", "order-x.xml", ""} {
		if IsInvoiceAttachment(name) {
			t.Errorf("%q ist nicht der Rechnungsdatensatz", name)
		}
	}
}

// Die offiziellen CII-Beispiele tragen echte Profilkennungen. Keines davon darf
// als unbekannt durchfallen.
func TestOfficialExamplesDeclareKnownProfiles(t *testing.T) {
	dir := strings.TrimSpace(os.Getenv("EN16931_CII_EXAMPLES"))
	if dir == "" {
		t.Skip("EN16931_CII_EXAMPLES ist nicht gesetzt")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.xml"))
	if len(files) == 0 {
		t.Skip("keine Beispiele gefunden")
	}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", filepath.Base(path), err)
		}
		inv, err := einvoice.ParseAny(data)
		if err != nil {
			t.Fatalf("%s: %v", filepath.Base(path), err)
		}
		if got := ProfileOf(inv); got == ProfileUnknown {
			t.Errorf("%s nennt das unbekannte Profil %q", filepath.Base(path), inv.Profile())
		}
	}
}

// ZUGFeRD 1 ist seit 2025 keine E-Rechnung mehr. Es trotzdem zu erkennen ist
// wichtiger, als es abzuweisen: solche Belege liegen in Archiven, werden
// geprüft, und der Empfänger muss erfahren, was angekommen ist.
func TestZUGFeRD1IsRecognisedAndFlagged(t *testing.T) {
	inv := lineInvoice()
	inv.SpecificationID = "urn:ferd:CrossIndustryDocument:invoice:1p0:comfort"

	profile := ProfileOf(inv)
	if profile != ProfileZUGFeRD1 {
		t.Fatalf("Profil = %q", profile)
	}
	if profile.FollowsEN16931() {
		t.Error("ZUGFeRD 1 folgt EN 16931 nicht")
	}
	result := einvoice.ValidateWith(inv, Ruleset())
	if len(result.ByRule(RuleProfileFollowsStandard)) == 0 {
		t.Error("ein Beleg nach ZUGFeRD 1 muss als solcher gemeldet werden")
	}
	if len(result.ByRule(RuleProfileKnown)) > 0 {
		t.Error("ZUGFeRD 1 ist bekannt, nur eben veraltet")
	}
}
