// Package oauth ist der Autorisierungsserver von Buchfink für verbundene
// Apps (OAuth 2.1 mit PKCE, DCR nach RFC 7591, Metadaten nach RFC 8414 und
// RFC 9728, Ressourcenbindung nach RFC 8707, iss nach RFC 9207). Er läuft im
// selben Prozess wie der MCP-Server; Tokens sind deshalb undurchsichtig und
// stehen nur gehasht im Speicher.
package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"gorm.io/gorm"
)

// Scopes der verbundenen Apps.
const (
	ScopeRead  = "buchfink:lesen"
	ScopeWrite = "buchfink:buchen"
	// offline_access wird angenommen und übergangen: Refresh-Tokens gibt es
	// ohnehin, und manche Clients fragen ihn an.
	scopeOffline = "offline_access"
)

// Pfade des Servers.
const (
	PathMCP                 = "/mcp"
	PathResourceMetadata    = "/.well-known/oauth-protected-resource"
	PathResourceMetadataMCP = "/.well-known/oauth-protected-resource/mcp"
	PathServerMetadata      = "/.well-known/oauth-authorization-server"
	PathRegister            = "/register"
	PathAuthorize           = "/authorize"
	PathAuthorizeWait       = "/authorize/warten"
	PathToken               = "/token"
	PathRevoke              = "/revoke"
)

// Vorgaben der Fristen.
const (
	defaultAccessTTL  = 10 * time.Minute
	defaultRefreshTTL = 30 * 24 * time.Hour
	defaultFamilyTTL  = 180 * 24 * time.Hour
	defaultCodeTTL    = 60 * time.Second
	// defaultRefreshGrace: so lange nach der Ersetzung gilt ein zweites
	// Einlösen desselben Refresh-Tokens als gleichzeitige Erneuerung, nicht
	// als Diebstahl.
	defaultRefreshGrace = 10 * time.Second
	defaultConsentTTL   = 5 * time.Minute
	defaultLongPoll     = 25 * time.Second
	defaultMaxPending   = 3
	registerPerMinute   = 10
	maxBodyBytes        = 64 << 10
)

// Config beschreibt den Autorisierungsserver an einem Port.
type Config struct {
	Port   int
	Store  *Store
	Broker ConsentBroker
	// Tenant liefert den geöffneten Mandanten, für den zugestimmt wird.
	Tenant func() (id, name string)
	// OnApproved meldet eine Zustimmung, etwa fürs Protokoll des Mandanten.
	OnApproved func(ctx context.Context, a Approved)
	// GrantedBy ist der Bearbeiter, der in der App zustimmt.
	GrantedBy string
	Clock     func() time.Time
	Logger    *slog.Logger

	AccessTTL, RefreshTTL, FamilyTTL, CodeTTL, ConsentTTL, LongPoll, RefreshGrace time.Duration
	MaxPending                                                                    int
}

// Approved beschreibt eine erteilte Zustimmung.
type Approved struct {
	ClientID, ClientName, Slug string
	TenantID                   string
	Scopes                     []string
}

// Server ist der Autorisierungsserver.
type Server struct {
	cfg     Config
	issuer  string
	pending *pendingSet

	registerMu   sync.Mutex
	registerSeen []time.Time

	touchMu sync.Mutex
	touched map[string]time.Time
}

// NewServer richtet den Autorisierungsserver für einen Port ein.
func NewServer(cfg Config) *Server {
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Tenant == nil {
		cfg.Tenant = func() (string, string) { return "", "" }
	}
	setDefault(&cfg.AccessTTL, defaultAccessTTL)
	setDefault(&cfg.RefreshTTL, defaultRefreshTTL)
	setDefault(&cfg.FamilyTTL, defaultFamilyTTL)
	setDefault(&cfg.CodeTTL, defaultCodeTTL)
	setDefault(&cfg.ConsentTTL, defaultConsentTTL)
	setDefault(&cfg.LongPoll, defaultLongPoll)
	setDefault(&cfg.RefreshGrace, defaultRefreshGrace)
	if cfg.MaxPending <= 0 {
		cfg.MaxPending = defaultMaxPending
	}
	s := &Server{
		cfg:     cfg,
		issuer:  fmt.Sprintf("http://127.0.0.1:%d", cfg.Port),
		touched: map[string]time.Time{},
	}
	// Ein erledigter Ausgang bleibt so lange wie ein Code gilt.
	s.pending = newPendingSet(cfg.MaxPending, cfg.CodeTTL, s.now)
	if err := cfg.Store.prune(context.Background(), s.now()); err != nil {
		cfg.Logger.Warn("verbundene Apps: Aufräumen gescheitert", "fehler", err)
	}
	return s
}

