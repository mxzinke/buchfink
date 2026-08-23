package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// minimalPDF is enough for http.DetectContentType to report application/pdf.
const minimalPDF = "%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n<< /Root 1 0 R >>\n%%EOF\n"

const xRechnungXML = `<?xml version="1.0" encoding="UTF-8"?>
<rsm:CrossIndustryInvoice xmlns:rsm="urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100">
  <rsm:ExchangedDocument><ram:ID>RE-4711</ram:ID></rsm:ExchangedDocument>
</rsm:CrossIndustryInvoice>`

func (e *testEnv) writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("Testdatei %s konnte nicht geschrieben werden: %v", name, err)
	}
	return path
}

// fileIncoming files a plain PDF receipt away and returns it.
func (e *testEnv) fileIncoming(t *testing.T, name string) *domain.Receipt {
	t.Helper()
	receipt, err := e.receipts.File(context.Background(), FileReceiptRequest{
		Direction: domain.DirectionIncoming,
		Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, Path: e.writeTempFile(t, name, minimalPDF)},
		},
	})
	if err != nil {
		t.Fatalf("Beleg %s konnte nicht abgelegt werden: %v", name, err)
	}
	return receipt
}

// Ein Beleg ist mehrere Dateien. Der hybride Fall ist der, an dem das alte
// Ein-Datei-Modell zerbrach: PDF zum Ansehen, XML zum Buchen.
func TestHybridReceiptCarriesBothParts(t *testing.T) {
	env := newTestEnv(t)

	receipt, err := env.receipts.File(context.Background(), FileReceiptRequest{
		Direction:   domain.DirectionIncoming,
		ReceivedVia: domain.ReceivedViaEmail,
		Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, Path: env.writeTempFile(t, "rechnung.pdf", minimalPDF)},
			{Role: domain.ReceiptRoleStructured, FileName: "factur-x.xml", Content: []byte(xRechnungXML), Derived: true},
		},
	})
	if err != nil {
		t.Fatalf("Hybridbeleg konnte nicht abgelegt werden: %v", err)
	}

	if len(receipt.Files) != 2 {
		t.Fatalf("erwartet 2 Dateien, bekommen %d", len(receipt.Files))
	}
	if err := receipt.ValidateBookable(); err != nil {
		t.Fatalf("ein Hybridbeleg ist buchbar, wurde aber abgelehnt: %v", err)
	}

	// Der Bildteil ist Anzeige, nie Buchungsquelle.
	display, ok := receipt.DisplayFile()
	if !ok || display.Role != domain.ReceiptRoleOriginal {
		t.Fatalf("angezeigt werden soll der PDF-Teil, bekommen %+v", display)
	}
	if display.MimeType != "application/pdf" {
		t.Errorf("Dateityp des Originals = %q, erwartet application/pdf", display.MimeType)
	}
	structured, ok := receipt.FileByRole(domain.ReceiptRoleStructured)
	if !ok || !structured.Derived {
		t.Fatal("der aus dem PDF gezogene strukturierte Teil muss als abgeleitet gekennzeichnet sein")
	}
}

// Ablegen und Buchen sind zwei Schritte. Eine XRechnung muss sofort ablegbar
// sein (GoBD Rz. 131), aber erst buchbar, wenn eine Darstellung existiert.
func TestXRechnungIsFileableButNotBookableWithoutRendering(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt, err := env.receipts.File(ctx, FileReceiptRequest{
		Direction: domain.DirectionIncoming,
		Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, FileName: "xrechnung.xml", Content: []byte(xRechnungXML)},
			{Role: domain.ReceiptRoleStructured, FileName: "xrechnung.xml", Content: []byte(xRechnungXML)},
		},
	})
	if err != nil {
		t.Fatalf("eine XRechnung muss ablegbar sein, wurde aber abgelehnt: %v", err)
	}
	if err := receipt.ValidateStructure(); err != nil {
		t.Fatalf("Strukturprüfung schlug fehl: %v", err)
	}

	err = receipt.ValidateBookable()
	if err == nil {
		t.Fatal("eine XRechnung ohne Darstellung darf nicht buchbar sein")
	}
	if !strings.Contains(err.Error(), "Darstellung") {
		t.Errorf("die Meldung soll die fehlende Darstellung nennen, lautet aber: %v", err)
	}

	// Mit der erzeugten Darstellung wird derselbe Beleg buchbar.
	withRendering, err := env.receipts.AddFile(ctx, receipt.ID, NewFile{
		Role: domain.ReceiptRoleRendering, FileName: "xrechnung.pdf",
		Content: []byte(minimalPDF), Derived: true,
	})
	if err != nil {
		t.Fatalf("Darstellung konnte nicht ergänzt werden: %v", err)
	}
	if err := withRendering.ValidateBookable(); err != nil {
		t.Fatalf("mit Darstellung muss die XRechnung buchbar sein: %v", err)
	}
	display, _ := withRendering.DisplayFile()
	if display.Role != domain.ReceiptRoleRendering {
		t.Errorf("angezeigt werden soll die erzeugte Darstellung, bekommen %q", display.Role)
	}
}

