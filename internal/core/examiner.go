package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/domain"
)

// Der Prüfermodus friert den Bestand eines Mandanten während einer
// Außenprüfung ein (§ 147 Abs. 6 AO). Sein Zeitraum steht in der
// Mandantendatenbank (domain.ExaminerPeriod) und reist mit Sicherung und
// Datenordner. Tenant.Write weist in ihm jeden Schreibvorgang der Oberfläche
// und jedes Agenten ab; Lesen, Ausgeben und Protokollieren bleiben möglich,
// ebenso Programmläufe (Erfassungsweg system) wie Zeitstempel und Sicherung,
// die die Bücher nicht ändern. Ein- und ausschalten lässt er sich nur in der
// App.

// ExaminerState ist der Stand des Prüfermodus eines Mandanten.
type ExaminerState struct {
	Active bool
	From   string
	Until  string
	Reason string
}

// examinerCache hält den jüngsten nicht beendeten Zeitraum. Ob er gilt, wird
// bei jeder Abfrage gegen den heutigen Tag gerechnet; ein abgelaufener Modus
// endet so ohne Schreibvorgang.
type examinerCache struct {
	period atomic.Pointer[domain.ExaminerPeriod]
}

func (c *examinerCache) state(today string) ExaminerState {
	p := c.period.Load()
	if p == nil {
		return ExaminerState{}
	}
	return ExaminerState{Active: p.ActiveOn(today), From: p.From, Until: p.Until, Reason: p.Reason}
}

// loadExaminer liest den Zeitraum aus der Datenbank in den Speicher.
func (t *Tenant) loadExaminer(ctx context.Context) error {
	return t.Read(systemContext(ctx), func(ctx context.Context, s *Services) error {
		latest, err := s.Repos.Examiner.Latest(ctx)
		if err != nil {
			return fmt.Errorf("der Prüfermodus ließ sich nicht lesen: %w", err)
		}
		t.examiner.period.Store(latest)
		return nil
	})
}

// Examiner liefert den Stand des Prüfermodus am heutigen Tag.
func (t *Tenant) Examiner() ExaminerState { return t.examiner.state(today()) }

// ensureWritable weist im Prüfermodus Schreibvorgänge der Oberfläche und der
// Agenten ab. Programmläufe (system) ändern die Bücher nicht und laufen weiter.
func (t *Tenant) ensureWritable(channel actor.Channel) error {
	if channel == actor.ChannelSystem {
		return nil
	}
	return examinerError(t.Examiner())
}

func examinerError(state ExaminerState) error {
	if !state.Active {
		return nil
	}
	message := fmt.Sprintf("Der Prüfermodus ist bis zum %s aktiv. Bis dahin nimmt die Buchführung keine Änderung auf",
		germanDate(state.Until))
	if strings.TrimSpace(state.Reason) != "" {
		message += fmt.Sprintf(" (Grund: %s)", state.Reason)
	}
	return errors.New(message + ".")
}

// EnableExaminer schaltet den Prüfermodus bis einschließlich until ein. Datum
// und Grund sind Pflicht: ohne Ende bliebe der Modus liegen, ohne Grund ließe
// sich später nicht sagen, welche Prüfung ihn veranlasst hat. Ein laufender
// Zeitraum wird dabei abgelöst und bleibt als Verlauf stehen.
func (t *Tenant) EnableExaminer(ctx context.Context, until, reason string) (ExaminerState, error) {
	if err := requireApp(ctx); err != nil {
		return t.Examiner(), err
	}
	day := today()
	if strings.TrimSpace(until) == "" {
		return t.Examiner(), fmt.Errorf("der Prüfermodus braucht ein Datum, bis zu dem er gilt")
	}
	if _, err := time.Parse(domain.DateLayout, until); err != nil {
		return t.Examiner(), fmt.Errorf("%q ist kein gültiges Datum (erwartet JJJJ-MM-TT)", until)
	}
	if until < day {
		return t.Examiner(), fmt.Errorf("der Prüfermodus kann nicht in der Vergangenheit enden")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return t.Examiner(), fmt.Errorf("bitte den Grund angeben, aus dem der Prüfermodus eingeschaltet wird")
	}
	channel, _ := actor.ChannelOf(ctx)
	return t.changeExaminer(ctx, func(ctx context.Context, s *Services, previous *domain.ExaminerPeriod) (*domain.ExaminerPeriod, string, error) {
		if previous.ActiveOn(day) {
			previous.EndedOn = &day
			previous.EndReason = fmt.Sprintf("abgelöst durch den Zeitraum bis %s", until)
			if err := t.repos.examiner.Save(ctx, previous); err != nil {
				return nil, "", err
			}
		}
		next := &domain.ExaminerPeriod{
			From: day, Until: until, Reason: reason,
			Actor: actor.Actor(), Channel: string(channel), CreatedAt: time.Now().UTC(),
		}
		if err := t.repos.examiner.Create(ctx, next); err != nil {
			return nil, "", err
		}
		return next, fmt.Sprintf("Prüfermodus eingeschaltet bis %s. Grund: %s", until, reason), nil
	})
}

