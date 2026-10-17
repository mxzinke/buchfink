package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/mcpserver"
	"github.com/buchfink/buchfink/internal/mcpserver/oauth"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Die Brücke wird gegen den echten MCP-Server der App getestet, nur ohne
// Fenster: ein Broker entscheidet, ein simulierter Browser folgt den Seiten.

type autoBroker struct {
	srv     atomic.Pointer[mcpserver.Server]
	approve atomic.Bool
}

func (b *autoBroker) Present(r oauth.ConsentRequest) {
	go func() { _ = b.srv.Load().Resolve(context.Background(), r.ID, r.Nonce, b.approve.Load()) }()
}
func (b *autoBroker) Withdraw(string) {}

type testApp struct {
	endpoint string
	srv      *mcpserver.Server
	broker   *autoBroker
	// close beendet die App vor dem Ende des Tests.
	close func()
}

func startApp(t *testing.T) *testApp { return startAppAt(t, "127.0.0.1:0") }

// startAppAt startet eine App mit leerem Speicher an einer Adresse.
func startAppAt(t *testing.T, addr string) *testApp {
	t.Helper()
	base := t.TempDir()
	host, err := core.NewHost(context.Background(), core.HostOptions{
		Config:  repository.NewAppConfigRepository(filepath.Join(base, "config")),
		BaseDir: base, Network: core.OfflineNetwork(),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = host.Close(context.Background()) })
	ui := actor.WithChannel(context.Background(), actor.ChannelUI)
	if _, err := host.CreateTenant(ui, "Pfennig Ventures GmbH", filepath.Join(base, "daten"),
		domain.CompanySettings{CompanyName: "Pfennig Ventures GmbH", FiscalYear: 2026}); err != nil {
		t.Fatal(err)
	}
	broker := &autoBroker{}
	broker.approve.Store(true)
	srv, err := mcpserver.New(mcpserver.Options{Host: host, StorePath: filepath.Join(base, mcpserver.StoreFile), Broker: broker,
		Auth: func(c *oauth.Config) { c.LongPoll = time.Second }})
	if err != nil {
		t.Fatal(err)
	}
	broker.srv.Store(srv)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	ts := httptest.NewUnstartedServer(srv.Handler(port))
	_ = ts.Listener.Close()
	ts.Listener = l
	ts.Start()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			ts.Close()
			_ = srv.Close(context.Background())
		})
	}
	t.Cleanup(stop)
	return &testApp{endpoint: fmt.Sprintf("http://127.0.0.1:%d/mcp", port), srv: srv, broker: broker, close: stop}
}

var refreshLink = regexp.MustCompile(`url=(/authorize/warten\?anfrage=[^"]+)"`)

// browser folgt der Anmeldeseite bis zum Rücksprung in die Brücke.
func browser(t *testing.T, opened *atomic.Int32) func(string) error {
	return func(authURL string) error {
		opened.Add(1)
		go func() {
			client := &http.Client{Timeout: 30 * time.Second}
			req, _ := http.NewRequest(http.MethodGet, authURL, nil)
			req.Header.Set("Sec-Fetch-Site", "none")
			resp, err := client.Do(req)
			for range 20 {
				if err != nil {
					t.Errorf("Browser: %v", err)
					return
				}
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				m := refreshLink.FindStringSubmatch(string(body))
				if m == nil {
					return // Rücksprung erreicht oder Fehlerseite
				}
				u, _ := resp.Request.URL.Parse(strings.ReplaceAll(m[1], "&amp;", "&"))
				resp, err = client.Get(u.String())
			}
		}()
		return nil
	}
}

func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// connect startet eine Brücke und verbindet einen stdio-Client mit ihr.
func connect(t *testing.T, cfg config) (*mcp.ClientSession, *bridge) {
	t.Helper()
	if cfg.Logger == nil {
		cfg.Logger = quietLogger()
	}
	b, err := newBridge(cfg)
	if err != nil {
		t.Fatal(err)
	}
	clientT, serverT := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = b.run(ctx, serverT)
		close(done)
	}()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "claude-desktop-test"}, nil).Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cs.Close()
		cancel()
		<-done
	})
	return cs, b
}

func call(t *testing.T, cs *mcp.ClientSession, name string) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return res
}

