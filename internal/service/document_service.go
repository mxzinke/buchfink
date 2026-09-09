package service

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/receiptstore"
)

// DocumentCategory ist der Zweig der Ablage, in dem die Unterlagen des
// Unternehmens liegen.
const DocumentCategory = "unternehmen"

// DocumentService führt die Dokumentenablage des Unternehmens.
//
// Sie ist der Ort für Papiere, die weder Beleg noch Anlagendokument sind: der
// Gesellschaftsvertrag, der Registerauszug, der Gewerbeschein, später der
// Mietvertrag und der Versicherungsschein. Sie gehören zum Unternehmen und
// gelten, solange es das Unternehmen gibt — ein Geschäftsjahr haben sie nicht.
//
// Der Ablageweg ist derselbe wie beim Beleg und beim Anlagendokument: die Datei
// liegt unter ihrem eigenen SHA256, herausgegeben wird sie erst, nachdem die
// Prüfsumme stimmt.
type DocumentService struct {
	repo      domain.DocumentRepository
	store     *receiptstore.Store
	auditRepo domain.AuditRepository
}

// NewDocumentService wires die Dokumentenablage.
func NewDocumentService(
	repo domain.DocumentRepository,
	store *receiptstore.Store,
	auditRepo domain.AuditRepository,
) *DocumentService {
	return &DocumentService{repo: repo, store: store, auditRepo: auditRepo}
}

// DocumentRequest ist eine abzulegende Unterlage.
type DocumentRequest struct {
	Kind         domain.DocumentKind `json:"kind"`
	Title        string              `json:"title"`
	DocumentDate string              `json:"documentDate"`
	ValidUntil   string              `json:"validUntil"`
	Note         string              `json:"note"`
	// DutyKey verbindet die Unterlage mit einer Gründungspflicht, wenn sie ihr
	// Nachweis ist. Leer bei jeder anderen.
	DutyKey string `json:"dutyKey"`
	// GeneratedBy hält fest, dass Buchfink die Unterlage selbst erzeugt hat.
	// Leer bei allem, was von außen kommt — und von außen kommt alles, was die
	// Oberfläche ablegt.
	GeneratedBy string `json:"-"`
	// Path ist der Weg zu einer Datei auf der Platte — der Weg des
	// Dateidialogs. Content ist der Inhalt selbst, für erzeugte Dokumente.
	Path     string `json:"path"`
	FileName string `json:"fileName"`
	Content  []byte `json:"-"`
}

// List liefert die Ablage.
func (s *DocumentService) List(ctx context.Context) ([]domain.Document, error) {
	if s.repo == nil {
		return []domain.Document{}, nil
	}
	return s.repo.FindAll(ctx)
}

// ForDuty liefert die Nachweise zu einer Gründungspflicht.
func (s *DocumentService) ForDuty(ctx context.Context, key string) ([]domain.Document, error) {
	if s.repo == nil {
		return []domain.Document{}, nil
	}
	return s.repo.FindByDutyKey(ctx, key)
}

// Attach legt eine Unterlage ab.
func (s *DocumentService) Attach(ctx context.Context, req DocumentRequest) (*domain.Document, error) {
	if s.repo == nil || s.store == nil {
		return nil, fmt.Errorf("die Dokumentenablage ist nicht verfügbar")
	}
	if !req.Kind.Valid() {
		return nil, fmt.Errorf("unbekannte Dokumentart %q", req.Kind)
	}

	stored, name, err := s.put(req)
	if err != nil {
		return nil, err
	}

	doc := &domain.Document{
		Kind:         req.Kind,
		Title:        strings.TrimSpace(req.Title),
		FileName:     name,
		MimeType:     stored.MimeType,
		Size:         stored.Size,
		SHA256:       stored.SHA256,
		StoredPath:   stored.RelPath,
		DocumentDate: strings.TrimSpace(req.DocumentDate),
		ValidUntil:   strings.TrimSpace(req.ValidUntil),
		DutyKey:      strings.TrimSpace(req.DutyKey),
		GeneratedBy:  strings.TrimSpace(req.GeneratedBy),
		Note:         strings.TrimSpace(req.Note),
	}
	applyCompanyRetention(doc)
	if err := doc.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, doc); err != nil {
		return nil, fmt.Errorf("die Unterlage konnte nicht abgelegt werden: %w", err)
	}
	s.audit(ctx, domain.AuditActionCreate, doc.ID, fmt.Sprintf(
		"Unterlage abgelegt: %s (%s), Prüfsumme %s", doc.DisplayTitle(), doc.Kind.Label(), doc.SHA256))
	return doc, nil
}

