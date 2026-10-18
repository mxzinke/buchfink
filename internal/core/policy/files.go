package policy

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/inbox"
)

// Ein Agent übergibt Belege nur als Pfad. Buchfink liest sie aus dem Eingang
// des Mandanten und aus weiteren Leseordnern, die der Mensch in der App
// freigibt, und schreibt Exporte nur in den Exportordner. Voreingestellt ist
// allein der Eingang im Datenordner. Downloads gehört nicht dazu: dorthin
// schreibt jede Webseite und jeder Mailanhang, eine dort abgelegte Datei
// bleibt als Beleg zehn Jahre im Archiv, und eine gefälschte E-Rechnung oder
// ein gefälschter Kontoauszug käme ohne Zutun des Menschen in Reichweite des
// Agenten. Wer Downloads will, nimmt ihn in der App als Leseordner auf.

const (
	// InboxDir ist der voreingestellte Eingang im Datenordner.
	InboxDir = domain.DefaultInboxDir
	// ExportDir ist der voreingestellte Exportordner im Datenordner.
	ExportDir = "Export"
	// MaxFileBytes ist die größte Datei, die ein Agent ablegen kann.
	MaxFileBytes = inbox.MaxFileBytes
)

// Folders sind die aufgelösten Ordner eines Mandanten.
type Folders struct {
	// Inbox ist der Eingang: genau einer, im Datenordner oder außerhalb.
	Inbox string `json:"eingang"`
	// Read sind weitere Leseordner.
	Read   []string `json:"leseordner"`
	Export string   `json:"exportordner"`
	// DataDir ist gesperrt, bis auf den voreingestellten Eingang darin.
	DataDir string `json:"-"`
	// Protected sind die weiteren gesperrten Ordner.
	Protected Protected `json:"-"`
}

// Protected sind die Ordner, aus denen kein Agent eines Mandanten liest.
type Protected struct {
	// DataDirs sind die Datenordner der übrigen Mandanten.
	DataDirs []string
	// Backups sind alle Sicherungsordner.
	Backups []string
}

func (p Protected) all() []string {
	return append(append([]string{}, p.DataDirs...), p.Backups...)
}

// FoldersFor löst die Einstellungen des Mandanten zu Ordnern auf; nicht
// gesetzte stehen für die Voreinstellung.
func FoldersFor(ctx context.Context, s *core.Services, dataDir string, protected Protected) (Folders, error) {
	settings, err := s.Settings.AgentSettings(ctx)
	if err != nil {
		return Folders{}, err
	}
	inboxFolder, err := s.Settings.InboxFolder(ctx)
	if err != nil {
		return Folders{}, err
	}
	return FoldersFrom(settings, inboxFolder, dataDir, protected), nil
}

// FoldersFrom löst Einstellungen zu Ordnern auf. Ein leerer Eingang ist
// <Datenordner>/Eingang.
func FoldersFrom(settings *domain.AgentSettings, inboxFolder, dataDir string, protected Protected) Folders {
	f := Folders{Inbox: domain.CleanFolder(inboxFolder), Read: []string{}, DataDir: dataDir,
		Export: filepath.Join(dataDir, ExportDir)}
	if f.Inbox == "" {
		f.Inbox = DefaultInbox(dataDir)
	}
	for _, d := range protected.DataDirs {
		if strings.TrimSpace(d) != "" {
			f.Protected.DataDirs = append(f.Protected.DataDirs, d)
		}
	}
	for _, d := range protected.Backups {
		if strings.TrimSpace(d) != "" {
			f.Protected.Backups = append(f.Protected.Backups, d)
		}
	}
	if settings != nil && settings.ExportFolder != "" {
		f.Export = settings.ExportFolder
	}
	if settings != nil {
		f.Read = append(f.Read, settings.ExtraReadFolders...)
	}
	return f
}

// DefaultInbox ist der voreingestellte Eingang im Datenordner.
func DefaultInbox(dataDir string) string {
	return filepath.Join(dataDir, InboxDir)
}

