package receiptstore

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

const samplePDF = "%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\n%%EOF\n"

// failingReader delivers a few bytes and then breaks, the way a truncated upload
// or a disconnected drive would.
type failingReader struct {
	head      string
	delivered bool
}

func (r *failingReader) Read(p []byte) (int, error) {
	if !r.delivered {
		r.delivered = true
		n := copy(p, r.head)
		return n, nil
	}
	return 0, fmt.Errorf("die Quelle ist abgerissen")
}

func incomingDir(dataDir string) string {
	return filepath.Join(dataDir, "belege", "2026", "eingang")
}

// Eine mitten im Schreiben abgebrochene Ablage darf keine halbe Belegdatei
// hinterlassen. Geschrieben wird in eine Temporärdatei und erst am Ende
// umbenannt — scheitert es vorher, bleibt nichts unter einer Prüfsumme liegen.
func TestAbortedWriteLeavesNoFileUnderItsDigest(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)

	_, err := store.Put(2026, domain.DirectionIncoming, "abriss.pdf", &failingReader{head: samplePDF})
	if err == nil {
		t.Fatal("ein Lesefehler muss die Ablage scheitern lassen")
	}

	entries, err := os.ReadDir(incomingDir(dir))
	if err != nil {
		t.Fatalf("Belegordner konnte nicht gelesen werden: %v", err)
	}
	for _, entry := range entries {
		t.Errorf("nach dem Abbruch liegt %q im Belegordner — auch die Temporärdatei muss weg sein", entry.Name())
	}
}

// Nach einem erfolgreichen Schreibvorgang bleibt keine Temporärdatei zurück.
func TestSuccessfulWriteLeavesNoTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)

	stored, err := store.Put(2026, domain.DirectionIncoming, "rechnung.pdf", strings.NewReader(samplePDF))
	if err != nil {
		t.Fatalf("Ablage schlug fehl: %v", err)
	}

	entries, err := os.ReadDir(incomingDir(dir))
	if err != nil {
		t.Fatalf("Belegordner konnte nicht gelesen werden: %v", err)
	}
	if len(entries) != 1 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("erwartet genau eine Datei, bekommen %v", names)
	}
	if entries[0].Name() != stored.SHA256+".pdf" {
		t.Errorf("Dateiname = %q, erwartet %q", entries[0].Name(), stored.SHA256+".pdf")
	}
}

// Belegdateien sind nicht für andere Nutzer des Rechners bestimmt.
func TestStoredFilesAreNotWorldReadable(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)

	stored, err := store.Put(2026, domain.DirectionIncoming, "rechnung.pdf", strings.NewReader(samplePDF))
	if err != nil {
		t.Fatalf("Ablage schlug fehl: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(stored.RelPath)))
	if err != nil {
		t.Fatalf("abgelegte Datei nicht gefunden: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("Dateirechte = %o, erwartet 600", perm)
	}
	dirInfo, err := os.Stat(incomingDir(dir))
	if err != nil {
		t.Fatalf("Belegordner nicht gefunden: %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0o700 {
		t.Errorf("Ordnerrechte = %o, erwartet 700", perm)
	}
}

// Verify ist der Unterschied zwischen "die Prüfsumme steht in der Datenbank" und
// "die Datei auf der Platte ist noch die, die gebucht wurde".
func TestVerifyDetectsAChangedFile(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)

	stored, err := store.Put(2026, domain.DirectionIncoming, "rechnung.pdf", strings.NewReader(samplePDF))
	if err != nil {
		t.Fatalf("Ablage schlug fehl: %v", err)
	}
	if err := store.Verify(stored.RelPath, stored.SHA256); err != nil {
		t.Fatalf("frisch abgelegte Datei gilt als verändert: %v", err)
	}

	abs := filepath.Join(dir, filepath.FromSlash(stored.RelPath))
	if err := os.WriteFile(abs, []byte(samplePDF+"nachtraeglich"), 0o600); err != nil {
		t.Fatalf("Testdatei konnte nicht verändert werden: %v", err)
	}
	if err := store.Verify(stored.RelPath, stored.SHA256); err == nil {
		t.Error("eine veränderte Datei muss auffallen")
	}
}

// Ein Pfad aus der Datenbank darf nicht aus dem Datenordner herausführen.
func TestReadRefusesPathsOutsideTheDataDirectory(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)

	secret := filepath.Join(filepath.Dir(dir), "geheim.txt")
	if err := os.WriteFile(secret, []byte("nicht für Buchfink"), 0o600); err != nil {
		t.Fatalf("Testdatei konnte nicht geschrieben werden: %v", err)
	}
	t.Cleanup(func() { os.Remove(secret) })

	for _, path := range []string{
		"../geheim.txt",
		"belege/../../geheim.txt",
		"/etc/passwd",
	} {
		if _, err := store.Read(path); err == nil {
			t.Errorf("der Pfad %q hätte abgelehnt werden müssen", path)
		}
	}
}

// Die Richtung entscheidet über den Ordner; alles andere ist ein Fehler, kein
// Auffangordner.
func TestDirectionDecidesTheFolder(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)

	outgoing, err := store.Put(2026, domain.DirectionOutgoing, "rechnung.pdf", strings.NewReader(samplePDF))
	if err != nil {
		t.Fatalf("Ablage schlug fehl: %v", err)
	}
	if !strings.Contains(outgoing.RelPath, "belege/2026/ausgang/") {
		t.Errorf("unerwarteter Pfad %q", outgoing.RelPath)
	}

	if _, err := store.Put(2026, domain.Direction("seitwaerts"), "x.pdf", strings.NewReader(samplePDF)); err == nil {
		t.Error("eine unbekannte Belegrichtung muss abgelehnt werden")
	}
}

