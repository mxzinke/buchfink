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

Es gibt keine CI. Geprüft, gebaut und veröffentlicht wird von Hand mit
[`scripts/release.sh`](../../scripts/release.sh), aufgerufen über das Taskfile:

1. `python3 scripts/version.py set X.Y.Z` schreibt die Fassung in alle
   Paketdateien. In [internal/changelog/CHANGELOG.md](../../internal/changelog/CHANGELOG.md)
   einen Eintrag `## vX.Y.Z — JJJJ-MM-TT` anlegen; er ist der Text des Release.
   Committen.
2. `task release:check VERSION=X.Y.Z` prüft Fassung und Eintrag, `go mod tidy`,
   das Frontend mit TypeScript, `task check`, die Bridge-Tests, die Tests unter
   extremen Zeitzonen, die E-Bilanz gegen die amtlichen Schemata und das
   Skill-Paket. Braucht einen sauberen Stand, Java ab 11 und Netz.
3. `git tag -a vX.Y.Z -m "Buchfink X.Y.Z"`. Eine Fassung wie `X.Y.Z-rc.1` wird
   Vorabversion.
4. `task release:build VERSION=X.Y.Z` baut auf macOS
   `buchfink-darwin-universal.zip` (Ad-hoc-Signatur), `buchfink-windows-amd64.exe`,
   die Brücke `buchfink-mcp-*` für macOS, Windows und Linux, `buchfink.mcpb`
   und das Skill-Paket `buchfink.zip` nach `dist/release/vX.Y.Z/`.
5. `task release:build:linux VERSION=X.Y.Z` baut `buchfink-linux-amd64.tar.gz`,
   auf Linux amd64 direkt, sonst in einem `linux/amd64`-Container mit Docker aus
   dem Stand des Tags.
6. `git push origin master vX.Y.Z`, dann `task release:publish VERSION=X.Y.Z`:
   prüft, dass jede Datei vorliegt, auch jede, auf die Dokumentation oder
   Webseite unter `releases/latest/download/` verweisen, schreibt `SHA256SUMS`
   und legt das GitHub-Release mit `gh` an.

Signiert wird nichts. Die Dateinamen enthalten keine Fassung, damit die Links
auf `releases/latest/download/<datei>` stabil bleiben.

## Abhängigkeiten und Webseite

Nach Änderungen an Abhängigkeiten erzeugt `task notices` die
[Drittkomponentenhinweise](../../THIRD-PARTY-NOTICES.md) neu. Vorschau und
Veröffentlichung der Projektseite stehen in
[website/README.md](../../website/README.md).
