package wailsbridge

import (
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die Verdrahtung eines Mandanten läuft durch und die Auswertungen der Welle 5c
// antworten auf einem leeren Bestand.
//
// Der Test prüft nichts Fachliches — das tun die Dienste. Er prüft, dass die
// Reihenfolge in initTenant stimmt: die Dienste hängen aneinander, und ein
// Dienst, der einen später erzeugten setzt, bekommt nil. Das fiele sonst erst
// beim ersten Aufruf im laufenden Programm auf, und dort als leerer Bildschirm
// ohne Meldung.
func TestTenantWiringAnswersOnAnEmptyLedger(t *testing.T) {
	dir := t.TempDir()
	b := &BuchfinkBridge{
		currentYear: 2026,
		appCfgRepo:  repository.NewAppConfigRepository(t.TempDir()),
	}
	tenant := &domain.TenantConfig{
		ID: "tenant_wiring", Name: "Pfennig Ventures GmbH", DataDir: dir,
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	if err := b.initTenant(tenant); err != nil {
		t.Fatalf("Mandant einrichten: %v", err)
	}

	if _, err := b.GetInputTaxCorrections(2026); err != nil {
		t.Errorf("Verzeichnis nach § 15a UStG: %v", err)
	}
	if _, err := b.GetNonDeductibleReport(2026); err != nil {
		t.Errorf("nicht abziehbare Betriebsausgaben: %v", err)
	}
	if _, err := b.GetWriteUpReport(2026); err != nil {
		t.Errorf("Wertaufholung: %v", err)
	}
	if _, err := b.GetPoolConsistencyReport(2026); err != nil {
		t.Errorf("Sammelposten-Einheitlichkeit: %v", err)
	}
	if _, err := b.GetSupplyEvidenceReport(2026); err != nil {
		t.Errorf("Belegnachweis: %v", err)
	}
	if _, err := b.GetVatExchangeRates("2026-01", "2026-12"); err != nil {
		t.Errorf("Umsatzsteuer-Umrechnungskurse: %v", err)
	}

	// Die drei neuen Abschlussbausteine stehen in der Schrittliste und fragen
	// ihre Karteien, ohne daran zu scheitern.
	steps, err := b.GetClosingSteps(2026)
	if err != nil {
		t.Fatalf("Schrittliste: %v", err)
	}
	want := map[domain.ClosingStepKey]bool{
		domain.ClosingStepWriteUp:            false,
		domain.ClosingStepCurrencyValuation:  false,
		domain.ClosingStepInputTaxCorrection: false,
	}
	for _, step := range steps.Steps {
		if _, ok := want[step.Key]; ok {
			want[step.Key] = true
		}
	}
	for key, found := range want {
		if !found {
			t.Errorf("der Baustein %q fehlt in der Schrittliste", key)
		}
	}

	if len(b.GetAfaRules().DegressiveWindows) == 0 {
		t.Error("die Abschreibungsregeln kommen leer aus der Ressource")
	}
	if len(b.GetSupplyEvidenceKinds()) == 0 {
		t.Error("die Belegarten des Nachweises kommen leer")
	}
	if len(b.GetNonDeductibleCategories()) == 0 {
		t.Error("die Kategorien des § 4 Abs. 5 EStG kommen leer")
	}
}