func text(res *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

func errorCode(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if !res.IsError {
		t.Fatalf("erwartet einen Fehler, bekam %s", text(res))
	}
	var body struct {
		Fehler struct {
			Code string `json:"code"`
		} `json:"fehler"`
	}
	if err := json.Unmarshal([]byte(text(res)), &body); err != nil {
		t.Fatalf("Fehler im falschen Format: %s", text(res))
	}
	return body.Fehler.Code
}

// Erster Start: der erste Aufruf meldet die Brücke an (Browser, Zustimmung
// in der App), danach reicht sie durch. Ein zweiter Prozess nutzt dieselbe
// Anmeldung ohne Browser.
func TestBridgeLogsInOnceAndForwards(t *testing.T) {
	app := startApp(t)
	dir := t.TempDir()
	var opened atomic.Int32
	cs, _ := connect(t, config{Endpoint: app.endpoint, StateDir: dir, ClientName: "Claude Desktop (Test)",
		OpenBrowser: browser(t, &opened), LoginWait: 20 * time.Second})

	res := call(t, cs, "status")
	if res.IsError {
		t.Fatalf("status über die Brücke: %s", text(res))
	}
	var out struct {
		Bereit     bool `json:"bereit"`
		Verbindung struct {
			Erfassungsweg string `json:"erfassungsweg"`
		} `json:"verbindung"`
	}
	if err := json.Unmarshal([]byte(text(res)), &out); err != nil || !out.Bereit ||
		out.Verbindung.Erfassungsweg != "agent:claude-desktop-test" {
		t.Fatalf("status: %s", text(res))
	}
	if opened.Load() != 1 {
		t.Errorf("Browser %d-mal geöffnet, erwartet einmal", opened.Load())
	}

	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	if len(files) != 2 {
		t.Fatalf("Anmeldung und Werkzeugliste fehlen: %v", files)
	}
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Errorf("%s: Rechte %v", filepath.Base(f), info.Mode().Perm())
		}
	}

	tools, err := cs.ListTools(context.Background(), nil)
	if err != nil || len(tools.Tools) == 0 || tools.Tools[0].Annotations == nil || !tools.Tools[0].Annotations.ReadOnlyHint ||
		tools.Tools[0].OutputSchema == nil {
		t.Errorf("Werkzeuge kommen nicht unverändert durch: %+v %v", tools, err)
	}

	second, _ := connect(t, config{Endpoint: app.endpoint, StateDir: dir, Logger: quietLogger(),
		OpenBrowser: func(string) error { t.Error("zweiter Prozess öffnet den Browser"); return nil }})
	if res := call(t, second, "status"); res.IsError {
		t.Errorf("zweiter Prozess: %s", text(res))
	}
}

// Ist Buchfink zu, antwortet die Brücke sofort und verständlich, und die
// Werkzeugliste vom letzten Kontakt bleibt stehen.
func TestBridgeAnswersWhenBuchfinkIsClosed(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	cs, _ := connect(t, config{Endpoint: "http://127.0.0.1:" + strconv.Itoa(port) + "/mcp", StateDir: t.TempDir(),
		OpenBrowser: func(string) error { t.Error("ohne App darf kein Browser aufgehen"); return nil }})

	tools, err := cs.ListTools(context.Background(), nil)
	if err != nil || len(tools.Tools) != 1 || tools.Tools[0].Name != "status" {
		t.Fatalf("Werkzeuge ohne App: %+v %v", tools, err)
	}
	start := time.Now()
	if code := errorCode(t, call(t, cs, "status")); code != "app_nicht_geoeffnet" {
		t.Errorf("Code %s", code)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("die Antwort ohne App dauerte %v", d)
	}
}

func TestBridgeReportsDeniedLogin(t *testing.T) {
	app := startApp(t)
	app.broker.approve.Store(false)
	var opened atomic.Int32
	cs, _ := connect(t, config{Endpoint: app.endpoint, StateDir: t.TempDir(),
		OpenBrowser: browser(t, &opened), LoginWait: 20 * time.Second})
	if code := errorCode(t, call(t, cs, "status")); code != "anmeldung_abgelehnt" {
		t.Errorf("Code %s", code)
	}
}

// Entzieht der Mensch den Zugriff, fragt die Brücke beim nächsten Aufruf
// neu an, statt mit einem toten Token weiterzumachen.
func TestBridgeLogsInAgainAfterRevocation(t *testing.T) {
	app := startApp(t)
	var opened atomic.Int32
	cs, _ := connect(t, config{Endpoint: app.endpoint, StateDir: t.TempDir(),
		OpenBrowser: browser(t, &opened), LoginWait: 20 * time.Second})
	if res := call(t, cs, "status"); res.IsError {
		t.Fatalf("erste Anmeldung: %s", text(res))
	}
	apps, err := app.srv.ConnectedApps(context.Background())
	if err != nil || len(apps) != 1 {
		t.Fatalf("verbundene Apps: %v %v", apps, err)
	}
	if err := app.srv.RevokeApp(context.Background(), apps[0].ClientID); err != nil {
		t.Fatal(err)
	}
	if res := call(t, cs, "status"); res.IsError {
		t.Fatalf("nach dem Entzug: %s", text(res))
	}
	if opened.Load() != 2 {
		t.Errorf("Browser %d-mal geöffnet, erwartet zweimal", opened.Load())
	}
}

