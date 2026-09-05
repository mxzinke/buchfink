package wailsbridge

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Die Bridge der steuerlichen Nebenpflichten (Welle 5c).
//
// Alle Methoden folgen dem Muster des übrigen Bridge-Codes: lesen unter der
// Lesesperre, schreiben unter der Schreibsperre und hinter ensureWritable, das
// den Prüfermodus abweist. Listen kommen leer und nie als nil zurück.
//
// Zwei Methoden schreiben trotz ihres Namens: GetExchangeRate und
// PreviewCurrencyValuation holen einen Kurs, wenn keiner gespeichert ist, und
// legen ihn in der Historie ab. Ein Kurs, der geholt und nicht behalten würde,
// wäre bei der nächsten Frage ein anderer — und eine Bewertung, die sich beim
// zweiten Ansehen ändert, ist keine. Beide bleiben im Prüfermodus trotzdem
// beantwortbar: dort rechnen sie über den nur lesenden Kursdienst, der nichts
// holt und nichts ablegt.

// -------------------------------------------------------------------------
// Vorsteuerberichtigung nach § 15a UStG
// -------------------------------------------------------------------------

// GetInputTaxCorrections liefert das Verzeichnis mit Blick auf ein
// Geschäftsjahr.
func (b *BuchfinkBridge) GetInputTaxCorrections(year int) (*service.InputTaxCorrectionYear, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.inputTaxSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.inputTaxSvc.Year(context.Background(), year)
}

// PreviewInputTaxCorrection ist dieselbe Sicht unter dem Namen, unter dem der
// Abschlussbaustein sie aufruft: was würde gebucht.
func (b *BuchfinkBridge) PreviewInputTaxCorrection(year int) (*service.InputTaxCorrectionYear, error) {
	return b.GetInputTaxCorrections(year)
}

// RegisterInputTaxCorrection nimmt ein Wirtschaftsgut ins Verzeichnis auf.
func (b *BuchfinkBridge) RegisterInputTaxCorrection(
	req service.RegisterInputTaxRequest,
) (*domain.InputTaxCorrection, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.inputTaxSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.inputTaxSvc.Register(context.Background(), req)
}

// CloseInputTaxCorrection schließt einen Eintrag vorzeitig ab.
func (b *BuchfinkBridge) CloseInputTaxCorrection(
	id uint, reason, date string,
) (*domain.InputTaxCorrection, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.inputTaxSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.inputTaxSvc.Close(context.Background(), id, reason, date)
}

// SaveInputTaxUsage bestätigt oder ändert den Verwendungsanteil eines Jahres.
func (b *BuchfinkBridge) SaveInputTaxUsage(
	req service.SaveInputTaxUsageRequest,
) (*service.InputTaxCorrectionYear, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.inputTaxSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.inputTaxSvc.SaveUsage(context.Background(), req)
}

// BookInputTaxCorrection bucht die Berichtigungen eines Geschäftsjahres.
func (b *BuchfinkBridge) BookInputTaxCorrection(year int) (*service.InputTaxCorrectionYear, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.inputTaxSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.inputTaxSvc.BookYear(context.Background(), year)
}

// -------------------------------------------------------------------------
// Bestätigung der USt-IdNr.
// -------------------------------------------------------------------------

// GetVatIDChecks liefert den Verlauf der Bestätigungsabfragen eines Kontakts.
func (b *BuchfinkBridge) GetVatIDChecks(contactID uint) ([]domain.VatIDCheck, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.vatIDSvc == nil {
		return []domain.VatIDCheck{}, nil
	}
	return emptyList(b.vatIDSvc.Checks(context.Background(), contactID))
}

// GetVatIDStatus beantwortet ohne Netzaufruf, ob eine frische Bestätigung
// vorliegt.
func (b *BuchfinkBridge) GetVatIDStatus(contactID uint) (*service.VatIDStatus, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.vatIDSvc == nil || b.contactRepo == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	contact, err := b.contactRepo.FindByID(context.Background(), contactID)
	if err != nil {
		return nil, fmt.Errorf("der Geschäftspartner konnte nicht geladen werden: %w", err)
	}
	return b.vatIDSvc.Status(context.Background(), contact)
}

// CheckVatID führt die qualifizierte Bestätigungsanfrage aus und hebt das
// Ergebnis auf.
func (b *BuchfinkBridge) CheckVatID(contactID uint) (*domain.VatIDCheck, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.vatIDSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.vatIDSvc.Check(context.Background(), contactID)
}

// -------------------------------------------------------------------------
// Belegnachweis der innergemeinschaftlichen Lieferung
// -------------------------------------------------------------------------

// GetSupplyEvidenceKinds liefert die Belegarten mit ihrer Gruppe nach
// Art. 45a MwStVO.
func (b *BuchfinkBridge) GetSupplyEvidenceKinds() []accounting.EvidenceKindInfo {
	return accounting.EvidenceKinds()
}

