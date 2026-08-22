package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Der Beleg-Hash steht in der Buchung, nicht der Pfad. Wird eine Belegdatei
// nachträglich ausgetauscht, bricht die Kette.
func TestChainCoversTheReceiptHash(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant", "DE", "")

	entry, err := env.posting.PostIncomingReceipt(ctx,
		env.receipt(t, vendor.ID, "buerobedarf", 5000, domain.TaxRateStandard, domain.TaxTreatmentDomestic))
	if err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}
	if entry.ReceiptID == nil || entry.ReceiptHash == "" {
		t.Fatalf("die Buchung muss auf den Beleg und seinen Hash zeigen, hat aber %+v", entry)
	}

	receipt, err := env.receipts.Get(ctx, *entry.ReceiptID)
	if err != nil {
		t.Fatalf("Beleg laden: %v", err)
	}
	if entry.ReceiptHash != receipt.ReceiptHash {
		t.Errorf("Beleg-Hash der Buchung = %q, am Beleg %q", entry.ReceiptHash, receipt.ReceiptHash)
	}
	if entry.DocumentNumber != receipt.ReceiptNumber {
		t.Errorf("das Belegfeld muss die Belegnummer tragen: %q vs %q", entry.DocumentNumber, receipt.ReceiptNumber)
	}

	// Eine ausgetauschte Belegdatei ändert den Beleg-Hash — und damit die Kette.
	if err := env.db.Model(&domain.JournalEntry{}).Where("id = ?", entry.ID).
		Update("receipt_hash", strings.Repeat("f", 64)).Error; err != nil {
		t.Fatalf("Testmanipulation fehlgeschlagen: %v", err)
	}
	result, err := env.journal.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatalf("Integritätsprüfung: %v", err)
	}
	if result.IsValid {
		t.Error("ein veränderter Belegverweis muss die Hash-Chain brechen")
	}
}

// Der Steuerfall steht an der Buchung und ist von der Kette gedeckt. Auf der
// Eingangsseite ist er das Einzige, was steuerfrei von nullbesteuert trennt.
func TestChainCoversTheTaxTreatment(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Solarteur GmbH", "DE", "")

	req := env.receipt(t, vendor.ID, "wareneingang", 500000, domain.TaxRateNone, domain.TaxTreatmentZeroRated)
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}
	if entry.TaxTreatment != domain.TaxTreatmentZeroRated {
		t.Fatalf("Steuerfall an der Buchung = %q, erwartet %q", entry.TaxTreatment, domain.TaxTreatmentZeroRated)
	}

	if err := env.db.Model(&domain.JournalEntry{}).Where("id = ?", entry.ID).
		Update("tax_treatment", string(domain.TaxTreatmentExempt)).Error; err != nil {
		t.Fatalf("Testmanipulation fehlgeschlagen: %v", err)
	}
	result, _ := env.journal.VerifyIntegrity(ctx)
	if result.IsValid {
		t.Error("ein umgeschriebener Steuerfall muss die Hash-Chain brechen")
	}
}

// Das Versiegeln liegt hinter dem Journalschreibvorgang: erst die Buchung, dann
// das Siegel.
func TestBookingSealsTheReceipt(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant", "DE", "")

	req := env.receipt(t, vendor.ID, "buerobedarf", 5000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)

	before, err := env.receipts.Get(ctx, req.ReceiptID)
	if err != nil {
		t.Fatalf("Beleg laden: %v", err)
	}
	if before.Status != domain.ReceiptStatusFiled {
		t.Fatalf("vor dem Buchen muss der Beleg abgelegt sein, ist aber %q", before.Status)
	}

	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}

	after, err := env.receipts.Get(ctx, req.ReceiptID)
	if err != nil {
		t.Fatalf("Beleg laden: %v", err)
	}
	if after.Status != domain.ReceiptStatusSealed {
		t.Errorf("nach dem Buchen muss der Beleg versiegelt sein, ist aber %q", after.Status)
	}
	if after.JournalEntryID == nil || *after.JournalEntryID != entry.ID {
		t.Errorf("der Beleg muss auf die Buchung zeigen, hat aber %v", after.JournalEntryID)
	}

	// Und ein zweites Mal buchen geht nicht.
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err == nil {
		t.Error("ein bereits gebuchter Beleg darf nicht erneut gebucht werden")
	}
}

