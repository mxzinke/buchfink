package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func manualTaxEntry(tax domain.Cents, key, account string) *domain.JournalEntry {
	entry := simpleEntry("6815", "1800", 10000+tax)
	entry.TaxTreatment = domain.TaxTreatmentDomestic
	entry.Lines = []domain.JournalLine{
		{Side: domain.SideDebit, Account: "6815", Amount: 10000},
		{Side: domain.SideDebit, Account: account, Amount: tax, TaxKey: key, TaxBase: 10000},
		{Side: domain.SideCredit, Account: "1800", Amount: 10000 + tax},
	}
	return entry
}

func TestPostChecksTaxKeysAgainstTheCatalog(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	if _, err := env.journal.Post(ctx, manualTaxEntry(700, "VST19", domain.AccountVorsteuer19)); err == nil ||
		!strings.Contains(err.Error(), "passen nicht") {
		t.Errorf("7 € Vorsteuer auf 100 € zum Satz 19 %% muss abgewiesen werden, Fehler: %v", err)
	}
	if _, err := env.journal.Post(ctx, manualTaxEntry(1900, "VST16", domain.AccountVorsteuer19)); err == nil ||
		!strings.Contains(err.Error(), "unbekannter Steuerschlüssel") {
		t.Errorf("ein unbekannter Schlüssel muss abgewiesen werden, Fehler: %v", err)
	}
	if _, err := env.journal.Post(ctx, manualTaxEntry(700, "VST7", domain.AccountVorsteuer19)); err == nil ||
		!strings.Contains(err.Error(), "gehört auf Konto 1401") {
		t.Errorf("VST7 auf dem 19-%%-Konto muss abgewiesen werden, Fehler: %v", err)
	}
	created, err := env.journal.Post(ctx, manualTaxEntry(1900, "VST19", domain.AccountVorsteuer19))
	if err != nil {
		t.Fatalf("die richtige Buchung muss durchgehen: %v", err)
	}
	if _, err := env.journal.Reverse(ctx, created.ID, "Test"); err != nil {
		t.Errorf("der Storno einer Steuerbuchung muss durchgehen: %v", err)
	}
}
