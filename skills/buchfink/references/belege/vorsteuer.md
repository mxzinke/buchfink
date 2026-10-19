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

Für Rechnungsvorsteuer ([§ 15 Abs. 1 Satz 1 Nr. 1 UStG][ustg15]):

- Ein anderer Unternehmer hat eine Lieferung oder sonstige Leistung ausgeführt.
- Die Leistung ist für das Unternehmen der GmbH ausgeführt.
- Die GmbH besitzt eine Rechnung nach §§ 14, 14a UStG (Satz 2).
- Abziehbar ist die gesetzlich geschuldete Steuer, also höchstens die richtige
  deutsche Steuer für diese Leistung.

Ein Gegenstand, den das Unternehmen zu weniger als 10 % nutzt, gilt nicht als
für das Unternehmen geliefert ([§ 15 Abs. 1 Satz 2 UStG][ustg15]).

Weitere Vorsteuerarten: Einfuhrumsatzsteuer (Nr. 2), Erwerbsteuer (Nr. 3),
Steuer nach § 13b (Nr. 4); für Nr. 3 und Nr. 4 ist keine Rechnung nach § 14
Voraussetzung.

## 2. Zeitpunkt

- Rechnungsvorsteuer: wenn Leistung ausgeführt ist und die Rechnung vorliegt.
- Anzahlung: wenn die Rechnung vorliegt und gezahlt ist
  ([§ 15 Abs. 1 Satz 1 Nr. 1 Satz 3 UStG][ustg15]). Buche die
  Anzahlungsrechnung deshalb erst, wenn die Zahlung im Kontoauszug steht.
- Buchfink nimmt das späteste von drei Daten: Ende des Leistungszeitraums,
  Belegdatum, Eingang des Belegs. Setze
  `eingang_am` beim Ablegen auf den echten Eingangstag; berichtigen lässt er
  sich mit `beleg_buchen_mit_rueckfrage`. Ein Beleg, der erst im
  Folgemonat eingeht, gehört mit seiner Vorsteuer in den Folgemonat.

## 3. Ausschlüsse

