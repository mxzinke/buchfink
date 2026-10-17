package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Prüfermodus friert den Bestand des offenen Mandanten während einer
// Außenprüfung ein (§ 147 Abs. 6 AO): Tenant.Write weist jeden Schreibvorgang
// ab, der nicht ausdrücklich ausgenommen ist. Der Stand steht in der
// Mandantenliste; ein- und ausschalten lässt er sich nur in der App.

// EnableExaminer schaltet den Prüfermodus bis einschließlich until ein. Datum
// und Grund sind Pflicht: ohne Ende bliebe der Modus liegen, ohne Grund ließe
// sich später nicht sagen, welche Prüfung ihn veranlasst hat.
func (h *Host) EnableExaminer(ctx context.Context, until, reason string) (domain.AppConfig, error) {
	if strings.TrimSpace(until) == "" {
		return h.Config(), fmt.Errorf("der Prüfermodus braucht ein Datum, bis zu dem er gilt")
	}
	if _, err := time.Parse("2006-01-02", until); err != nil {
		return h.Config(), fmt.Errorf("%q ist kein gültiges Datum (erwartet JJJJ-MM-TT)", until)
	}
	if until < today() {
		return h.Config(), fmt.Errorf("der Prüfermodus kann nicht in der Vergangenheit enden")
	}
	if strings.TrimSpace(reason) == "" {
		return h.Config(), fmt.Errorf("bitte den Grund angeben, aus dem der Prüfermodus eingeschaltet wird")
	}
	return h.setExaminer(ctx, until, reason, func(domain.TenantConfig) string {
		return fmt.Sprintf("Prüfermodus eingeschaltet bis %s. Grund: %s", until, reason)
	})
}

// DisableExaminer beendet den Prüfermodus vorzeitig. Der Grund ist Pflicht:
// ein vorzeitiges Ende hat jemand zu verantworten.
func (h *Host) DisableExaminer(ctx context.Context, reason string) (domain.AppConfig, error) {
	if strings.TrimSpace(reason) == "" {
		return h.Config(), fmt.Errorf("bitte den Grund angeben, aus dem der Prüfermodus beendet wird")
	}
	return h.setExaminer(ctx, "", "", func(previous domain.TenantConfig) string {
		return fmt.Sprintf("Prüfermodus beendet (war bis %s). Grund: %s", previous.ReadOnlyUntil, reason)
	})
}

// setExaminer schreibt den neuen Stand samt Vorher/Nachher ins Protokoll und
// in die Mandantenliste.
func (h *Host) setExaminer(
	ctx context.Context, until, reason string, describe func(previous domain.TenantConfig) string,
) (domain.AppConfig, error) {
	t, release, err := h.Acquire(ctx)
	if err != nil {
		return h.Config(), fmt.Errorf("kein aktiver Mandant gefunden")
	}
	defer release()

	var cfg domain.AppConfig
	err = t.Write(ctx, func(ctx context.Context, s *Services) error {
		previous, _ := h.tenantConfig(t.ID())
		if err := s.Repos.Audit.LogChange(ctx, domain.AuditActionUpdate, domain.AuditEntityReadOnly, t.ID(),
			describe(previous),
			map[string]string{"readOnlyUntil": previous.ReadOnlyUntil, "readOnlyReason": previous.ReadOnlyReason},
			map[string]string{"readOnlyUntil": until, "readOnlyReason": reason}); err != nil {
			return err
		}
		var err error
		cfg, err = h.updateTenantConfig(t.ID(), func(tc *domain.TenantConfig) error {
			tc.ReadOnlyUntil, tc.ReadOnlyReason = until, reason
			return nil
		})
		return err
	}, ExaminerExempt(), AppOnly())
	if err != nil {
		return h.Config(), err
	}
	return cfg, nil
}
