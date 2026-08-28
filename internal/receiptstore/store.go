// Package receiptstore keeps Belegdateien on disk, addressed by their content.
//
// Two decisions shape it. First, the file name on disk is the SHA256 of the
// content, not the Belegnummer: if the number were in the path it would have to
// be fixed before the first file is written — that is, before the transaction
// that inserts the Beleg — and a failed write would tear a gap into the number
// range. Content addressing decouples the two, and identical files happen to end
// up stored once.
//
// Second, Belegdateien are *not* encrypted. GoBD requires incoming documents to
// be kept unchanged in the form they were received. Only the path and the file
// name are encrypted, and those live in the database.
package receiptstore

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

const (
	dirPerm  = 0o700
	filePerm = 0o600
	// maxSniff is what http.DetectContentType looks at.
	maxSniff = 512
)

// Store writes and reads Belegdateien below a data directory.
type Store struct {
	dataDir string
}

// New creates a store rooted at a tenant's data directory.
func New(dataDir string) *Store { return &Store{dataDir: dataDir} }

// StoredFile is the result of filing one file away.
type StoredFile struct {
	// SHA256 of the content, lowercase hex.
	SHA256 string
	// RelPath is relative to the data directory, always with forward slashes so
	// the value is stable across operating systems.
	RelPath  string
	Size     int64
	MimeType string
	// Deduplicated reports that identical content was already stored and reused.
	Deduplicated bool
}

// Put stores the content of src under its own digest and returns where it landed.
//
// The write goes to a temporary file in the target directory, is flushed to disk
// and then renamed into place. An interrupted write therefore never leaves a
// half-written Beleg behind — it leaves a temporary file that belongs to nothing.
func (s *Store) Put(fiscalYear int, direction domain.Direction, originalName string, src io.Reader) (*StoredFile, error) {
	subDir, err := directorySegment(direction)
	if err != nil {
		return nil, err
	}
	return s.putInto([]string{"belege", fmt.Sprintf("%d", fiscalYear), subDir}, originalName, src)
}

// PutDocument stores a file that is not a Beleg.
//
// Ein Vertrag, ein Gutachten, ein Zulassungspapier gehören zu einem Anlagegut,
// aber nicht in den Belegkreis: sie tragen keine Belegnummer, sie werden nicht
// gebucht, und sie hängen an keinem Geschäftsjahr — ein Kaufvertrag erklärt die
// Anschaffung noch, wenn das Wirtschaftsgut zehn Jahre im Bestand ist. Der
// Ablageweg ist trotzdem derselbe: inhaltsadressiert, dedupliziert, mit
// atomarem Umbenennen. Zwei Speicher für zwei Dateiarten wären zwei Stellen,
// an denen dasselbe schiefgehen kann.
func (s *Store) PutDocument(category, originalName string, src io.Reader) (*StoredFile, error) {
	if err := checkSegment(category); err != nil {
		return nil, err
	}
	return s.putInto([]string{"dokumente", category}, originalName, src)
}

// PutDocumentPath stores a document from the local file system.
func (s *Store) PutDocumentPath(category, path string) (*StoredFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Datei %s konnte nicht gelesen werden: %w", filepath.Base(path), err)
	}
	defer f.Close()
	return s.PutDocument(category, filepath.Base(path), f)
}

