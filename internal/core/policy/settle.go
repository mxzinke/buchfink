package policy

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/inbox"
)

// Moved ist eine Datei, die nach der Ablage aus dem Eingang nach Übernommen/
// ging.
type Moved struct {
	From string
	To   string
}

// SettleInbox räumt nach einer gelungenen Ablage den Eingang auf: jede der
// Dateien, die im Eingang liegt und deren Inhalt im Archiv steht, geht nach
// Übernommen/<JJJJ-MM> des Tages day, auch eine exakte Dublette. Dateien in
// Leseordnern und in Übernommen/ bleiben liegen. Was sich nicht verschieben
// lässt, nennt ein Hinweis; die Ablage bleibt gültig. Es läuft nach dem
// Vorgang, außerhalb der Schreibsperre.
func SettleInbox(ctx context.Context, t *core.Tenant, f Folders, paths []string, day time.Time) ([]Moved, []string) {
	moved, hints := []Moved{}, []string{}
	if err := f.CheckInbox(); err != nil {
		return moved, append(hints, "Der Eingang ließ sich nicht aufräumen: "+err.Error())
	}
	hashes := map[string]string{}
	for _, p := range paths {
		if _, seen := hashes[p]; seen || !inbox.Contains(f.Inbox, p) {
			continue
		}
		sum, err := inbox.Hash(p)
		if err != nil {
			continue
		}
		hashes[p] = sum
	}
	if len(hashes) == 0 {
		return moved, hints
	}
	sums := make([]string, 0, len(hashes))
	for _, sum := range hashes {
		sums = append(sums, sum)
	}
	var archived map[string]uint
	if err := t.ReadAgent(ctx, func(ctx context.Context, s core.AgentServices) (err error) {
		archived, err = s.Receipts.ReceiptIDsByFileHash(ctx, sums)
		return err
	}); err != nil {
		return moved, append(hints, "Der Eingang ließ sich nicht aufräumen: "+err.Error())
	}
	for _, p := range paths {
		sum, ok := hashes[p]
		if !ok {
			continue
		}
		delete(hashes, p)
		name := filepath.Base(p)
		if _, done := archived[sum]; !done {
			hints = append(hints, fmt.Sprintf("%s hat sich seit der Ablage geändert und bleibt im Eingang.", name))
			continue
		}
		to, err := inbox.Settle(f.Inbox, p, day)
		if err != nil {
			hints = append(hints, fmt.Sprintf("%s ließ sich nicht nach %s verschieben (%v); abgelegt ist der Beleg "+
				"trotzdem.", name, inbox.DoneDir, err))
			continue
		}
		moved = append(moved, Moved{From: p, To: to})
	}
	if len(moved) > 0 {
		t.Notify(ctx, core.ChangeInbox)
	}
	return moved, hints
}
