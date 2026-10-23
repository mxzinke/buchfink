package wailsbridge

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/buchfink/buchfink/internal/mcpserver"
	"github.com/buchfink/buchfink/internal/mcpserver/oauth"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Verbundene Apps: der MCP-Server läuft im Prozess der App. Die Bridge
// startet ihn, bringt Zustimmungsanfragen als Ereignis ins Fenster und nimmt
// die Entscheidung entgegen. Zugestimmt wird nur hier, nie über HTTP.

// Ereignisse an die Oberfläche.
const (
	// EventAppRequest trägt eine oauth.ConsentRequest.
	EventAppRequest = "buchfink:app-anfrage"
	// EventAppRequestDone trägt die Kennung einer erledigten Anfrage.
	EventAppRequestDone = "buchfink:app-anfrage-erledigt"
)

// newMCPServer richtet den MCP-Server ein; die verbundenen Apps liegen neben
// der App-Konfiguration. Scheitert das, läuft die App ohne ihn weiter; der
// Status nennt den Grund.
func newMCPServer(b *BuchfinkBridge) {
	srv, err := mcpserver.New(mcpserver.Options{
		Host:      b.host,
		StorePath: filepath.Join(b.host.BaseDir(), mcpserver.StoreFile),
		Broker:    windowConsent{},
	})
	if err != nil {
		slog.Warn("MCP-Server nicht verfügbar", "fehler", err)
		b.mcpErr = err.Error()
		return
	}
	b.mcp = srv
}

// windowConsent zeigt Anfragen im Fenster und holt es nach vorn, außer wenn
// eine Anfrage nur die offene derselben App ablöst.
type windowConsent struct{}

func (windowConsent) Present(r oauth.ConsentRequest) {
	app := application.Get()
	if app == nil {
		return
	}
	app.Event.Emit(EventAppRequest, r)
	if r.Replacing {
		return
	}
	for _, w := range app.Window.GetAll() {
		w.UnMinimise()
		w.Show()
		w.Focus()
	}
}

func (windowConsent) Withdraw(id string) {
	if app := application.Get(); app != nil {
		app.Event.Emit(EventAppRequestDone, id)
	}
}

// ServiceStartup ist der Haken, den die Wails-Laufzeit beim Start ruft. Der
// Host ist dann offen; der MCP-Server lauscht ab hier. Ein belegter Port
// hält die App nicht auf, der Status meldet ihn.
func (b *BuchfinkBridge) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	if b.mcp == nil {
		return nil
	}
	if err := b.mcp.Start(); err != nil {
		slog.Warn("MCP-Server startet nicht", "fehler", err)
	}
	return nil
}

// stopMCP hält den MCP-Server vor dem Schließen des Hosts an.
func (b *BuchfinkBridge) stopMCP(ctx context.Context) error {
	if b.mcp == nil {
		return nil
	}
	return b.mcp.Close(ctx)
}

var errNoMCP = errors.New("der MCP-Server ist nicht verfügbar")

func (b *BuchfinkBridge) mcpServer() (*mcpserver.Server, error) {
	if b.mcp == nil {
		if b.mcpErr != "" {
			return nil, errors.New(b.mcpErr)
		}
		return nil, errNoMCP
	}
	return b.mcp, nil
}

// GetMCPServerStatus meldet, ob der Server läuft, auf welchem Port, mit
// welchem Fehler und ob BUCHFINK_MCP_PORT den eingestellten Port übersteuert.
func (b *BuchfinkBridge) GetMCPServerStatus() mcpserver.Status {
	srv, err := b.mcpServer()
	if err != nil {
		_, env := mcpserver.EnvironmentPort()
		return mcpserver.Status{Port: mcpserver.ResolvePort(b.host.Config()), Error: err.Error(), FromEnvironment: env}
	}
	return srv.Status()
}

// SetMCPPort speichert einen neuen Port und startet den Server dort neu.
// 0 setzt die Vorgabe. Ist der Port belegt, bleibt der Server aus.
func (b *BuchfinkBridge) SetMCPPort(port int) (mcpserver.Status, error) {
	srv, err := b.mcpServer()
	if err != nil {
		return b.GetMCPServerStatus(), err
	}
	if err := b.host.SetMCPPort(uiContext(), port); err != nil {
		return srv.Status(), err
	}
	err = srv.Restart(context.Background(), mcpserver.ResolvePort(b.host.Config()))
	return srv.Status(), err
}

// ConnectedApp ist eine zugelassene App in den Einstellungen.
type ConnectedApp struct {
	ClientID   string               `json:"clientId"`
	Name       string               `json:"name"`
	Channel    string               `json:"channel"`
	ApprovedAt string               `json:"approvedAt"`
	LastUsedAt string               `json:"lastUsedAt"`
	Tenants    []ConnectedAppTenant `json:"tenants"`
	Revoked    bool                 `json:"revoked"`
}

// ConnectedAppTenant ist ein Mandant, für den eine App zugelassen ist.
type ConnectedAppTenant struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	GrantedAt string   `json:"grantedAt"`
}

// GetConnectedApps listet die Apps, denen je zugestimmt wurde.
func (b *BuchfinkBridge) GetConnectedApps() ([]ConnectedApp, error) {
	srv, err := b.mcpServer()
	if err != nil {
		return []ConnectedApp{}, nil
	}
	apps, err := srv.ConnectedApps(context.Background())
	if err != nil {
		return []ConnectedApp{}, err
	}
	out := make([]ConnectedApp, 0, len(apps))
	for _, a := range apps {
		view := ConnectedApp{
			ClientID: a.ClientID, Name: a.Name, Channel: a.Channel,
			ApprovedAt: a.ApprovedAt.Format(time.RFC3339), Revoked: a.Revoked,
			Tenants: make([]ConnectedAppTenant, 0, len(a.Tenants)),
		}
		if a.LastUsedAt != nil {
			view.LastUsedAt = a.LastUsedAt.Format(time.RFC3339)
		}
		for _, t := range a.Tenants {
			view.Tenants = append(view.Tenants, ConnectedAppTenant{
				ID: t.ID, Name: t.Name, Scopes: emptyStrings(t.Scopes), GrantedAt: t.GrantedAt.Format(time.RFC3339),
			})
		}
		out = append(out, view)
	}
	return out, nil
}

// RevokeConnectedApp entzieht einer App den Zugriff: alle Zulassungen und
// Tokens. Ihr nächster Aufruf scheitert; eine neue Anmeldung fragt wieder hier.
func (b *BuchfinkBridge) RevokeConnectedApp(clientID string) error {
	srv, err := b.mcpServer()
	if err != nil {
		return err
	}
	return srv.RevokeApp(context.Background(), clientID)
}

// GetPendingAppRequests liefert die offenen Zustimmungsanfragen, falls die
// Oberfläche ein Ereignis verpasst hat.
func (b *BuchfinkBridge) GetPendingAppRequests() []oauth.ConsentRequest {
	srv, err := b.mcpServer()
	if err != nil {
		return []oauth.ConsentRequest{}
	}
	return srv.PendingRequests()
}

// ResolveAppRequest entscheidet eine Zustimmungsanfrage. Die Nonce stammt
// aus dem Ereignis und gilt einmal.
func (b *BuchfinkBridge) ResolveAppRequest(id, nonce string, approve bool) error {
	srv, err := b.mcpServer()
	if err != nil {
		return err
	}
	return srv.Resolve(context.Background(), id, nonce, approve)
}

func emptyStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
