package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// -------------------------------------------------------------------------
// Wertaufholung (§ 253 Abs. 5 Satz 1 HGB)
// -------------------------------------------------------------------------

// WriteUpCandidate ist ein Anlagegut mit außerplanmäßiger Abschreibung, für das
// sich die Frage der Zuschreibung stellt.
type WriteUpCandidate struct {
	AssetID         uint         `json:"assetId"`
	InventoryNumber string       `json:"inventoryNumber"`
	Name            string       `json:"name"`
	Account         string       `json:"account"`
	BookValue       domain.Cents `json:"bookValue"`
	// ContinuedCost ist der Buchwert, den das Anlagegut ohne die
	// außerplanmäßige Abschreibung heute hätte — die Obergrenze der Zuschreibung.
	ContinuedCost domain.Cents `json:"continuedCost"`
	// MaxWriteUp ist der Spielraum: fortgeführte Anschaffungskosten minus
	// Buchwert.
	MaxWriteUp domain.Cents `json:"maxWriteUp"`
	// Impairments sind die außerplanmäßigen Abschreibungen mit ihren Gründen.
	Impairments []WriteUpImpairment `json:"impairments"`
	// Confirmed sagt, ob für dieses Geschäftsjahr bereits bestätigt wurde, dass
	// der Grund fortbesteht.
	Confirmed     bool   `json:"confirmed"`
	ConfirmedNote string `json:"confirmedNote,omitempty"`
	Note          string `json:"note"`
}

// WriteUpImpairment ist eine frühere außerplanmäßige Abschreibung.
type WriteUpImpairment struct {
	FiscalYear int          `json:"fiscalYear"`
	Date       string       `json:"date"`
	Amount     domain.Cents `json:"amount"`
	Reason     string       `json:"reason"`
}

