package accounting

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// -------------------------------------------------------------------------
// Vorsteuerberichtigung nach § 15a UStG
// -------------------------------------------------------------------------

func params2026(t *testing.T) TaxParameters {
	t.Helper()
	p, err := TaxParametersFor("2026-12-31")
	if err != nil {
		t.Fatalf("steuerliche Werte: %v", err)
	}
	return p
}

// Der Lehrbuchfall: ein Pkw für 47.600 € brutto, 7.600 € Vorsteuer, zunächst zu
// 100 % für abziehbare Umsätze verwendet, im dritten Jahr nur noch zu 60 %.
// Ein Fünftel der Vorsteuer entfällt auf das Jahr, und davon sind 40 % zurück-
// zuzahlen: 7.600 / 5 × 40 % = 608 €.
func TestInputTaxCorrectionForACarWithChangedUse(t *testing.T) {
	got, err := AssessInputTaxCorrection(InputTaxCorrectionRequest{
		InputTaxAmount:   760_000,
		OriginalPermille: 1000,
		CurrentPermille:  600,
		PeriodYears:      CorrectionPeriodMovableYears,
	}, params2026(t))
	if err != nil {
		t.Fatalf("Bewertung: %v", err)
	}
	if !got.Required {
		t.Fatalf("bei einer Änderung von 100 auf 60 %% ist zu berichtigen: %s", got.Reason)
	}
	if got.Amount != -60_800 {
		t.Errorf("Berichtigungsbetrag %s € — erwartet -608,00 € (7.600 / 5 × 40 %%)", got.Amount)
	}
	if got.Account != InputTaxCorrectionRepayableMovable {
		t.Errorf("Konto %s — die zurückzuzahlende Vorsteuer eines beweglichen Wirtschaftsguts "+
			"gehört auf %s", got.Account, InputTaxCorrectionRepayableMovable)
	}
	// Unter 6.000 €: nach § 44 Abs. 3 UStDV erst bei der Steuerberechnung für das
	// Kalenderjahr.
	if !got.DeferToAnnual {
		t.Error("ein Betrag unter 6.000 € wird erst zum Jahresende berichtigt (§ 44 Abs. 3 UStDV)")
	}
}

// Die Gegenrichtung: steigt die abziehbare Verwendung, wird nachträglich
// Vorsteuer abziehbar, und der Betrag geht auf das andere Konto.
func TestInputTaxCorrectionCanFavourTheTaxpayer(t *testing.T) {
	got, err := AssessInputTaxCorrection(InputTaxCorrectionRequest{
		InputTaxAmount:   760_000,
		OriginalPermille: 600,
		CurrentPermille:  1000,
		PeriodYears:      CorrectionPeriodMovableYears,
	}, params2026(t))
	if err != nil {
		t.Fatalf("Bewertung: %v", err)
	}
	if got.Amount != 60_800 {
		t.Errorf("Berichtigungsbetrag %s € — erwartet +608,00 €", got.Amount)
	}
	if got.Account != InputTaxCorrectionDeductibleMovable {
		t.Errorf("Konto %s — erwartet %s", got.Account, InputTaxCorrectionDeductibleMovable)
	}
}

// § 44 Abs. 1 UStDV: bis 1.000 € Vorsteuer je Wirtschaftsgut entfällt die
// Berichtigung ganz. § 44 Abs. 2 UStDV: unter zehn Prozentpunkten Änderung
// *und* bis 1.000 € Betrag entfällt sie für das Jahr.
func TestInputTaxCorrectionMinorLimits(t *testing.T) {
	p := params2026(t)

	small, err := AssessInputTaxCorrection(InputTaxCorrectionRequest{
		InputTaxAmount:   90_000, // 900 €
		OriginalPermille: 1000,
		CurrentPermille:  500,
		PeriodYears:      CorrectionPeriodMovableYears,
	}, p)
	if err != nil {
		t.Fatalf("Bewertung: %v", err)
	}
	if small.Required || small.Amount != 0 {
		t.Errorf("bei 900 € Vorsteuer entfällt die Berichtigung nach § 44 Abs. 1 UStDV; "+
			"erhalten: %s € (%s)", small.Amount, small.Reason)
	}

	// Acht Prozentpunkte auf eine Vorsteuer, deren Jahresanteil klein genug ist:
	// 12.500 / 5 × 8 % = 200 €.
	minor, err := AssessInputTaxCorrection(InputTaxCorrectionRequest{
		InputTaxAmount:   1_250_000,
		OriginalPermille: 1000,
		CurrentPermille:  920,
		PeriodYears:      CorrectionPeriodMovableYears,
	}, p)
	if err != nil {
		t.Fatalf("Bewertung: %v", err)
	}
	if minor.Required {
		t.Errorf("acht Prozentpunkte und 200 € bleiben unter beiden Grenzen des § 44 Abs. 2 UStDV; "+
			"erhalten: %s € (%s)", minor.Amount, minor.Reason)
	}

	// Dieselben acht Prozentpunkte, aber ein Betrag über 1.000 €: dann wird
	// berichtigt, weil § 44 Abs. 2 UStDV beide Bedingungen zusammen verlangt.
	big, err := AssessInputTaxCorrection(InputTaxCorrectionRequest{
		InputTaxAmount:   10_000_000,
		OriginalPermille: 1000,
		CurrentPermille:  920,
		PeriodYears:      CorrectionPeriodMovableYears,
	}, p)
	if err != nil {
		t.Fatalf("Bewertung: %v", err)
	}
	if !big.Required {
		t.Errorf("eine kleine Änderung mit großem Betrag wird berichtigt: %s", big.Reason)
	}
	if big.Amount != -160_000 {
		t.Errorf("Berichtigungsbetrag %s € — erwartet -1.600,00 €", big.Amount)
	}
}

