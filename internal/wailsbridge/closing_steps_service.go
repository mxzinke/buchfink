package wailsbridge

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Die Bridge der Abschlussbausteine (Welle 5a).
//
// Alle Methoden folgen demselben Muster wie der Rest der Bridge: lesen unter der
// Lesesperre, schreiben unter der Schreibsperre und hinter ensureWritable, das
// den Prüfermodus abweist. Listen kommen leer und nie als nil zurück — das
// Frontend liest sie ohne Umweg, und `null.length` nähme im Render den ganzen
// Baum mit.

// -------------------------------------------------------------------------
// Schrittliste
// -------------------------------------------------------------------------

// GetClosingSteps liefert die Bausteine des Jahresabschlusses mit ihrem Zustand.
func (b *BuchfinkBridge) GetClosingSteps(year int) (*service.ClosingSteps, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closingStepsSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.closingStepsSvc.Steps(context.Background(), year)
}

// SkipClosingStep übergeht einen Baustein mit Grund.
func (b *BuchfinkBridge) SkipClosingStep(year int, key, reason string) (*service.ClosingSteps, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.closingStepsSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.closingStepsSvc.SkipStep(context.Background(), year, domain.ClosingStepKey(key), reason)
}

// MarkClosingStepDone hakt einen Baustein ausdrücklich ab.
func (b *BuchfinkBridge) MarkClosingStepDone(year int, key string) (*service.ClosingSteps, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.closingStepsSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.closingStepsSvc.SetStep(
		context.Background(), year, domain.ClosingStepKey(key), domain.ClosingStepDone, "")
}

// -------------------------------------------------------------------------
// Rechnungsabgrenzung
// -------------------------------------------------------------------------

// ProposeAccruals sucht die Buchungen, deren Leistung über den Stichtag
// hinausreicht.
func (b *BuchfinkBridge) ProposeAccruals(year int) (*service.AccrualProposal, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accrualSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.accrualSvc.Propose(context.Background(), year)
}

// PreviewAccrual rechnet einen Abgrenzungsposten, ohne ihn zu buchen.
func (b *BuchfinkBridge) PreviewAccrual(req service.AccrualRequest) (*service.AccrualPreview, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accrualSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.accrualSvc.Preview(context.Background(), req)
}

// BookAccrual legt den Posten an und bucht seine Bildung.
func (b *BuchfinkBridge) BookAccrual(req service.AccrualRequest) (*domain.Accrual, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.accrualSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.accrualSvc.Book(context.Background(), req)
}

// GetAccruals liefert die im Geschäftsjahr gebildeten Abgrenzungsposten.
func (b *BuchfinkBridge) GetAccruals(year int) ([]domain.Accrual, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accrualSvc == nil {
		return []domain.Accrual{}, nil
	}
	return b.accrualSvc.List(context.Background(), year)
}

// GetAccrualReport ist der Bestand aller Abgrenzungen zu einem Stichtag.
func (b *BuchfinkBridge) GetAccrualReport(cutoff string) (*service.AccrualReport, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accrualSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.accrualSvc.Report(context.Background(), cutoff)
}

// -------------------------------------------------------------------------
// Rückstellungen
// -------------------------------------------------------------------------

// GetProvisions liefert die Rückstellungen, die ein Geschäftsjahr betreffen.
func (b *BuchfinkBridge) GetProvisions(year int) ([]domain.Provision, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.provisionSvc == nil {
		return []domain.Provision{}, nil
	}
	return b.provisionSvc.List(context.Background(), year)
}

// PreviewProvision rechnet eine Rückstellung samt Abzinsung, ohne zu buchen.
func (b *BuchfinkBridge) PreviewProvision(req service.ProvisionRequest) (*service.ProvisionPreview, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.provisionSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.provisionSvc.Preview(context.Background(), req)
}

// BookProvisionFormation bildet eine Rückstellung.
func (b *BuchfinkBridge) BookProvisionFormation(req service.ProvisionRequest) (*domain.Provision, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.provisionSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.provisionSvc.BookFormation(context.Background(), req)
}

