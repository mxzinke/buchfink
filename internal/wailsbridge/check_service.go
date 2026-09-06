package wailsbridge

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Prüfberichte und Fristen.
//
// Der Prüflauf ist der Bericht vor der Festschreibung: er sagt, was danach nicht
// mehr zu ändern wäre. Die Fristen kommen aus denselben Daten — was übermittelt
// oder festgeschrieben ist, ist erledigt, und niemand hakt es zusätzlich ab.

// RunChecks führt einen Prüflauf bis zum Stichtag aus und gibt ihn zurück, ohne
// ihn zu speichern.
//
// Gespeichert wird der Lauf, den `CommitPeriod` vor der Festschreibung ausführt:
// an ihm hängt die Festschreibung, und an ihm steht die Begründung für ein
// Übergehen. Würde auch die Vorschau abgelegt, stünden je Festschreibung zwei
// Läufe im Protokoll — und die Begründung an dem, der nichts festgeschrieben hat.
//
// `periodType` ist der Anlass ("month", "quarter", "year"). Er schaltet die
// Regeln zu, die nur vor der Jahresfestschreibung gelten; ohne ihn zeigte die
// Vorschau weniger Befunde, als die Festschreibung danach verlangt.
func (b *BuchfinkBridge) RunChecks(cutoffDate, periodType string) (*domain.CheckRun, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.checkSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.checkSvc.Preview(context.Background(), service.CheckRequest{
		CutoffDate: cutoffDate,
		PeriodType: periodType,
	})
}

// GetCheckRuns liefert die gespeicherten Prüfläufe eines Jahres, der jüngste
// zuerst.
func (b *BuchfinkBridge) GetCheckRuns(year int) ([]domain.CheckRun, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.checkSvc == nil {
		return []domain.CheckRun{}, nil
	}
	return emptyList(b.checkSvc.Runs(context.Background(), year))
}

// GetDeadlines liefert alle Termine eines Jahres: Voranmeldungen,
// Zusammenfassende Meldungen, Sondervorauszahlung, Festschreibungen, Aufstellung
// und Offenlegung sowie die Gründungspflichten.
func (b *BuchfinkBridge) GetDeadlines(year int) ([]domain.Deadline, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.deadlineSvc == nil {
		return []domain.Deadline{}, nil
	}
	return emptyList(b.deadlineSvc.Deadlines(context.Background(), year))
}

// MarkDeadlineDone hakt einen Termin ab, der sich nicht aus den Daten ergibt —
// die Umsatzsteuer-Jahreserklärung etwa.
func (b *BuchfinkBridge) MarkDeadlineDone(key, date string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return err
	}
	if b.deadlineSvc == nil {
		return fmt.Errorf("kein aktiver Mandant")
	}
	return b.deadlineSvc.MarkDone(context.Background(), key, date)
}
