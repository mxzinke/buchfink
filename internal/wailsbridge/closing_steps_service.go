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

// MarkClosingStepDone hakt einen Baustein ausdrücklich ab.
func (b *BuchfinkBridge) MarkClosingStepDone(year int, key string) (*service.ClosingSteps, error) {
	year = b.yearOr(year)
	return write(b, func(ctx context.Context, s *core.Services) (*service.ClosingSteps, error) {
		return s.ClosingSteps.SetStep(
			ctx, year, domain.ClosingStepKey(key), domain.ClosingStepDone, "")
	})
}

// -------------------------------------------------------------------------
// Rechnungsabgrenzung
// -------------------------------------------------------------------------

// ProposeAccruals sucht die Buchungen, deren Leistung über den Stichtag
// hinausreicht.
func (b *BuchfinkBridge) ProposeAccruals(year int) (*service.AccrualProposal, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*service.AccrualProposal, error) {
		return s.Accruals.Propose(ctx, year)
	})
}

// PreviewAccrual rechnet einen Abgrenzungsposten, ohne ihn zu buchen.
func (b *BuchfinkBridge) PreviewAccrual(req service.AccrualRequest) (*service.AccrualPreview, error) {
	req.FiscalYear = b.yearOr(req.FiscalYear)
	return read(b, func(ctx context.Context, s *core.Services) (*service.AccrualPreview, error) {
		return s.Accruals.Preview(ctx, req)
	})
}

// BookAccrual legt den Posten an und bucht seine Bildung.
func (b *BuchfinkBridge) BookAccrual(req service.AccrualRequest) (*domain.Accrual, error) {
	req.FiscalYear = b.yearOr(req.FiscalYear)
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Accrual, error) {
		return s.Accruals.Book(ctx, req)
	})
}

// GetAccruals liefert die im Geschäftsjahr gebildeten Abgrenzungsposten.
func (b *BuchfinkBridge) GetAccruals(year int) ([]domain.Accrual, error) {
	year = b.yearOr(year)
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.Accrual, error) {
		return s.Accruals.List(ctx, year)
	})
}

// GetAccrualReport ist der Bestand aller Abgrenzungen zu einem Stichtag.
func (b *BuchfinkBridge) GetAccrualReport(cutoff string) (*service.AccrualReport, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*service.AccrualReport, error) {
		return s.Accruals.Report(ctx, cutoff)
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

// PreviewProvision rechnet eine Rückstellung samt Abzinsung, ohne zu buchen.
func (b *BuchfinkBridge) PreviewProvision(req service.ProvisionRequest) (*service.ProvisionPreview, error) {
	req.FiscalYear = b.yearOr(req.FiscalYear)
	return read(b, func(ctx context.Context, s *core.Services) (*service.ProvisionPreview, error) {
		return s.Provisions.Preview(ctx, req)
	})
}

// BookProvisionFormation bildet eine Rückstellung.
func (b *BuchfinkBridge) BookProvisionFormation(req service.ProvisionRequest) (*domain.Provision, error) {
	req.FiscalYear = b.yearOr(req.FiscalYear)
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Provision, error) {
		return s.Provisions.BookFormation(ctx, req)
	})
}

// BookProvisionIncrease führt einer bestehenden Rückstellung Betrag zu.
func (b *BuchfinkBridge) BookProvisionIncrease(req service.ProvisionRequest) (*domain.Provision, error) {
	req.FiscalYear = b.yearOr(req.FiscalYear)
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Provision, error) {
		return s.Provisions.BookIncrease(ctx, req)
	})
}

// BookProvisionRelease löst eine Rückstellung auf — nur mit Grund.
func (b *BuchfinkBridge) BookProvisionRelease(req service.ProvisionChangeRequest) (*domain.Provision, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Provision, error) {
		return s.Provisions.BookRelease(ctx, req)
	})
}

// BookProvisionConsumption verbraucht eine Rückstellung gegen ein
// Zahlungsmittelkonto.
func (b *BuchfinkBridge) BookProvisionConsumption(req service.ProvisionChangeRequest) (*domain.Provision, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Provision, error) {
		return s.Provisions.BookConsumption(ctx, req)
	})
}

// BookProvisionUnwinding bucht die Aufzinsung.
func (b *BuchfinkBridge) BookProvisionUnwinding(req service.ProvisionChangeRequest) (*domain.Provision, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Provision, error) {
		return s.Provisions.BookUnwinding(ctx, req)
	})
}

// SettleProvision erledigt eine Rückstellung: ein noch offener Rest wird mit
// Begründung aufgelöst.
func (b *BuchfinkBridge) SettleProvision(provisionID uint, date, reason string) (*domain.Provision, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Provision, error) {
		return s.Provisions.Settle(ctx, provisionID, date, reason)
	})
}

