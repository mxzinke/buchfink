package service

import (
	"context"
	"sort"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Jede bestätigte Berichtigung ersetzt nur die darin enthaltenen Schlüssel.
func zmReported(saved []domain.ZMReturn) map[string]domain.ZMLine {
	sort.Slice(saved, func(i, j int) bool { return saved[i].ID < saved[j].ID })
	out := map[string]domain.ZMLine{}
	for _, r := range saved {
		if r.Status != domain.VatReturnSubmitted {
			continue
		}
		for _, l := range r.Lines {
			out[zmKey(l)] = l
		}
	}
	return out
}

func zmKey(l domain.ZMLine) string { return l.VatID + ":" + string(l.Kind) }

func zmChanges(old map[string]domain.ZMLine, current []domain.ZMLine) []domain.ZMLine {
	next := map[string]domain.ZMLine{}
	for _, l := range current {
		next[zmKey(l)] = l
	}
	for key, l := range old {
		if _, ok := next[key]; !ok {
			l.Amount = 0
			l.EntryIDs = nil
			next[key] = l
		}
	}
	out := make([]domain.ZMLine, 0)
	for key, l := range next {
		previous := old[key]
		if accounting.TruncToEuro(previous.Amount) == accounting.TruncToEuro(l.Amount) {
			continue
		}
		l.ID, l.ZMReturnID = 0, 0
		l.PreviousAmount = previous.Amount
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool { return zmKey(out[i]) < zmKey(out[j]) })
	return out
}

func (s *ZMService) correctionPreview(ctx context.Context, fresh *domain.ZMReturn) error {
	saved, err := s.zmRepo.FindByPeriod(ctx, fresh.PeriodKey)
	if err != nil {
		return err
	}
	var latest uint
	for _, r := range saved {
		if r.Status == domain.VatReturnSubmitted && r.ID > latest {
			latest = r.ID
		}
	}
	if latest == 0 {
		return nil
	}
	fresh.IsCorrection, fresh.CorrectsID = true, &latest
	fresh.Lines = zmChanges(zmReported(saved), fresh.Lines)
	fresh.TotalServices = 0
	for _, l := range fresh.Lines {
		fresh.TotalServices += l.Amount
	}
	if len(fresh.Lines) == 0 {
		for _, r := range saved {
			if r.ID == latest {
				fresh.ID, fresh.Status, fresh.SubmittedAt, fresh.TransferTicket = r.ID, r.Status, r.SubmittedAt, r.TransferTicket
			}
		}
	}
	fresh.LateEntries = []domain.ZMLateEntry{}
	fresh.DueDate = ""
	for _, r := range saved {
		if r.IsCorrection && r.Status == domain.VatReturnDraft {
			fresh.DueDate = r.DueDate
		}
	}

	return nil
}
