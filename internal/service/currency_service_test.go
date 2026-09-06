package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/currency"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// ecbServer ist ein Kursdienst aus Papier. Er antwortet mit einem festen Kurs
// und zählt die Aufrufe — die Frage, ob ein zweites Mal gefragt wird, ist Teil
// der Prüfung.
func ecbServer(t *testing.T, rate string, date string) (*httptest.Server, *int) {
	t.Helper()
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(
			`{"amount":1.0,"base":"EUR","date":"` + date + `","rates":{"USD":` + rate + `}}`))
	}))
	t.Cleanup(server.Close)
	return server, &calls
}

func (e *testEnv) currency(t *testing.T, endpoint string) *CurrencyService {
	t.Helper()
	svc := NewCurrencyService(
		repository.NewExchangeRateRepository(e.db),
		currency.New(endpoint),
		repository.NewAuditRepository(e.db),
	)
	svc.SetJournalRepo(e.journalRepo)
	svc.SetJournalService(e.journal)
	return svc
}

// Der Kurs zum Belegdatum kommt vom Kursdienst und wird in der Historie
// abgelegt. Die zweite Frage nach demselben Tag geht nicht mehr ins Netz.
func TestExchangeRateIsFetchedOnceAndKept(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, calls := ecbServer(t, "1.0850", "2026-03-10")
	svc := env.currency(t, server.URL)

	rate, err := svc.RateAt(ctx, "USD", "2026-03-10")
	if err != nil {
		t.Fatalf("Kurs holen: %v", err)
	}
	if rate.RateMicros != 1_085_000 {
		t.Errorf("Kurs %d Millionstel — erwartet 1.085.000", rate.RateMicros)
	}
	if !strings.Contains(rate.Source, "EZB") {
		t.Errorf("Quelle %q — sie muss die Herkunft nennen", rate.Source)
	}

	again, err := svc.RateAt(ctx, "USD", "2026-03-10")
	if err != nil {
		t.Fatalf("Kurs erneut: %v", err)
	}
	if again.RateMicros != rate.RateMicros {
		t.Errorf("beim zweiten Mal %d statt %d Millionstel", again.RateMicros, rate.RateMicros)
	}
	if *calls != 1 {
		t.Errorf("%d Aufrufe — ein gespeicherter Kurs wird nicht erneut geholt", *calls)
	}
}

// Ohne Netz und ohne gespeicherten Kurs kommt ein Fehler und kein Wert. Die
// frühere Fassung lieferte still 1,0 zurück — ein Dollarbetrag wurde damit als
// Eurobetrag gebucht, in richtiger Größenordnung und mit falschem Wert.
func TestExchangeRateNeverGuesses(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, _ := ecbServer(t, "1.0850", "2026-03-10")
	url := server.URL
	server.Close()
	svc := env.currency(t, url)

	if _, err := svc.RateAt(ctx, "USD", "2026-03-10"); err == nil {
		t.Fatal("ohne Kurs muss ein Fehler kommen und kein Rückfallwert")
	}

	// Von Hand erfasst mit Quelle geht es weiter — aber nur mit Quelle.
	if _, err := svc.SaveRate(ctx, domain.ExchangeRate{
		Currency: "USD", Date: "2026-03-10", RateMicros: 1_090_000,
	}); err == nil {
		t.Error("ein von Hand erfasster Kurs ohne Quelle ist eine Behauptung")
	}
	saved, err := svc.SaveRate(ctx, domain.ExchangeRate{
		Currency: "USD", Date: "2026-03-10", RateMicros: 1_090_000,
		Source: "EZB-Referenzkurs, abgelesen am 11.03.2026",
	})
	if err != nil {
		t.Fatalf("Kurs von Hand: %v", err)
	}
	if !saved.Manual {
		t.Error("ein von Hand erfasster Kurs muss als solcher erkennbar sein")
	}
	got, err := svc.RateAt(ctx, "USD", "2026-03-10")
	if err != nil {
		t.Fatalf("Kurs lesen: %v", err)
	}
	if got.RateMicros != 1_090_000 {
		t.Errorf("Kurs %d Millionstel — erwartet den von Hand erfassten", got.RateMicros)
	}
}

