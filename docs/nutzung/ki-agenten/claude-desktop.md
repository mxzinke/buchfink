# Buchfink mit Claude Desktop verbinden

[Dokumentation](../../README.md) · [KI-Agenten](README.md)

Remote-Connectors von Claude Desktop laufen über die Infrastruktur von
Anthropic und erreichen die Buchfink-App auf Ihrem Rechner nicht. Die
Erweiterung `buchfink.mcpb` enthält deshalb ein kleines Brückenprogramm
([`buchfink-mcp`](../../../packaging/claude-desktop/README.md)): Es nimmt die
Anfragen von Claude Desktop entgegen und reicht sie an die laufende
Buchfink-App weiter. Den Skill laden Sie getrennt als ZIP-Datei hoch.

Claude Desktop gibt es für macOS und Windows. Unter Linux verwenden Sie
[Claude Code](claude-code.md). Der Bereich Code der Desktop-App ist Claude Code;
dort gilt ebenfalls die Anleitung für Claude Code.

## Voraussetzungen

- Buchfink läuft auf demselben Rechner, und ein Unternehmen ist geöffnet.
- Für Skills: ein Claude-Tarif mit Skills (Free, Pro, Max, Team, Enterprise)
  und eingeschaltete Code-Ausführung.

## Einrichten

1. Öffnen Sie Buchfink und darin das Unternehmen, mit dem Claude arbeiten soll.
2. Laden Sie die Erweiterung herunter:
   <https://github.com/mxzinke/buchfink/releases/latest/download/buchfink.mcpb>.
   Doppelklicken Sie die Datei oder ziehen Sie sie in das Fenster von Claude
   Desktop. Alternativ: Settings → Extensions → Advanced settings → „Install
   Extension…“. Prüfen Sie die Angaben und installieren Sie. Das Feld „Adresse
   der Buchfink-App“ bleibt unverändert, solange Buchfink Port 7465 nutzt.
3. Schalten Sie die Code-Ausführung ein: Settings → Capabilities → „Code
   execution and file creation“. In Team- und Enterprise-Tarifen schaltet sie
   der Inhaber unter Organization settings → Plugins & skills frei.
4. Laden Sie den [Skill](README.md#skills) (`buchfink.zip`) hoch. In Claude: Customize → Skills → „+“ → „Create skill“ → „Upload a skill“,
   Datei wählen, Skill einschalten. Die ZIP-Datei nicht entpacken.
5. Verbinden Sie: Schreiben Sie in einem neuen Chat „Rufe das Buchfink-Werkzeug
   status auf.“ Beim ersten Mal öffnet die Erweiterung den Browser;
   [lassen Sie die Anmeldung in Buchfink zu](README.md#anmeldung-zulassen).
   Wiederholt Claude den Aufruf danach nicht selbst, bitten Sie noch einmal
   darum.

Die Menüangaben folgen der englischen Oberfläche von Claude Desktop.

## Zulassen

Die Anfrage heißt „Claude Desktop (Buchfink-Brücke)“. Für einen weiteren
Mandanten öffnen Sie ihn in Buchfink; der nächste Werkzeugaufruf fragt dann
erneut an.

## Prüfen, ob es geht

Unter Settings → Extensions steht Buchfink als installiert und eingeschaltet.
Den Rest prüft [`status`](README.md#verbindung-prüfen).

## Belege übergeben

Eine Datei, die Sie in den Chat von Claude Desktop ziehen, erreicht Buchfink
nicht. Buchfink nimmt Belege und Kontoauszüge nur aus seinem Eingang, siehe
[Belege in den Eingang legen](../README.md#belege-in-den-eingang-legen).

## Rechte für schreibende Werkzeuge einschränken

Claude Desktop fragt vor dem Aufruf eines Werkzeugs um Erlaubnis. Erteilen Sie
eine dauerhafte Erlaubnis nur für lesende Werkzeuge. Was Buchfink selbst
abfängt, steht unter [Rechte einschränken](README.md#rechte-einschränken).

## Wenn es nicht geht

Zusätzlich zu den [allgemeinen Fällen](README.md#wenn-es-nicht-geht):

| Bild | Ursache | Abhilfe |
|---|---|---|
| Werkzeuge antworten mit `app_nicht_geoeffnet` | Buchfink ist nicht geöffnet. | Buchfink starten, Aufruf wiederholen. |
| Unter macOS startet die Erweiterung nicht, oder macOS meldet, `buchfink-mcp` sei blockiert | Die Brücke in der Erweiterung ist nicht signiert und nicht notarisiert. | Systemeinstellungen → Datenschutz & Sicherheit öffnen und bei der Meldung zu `buchfink-mcp` die Ausführung erlauben. Ohne Meldung dort im Terminal die Quarantäne-Markierung entfernen: `xattr -dr com.apple.quarantine "$HOME/Library/Application Support/Claude"`. Danach Claude Desktop neu starten. |
| Port gewechselt | Die Erweiterung zeigt auf den alten Port. | Unter Settings → Extensions → Buchfink die „Adresse der Buchfink-App“ auf `http://127.0.0.1:<Port>/mcp` setzen. |
| Werkzeuge antworten mit `anmeldung_erforderlich` | Die Erweiterung wartet auf Ihre Zustimmung in Buchfink. | Im Fenster von Buchfink zulassen, dann den Aufruf wiederholen. |
| Werkzeuge antworten mit `anmeldung_abgelehnt` | In Buchfink wurde auf Ablehnen geklickt. | Aufruf wiederholen und diesmal zulassen. |
| Der Skill-Upload scheitert | ZIP-Datei entpackt und neu gepackt, oder Code-Ausführung aus. | Originaldatei aus dem Release hochladen, Schritt 3 prüfen. |
| Neben `buchfink` erscheinen Skills mit Namen `buchfink-…` | Reste einer älteren Fassung. | Unter Customize → Skills alle Skills löschen, deren Name mit `buchfink-` beginnt. |

Die Erweiterung speichert ihre Anmeldung unter
`~/Library/Application Support/Buchfink/mcp-bruecke/` (macOS) bzw.
`%AppData%\Buchfink\mcp-bruecke\` (Windows).

## Datenschutz

Claude Desktop schickt Werkzeugergebnisse an Anthropic. Prüfen Sie, ob Ihr
Vertrag mit Anthropic eine Auftragsverarbeitung nach [Art. 28 DSGVO][dsgvo]
abdeckt, und siehe [Datenschutz](README.md#datenschutz).

## Quellen

Abgerufen am 17. Oktober 2026.

- Desktop-Erweiterungen (MCPB), Installation und Plattformen, Remote-Connectors
  über die Infrastruktur von Anthropic:
  <https://claude.com/docs/connectors/building/mcpb>
- Lokale MCP-Server in Claude Desktop:
  <https://support.claude.com/en/articles/10949351-getting-started-with-local-mcp-servers-on-claude-desktop>
- Skills verwenden, Code-Ausführung, Upload:
  <https://support.claude.com/en/articles/12512180-use-skills-in-claude>
- Aufbau der Skill-ZIP-Datei: <https://claude.com/docs/skills/how-to>
- Brücke `buchfink-mcp`: [packaging/claude-desktop/README.md](../../../packaging/claude-desktop/README.md)

[dsgvo]: https://eur-lex.europa.eu/legal-content/DE/TXT/?uri=CELEX:32016R0679
