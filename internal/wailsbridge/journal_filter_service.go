package wailsbridge

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/core"
)

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
