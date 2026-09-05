package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die Umbuchung der Geschenke bei einem Beleg mit mehreren Positionen und
// mehreren Empfängern (Runde 3).
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
		t.Fatalf("die Neubuchung trägt %d Aufzeichnungen — erwartet beide Empfänger: %+v",
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
		Description:  "Geschenk ohne passende Zeile",
		Source:       domain.EntrySourceManual,
		TaxTreatment: domain.TaxTreatmentDomestic,
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
	if !strings.Contains(err.Error(), "von Hand") {
		t.Errorf("die Meldung muss den Weg nennen: %v", err)
	}
}

// -------------------------------------------------------------------------
// Die Vorsteuer der Zugangsbuchung im Verzeichnis nach § 15a UStG
// -------------------------------------------------------------------------

// Der Weg vom Beleg ins Verzeichnis: der Pkw wird über den Eingangsbeleg
// gebucht, erscheint als Zugangskandidat und wird daraus aktiviert. Die
// Vorsteuer tippt dabei niemand ab — sie steht in der Buchung.
func TestAcquisitionCandidateCarriesTheInputTaxIntoTheRegister(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	assets := env.assets(t)
	register := env.inputTax(t)
	assets.SetInputTaxRegister(register)
	vendor := env.vendor(t, "Autohaus GmbH", "DE", "")

	req := env.receipt(t, vendor.ID, "", 4_000_000, domain.TaxRateStandard,
		domain.TaxTreatmentDomestic)
	req.Positions[0].PostingGroup = ""
	req.Positions[0].Account = "0520"
	req.Positions[0].Text = "Firmenwagen"
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Zugangsbeleg: %v", err)
	}
	if debitOn(entry, domain.AccountVorsteuer19) != 760_000 {
		t.Fatalf("Vorsteuer %s € — erwartet 7.600,00 €", debitOn(entry, domain.AccountVorsteuer19))
	}

	candidates, err := assets.AcquisitionCandidates(ctx)
	if err != nil {
		t.Fatalf("Zugangskandidaten: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("%d Kandidaten — erwartet die eine Zugangsbuchung: %+v", len(candidates), candidates)
	}
	if candidates[0].InputTaxAmount != 760_000 || candidates[0].InputTaxPermille != 1000 {
		t.Fatalf("Kandidat mit %s € Vorsteuer zu %d ‰ — erwartet 7.600,00 € bei voller Verwendung",
			candidates[0].InputTaxAmount, candidates[0].InputTaxPermille)
	}

	// Aktiviert wird aus dem Kandidaten heraus; die Vorsteuer setzt der Aufrufer
	// nicht.
	entryID := candidates[0].EntryID
	asset, err := assets.Save(ctx, &domain.FixedAsset{
		Name: "Firmenwagen", Class: domain.AssetClassTangible,
		Account: "0520", DepreciationAccount: "6222",
		AcquisitionDate: "2026-03-10", AcquisitionCost: 4_000_000,
		UsefulLifeMonths: 72, Method: domain.DepreciationLinear,
		AcquisitionEntryID: &entryID,
	})
	if err != nil {
		t.Fatalf("Aktivierung: %v", err)
	}
	if asset.InputTaxAmount != 760_000 {
		t.Errorf("das Anlagegut trägt %s € Vorsteuer — erwartet 7.600,00 € aus der Zugangsbuchung",
			asset.InputTaxAmount)
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
		t.Errorf("Eintrag mit %s € zu %d ‰ — erwartet 7.600,00 € bei voller Verwendung",
			row.InputTaxAmount, row.OriginalPermille)
	}
	if row.AssetID == nil || *row.AssetID != asset.ID {
		t.Errorf("der Eintrag muss auf das Anlagegut zeigen: %+v", row.AssetID)
	}
}

