package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Die Brücke ist auf der stdio-Seite ein MCP-Server und auf der HTTP-Seite
// ein MCP-Client der laufenden Buchfink-App. Werkzeuge reicht sie unverändert
// durch: Name, Beschreibung, Schemas, Annotations und _meta kommen aus der
// App. Ist die App zu, antwortet die Brücke selbst, statt auf eine
// Zeitüberschreitung zu warten.

const defaultInstructions = `Werkzeuge der Buchfink-App auf diesem Rechner. Beginne mit status.
Ist Buchfink nicht geöffnet, antworten die Werkzeuge mit app_nicht_geoeffnet.`

type config struct {
	Endpoint   string
	StateDir   string
	ClientName string
	Logger     *slog.Logger
	// OpenBrowser öffnet die Anmeldeseite.
	OpenBrowser func(url string) error
	// LoginWait ist, wie lange ein Werkzeugaufruf auf eine gerade begonnene
	// Anmeldung wartet; LoginTimeout, wie lange die Anmeldung insgesamt läuft.
	LoginWait, LoginTimeout time.Duration
	// ConnectTimeout begrenzt den Verbindungsaufbau zur App.
	ConnectTimeout time.Duration
}

type bridge struct {
	cfg    config
	server *mcp.Server
	auth   *oauthClient
	store  *stateStore
	http   *http.Client

	connMu sync.Mutex
	mu     sync.Mutex
	remote *mcp.ClientSession
	tools  map[string]bool

	loginMu sync.Mutex
	login   *loginRun
}

type loginRun struct {
	done chan struct{}
	err  error
}

// toolCache ist die zuletzt gesehene Werkzeugliste samt Hinweisen der App.
type toolCache struct {
	Instructions string      `json:"instructions"`
	Tools        []*mcp.Tool `json:"tools"`
}

func newBridge(cfg config) (*bridge, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	if cfg.LoginWait <= 0 {
		cfg.LoginWait = 45 * time.Second
	}
	if cfg.LoginTimeout <= 0 {
		cfg.LoginTimeout = 10 * time.Minute
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = 10 * time.Second
	}
	if cfg.OpenBrowser == nil {
		cfg.OpenBrowser = openBrowser
	}
	store, err := newStateStore(cfg.StateDir, cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	httpClient := &http.Client{Transport: &http.Transport{
		Proxy:                 nil, // Buchfink läuft auf diesem Rechner; kein Proxy dazwischen.
		DialContext:           (&net.Dialer{Timeout: 2 * time.Second}).DialContext,
		ResponseHeaderTimeout: 5 * time.Minute,
		MaxIdleConns:          4,
		IdleConnTimeout:       time.Minute,
	}}
	b := &bridge{
		cfg: cfg, store: store, http: httpClient, tools: map[string]bool{},
		auth: &oauthClient{
			endpoint: cfg.Endpoint, clientName: cfg.ClientName, store: store,
			http: httpClient, open: cfg.OpenBrowser, logger: cfg.Logger,
		},
	}
	cache := b.loadCache()
	instructions := cache.Instructions
	if instructions == "" {
		instructions = defaultInstructions
	}
	b.server = mcp.NewServer(&mcp.Implementation{Name: "buchfink", Title: "Buchfink", Version: version},
		&mcp.ServerOptions{Instructions: instructions})
	tools := cache.Tools
	if len(tools) == 0 {
		tools = []*mcp.Tool{placeholderStatus()}
	}
	b.applyTools(tools)
	return b, nil
}

// placeholderStatus steht in der Liste, solange die Brücke Buchfink noch nie
// erreicht hat.
func placeholderStatus() *mcp.Tool {
	return &mcp.Tool{
		Name: "status", Title: "Status von Buchfink",
		Description: "Überblick und Selbstauskunft der Buchfink-App. Zu Beginn jeder Sitzung aufrufen.",
		InputSchema: map[string]any{"type": "object"},
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}
}

// run bedient stdio, bis der Client die Verbindung schließt. Die Verbindung
// zur App entsteht im Hintergrund; ohne Anmeldung öffnet sie keinen Browser.
func (b *bridge) run(ctx context.Context, transport mcp.Transport) error {
	go func() {
		if _, err := b.session(ctx); err != nil {
			b.cfg.Logger.Info("Buchfink ist noch nicht verbunden", "grund", err)
		}
	}()
	err := b.server.Run(ctx, transport)
	b.closeRemote()
	return err
}

// session liefert die Verbindung zur App und baut sie bei Bedarf auf.
func (b *bridge) session(ctx context.Context) (*mcp.ClientSession, error) {
	b.connMu.Lock()
	defer b.connMu.Unlock()
	b.mu.Lock()
	cs := b.remote
	b.mu.Unlock()
	if cs != nil {
		return cs, nil
	}
	ctx, cancel := context.WithTimeout(ctx, b.cfg.ConnectTimeout)
	defer cancel()
	transport := &mcp.StreamableClientTransport{
		Endpoint: b.cfg.Endpoint, HTTPClient: b.http, OAuthHandler: b.auth,
		DisableStandaloneSSE: true, MaxRetries: -1,
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "buchfink-mcp", Version: version}, nil)
	cs, err := client.Connect(ctx, transport, nil)
	if err != nil {
		if b.auth.needLogin.Load() {
			return nil, errLoginRequired
		}
		return nil, err
	}
	b.mu.Lock()
	b.remote = cs
	b.mu.Unlock()
	if err := b.syncTools(ctx, cs); err != nil {
		b.cfg.Logger.Warn("Werkzeugliste nicht abgeglichen", "fehler", err)
	}
	return cs, nil
}

