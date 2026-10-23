package wailsbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func mcpBridge(t *testing.T) *BuchfinkBridge {
	t.Helper()
	b := testBridge(t)
	newMCPServer(b)
	if b.mcp == nil {
		t.Fatalf("MCP-Server nicht eingerichtet: %s", b.mcpErr)
	}
	t.Cleanup(func() { _ = b.stopMCP(context.Background()) })
	return b
}

// Der Port steht in der Konfiguration; ein belegter wird gemeldet und nicht
// umgangen.
func TestMCPPortIsStoredAndBusyPortReported(t *testing.T) {
	b := mcpBridge(t)
	port := freePort(t)
	st, err := b.SetMCPPort(port)
	if err != nil || !st.Running || st.Port != port || st.URL != fmt.Sprintf("http://127.0.0.1:%d/mcp", port) {
		t.Fatalf("Port setzen: %+v %v", st, err)
	}
	if got := b.GetAppConfig().MCPPort; got != port {
		t.Errorf("Port in der Konfiguration: %d", got)
	}

	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	st, err = b.SetMCPPort(busy.Addr().(*net.TCPAddr).Port)
	if err == nil || st.Running || !strings.Contains(st.Error, "belegt") {
		t.Errorf("belegter Port: %+v %v", st, err)
	}
	if _, err := b.SetMCPPort(80); err == nil {
		t.Error("ein Port unter 1024 wurde angenommen")
	}
}

// Die Zustimmung läuft über die Bridge: die Anfrage steht offen, die
// Entscheidung mit der Nonce führt im Browser zum Rücksprung.
func TestAppRequestIsResolvedThroughTheBridge(t *testing.T) {
	b := mcpBridge(t)
	port := freePort(t)
	if _, err := b.SetMCPPort(port); err != nil {
		t.Fatal(err)
	}
	base := fmt.Sprintf("http://127.0.0.1:%d", port)

	resp, err := http.Post(base+"/register", "application/json",
		strings.NewReader(`{"client_name":"Claude Code","redirect_uris":["http://127.0.0.1/callback"]}`))
	if err != nil {
		t.Fatal(err)
	}
	var reg map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&reg)
	resp.Body.Close()

	q := url.Values{
		"response_type": {"code"}, "client_id": {reg["client_id"].(string)},
		"redirect_uri":   {"http://127.0.0.1:5000/callback"},
		"code_challenge": {strings.Repeat("a", 43)}, "code_challenge_method": {"S256"}, "state": {"s"},
	}
	resp, err = http.Get(base + "/authorize?" + q.Encode())
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	pending := b.GetPendingAppRequests()
	if len(pending) != 1 || pending[0].ClientName != "Claude Code" || pending[0].TenantName != "Pfennig Ventures GmbH" {
		t.Fatalf("offene Anfragen: %+v", pending)
	}
	if err := b.ResolveAppRequest(pending[0].ID, "falsch", true); err == nil {
		t.Error("falsche Nonce angenommen")
	}
	if err := b.ResolveAppRequest(pending[0].ID, pending[0].Nonce, true); err != nil {
		t.Fatalf("Zustimmung: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err = client.Get(base + "/authorize/warten?anfrage=" + url.QueryEscape(pending[0].ID))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	loc, _ := url.Parse(resp.Header.Get("Location"))
	if resp.StatusCode != http.StatusFound || loc.Query().Get("code") == "" {
		t.Fatalf("Rücksprung: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}

	apps, err := b.GetConnectedApps()
	if err != nil || len(apps) != 1 || apps[0].Channel != "agent:claude-code" || len(apps[0].Tenants) != 1 ||
		apps[0].Tenants[0].Name != "Pfennig Ventures GmbH" || apps[0].ApprovedAt == "" {
		t.Fatalf("verbundene Apps: %+v %v", apps, err)
	}
	if err := b.RevokeConnectedApp(apps[0].ClientID); err != nil {
		t.Fatal(err)
	}
	if apps, _ := b.GetConnectedApps(); len(apps) != 1 || !apps[0].Revoked || len(apps[0].Tenants) != 0 {
		t.Errorf("nach dem Entzug: %+v", apps)
	}
}

// Ohne MCP-Server bleibt die App bedienbar: Listen sind leer, der Status
// nennt den Grund.
func TestBridgeWithoutMCPServer(t *testing.T) {
	b := testBridge(t)
	if apps, err := b.GetConnectedApps(); err != nil || apps == nil || len(apps) != 0 {
		t.Errorf("verbundene Apps: %v %v", apps, err)
	}
	if st := b.GetMCPServerStatus(); st.Running || st.Error == "" {
		t.Errorf("Status: %+v", st)
	}
	if pending := b.GetPendingAppRequests(); pending == nil || len(pending) != 0 {
		t.Errorf("offene Anfragen: %v", pending)
	}
	if err := b.ResolveAppRequest("x", "y", true); err == nil {
		t.Error("Entscheidung ohne Server angenommen")
	}
}
