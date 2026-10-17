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
// in den Exportordner. Die Voreinstellung ist der Eingang im Datenordner des
// Mandanten und der Downloads-Ordner: dort landen Anhänge und Downloads aus
// Portalen, und ein Agent ohne eigenen Dateizugriff (Claude Desktop) käme an
// eine Rechnung sonst nur, wenn der Mensch sie vorher verschiebt.

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
	// Denied sind weitere gesperrte Ordner (Sicherung).
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
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		f.Receipt = append(f.Receipt, filepath.Join(home, "Downloads"))
	}
	return f
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
// einmal. Symlinks und „.." werden aufgelöst, bevor geprüft wird; gelesen
// wird die aufgelöste Datei, und nur, wenn sie noch dieselbe ist.
func ReadFile(path string, f Folders, allowed ...Kind) (*File, error) {
	path = strings.TrimSpace(path)
	if path == "" || !filepath.IsAbs(path) {
		return nil, notAllowed("%q ist kein absoluter Pfad", path)
	}
	resolved, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return nil, invalidFile("die Datei %s ist nicht lesbar: %v", path, err)
	}
	if !f.allows(resolved) {
		return nil, notAllowed("%s liegt in keinem freigegebenen Belegordner (%s)", path, strings.Join(f.Receipt, ", "))
	}
	name := filepath.Base(resolved)
	if strings.HasPrefix(name, ".") {
		return nil, notAllowed("versteckte Dateien werden nicht abgelegt: %s", name)
	}
	kind, ok := extensions[strings.ToLower(filepath.Ext(name))]
	if !ok || (len(allowed) > 0 && !containsKind(allowed, kind)) {
		return nil, invalidFile("%s hat keine zulässige Endung (%s)", name, allowedList(allowed))
	}
	info, err := os.Lstat(resolved)
	if err != nil {
		return nil, invalidFile("die Datei %s ist nicht lesbar: %v", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, invalidFile("%s ist keine gewöhnliche Datei", path)
	}
	if info.Size() > MaxFileBytes {
		return nil, invalidFile("%s ist %d MB groß; höchstens %d MB", name, info.Size()>>20, MaxFileBytes>>20)
	}
	if info.Size() == 0 {
		return nil, invalidFile("%s ist leer", name)
	}
	file, err := os.Open(resolved)
	if err != nil {
		return nil, invalidFile("die Datei %s ist nicht lesbar: %v", path, err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, invalidFile("die Datei %s hat sich beim Öffnen geändert", path)
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxFileBytes+1))
	if err != nil {
		return nil, invalidFile("die Datei %s ist nicht lesbar: %v", path, err)
	}
	if len(data) > MaxFileBytes {
		return nil, invalidFile("%s ist größer als %d MB", name, MaxFileBytes>>20)
	}
	if !contentMatches(kind, data) {
		return nil, invalidFile("der Inhalt von %s passt nicht zur Endung %s", name, filepath.Ext(name))
	}
	return &File{Path: resolved, Name: name, Kind: kind, Data: data}, nil
}

// allows meldet, ob ein aufgelöster Pfad in einem Belegordner liegt und in
// keinem gesperrten. Im Datenordner ist nur der Eingang offen.
func (f Folders) allows(resolved string) bool {
	inFolder := false
	for _, folder := range f.Receipt {
		if within(resolveDir(folder), resolved) {
			inFolder = true
			break
		}
	}
	if !inFolder {
		return false
	}
	if f.DataDir != "" {
		data := resolveDir(f.DataDir)
		if within(data, resolved) && !within(filepath.Join(data, InboxDir), resolved) {
			return false
		}
	}
	for _, d := range f.Denied {
		if within(resolveDir(d), resolved) {
			return false
		}
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