// Ein geteilter Vorsteuerabzug geht mit dem Anteil ins Verzeichnis: 600 ‰
// gezogen, 7.600 € Vorsteuer angefallen. Genau diese beiden Zahlen braucht
// § 15a UStG später.
func TestAcquisitionCandidateCarriesTheInputTaxShare(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	assets := env.assets(t)
	register := env.inputTax(t)
	assets.SetInputTaxRegister(register)
	vendor := env.vendor(t, "Autohaus GmbH", "DE", "")

	req := env.receipt(t, vendor.ID, "", 4_000_000, domain.TaxRateStandard,
		domain.TaxTreatmentDomestic)
	req.Positions[0].PostingGroup = ""
	req.Positions[0].Account = "0520"
	req.Positions[0].InputTaxShare = 600
	req.Positions[0].InputTaxShareReason = "Kfz zu 60 % betrieblich genutzt, Fahrtenbuch 2026"
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
		t.Fatalf("Zugangsbeleg: %v", err)
	}

	candidates, err := assets.AcquisitionCandidates(ctx)
	if err != nil {
		t.Fatalf("Zugangskandidaten: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("%d Kandidaten — erwartet einen: %+v", len(candidates), candidates)
	}
	if candidates[0].InputTaxAmount != 760_000 || candidates[0].InputTaxPermille != 600 {
		t.Fatalf("Kandidat mit %s € zu %d ‰ — erwartet die volle Vorsteuer von 7.600,00 € und den "+
			"gezogenen Anteil von 600 ‰", candidates[0].InputTaxAmount, candidates[0].InputTaxPermille)
	}

	entryID := candidates[0].EntryID
	asset, err := assets.Save(ctx, &domain.FixedAsset{
		Name: "Firmenwagen", Class: domain.AssetClassTangible,
		Account: "0520", DepreciationAccount: "6222",
		AcquisitionDate: "2026-03-10", AcquisitionCost: 4_304_000,
		UsefulLifeMonths: 72, Method: domain.DepreciationLinear,
		AcquisitionEntryID: &entryID,
	})
	if err != nil {
		t.Fatalf("Aktivierung: %v", err)
	}
	if asset.InputTaxAmount != 760_000 || asset.InputTaxPermille != 600 {
		t.Fatalf("das Anlagegut trägt %s € zu %d ‰", asset.InputTaxAmount, asset.InputTaxPermille)
	}
	view, err := register.Year(ctx, 2026)
	if err != nil {
		t.Fatalf("Verzeichnis: %v", err)
	}
	if len(view.Rows) != 1 || view.Rows[0].Correction.OriginalPermille != 600 {
		t.Fatalf("Verzeichnis %+v — erwartet einen Eintrag mit 600 ‰", view.Rows)
	}
}

// -------------------------------------------------------------------------
// Die Berichtigung eines Jahres wird nicht zweimal gebucht
// -------------------------------------------------------------------------

// Bricht ein Lauf zwischen Buchung und Vermerk ab, steht die Berichtigung im
// Journal und gilt im Verzeichnis als offen. Der nächste Lauf darf sie dann
// nicht ein zweites Mal buchen — sie stünde doppelt in Kennziffer 64.
func TestInputTaxCorrectionIsNotBookedTwiceWhenTheNoteIsMissing(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	repo := repository.NewInputTaxCorrectionRepository(env.db)
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

	// Der Vermerk fällt weg — der Zustand, den ein Abbruch zwischen Buchung und
	// Vermerk hinterlässt.
	if err := repo.SaveUsage(ctx, &domain.InputTaxUsage{
		CorrectionID: correction.ID, FiscalYear: 2028,
		Permille: 600, Confirmed: true, Amount: -60_800,
		Reason: "Vermerk fehlt",
	}); err != nil {
		t.Fatalf("Vermerk zurücksetzen: %v", err)
	}
	view, err := svc.Year(ctx, 2028)
	if err != nil {
		t.Fatalf("Verzeichnis: %v", err)
	}
	if view.Rows[0].Booked {
		t.Fatal("der Testaufbau verlangt einen Eintrag ohne Vermerk")
	}

	if _, err := svc.BookYear(ctx, 2028); err == nil {
		t.Fatal("die Berichtigung darf nicht ein zweites Mal gebucht werden")
	} else if !strings.Contains(err.Error(), "bereits gebucht") {
		t.Errorf("die Meldung muss die stehende Buchung nennen: %v", err)
	}
}

// -------------------------------------------------------------------------
// Der Folgebefund endet mit der nachgeholten Bestätigung
// -------------------------------------------------------------------------

