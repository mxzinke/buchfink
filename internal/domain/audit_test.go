package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

// --- Vorher/Nachher des Änderungsprotokolls -------------------------------

func TestChangedFieldsKeepsOnlyWhatChanged(t *testing.T) {
	before := &Contact{
		ID: 7, Type: ContactTypeCustomer, Name: "Meier GmbH",
		LedgerAccount: "10001", Street: "Hauptstraße 1", City: "München",
		PaymentTermsDays: 14,
	}
	after := *before
	after.Street = "Nebenstraße 5"
	after.PaymentTermsDays = 30

	beforeJSON, afterJSON := ChangedFields(before, &after)
	if beforeJSON == "" || afterJSON == "" {
		t.Fatal("eine Änderung muss Vorher und Nachher liefern")
	}

	var beforeMap, afterMap map[string]any
	if err := json.Unmarshal([]byte(beforeJSON), &beforeMap); err != nil {
		t.Fatalf("Vorher ist kein JSON-Objekt: %v", err)
	}
	if err := json.Unmarshal([]byte(afterJSON), &afterMap); err != nil {
		t.Fatalf("Nachher ist kein JSON-Objekt: %v", err)
	}

	// Genau die beiden geänderten Felder — nicht der ganze Datensatz. Sonst
	// verbärge sich die eine Änderung zwischen dreißig unveränderten Feldern.
	wantKeys := map[string]bool{"street": true, "paymentTermsDays": true}
	if len(afterMap) != len(wantKeys) {
		t.Errorf("Nachher hat %d Felder (%v), erwartet %d", len(afterMap), afterMap, len(wantKeys))
	}
	for key := range wantKeys {
		if _, ok := afterMap[key]; !ok {
			t.Errorf("das geänderte Feld %q fehlt im Nachher", key)
		}
	}
	if _, ok := afterMap["name"]; ok {
		t.Error("der unveränderte Name gehört nicht ins Protokoll")
	}
	if beforeMap["street"] != "Hauptstraße 1" {
		t.Errorf("Vorher.street = %v, erwartet die alte Anschrift", beforeMap["street"])
	}
	if afterMap["street"] != "Nebenstraße 5" {
		t.Errorf("Nachher.street = %v, erwartet die neue Anschrift", afterMap["street"])
	}
}

func TestChangedFieldsOnCreateHasNoBefore(t *testing.T) {
	after := &Contact{ID: 1, Type: ContactTypeVendor, Name: "Neu GmbH", LedgerAccount: "70001"}

	beforeJSON, afterJSON := ChangedFields(nil, after)
	if beforeJSON != "" {
		t.Errorf("ein neu angelegter Datensatz hat kein Vorher, bekommen %q", beforeJSON)
	}
	if !strings.Contains(afterJSON, "Neu GmbH") {
		t.Errorf("das Nachher muss den angelegten Datensatz enthalten, bekommen %q", afterJSON)
	}
}

func TestChangedFieldsIsEmptyWithoutChange(t *testing.T) {
	c := &Contact{ID: 3, Name: "Unverändert GmbH", LedgerAccount: "10002"}
	same := *c

	beforeJSON, afterJSON := ChangedFields(c, &same)
	if beforeJSON != "" || afterJSON != "" {
		t.Errorf("ohne Änderung darf nichts protokolliert werden, bekommen %q / %q", beforeJSON, afterJSON)
	}
}

func TestChangedFieldsIgnoresNamedFields(t *testing.T) {
	// Der Saldo des Personenkontos wird beim Lesen gerechnet und nicht
	// gespeichert. Stünde er im Protokoll, meldete jedes Speichern eine
	// Änderung, die niemand vorgenommen hat.
	before := &Contact{ID: 4, Name: "Meier GmbH", OpenAmount: 10000}
	after := *before
	after.OpenAmount = 25000

	beforeJSON, afterJSON := ChangedFields(before, &after, "openAmount")
	if beforeJSON != "" || afterJSON != "" {
		t.Errorf("ausgenommene Felder gehören nicht ins Protokoll, bekommen %q / %q", beforeJSON, afterJSON)
	}
}

func TestChangedFieldsWorksOnSettings(t *testing.T) {
	before := &CompanySettings{CompanyName: "Alt GmbH", TaxationType: "SOLL", VatPeriod: "quarter"}
	after := &CompanySettings{CompanyName: "Alt GmbH", TaxationType: "IST", VatPeriod: "quarter"}

	beforeJSON, afterJSON := ChangedFields(before, after)
	if !strings.Contains(beforeJSON, "SOLL") || !strings.Contains(afterJSON, "IST") {
		t.Errorf("die Umstellung der Besteuerungsart muss im Protokoll stehen: %q → %q", beforeJSON, afterJSON)
	}
	if strings.Contains(afterJSON, "companyName") {
		t.Error("der unveränderte Firmenname gehört nicht ins Protokoll")
	}
}