func setDefault(d *time.Duration, v time.Duration) {
	if *d <= 0 {
		*d = v
	}
}

func (s *Server) now() time.Time { return s.cfg.Clock().UTC() }

// Issuer ist die Kennung des Autorisierungsservers. Sie lautet überall gleich,
// auch wenn eine App den Server über localhost erreicht: Clients vergleichen
// sie Zeichen für Zeichen.
func (s *Server) Issuer() string { return s.issuer }

// HostAllowed meldet, ob ein Host-Kopf diesen Server bezeichnet. Alles
// andere ist ein Versuch über DNS-Rebinding.
func (s *Server) HostAllowed(host string) bool {
	host = strings.ToLower(host)
	port := strconv.Itoa(s.cfg.Port)
	return host == "127.0.0.1:"+port || host == "localhost:"+port || host == "[::1]:"+port
}

// ResourceFor ist die MCP-Adresse in der Schreibweise, unter der die App den
// Server anspricht. An sie ist ein Token gebunden.
func (s *Server) ResourceFor(host string) string {
	return "http://" + strings.ToLower(host) + PathMCP
}

// ResourceMetadataURL ist die Adresse der Ressourcen-Metadaten für den
// WWW-Authenticate-Kopf.
func (s *Server) ResourceMetadataURL(host string) string {
	return "http://" + strings.ToLower(host) + PathResourceMetadataMCP
}

// canonicalResource prüft einen resource-Parameter. Erlaubt sind die drei
// Schreibweisen dieses Servers; ohne Angabe gilt die mit 127.0.0.1.
func (s *Server) canonicalResource(raw string) (string, bool) {
	if raw == "" {
		return s.issuer + PathMCP, true
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", false
	}
	if strings.TrimSuffix(u.Path, "/") != PathMCP || !s.HostAllowed(u.Host) {
		return "", false
	}
	return s.ResourceFor(u.Host), true
}

func supportedScopes() []string { return []string{ScopeRead, ScopeWrite} }

// parseScopes prüft eine angefragte Scope-Liste. Ohne Angabe gelten beide.
func parseScopes(raw string) ([]string, error) {
	var out []string
	for _, sc := range strings.Fields(raw) {
		switch sc {
		case ScopeRead, ScopeWrite:
			if !slices.Contains(out, sc) {
				out = append(out, sc)
			}
		case scopeOffline:
		default:
			return nil, fmt.Errorf("unbekannter Scope %q", sc)
		}
	}
	if len(out) == 0 {
		return supportedScopes(), nil
	}
	slices.Sort(out)
	return out, nil
}

// ProtectedResourceMetadata liefert die Metadaten nach RFC 9728.
func (s *Server) ProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"resource":                 s.ResourceFor(r.Host),
		"authorization_servers":    []string{s.issuer},
		"scopes_supported":         supportedScopes(),
		"bearer_methods_supported": []string{"header"},
		"resource_name":            "Buchfink",
	})
}

// AuthorizationServerMetadata liefert die Metadaten nach RFC 8414.
func (s *Server) AuthorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                         s.issuer,
		"authorization_endpoint":                         s.issuer + PathAuthorize,
		"token_endpoint":                                 s.issuer + PathToken,
		"registration_endpoint":                          s.issuer + PathRegister,
		"revocation_endpoint":                            s.issuer + PathRevoke,
		"response_types_supported":                       []string{"code"},
		"response_modes_supported":                       []string{"query"},
		"grant_types_supported":                          []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":               []string{"S256"},
		"token_endpoint_auth_methods_supported":          []string{"none"},
		"revocation_endpoint_auth_methods_supported":     []string{"none"},
		"scopes_supported":                               append(supportedScopes(), scopeOffline),
		"authorization_response_iss_parameter_supported": true,
	})
}

