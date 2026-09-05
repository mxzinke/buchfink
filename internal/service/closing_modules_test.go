package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// closingModules bündelt die Bausteine der Welle 5a auf der gemeinsamen
// Testumgebung. Sie hängen alle am ClosingService, weil jeder von ihnen den
// Bilanzstichtag braucht — und untereinander über die beiden Kopplungen, die
// diese Welle eingeführt hat: die Auflösung der Abgrenzung am Saldenvortrag und
// der Verbrauch der Rückstellung am Belegweg.
type closingModules struct {
	closing       *ClosingService
	accruals      *AccrualService
	provisions    *ProvisionService
	bookings      *ClosingBookingService
	appropriation *AppropriationService
	register      *TaxRegisterService
	steps         *ClosingStepsService
	provisionRepo domain.ProvisionRepository
}

func (e *testEnv) closingModules(t *testing.T) *closingModules {
	t.Helper()
	closing := e.closing(t)

	accrualRepo := repository.NewAccrualRepository(e.db)
	provisionRepo := repository.NewProvisionRepository(e.db)
	rateRepo := repository.NewDiscountRateRepository(e.db)
	inventoryRepo := repository.NewInventoryRepository(e.db)
	notesRepo := repository.NewNotesTextRepository(e.db)
	appropriationRepo := repository.NewAppropriationRepository(e.db)
	stepRepo := repository.NewClosingStepRepository(e.db)
	settingsRepo := repository.NewSettingsRepository(e.db)
	auditRepo := repository.NewAuditRepository(e.db)

	m := &closingModules{closing: closing, provisionRepo: provisionRepo}
	m.accruals = NewAccrualService(
		accrualRepo, e.journalRepo, e.journal, settingsRepo, auditRepo, closing, e.fiscalYear)
	m.accruals.SetReceiptService(e.receipts)
	closing.SetAccrualCarrier(m.accruals)

	m.provisions = NewProvisionService(
		provisionRepo, rateRepo, e.journalRepo, e.journal, settingsRepo, auditRepo,
		closing, e.fiscalYear)
	m.provisions.SetReceiptService(e.receipts)
	e.posting.SetProvisionConsumer(m.provisions)

	m.bookings = NewClosingBookingService(
		inventoryRepo, provisionRepo, e.journalRepo, e.journal, settingsRepo, auditRepo,
		closing, e.fiscalYear)
	m.bookings.SetReceiptService(e.receipts)

	m.appropriation = NewAppropriationService(
		appropriationRepo, notesRepo, e.journalRepo, e.journal, settingsRepo, auditRepo,
		closing, e.fiscalYear)
	m.appropriation.SetReceiptService(e.receipts)
	closing.SetNotesCopier(m.appropriation)

	m.register = NewTaxRegisterService(
		repository.NewAssetRepository(e.db), provisionRepo, e.journalRepo, settingsRepo,
		closing, e.fiscalYear)

	m.steps = NewClosingStepsService(
		stepRepo, accrualRepo, provisionRepo, inventoryRepo, appropriationRepo,
		repository.NewCheckRunRepository(e.db), e.journalRepo, auditRepo, closing, e.fiscalYear)
	return m
}

// setSetting schreibt eine Einstellung, von der ein Baustein abhängt.
func (e *testEnv) setSetting(t *testing.T, key, value string) {
	t.Helper()
	if err := repository.NewSettingsRepository(e.db).Set(context.Background(), key, value); err != nil {
		t.Fatalf("Einstellung %s: %v", key, err)
	}
}

// entryByID liest eine geschriebene Buchung zurück.
func entryByID(t *testing.T, e *testEnv, id uint) *domain.JournalEntry {
	t.Helper()
	entry, err := e.journalRepo.FindByID(context.Background(), id)
	if err != nil || entry == nil {
		t.Fatalf("Buchung %d lesen: %v", id, err)
	}
	return entry
}

// taxableProfit legt ein Jahresergebnis an, auf das sich eine Steuerrückstellung
// rechnen lässt.
func (e *testEnv) taxableProfit(t *testing.T) {
	t.Helper()
	e.setSetting(t, domain.SettingTradeTaxRate, "400")
	entry := &domain.JournalEntry{
		BookingDate: "2026-06-30", DocumentDate: "2026-06-30",
		ServiceDateFrom: "2026-06-30", ServiceDateTo: "2026-06-30",
		Description: "Jahresergebnis", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 10_000_000},
			{Side: domain.SideCredit, Account: "4400", Amount: 10_000_000},
		},
	}
	if _, err := e.journal.Post(context.Background(), entry); err != nil {
		t.Fatalf("Ergebnis buchen: %v", err)
	}
}

// containsSubstring meldet, ob eine der Zeilen den Text enthält.
func containsSubstring(lines []string, want string) bool {
	for _, line := range lines {
		if strings.Contains(line, want) {
			return true
		}
	}
	return false
}

// -------------------------------------------------------------------------
// Rechnungsabgrenzung
// -------------------------------------------------------------------------

// Das Lehrbuchbeispiel von Anfang bis Ende: 1.200 € Versicherung ab dem
// 1. Dezember, 1.100 € auf 1900 abgegrenzt, im Folgejahr mit dem Saldenvortrag
// wieder aufgelöst.
func TestAccrualIsFormedAndReleasedWithTheCarryForward(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	insurance := &domain.JournalEntry{
		BookingDate: "2026-12-01", DocumentDate: "2026-12-01",
		ServiceDateFrom: "2026-12-01", ServiceDateTo: "2027-11-30",
		Description: "Betriebshaftpflicht 12/2026 bis 11/2027",
		Source:      domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "6400", Amount: 120_000},
			{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 120_000},
		},
	}
	booked, err := env.journal.Post(ctx, insurance)
	if err != nil {
		t.Fatalf("Versicherungsrechnung buchen: %v", err)
	}

	proposal, err := m.accruals.Propose(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	if len(proposal.Items) != 1 {
		t.Fatalf("erwartet einen Vorschlag, bekommen %d: %+v", len(proposal.Items), proposal.Items)
	}
	item := proposal.Items[0]
	if item.Kind != domain.AccrualActive || item.DeferredAmount != 110_000 {
		t.Errorf("Vorschlag: %s über %s € — erwartet aktive Abgrenzung über 1.100,00 €",
			item.Kind, item.DeferredAmount)
	}

	accrual, err := m.accruals.Book(ctx, AccrualRequest{
		FiscalYear: 2026, Kind: domain.AccrualActive, SourceEntryID: booked.ID,
		Text: "Betriebshaftpflicht", TotalAmount: 120_000,
		StartDate: "2026-12-01", EndDate: "2027-11-30", Account: "6400",
	})
	if err != nil {
		t.Fatalf("Abgrenzung buchen: %v", err)
	}
	if accrual.DeferredAmount != 110_000 || accrual.FormationEntryID == nil {
		t.Fatalf("Abgrenzung %+v — erwartet 1.100,00 € mit Bildungsbuchung", accrual)
	}

	// Nach der Bildung trägt das Jahr 2026 nur noch ein Zwölftel Aufwand.
	after := balances(t, env, 2026)
	if after["6400"] != 10_000 {
		t.Errorf("Aufwand 2026 auf 6400 = %s € — erwartet 100,00 €", after["6400"])
	}
	if after[domain.AccountAktiveRAP] != 110_000 {
		t.Errorf("aktive Rechnungsabgrenzung %s € — erwartet 1.100,00 €", after[domain.AccountAktiveRAP])
	}

	// Der Saldenvortrag kündigt die Auflösung an …
	preview, err := m.closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortragsvorschau: %v", err)
	}
	if len(preview.AccrualReleases) != 1 || preview.AccrualReleases[0].Amount != 110_000 {
		t.Errorf("die Vortragsvorschau nennt %d Auflösungen — erwartet eine über 1.100,00 €",
			len(preview.AccrualReleases))
	}

	// … und bucht sie mit.
	if _, err := m.closing.CarryForward(ctx, 2027); err != nil {
		t.Fatalf("Saldenvortrag: %v", err)
	}
	next := balances(t, env, 2027)
	if next["6400"] != 110_000 {
		t.Errorf("Aufwand 2027 auf 6400 = %s € — erwartet 1.100,00 €", next["6400"])
	}
	if next[domain.AccountAktiveRAP] != 0 {
		t.Errorf("die Abgrenzung steht 2027 noch mit %s € — erwartet null",
			next[domain.AccountAktiveRAP])
	}
}

// Die Vorschlagsschwelle des § 5 Abs. 5 Satz 2 EStG kennzeichnet den kleinen
// Posten, verschweigt ihn aber nicht: das Handelsrecht kennt keine Grenze.
func TestAccrualProposalMarksButKeepsSmallAmounts(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	small := &domain.JournalEntry{
		BookingDate: "2026-12-01", DocumentDate: "2026-12-01",
		ServiceDateFrom: "2026-12-01", ServiceDateTo: "2027-11-30",
		Description: "Fachzeitschrift", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "6400", Amount: 12_000},
			{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 12_000},
		},
	}
	if _, err := env.journal.Post(ctx, small); err != nil {
		t.Fatalf("Buchung: %v", err)
	}

	proposal, err := m.accruals.Propose(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	if len(proposal.Items) != 1 {
		t.Fatalf("erwartet einen Vorschlag, bekommen %d", len(proposal.Items))
	}
	if !proposal.Items[0].BelowThreshold {
		t.Error("110,00 € liegen unter der Schwelle von 800,00 € und müssen als solche gekennzeichnet sein")
	}
	if !strings.Contains(proposal.Note, "§ 250 Abs. 1 HGB") {
		t.Errorf("der Erklärtext muss sagen, dass das Handelsrecht keine Grenze kennt: %q", proposal.Note)
	}
}

// Das Disagio läuft über die Laufzeit des Darlehens und damit über mehrere
// Jahre (§ 250 Abs. 3 HGB); der Bericht zeigt den Restbetrag je Stichtag.
func TestDisagioSpreadsOverTheLoanTermAndShowsInTheReport(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	accrual, err := m.accruals.Book(ctx, AccrualRequest{
		FiscalYear: 2026, Kind: domain.AccrualDisagio, Text: "Damnum Darlehen Nr. 4711",
		TotalAmount: 300_000, StartDate: "2026-07-01", EndDate: "2029-06-30",
		Account: domain.AccountZinsaufwandLangfristig,
	})
	if err != nil {
		t.Fatalf("Disagio buchen: %v", err)
	}
	if len(accrual.Releases) != 3 {
		t.Fatalf("erwartet drei Auflösungsjahre, bekommen %d", len(accrual.Releases))
	}
	var planned domain.Cents
	for _, release := range accrual.Releases {
		planned += release.Amount
	}
	if planned != accrual.DeferredAmount {
		t.Errorf("der Auflösungsplan über %s € deckt nicht den abgegrenzten Betrag von %s €",
			planned, accrual.DeferredAmount)
	}

	report, err := m.accruals.Report(ctx, "2026-12-31")
	if err != nil {
		t.Fatalf("Bericht: %v", err)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("erwartet eine Zeile im Bericht, bekommen %d", len(report.Rows))
	}
	row := report.Rows[0]
	if row.Remaining != accrual.DeferredAmount {
		t.Errorf("Restbetrag %s € — zum Stichtag ist noch nichts aufgelöst", row.Remaining)
	}
	if row.RemainingDays <= 0 {
		t.Error("die Restlaufzeit zum Stichtag muss positiv sein")
	}
	if report.TotalActive != accrual.DeferredAmount {
		t.Errorf("Summe der aktiven Posten %s € — erwartet %s €",
			report.TotalActive, accrual.DeferredAmount)
	}
}

// -------------------------------------------------------------------------
// Rückstellungen
// -------------------------------------------------------------------------

func (m *closingModules) rate(t *testing.T, month string, years int, micros int64) {
	t.Helper()
	if err := m.provisions.SaveDiscountRates(context.Background(), []domain.DiscountRate{
		{Month: month, Years: years, RateMicros: micros, Average: 7},
	}); err != nil {
		t.Fatalf("Zinssatz pflegen: %v", err)
	}
}

// § 253 Abs. 2 Satz 1 HGB: eine Rückstellung mit einer Restlaufzeit über einem
// Jahr wird mit dem laufzeitkongruenten Durchschnittszins abgezinst. 10.000 €
// in drei Jahren zu 1,5 % sind 9.563,17 €.
func TestProvisionIsDiscountedWithTheMaintainedRate(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	m.rate(t, "2026-12", 3, 15_000)

	provision, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionUncertainLiability,
		Text: "Rückbauverpflichtung", Amount: 1_000_000, ExpectedOn: "2029-12-30",
		Reason: "Mietvertrag § 12: Rückbau der Einbauten bei Auszug",
	})
	if err != nil {
		t.Fatalf("Rückstellung bilden: %v", err)
	}
	if provision.DiscountedAmount != 956_317 {
		t.Errorf("Barwert %s € — erwartet 9.563,17 €", provision.DiscountedAmount)
	}
	if provision.Balance() != 956_317 {
		t.Errorf("Bestand %s € — erwartet den gebuchten Barwert", provision.Balance())
	}
	after := balances(t, env, 2026)
	if after[domain.AccountRueckstellungSonstige] != -956_317 {
		t.Errorf("Rückstellungskonto %s € — erwartet 9.563,17 € im Haben",
			after[domain.AccountRueckstellungSonstige])
	}
}

// Fehlt der Satz, wird nicht abgezinst und ein Befund erzeugt. Ein geratener
// Zins sähe aus wie ein echter.
func TestProvisionWithoutRateReportsAFinding(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	m.rate(t, "2026-12", 3, 15_000)

	preview, err := m.provisions.Preview(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionUncertainLiability,
		Text: "Prozessrisiko", Amount: 1_000_000, ExpectedOn: "2033-12-30",
		Reason: "Klage anhängig",
	})
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.Discounted {
		t.Error("ohne hinterlegten Satz darf nicht abgezinst werden")
	}
	if preview.Amount != 1_000_000 {
		t.Errorf("gebucht würden %s € — erwartet den vollen Erfüllungsbetrag", preview.Amount)
	}
	if len(preview.Findings) == 0 {
		t.Error("das Fehlen des Satzes muss als Befund erscheinen")
	}
}

// § 249 Abs. 2 Satz 2 HGB lässt die Auflösung nur zu, soweit der Grund
// entfallen ist. Ohne genannten Grund wird nicht aufgelöst.
func TestProvisionReleaseNeedsItsReason(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	provision, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionClosingCosts, Text: "Jahresabschluss 2026",
		Amount: 300_000, ExpectedOn: "2027-06-30", Reason: "Angebot der Kanzlei vom 12.11.2026",
	})
	if err != nil {
		t.Fatalf("Rückstellung bilden: %v", err)
	}

	if _, err := m.provisions.BookRelease(ctx, ProvisionChangeRequest{
		ProvisionID: provision.ID, Amount: 100_000, Date: "2027-03-31",
	}); err == nil {
		t.Fatal("eine Auflösung ohne Grund darf nicht durchgehen")
	}

	released, err := m.provisions.BookRelease(ctx, ProvisionChangeRequest{
		ProvisionID: provision.ID, Amount: 100_000, Date: "2027-03-31",
		Reason: "Die Kanzlei hat 1.000 € weniger berechnet als angeboten",
	})
	if err != nil {
		t.Fatalf("Auflösung: %v", err)
	}
	if released.Balance() != 200_000 {
		t.Errorf("Bestand nach Teilauflösung %s € — erwartet 2.000,00 €", released.Balance())
	}
	after := balances(t, env, 2027)
	if after[domain.AccountErtragAufloesungRueckstellungen] != -100_000 {
		t.Errorf("die Auflösung muss als Ertrag auf 4930 stehen: %s €",
			after[domain.AccountErtragAufloesungRueckstellungen])
	}
}

