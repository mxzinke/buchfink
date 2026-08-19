package domain

import (
	"context"
	"time"
)

// Festschreibung records the GoBD "Festschreibung" (period commitment) of an
// accounting period. Once a period is committed, no new bookings may be
// backdated into it (corrections happen via Storno, dated at the correction
// time). Each Festschreibung silently anchors the current hash chain head with
// an RFC-3161 trusted timestamp as an additional, independent proof of existence.
type Festschreibung struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	FiscalYear  int    `gorm:"index;not null" json:"fiscalYear"`
	PeriodType  string `gorm:"size:10;not null" json:"periodType"`        // "month" | "quarter" | "year"
	PeriodLabel string `gorm:"size:40;not null" json:"periodLabel"`       // e.g. "Q1 2026", "März 2026"
	CutoffDate  string `gorm:"size:10;not null;index" json:"cutoffDate"`  // YYYY-MM-DD, inclusive — bookings up to here are locked
	ChainHead   string `gorm:"size:64;not null" json:"chainHead"`         // EntryHash of the last booking at commit time
	EntryCount  int    `json:"entryCount"`                                // bookings covered at commit time

	// Silent RFC-3161 trusted timestamp over ChainHead. Filled asynchronously;
	// if the TSA is unreachable at commit time the Festschreibung still stands and
	// the timestamp is fetched later (status "pending").
	TimestampToken  []byte     `gorm:"type:blob" json:"-"`
	TSAName         string     `gorm:"size:255" json:"tsaName"`
	TSAGenTime      *time.Time `json:"tsaGenTime,omitempty"`
	TimestampStatus string     `gorm:"size:20;default:'pending'" json:"timestampStatus"` // "confirmed" | "pending"

	CreatedAt time.Time `json:"createdAt"`
}

// FestschreibungRepository persists period commitments.
type FestschreibungRepository interface {
	Create(ctx context.Context, rec *Festschreibung) error
	Update(ctx context.Context, rec *Festschreibung) error
	FindByFiscalYear(ctx context.Context, fiscalYear int) ([]Festschreibung, error)
	FindByID(ctx context.Context, id uint) (*Festschreibung, error)
	// LatestCutoff returns the newest committed cutoff date for a fiscal year, or
	// "" if the year has no Festschreibung yet.
	LatestCutoff(ctx context.Context, fiscalYear int) (string, error)
	// FindPendingTimestamp returns commitments whose timestamp is still missing.
	FindPendingTimestamp(ctx context.Context) ([]Festschreibung, error)
}

// FestschreibungVerification is the UI-facing result of re-verifying a commitment.
type FestschreibungVerification struct {
	ID            uint       `json:"id"`
	HasTimestamp  bool       `json:"hasTimestamp"`
	IsValid       bool       `json:"isValid"`       // token verifies and covers the committed chain head
	CoversCurrent bool       `json:"coversCurrent"` // committed head still equals the live chain head
	GenTime       *time.Time `json:"genTime,omitempty"`
	TSAName       string     `json:"tsaName"`
	Message       string     `json:"message"`
}