// roots sind die Ordner, aus denen gelesen wird: Eingang und Leseordner.
func (f Folders) roots() []string {
	return append([]string{f.Inbox}, f.Read...)
}

// ProtectedDirs sind die Ordner, aus denen kein Agent eines Mandanten liest:
// die Datenordner aller anderen Mandanten und alle Sicherungsordner.
func ProtectedDirs(cfg domain.AppConfig, ownDataDir string) Protected {
	p := Protected{Backups: []string{cfg.BackupDir}}
	for _, t := range cfg.Tenants {
		p.Backups = append(p.Backups, t.BackupDir)
		if t.DataDir != "" && filepath.Clean(t.DataDir) != filepath.Clean(ownDataDir) {
			p.DataDirs = append(p.DataDirs, t.DataDir)
		}
	}
	return p
}

// CheckFolders weist einen Eingang oder Leseordner ab, der einen geschützten
// Ordner enthält oder in einem liegt: den eigenen Datenordner, die Datenordner
// der anderen Mandanten, eine Sicherung. Im eigenen Datenordner ist allein
// der voreingestellte Eingang erlaubt. Leer heißt beim Eingang Voreinstellung.
// Ein Leseordner darf den Eingang weder enthalten noch in ihm liegen: aus dem
// Eingang verschiebt Buchfink nach der Ablage, aus einem Leseordner nicht.
// Geprüft wird der Pfad so, wie er gespeichert wird (domain.CleanFolder).
func CheckFolders(inboxFolder string, read []string, ownDataDir string, protected Protected) error {
	inboxFolder = domain.CleanFolder(inboxFolder)
	effective := inboxFolder
	if inboxFolder == "" || inboxFolder == DefaultInbox(filepath.Clean(ownDataDir)) {
		inboxFolder, effective = "", DefaultInbox(filepath.Clean(ownDataDir))
	}
	if err := checkFolder("der Eingang", inboxFolder, ownDataDir, protected.all()); err != nil {
		return err
	}
	for _, folder := range read {
		folder = domain.CleanFolder(folder)
		if folder == "" {
			continue
		}
		if err := checkFolder("der Leseordner", folder, ownDataDir, protected.all()); err != nil {
			return err
		}
		if overlaps(folder, effective) {
			return fmt.Errorf("der Leseordner %s überschneidet sich mit dem Eingang %s; was im Eingang liegt, "+
				"verschiebt Buchfink nach der Ablage, aus einem Leseordner nicht", folder, effective)
		}
	}
	return nil
}

func checkFolder(what, folder, ownDataDir string, protected []string) error {
	if folder == "" {
		return nil
	}
	if !filepath.IsAbs(folder) || remotePath(folder) {
		return fmt.Errorf("%s %s ist kein lokaler absoluter Pfad", what, folder)
	}
	for _, p := range append([]string{ownDataDir}, protected...) {
		if strings.TrimSpace(p) == "" {
			continue
		}
		if overlaps(folder, p) {
			return fmt.Errorf("%s %s überschneidet sich mit %s (Datenordner eines Mandanten "+
				"oder Sicherung); er wäre für verbundene Apps lesbar", what, folder, p)
		}
	}
	return nil
}

// overlaps meldet, ob zwei Ordner gleich sind oder einer im anderen liegt, so
// wie angegeben oder aufgelöst.
func overlaps(a, b string) bool {
	for _, x := range []string{filepath.Clean(a), resolveDir(a)} {
		for _, y := range []string{filepath.Clean(b), resolveDir(b)} {
			if inside(x, y) || within(y, x) {
				return true
			}
		}
	}
	return false
}

// FileError ist ein abgewiesener Pfad (NotAllowed) oder eine ungeeignete
// Datei.
type FileError struct {
	NotAllowed bool
	msg        string
}

func (e *FileError) Error() string { return e.msg }