// Der Regelfall des Verbrauchs läuft über den Belegweg: die Rechnung wird der
// Rückstellung zugeordnet und bucht gegen sie statt gegen den Aufwand. Was sie
// übersteigt, bleibt Aufwand des laufenden Jahres.
func TestProvisionConsumptionThroughTheReceiptPathLeavesTheExcessAsExpense(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Kanzlei Habicht", "DE", "")

	provision, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionClosingCosts, Text: "Jahresabschluss 2026",
		Amount: 300_000, ExpectedOn: "2027-06-30", Reason: "Angebot der Kanzlei",
	})
	if err != nil {
		t.Fatalf("Rückstellung bilden: %v", err)
	}

	req := env.receipt(t, vendor.ID, "", 350_000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Positions = []ReceiptPosition{{
		Account: domain.AccountAbschlusskosten, Net: 350_000, TaxRate: domain.TaxRateStandard,
		Text: "Jahresabschluss 2026",
	}}
	req.ProvisionID = provision.ID
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Rechnung buchen: %v", err)
	}

	debits := map[string]domain.Cents{}
	for _, line := range entry.Lines {
		if line.Side == domain.SideDebit {
			debits[line.Account] += line.Amount
		}
	}
	if debits[domain.AccountRueckstellungAbschluss] != 300_000 {
		t.Errorf("gegen die Rückstellung gebucht: %s € — erwartet 3.000,00 €",
			debits[domain.AccountRueckstellungAbschluss])
	}
	if debits[domain.AccountAbschlusskosten] != 50_000 {
		t.Errorf("Mehraufwand %s € — erwartet 500,00 €", debits[domain.AccountAbschlusskosten])
	}
	if debits[domain.AccountVorsteuer19] != 66_500 {
		t.Errorf("Vorsteuer %s € — die Rückstellung berührt sie nicht", debits[domain.AccountVorsteuer19])
	}

	reloaded, err := m.provisionRepo.FindByID(ctx, provision.ID)
	if err != nil {
		t.Fatalf("Rückstellung laden: %v", err)
	}
	if reloaded.Balance() != 0 {
		t.Errorf("die Rückstellung steht nach dem Verbrauch auf %s € — erwartet null", reloaded.Balance())
	}
	mirror, err := m.provisions.Mirror(ctx, 2026)
	if err != nil {
		t.Fatalf("Rückstellungsspiegel: %v", err)
	}
	if mirror.Total.Used != 300_000 {
		t.Errorf("der Spiegel weist %s € Verbrauch aus — erwartet 3.000,00 €", mirror.Total.Used)
	}
	if mirror.Total.Closing != 0 {
		t.Errorf("Endbestand im Spiegel %s € — erwartet null", mirror.Total.Closing)
	}
}

// -------------------------------------------------------------------------
// Vorräte
// -------------------------------------------------------------------------

// Der Inventurwert liegt unter dem Buchwert: die Minderung ist Aufwand und
// läuft auf das Bestandsveränderungskonto des Materialaufwands.
func TestInventoryChangeCarriesTheRightSign(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	stock := &domain.JournalEntry{
		BookingDate: "2026-06-30", DocumentDate: "2026-06-30",
		ServiceDateFrom: "2026-06-30", ServiceDateTo: "2026-06-30",
		Description: "Wareneinkauf auf Bestand", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "1140", Amount: 500_000},
			{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 500_000},
		},
	}
	if _, err := env.journal.Post(ctx, stock); err != nil {
		t.Fatalf("Bestandsbuchung: %v", err)
	}
	list := env.fileIncoming(t, "inventurliste.pdf")

	preview, err := m.bookings.PreviewInventory(ctx, InventoryRequest{
		FiscalYear: 2026, Account: "1140", Amount: 400_000,
		CountedOn: "2026-12-31", Method: "Stichtagsinventur", ReceiptID: list.ID,
	})
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.Change != -100_000 || preview.ChangeAccount != domain.AccountBestandRHBWaren {
		t.Errorf("Veränderung %s € auf %s — erwartet −1.000,00 € auf %s",
			preview.Change, preview.ChangeAccount, domain.AccountBestandRHBWaren)
	}

	if _, err := m.bookings.BookInventory(ctx, InventoryRequest{
		FiscalYear: 2026, Account: "1140", Amount: 400_000,
		CountedOn: "2026-12-31", Method: "Stichtagsinventur", ReceiptID: list.ID,
	}); err != nil {
		t.Fatalf("Inventur buchen: %v", err)
	}
	after := balances(t, env, 2026)
	if after["1140"] != 400_000 {
		t.Errorf("Bestand nach der Buchung %s € — erwartet den Inventurwert von 4.000,00 €", after["1140"])
	}
	if after[domain.AccountBestandRHBWaren] != 100_000 {
		t.Errorf("Bestandsveränderung %s € — erwartet 1.000,00 € im Soll (Aufwand)",
			after[domain.AccountBestandRHBWaren])
	}
}

// Ohne Inventurliste gibt es keinen Inventurwert: die Aufnahme selbst ist der
// Beleg (§ 240 Abs. 1 HGB), und ein Bestand ohne sie ist eine Behauptung.
func TestInventoryNeedsItsCountSheet(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	if _, err := m.bookings.BookInventory(ctx, InventoryRequest{
		FiscalYear: 2026, Account: "1140", Amount: 400_000,
		CountedOn: "2026-12-31", Method: "Stichtagsinventur",
	}); err == nil {
		t.Fatal("ohne Inventurliste darf kein Inventurwert entstehen")
	}
}

// -------------------------------------------------------------------------
// Umsatzsteuer-Verrechnung
// -------------------------------------------------------------------------

// Am Jahresende werden Vorsteuer, Umsatzsteuer und Vorauszahlungen zu einem
// Saldo verrechnet — dem der Jahreserklärung. Danach stehen die Steuerkonten
// auf null.
func TestVatSettlementZeroesTheTaxAccounts(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	purchase := &domain.JournalEntry{
		BookingDate: "2026-03-10", DocumentDate: "2026-03-10",
		ServiceDateFrom: "2026-03-10", ServiceDateTo: "2026-03-10",
		Description: "Wareneinkauf", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "6300", Amount: 100_000},
			{Side: domain.SideDebit, Account: domain.AccountVorsteuer19, Amount: 19_000,
				TaxKey: "VST19", TaxBase: 100_000},
			{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 119_000},
		},
	}
	sale := &domain.JournalEntry{
		BookingDate: "2026-04-10", DocumentDate: "2026-04-10",
		ServiceDateFrom: "2026-04-10", ServiceDateTo: "2026-04-10",
		Description: "Erlös", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 238_000},
			{Side: domain.SideCredit, Account: "4400", Amount: 200_000},
			{Side: domain.SideCredit, Account: domain.AccountUmsatzsteuer19, Amount: 38_000,
				TaxKey: "UST19", TaxBase: 200_000},
		},
	}
	prepayment := &domain.JournalEntry{
		BookingDate: "2026-05-10", DocumentDate: "2026-05-10",
		ServiceDateFrom: "2026-05-10", ServiceDateTo: "2026-05-10",
		Description: "Umsatzsteuer-Vorauszahlung", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountUmsatzsteuerVorauszahlungen, Amount: 10_000},
			{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 10_000},
		},
	}
	for _, entry := range []*domain.JournalEntry{purchase, sale, prepayment} {
		if _, err := env.journal.Post(ctx, entry); err != nil {
			t.Fatalf("Buchung %q: %v", entry.Description, err)
		}
	}

	settlement, err := m.bookings.PreviewVatSettlement(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if settlement.InputTax != 19_000 || settlement.OutputTax != 38_000 || settlement.Prepaid != 10_000 {
		t.Errorf("Salden: Vorsteuer %s €, Umsatzsteuer %s €, Vorauszahlungen %s €",
			settlement.InputTax, settlement.OutputTax, settlement.Prepaid)
	}
	if settlement.Payable != 9_000 || settlement.Refund != 0 {
		t.Errorf("Zahllast %s €, Erstattung %s € — erwartet 90,00 € Zahllast",
			settlement.Payable, settlement.Refund)
	}

	if _, err := m.bookings.BookVatSettlement(ctx, 2026); err != nil {
		t.Fatalf("Verrechnung buchen: %v", err)
	}
	after := balances(t, env, 2026)
	for _, account := range []string{
		domain.AccountVorsteuer19, domain.AccountUmsatzsteuer19,
		domain.AccountUmsatzsteuerVorauszahlungen,
	} {
		if after[account] != 0 {
			t.Errorf("Konto %s steht nach der Verrechnung auf %s € — erwartet null", account, after[account])
		}
	}
	if after[domain.AccountUmsatzsteuerVorjahr] != -9_000 {
		t.Errorf("die Zahllast steht mit %s € auf %s — erwartet 90,00 € im Haben",
			after[domain.AccountUmsatzsteuerVorjahr], domain.AccountUmsatzsteuerVorjahr)
	}

	// Der Baustein gilt danach als erledigt.
	steps, err := m.steps.Steps(ctx, 2026)
	if err != nil {
		t.Fatalf("Schrittliste: %v", err)
	}
	for _, step := range steps.Steps {
		if step.Key == domain.ClosingStepVatSettlement && step.State != domain.ClosingStepDone {
			t.Errorf("der Schritt „%s\" steht auf %q — erwartet erledigt", step.Label, step.State)
		}
	}
}

// Übersteigt die Vorsteuer die Umsatzsteuer, entsteht eine Forderung.
func TestVatSettlementBooksARefundAsReceivable(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	purchase := &domain.JournalEntry{
		BookingDate: "2026-03-10", DocumentDate: "2026-03-10",
		ServiceDateFrom: "2026-03-10", ServiceDateTo: "2026-03-10",
		Description: "Große Anschaffung", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "6300", Amount: 100_000},
			{Side: domain.SideDebit, Account: domain.AccountVorsteuer19, Amount: 19_000,
				TaxKey: "VST19", TaxBase: 100_000},
			{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 119_000},
		},
	}
	if _, err := env.journal.Post(ctx, purchase); err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	if _, err := m.bookings.BookVatSettlement(ctx, 2026); err != nil {
		t.Fatalf("Verrechnung buchen: %v", err)
	}
	after := balances(t, env, 2026)
	if after[domain.AccountUmsatzsteuerforderung] != 19_000 {
		t.Errorf("die Erstattung steht mit %s € auf %s — erwartet 190,00 € im Soll",
			after[domain.AccountUmsatzsteuerforderung], domain.AccountUmsatzsteuerforderung)
	}
}

// -------------------------------------------------------------------------
// Steuerrückstellung
// -------------------------------------------------------------------------

// Das Rechenbeispiel am ganzen Weg: Ergebnis 100.000 €, nicht abziehbar
// 1.000 €, Hebesatz 400 % — und daraus zwei Rückstellungen.
func TestTaxProvisionIsComputedAndBooked(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	env.setSetting(t, domain.SettingTradeTaxRate, "400")

	result := &domain.JournalEntry{
		BookingDate: "2026-06-30", DocumentDate: "2026-06-30",
		ServiceDateFrom: "2026-06-30", ServiceDateTo: "2026-06-30",
		Description: "Jahresergebnis", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 10_000_000},
			{Side: domain.SideCredit, Account: "4400", Amount: 10_100_000},
			{Side: domain.SideDebit, Account: "6644", Amount: 100_000},
		},
	}
	if _, err := env.journal.Post(ctx, result); err != nil {
		t.Fatalf("Ergebnis buchen: %v", err)
	}

	preview, err := m.bookings.PreviewTaxProvision(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.Input.ProfitBeforeTax != 10_000_000 {
		t.Errorf("Ergebnis vor Steuern %s € — erwartet 100.000,00 €", preview.Input.ProfitBeforeTax)
	}
	if preview.Input.NonDeductible != 100_000 {
		t.Errorf("nicht abziehbar %s € — erwartet 1.000,00 €", preview.Input.NonDeductible)
	}
	if preview.CorporateTax != 1_515_000 || preview.Solidarity != 83_325 || preview.TradeTax != 1_414_000 {
		t.Errorf("Steuern: KSt %s €, SolZ %s €, GewSt %s €",
			preview.CorporateTax, preview.Solidarity, preview.TradeTax)
	}
	if preview.Warning == "" {
		t.Error("der Vorschlag muss ausdrücklich als Schätzung gekennzeichnet sein")
	}

	provisions, err := m.bookings.BookTaxProvision(ctx, TaxProvisionRequest{FiscalYear: 2026})
	if err != nil {
		t.Fatalf("Steuerrückstellung buchen: %v", err)
	}
	if len(provisions) != 2 {
		t.Fatalf("erwartet zwei Rückstellungen (Ertragsteuern und Gewerbesteuer), bekommen %d",
			len(provisions))
	}
	after := balances(t, env, 2026)
	if after[domain.AccountRueckstellungKoerperschaft] != -(1_515_000 + 83_325) {
		t.Errorf("Körperschaftsteuerrückstellung %s €", after[domain.AccountRueckstellungKoerperschaft])
	}
	if after[domain.AccountRueckstellungGewerbesteuer] != -1_414_000 {
		t.Errorf("Gewerbesteuerrückstellung %s €", after[domain.AccountRueckstellungGewerbesteuer])
	}
	if after[domain.AccountKoerperschaftsteuer] != 1_515_000 {
		t.Errorf("Körperschaftsteueraufwand %s €", after[domain.AccountKoerperschaftsteuer])
	}
}

// -------------------------------------------------------------------------
// Ergebnisverwendung
// -------------------------------------------------------------------------

