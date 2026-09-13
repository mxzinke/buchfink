# Dokumentation pflegen

[Dokumentation](../README.md) · [Mitwirken](../../CONTRIBUTING.md)

Wir pflegen diese Dokumente nach Thema. Neue Funktionen und Befunde ergänzen
die passende bestehende Seite. Ein neues Dokument braucht ein eigenständiges
Thema, das dauerhaft nachgeschlagen wird. Für einzelne Arbeitsschritte,
Prüfläufe oder erledigte Korrekturen legen wir keine neuen Berichte an.

Versionsänderungen stehen in der Änderungshistorie, offene Arbeiten in der
Roadmap und bekannte Grenzen im Umsetzungsstand. Prüfanleitungen stehen bei
den Prüfszenarien. Aufnahmen, Exporte und Laufprotokolle bleiben unter `.cache/`
außerhalb von Git; frühere Arbeitsberichte sind im Git-Verlauf erhalten.

Den Ist-Zustand beschreiben wir im Präsens und mit einer Fundstelle im Code.
Ziele stehen auf der Roadmap. Abgeschlossene Entwicklungsschritte gehören in
den Änderungsverlauf, nicht als Zukunftsversprechen in Codekommentare.
Kommentare erklären das aktuelle Verhalten und dessen Gründe, einschließlich
Rechtsgrundlage, Ausnahmen und fachlicher Einschränkungen.

Bei geänderten Funktionen werden Umsetzungsstand, betroffene Anforderungen und
öffentliche Beschreibung zusammen abgeglichen. Quellenlinks führen zum
Gesetzestext oder zur Originalveröffentlichung. Kriterienzahlen werden nur im
Anforderungskatalog gepflegt; sie sind keine Zusicherung der Gesetzeskonformität.

## Ablage und Zuständigkeit

| Inhalt | Ort |
|---|---|
| Installation und erste Nutzung | [nutzung/](../nutzung/README.md) |
| Verfügbare Funktionen und Grenzen | [Umsetzungsstand](../projekt/umsetzungsstand.md) |
| Geplante Arbeiten | [Roadmap](../projekt/roadmap.md) |
| Setup, Architektur und Prüfanleitungen | [entwicklung/](README.md) |
| Visuelle Regeln und Sprache | [gestaltung/](../gestaltung/README.md) |
| Fachliche Abläufe und Beispiele | [fachkonzepte/](../fachkonzepte/README.md) |
| Einzelkriterien mit Quellen und Codebezug | [anforderungen/](../anforderungen/README.md) |

Jeder Inhalt hat einen Pflegeort. Übersichtsseiten verlinken dorthin;
Funktionslisten, Kriterien und Regeln werden nicht in andere Seiten kopiert.
Die README im Repository bleibt ein kurzer Einstieg. Anleitungen zu einzelnen
Werkzeugen bleiben bei deren Code, etwa unter `scripts/` oder `website/`.

Beim Verschieben werden relative Links und Abschnittsverweise im gesamten
Repository angepasst, einschließlich Webseite, Codekommentaren und Prüfskripten.