// Scheitert die Buchung, bleibt der Beleg offen und kann korrigiert erneut
// gebucht werden. Die andere Reihenfolge hinterließe einen unveränderlichen
// Beleg ohne Buchung.
func TestFailedBookingLeavesTheReceiptOpen(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant", "DE", "")

	req := env.receipt(t, vendor.ID, "buerobedarf", 5000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Settlement = SettlementPaid // ohne Zahlungsmittelkonto: die Buchung scheitert

	if _, err := env.posting.PostIncomingReceipt(ctx, req); err == nil {
		t.Fatal("eine Sofortzahlung ohne Zahlungsmittel muss abgelehnt werden")
	}

	receipt, err := env.receipts.Get(ctx, req.ReceiptID)
	if err != nil {
		t.Fatalf("Beleg laden: %v", err)
	}
	if receipt.Status != domain.ReceiptStatusFiled {
		t.Fatalf("nach der gescheiterten Buchung muss der Beleg offen bleiben, ist aber %q", receipt.Status)
	}

	// Korrigiert lässt er sich buchen.
	req.PaymentAccount = domain.AccountKasse
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
		t.Fatalf("der korrigierte Beleg muss buchbar sein: %v", err)
	}
}

// Bricht der Prozess zwischen Buchung und Siegel ab, repariert das nächste Lesen
// den Zustand — sonst nähme ein gebuchter Beleg noch Dateien an.
func TestUnsealedButBookedReceiptIsRepairedOnRead(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant", "DE", "")

	req := env.receipt(t, vendor.ID, "buerobedarf", 5000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}

	// Den Abbruch nachstellen: die Buchung steht, das Siegel fehlt.
	if err := env.db.Model(&domain.Receipt{}).Where("id = ?", req.ReceiptID).
		Updates(map[string]any{"status": domain.ReceiptStatusFiled, "journal_entry_id": nil}).Error; err != nil {
		t.Fatalf("Testmanipulation fehlgeschlagen: %v", err)
	}

	repaired, err := env.receipts.Get(ctx, req.ReceiptID)
	if err != nil {
		t.Fatalf("Beleg laden: %v", err)
	}
	if repaired.Status != domain.ReceiptStatusSealed {
		t.Fatalf("der Beleg muss beim Lesen nachversiegelt werden, ist aber %q", repaired.Status)
	}
	if repaired.JournalEntryID == nil || *repaired.JournalEntryID != entry.ID {
		t.Errorf("das nachgeholte Siegel muss auf die Buchung zeigen, zeigt aber auf %v", repaired.JournalEntryID)
	}
}

// Die Generalumkehr zeigt auf denselben Beleg: sie korrigiert die Buchung, nicht
// das Dokument.
func TestReversalInheritsTheReceiptReference(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant", "DE", "")

	entry, err := env.posting.PostIncomingReceipt(ctx,
		env.receipt(t, vendor.ID, "buerobedarf", 5000, domain.TaxRateStandard, domain.TaxTreatmentDomestic))
	if err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}

	reversal, err := env.journal.Reverse(ctx, entry.ID, "doppelt erfasst")
	if err != nil {
		t.Fatalf("Storno: %v", err)
	}
	if reversal.ReceiptID == nil || *reversal.ReceiptID != *entry.ReceiptID {
		t.Errorf("die Generalumkehr muss auf denselben Beleg zeigen")
	}
	if reversal.ReceiptHash != entry.ReceiptHash {
		t.Errorf("die Generalumkehr muss denselben Beleg-Hash tragen")
	}
	if reversal.TaxTreatment != entry.TaxTreatment {
		t.Errorf("die Generalumkehr muss denselben Steuerfall tragen")
	}
	if reversal.DocumentNumber != entry.DocumentNumber {
		t.Errorf("die Generalumkehr muss dieselbe Belegnummer tragen")
	}
}