// § 5a Abs. 3 GmbHG: die UG stellt ein Viertel des Jahresüberschusses in die
// gesetzliche Rücklage ein, solange das Stammkapital unter 25.000 € liegt.
func TestAppropriationEnforcesTheUGReserve(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	env.setSetting(t, "legal_form", "UG (haftungsbeschränkt)")

	// Das verwendete Jahr trägt einen Jahresüberschuss von 4.000 €.
	result := &domain.JournalEntry{
		BookingDate: "2026-06-30", DocumentDate: "2026-06-30",
		ServiceDateFrom: "2026-06-30", ServiceDateTo: "2026-06-30",
		Description: "Erlös", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 400_000},
			{Side: domain.SideCredit, Account: "4400", Amount: 400_000},
		},
	}
	if _, err := env.journal.Post(ctx, result); err != nil {
		t.Fatalf("Ergebnis buchen: %v", err)
	}

	// Das Folgejahr trägt das Ergebnis auf dem Vortragskonto und das
	// Stammkapital von 1.000 €.
	opening := &domain.JournalEntry{
		BookingDate: "2027-01-01", DocumentDate: "2027-01-01",
		ServiceDateFrom: "2027-01-01", ServiceDateTo: "2027-01-01",
		Description: "Saldenvortrag", Source: domain.EntrySourceOpening,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 500_000},
			{Side: domain.SideCredit, Account: domain.AccountGezeichnetesKapital, Amount: 100_000},
			{Side: domain.SideCredit, Account: domain.AccountGewinnvortrag, Amount: 400_000},
		},
	}
	if _, err := env.journal.Post(ctx, opening); err != nil {
		t.Fatalf("Vortrag buchen: %v", err)
	}

	// Die Vorschau ist das leere Formular: sie nennt den Pflichtbetrag und warnt,
	// statt abzubrechen. Bräche sie ab, erführe niemand, was einzustellen ist.
	empty, err := m.appropriation.PreviewAppropriation(ctx, 2026, AppropriationRequest{
		DecisionDate: "2027-05-20",
	})
	if err != nil {
		t.Fatalf("Vorschau mit leerem Formular: %v", err)
	}
	if empty.RequiredLegalReserve != 100_000 {
		t.Errorf("Pflichtrücklage der leeren Vorschau %s € — erwartet 1.000,00 €",
			empty.RequiredLegalReserve)
	}
	if !containsSubstring(empty.Warnings, "§ 5a Abs. 3 GmbHG") {
		t.Errorf("die leere Vorschau muss auf die Pflichtrücklage hinweisen, hat aber %v",
			empty.Warnings)
	}

	// Gebucht wird ein Beschluss unter der Pflichtrücklage dagegen nicht.
	if _, err := m.appropriation.BookAppropriation(ctx, 2026, AppropriationRequest{
		DecisionDate: "2027-05-20", LegalReserve: 50_000, Text: "Beschluss",
	}); err == nil {
		t.Fatal("eine zu kleine Pflichtrücklage muss zurückgewiesen werden")
	}

	preview, err := m.appropriation.PreviewAppropriation(ctx, 2026, AppropriationRequest{
		DecisionDate: "2027-05-20", LegalReserve: 100_000, Distribution: 200_000,
	})
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if containsSubstring(preview.Warnings, "§ 5a Abs. 3 GmbHG") {
		t.Errorf("eine ausreichende Rücklage darf nicht mehr gemahnt werden: %v", preview.Warnings)
	}
	if preview.RequiredLegalReserve != 100_000 {
		t.Errorf("Pflichtrücklage %s € — erwartet ein Viertel von 4.000,00 €",
			preview.RequiredLegalReserve)
	}
	// 25 % Kapitalertragsteuer auf 2.000,00 € und 5,5 % Solidaritätszuschlag
	// darauf.
	if preview.Appropriation.WithholdingTax != 50_000 {
		t.Errorf("Kapitalertragsteuer %s € — erwartet 500,00 €", preview.Appropriation.WithholdingTax)
	}
	if preview.Appropriation.SolidarityOnWithholding != 2_750 {
		t.Errorf("Solidaritätszuschlag %s € — erwartet 27,50 €",
			preview.Appropriation.SolidarityOnWithholding)
	}
	if preview.Appropriation.CarryForward != 100_000 {
		t.Errorf("Vortrag auf neue Rechnung %s € — erwartet 1.000,00 €",
			preview.Appropriation.CarryForward)
	}

	if _, err := m.appropriation.BookAppropriation(ctx, 2026, AppropriationRequest{
		DecisionDate: "2027-05-20", LegalReserve: 100_000, Distribution: 200_000,
		Text: "Gesellschafterbeschluss vom 20.05.2027",
	}); err != nil {
		t.Fatalf("Beschluss buchen: %v", err)
	}
	after := balances(t, env, 2027)
	if after[domain.AccountGesetzlicheRuecklage] != -100_000 {
		t.Errorf("gesetzliche Rücklage %s €", after[domain.AccountGesetzlicheRuecklage])
	}
	if after[domain.AccountAusschuettung] != -147_250 {
		t.Errorf("Verbindlichkeit gegenüber den Gesellschaftern %s € — erwartet 1.472,50 €",
			after[domain.AccountAusschuettung])
	}
	if after[domain.AccountKapitalertragsteuer] != -52_750 {
		t.Errorf("einbehaltene Kapitalertragsteuer %s € — erwartet 527,50 €",
			after[domain.AccountKapitalertragsteuer])
	}
	// Der Vortrag auf neue Rechnung bleibt stehen und wird nicht gebucht.
	if after[domain.AccountGewinnvortrag] != -100_000 {
		t.Errorf("Vortragskonto %s € — erwartet den Rest von 1.000,00 €",
			after[domain.AccountGewinnvortrag])
	}
}

// Wer alles vorträgt, bucht nichts: der Betrag steht bereits auf dem
// Vortragskonto.
func TestAppropriationCarryForwardNeedsNoBooking(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	opening := &domain.JournalEntry{
		BookingDate: "2027-01-01", DocumentDate: "2027-01-01",
		ServiceDateFrom: "2027-01-01", ServiceDateTo: "2027-01-01",
		Description: "Saldenvortrag", Source: domain.EntrySourceOpening,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 400_000},
			{Side: domain.SideCredit, Account: domain.AccountGewinnvortrag, Amount: 400_000},
		},
	}
	if _, err := env.journal.Post(ctx, opening); err != nil {
		t.Fatalf("Vortrag buchen: %v", err)
	}
	before := len(entriesOf(t, env, 2027))

	decision, err := m.appropriation.BookAppropriation(ctx, 2026, AppropriationRequest{
		DecisionDate: "2027-05-20", Text: "Vortrag auf neue Rechnung",
	})
	if err != nil {
		t.Fatalf("Beschluss buchen: %v", err)
	}
	if decision.JournalEntryID != nil {
		t.Error("der reine Vortrag auf neue Rechnung darf keine Buchung erzeugen")
	}
	if decision.CarryForward != 400_000 {
		t.Errorf("Vortrag %s € — erwartet 4.000,00 €", decision.CarryForward)
	}
	if got := len(entriesOf(t, env, 2027)); got != before {
		t.Errorf("das Jahr 2027 hat nach dem Beschluss %d statt %d Buchungen", got, before)
	}
}

// -------------------------------------------------------------------------
// Verzeichnis und Überleitung
// -------------------------------------------------------------------------

// § 5 Abs. 1 Satz 2 EStG verlangt ein besonderes Verzeichnis für jedes
// Wirtschaftsgut, bei dem ein steuerliches Wahlrecht ausgeübt wurde. Die
// Sonderabschreibung erscheint dort — und nur dort, weil sie nicht gebucht wird.
func TestTaxElectionRegisterListsTheSpecialDepreciation(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	assets := env.assets(t)
	ctx := context.Background()

	if _, err := assets.Save(ctx, &domain.FixedAsset{
		Name: "Fräsmaschine", Class: domain.AssetClassTangible,
		AcquisitionDate: "2026-01-15", AcquisitionCost: 10_000_000, UsefulLifeMonths: 120,
		Method: domain.DepreciationLinear, Account: "0440", DepreciationAccount: "6220",
		SpecialPermille: 400, SpecialYears: 5,
		SpecialReason: "Gewinn 2025: 140.000 €; ausschließlich betriebliche Nutzung",
	}); err != nil {
		t.Fatalf("Anlagegut: %v", err)
	}
	if _, err := assets.BookDepreciation(ctx, BookDepreciationRequest{FiscalYear: 2026}); err != nil {
		t.Fatalf("Abschreibungslauf: %v", err)
	}

	register, err := m.register.Register(ctx, 2026)
	if err != nil {
		t.Fatalf("Verzeichnis: %v", err)
	}
	if len(register.Rows) != 1 {
		t.Fatalf("erwartet eine Zeile im Verzeichnis, bekommen %d", len(register.Rows))
	}
	row := register.Rows[0]
	if row.Provision != "§ 7g Abs. 5 EStG" {
		t.Errorf("Vorschrift %q — erwartet § 7g Abs. 5 EStG", row.Provision)
	}
	if len(row.Years) != 1 {
		t.Fatalf("erwartet ein Jahr, bekommen %d", len(row.Years))
	}
	year := row.Years[0]
	if year.Commercial != 1_000_000 {
		t.Errorf("handelsrechtliche AfA %s € — erwartet 10.000,00 €", year.Commercial)
	}
	if year.Tax != 1_800_000 {
		t.Errorf("steuerliche AfA %s € — erwartet 18.000,00 €", year.Tax)
	}
	if year.Difference != 800_000 {
		t.Errorf("Differenz %s € — erwartet 8.000,00 €", year.Difference)
	}

	reconciliation, err := m.register.Reconcile(ctx, 2026)
	if err != nil {
		t.Fatalf("Überleitung: %v", err)
	}
	if len(reconciliation.Rows) != 1 {
		t.Fatalf("erwartet eine Position in der Überleitung, bekommen %d", len(reconciliation.Rows))
	}
	if reconciliation.Rows[0].Difference != -800_000 {
		t.Errorf("Überleitung Anlagevermögen %s € — erwartet −8.000,00 €",
			reconciliation.Rows[0].Difference)
	}
	if reconciliation.EquityEffect != -800_000 {
		t.Errorf("Wirkung auf das Eigenkapital %s € — erwartet −8.000,00 €",
			reconciliation.EquityEffect)
	}

	csv, err := m.register.RegisterCSV(ctx, 2026)
	if err != nil {
		t.Fatalf("CSV: %v", err)
	}
	if !strings.Contains(csv, "Fräsmaschine") || !strings.Contains(csv, "§ 7g Abs. 5 EStG") {
		t.Errorf("die CSV nennt das Wirtschaftsgut und die Vorschrift nicht:\n%s", csv)
	}
}

// Die steuerliche Abzinsung mit 5,5 % weicht vom handelsrechtlichen Satz ab;
// die Differenz gehört in die Überleitung und nicht in eine zweite Buchung.
func TestReconciliationShowsTheTaxDiscountOfProvisions(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	m.rate(t, "2026-12", 3, 15_000)

	if _, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionUncertainLiability, Text: "Rückbau",
		Amount: 1_000_000, ExpectedOn: "2029-12-30", Reason: "Mietvertrag § 12",
	}); err != nil {
		t.Fatalf("Rückstellung: %v", err)
	}

	reconciliation, err := m.register.Reconcile(ctx, 2026)
	if err != nil {
		t.Fatalf("Überleitung: %v", err)
	}
	var found *ReconciliationRow
	for i := range reconciliation.Rows {
		if strings.HasPrefix(reconciliation.Rows[i].Position, "Rückstellungen") {
			found = &reconciliation.Rows[i]
		}
	}
	if found == nil {
		t.Fatalf("die Überleitung nennt die Abzinsung nicht: %+v", reconciliation.Rows)
	}
	if found.Commercial != 956_317 {
		t.Errorf("handelsrechtlicher Wert %s € — erwartet 9.563,17 €", found.Commercial)
	}
	// 10.000 € auf drei Jahre mit 5,5 % abgezinst.
	if found.Tax >= found.Commercial {
		t.Errorf("der steuerliche Wert %s € muss unter dem handelsrechtlichen %s € liegen: "+
			"5,5 %% ist der höhere Zins", found.Tax, found.Commercial)
	}
}

// -------------------------------------------------------------------------
// Anhang und Schrittliste
// -------------------------------------------------------------------------

// Beim Anlegen des Folgejahres werden die Anhangtexte als Vorlage übernommen:
// die Bilanzierungs- und Bewertungsmethoden ändern sich selten, und ein leerer
// Anhang bleibt in der Praxis leer.
func TestNotesTextsAreCopiedIntoTheNextYear(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	const text = "Sachanlagen werden linear über die betriebsgewöhnliche Nutzungsdauer abgeschrieben."
	if _, err := m.appropriation.SaveNotesText(ctx, 2026, domain.NotesSectionMethods, text); err != nil {
		t.Fatalf("Anhangtext speichern: %v", err)
	}
	if _, err := m.closing.CreateFiscalYear(ctx, 2027); err != nil {
		t.Fatalf("Folgejahr anlegen: %v", err)
	}

	texts, err := m.appropriation.NotesTexts(ctx, 2027)
	if err != nil {
		t.Fatalf("Anhangtexte lesen: %v", err)
	}
	var copied string
	for _, section := range texts {
		if section.Section == domain.NotesSectionMethods {
			copied = section.Text
		}
	}
	if copied != text {
		t.Errorf("der Text des Vorjahres wurde nicht übernommen: %q", copied)
	}
	if len(texts) != len(domain.AllNotesSections()) {
		t.Errorf("der Anhang zeigt %d Abschnitte — erwartet alle %d, auch die leeren",
			len(texts), len(domain.AllNotesSections()))
	}
}

// Ein Schritt wird nur mit Grund übersprungen: ein Abschluss ohne
// Rückstellungen ist eine Aussage und kein Versehen.
func TestSkippingAClosingStepNeedsAReason(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	if _, err := m.steps.SkipStep(ctx, 2026, domain.ClosingStepProvisions, ""); err == nil {
		t.Fatal("ein Schritt darf nicht ohne Grund übersprungen werden")
	}

	steps, err := m.steps.SkipStep(ctx, 2026, domain.ClosingStepProvisions,
		"Keine ungewissen Verbindlichkeiten zum Stichtag; Prüfung dokumentiert im Abschlussordner")
	if err != nil {
		t.Fatalf("Schritt überspringen: %v", err)
	}
	for _, step := range steps.Steps {
		if step.Key != domain.ClosingStepProvisions {
			continue
		}
		if step.State != domain.ClosingStepSkipped {
			t.Errorf("Zustand %q — erwartet übersprungen", step.State)
		}
		if step.Reason == "" {
			t.Error("der Grund muss am Schritt stehen")
		}
	}
	if len(steps.Steps) != len(domain.AllClosingSteps()) {
		t.Errorf("die Schrittliste zeigt %d Schritte — erwartet %d",
			len(steps.Steps), len(domain.AllClosingSteps()))
	}
}

