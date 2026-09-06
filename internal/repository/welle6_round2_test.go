package repository

import (
	"context"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Ein Geschäftsjahr besteht aus mehr als Journal und Belegen. Bleiben
// Rechnungen, Bankumsätze und Anlagenbewegungen stehen, hinterlässt „Jahr
// archivieren und löschen" eine Datei mit Verweisen ins Leere — eine bezahlte
// Rechnung lebte als offener Posten wieder auf, weil ihre Zahlungszuordnung auf
// eine Buchung zeigt, die es nicht mehr gibt.
func TestDeleteFiscalYearRemovesEveryObjectOfThatYear(t *testing.T) {
	db := welle6DB(t)
	journal := NewJournalRepository(db)
	retention := NewRetentionRepository(db)
	ctx := context.Background()

	current := welle6Entry(t, journal, "2026-01-10")
	old := &domain.JournalEntry{
		FiscalYear: 2015, BookingDate: "2015-01-10", DocumentDate: "2015-01-10",
		ServiceDateFrom: "2015-01-10", ServiceDateTo: "2015-01-10",
		Description: "Alte Buchung", Source: domain.EntrySourceManual,
		Kind: domain.EntryKindNormal, Currency: "EUR", ExchangeRateMicros: 1_000_000,
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: "1200", Amount: 119000},
			{Position: 2, Side: domain.SideCredit, Account: "4400", Amount: 119000},
		},
	}
	if err := journal.Append(ctx, old, accounting.NewHashChain().CalculateHash); err != nil {
		t.Fatalf("die alte Buchung konnte nicht angehängt werden: %v", err)
	}

	invoice := &domain.Invoice{
		FiscalYear: 2015, InvoiceNumber: "RE-2015-0001", Date: "2015-01-10",
		ServiceDateFrom: "2015-01-01", ServiceDateTo: "2015-01-10", DueDate: "2015-01-24",
		ContactID: 1, ContactName: "Kunde GmbH", NetAmount: 100000, TaxAmount: 19000,
		GrossAmount: 119000, JournalEntryID: &old.ID,
		Items: []domain.InvoiceItem{{
			Position: 1, Description: "Beratung", QuantityMilli: 1000, Unit: "Std",
			UnitPrice: 100000, TaxRate: domain.TaxRateStandard,
		}},
	}
	if err := db.Create(invoice).Error; err != nil {
		t.Fatalf("Rechnung anlegen: %v", err)
	}
	bankTx := &domain.BankTransaction{
		FiscalYear: 2015, AccountIBAN: "DE02120300000000202051", BookingDate: "2015-01-20",
		ValueDate: "2015-01-20", Amount: 119000, LedgerAccount: domain.AccountBank,
	}
	if err := db.Create(bankTx).Error; err != nil {
		t.Fatalf("Bankumsatz anlegen: %v", err)
	}
	asset := &domain.FixedAsset{
		InventoryNumber: "AN-0001", Name: "Serverschrank", Class: domain.AssetClassTangible,
		Account: "0640", AcquisitionDate: "2015-01-10", AcquisitionCost: 100000,
		Method: domain.DepreciationLinear, UsefulLifeMonths: 120,
		AcquisitionEntryID: &old.ID,
	}
	if err := db.Create(asset).Error; err != nil {
		t.Fatalf("Anlagegut anlegen: %v", err)
	}
	movement := &domain.AssetMovement{
		AssetID: asset.ID, Kind: domain.AssetMovementAcquisition, Date: "2015-01-10",
		FiscalYear: 2015, Account: "0640", CostAmount: 100000, JournalEntryID: &old.ID,
	}
	if err := db.Create(movement).Error; err != nil {
		t.Fatalf("Anlagenbewegung anlegen: %v", err)
	}
	allocation := &domain.PaymentAllocation{
		OpenItemEntryID: old.ID, PaymentEntryID: current.ID, ContactID: 1,
		SettledAmount: 119000, CashAmount: 119000,
	}
	if err := db.Create(allocation).Error; err != nil {
		t.Fatalf("Zahlungszuordnung anlegen: %v", err)
	}
	// Ein Beleg eines späteren Jahres, der auf eine Buchung des gelöschten
	// Jahres versiegelt ist: er bleibt, sein Verweis nicht.
	laterReceipt := &domain.Receipt{
		ReceiptNumber: "ER-2026-0001", FiscalYear: 2026,
		Direction: domain.DirectionIncoming, Kind: domain.ReceiptKindInvoice,
		Status: domain.ReceiptStatusSealed, ReceivedAt: "2026-01-10",
		JournalEntryID: &old.ID,
	}
	if err := db.Create(laterReceipt).Error; err != nil {
		t.Fatalf("Beleg des Folgejahres anlegen: %v", err)
	}

	counts, err := retention.CountObjects(ctx, 2015)
	if err != nil {
		t.Fatalf("die Zählung ist fehlgeschlagen: %v", err)
	}
	if counts.Invoices != 1 || counts.BankTransactions != 1 || counts.AssetMovements != 1 {
		t.Errorf("Zählung 2015: %d Rechnungen, %d Bankumsätze, %d Anlagenbewegungen — erwartet je 1",
			counts.Invoices, counts.BankTransactions, counts.AssetMovements)
	}

	deleted, _, err := retention.DeleteFiscalYear(ctx, 2015)
	if err != nil {
		t.Fatalf("das Löschen ist fehlgeschlagen: %v", err)
	}
	if deleted.Invoices != 1 || deleted.BankTransactions != 1 || deleted.AssetMovements != 1 {
		t.Errorf("das Protokoll nennt %d Rechnungen, %d Bankumsätze, %d Anlagenbewegungen",
			deleted.Invoices, deleted.BankTransactions, deleted.AssetMovements)
	}

	for _, check := range []struct {
		label string
		model any
		where string
		args  []any
	}{
		{"Rechnungen", &domain.Invoice{}, "fiscal_year = ?", []any{2015}},
		{"Rechnungszeilen", &domain.InvoiceItem{}, "invoice_id = ?", []any{invoice.ID}},
		{"Bankumsätze", &domain.BankTransaction{}, "fiscal_year = ?", []any{2015}},
		{"Anlagenbewegungen", &domain.AssetMovement{}, "fiscal_year = ?", []any{2015}},
		{"Zahlungszuordnungen", &domain.PaymentAllocation{}, "open_item_entry_id = ?", []any{old.ID}},
	} {
		var n int64
		if err := db.Model(check.model).Where(check.where, check.args...).Count(&n).Error; err != nil {
			t.Fatalf("%s zählen: %v", check.label, err)
		}
		if n != 0 {
			t.Errorf("%d %s des gelöschten Jahres stehen noch in der Datei", n, check.label)
		}
	}

	// Das Anlagegut überdauert das Jahr — seine Nutzungsdauer läuft weiter.
	// Sein Verweis auf die gelöschte Zugangsbuchung nicht.
	var keptAsset domain.FixedAsset
	if err := db.First(&keptAsset, asset.ID).Error; err != nil {
		t.Fatalf("das Anlagegut wurde mitgelöscht: %v", err)
	}
	if keptAsset.AcquisitionEntryID != nil {
		t.Errorf("das Anlagegut verweist noch auf die gelöschte Buchung %d", *keptAsset.AcquisitionEntryID)
	}
	var keptReceipt domain.Receipt
	if err := db.First(&keptReceipt, laterReceipt.ID).Error; err != nil {
		t.Fatalf("der Beleg des Folgejahres wurde mitgelöscht: %v", err)
	}
	if keptReceipt.JournalEntryID != nil {
		t.Errorf("der Beleg des Folgejahres verweist noch auf die gelöschte Buchung %d",
			*keptReceipt.JournalEntryID)
	}
	if deleted.ClearedReferences < 2 {
		t.Errorf("%d aufgelöste Verweise im Protokoll, erwartet mindestens 2 (Anlagegut und Beleg)",
			deleted.ClearedReferences)
	}

	// Das laufende Jahr bleibt vollständig.
	remaining, _ := journal.FindAll(ctx, 2026)
	if len(remaining) != 1 {
		t.Errorf("das laufende Jahr trägt %d Buchungen, erwartet 1", len(remaining))
	}
}

