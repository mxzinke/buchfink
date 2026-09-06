package wailsbridge

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Die Bedienung der Welle 7: Aufgabenliste, Monatsabschluss, Mahnwesen,
// Zuordnungsvorschlag und Prüfpfad.
//
// Sie stehen in einer eigenen Datei, weil sie eine Sache sind: die Antwort auf
// die Frage, was eine Anwenderin ohne Buchhaltungsausbildung als Nächstes tun
// soll — und wie sie das, was sie tut, wieder findet.

// --- Aufgabenliste --------------------------------------------------------

// GetTasks liefert die Aufgabenliste in ihren drei Gruppen.
//
// Der Prüfermodus wird mitgegeben und nicht im Dienst nachgeschlagen: er ist
// eine Eigenschaft der Bedienung und steht in der Mandantenkonfiguration, die
// die Bridge führt.
func (b *BuchfinkBridge) GetTasks() (*domain.TaskList, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.taskSvc == nil {
		empty := &domain.TaskList{}
		empty.EnsureLists()
		return empty, nil
	}
	active, until, reason := b.readOnlyStateLocked()
	opts := service.TaskOptions{}
	if active {
		opts.ReadOnlyUntil, opts.ReadOnlyReason = until, reason
	}
	list, err := b.taskSvc.Tasks(context.Background(), opts)
	if err != nil {
		return nil, err
	}
	list.EnsureLists()
	return list, nil
}

// --- Monatsabschluss ------------------------------------------------------

// GetMonthCloseState liefert den Stand eines Monats ("JJJJ-MM") in seinen drei
// Schritten.
func (b *BuchfinkBridge) GetMonthCloseState(month string) (*service.MonthCloseState, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.monthCloseSvc == nil {
		empty := &service.MonthCloseState{Month: month}
		empty.EnsureLists()
		return empty, nil
	}
	state, err := b.monthCloseSvc.State(context.Background(), month)
	if err != nil {
		return nil, err
	}
	state.EnsureLists()
	return state, nil
}

// --- Bankabgleich ---------------------------------------------------------

// SuggestBankMatches liefert die wahrscheinlichsten Zuordnungen zu einem
// Bankumsatz. Gebucht wird davon nichts.
func (b *BuchfinkBridge) SuggestBankMatches(bankTxID uint) (*service.BankSuggestions, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.bankSvc == nil {
		empty := &service.BankSuggestions{BankTxID: bankTxID}
		empty.EnsureLists()
		return empty, nil
	}
	suggestions, err := b.bankSvc.Suggest(context.Background(), bankTxID)
	if err != nil {
		return nil, err
	}
	suggestions.EnsureLists()
	return suggestions, nil
}

// GetBankRules liefert die gelernten Zuordnungen wiederkehrender Umsätze.
func (b *BuchfinkBridge) GetBankRules() ([]domain.BankRule, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.bankSvc == nil {
		return []domain.BankRule{}, nil
	}
	return emptyList(b.bankSvc.Rules(context.Background()))
}

// DeleteBankRule entfernt eine gelernte Zuordnung.
func (b *BuchfinkBridge) DeleteBankRule(id uint) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return err
	}
	if b.bankSvc == nil {
		return fmt.Errorf("der Bankimport ist noch nicht eingerichtet")
	}
	return b.bankSvc.DeleteRule(context.Background(), id)
}

// --- Mahnwesen ------------------------------------------------------------

// GetDunningProposals liefert die Mahnvorschläge je Kunde.
func (b *BuchfinkBridge) GetDunningProposals() ([]service.DunningProposal, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.dunningSvc == nil {
		return []service.DunningProposal{}, nil
	}
	return emptyList(b.dunningSvc.Proposals(context.Background(), ""))
}

// CreateDunningNotices erzeugt die Mahnschreiben der ausgewählten Kunden.
func (b *BuchfinkBridge) CreateDunningNotices(req service.DunningRunRequest) ([]domain.DunningNotice, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.dunningSvc == nil {
		return nil, fmt.Errorf("das Mahnwesen ist noch nicht eingerichtet")
	}
	notices, err := b.dunningSvc.CreateMany(context.Background(), req)
	if notices == nil {
		notices = make([]domain.DunningNotice, 0)
	}
	for i := range notices {
		notices[i].EnsureLists()
	}
	return notices, err
}

