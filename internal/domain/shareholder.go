package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Konten für Vorgänge zwischen der GmbH und ihren Gesellschaftern
// (SKR04-Katalog internal/accounting/skr04_2026.json).
const (
	AccountAusleihungenGesellschafter      = "0960" // Ausleihungen an Gesellschafter
	AccountForderungenGesellschafter       = "1307" // Forderungen gegen GmbH-Gesellschafter
	AccountForderungenGesellschafterBis1   = "1308" // dto. Restlaufzeit bis 1 Jahr
	AccountForderungenGesellschafterUeber1 = "1309" // dto. Restlaufzeit größer 1 Jahr
	AccountVerbindlichkeitenGesellschafter = "3640" // Verbindlichkeiten gegenüber GmbH-Gesellschaftern
	AccountVerbGesellschafterBis1          = "3641" // dto. Restlaufzeit bis 1 Jahr
	AccountVerbGesellschafter1Bis5         = "3642" // dto. Restlaufzeit 1 bis 5 Jahre
	AccountVerbGesellschafterUeber5        = "3643" // dto. Restlaufzeit größer 5 Jahre
	AccountSonstigeVerbindlichkeiten       = "3500" // Sonstige Verbindlichkeiten
	AccountZinsenGesellschafterdarlehen    = "7316" // Zinsen für Gesellschafterdarlehen
	AccountZinsenGesellschafterUeber25     = "7317" // Zinsen an Gesellschafter mit mehr als 25 % Beteiligung
	AccountSonstigeZinsertraege            = "7110" // Sonstige Zinserträge
	AccountKreditkartenabrechnung          = "3610" // Kreditkartenabrechnung
)

// LoanDirection sagt, wer wem Geld leiht.
type LoanDirection string

const (
	// LoanToCompany: der Gesellschafter leiht der GmbH.
	LoanToCompany LoanDirection = "to_company"
	// LoanToShareholder: die GmbH leiht dem Gesellschafter.
	LoanToShareholder LoanDirection = "to_shareholder"
)

// Valid meldet, ob die Richtung bekannt ist.
func (d LoanDirection) Valid() bool { return d == LoanToCompany || d == LoanToShareholder }

// DayCount ist die vereinbarte Zinsmethode.
type DayCount string

const (
	// DayCountAct365 zählt die tatsächlichen Tage über ein Jahr von 365 Tagen.
	DayCountAct365 DayCount = "act/365"
	// DayCountAct360 zählt die tatsächlichen Tage über ein Jahr von 360 Tagen.
	DayCountAct360 DayCount = "act/360"
	// DayCount30E360 ist die deutsche Zinsmethode: jeder Monat 30 Tage, das
	// Jahr 360 Tage.
	DayCount30E360 DayCount = "30/360"
)

// Valid meldet, ob die Zinsmethode bekannt ist.
func (d DayCount) Valid() bool {
	return d == DayCountAct365 || d == DayCountAct360 || d == DayCount30E360
}

