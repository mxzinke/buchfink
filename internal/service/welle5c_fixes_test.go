package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/vatid"
)

// Die Nachbesserungen der Welle 5c: die Fremdwährung am Belegweg, die Aufnahme
// ins Verzeichnis nach § 15a UStG bei der Aktivierung, die beiden Prüflaufregeln
// zur ig. Lieferung, die gespeicherte Beförderungsart, die Auflösung der
// Stichtagsbewertung beim Saldenvortrag und der Gebäudestichtag.

// -------------------------------------------------------------------------
// Fremdwährung am Belegweg (BEW-10)
// -------------------------------------------------------------------------

// currencyPosting verdrahtet Kursdienst und Belegweg gegen einen gefakten
// EZB-Endpunkt.
func (e *testEnv) currencyPosting(t *testing.T, endpoint string) *CurrencyService {
	t.Helper()
	svc := e.currency(t, endpoint)
	e.posting.SetCurrencyConverter(svc)
	return svc
}

// Ein Beleg über 1.085,00 USD vom 10.03.2026: der Aufwand folgt dem
// EZB-Tageskurs (1,0850 → 1.000,00 €), die Bemessungsgrundlage der Umsatzsteuer
// dem Durchschnittskurs des Monats (1,1000 → 986,36 €), und die Differenz ist
// Kursaufwand auf 6880. Kurs, Quelle und Kurstag stehen am Buchungskopf.
func TestForeignCurrencyReceiptBooksBothRates(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, _ := ecbServer(t, "1.0850", "2026-03-10")
	svc := env.currencyPosting(t, server.URL)
	if _, err := svc.SaveVatRate(ctx, domain.VatExchangeRate{
		Month: "2026-03", Currency: "USD", RateMicros: 1_100_000,
	}); err != nil {
		t.Fatalf("Durchschnittskurs: %v", err)
	}
	vendor := env.vendor(t, "Supplier Inc.", "US", "")

	req := env.receipt(t, vendor.ID, "fremdleistungen", 108_500,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Currency = "USD"
	req.ForeignAmount = 108_500

	preview, err := env.posting.PreviewIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.Conversion == nil || preview.Conversion.Rate.RateMicros != 1_085_000 {
		t.Fatalf("die Vorschau muss den verwendeten Kurs zeigen: %+v", preview.Conversion)
	}

	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}

	if entry.Currency != "USD" {
		t.Errorf("Währung %q — erwartet USD", entry.Currency)
	}
	if entry.ExchangeRateMicros != 1_085_000 {
		t.Errorf("Kurs %d Millionstel — erwartet 1.085.000 und niemals 1.000.000 (Kurs 1,0)",
			entry.ExchangeRateMicros)
	}
	if !strings.Contains(entry.ExchangeRateSource, "EZB") || entry.ExchangeRateDate != "2026-03-10" {
		t.Errorf("Quelle %q zum %q — beide gehören an die Buchung",
			entry.ExchangeRateSource, entry.ExchangeRateDate)
	}

	// Der Aufwand: 1.085,00 USD zu 1,0850 sind 1.000,00 €.
	if got := debitOn(entry, "5906"); got != 100_000 {
		t.Errorf("Aufwand %s € — erwartet 1.000,00 € zum Tageskurs", got)
	}
	var expense *domain.JournalLine
	for i := range entry.Lines {
		if entry.Lines[i].Account == "5906" {
			expense = &entry.Lines[i]
		}
	}
	if expense == nil || expense.ForeignAmount != 108_500 {
		t.Errorf("Fremdbetrag der Aufwandszeile %+v — erwartet 1.085,00 USD", expense)
	}

	// Die Steuerzeile: Bemessungsgrundlage aus dem Umsatzsteuerkurs (986,36 €),
	// Vorsteuer 19 % davon (187,41 €).
	var tax *domain.JournalLine
	for i := range entry.Lines {
		if entry.Lines[i].TaxKey == "VST19" {
			tax = &entry.Lines[i]
		}
	}
	if tax == nil {
		t.Fatalf("keine Vorsteuerzeile: %s", accountsOf(entry))
	}
	if tax.TaxBase != 98_636 {
		t.Errorf("Bemessungsgrundlage %s € — erwartet 986,36 € zum Durchschnittskurs des Monats "+
			"(§ 16 Abs. 6 UStG)", tax.TaxBase)
	}
	if tax.Amount != 18_741 {
		t.Errorf("Vorsteuer %s € — erwartet 187,41 €", tax.Amount)
	}

	// Die Differenz zwischen Tages- und Umsatzsteuerkurs ist Kursaufwand: 19 %
	// von 1.000,00 € sind 190,00 €, gebucht werden 187,41 € — 2,59 € auf 6880.
	if got := debitOn(entry, accounting.CurrencyLossAccount); got != 259 {
		t.Errorf("Kursaufwand %s € auf %s — erwartet 2,59 €",
			got, accounting.CurrencyLossAccount)
	}

	// Und die Gegenzeile trägt, was tatsächlich zu zahlen ist: 1.290,15 USD zum
	// Tageskurs sind 1.190,00 €.
	settlement := entry.Lines[len(entry.Lines)-1]
	if settlement.Amount != 119_000 {
		t.Errorf("Gegenzeile %s € — erwartet 1.190,00 € brutto zum Tageskurs", settlement.Amount)
	}
	if settlement.ForeignAmount != 129_115 {
		t.Errorf("Fremdbetrag der Gegenzeile %s — erwartet 1.291,15 USD", settlement.ForeignAmount)
	}
}

