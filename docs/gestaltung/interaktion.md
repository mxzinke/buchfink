# Interaktion

[Designkonzept](README.md) · [Dokumentation](../README.md)

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
und die Differenz wird laufend angezeigt. Das ist eine Rechenhilfe, kein Fehler,
solange die Buchung nicht abgeschickt ist.

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

## 13. Zustände

Jede Ansicht mit Daten braucht vier Zustände. Fehlt einer, ist die Ansicht nicht
fertig.

**Leer, noch nichts erfasst.** Zentriert, Icon 24 px auf `accent-soft`,
`text-heading` mit dem Grund, ein Satz in `text-body text-ink-muted`, darunter
die Aktion, die weiterhilft.

**Leer, weil der Filter greift.** Anderer Text: "Keine Buchungen für diesen
Filter", plus Button "Filter zurücksetzen". Wer diese beiden Zustände verwechselt,
lässt Nutzer glauben, ihre Daten seien weg.

**Lädt.** Skelettzeilen nach [Abschnitt 8.4](#84-warten).

**Fehler.** Was passiert ist, in einem Satz, und was jetzt zu tun ist. Technische
Details hinter "Details anzeigen". Keine Stack-Traces im Klartext.

---

## 14. Barrierefreiheit

- Kontrast: mindestens 4,5:1 für Text, 3:1 für Rahmen und bedeutungstragende
  Icons. Die Tokens in [Abschnitt 3](grundlagen.md#3-farbe) erfüllen das, `ink-faint` ist deshalb für Fließtext
  gesperrt.
- Fokus überall sichtbar: 2 px `accent`, 2 px Abstand. Nie `outline: none` ohne
  Ersatz.
- Kein Zustand allein über Farbe. Storno hat Rand, Badge und Text.
- Tabellen sind mit Pfeiltasten navigierbar, siehe [Abschnitt 8.6](#86-tastatur).
- Dialoge fangen den Fokus und geben ihn zurück.
- Jede Farbaussage muss einen Schwarz-Weiß-Ausdruck überstehen. Auswertungen
  werden gedruckt.

---
