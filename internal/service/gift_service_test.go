package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die Freigrenze für Geschenke (§ 4 Abs. 5 Satz 1 Nr. 1 EStG) und der Bericht
// über die nicht abziehbaren Betriebsausgaben.

func (e *testEnv) gifts(t *testing.T) *GiftService {
	t.Helper()
	svc := NewGiftService(
		e.journalRepo, e.journal,
		repository.NewSettingsRepository(e.db),
		repository.NewAuditRepository(e.db),
		e.fiscalYear,
	)
	e.posting.SetGiftRegister(svc)
	return svc
}

// giftReceipt baut einen Beleg über ein Geschenk an einen Empfänger.
func (e *testEnv) giftReceipt(
	t *testing.T, vendorID uint, recipient string, net domain.Cents,
) ReceiptRequest {
	t.Helper()
	req := e.receipt(t, vendorID, "geschenke", net, domain.TaxRateStandard,
		domain.TaxTreatmentDomestic)
	req.Positions[0].Gift = &GiftInput{Name: recipient, Occasion: "Jubiläum"}
	return req
}

// Zwei Geschenke an denselben Empfänger: 40 € bleiben unter der Freigrenze von
// 50 €, mit weiteren 20 € ist sie gerissen. Das zweite Geschenk geht auf das
// nicht abziehbare Konto und ohne Vorsteuerabzug, und der Anwender wird gewarnt.
func TestGiftsOverTheFreeLimitAreBookedWithoutDeduction(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.gifts(t)
	vendor := env.vendor(t, "Präsente GmbH", "DE", "")

	first, err := env.posting.PostIncomingReceipt(ctx,
		env.giftReceipt(t, vendor.ID, "Dr. Meyer", 4_000))
	if err != nil {
		t.Fatalf("erstes Geschenk: %v", err)
	}
	if !hasDebitOn(first, "6610") {
		t.Errorf("40 € bleiben unter der Freigrenze und gehören auf 6610: %s", accountsOf(first))
	}
	if debitOn(first, domain.AccountVorsteuer19) != 760 {
		t.Errorf("Vorsteuer %s € — bei einem abziehbaren Geschenk bleibt sie abziehbar",
			debitOn(first, domain.AccountVorsteuer19))
	}

	// Das zweite Geschenk reißt die Grenze. Die Vorschau warnt vorher.
	second := env.giftReceipt(t, vendor.ID, "Dr. Meyer", 2_000)
	preview, err := env.posting.PreviewIncomingReceipt(ctx, second)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	warned := false
	for _, w := range preview.Warnings {
		if w.Code == warningGiftLimitExceeded {
			warned = true
			if !strings.Contains(w.Detail, "Dr. Meyer") {
				t.Errorf("die Warnung muss den Empfänger nennen: %s", w.Detail)
			}
		}
	}
	if !warned {
		t.Fatalf("vor dem Überschreiten der Freigrenze muss gewarnt werden: %+v", preview.Warnings)
	}

	entry, err := env.posting.PostIncomingReceipt(ctx, second)
	if err != nil {
		t.Fatalf("zweites Geschenk: %v", err)
	}
	if !hasDebitOn(entry, "6620") {
		t.Errorf("über der Freigrenze gehört das Geschenk auf 6620: %s", accountsOf(entry))
	}
	if debitOn(entry, domain.AccountVorsteuer19) != 0 {
		t.Errorf("Vorsteuer %s € — mit dem Abzug entfällt sie nach § 15 Abs. 1a UStG",
			debitOn(entry, domain.AccountVorsteuer19))
	}
	// Der Bruttobetrag steht im Aufwand: die nicht abziehbare Vorsteuer gehört
	// dorthin (§ 9b Abs. 1 EStG).
	if got := debitOn(entry, "6620"); got != 2_380 {
		t.Errorf("Aufwand %s € — erwartet 23,80 € (20,00 netto plus 3,80 Vorsteuer)", got)
	}

	// Der Bericht nennt das erste Geschenk als umzubuchen.
	report, err := svc.NonDeductibleReport(ctx, 2026)
	if err != nil {
		t.Fatalf("Bericht: %v", err)
	}
	if len(report.Recipients) != 1 {
		t.Fatalf("%d Empfänger im Bericht — erwartet einen", len(report.Recipients))
	}
	row := report.Recipients[0]
	if !row.OverLimit {
		t.Errorf("60 € übersteigen die Freigrenze von %s €", report.Limit)
	}
	if len(row.ToRebook) != 1 || row.ToRebook[0].EntryNumber != first.EntryNumber {
		t.Errorf("umzubuchen sind %+v — erwartet genau die erste Buchung %s",
			row.ToRebook, first.EntryNumber)
	}

	// Die Umbuchung erzeugt Generalumkehr und Neubuchung.
	rebooking, err := svc.RebookGiftsForRecipient(ctx, RebookGiftsRequest{
		FiscalYear: 2026, RecipientKey: row.RecipientKey,
		Reason: "Freigrenze mit dem Geschenk vom 10.03.2026 überschritten",
	})
	if err != nil {
		t.Fatalf("Umbuchung: %v", err)
	}
	if len(rebooking.Reversals) != 1 || len(rebooking.Rebookings) != 1 {
		t.Fatalf("%d Stornos und %d Neubuchungen — erwartet je eine",
			len(rebooking.Reversals), len(rebooking.Rebookings))
	}

	// Danach steht nichts mehr zum Umbuchen an, und der Aufwand liegt auf 6620.
	after, err := svc.NonDeductibleReport(ctx, 2026)
	if err != nil {
		t.Fatalf("Bericht nach der Umbuchung: %v", err)
	}
	if len(after.Recipients[0].ToRebook) != 0 {
		t.Errorf("nach der Umbuchung ist nichts mehr offen: %+v", after.Recipients[0].ToRebook)
	}
	gifts := categoryRow(t, after, "gifts")
	if gifts.Deductible != 0 {
		t.Errorf("auf 6610 stehen noch %s € — nach der Umbuchung gehört dort nichts mehr hin",
			gifts.Deductible)
	}
	if gifts.NonDeductible != 2_380+4_760 {
		t.Errorf("auf 6620 stehen %s € — erwartet beide Geschenke brutto", gifts.NonDeductible)
	}
}

