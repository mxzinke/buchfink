package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/inbox"
)

type eingangAntwort struct {
	Eingang          string `json:"eingang"`
	AeltesteVorTagen int    `json:"aelteste_vor_tagen"`
	Dateien          []struct {
		Pfad            string `json:"pfad"`
		Name            string `json:"name"`
		Groesse         int64  `json:"groesse"`
		Art             string `json:"art"`
		BereitsAbgelegt *uint  `json:"bereits_abgelegt"`
	} `json:"dateien"`
	NaechsteSeite *string `json:"naechste_seite"`
	Gesamt        int     `json:"gesamt"`
}

// ageFile setzt die Änderungszeit einer Datei zurück.
func ageFile(t *testing.T, path string, days int) {
	t.Helper()
	when := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatal(err)
	}
}

// Der Eingang listet seine Dateien mit Art, seitenweise; status und aufgaben
// nennen die Zahl, sobald dort etwas liegt.
func TestInboxListsFilesWithKind(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	a := h.agent("")

	var empty eingangAntwort
	a.ok("eingang", nil, &empty)
	if empty.Gesamt != 0 || len(empty.Dateien) != 0 || empty.Eingang != filepath.Join(h.tenant.DataDir, "Eingang") {
		t.Fatalf("leerer Eingang: %+v", empty)
	}
	var tasks aufgabenOut
	a.ok("aufgaben", nil, &tasks)
	for _, task := range tasks.Offen {
		if task.Werkzeug == "eingang" {
			t.Errorf("Aufgabe ohne Dateien: %+v", task)
		}
	}

	old := h.inboxFile("quittung.pdf", pdf("Quittung"))
	ageFile(t, old, 20)
	h.inboxFile("RE-EIN-1.xml", []byte(h.eInvoiceXML(supplier, "RE-EIN-1", "2026-05-04", 10000)))
	h.inboxFile("auszug.xml", []byte(camt(ownIBAN, "1000.00", "2026-04-01", "1000.00", "2026-04-30")))
	h.inboxFile("notiz.txt", []byte("Notiz"))
	h.inboxFile(".versteckt.pdf", pdf("versteckt"))
	h.inboxFile(filepath.Join(inbox.DoneDir, "2026-04", "alt.pdf"), pdf("alt"))

	var page eingangAntwort
	a.ok("eingang", map[string]any{"limit": 3}, &page)
	if page.Gesamt != 4 || len(page.Dateien) != 3 || page.NaechsteSeite == nil || page.AeltesteVorTagen != 20 {
		t.Fatalf("erste Seite: %+v", page)
	}
	if page.Dateien[0].Name != "quittung.pdf" || page.Dateien[0].Pfad != old || page.Dateien[0].Art != "beleg" {
		t.Errorf("die älteste zuerst: %+v", page.Dateien[0])
	}
	var rest eingangAntwort
	a.ok("eingang", map[string]any{"limit": 3, "cursor": *page.NaechsteSeite}, &rest)
	if len(rest.Dateien) != 1 || rest.NaechsteSeite != nil {
		t.Fatalf("zweite Seite: %+v", rest)
	}
	kinds := map[string]string{}
	for _, f := range append(page.Dateien, rest.Dateien...) {
		kinds[f.Name] = f.Art
		if f.BereitsAbgelegt != nil {
			t.Errorf("%s gilt als abgelegt", f.Name)
		}
	}
	want := map[string]string{"quittung.pdf": "beleg", "RE-EIN-1.xml": "e_rechnung", "auszug.xml": "kontoauszug",
		"notiz.txt": "nicht_unterstuetzt"}
	for name, kind := range want {
		if kinds[name] != kind {
			t.Errorf("%s: %q, erwartet %q", name, kinds[name], kind)
		}
	}

	st := statusResult(t, callTool(t, a.cs, "status", nil))
	if st.Eingang == nil || st.Eingang.Dateien != 4 || st.Eingang.Hinweis != "4 Dateien im Eingang" || st.Eingang.AeltesteVorTagen != 20 {
		t.Errorf("status: %+v", st.Eingang)
	}
	a.ok("aufgaben", nil, &tasks)
	found := false
	for _, task := range tasks.Offen {
		if task.Werkzeug == "eingang" {
			found = task.Schluessel == "eingang" && task.Anzahl == 4 && task.Titel == "4 Dateien im Eingang"
		}
	}
	if !found {
		t.Errorf("aufgaben nennt den Eingang nicht: %+v", tasks.Offen)
	}
	a.fail("eingang", map[string]any{"cursor": "kaputt"}, CodeInvalidInput)
}

