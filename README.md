<p align="center">
  <img src="./assets/buchfink-logo.svg" alt="Buchfink Logo" width="200" />
</p>

<h1 align="center">Buchfink</h1>

<p align="center">
  <strong>Moderne Open-Source-Buchhaltungssoftware für bilanzierende Unternehmen</strong><br />
  Native Desktop-App &bull; Doppelte Buchführung &bull; SKR04 &bull; Bilanz & GuV &bull; GoBD-konform ab v1 &bull; E-Bilanz (XBRL) &bull; Local-First
</p>

<p align="center">
  <em>In Entwicklung, vor der ersten Erprobung. Was heute trägt, wo eine Funktion an
  ihrer Grenze endet und was noch fehlt, steht in
  <a href="./docs/stand-der-umsetzung.md">docs/stand-der-umsetzung.md</a>, jedes
  Kriterium mit Norm und Fundstelle in
  <a href="./docs/anforderungskatalog.md">docs/anforderungskatalog.md</a>.</em>
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

Buchfink ist für Menschen ohne Buchhaltungsausbildung gebaut: die Oberfläche
spricht in Vorgängen statt in Konten, und die Norm steht hinter dem
Erklärzeichen, nicht in der Arbeitsansicht. Anwender gibt es noch keine — der
Stand ist vor der ersten Erprobung.

Buchfink wird ohne Gewährleistung bereitgestellt (Artikel 7 und 8 der
[EUPL-1.2](LICENSE)). Wer damit bucht, prüft die Ergebnisse selbst: für die
Richtigkeit der Buchführung, der Voranmeldung und des Abschlusses haftet das
Unternehmen, das sie abgibt.

### ⚠️ Wichtiger Hinweis: Nicht geeignet für EÜR (Einnahmen-Überschuss-Rechnung)
Buchfink ist **nicht für kleine Selbstständige, Freiberufler oder Kleinunternehmer geeignet**, die lediglich eine einfache **Einnahmen-Überschuss-Rechnung (EÜR nach § 4 Abs. 3 EStG)** durchführen.
- Buchfink unterstützt **keine EÜR**.
- Die Software basiert vollständig auf dem geschlossenen System der doppelten Buchführung mit Soll und Haben, Bestandskonten (Aktiva/Passiva), Erfolgskonten (Aufwand/Ertrag), Bilanzierung und der amtlichen E-Bilanz-Taxonomie.

---

### Grundprinzipien

- **Local-First:** Alle Daten verbleiben auf dem eigenen Rechner in einer standardisierten SQLite-Datei je Mandant.
- **GoBD-konform ab v1:** Lückenlose Nachvollziehbarkeit durch kryptografische Hash-Chains, unveränderliche Belegablage, Festschreibung mit Zeitstempel und ein verkettetes Änderungsprotokoll. Für die Betriebsprüfung entsteht der Z3-Export nach dem Beschreibungsstandard.
- **Der Jahreslauf ist das Rückgrat:** Die Aufgabenliste ist die Startseite und sagt, was heute zu tun ist. Der Monatsabschluss läuft in drei Schritten (Prüfbericht, Festschreiben, Voranmeldung), der Jahresabschluss als geführter Weg durch die Abschlussbausteine. Siehe [docs/architektur.md](./docs/architektur.md), Abschnitt 6.
- **Automatisierungsfokus:** Der Alltag beginnt beim Kontoauszug: eine Zahlung wird ihrem offenen Posten oder einem Beleg zugeordnet, den Buchungssatz und die Steuer rechnet Buchfink daraus. Den passenden Posten schlägt Buchfink vor, gebucht wird erst nach Bestätigung.
- **E-Rechnung:** Eingehende ZUGFeRD-, Factur-X- und XRechnung-Dateien werden erkannt, gelesen und gegen das Regelwerk geprüft. Ausgestellt wird als ZUGFeRD-PDF oder als XRechnung im CII-Profil.
- **Verschlüsselt abgelegt:** Personenbezogene und geschäftliche Datenbankfelder liegen mit AES-256-GCM verschlüsselt, der Schlüssel im Schlüsselbund des Betriebssystems.

---

## Kernfunktionen

1. **Kontenverwaltung (SKR04)**
   - Vorinstallierter SKR04-Kontenrahmen mit Such- und Hilfefunktion für steuerliche Einsteiger.
