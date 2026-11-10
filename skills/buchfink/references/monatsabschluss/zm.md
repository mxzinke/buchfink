# Zusammenfassende Meldung (ZM)

Rechtsstand: 17. Oktober 2026.

Frist, Weg, Nachweis und Folgen:
[grundlagen/pflichten.md#zm](../grundlagen/pflichten.md#zm).

## Inhalt

1. Wann die GmbH meldet
2. Meldezeitraum und Frist
3. Inhalt und Abgleich
4. Übermittlung
5. Berichtigung

## 1. Wann die GmbH meldet

- Meldepflichtig sind sonstige Leistungen nach § 3a Abs. 2 UStG im übrigen
  Gemeinschaftsgebiet, für die der Empfänger die Steuer schuldet
  ([§ 18a Abs. 2 UStG][ustg18a]): die Rechnungen mit `reverse_charge_supply`
  ([rechnungen/ausland.md](../rechnungen/ausland.md)), in der Voranmeldung
  Kennziffer 21.
- Innergemeinschaftliche Lieferungen (§ 18a Abs. 1) bildet Buchfink nicht ab;
  verkauft die GmbH einen Gegenstand an einen EU-Unternehmer, empfiehl einen
  Steuerberater.

## 2. Meldezeitraum und Frist

- Kalendervierteljahr ([§ 18a Abs. 2 Satz 1 UStG][ustg18a]), nach Anzeige
  beim BZSt Kalendermonat (§ 18a Abs. 3).
- Gemeldet wird im Zeitraum der Ausführung (§ 18a Abs. 8 Satz 2).
- Kein Verspätungszuschlag (§ 18a Abs. 11 nimmt § 152 AO aus); Bußgeld:
  [grundlagen/pflichten.md#zm](../grundlagen/pflichten.md#zm).

## 3. Inhalt und Abgleich

Je Empfänger USt-IdNr., Summe der Bemessungsgrundlagen, Hinweis auf sonstige
Leistung (§ 18a Abs. 7 Satz 1 Nr. 3). Prüfe mit `erklaerung` (`art: zm`,
`zeitraum`):

1. Jede Zeile hat eine bestätigte USt-IdNr.; Befunde klärst du nach
   [rechnungen/ausland.md](../rechnungen/ausland.md).
2. Summe der Zeilen gleich Kennziffer 21 der Voranmeldungen desselben
   Zeitraums; eine Differenz heißt meist falscher Steuerfall. Das BZSt gleicht
   ebenfalls ab ([BZSt][bzstber]).
3. Volle Euro; die Cent fallen weg.

## 4. Übermittlung

Elektronisch an das BZSt über Mein ELSTER oder das BZSt-Online-Portal; Weg
und Nachweis: [grundlagen/pflichten.md#zm](../grundlagen/pflichten.md#zm).

## 5. Berichtigung

- Binnen eines Monats nach Erkennen ([§ 18a Abs. 10 UStG][ustg18a]); falsche
  Kennziffer 21 auch in der Voranmeldung unverzüglich
  ([§ 18b Satz 5 UStG][ustg18b]).
- Je Meldezeitraum eine berichtigte ZM nur mit den geänderten Zeilen; eine
  falsche USt-IdNr. mit Bemessungsgrundlage 0 stornieren und richtig neu melden
  ([BZSt][bzstber]).
- Buchfink bildet die berichtigte ZM noch nicht ab. `erklaerung` zeigt den
  ganzen Zeitraum neu; leg dem Menschen nur geänderte und stornierte Zeilen
  vor. Er trägt sie ein und bewahrt das Transferticket selbst auf.

[ustg18a]: https://www.gesetze-im-internet.de/ustg_1980/__18a.html
[ustg18b]: https://www.gesetze-im-internet.de/ustg_1980/__18b.html
[bzstber]: https://www.bzst.de/DE/Unternehmen/Umsatzsteuer/ZusammenfassendeMeldung/BerichtigungZM/berichtigung_node.html
