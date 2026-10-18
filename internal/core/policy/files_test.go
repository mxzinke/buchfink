package policy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/inbox"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/service"
)

func mkdirs(t *testing.T, dirs ...string) {
	t.Helper()
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func writeFile(t *testing.T, path, content string) string {
	t.Helper()
	mkdirs(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func hardLink(t *testing.T, target, link string) string {
	t.Helper()
	if err := os.Link(target, link); err != nil {
		t.Fatal(err)
	}
	return link
}

const pdfContent = "%PDF-1.7\n..."

// Gelesen wird nur aus dem Eingang und den Leseordnern; aus Übernommen/, das
// zum Eingang gehört, auch.
func TestReadFileStaysInsideInboxAndReadFolders(t *testing.T) {
	base := t.TempDir()
	data := filepath.Join(base, "daten")
	in := filepath.Join(data, InboxDir)
	downloads := filepath.Join(base, "Downloads")
	outside := filepath.Join(base, "privat")
	mkdirs(t, in, downloads, outside)
	writeFile(t, filepath.Join(data, "buchfink.sqlite"), "SQLite format 3")
	f := FoldersFrom(&domain.AgentSettings{ExtraReadFolders: []string{downloads}}, "", data, Protected{})
	if f.Inbox != in {
		t.Fatalf("voreingestellter Eingang: %s", f.Inbox)
	}

	if got, err := ReadFile(writeFile(t, filepath.Join(in, "rechnung.pdf"), pdfContent), f); err != nil || got.Kind != KindPDF {
		t.Fatalf("Eingang: %v %+v", err, got)
	}
	if _, err := ReadFile(writeFile(t, filepath.Join(downloads, "e-rechnung.xml"), "\xef\xbb\xbf <?xml version=\"1.0\"?><x/>"), f); err != nil {
		t.Errorf("XML im Leseordner: %v", err)
	}
	if _, err := ReadFile(writeFile(t, filepath.Join(in, inbox.DoneDir, "2026-05", "alt.pdf"), pdfContent), f); err != nil {
		t.Errorf("aus Übernommen/: %v", err)
	}

	secret := writeFile(t, filepath.Join(outside, "ausweis.pdf"), pdfContent)
	link := filepath.Join(in, "rechnung-link.pdf")
	if err := os.Symlink(secret, link); err != nil {
		t.Fatal(err)
	}
	dbLink := filepath.Join(in, "db.xml")
	if err := os.Symlink(filepath.Join(data, "buchfink.sqlite"), dbLink); err != nil {
		t.Fatal(err)
	}
	refused := []struct {
		name, path string
		notAllowed bool
	}{
		{"außerhalb", secret, true},
		{"Symlink hinaus", link, true},
		{"Punkte hinaus", in + "/../../privat/ausweis.pdf", true},
		{"relativ", InboxDir + "/rechnung.pdf", true},
		{"Datenbank über Symlink", dbLink, true},
		{"Datenordner", filepath.Join(data, "buchfink.sqlite"), true},
		{"versteckt", writeFile(t, filepath.Join(in, ".rechnung.pdf"), pdfContent), true},
		{"falsche Endung", writeFile(t, filepath.Join(in, "skript.sh"), "#!/bin/sh"), false},
		{"falscher Inhalt", writeFile(t, filepath.Join(in, "fake.pdf"), "MZ\x90"), false},
		{"leer", writeFile(t, filepath.Join(in, "leer.pdf"), ""), false},
		{"fehlt", filepath.Join(in, "nichtda.pdf"), false},
		{"fehlt außerhalb", filepath.Join(outside, "nichtda.pdf"), true},
		{"Ordner selbst", downloads, true},
		{"Netzwerkpfad", "//server/freigabe/rechnung.pdf", true},
		{"Netzwerkpfad Windows", `\\server\freigabe\rechnung.pdf`, true},
		{"Hardlink hinaus", hardLink(t, secret, filepath.Join(in, "hart.pdf")), true},
		{"Hardlink in den Datenordner", hardLink(t, writeFile(t, filepath.Join(data, "intern.pdf"), pdfContent),
			filepath.Join(in, "hart-daten.pdf")), true},
	}
	for _, tc := range refused {
		_, err := ReadFile(tc.path, f)
		var fe *FileError
		if !errors.As(err, &fe) {
			t.Errorf("%s: erwartet Abweisung, bekam %v", tc.name, err)
			continue
		}
		if fe.NotAllowed != tc.notAllowed {
			t.Errorf("%s: NotAllowed=%v (%v)", tc.name, fe.NotAllowed, err)
		}
	}
	// Nur XML, wo ein Kontoauszug erwartet wird.
	if _, err := ReadFile(filepath.Join(in, "rechnung.pdf"), f, KindXML); err == nil {
		t.Error("ein PDF ist kein Kontoauszug")
	}
	// Ohne Leseordner liest ein Agent nur den Eingang.
	if _, err := ReadFile(filepath.Join(downloads, "e-rechnung.xml"), FoldersFrom(nil, "", data, Protected{})); err == nil {
		t.Error("ohne Leseordner gelesen")
	}
}

// Ein eigener Eingang außerhalb des Datenordners, etwa in einem Sync-Ordner,
// macht den voreingestellten im Datenordner zu: dort liest dann niemand mehr.
func TestCustomInboxReplacesTheDefault(t *testing.T) {
	base := t.TempDir()
	data := filepath.Join(base, "daten")
	custom := filepath.Join(base, "Dropbox", "Scans")
	mkdirs(t, filepath.Join(data, InboxDir), custom)
	f := FoldersFrom(nil, custom, data, Protected{})
	if _, err := ReadFile(writeFile(t, filepath.Join(custom, "scan.pdf"), pdfContent), f); err != nil {
		t.Errorf("eigener Eingang: %v", err)
	}
	var fe *FileError
	if _, err := ReadFile(writeFile(t, filepath.Join(data, InboxDir, "alt.pdf"), pdfContent), f); !errors.As(err, &fe) || !fe.NotAllowed {
		t.Errorf("voreingestellter Eingang neben einem eigenen: %v", err)
	}
	// Ein Eingang, der den Datenordner selbst nennt, öffnet ihn nicht.
	whole := FoldersFrom(nil, data, data, Protected{})
	if _, err := ReadFile(writeFile(t, filepath.Join(data, "intern.pdf"), pdfContent), whole); !errors.As(err, &fe) || !fe.NotAllowed {
		t.Errorf("Datenordner als Eingang: %v", err)
	}
}

// Zwei Mandanten unter einem gemeinsamen Ordner: der Leseordner des einen
// umschließt den Datenordner des anderen. Gelesen wird trotzdem nur aus dem
// eigenen Eingang und aus Ordnern, die keinem Mandanten gehören.
func TestReadFolderNeverReachesAnotherTenant(t *testing.T) {
	parent := t.TempDir()
	own := filepath.Join(parent, "Pfennig GmbH")
	other := filepath.Join(parent, "Fremd GmbH")
	backup := filepath.Join(parent, "Sicherung")
	mkdirs(t, filepath.Join(own, InboxDir), filepath.Join(other, InboxDir), backup, filepath.Join(parent, "Scans"))
	xml := "<?xml version=\"1.0\"?><x/>"
	for _, p := range []string{filepath.Join(other, InboxDir, "geheim.xml"), filepath.Join(other, "beleg.xml"),
		filepath.Join(backup, "sicherung.xml"), filepath.Join(own, InboxDir, "eigen.xml"), filepath.Join(parent, "Scans", "scan.xml")} {
		writeFile(t, p, xml)
	}
	cfg := domain.AppConfig{BackupDir: backup, Tenants: []domain.TenantConfig{
		{ID: "a", DataDir: own, BackupDir: backup}, {ID: "b", DataDir: other},
	}}
	f := FoldersFrom(&domain.AgentSettings{ExtraReadFolders: []string{parent}}, "", own, ProtectedDirs(cfg, own))
	for _, p := range []string{filepath.Join(other, InboxDir, "geheim.xml"), filepath.Join(other, "beleg.xml"),
		filepath.Join(backup, "sicherung.xml")} {
		var fe *FileError
		if _, err := ReadFile(p, f); !errors.As(err, &fe) || !fe.NotAllowed {
			t.Errorf("%s: %v", p, err)
		}
	}
	for _, p := range []string{filepath.Join(own, InboxDir, "eigen.xml"), filepath.Join(parent, "Scans", "scan.xml")} {
		if _, err := ReadFile(p, f); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
}

// Eingang und Leseordner lassen sich nur einstellen, wenn sie keinen
// geschützten Ordner berühren.
func TestFoldersAreCheckedWhenSet(t *testing.T) {
	parent := t.TempDir()
	own := filepath.Join(parent, "Pfennig GmbH")
	other := filepath.Join(parent, "Fremd GmbH")
	backup := filepath.Join(parent, "Sicherung")
	scans := filepath.Join(parent, "Scans")
	mkdirs(t, own, other, backup, scans)
	cfg := domain.AppConfig{BackupDir: backup, Tenants: []domain.TenantConfig{
		{ID: "a", DataDir: own, BackupDir: backup}, {ID: "b", DataDir: other},
	}}
	protected := ProtectedDirs(cfg, own)
	for _, folder := range []string{parent, own, filepath.Join(own, "receipts"), other, filepath.Join(other, InboxDir),
		backup, filepath.Join(backup, "Eingang"), "relativ", `\\server\freigabe`} {
		if err := CheckFolders(folder, nil, own, protected); err == nil {
			t.Errorf("Eingang %s angenommen", folder)
		}
		if err := CheckFolders("", []string{folder}, own, protected); err == nil {
			t.Errorf("Leseordner %s angenommen", folder)
		}
	}
	if err := CheckFolders(filepath.Join(own, InboxDir), []string{scans}, own, protected); err != nil {
		t.Errorf("voreingestellter Eingang und ein fremder Ordner: %v", err)
	}
	if err := CheckFolders("", []string{filepath.Join(own, InboxDir)}, own, protected); err == nil {
		t.Error("der Eingang im Datenordner als Leseordner angenommen")
	}
	if err := CheckFolders(filepath.Join(scans, "Eingang"), nil, own, protected); err != nil {
		t.Errorf("eigener Eingang außerhalb: %v", err)
	}
	if err := CheckFolders("", nil, own, protected); err != nil {
		t.Errorf("Voreinstellung: %v", err)
	}
}

func TestDefaultFolders(t *testing.T) {
	data := t.TempDir()
	f := FoldersFrom(nil, "", data, Protected{})
	if f.Inbox != filepath.Join(data, InboxDir) || len(f.Read) != 0 || f.Read == nil {
		t.Errorf("Voreinstellung: Eingang %s, Leseordner %v", f.Inbox, f.Read)
	}
	if f.Export != filepath.Join(data, ExportDir) {
		t.Errorf("Voreinstellung Exportordner: %s", f.Export)
	}
	f.EnsureInbox()
	if _, err := os.Stat(f.Inbox); err != nil {
		t.Errorf("voreingestellter Eingang angelegt: %v", err)
	}
	path, err := f.ExportPath("e-bilanz-2026.xbrl")
	if err != nil || filepath.Dir(path) != f.Export {
		t.Fatalf("Exportpfad: %s %v", path, err)
	}
	if _, err := f.ExportPath("../hinaus.xml"); err == nil {
		t.Error("ein Dateiname mit Pfad darf nicht hinaus")
	}
	custom := FoldersFrom(&domain.AgentSettings{ExportFolder: filepath.Join(data, "fehlt")}, "", data, Protected{})
	if _, err := custom.ExportPath("x.json"); err == nil {
		t.Error("ein eingestellter Exportordner wird nicht angelegt")
	}
}

// openTenant legt einen Mandanten an und liefert ihn offen.
func openTenant(t *testing.T) (*core.Tenant, string) {
	t.Helper()
	base := t.TempDir()
	h, err := core.NewHost(context.Background(), core.HostOptions{
		Config:  repository.NewAppConfigRepository(filepath.Join(base, "config")),
		BaseDir: base, Network: core.OfflineNetwork(),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Close(context.Background()) })
	ui := actor.WithChannel(context.Background(), actor.ChannelUI)
	tc, err := h.CreateTenant(ui, "Pfennig GmbH", filepath.Join(base, "daten"), domain.CompanySettings{CompanyName: "Pfennig GmbH", FiscalYear: 2026})
	if err != nil {
		t.Fatal(err)
	}
	tenant, release, err := h.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	release()
	return tenant, tc.DataDir
}

// Nach der Ablage geht, was im Eingang lag und im Archiv steht, nach
// Übernommen/<JJJJ-MM>; Leseordner und Geändertes bleiben, ein gescheitertes
// Verschieben steht als Hinweis da.
func TestSettleInboxMovesWhatIsArchived(t *testing.T) {
	tenant, data := openTenant(t)
	ui := actor.WithChannel(context.Background(), actor.ChannelUI)
	read := t.TempDir()
	f := FoldersFrom(&domain.AgentSettings{ExtraReadFolders: []string{read}}, "", data, Protected{})
	in := f.Inbox
	day := time.Date(2026, 5, 31, 12, 0, 0, 0, time.Local)

	archive := func(content string) {
		t.Helper()
		if err := tenant.Write(ui, func(ctx context.Context, s *core.Services) error {
			_, err := s.Receipts.File(ctx, service.FileReceiptRequest{Direction: domain.DirectionIncoming,
				ReceivedAt: "2026-05-04", DocumentDate: "2026-05-04", IssuerName: "Papier AG", GrossAmount: 1190,
				Files: []service.NewFile{{Role: domain.ReceiptRoleOriginal, FileName: "x.pdf", Content: []byte(content)}}})
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	filed := writeFile(t, filepath.Join(in, "Rechnung.pdf"), pdfContent+"1")
	archive(pdfContent + "1")
	// Eine exakte Dublette unter anderem Namen ist ebenso abgelegt.
	duplicate := writeFile(t, filepath.Join(in, "Scans", "Kopie.pdf"), pdfContent+"1")
	// Derselbe Name ist im Monat schon vergeben.
	writeFile(t, filepath.Join(in, inbox.DoneDir, "2026-05", "Rechnung.pdf"), "älter")
	changed := writeFile(t, filepath.Join(in, "geaendert.pdf"), pdfContent+"anders")
	inRead := writeFile(t, filepath.Join(read, "Lesen.pdf"), pdfContent+"1")

	moved, hints := SettleInbox(ui, tenant, f, []string{filed, duplicate, changed, inRead, filed}, day)
	if len(moved) != 2 {
		t.Fatalf("verschoben: %+v %v", moved, hints)
	}
	if moved[0].From != filed || moved[0].To != filepath.Join(in, inbox.DoneDir, "2026-05", "Rechnung (2).pdf") {
		t.Errorf("Namenskonflikt: %+v", moved[0])
	}
	if moved[1].To != filepath.Join(in, inbox.DoneDir, "2026-05", "Kopie.pdf") {
		t.Errorf("Dublette: %+v", moved[1])
	}
	if len(hints) != 1 || !strings.Contains(hints[0], "geaendert.pdf") {
		t.Errorf("Hinweise: %v", hints)
	}
	for _, stays := range []string{changed, inRead} {
		if _, err := os.Stat(stays); err != nil {
			t.Errorf("%s ist weg: %v", stays, err)
		}
	}

	// Scheitert das Verschieben, sagt es ein Hinweis; die Datei bleibt.
	blocked := writeFile(t, filepath.Join(in, "Blockiert.pdf"), pdfContent+"2")
	archive(pdfContent + "2")
	next := day.AddDate(0, 1, 0)
	writeFile(t, filepath.Join(in, inbox.DoneDir, next.Format("2006-01")), "keine Ablage")
	moved, hints = SettleInbox(ui, tenant, f, []string{blocked}, next)
	if len(moved) != 0 || len(hints) != 1 || !strings.Contains(hints[0], "trotzdem") {
		t.Errorf("gescheitertes Verschieben: %+v %v", moved, hints)
	}
	if _, err := os.Stat(blocked); err != nil {
		t.Errorf("die Datei ist weg: %v", err)
	}
}

func sha(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// Das Archiv nennt zu jeder Prüfsumme den Beleg, gleich in welcher Rolle die
// Datei an ihm hängt.
func TestReceiptIDsByFileHash(t *testing.T) {
	tenant, _ := openTenant(t)
	ui := actor.WithChannel(context.Background(), actor.ChannelUI)
	var id uint
	if err := tenant.Write(ui, func(ctx context.Context, s *core.Services) error {
		r, err := s.Receipts.File(ctx, service.FileReceiptRequest{Direction: domain.DirectionIncoming,
			ReceivedAt: "2026-05-04", DocumentDate: "2026-05-04", IssuerName: "Papier AG", GrossAmount: 1190,
			Files: []service.NewFile{{Role: domain.ReceiptRoleOriginal, FileName: "x.pdf", Content: []byte(pdfContent + "o")},
				{Role: domain.ReceiptRoleAttachment, FileName: "y.pdf", Content: []byte(pdfContent + "a")}}})
		if r != nil {
			id = r.ID
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := tenant.Read(ui, func(ctx context.Context, s *core.Services) error {
		got, err := s.Receipts.ReceiptIDsByFileHash(ctx, []string{sha(pdfContent + "o"), sha(pdfContent + "a"), sha("fremd")})
		if err == nil && (len(got) != 2 || got[sha(pdfContent+"o")] != id || got[sha(pdfContent+"a")] != id) {
			t.Errorf("Zuordnung: %v", got)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

// Ein Eingang, der die Prüfung bestanden hat und später gegen einen Symlink in
// den Datenordner eines anderen Mandanten getauscht wird, öffnet diesen nicht:
// Lesen, Listen und Aufräumen weisen ihn ab. Eine Sicherung, die den Eingang
// umschließt, sperrt ihn dagegen nicht.
func TestSwappedInboxNeverOpensAnotherTenant(t *testing.T) {
	base := t.TempDir()
	own := filepath.Join(base, "d", "A")
	other := filepath.Join(base, "d", "B")
	scans := filepath.Join(base, "s", "Scans")
	mkdirs(t, own, filepath.Join(other, "Export"), scans)
	writeFile(t, filepath.Join(other, "Export", "ebilanz.xml"), "<?xml version=\"1.0\"?><x/>")
	cfg := domain.AppConfig{Tenants: []domain.TenantConfig{{ID: "a", DataDir: own}, {ID: "b", DataDir: other}}}
	protected := ProtectedDirs(cfg, own)
	if err := CheckFolders(scans, nil, own, protected); err != nil {
		t.Fatalf("Eingang außerhalb: %v", err)
	}
	if err := os.Remove(scans); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(other, "Export"), scans); err != nil {
		t.Fatal(err)
	}
	f := FoldersFrom(nil, scans, own, protected)
	var fe *FileError
	if _, err := ReadFile(filepath.Join(scans, "ebilanz.xml"), f); !errors.As(err, &fe) || !fe.NotAllowed {
		t.Errorf("Datei aus dem fremden Datenordner gelesen: %v", err)
	}
	if files, err := f.InboxFiles(); err == nil {
		t.Errorf("fremder Datenordner gelistet: %+v", files)
	}
	if err := f.CheckInbox(); err == nil {
		t.Error("ein Eingang im fremden Datenordner gilt als lesbar")
	}

	// Eine Sicherung über dem Eingang: der Eingang bleibt lesbar.
	dropbox := filepath.Join(base, "Dropbox")
	in := filepath.Join(dropbox, "Scans")
	writeFile(t, filepath.Join(in, "r.pdf"), pdfContent)
	withBackup := FoldersFrom(nil, in, own, Protected{Backups: []string{dropbox}})
	if _, err := ReadFile(filepath.Join(in, "r.pdf"), withBackup); err != nil {
		t.Errorf("Eingang unter einer Sicherung: %v", err)
	}
}

// Die Liste des Eingangs lässt gesperrte Ordner darin aus: eine Sicherung im
// Eingang zählt nicht als Datei, der Datenordner eines anderen Mandanten wird
// nicht gezeigt. Ein Eingang über einen Symlink zeigt seine Dateien.
func TestInboxListSkipsGuardedFolders(t *testing.T) {
	base := t.TempDir()
	own := filepath.Join(base, "A")
	in := filepath.Join(base, "Dropbox", "Buchfink")
	backup := filepath.Join(in, "Sicherung")
	other := filepath.Join(in, "B")
	writeFile(t, filepath.Join(in, "r.pdf"), pdfContent)
	writeFile(t, filepath.Join(backup, "pfennig-2026-10-18.zip"), "PK")
	writeFile(t, filepath.Join(other, InboxDir, "fremd.pdf"), pdfContent)
	writeFile(t, filepath.Join(other, "Export", "ebilanz.xml"), "<x/>")
	mkdirs(t, own)
	f := FoldersFrom(nil, in, own, Protected{DataDirs: []string{other}, Backups: []string{backup}})
	files, err := f.InboxFiles()
	if err != nil || len(files) != 1 || files[0].Name != "r.pdf" {
		t.Errorf("Eingang: %+v %v", files, err)
	}
	if n, _, err := f.InboxCount(); err != nil || n != 1 {
		t.Errorf("Zahl: %d %v", n, err)
	}

	link := filepath.Join(base, "Eingang-Link")
	if err := os.Symlink(in, link); err != nil {
		t.Fatal(err)
	}
	linked := FoldersFrom(nil, link, own, Protected{DataDirs: []string{other}, Backups: []string{backup}})
	if files, err := linked.InboxFiles(); err != nil || len(files) != 1 || files[0].Path != filepath.Join(link, "r.pdf") {
		t.Errorf("Eingang über Symlink: %+v %v", files, err)
	}
}

// Ein Leseordner darf den Eingang weder enthalten noch in ihm liegen: aus dem
// Eingang verschiebt Buchfink, aus einem Leseordner nicht. Geprüft wird der
// Pfad, wie er gespeichert wird; Leerzeichen am Ende gehören zum Namen.
func TestReadFoldersStayOutOfTheInbox(t *testing.T) {
	base := t.TempDir()
	own := filepath.Join(base, "daten")
	scans := filepath.Join(base, "Nextcloud", "Scans")
	mkdirs(t, own, scans)
	for _, read := range []string{filepath.Join(scans, "Verträge"), filepath.Dir(scans), scans} {
		if err := CheckFolders(scans, []string{read}, own, Protected{}); err == nil {
			t.Errorf("Leseordner %s neben dem Eingang %s angenommen", read, scans)
		}
	}
	if err := CheckFolders(scans, []string{filepath.Join(base, "Verträge")}, own, Protected{}); err != nil {
		t.Errorf("Leseordner neben dem Eingang: %v", err)
	}
	// Ohne eigenen Eingang gilt der voreingestellte im Datenordner.
	if err := CheckFolders(" ", []string{filepath.Join(base, "Verträge")}, own, Protected{}); err != nil {
		t.Errorf("Voreinstellung: %v", err)
	}
	if err := CheckFolders(own+" ", nil, own, Protected{}); err != nil {
		t.Errorf("ein Geschwisterordner mit Leerzeichen am Ende ist ein anderer Ordner: %v", err)
	}
	if err := CheckFolders(own+"/", nil, own, Protected{}); err == nil {
		t.Error("der Datenordner mit Schrägstrich angenommen")
	}
}
