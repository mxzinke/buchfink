# Buchfink mit Claude Code verbinden

[Dokumentation](../../README.md) · [KI-Agenten](README.md)

Stand: 17. Oktober 2026, geprüft mit Claude Code 2.1.302.

Gilt ab Buchfink 0.3.0. Die Befehle mit `mxzinke/buchfink` funktionieren erst
mit diesem Release.

Claude Code spricht Buchfink direkt über `http://127.0.0.1:7465/mcp` an. Das
Plugin `buchfink-buchhaltung` richtet die Verbindung ein und bringt die
Buchfink-Skills mit. Wer kein Plugin möchte, trägt nur den Server ein und
installiert die Skills getrennt.

## Voraussetzungen

- Buchfink läuft auf demselben Rechner, und ein Unternehmen ist geöffnet.
  Buchfink startet den Zugang für KI-Apps zusammen mit der App.
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
   `plugin:buchfink-buchhaltung:buchfink` wählen und die Anmeldung starten; der
   Browser öffnet sich. Aus der Shell:
   `claude mcp login "plugin:buchfink-buchhaltung:buchfink"`.
4. Bestätigen Sie die Anmeldung im Fenster von Buchfink (siehe unten).
5. Prüfen Sie die Verbindung (siehe unten). Die Skills erscheinen unter `/`
   als `/buchfink-buchhaltung:buchfink-grundlagen` usw.

In der Desktop-App erreichen Sie dieselbe Installation über **+ → Plugins →
Add plugin**, nachdem der Marketplace hinzugefügt ist.

Updates: Claude Code aktualisiert Plugins aus fremden Marketplaces nicht von
selbst. Aktualisieren Sie mit `claude plugin update buchfink-buchhaltung@buchfink`
oder schalten Sie unter `/plugin` → Marketplaces → buchfink „Enable
auto-update“ ein.

## Einrichten ohne Plugin

1. Öffnen Sie Buchfink und das Unternehmen.
2. Tragen Sie den Server für alle Projekte ein:
   ```sh
   claude mcp add --transport http --scope user buchfink http://127.0.0.1:7465/mcp
   ```
3. Melden Sie sich an: `claude mcp login buchfink` oder `/mcp` in einer Sitzung.
4. Installieren Sie die Skills nach `~/.claude/skills/` (braucht Node.js):
   ```sh
   npx skills add mxzinke/buchfink -g -a claude-code --skill '*'
   ```

Verwenden Sie nur einen der beiden Wege. Mit beiden sieht Claude jedes
Buchfink-Werkzeug doppelt.

## Was beim ersten Verbinden passiert