// Ohne Kurs wird nicht gebucht. Ein Rückfallwert von 1,0 bucht einen
// Dollarbetrag als Eurobetrag — in richtiger Größenordnung und mit falschem
// Wert, und niemandem fällt es auf.
func TestForeignCurrencyReceiptRefusedWithoutARate(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	// Ein Server, der sofort geschlossen wird: es gibt keinen Kurs.
	server, _ := ecbServer(t, "1.0850", "2026-03-10")
	url := server.URL
	server.Close()
	env.currencyPosting(t, url)
	vendor := env.vendor(t, "Supplier Inc.", "US", "")

	req := env.receipt(t, vendor.ID, "fremdleistungen", 108_500,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Currency = "USD"

	if _, err := env.posting.PostIncomingReceipt(ctx, req); err == nil {
		t.Fatal("ohne Kurs darf kein Fremdwährungsbeleg gebucht werden")
	}

	// Mit einem von Hand erfassten Kurs geht es — er trägt seine Quelle.
	svc := env.currency(t, url)
	env.posting.SetCurrencyConverter(svc)
	if _, err := svc.SaveRate(ctx, domain.ExchangeRate{
		Currency: "USD", Date: "2026-03-10", RateMicros: 1_085_000,
		Source: "EZB-Referenzkurs, abgelesen am 11.03.2026",
	}); err != nil {
		t.Fatalf("Kurs von Hand: %v", err)
	}
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("mit erfasstem Kurs muss der Beleg buchbar sein: %v", err)
	}
	if entry.ExchangeRateMicros != 1_085_000 {
		t.Errorf("Kurs %d Millionstel — erwartet den erfassten", entry.ExchangeRateMicros)
	}
}

// Die Kontrollsumme des Belegs wird gegen die Positionen gehalten. Ein
// Zahlendreher ergibt sonst eine Buchung, die in sich aufgeht und mit der
// Rechnung daneben nicht übereinstimmt.
func TestForeignCurrencyControlTotalIsChecked(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, _ := ecbServer(t, "1.0850", "2026-03-10")
	env.currencyPosting(t, server.URL)
	vendor := env.vendor(t, "Supplier Inc.", "US", "")

	req := env.receipt(t, vendor.ID, "fremdleistungen", 108_500,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Currency = "USD"
	req.ForeignAmount = 180_500

	if _, err := env.posting.PostIncomingReceipt(ctx, req); err == nil {
		t.Fatal("die Endsumme des Belegs muss zu seinen Positionen passen")
	}
}

// -------------------------------------------------------------------------
// Stichtagsbewertung: Bankkonten und die Auflösung beim Saldenvortrag
// -------------------------------------------------------------------------

// Ein Dollarkonto gehört in die Stichtagsbewertung: § 256a HGB nennt „auf
// Fremdwährung lautende Vermögensgegenstände" und meint nicht nur die offenen
// Posten.
func TestCurrencyValuationCoversForeignBankAccounts(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, _ := ecbServer(t, "1.0000", "2026-12-31")
	svc := env.currency(t, server.URL)
	svc.SetClosingService(env.closing(t))
	svc.SetOpenItemSource(NewPaymentService(
		env.journal, env.journalRepo,
		repository.NewPaymentAllocationRepository(env.db),
		env.contactRepo, repository.NewBankRepository(env.db), env.fiscalYear))

	// 11.000 USD zum Buchkurs 1,10 sind 10.000 € auf dem Dollarkonto. Zum
	// Stichtagskurs 1,00 sind sie 11.000 € wert: 1.000 € Kursgewinn.
	if _, err := env.journal.Post(ctx, &domain.JournalEntry{
		BookingDate: "2026-05-02", DocumentDate: "2026-05-02",
		ServiceDateFrom: "2026-05-02", ServiceDateTo: "2026-05-02",
		Description: "Zahlungseingang auf dem Dollarkonto", Source: domain.EntrySourceManual,
		DocumentNumber: "BK-USD-1", TaxTreatment: domain.TaxTreatmentNotTaxable,
		Currency: "USD", ExchangeRateMicros: 1_100_000,
		ExchangeRateSource: "EZB-Referenzkurs", ExchangeRateDate: "2026-05-02",
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "1800", Amount: 1_000_000, ForeignAmount: 1_100_000},
			{Side: domain.SideCredit, Account: "4400", Amount: 1_000_000, ForeignAmount: 1_100_000},
		},
	}); err != nil {
		t.Fatalf("Bankbuchung: %v", err)
	}

	preview, err := svc.PreviewCurrencyValuation(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	var bank *ForeignCurrencyValuationItem
	for i := range preview.Items {
		if preview.Items[i].Kind == "bank" {
			bank = &preview.Items[i]
		}
	}
	if bank == nil {
		t.Fatalf("das Fremdwährungskonto fehlt in der Bewertung: %+v", preview.Items)
	}
	if bank.Account != "1800" || bank.Currency != "USD" || bank.ForeignAmount != 1_100_000 {
		t.Errorf("Bankzeile %+v — erwartet 11.000,00 USD auf 1800", bank)
	}
	if !bank.Gain || !bank.Recognised || bank.Amount != 100_000 {
		t.Errorf("Bankzeile: Gewinn=%v erfasst=%v %s € — ein Guthaben ist jederzeit fällig und "+
			"damit stets kurzfristig (%s)", bank.Gain, bank.Recognised, bank.Amount, bank.Reason)
	}
	if preview.TotalGain != 100_000 {
		t.Errorf("Ertrag %s € — erwartet 1.000,00 €", preview.TotalGain)
	}
}

