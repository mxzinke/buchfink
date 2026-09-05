package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die zweite Nachbesserung der Welle 5c: der Weg zurück aus einer stornierten
// Abschlussbuchung, der Vorsteuerausschluss bei Steuerfällen mit eigener
// Steuerschuld und die Umbuchung der Geschenke als ein Vorgang.

// -------------------------------------------------------------------------
// § 15a UStG: nach dem Storno wird erneut berichtigt
// -------------------------------------------------------------------------

// Der Storno ist der Weg zurück, den jede Sperre dieser Bausteine empfiehlt.
// Solange die Buchungskennung am Verwendungsanteil stehen blieb, führte er in
// die Sackgasse: das Verzeichnis meldete weiter „gebucht", der Jahreslauf fand
// nichts zu tun, und die Berichtigung dieses Jahres kam nie mehr in die
// Kennziffer 64.
func TestInputTaxCorrectionCanBeBookedAgainAfterAReversal(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.inputTax(t)

	correction, err := svc.Register(ctx, RegisterInputTaxRequest{
		Label: "Pkw AN-2026-0001", Account: "0520",
		AcquisitionDate: "2026-01-15",
		NetAmount:       4_000_000, InputTaxAmount: 760_000,
	})
	if err != nil {
		t.Fatalf("Aufnahme ins Verzeichnis: %v", err)
	}
	if _, err := svc.SaveUsage(ctx, SaveInputTaxUsageRequest{
		CorrectionID: correction.ID, FiscalYear: 2028, Permille: 600,
		Reason: "Kfz ab 2028 zu 40 % für steuerfreie Vermietung eingesetzt",
	}); err != nil {
		t.Fatalf("Verwendungsanteil: %v", err)
	}
	if _, err := svc.BookYear(ctx, 2028); err != nil {
		t.Fatalf("erste Buchung: %v", err)
	}

	first := env.closingEntry(t, 2028, inputTaxCorrectionDocument(2028))
	if _, err := env.journal.Reverse(ctx, first.ID,
		"Verwendungsanteil beruhte auf einem falschen Fahrtenbuch"); err != nil {
		t.Fatalf("Generalumkehr: %v", err)
	}

	// Nach dem Storno ist das Jahr wieder offen — und zwar in allen drei
	// Antworten des Dienstes.
	view, err := svc.Year(ctx, 2028)
	if err != nil {
		t.Fatalf("Verzeichnis: %v", err)
	}
	if view.Rows[0].Booked {
		t.Error("eine per Generalumkehr zurückgenommene Berichtigung ist keine gebuchte")
	}
	if view.Rows[0].EntryNumber != "" {
		t.Errorf("Buchungsnummer %q — die Buchung steht nicht mehr", view.Rows[0].EntryNumber)
	}
	if view.TotalAmount != -60_800 {
		t.Errorf("zu buchen %s € — nach dem Storno steht die Berichtigung wieder an",
			view.TotalAmount)
	}

	// Der Anteil lässt sich wieder ändern: genau das war der Rat der Meldung.
	if _, err := svc.SaveUsage(ctx, SaveInputTaxUsageRequest{
		CorrectionID: correction.ID, FiscalYear: 2028, Permille: 500,
		Reason: "Fahrtenbuch berichtigt: nur noch 50 % abziehbare Verwendung",
	}); err != nil {
		t.Fatalf("Anteil nach dem Storno ändern: %v", err)
	}

	booked, err := svc.BookYear(ctx, 2028)
	if err != nil {
		t.Fatalf("zweite Buchung: %v", err)
	}
	if !booked.Rows[0].Booked {
		t.Fatal("nach dem zweiten Lauf muss die Berichtigung wieder als gebucht gelten")
	}
	// 7.600 € × 50 Prozentpunkte / 5 Jahre = 760 € zurückzuzahlen.
	if booked.Rows[0].Assessment.Amount != -76_000 {
		t.Errorf("Berichtigung %s € — erwartet -760,00 € aus dem geänderten Anteil",
			booked.Rows[0].Assessment.Amount)
	}

	second := env.standingClosingEntry(t, 2028, inputTaxCorrectionDocument(2028))
	if second.ID == first.ID {
		t.Fatal("die zweite Berichtigung muss eine neue Buchung sein")
	}
	var taxLine *domain.JournalLine
	for i := range second.Lines {
		if second.Lines[i].TaxKey == accounting.TaxKeyInputTaxCorrection {
			taxLine = &second.Lines[i]
		}
	}
	if taxLine == nil || taxLine.Amount != 76_000 {
		t.Fatalf("die neue Buchung trägt keine Zeile mit %s über 760,00 €: %+v",
			accounting.TaxKeyInputTaxCorrection, second.Lines)
	}
}

