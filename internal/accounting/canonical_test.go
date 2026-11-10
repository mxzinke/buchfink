package accounting

import (
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// --- Kanonische Form der Buchung ------------------------------------------

func sampleEntry() *domain.JournalEntry {
	return &domain.JournalEntry{
		EntryNumber:     "2025-000001",
		FiscalYear:      2025,
		BookingDate:     "2025-03-01",
		DocumentDate:    "2025-03-01",
		ServiceDateFrom: "2025-03-01",
		ServiceDateTo:   "2025-03-01",
		Description:     "Bürobedarf",
		Source:          domain.EntrySourceManual,
		Kind:            domain.EntryKindNormal,
		Currency:        "EUR",
		AppVersion:      "1.2.3",
		Actor:           "buchhalterin@arbeitsplatz",
		Channel:         "ui",
		CreatedAt:       time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC),
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: "6815", Amount: 10000},
			{Position: 2, Side: domain.SideCredit, Account: "1800", Amount: 10000},
		},
	}
}

// Es gibt genau eine kanonische Form: jedes Feld steht in jeder Buchung, auch
// leer oder null.
func TestCanonicalizeWritesEveryFieldForEveryEntry(t *testing.T) {
	entry := sampleEntry()
	entry.AppVersion = ""
	entry.Actor = ""
	entry.Channel = ""
	form := string(canonicalize(entry, domain.GenesisHash))

	for _, field := range []string{
		"app_version:0:\n", "actor:0:\n", "channel:0:\n", "legacy_ref:0:\n", "due_date:0:\n",
		"closing_period:1:0\n",
		"line_input_tax_share:1:0\n", "line_foreign_amount:1:0\n",
		"entertainment:1:0\n", "gifts:1:0\n",
	} {
		if !strings.Contains(form, "\n"+field) {
			t.Errorf("die kanonische Form enthält %q nicht", strings.TrimSuffix(field, "\n"))
		}
	}
}

func TestCanonicalizeCoversEveryField(t *testing.T) {
	chain := NewHashChain()
	base := chain.CalculateHash(sampleEntry(), domain.GenesisHash)

	contact := uint(7)
	changes := map[string]func(e *domain.JournalEntry){
		"Programmfassung":    func(e *domain.JournalEntry) { e.AppVersion = "9.9.9" },
		"Bearbeiter":         func(e *domain.JournalEntry) { e.Actor = "jemand.anderes@rechner" },
		"Erfassungsweg":      func(e *domain.JournalEntry) { e.Channel = "agent:claude-desktop" },
		"Herkunftskennung":   func(e *domain.JournalEntry) { e.LegacyRef = "ALT-4711" },
		"Fälligkeit":         func(e *domain.JournalEntry) { e.DueDate = "2025-04-01" },
		"Abschlussperiode":   func(e *domain.JournalEntry) { e.ClosingPeriod = true },
		"Vorsteuerschlüssel": func(e *domain.JournalEntry) { e.Lines[0].InputTaxShare = 500 },
		"Fremdbetrag":        func(e *domain.JournalEntry) { e.Lines[0].ForeignAmount = 10870 },
		"Geschenk": func(e *domain.JournalEntry) {
			e.Gifts = []domain.GiftRecord{{
				FiscalYear: 2025, Date: "2025-03-01", RecipientContactID: &contact,
				RecipientName: "Kunde GmbH", NetAmount: 3000, Account: "6610",
			}}
		},
	}
	for name, change := range changes {
		entry := sampleEntry()
		change(entry)
		if chain.CalculateHash(entry, domain.GenesisHash) == base {
			t.Errorf("%s muss in der kanonischen Form gedeckt sein", name)
		}
	}
}