// Die Auflösung der Bewertung wird nicht mehr sofort ins Folgejahr geschrieben,
// sondern vom Saldenvortrag gebucht — wie die Auflösung der Abgrenzung. Und der
// Vortrag ist zugleich der Nachholweg: er fragt das Journal und kein Merkzeichen.
func TestCurrencyValuationIsReversedByTheCarryForward(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, _ := ecbServer(t, "1.0000", "2026-12-31")
	closing := env.closing(t)
	svc := env.currency(t, server.URL)
	svc.SetClosingService(closing)
	svc.SetOpenItemSource(NewPaymentService(
		env.journal, env.journalRepo,
		repository.NewPaymentAllocationRepository(env.db),
		env.contactRepo, repository.NewBankRepository(env.db), env.fiscalYear))
	closing.SetCurrencyReverser(svc)

	customer := env.customer(t, "Client Inc.", "US", "")
	customer.PaymentTermsDays = 30
	if err := env.contacts.SaveContact(ctx, customer); err != nil {
		t.Fatalf("Kunde: %v", err)
	}
	env.foreignOpenItem(t, customer, "RE-USD-1", 1_000_000, 1_100_000)

	booked, err := svc.BookCurrencyValuation(ctx, 2026)
	if err != nil {
		t.Fatalf("Bewertung: %v", err)
	}
	if booked.EntryNumber == "" {
		t.Fatal("die Bewertungsbuchung fehlt")
	}
	// Die Auflösung steht noch nicht: sie kommt mit dem Vortrag.
	if booked.ReversalEntryNumber != "" {
		t.Errorf("Auflösung %q — sie gehört an den Saldenvortrag und nicht in denselben Zug",
			booked.ReversalEntryNumber)
	}
	entries2027, err := env.journalRepo.FindAll(ctx, 2027)
	if err != nil {
		t.Fatalf("Journal 2027: %v", err)
	}
	if len(entries2027) != 0 {
		t.Fatalf("im Folgejahr steht schon etwas: %d Buchungen", len(entries2027))
	}

	// Der Vortrag bucht sie.
	reversals, err := svc.ReverseInto(ctx, 2027)
	if err != nil {
		t.Fatalf("Auflösung: %v", err)
	}
	if len(reversals) != 1 {
		t.Fatalf("%d Auflösungsbuchungen — erwartet eine", len(reversals))
	}
	reversal := reversals[0]
	if reversal.BookingDate != "2027-01-01" {
		t.Errorf("Auflösung am %s — erwartet den ersten Tag des Folgejahres", reversal.BookingDate)
	}
	if got := debitOn(&reversal, accounting.CurrencyGainAccount); got != 100_000 {
		t.Errorf("die Auflösung muss den Ertrag zurücknehmen: %s € auf %s",
			got, accounting.CurrencyGainAccount)
	}

	// Ein zweiter Lauf bucht sie nicht noch einmal — und wäre der Nachholweg,
	// wenn der erste ausgefallen wäre.
	again, err := svc.ReverseInto(ctx, 2027)
	if err != nil {
		t.Fatalf("zweiter Lauf: %v", err)
	}
	if len(again) != 0 {
		t.Errorf("%d weitere Auflösungen — eine Bewertung wird einmal aufgelöst", len(again))
	}
}

// -------------------------------------------------------------------------
// Aufnahme ins Verzeichnis nach § 15a UStG bei der Aktivierung
// -------------------------------------------------------------------------

// Der Pkw aus dem Lehrbuch, diesmal über die Anlagenbuchhaltung: 7.600 €
// Vorsteuer sind mehr als die 1.000 € des § 44 Abs. 1 UStDV, also entsteht der
// Eintrag mit der Aktivierung und nicht erst, wenn jemand daran denkt.
func TestActivationRegistersTheInputTaxCorrection(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	assets := env.assets(t)
	register := env.inputTax(t)
	assets.SetInputTaxRegister(register)

	asset, err := assets.Save(ctx, &domain.FixedAsset{
		Name: "Firmenwagen", Class: domain.AssetClassTangible,
		Account: "0520", DepreciationAccount: "6222",
		AcquisitionDate: "2026-01-15", AcquisitionCost: 4_000_000,
		UsefulLifeMonths: 72, Method: domain.DepreciationLinear,
		InputTaxAmount: 760_000, InputTaxPermille: 1000,
	})
	if err != nil {
		t.Fatalf("Aktivierung: %v", err)
	}

	view, err := register.Year(ctx, 2026)
	if err != nil {
		t.Fatalf("Verzeichnis: %v", err)
	}
	if len(view.Rows) != 1 {
		t.Fatalf("%d Einträge im Verzeichnis — die Aktivierung muss einen anlegen", len(view.Rows))
	}
	row := view.Rows[0].Correction
	if row.InputTaxAmount != 760_000 || row.OriginalPermille != 1000 {
		t.Errorf("Eintrag %+v — erwartet 7.600,00 € Vorsteuer bei voller Verwendung", row)
	}
	if row.CorrectionPeriodYears != 5 {
		t.Errorf("Zeitraum %d Jahre — ein Pkw ist beweglich", row.CorrectionPeriodYears)
	}
	if row.AssetID == nil || *row.AssetID != asset.ID {
		t.Errorf("der Eintrag muss auf das Anlagegut zeigen: %+v", row.AssetID)
	}
	if !strings.Contains(row.Label, asset.InventoryNumber) {
		t.Errorf("Bezeichnung %q — sie muss die Inventarnummer nennen", row.Label)
	}

	// Ein Kopierer mit 900 € Vorsteuer bleibt draußen: unterhalb der
	// Bagatellgrenze wird nie berichtigt.
	if _, err := assets.Save(ctx, &domain.FixedAsset{
		Name: "Kopierer", Class: domain.AssetClassTangible,
		Account: "0650", DepreciationAccount: "6220",
		AcquisitionDate: "2026-03-01", AcquisitionCost: 473_700,
		UsefulLifeMonths: 156, Method: domain.DepreciationLinear,
		InputTaxAmount: 90_000,
	}); err != nil {
		t.Fatalf("zweite Aktivierung: %v", err)
	}
	after, err := register.Year(ctx, 2026)
	if err != nil {
		t.Fatalf("Verzeichnis: %v", err)
	}
	if len(after.Rows) != 1 {
		t.Errorf("%d Einträge — ein Wirtschaftsgut unter der Bagatellgrenze des § 44 Abs. 1 UStDV "+
			"wird nicht berichtigt und braucht keinen Eintrag", len(after.Rows))
	}
}

