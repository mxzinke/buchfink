# Design-Konzept

Dieses Dokument beschreibt die visuelle und interaktive Grundlage von Buchfink:
warum die Oberfläche so aussieht, welche Entscheidungen bindend sind und wie neue
Ansichten gebaut werden. Es ist die Referenz für Code-Reviews — eine Ansicht, die
gegen die Regeln hier verstößt, ist ein Fehler, keine Geschmacksfrage.

Die Tokens sind in [`frontend/src/index.css`](../frontend/src/index.css) als
Tailwind-`@theme` hinterlegt und damit direkt als Utility-Klassen (`bg-paper`,
`text-ink`, `rounded-card`) und als CSS-Variablen (`var(--color-ink)`) verfügbar.

**Visuelle Referenz:** [`design-konzept.html`](./design-konzept.html) — einfach im
Browser öffnen. Die Seite zeigt Palette, Schriftskala, Komponenten und die
fachlichen Muster als gerenderte Beispiele und ist selbst in den Tokens gesetzt,
die sie beschreibt.

---

## 1. Ausgangslage

Die bestehende Oberfläche hat eine stimmige Grundstimmung — warmes Papier,
Manrope, dunkle Navigationsspalte. Was fehlt, ist ein System dahinter. Eine
Auszählung des Frontends zeigt den Zustand:

| Dimension | Ist-Zustand | Folge |
|---|---|---|
| Eckenradien | 6 verschiedene (`rounded-xs` bis `rounded-2xl`), 118× `lg` neben 97× `xl` | Karten wirken zufällig unterschiedlich |
| Schriftgrößen | 9 Stufen, darunter `text-[9px]`, `text-[10px]`, `text-[11px]` | Hierarchie ist nicht ablesbar, 9 px ist unlesbar |
| Farbe | rund 45 verschiedene Farb-Utilities aus 4 Paletten | Bernstein bedeutet mal Marke, mal Aktion, mal Ergebnis |
| Zahlen | `font-mono` = Systemschrift | Sieht auf jedem Betriebssystem anders aus, bricht mit Manrope |
| Fokus | überwiegend Browser-Standard oder unterdrückt | Tastaturbedienung ist nicht verlässlich |

Nichts davon ist im Einzelfall falsch. In Summe kostet es die Ruhe, die eine
Buchhaltung braucht, und es macht jede neue Ansicht zu einer Neuerfindung.

---

## 2. Leitidee: Stilles Kontor

Buchfink ist Werkzeug, keine Bühne. Wer damit arbeitet, sitzt vier Stunden am
Stück vor Journal und Kontenabgleich und sucht Abweichungen. Die Oberfläche hat
dabei genau eine Aufgabe: nicht im Weg zu stehen.

**1. Ruhe ist Funktion, nicht Geschmack.**
Jedes Element, das Aufmerksamkeit zieht, ohne sie zu verdienen, verlängert die
Suche nach dem, was wirklich zählt. Keine Farbflächen ohne Bedeutung, keine
Schatten ohne Ebenenwechsel, keine Animation ohne Zustandswechsel.

**2. Farbe ist Information.**
In einer Buchhaltung ist Farbe reserviert. Grün heißt geprüft oder im Plus, Rot
heißt Storno oder im Minus, Bernstein heißt „das ist noch offen" — und markiert
zusätzlich den aktuellen Ort in der Navigation. Alles andere ist Papier und
Tinte. Eine dekorativ eingefärbte Fläche verbraucht Bedeutung, die später fehlt.

**3. Die Zahl ist der Held.**
Beträge, Salden und Belegnummern sind der Inhalt. Sie stehen rechtsbündig, in
gleicher Ziffernbreite, mit ausreichend Luft — und ohne dass Rahmen, Icons oder
Hintergründe mit ihnen konkurrieren.

**4. Nichts verschwindet.**
Die GoBD verlangt, dass Korrekturen sichtbar bleiben. Das ist keine Last, sondern
ein Gestaltungsprinzip: Stornierte Buchungen werden markiert, nicht versteckt,
und nie durchgestrichen — der ursprüngliche Betrag muss lesbar bleiben.