// Wer die Rechnung mit einem Grund ausgestellt und die Abfrage danach nachgeholt
// hat, hat den Befund erledigt. Vorher meldete der Prüflauf weiter, es liege
// keine gültige Bestätigung vor — und hängte im selben Satz an, die Nummer sei
// bestätigt worden.
func TestConfirmedVatIDEndsTheFollowUpFinding(t *testing.T) {
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

	// Ohne Bestätigung steht der Folgebefund.
	run := runChecks(t, checks, "2026-12-31")
	if findingByRule(run, domain.CheckRuleICSupplyUnconfirmed) == nil {
		t.Fatalf("ohne Bestätigung muss der Folgebefund stehen: %+v", run.Findings)
	}

	// Die Abfrage wird nachgeholt.
	if err := repository.NewVatIDCheckRepository(env.db).Save(ctx, &domain.VatIDCheck{
		ContactID: customer.ID, VatID: "FR12345678901",
		CheckedAt: time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		Status:    domain.VatIDValid, ResultCode: "200", RequestID: "BZST-4711",
	}); err != nil {
		t.Fatalf("Bestätigung speichern: %v", err)
	}

	after := runChecks(t, checks, "2026-12-31")
	if found := findingByRule(after, domain.CheckRuleICSupplyUnconfirmed); found != nil {
		t.Errorf("nach der nachgeholten Bestätigung darf der Befund nicht mehr stehen: %s",
			found.Message)
	}
}

// findingByRule sucht den ersten Befund einer Regel.
func findingByRule(run *domain.CheckRun, rule string) *domain.CheckFinding {
	for i := range run.Findings {
		if run.Findings[i].Rule == rule {
			return &run.Findings[i]
		}
	}
	return nil
}

// -------------------------------------------------------------------------
// Nachweisbelege werden nur an ihrer eigenen Rechnung gelöscht
// -------------------------------------------------------------------------

// Eine falsche Kombination aus Rechnung und Nachweis nimmt nicht den Nachweis
// einer anderen Rechnung mit.
func TestSupplyEvidenceIsOnlyRemovedFromItsOwnInvoice(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.supplyEvidence(t)
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	first := env.icSupply(t, customer, "RE-2026-0001")
	second := env.icSupply(t, customer, "RE-2026-0002")

	view, err := svc.Add(ctx, SupplyEvidenceRequest{
		InvoiceID: first.ID, Kind: "cmr_frachtbrief", Issuer: "Spedition Nord",
		Independent: true, Date: "2026-04-03",
	})
	if err != nil {
		t.Fatalf("Nachweis anlegen: %v", err)
	}
	if len(view.Items) != 1 {
		t.Fatalf("%d Nachweise — erwartet einen", len(view.Items))
	}
	evidenceID := view.Items[0].ID

	if _, err := svc.Remove(ctx, second.ID, evidenceID, ""); err == nil {
		t.Fatal("der Nachweis einer fremden Rechnung darf nicht gelöscht werden")
	}
	still, err := svc.View(ctx, first.ID, accounting.TransportBySupplier)
	if err != nil {
		t.Fatalf("Nachweisstand: %v", err)
	}
	if len(still.Items) != 1 {
		t.Fatalf("der Nachweis der ersten Rechnung ist verschwunden: %+v", still.Items)
	}

	if _, err := svc.Remove(ctx, first.ID, evidenceID, ""); err != nil {
		t.Fatalf("an der eigenen Rechnung muss das Löschen gehen: %v", err)
	}
	after, err := svc.View(ctx, first.ID, accounting.TransportBySupplier)
	if err != nil {
		t.Fatalf("Nachweisstand: %v", err)
	}
	if len(after.Items) != 0 {
		t.Errorf("%d Nachweise — erwartet keinen mehr", len(after.Items))
	}
}

// -------------------------------------------------------------------------
// Die Freistellungsbescheinigung steht in der Fristenliste
// -------------------------------------------------------------------------

