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
`gtk3` nimmt es GTK 3 und WebKitGTK 4.1.

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

`task test:timezones` führt die Go-Tests ohne Cache unter `Pacific/Kiritimati`
und `America/Los_Angeles` aus. Die zwei vollständigen Zusatzläufe stehen wegen
ihrer Laufzeit separat neben `task check`.

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

Testmandanten und ihre Generatoren stehen unter [Prüfszenarien](pruefszenarien.md).

## Änderungen prüfen

```sh
task check
npm --prefix frontend test
npm --prefix frontend run build
```

`task check` scheitert an unformatierten Go-Dateien und führt `go vet`, die
Go-Tests, die Prüfskripte für Gestaltung, Bridge, Dienste ohne Aufrufer,
Hilfetexte, Abschlussbausteine, Dokumentationslinks und Skill sowie die
Prüfung der Fassung in den Paketdateien aus. `task check:services` meldet jede
exportierte Methode eines Dienstes in `internal/service`, die außerhalb der
Tests niemand aufruft; was trotzdem bleibt, steht mit Grund in den Ausnahmen von
[`scripts/check_service_callers`](../../scripts/check_service_callers/main.go). Formatieren müssen Sie selbst (`gofmt -w`). Der Frontend-Build prüft auch
TypeScript.

| Aufgabe | Wann |
|---|---|
| `task test` | Go-Tests aller Pakete unter `internal/` und `cmd/` außer der Bridge |
| `task test:bridge` | Bridge-Tests; brauchen GTK, siehe oben |
| `task test:timezones` | Go-Tests unter `Pacific/Kiritimati` und `America/Los_Angeles`, rund acht Minuten |
| `task test:ebilanz-xsd` | E-Bilanz-Instanzen gegen die amtlichen Schemata 6.9 und 6.10; lädt die Pakete mit Prüfsumme nach `.cache/ebilanz-xsd/`, braucht bash, curl, unzip und Java ab 11 |
| `task check:clicks` | Klickweg von der Bilanz zum Beleg, braucht Chromium für Playwright ([Prüfszenarien](pruefszenarien.md)) |
| `task build` | Desktop-Build für das eigene Betriebssystem nach `bin/` |
| `task package` | Installationspaket für das eigene Betriebssystem |

Die Programmfassung setzt der Build: auf einem sauberen Stand mit Tag
`vX.Y.Z` ist sie `X.Y.Z`, sonst `dev+<Commit>`, bei ungesicherten Änderungen
mit `-dirty`. In Tests steht `dev`.

Die Brücke für Claude Desktop baut `task build:mcp` ohne Wails und ohne CGO,
für `GOOS` und `GOARCH` aus der Umgebung. Wie daraus die Erweiterung entsteht,
steht unter [packaging/claude-desktop](../../packaging/claude-desktop/README.md).

## Release

Die manuellen Release-Schritte stehen im [Release-Skill](../../.agents/skills/release/SKILL.md)
und in [`scripts/release.sh`](../../scripts/release.sh).

## Abhängigkeiten und Webseite

Nach Änderungen an Abhängigkeiten erzeugt `task notices` die
[Drittkomponentenhinweise](../../THIRD-PARTY-NOTICES.md) neu. Vorschau und
Veröffentlichung der Projektseite stehen in
[website/README.md](../../website/README.md).
