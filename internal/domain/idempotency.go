package domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// IdempotencyRecord hält das Ergebnis eines ausgeführten Schreibvorgangs unter
// dem Schlüssel fest, den der Aufrufer mitgeschickt hat. Wiederholt er den
// Vorgang mit demselben Schlüssel, bekommt er dieses Ergebnis, statt dass der
// Vorgang ein zweites Mal geschieht.
//
// Die Tabelle hält nur Ergebnisse ausgeführter Vorgänge; ein gescheiterter
// Vorgang hinterlässt keinen Eintrag.
type IdempotencyRecord struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	ClientID string `gorm:"size:100;not null;uniqueIndex:idx_idempotency_client_key" json:"clientId"`
	Key      string `gorm:"size:100;not null;uniqueIndex:idx_idempotency_client_key" json:"key"`
	// Operation benennt den Vorgang, etwa „beleg_buchen".
	Operation string `gorm:"size:60;not null" json:"operation"`
	// RequestHash ist SHA-256 über Vorgang und kanonische Eingabe ohne
	// Schlüssel (IdempotencyRequestHash).
	RequestHash string `gorm:"size:64;not null" json:"requestHash"`
	// ResultRef verweist auf das Ergebnis, etwa „buchung:812".
	ResultRef string `gorm:"size:100" json:"resultRef"`
	// Response ist die knappe Antwort des Vorgangs als JSON.
	Response  string    `gorm:"type:text" json:"response"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `gorm:"index;not null" json:"expiresAt"`
}

// Expired meldet, ob der Eintrag zum Zeitpunkt now nicht mehr gilt.
func (r *IdempotencyRecord) Expired(now time.Time) bool {
	return !now.Before(r.ExpiresAt)
}

// ErrIdempotencyKeyReused: derselbe Schlüssel kam mit einer anderen Eingabe.
var ErrIdempotencyKeyReused = errors.New(
	"der Idempotenzschlüssel wurde schon für einen anderen Vorgang verwendet; " +
		"für einen neuen Vorgang braucht es einen neuen Schlüssel")

// ValidateIdempotencyKey prüft einen Schlüssel: 8 bis 100 Zeichen aus
// Buchstaben, Ziffern, „-", „_", „.", „:".
func ValidateIdempotencyKey(key string) error {
	if len(key) < 8 || len(key) > 100 {
		return fmt.Errorf("der Idempotenzschlüssel muss 8 bis 100 Zeichen lang sein, hat aber %d", len(key))
	}
	for _, c := range key {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '-', c == '_', c == '.', c == ':':
		default:
			return fmt.Errorf("der Idempotenzschlüssel enthält das unzulässige Zeichen %q", c)
		}
	}
	return nil
}

// IdempotencyRequestHash bildet den Hash über Vorgang und Eingabe. Kanonisch ist
// die JSON-Form: Felder in der Reihenfolge ihres Typs, Schlüssel von Maps
// sortiert. Der Idempotenzschlüssel selbst gehört nicht in die Eingabe.
func IdempotencyRequestHash(operation string, request any) (string, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("die Eingabe ließ sich nicht kanonisch darstellen: %w", err)
	}
	h := sha256.New()
	h.Write([]byte(operation))
	h.Write([]byte{0})
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// IdempotencyRepository speichert die Ergebnisse ausgeführter Vorgänge.
type IdempotencyRepository interface {
	// Find liefert den Eintrag zu Client und Schlüssel, auch einen abgelaufenen,
	// oder nil.
	Find(ctx context.Context, clientID, key string) (*IdempotencyRecord, error)
	Create(ctx context.Context, record *IdempotencyRecord) error
	Delete(ctx context.Context, id uint) error
	// DeleteExpired entfernt alle Einträge, die zum Zeitpunkt now abgelaufen
	// sind, und liefert ihre Zahl.
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}
