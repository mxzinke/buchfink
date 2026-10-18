package core

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
)

// ReadAgent führt den Lesevorgang mit der begrenzten Dienstsicht aus.
func (t *Tenant) ReadAgent(ctx context.Context, fn func(context.Context, AgentServices) error) error {
	return t.Read(ctx, func(ctx context.Context, s *Services) error { return fn(ctx, s.AgentServices()) })
}

// WriteAgent verwendet die Schreibsperre, Transaktion und Prüfungen von Write.
func (t *Tenant) WriteAgent(ctx context.Context, fn func(context.Context, AgentServices) error, opts ...WriteOption) error {
	return t.Write(ctx, func(ctx context.Context, s *Services) error { return fn(ctx, s.AgentServices()) }, opts...)
}

// LogAppChange vermerkt Änderungen an der Zulassung einer verbundenen App.
func (t *Tenant) LogAppChange(ctx context.Context, entityID, details string) error {
	return t.Write(ctx, func(ctx context.Context, s *Services) error {
		return s.Repos.Audit.Log(ctx, domain.AuditActionUpdate, "VERBUNDENE_APP", entityID, details)
	}, ExaminerExempt())
}
