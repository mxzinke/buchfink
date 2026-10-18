package inbox

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

const (
	ciiXML  = `<?xml version="1.0" encoding="UTF-8"?><rsm:CrossIndustryInvoice xmlns:rsm="urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"/>`
	ublXML  = `<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"/>`
	camtXML = `<?xml version="1.0" encoding="ISO-8859-1"?><Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02"><BkToCstmrStmt/></Document>`
	camt52  = `<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.052.001.02"/>`
	pdfData = "%PDF-1.7\nRechnung"
)

var pngData = string([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a})

func write(t *testing.T, path, content string, mod time.Time) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if !mod.IsZero() {
		if err := os.Chtimes(path, mod, mod); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

// Der Eingang zeigt jede Datei mit ihrer Art, ohne Übernommen/, versteckte
// Dateien und Symlinks; die älteste zuerst.
func TestListShowsKindsAndSkipsWhatIsDone(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Eingang")
	now := time.Now()
	day := func(n int) time.Time { return now.Add(-time.Duration(n) * 24 * time.Hour) }
	write(t, filepath.Join(dir, "rechnung.pdf"), pdfData, day(3))
	write(t, filepath.Join(dir, "Scans", "quittung.png"), pngData, day(20))
	write(t, filepath.Join(dir, "xrechnung.xml"), ciiXML, day(2))
	write(t, filepath.Join(dir, "ubl.xml"), ublXML, day(2))
	write(t, filepath.Join(dir, "auszug.xml"), camtXML, day(1))
	write(t, filepath.Join(dir, "vormerkposten.xml"), camt52, day(1))
	write(t, filepath.Join(dir, "notiz.txt"), "Notiz", day(1))
	write(t, filepath.Join(dir, "falsch.pdf"), "kein PDF", day(1))
	write(t, filepath.Join(dir, ".versteckt.pdf"), pdfData, day(30))
	write(t, filepath.Join(dir, ".sync", "intern.pdf"), pdfData, day(30))
	write(t, filepath.Join(dir, DoneDir, "2026-05", "alt.pdf"), pdfData, day(40))
	outside := write(t, filepath.Join(t.TempDir(), "privat.pdf"), pdfData, day(50))
	if err := os.Symlink(outside, filepath.Join(dir, "link.pdf")); err != nil {
		t.Fatal(err)
	}

	files, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]Kind{}
	for _, f := range files {
		kinds[f.Name] = f.Kind
		if (f.Kind == KindUnsupported) != (f.SHA256 == "") {
			t.Errorf("%s: Prüfsumme %q bei Art %s", f.Name, f.SHA256, f.Kind)
		}
	}
	want := map[string]Kind{
		"rechnung.pdf": KindReceipt, "quittung.png": KindReceipt, "xrechnung.xml": KindEInvoice,
		"ubl.xml": KindEInvoice, "auszug.xml": KindStatement, "vormerkposten.xml": KindUnsupported,
		"notiz.txt": KindUnsupported, "falsch.pdf": KindUnsupported,
	}
	if len(kinds) != len(want) {
		t.Errorf("Dateien: %v", kinds)
	}
	for name, kind := range want {
		if kinds[name] != kind {
			t.Errorf("%s: %q, erwartet %q", name, kinds[name], kind)
		}
	}
	if files[0].Name != "quittung.png" {
		t.Errorf("die älteste zuerst: %s", files[0].Name)
	}
	if got := AgeDays(Oldest(files), now); got != 20 {
		t.Errorf("Alter der ältesten Datei: %d Tage", got)
	}
	if AgeDays(Oldest(nil), now) != 0 {
		t.Error("ein leerer Eingang hat kein Alter")
	}
	if n, oldest, err := Count(dir); err != nil || n != len(files) || !oldest.Equal(Oldest(files)) {
		t.Errorf("Count: %d %v %v", n, oldest, err)
	}

	missing, err := List(filepath.Join(t.TempDir(), "fehlt"))
	if err != nil || len(missing) != 0 || missing == nil {
		t.Errorf("fehlender Eingang: %v %v", missing, err)
	}
}

