package service

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
)

func (s *JournalService) AddEntryEffects(effects ...domain.EntryEffect) {
	for _, effect := range effects {
		if effect.Name == "" || (effect.BeforeAppend == nil && effect.AfterPost == nil && effect.AfterReverse == nil) {
			panic("ungültiger Buchungseffekt: " + effect.Name)
		}
		for _, existing := range s.entryEffects {
			if existing.Name == effect.Name {
				panic("doppelter Buchungseffekt: " + effect.Name)
			}
		}
		s.entryEffects = append(s.entryEffects, effect)
	}
}

func (s *JournalService) EntryEffectNames() []string {
	names := make([]string, 0, len(s.entryEffects))
	for _, effect := range s.entryEffects {
		names = append(names, effect.Name)
	}
	return names
}

func (s *JournalService) beforeAppend(ctx context.Context, entry *domain.JournalEntry, mode postMode) error {
	if mode.reversal {
		return nil
	}
	for _, effect := range s.entryEffects {
		if effect.BeforeAppend != nil {
			if err := effect.BeforeAppend(ctx, entry); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *JournalService) afterAppend(ctx context.Context, entry *domain.JournalEntry, mode postMode) error {
	for _, effect := range s.entryEffects {
		if mode.reversal {
			if effect.AfterReverse != nil {
				if err := effect.AfterReverse(ctx, mode.original, entry); err != nil {
					return err
				}
			}
		} else if effect.AfterPost != nil {
			if err := effect.AfterPost(ctx, entry); err != nil {
				return err
			}
		}
	}
	return nil
}
