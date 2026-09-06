package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/receiptstore"
)

// NewFile is one file on its way into a Beleg.
//
// It arrives either as a path on disk — the native file dialog and drag & drop
// both hand over paths, which keeps multi-megabyte scans off the IPC boundary —
// or as content produced in process, such as an extracted XML or a rendering.
type NewFile struct {
	Role domain.ReceiptFileRole

	// Path is a file on the local file system. Its base name becomes the
	// original file name.
	Path string

	// Content and FileName are used instead of Path for files Buchfink produced.
	Content  []byte
	FileName string

	// Derived marks a file that was made from another one rather than received.
	Derived bool
}

// FileReceiptRequest is the input for filing a Beleg away.
//
// Filing is not booking. It runs the structural check only, so an XRechnung —
// which has no image part and cannot be booked until a rendering exists — can
// still be stored the moment it arrives, as GoBD Rz. 131 requires.
type FileReceiptRequest struct {
	Direction  domain.Direction `json:"direction"`
	FiscalYear int              `json:"fiscalYear,omitempty"`

	// ReceiptNumber is preset for outgoing Belege, which carry the
	// Rechnungsnummer the invoice already allocated. Empty for incoming ones,
	// where the Beleg draws from its own counter.
	ReceiptNumber string `json:"receiptNumber,omitempty"`

	ReceivedAt  string `json:"receivedAt,omitempty"`
	ReceivedVia string `json:"receivedVia,omitempty"`

	// Kind ist die Belegart. Leer heißt Rechnung — der Regelfall darf keine
	// Eingabe verlangen.
	Kind domain.ReceiptKind `json:"kind,omitempty"`

	// Die Kopfdaten. Beim Ablegen freiwillig, beim Buchen Pflicht: eine
	// E-Rechnung liefert sie aus ihrem strukturierten Teil, ein Papierscan
	// nicht, und ein Beleg, der erst nach einer Eingabe abgelegt werden dürfte,
	// bliebe liegen — genau das, was GoBD Rz. 131 verhindern will.
	DocumentDate string       `json:"documentDate,omitempty"`
	IssuerName   string       `json:"issuerName,omitempty"`
	GrossAmount  domain.Cents `json:"grossAmount,omitempty"`
	TaxAmount    domain.Cents `json:"taxAmount,omitempty"`
	Currency     string       `json:"currency,omitempty"`
	Subject      string       `json:"subject,omitempty"`

	Files []NewFile `json:"-"`
}

// FileContent is a stored file handed back for display or for parsing.
type FileContent struct {
	Data     []byte `json:"-"`
	FileName string `json:"fileName"`
	MimeType string `json:"mimeType"`
	// Intact reports whether the bytes on disk still hash to the digest recorded
	// when the file was filed. A false value is shown, not hidden: the user has
	// to see that a Beleg was tampered with, and hiding it would be the worse
	// failure.
	Intact bool `json:"intact"`
}

// ReceiptService files Belege away, hands them out and seals them.
type ReceiptService struct {
	receiptRepo domain.ReceiptRepository
	journalRepo domain.JournalRepository
	store       *receiptstore.Store
	auditRepo   domain.AuditRepository
	fiscalYear  int
	// documents ist die Anlagenkartei, soweit der Belegprüflauf sie braucht.
	// Optional: ohne sie prüft er die Belege und sagt es.
	documents DocumentSource
}

// NewReceiptService creates the Beleg service.
func NewReceiptService(
	receiptRepo domain.ReceiptRepository,
	journalRepo domain.JournalRepository,
	store *receiptstore.Store,
	auditRepo domain.AuditRepository,
	fiscalYear int,
) *ReceiptService {
	return &ReceiptService{
		receiptRepo: receiptRepo,
		journalRepo: journalRepo,
		store:       store,
		auditRepo:   auditRepo,
		fiscalYear:  fiscalYear,
	}
}

// SetFiscalYear updates the year new Belege are filed under.
func (s *ReceiptService) SetFiscalYear(year int) { s.fiscalYear = year }

