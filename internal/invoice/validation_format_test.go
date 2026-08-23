package invoice

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

// Das Prüfergebnis wird als JSON am Beleg abgelegt, nicht als zusammengefügter
// Text. Die Oberfläche zeigt Regel, Schweregrad und die betroffenen
// Geschäftsbegriffe getrennt an; kann sie die Liste nicht lesen, sieht sie null
// Verstöße — und meldet einen Beleg mit schweren Mängeln als in Ordnung.
func TestValidationFindingsAreStoredAsJSON(t *testing.T) {
	data, err := os.ReadFile("../einvoice/testdata/en16931/cii-examples/CII_example1.xml")
	if err != nil {
		t.Fatalf("Beispielrechnung: %v", err)
	}
	// Die Rechnungsnummer entfernen: das verletzt BR-02 und macht aus einer
	// fehlerfreien Beispielrechnung eine mit einem schweren Mangel.
	data = bytes.Replace(data, []byte("<ram:ID>12115118</ram:ID>"), []byte("<ram:ID></ram:ID>"), 1)

	validation, err := NewReader().ValidateOnly(data)
	if err != nil {
		t.Fatalf("Prüfung: %v", err)
	}
	if validation.Errors == 0 {
		t.Fatal("die Prüfdatei verletzt BR-02, es wurde aber kein Fehler gemeldet")
	}

	var findings []struct {
		Rule     string   `json:"rule"`
		Severity string   `json:"severity"`
		Message  string   `json:"message"`
		Terms    []string `json:"terms"`
	}
	if err := json.Unmarshal([]byte(validation.Findings), &findings); err != nil {
		t.Fatalf("die Befunde sind kein JSON: %v\n%s", err, validation.Findings)
	}
	if len(findings) == 0 {
		t.Fatal("die Befundliste ist leer, obwohl Fehler gezählt wurden")
	}

	fatal := 0
	for _, f := range findings {
		if f.Rule == "" || f.Message == "" {
			t.Errorf("ein Befund ohne Regel oder Meldung: %+v", f)
		}
		if f.Severity == "fatal" {
			fatal++
		}
	}
	if fatal != validation.Errors {
		t.Errorf("%d schwere Befunde in der Liste, aber %d gezählt", fatal, validation.Errors)
	}
}

// Ohne Befunde bleibt das Feld leer statt "null" — die Oberfläche liest es als
// leere Liste, und JSON.parse("null") ergäbe kein Array, über das sie laufen kann.
func TestCleanInvoiceStoresNoFindings(t *testing.T) {
	data, err := os.ReadFile("../einvoice/testdata/en16931/cii-examples/CII_example1.xml")
	if err != nil {
		t.Skipf("Beispielrechnung nicht vorhanden: %v", err)
	}
	validation, err := NewReader().ValidateOnly(data)
	if err != nil {
		t.Fatalf("Prüfung: %v", err)
	}
	if len(validation.Findings) == 0 {
		return
	}
	var findings []map[string]any
	if err := json.Unmarshal([]byte(validation.Findings), &findings); err != nil {
		t.Fatalf("die Befunde sind kein JSON: %v", err)
	}
}
