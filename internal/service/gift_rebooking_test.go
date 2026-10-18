package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Dasselbe für das Geschenk über der Freigrenze: der Vorsteuerabzug fällt weg,
// die Erwerbsteuer bleibt geschuldet.
func TestGiftOverTheLimitKeepsTheAcquisitionTax(t *testing.T) {
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
	entry, err := env.posting.PostIncomingReceipt(ctx, over)
	if err != nil {
		t.Fatalf("Geschenk über der Freigrenze beim ig. Erwerb: %v", err)
	}
	if got := creditOn(entry, domain.AccountUmsatzsteuerIG19); got != 380 {
		t.Errorf("Erwerbsteuer %s € — erwartet 3,80 € auf 20,00 € Bemessungsgrundlage", got)
	}
	if got := debitOn(entry, domain.AccountVorsteuerIG19); got != 0 {
		t.Errorf("Vorsteuer %s € — über der Freigrenze gibt es keinen Abzug "+
			"(§ 15 Abs. 1a UStG)", got)
	}
}

// -------------------------------------------------------------------------
// Umbuchung der Geschenke: ein Vorgang, ein Tag
// -------------------------------------------------------------------------

// Storno und Neubuchung haben denselben Tag. Vorher datierte die Umkehr auf
// heute und die Neubuchung auf das Datum der ursprünglichen Buchung; lag das in
// einer festgeschriebenen Periode, ging die Umkehr durch und die Neubuchung
// nicht — der Aufwand war danach ganz aus den Büchern.
func TestGiftRebookingStaysOutOfACommittedPeriod(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.journal.SetFestschreibungRepo(repository.NewFestschreibungRepository(env.db))
	svc := env.gifts(t)
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

// Die Umbuchung der Geschenke bei einem Beleg mit mehreren Positionen und
// mehreren Empfängern.
//
// Der Beleg ist der Regelfall und nicht der Sonderfall: „zehn Präsentkörbe an
// zehn Empfänger sind ein Beleg und eine Buchung, aber zehn Aufzeichnungen".
// Die Umbuchung darf deshalb weder fremden Aufwand mitnehmen noch die
// Aufzeichnungen der übrigen Empfänger verlieren.

// entryByNumber sucht eine Buchung des Geschäftsjahres über ihre Nummer.
func entryByNumber(t *testing.T, env *testEnv, number string) *domain.JournalEntry {
	t.Helper()
	entries, err := env.journalRepo.FindAll(context.Background(), env.fiscalYear)
	if err != nil {
		t.Fatalf("Journal lesen: %v", err)
	}
	for i := range entries {
		if entries[i].EntryNumber == number {
			return &entries[i]
		}
	}
	t.Fatalf("die Buchung %s steht nicht im Journal", number)
	return nil
}

// Ein Beleg über ein Geschenk (40 €) und Bürobedarf (100 €): nach der Umbuchung
// steht nur das Geschenk brutto auf 6620. Der Bürobedarf bleibt, wo er war, und
// behält seinen Vorsteuerabzug.
func TestRebookingKeepsTheOtherPositionsOfTheReceipt(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.gifts(t)
	vendor := env.vendor(t, "Präsente GmbH", "DE", "")

	mixed := env.receipt(t, vendor.ID, "geschenke", 4_000, domain.TaxRateStandard,
		domain.TaxTreatmentDomestic)
	mixed.Positions[0].Gift = &GiftInput{Name: "Dr. Meyer", Occasion: "Jubiläum"}
	mixed.Positions = append(mixed.Positions, ReceiptPosition{
		Account: "6815", Net: 10_000, TaxRate: domain.TaxRateStandard, Text: "Bürobedarf",
	})
	first, err := env.posting.PostIncomingReceipt(ctx, mixed)
	if err != nil {
		t.Fatalf("gemischter Beleg: %v", err)
	}
	if debitOn(first, "6610") != 4_000 || debitOn(first, "6815") != 10_000 {
		t.Fatalf("der gemischte Beleg ist nicht wie erwartet gebucht: %s", accountsOf(first))
	}
	if got := debitOn(first, domain.AccountVorsteuer19); got != 2_660 {
		t.Fatalf("Vorsteuer %s € — erwartet 26,60 € aus 140,00 € netto", got)
	}

	// Das zweite Geschenk reißt die Freigrenze.
	if _, err := env.posting.PostIncomingReceipt(ctx,
		env.giftReceipt(t, vendor.ID, "Dr. Meyer", 2_000)); err != nil {
		t.Fatalf("zweites Geschenk: %v", err)
	}

	report, err := svc.NonDeductibleReport(ctx, env.fiscalYear)
	if err != nil {
		t.Fatalf("Bericht: %v", err)
	}
	rebooking, err := svc.RebookGiftsForRecipient(ctx, RebookGiftsRequest{
		FiscalYear:   env.fiscalYear,
		RecipientKey: report.Recipients[0].RecipientKey,
		Reason:       "Freigrenze mit dem zweiten Geschenk überschritten",
	})
	if err != nil {
		t.Fatalf("Umbuchung: %v", err)
	}
	if len(rebooking.Rebookings) != 1 {
		t.Fatalf("%d Neubuchungen — erwartet eine", len(rebooking.Rebookings))
	}

	rebooked := entryByNumber(t, env, rebooking.Rebookings[0])
	if !rebooked.IsBalanced() {
		t.Errorf("die Neubuchung ist nicht ausgeglichen: Soll %s € / Haben %s €",
			rebooked.DebitTotal(), rebooked.CreditTotal())
	}
	if got := debitOn(rebooked, "6620"); got != 4_760 {
		t.Errorf("auf 6620 stehen %s € — erwartet das Geschenk brutto (40,00 + 7,60)", got)
	}
	if got := debitOn(rebooked, "6815"); got != 10_000 {
		t.Errorf("der Bürobedarf steht mit %s € da — er ist von der Freigrenze nicht berührt", got)
	}
	if got := debitOn(rebooked, domain.AccountVorsteuer19); got != 1_900 {
		t.Errorf("Vorsteuer %s € — erwartet 19,00 € aus dem Bürobedarf; nur die des Geschenks "+
			"entfällt (§ 15 Abs. 1a UStG)", got)
	}
	if debitOn(rebooked, "6610") != 0 {
		t.Errorf("nach der Umbuchung steht auf 6610 nichts mehr: %s", accountsOf(rebooked))
	}
}

// Ein Beleg mit Geschenken an zwei Empfänger: die Umbuchung des einen lässt die
// Aufzeichnung des anderen nicht verschwinden.
func TestRebookingKeepsTheRecordsOfTheOtherRecipients(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.gifts(t)
	vendor := env.vendor(t, "Präsente GmbH", "DE", "")

	both := env.receipt(t, vendor.ID, "geschenke", 4_000, domain.TaxRateStandard,
		domain.TaxTreatmentDomestic)
	both.Positions[0].Gift = &GiftInput{Name: "Dr. Meyer", Occasion: "Jubiläum"}
	both.Positions = append(both.Positions, ReceiptPosition{
		PostingGroup: "geschenke", Net: 3_000, TaxRate: domain.TaxRateStandard,
		Gift: &GiftInput{Name: "Frau Schulz", Occasion: "Jubiläum"},
	})
	if _, err := env.posting.PostIncomingReceipt(ctx, both); err != nil {
		t.Fatalf("Beleg mit zwei Empfängern: %v", err)
	}
	if _, err := env.posting.PostIncomingReceipt(ctx,
		env.giftReceipt(t, vendor.ID, "Dr. Meyer", 2_000)); err != nil {
		t.Fatalf("zweites Geschenk an Dr. Meyer: %v", err)
	}

	report, err := svc.NonDeductibleReport(ctx, env.fiscalYear)
	if err != nil {
		t.Fatalf("Bericht: %v", err)
	}
	key := ""
	for _, r := range report.Recipients {
		if r.RecipientName == "Dr. Meyer" {
			key = r.RecipientKey
		}
	}
	if key == "" {
		t.Fatalf("Dr. Meyer fehlt im Bericht: %+v", report.Recipients)
	}
	rebooking, err := svc.RebookGiftsForRecipient(ctx, RebookGiftsRequest{
		FiscalYear: env.fiscalYear, RecipientKey: key,
		Reason: "Freigrenze überschritten",
	})
	if err != nil {
		t.Fatalf("Umbuchung: %v", err)
	}

	rebooked := entryByNumber(t, env, rebooking.Rebookings[0])
	if !rebooked.IsBalanced() {
		t.Errorf("die Neubuchung ist nicht ausgeglichen: Soll %s € / Haben %s €",
			rebooked.DebitTotal(), rebooked.CreditTotal())
	}
	if len(rebooked.Gifts) != 2 {
		t.Fatalf("die Neubuchung hat %d Aufzeichnungen — erwartet beide Empfänger: %+v",
			len(rebooked.Gifts), rebooked.Gifts)
	}
	for _, g := range rebooked.Gifts {
		switch g.RecipientName {
		case "Dr. Meyer":
			if !g.NonDeductible || g.Account != "6620" {
				t.Errorf("das Geschenk an Dr. Meyer gehört nicht abziehbar auf 6620: %+v", g)
			}
		case "Frau Schulz":
			if g.NonDeductible || g.Account != "6610" {
				t.Errorf("das Geschenk an Frau Schulz bleibt abziehbar: %+v", g)
			}
		default:
			t.Errorf("unbekannter Empfänger in der Neubuchung: %+v", g)
		}
	}
	if got := debitOn(rebooked, "6620"); got != 4_760 {
		t.Errorf("auf 6620 stehen %s € — erwartet 47,60 € für Dr. Meyer", got)
	}
	if got := debitOn(rebooked, "6610"); got != 3_000 {
		t.Errorf("auf 6610 stehen %s € — erwartet 30,00 € für Frau Schulz", got)
	}
	if got := debitOn(rebooked, domain.AccountVorsteuer19); got != 570 {
		t.Errorf("Vorsteuer %s € — erwartet 5,70 € aus dem Geschenk an Frau Schulz", got)
	}

	// Und die Kartei kennt beide Empfänger danach noch.
	after, err := svc.NonDeductibleReport(ctx, env.fiscalYear)
	if err != nil {
		t.Fatalf("Bericht nach der Umbuchung: %v", err)
	}
	found := map[string]GiftRecipientRow{}
	for _, r := range after.Recipients {
		found[r.RecipientName] = r
	}
	schulz, ok := found["Frau Schulz"]
	if !ok {
		t.Fatalf("Frau Schulz ist nach der Umbuchung aus der Kartei verschwunden: %+v", after.Recipients)
	}
	if schulz.Total != 3_000 || schulz.OverLimit {
		t.Errorf("Frau Schulz steht mit %s € und OverLimit=%v da — erwartet 30,00 € unter der Grenze",
			schulz.Total, schulz.OverLimit)
	}
	meyer := found["Dr. Meyer"]
	if len(meyer.ToRebook) != 0 {
		t.Errorf("an Dr. Meyer steht nach der Umbuchung nichts mehr offen: %+v", meyer.ToRebook)
	}
}

// Zwei Geschenke an denselben Empfänger auf einem Beleg: eine Buchung, ein
// Storno — und beide Aufzeichnungen nicht abziehbar.
func TestRebookingHandlesTwoGiftsToTheSameRecipientInOneBooking(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.gifts(t)
	vendor := env.vendor(t, "Präsente GmbH", "DE", "")

	req := env.receipt(t, vendor.ID, "geschenke", 3_000, domain.TaxRateStandard,
		domain.TaxTreatmentDomestic)
	req.Positions[0].Gift = &GiftInput{Name: "Dr. Meyer", Occasion: "Jubiläum"}
	req.Positions = append(req.Positions, ReceiptPosition{
		PostingGroup: "geschenke", Net: 3_000, TaxRate: domain.TaxRateStandard,
		Gift: &GiftInput{Name: "Dr. Meyer", Occasion: "Weihnachten"},
	})
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
		t.Fatalf("Beleg mit zwei Geschenken an denselben Empfänger: %v", err)
	}

	report, err := svc.NonDeductibleReport(ctx, env.fiscalYear)
	if err != nil {
		t.Fatalf("Bericht: %v", err)
	}
	if len(report.Recipients) != 1 || !report.Recipients[0].OverLimit {
		t.Fatalf("60 € an einen Empfänger reißen die Freigrenze: %+v", report.Recipients)
	}
	rebooking, err := svc.RebookGiftsForRecipient(ctx, RebookGiftsRequest{
		FiscalYear: env.fiscalYear, RecipientKey: report.Recipients[0].RecipientKey,
		Reason: "Freigrenze überschritten",
	})
	if err != nil {
		t.Fatalf("Umbuchung: %v", err)
	}
	if len(rebooking.Reversals) != 1 || len(rebooking.Rebookings) != 1 {
		t.Fatalf("%d Stornos und %d Neubuchungen — eine Buchung wird einmal zurückgenommen",
			len(rebooking.Reversals), len(rebooking.Rebookings))
	}
	rebooked := entryByNumber(t, env, rebooking.Rebookings[0])
	if got := debitOn(rebooked, "6620"); got != 7_140 {
		t.Errorf("auf 6620 stehen %s € — erwartet beide Geschenke brutto (2 × 35,70)", got)
	}
	if debitOn(rebooked, domain.AccountVorsteuer19) != 0 {
		t.Errorf("die Vorsteuerzeile muss vollständig entfallen: %s", accountsOf(rebooked))
	}
	if !rebooked.IsBalanced() {
		t.Errorf("die Neubuchung ist nicht ausgeglichen: Soll %s € / Haben %s €",
			rebooked.DebitTotal(), rebooked.CreditTotal())
	}

	after, err := svc.NonDeductibleReport(ctx, env.fiscalYear)
	if err != nil {
		t.Fatalf("Bericht nach der Umbuchung: %v", err)
	}
	if len(after.Recipients[0].ToRebook) != 0 {
		t.Errorf("nach der Umbuchung steht nichts mehr offen: %+v", after.Recipients[0].ToRebook)
	}
	if after.Recipients[0].Total != 6_000 {
		t.Errorf("die Kartei führt %s € — erwartet beide Geschenke", after.Recipients[0].Total)
	}
}

