package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/mcpserver/oauth"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

// Der ganze Weg eines Clients, wie die MCP-Spezifikation ihn vorgibt:
// 401 mit resource_metadata, Ressourcen-Metadaten, Server-Metadaten,
// Registrierung, Anfrage im Browser, Zustimmung in der App, Token, Aufruf.
func TestDiscoveryChainLikeAClient(t *testing.T) {
	h := newHarness(t)

	resp := h.do(http.MethodPost, "/mcp", strings.NewReader(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("ohne Token: %d, erwartet 401", resp.StatusCode)
	}
	challenge := resp.Header.Get("WWW-Authenticate")
	wantPRM := h.base + "/.well-known/oauth-protected-resource/mcp"
	if !strings.Contains(challenge, `resource_metadata="`+wantPRM+`"`) {
		t.Fatalf("WWW-Authenticate nennt die Metadaten nicht: %q", challenge)
	}

	for _, path := range []string{"/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/mcp"} {
		prm := decodeJSON(t, h.do(http.MethodGet, path, nil, nil))
		if prm["resource"] != h.base+"/mcp" {
			t.Errorf("%s: resource %v", path, prm["resource"])
		}
		if servers, _ := prm["authorization_servers"].([]any); len(servers) != 1 || servers[0] != h.base {
			t.Errorf("%s: authorization_servers %v", path, prm["authorization_servers"])
		}
	}

	asm := decodeJSON(t, h.do(http.MethodGet, "/.well-known/oauth-authorization-server", nil, nil))
	if asm["issuer"] != h.base {
		t.Errorf("issuer %v, erwartet %s", asm["issuer"], h.base)
	}
	for key, want := range map[string]string{
		"code_challenge_methods_supported":      "S256",
		"token_endpoint_auth_methods_supported": "none",
		"grant_types_supported":                 "refresh_token",
	} {
		values, _ := asm[key].([]any)
		if !slices.Contains(values, any(want)) {
			t.Errorf("%s ohne %s: %v", key, want, values)
		}
	}
	if methods, _ := asm["code_challenge_methods_supported"].([]any); len(methods) != 1 {
		t.Errorf("nur S256 darf angeboten werden: %v", methods)
	}
	if asm["authorization_response_iss_parameter_supported"] != true {
		t.Error("iss in der Autorisierungsantwort ist nicht angekündigt")
	}
	if asm["registration_endpoint"] != h.base+"/register" {
		t.Errorf("registration_endpoint %v", asm["registration_endpoint"])
	}
	if _, ok := asm["jwks_uri"]; ok {
		t.Error("ein leeres jwks_uri gehört nicht in die Metadaten")
	}

	clientID := h.registerClient("Claude Code")
	p := newPKCE()
	redirect := "http://localhost:49152/callback"
	loc := h.browse(h.authorizeQuery(clientID, redirect, p))
	if loc.Host != "localhost:49152" || loc.Path != "/callback" {
		t.Fatalf("Rücksprung an %s", loc)
	}
	if loc.Query().Get("state") != "zustand-123" || loc.Query().Get("iss") != h.base {
		t.Fatalf("state oder iss fehlen: %s", loc.RawQuery)
	}
	presented := h.broker.last()
	if presented.ClientName != "Claude Code" || presented.RedirectTo != "localhost:49152" ||
		presented.TenantID != h.tenant.ID || len(presented.UserCode) != 7 {
		t.Errorf("die App bekam eine unvollständige Anfrage: %+v", presented)
	}

	status, body := h.exchange(clientID, loc.Query().Get("code"), redirect, p.verifier)
	if status != http.StatusOK {
		t.Fatalf("Token: %d %v", status, body)
	}
	if body["token_type"] != "Bearer" || body["expires_in"] != float64(600) || body["refresh_token"] == "" {
		t.Errorf("Tokenantwort unvollständig: %v", body)
	}

	cs := h.mustSession(body["access_token"].(string))
	out := statusResult(t, callTool(t, cs, "status", map[string]any{}))
	if !out.Bereit || out.Mandant == nil || out.Mandant.ID != h.tenant.ID {
		t.Fatalf("status: %+v", out)
	}
	if out.Verbindung.Erfassungsweg != "agent:claude-code" || !out.Verbindung.MandantFreigegeben {
		t.Errorf("Verbindung: %+v", out.Verbindung)
	}
	if out.VerbundeneApps != 1 || len(out.Geschaeftsjahre) == 0 || out.Pruefermodus == nil || out.Pruefermodus.Aktiv {
		t.Errorf("status unvollständig: %+v", out)
	}
}

// Clients der Revision 2025-11-25 handeln mit initialize aus; neuere fragen
// zustandslos. Beide kommen an.
func TestLegacyAndCurrentProtocolVersions(t *testing.T) {
	h := newHarness(t)
	tok := h.login("Codex", "")
	for _, version := range []string{"2025-11-25", "2025-06-18", ""} {
		cs, err := h.session(tok.access, version)
		if err != nil {
			t.Fatalf("Fassung %q: %v", version, err)
		}
		got := cs.InitializeResult().ProtocolVersion
		if version != "" && got != version {
			t.Errorf("ausgehandelt %s, erwartet %s", got, version)
		}
		if version == "" && got != "2026-07-28" {
			t.Errorf("ohne Vorgabe ausgehandelt %s, erwartet 2026-07-28", got)
		}
		if out := statusResult(t, callTool(t, cs, "status", nil)); !out.Bereit {
			t.Errorf("Fassung %q: status nicht bereit", version)
		}
	}
}

// Der OAuth-Client des Go-SDK findet alles selbst und kommt durch.
func TestSDKAuthorizationHandlerGetsThrough(t *testing.T) {
	h := newHarness(t)
	handler, err := auth.NewAuthorizationCodeHandler(&auth.AuthorizationCodeHandlerConfig{
		DynamicClientRegistrationConfig: &auth.DynamicClientRegistrationConfig{
			Metadata: &oauthex.ClientRegistrationMetadata{
				ClientName: "SDK-Test", RedirectURIs: []string{"http://127.0.0.1/callback"},
				GrantTypes: []string{"authorization_code", "refresh_token"}, TokenEndpointAuthMethod: "none",
			},
		},
		RequestRefreshToken: true,
		AuthorizationCodeFetcher: func(ctx context.Context, args *auth.AuthorizationArgs) (*auth.AuthorizationResult, error) {
			u, err := url.Parse(args.URL)
			if err != nil {
				return nil, err
			}
			loc := h.browse(u.Query())
			q := loc.Query()
			return &auth.AuthorizationResult{Code: q.Get("code"), State: q.Get("state"), Iss: q.Get("iss")}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	transport := &mcp.StreamableClientTransport{Endpoint: h.base + "/mcp", OAuthHandler: handler, DisableStandaloneSSE: true}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "sdk"}, nil).Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Verbindung mit SDK-Anmeldung: %v", err)
	}
	defer cs.Close()
	if out := statusResult(t, callTool(t, cs, "status", nil)); out.Verbindung.Erfassungsweg != "agent:sdk-test" {
		t.Errorf("status: %+v", out)
	}
}

// Jede Erneuerung ersetzt das Refresh-Token. Wer ein ersetztes noch einmal
// einlöst, hat es gestohlen oder kopiert: die ganze Kette fällt.
func TestRefreshRotatesAndReuseRevokesTheChain(t *testing.T) {
	h := newHarness(t)
	tok := h.login("OpenCode", "")

	status, next := h.postForm(oauth.PathToken, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {tok.refresh}, "client_id": {tok.clientID},
	})
	if status != http.StatusOK || next["refresh_token"] == tok.refresh || next["access_token"] == tok.access {
		t.Fatalf("Erneuerung: %d %v", status, next)
	}
	if _, err := h.session(next["access_token"].(string), ""); err != nil {
		t.Fatalf("neues Access-Token: %v", err)
	}

	status, body := h.postForm(oauth.PathToken, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {tok.refresh}, "client_id": {tok.clientID},
	})
	if status != http.StatusBadRequest || body["error"] != "invalid_grant" {
		t.Fatalf("zweite Einlösung: %d %v", status, body)
	}
	status, body = h.postForm(oauth.PathToken, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {next["refresh_token"].(string)}, "client_id": {tok.clientID},
	})
	if status != http.StatusBadRequest || body["error"] != "invalid_grant" {
		t.Errorf("die Kette lebt nach der Wiederverwendung weiter: %d %v", status, body)
	}
	if _, err := h.session(next["access_token"].(string), ""); err == nil {
		t.Error("das Access-Token der widerrufenen Kette gilt noch")
	}
}