// --- Registrierung (RFC 7591) ---------------------------------------------

type registration struct {
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	ClientName              string   `json:"client_name"`
	SoftwareID              string   `json:"software_id"`
	SoftwareVersion         string   `json:"software_version"`
}

// Register nimmt eine App auf. Nur öffentliche Clients mit Rücksprung auf
// diesen Rechner; client_uri und logo_uri werden nie abgerufen.
func (s *Server) Register(w http.ResponseWriter, r *http.Request) {
	if mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt != "application/json" {
		writeOAuthError(w, &oauthError{Code: "invalid_client_metadata", Description: "erwartet application/json", Status: http.StatusBadRequest})
		return
	}
	var req registration
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		writeOAuthError(w, &oauthError{Code: "invalid_client_metadata", Description: "die Angaben sind kein gültiges JSON oder zu groß", Status: http.StatusBadRequest})
		return
	}
	if err := checkRegistration(&req); err != nil {
		writeOAuthError(w, err)
		return
	}
	if !s.allowRegistration() {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{
			"error": "invalid_client_metadata", "error_description": "zu viele Registrierungen; bitte in einer Minute erneut versuchen"})
		return
	}
	uris, _ := json.Marshal(req.RedirectURIs)
	client := &Client{
		ID: newSecret(""), Name: sanitizeName(req.ClientName), RedirectURIs: string(uris),
		SoftwareID: truncate(req.SoftwareID, 200), SoftwareVersion: truncate(req.SoftwareVersion, 100),
		CreatedAt: s.now(),
	}
	if err := s.cfg.Store.CreateClient(r.Context(), client); err != nil {
		s.cfg.Logger.Error("verbundene Apps: Registrierung nicht gespeichert", "fehler", err)
		writeOAuthError(w, &oauthError{Code: "server_error", Description: "die Registrierung ließ sich nicht speichern", Status: http.StatusInternalServerError})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"client_id":                  client.ID,
		"client_id_issued_at":        client.CreatedAt.Unix(),
		"client_name":                client.Name,
		"redirect_uris":              req.RedirectURIs,
		"grant_types":                req.GrantTypes,
		"response_types":             req.ResponseTypes,
		"token_endpoint_auth_method": "none",
	})
}

func checkRegistration(req *registration) *oauthError {
	bad := func(code, desc string) *oauthError {
		return &oauthError{Code: code, Description: desc, Status: http.StatusBadRequest}
	}
	if len(req.RedirectURIs) == 0 || len(req.RedirectURIs) > 5 {
		return bad("invalid_redirect_uri", "erwartet ein bis fünf redirect_uris")
	}
	for _, uri := range req.RedirectURIs {
		if err := checkRedirectURI(uri); err != nil {
			return bad("invalid_redirect_uri", err.Error())
		}
	}
	if m := req.TokenEndpointAuthMethod; m != "" && m != "none" {
		return bad("invalid_client_metadata", "nur öffentliche Clients (token_endpoint_auth_method none)")
	}
	if len(req.GrantTypes) == 0 {
		req.GrantTypes = []string{"authorization_code", "refresh_token"}
	}
	for _, g := range req.GrantTypes {
		if g != "authorization_code" && g != "refresh_token" {
			return bad("invalid_client_metadata", fmt.Sprintf("grant_type %q wird nicht unterstützt", g))
		}
	}
	if len(req.ResponseTypes) == 0 {
		req.ResponseTypes = []string{"code"}
	}
	for _, rt := range req.ResponseTypes {
		if rt != "code" {
			return bad("invalid_client_metadata", fmt.Sprintf("response_type %q wird nicht unterstützt", rt))
		}
	}
	return nil
}

