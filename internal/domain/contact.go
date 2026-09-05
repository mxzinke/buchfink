package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ContactType distinguishes customers (Debitoren) from vendors (Kreditoren).
type ContactType string

const (
	ContactTypeCustomer ContactType = "customer" // Debitor (Kunde)
	ContactTypeVendor   ContactType = "vendor"   // Kreditor (Lieferant)
)

// Contact is a business partner with its own Personenkonto.
//
// The collective accounts 1200 (Forderungen aus LuL) and 3300 (Verbindlichkeiten
// aus LuL) carry the balance sheet figures, but they cannot answer "who owes me
// what". Every partner therefore gets a real Personenkonto from the DATEV ranges
// (10000-69999 Debitoren, 70000-99999 Kreditoren); open items are booked on it,
// and the balance sheet aggregates them into the collective account.
type Contact struct {
	ID   uint        `gorm:"primaryKey" json:"id"`
	Type ContactType `gorm:"size:20;not null;index" json:"type"`

	// LedgerAccount is the Personenkonto, e.g. "10001" or "70023".
	LedgerAccount string `gorm:"size:10;uniqueIndex;not null" json:"ledgerAccount"`

	Name    string `gorm:"size:255;not null;index" json:"name"`
	Company string `gorm:"size:255;serializer:encrypted" json:"company"`
	Email   string `gorm:"size:255;serializer:encrypted" json:"email"`
	// Address is the unstructured address as it was captured before Welle 5b.
	// It stays as the fallback for records the parser could not split, and is
	// what MigrateAddress reads.
	Address string `gorm:"type:text;serializer:encrypted" json:"address"`
	// Street, PostalCode und City sind die Anschrift in ihren Bestandteilen.
	//
	// § 14 Abs. 4 Nr. 1 UStG verlangt die vollständige Anschrift des
	// Leistungsempfängers, und EN 16931 verlangt sie in Feldern (BT-50, BT-52,
	// BT-53): eine einzeilige Anschrift landete bisher komplett in BT-50, und
	// beim Empfänger stand die Stadt in der Straße. Ein Freitextfeld lässt sich
	// auch nicht prüfen — ob eine PLZ darin vorkommt, ist keine Prüfung.
	Street           string `gorm:"size:255;serializer:encrypted" json:"street"`
	PostalCode       string `gorm:"size:20;serializer:encrypted" json:"postalCode"`
	City             string `gorm:"size:120;serializer:encrypted" json:"city"`
	TaxID            string `gorm:"size:50;serializer:encrypted" json:"taxId"` // Steuernummer
	VatID            string `gorm:"size:50;serializer:encrypted" json:"vatId"` // USt-IdNr.
	CountryCode      string `gorm:"size:2;default:'DE'" json:"countryCode"`
	IBAN             string `gorm:"size:34;serializer:encrypted" json:"iban"`
	BIC              string `gorm:"size:11;serializer:encrypted" json:"bic"`
	PaymentTermsDays int    `gorm:"default:14" json:"paymentTermsDays"`

	// EInvoiceProfile ist das Zielformat, in dem dieser Empfänger seine
	// Rechnungen bekommt. Es ist eine Eigenschaft des Empfängers und keine
	// Entscheidung je Rechnung: eine Behörde nimmt XRechnung und sonst nichts,
	// und wer das bei jeder Rechnung neu wählt, wählt irgendwann falsch.
	EInvoiceProfile EInvoiceProfile `gorm:"size:30;not null;default:'zugferd_en16931'" json:"eInvoiceProfile"`
	// LeitwegID ist die Route-ID des öffentlichen Auftraggebers (BT-10). Bei
	// XRechnung ist sie Pflicht (BR-DE-15); ohne sie findet die Rechnung ihren
	// Empfänger in der Verwaltung nicht.
	LeitwegID string `gorm:"size:60;serializer:encrypted" json:"leitwegId"`

	// IsPrivate marks a partner who is not an Unternehmer. It decides whether the
	// e-invoice obligation of § 14 Abs. 2 Satz 2 Nr. 1 UStG can apply to a
	// document they issue.
	//
	// The field is phrased negatively on purpose. Business partners are the
	// overwhelming default, so that case has to be the zero value: a boolean with
	// a database default of true can never be set back to false through a struct,
	// and the flag would silently stick.
	//
	// It is master data rather than a guess from the VAT id or a company name — a
	// note about the input tax deduction must not hang on whether somebody filled
	// in a field.
	IsPrivate bool `gorm:"not null;default:false" json:"isPrivate"`
	// IsSmallBusiness marks a partner under § 19 UStG. They may always issue a
	// sonstige Rechnung (§ 34a UStDV), so no e-invoice is owed. This is a
	// property of the *counterparty*; Buchfink's own client is always a
	// bilanzierende Kapitalgesellschaft.
	IsSmallBusiness bool `gorm:"not null;default:false" json:"isSmallBusiness"`

	// Die Freistellungsbescheinigung nach § 48b EStG.
	//
	// Wer eine Bauleistung bezieht, hat nach § 48 EStG 15 % der Gegenleistung
	// einzubehalten und an das Finanzamt abzuführen — es sei denn, der
	// Leistende legt eine gültige Freistellungsbescheinigung vor. Der Bauabzug
	// selbst ist nicht Teil von Buchfink; die Bescheinigung wird trotzdem
	// geführt und mit ihrer Frist überwacht, weil sie am Kontakt hängt und weil
	// eine abgelaufene Bescheinigung sonst erst auffällt, wenn die Haftung schon
	// entstanden ist.
	ExemptionCertificateNumber string `gorm:"size:60;serializer:encrypted" json:"exemptionCertificateNumber,omitempty"`
	// ExemptionCertificateValidUntil ist der letzte Tag der Gültigkeit.
	ExemptionCertificateValidUntil string `gorm:"size:10;index" json:"exemptionCertificateValidUntil,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// OpenAmount is the balance of the Personenkonto, computed on read.
	OpenAmount Cents `gorm:"-" json:"openAmount"`

	// VatIDNotice ist der Hinweis, den das Speichern eines Kontakts mit einer
	// USt-IdNr. aus einem anderen Mitgliedstaat zurückgibt.
	//
	// Nicht gespeichert und nicht blockierend: er ist eine Auskunft über den
	// Stand der Bestätigung, und die ändert sich mit der Zeit — ein
	// festgeschriebener Hinweis wäre morgen falsch. Er steht am Kontakt und nicht
	// als eigener Rückgabewert, damit sich die Signatur des Speicherns nicht
	// ändert und jeder Aufrufer ihn bekommt, ohne danach zu fragen.
	VatIDNotice string `gorm:"-" json:"vatIdNotice,omitempty"`

	// TODO: Add support for partner-specific default revenue/expense accounts
	// TODO: Add support for SEPA direct debit mandates (SEPA-Lastschriftmandate)
}

// IsBusiness reports whether the partner is an Unternehmer.
func (c *Contact) IsBusiness() bool { return !c.IsPrivate }

// ExemptionCertificateExpiryWarningDays ist der Vorlauf, mit dem Buchfink auf
// eine ablaufende Freistellungsbescheinigung hinweist. Wer sie am Ablauftag
// erfährt, hat bei der nächsten Zahlung schon einbehalten müssen.
const ExemptionCertificateExpiryWarningDays = 30

// ExemptionCertificateState sagt, wie es um die Freistellungsbescheinigung
// steht: leer (keine erfasst), "valid", "expiring" oder "expired".
func (c *Contact) ExemptionCertificateState(today string) string {
	if c.ExemptionCertificateNumber == "" && c.ExemptionCertificateValidUntil == "" {
		return ""
	}
	if c.ExemptionCertificateValidUntil == "" {
		return "valid"
	}
	if c.ExemptionCertificateValidUntil < today {
		return "expired"
	}
	if t, err := time.Parse("2006-01-02", today); err == nil {
		warnUntil := t.AddDate(0, 0, ExemptionCertificateExpiryWarningDays).Format("2006-01-02")
		if c.ExemptionCertificateValidUntil <= warnUntil {
			return "expiring"
		}
	}
	return "valid"
}

// EInvoiceProfile names the format an outgoing invoice is issued in.
type EInvoiceProfile string

const (
	// EInvoiceProfileZUGFeRD ist der Regelfall: ein PDF/A-3 mit eingebettetem
	// CII-Datensatz nach EN 16931. Der Empfänger kann es lesen und sein System
	// auch.
	EInvoiceProfileZUGFeRD EInvoiceProfile = "zugferd_en16931"
	// EInvoiceProfileXRechnungCII ist die reine XML-Datei nach der deutschen
	// Ausprägung (CIUS) in der CII-Syntax — was Bund und Länder verlangen.
	EInvoiceProfileXRechnungCII EInvoiceProfile = "xrechnung_cii"
	// EInvoiceProfilePDFOnly ist die sonstige Rechnung ohne strukturierten
	// Teil. Sie ist nur innerhalb der Übergangsfrist des § 27 Abs. 38 UStG
	// zulässig; siehe accounting.EInvoiceIssueTransitionFor.
	EInvoiceProfilePDFOnly EInvoiceProfile = "pdf_only"
)

// EInvoiceProfileInfo describes a profile for the UI.
type EInvoiceProfileInfo struct {
	Profile EInvoiceProfile `json:"profile"`
	Label   string          `json:"label"`
	Hint    string          `json:"hint"`
}

// EInvoiceProfiles lists the profiles an outgoing invoice can be issued in.
//
// XRechnung in der UBL-Syntax fehlt bewusst: Buchfink hat keinen UBL-Schreiber,
// und ein Profil anzubieten, das nichts erzeugt, wäre ein Versprechen, das erst
// beim Ausstellen bricht.
func EInvoiceProfiles() []EInvoiceProfileInfo {
	return []EInvoiceProfileInfo{
		{EInvoiceProfileZUGFeRD, "ZUGFeRD (PDF mit Datensatz)",
			"PDF/A-3 mit eingebettetem Rechnungsdatensatz nach EN 16931. Der Regelfall im Geschäftsverkehr."},
		{EInvoiceProfileXRechnungCII, "XRechnung (nur XML)",
			"Reine XML-Datei nach der deutschen Ausprägung. Öffentliche Auftraggeber verlangen sie; die Leitweg-ID ist dann Pflicht."},
		{EInvoiceProfilePDFOnly, "Nur PDF (sonstige Rechnung)",
			"Ohne strukturierten Datensatz. Nur innerhalb der Übergangsfrist des § 27 Abs. 38 UStG zulässig."},
	}
}

// Label ist der Klartext für Meldungen und Oberfläche.
func (p EInvoiceProfile) Label() string {
	for _, info := range EInvoiceProfiles() {
		if info.Profile == p {
			return info.Label
		}
	}
	return string(p)
}

// Validate weist ein Zielformat zurück, das Buchfink nicht erzeugen kann.
//
// Der Wert kommt aus der Oberfläche und aus übernommenen Daten, und ein
// unbekanntes Profil bliebe sonst still: der Renderer behandelt alles, was
// nicht XRechnung oder „nur PDF" ist, als ZUGFeRD. Ein Kontakt mit
// „xrechnung_ubl" — ein Format, das die Fachplanung nennt und das es hier nicht
// gibt — bekäme also ein ZUGFeRD-PDF, ohne dass irgendwo stünde, dass er etwas
// anderes verlangt hat.
func (p EInvoiceProfile) Validate() error {
	profiles := EInvoiceProfiles()
	names := make([]string, 0, len(profiles))
	for _, info := range profiles {
		if info.Profile == p {
			return nil
		}
		names = append(names, string(info.Profile))
	}
	return fmt.Errorf("%q ist kein E-Rechnungsformat, das Buchfink ausstellen kann; möglich sind %s",
		string(p), strings.Join(names, ", "))
}

// ResolvedEInvoiceProfile is the profile to use, filling in the default for a
// contact captured before the field existed.
func (c *Contact) ResolvedEInvoiceProfile() EInvoiceProfile {
	if c.EInvoiceProfile == "" {
		return EInvoiceProfileZUGFeRD
	}
	return c.EInvoiceProfile
}

// PostalAddress liefert die Anschrift in ihren Bestandteilen, notfalls aus dem
// alten Freitextfeld gelesen.
func (c *Contact) PostalAddress() (street, postalCode, city string) {
	if c.Street != "" || c.PostalCode != "" || c.City != "" {
		return c.Street, c.PostalCode, c.City
	}
	return ParsePostalAddress(c.Address)
}

// HasCompleteAddress reports whether street, postal code and city are all
// known — the three parts § 14 Abs. 4 Nr. 1 UStG asks for.
func (c *Contact) HasCompleteAddress() bool {
	street, postalCode, city := c.PostalAddress()
	return street != "" && postalCode != "" && city != ""
}

// MigrateAddress fills the structured fields from the free-text address.
//
// It is best effort and says so: what it cannot split stays in Address, and the
// contact page shows the record as incomplete. Guessing silently would be
// worse — a wrong street on an invoice is a formal defect the recipient pays
// for with their input tax deduction.
func (c *Contact) MigrateAddress() bool {
	if c.Street != "" || c.PostalCode != "" || c.City != "" {
		return false
	}
	street, postalCode, city := ParsePostalAddress(c.Address)
	if street == "" || postalCode == "" || city == "" {
		return false
	}
	c.Street, c.PostalCode, c.City = street, postalCode, city
	return true
}

// ParsePostalAddress splits a free-text address into street, postal code and
// city.
//
// The expected shape is the German one: the last line is "PLZ Ort", everything
// before it is the street. Both a line break and a comma count as a separator,
// because both spellings sit in the existing data.
func ParsePostalAddress(address string) (street, postalCode, city string) {
	normalized := strings.ReplaceAll(address, "\n", ",")
	parts := make([]string, 0, 4)
	for _, p := range strings.Split(normalized, ",") {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	if len(parts) < 2 {
		return "", "", ""
	}
	last := parts[len(parts)-1]
	code, town, ok := splitPostalLine(last)
	if !ok {
		return "", "", ""
	}
	return strings.Join(parts[:len(parts)-1], ", "), code, town
}

// splitPostalLine reads "80331 München" into its two halves. The postal code is
// the leading run of digits; a line that does not start with digits is not a
// postal line, and pretending otherwise would put a street name into BT-53.
func splitPostalLine(line string) (postalCode, city string, ok bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", "", false
	}
	code := fields[0]
	for _, r := range code {
		if r < '0' || r > '9' {
			return "", "", false
		}
	}
	return code, strings.Join(fields[1:], " "), true
}

// CollectiveAccount returns the SKR04 Sammelkonto a partner's open items roll up
// into for the balance sheet.
func (c *Contact) CollectiveAccount() string {
	if c.Type == ContactTypeCustomer {
		return AccountForderungenLuL
	}
	return AccountVerbindlichkeitenLuL
}

// IsEUCounterparty reports whether the partner sits in another EU member state,
// which is what makes an innergemeinschaftlicher Erwerb or a tax-exempt
// intra-community supply possible.
func (c *Contact) IsEUCounterparty() bool {
	if c.CountryCode == "" || c.CountryCode == "DE" {
		return false
	}
	_, ok := euMemberStates[c.CountryCode]
	return ok
}

var euMemberStates = map[string]struct{}{
	"AT": {}, "BE": {}, "BG": {}, "CY": {}, "CZ": {}, "DK": {}, "EE": {}, "ES": {},
	"FI": {}, "FR": {}, "GR": {}, "HR": {}, "HU": {}, "IE": {}, "IT": {}, "LT": {},
	"LU": {}, "LV": {}, "MT": {}, "NL": {}, "PL": {}, "PT": {}, "RO": {}, "SE": {},
	"SI": {}, "SK": {},
}

// ContactRepository defines database operations for debtors and creditors.
type ContactRepository interface {
	FindAll(ctx context.Context) ([]Contact, error)
	FindByID(ctx context.Context, id uint) (*Contact, error)
	FindByLedgerAccount(ctx context.Context, account string) (*Contact, error)
	Save(ctx context.Context, contact *Contact) error
	Delete(ctx context.Context, id uint) error
	Count(ctx context.Context) (int64, error)
}
