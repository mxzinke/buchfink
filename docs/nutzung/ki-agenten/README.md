# KI-Agenten mit Buchfink verbinden

[Dokumentation](../../README.md) · [Erste Schritte](../README.md)

Gilt ab Buchfink 0.3.0.

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
nicht zurücknehmen lässt, und die Stammdaten bleiben bei Ihnen.

**Direkt, ohne Rückfrage:**

- lesen: Bücher, Belege, Bank, offene Posten, Auswertungen, Fristen, Eingang
- Belege und Kontoauszüge aus dem Eingang oder einem Leseordner ablegen,
  Kontakte anlegen
- Exporte in den Exportordner schreiben: Belegkopien, Erklärungen, E-Bilanz
- eine Pflicht als vorbereitet melden und eine Standardgruppe vorschlagen
- Routinefälle buchen: eine geprüfte E-Rechnung eines bekannten Lieferanten
  mit der Standardgruppe, die Sie bestätigt haben, und einen Bankumsatz, der
  genau zu einem offenen Posten passt

Eine E-Rechnung bucht der Agent nur dann ohne Rückfrage, wenn ihr
Bruttobetrag unter der Betragsgrenze liegt (voreingestellt 1.000 €) und der
Lieferant schon so oft mit seiner Standardgruppe gebucht wurde wie eingestellt
(voreingestellt dreimal). Beides stellen Sie unter Einstellungen › „Buchen
ohne Rückfrage“ ein. Eine Standardgruppe bestätigen Sie unter Unternehmen ›
Kontakte beim Lieferanten; bis dahin fragt der Agent bei jeder Rechnung
nach.

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
- Unternehmensdaten pflegen: Firma, Anschrift, Register, Steuernummer,
  Bankkonten, Gesellschafter, Gründung, steuerliche Beratung
- Angaben zur Steuererklärung bestätigen, die Ihnen vorbehalten sind
- Einstellungen zum Jahresabschluss ändern, etwa den Hebesatz
- Ablage, Eingang, Exportordner und weitere Leseordner wählen
- die Regeln für Buchen ohne Rückfrage einstellen und Standardgruppen
  bestätigen
- Prüfermodus ein- und ausschalten
- Apps zulassen und Zugriff entziehen

Ein Werkzeug für diese Vorgänge gibt es nicht. Versucht ein Agent sie über
einen anderen Weg, weist der Kern von Buchfink den Vorgang ab.

## Eingang und Exporte

Ein Agent übergibt Buchfink Belege und Kontoauszüge als Pfad zu einer Datei.
Buchfink liest sie aus dem Eingang, den Sie bei der Einrichtung gewählt haben,
voreingestellt der Ordner `Eingang` im Datenordner. Legen Sie eine Rechnung
dort ab oder ziehen Sie sie ins Fenster von Buchfink und sagen Sie dann etwa:
„Buche die Belege im Eingang.“ Nach der Ablage verschiebt Buchfink die Datei
in den Unterordner `Übernommen/<JJJJ-MM>`. Wie das im Einzelnen geht, steht
unter [Belege in den Eingang legen](../README.md#belege-in-den-eingang-legen).

Eine Datei aus einem anderen Ordner, etwa aus dem Download-Ordner, weist
Buchfink mit `pfad_nicht_erlaubt` ab. Soll der Agent auch dort oder in einem
Ordner mit Verträgen lesen, nehmen Sie ihn unter Einstellungen › „Eingang &
Ordner“ › „Erweitert“ als Leseordner auf. Aus einem Leseordner verschiebt
Buchfink nichts. Legen Sie nur echte Belege in Eingang und Leseordner: eine
abgelegte Datei bleibt im Belegarchiv und in jeder Sicherung, auch wenn der
Beleg später verworfen wird.

Was ein Agent erzeugt, schreibt Buchfink in den Exportordner, voreingestellt
`Export` im Datenordner: ausgestellte Rechnungen als PDF und E-Rechnung,
Kopien von Belegdateien, Steuererklärungen und die E-Bilanz. Einen anderen
Exportordner wählen Sie ebenfalls unter „Eingang & Ordner“.

## Datenschutz

Buchfink läuft auf Ihrem Rechner. Was ein Werkzeug zurückgibt, schickt die
KI-App an den Anbieter ihres Sprachmodells. Dazu gehören Buchungen, Beträge,
Namen von Kunden, Lieferanten und Beschäftigten, Texte aus Belegen und gelesene
Belegdateien. Sie entscheiden, welche Daten Sie so verarbeiten lassen, und
tragen dafür die Verantwortung. Jede Anleitung nennt, was Sie vorher mit dem
Anbieter klären sollten. Mehr dazu unter
[Datensicherheit](../datensicherheit.md#ki-anbieter).