// closingEntry sucht die erste Buchung eines Jahres unter einer Belegnummer.
func (e *testEnv) closingEntry(t *testing.T, year int, document string) *domain.JournalEntry {
	t.Helper()
	entries, err := e.journalRepo.FindAll(context.Background(), year)
	if err != nil {
		t.Fatalf("Journal %d: %v", year, err)
	}
	for i := range entries {
		if entries[i].DocumentNumber == document && entries[i].Kind != domain.EntryKindReversal {
			return &entries[i]
		}
	}
	t.Fatalf("keine Buchung %s im Journal %d", document, year)
	return nil
}

// standingClosingEntry sucht die Buchung eines Jahres, die noch steht — die
// stornierten bleiben draußen.
func (e *testEnv) standingClosingEntry(t *testing.T, year int, document string) *domain.JournalEntry {
	t.Helper()
	ctx := context.Background()
	entries, err := e.journalRepo.FindAll(ctx, year)
	if err != nil {
		t.Fatalf("Journal %d: %v", year, err)
	}
	for i := range entries {
		if entries[i].DocumentNumber != document || entries[i].Kind == domain.EntryKindReversal {
			continue
		}
		reversal, err := e.journalRepo.FindReversalOf(ctx, entries[i].ID)
		if err != nil {
			t.Fatalf("Storno zu %s: %v", entries[i].EntryNumber, err)
		}
		if reversal == nil {
			return &entries[i]
		}
	}
	t.Fatalf("keine stehende Buchung %s im Journal %d", document, year)
	return nil
}

// -------------------------------------------------------------------------
// Fremdwährungsbewertung: nach dem Storno wird erneut bewertet
// -------------------------------------------------------------------------

// Dieselbe Sackgasse an der Stichtagsbewertung: die Schrittliste meldete nach
// der Generalumkehr richtig „storniert, offen", und der Baustein ließ sich
// trotzdem nicht wiederholen.
func TestCurrencyValuationCanBeBookedAgainAfterAReversal(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, _ := ecbServer(t, "1.0000", "2026-12-31")
	svc := env.currency(t, server.URL)
	svc.SetClosingService(env.closing(t))
	svc.SetOpenItemSource(NewPaymentService(
		env.journal, env.journalRepo,
		repository.NewPaymentAllocationRepository(env.db),
		env.contactRepo, repository.NewBankRepository(env.db), env.fiscalYear))

	customer := env.customer(t, "Client Inc.", "US", "")
	customer.PaymentTermsDays = 30
	if err := env.contacts.SaveContact(ctx, customer); err != nil {
		t.Fatalf("Kunde: %v", err)
	}
	env.foreignOpenItem(t, customer, "RE-USD-1", 1_000_000, 1_100_000)

	booked, err := svc.BookCurrencyValuation(ctx, 2026)
	if err != nil {
		t.Fatalf("erste Bewertung: %v", err)
	}
	first := env.entryByNumber(t, 2026, booked.EntryNumber)

	// Ein zweiter Lauf wird abgewiesen, solange die Bewertung steht.
	if _, err := svc.BookCurrencyValuation(ctx, 2026); err == nil {
		t.Fatal("eine stehende Bewertung wird nicht ein zweites Mal gebucht")
	}

	if _, err := env.journal.Reverse(ctx, first.ID,
		"Stichtagskurs war der falsche"); err != nil {
		t.Fatalf("Generalumkehr: %v", err)
	}

	again, err := svc.BookCurrencyValuation(ctx, 2026)
	if err != nil {
		t.Fatalf("nach dem Storno muss die Bewertung erneut buchbar sein: %v", err)
	}
	if again.EntryNumber == "" || again.EntryNumber == booked.EntryNumber {
		t.Fatalf("Buchung %q — erwartet eine neue Bewertungsbuchung", again.EntryNumber)
	}

	// Und aufgelöst wird nur die stehende: die stornierte ist bereits
	// zurückgenommen, eine Auflösung dazu wäre eine Wertänderung ohne Grund.
	reversals, err := svc.ReverseInto(ctx, 2027)
	if err != nil {
		t.Fatalf("Auflösung: %v", err)
	}
	if len(reversals) != 1 {
		t.Fatalf("%d Auflösungsbuchungen — erwartet genau eine für die stehende Bewertung",
			len(reversals))
	}
}

// -------------------------------------------------------------------------
// Vorsteuerausschluss und eigene Steuerschuld
// -------------------------------------------------------------------------

