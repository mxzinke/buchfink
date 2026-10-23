package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"golang.org/x/oauth2"
)

// Die Brücke ist ihr eigener OAuth-Client: Claude Desktop startet für
// Erweiterungen keinen Anmeldeablauf. Entdeckung und Registrierung nutzen die
// Bausteine des Go-SDK (oauthex), Tokenablage und Erneuerung macht die Brücke
// selbst, weil sie die Anmeldung über Neustarts und zwischen Prozessen teilt.

// errLoginRequired: es gibt keine gültige Anmeldung; erst ein Mensch kann
// eine neue erteilen.
var errLoginRequired = errors.New("die Brücke ist bei Buchfink nicht angemeldet")

// errLoginDenied: in Buchfink wurde abgelehnt.
var errLoginDenied = errors.New("die Anmeldung wurde in Buchfink abgelehnt")

// oauthClient erfüllt auth.OAuthHandler für den Streamable-HTTP-Transport.
type oauthClient struct {
	endpoint   string
	clientName string
	store      *stateStore
	http       *http.Client
	open       func(string) error
	logger     *slog.Logger

	// needLogin wird gesetzt, wenn eine Anfrage an fehlender Anmeldung scheiterte.
	needLogin atomic.Bool
	// challenge ist der letzte WWW-Authenticate-Kopf des Servers.
	mu        sync.Mutex
	challenge []string
}

var _ auth.OAuthHandler = (*oauthClient)(nil)

// TokenSource liefert die gespeicherte Anmeldung; ohne eine geht die Anfrage
// ohne Token hinaus und kommt mit 401 zurück.
func (o *oauthClient) TokenSource(context.Context) (oauth2.TokenSource, error) {
	st, err := o.store.load(o.endpoint)
	if err != nil || st.Token == nil {
		return nil, nil
	}
	return o, nil
}

// Token liefert ein gültiges Access-Token und erneuert es bei Bedarf.
func (o *oauthClient) Token() (*oauth2.Token, error) {
	tok, err := o.refresh(context.Background(), "", false)
	if err != nil {
		if errors.Is(err, errLoginRequired) {
			// So geht die Anfrage ohne Token hinaus; der Server antwortet 401
			// und Authorize meldet, dass eine Anmeldung fehlt.
			return nil, &oauth2.RetrieveError{ErrorCode: "invalid_grant", ErrorDescription: err.Error()}
		}
		return nil, err
	}
	return tok, nil
}

// Authorize ruft der Transport bei 401 oder 403. Sie erneuert, wenn sie
// kann, und meldet sonst errLoginRequired: ein Browserfenster öffnet nur die
// Brücke selbst, wenn ein Werkzeug gebraucht wird.
func (o *oauthClient) Authorize(ctx context.Context, req *http.Request, resp *http.Response) error {
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("Buchfink weist die Anfrage ab (403)")
	}
	o.mu.Lock()
	o.challenge = resp.Header.Values("WWW-Authenticate")
	o.mu.Unlock()
	sent := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	if _, err := o.refresh(ctx, sent, true); err == nil {
		return nil
	}
	o.needLogin.Store(true)
	return errLoginRequired
}

func tokenValid(t *tokenData, now time.Time) bool {
	return t != nil && t.AccessToken != "" && now.Add(30*time.Second).Before(t.Expiry)
}