// Die Berichtigung eines abweichenden Wirtschaftsjahres wird zu seinem letzten
// Tag gebucht und nicht zum ersten des folgenden — sonst fiele sie in die
// Voranmeldung des nächsten Jahres.
func TestInputTaxCorrectionBooksOnTheLastDayOfTheFiscalYear(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	settings := repository.NewSettingsRepository(env.db)
	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Einstellungen: %v", err)
	}
	cfg.FiscalYearStartMonth = 7
	if err := settings.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatalf("Einstellungen setzen: %v", err)
	}

	svc := env.inputTax(t)
	date, err := svc.bookingDate(ctx, 2026)
	if err != nil {
		t.Fatalf("Buchungsdatum: %v", err)
	}
	if date != "2027-06-30" {
		t.Errorf("Buchungsdatum %q — ein Wirtschaftsjahr, das im Juli beginnt, endet am 30.06.", date)
	}
}

// -------------------------------------------------------------------------
// Prüflaufregeln zur steuerfreien innergemeinschaftlichen Lieferung
// -------------------------------------------------------------------------

// icSupply legt eine ausgestellte steuerfreie ig. Lieferung an.
func (e *testEnv) icSupply(t *testing.T, customer *domain.Contact, number string) *domain.Invoice {
	t.Helper()
	invoice := &domain.Invoice{
		FiscalYear: 2026, InvoiceNumber: number, Date: "2026-04-02",
		ServiceDateFrom: "2026-04-01", ServiceDateTo: "2026-04-01",
		ContactID: customer.ID, ContactName: customer.Name,
		TaxTreatment: domain.TaxTreatmentIntraCommunitySupply,
		Status:       domain.InvoiceStatusIssued, Currency: "EUR",
		NetAmount: 500_000, GrossAmount: 500_000,
	}
	if err := repository.NewInvoiceRepository(e.db).Save(context.Background(), invoice); err != nil {
		t.Fatalf("Rechnung %s: %v", number, err)
	}
	return invoice
}

// Der Prüflauf meldet beides: die Lieferung ohne vollständigen Belegnachweis
// und die ohne bestätigte USt-IdNr. Die zweite Regel ist der zugesagte
// Folgebefund zur Übersteuerung bei Offline-BZSt.
func TestCheckRunReportsIntraCommunitySupplies(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	invoice := env.icSupply(t, customer, "RE-2026-0001")
	invoice.VatIDOverrideReason = "Bundeszentralamt nicht erreichbar"
	if err := repository.NewInvoiceRepository(env.db).Save(ctx, invoice); err != nil {
		t.Fatalf("Rechnung: %v", err)
	}

	checks := env.checks(t)
	checks.SetSupplyEvidenceSource(env.supplyEvidence(t))
	checks.SetVatIDStatusSource(env.vatIDs(t, "https://example.invalid/evatr"))

	run := runChecks(t, checks, "2026-12-31")
	byRule := map[string]*domain.CheckFinding{}
	for i := range run.Findings {
		byRule[run.Findings[i].Rule] = &run.Findings[i]
	}

	evidence := byRule[domain.CheckRuleICSupplyEvidenceMissing]
	if evidence == nil {
		t.Fatalf("kein Befund %q: %+v", domain.CheckRuleICSupplyEvidenceMissing, run.Findings)
	}
	if evidence.Severity != domain.CheckWarning {
		t.Errorf("Schwere %q — der Nachweis darf nachgereicht werden", evidence.Severity)
	}
	if !strings.Contains(evidence.Message, "Voranmeldung") {
		t.Errorf("der Befund muss die Frist nennen: %s", evidence.Message)
	}
	if evidence.ObjectName != "RE-2026-0001" {
		t.Errorf("Bezugsobjekt %q — der Befund muss die Rechnung benennen", evidence.ObjectName)
	}

	unconfirmed := byRule[domain.CheckRuleICSupplyUnconfirmed]
	if unconfirmed == nil {
		t.Fatalf("kein Befund %q: %+v", domain.CheckRuleICSupplyUnconfirmed, run.Findings)
	}
	if !strings.Contains(unconfirmed.Message, "Bundeszentralamt nicht erreichbar") {
		t.Errorf("der Folgebefund muss den festgehaltenen Grund nennen: %s", unconfirmed.Message)
	}
	if !strings.Contains(unconfirmed.Reference, "§ 6a") {
		t.Errorf("Fundstelle %q — erwartet § 6a Abs. 1 Satz 1 Nr. 4 UStG", unconfirmed.Reference)
	}
}

// -------------------------------------------------------------------------
// Beförderungsart und Nachweisdatei
// -------------------------------------------------------------------------

