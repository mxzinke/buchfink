package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

// Die Steuerfälle einer Richtung gehen als Auswahl an die Oberfläche. Eine
// Richtung, die der Katalog nicht kennt, liefert keinen Fall — aber eine leere
// Liste: `null.map` nähme im Render den ganzen Baum mit.
func TestTaxTreatmentsUnknownDirectionIsEmptyList(t *testing.T) {
	for _, dir := range []Direction{"", Direction("keine")} {
		raw, err := json.Marshal(TaxTreatments(dir))
		if err != nil {
			t.Fatalf("Steuerfälle als JSON: %v", err)
		}
		if string(raw) == "null" {
			t.Errorf("die Steuerfälle der Richtung %q kommen als `null` an", dir)
		}
	}
	// Gegenprobe: die bekannten Richtungen liefern weiterhin Fälle.
	if len(TaxTreatments(DirectionIncoming)) == 0 {
		t.Error("für Eingangsbelege fehlen die Steuerfälle")
	}
}

// Die Konfiguration vor der Einrichtung: kein Mandant angelegt. Sie ist der
// erste Bildschirm, den ein neuer Anwender sieht.
func TestAppConfigEnsureListsReplacesNilWithEmpty(t *testing.T) {
	var cfg AppConfig
	cfg.EnsureLists()
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Konfiguration als JSON: %v", err)
	}
	if strings.Contains(string(raw), `"tenants":null`) {
		t.Errorf("die Mandantenliste kommt als `null` an: %s", raw)
	}

	// Gegenprobe: ohne die Zusage steht dort `null` — sonst prüfte der Test
	// nichts mehr.
	bare, err := json.Marshal(AppConfig{})
	if err != nil {
		t.Fatalf("Gegenprobe: %v", err)
	}
	if !strings.Contains(string(bare), `"tenants":null`) {
		t.Errorf("die Gegenprobe trägt nicht mehr: %s", bare)
	}

	// Eine belegte Liste bleibt, wie sie ist.
	filled := AppConfig{Tenants: []TenantConfig{{ID: "a", Name: "Hauptmandant"}}}
	filled.EnsureLists()
	if len(filled.Tenants) != 1 {
		t.Errorf("die vorhandenen Mandanten wurden ersetzt: %+v", filled.Tenants)
	}
}

// Die Listen, die an den Kindsätzen einer Verknüpfung hängen. GORM belegt sie
// nicht, wenn es keine Kindsätze gibt — der Beleg ohne Datei, die Abgrenzung
// ohne Auflösungsplan, die frisch gebildete Rückstellung, die Gründung ohne
// erfassten Gesellschafter. Genau diese Randfälle sieht die Oberfläche zuerst.
func TestEnsureListsReplacesNilAssociations(t *testing.T) {
	assertNoNullField := func(label string, v any, fields ...string) {
		t.Helper()
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("%s als JSON: %v", label, err)
		}
		for _, field := range fields {
			if strings.Contains(string(raw), `"`+field+`":null`) {
				t.Errorf("%s: das Feld %q kommt als `null` in der Oberfläche an", label, field)
			}
		}
	}

	receipt := &Receipt{}
	receipt.EnsureLists()
	assertNoNullField("Beleg", receipt, "files")

	accrual := &Accrual{}
	accrual.EnsureLists()
	assertNoNullField("Abgrenzung", accrual, "releases")

	provision := &Provision{}
	provision.EnsureLists()
	assertNoNullField("Rückstellung", provision, "movements")

	foundation := &Foundation{}
	foundation.EnsureLists()
	assertNoNullField("Gründung", foundation, "shareholders")

	group := &InvoiceGroup{}
	group.EnsureLists()
	assertNoNullField("Rechnungsverbund", group, "advances")

	statement := &FinancialStatement{}
	statement.EnsureLists()
	assertNoNullField("Jahresabschluss", statement, "deadlines")

	summary := &VatSummary{}
	summary.EnsureLists()
	assertNoNullField("Umsatzsteuerübersicht", summary, "taxableRevenue")

	correction := &InputTaxCorrection{}
	correction.EnsureLists()
	assertNoNullField("Eintrag nach § 15a UStG", correction, "usages")

	// Ein Aufruf auf dem Nullzeiger darf nicht in Panik enden: die Zusicherung
	// steht auf Lesewegen, die auch „nichts gefunden" liefern können.
	var missing *Receipt
	missing.EnsureLists()
}
