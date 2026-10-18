package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die Steuerangaben der Handbuchung prüft der Buchungsweg selbst und nicht nur
// der Vorbau PostManualEntry.
func TestPostRejectsAManualEntryWithoutATaxCase(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	entry := simpleEntry("6815", domain.AccountBank, 10000)
	entry.TaxTreatment = ""
	if _, err := env.journal.Post(ctx, entry); err == nil {
		t.Fatal("eine Handbuchung ohne Steuerfall wird abgewiesen")
	} else if !strings.Contains(err.Error(), "Steuerfall") {
		t.Errorf("die Meldung nennt den Grund nicht: %v", err)
	}

	// Mit Steuerfall läuft dieselbe Buchung durch.
	ok := simpleEntry("6815", domain.AccountBank, 10000)
	if _, err := env.journal.Post(ctx, ok); err != nil {
		t.Fatalf("mit Steuerfall ist die Buchung buchbar: %v", err)
	}

	// Eine Buchung aus dem Abschluss bleibt ausgenommen: sie hat eine andere
	// Quelle (Entscheidung 1, „Ausnahme: Source ≠ manual").
	closing := simpleEntry("6815", domain.AccountBank, 10000)
	closing.Source = domain.EntrySourceClosing
	closing.TaxTreatment = ""
	if err := env.journal.ValidatePostable(ctx, closing); err != nil &&
		strings.Contains(err.Error(), "Steuerfall") {
		t.Errorf("die Steuerzeilenpflicht gilt der Handbuchung, nicht dem Abschluss: %v", err)
	}
}

// Scheitert die Buchung erst beim Schreiben, bleibt weder der Eigenbeleg noch
// seine Datei zurück (Entscheidung 1).
//
// Der Fall, den der frühere Test nicht traf: dort scheiterte schon die
// Vorprüfung, und der Beleg entstand gar nicht erst. Hier läuft die Anlage
// durch, und erst das Schreiben der Buchung schlägt fehl.
func TestFailedJournalWriteRollsBackTheSelfIssuedReceipt(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.withSelfIssued(t)

	// Der Journalspeicher weist das Schreiben ab. Alles davor — Vorprüfung,
	// Anlage des Eigenbelegs — läuft durch.
	failing := NewJournalService(
		failingJournalRepo{JournalRepository: env.journalRepo},
		repository.NewAccountRepository(env.db),
		env.contactRepo,
		repository.NewAuditRepository(env.db),
		repository.NewSettingsRepository(env.db),
		repository.NewTxRunner(env.db),
	)
	failing.SetReceiptRepo(env.receiptRepo)
	posting := NewPostingService(failing, env.contactRepo)
	posting.SetReceiptService(env.receipts)

	if _, err := posting.PostManualEntry(ctx, ManualEntryRequest{
		Entry:      *manualEntry(),
		SelfIssued: &SelfIssuedReceiptRequest{Reason: "Parkgebühr"},
	}); err == nil {
		t.Fatal("scheitert das Schreiben der Buchung, darf kein Vorgang entstehen")
	}

	receipts, err := env.receipts.List(ctx, 2026, "")
	if err != nil {
		t.Fatalf("Belege lesen: %v", err)
	}
	if len(receipts) != 0 {
		t.Errorf("%d Belege abgelegt — die gescheiterte Buchung darf keinen hinterlassen",
			len(receipts))
	}
	if files := storedFiles(t, env.dataDir); len(files) != 0 {
		t.Errorf("%d Dateien im Belegspeicher — das PDF des zurückgerollten Eigenbelegs bleibt "+
			"sonst als verwaiste Datei liegen: %v", len(files), files)
	}
}

// failingJournalRepo weist das Schreiben einer Buchung ab und reicht alles
// andere durch.
type failingJournalRepo struct {
	domain.JournalRepository
}

func (r failingJournalRepo) Append(
	ctx context.Context, entry *domain.JournalEntry, hash domain.EntryHashFunc,
) error {
	return errWriteRefused
}

var errWriteRefused = errors.New("das Journal nimmt die Buchung nicht an")

// storedFiles listet die Dateien des Belegspeichers unter dem Datenordner.
func storedFiles(t *testing.T, dataDir string) []string {
	t.Helper()
	out := make([]string, 0)
	err := filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			out = append(out, strings.TrimPrefix(path, dataDir))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Belegspeicher lesen: %v", err)
	}
	return out
}

// manualEntry ist ein Buchungssatz, wie ihn die Maske zusammenstellt: ein
// Aufwand gegen Bank, ohne Steuer, mit dem Steuerfall dazu.
func manualEntry() *domain.JournalEntry {
	return &domain.JournalEntry{
		BookingDate: "2026-03-01", DocumentDate: "2026-03-01",
		Description:  "Parkgebühr Kundentermin",
		TaxTreatment: domain.TaxTreatmentNotTaxable,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "6300", Amount: 500},
			{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 500},
		},
	}
}

