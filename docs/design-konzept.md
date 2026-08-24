# Design-Konzept

Die visuelle und interaktive Grundlage von Buchfink. Dieses Dokument ist die
Referenz für Code-Reviews. Eine Ansicht, die gegen die Regeln hier verstößt, ist
ein Fehler und keine Geschmacksfrage.

Zwei Stellen setzen es um:

- [`frontend/src/index.css`](../frontend/src/index.css) hält die Tokens als
  Tailwind-`@theme`. Nutzbar als Utility (`bg-paper`, `text-ink`) und als
  CSS-Variable (`var(--color-ink)`).
- [`frontend/src/components/ui/`](../frontend/src/components/ui) hält die
  Bausteine. Wo dieses Dokument einen Baustein beschreibt, steht dort die
  verbindliche Umsetzung. Klassenketten gehören in die Komponente, nicht in
  dieses Dokument und nicht in eine Seite.

---

## 1. Ausgangslage

Die bestehende Oberfläche hat eine stimmige Grundstimmung. Warmes Papier,
Manrope, dunkle Navigationsspalte. Was fehlt, ist ein System dahinter. Eine
Auszählung des Frontends zeigt den Zustand:

| Dimension | Ist-Zustand | Folge |
|---|---|---|
| Container | 58 Karten-Flächen (`bg-white` plus Rahmen plus Radius) in 13 Dateien | Die Oberfläche zerfällt in Kacheln, statt als Blatt zu lesen |
| Eckenradien | 6 verschiedene, 118 mal `lg` neben 97 mal `xl` | Flächen wirken zufällig unterschiedlich |
| Schriftgrößen | 9 Stufen, darunter 9 px, 10 px, 11 px | Hierarchie ist nicht ablesbar, 9 px ist unlesbar |
| Farbe | rund 45 Farb-Utilities aus 4 Paletten | Bernstein bedeutet mal Marke, mal Aktion, mal Ergebnis |
| Zahlen | `font-mono` heißt Systemschrift | Auf jedem Betriebssystem anders, bricht mit Manrope |
| Fokus | überwiegend Browser-Standard oder unterdrückt | Tastaturbedienung ist nicht verlässlich |

Einzeln ist nichts davon falsch. In Summe kostet es die Ruhe, die eine
Buchhaltung braucht, und macht jede neue Ansicht zur Neuerfindung.

---

## 2. Leitidee: Stilles Kontor

Buchfink ist Werkzeug, keine Bühne. Wer damit arbeitet, sitzt vier Stunden am
Stück vor Journal und Kontenabgleich und sucht Abweichungen. Die Oberfläche hat
dabei eine Aufgabe: nicht im Weg zu stehen.

**1. Ruhe ist Funktion, nicht Geschmack.**
Jedes Element, das Aufmerksamkeit zieht, ohne sie zu verdienen, verlängert die
Suche nach dem, was zählt. Keine Farbfläche ohne Bedeutung, kein Schatten ohne
Ebenenwechsel, keine Animation ohne Zustandswechsel, kein Erklärsatz, den man ab
dem zweiten Mal überliest. Erklärungen gehören hinter ein Zeichen, das man
anklickt, wenn man sie braucht (§15).

**2. Das Blatt, nicht der Kasten.**
Struktur entsteht aus Weißraum, Haarlinien und Ausrichtung. Eine Karte sagt:
Das hier ist ein eigenes Objekt. In einer Buchhaltung ist fast nichts ein eigenes
Objekt. Journal, Kontenblatt und Auswertung sind ein fortlaufendes Blatt. Wer
jeden Abschnitt einrahmt, zieht Wände in ein Dokument. Flächen sind die Ausnahme
und in §6 abschließend aufgezählt.

**3. Farbe ist Information.**
Farbe ist reserviert. Salbeigrün heißt geprüft, Rosé heißt Storno, Bernstein
heißt offen, Himmelblau ist die Marke. Alles andere ist Papier und Tinte. Eine
dekorativ eingefärbte Fläche verbraucht Bedeutung, die später fehlt.

**4. Die Zahl ist der Held.**
Beträge, Salden und Belegnummern sind der Inhalt. Sie stehen rechtsbündig, in
gleicher Ziffernbreite, mit Luft, und ohne dass Rahmen oder Icons mit ihnen
konkurrieren.