// Ein Geschenk ohne Empfänger ist keine Aufzeichnung im Sinne des § 4 Abs. 7
// EStG — und ohne Empfänger ließe sich die Freigrenze nicht führen.
func TestGiftNeedsARecipient(t *testing.T) {
	env := newTestEnv(t)
	env.gifts(t)
	vendor := env.vendor(t, "Präsente GmbH", "DE", "")

	req := env.receipt(t, vendor.ID, "geschenke", 3_000,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	if _, err := env.posting.PostIncomingReceipt(context.Background(), req); err == nil {
		t.Fatal("ein Geschenk ohne Empfänger darf nicht gebucht werden")
	} else if !strings.Contains(err.Error(), "§ 4 Abs. 7") {
		t.Errorf("die Meldung muss die Aufzeichnungspflicht nennen: %v", err)
	}
}

// Der freie Kontoweg führt nicht an der Gruppe vorbei: wer 6610 von Hand wählt,
// bucht ohne Empfänger und an der Freigrenze vorbei.
func TestGiftAccountsAreNotFreelyChoosable(t *testing.T) {
	env := newTestEnv(t)
	env.gifts(t)
	vendor := env.vendor(t, "Präsente GmbH", "DE", "")

	req := env.receipt(t, vendor.ID, "", 3_000,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Positions[0].PostingGroup = ""
	req.Positions[0].Account = "6610"

	if _, err := env.posting.PostIncomingReceipt(context.Background(), req); err == nil {
		t.Fatal("das Konto 6610 darf nicht frei wählbar sein")
	} else if !strings.Contains(err.Error(), "§ 4 Abs. 7") {
		t.Errorf("die Meldung muss die Aufzeichnungspflicht nennen: %v", err)
	}

	// Ein gewöhnliches Aufwandskonto bleibt frei wählbar.
	req.Positions[0].Account = "6815"
	if _, err := env.posting.PostIncomingReceipt(context.Background(), req); err != nil {
		t.Fatalf("der freie Kontoweg muss für gewöhnliche Konten offen bleiben: %v", err)
	}
}

// Aufwendungen nach § 4 Abs. 5 Satz 1 Nr. 3 und 4 EStG sind in keiner Höhe
// abziehbar, und § 15 Abs. 1a UStG nimmt ihnen auch den Vorsteuerabzug.
func TestRepresentationExpensesCarryNoInputTax(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.gifts(t)
	vendor := env.vendor(t, "Yachthafen GmbH", "DE", "")

	req := env.receipt(t, vendor.ID, "repraesentation", 100_000,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	if debitOn(entry, domain.AccountVorsteuer19) != 0 {
		t.Error("für Aufwendungen nach § 4 Abs. 5 Satz 1 Nr. 4 EStG gibt es keinen Vorsteuerabzug")
	}
	if got := debitOn(entry, "6645"); got != 119_000 {
		t.Errorf("Aufwand %s € — erwartet den Bruttobetrag von 1.190,00 €", got)
	}
}

// -------------------------------------------------------------------------
// Hilfsfunktionen
// -------------------------------------------------------------------------

func debitOn(entry *domain.JournalEntry, account string) domain.Cents {
	var total domain.Cents
	for _, l := range entry.Lines {
		if l.Side == domain.SideDebit && l.Account == account {
			total += l.Amount
		}
	}
	return total
}

func hasDebitOn(entry *domain.JournalEntry, account string) bool {
	return debitOn(entry, account) > 0
}

func accountsOf(entry *domain.JournalEntry) string {
	parts := make([]string, 0, len(entry.Lines))
	for _, l := range entry.Lines {
		parts = append(parts, string(l.Side)+" "+l.Account)
	}
	return strings.Join(parts, ", ")
}

func categoryRow(t *testing.T, report *NonDeductibleReport, key string) NonDeductibleCategoryRow {
	t.Helper()
	for _, c := range report.Categories {
		if c.Key == key {
			return c
		}
	}
	t.Fatalf("die Kategorie %q fehlt im Bericht", key)
	return NonDeductibleCategoryRow{}
}
