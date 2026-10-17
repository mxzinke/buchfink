package wailsbridge

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
	"github.com/buchfink/buchfink/internal/timestamp"
)

// requestTimestamp ist der Weg zur Zeitstempelstelle.
//
// Als Variable und nicht als direkter Aufruf: die Prüfung der Systemzeit gegen
// die beglaubigte Zeit (QUE-04) lässt sich sonst nicht prüfen — ein Test müsste
// entweder die Uhr des Rechners verstellen oder auf eine Stelle im Netz
// zugreifen, und beides ist kein Test. Im laufenden Programm zeigt sie auf
// timestamp.RequestToken.
var requestTimestamp = timestamp.RequestToken

// CommitPeriod schreibt einen Zeitraum bis einschließlich cutoffDate fest.
// Danach nimmt der Zeitraum keine neue Buchung mehr auf. Der Kettenkopf wird
// mit einem qualifizierten Zeitstempel (RFC 3161) verankert.
//
// Drei Schritte, damit die Zeitstempelstelle (bis zu 20 s ohne Netz) keine
// Schreibsperre hält: erst die Festschreibung mit ausstehendem Stempel, dann
// ohne Sperre der Netzaufruf, dann der Stempel. Scheitert der Netzaufruf,
// bleibt die Festschreibung stehen und der Stempel wird nachgeholt
// (retryPendingTimestamps).
//
// Vor der Festschreibung läuft der Prüfbericht. Blockierende Befunde
// verhindern sie, es sei denn, der Anwender übergeht sie mit einer Begründung
// — die steht dann am Prüflauf und im Protokoll. Nur in der App.
func (b *BuchfinkBridge) CommitPeriod(periodType, periodLabel, cutoffDate, overrideReason string) (*domain.Festschreibung, error) {
	if err := domain.CheckDate("Stichtag", cutoffDate); err != nil {
		return nil, err
	}
	rec, err := write(b, func(ctx context.Context, s *core.Services) (*domain.Festschreibung, error) {
		// Das Jahr folgt aus dem Stichtag, nicht aus der Ansicht.
		year := s.Journal.FiscalYearOf(ctx, cutoffDate)
		if latest, err := s.Repos.Festschreibung.LatestCutoff(ctx, year); err == nil && latest != "" && cutoffDate <= latest {
			return nil, fmt.Errorf("Zeitraum bis %s ist bereits festgeschrieben", latest)
		}
		if err := runPreCommitChecks(ctx, s, year, periodType, cutoffDate, overrideReason); err != nil {
			return nil, err
		}

		chainHead := domain.GenesisHash
		if last, err := s.Repos.Journal.GetLastEntry(ctx, year); err == nil && last != nil {
			chainHead = last.EntryHash
		}
		count, _ := s.Repos.Journal.Count(ctx, year)

		// Die Systemzeit wird einmal genommen: Festschreibungszeitpunkt an den
		// Buchungen und Zeitpunkt der Festschreibung selbst sind derselbe.
		now := time.Now().UTC()
		rec := &domain.Festschreibung{
			FiscalYear:      year,
			PeriodType:      periodType,
			PeriodLabel:     periodLabel,
			CutoffDate:      cutoffDate,
			ChainHead:       chainHead,
			EntryCount:      int(count),
			TimestampStatus: "pending",
			AppVersion:      buildinfo.Version,
			Actor:           actor.Actor(),
			CreatedAt:       now,
		}
		if err := s.Repos.Festschreibung.Create(ctx, rec); err != nil {
			return nil, fmt.Errorf("Festschreibung speichern: %w", err)
		}
		// Erst jetzt bekommen die Buchungen ihren Festschreibungszeitpunkt: er
		// verweist auf die Festschreibung, und die hat vorher keine Kennung.
		stamped, err := s.Repos.Journal.MarkCommitted(ctx, year, cutoffDate, rec.ID, now)
		if err != nil {
			return nil, fmt.Errorf("Buchungen festschreiben: %w", err)
		}
		rec.EntriesStamped = int(stamped)
		if err := s.Repos.Festschreibung.Update(ctx, rec); err != nil {
			return nil, err
		}

		details := fmt.Sprintf("Festschreibung %s (bis %s, %d Buchungen, davon %d neu festgeschrieben)",
			periodLabel, cutoffDate, count, rec.EntriesStamped)
		if reason := strings.TrimSpace(overrideReason); reason != "" {
			details += fmt.Sprintf(". Blockierende Befunde übergangen mit der Begründung: %s", reason)
		}
		return rec, s.Repos.Audit.Log(ctx, domain.AuditActionExport, "FESTSCHREIBUNG",
			fmt.Sprintf("%d", rec.ID), details)
	}, core.AppOnly())
	if err != nil {
		return nil, err
	}

	// Nur der Hash verlässt den Rechner.
	res, err := requestTimestamp(context.Background(), timestamp.DefaultTSA, rec.ChainHead)
	if err != nil {
		return rec, nil
	}
	if err := b.change(func(ctx context.Context, s *core.Services) error {
		return stampCommitment(ctx, s, rec, res, true)
	}, core.ExaminerExempt()); err != nil {
		return nil, fmt.Errorf("die Festschreibung steht, der Zeitstempel ließ sich aber nicht speichern: %w", err)
	}
	return rec, nil
}