// Ein Abholfall ohne Gelangensbestätigung ist nicht nachgewiesen — auch im
// Jahresbericht. Vorher bewertete er jede Lieferung als Regelfall und meldete
// den Abholfall als erfüllt.
func TestSupplyEvidenceReportUsesTheStoredTransportKind(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.supplyEvidence(t)
	svc.SetReceiptService(env.receipts)
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	invoice := env.icSupply(t, customer, "RE-2026-0002")

	// Zwei unabhängige Belege: im Regelfall genügten sie.
	path := filepath.Join(t.TempDir(), "cmr.pdf")
	if err := os.WriteFile(path, []byte(minimalPDF), 0o600); err != nil {
		t.Fatalf("Datei schreiben: %v", err)
	}
	view, err := svc.Add(ctx, SupplyEvidenceRequest{
		InvoiceID: invoice.ID, Kind: string(accounting.EvidenceCMR),
		Issuer: "Spedition Nord", Independent: true, Date: "2026-04-03",
		Transport: string(accounting.TransportByCustomer),
		FilePath:  path,
	})
	if err != nil {
		t.Fatalf("Nachweisbeleg: %v", err)
	}
	// Die Datei ist im Belegspeicher gelandet und am Nachweis vermerkt.
	if len(view.Items) != 1 || view.Items[0].ReceiptID == nil {
		t.Fatalf("der Nachweis muss auf den abgelegten Beleg zeigen: %+v", view.Items)
	}
	if _, err := env.receipts.Get(ctx, *view.Items[0].ReceiptID); err != nil {
		t.Errorf("der abgelegte Beleg ist nicht lesbar: %v", err)
	}
	if _, err := svc.Add(ctx, SupplyEvidenceRequest{
		InvoiceID: invoice.ID, Kind: string(accounting.EvidenceInsurance),
		Issuer: "Transportversicherung AG", Independent: true, Date: "2026-04-03",
	}); err != nil {
		t.Fatalf("zweiter Nachweisbeleg: %v", err)
	}

	report, err := svc.Report(ctx, 2026)
	if err != nil {
		t.Fatalf("Bericht: %v", err)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("%d Zeilen — erwartet eine", len(report.Rows))
	}
	if report.Rows[0].Transport != accounting.TransportByCustomer {
		t.Errorf("Beförderungsart %q — sie steht an der Rechnung und muss im Bericht ankommen",
			report.Rows[0].Transport)
	}
	if report.Rows[0].Status.Fulfilled || report.Incomplete != 1 {
		t.Errorf("im Abholfall fehlt die Gelangensbestätigung: %s", report.Rows[0].Status.Reason)
	}

	// Mit der Gelangensbestätigung ist der Nachweis geführt.
	if _, err := svc.Add(ctx, SupplyEvidenceRequest{
		InvoiceID: invoice.ID, Kind: string(accounting.EvidenceArrival),
		Issuer: "Client SARL", Date: "2026-04-10",
	}); err != nil {
		t.Fatalf("Gelangensbestätigung: %v", err)
	}
	after, err := svc.Report(ctx, 2026)
	if err != nil {
		t.Fatalf("Bericht: %v", err)
	}
	if after.Incomplete != 0 {
		t.Errorf("%d unvollständig — mit der Gelangensbestätigung trägt der Nachweis: %s",
			after.Incomplete, after.Rows[0].Status.Reason)
	}
}

// -------------------------------------------------------------------------
// Gebäudestichtag, Erhaltungsaufwand und Nutzungsdauer
// -------------------------------------------------------------------------

// Ohne Stichtag gibt es keinen Gebäudesatz. Der frühere Rückfall auf das
// Anschaffungsdatum lieferte für ein Betriebsgebäude mit altem Bauantrag 3 %
// statt der 2 % des § 7 Abs. 4 Satz 1 Nr. 2 Buchst. b EStG — eine stille
// Überschreibung der AfA um die Hälfte.
func TestBuildingNeedsItsReferenceDate(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)

	_, err := svc.Save(context.Background(), &domain.FixedAsset{
		Name: "Betriebsgebäude", Class: domain.AssetClassTangible,
		Account: "0240", DepreciationAccount: "6221",
		AcquisitionDate: "2026-02-01", AcquisitionCost: 50_000_000,
		UsefulLifeMonths: 400, Method: domain.DepreciationBuildingLinear,
	})
	if err == nil {
		t.Fatal("ohne Stichtag darf kein Gebäude mit festem Satz angelegt werden")
	}
	if !strings.Contains(err.Error(), "Bauantrag") {
		t.Errorf("die Meldung muss den maßgeblichen Stichtag benennen: %v", err)
	}

	// Mit dem Bauantrag von 1980 gilt der Satz von 2 %.
	asset, err := svc.Save(context.Background(), &domain.FixedAsset{
		Name: "Betriebsgebäude", Class: domain.AssetClassTangible,
		Account: "0240", DepreciationAccount: "6221",
		AcquisitionDate: "2026-02-01", AcquisitionCost: 50_000_000,
		UsefulLifeMonths: 400, Method: domain.DepreciationBuildingLinear,
		BuildingReferenceDate: "1980-05-01",
	})
	if err != nil {
		t.Fatalf("Gebäude mit Stichtag: %v", err)
	}
	detail, err := svc.Get(context.Background(), asset.ID)
	if err != nil {
		t.Fatalf("Detailansicht: %v", err)
	}
	// 2 % von 500.000 € sind 10.000 €; im Anschaffungsjahr elf Zwölftel davon.
	if len(detail.Schedule) == 0 || detail.Schedule[0].Amount != 916_667 {
		t.Errorf("AfA des Anschaffungsjahres %+v — erwartet 2 %% zeitanteilig (9.166,67 €), "+
			"nicht 3 %%", detail.Schedule)
	}
}