**5. Nichts verschwindet.**
Die GoBD verlangt sichtbare Korrekturen. Das ist keine Last, sondern ein
Gestaltungsprinzip. Stornierte Buchungen werden markiert, nicht versteckt, und
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

### 3.2 Vier Familien, vier Rollen

Jede Farbfamilie hat vier Tokens mit fester Aufgabe. Das ist der Preis für
Pastell: Ein pastelliger Ton trägt keinen Text, ein textfähiger Ton ist nicht
pastellig. Die Trennung macht beides möglich.

| Endung | Rolle | Kontrastziel |
|---|---|---|
| `-soft` | Fläche (Badge, Hinweis, Zeilentönung) | keins, sie trägt nur |
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
   Primärbutton in jeder Ecke macht die Marke zur Tapete, und ein Himmelblau, das
   Weiß trägt, ist kein Himmelblau mehr.
2. **Blau ist die Marke, kein Zustand.** Es gibt keinen blauen Info-Hinweis und
   keinen blauen Status. Hinweise stehen in Papier und Tinte.
3. **Soll und Haben werden nie eingefärbt.** Farbe suggeriert hier eine Wertung,
   die es fachlich nicht gibt. Gefärbt wird das Ergebnis (Saldo, Gewinn, Verlust)
   und der Zustand (offen, storniert).
4. **Farbe steht nie allein.** Jeder farbige Zustand trägt zusätzlich Text oder
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

Manrope trägt die Oberfläche, in sechs Stufen. Was nicht in diese Skala passt,
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
| Innenabstand der Flächen aus §6.2 | 20 px |
| Label zu Feld | 4 px |
| Zwischen Feldern | 16 px |

Da Abschnitte keine Rahmen mehr haben, trägt der Abstand die Gliederung allein.
Zu knapper Weißraum fällt sofort auf. Im Zweifel die nächstgrößere Stufe.

Dichte, umschaltbar in den Einstellungen und pro Mandant gespeichert:

| Stufe | Zeilenhöhe | Einsatz |
|---|---|---|
| Kompakt | 32 px | Journal, Kontenabgleich, SuSa |
| Komfortabel | 40 px | Standard, Stammdaten, alles mit Bearbeitungsaktionen |

Berührungsziele bleiben mindestens 32 mal 32 px, auf Touch-Geräten 44 mal 44 px.
Eine Ausnahme: das Erklärzeichen aus §15.2 hat auf dem Desktop 24 mal 24 px. Es
sitzt inline neben einer Beschriftung, wo 32 px die Zeile auseinanderziehen
würden, und auf Touch-Geräten gilt auch dort 44 px.

---

## 6. Fläche, Form, Höhe

Der Abschnitt, der den Gesamteindruck entscheidet.

### 6.1 Die Seite ist die Fläche

Inhalt liegt direkt auf dem Papier. Weiß ist kein Hintergrund, sondern ein
Signal: Dort passiert etwas.

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

Statusmarker sind davon ausgenommen. Sie sind Rauten, siehe §10.

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

## 8. Interaktion

Die Regeln, die entscheiden, ob sich die Anwendung verlässlich anfühlt.

### 8.1 Speichern

Nichts wird gespeichert, ohne dass die Person es auslöst. Kein Autosave für
Buchungen, Rechnungen oder Stammdaten. Ein Dialog speichert beim Bestätigen, eine
Vollbildmaske über ihre Primäraktion.

Ausgenommen ist der Ansichtszustand: Filter, Sortierung, Spaltenbreiten und
Dichte werden sofort und lautlos gespeichert. Das sind keine Daten.

### 8.2 Rückgängig statt Rückfrage

Ein Bestätigungsdialog ist teuer. Er unterbricht, und wer ihn dreimal gesehen
hat, klickt ihn weg, ohne zu lesen. Deshalb gilt: Was rückgängig gemacht werden
kann, wird ohne Rückfrage ausgeführt und bekommt 8 Sekunden lang einen Toast mit
Rückgängig. Das betrifft Entwurf gelöscht, Zuordnung aufgehoben, Import
verworfen.

