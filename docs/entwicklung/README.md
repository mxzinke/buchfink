# Entwicklung

[Dokumentation](../README.md) · [Orientierung im Code](codebase.md) ·
[Mitwirken](../../CONTRIBUTING.md)

## Voraussetzungen

- Go ab 1.25, gemäß [go.mod](../../go.mod).
- Node.js `^20.19.0` oder `>=22.12.0` und npm für das Frontend.
- Wails v3 CLI passend zur Version in `go.mod`.
- Task v3 für die unten verwendeten Befehle.
- Python 3 für die Prüfskripte.

Installiere die CLI-Werkzeuge und stelle sicher, dass das Go-Binärverzeichnis
in deinem `PATH` liegt:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.9
go install github.com/go-task/task/v3/cmd/task@latest
```

Für die Desktop-Anwendung müssen auch die nativen Build-Werkzeuge und
WebView-Abhängigkeiten des Betriebssystems vorhanden sein. Prüfe die Umgebung
mit `wails3 doctor`. Unter Linux benötigt der Desktop-Build CGO sowie die
GTK-/WebKit-Entwicklungsbibliotheken. Die plattformspezifischen Build-Schritte
stehen unter [build/](../../build).

Typst wird über die eingebundene WebAssembly-Bibliothek ausgeführt; eine
separate Typst-Installation ist für die Anwendung nicht nötig.

## Lokal starten

```bash
git clone https://github.com/mxzinke/buchfink.git
cd buchfink
npm --prefix frontend ci
task dev
```

`task dev` startet Wails mit [build/config.yml](../../build/config.yml) und
aktualisiert Frontend und Backend bei Änderungen. Ohne separate Task-Installation
kannst du `wails3 task dev` verwenden.

Nur den Vite-Server startest du mit:

```bash
npm --prefix frontend run dev
```

Das stellt das Frontend bereit. Für Aufrufe des Go-Backends brauchst du die
laufende Wails-Anwendung. Eine Browseransicht mit Beispieldaten nutzt die
[Konfiguration des Screenshot-Werkzeugs](../../scripts/site-screenshots/README.md).

## Änderungen prüfen

Führe die Befehle im Repository-Stamm aus:

```bash
task check
npm --prefix frontend test
npm --prefix frontend run build
```

`task check` zeigt unformatierte Go-Dateien an und führt Vet, Backend-Tests
sowie Prüfungen für Design, Bridge, Hilfetexte und Abschlussnavigation aus.
Gemeldete Formatierungsabweichungen musst du selbst beheben. Der Frontend-Build
prüft auch TypeScript.

Auf einem Rechner mit Desktop-Build-Umgebung prüfe zusätzlich die Wails-Bridge
und den vollständigen Build:

```bash
task test:bridge
task build
```

`task test` und `task vet` lassen sich auch ohne native Desktop-Abhängigkeiten
ausführen. Ein direktes `go build ./...` im frischen Checkout benötigt dagegen
zuerst die Frontend-Dateien unter `frontend/dist`, die Go einbettet.

Bedienabläufe und die Vorbereitung von `task check:clicks` stehen in den
[Prüfszenarien](pruefszenarien.md). Alle einzelnen Prüfaufgaben findest du im
[Taskfile](../../Taskfile.yml).

## Anwendung bauen

```bash
task build
task package
```

Die Aufgaben bauen beziehungsweise paketieren für das aktuelle Betriebssystem.
Die Ergebnisse liegen unter `bin/`. Konfiguration und Plattformaufgaben liegen
unter [build/](../../build).

## Abhängigkeiten und Projektseite

Nach Änderungen an Abhängigkeiten aktualisiert `task notices` die
[Drittkomponentenhinweise](../../THIRD-PARTY-NOTICES.md). Die Regeln zu Lizenzen und
Beiträgen stehen in [CONTRIBUTING.md](../../CONTRIBUTING.md).

Vorschau und Veröffentlichung der Webseite sind in
[website/README.md](../../website/README.md) beschrieben, die Erzeugung ihrer Bilder
beim [Screenshot-Werkzeug](../../scripts/site-screenshots/README.md).
