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

// CorrectEntry storniert eine Buchung und bucht sie richtig neu.
//
// Der Vorgang, den GoBD Rz. 58 meint: die ursprüngliche Aufzeichnung bleibt
// feststellbar, die Korrektur ist als solche erkennbar, und beide sind
// miteinander verbunden. Bisher waren das zwei getrennte Handgriffe — Storno
// hier, neue Buchung dort —, und wer im Journal auf die Neubuchung sah, konnte
// nicht erkennen, dass sie eine falsche ersetzt. Die Verbindung fehlte in der
// einen Richtung ganz.
//
// Der Ablauf steht in dieser Reihenfolge, weil eine halb ausgeführte Korrektur
// schlimmer ist als keine: erst wird die Neubuchung geprüft, ohne sie zu
// schreiben, dann storniert, dann neu gebucht. Scheiterte die Neubuchung nach
// dem Storno, stünde der Vorgang ohne Buchung da — die alte zurückgenommen, die
// richtige nicht erfasst.
func (s *JournalService) CorrectEntry(
	ctx context.Context, entryID uint, reason string, replacement *domain.JournalEntry,
) (*CorrectionResult, error) {
	if replacement == nil {
		return nil, fmt.Errorf("zur Korrektur gehört die richtige Buchung")
	}
	if reason == "" {
		return nil, fmt.Errorf("zum Stornieren gehört ein Grund (GoBD Rz. 58)")
	}

	original, err := s.journalRepo.FindByID(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("die zu korrigierende Buchung wurde nicht gefunden: %w", err)
	}
	if original.Kind == domain.EntryKindReversal {
		return nil, fmt.Errorf(
			"Buchung %s ist selbst eine Generalumkehr und lässt sich nicht stornieren — storniere stattdessen die Buchung, die sie zurücknimmt",
			original.EntryNumber)
	}

	// Der Verweis steht vor der Prüfung, damit die Prüfung die Buchung sieht,
	// die tatsächlich geschrieben wird.
	id := original.ID
	replacement.CorrectsEntryID = &id
	if err := s.ValidatePostable(ctx, replacement); err != nil {
		return nil, fmt.Errorf("die richtige Buchung ist nicht buchbar, deshalb wurde nicht storniert: %w", err)
	}

	reversal, err := s.Reverse(ctx, entryID, reason)
	if err != nil {
		return nil, err
	}

	created, err := s.Post(ctx, replacement)
	if err != nil {
		// Der Storno steht bereits und bleibt stehen: eine zurückgenommene
		// Buchung wieder aufleben zu lassen ginge nur durch eine weitere
		// Buchung, und die wäre eine Behauptung über einen Vorgang, den es nicht
		// gab. Der Fehler nennt deshalb den Stand, in dem die Buchhaltung ist.
		return nil, fmt.Errorf(
			"Buchung %s wurde storniert (%s), die Neubuchung ist aber gescheitert: %w. Erfasse die richtige Buchung von Hand",
			original.EntryNumber, reversal.EntryNumber, err)
	}

	s.audit(ctx, domain.AuditActionStorno, created.ID, fmt.Sprintf(
		"Buchung %s ersetzt die stornierte Buchung %s (Storno %s, Grund: %s)",
		created.EntryNumber, original.EntryNumber, reversal.EntryNumber, reason))

	return &CorrectionResult{
		Original: original, Reversal: reversal, Replacement: created,
		Message: fmt.Sprintf(
			"Buchung %s wurde mit %s storniert und als %s neu gebucht. Alle drei bleiben im Journal sichtbar und verweisen aufeinander.",
			original.EntryNumber, reversal.EntryNumber, created.EntryNumber),
	}, nil
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
