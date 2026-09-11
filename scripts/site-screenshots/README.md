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
```

Das Skript startet den Vite-Server selbst, macht die Bilder und beendet ihn
wieder. Ergebnis: zwanzig PNG in `website/assets/screenshots/`, 2880 × 1800
(1440 × 900 bei doppelter Pixeldichte).

## Was hier liegt

| Datei | Aufgabe |
|---|---|
| `shoot.mjs` | Klickt sich mit Playwright durch die Ansichten, schreibt die Bilder |
| `four-clicks.mjs` | Der Klickweg des Prüfszenarios: Bilanz → Konto → Buchung → Beleg, mit Zähler |
| `dev-server.mjs` | Startet und beendet den Vite-Server; beide Werkzeuge nutzen ihn |
| `mock-bridge.ts` | Die Beispieldaten; tritt an die Stelle der Wails-Bridge |
| `demo-receipt.html` | Der Beispielbeleg, der als Bild in der Belegvorschau steht |
| `../../frontend/vite.screenshots.config.ts` | Vite ohne Wails-Plugin, mit dem Alias auf `mock-bridge.ts` |

## Beim Ändern beachten

- **Die Zahlen sind gerechnet, nicht gewürfelt.** Soll gleich Haben, 19 % auf
  das Entgelt, Zahllast gleich Umsatzsteuer minus Vorsteuer, Aktiva gleich
  Passiva plus Jahresergebnis, Summe der Sollsalden gleich Summe der
  Habensalden. Wer eine Zahl ändert, zieht die verbundenen mit.
- **Die Kontonummern stammen aus dem Projekt.** `internal/domain/skr04_accounts.go`
  und `internal/accounting/posting_groups.go` sind gegen die DATEV-Vorlage
  geprüft; neue Konten bitte von dort nehmen und nicht aus dem Gedächtnis.
- **Firmen und Personen sind erfunden.** Die IBANs sind Testnummern. Das soll so
  bleiben.
- **Die Browsersprache muss gesetzt sein.** Native Datumsfelder folgen der
  Locale des Browserprozesses, nicht der Seitensprache und auch nicht `--lang`.
  Ohne `LANG=de_DE.UTF-8` steht im Feld `08/10/2026` statt `10.08.2026`.
- **Neue Ansicht, neuer Eintrag.** Ein Screenshot besteht aus einem Eintrag in
  `shots` (Navigation, Wartebedingung, Dateiname) und den Daten, die die Ansicht
  dafür braucht.
- **Neue Bridge-Methode, neue Beispielantwort.** Was eine Ansicht beim Öffnen
  ruft, muss `mock-bridge.ts` kennen; sonst bricht sie hier mit einem
  TypeError ab, den es in der Anwendung nicht gibt. Was der Screenshot-Lauf
  nicht braucht, steht als `unsupported` dabei — das ist eine Antwort und kein
  Loch. `mock-bridge.ts` deckt seit Welle 7 alle Methoden von
  `frontend/src/services/bridge.ts` ab: die lesenden mit Beispieldaten, die
  schreibenden als `unsupported`.

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

`help.mjs` prüft Hover und Tastaturfokus am Fragezeichen, die Trennung von
Kurztext und Details, Quellenlinks, Escape, Fokusrückgabe sowie eine Hilfe
innerhalb eines geöffneten Dialogs.
