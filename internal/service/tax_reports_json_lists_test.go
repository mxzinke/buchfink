package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/repository"
)

// -------------------------------------------------------------------------
// Kein `null` in den Listen der Welle 5c
// -------------------------------------------------------------------------

// Die Auswertungen der steuerlichen Nebenpflichten am leeren Mandanten.
//
// Ein nicht belegter Go-Slice wird in JSON zu `null`, und `null.map` nimmt im
// Render den ganzen Baum mit. Betroffen wäre der Regelfall: das Verzeichnis ohne
// Eintrag, die Lieferung ohne Nachweis, das Jahr ohne Geschenk. Geprüft wird an
// der Ausgabe und nicht an den Feldern — siehe assertNoNilSlices.
func TestWelle5cOutputsHaveNoNilLists(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	year, err := env.inputTax(t).Year(ctx, 2026)
	if err != nil {
		t.Fatalf("Verzeichnis nach § 15a UStG: %v", err)
	}
	assertNoNilSlices(t, "Verzeichnis nach § 15a UStG", year)

	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	vatIDs := env.vatIDs(t, "https://api.example.invalid/")
	checks, err := vatIDs.Checks(ctx, customer.ID)
	if err != nil {
		t.Fatalf("Verlauf der Bestätigungsanfragen: %v", err)
	}
	assertNoNilSlices(t, "Verlauf der Bestätigungsanfragen", checks)
	status, err := vatIDs.Status(ctx, customer)
	if err != nil {
		t.Fatalf("Bestätigungsstand: %v", err)
	}
	assertNoNilSlices(t, "Bestätigungsstand", status)

	gifts, err := env.gifts(t).NonDeductibleReport(ctx, 2026)
	if err != nil {
		t.Fatalf("Bericht nicht abziehbare Betriebsausgaben: %v", err)
	}
	assertNoNilSlices(t, "Bericht nicht abziehbare Betriebsausgaben", gifts)

	rates := env.currency(t, "https://rates.example.invalid/")
	rates.SetClosingService(env.closing(t))
	rates.SetOpenItemSource(NewPaymentService(
		env.journal, env.journalRepo,
		repository.NewPaymentAllocationRepository(env.db),
		env.contactRepo, repository.NewBankRepository(env.db)))
	valuation, err := rates.PreviewCurrencyValuation(ctx, 2026)
	if err != nil {
		t.Fatalf("Fremdwährungsbewertung: %v", err)
	}
	assertNoNilSlices(t, "Fremdwährungsbewertung", valuation)

	assets := env.assets(t)
	writeUps, err := assets.WriteUpReport(ctx, 2026)
	if err != nil {
		t.Fatalf("Wertaufholungsbericht: %v", err)
	}
	assertNoNilSlices(t, "Wertaufholungsbericht", writeUps)
	pools, err := assets.PoolConsistency(ctx, 2026)
	if err != nil {
		t.Fatalf("Sammelposten-Konsistenz: %v", err)
	}
	assertNoNilSlices(t, "Sammelposten-Konsistenz", pools)

	// Die AfA-Regeln kommen aus der eingebetteten Ressource und gehen unverändert
	// an die Maske.
	assertNoNilSlices(t, "AfA-Regeln", accounting.AfARuleSet())
}

// Dieselben Auswertungen, nachdem etwas darin steht: die Zusage gilt auch den
// Listen *in* den Zeilen — der Verwendungshistorie eines Wirtschaftsguts, den
// Buchungen eines Empfängers, den Abschreibungen eines Anlageguts. Sie sind am
// leeren Mandanten nicht zu sehen, weil der Weg dorthin über eine Zeile führt,
// die es dort nicht gibt.
func TestWelle5cFilledOutputsHaveNoNilLists(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	register := env.inputTax(t)
	if _, err := register.Register(ctx, RegisterInputTaxRequest{
		Label: "Pkw AN-2026-0001", Account: "0520",
		AcquisitionDate: "2026-01-15",
		NetAmount:       4_000_000, InputTaxAmount: 760_000,
	}); err != nil {
		t.Fatalf("Aufnahme ins Verzeichnis: %v", err)
	}
	year, err := register.Year(ctx, 2027)
	if err != nil {
		t.Fatalf("Verzeichnis nach § 15a UStG: %v", err)
	}
	if len(year.Rows) == 0 {
		t.Fatal("der Eintrag fehlt im Jahreslauf — der Test prüfte dann nichts")
	}
	assertNoNilSlices(t, "Verzeichnis nach § 15a UStG (gefüllt)", year)

	gifts := env.gifts(t)
	vendor := env.vendor(t, "Präsente GmbH", "DE", "")
	if _, err := env.posting.PostIncomingReceipt(ctx,
		env.giftReceipt(t, vendor.ID, "Dr. Meyer", 4_000)); err != nil {
		t.Fatalf("Geschenk buchen: %v", err)
	}
	report, err := gifts.NonDeductibleReport(ctx, 2026)
	if err != nil {
		t.Fatalf("Bericht nicht abziehbare Betriebsausgaben: %v", err)
	}
	if len(report.Recipients) == 0 {
		t.Fatal("der Empfänger fehlt im Bericht — der Test prüfte dann nichts")
	}
	assertNoNilSlices(t, "Bericht nicht abziehbare Betriebsausgaben (gefüllt)", report)

	assets := env.assets(t)
	env.impairedAsset(t, assets)
	writeUps, err := assets.WriteUpReport(ctx, 2026)
	if err != nil {
		t.Fatalf("Wertaufholungsbericht: %v", err)
	}
	if len(writeUps.Candidates) == 0 {
		t.Fatal("das Anlagegut fehlt im Bericht — der Test prüfte dann nichts")
	}
	assertNoNilSlices(t, "Wertaufholungsbericht (gefüllt)", writeUps)
}