// Ein Beleg der falschen Richtung passt nicht zu einem Eingangsbeleg.
func TestBookingRejectsAReceiptOfTheWrongDirection(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant", "DE", "")

	outgoing, err := env.receipts.File(ctx, FileReceiptRequest{
		Direction:     domain.DirectionOutgoing,
		ReceiptNumber: "RE-2026-0001",
		Files:         []NewFile{{Role: domain.ReceiptRoleOriginal, FileName: "r.pdf", Content: []byte(minimalPDF)}},
	})
	if err != nil {
		t.Fatalf("Ausgangsbeleg ablegen: %v", err)
	}

	req := env.receipt(t, vendor.ID, "buerobedarf", 5000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.ReceiptID = outgoing.ID
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err == nil {
		t.Error("ein Ausgangsbeleg darf nicht als Eingangsbeleg gebucht werden")
	}
}

// Ein Beleg ohne ansehbare Darstellung ist nicht buchbar — die zweite der beiden
// Prüfungen.
func TestBookingRefusesAReceiptThatCannotBeDisplayed(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant", "DE", "")

	filed, err := env.receipts.File(ctx, FileReceiptRequest{
		Direction: domain.DirectionIncoming,
		Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, FileName: "xrechnung.xml", Content: []byte(xRechnungXML)},
			{Role: domain.ReceiptRoleStructured, FileName: "xrechnung.xml", Content: []byte(xRechnungXML)},
		},
	})
	if err != nil {
		t.Fatalf("XRechnung ablegen: %v", err)
	}

	req := env.receipt(t, vendor.ID, "software", 100000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.ReceiptID = filed.ID
	_, err = env.posting.PostIncomingReceipt(ctx, req)
	if err == nil {
		t.Fatal("ein Beleg ohne Darstellung darf nicht gebucht werden")
	}
	if !strings.Contains(err.Error(), "Darstellung") {
		t.Errorf("die Meldung soll die fehlende Darstellung nennen, lautet aber: %v", err)
	}
}

// Vorschau und Buchung laufen durch denselben Zeilenaufbau. Weichen sie ab, ist
// die Vorschau eine Lüge.
func TestPreviewMatchesTheBooking(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// wantGross ist, was tatsächlich an den Lieferanten geht. Bei Reverse Charge
	// und innergemeinschaftlichem Erwerb ist das der Nettobetrag: die Steuer
	// schuldet man selbst und zieht sie im selben Atemzug als Vorsteuer ab.
	cases := []struct {
		name      string
		country   string
		vatID     string
		group     string
		net       domain.Cents
		rate      domain.TaxRate
		treatment domain.TaxTreatment
		wantGross domain.Cents
	}{
		{"Inland 19 %", "DE", "", "fremdleistungen", 100000, domain.TaxRateStandard, domain.TaxTreatmentDomestic, 119000},
		{"Inland 7 %", "DE", "", "fremdleistungen", 100000, domain.TaxRateReduced, domain.TaxTreatmentDomestic, 107000},
		{"§ 13b", "IE", "IE6388047V", "software", 100000, domain.TaxRateStandard, domain.TaxTreatmentReverseCharge, 100000},
		{"i. g. Erwerb", "NL", "NL123456789B01", "wareneingang", 200000, domain.TaxRateStandard, domain.TaxTreatmentIntraCommunityAcquisition, 200000},
		{"Nullsteuersatz", "DE", "", "wareneingang", 500000, domain.TaxRateNone, domain.TaxTreatmentZeroRated, 500000},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vendor := env.vendor(t, "Lieferant "+tc.name, tc.country, tc.vatID)
			req := env.receipt(t, vendor.ID, tc.group, tc.net, tc.rate, tc.treatment)

			preview, err := env.posting.PreviewIncomingReceipt(ctx, req)
			if err != nil {
				t.Fatalf("Vorschau: %v", err)
			}
			if !preview.Balanced {
				t.Error("die Vorschau muss ausgeglichen sein")
			}

			entry, err := env.posting.PostIncomingReceipt(ctx, req)
			if err != nil {
				t.Fatalf("Buchung: %v", err)
			}

			if len(preview.Lines) != len(entry.Lines) {
				t.Fatalf("Vorschau hat %d Zeilen, die Buchung %d", len(preview.Lines), len(entry.Lines))
			}
			for i := range preview.Lines {
				p, b := preview.Lines[i], entry.Lines[i]
				if p.Side != b.Side || p.Account != b.Account || p.Amount != b.Amount ||
					p.TaxKey != b.TaxKey || p.TaxBase != b.TaxBase {
					t.Errorf("Zeile %d weicht ab:\nVorschau %s %s %s (%s)\nBuchung  %s %s %s (%s)",
						i+1, p.Side, p.Account, p.Amount, p.TaxKey, b.Side, b.Account, b.Amount, b.TaxKey)
				}
			}
			// Der Bruttobetrag der Vorschau ist die Gegenzeile — was gezahlt
			// wird —, nicht die Soll-Summe der Buchung. Bei § 13b fallen die
			// beiden auseinander, und die Gegenzeile ist die Zahl, die auf der
			// Rechnung steht.
			settlement := entry.Lines[len(entry.Lines)-1]
			if preview.Gross != settlement.Amount {
				t.Errorf("Bruttobetrag Vorschau %s, Gegenzeile der Buchung %s", preview.Gross, settlement.Amount)
			}
			if preview.Gross != tc.wantGross {
				t.Errorf("Bruttobetrag = %s, erwartet %s", preview.Gross, tc.wantGross)
			}
			if preview.Net != tc.net {
				t.Errorf("Nettobetrag der Vorschau = %s, erwartet %s", preview.Net, tc.net)
			}
			if preview.Tax != tc.wantGross-tc.net {
				t.Errorf("Steuerbetrag = %s, erwartet %s", preview.Tax, tc.wantGross-tc.net)
			}
		})
	}
}

