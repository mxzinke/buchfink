package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// CheckSeverity ist das Gewicht eines Befundes.
//
// Zwei Stufen, und der Unterschied ist keine Geschmacksfrage: ein blockierender
// Befund verhindert die Festschreibung, ein Hinweis nicht. Eine dritte Stufe
// „Info" gäbe es nur, damit etwas in der Liste steht, was niemand liest.
type CheckSeverity string

const (
	// CheckBlocking verhindert die Festschreibung, solange er nicht behoben oder
	// mit Begründung übergangen wird.
	CheckBlocking CheckSeverity = "blocking"
	// CheckWarning ist ein Hinweis: die Festschreibung geht durch.
	CheckWarning CheckSeverity = "warning"
)

// Die Regelschlüssel der Prüfläufe. Sie stehen in der Datenbank und dürfen sich
// deshalb nicht mehr ändern.
const (
	CheckRuleEntryWithoutReceipt = "entry_without_receipt"
	CheckRuleReceiptUnbooked     = "receipt_unbooked"
	CheckRuleReceiptOverdue      = "receipt_overdue"
	CheckRuleBankUnmatched       = "bank_unmatched"
	CheckRuleInterimBalance      = "interim_balance"
	CheckRuleDuplicateReceipt    = "duplicate_receipt"
	CheckRuleDuplicatePayment    = "duplicate_payment"
	CheckRuleNumberGap           = "number_gap"
	CheckRuleAccountUnmapped     = "account_unmapped"
	CheckRuleDepreciationMissing = "depreciation_missing"
	CheckRuleVatReturnMissing    = "vat_return_missing"
	CheckRuleCommitOverdue       = "commit_overdue"
)

// CheckFinding ist ein einzelner Befund eines Prüflaufs.
type CheckFinding struct {
	ID         uint          `gorm:"primaryKey" json:"id"`
	CheckRunID uint          `gorm:"index;not null" json:"checkRunId"`
	Rule       string        `gorm:"size:40;not null;index" json:"rule"`
	Severity   CheckSeverity `gorm:"size:20;not null;index" json:"severity"`

	// ObjectType und ObjectID zeigen auf das Bezugsobjekt, damit die Oberfläche
	// einen Knopf „hin dazu" anbieten kann. Ein Befund, den man nicht anspringen
	// kann, ist eine Hausaufgabe ohne Adresse.
	ObjectType string `gorm:"size:30" json:"objectType,omitempty"`
	ObjectID   string `gorm:"size:50" json:"objectId,omitempty"`
	ObjectName string `gorm:"size:120" json:"objectName,omitempty"`

	Message   string `gorm:"size:500;not null" json:"message"`
	Reference string `gorm:"size:120" json:"reference,omitempty"`
}

// CheckRun ist ein Prüflauf über einen Zeitraum bis zu einem Stichtag.
//
// Er wird gespeichert und nicht nur angezeigt: die Festschreibung hängt an ihm,
// und wenn ein blockierender Befund übergangen wurde, muss später nachvollziehbar
// sein, welcher und mit welcher Begründung (GoBD Rz. 34 ff., IKS).
type CheckRun struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	FiscalYear int    `gorm:"index;not null" json:"fiscalYear"`
	CutoffDate string `gorm:"size:10;not null;index" json:"cutoffDate"`
	// PeriodType ist der Anlass: "month", "quarter", "year" oder "" für einen
	// Lauf ohne bevorstehende Festschreibung.
	PeriodType string `gorm:"size:10" json:"periodType,omitempty"`

	CheckedEntries  int `json:"checkedEntries"`
	CheckedReceipts int `json:"checkedReceipts"`
	CheckedBankTx   int `json:"checkedBankTx"`

	// OverrideReason ist die Pflichtbegründung, mit der ein blockierender Befund
	// übergangen wurde. Leer heißt: es wurde nichts übergangen.
	OverrideReason string `gorm:"size:500" json:"overrideReason,omitempty"`

	Findings  []CheckFinding `gorm:"foreignKey:CheckRunID;constraint:OnDelete:CASCADE" json:"findings"`
	CreatedAt time.Time      `json:"createdAt"`
}

// EnsureLists ersetzt eine nicht belegte Befundliste durch eine leere.
//
// Der Lauf geht als JSON an die Oberfläche, und ein nil-Slice wird dort zu
// `null`. Der Festschreibungsdialog liest `findings.length` — ausgerechnet der
// saubere Lauf ohne Befund brächte ihn zu Fall, und die Festschreibung wäre über
// die Oberfläche nicht mehr möglich.
func (r *CheckRun) EnsureLists() {
	if r.Findings == nil {
		r.Findings = make([]CheckFinding, 0)
	}
}

// Blocking liefert die blockierenden Befunde.
func (r *CheckRun) Blocking() []CheckFinding {
	out := make([]CheckFinding, 0, len(r.Findings))
	for _, f := range r.Findings {
		if f.Severity == CheckBlocking {
			out = append(out, f)
		}
	}
	return out
}

// HasBlocking meldet, ob der Lauf die Festschreibung verhindert.
func (r *CheckRun) HasBlocking() bool { return len(r.Blocking()) > 0 }

// Warnings liefert die Hinweise.
func (r *CheckRun) Warnings() []CheckFinding {
	out := make([]CheckFinding, 0, len(r.Findings))
	for _, f := range r.Findings {
		if f.Severity == CheckWarning {
			out = append(out, f)
		}
	}
	return out
}

// BlockingSummary fasst die blockierenden Befunde für eine Fehlermeldung
// zusammen: die ersten drei im Wortlaut, der Rest gezählt.
//
// Eine Meldung, die zwanzig Befunde aufzählt, wird nicht gelesen; eine, die nur
// „es gibt Befunde" sagt, hilft nicht weiter.
func (r *CheckRun) BlockingSummary() string {
	blocking := r.Blocking()
	if len(blocking) == 0 {
		return ""
	}
	shown := make([]string, 0, 3)
	for _, f := range blocking {
		if len(shown) == 3 {
			break
		}
		shown = append(shown, f.Message)
	}
	out := strings.Join(shown, "; ")
	if len(blocking) > len(shown) {
		out += fmt.Sprintf(" und %d weitere", len(blocking)-len(shown))
	}
	return out
}

// CheckRunRepository persistiert die Prüfläufe.
type CheckRunRepository interface {
	Create(ctx context.Context, run *CheckRun) error
	FindByFiscalYear(ctx context.Context, fiscalYear int) ([]CheckRun, error)
	FindByID(ctx context.Context, id uint) (*CheckRun, error)
	// Latest liefert den jüngsten Lauf eines Geschäftsjahres oder nil.
	Latest(ctx context.Context, fiscalYear int) (*CheckRun, error)
}
