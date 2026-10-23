package oauth

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRedirectMatchIgnoresOnlyThePort(t *testing.T) {
	registered := []string{"http://127.0.0.1/callback", "http://localhost:3000/cb?x=1"}
	cases := map[string]bool{
		"http://127.0.0.1:53682/callback": true,
		"http://127.0.0.1/callback":       true,
		"http://localhost:9/cb?x=1":       true,
		"http://localhost:9/cb":           false,
		"http://localhost:9/cb?x=2":       false,
		"http://127.0.0.1:1/callback/":    false,
		"http://[::1]:1/callback":         false,
		"http://localhost:1/callback":     false,
		"https://127.0.0.1/callback":      false,
		"http://127.0.0.1.evil/callback":  false,
	}
	for uri, want := range cases {
		if got := matchRedirect(registered, uri); got != want {
			t.Errorf("%s: %v, erwartet %v", uri, got, want)
		}
	}
}

func TestNamesAreCleanedAndSlugsUnique(t *testing.T) {
	if got := sanitizeName("Claude\x00 Code\u202e\n" + strings.Repeat("x", 200)); strings.ContainsAny(got, "\x00\n\u202e") || len([]rune(got)) > maxNameRunes {
		t.Errorf("Name nicht bereinigt: %q", got)
	}
	if got := sanitizeName("  \t "); got != "Unbenannte App" {
		t.Errorf("leerer Name: %q", got)
	}
	for in, want := range map[string]string{
		"Claude Code":                        "claude-code",
		"Bürö-Assistent (Ä)":                 "bueroe-assistent-ae",
		"!!!":                                "agent",
		strings.Repeat("sehr-lang-", 10):     "sehr-lang-sehr-lang-sehr-lang-se",
		"Codex CLI 1.2 — mit Gedankenstrich": "codex-cli-1-2-mit-gedankenstrich",
	} {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, erwartet %q", in, got, want)
		}
	}

	store, err := OpenStore(filepath.Join(t.TempDir(), "verbindungen.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	var slugs []string
	for i := range 3 {
		id := newSecret("")
		if err := store.CreateClient(ctx, &Client{ID: id, Name: "Claude Code", RedirectURIs: "[]", CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
		family := Family{ID: newSecret(""), ClientID: id, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}
		client, err := store.approve(ctx, approval{ClientID: id, At: now, Family: family,
			Code: Code{Hash: hashSecret(newSecret("")), ClientID: id, RedirectURI: "x", Challenge: "y",
				FamilyID: family.ID, ExpiresAt: now.Add(time.Minute)}})
		if err != nil {
			t.Fatalf("Zustimmung %d: %v", i, err)
		}
		slugs = append(slugs, *client.Slug)
	}
	if slugs[0] != "claude-code" || slugs[1] != "claude-code-2" || slugs[2] != "claude-code-3" {
		t.Errorf("Kurznamen: %v", slugs)
	}
}

func TestUserCodeShape(t *testing.T) {
	for range 50 {
		code := newUserCode()
		if len(code) != 7 || code[3] != '-' {
			t.Fatalf("Abgleichcode %q", code)
		}
	}
}

func testPending(id, clientID string, at time.Time) *pending {
	return &pending{req: ConsentRequest{ID: id, ClientID: clientID}, done: make(chan struct{}), created: at}
}

// Die Anfragen bleiben wenige, gleich wie oft /authorize kommt: abgelöste
// sind sofort weg, erledigte nach keep, und mehr als maxEntries gibt es nicht.
func TestPendingSetStaysSmall(t *testing.T) {
	now := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	p := newPendingSet(3, time.Minute, func() time.Time { return now })

	for i := range 1000 {
		now = now.Add(clientInterval + time.Second)
		if _, err := p.add(testPending(fmt.Sprint("a", i), "schleife", now)); err != nil {
			t.Fatalf("Anfrage %d: %v", i, err)
		}
	}
	if len(p.items) != 1 || len(p.last) != 1 {
		t.Fatalf("%d Einträge, %d Merker nach 1000 Ablösungen", len(p.items), len(p.last))
	}

	// Eine reservierte Anfrage (die Zustimmung läuft) wird nicht abgelöst.
	claimed := p.get("a999")
	if !p.claim(claimed) {
		t.Fatal("Reservierung")
	}
	now = now.Add(clientInterval + time.Second)
	if _, err := p.add(testPending("b", "schleife", now)); err != nil {
		t.Fatal(err)
	}
	if isDone(claimed) || p.get("a999") == nil {
		t.Error("die reservierte Anfrage wurde abgelöst")
	}
	p.finish(claimed, true, "code", "", true)
	p.finish(p.get("b"), false, "", "abgelehnt", false)

	// Erledigt, aber nie abgeholt: nach keep weg.
	now = now.Add(2 * time.Minute)
	if _, err := p.add(testPending("c", "andere", now)); err != nil {
		t.Fatal(err)
	}
	if len(p.items) != 1 {
		t.Errorf("%d Einträge; erledigte bleiben über keep hinaus", len(p.items))
	}

	// Viele Apps, jede sofort abgelehnt und nie abgeholt: höchstens maxEntries.
	var err error
	for i := 0; i < 2*maxEntries && err == nil; i++ {
		item := testPending(fmt.Sprint("d", i), fmt.Sprint("app", i), now)
		if _, err = p.add(item); err == nil {
			p.finish(item, false, "", "abgelehnt", false)
		}
	}
	if !errors.Is(err, ErrTooManyRequests) || len(p.items) > maxEntries {
		t.Errorf("%d Einträge, Fehler %v", len(p.items), err)
	}
}

// Nach einer Ablehnung wartet die App den Mindestabstand ab, nach einer
// Zustimmung nicht.
func TestPendingSetSpacesRequestsOfOneApp(t *testing.T) {
	now := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	p := newPendingSet(3, time.Minute, func() time.Time { return now })
	first := testPending("1", "app", now)
	if _, err := p.add(first); err != nil {
		t.Fatal(err)
	}
	p.finish(first, false, "", "abgelehnt", false)
	if _, err := p.add(testPending("2", "app", now)); !errors.Is(err, ErrTooSoon) {
		t.Errorf("nach Ablehnung: %v", err)
	}
	now = now.Add(clientInterval)
	approved := testPending("3", "app", now)
	if _, err := p.add(approved); err != nil {
		t.Fatalf("nach dem Mindestabstand: %v", err)
	}
	p.finish(approved, true, "code", "", false)
	if _, err := p.add(testPending("4", "app", now)); err != nil {
		t.Errorf("nach Zustimmung: %v", err)
	}
}
