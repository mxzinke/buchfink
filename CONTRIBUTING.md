# Mitwirken an Buchfink

[Projektübersicht](README.md) · [Dokumentation](docs/README.md)

Beiträge sind Fehlerberichte, Rückmeldungen zur Bedienung und zur Arbeit mit
dem Agenten, verständlichere Texte, fachliche Prüfungen, Skills und Code.

## Fehler melden und Rückmeldung geben

Öffnen Sie ein [Issue](https://github.com/mxzinke/buchfink/issues) und
beschreiben Sie, was Sie erreichen wollten, welche Schritte Sie ausgeführt
haben und was stattdessen passiert ist. Nennen Sie Programmversion,
Betriebssystem und, wenn ein Agent beteiligt war, den KI-Client. Bei fachlichen
Fragen hilft ein konkreter Geschäftsvorfall mit erwartetem Ergebnis. Verwenden
Sie erfundene Daten und entfernen Sie persönliche und geschäftliche Angaben aus
Screenshots, Chatverläufen und Protokollen.

Für alles, was größer ist als ein Tippfehler, öffnen Sie zuerst ein Issue. Das
erspart einen Pull Request, der inhaltlich in eine andere Richtung soll.

## Lizenz und Rechte

Buchfink steht unter der [EUPL-1.2](LICENSE); die Lizenzdatei enthält den
deutschen und den englischen Wortlaut. Ihr Beitrag erscheint unter derselben
Lizenz. Quelldateien tragen keinen Lizenzkopf; es gilt die Lizenz des Projekts.

Für eingereichten Inhalt braucht es zusätzlich Ihre Zustimmung zur
[Vereinbarung über Beiträge (CLA)](CLA.md). Sie behalten das Urheberrecht; die
Vereinbarung erlaubt dem Projektinhaber, das Projekt aus einer Hand zu
lizenzieren. Schreiben Sie dazu einmalig in Ihren Pull Request:

```text
Ich habe die CLA gelesen und stimme ihr für diesen und alle meine
künftigen Beiträge zu.
```

Fehlerberichte und Diskussionsbeiträge brauchen keine Zustimmung.

**Fremdes Material.** Kopieren Sie keinen Code, keine Texte, Grafiken oder
Daten aus fremden Quellen ins Repository, ohne die Lizenz zu prüfen und im Pull
Request zu nennen. Das gilt auch für Fachliteratur, Kontenrahmen und Vordrucke.
Neues Material gehört in [THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md).

## Code

Einrichtung, Build und Prüfungen stehen unter
[Entwicklung](docs/entwicklung/README.md), der Aufbau unter
[Orientierung im Code](docs/entwicklung/codebase.md). Vor dem Pull Request:

```sh
task check
npm --prefix frontend test
npm --prefix frontend run build
```

Unter Linux mit GTK 3 brauchen Bridge-Tests und Build das Tag `gtk3`
(`GOFLAGS=-tags=gtk3 task test:bridge`, `task build EXTRA_TAGS=gtk3`). Nennen
Sie im Pull Request, was Sie geprüft haben und was nicht.

Buchungslogik, Hash-Ketten, Steuerschlüssel, Parser, Exporte und die Regeln für
Agenten im Kern brauchen Tests. Bei fachlichen Änderungen gehört in den Pull
Request, worauf sie sich stützen: Paragraph, GoBD-Randziffer, BMF-Schreiben
oder Taxonomie-Fassung.

Nach Änderungen an Abhängigkeiten erzeugt `task notices` die Hinweise zu
Drittkomponenten neu. Neue Abhängigkeiten müssen mit der EUPL vereinbar sein.
Permissive Lizenzen (MIT, BSD, ISC, Apache-2.0) sind unproblematisch;
Copyleft-Lizenzen wie GPL, AGPL oder MPL sprechen Sie bitte vorher im Issue an.

## Skills

Der Skill unter `skills/buchfink/` gibt dem Agenten Fachwissen und
Arbeitsschritte: `SKILL.md` mit den Grundregeln und einem Wegweiser, je
Fachbereich Referenzen unter `references/<bereich>/`. Jede fachliche Aussage
darin nennt ihre Quelle, etwa Gesetz, BMF-Schreiben oder Vordruck; das
Quellenregister ist `skills/QUELLEN.md`. Was Buchfink selbst rechnet und
prüft, steht im Code, nicht im Skill.

## Dokumentation

Dokumentation und Codekommentare sind deutsch und beschreiben nur den aktuellen
Stand. Funktionen und Grenzen stehen im
[Umsetzungsstand](docs/projekt/umsetzungsstand.md), fachliche Einzelheiten im
[passenden Konzept](docs/fachkonzepte/README.md). Für Formulierungen gilt die
[Schreibweise](docs/gestaltung/schreibweise.md), für Ablage und Verweise die
[Dokumentationspflege](docs/entwicklung/dokumentation.md).

## Commits und Pull Requests

- Betreff und PR-Titel folgen Conventional Commits, etwa
  `fix(accounting): Rundung der Skontokorrektur korrigieren`. Die
  Zusammenfassung ist deutsch.
- Ein Commit enthält eine zusammenhängende Änderung. Formatierung und Inhalt
  stehen in getrennten Commits.
- Der Text erklärt das *Warum*: Entscheidung, Begründung, Abwägung. Das *Was*
  steht im Diff.
- Der Pull Request beschreibt Motivation, Lösung und wie Sie geprüft haben, dass
  es funktioniert.

## Was nicht in Buchfink gehört

- andere Rechtsformen und die Einnahmen-Überschuss-Rechnung
- eine verpflichtende Cloud-Speicherung der Buchhaltung
- eigene Übermittlung an die Finanzverwaltung
- Umstellungen auf andere Frameworks ohne vorherige Abstimmung

Was sonst außerhalb des Umfangs liegt, steht unter
[Umfang](docs/nutzung/umfang.md).