func TestRefreshMayNarrowButNotWidenScopes(t *testing.T) {
	h := newHarness(t)
	tok := h.login("Pi", "buchfink:lesen")
	status, body := h.postForm(oauth.PathToken, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {tok.refresh}, "client_id": {tok.clientID},
		"scope": {"buchfink:lesen buchfink:buchen"},
	})
	if status != http.StatusBadRequest || body["error"] != "invalid_scope" {
		t.Errorf("Erweiterung: %d %v", status, body)
	}
}

// Ein Code gilt einmal. Die zweite Einlösung widerruft, was die erste
// ausgegeben hat (RFC 6749 §4.1.2).
func TestCodeReuseRevokesIssuedTokens(t *testing.T) {
	h := newHarness(t)
	clientID := h.registerClient("Claude Code")
	p := newPKCE()
	redirect := "http://127.0.0.1:5000/callback"
	code := h.browse(h.authorizeQuery(clientID, redirect, p)).Query().Get("code")

	status, first := h.exchange(clientID, code, redirect, p.verifier)
	if status != http.StatusOK {
		t.Fatalf("erste Einlösung: %d %v", status, first)
	}
	status, second := h.exchange(clientID, code, redirect, p.verifier)
	if status != http.StatusBadRequest || second["error"] != "invalid_grant" {
		t.Fatalf("zweite Einlösung: %d %v", status, second)
	}
	if _, err := h.session(first["access_token"].(string), ""); err == nil {
		t.Error("das Access-Token aus dem wiederverwendeten Code gilt noch")
	}
}