// Kein `null` in den Listen der Abschlussbausteine.
//
// Der Regelfall ist gerade der leere: ein Jahr ohne Abgrenzungsvorschlag, ein
// Bericht ohne Bestand, eine Vorschau ohne Befund. Käme dort `null` an, nähme
// `null.length` im Render den ganzen Baum mit — und zwar genau in dem Jahr, in
// dem alles in Ordnung ist.
func TestClosingModulesMarshalEmptyListsNotNull(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	env.setSetting(t, domain.SettingTradeTaxRate, "400")

	proposal, err := m.accruals.Propose(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	assertNoNullLists(t, "Abgrenzungsvorschlag", proposal, "items")

	report, err := m.accruals.Report(ctx, "2026-12-31")
	if err != nil {
		t.Fatalf("Abgrenzungsbericht: %v", err)
	}
	assertNoNullLists(t, "Abgrenzungsbericht", report, "rows")

	preview, err := m.accruals.Preview(ctx, AccrualRequest{
		FiscalYear: 2026, Kind: domain.AccrualActive, Text: "Wartungsvertrag",
		TotalAmount: 120_000, StartDate: "2026-12-01", EndDate: "2027-11-30", Account: "6400",
	})
	if err != nil {
		t.Fatalf("Abgrenzungsvorschau: %v", err)
	}
	assertNoNullLists(t, "Abgrenzungsvorschau", preview, "lines", "releases", "warnings")

	provisionPreview, err := m.provisions.Preview(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionClosingCosts, Text: "Jahresabschluss",
		Amount: 300_000, ExpectedOn: "2027-06-30", Reason: "Angebot der Kanzlei",
	})
	if err != nil {
		t.Fatalf("Rückstellungsvorschau: %v", err)
	}
	assertNoNullLists(t, "Rückstellungsvorschau", provisionPreview, "lines", "findings")
	// Auch die Liste im eingebetteten Objekt: eine neu gebildete Rückstellung
	// hat noch keine Bewegung, und `provision.movements.map` nähme als `null`
	// im Render den ganzen Baum mit.
	assertNoNullLists(t, "Rückstellung der Vorschau", provisionPreview.Provision, "movements")

	inventory, err := m.bookings.InventoryAccounts(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorräte: %v", err)
	}
	assertNoNullLists(t, "Vorräte", inventory, "accounts")

	settlement, err := m.bookings.PreviewVatSettlement(ctx, 2026)
	if err != nil {
		t.Fatalf("Umsatzsteuer-Verrechnung: %v", err)
	}
	assertNoNullLists(t, "Umsatzsteuer-Verrechnung", settlement, "rows", "lines")

	taxPreview, err := m.bookings.PreviewTaxProvision(ctx, 2026)
	if err != nil {
		t.Fatalf("Steuerrückstellung: %v", err)
	}
	assertNoNullLists(t, "Steuerrückstellung", taxPreview, "lines")

	steps, err := m.steps.Steps(ctx, 2026)
	if err != nil {
		t.Fatalf("Schrittliste: %v", err)
	}
	assertNoNullLists(t, "Abschlussschritte", steps, "steps")

	register, err := m.register.Register(ctx, 2026)
	if err != nil {
		t.Fatalf("Verzeichnis: %v", err)
	}
	assertNoNullLists(t, "Verzeichnis § 5 Abs. 1 Satz 2 EStG", register, "rows")

	reconciliation, err := m.register.Reconcile(ctx, 2026)
	if err != nil {
		t.Fatalf("Überleitung: %v", err)
	}
	assertNoNullLists(t, "Überleitung", reconciliation, "rows")

	carryForward, err := m.closing.CarryForwardState(ctx, 2027)
	if err != nil {
		t.Fatalf("Vortragsvorschau: %v", err)
	}
	assertNoNullLists(t, "Vortragsvorschau", carryForward, "accrualReleases")

	mirror, err := m.provisions.Mirror(ctx, 2026)
	if err != nil {
		t.Fatalf("Rückstellungsspiegel: %v", err)
	}
	assertNoNullLists(t, "Rückstellungsspiegel", mirror, "rows")
}

// -------------------------------------------------------------------------
// Nachgereichte Prüfungen der zweiten Runde
// -------------------------------------------------------------------------

// Die Zuführung eines späteren Jahres ist eine Bewegung dieses Jahres. Läge sie
// im Bildungsjahr, ginge der Rückstellungsspiegel beider Jahre nicht mehr auf.
func TestProvisionIncreaseBelongsToTheYearItIsBookedIn(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	provision, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionClosingCosts, Text: "Jahresabschluss 2026",
		Amount: 300_000, ExpectedOn: "2027-06-30", Reason: "Angebot der Kanzlei vom 12.11.2026",
	})
	if err != nil {
		t.Fatalf("Rückstellung bilden: %v", err)
	}

	// Ohne eigene Begründung geht die Zuführung nicht durch: sie ist eine
	// geänderte Schätzung.
	if _, err := m.provisions.BookIncrease(ctx, ProvisionRequest{
		ProvisionID: provision.ID, FiscalYear: 2027, Amount: 100_000, Date: "2027-03-31",
	}); err == nil {
		t.Fatal("eine Zuführung ohne Begründung darf nicht durchgehen")
	}

	increased, err := m.provisions.BookIncrease(ctx, ProvisionRequest{
		ProvisionID: provision.ID, FiscalYear: 2027, Amount: 100_000, Date: "2027-03-31",
		Reason: "Die Kanzlei hat den Aufwand für die E-Bilanz nachberechnet",
	})
	if err != nil {
		t.Fatalf("Zuführung buchen: %v", err)
	}
	if increased.SettlementAmount != 400_000 {
		t.Errorf("Erfüllungsbetrag nach der Zuführung %s € — erwartet 4.000,00 €",
			increased.SettlementAmount)
	}
	if increased.FiscalYear != 2026 {
		t.Errorf("die Rückstellung wandert ins Jahr %d — sie bleibt im Bildungsjahr 2026",
			increased.FiscalYear)
	}
	var found bool
	for _, movement := range increased.Movements {
		if movement.Kind != domain.ProvisionIncrease {
			continue
		}
		found = true
		if movement.FiscalYear != 2027 {
			t.Errorf("die Zuführung steht im Geschäftsjahr %d — gebucht wurde sie am %s",
				movement.FiscalYear, movement.Date)
		}
	}
	if !found {
		t.Fatal("die Zuführung hat keine Bewegung erzeugt")
	}

	// Der Spiegel 2027: Anfangsbestand ist die Bildung, die Zuführung steht in
	// ihrer eigenen Spalte, und der Endbestand geht auf.
	mirror, err := m.provisions.Mirror(ctx, 2027)
	if err != nil {
		t.Fatalf("Rückstellungsspiegel 2027: %v", err)
	}
	if mirror.Total.Opening != 300_000 {
		t.Errorf("Anfangsbestand 2027 %s € — erwartet 3.000,00 €", mirror.Total.Opening)
	}
	if mirror.Total.Additions != 100_000 {
		t.Errorf("Zuführung 2027 %s € — erwartet 1.000,00 €", mirror.Total.Additions)
	}
	if mirror.Total.Closing != 400_000 {
		t.Errorf("Endbestand 2027 %s € — erwartet 4.000,00 €", mirror.Total.Closing)
	}

	// Und der Spiegel 2026 kennt die Zuführung des Folgejahres nicht.
	prior, err := m.provisions.Mirror(ctx, 2026)
	if err != nil {
		t.Fatalf("Rückstellungsspiegel 2026: %v", err)
	}
	if prior.Total.Additions != 300_000 || prior.Total.Closing != 300_000 {
		t.Errorf("Spiegel 2026: Zuführung %s €, Endbestand %s € — erwartet je 3.000,00 €",
			prior.Total.Additions, prior.Total.Closing)
	}
}

// Scheitert die Buchung, entsteht keine Rückstellung. Sonst bliebe eine
// Rückstellung ohne Bewegung stehen — und der Abschlussassistent meldete den
// Schritt als erledigt.
func TestFailedProvisionBookingLeavesNoProvision(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	if _, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionUncertainLiability, Text: "Prozessrisiko",
		Amount: 500_000, ExpectedOn: "2027-06-30", Reason: "Klage anhängig",
		ExpenseAccount: "9999",
	}); err == nil {
		t.Fatal("eine Buchung auf ein unbekanntes Konto muss scheitern")
	}

	stored, err := m.provisionRepo.FindAll(ctx)
	if err != nil {
		t.Fatalf("Rückstellungen lesen: %v", err)
	}
	if len(stored) != 0 {
		t.Fatalf("nach der gescheiterten Buchung stehen %d Rückstellungen in der Kartei", len(stored))
	}

	steps, err := m.steps.Steps(ctx, 2026)
	if err != nil {
		t.Fatalf("Schrittliste: %v", err)
	}
	for _, step := range steps.Steps {
		if step.Key == domain.ClosingStepProvisions && step.State != domain.ClosingStepOpen {
			t.Errorf("der Schritt „%s\" steht auf %q — erwartet offen", step.Label, step.State)
		}
	}
}

// Die Steuerrückstellung ist ein eigener Schritt. Sie darf den Schritt
// „Rückstellungen" nicht miterledigen.
func TestTaxProvisionDoesNotCloseTheProvisionStep(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	env.setSetting(t, domain.SettingTradeTaxRate, "400")

	profit := &domain.JournalEntry{
		BookingDate: "2026-06-30", DocumentDate: "2026-06-30",
		ServiceDateFrom: "2026-06-30", ServiceDateTo: "2026-06-30",
		Description: "Erlös", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 10_000_000},
			{Side: domain.SideCredit, Account: "4400", Amount: 10_000_000},
		},
	}
	if _, err := env.journal.Post(ctx, profit); err != nil {
		t.Fatalf("Ergebnis buchen: %v", err)
	}
	if _, err := m.bookings.BookTaxProvision(ctx, TaxProvisionRequest{FiscalYear: 2026}); err != nil {
		t.Fatalf("Steuerrückstellung buchen: %v", err)
	}

	steps, err := m.steps.Steps(ctx, 2026)
	if err != nil {
		t.Fatalf("Schrittliste: %v", err)
	}
	for _, step := range steps.Steps {
		switch step.Key {
		case domain.ClosingStepProvisions:
			if step.State != domain.ClosingStepOpen {
				t.Errorf("der Schritt „Rückstellungen\" steht auf %q — die Steuerrückstellung "+
					"erledigt ihn nicht", step.State)
			}
		case domain.ClosingStepTaxProvision:
			if step.State != domain.ClosingStepDone {
				t.Errorf("der Schritt „Steuerrückstellung\" steht auf %q — erwartet erledigt", step.State)
			}
		}
	}
}

// Ein zweiter Lauf der Steuerrückstellung bildet sie nicht ein zweites Mal —
// und die Vorschau hält die eigene Buchung nicht für eine Vorauszahlung.
func TestTaxProvisionIsBookedOnlyOnce(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	env.setSetting(t, domain.SettingTradeTaxRate, "400")

	result := &domain.JournalEntry{
		BookingDate: "2026-06-30", DocumentDate: "2026-06-30",
		ServiceDateFrom: "2026-06-30", ServiceDateTo: "2026-06-30",
		Description: "Jahresergebnis", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 10_000_000},
			{Side: domain.SideCredit, Account: "4400", Amount: 10_000_000},
		},
	}
	if _, err := env.journal.Post(ctx, result); err != nil {
		t.Fatalf("Ergebnis buchen: %v", err)
	}
	// Eine echte Vorauszahlung — sie mindert die Rückstellung.
	prepayment := &domain.JournalEntry{
		BookingDate: "2026-09-10", DocumentDate: "2026-09-10",
		ServiceDateFrom: "2026-09-10", ServiceDateTo: "2026-09-10",
		Description: "Körperschaftsteuer-Vorauszahlung", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountKoerperschaftsteuer, Amount: 500_000},
			{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 500_000},
		},
	}
	if _, err := env.journal.Post(ctx, prepayment); err != nil {
		t.Fatalf("Vorauszahlung buchen: %v", err)
	}

	preview, err := m.bookings.PreviewTaxProvision(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.Input.PrepaidCorporate != 500_000 {
		t.Errorf("Vorauszahlung %s € — erwartet 5.000,00 €", preview.Input.PrepaidCorporate)
	}

	if _, err := m.bookings.BookTaxProvision(ctx, TaxProvisionRequest{FiscalYear: 2026}); err != nil {
		t.Fatalf("Steuerrückstellung buchen: %v", err)
	}
	// Nach der Buchung steht der Steueraufwand auf denselben Konten. Er darf
	// die Vorauszahlung nicht aufblähen.
	after, err := m.bookings.PreviewTaxProvision(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorschau nach der Buchung: %v", err)
	}
	if after.Input.PrepaidCorporate != 500_000 {
		t.Errorf("nach der Buchung gelten %s € als Vorauszahlung — erwartet unverändert 5.000,00 €",
			after.Input.PrepaidCorporate)
	}

	if _, err := m.bookings.BookTaxProvision(ctx, TaxProvisionRequest{
		FiscalYear: 2026, IncomeProvision: 100_000, TradeProvision: 100_000,
	}); err == nil {
		t.Fatal("die Steuerrückstellung darf nicht zweimal gebildet werden")
	}
	provisions, err := m.provisionRepo.FindByYear(ctx, 2026)
	if err != nil {
		t.Fatalf("Rückstellungen lesen: %v", err)
	}
	count := 0
	for _, p := range provisions {
		if p.Kind.IsTax() {
			count++
		}
	}
	if count != 2 {
		t.Errorf("%d Steuerrückstellungen — erwartet je eine für Ertrag- und Gewerbesteuer", count)
	}
}

// Zu einer Buchung gehört höchstens eine Abgrenzung: ein zweiter Griff in die
// Vorschlagsliste bucht sonst 1900 ein zweites Mal.
func TestAccrualIsBookedOnlyOncePerSourceEntry(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	insurance := &domain.JournalEntry{
		BookingDate: "2026-12-01", DocumentDate: "2026-12-01",
		ServiceDateFrom: "2026-12-01", ServiceDateTo: "2027-11-30",
		Description: "Betriebshaftpflicht", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "6400", Amount: 120_000},
			{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 120_000},
		},
	}
	booked, err := env.journal.Post(ctx, insurance)
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	req := AccrualRequest{
		FiscalYear: 2026, Kind: domain.AccrualActive, SourceEntryID: booked.ID,
		Text: "Betriebshaftpflicht", TotalAmount: 120_000,
		StartDate: "2026-12-01", EndDate: "2027-11-30", Account: "6400",
	}
	if _, err := m.accruals.Book(ctx, req); err != nil {
		t.Fatalf("Abgrenzung buchen: %v", err)
	}
	if _, err := m.accruals.Book(ctx, req); err == nil {
		t.Fatal("zu derselben Buchung darf keine zweite Abgrenzung entstehen")
	}
	after := balances(t, env, 2026)
	if after[domain.AccountAktiveRAP] != 110_000 {
		t.Errorf("aktive Rechnungsabgrenzung %s € — erwartet einmal 1.100,00 €",
			after[domain.AccountAktiveRAP])
	}
}

