package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// -------------------------------------------------------------------------
// Verzeichnis nach § 15a UStG: die Aktivierung ohne Vorsteuer
// -------------------------------------------------------------------------

// Ein Gebäude ohne gezogene Vorsteuer kommt nicht ins Verzeichnis — und die
// Aktivierung geht trotzdem durch.
//
// Der Grundstückserwerb ist nach § 4 Nr. 9 Buchst. a UStG steuerfrei; ohne
// Option nach § 9 UStG steht in der Rechnung keine Steuer. § 15a UStG
// berichtigt aber den Abzug, den es gegeben hat, und ein Eintrag über null Euro
// Vorsteuer ist keiner. Vor der Nachbesserung meldete das Speichern einen
// Fehler, obwohl das Anlagegut längst in der Kartei stand: der Anwender
// wiederholte den Vorgang und legte es ein zweites Mal an.
func TestBuildingWithoutInputTaxIsNotRegistered(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.assets(t)
	svc.SetInputTaxRegister(env.inputTax(t))

	building := env.building(t, svc)
	if building.InventoryNumber == "" {
		t.Fatal("das Gebäude wurde nicht aktiviert")
	}

	register := repository.NewInputTaxCorrectionRepository(env.db)
	entry, err := register.FindByAsset(ctx, building.ID)
	if err != nil {
		t.Fatalf("Verzeichnis lesen: %v", err)
	}
	if entry != nil {
		t.Errorf("das Verzeichnis führt %s mit %s € Vorsteuer — ohne Vorsteuer gibt es nichts zu "+
			"berichtigen", entry.Label, entry.InputTaxAmount)
	}
}

// Mit Vorsteuer — nach der Option des § 9 UStG — gehört das Gebäude ins
// Verzeichnis, und zwar mit dem zehnjährigen Berichtigungszeitraum des
// § 15a Abs. 1 Satz 2 UStG.
func TestBuildingWithInputTaxIsRegisteredForTenYears(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.assets(t)
	svc.SetInputTaxRegister(env.inputTax(t))

	building, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Betriebsgebäude", Class: domain.AssetClassTangible,
		Account: "0240", DepreciationAccount: "6221",
		AcquisitionDate: "2026-01-02", AcquisitionCost: 20_000_000,
		UsefulLifeMonths: 400, Method: domain.DepreciationBuildingLinear,
		BuildingReferenceDate: "2005-03-01",
		InputTaxAmount:        3_800_000,
	})
	if err != nil {
		t.Fatalf("Aktivierung mit Vorsteuer: %v", err)
	}

	register := repository.NewInputTaxCorrectionRepository(env.db)
	entry, err := register.FindByAsset(ctx, building.ID)
	if err != nil {
		t.Fatalf("Verzeichnis lesen: %v", err)
	}
	if entry == nil {
		t.Fatal("ein Gebäude mit gezogener Vorsteuer gehört ins Verzeichnis nach § 15a UStG")
	}
	if entry.CorrectionPeriodYears != 10 {
		t.Errorf("Berichtigungszeitraum %d Jahre — für ein Gebäude sind es zehn",
			entry.CorrectionPeriodYears)
	}
	if entry.InputTaxAmount != 3_800_000 {
		t.Errorf("Vorsteuer %s € — erwartet 38.000,00 €", entry.InputTaxAmount)
	}
}

// -------------------------------------------------------------------------
// Vorsteuerschlüssel: die Rundung je Steuersatzgruppe
// -------------------------------------------------------------------------

// Zwei Positionen zum selben Steuersatz mit geteiltem Vorsteuerabzug ergeben
// zusammen den Bruttobetrag der Rechnung.
//
// Die nicht abziehbare Vorsteuer wird je Steuersatzgruppe einmal gerundet, wie
// die Steuerzeile selbst. Wurde sie je Position gerundet, kamen 79,31 € statt
// 79,33 € heraus, und die Gegenzeile glich die zwei Cent still aus: die
// Verbindlichkeit stimmte dann mit der Rechnung des Lieferanten nicht überein.
func TestSplitInputTaxRoundsOncePerRateGroup(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Autohaus GmbH", "DE", "")

	req := env.receipt(t, vendor.ID, "fahrzeugkosten", 3_333,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Positions[0].InputTaxShare = 500
	req.Positions[0].InputTaxShareReason = "Kfz zur Hälfte betrieblich genutzt, Fahrtenbuch 2026"
	req.Positions = append(req.Positions, req.Positions[0])

	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}

	var expense, inputTax, settlement domain.Cents
	for _, l := range entry.Lines {
		switch {
		case l.TaxKey == "VST19":
			inputTax += l.Amount
		case l.Side == domain.SideDebit:
			expense += l.Amount
		default:
			settlement += l.Amount
		}
	}

	// 66,66 € netto, 19 % einmal gerundet: 12,67 € — brutto 79,33 €.
	if settlement != 7_933 {
		t.Errorf("Gegenzeile %s € — erwartet 79,33 € (66,66 € netto zuzüglich 12,67 € Steuer)",
			settlement)
	}
	// Abziehbar ist die Steuer auf 33,34 €: 6,33 €.
	if inputTax != 633 {
		t.Errorf("Vorsteuer %s € — erwartet 6,33 € (19 %% von 33,34 €)", inputTax)
	}
	// Der Aufwand trägt den Rest: 66,66 € zuzüglich 6,34 € nicht abziehbarer
	// Vorsteuer.
	if expense != 7_300 {
		t.Errorf("Aufwand %s € — erwartet 73,00 €", expense)
	}
	if !entry.IsBalanced() {
		t.Error("die Buchung ist nicht ausgeglichen")
	}
}

