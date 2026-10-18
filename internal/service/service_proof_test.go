package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Der Leistungsnachweis steht am Beleg, geht in den Prüfpfad — und ändert den
// Beleg-Hash nicht.
func TestServiceProofIsSavedOutsideTheReceiptHash(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	receipt := env.fileIncoming(t, "rechnung.pdf")
	before := receipt.ReceiptHash

	updated, err := env.receipts.SaveServiceProof(
		ctx, receipt.ID, "geprüft gegen Bestellung 4711 vom 03.03.2026", "2026-03-20")
	if err != nil {
		t.Fatalf("Leistungsnachweis: %v", err)
	}
	if updated.ServiceProof == "" || updated.ServiceProofAt != "2026-03-20" {
		t.Fatalf("der Vermerk steht nicht am Beleg: %+v", updated)
	}
	if updated.ReceiptHash != before {
		t.Errorf("der Beleg-Hash hat sich geändert (%s → %s) — das bräche die Kette jeder Buchung darauf",
			before, updated.ReceiptHash)
	}

	// Ohne Datum gilt der heutige Tag; ein unlesbares Datum wird abgewiesen.
	if _, err := env.receipts.SaveServiceProof(ctx, receipt.ID, "nochmal geprüft", "20.03.2026"); err == nil {
		t.Error("ein unlesbares Datum muss abgewiesen werden")
	}

	svc := NewAuditTrailService(
		env.receiptRepo, env.journalRepo, repository.NewPaymentAllocationRepository(env.db),
		repository.NewBankRepository(env.db), nil)
	trail, err := svc.Trail(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Prüfpfad: %v", err)
	}
	if !strings.Contains(trail.ServiceProof, "Bestellung 4711") {
		t.Errorf("der Prüfpfad nennt den Leistungsnachweis nicht: %q", trail.ServiceProof)
	}
}

// Der fehlende Leistungsnachweis über der Grenze steht im Prüfbericht — und
// darunter nicht.
//
// Ohne diese Regel ließe sich eine Rechnung über tausend Euro ohne jeden
// Prüfvermerk buchen und festschreiben, ohne dass der Monatsabschluss oder das
// Prüferpaket es meldet: die Aufgabenliste allein sieht niemand, der den
// Bericht liest.
func TestCheckFindsMissingServiceProofAboveTheThreshold(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Die Voreinstellung der Grenze sind 1.000 €.
	small := env.fileWithAmount(t, "klein.pdf", 50_000, "2026-03-01")
	large := env.fileWithAmount(t, "gross.pdf", 500_000, "2026-03-02")

	run := runChecks(t, env.checksOn(t, "2026-04-01"), "2026-03-31")
	found := findingsFor(run, domain.CheckRuleServiceProofMissing)
	if len(found) != 1 {
		t.Fatalf("erwartet einen Befund, erhalten %d: %+v", len(found), found)
	}
	if found[0].ObjectID != fmt.Sprintf("%d", large.ID) {
		t.Errorf("der Befund zeigt auf Beleg %s, erwartet %d", found[0].ObjectID, large.ID)
	}
	if found[0].Severity != domain.CheckWarning {
		t.Errorf("Schwere = %q, erwartet einen Hinweis: die Grenze ist eine eigene Vorgabe",
			found[0].Severity)
	}
	if small.GrossAmount >= 100_000 {
		t.Fatal("der kleine Beleg liegt nicht unter der Grenze — dann prüft der Test nichts")
	}

	// Mit Vermerk verschwindet der Befund.
	if _, err := env.receipts.SaveServiceProof(
		ctx, large.ID, "geprüft gegen Bestellung 4711 vom 02.03.2026", "2026-03-05"); err != nil {
		t.Fatalf("Leistungsnachweis: %v", err)
	}
	after := runChecks(t, env.checksOn(t, "2026-04-01"), "2026-03-31")
	if got := findingsFor(after, domain.CheckRuleServiceProofMissing); len(got) != 0 {
		t.Errorf("nach dem Vermerk bleibt ein Befund stehen: %+v", got)
	}
}

