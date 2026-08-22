package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ReceiptFileRole says what a file *is* within a Beleg. It is the load-bearing
// attribute of the whole model, not a label.
//
// A ZUGFeRD invoice is a PDF with embedded XML, an XRechnung is XML with no PDF
// at all, a scanned paper receipt is an image, and an entertainment receipt is a
// restaurant bill plus a self-issued note listing the participants. One file per
// Beleg cannot express any of that: you would have to pick a half and be wrong in
// both directions — keep the PDF and the input tax deduction loses its source,
// keep the XML and the user loses what they look at.
type ReceiptFileRole string

const (
	// ReceiptRoleOriginal is the file in the form it was received. Exactly one
	// per Beleg. GoBD Rz. 131 requires incoming documents to be kept in their
	// received format, which is why this role exists separately from the others.
	ReceiptRoleOriginal ReceiptFileRole = "original"
	// ReceiptRoleStructured is the structured invoice record. On a hybrid format
	// it is extracted from the original and therefore derived; on an XRechnung it
	// is the original, stored under both roles.
	ReceiptRoleStructured ReceiptFileRole = "structured"
	// ReceiptRoleRendering is a human-readable depiction produced by Buchfink
	// when the original has none. An XRechnung has no image part; without this
	// the user cannot look at the document they are about to book.
	ReceiptRoleRendering ReceiptFileRole = "rendering"
	// ReceiptRoleAttachment is a self-issued note, a participant list, a delivery
	// note or a proof of payment.
	ReceiptRoleAttachment ReceiptFileRole = "attachment"
)

// ReceiptStatus is the lifecycle of a Beleg.
//
// Filing and booking are two steps. An XRechnung must be storable immediately —
// the GoBD require keeping it in the received form, and the rendering can only be
// produced afterwards — but it must not be bookable before that rendering exists:
// nobody should approve a booking for a document they cannot look at.
type ReceiptStatus string

const (
	// ReceiptStatusFiled means the Beleg is stored but not yet booked. Files may
	// still be added or removed.
	ReceiptStatusFiled ReceiptStatus = "filed"
	// ReceiptStatusSealed means the Beleg is booked. Its file list is frozen,
	// because changing it would change the Beleg-Hash and break the journal's
	// chain. Anything arriving later — a reminder, a proof of payment, a
	// corrected invoice — is a Beleg of its own pointing at the same transaction.
	ReceiptStatusSealed ReceiptStatus = "sealed"
	// ReceiptStatusDiscarded means the Beleg was filed but will not be booked.
	// It is not deleted: it already carries a Belegnummer, and a received
	// document must not vanish without a trace.
	ReceiptStatusDiscarded ReceiptStatus = "discarded"
)

// displayableMimeTypes are the formats the user can actually look at. A Beleg
// whose original is not one of them needs a rendering before it may be booked.
var displayableMimeTypes = map[string]bool{
	"application/pdf": true,
	"image/png":       true,
	"image/jpeg":      true,
	"image/tiff":      true,
	"image/webp":      true,
	"image/gif":       true,
}

