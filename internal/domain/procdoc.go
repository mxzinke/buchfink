package domain

import (
	"context"
	"time"
)

// ProcedureDocumentation ist eine erzeugte Fassung der Verfahrensdokumentation.
//
// Sie wird als Datensatz geführt und nicht nur als Datei geschrieben, weil die
// Fassungen eine Historie bilden: GoBD Rz. 151 verlangt, dass zu jedem
// Geschäftsjahr die Verfahrensdokumentation vorliegt, die *damals* gegolten
// hat. Eine Datei, die bei jeder Erzeugung überschrieben wird, kann das nicht
// leisten.
type ProcedureDocumentation struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// Version ist die fortlaufende Fassungsbezeichnung, z. B. "2026-09-05-1".
	Version   string    `gorm:"size:40;not null;uniqueIndex" json:"version"`
	CreatedAt time.Time `gorm:"not null;index" json:"createdAt"`
	// FiscalYear ist das Geschäftsjahr, für das die Fassung erzeugt wurde.
	FiscalYear  int    `gorm:"index" json:"fiscalYear"`
	CompanyName string `gorm:"size:255;serializer:encrypted" json:"companyName"`
	AppVersion  string `gorm:"size:60" json:"appVersion"`
	RuleVersion string `gorm:"size:20" json:"ruleVersion"`
	Actor       string `gorm:"size:120" json:"actor"`

	// StoredPath ist der Ort der Datei im Belegspeicher, relativ zum
	// Datenordner (dokumente/verfahrensdokumentation/<Prüfsumme>.md).
	//
	// Der Pfad und kein Belegverweis: die Fassung ist kein Beleg — sie hat
	// keine Belegnummer, wird nicht gebucht und ist keinem
	// Geschäftsvorfall zugeordnet. Ohne den Pfad wäre sie erzeugt und nicht wiederzufinden;
	// das Prüferpaket legt sie über ihn bei.
	StoredPath string `gorm:"size:500" json:"storedPath,omitempty"`
	// FileName, SHA256 und Size beschreiben die abgelegte Datei. Die Prüfsumme
	// steht hier, damit sich eine Fassung auch dann noch identifizieren lässt,
	// wenn die Datei woanders liegt.
	FileName string `gorm:"size:255" json:"fileName"`
	SHA256   string `gorm:"size:64" json:"sha256"`
	Size     int64  `json:"size"`

	// Dieselben Angaben für den PDF-Satz derselben Fassung. Beide Formen
	// stehen nebeneinander und nicht statt einander: das Markdown ist die
	// maschinenlesbare Fassung, aus der auch das PDF gesetzt wird, das PDF die
	// Form, die ein Prüfer in die Hand nimmt. Leer heißt: der Satz war nicht
	// möglich (kein Renderer verdrahtet) — die Fassung gilt trotzdem, weil das
	// Markdown die Aussage enthält.
	PDFFileName   string `gorm:"size:255" json:"pdfFileName,omitempty"`
	PDFStoredPath string `gorm:"size:500" json:"pdfStoredPath,omitempty"`
	PDFSHA256     string `gorm:"size:64" json:"pdfSha256,omitempty"`
	PDFSize       int64  `json:"pdfSize,omitempty"`
}

// ProcedureDocumentationRepository persistiert die Fassungen.
type ProcedureDocumentationRepository interface {
	Create(ctx context.Context, doc *ProcedureDocumentation) error
	FindAll(ctx context.Context) ([]ProcedureDocumentation, error)
	// FindByID liefert eine einzelne Fassung; nil heißt: es gibt sie nicht.
	// Gebraucht für die Herausgabe: wer eine Fassung an den Prüfer gibt, gibt
	// eine bestimmte heraus, und zu jedem Geschäftsjahr gilt die, die damals
	// galt.
	FindByID(ctx context.Context, id uint) (*ProcedureDocumentation, error)
	// CountForDay zählt die Fassungen eines Tages. Die Fassungsbezeichnung
	// hängt daran: zwei Fassungen an einem Tag dürfen nicht gleich heißen.
	CountForDay(ctx context.Context, day string) (int64, error)
}

