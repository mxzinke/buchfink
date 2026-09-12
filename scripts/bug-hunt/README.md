# Buchfink isoliert prüfen

Diese Hilfen entstanden bei der [Prüfung vom 12. September 2026](../../docs/bug-hunt-2026-09-12.md).
Sie verwenden die echte Wails-Bridge, die echten Dienste, SQLite und den
Betriebssystem-Schlüsselbund. Der Server lauscht ausschließlich auf 127.0.0.1:9250.

## Anwendung starten

Voraussetzungen sind die normalen Go-/Wails-Buildwerkzeuge, Node und Typst.
Die Frontend-Abhängigkeiten müssen installiert sein.

Vom Repository-Stamm aus:

```sh
npm --prefix frontend run build
BUCHFINK_BUGHUNT_DIR=/tmp/buchfink-bughunt-2026-09-12 go test -tags server,bughunt ./internal/wailsbridge -run '^TestBugHuntServer$' -timeout 0 -v
```

Anschließend http://127.0.0.1:9250 öffnen. Ist der Prüfserver bereits aktiv,
kann er weiterverwendet werden. Strg-C beendet einen im Terminal gestarteten
Server.

BUCHFINK_BUGHUNT_DIR muss ein absoluter Pfad zur Testkonfiguration sein.
Der Test ändert weder HOME noch die normale Konfiguration unter ~/.buchfink.
Im Einrichtungsassistenten auch für die Unternehmensdaten einen neuen Testordner
angeben. Schlüssel für diese Testunternehmen werden im echten Schlüsselbund
angelegt. Der Server ist eine Entwicklungsumgebung und kein Netzwerkbetrieb
der Desktop-Anwendung.

## Hilfen

- rpc.mjs ruft Methoden der lokalen Bridge über das Wails-Protokoll auf.
  Der optionale Wert BUCHFINK_BUGHUNT_URL darf nur einen lokalen HTTP-Ursprung
  mit Port enthalten. Die Aufrufe schreiben Daten, wenn die jeweilige Methode
  dies tut.
- views.mjs öffnet standardmäßig die erste Fadenwerk-Testbuchhaltung und besucht alle
  20 Hauptansichten. Screenshots, Texte und JavaScript-Ausnahmen landen unter
  .cache/bughunt-2026-09-12/evidence/views. Benötigt die Playwright-Installation
  aus scripts/site-screenshots und deren Chromium-Browser. Mit
  BUCHFINK_BUGHUNT_COMPANY lässt sich der Firmenname, mit
  BUCHFINK_BUGHUNT_VIEWS der Ausgabeordner wählen.
- year-end.mjs bereitet das dokumentierte Nordlicht-Szenario für 2025 vor.
  Es übernimmt anfangs die Stammdaten der geöffneten Fadenwerk-Testbuchhaltung
  als Vorlage und erzeugt ausschließlich Beispieldaten. Die festen Pfade und
  Namen gehören zu diesem Audit. Das Protokoll erlaubt die Fortsetzung eines
  unterbrochenen Aufbaus, indem bereits erfolgreiche Aufrufe wiederverwendet
  werden. Es ist kein allgemeines Werkzeug zum Anlegen von Kundenbuchhaltungen.
  Festschreibung, Feststellung und Vortrag wurden anschließend in der UI
  ausgeführt und sind im vierten Video zu sehen.

```sh
node scripts/bug-hunt/views.mjs
node scripts/bug-hunt/year-end.mjs
```

Das Vorjahresszenario umgeht den dokumentierten Einrichtungsfehler ausdrücklich
durch Auswahl und Anlage von 2025. Der erste, vor dieser Gegenmaßnahme abgelegte
Beleg bleibt als Nachweis mit seiner falschen Jahreszuordnung erhalten.

followup-year-end.mjs legt das neue Nordlicht-Szenario im getrennten
Nachprüfungsordner an. Die zuvor geöffnete Leuchtspur-Testbuchhaltung liefert
die Stammdatenvorlage. Das Skript prüft die korrekte Übernahme von 2025, ohne
das Jahr nachträglich als Gegenmaßnahme anzulegen. Das ursprüngliche Skript
bleibt als Protokoll des ersten Testlaufs erhalten.

## Regressionstests ausführen

```sh
go test -tags bughunt ./internal/bank ./internal/repository ./internal/wailsbridge -run '^TestBugHunt' -count=1
```

Die sechs ursprünglichen Abnahmeproben für Bankimport und Jahresauswahl
bestehen nach den Korrekturen. Sie sind auch ohne das Build-Tag bughunt Teil
der normalen Testsuite. Sie verwenden temporäre Datenbanken und beim Bridge-Test
einen simulierten Schlüsselbund. Der Server-Test wird nur mit den beiden
Build-Tags server und bughunt eingebunden.

```sh
go test ./internal/...
go vet ./internal/...
npm --prefix frontend test
```

Videos, Exportdateien, JSON-Protokolle und Logs liegen lokal unter
.cache/bughunt-2026-09-12. Sie werden durch .gitignore ausgeschlossen und sind
deshalb separat aufzubewahren, wenn das Arbeitsverzeichnis bereinigt wird.
