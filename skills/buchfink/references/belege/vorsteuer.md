# Vorsteuer

Rechtsstand: 17. Oktober 2026. Wortlaut von UStG, UStDV, EStG und KStG am
17.10.2026 gelesen.

## Inhalt

1. Voraussetzungen
2. Zeitpunkt
3. Ausschlüsse
4. Aufteilung
5. Nicht abziehbare Vorsteuer in der Buchung
6. Berichtigung nach § 15a UStG
7. Minderung des Entgelts

## 1. Voraussetzungen

Rechnungsvorsteuer ([§ 15 Abs. 1 Satz 1 Nr. 1 UStG][ustg15]): Leistung eines
anderen Unternehmers für das Unternehmen der GmbH, Rechnung nach §§ 14, 14a
UStG liegt vor (Satz 2), abziehbar höchstens die gesetzlich geschuldete
deutsche Steuer. Einfuhrumsatzsteuer
(Nr. 2), Erwerbsteuer (Nr. 3) und Steuer nach § 13b (Nr. 4) sind ebenfalls
Vorsteuer; Nr. 3 und 4 setzen keine Rechnung nach § 14 voraus.

## 2. Zeitpunkt

- Rechnungsvorsteuer: Leistung ausgeführt und Rechnung liegt vor. Buchfink
  nimmt das späteste von Leistungsende, Belegdatum und `eingang_am`; setz
  `eingang_am` auf den echten Eingangstag.
- Anzahlung: Rechnung liegt vor und ist gezahlt
  ([§ 15 Abs. 1 Satz 1 Nr. 1 Satz 3 UStG][ustg15]). Buche die
  Anzahlungsrechnung erst, wenn die Zahlung im Kontoauszug steht.

## 3. Ausschlüsse

| Ausschluss | Fundstelle | Buchfink |
|---|---|---|
| Geschenke über der Freigrenze, Gästehäuser, Jagd, Fischerei, Jachten, unangemessene Repräsentation, Lebensführung | [§ 15 Abs. 1a Satz 1 UStG][ustg15] mit [§ 4 Abs. 5 Satz 1 Nr. 1 bis 4, 7 EStG][estg4], [§ 12 Nr. 1 EStG][estg12]; [bewirtung-geschenke.md](bewirtung-geschenke.md) | Gruppen `geschenke`, `repraesentation` |
| Bewirtung | [bewirtung-geschenke.md](bewirtung-geschenke.md#1-bewirtung) | Gruppe `bewirtung` |
| Grundstück teils unternehmensfremd genutzt | [§ 15 Abs. 1b UStG][ustg15] | vorlegen |
| Leistung dient steuerfreien Umsätzen (steuerfreie Vermietung, Finanzumsätze einer Holding) | [§ 15 Abs. 2 Satz 1 Nr. 1 UStG][ustg15]; Ausnahmen Abs. 3 | Vorsteueranteil |
| Eigenbeleg | Rechnung fehlt | keine Vorsteuer |

Ob eine Ausgabe zur privaten Lebensführung eines Gesellschafters gehört, ist
eine Wertungsfrage; Folgen in
[gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md).

## 4. Aufteilung

- Dient eine Leistung teils Umsätzen mit und teils ohne Vorsteuerabzug, ist der
  wirtschaftlich den Umsätzen ohne Abzug zuzuordnende Teil nicht abziehbar
  ([§ 15 Abs. 4 UStG][ustg15]): sachgerechte Schätzung; Umsatzschlüssel nur
  ohne präzisere Zuordnung; bei Grundstücken grundsätzlich die Nutzfläche.
- In Buchfink je Position `vorsteueranteil_promille` und
  `vorsteueranteil_grund`. Den Maßstab wählt der Mensch (Wertungsfrage); schreib
  ihn prüferfest („Nutzfläche 120 von 400 m² steuerfrei vermietet“). Ist gar
  keine Vorsteuer abziehbar, leg den Fall vor dem Buchen vor.

## 5. Nicht abziehbare Vorsteuer in der Buchung

- Nicht abziehbare Vorsteuer bleibt Teil der Anschaffungskosten oder des
  Aufwands, abziehbare gehört nicht dazu ([§ 9b Abs. 1 EStG][estg9b]); Buchfink
  bucht sie so.
- Vorsteuer auf Aufwendungen mit Abzugsverbot nach § 4 Abs. 5 Satz 1 Nr. 1
  bis 4, 7 oder Abs. 7 EStG ist bei der GmbH nicht abziehbar
  ([§ 10 Nr. 2 KStG][kstg10]). Buchfink bucht sie auf das Konto des nicht
  abziehbaren Aufwands; Hinzurechnung in
  [steuererklaerungen/kst.md](../steuererklaerungen/kst.md).

## 6. Berichtigung nach § 15a UStG

- Ändern sich bei einem Wirtschaftsgut innerhalb von fünf Jahren ab
  erstmaliger Verwendung (Grundstücke zehn) die Verhältnisse des Abzugs, wird
  je Jahr ein Fünftel bzw. Zehntel berichtigt ([§ 15a Abs. 1, 5 UStG][ustg15a]);
  nicht bei höchstens 1.000 € Vorsteuer auf das Wirtschaftsgut
  ([§ 44 Abs. 1 UStDV][ustdv44]; weitere Vereinfachungen Abs. 2, 3; Stand:
  Oktober 2026).
- Buchfink führt das Verzeichnis dieser Wirtschaftsgüter
  ([§ 22 Abs. 4 UStG][ustg22]), bucht die Berichtigung aber noch nicht. Ändert
  sich die Nutzung (Verkauf, steuerfreie Vermietung, private Nutzung), sag es
  dem Menschen.

## 7. Minderung des Entgelts

- Mindert sich das Entgelt (Skonto, Rabatt, Gutschrift, Rückgängigmachung),
  berichtigt die GmbH die Vorsteuer
  ([§ 17 Abs. 1 Satz 2, Abs. 2 Nr. 3 UStG][ustg17]) im Zeitraum der Änderung
  ([monatsabschluss/berichtigung.md, Abschnitt 1](../monatsabschluss/berichtigung.md#1-berichtigung-oder-laufender-zeitraum));
  Skonto mit der Zahlung
  ([bank/skonto-differenzen.md](../bank/skonto-differenzen.md#1-skonto)). Eine
  Rechnungsberichtigung ist dafür nicht nötig
  ([BMF vom 15.10.2025, Rn. 51a][bmf2025]).
- Ebenso, wenn die GmbH eine Verbindlichkeit wegen Uneinbringlichkeit nicht
  zahlt ([§ 17 Abs. 2 Nr. 1 UStG][ustg17]); sag es dem Menschen, wenn eine
  Eingangsrechnung dauerhaft offen bleibt.

[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
[ustg15a]: https://www.gesetze-im-internet.de/ustg_1980/__15a.html
[ustg17]: https://www.gesetze-im-internet.de/ustg_1980/__17.html
[ustdv44]: https://www.gesetze-im-internet.de/ustdv_1980/__44.html
[ustg22]: https://www.gesetze-im-internet.de/ustg_1980/__22.html
[estg4]: https://www.gesetze-im-internet.de/estg/__4.html
[estg9b]: https://www.gesetze-im-internet.de/estg/__9b.html
[estg12]: https://www.gesetze-im-internet.de/estg/__12.html
[kstg10]: https://www.gesetze-im-internet.de/kstg_1977/__10.html
[bmf2025]: https://www.steuerschroeder.de/blog/wp-content/uploads/2025/10/2025-10-15-einfuehrung-obligatorische-e-rechnung.pdf