// put schreibt den Inhalt in die Ablage — aus einer Datei oder aus dem Speicher.
func (s *DocumentService) put(req DocumentRequest) (*receiptstore.StoredFile, string, error) {
	if len(req.Content) > 0 {
		name := strings.TrimSpace(req.FileName)
		if name == "" {
			return nil, "", fmt.Errorf("der erzeugten Unterlage fehlt ihr Dateiname")
		}
		stored, err := s.store.PutDocument(DocumentCategory, name, bytes.NewReader(req.Content))
		if err != nil {
			return nil, "", fmt.Errorf("die Unterlage konnte nicht abgelegt werden: %w", err)
		}
		return stored, name, nil
	}
	if strings.TrimSpace(req.Path) == "" {
		return nil, "", fmt.Errorf("es wurde keine Datei ausgewählt")
	}
	stored, err := s.store.PutDocumentPath(DocumentCategory, req.Path)
	if err != nil {
		return nil, "", fmt.Errorf("die Datei konnte nicht abgelegt werden: %w", err)
	}
	name := strings.TrimSpace(req.FileName)
	if name == "" {
		name = baseName(req.Path)
	}
	return stored, name, nil
}

// applyCompanyRetention setzt die Aufbewahrungsfrist der Unterlage.
//
// Unterlagen des Unternehmens sind Organisationsunterlagen nach § 147 Abs. 1
// Nr. 1 AO: zehn Jahre ab dem Schluss des Jahres, in dem sie entstanden sind.
// Der Gesellschaftsvertrag wirkt darüber hinaus fort — die Frist ist die
// gesetzliche Untergrenze und keine Aufforderung, danach zu löschen.
func applyCompanyRetention(doc *domain.Document) {
	origin := time.Now().Year()
	if len(doc.DocumentDate) >= 4 {
		if year, err := strconv.Atoi(doc.DocumentDate[:4]); err == nil && year > 1900 {
			origin = year
		}
	}
	info := accounting.RetentionFor(domain.RetentionKindOrganisation, origin)
	doc.RetentionClass = info.Class
	doc.RetentionUntil = info.RetentionEnd
}

// Content gibt eine Unterlage heraus — erst, nachdem die Prüfsumme stimmt.
func (s *DocumentService) Content(ctx context.Context, id uint) (*domain.Document, []byte, error) {
	if s.repo == nil || s.store == nil {
		return nil, nil, fmt.Errorf("die Dokumentenablage ist nicht verfügbar")
	}
	doc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if doc == nil {
		return nil, nil, fmt.Errorf("die Unterlage %d wurde nicht gefunden", id)
	}
	// Geprüft und nicht bloß gelesen: eine Datei, die sich seit der Ablage
	// geändert hat, ist nicht mehr die abgelegte. Sie stumm herauszugeben hieße,
	// die Prüfsumme zur Zierde zu erklären.
	if err := s.store.Verify(doc.StoredPath, doc.SHA256); err != nil {
		return nil, nil, fmt.Errorf(
			"%s stimmt nicht mehr mit der Prüfsumme aus der Ablage überein: %w", doc.DisplayTitle(), err)
	}
	data, err := s.store.Read(doc.StoredPath)
	if err != nil {
		return nil, nil, err
	}
	return doc, data, nil
}

// Remove entfernt eine Unterlage aus der Ablage.
//
// Die Datei wird erst gelöscht, wenn kein anderes Dokument mehr auf sie zeigt:
// gleiche Inhalte liegen nur einmal auf der Platte.
func (s *DocumentService) Remove(ctx context.Context, id uint) error {
	if s.repo == nil {
		return fmt.Errorf("die Dokumentenablage ist nicht verfügbar")
	}
	doc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if doc == nil {
		return fmt.Errorf("die Unterlage %d wurde nicht gefunden", id)
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.audit(ctx, domain.AuditActionUpdate, id, fmt.Sprintf(
		"Unterlage entfernt: %s (%s), Prüfsumme %s", doc.DisplayTitle(), doc.Kind.Label(), doc.SHA256))

	if s.store != nil {
		// Scheitert das Zählen, bleibt die Datei liegen: der Eintrag ist weg, und
		// eine Datei ohne Eintrag kostet Platz, behauptet aber nichts.
		if n, err := s.repo.CountBySHA(ctx, doc.SHA256); err == nil && n == 0 {
			_ = s.store.Delete(doc.StoredPath)
		}
	}
	return nil
}

func (s *DocumentService) audit(ctx context.Context, action domain.AuditAction, id uint, details string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, action, "DOKUMENT", fmt.Sprintf("%d", id), details)
}

// baseName schneidet den Dateinamen aus einem Pfad — ohne path/filepath, weil
// der Pfad vom Dateidialog des Betriebssystems kommt und beide Trenner tragen
// kann.
func baseName(path string) string {
	if i := strings.LastIndexAny(path, `/\`); i >= 0 {
		return path[i+1:]
	}
	return path
}
