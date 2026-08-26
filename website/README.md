# Projektseite

Der Inhalt dieses Verzeichnisses ist die Projektseite von Buchfink, so wie sie
auf GitHub Pages ausgeliefert wird. Es gibt keinen Build-Schritt: Was hier
liegt, ist genau das, was im Netz steht.

```text
website/
├── index.html          # Projekt, Grundsätze, Funktionen mit Screenshots
├── installation.html   # Werkzeuge, Bauen, Prüfen, Stolpersteine
├── .nojekyll           # Pages soll nichts umbauen
└── assets/
    ├── site.css        # das komplette Stylesheet
    ├── buchfink-logo.svg
    ├── fonts/          # Manrope, selbst ausgeliefert (OFL, Lizenz liegt bei)
    └── screenshots/    # aus der laufenden Oberfläche, siehe unten
```

## Veröffentlichen

`.github/workflows/pages.yml` lädt das Verzeichnis bei jedem Push auf `master`
hoch, der `website/` berührt. Einmalig muss im Repository unter
**Settings → Pages** als *Source* **GitHub Actions** eingestellt sein.

Lokal ansehen genügt ein beliebiger statischer Server:

```bash
python3 -m http.server 8000 --directory website
```

## Gestaltung

Die Seite benutzt dieselben Tokens wie die Anwendung — die Farbwerte, Radien,
Abstände und Bewegungsregeln aus [`docs/design-konzept.md`](../docs/design-konzept.md)
stehen als CSS-Variablen am Anfang von `assets/site.css`. Auch die Regeln
gelten weiter: keine Karten, Abschnitte durch Überschrift, Abstand und
Haarlinie getrennt, Schatten nur an schwebenden Elementen, Primäraktion in
Tinte statt in Himmelblau, Farbe nur mit Bedeutung.

Zwei Stellen gehen bewusst über das Konzept hinaus, weil eine Textseite im
Browser andere Anforderungen hat als eine Arbeitsansicht:

- **Schriftgrade.** Die App-Skala endet bei 22 px, weil dort nichts gelesen,
  sondern gearbeitet wird. Die Projektseite ist Fließtext, deshalb liegt der
  Grundtext bei 15/26 px und es gibt zwei Grade über `text-display` für Titel.
  Die Schrift, die Gewichte (400/500/600) und der tabellarische Zahlensatz
  bleiben unverändert.
- **Rasterhintergrund im Kopf.** Das einzige dekorative Element der Seite: ein
  ausgeblendetes Millimeterraster hinter dem Titel. Es benutzt die Haarlinie
  `--line` und keine eigene Farbe.

Wer die Seite ändert, kann sich an derselben Prüfliste orientieren, die am Ende
des Design-Konzepts steht.

## Screenshots

Die Bilder in `assets/screenshots/` sind keine Nachbauten. Sie entstehen aus der
echten Oberfläche in `frontend/`; ersetzt ist nur die Wails-Bridge, an deren
Stelle Beispieldaten treten. Der Ablauf liegt in
[`scripts/site-screenshots/`](../scripts/site-screenshots/):

```bash
npm --prefix frontend install
npm --prefix scripts/site-screenshots install
node scripts/site-screenshots/shoot.mjs
```

Das Skript startet Vite mit `frontend/vite.screenshots.config.ts`, klickt sich
mit Playwright durch die Ansichten und schreibt die Bilder in dieses
Verzeichnis. Die Beispieldaten stehen in
`scripts/site-screenshots/mock-bridge.ts` und sind untereinander stimmig
gerechnet — Soll gleich Haben, Umsatzsteuer 19 % auf das Entgelt, Zahllast
gleich Umsatzsteuer minus Vorsteuer.

Alle Firmen, Namen, Beträge und Belege darin sind erfunden. Ändert sich eine
Ansicht in der Anwendung, genügt ein erneuter Lauf; ändert sich, welche Daten
eine Ansicht braucht, muss `mock-bridge.ts` mitgezogen werden.
