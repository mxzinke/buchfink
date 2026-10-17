package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Die Bridge-Methoden des Rechnungswesens aus Welle 5b.
//
// Sie stehen in einer eigenen Datei und nicht in app_service.go: das
// Rechnungswesen ist ein zusammenhängender Gegenstand — Korrektur, Storno,
// Versand, Anzahlungen, Nummernkreis —, und app_service.go ist ohnehin die
// Datei, in der man nichts mehr findet.

// RegenerateInvoiceDocument holt ein fehlendes Rechnungsdokument nach.
func (b *BuchfinkBridge) RegenerateInvoiceDocument(invoiceID uint) (*domain.Invoice, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Invoice, error) {
		return s.Invoices.RegenerateDocument(ctx, invoiceID)
	})
}

// CancelInvoiceWithDocument storniert eine Rechnung und stellt die
// Stornorechnung aus. Zurück kommt das Stornodokument, nicht die stornierte
// Rechnung — es ist das, was der Kunde bekommt.
func (b *BuchfinkBridge) CancelInvoiceWithDocument(invoiceID uint, reason string) (*domain.Invoice, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Invoice, error) {
		return s.Invoices.CancelWithDocument(ctx, invoiceID, reason)
	})
}

// CorrectInvoice storniert eine Rechnung und stellt die berichtigte aus.
func (b *BuchfinkBridge) CorrectInvoice(invoiceID uint, reason string, replacement domain.Invoice) (*domain.Invoice, error) {
	if err := b.prefetch(func(ctx context.Context, t *core.Tenant) error {
		return t.PrepareInvoice(ctx, &replacement)
	}); err != nil {
		return nil, err
	}
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Invoice, error) {
		return s.Invoices.CorrectInvoice(ctx, invoiceID, reason, &replacement)
	})
}

// MarkInvoiceSent vermerkt, wann und wie eine Rechnung hinausgegangen ist.
func (b *BuchfinkBridge) MarkInvoiceSent(invoiceID uint, date string, via string, note string) (*domain.Invoice, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Invoice, error) {
		return s.Invoices.MarkSent(ctx, invoiceID, date, domain.InvoiceSentVia(via), note)
	})
}

// GetInvoiceNumberGaps liefert den Lückenbericht des Rechnungsnummernkreises.
func (b *BuchfinkBridge) GetInvoiceNumberGaps(year int) (*domain.NumberGapReport, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*domain.NumberGapReport, error) {
		return s.Invoices.NumberGaps(ctx, year)
	})
}

// RecordInvoiceNumberGapReason dokumentiert, warum eine Nummer keine Rechnung
// hat. Die Betriebsprüfung fragt danach, und eine Antwort im Kopf des
// Geschäftsführers ist keine.
func (b *BuchfinkBridge) RecordInvoiceNumberGapReason(year int, sequence int64, reason string, detail string) error {
	year = b.yearOr(year)
	return b.change(func(ctx context.Context, s *core.Services) error {
		return s.Invoices.RecordNumberGapReason(
			ctx, year, sequence, domain.NumberGapReason(reason), detail)
	})
}

// GetInvoiceGroups liefert die Rechnungsverbünde mit ihrem Fortschritt.
func (b *BuchfinkBridge) GetInvoiceGroups() ([]domain.InvoiceGroup, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.InvoiceGroup, error) {
		return s.Invoices.GetInvoiceGroups(ctx, b.viewYear())
	})
}

// CreateInvoiceGroup legt einen Rechnungsverbund an.
func (b *BuchfinkBridge) CreateInvoiceGroup(req service.AdvanceGroupRequest) (*domain.InvoiceGroup, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.InvoiceGroup, error) {
		return s.Invoices.CreateInvoiceGroup(ctx, req)
	})
}

// IssueAdvanceInvoice stellt eine Abschlagsrechnung aus. Sie wird nicht
// gebucht: die Steuer entsteht erst mit der Vereinnahmung.
func (b *BuchfinkBridge) IssueAdvanceInvoice(req service.AdvanceInvoiceRequest) (*domain.Invoice, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Invoice, error) {
		return s.Invoices.IssueAdvanceInvoice(ctx, req)
	})
}

// SettleAdvance bucht den Zahlungseingang auf eine Abschlagsrechnung.
func (b *BuchfinkBridge) SettleAdvance(req service.SettleAdvanceRequest) (*domain.AdvanceItem, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.AdvanceItem, error) {
		return s.Invoices.SettleAdvance(ctx, req)
	})
}

