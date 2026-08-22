package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func entertainmentRecord() *domain.EntertainmentDetail {
	return &domain.EntertainmentDetail{
		Place:        "Restaurant Adler, München",
		Day:          "2026-03-10",
		Participants: "M. Zinke (Buchfink), A. Kunde (Kunde GmbH)",
		Occasion:     "Projektbesprechung Rollout Q2",
	}
}

// Bewirtung: 70 % abziehbar auf 6640, 30 % nicht abzugsfähig auf 6644 — und die
// Vorsteuer auf die volle Bemessungsgrundlage, weil § 15 Abs. 1a Satz 2 UStG
// Bewirtungsaufwendungen vom Vorsteuerausschluss ausnimmt.
func TestEntertainmentSplitsExpenseButNotInputTax(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Restaurant Adler", "DE", "")

	req := env.receipt(t, vendor.ID, "bewirtung", 20000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Entertainment = entertainmentRecord()

	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Bewirtungsbeleg buchen: %v", err)
	}

	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, "6640", 14000}, // 70 % abziehbar
		{domain.SideDebit, "6644", 6000},  // 30 % nicht abzugsfähig
		{domain.SideDebit, "1406", 3800},  // Vorsteuer auf die vollen 200,00
		{domain.SideCredit, vendor.LedgerAccount, 23800},
	})
}

