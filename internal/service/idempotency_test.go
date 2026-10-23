package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestIdempotentCallRunsOnceAndReplaysTheResult(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := NewIdempotencyService(repository.NewIdempotencyRepository(env.db), repository.NewTxRunner(env.db))

	type request struct {
		Account string       `json:"konto"`
		Amount  domain.Cents `json:"betrag"`
	}
	call := IdempotentCall{ClientID: "client-a", Key: "buchung-0001", Operation: "buchen",
		Request: request{Account: "6815", Amount: 10000}}
	runs := 0
	book := func(ctx context.Context) (string, any, error) {
		runs++
		entry, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 10000))
		if err != nil {
			return "", nil, err
		}
		return fmt.Sprintf("buchung:%d", entry.ID), map[string]string{"nummer": entry.EntryNumber}, nil
	}

	first, err := svc.Run(ctx, call, book)
	if err != nil {
		t.Fatalf("erster Lauf: %v", err)
	}
	second, err := svc.Run(ctx, call, book)
	if err != nil {
		t.Fatalf("Wiederholung: %v", err)
	}
	if runs != 1 || first.Replayed || !second.Replayed {
		t.Errorf("Läufe %d, erster wiederholt %v, zweiter wiederholt %v", runs, first.Replayed, second.Replayed)
	}
	if second.ResultRef != first.ResultRef || string(second.Response) != string(first.Response) {
		t.Errorf("die Wiederholung liefert %+v statt %+v", second, first)
	}
	if n := env.entryCount(t); n != 1 {
		t.Errorf("%d Buchungen, erwartet eine", n)
	}

	changed := call
	changed.Request = request{Account: "6815", Amount: 20000}
	if _, err := svc.Run(ctx, changed, book); !errors.Is(err, domain.ErrIdempotencyKeyReused) {
		t.Errorf("derselbe Schlüssel mit anderer Eingabe muss abgewiesen werden, Fehler: %v", err)
	}
	otherClient := call
	otherClient.ClientID = "client-b"
	if out, err := svc.Run(ctx, otherClient, book); err != nil || out.Replayed {
		t.Errorf("ein anderer Aufrufer hat seinen eigenen Schlüsselraum: %+v, %v", out, err)
	}
}

func TestFailedIdempotentCallLeavesNoKey(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := NewIdempotencyService(repository.NewIdempotencyRepository(env.db), repository.NewTxRunner(env.db))
	call := IdempotentCall{ClientID: "client-a", Key: "buchung-0002", Operation: "buchen", Request: 1}

	if _, err := svc.Run(ctx, call, func(ctx context.Context) (string, any, error) {
		if _, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 10000)); err != nil {
			return "", nil, err
		}
		return "", nil, errors.New("Vorgang gescheitert")
	}); err == nil {
		t.Fatal("der Fehler des Vorgangs muss ankommen")
	}
	if n := env.entryCount(t); n != 0 {
		t.Errorf("%d Buchungen nach dem gescheiterten Vorgang", n)
	}
	out, err := svc.Run(ctx, call, func(ctx context.Context) (string, any, error) { return "x:1", nil, nil })
	if err != nil || out.Replayed {
		t.Errorf("nach dem Fehlschlag ist der Schlüssel frei: %+v, %v", out, err)
	}
}

func TestIdempotencyKeysExpire(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	repo := repository.NewIdempotencyRepository(env.db)
	svc := NewIdempotencyService(repo, repository.NewTxRunner(env.db))
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	svc.SetClock(func() time.Time { return now })
	call := IdempotentCall{ClientID: "client-a", Key: "buchung-0003", Operation: "buchen", Request: 1}
	noop := func(ctx context.Context) (string, any, error) { return "x:1", nil, nil }

	if _, err := svc.Run(ctx, call, noop); err != nil {
		t.Fatal(err)
	}
	now = now.Add(IdempotencyRetention)
	call.Request = 2
	if out, err := svc.Run(ctx, call, noop); err != nil || out.Replayed {
		t.Errorf("ein abgelaufener Schlüssel gilt als neu: %+v, %v", out, err)
	}

	now = now.Add(IdempotencyRetention + time.Hour)
	if n, err := svc.Prune(ctx); err != nil || n != 1 {
		t.Errorf("Prune entfernte %d (%v), erwartet 1", n, err)
	}

	for _, bad := range []string{"kurz", "mit leerzeichen", "ümlaut-schluessel"} {
		call.Key = bad
		if _, err := svc.Run(ctx, call, noop); err == nil {
			t.Errorf("der Schlüssel %q muss abgewiesen werden", bad)
		}
	}
}
