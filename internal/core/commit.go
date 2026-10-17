package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
	"github.com/buchfink/buchfink/internal/timestamp"
)

// CommitRequest beschreibt eine Festschreibung. Das Geschäftsjahr folgt aus
// dem Stichtag.
type CommitRequest struct {
	PeriodType     string
	PeriodLabel    string
	CutoffDate     string
	OverrideReason string
}

// CommitPeriod schreibt einen Zeitraum bis einschließlich CutoffDate fest.
// Danach nimmt er keine neue Buchung mehr auf; der Kettenkopf wird mit einem
// qualifizierten Zeitstempel (RFC 3161) verankert. Nur in der App.
//
// Drei Schritte, damit die Zeitstempelstelle (bis zu 20 s ohne Netz) keine
// Schreibsperre hält:
//  1. Ein Schreibvorgang legt die Festschreibung mit ausstehendem Stempel an,
//     markiert die Buchungen und protokolliert — ganz oder gar nicht.
//  2. Ohne Sperre wird der Stempel geholt; nur der Hash verlässt den Rechner.
//  3. Ein kurzer Schreibvorgang trägt ihn ein. Scheitert Schritt 2, bleibt der
//     Stempel ausstehend und der Hintergrundlauf holt ihn nach.
//
// Vor der Festschreibung läuft der Prüfbericht. Blockierende Befunde
// verhindern sie, es sei denn, der Anwender übergeht sie mit einer Begründung;
// die steht dann am Prüflauf und im Protokoll.
func (t *Tenant) CommitPeriod(ctx context.Context, r CommitRequest) (*domain.Festschreibung, error) {
	if err := requireApp(ctx); err != nil {
		return nil, err
	}
	if err := domain.CheckDate("Stichtag", r.CutoffDate); err != nil {
		return nil, err
	}
	var rec *domain.Festschreibung
	if err := t.Write(ctx, func(ctx context.Context, s *Services) error {
		var err error
		rec, err = commit(ctx, s, t.repos, r)
		return err
	}, AppOnly()); err != nil {
		return nil, err
	}

	res, err := t.network.Timestamp(ctx, timestamp.DefaultTSA, rec.ChainHead)
	if err != nil {
		t.logger.Info("Zeitstempel ausstehend, wird nachgeholt", "festschreibung", rec.ID, "fehler", err)
		return rec, nil
	}
	stamped := false
	if err := t.Write(ctx, func(ctx context.Context, _ *Services) (err error) {
		stamped, err = stampCommitment(ctx, t.repos, rec, res, true)
		return err
	}, ExaminerExempt()); err != nil {
		return nil, fmt.Errorf("die Festschreibung steht, der Zeitstempel ließ sich aber nicht speichern: %w", err)
	}
	if !stamped {
		// Der Hintergrundlauf hat schneller gestempelt; sein Stand gilt.
		err = t.Read(ctx, func(ctx context.Context, s *Services) (err error) {
			rec, err = s.Repos.Festschreibung.FindByID(ctx, rec.ID)
			return err
		})
	}
	return rec, err
}

