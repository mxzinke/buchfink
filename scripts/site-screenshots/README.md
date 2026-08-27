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
node scripts/site-screenshots/shoot.mjs
```

Das Skript startet den Vite-Server selbst, macht die Bilder und beendet ihn
wieder. Ergebnis: zehn PNG in `website/assets/screenshots/`, 2880 × 1800
(1440 × 900 bei doppelter Pixeldichte).

## Was hier liegt

| Datei | Aufgabe |
|---|---|
| `shoot.mjs` | Startet Vite, klickt sich mit Playwright durch die Ansichten, schreibt die Bilder |
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

Playwright braucht einen Chromium-Build. Ist keiner vorhanden, holt ihn
`npx playwright install chromium`.
