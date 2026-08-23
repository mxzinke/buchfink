package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Buchungspfad kommt ohne das E-Rechnungsmodul aus.
//
// Kein Test in dieser Datei erzeugt oder liest ein XML. Was der Buchungspfad
// von einer empfangenen Rechnung braucht, wird von Hand hingeschrieben — und
// genau das ist der Sinn der Schnittstelle: die Buchungsregeln lassen sich
// prüfen, ohne dass ein Format im Spiel ist, und das Modul lässt sich prüfen,
// ohne dass ein Konto im Spiel ist.

// fakeReader liefert, was der Test vorgibt.
type fakeReader struct {
	invoice *domain.IncomingInvoice
	err     error
}

func (f fakeReader) Read(data []byte) (*domain.IncomingInvoice, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.invoice, nil
}

func (f fakeReader) ValidateOnly(data []byte) (domain.ReceiptValidation, error) {
	if f.err != nil {
		return domain.ReceiptValidation{}, f.err
	}
	return f.invoice.Validation, nil
}

// receivedInvoice is an ordinary incoming invoice, written out by hand.
func receivedInvoice() *domain.IncomingInvoice {
	return &domain.IncomingInvoice{
		Syntax:       "cii",
		Profile:      "urn:cen.eu:en16931:2017",
		ProfileLabel: "EN 16931 (COMFORT)",
		Kind:         domain.EInvoiceKindInvoice,
		Number:       "LR-2026-0815",
		IssueDate:    "2026-05-04",
		DeliveryDate: "2026-04-30",
		Currency:     "EUR",
		Supplier: domain.InvoiceParty{
			Name: "Netzwerk GmbH", VatID: "DE555666777", CountryCode: "DE",
		},
		Buyer:         domain.InvoiceParty{Name: "Pfennig Ventures GmbH", CountryCode: "DE"},
		NetAmount:     100000,
		TaxAmount:     19000,
		GrossAmount:   119000,
		TaxCategories: []string{"S"},
		Positions: []domain.IncomingPosition{
			{Text: "Wartung", Net: 100000, TaxRate: domain.TaxRateStandard, TaxCategory: "S"},
		},
	}
}

// filedReceipt puts a Beleg on file whose structured part is a placeholder: the
// fake reader never looks at the bytes, which is exactly the point.
func (e *testEnv) filedReceipt(t *testing.T) *domain.Receipt {
	t.Helper()
	ctx := context.Background()

	receipt, err := e.receipts.File(ctx, FileReceiptRequest{
		Direction: domain.DirectionIncoming, FiscalYear: e.fiscalYear,
		ReceivedAt: "2026-05-05", ReceivedVia: "E-Mail",
		Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, FileName: "rechnung.xml",
				Content: []byte("<Invoice/>")},
			{Role: domain.ReceiptRoleStructured, FileName: "rechnung.xml",
				Content: []byte("<Invoice/>")},
		},
	})
	if err != nil {
		t.Fatalf("Beleg ablegen: %v", err)
	}
	return receipt
}

func (e *testEnv) einvoicesWith(reader EInvoiceReader) *EInvoiceService {
	return NewEInvoiceService(e.receipts, e.contactRepo, reader, e.fiscalYear)
}

// Der Vorschlag entsteht aus dem, was die Schnittstelle liefert — nicht aus
// einem Dokument.
func TestProposalIsBuiltFromTheInterface(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	receipt := env.filedReceipt(t)

	vendor := &domain.Contact{
		Name: "Netzwerk GmbH", Type: domain.ContactTypeVendor,
		VatID: "DE555666777", CountryCode: "DE",
	}
	if err := env.contacts.SaveContact(ctx, vendor); err != nil {
		t.Fatalf("Lieferant anlegen: %v", err)
	}

	svc := env.einvoicesWith(fakeReader{invoice: receivedInvoice()})
	proposal, err := svc.Propose(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}

	if proposal.Request.TaxTreatment != domain.TaxTreatmentDomestic {
		t.Errorf("Steuerfall = %q", proposal.Request.TaxTreatment)
	}
	if !proposal.MatchedContact || proposal.Request.ContactID != vendor.ID {
		t.Errorf("Lieferant nicht zugeordnet: %v / %d", proposal.MatchedContact, proposal.Request.ContactID)
	}
	if proposal.Request.ServiceDateFrom != "2026-04-30" {
		t.Errorf("Leistungsdatum = %q", proposal.Request.ServiceDateFrom)
	}
	if len(proposal.Request.Positions) != 1 || proposal.Request.Positions[0].Net != 100000 {
		t.Errorf("Positionen = %+v", proposal.Request.Positions)
	}
	if proposal.GrossAmount != 119000 {
		t.Errorf("Bruttobetrag = %s", proposal.GrossAmount)
	}
	// Die Buchungsgruppe bleibt offen: welches Aufwandskonto zutrifft, sagt
	// keine Rechnung.
	if proposal.Request.Positions[0].PostingGroup != "" {
		t.Error("die Buchungsgruppe darf nicht vorbelegt sein")
	}
}

