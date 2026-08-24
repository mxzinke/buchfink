<p align="center">
  <img src="./assets/buchfink-logo.svg" alt="Buchfink Logo" width="200" />
</p>

<h1 align="center">Buchfink</h1>

<p align="center">
  <strong>Moderne Open-Source-Buchhaltungssoftware für bilanzierende Unternehmen</strong><br />
  Native Desktop-App &bull; Doppelte Buchführung &bull; SKR04 &bull; Bilanz & GuV &bull; GoBD-konform ab v1 &bull; E-Bilanz (XBRL) &bull; Local-First
</p>

<p align="center">
  <a href="#ziel--anwendungsbereich">Ziel & Anwendungsbereich</a> &bull;
  <a href="#kernfunktionen">Kernfunktionen</a> &bull;
  <a href="#speicherung--integrität">Speicherung & Integrität</a> &bull;
  <a href="#tech-stack">Tech-Stack</a> &bull;
  <a href="#entwicklung">Entwicklung & Setup</a> &bull;
  <a href="#scope--entscheidungen">Scope & Entscheidungen</a> &bull;
  <a href="#mitwirken">Mitwirken</a> &bull;
  <a href="#lizenz">Lizenz</a>
</p>

---

## Ziel & Anwendungsbereich

**Buchfink** ist eine native Desktop-Buchhaltungssoftware für die **doppelte kaufmännische Buchführung und Bilanzierung** nach dem deutschen Kontenrahmen **SKR04**. Sie richtet sich an bilanzierende Unternehmen, die eine schlanke, GoBD-konforme Lösung ohne Cloud-Zwang oder Abo-Modell suchen.

### 🎯 Zielgruppe & Voraussetzungen
Buchfink ist speziell für Unternehmen konzipiert, die zur **doppelten Buchführung und Bilanzierung** (Erstellung von Bilanz, Gewinn- und Verlustrechnung sowie E-Bilanz) verpflichtet sind oder freiwillig bilanzieren:
- **Kapitalgesellschaften:** z. B. UG (haftungsbeschränkt), GmbH, AG
- **Personenhandelsgesellschaften:** z. B. GmbH & Co. KG, KG, OHG
- **Bilanzierende Einzelunternehmen & Kaufleute (e.K.)**

### ⚠️ Wichtiger Hinweis: Nicht geeignet für EÜR (Einnahmen-Überschuss-Rechnung)
Buchfink ist **nicht für kleine Selbstständige, Freiberufler oder Kleinunternehmer geeignet**, die lediglich eine einfache **Einnahmen-Überschuss-Rechnung (EÜR nach § 4 Abs. 3 EStG)** durchführen.
- Buchfink unterstützt **keine EÜR**.
- Die Software basiert vollständig auf dem geschlossenen System der doppelten Buchführung mit Soll und Haben, Bestandskonten (Aktiva/Passiva), Erfolgskonten (Aufwand/Ertrag), Bilanzierung und der amtlichen E-Bilanz-Taxonomie.

---

### Grundprinzipien

- **Local-First:** Alle Daten verbleiben auf dem eigenen Rechner in standardisierten SQLite-Dateien.
- **GoBD-konform ab v1:** Lückenlose Nachvollziehbarkeit durch kryptografische Hash-Chains, unveränderliche Belegablage und integriertes Audit-Log.
- **Automatisierungsfokus:** Buchungen entstehen primär aus dem Abgleich von Banktransaktionen mit Belegen – kein manuelles Soll/Haben-Tippen im Alltag.
- **E-Bilanz bereit:** Integrierter XBRL-Export für die E-Bilanz zur direkten Übergabe an Mein ELSTER oder Bridges.

---

## Kernfunktionen

1. **Kontenverwaltung (SKR04)**
   - Vorinstallierter, erweiterbarer SKR04-Kontenrahmen mit Such- und Hilfefunktion für steuerliche Einsteiger.
2. **Automatisiertes Journal**
   - Transparente Soll/Haben-Ansicht. Buchungen entstehen automatisch durch Zuordnung von Banktransaktionen zu Belegen/Kontakten.
   - Lückenlose Belegnummerierung und GoBD-Korrekturen ausschließlich per Storno.
3. **Kunden & Lieferanten (Offene Posten)**
   - Stammdatenverwaltung, OPOS-Listen und intelligenter Zahlungsabgleich.
