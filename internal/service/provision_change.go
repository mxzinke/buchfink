package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

type ProvisionChangePreview struct {
	Provision domain.Provision             `json:"provision"`
	Entry     domain.JournalEntry          `json:"entry"`
	Kind      domain.ProvisionMovementKind `json:"kind"`
	Amount    domain.Cents                 `json:"amount"`
	Reason    string                       `json:"reason"`
}

func (s *ProvisionService) PreviewChange(ctx context.Context, kind domain.ProvisionMovementKind, req ProvisionChangeRequest) (*ProvisionChangePreview, error) {
	if strings.TrimSpace(req.Reason) == "" {
		return nil, fmt.Errorf("die Bewegung braucht Grund und Nachweis")
	}
	if req.Amount < 0 {
		return nil, fmt.Errorf("der Betrag darf nicht negativ sein")
	}
	p, err := s.load(ctx, req.ProvisionID)
	if err != nil {
		return nil, err
	}
	if p.Kind.IsTax() {
		return nil, fmt.Errorf("Steuerrückstellungen werden im eigenen Abschlussbaustein bearbeitet")
	}
	for _, m := range p.Movements {
		if m.Date > req.Date {
			return nil, fmt.Errorf("das Datum liegt vor einer bestehenden Bewegung; spätere Bewegungen zuerst stornieren")
		}
	}
	switch kind {
	case domain.ProvisionRelease:
		return s.previewRelease(ctx, req)
	case domain.ProvisionUnwinding:
		return s.previewUnwinding(ctx, req)
	default:
		return nil, fmt.Errorf("unbekannte Bewegung %q", kind)
	}
}

func (s *ProvisionService) BookChange(ctx context.Context, kind domain.ProvisionMovementKind, req ProvisionChangeRequest) (*domain.Provision, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.Provision, error) {
		p, err := s.PreviewChange(ctx, kind, req)
		if err != nil {
			return nil, err
		}
		return s.applyMovement(ctx, &p.Provision, &p.Entry, kind, p.Amount, req.Date, req.Reason, kind.Label()+": "+req.Reason)
	})
}

type ProvisionRateEvidence struct {
	Rate   ProvisionRate `json:"rate"`
	Source string        `json:"source"`
}

// ProvisionRate enthält nur die fachlichen Angaben des veröffentlichten Satzes.
type ProvisionRate struct {
	Month      string `json:"month"`
	Years      int    `json:"years"`
	Average    int    `json:"average"`
	RateMicros int64  `json:"rateMicros"`
}