// File stores the files on disk and inserts the Beleg.
//
// The order matters: the files go to disk first, under their own digests, and
// only then does the transaction allocate the Belegnummer and insert the record.
// Nothing on disk depends on the number, so a failed insert leaves orphaned
// content instead of a gap in the number range — the cheaper of the two failures.
func (s *ReceiptService) File(ctx context.Context, req FileReceiptRequest) (*domain.Receipt, error) {
	if len(req.Files) == 0 {
		return nil, fmt.Errorf("ein Beleg braucht mindestens eine Datei")
	}
	fiscalYear := req.FiscalYear
	if fiscalYear == 0 {
		fiscalYear = s.fiscalYear
	}
	if fiscalYear == 0 {
		return nil, fmt.Errorf("kein Geschäftsjahr gesetzt")
	}

	kind := req.Kind
	if kind == "" {
		kind = domain.ReceiptKindInvoice
	}

	receipt := &domain.Receipt{
		FiscalYear:    fiscalYear,
		ReceiptNumber: req.ReceiptNumber,
		Direction:     req.Direction,
		Kind:          kind,
		Status:        domain.ReceiptStatusFiled,
		ReceivedAt:    req.ReceivedAt,
		ReceivedVia:   req.ReceivedVia,
		DocumentDate:  req.DocumentDate,
		IssuerName:    req.IssuerName,
		GrossAmount:   req.GrossAmount,
		TaxAmount:     req.TaxAmount,
		Currency:      req.Currency,
		Subject:       req.Subject,
	}
	applyRetention(receipt)
	if receipt.ReceivedAt == "" && req.Direction == domain.DirectionIncoming {
		receipt.ReceivedAt = todayLocal()
	}
	if receipt.ReceivedVia == "" {
		if req.Direction == domain.DirectionOutgoing {
			receipt.ReceivedVia = domain.ReceivedViaSelfIssued
		} else {
			receipt.ReceivedVia = domain.ReceivedViaUpload
		}
	}

	files, err := s.storeAll(fiscalYear, req.Direction, req.Files)
	if err != nil {
		return nil, err
	}
	receipt.Files = files

	// The structural check runs before the insert so a malformed Beleg never
	// consumes a Belegnummer.
	if err := receipt.ValidateStructure(); err != nil {
		return nil, err
	}

	if err := s.receiptRepo.Create(ctx, receipt, accounting.ReceiptHash); err != nil {
		return nil, fmt.Errorf("Beleg konnte nicht abgelegt werden: %w", err)
	}

	s.log(ctx, domain.AuditActionCreate, receipt,
		fmt.Sprintf("Beleg %s mit %d Datei(en) abgelegt", receipt.ReceiptNumber, len(receipt.Files)))
	return receipt, nil
}

