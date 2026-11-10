package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestVersionOrder(t *testing.T) {
	for _, c := range []struct {
		found, current string
		newer          bool
	}{
		{"0.4.0", "0.3.0", true},
		{"v0.3.1", "0.3.0", true},
		{"1.0.0", "0.9.9", true},
		{"0.3.0", "0.3.0", false},
		{"0.2.9", "0.3.0", false},
		{"0.4.0", "0.4.0-rc.1", true},
		{"0.4.0-rc.1", "0.4.0", false},
		{"0.4.0-rc.2", "0.4.0-rc.1", true},
		{"0.4.0-rc.10", "0.4.0-rc.9", true},
		{"0.4.0-rc.1", "0.3.9", true},
	} {
		found, ok := parseVersion(c.found)
		current, ok2 := parseVersion(c.current)
		if !ok || !ok2 {
			t.Fatalf("%s oder %s ist keine Fassung", c.found, c.current)
		}
		if got := found.newer(current); got != c.newer {
			t.Errorf("%s neuer als %s: %v, erwartet %v", c.found, c.current, got, c.newer)
		}
	}
	for _, s := range []string{"dev", "dev+8cd1f9a", "dev+8cd1f9a-dirty", "", "1.2", "1.2.x", "v0.2"} {
		if _, ok := parseVersion(s); ok {
			t.Errorf("%q gilt als Fassung", s)
		}
	}
}

// Die Dateinamen eines Release sind die Schnittstelle zwischen release.sh und
// dem Programm, das sich daraus aktualisiert.
func TestAssetNamesMatchTheReleaseScript(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("..", "..", "scripts", "release.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for _, platform := range [][2]string{{"darwin", "arm64"}, {"darwin", "amd64"}, {"windows", "amd64"}, {"linux", "amd64"}} {
		name := assetName(platform[0], platform[1])
		if name == "" || !bytes.Contains(script, []byte(name)) {
			t.Errorf("%s/%s: scripts/release.sh baut keine Datei %q", platform[0], platform[1], name)
		}
	}
	if !bytes.Contains(script, []byte(checksumsName)) {
		t.Errorf("scripts/release.sh schreibt keine %s", checksumsName)
	}
	if assetName("linux", "arm64") != "" {
		t.Error("für linux/arm64 baut das Release keine App")
	}
}

func TestOnlyGitHubIsAllowed(t *testing.T) {
	for host, want := range map[string]bool{
		"github.com": true, "api.github.com": true, "objects.githubusercontent.com": true,
		"release-assets.githubusercontent.com": true, "githubusercontent.com.example.org": false,
		"example.org": false, "github.com.example.org": false, "": false,
	} {
		if got := allowedHost(host); got != want {
			t.Errorf("%q zugelassen: %v, erwartet %v", host, got, want)
		}
	}
}

// fakeRelease ist ein Release auf einem Testserver.
type fakeRelease struct {
	server *httptest.Server
	tag    string
	files  map[string][]byte
	// sums ersetzt die gerechnete SHA256SUMS.
	sums string
	hits int
}

func newFakeRelease(t *testing.T, tag string, files map[string][]byte) *fakeRelease {
	t.Helper()
	f := &fakeRelease{tag: tag, files: files}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.hits++
		name := strings.TrimPrefix(r.URL.Path, "/")
		switch {
		case name == "latest":
			type assetJSON struct {
				Name string `json:"name"`
				Size int    `json:"size"`
				URL  string `json:"browser_download_url"`
			}
			assets := []assetJSON{{Name: checksumsName, URL: f.server.URL + "/" + checksumsName}}
			for n, data := range f.files {
				assets = append(assets, assetJSON{Name: n, Size: len(data), URL: f.server.URL + "/" + n})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tag_name": f.tag, "body": "## Neu\n\n- Eine Änderung.", "html_url": f.server.URL + "/seite",
				"published_at": "2026-11-20T10:00:00Z", "assets": assets,
			})
		case name == checksumsName:
			if f.sums != "" {
				fmt.Fprint(w, f.sums)
				return
			}
			for n, data := range f.files {
				sum := sha256.Sum256(data)
				fmt.Fprintf(w, "%s  %s\n", hex.EncodeToString(sum[:]), n)
			}
		case f.files[name] != nil:
			_, _ = w.Write(f.files[name])
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeRelease) service(current, goos string, installation Installation) *Service {
	s := New(current, slog.New(slog.DiscardHandler))
	s.goos, s.goarch = goos, "amd64"
	s.endpoint, s.client = f.server.URL+"/latest", f.server.Client()
	s.locate = func() (Installation, error) { return installation, nil }
	s.now = func() time.Time { return time.Date(2026, 11, 21, 8, 0, 0, 0, time.UTC) }
	return s
}

func tarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zipped := gzip.NewWriter(&buf)
	w := tar.NewWriter(zipped)
	for name, content := range files {
		if err := w.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zipped.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type zipEntry struct {
	name, content string
	mode          os.FileMode
}

func zipOf(t *testing.T, entries ...zipEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, e := range entries {
		header := &zip.FileHeader{Name: e.name, Method: zip.Deflate}
		header.SetMode(e.mode)
		out, err := w.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := out.Write([]byte(e.content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func installedBinary(t *testing.T, name string) Installation {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("alte Fassung"), 0o755); err != nil {
		t.Fatal(err)
	}
	return Installation{Path: path}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func entriesOf(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// Linux: Suche findet 0.4.0, die Installation tauscht die Programmdatei aus
// dem Paket; die alte Fassung liegt daneben, bis der nächste Start aufräumt.
func TestInstallReplacesTheBinaryFromTheTarball(t *testing.T) {
	release := newFakeRelease(t, "v0.4.0", map[string][]byte{
		"buchfink-linux-amd64.tar.gz": tarGz(t, map[string]string{"buchfink": "neue Fassung"}),
		"buchfink-windows-amd64.exe":  []byte("für ein anderes System"),
	})
	installation := installedBinary(t, "buchfink")
	s := release.service("0.3.0", "linux", installation)
	ctx := context.Background()

	if _, err := s.Install(ctx); err == nil {
		t.Error("ohne gefundene Fassung wurde installiert")
	}
	found := s.Check(ctx)
	if found.Phase != PhaseAvailable || found.Release == nil || found.Release.Version != "0.4.0" ||
		found.ManualReason != "" || found.CheckedAt != "2026-11-21T08:00:00Z" || !strings.Contains(found.Release.Notes, "Eine Änderung") {
		t.Fatalf("Suche: %+v", found)
	}
	if err := s.Relaunch(); err == nil {
		t.Error("vor der Installation wurde neu gestartet")
	}

	done, err := s.Install(ctx)
	if err != nil {
		t.Fatalf("Installation: %v", err)
	}
	if done.Phase != PhaseInstalled || done.Progress != 100 {
		t.Errorf("Stand nach der Installation: %+v", done)
	}
	if got := read(t, installation.Path); got != "neue Fassung" {
		t.Errorf("Programmdatei enthält %q", got)
	}
	if info, err := os.Stat(installation.Path); err != nil || info.Mode().Perm()&0o111 == 0 {
		t.Errorf("die neue Programmdatei ist nicht ausführbar: %v", info)
	}
	if got := read(t, installation.Path+oldSuffix); got != "alte Fassung" {
		t.Errorf("die ersetzte Fassung enthält %q", got)
	}
	if names := entriesOf(t, installation.dir()); len(names) != 2 {
		t.Errorf("nach der Installation liegt im Ordner: %v", names)
	}
	// Eine installierte Fassung bleibt stehen, auch wenn noch einmal gesucht wird.
	if again := s.Check(ctx); again.Phase != PhaseInstalled {
		t.Errorf("Suche nach der Installation: %+v", again)
	}

	installation.Cleanup()
	if names := entriesOf(t, installation.dir()); len(names) != 1 || names[0] != "buchfink" {
		t.Errorf("nach dem Aufräumen liegt im Ordner: %v", names)
	}
}

// Windows: die Datei des Release ist die Programmdatei selbst.
func TestInstallReplacesTheWindowsExecutable(t *testing.T) {
	release := newFakeRelease(t, "v0.4.0", map[string][]byte{"buchfink-windows-amd64.exe": []byte("neue Fassung")})
	installation := installedBinary(t, "buchfink.exe")
	s := release.service("0.3.0", "windows", installation)
	s.Check(context.Background())
	if _, err := s.Install(context.Background()); err != nil {
		t.Fatalf("Installation: %v", err)
	}
	if got := read(t, installation.Path); got != "neue Fassung" {
		t.Errorf("Programmdatei enthält %q", got)
	}
	if got := read(t, installation.Path+oldSuffix); got != "alte Fassung" {
		t.Errorf("die ersetzte Fassung enthält %q", got)
	}
}

// macOS: das App-Paket wird als Ganzes getauscht, mit seinen Verweisen und
// Rechten; was das Archiv daneben enthält, bleibt draußen.
func TestInstallReplacesTheAppBundle(t *testing.T) {
	archive := zipOf(t,
		zipEntry{"buchfink.app/", "", os.ModeDir | 0o755},
		zipEntry{"buchfink.app/._Contents", "attribute", 0o644},
		zipEntry{"buchfink.app/Contents/Info.plist", "<plist/>", 0o644},
		zipEntry{"buchfink.app/Contents/._Info.plist", "attribute", 0o644},
		zipEntry{"buchfink.app/Contents/MacOS/._buchfink", "attribute", 0o644},
		zipEntry{"buchfink.app/Contents/MacOS/buchfink", "neue Fassung", 0o755},
		zipEntry{"buchfink.app/Contents/Frameworks/Current", "A", os.ModeSymlink | 0o777},
		zipEntry{"buchfink.app/Contents/Frameworks/A/lib", "bibliothek", 0o644},
		zipEntry{"__MACOSX/buchfink.app/._Info.plist", "ballast", 0o644},
	)
	release := newFakeRelease(t, "v0.4.0", map[string][]byte{"buchfink-darwin-universal.zip": archive})
	dir := t.TempDir()
	bundle := filepath.Join(dir, "Buchfink.app")
	if err := os.MkdirAll(filepath.Join(bundle, bundleBinary), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, bundleBinary, "buchfink"), []byte("alte Fassung"), 0o755); err != nil {
		t.Fatal(err)
	}
	installation := Installation{Path: bundle, Bundle: true}
	s := release.service("0.3.0", "darwin", installation)
	s.Check(context.Background())
	if _, err := s.Install(context.Background()); err != nil {
		t.Fatalf("Installation: %v", err)
	}
	binary := filepath.Join(bundle, bundleBinary, "buchfink")
	if got := read(t, binary); got != "neue Fassung" {
		t.Errorf("Programmdatei enthält %q", got)
	}
	if info, err := os.Stat(binary); err != nil || info.Mode().Perm()&0o111 == 0 {
		t.Errorf("die Programmdatei im Paket ist nicht ausführbar: %v", info)
	}
	if link, err := os.Readlink(filepath.Join(bundle, "Contents", "Frameworks", "Current")); err != nil || link != "A" {
		t.Errorf("der Verweis im Paket ist %q (%v)", link, err)
	}
	if got := read(t, filepath.Join(bundle+oldSuffix, bundleBinary, "buchfink")); got != "alte Fassung" {
		t.Errorf("das ersetzte Paket enthält %q", got)
	}
	for _, sidecar := range []string{"._Contents", "Contents/._Info.plist", "Contents/MacOS/._buchfink"} {
		if _, err := os.Stat(filepath.Join(bundle, sidecar)); err == nil {
			t.Errorf("die Begleitdatei %s liegt im Paket", sidecar)
		}
	}
	if names := entriesOf(t, dir); len(names) != 2 {
		t.Errorf("nach der Installation liegt im Ordner: %v", names)
	}
	installation.Cleanup()
	if names := entriesOf(t, dir); len(names) != 1 || names[0] != "Buchfink.app" {
		t.Errorf("nach dem Aufräumen liegt im Ordner: %v", names)
	}
}

// Stimmt die Prüfsumme nicht, bleibt die Installation, wie sie war, und im
// Ordner bleibt nichts liegen. Dasselbe gilt für ein Archiv, das aus sich
// heraus schreibt oder verweist.
func TestInstallKeepsTheOldVersionWhenTheDownloadIsWrong(t *testing.T) {
	ctx := context.Background()
	check := func(t *testing.T, s *Service, installation Installation, want string) {
		t.Helper()
		s.Check(ctx)
		st, err := s.Install(ctx)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("erwartet Fehler mit %q, erhalten %v", want, err)
		}
		if st.Phase != PhaseAvailable || st.Error == "" {
			t.Errorf("Stand nach dem Fehlschlag: %+v", st)
		}
		installation.Cleanup()
		if names := entriesOf(t, installation.dir()); len(names) != 1 {
			t.Errorf("im Ordner liegt: %v", names)
		}
	}

	t.Run("Prüfsumme", func(t *testing.T) {
		release := newFakeRelease(t, "v0.4.0", map[string][]byte{
			"buchfink-linux-amd64.tar.gz": tarGz(t, map[string]string{"buchfink": "manipuliert"})})
		release.sums = strings.Repeat("0", 64) + "  buchfink-linux-amd64.tar.gz\n"
		installation := installedBinary(t, "buchfink")
		check(t, release.service("0.3.0", "linux", installation), installation, "Prüfsumme")
		if got := read(t, installation.Path); got != "alte Fassung" {
			t.Errorf("Programmdatei enthält %q", got)
		}
	})
	t.Run("ohne Prüfsumme", func(t *testing.T) {
		release := newFakeRelease(t, "v0.4.0", map[string][]byte{
			"buchfink-linux-amd64.tar.gz": tarGz(t, map[string]string{"buchfink": "neu"})})
		release.sums = strings.Repeat("0", 64) + "  andere-datei\n"
		installation := installedBinary(t, "buchfink")
		check(t, release.service("0.3.0", "linux", installation), installation, "keine Prüfsumme")
	})
	t.Run("Paket ohne Programmdatei", func(t *testing.T) {
		release := newFakeRelease(t, "v0.4.0", map[string][]byte{
			"buchfink-linux-amd64.tar.gz": tarGz(t, map[string]string{"anderes": "neu"})})
		installation := installedBinary(t, "buchfink")
		check(t, release.service("0.3.0", "linux", installation), installation, "keine Programmdatei")
	})
	for name, entries := range map[string][]zipEntry{
		"Eintrag außerhalb": {
			{"buchfink.app/Contents/MacOS/buchfink", "neu", 0o755},
			{"buchfink.app/../../ausbruch", "x", 0o644}},
		"Verweis nach außen": {
			{"buchfink.app/Contents/MacOS/buchfink", "neu", 0o755},
			{"buchfink.app/Contents/passwd", "/etc/passwd", os.ModeSymlink | 0o777}},
		"zweites Paket": {
			{"buchfink.app/Contents/MacOS/buchfink", "neu", 0o755},
			{"anderes.app/Contents/MacOS/buchfink", "neu", 0o755}},
	} {
		t.Run(name, func(t *testing.T) {
			release := newFakeRelease(t, "v0.4.0", map[string][]byte{"buchfink-darwin-universal.zip": zipOf(t, entries...)})
			dir := t.TempDir()
			bundle := filepath.Join(dir, "buchfink.app")
			if err := os.MkdirAll(filepath.Join(bundle, bundleBinary), 0o755); err != nil {
				t.Fatal(err)
			}
			installation := Installation{Path: bundle, Bundle: true}
			check(t, release.service("0.3.0", "darwin", installation), installation, "Paket")
			if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "ausbruch")); err == nil {
				t.Error("das Archiv hat außerhalb des Installationsordners geschrieben")
			}
		})
	}
}

func TestCheckReportsWhatBuchfinkCannotInstallItself(t *testing.T) {
	ctx := context.Background()
	release := newFakeRelease(t, "v0.4.0", map[string][]byte{"buchfink-windows-amd64.exe": []byte("neu")})

	// Für Linux enthält dieses Release keine Datei.
	s := release.service("0.3.0", "linux", installedBinary(t, "buchfink"))
	st := s.Check(ctx)
	if st.Phase != PhaseAvailable || !strings.Contains(st.ManualReason, "keine Datei") {
		t.Errorf("ohne Datei für das System: %+v", st)
	}
	if _, err := s.Install(ctx); err == nil {
		t.Error("ohne Datei für das System wurde installiert")
	}

	// Die Installation lässt sich nicht bestimmen, etwa bei einem Start aus
	// dem Download unter macOS.
	s = release.service("0.3.0", "windows", Installation{})
	s.locate = func() (Installation, error) {
		return installationOf("darwin", "/private/var/folders/x/AppTranslocation/y/d/buchfink.app/Contents/MacOS/buchfink")
	}
	if st := s.Check(ctx); !strings.Contains(st.ManualReason, "Programme") {
		t.Errorf("Start aus dem Download: %+v", st)
	}

	// Kein Schreibrecht im Installationsordner.
	if os.Getuid() != 0 {
		installation := installedBinary(t, "buchfink.exe")
		if err := os.Chmod(installation.dir(), 0o555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(installation.dir(), 0o755) })
		s = release.service("0.3.0", "windows", installation)
		if st := s.Check(ctx); !strings.Contains(st.ManualReason, "nicht schreiben") {
			t.Errorf("ohne Schreibrecht: %+v", st)
		}
	}
}

func TestCheckFindsNothingNewer(t *testing.T) {
	ctx := context.Background()
	release := newFakeRelease(t, "v0.3.0", map[string][]byte{"buchfink-linux-amd64.tar.gz": []byte("x")})
	s := release.service("0.3.0", "linux", installedBinary(t, "buchfink"))
	if st := s.Check(ctx); st.Phase != PhaseIdle || st.Release != nil || st.Error != "" || st.CheckedAt == "" {
		t.Errorf("gleiche Fassung: %+v", st)
	}

	// Ein Arbeitsstand fragt nicht nach.
	release.hits = 0
	dev := release.service("dev+8cd1f9a", "linux", installedBinary(t, "buchfink"))
	if st := dev.Check(ctx); st.Supported || st.Phase != PhaseIdle || release.hits != 0 {
		t.Errorf("Arbeitsstand: %+v nach %d Abrufen", st, release.hits)
	}
	done := make(chan struct{})
	go func() { dev.Run(ctx, func() bool { return true }); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("ein Arbeitsstand sucht im Hintergrund")
	}

	// Antwortet der Server nicht, steht der Grund am Stand.
	release.server.Close()
	if st := s.Check(ctx); st.Phase != PhaseIdle || st.Error == "" {
		t.Errorf("Server nicht erreichbar: %+v", st)
	}
}

func TestInstallationOfTheRunningProgram(t *testing.T) {
	got, err := installationOf("darwin", "/Applications/Buchfink.app/Contents/MacOS/buchfink")
	if err != nil || !got.Bundle || got.Path != "/Applications/Buchfink.app" {
		t.Errorf("App-Paket: %+v, %v", got, err)
	}
	if _, err := installationOf("darwin", "/Users/x/repos/buchfink/bin/buchfink"); err == nil {
		t.Error("eine Programmdatei außerhalb eines Pakets gilt unter macOS als Installation")
	}
	got, err = installationOf("linux", "/opt/buchfink/buchfink")
	if err != nil || got.Bundle || got.Path != "/opt/buchfink/buchfink" {
		t.Errorf("Programmdatei: %+v, %v", got, err)
	}
}

func TestRelaunchNamesThePredecessor(t *testing.T) {
	cmd := relaunchCommand(Installation{Path: "/opt/buchfink/buchfink"}, 4711)
	if cmd.Path != "/opt/buchfink/buchfink" || len(cmd.Args) != 2 || cmd.Args[1] != predecessorFlag+"4711" {
		t.Errorf("Programmdatei: %v", cmd.Args)
	}
	cmd = relaunchCommand(Installation{Path: "/Applications/Buchfink.app", Bundle: true}, 4711)
	want := []string{"/usr/bin/open", "-n", "/Applications/Buchfink.app", "--args", predecessorFlag + "4711"}
	if strings.Join(cmd.Args, " ") != strings.Join(want, " ") {
		t.Errorf("App-Paket: %v", cmd.Args)
	}
}

// Der Nachfolger wartet auf das Ende des Vorgängers, aber nicht auf einen
// Prozess, den es nicht mehr gibt, und nicht ohne die Angabe.
func TestWaitForPredecessor(t *testing.T) {
	child := exec.Command(os.Args[0], "-test.run=^$")
	if err := child.Run(); err != nil {
		t.Fatal(err)
	}
	gone := child.Process.Pid
	start := time.Now()
	WaitForPredecessor([]string{"buchfink", predecessorFlag + fmt.Sprint(gone)})
	WaitForPredecessor([]string{"buchfink"})
	WaitForPredecessor([]string{"buchfink", predecessorFlag + "kein-prozess"})
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("auf einen beendeten Prozess wurde %s gewartet", elapsed)
	}

	running := exec.Command(os.Args[0], "-test.run=^TestHelperSleep$")
	running.Env = append(os.Environ(), "BUCHFINK_TEST_SLEEP=1")
	if err := running.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { _ = running.Wait() }()
	start = time.Now()
	waitForExit(running.Process.Pid, 10*time.Second)
	if elapsed := time.Since(start); elapsed < 500*time.Millisecond {
		t.Errorf("auf den laufenden Prozess wurde nur %s gewartet", elapsed)
	}
}

// TestHelperSleep ist der Prozess, auf den TestWaitForPredecessor wartet.
func TestHelperSleep(t *testing.T) {
	if os.Getenv("BUCHFINK_TEST_SLEEP") != "" {
		time.Sleep(time.Second)
	}
}
