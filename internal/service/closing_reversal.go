package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Storno-Weg der Abschlussbausteine.
//
// Jeder Baustein weist eine zweite Buchung desselben Vorgangs ab und nennt in
// seiner Fehlermeldung den Storno als Weg zur Korrektur. Das ist richtig:
// § 239 Abs. 3 HGB lässt Gebuchtes nicht verändern, die Berichtigung ist die
// Generalumkehr. Nur muss dieser Weg dann auch zu Ende führen — und dazu gehört,
// dass jede Sperre und jede Auswertung die Generalumkehr erkennt.
//
// Eine stornierte Bildungsbuchung ist keine Bildung mehr. Sie darf im Folgejahr
// keine Auflösung nach sich ziehen, sie darf keinen Bestand ausweisen, und sie
// darf den Baustein nicht dauerhaft sperren. Ohne das wäre der Rat „storniere
// zuerst ihre Buchung" ein Rat in die Sackgasse: nach dem Storno stünde die
// Kartei weiter da, und ein zweiter Lauf wäre für immer abgewiesen.
//
// Gelesen wird über journalRepo.FindReversalOf: der Storno trägt den Verweis auf
// die Ursprungsbuchung (ReversalOfID), und nur so lässt sich von der Kartei aus
// feststellen, ob ihre Buchung noch steht.

// reversalIndex beantwortet die Frage „ist diese Buchung storniert?" und merkt
// sich die Antwort. Bericht und Saldenvortrag fragen dieselbe Bildungsbuchung
// mehrfach; ohne das Gedächtnis liefe je Auflösung eine Abfrage.
type reversalIndex struct {
	repo  domain.JournalRepository
	known map[uint]bool
}

func newReversalIndex(repo domain.JournalRepository) *reversalIndex {
	return &reversalIndex{repo: repo, known: map[uint]bool{}}
}

// reversed meldet, ob zu der Buchung eine Generalumkehr steht. Eine fehlende
// Buchungs-ID ist keine stornierte Buchung: ein Karteisatz ohne Buchung — der
// reine Vortrag auf neue Rechnung, der Inventurwert ohne Veränderung — bleibt
// gültig.
func (x *reversalIndex) reversed(ctx context.Context, entryID *uint) (bool, error) {
	if x == nil || x.repo == nil || entryID == nil || *entryID == 0 {
		return false, nil
	}
	if answer, ok := x.known[*entryID]; ok {
		return answer, nil
	}
	reversal, err := x.repo.FindReversalOf(ctx, *entryID)
	if err != nil {
		return false, fmt.Errorf(
			"zur Buchung %d ließ sich nicht feststellen, ob sie storniert wurde: %w", *entryID, err)
	}
	x.known[*entryID] = reversal != nil
	return reversal != nil, nil
}

// entryIsReversed ist die Einzelfrage ohne Gedächtnis.
func entryIsReversed(ctx context.Context, repo domain.JournalRepository, entryID *uint) (bool, error) {
	return newReversalIndex(repo).reversed(ctx, entryID)
}

// liveAccruals lässt die Posten aus, deren Bildungsbuchung storniert wurde.
//
// Der Auflösungsplan bleibt an ihnen hängen, damit die Kartei zeigt, was einmal
// geplant war; gebucht und ausgewiesen wird er nicht mehr.
func liveAccruals(
	ctx context.Context, repo domain.JournalRepository, accruals []domain.Accrual,
) ([]domain.Accrual, error) {
	index := newReversalIndex(repo)
	out := make([]domain.Accrual, 0, len(accruals))
	for i := range accruals {
		voided, err := index.reversed(ctx, accruals[i].FormationEntryID)
		if err != nil {
			return nil, err
		}
		if voided {
			continue
		}
		accrual := accruals[i]
		// Auch eine stornierte Auflösungsbuchung zählt nicht: sie ist wieder
		// offen und gehört erneut in den Vortrag des Zieljahres.
		releases := make([]domain.AccrualRelease, 0, len(accrual.Releases))
		for _, release := range accrual.Releases {
			gone, err := index.reversed(ctx, release.JournalEntryID)
			if err != nil {
				return nil, err
			}
			if gone {
				release.JournalEntryID = nil
			}
			releases = append(releases, release)
		}
		accrual.Releases = releases
		out = append(out, accrual)
	}
	return out, nil
}

// liveProvisions streicht die Bewegungen, deren Buchung per Generalumkehr
// zurückgenommen wurde, und mit ihnen die Rückstellungen, von denen danach
// nichts mehr übrig ist.
//
// Der Bestand einer Rückstellung ist die Summe ihrer Bewegungen. Bliebe eine
// stornierte Bildung darin stehen, zeigte der Rückstellungsspiegel einen
// Bestand, dem im Journal nichts mehr gegenübersteht — und die Bilanz wiese ihn
// aus, obwohl das Konto längst wieder auf null steht.
func liveProvisions(
	ctx context.Context, repo domain.JournalRepository, provisions []domain.Provision,
) ([]domain.Provision, error) {
	index := newReversalIndex(repo)
	out := make([]domain.Provision, 0, len(provisions))
	for i := range provisions {
		provision := provisions[i]
		movements := make([]domain.ProvisionMovement, 0, len(provision.Movements))
		for _, movement := range provision.Movements {
			voided, err := index.reversed(ctx, movement.JournalEntryID)
			if err != nil {
				return nil, err
			}
			if voided {
				continue
			}
			movements = append(movements, movement)
		}
		if len(provision.Movements) > 0 && len(movements) == 0 {
			// Jede Bewegung storniert: die Rückstellung ist nie entstanden.
			continue
		}
		provision.Movements = movements
		out = append(out, provision)
	}
	return out, nil
}