// Grundstücke und Gebäude haben einen Zeitraum von zehn Jahren und eigene
// Konten.
func TestInputTaxCorrectionPeriodForBuildings(t *testing.T) {
	if got := CorrectionPeriodYears(true); got != 10 {
		t.Errorf("Berichtigungszeitraum für ein Grundstück: %d Jahre — erwartet zehn", got)
	}
	if got := CorrectionPeriodYears(false); got != 5 {
		t.Errorf("Berichtigungszeitraum für ein bewegliches Wirtschaftsgut: %d Jahre — erwartet fünf", got)
	}
	got, err := AssessInputTaxCorrection(InputTaxCorrectionRequest{
		InputTaxAmount:   19_000_00,
		OriginalPermille: 1000,
		CurrentPermille:  0,
		PeriodYears:      CorrectionPeriodImmovableYears,
		Immovable:        true,
	}, params2026(t))
	if err != nil {
		t.Fatalf("Bewertung: %v", err)
	}
	if got.Account != InputTaxCorrectionRepayableImmovable {
		t.Errorf("Konto %s — ein unbewegliches Wirtschaftsgut gehört auf %s",
			got.Account, InputTaxCorrectionRepayableImmovable)
	}
	if got.Amount != -19_000_00/10 {
		t.Errorf("Berichtigungsbetrag %s € — erwartet ein Zehntel der Vorsteuer", got.Amount)
	}
}

// -------------------------------------------------------------------------
// Belegnachweis der innergemeinschaftlichen Lieferung
// -------------------------------------------------------------------------

func TestSupplyEvidencePresumption(t *testing.T) {
	tests := []struct {
		name      string
		transport TransportKind
		items     []EvidenceItem
		fulfilled bool
	}{
		{
			name:      "zwei Beförderungsbelege unabhängiger Aussteller",
			transport: TransportBySupplier,
			items: []EvidenceItem{
				{Kind: EvidenceCMR, Issuer: "Spedition Nord", Independent: true},
				{Kind: EvidenceForwarderInvoic, Issuer: "Logistik Süd", Independent: true},
			},
			fulfilled: true,
		},
		{
			name:      "ein Beleg aus Gruppe a und einer aus Gruppe b",
			transport: TransportBySupplier,
			items: []EvidenceItem{
				{Kind: EvidenceCMR, Issuer: "Spedition Nord", Independent: true},
				{Kind: EvidenceInsurance, Issuer: "Transportversicherung AG", Independent: true},
			},
			fulfilled: true,
		},
		{
			name:      "zwei Belege derselben Partei",
			transport: TransportBySupplier,
			items: []EvidenceItem{
				{Kind: EvidenceCMR, Issuer: "Spedition Nord", Independent: true},
				{Kind: EvidenceForwarderInvoic, Issuer: "Spedition Nord", Independent: true},
			},
			fulfilled: false,
		},
		{
			name:      "Abholfall ohne Gelangensbestätigung",
			transport: TransportByCustomer,
			items: []EvidenceItem{
				{Kind: EvidenceCMR, Issuer: "Spedition Nord", Independent: true},
				{Kind: EvidenceInsurance, Issuer: "Transportversicherung AG", Independent: true},
			},
			fulfilled: false,
		},
		{
			name:      "Abholfall mit Gelangensbestätigung",
			transport: TransportByCustomer,
			items: []EvidenceItem{
				{Kind: EvidenceCMR, Issuer: "Spedition Nord", Independent: true},
				{Kind: EvidenceInsurance, Issuer: "Transportversicherung AG", Independent: true},
				{Kind: EvidenceArrival, Issuer: "Kunde SARL", Independent: false},
			},
			fulfilled: true,
		},
		{
			name:      "Rechnungsdoppel und Gelangensbestätigung nach § 17b UStDV",
			transport: TransportByCustomer,
			items: []EvidenceItem{
				{Kind: EvidenceInvoiceCopy, Issuer: "Pfennig Ventures GmbH", Independent: false},
				{Kind: EvidenceArrival, Issuer: "Kunde SARL", Independent: false},
			},
			fulfilled: true,
		},
		{
			name:      "nur ein Sendungsverfolgungsprotokoll",
			transport: TransportBySupplier,
			items: []EvidenceItem{
				{Kind: EvidenceTracking, Issuer: "Paketdienst", Independent: true},
			},
			fulfilled: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AssessSupplyEvidence(tc.transport, tc.items)
			if got.Fulfilled != tc.fulfilled {
				t.Errorf("erfüllt = %v, erwartet %v: %s", got.Fulfilled, tc.fulfilled, got.Reason)
			}
			if !got.Fulfilled && len(got.Missing) == 0 {
				t.Error("ein nicht geführter Nachweis muss sagen, was fehlt")
			}
			if got.Fulfilled && got.Basis == "" {
				t.Error("ein geführter Nachweis muss die Vorschrift nennen, auf der er beruht")
			}
		})
	}
}

