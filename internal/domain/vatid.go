package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Die Bestätigung ausländischer USt-IdNr. beim Bundeszentralamt für Steuern.
//
// § 6a Abs. 1 Satz 1 Nr. 4 UStG macht die gültige, vom Bestimmungsland erteilte
// USt-IdNr. des Abnehmers zur materiellen Voraussetzung der Steuerbefreiung; wer
// sie nicht prüft, trägt das Risiko. Die qualifizierte Bestätigung nach § 18e
// UStG ist der Nachweis dafür — und sie ist nur etwas wert, wenn sie
// aufgehoben wird: eine Auskunft, die nirgends steht, hat es für den Prüfer
// nicht gegeben.

// VatIDCheckStatus fasst das Ergebnis einer Abfrage zusammen.
type VatIDCheckStatus string

const (
	// VatIDValid: die USt-IdNr. ist gültig (REST-Code evatr-0000, in der
	// abgeschalteten XML-RPC-Fassung 200).
	VatIDValid VatIDCheckStatus = "valid"
	// VatIDInvalid: die USt-IdNr. ist nicht vergeben, zum Anfragezeitpunkt nicht
	// gültig oder syntaktisch falsch (REST-Codes evatr-2001, -2002, -2006,
	// -0005, -0012, -2003; früher 201 bis 223).
	VatIDInvalid VatIDCheckStatus = "invalid"
	// VatIDUnavailable: das Bundeszentralamt war nicht erreichbar oder hat einen
	// Fehler gemeldet. Das ist kein negatives Ergebnis — es ist gar keins, und
	// die beiden dürfen nicht dasselbe bedeuten.
	VatIDUnavailable VatIDCheckStatus = "unavailable"
)

// Label ist der Klartext für Meldungen und Oberfläche.
func (s VatIDCheckStatus) Label() string {
	switch s {
	case VatIDValid:
		return "gültig"
	case VatIDInvalid:
		return "ungültig"
	case VatIDUnavailable:
		return "nicht erreichbar"
	default:
		return string(s)
	}
}

// VatIDFieldResult ist die Rückmeldung des Bundeszentralamts zu einem einzelnen
// Feld der qualifizierten Abfrage.
//
// Die Schnittstelle antwortet je Feld mit A (stimmt überein), B (stimmt nicht
// überein), C (nicht angefragt) oder D (vom Mitgliedstaat nicht mitgeteilt). Der
// Unterschied zwischen B und D ist der Unterschied zwischen „falsch" und
// „unbekannt", und er entscheidet darüber, ob eine Rechnung ein Problem hat.
type VatIDFieldResult string

const (
	VatIDFieldMatch    VatIDFieldResult = "A"
	VatIDFieldMismatch VatIDFieldResult = "B"
	VatIDFieldNotAsked VatIDFieldResult = "C"
	VatIDFieldUnknown  VatIDFieldResult = "D"
)

// Label ist der Klartext eines Feldergebnisses.
func (r VatIDFieldResult) Label() string {
	switch r {
	case VatIDFieldMatch:
		return "stimmt überein"
	case VatIDFieldMismatch:
		return "stimmt nicht überein"
	case VatIDFieldNotAsked:
		return "nicht angefragt"
	case VatIDFieldUnknown:
		return "vom Mitgliedstaat nicht mitgeteilt"
	default:
		return string(r)
	}
}