4. **Rechnungserstellung mit Typst & ZUGFeRD**
   - Professionelles Rechnungslayout via [Typst](https://typst.app/).
   - Generierung von ZUGFeRD-/Factur-X-konformen PDF/A-3-Dokumenten mit eingebettetem XML.
5. **Bankumsatz-Import (CAMT.053)**
   - Schneller Import von standardisierten CAMT.053-Bankauszügen.
   - Automatischer Vorschlag für Beleg- und Kontenzuordnungen.
6. **Bilanz & GuV**
   - Echtzeit-Auswertung von Bilanz, Gewinn- und Verlustrechnung (GuV) sowie Summen- und Saldenliste (SuSa).
7. **E-Bilanz-Export (XBRL)**
   - Automatisches Mapping der SKR04-Konten auf die offizielle E-Bilanz-Taxonomie.
   - Direkte Erzeugung einer validen XBRL-Datei inkl. Kontennachweis (Übermittlung via Mein ELSTER / externe Bridge).
8. **Fremdwährungsumrechnung**
   - Buchung in EUR mit tagesaktuellen EZB-Referenzkursen (protokolliert mit Kursquelle und Zeitstempel).
9. **Verfahrensdokumentation & Audit-Log**
   - Live-Prüfung der Hash-Chain-Integrität in der UI.
   - Vollständiges Änderungsprotokoll aller Stammdaten und verknüpfte Verfahrensdokumentation (Markdown).

---

## Speicherung & Integrität

```text
buchfink-data/
├── 2024.sqlite                  # SQLite DB pro Geschäftsjahr (Konfig, Konten, Journal)
└── belege/
    └── 2024/
        ├── RE-2024-001-a1b2c3.pdf # Originalbelege mit SHA256-Hash-Präfix
        └── ...
```

- **Hash-Chain:** Jede Buchungszeile enthält den SHA256-Hash der vorangehenden Buchung (Git-Prinzip). Manipulationen an vergangenen Perioden werden sofort sichtbar.
- **Isolation:** Eine SQLite-Datei pro Geschäftsjahr. Abgeschlossene Jahre werden schreibgeschützt eingefroren.
- **Belegintegrität:** Originaldateien bleiben unverändert; Hash-basierte Dateiablage.

---

## UI & Design-Prinzipien

Leitidee: **Stilles Kontor** – die Oberfläche ist Werkzeug, keine Bühne. Das vollständige Konzept steht in [`docs/design-konzept.md`](./docs/design-konzept.md), die Bausteine in [`frontend/src/components/ui/`](./frontend/src/components/ui).

- **Typografie:** [Manrope](https://fonts.google.com/specimen/Manrope) in sechs Stufen – modern, klar lesbar und neutral. Beträge in tabellarischen Ziffern, damit Spalten untereinander stehen.
- **Farbwelt:** Warmes Papier und Tinte als Grundfläche, dazu vier pastellige Familien: Himmelblau als Marke, Bernstein für offen, Salbei für geprüft, Rosé für Storno. Jede Familie hat vier Rollen (Fläche, Rand, Marker, Text), damit Pastell die Kontrastvorgaben hält.
- **Flach statt gekachelt:** Inhalt liegt direkt auf dem Papier. Abschnitte trennen Überschrift, Abstand und Haarlinie, keine Karten. Eine eigene Fläche bekommt nur, was vom Blatt gelöst ist: Datentabellen, Overlays, Belegvorschau.
- **Bewegung:** sechs erlaubte Übergänge, 120 bis 180 ms. Zahlen animieren nie.
- **Wenig Text:** Arbeitsansichten enthalten keinen Fließtext. Erklärungen liegen hinter einem Erklärzeichen, in drei Stufen: Tooltip, Popover, Dialog.
- **Zahlenformatierung:** Konsequent de-DE (`1.234,56 €`, `01.01.2024`).
- **Fachliche Muster:** Soll und Haben zweispaltig und neutral, Summen mit buchhalterischer Doppellinie, Storno sichtbar markiert statt durchgestrichen.
- **Barrierearme Buchhaltung:** Versteckte Fachbegriff-Erklärungen, Tastatur-Shortcuts und geführte Workflows für Nicht-Buchhalter.

---

## Tech-Stack

| Schicht | Technologie | Beschreibung |
|---|---|---|
| **Desktop Shell** | [Wails v3](https://v3.wails.io/) | Schlanke, native WebView-Desktop-Shell (macOS, Windows, Linux) |
| **Backend** | Go (Golang) | Performante Geschäftslogik, Hash-Chain, XML/XBRL & Bankparser |
| **Datenbank** | SQLite (Pure Go) | Eine SQLite-Datei pro Geschäftsjahr, CGO-frei |
| **Frontend** | React, TypeScript, Vite | Schnelles, reaktives UI ohne schweren Design-System-Overhead |
| **UI-Bausteine** | [Base UI](https://base-ui.com) | Unstyled: Fokusfang, Positionierung und Tastaturführung. Gestalt kommt aus dem eigenen Design-System |
| **Styling** | Tailwind CSS & Lucide Icons | Minimalistisch, flach und warm gestaltet |
| **Dokumente** | Typst | Layout-Engine für ZUGFeRD PDF/A-3 Rechnungen |

---

## Entwicklung & Weiterentwicklung

### Voraussetzungen

- **Go:** `>= 1.22` ([Download](https://golang.org/dl/))
- **Node.js:** `>= 20` ([Download](https://nodejs.org/))
- **Wails v3 CLI:**
  ```bash
  go install github.com/wailsapp/wails/v3/cmd/wails3@latest
  ```
- *(Optional)* **Taskfile:** `brew install go-task` oder `go install github.com/go-task/task/v3/cmd/task@latest`
- *(Optional)* **Typst:** für lokale Rechnungs-Kompilierung (`brew install typst`)

### Projekt starten

1. **Repository klonen und Frontend-Abhängigkeiten installieren:**
   ```bash
   git clone https://github.com/buchfink/buchfink.git
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

4. **Desktop-Build erstellen:**
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
│   │   ├── pages/          # Ansichten (Journal, Konten, Bank, Bilanz, ...)
│   │   ├── services/       # Wails Go-Bindings / API Client
│   │   └── utils/          # Formatierung (Währung, Datum), Hilfsfunktionen
│   └── package.json
├── internal/               # Go Backend Module
│   ├── accounting/         # SKR04 Kontenplan, Buchungssätze, Hash-Chain
│   ├── audit/              # GoBD Audit-Log & Integritätsprüfung
│   ├── bank/               # CAMT.053 Parser & Bank-Zuordnungslogik
│   ├── currency/           # EZB-Referenzkurse
│   ├── database/           # SQLite Initialisierung & Migrationen
│   ├── ebilanz/            # XBRL Taxonomie-Mapping & XML-Export
│   └── invoice/            # ZUGFeRD / Factur-X XML & Typst Rendering
├── main.go                 # App Entrypoint & Wails Service Registration
├── Taskfile.yml            # Build & Automation Tasks
└── go.mod
```

---

## Scope & Entscheidungen

| Thema | Entscheidung in Buchfink |
|---|---|
| **Anwendungsbereich & Zielgruppe** | **Ausschließlich bilanzierende Unternehmen** (z. B. UG, GmbH, AG, bilanzierende Kaufleute). **Nicht geeignet** für kleine Selbstständige, Freiberufler oder Kleinunternehmer mit einfacher Einnahmen-Überschuss-Rechnung (EÜR). |
| **Buchungsansatz** | Doppelte Buchführung (Soll & Haben) nach dem Prinzip „Buchung folgt Bankumsatz“: Transaktionen werden Belegen zugeordnet und generieren automatisch Soll/Haben-Sätze. |
| **Kontenrahmen** | SKR04 als Standard für v1 (Abschlussgliederungsprinzip für Bilanz & GuV). |
| **GoBD** | Vollständige GoBD-Konformität von Tag 1 (Unveränderbarkeit, Hash-Chains, Storno-Prinzip). |
| **E-Bilanz / ERiC** | Eigene Erzeugung gültiger XBRL-Dateien im Tool inkl. Kontennachweis nach amtlicher Taxonomie. Direkte ERiC-Übermittlung ist bewusst out-of-scope (proprietäre C-Bibliothek); Einreichung erfolgt über Mein ELSTER oder Bridges (z. B. eBilanz+). |
| **Out-of-Scope (v1)** | Einnahmen-Überschuss-Rechnung (EÜR), Lohnbuchhaltung, Lagerverwaltung/Inventur, mehrsprachige UI (v1 fokussiert auf Deutsch/DACH). |

---

## Mitwirken

Fehlerberichte und Pull Requests sind willkommen. Wie der Ablauf aussieht, was
vor einem Pull Request laufen sollte und welche Regeln für fremdes Material
gelten, steht in [CONTRIBUTING.md](CONTRIBUTING.md).

Eingereichter Inhalt setzt die Zustimmung zur
[Vereinbarung über Beiträge](CLA.md) voraus. Das Urheberrecht am eigenen
Beitrag bleibt beim Beitragenden; die Vereinbarung hält dem Projekt die
Möglichkeit offen, seine Lizenzierung aus einer Hand anzupassen.

---

## Lizenz

```text
Copyright (c) 2026 Maximilian Pfennig

Lizenziert unter der EUPL
```

Buchfink steht unter der **[Open-Source-Lizenz für die Europäische Union
v1.2](LICENSE)** (EUPL-1.2). Die [`LICENSE`](LICENSE) enthält den amtlichen
deutschen und den amtlichen englischen Wortlaut. Nach Artikel 13 der Lizenz
haben alle Sprachfassungen denselben Rang; du kannst dich auf die Fassung
deiner Wahl berufen.

Was das praktisch bedeutet:

- **Nutzen und anpassen:** uneingeschränkt. Wer Buchfink für den eigenen
  Betrieb umbaut und intern einsetzt, muss nichts veröffentlichen.
- **Weitergeben:** Wer eine veränderte Fassung verbreitet oder ihre wesentlichen
  Funktionen online zugänglich macht — auch als gehostete Anwendung —, gibt sie
  unter der EUPL weiter und liefert den Quellcode mit oder nennt einen frei
  zugänglichen Speicherort.
- **Hinweise erhalten:** Urheberrechts- und Lizenzhinweise bleiben stehen,
  Änderungen werden mit Datum kenntlich gemacht.
- **Name und Logo:** „Buchfink“ und die Kennzeichen des Projekts sind von der
  Lizenz nicht erfasst (Artikel 5 EUPL). Ein Fork braucht einen eigenen Namen.
- **Recht und Gerichtsstand:** deutsches Recht, Gericht am Sitz des
  Lizenzgebers (Artikel 14 und 15 EUPL).

Mitgelieferte Komponenten Dritter behalten ihre eigenen Lizenzen und sind in
[THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md) aufgeführt.
