package policy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
)

// Ein Agent übergibt Belege nur als Pfad. Buchfink liest sie aus den
// Belegordnern, die der Mensch in der App freigibt, und schreibt Exporte nur
// in den Exportordner. Voreingestellt ist allein der Eingang im Datenordner
// des Mandanten. Downloads gehört nicht dazu: dorthin schreibt jede Webseite
// und jeder Mailanhang, eine dort abgelegte Datei bleibt als Beleg zehn Jahre
// im Archiv, und eine gefälschte E-Rechnung oder ein gefälschter Kontoauszug
// käme ohne Zutun des Menschen in Reichweite des Agenten. Wer Downloads will,
// nimmt ihn in der App auf.

const (
	// InboxDir ist der Eingangsordner im Datenordner.
	InboxDir = "Belegeingang"
	// ExportDir ist der voreingestellte Exportordner im Datenordner.
	ExportDir = "Export"
	// MaxFileBytes ist die größte Datei, die ein Agent ablegen kann.
	MaxFileBytes = 50 << 20
)

// Folders sind die aufgelösten Ordner eines Mandanten.
type Folders struct {
	Receipt []string `json:"belegordner"`
	Export  string   `json:"exportordner"`
	// DataDir ist gesperrt, bis auf den Eingang darin.
	DataDir string `json:"-"`
	// Denied sind weitere gesperrte Ordner: die Datenordner der übrigen
	// Mandanten und alle Sicherungsordner.
	Denied []string `json:"-"`
}

// FoldersFor löst die Einstellungen des Mandanten zu Ordnern auf; nicht
// gesetzte stehen für die Voreinstellung.
func FoldersFor(ctx context.Context, s *core.Services, dataDir string, denied ...string) (Folders, error) {
	settings, err := s.Settings.AgentSettings(ctx)
	if err != nil {
		return Folders{}, err
	}
	return FoldersFrom(settings, dataDir, denied...), nil
}

// FoldersFrom löst Einstellungen zu Ordnern auf.
func FoldersFrom(settings *domain.AgentSettings, dataDir string, denied ...string) Folders {
	f := Folders{DataDir: dataDir, Export: filepath.Join(dataDir, ExportDir)}
	for _, d := range denied {
		if strings.TrimSpace(d) != "" {
			f.Denied = append(f.Denied, d)
		}
	}
	if settings != nil && settings.ExportFolder != "" {
		f.Export = settings.ExportFolder
	}
	if settings != nil && settings.ReceiptFolders != nil {
		f.Receipt = append([]string{}, settings.ReceiptFolders...)
		return f
	}
	f.Receipt = []string{filepath.Join(dataDir, InboxDir)}
	return f
}

// ProtectedDirs sind die Ordner, aus denen kein Agent eines Mandanten liest:
// die Datenordner aller anderen Mandanten und alle Sicherungsordner.
func ProtectedDirs(cfg domain.AppConfig, ownDataDir string) []string {
	out := []string{cfg.BackupDir}
	for _, t := range cfg.Tenants {
		out = append(out, t.BackupDir)
		if t.DataDir != "" && filepath.Clean(t.DataDir) != filepath.Clean(ownDataDir) {
			out = append(out, t.DataDir)
		}
	}
	return out
}

