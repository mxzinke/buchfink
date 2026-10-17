package oauth

import (
	"context"
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
