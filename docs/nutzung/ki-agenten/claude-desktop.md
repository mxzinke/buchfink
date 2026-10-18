# Buchfink mit Claude Desktop verbinden

[Dokumentation](../../README.md) · [KI-Agenten](README.md)

Stand: 17. Oktober 2026.

Gilt ab Buchfink 0.3.0. Die Download-Links und die Befehle mit
`mxzinke/buchfink` funktionieren erst mit diesem Release.

Claude Desktop erreicht die Buchfink-App auf Ihrem Rechner nicht über einen
Remote-Connector, weil dieser über die Infrastruktur von Anthropic läuft. Der
einfachste Weg zu einem lokalen Programm ist eine Erweiterung. Die Erweiterung
`buchfink.mcpb` enthält ein kleines Brückenprogramm. Es nimmt
die Anfragen von Claude Desktop entgegen und reicht sie an die laufende
Buchfink-App weiter. Die Skills laden Sie getrennt als ZIP-Dateien hoch.

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
   der Buchfink-App“ bleibt unverändert, solange Buchfink den Standardport
   nutzt.
3. Schalten Sie die Code-Ausführung ein: Settings → Capabilities → „Code
   execution and file creation“. In Team- und Enterprise-Tarifen schaltet sie
   der Inhaber unter Organization settings → Plugins & skills frei.
4. Laden Sie die Skills hoch. Jede ZIP-Datei ist ein Skill. Beginnen Sie mit
   `buchfink-grundlagen.zip`; die anderen Skills setzen ihn voraus.
   Download: `https://github.com/mxzinke/buchfink/releases/latest/download/<skill>.zip`,
   zum Beispiel
   <https://github.com/mxzinke/buchfink/releases/latest/download/buchfink-grundlagen.zip>.
   In Claude: Customize → Skills → „+“ → „Create skill“ → „Upload a skill“,
   Datei wählen, Skill einschalten. Die ZIP-Datei nicht entpacken.
5. Verbinden Sie: Schreiben Sie in einem neuen Chat „Rufe das Buchfink-Werkzeug
   status auf.“ Beim ersten Mal öffnet die Erweiterung den Browser. Bestätigen
   Sie in Buchfink (siehe unten). Wiederholt Claude den Aufruf danach nicht
   selbst, bitten Sie noch einmal darum.

Die Menüangaben folgen der englischen Oberfläche von Claude Desktop.

## Was beim ersten Verbinden passiert

Die Browserseite zeigt „Bitte in Buchfink bestätigen“ und einen Abgleichcode.
Buchfink holt gleichzeitig sein Fenster nach vorn: „Eine App möchte auf Buchfink
zugreifen“, mit dem Namen „Claude Desktop (Buchfink-Brücke)“, demselben
Abgleichcode, der Rücksprungadresse, dem geöffneten Mandanten und den Rechten
(Lesen, Buchen).

Klicken Sie auf **Zulassen**, wenn beide Codes übereinstimmen und Sie gerade
selbst ein Buchfink-Werkzeug aufgerufen haben. Sonst klicken Sie auf
**Ablehnen**. Die Anfrage verfällt nach fünf Minuten. Die Zulassung gilt für den
geöffneten Mandanten. Für einen weiteren Mandanten öffnen Sie ihn in Buchfink;
der nächste Werkzeugaufruf fragt dann erneut an. Entziehen können Sie den
Zugriff jederzeit in Buchfink unter Einstellungen → Apps verbinden.

## Prüfen, ob es geht

Die Antwort auf `status` nennt das geöffnete Unternehmen und `bereit: true`.
Unter Settings → Extensions steht Buchfink als installiert und eingeschaltet.

## Belege übergeben