// commit ist der erste Schritt der Festschreibung.
//
// Die Jahresfestschreibung darf denselben Stichtag tragen wie die des letzten
// Monats: dazwischen liegt die Abschlussperiode, in der die Abschlussbuchungen
// zum Bilanzstichtag entstehen. Erst die Jahresfestschreibung schließt sie.
func commit(ctx context.Context, s *Services, st stores, r CommitRequest) (*domain.Festschreibung, error) {
	year := s.Journal.FiscalYearOf(ctx, r.CutoffDate)
	committed, err := s.Repos.Festschreibung.FindByFiscalYear(ctx, year)
	if err != nil {
		return nil, err
	}
	latest := ""
	for _, rec := range committed {
		if rec.PeriodType == domain.PeriodTypeYear {
			return nil, fmt.Errorf("das Geschäftsjahr %d ist bereits festgeschrieben (bis %s)", year, rec.CutoffDate)
		}
		if rec.CutoffDate > latest {
			latest = rec.CutoffDate
		}
	}
	if r.PeriodType == domain.PeriodTypeYear {
		fy, err := s.Closing.PeriodOf(ctx, year)
		if err != nil {
			return nil, err
		}
		if r.CutoffDate != fy.EndDate {
			return nil, fmt.Errorf("die Jahresfestschreibung gilt bis zum Bilanzstichtag %s, nicht bis %s",
				fy.EndDate, r.CutoffDate)
		}
	} else if latest != "" && r.CutoffDate <= latest {
		return nil, fmt.Errorf("Zeitraum bis %s ist bereits festgeschrieben", latest)
	}
	if err := preCommitChecks(ctx, s, year, r); err != nil {
		return nil, err
	}

	chainHead := domain.GenesisHash
	last, err := s.Repos.Journal.GetLastEntry(ctx, year)
	if err != nil {
		return nil, err
	}
	if last != nil {
		chainHead = last.EntryHash
	}
	count, err := s.Repos.Journal.Count(ctx, year)
	if err != nil {
		return nil, err
	}

	// Die Systemzeit wird einmal genommen: Festschreibungszeitpunkt an den
	// Buchungen und Zeitpunkt der Festschreibung selbst sind derselbe.
	now := time.Now().UTC()
	rec := &domain.Festschreibung{
		FiscalYear:      year,
		PeriodType:      r.PeriodType,
		PeriodLabel:     r.PeriodLabel,
		CutoffDate:      r.CutoffDate,
		ChainHead:       chainHead,
		EntryCount:      int(count),
		TimestampStatus: "pending",
		AppVersion:      buildinfo.Version,
		Actor:           actor.Actor(),
		CreatedAt:       now,
	}
	if err := st.festschreibung.Create(ctx, rec); err != nil {
		return nil, fmt.Errorf("Festschreibung speichern: %w", err)
	}
	// Erst jetzt bekommen die Buchungen ihren Festschreibungszeitpunkt: er
	// verweist auf die Festschreibung, und die hat vorher keine Kennung.
	stamped, err := st.journal.MarkCommitted(ctx, year, r.CutoffDate, rec.ID, now)
	if err != nil {
		return nil, fmt.Errorf("Buchungen festschreiben: %w", err)
	}
	rec.EntriesStamped = int(stamped)
	if err := st.festschreibung.Update(ctx, rec); err != nil {
		return nil, err
	}

	details := fmt.Sprintf("Festschreibung %s (bis %s, %d Buchungen, davon %d neu festgeschrieben)",
		r.PeriodLabel, r.CutoffDate, count, rec.EntriesStamped)
	if reason := strings.TrimSpace(r.OverrideReason); reason != "" {
		details += fmt.Sprintf(". Blockierende Befunde übergangen mit der Begründung: %s", reason)
	}
	return rec, s.Repos.Audit.Log(ctx, domain.AuditActionCommit, "FESTSCHREIBUNG", fmt.Sprintf("%d", rec.ID), details)
}

// preCommitChecks führt den Prüflauf aus und entscheidet, ob festgeschrieben
// werden darf. Vor der Jahresfestschreibung muss die gesetzliche Rücklage
// gebucht sein.
func preCommitChecks(ctx context.Context, s *Services, year int, r CommitRequest) error {
	fy, err := s.Closing.PeriodOf(ctx, year)
	if err != nil {
		return err
	}
	if r.PeriodType == domain.PeriodTypeYear || r.CutoffDate >= fy.EndDate {
		if err := s.Closing.EnsureLegalReserve(ctx, year); err != nil {
			return err
		}
	}
	_, err = s.Checks.EnsureCommittable(ctx, service.CheckRequest{
		FiscalYear:     year,
		CutoffDate:     r.CutoffDate,
		PeriodType:     r.PeriodType,
		OverrideReason: r.OverrideReason,
	})
	return err
}

