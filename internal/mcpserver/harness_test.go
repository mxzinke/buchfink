package mcpserver

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/mcpserver/oauth"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Die Tests sprechen den Server so an wie ein Client: über HTTP, mit
// Discovery, Registrierung, Browser-Schritten und Token. Statt des Fensters
// entscheidet ein Fake-Broker; statt der echten Uhr eine, die sich vorstellen
// lässt.

var uiCtx = actor.WithChannel(context.Background(), actor.ChannelUI)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

// fakeBroker entscheidet wie der Mensch im Fenster. decide nil heißt: es
// entscheidet niemand von selbst.
type fakeBroker struct {
	mu        sync.Mutex
	srv       func() *Server
	decide    func(oauth.ConsentRequest) *bool
	presented []oauth.ConsentRequest
	withdrawn []string
}

func (b *fakeBroker) Present(r oauth.ConsentRequest) {
	b.mu.Lock()
	b.presented = append(b.presented, r)
	decide := b.decide
	b.mu.Unlock()
	if decide == nil {
		return
	}
	if approve := decide(r); approve != nil {
		go func() { _ = b.srv().Resolve(context.Background(), r.ID, r.Nonce, *approve) }()
	}
}

func (b *fakeBroker) Withdraw(id string) {
	b.mu.Lock()
	b.withdrawn = append(b.withdrawn, id)
	b.mu.Unlock()
}

func (b *fakeBroker) setDecision(approve *bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if approve == nil {
		b.decide = nil
		return
	}
	value := *approve
	b.decide = func(oauth.ConsentRequest) *bool { return &value }
}

func (b *fakeBroker) last() oauth.ConsentRequest {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.presented) == 0 {
		return oauth.ConsentRequest{}
	}
	return b.presented[len(b.presented)-1]
}

func ptr[T any](v T) *T { return &v }

type harness struct {
	t      *testing.T
	host   *core.Host
	srv    *Server
	broker *fakeBroker
	clock  *fakeClock
	base   string
	port   int
	tenant *domain.TenantConfig
	http   *http.Client
}

type harnessOption func(*harnessConfig)

type harnessConfig struct {
	noTenant bool
	auth     func(*oauth.Config)
}

func withoutTenant() harnessOption { return func(c *harnessConfig) { c.noTenant = true } }

func withAuth(f func(*oauth.Config)) harnessOption {
	return func(c *harnessConfig) { c.auth = f }
}

