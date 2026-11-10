package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// --- Gesperrter Geschäftspartner an den Schreibwegen -----------------------

// Die Sperre wirkt nicht nur in den Auswahllisten: sie verlässt sich sonst
// darauf, dass jede Maske die gefilterte Liste verwendet, und ein Weg daneben — ein
// Eröffnungsposten, ein Belegbuchen mit bekannter Kennung — führte die Sperre
// vor.
func TestBlockedContactIsRefusedOnTheWritePaths(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	customer := env.customer(t, "Kunde GmbH", "DE", "")
	if _, _, err := env.contacts.BlockContact(ctx, customer.ID, "Löschverlangen nach Art. 17 DSGVO"); err != nil {
		t.Fatalf("Kontakt sperren: %v", err)
	}

	// Neue Ausgangsrechnung.
	inv := &domain.Invoice{
		FiscalYear: 2026, ContactID: customer.ID, Date: "2026-03-01",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-01",
		Items: []domain.InvoiceItem{{
			Position: 1, Description: "Beratung", QuantityMilli: 1000, Unit: "HUR",
			UnitPrice: 100000, TaxRate: domain.TaxRateStandard,
		}},
	}
	err := env.invoices(t).Issue(ctx, inv)
	if err == nil {
		t.Fatal("eine neue Rechnung an einen gesperrten Kontakt muss abgewiesen werden")
	}
	if !strings.Contains(err.Error(), "gesperrt") {
		t.Errorf("die Meldung muss die Sperre nennen: %v", err)
	}

	// Eröffnungsposten des Umsteigers.
	closing := fileClosingStatement(t, env)
	_, err = env.journal.PreviewOpeningBalance(ctx, openingRequest(t, env, customer.ID, closing.ID))
	if err == nil {
		t.Fatal("ein Eröffnungsposten auf einen gesperrten Kontakt muss abgewiesen werden")
	}
	if !strings.Contains(err.Error(), "gesperrt") {
		t.Errorf("die Meldung muss die Sperre nennen: %v", err)
	}
}

// --- Gesperrte Kontakte ---------------------------------------------------

func TestBlockedContactStaysOutOfSelections(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	keep := env.customer(t, "Bleibt GmbH", "DE", "")
	block := env.customer(t, "Löschverlangen GmbH", "DE", "")

	if _, _, err := env.contacts.BlockContact(ctx, block.ID, ""); err == nil {
		t.Error("zum Sperren gehört ein Grund")
	}

	blocked, answer, err := env.contacts.BlockContact(ctx, block.ID, "Löschverlangen nach Art. 17 DSGVO")
	if err != nil {
		t.Fatalf("das Sperren ist fehlgeschlagen: %v", err)
	}
	if !blocked.Blocked || blocked.BlockedAt == "" {
		t.Error("der gesperrte Kontakt muss die Sperre mit Datum haben")
	}
	if !strings.Contains(answer, "Art. 17 Abs. 3 Buchst. b DSGVO") {
		t.Errorf("die Antwort an die betroffene Person muss die Norm nennen: %q", answer)
	}

	selectable, err := env.contacts.SelectableContacts(ctx, 2026)
	if err != nil {
		t.Fatalf("die Auswahl ließ sich nicht lesen: %v", err)
	}
	for _, c := range selectable {
		if c.ID == block.ID {
			t.Error("ein gesperrter Kontakt darf in keiner Auswahl mehr stehen")
		}
	}
	var keptFound bool
	for _, c := range selectable {
		if c.ID == keep.ID {
			keptFound = true
		}
	}
	if !keptFound {
		t.Error("die übrigen Kontakte bleiben wählbar")
	}

	// In der Kontaktliste bleibt er sichtbar: die Buchungen, in denen er steht,
	// sind aufzubewahren.
	all, _ := env.contacts.GetContacts(ctx, 2026)
	var stillThere bool
	for _, c := range all {
		if c.ID == block.ID {
			stillThere = true
		}
	}
	if !stillThere {
		t.Error("ein gesperrter Kontakt wird nicht gelöscht — er bleibt in Buchungen und Exporten sichtbar")
	}
}
