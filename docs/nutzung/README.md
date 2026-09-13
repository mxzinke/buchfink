# Erste Schritte

[Dokumentation](../README.md) · [Funktionen und Grenzen](../projekt/umsetzungsstand.md)

Buchfink ist eine Vorschau für kleine Unternehmen mit doppelter Buchführung,
vor allem GmbHs und UGs. Die Vorschau wird aus dem Quellcode gebaut;
fertige, signierte Installationsdateien sind ein Ziel der [Roadmap](../projekt/roadmap.md).

## Anwendung starten

Richte die [Entwicklungsumgebung](../entwicklung/README.md#voraussetzungen) ein und folge
[dem lokalen Start](../entwicklung/README.md#lokal-starten). Dafür sind derzeit
Entwicklungswerkzeuge nötig. Einen Überblick ohne Installation bietet die
[Projektseite](https://mxzinke.github.io/buchfink/).

## Ein Testunternehmen einrichten

1. Lege im Einrichtungsassistenten ein Testunternehmen mit erfundenen Daten an.
   Wähle einen eigenen, leeren Datenordner.
2. Wähle, ob du eine Gründung oder bestehende Buchhaltung erfassen möchtest,
   und gib das erste Geschäftsjahr sowie die Unternehmensdaten an.
3. Sichere den Wiederherstellungsschlüssel im letzten Einrichtungsschritt.
   Bewahre die Datei getrennt vom Rechner und der Datensicherung auf.
4. Erprobe einen Beleg und seine Buchung. Prüfe anschließend, ob du die
   Buchung im Journal und in den Auswertungen wiederfindest. Die Fragezeichen
   in der Oberfläche erklären Begriffe und nächste Schritte.

Für ausführlichere Tests stehen [Prüfszenarien](../entwicklung/pruefszenarien.md) mit
Beispieldaten und erwarteten Ergebnissen bereit.

## Daten sichern

Richte ein Sicherungsziel ein. Eine Datensicherung ersetzt die
Wiederherstellungsdatei nicht. Erprobe beide zusammen durch eine
Wiederherstellung in einen neuen Ordner. Belegdateien und das Sicherungsarchiv
sind derzeit unverschlüsselt. Den genauen Umfang beschreibt
[Wiederherstellung und Sicherung](datensicherheit.md#wiederherstellung-und-sicherung).

## Rückmeldung geben

Melde Fehler und unverständliche Abläufe über die
[GitHub-Issues](https://github.com/mxzinke/buchfink/issues). Nenne die
Programmversion, dein Betriebssystem, die Schritte und das erwartete Ergebnis.
Verwende Beispieldaten statt echter Belege oder Unternehmensdaten.
Wie du auch mit Texten oder fachlicher Prüfung helfen kannst, steht unter
[Mitwirken](../../CONTRIBUTING.md).