func newHarness(t *testing.T, opts ...harnessOption) *harness {
	t.Helper()
	// Jeder Test hat Host, Mandant, Datei und Port für sich.
	t.Parallel()
	var hc harnessConfig
	for _, o := range opts {
		o(&hc)
	}
	base := t.TempDir()
	host, err := core.NewHost(context.Background(), core.HostOptions{
		Config:  repository.NewAppConfigRepository(filepath.Join(base, "config")),
		BaseDir: base,
		Network: core.OfflineNetwork(),
	})
	if err != nil {
		t.Fatalf("Host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close(context.Background()) })

	h := &harness{t: t, host: host, clock: &fakeClock{now: time.Now()}}
	if !hc.noTenant {
		h.tenant = h.createTenant("Pfennig Ventures GmbH")
	}
	h.broker = &fakeBroker{srv: func() *Server { return h.srv }}
	h.broker.setDecision(ptr(true))
	h.srv, err = New(Options{
		Host: host, StorePath: filepath.Join(base, StoreFile), Broker: h.broker, Clock: h.clock.Now,
		Auth: func(c *oauth.Config) {
			c.LongPoll = 2 * time.Second
			if hc.auth != nil {
				hc.auth(c)
			}
		},
	})
	if err != nil {
		t.Fatalf("MCP-Server: %v", err)
	}
	registerTestWriteTool(h.srv.Toolbox())

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	h.port = l.Addr().(*net.TCPAddr).Port
	ts := httptest.NewUnstartedServer(h.srv.Handler(h.port))
	_ = ts.Listener.Close()
	ts.Listener = l
	ts.Start()
	t.Cleanup(func() {
		ts.Close()
		_ = h.srv.Close(context.Background())
	})
	h.base = fmt.Sprintf("http://127.0.0.1:%d", h.port)
	h.http = &http.Client{
		Timeout:       20 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	return h
}

func (h *harness) createTenant(name string) *domain.TenantConfig {
	h.t.Helper()
	tc, err := h.host.CreateTenant(uiCtx, name, filepath.Join(h.t.TempDir(), "daten"),
		domain.CompanySettings{CompanyName: name, FiscalYear: 2026, LegalForm: domain.LegalFormGmbH})
	if err != nil {
		h.t.Fatalf("Mandant %s anlegen: %v", name, err)
	}
	return tc
}

// do schickt eine Anfrage; headers setzen oder überschreiben Köpfe, „Host"
// eingeschlossen.
func (h *harness) do(method, path string, body io.Reader, headers map[string]string) *http.Response {
	h.t.Helper()
	req, err := http.NewRequest(method, h.base+path, body)
	if err != nil {
		h.t.Fatal(err)
	}
	for k, v := range headers {
		if k == "Host" {
			req.Host = v
			continue
		}
		req.Header.Set(k, v)
	}
	resp, err := h.http.Do(req)
	if err != nil {
		h.t.Fatalf("%s %s: %v", method, path, err)
	}
	h.t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func decodeJSON(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("Antwort ist kein JSON (%d): %v", resp.StatusCode, err)
	}
	return m
}

func (h *harness) postForm(path string, form url.Values) (int, map[string]any) {
	h.t.Helper()
	resp := h.do(http.MethodPost, path, strings.NewReader(form.Encode()),
		map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if resp.StatusCode == http.StatusOK && path == oauth.PathRevoke {
		return resp.StatusCode, nil
	}
	return resp.StatusCode, decodeJSON(h.t, resp)
}

func (h *harness) register(name string, redirects ...string) (int, map[string]any) {
	h.t.Helper()
	body, _ := json.Marshal(map[string]any{
		"client_name": name, "redirect_uris": redirects,
		"grant_types": []string{"authorization_code", "refresh_token"}, "token_endpoint_auth_method": "none",
	})
	resp := h.do(http.MethodPost, oauth.PathRegister, strings.NewReader(string(body)),
		map[string]string{"Content-Type": "application/json"})
	return resp.StatusCode, decodeJSON(h.t, resp)
}

func (h *harness) registerClient(name string) string {
	h.t.Helper()
	status, body := h.register(name, "http://127.0.0.1/callback", "http://localhost/callback")
	if status != http.StatusCreated {
		h.t.Fatalf("Registrierung: %d %v", status, body)
	}
	return body["client_id"].(string)
}

type pkce struct{ verifier, challenge string }

func newPKCE() pkce {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	v := base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(v))
	return pkce{verifier: v, challenge: base64.RawURLEncoding.EncodeToString(sum[:])}
}

// authorizeQuery sind die Parameter einer gültigen Anfrage.
func (h *harness) authorizeQuery(clientID, redirect string, p pkce) url.Values {
	return url.Values{
		"response_type": {"code"}, "client_id": {clientID}, "redirect_uri": {redirect},
		"code_challenge": {p.challenge}, "code_challenge_method": {"S256"},
		"state": {"zustand-123"}, "scope": {"buchfink:lesen buchfink:buchen"},
		"resource": {h.base + "/mcp"},
	}
}

var waitLink = regexp.MustCompile(`url=(/authorize/warten\?anfrage=[^"]+)"`)

// browse folgt dem Browser: /authorize, dann /authorize/warten, bis zum
// Rücksprung. Es liefert die Rücksprungadresse.
func (h *harness) browse(q url.Values) *url.URL {
	h.t.Helper()
	resp := h.do(http.MethodGet, oauth.PathAuthorize+"?"+q.Encode(), nil, map[string]string{"Sec-Fetch-Site": "none"})
	for range 20 {
		switch resp.StatusCode {
		case http.StatusFound:
			loc, err := url.Parse(resp.Header.Get("Location"))
			if err != nil {
				h.t.Fatal(err)
			}
			return loc
		case http.StatusOK:
			m := waitLink.FindStringSubmatch(readBody(h.t, resp))
			if m == nil {
				h.t.Fatalf("Warteseite ohne Weiterleitung")
			}
			link := strings.ReplaceAll(m[1], "&amp;", "&")
			resp = h.do(http.MethodGet, link, nil, map[string]string{"Sec-Fetch-Site": "same-origin"})
		default:
			h.t.Fatalf("Browser: unerwartet %d: %s", resp.StatusCode, readBody(h.t, resp))
		}
	}
	h.t.Fatal("Browser: keine Entscheidung")
	return nil
}

func (h *harness) exchange(clientID, code, redirect, verifier string) (int, map[string]any) {
	return h.postForm(oauth.PathToken, url.Values{
		"grant_type": {"authorization_code"}, "client_id": {clientID}, "code": {code},
		"redirect_uri": {redirect}, "code_verifier": {verifier}, "resource": {h.base + "/mcp"},
	})
}

type tokens struct {
	clientID, access, refresh, scope string
}

// login ist der ganze Weg einer App: registrieren, im Browser anfragen, in
// der App zustimmen lassen, Code einlösen.
func (h *harness) login(name, scope string) tokens {
	h.t.Helper()
	clientID := h.registerClient(name)
	p := newPKCE()
	redirect := "http://127.0.0.1:53682/callback"
	q := h.authorizeQuery(clientID, redirect, p)
	if scope != "" {
		q.Set("scope", scope)
	}
	loc := h.browse(q)
	code := loc.Query().Get("code")
	if code == "" {
		h.t.Fatalf("kein Code im Rücksprung: %s", loc)
	}
	status, body := h.exchange(clientID, code, redirect, p.verifier)
	if status != http.StatusOK {
		h.t.Fatalf("Token: %d %v", status, body)
	}
	return tokens{clientID: clientID, access: body["access_token"].(string),
		refresh: body["refresh_token"].(string), scope: body["scope"].(string)}
}

type bearerTransport struct{ token string }

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(r)
}

// session verbindet einen MCP-Client mit festem Token.
func (h *harness) session(token, version string) (*mcp.ClientSession, error) {
	h.t.Helper()
	transport := &mcp.StreamableClientTransport{
		Endpoint: h.base + "/mcp", HTTPClient: &http.Client{Transport: bearerTransport{token}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "testclient", Version: "1"}, nil).
		Connect(ctx, transport, &mcp.ClientSessionOptions{ProtocolVersion: version})
	if err != nil {
		return nil, err
	}
	h.t.Cleanup(func() { _ = cs.Close() })
	return cs, nil
}

func (h *harness) mustSession(token string) *mcp.ClientSession {
	h.t.Helper()
	cs, err := h.session(token, "")
	if err != nil {
		h.t.Fatalf("MCP-Verbindung: %v", err)
	}
	return cs
}

func callTool(t *testing.T, cs *mcp.ClientSession, name string, args any) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return res
}

func resultText(res *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

func statusResult(t *testing.T, res *mcp.CallToolResult) statusOut {
	t.Helper()
	if res.IsError {
		t.Fatalf("status scheitert: %s", resultText(res))
	}
	var out statusOut
	if err := json.Unmarshal([]byte(resultText(res)), &out); err != nil {
		t.Fatalf("status: %v", err)
	}
	return out
}

func toolErrorOf(t *testing.T, res *mcp.CallToolResult) ToolError {
	t.Helper()
	if !res.IsError {
		t.Fatalf("erwartet einen Fehler, bekam %s", resultText(res))
	}
	var body struct {
		Fehler ToolError `json:"fehler"`
	}
	if err := json.Unmarshal([]byte(resultText(res)), &body); err != nil {
		t.Fatalf("Fehler im falschen Format: %s", resultText(res))
	}
	return body.Fehler
}

// Ein schreibendes Werkzeug nur für die Tests des Rahmens: es legt einen
// Lieferanten an.
type testContactIn struct {
	WriteArgs
	Name string `json:"name"`
}

type testContactOut struct {
	Wiederholung
	ID uint `json:"id"`
}

func (o testContactOut) Ref() string { return fmt.Sprintf("kontakt:%d", o.ID) }

func registerTestWriteTool(tb *Toolbox) {
	AddWrite(tb, Spec{Name: "test_lieferant", Title: "Lieferant anlegen (Test)", Description: "nur für Tests"},
		func(ctx context.Context, c *Call, s *core.Services, in testContactIn) (testContactOut, error) {
			contact := domain.Contact{Type: domain.ContactTypeVendor, Name: in.Name}
			if err := s.Contacts.SaveContact(ctx, &contact); err != nil {
				return testContactOut{}, err
			}
			return testContactOut{ID: contact.ID}, nil
		})
}

func (h *harness) countVendors(name string) int {
	h.t.Helper()
	n := 0
	err := h.host.Read(uiCtx, func(ctx context.Context, s *core.Services) error {
		contacts, err := s.Repos.Contacts.FindAll(ctx)
		for _, c := range contacts {
			if c.Name == name {
				n++
			}
		}
		return err
	})
	if err != nil {
		h.t.Fatal(err)
	}
	return n
}
