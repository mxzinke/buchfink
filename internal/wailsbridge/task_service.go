package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// --- Aufgabenliste --------------------------------------------------------

// GetTasks liefert die Aufgabenliste in ihren drei Gruppen. Der Prüfermodus
// kommt aus dem Kern und wird mitgegeben.
func (b *BuchfinkBridge) GetTasks() (*domain.TaskList, error) {
	cfg := b.host.Config()
	opts := service.TaskOptions{FiscalYear: b.viewYear()}
	if cfg.ReadOnly {
		opts.ReadOnlyUntil, opts.ReadOnlyReason = cfg.ReadOnlyUntil, cfg.ReadOnlyReason
	}
	return readOr(b, func() (*domain.TaskList, error) {
		empty := &domain.TaskList{}
		empty.EnsureLists()
		return empty, nil
	}, func(ctx context.Context, s *core.Services) (*domain.TaskList, error) {
		list, err := s.Tasks.Tasks(ctx, opts)
		if err != nil {
			return nil, err
		}
		list.EnsureLists()
		return list, nil
	})
}
