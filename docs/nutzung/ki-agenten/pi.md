# Buchfink mit Pi verbinden

[Dokumentation](../../README.md) · [KI-Agenten](README.md)

Stand: 17. Oktober 2026.

Gilt ab Buchfink 0.3.0. Die Download-Links und die Befehle mit
`mxzinke/buchfink` funktionieren erst mit diesem Release.

Pi spricht Buchfink direkt über `http://127.0.0.1:7465/mcp` an und meldet sich
selbst an. Die Skills installieren Sie getrennt.

## Voraussetzungen

- Buchfink läuft auf demselben Rechner, und ein Unternehmen ist geöffnet.
- Pi ab Version 0.99.0. Ältere Fassungen haben keine eingebaute
  MCP-Unterstützung.
- Für den Skill-Befehl: Node.js. Ohne Node.js entpacken Sie die Skills von Hand.

## Einrichten

1. Öffnen Sie Buchfink und darin das Unternehmen, mit dem Pi arbeiten soll.
2. Tragen Sie den Server ein. `--exposure direct` gibt dem Modell die
   Buchfink-Werkzeuge direkt in die Hand. Ohne die Angabe erreicht Pi sie nur
   über Skripte (`codemode`).
   ```sh
   pi mcp add buchfink --url http://127.0.0.1:7465/mcp --exposure direct --description "Buchfink: lokale Buchhaltung für GmbH und UG"
   ```
3. Melden Sie sich an und bestätigen Sie in Buchfink (siehe unten):
   ```sh
   pi mcp login buchfink
   ```
4. Installieren Sie die Skills nach `~/.agents/skills/`:
   ```sh
   npx skills add mxzinke/buchfink -g -a pi --skill '*'
   ```
   Ohne Node.js: Laden Sie
   <https://github.com/mxzinke/buchfink/releases/latest/download/buchfink.zip>
   und entpacken Sie es nach `~/.pi/agent/skills/`, unter macOS und Linux mit
   `mkdir -p ~/.pi/agent/skills && unzip -o buchfink.zip -d ~/.pi/agent/skills`.
   In einer laufenden Sitzung lädt `/reload` die Skills nach.
5. Prüfen Sie die Verbindung (siehe unten).

## Was beim ersten Verbinden passiert

`pi mcp login` öffnet den Browser. Die Seite zeigt „Bitte in Buchfink
bestätigen“ und einen Abgleichcode. Buchfink holt gleichzeitig sein Fenster nach
vorn: „Eine App möchte auf Buchfink zugreifen“, mit dem Namen der App, demselben
Abgleichcode, der Rücksprungadresse, dem geöffneten Mandanten und den Rechten
(Lesen, Buchen).

Klicken Sie auf **Zulassen**, wenn beide Codes übereinstimmen und Sie die
Anmeldung gerade selbst gestartet haben. Sonst klicken Sie auf **Ablehnen**. Die
Anfrage verfällt nach fünf Minuten. Die Zulassung gilt für den geöffneten
Mandanten. Für einen weiteren Mandanten öffnen Sie ihn in Buchfink und melden
sich erneut an. Entziehen können Sie den Zugriff jederzeit in Buchfink unter
Einstellungen → Apps verbinden.

## Prüfen, ob es geht

```sh
pi mcp list
```

Der Befehl verbindet sich mit jedem Server und listet Zustand und Werkzeuge. Er
endet mit Fehlercode 1, wenn `buchfink` nicht verbunden ist. In einer Sitzung
zeigt `/mcp` dasselbe. Bitten Sie Pi dann: „Rufe das Buchfink-Werkzeug status
auf.“ Die Antwort nennt das geöffnete Unternehmen und `bereit: true`. In Pi
heißen die Werkzeuge `mcp__buchfink__status`, `mcp__buchfink__…`.

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
`"toolExposure": { "<werkzeug>": "hidden" }` aus. Festschreiben, Übermitteln,
Aufstellen, Feststellen, Offenlegen, Stammdaten und Prüfermodus gibt es nicht
als Werkzeug. Das erledigen Sie in Buchfink.

## Wenn es nicht geht

| Bild | Ursache | Abhilfe |
|---|---|---|
| `pi mcp list` meldet `buchfink` als nicht verbunden | Buchfink ist nicht geöffnet, oder der Zugang läuft auf einem anderen Port. | Buchfink starten. In der Sitzung über `/mcp` neu verbinden. |
| Buchfink meldet unter Einstellungen → Apps verbinden, Port 7465 sei belegt | Ein anderes Programm nutzt den Port. | Prüfen mit `lsof -nP -iTCP:7465 -sTCP:LISTEN` (macOS, Linux) oder `netstat -ano \| findstr :7465` (Windows). In Buchfink einen freien Port wählen, `pi mcp remove buchfink`, Schritt 2 mit der neuen Adresse wiederholen, neu anmelden. |
| `/mcp` zeigt, dass eine Anmeldung nötig ist | Anmeldung abgelaufen oder in Buchfink entzogen. | `pi mcp login buchfink` wiederholen. |
| Das Modell findet die Buchfink-Werkzeuge nicht | Der Server steht auf `codemode`. | In `/mcp` die Freigabe auf `direct` stellen. |
| `status` antwortet mit `bereit: false` und „In Buchfink ist kein Mandant geöffnet.“; andere Werkzeuge melden `kein_mandant_offen` | In Buchfink ist kein Unternehmen geöffnet. | Unternehmen öffnen, `status` erneut aufrufen. |
| `status` antwortet mit „Diese App ist für den geöffneten Mandanten nicht freigegeben.“; andere Werkzeuge melden `mandant_nicht_freigegeben` | Die Zulassung gilt für ein anderes Unternehmen. | Anmeldung wiederholen, während das gewünschte Unternehmen geöffnet ist. |
| Neben `buchfink` erscheinen ältere Buchfink-Skills wie `buchfink-grundlagen` oder `buchfink-belege` | Die früheren elf Skills sind noch installiert; beide Fassungen stehen nebeneinander. | In `~/.agents/skills/` und `~/.pi/agent/skills/` alle Ordner entfernen, deren Name mit `buchfink-` beginnt; `buchfink` bleibt. In der Sitzung `/reload`. |

Ob Buchfink auf dem Port antwortet, zeigt
`curl -s http://127.0.0.1:7465/.well-known/oauth-protected-resource`. Buchfink
antwortet mit JSON, das `"resource"` enthält.

## Datenschutz

Buchfink läuft auf Ihrem Rechner. Was ein Werkzeug zurückgibt, schickt Pi an
den Anbieter des Sprachmodells, das Sie in Pi gewählt haben: Buchungen,
Beträge, Namen von Kunden, Lieferanten und Beschäftigten, Texte aus Belegen und
gelesene Belegdateien. Sie entscheiden, welche Daten Sie so verarbeiten lassen,
und tragen dafür die Verantwortung. Prüfen Sie vor dem Einsatz mit echten
Daten, ob Ihr Vertrag mit diesem Anbieter die Verarbeitung personenbezogener
Daten abdeckt, etwa durch eine Vereinbarung zur Auftragsverarbeitung nach
[Art. 28 DSGVO][dsgvo], und ob Ihre Eingaben zum Training genutzt werden.

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
