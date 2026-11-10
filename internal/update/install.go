package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Installation ist das laufende Programm auf der Platte: unter macOS das
// App-Paket, sonst die Programmdatei.
type Installation struct {
	// Path ist das Paket oder die Datei, die ein Update ersetzt.
	Path   string
	Bundle bool
}

// oldSuffix trägt die ersetzte Fassung, bis der nächste Start sie entfernt.
// Unter Windows lässt sich eine laufende Programmdatei umbenennen, aber nicht
// löschen.
const oldSuffix = ".alt"

// stagePrefix beginnt jeden Zwischenstand eines Updates im Installationsordner.
const stagePrefix = ".buchfink-update-"

// bundleBinary ist der Ort der Programmdatei in einem App-Paket.
const bundleBinary = "Contents/MacOS"

// Locate findet die Installation zur laufenden Programmdatei.
func Locate(goos string) (Installation, error) {
	exe, err := os.Executable()
	if err != nil {
		return Installation{}, fmt.Errorf("die laufende Programmdatei ließ sich nicht bestimmen: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return installationOf(goos, exe)
}

func installationOf(goos, exe string) (Installation, error) {
	if goos != "darwin" {
		return Installation{Path: exe}, nil
	}
	// Ein heruntergeladenes Paket, das nie verschoben wurde, startet macOS aus
	// einer schreibgeschützten Kopie mit zufälligem Pfad.
	if strings.Contains(exe, "/AppTranslocation/") {
		return Installation{}, fmt.Errorf("Buchfink läuft aus dem Download heraus. Verschieben Sie die App in den " +
			"Ordner „Programme“ und starten Sie sie von dort")
	}
	bundle := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
	if !strings.HasSuffix(bundle, ".app") || filepath.Base(filepath.Dir(filepath.Dir(exe))) != "Contents" {
		return Installation{}, fmt.Errorf("Buchfink läuft nicht aus einem App-Paket")
	}
	return Installation{Path: bundle, Bundle: true}, nil
}

// dir ist der Ordner, in dem die Installation liegt und in dem ein Update
// seinen Zwischenstand ablegt: derselbe Datenträger, damit der Tausch ein
// Umbenennen bleibt.
func (i Installation) dir() string { return filepath.Dir(i.Path) }

// checkWritable prüft, ob sich im Installationsordner schreiben lässt.
func (i Installation) checkWritable() error {
	probe, err := os.CreateTemp(i.dir(), stagePrefix+"*")
	if err != nil {
		return fmt.Errorf("im Ordner %s darf Buchfink nicht schreiben", i.dir())
	}
	probe.Close()
	return os.Remove(probe.Name())
}

// Cleanup entfernt, was ein früheres Update zurückgelassen hat: die ersetzte
// Fassung und abgebrochene Zwischenstände.
func (i Installation) Cleanup() {
	_ = os.RemoveAll(i.Path + oldSuffix)
	i.removeStages()
}

// removeStages entfernt die Zwischenstände eines Updates aus dem
// Installationsordner.
func (i Installation) removeStages() {
	entries, err := os.ReadDir(i.dir())
	if err != nil {
		return
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), stagePrefix) {
			_ = os.RemoveAll(filepath.Join(i.dir(), entry.Name()))
		}
	}
}

