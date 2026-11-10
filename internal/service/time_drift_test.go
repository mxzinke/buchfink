package service

import (
	"strings"
	"testing"
	"time"
)

// --- Zeitabgleich mit dem Zeitstempeldienst -------------------------------

func TestTimeDriftNoteOnlyBeyondTolerance(t *testing.T) {
	trusted := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	if note := TimeDriftNote(trusted.Add(2*time.Minute), trusted, "TSA"); note != "" {
		t.Errorf("zwei Minuten liegen in der Toleranz, bekommen: %q", note)
	}
	if note := TimeDriftNote(trusted.Add(-2*time.Minute), trusted, "TSA"); note != "" {
		t.Errorf("zwei Minuten Rückstand liegen in der Toleranz, bekommen: %q", note)
	}

	ahead := TimeDriftNote(trusted.Add(90*time.Minute), trusted, "DigiCert")
	if ahead == "" {
		t.Fatal("90 Minuten Abweichung müssen gemeldet werden")
	}
	if !strings.Contains(ahead, "geht vor") || !strings.Contains(ahead, "DigiCert") {
		t.Errorf("die Meldung muss Richtung und Dienst nennen: %q", ahead)
	}

	behind := TimeDriftNote(trusted.Add(-3*24*time.Hour), trusted, "DigiCert")
	if !strings.Contains(behind, "geht nach") || !strings.Contains(behind, "3 Tage") {
		t.Errorf("die Meldung muss Richtung und Größe nennen: %q", behind)
	}
}
