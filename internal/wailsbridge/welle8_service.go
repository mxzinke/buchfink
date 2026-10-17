package wailsbridge

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Die Bedienung der Welle 8: die laufende Buchhaltung ohne Lücken.
//
// Sie richtet sich nach einem Satz: keine Buchung ohne Beleg. Daraus folgt der
// Eigenbeleg für die Fälle ohne Fremdbeleg, der Belegverweis in beide
// Richtungen, die Klärungsliste der beanstandeten Rechnungen und der Filter,
// mit dem sich eine Buchung im Journal überhaupt wiederfinden lässt.

// --- Handbuchung mit Beleg ------------------------------------------------

// PostManualEntry bucht einen von Hand erfassten Buchungssatz mit seinem Beleg.
//
// Entweder verweist die Buchung auf einen abgelegten Beleg (ReceiptID), oder
// die Angaben eines Eigenbelegs kommen mit (SelfIssued) — dann entstehen Beleg
// und Buchung in einer Transaktion.
func (b *BuchfinkBridge) PostManualEntry(req service.ManualEntryRequest) (*domain.JournalEntry, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.JournalEntry, error) {
		return s.Posting.PostManualEntry(ctx, req)
	})
}

// --- Eigenbeleg -----------------------------------------------------------

// CreateSelfIssuedReceipt erzeugt einen Eigenbeleg mit PDF und Kopfdaten und
// legt ihn im Belegspeicher ab.
func (b *BuchfinkBridge) CreateSelfIssuedReceipt(
	req service.SelfIssuedReceiptRequest,
) (*domain.Receipt, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Receipt, error) {
		return s.Receipts.CreateSelfIssued(ctx, req)
	})
}

// --- Aufbewahrung ---------------------------------------------------------

// OverrideReceiptRetention verlängert die Aufbewahrungsfrist eines Belegs.
func (b *BuchfinkBridge) OverrideReceiptRetention(
	receiptID uint, class string, reason string,
) (*domain.Receipt, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Receipt, error) {
		return s.Receipts.OverrideRetention(
			ctx, receiptID, domain.RetentionClass(class), reason)
	})
}

// GetRetentionRules liefert die Fristentabelle mit ihrer Quelle und ihrem
// Gültigkeitsstand — für die Anzeige in den Einstellungen.
func (b *BuchfinkBridge) GetRetentionRules() accounting.RetentionRules {
	return accounting.LoadedRetentionRules()
}

// --- Beleg und Buchung in beide Richtungen --------------------------------

// GetEntriesForReceipt liefert die Buchungen zu einem Beleg.
//
// Der Weg zurück: das Journal zeigt seit jeher den Beleg, der Beleg zeigte die
// Buchung nicht. Ein Prüfer, der von der Ablage ausgeht — und das ist der
// übliche Weg —, kam damit nicht ins Journal (GOB-02 K2, BEL-01 K3).
func (b *BuchfinkBridge) GetEntriesForReceipt(receiptID uint) ([]domain.JournalEntry, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.JournalEntry, error) {
		return entriesForReceipt(ctx, s.Repos.Journal, s.FiscalYear(), receiptID)
	})
}

// entriesForReceipt sucht die Buchungen eines Belegs im aktiven Geschäftsjahr
// und im Jahr davor.
//
// Auch das Vorjahr, weil eine Buchung nicht im Geschäftsjahr des Belegs liegen
// muss: eine Dezemberrechnung wird im Januar gebucht, und ein Storno hat das
// Datum seiner Erstellung. Wer nur das aktive Jahr durchsähe, zeigte am Beleg
// „keine Buchung", während sie einen Klick weiter im Journal steht.
func entriesForReceipt(
	ctx context.Context, repo domain.JournalRepository, year int, receiptID uint,
) ([]domain.JournalEntry, error) {
	out := make([]domain.JournalEntry, 0, 2)
	years := []int{year}
	if year > 0 {
		years = append(years, year+1, year-1)
	}
	seen := map[uint]bool{}
	for _, y := range years {
		entries, err := repo.FindAll(ctx, y)
		if err != nil {
			continue
		}
		for i := range entries {
			e := &entries[i]
			if e.ReceiptID == nil || *e.ReceiptID != receiptID || seen[e.ID] {
				continue
			}
			seen[e.ID] = true
			out = append(out, *e)
		}
	}
	return out, nil
}