// Die 15-%-Prüfung kommt mit der Buchung zurück und nicht nur ins Protokoll.
// Wer bucht, liest kein Protokoll.
func TestMaintenanceReturnsTheNearAcquisitionCheck(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.assets(t)
	vendor := env.vendor(t, "Bau GmbH", "DE", "")
	building := env.building(t, svc)

	first, err := svc.BookMaintenance(ctx, MaintenanceRequest{
		AssetID: building.ID, Date: "2026-06-01", Amount: 2_000_000,
		TaxTreatment: domain.TaxTreatmentDomestic, TaxRate: domain.TaxRateStandard,
		Settlement: SettlementOpen, ContactID: vendor.ID,
		Note: "Dachsanierung, Wiederherstellung des ursprünglichen Zustands",
	})
	if err != nil {
		t.Fatalf("erster Erhaltungsaufwand: %v", err)
	}
	if first.NearAcquisition == nil || !first.NearAcquisition.Applicable {
		t.Fatalf("die Prüfung gehört ins Ergebnis: %+v", first.NearAcquisition)
	}
	if first.NearAcquisition.Exceeded {
		t.Errorf("20.000 € bleiben unter dem Rahmen von 30.000 €: %s", first.NearAcquisition.Note)
	}

	second, err := svc.BookMaintenance(ctx, MaintenanceRequest{
		AssetID: building.ID, Date: "2027-03-01", Amount: 1_200_000,
		TaxTreatment: domain.TaxTreatmentDomestic, TaxRate: domain.TaxRateStandard,
		Settlement: SettlementOpen, ContactID: vendor.ID,
		Note: "Innenausbau, Instandsetzung",
	})
	if err != nil {
		t.Fatalf("zweiter Erhaltungsaufwand: %v", err)
	}
	if second.NearAcquisition == nil || !second.NearAcquisition.Exceeded {
		t.Fatalf("mit 32.000 € ist der Rahmen gerissen: %+v", second.NearAcquisition)
	}

	// Und der Vorschlag lässt sich ausführen: die Aufwandsbuchungen werden auf
	// das Gebäudekonto umgebucht.
	updated, err := svc.CapitalizeNearAcquisitionCost(ctx, CapitalizeNearAcquisitionCostRequest{
		AssetID: building.ID, Date: "2027-03-31",
		Reason: "Instandsetzung übersteigt 15 % der Anschaffungskosten (§ 6 Abs. 1 Nr. 1a EStG)",
	})
	if err != nil {
		t.Fatalf("Aktivierung: %v", err)
	}
	_ = updated
	// Die Anschaffungskosten sind fortgeschrieben — sichtbar mit Blick auf das
	// Jahr der Umbuchung, denn die Kartei rechnet je Geschäftsjahr.
	svc.SetFiscalYear(2027)
	detail, err := svc.Get(ctx, building.ID)
	if err != nil {
		t.Fatalf("Detailansicht: %v", err)
	}
	if detail.Asset.Cost != 20_000_000+3_200_000 {
		t.Errorf("Anschaffungskosten %s € — erwartet 200.000 € zuzüglich 32.000 €",
			detail.Asset.Cost)
	}
	svc.SetFiscalYear(2026)

	entries, err := env.journalRepo.FindAll(ctx, 2027)
	if err != nil {
		t.Fatalf("Journal: %v", err)
	}
	var booking *domain.JournalEntry
	for i := range entries {
		if strings.Contains(entries[i].Description, "Anschaffungsnahe Herstellungskosten") {
			booking = &entries[i]
		}
	}
	if booking == nil {
		t.Fatalf("die Umbuchung steht nicht im Journal")
	}
	if got := debitOn(booking, "0240"); got != 3_200_000 {
		t.Errorf("SOLL Gebäude %s € — erwartet 32.000,00 €", got)
	}

	// Ein zweiter Lauf findet nichts mehr: die Bewegungen zählen nicht mehr
	// gegen den Rahmen.
	if _, err := svc.CapitalizeNearAcquisitionCost(ctx, CapitalizeNearAcquisitionCostRequest{
		AssetID: building.ID, Reason: "Wiederholung",
	}); err == nil {
		t.Error("derselbe Aufwand darf nicht zweimal aktiviert werden")
	}
}

// building ist ein Betriebsgebäude über 200.000 € mit Bauantrag nach 1985.
func (e *testEnv) building(t *testing.T, svc *AssetService) *domain.FixedAsset {
	t.Helper()
	asset, err := svc.Save(context.Background(), &domain.FixedAsset{
		Name: "Betriebsgebäude", Class: domain.AssetClassTangible,
		Account: "0240", DepreciationAccount: "6221",
		AcquisitionDate: "2026-01-02", AcquisitionCost: 20_000_000,
		UsefulLifeMonths: 400, Method: domain.DepreciationBuildingLinear,
		BuildingReferenceDate: "2005-03-01",
	})
	if err != nil {
		t.Fatalf("Gebäude: %v", err)
	}
	return asset
}

// Die Begründungspflicht für eine abweichende Nutzungsdauer gilt dem Wahlrecht
// des BMF-Schreibens vom 22.02.2022 und nicht jeder AfA-Tabelle. Für einen Pkw
// steht dort ein Erfahrungswert, und eine Begründung dafür zu verlangen ginge
// über die Vorschrift hinaus — und sperrte bestehende Anlagen beim Speichern.
func TestUsefulLifeReasonOnlyForTheDigitalProposal(t *testing.T) {
	env := newTestEnv(t)
	svc := env.assets(t)

	if _, err := svc.Save(context.Background(), &domain.FixedAsset{
		Name: "Firmenwagen", Class: domain.AssetClassTangible,
		Account: "0520", DepreciationAccount: "6222",
		AcquisitionDate: "2026-01-15", AcquisitionCost: 4_000_000,
		UsefulLifeMonths: 84, Method: domain.DepreciationLinear,
	}); err != nil {
		t.Errorf("eine abweichende Nutzungsdauer nach der AfA-Tabelle braucht keine Begründung: %v",
			err)
	}

	// Für die Hardware bleibt sie Pflicht: zwölf Monate sind ein Wahlrecht.
	if _, err := svc.Save(context.Background(), &domain.FixedAsset{
		Name: "Server", Class: domain.AssetClassTangible,
		Account: "0690", DepreciationAccount: "6220",
		AcquisitionDate: "2026-01-15", AcquisitionCost: 900_000,
		UsefulLifeMonths: 60, Method: domain.DepreciationLinear,
	}); err == nil {
		t.Error("die Abweichung vom BMF-Vorschlag ist zu begründen")
	}
}