// -------------------------------------------------------------------------
// Nutzungsdauer: Begründung nur bei der Entscheidung
// -------------------------------------------------------------------------

// Eine Stammdatenänderung an einem bestehenden Anlagegut verlangt keine
// Begründung für die Nutzungsdauer.
//
// Die Pflicht gehört zur Entscheidung über die Nutzungsdauer. Griffe sie bei
// jedem Speichern, ließe sich der vor dieser Welle mit 36 Monaten angelegte
// Server nicht einmal mehr umbenennen — eine Vorschrift rückwirkend auf einen
// abgeschlossenen Vorgang anzuwenden, ist nicht ihr Zweck.
func TestUsefulLifeReasonOnlyWhenTheDurationChanges(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.assets(t)

	server, err := svc.Save(ctx, &domain.FixedAsset{
		Name: "Server", Class: domain.AssetClassTangible,
		Account: "0690", DepreciationAccount: "6220",
		AcquisitionDate: "2026-01-15", AcquisitionCost: 900_000,
		UsefulLifeMonths: 36, Method: domain.DepreciationLinear,
		UsefulLifeReason: "Herstellergarantie über drei Jahre, Nutzung im Rechenzentrum",
	})
	if err != nil {
		t.Fatalf("Zugang mit Begründung: %v", err)
	}

	// Der Bestand aus der Zeit vor dieser Welle: die Begründung fehlt, die
	// Nutzungsdauer bleibt.
	server.Name = "Server (Rack 2)"
	server.UsefulLifeReason = ""
	if _, err := svc.Save(ctx, server); err != nil {
		t.Errorf("die Stammdatenänderung darf nicht an der Begründung hängen: %v", err)
	}

	// Die geänderte Nutzungsdauer dagegen ist die Entscheidung, um die es geht.
	server.UsefulLifeMonths = 48
	_, err = svc.Save(ctx, server)
	if err == nil {
		t.Fatal("eine geänderte Nutzungsdauer ohne Begründung ist abzuweisen")
	}
	if !strings.Contains(err.Error(), "Computerhardware") {
		t.Errorf("die Meldung soll sagen, wofür der Vorschlag gilt: %v", err)
	}
}

// -------------------------------------------------------------------------
// USt-IdNr.: die eigene Nummer ist kein Netzproblem
// -------------------------------------------------------------------------

// Fehlt die eigene USt-IdNr., kommt die Bestätigungsanfrage nach § 18e UStG
// gar nicht zustande. Dieser Fall darf nicht wie ein nicht erreichbares
// Bundeszentralamt behandelt werden: er ließe sich sonst mit einem Grund
// übersteuern, obwohl der Mangel in den eigenen Stammdaten liegt und ohne Netz
// zu beheben ist.
func TestMissingOwnVatIDIsNotAnOverridableOutage(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, calls := bzstServer(t, "200", map[string]string{
		"ergName": "A", "ergOrt": "A", "ergPlz": "A", "ergStr": "A",
	})
	svc := env.vatIDs(t, server.URL)
	env.setSettings(t, func(c *domain.CompanySettings) { c.VatID = "" })
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")

	err := svc.EnsureConfirmed(ctx, customer, "")
	if err == nil {
		t.Fatal("ohne eigene USt-IdNr. gibt es keine Bestätigung")
	}
	if !strings.Contains(err.Error(), "Unternehmensdaten") {
		t.Errorf("die Meldung muss auf die Unternehmensdaten verweisen: %v", err)
	}

	// Und ein Grund hilft hier nicht: die Abfrage ist nicht fehlgeschlagen,
	// sondern noch nicht möglich.
	if err := svc.EnsureConfirmed(ctx, customer,
		"Bundeszentralamt nicht erreichbar"); err == nil {
		t.Error("der fehlende Stammdatensatz ist nicht übersteuerbar")
	}
	if *calls != 0 {
		t.Errorf("%d Aufrufe — eine Anfrage ohne eigene USt-IdNr. wird gar nicht erst gestellt",
			*calls)
	}

	// Mit der eigenen Nummer läuft die Abfrage wieder.
	env.setSettings(t, func(c *domain.CompanySettings) { c.VatID = "DE123456789" })
	if err := svc.EnsureConfirmed(ctx, customer, ""); err != nil {
		t.Errorf("mit eigener USt-IdNr. muss die Bestätigung durchgehen: %v", err)
	}
}

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

	evidence := env.supplyEvidence(t)
	invoice := env.icSupply(t, customer, "RE-2026-0001")
	view, err := evidence.View(ctx, invoice.ID, accounting.TransportBySupplier)
	if err != nil {
		t.Fatalf("Belegnachweis: %v", err)
	}
	assertNoNilSlices(t, "Belegnachweis", view)
	report, err := evidence.Report(ctx, 2026)
	if err != nil {
		t.Fatalf("Bericht Belegnachweis: %v", err)
	}
	assertNoNilSlices(t, "Bericht Belegnachweis", report)

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
		env.contactRepo, repository.NewBankRepository(env.db), env.fiscalYear))
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
