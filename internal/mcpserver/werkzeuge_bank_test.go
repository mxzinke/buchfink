package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
)

const camtTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
  <BkToCstmrStmt>
    <Stmt>
      <Acct><Id><IBAN>{IBAN}</IBAN></Id></Acct>
      <Ntry>
        <Amt Ccy="EUR">119.00</Amt>
        <CdtDbtInd>DBIT</CdtDbtInd>
        <BookgDt><Dt>2026-04-10</Dt></BookgDt>
        <ValDt><Dt>2026-04-10</Dt></ValDt>
        <NtryDtls><TxDtls>
          <Refs><EndToEndId>E2E-1</EndToEndId></Refs>
          <RltdPties><Cdtr><Nm>Netzwerk GmbH</Nm></Cdtr></RltdPties>
          <RmtInf><Ustrd>Rechnung RE-LIEF-2001</Ustrd></RmtInf>
        </TxDtls></NtryDtls>
      </Ntry>
      <Ntry>
        <Amt Ccy="EUR">42.00</Amt>
        <CdtDbtInd>DBIT</CdtDbtInd>
        <BookgDt><Dt>2026-04-11</Dt></BookgDt>
        <ValDt><Dt>2026-04-11</Dt></ValDt>
        <NtryDtls><TxDtls>
          <RltdPties><Cdtr><Nm>Stadtwerke</Nm></Cdtr></RltdPties>
          <RmtInf><Ustrd>Abschlag Strom</Ustrd></RmtInf>
        </TxDtls></NtryDtls>
      </Ntry>
      <Ntry>
        <Amt Ccy="EUR">10.00</Amt>
        <CdtDbtInd>CRDT</CdtDbtInd>
        <BookgDt><Dt>2026-04-12</Dt></BookgDt>
        <ValDt><Dt>2026-04-12</Dt></ValDt>
        <NtryDtls><TxDtls>
          <RltdPties><Dbtr><Nm>Hausbank</Nm></Dbtr></RltdPties>
          <RmtInf><Ustrd>Zinsen Q1</Ustrd></RmtInf>
        </TxDtls></NtryDtls>
      </Ntry>
    </Stmt>
  </BkToCstmrStmt>
</Document>`

const ownIBAN = "DE89370400440532013000"

type umsatz struct {
	ID          uint   `json:"id"`
	Betrag      string `json:"betrag"`
	Status      string `json:"status"`
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

	statement := h.inboxFile("auszug.xml", []byte(strings.ReplaceAll(camtTemplate, "{IBAN}", ownIBAN)))
	// Das Bankkonto richtet nur der Mensch ein.
	e := a.fail("kontoauszug_importieren", a.args(map[string]any{"pfad": statement}), CodeAppOnly)
	if e.Werkzeug != "nur_app" {
		t.Errorf("unbekanntes Bankkonto: %+v", e)
	}
	h.ui(func(ctx context.Context, s *core.Services) error {
		return s.Bank.ConfigureAccounts(ctx, []domain.BankAccount{{IBAN: ownIBAN, Name: "Geschäftskonto", LedgerAccount: "1800", Currency: "EUR"}})
	})
	var imported struct {
		BelegID  uint     `json:"beleg_id"`
		Umsaetze int      `json:"umsaetze"`
		Neu      int      `json:"neu"`
		Konten   []string `json:"konten"`
	}
	a.ok("kontoauszug_importieren", a.args(map[string]any{"pfad": statement}), &imported)
	if imported.Umsaetze != 3 || imported.Neu != 3 || imported.BelegID == 0 || len(imported.Konten) != 1 || imported.Konten[0] != "1800" {
		t.Fatalf("Import: %+v", imported)
	}
	a.ok("kontoauszug_importieren", a.args(map[string]any{"pfad": statement}), &imported)
	if imported.Neu != 0 || imported.Umsaetze != 3 {
		t.Errorf("zweiter Import desselben Auszugs: %+v", imported)
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
		payment.Vorschlaege[0].Posten[0].BuchungID != payable {
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
	if len(list.Umsaetze) != 3 {
		t.Errorf("alle Umsätze: %+v", list)
	}
}