// download lädt die Datei eines Release in den Installationsordner und prüft
// sie gegen die Summe aus dem Release. progress bekommt die geladenen Bytes.
func (s *Service) download(ctx context.Context, release *Release, dir string, progress func(done int64)) (string, error) {
	want, err := s.expectedSum(ctx, release, release.asset.name)
	if err != nil {
		return "", err
	}
	resp, err := s.get(ctx, release.asset.url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	file, err := os.CreateTemp(dir, stagePrefix+"*")
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(file, hash), &progressReader{
		reader: io.LimitReader(resp.Body, maxAssetBytes+1), report: progress})
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	switch {
	case err != nil:
		err = fmt.Errorf("der Download wurde nicht vollständig: %w", err)
	case written > maxAssetBytes:
		err = fmt.Errorf("die Datei ist größer als erwartet")
	case hex.EncodeToString(hash.Sum(nil)) != want:
		err = fmt.Errorf("die Prüfsumme von %s stimmt nicht mit dem Release überein; die Datei wurde verworfen",
			release.asset.name)
	}
	if err != nil {
		os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

type progressReader struct {
	reader io.Reader
	done   int64
	report func(int64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.reader.Read(b)
	p.done += int64(n)
	if p.report != nil && n > 0 {
		p.report(p.done)
	}
	return n, err
}

// stage legt aus der geladenen Datei bereit, was die Installation ersetzt:
// die Programmdatei oder das App-Paket. Der Rückgabewert liegt im
// Installationsordner.
func (i Installation) stage(archive, assetName string) (string, error) {
	switch {
	case strings.HasSuffix(assetName, ".zip"):
		if !i.Bundle {
			return "", fmt.Errorf("%s enthält ein App-Paket, installiert ist eine Programmdatei", assetName)
		}
		return stageBundle(archive, i.dir(), filepath.Base(i.Path))
	case i.Bundle:
		return "", fmt.Errorf("%s enthält kein App-Paket", assetName)
	case strings.HasSuffix(assetName, ".tar.gz"):
		return stageFromTar(archive, i.dir())
	default:
		if err := os.Chmod(archive, 0o755); err != nil {
			return "", err
		}
		return archive, nil
	}
}

// tarBinary ist der Name der Programmdatei im Linux-Paket.
const tarBinary = "buchfink"

func stageFromTar(archive, dir string) (string, error) {
	in, err := os.Open(archive)
	if err != nil {
		return "", err
	}
	defer in.Close()
	unzipped, err := gzip.NewReader(in)
	if err != nil {
		return "", fmt.Errorf("das Paket ist nicht lesbar: %w", err)
	}
	reader := tar.NewReader(unzipped)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return "", fmt.Errorf("das Paket enthält keine Programmdatei %q", tarBinary)
		}
		if err != nil {
			return "", fmt.Errorf("das Paket ist nicht lesbar: %w", err)
		}
		if header.Typeflag != tar.TypeReg || filepath.Clean(header.Name) != tarBinary {
			continue
		}
		out, err := os.CreateTemp(dir, stagePrefix+"*")
		if err != nil {
			return "", err
		}
		_, err = io.Copy(out, io.LimitReader(reader, maxAssetBytes))
		if closeErr := out.Close(); err == nil {
			err = closeErr
		}
		if err == nil {
			err = os.Chmod(out.Name(), 0o755)
		}
		if err != nil {
			os.Remove(out.Name())
			return "", err
		}
		return out.Name(), nil
	}
}

// stageBundle entpackt das App-Paket aus dem Archiv. Das Archiv enthält genau
// einen Ordner „<name>.app"; Einträge daneben (__MACOSX) bleiben liegen.
//
// ditto legt erweiterte Attribute als Begleitdateien „._<name>" neben jeden
// Eintrag. Als lose Dateien im Paket verletzten sie das Siegel seiner
// Signatur; sie bleiben deshalb draußen.
func stageBundle(archive, dir, bundleName string) (string, error) {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return "", fmt.Errorf("das Paket ist nicht lesbar: %w", err)
	}
	defer reader.Close()
	root, err := os.MkdirTemp(dir, stagePrefix+"*")
	if err != nil {
		return "", err
	}
	top := ""
	for _, entry := range reader.File {
		name := filepath.ToSlash(filepath.Clean(entry.Name))
		first, _, _ := strings.Cut(name, "/")
		if filepath.IsLocal(name) && (!strings.HasSuffix(first, ".app") || strings.HasPrefix(filepath.Base(name), "._")) {
			continue
		}
		if top == "" {
			top = first
		}
		if first != top || !filepath.IsLocal(name) {
			os.RemoveAll(root)
			return "", fmt.Errorf("das Paket hat einen unerwarteten Aufbau (%s)", entry.Name)
		}
		if err := extractEntry(entry, filepath.Join(root, filepath.FromSlash(name))); err != nil {
			os.RemoveAll(root)
			return "", err
		}
	}
	staged := filepath.Join(root, top)
	if info, err := os.Stat(filepath.Join(staged, bundleBinary)); top == "" || err != nil || !info.IsDir() {
		os.RemoveAll(root)
		return "", fmt.Errorf("das Paket enthält kein App-Paket für %s", bundleName)
	}
	return staged, nil
}

func extractEntry(entry *zip.File, target string) error {
	mode := entry.Mode()
	switch {
	case mode.IsDir():
		return os.MkdirAll(target, 0o755)
	case mode&os.ModeSymlink != 0:
		// Ein App-Paket enthält Verweise innerhalb seiner Frameworks. Ein Ziel
		// außerhalb des Pakets gehört nicht hinein.
		in, err := entry.Open()
		if err != nil {
			return err
		}
		link, err := io.ReadAll(io.LimitReader(in, 4096))
		in.Close()
		if err != nil {
			return err
		}
		if filepath.IsAbs(string(link)) || !filepath.IsLocal(filepath.Join(filepath.Dir(entry.Name), string(link))) {
			return fmt.Errorf("das Paket verweist aus sich heraus (%s)", entry.Name)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.Symlink(string(link), target)
	case !mode.IsRegular():
		return fmt.Errorf("das Paket enthält einen unerwarteten Eintrag (%s)", entry.Name)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	in, err := entry.Open()
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode.Perm()|0o600)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, io.LimitReader(in, maxAssetBytes))
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	return err
}

// swap setzt staged an die Stelle der Installation. Die bisherige Fassung
// bleibt daneben liegen, bis Cleanup sie entfernt; scheitert der zweite
// Schritt, kommt sie zurück.
func (i Installation) swap(staged string) error {
	old := i.Path + oldSuffix
	if err := os.RemoveAll(old); err != nil {
		return fmt.Errorf("die Reste der vorigen Fassung ließen sich nicht entfernen: %w", err)
	}
	if err := os.Rename(i.Path, old); err != nil {
		return fmt.Errorf("die laufende Fassung ließ sich nicht beiseitelegen: %w", err)
	}
	if err := os.Rename(staged, i.Path); err != nil {
		if back := os.Rename(old, i.Path); back != nil {
			return fmt.Errorf("die neue Fassung ließ sich nicht einsetzen (%v) und die bisherige liegt unter %s: %w",
				err, old, back)
		}
		return fmt.Errorf("die neue Fassung ließ sich nicht einsetzen: %w", err)
	}
	return nil
}
