package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// --- Vorher/Nachher an den Stammdatenwegen ---------------------------------

// latestAudit liefert den jüngsten Protokolleintrag zu einer Objektart. Die
// Abfrage liefert absteigend nach Kennung — der erste Eintrag ist der jüngste.
func latestAudit(t *testing.T, env *testEnv, entityType string) domain.AuditLogEntry {
	t.Helper()
	entries, err := repository.NewAuditRepository(env.db).FindFiltered(
		context.Background(), 0, domain.AuditFilter{EntityType: entityType})
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("kein Protokolleintrag der Art %s gefunden", entityType)
	}
	return entries[0]
}

// fields liest das Vorher oder Nachher eines Eintrags als Feldkarte.
func fields(t *testing.T, raw string) map[string]any {
	t.Helper()
	if raw == "" {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("Vorher/Nachher ist kein JSON: %v (%s)", err, raw)
	}
	return out
}

// Der tatsächliche Weg, nicht nur die Hilfsfunktion: SaveContact liest den
// bisherigen Stand und protokolliert nur, was sich geändert hat.
func TestSaveContactLogsOnlyTheChangedFields(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	contact := &domain.Contact{
		Type: domain.ContactTypeVendor, Name: "Lieferant GmbH",
		Street: "Alte Straße 1", PostalCode: "20095", City: "Hamburg", CountryCode: "DE",
	}
	if err := env.contacts.SaveContact(ctx, contact); err != nil {
		t.Fatalf("Kontakt anlegen: %v", err)
	}

	contact.Street = "Neue Straße 7"
	if err := env.contacts.SaveContact(ctx, contact); err != nil {
		t.Fatalf("Kontakt ändern: %v", err)
	}

	entry := latestAudit(t, env, "CONTACT")
	before := fields(t, entry.Before)
	after := fields(t, entry.After)

	if before["street"] != "Alte Straße 1" {
		t.Errorf("das Vorher hat die alte Straße nicht: %v", before)
	}
	if after["street"] != "Neue Straße 7" {
		t.Errorf("das Nachher hat die neue Straße nicht: %v", after)
	}
	// Der unveränderte Name gehört nicht ins Protokoll: er verbärge die eine
	// Änderung zwischen den unveränderten Feldern.
	if _, ok := after["name"]; ok {
		t.Errorf("der unveränderte Name steht im Protokoll: %v", after)
	}
	if _, ok := after["vatPeriod"]; ok {
		t.Errorf("ein Feld, das der Repository-Weg normalisiert, erzeugt Rauschen: %v", after)
	}
}

// Dasselbe für die Firmeneinstellungen: eine geänderte Besteuerungsart steht mit
// beiden Ständen im Protokoll, der Rest nicht.
func TestUpdateCompanySettingsLogsOnlyTheChangedFields(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	settings := NewSettingsService(
		repository.NewSettingsRepository(env.db), repository.NewAuditRepository(env.db))

	current, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Unternehmensdaten lesen: %v", err)
	}
	current.TaxationType = "IST"
	if err := settings.UpdateCompanySettings(ctx, current); err != nil {
		t.Fatalf("Unternehmensdaten ändern: %v", err)
	}

	entry := latestAudit(t, env, "SETTINGS")
	before := fields(t, entry.Before)
	after := fields(t, entry.After)

	if before["taxationType"] != "SOLL" || after["taxationType"] != "IST" {
		t.Errorf("der Wechsel der Besteuerungsart fehlt im Protokoll: %v → %v", before, after)
	}
	if _, ok := after["companyName"]; ok {
		t.Errorf("der unveränderte Firmenname steht im Protokoll: %v", after)
	}
}

// Die Anlagenstammdaten sind der dritte Weg, den die Spezifikation nennt.
func TestSaveFixedAssetLogsBeforeAndAfter(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	assets := env.assets(t)

	asset, err := assets.Save(ctx, &domain.FixedAsset{
		Name: "Fräsmaschine", Class: domain.AssetClassTangible,
		AcquisitionDate: "2026-01-15", AcquisitionCost: 1_200_000, UsefulLifeMonths: 120,
		Method: domain.DepreciationLinear, Account: "0440", DepreciationAccount: "6220",
	})
	if err != nil {
		t.Fatalf("Anlagegut anlegen: %v", err)
	}

	asset.UsefulLifeMonths = 60
	asset.UsefulLifeReason = "Zweischichtbetrieb, amtliche AfA-Tabelle unterschritten"
	if _, err := assets.Save(ctx, asset); err != nil {
		t.Fatalf("Anlagegut ändern: %v", err)
	}

	entry := latestAudit(t, env, "ANLAGE")
	before := fields(t, entry.Before)
	after := fields(t, entry.After)
	if before["usefulLifeMonths"] != float64(120) || after["usefulLifeMonths"] != float64(60) {
		t.Errorf("die geänderte Nutzungsdauer fehlt im Protokoll: %v → %v", before, after)
	}
	if _, ok := after["name"]; ok {
		t.Errorf("der unveränderte Name steht im Protokoll: %v", after)
	}
	if _, ok := after["movements"]; ok {
		t.Errorf("die Bewegungen gehören nicht ins Vorher/Nachher: %v", after)
	}
}