func (b *bridge) dropSession(cs *mcp.ClientSession) {
	b.mu.Lock()
	if b.remote == cs {
		b.remote = nil
	}
	b.mu.Unlock()
	_ = cs.Close()
}

func (b *bridge) closeRemote() {
	b.mu.Lock()
	cs := b.remote
	b.remote = nil
	b.mu.Unlock()
	if cs != nil {
		_ = cs.Close()
	}
}

// syncTools übernimmt die Werkzeuge der App und merkt sie sich für den
// nächsten Start ohne App.
func (b *bridge) syncTools(ctx context.Context, cs *mcp.ClientSession) error {
	var tools []*mcp.Tool
	for tool, err := range cs.Tools(ctx, nil) {
		if err != nil {
			return err
		}
		tools = append(tools, tool)
	}
	if len(tools) == 0 {
		return nil
	}
	b.applyTools(tools)
	instructions := ""
	if res := cs.InitializeResult(); res != nil {
		instructions = res.Instructions
	}
	data, err := json.Marshal(toolCache{Instructions: instructions, Tools: tools})
	if err != nil {
		return err
	}
	return writeFileAtomic(b.store.tools, data)
}

func (b *bridge) loadCache() toolCache {
	var cache toolCache
	if data, err := os.ReadFile(b.store.tools); err == nil {
		_ = json.Unmarshal(data, &cache)
	}
	cache.Tools = slices.DeleteFunc(cache.Tools, func(t *mcp.Tool) bool { return t == nil || t.Name == "" || t.InputSchema == nil })
	return cache
}

// applyTools gleicht die Werkzeuge der stdio-Seite ab; der Client bekommt
// tools/list_changed.
func (b *bridge) applyTools(tools []*mcp.Tool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	keep := map[string]bool{}
	for _, t := range tools {
		keep[t.Name] = true
	}
	var gone []string
	for name := range b.tools {
		if !keep[name] {
			gone = append(gone, name)
			delete(b.tools, name)
		}
	}
	if len(gone) > 0 {
		b.server.RemoveTools(gone...)
	}
	for _, t := range tools {
		b.server.AddTool(t, b.forward)
		b.tools[t.Name] = true
	}
}

// forward reicht einen Werkzeugaufruf an die App weiter.
func (b *bridge) forward(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	params := &mcp.CallToolParams{Name: req.Params.Name, Arguments: req.Params.Arguments}
	for attempt := 0; attempt < 2; attempt++ {
		cs, err := b.session(ctx)
		if errors.Is(err, errLoginRequired) {
			if res := b.awaitLogin(ctx); res != nil {
				return res, nil
			}
			continue
		}
		if err != nil {
			return b.offline(err), nil
		}
		res, err := cs.CallTool(ctx, params)
		if err == nil {
			return res, nil
		}
		if rpcErr, ok := appAnswer(err); ok {
			return nil, rpcErr
		}
		b.dropSession(cs)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if b.auth.needLogin.Load() {
			if res := b.awaitLogin(ctx); res != nil {
				return res, nil
			}
			continue
		}
		b.cfg.Logger.Info("Aufruf an Buchfink gescheitert, neuer Versuch", "werkzeug", params.Name, "fehler", err)
	}
	return b.offline(errors.New("keine Verbindung")), nil
}