// Die beiden Aufwandszeilen ergeben zusammen exakt den Nettobetrag. Der Rest wird
// als Differenz gebucht und nicht ein zweites Mal gerundet.
func TestEntertainmentSplitAddsUpExactly(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Restaurant", "DE", "")

	// 33,33 € netto: 70 % sind 23,331 — kaufmännisch 23,33, Rest 10,00.
	for _, net := range []domain.Cents{3333, 1, 7, 999999, 10000} {
		req := env.receipt(t, vendor.ID, "bewirtung", net, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
		req.Entertainment = entertainmentRecord()

		entry, err := env.posting.PostIncomingReceipt(ctx, req)
		if err != nil {
			t.Fatalf("Netto %s: %v", net, err)
		}

		var sum domain.Cents
		for _, l := range entry.Lines {
			if l.Account == "6640" || l.Account == "6644" {
				sum += l.Amount
			}
		}
		if sum != net {
			t.Errorf("Netto %s: 6640 + 6644 = %s, erwartet %s", net, sum, net)
		}
		if !entry.IsBalanced() {
			t.Errorf("Netto %s: Buchung ist nicht ausgeglichen", net)
		}
	}
}

// Ohne die Aufzeichnung nach § 4 Abs. 5 Satz 1 Nr. 2 EStG ist der Abzug auch für
// die abziehbaren 70 % verloren — also wird nicht gebucht.
func TestEntertainmentWithoutRecordIsRefused(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Restaurant", "DE", "")

	req := env.receipt(t, vendor.ID, "bewirtung", 20000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	_, err := env.posting.PostIncomingReceipt(ctx, req)
	if err == nil {
		t.Fatal("eine Bewirtung ohne Aufzeichnung muss abgelehnt werden")
	}
	for _, want := range []string{"Ort", "Teilnehmer", "Anlass"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("die Meldung soll %q nennen, lautet aber: %v", want, err)
		}
	}

	// Eine halbe Aufzeichnung ist keine.
	req = env.receipt(t, vendor.ID, "bewirtung", 20000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Entertainment = &domain.EntertainmentDetail{Place: "Adler", Day: "2026-03-10"}
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err == nil {
		t.Error("eine unvollständige Aufzeichnung muss abgelehnt werden")
	}
}

// Die Aufzeichnung fällt unter die Hash-Chain: eine nachträglich geänderte
// Teilnehmerliste bricht sie.
func TestChainCoversTheEntertainmentRecord(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Restaurant", "DE", "")

	req := env.receipt(t, vendor.ID, "bewirtung", 20000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Entertainment = entertainmentRecord()
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Bewirtungsbeleg buchen: %v", err)
	}

	if err := env.db.Model(&domain.EntertainmentDetail{}).Where("entry_id = ?", entry.ID).
		Update("participants", "nur ich").Error; err != nil {
		t.Fatalf("Testmanipulation fehlgeschlagen: %v", err)
	}

	result, err := env.journal.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatalf("Integritätsprüfung: %v", err)
	}
	if result.IsValid {
		t.Error("eine geänderte Teilnehmerliste muss die Hash-Chain brechen")
	}
}

// Die Auswertung sieht dieselbe Vorsteuer wie bei einem gewöhnlichen Beleg über
// denselben Nettobetrag — die Zeile auf 6644 trägt keinen Steuerschlüssel.
func TestEntertainmentDoesNotChangeTheInputTaxFigure(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Restaurant", "DE", "")

	req := env.receipt(t, vendor.ID, "bewirtung", 20000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Entertainment = entertainmentRecord()
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
		t.Fatalf("Bewirtungsbeleg: %v", err)
	}

	plain := newTestEnv(t)
	plainVendor := plain.vendor(t, "Bürohandel", "DE", "")
	if _, err := plain.posting.PostIncomingReceipt(ctx,
		plain.receipt(t, plainVendor.ID, "buerobedarf", 20000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)); err != nil {
		t.Fatalf("Vergleichsbeleg: %v", err)
	}

	withEntertainment, err := env.vat().Summary(ctx, "", "")
	if err != nil {
		t.Fatalf("USt-Auswertung: %v", err)
	}
	comparison, err := plain.vat().Summary(ctx, "", "")
	if err != nil {
		t.Fatalf("USt-Auswertung Vergleich: %v", err)
	}

	if withEntertainment.InputTax != comparison.InputTax {
		t.Errorf("Vorsteuer bei Bewirtung = %s, bei gewöhnlichem Beleg = %s — sie muss gleich sein",
			withEntertainment.InputTax, comparison.InputTax)
	}
	if withEntertainment.InputTax != 3800 {
		t.Errorf("Vorsteuer = %s, erwartet 38,00 auf die vollen 200,00", withEntertainment.InputTax)
	}
}

// Der Storno einer Bewirtung nimmt die Aufzeichnung mit: er korrigiert die
// Buchung, nicht das Essen.
func TestReversalKeepsTheEntertainmentRecord(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Restaurant", "DE", "")

	req := env.receipt(t, vendor.ID, "bewirtung", 20000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Entertainment = entertainmentRecord()
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Bewirtungsbeleg buchen: %v", err)
	}

	reversal, err := env.journal.Reverse(ctx, entry.ID, "doppelt erfasst")
	if err != nil {
		t.Fatalf("Storno: %v", err)
	}
	if reversal.Entertainment == nil {
		t.Fatal("die Generalumkehr muss die Aufzeichnung übernehmen")
	}
	if reversal.Entertainment.Participants != req.Entertainment.Participants {
		t.Errorf("Teilnehmer der Generalumkehr = %q", reversal.Entertainment.Participants)
	}

	// Und die Verkehrszahlen beider Aufwandskonten stehen wieder auf null.
	turnovers, err := env.journalRepo.AccountTurnovers(ctx, env.fiscalYear)
	if err != nil {
		t.Fatalf("Verkehrszahlen: %v", err)
	}
	for _, account := range []string{"6640", "6644"} {
		if got := turnovers[account].Debit; got != 0 {
			t.Errorf("Konto %s hat nach dem Storno noch %s im Soll", account, got)
		}
	}
}

// Die Integritätsprüfung liest die Aufzeichnung mit. Täte sie es nicht, würde
// jede Bewirtungsbuchung als gebrochen gemeldet — die Kanonisierung deckt sie ab.
func TestIntegrityCheckReadsTheEntertainmentRecord(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Restaurant", "DE", "")

	req := env.receipt(t, vendor.ID, "bewirtung", 20000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Entertainment = entertainmentRecord()
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
		t.Fatalf("Bewirtungsbeleg buchen: %v", err)
	}

	result, err := env.journal.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatalf("Integritätsprüfung: %v", err)
	}
	if !result.IsValid {
		t.Fatalf("eine unveränderte Bewirtungsbuchung muss die Prüfung bestehen: %s", result.Message)
	}
}
