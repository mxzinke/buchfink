package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/invoice"
	"github.com/buchfink/buchfink/internal/repository"
)

// Gegenüber einem Verbraucher tritt der Verzug nach dreißig Tagen nur ein, wenn
// die Rechnung darauf hingewiesen hat (§ 286 Abs. 3 Satz 1 Halbsatz 2 BGB).
//
// Ohne den Hinweis gibt es keinen Verzug von selbst, also weder Zinsen noch die
// Pauschale — er entsteht erst mit einer Mahnung (§ 286 Abs. 1 BGB). Für die
// Rechnungen aus der Zeit vor dem Hinweis steht das Kennzeichen deshalb an der
// Rechnung und nicht in einer Einstellung.
func TestDunningNeedsTheConsumerNoticeOnTheInvoice(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	invoiceRepo := repository.NewInvoiceRepository(env.db)

	customer := env.customer(t, "Erika Mustermann", "DE", "")
	customer.IsPrivate = true
	if err := env.contacts.SaveContact(ctx, customer); err != nil {
		t.Fatalf("Kunde speichern: %v", err)
	}
	env.openReceivable(t, customer, 1_000_000, "2026-01-17", "2026-01-31", "RE-2026-0100")

	// Eine Bestandsrechnung aus der Zeit vor dem Hinweis: derselbe Nummernkreis,
	// aber ohne das Kennzeichen.
	if err := invoiceRepo.Save(ctx, &domain.Invoice{
		InvoiceNumber: "RE-2026-0100", ContactID: customer.ID, FiscalYear: 2026,
		Date: "2026-01-17", ServiceDateFrom: "2026-01-17", ServiceDateTo: "2026-01-17",
		Status: domain.InvoiceStatusIssued, TaxTreatment: domain.TaxTreatmentDomestic,
		GrossAmount: 1_000_000, ConsumerNoticePrinted: false,
	}); err != nil {
		t.Fatalf("Bestandsrechnung: %v", err)
	}

	svc := env.dunning(t, nil)
	proposals, err := svc.Proposals(ctx, "2026-04-17")
	if err != nil {
		t.Fatalf("Mahnvorschläge: %v", err)
	}
	if len(proposals) != 1 || len(proposals[0].Items) != 1 {
		t.Fatalf("erwartet einen Vorschlag mit einem Posten: %+v", proposals)
	}
	item := proposals[0].Items[0]
	if item.Interest != 0 || item.DefaultFrom != "" {
		t.Errorf("ohne den Hinweis läuft kein Verzug: Zinsen %s €, Verzugsbeginn %q",
			item.Interest, item.DefaultFrom)
	}
	if !strings.Contains(item.Note, "286") {
		t.Errorf("der Posten muss den Grund nennen: %q", item.Note)
	}

	// Dieselbe Rechnung mit Hinweis: der Verzug tritt nach dreißig Tagen ein.
	stored, err := invoiceRepo.FindByNumber(ctx, "RE-2026-0100")
	if err != nil {
		t.Fatalf("Rechnung lesen: %v", err)
	}
	stored.ConsumerNoticePrinted = true
	if err := invoiceRepo.Save(ctx, stored); err != nil {
		t.Fatalf("Kennzeichen setzen: %v", err)
	}

	proposals, err = svc.Proposals(ctx, "2026-04-17")
	if err != nil {
		t.Fatalf("Mahnvorschläge: %v", err)
	}
	item = proposals[0].Items[0]
	if item.DefaultFrom != "2026-03-03" {
		t.Errorf("Verzugsbeginn = %q, erwartet 2026-03-03", item.DefaultFrom)
	}
	if item.Interest != 7730 {
		t.Errorf("Zinsen = %s €, erwartet 77,30 (fünf Prozentpunkte)", item.Interest)
	}
}

// Die Rechnung an einen Verbraucher hat den Hinweis, und das Kennzeichen ist
// danach an ihr gesetzt.
//
// Beides gehört zusammen: der Satz auf dem Dokument ist die Voraussetzung des
// Verzugseintritts, und das Kennzeichen ist das, woran das Mahnwesen ihn später
// erkennt. Stünde nur eines von beidem, forderte Buchfink entweder Zinsen ohne
// Hinweis oder ließe sie trotz Hinweis aus.
func TestConsumerInvoiceCarriesTheDefaultNotice(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung ist zu langsam für -short")
	}
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.invoicesWiredWithDocuments(t)

	consumer := env.customer(t, "Erika Mustermann", "DE", "")
	consumer.IsPrivate = true
	if err := env.contacts.SaveContact(ctx, consumer); err != nil {
		t.Fatalf("Kunde speichern: %v", err)
	}
	business := env.customer(t, "Kunde GmbH", "DE", "")

	toConsumer := env.simpleInvoice(consumer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, toConsumer); err != nil {
		t.Fatalf("Rechnung an den Verbraucher: %v", err)
	}
	if !toConsumer.ConsumerNoticePrinted {
		t.Error("die Rechnung an einen Verbraucher muss den Hinweis vermerken")
	}

	toBusiness := env.simpleInvoice(business.ID, "2026-03-02", 100000)
	if err := svc.Issue(ctx, toBusiness); err != nil {
		t.Fatalf("Rechnung an das Unternehmen: %v", err)
	}
	if toBusiness.ConsumerNoticePrinted {
		t.Error("gegenüber einem Unternehmer braucht es den Hinweis nicht")
	}

	// Und der Satz steht auf dem Dokument.
	seller, err := repository.NewSettingsRepository(env.db).GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Unternehmensdaten: %v", err)
	}
	consumerDoc := invoice.GenerateTypstTemplate(toConsumer, seller, consumer)
	if !strings.Contains(consumerDoc, "286") {
		t.Error("die Rechnung an einen Verbraucher trägt den Verzugshinweis nicht")
	}
	businessDoc := invoice.GenerateTypstTemplate(toBusiness, seller, business)
	if strings.Contains(businessDoc, "286 Abs. 3") {
		t.Error("gegenüber einem Unternehmer gehört der Hinweis nicht auf die Rechnung")
	}
}
