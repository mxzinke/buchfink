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

func TestCAMTKeepsAccountOfEachStatement(t *testing.T) {
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

func TestCAMTRejectsInvalidDateAndDirection(t *testing.T) {
	for _, entry := range []string{bankEntry("2026-02-31", "100.00", "CRDT"), bankEntry("2026-09-01", "100.00", "SIDEWAYS"), bankEntry("2026-09-01", "-100.00", "DBIT")} {
		xml := `<Document><BkToCstmrStmt>` + bankStatement("DE89370400440532013000", entry) + `</BkToCstmrStmt></Document>`
		if txs, err := bank.ParseCAMT053(strings.NewReader(xml)); err == nil {
			t.Errorf("invalid entry accepted: %+v", txs)
		}
	}
}

func bankBalance(code, amount, direction, date string) string {
	return `<Bal><Tp><CdOrPrtry><Cd>` + code + `</Cd></CdOrPrtry></Tp><Amt Ccy="EUR">` + amount + `</Amt><CdtDbtInd>` +
		direction + `</CdtDbtInd><Dt><Dt>` + date + `</Dt></Dt></Bal>`
}

// Ein Auszug ist stimmig, wenn beide Salden dastehen und die Umsätze ihre
// Differenz ergeben; ein negativer Saldo steht mit DBIT.
func TestCAMTStatementBalances(t *testing.T) {
	entries := bankEntry("2026-09-01", "100.00", "CRDT") + bankEntry("2026-09-02", "30.00", "DBIT")
	cases := []struct {
		name       string
		balances   string
		consistent bool
	}{
		{"stimmig", bankBalance("PRCD", "50.00", "DBIT", "2026-08-31") + bankBalance("CLBD", "20.00", "CRDT", "2026-09-30"), true},
		{"Differenz falsch", bankBalance("OPBD", "50.00", "CRDT", "2026-09-01") + bankBalance("CLBD", "20.00", "CRDT", "2026-09-30"), false},
		{"ohne Endsaldo", bankBalance("OPBD", "50.00", "DBIT", "2026-09-01"), false},
		{"ohne Salden", "", false},
	}
	for _, tc := range cases {
		xml := `<Document><BkToCstmrStmt>` + bankStatement("DE89370400440532013000", tc.balances+entries) + `</BkToCstmrStmt></Document>`
		statements, err := bank.ParseCAMT053Statements(strings.NewReader(xml))
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if len(statements) != 1 || len(statements[0].Transactions) != 2 || statements[0].Consistent() != tc.consistent {
			t.Errorf("%s: %+v", tc.name, statements)
		}
	}
}
