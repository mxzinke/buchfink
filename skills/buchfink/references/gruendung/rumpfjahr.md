# Rumpfgeschäftsjahr und erste Fristen

Rechtsstand: 17. Oktober 2026.

## Inhalt

1. Was ein Rumpfgeschäftsjahr ist
2. In Buchfink
3. Pflichten zum Ende des ersten Geschäftsjahres

## 1. Was ein Rumpfgeschäftsjahr ist

- Ein Geschäftsjahr dauert höchstens zwölf Monate
  ([grundlagen/arbeitsweise.md, Abschnitt 10](../grundlagen/arbeitsweise.md#10-geschäftsjahr-und-zeiträume));
  bei Eröffnung des Betriebs darf das Wirtschaftsjahr kürzer sein
  ([§ 8b Satz 1, 2 Nr. 1 EStDV][estdv8b]). Beginnt die GmbH im Laufe des
  Jahres und endet das Geschäftsjahr laut Vertrag am 31.12., ist das erste ein
  Rumpfgeschäftsjahr.
- Ein abweichendes Geschäftsjahr kann bei der Gründung frei gewählt werden;
  eine spätere Umstellung braucht das Einvernehmen des Finanzamts
  ([§ 4a Abs. 1 Satz 2 Nr. 2 EStG][estg4a], [§ 7 Abs. 4 Satz 3 KStG][kstg7]).

## 2. In Buchfink

Das erste Geschäftsjahr beginnt am Beurkundungstag und endet laut Vertrag;
prüfe das mit `status` (`geschaeftsjahre[]`). Weicht es ab, berichtigt der
Mensch das Gründungsdatum in der App. Buchfink zieht den Beginn nach, solange
der Abschluss nicht festgestellt, kein Zeitraum festgeschrieben und keine
Buchung vor dem neuen Beginn vorhanden ist.

## 3. Pflichten zum Ende des ersten Geschäftsjahres

| Pflicht | Im ersten Geschäftsjahr | Regel |
|---|---|---|
| Inventar und Jahresabschluss, Aufstellung, Feststellung und Ergebnisverwendung, Offenlegung | Fristen wie in jedem Jahr, ab dem Ende des Rumpfgeschäftsjahres | [bilanzstichtag](../grundlagen/pflichten.md#bilanzstichtag), [aufstellung](../grundlagen/pflichten.md#aufstellung), [feststellung](../grundlagen/pflichten.md#feststellung), [ergebnisverwendung](../grundlagen/pflichten.md#ergebnisverwendung), [offenlegung](../grundlagen/pflichten.md#offenlegung) |
| Größenklasse | Merkmale schon am ersten Abschlussstichtag | [jahresabschluss/groessenklassen-anhang.md](../jahresabschluss/groessenklassen-anhang.md) |
| UG: gesetzliche Rücklage | schon aus dem ersten Jahresüberschuss | [jahresabschluss/ergebnisverwendung-ug.md](../jahresabschluss/ergebnisverwendung-ug.md) |
| Abschreibung | zeitanteilig im Anschaffungsjahr | [jahresabschluss/abschreibungen.md](../jahresabschluss/abschreibungen.md) |
| Körperschaftsteuer | Zeitraum der Steuerpflicht statt Kalenderjahr | [steuererklaerungen/kst.md, Abschnitt 1](../steuererklaerungen/kst.md#1-steuerpflicht-und-zeitraum) |
| Gewerbesteuer | abgekürzter Erhebungszeitraum | [steuererklaerungen/gewst.md, Abschnitt 1](../steuererklaerungen/gewst.md#1-steuerpflicht-erklärung-bescheide) |
| Umsatzsteuer | Teil des Jahres statt Kalenderjahr; Frist im Gründungsjahr | [steuererklaerungen/ust-jahr.md, Abschnitt 1](../steuererklaerungen/ust-jahr.md#1-pflicht-zeitraum-frist) |
| E-Bilanz | auch für die Eröffnungsbilanz | [steuererklaerungen/ebilanz.md, Abschnitt 6](../steuererklaerungen/ebilanz.md#6-eröffnungsbilanz), [eroeffnungsbilanz.md, Abschnitt 5](eroeffnungsbilanz.md#5-unterzeichnen-aufbewahren-übermitteln) |

Die Termine des Mandanten nennt `fristen`. Monatsfristen laufen nach dem BGB
([§ 108 Abs. 1 AO][ao108]); Fristende am Wochenende oder Feiertag:
[pflichten.md#fristende-zahlung-verlängerung](../grundlagen/pflichten.md#fristende-zahlung-verlängerung).
Den Abschluss begleitet
[jahresabschluss/ablauf.md](../jahresabschluss/ablauf.md), die Erklärungen
[steuererklaerungen/ablauf.md](../steuererklaerungen/ablauf.md).

[ao108]: https://www.gesetze-im-internet.de/ao_1977/__108.html
[estdv8b]: https://www.gesetze-im-internet.de/estdv_1955/__8b.html
[estg4a]: https://www.gesetze-im-internet.de/estg/__4a.html
[kstg7]: https://www.gesetze-im-internet.de/kstg_1977/__7.html
