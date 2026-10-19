# Rumpfgeschäftsjahr und erste Fristen

Rechtsstand: 17. Oktober 2026. Wortlaut von HGB, GmbHG, EStG, EStDV, KStG,
GewStG, UStG und AO am 17.10.2026 gelesen.

## Inhalt

1. Was ein Rumpfgeschäftsjahr ist
2. In Buchfink
3. Pflichten zum Ende des ersten Geschäftsjahres
4. Beispiel mit Terminen

## 1. Was ein Rumpfgeschäftsjahr ist

- Ein Geschäftsjahr dauert höchstens zwölf Monate
  ([grundlagen/arbeitsweise.md, Abschnitt 10](../grundlagen/arbeitsweise.md#10-geschäftsjahr-und-zeiträume)).
  Ein erstes Geschäftsjahr, das länger liefe, ist deshalb nicht zulässig.
- Steuerlich umfasst das Wirtschaftsjahr zwölf Monate; kürzer darf es sein,
  wenn ein Betrieb eröffnet wird ([§ 8b Satz 1, 2 Nr. 1 EStDV][estdv8b]).
- Beginnt die GmbH ihre Tätigkeit im Laufe eines Jahres und endet das
  Geschäftsjahr laut Vertrag am 31.12., ist das erste Geschäftsjahr ein
  Rumpfgeschäftsjahr.
- Ein vom Kalenderjahr abweichendes Geschäftsjahr kann bei der Gründung gewählt
  werden; erst eine spätere Umstellung braucht das Einvernehmen des Finanzamts
  ([§ 4a Abs. 1 Satz 2 Nr. 2 EStG][estg4a], [§ 7 Abs. 4 Satz 3 KStG][kstg7]).

## 2. In Buchfink

- Buchfink beginnt das erste Geschäftsjahr am Beurkundungstag, wenn der Mensch
  die Gründung im Einrichtungsassistenten erfasst. Berichtigt er das
  Gründungsdatum später in den Stammdaten, zieht Buchfink den Beginn nach,
  solange der Abschluss nicht festgestellt, kein Zeitraum festgeschrieben und
  keine Buchung vor dem neuen Beginn vorhanden ist.
- Prüfe mit `status`: `geschaeftsjahre[]` mit `von` gleich Beurkundungstag und
  `bis` gleich Ende laut Vertrag. Weicht es ab, sag es dem Menschen; ein
  Werkzeug zum Ändern gibt es nicht.

## 3. Pflichten zum Ende des ersten Geschäftsjahres

| Pflicht | Im ersten Geschäftsjahr | Regel |
|---|---|---|
| Inventar und Jahresabschluss, Aufstellung, Feststellung und Ergebnisverwendung, Offenlegung | Fristen wie in jedem Jahr, gerechnet ab dem Ende des Rumpfgeschäftsjahres | [bilanzstichtag](../grundlagen/pflichten.md#bilanzstichtag), [aufstellung](../grundlagen/pflichten.md#aufstellung), [feststellung](../grundlagen/pflichten.md#feststellung), [ergebnisverwendung](../grundlagen/pflichten.md#ergebnisverwendung), [offenlegung](../grundlagen/pflichten.md#offenlegung) |
| Größenklasse | Merkmale schon am ersten Abschlussstichtag | [jahresabschluss/groessenklassen-anhang.md](../jahresabschluss/groessenklassen-anhang.md) |
| UG: gesetzliche Rücklage | schon aus dem ersten Jahresüberschuss | [jahresabschluss/ergebnisverwendung-ug.md](../jahresabschluss/ergebnisverwendung-ug.md) |
| Abschreibung | zeitanteilig im Anschaffungsjahr | [jahresabschluss/abschreibungen.md](../jahresabschluss/abschreibungen.md) |
| Körperschaftsteuer | Zeitraum der Steuerpflicht statt Kalenderjahr | [steuererklaerungen/kst.md, Abschnitt 1](../steuererklaerungen/kst.md#1-steuerpflicht-und-zeitraum) |
| Gewerbesteuer | abgekürzter Erhebungszeitraum | [steuererklaerungen/gewst.md, Abschnitt 1](../steuererklaerungen/gewst.md#1-steuerpflicht-erklärung-bescheide) |
| Umsatzsteuer | der Teil des Jahres tritt an die Stelle des Kalenderjahres; Frist im Gründungsjahr | [steuererklaerungen/ust-jahr.md, Abschnitt 1](../steuererklaerungen/ust-jahr.md#1-pflicht-zeitraum-frist) |
| E-Bilanz | auch für die Eröffnungsbilanz | [steuererklaerungen/ebilanz.md, Abschnitt 6](../steuererklaerungen/ebilanz.md#6-eröffnungsbilanz), [eroeffnungsbilanz.md, Abschnitt 5](eroeffnungsbilanz.md#5-unterzeichnen-aufbewahren-übermitteln) |

Stand: Oktober 2026. Den Abschluss begleitet
[jahresabschluss/ablauf.md](../jahresabschluss/ablauf.md), die Erklärungen
[steuererklaerungen/ablauf.md](../steuererklaerungen/ablauf.md).

## 4. Beispiel mit Terminen

Rechnerisches Beispiel: Beurkundung am 03.11.2026, Geschäftsjahr gleich
Kalenderjahr, kleine GmbH, ohne Steuerberater.

| Termin | Datum | Grundlage |
|---|---|---|
| Fragebogen zur steuerlichen Erfassung | bis 03.12.2026, gerechnet ab der Beurkundung | [§ 137 Abs. 2 AO][ao137], [§ 138 Abs. 4 AO][ao138] |
| Ende des Rumpfgeschäftsjahres | 31.12.2026 | Vertrag, [§ 8b EStDV][estdv8b] |
| Aufstellung des Abschlusses | bis 30.06.2027 | [§ 264 Abs. 1 Satz 4 HGB][hgb264] |
| Steuererklärungen 2026 | bis 02.08.2027 (31.07.2027 ist ein Samstag) | [§ 149 Abs. 2 AO][ao149], [§ 108 Abs. 3 AO][ao108] |
| Feststellung | bis 30.11.2027 | [§ 42a Abs. 2 GmbHG][gmbhg42a] |
| Offenlegung | bis 31.12.2027 | [§ 325 Abs. 1a HGB][hgb325] |

Fristen nach Monaten rechnest du nach den Regeln des BGB
([§ 108 Abs. 1 AO][ao108]); Fristende an Wochenende oder Feiertag:
[pflichten.md#fristende-zahlung-verlängerung](../grundlagen/pflichten.md#fristende-zahlung-verlängerung).
Die Fälligkeiten des Mandanten nennt `fristen`; mit beauftragtem
Steuerberater liegen die Steuererklärungen später.

[ao108]: https://www.gesetze-im-internet.de/ao_1977/__108.html
[ao137]: https://www.gesetze-im-internet.de/ao_1977/__137.html
[ao138]: https://www.gesetze-im-internet.de/ao_1977/__138.html
[ao149]: https://www.gesetze-im-internet.de/ao_1977/__149.html
[estdv8b]: https://www.gesetze-im-internet.de/estdv_1955/__8b.html
[estg4a]: https://www.gesetze-im-internet.de/estg/__4a.html
[gmbhg42a]: https://www.gesetze-im-internet.de/gmbhg/__42a.html
[hgb264]: https://www.gesetze-im-internet.de/hgb/__264.html
[hgb325]: https://www.gesetze-im-internet.de/hgb/__325.html
[kstg7]: https://www.gesetze-im-internet.de/kstg_1977/__7.html