---

## 3. Farbe

### 3.1 Papier und Tinte — die Grundfläche

Warme Neutraltöne statt Grau. Grau wirkt technisch, warmes Papier passt zu
Belegen und ermüdet bei langer Nutzung weniger.

| Token | Wert | Einsatz |
|---|---|---|
| `paper` | `#FAF8F5` | App-Hintergrund, Seitenfläche |
| `surface` | `#FFFFFF` | Karten, Tabellenzeilen, Dialoge |
| `sunken` | `#F4F1EB` | Tabellenkopf, Wells, gesperrte Bereiche |
| `line` | `#E9E4DC` | Haarlinie, Standardtrennung |
| `line-strong` | `#D8D1C6` | Außenkanten, Summenlinie |
| `control-border` | `#948E85` | Rahmen bedienbarer Elemente — 3,3:1 auf Weiß |
| `ink` | `#1C1917` | Primärtext, Primärbutton — 16,5:1 auf Papier |
| `ink-muted` | `#57514A` | Sekundärtext, Beschreibungen — 7,4:1 |
| `ink-subtle` | `#756E65` | Labels, Metadaten, Tabellenkopf — 4,7:1 |
| `ink-faint` | `#A79F94` | Deaktiviert, dekorative Icons — **nie Fließtext** |

`control-border` ist bewusst dunkler als die Haarlinien. Eine Haarlinie erreicht
die von WCAG 1.4.11 geforderten 3:1 nicht, und ein Eingabefeld muss als
bedienbar erkennbar sein. Struktur (Karten, Trennlinien) darf hell bleiben,
Bedienelemente nicht.

Die dunkle Navigationsspalte behält ihre eigene Skala, damit sie als eigener Raum
lesbar bleibt:

| Token | Wert | Einsatz |
|---|---|---|
| `shell` | `#24211E` | Navigationsfläche |
| `shell-deep` | `#1B1917` | Fußbereich der Navigation, Startbildschirm |
| `shell-raised` | `#2E2A26` | Aktiver Eintrag, Hover |
| `shell-line` | `#37322D` | Trennlinien im Dunkeln |
| `shell-text` | `#D8D2CA` | Text auf dunkler Fläche |
| `shell-text-muted` | `#968E84` | Gruppenlabels, Sekundärtext im Dunkeln |

### 3.2 Die drei Signalfarben

Alle drei stammen aus dem Logo. Das ist kein Zufall, sondern hält Marke und
Semantik zusammen.

| Token | Wert | Bedeutung — und nur diese |
|---|---|---|
| `accent` | `#A8620A` | Marke, aktueller Ort in der Navigation, **offen / zu erledigen** |
| `accent-strong` | `#8A4F06` | Hover- und Aktivzustand von `accent` |
| `accent-light` | `#E0A356` | dieselbe Rolle auf dunkler Fläche |
| `accent-soft` / `accent-line` | `#F7EDDD` / `#E8D5B4` | getönte Fläche und deren Rahmen |
| `positive` | `#1B5E20` | geprüft, abgestimmt, festgeschrieben, im Plus |
| `positive-soft` / `positive-line` | `#E9F3EA` / `#CBE3CE` | Fläche und Rahmen |
| `negative` | `#A5342A` | Storno, Fehler, Integritätsbruch, im Minus |
| `negative-soft` / `negative-line` | `#FBECE9` / `#F0D2CB` | Fläche und Rahmen |

**Blau kommt nicht vor.** Es ist die Standardfarbe für „Information" in fast
jeder Business-Software und genau deshalb der schnellste Weg, die Anwendung
austauschbar aussehen zu lassen. Hinweise werden in Papier und Tinte gesetzt.

### 3.3 Verbindliche Regeln

1. **Primäraktionen sind Tinte, nicht Bernstein.** `bg-ink text-white`. Eine
   Buchhaltung hat auf jeder Seite eine wichtigste Aktion; sie muss sich
   abheben, ohne zu leuchten. Bernstein bleibt dadurch als Statusfarbe frei.