// refresh liefert ein gültiges Token. Unter der Dateisperre liest es die
// Datei neu: hat ein anderer Prozess schon erneuert, nimmt es dessen Token.
// force erneuert auch ein noch gültiges Token, wenn es dem gesendeten
// entspricht (der Server hat es abgelehnt).
func (o *oauthClient) refresh(ctx context.Context, sent string, force bool) (*oauth2.Token, error) {
	unlock, err := o.store.lock()
	if err != nil {
		return nil, err
	}
	defer unlock()
	st, err := o.store.load(o.endpoint)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if st.Token == nil {
		return nil, errLoginRequired
	}
	if tokenValid(st.Token, now) && (!force || st.Token.AccessToken != sent) {
		return oauthToken(st.Token), nil
	}
	if st.Token.RefreshToken == "" || st.ClientID == "" || o.onServer(st.TokenEndpoint) != nil {
		return nil, errLoginRequired
	}
	tok, err := o.tokenRequest(ctx, st.TokenEndpoint, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {st.Token.RefreshToken},
		"client_id": {st.ClientID}, "resource": {o.endpoint},
	})
	if err != nil {
		var re *oauth2.RetrieveError
		if errors.As(err, &re) {
			switch re.ErrorCode {
			case "invalid_grant":
				st.Token = nil
				_ = o.store.save(st)
				return nil, errLoginRequired
			case "invalid_client":
				// Buchfink kennt die Registrierung nicht mehr, etwa nach
				// einem Zurücksetzen: neu registrieren statt mit der alten
				// Kennung an /authorize zu scheitern.
				st.Issuer, st.ClientID, st.ClientApproved, st.TokenEndpoint, st.Token = "", "", false, "", nil
				_ = o.store.save(st)
				return nil, errLoginRequired
			}
		}
		return nil, err
	}
	st.Token = tok
	if err := o.store.save(st); err != nil {
		return nil, err
	}
	o.needLogin.Store(false)
	return oauthToken(tok), nil
}

func oauthToken(t *tokenData) *oauth2.Token {
	return &oauth2.Token{AccessToken: t.AccessToken, TokenType: "Bearer", RefreshToken: t.RefreshToken, Expiry: t.Expiry}
}

// tokenRequest schickt eine Anfrage an den Token-Endpunkt.
func (o *oauthClient) tokenRequest(ctx context.Context, endpoint string, form url.Values) (*tokenData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := o.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		_ = json.Unmarshal(body, &e)
		return nil, &oauth2.RetrieveError{Response: resp, Body: body, ErrorCode: e.Error, ErrorDescription: e.Description}
	}
	var t struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
	}
	if err := json.Unmarshal(body, &t); err != nil || t.AccessToken == "" || !strings.EqualFold(t.TokenType, "bearer") {
		return nil, fmt.Errorf("die Tokenantwort von Buchfink ist unvollständig")
	}
	expiry := time.Now().Add(time.Duration(t.ExpiresIn) * time.Second)
	if t.ExpiresIn <= 0 {
		expiry = time.Now().Add(5 * time.Minute)
	}
	return &tokenData{AccessToken: t.AccessToken, RefreshToken: t.RefreshToken, Expiry: expiry, Scope: t.Scope}, nil
}

