package wailsbridge

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Die Bedienung der Welle 7: Aufgabenliste, Monatsabschluss,
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
	cfg := b.host.Config()
	opts := service.TaskOptions{}
	if cfg.ReadOnly {
		opts.ReadOnlyUntil, opts.ReadOnlyReason = cfg.ReadOnlyUntil, cfg.ReadOnlyReason
	}
	return readOr(b, func() (*domain.TaskList, error) {
		empty := &domain.TaskList{}
		empty.EnsureLists()
		return empty, nil
	}, func(ctx context.Context, s *core.Services) (*domain.TaskList, error) {
		list, err := s.Tasks.Tasks(ctx, opts)
		if err != nil {
			return nil, err
		}
		list.EnsureLists()
		return list, nil
	})
}

// --- Monatsabschluss ------------------------------------------------------

// GetMonthCloseState liefert den Stand eines Monats ("JJJJ-MM") in seinen drei
// Schritten.
func (b *BuchfinkBridge) GetMonthCloseState(month string) (*service.MonthCloseState, error) {
	return readOr(b, func() (*service.MonthCloseState, error) {
		empty := &service.MonthCloseState{Month: month}
		empty.EnsureLists()
		return empty, nil
	}, func(ctx context.Context, s *core.Services) (*service.MonthCloseState, error) {
		state, err := s.MonthClose.State(ctx, month)
		if err != nil {
			return nil, err
		}
		state.EnsureLists()
		return state, nil
	})
}

// --- Bankabgleich ---------------------------------------------------------

// SuggestBankMatches liefert die wahrscheinlichsten Zuordnungen zu einem
// Bankumsatz. Gebucht wird davon nichts.
func (b *BuchfinkBridge) SuggestBankMatches(bankTxID uint) (*service.BankSuggestions, error) {
	return readOr(b, func() (*service.BankSuggestions, error) {
		empty := &service.BankSuggestions{BankTxID: bankTxID}
		empty.EnsureLists()
		return empty, nil
	}, func(ctx context.Context, s *core.Services) (*service.BankSuggestions, error) {
		suggestions, err := s.Bank.Suggest(ctx, bankTxID)
		if err != nil {
			return nil, err
		}
		suggestions.EnsureLists()
		return suggestions, nil
	})
}

// GetBankRules liefert die gelernten Zuordnungen wiederkehrender Umsätze.
func (b *BuchfinkBridge) GetBankRules() ([]domain.BankRule, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.BankRule, error) {
		return s.Bank.Rules(ctx)
	})
}

// DeleteBankRule entfernt eine gelernte Zuordnung.
func (b *BuchfinkBridge) DeleteBankRule(id uint) error {
	return b.change(func(ctx context.Context, s *core.Services) error {
		return s.Bank.DeleteRule(ctx, id)
	})
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
	// Erst den Pfad wählen: der Speichern-Dialog steht offen, solange der
	// Anwender überlegt.
	name := fmt.Sprintf("buchfink-pruefpfad-beleg-%d.%s", receiptID, format)
	path, err := b.saveFileDialog("Prüfpfad speichern", name)
	if err != nil || path == "" {
		return "", err
	}

	return read(b, func(ctx context.Context, s *core.Services) (string, error) {
		if format == "csv" {
			return s.AuditTrail.ExportCSV(ctx, receiptID, path)
		}
		return s.AuditTrail.ExportPDF(ctx, receiptID, path)
	})
}

// GetAuditTrail liefert den Prüfpfad eines Belegs zur Anzeige.
//
// Dieselbe Kette, die auch ausgegeben wird: die Anwenderin sieht auf dem Schirm,
// was der Prüfer in der Datei bekommt.
func (b *BuchfinkBridge) GetAuditTrail(receiptID uint) (*service.AuditTrail, error) {
	return readOr(b, func() (*service.AuditTrail, error) {
		empty := &service.AuditTrail{ReceiptID: receiptID}
		empty.EnsureLists()
		return empty, nil
	}, func(ctx context.Context, s *core.Services) (*service.AuditTrail, error) {
		trail, err := s.AuditTrail.Trail(ctx, receiptID)
		if err != nil {
			return nil, err
		}
		trail.EnsureLists()
		return trail, nil
	})
}

// SaveServiceProof schreibt den Leistungsnachweis an einen Eingangsbeleg.
func (b *BuchfinkBridge) SaveServiceProof(receiptID uint, text, date string) (*domain.Receipt, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Receipt, error) {
		return s.Receipts.SaveServiceProof(ctx, receiptID, text, date)
	})
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
