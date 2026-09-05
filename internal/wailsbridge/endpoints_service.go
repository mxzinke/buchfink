package wailsbridge

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/buchfink/buchfink/internal/currency"
	"github.com/buchfink/buchfink/internal/service"
	"github.com/buchfink/buchfink/internal/vatid"
)

// Die Adressen der beiden Netzdienste.
//
// Buchfink ruft von sich aus zwei fremde Stellen: das Bundeszentralamt für
// Steuern für die Bestätigungsanfrage zur USt-IdNr. und den Kursdienst für die
// Referenzkurse der Europäischen Zentralbank. Beide Adressen sind Einstellungen
// und keine Literale im Code — wechselt eine Stelle ihre Schnittstelle, soll das
// keine neue Programmfassung nötig machen.
//
// Damit die Oberfläche sie zeigen und ändern kann, ohne die Schlüssel zu raten,
// gehen sie über diese beiden Methoden und nicht über einen freien
// Einstellungszugriff: ein geratener Schlüssel fiele nirgends auf, weil ein
// fehlender Wert die Voreinstellung bedeutet.

// ServiceEndpoints sind die eingestellten Adressen samt ihren Voreinstellungen.
type ServiceEndpoints struct {
	// VatID ist die Adresse der Bestätigungsanfrage, VatIDDefault die
	// Voreinstellung, auf die ein leerer Wert zurückfällt.
	VatID        string `json:"vatIdEndpoint"`
	VatIDDefault string `json:"vatIdDefault"`
	// ExchangeRate ist die Adresse des Kursdienstes.
	ExchangeRate        string `json:"exchangeRateEndpoint"`
	ExchangeRateDefault string `json:"exchangeRateDefault"`
}

// GetServiceEndpoints liefert die Adressen der beiden Netzdienste.
func (b *BuchfinkBridge) GetServiceEndpoints() ServiceEndpoints {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.serviceEndpointsLocked()
}

// serviceEndpointsLocked liest die Adressen; die Sperre hält der Aufrufer.
//
// Getrennt von GetServiceEndpoints, weil das Fortschreiben unter der
// Schreibsperre steht und dieselbe Antwort zurückgibt: ein Aufruf der
// öffentlichen Methode von dort aus liefe in die eigene Sperre.
func (b *BuchfinkBridge) serviceEndpointsLocked() ServiceEndpoints {
	return ServiceEndpoints{
		VatID:               b.settingValue(service.SettingVatIDEndpoint),
		VatIDDefault:        vatid.DefaultEndpoint,
		ExchangeRate:        b.settingValue(service.SettingExchangeRateEndpoint),
		ExchangeRateDefault: currency.DefaultEndpoint,
	}
}

// SaveServiceEndpoints schreibt die Adressen fort. Ein leerer Wert setzt die
// jeweilige Voreinstellung wieder in Kraft.
func (b *BuchfinkBridge) SaveServiceEndpoints(in ServiceEndpoints) (ServiceEndpoints, error) {
	// Unter der Sperre wie jede andere schreibende Methode: ensureWritable liest
	// den Prüfermodus über readOnlyStateLocked, und dessen Name sagt, dass die
	// Sperre dabei zu halten ist. Ohne sie liefe die Prüfung gegen einen Zustand,
	// den das Umschalten des Prüfermodus gerade wechselt.
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return ServiceEndpoints{}, err
	}
	repo := b.settingsRepo
	if repo == nil {
		return ServiceEndpoints{}, fmt.Errorf("es ist kein Mandant geöffnet")
	}

	ctx := context.Background()
	for key, value := range map[string]string{
		service.SettingVatIDEndpoint:        in.VatID,
		service.SettingExchangeRateEndpoint: in.ExchangeRate,
	} {
		trimmed := strings.TrimSpace(value)
		if err := validEndpoint(trimmed); err != nil {
			return ServiceEndpoints{}, err
		}
		if err := repo.Set(ctx, key, trimmed); err != nil {
			return ServiceEndpoints{}, fmt.Errorf("die Adresse ließ sich nicht speichern: %w", err)
		}
	}
	return b.serviceEndpointsLocked(), nil
}

// validEndpoint weist ab, was keine Adresse ist. Leer bleibt zulässig: das ist
// die Voreinstellung.
func validEndpoint(value string) error {
	if value == "" {
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("%q ist keine vollständige Adresse (erwartet https://…)", value)
	}
	if parsed.Scheme != "https" {
		// Die Abfrage trägt eine USt-IdNr. und den Namen des Geschäftspartners.
		// Sie geht verschlüsselt oder gar nicht.
		return fmt.Errorf(
			"die Adresse %q ist nicht verschlüsselt. Die Abfrage trägt die USt-IdNr. und den Namen "+
				"des Geschäftspartners; sie geht über https", value)
	}
	return nil
}