// fileWithAmount legt einen Eingangsbeleg mit einem bestimmten Bruttobetrag ab.
func (e *testEnv) fileWithAmount(
	t *testing.T, name string, gross domain.Cents, documentDate string,
) *domain.Receipt {
	t.Helper()
	receipt, err := e.receipts.File(context.Background(), FileReceiptRequest{
		DocumentDate: documentDate, IssuerName: "Lieferant GmbH", GrossAmount: gross,
		Direction: domain.DirectionIncoming, ReceivedAt: documentDate,
		Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, Path: e.writeTempFile(t, name, minimalPDF)},
		},
	})
	if err != nil {
		t.Fatalf("Beleg %s ablegen: %v", name, err)
	}
	return receipt
}

// Der Bestellbezug geht ins Änderungsprotokoll — wie der Leistungsnachweis.
//
// Beide Felder stehen außerhalb des Beleg-Hashes; was sie unveränderbar macht,
// ist allein das Protokoll. Ein Bestellbezug, der sich still ändern ließe,
// entwertete den Prüfpfad, den er stützen soll.
func TestOrderReferenceIsLogged(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	receipt := env.fileIncoming(t, "rechnung.pdf")

	updated, err := env.receipts.SaveOrderReference(ctx, receipt.ID, "Bestellung 4711")
	if err != nil {
		t.Fatalf("Bestellbezug: %v", err)
	}
	if updated.OrderReference != "Bestellung 4711" {
		t.Fatalf("der Bestellbezug steht nicht am Beleg: %+v", updated)
	}
	if updated.ReceiptHash != receipt.ReceiptHash {
		t.Errorf("der Beleg-Hash hat sich geändert (%s → %s)", receipt.ReceiptHash, updated.ReceiptHash)
	}

	entry := latestAudit(t, env, "RECEIPT")
	if !strings.Contains(entry.Details, "Bestellbezug") {
		t.Errorf("der Protokolleintrag nennt den Vorgang nicht: %q", entry.Details)
	}
	after := fields(t, entry.After)
	if after["orderReference"] != "Bestellung 4711" {
		t.Errorf("das Protokoll hält den neuen Stand nicht fest: %v", after)
	}
	before := fields(t, entry.Before)
	if before["orderReference"] != "" {
		t.Errorf("das Protokoll hält den alten Stand nicht fest: %v", before)
	}
}

// Ab der Grenze wird ein Eingangsbeleg ohne Leistungsnachweis nicht gebucht
// (RECH-08).
//
// Der Vermerk ist Pflichtfeld und nicht Hinweis: ein Kontrollschritt, den man
// mit einem Klick übergeht, ist keiner. Geprüft wird deshalb der Buchungsweg
// selbst — die Aufgabenliste und der Prüfbericht melden denselben Beleg, aber
// erst, wenn er schon gebucht ist.
func TestBookingRefusesLargeReceiptWithoutServiceProof(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	// Wie in der Anwendung (wailsbridge.initTenant): der Belegweg liest die
	// Grenze aus den Einstellungen. Voreinstellung sind 1.000 €.
	env.posting.SetSettingsSource(repository.NewSettingsRepository(env.db))
	vendor := env.vendor(t, "Grossauftrag GmbH", "DE", "")

	large := env.fileWithAmount(t, "gross.pdf", 595_000, "2026-03-02")
	req := ReceiptRequest{
		ContactID: vendor.ID, ReceiptID: large.ID,
		BookingDate: "2026-03-10", DocumentDate: "2026-03-02",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-31",
		Description: "Fremdleistung", TaxTreatment: domain.TaxTreatmentDomestic,
		Positions:  []ReceiptPosition{{PostingGroup: "fremdleistungen", Net: 500_000, TaxRate: domain.TaxRateStandard}},
		Settlement: SettlementOpen,
	}

	// Die Vorschau sagt es, bevor jemand bucht.
	preview, err := env.posting.PreviewIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Buchungsvorschau: %v", err)
	}
	var warned bool
	for _, w := range preview.Warnings {
		if w.Code == serviceProofWarningCode {
			warned = true
		}
	}
	if !warned {
		t.Errorf("die Vorschau nennt den fehlenden Leistungsnachweis nicht: %+v", preview.Warnings)
	}

	_, err = env.posting.PostIncomingReceipt(ctx, req)
	if err == nil {
		t.Fatal("ein Eingangsbeleg über der Grenze darf ohne Leistungsnachweis nicht gebucht werden")
	}
	if !strings.Contains(err.Error(), "Leistungsnachweis") || !strings.Contains(err.Error(), "1.000,00") {
		t.Errorf("die Meldung nennt weder den Vermerk noch die Grenze: %v", err)
	}
	booked, err := env.receipts.Get(ctx, large.ID)
	if err != nil {
		t.Fatalf("Beleg lesen: %v", err)
	}
	if booked.Status == domain.ReceiptStatusSealed || booked.JournalEntryID != nil {
		t.Errorf("der abgewiesene Beleg darf nicht gebucht sein: %+v", booked)
	}

	// Mit dem Vermerk läuft dieselbe Buchung durch.
	if _, err := env.receipts.SaveServiceProof(
		ctx, large.ID, "geprüft gegen Bestellung 4711 vom 28.02.2026", "2026-03-09"); err != nil {
		t.Fatalf("Leistungsnachweis: %v", err)
	}
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Beleg mit Leistungsnachweis buchen: %v", err)
	}
	if entry.ReceiptID == nil || *entry.ReceiptID != large.ID {
		t.Errorf("die Buchung zeigt nicht auf den Beleg: %+v", entry)
	}

	// Unter der Grenze verlangt nichts einen Vermerk: der Regelfall bleibt ein
	// Klick.
	small := env.fileWithAmount(t, "klein.pdf", 11_900, "2026-03-03")
	if _, err := env.posting.PostIncomingReceipt(ctx, ReceiptRequest{
		ContactID: vendor.ID, ReceiptID: small.ID,
		BookingDate: "2026-03-11", DocumentDate: "2026-03-03",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-31",
		Description: "Bürobedarf", TaxTreatment: domain.TaxTreatmentDomestic,
		Positions:  []ReceiptPosition{{PostingGroup: "buerobedarf", Net: 10_000, TaxRate: domain.TaxRateStandard}},
		Settlement: SettlementOpen,
	}); err != nil {
		t.Fatalf("kleiner Beleg buchen: %v", err)
	}
}

