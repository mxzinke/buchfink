<p align="center">
  <img src="./assets/buchfink-logo.svg" alt="Buchfink Logo" width="200" />
</p>

# Buchfink

Freie Buchhaltungssoftware für kleine Unternehmen, Gründer und Holdings,
die ihre Buchhaltung selbst erledigen wollen. Kostenlos, quelloffen und auf
dem eigenen Rechner.

[Projektseite](https://mxzinke.github.io/buchfink/) ·
[Dokumentation](docs/README.md) · [Mitwirken](CONTRIBUTING.md)

## Wofür wir Buchfink entwickeln

Kleine Unternehmen sollen ihre Buchhaltung unabhängig führen können, auch ohne
Buchhaltungsausbildung. Unser Ziel ist eine gesetzeskonforme Software, an der
jeder mitwirken kann. Dazu gehören verständliche Erklärungen ebenso wie
fachliche Prüfungen und Verbesserungen am Code.

Langfristig sollen sich damit auch einfache Jahresabschlüsse selbst erstellen
lassen, für die viele Unternehmen heute eine Steuerkanzlei beauftragen.
Steuerberatung kann sich dann auf die steuerliche Gestaltung und konkrete
Fragen des Unternehmens konzentrieren.

## Aktueller Stand

Buchfink ist eine Vorschau-Version in Entwicklung. Implementiert sind unter
anderem die Belegablage, Buchungen, Bankimport, Rechnungen, Umsatzsteuer-
Voranmeldung und die Vorbereitung des Jahresabschlusses. Bilanz und Gewinn-
und Verlustrechnung lassen sich ausgeben. Die Oberfläche erklärt Begriffe
kurz am Fragezeichen; „Mehr erfahren“ öffnet Details mit Quellenlinks.

Ein vollständig geprüfter Abschluss einschließlich Einreichung ist noch
nicht erreicht. Die E-Bilanz-Zuordnung ist ungeprüft, und Teile des Abschluss-
und Offenlegungsablaufs fehlen. Die vorhandenen Prüfungen sind keine
Zusicherung, dass jeder Geschäftsvorgang gesetzeskonform abgedeckt ist.
Der [Umsetzungsstand](docs/stand-der-umsetzung.md) nennt die konkreten Grenzen.

Der Schwerpunkt liegt auf kleinen GmbHs und UGs mit doppelter Buchführung und
Bilanz. Das schließt Holdings mit überschaubaren Geschäftsvorgängen ein,
aber keine Konzernrechnungslegung. Andere wählbare Rechtsformen sind teils nur
eingeschränkt abgedeckt. EÜR, eigene Kleinunternehmerbesteuerung, Lohnabrechnung,
Kassenbuch und DATEV-Export sind nicht implementiert.

## Unabhängig arbeiten und weiterentwickeln

Buchfink speichert die Daten je Unternehmen lokal in einer SQLite-Datenbank
und legt Belege im Original ab. Sicherungen und Exporte geben Zugriff auf die
eigene Buchhaltung. Personenbezogene und geschäftliche Datenbankfelder sind
verschlüsselt; Belegdateien sind nicht verschlüsselt. Die Grenzen beschreibt
das [Sicherheitskonzept](docs/security-concept.md).

Einzelne Funktionen nutzen externe Dienste, etwa Wechselkurse, Zeitstempel und
Umsatzsteuer-ID-Prüfungen. Ein lokaler MCP-Server zur Anbindung des Chatbots der
Wahl ist geplant und noch nicht verfügbar. Ziele und offene Arbeiten stehen
auf der [Roadmap](docs/roadmap.md).

## Tech-Stack

| Schicht | Technologie | Beschreibung |
|---|---|---|
| **Desktop Shell** | [Wails v3](https://v3.wails.io/) | Schlanke, native WebView-Desktop-Shell (macOS, Windows, Linux) |
| **Backend** | Go (Golang) | Performante Geschäftslogik, Hash-Chain, XML/XBRL & Bankparser |
| **Datenbank** | SQLite (Pure Go) | Eine SQLite-Datei je Mandant, CGO-frei, Feldverschlüsselung über einen GORM-Serializer |
| **Frontend** | React, TypeScript, Vite | Schnelles, reaktives UI ohne schweren Design-System-Overhead |
| **UI-Bausteine** | [Base UI](https://base-ui.com) | Unstyled: Fokusfang, Positionierung und Tastaturführung. Gestalt kommt aus dem eigenen Design-System |
| **Styling** | Tailwind CSS & Lucide Icons | Minimalistisch, flach und warm gestaltet |
| **Dokumente** | Typst | Layout-Engine für ZUGFeRD PDF/A-3 Rechnungen |

---

## Entwicklung & Weiterentwicklung

### Voraussetzungen

- **Go:** `>= 1.25` ([Download](https://golang.org/dl/))
- **Node.js:** `^20.19.0` oder `>= 22.12.0` ([Download](https://nodejs.org/))
- **Wails v3 CLI:**
  ```bash
  go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.9
  ```
- *(Optional)* **Taskfile:** `brew install go-task` oder `go install github.com/go-task/task/v3/cmd/task@latest`
- *(Optional)* **Typst:** für lokale Rechnungs-Kompilierung (`brew install typst`)

### Projekt starten

1. **Repository klonen und Frontend-Abhängigkeiten installieren:**
   ```bash
   git clone https://github.com/mxzinke/buchfink.git
   cd buchfink
   cd frontend && npm install && cd ..
   ```

2. **Entwicklungsmodus starten (Hot-Reload für Frontend & Backend):**
   ```bash
   wails3 dev
   # oder mit Taskfile:
   task dev
   ```

3. **Frontend isoliert im Browser testen:**
   ```bash
   cd frontend
   npm run dev
   ```

4. **Prüfen, was vor einem Commit läuft:**
   ```bash
   task check          # gofmt, vet, Tests, check:design, check:bridge, check:text, check:steps
   task check:text     # Hilfetexte und Gesetzesverweise in Arbeitsansichten
   task check:steps    # Abschlussbaustein ohne Ziel im geführten Weg
   task check:clicks   # fährt das Prüfszenario ab und zählt die Klicks (braucht Chromium)
   ```

5. **Desktop-Build erstellen:**
   ```bash
   wails3 build
   # oder mit Taskfile:
   task build
   ```

### Projektstruktur

```text
buchfink/
├── assets/                 # App-Icons, Logos und Brand-Assets
├── build/                  # Wails v3 Build- und Packaging-Konfigurationen
├── frontend/               # React + TypeScript + Vite Frontend
│   ├── src/
│   │   ├── components/     # UI-Komponenten (Sidebar, Header, Dialoge)
│   │   │   └── ui/         # Bausteine des Design-Systems
│   │   ├── pages/          # Ansichten (Journal, Konten, Bank, Auswertungen, ...)
│   │   ├── services/       # Wails Go-Bindings / API Client
│   │   ├── types/          # Gemeinsame TypeScript-Typen
│   │   └── utils/          # Formatierung (Währung, Datum), Hilfsfunktionen
│   └── package.json
├── internal/               # Go Backend Module
│   ├── accounting/         # SKR04-Kontenplan, Buchungsgruppen, Steuerschlüssel, AfA, Größenklassen, Bilanzgliederung, Hash-Chain
│   ├── actor/              # Bearbeiterkennung aus Betriebssystem-Benutzer und Rechnername
│   ├── bank/               # CAMT.053-Parser
│   ├── buildinfo/          # Programmfassung je Buchung
│   ├── changelog/          # Programmstände, gehen mit der Datenüberlassung hinaus
│   ├── currency/           # EZB-Referenzkurse
│   ├── domain/             # Fachliche Typen und Repository-Schnittstellen
│   ├── ebilanz/            # XBRL-Zuordnung, Taxonomie-Ressource & Instanzerzeugung
│   ├── einvoice/           # E-Rechnung: CII, UBL, ZUGFeRD, XRechnung, Regelwerk
│   ├── export/             # Z3-Datenüberlassung: CSV, index.xml, Feldbeschreibung
│   ├── invoice/            # Ausgangsrechnung: ZUGFeRD-XML & Typst-Rendering
│   ├── procdoc/            # Verfahrensdokumentation aus dem laufenden System
│   ├── receiptstore/       # Belegablage unter SHA256
│   ├── repository/         # GORM/SQLite-Persistenz & Feldverschlüsselung
│   ├── security/           # Vault, Schlüsselbund, Wiederherstellung
│   ├── service/            # Anwendungsfälle (Buchen, Zahlen, Anlagen, Belege, Abschluss, Export, ...)
│   ├── timestamp/          # RFC-3161-Zeitstempel für die Festschreibung
│   ├── vatid/              # Qualifizierte Bestätigung der USt-IdNr. beim BZSt
│   └── wailsbridge/        # Aufrufbare Oberfläche für das Frontend
├── scripts/                # Prüf- und Erzeugungsskripte
│   ├── check_ui_text.py    # Normen im Detaildialog (task check:text)
│   ├── check_closing_steps.py  # jeder Abschlussbaustein hat ein Ziel (task check:steps)
│   └── site-screenshots/   # Screenshots der Projektseite, four-clicks.mjs zählt den Prüfweg
├── website/                # Projektseite für GitHub Pages
├── main.go                 # App Entrypoint & Wails Service Registration
├── Taskfile.yml            # Build & Automation Tasks
└── go.mod
```

Die Projektseite unter [`website/`](./website) wird mit `task pages:publish`
veröffentlicht — von Hand, ohne CI. Ihre Screenshots entstehen aus der echten
Oberfläche mit Beispieldaten; wie das läuft, steht in
[`scripts/site-screenshots/`](./scripts/site-screenshots).

---

## Mitwirken

Rückmeldungen aus dem Unternehmensalltag, verständlichere Texte, fachliche
Prüfungen, Fehlerberichte und Codebeiträge sind willkommen. Programmierkenntnisse
sind keine Voraussetzung, um dem Projekt zu helfen.

Den Ablauf beschreibt [CONTRIBUTING.md](CONTRIBUTING.md). Für eingereichte
Beiträge gilt die [Vereinbarung über Beiträge](CLA.md).

## Lizenz

Copyright © 2026 Maximilian Pfennig. Buchfink steht unter der
[EUPL-1.2](LICENSE). Die Lizenz erlaubt Nutzung, Anpassung und Weitergabe unter
ihren Bedingungen und enthält die Regelungen zu Gewährleistung und Haftung.
Mitgelieferte Komponenten und ihre Lizenzen stehen in
[THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md).
