package accounting

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

// docAccountPattern matches the account numbers the concept document states as
// part of a Buchungssatz: bold ("**5906**") or as a mapping target ("→ 5906").
var docAccountPattern = regexp.MustCompile(`\*\*(\d{4})\*\*|→\s+(\d{4})\b`)

// TestConceptDocumentsUseRealSKR04Accounts checks every account number the
// requirement documents name against the DATEV catalog.
//
// The main concept previously carried SKR03 numbers throughout, and nothing
// caught it: the numbers look plausible, and several of them exist in SKR04 with
// a completely different meaning. Prose is not compiled, so this test compiles
// it — for every document under docs/, so a new one is covered from the start.
func TestConceptDocumentsUseRealSKR04Accounts(t *testing.T) {
	paths, err := filepath.Glob("../../docs/*.md")
	if err != nil || len(paths) == 0 {
		t.Skipf("keine Dokumente gefunden: %v", err)
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
