package wailsbridge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/export"
	"github.com/buchfink/buchfink/internal/service"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Die Datenüberlassung nach § 147 Abs. 6 AO und die Stichtagsauswertungen.
//
// Alle drei Umfänge — Z3, Archiv, Prüferpaket — laufen über denselben Dienst
// und unterscheiden sich nur darin, was neben den Tabellen mitgeht. Die Bridge
// wählt den Ordner und reicht durch.

// programVersion ist die Fassung, die in jeden Export und jede Sicherung geht.
func programVersion() string { return buildinfo.Version }

// ExportZ3 schreibt die Datenüberlassung eines Geschäftsjahres in einen Ordner.
func (b *BuchfinkBridge) ExportZ3(year int, targetDir string) (*export.Result, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.exportSvc == nil {
		return nil, fmt.Errorf("der Export ist noch nicht initialisiert")
	}
	return b.exportSvc.ExportZ3(context.Background(), year, targetDir)
}

// ExportArchive schreibt die Datenüberlassung samt Belegdateien.
func (b *BuchfinkBridge) ExportArchive(year int, targetDir string) (*export.Result, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.exportSvc == nil {
		return nil, fmt.Errorf("der Export ist noch nicht initialisiert")
	}
	return b.exportSvc.ExportArchive(context.Background(), year, targetDir)
}

// ExportAuditPackage schreibt das Prüferpaket: Archiv, Integritätsnachweis und
// Verfahrensdokumentation in einem Ordner.
func (b *BuchfinkBridge) ExportAuditPackage(year int, targetDir string) (*export.Result, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.exportSvc == nil {
		return nil, fmt.Errorf("der Export ist noch nicht initialisiert")
	}
	return b.exportSvc.ExportAuditPackage(context.Background(), year, targetDir)
}

// ExportJournalCSV schreibt das Journal eines Zeitraums als CSV und liefert den
// Pfad. Leere Grenzen bedeuten: keine Grenze.
func (b *BuchfinkBridge) ExportJournalCSV(from, to string) (string, error) {
	// Erst den Pfad wählen, dann sperren. Der Speichern-Dialog ist modal und
	// steht offen, solange der Anwender überlegt; jede schreibende
	// Bridge-Methode wartete währenddessen auf die Sperre.
	name := fmt.Sprintf("buchfink-journal-%s-bis-%s.csv", orDefault(from, "anfang"), orDefault(to, "heute"))
	path, err := b.saveFileDialog("Journal als CSV speichern", name)
	if err != nil || path == "" {
		return "", err
	}

	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.exportSvc == nil {
		return "", fmt.Errorf("der Export ist noch nicht initialisiert")
	}
	return b.exportSvc.ExportJournalCSV(context.Background(), from, to, path)
}

// ExportKeyDirectory schreibt das Schlüsselverzeichnis als CSV.
func (b *BuchfinkBridge) ExportKeyDirectory() (string, error) {
	path, err := b.saveFileDialog("Schlüsselverzeichnis speichern", "buchfink-schluesselverzeichnis.csv")
	if err != nil || path == "" {
		return "", err
	}

	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.exportSvc == nil {
		return "", fmt.Errorf("der Export ist noch nicht initialisiert")
	}
	return b.exportSvc.ExportKeyDirectory(context.Background(), path)
}

