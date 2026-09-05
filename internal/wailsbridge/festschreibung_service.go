package wailsbridge

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
	"github.com/buchfink/buchfink/internal/timestamp"
)

// CommitPeriod festschreibt (commits) an accounting period up to and including
// cutoffDate. After this, no new original booking may be backdated into the
// period. It silently anchors the current hash chain head with an RFC-3161
// trusted timestamp; if the TSA is offline the commitment still stands and the
// timestamp is fetched later (see retryPendingTimestamps).
//
// Vor der Festschreibung läuft der Prüfbericht. Blockierende Befunde verhindern
// sie, es sei denn, der Anwender übergeht sie mit einer Begründung — die steht
// dann am Prüflauf und im Protokoll. Ohne diese Reihenfolge wäre die
// Festschreibung ein Knopf, der einen unfertigen Stand für immer festhält.
func (b *BuchfinkBridge) CommitPeriod(periodType, periodLabel, cutoffDate, overrideReason string) (*domain.Festschreibung, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}

	if b.festschreibungRepo == nil || b.journalRepo == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	if len(cutoffDate) != 10 {
		return nil, fmt.Errorf("ungültiges Stichtagsdatum (erwartet YYYY-MM-DD)")
	}
	ctx := context.Background()

	// Guard against re-committing an already committed (or earlier) period.
	if latest, err := b.festschreibungRepo.LatestCutoff(ctx, b.currentYear); err == nil && latest != "" && cutoffDate <= latest {
		return nil, fmt.Errorf("Zeitraum bis %s ist bereits festgeschrieben", latest)
	}

	if err := b.runPreCommitChecks(ctx, periodType, cutoffDate, overrideReason); err != nil {
		return nil, err
	}

	// Anchor the current chain head of the fiscal year.
	chainHead := domain.GenesisHash
	if last, err := b.journalRepo.GetLastEntry(ctx, b.currentYear); err == nil && last != nil {
		chainHead = last.EntryHash
	}
	count, _ := b.journalRepo.Count(ctx, b.currentYear)

	rec := &domain.Festschreibung{
		FiscalYear:      b.currentYear,
		PeriodType:      periodType,
		PeriodLabel:     periodLabel,
		CutoffDate:      cutoffDate,
		ChainHead:       chainHead,
		EntryCount:      int(count),
		TimestampStatus: "pending",
		CreatedAt:       time.Now(),
	}

	// Best-effort trusted timestamp (only the hash leaves the machine).
	if res, err := timestamp.RequestToken(ctx, timestamp.DefaultTSA, chainHead); err == nil {
		gt := res.GenTime
		rec.TimestampToken = res.Token
		rec.TSAName = res.TSAName
		rec.TSAGenTime = &gt
		rec.TimestampStatus = "confirmed"
	}

	if err := b.festschreibungRepo.Create(ctx, rec); err != nil {
		return nil, fmt.Errorf("Festschreibung speichern: %w", err)
	}
	if b.auditRepo != nil {
		details := fmt.Sprintf("Festschreibung %s (bis %s, %d Buchungen, Zeitstempel: %s)",
			periodLabel, cutoffDate, count, rec.TimestampStatus)
		if reason := strings.TrimSpace(overrideReason); reason != "" {
			details += fmt.Sprintf(". Blockierende Befunde übergangen mit der Begründung: %s", reason)
		}
		_ = b.auditRepo.Log(ctx, domain.AuditActionExport, "FESTSCHREIBUNG",
			fmt.Sprintf("%d", rec.ID), details)
	}
	return rec, nil
}

// runPreCommitChecks führt den Prüflauf aus und entscheidet, ob festgeschrieben
// werden darf.
//
// Ohne Prüfdienst bleibt es bei der bisherigen AfA-Prüfung: ein Mandant, dessen
// Prüfläufe sich nicht speichern lassen, soll nicht ohne jede Prüfung
// festschreiben können.
func (b *BuchfinkBridge) runPreCommitChecks(ctx context.Context, periodType, cutoffDate, overrideReason string) error {
	if b.checkSvc == nil {
		return b.ensureDepreciationBooked(ctx, periodType)
	}
	_, err := b.checkSvc.EnsureCommittable(ctx, service.CheckRequest{
		CutoffDate:     cutoffDate,
		PeriodType:     periodType,
		OverrideReason: overrideReason,
	})
	return err
}