func notAllowed(format string, args ...any) error {
	return &FileError{NotAllowed: true, msg: fmt.Sprintf(format, args...)}
}

func invalidFile(format string, args ...any) error {
	return &FileError{msg: fmt.Sprintf(format, args...)}
}

// Kind ist das erkannte Format einer Datei.
type Kind = inbox.Format

const (
	KindPDF   = inbox.FormatPDF
	KindImage = inbox.FormatImage
	KindXML   = inbox.FormatXML
)

// File ist eine geprüfte und gelesene Datei.
type File struct {
	Path string
	Name string
	Kind Kind
	Data []byte
}

// ReadFile prüft einen Pfad gegen Eingang und Leseordner und liest die Datei
// einmal. Zuerst wird der Pfad ohne Zugriff auf das Dateisystem geprüft:
// liegt er weder im Eingang noch in einem Leseordner, erfährt der Agent nicht
// einmal, ob es die Datei gibt, und ein Netzwerkpfad wird nie berührt. Danach
// werden Symlinks aufgelöst und erneut geprüft; gelesen wird nur eine
// gewöhnliche Datei ohne weiteren harten Verweis, und nach dem Öffnen muss sie
// noch an derselben Stelle liegen.
func ReadFile(path string, f Folders, allowed ...Kind) (*File, error) {
	path = strings.TrimSpace(path)
	refused := notAllowed("%s liegt weder im Eingang noch in einem freigegebenen Leseordner (%s)", path,
		strings.Join(f.roots(), ", "))
	if path == "" || !filepath.IsAbs(path) || remotePath(path) {
		return nil, refused
	}
	clean := filepath.Clean(path)
	if !f.lexicallyAllows(clean) {
		return nil, refused
	}
	name := filepath.Base(clean)
	resolved, err := filepath.EvalSymlinks(clean)
	if err != nil {
		return nil, invalidFile("die Datei %s gibt es nicht oder sie ist nicht lesbar", name)
	}
	if remotePath(resolved) || !f.allows(resolved) {
		return nil, refused
	}
	name = filepath.Base(resolved)
	if strings.HasPrefix(name, ".") {
		return nil, notAllowed("versteckte Dateien werden nicht abgelegt: %s", name)
	}
	kind, ok := inbox.FormatOf(name)
	if !ok || (len(allowed) > 0 && !containsKind(allowed, kind)) {
		return nil, invalidFile("%s hat keine zulässige Endung (%s)", name, allowedList(allowed))
	}
	info, err := os.Lstat(resolved)
	if err != nil {
		return nil, invalidFile("die Datei %s ist nicht lesbar", name)
	}
	if !info.Mode().IsRegular() {
		return nil, invalidFile("%s ist keine gewöhnliche Datei", name)
	}
	if info.Size() > MaxFileBytes {
		return nil, invalidFile("%s ist %d MB groß; höchstens %d MB", name, info.Size()>>20, MaxFileBytes>>20)
	}
	if info.Size() == 0 {
		return nil, invalidFile("%s ist leer", name)
	}
	file, err := openNoFollow(resolved)
	if err != nil {
		return nil, invalidFile("die Datei %s ist nicht lesbar", name)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, invalidFile("die Datei %s hat sich beim Öffnen geändert", name)
	}
	// Ein harter Verweis macht eine Datei außerhalb von Eingang und
	// Leseordnern unter einem Namen darin lesbar; kein Beleg braucht einen.
	if links, err := linkCount(file, opened); err != nil || links > 1 {
		return nil, notAllowed("%s hat weitere harte Verweise im Dateisystem und wird nicht abgelegt", name)
	}
	// Nach dem Öffnen noch einmal: ein Ordner auf dem Weg könnte zwischen
	// Prüfung und Öffnen gegen einen Symlink getauscht worden sein.
	if again, err := filepath.EvalSymlinks(clean); err != nil || again != resolved || !f.allows(again) {
		return nil, invalidFile("die Datei %s hat sich beim Öffnen geändert", name)
	}
	if actual, ok := openedPath(file); ok && (actual != resolved || !f.allows(actual)) {
		return nil, invalidFile("die Datei %s hat sich beim Öffnen geändert", name)
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxFileBytes+1))
	if err != nil {
		return nil, invalidFile("die Datei %s ist nicht lesbar", name)
	}
	if len(data) > MaxFileBytes {
		return nil, invalidFile("%s ist größer als %d MB", name, MaxFileBytes>>20)
	}
	if !inbox.ContentMatches(kind, data) {
		return nil, invalidFile("der Inhalt von %s passt nicht zur Endung %s", name, filepath.Ext(name))
	}
	return &File{Path: resolved, Name: name, Kind: kind, Data: data}, nil
}