// BookProvisionIncrease führt einer bestehenden Rückstellung Betrag zu.
func (b *BuchfinkBridge) BookProvisionIncrease(req service.ProvisionRequest) (*domain.Provision, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.provisionSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.provisionSvc.BookIncrease(context.Background(), req)
}

// BookProvisionRelease löst eine Rückstellung auf — nur mit Grund.
func (b *BuchfinkBridge) BookProvisionRelease(req service.ProvisionChangeRequest) (*domain.Provision, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.provisionSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.provisionSvc.BookRelease(context.Background(), req)
}

// BookProvisionConsumption verbraucht eine Rückstellung gegen ein
// Zahlungsmittelkonto.
func (b *BuchfinkBridge) BookProvisionConsumption(req service.ProvisionChangeRequest) (*domain.Provision, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.provisionSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.provisionSvc.BookConsumption(context.Background(), req)
}

// BookProvisionUnwinding bucht die Aufzinsung.
func (b *BuchfinkBridge) BookProvisionUnwinding(req service.ProvisionChangeRequest) (*domain.Provision, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.provisionSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.provisionSvc.BookUnwinding(context.Background(), req)
}

// SettleProvision erledigt eine Rückstellung: ein noch offener Rest wird mit
// Begründung aufgelöst.
func (b *BuchfinkBridge) SettleProvision(provisionID uint, date, reason string) (*domain.Provision, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.provisionSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.provisionSvc.Settle(context.Background(), provisionID, date, reason)
}

// GetProvisionMirror ist der Rückstellungsspiegel des Anhangs.
func (b *BuchfinkBridge) GetProvisionMirror(year int) (*accounting.ProvisionMirror, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.provisionSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.provisionSvc.Mirror(context.Background(), year)
}

// GetDiscountRates liefert die Abzinsungssätze eines Monats; leerer Monat heißt:
// die jüngsten hinterlegten.
func (b *BuchfinkBridge) GetDiscountRates(month string) ([]domain.DiscountRate, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.provisionSvc == nil {
		return []domain.DiscountRate{}, nil
	}
	return b.provisionSvc.DiscountRates(context.Background(), month)
}

// GetDiscountRateMonths nennt die Monate, für die Sätze hinterlegt sind.
func (b *BuchfinkBridge) GetDiscountRateMonths() ([]string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.provisionSvc == nil {
		return []string{}, nil
	}
	return b.provisionSvc.DiscountRateMonths(context.Background())
}

// SaveDiscountRates schreibt gepflegte Zinssätze fort.
func (b *BuchfinkBridge) SaveDiscountRates(rows []domain.DiscountRate) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return err
	}
	if b.provisionSvc == nil {
		return fmt.Errorf("kein aktiver Mandant")
	}
	return b.provisionSvc.SaveDiscountRates(context.Background(), rows)
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
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return 0, err
	}
	if b.provisionSvc == nil {
		return 0, fmt.Errorf("kein aktiver Mandant")
	}
	return b.provisionSvc.ImportDiscountRatesCSV(context.Background(), path, month, average)
}

// -------------------------------------------------------------------------
// Vorräte
// -------------------------------------------------------------------------

// GetInventoryAccounts listet die Vorratskonten mit Buchwert und Erfassung.
func (b *BuchfinkBridge) GetInventoryAccounts(year int) (*service.InventoryOverview, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closingBookingSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.closingBookingSvc.InventoryAccounts(context.Background(), year)
}

// PreviewInventory rechnet die Bestandsveränderung aus dem Inventurwert.
func (b *BuchfinkBridge) PreviewInventory(req service.InventoryRequest) (*service.InventoryPreview, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closingBookingSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.closingBookingSvc.PreviewInventory(context.Background(), req)
}

// BookInventory erfasst den Inventurwert und bucht die Bestandsveränderung.
func (b *BuchfinkBridge) BookInventory(req service.InventoryRequest) (*domain.InventoryCount, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.closingBookingSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.closingBookingSvc.BookInventory(context.Background(), req)
}

// -------------------------------------------------------------------------
// Umsatzsteuer-Verrechnung und Steuerrückstellung
// -------------------------------------------------------------------------

