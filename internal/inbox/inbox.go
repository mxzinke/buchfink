// Package inbox ist der Eingang eines Mandanten: der eine Ordner, in den
// Belege, E-Rechnungen und Kontoauszüge kommen, bevor Buchfink sie ablegt.
// Abgelegtes wandert nach Übernommen/<JJJJ-MM>; Buchfink löscht im Eingang
// nichts.
package inbox

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/einvoice"
)

// DoneDir nimmt im Eingang auf, was Buchfink abgelegt hat.
const DoneDir = "Übernommen"

// Kind ist die Art einer Datei im Eingang.
type Kind string

const (
	KindReceipt     Kind = "beleg"
	KindEInvoice    Kind = "e_rechnung"
	KindStatement   Kind = "kontoauszug"
	KindUnsupported Kind = "nicht_unterstuetzt"
)

// camtStatement ist der Namensraum eines Kontoauszugs nach ISO 20022
// (camt.053, alle Fassungen).
const camtStatement = "urn:iso:std:iso:20022:tech:xsd:camt.053."

// File ist eine Datei im Eingang.
type File struct {
	Path    string
	Name    string
	Size    int64
	ModTime time.Time
	Kind    Kind
	// SHA256 ist die Prüfsumme des Inhalts; leer bei einer Datei, die
	// Buchfink nicht ablegen kann.
	SHA256 string
}

// Skip meldet für einen aufgelösten Ordner im Eingang, ob List ihn mit
// allem darin auslässt, etwa eine Sicherung oder den Datenordner eines
// Mandanten.
type Skip func(dir string) bool

// List liest den Eingang mit Unterordnern, ohne Übernommen/, versteckte
// Dateien und Ordner, Symlinks und die Ordner, die skip nennt, die älteste
// Datei zuerst. Ein fehlender Eingang ist leer. Ist der Eingang selbst ein
// Symlink, liest List den Ordner, auf den er zeigt; die Pfade stehen trotzdem
// unter dem Eingang, wie er eingestellt ist.
func List(dir string, skip Skip) ([]File, error) {
	out := []File{}
	err := walk(dir, skip, func(path, real string, info fs.FileInfo) {
		f := describe(real, info)
		f.Path = path
		out = append(out, f)
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].ModTime.Equal(out[j].ModTime) {
			return out[i].ModTime.Before(out[j].ModTime)
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}

// Count zählt dieselben Dateien wie List, ohne sie zu lesen, und nennt den
// Zeitpunkt der ältesten.
func Count(dir string, skip Skip) (int, time.Time, error) {
	n, oldest := 0, time.Time{}
	err := walk(dir, skip, func(_, _ string, info fs.FileInfo) {
		n++
		if oldest.IsZero() || info.ModTime().Before(oldest) {
			oldest = info.ModTime()
		}
	})
	return n, oldest, err
}

// walk ruft fn für jede Datei des Eingangs, die List zeigt, mit dem Pfad unter
// dem eingestellten Eingang und dem aufgelösten.
func walk(dir string, skip Skip, fn func(path, real string, info fs.FileInfo)) error {
	if strings.TrimSpace(dir) == "" {
		return nil
	}
	shown := filepath.Clean(dir)
	root := resolve(shown)
	if _, err := os.Stat(root); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	done := filepath.Join(root, DoneDir)
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				return err
			}
			return nil
		}
		if path == root {
			return nil
		}
		if d.IsDir() && (strings.HasPrefix(d.Name(), ".") || path == done || (skip != nil && skip(path))) {
			return filepath.SkipDir
		}
		if strings.HasPrefix(d.Name(), ".") || d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		if info, err := d.Info(); err == nil {
			fn(filepath.Join(shown, rel), path, info)
		}
		return nil
	})
}

// Oldest ist der Zeitpunkt der ältesten Datei; null ohne Dateien.
func Oldest(files []File) time.Time {
	var oldest time.Time
	for _, f := range files {
		if oldest.IsZero() || f.ModTime.Before(oldest) {
			oldest = f.ModTime
		}
	}
	return oldest
}

// AgeDays sind die vollen Tage seit einem Zeitpunkt, etwa dem der ältesten
// Datei; null ohne Zeitpunkt.
func AgeDays(oldest, now time.Time) int {
	if oldest.IsZero() || now.Before(oldest) {
		return 0
	}
	return int(now.Sub(oldest) / (24 * time.Hour))
}

func describe(path string, info fs.FileInfo) File {
	f := File{Path: path, Name: info.Name(), Size: info.Size(), ModTime: info.ModTime(), Kind: KindUnsupported}
	format, ok := FormatOf(info.Name())
	if !ok || info.Size() == 0 || info.Size() > MaxFileBytes {
		return f
	}
	data, err := readFile(path)
	if err != nil || !ContentMatches(format, data) {
		return f
	}
	f.Kind = classify(format, data)
	if f.Kind != KindUnsupported {
		sum := sha256.Sum256(data)
		f.SHA256 = hex.EncodeToString(sum[:])
	}
	return f
}

func readFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, MaxFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxFileBytes {
		return nil, errors.New("zu groß")
	}
	return data, nil
}

// Classify bestimmt die Art einer Datei aus Format und Inhalt: eine
// E-Rechnung am XRechnung- oder ZUGFeRD-Datensatz, einen Kontoauszug am
// Namensraum von camt.053.
func Classify(name string, data []byte) Kind {
	format, ok := FormatOf(name)
	if !ok || !ContentMatches(format, data) {
		return KindUnsupported
	}
	return classify(format, data)
}

func classify(format Format, data []byte) Kind {
	switch format {
	case FormatImage:
		return KindReceipt
	case FormatPDF:
		if embedded, err := einvoice.ExtractFromPDF(data); err == nil {
			if _, err := einvoice.DetectSyntax(embedded.Data); err == nil {
				return KindEInvoice
			}
		}
		return KindReceipt
	case FormatXML:
		if _, err := einvoice.DetectSyntax(data); err == nil {
			return KindEInvoice
		}
		if strings.HasPrefix(rootNamespace(data), camtStatement) {
			return KindStatement
		}
	}
	return KindUnsupported
}

// rootNamespace ist der Namensraum des Wurzelelements.
func rootNamespace(data []byte) string {
	decoder := xml.NewDecoder(bytes.NewReader(bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})))
	// Gebraucht wird nur der Namensraum, und der ist ASCII; eine andere
	// Kodierung als UTF-8 darf ihn nicht verdecken.
	decoder.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) { return input, nil }
	for {
		token, err := decoder.Token()
		if err != nil {
			return ""
		}
		if start, ok := token.(xml.StartElement); ok {
			return start.Name.Space
		}
	}
}