Claude Code öffnet den Browser. Die Seite zeigt „Bitte in Buchfink bestätigen“
und einen Abgleichcode. Buchfink holt gleichzeitig sein Fenster nach vorn: „Eine
App möchte auf Buchfink zugreifen“, mit dem Namen der App, demselben
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
claude mcp list
```

Der Buchfink-Server steht dort mit „Connected“. Bitten Sie Claude dann:
„Rufe das Buchfink-Werkzeug status auf.“ Die Antwort nennt das geöffnete
Unternehmen und `bereit: true`.

## Rechte für schreibende Werkzeuge einschränken

Buchfink teilt seine Werkzeuge in lesende, direkt buchende und solche mit
Rückfrage. Vor einem Werkzeug mit Rückfrage legt Claude Ihnen den Vorgang im
Chat vor und ruft es erst nach Ihrem Ja auf; so verlangen es die
Buchfink-Skills. Claude Code selbst fragt vor einem Buchfink-Werkzeug, bis Sie
„Yes, and don't ask again“ wählen.

Soll Claude Code vor jedem Buchfink-Werkzeug fragen, tragen Sie in
`~/.claude/settings.json` ein:

```json
{
  "permissions": {
    "ask": ["mcp__plugin_buchfink-buchhaltung_buchfink__*"]
  }
}
```

Ohne Plugin heißt die Regel `mcp__buchfink__*`. Früher erteilte Erlaubnisse
sehen und entfernen Sie mit `/permissions`. Festschreiben, Übermitteln,
Aufstellen, Feststellen, Offenlegen, Stammdaten und Prüfermodus gibt es nicht
als Werkzeug. Das erledigen Sie in Buchfink.

## Wenn es nicht geht

| Bild | Ursache | Abhilfe |
|---|---|---|
| `claude mcp list` zeigt „Failed to connect“ mit `ECONNREFUSED` | Buchfink ist nicht geöffnet, oder der Zugang läuft auf einem anderen Port. | Buchfink starten. In `/mcp` den Server neu verbinden. |
| Buchfink meldet unter Einstellungen → Apps verbinden, Port 7465 sei belegt | Ein anderes Programm nutzt den Port. | Prüfen mit `lsof -nP -iTCP:7465 -sTCP:LISTEN` (macOS, Linux) oder `netstat -ano \| findstr :7465` (Windows). In Buchfink einen freien Port wählen. Plugin-Server in `/mcp` abschalten und den Server ohne Plugin mit der neuen Adresse eintragen, dann neu anmelden. |
| „Needs authentication“ | Anmeldung abgelaufen oder in Buchfink entzogen. | Erneut anmelden (Schritt 3). |
| `status` antwortet mit `bereit: false` und „In Buchfink ist kein Mandant geöffnet.“; andere Werkzeuge melden `kein_mandant_offen` | In Buchfink ist kein Unternehmen geöffnet. | Unternehmen öffnen, `status` erneut aufrufen. |
| `status` antwortet mit „Diese App ist für den geöffneten Mandanten nicht freigegeben.“; andere Werkzeuge melden `mandant_nicht_freigegeben` | Die Zulassung gilt für ein anderes Unternehmen. | Anmeldung wiederholen, während das gewünschte Unternehmen geöffnet ist. |
| Die Skills fehlen unter `/` | Plugin nicht geladen. | `/reload-plugins` ausführen, Fehler unter `/plugin` → Errors lesen. |

Ob Buchfink auf dem Port antwortet, zeigt
`curl -s http://127.0.0.1:7465/.well-known/oauth-protected-resource`. Buchfink
antwortet mit JSON, das `"resource"` enthält.

## Datenschutz

Buchfink läuft auf Ihrem Rechner. Was ein Werkzeug zurückgibt, schickt Claude
Code an Anthropic: Buchungen, Beträge, Namen von Kunden, Lieferanten und
Beschäftigten, Texte aus Belegen und gelesene Belegdateien. Sie entscheiden,
welche Daten Sie so verarbeiten lassen, und tragen dafür die Verantwortung.
Prüfen Sie vor dem Einsatz mit echten Daten, ob Ihr Vertrag mit Anthropic die
Verarbeitung personenbezogener Daten abdeckt, etwa durch eine Vereinbarung zur
Auftragsverarbeitung nach [Art. 28 DSGVO][dsgvo], und ob Ihre Eingaben zum
Training genutzt werden.

## Quellen

Abgerufen am 17. Oktober 2026.

- Claude Code, MCP: <https://code.claude.com/docs/en/mcp> (`claude mcp add`,
  `claude mcp login`, `claude mcp list`, Plugin-Servernamen)
- Claude Code, Plugins installieren: <https://code.claude.com/docs/en/plugins/install>
- Claude Code, Berechtigungen: <https://code.claude.com/docs/en/permissions>
- Claude Code, Laden und Updates von Plugins: <https://code.claude.com/docs/en/plugins/loading>
- skills-CLI: <https://github.com/vercel-labs/skills> (geprüft mit 1.7.2)

[dsgvo]: https://eur-lex.europa.eu/legal-content/DE/TXT/?uri=CELEX:32016R0679
