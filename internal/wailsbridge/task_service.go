package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Die Bedienung der Welle 7: Aufgabenliste, Monatsabschluss,
// Zuordnungsvorschlag und Prüfpfad.
//
// Sie stehen in einer eigenen Datei, weil sie eine Sache sind: die Antwort auf
// die Frage, was eine Anwenderin ohne Buchhaltungsausbildung als Nächstes tun
// soll — und wie sie das, was sie tut, wieder findet.

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