// login meldet die Brücke neu an: Entdeckung, Registrierung (einmal je
// Server), Browser, Rücksprung auf 127.0.0.1, Token. Zugestimmt wird im
// Fenster von Buchfink.
func (o *oauthClient) login(ctx context.Context) error {
	prm, err := o.resourceMetadata(ctx)
	if err != nil {
		return err
	}
	if err := o.onServer(prm.AuthorizationServers[0]); err != nil {
		return fmt.Errorf("die Ressourcen-Metadaten nennen eine fremde Adresse: %w", err)
	}
	asm, err := auth.GetAuthServerMetadata(ctx, prm.AuthorizationServers[0], o.http)
	if err != nil || asm == nil {
		return fmt.Errorf("die Anmeldedaten von Buchfink ließen sich nicht lesen: %v", err)
	}
	for _, endpoint := range []string{asm.Issuer, asm.AuthorizationEndpoint, asm.TokenEndpoint} {
		if err := o.onServer(endpoint); err != nil {
			return fmt.Errorf("die Anmeldedaten nennen eine fremde Adresse: %w", err)
		}
	}
	st, err := o.ensureClient(ctx, asm)
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("der Rücksprung der Anmeldung ließ sich nicht öffnen: %w", err)
	}
	defer listener.Close()
	redirect := fmt.Sprintf("http://127.0.0.1:%d/callback", listener.Addr().(*net.TCPAddr).Port)
	verifier := oauth2.GenerateVerifier()
	stateParam := rand.Text()
	query := url.Values{
		"response_type": {"code"}, "client_id": {st.ClientID}, "redirect_uri": {redirect},
		"code_challenge": {oauth2.S256ChallengeFromVerifier(verifier)}, "code_challenge_method": {"S256"},
		"state": {stateParam}, "resource": {o.endpoint},
	}
	if len(prm.ScopesSupported) > 0 {
		query.Set("scope", strings.Join(prm.ScopesSupported, " "))
	}
	authURL := asm.AuthorizationEndpoint + "?" + query.Encode()

	result := make(chan callbackResult, 1)
	srv := &http.Server{ReadHeaderTimeout: 10 * time.Second, Handler: callbackHandler(stateParam, asm, result)}
	go func() { _ = srv.Serve(listener) }()
	defer srv.Close()

	o.logger.Info("Anmeldung bei Buchfink: Browser öffnet sich; bitte im Buchfink-Fenster bestätigen", "adresse", authURL)
	if err := o.open(authURL); err != nil {
		o.logger.Warn("Browser ließ sich nicht öffnen; die Adresse bitte selbst öffnen", "adresse", authURL, "fehler", err)
	}

	var res callbackResult
	select {
	case res = <-result:
	case <-ctx.Done():
		return fmt.Errorf("die Anmeldung wurde nicht rechtzeitig bestätigt: %w", ctx.Err())
	}
	if res.err != nil {
		return res.err
	}
	tok, err := o.tokenRequest(ctx, asm.TokenEndpoint, url.Values{
		"grant_type": {"authorization_code"}, "code": {res.code}, "redirect_uri": {redirect},
		"client_id": {st.ClientID}, "code_verifier": {verifier}, "resource": {o.endpoint},
	})
	if err != nil {
		return fmt.Errorf("der Code ließ sich nicht einlösen: %w", err)
	}

	unlock, err := o.store.lock()
	if err != nil {
		return err
	}
	defer unlock()
	current, err := o.store.load(o.endpoint)
	if err != nil {
		return err
	}
	current.Issuer, current.ClientID, current.TokenEndpoint, current.Token = asm.Issuer, st.ClientID, asm.TokenEndpoint, tok
	current.ClientApproved = true
	if err := o.store.save(current); err != nil {
		return err
	}
	o.needLogin.Store(false)
	return nil
}