// ReceiptFile is one file of a Beleg.
//
// The content is addressed by its own SHA256; StoredPath only says where that
// content currently lies. Moving the data directory must not invalidate anything,
// which is why the hash and not the path is what the Beleg-Hash covers.
type ReceiptFile struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	ReceiptID uint            `gorm:"index;not null" json:"receiptId"`
	Position  int             `gorm:"not null" json:"position"`
	Role      ReceiptFileRole `gorm:"size:20;not null;index" json:"role"`

	// FileName is the name the file was received under. It is part of the
	// Beleg-Hash: a renamed attachment is a different Beleg.
	FileName string `gorm:"size:255;not null;serializer:encrypted" json:"fileName"`
	MimeType string `gorm:"size:127;not null" json:"mimeType"`
	Size     int64  `gorm:"not null" json:"size"`

	// SHA256 is the digest of the file content, lowercase hex. It doubles as the
	// file name on disk.
	SHA256 string `gorm:"size:64;not null;index" json:"sha256"`

	// Derived marks a file Buchfink produced from another one — the XML pulled
	// out of a hybrid PDF, or a rendering. GoBD Rz. 125 forbids losing the
	// structured part to a format conversion, so it has to be visible which file
	// is the received one and which was made from it.
	Derived bool `gorm:"not null;default:false" json:"derived"`

	// StoredPath is relative to the data directory.
	StoredPath string    `gorm:"size:255;not null;serializer:encrypted" json:"storedPath"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Receipt is a Beleg: the entity between the files and the booking.
//
// It exists because a Beleg is several files, not one, and because filing and
// booking are two separate acts with two separate checks.
type Receipt struct {
	ID         uint `gorm:"primaryKey" json:"id"`
	FiscalYear int  `gorm:"index;not null" json:"fiscalYear"`
	// ReceiptNumber comes from its own counter per fiscal year and is the
	// Belegfeld an auditor uses to find the document.
	ReceiptNumber string        `gorm:"size:50;not null;uniqueIndex" json:"receiptNumber"`
	Direction     Direction     `gorm:"size:20;not null;index" json:"direction"`
	Status        ReceiptStatus `gorm:"size:20;not null;index" json:"status"`

	Files []ReceiptFile `gorm:"foreignKey:ReceiptID;constraint:OnDelete:CASCADE" json:"files"`

	// ReceiptHash covers the ordered file list. It is the single value that
	// travels into the booking, so the journal's chain pins every file at once
	// without carrying n digests.
	ReceiptHash string `gorm:"size:64;not null" json:"receiptHash"`

	// ReceivedAt and ReceivedVia record how an incoming document entered the
	// business. Both are empty on documents Buchfink issued itself.
	ReceivedAt  string `gorm:"size:10" json:"receivedAt,omitempty"`
	ReceivedVia string `gorm:"size:30" json:"receivedVia,omitempty"`

	// The E-Rechnung fields. They stay empty on a Beleg without a structured
	// part — a scan or a plain PDF is not an E-Rechnung and must not look like a
	// failed one.
	//
	// The validation result is kept with the time, the rule set and its version.
	// A verdict without the rules that produced it cannot be reproduced later,
	// and reproducibility is the whole point of recording it.
	DetectedFormat  string `gorm:"size:20" json:"detectedFormat,omitempty"`
	DetectedProfile string `gorm:"size:120" json:"detectedProfile,omitempty"`

	ValidatedAt       string `gorm:"size:25" json:"validatedAt,omitempty"`
	ValidationRuleset string `gorm:"size:40" json:"validationRuleset,omitempty"`
	ValidationVersion string `gorm:"size:20" json:"validationVersion,omitempty"`
	// ValidationCoverage says how far the check went. There is no value meaning
	// "fully validated" — see internal/invoice/en16931.go.
	ValidationCoverage string `gorm:"size:20" json:"validationCoverage,omitempty"`
	ValidationErrors   int    `gorm:"default:0" json:"validationErrors"`
	// ValidationFindings holds the findings as JSON. They are display material,
	// not booking data, which is why they are not part of the Beleg-Hash.
	ValidationFindings string `gorm:"type:text;serializer:encrypted" json:"validationFindings,omitempty"`

	// JournalEntryID is set when the Beleg is sealed.
	JournalEntryID *uint  `gorm:"index" json:"journalEntryId,omitempty"`
	DiscardReason  string `gorm:"size:255;serializer:encrypted" json:"discardReason,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ReceivedVia values. The transport itself is not part of v1 — no mailbox, no
// server; UStAE 14.1 Abs. 5 Satz 3 does not require one. What is recorded is how
// the user says the document arrived.
const (
	ReceivedViaUpload     = "upload"      // Datei vom Rechner abgelegt
	ReceivedViaEmail      = "email"       // per E-Mail empfangen
	ReceivedViaScan       = "scan"        // Papier eingescannt
	ReceivedViaSelfIssued = "self_issued" // von Buchfink selbst erzeugt
)

// FileByRole returns the single file carrying a role.
func (r *Receipt) FileByRole(role ReceiptFileRole) (*ReceiptFile, bool) {
	for i := range r.Files {
		if r.Files[i].Role == role {
			return &r.Files[i], true
		}
	}
	return nil, false
}

// DisplayFile returns the file the user is shown.
//
// On a hybrid Beleg the image part is display only, never the booking source: if
// it differs from the XML that is potentially a second invoice with § 14c
// consequences. Booking always reads the structured part.
func (r *Receipt) DisplayFile() (*ReceiptFile, bool) {
	if original, ok := r.FileByRole(ReceiptRoleOriginal); ok && displayableMimeTypes[original.MimeType] {
		return original, true
	}
	return r.FileByRole(ReceiptRoleRendering)
}

// IsDisplayable reports whether the Beleg can be shown to the user at all.
func (r *Receipt) IsDisplayable() bool {
	_, ok := r.DisplayFile()
	return ok
}

// IsOpen reports whether files may still be added or removed.
func (r *Receipt) IsOpen() bool { return r.Status == ReceiptStatusFiled }