// stampCommitment trägt den Zeitstempel an einer Festschreibung ein.
//
// Gleich nach der Festschreibung geholt, vergleicht er die beglaubigte Zeit
// mit der Systemzeit (QUE-04): eine Abweichung über der Toleranz bekommt einen
// eigenen Protokolleintrag mit beiden Zeiten. Ein nachgeholter Stempel sagt
// über die Uhr nichts, er ist um die Zeit ohne Netz später.
func stampCommitment(
	ctx context.Context, s *core.Services, rec *domain.Festschreibung, res *timestamp.Result, checkDrift bool,
) error {
	gt := res.GenTime
	rec.TimestampToken = res.Token
	rec.TSAName = res.TSAName
	rec.TSAGenTime = &gt
	rec.TimestampStatus = "confirmed"
	if checkDrift {
		rec.TimeDriftNote = service.TimeDriftNote(rec.CreatedAt, gt, res.TSAName)
	}
	if err := s.Repos.Festschreibung.Update(ctx, rec); err != nil {
		return err
	}
	if rec.TimeDriftNote == "" {
		return nil
	}
	return s.Repos.Audit.Log(ctx, domain.AuditActionIntegrityCheck, "TIME_DRIFT",
		fmt.Sprintf("%d", rec.ID),
		fmt.Sprintf("%s Systemzeit %s, beglaubigte Zeit %s.",
			rec.TimeDriftNote,
			rec.CreatedAt.UTC().Format(time.RFC3339),
			gt.UTC().Format(time.RFC3339)))
}

// runPreCommitChecks führt den Prüflauf aus und entscheidet, ob
// festgeschrieben werden darf. Vor der Jahresfestschreibung muss die
// gesetzliche Rücklage gebucht sein.
func runPreCommitChecks(ctx context.Context, s *core.Services, year int, periodType, cutoffDate, overrideReason string) error {
	fy, err := s.Closing.PeriodOf(ctx, year)
	if err != nil {
		return err
	}
	if periodType == "year" || cutoffDate >= fy.EndDate {
		if err := s.Closing.EnsureLegalReserve(ctx, year); err != nil {
			return err
		}
	}
	_, err = s.Checks.EnsureCommittable(ctx, service.CheckRequest{
		FiscalYear:     year,
		CutoffDate:     cutoffDate,
		PeriodType:     periodType,
		OverrideReason: overrideReason,
	})
	return err
}

// GetFestschreibungen returns the period commitments of the active fiscal year.
func (b *BuchfinkBridge) GetFestschreibungen() ([]domain.Festschreibung, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.Festschreibung, error) {
		return s.Repos.Festschreibung.FindByFiscalYear(ctx, b.viewYear())
	})
}

// VerifyFestschreibung re-checks a commitment offline: its timestamp token
// validity and whether the committed chain head still matches the live chain.
func (b *BuchfinkBridge) VerifyFestschreibung(id uint) (*domain.FestschreibungVerification, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*domain.FestschreibungVerification, error) {
		rec, err := s.Repos.Festschreibung.FindByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("Festschreibung nicht gefunden")
		}
		v := &domain.FestschreibungVerification{ID: id, TSAName: rec.TSAName}

		if last, err := s.Repos.Journal.GetLastEntry(ctx, rec.FiscalYear); err == nil {
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
	})
}

// retryPendingTimestamps holt die Zeitstempel der Festschreibungen nach, die
// ohne Netz entstanden. Ein Hintergrundlauf: der Netzaufruf hält keine Sperre,
// nur das Eintragen ist ein kurzer Schreibvorgang.
func (b *BuchfinkBridge) retryPendingTimestamps() {
	ctx := actor.WithChannel(context.Background(), actor.ChannelSystem)
	t, release, err := b.host.Acquire(ctx)
	if err != nil {
		return
	}
	defer release()
	var pending []domain.Festschreibung
	if err := t.Read(ctx, func(ctx context.Context, s *core.Services) error {
		var err error
		pending, err = s.Repos.Festschreibung.FindPendingTimestamp(ctx)
		return err
	}); err != nil {
		return
	}
	for i := range pending {
		rec := pending[i]
		res, err := requestTimestamp(ctx, timestamp.DefaultTSA, rec.ChainHead)
		if err != nil {
			continue // weiter ohne Netz; beim nächsten Start
		}
		_ = t.Write(ctx, func(ctx context.Context, s *core.Services) error {
			return stampCommitment(ctx, s, &rec, res, false)
		}, core.ExaminerExempt())
	}
}