// Abgelegtes wandert nach Übernommen/<JJJJ-MM>; ein vorhandener Name bekommt
// ein Suffix.
func TestSettleMovesIntoTheMonth(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Eingang")
	day := time.Date(2026, 5, 31, 23, 0, 0, 0, time.Local)
	first := write(t, filepath.Join(dir, "Rechnung.pdf"), pdfData, time.Time{})
	moved, err := Settle(dir, first, day)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, DoneDir, "2026-05", "Rechnung.pdf"); moved != want {
		t.Errorf("verschoben nach %s, erwartet %s", moved, want)
	}
	if _, err := os.Stat(first); !errors.Is(err, os.ErrNotExist) {
		t.Error("die Datei liegt noch im Eingang")
	}

	second := write(t, filepath.Join(dir, "Unterordner", "Rechnung.pdf"), pdfData+"2", time.Time{})
	moved, err = Settle(dir, second, day)
	if err != nil || filepath.Base(moved) != "Rechnung (2).pdf" {
		t.Errorf("Namenskonflikt: %s %v", moved, err)
	}
	third := write(t, filepath.Join(dir, "Rechnung.pdf"), pdfData+"3", time.Time{})
	if moved, err = Settle(dir, third, day); err != nil || filepath.Base(moved) != "Rechnung (3).pdf" {
		t.Errorf("zweiter Namenskonflikt: %s %v", moved, err)
	}

	// Was schon in Übernommen/ oder außerhalb liegt, bleibt, wo es ist.
	if _, err := Settle(dir, moved, day); !errors.Is(err, ErrOutside) {
		t.Errorf("aus Übernommen/: %v", err)
	}
	outside := write(t, filepath.Join(t.TempDir(), "Downloads", "x.pdf"), pdfData, time.Time{})
	if _, err := Settle(dir, outside, day); !errors.Is(err, ErrOutside) {
		t.Errorf("außerhalb: %v", err)
	}
	if _, err := Settle(dir, dir+"/../x.pdf", day); !errors.Is(err, ErrOutside) {
		t.Errorf("Punkte hinaus: %v", err)
	}
	if _, err := Settle(dir, filepath.Join(dir, "fehlt.pdf"), day); err == nil {
		t.Error("eine fehlende Datei lässt sich nicht verschieben")
	}
}

// Scheitert das Verschieben, bleibt die Datei liegen und der Fehler kommt an.
func TestSettleReportsAFailedMove(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Eingang")
	path := write(t, filepath.Join(dir, "Rechnung.pdf"), pdfData, time.Time{})
	// Eine Datei namens Übernommen versperrt den Ordner, auch für root.
	write(t, filepath.Join(dir, DoneDir), "keine Ablage", time.Time{})
	if _, err := Settle(dir, path, time.Now()); err == nil {
		t.Fatal("Verschieben ohne Schreibrecht ist gelungen")
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("die Datei ist weg: %v", err)
	}
}

// Hineingezogene Dateien kommen als Kopie in den Eingang, nur Belegformate.
func TestAddCopiesSupportedFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Eingang")
	src := t.TempDir()
	pdf := write(t, filepath.Join(src, "Rechnung.pdf"), pdfData, time.Time{})
	xml := write(t, filepath.Join(src, "auszug.xml"), camtXML, time.Time{})
	txt := write(t, filepath.Join(src, "notiz.txt"), "Notiz", time.Time{})
	fake := write(t, filepath.Join(src, "fake.pdf"), "MZ", time.Time{})
	hidden := write(t, filepath.Join(src, ".geheim.pdf"), pdfData, time.Time{})

	added, skipped, err := Add(dir, []string{pdf, xml, txt, fake, hidden, src, "relativ.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(added, []string{"Rechnung.pdf", "auszug.xml"}) {
		t.Errorf("übernommen: %v", added)
	}
	if len(skipped) != 5 {
		t.Errorf("übergangen: %+v", skipped)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "Rechnung.pdf")); err != nil || string(data) != pdfData {
		t.Errorf("Kopie: %q %v", data, err)
	}
	if _, err := os.Stat(pdf); err != nil {
		t.Error("das Original ist weg")
	}

	again, _, err := Add(dir, []string{pdf})
	if err != nil || !slices.Equal(again, []string{"Rechnung (2).pdf"}) {
		t.Errorf("zweites Mal: %v %v", again, err)
	}
	inside, skipped, _ := Add(dir, []string{filepath.Join(dir, "Rechnung.pdf")})
	if len(inside) != 0 || len(skipped) != 1 {
		t.Errorf("aus dem Eingang selbst: %v %+v", inside, skipped)
	}
	files, _ := List(dir)
	if len(files) != 3 {
		t.Errorf("keine Zwischendatei bleibt sichtbar: %+v", files)
	}
}
