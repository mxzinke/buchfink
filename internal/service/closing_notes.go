package service

import (
	"context"
	"fmt"
	"github.com/buchfink/buchfink/internal/domain"
	"strings"
)

// ClosingNotes lesen und übernehmen die Anhangtexte eines Geschäftsjahres.
type ClosingNotes interface {
	NotesTextSource
	NotesCopier
}

type notesStore struct {
	auditor[int]
	notesRepo  domain.NotesTextRepository
	journalSvc *JournalService
}

// NotesTexts liefert alle Abschnitte eines Jahres, auch die leeren: der Anhang
// ist eine Gliederung, und ein Abschnitt, der nur erscheint, wenn er gefüllt
// ist, wird nie gefüllt.
func (s *notesStore) NotesTexts(ctx context.Context, year int) ([]NotesTextView, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	stored, err := s.notesRepo.FindByYear(ctx, year)
	if err != nil {
		return nil, err
	}
	bySection := map[domain.NotesSection]string{}
	for _, text := range stored {
		bySection[text.Section] = text.Text
	}
	out := make([]NotesTextView, 0, len(domain.AllNotesSections()))
	for _, def := range domain.AllNotesSections() {
		out = append(out, NotesTextView{NotesSectionDefinition: def, Text: bySection[def.Section]})
	}
	return out, nil
}

// CopyNotesInto übernimmt die Anhangtexte des Vorjahres als Vorlage.
//
// Die Bilanzierungs- und Bewertungsmethoden ändern sich selten, und ein leerer
// Anhang im neuen Jahr führt dazu, dass die Angaben schlicht fehlen. Kopiert
// wird nur, was noch nicht dasteht: eine bereits geschriebene Fassung darf eine
// Vorlage nicht überschreiben.
func (s *notesStore) CopyNotesInto(ctx context.Context, toYear int) (int, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (int, error) {
		return s.copyNotesInto(ctx, toYear)
	})
}

func (s *notesStore) copyNotesInto(ctx context.Context, toYear int) (int, error) {
	if toYear <= 1 {
		return 0, nil
	}
	source, err := s.notesRepo.FindByYear(ctx, toYear-1)
	if err != nil || len(source) == 0 {
		return 0, err
	}
	target, err := s.notesRepo.FindByYear(ctx, toYear)
	if err != nil {
		return 0, err
	}
	present := map[domain.NotesSection]bool{}
	for _, text := range target {
		if strings.TrimSpace(text.Text) != "" {
			present[text.Section] = true
		}
	}
	copied := 0
	for _, text := range source {
		if present[text.Section] || strings.TrimSpace(text.Text) == "" {
			continue
		}
		entry := &domain.NotesText{Year: toYear, Section: text.Section, Text: text.Text}
		if err := s.notesRepo.Save(ctx, entry); err != nil {
			return copied, err
		}
		copied++
	}
	if copied > 0 {
		if err := s.auditEntity(ctx, domain.AuditActionUpdate, "NOTES", toYear, fmt.Sprintf(
			"%d Anhangtexte aus dem Geschäftsjahr %d als Vorlage übernommen", copied, toYear-1)); err != nil {
			return 0, err
		}
	}
	return copied, nil
}
