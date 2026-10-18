package policy

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/receiptstore"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/service"
)

func TestGoldenReceiptSpecials(t *testing.T) {
	cases := []struct {
		name   string
		change func(*service.ReceiptRequest)
	}{
		{"routine", func(r *service.ReceiptRequest) {}},
		{"konto", func(r *service.ReceiptRequest) { r.Positions[0].Account = "6815" }},
		{"anteil", func(r *service.ReceiptRequest) {
			r.Positions[0].InputTaxShare = 600
			r.Positions[0].InputTaxShareReason = "Flächenmaßstab"
		}},
		{"anteil_null", func(r *service.ReceiptRequest) { r.Positions[0].InputTaxShare = 0 }},
		{"anteil_voll", func(r *service.ReceiptRequest) { r.Positions[0].InputTaxShare = 1000 }},
		{"geschenk", func(r *service.ReceiptRequest) {
			r.Positions[0].Gift = &service.GiftInput{Name: "Mara", Occasion: "Jubiläum"}
		}},
		{"gwg", func(r *service.ReceiptRequest) { r.Positions[0].PostingGroup = "gwg" }},
		{"ausschluss", func(r *service.ReceiptRequest) { r.Positions[0].PostingGroup = "repraesentation" }},
		{"bewirtungsgruppe", func(r *service.ReceiptRequest) { r.Positions[0].PostingGroup = "bewirtung" }},
		{"geschenkgruppe", func(r *service.ReceiptRequest) { r.Positions[0].PostingGroup = "geschenke" }},
		{"unbekannte_gruppe", func(r *service.ReceiptRequest) { r.Positions[0].PostingGroup = "unbekannt" }},
		{"keine_gruppe", func(r *service.ReceiptRequest) { r.Positions[0].PostingGroup = "" }},
		{"uebersteuerung", func(r *service.ReceiptRequest) { r.OverrideReason = "Berichtigte Rechnung" }},
		{"leere_uebersteuerung", func(r *service.ReceiptRequest) { r.OverrideReason = " \t" }},
		{"bewirtung", func(r *service.ReceiptRequest) { r.Entertainment = &domain.EntertainmentDetail{} }},
		{"rueckstellung", func(r *service.ReceiptRequest) { r.ProvisionID = 1 }},
		{"anzahlung", func(r *service.ReceiptRequest) { r.AdvanceTarget = accounting.AdvanceTargetInventory }},
		{"schlussrechnung", func(r *service.ReceiptRequest) { r.SettledAdvanceIDs = []uint{1} }},
		{"auslage", func(r *service.ReceiptRequest) { r.CreditorContactID = 2 }},
		{"fremdbetrag", func(r *service.ReceiptRequest) { r.ForeignAmount = 10000 }},
		{"leistungsnachweis", func(r *service.ReceiptRequest) { r.ServiceProof = "Bestellung geprüft" }},
		{"mehrere_positionen", func(r *service.ReceiptRequest) {
			r.Positions[0].Account = "6815"
			r.Positions[0].InputTaxShare = 600
			r.Positions[0].Gift = &service.GiftInput{Name: "Mara"}
			r.Positions[0].PostingGroup = "geschenke"
			r.Positions = append(r.Positions, r.Positions[0])
			r.OverrideReason = "Prüfung"
			r.Entertainment = &domain.EntertainmentDetail{}
			r.ProvisionID, r.CreditorContactID = 1, 2
			r.AdvanceTarget = accounting.AdvanceTargetInventory
			r.ForeignAmount = 20000
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, req := receiptGoldenServices(t)
			tc.change(&req)
			// Ohne vollständige Gruppierung und bei Fremdwährungen wird nicht
			// gerechnet. Auch diese Einstufung muss alle Eingaben behalten.
			for _, mode := range []string{"ohne_vorschau", "fremdwaehrung"} {
				if mode == "fremdwaehrung" {
					req.Currency = "USD"
				}
				facts, err := receiptFacts(context.Background(), s, req, "2026-05-10", mode == "fremdwaehrung")
				if err != nil {
					t.Fatal(err)
				}
				assertGolden(t, "receipt_specials/"+tc.name+"_"+mode, ClassifyReceipt(facts, LimitsFrom(nil)))
			}
		})
	}
}

func TestGoldenReceiptPreviewFindings(t *testing.T) {
	for _, name := range []string{"routine", "anteil", "ausschluss", "befund_vorsteuer", "uebersteuerung"} {
		t.Run(name, func(t *testing.T) {
			s, req := receiptGoldenServices(t)
			switch name {
			case "anteil":
				req.Positions[0].InputTaxShare, req.Positions[0].InputTaxShareReason = 600, "Flächenmaßstab"
			case "ausschluss":
				req.Positions[0].PostingGroup = "repraesentation"
			case "befund_vorsteuer", "uebersteuerung":
				req.Positions[0].Net = 30000
				if name == "uebersteuerung" {
					req.OverrideReason = "Berichtigte Rechnung"
				}
			}
			f, err := ReceiptFactsFor(context.Background(), s, req, "2026-05-10")
			if err != nil {
				t.Fatal(err)
			}
			assertGolden(t, "receipt_findings/"+name, ClassifyReceipt(f, LimitsFrom(nil)))
		})
	}
}

func receiptGoldenServices(t *testing.T) (*core.Services, service.ReceiptRequest) {
	t.Helper()
	ctx := context.Background()
	db, err := repository.InitInMemoryDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repository.CloseDB(db) })
	if err := repository.SeedDefaultsIfEmpty(ctx, db, 2026); err != nil {
		t.Fatal(err)
	}
	contacts := repository.NewContactRepository(db)
	contact := &domain.Contact{Type: domain.ContactTypeVendor, Name: "Lieferant GmbH", CountryCode: "DE", Street: "Weg 3", PostalCode: "20095", City: "Hamburg", LedgerAccount: "70001"}
	if err := contacts.Save(ctx, contact); err != nil {
		t.Fatal(err)
	}
	audit := repository.NewAuditRepository(db)
	receipts := service.NewReceiptService(repository.NewReceiptRepository(db), receiptstore.New(t.TempDir()), audit)
	receipt, err := receipts.File(ctx, service.FileReceiptRequest{
		Direction: domain.DirectionIncoming, DocumentDate: "2026-03-10", IssuerName: contact.Name, GrossAmount: 11900,
		Files: []service.NewFile{{Role: domain.ReceiptRoleOriginal, FileName: "beleg.pdf", Content: []byte("%PDF-1.4\nTestbeleg\n%%EOF\n")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	journal := service.NewJournalService(repository.NewJournalRepository(db), repository.NewAccountRepository(db), contacts, audit, repository.NewSettingsRepository(db), repository.NewTxRunner(db))
	posting := service.NewPostingService(journal, contacts)
	posting.SetReceiptService(receipts)
	s := &core.Services{Repos: core.Repos{Contacts: contacts}, Receipts: receipts, Posting: posting}
	return s, service.ReceiptRequest{ReceiptID: receipt.ID, ContactID: contact.ID, BookingDate: "2026-03-10", DocumentDate: "2026-03-10",
		TaxTreatment: domain.TaxTreatmentDomestic, Settlement: service.SettlementOpen,
		Positions: []service.ReceiptPosition{{PostingGroup: "software", Net: 10000, TaxRate: domain.TaxRateStandard}},
	}
}
