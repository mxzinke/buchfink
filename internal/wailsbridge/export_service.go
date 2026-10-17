package wailsbridge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/core"
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

// Die Exporte laufen als Lesevorgang: ohne Schreibsperre, auf einem in sich
// stimmigen Stand der Datei. Ein Export über ein ganzes Geschäftsjahr dauert
// Sekunden bis Minuten, und währenddessen bleibt das Buchen möglich.

// ExportZ3 schreibt die Datenüberlassung eines Geschäftsjahres in einen Ordner.
func (b *BuchfinkBridge) ExportZ3(year int, targetDir string) (*export.Result, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*export.Result, error) {
		return s.Export.ExportZ3(ctx, year, targetDir)
	})
}

// ExportArchive schreibt die Datenüberlassung samt Belegdateien.
func (b *BuchfinkBridge) ExportArchive(year int, targetDir string) (*export.Result, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*export.Result, error) {
		return s.Export.ExportArchive(ctx, year, targetDir)
	})
}

// ExportAuditPackage schreibt das Prüferpaket: Archiv, Integritätsnachweis und
// Verfahrensdokumentation in einem Ordner.
func (b *BuchfinkBridge) ExportAuditPackage(year int, targetDir string) (*export.Result, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*export.Result, error) {
		return s.Export.ExportAuditPackage(ctx, year, targetDir)
	})
}

// ExportJournalCSV schreibt das Journal eines Zeitraums als CSV und liefert den
// Pfad. Leere Grenzen bedeuten: keine Grenze. Der Speichern-Dialog steht vor
// dem Lesevorgang offen.
func (b *BuchfinkBridge) ExportJournalCSV(from, to string) (string, error) {
	name := fmt.Sprintf("buchfink-journal-%s-bis-%s.csv", orDefault(from, "anfang"), orDefault(to, "heute"))
	path, err := b.saveFileDialog("Journal als CSV speichern", name)
	if err != nil || path == "" {
		return "", err
	}
	return read(b, func(ctx context.Context, s *core.Services) (string, error) {
		return s.Export.ExportJournalCSV(ctx, from, to, path)
	})
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
	// Gelesen wird vor dem Dialog: er ist modal und steht offen, solange der
	// Anwender überlegt.
	content, err := read(b, func(ctx context.Context, s *core.Services) (*service.FileContent, error) {
		return s.Receipts.Content(ctx, receiptID, fileID)
	})
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
	if err := b.writeReceiptFileTo(receiptID, content, path); err != nil {
		return "", err
	}
	return path, nil
}

// writeReceiptFileTo schreibt die Belegdatei und protokolliert die Herausgabe
// (QUE-02 K2). Getrennt vom Dialog, weil die Herausgabe die Handlung ist, die
// ins Protokoll gehört.
func (b *BuchfinkBridge) writeReceiptFileTo(
	receiptID uint, content *service.FileContent, path string,
) error {
	if err := os.WriteFile(path, content.Data, 0o600); err != nil {
		return fmt.Errorf("%s konnte nicht geschrieben werden: %w", filepath.Base(path), err)
	}
	return b.logAccess("RECEIPT", strconv.FormatUint(uint64(receiptID), 10), fmt.Sprintf(
		"Belegdatei %s aus Beleg %d nach %s herausgegeben", content.FileName, receiptID, path))
}

// -------------------------------------------------------------
// INTEGRITÄT
// -------------------------------------------------------------

// VerifyReceiptFiles prüft jede Belegdatei gegen ihre Prüfsumme.
func (b *BuchfinkBridge) VerifyReceiptFiles() (*domain.FileCheckResult, error) {
	return readOr(b, func() (*domain.FileCheckResult, error) {
		return &domain.FileCheckResult{
			Issues:    make([]domain.FileCheckIssue, 0),
			IsValid:   true,
			Message:   "Die Belegablage ist noch nicht initialisiert.",
			CheckedAt: time.Now().UTC().Format(time.RFC3339),
		}, nil
	}, func(ctx context.Context, s *core.Services) (*domain.FileCheckResult, error) {
		return s.Receipts.VerifyReceiptFiles(ctx)
	})
}

// -------------------------------------------------------------
// STICHTAGSAUSWERTUNGEN
// -------------------------------------------------------------

// GetSuSaOverviewAt liefert die Summen- und Saldenliste zu einem Stichtag.
// Leerer Stichtag heißt: das ganze Geschäftsjahr.
func (b *BuchfinkBridge) GetSuSaOverviewAt(cutoff string) (*domain.SuSaOverview, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*domain.SuSaOverview, error) {
		return s.Accounting.GetSuSaOverviewAt(ctx, cutoff)
	})
}

// GetOpenItemsAt liefert die offenen Posten zu einem Stichtag.
func (b *BuchfinkBridge) GetOpenItemsAt(cutoff string) ([]domain.OpenItem, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.OpenItem, error) {
		return s.Payments.OpenItemsAt(ctx, cutoff)
	})
}

// GetPaymentAllocations liefert die Einzelposten einer Zahlungsbuchung.
func (b *BuchfinkBridge) GetPaymentAllocations(entryID uint) ([]domain.PaymentAllocationDetail, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.PaymentAllocationDetail, error) {
		return s.Payments.Allocations(ctx, entryID)
	})
}

// -------------------------------------------------------------
// BANKIMPORT ÜBER DEN DATEIPFAD
// -------------------------------------------------------------

// ImportCAMT053File legt die Kontoauszugsdatei als Beleg ab und importiert die
// Umsätze daraus. Sie bekommt den Pfad und nicht den Inhalt, damit die Datei
// selbst archiviert wird.
func (b *BuchfinkBridge) ImportCAMT053File(path, ledgerAccount string) (int, error) {
	return write(b, func(ctx context.Context, s *core.Services) (int, error) {
		return s.Bank.ImportCAMT053File(ctx, path, ledgerAccount)
	})
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