// Der Buchungsdialog schickt den Leistungsnachweis mit der Buchung — ein Ruf
// statt zwei.
//
// Ohne dieses Feld müsste die Maske erst SaveServiceProof rufen und dann buchen;
// bricht der zweite Ruf ab, stünde ein Vermerk an einem Beleg, der nie gebucht
// wurde. Der Vermerk geht trotzdem über den Belegdienst — mit derselben Prüfung
// des Datums, demselben Protokolleintrag und unverändertem Beleg-Hash.
func TestBookingTakesTheServiceProofWithTheRequest(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.posting.SetSettingsSource(repository.NewSettingsRepository(env.db))
	vendor := env.vendor(t, "Grossauftrag GmbH", "DE", "")
	large := env.fileWithAmount(t, "gross.pdf", 595_000, "2026-03-02")
	before := large.ReceiptHash

	entry, err := env.posting.PostIncomingReceipt(ctx, ReceiptRequest{
		ContactID: vendor.ID, ReceiptID: large.ID,
		BookingDate: "2026-03-10", DocumentDate: "2026-03-02",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-31",
		Description: "Fremdleistung", TaxTreatment: domain.TaxTreatmentDomestic,
		Positions:      []ReceiptPosition{{PostingGroup: "fremdleistungen", Net: 500_000, TaxRate: domain.TaxRateStandard}},
		Settlement:     SettlementOpen,
		ServiceProof:   "geprüft gegen Bestellung 4711 vom 28.02.2026",
		ServiceProofAt: "2026-03-09",
	})
	if err != nil {
		t.Fatalf("Beleg mit mitgeschicktem Leistungsnachweis buchen: %v", err)
	}
	booked, err := env.receipts.Get(ctx, large.ID)
	if err != nil {
		t.Fatalf("Beleg lesen: %v", err)
	}
	if booked.ServiceProof == "" || booked.ServiceProofAt != "2026-03-09" {
		t.Errorf("der Vermerk steht nicht am Beleg: %+v", booked)
	}
	if booked.ReceiptHash != before {
		t.Errorf("der Beleg-Hash hat sich geändert (%s → %s)", before, booked.ReceiptHash)
	}
	if entry.ReceiptID == nil || *entry.ReceiptID != large.ID {
		t.Errorf("die Buchung zeigt nicht auf den Beleg: %+v", entry)
	}

	// Ein unlesbares Datum weist dieselbe Prüfung ab wie beim Nachtragen — und
	// dann wird auch nicht gebucht.
	second := env.fileWithAmount(t, "gross2.pdf", 595_000, "2026-03-03")
	if _, err := env.posting.PostIncomingReceipt(ctx, ReceiptRequest{
		ContactID: vendor.ID, ReceiptID: second.ID,
		BookingDate: "2026-03-11", DocumentDate: "2026-03-03",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-31",
		Description: "Fremdleistung", TaxTreatment: domain.TaxTreatmentDomestic,
		Positions:      []ReceiptPosition{{PostingGroup: "fremdleistungen", Net: 500_000, TaxRate: domain.TaxRateStandard}},
		Settlement:     SettlementOpen,
		ServiceProof:   "geprüft gegen Bestellung 4712",
		ServiceProofAt: "03.03.2026",
	}); err == nil {
		t.Error("ein unlesbares Datum des Vermerks muss die Buchung abweisen")
	}
	unbooked, err := env.receipts.Get(ctx, second.ID)
	if err != nil {
		t.Fatalf("Beleg lesen: %v", err)
	}
	if unbooked.JournalEntryID != nil {
		t.Errorf("der abgewiesene Beleg darf nicht gebucht sein: %+v", unbooked)
	}
}

