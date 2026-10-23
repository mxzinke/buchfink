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

	Name    string `gorm:"type:text;not null" json:"name"`
	Company string `gorm:"size:255" json:"company"`
	Email   string `gorm:"size:255" json:"email"`
	// Street, PostalCode und City sind die Anschrift in ihren Bestandteilen.
	//
	// § 14 Abs. 4 Nr. 1 UStG verlangt die vollständige Anschrift des
	// Leistungsempfängers, und EN 16931 verlangt sie in Feldern (BT-50, BT-52,
	// BT-53). Ein Freitextfeld ließe sich nicht prüfen — ob eine PLZ darin
	// vorkommt, ist keine Prüfung.
	Street           string `gorm:"size:255" json:"street"`
	PostalCode       string `gorm:"size:20" json:"postalCode"`
	City             string `gorm:"size:120" json:"city"`
	TaxID            string `gorm:"size:50" json:"taxId"` // Steuernummer
	VatID            string `gorm:"size:50" json:"vatId"` // USt-IdNr.
	CountryCode      string `gorm:"size:2;default:'DE'" json:"countryCode"`
	IBAN             string `gorm:"size:34" json:"iban"`
	BIC              string `gorm:"size:11" json:"bic"`
	PaymentTermsDays int    `gorm:"default:14" json:"paymentTermsDays"`

	// EInvoiceProfile ist das Zielformat, in dem dieser Empfänger seine
	// Rechnungen bekommt. Es ist eine Eigenschaft des Empfängers und keine
	// Entscheidung je Rechnung: eine Behörde nimmt XRechnung und sonst nichts,
	// und wer das bei jeder Rechnung neu wählt, wählt irgendwann falsch.
	EInvoiceProfile EInvoiceProfile `gorm:"size:30;not null;default:'zugferd_en16931'" json:"eInvoiceProfile"`
	// LeitwegID ist die Route-ID des öffentlichen Auftraggebers (BT-10). Bei
	// XRechnung ist sie Pflicht (BR-DE-15); ohne sie findet die Rechnung ihren
	// Empfänger in der Verwaltung nicht.
	LeitwegID string `gorm:"size:60" json:"leitwegId"`

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
	ExemptionCertificateNumber string `gorm:"size:60" json:"exemptionCertificateNumber,omitempty"`
	// ExemptionCertificateValidUntil ist der letzte Tag der Gültigkeit.
	ExemptionCertificateValidUntil string `gorm:"size:10;index" json:"exemptionCertificateValidUntil,omitempty"`

	// Die Sperre nach einem Löschverlangen (Art. 17 DSGVO).
	//
	// Ein Geschäftspartner, der die Löschung seiner Daten verlangt, kann sie
	// nicht bekommen, solange die Buchungen aufzubewahren sind: Art. 17 Abs. 3
	// Buchst. b DSGVO nimmt die Verarbeitung aus, die zur Erfüllung einer
	// rechtlichen Verpflichtung erforderlich ist, und § 257 HGB und § 147 AO
	// sind solche. Was geht, ist die Sperre: der Kontakt steht in keiner
	// Auswahl mehr, bleibt aber in Buchungen und Exporten sichtbar — ihn dort
	// zu entfernen hieße, die Buchführung unvollständig zu machen.
	Blocked       bool   `gorm:"not null;default:false;index" json:"blocked"`
	BlockedAt     string `gorm:"size:10" json:"blockedAt,omitempty"`
	BlockedReason string `gorm:"size:500" json:"blockedReason,omitempty"`

	// Die Standard-Buchungsgruppe: die Gruppe, mit der die Eingangsrechnungen
	// dieses Lieferanten in der Regel gebucht werden. DefaultPostingGroup ist
	// immer eine vom Menschen in der App bestätigte Gruppe; erst mit ihr darf
	// ein verbundener Agent E-Rechnungen des Lieferanten ohne Rückfrage
	// buchen. Ein Agent schlägt eine Gruppe nur vor (ProposedPostingGroup).
	// SaveContact ändert keines dieser Felder; dafür gibt es
	// ContactService.ConfirmDefaultPostingGroup und ProposeDefaultPostingGroup.
	DefaultPostingGroup            string `gorm:"size:50" json:"defaultPostingGroup,omitempty"`
	DefaultPostingGroupConfirmedAt string `gorm:"size:25" json:"defaultPostingGroupConfirmedAt,omitempty"`
	DefaultPostingGroupConfirmedBy string `gorm:"size:120" json:"defaultPostingGroupConfirmedBy,omitempty"`
	ProposedPostingGroup           string `gorm:"size:50" json:"proposedPostingGroup,omitempty"`
	ProposedPostingGroupNote       string `gorm:"size:500" json:"proposedPostingGroupNote,omitempty"`
	ProposedPostingGroupBy         string `gorm:"size:60" json:"proposedPostingGroupBy,omitempty"`

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

	// Partner-specific default accounts and SEPA direct debit mandates are not stored.
}

// IsBusiness reports whether the partner is an Unternehmer.
func (c *Contact) IsBusiness() bool { return !c.IsPrivate }

// IsConsumer meldet den Verbraucher des § 13 BGB.
//
// Derselbe Wert wie IsPrivate, unter dem Namen, unter dem die Verzugsrechnung
// ihn braucht: Verzugszinssatz und Pauschale des § 288 BGB richten sich danach, ob
// ein Verbraucher beteiligt ist. Kein zweites Feld am Kontakt — ein
// Geschäftspartner, der kein Unternehmer ist, ist der Verbraucher, und zwei
// Felder für eine Eigenschaft geraten irgendwann auseinander.
func (c *Contact) IsConsumer() bool { return c.IsPrivate }

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

// PostalAddress liefert die Anschrift in ihren Bestandteilen.
func (c *Contact) PostalAddress() (street, postalCode, city string) {
	return c.Street, c.PostalCode, c.City
}

// HasCompleteAddress reports whether street, postal code and city are all
// known — the three parts § 14 Abs. 4 Nr. 1 UStG asks for.
func (c *Contact) HasCompleteAddress() bool {
	return c.Street != "" && c.PostalCode != "" && c.City != ""
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

// LegalName is the invoice recipient; Name may be the company's contact person.
func (c *Contact) LegalName() string {
	if company := strings.TrimSpace(c.Company); company != "" {
		return company
	}
	return strings.TrimSpace(c.Name)
}
