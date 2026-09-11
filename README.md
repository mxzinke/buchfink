# Projektseite

Der Inhalt dieses Verzeichnisses ist die Projektseite von Buchfink, so wie sie
auf GitHub Pages ausgeliefert wird. Es gibt keinen Build-Schritt: Was hier
liegt, ist genau das, was im Netz steht.

```text
website/
├── index.html          # Was Buchfink ist, der Alltag, die Pflichten, für wen, die Daten
├── buchhaltung.html    # Bank, Belege, Journal, Rechnungen, Anzahlungen, Konten
├── umsatzsteuer.html   # Voranmeldung, Meldungen, Fristen, Nebenpflichten
├── abschluss.html      # Monatsabschluss, Abschlussbausteine, Bilanz und GuV, E-Bilanz
├── nachweise.html      # Protokoll, Aufbewahrung, Sicherung, Prüferpaket
├── roadmap.html        # Zeitleiste: was steht, was bis v1 fehlt, Prüfpunkte mit Datum
├── installation.html   # Bauen je Betriebssystem, erster Start, Datenordner, Hilfe
├── .nojekyll           # Pages soll nichts umbauen
└── assets/
    ├── site.css        # das komplette Stylesheet
    ├── site.js         # das einzige Skript: Menü schließen, Dialoge öffnen
    ├── buchfink-logo.svg
    ├── fonts/          # Manrope, selbst ausgeliefert (OFL, Lizenz liegt bei)
    └── screenshots/    # aus der laufenden Oberfläche, siehe unten
```

Alle sieben Seiten haben dieselbe Navigation. Sie steht in jeder Datei als
Markup, weil es keinen Build-Schritt gibt: Wer einen Eintrag ändert, ändert ihn
siebenmal. Der Eintrag der aktuellen Seite trägt `aria-current="page"`.

Der Stylesheet-Link enthält eine Versionskennung, damit Browser nach Änderungen
die neue CSS-Datei laden. Bei Änderungen an `assets/site.css` wird `?v=…` in allen
sieben HTML-Dateien auf die ersten zwölf Zeichen ihres SHA-256-Hashs aktualisiert.
Den Hash liefert `shasum -a 256 website/assets/site.css` im Projektverzeichnis.

Die Navigation ist gruppiert: Start, ein Klappmenü **Funktionen** mit den vier
Funktionsseiten, dann Roadmap und Installation, rechts die Schaltfläche zu
GitHub. Im schmalen Fenster tritt an die Stelle der Reihe ein Schubfach mit
derselben Gliederung, dort mit Zwischenüberschriften. Beide Menüs sind
`<details>`-Elemente und funktionieren ohne JavaScript; `assets/site.js`
schließt nur ein offenes Menü wieder — beim Klick daneben, mit Escape und
sobald ein zweites aufgeht.

## Veröffentlichen

Von Hand, ohne Actions-Lauf:

```bash
task pages:publish        # git subtree push --prefix website origin gh-pages
```

Der Befehl schiebt den Inhalt von `website/` als Wurzel in den Zweig
`gh-pages`. Er muss nach jeder Änderung an der Seite laufen — sonst steht im
Netz weiter der alte Stand.

Einmalige Einrichtung im Repository unter **Settings → Pages**:
*Source: Deploy from a branch*, Zweig `gh-pages`, Ordner `/ (root)`.

Weist der Push zurück, weil `gh-pages` auseinandergelaufen ist, hilft die
ausdrückliche Form:

```bash
git push origin $(git subtree split --prefix website HEAD):gh-pages --force
```

### Warum ein eigener Zweig

Pages bedient beim Ausliefern aus einem Zweig nur dessen Wurzel oder den Ordner
`docs/` — ein beliebiges Unterverzeichnis wie `website/` geht nicht, und `docs/`
ist hier von den Fachkonzepten belegt. Deshalb der Umweg über `gh-pages`, in dem
`website/` die Wurzel ist.

Dabei lässt GitHub standardmäßig Jekyll über die Dateien laufen. `.nojekyll`
verhindert das und wandert beim `subtree push` mit in die Wurzel — deshalb muss
die Datei bleiben, wo sie ist.

### Lokal ansehen

```bash
task pages:preview        # http://127.0.0.1:8000
```

### Adresse

`buchfink.github.io` steht nicht zur Verfügung: Eine solche Adresse gehört zum
GitHub-Konto gleichen Namens, und das Konto `Buchfink` ist vergeben.
Was bleibt, ist die Projektadresse `mxzinke.github.io/buchfink/` — oder eine
eigene Domain, die sich unter **Settings → Pages → Custom domain** eintragen
lässt und dann als `CNAME`-Datei in diesem Verzeichnis landet.

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
- **Schatten an vier Stellen.** Das Konzept erlaubt sie an schwebenden
  Elementen (§6.2). Schwebend sind hier: das aufgeklappte Menü in der
  Kopfleiste, der Dialog, das Bild im Kopf der Startseite und ein Screenshot,
  solange der Zeiger darauf steht. Ein ruhender Screenshot hat nur seine
  Haarlinie.
