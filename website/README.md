# Projektseite

Der Inhalt dieses Verzeichnisses ist die Projektseite von Buchfink, so wie sie
auf GitHub Pages ausgeliefert wird. Es gibt keinen Build-Schritt: Was hier
liegt, ist genau das, was im Netz steht.

```text
website/
├── index.html          # Ablauf, Funktionen mit Screenshots, Abgrenzung, Preis
├── preis.html          # kostenlos: was das heißt und was nicht
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

Drei Stellen gehen bewusst über das Konzept hinaus, weil eine Textseite im
Browser andere Anforderungen hat als eine Arbeitsansicht:

- **Schriftgrade.** Die App-Skala endet bei 22 px, weil dort nichts gelesen,
  sondern gearbeitet wird. Die Projektseite ist Fließtext, deshalb liegt der
  Grundtext bei 15/26 px und es gibt zwei Grade über `text-display` für Titel.
  Die Schrift, die Gewichte (400/500/600) und der tabellarische Zahlensatz
  bleiben unverändert.
- **Rasterhintergrund im Kopf.** Das einzige dekorative Element der Seite: ein
  ausgeblendetes Millimeterraster hinter dem Titel. Es benutzt die Haarlinie
  `--line` und keine eigene Farbe.
- **Leiserer Hinweisstreifen.** In der Anwendung ist ein Hinweis pastellig
  ausgefüllt, weil er den Lesefluss unterbrechen soll. Auf einer Seite, die man
  einmal von oben nach unten liest, zieht dieselbe Fläche den Blick vom Inhalt
  weg. Hier trägt deshalb nur die senkrechte Leiste die Farbe der Familie; die
  Bedeutung bleibt ablesbar, ohne dass der Abschnitt leuchtet.

Zwei Regeln des Konzepts sind für die Seite besonders wichtig und werden streng
eingehalten:

- **Alle Funktionsblöcke sind gleich ausgerichtet** — Text links, Bild rechts.
  Ein Wechsel der Seite von Block zu Block sieht nach Abwechslung aus, kostet
  aber bei jedem Block einen neuen Lesestart.
- **Diagramme liegen ohne Rahmen direkt auf dem Papier.** Eine Fläche darum wäre
  eine Karte (§6.2), und die Kästen im Bild bringen ihre Abgrenzung schon mit.

Wer die Seite ändert, kann sich an derselben Prüfliste orientieren, die am Ende
des Design-Konzepts steht.

## Diagramme

Die drei Schaubilder auf `index.html` — der Ablauf eines Monats, die Ablage der
Daten und die Hash-Kette — sind handgeschriebenes SVG direkt im HTML. Kein
Werkzeug, keine Bibliothek, kein Build. Sie benutzen dieselben Farbwerte wie der
Rest der Seite; da SVG keine CSS-Variablen erbt, stehen die Werte dort als
Hex-Literale. Wer eine Farbe ändert, ändert sie an beiden Stellen.

Jedes Diagramm trägt `<title>` und `<desc>` und ist über `aria-labelledby`
damit verbunden, damit es auch vorgelesen brauchbar bleibt.

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
