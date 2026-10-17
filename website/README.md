# Projektseite

Dieses Verzeichnis ist die Projektseite von Buchfink, so wie GitHub Pages sie
ausliefert. Es gibt keinen Build-Schritt.

```text
website/
├── index.html             # Was Buchfink ist, Arbeitsteilung, Freigabe, für wen, Daten
├── so-funktionierts.html  # Verbinden, Belege, Monat, Jahr, Erklärungen, Pflichten, Grenzen
├── installation.html      # App je Betriebssystem, Einrichten, KI-Agent verbinden, Hilfe
├── roadmap.html           # gebaut, bis Version 1, danach, nicht geplant
├── .nojekyll              # Pages soll nichts umbauen
└── assets/
    ├── site.css           # das komplette Stylesheet
    ├── site.js            # Menü schließen, Dialoge öffnen
    ├── buchfink-logo.svg
    ├── fonts/             # Manrope, selbst ausgeliefert (OFL, Lizenz liegt bei)
    └── screenshots/       # Bilder der Oberfläche, siehe unten
```

Alle vier Seiten haben dieselbe Kopfleiste: Start, So funktioniert’s,
Installation, Roadmap und rechts GitHub. Sie steht in jeder Datei als Markup;
wer einen Eintrag ändert, ändert ihn viermal. Der Eintrag der aktuellen Seite
trägt `aria-current="page"`. Im schmalen Fenster tritt ein Schubfach an die
Stelle der Reihe. Beide Menüs sind `<details>`-Elemente und funktionieren ohne
JavaScript.

Der Stylesheet-Link trägt eine Versionskennung `?v=…`: die ersten zwölf Zeichen
von `shasum -a 256 website/assets/site.css`. Nach jeder Änderung an
`site.css` ist sie in allen vier Seiten nachzuziehen.

## Veröffentlichen

Von Hand, ohne CI:

```bash
task pages:preview        # http://127.0.0.1:8000
task pages:publish        # git subtree push --prefix website origin gh-pages
```

`pages:publish` schiebt `website/` als Wurzel in den Zweig `gh-pages`. In den
Repository-Einstellungen steht Pages auf *Deploy from a branch*, Zweig
`gh-pages`, Ordner `/ (root)`. Pages liefert aus einem Zweig nur die Wurzel oder
`docs/` aus; deshalb der eigene Zweig. `.nojekyll` muss im Verzeichnis bleiben,
sonst baut Pages die Dateien mit Jekyll um. Weist der Push zurück, weil
`gh-pages` auseinandergelaufen ist:

```bash
git push origin $(git subtree split --prefix website HEAD):gh-pages --force
```

Die Seite liegt unter `mxzinke.github.io/buchfink/`. Eine eigene Domain lässt
sich unter **Settings → Pages → Custom domain** eintragen und landet dann als
`CNAME` in diesem Verzeichnis.

`pages:publish` bricht ab, solange ein Bild fehlt, auf das eine Seite
verweist.

## Screenshots

Die Seiten zeigen sieben Bilder, alle 2880 × 1800 Pixel:

| Datei | Seite | Inhalt |
|---|---|---|
| `uebersicht.png` | Start | Übersicht mit Aufgaben und Fristen |
| `zustimmung.png` | Start | Zustimmungsdialog für eine KI-App |
| `einstellungen-apps.png` | So funktioniert’s | verbundene Apps mit ihren Rechten |
| `buecher.png` | So funktioniert’s | Bücher mit Belegen, Bank und Buchungen |
| `abschluesse.png` | So funktioniert’s | Stand der Monate und des Jahres |
| `erklaerungen.png` | So funktioniert’s | Erklärung formularnah mit Herkunft je Wert |
| `unternehmen.png` | So funktioniert’s | Unternehmensdaten mit steuerlicher Beratung |

Sie entstehen aus der echten Oberfläche in `frontend/`; nur die Wails-Bridge ist
durch Beispieldaten ersetzt. Ablauf und Dateinamen beschreibt
[`scripts/site-screenshots/`](../scripts/site-screenshots/README.md):

```bash
npm --prefix frontend install
npm --prefix scripts/site-screenshots install
node scripts/site-screenshots/shoot.mjs
```

Jeder Dateiname aus der Tabelle muss dort erzeugt werden. Alle Firmen, Namen
und Beträge sind erfunden.

Über jedem Bild steht eine Leiste mit drei Punkten (`.shot__chrome`); sie ist
Markup und gehört in jede `<figure class="shot">`.

## Gestaltung und Text

Die Seite benutzt die Tokens der Anwendung aus den
[visuellen Grundlagen](../docs/gestaltung/grundlagen.md) als CSS-Variablen am
Anfang von `site.css`, mit drei Abweichungen für eine Leseseite: größere
Schriftgrade für Fließtext und Titel, Schatten auch am Bild im Kopf der
Startseite und an einem Bild unter dem Zeiger, und Hinweisstreifen nur mit
farbiger Leiste.

Die Texte folgen der [Schreibweise](../docs/gestaltung/schreibweise.md): Sie-Form,
kurze Sätze, keine Zusagen über Gesetzeskonformität. Die Seite erzählt aus der
Sicht eines Gründers, der mit seinem KI-Agenten arbeitet: was der Agent tut, was
nur er selbst freigibt, wo die Grenzen liegen. Grenzen bleiben sichtbar.
Einzelheiten stehen in der Dokumentation auf GitHub, auf die die Seiten
verlinken.

Was über den Hauptgedanken hinausgeht, kann in einem `<dialog class="modal">` am
Ende von `<main>` stehen, ausgelöst von `<button class="reveal"
data-dialog="…">`. Der Dialog braucht eine eindeutige `id` und ein
`aria-labelledby` auf seinen Titel. Ohne JavaScript öffnet kein Dialog; eine
Seite mit Dialog trägt deshalb im Kopf einen `<noscript>`-Block, der ihn offen
ans Seitenende stellt.
