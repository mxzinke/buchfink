package inbox

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrOutside heißt: die Datei liegt nicht im Eingang oder schon in
// Übernommen/.
var ErrOutside = errors.New("die Datei liegt nicht im Eingang")

// Contains meldet, ob ein Pfad im Eingang liegt und nicht in Übernommen/.
// Der Ordner auf dem Weg wird aufgelöst, das letzte Glied nicht: ein Symlink
// im Eingang ist eine Datei im Eingang, nicht die, auf die er zeigt.
func Contains(dir, path string) bool {
	_, rel, ok := locate(dir, path)
	return ok && rel != DoneDir && !strings.HasPrefix(rel, DoneDir+string(filepath.Separator))
}

// locate löst Eingang und Elternordner des Pfads auf und liefert den
// aufgelösten Pfad und seinen Weg relativ zum Eingang.
func locate(dir, path string) (string, string, bool) {
	if strings.TrimSpace(dir) == "" || !filepath.IsAbs(path) {
		return "", "", false
	}
	root := resolve(dir)
	clean := filepath.Clean(path)
	resolved := filepath.Join(resolve(filepath.Dir(clean)), filepath.Base(clean))
	rel, err := filepath.Rel(root, resolved)
	if err != nil || rel == "." || rel == ".." || filepath.IsAbs(rel) ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", false
	}
	return resolved, rel, true
}

func resolve(dir string) string {
	dir = filepath.Clean(dir)
	if r, err := filepath.EvalSymlinks(dir); err == nil {
		return r
	}
	return dir
}

// Settle verschiebt eine abgelegte Datei aus dem Eingang nach
// Übernommen/<JJJJ-MM>/ (Monat von day) und liefert den neuen Pfad, unter dem
// Eingang, wie er eingestellt ist. Gibt es den Namen dort schon, bekommt die
// Datei ein Suffix: „Rechnung (2).pdf".
func Settle(dir, path string, day time.Time) (string, error) {
	if !Contains(dir, path) {
		return "", ErrOutside
	}
	resolved, _, _ := locate(dir, path)
	info, err := os.Lstat(resolved)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s ist keine gewöhnliche Datei", info.Name())
	}
	month := day.Format("2006-01")
	target := filepath.Join(resolve(dir), DoneDir, month)
	if err := os.MkdirAll(target, 0o755); err != nil {
		return "", err
	}
	name, err := uniqueName(target, info.Name())
	if err != nil {
		return "", err
	}
	if err := os.Rename(resolved, filepath.Join(target, name)); err != nil {
		return "", err
	}
	return filepath.Join(filepath.Clean(dir), DoneDir, month, name), nil
}

// Hash ist die Prüfsumme einer Datei bis MaxFileBytes.
func Hash(path string) (string, error) {
	data, err := readFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// Skipped ist eine Datei, die Add nicht übernommen hat, mit Grund.
type Skipped struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// Add kopiert Dateien in den Eingang: nur Belegformate (PDF, Bild, XML) bis
// 50 MB, deren Inhalt zur Endung passt. Ein vorhandener Name bekommt ein
// Suffix. Geliefert werden die Namen im Eingang und die übergangenen Dateien.
func Add(dir string, paths []string) ([]string, []Skipped, error) {
	added, skipped := []string{}, []Skipped{}
	if strings.TrimSpace(dir) == "" {
		return added, skipped, errors.New("es ist kein Eingang eingerichtet")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return added, skipped, fmt.Errorf("der Eingang %s ließ sich nicht anlegen: %w", dir, err)
	}
	for _, path := range paths {
		name, reason := addOne(dir, path)
		if reason != "" {
			skipped = append(skipped, Skipped{Name: filepath.Base(path), Reason: reason})
			continue
		}
		added = append(added, name)
	}
	return added, skipped, nil
}

func addOne(dir, path string) (string, string) {
	path = strings.TrimSpace(path)
	if path == "" || !filepath.IsAbs(path) {
		return "", "kein absoluter Pfad"
	}
	name := filepath.Base(path)
	info, err := os.Stat(path)
	switch {
	case err != nil:
		return "", "nicht lesbar"
	case info.IsDir():
		return "", "ein Ordner, keine Datei"
	case !info.Mode().IsRegular():
		return "", "keine gewöhnliche Datei"
	case strings.HasPrefix(name, "."):
		return "", "versteckte Datei"
	case info.Size() == 0:
		return "", "leer"
	case info.Size() > MaxFileBytes:
		return "", fmt.Sprintf("größer als %d MB", MaxFileBytes>>20)
	}
	format, ok := FormatOf(name)
	if !ok {
		return "", "kein Beleg (PDF, Bild oder XML)"
	}
	if within(resolve(dir), resolve(path)) {
		return "", "liegt schon im Eingang"
	}
	data, err := readFile(path)
	if err != nil {
		return "", "nicht lesbar"
	}
	if !ContentMatches(format, data) {
		return "", "der Inhalt passt nicht zur Endung"
	}
	tmp, err := os.CreateTemp(dir, "."+name+".*")
	if err != nil {
		return "", "der Eingang ist nicht beschreibbar"
	}
	_, writeErr := tmp.Write(data)
	if err := errors.Join(writeErr, tmp.Close()); err != nil {
		_ = os.Remove(tmp.Name())
		return "", "der Eingang ist nicht beschreibbar"
	}
	final, err := uniqueName(dir, name)
	if err == nil {
		err = os.Rename(tmp.Name(), filepath.Join(dir, final))
	}
	if err != nil {
		_ = os.Remove(tmp.Name())
		return "", "der Eingang ist nicht beschreibbar"
	}
	return final, ""
}

// uniqueName ist name, wenn der Ordner ihn noch nicht hat, sonst
// „Name (2).pdf", „Name (3).pdf" …
func uniqueName(dir, name string) (string, error) {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 1; i <= 9999; i++ {
		candidate := name
		if i > 1 {
			candidate = fmt.Sprintf("%s (%d)%s", stem, i, ext)
		}
		if _, err := os.Lstat(filepath.Join(dir, candidate)); errors.Is(err, fs.ErrNotExist) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("für %s ist kein freier Name mehr da", name)
}

func within(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil || rel == "." || filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