// checkSegment refuses a path segment that could leave the data directory. Der
// Aufrufer gibt eine Kategorie vor, keinen Pfad.
func checkSegment(segment string) error {
	if segment == "" {
		return fmt.Errorf("ohne Kategorie lässt sich die Datei nicht ablegen")
	}
	if strings.ContainsAny(segment, `/\`) || segment == "." || segment == ".." {
		return fmt.Errorf("%q ist keine gültige Ablagekategorie", segment)
	}
	return nil
}

// putInto is the shared body: hash while writing, name the file after its own
// digest, rename it into place.
func (s *Store) putInto(segments []string, originalName string, src io.Reader) (*StoredFile, error) {
	if s.dataDir == "" {
		return nil, fmt.Errorf("kein Datenordner konfiguriert")
	}

	absDir := filepath.Join(append([]string{s.dataDir}, segments...)...)
	if err := os.MkdirAll(absDir, dirPerm); err != nil {
		return nil, fmt.Errorf("Ablageordner konnte nicht angelegt werden: %w", err)
	}

	tmp, err := os.CreateTemp(absDir, ".upload-*")
	if err != nil {
		return nil, fmt.Errorf("Die Datei konnte nicht angelegt werden: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpName) // no-op once the rename succeeded
	}()
	if err := tmp.Chmod(filePerm); err != nil {
		return nil, fmt.Errorf("Die Rechte der Datei konnten nicht gesetzt werden: %w", err)
	}

	digest := sha256.New()
	head := make([]byte, 0, maxSniff)
	size, err := copyHashingAndSniffing(tmp, digest, &head, src)
	if err != nil {
		return nil, fmt.Errorf("Die Datei konnte nicht geschrieben werden: %w", err)
	}
	if size == 0 {
		return nil, fmt.Errorf("die Datei %s ist leer", originalName)
	}
	if err := tmp.Sync(); err != nil {
		return nil, fmt.Errorf("Die Datei konnte nicht gesichert werden: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return nil, fmt.Errorf("Die Datei konnte nicht geschlossen werden: %w", err)
	}

	sum := hex.EncodeToString(digest.Sum(nil))
	mimeType := detectMime(head, originalName)
	fileName := sum + extensionFor(originalName, mimeType)
	absPath := filepath.Join(absDir, fileName)
	relPath := filepath.ToSlash(filepath.Join(append(append([]string{}, segments...), fileName)...))

	// Identical content is stored once. Two Belege may legitimately share a file,
	// and so may two Anlagegüter share one contract — re-filing the same scan must
	// not double the disk usage.
	if existing, err := os.Stat(absPath); err == nil {
		if existing.Size() == size {
			return &StoredFile{SHA256: sum, RelPath: relPath, Size: size, MimeType: mimeType, Deduplicated: true}, nil
		}
		// Same digest, different length: that is a broken file on disk, not a
		// collision. Overwrite it rather than trust it.
	}

	if err := os.Rename(tmpName, absPath); err != nil {
		return nil, fmt.Errorf("Die Datei konnte nicht abgelegt werden: %w", err)
	}
	return &StoredFile{SHA256: sum, RelPath: relPath, Size: size, MimeType: mimeType}, nil
}

// PutPath stores a file from the local file system, keeping its base name as the
// original name.
func (s *Store) PutPath(fiscalYear int, direction domain.Direction, path string) (*StoredFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Datei %s konnte nicht gelesen werden: %w", filepath.Base(path), err)
	}
	defer f.Close()
	return s.Put(fiscalYear, direction, filepath.Base(path), f)
}

// Read returns the content of a stored file.
func (s *Store) Read(relPath string) ([]byte, error) {
	abs, err := s.resolve(relPath)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("Belegdatei konnte nicht gelesen werden: %w", err)
	}
	return data, nil
}

// Verify recomputes the digest of a stored file and compares it to the expected
// one. It is what turns "the hash is in the database" into "the file on disk is
// still the one that was booked".
func (s *Store) Verify(relPath, expectedSHA256 string) error {
	data, err := s.Read(relPath)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != expectedSHA256 {
		return fmt.Errorf("die Belegdatei %s stimmt nicht mehr mit ihrer Prüfsumme überein", relPath)
	}
	return nil
}

// Exists reports whether a stored file is still present.
func (s *Store) Exists(relPath string) bool {
	abs, err := s.resolve(relPath)
	if err != nil {
		return false
	}
	info, err := os.Stat(abs)
	return err == nil && !info.IsDir()
}

// Delete removes a stored file.
//
// Für Belege gibt es das nicht: ein empfangenes Dokument verschwindet nicht,
// es wird verworfen und bleibt liegen. Für ein Dokument am Anlagegut schon —
// wer eine falsche Datei hochgeladen hat, soll sie wieder loswerden. Der
// Aufrufer prüft vorher, dass niemand sonst auf sie zeigt; identische Inhalte
// werden hier nur einmal gespeichert.
func (s *Store) Delete(relPath string) error {
	abs, err := s.resolve(relPath)
	if err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("die Datei konnte nicht entfernt werden: %w", err)
	}
	return nil
}

// resolve turns a stored relative path into an absolute one, refusing anything
// that would escape the data directory.
func (s *Store) resolve(relPath string) (string, error) {
	if s.dataDir == "" {
		return "", fmt.Errorf("kein Datenordner konfiguriert")
	}
	clean := filepath.Clean(filepath.FromSlash(relPath))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("ungültiger Belegpfad %q", relPath)
	}
	return filepath.Join(s.dataDir, clean), nil
}

func directorySegment(direction domain.Direction) (string, error) {
	switch direction {
	case domain.DirectionIncoming:
		return "eingang", nil
	case domain.DirectionOutgoing:
		return "ausgang", nil
	default:
		return "", fmt.Errorf("unbekannte Belegrichtung %q", direction)
	}
}

// copyHashingAndSniffing streams src to dst while feeding the digest and keeping
// the first bytes for content-type detection.
func copyHashingAndSniffing(dst io.Writer, digest io.Writer, head *[]byte, src io.Reader) (int64, error) {
	var total int64
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			if len(*head) < maxSniff {
				room := maxSniff - len(*head)
				if room > n {
					room = n
				}
				*head = append(*head, chunk[:room]...)
			}
			if _, werr := dst.Write(chunk); werr != nil {
				return total, werr
			}
			if _, werr := digest.Write(chunk); werr != nil {
				return total, werr
			}
			total += int64(n)
		}
		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			return total, err
		}
	}
}

// detectMime prefers what the content says over what the name claims, and falls
// back to the extension when sniffing yields nothing useful. A receipt renamed to
// .txt is still a PDF.
func detectMime(head []byte, originalName string) string {
	sniffed := stripMimeParams(http.DetectContentType(head))
	byExtension := stripMimeParams(mime.TypeByExtension(strings.ToLower(filepath.Ext(originalName))))

	switch sniffed {
	case "application/octet-stream", "text/plain", "":
		if byExtension != "" {
			return byExtension
		}
	case "text/xml":
		// An e-invoice is XML; naming it application/xml keeps one spelling.
		return "application/xml"
	}
	if sniffed == "" {
		return "application/octet-stream"
	}
	return sniffed
}

func stripMimeParams(v string) string {
	if i := strings.IndexByte(v, ';'); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}

// extensionFor keeps the original suffix on the stored file.
//
// The concept says the file name on disk is the checksum. Keeping the extension
// as well is a deliberate deviation: an auditor who opens the Belegordner outside
// Buchfink otherwise finds a directory of typeless blobs.
func extensionFor(originalName, mimeType string) string {
	ext := strings.ToLower(filepath.Ext(originalName))
	if ext != "" && len(ext) <= 8 && isSafeExtension(ext) {
		return ext
	}
	if exts, err := mime.ExtensionsByType(mimeType); err == nil && len(exts) > 0 {
		return exts[0]
	}
	return ".bin"
}

func isSafeExtension(ext string) bool {
	for _, c := range ext[1:] {
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		default:
			return false
		}
	}
	return len(ext) > 1
}
