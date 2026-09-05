package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// VatPeriodType is the length of a Voranmeldungszeitraum (§ 18 Abs. 2 UStG).
type VatPeriodType string

const (
	VatPeriodMonth   VatPeriodType = "month"
	VatPeriodQuarter VatPeriodType = "quarter"
	VatPeriodYear    VatPeriodType = "year"
)

// Valid meldet, ob der Zeitraumtyp einer der drei bekannten ist.
func (p VatPeriodType) Valid() bool {
	switch p {
	case VatPeriodMonth, VatPeriodQuarter, VatPeriodYear:
		return true
	}
	return false
}

// VatReturnStatus ist der Stand einer Voranmeldung.
//
// Es gibt nur zwei Stände, und das ist Absicht. Buchfink übermittelt nicht
// selbst (kein ERiC): das Blatt wird erzeugt, in Mein ELSTER eingegeben und die
// Übermittlung anschließend mit dem Transferticket bestätigt. Ein Zwischenstand
// „übermittelt, aber noch ohne Ticket" wäre eine Behauptung ohne Nachweis.
type VatReturnStatus string

const (
	VatReturnDraft     VatReturnStatus = "draft"
	VatReturnSubmitted VatReturnStatus = "submitted"
)

// VatReturnLine ist eine Zeile des Vordrucks USt 1 A.
//
// Bemessungsgrundlage und Steuerbetrag stehen in einer Zeile, weil sie im
// Vordruck in einer Zeile stehen — teils unter derselben Kennziffer (81, 86,
// 89), teils unter zweien (35/36, 46/47). TaxCode benennt die zweite; ist sie
// leer, trägt die Zeile ihren Steuerbetrag unter der eigenen Kennziffer.
type VatReturnLine struct {
	Code      string `json:"code"`
	Label     string `json:"label"`
	Reference string `json:"reference,omitempty"`

	// HasBase und HasTax sagen, welche Felder der Vordruck in dieser Zeile
	// überhaupt kennt. Ohne sie ließe sich eine leere Bemessungsgrundlage nicht
	// von einer unterscheiden, die es in dieser Zeile gar nicht gibt.
	HasBase bool   `json:"hasBase"`
	Base    Cents  `json:"base"`
	HasTax  bool   `json:"hasTax"`
	TaxCode string `json:"taxCode,omitempty"`
	Tax     Cents  `json:"tax"`

	// ExpectedTax ist die aus der Bemessungsgrundlage errechnete Steuer. Der
	// Vordruck trägt die *gebuchte* Steuer, nicht die nachgerechnete: die
	// Rundung je Rechnung ist die richtige, und eine Nachrechnung über den
	// Monatsumsatz weicht regelmäßig um Cent ab. Damit die Abweichung nicht
	// unbemerkt bleibt, steht sie daneben.
	ExpectedTax Cents `json:"expectedTax"`

	// EntryIDs sind die Buchungen, aus denen die Zeile entstanden ist — der
	// Drill-down. Ohne ihn ist eine Kennziffer eine Zahl, die niemand prüfen
	// kann.
	EntryIDs []uint `json:"entryIds,omitempty"`
}

// Deviation ist der Unterschied zwischen gebuchter und rechnerischer Steuer.
func (l *VatReturnLine) Deviation() Cents {
	if !l.HasTax || !l.HasBase {
		return 0
	}
	return l.Tax - l.ExpectedTax
}

// VatLateEntry ist ein Nachtrag: eine Buchung, deren Voranmeldungszeitraum
// bereits übermittelt ist.
//
// Sie wird nicht stillschweigend in den laufenden Zeitraum geschoben. Das wäre
// bequem und falsch: § 18 Abs. 1 UStG ordnet den Umsatz dem Zeitraum zu, in dem
// er entstanden ist, und eine Verschiebung machte aus zwei richtigen
// Voranmeldungen zwei falsche.
type VatLateEntry struct {
	EntryID     uint   `json:"entryId"`
	EntryNumber string `json:"entryNumber"`
	BookingDate string `json:"bookingDate"`
	// PeriodKey ist der Zeitraum, in den die Buchung gehört.
	PeriodKey   string `json:"periodKey"`
	Description string `json:"description"`
	Code        string `json:"code"`
	Base        Cents  `json:"base"`
	Tax         Cents  `json:"tax"`
}

