# Buchfink mit ChatGPT Desktop verbinden

[Dokumentation](../../README.md) · [KI-Agenten](README.md)

Stand: 19. Oktober 2026. Gilt ab Buchfink 0.3.0.

ChatGPT Desktop kann Buchfink direkt auf Ihrem Rechner erreichen. Sie richten
die Verbindung in den Einstellungen ein. Dafür brauchen Sie weder eine
Erweiterung noch einen Terminalbefehl.

## Voraussetzungen

- Buchfink und ChatGPT Desktop sind auf demselben Rechner geöffnet.
- In Buchfink ist das Unternehmen geöffnet, mit dem Sie arbeiten möchten.
- ChatGPT Desktop zeigt in den Einstellungen den Bereich „MCP servers“ an.
- Die Aufgabe läuft auf Ihrem Rechner. Eine reine Cloud-Aufgabe erreicht
  Buchfink unter der lokalen Adresse nicht.

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
6. Wechseln Sie zu Buchfink. Dort erscheint die Anfrage „Eine App möchte auf
   Buchfink zugreifen“. Prüfen Sie den Namen der App, das Unternehmen, die
   angefragten Rechte und den Code. Stimmen die Codes überein und haben Sie
   die Anmeldung gerade selbst gestartet, wählen Sie **Zulassen**.

Die Freigabe gilt für das geöffnete Unternehmen. Für ein anderes Unternehmen
öffnen Sie es in Buchfink und melden sich erneut an. Den Zugriff entziehen
Sie in Buchfink unter **Einstellungen → Apps verbinden**.

ChatGPT Desktop und Codex teilen bei lokalen Aufgaben auf demselben Rechner
ihre MCP-Einstellungen. Haben Sie Buchfink dort schon für Codex eingerichtet,
prüfen Sie zuerst, ob `buchfink` in der Serverliste vorhanden ist.

## Die Buchhaltungsanleitungen hinzufügen

Die Verbindung stellt die Werkzeuge bereit. Die Buchfink-Skills ergänzen
Fachwissen und Arbeitsschritte für den Agenten.

Laden Sie das [Paket mit den Buchfink-Skills](https://github.com/mxzinke/buchfink/releases/latest/download/buchfink-skills.zip)
herunter und entpacken Sie die enthaltenen Skill-Ordner in den Ordner
`.agents/skills` in Ihrem Benutzerordner. Der vollständige Pfad ist unter
macOS und Linux `~/.agents/skills/`, unter Windows
`%USERPROFILE%\.agents\skills\`. Jeder Skill liegt dort in einem eigenen
Unterordner mit seiner `SKILL.md`.

Öffnen Sie in ChatGPT Desktop **Skills** und prüfen Sie, ob die
Buchfink-Skills erscheinen. Starten Sie die App neu, falls sie fehlen.
Alternativ beschreibt die [Codex-Anleitung](codex.md#einrichten), wie Sie
das Paket mit einem Terminalbefehl installieren.

## Prüfen, ob es geht

Geben Sie im Eingabefeld `/mcp` ein. Dort sollte `buchfink` als verbundener
Server erscheinen. Schreiben Sie anschließend:

> Rufe das Buchfink-Werkzeug status auf und zeige mir, welches Unternehmen geöffnet ist.

Die Antwort nennt Ihr Unternehmen und `bereit: true`. Danach können Sie zum
Beispiel fragen: „Welche Belege liegen im Eingang?“

Monate abschließen, den Jahresabschluss bestätigen und Unternehmensdaten
ändern können Sie nur selbst in Buchfink. Was der Agent übernehmen darf,
beschreibt [Was der Agent darf](README.md#was-der-agent-darf).

## Wenn es nicht geht

| Problem | Was Sie tun können |
|---|---|
| Buchfink ist nicht erreichbar | Öffnen Sie Buchfink und Ihr Unternehmen. Prüfen Sie, ob die Aufgabe lokal auf demselben Rechner läuft. |
| Die Anmeldung wartet | Wechseln Sie zum Fenster von Buchfink und prüfen Sie die Anfrage. |
| Die Freigabe gilt nicht mehr oder für ein anderes Unternehmen | Öffnen Sie das gewünschte Unternehmen und wählen Sie in ChatGPT Desktop erneut **Authenticate**. |
| Sie haben in Buchfink den Port geändert | Passen Sie die Zahl `7465` in der Serveradresse an, speichern Sie und melden Sie sich erneut an. |
| „MCP servers“ fehlt | Prüfen Sie Ihre App-Version und die unten verlinkte OpenAI-Anleitung. Die Weboberfläche liest diese lokale Konfiguration nicht. |

## Daten beim KI-Anbieter

Was ChatGPT aus Buchfink liest, kann zur Verarbeitung an OpenAI gehen,
einschließlich Beleginhalten, Namen und Beträgen. Die Speicherung in Buchfink
auf Ihrem Rechner bedeutet keine rein lokale Verarbeitung durch die KI.
Hinweise zur Wahl des Anbieters und zu personenbezogenen Daten stehen unter
[Datensicherheit](../datensicherheit.md#ki-anbieter).

## Quellen und Prüfstand

Die Einrichtung folgt der offiziellen OpenAI-Dokumentation, abgerufen am
19. Oktober 2026. Ein vollständiger Verbindungstest mit Buchfink steht noch aus.

- [MCP einrichten, einschließlich ChatGPT Desktop](https://learn.chatgpt.com/docs/extend/mcp)
- [Skills in ChatGPT und Codex](https://learn.chatgpt.com/docs/build-skills)
