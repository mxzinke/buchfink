# Dokumentation pflegen

[Dokumentation](../README.md) · [Mitwirken](../../CONTRIBUTING.md)

Die Dokumentation beschreibt den aktuellen Stand. Was früher galt, steht im
Git-Verlauf und in der [Änderungshistorie](../../internal/changelog/CHANGELOG.md),
Pläne stehen in der [Roadmap](../projekt/roadmap.md). Viel Dokumentation zu
einer Stelle ist ein Zeichen, dass Code, Name oder Schnitt nicht stimmen.

## Wohin was gehört

| Inhalt | Ort |
|---|---|
| Installation, Einrichtung, Anleitungen für KI-Clients, Datensicherheit | [nutzung/](../nutzung/README.md) |
| Gebaute Funktionen und bekannte Grenzen | [Umsetzungsstand](../projekt/umsetzungsstand.md) |
| Geplante Arbeit | [Roadmap](../projekt/roadmap.md) |
| Setup, Architektur, Prüfwege | [entwicklung/](README.md) |
| Werkzeuge des MCP-Servers | [MCP-Server](mcp.md) |
| Gestaltung und Sprache | [gestaltung/](../gestaltung/README.md) |
| Fachliche Abläufe und Entscheidungen | [fachkonzepte/](../fachkonzepte/README.md) |
| Anforderungen mit Rechtsquelle und Umsetzung | [anforderungen/](../anforderungen/README.md) |
| Fachwissen für den Agenten, mit Quellen | `skills/<name>/` |
| Anleitung zu einem Werkzeug im Repository | bei seinem Code, etwa `scripts/`, `website/`, `packaging/` |
| Begründung einer Änderung | Commit-Text |

Jeder Inhalt hat einen Ort. Andere Seiten verlinken dorthin, statt Listen oder
Regeln zu kopieren. Die README im Wurzelordner bleibt ein kurzer Einstieg.

## Regeln

- Präsens, aktueller Stand. Keine Erzählung, wie es dazu kam.
- Fundstellen als Paket, Funktion oder Test, nie als Datei mit Zeilennummer.
- Rechtsquellen als Link auf den Gesetzestext oder die Originalveröffentlichung.
- Mechanismen und Grenzen beschreiben. Gesetzeskonformität sichert die
  Dokumentation nicht zu.
- Schreibweise nach [Schreibweise](../gestaltung/schreibweise.md).
- Kontonummern, die als Teil eines Buchungssatzes fett (`**1800**`) oder als
  Ziel (`→ 1800`) dastehen, prüft
  `accounting.TestConceptDocumentsUseRealSKR04Accounts` gegen den Kontenrahmen.

## Ändern

Mit einer Funktion ändern sich Umsetzungsstand, betroffene Anforderungen und
gegebenenfalls Webseite und Skills im selben Pull Request. Beim Verschieben
oder Löschen einer Seite werden relative Links im ganzen Repository angepasst,
auch in Webseite, Codekommentaren und Skripten.