// GetSupplyEvidence liefert den Nachweisstand einer Rechnung.
//
// transport ist „supplier" (Beförderung durch den Lieferer) oder „customer"
// (Abholfall); leer wird als Regelfall gelesen.
func (b *BuchfinkBridge) GetSupplyEvidence(invoiceID uint, transport string) (*service.SupplyEvidenceView, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.supplyEvidenceSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.supplyEvidenceSvc.View(
		context.Background(), invoiceID, accounting.TransportKind(transport))
}

// AddSupplyEvidence legt einen Nachweisbeleg zu einer Rechnung ab.
func (b *BuchfinkBridge) AddSupplyEvidence(req service.SupplyEvidenceRequest) (*service.SupplyEvidenceView, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.supplyEvidenceSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.supplyEvidenceSvc.Add(context.Background(), req)
}

// SetSupplyTransport hält an der Rechnung fest, wer den Gegenstand befördert
// hat. transport ist „supplier" oder „customer".
func (b *BuchfinkBridge) SetSupplyTransport(invoiceID uint, transport string) (*service.SupplyEvidenceView, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.supplyEvidenceSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.supplyEvidenceSvc.SetTransport(context.Background(), invoiceID, transport)
}

// RemoveSupplyEvidence nimmt einen Nachweisbeleg zurück.
func (b *BuchfinkBridge) RemoveSupplyEvidence(invoiceID, evidenceID uint, transport string) (*service.SupplyEvidenceView, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.supplyEvidenceSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.supplyEvidenceSvc.Remove(context.Background(), invoiceID, evidenceID, transport)
}

// GetSupplyEvidenceReport listet die steuerfreien ig. Lieferungen eines Jahres
// mit ihrem Nachweisstand.
func (b *BuchfinkBridge) GetSupplyEvidenceReport(year int) (*service.SupplyEvidenceReport, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.supplyEvidenceSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.supplyEvidenceSvc.Report(context.Background(), year)
}

// -------------------------------------------------------------------------
// Nicht abziehbare Betriebsausgaben
// -------------------------------------------------------------------------

// GetNonDeductibleCategories liefert die Kategorien des § 4 Abs. 5 EStG mit
// ihren Konten.
func (b *BuchfinkBridge) GetNonDeductibleCategories() []accounting.NonDeductibleCategory {
	return accounting.NonDeductibleCategories()
}

// GetNonDeductibleReport ist der Bericht eines Geschäftsjahres.
func (b *BuchfinkBridge) GetNonDeductibleReport(year int) (*service.NonDeductibleReport, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.giftSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.giftSvc.NonDeductibleReport(context.Background(), year)
}

// RebookGiftsForRecipient bucht die noch abziehbar stehenden Geschenke eines
// Empfängers auf das nicht abziehbare Konto um.
func (b *BuchfinkBridge) RebookGiftsForRecipient(req service.RebookGiftsRequest) (*service.GiftRebooking, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.giftSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.giftSvc.RebookGiftsForRecipient(context.Background(), req)
}

// -------------------------------------------------------------------------
// Fremdwährung
// -------------------------------------------------------------------------

// GetExchangeRate liefert den Kurs eines Tages.
//
// Im laufenden Betrieb ist die Methode schreibend: liegt kein Kurs vor, wird er
// geholt und in der Historie abgelegt. Ein Kurs, der nicht behalten würde, wäre
// beim nächsten Aufruf ein anderer, und dieselbe Buchung ergäbe zweimal
// verschiedene Beträge.
//
// Im Prüfermodus antwortet der nur lesende Kursdienst: er nimmt, was in der
// Historie steht, und holt nichts nach. Ein Prüfer muss den Kurs sehen können,
// mit dem gebucht wurde — und im abgeschlossenen Jahr steht er längst da.
func (b *BuchfinkBridge) GetExchangeRate(currency, date string) (*domain.ExchangeRate, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.currencySvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	svc := b.currencySvc
	if active, _, _ := b.readOnlyStateLocked(); active {
		svc = svc.ReadOnly()
	}
	return svc.RateAt(context.Background(), currency, date)
}

// GetExchangeRates liefert die Kurshistorie einer Währung.
func (b *BuchfinkBridge) GetExchangeRates(currency, from, to string) ([]domain.ExchangeRate, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.currencySvc == nil {
		return []domain.ExchangeRate{}, nil
	}
	return emptyList(b.currencySvc.Rates(context.Background(), currency, from, to))
}

// SaveExchangeRate nimmt einen von Hand erfassten Kurs auf.
func (b *BuchfinkBridge) SaveExchangeRate(rate domain.ExchangeRate) (*domain.ExchangeRate, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.currencySvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.currencySvc.SaveRate(context.Background(), rate)
}

// GetVatExchangeRates liefert die Umsatzsteuer-Durchschnittskurse eines
// Zeitraums.
func (b *BuchfinkBridge) GetVatExchangeRates(from, to string) ([]domain.VatExchangeRate, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.currencySvc == nil {
		return []domain.VatExchangeRate{}, nil
	}
	return emptyList(b.currencySvc.VatRates(context.Background(), from, to))
}

