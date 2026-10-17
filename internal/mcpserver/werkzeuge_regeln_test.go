package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/core/policy"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Eine Vorschau verbraucht den Idempotenzschlüssel nicht: derselbe Schlüssel
// bucht danach.
func TestPreviewKeepsTheKey(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	vendorID := h.vendor()
	a := h.agent("")
	var photo ablageOut
	a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": "2026-05-07", "dateien": []any{map[string]any{"pfad": h.inboxFile("quittung.pdf", pdf("Quittung"))}},
		"belegdatum": "2026-05-07", "aussteller": supplier.CompanyName, "brutto": "119.00"}), &photo)
	args := a.args(bookArgs(photo.Beleg.ID, vendorID, "2026-05-07", "100.00"))
	args["idempotenz_schluessel"] = "quittung-0507"
	args["nur_vorschau"] = true
	var preview, booked buchungAntwort
	a.ok("beleg_buchen_mit_rueckfrage", args, &preview)
	delete(args, "nur_vorschau")
	a.ok("beleg_buchen_mit_rueckfrage", args, &booked)
	if preview.Vorschau == nil || preview.Buchung != nil || booked.Buchung == nil || booked.Wiederholt {
		t.Fatalf("Vorschau und Buchung: %+v / %+v", preview, booked)
	}

	// Die Datei des Belegs liest der Agent aus einer Kopie im Exportordner.
	var copied struct {
		Pfad   string `json:"pfad"`
		Rolle  string `json:"rolle"`
		SHA256 string `json:"sha256"`
	}
	a.ok("beleg_datei_exportieren", a.args(map[string]any{"beleg_id": photo.Beleg.ID}), &copied)
	if data, err := os.ReadFile(copied.Pfad); err != nil || string(data) != string(pdf("Quittung")) || copied.Rolle != "original" {
		t.Errorf("Kopie: %+v %v", copied, err)
	}
	a.fail("beleg_datei_exportieren", a.args(map[string]any{"beleg_id": photo.Beleg.ID, "datei_id": 999}), CodeNotFound)
	a.fail("beleg_datei_exportieren", a.args(map[string]any{"beleg_id": 999}), CodeNotFound)
}

// Eine Handbuchung eines Agenten belegt keine Vorauszahlung; erledigt ist die
// Pflicht erst mit einer Zahlung, die der Mensch bucht oder die vom Konto kommt.
func TestPrepaymentNeedsARealPayment(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")
	a.ok("anlass_erfassen", a.args(map[string]any{"art": "vorauszahlungsbescheid-kst", "datum": "2026-02-01",
		"idempotenz_schluessel": "bescheid-kst-2026"}), nil)
	status := func() string {
		var due struct {
			Pflichten []struct {
				Status string `json:"status"`
			} `json:"pflichten"`
		}
		a.ok("fristen", map[string]any{"schluessel": "kst-vorauszahlung.2026-Q3"}, &due)
		if len(due.Pflichten) != 1 {
			t.Fatalf("Pflicht: %+v", due)
		}
		return due.Pflichten[0].Status
	}
	if st := status(); st == "erledigt" {
		t.Fatalf("vor der Zahlung: %s", st)
	}
	var notice ablageOut
	a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": "2026-02-03", "dateien": []any{map[string]any{"pfad": h.inboxFile("bescheid.pdf", pdf("Bescheid"))}},
		"belegdatum": "2026-02-01", "aussteller": "Finanzamt", "brutto": "2000.00"}), &notice)
	a.ok("buchung_erfassen", a.args(map[string]any{"idempotenz_schluessel": "kst-q3-laut-bescheid", "buchungsdatum": "2026-09-10",
		"beschreibung": "KSt-Vorauszahlung Q3 laut Bescheid", "steuerfall": "not_taxable", "beleg_id": notice.Beleg.ID,
		"zeilen": []any{map[string]any{"konto": "7600", "seite": "S", "betrag": "500.00"},
			map[string]any{"konto": "1800", "seite": "H", "betrag": "500.00"}}}), nil)
	if st := status(); st == "erledigt" {
		t.Errorf("eine Handbuchung des Agenten erledigt die Vorauszahlung: %s", st)
	}
	h.ui(func(ctx context.Context, s *core.Services) error {
		r, err := s.Receipts.File(ctx, service.FileReceiptRequest{Direction: domain.DirectionIncoming, ReceivedAt: "2026-09-11",
			DocumentDate: "2026-09-11", IssuerName: "Finanzamt", GrossAmount: 50000,
			Files: []service.NewFile{{Role: domain.ReceiptRoleOriginal, FileName: "zahlung.pdf", Content: pdf("Zahlung")}}})
		if err != nil {
			return err
		}
		_, err = s.Posting.PostManualEntry(ctx, service.ManualEntryRequest{ReceiptID: r.ID,
			Entry: domain.JournalEntry{BookingDate: "2026-09-11", DocumentDate: "2026-09-11", ServiceDateFrom: "2026-09-11",
				ServiceDateTo: "2026-09-11", Description: "KSt-Vorauszahlung Q3", TaxTreatment: domain.TaxTreatmentNotTaxable,
				Lines: []domain.JournalLine{{Position: 1, Side: domain.SideDebit, Account: "7600", Amount: 50000},
					{Position: 2, Side: domain.SideCredit, Account: "1800", Amount: 50000}}}})
		return err
	})
	if st := status(); st != "erledigt" {
		t.Errorf("nach der Zahlung des Menschen: %s", st)
	}
}