// --- Protokollkette und Transaktionen -------------------------------------

// Ein Protokolleintrag innerhalb einer laufenden Transaktion darf nicht auf eine
// Sperre warten, die ihrerseits auf diese Transaktion wartet.
//
// Der Fall: ein Weg außerhalb einer Transaktion hält die Anhänge-Sperre und
// wartet auf die Schreibsperre der Datei, die eine laufende Buchungstransaktion
// hält; die schreibt in sich noch ihren Protokolleintrag und wartet dafür auf
// die Anhänge-Sperre. Beide kämen erst über den busy_timeout wieder frei — nach
// fünf Sekunden, und einer der beiden Einträge ginge verloren, weil fast jede
// Aufrufstelle den Fehler des Protokollierens verwirft.
func TestAuditLogInsideATransactionDoesNotWaitForTheOutsideWriter(t *testing.T) {
	// Eine Datei und kein Speicher: die Schreibsperre von SQLite ist der
	// Gegenstand des Tests, und im Speicher gibt es sie so nicht.
	db, err := InitTenantDB(t.TempDir())
	if err != nil {
		t.Fatalf("Testdatenbank anlegen: %v", err)
	}
	audit := NewAuditRepository(db)
	runner := NewTxRunner(db)

	holdsWriteLock := make(chan struct{})
	inner := make(chan error, 1)
	outer := make(chan error, 1)

	go func() {
		inner <- runner.RunInTx(context.Background(), func(ctx context.Context) error {
			// Erst schreiben: damit hält die Transaktion die Schreibsperre.
			if err := audit.Log(ctx, domain.AuditActionCreate, "TEST", "1", "erste Schreiboperation"); err != nil {
				return err
			}
			close(holdsWriteLock)
			// Der Schreiber draußen soll in dieser Zeit die Sperre nehmen und
			// an der Schreibsperre hängen bleiben.
			time.Sleep(300 * time.Millisecond)
			return audit.Log(ctx, domain.AuditActionUpdate, "TEST", "1", "Eintrag aus der Transaktion")
		})
	}()

	<-holdsWriteLock
	go func() {
		outer <- audit.Log(context.Background(), domain.AuditActionCreate, "TEST", "2", "Eintrag von draußen")
	}()

	deadline := time.After(3 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case err := <-inner:
			if err != nil {
				t.Fatalf("der Eintrag aus der Transaktion ist gescheitert: %v", err)
			}
		case err := <-outer:
			if err != nil {
				t.Fatalf("der Eintrag von draußen ist gescheitert: %v", err)
			}
		case <-deadline:
			t.Fatal("die beiden Schreiber blockieren sich gegenseitig: nach drei Sekunden ist erst einer durch")
		}
	}

	entries, err := audit.FindAllAscending(context.Background())
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	written := 0
	for i := range entries {
		if entries[i].EntityType == "TEST" {
			written++
		}
	}
	if written != 3 {
		t.Errorf("%d der drei Protokolleinträge sind angekommen — einer ist verloren gegangen", written)
	}
	result := accounting.NewAuditChain().Verify(entries)
	if !result.IsValid {
		t.Errorf("die Protokollkette ist gebrochen: %s", result.Message)
	}
}