func truncate(s string, n int) string {
	s = sanitizeName(s)
	if s == "Unbenannte App" {
		return ""
	}
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// allowRegistration begrenzt die angenommenen Registrierungen auf zehn je
// Minute; jede legt eine Zeile an.
func (s *Server) allowRegistration() bool {
	s.registerMu.Lock()
	defer s.registerMu.Unlock()
	now := s.now()
	kept := s.registerSeen[:0]
	for _, t := range s.registerSeen {
		if now.Sub(t) < time.Minute {
			kept = append(kept, t)
		}
	}
	s.registerSeen = kept
	if len(kept) >= registerPerMinute {
		return false
	}
	s.registerSeen = append(s.registerSeen, now)
	return true
}

// --- Autorisierung ---------------------------------------------------------

// Authorize prüft die Anfrage, legt sie der App vor und zeigt im Browser die
// Warteseite. Unbekannte App und fremder Rücksprung enden auf einer
// Fehlerseite ohne Weiterleitung.
func (s *Server) Authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	for key, values := range q {
		if len(values) > 1 {
			errorPage(w, http.StatusBadRequest, "Anmeldung nicht möglich", fmt.Sprintf("Der Parameter %s ist doppelt angegeben.", key))
			return
		}
	}
	client, err := s.cfg.Store.Client(r.Context(), q.Get("client_id"))
	if err != nil {
		errorPage(w, http.StatusInternalServerError, "Anmeldung nicht möglich", "Die verbundenen Apps ließen sich nicht lesen.")
		return
	}
	if client == nil {
		errorPage(w, http.StatusBadRequest, "Unbekannte App",
			"Diese App ist bei Buchfink nicht (mehr) registriert, etwa weil die Liste der verbundenen Apps zurückgesetzt wurde oder die App nie zugelassen war. "+
				"Eine neue Anmeldung mit derselben Kennung führt immer hierher.",
			"Bitte die gespeicherte Anmeldung in der App verwerfen und neu beginnen: in Claude Code mit „claude mcp logout <name>“, in anderen Apps über deren Abmelden.",
			"Für die Buchfink-Brücke in Claude Desktop die Dateien im Ordner „Buchfink/mcp-bruecke“ des Benutzer-Konfigurationsordners löschen.")
		return
	}
	var registered []string
	_ = json.Unmarshal([]byte(client.RedirectURIs), &registered)
	redirectURI := q.Get("redirect_uri")
	if redirectURI == "" || !matchRedirect(registered, redirectURI) {
		errorPage(w, http.StatusBadRequest, "Anmeldung nicht möglich", "Die Rücksprungadresse gehört nicht zu dieser App.")
		return
	}
	state := q.Get("state")
	fail := func(code, desc string) { s.redirectError(w, r, redirectURI, state, code, desc) }

	if q.Get("response_type") != "code" {
		fail("unsupported_response_type", "nur response_type=code")
		return
	}
	if q.Get("code_challenge_method") != "S256" {
		fail("invalid_request", "PKCE mit code_challenge_method=S256 ist Pflicht")
		return
	}
	challenge := q.Get("code_challenge")
	if !validPKCEValue(challenge) {
		fail("invalid_request", "code_challenge fehlt oder ist ungültig")
		return
	}
	if len(state) > 1000 {
		fail("invalid_request", "state ist zu lang")
		return
	}
	scopes, err := parseScopes(q.Get("scope"))
	if err != nil {
		fail("invalid_scope", err.Error())
		return
	}
	resource, ok := s.canonicalResource(q.Get("resource"))
	if !ok {
		fail("invalid_target", "resource bezeichnet nicht den MCP-Server von Buchfink")
		return
	}

	now := s.now()
	tenantID, tenantName := s.cfg.Tenant()
	item := &pending{
		req: ConsentRequest{
			ID: newSecret("")[:22], Nonce: newSecret(""),
			ClientName: client.Name, ClientID: client.ID, ClientRegisteredAt: client.CreatedAt,
			KnownSince:  client.FirstApprovedAt,
			RedirectURI: redirectURI, RedirectTo: redirectHostPort(redirectURI),
			Scopes: scopes, TenantID: tenantID, TenantName: tenantName,
			UserCode: newUserCode(), ExpiresAt: now.Add(s.cfg.ConsentTTL),
		},
		redirectURI: redirectURI, state: state, challenge: challenge, resource: resource,
		done: make(chan struct{}), created: now,
	}
	item.timer = time.AfterFunc(s.cfg.ConsentTTL, func() {
		if s.pending.finish(item, false, "", "nicht rechtzeitig bestätigt", false) && s.cfg.Broker != nil {
			s.cfg.Broker.Withdraw(item.req.ID)
		}
	})
	superseded, err := s.pending.add(item)
	switch {
	case errors.Is(err, ErrTooSoon):
		item.timer.Stop()
		errorPage(w, http.StatusTooManyRequests, "Bitte einen Moment warten",
			"Diese App hat gerade erst um Zugriff gebeten. Bitte einige Sekunden warten und die Anmeldung dann in der App neu beginnen.")
		return
	case err != nil:
		item.timer.Stop()
		errorPage(w, http.StatusTooManyRequests, "Zu viele offene Anmeldungen",
			"In Buchfink warten schon mehrere Anmeldungen auf Ihre Entscheidung. Bitte entscheiden Sie diese zuerst und versuchen Sie es dann erneut.")
		return
	}
	if s.cfg.Broker != nil {
		for _, id := range superseded {
			s.cfg.Broker.Withdraw(id)
		}
		s.cfg.Broker.Present(item.req)
	}
	writePage(w, http.StatusOK, waitingPage(item, s.waitURL(item)))
}