// Genau ein Original je Beleg. Zwei empfangene Formen sind zwei Belege.
func TestStructureRejectsTwoOriginals(t *testing.T) {
	env := newTestEnv(t)

	_, err := env.receipts.File(context.Background(), FileReceiptRequest{
		Direction: domain.DirectionIncoming,
		Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, Path: env.writeTempFile(t, "a.pdf", minimalPDF)},
			{Role: domain.ReceiptRoleOriginal, Path: env.writeTempFile(t, "b.pdf", minimalPDF+"zweite")},
		},
	})
	if err == nil {
		t.Fatal("zwei Originaldateien in einem Beleg müssen abgelehnt werden")
	}
	if !strings.Contains(err.Error(), "genau eine") {
		t.Errorf("unerwartete Meldung: %v", err)
	}

	// Und ein Beleg ohne Original ebenso.
	_, err = env.receipts.File(context.Background(), FileReceiptRequest{
		Direction: domain.DirectionIncoming,
		Files: []NewFile{
			{Role: domain.ReceiptRoleAttachment, FileName: "eigenbeleg.txt", Content: []byte("Teilnehmer: A, B")},
		},
	})
	if err == nil {
		t.Fatal("ein Beleg ohne empfangene Originaldatei muss abgelehnt werden")
	}
}

// Der Beleg-Hash deckt die geordnete Dateiliste ab: jede zusätzliche, entfernte
// oder umsortierte Datei ändert ihn.
func TestReceiptHashCoversTheOrderedFileList(t *testing.T) {
	base := &domain.Receipt{Files: []domain.ReceiptFile{
		{Position: 1, Role: domain.ReceiptRoleOriginal, FileName: "rechnung.pdf", SHA256: strings.Repeat("a", 64)},
		{Position: 2, Role: domain.ReceiptRoleStructured, FileName: "factur-x.xml", SHA256: strings.Repeat("b", 64)},
	}}
	want := accounting.ReceiptHash(base)

	stable := &domain.Receipt{Files: []domain.ReceiptFile{
		{Position: 2, Role: domain.ReceiptRoleStructured, FileName: "factur-x.xml", SHA256: strings.Repeat("b", 64)},
		{Position: 1, Role: domain.ReceiptRoleOriginal, FileName: "rechnung.pdf", SHA256: strings.Repeat("a", 64)},
	}}
	if got := accounting.ReceiptHash(stable); got != want {
		t.Error("die Lesereihenfolge aus der Datenbank darf den Beleg-Hash nicht ändern")
	}

	cases := map[string]func(r *domain.Receipt){
		"zusätzliche Datei": func(r *domain.Receipt) {
			r.Files = append(r.Files, domain.ReceiptFile{
				Position: 3, Role: domain.ReceiptRoleAttachment,
				FileName: "eigenbeleg.pdf", SHA256: strings.Repeat("c", 64),
			})
		},
		"entfernte Datei":       func(r *domain.Receipt) { r.Files = r.Files[:1] },
		"ausgetauschter Inhalt": func(r *domain.Receipt) { r.Files[1].SHA256 = strings.Repeat("d", 64) },
		"umbenannte Datei":      func(r *domain.Receipt) { r.Files[0].FileName = "andere.pdf" },
		"geänderte Rolle":       func(r *domain.Receipt) { r.Files[1].Role = domain.ReceiptRoleAttachment },
		"vertauschte Positionen": func(r *domain.Receipt) {
			r.Files[0].Position, r.Files[1].Position = 2, 1
		},
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			changed := &domain.Receipt{Files: append([]domain.ReceiptFile(nil), base.Files...)}
			mutate(changed)
			if got := accounting.ReceiptHash(changed); got == want {
				t.Errorf("%s hat den Beleg-Hash nicht verändert", name)
			}
		})
	}
}

