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
  ([§ 240 Abs. 2 Satz 2 HGB][hgb240]). Ein erstes Geschäftsjahr, das länger
  liefe, ist nicht zulässig.
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
  die Gründung im Einrichtungsassistenten erfasst. Den Beginn zieht Buchfink nur
  nach, solange der Abschluss nicht festgestellt, kein Zeitraum festgeschrieben
  und keine Buchung vor dem Beurkundungstag vorhanden ist.
- Prüfe mit `status`: `geschaeftsjahre[]` mit `von` gleich Beurkundungstag und
  `bis` gleich Ende laut Vertrag. Weicht es ab, sag es dem Menschen; ein
  Werkzeug zum Ändern gibt es nicht.

## 3. Pflichten zum Ende des ersten Geschäftsjahres

| Pflicht | Regel | Grundlage |
|---|---|---|
| Inventar und Jahresabschluss | zum Schluss des Geschäftsjahres | [§ 240 Abs. 2 Satz 1 HGB][hgb240], [§ 242 Abs. 1, 2 HGB][hgb242] |
| Aufstellung | in den ersten drei Monaten des neuen Geschäftsjahres; kleine Kapitalgesellschaft bis sechs Monate, wenn das einem ordnungsgemäßen Geschäftsgang entspricht | [§ 264 Abs. 1 Satz 3, 4 HGB][hgb264] |
| Größenklasse | bei Neugründung entscheiden schon die Merkmale am ersten Abschlussstichtag | [§ 267 Abs. 4 Satz 2 HGB][hgb267] |
| Feststellung und Ergebnisverwendung | binnen acht Monaten, kleine Gesellschaft elf Monate des neuen Geschäftsjahres | [§ 42a Abs. 2 Satz 1 GmbHG][gmbhg42a] |
| Offenlegung | spätestens ein Jahr nach dem Abschlussstichtag | [§ 325 Abs. 1a Satz 1 HGB][hgb325] |
| UG: gesetzliche Rücklage | ein Viertel des um einen Verlustvortrag geminderten Jahresüberschusses | [§ 5a Abs. 3 Satz 1 GmbHG][gmbhg5a] |
| Abschreibung | im Jahr der Anschaffung um ein Zwölftel je vollem Monat vor dem Anschaffungsmonat gekürzt | [§ 7 Abs. 1 Satz 4 EStG][estg7] |
| Körperschaftsteuer | Zeitraum der Steuerpflicht statt Kalenderjahr | [§ 7 Abs. 3 Satz 3 KStG][kstg7] |
| Gewerbesteuer | abgekürzter Erhebungszeitraum | [§ 14 Satz 3 GewStG][gewstg14] |
| Umsatzsteuer | der Teil des Jahres tritt an die Stelle des Kalenderjahres | [§ 16 Abs. 3 UStG][ustg16] |
| E-Bilanz | auch für die Eröffnungsbilanz | [§ 5b Abs. 1 Satz 4 EStG][estg5b] |

Stand: Oktober 2026. Den Abschluss begleitet buchfink-jahresabschluss, die
Erklärungen buchfink-steuererklaerungen.

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
([§ 108 Abs. 1 AO][ao108]); fällt ein Fristende der AO auf Wochenende oder
Feiertag, gilt der nächste Werktag ([§ 108 Abs. 3 AO][ao108]). Die Fälligkeiten
des Mandanten nennt `fristen`; mit beauftragtem Steuerberater liegen die
Steuererklärungen später.

[ao108]: https://www.gesetze-im-internet.de/ao_1977/__108.html
[ao137]: https://www.gesetze-im-internet.de/ao_1977/__137.html
[ao138]: https://www.gesetze-im-internet.de/ao_1977/__138.html
[ao149]: https://www.gesetze-im-internet.de/ao_1977/__149.html
[estdv8b]: https://www.gesetze-im-internet.de/estdv_1955/__8b.html
[estg4a]: https://www.gesetze-im-internet.de/estg/__4a.html
[estg5b]: https://www.gesetze-im-internet.de/estg/__5b.html
[estg7]: https://www.gesetze-im-internet.de/estg/__7.html
[gewstg14]: https://www.gesetze-im-internet.de/gewstg/__14.html
[gmbhg5a]: https://www.gesetze-im-internet.de/gmbhg/__5a.html
[gmbhg42a]: https://www.gesetze-im-internet.de/gmbhg/__42a.html
[hgb240]: https://www.gesetze-im-internet.de/hgb/__240.html
[hgb242]: https://www.gesetze-im-internet.de/hgb/__242.html
[hgb264]: https://www.gesetze-im-internet.de/hgb/__264.html
[hgb267]: https://www.gesetze-im-internet.de/hgb/__267.html
[hgb325]: https://www.gesetze-im-internet.de/hgb/__325.html
[kstg7]: https://www.gesetze-im-internet.de/kstg_1977/__7.html
[ustg16]: https://www.gesetze-im-internet.de/ustg_1980/__16.html