// SaveVatExchangeRate nimmt einen Durchschnittskurs auf.
func (b *BuchfinkBridge) SaveVatExchangeRate(rate domain.VatExchangeRate) (*domain.VatExchangeRate, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.currencySvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.currencySvc.SaveVatRate(context.Background(), rate)
}

// ImportVatExchangeRatesCSV liest die Durchschnittskurse aus einer CSV-Datei.
func (b *BuchfinkBridge) ImportVatExchangeRatesCSV(path string) (*service.VatRateImport, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.currencySvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.currencySvc.ImportVatRatesCSV(context.Background(), path)
}

// PreviewCurrencyValuation rechnet die Stichtagsbewertung, ohne sie zu buchen.
//
// Wie GetExchangeRate: im laufenden Betrieb holt sie den fehlenden Stichtagskurs
// und legt ihn ab, im Prüfermodus rechnet sie mit den gespeicherten Kursen und
// nennt es, wo einer fehlt. Die Vorschau ganz zu sperren wäre die schlechteste
// Antwort gewesen — sie ist eine Auswertung, und der Prüfermodus ist der Modus,
// in dem ausgewertet wird.
func (b *BuchfinkBridge) PreviewCurrencyValuation(year int) (*service.ForeignCurrencyValuation, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.currencySvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	svc := b.currencySvc
	if active, _, _ := b.readOnlyStateLocked(); active {
		svc = svc.ReadOnly()
	}
	return svc.PreviewCurrencyValuation(context.Background(), year)
}

// BookCurrencyValuation bucht die Stichtagsbewertung und ihre Auflösung.
func (b *BuchfinkBridge) BookCurrencyValuation(year int) (*service.ForeignCurrencyValuation, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.currencySvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.currencySvc.BookCurrencyValuation(context.Background(), year)
}

// -------------------------------------------------------------------------
// Anlagen
// -------------------------------------------------------------------------

// GetAfaRules liefert die Abschreibungsregeln aus der eingebetteten Ressource —
// dieselbe Datei, aus der gerechnet wird.
func (b *BuchfinkBridge) GetAfaRules() accounting.AfARules {
	return accounting.AfARuleSet()
}

// GetWriteUpReport listet die Anlagegüter, bei denen die Wertaufholung zu prüfen
// ist.
func (b *BuchfinkBridge) GetWriteUpReport(year int) (*service.WriteUpReport, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.assetSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.assetSvc.WriteUpReport(context.Background(), year)
}

// ConfirmImpairmentPersists hält fest, dass der Grund einer außerplanmäßigen
// Abschreibung fortbesteht.
func (b *BuchfinkBridge) ConfirmImpairmentPersists(assetID uint, year int, note string) (*service.WriteUpReport, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.assetSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.assetSvc.ConfirmImpairmentPersists(context.Background(), assetID, year, note)
}

// GetPoolConsistencyReport prüft die Einheitlichkeit des Wahlrechts nach
// § 6 Abs. 2a Satz 5 EStG.
func (b *BuchfinkBridge) GetPoolConsistencyReport(year int) (*service.PoolConsistencyReport, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.assetSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.assetSvc.PoolConsistency(context.Background(), year)
}

// CheckNearAcquisitionCost prüft den 15-%-Rahmen des § 6 Abs. 1 Nr. 1a EStG für
// eine geplante Instandsetzung.
func (b *BuchfinkBridge) CheckNearAcquisitionCost(
	assetID uint, date string, amount domain.Cents,
) (*service.NearAcquisitionCheck, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.assetSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.assetSvc.CheckNearAcquisitionCost(context.Background(), assetID, date, amount)
}

// CapitalizeNearAcquisitionCost bucht den Erhaltungsaufwand der ersten drei
// Jahre als nachträgliche Herstellungskosten auf das Gebäudekonto um.
//
// Der Vorschlag, den CheckNearAcquisitionCost macht, wird hier ausgeführt: eine
// Umbuchung SOLL Gebäude an HABEN Aufwandskonto samt Fortschreibung der
// Anschaffungskosten in der Kartei.
func (b *BuchfinkBridge) CapitalizeNearAcquisitionCost(
	req service.CapitalizeNearAcquisitionCostRequest,
) (*domain.FixedAsset, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.assetSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.assetSvc.CapitalizeNearAcquisitionCost(context.Background(), req)
}

// GetExemptionCertificateWarnings liefert die Freistellungsbescheinigungen nach
// § 48b EStG, die in den nächsten 30 Tagen ablaufen oder abgelaufen sind.
//
// today ist der Stichtag; leer heißt heute. Er steht in der Signatur, weil die
// Frist sonst nicht prüfbar wäre — und weil ein Prüfer denselben Stand zu einem
// vergangenen Tag sehen können muss.
func (b *BuchfinkBridge) GetExemptionCertificateWarnings(
	today string,
) ([]service.ExemptionCertificateWarning, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.contactSvc == nil {
		return []service.ExemptionCertificateWarning{}, nil
	}
	return b.contactSvc.ExemptionCertificateWarnings(context.Background(), today)
}