// Ein Dateiname darf keine Feldgrenze vortäuschen können — deshalb ist die
// Kanonisierung längenpräfigiert.
func TestReceiptHashResistsFieldInjectionViaFileName(t *testing.T) {
	honest := &domain.Receipt{Files: []domain.ReceiptFile{
		{Position: 1, Role: domain.ReceiptRoleOriginal, FileName: "a.pdf", SHA256: strings.Repeat("a", 64)},
		{Position: 2, Role: domain.ReceiptRoleAttachment, FileName: "b.pdf", SHA256: strings.Repeat("b", 64)},
	}}
	forged := &domain.Receipt{Files: []domain.ReceiptFile{
		{
			Position: 1, Role: domain.ReceiptRoleOriginal,
			FileName: "a.pdf\nsha256:64:" + strings.Repeat("a", 64) + "\nrole:10:attachment\nname:5:b.pdf",
			SHA256:   strings.Repeat("b", 64),
		},
	}}
	if accounting.ReceiptHash(honest) == accounting.ReceiptHash(forged) {
		t.Fatal("ein präparierter Dateiname konnte eine zweite Datei vortäuschen")
	}
}

// Die Ablage ist inhaltsadressiert: identische Dateien liegen einmal auf der
// Platte, und der Dateiname ist die Prüfsumme.
func TestIdenticalContentIsStoredOnce(t *testing.T) {
	env := newTestEnv(t)

	first := env.fileIncoming(t, "scan.pdf")
	second := env.fileIncoming(t, "gleicher-scan.pdf")

	if first.Files[0].SHA256 != second.Files[0].SHA256 {
		t.Fatal("gleicher Inhalt muss dieselbe Prüfsumme ergeben")
	}
	if first.Files[0].StoredPath != second.Files[0].StoredPath {
		t.Errorf("gleicher Inhalt muss am selben Ort liegen: %q vs %q",
			first.Files[0].StoredPath, second.Files[0].StoredPath)
	}

	// Der Dateiname auf der Platte ist die Prüfsumme, nicht die Belegnummer.
	name := filepath.Base(first.Files[0].StoredPath)
	if !strings.HasPrefix(name, first.Files[0].SHA256) {
		t.Errorf("der Dateiname %q soll mit der Prüfsumme beginnen", name)
	}
	if strings.Contains(first.Files[0].StoredPath, first.ReceiptNumber) {
		t.Error("die Belegnummer darf nicht im Pfad stehen — sonst hinge die Ablage am Nummernkreis")
	}

	// Und die Ablage liegt nach Jahr und Richtung.
	if !strings.Contains(first.Files[0].StoredPath, "belege/2026/eingang/") {
		t.Errorf("unerwarteter Ablagepfad %q", first.Files[0].StoredPath)
	}

	entries, err := os.ReadDir(filepath.Join(env.dataDir, "belege", "2026", "eingang"))
	if err != nil {
		t.Fatalf("Belegordner konnte nicht gelesen werden: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("erwartet genau eine Datei auf der Platte, bekommen %d", len(entries))
	}
}

// Eine unlesbare Quelldatei scheitert, bevor irgendetwas abgelegt wird. Der
// Abbruch mitten im Schreiben ist in internal/receiptstore geprüft.
func TestUnreadableSourceFileIsRejected(t *testing.T) {
	env := newTestEnv(t)

	_, err := env.receipts.File(context.Background(), FileReceiptRequest{
		Direction: domain.DirectionIncoming,
		Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, Path: filepath.Join(t.TempDir(), "gibtesnicht.pdf")},
		},
	})
	if err == nil {
		t.Fatal("eine fehlende Quelldatei muss zu einem Fehler führen")
	}

	dir := filepath.Join(env.dataDir, "belege", "2026", "eingang")
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		return // Verzeichnis wurde gar nicht erst angelegt: auch richtig.
	}
	for _, entry := range entries {
		t.Errorf("nach der gescheiterten Ablage liegt noch %q im Belegordner", entry.Name())
	}
}

