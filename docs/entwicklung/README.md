# Entwicklung

[Dokumentation](../README.md) · [Architektur](architektur.md) ·
[Orientierung im Code](codebase.md) · [MCP-Server](mcp.md) ·
[Mitwirken](../../CONTRIBUTING.md)

## Voraussetzungen

- Go in der Fassung aus [go.mod](../../go.mod)
- Node.js `^20.19.0` oder `>=22.12.0` mit npm
- [Task](https://taskfile.dev) v3
- Wails-v3-CLI in der Fassung aus `go.mod`, für `task dev` und `task build`
- Python 3 für die Prüfskripte

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.9
go install github.com/go-task/task/v3/cmd/task@latest
```

Das Go-Binärverzeichnis (`$HOME/go/bin`) muss im `PATH` liegen. `wails3 doctor`
prüft die Umgebung für den Desktop-Build.

### Linux: GTK 4 oder GTK 3

Wails baut unter Linux mit CGO gegen GTK 4 und WebKitGTK 6.0. Mit dem Build-Tag
`gtk3` nimmt es GTK 3 und WebKitGTK 4.1, das auf älteren Distributionen und in
vielen CI-Umgebungen vorhanden ist.

| Stapel | Pakete (Debian, Ubuntu) | Tag |
|---|---|---|
| GTK 4 | `libgtk-4-dev libwebkitgtk-6.0-dev` | keiner |
| GTK 3 | `libgtk-3-dev libwebkit2gtk-4.1-dev` | `gtk3` |

Mit GTK 3 bekommt jeder Befehl, der `internal/wailsbridge` oder `main.go`
übersetzt, das Tag:

```sh
GOFLAGS=-tags=gtk3 task test:bridge
task build EXTRA_TAGS=gtk3
EXTRA_TAGS=gtk3 task dev
```

`task test` und `task vet` übersetzen die Bridge nicht und brauchen weder GTK
noch das Tag.

## Lokal starten

```sh
git clone https://github.com/mxzinke/buchfink.git
cd buchfink
npm --prefix frontend ci
task dev
```

`task dev` startet Wails mit [build/config.yml](../../build/config.yml) und baut
bei Änderungen neu. `npm --prefix frontend run dev` startet nur den Vite-Server;
für Aufrufe des Go-Backends braucht es die laufende Wails-Anwendung.

Der MCP-Server lauscht im Entwicklungsbetrieb auf demselben Port wie die
installierte App. `BUCHFINK_MCP_PORT=7466 task dev` weicht aus, ohne die
Einstellung zu ändern.

Ein frisches `go build ./...` scheitert, solange `frontend/dist` fehlt, weil
`main.go` das Frontend einbettet. `npm --prefix frontend run build` legt es an.

## Änderungen prüfen

```sh
task check
npm --prefix frontend test
npm --prefix frontend run build
```

`task check` meldet unformatierte Go-Dateien und führt `go vet`, die Go-Tests
und die Prüfskripte für Gestaltung, Bridge, Hilfetexte, Abschlussbausteine und
Dokumentationslinks aus. Formatieren müssen Sie selbst (`gofmt -w`). Der Frontend-Build prüft auch
TypeScript.

| Aufgabe | Wann |
|---|---|
| `task test` | Go-Tests aller Pakete unter `internal/` und `cmd/` außer der Bridge |
| `task test:bridge` | Bridge-Tests; brauchen GTK, siehe oben |
| `task test:ebilanz-xsd` | E-Bilanz-Instanzen gegen die amtlichen Schemata; die Schemata liegen nicht im Repository, siehe `internal/ebilanz/xsd_test.go` |
| `task check:clicks` | Klickweg von der Bilanz zum Beleg, braucht Chromium für Playwright ([Prüfszenarien](pruefszenarien.md)) |
| `task build` | Desktop-Build für das eigene Betriebssystem nach `bin/` |
| `task package` | Installationspaket für das eigene Betriebssystem |

Die Programmfassung setzt der Build aus `git describe --tags`. Ohne Tag steht
dort der Commit, im Test `dev`.

Die Brücke für Claude Desktop baut man ohne Wails und ohne CGO; die Schritte
stehen unter [packaging/claude-desktop](../../packaging/claude-desktop/README.md).

## Abhängigkeiten und Webseite

Nach Änderungen an Abhängigkeiten erzeugt `task notices` die
[Drittkomponentenhinweise](../../THIRD-PARTY-NOTICES.md) neu. Vorschau und
Veröffentlichung der Projektseite stehen in
[website/README.md](../../website/README.md).
