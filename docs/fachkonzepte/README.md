# Fachkonzepte und Anforderungen

[Dokumentation](../README.md) · [Orientierung im Code](../entwicklung/codebase.md)

Der [Umsetzungsstand](../projekt/umsetzungsstand.md) beschreibt verfügbare Funktionen.
Die Konzepte hier erklären fachliche Abläufe und enthalten auch Entwürfe;
eine beschriebene Funktion ist damit noch nicht verfügbar.

## Einen Ablauf verstehen

| Thema | Konzept | Ergänzung |
|---|---|---|
| Beleg erfassen, buchen und bezahlen | [Beleg- und Buchungsflow](beleg-buchungsflow.md) | [Architektur und Bedienkonzept](../entwicklung/architektur.md) |
| Buchungssätze an Beispielen nachvollziehen | [Buchungsbeispiele](buchungsbeispiele.md) | [Prüfszenarien](../entwicklung/pruefszenarien.md) |
| Belege und Dateien zuordnen | [Belegmodell](belege.md) | [Datensicherheit](../nutzung/datensicherheit.md) |
| E-Rechnung empfangen oder ausstellen | [E-Rechnung](e-rechnung.md) | [Technischer Modulaufbau](../entwicklung/e-rechnung.md) |
| Abschlag und Schlussrechnung | [Anzahlungen](anzahlungen.md) | [Beleg- und Buchungsflow](beleg-buchungsflow.md) |
| Anlagegut und Abschreibung | [Anlagenverwaltung](anlagenverwaltung.md) | [Anforderungskatalog](../anforderungen/bewertung.md#g-bewertung-anlagen-fremdwährung) |
| Aufwand und Ertrag zeitlich zuordnen | [Rechnungsabgrenzung](rechnungsabgrenzung.md) | [Anforderungskatalog](../anforderungen/bewertung.md#g-bewertung-anlagen-fremdwährung) |
| Kapitalgesellschaft gründen | [Gründung](gruendung.md) | [Prüfszenarien](../entwicklung/pruefszenarien.md) |
| DATEV-Export entwerfen | [DATEV-Export](datev-export.md) | Nicht implementiert; offene Entscheidungen im Entwurf |

## Anforderungen prüfen

Der [Anforderungskatalog](../anforderungen/README.md) enthält Kriterien,
Rechtsquellen und Fundstellen im Code. Über sein Themenverzeichnis gelangst
du direkt zum betreffenden Modul. Sein Geltungsbereich beschreibt das
Prüfraster und geht über den aktuell unterstützten Funktionsumfang hinaus.

Für eine fachliche Änderung prüfe das betroffene Kriterium gegen Code und
Tests. Nenne im Beitrag die Fundstelle und den Geschäftsvorfall. Konkrete
Beispiele und erwartete Ergebnisse stehen in den [Prüfszenarien](../entwicklung/pruefszenarien.md).
Offene Entwicklungsziele stehen auf der [Roadmap](../projekt/roadmap.md).