// -------------------------------------------------------------------------
// Geschenke: Empfänger und Generalumkehr
// -------------------------------------------------------------------------

// Der Empfänger eines Geschenks wird aus der Kartei geladen. „Kontakt 17" ist
// keine Aufzeichnung nach § 4 Abs. 7 EStG, und eine Kennung, die es nicht gibt,
// führte eine Freigrenze über einen Empfänger, den es nicht gibt.
func TestGiftRecipientComesFromTheContact(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.gifts(t)
	vendor := env.vendor(t, "Präsente GmbH", "DE", "")
	recipient := env.customer(t, "Dr. Meyer", "DE", "")

	req := env.receipt(t, vendor.ID, "geschenke", 4_000,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Positions[0].Gift = &GiftInput{ContactID: recipient.ID, Occasion: "Jubiläum"}

	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Geschenk buchen: %v", err)
	}
	if len(entry.Gifts) != 1 || entry.Gifts[0].RecipientName != "Dr. Meyer" {
		t.Fatalf("Aufzeichnung %+v — erwartet den Namen aus der Kartei", entry.Gifts)
	}

	// Ein Empfänger, den es nicht gibt, wird abgewiesen.
	bad := env.receipt(t, vendor.ID, "geschenke", 1_000,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	bad.Positions[0].Gift = &GiftInput{ContactID: 9999}
	if _, err := env.posting.PostIncomingReceipt(ctx, bad); err == nil {
		t.Error("ein unbekannter Empfänger darf keine Aufzeichnung tragen")
	}
}

// Die Generalumkehr trägt die Aufzeichnung mit — wie bei der Bewirtung. Gezählt
// wird sie nicht: sonst liefe die Freigrenze über das stornierte Geschenk weiter.
func TestReversalCarriesTheGiftRecord(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.gifts(t)
	vendor := env.vendor(t, "Präsente GmbH", "DE", "")

	entry, err := env.posting.PostIncomingReceipt(ctx,
		env.giftReceipt(t, vendor.ID, "Dr. Meyer", 4_000))
	if err != nil {
		t.Fatalf("Geschenk: %v", err)
	}
	reversal, err := env.journal.Reverse(ctx, entry.ID, "Falscher Empfänger")
	if err != nil {
		t.Fatalf("Storno: %v", err)
	}
	if len(reversal.Gifts) != 1 || reversal.Gifts[0].RecipientName != "Dr. Meyer" {
		t.Errorf("die Umkehr muss die Aufzeichnung mittragen: %+v", reversal.Gifts)
	}

	records, err := svc.GiftsInYear(ctx, 2026)
	if err != nil {
		t.Fatalf("Geschenkkartei: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("%d Geschenke in der Kartei — das stornierte zählt nicht mehr mit", len(records))
	}
}

// -------------------------------------------------------------------------
// Ungeprüfte E-Rechnung
// -------------------------------------------------------------------------

// Ein Rechnungsdatensatz, den nie jemand geprüft hat, ist kein geprüfter. Ihn
// durchzulassen, weil kein Fehler gemeldet wurde, hieße Schweigen als Zustimmung
// zu lesen — es wurde nur nie gefragt.
func TestUncheckedEInvoiceIsAFinding(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Agentur GmbH", "DE", "")
	receipt, err := env.receipts.File(ctx, FileReceiptRequest{
		Direction: domain.DirectionIncoming, FiscalYear: env.fiscalYear,
		Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, Path: env.writeTempFile(t, "rechnung.pdf", minimalPDF)},
			{Role: domain.ReceiptRoleStructured, FileName: "rechnung.xml",
				Content: []byte("<Invoice/>"), Derived: true},
		},
	})
	if err != nil {
		t.Fatalf("Beleg ablegen: %v", err)
	}

	req := env.receipt(t, vendor.ID, "fremdleistungen", 100_000,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.ReceiptID = receipt.ID

	preview, err := env.posting.PreviewIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	found := false
	for _, f := range preview.InputTaxFindings {
		if f.Code == findingEInvoiceUnchecked {
			found = true
		}
	}
	if !found {
		t.Fatalf("kein Befund zum ungeprüften Rechnungsdatensatz: %+v", preview.InputTaxFindings)
	}
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err == nil {
		t.Error("ohne Prüfergebnis und ohne Grund wird nicht mit Vorsteuer gebucht")
	}

	req.OverrideReason = "Datensatz vom Lieferanten nachgereicht, Prüfung folgt"
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
		t.Errorf("mit festgehaltenem Grund muss die Buchung durchgehen: %v", err)
	}
}

// -------------------------------------------------------------------------
// Der Rechnungsweg gegen das Bundeszentralamt
// -------------------------------------------------------------------------

