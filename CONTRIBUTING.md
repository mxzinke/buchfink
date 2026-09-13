# Mitwirken an Buchfink

[Projektübersicht](README.md) · [Dokumentation](docs/README.md)

Du kannst mit Fehlerberichten, Rückmeldungen zur Bedienung, verständlicheren
Texten, fachlichen Prüfungen oder Code beitragen.

## Fehler melden und Rückmeldung geben

Öffne ein [Issue](https://github.com/mxzinke/buchfink/issues) und beschreibe,
was du erreichen wolltest, welche Schritte du ausgeführt hast und was
stattdessen passiert ist. Nenne Programmversion und Betriebssystem. Bei
fachlichen Fragen hilft ein konkreter Geschäftsvorfall mit erwartetem Ergebnis.
Verwende erfundene Daten und entferne persönliche oder geschäftliche Angaben
aus Screenshots und Protokollen.

## Bevor du loslegst

Für alles, was größer ist als ein Tippfehler: öffne zuerst ein Issue und
beschreibe das Problem. Das erspart dir die Arbeit an einem Pull Request, der
inhaltlich in eine andere Richtung soll.

Fehlerberichte, Reproduktionen und Diskussionsbeiträge sind ohne weitere
Formalien willkommen. Die Vereinbarung im nächsten Abschnitt betrifft nur
eingereichten Inhalt.

## Lizenz und Rechte

Buchfink steht unter der [EUPL-1.2](LICENSE); die Lizenzdatei enthält den
deutschen und den englischen Wortlaut. Dein Beitrag wird unter derselben
Lizenz veröffentlicht. Einzelne Quelldateien tragen keinen Lizenzkopf — es
gilt die Lizenz des Projekts.

Zusätzlich brauchen wir deine Zustimmung zur
[Vereinbarung über Beiträge (CLA)](CLA.md). Du behältst das Urheberrecht an
deinem Beitrag; die Vereinbarung erlaubt dem Projektinhaber, das Projekt
langfristig aus einer Hand zu lizenzieren. Schreibe dazu einmalig in deinen
Pull Request:

```text
Ich habe die CLA gelesen und stimme ihr für diesen und alle meine
künftigen Beiträge zu.
```

Für alle weiteren Beiträge genügt diese eine Zustimmung.

**Fremdes Material:** Kopiere keinen Code, keine Texte, keine Grafiken und
keine Daten aus fremden Quellen ins Repository, ohne die Lizenz zu prüfen und
im Pull Request zu nennen. Das gilt auch für Auszüge aus Fachliteratur,
Kontenrahmen-Veröffentlichungen und Formularen. Kommt neues Material dazu,
gehört es in [`THIRD-PARTY-NOTICES.md`](THIRD-PARTY-NOTICES.md).

## Entwicklungsumgebung

Voraussetzungen, Start und Prüfungen stehen unter
[Entwicklung](docs/entwicklung/README.md). Die [Orientierung im Code](docs/entwicklung/codebase.md)
zeigt, wo du die betroffenen Module findest.

## Vor dem Pull Request

Führe die [Prüfungen für deine Änderung](docs/entwicklung/README.md#änderungen-prüfen)
aus und behebe die Befunde. Nenne im Pull Request, was du geprüft hast und
welche Prüfungen du nicht ausführen konntest.

Nach Änderungen an Abhängigkeiten erzeuge die Hinweise zu Drittkomponenten neu:

```bash
task notices
```

Neue Abhängigkeiten müssen mit der EUPL vereinbar sein. Permissive Lizenzen
(MIT, BSD, ISC, Apache-2.0) sind unproblematisch. Alles mit Copyleft — GPL,
AGPL, MPL — bitte vorher im Issue ansprechen: solche Abhängigkeiten können
über die Kompatibilitätsklausel der EUPL die Lizenz des Gesamtwerks
verschieben.

## Tests

Buchungslogik, Hash-Chain, Steuerkennzeichen, Parser und Exportformate brauchen
Tests. Orientiere dich an den vorhandenen Tests unter `internal/service/` und
`internal/accounting/`. Bei fachlichen Änderungen gehört in den Pull Request,
worauf sie sich stützt — Paragraph, GoBD-Randziffer oder Taxonomie-Version.

## Dokumentation und Texte

Schreibe Dokumentation und Codekommentare auf Deutsch. Pflege Funktionen und
Grenzen im [Umsetzungsstand](docs/projekt/umsetzungsstand.md), fachliche Details im
[passenden Konzept](docs/fachkonzepte/README.md). Für Formulierungen gilt die
[Schreibweise](docs/gestaltung/schreibweise.md), für Ablage und Verweise die
[Dokumentationspflege](docs/entwicklung/dokumentation.md).

## Commits und Pull Requests

- Commit-Betreff und PR-Titel folgen Conventional Commits, etwa
  `docs(readme): Straffe den Projekteinstieg` oder
  `fix(accounting): Korrigiere die Rundung`. Formuliere die Zusammenfassung
  auf Deutsch und im Imperativ.
- Halte jeden Commit auf eine zusammenhängende Änderung begrenzt.
- Der Text erklärt das *Warum*; das *Was* steht im Diff.
- Der Pull Request beschreibt Motivation, Lösungsweg und wie du geprüft hast,
  dass es funktioniert.
- Formatierung und inhaltliche Änderung nicht im selben Commit mischen.

## Was besser nicht kommt

- Umstellungen auf andere Frameworks oder Bibliotheken ohne vorherige
  Abstimmung.
- Eine verpflichtende Cloud-Speicherung der Buchhaltung. Die Datenhaltung ist
  lokal; bestehende externe Dienste sind im
  [Sicherheitskonzept](docs/nutzung/datensicherheit.md) beschrieben.
- Unterstützung für die Einnahmen-Überschuss-Rechnung (EÜR). Die ist bewusst
  außerhalb des Anwendungsbereichs.