// Gleicher Inhalt, zweimal abgelegt: eine Datei, als Dublette gemeldet.
func TestDuplicateContentIsReported(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)

	first, err := store.Put(2026, domain.DirectionIncoming, "scan.pdf", strings.NewReader(samplePDF))
	if err != nil {
		t.Fatalf("Ablage schlug fehl: %v", err)
	}
	if first.Deduplicated {
		t.Error("die erste Ablage ist keine Dublette")
	}

	second, err := store.Put(2026, domain.DirectionIncoming, "kopie.pdf", strings.NewReader(samplePDF))
	if err != nil {
		t.Fatalf("zweite Ablage schlug fehl: %v", err)
	}
	if !second.Deduplicated {
		t.Error("gleicher Inhalt muss als Dublette erkannt werden")
	}
	if second.RelPath != first.RelPath {
		t.Errorf("Dublette liegt woanders: %q vs %q", second.RelPath, first.RelPath)
	}
}

// XML wird als E-Rechnungsformat erkannt, auch ohne Endung im Namen.
func TestXMLIsDetectedAsApplicationXML(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)

	xml := `<?xml version="1.0" encoding="UTF-8"?><Invoice><ID>1</ID></Invoice>`
	stored, err := store.Put(2026, domain.DirectionIncoming, "xrechnung.xml", strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Ablage schlug fehl: %v", err)
	}
	if stored.MimeType != "application/xml" {
		t.Errorf("Dateityp = %q, erwartet application/xml", stored.MimeType)
	}
}

// Größere Dateien laufen durch denselben gepufferten Weg wie kleine.
func TestLargeFileIsHashedAndStoredWhole(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)

	payload := samplePDF + strings.Repeat("A", 200*1024)
	stored, err := store.Put(2026, domain.DirectionIncoming, "grosser-scan.pdf", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("Ablage schlug fehl: %v", err)
	}
	if stored.Size != int64(len(payload)) {
		t.Errorf("Größe = %d, erwartet %d", stored.Size, len(payload))
	}
	if stored.MimeType != "application/pdf" {
		t.Errorf("Dateityp = %q, erwartet application/pdf", stored.MimeType)
	}
	if err := store.Verify(stored.RelPath, stored.SHA256); err != nil {
		t.Errorf("die abgelegte Datei stimmt nicht mit ihrer Prüfsumme überein: %v", err)
	}

	data, err := store.Read(stored.RelPath)
	if err != nil {
		t.Fatalf("Lesen schlug fehl: %v", err)
	}
	if string(data) != payload {
		t.Error("der gelesene Inhalt weicht vom geschriebenen ab")
	}
}

var _ io.Reader = (*failingReader)(nil)