// Für die Umsatzsteuer gilt der monatliche Durchschnittskurs des BMF
// (§ 16 Abs. 6 UStG). Liegt er vor, weicht die Bemessungsgrundlage vom Aufwand
// ab, und die Differenz ist Kursaufwand oder Kursertrag.
func TestVatAverageRateSplitsBaseFromExpense(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, _ := ecbServer(t, "1.0850", "2026-03-10")
	svc := env.currency(t, server.URL)

	// Ohne Durchschnittskurs bleibt es beim Tageskurs.
	plain, err := svc.Convert(ctx, "USD", "2026-03-10", 1_085_000)
	if err != nil {
		t.Fatalf("Umrechnung: %v", err)
	}
	if plain.Amount != 1_000_000 {
		t.Errorf("Eurobetrag %s € — erwartet 10.000,00 €", plain.Amount)
	}
	if plain.TaxBaseAmount != plain.Amount || plain.Difference != 0 {
		t.Errorf("ohne Durchschnittskurs sind Aufwand und Bemessungsgrundlage gleich: %s / %s",
			plain.Amount, plain.TaxBaseAmount)
	}
	if !strings.Contains(plain.Note, "§ 16 Abs. 6") {
		t.Errorf("der Hinweis muss die Vorschrift nennen: %s", plain.Note)
	}

	if _, err := svc.SaveVatRate(ctx, domain.VatExchangeRate{
		Month: "2026-03", Currency: "USD", RateMicros: 1_100_000,
	}); err != nil {
		t.Fatalf("Durchschnittskurs: %v", err)
	}

	withVat, err := svc.Convert(ctx, "USD", "2026-03-10", 1_085_000)
	if err != nil {
		t.Fatalf("Umrechnung: %v", err)
	}
	if withVat.Amount != 1_000_000 {
		t.Errorf("Aufwand %s € — er folgt weiter dem Tageskurs", withVat.Amount)
	}
	// 10.850 USD zu 1,10 sind 9.863,64 €.
	if withVat.TaxBaseAmount != 986_364 {
		t.Errorf("Bemessungsgrundlage %s € — erwartet 9.863,64 €", withVat.TaxBaseAmount)
	}
	if withVat.Difference != withVat.TaxBaseAmount-withVat.Amount {
		t.Errorf("Differenz %s € — sie ist der Abstand zwischen beiden Kursen", withVat.Difference)
	}
	if withVat.VatRate == nil || withVat.VatRate.Month != "2026-03" {
		t.Error("der verwendete Durchschnittskurs gehört ins Ergebnis")
	}
}

// Die Durchschnittskurse kommen als CSV herein — die amtliche
// Veröffentlichung ist ein PDF, und was daraus wird, tippt oder kopiert jemand.
func TestVatRateCSVImport(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.currency(t, "")

	path := filepath.Join(t.TempDir(), "kurse.csv")
	content := "Monat;Waehrung;Kurs\n2026-01;USD;1,0912\n2026-02;USD;1,0834\n2026-02;CHF;0,9412\n" +
		"2026-03;;1,10\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("CSV schreiben: %v", err)
	}

	result, err := svc.ImportVatRatesCSV(ctx, path)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if result.Imported != 3 {
		t.Errorf("%d übernommen — erwartet drei", result.Imported)
	}
	if result.Skipped != 1 || len(result.Problems) != 1 {
		t.Errorf("%d übergangen mit %d Meldungen — die Zeile ohne Währung muss auffallen",
			result.Skipped, len(result.Problems))
	}

	rate, err := svc.VatRateFor(ctx, "USD", "2026-01-15")
	if err != nil {
		t.Fatalf("Durchschnittskurs: %v", err)
	}
	if rate == nil || rate.RateMicros != 1_091_200 {
		t.Errorf("Kurs %+v — erwartet 1.091.200 Millionstel", rate)
	}
}

// -------------------------------------------------------------------------
// Stichtagsbewertung (§ 256a HGB)
// -------------------------------------------------------------------------

