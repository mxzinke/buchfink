# Ausweis gegenüber Gesellschaftern im Abschluss

Rechtsstand: 17. Oktober 2026.

Den Abschluss regelt [jahresabschluss/ablauf.md](../jahresabschluss/ablauf.md);
hier steht, was Gesellschafter betrifft.

## Inhalt

1. Regel
2. Konten und Bilanzposten
3. Prüfung vor dem Abschluss

## 1. Regel

- Ausleihungen, Forderungen und Verbindlichkeiten gegenüber Gesellschaftern
  sind in der Regel gesondert auszuweisen oder im Anhang anzugeben; stehen sie
  unter anderen Posten, ist das zu vermerken ([§ 42 Abs. 3 GmbHG][gmbhg42]).
- Keine Verrechnung von Forderungen und Verbindlichkeiten gegenüber einem
  Gesellschafter; Restlaufzeiten je Posten
  ([jahresabschluss/forderungen.md, Abschnitt 5](../jahresabschluss/forderungen.md#5-ausweis-und-restlaufzeiten)).

## 2. Konten und Bilanzposten

Gesellschafterkonten im SKR04: 0960 Ausleihungen an Gesellschafter, 1250
Lieferforderungen gegen Gesellschafter, 1307 bis 1309 Forderungen gegen
GmbH-Gesellschafter, 3340 Lieferverbindlichkeiten gegenüber Gesellschaftern,
3519 offene Ausschüttungen, 3640 bis 3643 Verbindlichkeiten gegenüber
GmbH-Gesellschaftern, 2928 Kapitalrücklage aus Zuzahlungen. Den Bilanzposten
jedes Kontos nennt `konten`.

## 3. Prüfung vor dem Abschluss

1. `auswertung` (`art: susa`, `geschaeftsjahr`): Salden der Konten aus
   Abschnitt 2.
2. Je Gesellschafter: Passt der Saldo zu Vertrag, Kontoauszug oder
   Saldenbestätigung des Gesellschafters? Frag den Menschen.
3. Steht ein Saldo auf 1307 statt 1308 oder 1309 bzw. auf 3640 statt 3641
   bis 3643, schlag die Umbuchung auf das Restlaufzeitkonto vor
   (`buchung_erfassen`, Rückfrage).
4. Rechnungen mit Gesellschaftern als Kunden oder Lieferanten laufen über ihr
   Personenkonto; ob sie unter „Forderungen gegen Gesellschafter“ gehören,
   klärst du bei der Gliederung
   ([jahresabschluss/groessenklassen-anhang.md](../jahresabschluss/groessenklassen-anhang.md)).

[gmbhg42]: https://www.gesetze-im-internet.de/gmbhg/__42.html