Eine Datei, die Sie in den Chat von Claude Desktop ziehen, erreicht Buchfink
nicht. Buchfink nimmt Belege und Kontoauszüge nur aus seinem Eingang. Legen
Sie die Dateien dort ab oder ziehen Sie sie ins Fenster von Buchfink und
schreiben Sie dann: „Buche die Belege im Eingang.“ Mehr unter
[Belege in den Eingang legen](../README.md#belege-in-den-eingang-legen).

## Rechte für schreibende Werkzeuge einschränken

Claude Desktop fragt vor dem Aufruf eines Werkzeugs um Erlaubnis. Erteilen Sie
eine dauerhafte Erlaubnis nur für lesende Werkzeuge. Buchfink kennzeichnet
lesende und schreibende Werkzeuge; Werkzeuge mit Rückfrage sind als
folgenreich markiert. Festschreiben, Übermitteln, Aufstellen, Feststellen,
Offenlegen, Stammdaten und Prüfermodus gibt es nicht als Werkzeug. Das erledigen
Sie in Buchfink.

## Wenn es nicht geht

| Bild | Ursache | Abhilfe |
|---|---|---|
| Werkzeuge antworten mit `app_nicht_geoeffnet` | Buchfink ist nicht geöffnet. | Buchfink starten, Aufruf wiederholen. |
| Unter macOS startet die Erweiterung nicht, oder macOS meldet, `buchfink-mcp` sei blockiert | Die Brücke in der Erweiterung ist nicht signiert und nicht notarisiert. | Systemeinstellungen → Datenschutz & Sicherheit öffnen und bei der Meldung zu `buchfink-mcp` die Ausführung erlauben. Ohne Meldung dort im Terminal die Quarantäne-Markierung entfernen: `xattr -dr com.apple.quarantine "$HOME/Library/Application Support/Claude"`. Danach Claude Desktop neu starten. |
| Buchfink meldet unter Einstellungen → Apps verbinden, Port 7465 sei belegt | Ein anderes Programm nutzt den Port. | Prüfen mit `lsof -nP -iTCP:7465 -sTCP:LISTEN` (macOS) oder `netstat -ano \| findstr :7465` (Windows). In Buchfink einen freien Port wählen. In Claude Desktop unter Settings → Extensions → Buchfink die „Adresse der Buchfink-App“ auf `http://127.0.0.1:<Port>/mcp` setzen. |
| Werkzeuge antworten mit `anmeldung_erforderlich` | Die Erweiterung wartet auf Ihre Zustimmung in Buchfink. | Im Fenster von Buchfink zulassen, dann den Aufruf wiederholen. |
| Werkzeuge antworten mit `anmeldung_abgelehnt` | In Buchfink wurde auf Ablehnen geklickt. | Aufruf wiederholen und diesmal zulassen. |
| `status` antwortet mit `bereit: false` und „In Buchfink ist kein Mandant geöffnet.“; andere Werkzeuge melden `kein_mandant_offen` | In Buchfink ist kein Unternehmen geöffnet. | Unternehmen öffnen, `status` erneut aufrufen. |
| `status` antwortet mit „Diese App ist für den geöffneten Mandanten nicht freigegeben.“; andere Werkzeuge melden `mandant_nicht_freigegeben` | Die Zulassung gilt für ein anderes Unternehmen. | Unternehmen öffnen, Werkzeug erneut aufrufen und in Buchfink zulassen. |
| Der Skill-Upload scheitert | ZIP-Datei entpackt und neu gepackt, oder Code-Ausführung aus. | Originaldatei aus dem Release hochladen, Schritt 3 prüfen. |

Die Erweiterung speichert ihre Anmeldung unter
`~/Library/Application Support/Buchfink/mcp-bruecke/` (macOS) bzw.
`%AppData%\Buchfink\mcp-bruecke\` (Windows).

## Datenschutz

Buchfink läuft auf Ihrem Rechner. Was ein Werkzeug zurückgibt, schickt Claude
Desktop an Anthropic: Buchungen, Beträge, Namen von Kunden, Lieferanten und
Beschäftigten, Texte aus Belegen und gelesene Belegdateien. Sie entscheiden,
welche Daten Sie so verarbeiten lassen, und tragen dafür die Verantwortung.
Prüfen Sie vor dem Einsatz mit echten Daten, ob Ihr Vertrag mit Anthropic die
Verarbeitung personenbezogener Daten abdeckt, etwa durch eine Vereinbarung zur
Auftragsverarbeitung nach [Art. 28 DSGVO][dsgvo], und ob Ihre Eingaben zum
Training genutzt werden.

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
