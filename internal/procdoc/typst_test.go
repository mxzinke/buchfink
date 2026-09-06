package procdoc

import (
	"strings"
	"testing"
	"time"
)

// Der Satz übersetzt das erzeugte Markdown. Geprüft wird deshalb an dem
// Markdown, das die Vorlage tatsächlich erzeugt — nicht an einem Beispiel, das
// nur so ähnlich aussieht.
func TestTypstTranslatesTheGeneratedDocument(t *testing.T) {
	markdown, err := Render(Input{
		CompanyName: "Pfennig Ventures GmbH", LegalForm: "GmbH", FiscalYear: 2026,
		AppVersion: "1.0.0", RuleVersion: "2026.1", Version: "2026-09-05-1",
		CreatedAt: time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC), Actor: "anna@rechner",
		TaxCases:      []string{"Inlandsumsatz zum Regelsatz"},
		ExcludedCases: []string{"Reverse-Charge im Drittland"},
		DataDir:       "/home/anna/.buchfink/data",
		NumberRanges:  []NumberRange{{Name: "Ausgangsrechnungen", Format: "RE-{JJJJ}-{NNNN}", Next: "RE-2026-0001", Scope: "je Jahr"}},
		RetentionRows: []RetentionRow{{Category: "Buchungsbelege", Class: "vouchers", Years: 8, LegalBasis: "§ 147 Abs. 3 AO"}},
		CheckRules:    []CheckRule{{Key: "beleg_ohne_buchung", Severity: "blocker", Purpose: "Abgelegte, aber nicht gebuchte Belege"}},
	})
	if err != nil {
		t.Fatalf("die Verfahrensdokumentation ließ sich nicht erzeugen: %v", err)
	}

	template := Typst(markdown, "Verfahrensdokumentation 2026-09-05-1", time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC))

	for _, want := range []string{
		`#set page(paper: "a4"`,
		`#set document(title: "Verfahrensdokumentation 2026-09-05-1"`,
		"datetime(year: 2026, month: 9, day: 5)",
		"= Verfahrensdokumentation",
		"== 1. Allgemeine Beschreibung",
		"== 5. Internes Kontrollsystem",
		"#table(",
		"Pfennig Ventures GmbH",
	} {
		if !strings.Contains(template, want) {
			t.Errorf("die Vorlage enthält %q nicht", want)
		}
	}

	// Keine Markdown-Reste: eine Tabellenzeile mit Strichen und eine Zeile mit
	// Rautenüberschrift wären im Satz sichtbarer Unsinn.
	for _, line := range strings.Split(template, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "| ") || strings.HasPrefix(trimmed, "#### ") ||
			strings.HasPrefix(trimmed, "## ") {
			t.Errorf("unübersetzte Markdown-Zeile im Satz: %q", trimmed)
		}
	}
}

func TestTypstEscapesWhatWouldBeMarkup(t *testing.T) {
	got := Typst("Ein Satz mit #Raute, *Stern* und $Dollar$.", "Titel", time.Time{})
	if !strings.Contains(got, `\#Raute`) || !strings.Contains(got, `\*Stern\*`) ||
		!strings.Contains(got, `\$Dollar\$`) {
		t.Errorf("die Sonderzeichen sind nicht geschützt: %q", got)
	}
	if !strings.Contains(got, "#set document(title: \"Titel\", date: auto)") {
		t.Errorf("ohne Datum bleibt es bei auto: %q", got)
	}
}

func TestTypstKeepsBoldAndCode(t *testing.T) {
	got := Typst("**Eingang.** Der Ordner `dokumente/` bleibt.", "Titel", time.Time{})
	if !strings.Contains(got, "*Eingang.*") {
		t.Errorf("fetter Text wird zur Typst-Auszeichnung: %q", got)
	}
	if !strings.Contains(got, `#raw("dokumente/")`) {
		t.Errorf("Text in fester Breite bleibt erhalten: %q", got)
	}
}

func TestPDFFileNameFollowsTheMarkdownName(t *testing.T) {
	md := FileName("Pfennig Ventures GmbH", "2026-09-05-1")
	pdf := PDFFileName("Pfennig Ventures GmbH", "2026-09-05-1")
	if strings.TrimSuffix(md, ".md") != strings.TrimSuffix(pdf, ".pdf") {
		t.Errorf("die beiden Fassungen heißen verschieden: %q / %q", md, pdf)
	}
}