// Das Ausstellen einer steuerfreien ig. Lieferung geht durch die
// Bestätigungsabfrage: bestätigt sie, wird die Nummer vergeben; weist sie
// zurück, nicht; antwortet niemand, nur mit festgehaltenem Grund. Geprüft wird
// hier der Weg über Issue und nicht EnsureConfirmed allein — dazwischen liegen
// prepareForIssue und die Frage, ob der Empfänger überhaupt im übrigen
// Gemeinschaftsgebiet sitzt.
func TestIssuingAnICSupplyGoesThroughTheConfirmation(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Der Ergebniscode wird je Aufruf umgestellt; jede Rechnung geht an einen
	// eigenen Empfänger, damit die Frist von 90 Tagen die zweite Abfrage nicht
	// aus dem Zwischenspeicher beantwortet.
	status := "evatr-0000"
	code := http.StatusOK
	hang := false
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hang {
			<-release
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = fmt.Fprintf(w,
			`{"status":%q,"id":"2026-0001","ergFirmenname":"A","ergOrt":"A"}`, status)
	}))
	defer func() {
		close(release)
		server.Close()
	}()

	vatIDs := env.vatIDs(t, server.URL)
	// Der Ausfall wird über einen Server geprüft, der nicht antwortet. Mit der
	// Voreinstellung von zehn Sekunden wartete der Test genau so lange real;
	// geprüft wird aber die Behandlung der Frist und nicht ihre Länge.
	vatIDs.SetClientFactory(func(endpoint string) *vatid.Client {
		client := vatid.New(endpoint)
		client.SetTimeout(200 * time.Millisecond)
		return client
	})
	invoices := env.invoicesWired(t)
	invoices.SetVatIDConfirmer(vatIDs)

	issue := func(customer *domain.Contact, date, reason string) (*domain.Invoice, error) {
		inv := env.simpleInvoice(customer.ID, date, 500_000)
		inv.TaxTreatment = domain.TaxTreatmentIntraCommunitySupply
		inv.Items[0].TaxRate = domain.TaxRateNone
		inv.VatIDOverrideReason = reason
		return inv, invoices.Issue(ctx, inv)
	}

	confirmed, err := issue(env.customer(t, "Client SARL", "FR", "FR12345678901"), "2026-03-01", "")
	if err != nil {
		t.Fatalf("mit bestätigter USt-IdNr. muss die Rechnung hinausgehen: %v", err)
	}
	if confirmed.InvoiceNumber == "" {
		t.Error("die bestätigte Lieferung bekommt ihre Nummer")
	}

	// Das Amt weist die Nummer zurück: keine Nummer, keine Rechnung.
	status, code = "evatr-2001", http.StatusNotFound
	rejected, err := issue(env.customer(t, "Client Nord SARL", "FR", "FR99999999999"), "2026-03-02", "")
	if err == nil {
		t.Fatal("eine zurückgewiesene USt-IdNr. trägt keine steuerfreie Lieferung")
	}
	if !strings.Contains(err.Error(), "evatr-2001") {
		t.Errorf("die Meldung muss den Ergebniscode nennen: %v", err)
	}
	if rejected.InvoiceNumber != "" {
		t.Errorf("Nummer %q — die abgelehnte Rechnung darf keine verbrauchen", rejected.InvoiceNumber)
	}

	// Und ein Amt, das nicht antwortet, ist kein negatives Ergebnis: die
	// Rechnung geht nur mit einem festgehaltenen Grund hinaus.
	hang = true
	blocked := env.customer(t, "Client Süd SARL", "FR", "FR11111111111")
	if _, err := issue(blocked, "2026-03-03", ""); err == nil {
		t.Fatal("ohne Antwort und ohne Grund geht die Rechnung nicht hinaus")
	} else if strings.Contains(err.Error(), "nicht bestätigt") {
		t.Errorf("eine ausgebliebene Antwort ist kein negatives Ergebnis: %v", err)
	}
	withReason, err := issue(blocked, "2026-03-03",
		"Bundeszentralamt antwortet nicht, Nummer aus dem Vorjahr bestätigt")
	if err != nil {
		t.Fatalf("mit festgehaltenem Grund muss die Rechnung hinausgehen: %v", err)
	}
	if withReason.InvoiceNumber == "" {
		t.Error("die übersteuerte Lieferung bekommt ihre Nummer")
	}
}

// -------------------------------------------------------------------------
// Der Vorsteuerausschluss an der Zeile
// -------------------------------------------------------------------------

// Null Promille abziehbar ist etwas anderes als „kein Vorsteuerschlüssel".
// Solange beides als null an der Zeile stand, ließ sich der Ausschluss des
// § 15 Abs. 1a UStG von der vollen Abziehbarkeit nicht unterscheiden.
func TestExcludedInputTaxCarriesItsOwnMarker(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.gifts(t)
	vendor := env.vendor(t, "Yachthafen GmbH", "DE", "")

	excluded, err := env.posting.PostIncomingReceipt(ctx, env.receipt(
		t, vendor.ID, "repraesentation", 100_000,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic))
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	if got := shareOn(excluded, "6645"); got != domain.InputTaxExcluded {
		t.Errorf("Vorsteueranteil %d — erwartet den Marker %d für den Ausschluss nach "+
			"§ 15 Abs. 1a UStG", got, domain.InputTaxExcluded)
	}

	plain, err := env.posting.PostIncomingReceipt(ctx, env.receipt(
		t, vendor.ID, "fremdleistungen", 100_000,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic))
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	if got := shareOn(plain, "5906"); got != 0 {
		t.Errorf("Vorsteueranteil %d — bei vollem Abzug bleibt das Feld leer, damit die "+
			"Hash-Kette bestehender Buchungen unverändert bleibt", got)
	}
}

// shareOn liefert den Vorsteueranteil der ersten Zeile auf einem Konto.
func shareOn(entry *domain.JournalEntry, account string) int {
	for _, l := range entry.Lines {
		if l.Account == account {
			return l.InputTaxShare
		}
	}
	return 0
}
