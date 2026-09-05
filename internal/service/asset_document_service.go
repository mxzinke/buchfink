package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/receiptstore"
)

// Dokumente am Anlagegut.
//
// Ein Anlagegut wird über Jahre geführt, und die Papiere, die es erklären,
// liegen sonst überall verstreut: der Kaufvertrag im Ordner, das Gutachten im
// Postfach, der Fahrzeugbrief im Safe. Die Buchung selbst hat ihren Beleg — was
// hier dazukommt, ist alles, was nicht gebucht wird und trotzdem zum
// Wirtschaftsgut gehört.
//
// Es ist bewusst kein zweiter Belegkreis: kein lückenloser Nummernkreis, kein
// Geschäftsjahr, keine Versiegelung. Der Ablageweg ist derselbe wie beim Beleg —
// inhaltsadressiert und dedupliziert —, damit es nur eine Stelle gibt, an der
// das Schreiben einer Datei schiefgehen kann.

// SetDocumentStore wires the file store. Ohne ihn nimmt die Kartei keine
// Dokumente auf; sie funktioniert im Übrigen weiter.
func (s *AssetService) SetDocumentStore(store *receiptstore.Store) { s.docStore = store }

// AllDocuments liefert alle Anlagendokumente.
//
// Der Belegprüflauf fragt danach: die Prüfsumme eines Vertrags ist genauso zu
// prüfen wie die einer Rechnung, und über die Anlagegüter einzeln zu gehen
// hieße, die Kartei so oft zu lesen, wie sie Einträge hat.
func (s *AssetService) AllDocuments(ctx context.Context) ([]domain.AssetDocument, error) {
	return s.assetRepo.FindAllDocuments(ctx)
}

// AttachDocumentRequest legt ein Dokument zu einem Anlagegut ab.
type AttachDocumentRequest struct {
	AssetID uint                     `json:"assetId"`
	Kind    domain.AssetDocumentKind `json:"kind"`
	// Path ist eine Datei im Dateisystem. Dateien reisen als Pfad und nicht als
	// Inhalt — dieselbe Entscheidung wie beim Beleg: ein mehrere Megabyte großer
	// Scan hat nichts an der Schnittstelle zur Oberfläche verloren.
	Path string `json:"path"`
	// Content und FileName treten an die Stelle von Path, wo Buchfink die Datei
	// selbst erzeugt hat.
	Content  []byte `json:"-"`
	FileName string `json:"fileName,omitempty"`

	Title        string `json:"title,omitempty"`
	DocumentDate string `json:"documentDate,omitempty"`
	ValidUntil   string `json:"validUntil,omitempty"`
	Note         string `json:"note,omitempty"`
}

// AttachDocument stores a file and links it to the Anlagegut.
func (s *AssetService) AttachDocument(ctx context.Context, req AttachDocumentRequest) (*domain.FixedAsset, error) {
	if s.docStore == nil {
		return nil, fmt.Errorf("die Dokumentenablage ist nicht eingerichtet")
	}
	asset, err := s.assetRepo.FindByID(ctx, req.AssetID)
	if err != nil {
		return nil, fmt.Errorf("Anlagegut %d wurde nicht gefunden: %w", req.AssetID, err)
	}
	kind := req.Kind
	if kind == "" {
		kind = domain.AssetDocOther
	}
	if !kind.Valid() {
		return nil, fmt.Errorf("unbekannte Dokumentart %q", kind)
	}

	// Die Ablagekategorie ist die Dokumentart und nicht das Anlagegut: derselbe
	// Rahmenvertrag kann zu mehreren Wirtschaftsgütern gehören, und dann liegt
	// er einmal auf der Platte und nicht dreimal.
	var (
		stored *receiptstore.StoredFile
		name   string
	)
	switch {
	case req.Path != "":
		name = filepath.Base(req.Path)
		stored, err = s.docStore.PutDocumentPath(string(kind), req.Path)
	case len(req.Content) > 0:
		name = req.FileName
		if name == "" {
			name = "dokument"
		}
		stored, err = s.docStore.PutDocument(string(kind), name, bytes.NewReader(req.Content))
	default:
		return nil, fmt.Errorf("es wurde keine Datei übergeben")
	}
	if err != nil {
		return nil, err
	}

	document := &domain.AssetDocument{
		AssetID:      asset.ID,
		Kind:         kind,
		Title:        strings.TrimSpace(req.Title),
		FileName:     name,
		MimeType:     stored.MimeType,
		Size:         stored.Size,
		SHA256:       stored.SHA256,
		StoredPath:   stored.RelPath,
		DocumentDate: req.DocumentDate,
		ValidUntil:   req.ValidUntil,
		Note:         req.Note,
	}
	if err := document.Validate(); err != nil {
		return nil, err
	}
	if err := s.assetRepo.AddDocument(ctx, document); err != nil {
		return nil, fmt.Errorf("das Dokument konnte nicht gespeichert werden: %w", err)
	}
	s.audit(ctx, domain.AuditActionCreate, asset.ID, fmt.Sprintf(
		"Dokument zu %s abgelegt: %s (%s)", asset.InventoryNumber, document.DisplayTitle(), kind.Label()))
	return s.reload(ctx, asset.ID)
}