// CheckReceiptFolders weist Belegordner ab, die einen geschützten Ordner
// enthalten oder in einem liegen. Der eigene Eingang ist erlaubt, auch wenn er
// im eigenen Datenordner liegt.
func CheckReceiptFolders(folders []string, ownDataDir string, protected []string) error {
	inbox := filepath.Join(ownDataDir, InboxDir)
	guarded := append([]string{ownDataDir}, protected...)
	for _, folder := range folders {
		clean := filepath.Clean(folder)
		if clean == inbox {
			continue
		}
		for _, p := range guarded {
			if strings.TrimSpace(p) == "" {
				continue
			}
			p = filepath.Clean(p)
			if clean == p || within(p, clean) || within(clean, p) {
				return fmt.Errorf("der Belegordner %s überschneidet sich mit %s (Datenordner eines Mandanten "+
					"oder Sicherung); er wäre für verbundene Apps lesbar", folder, p)
			}
		}
	}
	return nil
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

// Kind ist die erkannte Art einer Datei.
type Kind string

const (
	KindPDF   Kind = "pdf"
	KindImage Kind = "bild"
	KindXML   Kind = "xml"
)

// File ist eine geprüfte und gelesene Datei.
type File struct {
	Path string
	Name string
	Kind Kind
	Data []byte
}

var extensions = map[string]Kind{
	".pdf": KindPDF, ".png": KindImage, ".jpg": KindImage, ".jpeg": KindImage,
	".tif": KindImage, ".tiff": KindImage, ".webp": KindImage, ".xml": KindXML,
}

// ReadFile prüft einen Pfad gegen die Belegordner und liest die Datei
// einmal. Zuerst wird der Pfad ohne Zugriff auf das Dateisystem geprüft:
// liegt er nicht in einem Belegordner, erfährt der Agent nicht einmal, ob es
// die Datei gibt, und ein Netzwerkpfad wird nie berührt. Danach werden
// Symlinks aufgelöst und erneut geprüft; gelesen wird nur eine gewöhnliche
// Datei ohne weiteren harten Verweis, und nach dem Öffnen muss sie noch an
// derselben Stelle liegen.
func ReadFile(path string, f Folders, allowed ...Kind) (*File, error) {
	path = strings.TrimSpace(path)
	refused := notAllowed("%s liegt in keinem freigegebenen Belegordner (%s)", path, strings.Join(f.Receipt, ", "))
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
	kind, ok := extensions[strings.ToLower(filepath.Ext(name))]
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
	// Ein harter Verweis macht eine Datei außerhalb der Belegordner unter einem
	// Namen darin lesbar; kein Beleg braucht einen.
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
	if !contentMatches(kind, data) {
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
// er liegt in einem Belegordner, so wie er eingestellt ist oder aufgelöst.
func (f Folders) lexicallyAllows(clean string) bool {
	for _, folder := range f.Receipt {
		if remotePath(folder) {
			continue
		}
		raw := filepath.Clean(folder)
		if within(raw, clean) || within(resolveDir(raw), clean) {
			return true
		}
	}
	return false
}

// allows meldet, ob ein aufgelöster Pfad in einem Belegordner liegt und in
// keinem gesperrten. Im eigenen Datenordner ist nur der Eingang offen; ein
// gesperrter Ordner, der den eigenen Eingang umschließt, sperrt ihn nicht.
func (f Folders) allows(resolved string) bool {
	inFolder := false
	for _, folder := range f.Receipt {
		if remotePath(folder) {
			continue
		}
		if within(resolveDir(folder), resolved) {
			inFolder = true
			break
		}
	}
	if !inFolder {
		return false
	}
	inInbox := false
	var inbox string
	if f.DataDir != "" {
		data := resolveDir(f.DataDir)
		inbox = filepath.Join(data, InboxDir)
		inInbox = within(inbox, resolved)
		if within(data, resolved) && !inInbox {
			return false
		}
	}
	for _, d := range f.Denied {
		dir := resolveDir(d)
		if !within(dir, resolved) {
			continue
		}
		if inInbox && within(dir, inbox) {
			continue
		}
		return false
	}
	return true
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

// EnsureInbox legt den Eingang im Datenordner an, damit der Mensch weiß,
// wohin er Belege legt.
func (f Folders) EnsureInbox() {
	if f.DataDir == "" {
		return
	}
	inbox := filepath.Join(f.DataDir, InboxDir)
	for _, folder := range f.Receipt {
		if filepath.Clean(folder) == inbox {
			_ = os.MkdirAll(inbox, 0o755)
			return
		}
	}
}

func resolveDir(dir string) string {
	dir = filepath.Clean(dir)
	if r, err := filepath.EvalSymlinks(dir); err == nil {
		return r
	}
	return dir
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

func contentMatches(kind Kind, data []byte) bool {
	switch kind {
	case KindPDF:
		return bytes.HasPrefix(data, []byte("%PDF-"))
	case KindImage:
		return bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G'}) ||
			bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}) ||
			bytes.HasPrefix(data, []byte("II*\x00")) || bytes.HasPrefix(data, []byte("MM\x00*")) ||
			(len(data) > 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP")
	case KindXML:
		trimmed := bytes.TrimLeft(bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}), " \t\r\n")
		return bytes.HasPrefix(trimmed, []byte("<"))
	}
	return false
}
