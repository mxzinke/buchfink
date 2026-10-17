package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SettingItem represents a key-value configuration item in the SQLite database.
type SettingItem struct {
	Key       string    `gorm:"primaryKey;size:100" json:"key"`
	Value     string    `gorm:"type:text;not null" json:"value"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CompanySettings holds metadata for the business and fiscal year.
type CompanySettings struct {
	ManagingDirectors     string `json:"managingDirectors"`
	SupervisoryBoardChair string `json:"supervisoryBoardChair"`
	SellerIdentifier      string `json:"sellerIdentifier"`
	CompanyName           string `json:"companyName"`
	LegalForm             string `json:"legalForm"`            // LegalFormGmbH oder LegalFormUG
	FiscalYear            int    `json:"fiscalYear"`           // Active fiscal year (e.g. 2026)
	FiscalYearStartMonth  int    `json:"fiscalYearStartMonth"` // 1 = Jan (Kalenderjahr), 7 = Jul (abweichendes Geschäftsjahr), etc.
	TaxNumber             string `json:"taxNumber"`            // Steuernummer
	VatID                 string `json:"vatId"`                // USt-IdNr.
	TaxOffice             string `json:"taxOffice"`            // Finanzamt

	// FederalState ist das Bundesland am Sitz als Kürzel („BY"). Es bestimmt
	// die Feiertage für das Fristende (§ 108 Abs. 3 AO) und die
	// ELSTER-Steuernummer.
	FederalState string `json:"federalState"`
	// TaxNumberElster ist die Steuernummer im 13-stelligen Bundesschema,
	// abgeleitet aus TaxNumber und FederalState und nicht gespeichert.
	TaxNumberElster string `json:"taxNumberElster"`
	// TaxAdvisor sagt, ob und für welche Erklärungen ein Steuerberater
	// beauftragt ist; daran hängt die Abgabefrist (§ 149 Abs. 3 AO).
	TaxAdvisor TaxAdvisor `json:"taxAdvisor"`
	// NonProfit, InLiquidation und FiscalUnity sind Merkmale, für die Buchfink
	// weder Abschluss noch Steuerrückstellung rechnet (CompanySettings.Unsupported).
	NonProfit     bool `json:"nonProfit"`
	InLiquidation bool `json:"inLiquidation"`
	FiscalUnity   bool `json:"fiscalUnity"`

	// Die Geschäftsanschrift in ihren Bestandteilen, wie EN 16931 sie verlangt
	// (BT-35 bis BT-40). Street trägt die Straße oder ein Postfach,
	// HouseNumber die Hausnummer, AddressAddition einen Zusatz wie „c/o".
	Street          string `json:"street"`
	HouseNumber     string `json:"houseNumber"`
	AddressAddition string `json:"addressAddition"`
	PostalCode      string `json:"postalCode"`
	City            string `json:"city"`
	CountryCode     string `json:"countryCode"` // ISO 3166-1 Alpha-2

	// FoundedOn ist der Tag der Gründung (JJJJ-MM-TT): die notarielle
	// Beurkundung des Gesellschaftsvertrags.
	FoundedOn string `json:"foundedOn"`
	// Notary und DeedNumber bezeichnen die Urkunde des Gesellschaftsvertrags.
	Notary     string `json:"notary"`
	DeedNumber string `json:"deedNumber"` // Urkundenrollennummer
	// ShareCapital und Shareholders sind der aktuelle Stand von Stammkapital
	// und Gesellschafterliste. Den Stand bei Beurkundung, mit den
	// geleisteten Einlagen, hält die Gründung (Foundation).
	ShareCapital Cents                `json:"shareCapital"`
	Shareholders []CompanyShareholder `json:"shareholders"`

	// IBAN, BIC und BankName sind die Zahlungsverbindung auf Rechnungen. Sie
	// werden beim Lesen aus dem Bankkonto gesetzt, das als Konto für Rechnungen
	// festgelegt ist (BankAccount.IsInvoiceAccount), und nicht mit den
	// Unternehmensdaten gespeichert.
	IBAN     string `json:"iban"`
	BIC      string `json:"bic"`
	BankName string `json:"bankName"`

	// Ansprechpartner, Telefon und E-Mail des Ausstellers.
	//
	// Sie sind bei einer XRechnung Pflicht (BR-DE-2 bis BR-DE-7): eine Behörde,
	// die zu einer Rechnung nicht zurückfragen kann, weist sie zurück. Auf einer
	// ZUGFeRD-Rechnung bleiben sie ohne Wirkung, wenn sie nicht gepflegt sind.
	ContactName  string `json:"contactName"`
	ContactPhone string `json:"contactPhone"`
	ContactEmail string `json:"contactEmail"`

	// InvoiceNumberFormat ist die Systematik des Rechnungsnummernkreises mit
	// den Platzhaltern {JAHR} und {NR:n}. Leer heißt: die Voreinstellung.
	InvoiceNumberFormat string `json:"invoiceNumberFormat"`
	// ReceiptNumberFormat ist dieselbe Systematik für den Belegnummernkreis
	// (BEL-02 K4). Leer heißt: die Voreinstellung „ER-{JAHR}-{NR:4}".
	ReceiptNumberFormat string `json:"receiptNumberFormat"`
	// Seat, RegisterCourt und RegisterNumber sind die Pflichtangaben des
	// § 264 Abs. 1a HGB: auf jedem Jahresabschluss sind Firma, Sitz,
	// Registergericht und Registernummer anzugeben. Sie standen bisher nur an
	// der Gründung (domain.Foundation) und fehlten damit jedem Mandanten, der
	// nicht über den Gründungsweg entstanden ist.
	Seat           string `json:"seat"`
	RegisterCourt  string `json:"registerCourt"`
	RegisterNumber string `json:"registerNumber"`
	RegisteredOn   string `json:"registeredOn"`
	Currency       string `json:"currency"`
	SKR            string `json:"skr"`          // "SKR04"
	VatPeriod      string `json:"vatPeriod"`    // "month", "quarter", "year", "unknown", "none"
	TaxationType   string `json:"taxationType"` // "IST", "SOLL"

	// PermanentExtension ist die Dauerfristverlängerung nach §§ 46 bis 48 UStDV:
	// jede Voranmeldung wird einen Monat später fällig — bei monatlicher wie bei
	// vierteljährlicher Abgabe.
	PermanentExtension bool `json:"permanentExtension"`
	// SpecialPrepayment ist die Sondervorauszahlung von einem Elftel der
	// Vorauszahlungen des Vorjahres (§ 47 Abs. 1 UStDV). Sie wird in der letzten
	// Voranmeldung des Jahres angerechnet (Kennziffer 39, § 48 Abs. 4 UStDV).
	//
	// Sie gilt nur bei monatlichem Voranmeldungszeitraum: § 47 Abs. 1 UStDV
	// verlangt sie von den Unternehmern, die ihre Voranmeldungen monatlich
	// abzugeben haben. Wer vierteljährlich voranmeldet, erhält die
	// Dauerfristverlängerung nach § 46 UStDV ohne Sondervorauszahlung — für ihn
	// bleibt dieses Feld ohne Wirkung, auch wenn ein Betrag darin steht.
	//
	// Sie steht hier als erfasster Betrag und nicht als Rechnung: maßgeblich ist,
	// was angemeldet und gezahlt wurde, nicht was Buchfink daraus errechnet.
	SpecialPrepayment Cents `json:"specialPrepayment"`
	// ReceiptCaptureDays ist die Frist, nach der ein abgelegter, aber nicht
	// gebuchter Beleg im Prüflauf auffällt. GoBD Rz. 47 nennt zehn Tage für die
	// Erfassung unbarer Geschäftsvorfälle.
	ReceiptCaptureDays int `json:"receiptCaptureDays"`
	// CommitGraceDays ist die Nachfrist, nach der ein nicht festgeschriebener
	// Vormonat auf der Aufgabenliste erscheint. Null heißt: bis zum Ende des
	// Folgemonats.
	CommitGraceDays int `json:"commitGraceDays"`
	// InvoiceCheckThreshold ist der Betrag, ab dem ein Eingangsbeleg einen
	// Leistungsnachweis haben muss (RECH-08).
	//
	// Eine Grenze und keine Pflicht für jeden Beleg: den Kaffeekassenbon gegen
	// eine Bestellung zu prüfen, kostet mehr, als er wert ist. Oberhalb der
	// Grenze ist der Vermerk das, was die sachliche Richtigkeit belegt — ohne
	// ihn steht später nur die Rechnung da und niemand, der sie geprüft hat.
	//
	// Null heißt: die Voreinstellung von 1.000 Euro — nicht „kein Nachweis
	// verlangt". Ein Formular, das dieses Feld nicht kennt, schickt eine Null,
	// und eine Null, die die Prüfregel stumm abschaltete, wäre ein
	// Kontrollsystem, das sich durch Unterlassen ausschalten lässt. Wer keinen
	// Nachweis will, setzt die Grenze hoch.
	InvoiceCheckThreshold Cents `json:"invoiceCheckThreshold"`
	// InvoiceCheckSince ist der Tag, ab dem der Prüflauf den fehlenden
	// Leistungsnachweis beanstandet (JJJJ-MM-TT).
	//
	// Eine Festlegung des internen Kontrollsystems gilt ab dem Tag, an dem sie
	// getroffen wurde. Ohne diesen Tag meldete der erste Abschluss nach dem
	// Setzen der Grenze jeden Altbeleg, an dem der Vermerk fehlt — für Belege
	// aus einer Zeit, in der niemand ihn verlangt hat, und regelmäßig aus
	// festgeschriebenen Zeiträumen. Ein Prüfbericht mit hundert Befunden, die
	// sich nicht abarbeiten lassen, ist keiner.
	//
	// Gesetzt wird er selbsttätig, wenn die Grenze zum ersten Mal gespeichert
	// wird; überschreiben lässt er sich, wer die Regel rückwirkend anwenden
	// will, trägt ein früheres Datum ein. Leer heißt: keine Grenze nach hinten.
	InvoiceCheckSince string `json:"invoiceCheckSince,omitempty"`
	// InvestorOverride legt die Anlegerstellung für § 20 InvStG ausdrücklich
	// fest — und ist normalerweise leer.
	//
	// Der Regelfall folgt aus der Rechtsform: eine GmbH unterliegt dem KStG.
	// Die Ausnahmen des § 20 Abs. 1 Sätze 4 und 5 entscheidet sie nicht — für
	// Lebens- und Krankenversicherer, für Kreditinstitute mit Handelsbestand
	// und für Pensionsfonds gelten die erhöhten Sätze nicht, obwohl es
	// Körperschaften sind.
	//
	// Gelesen wird es über InvestorTypeOrDerived, nie direkt: sonst stünde an
	// jeder Aufrufstelle noch einmal, was die Ableitung ist.
	InvestorOverride InvestorType `json:"investorOverride"`

	// InGruendung sagt, dass die Gesellschaft beurkundet, aber noch nicht
	// eingetragen ist — sie ist Vorgesellschaft.
	//
	// Abgeleitet und nicht gespeichert: das Feld wird beim Lesen der
	// Unternehmensdaten aus der Gründung gesetzt (repository/settings_gorm.go).
	// Ein Kennzeichen, das man unabhängig vom Eintragungsdatum setzen kann, geht
	// irgendwann mit ihm auseinander — derselbe Grund, aus dem
	// Foundation.Stage() abgeleitet ist.
	//
	// Gelesen wird es über FirmName, nie direkt.
	InGruendung bool `gorm:"-" json:"inGruendung"`
}

// FirmName ist der Name, unter dem das Unternehmen nach außen auftritt.
//
// Bis zur Eintragung ins Handelsregister ist das der Name mit dem Zusatz
// „i. G.": Die Vorgesellschaft ist noch keine juristische Person, die
// Haftungsbeschränkung greift noch nicht, und wer mit ihr abschließt, soll das
// am Namen erkennen. Mit der Eintragung fällt der Zusatz weg — von selbst, weil
// er nirgends gespeichert ist.
//
// Jede Ausgabe nach außen liest den Namen hierüber: Rechnung, E-Rechnung,
// E-Bilanz, Abschlusskopf, Eigenbeleg. Roh bleibt CompanyName
// dort, wo er Eingabe oder Schlüssel ist — im Einstellungsfeld, im
// Mandantennamen und im Änderungsprotokoll.
func (s *CompanySettings) FirmName() string {
	name := strings.TrimSpace(s.CompanyName)
	if name == "" || !s.InGruendung {
		return name
	}
	// Mit schmalem Leerraum wäre der Zusatz typografisch schöner und in einer
	// Registerabfrage oder einem Rechnungsdatensatz nicht mehr zu finden.
	return name + " i. G."
}

// InvestorType ist die Anlegerstellung, an der § 20 InvStG die Höhe der
// Teilfreistellung festmacht.
type InvestorType string

const (
	// InvestorUnknown heißt: nicht festgelegt. Buchfink rechnet dann keine
	// Teilfreistellung, sondern sagt, was zu entscheiden ist.
	InvestorUnknown InvestorType = ""
	// InvestorBasic sind die Grundsätze des § 20 Abs. 1 Satz 1 und Abs. 2
	// InvStG: 30 % bei Aktienfonds, die Hälfte bei Mischfonds — die Fälle, in
	// denen § 20 Abs. 1 Sätze 4 und 5 die erhöhten Sätze zurücknehmen.
	InvestorBasic InvestorType = "basic"
	// InvestorCorporate ist der Anleger, der dem Körperschaftsteuergesetz
	// unterliegt (§ 20 Abs. 1 Satz 3): 80 % bzw. 40 %.
	InvestorCorporate InvestorType = "corporate"
)

// Label renders the investor type for the UI.
func (t InvestorType) Label() string {
	switch t {
	case InvestorBasic:
		return "Grundsatz (Ausnahme nach § 20 Abs. 1 Sätze 4 und 5 InvStG)"
	case InvestorCorporate:
		return "Anleger unterliegt dem Körperschaftsteuergesetz"
	default:
		return "Nicht festgelegt"
	}
}

// Valid reports whether the investor type is one of the known ones.
func (t InvestorType) Valid() bool {
	switch t {
	case InvestorBasic, InvestorCorporate:
		return true
	default:
		return false
	}
}

// AllInvestorTypes returns the choices in the order the mask offers them.
func AllInvestorTypes() []InvestorType {
	return []InvestorType{InvestorCorporate, InvestorBasic}
}

// GetFiscalYearForDate computes the fiscal year for a given date (YYYY-MM-DD)
// taking into account the configured starting month (1 = Jan, 7 = Jul, etc.).
func GetFiscalYearForDate(dateStr string, startMonth int) int {
	if startMonth <= 0 || startMonth > 12 {
		startMonth = 1
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		t = time.Now()
	}
	year := t.Year()
	month := int(t.Month())

	if startMonth == 1 {
		return year
	}

	// Deviating fiscal year (e.g., starts in July):
	// Dates Jan..Jun belong to the fiscal year that started in (year - 1).
	// Dates Jul..Dec belong to the fiscal year starting in (year).
	if month < startMonth {
		return year - 1
	}
	return year
}

// FiscalYearFirstDay ist der erste Tag des Geschäftsjahres year. Ein
// abweichendes Geschäftsjahr beginnt im Kalenderjahr mit derselben Jahreszahl.
func FiscalYearFirstDay(year, startMonth int) string {
	if startMonth <= 0 || startMonth > 12 {
		startMonth = 1
	}
	return fmt.Sprintf("%04d-%02d-01", year, startMonth)
}

// SettingsRepository defines database operations for application configuration.
type SettingsRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string) error
	GetCompanySettings(ctx context.Context) (*CompanySettings, error)
	UpdateCompanySettings(ctx context.Context, settings *CompanySettings) error
}

// BusinessLetterDetails is printed on outgoing business correspondence.
func (s *CompanySettings) BusinessLetterDetails() string {
	parts := []string{s.FirmName()}
	if s.Seat != "" {
		parts = append(parts, "Sitz: "+s.Seat)
	}
	if s.RegisterCourt != "" || s.RegisterNumber != "" {
		parts = append(parts, strings.TrimSpace(s.RegisterCourt+" "+s.RegisterNumber))
	}
	if s.ManagingDirectors != "" {
		parts = append(parts, "Geschäftsführung: "+s.ManagingDirectors)
	}
	if s.SupervisoryBoardChair != "" {
		parts = append(parts, "Aufsichtsratsvorsitz: "+s.SupervisoryBoardChair)
	}
	return strings.Join(parts, " · ")
}

// InvoiceSellerFindings lists missing issuer data before invoice issuance.
func (s *CompanySettings) InvoiceSellerFindings(structured bool) []string {
	missing := []string{}
	if strings.TrimSpace(s.CompanyName) == "" {
		missing = append(missing, "Firmenname")
	}
	if !s.HasCompleteAddress() {
		missing = append(missing, "vollständige Geschäftsanschrift")
	}
	if IsSupportedLegalForm(s.LegalForm) {
		if strings.TrimSpace(s.ManagingDirectors) == "" {
			missing = append(missing, "alle Geschäftsführer mit ausgeschriebenem Vor- und Nachnamen")
		}
		if strings.TrimSpace(s.Seat) == "" {
			missing = append(missing, "Sitz der Gesellschaft")
		}
		if !s.InGruendung {
			if strings.TrimSpace(s.RegisterCourt) == "" {
				missing = append(missing, "Registergericht")
			}
			if strings.TrimSpace(s.RegisterNumber) == "" {
				missing = append(missing, "Registernummer")
			}
		}
	}
	if structured && strings.TrimSpace(s.VatID) == "" && strings.TrimSpace(s.RegisterNumber) == "" && strings.TrimSpace(s.SellerIdentifier) == "" {
		missing = append(missing, "Verkäuferkennung für die E-Rechnung (alternativ USt-ID oder Registernummer)")
	}
	return missing
}
