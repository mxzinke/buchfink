package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Der Kontoauszug als Beleg.
//
// Die importierte Zeile ist nur, was Buchfink aus der Datei gelesen hat.
// Aufzubewahren ist das empfangene Dokument selbst (§ 147 Abs. 1 Nr. 4 AO,
// GoBD Rz. 130 f.) — deshalb wird die CAMT-Datei abgelegt, bevor sie geparst
// wird.

const sampleCAMT = `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
  <BkToCstmrStmt>
    <Stmt>
      <Acct><Id><IBAN>DE89370400440532013000</IBAN></Id></Acct>
      <Ntry>
        <Amt Ccy="EUR">1190.00</Amt>
        <CdtDbtInd>CRDT</CdtDbtInd>
        <BookgDt><Dt>2026-04-10</Dt></BookgDt>
        <ValDt><Dt>2026-04-10</Dt></ValDt>
        <NtryDtls><TxDtls>
          <Refs><EndToEndId>E2E-1</EndToEndId></Refs>
          <RltdPties><Dbtr><Nm>Kunde Alpha</Nm></Dbtr></RltdPties>
          <RmtInf><Ustrd>Rechnung RE-2026-001</Ustrd></RmtInf>
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
    </Stmt>
  </BkToCstmrStmt>
</Document>`

func (e *testEnv) banking(t *testing.T) *BankService {
	t.Helper()
	svc := NewBankService(
		repository.NewBankRepository(e.db), e.journal, repository.NewAuditRepository(e.db))
	svc.SetReceiptService(e.receipts)
	return svc
}

// Der Import legt die Datei als Beleg ab und verknüpft jeden Umsatz mit ihm.
func TestImportCAMT053FileArchivesTheStatement(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	path := filepath.Join(t.TempDir(), "auszug-april.xml")
	if err := os.WriteFile(path, []byte(sampleCAMT), 0o600); err != nil {
		t.Fatalf("Testdatei: %v", err)
	}

	inserted, err := env.banking(t).ImportCAMT053File(ctx, path, "1800")
	if err != nil {
		t.Fatalf("Kontoauszug importieren: %v", err)
	}
	if inserted != 2 {
		t.Fatalf("erwartet 2 importierte Umsätze, erhalten %d", inserted)
	}

	receipts, err := env.receiptRepo.FindAll(ctx, 0)
	if err != nil {
		t.Fatalf("Belege lesen: %v", err)
	}
	if len(receipts) != 1 {
		t.Fatalf("erwartet einen abgelegten Kontoauszug, erhalten %d Belege", len(receipts))
	}
	statement := receipts[0]
	if statement.Kind != domain.ReceiptKindStatement {
		t.Errorf("Belegart %q, erwartet %q", statement.Kind, domain.ReceiptKindStatement)
	}
	if statement.Direction != domain.DirectionIncoming {
		t.Errorf("Richtung %q, erwartet %q", statement.Direction, domain.DirectionIncoming)
	}
	if statement.ReceivedVia != domain.ReceivedViaUpload {
		t.Errorf("Eingangsweg %q, erwartet %q", statement.ReceivedVia, domain.ReceivedViaUpload)
	}
	if len(statement.Files) != 1 || statement.Files[0].FileName != "auszug-april.xml" {
		t.Fatalf("der Originaldateiname wurde nicht übernommen: %+v", statement.Files)
	}
	if statement.ReceiptNumber == "" {
		t.Error("der Kontoauszug hat keine Belegnummer bekommen")
	}

	// Die Datei liegt im Belegspeicher und stimmt mit ihrer Prüfsumme überein.
	if err := env.store.Verify(statement.Files[0].StoredPath, statement.Files[0].SHA256); err != nil {
		t.Errorf("die abgelegte Auszugsdatei ist nicht unversehrt: %v", err)
	}

	transactions, err := repository.NewBankRepository(env.db).FindAll(ctx, 0)
	if err != nil {
		t.Fatalf("Bankumsätze lesen: %v", err)
	}
	if len(transactions) != 2 {
		t.Fatalf("erwartet 2 Bankumsätze, erhalten %d", len(transactions))
	}
	for _, tx := range transactions {
		if tx.StatementReceiptID == nil || *tx.StatementReceiptID != statement.ID {
			t.Errorf("der Umsatz vom %s verweist nicht auf den abgelegten Auszug", tx.BookingDate)
		}
	}
}