// DisableExaminer beendet den Prüfermodus vorzeitig. Der Grund ist Pflicht:
// ein vorzeitiges Ende hat jemand zu verantworten.
func (t *Tenant) DisableExaminer(ctx context.Context, reason string) (ExaminerState, error) {
	if err := requireApp(ctx); err != nil {
		return t.Examiner(), err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return t.Examiner(), fmt.Errorf("bitte den Grund angeben, aus dem der Prüfermodus beendet wird")
	}
	day := today()
	return t.changeExaminer(ctx, func(ctx context.Context, s *Services, previous *domain.ExaminerPeriod) (*domain.ExaminerPeriod, string, error) {
		if !previous.ActiveOn(day) {
			return nil, "", fmt.Errorf("der Prüfermodus ist nicht eingeschaltet")
		}
		previous.EndedOn = &day
		previous.EndReason = reason
		if err := t.repos.examiner.Save(ctx, previous); err != nil {
			return nil, "", err
		}
		return nil, fmt.Sprintf("Prüfermodus beendet (war bis %s). Grund: %s", previous.Until, reason), nil
	})
}

// changeExaminer schreibt einen neuen Stand samt Vorher/Nachher ins Protokoll
// und lädt ihn nach dem Schreiben in den Speicher.
func (t *Tenant) changeExaminer(
	ctx context.Context,
	change func(ctx context.Context, s *Services, previous *domain.ExaminerPeriod) (*domain.ExaminerPeriod, string, error),
) (ExaminerState, error) {
	err := t.Write(ctx, func(ctx context.Context, s *Services) error {
		previous, err := s.Repos.Examiner.Latest(ctx)
		if err != nil {
			return err
		}
		before := examinerFields(previous, today())
		next, details, err := change(ctx, s, previous)
		if err != nil {
			return err
		}
		return t.repos.audit.LogChange(ctx, domain.AuditActionUpdate, domain.AuditEntityReadOnly, t.ID(),
			details, before, examinerFields(next, today()))
	}, ExaminerExempt(), AppOnly())
	if err != nil {
		return t.Examiner(), err
	}
	if err := t.loadExaminer(ctx); err != nil {
		return t.Examiner(), err
	}
	return t.Examiner(), nil
}

func examinerFields(p *domain.ExaminerPeriod, day string) map[string]string {
	if !p.ActiveOn(day) {
		return map[string]string{"readOnlyUntil": "", "readOnlyReason": ""}
	}
	return map[string]string{"readOnlyUntil": p.Until, "readOnlyReason": p.Reason}
}

// EnableExaminer schaltet den Prüfermodus des offenen Mandanten ein.
func (h *Host) EnableExaminer(ctx context.Context, until, reason string) (domain.AppConfig, error) {
	t, release, err := h.Acquire(ctx)
	if err != nil {
		return h.Config(), fmt.Errorf("kein aktiver Mandant gefunden")
	}
	defer release()
	_, err = t.EnableExaminer(ctx, until, reason)
	return h.Config(), err
}

// DisableExaminer beendet den Prüfermodus des offenen Mandanten.
func (h *Host) DisableExaminer(ctx context.Context, reason string) (domain.AppConfig, error) {
	t, release, err := h.Acquire(ctx)
	if err != nil {
		return h.Config(), fmt.Errorf("kein aktiver Mandant gefunden")
	}
	defer release()
	_, err = t.DisableExaminer(ctx, reason)
	return h.Config(), err
}