// RefundAdvance bucht die Rückzahlung einer vereinnahmten Anzahlung.
//
// Sie ist der Weg vor dem Storno einer bezahlten Abschlagsrechnung: die Steuer
// der Anzahlung wird erst mit der Rückzahlung berichtigt (§ 17 Abs. 2 Nr. 2
// UStG).
func (b *BuchfinkBridge) RefundAdvance(req service.RefundAdvanceRequest) (*domain.AdvanceItem, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.AdvanceItem, error) {
		return s.Invoices.RefundAdvance(ctx, req)
	})
}

// GetOpenVendorAdvances liefert die geleisteten, noch nicht verrechneten
// Anzahlungen — die Schlussrechnung des Lieferanten setzt sie ab.
func (b *BuchfinkBridge) GetOpenVendorAdvances(contactID uint) ([]domain.VendorAdvance, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.VendorAdvance, error) {
		return s.Posting.OpenVendorAdvances(ctx, contactID)
	})
}

// GetAdvanceTargets liefert die Verwendungen einer geleisteten Anzahlung: wofür
// angezahlt wurde, entscheidet über den Bilanzposten und lässt sich aus dem
// Betrag nicht ableiten.
func (b *BuchfinkBridge) GetAdvanceTargets() []AdvanceTargetOption {
	targets := accounting.AllAdvanceTargets()
	out := make([]AdvanceTargetOption, 0, len(targets))
	for _, t := range targets {
		account, err := accounting.VendorAdvanceAccountFor(t)
		if err != nil {
			continue
		}
		out = append(out, AdvanceTargetOption{
			Key: string(t), Label: accounting.AdvanceTargetLabel(t), Account: account,
		})
	}
	return out
}

// AdvanceTargetOption ist eine Verwendung mit ihrem Konto für die Oberfläche.
type AdvanceTargetOption struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Account string `json:"account"`
}

// IssueFinalInvoice stellt die Schlussrechnung aus und verrechnet die
// Anzahlungen.
func (b *BuchfinkBridge) IssueFinalInvoice(req service.FinalInvoiceRequest) (*domain.Invoice, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Invoice, error) {
		return s.Invoices.IssueFinalInvoice(ctx, req)
	})
}

// GetOpenAdvances liefert die gestellten, noch nicht vereinnahmten Abschläge
// als offene Posten — die zweite Quelle der OP-Liste.
func (b *BuchfinkBridge) GetOpenAdvances() ([]domain.OpenItem, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.OpenItem, error) {
		return s.Invoices.OpenAdvanceItems(ctx)
	})
}

// WriteOffOpenItem bucht eine uneinbringliche Forderung aus.
func (b *BuchfinkBridge) WriteOffOpenItem(req service.WriteOffRequest) (*domain.JournalEntry, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.JournalEntry, error) {
		return s.Payments.WriteOffOpenItem(ctx, req)
	})
}

// GetUnitCodes liefert die Mengeneinheiten nach UN/ECE Rec. 20, die eine
// Rechnungsposition haben kann (BT-130).
func (b *BuchfinkBridge) GetUnitCodes() []domain.UnitCode {
	return domain.UnitCodes()
}

// GetEInvoiceProfiles liefert die Zielformate, in denen eine Rechnung
// ausgestellt werden kann.
func (b *BuchfinkBridge) GetEInvoiceProfiles() []domain.EInvoiceProfileInfo {
	return domain.EInvoiceProfiles()
}

// GetInvoiceSentViaOptions liefert die Wege, auf denen eine Rechnung als
// versendet vermerkt werden kann.
//
// Wie Einheiten und Profile kommt die Auswahl aus dem Fachmodell: Die Werte
// stehen in `domain.InvoiceSentVia`, und wer die Beschriftungen daneben in der
// Oberfläche pflegt, hat zwei Wahrheiten für dieselbe Liste.
func (b *BuchfinkBridge) GetInvoiceSentViaOptions() []domain.InvoiceSentViaOption {
	return domain.InvoiceSentViaOptions()
}

// GetNumberGapReasons liefert die Gründe, mit denen eine Lücke im
// Nummernkreis begründet werden kann.
func (b *BuchfinkBridge) GetNumberGapReasons() []domain.NumberGapReasonOption {
	return domain.NumberGapReasons()
}
