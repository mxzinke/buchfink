package domain

import (
	"context"
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
	CompanyName          string `json:"companyName"`
	LegalForm            string `json:"legalForm"`            // e.g. "GmbH", "UG (haftungsbeschränkt)", "Einzelunternehmen"
	FiscalYear           int    `json:"fiscalYear"`           // Active fiscal year (e.g. 2026)
	FiscalYearStartMonth int    `json:"fiscalYearStartMonth"` // 1 = Jan (Kalenderjahr), 7 = Jul (abweichendes Geschäftsjahr), etc.
	TaxNumber            string `json:"taxNumber"`            // Steuernummer
	VatID                string `json:"vatId"`                // USt-IdNr.
	TaxOffice            string `json:"taxOffice"`            // Finanzamt
	IBAN                 string `json:"iban"`
	BIC                  string `json:"bic"`
	BankName             string `json:"bankName"`
	Street               string `json:"street"`
	ZipCity              string `json:"zipCity"`
	Country              string `json:"country"`
	Currency             string `json:"currency"`
	SKR                  string `json:"skr"`          // "SKR04"
	VatPeriod            string `json:"vatPeriod"`    // "month", "quarter", "year"
	TaxationType         string `json:"taxationType"` // "IST", "SOLL"
	// InvestorType entscheidet über die Teilfreistellung nach § 20 InvStG.
	//
	// Es ist bewusst eine eigene Angabe und keine Ableitung aus der Rechtsform:
	// aus „GmbH & Co. KG" folgt nichts. Die Gesellschaft ist keine
	// Körperschaft, ihre Gesellschafter können Körperschaften sein oder
	// natürliche Personen oder beides, und § 20 Abs. 3a InvStG bestimmt den
	// Satz nach dem Gesellschafter. Auch eine Kapitalgesellschaft trägt nicht
	// immer 80 %: für Lebens- und Krankenversicherer, für Kreditinstitute mit
	// Handelsbestand und für Pensionsfonds nimmt § 20 Abs. 1 Sätze 4 und 5 die
	// erhöhten Sätze ausdrücklich zurück. Wer das aus einem Freitextfeld rät,
	// rechnet still falsch.
	InvestorType InvestorType `json:"investorType"`
}

// InvestorType ist die Anlegerstellung, an der § 20 InvStG die Höhe der
// Teilfreistellung festmacht.
type InvestorType string

const (
	// InvestorUnknown heißt: nicht festgelegt. Buchfink rechnet dann keine
	// Teilfreistellung, sondern sagt, was zu entscheiden ist.
	InvestorUnknown InvestorType = ""
	// InvestorBasic sind die Grundsätze des § 20 Abs. 1 Satz 1 und Abs. 2
	// InvStG: 30 % bei Aktienfonds, die Hälfte bei Mischfonds. Das trifft
	// Anteile im Privatvermögen und die Fälle, in denen § 20 Abs. 1 Sätze 4
	// und 5 die erhöhten Sätze zurücknehmen.
	InvestorBasic InvestorType = "basic"
	// InvestorIndividualBusiness ist die natürliche Person, die ihre Anteile im
	// Betriebsvermögen hält (§ 20 Abs. 1 Satz 2): 60 % bzw. 30 %.
	InvestorIndividualBusiness InvestorType = "individual_business"
	// InvestorCorporate ist der Anleger, der dem Körperschaftsteuergesetz
	// unterliegt (§ 20 Abs. 1 Satz 3): 80 % bzw. 40 %.
	InvestorCorporate InvestorType = "corporate"
	// InvestorMixed ist die Personengesellschaft mit Gesellschaftern
	// unterschiedlicher Besteuerung. § 20 Abs. 3a InvStG bestimmt den Satz nach
	// dem Gesellschafter; ein einziger Prozentsatz kann das nicht ausdrücken,
	// und Buchfink weist die Erträge dann ungekürzt aus.
	InvestorMixed InvestorType = "mixed"
)

// Label renders the investor type for the UI.
func (t InvestorType) Label() string {
	switch t {
	case InvestorBasic:
		return "Grundsatz (Privatvermögen oder Ausnahme nach § 20 Abs. 1 Sätze 4 und 5 InvStG)"
	case InvestorIndividualBusiness:
		return "Natürliche Person, Anteile im Betriebsvermögen"
	case InvestorCorporate:
		return "Anleger unterliegt dem Körperschaftsteuergesetz"
	case InvestorMixed:
		return "Personengesellschaft mit gemischt besteuerten Gesellschaftern"
	default:
		return "Nicht festgelegt"
	}
}

// Valid reports whether the investor type is one of the known ones.
func (t InvestorType) Valid() bool {
	switch t {
	case InvestorBasic, InvestorIndividualBusiness, InvestorCorporate, InvestorMixed:
		return true
	default:
		return false
	}
}

// AllInvestorTypes returns the choices in the order the mask offers them.
func AllInvestorTypes() []InvestorType {
	return []InvestorType{
		InvestorCorporate, InvestorIndividualBusiness, InvestorBasic, InvestorMixed,
	}
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

// SettingsRepository defines database operations for application configuration.
type SettingsRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string) error
	GetCompanySettings(ctx context.Context) (*CompanySettings, error)
	UpdateCompanySettings(ctx context.Context, settings *CompanySettings) error
}