// Die Handbuchung ohne Beleg wird abgewiesen (§ 146 Abs. 1 AO, GoBD Rz. 61).
func TestManualEntryWithoutReceiptIsRefused(t *testing.T) {
	env := newTestEnv(t)

	_, err := env.posting.PostManualEntry(context.Background(), ManualEntryRequest{
		Entry: *manualEntry(),
	})
	if err == nil {
		t.Fatal("eine Buchung ohne Beleg darf nicht entstehen")
	}
	if !strings.Contains(err.Error(), "Beleg") || !strings.Contains(err.Error(), "146") {
		t.Errorf("die Meldung nennt weder den Beleg noch die Vorschrift: %v", err)
	}
	entries, err := env.journalRepo.FindAll(context.Background(), 2026)
	if err != nil {
		t.Fatalf("Journal lesen: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("%d Buchungen im Journal — die abgewiesene darf nichts hinterlassen", len(entries))
	}
}

// Mit einem Eigenbeleg geht dieselbe Buchung durch, und beide entstehen
// zusammen: die Buchung verweist auf den Beleg, der Beleg ist versiegelt.
func TestManualEntryCreatesItsSelfIssuedReceipt(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.withSelfIssued(t)

	entry, err := env.posting.PostManualEntry(ctx, ManualEntryRequest{
		Entry: *manualEntry(),
		SelfIssued: &SelfIssuedReceiptRequest{
			Reason: "Parkgebühr am Automaten, Quittung nicht ausgedruckt",
		},
	})
	if err != nil {
		t.Fatalf("Handbuchung mit Eigenbeleg: %v", err)
	}
	if entry.ReceiptID == nil {
		t.Fatal("die Buchung verweist auf keinen Beleg")
	}
	receipt, err := env.receipts.Get(ctx, *entry.ReceiptID)
	if err != nil {
		t.Fatalf("Beleg lesen: %v", err)
	}
	if receipt.Status != domain.ReceiptStatusSealed {
		t.Errorf("Belegstatus %q — der gebuchte Beleg ist versiegelt", receipt.Status)
	}
	if receipt.GrossAmount != 500 {
		t.Errorf("Betrag des Eigenbelegs %s € — erwartet den Bruttobetrag der Buchung", receipt.GrossAmount)
	}
	if entry.ReceiptHash != receipt.ReceiptHash {
		t.Error("die Buchung hat einen anderen Beleg-Hash als der Beleg")
	}
	if entry.DocumentNumber != receipt.ReceiptNumber {
		t.Errorf("Belegfeld %q — erwartet die Belegnummer %q",
			entry.DocumentNumber, receipt.ReceiptNumber)
	}
}

// Scheitert die Buchung, bleibt kein Eigenbeleg im Speicher zurück: er wäre
// genau der ungebuchte Beleg, den der Prüflauf meldet.
func TestFailedManualEntryLeavesNoSelfIssuedReceipt(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.withSelfIssued(t)

	broken := manualEntry()
	broken.Lines[1].Account = "9999999"
	_, err := env.posting.PostManualEntry(ctx, ManualEntryRequest{
		Entry:      *broken,
		SelfIssued: &SelfIssuedReceiptRequest{Reason: "Parkgebühr"},
	})
	if err == nil {
		t.Fatal("eine Buchung auf ein unbekanntes Konto darf nicht entstehen")
	}
	receipts, err := env.receipts.List(ctx, 2026, "")
	if err != nil {
		t.Fatalf("Belege lesen: %v", err)
	}
	if len(receipts) != 0 {
		t.Errorf("%d Belege abgelegt — die gescheiterte Buchung darf keinen hinterlassen",
			len(receipts))
	}
}

// Die Handbuchung hat den Steuerfall, und eine Zeile auf einem Steuerkonto
// hat Schlüssel und Bemessungsgrundlage.
func TestManualEntryRequiresItsTaxDetails(t *testing.T) {
	withoutTreatment := manualEntry()
	withoutTreatment.TaxTreatment = ""
	if err := ValidateManualTaxLines(withoutTreatment); err == nil {
		t.Error("eine Handbuchung ohne Steuerfall läuft an der Voranmeldung vorbei")
	} else if !strings.Contains(err.Error(), "Steuerfall") {
		t.Errorf("die Meldung nennt nicht, was fehlt: %v", err)
	}

	// Ein steuerpflichtiger Inlandsumsatz ohne Steuerzeile ist keiner.
	withoutTaxLine := manualEntry()
	withoutTaxLine.TaxTreatment = domain.TaxTreatmentDomestic
	if err := ValidateManualTaxLines(withoutTaxLine); err == nil {
		t.Error("ein steuerpflichtiger Inlandsumsatz ohne Steuerzeile darf nicht durchgehen")
	}

	// Die Steuerzeile ohne Bemessungsgrundlage meldet die Steuer ohne den
	// Umsatz darunter.
	withoutBase := manualEntry()
	withoutBase.TaxTreatment = domain.TaxTreatmentDomestic
	withoutBase.Lines = []domain.JournalLine{
		{Side: domain.SideDebit, Account: "6300", Amount: 10000},
		{Side: domain.SideDebit, Account: domain.AccountVorsteuer19, Amount: 1900, TaxKey: "VST19"},
		{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 11900},
	}
	err := ValidateManualTaxLines(withoutBase)
	if err == nil {
		t.Fatal("eine Steuerzeile ohne Bemessungsgrundlage darf nicht durchgehen")
	}
	if !strings.Contains(err.Error(), "Bemessungsgrundlage") {
		t.Errorf("die Meldung nennt nicht, was fehlt: %v", err)
	}

	// Vollständig geht sie durch.
	complete := manualEntry()
	complete.TaxTreatment = domain.TaxTreatmentDomestic
	complete.Lines = []domain.JournalLine{
		{Side: domain.SideDebit, Account: "6300", Amount: 10000},
		{Side: domain.SideDebit, Account: domain.AccountVorsteuer19, Amount: 1900,
			TaxKey: "VST19", TaxBase: 10000},
		{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 11900},
	}
	if err := ValidateManualTaxLines(complete); err != nil {
		t.Errorf("die vollständige Buchung muss durchgehen: %v", err)
	}
}
