package accounting

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

// docAccountPattern matches the account numbers the concept document states as
// part of a Buchungssatz: bold ("**5906**") or as a mapping target ("→ 5906").
var docAccountPattern = regexp.MustCompile(`\*\*(\d{4})\*\*|→\s+(\d{4})\b`)

// TestConceptDocumentsUseRealSKR04Accounts prüft die Kontonummern in allen
// Markdown-Dateien unter docs/, einschließlich der Unterverzeichnisse.
func TestConceptDocumentsUseRealSKR04Accounts(t *testing.T) {
	var paths []string
	err := filepath.WalkDir("../../docs", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && filepath.Ext(path) == ".md" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil || len(paths) == 0 {
		t.Fatalf("Dokumentation nicht vollständig lesbar oder leer: %v", err)
	}

	chart := chartForTest(t)
	total := 0

	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s ist nicht lesbar: %v", filepath.Base(path), err)
			continue
		}

		seen := map[string]bool{}
		for _, match := range docAccountPattern.FindAllStringSubmatch(string(content), -1) {
			number := match[1]
			if number == "" {
				number = match[2]
			}
			seen[number] = true
		}

		numbers := make([]string, 0, len(seen))
		for n := range seen {
			numbers = append(numbers, n)
		}
		sort.Strings(numbers)
		total += len(numbers)

		for _, number := range numbers {
			if err := chart.EnsurePostable(number); err != nil {
				t.Errorf("%s nennt Konto %s: %v", filepath.Base(path), number, err)
			}
		}
	}

	// Guard against a search pattern that silently stops matching.
	if total < 50 {
		t.Fatalf("über alle Dokumente wurden nur %d Kontonummern gefunden – prüfe das Suchmuster", total)
	}
}
