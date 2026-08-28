package domain

import (
	"fmt"
	"strings"
	"time"
)

// AssetDocumentKind says what a document *is* zu einem Anlagegut.
//
// Es ist kein Etikett, sondern das, wonach später gesucht wird. „Wo ist der
// Kaufvertrag zu der Maschine" ist die Frage, die zu diesem Modell geführt hat —
// und sie lässt sich nur beantworten, wenn die Art des Papiers festgehalten ist
// und nicht bloß sein Dateiname.
type AssetDocumentKind string

const (
	// AssetDocContract ist der Vertrag, aus dem das Anlagegut stammt:
	// Kaufvertrag, Darlehensvertrag, Leasingvertrag, Zeichnungsschein.
	AssetDocContract AssetDocumentKind = "contract"
	// AssetDocInvoice ist die Rechnung als Kopie. Das Original ist ein Beleg und
	// steht im Belegkreis; hier liegt es, damit es beim Anlagegut auffindbar ist.
	AssetDocInvoice AssetDocumentKind = "invoice"
	// AssetDocValuation ist ein Gutachten oder eine Wertermittlung — die
	// Begründung einer außerplanmäßigen Abschreibung oder einer Nutzungsdauer,
	// die von der AfA-Tabelle abweicht.
	AssetDocValuation AssetDocumentKind = "valuation"
	// AssetDocRegistration ist ein Register- oder Zulassungspapier:
	// Fahrzeugbrief, Grundbuchauszug, Handelsregisterauszug, Depotbestätigung.
	AssetDocRegistration AssetDocumentKind = "registration"
	// AssetDocInsurance ist eine Versicherungspolice.
	AssetDocInsurance AssetDocumentKind = "insurance"
	// AssetDocMaintenance ist ein Wartungs-, Prüf- oder Reparaturbericht.
	AssetDocMaintenance AssetDocumentKind = "maintenance"
	// AssetDocStatement ist eine Abrechnung: Depotauszug, Ausschüttungsanzeige,
	// Jahressteuerbescheinigung, Tilgungsplan.
	AssetDocStatement AssetDocumentKind = "statement"
	// AssetDocPhoto ist ein Bild des Wirtschaftsguts — bei der Inventur das
	// schnellste Mittel, ein Gerät wiederzuerkennen.
	AssetDocPhoto AssetDocumentKind = "photo"
	AssetDocOther AssetDocumentKind = "other"
)

// Label renders the kind for the UI.
func (k AssetDocumentKind) Label() string {
	switch k {
	case AssetDocContract:
		return "Vertrag"
	case AssetDocInvoice:
		return "Rechnung (Kopie)"
	case AssetDocValuation:
		return "Gutachten"
	case AssetDocRegistration:
		return "Register- oder Zulassungspapier"
	case AssetDocInsurance:
		return "Versicherung"
	case AssetDocMaintenance:
		return "Wartung und Prüfung"
	case AssetDocStatement:
		return "Abrechnung"
	case AssetDocPhoto:
		return "Bild"
	case AssetDocOther:
		return "Sonstiges"
	default:
		return string(k)
	}
}

// Valid reports whether the kind is one of the known ones.
func (k AssetDocumentKind) Valid() bool {
	return k.Label() != string(k)
}

// AllAssetDocumentKinds returns the catalog in the order the input mask shows it.
func AllAssetDocumentKinds() []AssetDocumentKind {
	return []AssetDocumentKind{
		AssetDocContract, AssetDocInvoice, AssetDocStatement, AssetDocValuation,
		AssetDocRegistration, AssetDocInsurance, AssetDocMaintenance, AssetDocPhoto,
		AssetDocOther,
	}
}

// AssetDocument is one file kept alongside an Anlagegut.
//
// Es ist bewusst kein Beleg. Ein Beleg trägt eine Belegnummer aus einem
// lückenlosen Kreis, gehört zu einem Geschäftsjahr, wird gebucht und ist danach
// versiegelt, weil sein Hash in der Journalkette hängt. Ein Kaufvertrag ist
// nichts davon: er wird nicht gebucht, er gehört zum Wirtschaftsgut und nicht
// zum Jahr, und er erklärt die Anschaffung noch, wenn die Maschine zehn Jahre
// im Bestand ist. Ihn in das Belegmodell zu zwingen hieße, ihm eine
// Belegnummer zu geben, die nie in einer Buchung auftaucht.
//
// Der Ablageweg ist trotzdem derselbe wie beim Beleg: die Datei liegt unter
// ihrem eigenen SHA256, unverschlüsselt, und nur Pfad und Dateiname sind in der
// Datenbank verschlüsselt.
type AssetDocument struct {
	ID      uint              `gorm:"primaryKey" json:"id"`
	AssetID uint              `gorm:"index;not null" json:"assetId"`
	Kind    AssetDocumentKind `gorm:"size:20;not null;index" json:"kind"`

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
	//
	// Die Frist ist der Grund, warum das Datum hier steht und nicht nur im
	// Dokument: eine Police, die zum Jahresende ausläuft, und ein Darlehen, das
	// fällig wird, sollen auffallen, bevor sie ablaufen — nicht danach.
	DocumentDate string `gorm:"size:10;index" json:"documentDate,omitempty"`
	ValidUntil   string `gorm:"size:10;index" json:"validUntil,omitempty"`

	Note      string    `gorm:"size:500;serializer:encrypted" json:"note,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// DisplayTitle is the name to show: the given title, or the file name.
func (d *AssetDocument) DisplayTitle() string {
	if strings.TrimSpace(d.Title) != "" {
		return d.Title
	}
	return d.FileName
}

// Validate enforces what has to hold before a document is stored.
func (d *AssetDocument) Validate() error {
	if d.AssetID == 0 {
		return fmt.Errorf("das Dokument gehört zu keinem Anlagegut")
	}
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