| Ausschluss | Fundstelle | Buchfink |
|---|---|---|
| Geschenke über der Freigrenze, Gästehäuser, Jagd, Fischerei, Jachten, unangemessene Repräsentation, Kosten der Lebensführung | [§ 15 Abs. 1a Satz 1 UStG][ustg15] mit [§ 4 Abs. 5 Satz 1 Nr. 1 bis 4, 7 EStG][estg4], [§ 12 Nr. 1 EStG][estg12]; Einzelheiten in [bewirtung-geschenke.md](bewirtung-geschenke.md) | Gruppen `geschenke`, `repraesentation` |
| Bewirtung | [bewirtung-geschenke.md](bewirtung-geschenke.md#1-bewirtung) | Gruppe `bewirtung` |
| Grundstück teils unternehmensfremd genutzt | [§ 15 Abs. 1b UStG][ustg15] | vorlegen |
| Eingangsleistung dient steuerfreien Umsätzen (etwa steuerfreie Vermietung, Finanzumsätze einer Holding) | [§ 15 Abs. 2 Satz 1 Nr. 1 UStG][ustg15]; Ausnahmen in Abs. 3 | Vorsteueranteil |
| Eigenbeleg | Rechnung fehlt | keine Vorsteuer |

Ob eine Ausgabe zur privaten Lebensführung eines Gesellschafters gehört, ist
eine Wertungsfrage; die Folgen für die GmbH stehen in
[gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md).

## 4. Aufteilung

- Dient eine Eingangsleistung teils Umsätzen mit Vorsteuerabzug und teils
  Umsätzen ohne, ist der Teil nicht abziehbar, der den schädlichen Umsätzen
  wirtschaftlich zuzuordnen ist ([§ 15 Abs. 4 Satz 1 UStG][ustg15]).
- Der Unternehmer darf sachgerecht schätzen (Satz 2). Ein Umsatzschlüssel ist
  nur zulässig, wenn keine präzisere Zuordnung möglich ist (Satz 3). Bei
  Grundstücken gilt grundsätzlich das Verhältnis der Nutzflächen (Satz 4).
- In Buchfink: je Position `vorsteueranteil_promille` (abziehbarer Anteil,
  etwa 600 für 60 %; ohne Angabe und mit 0 voll abziehbar) und
  `vorsteueranteil_grund` mit dem Maßstab. Ist gar keine Vorsteuer abziehbar,
  leg den Fall dem Menschen vor, bevor du buchst. Den Maßstab wählt der Mensch
  (Wertungsfrage). Schreib ihn so, dass ein Prüfer ihn ohne Chat versteht
  („Nutzfläche 120 von 400 m² steuerfrei vermietet“).

## 5. Nicht abziehbare Vorsteuer in der Buchung

- Abziehbare Vorsteuer gehört nicht zu den Anschaffungskosten
  ([§ 9b Abs. 1 EStG][estg9b]). Nicht abziehbare Vorsteuer bleibt Teil der
  Anschaffungskosten oder des Aufwands. Buchfink bucht sie so.
- Vorsteuer auf Aufwendungen mit Abzugsverbot nach § 4 Abs. 5 Satz 1 Nr. 1
  bis 4 und 7 oder Abs. 7 EStG ist bei der GmbH nicht abziehbar
  ([§ 10 Nr. 2 KStG][kstg10]). Buchfink bucht sie auf das Konto des nicht
  abziehbaren Aufwands; die Hinzurechnung steht in
  [steuererklaerungen/kst.md](../steuererklaerungen/kst.md).

## 6. Berichtigung nach § 15a UStG

- Ändern sich bei einem Wirtschaftsgut, das mehrfach verwendet wird, innerhalb
  von fünf Jahren ab erstmaliger Verwendung die Verhältnisse des ursprünglichen
  Vorsteuerabzugs, wird für jedes Jahr der Änderung berichtigt; bei Grundstücken
  zehn Jahre ([§ 15a Abs. 1 UStG][ustg15a]). Je Jahr ein Fünftel bzw. Zehntel
  der Vorsteuer ([§ 15a Abs. 5 UStG][ustg15a]).
- Vereinfachungen ([§ 44 UStDV][ustdv44]), Stand: Oktober 2026:
  - keine Berichtigung, wenn die Vorsteuer auf das Wirtschaftsgut 1.000 € nicht
    übersteigt (Abs. 1);
  - keine Berichtigung für ein Jahr, wenn sich die Verhältnisse um weniger als
    zehn Prozentpunkte geändert haben, außer der Berichtigungsbetrag übersteigt
    1.000 € (Abs. 2);
  - bis 6.000 € Berichtigungsbetrag erst in der Jahreserklärung (Abs. 3).
- Über der Grenze sind die Berechnungsgrundlagen aufzuzeichnen
  ([§ 22 Abs. 4 UStG][ustg22]). Buchfink führt ein Verzeichnis der
  Wirtschaftsgüter mit Vorsteuer über der Grenze (Konten 1396, 1397, 1398,
  1399) und übernimmt die Vorsteuer aus der Buchung. Den Verwendungsanteil zu
  erfassen und die Berichtigung zu buchen bildet es noch nicht ab. Dein Teil:
  Erfährst du, dass sich die Nutzung ändert (Verkauf, steuerfreie Vermietung,
  private Nutzung), sag es dem Menschen.

## 7. Minderung des Entgelts

- Mindert sich das Entgelt (Skonto, Rabatt, Gutschrift, Rückgängigmachung),
  berichtigt die GmbH die Vorsteuer
  ([§ 17 Abs. 1 Satz 2, Abs. 2 Nr. 3 UStG][ustg17]), und zwar im Zeitraum der
  Änderung ([monatsabschluss/berichtigung.md, Abschnitt 1](../monatsabschluss/berichtigung.md#1-berichtigung-oder-laufender-zeitraum)). Skonto bucht
  Buchfink mit der Zahlung
  ([bank/skonto-differenzen.md](../bank/skonto-differenzen.md#1-skonto)).
- Zahlt die GmbH eine Verbindlichkeit nicht, weil sie uneinbringlich ist, gilt
  dasselbe ([§ 17 Abs. 2 Nr. 1 UStG][ustg17]). Sag es dem Menschen, wenn eine
  Eingangsrechnung dauerhaft offen bleibt.
- Eine Rechnungsberichtigung ist bei einer bloßen Minderung nach § 17 nicht
  nötig ([BMF vom 15.10.2025, Rn. 51a][bmf2025]).

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
