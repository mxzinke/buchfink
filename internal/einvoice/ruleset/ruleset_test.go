package ruleset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/einvoice"
	"github.com/buchfink/buchfink/internal/einvoice/xrechnung"
	"github.com/buchfink/buchfink/internal/einvoice/zugferd"
)

// Welche Schichten laufen, entscheidet das Dokument. Ein Absender, der
// XRechnung angibt, hat diese Regeln zugesagt — sie zu prüfen ist das, was dem
// Empfänger sagt, ob die Zusage trägt.
func TestLayersFollowWhatTheDocumentDeclares(t *testing.T) {
	plain := &einvoice.Invoice{SpecificationID: einvoice.ProfileEN16931}
	if got := len(For(plain)); got != 1 {
		t.Errorf("für reines EN 16931 laufen %d Schichten, erwartet 1 (nur das Profil)", got)
	}

	german := &einvoice.Invoice{SpecificationID: xrechnung.Current}
	ids := map[string]bool{}
	for _, layer := range For(german) {
		ids[layer.ID()] = true
	}
	if !ids[xrechnung.Version] {
		t.Error("eine XRechnung muss auch gegen die deutsche Ausprägung geprüft werden")
	}
	if !ids[zugferd.Version] {
		t.Error("das Profil wird immer geprüft")
	}
}

// Was ein Dokument über sich sagt, ist die erste Frage — vor jeder Beurteilung.
func TestDescribeSeparatesTheQuestions(t *testing.T) {
	minimum := &einvoice.Invoice{
		Syntax:          einvoice.SyntaxCII,
		SpecificationID: zugferd.ProfileMinimum.Identifier(),
		TypeCode:        "380",
	}
	d := Describe(minimum)
	if d.Profile != zugferd.ProfileMinimum {
		t.Errorf("Profil = %q", d.Profile)
	}
	if d.BookableAsInvoice {
		t.Error("MINIMUM trägt keine vollständige Rechnung")
	}
	if !d.FollowsEN16931 {
		t.Error("MINIMUM folgt der Norm, es ist nur unvollständig")
	}
	if d.Kind != einvoice.KindInvoice {
		t.Errorf("Art = %q", d.Kind)
	}

	credit := &einvoice.Invoice{
		Syntax:          einvoice.SyntaxUBL,
		SpecificationID: xrechnung.Current,
		TypeCode:        "381",
	}
	d = Describe(credit)
	if d.Kind != einvoice.KindCreditNote {
		t.Errorf("eine Gutschrift muss als solche erkannt werden, erkannt wurde %q", d.Kind)
	}
	if !d.IsXRechnung || d.UsesExtension {
		t.Errorf("XRechnung=%v, Extension=%v", d.IsXRechnung, d.UsesExtension)
	}
}

// Der Durchstich: eine echte XRechnung von KoSIT, gelesen und über alle
// zutreffenden Schichten geprüft, ohne dass der Aufrufer eine davon kennt.
func TestKositReferenceRunsThroughEveryLayer(t *testing.T) {
	dir := strings.TrimSpace(os.Getenv("XRECHNUNG_TEST_INSTANCES"))
	if dir == "" {
		t.Skip("XRECHNUNG_TEST_INSTANCES ist nicht gesetzt")
	}
	data, err := os.ReadFile(filepath.Join(dir, "ubl001.xml"))
	if err != nil {
		t.Skipf("ubl001.xml nicht vorhanden: %v", err)
	}
	inv, err := einvoice.ParseAny(data)
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}

	result := Validate(inv)
	if len(result.Rulesets) < 3 {
		t.Errorf("gelaufen sind %v, erwartet Norm, Profil und XRechnung", result.Rulesets)
	}
	for _, f := range result.Findings {
		if f.Severity == einvoice.SeverityFatal {
			t.Errorf("%s %s: %s", f.Rule, f.Where, f.Message)
		}
	}
	if d := Describe(inv); !d.IsXRechnung || !d.BookableAsInvoice {
		t.Errorf("Beschreibung: %+v", d)
	}
}
