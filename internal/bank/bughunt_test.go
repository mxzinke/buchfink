package bank_test

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/bank"
)

func bankEntry(date, amount, direction string) string {
	return `<Ntry><Amt Ccy="EUR">` + amount + `</Amt><CdtDbtInd>` + direction + `</CdtDbtInd><BookgDt><Dt>` + date + `</Dt></BookgDt><ValDt><Dt>` + date + `</Dt></ValDt></Ntry>`
}

func bankStatement(iban, entry string) string {
	return `<Stmt><Acct><Id><IBAN>` + iban + `</IBAN></Id></Acct>` + entry + `</Stmt>`
}

func TestBugHuntCAMTKeepsAccountOfEachStatement(t *testing.T) {
	xml := `<Document><BkToCstmrStmt>` + bankStatement("DE89370400440532013000", bankEntry("2026-09-01", "100.00", "CRDT")) + bankStatement("DE02120300000000202051", bankEntry("2026-09-02", "200.00", "CRDT")) + `</BkToCstmrStmt></Document>`
	txs, err := bank.ParseCAMT053(strings.NewReader(xml))
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 2 {
		t.Fatalf("expected two payments, got %d", len(txs))
	}
	if txs[0].AccountIBAN != "DE89370400440532013000" || txs[1].AccountIBAN != "DE02120300000000202051" {
		t.Errorf("statement accounts were mixed: %q, %q", txs[0].AccountIBAN, txs[1].AccountIBAN)
	}
}

func TestBugHuntCAMTRejectsInvalidDateAndDirection(t *testing.T) {
	for _, entry := range []string{bankEntry("2026-02-31", "100.00", "CRDT"), bankEntry("2026-09-01", "100.00", "SIDEWAYS"), bankEntry("2026-09-01", "-100.00", "DBIT")} {
		xml := `<Document><BkToCstmrStmt>` + bankStatement("DE89370400440532013000", entry) + `</BkToCstmrStmt></Document>`
		if txs, err := bank.ParseCAMT053(strings.NewReader(xml)); err == nil {
			t.Errorf("invalid entry accepted: %+v", txs)
		}
	}
}