2. **Bernstein markiert genau zwei Dinge:** wo man gerade ist, und was noch
   offen ist. Nicht die Marke auf jeder zweiten Fläche.
3. **Soll und Haben werden nie eingefärbt.** Beide sind neutral. Farbe an dieser
   Stelle suggeriert eine Wertung, die es fachlich nicht gibt. Gefärbt wird nur
   das *Ergebnis* (Saldo, Gewinn/Verlust) und der *Zustand* (offen, storniert).
4. **Farbe steht nie allein.** Jeder farbige Zustand trägt zusätzlich Text oder
   ein Icon — für Rot-Grün-Sehschwäche und für Schwarz-Weiß-Ausdrucke.
5. **Getönte Flächen bekommen immer den passenden Rahmen** (`bg-positive-soft
   border border-positive-line`). Eine randlose Farbfläche wirkt wie ein Fehler
   im Rendering.

---

## 4. Typografie

Manrope trägt die gesamte Oberfläche, in sieben Stufen. Mehr braucht es nicht;
was nicht in diese Tabelle passt, ist ein Layoutproblem, kein Schriftgrößenproblem.

| Token | Größe / Zeilenhöhe | Gewicht | Einsatz |
|---|---|---|---|
| `text-display` | 22 / 28 px | 600 | Seitentitel — genau einer pro Ansicht |
| `text-heading` | 16 / 22 px | 600 | Karten- und Abschnittstitel, Dialogtitel |
| `text-body` | 13 / 20 px | 400 | Fließtext, Tabellenzellen, Formularwerte |
| `text-label` | 12 / 16 px | 500 | Feldbeschriftungen, Tabellenkopf, Buttons |
| `text-caption` | 11 / 15 px | 400 | Hilfstexte, Metadaten, Zeitstempel |
| `text-overline` | 10 / 14 px | 600, 0,08 em, Versalien | Gruppenlabels in der Navigation |

Zwei Anmerkungen zur Umstellung: Der bisherige Standard war `text-xs` (12 px) für
fast alles. 13 px als Fließtext ist bei stundenlanger Arbeit spürbar
angenehmer, ohne dass die Dichte leidet. Und `text-[9px]` entfällt ersatzlos —
das ist unter jeder vertretbaren Lesbarkeitsgrenze.

**Gewichte:** 400 für Text, 500 für Labels, 600 für Überschriften und Beträge in
Summenzeilen. 700 und 800 werden nicht verwendet — Manrope wird in diesen
Schnitten laut und passt nicht zur ruhigen Anmutung.

### 4.1 Zahlensatz

Manrope bringt tabellarische Ziffern (`tnum`) und eine durchgestrichene Null
(`zero`) mit. Damit ist kein zweiter Schriftschnitt nötig, und Beträge bleiben
Teil des Textbildes statt als Fremdkörper im Monospace zu stehen.

| Utility | Wirkung | Einsatz |
|---|---|---|
| `.num` | tabellarische Ziffern | **Alle** Beträge, Salden, Prozentsätze, Datumsangaben, Mengen |
| `.code-num` | tabellarische Ziffern + durchgestrichene Null | Beleg- und Kontonummern, IBAN, Steuer- und USt-IdNr. |
| `font-mono` | Systemschrift | Nur Hashes, Dateipfade, XML-Fragmente |

Ohne `.num` springen Zahlen beim Aktualisieren und Spalten stehen nicht
untereinander. Eine Betragsspalte ohne `.num` ist ein Fehler.

**Formatierung** bleibt durchgängig de-DE und liegt bereits in
`utils/formatters.ts`: `1.234,56 €`, `01.01.2024`, echtes Minuszeichen `−`
(U+2212, nicht der Bindestrich), `—` für „kein Wert".

---

## 5. Raster, Abstände, Dichte

Alles ist ein Vielfaches von 4 px. Erlaubt sind: **4, 8, 12, 16, 24, 32, 48, 64**.

