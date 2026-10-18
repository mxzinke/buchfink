package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Der Lückenbericht vergleicht den Zählerstand mit den vergebenen Nummern und
// nennt zu jeder Lücke den dokumentierten Grund.
func TestInvoiceNumberGapReport(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)

	for i := 0; i < 3; i++ {
		inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
		if err := svc.Issue(ctx, inv); err != nil {
			t.Fatalf("Rechnung %d ausstellen: %v", i+1, err)
		}
	}
	report, err := svc.NumberGaps(ctx, 2026)
	if err != nil {
		t.Fatalf("Lückenbericht: %v", err)
	}
	if report.Issued != 3 || len(report.Gaps) != 0 {
		t.Fatalf("drei ausgestellte Rechnungen, keine Lücke — erhalten %+v", report)
	}

	// Ein Eingriff von außen: die zweite Rechnung verschwindet aus dem Bestand.
	// Genau das soll der Bericht sichtbar machen, denn der Zähler weiß noch,
	// dass die Nummer vergeben war.
	if err := env.db.Where("invoice_number = ?", "RE-2026-0002").
		Delete(&domain.Invoice{}).Error; err != nil {
		t.Fatalf("Rechnung entfernen: %v", err)
	}
	report, err = svc.NumberGaps(ctx, 2026)
	if err != nil {
		t.Fatalf("Lückenbericht: %v", err)
	}
	if len(report.Gaps) != 1 || report.Gaps[0].Number != "RE-2026-0002" {
		t.Fatalf("erwartet die Lücke RE-2026-0002, erhalten %+v", report.Gaps)
	}
	if report.Gaps[0].Reason != domain.NumberGapUnknown {
		t.Errorf("eine unbegründete Lücke muss als solche erscheinen, ist aber %q", report.Gaps[0].Reason)
	}

	if err := svc.RecordNumberGapReason(ctx, 2026, 2, domain.NumberGapTest, "Probelauf bei der Einrichtung"); err != nil {
		t.Fatalf("Grund dokumentieren: %v", err)
	}
	report, err = svc.NumberGaps(ctx, 2026)
	if err != nil {
		t.Fatalf("Lückenbericht: %v", err)
	}
	if len(report.Gaps) != 1 || report.Gaps[0].Reason != domain.NumberGapTest {
		t.Fatalf("der dokumentierte Grund fehlt: %+v", report.Gaps)
	}
	if report.Gaps[0].Detail != "Probelauf bei der Einrichtung" || report.Gaps[0].RecordedAt == "" {
		t.Errorf("Begründung und Zeitpunkt gehören an die Lücke: %+v", report.Gaps[0])
	}

	// Der Grund steht auch im Protokoll und nicht nur in der Tabelle. Er wird
	// nachträglich erfasst; wer wann welche Begründung eingetragen hat, ist
	// dann selbst eine Angabe (GoBD Rz. 58) — und es ist die Angabe, nach der
	// die Betriebsprüfung fragt.
	entries, err := repository.NewAuditRepository(env.db).FindAll(ctx, 200)
	if err != nil {
		t.Fatal(err)
	}
	logged := false
	for _, e := range entries {
		if e.EntityType == "NUMBER_GAP" && strings.Contains(e.Details, "RE-2026-0002") &&
			strings.Contains(e.Details, "Probelauf bei der Einrichtung") {
			logged = true
		}
	}
	if !logged {
		t.Errorf("die Begründung der Lücke RE-2026-0002 fehlt im Protokoll: %+v", entries)
	}
}

// Das Nummernformat ist einstellbar; die Rechnung folgt der Einstellung.
func TestConfiguredInvoiceNumberFormatIsUsed(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	settings := repository.NewSettingsRepository(env.db)
	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cfg.InvoiceNumberFormat = "AR-{JAHR}-{NR:5}"
	if err := settings.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	customer := env.customer(t, "Kunde GmbH", "DE", "")
	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := env.invoicesWired(t).Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}
	if inv.InvoiceNumber != "AR-2026-00001" {
		t.Errorf("Rechnungsnummer = %q, erwartet AR-2026-00001", inv.InvoiceNumber)
	}
}
