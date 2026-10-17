# Buchfink mit OpenCode verbinden

[Dokumentation](../../README.md) · [KI-Agenten](README.md)

Stand: 17. Oktober 2026.

Gilt ab Buchfink 0.3.0. Die Download-Links und die Befehle mit
`mxzinke/buchfink` funktionieren erst mit diesem Release.

OpenCode spricht Buchfink direkt über `http://127.0.0.1:7465/mcp` an und meldet
sich selbst an. Die Skills installieren Sie getrennt.

## Voraussetzungen

- Buchfink läuft auf demselben Rechner, und ein Unternehmen ist geöffnet.
- OpenCode mit der Konfiguration im Format von <https://opencode.ai/config.json>.
- Für den Skill-Befehl: Node.js. Ohne Node.js entpacken Sie die Skills von Hand.

## Einrichten

1. Öffnen Sie Buchfink und darin das Unternehmen, mit dem OpenCode arbeiten
   soll.
2. Ergänzen Sie die globale Konfiguration `~/.config/opencode/opencode.json`.
   Besteht die Datei schon, fügen Sie nur den Eintrag unter `mcp` hinzu.
   ```json
   {
     "$schema": "https://opencode.ai/config.json",
     "mcp": {
       "buchfink": {
         "type": "remote",
         "url": "http://127.0.0.1:7465/mcp",
         "enabled": true
       }
     }
   }
   ```
3. Melden Sie sich an und bestätigen Sie in Buchfink (siehe unten):
   ```sh
   opencode mcp auth buchfink
   ```
4. Installieren Sie die Skills nach `~/.agents/skills/`, wo OpenCode sie findet:
   ```sh
   npx skills add mxzinke/buchfink -g -a opencode --skill '*'
   ```
   Ohne Node.js: Laden Sie
   <https://github.com/mxzinke/buchfink/releases/latest/download/buchfink-skills.zip>
   und entpacken Sie es in denselben Ordner, unter macOS und Linux mit
   `mkdir -p ~/.agents/skills && unzip -o buchfink-skills.zip -d ~/.agents/skills`.
5. Prüfen Sie die Verbindung (siehe unten).

## Was beim ersten Verbinden passiert

`opencode mcp auth` öffnet den Browser. Die Seite zeigt „Bitte in Buchfink
bestätigen“ und einen Abgleichcode. Buchfink holt gleichzeitig sein Fenster nach
vorn: „Eine App möchte auf Buchfink zugreifen“, mit dem Namen der App, demselben
Abgleichcode, der Rücksprungadresse, dem geöffneten Mandanten und den Rechten
(Lesen, Buchen).

Klicken Sie auf **Zulassen**, wenn beide Codes übereinstimmen und Sie die
Anmeldung gerade selbst gestartet haben. Sonst klicken Sie auf **Ablehnen**. Die
Anfrage verfällt nach fünf Minuten. Die Zulassung gilt für den geöffneten
Mandanten. Für einen weiteren Mandanten öffnen Sie ihn in Buchfink und melden
sich erneut an. Entziehen können Sie den Zugriff jederzeit in Buchfink unter
Einstellungen → Verbundene Apps.

## Prüfen, ob es geht

```sh
opencode mcp list
```

Der Eintrag `buchfink` steht dort als verbunden und angemeldet. Bitten Sie
OpenCode dann: „Rufe das Buchfink-Werkzeug status auf.“ Die Antwort nennt das
geöffnete Unternehmen und `bereit: true`. In OpenCode heißen die Werkzeuge
`buchfink_status`, `buchfink_…`.

## Rechte für schreibende Werkzeuge einschränken

Soll OpenCode vor jedem Buchfink-Werkzeug fragen, ergänzen Sie in derselben
Datei:

```json
{
  "permission": {
    "buchfink_*": "ask"
  }
}
```

Mit `"deny"` statt `"ask"` sperren Sie Werkzeuge ganz. Buchfink sichert
Unumkehrbares zusätzlich ab: Festschreiben, Übermitteln, Aufstellen,
Feststellen, Offenlegen, Stammdaten und Prüfermodus gibt es nicht als Werkzeug.
Das erledigen Sie in Buchfink.