// GetReceiptFindings liefert die Klärungsliste eines Belegs: die Befunde nach
// Formatfehler, Geschäftsregelfehler und Inhaltsfehler getrennt, je Befund mit
// Regel, Norm und Folge für den Vorsteuerabzug.
func (b *BuchfinkBridge) GetReceiptFindings(receiptID uint) (*domain.ReceiptFindings, error) {
	return readOr(b, func() (*domain.ReceiptFindings, error) {
		empty := &domain.ReceiptFindings{}
		empty.EnsureLists()
		return empty, nil
	}, func(ctx context.Context, s *core.Services) (*domain.ReceiptFindings, error) {
		return s.Receipts.Findings(ctx, receiptID)
	})
}

// --- Eigene Konten --------------------------------------------------------

// CreateCustomAccount legt ein eigenes Konto im freien Bereich des SKR04 an.
func (b *BuchfinkBridge) CreateCustomAccount(
	req service.CustomAccountRequest,
) (*domain.Account, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Account, error) {
		return s.Accounts.CreateCustom(ctx, req)
	})
}

// SetAccountBlocked sperrt ein eigenes Konto für neue Buchungen oder gibt es
// wieder frei.
func (b *BuchfinkBridge) SetAccountBlocked(
	number string, blocked bool, reason string,
) (*domain.Account, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Account, error) {
		return s.Accounts.SetBlocked(ctx, number, blocked, reason)
	})
}

// GetCustomAccounts liefert die selbst angelegten Konten.
func (b *BuchfinkBridge) GetCustomAccounts() ([]domain.Account, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.Account, error) {
		return s.Accounts.CustomAccounts(ctx)
	})
}

// GetStatementPositions liefert die Gliederungspositionen, unter denen ein
// eigenes Konto stehen darf.
func (b *BuchfinkBridge) GetStatementPositions() ([]domain.StatementPositionOption, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.StatementPositionOption, error) {
		return s.Accounts.AvailablePositions()
	})
}

// --- Umsatzsteuer ---------------------------------------------------------

// GetVatPeriodProposal liefert den Voranmeldungszeitraum, der sich aus der
// Steuer des Vorjahres ergibt (§ 18 Abs. 2 UStG).
func (b *BuchfinkBridge) GetVatPeriodProposal(year int) (*service.VatPeriodProposal, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*service.VatPeriodProposal, error) {
		return s.VatReturns.SuggestPeriodType(ctx, year)
	})
}

// --- Journalfilter --------------------------------------------------------

// GetFilteredJournal liefert die gefilterte Menge der Journalzeilen mit ihrer
// Summenzeile (PRF-01 K3).
func (b *BuchfinkBridge) GetFilteredJournal(
	filter accounting.JournalFilter,
) (*accounting.JournalFilterResult, error) {
	return readOr(b, func() (*accounting.JournalFilterResult, error) {
		empty := &accounting.JournalFilterResult{Rows: []accounting.JournalFilterRow{}}
		return empty, nil
	}, func(ctx context.Context, s *core.Services) (*accounting.JournalFilterResult, error) {
		return s.Accounting.FilterEntries(ctx, filter)
	})
}

// SaveFilteredJournalCSV schreibt die gefilterte Menge in eine gewählte Datei.
func (b *BuchfinkBridge) SaveFilteredJournalCSV(
	filter accounting.JournalFilter, path string,
) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("ohne Zielpfad lässt sich die Datei nicht schreiben")
	}
	// Ein Lesevorgang: die Herausgabe protokolliert sich (QUE-02 K2) und ist
	// im Prüfermodus zulässig, ihr Zweck ist die Ausgabe von Daten.
	csv, err := read(b, func(ctx context.Context, s *core.Services) (string, error) {
		return s.Accounting.FilterEntriesCSV(ctx, filter)
	})
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(csv), 0o600); err != nil {
		return "", fmt.Errorf("die Datei konnte nicht geschrieben werden: %w", err)
	}
	return path, nil
}