// ValidateStructure is the check that runs when a Beleg is filed.
//
// It deliberately does not require the Beleg to be displayable — that would make
// an XRechnung unfileable and would breach the very rule it is meant to uphold.
func (r *Receipt) ValidateStructure() error {
	if r.Direction != DirectionIncoming && r.Direction != DirectionOutgoing {
		return fmt.Errorf("unbekannte Belegrichtung %q", r.Direction)
	}
	if len(r.Files) == 0 {
		return fmt.Errorf("ein Beleg braucht mindestens die empfangene Originaldatei")
	}

	counts := map[ReceiptFileRole]int{}
	positions := map[int]bool{}

	for i := range r.Files {
		f := &r.Files[i]
		n := i + 1

		switch f.Role {
		case ReceiptRoleOriginal, ReceiptRoleStructured, ReceiptRoleRendering, ReceiptRoleAttachment:
		default:
			return fmt.Errorf("Datei %d: unbekannte Rolle %q", n, f.Role)
		}
		counts[f.Role]++

		if f.Position <= 0 {
			return fmt.Errorf("Datei %d (%s): Position fehlt", n, f.FileName)
		}
		if positions[f.Position] {
			return fmt.Errorf("Position %d ist doppelt vergeben", f.Position)
		}
		positions[f.Position] = true

		if strings.TrimSpace(f.FileName) == "" {
			return fmt.Errorf("Datei %d: der Originaldateiname fehlt", n)
		}
		if !isSHA256(f.SHA256) {
			return fmt.Errorf("Datei %d (%s): die Prüfsumme fehlt oder ist ungültig", n, f.FileName)
		}
		if strings.TrimSpace(f.StoredPath) == "" {
			return fmt.Errorf("Datei %d (%s): der Ablagepfad fehlt", n, f.FileName)
		}
		if f.MimeType == "" {
			return fmt.Errorf("Datei %d (%s): der Dateityp fehlt", n, f.FileName)
		}
		if f.Size <= 0 {
			return fmt.Errorf("Datei %d (%s): die Datei ist leer", n, f.FileName)
		}
	}

	if counts[ReceiptRoleOriginal] != 1 {
		return fmt.Errorf(
			"ein Beleg braucht genau eine Datei in der empfangenen Form, hat aber %d",
			counts[ReceiptRoleOriginal])
	}
	if counts[ReceiptRoleStructured] > 1 {
		return fmt.Errorf("ein Beleg kann höchstens einen strukturierten Rechnungsdatensatz tragen, hat aber %d", counts[ReceiptRoleStructured])
	}
	if counts[ReceiptRoleRendering] > 1 {
		return fmt.Errorf("ein Beleg kann höchstens eine erzeugte Darstellung tragen, hat aber %d", counts[ReceiptRoleRendering])
	}

	if original, ok := r.FileByRole(ReceiptRoleOriginal); ok && original.Derived {
		return fmt.Errorf("die Originaldatei %s kann nicht abgeleitet sein — sie ist die empfangene Form", original.FileName)
	}
	if rendering, ok := r.FileByRole(ReceiptRoleRendering); ok && !rendering.Derived {
		return fmt.Errorf("die Darstellung %s muss als abgeleitet gekennzeichnet sein", rendering.FileName)
	}

	return nil
}

// ValidateBookable is the check that runs when a Beleg is booked. It adds the one
// requirement filing must not impose: the Beleg has to be viewable.
func (r *Receipt) ValidateBookable() error {
	if err := r.ValidateStructure(); err != nil {
		return err
	}
	if r.Status == ReceiptStatusDiscarded {
		return fmt.Errorf("Beleg %s wurde verworfen und kann nicht gebucht werden", r.ReceiptNumber)
	}
	if !r.IsDisplayable() {
		return fmt.Errorf(
			"Beleg %s hat keine ansehbare Darstellung. Ein rein strukturierter Beleg (z. B. eine XRechnung) muss vor dem Buchen eine erzeugte Darstellung bekommen",
			r.ReceiptNumber)
	}
	return nil
}

func isSHA256(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f':
		default:
			return false
		}
	}
	return true
}

// ReceiptHashFunc computes the Beleg-Hash over the ordered file list.
type ReceiptHashFunc func(r *Receipt) string

// ReceiptRepository defines persistence for Belege.
type ReceiptRepository interface {
	FindByID(ctx context.Context, id uint) (*Receipt, error)
	FindAll(ctx context.Context, fiscalYear int) ([]Receipt, error)
	FindByStatus(ctx context.Context, fiscalYear int, status ReceiptStatus) ([]Receipt, error)
	FindByJournalEntry(ctx context.Context, entryID uint) (*Receipt, error)
	// Create allocates the Belegnummer, computes the Beleg-Hash and inserts the
	// Beleg with its files in one transaction.
	Create(ctx context.Context, receipt *Receipt, hash ReceiptHashFunc) error
	// ReplaceFiles swaps the file list of an unsealed Beleg and recomputes the
	// Beleg-Hash.
	ReplaceFiles(ctx context.Context, receiptID uint, files []ReceiptFile, hash ReceiptHashFunc) (*Receipt, error)
	// Seal marks a Beleg as booked. It is idempotent for the same entry, so a
	// crash between the journal write and the seal can be repaired by repeating it.
	Seal(ctx context.Context, receiptID uint, entryID uint) error
	Discard(ctx context.Context, receiptID uint, reason string) error
	// SaveValidation records the outcome of reading and checking the structured
	// part. It touches no file, so the Beleg-Hash is unaffected.
	SaveValidation(ctx context.Context, receiptID uint, v ReceiptValidation) error
}

// ReceiptValidation is what a check of the structured part leaves behind.
type ReceiptValidation struct {
	Format   string
	Profile  string
	At       string
	Ruleset  string
	Version  string
	Coverage string
	Errors   int
	Findings string
}
