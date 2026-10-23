# Buchfink für Claude Desktop (MCPB)

Claude Desktop bindet lokale MCP-Server nur über stdio ein. Diese Erweiterung
liefert dafür die Brücke `buchfink-mcp` (`cmd/buchfink-mcp`): sie spricht auf
stdio mit Claude Desktop und über HTTP mit dem MCP-Server der laufenden
Buchfink-App (`http://127.0.0.1:7465/mcp`). Die Brücke ist ein eigenes kleines
Programm ohne Wails und ohne CGO; alles Fachliche bleibt in der App.

Claude Code, Codex, OpenCode und Pi brauchen die Brücke nicht. Sie verbinden
sich direkt mit `http://127.0.0.1:7465/mcp` und melden sich selbst an.

## Bauen

Voraussetzungen: Go aus `go.mod`, Node.js und das MCPB-Werkzeug
(`npm install -g @anthropic-ai/mcpb`). Alle Befehle im Wurzelordner des
Repositorys.

```sh
VERSION=0.3.0
LDFLAGS="-s -w -X main.version=$VERSION"
OUT=packaging/claude-desktop/server
mkdir -p "$OUT"

# macOS: ein Universal-Binary, weil das Manifest nur das Betriebssystem
# unterscheidet und nicht die Architektur.
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o "$OUT/buchfink-mcp-arm64" ./cmd/buchfink-mcp
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "$OUT/buchfink-mcp-amd64" ./cmd/buchfink-mcp
lipo -create -output "$OUT/buchfink-mcp" "$OUT/buchfink-mcp-arm64" "$OUT/buchfink-mcp-amd64"
rm "$OUT/buchfink-mcp-arm64" "$OUT/buchfink-mcp-amd64"

# Windows
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "$OUT/buchfink-mcp.exe" ./cmd/buchfink-mcp

cp build/appicon.png packaging/claude-desktop/icon.png
```

`lipo` gibt es nur unter macOS. Für ein Paket nur für Windows reicht der
Windows-Schritt; `entry_point` im Manifest muss dann trotzdem auf eine
vorhandene Datei zeigen.

Der Release signiert und notarisiert die Brücke nicht. Der arm64-Teil trägt
nur die Ad-hoc-Signatur, die der Go-Linker für darwin/arm64 setzt; `lipo`
lässt sie stehen. Ob Gatekeeper die Brücke in Claude Desktop blockiert, hängt
an der Quarantäne-Markierung der entpackten Dateien und ist auf einem Mac
nicht geprüft. Die Abhilfe für Nutzer steht in
[Claude Desktop](../../docs/nutzung/ki-agenten/claude-desktop.md#wenn-es-nicht-geht).

## Prüfen und packen

```sh
cd packaging/claude-desktop
mcpb validate manifest.json
mcpb pack . buchfink.mcpb
```

Die Fassung im Manifest (`version`) und `VERSION` beim Bauen gehören zur
Fassung der App.

## Installieren und anmelden

1. `buchfink.mcpb` doppelklicken oder in Claude Desktop unter Einstellungen →
   Erweiterungen hineinziehen.
2. Buchfink öffnen.
3. In Claude ein Buchfink-Werkzeug aufrufen, etwa `status`. Beim ersten Mal
   öffnet die Brücke den Browser; dort steht ein Abgleichcode. Buchfink zeigt
   denselben Code und fragt, ob die App zugreifen darf.
4. Nach dem Zulassen den Aufruf wiederholen, falls Claude ihn nicht selbst
   wiederholt.

Ist Buchfink nicht geöffnet, antworten die Werkzeuge sofort mit
`app_nicht_geoeffnet`. Den Zugriff entzieht man in Buchfink; die Brücke fragt
dann beim nächsten Aufruf neu an.

Die Brücke legt Anmeldung und Werkzeugliste unter
`<Benutzerkonfiguration>/Buchfink/mcp-bruecke/` ab (Dateirechte 0600,
Dateisperre zwischen mehreren Prozessen). Unter macOS ist das
`~/Library/Application Support/Buchfink/mcp-bruecke/`, unter Windows
`%AppData%\Buchfink\mcp-bruecke\`.

## Optionen der Brücke

| Option | Umgebung | Vorgabe |
|---|---|---|
| `--url` | `BUCHFINK_MCP_URL` | `http://127.0.0.1:7465/mcp` |
| `--state-dir` | `BUCHFINK_MCP_STATE_DIR` | siehe oben |
| `--client-name` | – | `Claude Desktop (Buchfink-Brücke)` |

`--url` nimmt nur Adressen auf diesem Rechner an. Wer in Buchfink einen
anderen Port einstellt, trägt dieselbe Adresse in Claude Desktop in den
Einstellungen der Erweiterung ein.
