package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Die Runde-4-Nachbesserungen: der ursprüngliche Verwendungsanteil von null.

// Ein ursprünglicher Anteil von 0 ‰ bleibt null.
//
// Er ist der klassische Fall der Berichtigung nach oben: angeschafft ohne
// Vorsteuerabzug, weil das Wirtschaftsgut für steuerfreie Umsätze verwendet
// wurde, später für abzugsberechtigende Umsätze eingesetzt (§ 15a Abs. 1 UStG).
// Solange die Angabe eine Zahl mit `omitempty` war, wurde daraus stillschweigend
// volle Verwendung — und damit aus einer Berichtigung zugunsten des Anwenders
// eine zu seinen Lasten.
func TestOriginalPermilleOfZeroSurvives(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.inputTax(t)

	none := 0
	correction, err := svc.Register(ctx, RegisterInputTaxRequest{
		Label: "Halle AN-2026-0002", Account: "0240",
		AcquisitionDate:  "2026-01-15",
		NetAmount:        20_000_000,
		InputTaxAmount:   3_800_000,
		OriginalPermille: &none,
		Note:             "Anschaffung für steuerfreie Vermietung, ohne Vorsteuerabzug",
	})
	if err != nil {
		t.Fatalf("Aufnahme ins Verzeichnis: %v", err)
	}
	if correction.OriginalPermille != 0 {
		t.Fatalf("ursprünglicher Anteil %d ‰ — erwartet 0 ‰",
			correction.OriginalPermille)
	}

	// Ohne Angabe bleibt es beim Regelfall: volle Verwendung.
	silent, err := svc.Register(ctx, RegisterInputTaxRequest{
		Label: "Pkw AN-2026-0003", Account: "0520",
		AcquisitionDate: "2026-02-01",
		NetAmount:       4_000_000, InputTaxAmount: 760_000,
	})
	if err != nil {
		t.Fatalf("Aufnahme ohne Anteil: %v", err)
	}
	if silent.OriginalPermille != 1000 {
		t.Errorf("ohne Angabe %d ‰ — erwartet 1000 ‰", silent.OriginalPermille)
	}

	// Und die Erhöhung wird gerechnet: von 0 auf 1000 ‰ im Berichtigungszeitraum
	// eines Grundstücks sind 3.800.000 / 10 = 380.000 Cent zugunsten des
	// Anwenders.
	view, err := svc.SaveUsage(ctx, SaveInputTaxUsageRequest{
		CorrectionID: correction.ID, FiscalYear: 2028, Permille: 1000,
		Reason: "Halle ab 2028 vollständig für steuerpflichtige Umsätze genutzt",
	})
	if err != nil {
		t.Fatalf("Verwendungsanteil speichern: %v", err)
	}
	var found bool
	for _, row := range view.Rows {
		if row.Correction.ID != correction.ID {
			continue
		}
		found = true
		if row.Assessment.Amount != domain.Cents(380_000) {
			t.Errorf("Berichtigung %s — erwartet 3.800,00 € zugunsten des Anwenders",
				row.Assessment.Amount)
		}
	}
	if !found {
		t.Fatalf("die Halle steht nicht im Verzeichnis des Jahres (%d Zeilen)", len(view.Rows))
	}
}

// -------------------------------------------------------------------------
// Die Beförderungsart wird gespeichert
// -------------------------------------------------------------------------

// Die Auswahl der Beförderungsart bleibt an der Rechnung stehen.
//
// Sie war bisher nur ein Parameter der Ansicht: wer sie umstellte und nichts
// weiter tat, fand nach dem Neuladen wieder den alten Wert — und der Abholfall
// braucht die Gelangensbestätigung, die die Bewertung dann nicht verlangte. Ein
// Feld, das wie eine gespeicherte Einstellung aussieht und keine ist, ist
// schlimmer als keines.
func TestSupplyTransportIsPersisted(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.supplyEvidence(t)
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	invoice := env.icSupply(t, customer, "RE-2026-0001")

	view, err := svc.SetTransport(ctx, invoice.ID, string(accounting.TransportByCustomer))
	if err != nil {
		t.Fatalf("Beförderungsart setzen: %v", err)
	}
	if view.Transport != accounting.TransportByCustomer {
		t.Fatalf("Ansicht meldet %q — erwartet den Abholfall", view.Transport)
	}

	// Ohne weitere Angabe gelesen: der gespeicherte Wert trägt, nicht der
	// Regelfall.
	again, err := svc.View(ctx, invoice.ID, "")
	if err != nil {
		t.Fatalf("Nachweisstand: %v", err)
	}
	if again.Transport != accounting.TransportByCustomer {
		t.Errorf("nach dem Neuladen %q — die Auswahl ist nicht gespeichert", again.Transport)
	}

	// Ein unbekannter Wert wird abgewiesen und ändert nichts.
	if _, err := svc.SetTransport(ctx, invoice.ID, "luftpost"); err == nil {
		t.Error("eine unbekannte Beförderungsart darf nicht gespeichert werden")
	}
	unchanged, err := svc.View(ctx, invoice.ID, "")
	if err != nil {
		t.Fatalf("Nachweisstand: %v", err)
	}
	if unchanged.Transport != accounting.TransportByCustomer {
		t.Errorf("nach der abgewiesenen Eingabe %q — der Wert hat sich geändert",
			unchanged.Transport)
	}
}
