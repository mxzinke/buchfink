package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/buchfink/buchfink/internal/declaration"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Declaration rechnet eine Jahressteuererklärung aus dem heutigen Stand.
func (t *Tenant) Declaration(ctx context.Context, kind declaration.Kind, year int) (*declaration.Declaration, error) {
	var d *declaration.Declaration
	err := t.Read(ctx, func(ctx context.Context, s *Services) (err error) {
		d, err = s.Declarations.Build(ctx, kind, year)
		return err
	})
	return d, err
}

// SetDeclarationInput legt eine Angabe zu einer Zeile ab. Ein Agent setzt mit
// Rückfrage beim Menschen; bei Zeilen, die dem Menschen vorbehalten sind,
// bleibt seine Angabe ein Vorschlag.
func (t *Tenant) SetDeclarationInput(ctx context.Context, req service.InputRequest) (*declaration.Input, error) {
	var in *declaration.Input
	err := t.Write(ctx, func(ctx context.Context, s *Services) (err error) {
		in, err = s.Declarations.SetInput(ctx, req)
		return err
	})
	return in, err
}

// ConfirmDeclarationInput bestätigt den Vorschlag eines Agenten. Nur in der
// App.
func (t *Tenant) ConfirmDeclarationInput(ctx context.Context, kind declaration.Kind, year int, key string) (*declaration.Input, error) {
	var in *declaration.Input
	err := t.Write(ctx, func(ctx context.Context, s *Services) (err error) {
		in, err = s.Declarations.ConfirmInput(ctx, kind, year, key)
		return err
	}, AppOnly())
	return in, err
}

// DeleteDeclarationInput nimmt eine Angabe zurück.
func (t *Tenant) DeleteDeclarationInput(ctx context.Context, kind declaration.Kind, year int, key string) error {
	return t.Write(ctx, func(ctx context.Context, s *Services) error {
		return s.Declarations.DeleteInput(ctx, kind, year, key)
	})
}

// DeclarationExport ist eine geschriebene Exportdatei.
type DeclarationExport struct {
	Path        string             `json:"pfad"`
	SHA256      string             `json:"sha256"`
	FiguresHash string             `json:"zahlen_hash"`
	Status      declaration.Status `json:"status"`
	Blocking    int                `json:"befunde_blockierend"`
	RecordID    uint               `json:"stand_id"`
}

// ExportDeclaration schreibt die Erklärung als JSON (Schema
// buchfink.erklaerung/1) in den Ordner und hält den Export fest: gerechnet
// im Lesen, die Datei ohne Sperre in eine Zwischendatei, Vermerk und
// Umbenennung in einem kurzen Schreibvorgang. Scheitert der Vermerk, bleibt
// keine Exportdatei liegen. Der Export ändert die Bücher nicht und läuft
// deshalb auch im Prüfermodus.
func (t *Tenant) ExportDeclaration(ctx context.Context, kind declaration.Kind, year int, dir string) (*DeclarationExport, error) {
	if !filepath.IsAbs(dir) {
		return nil, fmt.Errorf("der Exportordner %q ist kein absoluter Pfad", dir)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("der Exportordner %s ist nicht vorhanden", dir)
	}
	d, err := t.Declaration(ctx, kind, year)
	if err != nil {
		return nil, err
	}
	if d.Status == declaration.StatusDraft {
		d.Status = declaration.StatusExported
	}
	data, err := declaration.ExportJSON(d)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	sha := hex.EncodeToString(sum[:])
	path := filepath.Join(dir, fmt.Sprintf("erklaerung-%s-%d-%s.json", kind, year, d.FiguresHash[:12]))
	tmp, err := writeTempFile(dir, data)
	if err != nil {
		return nil, fmt.Errorf("die Exportdatei ließ sich nicht schreiben: %w", err)
	}
	placed := false
	var rec *domain.DeclarationRecord
	if err := t.Write(ctx, func(ctx context.Context, s *Services) (err error) {
		rec, err = s.Declarations.RecordExport(ctx, d, path, sha)
		if err != nil {
			return err
		}
		if err := os.Rename(tmp, path); err != nil {
			return fmt.Errorf("die Exportdatei ließ sich nicht ablegen: %w", err)
		}
		placed = true
		return nil
	}, ExaminerExempt()); err != nil {
		_ = os.Remove(tmp)
		if placed {
			_ = os.Remove(path)
		}
		return nil, err
	}
	return &DeclarationExport{Path: path, SHA256: sha, FiguresHash: d.FiguresHash,
		Status: declaration.StatusExported, Blocking: len(d.Blocking()), RecordID: rec.ID}, nil
}

// SubmitDeclaration hält die Übermittlung in Mein ELSTER mit Transferticket
// und Datum fest. Nur in der App.
func (t *Tenant) SubmitDeclaration(ctx context.Context, req service.SubmissionRequest) (*domain.DeclarationRecord, error) {
	var rec *domain.DeclarationRecord
	err := t.Write(ctx, func(ctx context.Context, s *Services) (err error) {
		rec, err = s.Declarations.MarkSubmitted(ctx, req)
		return err
	}, AppOnly())
	return rec, err
}

// writeTempFile schreibt eine Zwischendatei im Exportordner, damit kein
// halber Export unter dem endgültigen Namen liegt.
func writeTempFile(dir string, data []byte) (string, error) {
	tmp, err := os.CreateTemp(dir, ".erklaerung-*.json")
	if err != nil {
		return "", err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return name, nil
}
