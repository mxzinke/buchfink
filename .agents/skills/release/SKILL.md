---
name: release
description: "Release von Buchfink erstellen und Release Notes schreiben: Fassung festlegen, Eintrag in internal/changelog/CHANGELOG.md, prüfen, taggen, bauen, auf GitHub veröffentlichen. Bei: Release, neue Fassung oder Version, Tag, Release Notes, Changelog-Eintrag."
---

# Release

Ein Release läuft von Hand über `scripts/release.sh` und das Taskfile; Einzelheiten in
`docs/entwicklung/README.md` („Release“). Tags heißen `vX.Y.Z`, eine Vorabversion `vX.Y.Z-rc.N`.

## Ablauf

1. **Fassung festlegen.** Patch für Fehlerbehebungen, Minor für neue Funktionen, Pflichten oder
   Werkzeuge. Was ein Mensch beim Aktualisieren tun muss (Mandant neu anlegen, App neu verbinden),
   macht die Fassung zur Minor-Fassung und gehört in die Zusammenfassung.
2. **Release Notes schreiben** (unten). Fertig, wenn jeder Commit seit dem letzten Tag entweder
   in einem Strichpunkt steckt oder nichts für Anwender ändert.
3. `python3 scripts/version.py set X.Y.Z`, Eintrag und Paketdateien committen:
   `chore(release): vX.Y.Z`.
4. `task release:check VERSION=X.Y.Z`. Fertig, wenn es mit „Alle Prüfungen … bestanden“ endet.
   Rot heißt: beheben, committen, erneut prüfen.
5. `git tag -a vX.Y.Z -m "Buchfink X.Y.Z"`.
6. `task release:build VERSION=X.Y.Z`, dann `task release:build:linux VERSION=X.Y.Z` (auf dem Mac
   muss Docker bzw. OrbStack laufen). Fertig, wenn `dist/release/vX.Y.Z/` neun Dateien enthält.
7. **Erst nach ausdrücklichem Ja des Menschen:** `git push origin master vX.Y.Z`, dann
   `task release:publish VERSION=X.Y.Z`. Fertig, wenn `gh release view vX.Y.Z` zehn Dateien zeigt
   (die neun und `SHA256SUMS`).

Ein noch nicht gepushter Tag lässt sich nach einer Korrektur löschen und neu setzen. Ein
gepushter Tag bleibt; Korrekturen kommen in die nächste Fassung.

## Release Notes

Der Eintrag in `internal/changelog/CHANGELOG.md` ist zugleich Text des GitHub-Release, steht in
der App unter „Systemhistorie“ und liegt dem Prüferpaket bei. Leser sind Anwender und Prüfer;
er sagt, was die Fassung für die Buchführung ändert. Das Format liest `internal/changelog`:

```markdown
## vX.Y.Z — JJJJ-MM-TT

Ein Absatz: wofür diese Fassung steht und was beim Aktualisieren zu tun ist.

- Eine Änderung je Strichpunkt, in ganzen Sätzen.
```

- Quelle ist `git log --no-merges <letzter Tag>..HEAD`. Aus den Conventional Commits zählen
  `feat` und `fix`; `refactor`, `test`, `chore`, `build` und `docs` nur, wenn Anwender oder Agent
  etwas davon merken.
- Zusammenfassen nach Wirkung, nicht nach Commit: mehrere Commits zu einer Sache sind ein
  Strichpunkt. Neues zuerst, dann Korrekturen, dann bekannte Grenzen.
- Fachlich benennen, wie in der App: „Voranmeldung“, „Pflichtenkalender“, Kontonummer und
  Rechtsgrundlage, wo sie die Änderung erklären. Werkzeugnamen nur bei Änderungen für Agenten.
  Keine Commit-Hashes, Dateipfade oder Go-Bezeichner.
- Eine Korrektur sagt, was vorher falsch war und ab wann es stimmt, damit ein Prüfer betroffene
  Buchungen findet.
- Schreibweise nach `docs/gestaltung/schreibweise.md`.