// Vergisst Buchfink die Registrierung (Speicher zurückgesetzt), meldet die
// Brücke sich neu an, statt mit der alten Kennung festzuhängen.
func TestBridgeRegistersAgainWhenBuchfinkForgotIt(t *testing.T) {
	app := startApp(t)
	dir := t.TempDir()
	var opened atomic.Int32
	cs, b := connect(t, config{Endpoint: app.endpoint, StateDir: dir,
		OpenBrowser: browser(t, &opened), LoginWait: 20 * time.Second})
	if res := call(t, cs, "status"); res.IsError {
		t.Fatalf("erste Anmeldung: %s", text(res))
	}
	before, err := b.store.load(app.endpoint)
	if err != nil || before.ClientID == "" || !before.ClientApproved {
		t.Fatalf("Anmeldung: %+v %v", before, err)
	}

	app.close()
	u, _ := url.Parse(app.endpoint)
	startAppAt(t, u.Host)
	if res := call(t, cs, "status"); res.IsError {
		t.Fatalf("nach dem Zurücksetzen: %s", text(res))
	}
	after, _ := b.store.load(app.endpoint)
	if after.ClientID == before.ClientID || opened.Load() != 2 {
		t.Errorf("Registrierung %s → %s, Browser %d-mal", before.ClientID, after.ClientID, opened.Load())
	}
}

// Eine Registrierung, mit der nie ein Token kam, verwendet die Brücke nicht
// wieder: Buchfink räumt sie ab, und die alte Kennung führte auf die
// Fehlerseite von /authorize.
func TestBridgeDropsRegistrationThatWasNeverApproved(t *testing.T) {
	app := startApp(t)
	app.broker.approve.Store(false)
	dir := t.TempDir()
	var opened atomic.Int32
	cs, b := connect(t, config{Endpoint: app.endpoint, StateDir: dir,
		OpenBrowser: browser(t, &opened), LoginWait: 20 * time.Second})
	if code := errorCode(t, call(t, cs, "status")); code != "anmeldung_abgelehnt" {
		t.Fatalf("Code %s", code)
	}
	denied, _ := b.store.load(app.endpoint)
	if denied.ClientID == "" || denied.ClientApproved {
		t.Fatalf("nach der Ablehnung: %+v", denied)
	}
	app.broker.approve.Store(true)
	if res := call(t, cs, "status"); res.IsError {
		t.Fatalf("zweiter Versuch: %s", text(res))
	}
	approved, _ := b.store.load(app.endpoint)
	if approved.ClientID == denied.ClientID || !approved.ClientApproved {
		t.Errorf("Registrierung %s → %s", denied.ClientID, approved.ClientID)
	}
}

// Belegt ein fremder Prozess den Port, während Buchfink zu ist, nennt er
// beliebige Anmeldeadressen. Die Brücke folgt nur Adressen auf demselben
// Server und öffnet nie etwas anderes als eine Webadresse.
func TestBridgeFollowsOnlyItsOwnServer(t *testing.T) {
	cases := map[string]func(base string, meta map[string]any){
		"Programm über file:": func(_ string, m map[string]any) {
			m["authorization_endpoint"] = "file://localhost/C:/Windows/System32/calc.exe"
		},
		"fremde Seite": func(_ string, m map[string]any) {
			m["authorization_endpoint"] = "https://anmeldung.example/authorize"
		},
		"Token-Endpunkt woanders": func(_ string, m map[string]any) {
			m["token_endpoint"] = "http://127.0.0.1:1/token"
		},
		"Registrierung woanders": func(_ string, m map[string]any) {
			m["registration_endpoint"] = "https://anmeldung.example/register"
		},
		"Anmeldeserver woanders": func(_ string, m map[string]any) {
			m["prm_authorization_server"] = "http://127.0.0.1:1"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			endpoint := foreignServer(t, mutate)
			var opened []string
			o := foreignClient(t, endpoint, &opened)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := o.login(ctx)
			if err == nil || !strings.Contains(err.Error(), "fremde Adresse") {
				t.Errorf("Anmeldung: %v", err)
			}
			if len(opened) != 0 {
				t.Errorf("Browser geöffnet mit %v", opened)
			}
		})
	}

	// Gegenprobe: derselbe Server mit eigenen Adressen kommt bis zum Browser.
	endpoint := foreignServer(t, func(string, map[string]any) {})
	var opened []string
	o := foreignClient(t, endpoint, &opened)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := o.login(ctx)
	if len(opened) != 1 || !strings.HasPrefix(opened[0], strings.TrimSuffix(endpoint, "/mcp")+"/authorize?") {
		t.Errorf("Gegenprobe: %v %v", opened, err)
	}
}

