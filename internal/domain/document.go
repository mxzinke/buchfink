package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Die Dokumentenablage des Unternehmens.
//
// Sie steht neben dem Beleg und neben dem Anlagendokument, weil sie eine dritte
// Sache ist. Ein Beleg gehört zu einer Buchung und einem Geschäftsjahr, ein
// Anlagendokument zu einem Wirtschaftsgut. Der Gesellschaftsvertrag gehört zu
// keinem von beidem: er gehört zum Unternehmen und gilt, solange es das
// Unternehmen gibt.
//
// Der Ablageweg ist derselbe wie überall: die Datei liegt unter ihrem eigenen
// SHA256, unverschlüsselt, und nur Pfad und Dateiname sind in der Datenbank
// verschlüsselt. Herausgegeben wird sie erst, nachdem die Prüfsumme stimmt.

// DocumentKind says what a document *is*.
//
// Wie beim Anlagendokument kein Etikett, sondern das, wonach gesucht wird. „Wo
// ist der Handelsregisterauszug" ist die Frage, und sie lässt sich nur
// beantworten, wenn die Art des Papiers festgehalten ist und nicht bloß sein
// Dateiname.
type DocumentKind string

const (
	// DocGesellschaftsvertrag ist die notarielle Urkunde, mit der die
	// Gesellschaft entsteht (§ 2 GmbHG). Sie gilt fort, solange es die
	// Gesellschaft gibt.
	DocGesellschaftsvertrag DocumentKind = "gesellschaftsvertrag"
	// DocGesellschafterliste ist die Liste nach § 40 GmbHG.
	DocGesellschafterliste DocumentKind = "gesellschafterliste"
	// DocHandelsregister ist der Registerauszug oder die Eintragungsnachricht
	// des Gerichts — der Nachweis, dass die Gesellschaft eingetragen ist.
	DocHandelsregister DocumentKind = "handelsregister"
	// DocEroeffnungsbilanz ist die Bilanz auf den Tag der Beurkundung
	// (§ 242 Abs. 1 HGB).
	DocEroeffnungsbilanz DocumentKind = "eroeffnungsbilanz"
	// DocSteuerlicheErfassung ist der Nachweis über den Fragebogen zur
	// steuerlichen Erfassung — das Übermittlungsprotokoll aus Mein ELSTER oder
	// die Mitteilung der Steuernummer.
	DocSteuerlicheErfassung DocumentKind = "steuerliche_erfassung"
	// DocGewerbeanmeldung ist der Gewerbeschein der Gemeinde.
	DocGewerbeanmeldung DocumentKind = "gewerbeanmeldung"
	// DocTransparenzregister ist die Bestätigung der Meldung nach § 20 GwG.
	DocTransparenzregister DocumentKind = "transparenzregister"
	// DocVertrag ist ein Vertrag des laufenden Betriebs: Miete, Versicherung,
	// Darlehen, Dienstleistung.
	DocVertrag DocumentKind = "vertrag"
	// DocBehoerde ist ein Schreiben einer Behörde: Bescheid, Anforderung,
	// Mitteilung.
	DocBehoerde  DocumentKind = "behoerde"
	DocSonstiges DocumentKind = "sonstiges"
)

// Label renders the kind for the UI.
func (k DocumentKind) Label() string {
	switch k {
	case DocGesellschaftsvertrag:
		return "Gesellschaftsvertrag"
	case DocGesellschafterliste:
		return "Gesellschafterliste"
	case DocHandelsregister:
		return "Handelsregisterauszug"
	case DocEroeffnungsbilanz:
		return "Eröffnungsbilanz"
	case DocSteuerlicheErfassung:
		return "Steuerliche Erfassung"
	case DocGewerbeanmeldung:
		return "Gewerbeanmeldung"
	case DocTransparenzregister:
		return "Transparenzregister"
	case DocVertrag:
		return "Vertrag"
	case DocBehoerde:
		return "Behördenschreiben"
	case DocSonstiges:
		return "Sonstiges"
	default:
		return string(k)
	}
}

// Valid reports whether the kind is one of the known ones.
func (k DocumentKind) Valid() bool {
	return k.Label() != string(k)
}

// AllDocumentKinds returns the catalog in the order the input mask shows it.
func AllDocumentKinds() []DocumentKind {
	return []DocumentKind{
		DocGesellschaftsvertrag, DocGesellschafterliste, DocHandelsregister,
		DocEroeffnungsbilanz, DocSteuerlicheErfassung, DocGewerbeanmeldung,
		DocTransparenzregister, DocVertrag, DocBehoerde, DocSonstiges,
	}
}

