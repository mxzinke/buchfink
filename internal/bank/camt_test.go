package bank_test

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/bank"
)

func TestParseCAMT053(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
  <BkToCstmrStmt>
    <Stmt>
      <Acct>
        <Id><IBAN>DE89370400440532013000</IBAN></Id>
      </Acct>
      <Ntry>
        <Amt Ccy="EUR">1190.00</Amt>
        <CdtDbtInd>CRDT</CdtDbtInd>
        <BookgDt><Dt>2024-04-10</Dt></BookgDt>
        <ValDt><Dt>2024-04-10</Dt></ValDt>
        <NtryDtls>
          <TxDtls>
            <Refs><EndToEndId>E2E-12345</EndToEndId></Refs>
            <RltdPties>
              <Dbtr><Nm>Kunde Alpha</Nm></Dbtr>
              <DbtrAcct><Id><IBAN>DE12345678901234567890</IBAN></Id></DbtrAcct>
            </RltdPties>
            <RmtInf><Ustrd>Rechnung RE-2024-001 Beratung</Ustrd></RmtInf>
          </TxDtls>
        </NtryDtls>
      </Ntry>
    </Stmt>
  </BkToCstmrStmt>
</Document>`

	txs, err := bank.ParseCAMT053(strings.NewReader(xmlData))
	if err != nil {
		t.Fatalf("unexpected error parsing CAMT.053: %v", err)
	}

	if len(txs) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txs))
	}

	tx := txs[0]
	if tx.Amount != 1190.00 {
		t.Fatalf("expected amount 1190.00, got %f", tx.Amount)
	}
	if tx.CounterpartyName != "Kunde Alpha" {
		t.Fatalf("expected CounterpartyName 'Kunde Alpha', got '%s'", tx.CounterpartyName)
	}
	if tx.SuggestedAccount != "4400" {
		t.Fatalf("expected suggested revenue account 4400, got %s", tx.SuggestedAccount)
	}
}