// WriteUpReport ist der Bericht eines Geschäftsjahres.
type WriteUpReport struct {
	FiscalYear int                `json:"fiscalYear"`
	Candidates []WriteUpCandidate `json:"candidates"`
	// Open zählt die Anlagegüter, für die weder zugeschrieben noch bestätigt
	// wurde.
	Open int    `json:"open"`
	Note string `json:"note"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
func (r *WriteUpReport) EnsureLists() {
	if r.Candidates == nil {
		r.Candidates = make([]WriteUpCandidate, 0)
	}
}

// WriteUpReport listet die Anlagegüter, bei denen zu prüfen ist, ob der Grund
// der außerplanmäßigen Abschreibung weggefallen ist.
//
// Die Zuschreibung ist ein Gebot und kein Wahlrecht (§ 253 Abs. 5 Satz 1 HGB):
// fällt der Grund weg, ist zuzuschreiben — höchstens bis zu den fortgeführten
// Anschaffungskosten. Ob er weggefallen ist, weiß nur der Bilanzierende. Der
// Bericht stellt deshalb die Frage und rechnet den Spielraum; die Antwort gibt
// ein Mensch, und sie wird festgehalten, damit sie im nächsten Jahr erneut
// gestellt werden kann und nicht in diesem zweimal.
func (s *AssetService) WriteUpReport(ctx context.Context, year int) (*WriteUpReport, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	out := &WriteUpReport{FiscalYear: year}
	out.EnsureLists()

	startMonth := s.fiscalYearStartMonth(ctx)
	// Einmal für den ganzen Bericht, nicht je Anlagegut.
	periods := s.fiscalPeriods(ctx)
	assets, err := s.assetRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	for i := range assets {
		asset := &assets[i]
		if asset.IsDisposed() && disposedBefore(asset, year) {
			continue
		}
		impairments := make([]WriteUpImpairment, 0, 2)
		for _, m := range asset.Movements {
			if m.Kind == domain.AssetMovementImpairment && m.FiscalYear <= year {
				impairments = append(impairments, WriteUpImpairment{
					FiscalYear: m.FiscalYear, Date: m.Date,
					Amount: m.DepreciationAmount, Reason: m.Note,
				})
			}
		}
		if len(impairments) == 0 {
			continue
		}
		if asset.Account == accounting.GoodwillAccount {
			// § 253 Abs. 5 Satz 2 HGB verbietet die Zuschreibung auf den
			// Geschäfts- oder Firmenwert. Ihn in eine Liste zu stellen, aus der
			// nichts folgen darf, wäre eine Frage ohne mögliche Antwort.
			continue
		}

		s.enrich(asset, year, startMonth, periods)
		ceiling, err := s.writeUpCeiling(ctx, asset, year, startMonth)
		if err != nil {
			return nil, err
		}
		if ceiling <= 0 {
			continue
		}

		candidate := WriteUpCandidate{
			AssetID: asset.ID, InventoryNumber: asset.InventoryNumber, Name: asset.Name,
			Account: asset.Account, BookValue: asset.BookValue,
			ContinuedCost: asset.BookValue + ceiling, MaxWriteUp: ceiling,
			Impairments:   impairments,
			Confirmed:     asset.ImpairmentPersistsYear == year,
			ConfirmedNote: asset.ImpairmentPersistsNote,
		}
		if candidate.Confirmed {
			candidate.Note = fmt.Sprintf(
				"Für %d ist bestätigt, dass der Grund der außerplanmäßigen Abschreibung fortbesteht: %s",
				year, asset.ImpairmentPersistsNote)
		} else {
			out.Open++
			candidate.Note = fmt.Sprintf(
				"Bis zu %s € könnten zugeschrieben werden. Zu prüfen ist, ob der Grund der außerplanmäßigen "+
					"Abschreibung weggefallen ist — dann ist die Zuschreibung nach § 253 Abs. 5 Satz 1 "+
					"HGB geboten und kein Wahlrecht.", ceiling)
		}
		out.Candidates = append(out.Candidates, candidate)
	}

	sort.SliceStable(out.Candidates, func(i, j int) bool {
		if out.Candidates[i].Confirmed != out.Candidates[j].Confirmed {
			return !out.Candidates[i].Confirmed
		}
		return out.Candidates[i].InventoryNumber < out.Candidates[j].InventoryNumber
	})
	out.Note = fmt.Sprintf(
		"%d von %d Anlagegütern mit außerplanmäßiger Abschreibung sind für %d noch nicht beantwortet. "+
			"Entweder wird zugeschrieben oder festgehalten, dass der Grund fortbesteht.",
		out.Open, len(out.Candidates), year)
	return out, nil
}

// ConfirmImpairmentPersists hält fest, dass der Grund einer außerplanmäßigen
// Abschreibung im Geschäftsjahr fortbesteht.
func (s *AssetService) ConfirmImpairmentPersists(
	ctx context.Context, assetID uint, year int, note string,
) (*WriteUpReport, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*WriteUpReport, error) {
		return s.confirmImpairmentPersists(ctx, assetID, year, note)
	})
}

func (s *AssetService) confirmImpairmentPersists(
	ctx context.Context, assetID uint, year int, note string,
) (*WriteUpReport, error) {
	if strings.TrimSpace(note) == "" {
		return nil, fmt.Errorf(
			"halte fest, warum der Grund der außerplanmäßigen Abschreibung fortbesteht. Die " +
				"Zuschreibung ist ein Gebot; sie zu unterlassen ist eine Aussage und braucht eine " +
				"Begründung")
	}
	if err := requireYear(year); err != nil {
		return nil, err
	}
	asset, err := s.assetRepo.FindByID(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("Anlagegut %d wurde nicht gefunden: %w", assetID, err)
	}
	asset.ImpairmentPersistsYear = year
	asset.ImpairmentPersistsNote = strings.TrimSpace(note)
	if err := s.assetRepo.Save(ctx, asset); err != nil {
		return nil, fmt.Errorf("die Bestätigung ließ sich nicht speichern: %w", err)
	}
	if err := s.audit(ctx, domain.AuditActionUpdate, asset.ID, fmt.Sprintf(
		"Wertaufholung %d geprüft: der Grund der außerplanmäßigen Abschreibung auf %s besteht fort — %s",
		year, asset.InventoryNumber, note)); err != nil {
		return nil, err
	}
	return s.WriteUpReport(ctx, year)
}
