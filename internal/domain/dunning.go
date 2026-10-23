package domain

import (
	"context"
	"time"
)

// DunningNote ist der Vermerk, dass ein offener Posten gemahnt wurde. Das
// Schreiben selbst entsteht außerhalb von Buchfink. Der Vermerk zählt, weil
// gegenüber einem Verbraucher ohne Hinweis in der Rechnung der Verzug erst
// mit der Mahnung eintritt (§ 286 Abs. 1 und 3 BGB).
type DunningNote struct {
	ID      uint `gorm:"primaryKey" json:"id"`
	EntryID uint `gorm:"index;not null" json:"entryId"`
	// Date ist der Tag, an dem die Mahnung dem Kunden zugegangen ist.
	Date      string    `gorm:"size:10;not null" json:"date"`
	Level     int       `gorm:"not null;default:1" json:"level"`
	Note      string    `gorm:"size:1000" json:"note,omitempty"`
	Channel   string    `gorm:"size:60" json:"channel"`
	Actor     string    `gorm:"size:200" json:"actor"`
	CreatedAt time.Time `json:"createdAt"`
}

// DunningRepository speichert die Mahnvermerke.
type DunningRepository interface {
	Create(ctx context.Context, note *DunningNote) error
	FindAll(ctx context.Context) ([]DunningNote, error)
}