// RemoveDocument drops a document from an Anlagegut.
//
// Die Datei wird erst gelöscht, wenn kein anderes Dokument mehr auf sie zeigt —
// identische Inhalte liegen nur einmal auf der Platte, und ein zweites
// Anlagegut darf sich denselben Rahmenvertrag teilen.
//
// Der Vorgang steht im Protokoll. Ein Vertrag ist nach § 147 Abs. 1 Nr. 2 AO
// aufzubewahren; Buchfink hält ihn hier, ersetzt damit aber nicht die
// Aufbewahrung des Originals — und wer ihn entfernt, soll das später
// nachvollziehen können.
func (s *AssetService) RemoveDocument(ctx context.Context, assetID, documentID uint) (*domain.FixedAsset, error) {
	document, err := s.assetRepo.FindDocument(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("Dokument %d wurde nicht gefunden: %w", documentID, err)
	}
	if document.AssetID != assetID {
		return nil, fmt.Errorf("das Dokument gehört zu einem anderen Anlagegut")
	}
	asset, err := s.assetRepo.FindByID(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("Anlagegut %d wurde nicht gefunden: %w", assetID, err)
	}

	if err := s.assetRepo.DeleteDocument(ctx, documentID); err != nil {
		return nil, fmt.Errorf("das Dokument konnte nicht entfernt werden: %w", err)
	}
	if s.docStore != nil {
		remaining, err := s.assetRepo.CountDocumentsBySHA(ctx, document.SHA256)
		if err == nil && remaining == 0 {
			// Schlägt das Löschen fehl, bleibt eine verwaiste Datei liegen. Das
			// ist der harmlosere Ausgang: der Eintrag ist weg, und eine Datei
			// ohne Eintrag kostet Platz, aber sie behauptet nichts.
			_ = s.docStore.Delete(document.StoredPath)
		}
	}
	s.audit(ctx, domain.AuditActionUpdate, assetID, fmt.Sprintf(
		"Dokument zu %s entfernt: %s (%s)", asset.InventoryNumber, document.DisplayTitle(),
		document.Kind.Label()))
	return s.reload(ctx, assetID)
}

// DocumentContent returns the bytes of a document, together with the answer to
// the only question that matters about a stored file: liegt dort noch, was
// abgelegt wurde?
func (s *AssetService) DocumentContent(ctx context.Context, documentID uint) (*FileContent, error) {
	if s.docStore == nil {
		return nil, fmt.Errorf("die Dokumentenablage ist nicht eingerichtet")
	}
	document, err := s.assetRepo.FindDocument(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("Dokument %d wurde nicht gefunden: %w", documentID, err)
	}
	data, err := s.docStore.Read(document.StoredPath)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	return &FileContent{
		Data:     data,
		FileName: document.FileName,
		MimeType: document.MimeType,
		Intact:   hex.EncodeToString(sum[:]) == document.SHA256,
	}, nil
}

// ExpiringDocuments lists documents whose Frist runs out on or before a day.
//
// Eine Police, die zum Jahresende ausläuft, und ein Darlehen, das fällig wird,
// sollen auffallen, bevor sie ablaufen. Ohne diese Abfrage wäre das
// Ablaufdatum eine Angabe, die niemand je wieder liest.
func (s *AssetService) ExpiringDocuments(ctx context.Context, until string) ([]ExpiringDocument, error) {
	assets, err := s.assetRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("das Anlagenverzeichnis konnte nicht gelesen werden: %w", err)
	}
	// Belegt statt nil: der Regelfall ist die Ablage ohne ablaufendes
	// Dokument, und die Ansicht läse dort sonst `null`.
	out := make([]ExpiringDocument, 0)
	for i := range assets {
		asset := &assets[i]
		if asset.IsDisposed() {
			continue
		}
		for _, document := range asset.Documents {
			if document.ValidUntil == "" || document.ValidUntil > until {
				continue
			}
			out = append(out, ExpiringDocument{
				AssetID:         asset.ID,
				InventoryNumber: asset.InventoryNumber,
				AssetName:       asset.Name,
				DocumentID:      document.ID,
				Kind:            document.Kind,
				KindLabel:       document.Kind.Label(),
				Title:           document.DisplayTitle(),
				ValidUntil:      document.ValidUntil,
			})
		}
	}
	return out, nil
}

// ExpiringDocument is one document whose Frist is up.
type ExpiringDocument struct {
	AssetID         uint                     `json:"assetId"`
	InventoryNumber string                   `json:"inventoryNumber"`
	AssetName       string                   `json:"assetName"`
	DocumentID      uint                     `json:"documentId"`
	Kind            domain.AssetDocumentKind `json:"kind"`
	KindLabel       string                   `json:"kindLabel"`
	Title           string                   `json:"title"`
	ValidUntil      string                   `json:"validUntil"`
}
