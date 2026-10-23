package wailsbridge

import (
	"context"
	"sort"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Die Übersicht: die Pflichten aus dem Pflichtenkalender, die der Mensch in der
// App bestätigt, und die Buchungen der verbundenen Apps seit der letzten
// Festschreibung.

// ObligationView ist eine Pflicht mit der Stelle der Oberfläche, an der sie
// vorbereitet oder bestätigt wird.
type ObligationView struct {
	domain.Obligation
	Target domain.TaskTarget `json:"target"`
}

func obligationView(o *domain.Obligation) *ObligationView {
	return &ObligationView{Obligation: *o, Target: service.ObligationTarget(*o)}
}

// GetNextObligations liefert die Pflichten, die noch etwas verlangen:
// überfällige, offene und vorbereitete, nach Fälligkeit. limit ≤ 0 heißt alle.
func (b *BuchfinkBridge) GetNextObligations(limit int) ([]ObligationView, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]ObligationView, error) {
		list, err := s.Obligations.Next(ctx, "", limit)
		if err != nil {
			return nil, err
		}
		out := make([]ObligationView, 0, len(list))
		for i := range list {
			out = append(out, *obligationView(&list[i]))
		}
		return out, nil
	})
}

// GetObligation liefert eine Pflicht mit Stand, Vorbereitung und Nachweis.
func (b *BuchfinkBridge) GetObligation(key string) (*ObligationView, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*ObligationView, error) {
		o, err := s.Obligations.Get(ctx, key, "")
		if err != nil {
			return nil, err
		}
		return obligationView(o), nil
	})
}

// ConfirmObligation bestätigt eine Pflicht als erledigt, mit Datum und
// Nachweis (Transferticket, Aktenzeichen, Auftragsnummer oder Bestätigung).
// Nur in der App: ein Agent meldet höchstens „vorbereitet“.
func (b *BuchfinkBridge) ConfirmObligation(key string, evidence service.ConfirmInput) (*ObligationView, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*ObligationView, error) {
		o, err := s.Obligations.Confirm(ctx, key, evidence)
		if err != nil {
			return nil, err
		}
		return obligationView(o), nil
	}, core.AppOnly())
}

// WaiveObligation kennzeichnet eine Pflicht mit Grund als entfallen. Nur in
// der App.
func (b *BuchfinkBridge) WaiveObligation(key, reason string) (*ObligationView, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*ObligationView, error) {
		o, err := s.Obligations.Waive(ctx, key, reason)
		if err != nil {
			return nil, err
		}
		return obligationView(o), nil
	}, core.AppOnly())
}

// ReopenObligation nimmt eine Bestätigung oder ein Entfallen mit Grund
// zurück. Nur in der App.
func (b *BuchfinkBridge) ReopenObligation(key, reason string) (*ObligationView, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*ObligationView, error) {
		o, err := s.Obligations.Reopen(ctx, key, reason)
		if err != nil {
			return nil, err
		}
		return obligationView(o), nil
	}, core.AppOnly())
}

// AgentBookings sind die Buchungen verbundener Apps, die noch nicht
// festgeschrieben sind. Bis zur Festschreibung prüft der Mensch sie; eine
// falsche hebt ein Storno auf.
type AgentBookings struct {
	// LastCommit ist die Festschreibung mit dem jüngsten Stichtag; ohne
	// Festschreibung fehlt sie.
	LastCommit *domain.Festschreibung `json:"lastCommit,omitempty"`
	// Total zählt alle solchen Buchungen, Entries enthält die jüngsten.
	Total   int            `json:"total"`
	Entries []AgentBooking `json:"entries"`
}

// AgentBooking ist eine Buchung einer verbundenen App.
type AgentBooking struct {
	domain.JournalEntry
	// Reversed: eine Generalumkehr hebt die Buchung auf.
	Reversed bool `json:"reversed"`
}

// GetAgentBookings liefert die noch nicht festgeschriebenen Buchungen mit
// Erfassungsweg agent:*, über alle Geschäftsjahre, die jüngste zuerst. limit ≤ 0
// heißt alle.
//
// Ein festgeschriebenes Geschäftsjahr enthält keine solche Buchung mehr und
// wird nicht gelesen.
func (b *BuchfinkBridge) GetAgentBookings(limit int) (*AgentBookings, error) {
	return readOr(b, func() (*AgentBookings, error) {
		return &AgentBookings{Entries: []AgentBooking{}}, nil
	}, func(ctx context.Context, s *core.Services) (*AgentBookings, error) {
		years, err := s.Repos.FiscalYears.FindAll(ctx)
		if err != nil {
			return nil, err
		}
		out := &AgentBookings{Entries: []AgentBooking{}}
		reversed := map[uint]bool{}
		var found []domain.JournalEntry
		for _, fy := range years {
			commits, err := s.Repos.Festschreibung.FindByFiscalYear(ctx, fy.Year)
			if err != nil {
				return nil, err
			}
			closed := false
			for i := range commits {
				if out.LastCommit == nil || commits[i].CutoffDate > out.LastCommit.CutoffDate {
					out.LastCommit = &commits[i]
				}
				closed = closed || commits[i].PeriodType == domain.PeriodTypeYear
			}
			if closed {
				continue
			}
			entries, err := s.Repos.Journal.FindAll(ctx, fy.Year)
			if err != nil {
				return nil, err
			}
			for _, e := range entries {
				if e.ReversalOfID != nil {
					reversed[*e.ReversalOfID] = true
				}
				if e.CommittedAt == nil && actor.Channel(e.Channel).IsAgent() {
					found = append(found, e)
				}
			}
		}
		sort.SliceStable(found, func(i, j int) bool {
			if !found[i].CreatedAt.Equal(found[j].CreatedAt) {
				return found[i].CreatedAt.After(found[j].CreatedAt)
			}
			return found[i].ID > found[j].ID
		})
		out.Total = len(found)
		if limit > 0 && len(found) > limit {
			found = found[:limit]
		}
		for _, e := range found {
			if e.Lines == nil {
				e.Lines = []domain.JournalLine{}
			}
			out.Entries = append(out.Entries, AgentBooking{JournalEntry: e, Reversed: reversed[e.ID]})
		}
		return out, nil
	})
}