// GetDunningNotices liefert die Mahnschreiben eines Kunden; 0 heißt: alle.
func (b *BuchfinkBridge) GetDunningNotices(contactID uint) ([]domain.DunningNotice, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.dunningSvc == nil {
		return []domain.DunningNotice{}, nil
	}
	return emptyList(b.dunningSvc.Notices(context.Background(), contactID))
}

// GetBaseRates liefert die Basiszinssätze nach § 247 BGB.
func (b *BuchfinkBridge) GetBaseRates() ([]domain.BaseRate, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.dunningSvc == nil {
		return []domain.BaseRate{}, nil
	}
	return emptyList(b.dunningSvc.BaseRates(context.Background()))
}

// SaveBaseRate trägt einen bekanntgegebenen Basiszinssatz nach.
//
// basisPoints ist der Satz in Hundertsteln eines Prozentpunktes: 152 sind
// 1,52 %. Ganzzahlig, weil eine Gleitkommazahl über die Brücke die Zinsrechnung
// um Cents verschöbe.
func (b *BuchfinkBridge) SaveBaseRate(validFrom string, basisPoints int) ([]domain.BaseRate, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.dunningSvc == nil {
		return nil, fmt.Errorf("das Mahnwesen ist noch nicht eingerichtet")
	}
	return emptyList(b.dunningSvc.SaveBaseRate(context.Background(), validFrom, basisPoints))
}

// --- Prüfpfad und Leistungsnachweis ---------------------------------------

// ExportAuditTrail schreibt den Prüfpfad eines Belegs und liefert den Pfad.
//
// format ist "csv" oder "pdf"; leer heißt PDF — der Prüfpfad wird gelesen und
// nicht weiterverarbeitet.
func (b *BuchfinkBridge) ExportAuditTrail(receiptID uint, format string) (string, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "pdf"
	}
	if format != "csv" && format != "pdf" {
		return "", fmt.Errorf("%q ist kein Ausgabeformat des Prüfpfads (erwartet csv oder pdf)", format)
	}
	// Erst den Pfad wählen, dann sperren: der Speichern-Dialog steht offen,
	// solange der Anwender überlegt.
	name := fmt.Sprintf("buchfink-pruefpfad-beleg-%d.%s", receiptID, format)
	path, err := b.saveFileDialog("Prüfpfad speichern", name)
	if err != nil || path == "" {
		return "", err
	}

	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.auditTrailSvc == nil {
		return "", fmt.Errorf("der Prüfpfad ist noch nicht eingerichtet")
	}
	if format == "csv" {
		return b.auditTrailSvc.ExportCSV(context.Background(), receiptID, path)
	}
	return b.auditTrailSvc.ExportPDF(context.Background(), receiptID, path)
}

// GetAuditTrail liefert den Prüfpfad eines Belegs zur Anzeige.
//
// Dieselbe Kette, die auch ausgegeben wird: die Anwenderin sieht auf dem Schirm,
// was der Prüfer in der Datei bekommt.
func (b *BuchfinkBridge) GetAuditTrail(receiptID uint) (*service.AuditTrail, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.auditTrailSvc == nil {
		empty := &service.AuditTrail{ReceiptID: receiptID}
		empty.EnsureLists()
		return empty, nil
	}
	trail, err := b.auditTrailSvc.Trail(context.Background(), receiptID)
	if err != nil {
		return nil, err
	}
	trail.EnsureLists()
	return trail, nil
}

// SaveServiceProof schreibt den Leistungsnachweis an einen Eingangsbeleg.
func (b *BuchfinkBridge) SaveServiceProof(receiptID uint, text, date string) (*domain.Receipt, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.receiptSvc == nil {
		return nil, fmt.Errorf("die Belegverwaltung ist noch nicht eingerichtet")
	}
	return b.receiptSvc.SaveServiceProof(context.Background(), receiptID, text, date)
}

// GetPaymentTermNotice liefert den Hinweis zu einem langen Zahlungsziel.
//
// Er kommt aus dem Fachbereich und nicht aus der Ansicht: die Grenze des
// § 271a BGB ist eine Rechtsfrage, und eine Norm, die in einer Maske noch
// einmal steht, wird bei der nächsten Änderung an einer der beiden Stellen
// vergessen. Leer heißt: unauffällig.
func (b *BuchfinkBridge) GetPaymentTermNotice(dueDays int) string {
	return domain.PaymentTermNotice(dueDays)
}
