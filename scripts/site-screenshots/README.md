# Screenshots der Projektseite

Nimmt die Bilder für [`website/`](../../website/) auf. Gefahren wird die echte
Oberfläche aus `frontend/` — dieselben Seiten, dieselben Bausteine, dieselben
Formatierer. Ersetzt ist allein `frontend/src/services/bridge.ts`, an dessen
Stelle `mock-bridge.ts` mit Beispieldaten tritt. Damit zeigen die Screenshots
die Anwendung und keinen Nachbau, der beim nächsten Umbau der Oberfläche still
veraltet.

## Aufrufen

```bash
npm --prefix frontend install            # einmalig
npm --prefix scripts/site-screenshots install
node scripts/site-screenshots/shoot.mjs        # die Bilder
node scripts/site-screenshots/four-clicks.mjs  # der Klickweg (task check:clicks)
node scripts/site-screenshots/help.mjs         # Tooltips, Dialoge, Quellenlinks
node scripts/site-screenshots/peek.mjs '#/books/journal' /tmp/journal.png  # ein Ziel ansehen
```

Das Skript startet den Vite-Server selbst, macht die Bilder und beendet ihn
wieder. Ergebnis: PNG in `website/assets/screenshots/`, 2880 × 1800 (1440 × 900
bei doppelter Pixeldichte), je Bereich der Navigation mit dessen Namen
vorn (`buecher-journal.png`). Eine Aufnahme mit `site` liegt zusätzlich unter
dem Namen, den die Projektseite einbindet (`buecher.png`); die sieben Namen
stehen in [`website/README.md`](../../website/README.md#screenshots).

Die Uhr des Browsers steht auf dem 25.08.2026, dem Stand der Beispieldaten.
Fälligkeiten und der laufende Monat passen so zu den Daten.

| Variable | Wirkung |
|---|---|
| `SHOTS=buecher,einstellungen` | nur diese Bereiche aus `shots/` |
| `SCREENSHOT_DIR=/tmp/bilder` | anderer Zielordner |
| `SCREENSHOT_PORT=9300` | anderer Port, damit zwei Arbeitskopien gleichzeitig aufnehmen |

## Was hier liegt

| Datei | Aufgabe |
|---|---|
| `shoot.mjs` | Klickt sich mit Playwright durch die Ansichten, schreibt die Bilder |
| `shots/*.mjs` | Die Aufnahmen je Bereich: Ziel, Wartebedingung, Dateiname, Name auf der Projektseite |
| `peek.mjs` | Ein Ziel öffnen und ein Bild ablegen, mit den Fehlern des Browsers |
| `four-clicks.mjs` | Der Klickweg des Prüfszenarios: Bilanz → Konto → Buchung → Beleg, mit Zähler |
| `dev-server.mjs` | Startet und beendet den Vite-Server; beide Werkzeuge nutzen ihn |
| `journal-mock.ts` | Das Journal des Beispieljahres mit Saldenvortrag und Belegen; daraus leiten sich Konten, Bank, offene Posten, Rechnungen, Voranmeldungen und Bilanz ab |
| `mock-bridge.ts` | Die Beispieldaten; tritt an die Stelle der Wails-Bridge |
| `closings-mock.ts`, `declarations.json` | Pflichtenkalender, Karteien und Jahreserklärungen der Abschlüsse |
| `demo-receipt.html` | Der Beispielbeleg, der als Bild in der Belegvorschau steht |
| `../../frontend/vite.screenshots.config.ts` | Vite ohne Wails-Plugin, mit dem Alias auf `mock-bridge.ts` |

## Beim Ändern beachten

- **Die Zahlen sind gerechnet, nicht gewürfelt.** Soll gleich Haben, 19 % auf
  das Entgelt, Zahllast gleich Umsatzsteuer minus Vorsteuer, Aktiva gleich
  Passiva plus Jahresergebnis, Summe der Sollsalden gleich Summe der
  Habensalden. Eine neue Buchung gehört in `journal-mock.ts`; Konten,
  Kontoblätter, Bank, Belege und Bilanz ziehen dann von selbst mit. Die
  Jahreserklärungen in `declarations.json` und die Steuervorschauen in
  `closings-mock.ts` sind von Hand nachzurechnen.
- **Buchungs- und Belegnummern vergibt das Journal nach Datum.** Eine neue
  Buchung verschiebt die Nummern danach; die Wartebedingungen in `shots/` und
  `four-clicks.mjs` nennen Nummern und sind dann mitzuziehen.
- **Die Uhr steht am 25.08.2026 um 10:12.** Kein Zeitstempel liegt danach.
- **Die Kontonummern stammen aus dem Projekt.** `internal/domain/skr04_accounts.go`
  und `internal/accounting/posting_groups.go` sind gegen die DATEV-Vorlage
  geprüft; neue Konten bitte von dort nehmen und nicht aus dem Gedächtnis.
- **Firmen und Personen sind erfunden.** Die IBANs sind Testnummern. Das soll so
  bleiben.
- **Die Browsersprache muss gesetzt sein.** Native Datumsfelder folgen der
  Locale des Browserprozesses, nicht der Seitensprache und auch nicht `--lang`.
  Ohne `LANG=de_DE.UTF-8` steht im Feld `08/10/2026` statt `10.08.2026`.
- **Neue Ansicht, neuer Eintrag.** Ein Screenshot besteht aus einem Eintrag in
  der Datei seines Bereichs unter `shots/` (Ziel, Wartebedingung, Dateiname)
  und den Daten, die die Ansicht dafür braucht. `open('#/books/journal')` setzt
  das Ziel wie ein Verweis in der Anwendung, `nav` und `tab` klicken Navigation
  und Reiter.
- **Neue Bridge-Methode, neue Beispielantwort.** Was eine Ansicht beim Öffnen
  ruft, muss `mock-bridge.ts` kennen; sonst bricht sie hier mit einem
  TypeError ab, den es in der Anwendung nicht gibt. Was der Screenshot-Lauf
  nicht braucht, steht als `unsupported` dabei — das ist eine Antwort und kein
  Loch. `mock-bridge.ts` deckt alle Methoden von
  `frontend/src/services/bridge.ts` ab: die lesenden mit Beispieldaten, die
  schreibenden als `unsupported`.

- **Ereignisse des Backends spielen die Aufnahmen nach.** Ein Ereignis wie
  `buchfink:eingang-abgelegt` schickt `window._wails.dispatchWailsEvent`, das die
  Wails-Laufzeit auch in der Vorschau anlegt; Dateien über dem Fenster sind
  DOM-Ereignisse mit einem `DataTransfer` (`shots/uebersicht.mjs`).

- **Einrichtung und Zustimmung über Schalter.** Steht `mock:setup` mit dem Wert
  `1` im lokalen Speicher, startet die Vorschau in der Einrichtung; `mock:consent`
  lässt Claude Code um Zustimmung bitten. `shots/einrichtung.mjs` setzt beide
  und lädt die Seite neu, deshalb läuft dieser Bereich zuletzt.
- **Die Datei liegt außerhalb der tsconfig.** `frontend/tsconfig.json` nimmt nur
  `src` und `bindings` auf; `npx tsc --noEmit` prüft `mock-bridge.ts` also
  nicht. Geprüft wird sie faktisch durch den Lauf selbst — bricht eine Ansicht
  ab, fehlt eine Methode oder ein Feld.

## Der Klickweg des Prüfszenarios

`four-clicks.mjs` misst, was GOB-02 als Bedienbarkeit versteht: von der
Bilanzposition, unter der das Bankkonto steht, bis zum Beleg der Buchung in
höchstens vier Klicks. Gezählt wird ab der Bilanz — der Weg dorthin ist
Navigation und nicht der Vorgang. Der Lauf schlägt fehl, wenn ein Schritt nicht
klickbar ist oder die Grenze überschritten wird; er ist damit auch die Probe
darauf, dass der Weg Buchung → Beleg im Journal überhaupt besteht.

Playwright braucht einen Chromium-Build. Ist keiner vorhanden, holt ihn
`npx playwright install chromium`.

`help.mjs` prüft das Popover am Fragezeichen bei Hover, Tastaturfokus und
Enter, die Trennung von Kurztext und Details, Quellenlinks, Escape,
Fokusrückgabe sowie eine Hilfe innerhalb eines geöffneten Dialogs.
