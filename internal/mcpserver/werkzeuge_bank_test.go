package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/inbox"
	"github.com/buchfink/buchfink/internal/service"
)

// camtEntry ist ein Umsatz eines CAMT.053-Auszugs; betrag negativ für eine
// Abbuchung.
type camtEntry struct {
	amount, date, e2e, party, text string
}

// camt baut einen Auszug; opening und closing leer heißt: ohne Salden.
func camt(iban, opening, openDate, closing, closeDate string, entries ...camtEntry) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02"><BkToCstmrStmt><Stmt>
<Acct><Id><IBAN>` + iban + `</IBAN></Id></Acct>`)
	balance := func(code, amount, date string) {
		dir := "CRDT"
		if strings.HasPrefix(amount, "-") {
			dir, amount = "DBIT", strings.TrimPrefix(amount, "-")
		}
		fmt.Fprintf(&b, `<Bal><Tp><CdOrPrtry><Cd>%s</Cd></CdOrPrtry></Tp><Amt Ccy="EUR">%s</Amt><CdtDbtInd>%s</CdtDbtInd><Dt><Dt>%s</Dt></Dt></Bal>`,
			code, amount, dir, date)
	}
	if opening != "" {
		balance("OPBD", opening, openDate)
	}
	if closing != "" {
		balance("CLBD", closing, closeDate)
	}
	for _, e := range entries {
		dir, party, amount := "CRDT", "Dbtr", e.amount
		if strings.HasPrefix(amount, "-") {
			dir, party, amount = "DBIT", "Cdtr", strings.TrimPrefix(amount, "-")
		}
		fmt.Fprintf(&b, `<Ntry><Amt Ccy="EUR">%s</Amt><CdtDbtInd>%s</CdtDbtInd><BookgDt><Dt>%s</Dt></BookgDt><ValDt><Dt>%s</Dt></ValDt>`+
			`<NtryDtls><TxDtls><Refs><EndToEndId>%s</EndToEndId></Refs><RltdPties><%s><Nm>%s</Nm></%s></RltdPties>`+
			`<RmtInf><Ustrd>%s</Ustrd></RmtInf></TxDtls></NtryDtls></Ntry>`,
			amount, dir, e.date, e.date, e.e2e, party, e.party, party, e.text)
	}
	b.WriteString(`</Stmt></BkToCstmrStmt></Document>`)
	return b.String()
}

// aprilEntries sind die Umsätze des Auszugs April.
var aprilEntries = []camtEntry{
	{"-119.00", "2026-04-10", "E2E-1", "Netzwerk GmbH", "Rechnung RE-LIEF-2001"},
	{"-42.00", "2026-04-11", "E2E-2", "Stadtwerke", "Abschlag Strom"},
	{"10.00", "2026-04-12", "E2E-3", "Hausbank", "Zinsen Q1"},
}

const ownIBAN = "DE89370400440532013000"

type umsatz struct {
	ID          uint   `json:"id"`
	Betrag      string `json:"betrag"`
	Status      string `json:"status"`
	Geprueft    bool   `json:"auszug_geprueft"`
	Gesamt      int    `json:"vorschlaege_gesamt"`
	Vorschlaege []struct {
		Art        string `json:"art"`
		Einstufung string `json:"einstufung"`
		Posten     []struct {
			BuchungID uint `json:"buchung_id"`
		} `json:"posten"`
	} `json:"vorschlaege"`
}

// Der Kontoauszug kommt per Pfad; nur der exakte Treffer wird direkt
// zugeordnet, alles andere geht mit Rückfrage.
func TestBankStatementAndMatching(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	vendorID := h.vendor()
	a := h.agent("")

	invoice := a.file(h.eInvoice("RE-LIEF-2001", "2026-04-01", 10000))
	var booked buchungAntwort
	a.ok("beleg_buchen_mit_rueckfrage", a.args(bookArgs(invoice.Beleg.ID, vendorID, "2026-04-01", "100.00")), &booked)
	payable := booked.Buchung.ID

	var open struct {
		Posten []struct {
			BuchungID uint   `json:"buchung_id"`
			Richtung  string `json:"richtung"`
			Offen     string `json:"offen"`
		} `json:"posten"`
		SummeOffen     string `json:"summe_offen"`
		Altersstruktur []struct {
			Faelligkeit []struct {
				Schluessel string `json:"schluessel"`
			} `json:"faelligkeit"`
		} `json:"altersstruktur"`
	}
	a.ok("offene_posten", map[string]any{"richtung": "verbindlichkeit", "mit_altersstruktur": true}, &open)
	if len(open.Posten) != 1 || open.Posten[0].BuchungID != payable || open.Posten[0].Offen != "119.00" ||
		len(open.Altersstruktur) == 0 || len(open.Altersstruktur[0].Faelligkeit) == 0 {
		t.Fatalf("offene Posten: %+v", open)
	}

	statement := h.inboxFile("auszug.xml", []byte(camt(ownIBAN, "1000.00", "2026-04-01", "849.00", "2026-04-30", aprilEntries...)))
	// Das Bankkonto richtet nur der Mensch ein.
	e := a.fail("kontoauszug_importieren", a.args(map[string]any{"pfad": statement}), CodeAppOnly)
	if e.Werkzeug != "nur_app" {
		t.Errorf("unbekanntes Bankkonto: %+v", e)
	}
	// Der Mensch richtet das Konto ein und liest den März ein; an dessen
	// Endsaldo schließt der April an.
	h.ui(func(ctx context.Context, s *core.Services) error {
		if err := s.Bank.ConfigureAccounts(ctx, []domain.BankAccount{{IBAN: ownIBAN, Name: "Geschäftskonto", LedgerAccount: "1800", Currency: "EUR"}}); err != nil {
			return err
		}
		_, err := s.Bank.ImportStatement(ctx, []byte(camt(ownIBAN, "1000.00", "2026-03-01", "1000.00", "2026-03-31")), "", nil)
		return err
	})

	// Ein selbst geschriebener Auszug ohne Salden lässt sich einlesen, aber
	// nichts daraus wird ohne Rückfrage zugeordnet.
	fake := h.inboxFile("gefaelscht.xml", []byte(camt(ownIBAN, "", "", "", "",
		camtEntry{"-119.00", "2026-04-09", "FAKE-1", "Netzwerk GmbH", "RE-LIEF-2001"})))
	var forged struct {
		Geprueft bool     `json:"salden_geprueft"`
		Hinweise []string `json:"hinweise"`
	}
	a.ok("kontoauszug_importieren", a.args(map[string]any{"pfad": fake}), &forged)
	if forged.Geprueft || len(forged.Hinweise) == 0 || !strings.Contains(forged.Hinweise[0], "keinen Anfangs- und Endsaldo") {
		t.Fatalf("Auszug ohne Salden: %+v", forged)
	}
	var forgedList struct {
		Umsaetze []umsatz `json:"umsaetze"`
	}
	a.ok("bankumsaetze", nil, &forgedList)
	if len(forgedList.Umsaetze) != 1 || forgedList.Umsaetze[0].Geprueft {
		t.Fatalf("ungeprüfter Umsatz: %+v", forgedList)
	}
	e = a.fail("bankumsatz_zuordnen", a.args(map[string]any{"bankumsatz_id": forgedList.Umsaetze[0].ID, "buchung_id": payable}),
		CodeApprovalNeeded)
	if !hasReason(e, "auszug_ungeprueft") || hasReason(e, "kein_exakter_treffer") {
		t.Fatalf("Zuordnung aus ungeprüftem Auszug: %+v", e.Gruende)
	}
	a.ok("bankumsatz_buchen", a.args(map[string]any{"bankumsatz_id": forgedList.Umsaetze[0].ID,
		"ignorieren": map[string]any{"grund": "Auszug nicht von der Bank"}}), nil)

	var imported struct {
		BelegID    uint     `json:"beleg_id"`
		Umsaetze   int      `json:"umsaetze"`
		Neu        int      `json:"neu"`
		Konten     []string `json:"konten"`
		Geprueft   bool     `json:"salden_geprueft"`
		Verschoben []struct {
			Von  string `json:"von"`
			Nach string `json:"nach"`
		} `json:"verschoben"`
	}
	a.ok("kontoauszug_importieren", a.args(map[string]any{"pfad": statement}), &imported)
	if imported.Umsaetze != 3 || imported.Neu != 3 || imported.BelegID == 0 || len(imported.Konten) != 1 ||
		imported.Konten[0] != "1800" || !imported.Geprueft {
		t.Fatalf("Import: %+v", imported)
	}
	// Der Auszug ging aus dem Eingang nach Übernommen/<JJJJ-MM>.
	if len(imported.Verschoben) != 1 || imported.Verschoben[0].Von != statement ||
		filepath.Dir(imported.Verschoben[0].Nach) != filepath.Join(filepath.Dir(statement), inbox.DoneDir, time.Now().Format("2006-01")) {
		t.Fatalf("verschoben: %+v", imported.Verschoben)
	}
	again := imported
	again.Verschoben = nil
	a.ok("kontoauszug_importieren", a.args(map[string]any{"pfad": imported.Verschoben[0].Nach}), &again)
	if again.Neu != 0 || again.Umsaetze != 3 || len(again.Verschoben) != 0 {
		t.Errorf("zweiter Import desselben Auszugs: %+v", again)
	}
	a.fail("kontoauszug_importieren", a.args(map[string]any{"pfad": "/etc/passwd"}), CodePathNotAllowed)

	var list struct {
		Umsaetze []umsatz `json:"umsaetze"`
	}
	a.ok("bankumsaetze", nil, &list)
	if len(list.Umsaetze) != 3 {
		t.Fatalf("offene Umsätze: %+v", list)
	}
	byAmount := map[string]umsatz{}
	for _, u := range list.Umsaetze {
		byAmount[u.Betrag] = u
	}
	payment, power, interest := byAmount["-119.00"], byAmount["-42.00"], byAmount["10.00"]
	if len(payment.Vorschlaege) == 0 || payment.Vorschlaege[0].Einstufung != "direkt" ||
		payment.Vorschlaege[0].Posten[0].BuchungID != payable || !payment.Geprueft {
		t.Fatalf("Vorschlag zum exakten Treffer: %+v", payment)
	}

	// Ohne Nummer im Verwendungszweck ist es keine Routine.
	e = a.fail("bankumsatz_zuordnen", a.args(map[string]any{"bankumsatz_id": power.ID, "buchung_id": payable}), CodeApprovalNeeded)
	if e.Werkzeug != "bankumsatz_buchen" || !hasReason(e, "kein_exakter_treffer") {
		t.Errorf("Zuordnung ohne Treffer: %+v", e)
	}
	matchArgs := a.args(map[string]any{"bankumsatz_id": payment.ID, "buchung_id": payable, "idempotenz_schluessel": "bank-119-zuordnen"})
	var matched, replay buchungAntwort
	a.ok("bankumsatz_zuordnen", matchArgs, &matched)
	a.ok("bankumsatz_zuordnen", matchArgs, &replay)
	if matched.Einstufung != "direkt" || matched.Buchung == nil || !replay.Wiederholt || replay.Buchung.ID != matched.Buchung.ID {
		t.Fatalf("direkte Zuordnung: %+v / %+v", matched, replay)
	}
	a.ok("offene_posten", map[string]any{"richtung": "verbindlichkeit"}, &open)
	if len(open.Posten) != 0 {
		t.Errorf("der Posten ist ausgeglichen: %+v", open)
	}

	// Mit Rückfrage: ignorieren mit Grund, Zinsen auf ein Konto.
	a.fail("bankumsatz_buchen", a.args(map[string]any{"bankumsatz_id": power.ID}), CodeInvalidInput)
	var ignored struct {
		Status string `json:"status"`
	}
	a.ok("bankumsatz_buchen", a.args(map[string]any{"bankumsatz_id": power.ID,
		"ignorieren": map[string]any{"grund": "privat gezahlt, Erstattung folgt über Auslage"}}), &ignored)
	if ignored.Status != "ignoriert" {
		t.Errorf("ignoriert: %+v", ignored)
	}
	var split struct {
		Buchung *struct {
			ID uint `json:"id"`
		} `json:"buchung"`
		Status string `json:"status"`
	}
	a.ok("bankumsatz_buchen", a.args(map[string]any{"bankumsatz_id": interest.ID,
		"aufteilung": []any{map[string]any{"konto": "7100", "betrag": "10.00", "text": "Zinsen Q1"}}}), &split)
	if split.Buchung == nil || split.Status != "zugeordnet" {
		t.Errorf("Aufteilung: %+v", split)
	}
	a.ok("bankumsaetze", nil, &list)
	if len(list.Umsaetze) != 0 {
		t.Errorf("alle Umsätze erledigt: %+v", list)
	}
	a.ok("bankumsaetze", map[string]any{"status": "alle"}, &list)
	if len(list.Umsaetze) != 4 {
		t.Errorf("alle Umsätze: %+v", list)
	}
}

// Viele offene Posten eines Lieferanten und viele Umsätze: bankumsaetze
// bleibt knapp. Je Umsatz höchstens drei Vorschläge, kein Vorschlag allein
// wegen der Fälligkeit, und die volle Liste gibt es je Umsatz.
func TestBankTransactionsStayCompact(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	vendorID := h.vendor()
	a := h.agent("")
	var entries []camtEntry
	h.ui(func(ctx context.Context, s *core.Services) error {
		c, err := s.Contacts.Contact(ctx, vendorID)
		if err != nil {
			return err
		}
		if err := s.Bank.ConfigureAccounts(ctx, []domain.BankAccount{{IBAN: ownIBAN, Name: "Geschäftskonto", LedgerAccount: "1800", Currency: "EUR"}}); err != nil {
			return err
		}
		for i := 0; i < 30; i++ {
			amount := domain.Cents(10000 + 100*i)
			date := fmt.Sprintf("2026-04-%02d", i%28+1)
			r, err := s.Receipts.File(ctx, service.FileReceiptRequest{Direction: domain.DirectionIncoming, ReceivedAt: date,
				DocumentDate: date, IssuerName: c.Name, GrossAmount: amount, Subject: fmt.Sprintf("Rechnung %d", i),
				Files: []service.NewFile{{Role: domain.ReceiptRoleOriginal, FileName: fmt.Sprintf("r%d.pdf", i), Content: pdf(fmt.Sprint(i))}}})
			if err != nil {
				return err
			}
			id := vendorID
			if _, err := s.Posting.PostManualEntry(ctx, service.ManualEntryRequest{ReceiptID: r.ID, Entry: domain.JournalEntry{
				BookingDate: date, DocumentDate: date, ServiceDateFrom: date, ServiceDateTo: date, DueDate: date,
				Description: fmt.Sprintf("Wartung %d", i), TaxTreatment: domain.TaxTreatmentNotTaxable, ContactID: &id,
				Lines: []domain.JournalLine{{Position: 1, Side: domain.SideDebit, Account: "6300", Amount: amount},
					{Position: 2, Side: domain.SideCredit, Account: c.LedgerAccount, Amount: amount, ContactID: &id}}}}); err != nil {
				return err
			}
			entries = append(entries, camtEntry{fmt.Sprintf("-%d.%02d", (amount+700)/100, (amount+700)%100), date,
				fmt.Sprintf("E2E-%d", i), "Netzwerk GmbH", fmt.Sprintf("Zahlung %d", i)})
		}
		_, err = s.Bank.ImportStatement(ctx, []byte(camt(ownIBAN, "", "", "", "", entries...)), "", nil)
		return err
	})

	res := callTool(t, a.cs, "bankumsaetze", map[string]any{"limit": 25})
	if res.IsError {
		t.Fatal(resultText(res))
	}
	body := resultText(res)
	var list struct {
		Umsaetze []umsatz `json:"umsaetze"`
	}
	if err := json.Unmarshal([]byte(body), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Umsaetze) != 25 {
		t.Fatalf("Seite: %d Umsätze", len(list.Umsaetze))
	}
	most := 0
	for _, u := range list.Umsaetze {
		if len(u.Vorschlaege) > maxSuggestions || u.Gesamt < len(u.Vorschlaege) {
			t.Fatalf("Umsatz %d: %d Vorschläge, gesamt %d", u.ID, len(u.Vorschlaege), u.Gesamt)
		}
		most = max(most, u.Gesamt)
	}
	t.Logf("bankumsaetze: %d Bytes, höchstens %d Vorschläge je Umsatz", len(body), most)
	if len(body) > 60_000 {
		t.Errorf("Antwort mit %d Bytes zu groß", len(body))
	}
	if most <= maxSuggestions {
		t.Fatalf("der Test braucht einen Umsatz mit mehr als %d Vorschlägen (höchstens %d)", maxSuggestions, most)
	}
	for _, u := range list.Umsaetze {
		if u.Gesamt <= maxSuggestions {
			continue
		}
		var one struct {
			Umsaetze []umsatz `json:"umsaetze"`
		}
		a.ok("bankumsaetze", map[string]any{"bankumsatz_id": u.ID}, &one)
		if len(one.Umsaetze) != 1 || len(one.Umsaetze[0].Vorschlaege) != u.Gesamt {
			t.Errorf("volle Liste zu Umsatz %d: %+v", u.ID, one)
		}
		break
	}
}
