<!-- SPDX-FileCopyrightText: 2026 Maximilian Pfennig -->
<!-- SPDX-License-Identifier: EUPL-1.2 -->

# Mitwirken an Buchfink

Danke für dein Interesse. Buchfink ist Buchhaltungssoftware — Fehler haben hier
handfeste Folgen für Jahresabschluss und Betriebsprüfung. Entsprechend genau
schauen wir auf Änderungen, die Buchungslogik, Hash-Chain, Steuerkennzeichen
oder Exportformate berühren.

## Bevor du loslegst

Für alles, was größer ist als ein Tippfehler: öffne zuerst ein Issue und
beschreibe das Problem. Das erspart dir die Arbeit an einem Pull Request, der
inhaltlich in eine andere Richtung soll.

Fehlerberichte, Reproduktionen und Diskussionsbeiträge sind ohne weitere
Formalien willkommen. Die Vereinbarung im nächsten Abschnitt betrifft nur
eingereichten Inhalt.

## Lizenz und Rechte

Buchfink steht unter der [EUPL-1.2](LICENSE) (deutscher Lizenztext:
[`docs/lizenz-eupl-1.2-de.txt`](docs/lizenz-eupl-1.2-de.txt)). Dein Beitrag
wird unter derselben Lizenz veröffentlicht.

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

## Lizenzkopf in neuen Dateien

Jede Quelldatei trägt oben zwei Zeilen — im jeweils passenden Kommentarstil:

```go
// SPDX-FileCopyrightText: 2026 Maximilian Pfennig
// SPDX-License-Identifier: EUPL-1.2
```

Generierte Dateien (`frontend/bindings/`) sind davon ausgenommen.

## Entwicklungsumgebung

Voraussetzungen und Setup stehen im [README](README.md#entwicklung--weiterentwicklung).

## Vor dem Pull Request

Lass diese Prüfungen durchlaufen und behebe, was sie melden:

```bash
go build ./...
go vet ./...
go test ./...

cd frontend
npx tsc --noEmit
npm run build
```

Wenn du Abhängigkeiten hinzugefügt, entfernt oder aktualisiert hast, erzeuge
die Hinweise zu Drittkomponenten neu:

```bash
go mod download
cd frontend && npm install && cd ..
python3 scripts/gen_third_party_notices.py
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

## Commits und Pull Requests

- Eine Änderung pro Commit, Betreffzeile im Imperativ und auf Deutsch.
- Der Text erklärt das *Warum*; das *Was* steht im Diff.
- Der Pull Request beschreibt Motivation, Lösungsweg und wie du geprüft hast,
  dass es funktioniert.
- Formatierung und inhaltliche Änderung nicht im selben Commit mischen.

## Was besser nicht kommt

- Umstellungen auf andere Frameworks oder Bibliotheken ohne vorherige
  Abstimmung.
- Cloud-Abhängigkeiten. Buchfink ist local-first; Daten bleiben auf dem Rechner
  des Anwenders.
- Unterstützung für die Einnahmen-Überschuss-Rechnung (EÜR). Die ist bewusst
  außerhalb des Anwendungsbereichs.