func TestParallelRedemptionHasOneWinner(t *testing.T) {
	h := newHarness(t)
	clientID := h.registerClient("Claude Code")
	p := newPKCE()
	redirect := "http://127.0.0.1:5000/callback"
	code := h.browse(h.authorizeQuery(clientID, redirect, p)).Query().Get("code")

	var wg sync.WaitGroup
	var mu sync.Mutex
	wins := 0
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.PostForm(h.base+oauth.PathToken, url.Values{
				"grant_type": {"authorization_code"}, "client_id": {clientID}, "code": {code},
				"redirect_uri": {redirect}, "code_verifier": {p.verifier},
			})
			if err != nil {
				t.Error(err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Errorf("%d Einlösungen erfolgreich, erwartet genau eine", wins)
	}
}

// Rücksprung: Port frei, Pfad und Host genau. Ein fremder Rücksprung führt
// auf eine Fehlerseite und nicht dorthin.
func TestRedirectURIMatching(t *testing.T) {
	h := newHarness(t)
	clientID := h.registerClient("Claude Code")
	p := newPKCE()

	for _, bad := range []string{
		"http://127.0.0.1:5000/anders", "http://evil.example/callback", "http://192.168.1.5/callback",
		"http://127.0.0.1:5000/callback?x=1",
	} {
		resp := h.do(http.MethodGet, oauth.PathAuthorize+"?"+h.authorizeQuery(clientID, bad, p).Encode(), nil, nil)
		if resp.StatusCode != http.StatusBadRequest || resp.Header.Get("Location") != "" {
			t.Errorf("%s: %d, Location %q", bad, resp.StatusCode, resp.Header.Get("Location"))
		}
	}
	resp := h.do(http.MethodGet, oauth.PathAuthorize+"?"+h.authorizeQuery("unbekannt", "http://127.0.0.1/callback", p).Encode(), nil, nil)
	if resp.StatusCode != http.StatusBadRequest || resp.Header.Get("Location") != "" {
		t.Errorf("unbekannte App: %d", resp.StatusCode)
	}

	redirect := "http://127.0.0.1:61000/callback"
	code := h.browse(h.authorizeQuery(clientID, redirect, p)).Query().Get("code")
	status, body := h.exchange(clientID, code, "http://127.0.0.1:61001/callback", p.verifier)
	if status != http.StatusBadRequest || body["error"] != "invalid_grant" {
		t.Errorf("Token mit anderem redirect_uri: %d %v", status, body)
	}
	if status, _ := h.exchange(clientID, code, redirect, p.verifier); status != http.StatusOK {
		t.Errorf("ein gescheiterter Versuch darf den Code nicht verbrauchen: %d", status)
	}
}

func TestPKCEIsMandatory(t *testing.T) {
	h := newHarness(t)
	clientID := h.registerClient("Claude Code")
	p := newPKCE()
	redirect := "http://127.0.0.1:5000/callback"

	cases := map[string]func(url.Values){
		"ohne Challenge": func(q url.Values) { q.Del("code_challenge") },
		"plain":          func(q url.Values) { q.Set("code_challenge_method", "plain") },
		"ohne Methode":   func(q url.Values) { q.Del("code_challenge_method") },
		"zu kurz":        func(q url.Values) { q.Set("code_challenge", "kurz") },
	}
	for name, mutate := range cases {
		q := h.authorizeQuery(clientID, redirect, p)
		mutate(q)
		resp := h.do(http.MethodGet, oauth.PathAuthorize+"?"+q.Encode(), nil, nil)
		loc, _ := url.Parse(resp.Header.Get("Location"))
		if resp.StatusCode != http.StatusFound || loc.Query().Get("error") != "invalid_request" {
			t.Errorf("%s: %d %s", name, resp.StatusCode, resp.Header.Get("Location"))
		}
	}

	code := h.browse(h.authorizeQuery(clientID, redirect, p)).Query().Get("code")
	if status, body := h.exchange(clientID, code, redirect, newPKCE().verifier); body["error"] != "invalid_grant" {
		t.Errorf("falscher Verifier: %d %v", status, body)
	}
	if status, body := h.exchange(clientID, code, redirect, ""); body["error"] != "invalid_grant" {
		t.Errorf("ohne Verifier: %d %v", status, body)
	}
}

// Ein Token gilt nur für die Adresse, für die es ausgestellt wurde, und nur
// für diesen Server.
func TestAudienceIsBoundAndChecked(t *testing.T) {
	h := newHarness(t)
	clientID := h.registerClient("Claude Code")
	p := newPKCE()
	redirect := "http://127.0.0.1:5000/callback"

	for _, resource := range []string{"http://evil.example/mcp", "http://127.0.0.1:1/mcp", h.base + "/anders"} {
		q := h.authorizeQuery(clientID, redirect, p)
		q.Set("resource", resource)
		resp := h.do(http.MethodGet, oauth.PathAuthorize+"?"+q.Encode(), nil, nil)
		loc, _ := url.Parse(resp.Header.Get("Location"))
		if loc == nil || loc.Query().Get("error") != "invalid_target" {
			t.Errorf("resource %s: %d %s", resource, resp.StatusCode, resp.Header.Get("Location"))
		}
	}

	code := h.browse(h.authorizeQuery(clientID, redirect, p)).Query().Get("code")
	status, body := h.postForm(oauth.PathToken, url.Values{
		"grant_type": {"authorization_code"}, "client_id": {clientID}, "code": {code},
		"redirect_uri": {redirect}, "code_verifier": {p.verifier},
		"resource": {fmt.Sprintf("http://localhost:%d/mcp", h.port)},
	})
	if status != http.StatusBadRequest || body["error"] != "invalid_target" {
		t.Errorf("Token für eine andere Schreibweise: %d %v", status, body)
	}
	status, body = h.exchange(clientID, code, redirect, p.verifier)
	if status != http.StatusOK {
		t.Fatalf("Token: %d %v", status, body)
	}
	access := body["access_token"].(string)
	resp := h.do(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`), map[string]string{
		"Authorization": "Bearer " + access, "Content-Type": "application/json",
		"Accept": "application/json, text/event-stream", "Host": fmt.Sprintf("localhost:%d", h.port),
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Token für 127.0.0.1 gilt unter localhost: %d", resp.StatusCode)
	}
	if !strings.Contains(resp.Header.Get("WWW-Authenticate"), fmt.Sprintf("http://localhost:%d/.well-known/oauth-protected-resource/mcp", h.port)) {
		t.Errorf("WWW-Authenticate nennt nicht die angefragte Schreibweise: %q", resp.Header.Get("WWW-Authenticate"))
	}
}

func TestExpiredAccessTokenIsRefused(t *testing.T) {
	h := newHarness(t)
	tok := h.login("Claude Code", "")
	h.clock.Advance(11 * time.Minute)
	if _, err := h.session(tok.access, ""); err == nil {
		t.Error("ein abgelaufenes Access-Token gilt noch")
	}
}

// Eine Webseite im Browser erreicht 127.0.0.1. Sie darf weder über einen
// fremden Namen (DNS-Rebinding) noch mit ihrem Origin etwas auslösen.
func TestHostAndOriginAttacksAreRefused(t *testing.T) {
	h := newHarness(t)
	tok := h.login("Claude Code", "")
	paths := []struct{ method, path string }{
		{http.MethodGet, "/.well-known/oauth-protected-resource"},
		{http.MethodGet, "/.well-known/oauth-authorization-server"},
		{http.MethodPost, "/register"}, {http.MethodGet, "/authorize"},
		{http.MethodPost, "/token"}, {http.MethodPost, "/revoke"}, {http.MethodPost, "/mcp"},
	}
	for _, p := range paths {
		resp := h.do(p.method, p.path, nil, map[string]string{"Host": "evil.example:" + fmt.Sprint(h.port)})
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Host evil.example %s %s: %d", p.method, p.path, resp.StatusCode)
		}
		resp = h.do(p.method, p.path, nil, map[string]string{"Host": "127.0.0.1:1"})
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("fremder Port %s %s: %d", p.method, p.path, resp.StatusCode)
		}
	}
	for _, p := range paths[2:] {
		resp := h.do(p.method, p.path, nil, map[string]string{"Origin": "https://evil.example", "Authorization": "Bearer " + tok.access})
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Origin %s %s: %d", p.method, p.path, resp.StatusCode)
		}
		if resp.Header.Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("%s %s antwortet mit CORS", p.method, p.path)
		}
		resp = h.do(p.method, p.path, nil, map[string]string{"Sec-Fetch-Site": "cross-site"})
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Sec-Fetch-Site cross-site %s %s: %d", p.method, p.path, resp.StatusCode)
		}
	}
	for _, path := range []string{"/mcp", "/token"} {
		resp := h.do(http.MethodOptions, path, nil, map[string]string{
			"Origin": "https://evil.example", "Access-Control-Request-Method": "POST"})
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent ||
			resp.Header.Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("Preflight %s: %d %q", path, resp.StatusCode, resp.Header.Get("Access-Control-Allow-Origin"))
		}
	}
	resp := h.do(http.MethodGet, "/.well-known/oauth-authorization-server", nil, map[string]string{"Origin": "https://inspector.example"})
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Metadaten sind öffentlich: %d", resp.StatusCode)
	}
}

// Es gibt genau diese Endpunkte; keiner davon erteilt eine Zustimmung.
func TestRoutesAreFixed(t *testing.T) {
	h := newHarness(t)
	tok := h.login("Claude Code", "")
	cases := []struct {
		method, path string
		want         int
	}{
		{http.MethodPost, "/authorize?approve=1", http.StatusMethodNotAllowed},
		{http.MethodPost, "/authorize/warten", http.StatusMethodNotAllowed},
		{http.MethodGet, "/token", http.StatusMethodNotAllowed},
		{http.MethodGet, "/register", http.StatusMethodNotAllowed},
		{http.MethodGet, "/revoke", http.StatusMethodNotAllowed},
		{http.MethodPost, "/.well-known/oauth-authorization-server", http.StatusMethodNotAllowed},
		{http.MethodGet, "/", http.StatusNotFound},
		{http.MethodGet, "/consent", http.StatusNotFound},
		{http.MethodPost, "/resolve", http.StatusNotFound},
		{http.MethodGet, "/.well-known/openid-configuration", http.StatusNotFound},
	}
	for _, c := range cases {
		if resp := h.do(c.method, c.path, nil, nil); resp.StatusCode != c.want {
			t.Errorf("%s %s: %d, erwartet %d", c.method, c.path, resp.StatusCode, c.want)
		}
	}
	resp := h.do(http.MethodGet, "/mcp", nil, map[string]string{"Authorization": "Bearer " + tok.access, "Accept": "text/event-stream"})
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET /mcp im zustandslosen Betrieb: %d", resp.StatusCode)
	}
}

func TestDeniedConsentReturnsAccessDenied(t *testing.T) {
	h := newHarness(t)
	h.broker.setDecision(ptr(false))
	clientID := h.registerClient("Claude Code")
	loc := h.browse(h.authorizeQuery(clientID, "http://127.0.0.1:5000/callback", newPKCE()))
	q := loc.Query()
	if q.Get("error") != "access_denied" || q.Get("code") != "" || q.Get("state") != "zustand-123" || q.Get("iss") != h.base {
		t.Fatalf("Ablehnung: %s", loc.RawQuery)
	}
	if apps, _ := h.srv.ConnectedApps(context.Background()); len(apps) != 0 {
		t.Errorf("nach einer Ablehnung steht eine App in der Liste: %+v", apps)
	}
}

func TestConsentExpiresWithoutDecision(t *testing.T) {
	h := newHarness(t, withAuth(func(c *oauth.Config) { c.ConsentTTL = 300 * time.Millisecond }))
	h.broker.setDecision(nil)
	clientID := h.registerClient("Claude Code")
	loc := h.browse(h.authorizeQuery(clientID, "http://127.0.0.1:5000/callback", newPKCE()))
	if loc.Query().Get("error") != "access_denied" {
		t.Fatalf("Ablauf: %s", loc.RawQuery)
	}
	if !slices.Contains(h.broker.withdrawn, h.broker.last().ID) {
		t.Error("die abgelaufene Anfrage bleibt in der App stehen")
	}
}

// Zugestimmt wird mit der Nonce aus der Anfrage, genau einmal.
func TestResolveNeedsTheNonceOnce(t *testing.T) {
	h := newHarness(t)
	h.broker.setDecision(nil)
	clientID := h.registerClient("Claude Code")
	resp := h.do(http.MethodGet, oauth.PathAuthorize+"?"+h.authorizeQuery(clientID, "http://127.0.0.1:5000/callback", newPKCE()).Encode(), nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Anfrage: %d", resp.StatusCode)
	}
	req := h.broker.last()
	ctx := context.Background()
	if err := h.srv.Resolve(ctx, req.ID, "falsch", true); err == nil {
		t.Error("falsche Nonce angenommen")
	}
	if len(h.srv.PendingRequests()) != 1 {
		t.Fatal("eine falsche Nonce darf die Anfrage nicht verändern")
	}
	if err := h.srv.Resolve(ctx, req.ID, req.Nonce, true); err != nil {
		t.Fatalf("Zustimmung: %v", err)
	}
	if err := h.srv.Resolve(ctx, req.ID, req.Nonce, false); err == nil {
		t.Error("eine Anfrage ließ sich zweimal entscheiden")
	}
	if err := h.srv.Resolve(ctx, "unbekannt", req.Nonce, true); err == nil {
		t.Error("eine unbekannte Anfrage ließ sich entscheiden")
	}
}

func TestOpenRequestsAreLimited(t *testing.T) {
	h := newHarness(t)
	h.broker.setDecision(nil)
	for i := range 4 {
		clientID := h.registerClient(fmt.Sprintf("App %d", i))
		resp := h.do(http.MethodGet, oauth.PathAuthorize+"?"+h.authorizeQuery(clientID, "http://127.0.0.1:5000/callback", newPKCE()).Encode(), nil, nil)
		want := http.StatusOK
		if i == 3 {
			want = http.StatusTooManyRequests
		}
		if resp.StatusCode != want {
			t.Errorf("Anfrage %d: %d, erwartet %d", i+1, resp.StatusCode, want)
		}
	}
	// Dieselbe App ersetzt ihre offene Anfrage, statt eine weitere zu stellen.
	if n := len(h.srv.PendingRequests()); n != 3 {
		t.Errorf("%d offene Anfragen, erwartet 3", n)
	}
}

// Wer den Zugriff entzieht, sperrt die App sofort aus, auch mit gültigem Token.
func TestRevokedAppIsLockedOut(t *testing.T) {
	h := newHarness(t)
	tok := h.login("Claude Code", "")
	if _, err := h.session(tok.access, ""); err != nil {
		t.Fatal(err)
	}
	apps, err := h.srv.ConnectedApps(context.Background())
	if err != nil || len(apps) != 1 || apps[0].Revoked || len(apps[0].Tenants) != 1 {
		t.Fatalf("verbundene Apps: %+v %v", apps, err)
	}
	if err := h.srv.RevokeApp(context.Background(), tok.clientID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.session(tok.access, ""); err == nil {
		t.Error("das Access-Token gilt nach dem Entzug weiter")
	}
	status, body := h.postForm(oauth.PathToken, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {tok.refresh}, "client_id": {tok.clientID},
	})
	if status != http.StatusBadRequest || body["error"] != "invalid_grant" {
		t.Errorf("Erneuerung nach dem Entzug: %d %v", status, body)
	}
	apps, _ = h.srv.ConnectedApps(context.Background())
	if len(apps) != 1 || !apps[0].Revoked {
		t.Errorf("die App steht nicht als entzogen in der Liste: %+v", apps)
	}
}

func TestRevocationEndpoint(t *testing.T) {
	h := newHarness(t)
	tok := h.login("Claude Code", "")
	if status, _ := h.postForm(oauth.PathRevoke, url.Values{"token": {"bfr_unbekannt"}}); status != http.StatusOK {
		t.Errorf("unbekanntes Token: %d", status)
	}
	if status, _ := h.postForm(oauth.PathRevoke, url.Values{"token": {tok.refresh}, "client_id": {tok.clientID}}); status != http.StatusOK {
		t.Fatalf("Widerruf: %d", status)
	}
	if _, err := h.session(tok.access, ""); err == nil {
		t.Error("der Widerruf des Refresh-Tokens lässt das Access-Token seiner Kette gelten")
	}
}

// Ein Agent schreibt nie in einen Mandanten, zu dem der Mensch inzwischen
// gewechselt hat, und nie in einen, für den die App nicht zugelassen ist.
func TestTenantSwitchIsDetected(t *testing.T) {
	h := newHarness(t)
	tok := h.login("Claude Code", "")
	cs := h.mustSession(tok.access)
	first := h.tenant.ID
	if res := callTool(t, cs, "test_lieferant", map[string]any{"mandant_id": first, "name": "Büromarkt GmbH"}); res.IsError {
		t.Fatalf("Schreiben im zugelassenen Mandanten: %s", resultText(res))
	}

	second := h.createTenant("Zweite GmbH")
	out := statusResult(t, callTool(t, cs, "status", nil))
	if out.Bereit || out.Verbindung.MandantFreigegeben || out.Mandant != nil {
		t.Errorf("status nach dem Wechsel: %+v", out)
	}
	if e := toolErrorOf(t, callTool(t, cs, "test_lieferant", map[string]any{"mandant_id": first, "name": "X"})); e.Code != CodeTenantSwitched {
		t.Errorf("alter Mandant: %+v", e)
	}
	if e := toolErrorOf(t, callTool(t, cs, "test_lieferant", map[string]any{"mandant_id": second.ID, "name": "X"})); e.Code != CodeTenantNotGranted {
		t.Errorf("nicht zugelassener Mandant: %+v", e)
	}
	if n := h.countVendors("X"); n != 0 {
		t.Errorf("%d Lieferanten im falschen Mandanten angelegt", n)
	}
}

// Die Rechte gelten je Mandant. Ein Token hängt an der App; schreibt es in
// einem Mandanten, für den die App nur lesen darf, wird nichts geschrieben.
// Eine neue Zulassung mit weniger Rechten gilt auch für ältere Tokens.
func TestScopesFollowTheGrantOfTheOpenTenant(t *testing.T) {
	h := newHarness(t)
	clientID := h.registerClient("Claude Code")
	writer := h.authorize(clientID, "buchfink:lesen buchfink:buchen")

	second := h.createTenant("Nur-Lesen GmbH")
	h.authorize(clientID, "buchfink:lesen")
	cs := h.mustSession(writer.access)
	write := func(name string) *mcp.CallToolResult {
		return callTool(t, cs, "test_lieferant", map[string]any{"mandant_id": second.ID, "name": name})
	}
	if e := toolErrorOf(t, write("Fremd GmbH")); e.Code != CodeMissingScope {
		t.Errorf("Token aus dem ersten Mandanten schreibt im zweiten: %+v", e)
	}
	if n := h.countVendors("Fremd GmbH"); n != 0 {
		t.Errorf("%d Lieferanten trotz Nur-Lesen angelegt", n)
	}
	out := statusResult(t, callTool(t, cs, "status", nil))
	if !out.Bereit || !slices.Equal(out.Verbindung.Rechte, []string{"buchfink:lesen"}) {
		t.Errorf("status nennt nicht die Rechte des Mandanten: %+v", out.Verbindung)
	}

	h.authorize(clientID, "buchfink:lesen buchfink:buchen")
	if res := write("Erlaubt GmbH"); res.IsError {
		t.Fatalf("nach der Zulassung zum Buchen: %s", resultText(res))
	}
	h.authorize(clientID, "buchfink:lesen")
	if e := toolErrorOf(t, write("Zurückgenommen GmbH")); e.Code != CodeMissingScope {
		t.Errorf("ältere Kette behält das Buchen nach neuer Zulassung zum Lesen: %+v", e)
	}
}

func TestExaminerModeBlocksAgentWrites(t *testing.T) {
	h := newHarness(t)
	tok := h.login("Claude Code", "")
	cs := h.mustSession(tok.access)
	until := time.Now().AddDate(0, 0, 10).Format("2006-01-02")
	if _, err := h.host.EnableExaminer(uiCtx, until, "Betriebsprüfung 2024"); err != nil {
		t.Fatal(err)
	}
	out := statusResult(t, callTool(t, cs, "status", nil))
	if out.Pruefermodus == nil || !out.Pruefermodus.Aktiv || out.Pruefermodus.Bis != until || !strings.Contains(out.Hinweis, "Prüfermodus") {
		t.Errorf("status im Prüfermodus: %+v", out)
	}
	e := toolErrorOf(t, callTool(t, cs, "test_lieferant", map[string]any{"mandant_id": h.tenant.ID, "name": "Gesperrt GmbH"}))
	if e.Code != CodeExaminer || e.NaechsterSchritt == "" {
		t.Errorf("Schreiben im Prüfermodus: %+v", e)
	}
	if n := h.countVendors("Gesperrt GmbH"); n != 0 {
		t.Error("im Prüfermodus wurde geschrieben")
	}
}

// Derselbe Schlüssel mit derselben Eingabe schreibt nicht zweimal; mit einer
// anderen Eingabe ist er ein Fehler.
func TestIdempotencyKey(t *testing.T) {
	h := newHarness(t)
	cs := h.mustSession(h.login("Claude Code", "").access)
	args := map[string]any{"mandant_id": h.tenant.ID, "name": "Einmal GmbH", "idempotenz_schluessel": "lieferant-0001"}

	first := callTool(t, cs, "test_lieferant", args)
	second := callTool(t, cs, "test_lieferant", args)
	var a, b testContactOut
	_ = json.Unmarshal([]byte(resultText(first)), &a)
	_ = json.Unmarshal([]byte(resultText(second)), &b)
	if first.IsError || second.IsError || a.ID == 0 || a.ID != b.ID || a.Wiederholt || !b.Wiederholt {
		t.Fatalf("Wiederholung: %s / %s", resultText(first), resultText(second))
	}
	if n := h.countVendors("Einmal GmbH"); n != 1 {
		t.Errorf("%d Lieferanten, erwartet einer", n)
	}
	args["name"] = "Anders GmbH"
	if e := toolErrorOf(t, callTool(t, cs, "test_lieferant", args)); e.Code != CodeKeyReused {
		t.Errorf("anderer Inhalt unter demselben Schlüssel: %+v", e)
	}
}

func TestReadOnlyTokenCannotWrite(t *testing.T) {
	h := newHarness(t)
	tok := h.login("Claude Code", "buchfink:lesen")
	if tok.scope != "buchfink:lesen" {
		t.Fatalf("Scope %q", tok.scope)
	}
	cs := h.mustSession(tok.access)
	if e := toolErrorOf(t, callTool(t, cs, "test_lieferant", map[string]any{"mandant_id": h.tenant.ID, "name": "Nein GmbH"})); e.Code != CodeMissingScope {
		t.Errorf("Schreiben mit Lese-Token: %+v", e)
	}
}

func TestStatusWithoutOpenTenant(t *testing.T) {
	h := newHarness(t, withoutTenant())
	tok := h.login("Claude Code", "")
	out := statusResult(t, callTool(t, h.mustSession(tok.access), "status", nil))
	if out.Bereit || out.Mandant != nil || !strings.Contains(out.Hinweis, "kein Mandant") || out.NaechsterSchritt == "" {
		t.Errorf("status ohne Mandant: %+v", out)
	}
	if out.Version == "" {
		t.Error("status ohne Fassung")
	}
}

func TestToolsCarryAnnotationsAndSchemas(t *testing.T) {
	h := newHarness(t)
	cs := h.mustSession(h.login("Claude Code", "").access)
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]*mcp.Tool{}
	for _, tool := range res.Tools {
		byName[tool.Name] = tool
	}
	status := byName["status"]
	if status == nil || status.Annotations == nil || !status.Annotations.ReadOnlyHint || status.OutputSchema == nil || status.Title == "" {
		t.Fatalf("status: %+v", status)
	}
	write := byName["test_lieferant"]
	if write == nil || write.Annotations.ReadOnlyHint || write.Annotations.DestructiveHint == nil {
		t.Fatalf("schreibendes Werkzeug: %+v", write)
	}
	schema, _ := json.Marshal(write.InputSchema)
	if !strings.Contains(string(schema), `"required":["mandant_id","name"]`) {
		t.Errorf("mandant_id ist keine Pflicht: %s", schema)
	}
}

// Clients prüfen structuredContent gegen das Ausgabeschema (Claude Code
// tut es). Jede Lage von status muss deshalb zu seinem Schema passen.
func TestOutputsMatchTheirSchemas(t *testing.T) {
	h := newHarness(t)
	cs := h.mustSession(h.login("Claude Code", "").access)
	listed, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	schemas := map[string]*jsonschema.Resolved{}
	for _, tool := range listed.Tools {
		raw, _ := json.Marshal(tool.OutputSchema)
		var s jsonschema.Schema
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Fatalf("%s: %v", tool.Name, err)
		}
		resolved, err := s.Resolve(nil)
		if err != nil {
			t.Fatalf("%s: %v", tool.Name, err)
		}
		schemas[tool.Name] = resolved
	}
	check := func(t *testing.T, situation, tool string, res *mcp.CallToolResult) {
		t.Helper()
		if res.IsError {
			t.Fatalf("%s: %s", situation, resultText(res))
		}
		var instance any
		raw, _ := json.Marshal(res.StructuredContent)
		_ = json.Unmarshal(raw, &instance)
		if err := schemas[tool].Validate(instance); err != nil {
			t.Errorf("%s: Ausgabe passt nicht zum Schema: %v\n%s", situation, err, raw)
		}
	}

	check(t, "bereit", "status", callTool(t, cs, "status", nil))
	check(t, "geschrieben", "test_lieferant", callTool(t, cs, "test_lieferant",
		map[string]any{"mandant_id": h.tenant.ID, "name": "Schema GmbH", "idempotenz_schluessel": "schema-0001"}))
	check(t, "wiederholt", "test_lieferant", callTool(t, cs, "test_lieferant",
		map[string]any{"mandant_id": h.tenant.ID, "name": "Schema GmbH", "idempotenz_schluessel": "schema-0001"}))
	until := time.Now().AddDate(0, 0, 3).Format("2006-01-02")
	if _, err := h.host.EnableExaminer(uiCtx, until, "Prüfung"); err != nil {
		t.Fatal(err)
	}
	check(t, "Prüfermodus", "status", callTool(t, cs, "status", nil))
	h.createTenant("Andere GmbH")
	check(t, "nicht freigegeben", "status", callTool(t, cs, "status", nil))

	t.Run("ohne Mandant", func(t *testing.T) {
		empty := newHarness(t, withoutTenant())
		ecs := empty.mustSession(empty.login("Claude Code", "").access)
		check(t, "ohne Mandant", "status", callTool(t, ecs, "status", nil))
	})
}

// In der Datei der verbundenen Apps steht kein Code und kein Token im
// Klartext.
func TestStoreHoldsNoPlaintextSecrets(t *testing.T) {
	h := newHarness(t)
	clientID := h.registerClient("Claude Code")
	p := newPKCE()
	redirect := "http://127.0.0.1:5000/callback"
	code := h.browse(h.authorizeQuery(clientID, redirect, p)).Query().Get("code")
	_, body := h.exchange(clientID, code, redirect, p.verifier)
	secrets := []string{code, body["access_token"].(string), body["refresh_token"].(string)}

	dir := filepath.Dir(h.srv.opts.StorePath)
	files, _ := filepath.Glob(filepath.Join(dir, StoreFile+"*"))
	if len(files) == 0 {
		t.Fatal("keine Datei der verbundenen Apps gefunden")
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range secrets {
			if bytes.Contains(data, []byte(s)) || bytes.Contains(data, []byte(strings.SplitN(s, "_", 2)[1])) {
				t.Errorf("%s enthält ein Geheimnis im Klartext", filepath.Base(f))
			}
		}
	}
}

func TestRegistrationOnlyForLoopbackPublicClients(t *testing.T) {
	h := newHarness(t)
	for _, ok := range []string{"http://127.0.0.1:8080/cb", "http://127.0.0.1/cb", "http://localhost:3000/callback", "http://[::1]:9000/cb"} {
		if status, body := h.register("App", ok); status != http.StatusCreated {
			t.Errorf("%s: %d %v", ok, status, body)
		}
	}
	for _, bad := range []string{"https://evil.example/cb", "http://192.168.1.5/cb", "myapp://cb",
		"http://127.0.0.1/cb#frag", "http://nutzer@127.0.0.1/cb", "http://127.0.0.1.evil.example/cb"} {
		if status, body := h.register("App", bad); status != http.StatusBadRequest || body["error"] != "invalid_redirect_uri" {
			t.Errorf("%s: %d %v", bad, status, body)
		}
	}
	six := []string{}
	for i := range 6 {
		six = append(six, fmt.Sprintf("http://127.0.0.1/cb%d", i))
	}
	if status, _ := h.register("App", six...); status != http.StatusBadRequest {
		t.Errorf("sechs Rücksprünge: %d", status)
	}
	for name, body := range map[string]string{
		"Secret":   `{"redirect_uris":["http://127.0.0.1/cb"],"token_endpoint_auth_method":"client_secret_basic"}`,
		"Implicit": `{"redirect_uris":["http://127.0.0.1/cb"],"grant_types":["implicit"]}`,
		"zu groß":  `{"redirect_uris":["http://127.0.0.1/cb"],"client_name":"` + strings.Repeat("x", 70<<10) + `"}`,
	} {
		resp := h.do(http.MethodPost, oauth.PathRegister, strings.NewReader(body), map[string]string{"Content-Type": "application/json"})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: %d", name, resp.StatusCode)
		}
	}
}

func TestRegistrationIsRateLimited(t *testing.T) {
	h := newHarness(t)
	for i := range 10 {
		if status, _ := h.register("App", "http://127.0.0.1/cb"); status != http.StatusCreated {
			t.Fatalf("Registrierung %d: %d", i+1, status)
		}
	}
	if status, _ := h.register("App", "http://127.0.0.1/cb"); status != http.StatusTooManyRequests {
		t.Errorf("elfte Registrierung in einer Minute: %d", status)
	}
	h.clock.Advance(time.Minute)
	if status, _ := h.register("App", "http://127.0.0.1/cb"); status != http.StatusCreated {
		t.Errorf("nach einer Minute: %d", status)
	}
}

func TestPaginate(t *testing.T) {
	items := make([]int, 60)
	for i := range items {
		items[i] = i
	}
	page, end, err := Paginate(items, Seite{})
	if err != nil || len(page) != 25 || end.NaechsteSeite == nil {
		t.Fatalf("erste Seite: %d %v %v", len(page), end, err)
	}
	page, end, err = Paginate(items, Seite{Limit: 50, Cursor: *end.NaechsteSeite})
	if err != nil || len(page) != 35 || page[0] != 25 || end.NaechsteSeite != nil {
		t.Fatalf("letzte Seite: %d %v %v", len(page), end, err)
	}
	for _, bad := range []Seite{{Limit: 101}, {Limit: -1}, {Cursor: "kaputt"}} {
		if _, _, err := Paginate(items, bad); err == nil {
			t.Errorf("%+v angenommen", bad)
		}
	}
}

// Ein belegter Port wird gemeldet, nicht umgangen.
func TestBusyPortIsReportedNotAvoided(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	if err := h.srv.Restart(ctx, h.port); err == nil || !strings.Contains(err.Error(), "belegt") {
		t.Fatalf("belegter Port: %v", err)
	}
	st := h.srv.Status()
	if st.Running || st.Port != h.port || !strings.Contains(st.Error, "belegt") {
		t.Errorf("Status: %+v", st)
	}
}
