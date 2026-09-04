<p align="center">
  <img src="./assets/buchfink-logo.svg" alt="Buchfink Logo" width="200" />
</p>

<h1 align="center">Buchfink</h1>

<p align="center">
  <strong>Moderne Open-Source-Buchhaltungssoftware für bilanzierende Unternehmen</strong><br />
  Native Desktop-App &bull; Doppelte Buchführung &bull; SKR04 &bull; Bilanz & GuV &bull; GoBD-konform ab v1 &bull; E-Bilanz (XBRL) &bull; Local-First
</p>

<p align="center">
  <em>In Entwicklung. Was heute trägt, wo eine Funktion an ihrer Grenze endet und was
  noch fehlt, steht in <a href="./docs/stand-der-umsetzung.md">docs/stand-der-umsetzung.md</a>.
  Für ein volles Geschäftsjahr fehlt derzeit vor allem der Jahreswechsel mit
  Saldenvortrag.</em>
</p>

<p align="center">
  <a href="https://mxzinke.github.io/buchfink/"><strong>Projektseite mit Screenshots</strong></a>
</p>

<p align="center">
  <a href="#ziel--anwendungsbereich">Ziel & Anwendungsbereich</a> &bull;
  <a href="#kernfunktionen">Kernfunktionen</a> &bull;
  <a href="#speicherung--integrität">Speicherung & Integrität</a> &bull;
  <a href="#tech-stack">Tech-Stack</a> &bull;
  <a href="#entwicklung">Entwicklung & Setup</a> &bull;
  <a href="#scope--entscheidungen">Scope & Entscheidungen</a> &bull;
  <a href="#stand-der-umsetzung">Stand der Umsetzung</a> &bull;
  <a href="#mitwirken">Mitwirken</a> &bull;
  <a href="#lizenz">Lizenz</a>
</p>

---

## Ziel & Anwendungsbereich

**Buchfink** ist eine native Desktop-Buchhaltungssoftware für die **doppelte kaufmännische Buchführung und Bilanzierung** nach dem deutschen Kontenrahmen **SKR04**. Sie richtet sich an bilanzierende Unternehmen, die eine schlanke, GoBD-konforme Lösung ohne Cloud-Zwang oder Abo-Modell suchen.

### 🎯 Zielgruppe & Voraussetzungen
Buchfink ist speziell für Unternehmen konzipiert, die zur **doppelten Buchführung und Bilanzierung** (Erstellung von Bilanz, Gewinn- und Verlustrechnung sowie E-Bilanz) verpflichtet sind oder freiwillig bilanzieren:
- **Kapitalgesellschaften:** z. B. UG (haftungsbeschränkt), GmbH, AG – auch schon in Gründung, von der Beurkundung bis zur Eintragung
- **Personenhandelsgesellschaften:** z. B. GmbH & Co. KG, KG, OHG
- **Bilanzierende Einzelunternehmen & Kaufleute (e.K.)**

### ⚠️ Wichtiger Hinweis: Nicht geeignet für EÜR (Einnahmen-Überschuss-Rechnung)
Buchfink ist **nicht für kleine Selbstständige, Freiberufler oder Kleinunternehmer geeignet**, die lediglich eine einfache **Einnahmen-Überschuss-Rechnung (EÜR nach § 4 Abs. 3 EStG)** durchführen.
- Buchfink unterstützt **keine EÜR**.
- Die Software basiert vollständig auf dem geschlossenen System der doppelten Buchführung mit Soll und Haben, Bestandskonten (Aktiva/Passiva), Erfolgskonten (Aufwand/Ertrag), Bilanzierung und der amtlichen E-Bilanz-Taxonomie.

---

### Grundprinzipien

