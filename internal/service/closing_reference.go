package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// standingClosingEntries sind die Abschlussbuchungen eines Jahres mit dieser
// Belegnummer, die nicht storniert sind. Die Bausteine ohne eigene Kartei
// erkennen an ihr, ob sie gebucht sind; der Storno übernimmt die Belegnummer,
// zählt aber nicht.
func standingClosingEntries(
	ctx context.Context, repo domain.JournalRepository, year int, reference string,
) ([]domain.JournalEntry, error) {
	return standingEntries(ctx, repo, year, domain.EntrySourceClosing, reference)
}

// standingOpeningEntries sind die nicht stornierten Auflösungen eines
// Abschlussbausteins am Beginn des Jahres.
func standingOpeningEntries(
	ctx context.Context, repo domain.JournalRepository, year int, reference string,
) ([]domain.JournalEntry, error) {
	return standingEntries(ctx, repo, year, domain.EntrySourceOpening, reference)
}

func standingEntries(
	ctx context.Context, repo domain.JournalRepository, year int, source domain.EntrySource, reference string,
) ([]domain.JournalEntry, error) {
	entries, err := repo.FindAll(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("die Buchungen des Geschäftsjahres %d konnten nicht gelesen werden: %w", year, err)
	}
	index := newReversalIndex(repo)
	out := make([]domain.JournalEntry, 0, 1)
	for i := range entries {
		e := &entries[i]
		if e.Source != source || e.DocumentNumber != reference || e.Kind == domain.EntryKindReversal {
			continue
		}
		voided, err := index.reversed(ctx, &e.ID)
		if err != nil {
			return nil, err
		}
		if !voided {
			out = append(out, *e)
		}
	}
	return out, nil
}

// refuseSecondClosingEntry weist einen zweiten Lauf eines Bausteins ab, dessen
// Buchung noch steht. Ändern heißt stornieren und neu buchen (§ 239 Abs. 3 HGB).
func refuseSecondClosingEntry(
	ctx context.Context, repo domain.JournalRepository, year int, reference, what string,
) error {
	standing, err := standingClosingEntries(ctx, repo, year, reference)
	if err != nil {
		return err
	}
	if len(standing) > 0 {
		return fmt.Errorf(
			"%s %d ist mit der Buchung %s bereits gebucht. Sie ist zuerst zu stornieren; ein zweiter Lauf "+
				"buchte sie ein zweites Mal", what, year, standing[0].EntryNumber)
	}
	return nil
}