// PreviewVatSettlement zeigt die Salden der Steuerkonten und den Jahressaldo.
func (b *BuchfinkBridge) PreviewVatSettlement(year int) (*service.VatSettlement, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closingBookingSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.closingBookingSvc.PreviewVatSettlement(context.Background(), year)
}

// BookVatSettlement bucht die Jahresverrechnung der Umsatzsteuer.
func (b *BuchfinkBridge) BookVatSettlement(year int) (*domain.JournalEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.closingBookingSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.closingBookingSvc.BookVatSettlement(context.Background(), year)
}

// PreviewTaxProvision rechnet den Vorschlag für die Steuerrückstellung.
func (b *BuchfinkBridge) PreviewTaxProvision(year int) (*service.TaxProvisionPreview, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closingBookingSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.closingBookingSvc.PreviewTaxProvision(context.Background(), year)
}

// BookTaxProvision bildet die Steuerrückstellungen und bucht sie.
func (b *BuchfinkBridge) BookTaxProvision(req service.TaxProvisionRequest) ([]domain.Provision, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.closingBookingSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.closingBookingSvc.BookTaxProvision(context.Background(), req)
}

// -------------------------------------------------------------------------
// Ergebnisverwendung und Anhang
// -------------------------------------------------------------------------

// PreviewAppropriation rechnet den Beschluss über die Ergebnisverwendung.
func (b *BuchfinkBridge) PreviewAppropriation(
	year int, req service.AppropriationRequest,
) (*service.AppropriationPreview, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.appropriationSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.appropriationSvc.PreviewAppropriation(context.Background(), year, req)
}

// BookAppropriation hält den Beschluss fest und bucht ihn.
func (b *BuchfinkBridge) BookAppropriation(
	year int, req service.AppropriationRequest,
) (*domain.Appropriation, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.appropriationSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.appropriationSvc.BookAppropriation(context.Background(), year, req)
}

// GetAppropriation liefert den gespeicherten Beschluss eines Jahres.
func (b *BuchfinkBridge) GetAppropriation(year int) (*domain.Appropriation, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.appropriationSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.appropriationSvc.Appropriation(context.Background(), year)
}

// GetNotesTexts liefert die Abschnitte des Anhangs mit ihrem Text.
func (b *BuchfinkBridge) GetNotesTexts(year int) ([]service.NotesTextView, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.appropriationSvc == nil {
		return []service.NotesTextView{}, nil
	}
	return b.appropriationSvc.NotesTexts(context.Background(), year)
}

// SaveNotesText schreibt einen Anhangabschnitt fort.
func (b *BuchfinkBridge) SaveNotesText(year int, section, text string) ([]service.NotesTextView, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.appropriationSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.appropriationSvc.SaveNotesText(
		context.Background(), year, domain.NotesSection(section), text)
}

// -------------------------------------------------------------------------
// Verzeichnis nach § 5 Abs. 1 Satz 2 EStG und Überleitung
// -------------------------------------------------------------------------

// GetTaxElectionRegister liefert das Verzeichnis der steuerlichen Wahlrechte.
func (b *BuchfinkBridge) GetTaxElectionRegister(year int) (*service.TaxElectionRegister, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.taxRegisterSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.taxRegisterSvc.Register(context.Background(), year)
}

// ExportTaxElectionRegisterCSV gibt das Verzeichnis als CSV-Text zurück.
//
// Als Text und nicht als Datei: wohin die Datei gehört, entscheidet der
// Anwender im Speichern-Dialog, und ein Pfad, den das Backend wählt, wäre auf
// jedem Betriebssystem ein anderer falscher.
func (b *BuchfinkBridge) ExportTaxElectionRegisterCSV(year int) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.taxRegisterSvc == nil {
		return "", fmt.Errorf("kein aktiver Mandant")
	}
	return b.taxRegisterSvc.RegisterCSV(context.Background(), year)
}

// GetReconciliation ist die Überleitung Handelsbilanz → Steuerbilanz.
func (b *BuchfinkBridge) GetReconciliation(year int) (*service.Reconciliation, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.taxRegisterSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.taxRegisterSvc.Reconcile(context.Background(), year)
}