- **Local-First:** Alle Daten verbleiben auf dem eigenen Rechner in einer standardisierten SQLite-Datei je Mandant.
- **GoBD-konform ab v1:** Lückenlose Nachvollziehbarkeit durch kryptografische Hash-Chains, unveränderliche Belegablage, Festschreibung mit Zeitstempel und integriertes Audit-Log. Der Datenexport für die Betriebsprüfung (Z3) fehlt noch.
- **Automatisierungsfokus:** Der Alltag beginnt beim Kontoauszug: eine Zahlung wird ihrem offenen Posten oder einem Beleg zugeordnet, den Buchungssatz und die Steuer rechnet Buchfink daraus. Welcher Posten zu welcher Zahlung gehört, entscheidet heute noch der Nutzer.
- **E-Rechnung:** Eingehende ZUGFeRD-, Factur-X- und XRechnung-Dateien werden erkannt, gelesen und gegen das Regelwerk geprüft. Ausgestellt wird als ZUGFeRD-PDF.
- **Verschlüsselt abgelegt:** Personenbezogene und geschäftliche Datenbankfelder liegen mit AES-256-GCM verschlüsselt, der Schlüssel im Schlüsselbund des Betriebssystems.

---

## Kernfunktionen

1. **Kontenverwaltung (SKR04)**
   - Vorinstallierter, erweiterbarer SKR04-Kontenrahmen mit Such- und Hilfefunktion für steuerliche Einsteiger.
2. **Journal aus dem Belegfluss**
   - Transparente Soll/Haben-Ansicht. Den Buchungssatz und die Steuer rechnet Buchfink aus dem erfassten Beleg, aus der Rechnung oder aus der zugeordneten Zahlung.
   - Lückenlose Belegnummerierung und GoBD-Korrekturen ausschließlich per Storno.
3. **Kunden & Lieferanten (Offene Posten)**
   - Stammdatenverwaltung, OPOS-Liste und Zahlungsausgleich mit Teilzahlung, Skonto und Differenzgründen.