// KeyDirectoryEntry ist eine Zeile des Schlüsselverzeichnisses für die Anzeige.
type KeyDirectoryEntry struct {
	Category    string `json:"category"`
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// GetKeyDirectory liefert das Schlüsselverzeichnis zur Anzeige.
//
// Dieselbe Tabelle, die auch exportiert wird: der Anwender sieht auf dem Schirm,
// was der Prüfer auf dem Datenträger bekommt.
func (b *BuchfinkBridge) GetKeyDirectory() ([]KeyDirectoryEntry, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	entries := make([]KeyDirectoryEntry, 0)
	if b.exportSvc == nil {
		return emptyList(entries, nil)
	}
	table, err := b.exportSvc.KeyDirectory()
	if err != nil {
		return emptyList(entries, err)
	}
	for _, row := range table.Rows {
		if len(row) < 4 {
			continue
		}
		entries = append(entries, KeyDirectoryEntry{
			Category: row[0], Key: row[1], Label: row[2], Description: row[3],
		})
	}
	return emptyList(entries, nil)
}

// SelectExportDirectoryDialog öffnet den Ordnerdialog für einen Export.
func (b *BuchfinkBridge) SelectExportDirectoryDialog(title string) (string, error) {
	if title == "" {
		title = "Zielordner für den Export wählen"
	}
	app := application.Get()
	if app == nil || app.Dialog == nil {
		return "", nil
	}
	return app.Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(true).
		SetTitle(title).
		PromptForSingleSelection()
}

// SelectSaveFileDialog öffnet den Speichern-Dialog und liefert den gewählten
// Pfad. Leer heißt: abgebrochen.
func (b *BuchfinkBridge) SelectSaveFileDialog(title, suggestedName string) (string, error) {
	return b.saveFileDialog(title, suggestedName)
}

// saveFileDialog kapselt den Speichern-Dialog. Ohne laufende Anwendung — im
// Test — liefert er einen leeren Pfad statt eines Fehlers.
func (b *BuchfinkBridge) saveFileDialog(title, suggestedName string) (string, error) {
	app := application.Get()
	if app == nil || app.Dialog == nil {
		return "", nil
	}
	// Der Speichern-Dialog kennt keinen Titel, sondern eine Meldung; sie steht
	// im Fenster an derselben Stelle.
	return app.Dialog.SaveFile().
		SetMessage(title).
		SetFilename(suggestedName).
		CanCreateDirectories(true).
		PromptForSingleSelection()
}

// SaveReceiptFileAs schreibt eine Belegdatei unter ihrem Originalnamen an einen
// vom Anwender gewählten Ort.
//
// Der Beleg bleibt dabei im Archiv: dies ist eine Kopie hinaus und kein Umzug.
// Die Prüfsumme wird beim Lesen geprüft — wer eine beschädigte Datei
// weitergibt, soll das erfahren.
func (b *BuchfinkBridge) SaveReceiptFileAs(receiptID, fileID uint) (string, error) {
	// Die Datei wird unter der Sperre gelesen, der Dialog steht ohne sie offen:
	// er ist modal, und solange er offen ist, käme sonst keine andere
	// Bridge-Methode mehr durch.
	content, err := func() (*service.FileContent, error) {
		b.mu.RLock()
		defer b.mu.RUnlock()
		if b.receiptSvc == nil {
			return nil, fmt.Errorf("die Belegablage ist noch nicht initialisiert")
		}
		return b.receiptSvc.Content(context.Background(), receiptID, fileID)
	}()
	if err != nil {
		return "", err
	}
	if !content.Intact {
		return "", fmt.Errorf(
			"die Datei %s stimmt nicht mehr mit ihrer Prüfsumme überein und wird deshalb nicht ausgegeben",
			content.FileName)
	}

	path, err := b.saveFileDialog("Belegdatei speichern", content.FileName)
	if err != nil || path == "" {
		return "", err
	}
	if err := os.WriteFile(path, content.Data, 0o600); err != nil {
		return "", fmt.Errorf("%s konnte nicht geschrieben werden: %w", filepath.Base(path), err)
	}
	return path, nil
}

// -------------------------------------------------------------
// INTEGRITÄT
// -------------------------------------------------------------

// VerifyReceiptFiles prüft jede Belegdatei und jedes Anlagendokument gegen
// seine Prüfsumme.
func (b *BuchfinkBridge) VerifyReceiptFiles() (*domain.FileCheckResult, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.receiptSvc == nil {
		return &domain.FileCheckResult{
			Issues:    make([]domain.FileCheckIssue, 0),
			IsValid:   true,
			Message:   "Die Belegablage ist noch nicht initialisiert.",
			CheckedAt: time.Now().Format("02.01.2006 15:04:05"),
		}, nil
	}
	return b.receiptSvc.VerifyReceiptFiles(context.Background())
}

// -------------------------------------------------------------
// STICHTAGSAUSWERTUNGEN
// -------------------------------------------------------------

// GetSuSaOverviewAt liefert die Summen- und Saldenliste zu einem Stichtag.
// Leerer Stichtag heißt: das ganze Geschäftsjahr.
func (b *BuchfinkBridge) GetSuSaOverviewAt(cutoff string) (*domain.SuSaOverview, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.accountingSvc.GetSuSaOverviewAt(context.Background(), cutoff)
}

// GetAccountLedgerRange liefert das Kontoblatt eines Zeitraums über alle
// Geschäftsjahre hinweg.
func (b *BuchfinkBridge) GetAccountLedgerRange(accountNumber, from, to string) (*domain.AccountLedger, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.accountingSvc.GetAccountLedgerRange(context.Background(), accountNumber, from, to)
}

// GetOpenItemsAt liefert die offenen Posten zu einem Stichtag.
func (b *BuchfinkBridge) GetOpenItemsAt(cutoff string) ([]domain.OpenItem, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.paymentSvc == nil {
		return make([]domain.OpenItem, 0), nil
	}
	items, err := b.paymentSvc.OpenItemsAt(context.Background(), cutoff)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = make([]domain.OpenItem, 0)
	}
	return emptyList(items, nil)
}