// Ein zweiter Beschluss zum selben Jahr wird abgewiesen, solange die erste
// Buchung steht.
func TestAppropriationIsDecidedOnlyOnce(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	opening := &domain.JournalEntry{
		BookingDate: "2027-01-01", DocumentDate: "2027-01-01",
		ServiceDateFrom: "2027-01-01", ServiceDateTo: "2027-01-01",
		Description: "Saldenvortrag", Source: domain.EntrySourceOpening,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 400_000},
			{Side: domain.SideCredit, Account: domain.AccountGewinnvortrag, Amount: 400_000},
		},
	}
	if _, err := env.journal.Post(ctx, opening); err != nil {
		t.Fatalf("Vortrag buchen: %v", err)
	}

	if _, err := m.appropriation.BookAppropriation(ctx, 2026, AppropriationRequest{
		DecisionDate: "2027-05-20", Distribution: 100_000, Text: "Beschluss",
	}); err != nil {
		t.Fatalf("Beschluss buchen: %v", err)
	}
	before := len(entriesOf(t, env, 2027))

	if _, err := m.appropriation.BookAppropriation(ctx, 2026, AppropriationRequest{
		DecisionDate: "2027-06-20", Distribution: 100_000, Text: "Zweiter Beschluss",
	}); err == nil {
		t.Fatal("ein zweiter Beschluss zum selben Jahr muss abgewiesen werden")
	}
	if got := len(entriesOf(t, env, 2027)); got != before {
		t.Errorf("das Jahr 2027 hat nach dem zweiten Versuch %d statt %d Buchungen", got, before)
	}

	// Und ein Beschlussdatum außerhalb des Buchungsjahres gehört zurückgewiesen.
	if _, err := m.appropriation.PreviewAppropriation(ctx, 2026, AppropriationRequest{
		DecisionDate: "2028-05-20",
	}); err == nil {
		t.Error("ein Beschlussdatum außerhalb des Buchungsjahres muss abgewiesen werden")
	}
}

// § 5a Abs. 3 GmbHG bemisst die Pflichtrücklage am Jahresüberschuss und nicht am
// Saldo des Vortragskontos: 3.000 € Gewinnvortrag aus 2025 und 1.000 €
// Jahresüberschuss 2026 ergeben 250 € und nicht 1.000 €.
func TestUGReserveCountsOnlyTheYearsResult(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	env.setSetting(t, "legal_form", "Unternehmergesellschaft (haftungsbeschränkt)")

	profit := &domain.JournalEntry{
		BookingDate: "2026-06-30", DocumentDate: "2026-06-30",
		ServiceDateFrom: "2026-06-30", ServiceDateTo: "2026-06-30",
		Description: "Erlös 2026", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 100_000},
			{Side: domain.SideCredit, Account: "4400", Amount: 100_000},
		},
	}
	if _, err := env.journal.Post(ctx, profit); err != nil {
		t.Fatalf("Ergebnis buchen: %v", err)
	}
	// Der Vortrag ins Jahr 2027 bringt den Gewinnvortrag aus 2025 mit.
	opening := &domain.JournalEntry{
		BookingDate: "2027-01-01", DocumentDate: "2027-01-01",
		ServiceDateFrom: "2027-01-01", ServiceDateTo: "2027-01-01",
		Description: "Saldenvortrag", Source: domain.EntrySourceOpening,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 500_000},
			{Side: domain.SideCredit, Account: domain.AccountGezeichnetesKapital, Amount: 100_000},
			{Side: domain.SideCredit, Account: domain.AccountGewinnvortrag, Amount: 400_000},
		},
	}
	if _, err := env.journal.Post(ctx, opening); err != nil {
		t.Fatalf("Vortrag buchen: %v", err)
	}

	preview, err := m.appropriation.PreviewAppropriation(ctx, 2026, AppropriationRequest{
		DecisionDate: "2027-05-20", LegalReserve: 25_000,
	})
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.YearResult != 100_000 {
		t.Errorf("Jahresüberschuss 2026 %s € — erwartet 1.000,00 €", preview.YearResult)
	}
	if preview.NetIncome != 400_000 {
		t.Errorf("verwendbares Ergebnis %s € — erwartet 4.000,00 € vom Vortragskonto", preview.NetIncome)
	}
	if preview.RequiredLegalReserve != 25_000 {
		t.Errorf("Pflichtrücklage %s € — erwartet ein Viertel von 1.000,00 €",
			preview.RequiredLegalReserve)
	}
}

// Die Zunahme des Bestands ist ein Ertrag: SOLL Bestandskonto, HABEN
// Bestandsveränderung fertige Erzeugnisse (§ 275 Abs. 2 Nr. 2 HGB).
func TestInventoryIncreaseBooksAsIncome(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	list := env.fileIncoming(t, "inventurliste-fertige.pdf")

	preview, err := m.bookings.PreviewInventory(ctx, InventoryRequest{
		FiscalYear: 2026, Account: "1110", Amount: 250_000,
		CountedOn: "2026-12-31", Method: "Stichtagsinventur", ReceiptID: list.ID,
	})
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.Change != 250_000 || preview.ChangeAccount != domain.AccountBestandFertige {
		t.Fatalf("Veränderung %s € auf %s — erwartet 2.500,00 € auf %s",
			preview.Change, preview.ChangeAccount, domain.AccountBestandFertige)
	}
	if len(preview.Lines) != 2 ||
		preview.Lines[0].Side != domain.SideDebit || preview.Lines[0].Account != "1110" ||
		preview.Lines[1].Side != domain.SideCredit ||
		preview.Lines[1].Account != domain.AccountBestandFertige {
		t.Fatalf("Buchungssatz %+v — erwartet SOLL 1110 an HABEN %s",
			preview.Lines, domain.AccountBestandFertige)
	}

	req := InventoryRequest{
		FiscalYear: 2026, Account: "1110", Amount: 250_000,
		CountedOn: "2026-12-31", Method: "Stichtagsinventur", ReceiptID: list.ID,
	}
	if _, err := m.bookings.BookInventory(ctx, req); err != nil {
		t.Fatalf("Inventur buchen: %v", err)
	}
	after := balances(t, env, 2026)
	if after["1110"] != 250_000 {
		t.Errorf("Bestand %s € — erwartet den Inventurwert von 2.500,00 €", after["1110"])
	}
	if after[domain.AccountBestandFertige] != -250_000 {
		t.Errorf("Bestandsveränderung %s € — erwartet 2.500,00 € im Haben (Ertrag)",
			after[domain.AccountBestandFertige])
	}

	// Ein zweiter Inventurwert zum selben Konto und Jahr wird abgewiesen; sonst
	// stünden zwei Aufnahmen nebeneinander.
	if _, err := m.bookings.BookInventory(ctx, req); err == nil {
		t.Fatal("ein zweiter Inventurwert für dasselbe Konto und Jahr muss abgewiesen werden")
	}
	overview, err := m.bookings.InventoryAccounts(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorräte: %v", err)
	}
	counted := 0
	for _, account := range overview.Accounts {
		if account.Account == "1110" && account.Counted > 0 {
			counted++
		}
	}
	if counted != 1 {
		t.Errorf("das Konto 1110 erscheint %d-mal mit Inventurwert — erwartet einmal", counted)
	}

	// Und eine Inventurliste, die es nicht gibt, wird abgewiesen.
	if _, err := m.bookings.BookInventory(ctx, InventoryRequest{
		FiscalYear: 2026, Account: "1140", Amount: 100_000,
		CountedOn: "2026-12-31", Method: "Stichtagsinventur", ReceiptID: 987_654,
	}); err == nil {
		t.Fatal("ein Belegverweis ins Leere darf nicht als Inventurliste gelten")
	}
}

// Der Schritt „Prüfbericht" folgt aus dem letzten Prüflauf des Jahres — er ist
// als abgeleitet definiert und muss sich auch ableiten lassen.
func TestCheckRunClosesItsClosingStep(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	checks := env.checks(t)
	ctx := context.Background()

	steps, err := m.steps.Steps(ctx, 2026)
	if err != nil {
		t.Fatalf("Schrittliste: %v", err)
	}
	for _, step := range steps.Steps {
		if step.Key == domain.ClosingStepCheckRun && step.State != domain.ClosingStepOpen {
			t.Errorf("ohne Prüflauf steht der Schritt auf %q — erwartet offen", step.State)
		}
	}

	if _, err := checks.Run(ctx, CheckRequest{CutoffDate: "2026-12-31", PeriodType: "year"}); err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}

	steps, err = m.steps.Steps(ctx, 2026)
	if err != nil {
		t.Fatalf("Schrittliste: %v", err)
	}
	for _, step := range steps.Steps {
		if step.Key != domain.ClosingStepCheckRun {
			continue
		}
		if step.State != domain.ClosingStepDone {
			t.Errorf("nach dem Prüflauf steht der Schritt auf %q — erwartet erledigt (%s)",
				step.State, step.Detail)
		}
		if step.Detail == "" {
			t.Error("der Schritt nennt den Prüflauf nicht")
		}
	}
}

// Das Verzeichnis führt die Jahre nebeneinander: im Begünstigungszeitraum liegt
// die steuerliche AfA über der handelsrechtlichen, danach kehrt sich die
// Differenz um (§ 7a Abs. 9 EStG). Über die Nutzungsdauer hebt sie sich auf.
func TestTaxElectionRegisterCarriesEveryYear(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	assets := env.assets(t)
	ctx := context.Background()

	if _, err := assets.Save(ctx, &domain.FixedAsset{
		Name: "Fräsmaschine", Class: domain.AssetClassTangible,
		AcquisitionDate: "2026-01-15", AcquisitionCost: 10_000_000, UsefulLifeMonths: 120,
		Method: domain.DepreciationLinear, Account: "0440", DepreciationAccount: "6220",
		SpecialPermille: 400, SpecialYears: 5,
		SpecialReason: "Gewinn 2025: 140.000 €; ausschließlich betriebliche Nutzung",
	}); err != nil {
		t.Fatalf("Anlagegut: %v", err)
	}
	for year := 2026; year <= 2031; year++ {
		assets.SetFiscalYear(year)
		if _, err := assets.BookDepreciation(ctx, BookDepreciationRequest{FiscalYear: year}); err != nil {
			t.Fatalf("Abschreibungslauf %d: %v", year, err)
		}
	}

	m.register.SetFiscalYear(2031)
	register, err := m.register.Register(ctx, 2031)
	if err != nil {
		t.Fatalf("Verzeichnis: %v", err)
	}
	if len(register.Rows) != 1 {
		t.Fatalf("erwartet eine Zeile im Verzeichnis, bekommen %d", len(register.Rows))
	}
	row := register.Rows[0]
	if len(row.Years) != 6 {
		t.Fatalf("erwartet sechs Jahre im Verzeichnis, bekommen %d", len(row.Years))
	}
	for i, year := range row.Years {
		if year.Commercial != 1_000_000 {
			t.Errorf("%d: handelsrechtliche AfA %s € — erwartet unverändert 10.000,00 €",
				year.FiscalYear, year.Commercial)
		}
		want := domain.Cents(1_800_000)
		if i == 5 {
			// Nach dem Begünstigungszeitraum verteilt sich der steuerliche
			// Restwert von 10.000 € auf die fünf verbliebenen Jahre.
			want = 200_000
		}
		if year.Tax != want {
			t.Errorf("%d: steuerliche AfA %s € — erwartet %s €", year.FiscalYear, year.Tax, want)
		}
		if year.Difference != year.Tax-year.Commercial {
			t.Errorf("%d: Differenz %s € passt nicht zu %s € und %s €",
				year.FiscalYear, year.Difference, year.Tax, year.Commercial)
		}
	}
	if row.TotalCommercial != 6_000_000 {
		t.Errorf("Summe handelsrechtlich %s € — erwartet 60.000,00 €", row.TotalCommercial)
	}
	if row.TotalDifference != 3_200_000 {
		t.Errorf("Summe der Differenzen %s € — erwartet 32.000,00 €", row.TotalDifference)
	}

	csv, err := m.register.RegisterCSV(ctx, 2031)
	if err != nil {
		t.Fatalf("CSV: %v", err)
	}
	if lines := strings.Count(csv, "\n"); lines != 7 {
		t.Errorf("die CSV hat %d Zeilen — erwartet Kopf und sechs Jahre", lines)
	}
}

// Der Anhang gehört in den Jahresabschluss: Rückstellungsspiegel, Überleitung
// und Freitexte kommen mit der Bilanz, nicht daneben.
func TestFinancialStatementCarriesTheNotes(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	statements := env.statements(t)
	statements.SetNotesSources(NotesSources{
		Provisions: m.provisions, Reconciliation: m.register, Texts: m.appropriation,
	})
	ctx := context.Background()
	m.rate(t, "2026-12", 3, 15_000)

	if _, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionUncertainLiability, Text: "Rückbau",
		Amount: 1_000_000, ExpectedOn: "2029-12-31", Reason: "Mietvertrag § 12",
	}); err != nil {
		t.Fatalf("Rückstellung: %v", err)
	}
	const method = "Sachanlagen werden linear über die Nutzungsdauer abgeschrieben."
	if _, err := m.appropriation.SaveNotesText(ctx, 2026, domain.NotesSectionMethods, method); err != nil {
		t.Fatalf("Anhangtext: %v", err)
	}

	fs, err := statements.Build(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatalf("Jahresabschluss: %v", err)
	}
	if len(fs.Notes.ProvisionMirror.Rows) != 1 {
		t.Fatalf("der Anhang trägt %d Zeilen im Rückstellungsspiegel — erwartet eine",
			len(fs.Notes.ProvisionMirror.Rows))
	}
	if fs.Notes.ProvisionMirror.Total.Closing != 956_317 {
		t.Errorf("Endbestand im Spiegel %s € — erwartet den Barwert von 9.563,17 €",
			fs.Notes.ProvisionMirror.Total.Closing)
	}
	if len(fs.Notes.Reconciliation.Rows) == 0 {
		t.Fatal("der Anhang trägt die Überleitungsrechnung nicht")
	}
	var text string
	for _, section := range fs.Notes.Texts {
		if section.Section == domain.NotesSectionMethods {
			text = section.Text
		}
	}
	if text != method {
		t.Errorf("der Anhangtext fehlt im Abschluss: %q", text)
	}

	// Und beide Ausgabewege zeigen ihn.
	csv, err := statements.ExportCSV(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatalf("CSV: %v", err)
	}
	for _, want := range []string{"rueckstellungsspiegel", "ueberleitung", method} {
		if !strings.Contains(csv, want) {
			t.Errorf("die CSV enthält %q nicht", want)
		}
	}
	typst := statementTypst(fs)
	for _, want := range []string{"= Anhang", "Rückstellungsspiegel", "Überleitung zur Steuerbilanz"} {
		if !strings.Contains(typst, want) {
			t.Errorf("das Dokument enthält %q nicht", want)
		}
	}
}