// ShareholderLoan ist ein Darlehensvertrag zwischen der GmbH und einem
// Gesellschafter. Der Bestand steht im Journal: Auszahlung und Tilgung sind
// Buchungen auf Account mit dem Gesellschafter als Kontakt.
type ShareholderLoan struct {
	ID        uint          `gorm:"primaryKey" json:"id"`
	ContactID uint          `gorm:"not null;index" json:"contactId"`
	Direction LoanDirection `gorm:"size:20;not null" json:"direction"`
	Principal Cents         `gorm:"not null" json:"principal"`
	// InterestRateBP ist der Zinssatz in Hundertstel Prozent: 350 = 3,50 %.
	InterestRateBP int      `gorm:"not null" json:"interestRateBp"`
	DayCount       DayCount `gorm:"size:10;not null" json:"dayCount"`
	ContractDate   string   `gorm:"size:10" json:"contractDate"`
	StartDate      string   `gorm:"size:10;not null" json:"startDate"`
	// MaturityDate leer heißt: ohne bestimmte Laufzeit (§ 488 Abs. 3 BGB).
	MaturityDate    string `gorm:"size:10" json:"maturityDate,omitempty"`
	Account         string `gorm:"size:10;not null" json:"account"`
	InterestAccount string `gorm:"size:10;not null" json:"interestAccount"`
	// ContractReceiptID ist der abgelegte Darlehensvertrag.
	ContractReceiptID *uint  `json:"contractReceiptId,omitempty"`
	Collateral        string `gorm:"type:text" json:"collateral,omitempty"`
	// ComparisonRateBP ist der belegte Zins eines fremden Dritten für den
	// Fremdvergleich (R 8.5 KStR 2022), ComparisonSource sein Nachweis.
	ComparisonRateBP *int   `json:"comparisonRateBp,omitempty"`
	ComparisonSource string `gorm:"type:text" json:"comparisonSource,omitempty"`
	// Subordinated: Rangrücktritt vereinbart (§ 19 Abs. 2 Satz 2 InsO).
	Subordinated bool      `gorm:"not null;default:false" json:"subordinated"`
	Note         string    `gorm:"type:text" json:"note,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Validate prüft die Angaben des Vertrags.
func (l *ShareholderLoan) Validate() error {
	if l.ContactID == 0 {
		return fmt.Errorf("zum Darlehen gehört der Gesellschafter als Kontakt")
	}
	if !l.Direction.Valid() {
		return fmt.Errorf("unbekannte Darlehensrichtung %q (to_company oder to_shareholder)", l.Direction)
	}
	if l.Principal <= 0 {
		return fmt.Errorf("der Darlehensbetrag muss größer als null sein")
	}
	if l.InterestRateBP < 0 || l.InterestRateBP > 10000 {
		return fmt.Errorf("der Zinssatz von %d Hundertstel Prozent liegt außerhalb von 0 bis 100 %%", l.InterestRateBP)
	}
	if !l.DayCount.Valid() {
		return fmt.Errorf("unbekannte Zinsmethode %q (act/365, act/360 oder 30/360)", l.DayCount)
	}
	if err := CheckDate("Beginn", l.StartDate); err != nil {
		return err
	}
	for _, d := range []struct{ field, value string }{
		{"Vertragsdatum", l.ContractDate}, {"Fälligkeit", l.MaturityDate},
	} {
		if d.value == "" {
			continue
		}
		if err := CheckDate(d.field, d.value); err != nil {
			return err
		}
	}
	if l.MaturityDate != "" && l.MaturityDate <= l.StartDate {
		return fmt.Errorf("die Fälligkeit %s liegt nicht nach dem Beginn %s",
			GermanDate(l.MaturityDate), GermanDate(l.StartDate))
	}
	if l.ComparisonRateBP != nil && strings.TrimSpace(l.ComparisonSource) == "" {
		return fmt.Errorf("ein Vergleichszins braucht seinen Nachweis, etwa das Angebot einer Bank")
	}
	return nil
}

// RateLabel zeigt den Zinssatz als „3,50 %".
func RateLabel(bp int) string {
	return strings.Replace(fmt.Sprintf("%d.%02d %%", bp/100, bp%100), ".", ",", 1)
}

// ShareholderLoanInterest ist eine gebuchte Zinsperiode eines Darlehens.
type ShareholderLoanInterest struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	LoanID     uint      `gorm:"not null;index" json:"loanId"`
	PeriodFrom string    `gorm:"size:10;not null" json:"periodFrom"`
	PeriodTo   string    `gorm:"size:10;not null" json:"periodTo"`
	Amount     Cents     `gorm:"not null" json:"amount"`
	EntryID    uint      `gorm:"not null;index" json:"entryId"`
	CreatedAt  time.Time `json:"createdAt"`
}

// ShareholderLoanRepository speichert Darlehensverträge und ihre Zinsläufe.
type ShareholderLoanRepository interface {
	Save(ctx context.Context, loan *ShareholderLoan) error
	FindByID(ctx context.Context, id uint) (*ShareholderLoan, error)
	FindAll(ctx context.Context) ([]ShareholderLoan, error)
	SaveInterest(ctx context.Context, interest *ShareholderLoanInterest) error
	FindInterest(ctx context.Context, loanID uint) ([]ShareholderLoanInterest, error)
}

// Regelschlüssel der Prüfläufe für Gesellschafter und Lohn.
const (
	CheckRuleShareholderLoan     = "shareholder_loan"
	CheckRuleShareholderClearing = "shareholder_clearing"
	CheckRulePayrollAccounts     = "payroll_accounts"
)