// resourceMetadata liest die Ressourcen-Metadaten: zuerst die Adresse aus dem
// letzten WWW-Authenticate, sonst die Standardadresse zur MCP-Adresse.
func (o *oauthClient) resourceMetadata(ctx context.Context) (*oauthex.ProtectedResourceMetadata, error) {
	var candidates []string
	o.mu.Lock()
	challenges, _ := oauthex.ParseWWWAuthenticate(o.challenge)
	o.mu.Unlock()
	for _, c := range challenges {
		if u := c.Params["resource_metadata"]; u != "" && o.onServer(u) == nil {
			candidates = append(candidates, u)
		}
	}
	u, err := url.Parse(o.endpoint)
	if err != nil {
		return nil, err
	}
	u.Path = "/.well-known/oauth-protected-resource" + u.Path
	candidates = append(candidates, u.String())
	var lastErr error
	for _, c := range candidates {
		prm, err := oauthex.GetProtectedResourceMetadata(ctx, c, o.endpoint, o.http)
		if err == nil && prm != nil && len(prm.AuthorizationServers) > 0 {
			return prm, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("Buchfink nennt keinen Anmeldeserver: %v", lastErr)
}

// ensureClient registriert die Brücke einmal je Anmeldeserver.
func (o *oauthClient) ensureClient(ctx context.Context, asm *oauthex.AuthServerMeta) (*state, error) {
	unlock, err := o.store.lock()
	if err != nil {
		return nil, err
	}
	defer unlock()
	st, err := o.store.load(o.endpoint)
	if err != nil {
		return nil, err
	}
	// Eine nie zugelassene Registrierung räumt Buchfink nach 30 Tagen ab;
	// eine neue kostet nichts, weil der Kurzname erst bei der Zustimmung
	// entsteht.
	if st.ClientID != "" && st.Issuer == asm.Issuer && (st.ClientApproved || st.Token != nil) {
		return st, nil
	}
	if asm.RegistrationEndpoint == "" {
		return nil, fmt.Errorf("Buchfink nimmt keine Registrierung an")
	}
	if err := o.onServer(asm.RegistrationEndpoint); err != nil {
		return nil, fmt.Errorf("die Anmeldedaten nennen eine fremde Adresse: %w", err)
	}
	reg, err := oauthex.RegisterClient(ctx, asm.RegistrationEndpoint, &oauthex.ClientRegistrationMetadata{
		ClientName: o.clientName, RedirectURIs: []string{"http://127.0.0.1/callback"},
		GrantTypes: []string{"authorization_code", "refresh_token"}, ResponseTypes: []string{"code"},
		TokenEndpointAuthMethod: "none", ApplicationType: "native",
		SoftwareID: "buchfink-mcp", SoftwareVersion: version,
	}, o.http)
	if err != nil {
		return nil, fmt.Errorf("die Brücke ließ sich bei Buchfink nicht registrieren: %w", err)
	}
	st.Issuer, st.ClientID, st.ClientApproved, st.TokenEndpoint, st.Token = asm.Issuer, reg.ClientID, false, asm.TokenEndpoint, nil
	if err := o.store.save(st); err != nil {
		return nil, err
	}
	return st, nil
}

// onServer prüft eine Adresse aus WWW-Authenticate oder den Metadaten: http
// auf diesem Rechner, derselbe Port wie die Buchfink-Adresse, Host wie dort
// oder 127.0.0.1 (der Aussteller von Buchfink). Belegt ein fremder Prozess
// den Port, während Buchfink zu ist, könnte er sonst jede Adresse nennen,
// und die Brücke öffnete sie (file:, fremde Seiten) oder schickte ihr Tokens.
func (o *oauthClient) onServer(raw string) error {
	endpoint, err := url.Parse(o.endpoint)
	if err != nil {
		return err
	}
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	host := strings.ToLower(u.Hostname())
	if u.Scheme != "http" || u.User != nil || u.Port() != endpoint.Port() ||
		(host != strings.ToLower(endpoint.Hostname()) && host != "127.0.0.1") {
		return fmt.Errorf("%q liegt nicht auf %s", raw, endpoint.Host)
	}
	return nil
}

type callbackResult struct {
	code string
	err  error
}

// callbackHandler nimmt den Rücksprung aus dem Browser an. state und iss
// müssen stimmen (RFC 9207), sonst gilt der Code nicht.
func callbackHandler(wantState string, asm *oauthex.AuthServerMeta, result chan<- callbackResult) http.Handler {
	var once sync.Once
	finish := func(r callbackResult) { once.Do(func() { result <- r }) }
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/callback" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		if q.Get("state") != wantState {
			writeCallbackPage(w, http.StatusBadRequest, "Diese Anmeldung gehört nicht zu dieser Brücke.")
			return
		}
		if iss := q.Get("iss"); iss != asm.Issuer && (iss != "" || asm.AuthorizationResponseIssParameterSupported) {
			writeCallbackPage(w, http.StatusBadRequest, "Der Rücksprung kommt nicht von Buchfink.")
			finish(callbackResult{err: fmt.Errorf("der Rücksprung nennt den Aussteller %q statt %q", iss, asm.Issuer)})
			return
		}
		if e := q.Get("error"); e != "" {
			writeCallbackPage(w, http.StatusOK, "Die Anmeldung wurde nicht erteilt. Sie können dieses Fenster schließen.")
			if e == "access_denied" {
				finish(callbackResult{err: errLoginDenied})
			} else {
				finish(callbackResult{err: fmt.Errorf("Buchfink lehnt die Anmeldung ab: %s %s", e, q.Get("error_description"))})
			}
			return
		}
		code := q.Get("code")
		if code == "" {
			writeCallbackPage(w, http.StatusBadRequest, "Im Rücksprung fehlt der Code.")
			return
		}
		writeCallbackPage(w, http.StatusOK, "Verbunden. Sie können dieses Fenster schließen und zur App zurückkehren.")
		finish(callbackResult{code: code})
	})
}

func writeCallbackPage(w http.ResponseWriter, status int, text string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	fmt.Fprintf(w, `<!doctype html><html lang="de"><meta charset="utf-8"><title>Buchfink</title>`+
		`<body style="font-family:system-ui,sans-serif;max-width:30rem;margin:4rem auto;padding:0 1rem">`+
		`<h1 style="font-size:1.3rem">Buchfink-Brücke</h1><p>%s</p></body></html>`, html.EscapeString(text))
}