| Maß | Wert |
|---|---|
| Seitenrand (Desktop / Mobil) | 32 px / 16 px |
| Maximale Inhaltsbreite | 1200 px, zentriert — Tabellen dürfen auf volle Breite |
| Abstand zwischen Abschnitten | 32 px |
| Karten-Innenabstand | 20 px |
| Abstand Label → Feld | 4 px |
| Abstand zwischen Feldern | 16 px |

**Dichte.** Zwei Stufen, umschaltbar in den Einstellungen und pro Mandant
gespeichert:

| Stufe | Zeilenhöhe | Einsatz |
|---|---|---|
| Kompakt | 32 px | Journal, Kontenabgleich, SuSa — viele Zeilen im Blick |
| Komfortabel | 40 px | Standard, Stammdaten, alles mit Bearbeitungsaktionen |

Berührungsziele bleiben in beiden Stufen mindestens 32 × 32 px; auf Touch-Geräten
44 × 44 px.

---

## 6. Form und Höhe

**Radien** — drei plus rund, mehr nicht:

| Token | Wert | Einsatz |
|---|---|---|
| `rounded-control` | 6 px | Buttons, Eingabefelder, Chips, Badges |
| `rounded-card` | 10 px | Karten, Tabellencontainer, Panels |
| `rounded-overlay` | 14 px | Dialoge, Popover, Dropdowns |
| `rounded-full` | — | Avatare, Statuspunkte, Zähler-Pills |

**Rahmen statt Schatten.** Elemente im Textfluss werden durch eine Haarlinie
(`border border-line`) abgegrenzt, nicht durch Schatten. Ein Schatten bedeutet:
Dieses Element schwebt wirklich über der Seite.

| Token | Einsatz |
|---|---|
| kein Schatten | Karten, Tabellen, Panels, Buttons, Eingabefelder |
| `shadow-popover` | Dropdowns, Popover, Tooltips, Kontextmenüs |
| `shadow-dialog` | Modale Dialoge |

Damit verschwinden `shadow-xs`, `shadow-sm` und `shadow-xl` aus dem Code.

---

## 7. Bewegung

Bewegung zeigt Zustandswechsel an, sonst nichts.

| Zweck | Dauer | Kurve |
|---|---|---|
| Hover, Fokus, Aktiv | 120 ms | `ease-quiet` |
| Ein-/Ausblenden von Overlays | 180 ms | `ease-quiet` |
| Seitenwechsel | ohne Animation | — |

`ease-quiet` ist `cubic-bezier(.2,.7,.3,1)`: schneller Start, weiches Ende, kein
Nachschwingen. Kein Federn, kein Skalieren von Karten beim Hover, keine
Dauerbewegung außer dem Ladeindikator der Integritätsprüfung.
`prefers-reduced-motion` schaltet alles ab — das ist im Basis-Layer bereits
umgesetzt.

---

## 8. Ikonografie

Lucide, ausschließlich, mit `stroke-width={1.5}`. Der Standardwert 2 ist für die
kleinen Größen in dieser Anwendung zu fett.

| Größe | Einsatz |
|---|---|
| 14 px | in Tabellenzeilen, Badges, Buttons `sm` |
| 16 px | Navigation, Buttons `md`, Feldsymbole |
| 20 px | Dialogtitel, Leerzustände |
| 24 px | Leerzustände mit getönter Fläche |

Icons sind Begleiter, nicht Ersatz für Text. Eine Aktion, die nur als Icon
existiert, braucht ein `title` **und** ein `aria-label`. Farbige Icons folgen der
Farbregel: bernstein nur für „offen", grün nur für „geprüft", rot nur für Fehler.

---

## 9. Komponenten

Die folgenden Klassenketten sind die verbindliche Umsetzung. Sie gehören in
`components/ui/` und werden nicht pro Seite neu geschrieben.

### Buttons