// remotePath erkennt UNC-, Geräte- und Netzwerkpfade (\\server\freigabe,
// \\?\, \\.\, //server). Ein Zugriff darauf meldet sich unter Windows beim
// fremden Rechner an; er unterbleibt deshalb vor jeder anderen Prüfung.
func remotePath(path string) bool {
	if strings.HasPrefix(path, `\\`) || strings.HasPrefix(path, "//") || strings.HasPrefix(path, `/\`) ||
		strings.HasPrefix(path, `\/`) {
		return true
	}
	return len(filepath.VolumeName(path)) > 2
}

// lexicallyAllows prüft den bereinigten Pfad, ohne das Dateisystem zu fragen:
// er liegt im Eingang oder einem Leseordner, so wie eingestellt oder
// aufgelöst.
func (f Folders) lexicallyAllows(clean string) bool {
	for _, folder := range f.roots() {
		if strings.TrimSpace(folder) == "" || remotePath(folder) {
			continue
		}
		raw := filepath.Clean(folder)
		if within(raw, clean) || within(resolveDir(raw), clean) {
			return true
		}
	}
	return false
}

// allows meldet, ob ein aufgelöster Pfad im Eingang oder einem Leseordner
// liegt und in keinem gesperrten.
func (f Folders) allows(resolved string) bool {
	for _, folder := range f.roots() {
		if strings.TrimSpace(folder) == "" || remotePath(folder) {
			continue
		}
		if within(resolveDir(folder), resolved) {
			return !f.guarded(resolved)
		}
	}
	return false
}

// guarded meldet, ob ein aufgelöster Pfad in einem gesperrten Ordner liegt:
// im eigenen Datenordner, im Datenordner eines anderen Mandanten oder in einer
// Sicherung. Offen bleibt der voreingestellte Eingang im eigenen Datenordner,
// wenn er der eingestellte ist, auch wenn der Datenordner eines anderen
// Mandanten ihn umschließt. Eine Sicherung, die den eingestellten Eingang
// umschließt, sperrt ihn nicht. Der Datenordner eines anderen Mandanten sperrt
// immer, auch wenn der Eingang inzwischen dorthin zeigt.
func (f Folders) guarded(resolved string) bool {
	inboxDir := ""
	if strings.TrimSpace(f.Inbox) != "" && !remotePath(f.Inbox) {
		inboxDir = resolveDir(f.Inbox)
	}
	ownInbox := ""
	if f.DataDir != "" {
		data := resolveDir(f.DataDir)
		if def := DefaultInbox(data); inboxDir == def {
			ownInbox = def
		}
		if inside(data, resolved) && (ownInbox == "" || !inside(ownInbox, resolved)) {
			return true
		}
	}
	for _, d := range f.Protected.DataDirs {
		dir := resolveDir(d)
		if inside(dir, resolved) && (ownInbox == "" || !inside(ownInbox, resolved) || !within(dir, ownInbox)) {
			return true
		}
	}
	for _, d := range f.Protected.Backups {
		dir := resolveDir(d)
		if inside(dir, resolved) && (inboxDir == "" || !inside(inboxDir, resolved) || !within(dir, inboxDir)) {
			return true
		}
	}
	return false
}

// CheckInbox weist einen Eingang ab, der aufgelöst in einem gesperrten Ordner
// liegt, etwa weil er inzwischen ein Symlink in den Datenordner eines anderen
// Mandanten ist, oder der ein Netzwerkpfad ist.
func (f Folders) CheckInbox() error {
	if strings.TrimSpace(f.Inbox) == "" || remotePath(f.Inbox) {
		return notAllowed("der Eingang %s ist kein lokaler Ordner", f.Inbox)
	}
	if f.guarded(resolveDir(f.Inbox)) {
		return notAllowed("der Eingang %s liegt im Datenordner eines Mandanten oder in einer Sicherung; "+
			"Buchfink liest ihn nicht", f.Inbox)
	}
	return nil
}

// InboxFiles sind die Dateien im Eingang (inbox.List) ohne die gesperrten
// Ordner darin, etwa eine Sicherung im Eingang.
func (f Folders) InboxFiles() ([]inbox.File, error) {
	if err := f.CheckInbox(); err != nil {
		return nil, err
	}
	return inbox.List(f.Inbox, f.guarded)
}

// InboxCount zählt dieselben Dateien wie InboxFiles und nennt den Zeitpunkt
// der ältesten.
func (f Folders) InboxCount() (int, time.Time, error) {
	if err := f.CheckInbox(); err != nil {
		return 0, time.Time{}, err
	}
	return inbox.Count(f.Inbox, f.guarded)
}

// ExportPath bereitet den Exportordner vor und liefert den Zielpfad einer
// Datei darin. Der voreingestellte Ordner im Datenordner wird angelegt, ein
// eingestellter muss bestehen.
func (f Folders) ExportPath(name string) (string, error) {
	if f.Export == "" || !filepath.IsAbs(f.Export) {
		return "", notAllowed("es ist kein Exportordner eingerichtet")
	}
	if f.DataDir != "" && filepath.Clean(f.Export) == filepath.Join(f.DataDir, ExportDir) {
		if err := os.MkdirAll(f.Export, 0o755); err != nil {
			return "", invalidFile("der Exportordner %s ließ sich nicht anlegen: %v", f.Export, err)
		}
	}
	info, err := os.Stat(f.Export)
	if err != nil || !info.IsDir() {
		return "", notAllowed("der Exportordner %s besteht nicht", f.Export)
	}
	if strings.ContainsAny(name, `/\`) || name == "" || strings.HasPrefix(name, ".") {
		return "", invalidFile("unzulässiger Dateiname %q", name)
	}
	return filepath.Join(f.Export, name), nil
}

// EnsureInbox legt den voreingestellten Eingang im Datenordner an, damit der
// Mensch weiß, wohin er Belege legt. Einen eigenen Eingang legt die App an,
// wenn der Mensch ihn wählt.
func (f Folders) EnsureInbox() {
	if f.DataDir == "" {
		return
	}
	if inbox := DefaultInbox(f.DataDir); filepath.Clean(f.Inbox) == inbox {
		_ = os.MkdirAll(inbox, 0o755)
	}
}

func resolveDir(dir string) string {
	dir = filepath.Clean(dir)
	if r, err := filepath.EvalSymlinks(dir); err == nil {
		return r
	}
	return dir
}

// inside meldet, ob path der Ordner dir ist oder in ihm liegt.
func inside(dir, path string) bool {
	return filepath.Clean(dir) == filepath.Clean(path) || within(dir, path)
}

func within(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil || rel == "." || filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func containsKind(list []Kind, k Kind) bool {
	for _, x := range list {
		if x == k {
			return true
		}
	}
	return false
}

func allowedList(allowed []Kind) string {
	if len(allowed) == 1 && allowed[0] == KindXML {
		return "xml"
	}
	return "pdf, png, jpg, jpeg, tif, tiff, webp, xml"
}
