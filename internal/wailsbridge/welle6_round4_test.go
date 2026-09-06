package wailsbridge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Eine falsch getippte Bestätigung darf keinen Archivexport hinterlassen.
//
// Der Archivexport schreibt einen vollständigen Abzug der Buchführung eines
// Jahres auf die Platte. Lief er, bevor die Bestätigung geprüft wurde, blieb
// dieser Abzug nach dem abgebrochenen Vorgang liegen — ungefragt, unbemerkt und
// mit allen Belegdateien darin.
func TestArchiveIsNotWrittenWhenTheConfirmationIsWrong(t *testing.T) {
	b := wiredBridge(t)
	ctx := context.Background()

	// Eine Buchung in einem Jahr, dessen Frist längst abgelaufen ist — an der
	// Festschreibungsprüfung vorbei, weil hier die Reihenfolge geprüft wird und
	// nicht der Buchungsweg.
	date := "2010-06-01"
	entry := &domain.JournalEntry{
		FiscalYear: 2010, BookingDate: date, DocumentDate: date,
		ServiceDateFrom: date, ServiceDateTo: date,
		Description: "Altbuchung", Source: domain.EntrySourceManual, TaxTreatment: domain.TaxTreatmentNotTaxable,
		Kind: domain.EntryKindNormal, Currency: "EUR", ExchangeRateMicros: 1_000_000,
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: "6815", Amount: 10000},
			{Position: 2, Side: domain.SideCredit, Account: "1800", Amount: 10000},
		},
	}
	if err := b.journalRepo.Append(ctx, entry, accounting.NewHashChain().CalculateHash); err != nil {
		t.Fatalf("die Altbuchung konnte nicht angelegt werden: %v", err)
	}

	archiveDir := filepath.Join(b.dataDir, "archiv", "2010")
	if _, err := b.ArchiveAndDeleteFiscalYear(2010, "ja"); err == nil {
		t.Fatal("eine falsche Bestätigung muss den Vorgang aufhalten")
	} else if !strings.Contains(err.Error(), "Jahreszahl") {
		t.Errorf("die Meldung muss die Bestätigung erklären: %v", err)
	}
	if _, err := os.Stat(archiveDir); !os.IsNotExist(err) {
		t.Errorf("nach der falschen Bestätigung liegt ein Archivexport unter %s", archiveDir)
	}

	// Und die Buchung steht noch: gelöscht wurde nichts.
	remaining, err := b.journalRepo.FindAll(ctx, 2010)
	if err != nil {
		t.Fatalf("Buchungen lesen: %v", err)
	}
	if len(remaining) != 1 {
		t.Errorf("%d Buchungen im Jahr 2010, erwartet 1", len(remaining))
	}

	// Mit der richtigen Bestätigung läuft der Vorgang durch — und dann liegt
	// das Archiv auch da.
	result, err := b.ArchiveAndDeleteFiscalYear(2010, "2010")
	if err != nil {
		t.Fatalf("nach Fristablauf muss die Löschung durchgehen: %v", err)
	}
	if _, err := os.Stat(result.ArchivePath); err != nil {
		t.Errorf("das Archiv fehlt unter %s: %v", result.ArchivePath, err)
	}
}

// Die Antworten der Nachweisseite tragen leere Listen, keine `null`. Sie werden
// im Regelfall gelesen, bevor irgendetwas erfasst wurde.
func TestNachweiseAnswersCarryEmptyListsNotNull(t *testing.T) {
	b := wiredBridge(t)

	overview, err := b.GetRetentionOverview(0)
	if err != nil {
		t.Fatalf("Fristenübersicht: %v", err)
	}
	if overview.Years == nil || overview.Concept == nil {
		t.Errorf("die Fristenübersicht trägt eine nicht belegte Liste: %+v", overview)
	}

	chain, err := b.VerifyAuditChain()
	if err != nil {
		t.Fatalf("Kettenprüfung: %v", err)
	}
	if chain.Breaks == nil {
		t.Error("die Kettenprüfung liefert `null` statt einer leeren Bruchliste")
	}

	aging, err := b.GetOpenItemsAging("2026-06-30")
	if err != nil {
		t.Fatalf("Altersstruktur: %v", err)
	}
	if aging.Sides == nil {
		t.Fatal("die Altersstruktur liefert `null` statt einer leeren Seitenliste")
	}
	for _, side := range aging.Sides {
		if side.Buckets == nil || side.Maturities == nil {
			t.Errorf("die Seite %q trägt eine nicht belegte Liste", side.Side)
		}
	}

	for label, load := range map[string]func() (int, error){
		"Aussetzungen": func() (int, error) {
			rows, err := b.GetRetentionHolds()
			if rows == nil {
				return 0, fmt.Errorf("`null` statt einer leeren Liste")
			}
			return len(rows), err
		},
		"abgelaufene Objekte": func() (int, error) {
			rows, err := b.GetExpiredObjects()
			if rows == nil {
				return 0, fmt.Errorf("`null` statt einer leeren Liste")
			}
			return len(rows), err
		},
		"Schemaänderungen": func() (int, error) {
			rows, err := b.GetSchemaMigrations()
			if rows == nil {
				return 0, fmt.Errorf("`null` statt einer leeren Liste")
			}
			return len(rows), err
		},
		"Datenübernahmen": func() (int, error) {
			rows, err := b.GetMigrationRecords()
			if rows == nil {
				return 0, fmt.Errorf("`null` statt einer leeren Liste")
			}
			return len(rows), err
		},
		"Fassungen der Verfahrensdokumentation": func() (int, error) {
			rows, err := b.GetProcedureDocumentations()
			if rows == nil {
				return 0, fmt.Errorf("`null` statt einer leeren Liste")
			}
			return len(rows), err
		},
	} {
		if _, err := load(); err != nil {
			t.Errorf("%s: %v", label, err)
		}
	}
}