| Variante | Klassen | Einsatz |
|---|---|---|
| Primär | `h-9 px-4 rounded-control bg-ink text-white text-label font-semibold hover:bg-ink/90 disabled:bg-ink-faint disabled:cursor-not-allowed transition-colors duration-120` | genau eine pro Ansicht |
| Sekundär | `h-9 px-4 rounded-control border border-line-strong bg-surface text-ink-muted text-label font-semibold hover:bg-sunken hover:text-ink transition-colors` | alle weiteren Aktionen |
| Unauffällig | `h-8 px-2.5 rounded-control text-ink-subtle text-label hover:bg-sunken hover:text-ink transition-colors` | Zeilenaktionen, Toolbars |
| Destruktiv | `h-9 px-4 rounded-control bg-negative text-white text-label font-semibold hover:brightness-95` | nur Storno und Löschen |

Höhen: 36 px (`h-9`) Standard, 32 px (`h-8`) kompakt. Icon-only-Buttons sind
quadratisch. Ein deaktivierter Button braucht immer eine Erklärung im `title` —
„ausgegraut ohne Grund" ist die häufigste Frustquelle in Buchhaltungssoftware.

### Eingabefelder

```
h-9 w-full px-3 rounded-control border border-control-border bg-surface text-body
placeholder:text-ink-faint
focus:border-accent focus:ring-2 focus:ring-accent/20 outline-none
disabled:bg-sunken disabled:text-ink-faint
aria-[invalid=true]:border-negative aria-[invalid=true]:ring-2 aria-[invalid=true]:ring-negative/15
```

Betragsfelder zusätzlich `text-right num`. Label darüber (`text-label
text-ink-muted mb-1`), Hilfstext darunter (`text-caption text-ink-subtle mt-1`),
Fehlertext ersetzt den Hilfstext in `text-negative`. Pflichtfelder werden nicht
mit Sternchen markiert — stattdessen werden optionale Felder mit „(optional)"
gekennzeichnet, das sind in Buchfink die wenigeren.

### Karte

```
rounded-card border border-line bg-surface
```

Kopfzeile `px-5 py-4 border-b border-line`, Inhalt `p-5`. Kein Schatten. Karten
werden nicht verschachtelt — eine Karte in einer Karte ist ein Zeichen dafür,
dass der Abschnitt eine eigene Ansicht sein sollte.

### Tabelle

Das wichtigste Element der Anwendung.

| Teil | Umsetzung |
|---|---|
| Container | `rounded-card border border-line bg-surface overflow-hidden` |
| Kopf | `bg-sunken border-b border-line text-label text-ink-subtle font-medium`, `sticky top-0 z-10` |
| Zelle | `px-4 h-10 text-body` (kompakt: `h-8`) |
| Trennung | `divide-y divide-line` — **keine Zebrastreifen** |
| Hover | `hover:bg-sunken/60` |
| Ausgewählt | `bg-accent-soft` |
| Zahlenspalten | `text-right num`, Kopf ebenfalls rechtsbündig |
| Summenzeile | `rule-total font-semibold bg-sunken` |
| Zeilenaktionen | bei Hover und Fokus sichtbar, per Tastatur immer erreichbar |

Spaltenreihenfolge im Journal, fest: Belegnummer · Datum · Buchungstext · Konten
(Soll → Haben) · Betrag · Status. Text links, Zahlen rechts, nichts zentriert.

### Status-Badge

```
inline-flex items-center gap-1.5 h-5 px-2 rounded-control text-caption font-medium
border
```

Dazu das Farbpaar des Zustands, zum Beispiel `bg-positive-soft text-positive
border-positive-line`. Ein Badge enthält immer Text, nie nur einen Punkt.

Eine Ausnahme bei Bernstein: auf `accent-soft` wird `accent-strong` als
Textfarbe verwendet, nicht `accent`. Letzteres erreicht auf der getönten Fläche
nur 4,1:1 und ist bei 11 px zu wenig.

### Dialog

Overlay `bg-ink/40`, Panel `rounded-overlay border border-line bg-surface
shadow-dialog max-w-2xl`. Kopf `px-6 py-4 border-b border-line` mit
`text-heading`, Fuß `px-6 py-4 border-t border-line` mit rechtsbündigen
Aktionen (Sekundär links von Primär). Escape schließt, Fokus wird gefangen und
beim Schließen an den auslösenden Button zurückgegeben.

