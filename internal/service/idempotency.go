package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// IdempotencyRetention ist die Zeit, die ein Idempotenzschlüssel gilt.
const IdempotencyRetention = 30 * 24 * time.Hour

// IdempotentCall beschreibt einen Schreibvorgang unter einem Schlüssel des
// Aufrufers.
type IdempotentCall struct {
	ClientID  string
	Key       string
	Operation string
	// Request ist die Eingabe ohne den Schlüssel.
	Request any
}

// IdempotentOutcome ist das Ergebnis eines solchen Vorgangs. Replayed heißt:
// der Vorgang lief schon einmal, das Ergebnis stammt aus dem ersten Lauf.
type IdempotentOutcome struct {
	ResultRef string          `json:"resultRef"`
	Response  json.RawMessage `json:"response"`
	Replayed  bool            `json:"replayed"`
}

// IdempotencyService sorgt dafür, dass ein Vorgang unter einem Schlüssel genau
// einmal geschieht.
type IdempotencyService struct {
	repo domain.IdempotencyRepository
	tx   domain.TxRunner
	now  func() time.Time
}

// NewIdempotencyService verdrahtet die Ablage der Schlüssel.
func NewIdempotencyService(repo domain.IdempotencyRepository, tx domain.TxRunner) *IdempotencyService {
	return &IdempotencyService{repo: repo, tx: tx, now: time.Now}
}

// SetClock ersetzt die Uhr, für Tests.
func (s *IdempotencyService) SetClock(now func() time.Time) { s.now = now }

// Run führt fn aus, wenn der Schlüssel neu ist, und hält das Ergebnis fest.
// Kommt derselbe Schlüssel mit derselben Eingabe wieder, liefert Run das
// gespeicherte Ergebnis, ohne fn aufzurufen; mit einer anderen Eingabe
// domain.ErrIdempotencyKeyReused. Nachschlagen, Vorgang und Eintrag stehen in
// einer Transaktion: scheitert fn, bleibt kein Eintrag zurück.
func (s *IdempotencyService) Run(
	ctx context.Context, call IdempotentCall,
	fn func(ctx context.Context) (resultRef string, response any, err error),
) (*IdempotentOutcome, error) {
	if strings.TrimSpace(call.ClientID) == "" || strings.TrimSpace(call.Operation) == "" {
		return nil, fmt.Errorf("zu einem idempotenten Vorgang gehören Aufrufer und Vorgang")
	}
	if err := domain.ValidateIdempotencyKey(call.Key); err != nil {
		return nil, err
	}
	hash, err := domain.IdempotencyRequestHash(call.Operation, call.Request)
	if err != nil {
		return nil, err
	}

	var outcome *IdempotentOutcome
	err = s.tx.RunInTx(ctx, func(ctx context.Context) error {
		now := s.now().UTC()
		existing, err := s.repo.Find(ctx, call.ClientID, call.Key)
		if err != nil {
			return fmt.Errorf("der Idempotenzschlüssel ließ sich nicht nachschlagen: %w", err)
		}
		if existing != nil && existing.Expired(now) {
			if err := s.repo.Delete(ctx, existing.ID); err != nil {
				return err
			}
			existing = nil
		}
		if existing != nil {
			if existing.RequestHash != hash {
				return fmt.Errorf("%w (zuerst für „%s\" am %s)", domain.ErrIdempotencyKeyReused,
					existing.Operation, existing.CreatedAt.UTC().Format(time.RFC3339))
			}
			outcome = &IdempotentOutcome{
				ResultRef: existing.ResultRef, Response: json.RawMessage(existing.Response), Replayed: true,
			}
			return nil
		}

		ref, response, err := fn(ctx)
		if err != nil {
			return err
		}
		body, err := json.Marshal(response)
		if err != nil {
			return fmt.Errorf("die Antwort ließ sich nicht festhalten: %w", err)
		}
		if err := s.repo.Create(ctx, &domain.IdempotencyRecord{
			ClientID: call.ClientID, Key: call.Key, Operation: call.Operation,
			RequestHash: hash, ResultRef: ref, Response: string(body),
			CreatedAt: now, ExpiresAt: now.Add(IdempotencyRetention),
		}); err != nil {
			return fmt.Errorf("der Idempotenzschlüssel ließ sich nicht festhalten: %w", err)
		}
		outcome = &IdempotentOutcome{ResultRef: ref, Response: body}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return outcome, nil
}

// Prune entfernt die abgelaufenen Schlüssel und liefert ihre Zahl.
func (s *IdempotencyService) Prune(ctx context.Context) (int64, error) {
	return s.repo.DeleteExpired(ctx, s.now().UTC())
}
