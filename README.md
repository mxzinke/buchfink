<p align="center">
  <img src="./assets/buchfink-logo.svg" alt="Buchfink Logo" width="200" />
</p>

# Buchfink

Buchfink ist eine Buchhaltung für GmbH und UG, die auf dem eigenen Rechner
läuft. Ein KI-Agent führt die Bücher über MCP: Er legt Belege ab, bucht
Routinefälle nach festen Regeln und bereitet Meldungen und Erklärungen vor.
Die App zeigt den Stand und nimmt die Freigaben an, die nur der Mensch geben
darf, etwa Festschreibung, Übermittlung und Jahresabschluss.

[Projektseite](https://mxzinke.github.io/buchfink/) ·
[Dokumentation](docs/README.md) · [Mitwirken](CONTRIBUTING.md)

## Stand

Version 0.3 ist eine Vorschau. Die Installationsdateien auf der
[Releases-Seite](https://github.com/mxzinke/buchfink/releases) sind nicht
signiert. Probieren Sie Buchfink mit einem Testunternehmen aus.

Gebaut sind laufende Buchhaltung im SKR04, Umsatzsteuer-Voranmeldung,
Jahresabschluss nach HGB, Erklärungen zu KSt, GewSt und USt als Eingabehilfe
für Mein ELSTER, E-Bilanz, ein Pflichtenkalender und der MCP-Server mit
Zustimmung im Fenster der App. Buchfink übermittelt nichts selbst an die
Finanzverwaltung. Funktionen und Grenzen stehen im
[Umsetzungsstand](docs/projekt/umsetzungsstand.md) und unter
[Umfang](docs/nutzung/umfang.md).

Die Daten liegen je Unternehmen in einem Ordner und sind nicht verschlüsselt.
Hash-Ketten und Zeitstempel nach RFC 3161 machen Änderungen erkennbar. Was der
Agent liest, geht an den Anbieter seines Sprachmodells
([Datensicherheit](docs/nutzung/datensicherheit.md)).

## Einstieg

| Ich möchte … | Einstieg |
|---|---|
| Buchfink installieren und einrichten | [Erste Schritte](docs/nutzung/README.md) |
| meinen KI-Agenten verbinden | [KI-Agenten](docs/nutzung/ki-agenten/README.md) |
| wissen, wie Buchfink fachlich arbeitet | [Fachkonzepte](docs/fachkonzepte/README.md) |
| am Code arbeiten | [Entwicklung](docs/entwicklung/README.md) |
| Fehler melden oder beitragen | [Mitwirken](CONTRIBUTING.md) |

## Lizenz

Copyright © 2026 Maximilian Pfennig. Buchfink steht unter der
[EUPL-1.2](LICENSE). Mitgelieferte Komponenten und ihre Lizenzen stehen in
[THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md).
