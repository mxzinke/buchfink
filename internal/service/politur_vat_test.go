package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Ein überholter Entwurf der Zusammenfassenden Meldung wird nicht bestätigt.
//
// Dieselbe Regel wie bei der Voranmeldung: bestätigt wird, was übermittelt
// wurde. Hat sich das Journal seit dem Speichern bewegt, hätte die gespeicherte
// Meldung ein Transferticket für Zahlen, die so nie beim Bundeszentralamt
// ankamen.
func TestZMConfirmationRejectsAnOutdatedDraft(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euInvoice(t, customer.ID, "2026-02-10", 500000, domain.TaxTreatmentIntraCommunitySupply)

	svc := env.zmReturns(t)
	saved, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("speichern: %v", err)
	}

	// Eine Nachbuchung im selben Zeitraum, danach die Festschreibung.
	env.euInvoice(t, customer.ID, "2026-03-05", 200000, domain.TaxTreatmentIntraCommunitySupply)
	env.commitUntil(t, "2026-03-31")

	if _, err := svc.ConfirmSubmitted(ctx, saved.ID, "2026-04-24", "TT-ZM", ""); err == nil {
		t.Fatal("ein Entwurf, der nicht mehr zum Journal passt, darf nicht bestätigt werden")
	} else if !strings.Contains(err.Error(), "stimmt nicht mehr") {
		t.Errorf("die Meldung sollte den überholten Entwurf benennen: %v", err)
	}

	// Neu gespeichert geht die Bestätigung durch.
	fresh, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("neu speichern: %v", err)
	}
	if _, err := svc.ConfirmSubmitted(ctx, fresh.ID, "2026-04-24", "TT-ZM", ""); err != nil {
		t.Fatalf("der neu gespeicherte Entwurf muss sich bestätigen lassen: %v", err)
	}
}

// Die Summe eines Meldezeitraums zählt nur, was sich melden lässt.
//
// Ein Umsatz an einen Geschäftspartner ohne USt-IdNr. kommt in keine Meldezeile
// (§ 18a Abs. 7 UStG). Zählte er in der Übersicht mit, stünde dort eine Summe,
// die in der Meldung selbst nirgends auftaucht.
func TestZMPeriodTotalIgnoresTurnoverWithoutVatID(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	withID := env.customer(t, "Client SARL", "FR", "FR12345678901")
	withoutID := env.customer(t, "Cliente Srl", "IT", "IT12345678901")

	env.euInvoice(t, withID.ID, "2026-02-10", 500000, domain.TaxTreatmentIntraCommunitySupply)
	env.euInvoice(t, withoutID.ID, "2026-02-11", 300000, domain.TaxTreatmentIntraCommunitySupply)

	// Die USt-IdNr. des zweiten Kunden fällt nachträglich weg — der Fall, den
	// die Meldung mit einem Befund quittiert und der in keine Meldezeile kommt.
	withoutID.VatID = ""
	if err := env.contacts.SaveContact(ctx, withoutID); err != nil {
		t.Fatalf("USt-IdNr. entfernen: %v", err)
	}

	periods, err := env.zmReturns(t).Periods(ctx, 2026)
	if err != nil {
		t.Fatalf("Meldezeiträume: %v", err)
	}
	q1 := zmPeriod(t, periods, "2026-Q1")
	if q1.Total != 500000 {
		t.Errorf("Summe = %s €, erwartet 5.000,00 — der Umsatz ohne USt-IdNr. steht in keiner Meldezeile",
			q1.Total)
	}
}

