<p align="center">
  <img src="./assets/buchfink-logo.svg" alt="Buchfink Logo" width="200" />
</p>

# Buchfink

Freie Buchhaltungssoftware für kleine Unternehmen, Gründer und Holdings,
die ihre Buchhaltung selbst erledigen wollen. Kostenlos, quelloffen und auf
dem eigenen Rechner. Der Schwerpunkt liegt auf kleinen GmbHs und UGs mit
doppelter Buchführung im SKR04.

[Projektseite](https://mxzinke.github.io/buchfink/) ·
[Ausprobieren](docs/nutzung/README.md) ·
[Dokumentation](docs/README.md) · [Mitwirken](CONTRIBUTING.md)

## Aktueller Stand

Buchfink ist eine Vorschau in Entwicklung und wird derzeit aus dem Quellcode
gebaut. Belegablage, Buchungen, Bankimport, Rechnungen, Umsatzsteuer-Voranmeldung
und die Vorbereitung des Jahresabschlusses sind vorhanden.

Ein vollständig geprüfter Abschluss einschließlich Einreichung ist noch nicht
möglich. Die E-Bilanz-Zuordnung ist ungeprüft; EÜR, eigene
Kleinunternehmerbesteuerung, Lohnabrechnung, Kassenbuch und DATEV-Export fehlen.
Der [Umsetzungsstand](docs/projekt/umsetzungsstand.md) beschreibt Funktionen und
Grenzen, die [Roadmap](docs/projekt/roadmap.md) die geplanten Arbeiten.

Daten liegen lokal je Unternehmen. Ausgewählte Datenbankfelder sind
verschlüsselt, Belegdateien und Sicherungsarchive nicht. Einzelne Funktionen
nutzen externe Dienste. Details stehen im
[Sicherheitskonzept](docs/nutzung/datensicherheit.md).

## Einsteigen

| Ich möchte … | Einstieg |
|---|---|
| die Vorschau ausprobieren | [Erste Schritte](docs/nutzung/README.md) |
| am Code arbeiten | [Entwicklungsumgebung und Prüfungen](docs/entwicklung/README.md) |
| den Aufbau verstehen | [Orientierung im Code](docs/entwicklung/codebase.md) |
| fachliche Abläufe nachvollziehen | [Fachkonzepte und Anforderungen](docs/fachkonzepte/README.md) |
| Fehler melden oder etwas beitragen | [Mitwirken](CONTRIBUTING.md) |

Rückmeldungen aus dem Unternehmensalltag, verständlichere Texte und fachliche
Prüfungen helfen ebenso wie Codebeiträge. Programmierkenntnisse sind dafür
keine Voraussetzung.

## Lizenz

Copyright © 2026 Maximilian Pfennig. Buchfink steht unter der
[EUPL-1.2](LICENSE). Die Lizenz erlaubt Nutzung, Anpassung und Weitergabe unter
ihren Bedingungen und enthält die Regelungen zu Gewährleistung und Haftung.
Mitgelieferte Komponenten und ihre Lizenzen stehen in
[THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md).
