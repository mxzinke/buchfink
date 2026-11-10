# Buchfink mit ChatGPT Desktop verbinden

[Dokumentation](../../README.md) · [KI-Agenten](README.md)

ChatGPT Desktop kann Buchfink direkt auf Ihrem Rechner erreichen. Sie richten
die Verbindung in den Einstellungen ein. Dafür brauchen Sie weder eine
Erweiterung noch einen Terminalbefehl.

## Voraussetzungen

- Buchfink und ChatGPT Desktop sind auf demselben Rechner geöffnet.
- In Buchfink ist das Unternehmen geöffnet, mit dem Sie arbeiten möchten.
- ChatGPT Desktop zeigt in den Einstellungen den Bereich „MCP servers“ an.

## Die Verbindung einrichten

1. Öffnen Sie in ChatGPT Desktop die Einstellungen, **Settings**, und dann
   **MCP servers**.
2. Wählen Sie **Add server** und geben Sie als Namen `buchfink` ein.
3. Wählen Sie als Verbindungstyp **Streamable HTTP** und tragen Sie diese
   Adresse ein:
   ```text
   http://127.0.0.1:7465/mcp
   ```
   Das ist die Adresse von Buchfink auf Ihrem eigenen Rechner.
4. Speichern Sie und wählen Sie **Restart**, um die Verbindung zu laden.
5. Wählen Sie beim Eintrag `buchfink` **Authenticate**, um die Anmeldung zu
   starten. Der Browser öffnet sich und zeigt einen Abgleichcode.
6. Wechseln Sie zu Buchfink und [lassen Sie die Anmeldung zu](README.md#anmeldung-zulassen).

ChatGPT Desktop und Codex teilen bei lokalen Aufgaben auf demselben Rechner
ihre MCP-Einstellungen. Haben Sie Buchfink dort schon für Codex eingerichtet,
prüfen Sie zuerst, ob `buchfink` in der Serverliste vorhanden ist.

## Den Skill hinzufügen

Laden Sie den [Skill](README.md#skills) (`buchfink.zip`) herunter und entpacken
Sie ihn nach `~/.agents/skills/` (Windows: `%USERPROFILE%\.agents\skills\`).
Er liegt dort im Unterordner `buchfink` mit seiner `SKILL.md`. Öffnen Sie in
ChatGPT Desktop **Skills** und prüfen Sie, ob er erscheint; sonst starten Sie
die App neu. Mit Terminal geht es wie in der
[Codex-Anleitung](codex.md#einrichten).

## Prüfen, ob es geht

Geben Sie im Eingabefeld `/mcp` ein. Dort sollte `buchfink` als verbundener
Server erscheinen. Den Rest prüft [`status`](README.md#verbindung-prüfen).
Danach können Sie zum Beispiel fragen: „Welche Belege liegen im Eingang?“

Was der Agent darf und was nur Sie in Buchfink erledigen, steht unter
[Was der Agent darf](README.md#was-der-agent-darf).

## Wenn es nicht geht

Zusätzlich zu den [allgemeinen Fällen](README.md#wenn-es-nicht-geht):

| Problem | Was Sie tun können |
|---|---|
| Buchfink ist nicht erreichbar | Prüfen Sie, ob die Aufgabe lokal auf demselben Rechner läuft. Eine reine Cloud-Aufgabe erreicht die lokale Adresse nicht. |
| Die Anmeldung wartet | Wechseln Sie zum Fenster von Buchfink und prüfen Sie die Anfrage. |
| Freigabe abgelaufen oder für ein anderes Unternehmen | Öffnen Sie das gewünschte Unternehmen und wählen Sie erneut **Authenticate**. |
| Sie haben den Port geändert | Passen Sie `7465` in der Serveradresse an, speichern Sie und melden Sie sich erneut an. |
| „MCP servers“ fehlt | Prüfen Sie Ihre App-Version und die verlinkte OpenAI-Anleitung. Die Weboberfläche liest diese lokale Konfiguration nicht. |

## Datenschutz

ChatGPT schickt Werkzeugergebnisse zur Verarbeitung an OpenAI, einschließlich
Beleginhalten, Namen und Beträgen. Dass die Daten in Buchfink lokal liegen,
macht die Verarbeitung durch die KI nicht lokal. Siehe
[Datenschutz](README.md#datenschutz).

## Quellen und Prüfstand

Die Einrichtung folgt der offiziellen OpenAI-Dokumentation, abgerufen am
19. Oktober 2026. Ein vollständiger Verbindungstest mit Buchfink steht noch aus.

- [MCP einrichten, einschließlich ChatGPT Desktop](https://learn.chatgpt.com/docs/extend/mcp)
- [Skills in ChatGPT und Codex](https://learn.chatgpt.com/docs/build-skills)