// § 15 Abs. 1a UStG nimmt den Abzug, nicht die Steuerschuld. Beim
// innergemeinschaftlichen Erwerb und beim Reverse Charge entstehen beide aus
// derselben Bemessungsgrundlage; wird sie für den Ausschluss auf null gesetzt,
// fällt mit der Vorsteuerzeile auch die geschuldete Steuer weg, und die
// Voranmeldung meldet einen steuerpflichtigen Erwerb schlicht nicht.
func TestExcludedInputTaxIsRefusedWhereTheTaxIsOwed(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.gifts(t)
	vendor := env.vendor(t, "Yacht Charter SARL", "FR", "FR12345678901")

	for _, treatment := range []domain.TaxTreatment{
		domain.TaxTreatmentIntraCommunityAcquisition,
		domain.TaxTreatmentReverseCharge,
	} {
		req := env.receipt(t, vendor.ID, "repraesentation", 100_000,
			domain.TaxRateStandard, treatment)
		_, err := env.posting.PostIncomingReceipt(ctx, req)
		if err == nil {
			t.Fatalf("%s: ein Vorsteuerausschluss darf die geschuldete Steuer nicht "+
				"verschwinden lassen", treatment)
		}
		if !strings.Contains(err.Error(), "§ 15 Abs. 1a UStG") {
			t.Errorf("%s: die Meldung muss die Vorschrift nennen: %v", treatment, err)
		}
	}

	// Im Inland bleibt der Ausschluss, was er war: die Steuer gehört zum
	// Aufwand, eine Vorsteuerzeile entsteht nicht.
	entry, err := env.posting.PostIncomingReceipt(ctx, env.receipt(
		t, vendor.ID, "repraesentation", 100_000,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic))
	if err != nil {
		t.Fatalf("Inlandsumsatz: %v", err)
	}
	if got := debitOn(entry, "6645"); got != 119_000 {
		t.Errorf("Aufwand %s € — erwartet 1.190,00 € brutto (§ 9b Abs. 1 EStG)", got)
	}
	if debitOn(entry, domain.AccountVorsteuer19) != 0 {
		t.Error("zu einer Aufwendung des § 4 Abs. 5 Satz 1 Nr. 4 EStG gehört keine Vorsteuer")
	}
}

// Dasselbe für das Geschenk über der Freigrenze: auch dort fällt der
// Vorsteuerabzug weg, und auch dort bleibt die Erwerbsteuer geschuldet.
func TestGiftOverTheLimitIsRefusedOnAnIntraCommunityAcquisition(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.gifts(t)
	vendor := env.vendor(t, "Cadeaux SARL", "FR", "FR12345678901")

	// Das erste Geschenk bleibt unter der Freigrenze und geht durch.
	if _, err := env.posting.PostIncomingReceipt(ctx,
		env.giftReceipt(t, vendor.ID, "Dr. Meyer", 4_000)); err != nil {
		t.Fatalf("erstes Geschenk: %v", err)
	}

	over := env.giftReceipt(t, vendor.ID, "Dr. Meyer", 2_000)
	over.TaxTreatment = domain.TaxTreatmentIntraCommunityAcquisition
	if _, err := env.posting.PostIncomingReceipt(ctx, over); err == nil {
		t.Fatal("ein Geschenk über der Freigrenze beim ig. Erwerb darf nicht die " +
			"Erwerbsteuer entfallen lassen")
	} else if !strings.Contains(err.Error(), "Steuerschuld") {
		t.Errorf("die Meldung muss die fortbestehende Steuerschuld nennen: %v", err)
	}
}

// -------------------------------------------------------------------------
// Umbuchung der Geschenke: ein Vorgang, ein Tag
// -------------------------------------------------------------------------

