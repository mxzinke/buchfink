package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
)

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
