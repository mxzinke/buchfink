package mcpserver

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
)

// routineVendor richtet einen Lieferanten ein, dessen E-Rechnungen direkt
// gehen: bestätigte Standardgruppe und drei frühere Buchungen.
func routineVendor(t *testing.T, h *harness, a *agent) uint {
	t.Helper()
	vendorID := h.vendor()
	h.ui(func(ctx context.Context, s *core.Services) error {
		_, err := s.Contacts.ConfirmDefaultPostingGroup(ctx, vendorID, "software")
		return err
	})
	for _, inv := range []struct{ number, date string }{
		{"RE-P-0001", "2026-02-02"}, {"RE-P-0002", "2026-03-02"}, {"RE-P-0003", "2026-04-02"},
	} {
		r := a.file(h.eInvoice(inv.number, inv.date, 10000))
		a.ok("beleg_buchen_mit_rueckfrage", a.args(bookArgs(r.Beleg.ID, vendorID, inv.date, "100.00")), nil)
	}
	return vendorID
}

// Eine Dublette erkennt Buchfink an Rechnungsnummer und Aussteller aus der
// E-Rechnung. Kopfdaten im Aufruf ändern daran nichts: bei einer E-Rechnung
// gelten sie nicht, und auch ein in ein anderes Jahr verschobener Beleg wird
// gefunden.
func TestDirectBookingFindsDuplicatesFromTheEInvoice(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")
	vendorID := routineVendor(t, h, a)

	xml := h.eInvoiceXML(supplier, "RE-P-DUP", "2026-05-04", 50000)
	first := a.file(h.inboxFile("RE-P-DUP.xml", []byte(xml)))
	var direct buchungAntwort
	a.ok("beleg_buchen", a.args(bookArgs(first.Beleg.ID, vendorID, "2026-05-04", "500.00")), &direct)
	if direct.Einstufung != "direkt" || direct.Buchung == nil {
		t.Fatalf("Routine: %+v", direct)
	}

	variants := []struct {
		name   string
		header map[string]any
	}{
		{"ohne Kopfdaten", nil},
		{"Betreff und anderes Datum", map[string]any{"betreff": "Wartung Mai", "belegdatum": "2026-05-05"}},
		{"Datum im Vorvorjahr", map[string]any{"belegdatum": "2024-05-04"}},
		{"Betreff und anderer Betrag", map[string]any{"betreff": "Wartung", "brutto": "595.01"}},
	}
	for i, v := range variants {
		copyPath := h.inboxFile("RE-P-DUP-"+string(rune('a'+i))+".xml", []byte(xml+"\n"+string(rune('a'+i))))
		view := h.inboxFile("RE-P-DUP-"+string(rune('a'+i))+".pdf", pdf("Darstellung"))
		in := map[string]any{"eingang_am": today(), "dateien": []any{map[string]any{"pfad": copyPath},
			map[string]any{"pfad": view, "rolle": "darstellung"}}}
		for k, val := range v.header {
			in[k] = val
		}
		var filed ablageOut
		a.ok("beleg_ablegen", a.args(in), &filed)
		if filed.BereitsVorhanden || !filed.ERechnung {
			t.Fatalf("%s: %+v", v.name, filed)
		}
		e := a.fail("beleg_buchen", a.args(bookArgs(filed.Beleg.ID, vendorID, "2026-05-04", "500.00")), CodeApprovalNeeded)
		if !hasReason(e, "dublettenverdacht") {
			t.Errorf("%s: %+v", v.name, e.Gruende)
		}
	}

	// Ändert jemand die Kopfdaten in der App, bleibt es eine Dublette, und die
	// Abweichung von der E-Rechnung ist ein eigener Grund.
	edited := a.file(h.inboxFile("RE-P-DUP-x.xml", []byte(xml+"\n<!-- x -->")))
	h.ui(func(ctx context.Context, s *core.Services) error {
		r, err := s.Receipts.Get(ctx, edited.Beleg.ID)
		if err != nil {
			return err
		}
		_, err = s.Receipts.SaveHeader(ctx, r.ID, domain.ReceiptHeader{Kind: r.Kind, DocumentDate: "2025-01-10",
			IssuerName: r.IssuerName, GrossAmount: r.GrossAmount + 1, Currency: r.Currency, Subject: "anders"})
		return err
	})
	e := a.fail("beleg_buchen", a.args(bookArgs(edited.Beleg.ID, vendorID, "2026-05-04", "500.00")), CodeApprovalNeeded)
	if !hasReason(e, "dublettenverdacht") || !hasReason(e, "kopf_weicht_ab") {
		t.Errorf("geänderte Kopfdaten: %+v", e.Gruende)
	}
}

// Ein Aussteller, der nur den Namen eines vertrauten Lieferanten trägt, ist
// nicht dieser Lieferant. Direkt geht nur, was über USt-IdNr. oder
// Steuernummer zugeordnet ist und den Stammdaten nicht widerspricht.
func TestDirectBookingNeedsAStrongSupplierMatch(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")
	vendorID := routineVendor(t, h, a)

	impostor := *supplier
	impostor.VatID, impostor.TaxNumber = "DE136695976", "11/222/33333"
	impostor.Street, impostor.PostalCode, impostor.City = "Fremdweg 1", "10115", "Berlin"
	impostor.IBAN = "DE89370400440532013000"
	fake := a.file(h.inboxFile("RE-FREMD-1.xml", []byte(h.eInvoiceXML(&impostor, "RE-FREMD-1", "2026-05-06", 80000))))
	var read struct {
		Vorschlag *struct {
			KontaktErkannt bool   `json:"kontakt_erkannt"`
			ErkanntAn      string `json:"erkannt_an"`
		} `json:"vorschlag"`
	}
	a.ok("beleg_lesen", map[string]any{"beleg_id": fake.Beleg.ID}, &read)
	if read.Vorschlag == nil || read.Vorschlag.ErkanntAn != "name" {
		t.Fatalf("Zuordnung über den Namen: %+v", read.Vorschlag)
	}
	e := a.fail("beleg_buchen", a.args(bookArgs(fake.Beleg.ID, vendorID, "2026-05-06", "800.00")), CodeApprovalNeeded)
	if !hasReason(e, "kontakt_nicht_erkannt") {
		t.Errorf("fremder Aussteller: %+v", e.Gruende)
	}

	// Derselbe Lieferant, aber Zahlung auf ein anderes Konto als hinterlegt.
	h.ui(func(ctx context.Context, s *core.Services) error {
		c, err := s.Contacts.Contact(ctx, vendorID)
		if err != nil {
			return err
		}
		c.IBAN = "DE02120300000000202051"
		return s.Contacts.SaveContact(ctx, c)
	})
	other := *supplier
	other.IBAN = "DE89370400440532013000"
	redirected := a.file(h.inboxFile("RE-P-IBAN.xml", []byte(h.eInvoiceXML(&other, "RE-P-IBAN", "2026-05-07", 20000))))
	e = a.fail("beleg_buchen", a.args(bookArgs(redirected.Beleg.ID, vendorID, "2026-05-07", "200.00")), CodeApprovalNeeded)
	if !hasReason(e, "iban_weicht_ab") || hasReason(e, "kontakt_nicht_erkannt") {
		t.Errorf("anderes Zahlungskonto: %+v", e.Gruende)
	}
}
