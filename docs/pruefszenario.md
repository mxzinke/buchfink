# Prüfszenario: vom Abschluss zum Beleg

Ein sachverständiger Dritter muss sich in Buchfink ohne Erklärung zurechtfinden
(GOB-02, GoBD Rz. 145). Das Prüfszenario stellt diese Behauptung auf die
Probe: ein Beispielmandant, eine Aufgabe, eine Klickgrenze. Es läuft als
automatischer Test und lässt sich von Hand nachvollziehen.

## Der Beispielmandant

Die Beispieldaten des Screenshot-Werkzeugs (`scripts/site-screenshots/mock-bridge.ts`)
bilden einen kleinen Mandanten mit einem Geschäftsjahr, Bankkonto 1800,
Ausgangs- und Eingangsrechnungen, Bankumsätzen, Belegen und einer Bilanz. Die
Zahlen sind gerechnet, nicht gewürfelt: Soll gleich Haben, Aktiva gleich
Passiva, die Zahllast aus Umsatzsteuer und Vorsteuer. Wer eine Zahl ändert,
zieht die verbundenen mit.

## Die Aufgabe

Von der Bilanzposition, unter der das Bankkonto steht, zum Beleg der Buchung
B-2026-0052 in höchstens vier Klicks. Der Weg zur Bilanz selbst zählt nicht;
das Szenario beginnt dort, wo ein Prüfer sitzt, wenn er eine Position
hinterfragt.

| Klick | Wo | Was passiert |
|---|---|---|
| 1 | Bilanz, Position „Umlaufvermögen" | Die Position klappt auf und zeigt ihre Konten. |
| 2 | Konto 1800 | Das Kontoblatt öffnet sich mit allen Buchungen des Jahres. |
| 3 | Zeile B-2026-0052 | Die Buchung öffnet sich im Journal mit Soll und Haben. |
| 4 | „Beleg zur Buchung anzeigen" | Der versiegelte Beleg BE-2026-0229 erscheint mit seiner Datei. |

Jeder Schritt führt in die Richtung Beleg, keiner in ein Menü. Die Rückrichtung
funktioniert genauso: vom Beleg zur Buchung, von der Buchung ins Kontoblatt.

## Der Test

```bash
npm --prefix frontend install
npm --prefix scripts/site-screenshots install
task check:clicks          # oder: node scripts/site-screenshots/four-clicks.mjs
```

Der Test startet den Vite-Server mit der echten Oberfläche und der
Beispiel-Bridge, klickt den Weg und zählt. Er schlägt fehl, wenn ein Schritt
nicht klickbar ist oder mehr als vier Klicks nötig sind. Ändert sich die
Oberfläche so, dass der Weg länger wird, ist das ein Befund und keine
Anpassung der Grenze.

## Erwartetes Ergebnis

```
  1. Bilanzposition „Umlaufvermögen" aufklappen
  2. Konto 1800 öffnen
  3. Buchung B-2026-0052 öffnen
  4. Beleg zur Buchung B-2026-0052 öffnen

Bilanz → Konto → Buchung → Beleg in 4 Klicks (erlaubt: 4).
```