// Ist der Eingang nicht lesbar, etwa weil das System der App den Zugriff
// verweigert, liefert status trotzdem Mandant und Bereitschaft und nennt den
// Grund; aufgaben nennt ihn als Aufgabe für den Menschen.
func TestStatusSurvivesAnUnreadableInbox(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	blocker := filepath.Join(t.TempDir(), "datei")
	if err := os.WriteFile(blocker, []byte("kein Ordner"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Unter einer Datei lässt sich kein Ordner lesen, auch nicht als root.
	unreadable := filepath.Join(blocker, "Eingang")
	h.ui(func(ctx context.Context, s *core.Services) error {
		_, err := s.Settings.SetInboxFolder(ctx, unreadable)
		return err
	})
	a := h.agent("")
	st := statusResult(t, callTool(t, a.cs, "status", nil))
	if !st.Bereit || st.Mandant == nil || st.Mandant.ID == "" {
		t.Fatalf("status ohne Mandant: %+v", st)
	}
	if st.Eingang == nil || st.Eingang.Pfad != unreadable || !strings.Contains(st.Eingang.Fehler, "nicht lesbar") ||
		!strings.Contains(st.Hinweis, "nicht lesbar") {
		t.Errorf("Grund fehlt: %+v %q", st.Eingang, st.Hinweis)
	}
	var tasks aufgabenOut
	a.ok("aufgaben", nil, &tasks)
	found := false
	for _, task := range tasks.Offen {
		found = found || (task.Schluessel == "eingang" && task.Werkzeug == "nur_app")
	}
	if !found {
		t.Errorf("aufgaben nennt den unlesbaren Eingang nicht: %+v", tasks.Offen)
	}

	// Ohne Leserecht auf den Ordner gilt dasselbe; root liest trotzdem.
	if os.Geteuid() == 0 {
		return
	}
	locked := filepath.Join(t.TempDir(), "Eingang")
	if err := os.Mkdir(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	h.ui(func(ctx context.Context, s *core.Services) error {
		_, err := s.Settings.SetInboxFolder(ctx, locked)
		return err
	})
	if st := statusResult(t, callTool(t, a.cs, "status", nil)); !st.Bereit || st.Eingang == nil || st.Eingang.Fehler == "" {
		t.Errorf("Eingang ohne Leserecht: %+v", st.Eingang)
	}
}

// Nach der Ablage geht die Datei aus dem Eingang nach Übernommen/, auch eine
// exakte Dublette; eine Datei aus einem Leseordner bleibt liegen.
func TestFilingSettlesTheInbox(t *testing.T) {
	h := newHarness(t)
	h.completeCompany()
	read := t.TempDir()
	h.ui(func(ctx context.Context, s *core.Services) error {
		_, err := s.Settings.SaveAgentSettings(ctx, domain.AgentSettings{ExtraReadFolders: []string{read}})
		return err
	})
	a := h.agent("")
	header := map[string]any{"eingang_am": "2026-05-04", "belegdatum": "2026-05-04", "aussteller": "Papier AG", "brutto": "11.90"}
	file := func(path string) ablageOut {
		t.Helper()
		in := map[string]any{"dateien": []any{map[string]any{"pfad": path}}}
		for k, v := range header {
			in[k] = v
		}
		var out ablageOut
		a.ok("beleg_ablegen", a.args(in), &out)
		return out
	}

	content := pdf("Papier")
	fromRead := filepath.Join(read, "papier.pdf")
	if err := os.WriteFile(fromRead, content, 0o600); err != nil {
		t.Fatal(err)
	}
	first := file(fromRead)
	if len(first.Verschoben) != 0 {
		t.Errorf("aus dem Leseordner verschoben: %+v", first.Verschoben)
	}
	if _, err := os.Stat(fromRead); err != nil {
		t.Errorf("die Datei im Leseordner ist weg: %v", err)
	}

	copyInInbox := h.inboxFile("papier-kopie.pdf", content)
	var listed eingangAntwort
	a.ok("eingang", nil, &listed)
	if len(listed.Dateien) != 1 || listed.Dateien[0].BereitsAbgelegt == nil || *listed.Dateien[0].BereitsAbgelegt != first.Beleg.ID {
		t.Fatalf("bereits abgelegt: %+v", listed)
	}
	dup := file(copyInInbox)
	month := time.Now().Format("2006-01")
	if !dup.BereitsVorhanden || dup.Beleg.ID != first.Beleg.ID || len(dup.Verschoben) != 1 ||
		dup.Verschoben[0].Nach != filepath.Join(h.tenant.DataDir, "Eingang", inbox.DoneDir, month, "papier-kopie.pdf") {
		t.Fatalf("Dublette: %+v", dup)
	}
	if _, err := os.Stat(dup.Verschoben[0].Nach); err != nil {
		t.Errorf("verschobene Datei: %v", err)
	}

	// Ein Namenskonflikt im Monat bekommt ein Suffix.
	second := file(h.inboxFile("papier-kopie.pdf", pdf("Papier zwei")))
	if len(second.Verschoben) != 1 || filepath.Base(second.Verschoben[0].Nach) != "papier-kopie (2).pdf" {
		t.Errorf("Namenskonflikt: %+v", second.Verschoben)
	}
	a.ok("eingang", nil, &listed)
	if listed.Gesamt != 0 {
		t.Errorf("der Eingang ist nicht leer: %+v", listed)
	}

	// Scheitert das Verschieben, bleibt die Ablage gültig und ein Hinweis steht da.
	blocked := h.inboxFile("gesperrt.pdf", pdf("Gesperrt"))
	done := filepath.Join(h.tenant.DataDir, "Eingang", inbox.DoneDir)
	if err := os.RemoveAll(done); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(done, []byte("versperrt"), 0o600); err != nil {
		t.Fatal(err)
	}
	failed := file(blocked)
	if failed.Beleg.ID == 0 || len(failed.Verschoben) != 0 || len(failed.Hinweise) == 0 {
		t.Fatalf("gescheitertes Verschieben: %+v", failed)
	}
	if _, err := os.Stat(blocked); err != nil {
		t.Errorf("die Datei ist weg: %v", err)
	}
}
