# Buchfink mit Codex verbinden

[Dokumentation](../../README.md) · [KI-Agenten](README.md)

Codex spricht Buchfink direkt über `http://127.0.0.1:7465/mcp` an und meldet
sich selbst an. Der Skill wird getrennt installiert.

## Voraussetzungen

- Buchfink läuft auf demselben Rechner, und ein Unternehmen ist geöffnet.
- Codex CLI. Die Konfiguration liegt in `~/.codex/config.toml`.
- Für den Skill-Befehl: Node.js. Ohne Node.js entpacken Sie den Skill von Hand.

## Einrichten

1. Öffnen Sie Buchfink und darin das Unternehmen, mit dem Codex arbeiten soll.
2. Tragen Sie den Server ein:
   ```sh
   codex mcp add buchfink --url http://127.0.0.1:7465/mcp
   ```
3. Melden Sie sich an und [lassen Sie die Anmeldung in Buchfink zu](README.md#anmeldung-zulassen):
   ```sh
   codex mcp login buchfink
   ```
4. Installieren Sie den [Skill](README.md#skills) nach `~/.agents/skills/`:
   ```sh
   npx skills add mxzinke/buchfink -g -a codex --skill '*'
   ```
   Ohne Node.js entpacken Sie `buchfink.zip` nach `~/.agents/skills/`, unter
   macOS und Linux mit
   `mkdir -p ~/.agents/skills && unzip -o buchfink.zip -d ~/.agents/skills`,
   unter Windows in PowerShell mit
   `Expand-Archive buchfink.zip -DestinationPath "$HOME\.agents\skills" -Force`.
   Erscheint der Skill nicht, starten Sie Codex neu.
5. Prüfen Sie die Verbindung (siehe unten).

## Prüfen, ob es geht

```sh
codex mcp list
```

`buchfink` steht dort mit seiner Adresse. In einer Sitzung zeigt `/mcp` die
verbundenen Server. Den Rest prüft [`status`](README.md#verbindung-prüfen).

## Rechte für schreibende Werkzeuge einschränken

Ergänzen Sie in `~/.codex/config.toml` unter dem Eintrag aus Schritt 2:

```toml
[mcp_servers.buchfink]
url = "http://127.0.0.1:7465/mcp"
default_tools_approval_mode = "writes"
```

Mit `writes` fragt Codex vor jedem Werkzeug, das Buchfink nicht als lesend
kennzeichnet. Einzelne Werkzeuge sperren Sie mit `disabled_tools = ["…"]`. Was
Buchfink selbst abfängt, steht unter
[Rechte einschränken](README.md#rechte-einschränken).

## Wenn es nicht geht

Zusätzlich zu den [allgemeinen Fällen](README.md#wenn-es-nicht-geht):

| Bild | Ursache | Abhilfe |
|---|---|---|
| Codex meldet beim Start, `buchfink` sei nicht erreichbar | Buchfink ist nicht geöffnet. | Buchfink starten, Codex neu starten. |
| Port gewechselt | `url` zeigt auf den alten Port. | `url` in `~/.codex/config.toml` anpassen, `codex mcp login buchfink` wiederholen. |
| Anmeldung abgelaufen oder entzogen | Die gespeicherte Anmeldung gilt nicht mehr. | `codex mcp login buchfink` wiederholen. |

## Datenschutz

Codex schickt Werkzeugergebnisse an OpenAI. Prüfen Sie, ob Ihr Vertrag mit
OpenAI eine Auftragsverarbeitung nach [Art. 28 DSGVO][dsgvo] abdeckt, und siehe
[Datenschutz](README.md#datenschutz).

## Quellen

Abgerufen am 17. Oktober 2026.

- Codex, MCP: <https://learn.chatgpt.com/docs/extend/mcp?surface=cli>
  (`codex mcp add --url`, `codex mcp login`, `codex mcp list`, `/mcp`,
  `default_tools_approval_mode`, `disabled_tools`)
- Codex, Skills: <https://learn.chatgpt.com/docs/build-skills> (`~/.agents/skills`,
  Symlinks, Neustart)
- skills-CLI: <https://github.com/vercel-labs/skills> (geprüft mit 1.7.2)

[dsgvo]: https://eur-lex.europa.eu/legal-content/DE/TXT/?uri=CELEX:32016R0679
