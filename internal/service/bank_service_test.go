package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestBankService_ImportAndReconcile(t *testing.T) {
	ctx := context.Background()
	accSvc, bankSvc, _, _, _ := setupTestServices(t)

	camtXML := `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
  <BkToCstmrStmt>
    <Stmt>
      <Acct><Id><IBAN>DE89370400440532013000</IBAN></Id></Acct>
      <Ntry>
        <Amt Ccy="EUR">2380.00</Amt>
        <CdtDbtInd>CRDT</CdtDbtInd>
        <BookgDt><Dt>2024-05-10</Dt></BookgDt>
        <ValDt><Dt>2024-05-10</Dt></ValDt>
        <NtryDtls><TxDtls>
          <Refs><EndToEndId>E2E-TEST-001</EndToEndId></Refs>
          <RltdPties>
            <Dbtr><Nm>Kunde Beta GmbH</Nm></Dbtr>
            <DbtrAcct><Id><IBAN>DE12345678901234567890</IBAN></Id></DbtrAcct>
          </RltdPties>
          <RmtInf><Ustrd>Rechnung RE-2024-099 Softwareprojekt</Ustrd></RmtInf>
        </TxDtls></NtryDtls>
      </Ntry>
    </Stmt>
  </BkToCstmrStmt>
</Document>`

	// 1. Import CAMT.053
	imported, err := bankSvc.ImportCAMT053(ctx, strings.NewReader(camtXML))
	if err != nil || imported != 1 {
		t.Fatalf("expected 1 imported transaction, got %d (err: %v)", imported, err)
	}

	txs, err := bankSvc.GetTransactions(ctx)
	if err != nil || len(txs) != 1 {
		t.Fatalf("expected 1 bank transaction in db, got %d", len(txs))
	}
	bankTx := txs[0]
	if bankTx.MatchStatus != domain.MatchStatusUnmatched {
		t.Fatalf("expected match status unmatched, got %s", bankTx.MatchStatus)
	}

	// 2. Automated Reconciliation: Match and Book
	booking, err := bankSvc.MatchAndBook(
		ctx,
		bankTx.ID,
		"1800", // Soll: Bank
		"4400", // Haben: Erlöse 19%
		"RE-2024-099",
		"Zahlungseingang RE-2024-099",
	)
	if err != nil {
		t.Fatalf("failed to match and book: %v", err)
	}
	if booking.Amount != 2380.00 || booking.DebitAccount != "1800" || booking.CreditAccount != "4400" {
		t.Fatalf("unexpected booking created: %+v", booking)
	}

	// 3. Check Bank Transaction updated status
	txsAfter, _ := bankSvc.GetTransactions(ctx)
	if txsAfter[0].MatchStatus != domain.MatchStatusMatched {
		t.Fatalf("expected matched status, got %s", txsAfter[0].MatchStatus)
	}
	if txsAfter[0].MatchedBookingID == nil || *txsAfter[0].MatchedBookingID != booking.ID {
		t.Fatalf("expected matched booking ID %d, got %v", booking.ID, txsAfter[0].MatchedBookingID)
	}

	// 4. Check Hash Chain in AccountingService
	integrity, err := accSvc.VerifyIntegrity(ctx)
	if err != nil || !integrity.IsValid {
		t.Fatalf("expected valid integrity check, got %+v", integrity)
	}
}