// Eine leere Datei ist kein Beleg.
func TestEmptyFileIsRejected(t *testing.T) {
	env := newTestEnv(t)

	_, err := env.receipts.File(context.Background(), FileReceiptRequest{
		Direction: domain.DirectionIncoming,
		Files:     []NewFile{{Role: domain.ReceiptRoleOriginal, Path: env.writeTempFile(t, "leer.pdf", "")}},
	})
	if err == nil {
		t.Fatal("eine leere Datei muss abgelehnt werden")
	}
}

// Belegnummern kommen aus einem eigenen, lückenlosen Nummernkreis je Richtung.
func TestReceiptNumbersAreGaplessPerDirection(t *testing.T) {
	env := newTestEnv(t)

	for i, want := range []string{"ER-2026-0001", "ER-2026-0002", "ER-2026-0003"} {
		receipt := env.fileIncoming(t, string(rune('a'+i))+".pdf")
		if receipt.ReceiptNumber != want {
			t.Fatalf("Belegnummer %d = %q, erwartet %q", i+1, receipt.ReceiptNumber, want)
		}
	}

	// Ein Ausgangsbeleg trägt die Rechnungsnummer, die die Rechnung vergeben hat.
	outgoing, err := env.receipts.File(context.Background(), FileReceiptRequest{
		Direction:     domain.DirectionOutgoing,
		ReceiptNumber: "RE-2026-0007",
		Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, FileName: "rechnung.pdf", Content: []byte(minimalPDF + "ausgang")},
		},
	})
	if err != nil {
		t.Fatalf("Ausgangsbeleg konnte nicht abgelegt werden: %v", err)
	}
	if outgoing.ReceiptNumber != "RE-2026-0007" {
		t.Errorf("Ausgangsbeleg soll die Rechnungsnummer tragen, hat aber %q", outgoing.ReceiptNumber)
	}
	if !strings.Contains(outgoing.Files[0].StoredPath, "belege/2026/ausgang/") {
		t.Errorf("Ausgangsbelege gehören nach .../ausgang/, liegen aber unter %q", outgoing.Files[0].StoredPath)
	}
}

// Mit der Buchung ist der Beleg versiegelt: nachträglich eine Datei anzuhängen
// würde den Beleg-Hash und damit die Kette brechen.
func TestSealedReceiptRefusesFileChanges(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt := env.fileIncoming(t, "rechnung.pdf")
	hashBefore := receipt.ReceiptHash

	if err := env.receipts.Seal(ctx, receipt.ID, 42); err != nil {
		t.Fatalf("Versiegeln schlug fehl: %v", err)
	}

	sealed, err := env.receipts.Get(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Beleg konnte nicht geladen werden: %v", err)
	}
	if sealed.Status != domain.ReceiptStatusSealed || sealed.JournalEntryID == nil || *sealed.JournalEntryID != 42 {
		t.Fatalf("Beleg ist nicht versiegelt: %+v", sealed)
	}
	if sealed.ReceiptHash != hashBefore {
		t.Error("das Versiegeln darf den Beleg-Hash nicht verändern")
	}

	_, err = env.receipts.AddFile(ctx, receipt.ID, NewFile{
		Role: domain.ReceiptRoleAttachment, FileName: "mahnung.pdf", Content: []byte(minimalPDF + "mahnung"),
	})
	if err == nil {
		t.Fatal("an einen gebuchten Beleg darf keine Datei mehr angehängt werden")
	}
	if !strings.Contains(err.Error(), "eigener Beleg") {
		t.Errorf("die Meldung soll auf den eigenen Beleg verweisen, lautet aber: %v", err)
	}
}