// Der Steuerfall wird gedreht — auch das ohne jedes Dokument.
func TestTreatmentIsDerivedFromTheCategoryAlone(t *testing.T) {
	cases := []struct {
		category string
		want     domain.TaxTreatment
	}{
		{"S", domain.TaxTreatmentDomestic},
		{"AE", domain.TaxTreatmentReverseCharge},
		{"K", domain.TaxTreatmentIntraCommunityAcquisition},
		{"Z", domain.TaxTreatmentZeroRated},
	}
	for _, c := range cases {
		read := receivedInvoice()
		read.TaxCategories = []string{c.category}
		got, note := resolveTreatment(read)
		if note != "" {
			t.Errorf("%s: %s", c.category, note)
		}
		if got != c.want {
			t.Errorf("%s ergibt %q, erwartet %q", c.category, got, c.want)
		}
	}

	// Mehrere Kategorien auf einem Beleg: Buchfink führt den Steuerfall je
	// Beleg, nicht je Position.
	mixed := receivedInvoice()
	mixed.TaxCategories = []string{"S", "AE"}
	if got, note := resolveTreatment(mixed); got != "" || !strings.Contains(note, "mischt") {
		t.Errorf("gemischte Kategorien: %q / %q", got, note)
	}

	// Ohne Aufschlüsselung bleibt der Fall offen statt geraten.
	none := receivedInvoice()
	none.TaxCategories = nil
	if got, note := resolveTreatment(none); got != "" || note == "" {
		t.Errorf("ohne Kategorie: %q / %q", got, note)
	}
}

// Eine Gutschrift trägt positive Beträge und sagt nur im Rechnungstyp, was sie
// ist. Sie als Eingangsrechnung vorzuschlagen dreht das Vorzeichen der
// Vorsteuer und eröffnet einen offenen Posten, wo einer zu schließen wäre — und
// es sähe richtig aus.
func TestNonInvoiceDocumentsAreRefusedWithTheirName(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	receipt := env.filedReceipt(t)

	for _, kind := range []domain.EInvoiceKind{
		domain.EInvoiceKindCreditNote,
		domain.EInvoiceKindCorrection,
		domain.EInvoiceKindPrepayment,
		domain.EInvoiceKindSelfBilled,
	} {
		read := receivedInvoice()
		read.Kind = kind
		_, err := env.einvoicesWith(fakeReader{invoice: read}).Propose(ctx, receipt.ID)
		if err == nil {
			t.Errorf("%s hätte keinen Buchungsvorschlag ergeben dürfen", kind.Label())
			continue
		}
		if !strings.Contains(err.Error(), kind.Label()) {
			t.Errorf("die Meldung nennt den Fall nicht: %v", err)
		}
	}
}

// Was der Datensatz mitschickt, gehört zum Beleg. Der Stundenzettel zur
// Rechnung ist Aufbewahrungsgegenstand wie die Rechnung selbst.
func TestEnclosuresBecomeReceiptFiles(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt, err := env.receipts.File(ctx, FileReceiptRequest{
		Direction: domain.DirectionIncoming, FiscalYear: env.fiscalYear,
		ReceivedAt: "2026-05-05", ReceivedVia: "E-Mail",
		Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, FileName: "rechnung.xml", Content: []byte("<Invoice/>")},
		},
	})
	if err != nil {
		t.Fatalf("Beleg ablegen: %v", err)
	}

	read := receivedInvoice()
	read.Attachments = []domain.IncomingAttachment{
		{FileName: "stundenzettel.pdf", MimeType: "application/pdf", Content: []byte("%PDF-1.4 Stunden")},
	}
	read.Validation = domain.ReceiptValidation{
		Format: "cii", Profile: read.Profile, At: "2026-05-05 10:00:00",
		Ruleset: "buchfink-en16931/2026.3", Version: "buchfink-en16931/2026.3",
		Coverage: "full",
	}

	updated, err := env.einvoicesWith(fakeReader{invoice: read}).ExtractStructuredPart(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("strukturierten Teil holen: %v", err)
	}

	var enclosures []domain.ReceiptFile
	for _, f := range updated.Files {
		if f.Role == domain.ReceiptRoleAttachment {
			enclosures = append(enclosures, f)
		}
	}
	if len(enclosures) != 1 {
		t.Fatalf("%d Anlagen am Beleg, erwartet 1", len(enclosures))
	}
	if enclosures[0].FileName != "stundenzettel.pdf" {
		t.Errorf("Dateiname = %q", enclosures[0].FileName)
	}
	if !enclosures[0].Derived {
		t.Error("eine aus dem Datensatz gezogene Anlage ist abgeleitet")
	}
	if updated.ValidationCoverage != "full" {
		t.Errorf("Prüfumfang = %q", updated.ValidationCoverage)
	}
}

// Was sich nicht lesen lässt, wird auch nicht als strukturierter Teil an den
// Beleg gehängt — sonst stünde dort eine Datei, aus der niemand etwas gewinnt.
func TestUnreadableRecordLeavesTheReceiptUnchanged(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt, err := env.receipts.File(ctx, FileReceiptRequest{
		Direction: domain.DirectionIncoming, FiscalYear: env.fiscalYear,
		ReceivedAt: "2026-05-05", ReceivedVia: "E-Mail",
		Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, FileName: "rechnung.xml", Content: []byte("<Invoice/>")},
		},
	})
	if err != nil {
		t.Fatalf("Beleg ablegen: %v", err)
	}

	svc := env.einvoicesWith(fakeReader{err: fmt.Errorf("das Profil ZUGFeRD MINIMUM enthält keine vollständige Rechnung")})
	if _, err := svc.ExtractStructuredPart(ctx, receipt.ID); err == nil {
		t.Fatal("ein unlesbarer Datensatz muss abgewiesen werden")
	}

	after, err := env.receipts.Get(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Beleg laden: %v", err)
	}
	if _, ok := after.FileByRole(domain.ReceiptRoleStructured); ok {
		t.Error("der Beleg trägt einen strukturierten Teil, obwohl das Lesen scheiterte")
	}
	if len(after.Files) != 1 {
		t.Errorf("%d Dateien am Beleg, erwartet 1", len(after.Files))
	}
}
