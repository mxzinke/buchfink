package wailsbridge

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Die Bedienung der Welle 8: die laufende Buchhaltung ohne Lücken.
//
// Sie hängt an einem Satz: keine Buchung ohne Beleg. Daraus folgt der
// Eigenbeleg für die Fälle ohne Fremdbeleg, der Belegverweis in beide
// Richtungen, die Klärungsliste der beanstandeten Rechnungen und der Filter,
// mit dem sich eine Buchung im Journal überhaupt wiederfinden lässt.

// --- Handbuchung mit Beleg ------------------------------------------------

// PostManualEntry bucht einen von Hand erfassten Buchungssatz mit seinem Beleg.
//
// Entweder verweist die Buchung auf einen abgelegten Beleg (ReceiptID), oder
// die Angaben eines Eigenbelegs kommen mit (SelfIssued) — dann entstehen Beleg
// und Buchung in einer Transaktion.
func (b *BuchfinkBridge) PostManualEntry(req service.ManualEntryRequest) (*domain.JournalEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.postingSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.postingSvc.PostManualEntry(context.Background(), req)
}

// --- Eigenbeleg -----------------------------------------------------------

// CreateSelfIssuedReceipt erzeugt einen Eigenbeleg mit PDF und Kopfdaten und
// legt ihn im Belegspeicher ab.
func (b *BuchfinkBridge) CreateSelfIssuedReceipt(
	req service.SelfIssuedReceiptRequest,
) (*domain.Receipt, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.receiptSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.receiptSvc.CreateSelfIssued(context.Background(), req)
}

// --- Aufbewahrung ---------------------------------------------------------

// OverrideReceiptRetention verlängert die Aufbewahrungsfrist eines Belegs.
func (b *BuchfinkBridge) OverrideReceiptRetention(
	receiptID uint, class string, reason string,
) (*domain.Receipt, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.receiptSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.receiptSvc.OverrideRetention(
		context.Background(), receiptID, domain.RetentionClass(class), reason)
}

// GetRetentionRules liefert die Fristentabelle mit ihrer Quelle und ihrem
// Gültigkeitsstand — für die Anzeige in den Einstellungen.
func (b *BuchfinkBridge) GetRetentionRules() accounting.RetentionRules {
	return accounting.LoadedRetentionRules()
}

// --- Beleg und Buchung in beide Richtungen --------------------------------

// GetEntriesForReceipt liefert die Buchungen zu einem Beleg.
//
// Der Weg zurück: das Journal zeigt seit jeher den Beleg, der Beleg zeigte die
// Buchung nicht. Ein Prüfer, der von der Ablage ausgeht — und das ist der
// übliche Weg —, kam damit nicht ins Journal (GOB-02 K2, BEL-01 K3).
func (b *BuchfinkBridge) GetEntriesForReceipt(receiptID uint) ([]domain.JournalEntry, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.journalRepo == nil {
		return []domain.JournalEntry{}, nil
	}
	return emptyList(entriesForReceipt(context.Background(), b.journalRepo, b.currentYear, receiptID))
}

// entriesForReceipt sucht die Buchungen eines Belegs im aktiven Geschäftsjahr
// und im Jahr davor.
//
// Auch das Vorjahr, weil eine Buchung nicht im Geschäftsjahr des Belegs liegen
// muss: eine Dezemberrechnung wird im Januar gebucht, und ein Storno trägt das
// Datum seiner Erstellung. Wer nur das aktive Jahr durchsähe, zeigte am Beleg
// „keine Buchung", während sie einen Klick weiter im Journal steht.
func entriesForReceipt(
	ctx context.Context, repo domain.JournalRepository, year int, receiptID uint,
) ([]domain.JournalEntry, error) {
	out := make([]domain.JournalEntry, 0, 2)
	years := []int{year}
	if year > 0 {
		years = append(years, year+1, year-1)
	}
	seen := map[uint]bool{}
	for _, y := range years {
		entries, err := repo.FindAll(ctx, y)
		if err != nil {
			continue
		}
		for i := range entries {
			e := &entries[i]
			if e.ReceiptID == nil || *e.ReceiptID != receiptID || seen[e.ID] {
				continue
			}
			seen[e.ID] = true
			out = append(out, *e)
		}
	}
	return out, nil
}