4. **Rechnungserstellung mit Typst & ZUGFeRD**
   - Professionelles Rechnungslayout via [Typst](https://typst.app/).
   - Generierung von ZUGFeRD-/Factur-X-konformen PDF/A-3-Dokumenten mit eingebettetem XML.
5. **E-Rechnungs-Empfang**
   - Erkennt ZUGFeRD, Factur-X und XRechnung im eingegangenen Beleg, liest CII und UBL und prüft beides gegen das Regelwerk der Norm.
   - Aus dem gelesenen Datensatz entsteht ein Buchungsvorschlag; der Beleg bleibt unverändert, wie er ankam.
6. **Bankumsatz-Import (CAMT.053)**
   - Import von standardisierten CAMT.053-Bankauszügen.
   - Zu einem Umsatz zeigt Buchfink die offenen Posten der passenden Richtung; welcher davon gemeint ist, wählt der Nutzer. Einen Vorschlag nach Betrag oder Verwendungszweck gibt es noch nicht.
7. **Anlagevermögen (Anlagenverzeichnis, AfA, Anlagenspiegel)**
   - Verzeichnis für Sach-, Finanz- und immaterielle Anlagen mit Inventarnummer, Bewegungen und jahresübergreifender Kartei.
   - Wertgrenzen des § 6 Abs. 2 und 2a EStG (GWG, Sammelposten), lineare und degressive AfA mit automatischem Übergang, Sonderabschreibung nach § 7g Abs. 5 EStG samt Restwertverteilung des § 7a Abs. 9 EStG, außerplanmäßige Abschreibung und Zuschreibung.
   - Abschreibungslauf als Abschlussbuchung mit Vorschau; die Jahres-Festschreibung prüft vorher, ob die AfA vollständig gebucht ist.
   - Fertigstellung von Anlagen im Bau, Erhaltungsaufwand und laufende Erträge am Anlagegut, Stückzahlen und Fremdwährungsbewertung nach § 256a HGB bei Finanzanlagen.
   - Verträge, Gutachten, Zulassungen und Policen am Anlagegut ablegen — mit Ablaufdatum, das wieder gelesen wird.
   - Darlehen und Ausleihungen mit Fälligkeit und Tilgung als eigenem Weg: eine Rückzahlung ist kein Verkauf und erzeugt keinen Erlös.
   - Investmentanteile (ETF, Aktien- und Immobilienfonds): Teilfreistellung nach § 20 InvStG und Vorabpauschale nach § 18 InvStG als außerbilanzielle Nebenrechnung.
   - Skonto auf eine Anlagenrechnung mindert im Zahlungsflow die Anschaffungskosten (§ 255 Abs. 1 Satz 3 HGB) statt den Aufwand.
   - Abgang mit Erlöskonto nach Buchgewinn oder -verlust, Teilabgang nach Stück bei Finanzanlagen und Anlagenspiegel nach § 284 Abs. 3 HGB — auch als Kontennachweis in der E-Bilanz.
8. **Auswertungen**
   - Kontenblatt, Summen- und Saldenliste, Gewinn- und Verlustrechnung, Bilanz und eine Umsatzsteuer-Übersicht, alle direkt aus den Buchungen des Geschäftsjahres.
   - Bilanz und GuV sind heute nach Kontenklassen gruppiert, nicht nach § 266 und § 275 HGB gegliedert, und es gibt keine Vorjahresspalte und keine Ausgabe als Datei.
   - Die Umsatzsteuer-Ansicht zeigt vier Kennziffern des amtlichen Vordrucks (81, 86, 66, 83) zum Abtippen in Mein ELSTER. Eine vollständige Voranmeldung ist sie nicht.
9. **E-Bilanz-Export (XBRL)**
   - Zuordnung der SKR04-Konten auf die E-Bilanz-Taxonomie, Erzeugung einer XBRL-Instanz mit Kontennachweis und Anlagenspiegel.
   - Noch ein Gerüst: rund fünfzig Konten sind zugeordnet, aus dem GAAP-Modul werden drei Summenwerte geschrieben, eine Bilanz steht nicht in der Instanz. Vor einer Übermittlung über Mein ELSTER ist die Datei von Hand zu prüfen. Einzelheiten in [docs/stand-der-umsetzung.md](./docs/stand-der-umsetzung.md).
10. **Fremdwährung**
   - Kurs, Kursquelle und Kursdatum hängen an der Buchung und gehen in die Hash-Chain ein; Finanzanlagen werden nach § 256a HGB zum Stichtag bewertet.
   - Die Kurse werden heute von Hand erfasst. Der Abruf der EZB-Referenzkurse ist vorbereitet, aber noch nicht an die Oberfläche angeschlossen.
11. **Festschreibung, Audit-Log & Integrität**
   - Live-Prüfung der Hash-Chain in der Oberfläche.
   - Festschreibung je Monat, Quartal oder Jahr, beglaubigt durch einen RFC-3161-Zeitstempel über den Kettenkopf.
   - Änderungsprotokoll über Buchungen, Belege und Stammdaten.
12. **Gründung einer Kapitalgesellschaft**
   - Erfassung im Einrichtungsassistenten: Beurkundungsdatum, Stammkapital, Gesellschafter und der Gründungsaufwand laut Satzung. Aus dem Beurkundungsdatum folgen Rumpfgeschäftsjahr und Voranmeldungszeitraum.
   - Prüfung der Kapitalaufbringung vor der Anmeldung zum Handelsregister: Viertelregel je Geschäftsanteil und Untergrenze nach § 7 Abs. 2 GmbHG, Volleinzahlung und Sacheinlageverbot der UG nach § 5a Abs. 2 GmbHG, § 36a AktG bei der AG.
   - **Unterbilanzhaftung**: laufende Rechnung, um wie viel das Reinvermögen hinter dem Stammkapital zurückbleibt, aufgeteilt auf die Gesellschafter. Mit der Eintragung steht sie fest.
   - Gründungsbuchungen als Vorschlag mit Vorschau, und die Fristen der Gründung von der Gewerbeanmeldung bis zum Transparenzregister. Einzelheiten in [docs/anforderung-gruendung.md](./docs/anforderung-gruendung.md).
13. **Mandanten & Verschlüsselung**
   - Mehrere Unternehmen nebeneinander, je eigener Datenordner und eigener Schlüssel.
   - Felder mit personenbezogenem oder geschäftlichem Inhalt liegen mit AES-256-GCM verschlüsselt in der Datenbank, dazu eine Wiederherstellungsdatei für den Fall eines verlorenen Schlüsselbunds. Siehe [docs/security-concept.md](./docs/security-concept.md).

---

## Speicherung & Integrität

```text
buchfink-data/                        # Datenordner eines Mandanten
├── buchfink.sqlite                   # Konfiguration, Konten, Journal, Belegsätze
├── buchfink.keyfile.json             # Gewrappter Datenschlüssel, zwei Slots
├── belege/
│   └── 2026/
│       ├── eingang/
│       │   └── 3f7a…c1.pdf           # Ablage unter dem eigenen SHA256
│       └── ausgang/
│           └── 9b2e…44.pdf
└── dokumente/
    └── contract/                     # Dokumente am Anlagegut, je Art ein Ordner
        └── 71d0…8a.pdf
```

- **Hash-Chain:** Jede Buchung enthält den SHA256-Hash der vorangehenden (Git-Prinzip). Manipulationen an vergangenen Perioden werden sofort sichtbar.
- **Festschreibung:** Ein festgeschriebener Zeitraum nimmt keine rückdatierten Buchungen mehr an. Der Kettenkopf wird dabei durch einen RFC-3161-Zeitstempel eines unabhängigen Dienstes beglaubigt.
- **Isolation:** Eine SQLite-Datei je Mandant, das Geschäftsjahr ist ein Feld an der Buchung. Auswertungen und Erfassung laufen immer gegen das aktive Jahr.
- **Belegintegrität:** Originaldateien bleiben unverändert. Der Dateiname ist der SHA256 des Inhalts, gleicher Inhalt wird nur einmal abgelegt.
- **Verschlüsselung:** Datenbankfelder mit personenbezogenem oder geschäftlichem Inhalt sind mit AES-256-GCM verschlüsselt. Die Belegdateien selbst bleiben bewusst im Original, weil die GoBD den unveränderten Eingangsbeleg verlangt.

> Eine Sicherung des Datenordners und ein Rückspielweg fehlen bisher. Wer Buchfink
> ernsthaft einsetzt, sichert den Ordner heute selbst.

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
| **Datenbank** | SQLite (Pure Go) | Eine SQLite-Datei je Mandant, CGO-frei, Feldverschlüsselung über einen GORM-Serializer |
| **Frontend** | React, TypeScript, Vite | Schnelles, reaktives UI ohne schweren Design-System-Overhead |
| **UI-Bausteine** | [Base UI](https://base-ui.com) | Unstyled: Fokusfang, Positionierung und Tastaturführung. Gestalt kommt aus dem eigenen Design-System |
| **Styling** | Tailwind CSS & Lucide Icons | Minimalistisch, flach und warm gestaltet |
| **Dokumente** | Typst | Layout-Engine für ZUGFeRD PDF/A-3 Rechnungen |

---

## Entwicklung & Weiterentwicklung

### Voraussetzungen

- **Go:** `>= 1.25` ([Download](https://golang.org/dl/))
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
│   │   │   └── ui/         # Bausteine des Design-Systems
│   │   ├── pages/          # Ansichten (Journal, Konten, Bank, Auswertungen, ...)
│   │   ├── services/       # Wails Go-Bindings / API Client
│   │   ├── types/          # Gemeinsame TypeScript-Typen
│   │   └── utils/          # Formatierung (Währung, Datum), Hilfsfunktionen
│   └── package.json
├── internal/               # Go Backend Module
│   ├── accounting/         # SKR04-Kontenplan, Buchungsgruppen, Steuerschlüssel, AfA, Hash-Chain
│   ├── bank/               # CAMT.053-Parser
│   ├── currency/           # EZB-Referenzkurse
│   ├── domain/             # Fachliche Typen und Repository-Schnittstellen
│   ├── ebilanz/            # XBRL-Zuordnung & Instanzerzeugung
│   ├── einvoice/           # E-Rechnung: CII, UBL, ZUGFeRD, XRechnung, Regelwerk
│   ├── invoice/            # Ausgangsrechnung: ZUGFeRD-XML & Typst-Rendering
│   ├── receiptstore/       # Belegablage unter SHA256
│   ├── repository/         # GORM/SQLite-Persistenz & Feldverschlüsselung
│   ├── security/           # Vault, Schlüsselbund, Wiederherstellung
│   ├── service/            # Anwendungsfälle (Buchen, Zahlen, Anlagen, Belege, ...)
│   ├── timestamp/          # RFC-3161-Zeitstempel für die Festschreibung
│   └── wailsbridge/        # Aufrufbare Oberfläche für das Frontend
├── scripts/                # Prüf- und Erzeugungsskripte
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

## Scope & Entscheidungen

| Thema | Entscheidung in Buchfink |
|---|---|
| **Anwendungsbereich & Zielgruppe** | **Ausschließlich bilanzierende Unternehmen** (z. B. UG, GmbH, AG, bilanzierende Kaufleute). **Nicht geeignet** für kleine Selbstständige, Freiberufler oder Kleinunternehmer mit einfacher Einnahmen-Überschuss-Rechnung (EÜR). |
| **Buchungsansatz** | Doppelte Buchführung (Soll & Haben) nach dem Prinzip „Buchung folgt Bankumsatz“: Transaktionen werden Belegen zugeordnet und generieren automatisch Soll/Haben-Sätze. |
| **Kontenrahmen** | SKR04 als Standard für v1 (Abschlussgliederungsprinzip für Bilanz & GuV). |
| **GoBD** | Unveränderbarkeit, Hash-Chains, Storno-Prinzip und Festschreibung sind ab Tag 1 gebaut. Der Datenexport für die Betriebsprüfung (Z3) fehlt noch. |
| **E-Bilanz / ERiC** | Buchfink erzeugt die XBRL-Datei selbst, inklusive Kontennachweis und Anlagenspiegel. Direkte ERiC-Übermittlung ist bewusst out-of-scope (proprietäre C-Bibliothek); Einreichung erfolgt über Mein ELSTER oder Bridges (z. B. eBilanz+). Der Export ist heute ein Gerüst, siehe [Stand der Umsetzung](./docs/stand-der-umsetzung.md). |
| **Out-of-Scope (v1)** | Einnahmen-Überschuss-Rechnung (EÜR), Lohnbuchhaltung, Lagerverwaltung/Inventur, mehrsprachige UI (v1 fokussiert auf Deutsch/DACH). |

---

## Stand der Umsetzung

Buchfink ist in Entwicklung, und das README beschreibt das Ziel. Was davon im Code
angekommen ist, steht mit Fundstellen in
[docs/stand-der-umsetzung.md](./docs/stand-der-umsetzung.md). Die größten offenen
Punkte in Kurzform:

| Lücke | Wirkung |
|---|---|
| **Jahreswechsel und Saldenvortrag** | Kontensalden entstehen nur aus den Buchungen des aktiven Jahres. Ein Bestandskonto zeigt im zweiten Jahr nicht seinen Bestand. Damit endet eine Buchhaltung heute nach einem Geschäftsjahr. |
| **Abschlussbuchungen** | Es gibt den Abschreibungslauf. Der Abschluss der Erfolgskonten, die Umsatzsteuer-Verrechnung, die Steuerrückstellung und die Ergebnisverwendung fehlen. |
| **Umsatzsteuer-Voranmeldung** | Vorhanden ist eine Auswertung mit vier Kennziffern, kein vollständiger Vordruck und keine Übermittlungsdatei. Ebenso fehlen Dauerfristverlängerung und Zusammenfassende Meldung. |
| **Rechnungsabgrenzung, Rückstellungen, Anzahlungen** | Für die Bilanz nicht verzichtbar, jeweils beschrieben und noch nicht gebaut. |
| **DATEV- und Z3-Export** | Ohne sie bleibt die Buchhaltung eine Insel und die Betriebsprüfung ohne Datenträger. |
| **Datensicherung** | Kein Sicherungs- und Rückspielweg. Bei einer Anwendung, die alles lokal hält, das größte Betriebsrisiko. |
| **Mahnwesen, Kassenbuch, wiederkehrende Buchungen** | Alltagsfunktionen, die noch fehlen. |

Ausführlich, mit den offenen Punkten aus dem Zahlungsverkehr, dem Bankimport und
der Ausgangsrechnung: [docs/stand-der-umsetzung.md](./docs/stand-der-umsetzung.md).

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