// -------------------------------------------------------------------------
// Abschreibungsregeln aus der Ressource
// -------------------------------------------------------------------------

// Die Regeln werden aus afa_rules.json geladen und nicht aus Go-Literalen. Der
// Test liest dieselbe Datei und vergleicht ein Fenster — driftete die Ressource
// von der Rechnung ab, fiele es hier auf und nicht im Abschluss.
func TestAfARulesComeFromTheEmbeddedResource(t *testing.T) {
	raw, err := os.ReadFile("afa_rules.json")
	if err != nil {
		t.Fatalf("afa_rules.json: %v", err)
	}
	var fromFile AfARules
	if err := json.Unmarshal(raw, &fromFile); err != nil {
		t.Fatalf("afa_rules.json ist nicht lesbar: %v", err)
	}

	loaded := AfARuleSet()
	if loaded.Version != fromFile.Version {
		t.Errorf("Version %q geladen, %q in der Datei", loaded.Version, fromFile.Version)
	}
	if len(loaded.DegressiveWindows) != len(fromFile.DegressiveWindows) {
		t.Fatalf("%d Fenster geladen, %d in der Datei",
			len(loaded.DegressiveWindows), len(fromFile.DegressiveWindows))
	}

	// Das Fenster, das im Jahr 2026 offen ist: § 7 Abs. 2 Sätze 1 und 2 EStG mit
	// dem Dreifachen des linearen Satzes, höchstens 30 %.
	window, ok := DegressiveWindowFor("2026-05-01")
	if !ok {
		t.Fatal("für eine Anschaffung im Mai 2026 ist die degressive Abschreibung offen")
	}
	if window.FactorPermille != 3000 || window.MaxPermille != 300 {
		t.Errorf("Faktor %d, Deckel %d — erwartet 3000 und 300",
			window.FactorPermille, window.MaxPermille)
	}
	found := false
	for _, w := range fromFile.DegressiveWindows {
		if w.From == window.From && w.Until == window.Until {
			found = true
			if w.FactorPermille != window.FactorPermille || w.MaxPermille != window.MaxPermille {
				t.Errorf("die Datei nennt Faktor %d und Deckel %d, gerechnet wird mit %d und %d",
					w.FactorPermille, w.MaxPermille, window.FactorPermille, window.MaxPermille)
			}
		}
	}
	if !found {
		t.Errorf("das Fenster %s bis %s steht nicht in der Datei", window.From, window.Until)
	}

	// Und die Wertgrenzen: ab 2018 endet der Sofortabzug bei 800 €.
	params, err := AfAParametersFor("2026-01-01")
	if err != nil {
		t.Fatalf("Wertgrenzen: %v", err)
	}
	if params.GWGImmediateLimit != 80000 {
		t.Errorf("GWG-Grenze %s € — erwartet 800,00 €", params.GWGImmediateLimit)
	}
}