// Eine gescheiterte Buchung hinterlässt keinen Leistungsnachweis.
//
// Der Vermerk und die Buchung sind ein Vorgang. Stünde der Vermerk vor der
// Transaktion am Beleg, bliebe er samt Protokolleintrag stehen, wenn die
// Buchung danach scheitert — hier am festgeschriebenen Zeitraum. Der Beleg
// behauptete dann eine Prüfung gegen die Bestellung, zu der es keine Buchung
// gibt, und beim zweiten Versuch verlangte niemand mehr den Vermerk: die
// Buchungssperre sieht ihn ja am Beleg.
func TestAbortedBookingLeavesNoServiceProof(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.posting.SetSettingsSource(repository.NewSettingsRepository(env.db))
	env.journal.SetFestschreibungRepo(repository.NewFestschreibungRepository(env.db))

	if err := repository.NewFestschreibungRepository(env.db).Create(ctx, &domain.Festschreibung{
		FiscalYear: 2026, PeriodType: "month", PeriodLabel: "März 2026",
		CutoffDate: "2026-03-31", ChainHead: "kettenkopf",
	}); err != nil {
		t.Fatalf("Festschreibung anlegen: %v", err)
	}

	vendor := env.vendor(t, "Grossauftrag GmbH", "DE", "")
	large := env.fileWithAmount(t, "gross-abbruch.pdf", 595_000, "2026-03-02")

	// Gezählt wird ab hier: Lieferant und Beleg stehen, und was das Protokoll
	// danach aufnimmt, gehört zum abgebrochenen Buchungsversuch.
	auditRepo := repository.NewAuditRepository(env.db)
	before, err := auditRepo.Count(ctx)
	if err != nil {
		t.Fatalf("Protokoll zählen: %v", err)
	}
	if _, err := env.posting.PostIncomingReceipt(ctx, ReceiptRequest{
		ContactID: vendor.ID, ReceiptID: large.ID,
		BookingDate: "2026-03-10", DocumentDate: "2026-03-02",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-31",
		Description: "Fremdleistung", TaxTreatment: domain.TaxTreatmentDomestic,
		Positions:      []ReceiptPosition{{PostingGroup: "fremdleistungen", Net: 500_000, TaxRate: domain.TaxRateStandard}},
		Settlement:     SettlementOpen,
		ServiceProof:   "geprüft gegen Bestellung 4714 vom 28.02.2026",
		ServiceProofAt: "2026-03-09",
	}); err == nil {
		t.Fatal("eine Buchung in einen festgeschriebenen Zeitraum muss abgewiesen werden")
	}

	after, err := env.receipts.Get(ctx, large.ID)
	if err != nil {
		t.Fatalf("Beleg lesen: %v", err)
	}
	if after.ServiceProof != "" || after.ServiceProofAt != "" {
		t.Errorf("der Vermerk steht am Beleg, obwohl keine Buchung entstand: %+v", after)
	}
	if after.JournalEntryID != nil {
		t.Errorf("der abgewiesene Beleg darf nicht gebucht sein: %+v", after)
	}
	count, err := auditRepo.Count(ctx)
	if err != nil {
		t.Fatalf("Protokoll zählen: %v", err)
	}
	if count != before {
		t.Errorf("das Änderungsprotokoll hat %d Einträge mehr, erwartet keinen", count-before)
	}
}