## Wenn es nicht geht

| Bild | Ursache | Abhilfe |
|---|---|---|
| `opencode mcp list` zeigt `buchfink` als nicht verbunden | Buchfink ist nicht geöffnet, oder der Zugang läuft auf einem anderen Port. | Buchfink starten. `opencode mcp debug buchfink` prüft Verbindung und Anmeldung. |
| Buchfink meldet unter Einstellungen → Verbundene Apps, Port 7465 sei belegt | Ein anderes Programm nutzt den Port. | Prüfen mit `lsof -nP -iTCP:7465 -sTCP:LISTEN` (macOS, Linux) oder `netstat -ano \| findstr :7465` (Windows). In Buchfink einen freien Port wählen, `url` in `opencode.json` anpassen, `opencode mcp logout buchfink` und neu anmelden. |
| Anmeldung abgelaufen oder in Buchfink entzogen | Die gespeicherte Anmeldung gilt nicht mehr. | `opencode mcp auth buchfink` wiederholen. |
| `status` antwortet mit `bereit: false` und „In Buchfink ist kein Mandant geöffnet.“; andere Werkzeuge melden `kein_mandant_offen` | In Buchfink ist kein Unternehmen geöffnet. | Unternehmen öffnen, `status` erneut aufrufen. |
| `status` antwortet mit „Diese App ist für den geöffneten Mandanten nicht freigegeben.“; andere Werkzeuge melden `mandant_nicht_freigegeben` | Die Zulassung gilt für ein anderes Unternehmen. | Anmeldung wiederholen, während das gewünschte Unternehmen geöffnet ist. |
| Skills erscheinen doppelt | Dieselben Skills liegen in `~/.agents/skills/` und `~/.claude/skills/`. OpenCode liest beide Ordner. | Eine der beiden Kopien entfernen. |

Ob Buchfink auf dem Port antwortet, zeigt
`curl -s http://127.0.0.1:7465/.well-known/oauth-protected-resource`. Buchfink
antwortet mit JSON, das `"resource"` enthält.

## Datenschutz

Buchfink läuft auf Ihrem Rechner. Was ein Werkzeug zurückgibt, schickt OpenCode
an den Anbieter des Sprachmodells, das Sie in OpenCode gewählt haben:
Buchungen, Beträge, Namen von Kunden, Lieferanten und Beschäftigten, Texte aus
Belegen und gelesene Belegdateien. Sie entscheiden, welche Daten Sie so
verarbeiten lassen, und tragen dafür die Verantwortung. Prüfen Sie vor dem
Einsatz mit echten Daten, ob Ihr Vertrag mit diesem Anbieter die Verarbeitung
personenbezogener Daten abdeckt, etwa durch eine Vereinbarung zur
Auftragsverarbeitung nach [Art. 28 DSGVO][dsgvo], und ob Ihre Eingaben zum
Training genutzt werden.

## Quellen

Abgerufen am 17. Oktober 2026.

- OpenCode, MCP-Server: <https://opencode.ai/docs/mcp-servers/> (`type: remote`,
  `opencode mcp auth|list|logout|debug`, Werkzeugnamen mit Serverpräfix)
- OpenCode, Konfiguration: <https://opencode.ai/docs/config/>
- OpenCode, Werkzeuge und Berechtigungen: <https://opencode.ai/docs/tools/>,
  <https://opencode.ai/docs/permissions/>
- OpenCode liest global `~/.config/opencode/skills/`, `~/.claude/skills/` und
  `~/.agents/skills/`: <https://opencode.ai/docs/skills/>
- skills-CLI: <https://github.com/vercel-labs/skills>. Verhalten mit 1.7.2
  geprüft: `-g -a opencode` installiert nach `~/.agents/skills/`. Die README des
  Tools nennt für OpenCode noch `~/.config/opencode/skills/`.

[dsgvo]: https://eur-lex.europa.eu/legal-content/DE/TXT/?uri=CELEX:32016R0679