// foreignOpenItem bucht eine Forderung oder Verbindlichkeit in Fremdwährung.
func (e *testEnv) foreignOpenItem(
	t *testing.T, contact *domain.Contact, number string, amount domain.Cents, rateMicros int64,
) *domain.JournalEntry {
	t.Helper()
	lines := []domain.JournalLine{
		{Side: domain.SideDebit, Account: contact.LedgerAccount, ContactID: &contact.ID,
			Amount: amount},
		{Side: domain.SideCredit, Account: "4400", Amount: amount},
	}
	if contact.Type == domain.ContactTypeVendor {
		lines = []domain.JournalLine{
			{Side: domain.SideDebit, Account: "5906", Amount: amount},
			{Side: domain.SideCredit, Account: contact.LedgerAccount, ContactID: &contact.ID,
				Amount: amount},
		}
	}
	entry, err := e.journal.Post(context.Background(), &domain.JournalEntry{
		BookingDate: "2026-12-01", DocumentDate: "2026-12-01",
		ServiceDateFrom: "2026-12-01", ServiceDateTo: "2026-12-01",
		Description: "Fremdwährungsposten " + number,
		Source:      domain.EntrySourceInvoice, DocumentNumber: number,
		TaxTreatment: domain.TaxTreatmentNotTaxable,
		ContactID:    &contact.ID,
		Currency:     "USD", ExchangeRateMicros: rateMicros,
		ExchangeRateSource: "EZB-Referenzkurs", ExchangeRateDate: "2026-12-01",
		Lines: lines,
	})
	if err != nil {
		t.Fatalf("Fremdwährungsposten %s: %v", number, err)
	}
	return entry
}