- **Seitenbreite.** `--wide` liegt bei 1200 px, `--page` bei 1040 px. Vorher
  waren es 1400 und 1120; auf einem 16-Zoll-Bildschirm lief die Seite damit
  fast von Kante zu Kante, und ein Funktionsblock zerfiel in zwei weit
  auseinanderliegende Hälften.
- **Leiserer Hinweisstreifen.** In der Anwendung ist ein Hinweis pastellig
  ausgefüllt, weil er den Lesefluss unterbrechen soll. Auf einer Seite, die man
  einmal von oben nach unten liest, zieht dieselbe Fläche den Blick vom Inhalt
  weg. Deshalb hat hier nur die senkrechte Leiste die Farbe der Familie; die
  Bedeutung bleibt ablesbar, ohne dass der Abschnitt leuchtet.

Zwei Regeln des Konzepts sind für die Seite besonders wichtig und werden streng
eingehalten:

- **Alle Funktionsblöcke sind gleich ausgerichtet** — Text links, Bild rechts.
  Ein Wechsel der Seite von Block zu Block sieht nach Abwechslung aus, kostet
  aber bei jedem Block einen neuen Lesestart.
- **Diagramme liegen ohne Rahmen direkt auf dem Papier.** Eine Fläche darum wäre
  eine Karte (§6.2), und die Kästen im Bild bringen ihre Abgrenzung schon mit.

## Text

Die Seite schreibt für jemanden, der eine GmbH gegründet hat und keine
Buchhaltung gelernt hat. Was sie sagt, richtet sich nach dessen Fragen, nicht
nach dem Aufbau des Programms: Was mache ich damit jeden Tag? Welche Frist
nimmt es mir ab? Passt es zu meiner Rechtsform? Wo liegen meine Zahlen? Die
Startseite geht diese Fragen der Reihe nach durch.

Die Startseite erklärt Selbstständigkeit, Unabhängigkeit und Mitwirkung.
Einfache Jahresabschlüsse und eine lokale MCP-Anbindung sind als Ziele
gekennzeichnet, solange sie nicht vollständig implementiert sind. Die Roadmap
enthält keine kopierte Kriterienstatistik; diese wird im Anforderungskatalog
gepflegt.

Funktionsseiten erklären zuerst, was Anwender erledigen können. Details und
Rechtsgrundlagen stehen in Dialogen hinter „Mehr erfahren“. Normverweise werden
als Links zum Gesetzestext geschrieben. Technische Einschränkungen, die über
die Nutzbarkeit entscheiden, etwa eine ungeprüfte E-Bilanz, bleiben sichtbar.
Die gemeinsamen Regeln stehen in [Schreibweise](../docs/schreibweise.md).

## Vertiefungen im Dialog

Was über den Hauptgedanken hinausgeht, steht in einem `<dialog class="modal">`
am Ende von `<main>`. Ausgelöst wird er von einer Schaltfläche
`<button class="reveal" data-dialog="…">`, die neben dem Absatz steht, zu dem
er gehört. Vorher war das ein aufklappendes `<details>`; das schob beim Öffnen
den halben Abschnitt nach unten, und in einer haftenden Textspalte sprang das
Bild daneben mit.

`assets/site.js` verbindet beides und schließt den Dialog bei Escape und beim
Klick daneben. Ohne JavaScript öffnet kein Dialog, deshalb steht in jeder Seite
ein `<noscript>`-Block, der die Dialoge stattdessen offen an das Ende der Seite
stellt und die Schaltflächen ausblendet.

Ein neuer Dialog braucht drei Dinge: eine eindeutige `id`, dieselbe `id` im
`data-dialog` der Schaltfläche und ein `aria-labelledby` auf seinen Titel.

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

Auf der Seite sitzt über jedem Bild eine Leiste mit drei Punkten
(`.shot__chrome`). Die Aufnahmen zeigen ein Programmfenster ohne seinen Rahmen;
ohne die Leiste schwimmt der Bildinhalt im Blatt. Sie ist Markup und steht
deshalb in jeder `<figure class="shot">` — wer ein Bild einfügt, fügt sie mit
ein.

Zwanzig Bilder entstehen dabei, alle 2880 × 1800 (1440 × 900 bei doppelter
Pixeldichte). Welche Seite welches Bild zeigt, steht in der Liste `shots` in
`shoot.mjs`; jedes Bild wird von mindestens einer Seite verwendet. Ein Bild,
das keine Seite mehr zeigt, gehört gelöscht.

Alle Firmen, Namen, Beträge und Belege darin sind erfunden. Ändert sich eine
Ansicht in der Anwendung, genügt ein erneuter Lauf; ändert sich, welche Daten
eine Ansicht braucht, muss `mock-bridge.ts` mitgezogen werden.
