package mcpserver

import (
	"context"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/core/policy"
	"github.com/buchfink/buchfink/internal/inbox"
)

// --- eingang ----------------------------------------------------------------

type eingangIn struct {
	Seite
}

type eingangDatei struct {
	Pfad            string `json:"pfad" jsonschema:"so an beleg_ablegen oder kontoauszug_importieren übergeben"`
	Name            string `json:"name"`
	Groesse         int64  `json:"groesse" jsonschema:"Bytes"`
	GeaendertAm     string `json:"geaendert_am"`
	Art             string `json:"art" jsonschema:"beleg, e_rechnung, kontoauszug oder nicht_unterstuetzt"`
	BereitsAbgelegt *uint  `json:"bereits_abgelegt,omitempty" jsonschema:"Beleg-ID, wenn derselbe Inhalt schon im Archiv liegt (auch ein Kontoauszug)"`
}

type eingangOut struct {
	Eingang          string         `json:"eingang" jsonschema:"Pfad des Eingangs"`
	AeltesteVorTagen int            `json:"aelteste_vor_tagen" jsonschema:"Alter der ältesten Datei in Tagen"`
	Dateien          []eingangDatei `json:"dateien"`
	Seitenende
	NaechsterSchritt string `json:"naechster_schritt"`
}

// inboxState ist der Stand des Eingangs für status und aufgaben.
type inboxState struct {
	Pfad             string `json:"pfad"`
	Dateien          int    `json:"dateien"`
	AeltesteVorTagen int    `json:"aelteste_vor_tagen"`
	Hinweis          string `json:"hinweis,omitempty" jsonschema:"N Dateien im Eingang, sobald dort etwas liegt"`
	Fehler           string `json:"fehler,omitempty" jsonschema:"warum Buchfink den Eingang nicht lesen kann; die übrigen Werkzeuge arbeiten weiter"`
}

// inboxStateOf zählt die Dateien im Eingang. Ist er nicht lesbar, etwa weil
// das System der App den Zugriff auf den Ordner verweigert, nennt Fehler den
// Grund; status und aufgaben liefern ihren übrigen Stand trotzdem.
func inboxStateOf(dirs policy.Folders, now time.Time) inboxState {
	n, oldest, err := dirs.InboxCount()
	if err != nil {
		return inboxState{Pfad: dirs.Inbox, Fehler: "Der Eingang ist nicht lesbar: " + err.Error()}
	}
	st := inboxState{Pfad: dirs.Inbox, Dateien: n, AeltesteVorTagen: inbox.AgeDays(oldest, now)}
	if n > 0 {
		st.Hinweis = inboxCount(n) + " im Eingang"
	}
	return st
}

func inboxCount(n int) string {
	if n == 1 {
		return "1 Datei"
	}
	return fmt.Sprintf("%d Dateien", n)
}

// verschoben ist eine Datei, die nach der Ablage aus dem Eingang nach
// Übernommen/ ging.
type verschoben struct {
	Von  string `json:"von"`
	Nach string `json:"nach"`
}

// settleInbox verschiebt nach der Ablage, was im Eingang lag (policy.SettleInbox).
func settleInbox(ctx context.Context, c *Call, paths []string) ([]verschoben, []string) {
	var dirs policy.Folders
	if err := c.Tenant.Read(ctx, func(ctx context.Context, s *core.Services) (err error) {
		dirs, err = folders(ctx, c, s)
		return err
	}); err != nil {
		return []verschoben{}, []string{"Der Eingang ließ sich nicht aufräumen: " + err.Error()}
	}
	moved, hints := policy.SettleInbox(ctx, c.Tenant, dirs, paths, time.Now())
	out := make([]verschoben, 0, len(moved))
	for _, m := range moved {
		out = append(out, verschoben{Von: m.From, Nach: m.To})
	}
	return out, hints
}

func registerEingang(tb *Toolbox) {
	AddRead(tb, Spec{
		Name:  "eingang",
		Title: "Eingang",
		Description: "Die Dateien im Eingang des Mandanten, die älteste zuerst: Belege, E-Rechnungen und " +
			"Kontoauszüge, die der Mensch dort abgelegt hat, ohne Übernommen/. art sagt, wohin eine Datei geht " +
			"(beleg und e_rechnung an beleg_ablegen, kontoauszug an kontoauszug_importieren), bereits_abgelegt " +
			"nennt den Beleg, wenn derselbe Inhalt schon im Archiv liegt. Nur ablegen, wenn der Mensch es sagt; " +
			"nach der Ablage verschiebt Buchfink die Datei nach Übernommen/<JJJJ-MM>.",
	}, func(ctx context.Context, c *Call, s *core.Services, in eingangIn) (eingangOut, error) {
		out := eingangOut{Dateien: []eingangDatei{}}
		dirs, err := folders(ctx, c, s)
		if err != nil {
			return out, err
		}
		dirs.EnsureInbox()
		out.Eingang = dirs.Inbox
		files, err := dirs.InboxFiles()
		if err != nil {
			return out, fmt.Errorf("der Eingang %s ließ sich nicht lesen: %w", dirs.Inbox, err)
		}
		out.AeltesteVorTagen = inbox.AgeDays(inbox.Oldest(files), time.Now())
		page, end, err := Paginate(files, in.Seite)
		if err != nil {
			return out, err
		}
		out.Seitenende = end
		sums := []string{}
		for _, f := range page {
			if f.SHA256 != "" {
				sums = append(sums, f.SHA256)
			}
		}
		archived, err := s.Receipts.ReceiptIDsByFileHash(ctx, sums)
		if err != nil {
			return out, err
		}
		for _, f := range page {
			row := eingangDatei{Pfad: f.Path, Name: f.Name, Groesse: f.Size, GeaendertAm: f.ModTime.Format(time.RFC3339),
				Art: string(f.Kind)}
			if id, ok := archived[f.SHA256]; ok && f.SHA256 != "" {
				row.BereitsAbgelegt = &id
			}
			out.Dateien = append(out.Dateien, row)
		}
		switch {
		case out.Gesamt == 0:
			out.NaechsterSchritt = "Der Eingang ist leer. Neue Belege legt der Mensch dorthin oder zieht sie ins Buchfink-Fenster."
		default:
			out.NaechsterSchritt = "Auf Anweisung des Menschen: beleg_ablegen für beleg und e_rechnung, " +
				"kontoauszug_importieren für kontoauszug."
		}
		return out, nil
	})
}