// stampCommitment trägt den Zeitstempel an einer Festschreibung ein, nur wenn
// er noch aussteht: Festschreibung und Hintergrundlauf können beide stempeln,
// und der zweite darf Token und Abweichungsvermerk des ersten nicht ersetzen.
// Es meldet, ob eingetragen wurde.
//
// Gleich nach der Festschreibung geholt, vergleicht er die beglaubigte Zeit
// mit der Systemzeit (QUE-04): eine Abweichung über der Toleranz bekommt einen
// eigenen Protokolleintrag mit beiden Zeiten. Ein nachgeholter Stempel sagt
// über die Uhr nichts, er ist um die Zeit ohne Netz später.
func stampCommitment(
	ctx context.Context, st stores, rec *domain.Festschreibung, res *timestamp.Result, checkDrift bool,
) (bool, error) {
	gt := res.GenTime
	next := *rec
	next.TimestampToken = res.Token
	next.TSAName = res.TSAName
	next.TSAGenTime = &gt
	next.TimestampStatus = "confirmed"
	if checkDrift {
		next.TimeDriftNote = service.TimeDriftNote(rec.CreatedAt, gt, res.TSAName)
	}
	stamped, err := st.festschreibung.ConfirmTimestamp(ctx, &next)
	if err != nil || !stamped {
		return false, err
	}
	*rec = next
	if rec.TimeDriftNote == "" {
		return true, nil
	}
	return true, st.audit.Log(ctx, domain.AuditActionIntegrityCheck, "TIME_DRIFT",
		fmt.Sprintf("%d", rec.ID),
		fmt.Sprintf("%s Systemzeit %s, beglaubigte Zeit %s.",
			rec.TimeDriftNote,
			rec.CreatedAt.UTC().Format(time.RFC3339),
			gt.UTC().Format(time.RFC3339)))
}

// RetryTimestamps holt die Zeitstempel der Festschreibungen nach, die ohne Netz
// entstanden, und meldet, wie viele eingetragen wurden. Der Netzaufruf hält
// keine Sperre, nur das Eintragen ist ein kurzer Schreibvorgang.
func (t *Tenant) RetryTimestamps(ctx context.Context) (int, error) {
	ctx = systemContext(ctx)
	var pending []domain.Festschreibung
	if err := t.Read(ctx, func(ctx context.Context, s *Services) error {
		var err error
		pending, err = s.Repos.Festschreibung.FindPendingTimestamp(ctx)
		return err
	}); err != nil {
		return 0, err
	}
	stamped := 0
	for i := range pending {
		rec := pending[i]
		res, err := t.network.Timestamp(ctx, timestamp.DefaultTSA, rec.ChainHead)
		if err != nil {
			// Weiter ohne Netz; beim nächsten Lauf.
			continue
		}
		applied := false
		if err := t.Write(ctx, func(ctx context.Context, _ *Services) (err error) {
			applied, err = stampCommitment(ctx, t.repos, &rec, res, false)
			return err
		}, ExaminerExempt()); err != nil {
			return stamped, err
		}
		if applied {
			stamped++
		}
	}
	return stamped, nil
}

// VerifyCommitment prüft eine Festschreibung ohne Netz: ob ihr Zeitstempel
// gültig ist und ob der festgeschriebene Kettenkopf noch dem Journal
// entspricht.
func (t *Tenant) VerifyCommitment(ctx context.Context, id uint) (*domain.FestschreibungVerification, error) {
	var v *domain.FestschreibungVerification
	err := t.Read(ctx, func(ctx context.Context, s *Services) error {
		rec, err := s.Repos.Festschreibung.FindByID(ctx, id)
		if err != nil {
			return fmt.Errorf("Festschreibung nicht gefunden")
		}
		v = &domain.FestschreibungVerification{ID: id, TSAName: rec.TSAName}

		if last, err := s.Repos.Journal.GetLastEntry(ctx, rec.FiscalYear); err == nil {
			head := domain.GenesisHash
			if last != nil {
				head = last.EntryHash
			}
			v.CoversCurrent = head == rec.ChainHead
		}

		if len(rec.TimestampToken) == 0 {
			v.HasTimestamp = false
			v.IsValid = true // die Festschreibung selbst gilt; der Stempel steht noch aus
			v.Message = fmt.Sprintf("Festgeschrieben bis %s (%d Buchungen). Zeitstempel noch ausstehend (offline) – wird automatisch nachgeholt.", rec.CutoffDate, rec.EntryCount)
			return nil
		}

		res, err := timestamp.VerifyToken(rec.TimestampToken, rec.ChainHead)
		if err != nil {
			v.HasTimestamp = true
			v.IsValid = false
			v.Message = fmt.Sprintf("Zeitstempel ungültig: %v", err)
			return nil
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
		return nil
	})
	return v, err
}
