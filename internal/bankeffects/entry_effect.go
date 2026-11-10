package bankeffects

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// Transactions ist die Sicht des Effekts auf Bankumsätze.
type Transactions interface {
	FindByID(context.Context, uint) (*domain.BankTransaction, error)
	SetMatchStatus(context.Context, uint, domain.MatchStatus) error
}

// Entries liefert die noch gültigen Buchungen eines Umsatzes.
type Entries interface {
	FindActiveByBankTx(context.Context, uint) ([]domain.JournalEntry, error)
}

// EntryEffect hält Bankzuordnung und Journal in derselben Transaktion.
func EntryEffect(transactions Transactions, entries Entries) domain.EntryEffect {
	e := bankEffect{transactions, entries}
	return domain.EntryEffect{
		Name: "bank_transaction", BeforeAppend: e.claim,
		AfterPost: e.settle, AfterReverse: e.release,
	}
}

type bankEffect struct {
	transactions Transactions
	entries      Entries
}

func (s bankEffect) claim(ctx context.Context, e *domain.JournalEntry) error {
	if e.BankTxID == nil {
		return nil
	}

	tx, err := s.transactions.FindByID(ctx, *e.BankTxID)
	if err != nil {
		return fmt.Errorf("Bankumsatz %d wurde nicht gefunden: %w", *e.BankTxID, err)
	}
	if tx.MatchStatus == domain.MatchStatusIgnored {
		return fmt.Errorf(
			"Bankumsatz %d ist als nicht zu buchen markiert; eine Buchung dazu stünde neben dieser Entscheidung", tx.ID)
	}
	holders, err := s.entries.FindActiveByBankTx(ctx, tx.ID)
	if err != nil {
		return fmt.Errorf("die Buchungen zu Bankumsatz %d ließen sich nicht prüfen: %w", tx.ID, err)
	}
	if len(holders) > 0 {
		return fmt.Errorf(
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
		return fmt.Errorf(
			"die Buchung bewegt auf dem Bankkonto %s %s €, der Bankumsatz %d lautet über %s €. "+
				"Der Betrag kommt aus dem Kontoauszug und wird nicht angepasst",
			tx.LedgerAccount, moved, tx.ID, tx.Amount)
	}
	return nil
}

func (s bankEffect) settle(ctx context.Context, e *domain.JournalEntry) error {
	if e.BankTxID == nil {
		return nil
	}
	if err := s.transactions.SetMatchStatus(ctx, *e.BankTxID, domain.MatchStatusMatched); err != nil {
		return fmt.Errorf("der Bankumsatz konnte nicht als zugeordnet vermerkt werden: %w", err)
	}
	return nil
}

func (s bankEffect) release(ctx context.Context, _, reversal *domain.JournalEntry) error {
	if reversal.BankTxID == nil {
		return nil
	}
	holders, err := s.entries.FindActiveByBankTx(ctx, *reversal.BankTxID)
	if err != nil {
		return fmt.Errorf("die Buchungen zu Bankumsatz %d ließen sich nicht prüfen: %w", *reversal.BankTxID, err)
	}
	if len(holders) > 0 {
		return nil
	}
	tx, err := s.transactions.FindByID(ctx, *reversal.BankTxID)
	if err != nil || tx.MatchStatus != domain.MatchStatusMatched {
		return nil
	}
	if err := s.transactions.SetMatchStatus(ctx, tx.ID, domain.MatchStatusUnmatched); err != nil {
		return fmt.Errorf("der Bankumsatz konnte nicht wieder geöffnet werden: %w", err)
	}
	return nil
}