// appAnswer erkennt eine Fehlerantwort der App, etwa auf ein Werkzeug, das
// es nicht mehr gibt. Sie geht als solche zurück; nur eine abgerissene oder
// abgewiesene Verbindung heißt „nicht geöffnet". Die Codes -32001 bis -32005
// setzt der Transport des SDK selbst.
func appAnswer(err error) (*jsonrpc.Error, bool) {
	var rpcErr *jsonrpc.Error
	if !errors.As(err, &rpcErr) {
		return nil, false
	}
	if rpcErr.Code <= -32001 && rpcErr.Code >= -32005 {
		return nil, false
	}
	return rpcErr, true
}

// awaitLogin startet die Anmeldung, falls keine läuft, und wartet eine
// Weile darauf. Ist sie dann durch, liefert es nil und der Aufruf läuft
// weiter; sonst die Antwort an den Agenten.
func (b *bridge) awaitLogin(ctx context.Context) *mcp.CallToolResult {
	run := b.startLogin()
	timer := time.NewTimer(b.cfg.LoginWait)
	defer timer.Stop()
	select {
	case <-run.done:
		if run.err == nil {
			return nil
		}
		if errors.Is(run.err, errLoginDenied) {
			return toolFailure("anmeldung_abgelehnt",
				"Die Anmeldung der Brücke wurde in Buchfink abgelehnt.",
				"Den Menschen fragen, ob die App Zugriff bekommen soll, und den Aufruf dann wiederholen.", false)
		}
		return toolFailure("anmeldung_gescheitert", "Die Anmeldung bei Buchfink ist gescheitert: "+run.err.Error(),
			"Prüfen, ob Buchfink geöffnet ist, und den Aufruf wiederholen.", true)
	case <-timer.C:
		return toolFailure("anmeldung_erforderlich",
			"Die Brücke wartet auf die Zustimmung in Buchfink. Im Browser steht ein Abgleichcode.",
			"Den Menschen bitten, im Fenster von Buchfink denselben Code zu sehen und zuzulassen; danach den Aufruf wiederholen.", true)
	case <-ctx.Done():
		return toolFailure("anmeldung_erforderlich", "Der Aufruf wurde abgebrochen, bevor die Anmeldung bestätigt war.",
			"Nach der Zustimmung in Buchfink den Aufruf wiederholen.", true)
	}
}

func (b *bridge) startLogin() *loginRun {
	b.loginMu.Lock()
	defer b.loginMu.Unlock()
	if b.login != nil {
		select {
		case <-b.login.done:
			if b.login.err == nil && !b.auth.needLogin.Load() {
				return b.login
			}
		default:
			return b.login
		}
	}
	run := &loginRun{done: make(chan struct{})}
	b.login = run
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), b.cfg.LoginTimeout)
		defer cancel()
		run.err = b.auth.login(ctx)
		if run.err != nil {
			b.cfg.Logger.Warn("Anmeldung bei Buchfink gescheitert", "fehler", run.err)
		}
		close(run.done)
	}()
	return run
}

// offline ist die Antwort, solange Buchfink nicht erreichbar ist.
func (b *bridge) offline(cause error) *mcp.CallToolResult {
	b.cfg.Logger.Info("Buchfink nicht erreichbar", "adresse", b.cfg.Endpoint, "grund", cause)
	return toolFailure("app_nicht_geoeffnet",
		fmt.Sprintf("Buchfink ist nicht geöffnet oder unter %s nicht erreichbar.", b.cfg.Endpoint),
		"Den Menschen bitten, Buchfink zu öffnen, und den Aufruf dann wiederholen.", true)
}

// toolFailure hat dasselbe Format wie die Fehler der App.
func toolFailure(code, message, next string, retry bool) *mcp.CallToolResult {
	body, _ := json.Marshal(map[string]any{"fehler": map[string]any{
		"code": code, "nachricht": message, "naechster_schritt": next, "wiederholbar": retry,
	}})
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: string(body)}}}
}
