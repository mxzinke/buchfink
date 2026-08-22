package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// businessVendor legt einen inländischen Lieferanten an. Unternehmer ist der
// Normalfall und damit der Nullwert — genau der Fall, in dem die
// E-Rechnungspflicht greift.
func (e *testEnv) businessVendor(t *testing.T, name string) *domain.Contact {
	t.Helper()
	c := &domain.Contact{
		Type: domain.ContactTypeVendor, Name: name, CountryCode: "DE",
	}
	if err := e.contacts.SaveContact(context.Background(), c); err != nil {
		t.Fatalf("Lieferant %s: %v", name, err)
	}
	return c
}

func noticeFor(t *testing.T, env *testEnv, req ReceiptRequest) *PostingWarning {
	t.Helper()
	preview, err := env.posting.PreviewIncomingReceipt(context.Background(), req)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	for i := range preview.Warnings {
		if preview.Warnings[i].Code == warningEInvoiceMissing {
			return &preview.Warnings[i]
		}
	}
	return nil
}

// Ein inländischer Unternehmer, der eine PDF-Rechnung stellt: der Hinweis
// erscheint, blockiert aber nichts.
func TestEInvoiceNoticeAppearsForADomesticBusinessSupplier(t *testing.T) {
	env := newTestEnv(t)
	vendor := env.businessVendor(t, "Agentur GmbH")

	req := env.receipt(t, vendor.ID, "fremdleistungen", 100000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	notice := noticeFor(t, env, req)
	if notice == nil {
		t.Fatal("für eine PDF-Rechnung eines inländischen Unternehmers muss der Hinweis erscheinen")
	}
	if notice.SupplierNote == "" {
		t.Error("zum Hinweis gehört ein Text, den man dem Lieferanten weitergeben kann")
	}

	// Und die Buchung geht trotzdem durch — der Hinweis bewertet nicht.
	if _, err := env.posting.PostIncomingReceipt(context.Background(), req); err != nil {
		t.Fatalf("der Hinweis darf die Buchung nicht blockieren: %v", err)
	}
}

// Der Text hängt am Belegdatum: bis Ende 2026 ist die sonstige Rechnung nach
// § 27 Abs. 38 Nr. 1 UStG noch zulässig, ab 2027 hängt es am Vorjahresumsatz des
// Ausstellers — den Buchfink nicht kennt und deshalb auch nicht behauptet.
func TestEInvoiceNoticeChangesWithTheDeadline(t *testing.T) {
	env := newTestEnv(t)
	vendor := env.businessVendor(t, "Agentur GmbH")

	cases := []struct {
		date         string
		wantSeverity string
		wantMention  string
	}{
		{"2026-12-31", "info", "31.12.2026"},
		{"2027-06-15", "warning", "800.000"},
		{"2028-01-02", "warning", "keine Übergangsregelung"},
	}

	for _, tc := range cases {
		t.Run(tc.date, func(t *testing.T) {
			req := env.receipt(t, vendor.ID, "fremdleistungen", 100000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
			req.DocumentDate = tc.date

			notice := noticeFor(t, env, req)
			if notice == nil {
				t.Fatal("Hinweis fehlt")
			}
			if notice.Severity != tc.wantSeverity {
				t.Errorf("Schweregrad = %q, erwartet %q", notice.Severity, tc.wantSeverity)
			}
			if !strings.Contains(notice.Detail, tc.wantMention) {
				t.Errorf("der Text soll %q nennen, lautet aber: %s", tc.wantMention, notice.Detail)
			}
		})
	}
}

// Die Fälle, in denen kein Hinweis erscheinen darf.
func TestEInvoiceNoticeStaysQuietWhereNoObligationExists(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	t.Run("strukturierter Teil liegt vor", func(t *testing.T) {
		vendor := env.businessVendor(t, "Agentur mit E-Rechnung")
		filed, err := env.receipts.File(ctx, FileReceiptRequest{
			Direction: domain.DirectionIncoming,
			Files: []NewFile{
				{Role: domain.ReceiptRoleOriginal, FileName: "rechnung.pdf", Content: []byte(minimalPDF)},
				{Role: domain.ReceiptRoleStructured, FileName: "factur-x.xml", Content: []byte(xRechnungXML), Derived: true},
			},
		})
		if err != nil {
			t.Fatalf("Hybridbeleg: %v", err)
		}
		req := env.receipt(t, vendor.ID, "fremdleistungen", 100000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
		req.ReceiptID = filed.ID
		if notice := noticeFor(t, env, req); notice != nil {
			t.Error("bei vorhandenem strukturiertem Teil darf kein Hinweis erscheinen")
		}
	})

	t.Run("ausländischer Lieferant", func(t *testing.T) {
		vendor := &domain.Contact{
			Type: domain.ContactTypeVendor, Name: "SaaS Ireland Ltd.", CountryCode: "IE",
			VatID: "IE6388047V",
		}
		if err := env.contacts.SaveContact(ctx, vendor); err != nil {
			t.Fatalf("Lieferant: %v", err)
		}
		req := env.receipt(t, vendor.ID, "software", 100000, domain.TaxRateStandard, domain.TaxTreatmentReverseCharge)
		if notice := noticeFor(t, env, req); notice != nil {
			t.Error("die Pflicht trifft nur inländische Unternehmer")
		}
	})

	t.Run("Kleinunternehmer", func(t *testing.T) {
		vendor := &domain.Contact{
			Type: domain.ContactTypeVendor, Name: "Kleinunternehmer",
			CountryCode: "DE", IsSmallBusiness: true,
		}
		if err := env.contacts.SaveContact(ctx, vendor); err != nil {
			t.Fatalf("Lieferant: %v", err)
		}
		req := env.receipt(t, vendor.ID, "fremdleistungen", 100000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
		if notice := noticeFor(t, env, req); notice != nil {
			t.Error("ein Kleinunternehmer darf nach § 34a UStDV immer eine sonstige Rechnung ausstellen")
		}
	})

	t.Run("Privatperson", func(t *testing.T) {
		vendor := &domain.Contact{
			Type: domain.ContactTypeVendor, Name: "Privatperson", CountryCode: "DE",
			IsPrivate: true,
		}
		if err := env.contacts.SaveContact(ctx, vendor); err != nil {
			t.Fatalf("Lieferant: %v", err)
		}
		req := env.receipt(t, vendor.ID, "sonstige_aufwendungen", 100000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
		if notice := noticeFor(t, env, req); notice != nil {
			t.Error("die Pflicht besteht nur zwischen Unternehmern")
		}
	})

	t.Run("Kleinbetragsrechnung", func(t *testing.T) {
		vendor := env.businessVendor(t, "Kiosk")
		// 200,00 netto plus 19 % sind 238,00 brutto — unter der Grenze des § 33 UStDV.
		req := env.receipt(t, vendor.ID, "buerobedarf", 20000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
		if notice := noticeFor(t, env, req); notice != nil {
			t.Error("eine Kleinbetragsrechnung darf immer eine sonstige Rechnung sein (§ 33 UStDV)")
		}

		// Knapp darüber schon.
		req = env.receipt(t, vendor.ID, "buerobedarf", 25000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
		if notice := noticeFor(t, env, req); notice == nil {
			t.Error("oberhalb der Kleinbetragsgrenze muss der Hinweis erscheinen")
		}
	})

	t.Run("steuerfreier Umsatz", func(t *testing.T) {
		vendor := env.businessVendor(t, "Versicherungsmakler")
		req := env.receipt(t, vendor.ID, "versicherungen", 100000, domain.TaxRateNone, domain.TaxTreatmentExempt)
		if notice := noticeFor(t, env, req); notice != nil {
			t.Error("§ 4 Nr. 8 bis 29 UStG nimmt den Umsatz aus der Pflicht heraus")
		}
	})
}