// Eine zweite Berichtigung schreibt den bestehenden Entwurf fort.
//
// Der Anwender ruft die Berichtigung genau dann noch einmal auf, wenn er
// zwischendurch nachgebucht hat. Zwei Entwürfe zu demselben Zeitraum ließen
// hinterher nicht mehr erkennen, welcher der übermittelte ist.
func TestCreateCorrectionIsIdempotent(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euInvoice(t, customer.ID, "2026-02-10", 500000, domain.TaxTreatmentIntraCommunitySupply)

	vat := env.vatReturns(t)
	zm := env.zmReturns(t)
	env.commitUntil(t, "2026-03-31")

	vatSaved, err := vat.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Voranmeldung speichern: %v", err)
	}
	if _, err := vat.ConfirmSubmitted(ctx, vatSaved.ID, "2026-04-10", "TT-USTVA", ""); err != nil {
		t.Fatalf("Voranmeldung bestätigen: %v", err)
	}
	zmSaved, err := zm.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Meldung speichern: %v", err)
	}
	if _, err := zm.ConfirmSubmitted(ctx, zmSaved.ID, "2026-04-24", "TT-ZM", ""); err != nil {
		t.Fatalf("Meldung bestätigen: %v", err)
	}

	firstVat, err := vat.CreateCorrection(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Berichtigung der Voranmeldung: %v", err)
	}
	firstZM, err := zm.CreateCorrection(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Berichtigung der Meldung: %v", err)
	}

	// Zwischendurch wird nachgebucht, danach wird die Berichtigung ein zweites
	// Mal angestoßen.
	env.euInvoice(t, customer.ID, "2026-03-20", 100000, domain.TaxTreatmentIntraCommunitySupply)

	secondVat, err := vat.CreateCorrection(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("zweite Berichtigung der Voranmeldung: %v", err)
	}
	if secondVat.ID != firstVat.ID {
		t.Errorf("die zweite Berichtigung legt die Anmeldung %d neben %d an", secondVat.ID, firstVat.ID)
	}
	secondZM, err := zm.CreateCorrection(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("zweite Berichtigung der Meldung: %v", err)
	}
	if secondZM.ID != firstZM.ID {
		t.Errorf("die zweite Berichtigung legt die Meldung %d neben %d an", secondZM.ID, firstZM.ID)
	}
	if secondZM.TotalSupplies != 600000 {
		t.Errorf("die fortgeschriebene Meldung führt %s €, erwartet 6.000,00", secondZM.TotalSupplies)
	}

	// Und je Zeitraum steht danach genau ein Entwurf.
	vatList, err := vat.List(ctx, 2026)
	if err != nil {
		t.Fatalf("Voranmeldungen: %v", err)
	}
	drafts := 0
	for _, r := range vatList {
		if r.PeriodKey == "2026-Q1" && r.Status == domain.VatReturnDraft {
			drafts++
		}
	}
	if drafts != 1 {
		t.Errorf("zum Zeitraum 2026-Q1 stehen %d Entwürfe der Voranmeldung, erwartet 1", drafts)
	}
	zmList, err := zm.List(ctx, 2026)
	if err != nil {
		t.Fatalf("Meldungen: %v", err)
	}
	drafts = 0
	for _, r := range zmList {
		if r.PeriodKey == "2026-Q1" && r.Status == domain.VatReturnDraft {
			drafts++
		}
	}
	if drafts != 1 {
		t.Errorf("zum Zeitraum 2026-Q1 stehen %d Entwürfe der Meldung, erwartet 1", drafts)
	}
}

// Das gewährte Skonto mindert die Bemessungsgrundlage und die Steuer der
// Voranmeldung.
//
// § 17 Abs. 1 UStG verlangt die Berichtigung, sobald sich das Entgelt geändert
// hat — für den Zeitraum, in dem die Änderung eingetreten ist (Satz 8). Die
// Korrektur hat im Journal einen eigenen Schlüssel; im Vordruck läuft sie in
// dieselbe Kennziffer 81 wie der Umsatz, den sie mindert.
func TestSkontoReducesTheStandardRateFigures(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.payments(t)
	customer := env.customer(t, "Kunde", "DE", "")

	inv := &domain.Invoice{
		ContactID: customer.ID, Date: "2026-03-01",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-01",
		TaxTreatment: domain.TaxTreatmentDomestic,
		Items: []domain.InvoiceItem{{
			Description: "Leistung", QuantityMilli: 1000, UnitPrice: 100000, TaxRate: domain.TaxRateStandard,
		}},
	}
	if err := env.invoices(t).Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung: %v", err)
	}

	// 2 % Skonto auf 1.190,00 € brutto = 23,80 € (20,00 netto + 3,80 Steuer).
	if _, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2026-03-05",
		Allocations: []AllocationRequest{{
			OpenItemEntryID:  *inv.JournalEntryID,
			SettledAmount:    119000,
			DifferenceKind:   domain.DifferenceSkonto,
			DifferenceAmount: 2380,
		}},
	}); err != nil {
		t.Fatalf("Zahlungseingang mit Skonto: %v", err)
	}

	ret, err := env.vatReturns(t).Draft(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Voranmeldung: %v", err)
	}
	if got := ret.Base("81"); got != 98000 {
		t.Errorf("Kz 81 Bemessungsgrundlage = %s €, erwartet 980,00 (1.000,00 abzüglich 20,00 Skonto)", got)
	}
	if got := ret.Tax("81"); got != 18620 {
		t.Errorf("Kz 81 Steuer = %s €, erwartet 186,20 (190,00 abzüglich 3,80 Skonto)", got)
	}
}