// Fehlt die Zinstabelle des Stichtagsmonats, rechnet Buchfink mit dem jüngsten
// älteren Monat — sagt aber, mit welchem, und meldet es dem Prüflauf.
func TestDiscountRateFromAnOlderMonthIsNamedAndReported(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	checks := env.checks(t)
	checks.SetProvisionSource(m.provisions)
	ctx := context.Background()
	m.rate(t, "2026-09", 3, 15_000)

	preview, err := m.provisions.Preview(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionUncertainLiability, Text: "Rückbau",
		Amount: 1_000_000, ExpectedOn: "2029-12-31", Reason: "Mietvertrag § 12",
	})
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if !preview.Discounted || preview.DiscountMonth != "2026-09" {
		t.Fatalf("abgezinst: %v mit der Tabelle vom %q — erwartet den Rückgriff auf 2026-09",
			preview.Discounted, preview.DiscountMonth)
	}
	if len(preview.Findings) != 1 || !strings.Contains(preview.Findings[0], "2026-09") {
		t.Errorf("der Befund nennt den verwendeten Monat nicht: %v", preview.Findings)
	}

	if _, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionUncertainLiability, Text: "Rückbau",
		Amount: 1_000_000, ExpectedOn: "2029-12-31", Reason: "Mietvertrag § 12",
	}); err != nil {
		t.Fatalf("Rückstellung bilden: %v", err)
	}

	run, err := checks.Preview(ctx, CheckRequest{CutoffDate: "2026-12-31", PeriodType: "year"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	var found bool
	for _, finding := range run.Findings {
		if finding.Rule == domain.CheckRuleProvisionDiscount {
			found = true
			if !strings.Contains(finding.Message, "2026-09") {
				t.Errorf("der Befund nennt den verwendeten Monat nicht: %q", finding.Message)
			}
		}
	}
	if !found {
		t.Errorf("der Prüflauf meldet die Abzinsung nicht: %+v", run.Findings)
	}
}

// Erledigen löst den Rest auf und schließt die Rückstellung — mit Grund, wie
// jede Auflösung (§ 249 Abs. 2 Satz 2 HGB).
func TestSettlingAProvisionReleasesTheRest(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	provision, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionClosingCosts, Text: "Jahresabschluss 2026",
		Amount: 300_000, ExpectedOn: "2027-06-30", Reason: "Angebot der Kanzlei",
	})
	if err != nil {
		t.Fatalf("Rückstellung bilden: %v", err)
	}
	if _, err := m.provisions.Settle(ctx, provision.ID, "2027-06-30", ""); err == nil {
		t.Fatal("ohne Grund darf nichts erledigt werden")
	}

	settled, err := m.provisions.Settle(ctx, provision.ID, "2027-06-30",
		"Die Rechnung der Kanzlei ist bezahlt; der Rest wird nicht mehr benötigt")
	if err != nil {
		t.Fatalf("Erledigen: %v", err)
	}
	if settled.Balance() != 0 {
		t.Errorf("Bestand nach dem Erledigen %s € — erwartet null", settled.Balance())
	}
	if settled.SettledOn == "" {
		t.Error("die erledigte Rückstellung trägt kein Erledigungsdatum")
	}
	after := balances(t, env, 2027)
	if after[domain.AccountErtragAufloesungRueckstellungen] != -300_000 {
		t.Errorf("die Auflösung steht mit %s € auf 4930 — erwartet 3.000,00 € im Haben",
			after[domain.AccountErtragAufloesungRueckstellungen])
	}
}

// Der Schritt „Vortrag und Ergebnisverwendung" hat zwei Hälften: den
// Saldenvortrag und den Beschluss. Erledigt ist er erst mit beiden.
func TestAppropriationStepNeedsCarryForwardAndDecision(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	profit := &domain.JournalEntry{
		BookingDate: "2026-06-30", DocumentDate: "2026-06-30",
		ServiceDateFrom: "2026-06-30", ServiceDateTo: "2026-06-30",
		Description: "Erlös", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 400_000},
			{Side: domain.SideCredit, Account: "4400", Amount: 400_000},
		},
	}
	if _, err := env.journal.Post(ctx, profit); err != nil {
		t.Fatalf("Ergebnis buchen: %v", err)
	}

	state := func() ClosingStepView {
		t.Helper()
		steps, err := m.steps.Steps(ctx, 2026)
		if err != nil {
			t.Fatalf("Schrittliste: %v", err)
		}
		for _, step := range steps.Steps {
			if step.Key == domain.ClosingStepAppropriation {
				return step
			}
		}
		t.Fatal("der Schritt fehlt in der Liste")
		return ClosingStepView{}
	}

	if step := state(); step.State != domain.ClosingStepOpen {
		t.Errorf("ohne Vortrag steht der Schritt auf %q — erwartet offen", step.State)
	}

	if _, err := m.closing.CarryForward(ctx, 2027); err != nil {
		t.Fatalf("Saldenvortrag: %v", err)
	}
	step := state()
	if step.State != domain.ClosingStepOpen {
		t.Errorf("nach dem Vortrag allein steht der Schritt auf %q — der Beschluss fehlt noch",
			step.State)
	}
	if !strings.Contains(step.Detail, "Beschluss") {
		t.Errorf("der Schritt sagt nicht, was fehlt: %q", step.Detail)
	}

	if _, err := m.appropriation.BookAppropriation(ctx, 2026, AppropriationRequest{
		DecisionDate: "2027-05-20", Text: "Vortrag auf neue Rechnung",
	}); err != nil {
		t.Fatalf("Beschluss: %v", err)
	}
	if step := state(); step.State != domain.ClosingStepDone {
		t.Errorf("mit Vortrag und Beschluss steht der Schritt auf %q — erwartet erledigt (%s)",
			step.State, step.Detail)
	}
}

// -------------------------------------------------------------------------
// Der Storno-Weg
// -------------------------------------------------------------------------

// Jeder Baustein nennt den Storno als Weg zur Korrektur. Dieser Weg muss auch
// ans Ziel führen: eine stornierte Bildung zieht keine Auflösung nach sich, sie
// steht in keinem Bestand, und sie sperrt den Baustein nicht auf Dauer.
func TestReversedAccrualIsNeitherReleasedNorReported(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	insurance := &domain.JournalEntry{
		BookingDate: "2026-12-01", DocumentDate: "2026-12-01",
		ServiceDateFrom: "2026-12-01", ServiceDateTo: "2027-11-30",
		Description: "Betriebshaftpflicht", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "6400", Amount: 120_000},
			{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 120_000},
		},
	}
	booked, err := env.journal.Post(ctx, insurance)
	if err != nil {
		t.Fatalf("Versicherungsrechnung buchen: %v", err)
	}
	req := AccrualRequest{
		FiscalYear: 2026, Kind: domain.AccrualActive, SourceEntryID: booked.ID,
		Text: "Betriebshaftpflicht", TotalAmount: 120_000,
		StartDate: "2026-12-01", EndDate: "2027-11-30", Account: "6400",
	}
	accrual, err := m.accruals.Book(ctx, req)
	if err != nil {
		t.Fatalf("Abgrenzung buchen: %v", err)
	}
	if _, err := env.journal.Reverse(ctx, *accrual.FormationEntryID, "Betrag falsch"); err != nil {
		t.Fatalf("Bildung stornieren: %v", err)
	}

	// Kein Bestand mehr: der Bericht führt den Posten nicht weiter.
	report, err := m.accruals.Report(ctx, "2026-12-31")
	if err != nil {
		t.Fatalf("Bericht: %v", err)
	}
	if len(report.Rows) != 0 || report.TotalActive != 0 {
		t.Errorf("der Bericht führt %d Zeilen über %s € — erwartet keine",
			len(report.Rows), report.TotalActive)
	}

	// Keine Auflösung: weder in der Vorschau des Vortrags noch in seiner Buchung.
	due, err := m.accruals.PendingReleases(ctx, 2027)
	if err != nil {
		t.Fatalf("fällige Auflösungen: %v", err)
	}
	if len(due) != 0 {
		t.Errorf("%d fällige Auflösungen — erwartet keine", len(due))
	}
	if _, err := m.closing.CarryForward(ctx, 2027); err != nil {
		t.Fatalf("Saldenvortrag: %v", err)
	}
	next := balances(t, env, 2027)
	if next["6400"] != 0 {
		t.Errorf("das Jahr 2027 trägt %s € Aufwand aus einer stornierten Abgrenzung", next["6400"])
	}

	// Keine Sperre: die Abgrenzung darf neu gebildet werden.
	req.DeferredAmount = 100_000
	again, err := m.accruals.Book(ctx, req)
	if err != nil {
		t.Fatalf("nach dem Storno muss die Abgrenzung neu entstehen dürfen: %v", err)
	}
	if again.DeferredAmount != 100_000 {
		t.Errorf("neue Abgrenzung über %s € — erwartet 1.000,00 €", again.DeferredAmount)
	}
	// Und der Vorschlag zählt die stornierte nicht mehr als gebucht.
	proposal, err := m.accruals.Propose(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	if len(proposal.Items) != 1 || !proposal.Items[0].AlreadyBooked {
		t.Errorf("der Vorschlag muss genau die neue Abgrenzung als gebucht führen: %+v", proposal.Items)
	}
}

// Eine per Generalumkehr aufgehobene Rechnung ist keine Ausgabe mehr; sie
// gehört nicht mehr in die Vorschlagsliste.
func TestAccrualProposalSkipsReversedEntries(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	invoice := &domain.JournalEntry{
		BookingDate: "2026-12-01", DocumentDate: "2026-12-01",
		ServiceDateFrom: "2026-12-01", ServiceDateTo: "2027-11-30",
		Description: "Wartungsvertrag", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "6400", Amount: 120_000},
			{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 120_000},
		},
	}
	booked, err := env.journal.Post(ctx, invoice)
	if err != nil {
		t.Fatalf("Rechnung buchen: %v", err)
	}
	if _, err := env.journal.Reverse(ctx, booked.ID, "doppelt erfasst"); err != nil {
		t.Fatalf("Rechnung stornieren: %v", err)
	}
	proposal, err := m.accruals.Propose(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	if len(proposal.Items) != 0 {
		t.Errorf("eine stornierte Rechnung darf nicht vorgeschlagen werden: %+v", proposal.Items)
	}
}

// Nach dem Storno der Steuerrückstellung ist der Lauf wieder frei — sonst wäre
// der Rat der Fehlermeldung ein Rat in die Sackgasse.
func TestReversedTaxProvisionCanBeBookedAgain(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	env.taxableProfit(t)

	first, err := m.bookings.BookTaxProvision(ctx, TaxProvisionRequest{
		FiscalYear: 2026, IncomeProvision: 100_000, TradeProvision: 100_000,
	})
	if err != nil {
		t.Fatalf("Steuerrückstellung buchen: %v", err)
	}
	if len(first) != 2 {
		t.Fatalf("erwartet zwei Rückstellungen, bekommen %d", len(first))
	}
	entryID := first[0].Movements[0].JournalEntryID
	if entryID == nil {
		t.Fatal("die Bildung trägt keine Buchung")
	}
	if _, err := env.journal.Reverse(ctx, *entryID, "Verlustvortrag übersehen"); err != nil {
		t.Fatalf("Steuerrückstellung stornieren: %v", err)
	}

	// Kein Bestand mehr: der Spiegel führt sie nicht weiter.
	mirror, err := m.provisions.Mirror(ctx, 2026)
	if err != nil {
		t.Fatalf("Spiegel: %v", err)
	}
	if mirror.Total.Closing != 0 {
		t.Errorf("Endbestand des Spiegels %s € — erwartet null nach dem Storno", mirror.Total.Closing)
	}

	// Und keine Sperre: der Lauf darf mit dem richtigen Betrag wiederholt werden.
	second, err := m.bookings.BookTaxProvision(ctx, TaxProvisionRequest{
		FiscalYear: 2026, IncomeProvision: 50_000, TradeProvision: 50_000,
	})
	if err != nil {
		t.Fatalf("nach dem Storno muss die Steuerrückstellung neu entstehen dürfen: %v", err)
	}
	if len(second) != 2 {
		t.Fatalf("erwartet zwei neue Rückstellungen, bekommen %d", len(second))
	}
	list, err := m.provisions.List(ctx, 2026)
	if err != nil {
		t.Fatalf("Liste: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("die Liste führt %d Rückstellungen — erwartet nur die beiden neuen", len(list))
	}
}

// Dasselbe für den Inventurwert: nach dem Storno darf er neu erfasst werden.
func TestReversedInventoryCountCanBeBookedAgain(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	stock := &domain.JournalEntry{
		BookingDate: "2026-06-30", DocumentDate: "2026-06-30",
		ServiceDateFrom: "2026-06-30", ServiceDateTo: "2026-06-30",
		Description: "Wareneinkauf auf Bestand", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "1140", Amount: 500_000},
			{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 500_000},
		},
	}
	if _, err := env.journal.Post(ctx, stock); err != nil {
		t.Fatalf("Bestandsbuchung: %v", err)
	}
	list := env.fileIncoming(t, "inventurliste.pdf")
	req := InventoryRequest{
		FiscalYear: 2026, Account: "1140", Amount: 400_000,
		CountedOn: "2026-12-31", Method: "Stichtagsinventur", ReceiptID: list.ID,
	}
	count, err := m.bookings.BookInventory(ctx, req)
	if err != nil {
		t.Fatalf("Inventur buchen: %v", err)
	}
	if count.JournalEntryID == nil {
		t.Fatal("die Bestandsveränderung wurde nicht gebucht")
	}
	if _, err := m.bookings.BookInventory(ctx, req); err == nil {
		t.Fatal("ein zweiter Inventurwert muss abgewiesen werden, solange die Buchung steht")
	}
	if _, err := env.journal.Reverse(ctx, *count.JournalEntryID, "Zählfehler"); err != nil {
		t.Fatalf("Bestandsveränderung stornieren: %v", err)
	}

	req.Amount = 450_000
	corrected, err := m.bookings.BookInventory(ctx, req)
	if err != nil {
		t.Fatalf("nach dem Storno muss der Inventurwert neu entstehen dürfen: %v", err)
	}
	if corrected.ID != count.ID {
		t.Errorf("der Karteisatz wurde neu angelegt (%d statt %d) statt fortgeschrieben",
			corrected.ID, count.ID)
	}
	after := balances(t, env, 2026)
	if after["1140"] != 450_000 {
		t.Errorf("Bestand %s € — erwartet den berichtigten Inventurwert von 4.500,00 €", after["1140"])
	}
}

// Und für die Ergebnisverwendung.
func TestReversedAppropriationCanBeDecidedAgain(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	opening := &domain.JournalEntry{
		BookingDate: "2027-01-01", DocumentDate: "2027-01-01",
		ServiceDateFrom: "2027-01-01", ServiceDateTo: "2027-01-01",
		Description: "Saldenvortrag", Source: domain.EntrySourceOpening,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 400_000},
			{Side: domain.SideCredit, Account: domain.AccountGewinnvortrag, Amount: 400_000},
		},
	}
	if _, err := env.journal.Post(ctx, opening); err != nil {
		t.Fatalf("Vortrag buchen: %v", err)
	}
	first, err := m.appropriation.BookAppropriation(ctx, 2026, AppropriationRequest{
		DecisionDate: "2027-05-20", Distribution: 100_000, Text: "Beschluss",
	})
	if err != nil {
		t.Fatalf("Beschluss buchen: %v", err)
	}
	if first.JournalEntryID == nil {
		t.Fatal("der Beschluss wurde nicht gebucht")
	}
	if _, err := env.journal.Reverse(ctx, *first.JournalEntryID, "Beschluss aufgehoben"); err != nil {
		t.Fatalf("Beschluss stornieren: %v", err)
	}
	if _, err := m.appropriation.BookAppropriation(ctx, 2026, AppropriationRequest{
		DecisionDate: "2027-06-20", Distribution: 200_000, Text: "Neuer Beschluss",
	}); err != nil {
		t.Fatalf("nach dem Storno muss neu beschlossen werden dürfen: %v", err)
	}
	stored, err := m.appropriation.Appropriation(ctx, 2026)
	if err != nil || stored == nil {
		t.Fatalf("Beschluss lesen: %v", err)
	}
	if stored.Distribution != 200_000 {
		t.Errorf("Ausschüttung %s € — erwartet den neuen Beschluss über 2.000,00 €",
			stored.Distribution)
	}
}