// Wo sich die Zeile eines Geschenks nicht eindeutig zuordnen lässt, wird die
// Umbuchung abgewiesen statt geraten.
func TestRebookingRefusesWhenTheGiftLineIsNotIdentifiable(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.gifts(t)
	vendor := env.vendor(t, "Präsente GmbH", "DE", "")

	// Zwei Geschenke über denselben Betrag an zwei Empfänger auf einem Beleg
	// sind zuzuordnen (zwei Zeilen, zwei Aufzeichnungen). Nicht zuzuordnen ist
	// eine Aufzeichnung, deren Zeile es nicht gibt — hier nachgestellt über eine
	// Buchung, die von Hand geschrieben wurde.
	entry := &domain.JournalEntry{
		BookingDate: "2026-03-10", DocumentDate: "2026-03-10",
		ServiceDateFrom: "2026-03-10", ServiceDateTo: "2026-03-10",
		Description: "Geschenk ohne passende Zeile",
		Source:      domain.EntrySourceManual,
		// Ohne Steuerzeile ist der Steuerfall nicht der Inlandsumsatz: die
		// Buchung erfasst den Bruttobetrag des Geschenks (§ 15 Abs. 1a UStG).
		TaxTreatment: domain.TaxTreatmentNotTaxable,
		ContactID:    &vendor.ID,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "6610", Amount: 9_000},
			{Side: domain.SideCredit, Account: "1800", Amount: 9_000},
		},
		Gifts: []domain.GiftRecord{{
			FiscalYear: 2026, Date: "2026-03-10", RecipientName: "Dr. Meyer",
			NetAmount: 6_000, Account: "6610",
		}},
	}
	if _, err := env.journal.Post(ctx, entry); err != nil {
		t.Fatalf("Buchung von Hand: %v", err)
	}

	report, err := svc.NonDeductibleReport(ctx, env.fiscalYear)
	if err != nil {
		t.Fatalf("Bericht: %v", err)
	}
	_, err = svc.RebookGiftsForRecipient(ctx, RebookGiftsRequest{
		FiscalYear: env.fiscalYear, RecipientKey: report.Recipients[0].RecipientKey,
		Reason: "Freigrenze überschritten",
	})
	if err == nil {
		t.Fatal("eine Buchung, deren Geschenkzeile nicht zu finden ist, darf nicht umgebucht werden")
	}
	if !strings.Contains(err.Error(), "zu stornieren") {
		t.Errorf("die Meldung muss den Weg nennen: %v", err)
	}
}
