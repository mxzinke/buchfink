package inbox

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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
// Datei ein Suffix: „Rechnung (2).pdf". Verschoben wird nur innerhalb des
// Eingangs: Übernommen/ und der Monatsordner müssen echte Ordner sein, ein
// Symlink an ihrer Stelle führt sonst aus dem Eingang hinaus.
func Settle(dir, path string, day time.Time) (string, error) {
	if !Contains(dir, path) {
		return "", ErrOutside
	}
	_, rel, _ := locate(dir, path)
	root, err := os.OpenRoot(resolve(dir))
	if err != nil {
		return "", err
	}
	defer root.Close()
	info, err := root.Lstat(rel)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s ist keine gewöhnliche Datei", info.Name())
	}
	month := day.Format("2006-01")
	target := filepath.Join(DoneDir, month)
	for _, d := range []string{DoneDir, target} {
		if err := ensureDir(root, d); err != nil {
			return "", err
		}
	}
	name, err := place(root, rel, target, info.Name())
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Clean(dir), DoneDir, month, name), nil
}

// ensureDir legt einen Ordner im Eingang an, falls er fehlt. Steht dort ein
// Symlink oder eine Datei, gibt es einen Fehler.
func ensureDir(root *os.Root, name string) error {
	info, err := root.Lstat(name)
	if errors.Is(err, fs.ErrNotExist) {
		if err := root.Mkdir(name, 0o755); err != nil && !errors.Is(err, fs.ErrExist) {
			return err
		}
		info, err = root.Lstat(name)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s im Eingang ist kein Ordner", name)
	}
	return nil
}

// placing hält Buchfink davon ab, zwei Dateien zugleich unter denselben Namen
// zu legen; Ablagen aus App und Agenten laufen nebeneinander.
var placing sync.Mutex

// place legt die Datei src unter dir mit ihrem Namen oder dem nächsten
// freien Suffix ab und liefert den Namen. Ein vorhandenes Ziel bleibt
// unberührt: der harte Verweis scheitert dann, und der nächste Name kommt
// dran. Dateisysteme ohne harte Verweise (FAT, manche Netzlaufwerke)
// bekommen ein Umbenennen, nachdem der Name frei war.
func place(root *os.Root, src, dir, name string) (string, error) {
	placing.Lock()
	defer placing.Unlock()
	for i := 1; i <= 9999; i++ {
		candidate := numbered(name, i)
		dst := filepath.Join(dir, candidate)
		err := root.Link(src, dst)
		switch {
		case err == nil:
			if err := root.Remove(src); err != nil {
				_ = root.Remove(dst)
				return "", err
			}
			return candidate, nil
		case errors.Is(err, fs.ErrExist):
			continue
		}
		if _, err := root.Lstat(dst); !errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err := root.Rename(src, dst); err != nil {
			return "", err
		}
		return candidate, nil
	}
	return "", fmt.Errorf("für %s ist kein freier Name mehr da", name)
}

// numbered ist name für i = 1, sonst „Name (i).pdf".
func numbered(name string, i int) string {
	if i == 1 {
		return name
	}
	ext := filepath.Ext(name)
	return fmt.Sprintf("%s (%d)%s", strings.TrimSuffix(name, ext), i, ext)
}

// Settled sucht eine Datei des Eingangs, die es unter ihrem Pfad nicht mehr
// gibt, in Übernommen/: den Pfad im jüngsten Monat mit demselben Namen, oder
// "". Wer nach der Ablage den alten Pfad nennt, erfährt so den neuen.
func Settled(dir, path string) string {
	if strings.TrimSpace(dir) == "" || !filepath.IsAbs(path) {
		return ""
	}
	clean, root := filepath.Clean(path), filepath.Clean(dir)
	rel, err := filepath.Rel(root, clean)
	if err != nil || !within(root, clean) || rel == DoneDir || strings.HasPrefix(rel, DoneDir+string(filepath.Separator)) {
		return ""
	}
	matches, _ := filepath.Glob(filepath.Join(root, DoneDir, "*", filepath.Base(clean)))
	sort.Strings(matches)
	if len(matches) == 0 {
		return ""
	}
	return matches[len(matches)-1]
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
	root, err := os.OpenRoot(dir)
	if err != nil {
		return "", "der Eingang ist nicht beschreibbar"
	}
	defer root.Close()
	tmp, err := os.CreateTemp(dir, "."+name+".*")
	if err != nil {
		return "", "der Eingang ist nicht beschreibbar"
	}
	_, writeErr := tmp.Write(data)
	if err := errors.Join(writeErr, tmp.Close()); err != nil {
		_ = os.Remove(tmp.Name())
		return "", "der Eingang ist nicht beschreibbar"
	}
	final, err := place(root, filepath.Base(tmp.Name()), ".", name)
	if err != nil {
		_ = os.Remove(tmp.Name())
		return "", "der Eingang ist nicht beschreibbar"
	}
	return final, ""
}

func within(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil || rel == "." || filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
