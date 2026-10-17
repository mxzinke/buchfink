# Buchfink mit Codex verbinden

[Dokumentation](../../README.md) · [KI-Agenten](README.md)

Stand: 17. Oktober 2026.

Gilt ab Buchfink 0.3.0. Die Download-Links und die Befehle mit
`mxzinke/buchfink` funktionieren erst mit diesem Release.

Codex spricht Buchfink direkt über `http://127.0.0.1:7465/mcp` an und meldet
sich selbst an. Die Skills installieren Sie getrennt.

## Voraussetzungen

- Buchfink läuft auf demselben Rechner, und ein Unternehmen ist geöffnet.
- Codex CLI. Die Konfiguration liegt in `~/.codex/config.toml`.
- Für den Skill-Befehl: Node.js. Ohne Node.js entpacken Sie die Skills von Hand.

## Einrichten

1. Öffnen Sie Buchfink und darin das Unternehmen, mit dem Codex arbeiten soll.
2. Tragen Sie den Server ein:
   ```sh
   codex mcp add buchfink --url http://127.0.0.1:7465/mcp
   ```
3. Melden Sie sich an und bestätigen Sie in Buchfink (siehe unten):
   ```sh
   codex mcp login buchfink
   ```
4. Installieren Sie die Skills nach `~/.agents/skills/`:
   ```sh
   npx skills add mxzinke/buchfink -g -a codex --skill '*'
   ```
   Ohne Node.js: Laden Sie
   <https://github.com/mxzinke/buchfink/releases/latest/download/buchfink-skills.zip>
   und entpacken Sie es nach `~/.agents/skills/`, unter macOS und Linux mit
   `mkdir -p ~/.agents/skills && unzip -o buchfink-skills.zip -d ~/.agents/skills`,
   unter Windows in PowerShell mit
   `Expand-Archive buchfink-skills.zip -DestinationPath "$HOME\.agents\skills" -Force`.
   Erscheinen die Skills nicht, starten Sie Codex neu.
5. Prüfen Sie die Verbindung (siehe unten).

## Was beim ersten Verbinden passiert

`codex mcp login` öffnet den Browser. Die Seite zeigt „Bitte in Buchfink
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
codex mcp list
```

`buchfink` steht dort mit seiner Adresse. In einer Codex-Sitzung zeigt `/mcp`
die verbundenen Server. Bitten Sie Codex dann: „Rufe das Buchfink-Werkzeug
status auf.“ Die Antwort nennt das geöffnete Unternehmen und `bereit: true`.

## Rechte für schreibende Werkzeuge einschränken

Ergänzen Sie in `~/.codex/config.toml` unter dem Eintrag, den Schritt 2
angelegt hat:

```toml
[mcp_servers.buchfink]
url = "http://127.0.0.1:7465/mcp"
default_tools_approval_mode = "writes"
```

Mit `writes` fragt Codex vor jedem Werkzeug, das Buchfink nicht als lesend
kennzeichnet. Einzelne Werkzeuge sperren Sie mit `disabled_tools = ["…"]`.
Festschreiben, Übermitteln, Aufstellen, Feststellen, Offenlegen, Stammdaten und
Prüfermodus gibt es nicht als Werkzeug. Das erledigen Sie in Buchfink.

## Wenn es nicht geht

| Bild | Ursache | Abhilfe |
|---|---|---|
| Codex meldet beim Start, `buchfink` sei nicht erreichbar | Buchfink ist nicht geöffnet, oder der Zugang läuft auf einem anderen Port. | Buchfink starten, Codex neu starten. |
| Buchfink meldet unter Einstellungen → Verbundene Apps, Port 7465 sei belegt | Ein anderes Programm nutzt den Port. | Prüfen mit `lsof -nP -iTCP:7465 -sTCP:LISTEN` (macOS, Linux) oder `netstat -ano \| findstr :7465` (Windows). In Buchfink einen freien Port wählen, `url` in `~/.codex/config.toml` anpassen, `codex mcp login buchfink` wiederholen. |
| Anmeldung abgelaufen oder in Buchfink entzogen | Die gespeicherte Anmeldung gilt nicht mehr. | `codex mcp login buchfink` wiederholen. |
| `status` antwortet mit `bereit: false` und „In Buchfink ist kein Mandant geöffnet.“; andere Werkzeuge melden `kein_mandant_offen` | In Buchfink ist kein Unternehmen geöffnet. | Unternehmen öffnen, `status` erneut aufrufen. |
| `status` antwortet mit „Diese App ist für den geöffneten Mandanten nicht freigegeben.“; andere Werkzeuge melden `mandant_nicht_freigegeben` | Die Zulassung gilt für ein anderes Unternehmen. | Anmeldung wiederholen, während das gewünschte Unternehmen geöffnet ist. |

Ob Buchfink auf dem Port antwortet, zeigt
`curl -s http://127.0.0.1:7465/.well-known/oauth-protected-resource`. Buchfink
antwortet mit JSON, das `"resource"` enthält.

## Datenschutz

Buchfink läuft auf Ihrem Rechner. Was ein Werkzeug zurückgibt, schickt Codex an
OpenAI: Buchungen, Beträge, Namen von Kunden, Lieferanten und Beschäftigten,
Texte aus Belegen und gelesene Belegdateien. Sie entscheiden, welche Daten Sie
so verarbeiten lassen, und tragen dafür die Verantwortung. Prüfen Sie vor dem
Einsatz mit echten Daten, ob Ihr Vertrag mit OpenAI die Verarbeitung
personenbezogener Daten abdeckt, etwa durch eine Vereinbarung zur
Auftragsverarbeitung nach [Art. 28 DSGVO][dsgvo], und ob Ihre Eingaben zum
Training genutzt werden.

## Quellen

Abgerufen am 17. Oktober 2026.

- Codex, MCP: <https://learn.chatgpt.com/docs/extend/mcp?surface=cli>
  (`codex mcp add --url`, `codex mcp login`, `codex mcp list`, `/mcp`,
  `default_tools_approval_mode`, `disabled_tools`)
- Codex, Skills: <https://learn.chatgpt.com/docs/build-skills> (`~/.agents/skills`,
  Symlinks, Neustart)
- skills-CLI: <https://github.com/vercel-labs/skills> (geprüft mit 1.7.2)

[dsgvo]: https://eur-lex.europa.eu/legal-content/DE/TXT/?uri=CELEX:32016R0679