// GetReceiptFindings liefert die Klärungsliste eines Belegs: die Befunde nach
// Formatfehler, Geschäftsregelfehler und Inhaltsfehler getrennt, je Befund mit
// Regel, Norm und Folge für den Vorsteuerabzug.
func (b *BuchfinkBridge) GetReceiptFindings(receiptID uint) (*domain.ReceiptFindings, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.receiptSvc == nil {
		empty := &domain.ReceiptFindings{}
		empty.EnsureLists()
		return empty, nil
	}
	return b.receiptSvc.Findings(context.Background(), receiptID)
}

// --- Eigene Konten --------------------------------------------------------

// CreateCustomAccount legt ein eigenes Konto im freien Bereich des SKR04 an.
func (b *BuchfinkBridge) CreateCustomAccount(
	req service.CustomAccountRequest,
) (*domain.Account, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.accountSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.accountSvc.CreateCustom(context.Background(), req)
}

// SetAccountBlocked sperrt ein eigenes Konto für neue Buchungen oder gibt es
// wieder frei.
func (b *BuchfinkBridge) SetAccountBlocked(
	number string, blocked bool, reason string,
) (*domain.Account, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.accountSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.accountSvc.SetBlocked(context.Background(), number, blocked, reason)
}

// GetCustomAccounts liefert die selbst angelegten Konten.
func (b *BuchfinkBridge) GetCustomAccounts() ([]domain.Account, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountSvc == nil {
		return []domain.Account{}, nil
	}
	return emptyList(b.accountSvc.CustomAccounts(context.Background()))
}

// GetStatementPositions liefert die Gliederungspositionen, unter denen ein
// eigenes Konto stehen darf.
func (b *BuchfinkBridge) GetStatementPositions() ([]domain.StatementPositionOption, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountSvc == nil {
		return []domain.StatementPositionOption{}, nil
	}
	return emptyList(b.accountSvc.AvailablePositions())
}

// --- Umsatzsteuer ---------------------------------------------------------

// GetVatPeriodProposal liefert den Voranmeldungszeitraum, der sich aus der
// Steuer des Vorjahres ergibt (§ 18 Abs. 2 UStG).
func (b *BuchfinkBridge) GetVatPeriodProposal(year int) (*service.VatPeriodProposal, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.vatReturnSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.vatReturnSvc.SuggestPeriodType(context.Background(), year)
}

// GetVatRatePeriods liefert die datierte Tabelle der Umsatzsteuersätze.
func (b *BuchfinkBridge) GetVatRatePeriods() []accounting.VatRatePeriod {
	return accounting.VatRatePeriods()
}

// --- Journalfilter --------------------------------------------------------

// GetFilteredJournal liefert die gefilterte Menge der Journalzeilen mit ihrer
// Summenzeile (PRF-01 K3).
func (b *BuchfinkBridge) GetFilteredJournal(
	filter accounting.JournalFilter,
) (*accounting.JournalFilterResult, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		empty := &accounting.JournalFilterResult{Rows: []accounting.JournalFilterRow{}}
		return empty, nil
	}
	return b.accountingSvc.FilterEntries(context.Background(), filter)
}

// GetFilteredJournalCSV liefert dieselbe Menge als CSV.
//
// Als Zeichenkette und nicht als Datei: wohin sie geschrieben wird, entscheidet
// der Dateidialog, und ein Dienst, der selbst schreibt, schriebe irgendwohin.
func (b *BuchfinkBridge) GetFilteredJournalCSV(filter accounting.JournalFilter) (string, error) {
	// Schreibsperre trotz Lesevorgangs: die Herausgabe schreibt einen
	// Protokolleintrag (QUE-02 K2).
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.accountingSvc == nil {
		return "", fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.accountingSvc.FilterEntriesCSV(context.Background(), filter)
}

// SaveFilteredJournalCSV schreibt die gefilterte Menge in eine gewählte Datei.
func (b *BuchfinkBridge) SaveFilteredJournalCSV(
	filter accounting.JournalFilter, path string,
) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// Schreibsperre trotz Ausgabe: die Herausgabe schreibt einen
	// Protokolleintrag (QUE-02 K2). Im Prüfermodus bleibt sie zulässig — die
	// Ausgabe von Daten ist sein Zweck, und sie ändert die Buchführung nicht.
	if b.accountingSvc == nil {
		return "", fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("ohne Zielpfad lässt sich die Datei nicht schreiben")
	}
	csv, err := b.accountingSvc.FilterEntriesCSV(context.Background(), filter)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(csv), 0o600); err != nil {
		return "", fmt.Errorf("die Datei konnte nicht geschrieben werden: %w", err)
	}
	return path, nil
}