Ein Dialog erscheint nur, wenn der Schritt wirklich unumkehrbar ist:
Festschreiben, Periode abschließen, Mandant löschen. Er benennt die Folge, nicht
die Aktion. Nicht "Wirklich festschreiben?", sondern: "Festgeschriebene Buchungen
lassen sich nur noch per Storno korrigieren."

### 8.3 Validierung

Geprüft wird beim Verlassen des Feldes, nicht bei jedem Tastendruck. Beim
Verlassen werden Beträge, Datumsangaben und IBAN normalisiert, aus `1234,5` wird
`1.234,50`.

Ein Fehler bleibt am Feld stehen, bis er behoben ist. Der Absenden-Button bleibt
aktiv. Beim Klick springt der Fokus auf das erste fehlerhafte Feld, und die
Meldung sagt, was zu tun ist. Ein deaktivierter Absenden-Button versteckt den
Grund und ist deshalb die schlechtere Lösung.

Die Buchungsgleichheit ist ein Sonderfall: Soll und Haben werden live verrechnet
und die Differenz wird laufend angezeigt. Das ist kein Fehler, sondern eine
Rechenhilfe, solange die Buchung nicht abgeschickt ist.

### 8.4 Warten

| Dauer | Anzeige |
|---|---|
| unter 200 ms | nichts |
| 200 ms bis 2 s | Skelettzeilen in der Form des erwarteten Inhalts |
| über 2 s | Skelett plus ein Satz, was gerade passiert |
| über 10 s (Import, XBRL-Export) | Fortschritt mit Anzahl, abbrechbar |

Ein ausgelöster Button bleibt an seiner Stelle, behält seine Breite und zeigt
einen Spinner. Er verschwindet nicht und wechselt nicht die Beschriftung.

### 8.5 Rückmeldung

Ein Toast erscheint für abgeschlossene Aktionen, deren Ergebnis man nicht ohnehin
sieht. Unten rechts, 4 Sekunden.

Keinen Toast gibt es für: Speichern in einem Dialog, der sich schließende Dialog
ist die Rückmeldung. Für Filter und Navigation. Für alles, dessen Wirkung direkt
im Bild steht.

Dauerhafte Zustände wie Integrität und Periodensperre stehen in der Oberfläche,
nie in einem Toast.

### 8.6 Tastatur

Die häufigste Tätigkeit muss ohne Maus funktionieren. Eine Buchung erfassen:
Datum, Tab, Betrag, Tab, Konto (die Suche öffnet beim Tippen, Enter übernimmt den
Treffer), Tab, Buchungstext, Enter bucht.

| Kürzel | Wirkung |
|---|---|
| `⌘K` | Suche über Belege, Konten, Kontakte |
| `⌘N` | neuer Datensatz im aktuellen Kontext |
| `⌘Enter` | Formular abschicken |
| `Esc` | Dialog schließen, Suche verlassen |
| `↑` `↓` | Zeile wechseln, in Betragsfeldern 1,00 € (mit Shift 100,00 €) |
| `Enter` | markierte Zeile öffnen |
| `Leertaste` | Zeile auswählen |

Jedes Kürzel steht im Tooltip der zugehörigen Schaltfläche. Ein Kürzel, das man
nur aus der Dokumentation kennt, existiert nicht.

### 8.7 Zustand bewahren

Filter, Sortierung, Scrollposition und Auswahl überleben es, wenn man eine Zeile
öffnet und zurückkehrt. Ein Wechsel des Geschäftsjahres setzt Filter zurück und
sagt das.

Eine begonnene Eingabe geht nicht verloren. Wer einen Dialog mit Inhalt schließt,
wird gefragt. Wer die Anwendung schließt, findet den Entwurf beim nächsten Start.

### 8.8 Fokus

Beim Öffnen eines Dialogs liegt der Fokus auf dem ersten Eingabefeld, nie auf dem
Bestätigen-Button. Beim Schließen kehrt er zum auslösenden Element zurück. Nach
dem Löschen einer Zeile springt er auf die nächste, nicht an den Seitenanfang.

---

## 9. Ikonografie

Lucide, ausschließlich, mit `stroke-width={1.5}`. Der Standardwert 2 ist für
kleine Größen zu fett.

