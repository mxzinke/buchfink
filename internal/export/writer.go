package export

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MetaFileName ist der Name der Metadatei.
const MetaFileName = "export.json"

const (
	dirPerm  = 0o700
	filePerm = 0o600
)

// Kind ist der Umfang eines Exports.
type Kind string

const (
	// KindZ3 ist die reine Datenüberlassung: Tabellen, Beschreibungsdatei,
	// Feldbeschreibung.
	KindZ3 Kind = "z3"
	// KindArchive ist die Datenüberlassung samt Belegdateien und
	// Anlagendokumenten.
	KindArchive Kind = "archive"
	// KindAuditPackage ist das Prüferpaket: Archiv plus Integritätsnachweis und
	// Verfahrensdokumentation.
	KindAuditPackage Kind = "audit_package"
	// KindJournal ist der Journalexport eines Zeitraums als einzelne Datei.
	KindJournal Kind = "journal"
	// KindKeyDirectory ist das Schlüsselverzeichnis als einzelne Datei.
	KindKeyDirectory Kind = "key_directory"
)

// TableInfo ist eine Tabelle im Ergebnis.
type TableInfo struct {
	Name string `json:"name"`
	File string `json:"file"`
	Rows int    `json:"rows"`
}

// FileInfo ist eine erzeugte Datei mit ihrer Prüfsumme.
//
// Die Prüfsumme steht dabei, weil ein Datenträger unterwegs beschädigt werden
// kann und der Empfänger sonst keine Möglichkeit hat, das zu bemerken.
type FileInfo struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