// § 256a HGB: bei einer Restlaufzeit bis zu einem Jahr wirken Gewinn und Verlust
// erfolgswirksam, darüber nur der Verlust. Und die Bewertung wird am ersten Tag
// des Folgejahres wieder aufgelöst — sie gilt dem Stichtag und nicht dem Posten.
func TestCurrencyValuationFollowsTheOneYearRule(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, _ := ecbServer(t, "1.0000", "2026-12-31")
	svc := env.currency(t, server.URL)
	svc.SetClosingService(env.closing(t))
	svc.SetOpenItemSource(NewPaymentService(
		env.journal, env.journalRepo,
		repository.NewPaymentAllocationRepository(env.db),
		env.contactRepo, repository.NewBankRepository(env.db), env.fiscalYear))

	// Der Buchkurs ist 1,10 USD je Euro, der Stichtagskurs 1,00. Eine Forderung
	// über 11.000 USD stand mit 10.000 € in den Büchern und ist am Stichtag
	// 11.000 € wert: ein Gewinn von 1.000 €.
	shortTerm := env.customer(t, "Client Inc.", "US", "")
	shortTerm.PaymentTermsDays = 30
	if err := env.contacts.SaveContact(ctx, shortTerm); err != nil {
		t.Fatalf("Kunde: %v", err)
	}
	longTerm := env.customer(t, "Client Long Inc.", "US", "")
	longTerm.PaymentTermsDays = 500
	if err := env.contacts.SaveContact(ctx, longTerm); err != nil {
		t.Fatalf("Kunde: %v", err)
	}
	vendor := env.vendor(t, "Supplier Inc.", "US", "")
	vendor.PaymentTermsDays = 30
	if err := env.contacts.SaveContact(ctx, vendor); err != nil {
		t.Fatalf("Lieferant: %v", err)
	}

	env.foreignOpenItem(t, shortTerm, "RE-USD-1", 1_000_000, 1_100_000)
	env.foreignOpenItem(t, longTerm, "RE-USD-2", 1_000_000, 1_100_000)
	env.foreignOpenItem(t, vendor, "ER-USD-1", 1_000_000, 1_100_000)

	preview, err := svc.PreviewCurrencyValuation(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if len(preview.Items) != 3 {
		t.Fatalf("%d Posten bewertet — erwartet drei", len(preview.Items))
	}
	byNumber := map[string]ForeignCurrencyValuationItem{}
	for _, item := range preview.Items {
		byNumber[strings.Fields(item.Description)[0]] = item
	}

	short := byNumber["RE-USD-1"]
	if !short.Gain || !short.Recognised || short.Amount != 100_000 {
		t.Errorf("kurzfristige Forderung: Gewinn=%v erfasst=%v %s € — erwartet einen erfassten "+
			"Gewinn von 1.000,00 €", short.Gain, short.Recognised, short.Amount)
	}
	long := byNumber["RE-USD-2"]
	if !long.Gain || long.Recognised {
		t.Errorf("langfristige Forderung: Gewinn=%v erfasst=%v — ein unrealisierter Gewinn über "+
			"einem Jahr wird nicht gebucht (%s)", long.Gain, long.Recognised, long.Reason)
	}
	payable := byNumber["ER-USD-1"]
	if payable.Gain || !payable.Recognised || payable.Amount != 100_000 {
		t.Errorf("Verbindlichkeit: Gewinn=%v erfasst=%v %s € — ein höherer Eurowert der Schuld ist "+
			"ein Verlust und wird immer erfasst", payable.Gain, payable.Recognised, payable.Amount)
	}
	if preview.TotalGain != 100_000 || preview.TotalLoss != 100_000 {
		t.Errorf("Ertrag %s €, Aufwand %s € — erwartet je 1.000,00 €",
			preview.TotalGain, preview.TotalLoss)
	}

	booked, err := svc.BookCurrencyValuation(ctx, 2026)
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	if booked.EntryNumber == "" {
		t.Fatalf("die Bewertungsbuchung fehlt")
	}
	if booked.ReversalDate != "2027-01-01" {
		t.Errorf("Auflösung am %s — erwartet den ersten Tag des Folgejahres", booked.ReversalDate)
	}

	valuation := env.entryByNumber(t, 2026, booked.EntryNumber)
	if got := debitOn(valuation, accounting.CurrencyLossAccount); got != 100_000 {
		t.Errorf("Kursaufwand %s € — erwartet 1.000,00 € auf %s",
			got, accounting.CurrencyLossAccount)
	}
	var gain domain.Cents
	for _, l := range valuation.Lines {
		if l.Side == domain.SideCredit && l.Account == accounting.CurrencyGainAccount {
			gain += l.Amount
		}
	}
	if gain != 100_000 {
		t.Errorf("Kursertrag %s € — erwartet 1.000,00 € auf %s", gain, accounting.CurrencyGainAccount)
	}

	// Die Auflösung folgt mit dem Saldenvortrag ins neue Jahr — wie die der
	// Rechnungsabgrenzung. Sie sofort mitzubuchen wäre der schlechtere Weg: ein
	// noch nicht angelegtes oder bereits festgestelltes Folgejahr ließe die
	// Bewertung ohne ihre Auflösung stehen, und ein erneuter Lauf würde mit
	// „bereits gebucht" abgewiesen.
	reversals, err := svc.ReverseInto(ctx, 2027)
	if err != nil {
		t.Fatalf("Auflösung: %v", err)
	}
	if len(reversals) != 1 {
		t.Fatalf("%d Auflösungsbuchungen — erwartet eine", len(reversals))
	}
	reversal := &reversals[0]
	if got := debitOn(reversal, accounting.CurrencyGainAccount); got != 100_000 {
		t.Errorf("die Auflösung muss den Ertrag zurücknehmen: %s € auf %s",
			got, accounting.CurrencyGainAccount)
	}

	// Ein zweiter Lauf über dasselbe Jahr wird abgewiesen: eine doppelt gebuchte
	// Bewertung wäre eine doppelte Wertänderung.
	if _, err := svc.BookCurrencyValuation(ctx, 2026); err == nil {
		t.Error("die Bewertung eines Jahres wird nur einmal gebucht")
	}
}

// entryByNumber sucht eine Buchung eines Jahres über ihre Nummer.
func (e *testEnv) entryByNumber(t *testing.T, year int, number string) *domain.JournalEntry {
	t.Helper()
	entries, err := e.journalRepo.FindAll(context.Background(), year)
	if err != nil {
		t.Fatalf("Journal %d: %v", year, err)
	}
	for i := range entries {
		if entries[i].EntryNumber == number {
			return &entries[i]
		}
	}
	t.Fatalf("die Buchung %s steht nicht im Journal %d", number, year)
	return nil
}