// Der Abschlussstand des Geschäftsjahres ist der vierte Weg.
func TestFiscalYearStatusLogsBeforeAndAfter(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	closing := env.closing(t)

	if _, err := closing.SetAverageEmployees(ctx, 2026, 7); err != nil {
		t.Fatalf("Arbeitnehmerzahl setzen: %v", err)
	}
	entry := latestAudit(t, env, "FISCAL_YEAR")
	before := fields(t, entry.Before)
	after := fields(t, entry.After)
	if after["averageEmployees"] != float64(7) {
		t.Errorf("die neue Arbeitnehmerzahl fehlt im Protokoll: %v", after)
	}
	if _, ok := before["averageEmployees"]; ok && before["averageEmployees"] == float64(7) {
		t.Errorf("das Vorher hat schon den neuen Wert: %v", before)
	}
	if _, ok := after["year"]; ok {
		t.Errorf("das unveränderte Jahr steht im Protokoll: %v", after)
	}
}

// Die Kopfdatenerfassung selbst wird mit Vorher/Nachher protokolliert: die
// Kopfdaten gehen in den Beleg-Hash und damit in jede Buchung darauf.
func TestSaveHeaderLogsBeforeAndAfter(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt := filePDFReceipt(t, env, FileReceiptRequest{})
	if _, err := env.receipts.SaveHeader(ctx, receipt.ID, domain.ReceiptHeader{
		DocumentDate: "2026-03-01", IssuerName: "Lieferant GmbH",
		GrossAmount: 11900, TaxAmount: 1900, Currency: "EUR",
	}); err != nil {
		t.Fatalf("Kopfdaten erfassen: %v", err)
	}

	entry := latestAudit(t, env, "RECEIPT")
	after := fields(t, entry.After)
	if after["documentDate"] != "2026-03-01" || after["issuerName"] != "Lieferant GmbH" {
		t.Errorf("die erfassten Kopfdaten fehlen im Protokoll: %v", after)
	}
	if _, ok := after["receiptHash"]; !ok {
		t.Errorf("der neu gerechnete Beleg-Hash gehört ins Protokoll: %v", after)
	}
	if _, ok := after["files"]; ok {
		t.Errorf("die Dateiliste gehört nicht ins Vorher/Nachher der Kopfdaten: %v", after)
	}
}

// --- Vorher/Nachher an der Voranmeldung ------------------------------------

// Die Bestätigung einer Voranmeldung ist ein Statuswechsel und wird mit beiden
// Ständen protokolliert; die Bearbeiterkennung steht an der Anmeldung selbst.
func TestVatReturnSubmissionLogsBeforeAndAfter(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	vatRepo := repository.NewVatReturnRepository(env.db)
	rec := &domain.VatReturn{
		FiscalYear: 2026, PeriodType: domain.VatPeriodMonth, PeriodKey: "2026-03",
		PeriodFrom: "2026-03-01", PeriodTo: "2026-03-31", Status: domain.VatReturnDraft,
	}
	if err := vatRepo.Create(ctx, rec); err != nil {
		t.Fatalf("Entwurf anlegen: %v", err)
	}

	before := *rec
	rec.Status = domain.VatReturnSubmitted
	rec.SubmittedAt = "2026-04-10"
	rec.TransferTicket = "TT-4711"
	if err := vatRepo.Update(ctx, rec); err != nil {
		t.Fatalf("Bestätigung speichern: %v", err)
	}
	if err := repository.NewAuditRepository(env.db).LogChange(ctx, domain.AuditActionUpdate,
		"VAT_RETURN", "1", "Voranmeldung übermittelt", &before, rec); err != nil {
		t.Fatalf("Protokolleintrag: %v", err)
	}

	entry := latestAudit(t, env, "VAT_RETURN")
	beforeFields := fields(t, entry.Before)
	afterFields := fields(t, entry.After)
	if beforeFields["status"] != string(domain.VatReturnDraft) ||
		afterFields["status"] != string(domain.VatReturnSubmitted) {
		t.Errorf("der Statuswechsel fehlt im Protokoll: %v → %v", beforeFields, afterFields)
	}
	if _, ok := afterFields["figures"]; ok {
		t.Errorf("die Kennziffern gehören nicht ins Vorher/Nachher: %v", afterFields)
	}
}