// ensureDepreciationBooked blocks the Festschreibung of a whole year while an
// Anlagegut still has AfA open for it.
//
// AfA is an Abschlussbuchung zum Bilanzstichtag: sie entsteht nicht im Lauf des
// Jahres, sondern zum Abschluss — und ein festgeschriebenes Jahr nimmt keine
// Buchung mehr auf. Ohne diese Prüfung ließe sich ein Jahr schließen, dessen
// Abschreibung fehlt, und das ließe sich danach nicht mehr geradeziehen.
//
// Monats- und Quartalsfestschreibungen prüfen nicht: dort ist die AfA nicht
// fällig.
func (b *BuchfinkBridge) ensureDepreciationBooked(ctx context.Context, periodType string) error {
	if periodType != "year" || b.assetSvc == nil {
		return nil
	}
	pending, err := b.assetSvc.PendingDepreciation(ctx, b.currentYear)
	if err != nil {
		return fmt.Errorf("die fällige Abschreibung konnte nicht geprüft werden: %w", err)
	}
	if len(pending) == 0 {
		return nil
	}

	var total domain.Cents
	names := make([]string, 0, len(pending))
	for _, p := range pending {
		total += p.Due
		if len(names) < 3 {
			names = append(names, fmt.Sprintf("%s %s", p.InventoryNumber, p.Name))
		}
	}
	list := strings.Join(names, ", ")
	if len(pending) > len(names) {
		list += fmt.Sprintf(" und %d weitere", len(pending)-len(names))
	}
	return fmt.Errorf(
		"für %d ist die Abschreibung auf %d Anlagegüter über zusammen %s € noch nicht gebucht (%s). "+
			"Die AfA ist eine Abschlussbuchung zum Bilanzstichtag und lässt sich nach der Festschreibung "+
			"nicht mehr nachholen — buche sie zuerst im Anlagenverzeichnis unter „Abschreibungen\"",
		b.currentYear, len(pending), total, list)
}

// GetFestschreibungen returns the period commitments of the active fiscal year.
func (b *BuchfinkBridge) GetFestschreibungen() ([]domain.Festschreibung, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.festschreibungRepo == nil {
		return []domain.Festschreibung{}, nil
	}
	return emptyList(b.festschreibungRepo.FindByFiscalYear(context.Background(), b.currentYear))
}

// VerifyFestschreibung re-checks a commitment offline: its timestamp token
// validity and whether the committed chain head still matches the live chain.
func (b *BuchfinkBridge) VerifyFestschreibung(id uint) (*domain.FestschreibungVerification, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.festschreibungRepo == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	ctx := context.Background()

	rec, err := b.festschreibungRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("Festschreibung nicht gefunden")
	}
	v := &domain.FestschreibungVerification{ID: id, TSAName: rec.TSAName}

	if last, err := b.journalRepo.GetLastEntry(ctx, rec.FiscalYear); err == nil {
		head := domain.GenesisHash
		if last != nil {
			head = last.EntryHash
		}
		v.CoversCurrent = head == rec.ChainHead
	}

	if len(rec.TimestampToken) == 0 {
		v.HasTimestamp = false
		v.IsValid = true // the commitment itself is valid; timestamp is still pending
		v.Message = fmt.Sprintf("Festgeschrieben bis %s (%d Buchungen). Zeitstempel noch ausstehend (offline) – wird automatisch nachgeholt.", rec.CutoffDate, rec.EntryCount)
		return v, nil
	}

	res, err := timestamp.VerifyToken(rec.TimestampToken, rec.ChainHead)
	if err != nil {
		v.HasTimestamp = true
		v.IsValid = false
		v.Message = fmt.Sprintf("Zeitstempel ungültig: %v", err)
		return v, nil
	}
	v.HasTimestamp = true
	v.IsValid = true
	gt := res.GenTime
	v.GenTime = &gt
	v.TSAName = res.TSAName
	if v.CoversCurrent {
		v.Message = fmt.Sprintf("Gültig – %d Buchungen bis %s, beglaubigt am %s durch %s (aktueller Stand).",
			rec.EntryCount, rec.CutoffDate, res.GenTime.Format("02.01.2006 15:04"), res.TSAName)
	} else {
		v.Message = fmt.Sprintf("Gültig – Stand vom %s durch %s beglaubigt; seitdem kamen weitere Buchungen hinzu.",
			res.GenTime.Format("02.01.2006 15:04"), res.TSAName)
	}
	return v, nil
}

// retryPendingTimestamps attempts to fetch trusted timestamps for commitments
// that were created while offline. Best-effort, non-blocking on failure.
func (b *BuchfinkBridge) retryPendingTimestamps() {
	if b.festschreibungRepo == nil {
		return
	}
	ctx := context.Background()
	pending, err := b.festschreibungRepo.FindPendingTimestamp(ctx)
	if err != nil {
		return
	}
	for i := range pending {
		rec := pending[i]
		res, err := timestamp.RequestToken(ctx, timestamp.DefaultTSA, rec.ChainHead)
		if err != nil {
			continue // still offline; try again next time
		}
		gt := res.GenTime
		rec.TimestampToken = res.Token
		rec.TSAName = res.TSAName
		rec.TSAGenTime = &gt
		rec.TimestampStatus = "confirmed"
		_ = b.festschreibungRepo.Update(ctx, &rec)
	}
}