// AddFile appends a file to a Beleg that is not yet booked.
func (s *ReceiptService) AddFile(ctx context.Context, receiptID uint, file NewFile) (*domain.Receipt, error) {
	receipt, err := s.Get(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	if !receipt.IsOpen() {
		return nil, fmt.Errorf(
			"Beleg %s ist gebucht und damit versiegelt. Was später dazukommt — eine Mahnung, ein Zahlungsnachweis, eine korrigierte Rechnung — ist ein eigener Beleg auf denselben Geschäftsvorfall",
			receipt.ReceiptNumber)
	}

	stored, err := s.storeAll(receipt.FiscalYear, receipt.Direction, []NewFile{file})
	if err != nil {
		return nil, err
	}
	return s.replaceFiles(ctx, receipt, append(receipt.Files, stored...))
}

// RemoveFile drops a file from a Beleg that is not yet booked.
func (s *ReceiptService) RemoveFile(ctx context.Context, receiptID, fileID uint) (*domain.Receipt, error) {
	receipt, err := s.Get(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	if !receipt.IsOpen() {
		return nil, fmt.Errorf("Beleg %s ist gebucht und seine Dateien lassen sich nicht mehr ändern", receipt.ReceiptNumber)
	}

	kept := make([]domain.ReceiptFile, 0, len(receipt.Files))
	var removed *domain.ReceiptFile
	for i := range receipt.Files {
		if receipt.Files[i].ID == fileID {
			removed = &receipt.Files[i]
			continue
		}
		kept = append(kept, receipt.Files[i])
	}
	if removed == nil {
		return nil, fmt.Errorf("die Datei gehört nicht zu Beleg %s", receipt.ReceiptNumber)
	}
	// Die empfangene Originaldatei bleibt (BEL-03).
	//
	// GoBD Rz. 131 verlangt, eingehende Dokumente in der empfangenen Form
	// aufzubewahren. Ein Beleg, dessen Original entfernt werden kann, hält
	// nichts fest: was danach dasteht, ist eine von Buchfink erzeugte
	// Darstellung oder ein Anhang, und der Nachweis, wie das Dokument
	// angekommen ist, ist weg. Die Strukturprüfung hätte den Beleg zwar ohnehin
	// zurückgewiesen — sie verlangt genau eine Datei in der empfangenen Form —,
	// aber mit einer Meldung über Positionen und Rollen. Hier steht der Grund.
	if removed.Role == domain.ReceiptRoleOriginal {
		return nil, fmt.Errorf(
			"die Originaldatei %s von Beleg %s lässt sich nicht entfernen: eingehende Dokumente sind in der empfangenen Form aufzubewahren (GoBD Rz. 131). "+
				"Wenn der ganze Beleg nicht gebucht werden soll, verwirf ihn mit einer Begründung",
			removed.FileName, receipt.ReceiptNumber)
	}

	// The content itself stays on disk: another Beleg may share it, and the store
	// is content-addressed precisely so identical files exist once.
	updated, err := s.replaceFiles(ctx, receipt, kept)
	if err != nil {
		return nil, err
	}
	// Name und Prüfsumme der entfernten Datei stehen im Protokoll. Ohne sie
	// bliebe von einer entfernten Datei nichts als die Zahl der verbliebenen,
	// und die Frage, was da war, ließe sich nicht mehr beantworten.
	s.log(ctx, domain.AuditActionUpdate, updated, fmt.Sprintf(
		"Aus Beleg %s wurde die Datei %s (Rolle %s, SHA256 %s) entfernt",
		receipt.ReceiptNumber, removed.FileName, removed.Role, removed.SHA256))
	return updated, nil
}

// applyRetention setzt Aufbewahrungsklasse und Fristende eines Belegs.
//
// Beim Ablegen und nicht beim Anzeigen: welche Frist gilt, hängt am
// Entstehungsjahr und am geltenden Recht, und beides ist zum Zeitpunkt der
// Ablage bekannt. Später gerechnet käme für denselben Beleg irgendwann eine
// andere Zahl heraus.
func applyRetention(receipt *domain.Receipt) {
	origin := receipt.FiscalYear
	if receipt.DocumentDate != "" && len(receipt.DocumentDate) >= 4 {
		if year, err := strconv.Atoi(receipt.DocumentDate[:4]); err == nil && year > 1900 {
			origin = year
		}
	}
	info := accounting.RetentionFor(domain.RetentionKindOf(receipt.Kind), origin)
	receipt.RetentionClass = info.Class
	receipt.RetentionUntil = info.RetentionEnd
}

// SaveHeader schreibt die Kopfdaten eines abgelegten Belegs nach.
//
// Der Weg für den Papierscan: die Datei liegt sofort im Speicher, die Kopfdaten
// trägt jemand nach, und erst dann ist der Beleg buchbar. Die
// Aufbewahrungsfrist wird dabei neu bestimmt, weil das Belegdatum sie
// verschieben kann.
func (s *ReceiptService) SaveHeader(ctx context.Context, receiptID uint, header domain.ReceiptHeader) (*domain.Receipt, error) {
	receipt, err := s.Get(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	if header.Kind == "" {
		header.Kind = receipt.Kind
	}

	probe := *receipt
	probe.Kind = header.Kind
	probe.DocumentDate = header.DocumentDate
	applyRetention(&probe)
	header.RetentionClass = probe.RetentionClass
	header.RetentionUntil = probe.RetentionUntil

	updated, err := s.receiptRepo.SaveHeader(ctx, receiptID, header, accounting.ReceiptHash)
	if err != nil {
		return nil, err
	}
	// Vorher/Nachher und nicht nur Freitext: die Kopfdaten sind die Stammdaten
	// des Belegs, sie gehen in den Beleg-Hash und damit in den Hash jeder
	// Buchung, die auf ihn verweist. Wer das Belegdatum eines abgelegten Belegs
	// ändert, verschiebt seine Aufbewahrungsfrist und die Beurteilung seiner
	// zeitgerechten Erfassung — beides gehört mit beiden Ständen ins Protokoll
	// (GoBD Rz. 34).
	if s.auditRepo != nil {
		_ = s.auditRepo.LogChange(ctx, domain.AuditActionUpdate, "RECEIPT", fmt.Sprintf("%d", receiptID),
			fmt.Sprintf("Kopfdaten von Beleg %s erfasst: %s, %s, %s €",
				updated.ReceiptNumber, updated.DocumentDate, updated.IssuerName, updated.GrossAmount),
			receipt, updated)
	}
	return s.Get(ctx, receiptID)
}

func (s *ReceiptService) replaceFiles(ctx context.Context, receipt *domain.Receipt, files []domain.ReceiptFile) (*domain.Receipt, error) {
	probe := *receipt
	probe.Files = files
	for i := range probe.Files {
		probe.Files[i].Position = i + 1
	}
	if err := probe.ValidateStructure(); err != nil {
		return nil, err
	}

	updated, err := s.receiptRepo.ReplaceFiles(ctx, receipt.ID, files, accounting.ReceiptHash)
	if err != nil {
		return nil, err
	}
	s.log(ctx, domain.AuditActionUpdate, updated,
		fmt.Sprintf("Beleg %s trägt jetzt %d Datei(en)", updated.ReceiptNumber, len(updated.Files)))
	return s.Get(ctx, receipt.ID)
}

// Get returns a Beleg with its files in hash order, repairing a missing seal on
// the way.
//
// The seal is a second write, after the journal transaction has committed. If
// the process dies in between, a booked Beleg still looks open — and would then
// accept another file, which would change its hash and break the chain of an
// entry that is already written. Reading is the natural place to notice and fix
// that, because nothing can be booked without reading the Beleg first.
func (s *ReceiptService) Get(ctx context.Context, id uint) (*domain.Receipt, error) {
	receipt, err := s.receiptRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("Beleg konnte nicht geladen werden: %w", err)
	}
	if receipt.Status == domain.ReceiptStatusFiled && s.journalRepo != nil {
		entry, lookupErr := s.journalRepo.FindByReceipt(ctx, receipt.ID)
		if lookupErr == nil && entry != nil {
			if sealErr := s.receiptRepo.Seal(ctx, receipt.ID, entry.ID); sealErr == nil {
				s.log(ctx, domain.AuditActionUpdate, receipt, fmt.Sprintf(
					"Beleg %s nachträglich mit Buchung %s versiegelt", receipt.ReceiptNumber, entry.EntryNumber))
				return s.receiptRepo.FindByID(ctx, id)
			}
		}
	}
	return receipt, nil
}

// List returns the Belege of the current fiscal year, newest first. An empty
// status returns all of them.
func (s *ReceiptService) List(ctx context.Context, status domain.ReceiptStatus) ([]domain.Receipt, error) {
	if status == "" {
		return s.receiptRepo.FindAll(ctx, s.fiscalYear)
	}
	return s.receiptRepo.FindByStatus(ctx, s.fiscalYear, status)
}

// Content returns a stored file for display or for parsing, together with
// whether it still matches its recorded digest.
func (s *ReceiptService) Content(ctx context.Context, receiptID, fileID uint) (*FileContent, error) {
	receipt, err := s.Get(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	for i := range receipt.Files {
		f := &receipt.Files[i]
		if f.ID != fileID {
			continue
		}
		data, err := s.store.Read(f.StoredPath)
		if err != nil {
			return nil, err
		}
		// Die Prüfsumme aus den gelesenen Bytes, nicht aus einem zweiten Lesen:
		// eine Belegdatei kann zweistellig MB groß sein, und die Vorschau lädt
		// sie ohnehin schon einmal ganz.
		sum := sha256.Sum256(data)
		return &FileContent{
			Data:     data,
			FileName: f.FileName,
			MimeType: f.MimeType,
			Intact:   hex.EncodeToString(sum[:]) == f.SHA256,
		}, nil
	}
	return nil, fmt.Errorf("die Datei gehört nicht zu Beleg %s", receipt.ReceiptNumber)
}

// DisplayContent returns the file the user is shown for a Beleg.
func (s *ReceiptService) DisplayContent(ctx context.Context, receiptID uint) (*FileContent, error) {
	receipt, err := s.Get(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	file, ok := receipt.DisplayFile()
	if !ok {
		return nil, fmt.Errorf("Beleg %s hat keine ansehbare Darstellung", receipt.ReceiptNumber)
	}
	return s.Content(ctx, receiptID, file.ID)
}

// Seal marks a Beleg as booked. It runs after the journal write has committed and
// is idempotent, so a crash in between can be repaired by repeating it.
func (s *ReceiptService) Seal(ctx context.Context, receiptID, entryID uint) error {
	// Die Kopfdaten sind beim Buchen Pflicht, und das Versiegeln ist das
	// Buchen: hier bekommt der Beleg seine Buchung (BEL-02).
	//
	// Geprüft wird ValidateHeader und nicht das vollständige ValidateBookable:
	// die Ansehbarkeit, die dort zusätzlich verlangt wird, gilt dem Beleg, den
	// ein Mensch vor sich hat und bucht (siehe PostingService.BookReceipt).
	// Ein Eigenbeleg der Abschlussbuchung ist eine JSON-Datei — prüfbar, aber
	// nicht „ansehbar" —, und ihn hier abzuweisen hieße, die Abschlussbuchung
	// an einer Regel scheitern zu lassen, die für sie nicht gedacht ist.
	receipt, err := s.receiptRepo.FindByID(ctx, receiptID)
	if err != nil {
		return fmt.Errorf("Beleg %d wurde nicht gefunden: %w", receiptID, err)
	}
	if err := receipt.ValidateHeader(); err != nil {
		return err
	}
	if err := s.receiptRepo.Seal(ctx, receiptID, entryID); err != nil {
		return err
	}
	return nil
}

// SaveValidation records the outcome of checking the structured part of a Beleg.
//
// The result arrives ready to store. Which rules ran and how far they reached
// is the reader's business, not the Beleg's — the Beleg only has to keep what
// was found, so that a later run under a newer rule set is comparable.
func (s *ReceiptService) SaveValidation(ctx context.Context, receiptID uint, result domain.ReceiptValidation) error {
	return s.receiptRepo.SaveValidation(ctx, receiptID, result)
}

// SaveInputTaxOverride hält den Grund fest, mit dem ein blockierender Befund der
// Rechnungsprüfung übersteuert wurde.
func (s *ReceiptService) SaveInputTaxOverride(ctx context.Context, receiptID uint, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("ohne Grund wird nicht übersteuert")
	}
	if err := s.receiptRepo.SaveInputTaxOverride(
		ctx, receiptID, strings.TrimSpace(reason), time.Now().UTC().Format(time.RFC3339)); err != nil {
		return err
	}
	// Ins Protokoll: die Übersteuerung ist eine Entscheidung des Anwenders gegen
	// eine Prüfung, und solche Entscheidungen gehören nachvollziehbar
	// festgehalten (GoBD Rz. 36).
	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, domain.AuditActionUpdate, "RECEIPT", fmt.Sprintf("%d", receiptID),
			"Befund der Rechnungsprüfung übersteuert, Vorsteuer trotzdem gezogen: "+
				strings.TrimSpace(reason))
	}
	return nil
}

// Discard retires a filed Beleg. It keeps its number and stays findable.
func (s *ReceiptService) Discard(ctx context.Context, receiptID uint, reason string) error {
	if reason == "" {
		return fmt.Errorf("zum Verwerfen eines Belegs gehört eine Begründung")
	}
	receipt, err := s.Get(ctx, receiptID)
	if err != nil {
		return err
	}
	if err := s.receiptRepo.Discard(ctx, receiptID, reason); err != nil {
		return err
	}
	s.log(ctx, domain.AuditActionUpdate, receipt,
		fmt.Sprintf("Beleg %s verworfen: %s", receipt.ReceiptNumber, reason))
	return nil
}

func (s *ReceiptService) storeAll(fiscalYear int, direction domain.Direction, files []NewFile) ([]domain.ReceiptFile, error) {
	stored := make([]domain.ReceiptFile, 0, len(files))
	for i, f := range files {
		name := f.FileName
		var result *receiptstore.StoredFile
		var err error

		switch {
		case f.Path != "":
			if name == "" {
				name = filepath.Base(f.Path)
			}
			result, err = s.store.PutPath(fiscalYear, direction, f.Path)
		case len(f.Content) > 0:
			if name == "" {
				return nil, fmt.Errorf("Datei %d: für erzeugte Inhalte muss ein Dateiname angegeben werden", i+1)
			}
			result, err = s.store.Put(fiscalYear, direction, name, bytes.NewReader(f.Content))
		default:
			return nil, fmt.Errorf("Datei %d: weder ein Pfad noch ein Inhalt angegeben", i+1)
		}
		if err != nil {
			return nil, err
		}

		stored = append(stored, domain.ReceiptFile{
			Position:   len(stored) + 1,
			Role:       f.Role,
			FileName:   name,
			MimeType:   result.MimeType,
			Size:       result.Size,
			SHA256:     result.SHA256,
			Derived:    f.Derived,
			StoredPath: result.RelPath,
		})
	}
	return stored, nil
}

func (s *ReceiptService) log(ctx context.Context, action domain.AuditAction, receipt *domain.Receipt, details string) {
	if s.auditRepo == nil || receipt == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, action, "RECEIPT", fmt.Sprintf("%d", receipt.ID), details)
}

// FindByOriginalHash liefert den Beleg zu einer bereits abgelegten
// Originaldatei, oder nil.
func (s *ReceiptService) FindByOriginalHash(ctx context.Context, sha256 string) (*domain.Receipt, error) {
	return s.receiptRepo.FindByOriginalHash(ctx, sha256)
}
