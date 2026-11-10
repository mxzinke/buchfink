package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Scheitert die Generalumkehr, entsteht kein Stornodokument und keine Nummer
// wird verbraucht.
//
// Das ist die Klammer der Entscheidung 1, angewandt auf den Storno: Nummer,
// Dokument und Buchung entstehen zusammen oder gar nicht. Vorher lag die
// Generalumkehr hinter der Nummernvergabe — ein nummeriertes Stornodokument
// ohne Buchung blieb zurück, und der nächste Versuch verbrauchte eine zweite
// Nummer.
func TestCancelIsAtomicWhenTheReversalFails(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)

	original := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, original); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}
	// Die Buchung wird außerhalb des Rechnungswegs storniert. Der Storno über
	// den Rechnungsweg muss daran scheitern — eine Buchung lässt sich nur einmal
	// durch Generalumkehr zurücknehmen.
	if _, err := env.journal.Reverse(ctx, *original.JournalEntryID, "Korrektur im Journal"); err != nil {
		t.Fatalf("Generalumkehr im Journal: %v", err)
	}

	_, err := svc.CancelWithDocument(ctx, original.ID, "Leistung nicht erbracht")
	if err == nil {
		t.Fatal("der Storno muss scheitern, wenn die Generalumkehr nicht möglich ist")
	}

	// Keine zweite Nummer verbraucht …
	next, err := env.numberRepo.Peek(ctx, domain.NumberRangeInvoice, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if next != 2 {
		t.Errorf("der Zähler steht auf %d, erwartet 2 — der gescheiterte Storno hat eine Nummer verbraucht", next)
	}
	// … kein Stornodokument entstanden …
	numbers, err := env.invoiceRepoOf(t).FindNumbers(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(numbers) != 1 {
		t.Errorf("erwartet nur die Ursprungsrechnung, erhalten %v", numbers)
	}
	// … und die Ursprungsrechnung steht unverändert da.
	reloaded, err := env.invoiceRepoOf(t).FindByID(ctx, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Status == domain.InvoiceStatusCancelled || reloaded.CancelledByInvoiceID != nil {
		t.Errorf("die Ursprungsrechnung gilt als storniert, obwohl der Storno gescheitert ist: %q", reloaded.Status)
	}
	// Und der Lückenbericht meldet keine Lücke: die Transaktion ist
	// zurückgerollt.
	report, err := svc.NumberGaps(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Gaps) != 0 {
		t.Errorf("der Bericht meldet %d Lücken, erwartet keine: %+v", len(report.Gaps), report.Gaps)
	}
}

// Der Storno: eigenes Dokument mit eigener Nummer, negierte Beträge, Bezug auf
// die Ursprungsrechnung, Generalumkehr im Journal — und die Ursprungsrechnung
// bleibt unverändert archiviert.
func TestCancelWithDocument(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)

	original := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, original); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}

	if _, err := svc.CancelWithDocument(ctx, original.ID, ""); err == nil {
		t.Error("ein Storno ohne Begründung darf nicht durchgehen (GoBD Rz. 58)")
	}

	storno, err := svc.CancelWithDocument(ctx, original.ID, "Leistung nicht erbracht")
	if err != nil {
		t.Fatalf("stornieren: %v", err)
	}

	if storno.InvoiceNumber == original.InvoiceNumber {
		t.Error("das Stornodokument braucht eine eigene Nummer")
	}
	if storno.Kind != domain.InvoiceKindCancellation {
		t.Errorf("Dokumentart = %q, erwartet %q", storno.Kind, domain.InvoiceKindCancellation)
	}
	if storno.GrossAmount != -original.GrossAmount {
		t.Errorf("Storno über %s, erwartet %s", storno.GrossAmount, -original.GrossAmount)
	}
	if storno.CorrectsInvoiceNumber != original.InvoiceNumber || storno.CorrectsInvoiceDate != original.Date {
		t.Errorf("der Bezug auf die stornierte Rechnung fehlt: %q vom %q",
			storno.CorrectsInvoiceNumber, storno.CorrectsInvoiceDate)
	}

	// Die Kette ist in beide Richtungen lesbar.
	reloaded, err := env.invoiceRepoOf(t).FindByID(ctx, original.ID)
	if err != nil {
		t.Fatalf("Ursprungsrechnung laden: %v", err)
	}
	if reloaded.Status != domain.InvoiceStatusCancelled {
		t.Errorf("die Ursprungsrechnung steht auf %q, erwartet %q", reloaded.Status, domain.InvoiceStatusCancelled)
	}
	if reloaded.CancelledByInvoiceID == nil || *reloaded.CancelledByInvoiceID != storno.ID {
		t.Error("die Ursprungsrechnung muss auf ihr Stornodokument zeigen")
	}
	// Unverändert archiviert: Beträge und Positionen bleiben, wie sie waren.
	if reloaded.GrossAmount != original.GrossAmount || len(reloaded.Items) != 1 {
		t.Error("die Ursprungsrechnung darf nicht verändert werden")
	}

	// Und im Journal steht die Generalumkehr.
	reversal, err := env.journalRepo.FindReversalOf(ctx, *original.JournalEntryID)
	if err != nil || reversal == nil {
		t.Fatalf("die Generalumkehr fehlt: %v", err)
	}
	for _, l := range reversal.Lines {
		if l.Amount > 0 {
			t.Errorf("eine Generalumkehr trägt negative Beträge, hier steht %s auf %s", l.Amount, l.Account)
		}
	}
	if storno.JournalEntryID == nil || *storno.JournalEntryID != reversal.ID {
		t.Error("das Stornodokument muss auf die Generalumkehr zeigen")
	}

	// Ein zweites Storno gibt es nicht.
	if _, err := svc.CancelWithDocument(ctx, original.ID, "nochmal"); err == nil {
		t.Error("eine bereits stornierte Rechnung darf nicht erneut storniert werden")
	}
}