// Document is one file kept for the company as a whole.
type Document struct {
	ID   uint         `gorm:"primaryKey" json:"id"`
	Kind DocumentKind `gorm:"size:30;not null;index" json:"kind"`

	// Title is what the user reads in the list. Leer heißt: der Dateiname.
	Title string `gorm:"size:200;serializer:encrypted" json:"title,omitempty"`

	FileName string `gorm:"size:255;not null;serializer:encrypted" json:"fileName"`
	MimeType string `gorm:"size:127;not null" json:"mimeType"`
	Size     int64  `gorm:"not null" json:"size"`
	// SHA256 is the digest of the content, lowercase hex, and doubles as the file
	// name on disk.
	SHA256 string `gorm:"size:64;not null;index" json:"sha256"`
	// StoredPath is relative to the data directory.
	StoredPath string `gorm:"size:255;not null;serializer:encrypted" json:"storedPath"`

	// DocumentDate is the day the document bears, ValidUntil the day it runs out.
	DocumentDate string `gorm:"size:10;index" json:"documentDate,omitempty"`
	ValidUntil   string `gorm:"size:10;index" json:"validUntil,omitempty"`

	// DutyKey verbindet das Dokument mit einer Gründungspflicht, wenn es ihr
	// Nachweis ist — der Registerauszug mit der Anmeldung, der Gewerbeschein mit
	// der Gewerbeanmeldung.
	//
	// Leer bei jedem anderen Dokument. Es ist eine Verknüpfung und keine
	// Zugehörigkeit: der Registerauszug bleibt in der Ablage, auch wenn die
	// Gründung längst vorbei ist, und er ist dort zu finden, ohne den
	// Gründungsweg zu kennen.
	DutyKey string `gorm:"size:40;index" json:"dutyKey,omitempty"`

	// GeneratedBy hält fest, dass Buchfink das Dokument selbst erzeugt hat —
	// „eroeffnungsbilanz" oder „fragebogen".
	//
	// Der Unterschied zählt: ein selbst erzeugtes Dokument lässt sich jederzeit
	// neu herstellen, ein hochgeladenes nicht. Die Ansicht sagt das, damit
	// niemand die eine Datei für so unersetzlich hält wie die andere.
	GeneratedBy string `gorm:"size:40" json:"generatedBy,omitempty"`

	// RetentionClass und RetentionUntil halten die Aufbewahrungsfrist fest, die
	// beim Ablegen galt.
	//
	// Unterlagen des Unternehmens sind Organisationsunterlagen im Sinne des
	// § 147 Abs. 1 Nr. 1 AO: zehn Jahre, nicht die verkürzte Belegfrist von
	// acht. Gespeichert und nicht bei jedem Lesen gerechnet — welche Frist
	// einmal galt, ist eine Tatsache über das Dokument.
	RetentionClass RetentionClass `gorm:"size:20;index" json:"retentionClass,omitempty"`
	RetentionUntil string         `gorm:"size:10;index" json:"retentionUntil,omitempty"`

	Note      string    `gorm:"size:500;serializer:encrypted" json:"note,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// DisplayTitle is the name to show: the given title, or the file name.
func (d *Document) DisplayTitle() string {
	if strings.TrimSpace(d.Title) != "" {
		return d.Title
	}
	return d.FileName
}

// Validate enforces what has to hold before a document is stored.
func (d *Document) Validate() error {
	if !d.Kind.Valid() {
		return fmt.Errorf("unbekannte Dokumentart %q", d.Kind)
	}
	if d.FileName == "" || d.SHA256 == "" || d.StoredPath == "" {
		return fmt.Errorf("die Datei des Dokuments fehlt")
	}
	if d.DocumentDate != "" && len(d.DocumentDate) != 10 {
		return fmt.Errorf("das Datum des Dokuments ist unvollständig (erwartet JJJJ-MM-TT)")
	}
	if d.ValidUntil != "" && len(d.ValidUntil) != 10 {
		return fmt.Errorf("das Ablaufdatum ist unvollständig (erwartet JJJJ-MM-TT)")
	}
	if d.DocumentDate != "" && d.ValidUntil != "" && d.ValidUntil < d.DocumentDate {
		return fmt.Errorf("das Dokument liefe am %s ab, bevor es am %s ausgestellt wurde",
			d.ValidUntil, d.DocumentDate)
	}
	return nil
}

// documentJSON ist das Dokument ohne seine eigene Marshal-Methode; siehe
// receiptJSON zur Begründung des Umwegs.
type documentJSON Document

// MarshalJSON liefert das Dokument mit dem frühesten Löschdatum daneben.
func (d Document) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		documentJSON
		EarliestDeletion string `json:"earliestDeletion,omitempty"`
	}{documentJSON(d), EarliestDeletionAfter(d.RetentionUntil)})
}

// DocumentRepository persists the Dokumentenablage des Unternehmens.
type DocumentRepository interface {
	FindAll(ctx context.Context) ([]Document, error)
	FindByID(ctx context.Context, id uint) (*Document, error)
	// FindByDutyKey liefert die Nachweise zu einer Gründungspflicht.
	FindByDutyKey(ctx context.Context, key string) ([]Document, error)
	Create(ctx context.Context, doc *Document) error
	Delete(ctx context.Context, id uint) error
	// CountBySHA sagt, wie viele Dokumente sich eine Datei teilen. Zwei
	// identische Inhalte liegen nur einmal auf der Platte, und die Datei darf
	// erst weg, wenn der letzte Eintrag darauf entfernt ist.
	CountBySHA(ctx context.Context, sha256 string) (int64, error)
}
