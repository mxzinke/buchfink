package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// CorrectionResult ist das Ergebnis von „stornieren und neu buchen".
type CorrectionResult struct {
	// Original ist die stornierte Buchung, Reversal die Generalumkehr und
	// Replacement die richtige Buchung, die an ihre Stelle tritt.
	Original    *domain.JournalEntry `json:"original"`
	Reversal    *domain.JournalEntry `json:"reversal"`
	Replacement *domain.JournalEntry `json:"replacement"`
	Message     string               `json:"message"`
}

// EnsureLists ersetzt die nicht belegten Listen der drei Buchungen durch leere.
func (r *CorrectionResult) EnsureLists() {
	if r == nil {
		return
	}
	r.Original.EnsureLists()
	r.Reversal.EnsureLists()
	r.Replacement.EnsureLists()
}

// CorrectEntry storniert eine Buchung und bucht sie richtig neu.
//
// Der Vorgang, den GoBD Rz. 58 meint: die ursprüngliche Aufzeichnung bleibt
// feststellbar, die Korrektur ist als solche erkennbar, und beide sind
// miteinander verbunden. Storno, Neubuchung und Protokoll stehen in einer
// Transaktion; scheitert die Neubuchung, gibt es auch den Storno nicht.
func (s *JournalService) CorrectEntry(
	ctx context.Context, entryID uint, reason string, replacement *domain.JournalEntry,
) (*CorrectionResult, error) {
	if replacement == nil {
		return nil, fmt.Errorf("zur Korrektur gehört die richtige Buchung")
	}
	if reason == "" {
		return nil, fmt.Errorf("zum Stornieren gehört ein Grund (GoBD Rz. 58)")
	}
	return atomically(ctx, s, func(ctx context.Context) (*CorrectionResult, error) {
		return s.correctEntry(ctx, entryID, reason, replacement)
	})
}

func (s *JournalService) correctEntry(
	ctx context.Context, entryID uint, reason string, replacement *domain.JournalEntry,
) (*CorrectionResult, error) {
	original, err := s.journalRepo.FindByID(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("die zu korrigierende Buchung wurde nicht gefunden: %w", err)
	}
	if original.Kind == domain.EntryKindReversal {
		return nil, fmt.Errorf(
			"Buchung %s ist selbst eine Generalumkehr und lässt sich nicht stornieren — zu stornieren ist stattdessen die Buchung, die sie zurücknimmt",
			original.EntryNumber)
	}

	// Die Neubuchung ist eine gewöhnliche Buchung. Die Quelle „opening" gibt
	// den Steuerkonten frei und gehört allein der Eröffnungsbilanz.
	if replacement.Source == domain.EntrySourceOpening {
		return nil, fmt.Errorf(
			"eine Eröffnungsbuchung lässt sich nicht über „stornieren und neu buchen\" ersetzen. " +
				"Sie ist zu stornieren; die Eröffnungswerte werden über die Eröffnungsbilanz neu erfasst")
	}
	id := original.ID
	mode := postMode{correctsID: &id}
	if err := requireShareholderContacts(replacement.Lines); err != nil {
		return nil, fmt.Errorf("die richtige Buchung ist nicht buchbar, deshalb wurde nicht storniert: %w", err)
	}
	if err := s.validatePostable(ctx, replacement, mode); err != nil {
		return nil, fmt.Errorf("die richtige Buchung ist nicht buchbar, deshalb wurde nicht storniert: %w", err)
	}

	reversal, err := s.reverseOn(ctx, entryID, reason, "")
	if err != nil {
		return nil, err
	}
	created, err := s.post(ctx, replacement, mode)
	if err != nil {
		return nil, fmt.Errorf("die Neubuchung ist gescheitert, deshalb wurde auch nicht storniert: %w", err)
	}

	if err := s.audit(ctx, domain.AuditActionStorno, created.ID, fmt.Sprintf(
		"Buchung %s ersetzt die stornierte Buchung %s (Storno %s, Grund: %s)",
		created.EntryNumber, original.EntryNumber, reversal.EntryNumber, reason)); err != nil {
		return nil, err
	}

	result := &CorrectionResult{
		Original: original, Reversal: reversal, Replacement: created,
		Message: fmt.Sprintf(
			"Buchung %s wurde mit %s storniert und als %s neu gebucht. Alle drei bleiben im Journal sichtbar und verweisen aufeinander.",
			original.EntryNumber, reversal.EntryNumber, created.EntryNumber),
	}
	result.EnsureLists()
	return result, nil
}

// CorrectionOf liefert die Neubuchung, die eine stornierte Buchung ersetzt,
// oder nil.
//
// Die Gegenrichtung zu CorrectsEntryID. Das Journal zeigt beide: von der
// Neubuchung auf die ersetzte und von der ersetzten auf die Neubuchung — sonst
// bliebe die Frage „wurde das je richtig gebucht?" an der stornierten Buchung
// unbeantwortet.
func (s *JournalService) CorrectionOf(ctx context.Context, entryID uint) (*domain.JournalEntry, error) {
	return s.journalRepo.FindCorrectionOf(ctx, entryID)
}