Ein Dialog ist für eine abgeschlossene Eingabe da. Alles, was länger als ein
Bildschirm ist — Rechnungserfassung, Beleg-Buchungsflow — gehört in eine eigene
Ansicht, nicht in ein Modal.

### Meldungen

| Art | Umsetzung |
|---|---|
| Erfolg einer Aktion | Toast unten rechts, 4 s, `text-body` |
| Fachlicher Fehler im Formular | Inline am Feld, nie als Toast |
| Fehler aus dem Backend | Kasten über den Aktionen: `rounded-control border border-negative-line bg-negative-soft px-4 py-3 text-body text-negative` |
| Dauerhafter Zustand | Statusleiste, kein Toast — Integrität und Periodensperre gehören dorthin |

---

## 10. Fachliche Muster

Der Teil, der Buchfink von einer beliebigen App unterscheidet.

### 10.1 Soll und Haben

Zwei getrennte, rechtsbündige Spalten mit `.num` — nicht eine Spalte mit
Vorzeichen. Soll steht links, Haben rechts, so wie im Buch. Beide Spalten sind
farblich neutral. Unter der letzten Position steht die Summe mit der
buchhalterischen Doppellinie (`rule-total`); Soll- und Habensumme müssen
sichtbar gleich sein — das ist die Kontrolle, die Buchhalter als erstes suchen.

Kontonummern in `.code-num`, dahinter die Kontobezeichnung in `text-ink-muted`.
Die Nummer allein reicht niemandem, der SKR04 nicht auswendig kann.

### 10.2 Storno und Generalumkehr

Stornierte Buchungen werden **nie durchgestrichen und nie ausgeblendet**. Der
ursprüngliche Betrag muss lesbar bleiben, sonst ist die Nachvollziehbarkeit dahin.

```
Zeile:  border-l-2 border-negative bg-negative-soft/40
Badge:  „Storniert" (rot) bzw. „Storno zu #RE-2024-014" auf der Gegenbuchung
Betrag: normale Darstellung, kein line-through
```

Die Stornobuchung verlinkt auf das Original und umgekehrt. Beide Richtungen,
immer.

### 10.3 Status-Vokabular

Ein Wort pro Zustand, in der ganzen Anwendung dasselbe. Diese Liste ist
abschließend:

| Status | Farbe | Gilt für |
|---|---|---|
| Entwurf | neutral (`ink-subtle`) | Rechnung, Buchung vor dem Festschreiben |
| Offen | bernstein | Rechnung, Offener Posten, Beleg ohne Buchung |
| Teilweise ausgeglichen | bernstein, nur Rahmen | Offener Posten |
| Zugeordnet | neutral | Banktransaktion mit Beleg |
| Gebucht | grün | Buchung im Journal |
| Ausgeglichen | grün | Offener Posten |
| Festgeschrieben | grün, mit Schloss-Icon | Buchung, Periode |
| Überfällig | rot | Rechnung, Steuerfrist |
| Storniert | rot | Buchung, Rechnung |
| Fehlerhaft | rot | Import, Validierung, Integritätsprüfung |

Nicht erlaubt: Synonyme wie „erledigt", „fertig", „abgeschlossen" für denselben
Zustand.

### 10.4 Integrität der Hash-Chain

Der Zustand steht dauerhaft im Fuß der Navigation, nie als Toast. Drei
Zustände, drei Formulierungen:

| Zustand | Darstellung |
|---|---|
| geprüft | grüner Haken, „Daten unverändert", darunter Zeitpunkt der Prüfung |
| wird geprüft | rotierendes Icon in `ink-subtle`, „Prüfung läuft" |
| gebrochen | rotes Schild, „Integrität verletzt", Klick öffnet das Protokoll mit der ersten abweichenden Buchung |

Ein Integritätsbruch ist der einzige Fall, in dem die Oberfläche laut werden
darf: roter Balken über dem gesamten Inhalt, bis er quittiert ist.

### 10.5 Gesperrte Perioden