// Result ist das Ergebnis eines Exports. Es geht als JSON in die Metadatei und
// an die Oberfläche.
type Result struct {
	Kind            Kind   `json:"kind"`
	Dir             string `json:"dir"`
	TenantName      string `json:"tenantName"`
	FiscalYear      int    `json:"fiscalYear"`
	From            string `json:"from,omitempty"`
	To              string `json:"to,omitempty"`
	CreatedAt       string `json:"createdAt"`
	ProgramVersion  string `json:"programVersion"`
	StandardVersion string `json:"standardVersion"`

	Tables []TableInfo `json:"tables"`
	Files  []FileInfo  `json:"files"`

	// ReceiptFiles und DocumentFiles zählen die mitgegebenen Originaldateien.
	ReceiptFiles  int `json:"receiptFiles"`
	DocumentFiles int `json:"documentFiles"`

	// Notes sind Hinweise, die der Export nicht selbst beheben kann — eine
	// fehlende Verfahrensdokumentation etwa. Sie stehen im Ergebnis und nicht
	// nur im Log, weil sonst niemand sie liest.
	Notes []string `json:"notes"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
func (r *Result) EnsureLists() {
	if r.Tables == nil {
		r.Tables = make([]TableInfo, 0)
	}
	if r.Files == nil {
		r.Files = make([]FileInfo, 0)
	}
	if r.Notes == nil {
		r.Notes = make([]string, 0)
	}
}

// Builder schreibt einen Export in einen Ordner.
//
// Er sammelt die Prüfsummen aller geschriebenen Dateien mit, statt am Ende den
// Ordner noch einmal zu lesen: was geschrieben wurde, ist bekannt, und ein
// zweiter Durchgang über die Platte könnte etwas aufnehmen, das gar nicht zum
// Export gehört.
type Builder struct {
	dir    string
	result *Result
}

// NewBuilder legt den Zielordner an.
func NewBuilder(dir string, kind Kind, d *Dataset) (*Builder, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("kein Zielordner angegeben")
	}
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return nil, fmt.Errorf("Zielordner konnte nicht angelegt werden: %w", err)
	}
	return &Builder{
		dir: dir,
		result: &Result{
			Kind:            kind,
			Dir:             dir,
			TenantName:      d.TenantName,
			FiscalYear:      d.FiscalYear,
			From:            d.From,
			To:              d.To,
			CreatedAt:       d.CreatedAt,
			ProgramVersion:  d.ProgramVersion,
			StandardVersion: StandardVersion,
			Tables:          make([]TableInfo, 0, len(d.Tables)),
			Files:           make([]FileInfo, 0, len(d.Tables)+4),
			Notes:           make([]string, 0),
		},
	}, nil
}

// Dir ist der Zielordner.
func (b *Builder) Dir() string { return b.dir }

// Note vermerkt einen Hinweis im Ergebnis.
func (b *Builder) Note(format string, args ...any) {
	b.result.Notes = append(b.result.Notes, fmt.Sprintf(format, args...))
}

// WriteDataset schreibt die Tabellen, die Beschreibungsdatei, ihre Grammatik und
// die Feldbeschreibung.
func (b *Builder) WriteDataset(d *Dataset) error {
	for i := range d.Tables {
		t := &d.Tables[i]
		if err := b.WriteFile(t.FileName, RenderCSV(*t)); err != nil {
			return err
		}
		b.result.Tables = append(b.result.Tables, TableInfo{
			Name: t.Name, File: t.FileName, Rows: len(t.Rows),
		})
	}
	if err := b.WriteFile(IndexFileName, RenderIndexXML(d)); err != nil {
		return err
	}
	if err := b.WriteFile(DTDFileName, DTD()); err != nil {
		return err
	}
	return b.WriteFile(FieldDocFileName, RenderFieldDoc(d))
}

// WriteFile schreibt eine Datei relativ zum Zielordner und merkt sich ihre
// Prüfsumme.
func (b *Builder) WriteFile(relPath string, data []byte) error {
	abs, err := b.resolve(relPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), dirPerm); err != nil {
		return fmt.Errorf("Ordner für %s konnte nicht angelegt werden: %w", relPath, err)
	}
	if err := os.WriteFile(abs, data, filePerm); err != nil {
		return fmt.Errorf("%s konnte nicht geschrieben werden: %w", relPath, err)
	}
	sum := sha256.Sum256(data)
	b.result.Files = append(b.result.Files, FileInfo{
		Path:   filepath.ToSlash(relPath),
		SHA256: hex.EncodeToString(sum[:]),
		Bytes:  int64(len(data)),
	})
	return nil
}

// CopyFile kopiert eine Datei in den Export und liefert ihre Prüfsumme zurück.
//
// Sie wird beim Kopieren gerechnet und nicht aus der Datenbank übernommen: der
// Export soll belegen, was auf dem Datenträger liegt, nicht wiederholen, was
// beim Ablegen einmal gemessen wurde.
func (b *Builder) CopyFile(relPath, sourcePath string) (string, error) {
	abs, err := b.resolve(relPath)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(abs), dirPerm); err != nil {
		return "", fmt.Errorf("Ordner für %s konnte nicht angelegt werden: %w", relPath, err)
	}

	src, err := os.Open(sourcePath)
	if err != nil {
		return "", fmt.Errorf("%s konnte nicht gelesen werden: %w", filepath.Base(sourcePath), err)
	}
	defer src.Close()

	dst, err := os.OpenFile(abs, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, filePerm)
	if err != nil {
		return "", fmt.Errorf("%s konnte nicht angelegt werden: %w", relPath, err)
	}
	defer dst.Close()

	digest := sha256.New()
	written, err := io.Copy(io.MultiWriter(dst, digest), src)
	if err != nil {
		return "", fmt.Errorf("%s konnte nicht kopiert werden: %w", relPath, err)
	}
	sum := hex.EncodeToString(digest.Sum(nil))
	b.result.Files = append(b.result.Files, FileInfo{
		Path: filepath.ToSlash(relPath), SHA256: sum, Bytes: written,
	})
	return sum, nil
}

// CountReceiptFile und CountDocumentFile zählen die mitgegebenen Originale.
func (b *Builder) CountReceiptFile()  { b.result.ReceiptFiles++ }
func (b *Builder) CountDocumentFile() { b.result.DocumentFiles++ }

// Finish schreibt die Metadatei und liefert das Ergebnis.
//
// Die Metadatei führt sich selbst nicht auf: sie enthält die Prüfsummen aller
// übrigen Dateien, und eine Prüfsumme über eine Datei, die diese Prüfsumme
// enthält, gibt es nicht.
func (b *Builder) Finish() (*Result, error) {
	sort.Slice(b.result.Files, func(i, j int) bool {
		return b.result.Files[i].Path < b.result.Files[j].Path
	})
	b.result.EnsureLists()

	data, err := json.MarshalIndent(b.result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("Metadaten konnten nicht erzeugt werden: %w", err)
	}
	abs, err := b.resolve(MetaFileName)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(abs, append(data, '\n'), filePerm); err != nil {
		return nil, fmt.Errorf("%s konnte nicht geschrieben werden: %w", MetaFileName, err)
	}
	return b.result, nil
}

// resolve verhindert, dass ein Dateiname aus dem Zielordner herausführt. Die
// Namen stammen aus Belegdateinamen und damit aus einer fremden Quelle.
func (b *Builder) resolve(relPath string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(relPath))
	if filepath.IsAbs(clean) || clean == ".." ||
		strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("ungültiger Pfad im Export: %q", relPath)
	}
	return filepath.Join(b.dir, clean), nil
}

// SafeName macht aus einem beliebigen Namen einen brauchbaren Datei- oder
// Ordnernamen.
//
// Belegnummern und Originaldateinamen kommen aus der Welt draußen. Ein Name mit
// einem Schrägstrich legte die Datei in einem anderen Ordner ab, einer mit „..“
// außerhalb des Exports; und Zeichen, die Windows verbietet, machten den
// Datenträger dort unlesbar, wo er meistens gelesen wird.
func SafeName(name string) string {
	replaced := strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', 0:
			return '_'
		}
		if r < 0x20 {
			return '_'
		}
		return r
	}, name)
	replaced = strings.Trim(replaced, " .")
	if replaced == "" {
		return "unbenannt"
	}
	return replaced
}
