package wailsbridge

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/currency"
	"github.com/buchfink/buchfink/internal/vatid"
)

// Die Oberfläche muss die beiden Adressen lesen und ändern können, ohne die
// Schlüssel der Einstellungen zu raten.
func TestServiceEndpointsAreReadableAndWritable(t *testing.T) {
	b := testBridge(t)

	first := b.GetServiceEndpoints()
	if first.VatIDDefault != vatid.DefaultEndpoint ||
		first.ExchangeRateDefault != currency.DefaultEndpoint {
		t.Fatalf("die Voreinstellungen fehlen: %+v", first)
	}
	if first.VatID != "" || first.ExchangeRate != "" {
		t.Errorf("ohne Einstellung sind beide Adressen leer: %+v", first)
	}

	saved, err := b.SaveServiceEndpoints(ServiceEndpoints{
		VatID:        "https://api.evatr.example/app/v1/abfrage",
		ExchangeRate: "https://kurse.example",
	})
	if err != nil {
		t.Fatalf("Adressen speichern: %v", err)
	}
	if saved.VatID != "https://api.evatr.example/app/v1/abfrage" ||
		saved.ExchangeRate != "https://kurse.example" {
		t.Errorf("gespeichert %+v", saved)
	}
	if again := b.GetServiceEndpoints(); again.ExchangeRate != saved.ExchangeRate {
		t.Errorf("die Adresse steht nach dem Speichern nicht in den Einstellungen: %+v", again)
	}

	// Unverschlüsselt geht nicht: die Abfrage trägt die USt-IdNr. und den Namen
	// des Geschäftspartners.
	if _, err := b.SaveServiceEndpoints(ServiceEndpoints{
		VatID: "http://evatr.example/abfrage",
	}); err == nil {
		t.Error("eine unverschlüsselte Adresse darf nicht gespeichert werden")
	} else if !strings.Contains(err.Error(), "https") {
		t.Errorf("die Meldung muss das Verfahren nennen: %v", err)
	}
}
