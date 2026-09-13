# Visuelle Grundlagen

[Designkonzept](README.md) · [Dokumentation](../README.md)

## 2. Leitidee: Stilles Kontor

Buchfink ist Werkzeug, keine Bühne. Wer damit arbeitet, sitzt vier Stunden am
Stück vor Journal und Kontenabgleich und sucht Abweichungen. Die Oberfläche hat
dabei eine Aufgabe: nicht im Weg zu stehen.

**1. Ruhe ist Funktion, nicht Geschmack.**
Jedes Element, das Aufmerksamkeit zieht, ohne sie zu verdienen, verlängert die
Suche nach dem, was zählt. Keine Farbfläche ohne Bedeutung, kein Schatten ohne
Ebenenwechsel, keine Animation ohne Zustandswechsel, kein Erklärsatz, den man ab
dem zweiten Mal überliest. Erklärungen gehören hinter ein Zeichen, das man
anklickt, wenn man sie braucht ([Abschnitt 15](hilfetexte.md#15-text)).

**2. Das Blatt, nicht der Kasten.**
Struktur entsteht aus Weißraum, Haarlinien und Ausrichtung. Eine Karte sagt:
Das hier ist ein eigenes Objekt. In einer Buchhaltung ist fast nichts ein eigenes
Objekt. Journal, Kontenblatt und Auswertung sind ein fortlaufendes Blatt. Wer
jeden Abschnitt einrahmt, zieht Wände in ein Dokument. Flächen sind die Ausnahme
und in [Abschnitt 6](#6-fläche-form-höhe) abschließend aufgezählt.

**3. Farbe ist Information.**
Farbe ist reserviert. Salbeigrün heißt geprüft, Rosé heißt Storno, Bernstein
heißt offen, Himmelblau ist die Marke. Alles andere ist Papier und Tinte. Eine
dekorativ eingefärbte Fläche verbraucht Bedeutung, die später fehlt.

**4. Die Zahl ist der Held.**
Beträge, Salden und Belegnummern sind der Inhalt. Sie stehen rechtsbündig, in
gleicher Ziffernbreite, mit Luft, und ohne dass Rahmen oder Icons mit ihnen
konkurrieren.

**5. Nichts verschwindet.**
Die GoBD verlangt sichtbare Korrekturen. Das ist ein Gestaltungsprinzip, keine
Last. Stornierte Buchungen werden markiert, nicht versteckt, und
nie durchgestrichen.

---

## 3. Farbe

### 3.1 Papier und Tinte

Warme Neutraltöne statt Grau. Grau wirkt technisch, warmes Papier passt zu
Belegen und ermüdet bei langer Nutzung weniger.

| Token | Wert | Einsatz |
|---|---|---|
| `paper` | `#FAF8F5` | Standardgrund: Seiten, Tabellen, Abschnitte |
| `surface` | `#FFFFFF` | Eingabefelder, Overlays, Belegvorschau |
| `sunken` | `#F4F1EB` | Zeilen-Hover, gesperrte Perioden |
| `line` | `#E9E4DC` | Haarlinie, Standardtrennung |
| `line-strong` | `#D8D1C6` | Tabellenkopf, Summenlinie |
| `control-border` | `#948E85` | Rahmen bedienbarer Elemente, 3,1:1 auf Papier |
| `ink` | `#1C1917` | Primärtext, Primärbutton, 16,5:1 |
| `ink-muted` | `#57514A` | Sekundärtext, 7,4:1 |
| `ink-subtle` | `#756E65` | Labels, Metadaten, Tabellenkopf, 4,8:1 |
| `ink-faint` | `#A79F94` | Deaktiviert, dekorative Icons, **nie Fließtext** |

`control-border` ist bewusst dunkler als die Haarlinien. Eine Haarlinie erreicht
die von WCAG 1.4.11 geforderten 3:1 nicht, und ein Eingabefeld muss als bedienbar
erkennbar sein. Struktur darf hell bleiben, Bedienelemente nicht.

Die dunkle Navigationsspalte hat eine eigene Skala, damit sie als eigener Raum
lesbar bleibt: `shell` `#24211E`, `shell-raised` `#2E2A26`, `shell-line`
`#37322D`, `shell-text` `#D8D2CA`, `shell-text-muted` `#968E84`.

Für die Statuszeile im Fuß der Navigation kommen zwei aufgehellte Signalfarben
dazu, weil die Werte aus [Abschnitt 3.3](#33-die-vier-familien) für helles Papier gerechnet sind und auf der
dunklen Fläche den Kontrast verlieren: `shell-positive` `#9BCFA6` (9,9:1) und
`shell-negative` `#E8A79C` (8,7:1).

### 3.2 Vier Familien, vier Rollen

Jede Farbfamilie hat vier Tokens mit fester Aufgabe. Das ist der Preis für
Pastell: Ein pastelliger Ton ist mit Text nicht lesbar, ein textfähiger Ton ist
nicht pastellig. Die Trennung macht beides möglich.

| Endung | Rolle | Kontrastziel |
|---|---|---|
| `-soft` | Fläche (Badge, Hinweis, Zeilentönung) | keins, reine Flächenfarbe |
| `-line` | Rand dieser Fläche | sichtbar gegen Papier, rund 1,8:1 |
| (Basis) | Marker: Punkt, Leiste, Fokusring, Diagrammlinie | 3:1 gegen Papier |
| `-text` | Text und Icons | 4,5:1 gegen Papier und gegen `-soft` |

### 3.3 Die vier Familien

**Himmelblau** ist die Marke. Es steht für Vertrauen und Weite, und es ist die
einzige Farbe, die etwas über Buchfink aussagt statt über einen Datensatz.

| Token | Wert | Kontrast |
|---|---|---|
| `accent` | `#4090C0` | 3,3:1 Marker |
| `accent-text` | `#1D6A96` | 5,6:1 Text |
| `accent-soft` | `#E8F2F9` | Fläche |
| `accent-line` | `#93C1DF` | 1,8:1 Rand |
| `accent-light` | `#8FC4E4` | 7,6:1 auf `shell-raised` |

Einsatz: Wortmarke, aktiver Navigationseintrag, Fokusring, Links, Auswahl,
Diagrammlinien.

**Bernstein** heißt offen und zu erledigen. Aus dem Logo abgeleitet.

| Token | Wert | Kontrast |
|---|---|---|
| `attention` | `#B37F33` | 3,3:1 |
| `attention-text` | `#8A5A15` | 5,6:1 |
| `attention-soft` | `#FBF0DE` | Fläche |
| `attention-line` | `#D6B575` | 1,9:1 |

**Salbei** heißt geprüft, abgestimmt, festgeschrieben, im Plus. Aus dem Logo
abgeleitet.

| Token | Wert | Kontrast |
|---|---|---|
| `positive` | `#5A9A6B` | 3,2:1 |
| `positive-text` | `#2E6B3C` | 6,0:1 |
| `positive-soft` | `#E6F1E8` | Fläche |
| `positive-line` | `#93BF9F` | 1,9:1 |

**Rosé** heißt Storno, Fehler, Integritätsbruch, im Minus.

| Token | Wert | Kontrast |
|---|---|---|
| `negative` | `#C4736A` | 3,3:1 |
| `negative-text` | `#A0453A` | 5,8:1 |
| `negative-soft` | `#FAEBE8` | Fläche |
| `negative-line` | `#DB9E93` | 2,1:1 |

### 3.4 Regeln

1. **Primäraktionen sind Tinte, nicht Himmelblau.** Pro Ansicht gibt es eine
   wichtigste Aktion. Sie muss sich abheben, ohne zu leuchten. Ein blauer
   Primärbutton in jeder Ecke macht die Marke zur Tapete, und ein Himmelblau,
   auf dem Weiß lesbar ist, ist kein Himmelblau mehr.
2. **Blau ist die Marke, kein Zustand.** Es gibt keinen blauen Info-Hinweis und
   keinen blauen Status. Hinweise stehen in Papier und Tinte.
3. **Soll und Haben werden nie eingefärbt.** Farbe suggeriert hier eine Wertung,
   die es fachlich nicht gibt. Gefärbt wird das Ergebnis (Saldo, Gewinn, Verlust)
   und der Zustand (offen, storniert).
4. **Farbe steht nie allein.** Jeder farbige Zustand hat zusätzlich Text oder
   ein Icon. Für Rot-Grün-Sehschwäche, und weil Auswertungen gedruckt werden.
5. **Fläche und Rand gehören zusammen.** `bg-positive-soft border
   border-positive-line`. Eine randlose Pastellfläche verschwindet auf Papier,
   sie liegt nur bei 1,1:1.

### 3.5 Marke und Logo

Das Logo zeigt einen Buchfinken in Bernstein und Grün. Die Marke der Oberfläche
ist Himmelblau. Das ist kein Widerspruch, solange das Logo eine Illustration
bleibt und keine Farbfläche. Wenn das Logo später eine blaue Fassung bekommen
soll, ändert das an diesem Konzept nichts. Die Signalfarben Bernstein und Salbei
stammen weiter aus dem Logo, ihre Bedeutung ist davon unabhängig.

---

## 4. Typografie

Manrope ist die Schrift der Oberfläche, in sechs Stufen. Was nicht in diese Skala passt,
ist ein Layoutproblem.

| Token | Größe / Zeilenhöhe | Gewicht | Einsatz |
|---|---|---|---|
| `text-display` | 22 / 28 px | 600 | Seitentitel, einer pro Ansicht |
| `text-heading` | 16 / 22 px | 600 | Abschnitts- und Dialogtitel |
| `text-body` | 13 / 20 px | 400 | Fließtext, Tabellenzellen, Formularwerte |
| `text-label` | 12 / 16 px | 500 | Feldbeschriftungen, Tabellenkopf, Buttons |
| `text-caption` | 11 / 15 px | 400 | Hilfstexte, Metadaten, Zeitstempel |
| `text-overline` | 10 / 14 px | 600, 0,08 em, Versalien | Gruppenlabels in der Navigation |

Bisher war `text-xs` (12 px) der Standard für fast alles. 13 px im Fließtext ist
bei stundenlanger Arbeit spürbar angenehmer, ohne dass die Dichte leidet. 9 px
entfällt ersatzlos.

Gewichte: 400 für Text, 500 für Labels, 600 für Überschriften und Summen. 700 und
800 kommen nicht vor, Manrope wird in diesen Schnitten laut.

### 4.1 Zahlensatz

Manrope bringt tabellarische Ziffern (`tnum`) und eine durchgestrichene Null
(`zero`) mit. Ein zweiter Schriftschnitt ist deshalb überflüssig, und Beträge
bleiben Teil des Textbildes.

| Utility | Wirkung | Einsatz |
|---|---|---|
| `.num` | tabellarische Ziffern | alle Beträge, Salden, Prozentsätze, Datumsangaben, Mengen |
| `.code-num` | zusätzlich durchgestrichene Null | Beleg- und Kontonummern, IBAN, Steuernummern |
| `font-mono` | Systemschrift | nur Hashes, Dateipfade, XML-Fragmente |

Ohne `.num` springen Zahlen beim Aktualisieren und Spalten stehen nicht
untereinander. Eine Betragsspalte ohne `.num` ist ein Fehler.

Formatierung bleibt de-DE und liegt in `utils/formatters.ts`: `1.234,56 €`,
`01.01.2024`, echtes Minuszeichen `−` (U+2212), `—` für keinen Wert.

---

## 5. Raster und Dichte

Alles ist ein Vielfaches von 4 px. Erlaubt: 4, 8, 12, 16, 24, 32, 48, 64.

| Maß | Wert |
|---|---|
| Seitenrand Desktop / Mobil | 32 px / 16 px |
| Maximale Inhaltsbreite | 1200 px, zentriert, Tabellen dürfen auf volle Breite |
| Abstand zwischen Abschnitten | 32 px vor der Trennlinie, 24 px danach |
| Innenabstand der Flächen aus [Abschnitt 6.2](#62-wann-eine-fläche-erlaubt-ist) | 20 px |
| Label zu Feld | 4 px |
| Zwischen Feldern | 16 px |

Da Abschnitte keine Rahmen mehr haben, übernimmt der Abstand die Gliederung allein.
Zu knapper Weißraum fällt sofort auf. Im Zweifel die nächstgrößere Stufe.

Dichte, umschaltbar in den Einstellungen und pro Mandant gespeichert:

| Stufe | Zeilenhöhe | Einsatz |
|---|---|---|
| Kompakt | 32 px | Journal, Kontenabgleich, SuSa |
| Komfortabel | 40 px | Standard, Stammdaten, alles mit Bearbeitungsaktionen |

Berührungsziele bleiben mindestens 32 mal 32 px, auf Touch-Geräten 44 mal 44 px.
Eine Ausnahme: das Erklärzeichen aus [Abschnitt 15.2](hilfetexte.md#152-tooltip-und-detaildialog) hat auf dem Desktop 24 mal 24 px. Es
sitzt inline neben einer Beschriftung, wo 32 px die Zeile auseinanderziehen
würden, und auf Touch-Geräten gilt auch dort 44 px.

---

## 6. Fläche, Form, Höhe

Der Abschnitt, der den Gesamteindruck entscheidet.

### 6.1 Die Seite ist die Fläche

Inhalt liegt direkt auf dem Papier. Weiß ist ein Signal, kein Hintergrund:
Dort passiert etwas.

| Fläche | Wo sie gilt |
|---|---|
| `paper` | Standard: Seiten, Abschnitte, Formulare, Kennzahlen, Filterleisten |
| `surface` | Eingabefelder, Overlays, Datentabellen, Belegvorschau |
| `sunken` | Zeilen-Hover, gesperrte Perioden |

Ein weißes Eingabefeld auf Papiergrund ist eine stärkere und flachere
Bedienbarkeits-Aussage als jede Karte darum herum.

### 6.2 Wann eine Fläche erlaubt ist

Abschließende Liste. Was nicht hier steht, bekommt keinen Rahmen und keinen
eigenen Hintergrund.

1. **Overlays.** Dialog, Popover, Dropdown, Tooltip. Die schweben wirklich.
2. **Eine Datentabelle.** Journal, Kontenliste, Offene Posten, SuSa. Der Inhalt
   ist vom Blatt gelöst: eigene Spalten, eigenes seitliches Scrollen, ein Kopf,
   der beim Scrollen stehen bleibt. Die Fläche macht diese Ablösung sichtbar und
   gibt der stehenden Kopfzeile einen Grund. Ein Formular oder eine Kennzahl hat
   nichts davon und bekommt deshalb auch keine Fläche.
3. **Ein Fremdkörper auf dem Blatt.** Die Belegvorschau, ein eingebettetes PDF.
   Das ist ein Dokument, keine Oberfläche.
4. **Ein Hinweis, der den Lesefluss unterbrechen soll.** Periodensperre,
   Integritätsbruch, Fehlermeldung aus dem Backend.
5. **Ein Leerzustand**, der die Stelle füllt, an der sonst eine Tabelle stünde.

### 6.3 Verschachtelung ist verboten

Eine Fläche in einer Fläche gibt es nicht, ohne Ausnahme. Wenn ein Abschnitt
innerhalb einer Fläche eine eigene Fläche zu brauchen scheint, ist der Abschnitt
zu groß und gehört in eine eigene Ansicht. Eingabefelder in einem Dialog sind
kein Verstoß, ein Feld ist Bedienelement.

### 6.4 Abschnitte trennen ohne Kasten

Das Ersatzmuster für die Karte, überall gleich:

```
<section class="pt-8 mt-8 border-t border-line">
  <h2 class="text-heading">Offene Posten</h2>
  <p class="text-caption text-ink-subtle mt-1">7 Rechnungen · 12.470,00 €</p>
  <div class="mt-5"> … Inhalt direkt auf dem Papier … </div>
</section>
```

Der erste Abschnitt einer Ansicht bekommt keine Linie, er steht schon unter dem
Seitenkopf. Reicht der Abstand zur Trennung, entfällt auch die Linie. Weißraum
ist das leiseste Trennmittel und deshalb das erste.

### 6.5 Radien

| Token | Wert | Einsatz |
|---|---|---|
| `rounded-control` | 6 px | Buttons, Felder, Chips, Badges, Hinweise |
| `rounded-card` | 10 px | Datentabellen, Belegvorschau |
| `rounded-overlay` | 14 px | Dialoge, Popover, Dropdowns |
| `rounded-full` | | Avatare, Zähler-Pills |

Statusmarker sind davon ausgenommen. Sie sind Rauten, siehe [Abschnitt 10](komponenten.md#10-komponenten).

**Eine einseitige Markierung ist keine Border.** Eine Border folgt dem
Eckenradius und läuft an den Enden krumm. Wo etwas an einer Kante markiert wird,
etwa der aktive Navigationseintrag oder eine Stornozeile, bleibt die Fläche
abgerundet und die Markierung ist eine eigene Pille: 2 px breit, 16 px hoch,
`rounded-full`, senkrecht zentriert. Im Fluss als Element, in Tabellenzellen als
Pseudo-Element.

### 6.6 Höhe

| Token | Einsatz |
|---|---|
| kein Schatten | alles im Blatt |
| `shadow-popover` | Dropdowns, Popover, Tooltips, Kontextmenüs |
| `shadow-dialog` | modale Dialoge |

`shadow-xs`, `shadow-sm` und `shadow-xl` verschwinden aus dem Code. Ein Schatten
bedeutet, dass ein Element über der Seite schwebt, nicht dass es wichtig ist.

---

## 7. Bewegung

Bewegung zeigt einen Zustandswechsel. In einer Anwendung, die acht Stunden offen
ist, wird jede Animation zur Wiederholung.

Was sich bewegen darf, abschließend:

| Auslöser | Was | Dauer |
|---|---|---|
| Hover, Fokus, Aktiv | Farbe, Rahmenfarbe | 120 ms |
| Overlay öffnet | Deckkraft 0 auf 1, 4 px Versatz nach oben | 180 ms |
| Overlay schließt | nur Deckkraft | 120 ms |
| Zeile aufklappen | Höhe | 180 ms |
| Toast erscheint | Deckkraft, 8 px von rechts | 180 ms |
| Integritätsprüfung läuft | Rotation, wiederholt | 900 ms, linear |

Alles andere bewegt sich nicht. Kein Übergang beim Seitenwechsel, keine Animation
beim Filtern einer Tabelle, kein gestaffeltes Einblenden von Listen, kein
Skalieren beim Hover, kein Federn.

`ease-quiet` ist `cubic-bezier(.2, .7, .3, 1)`. Schneller Start, weiches Ende,
kein Nachschwingen. Die Rotation des Ladeindikators ist die einzige lineare
Bewegung.

Zahlen animieren nie. Ein hochzählender Saldo ist in einer Buchhaltung eine
Zumutung, weil man ihn liest, während er noch falsch ist.

Bei `prefers-reduced-motion: reduce` fällt alles auf 0 ms, das ist im Basis-Layer
umgesetzt. Der Ladeindikator wird dann zu einem statischen Text.

---

## 16. Dunkelmodus

Vorbereitet, nicht Teil der ersten Umsetzung. Da alle Farben über Tokens laufen,
genügt ein zweiter Wertesatz unter `[data-theme="dark"]`. `paper` wird `#1B1917`,
`surface` wird `#24211E`, die Tinte-Skala kehrt sich um.

Die vier Familien brauchen aufgehellte Werte, weil die dunklen Töne auf dunklem
Grund den Kontrast verlieren:

| Familie | `-text` | Basis | `-soft` | `-line` |
|---|---|---|---|---|
| Himmelblau | `#8FC4E4` | `#5FA5CE` | `#1B2F3C` | `#2E4A5C` |
| Bernstein | `#E3B77E` | `#C89043` | `#322718` | `#4C3A1F` |
| Salbei | `#9BCFA6` | `#6FAE80` | `#1D2C21` | `#33513A` |
| Rosé | `#E8A79C` | `#C6837A` | `#32211E` | `#56342E` |

Bedingung für den Start: kein Hex-Literal mehr im Komponentencode.

---