// -------------------------------------------------------------------------
// Belege der Abschlussbuchungen
// -------------------------------------------------------------------------

// Keine Buchung ohne Beleg (GoBD Rz. 61). Das gilt auch für die drei Buchungen,
// die keine Rechnung von außen haben: die Bestandsveränderung, die Auflösung der
// Abgrenzung und die Ergebnisverwendung.
func TestClosingBookingsCarryTheirVoucher(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	// 1. Inventurwert: Eigenbeleg an der Buchung, Inventurliste versiegelt.
	stock := &domain.JournalEntry{
		BookingDate: "2026-06-30", DocumentDate: "2026-06-30",
		ServiceDateFrom: "2026-06-30", ServiceDateTo: "2026-06-30",
		Description: "Wareneinkauf auf Bestand", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "1140", Amount: 500_000},
			{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 500_000},
		},
	}
	if _, err := env.journal.Post(ctx, stock); err != nil {
		t.Fatalf("Bestandsbuchung: %v", err)
	}
	sheet := env.fileIncoming(t, "inventurliste.pdf")
	count, err := m.bookings.BookInventory(ctx, InventoryRequest{
		FiscalYear: 2026, Account: "1140", Amount: 400_000,
		CountedOn: "2026-12-31", Method: "Stichtagsinventur", ReceiptID: sheet.ID,
	})
	if err != nil {
		t.Fatalf("Inventur buchen: %v", err)
	}
	inventoryEntry := entryByID(t, env, *count.JournalEntryID)
	if inventoryEntry.ReceiptID == nil || inventoryEntry.ReceiptHash == "" {
		t.Errorf("die Bestandsveränderung trägt keinen Beleg: %+v", inventoryEntry)
	}
	sealed, err := env.receipts.Get(ctx, sheet.ID)
	if err != nil {
		t.Fatalf("Inventurliste lesen: %v", err)
	}
	if sealed.JournalEntryID == nil || *sealed.JournalEntryID != *count.JournalEntryID {
		t.Errorf("die Inventurliste ist nicht mit der Buchung versiegelt: %+v", sealed)
	}

	// 2. Auflösung der Abgrenzung im Folgejahr.
	insurance := &domain.JournalEntry{
		BookingDate: "2026-12-01", DocumentDate: "2026-12-01",
		ServiceDateFrom: "2026-12-01", ServiceDateTo: "2027-11-30",
		Description: "Betriebshaftpflicht", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "6400", Amount: 120_000},
			{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 120_000},
		},
	}
	booked, err := env.journal.Post(ctx, insurance)
	if err != nil {
		t.Fatalf("Versicherungsrechnung buchen: %v", err)
	}
	if _, err := m.accruals.Book(ctx, AccrualRequest{
		FiscalYear: 2026, Kind: domain.AccrualActive, SourceEntryID: booked.ID,
		Text: "Betriebshaftpflicht", TotalAmount: 120_000,
		StartDate: "2026-12-01", EndDate: "2027-11-30", Account: "6400",
	}); err != nil {
		t.Fatalf("Abgrenzung buchen: %v", err)
	}
	releases, err := m.accruals.ReleaseInto(ctx, 2027)
	if err != nil {
		t.Fatalf("Auflösung buchen: %v", err)
	}
	if len(releases) != 1 {
		t.Fatalf("erwartet eine Auflösungsbuchung, bekommen %d", len(releases))
	}
	if releases[0].ReceiptID == nil || releases[0].ReceiptHash == "" {
		t.Errorf("die Auflösung trägt keinen Eigenbeleg: %+v", releases[0])
	}

	// 3. Ergebnisverwendung, hier mit Beschlussdokument.
	opening := &domain.JournalEntry{
		BookingDate: "2027-01-02", DocumentDate: "2027-01-02",
		ServiceDateFrom: "2027-01-02", ServiceDateTo: "2027-01-02",
		Description: "Saldenvortrag", Source: domain.EntrySourceOpening,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 400_000},
			{Side: domain.SideCredit, Account: domain.AccountGewinnvortrag, Amount: 400_000},
		},
	}
	if _, err := env.journal.Post(ctx, opening); err != nil {
		t.Fatalf("Vortrag buchen: %v", err)
	}
	decision := env.fileIncoming(t, "gesellschafterbeschluss.pdf")
	appropriation, err := m.appropriation.BookAppropriation(ctx, 2026, AppropriationRequest{
		DecisionDate: "2027-05-20", Distribution: 100_000, Text: "Beschluss",
		ReceiptID: decision.ID,
	})
	if err != nil {
		t.Fatalf("Beschluss buchen: %v", err)
	}
	entry := entryByID(t, env, *appropriation.JournalEntryID)
	if entry.ReceiptID == nil || *entry.ReceiptID != decision.ID {
		t.Errorf("die Ergebnisverwendung trägt nicht das Beschlussdokument: %+v", entry)
	}
	sealedDecision, err := env.receipts.Get(ctx, decision.ID)
	if err != nil {
		t.Fatalf("Beschlussdokument lesen: %v", err)
	}
	if sealedDecision.JournalEntryID == nil {
		t.Error("das Beschlussdokument ist nicht mit der Buchung versiegelt")
	}
}

// Ohne Beschlussdokument tritt der Eigenbeleg an seine Stelle: eine
// Abschlussbuchung ohne jeden Beleg gibt es nicht.
func TestAppropriationWithoutADocumentGetsItsOwnVoucher(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	opening := &domain.JournalEntry{
		BookingDate: "2027-01-01", DocumentDate: "2027-01-01",
		ServiceDateFrom: "2027-01-01", ServiceDateTo: "2027-01-01",
		Description: "Saldenvortrag", Source: domain.EntrySourceOpening,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 400_000},
			{Side: domain.SideCredit, Account: domain.AccountGewinnvortrag, Amount: 400_000},
		},
	}
	if _, err := env.journal.Post(ctx, opening); err != nil {
		t.Fatalf("Vortrag buchen: %v", err)
	}
	appropriation, err := m.appropriation.BookAppropriation(ctx, 2026, AppropriationRequest{
		DecisionDate: "2027-05-20", Distribution: 100_000, Text: "Beschluss",
	})
	if err != nil {
		t.Fatalf("Beschluss buchen: %v", err)
	}
	entry := entryByID(t, env, *appropriation.JournalEntryID)
	if entry.ReceiptID == nil || entry.ReceiptHash == "" {
		t.Errorf("die Ergebnisverwendung trägt keinen Beleg: %+v", entry)
	}
	if entry.DocumentNumber != "EV 2026" {
		t.Errorf("Belegnummer %q — erwartet die Kennung des Bausteins EV 2026", entry.DocumentNumber)
	}
}

// -------------------------------------------------------------------------
// Monatliche Auflösung
// -------------------------------------------------------------------------

// Wer unterjährig auswertet, stellt den Auflösungstakt auf „monatlich": sonst
// trägt der Januar den ganzen Vorjahresaufwand.
func TestMonthlyReleaseSpreadsTheAccrualOverTheYear(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	env.setSetting(t, domain.SettingAccrualRelease, string(domain.AccrualReleaseMonthly))

	insurance := &domain.JournalEntry{
		BookingDate: "2026-12-01", DocumentDate: "2026-12-01",
		ServiceDateFrom: "2026-12-01", ServiceDateTo: "2027-11-30",
		Description: "Betriebshaftpflicht", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "6400", Amount: 120_000},
			{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 120_000},
		},
	}
	booked, err := env.journal.Post(ctx, insurance)
	if err != nil {
		t.Fatalf("Versicherungsrechnung buchen: %v", err)
	}
	accrual, err := m.accruals.Book(ctx, AccrualRequest{
		FiscalYear: 2026, Kind: domain.AccrualActive, SourceEntryID: booked.ID,
		Text: "Betriebshaftpflicht", TotalAmount: 120_000,
		StartDate: "2026-12-01", EndDate: "2027-11-30", Account: "6400",
	})
	if err != nil {
		t.Fatalf("Abgrenzung buchen: %v", err)
	}
	if len(accrual.Releases) != 11 {
		t.Fatalf("erwartet elf monatliche Auflösungen, bekommen %d", len(accrual.Releases))
	}
	if accrual.Releases[0].Date != "2027-01-01" {
		t.Errorf("erste Auflösung am %s — erwartet den 1. Januar", accrual.Releases[0].Date)
	}
	if accrual.Releases[10].Date != "2027-11-01" {
		t.Errorf("letzte Auflösung am %s — erwartet den 1. November", accrual.Releases[10].Date)
	}
	var planned domain.Cents
	for _, release := range accrual.Releases {
		planned += release.Amount
	}
	if planned != accrual.DeferredAmount {
		t.Errorf("der Plan über %s € deckt nicht die abgegrenzten %s €",
			planned, accrual.DeferredAmount)
	}

	// Der Saldenvortrag bucht alle Auflösungen des Zieljahres, jede an ihrem Tag.
	entries, err := m.accruals.ReleaseInto(ctx, 2027)
	if err != nil {
		t.Fatalf("Auflösung buchen: %v", err)
	}
	if len(entries) != 11 {
		t.Fatalf("erwartet elf Auflösungsbuchungen, bekommen %d", len(entries))
	}
	next := balances(t, env, 2027)
	if next["6400"] != 110_000 {
		t.Errorf("Aufwand 2027 auf 6400 = %s € — erwartet 1.100,00 €", next["6400"])
	}
	if next[domain.AccountAktiveRAP] != -110_000 {
		t.Errorf("die Abgrenzung wurde nicht vollständig aufgelöst: %s €",
			next[domain.AccountAktiveRAP])
	}
}

// Das Disagio läuft über Zinsaufwand. Ohne Konto schlägt Buchfink 7320 vor, und
// ein Konto außerhalb der 7300er wird zurückgewiesen.
func TestDisagioDefaultsToTheInterestAccount(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	preview, err := m.accruals.Preview(ctx, AccrualRequest{
		FiscalYear: 2026, Kind: domain.AccrualDisagio, Text: "Damnum Darlehen Nr. 4711",
		TotalAmount: 300_000, StartDate: "2026-07-01", EndDate: "2029-06-30",
	})
	if err != nil {
		t.Fatalf("Vorschau ohne Konto: %v", err)
	}
	if preview.Accrual.Account != domain.AccountZinsaufwandLangfristig {
		t.Errorf("vorgeschlagenes Konto %q — erwartet %s",
			preview.Accrual.Account, domain.AccountZinsaufwandLangfristig)
	}

	if _, err := m.accruals.Preview(ctx, AccrualRequest{
		FiscalYear: 2026, Kind: domain.AccrualDisagio, Text: "Damnum Darlehen Nr. 4711",
		TotalAmount: 300_000, StartDate: "2026-07-01", EndDate: "2029-06-30",
		Account: "6400",
	}); err == nil {
		t.Fatal("ein Aufwandskonto außerhalb der 7300er darf das Disagio nicht tragen")
	}
}

// Eine Rückstellung, die im Jahr verbraucht wurde, gehört in die Liste dieses
// Jahres — auch wenn sie am Jahresende nichts mehr trägt. Der Spiegel zeigt
// ihren Verbrauch, und eine Liste ohne sie widerspräche ihm.
func TestProvisionListCarriesTheYearOfItsMovements(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	provision, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionClosingCosts, Text: "Jahresabschluss 2026",
		Amount: 200_000, ExpectedOn: "2027-06-30", Reason: "Angebot des Steuerberaters",
	})
	if err != nil {
		t.Fatalf("Rückstellung bilden: %v", err)
	}
	if _, err := m.provisions.BookRelease(ctx, ProvisionChangeRequest{
		ProvisionID: provision.ID, Amount: 200_000, Date: "2027-06-30",
		Reason: "Der Steuerberater hat nicht abgerechnet; der Grund ist entfallen",
	}); err != nil {
		t.Fatalf("Rückstellung auflösen: %v", err)
	}

	list, err := m.provisions.List(ctx, 2027)
	if err != nil {
		t.Fatalf("Liste 2027: %v", err)
	}
	if len(list) != 1 || list[0].ID != provision.ID {
		t.Errorf("die Liste 2027 führt %d Rückstellungen — erwartet die im Jahr aufgelöste", len(list))
	}
}

// -------------------------------------------------------------------------
// Nachgereichte Prüfungen der dritten Runde
// -------------------------------------------------------------------------

// vatTurnover legt Vorsteuer und Umsatzsteuer an, die sich verrechnen lassen.
func (e *testEnv) vatTurnover(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	entries := []*domain.JournalEntry{
		{
			BookingDate: "2026-03-10", DocumentDate: "2026-03-10",
			ServiceDateFrom: "2026-03-10", ServiceDateTo: "2026-03-10",
			Description: "Wareneinkauf", Source: domain.EntrySourceManual,
			Lines: []domain.JournalLine{
				{Side: domain.SideDebit, Account: "6300", Amount: 100_000},
				{Side: domain.SideDebit, Account: domain.AccountVorsteuer19, Amount: 19_000,
					TaxKey: "VST19", TaxBase: 100_000},
				{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 119_000},
			},
		},
		{
			BookingDate: "2026-04-10", DocumentDate: "2026-04-10",
			ServiceDateFrom: "2026-04-10", ServiceDateTo: "2026-04-10",
			Description: "Erlös", Source: domain.EntrySourceManual,
			Lines: []domain.JournalLine{
				{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 238_000},
				{Side: domain.SideCredit, Account: "4400", Amount: 200_000},
				{Side: domain.SideCredit, Account: domain.AccountUmsatzsteuer19, Amount: 38_000,
					TaxKey: "UST19", TaxBase: 200_000},
			},
		},
	}
	for _, entry := range entries {
		if _, err := e.journal.Post(ctx, entry); err != nil {
			t.Fatalf("Buchung %q: %v", entry.Description, err)
		}
	}
}

// stepState liest den Zustand eines Bausteins aus der Schrittliste.
func (m *closingModules) stepState(
	t *testing.T, year int, key domain.ClosingStepKey,
) ClosingStepView {
	t.Helper()
	steps, err := m.steps.Steps(context.Background(), year)
	if err != nil {
		t.Fatalf("Schrittliste: %v", err)
	}
	for _, step := range steps.Steps {
		if step.Key == key {
			return step
		}
	}
	t.Fatalf("der Baustein %q steht nicht in der Schrittliste", key)
	return ClosingStepView{}
}

