package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/domain"
)

// SetDunning hängt die Mahnvermerke und das Protokoll an.
func (s *OverdueService) SetDunning(repo domain.DunningRepository, audit domain.AuditRepository) {
	s.dunning, s.auditRepo = repo, audit
}

func (s *OverdueService) reminders(ctx context.Context) (map[uint][]domain.DunningNote, error) {
	out := map[uint][]domain.DunningNote{}
	if s.dunning == nil {
		return out, nil
	}
	notes, err := s.dunning.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("die Mahnvermerke ließen sich nicht lesen: %w", err)
	}
	for _, n := range notes {
		out[n.EntryID] = append(out[n.EntryID], n)
	}
	return out, nil
}

// reminderOf liefert die letzte Mahnung eines Postens und den ersten
// Verzugstag aus der frühesten Mahnung ab Fälligkeit (leer ohne eine
// solche). Die Vermerke kommen nach Datum geordnet.
func reminderOf(notes []domain.DunningNote, dueDate string) (*domain.DunningNote, string) {
	if len(notes) == 0 {
		return nil, ""
	}
	last := &notes[len(notes)-1]
	for _, n := range notes {
		if dueDate == "" || n.Date < dueDate {
			continue
		}
		day, err := time.Parse("2006-01-02", n.Date)
		if err != nil {
			continue
		}
		return last, day.AddDate(0, 0, 1).Format("2006-01-02")
	}
	return last, ""
}

// ReminderRequest ist ein Mahnvermerk zu einem offenen Posten.
type ReminderRequest struct {
	EntryID uint   `json:"entryId"`
	Date    string `json:"date"`
	Level   int    `json:"level,omitempty"`
	Note    string `json:"note,omitempty"`
}

// RecordReminder vermerkt, dass ein offener Posten gemahnt wurde. Buchfink
// schreibt die Mahnung nicht; es hält fest, wann sie zugegangen ist, und
// rechnet den Verzug danach.
func (s *OverdueService) RecordReminder(ctx context.Context, req ReminderRequest) (*domain.DunningNote, error) {
	if s.dunning == nil || s.openItems == nil {
		return nil, fmt.Errorf("Mahnvermerke sind nicht eingerichtet")
	}
	if err := domain.CheckDate("Zugang der Mahnung", req.Date); err != nil {
		return nil, err
	}
	if req.Date > s.today() {
		return nil, fmt.Errorf("die Mahnung vom %s liegt in der Zukunft; vermerkt wird eine zugegangene Mahnung",
			domain.GermanDate(req.Date))
	}
	items, err := s.openItems.OpenItems(ctx)
	if err != nil {
		return nil, err
	}
	var item *domain.OpenItem
	for i := range items {
		if items[i].EntryID == req.EntryID {
			item = &items[i]
			break
		}
	}
	if item == nil || item.OpenAmount <= 0 {
		return nil, fmt.Errorf("zur Buchung %d gibt es keinen offenen Posten", req.EntryID)
	}
	if item.ContactType != domain.ContactTypeCustomer {
		return nil, fmt.Errorf("%s ist eine Verbindlichkeit; gemahnt werden Forderungen", documentOf(item))
	}
	if item.DueDate != "" && req.Date < item.DueDate {
		return nil, fmt.Errorf("%s ist erst am %s fällig; eine Mahnung vor Fälligkeit setzt nicht in Verzug",
			documentOf(item), domain.GermanDate(item.DueDate))
	}
	level := req.Level
	if level <= 0 {
		level = 1
	}
	note := &domain.DunningNote{
		EntryID: req.EntryID, Date: req.Date, Level: level, Note: strings.TrimSpace(req.Note),
		Channel: string(actor.RecordedChannel(ctx)), Actor: actor.Actor(),
	}
	if err := s.dunning.Create(ctx, note); err != nil {
		return nil, fmt.Errorf("der Mahnvermerk ließ sich nicht speichern: %w", err)
	}
	if err := logAudit(ctx, s.auditRepo, domain.AuditActionCreate, "DUNNING_NOTE", fmt.Sprintf("%d", note.ID),
		fmt.Sprintf("%d. Mahnung zu %s (%s) vermerkt, zugegangen am %s",
			level, documentOf(item), item.ContactName, domain.GermanDate(req.Date))); err != nil {
		return nil, err
	}
	return note, nil
}
