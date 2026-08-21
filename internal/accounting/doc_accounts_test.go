package accounting

import (
	"os"
	"regexp"
	"sort"
	"testing"
)

// docAccountPattern matches the account numbers the concept document states as
// part of a Buchungssatz: bold ("**5906**") or as a mapping target ("→ 5906").
var docAccountPattern = regexp.MustCompile(`\*\*(\d{4})\*\*|→\s+(\d{4})\b`)

// TestConceptDocumentUsesRealSKR04Accounts checks every account number the
// concept document names against the DATEV catalog.
//
// The document previously carried SKR03 numbers throughout, and nothing caught
// it: the numbers look plausible, and several of them exist in SKR04 with a
// completely different meaning. Prose is not compiled, so this test compiles it.
func TestConceptDocumentUsesRealSKR04Accounts(t *testing.T) {
	const docPath = "../../docs/anforderung-beleg-buchungsflow.md"

	content, err := os.ReadFile(docPath)
	if err != nil {
		t.Skipf("Konzeptdokument nicht lesbar: %v", err)
	}

	chart := chartForTest(t)

	seen := map[string]bool{}
	for _, match := range docAccountPattern.FindAllStringSubmatch(string(content), -1) {
		number := match[1]
		if number == "" {
			number = match[2]
		}
		seen[number] = true
	}

	if len(seen) < 30 {
		t.Fatalf("im Konzeptdokument wurden nur %d Kontonummern gefunden – prüfe das Suchmuster", len(seen))
	}

	numbers := make([]string, 0, len(seen))
	for n := range seen {
		numbers = append(numbers, n)
	}
	sort.Strings(numbers)

	for _, number := range numbers {
		if err := chart.EnsurePostable(number); err != nil {
			t.Errorf("Konzeptdokument nennt Konto %s: %v", number, err)
		}
	}
}