Abgeschlossene Geschäftsjahre sind schreibgeschützt. Sichtbar durch:
Hintergrund `sunken` statt `surface`, ein Schloss-Icon neben dem Jahr in der
Kopfzeile, und ein Hinweisstreifen am Kopf der Ansicht: „Geschäftsjahr 2024 ist
abgeschlossen. Buchungen sind nur noch im laufenden Jahr möglich."
Bearbeitungsaktionen werden deaktiviert und behalten ihre Position — sie
verschwinden nicht, damit die Ansicht zwischen den Jahren gleich aussieht.

### 10.6 Belege

Zwei Spalten: links das Dokument, rechts die Erfassung. Die Vorschau bleibt
sichtbar, während gebucht wird — die häufigste Tätigkeit ist der Abgleich
zwischen Beleg und Feld, und der darf keinen Fenster- oder Tab-Wechsel kosten.
Erkannte Werte aus der E-Rechnung werden als Vorschlag im Feld angezeigt, mit
einem dezenten Hinweis „aus ZUGFeRD übernommen" — übernommene Werte sind
editierbar und werden nicht als gesichert dargestellt.

---

## 11. Layout

```
┌──────────────┬─────────────────────────────────────────────┐
│              │  Kopfzeile 56 px — Geschäftsjahr, Mandant   │
│  Navigation  ├─────────────────────────────────────────────┤
│  240 px      │                                             │
│  dunkel      │  Seitentitel + Primäraktion                 │
│              │  ─────────────────────────────────────────  │
│              │  Inhalt, max. 1200 px                       │
│              │                                             │
│  ──────────  │                                             │
│  Integrität  │                                             │
└──────────────┴─────────────────────────────────────────────┘
```

**Navigation.** 240 px, fünf Gruppen (Übersicht, Buchhaltung, Stammdaten,
Auswertungen, Verwaltung). Aktiver Eintrag: `bg-shell-raised text-white` mit
2 px bernsteinfarbener Leiste links. Kein bernsteinfarbener Hintergrund — das
war bisher zu laut für einen Zustand, der dauerhaft sichtbar ist.

**Seitenkopf.** Jede Ansicht beginnt gleich: `text-display` Titel, darunter eine
Zeile Kontext in `text-caption text-ink-subtle`, rechts die Primäraktion. Kein
Icon neben dem Titel.

**Mobil.** Unter 768 px wird die Navigation zur Schublade. Tabellen mit mehr als
vier Spalten werden zu Listenkarten, statt horizontal zu scrollen — bei Beträgen
ist seitliches Scrollen unbrauchbar.

---

## 12. Zustände

Jede Ansicht mit Daten braucht vier Zustände. Fehlt einer, ist die Ansicht nicht
fertig.

**Leer (noch nichts erfasst).** Zentriert, Icon 24 px auf getönter Fläche
(`bg-accent-soft`), `text-heading` mit dem Grund, ein Satz in `text-body
text-ink-muted`, darunter die eine Aktion, die weiterhilft. Kein Verweis auf
Dokumentation als einzige Option.

**Leer (Filter greift nicht).** Anderer Text als oben: „Keine Buchungen für
diesen Filter" plus Button „Filter zurücksetzen". Die Verwechslung dieser beiden
Zustände lässt Nutzer glauben, ihre Daten seien weg.

**Lädt.** Skelettzeilen in `bg-sunken` in der Form der erwarteten Tabelle, kein
Spinner und kein Text „wird geladen". Unter 200 ms wird gar nichts angezeigt.

**Fehler.** Was passiert ist, in einem Satz, und was jetzt zu tun ist. Technische
Details hinter „Details anzeigen". Keine Stack-Traces im Klartext.

---

## 13. Barrierefreiheit und Tastatur

- Kontrast: mindestens 4,5:1 für Text, 3:1 für Rahmen und Icons, die Bedeutung
  tragen. Die Tokens in Abschnitt 3 erfüllen das; `ink-faint` ist deshalb für
  Fließtext gesperrt.
- Fokus ist überall sichtbar: 2 px `accent`, 2 px Abstand. Nie `outline: none`
  ohne Ersatz.