func (s *Server) waitURL(item *pending) string {
	return PathAuthorizeWait + "?anfrage=" + url.QueryEscape(item.req.ID)
}

// validPKCEValue prüft Verifier und Challenge nach RFC 7636 §4.1.
func validPKCEValue(v string) bool {
	if len(v) < 43 || len(v) > 128 {
		return false
	}
	for _, c := range v {
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		case c == '-', c == '.', c == '_', c == '~':
		default:
			return false
		}
	}
	return true
}

func (s *Server) redirectError(w http.ResponseWriter, r *http.Request, redirectURI, state, code, desc string) {
	params := url.Values{"error": {code}, "error_description": {desc}}
	s.redirect(w, r, redirectURI, state, params)
}

// redirect schickt den Browser zur App zurück, mit state und iss (RFC 9207).
func (s *Server) redirect(w http.ResponseWriter, r *http.Request, redirectURI, state string, params url.Values) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		errorPage(w, http.StatusBadRequest, "Anmeldung nicht möglich", "Die Rücksprungadresse ist ungültig.")
		return
	}
	q := u.Query()
	for k, v := range params {
		q[k] = v
	}
	if state != "" {
		q.Set("state", state)
	}
	q.Set("iss", s.issuer)
	u.RawQuery = q.Encode()
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Redirect(w, r, u.String(), http.StatusFound)
}

// AuthorizeWait wartet bis zu LongPoll auf die Entscheidung und leitet dann
// zur App zurück; sonst zeigt es die Warteseite erneut.
func (s *Server) AuthorizeWait(w http.ResponseWriter, r *http.Request) {
	item := s.pending.get(r.URL.Query().Get("anfrage"))
	if item == nil {
		errorPage(w, http.StatusBadRequest, "Anmeldung abgelaufen",
			"Diese Anmeldung ist unbekannt oder abgelaufen. Bitte die Anmeldung in der App neu beginnen.")
		return
	}
	timer := time.NewTimer(s.cfg.LongPoll)
	defer timer.Stop()
	select {
	case <-item.done:
	case <-timer.C:
		writePage(w, http.StatusOK, waitingPage(item, s.waitURL(item)))
		return
	case <-r.Context().Done():
		return
	}
	if !s.pending.take(item) {
		errorPage(w, http.StatusOK, "Anmeldung abgeschlossen", "Diese Anmeldung ist schon abgeschlossen.")
		return
	}
	if !item.approved {
		s.redirectError(w, r, item.redirectURI, item.state, "access_denied", item.reason)
		return
	}
	s.redirect(w, r, item.redirectURI, item.state, url.Values{"code": {item.code}})
}

// Pending liefert die offenen Anfragen für die App, falls sie ein Ereignis
// verpasst hat.
func (s *Server) Pending() []ConsentRequest { return s.pending.open() }