2. **Journal aus dem Belegfluss**
   - Transparente Soll/Haben-Ansicht. Den Buchungssatz und die Steuer rechnet Buchfink aus dem erfassten Beleg, aus der Rechnung oder aus der zugeordneten Zahlung.
   - Lückenlose Belegnummerierung und GoBD-Korrekturen ausschließlich per Storno, mit Verweis auf die Neubuchung.
3. **Geschäftsjahr und Saldenvortrag**
   - Das Geschäftsjahr ist eine Entität mit Beginn, Ende, Rumpfjahr-Kennzeichen, Vortragsstand und Abschlussstatus.
   - Der Saldenvortrag bringt Bestandskonten und offene Posten ins Folgejahr; eine Differenz wird ausgewiesen und lässt sich durch Storno und Neuvortrag korrigieren.
4. **Kunden & Lieferanten (Offene Posten)**
   - Stammdatenverwaltung, OPOS-Liste zum Stichtag mit Altersstruktur und Restlaufzeiten, Zahlungsausgleich mit Teilzahlung, Skonto und Differenzgründen, Ausbuchung uneinbringlicher Posten.
   - Mahnwesen mit konfigurierbaren Stufen, taggenauen Verzugszinsen nach § 288 BGB auf dem datierten Basiszinssatz und der Pauschale von 40 Euro; das Mahnschreiben wird als Dokument abgelegt.