// Die Warnung 30 Tage vor Ablauf erreicht die Terminliste. Vorher gab es sie
// nur über eine eigene Abfrage — und eine Frist, die keine Liste zeigt, sieht
// niemand.
func TestExpiringExemptionCertificateShowsUpAmongTheDeadlines(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	soon := env.vendor(t, "Bau Bald GmbH", "DE", "")
	soon.ExemptionCertificateNumber = "FS-2026-002"
	soon.ExemptionCertificateValidUntil = time.Now().AddDate(0, 0, 20).Format("2006-01-02")
	if err := env.contacts.SaveContact(ctx, soon); err != nil {
		t.Fatalf("Lieferant: %v", err)
	}

	svc := env.deadlines(t)
	svc.SetExemptionSource(env.contacts)
	deadlines, err := svc.Deadlines(ctx, env.fiscalYear)
	if err != nil {
		t.Fatalf("Fristenliste: %v", err)
	}
	key := fmt.Sprintf("%s.%d", DeadlineKeyExemption, soon.ID)
	for _, d := range deadlines {
		if d.Key != key {
			continue
		}
		if d.DueDate != soon.ExemptionCertificateValidUntil {
			t.Errorf("Termin am %s — erwartet den Ablauftag %s",
				d.DueDate, soon.ExemptionCertificateValidUntil)
		}
		if !strings.Contains(d.Reference, "§ 48b") {
			t.Errorf("Fundstelle %q — erwartet § 48b EStG", d.Reference)
		}
		return
	}
	t.Fatalf("die ablaufende Freistellungsbescheinigung fehlt in der Fristenliste: %+v", deadlines)
}

// -------------------------------------------------------------------------
// Das Aufwandskonto der Erhaltungsaufwandsbewegung steht in einem Feld
// -------------------------------------------------------------------------

// Die Aktivierung der anschaffungsnahen Herstellungskosten bucht den Aufwand von
// dem Konto zurück, auf das er gebucht wurde. Welches das war, steht seit dieser
// Runde an der Bewegung und wird nicht mehr aus ihrem Text gelesen.
func TestMaintenanceMovementCarriesItsExpenseAccount(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.assets(t)
	vendor := env.vendor(t, "Bau GmbH", "DE", "")
	building := env.building(t, svc)

	// Ein ausdrücklich gewähltes Konto statt des Vorschlags 6450.
	if _, err := svc.BookMaintenance(ctx, MaintenanceRequest{
		AssetID: building.ID, Date: "2026-06-01", Amount: 3_200_000, Account: "6490",
		TaxTreatment: domain.TaxTreatmentDomestic, TaxRate: domain.TaxRateStandard,
		Settlement: SettlementOpen, ContactID: vendor.ID,
		Note: "Dachsanierung, Wiederherstellung des ursprünglichen Zustands",
	}); err != nil {
		t.Fatalf("Erhaltungsaufwand: %v", err)
	}

	detail, err := svc.Get(ctx, building.ID)
	if err != nil {
		t.Fatalf("Detailansicht: %v", err)
	}
	found := false
	for _, m := range detail.Asset.Movements {
		if m.Kind != domain.AssetMovementMaintenance {
			continue
		}
		found = true
		if m.ExpenseAccount != "6490" {
			t.Errorf("Aufwandskonto %q an der Bewegung — erwartet 6490", m.ExpenseAccount)
		}
	}
	if !found {
		t.Fatalf("die Bewegung zum Erhaltungsaufwand fehlt: %+v", detail.Asset.Movements)
	}

	if _, err := svc.CapitalizeNearAcquisitionCost(ctx, CapitalizeNearAcquisitionCostRequest{
		AssetID: building.ID, Date: "2026-12-31",
		Reason: "Instandsetzung übersteigt 15 % der Anschaffungskosten (§ 6 Abs. 1 Nr. 1a EStG)",
	}); err != nil {
		t.Fatalf("Aktivierung: %v", err)
	}

	entries, err := env.journalRepo.FindAll(ctx, env.fiscalYear)
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
	credit := domain.Cents(0)
	for _, l := range booking.Lines {
		if l.Side == domain.SideCredit && l.Account == "6490" {
			credit += l.Amount
		}
	}
	if credit != 3_200_000 {
		t.Errorf("HABEN 6490 %s € — der Aufwand geht von dem Konto zurück, auf das er gebucht "+
			"wurde: %s", credit, accountsOf(booking))
	}
}