func TestBrowserOpensOnlyWebAddresses(t *testing.T) {
	for _, raw := range []string{"file:///etc/passwd", "file://localhost/C:/Windows/System32/calc.exe",
		"ms-settings:", "javascript:alert(1)", "/authorize"} {
		if err := openBrowser(raw); err == nil {
			t.Errorf("%s wurde geöffnet", raw)
		}
	}
}

// foreignServer spielt einen Prozess, der den Port von Buchfink belegt hat.
// mutate ändert die Anmeldedaten, die er nennt.
func foreignServer(t *testing.T, mutate func(base string, meta map[string]any)) string {
	t.Helper()
	var base string
	meta := map[string]any{}
	writeJSON := func(w http.ResponseWriter, status int, body any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+base+`/.well-known/oauth-protected-resource/mcp"`)
		w.WriteHeader(http.StatusUnauthorized)
	})
	mux.HandleFunc("/.well-known/oauth-protected-resource/mcp", func(w http.ResponseWriter, r *http.Request) {
		as := base
		if v, ok := meta["prm_authorization_server"].(string); ok {
			as = v
		}
		writeJSON(w, http.StatusOK, map[string]any{"resource": base + "/mcp", "authorization_servers": []string{as}})
	})
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		out := map[string]any{}
		for k, v := range meta {
			if k != "prm_authorization_server" {
				out[k] = v
			}
		}
		writeJSON(w, http.StatusOK, out)
	})
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusCreated, map[string]any{"client_id": "fremd", "redirect_uris": []string{"http://127.0.0.1/callback"}})
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	base = ts.URL
	for k, v := range map[string]any{
		"issuer": base, "authorization_endpoint": base + "/authorize", "token_endpoint": base + "/token",
		"registration_endpoint": base + "/register", "response_types_supported": []string{"code"},
		"code_challenge_methods_supported": []string{"S256"},
	} {
		meta[k] = v
	}
	mutate(base, meta)
	return base + "/mcp"
}

func foreignClient(t *testing.T, endpoint string, opened *[]string) *oauthClient {
	t.Helper()
	store, err := newStateStore(t.TempDir(), endpoint)
	if err != nil {
		t.Fatal(err)
	}
	o := &oauthClient{endpoint: endpoint, clientName: "Test", store: store, http: &http.Client{Timeout: 5 * time.Second},
		open: func(u string) error { *opened = append(*opened, u); return nil }, logger: quietLogger()}
	resp, err := http.Post(endpoint, "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	o.challenge = resp.Header.Values("WWW-Authenticate")
	return o
}

// Eine Fehlerantwort der App ist keine abgerissene Verbindung: sie kommt als
// solche beim Client an und nicht als „Buchfink ist nicht geöffnet".
func TestBridgePassesProtocolErrorsThrough(t *testing.T) {
	app := startApp(t)
	var opened atomic.Int32
	cs, b := connect(t, config{Endpoint: app.endpoint, StateDir: t.TempDir(),
		OpenBrowser: browser(t, &opened), LoginWait: 20 * time.Second})
	if res := call(t, cs, "status"); res.IsError {
		t.Fatalf("status: %s", text(res))
	}
	b.applyTools([]*mcp.Tool{placeholderStatus(), {Name: "gibt_es_nicht", InputSchema: map[string]any{"type": "object"}}})
	_, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "gibt_es_nicht", Arguments: map[string]any{}})
	if err == nil || strings.Contains(err.Error(), "app_nicht_geoeffnet") {
		t.Errorf("unbekanntes Werkzeug: %v", err)
	}
}

func TestEndpointMustBeThisComputer(t *testing.T) {
	for raw, ok := range map[string]bool{
		"http://127.0.0.1:7465/mcp":   true,
		"http://localhost:7465/mcp":   true,
		"http://[::1]:7465/mcp":       true,
		"https://example.com/mcp":     false,
		"http://192.168.1.5:7465/mcp": false,
		"http://127.0.0.1/mcp":        false,
	} {
		if err := checkEndpoint(raw); (err == nil) != ok {
			t.Errorf("%s: %v", raw, err)
		}
	}
}

// Die Brücke ist ein eigenes kleines Programm: kein Wails, kein Kern, keine
// Datenbank.
func TestBridgeImportsNeitherWailsNorTheCore(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range pkgs {
		for name, file := range pkg.Files {
			for _, imp := range file.Imports {
				path := strings.Trim(imp.Path.Value, `"`)
				if strings.Contains(path, "wailsapp") || strings.Contains(path, "buchfink/internal") ||
					strings.Contains(path, "gorm") || strings.Contains(path, "sqlite") {
					t.Errorf("%s importiert %s", name, path)
				}
			}
		}
	}
}
