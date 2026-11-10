// Command buchfink-mcp verbindet MCP-Clients, die nur stdio sprechen (Claude
// Desktop über die MCPB-Erweiterung), mit dem MCP-Server der laufenden
// Buchfink-App auf diesem Rechner. Es kennt weder Wails noch die Bücher:
// alles Fachliche bleibt in der App.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// version setzt der Bau (-ldflags "-X main.version=…").
var version = "dev"

const defaultEndpoint = "http://127.0.0.1:7465/mcp"

func main() {
	endpoint := flag.String("url", envOr("BUCHFINK_MCP_URL", defaultEndpoint), "Adresse des MCP-Servers der Buchfink-App")
	stateDir := flag.String("state-dir", os.Getenv("BUCHFINK_MCP_STATE_DIR"), "Ordner für Anmeldung und Werkzeugliste")
	clientName := flag.String("client-name", "Claude Desktop (Buchfink-Brücke)", "Name, unter dem die Brücke sich in Buchfink anmeldet")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := checkEndpoint(*endpoint); err != nil {
		logger.Error("ungültige Adresse", "fehler", err)
		os.Exit(2)
	}
	dir := *stateDir
	if dir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			logger.Error("kein Konfigurationsordner", "fehler", err)
			os.Exit(1)
		}
		dir = filepath.Join(base, "Buchfink", "mcp-bruecke")
	}
	b, err := newBridge(config{Endpoint: *endpoint, StateDir: dir, ClientName: *clientName, Logger: logger})
	if err != nil {
		logger.Error("Brücke nicht eingerichtet", "fehler", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := b.run(ctx, &mcp.StdioTransport{}); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("Brücke beendet", "fehler", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// checkEndpoint lässt nur Buchfink auf diesem Rechner zu: Tokens der Brücke
// gehen nie an einen anderen Server.
func checkEndpoint(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" {
		return fmt.Errorf("%s: erwartet http://127.0.0.1:<port>/mcp", raw)
	}
	switch strings.ToLower(u.Hostname()) {
	case "127.0.0.1", "localhost", "::1":
	default:
		return fmt.Errorf("%s: Buchfink läuft nur auf diesem Rechner (127.0.0.1)", raw)
	}
	if u.Port() == "" || u.Path == "" {
		return fmt.Errorf("%s: Port und Pfad fehlen, etwa http://127.0.0.1:7465/mcp", raw)
	}
	return nil
}