// GetProvisionMirror ist der Rückstellungsspiegel des Anhangs.
func (b *BuchfinkBridge) GetProvisionMirror(year int) (*accounting.ProvisionMirror, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*accounting.ProvisionMirror, error) {
		return s.Provisions.Mirror(ctx, year)
	})
}

// GetDiscountRates liefert die Abzinsungssätze eines Monats; leerer Monat heißt:
// die jüngsten hinterlegten.
func (b *BuchfinkBridge) GetDiscountRates(month string) ([]domain.DiscountRate, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.DiscountRate, error) {
		return s.Provisions.DiscountRates(ctx, month)
	})
}

// GetDiscountRateMonths nennt die Monate, für die Sätze hinterlegt sind.
func (b *BuchfinkBridge) GetDiscountRateMonths() ([]string, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]string, error) {
		return s.Provisions.DiscountRateMonths(ctx)
	})
}

// SaveDiscountRates schreibt gepflegte Zinssätze fort.
func (b *BuchfinkBridge) SaveDiscountRates(rows []domain.DiscountRate) error {
	return b.change(func(ctx context.Context, s *core.Services) error {
		return s.Provisions.SaveDiscountRates(ctx, rows)
	})
}

// ImportDiscountRatesCSV liest die Veröffentlichung der Deutschen Bundesbank.
//
// Die Signatur weicht bewusst von der Spezifikation ab, die nur den Pfad nennt:
// in der CSV der Bundesbank stehen Restlaufzeit und Satz, aber weder der Monat
// der Veröffentlichung noch die Durchschnittsdauer. Beides ist Teil des
// Schlüssels der Zinstabelle und muss deshalb vom Aufrufer kommen — geraten
// würde sonst der Stichtagsbezug des § 253 Abs. 2 Satz 1 HGB. Das Frontend
// reicht alle drei Argumente durch (bridge.ts, api.ts).
func (b *BuchfinkBridge) ImportDiscountRatesCSV(path, month string, average int) (int, error) {
	return write(b, func(ctx context.Context, s *core.Services) (int, error) {
		return s.Provisions.ImportDiscountRatesCSV(ctx, path, month, average)
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

// BookVatSettlement bucht die Jahresverrechnung der Umsatzsteuer.
func (b *BuchfinkBridge) BookVatSettlement(year int) (*domain.JournalEntry, error) {
	year = b.yearOr(year)
	return write(b, func(ctx context.Context, s *core.Services) (*domain.JournalEntry, error) {
		return s.ClosingBookings.BookVatSettlement(ctx, year)
	})
}

// PreviewTaxProvision rechnet den Vorschlag für die Steuerrückstellung.
func (b *BuchfinkBridge) PreviewTaxProvision(year int) (*service.TaxProvisionPreview, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*service.TaxProvisionPreview, error) {
		return s.ClosingBookings.PreviewTaxProvision(ctx, year)
	})
}

// BookTaxProvision bildet die Steuerrückstellungen und bucht sie.
func (b *BuchfinkBridge) BookTaxProvision(req service.TaxProvisionRequest) ([]domain.Provision, error) {
	req.FiscalYear = b.yearOr(req.FiscalYear)
	return emptyList(write(b, func(ctx context.Context, s *core.Services) ([]domain.Provision, error) {
		return s.ClosingBookings.BookTaxProvision(ctx, req)
	}))
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
	})
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

// SaveNotesText schreibt einen Anhangabschnitt fort.
func (b *BuchfinkBridge) SaveNotesText(year int, section, text string) ([]service.NotesTextView, error) {
	year = b.yearOr(year)
	return emptyList(write(b, func(ctx context.Context, s *core.Services) ([]service.NotesTextView, error) {
		return s.Appropriation.SaveNotesText(ctx, year, domain.NotesSection(section), text)
	}))
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

// ExportTaxElectionRegisterCSV gibt das Verzeichnis als CSV-Text zurück.
//
// Als Text und nicht als Datei: wohin die Datei gehört, entscheidet der
// Anwender im Speichern-Dialog, und ein Pfad, den das Backend wählt, wäre auf
// jedem Betriebssystem ein anderer falscher.
func (b *BuchfinkBridge) ExportTaxElectionRegisterCSV(year int) (string, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (string, error) {
		return s.TaxRegister.RegisterCSV(ctx, year)
	})
}

// GetReconciliation ist die Überleitung Handelsbilanz → Steuerbilanz.
func (b *BuchfinkBridge) GetReconciliation(year int) (*service.Reconciliation, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*service.Reconciliation, error) {
		return s.TaxRegister.Reconcile(ctx, year)
	})
}
