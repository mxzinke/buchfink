# KI-Agenten mit Buchfink verbinden

[Dokumentation](../../README.md) · [Erste Schritte](../README.md)

Gilt ab Buchfink 0.3.0. Die Download-Links und die Befehle mit
`mxzinke/buchfink` in den Anleitungen funktionieren erst mit diesem Release.

Buchfink öffnet beim Start einen Zugang für KI-Apps auf Ihrem Rechner unter
`http://127.0.0.1:7465/mcp`. Eine App erhält Zugriff erst, wenn Sie ihn im
Fenster von Buchfink zulassen. Die Zulassung gilt für den Mandanten, der
gerade geöffnet ist, und lässt sich in den Einstellungen unter „Apps
verbinden“ jederzeit entziehen. Dort ändern Sie auch den Port, falls 7465
belegt ist. Buchfink startet den Zugang sofort auf dem neuen Port; tragen Sie
die neue Adresse in Ihrer App ein und starten Sie sie neu.

| App | Verbindung | Skills | Anleitung |
|---|---|---|---|
| Claude Code | Plugin `buchfink-buchhaltung` oder `claude mcp add` | im Plugin enthalten | [claude-code.md](claude-code.md) |
| Claude Desktop (macOS, Windows) | Erweiterung `buchfink.mcpb` | ZIP-Upload | [claude-desktop.md](claude-desktop.md) |
| ChatGPT Desktop | Einstellungen → MCP servers → Streamable HTTP | Skill-Ordner unter `~/.agents/skills/` | [chatgpt-desktop.md](chatgpt-desktop.md) |
| OpenCode | Eintrag in `opencode.json` | `npx skills add` | [opencode.md](opencode.md) |
| Codex | `codex mcp add` | `npx skills add` | [codex.md](codex.md) |
| Pi | `pi mcp add` | `npx skills add` | [pi.md](pi.md) |

Die Skills liegen im Repository unter [`skills/`](../../../skills/), das
Plugin für Claude Code unter
[`plugins/buchfink-buchhaltung/`](../../../plugins/buchfink-buchhaltung/). Die
Werkzeuge des Servers beschreibt [MCP-Server](../../entwicklung/mcp.md).

## Was der Agent darf

Jede Buchung speichert, wer sie verantwortet und über welchen Weg sie kam: Sie
in der App, ein Hintergrundlauf von Buchfink oder eine verbundene App mit ihrem
Kurznamen. Bis zur Festschreibung lässt sich jede Buchung durch Storno und
neue Buchung korrigieren. Was sich so korrigieren lässt, darf der Agent:
Routine ohne Rückfrage, alles andere nach Ihrer Zustimmung im Chat. Was sich
nicht zurücknehmen lässt, bleibt bei Ihnen; Stammdaten schlägt der Agent nur vor.

**Direkt, ohne Rückfrage:**

- lesen: Bücher, Belege, Bank, offene Posten, Auswertungen, Fristen, Eingang
- Belege und Kontoauszüge ablegen, Kontakte anlegen
- Exporte in den Exportordner schreiben: Belegkopien, Erklärungen, E-Bilanz
- eine Pflicht als vorbereitet melden und eine Standardgruppe vorschlagen
- Unternehmensdaten, Gründung und Bankkonten vorschlagen. Der Vorschlag ändert
  nichts: Buchfink zeigt die Vorschau mit bisherigem und neuem Wert, und erst
  wenn Sie „Übernehmen“ wählen, gilt er. Lehnen Sie ab, sieht der Agent Ihren
  Grund.
- Routinefälle buchen: eine geprüfte E-Rechnung eines bekannten Lieferanten
  mit der Standardgruppe, die Sie bestätigt haben, und einen Bankumsatz, der
  genau zu einem offenen Posten passt