// Ein Storno des Stornos gibt es nicht — weder über den Storno- noch über den
// Berichtigungsweg.
//
// Der Journalweg fängt den Fall nur zufällig ab: eine Generalumkehr lässt sich
// nicht erneut umkehren. Bei der Abschlagsrechnung greift das nicht, denn sie
// bucht beim Ausstellen nichts — ihr Stornodokument hat keine Buchung, die sich
// sperren ließe, und ohne die Prüfung der Dokumentart entstünde ein zweites
// Dokument mit den wieder positiven Beträgen der Ursprungsrechnung.
func TestCancellationCannotBeCancelledOrCorrected(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)

	original := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, original); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}
	storno, err := svc.CancelWithDocument(ctx, original.ID, "Leistung nicht erbracht")
	if err != nil {
		t.Fatalf("stornieren: %v", err)
	}

	// Das Stornodokument steht auf „ausgestellt" wie jede Rechnung; die Sperre
	// richtet sich deshalb nach der Dokumentart und nicht nach dem Status.
	if !storno.Status.IsIssued() {
		t.Fatalf("das Stornodokument steht auf %q, erwartet ausgestellt", storno.Status)
	}

	_, err = svc.CancelWithDocument(ctx, storno.ID, "doch nicht")
	if err == nil || !strings.Contains(err.Error(), "Stornorechnung") {
		t.Errorf("das Storno einer Stornorechnung muss mit einem Hinweis auf die Dokumentart "+
			"abgewiesen werden, erhalten: %v", err)
	}
	if err != nil && !strings.Contains(err.Error(), original.InvoiceNumber) {
		t.Errorf("die Abweisung soll die berichtigbare Rechnung nennen, erhalten: %v", err)
	}

	replacement := env.simpleInvoice(customer.ID, "2026-03-20", 90000)
	_, err = svc.CorrectInvoice(ctx, storno.ID, "doch nicht", replacement)
	if err == nil || !strings.Contains(err.Error(), "Stornorechnung") {
		t.Errorf("die Berichtigung einer Stornorechnung muss abgewiesen werden, erhalten: %v", err)
	}

	// Und der abgewiesene Versuch hat nichts angefasst.
	reloaded, err := env.invoiceRepoOf(t).FindByID(ctx, storno.ID)
	if err != nil {
		t.Fatalf("Stornodokument laden: %v", err)
	}
	if reloaded.Status != storno.Status || reloaded.CancelledByInvoiceID != nil {
		t.Errorf("das Stornodokument steht nach dem abgewiesenen Versuch auf %q (storniert durch %v)",
			reloaded.Status, reloaded.CancelledByInvoiceID)
	}
	if replacement.InvoiceNumber != "" {
		t.Errorf("die abgewiesene Berichtigung hat die Nummer %s verbraucht", replacement.InvoiceNumber)
	}

	// Der Abschlag: sein Storno hat keine Buchung, und der Journalweg sperrt
	// hier nichts.
	group := env.group(t, svc, customer.ID, 1000000)
	advance, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-01", Net: 400000,
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung ausstellen: %v", err)
	}
	advanceStorno, err := svc.CancelWithDocument(ctx, advance.ID, "Auftrag geändert")
	if err != nil {
		t.Fatalf("Abschlag stornieren: %v", err)
	}
	if advanceStorno.JournalEntryID != nil {
		t.Fatal("das Storno eines nicht vereinnahmten Abschlags bucht nichts; der Test prüft sonst den Journalweg")
	}
	if _, err := svc.CancelWithDocument(ctx, advanceStorno.ID, "doch nicht"); err == nil {
		t.Error("auch das Storno eines Abschlagsstornos muss abgewiesen werden")
	}
}

// Die Berichtigung: Storno plus neue Rechnung mit Bezug auf die berichtigte.
func TestCorrectInvoiceIssuesReplacementWithReference(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)

	original := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, original); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}

	replacement := env.simpleInvoice(customer.ID, "2026-03-15", 120000)
	corrected, err := svc.CorrectInvoice(ctx, original.ID, "Falscher Stundensatz", replacement)
	if err != nil {
		t.Fatalf("berichtigen: %v", err)
	}
	if corrected.Kind != domain.InvoiceKindCorrection {
		t.Errorf("die berichtigte Rechnung ist %q, erwartet %q", corrected.Kind, domain.InvoiceKindCorrection)
	}
	if corrected.CorrectsInvoiceNumber != original.InvoiceNumber {
		t.Errorf("der Bezug fehlt: %q", corrected.CorrectsInvoiceNumber)
	}
	if corrected.JournalEntryID == nil {
		t.Error("die berichtigte Rechnung muss gebucht sein")
	}
	if corrected.NetAmount != 120000 {
		t.Errorf("die berichtigte Rechnung lautet über %s, erwartet 1200,00", corrected.NetAmount)
	}

	// Drei Dokumente, drei Nummern: Ursprung, Storno, Berichtigung.
	numbers, err := env.invoiceRepoOf(t).FindNumbers(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(numbers) != 3 {
		t.Errorf("erwartet drei Dokumente im Nummernkreis, erhalten %d: %v", len(numbers), numbers)
	}
}