// Die Vorschau schreibt nichts.
func TestPreviewWritesNothing(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant", "DE", "")

	req := env.receipt(t, vendor.ID, "buerobedarf", 5000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	for i := 0; i < 3; i++ {
		if _, err := env.posting.PreviewIncomingReceipt(ctx, req); err != nil {
			t.Fatalf("Vorschau %d: %v", i+1, err)
		}
	}

	count, err := env.journalRepo.Count(ctx, env.fiscalYear)
	if err != nil {
		t.Fatalf("Buchungen zählen: %v", err)
	}
	if count != 0 {
		t.Errorf("nach drei Vorschauen stehen %d Buchungen im Journal", count)
	}
	receipt, err := env.receipts.Get(ctx, req.ReceiptID)
	if err != nil {
		t.Fatalf("Beleg laden: %v", err)
	}
	if receipt.Status != domain.ReceiptStatusFiled {
		t.Errorf("eine Vorschau darf den Beleg nicht versiegeln, Status ist %q", receipt.Status)
	}
}

// Bei Istversteuerung entsteht die Steuer erst mit der Vereinnahmung. Der ganze
// Flow setzt Sollversteuerung voraus, also wird gesagt statt still falsch gebucht.
func TestCashAccountingIsRefused(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	settings := repository.NewSettingsRepository(env.db)
	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Einstellungen: %v", err)
	}
	cfg.TaxationType = "IST"
	if err := settings.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatalf("Einstellungen speichern: %v", err)
	}

	_, err = env.journal.Post(ctx, simpleEntry("6815", "1800", 10000))
	if err == nil {
		t.Fatal("bei Istversteuerung darf nicht gebucht werden")
	}
	for _, want := range []string{"Sollversteuerung", "§ 16 Abs. 1", "§ 13 Abs. 1"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("die Meldung soll %q nennen, lautet aber: %v", want, err)
		}
	}

	cfg.TaxationType = "SOLL"
	if err := settings.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatalf("Einstellungen speichern: %v", err)
	}
	if _, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 10000)); err != nil {
		t.Fatalf("mit Sollversteuerung muss gebucht werden können: %v", err)
	}
}

// Keine Buchung ohne Beleg. Wo kein Dokument des Lieferanten vorliegt, gehört ein
// Eigenbeleg abgelegt — ein leeres Belegfeld ist kein Ersatz.
func TestBookingRequiresAReceipt(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant", "DE", "")

	req := env.receipt(t, vendor.ID, "buerobedarf", 5000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.ReceiptID = 0

	_, err := env.posting.PostIncomingReceipt(ctx, req)
	if err == nil {
		t.Fatal("ein Eingangsbeleg ohne abgelegten Beleg muss abgelehnt werden")
	}
	if !strings.Contains(err.Error(), "Eigenbeleg") {
		t.Errorf("die Meldung soll den Ausweg über den Eigenbeleg nennen, lautet aber: %v", err)
	}
}
