# Buchfink mit Pi verbinden

[Dokumentation](../../README.md) · [KI-Agenten](README.md)

Pi spricht Buchfink direkt über `http://127.0.0.1:7465/mcp` an und meldet sich
selbst an. Der Skill wird getrennt installiert.

## Voraussetzungen

- Buchfink läuft auf demselben Rechner, und ein Unternehmen ist geöffnet.
- Pi ab Version 0.99.0. Ältere Fassungen haben keine eingebaute
  MCP-Unterstützung.
- Für den Skill-Befehl: Node.js. Ohne Node.js entpacken Sie den Skill von Hand.

## Einrichten

1. Öffnen Sie Buchfink und darin das Unternehmen, mit dem Pi arbeiten soll.
2. Tragen Sie den Server ein. `--exposure direct` gibt dem Modell die
   Buchfink-Werkzeuge direkt in die Hand. Ohne die Angabe erreicht Pi sie nur
   über Skripte (`codemode`).
   ```sh
   pi mcp add buchfink --url http://127.0.0.1:7465/mcp --exposure direct --description "Buchfink: lokale Buchhaltung für GmbH und UG"
   ```
3. Melden Sie sich an und [lassen Sie die Anmeldung in Buchfink zu](README.md#anmeldung-zulassen):
   ```sh
   pi mcp login buchfink
   ```
4. Installieren Sie den [Skill](README.md#skills) nach `~/.agents/skills/`:
   ```sh
   npx skills add mxzinke/buchfink -g -a pi --skill '*'
   ```
   Ohne Node.js entpacken Sie `buchfink.zip` nach `~/.pi/agent/skills/`, unter
   macOS und Linux mit
   `mkdir -p ~/.pi/agent/skills && unzip -o buchfink.zip -d ~/.pi/agent/skills`.
   In einer laufenden Sitzung lädt `/reload` den Skill nach.
5. Prüfen Sie die Verbindung (siehe unten).

## Prüfen, ob es geht

```sh
pi mcp list
```

Der Befehl verbindet sich mit jedem Server und listet Zustand und Werkzeuge. Er
endet mit Fehlercode 1, wenn `buchfink` nicht verbunden ist. In einer Sitzung
zeigt `/mcp` dasselbe. Den Rest prüft [`status`](README.md#verbindung-prüfen).
In Pi heißen die Werkzeuge `mcp__buchfink__status`, `mcp__buchfink__…`.

## Rechte für schreibende Werkzeuge einschränken

Pi fragt von sich aus nicht vor Werkzeugaufrufen. Eine kleine Erweiterung holt
Ihre Bestätigung vor jedem Buchfink-Werkzeug ein, das Buchfink nicht als lesend
kennzeichnet. Legen Sie dazu `~/.pi/agent/extensions/buchfink-freigabe.ts` an:

```typescript
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

export default function (pi: ExtensionAPI) {
  pi.on("tool_call", async (event, ctx) => {
    if (!event.toolName.startsWith("mcp__buchfink__")) return;
    const hints = pi.getAllTools().find((tool) => tool.name === event.toolName)?.annotations;
    if (hints?.readOnlyHint) return;
    if (!(await ctx.ui.confirm("Buchfink-Werkzeug ausführen?", event.toolName))) {
      return { block: true, reason: `${event.toolName} wurde nicht freigegeben` };
    }
  });
}
```

Einzelne Werkzeuge blenden Sie in `~/.pi/agent/mcp.json` mit
`"toolExposure": { "<werkzeug>": "hidden" }` aus. Was Buchfink selbst abfängt,
steht unter [Rechte einschränken](README.md#rechte-einschränken).

## Wenn es nicht geht

Zusätzlich zu den [allgemeinen Fällen](README.md#wenn-es-nicht-geht):

| Bild | Ursache | Abhilfe |
|---|---|---|
| `pi mcp list` meldet `buchfink` als nicht verbunden | Buchfink ist nicht geöffnet. | Buchfink starten, in der Sitzung über `/mcp` neu verbinden. |
| Port gewechselt | Der Eintrag zeigt auf den alten Port. | `pi mcp remove buchfink`, Schritt 2 mit der neuen Adresse wiederholen, neu anmelden. |
| `/mcp` zeigt, dass eine Anmeldung nötig ist | Anmeldung abgelaufen oder entzogen. | `pi mcp login buchfink` wiederholen. |
| Das Modell findet die Buchfink-Werkzeuge nicht | Der Server steht auf `codemode`. | In `/mcp` die Freigabe auf `direct` stellen. |

## Datenschutz

Pi schickt Werkzeugergebnisse an den Anbieter des Sprachmodells, das Sie in Pi
gewählt haben. Prüfen Sie, ob Ihr Vertrag mit ihm eine Auftragsverarbeitung nach
[Art. 28 DSGVO][dsgvo] abdeckt, und siehe [Datenschutz](README.md#datenschutz).

## Quellen

Abgerufen am 17. Oktober 2026.

- Pi, MCP-Server: <https://pi.dev/docs/latest/mcp> (Exposure, Werkzeugnamen,
  `pi mcp login|list`, `/mcp`, Berechtigungen über Erweiterungen)
- Pi, Befehlszeile: <https://pi.dev/docs/latest/cli> (`pi mcp add --url
  --exposure --description`, `pi mcp remove`)
- Pi, Skills und Konfiguration: <https://pi.dev/docs/latest/skills>,
  <https://github.com/badlogic/pi-mono/blob/main/packages/coding-agent/docs/configuration.md>
  (`~/.pi/agent/skills/`, `~/.agents/skills/`)
- Pi, Erweiterungen: <https://github.com/badlogic/pi-mono/blob/main/packages/coding-agent/docs/extensions.md>
  (Bestätigung anhand der Werkzeug-Annotations)
- Pi 0.99.0, MCP eingebaut: <https://pi.dev/changelog/releases/0.99.0>
- skills-CLI: <https://github.com/vercel-labs/skills> (geprüft mit 1.7.2)

[dsgvo]: https://eur-lex.europa.eu/legal-content/DE/TXT/?uri=CELEX:32016R0679
