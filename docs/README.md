# Dokumentation

Buchfink richtet sich an kleine Unternehmen, Gründer und Holdings, die ihre
Buchhaltung selbst führen wollen. Die Webseite erklärt den Nutzen. Die
Dokumente hier beschreiben die Umsetzung, fachliche Anforderungen und die
Entscheidungen für Mitwirkende.

| Frage | Dokument |
|---|---|
| Was kann die aktuelle Version, und wo liegen ihre Grenzen? | [Stand der Umsetzung](stand-der-umsetzung.md) |
| Was soll Buchfink künftig ermöglichen? | [Ziele und nächste Schritte](roadmap.md) |
| Was wurde beim Abgleich von Text und Code berichtigt? | [Dokumentationsprüfung vom 11. September 2026](dokumentationspruefung-2026-09-11.md) |
| Welche Anforderungen werden fachlich geprüft? | [Anforderungskatalog](anforderungskatalog.md) |
| Wie ist die Anwendung aufgebaut? | [Architektur](architektur.md), [E-Rechnungs-Module](architektur-e-rechnung.md) |
| Wie werden Daten geschützt und gesichert? | [Sicherheitskonzept](security-concept.md) |
| Wie schreiben und gestalten wir? | [Schreibweise](schreibweise.md), [Designkonzept](design-konzept.md) |
| Wie lässt sich der Weg bis zum Beleg prüfen? | [Prüfszenario](pruefszenario.md) |

## Fachkonzepte

Die folgenden Dokumente erklären Anforderungen und fachliche Entscheidungen.
Ihr Entwurfsdatum ist kein Nachweis über den aktuellen Funktionsumfang.
Bei Abweichungen muss die Beschreibung anhand des Codes und der Tests geprüft
werden; eine geplante Funktion wird nicht durch ihre Beschreibung verfügbar.

- [Beleg- und Buchungsflow](anforderung-beleg-buchungsflow.md)
- [E-Rechnung](anforderung-e-rechnung.md)
- [Anzahlungen](anforderung-anzahlungen.md)
- [Anlagenverwaltung](anforderung-anlagenverwaltung.md)
- [Rechnungsabgrenzung](anforderung-rechnungsabgrenzung.md)
- [Gründung](anforderung-gruendung.md)
- [DATEV-Export, nicht implementierter Entwurf](anforderung-datev-export.md)

## Pflege

Den Ist-Zustand beschreiben wir im Präsens und mit einer Fundstelle im Code.
Ziele stehen auf der Roadmap. Abgeschlossene Entwicklungsschritte gehören in
den Änderungsverlauf, nicht als Zukunftsversprechen in Codekommentare.
Kommentare erklären das aktuelle Verhalten und dessen Gründe, einschließlich
Rechtsgrundlage, Ausnahmen und fachlicher Einschränkungen.

Bei geänderten Funktionen werden Umsetzungsstand, betroffene Anforderungen und
öffentliche Beschreibung zusammen abgeglichen. Quellenlinks führen zum
Gesetzestext oder zur Originalveröffentlichung. Kriterienzahlen werden nur im
Anforderungskatalog gepflegt; sie sind keine Zusicherung der Gesetzeskonformität.
