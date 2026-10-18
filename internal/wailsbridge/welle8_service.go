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

// Die laufende Buchhaltung in der Oberfläche: Belegverweis in beide Richtungen,
// Aufbewahrung und der Filter, mit dem sich eine Buchung im Journal
// wiederfinden lässt. Gebucht wird hier nicht; das tut der Agent über den Kern.

// --- Aufbewahrung ---------------------------------------------------------

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
		return entriesForReceipt(ctx, s.Repos.Journal, b.viewYear(), receiptID)
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
	ctx context.Context, repo core.JournalReader, year int, receiptID uint,
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

// --- Umsatzsteuer ---------------------------------------------------------

// GetVatPeriodProposal liefert den Voranmeldungszeitraum, der sich aus der
// Steuer des Vorjahres ergibt (§ 18 Abs. 2 UStG).
func (b *BuchfinkBridge) GetVatPeriodProposal(year int) (*service.VatPeriodProposal, error) {
	year = b.yearOr(year)
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