// Der Storno-Weg endet auch bei der Umsatzsteuer-Verrechnung nicht in der
// Sackgasse. Der Storno trägt Quelle und Belegnummer der Ursprungsbuchung;
// wird nur er ausgelassen, bliebe der Schritt „erledigt", obwohl die
// Steuerkonten wieder ihre Salden tragen.
func TestReversedVatSettlementReopensItsStep(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	env.vatTurnover(t)

	settlement, err := m.bookings.BookVatSettlement(ctx, 2026)
	if err != nil {
		t.Fatalf("Verrechnung buchen: %v", err)
	}
	if step := m.stepState(t, 2026, domain.ClosingStepVatSettlement); step.State != domain.ClosingStepDone {
		t.Fatalf("nach der Verrechnung steht der Schritt auf %q — erwartet erledigt", step.State)
	}

	if _, err := env.journal.Reverse(ctx, settlement.ID, "Voranmeldung war berichtigt"); err != nil {
		t.Fatalf("Verrechnung stornieren: %v", err)
	}

	step := m.stepState(t, 2026, domain.ClosingStepVatSettlement)
	if step.State != domain.ClosingStepOpen {
		t.Errorf("nach dem Storno steht der Schritt auf %q (%s) — erwartet offen",
			step.State, step.Detail)
	}
	if !strings.Contains(step.Detail, "storniert") {
		t.Errorf("der Schritt nennt den Storno nicht: %q", step.Detail)
	}

	// Auch ein von Hand gesetzter Haken verdeckt den Storno nicht: die
	// Ableitung hat einen Rest gefunden und gewinnt.
	if _, err := m.steps.SetStep(
		ctx, 2026, domain.ClosingStepVatSettlement, domain.ClosingStepDone, ""); err != nil {
		t.Fatalf("Schritt abhaken: %v", err)
	}
	if hidden := m.stepState(t, 2026, domain.ClosingStepVatSettlement); hidden.State != domain.ClosingStepOpen {
		t.Errorf("der abgehakte Schritt steht auf %q — der Storno darf nicht verdeckt werden",
			hidden.State)
	}

	// Die Steuerkonten tragen wieder ihre Salden, und die Verrechnung darf
	// erneut gebucht werden.
	after := balances(t, env, 2026)
	if after[domain.AccountUmsatzsteuer19] != -38_000 || after[domain.AccountVorsteuer19] != 19_000 {
		t.Errorf("nach dem Storno stehen Umsatzsteuer %s € und Vorsteuer %s € — erwartet die "+
			"ursprünglichen Salden", after[domain.AccountUmsatzsteuer19], after[domain.AccountVorsteuer19])
	}
	second, err := m.bookings.BookVatSettlement(ctx, 2026)
	if err != nil {
		t.Fatalf("nach dem Storno muss die Verrechnung erneut möglich sein: %v", err)
	}
	if second.ID == settlement.ID {
		t.Error("die zweite Verrechnung ist dieselbe Buchung wie die erste")
	}
	if step := m.stepState(t, 2026, domain.ClosingStepVatSettlement); step.State != domain.ClosingStepDone {
		t.Errorf("nach der zweiten Verrechnung steht der Schritt auf %q — erwartet erledigt", step.State)
	}
}

// Ein von Hand gesetzter Haken bleibt stehen, solange die Ableitung nichts
// gefunden hat: ein Jahr ohne Rückstellungen sähe sonst für immer unerledigt
// aus, und der Haken ist die einzige Auskunft, die es gibt.
func TestManualClosingStepDoneStandsWhereNothingWasFound(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	if before := m.stepState(t, 2026, domain.ClosingStepProvisions); before.State != domain.ClosingStepOpen {
		t.Fatalf("ohne Rückstellungen steht der Schritt auf %q — erwartet offen", before.State)
	}
	if _, err := m.steps.SetStep(
		ctx, 2026, domain.ClosingStepProvisions, domain.ClosingStepDone, ""); err != nil {
		t.Fatalf("Schritt abhaken: %v", err)
	}

	step := m.stepState(t, 2026, domain.ClosingStepProvisions)
	if step.State != domain.ClosingStepDone {
		t.Errorf("der abgehakte Schritt steht auf %q — erwartet erledigt", step.State)
	}
	if !strings.Contains(step.Detail, "von Hand abgehakt") {
		t.Errorf("der Schritt sagt nicht, woher sein Zustand kommt: %q", step.Detail)
	}
}

// Findet die Ableitung dagegen einen Rest, gewinnt sie über den Haken: sonst
// verdeckte ein früher gesetzter Haken die AfA, die erst später aufgefallen ist.
func TestManualClosingStepDoneDoesNotHideAFoundRest(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	if _, err := m.steps.SetStep(
		ctx, 2026, domain.ClosingStepDepreciation, domain.ClosingStepDone, ""); err != nil {
		t.Fatalf("Schritt abhaken: %v", err)
	}
	m.steps.SetDepreciationSource(depreciationStub{due: []DepreciationDue{
		{AssetID: 1, InventoryNumber: "AV-2026-0001", Name: "Maschine", Due: 120_000},
	}})

	step := m.stepState(t, 2026, domain.ClosingStepDepreciation)
	if step.State != domain.ClosingStepOpen {
		t.Errorf("der Schritt steht auf %q — erwartet offen, weil AfA offen ist", step.State)
	}
	if step.Detail == "" {
		t.Error("der Schritt nennt den gefundenen Rest nicht")
	}

	// Ist die AfA gebucht, trägt der Haken wieder.
	m.steps.SetDepreciationSource(depreciationStub{})
	if again := m.stepState(t, 2026, domain.ClosingStepDepreciation); again.State != domain.ClosingStepDone {
		t.Errorf("nach der AfA steht der Schritt auf %q — erwartet erledigt", again.State)
	}
}

// § 253 Abs. 2 Satz 2 HGB lässt für Altersversorgungsverpflichtungen den
// Zehnjahresdurchschnitt zu; die Zinstabelle führt beide Reihen, und welche
// gilt, folgt aus der Art. Bewertet wird die Pensionszusage nicht: der
// steuerliche Wert nach § 6a EStG entsteht nicht aus den 5,5 % des
// § 6 Abs. 1 Nr. 3a EStG.
func TestPensionProvisionUsesTheTenYearAverage(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	m.rate(t, "2026-12", 3, 15_000)

	req := ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionPension, Text: "Pensionszusage Geschäftsführer",
		Amount: 1_000_000, ExpectedOn: "2029-12-30", Reason: "Zusage vom 1. Juli 2026",
	}
	preview, err := m.provisions.Preview(ctx, req)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.Discounted {
		t.Error("mit dem Siebenjahresdurchschnitt darf eine Pensionsrückstellung nicht abgezinst werden")
	}
	if !containsSubstring(preview.Findings, "10-Jahres-Durchschnitt") {
		t.Errorf("der fehlende Zehnjahressatz wird nicht benannt: %v", preview.Findings)
	}
	if !containsSubstring(preview.Findings, "§ 6a EStG") {
		t.Errorf("die fehlende Bewertung wird nicht benannt: %v", preview.Findings)
	}
	if preview.TaxAmount != preview.Amount {
		t.Errorf("steuerlicher Wert %s € gegen handelsrechtlich %s € — für Pensionen rechnet "+
			"Buchfink keine Differenz", preview.TaxAmount, preview.Amount)
	}

	// Mit dem Zehnjahressatz wird abgezinst: 10.000 € in drei Jahren zu 1,5 %
	// sind 9.563,17 €.
	if err := m.provisions.SaveDiscountRates(ctx, []domain.DiscountRate{
		{Month: "2026-12", Years: 3, RateMicros: 15_000, Average: 10},
	}); err != nil {
		t.Fatalf("Zehnjahressatz pflegen: %v", err)
	}
	preview, err = m.provisions.Preview(ctx, req)
	if err != nil {
		t.Fatalf("zweite Vorschau: %v", err)
	}
	if !preview.Discounted || preview.Amount != 956_317 {
		t.Errorf("Barwert %s € (abgezinst %v) — erwartet 9.563,17 € mit dem Zehnjahresdurchschnitt",
			preview.Amount, preview.Discounted)
	}

	// Und die Überleitung weist für die Pension keine steuerliche Differenz aus.
	if _, err := m.provisions.BookFormation(ctx, req); err != nil {
		t.Fatalf("Pensionsrückstellung bilden: %v", err)
	}
	reconciliation, err := m.register.Reconcile(ctx, 2026)
	if err != nil {
		t.Fatalf("Überleitung: %v", err)
	}
	for _, row := range reconciliation.Rows {
		if strings.Contains(row.Position, "Rückstellungen") && row.Difference != 0 {
			t.Errorf("die Überleitung zeigt für die Pensionsrückstellung eine Differenz von %s €",
				row.Difference)
		}
	}
}

// Die Überleitung stellt Wertansätze gegenüber und keine Differenzen: unter den
// Spaltenköpfen „Handelsbilanz" und „Steuerbilanz" steht in jeder Zeile
// dasselbe — beim Anlagevermögen die Buchwerte der Wirtschaftsgüter mit § 7g.
func TestReconciliationShowsBookValuesForFixedAssets(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	assets := env.assets(t)
	ctx := context.Background()

	if _, err := assets.Save(ctx, &domain.FixedAsset{
		Name: "Fräsmaschine", Class: domain.AssetClassTangible,
		AcquisitionDate: "2026-01-15", AcquisitionCost: 10_000_000, UsefulLifeMonths: 120,
		Method: domain.DepreciationLinear, Account: "0440", DepreciationAccount: "6220",
		SpecialPermille: 400, SpecialYears: 5,
		SpecialReason: "Gewinn 2025: 140.000 €; ausschließlich betriebliche Nutzung",
	}); err != nil {
		t.Fatalf("Anlagegut: %v", err)
	}
	if _, err := assets.BookDepreciation(ctx, BookDepreciationRequest{FiscalYear: 2026}); err != nil {
		t.Fatalf("Abschreibungslauf: %v", err)
	}

	register, err := m.register.Register(ctx, 2026)
	if err != nil {
		t.Fatalf("Verzeichnis: %v", err)
	}
	if len(register.Rows) != 1 {
		t.Fatalf("das Verzeichnis führt %d Zeilen — erwartet eine", len(register.Rows))
	}
	row := register.Rows[0]
	// 100.000 € Anschaffungskosten, 10.000 € handelsrechtliche AfA: Buchwert
	// 90.000 €. Steuerlich mindern 8.000 € Mehr-AfA den Wertansatz auf 82.000 €.
	if row.BookValue != 9_000_000 {
		t.Errorf("handelsrechtlicher Buchwert %s € — erwartet 90.000,00 €", row.BookValue)
	}
	if row.TaxBookValue != 8_200_000 {
		t.Errorf("steuerlicher Buchwert %s € — erwartet 82.000,00 €", row.TaxBookValue)
	}
	if row.TaxBookValue != row.BookValue-row.TotalDifference {
		t.Errorf("Buchwerte %s €/%s € und Differenz %s € gehen nicht auf",
			row.BookValue, row.TaxBookValue, row.TotalDifference)
	}

	reconciliation, err := m.register.Reconcile(ctx, 2026)
	if err != nil {
		t.Fatalf("Überleitung: %v", err)
	}
	found := false
	for _, line := range reconciliation.Rows {
		if line.Position != "Anlagevermögen" {
			continue
		}
		found = true
		if line.Commercial != register.TotalBookValue || line.Tax != register.TotalTaxBookValue {
			t.Errorf("die Zeile trägt %s € / %s € — erwartet die Buchwerte %s € / %s €",
				line.Commercial, line.Tax, register.TotalBookValue, register.TotalTaxBookValue)
		}
		if line.Difference != line.Tax-line.Commercial {
			t.Errorf("die Differenz %s € passt nicht zu den Spalten", line.Difference)
		}
		if line.Commercial <= 0 {
			t.Error("die handelsrechtliche Spalte steht auf null — sie trägt einen Wertansatz")
		}
	}
	if !found {
		t.Error("die Überleitung nennt das Anlagevermögen nicht")
	}
}

// Ein Auflösungsbetrag über dem Bestand wird abgelehnt, nicht stillschweigend
// gekappt: die Auflösung ist begründungspflichtig, und ein Tippfehler darf die
// Rückstellung nicht verschwinden lassen.
func TestProvisionReleaseAboveBalanceIsRejected(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()

	provision, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionClosingCosts, Text: "Jahresabschluss 2026",
		Amount: 300_000, ExpectedOn: "2027-06-30", Reason: "Angebot der Kanzlei",
	})
	if err != nil {
		t.Fatalf("Rückstellung bilden: %v", err)
	}
	if _, err := m.provisions.BookRelease(ctx, ProvisionChangeRequest{
		ProvisionID: provision.ID, Amount: 500_000, Date: "2027-03-31", Reason: "Grund entfallen",
	}); err == nil {
		t.Fatal("eine Auflösung über den Bestand hinaus darf nicht durchgehen")
	}
	reloaded, err := m.provisions.load(ctx, provision.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Balance() != 300_000 {
		t.Errorf("Bestand nach abgelehnter Auflösung %s € — erwartet unverändert 3.000,00 €", reloaded.Balance())
	}
}

// Die automatische Aufzinsung gilt dem Rest, der nach einem Teilverbrauch
// noch zu Buche steht, nicht dem ursprünglichen Erfüllungsbetrag. 10.000 € auf
// drei Jahre zu 1,5 % sind 9.563,17 €; nach Auflösung der Hälfte (4.781,59 €)
// gilt zum nächsten Stichtag der Barwert von 5.000 € auf zwei Jahre
// (4.853,73 €), also eine Aufzinsung von 72,14 € — nicht von über 5.000 €.
func TestProvisionUnwindingAfterPartialReleaseFollowsTheRemainder(t *testing.T) {
	env := newTestEnv(t)
	m := env.closingModules(t)
	ctx := context.Background()
	m.rate(t, "2026-12", 3, 15_000)
	m.rate(t, "2027-12", 2, 15_000)

	provision, err := m.provisions.BookFormation(ctx, ProvisionRequest{
		FiscalYear: 2026, Kind: domain.ProvisionUncertainLiability,
		Text: "Rückbauverpflichtung", Amount: 1_000_000, ExpectedOn: "2029-12-30",
		Reason: "Mietvertrag § 12",
	})
	if err != nil {
		t.Fatalf("Rückstellung bilden: %v", err)
	}
	half := provision.Balance() / 2
	if _, err := m.provisions.BookRelease(ctx, ProvisionChangeRequest{
		ProvisionID: provision.ID, Amount: half, Date: "2027-06-30", Reason: "Hälfte der Einbauten übernommen",
	}); err != nil {
		t.Fatalf("Teilauflösung: %v", err)
	}
	unwound, err := m.provisions.BookUnwinding(ctx, ProvisionChangeRequest{
		ProvisionID: provision.ID, Date: "2027-12-31", Reason: "Stichtag",
	})
	if err != nil {
		t.Fatalf("Aufzinsung: %v", err)
	}
	got := unwound.Balance() - (provision.Balance() - half)
	if got < 7_000 || got > 7_400 {
		t.Errorf("Aufzinsung %s € — erwartet rund 72 € (Barwert des Restbetrags), nicht das Auffüllen des aufgelösten Teils", got)
	}
}