// Die Staffel des § 7 Abs. 2a EStG: 75, 10, 5, 5, 3 und 2 % der
// Anschaffungskosten, ohne Zeitanteil im Anschaffungsjahr, und am Ende genau
// null.
func TestElectricVehicleSchedule(t *testing.T) {
	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2026-11-15",
		Cost:                 10_000_000, // 100.000 €
		Method:               domain.DepreciationElectricVehicle,
		FiscalYearStartMonth: 1,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	want := []domain.Cents{7_500_000, 1_000_000, 500_000, 500_000, 300_000, 200_000}
	if len(rows) != len(want) {
		t.Fatalf("%d Jahre — erwartet %d", len(rows), len(want))
	}
	var total domain.Cents
	for i, row := range rows {
		if row.Amount != want[i] {
			t.Errorf("Jahr %d: %s € — erwartet %s €", row.FiscalYear, row.Amount, want[i])
		}
		if row.Months != 12 {
			t.Errorf("Jahr %d: %d Monate — die Staffel kürzt im Anschaffungsjahr nicht zeitanteilig",
				row.FiscalYear, row.Months)
		}
		total += row.Amount
	}
	if total != 10_000_000 {
		t.Errorf("Summe %s € — erwartet die vollen Anschaffungskosten", total)
	}
	if rows[len(rows)-1].ClosingBookValue != 0 {
		t.Errorf("Restbuchwert am Ende %s € — erwartet null", rows[len(rows)-1].ClosingBookValue)
	}
	if rows[0].FiscalYear != 2026 {
		t.Errorf("erstes Jahr %d — erwartet 2026", rows[0].FiscalYear)
	}
}

// Außerhalb des Fensters gibt es die Staffel nicht.
func TestElectricVehicleScheduleOutsideItsWindow(t *testing.T) {
	if _, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2024-03-01",
		Cost:                 5_000_000,
		Method:               domain.DepreciationElectricVehicle,
		FiscalYearStartMonth: 1,
	}); err == nil {
		t.Error("für eine Anschaffung im März 2024 gibt es § 7 Abs. 2a EStG noch nicht")
	}
}

// Ein Betriebsgebäude wird mit 3 % linear abgeschrieben, ohne Übergang auf eine
// Restwertabschreibung — nur das Anschaffungsjahr ist zeitanteilig.
func TestBuildingScheduleUsesTheStatutoryRate(t *testing.T) {
	rate, err := BuildingRateFor(false, "2020-06-01")
	if err != nil {
		t.Fatalf("Gebäudesatz: %v", err)
	}
	if rate.Permille != 30 {
		t.Fatalf("Satz %d Promille — erwartet 30 (3 %%)", rate.Permille)
	}

	rows, err := BuildAfASchedule(AfAPlan{
		AcquisitionDate:      "2026-07-01",
		Cost:                 100_000_000, // 1.000.000 €
		Method:               domain.DepreciationBuildingLinear,
		BuildingPermille:     rate.Permille,
		FiscalYearStartMonth: 1,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(rows) < 34 {
		t.Fatalf("%d Jahre — bei 3 %% läuft der Plan über gut 33 Jahre", len(rows))
	}
	// Im Anschaffungsjahr sechs von zwölf Monaten: 30.000 / 2 = 15.000 €.
	if rows[0].Amount != 1_500_000 || rows[0].Months != 6 {
		t.Errorf("erstes Jahr: %s € über %d Monate — erwartet 15.000,00 € über sechs Monate",
			rows[0].Amount, rows[0].Months)
	}
	if rows[1].Amount != 3_000_000 {
		t.Errorf("zweites Jahr: %s € — erwartet 30.000,00 € (3 %% von 1.000.000 €)", rows[1].Amount)
	}
	// Kein Übergang: jedes volle Jahr trägt denselben Betrag, bis der Restwert
	// aufgebraucht ist.
	if rows[10].Amount != 3_000_000 {
		t.Errorf("elftes Jahr: %s € — der feste Satz läuft unverändert weiter", rows[10].Amount)
	}
	var total domain.Cents
	for _, r := range rows {
		total += r.Amount
	}
	if total != 100_000_000 {
		t.Errorf("Summe %s € — erwartet die vollen Anschaffungskosten", total)
	}
	if rows[len(rows)-1].ClosingBookValue != 0 {
		t.Errorf("Restbuchwert am Ende %s € — erwartet null", rows[len(rows)-1].ClosingBookValue)
	}
}

// Der Investitionsabzugsbetrag des § 7g Abs. 1 EStG fehlt in Buchfink nicht, er
// gehört nicht hierher: er wird außerbilanziell abgezogen und in der
// Steuererklärung geltend gemacht. Ohne einen Satz dazu sähe die Anlagenmaske
// aus, als kenne Buchfink den § 7g nur zur Hälfte — die Sonderabschreibung des
// Absatzes 5 rechnet sie ja.
func TestAfARulesExplainTheInvestmentDeduction(t *testing.T) {
	note := AfARuleSet().InvestmentDeductionNote
	for _, want := range []string{"§ 7g Abs. 1", "außerbilanziell", "Steuererklärung", "§ 7g Abs. 2"} {
		if !strings.Contains(note, want) {
			t.Errorf("der Hinweis nennt %q nicht: %s", want, note)
		}
	}
}

// Wohngebäude folgen eigenen Sätzen, und der Stichtag ist die Fertigstellung.
func TestBuildingRatesFollowTheirReferenceDate(t *testing.T) {
	tests := []struct {
		name        string
		residential bool
		reference   string
		permille    int64
	}{
		{"Betriebsgebäude, Bauantrag 1990", false, "1990-05-01", 30},
		{"Betriebsgebäude, Bauantrag 1980", false, "1980-05-01", 20},
		// Nummer 2 gilt für jedes Gebäude, das die Voraussetzungen der Nummer 1
		// nicht erfüllt — auch für ein Betriebsgebäude. Buchstabe c gibt ihm vor
		// 1925 dieselben 2,5 % wie einem Wohngebäude; pauschal 2 % wären für ein
		// sehr altes Betriebsgebäude zu wenig.
		{"Betriebsgebäude, fertiggestellt 1910", false, "1910-05-01", 25},
		{"Wohngebäude, fertiggestellt 2024", true, "2024-03-01", 30},
		{"Wohngebäude, fertiggestellt 2010", true, "2010-03-01", 20},
		{"Wohngebäude, fertiggestellt 1910", true, "1910-03-01", 25},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rate, err := BuildingRateFor(tc.residential, tc.reference)
			if err != nil {
				t.Fatalf("Gebäudesatz: %v", err)
			}
			if rate.Permille != tc.permille {
				t.Errorf("Satz %d Promille — erwartet %d (%s)",
					rate.Permille, tc.permille, rate.Source)
			}
		})
	}
	if _, err := BuildingRateFor(false, ""); err == nil {
		t.Error("ohne Stichtag lässt sich kein Gebäudesatz bestimmen")
	}
}

