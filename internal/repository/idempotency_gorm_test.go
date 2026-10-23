package repository

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Ob ein Schlüssel abgelaufen ist, hängt nicht von der Zone ab, in der sein
// Ablauf angegeben wurde.
func TestIdempotencyDeleteExpiredIgnoresTheZone(t *testing.T) {
	db := auditTestDB(t)
	repo := NewIdempotencyRepository(db)
	ctx := context.Background()
	now := time.Date(2026, 10, 19, 11, 0, 0, 0, time.UTC)

	zones := []*time.Location{
		time.UTC,
		time.FixedZone("MESZ", 2*3600),
		time.FixedZone("JST", 9*3600),
		time.FixedZone("EDT", -4*3600),
	}
	var valid []string
	for _, zone := range zones {
		for _, rec := range []struct {
			key     string
			expires time.Time
		}{
			{"abgelaufen-" + zone.String(), now.Add(-time.Hour)},
			{"gueltig-" + zone.String(), now.Add(time.Hour)},
		} {
			if err := repo.Create(ctx, &domain.IdempotencyRecord{
				ClientID: "agent", Key: rec.key, Operation: "test", RequestHash: "x",
				ExpiresAt: rec.expires.In(zone),
			}); err != nil {
				t.Fatal(err)
			}
		}
		valid = append(valid, "gueltig-"+zone.String())
	}

	n, err := repo.DeleteExpired(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	var remaining []string
	if err := db.Model(&domain.IdempotencyRecord{}).Pluck("key", &remaining).Error; err != nil {
		t.Fatal(err)
	}
	slices.Sort(remaining)
	slices.Sort(valid)
	if n != int64(len(zones)) || !slices.Equal(remaining, valid) {
		t.Fatalf("%d gelöscht, übrig %v; erwartet %d gelöscht, übrig %v", n, remaining, len(zones), valid)
	}
}