// Ein Kontoauszug ist ein Beleg ohne Buchungspflicht. Der Prüflauf darf ihn
// nicht als „abgelegt, aber nicht gebucht" melden — sonst blockierte jeder
// archivierte Auszug die Festschreibung.
func TestUnbookedCheckSkipsBankStatements(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	path := filepath.Join(t.TempDir(), "auszug-april.xml")
	if err := os.WriteFile(path, []byte(sampleCAMT), 0o600); err != nil {
		t.Fatalf("Testdatei: %v", err)
	}
	if _, err := env.banking(t).ImportCAMT053File(ctx, path, "1800"); err != nil {
		t.Fatalf("Kontoauszug importieren: %v", err)
	}
	// Ein gewöhnlicher Beleg daneben, damit der Test nicht schon deshalb
	// besteht, weil der Prüflauf gar nichts findet.
	invoice := env.fileIncoming(t, "rechnung.pdf")

	run, err := env.checks(t).Run(ctx, CheckRequest{CutoffDate: "2026-12-31", PeriodType: "year"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}

	var unbooked []domain.CheckFinding
	for _, f := range run.Findings {
		if f.Rule == domain.CheckRuleReceiptUnbooked {
			unbooked = append(unbooked, f)
		}
	}
	if len(unbooked) != 1 {
		t.Fatalf("erwartet genau einen ungebuchten Beleg (die Rechnung), gemeldet %d: %+v",
			len(unbooked), unbooked)
	}
	if unbooked[0].ObjectName != invoice.ReceiptNumber {
		t.Errorf("gemeldet wurde %q, erwartet die Rechnung %q",
			unbooked[0].ObjectName, invoice.ReceiptNumber)
	}
}

// Ausgenommen ist allein der Kontoauszug. Ein Beleg der Art „Sonstiges" bleibt
// buchungspflichtig: das ist die Art, die jemand wählt, der die richtige nicht
// findet — eine dort abgelegte Rechnung darf nicht stillschweigend aus der
// Aufsicht fallen.
func TestUnbookedCheckStillReportsOtherReceipts(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt := env.fileIncoming(t, "vertrag.pdf")
	if err := env.db.Model(&domain.Receipt{}).Where("id = ?", receipt.ID).
		Update("kind", domain.ReceiptKindOther).Error; err != nil {
		t.Fatalf("Belegart setzen: %v", err)
	}

	run, err := env.checks(t).Run(ctx, CheckRequest{CutoffDate: "2026-12-31", PeriodType: "year"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}

	found := false
	for _, f := range run.Findings {
		if f.Rule == domain.CheckRuleReceiptUnbooked && f.ObjectName == receipt.ReceiptNumber {
			found = true
		}
	}
	if !found {
		t.Errorf("der Beleg der Art %q wurde nicht als ungebucht gemeldet", domain.ReceiptKindOther)
	}
	if !domain.ReceiptKindOther.RequiresBooking() {
		t.Error("die Art „Sonstiges“ gilt als nicht buchungspflichtig — ausgenommen ist nur der Kontoauszug")
	}
	if domain.ReceiptKindStatement.RequiresBooking() {
		t.Error("der Kontoauszug gilt als buchungspflichtig")
	}
}

// Der alte Weg über den Dateiinhalt bleibt nutzbar — und legt dann eben keine
// Datei ab. Ohne diese Prüfung könnte die Umstellung ihn stillschweigend
// zerbrechen.
func TestImportCAMT053FromContentStillWorks(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	inserted, err := env.banking(t).ImportCAMT053(ctx, strings.NewReader(sampleCAMT), "1800")
	if err != nil {
		t.Fatalf("Import über den Inhalt: %v", err)
	}
	if inserted != 2 {
		t.Errorf("erwartet 2 importierte Umsätze, erhalten %d", inserted)
	}

	receipts, err := env.receiptRepo.FindAll(ctx, 0)
	if err != nil {
		t.Fatalf("Belege lesen: %v", err)
	}
	if len(receipts) != 0 {
		t.Errorf("der Import über den Inhalt kann keine Datei ablegen, hat aber %d Belege erzeugt", len(receipts))
	}
}

// Ein zweites Mal importierter Auszug legt keinen zweiten Beleg an: der
// Kontoauszug ist ein Beleg, nicht einer je Importlauf.
func TestImportingTheSameStatementTwiceKeepsOneReceipt(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "auszug.xml")
	if err := os.WriteFile(path, []byte(sampleCAMT), 0o644); err != nil {
		t.Fatal(err)
	}
	banking := env.banking(t)
	if _, err := banking.ImportCAMT053File(ctx, path, "1800"); err != nil {
		t.Fatalf("erster Import: %v", err)
	}
	inserted, err := banking.ImportCAMT053File(ctx, path, "1800")
	if err != nil {
		t.Fatalf("zweiter Import: %v", err)
	}
	if inserted != 0 {
		t.Errorf("der zweite Import hat %d Umsätze eingefügt, erwartet 0", inserted)
	}
	receipts, err := env.receipts.List(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	var statements int
	for _, r := range receipts {
		if r.Kind == domain.ReceiptKindStatement {
			statements++
		}
	}
	if statements != 1 {
		t.Errorf("es gibt %d Kontoauszug-Belege, erwartet 1", statements)
	}
}
