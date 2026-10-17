package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Die Bridge der Abschlussbausteine (Welle 5a). Lesen und Schreiben gehen wie
// überall über read und write (session.go).

// -------------------------------------------------------------------------
// Schrittliste
// -------------------------------------------------------------------------

// GetClosingSteps liefert die Bausteine des Jahresabschlusses mit ihrem Zustand.
func (b *BuchfinkBridge) GetClosingSteps(year int) (*service.ClosingSteps, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*service.ClosingSteps, error) {
		return s.ClosingSteps.Steps(ctx, year)
	})
}

// SkipClosingStep übergeht einen Baustein mit Grund.
func (b *BuchfinkBridge) SkipClosingStep(year int, key, reason string) (*service.ClosingSteps, error) {
	year = b.yearOr(year)
	return write(b, func(ctx context.Context, s *core.Services) (*service.ClosingSteps, error) {
		return s.ClosingSteps.SkipStep(ctx, year, domain.ClosingStepKey(key), reason)
	})
}

// ReopenClosingStep nimmt das Überspringen eines Bausteins zurück: er steht
// wieder offen. Möglich, solange das Jahr weder festgeschrieben noch
// festgestellt ist — danach führt der Weg zurück über den Storno.
func (b *BuchfinkBridge) ReopenClosingStep(year int, key, reason string) (*service.ClosingSteps, error) {
	year = b.yearOr(year)
	return write(b, func(ctx context.Context, s *core.Services) (*service.ClosingSteps, error) {
		return s.ClosingSteps.ReopenStep(ctx, year, domain.ClosingStepKey(key), reason)
	})
}

// -------------------------------------------------------------------------
// Rechnungsabgrenzung
// -------------------------------------------------------------------------

// GetAccruals liefert die im Geschäftsjahr gebildeten Abgrenzungsposten.
func (b *BuchfinkBridge) GetAccruals(year int) ([]domain.Accrual, error) {
	year = b.yearOr(year)
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.Accrual, error) {
		return s.Accruals.List(ctx, year)
	})
}

// -------------------------------------------------------------------------
// Rückstellungen
// -------------------------------------------------------------------------

// GetProvisions liefert die Rückstellungen, die ein Geschäftsjahr betreffen.
func (b *BuchfinkBridge) GetProvisions(year int) ([]domain.Provision, error) {
	year = b.yearOr(year)
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.Provision, error) {
		return s.Provisions.List(ctx, year)
	})
}

// GetProvisionMirror ist der Rückstellungsspiegel des Anhangs.
func (b *BuchfinkBridge) GetProvisionMirror(year int) (*accounting.ProvisionMirror, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*accounting.ProvisionMirror, error) {
		return s.Provisions.Mirror(ctx, year)
	})
}

// -------------------------------------------------------------------------
// Umsatzsteuer-Verrechnung und Steuerrückstellung
// -------------------------------------------------------------------------

// PreviewVatSettlement zeigt die Salden der Steuerkonten und den Jahressaldo.
func (b *BuchfinkBridge) PreviewVatSettlement(year int) (*service.VatSettlement, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*service.VatSettlement, error) {
		return s.ClosingBookings.PreviewVatSettlement(ctx, year)
	})
}

// PreviewTaxProvision rechnet den Vorschlag für die Steuerrückstellung.
func (b *BuchfinkBridge) PreviewTaxProvision(year int) (*service.TaxProvisionPreview, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*service.TaxProvisionPreview, error) {
		return s.ClosingBookings.PreviewTaxProvision(ctx, year, service.TaxProvisionOptions{})
	})
}

// -------------------------------------------------------------------------
// Ergebnisverwendung und Anhang
// -------------------------------------------------------------------------

// PreviewAppropriation rechnet den Beschluss über die Ergebnisverwendung.
func (b *BuchfinkBridge) PreviewAppropriation(
	year int, req service.AppropriationRequest,
) (*service.AppropriationPreview, error) {
	if year == 0 {
		// Verwendet wird das Ergebnis des Vorjahres der Ansicht.
		year = b.viewYear() - 1
	}
	return read(b, func(ctx context.Context, s *core.Services) (*service.AppropriationPreview, error) {
		return s.Appropriation.PreviewAppropriation(ctx, year, req)
	})
}

// BookAppropriation hält den Beschluss fest und bucht ihn.
func (b *BuchfinkBridge) BookAppropriation(
	year int, req service.AppropriationRequest,
) (*domain.Appropriation, error) {
	if year == 0 {
		// Verwendet wird das Ergebnis des Vorjahres der Ansicht.
		year = b.viewYear() - 1
	}
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Appropriation, error) {
		return s.Appropriation.BookAppropriation(ctx, year, req)
	}, core.AppOnly())
}

// GetAppropriation liefert den gespeicherten Beschluss eines Jahres.
func (b *BuchfinkBridge) GetAppropriation(year int) (*domain.Appropriation, error) {
	if year == 0 {
		// Verwendet wird das Ergebnis des Vorjahres der Ansicht.
		year = b.viewYear() - 1
	}
	return read(b, func(ctx context.Context, s *core.Services) (*domain.Appropriation, error) {
		return s.Appropriation.Appropriation(ctx, year)
	})
}

// GetNotesTexts liefert die Abschnitte des Anhangs mit ihrem Text.
func (b *BuchfinkBridge) GetNotesTexts(year int) ([]service.NotesTextView, error) {
	year = b.yearOr(year)
	return readList(b, func(ctx context.Context, s *core.Services) ([]service.NotesTextView, error) {
		return s.Appropriation.NotesTexts(ctx, year)
	})
}

// -------------------------------------------------------------------------
// Verzeichnis nach § 5 Abs. 1 Satz 2 EStG und Überleitung
// -------------------------------------------------------------------------

// GetTaxElectionRegister liefert das Verzeichnis der steuerlichen Wahlrechte.
func (b *BuchfinkBridge) GetTaxElectionRegister(year int) (*service.TaxElectionRegister, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*service.TaxElectionRegister, error) {
		return s.TaxRegister.Register(ctx, year)
	})
}

// GetReconciliation ist die Überleitung Handelsbilanz → Steuerbilanz.
func (b *BuchfinkBridge) GetReconciliation(year int) (*service.Reconciliation, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*service.Reconciliation, error) {
		return s.TaxRegister.Reconcile(ctx, year)
	})
}
