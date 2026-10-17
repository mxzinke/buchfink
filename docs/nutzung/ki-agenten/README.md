# KI-Agenten mit Buchfink verbinden

[Dokumentation](../../README.md) · [Erste Schritte](../README.md)

Gilt ab Buchfink 0.3.0.

Buchfink öffnet beim Start einen Zugang für KI-Apps auf Ihrem Rechner unter
`http://127.0.0.1:7465/mcp`. Eine App erhält Zugriff erst, wenn Sie ihn im
Fenster von Buchfink zulassen. Die Zulassung gilt für den Mandanten, der
gerade geöffnet ist, und lässt sich in den Einstellungen unter „Verbundene
Apps“ jederzeit entziehen. Dort ändern Sie auch den Port, falls 7465 belegt ist.

| App | Verbindung | Skills | Anleitung |
|---|---|---|---|
| Claude Code | Plugin `buchfink-buchhaltung` oder `claude mcp add` | im Plugin enthalten | [claude-code.md](claude-code.md) |
| Claude Desktop (macOS, Windows) | Erweiterung `buchfink.mcpb` | ZIP-Upload | [claude-desktop.md](claude-desktop.md) |
| OpenCode | Eintrag in `opencode.json` | `npx skills add` | [opencode.md](opencode.md) |
| Codex | `codex mcp add` | `npx skills add` | [codex.md](codex.md) |
| Pi | `pi mcp add` | `npx skills add` | [pi.md](pi.md) |

Die Skills liegen im Repository unter [`skills/`](../../../skills/), das
Plugin für Claude Code unter
[`plugins/buchfink-buchhaltung/`](../../../plugins/buchfink-buchhaltung/). Die
Werkzeuge des Servers beschreibt [MCP-Server](../../entwicklung/mcp.md).

## Was der Agent darf

Der Agent liest Bücher und Auswertungen. Routinefälle bucht er nach den Regeln,
die Buchfink vorgibt. Alles andere legt er Ihnen im Chat als Rückfrage vor und
führt es erst aus, wenn Sie zugestimmt haben. Jede Buchung speichert, wer sie
verantwortet und über welchen Weg sie kam: Sie in der App, ein Hintergrundlauf
von Buchfink oder eine verbundene App mit ihrem Kurznamen.

Nur Sie in Buchfink:

- Zeiträume festschreiben
- die Übermittlung von Voranmeldung, Zusammenfassender Meldung und
  Jahreserklärungen bestätigen und das Transferticket eintragen
- Jahresabschluss aufstellen, feststellen und offenlegen, Ergebnisverwendung
  beschließen
- Unternehmensdaten pflegen: Firma, Anschrift, Register, Steuernummer,
  Bankkonten, Gesellschafter, steuerliche Beratung
- Gesellschafterdarlehen erfassen und berichtigen
- Pflichten als erledigt bestätigen
- Angaben zur Steuererklärung bestätigen, die Ihnen vorbehalten sind
- Prüfermodus ein- und ausschalten
- Apps zulassen und Zugriff entziehen

Ein Werkzeug für diese Vorgänge gibt es nicht. Versucht ein Agent sie über
einen anderen Weg, weist der Kern von Buchfink den Vorgang ab.

## Datenschutz

Buchfink läuft auf Ihrem Rechner. Was ein Werkzeug zurückgibt, schickt die
KI-App an den Anbieter ihres Sprachmodells. Dazu gehören Buchungen, Beträge,
Namen von Kunden, Lieferanten und Beschäftigten, Texte aus Belegen und gelesene
Belegdateien. Sie entscheiden, welche Daten Sie so verarbeiten lassen, und
tragen dafür die Verantwortung. Jede Anleitung nennt, was Sie vorher mit dem
Anbieter klären sollten. Mehr dazu unter
[Datensicherheit](../datensicherheit.md#ki-anbieter).
