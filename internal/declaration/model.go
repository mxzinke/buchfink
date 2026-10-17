// Package declaration stellt die Jahressteuererklärungen einer GmbH oder UG
// formularnah zusammen: Umsatzsteuererklärung (USt 2 A),
// Körperschaftsteuererklärung (KSt 1 mit Anlagen) und Gewerbesteuererklärung
// (GewSt 1 A). Buchfink übermittelt nicht; die Erklärung ist eine Eingabehilfe
// für Mein ELSTER und lässt sich als JSON ausgeben.
//
// Zeilen, Kennzahlen und Bezeichnungen stehen je Vordruckjahr im eingebetteten
// Zeilenkatalog (catalog/*.json), mit Prüfvermerk und Quelle je Zeile. Die
// Rechnung hängt am Schlüssel der Zeile, nicht an ihrer Nummer. Das Paket kennt
// weder Datenbank noch Dienste: es rechnet aus den Größen, die ihm der Dienst
// übergibt.
package declaration

import (
	"encoding/json"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Schema ist die Kennung des JSON-Exports.
const Schema = "buchfink.erklaerung/1"

// Kind ist die Art einer Erklärung. Die Werte sind die der Stammdaten
// (domain.Declaration*) und des MCP-Werkzeugs `erklaerung`.
type Kind string

const (
	KindVatYear      Kind = domain.DeclarationAnnualVat
	KindCorporateTax Kind = domain.DeclarationCorporateTax
	KindTradeTax     Kind = domain.DeclarationTradeTax
)

// Kinds sind die Arten, die dieses Paket rechnet.
func Kinds() []Kind { return []Kind{KindVatYear, KindCorporateTax, KindTradeTax} }

// Valid meldet, ob die Art eine der gerechneten ist.
func (k Kind) Valid() bool {
	for _, known := range Kinds() {
		if k == known {
			return true
		}
	}
	return false
}

// Label benennt die Art.
func (k Kind) Label() string {
	switch k {
	case KindVatYear:
		return "Umsatzsteuererklärung"
	case KindCorporateTax:
		return "Körperschaftsteuererklärung"
	case KindTradeTax:
		return "Gewerbesteuererklärung"
	}
	return string(k)
}

// Declaration ist eine Erklärung für ein Jahr, wie Buchfink sie zeigt und
// exportiert.
type Declaration struct {
	Schema   string  `json:"schema"`
	Kind     Kind    `json:"art"`
	Title    string  `json:"titel"`
	Period   Period  `json:"zeitraum"`
	FormYear int     `json:"vordruckjahr"`
	Subject  Subject `json:"steuerpflichtiger"`
	Forms    []Form  `json:"formulare"`
	// Reconciliation ist die Abstimmung gegen die Voranmeldungen; nur bei der
	// Umsatzsteuererklärung.
	Reconciliation []ReconciliationRow `json:"abstimmung,omitempty"`
	Inputs         []Input             `json:"eingaben"`
	Findings       []Finding           `json:"befunde"`
	Status         Status              `json:"status"`
	Basis          Basis               `json:"grundlage"`
	FiguresHash    string              `json:"zahlen_hash"`
	Submission     *Submission         `json:"uebermittlung,omitempty"`
}

// Period ist der Besteuerungszeitraum. Bei abweichendem Wirtschaftsjahr ist es
// das Wirtschaftsjahr, das im Jahr endet (§ 7 Abs. 4 KStG, § 10 Abs. 2 GewStG).
type Period struct {
	Year int    `json:"jahr"`
	From string `json:"von"`
	To   string `json:"bis"`
}

// Subject ist der Steuerpflichtige aus den Stammdaten.
type Subject struct {
	Name            string  `json:"name"`
	LegalForm       string  `json:"rechtsform"`
	TaxNumber       string  `json:"steuernummer"`
	TaxNumberElster string  `json:"steuernummer_elster"`
	TaxOfficeNumber string  `json:"finanzamtsnummer"`
	TaxOffice       string  `json:"finanzamt"`
	FederalState    string  `json:"bundesland"`
	VatID           string  `json:"ust_id"`
	Address         Address `json:"anschrift"`
}

// Address ist die Geschäftsanschrift.
type Address struct {
	Street      string `json:"strasse"`
	HouseNumber string `json:"hausnummer"`
	Addition    string `json:"adressergaenzung,omitempty"`
	PostalCode  string `json:"plz"`
	City        string `json:"ort"`
	Country     string `json:"land"`
}

// Requirement sagt, ob ein Vordruck abzugeben ist.
type Requirement string

const (
	RequirementMandatory Requirement = "pflicht"
	RequirementIfNeeded  Requirement = "bei_bedarf"
	RequirementNone      Requirement = "entfaellt"
	// RequirementControl kennzeichnet keine Anlage, sondern Buchfinks
	// Kontrollrechnung neben der Erklärung.
	RequirementControl Requirement = "kontrolle"
)

// Form ist ein Vordruck oder eine Anlage.
type Form struct {
	ID       string      `json:"id"`
	Title    string      `json:"titel"`
	Required Requirement `json:"erforderlich"`
	Reason   string      `json:"grund,omitempty"`
	Source   Ref         `json:"quelle"`
	Sections []Section   `json:"abschnitte"`
	Summary  Summary     `json:"uebersicht"`
}

// Section ist ein Abschnitt eines Vordrucks.
type Section struct {
	ID      string  `json:"id"`
	Title   string  `json:"titel"`
	Lines   []Line  `json:"zeilen"`
	Summary Summary `json:"uebersicht"`
}

// Summary zählt die Zeilen eines Abschnitts oder Vordrucks nach Stand. Die
// Oberfläche klappt danach auf: offene und vorgeschlagene zuerst, leere zuletzt.
type Summary struct {
	Lines    int `json:"zeilen"`
	Filled   int `json:"mit_wert"`
	Open     int `json:"offen"`
	Proposed int `json:"vorgeschlagen"`
	Inputs   int `json:"eingaben"`
}

// Unit ist die Art eines Werts.
type Unit string

const (
	UnitEuro    Unit = "euro_voll"
	UnitCents   Unit = "euro_cent"
	UnitText    Unit = "text"
	UnitYesNo   Unit = "ja_nein"
	UnitDate    Unit = "datum"
	UnitPercent Unit = "prozent"
	UnitKey     Unit = "schluessel"
)

// Rounding sagt, wie ein Betrag in den Vordruck gelangt. Intern bleibt er in
// Cent.
type Rounding string

const (
	RoundingNone Rounding = "keine"
	// RoundingTruncate schneidet die Cent ab (Bemessungsgrundlagen der
	// Umsatzsteuer, zum Betrag hin).
	RoundingTruncate Rounding = "abschneiden"
	// RoundingFavour rundet zugunsten der Gesellschaft auf volle Euro.
	RoundingFavour Rounding = "zugunsten"
)

// LineStatus ist der Stand einer Zeile.
type LineStatus string

const (
	// StatusAuto: aus den Büchern gerechnet, ohne Wertungsfrage.
	StatusAuto LineStatus = "auto"
	// StatusProposed: Wert liegt vor, braucht die Bestätigung der zuständigen
	// Stelle (Agent mit Rückfrage oder Mensch).
	StatusProposed LineStatus = "vorgeschlagen"
	// StatusInput: Wert aus einer bestätigten Eingabe.
	StatusInput LineStatus = "eingabe"
	// StatusOpen: Wert fehlt; ohne ihn ist die Erklärung nicht vollständig.
	StatusOpen LineStatus = "offen"
	// StatusNotApplicable: trifft für die Gesellschaft nicht zu.
	StatusNotApplicable LineStatus = "entfaellt"
)

// Responsibility ist, wer den Wert einer Zeile verantwortet.
type Responsibility string

const (
	OwnerBooks Responsibility = "buecher"
	// OwnerAgent: ein Agent schlägt vor und setzt mit Rückfrage beim Menschen.
	OwnerAgent Responsibility = "agent_rueckfrage"
	// OwnerHuman: nur der Mensch, etwa ein Wert aus einem Bescheid oder ein
	// Antrag.
	OwnerHuman Responsibility = "mensch"
)

// Line ist eine Zeile eines Vordrucks.
type Line struct {
	Key string `json:"schluessel"`
	// Row ist die Zeilennummer des Vordrucks; leer, wo keine amtliche Quelle sie
	// nennt.
	Row      string         `json:"zeile,omitempty"`
	Code     string         `json:"kz,omitempty"`
	Label    string         `json:"bezeichnung"`
	Value    Value          `json:"wert"`
	Unit     Unit           `json:"einheit"`
	Rounding Rounding       `json:"rundung"`
	Origin   Origin         `json:"herkunft"`
	Status   LineStatus     `json:"status"`
	Owner    Responsibility `json:"zustaendig"`
	// InputAllowed sagt, ob der Wert als Eingabe gesetzt werden kann.
	InputAllowed bool `json:"eingabe_moeglich"`
	Required     bool `json:"pflicht"`
	// Provisional: die Zeile rechnet aus einer offenen oder vorgeschlagenen.
	Provisional bool         `json:"vorlaeufig,omitempty"`
	Legal       []string     `json:"rechtsgrundlage,omitempty"`
	Check       Verification `json:"pruefvermerk"`
	Notes       []string     `json:"hinweise,omitempty"`
}

// Verification ist der Prüfvermerk einer Zeile.
type Verification struct {
	// OnForm: Zeile, Kennzahl und Bezeichnung sind am amtlichen Vordruck des
	// Jahres geprüft.
	OnForm bool   `json:"am_vordruck_geprueft"`
	Source string `json:"quelle"`
}

// Value ist der Wert einer Zeile: ein Betrag in Cent, ein Text, ja/nein oder
// ein Datum.
type Value struct {
	Cents *domain.Cents
	Text  *string
	Bool  *bool
	Date  *string
}

// Empty meldet, ob kein Wert vorliegt.
func (v Value) Empty() bool {
	return v.Cents == nil && v.Text == nil && v.Bool == nil && v.Date == nil
}

// Amount ist der Betrag, null ohne Betrag.
func (v Value) Amount() domain.Cents {
	if v.Cents == nil {
		return 0
	}
	return *v.Cents
}

// IsZero meldet einen leeren Wert oder einen Betrag von null.
func (v Value) IsZero() bool {
	switch {
	case v.Cents != nil:
		return *v.Cents == 0
	case v.Text != nil:
		return *v.Text == ""
	case v.Bool != nil:
		return !*v.Bool
	case v.Date != nil:
		return *v.Date == ""
	}
	return true
}

// Canonical ist die Form des Werts im Zahlen-Hash.
func (v Value) Canonical() string {
	switch {
	case v.Cents != nil:
		return v.Cents.Decimal()
	case v.Text != nil:
		return "t:" + *v.Text
	case v.Bool != nil:
		if *v.Bool {
			return "b:1"
		}
		return "b:0"
	case v.Date != nil:
		return "d:" + *v.Date
	}
	return ""
}

// Cents baut einen Betragswert.
func Cents(c domain.Cents) Value { return Value{Cents: &c} }

// Text baut einen Textwert.
func Text(s string) Value { return Value{Text: &s} }

// Bool baut einen Ja/Nein-Wert.
func Bool(b bool) Value { return Value{Bool: &b} }

// Date baut einen Datumswert.
func Date(s string) Value { return Value{Date: &s} }

type valueJSON struct {
	Cents  *domain.Cents `json:"cent,omitempty"`
	Amount *string       `json:"betrag,omitempty"`
	Text   *string       `json:"text,omitempty"`
	Bool   *bool         `json:"ja,omitempty"`
	Date   *string       `json:"datum,omitempty"`
}

// MarshalJSON schreibt Beträge zusätzlich als Zeichenkette "1234.56": ein
// Leser ohne Ganzzahl-Cent soll sie ohne Gleitkomma übernehmen können.
func (v Value) MarshalJSON() ([]byte, error) {
	out := valueJSON{Cents: v.Cents, Text: v.Text, Bool: v.Bool, Date: v.Date}
	if v.Cents != nil {
		s := v.Cents.Decimal()
		out.Amount = &s
	}
	return json.Marshal(out)
}

// UnmarshalJSON liest den Wert; maßgeblich ist der Betrag in Cent.
func (v *Value) UnmarshalJSON(data []byte) error {
	var in valueJSON
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	*v = Value{Cents: in.Cents, Text: in.Text, Bool: in.Bool, Date: in.Date}
	if v.Cents == nil && in.Amount != nil {
		c, err := domain.ParseCents(*in.Amount)
		if err != nil {
			return err
		}
		v.Cents = &c
	}
	return nil
}

// OriginKind ist die Art der Herkunft eines Werts.
type OriginKind string

const (
	OriginAccounts    OriginKind = "konten"
	OriginEntries     OriginKind = "buchungen"
	OriginComputation OriginKind = "rechnung"
	OriginSettings    OriginKind = "stammdaten"
	OriginInput       OriginKind = "eingabe"
	OriginVatReturns  OriginKind = "voranmeldungen"
	OriginAssets      OriginKind = "anlagenkartei"
	OriginStatement   OriginKind = "abschluss"
	OriginNone        OriginKind = "keine"
)

// Origin sagt, woher ein Wert stammt.
type Origin struct {
	Kind      OriginKind    `json:"art"`
	Accounts  []AccountPart `json:"konten,omitempty"`
	EntryIDs  []uint        `json:"buchungen,omitempty"`
	FromLines []string      `json:"aus_zeilen,omitempty"`
	Formula   string        `json:"rechnung,omitempty"`
	InputKey  string        `json:"eingabe,omitempty"`
	Text      string        `json:"erlaeuterung,omitempty"`
}

// AccountPart ist der Beitrag eines Kontos zu einer Zeile: Saldo mal Anteil.
type AccountPart struct {
	Account string       `json:"konto"`
	Name    string       `json:"bezeichnung"`
	Balance domain.Cents `json:"saldo"`
	Num     int64        `json:"anteil_zaehler"`
	Den     int64        `json:"anteil_nenner"`
	Amount  domain.Cents `json:"beitrag"`
}

// InputSource ist der Beleg einer Eingabe.
type InputSource string

const (
	InputFromNotice     InputSource = "bescheid"
	InputFromResolution InputSource = "beschluss"
	InputFromReceipt    InputSource = "beleg"
	InputFromPerson     InputSource = "mensch"
	InputFromBooks      InputSource = "buecher"
)

// Valid meldet, ob die Quelle eine der bekannten ist.
func (s InputSource) Valid() bool {
	switch s {
	case InputFromNotice, InputFromResolution, InputFromReceipt, InputFromPerson, InputFromBooks:
		return true
	}
	return false
}

// Input ist eine Angabe, die nicht in den Büchern steht: Verlustvortrag,
// Einlagekonto des Vorjahres, Gewerbeverlust, Beteiligungsquote am 01.01.
type Input struct {
	Key       string      `json:"schluessel"`
	Value     Value       `json:"wert"`
	Source    InputSource `json:"quelle"`
	ReceiptID *uint       `json:"beleg,omitempty"`
	Note      string      `json:"notiz,omitempty"`
	// Channel ist der Erfassungsweg (ui, agent:<kurzname>).
	Channel   string    `json:"erfassungsweg"`
	EnteredBy string    `json:"erfasst_von"`
	EnteredAt time.Time `json:"erfasst_am"`
	// Confirmed: ein Mensch hat den Wert in der App gesetzt oder bestätigt,
	// oder ein Agent hat einen Wert gesetzt, der ihm mit Rückfrage zusteht.
	Confirmed   bool       `json:"bestaetigt"`
	ConfirmedBy string     `json:"bestaetigt_von,omitempty"`
	ConfirmedAt *time.Time `json:"bestaetigt_am,omitempty"`
}

// Severity ist die Schwere eines Befunds.
type Severity string

const (
	// SeverityBlocking: die Erklärung ist so nicht abgabereif.
	SeverityBlocking Severity = "blockiert"
	SeverityWarning  Severity = "warnung"
	SeverityHint     Severity = "hinweis"
)

// Finding ist ein Befund oder eine Plausibilitätsprüfung.
type Finding struct {
	Severity Severity `json:"schwere"`
	Code     string   `json:"code"`
	LineKey  string   `json:"zeile,omitempty"`
	Text     string   `json:"text"`
	Legal    string   `json:"rechtsgrundlage,omitempty"`
}

// Status ist der Stand einer Erklärung.
type Status string

const (
	StatusDraft     Status = "entwurf"
	StatusExported  Status = "exportiert"
	StatusSubmitted Status = "uebermittelt"
)

// Basis hält fest, auf welchem Stand gerechnet wurde.
type Basis struct {
	// ClosingState ist der Stand des Geschäftsjahres (offen, aufgestellt, ...).
	ClosingState   string    `json:"abschlussstand"`
	JournalHash    string    `json:"journal_hash"`
	JournalEntries int       `json:"buchungen"`
	ProgramVersion string    `json:"programmversion"`
	CatalogVersion string    `json:"zeilenkatalog"`
	ComputedAt     time.Time `json:"gerechnet_am"`
}

// Submission ist die Übermittlung in Mein ELSTER, erfasst in der App.
type Submission struct {
	At             string `json:"am"`
	TransferTicket string `json:"transferticket"`
	FiguresHash    string `json:"zahlen_hash"`
	Actor          string `json:"bearbeiter"`
	OverrideReason string `json:"begruendung,omitempty"`
}

// Ref ist eine Quelle.
type Ref struct {
	ID    string `json:"id"`
	Title string `json:"titel"`
	URL   string `json:"url,omitempty"`
	Date  string `json:"stand,omitempty"`
}

// ReconciliationRow vergleicht eine Kennzahl der Jahreserklärung mit der Summe
// der übermittelten Voranmeldungen.
type ReconciliationRow struct {
	Code          string       `json:"kz_voranmeldung"`
	Label         string       `json:"bezeichnung"`
	Field         string       `json:"feld"` // bemessungsgrundlage oder steuer
	Year          domain.Cents `json:"jahr"`
	Submitted     domain.Cents `json:"voranmeldungen"`
	Difference    domain.Cents `json:"differenz"`
	Explanation   string       `json:"grund,omitempty"`
	UnreportedIDs []uint       `json:"nicht_gemeldete_buchungen,omitempty"`
}

// Blocking meldet, ob ein Befund die Erklärung sperrt.
func (d *Declaration) Blocking() []Finding {
	out := []Finding{}
	for _, f := range d.Findings {
		if f.Severity == SeverityBlocking {
			out = append(out, f)
		}
	}
	return out
}

// Line sucht eine Zeile über alle Vordrucke.
func (d *Declaration) Line(key string) (*Line, bool) {
	for i := range d.Forms {
		for j := range d.Forms[i].Sections {
			for k := range d.Forms[i].Sections[j].Lines {
				if d.Forms[i].Sections[j].Lines[k].Key == key {
					return &d.Forms[i].Sections[j].Lines[k], true
				}
			}
		}
	}
	return nil, false
}

// Amount ist der Betrag einer Zeile, null ohne Zeile oder Betrag.
func (d *Declaration) Amount(key string) domain.Cents {
	if l, ok := d.Line(key); ok {
		return l.Value.Amount()
	}
	return 0
}