// Storno und Neubuchung tragen denselben Tag. Vorher datierte die Umkehr auf
// heute und die Neubuchung auf das Datum der ursprünglichen Buchung; lag das in
// einer festgeschriebenen Periode, ging die Umkehr durch und die Neubuchung
// nicht — der Aufwand war danach ganz aus den Büchern.
func TestGiftRebookingStaysOutOfACommittedPeriod(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.journal.SetFestschreibungRepo(repository.NewFestschreibungRepository(env.db))
	svc := env.gifts(t)
	svc.SetTxRunner(repository.NewTxRunner(env.db))
	vendor := env.vendor(t, "Präsente GmbH", "DE", "")

	if _, err := env.posting.PostIncomingReceipt(ctx,
		env.giftReceipt(t, vendor.ID, "Dr. Meyer", 4_000)); err != nil {
		t.Fatalf("erstes Geschenk: %v", err)
	}
	if _, err := env.posting.PostIncomingReceipt(ctx,
		env.giftReceipt(t, vendor.ID, "Dr. Meyer", 2_000)); err != nil {
		t.Fatalf("zweites Geschenk: %v", err)
	}

	// Der Zeitraum, in dem beide Geschenke stehen, wird festgeschrieben.
	festschreibung := repository.NewFestschreibungRepository(env.db)
	if err := festschreibung.Create(ctx, &domain.Festschreibung{
		FiscalYear: 2026, PeriodType: "quarter", PeriodLabel: "1. Quartal 2026",
		CutoffDate: "2026-03-31", ChainHead: domain.GenesisHash,
	}); err != nil {
		t.Fatalf("Festschreibung: %v", err)
	}

	report, err := svc.NonDeductibleReport(ctx, 2026)
	if err != nil {
		t.Fatalf("Bericht: %v", err)
	}
	key := report.Recipients[0].RecipientKey

	// Ein ausdrücklich in die gesperrte Periode gelegtes Datum wird ganz
	// abgewiesen — und zwar bevor das erste Mal geschrieben wird.
	before := env.countEntries(t, 2026)
	if _, err := svc.RebookGiftsForRecipient(ctx, RebookGiftsRequest{
		FiscalYear: 2026, RecipientKey: key, Date: "2026-03-10",
		Reason: "Freigrenze überschritten",
	}); err == nil {
		t.Fatal("eine Umbuchung in eine festgeschriebene Periode muss abgewiesen werden")
	}
	if got := env.countEntries(t, 2026); got != before {
		t.Fatalf("%d statt %d Buchungen — die abgewiesene Umbuchung darf nichts "+
			"zurücklassen, auch keine Generalumkehr", got, before)
	}

	// Ohne Datum läuft sie vollständig auf den Tag der Korrektur.
	rebooking, err := svc.RebookGiftsForRecipient(ctx, RebookGiftsRequest{
		FiscalYear: 2026, RecipientKey: key,
		Reason: "Freigrenze mit dem zweiten Geschenk überschritten",
	})
	if err != nil {
		t.Fatalf("Umbuchung auf den freien Tag: %v", err)
	}
	if len(rebooking.Reversals) != 1 || len(rebooking.Rebookings) != 1 {
		t.Fatalf("%d Stornos und %d Neubuchungen — erwartet je eine",
			len(rebooking.Reversals), len(rebooking.Rebookings))
	}
	today := time.Now().Format("2006-01-02")
	rebooked := env.entryByNumberAnyYear(t, rebooking.Rebookings[0])
	if rebooked.BookingDate != today {
		t.Errorf("Neubuchung zum %s — erwartet den Tag der Korrektur (%s), an dem auch die "+
			"Generalumkehr steht", rebooked.BookingDate, today)
	}
	reversal := env.entryByNumberAnyYear(t, rebooking.Reversals[0])
	if reversal.BookingDate != rebooked.BookingDate {
		t.Errorf("Storno zum %s, Neubuchung zum %s — beide gehören auf denselben Tag",
			reversal.BookingDate, rebooked.BookingDate)
	}

	after, err := svc.NonDeductibleReport(ctx, 2026)
	if err != nil {
		t.Fatalf("Bericht nach der Umbuchung: %v", err)
	}
	if len(after.Recipients[0].ToRebook) != 0 {
		t.Errorf("nach der Umbuchung steht nichts mehr offen: %+v", after.Recipients[0].ToRebook)
	}
}

// entryByNumberAnyYear sucht eine Buchung über alle Geschäftsjahre. Eine
// Korrektur trägt den Tag ihrer Erstellung und liegt deshalb nicht zwingend im
// Jahr der Buchung, die sie korrigiert.
func (e *testEnv) entryByNumberAnyYear(t *testing.T, number string) *domain.JournalEntry {
	t.Helper()
	ctx := context.Background()
	years, err := e.journalRepo.GetAvailableFiscalYears(ctx)
	if err != nil {
		t.Fatalf("Geschäftsjahre: %v", err)
	}
	for _, year := range years {
		entries, err := e.journalRepo.FindAll(ctx, year)
		if err != nil {
			t.Fatalf("Journal %d: %v", year, err)
		}
		for i := range entries {
			if entries[i].EntryNumber == number {
				return &entries[i]
			}
		}
	}
	t.Fatalf("die Buchung %s steht in keinem Geschäftsjahr", number)
	return nil
}

// countEntries zählt die Buchungen eines Geschäftsjahres.
func (e *testEnv) countEntries(t *testing.T, year int) int {
	t.Helper()
	entries, err := e.journalRepo.FindAll(context.Background(), year)
	if err != nil {
		t.Fatalf("Journal %d: %v", year, err)
	}
	return len(entries)
}