5. **Rechnungswesen (Ausgangsrechnungen)**
   - Rechnungslayout via [Typst](https://typst.app/), Ausgabe als ZUGFeRD-/Factur-X-konformes PDF/A-3 mit eingebettetem XML oder als XRechnung im CII-Profil, Zielformat je Empfänger.
   - Rechnungsnummer, Datensatz und Buchung entstehen in einer Transaktion; die Pflichtangaben des § 14 Abs. 4 UStG werden vor der Nummernvergabe geprüft, Nummernlücken meldet der Prüfbericht.
   - Storno- und Korrekturbeleg als eigene Dokumente mit Bezug in beide Richtungen, Kleinbetragsrechnung nach § 33 UStDV, Anzahlungen als Rechnungsverbund mit Vereinnahmung und Schlussrechnung.
6. **E-Rechnungs-Empfang**
   - Erkennt ZUGFeRD, Factur-X und XRechnung im eingegangenen Beleg, liest CII und UBL und prüft beides gegen das Regelwerk der Norm.
   - Aus dem gelesenen Datensatz entsteht ein Buchungsvorschlag; der Beleg bleibt unverändert, wie er ankam.
7. **Bankumsatz-Import (CAMT.053)**
   - Import von standardisierten CAMT.053-Bankauszügen.
   - Zu einem Umsatz schlägt Buchfink den offenen Posten vor — nach Betrag, Verwendungszweck, Rechnungsnummer und Datumsnähe, mit Sammelzahlung und gelernten Regeln. Gebucht wird nach Bestätigung.
8. **Anlagevermögen (Anlagenverzeichnis, AfA, Anlagenspiegel)**
   - Verzeichnis für Sach-, Finanz- und immaterielle Anlagen mit Inventarnummer, Bewegungen und jahresübergreifender Kartei.
   - Wertgrenzen des § 6 Abs. 2 und 2a EStG (GWG, Sammelposten), lineare und degressive AfA mit automatischem Übergang, Staffel des § 7 Abs. 2a EStG, Gebäudesätze des § 7 Abs. 4 EStG, Sonderabschreibung nach § 7g Abs. 5 EStG samt Restwertverteilung des § 7a Abs. 9 EStG, außerplanmäßige Abschreibung und Zuschreibung — die Sätze stehen als datierte Ressource neben dem Code.
   - Abschreibungslauf als Abschlussbuchung mit Vorschau; die Jahres-Festschreibung prüft vorher, ob die AfA vollständig gebucht ist.
   - Fertigstellung von Anlagen im Bau, Erhaltungsaufwand und laufende Erträge am Anlagegut, Stückzahlen und Fremdwährungsbewertung nach § 256a HGB bei Finanzanlagen.
   - Verträge, Gutachten, Zulassungen und Policen am Anlagegut ablegen — mit Ablaufdatum, das wieder gelesen wird.
   - Darlehen und Ausleihungen mit Fälligkeit und Tilgung als eigenem Weg: eine Rückzahlung ist kein Verkauf und erzeugt keinen Erlös.
   - Investmentanteile (ETF, Aktien- und Immobilienfonds): Teilfreistellung nach § 20 InvStG und Vorabpauschale nach § 18 InvStG als außerbilanzielle Nebenrechnung.
   - Skonto auf eine Anlagenrechnung mindert im Zahlungsflow die Anschaffungskosten (§ 255 Abs. 1 Satz 3 HGB) statt den Aufwand.
   - Abgang mit Erlöskonto nach Buchgewinn oder -verlust, Teilabgang nach Stück bei Finanzanlagen und Anlagenspiegel nach § 284 Abs. 3 HGB — auch als Kontennachweis in der E-Bilanz.
9. **Abschlussbausteine**
   - Rechnungsabgrenzung mit monatlicher Auflösung, Rückstellungen mit Abzinsung, Verbrauch und Auflösung, Inventurwert der Vorräte als Bestandsveränderung.
   - Umsatzsteuer-Verrechnung auf die Zahllast, Steuerrückstellung, Abschluss der Erfolgskonten und Ergebnisverwendung mit Gesellschafterbeschluss.
   - Verzeichnis der steuerlichen Wahlrechte nach § 5 Abs. 1 S. 2 EStG und die Überleitungsrechnung zur Steuerbilanz.
10. **Auswertungen**
   - Kontenblatt, Summen- und Saldenliste zu jedem Stichtag, Journal mit Volltextsuche über Jahresgrenzen und Ausgabe als CSV.
11. **Bilanz und GuV nach §§ 266, 275 HGB**
   - Gliederung im Backend aus denselben Kontensalden, mit Vorjahresspalte, Bilanzgewinn und Ausgabe als PDF und CSV.
   - Größenklasse nach §§ 267, 267a HGB aus zwei aufeinanderfolgenden Stichtagen; sie setzt Gliederungstiefe, Anhangumfang und Offenlegungsumfang.
   - Anhang mit Anlagenspiegel und Restlaufzeitengliederung aus den Buchungsdaten; für Kleinstgesellschaften die Angaben unter der Bilanz.
12. **Umsatzsteuer und Meldewesen**
   - Voranmeldung mit allen Kennziffern des Vordrucks USt 1 A aus den Steuerzeilen der Buchungen, mit Drill-down bis zur Buchung, Berichtigung, Dauerfristverlängerung und Ausgabe als Kennziffernblatt für Mein ELSTER.
   - Zusammenfassende Meldung nach § 18a UStG mit Zeitraumbestimmung und Meldezeilen je USt-IdNr.
   - Übermittlungsprotokoll mit Datum und Transferticket, nach der Erfassung unveränderlich.
   - Prüfläufe vor jeder Festschreibung: dreizehn Regeln über Buchungen, Belege, Bank, Nummernkreise und Voranmeldung, im Jahreslauf vier weitere; blockierende Befunde halten die Festschreibung auf.
   - Fristenseite mit Steuerterminen, Abschlussterminen und Gründungspflichten.
13. **Steuerliche Nebenpflichten**
   - Vorsteuerabzug an die geprüfte Rechnung gekoppelt, Vorsteuerschlüssel mit Aufteilungsmaßstab nach § 15 Abs. 4 UStG.
   - Verzeichnis der Vorsteuerberichtigung nach § 15a UStG mit Meldung in Kennziffer 64.
   - Qualifizierte Bestätigung der USt-IdNr. beim Bundeszentralamt vor der steuerfreien Lieferung, Belegnachweis nach §§ 17a, 17b UStDV je Lieferung.
   - Geschenke und die weiteren Kategorien des § 4 Abs. 5 und 7 EStG auf eigenen Konten, Bewirtungsanteil als datierter Parameter.
14. **E-Bilanz-Export (XBRL)**
   - Die Instanz entsteht aus derselben Gliederung wie die Bilanz: Bilanz- und GuV-Positionen mit Vorjahreskontext, unverdichteter Kontennachweis, Anlagenspiegel und Überleitungsrechnung.
   - Die Taxonomie liegt als Ressource neben dem Code (`internal/ebilanz/taxonomy_6.9.json`). Ihre Elementnamen sind nach der Systematik gebildet und tragen `verified: false`; vor der ersten Übermittlung sind sie gegen die amtliche Fassung abzugleichen.
15. **Fremdwährung**
   - Kurs, Kursquelle und Kursdatum hängen an der Buchung und gehen in die Hash-Chain ein; Finanzanlagen werden nach § 256a HGB zum Stichtag bewertet.
   - Die EZB-Referenzkurse werden abgerufen und lassen sich von Hand nachtragen.
16. **Festschreibung, Änderungsprotokoll & Integrität**
   - Live-Prüfung der Hash-Chain in der Oberfläche.
   - Festschreibung je Monat, Quartal oder Jahr, beglaubigt durch einen RFC-3161-Zeitstempel über den Kettenkopf; der Zeitpunkt steht an der einzelnen Buchung.
   - Änderungsprotokoll mit Vorher und Nachher in einer eigenen Hashkette, dazu Bearbeiterkennung, Programmfassung und Zeit in UTC an jeder Buchung.
17. **Aufbewahrung, Sicherung und Datenzugriff**
   - Aufbewahrungsfrist je Beleg aus seiner Art, mit Fristbeginn, frühestem Löschdatum und Aussetzung je Geschäftsjahr; die Löschung eines abgelaufenen Jahres ist ein Vorgang mit Archivexport davor und Protokoll danach.
   - Z3-Export nach dem Beschreibungsstandard (Datendateien, `index.xml`, Feldbeschreibung, Schlüsselverzeichnis, Prüfpfad), Archivexport der Belege mit Index, Prüferpaket in einem Ordner.
   - Schreibgeschützter Prüfermodus mit Frist und Grund, protokolliert beim Ein- und Ausschalten.
   - Sicherung als offene ZIP-Datei mit Datenbank, Belegen, Dokumenten und Schlüsseldatei, Wiederherstellung mit anschließender Integritätsprüfung.
   - Verfahrensdokumentation aus dem laufenden System, als Fassung im Belegspeicher abgelegt.
18. **Gründung einer Kapitalgesellschaft**
   - Erfassung im Einrichtungsassistenten: Beurkundungsdatum, Stammkapital, Gesellschafter und der Gründungsaufwand laut Satzung. Aus dem Beurkundungsdatum folgen Rumpfgeschäftsjahr und Voranmeldungszeitraum.
   - Prüfung der Kapitalaufbringung vor der Anmeldung zum Handelsregister: Viertelregel je Geschäftsanteil und Untergrenze nach § 7 Abs. 2 GmbHG, Volleinzahlung und Sacheinlageverbot der UG nach § 5a Abs. 2 GmbHG, § 36a AktG bei der AG.
   - **Unterbilanzhaftung**: laufende Rechnung, um wie viel das Reinvermögen hinter dem Stammkapital zurückbleibt, aufgeteilt auf die Gesellschafter. Mit der Eintragung steht sie fest.
   - Gründungsbuchungen als Vorschlag mit Vorschau, und die Fristen der Gründung von der Gewerbeanmeldung bis zum Transparenzregister. Einzelheiten in [docs/anforderung-gruendung.md](./docs/anforderung-gruendung.md).
19. **Mandanten & Verschlüsselung**
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

> Buchfink sichert den Datenordner beim Beenden und einmal täglich in einen
> gewählten Zielordner und bietet beim Start eine Wiederherstellung an, nach der
> die Integritätsprüfung läuft. Die Sicherung ist eine ZIP-Datei mit Datenbank,
> Belegen, Dokumenten und Schlüsseldatei; sie lässt sich ohne die Software öffnen.

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

4. **Prüfen, was vor einem Commit läuft:**
   ```bash
   task check          # gofmt, vet, Tests, check:design, check:bridge, check:text, check:steps
   task check:text     # Norm in einer Arbeitsansicht statt hinter dem Erklärzeichen
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
│   ├── check_ui_text.py    # Norm nur hinter dem Erklärzeichen (task check:text)
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

## Scope & Entscheidungen

Die Tabelle gibt die Grundentscheidungen aus [docs/architektur.md](./docs/architektur.md),
Abschnitt 2, wieder. Jede wirkt auf den Anforderungskatalog, wo die betroffenen
Kriterien den Status `⛔` tragen.

| Thema | Entscheidung in Buchfink |
|---|---|
| **Anwendungsbereich & Zielgruppe** | **Ausschließlich bilanzierende Unternehmen** (z. B. UG, GmbH, AG, bilanzierende Kaufleute). **Nicht geeignet** für kleine Selbstständige, Freiberufler oder Kleinunternehmer mit einfacher Einnahmen-Überschuss-Rechnung (EÜR). |
| **Buchungsansatz** | Doppelte Buchführung (Soll & Haben) nach dem Prinzip „Buchung folgt Bankumsatz“: Transaktionen werden Belegen zugeordnet und generieren automatisch Soll/Haben-Sätze. |
| **Einzelplatz, ein Bearbeiter** | Keine Benutzerverwaltung, keine Rollen, keine Funktionstrennung im System. An ihrer Stelle stehen die Bearbeiterkennung (Betriebssystem-Benutzer und Rechnername) an jeder Buchung und jeder Protokollzeile sowie ein schreibgeschützter Prüfermodus für Dritte. |
| **Kontenrahmen** | SKR04 als Einheitsbilanz: ein Kontenrahmen, ein Wertansatz. Abweichende steuerliche Werte entstehen nur aus der Sonderabschreibung nach § 7g Abs. 5 EStG und werden am Anlagegut mitgeführt; daraus entstehen das Verzeichnis nach § 5 Abs. 1 S. 2 EStG und die Überleitungsrechnung. Latente Steuern entfallen für kleine Kapitalgesellschaften nach § 274a Nr. 4 HGB. |
| **Steuerfälle** | Geschlossene Liste. Unterstützt: Inland 19 %, 7 %, 0 %, steuerfrei, innergemeinschaftlicher Erwerb und Lieferung, Reverse Charge als Empfänger und als Leistender, Ausfuhr. Ausgeschlossen: Kleinunternehmer, Differenzbesteuerung, Reiseleistungen, OSS/IOSS, Dreiecksgeschäft, Konsignationslager, Bauleistungen nach § 13b Abs. 2 Nr. 4 UStG. Die Oberfläche sagt bei einem ausgeschlossenen Fall, dass Buchfink ihn nicht abbildet. |
| **GoBD** | Unveränderbarkeit, Hash-Chains, Storno-Prinzip, Festschreibung mit Zeitstempel und ein verkettetes Änderungsprotokoll sind ab Tag 1 gebaut. Der Datenexport für die Betriebsprüfung (Z3) entsteht nach dem Beschreibungsstandard mit `index.xml`, Feldbeschreibung und Schlüsselverzeichnis; ein Testeinlesen in eine Prüfsoftware steht aus. |
| **E-Bilanz / ERiC** | Buchfink erzeugt die XBRL-Datei selbst, aus derselben Gliederung wie die Bilanz, inklusive Kontennachweis, Anlagenspiegel und Überleitung. Die Taxonomie-Ressource `internal/ebilanz/taxonomy_6.9.json` trägt durchgehend `verified: false` — ihre Elementnamen sind vor der ersten Übermittlung gegen die amtliche Fassung abzugleichen. Direkte ERiC-Übermittlung ist bewusst out-of-scope (proprietäre C-Bibliothek); die Einreichung läuft über Mein ELSTER oder den Steuerberater, das Übermittlungsprotokoll wird danach von Hand erfasst. |
| **Rechtsformen** | Kapitalgesellschaften zuerst. Gründungsweg, Kapitalaufbringung, Größenklassen und Offenlegung sind für UG, GmbH und AG gebaut. KG, OHG und e.K. bleiben wählbar und tragen in der Oberfläche den Hinweis, dass Kapitalkonten und Entnahmen in dieser Fassung nicht abgebildet sind. |
| **Out-of-Scope (v1)** | Einnahmen-Überschuss-Rechnung (EÜR), Kassenbuch, Lagerverwaltung, Lohnabrechnung (der Lohn kommt als Sammelbuchung aus dem Lohnjournal, die Vorräte als Inventurwert zum Stichtag), ersetzendes Scannen, Versandwege wie Peppol oder EDI, mehrsprachige UI. |
| **DATEV-Export** | Nicht vorhanden. Buchfink schreibt keinen Buchungsstapel und keine Belegbilder im Format der Kanzleisoftware. Für die Übergabe an den Steuerberater stehen der Z3-Export nach dem Beschreibungsstandard, der Archivexport der Belege und die CSV-Ausgabe von Journal, Konten und Auswertungen. |
| **Gewährleistung** | Keine. Buchfink wird nach Artikel 7 und 8 der EUPL-1.2 „so, wie es ist“ und ohne Gewährleistung bereitgestellt; für die Richtigkeit der Buchführung und der Meldungen haftet das Unternehmen, das sie abgibt. |

---

## Stand der Umsetzung

Der [Anforderungskatalog](./docs/anforderungskatalog.md) misst Buchfink an 349
Akzeptanzkriterien aus 82 Anforderungen und nennt zu jedem die Fundstelle im
Code. Sieben Umsetzungswellen sind gebaut; nach Welle 7 folgt keine weitere.

| Modul | ✅ erfüllt | 🟡 teilweise | ❌ fehlt | ⛔ außerhalb | Kriterien |
|---|---|---|---|---|---|
| A. Buchführungspflicht und Grundsätze | 17 | 5 | 0 | 0 | 22 |
| B. Beleg, Journal, Konten | 26 | 4 | 1 | 4 | 35 |
| C. Unveränderbarkeit und Protokollierung | 17 | 2 | 0 | 3 | 22 |
| D. Aufbewahrung und Archivierung | 22 | 6 | 0 | 3 | 31 |
| E. Ausgangsrechnungen und E-Rechnung | 32 | 5 | 0 | 6 | 43 |
| F. Umsatzsteuer, Aufzeichnung und Meldewesen | 24 | 5 | 0 | 13 | 42 |
| G. Bewertung, Anlagen, Fremdwährung | 33 | 12 | 3 | 15 | 63 |
| H. Jahresabschluss, E-Bilanz, Offenlegung | 24 | 9 | 8 | 3 | 44 |
| I. Betriebsprüfung und Verfahrensdokumentation | 10 | 4 | 2 | 7 | 23 |
| J. Querschnitt | 18 | 4 | 1 | 1 | 24 |
| **Summe** | **223** | **56** | **15** | **55** | **349** |

Die fünfzehn offenen Punkte in Kurzform:

- **Der Jahresabschluss als Dokument.** Feststellungsbeschluss, unterzeichneter Abschluss und Prüfungsvermerk lassen sich nirgends anhängen; daran hängt auch die Frist des § 42a GmbHG gegenüber den Gesellschaftern.
- **Offenlegung.** Kein Datensatz für das Unternehmensregister, keine Hinterlegung nach § 326 Abs. 2 HGB als Wahl, kein Einreichungsnachweis. Für die E-Bilanz fehlt das Übermittlungsprotokoll, das an der Voranmeldung schon steht.
- **Bewertung.** Kein Bericht über geänderte Bewertungsmethoden, keine getrennte Erfassung der Pflicht- und Wahlbestandteile der Herstellungskosten, die Befreiung von den latenten Steuern hängt nicht an der Größenklasse.
- **Belege und Abstimmung.** Kein Erfassungsweg für Eigenbelege, kein Saldenbestätigungslauf für Debitoren und Kreditoren.
- **Nachweise außerhalb des Programms.** Ein Testeinlesen der Datenüberlassung in eine Prüfsoftware hat nicht stattgefunden; der Prüfpunkt zur Verordnung nach § 147b AO steht seit dieser Fassung in [docs/architektur.md](./docs/architektur.md), Abschnitt 4.

Wo eine Funktion an ihrer Grenze endet und was bewusst außerhalb des Umfangs
liegt, steht mit Fundstellen in
[docs/stand-der-umsetzung.md](./docs/stand-der-umsetzung.md); die Entscheidungen
dahinter in [docs/architektur.md](./docs/architektur.md), Abschnitt 2, und die
Wellen in Abschnitt 7.

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
- **Ohne Gewährleistung:** Das Werk wird „so, wie es ist“ bereitgestellt, ohne
  Gewährleistung für Rechtsmängel, Marktgängigkeit, Eignung für einen bestimmten
  Zweck oder Fehlerfreiheit (Artikel 7 EUPL). Der Lizenzgeber haftet außer bei
  Vorsatz und Personenschäden nicht für Schäden aus der Benutzung des Werks
  (Artikel 8 EUPL). Für Buchhaltungssoftware heißt das: die Prüfung der
  Ergebnisse bleibt beim Unternehmen.

Mitgelieferte Komponenten Dritter behalten ihre eigenen Lizenzen und sind in
[THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md) aufgeführt.