// Resolve entscheidet eine Anfrage. Nur die App ruft das, mit der Nonce aus
// der Anfrage; sie gilt einmal.
func (s *Server) Resolve(ctx context.Context, id, nonce string, approve bool) error {
	item := s.pending.get(id)
	if item == nil || isDone(item) {
		return ErrUnknownRequest
	}
	if !equalSecret(item.req.Nonce, nonce) {
		return ErrInvalidNonce
	}
	if !s.now().Before(item.req.ExpiresAt) {
		return ErrExpiredRequest
	}
	if !approve {
		if !s.pending.finish(item, false, "", "in Buchfink abgelehnt", false) {
			return ErrUnknownRequest
		}
		s.withdraw(item)
		return nil
	}
	if !s.pending.claim(item) {
		return ErrUnknownRequest
	}
	code, client, err := s.issueCode(ctx, item)
	if err != nil {
		s.pending.finish(item, false, "", "die Zustimmung ließ sich nicht speichern", true)
		s.withdraw(item)
		return err
	}
	s.pending.finish(item, true, code, "", true)
	s.withdraw(item)
	if s.cfg.OnApproved != nil && client.Slug != nil {
		s.cfg.OnApproved(ctx, Approved{
			ClientID: client.ID, ClientName: client.Name, Slug: *client.Slug,
			TenantID: item.req.TenantID, Scopes: item.req.Scopes,
		})
	}
	return nil
}

func (s *Server) withdraw(item *pending) {
	if s.cfg.Broker != nil {
		s.cfg.Broker.Withdraw(item.req.ID)
	}
}

func (s *Server) issueCode(ctx context.Context, item *pending) (string, *Client, error) {
	now := s.now()
	code := newSecret(prefixCode)
	scopes := strings.Join(item.req.Scopes, " ")
	family := Family{
		ID: newSecret(""), ClientID: item.req.ClientID, Scopes: scopes, Resource: item.resource,
		CreatedAt: now, ExpiresAt: now.Add(s.cfg.FamilyTTL),
	}
	client, err := s.cfg.Store.approve(ctx, approval{
		ClientID: item.req.ClientID, TenantID: item.req.TenantID, TenantName: item.req.TenantName,
		GrantedBy: s.cfg.GrantedBy, At: now, Family: family,
		Code: Code{
			Hash: hashSecret(code), ClientID: item.req.ClientID, RedirectURI: item.redirectURI,
			Challenge: item.challenge, Scopes: scopes, Resource: item.resource,
			FamilyID: family.ID, ExpiresAt: now.Add(s.cfg.CodeTTL),
		},
	})
	if err != nil {
		return "", nil, err
	}
	return code, client, nil
}