// -------------------------------------------------------------------------
// Die Konten der § 4 Abs. 5 EStG-Kategorien
// -------------------------------------------------------------------------

// Der Bericht und der Gruppenkatalog müssen dieselben Konten nennen. Zwei
// Listen, die auseinanderlaufen, ergäben einen Bericht, in dem eine Buchung
// nicht vorkommt.
func TestNonDeductibleCategoriesMatchGroups(t *testing.T) {
	inCatalog := map[string]bool{}
	for _, g := range PostingGroups(domain.DirectionIncoming) {
		if g.Account != "" {
			inCatalog[g.Account] = true
		}
		if g.NonDeductibleAccount != "" {
			inCatalog[g.NonDeductibleAccount] = true
		}
	}
	for _, c := range NonDeductibleCategories() {
		for _, account := range []string{c.DeductibleAccount, c.NonDeductibleAccount} {
			if account == "" {
				continue
			}
			if !inCatalog[account] {
				t.Errorf("die Kategorie %q nennt das Konto %s, das keine Buchungsgruppe erreicht",
					c.Label, account)
			}
			if _, ok := NonDeductibleCategoryForAccount(account); !ok {
				t.Errorf("das Konto %s findet seine Kategorie nicht zurück", account)
			}
		}
	}
}

// Die Konten der beschränkt abziehbaren Betriebsausgaben sind nur über ihre
// Gruppe erreichbar — sonst ließe sich an der Aufzeichnungspflicht des
// § 4 Abs. 7 EStG vorbeibuchen.
func TestAccountsRequiringGroupCoversTheLimitedCategories(t *testing.T) {
	blocked := AccountsRequiringGroup()
	for _, account := range []string{"6610", "6620", "6625", "6640", "6644", "6645"} {
		if _, ok := blocked[account]; !ok {
			t.Errorf("das Konto %s muss über seine Gruppe erreichbar sein und nicht frei", account)
		}
	}
	// Ein gewöhnliches Aufwandskonto bleibt frei wählbar: der Notausgang für
	// Fälle, die der Katalog nicht kennt, muss offen bleiben.
	if _, ok := blocked["6815"]; ok {
		t.Error("das Konto 6815 (Bürobedarf) darf frei wählbar bleiben")
	}
}
