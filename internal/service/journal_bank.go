package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// Ein Bankumsatz steht in genau einer gültigen Buchung (§ 146 Abs. 1 AO, GoBD
// Rz. 36). Gleich über welchen Weg eine Buchung mit Bankbezug entsteht —
// Zuordnung, Aufteilung, Handbuchung oder Neubuchung nach Storno —, hier wird
// der Umsatz geprüft und als zugeordnet vermerkt. Die Generalumkehr gibt ihn
// wieder frei, wenn keine andere Buchung ihn mehr trägt.

// SetBankRepo hängt die Bankumsätze an. Ohne sie bucht der Dienst Buchungen
// mit Bankbezug ungeprüft.
func (s *JournalService) SetBankRepo(r domain.BankRepository) { s.bankRepo = r }

// claimBankTx prüft vor dem Schreiben, ob die Buchung ihren Bankumsatz buchen
// darf: er existiert, ist nicht als „nicht zu buchen" markiert, steht in keiner
// anderen gültigen Buchung, und die Buchung bewegt auf seinem Konto genau
// seinen Betrag.
func (s *JournalService) claimBankTx(ctx context.Context, e *domain.JournalEntry) (*domain.BankTransaction, error) {
	if s.bankRepo == nil || e.BankTxID == nil || e.Kind == domain.EntryKindReversal {
		return nil, nil
	}
	tx, err := s.bankRepo.FindByID(ctx, *e.BankTxID)
	if err != nil {
		return nil, fmt.Errorf("Bankumsatz %d wurde nicht gefunden: %w", *e.BankTxID, err)
	}
	if tx.MatchStatus == domain.MatchStatusIgnored {
		return nil, fmt.Errorf(
			"Bankumsatz %d ist als nicht zu buchen markiert; eine Buchung dazu stünde neben dieser Entscheidung", tx.ID)
	}
	holders, err := s.journalRepo.FindActiveByBankTx(ctx, tx.ID)
	if err != nil {
		return nil, fmt.Errorf("die Buchungen zu Bankumsatz %d ließen sich nicht prüfen: %w", tx.ID, err)
	}
	if len(holders) > 0 {
		return nil, fmt.Errorf(
			"Bankumsatz %d vom %s über %s € ist bereits mit Buchung %s gebucht. Ein Umsatz wird genau einmal "+
				"gebucht; eine falsche Buchung wird storniert und neu gebucht",
			tx.ID, domain.GermanDate(tx.BookingDate), tx.Amount, holders[0].EntryNumber)
	}
	var moved domain.Cents
	for _, l := range e.Lines {
		if l.Account != tx.LedgerAccount {
			continue
		}
		if l.Side == domain.SideDebit {
			moved += l.Amount
		} else {
			moved -= l.Amount
		}
	}
	if moved != tx.Amount {
		return nil, fmt.Errorf(
			"die Buchung bewegt auf dem Bankkonto %s %s €, der Bankumsatz %d lautet über %s €. "+
				"Der Betrag kommt aus dem Kontoauszug und wird nicht angepasst",
			tx.LedgerAccount, moved, tx.ID, tx.Amount)
	}
	return tx, nil
}

// settleBankTx vermerkt nach dem Schreiben, ob der Umsatz gebucht ist: die
// neue Buchung trägt ihn, eine Generalumkehr gibt ihn frei, sofern ihn keine
// andere gültige Buchung mehr trägt.
func (s *JournalService) settleBankTx(ctx context.Context, e *domain.JournalEntry, claimed *domain.BankTransaction) error {
	if s.bankRepo == nil || e.BankTxID == nil {
		return nil
	}
	if e.Kind != domain.EntryKindReversal {
		if claimed == nil {
			return nil
		}
		if err := s.bankRepo.SetMatchStatus(ctx, claimed.ID, domain.MatchStatusMatched); err != nil {
			return fmt.Errorf("der Bankumsatz konnte nicht als zugeordnet vermerkt werden: %w", err)
		}
		return nil
	}
	holders, err := s.journalRepo.FindActiveByBankTx(ctx, *e.BankTxID)
	if err != nil {
		return fmt.Errorf("die Buchungen zu Bankumsatz %d ließen sich nicht prüfen: %w", *e.BankTxID, err)
	}
	if len(holders) > 0 {
		return nil
	}
	tx, err := s.bankRepo.FindByID(ctx, *e.BankTxID)
	if err != nil || tx.MatchStatus != domain.MatchStatusMatched {
		return nil
	}
	if err := s.bankRepo.SetMatchStatus(ctx, tx.ID, domain.MatchStatusUnmatched); err != nil {
		return fmt.Errorf("der Bankumsatz konnte nicht wieder geöffnet werden: %w", err)
	}
	return nil
}