// Das Versiegeln liegt hinter dem Journalschreibvorgang und ist deshalb
// wiederholbar: bricht es ab, muss der zweite Versuch reparieren statt scheitern.
func TestSealIsIdempotentButRefusesASecondBooking(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt := env.fileIncoming(t, "rechnung.pdf")
	if err := env.receipts.Seal(ctx, receipt.ID, 7); err != nil {
		t.Fatalf("erstes Versiegeln schlug fehl: %v", err)
	}
	if err := env.receipts.Seal(ctx, receipt.ID, 7); err != nil {
		t.Fatalf("das Versiegeln muss wiederholbar sein, schlug aber fehl: %v", err)
	}
	if err := env.receipts.Seal(ctx, receipt.ID, 8); err == nil {
		t.Fatal("ein Beleg darf nicht an zwei Buchungen hängen")
	}
}

// Ein abgelegter Beleg wird nicht gelöscht, sondern verworfen — er hat eine
// Belegnummer, und ein empfangenes Dokument darf nicht spurlos verschwinden.
func TestDiscardKeepsTheReceiptAndRequiresAReason(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt := env.fileIncoming(t, "doppelt.pdf")

	if err := env.receipts.Discard(ctx, receipt.ID, ""); err == nil {
		t.Fatal("Verwerfen ohne Begründung muss abgelehnt werden")
	}
	if err := env.receipts.Discard(ctx, receipt.ID, "doppelt erfasst"); err != nil {
		t.Fatalf("Verwerfen schlug fehl: %v", err)
	}

	discarded, err := env.receipts.Get(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("ein verworfener Beleg muss auffindbar bleiben: %v", err)
	}
	if discarded.Status != domain.ReceiptStatusDiscarded {
		t.Errorf("Status = %q, erwartet %q", discarded.Status, domain.ReceiptStatusDiscarded)
	}
	if discarded.ReceiptNumber != receipt.ReceiptNumber {
		t.Error("die Belegnummer muss erhalten bleiben")
	}
	if err := discarded.ValidateBookable(); err == nil {
		t.Error("ein verworfener Beleg darf nicht buchbar sein")
	}
}

// Die Vorschau liefert den Inhalt und sagt, ob er noch zu seiner Prüfsumme passt.
func TestContentReportsTamperedFiles(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt := env.fileIncoming(t, "rechnung.pdf")
	content, err := env.receipts.DisplayContent(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Belegdatei konnte nicht gelesen werden: %v", err)
	}
	if !content.Intact {
		t.Fatal("eine frisch abgelegte Datei muss zu ihrer Prüfsumme passen")
	}
	if content.MimeType != "application/pdf" || content.FileName != "rechnung.pdf" {
		t.Errorf("unerwartete Metadaten: %+v", content)
	}

	onDisk := filepath.Join(env.dataDir, filepath.FromSlash(receipt.Files[0].StoredPath))
	if err := os.WriteFile(onDisk, []byte(minimalPDF+"manipuliert"), 0o600); err != nil {
		t.Fatalf("Testdatei konnte nicht verändert werden: %v", err)
	}

	content, err = env.receipts.DisplayContent(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("eine veränderte Datei muss trotzdem anzeigbar sein: %v", err)
	}
	if content.Intact {
		t.Error("eine veränderte Datei muss als solche gemeldet werden")
	}
}

// Ein Dateityp, den der Nutzer nicht ansehen kann, macht den Beleg nicht
// unablegbar — nur unbuchbar. Der Dateiname lügt dabei nicht über den Inhalt.
func TestMimeTypeComesFromTheContentNotTheName(t *testing.T) {
	env := newTestEnv(t)

	receipt, err := env.receipts.File(context.Background(), FileReceiptRequest{
		Direction: domain.DirectionIncoming,
		Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, Path: env.writeTempFile(t, "rechnung.txt", minimalPDF)},
		},
	})
	if err != nil {
		t.Fatalf("Beleg konnte nicht abgelegt werden: %v", err)
	}
	if receipt.Files[0].MimeType != "application/pdf" {
		t.Errorf("Dateityp = %q, erwartet application/pdf — der Inhalt schlägt den Namen",
			receipt.Files[0].MimeType)
	}
}