// OrganisationTexts sind die unternehmensindividuellen Teile der
// Verfahrensdokumentation.
//
// Sie sind Freitext, weil sie es sein müssen: wer scannt, wer prüft, wer
// freigibt und wie vertreten wird, weiß nur das Unternehmen. Buchfink gibt
// Muster vor, damit die Felder nicht leer bleiben — eine
// Verfahrensdokumentation mit leeren Abschnitten ist schlechter als keine, weil
// sie Vollständigkeit behauptet.
type OrganisationTexts struct {
	// Responsibilities: wer die Buchführung führt und wer sie verantwortet.
	Responsibilities string `json:"responsibilities"`
	// ReceiptFlow: wie Belege ins Haus kommen, wer sie erfasst und wer prüft.
	ReceiptFlow string `json:"receiptFlow"`
	// Scanning: die Organisationsanweisung zum Einscannen von Papier.
	Scanning string `json:"scanning"`
	// Approval: wer eine Buchung freigibt und wer festschreibt.
	Approval string `json:"approval"`
	// Substitution: die Vertretungsregelung.
	Substitution string `json:"substitution"`
	// Backup: wer die Sicherung überwacht und wo die Kopien liegen.
	Backup string `json:"backup"`
	// Notes: alles Weitere.
	Notes string `json:"notes"`
}

// Die Schlüssel, unter denen die Texte in den Einstellungen liegen.
const (
	SettingOrgResponsibilities = "org_responsibilities"
	SettingOrgReceiptFlow      = "org_receipt_flow"
	SettingOrgScanning         = "org_scanning"
	SettingOrgApproval         = "org_approval"
	SettingOrgSubstitution     = "org_substitution"
	SettingOrgBackup           = "org_backup"
	SettingOrgNotes            = "org_notes"
	// SettingSystemChangeDate ist der Umstellungszeitpunkt bei der Übernahme
	// aus einem Altsystem. Nach ihm richtet sich die Fünfjahresfrist des § 147
	// Abs. 6 Satz 6 AO: so lange muss das Altsystem für den Datenzugriff
	// verfügbar bleiben.
	SettingSystemChangeDate = "system_change_date"
)

// DefaultOrganisationTexts sind die Muster, mit denen die Freitextfelder
// vorbelegt werden.
//
// Sie sind als Vorschlag formuliert und nicht als Behauptung: wer sie
// unverändert übernimmt, hat eine Verfahrensdokumentation, die den
// Einzelplatzbetrieb beschreibt — und das ist für ein Ein-Personen-Unternehmen
// die Wahrheit.
func DefaultOrganisationTexts() OrganisationTexts {
	return OrganisationTexts{
		Responsibilities: "Die Buchführung wird von der Inhaberin bzw. dem Inhaber des Unternehmens selbst geführt. " +
			"Es gibt keine weiteren Bearbeiter. Die Bearbeiterkennung besteht aus Benutzerkonto und Rechnername " +
			"des verwendeten Rechners und wird an jeder Buchung und jedem Protokolleintrag festgehalten.",
		ReceiptFlow: "Eingehende Belege werden unmittelbar nach Eingang in Buchfink abgelegt und dort mit " +
			"Belegdatum, Aussteller und Betrag erfasst. Die Buchung erfolgt innerhalb von zehn Tagen " +
			"(GoBD Rz. 47). Papierbelege werden zusätzlich im Original abgeheftet.",
		Scanning: "Papierbelege werden mit einem Flachbett- oder Einzugsscanner in Farbe und mindestens 300 dpi " +
			"als PDF erfasst. Nach dem Einscannen wird die Lesbarkeit am Bildschirm geprüft. " +
			"Ersetzendes Scannen findet nicht statt: das Papieroriginal wird bis zum Ablauf der " +
			"Aufbewahrungsfrist aufbewahrt.",
		Approval: "Buchung und Freigabe liegen in einer Hand. Die Kontrolle erfolgt über die Prüfläufe vor der " +
			"Festschreibung und über den monatlichen Abgleich der Bankkonten mit den Kontoauszügen.",
		Substitution: "Im Verhinderungsfall übernimmt der steuerliche Berater den Zugriff. Ihm liegen der " +
			"Wiederherstellungsschlüssel und der Speicherort der Sicherungen vor.",
		Backup: "Die Sicherung läuft nach dem in den Einstellungen hinterlegten Rhythmus auf ein Ziel außerhalb " +
			"des Datenordners. Die Läufe werden protokolliert; einmal jährlich wird eine Sicherung " +
			"probeweise wiederhergestellt und geprüft.",
		Notes: "",
	}
}