- Kein Zustand allein über Farbe. Storno hat Rand, Badge und Text.
- Tabellen sind mit Pfeiltasten navigierbar, `Enter` öffnet die Zeile.
- Shortcuts, global: `⌘K` Suche, `⌘N` neue Buchung im aktuellen Kontext,
  `⌘S` speichern im Dialog, `Esc` schließen. In Zahlenfeldern erhöht `↑`/`↓`
  um 1,00 €.
- Dialoge fangen den Fokus und geben ihn beim Schließen zurück.
- Jede Farbaussage muss einen Schwarz-Weiß-Ausdruck überstehen — Auswertungen
  werden gedruckt.

---

## 14. Sprache

- Sie-Form, kurze Sätze, keine Ausrufezeichen.
- Fachbegriffe werden verwendet, nicht umschrieben — die Zielgruppe bilanziert.
  Erklärungen kommen über `HelpTooltip`, nicht durch vereinfachte Labels.
- Fehlermeldungen nennen die Ursache und den nächsten Schritt: „Die Buchung ist
  nicht ausgeglichen. Soll (1.190,00 €) und Haben (1.000,00 €) müssen
  übereinstimmen." Nicht: „Ungültige Eingabe."
- Buttons tragen Verben: „Buchung festschreiben", nicht „OK".
- Bestätigungen für irreversible Schritte benennen die Folge: „Festschreiben
  kann nicht rückgängig gemacht werden. Korrekturen sind danach nur per Storno
  möglich."

---

## 15. Dunkelmodus

Vorbereitet, nicht Teil der ersten Umsetzung. Da alle Farben über Tokens laufen,
genügt später ein zweiter Satz Werte unter `[data-theme="dark"]`: `paper` wird
`#1B1917`, `surface` wird `#24211E`, die Tinte-Skala kehrt sich um. Die
Signalfarben brauchen hellere Varianten (`positive` → `#7CC08A`, `negative` →
`#E58A7E`, `accent` → `accent-light`), weil die dunklen Töne auf dunklem Grund
den Kontrast verlieren. Bedingung für den Start: kein Hardcode-Hex mehr im
Komponentencode.

---

## 16. Umsetzung

Die Tokens stehen bereits in `frontend/src/index.css`. Alle bisherigen
Tailwind-Klassen funktionieren unverändert weiter, die Migration kann also
schrittweise laufen.

**Schritt 1 — Bausteine.** `components/ui/` mit `Button`, `Input`, `Field`,
`Card`, `Table`, `StatusBadge`, `Dialog`, `EmptyState`, `PageHeader`. Die
Spezifikationen aus Abschnitt 9 einmal umsetzen. `components/Form.tsx` geht darin
auf.

**Schritt 2 — Rahmen.** `Sidebar`, `Header`, `App` auf Tokens umstellen. Danach
ist der erste Eindruck stimmig, auch wenn die Seiten noch alt aussehen.

**Schritt 3 — Seiten.** In dieser Reihenfolge, weil dort die meiste Zeit
verbracht wird: Journal → Bank → Belege → Rechnungen → Auswertungen → Rest.

**Schritt 4 — Aufräumen.** `stone-*`, `amber-*`, `rose-*`, `emerald-*` und alle
`#`-Literale aus `src/` entfernen. Danach als Grep-Prüfung in CI, damit es so
bleibt.

### Prüfliste für jeden Pull Request

- [ ] Keine Hex-Farben und keine `stone-/amber-/rose-/emerald-`Klassen
- [ ] Schriftgrößen ausschließlich aus der Skala in Abschnitt 4
- [ ] Radien nur `control`, `card`, `overlay`, `full`
- [ ] Schatten nur an Popover und Dialog
- [ ] Jede Zahlenspalte hat `.num` und ist rechtsbündig
- [ ] Farbige Zustände tragen zusätzlich Text oder Icon
- [ ] Leer-, Lade- und Fehlerzustand vorhanden
- [ ] Mit der Tastatur bedienbar, Fokus sichtbar
- [ ] Beträge und Daten über `utils/formatters.ts`, kein `toLocaleString` im JSX