// Eine unbekannte Kennung nennt das Feld und das Werkzeug, mit dem sie sich
// finden lässt.
func TestUnknownIDsNameTheirField(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")
	var photo ablageOut
	a.ok("beleg_ablegen", a.args(map[string]any{"eingang_am": "2026-05-07", "dateien": []any{map[string]any{"pfad": h.inboxFile("q.pdf", pdf("Q"))}},
		"belegdatum": "2026-05-07", "aussteller": "Laden", "brutto": "11.90"}), &photo)
	e := a.fail("beleg_buchen_mit_rueckfrage", a.args(bookArgs(photo.Beleg.ID, 999, "2026-05-07", "10.00")), CodeNotFound)
	if e.Werkzeug != "kontakte_suchen" || !strings.Contains(e.Nachricht, "kontakt_id") {
		t.Errorf("unbekannter Kontakt: %+v", e)
	}
	e = a.fail("beleg_buchen", a.args(bookArgs(999, 999, "2026-05-07", "10.00")), CodeNotFound)
	if e.Werkzeug == "" {
		t.Errorf("unbekannter Beleg: %+v", e)
	}
}

// Die Pflichtart heißt mancherorts anders als ihr Abschnitt in pflichten.md;
// fristen nimmt beide an und nennt den Anker.
func TestObligationAnchorsAreAccepted(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")
	var due struct {
		Pflichten []struct {
			Schluessel string `json:"schluessel"`
			Art        string `json:"art"`
			Anker      string `json:"anker"`
			Verweis    string `json:"verweis"`
		} `json:"pflichten"`
	}
	a.ok("fristen", map[string]any{"art": []any{"monatsfestschreibung"}, "geschaeftsjahr": 2026}, &due)
	if len(due.Pflichten) == 0 {
		t.Fatalf("Festschreibung über den Anker: %+v", due)
	}
	p := due.Pflichten[0]
	if p.Art != "festschreibung" || p.Anker != "monatsfestschreibung" || !strings.HasSuffix(p.Verweis, "#monatsfestschreibung") {
		t.Errorf("Anker: %+v", p)
	}
	alias := strings.Replace(p.Schluessel, "festschreibung.", "monatsfestschreibung.", 1)
	a.ok("fristen", map[string]any{"schluessel": alias}, &due)
	if len(due.Pflichten) != 1 || due.Pflichten[0].Schluessel != p.Schluessel {
		t.Errorf("Schlüssel über den Anker: %+v", due)
	}
}

// Ein Belegordner, der die Datenordner mehrerer Mandanten umschließt, macht
// die Belege der anderen nicht lesbar.
func TestReceiptFolderSkipsOtherTenants(t *testing.T) {
	h := newHarness(t)
	parent := t.TempDir()
	other, err := h.host.CreateTenant(uiCtx, "Fremd GmbH", filepath.Join(parent, "Fremd GmbH"),
		domain.CompanySettings{CompanyName: "Fremd GmbH", FiscalYear: 2026, LegalForm: domain.LegalFormGmbH})
	if err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(other.DataDir, policy.InboxDir, "geheim.xml")
	if err := os.MkdirAll(filepath.Dir(secret), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secret, []byte(h.eInvoiceXML(supplier, "GEHEIM-1", "2026-05-04", 1234500)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := h.host.Switch(uiCtx, h.tenant.ID); err != nil {
		t.Fatal(err)
	}
	h.ui(func(ctx context.Context, s *core.Services) error {
		_, err := s.Settings.SaveAgentSettings(ctx, domain.AgentSettings{ReceiptFolders: []string{parent}})
		return err
	})
	a := h.agent("")
	a.fail("beleg_ablegen", a.args(map[string]any{"eingang_am": "2026-05-04", "dateien": []any{map[string]any{"pfad": secret}}}),
		CodePathNotAllowed)
}