Eine E-Rechnung bucht der Agent nur dann ohne Rückfrage, wenn ihr
Bruttobetrag unter der Betragsgrenze liegt (voreingestellt 1.000 €) und der
Lieferant schon so oft mit seiner Standardgruppe gebucht wurde wie eingestellt
(voreingestellt dreimal). Beides stellen Sie unter Einstellungen › „Buchen
ohne Rückfrage“ ein. Eine Standardgruppe bestätigen Sie unter Unternehmen ›
Kontakte beim Lieferanten. Die Regeln im Einzelnen stehen unter
[Regeln für direkt](../../entwicklung/mcp.md#regeln-für-direkt).

**Mit Rückfrage:** Diese Vorgänge legt der Agent Ihnen im Chat vor und führt
sie erst aus, wenn Sie zugestimmt haben:

- Belege, die keine Routine sind, Fotos und PDF ohne E-Rechnung, Auslagen,
  einen Beleg verwerfen
- Handbuchungen, Stornos, Aufteilung eines Bankumsatzes, Lohnjournal
- Rechnungen ausstellen und Mahnungen vermerken
- Abschlussbuchungen und Angaben zu Steuererklärungen
- Anlässe erfassen, etwa einen Gesellschafterwechsel oder einen
  Vorauszahlungsbescheid
- Kapital, Rücklagen, Ausschüttung und Gründungsbuchungen
- Gesellschafterdarlehen erfassen und berichtigen
- die Ergebnisverwendung buchen, nachdem Sie den Jahresabschluss festgestellt
  haben
- die Texte der Verfahrensdokumentation, die beschreiben, wie Ihr Unternehmen
  arbeitet

**Nur Sie in Buchfink:**

- Zeiträume festschreiben
- die Übermittlung von Voranmeldung, Zusammenfassender Meldung,
  Jahreserklärungen und E-Bilanz bestätigen und das Transferticket eintragen
- Pflichten als erledigt bestätigen
- Jahresabschluss aufstellen, feststellen und offenlegen
- Unternehmensdaten pflegen und Vorschläge des Agenten dazu übernehmen oder
  ablehnen: Firma, Anschrift, Register, Steuernummer, Bankkonten,
  Gesellschafter, Gründung, steuerliche Beratung
- Angaben zur Steuererklärung bestätigen, die Ihnen vorbehalten sind
- Einstellungen zum Jahresabschluss ändern, etwa den Hebesatz
- Ablage, Eingang, Exportordner und weitere Leseordner wählen
- die Regeln für Buchen ohne Rückfrage einstellen und Standardgruppen
  bestätigen
- Prüfermodus ein- und ausschalten
- Apps zulassen und Zugriff entziehen

Ein Werkzeug für diese Vorgänge gibt es nicht, für Stammdaten nur das
Vorschlagen. Versucht ein Agent sie über einen anderen Weg, weist der Kern von
Buchfink den Vorgang ab.

## Ordner des Agenten

Belege und Kontoauszüge übergibt ein Agent als Pfad zu einer Datei im Eingang,
wie unter [Belege in den Eingang legen](../README.md#belege-in-den-eingang-legen)
beschrieben. Eine Datei aus einem anderen Ordner, etwa dem Download-Ordner,
weist Buchfink mit `pfad_nicht_erlaubt` ab. Soll der Agent dort lesen, nehmen
Sie den Ordner unter Einstellungen › „Eingang & Ordner“ › „Erweitert“ als
Leseordner auf. Aus einem Leseordner verschiebt Buchfink nichts.

Was ein Agent erzeugt, schreibt Buchfink in den Exportordner, voreingestellt
`Export` im Datenordner: ausgestellte Rechnungen als PDF und E-Rechnung,
Kopien von Belegdateien, Steuererklärungen und die E-Bilanz. Einen anderen
Exportordner wählen Sie ebenfalls unter „Eingang & Ordner“.

## Anmeldung zulassen

Das gilt für jeden Client. Er öffnet den Browser; die Seite zeigt „Bitte in
Buchfink bestätigen“ und einen Abgleichcode. Buchfink holt gleichzeitig sein
Fenster nach vorn: „Eine App möchte auf Buchfink zugreifen“, mit dem Namen der
App, demselben Abgleichcode, der Rücksprungadresse, dem geöffneten Mandanten
und den Rechten (Lesen, Buchen).

Klicken Sie auf **Zulassen**, wenn beide Codes übereinstimmen und Sie die
Anmeldung gerade selbst gestartet haben. Sonst klicken Sie auf **Ablehnen**.
Die Anfrage verfällt nach fünf Minuten. Die Zulassung gilt für den geöffneten
Mandanten. Für einen weiteren Mandanten öffnen Sie ihn in Buchfink und melden
sich erneut an. Entziehen können Sie den Zugriff jederzeit unter Einstellungen
› „Apps verbinden“. Die Sicherheitsmerkmale beschreibt
[Datensicherheit](../datensicherheit.md#verbundene-apps).

## Verbindung prüfen

Bitten Sie den Agenten: „Rufe das Buchfink-Werkzeug status auf.“ Die Antwort
nennt das geöffnete Unternehmen und `bereit: true`. Jede Anleitung nennt
zusätzlich den Befehl, der die Verbindung des Clients listet.

## Skills

Die Verbindung stellt die Werkzeuge bereit, der Skill `buchfink` das Fachwissen
und die Arbeitsschritte. Das Release enthält ihn als `buchfink.zip`
(<https://github.com/mxzinke/buchfink/releases/latest/download/buchfink.zip>).
Clients, die Skills aus `~/.agents/skills/` lesen, bekommen ihn mit der
skills-CLI (braucht Node.js):

```sh
npx skills add mxzinke/buchfink -g -a <client> --skill '*'
```

Die Anleitungen nennen den Wert für `<client>` und, wo es abweicht, den Ordner
für den Handeinbau der ZIP-Datei. Skills mit Namen wie `buchfink-grundlagen`
oder `buchfink-belege` sind Reste einer älteren Fassung; löschen Sie sie, nur
`buchfink` bleibt.

## Rechte einschränken

Buchfink kennzeichnet seine Werkzeuge als lesend, direkt buchend oder mit
Rückfrage ([Freigabestufen](../../entwicklung/mcp.md#freigabestufen)). Vor
einem Werkzeug mit Rückfrage legt der Agent Ihnen den Vorgang im Chat vor, so
verlangen es die Skills. Jeder Client kann zusätzlich selbst fragen; die
Anleitungen zeigen, wie. Festschreiben, Übermitteln, Aufstellen, Feststellen,
Offenlegen, das Übernehmen von Stammdaten und der Prüfermodus gibt es nicht als
Werkzeug. Das erledigen Sie in Buchfink.

## Wenn es nicht geht

Diese Fälle gelten für jeden Client. Die Anleitungen ergänzen, was nur ihn
betrifft.

| Bild | Ursache | Abhilfe |
|---|---|---|
| Der Client erreicht Buchfink nicht | Buchfink ist nicht geöffnet, oder der Zugang läuft auf einem anderen Port. | Buchfink starten. |
| Buchfink meldet unter Einstellungen › „Apps verbinden“, Port 7465 sei belegt | Ein anderes Programm nutzt den Port. | Prüfen mit `lsof -nP -iTCP:7465 -sTCP:LISTEN` (macOS, Linux) oder `netstat -ano \| findstr :7465` (Windows). In Buchfink einen freien Port wählen, die Adresse im Client anpassen und neu anmelden. |
| Anmeldung abgelaufen oder in Buchfink entzogen | Die gespeicherte Anmeldung gilt nicht mehr. | Erneut anmelden. |
| `status` antwortet mit `bereit: false` und „In Buchfink ist kein Mandant geöffnet.“; andere Werkzeuge melden `kein_mandant_offen` | In Buchfink ist kein Unternehmen geöffnet. | Unternehmen öffnen, `status` erneut aufrufen. |
| `status` antwortet mit „Diese App ist für den geöffneten Mandanten nicht freigegeben.“; andere Werkzeuge melden `mandant_nicht_freigegeben` | Die Zulassung gilt für ein anderes Unternehmen. | Anmeldung wiederholen, während das gewünschte Unternehmen geöffnet ist. |

Ob Buchfink auf dem Port antwortet, zeigt
`curl -s http://127.0.0.1:7465/.well-known/oauth-protected-resource`. Buchfink
antwortet mit JSON, das `"resource"` enthält.

## Datenschutz

Buchfink läuft auf Ihrem Rechner. Was ein Werkzeug zurückgibt, schickt die
KI-App an den Anbieter ihres Sprachmodells. Dazu gehören Buchungen, Beträge,
Namen von Kunden, Lieferanten und Beschäftigten, Texte aus Belegen und gelesene
Belegdateien. Sie entscheiden, welche Daten Sie so verarbeiten lassen, und
tragen dafür die Verantwortung. Klären Sie vor dem Einsatz mit echten Daten mit
dem Anbieter, ob sein Vertrag die Verarbeitung personenbezogener Daten abdeckt
und ob Ihre Eingaben zum Training genutzt werden; siehe
[KI-Anbieter](../datensicherheit.md#ki-anbieter). Welcher Anbieter dahintersteht,
nennt jede Anleitung.