// VatReturn ist die Umsatzsteuer-Voranmeldung eines Zeitraums.
//
// Sie ist eine Entität und keine Auswertung: was übermittelt wurde, muss
// nachweisbar bleiben, auch wenn sich das Journal danach durch Stornos ändert.
// Die Entität *ist* das Übermittlungsprotokoll — Datum und Transferticket
// stehen an ihr.
type VatReturn struct {
	ID         uint          `gorm:"primaryKey" json:"id"`
	FiscalYear int           `gorm:"index;not null" json:"fiscalYear"`
	PeriodType VatPeriodType `gorm:"size:10;not null" json:"periodType"`
	// PeriodKey ist der Schlüssel des Zeitraums: "2026-03", "2026-Q1", "2026".
	PeriodKey  string `gorm:"size:20;not null;index" json:"periodKey"`
	PeriodFrom string `gorm:"size:10;not null" json:"periodFrom"`
	PeriodTo   string `gorm:"size:10;not null" json:"periodTo"`

	// IsCorrection setzt die Kennziffer 10 des Vordrucks: berichtigte Anmeldung.
	IsCorrection bool  `gorm:"not null;default:false" json:"isCorrection"`
	CorrectsID   *uint `gorm:"index" json:"correctsId,omitempty"`

	Status         VatReturnStatus `gorm:"size:20;not null;default:'draft';index" json:"status"`
	SubmittedAt    string          `gorm:"size:10" json:"submittedAt,omitempty"`
	TransferTicket string          `gorm:"size:100" json:"transferTicket,omitempty"`
	SubmissionNote string          `gorm:"size:500" json:"submissionNote,omitempty"`

	// Payable ist die verbleibende Vorauszahlung (Kennziffer 83); negativ
	// bedeutet Überschuss zugunsten des Unternehmers.
	Payable Cents `gorm:"not null;default:0" json:"payable"`
	// DueDate ist die Fälligkeit nach § 18 Abs. 1 UStG, ggf. mit
	// Dauerfristverlängerung.
	DueDate string `gorm:"size:10" json:"dueDate,omitempty"`

	// ProgramVersion hält fest, welche Fassung des Programms das Blatt gerechnet
	// hat. Ohne sie ließe sich eine übermittelte Anmeldung später nicht
	// nachvollziehen: die Zuordnung Steuerfall → Kennziffer kann sich ändern.
	ProgramVersion string `gorm:"size:40" json:"programVersion,omitempty"`

	// FiguresJSON und LateEntriesJSON sind die Speicherform der beiden Listen.
	// Die Kennziffern eines Vordrucks sind kein Datenmodell, sondern ein
	// Formular: als Spalten wäre jede Änderung des Vordrucks eine Migration.
	FiguresJSON     string `gorm:"type:text" json:"-"`
	LateEntriesJSON string `gorm:"type:text" json:"-"`

	Figures     []VatReturnLine `gorm:"-" json:"figures"`
	LateEntries []VatLateEntry  `gorm:"-" json:"lateEntries"`

	CreatedAt time.Time `json:"createdAt"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
//
// Die Anmeldung geht als JSON an die Oberfläche, und ein nil-Slice wird dort zu
// `null`. Die Ansicht liest die Listen ohne Umweg — `figures.length`,
// `lateEntries.length` —, und `null.length` bringt sie zu Fall. Betroffen wäre
// der Regelfall: der Zeitraum ohne Nachtrag. Die Zusage „leer statt null" gehört
// deshalb an die Entität und nicht an jede Stelle, die sie erzeugt oder lädt.
func (r *VatReturn) EnsureLists() {
	if r.Figures == nil {
		r.Figures = make([]VatReturnLine, 0)
	}
	if r.LateEntries == nil {
		r.LateEntries = make([]VatLateEntry, 0)
	}
}

// Line liefert eine Kennziffernzeile.
func (r *VatReturn) Line(code string) (*VatReturnLine, bool) {
	for i := range r.Figures {
		if r.Figures[i].Code == code {
			return &r.Figures[i], true
		}
	}
	return nil, false
}

// Tax liefert den Steuerbetrag einer Kennziffer, 0 wenn es sie nicht gibt.
func (r *VatReturn) Tax(code string) Cents {
	if l, ok := r.Line(code); ok {
		return l.Tax
	}
	return 0
}

// Base liefert die Bemessungsgrundlage einer Kennziffer.
func (r *VatReturn) Base(code string) Cents {
	if l, ok := r.Line(code); ok {
		return l.Base
	}
	return 0
}

// EntryIDs liefert die Buchungen hinter einer Kennziffer (Drill-down).
func (r *VatReturn) EntryIDs(code string) []uint {
	if l, ok := r.Line(code); ok {
		return l.EntryIDs
	}
	return nil
}

// ValidateSubmission prüft, was zur Bestätigung einer Übermittlung gehört.
//
// Das Transferticket ist Pflicht, weil es der einzige Nachweis ist, dass die
// Anmeldung beim Finanzamt angekommen ist. Ein Häkchen ohne Ticket wäre eine
// Selbstauskunft; das Format bleibt frei, weil ELSTER es über die Jahre
// geändert hat.
func (r *VatReturn) ValidateSubmission(date, ticket string) error {
	if r.Status == VatReturnSubmitted {
		return fmt.Errorf(
			"die Voranmeldung %s ist bereits am %s als übermittelt bestätigt (Transferticket %s) und kann nicht geändert werden. "+
				"Eine Änderung geschieht über eine berichtigte Anmeldung",
			r.PeriodKey, r.SubmittedAt, r.TransferTicket)
	}
	if len(date) != 10 {
		return fmt.Errorf("zur Bestätigung gehört das Datum der Übermittlung (erwartet JJJJ-MM-TT)")
	}
	if strings.TrimSpace(ticket) == "" {
		return fmt.Errorf(
			"zur Bestätigung gehört das Transferticket aus Mein ELSTER. Es ist der Nachweis, dass die Anmeldung angekommen ist")
	}
	return nil
}

// VatReturnRepository persistiert die Voranmeldungen.
type VatReturnRepository interface {
	Create(ctx context.Context, r *VatReturn) error
	Update(ctx context.Context, r *VatReturn) error
	FindByID(ctx context.Context, id uint) (*VatReturn, error)
	FindByFiscalYear(ctx context.Context, fiscalYear int) ([]VatReturn, error)
	// FindByPeriod liefert alle Anmeldungen eines Zeitraums, die jüngste zuerst
	// — die Berichtigung steht vor der berichtigten Anmeldung.
	FindByPeriod(ctx context.Context, periodKey string) ([]VatReturn, error)
	Delete(ctx context.Context, id uint) error
}

// ZMLineKind ist die Art des gemeldeten Umsatzes in der Zusammenfassenden
// Meldung (§ 18a UStG).
type ZMLineKind string

const (
	// ZMKindSupply ist die innergemeinschaftliche Lieferung.
	ZMKindSupply ZMLineKind = "L"
	// ZMKindService ist die im übrigen Gemeinschaftsgebiet steuerpflichtige
	// sonstige Leistung nach § 3a Abs. 2 UStG.
	ZMKindService ZMLineKind = "S"
	// ZMKindTriangular ist das innergemeinschaftliche Dreiecksgeschäft. Buchfink
	// unterstützt es nicht; die Art steht hier, damit die Spalte des
	// BZSt-Formats einen benannten Wert hat und nicht eine leere Stelle.
	ZMKindTriangular ZMLineKind = "D"
)

// ZMLine ist eine Meldezeile: je USt-IdNr. und Art ein Betrag.
type ZMLine struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	ZMReturnID  uint       `gorm:"index;not null" json:"zmReturnId"`
	CountryCode string     `gorm:"size:2;not null" json:"countryCode"`
	VatID       string     `gorm:"size:20;not null" json:"vatId"`
	Kind        ZMLineKind `gorm:"size:1;not null" json:"kind"`
	Amount      Cents      `gorm:"not null" json:"amount"`

	ContactID   uint   `gorm:"index" json:"contactId"`
	ContactName string `gorm:"-" json:"contactName,omitempty"`
	// EntryIDs trägt den Drill-down wie bei der Voranmeldung.
	EntryIDsJSON string `gorm:"type:text" json:"-"`
	EntryIDs     []uint `gorm:"-" json:"entryIds,omitempty"`
}

// ZMLateEntry ist ein Nachtrag zur Zusammenfassenden Meldung: ein
// meldepflichtiger Umsatz, dessen Meldezeitraum bereits übermittelt ist und der
// dort nicht gemeldet wurde.
//
// Er wird so wenig stillschweigend in den laufenden Zeitraum geschoben wie der
// Nachtrag zur Voranmeldung. § 18a Abs. 10 UStG verlangt die Berichtigung der
// ursprünglichen Meldung binnen eines Monats — eine Verschiebung machte aus
// zwei richtigen Meldungen zwei falsche, und das Bundeszentralamt gleicht die
// Beträge mit den Erwerbsmeldungen der anderen Mitgliedstaaten ab.
type ZMLateEntry struct {
	EntryID     uint       `json:"entryId"`
	EntryNumber string     `json:"entryNumber"`
	PeriodKey   string     `json:"periodKey"`
	Date        string     `json:"date"`
	VatID       string     `json:"vatId,omitempty"`
	Kind        ZMLineKind `json:"kind"`
	Amount      Cents      `json:"amount"`
}

// ZMReturn ist die Zusammenfassende Meldung eines Meldezeitraums.
type ZMReturn struct {
	ID         uint          `gorm:"primaryKey" json:"id"`
	FiscalYear int           `gorm:"index;not null" json:"fiscalYear"`
	PeriodType VatPeriodType `gorm:"size:10;not null" json:"periodType"`
	PeriodKey  string        `gorm:"size:20;not null;index" json:"periodKey"`
	PeriodFrom string        `gorm:"size:10;not null" json:"periodFrom"`
	PeriodTo   string        `gorm:"size:10;not null" json:"periodTo"`

	IsCorrection bool  `gorm:"not null;default:false" json:"isCorrection"`
	CorrectsID   *uint `gorm:"index" json:"correctsId,omitempty"`

	Status         VatReturnStatus `gorm:"size:20;not null;default:'draft';index" json:"status"`
	SubmittedAt    string          `gorm:"size:10" json:"submittedAt,omitempty"`
	TransferTicket string          `gorm:"size:100" json:"transferTicket,omitempty"`
	SubmissionNote string          `gorm:"size:500" json:"submissionNote,omitempty"`
	DueDate        string          `gorm:"size:10" json:"dueDate,omitempty"`

	TotalSupplies Cents `gorm:"not null;default:0" json:"totalSupplies"`
	TotalServices Cents `gorm:"not null;default:0" json:"totalServices"`

	Lines []ZMLine `gorm:"foreignKey:ZMReturnID;constraint:OnDelete:CASCADE" json:"lines"`

	// Reconciliation ist die Abstimmung gegen die Voranmeldungen desselben
	// Zeitraums. Sie wird nicht gespeichert: sie ist eine Aussage über den
	// heutigen Stand beider Meldungen, nicht über den von damals.
	Reconciliation *ZMReconciliation `gorm:"-" json:"reconciliation,omitempty"`
	// Findings sind die Befunde, die eine Bestätigung verhindern — allen voran
	// eine fehlende USt-IdNr. am Kontakt.
	Findings []string `gorm:"-" json:"findings"`
	// LateEntries sind die Nachträge zu bereits übermittelten Meldezeiträumen.
	// Sie werden wie die Abstimmung nicht gespeichert: sie sind eine Aussage
	// über den heutigen Journalstand gegen die übermittelten Meldungen, nicht
	// über den Stand von damals.
	LateEntries []ZMLateEntry `gorm:"-" json:"lateEntries"`

	CreatedAt time.Time `json:"createdAt"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere — aus demselben Grund wie
// bei der Voranmeldung: die Oberfläche darf für „nichts zu melden" kein `null`
// lesen. Die Listen tragen deshalb auch kein `omitempty`: ein fehlendes Feld
// wäre in der Ansicht `undefined` und damit derselbe Absturz.
func (r *ZMReturn) EnsureLists() {
	if r.Lines == nil {
		r.Lines = make([]ZMLine, 0)
	}
	if r.LateEntries == nil {
		r.LateEntries = make([]ZMLateEntry, 0)
	}
	if r.Findings == nil {
		r.Findings = make([]string, 0)
	}
}

// ZMReconciliation stellt die Summen der Meldung den Kennziffern 41 und 21 der
// Voranmeldungen desselben Zeitraums gegenüber.
//
// Beide Meldungen beschreiben denselben Umsatz. Gehen sie auseinander, ist eine
// von beiden falsch — und das Finanzamt gleicht sie ab (§ 18a Abs. 8 UStG).
// ScopeKey und ScopeLabel benennen den Zeitraum, über den verglichen wurde.
// Er ist nicht immer der Meldezeitraum: wer monatlich meldet und
// vierteljährlich voranmeldet (§ 18a Abs. 1 Satz 2 UStG neben § 18 Abs. 2 Satz 2
// UStG), hat für einen ZM-Monat gar keine Anmeldung. Verglichen wird dann das
// umschließende Quartal gegen die Summe seiner drei ZM-Monate — sonst stünde in
// genau der Lage, für die es die Monatsmeldung gibt, immer die volle Summe als
// Abweichung.
type ZMReconciliation struct {
	ScopeKey        string `json:"scopeKey,omitempty"`
	ScopeLabel      string `json:"scopeLabel,omitempty"`
	SuppliesZM      Cents  `json:"suppliesZm"`
	SuppliesVat     Cents  `json:"suppliesVat"`
	ServicesZM      Cents  `json:"servicesZm"`
	ServicesVat     Cents  `json:"servicesVat"`
	VatReturnsFound int    `json:"vatReturnsFound"`
}

// SuppliesDifference ist die Abweichung der ig. Lieferungen gegen Kz 41.
func (r *ZMReconciliation) SuppliesDifference() Cents { return r.SuppliesZM - r.SuppliesVat }

// ServicesDifference ist die Abweichung der sonstigen Leistungen gegen Kz 21.
func (r *ZMReconciliation) ServicesDifference() Cents { return r.ServicesZM - r.ServicesVat }

// ZMReturnRepository persistiert die Zusammenfassenden Meldungen.
type ZMReturnRepository interface {
	Create(ctx context.Context, r *ZMReturn) error
	Update(ctx context.Context, r *ZMReturn) error
	FindByID(ctx context.Context, id uint) (*ZMReturn, error)
	FindByFiscalYear(ctx context.Context, fiscalYear int) ([]ZMReturn, error)
	FindByPeriod(ctx context.Context, periodKey string) ([]ZMReturn, error)
	Delete(ctx context.Context, id uint) error
}

// DeadlineDone hält den Haken an einem Termin fest, der sich nicht aus den
// Daten ergibt.
//
// Alles, was ableitbar ist — die übermittelte Voranmeldung, die Festschreibung,
// die Aufstellung —, wird abgeleitet und nicht abgehakt. Übrig bleiben Termine
// wie die Umsatzsteuer-Jahreserklärung, von der Buchfink nichts sieht. Der
// Haken steht in der Datenbank und nicht im localStorage: er ist eine Aussage
// über den Mandanten und nicht über den Browser.
type DeadlineDone struct {
	Key       string    `gorm:"primaryKey;size:100" json:"key"`
	DoneOn    string    `gorm:"size:10;not null" json:"doneOn"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// DeadlineRepository persistiert die manuellen Haken.
type DeadlineRepository interface {
	FindAll(ctx context.Context) ([]DeadlineDone, error)
	Mark(ctx context.Context, key, doneOn string) error
}