| Größe | Einsatz |
|---|---|
| 14 px | Tabellenzeilen, Badges, kleine Buttons |
| 16 px | Navigation, Buttons, Feldsymbole |
| 20 px | Dialogtitel, Leerzustände |
| 24 px | Leerzustände mit getönter Fläche |

Icons begleiten Text, sie ersetzen ihn nicht. Eine Aktion, die nur als Icon
existiert, braucht `title` und `aria-label`. Farbige Icons folgen der Farbregel.

---

## 10. Komponenten

Die Bausteine liegen in [`frontend/src/components/ui/`](../frontend/src/components/ui).
Dort steht die verbindliche Umsetzung, hier stehen nur die Regeln, die man dem
Code nicht ansieht. Klassenketten werden nicht in Seiten geschrieben und nicht in
diesem Dokument gepflegt.

### 10.1 Verhalten kommt von Base UI

Alles, was Fokus fängt, positioniert oder auf Tasten hört, kommt aus
[`@base-ui/react`](https://base-ui.com). Selbst gebaut waren diese Teile
fehleranfällig, und die Fehler zeigen sich erst spät: ein Tooltip, der am
Bildschirmrand hinausläuft, ein Dialog, aus dem die Tabulatortaste
herausspringt, ein Menü ohne Typeahead.

Base UI liefert nur Verhalten und keine Gestalt. Die Gestalt kommt von hier, in
Tailwind-Klassen aus den Tokens in §3. Zustände hängen an den Datenattributen
der Bibliothek (`data-[open]`, `data-[highlighted]`, `data-[invalid]`), das
bleibt also Tailwind ohne Zwischenschicht.

Rein darstellende Bausteine bauen wir selbst, weil es dort nichts falsch zu
machen gibt.

### 10.2 Was es gibt

| Baustein | Datei | Verhalten von |
|---|---|---|
| `Button` | `Button.tsx` | eigen |
| `Input`, `Textarea` | `Input.tsx` | Base UI (Field-Anbindung) |
| `AmountInput` | `Input.tsx` | Base UI NumberField |
| `Select` | `Select.tsx` | Base UI Select |
| `Combobox` | `Combobox.tsx` | Base UI Combobox |
| `Checkbox`, `RadioGroup`, `Switch` | `Toggle.tsx` | Base UI |
| `Field`, `FieldRow` | `Field.tsx` | Base UI Field |
| `Dialog`, `ConfirmDialog` | `Dialog.tsx` | Base UI Dialog, AlertDialog |
| `Menu` und Einträge | `Menu.tsx` | Base UI Menu |
| `HelpTooltip`, `HelpPopover`, `InfoPopover` | `Help.tsx` | Base UI Tooltip, Popover |
| `Tabs`, `TabPanel`, `Separator` | `Tabs.tsx` | Base UI |
| `Progress`, `Skeleton`, `SkeletonRows`, `toast` | `Feedback.tsx` | Base UI Progress, Sonner |
| `FileDrop` | `FileDrop.tsx` | eigen |
| `Section`, `PageHeader` | `Section.tsx` | eigen |
| `StatRow`, `Stat` | `StatRow.tsx` | eigen |
| `Table` und Zellen | `Table.tsx` | eigen |
| `StatusBadge` | `StatusBadge.tsx` | eigen |
| `EmptyState` | `EmptyState.tsx` | eigen |
| `cn` | `cn.ts` | Klassen zusammenführen, letzte Angabe gewinnt |

Bewusst gibt es keine `Card`. Wo eine Fläche nötig ist (§6.2), steht sie an genau
dieser Stelle im Code und nicht als Baustein, der sich unbemerkt vermehrt.

### 10.3 Abdeckung

Abgeglichen mit dem, was die zwölf Seiten heute benutzen:

| Bedarf | Heute im Code | Baustein |
|---|---|---|
| Textfeld | 20 mal | `Input` |
| Auswahlliste | 19 mal | `Select` |
| Datum | 12 mal | `Input type="date"` |
| Dialog | 8 mal | `Dialog`, `ConfirmDialog` |
| Aufklappmenü | 9 mal | `Menu` |
| Tabelle | 8 mal | `Table` |
| Rückmeldung | 26 mal | `toast` |
| Kästchen, Radio | 5 mal | `Checkbox`, `RadioGroup` |
| Zahl, Betrag | 2 mal | `AmountInput` |
| Mehrzeilig | 1 mal | `Textarea` |
| Datei | 1 mal | `FileDrop` |
| Kontosuche | fehlte | `Combobox` |
| Schalter | fehlte | `Switch` |
| Reiter | fehlte | `Tabs` |
| Ladezustand | fehlte | `SkeletonRows` |
| Fortschritt | fehlte | `Progress` |

Ein echter Datumswähler mit Kalender fehlt noch. Bis dahin bleibt es beim
nativen Feld, das im Desktop-WebView brauchbar ist.

### 10.4 Regeln, die der Code nicht zeigt

**Buttons.** Genau eine Primäraktion pro Ansicht, in Tinte. Ein deaktivierter
Button braucht eine Erklärung im `title`, sonst versteckt er seinen Grund. Beim
Laden bleibt er an seiner Stelle, behält die Breite und tauscht die Beschriftung
nicht aus.

**Felder.** Pflichtfelder tragen kein Sternchen. Gekennzeichnet wird das
Seltenere: optional. Der Fehlertext ersetzt den Hinweis, solange er steht.

**Auswahl gegen Suche.** `Select` für kurze feste Listen, `Combobox` für alles,
was man suchen muss. Der SKR04-Kontenrahmen ist immer eine Suche.

**Bestätigung.** `ConfirmDialog` nur für unumkehrbare Schritte (§8.2). Er lässt
sich nicht durch einen Klick daneben schließen. Was rückgängig gemacht werden
kann, läuft ohne Rückfrage und bekommt `toast.undo`.

**Tabelle.** Die Fläche ersetzt nicht die Kopflinie: Der Kopf bleibt ohne
Füllung. Die Überschrift des Abschnitts steht über der Fläche, nicht darin,
sonst entsteht wieder eine Karte mit Kopfzeile. Keine Zebrastreifen.

**Status-Badge.** Enthält immer Text, nie nur den Marker. Der Marker ist eine
Raute, 7 mal 7 px, in der Basisfarbe der Familie. Der Kreis ist die weichste
Form, die es gibt, und in einer Statusspalte beliebig; die Raute hat vier
definierte Kanten und eine Achse.

Sie entsteht über `clip-path`, nicht über `rotate(45deg)`. Eine gedrehte Fläche
behält ihre ursprüngliche Layoutbox: Bei 5 mal 5 px malt sie 7,07 px, die
Spitzen ragen über die Box hinaus und verkürzen den Abstand zum Text auf 5 px,
obwohl 6 px gesetzt sind. Die Geometrie liegt als Utility `mark-diamond` in
`index.css`. Dieselbe Raute markiert den Integritätszustand im Fuß der
Navigation (§11.4), andere Statuspunkte gibt es nicht.

**Meldungen.**

| Art | Umsetzung |
|---|---|
| Erfolg einer Aktion | `toast.success`, 4 s |
| Umkehrbarer Schritt | `toast.undo`, 8 s |
| Fachlicher Fehler im Formular | am Feld, nie als Toast |
| Fehler aus dem Backend | Hinweisfläche in Rosé über den Aktionen |
| Dauerhafter Zustand | Statusleiste, kein Toast |

---

## 11. Fachliche Muster

Der Teil, der Buchfink von einer beliebigen Anwendung unterscheidet.

### 11.1 Soll und Haben

Zwei getrennte, rechtsbündige Spalten mit `.num`, nicht eine Spalte mit
Vorzeichen. Soll links, Haben rechts, wie im Buch. Beide farblich neutral. Unter
der letzten Position steht die Summe mit der buchhalterischen Doppellinie
(`rule-total`). Soll- und Habensumme müssen sichtbar gleich sein, das ist die
Kontrolle, die Buchhalter zuerst suchen.

Kontonummern in `.code-num`, dahinter die Bezeichnung in `text-ink-muted`. Die
Nummer allein hilft niemandem, der SKR04 nicht auswendig kann.

### 11.2 Storno und Generalumkehr

Stornierte Buchungen werden nie durchgestrichen und nie ausgeblendet. Der
ursprüngliche Betrag muss lesbar bleiben.

```
Zeile:  border-l-2 border-negative bg-negative-soft/50
Badge:  Storniert, auf der Gegenbuchung: Storno zu RE-2024-014
Betrag: normale Darstellung, kein line-through
```

Stornobuchung und Original verlinken sich gegenseitig, in beide Richtungen.

### 11.3 Status-Vokabular

Ein Wort pro Zustand, in der ganzen Anwendung dasselbe. Die Liste ist
abschließend.

| Status | Familie | Gilt für |
|---|---|---|
| Entwurf | neutral | Rechnung, Buchung vor dem Festschreiben |
| Offen | Bernstein | Rechnung, Offener Posten, Beleg ohne Buchung |
| Teilweise ausgeglichen | Bernstein, nur Rand | Offener Posten |
| Zugeordnet | neutral | Banktransaktion mit Beleg |
| Gebucht | Salbei | Buchung im Journal |
| Ausgeglichen | Salbei | Offener Posten |
| Festgeschrieben | Salbei mit Schloss-Icon | Buchung, Periode |
| Überfällig | Rosé | Rechnung, Steuerfrist |
| Storniert | Rosé | Buchung, Rechnung |
| Fehlerhaft | Rosé | Import, Validierung, Integritätsprüfung |

Synonyme wie erledigt, fertig oder abgeschlossen für denselben Zustand sind nicht
erlaubt.

### 11.4 Integrität der Hash-Chain

Der Zustand steht dauerhaft im Fuß der Navigation, nie als Toast.

| Zustand | Darstellung |
|---|---|
| geprüft | Raute in Salbei, "Daten unverändert", darunter der Zeitpunkt |
| wird geprüft | rotierendes Icon in `ink-subtle`, "Prüfung läuft" |
| gebrochen | Schild in Rosé, "Integrität verletzt", Klick öffnet das Protokoll bei der ersten abweichenden Buchung |

Ein Integritätsbruch ist der einzige Fall, in dem die Oberfläche laut werden darf:
ein Balken in `negative-soft` über dem gesamten Inhalt, bis er quittiert ist.

### 11.5 Gesperrte Perioden

Abgeschlossene Geschäftsjahre sind schreibgeschützt. Sichtbar durch `sunken`
statt `paper` als Grund, ein Schloss neben der Jahreszahl in der Kopfzeile und
einen Hinweisstreifen: "Geschäftsjahr 2024 ist abgeschlossen. Buchungen sind nur
im laufenden Jahr möglich."

Bearbeitungsaktionen werden deaktiviert und behalten ihre Position, damit die
Ansicht zwischen den Jahren gleich aussieht.

### 11.6 Belege

Zwei Spalten: links das Dokument, rechts die Erfassung. Die Belegvorschau ist der
eine Ort, an dem eine Fläche richtig ist (§6.2, Fall 2): `rounded-card border
border-line bg-surface`. Die Erfassung daneben bekommt keine.

Die Vorschau bleibt sichtbar, während gebucht wird. Der Abgleich zwischen Beleg
und Feld ist die häufigste Tätigkeit und darf keinen Fensterwechsel kosten.

Werte aus der E-Rechnung erscheinen als Vorschlag im Feld, mit dem Hinweis "aus
ZUGFeRD übernommen". Sie sind editierbar und werden nicht als gesichert
dargestellt.

---

## 12. Layout

```
┌──────────────┬─────────────────────────────────────────────┐
│              │  Kopfzeile 56 px, Geschäftsjahr, Mandant    │
│  Navigation  ├─────────────────────────────────────────────┤
│  240 px      │  Seitentitel + Primäraktion                 │
│  dunkel      │  ─────────────────────────────────────────  │
│              │  Kennzahlen und Filter auf dem Papier,      │
│  ──────────  │  die Tabelle in ihrer eigenen Fläche        │
│  Integrität  │                                             │
└──────────────┴─────────────────────────────────────────────┘
```

**Navigation.** 240 px, fünf Gruppen: Übersicht, Buchhaltung, Stammdaten,
Auswertungen, Verwaltung. Aktiver Eintrag `bg-shell-raised text-white` mit 2 px
Leiste in `accent-light` links. Kein farbiger Hintergrund, ein dauerhaft
sichtbarer Zustand darf nicht laut sein.

**Seitenkopf.** Titel in `text-display`, darunter eine Zeile Kontext in
`text-caption text-ink-subtle`, rechts die Primäraktion, darunter eine Haarlinie
über die volle Breite. Kein Icon neben dem Titel, kein Kasten um den Kopf.

**Mobil.** Unter 768 px wird die Navigation zur Schublade. Tabellen mit mehr als
vier Spalten werden zu Listenzeilen statt horizontal zu scrollen. Bei Beträgen
ist seitliches Scrollen unbrauchbar.

---

## 13. Zustände

Jede Ansicht mit Daten braucht vier Zustände. Fehlt einer, ist die Ansicht nicht
fertig.

**Leer, noch nichts erfasst.** Zentriert, Icon 24 px auf `accent-soft`,
`text-heading` mit dem Grund, ein Satz in `text-body text-ink-muted`, darunter
die Aktion, die weiterhilft.

**Leer, weil der Filter greift.** Anderer Text: "Keine Buchungen für diesen
Filter", plus Button "Filter zurücksetzen". Wer diese beiden Zustände verwechselt,
lässt Nutzer glauben, ihre Daten seien weg.

**Lädt.** Skelettzeilen nach §8.4.

**Fehler.** Was passiert ist, in einem Satz, und was jetzt zu tun ist. Technische
Details hinter "Details anzeigen". Keine Stack-Traces im Klartext.

---

## 14. Barrierefreiheit

- Kontrast: mindestens 4,5:1 für Text, 3:1 für Rahmen und bedeutungstragende
  Icons. Die Tokens in §3 erfüllen das, `ink-faint` ist deshalb für Fließtext
  gesperrt.
- Fokus überall sichtbar: 2 px `accent`, 2 px Abstand. Nie `outline: none` ohne
  Ersatz.
- Kein Zustand allein über Farbe. Storno hat Rand, Badge und Text.
- Tabellen sind mit Pfeiltasten navigierbar, siehe §8.6.
- Dialoge fangen den Fokus und geben ihn zurück.
- Jede Farbaussage muss einen Schwarz-Weiß-Ausdruck überstehen. Auswertungen
  werden gedruckt.

---

## 15. Text

Buchfink erklärt, ohne zuzutexten. Wer täglich damit arbeitet, liest den
Erklärsatz beim zwanzigsten Mal nicht mehr, sondern scrollt an ihm vorbei. Text,
der immer sichtbar ist, obwohl man ihn selten braucht, ist deshalb kein Service,
sondern Lärm.

Die Regel: **Eine Arbeitsansicht enthält keinen Fließtext.** Was länger als ein
Satz ist, wandert hinter ein Erklärzeichen.

### 15.1 Textbudget

| Ort | Erlaubt |
|---|---|
| Seitenkopf | Titel plus eine Zeile Kontext, höchstens 60 Zeichen |
| Feldhilfe unter dem Feld | höchstens sechs Wörter, sonst Tooltip |
| Fehlermeldung | zwei Sätze: Ursache, nächster Schritt |
| Leerzustand | Überschrift, ein Satz, eine Aktion |
| Hinweisstreifen | ein Satz |
| Tabellenzelle | kein Erklärtext, nie |

Alles darüber gehört in eine Erklärung nach §15.2. Wer beim Schreiben merkt, dass
ein Absatz nötig wäre, hat entweder die Oberfläche zu erklärungsbedürftig gebaut
oder schreibt gerade Dokumentation an der falschen Stelle.

### 15.2 Drei Stufen der Erklärung

Ausgelöst wird immer bewusst, nie automatisch. Es gibt keine Tour, kein
Popover beim ersten Besuch, keinen Hinweis, der von selbst aufgeht.

| Stufe | Umfang | Auslöser | Einsatz |
|---|---|---|---|
| Tooltip | ein Satz, kein Link, nichts Klickbares | Hover und Fokus, 400 ms Verzögerung | Was ist dieses Feld, was tut dieser Button |
| Popover | bis drei Sätze, ein Link "Mehr dazu" erlaubt | Klick auf das Erklärzeichen | Fachbegriff, Rechenweg, warum eine Aktion gesperrt ist |
| Dialog | mehr, mit Beispiel oder Tabelle | Klick auf "Mehr dazu" | SKR04-Kontenlogik, GoBD-Regeln, E-Bilanz-Mapping |

Das Erklärzeichen ist ein Fragezeichen in `ink-faint`, das bei Hover und Fokus
auf `ink-muted` wechselt. Klickfeld 24 mal 24 px, auf Touch-Geräten 44. Es steht
hinter der Beschriftung, nie davor, und ist mit der Tastatur erreichbar. Escape
schließt.

Ein Tooltip erklärt nur. Sobald etwas darin klickbar wäre, ist es ein Popover.

### 15.3 Wortwahl

- Sie-Form, kurze Sätze, keine Ausrufezeichen.
- Fachbegriffe werden verwendet, nicht umschrieben. Die Zielgruppe bilanziert.
  Erklärt wird über §15.2, nicht durch vereinfachte Labels.
- Fehlermeldungen nennen Ursache und nächsten Schritt: "Die Buchung ist nicht
  ausgeglichen. Soll (1.190,00 €) und Haben (1.000,00 €) müssen übereinstimmen."
  Nicht: "Ungültige Eingabe."
- Buttons tragen Verben: "Buchung festschreiben", nicht "OK".
- Bestätigungen benennen die Folge, siehe §8.2.
- Kein Text, der nur beschreibt, was ohnehin zu sehen ist. Eine Tabelle mit fünf
  Spalten braucht keinen Satz darüber, der die fünf Spalten aufzählt.

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

## 17. Umsetzung

Die Tokens stehen in `frontend/src/index.css`. Alle bisherigen Tailwind-Klassen
funktionieren weiter, die Migration läuft schrittweise.

**Schritt 1, Bausteine. Erledigt.** `components/ui/` steht, siehe §10, mit
Base UI als Verhaltensebene. Noch nicht erledigt ist die Ablösung:
`components/Form.tsx` und `components/HelpTooltip.tsx` bleiben in Betrieb, bis
die Seiten umgestellt sind, und verschwinden dann.

Solange keine Seite die Bausteine importiert, kostet Base UI nichts: Das
JS-Bundle bleibt bei 2.173 kB, die Bibliothek wird vollständig wegoptimiert. Das
CSS wächst um 12,7 kB, weil Tailwind die Klassen der Bausteine bereits erzeugt.

**Schritt 2, Rahmen.** Sidebar, Header und App auf Tokens umstellen. Danach
stimmt der erste Eindruck, auch wenn die Seiten noch alt aussehen.

**Schritt 3, Seiten.** Nach verbrachter Arbeitszeit sortiert: Journal, Bank,
Belege, Rechnungen, Auswertungen, Rest. Dabei fallen die 58 Karten.

**Schritt 4, Aufräumen.** `stone-*`, `amber-*`, `rose-*`, `emerald-*` und alle
Hex-Literale aus `src/` entfernen, danach als Grep-Prüfung in CI.

### Prüfliste für jeden Pull Request

- [ ] Keine neue Fläche außerhalb der fünf Fälle aus §6.2, keine Fläche in einer Fläche
- [ ] Abschnitte durch Überschrift, Abstand und Haarlinie getrennt, nicht durch Kästen
- [ ] Tabellenüberschrift über der Fläche, Kopfzeile ohne Füllung
- [ ] Kein Fließtext in der Ansicht, Erklärungen hinter dem Erklärzeichen (§15)
- [ ] Textbudget je Ort eingehalten (§15.1)
- [ ] Keine Hex-Farben, keine Klassen aus den alten Paletten
- [ ] `-text` für Text, Basis für Marker, `-soft` mit `-line` für Flächen
- [ ] Schriftgrößen aus der Skala in §4
- [ ] Radien nur `control`, `overlay`, `full`
- [ ] Schatten nur an Popover und Dialog
- [ ] Jede Zahlenspalte hat `.num` und ist rechtsbündig
- [ ] Nur Bewegungen aus der Tabelle in §7
- [ ] Farbige Zustände tragen zusätzlich Text oder Icon
- [ ] Leer-, Lade- und Fehlerzustand vorhanden
- [ ] Mit der Tastatur bedienbar, Fokus sichtbar, Fokusrückgabe geregelt
- [ ] Beträge und Daten über `utils/formatters.ts`
