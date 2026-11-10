# Buchfink mit Claude Code verbinden

[Dokumentation](../../README.md) · [KI-Agenten](README.md)

Geprüft mit Claude Code 2.1.302.

Claude Code spricht Buchfink direkt über `http://127.0.0.1:7465/mcp` an. Das
Plugin `buchfink-buchhaltung` richtet die Verbindung ein und bringt den Skill
mit. Wer kein Plugin möchte, trägt nur den Server ein und installiert den Skill
getrennt.

## Voraussetzungen

- Buchfink läuft auf demselben Rechner, und ein Unternehmen ist geöffnet.
- Claude Code im Terminal, in der Desktop-App (Bereich Code) oder in VS Code.

## Einrichten mit dem Plugin

1. Öffnen Sie Buchfink und darin das Unternehmen, mit dem Claude arbeiten soll.
2. Fügen Sie in Claude Code den Marketplace hinzu und installieren Sie das
   Plugin. Wählen Sie dabei „Install for you (user scope)“.
   ```text
   /plugin marketplace add mxzinke/buchfink
   /plugin install buchfink-buchhaltung@buchfink
   ```
   Aus der Shell geht beides in einem Befehl (ab Claude Code 2.1.292):
   ```sh
   claude plugin install buchfink-buchhaltung --marketplace mxzinke/buchfink
   ```
3. Melden Sie sich an: `/mcp` öffnen, den Server
   `plugin:buchfink-buchhaltung:buchfink` wählen und die Anmeldung starten. Aus
   der Shell: `claude mcp login "plugin:buchfink-buchhaltung:buchfink"`.
4. [Lassen Sie die Anmeldung in Buchfink zu](README.md#anmeldung-zulassen).
5. Prüfen Sie die Verbindung (siehe unten). Der Skill erscheint unter `/` als
   `/buchfink-buchhaltung:buchfink`.

In der Desktop-App erreichen Sie dieselbe Installation über **+ → Plugins →
Add plugin**, nachdem der Marketplace hinzugefügt ist.

Claude Code aktualisiert Plugins aus fremden Marketplaces nicht von selbst.
Aktualisieren Sie mit `claude plugin update buchfink-buchhaltung@buchfink` oder
schalten Sie unter `/plugin` → Marketplaces → buchfink „Enable auto-update“ ein.

## Einrichten ohne Plugin

1. Öffnen Sie Buchfink und das Unternehmen.
2. Tragen Sie den Server für alle Projekte ein:
   ```sh
   claude mcp add --transport http --scope user buchfink http://127.0.0.1:7465/mcp
   ```
3. Melden Sie sich an (`claude mcp login buchfink` oder `/mcp` in einer
   Sitzung) und [lassen Sie die Anmeldung in Buchfink zu](README.md#anmeldung-zulassen).
4. Installieren Sie den [Skill](README.md#skills) mit `-a claude-code` nach
   `~/.claude/skills/`:
   ```sh
   npx skills add mxzinke/buchfink -g -a claude-code --skill '*'
   ```

Verwenden Sie nur einen der beiden Wege. Mit beiden sieht Claude jedes
Buchfink-Werkzeug doppelt.

## Prüfen, ob es geht

```sh
claude mcp list
```

Der Buchfink-Server steht dort mit „Connected“. Den Rest prüft
[`status`](README.md#verbindung-prüfen).

## Rechte für schreibende Werkzeuge einschränken

Claude Code fragt vor einem Buchfink-Werkzeug, bis Sie „Yes, and don't ask
again“ wählen. Soll es vor jedem fragen, tragen Sie in `~/.claude/settings.json`
ein:

```json
{
  "permissions": {
    "ask": ["mcp__plugin_buchfink-buchhaltung_buchfink__*"]
  }
}
```

Ohne Plugin heißt die Regel `mcp__buchfink__*`. Erlaubnisse sehen und entfernen
Sie mit `/permissions`. Was Buchfink selbst abfängt, steht unter
[Rechte einschränken](README.md#rechte-einschränken).

## Wenn es nicht geht

Zusätzlich zu den [allgemeinen Fällen](README.md#wenn-es-nicht-geht):

| Bild | Ursache | Abhilfe |
|---|---|---|
| `claude mcp list` zeigt „Failed to connect“ mit `ECONNREFUSED` | Buchfink ist nicht geöffnet. | Buchfink starten, in `/mcp` den Server neu verbinden. |
| Port gewechselt | Das Plugin trägt Port 7465 ein. | Plugin-Server in `/mcp` abschalten, den Server ohne Plugin mit der neuen Adresse eintragen, neu anmelden. |
| „Needs authentication“ | Anmeldung abgelaufen oder in Buchfink entzogen. | Erneut anmelden (Schritt 3). |
| Die Skills fehlen unter `/` | Plugin nicht geladen. | `/reload-plugins` ausführen, Fehler unter `/plugin` → Errors lesen. |

## Datenschutz

Claude Code schickt Werkzeugergebnisse an Anthropic. Prüfen Sie, ob Ihr Vertrag
mit Anthropic eine Auftragsverarbeitung nach [Art. 28 DSGVO][dsgvo] abdeckt, und
siehe [Datenschutz](README.md#datenschutz).

## Quellen

Abgerufen am 17. Oktober 2026.

- Claude Code, MCP: <https://code.claude.com/docs/en/mcp> (`claude mcp add`,
  `claude mcp login`, `claude mcp list`, Plugin-Servernamen)
- Claude Code, Plugins installieren: <https://code.claude.com/docs/en/plugins/install>
- Claude Code, Berechtigungen: <https://code.claude.com/docs/en/permissions>
- Claude Code, Laden und Updates von Plugins: <https://code.claude.com/docs/en/plugins/loading>
- skills-CLI: <https://github.com/vercel-labs/skills> (geprüft mit 1.7.2)

[dsgvo]: https://eur-lex.europa.eu/legal-content/DE/TXT/?uri=CELEX:32016R0679