// GetPaymentAllocations liefert die Einzelposten einer Zahlungsbuchung.
func (b *BuchfinkBridge) GetPaymentAllocations(entryID uint) ([]domain.PaymentAllocationDetail, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.paymentSvc == nil {
		return make([]domain.PaymentAllocationDetail, 0), nil
	}
	return emptyList(b.paymentSvc.Allocations(context.Background(), entryID))
}

// -------------------------------------------------------------
// BANKIMPORT ÜBER DEN DATEIPFAD
// -------------------------------------------------------------

// ImportCAMT053File legt die Kontoauszugsdatei als Beleg ab und importiert die
// Umsätze daraus.
//
// Sie tritt an die Stelle von ImportCAMT053XML: die alte Methode bekam den
// Inhalt und konnte die Datei deshalb gar nicht archivieren.
func (b *BuchfinkBridge) ImportCAMT053File(path, ledgerAccount string) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return 0, err
	}
	if b.bankSvc == nil {
		return 0, fmt.Errorf("Bankimport ist noch nicht initialisiert")
	}
	return b.bankSvc.ImportCAMT053File(context.Background(), path, ledgerAccount)
}

// SelectStatementFileDialog öffnet den Dateidialog für eine CAMT.053-Datei.
func (b *BuchfinkBridge) SelectStatementFileDialog(title string) (string, error) {
	if title == "" {
		title = "Kontoauszug (CAMT.053) auswählen"
	}
	app := application.Get()
	if app == nil || app.Dialog == nil {
		return "", nil
	}
	return app.Dialog.OpenFile().
		CanChooseFiles(true).
		CanChooseDirectories(false).
		AddFilter("Kontoauszüge (*.xml)", "*.xml").
		SetTitle(title).
		PromptForSingleSelection()
}

func orDefault(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// integrityChecks bündelt die beiden Prüfläufe für den Export.
//
// Der Exportdienst soll weder den Journaldienst noch die Belegablage kennen —
// er braucht von beiden je eine Frage. Zwei Methoden statt zweier Dienste.
type integrityChecks struct {
	journal  *service.JournalService
	receipts *service.ReceiptService
}

func (c integrityChecks) VerifyIntegrity(ctx context.Context) (domain.IntegrityCheckResult, error) {
	if c.journal == nil {
		return domain.IntegrityCheckResult{IsValid: true, Message: "Keine Buchhaltung geöffnet."}, nil
	}
	return c.journal.VerifyIntegrity(ctx)
}

func (c integrityChecks) VerifyReceiptFiles(ctx context.Context) (*domain.FileCheckResult, error) {
	if c.receipts == nil {
		return &domain.FileCheckResult{
			Issues: make([]domain.FileCheckIssue, 0), IsValid: true,
			Message: "Keine Belegablage geöffnet.",
		}, nil
	}
	return c.receipts.VerifyReceiptFiles(ctx)
}
