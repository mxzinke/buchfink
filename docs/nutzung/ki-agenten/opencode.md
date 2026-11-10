# Buchfink mit OpenCode verbinden

[Dokumentation](../../README.md) · [KI-Agenten](README.md)

OpenCode spricht Buchfink direkt über `http://127.0.0.1:7465/mcp` an und meldet
sich selbst an. Der Skill wird getrennt installiert.

## Voraussetzungen

- Buchfink läuft auf demselben Rechner, und ein Unternehmen ist geöffnet.
- OpenCode mit der Konfiguration im Format von <https://opencode.ai/config.json>.
- Für den Skill-Befehl: Node.js. Ohne Node.js entpacken Sie den Skill von Hand.

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
3. Melden Sie sich an und [lassen Sie die Anmeldung in Buchfink zu](README.md#anmeldung-zulassen):
   ```sh
   opencode mcp auth buchfink
   ```
4. Installieren Sie den [Skill](README.md#skills) nach `~/.agents/skills/`, wo
   OpenCode ihn findet:
   ```sh
   npx skills add mxzinke/buchfink -g -a opencode --skill '*'
   ```
   Ohne Node.js entpacken Sie `buchfink.zip` in denselben Ordner, unter macOS
   und Linux mit
   `mkdir -p ~/.agents/skills && unzip -o buchfink.zip -d ~/.agents/skills`.
5. Prüfen Sie die Verbindung (siehe unten).

## Prüfen, ob es geht

```sh
opencode mcp list
```

Der Eintrag `buchfink` steht dort als verbunden und angemeldet. Den Rest prüft
[`status`](README.md#verbindung-prüfen). In OpenCode heißen die Werkzeuge
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

Mit `"deny"` statt `"ask"` sperren Sie Werkzeuge ganz. Was Buchfink selbst
abfängt, steht unter [Rechte einschränken](README.md#rechte-einschränken).

## Wenn es nicht geht

Zusätzlich zu den [allgemeinen Fällen](README.md#wenn-es-nicht-geht):

| Bild | Ursache | Abhilfe |
|---|---|---|
| `opencode mcp list` zeigt `buchfink` als nicht verbunden | Buchfink ist nicht geöffnet. | Buchfink starten. `opencode mcp debug buchfink` prüft Verbindung und Anmeldung. |
| Port gewechselt | `url` zeigt auf den alten Port. | `url` in `opencode.json` anpassen, `opencode mcp logout buchfink` und neu anmelden. |
| Anmeldung abgelaufen oder entzogen | Die gespeicherte Anmeldung gilt nicht mehr. | `opencode mcp auth buchfink` wiederholen. |
| Skills erscheinen doppelt | Dieselben Skills liegen in `~/.agents/skills/` und `~/.claude/skills/`. OpenCode liest beide Ordner. | Eine der beiden Kopien entfernen. |

## Datenschutz

OpenCode schickt Werkzeugergebnisse an den Anbieter des Sprachmodells, das Sie
in OpenCode gewählt haben. Prüfen Sie, ob Ihr Vertrag mit ihm eine
Auftragsverarbeitung nach [Art. 28 DSGVO][dsgvo] abdeckt, und siehe
[Datenschutz](README.md#datenschutz).

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