// --- Token -----------------------------------------------------------------

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// Token stellt Tokens aus: gegen einen Code oder ein Refresh-Token.
func (s *Server) Token(w http.ResponseWriter, r *http.Request) {
	form, oerr := readForm(w, r)
	if oerr != nil {
		writeOAuthError(w, oerr)
		return
	}
	var (
		resp *tokenResponse
		err  error
	)
	switch form.Get("grant_type") {
	case "authorization_code":
		resp, err = s.exchangeCode(r.Context(), form)
	case "refresh_token":
		resp, err = s.refresh(r.Context(), form)
	default:
		writeOAuthError(w, &oauthError{Code: "unsupported_grant_type", Description: "erwartet authorization_code oder refresh_token", Status: http.StatusBadRequest})
		return
	}
	if err != nil {
		var oe *oauthError
		if errors.As(err, &oe) {
			writeOAuthError(w, oe)
			return
		}
		s.cfg.Logger.Error("verbundene Apps: Token nicht ausgestellt", "fehler", err)
		writeOAuthError(w, &oauthError{Code: "server_error", Description: "das Token ließ sich nicht ausstellen", Status: http.StatusInternalServerError})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// readForm liest einen Formular-Body; jeder Parameter höchstens einmal.
func readForm(w http.ResponseWriter, r *http.Request) (url.Values, *oauthError) {
	if mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt != "application/x-www-form-urlencoded" {
		return nil, errInvalidRequest("erwartet application/x-www-form-urlencoded")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := r.ParseForm(); err != nil {
		return nil, errInvalidRequest("der Inhalt ließ sich nicht lesen")
	}
	for key, values := range r.PostForm {
		if len(values) > 1 {
			return nil, errInvalidRequest(fmt.Sprintf("der Parameter %s ist doppelt angegeben", key))
		}
	}
	return r.PostForm, nil
}

func (s *Server) exchangeCode(ctx context.Context, form url.Values) (*tokenResponse, error) {
	clientID := form.Get("client_id")
	if clientID == "" {
		return nil, errInvalidClient("client_id fehlt")
	}
	client, err := s.cfg.Store.Client(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errInvalidClient("unbekannte App")
	}
	code, verifier := form.Get("code"), form.Get("code_verifier")
	if code == "" {
		return nil, errInvalidRequest("code fehlt")
	}
	if !validPKCEValue(verifier) {
		return nil, errInvalidGrant("code_verifier fehlt oder ist ungültig")
	}
	resourceParam := form.Get("resource")
	now := s.now()
	var resp *tokenResponse
	err = s.cfg.Store.redeemCode(ctx, hashSecret(code), now, func(c *Code) error {
		if c.ClientID != clientID {
			return errInvalidGrant("der Code gehört zu einer anderen App")
		}
		if c.RedirectURI != form.Get("redirect_uri") {
			return errInvalidGrant("redirect_uri stimmt nicht mit der Anfrage überein")
		}
		if !equalSecret(pkceS256(verifier), c.Challenge) {
			return errInvalidGrant("code_verifier passt nicht zur code_challenge")
		}
		if resourceParam != "" {
			if res, ok := s.canonicalResource(resourceParam); !ok || res != c.Resource {
				return errInvalidTarget("resource stimmt nicht mit der Anfrage überein")
			}
		}
		return nil
	}, func(tx *gorm.DB, family *Family) error {
		resp, err = s.issueTokens(tx, family, family.Scopes, now)
		return err
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Server) refresh(ctx context.Context, form url.Values) (*tokenResponse, error) {
	clientID := form.Get("client_id")
	if clientID == "" {
		return nil, errInvalidClient("client_id fehlt")
	}
	// Eine unbekannte App heißt invalid_client, nicht invalid_grant: dann
	// verwirft der Client seine Registrierung und meldet sich neu an, statt
	// mit der alten Kennung auf der Fehlerseite von /authorize zu landen.
	client, err := s.cfg.Store.Client(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errInvalidClient("unbekannte App; bitte neu registrieren")
	}
	refresh := form.Get("refresh_token")
	if refresh == "" {
		return nil, errInvalidRequest("refresh_token fehlt")
	}
	requested := strings.Fields(form.Get("scope"))
	resourceParam := form.Get("resource")
	now := s.now()
	var resp *tokenResponse
	err = s.cfg.Store.rotateRefresh(ctx, hashSecret(refresh), now, s.cfg.RefreshGrace, func(t *Token, f *Family) error {
		if t.ClientID != clientID {
			return errInvalidGrant("das Refresh-Token gehört zu einer anderen App")
		}
		if resourceParam != "" {
			if res, ok := s.canonicalResource(resourceParam); !ok || res != t.Resource {
				return errInvalidTarget("resource stimmt nicht mit der Anmeldung überein")
			}
		}
		granted := strings.Fields(t.Scopes)
		for _, sc := range requested {
			if sc != scopeOffline && !slices.Contains(granted, sc) {
				return errInvalidScope(fmt.Sprintf("der Scope %q wurde nicht erteilt", sc))
			}
		}
		return nil
	}, func(tx *gorm.DB, family *Family, old *Token) error {
		scopes := old.Scopes
		if narrowed := narrowScopes(requested); narrowed != "" {
			scopes = narrowed
		}
		var err error
		resp, err = s.issueTokens(tx, family, scopes, now)
		return err
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func narrowScopes(requested []string) string {
	var out []string
	for _, sc := range requested {
		if sc != scopeOffline && !slices.Contains(out, sc) {
			out = append(out, sc)
		}
	}
	slices.Sort(out)
	return strings.Join(out, " ")
}

// issueTokens legt ein Access- und ein Refresh-Token in der Kette an.
func (s *Server) issueTokens(tx *gorm.DB, family *Family, scopes string, now time.Time) (*tokenResponse, error) {
	access, refresh := newSecret(prefixAccess), newSecret(prefixRefresh)
	refreshUntil := now.Add(s.cfg.RefreshTTL)
	if family.ExpiresAt.Before(refreshUntil) {
		refreshUntil = family.ExpiresAt
	}
	rows := []Token{
		{Hash: hashSecret(access), Kind: kindAccess, ClientID: family.ClientID, FamilyID: family.ID,
			Scopes: scopes, Resource: family.Resource, IssuedAt: now, ExpiresAt: now.Add(s.cfg.AccessTTL)},
		{Hash: hashSecret(refresh), Kind: kindRefresh, ClientID: family.ClientID, FamilyID: family.ID,
			Scopes: scopes, Resource: family.Resource, IssuedAt: now, ExpiresAt: refreshUntil},
	}
	if err := tx.Create(&rows).Error; err != nil {
		return nil, err
	}
	return &tokenResponse{
		AccessToken: access, TokenType: "Bearer", ExpiresIn: int(s.cfg.AccessTTL.Seconds()),
		RefreshToken: refresh, Scope: scopes,
	}, nil
}

// Revoke widerruft ein Token (RFC 7009). Die Antwort ist immer 200, auch für
// unbekannte Tokens.
func (s *Server) Revoke(w http.ResponseWriter, r *http.Request) {
	form, oerr := readForm(w, r)
	if oerr != nil {
		writeOAuthError(w, oerr)
		return
	}
	if token := form.Get("token"); token != "" {
		if err := s.cfg.Store.revokeToken(r.Context(), hashSecret(token), form.Get("client_id"), s.now()); err != nil {
			s.cfg.Logger.Error("verbundene Apps: Widerruf gescheitert", "fehler", err)
			writeOAuthError(w, &oauthError{Code: "server_error", Description: "der Widerruf ließ sich nicht speichern", Status: http.StatusInternalServerError})
			return
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
}

// --- Prüfung am MCP-Endpunkt ------------------------------------------------

// Verify prüft ein Access-Token bei jedem Aufruf: bekannt, nicht abgelaufen,
// nicht widerrufen, Kette gültig und an genau die Adresse gebunden, unter der
// die App gerade anfragt.
func (s *Server) Verify(ctx context.Context, token string, req *http.Request) (*auth.TokenInfo, error) {
	now := s.now()
	tok, client, err := s.cfg.Store.accessToken(ctx, hashSecret(token), now)
	if err != nil {
		return nil, err
	}
	if tok == nil || client == nil || client.Slug == nil {
		return nil, fmt.Errorf("%w: das Token ist unbekannt, abgelaufen oder widerrufen", auth.ErrInvalidToken)
	}
	if tok.Resource != s.ResourceFor(req.Host) {
		return nil, fmt.Errorf("%w: das Token gilt für eine andere Adresse", auth.ErrInvalidToken)
	}
	s.touch(ctx, client.ID, now)
	return &auth.TokenInfo{
		Scopes:     strings.Fields(tok.Scopes),
		Expiration: tok.ExpiresAt,
		UserID:     client.ID,
		Extra:      map[string]any{"slug": *client.Slug, "name": client.Name},
	}, nil
}

// touch schreibt den letzten Zugriff höchstens einmal je Minute.
func (s *Server) touch(ctx context.Context, clientID string, now time.Time) {
	s.touchMu.Lock()
	last, ok := s.touched[clientID]
	if ok && now.Sub(last) < time.Minute {
		s.touchMu.Unlock()
		return
	}
	s.touched[clientID] = now
	s.touchMu.Unlock()
	if err := s.cfg.Store.touch(ctx, clientID, now); err != nil {
		s.cfg.Logger.Warn("verbundene Apps: letzter Zugriff nicht gespeichert", "fehler", err)
	}
}