// VatIDCheck ist eine aufgehobene Bestätigungsabfrage.
type VatIDCheck struct {
	ID        uint `gorm:"primaryKey" json:"id"`
	ContactID uint `gorm:"index;not null" json:"contactId"`

	// VatID ist die geprüfte Nummer in der Schreibweise, in der sie abgefragt
	// wurde. Sie steht am Ergebnis und nicht nur am Kontakt: wird die Nummer
	// später geändert, gilt die alte Bestätigung nicht mehr für sie.
	VatID string `gorm:"size:50;not null;index;serializer:encrypted" json:"vatId"`
	// OwnVatID ist die eigene USt-IdNr., mit der abgefragt wurde. Ohne sie ist
	// die Abfrage keine qualifizierte.
	OwnVatID string `gorm:"size:50;serializer:encrypted" json:"ownVatId,omitempty"`

	// CheckedAt ist der Zeitpunkt der Abfrage als RFC3339. Die Frist von 90
	// Tagen hängt an ihm.
	CheckedAt string           `gorm:"size:25;not null;index" json:"checkedAt"`
	Status    VatIDCheckStatus `gorm:"size:20;not null;index" json:"status"`
	// ResultCode ist der Ergebniscode des Bundeszentralamts („evatr-0000",
	// „evatr-2001" …; aus der alten XML-RPC-Fassung 200, 201 …).
	ResultCode string `gorm:"size:10" json:"resultCode,omitempty"`
	// ResultText ist der Wortlaut, den die Schnittstelle dazu liefert.
	ResultText string `gorm:"size:500;serializer:encrypted" json:"resultText,omitempty"`
	// RequestID ist die Abfrage-Identifikationsnummer aus der Antwort. Sie ist
	// der Beleg gegenüber der Finanzverwaltung.
	RequestID string `gorm:"size:60;serializer:encrypted" json:"requestId,omitempty"`

	// Die vier Feldergebnisse der qualifizierten Abfrage.
	NameResult       VatIDFieldResult `gorm:"size:1" json:"nameResult,omitempty"`
	CityResult       VatIDFieldResult `gorm:"size:1" json:"cityResult,omitempty"`
	PostalCodeResult VatIDFieldResult `gorm:"size:1" json:"postalCodeResult,omitempty"`
	StreetResult     VatIDFieldResult `gorm:"size:1" json:"streetResult,omitempty"`

	// RawResponse ist die Antwort, wie sie kam. GoBD Rz. 130: der empfangene
	// Datensatz bleibt in seiner ursprünglichen Form erhalten — eine
	// aufbereitete Zusammenfassung ist kein Nachweis.
	RawResponse string `gorm:"type:text;serializer:encrypted" json:"rawResponse,omitempty"`

	// Endpoint hält fest, wohin gefragt wurde. Die Adresse ist eine Einstellung;
	// ohne diesen Vermerk wäre später nicht mehr erkennbar, welche Stelle
	// geantwortet hat.
	Endpoint string `gorm:"size:255" json:"endpoint,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

// VatIDCheckValidityDays ist die Frist, für die Buchfink eine Bestätigung als
// hinreichend frisch ansieht.
//
// Das Gesetz nennt keine Frist — es verlangt, dass der Unternehmer die
// Sorgfalt eines ordentlichen Kaufmanns anwendet (§ 6a Abs. 4 UStG). Bei
// laufenden Geschäftsbeziehungen ist eine regelmäßige, nicht bloß einmalige
// Abfrage üblich; 90 Tage sind der Wert, den Buchfink dafür setzt. Er ist eine
// Konvention und wird als solche benannt, nicht als Rechtsnorm ausgegeben.
const VatIDCheckValidityDays = 90

// Fresh meldet, ob eine gültige Bestätigung zum Stichtag noch innerhalb der
// Frist liegt.
func (c *VatIDCheck) Fresh(at time.Time) bool {
	if c.Status != VatIDValid {
		return false
	}
	checked, err := time.Parse(time.RFC3339, c.CheckedAt)
	if err != nil {
		return false
	}
	return !checked.Add(VatIDCheckValidityDays * 24 * time.Hour).Before(at)
}

// Summary ist ein Satz über das Ergebnis, wie er in einer Fehlermeldung steht.
func (c *VatIDCheck) Summary() string {
	parts := make([]string, 0, 4)
	for _, f := range []struct {
		label  string
		result VatIDFieldResult
	}{
		{"Name", c.NameResult},
		{"Ort", c.CityResult},
		{"PLZ", c.PostalCodeResult},
		{"Straße", c.StreetResult},
	} {
		if f.result == VatIDFieldMismatch {
			parts = append(parts, f.label)
		}
	}
	out := fmt.Sprintf("Ergebnis %s (Code %s)", c.Status.Label(), c.ResultCode)
	if c.ResultText != "" {
		out += ": " + c.ResultText
	}
	if len(parts) > 0 {
		out += ". Abweichend: " + strings.Join(parts, ", ")
	}
	return out
}

// VatIDCheckRepository persistiert die Abfragen.
//
// Gelöscht wird nichts: die Historie ist der Nachweis, dass regelmäßig geprüft
// wurde, und ein einzelnes Ergebnis ohne sie sagt nur etwas über einen Tag aus.
type VatIDCheckRepository interface {
	FindByContact(ctx context.Context, contactID uint) ([]VatIDCheck, error)
	// FindLatestValid liefert die jüngste gültige Bestätigung zu einer Nummer,
	// oder nil.
	FindLatestValid(ctx context.Context, contactID uint, vatID string) (*VatIDCheck, error)
	Save(ctx context.Context, check *VatIDCheck) error
}
