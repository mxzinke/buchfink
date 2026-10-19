# Zusammenfassende Meldung (ZM)

Rechtsstand: 17. Oktober 2026. Wortlaut von § 18a und § 18b UStG am
17.10.2026 gelesen; Übermittlungswege nach den Seiten des Bundeszentralamts
für Steuern (BZSt), abgerufen am 17.10.2026.

Frist, Weg, Nachweis und Folgen einer Versäumnis stehen in
[grundlagen/pflichten.md#zm](../grundlagen/pflichten.md#zm).

## Inhalt

1. Wann die GmbH meldet
2. Meldezeitraum und Frist
3. Inhalt und Abgleich
4. Übermittlung
5. Berichtigung

## 1. Wann die GmbH meldet

- Meldepflichtig sind steuerpflichtige sonstige Leistungen im übrigen
  Gemeinschaftsgebiet nach § 3a Abs. 2 UStG, für die der Empfänger in seinem
  Mitgliedstaat die Steuer schuldet ([§ 18a Abs. 2 UStG][ustg18a]). Das sind die
  Rechnungen an Unternehmer im EU-Ausland mit Hinweis auf die
  Steuerschuldnerschaft des Empfängers
  ([rechnungen/ausland.md](../rechnungen/ausland.md)).
- Innergemeinschaftliche Warenlieferungen sind ebenfalls meldepflichtig
  (§ 18a Abs. 1). Buchfink bildet den Warenverkehr auf der Ausgangsseite nicht
  ab. Verkauft die GmbH doch einen Gegenstand an einen EU-Unternehmer, sag das
  dem Menschen und empfiehl einen Steuerberater.
- In der Voranmeldung erscheinen dieselben Leistungen in Kennziffer 21
  ([ustva.md, Abschnitt 1](ustva.md#1-pflicht-und-form)).

## 2. Meldezeitraum und Frist

- Meldezeitraum ist das Kalendervierteljahr
  ([§ 18a Abs. 2 Satz 1 UStG][ustg18a]). Zeigt die GmbH dem BZSt die
  monatliche Meldung an, ist es der Kalendermonat (§ 18a Abs. 3). Frist und
  Dauerfristverlängerung:
  [grundlagen/pflichten.md#zm](../grundlagen/pflichten.md#zm).
- Die Angaben gehören in den Meldezeitraum, in dem die Leistung ausgeführt
  wurde (§ 18a Abs. 8 Satz 2).
- Für die ZM gelten die Vorschriften der AO über Steuererklärungen, ausgenommen
  § 152 AO (§ 18a Abs. 11); Bußgeld statt Verspätungszuschlag
  ([grundlagen/pflichten.md#zm](../grundlagen/pflichten.md#zm)).

Stand: Oktober 2026.

## 3. Inhalt und Abgleich

Je Empfänger: seine USt-IdNr., die Summe der Bemessungsgrundlagen und der
Hinweis auf eine sonstige Leistung (§ 18a Abs. 7 Satz 1 Nr. 3).

Prüfe mit `erklaerung` (`art: zm`, `zeitraum`):

1. Jede Zeile hat eine USt-IdNr., die Buchfink beim BZSt bestätigt hat. Fehlt
   die Bestätigung, meldet Buchfink einen Befund; kläre ihn nach
   [rechnungen/ausland.md](../rechnungen/ausland.md).
2. Summe der Zeilen gleich Summe der Kennziffer 21 aus den Voranmeldungen
   desselben Zeitraums. Buchfink stellt beide gegenüber und speichert
   Differenzen. Eine Differenz bedeutet meist eine Buchung mit falschem
   Steuerfall.
3. Bemessungsgrundlagen in vollen Euro (Buchfink schneidet die Cent ab).

Das BZSt gleicht die ZM mit den Voranmeldungen ab und schreibt bei
Abweichungen an (BZSt, Berichtigung einer ZM).

## 4. Übermittlung

- Die ZM geht elektronisch an das BZSt, über Mein ELSTER oder das
  BZSt-Online-Portal (BOP). Weg, Ablauf mit dem Menschen und Nachweis stehen
  in [grundlagen/pflichten.md#zm](../grundlagen/pflichten.md#zm).
- Beide Formulare nehmen eine CSV-Datei mit den Meldezeilen an; ein Formular
  fasst bis zu 1.500 Meldezeilen (BZSt, Elektronische Abgabe; Stand Oktober
  2026).

## 5. Berichtigung

- Erkennt die GmbH, dass eine abgegebene ZM unrichtig oder unvollständig ist,
  berichtigt sie diese innerhalb eines Monats ([§ 18a Abs. 10 UStG][ustg18a]).
- Je betroffenem Meldezeitraum eine eigene berichtigte ZM. Korrekt gemeldete
  Zeilen werden nicht wiederholt. Eine Zeile mit falscher USt-IdNr. wird mit
  Bemessungsgrundlage 0 storniert und mit der richtigen Nummer neu gemeldet
  (BZSt, Berichtigung einer ZM).
- Ist Kennziffer 21 einer Voranmeldung falsch, ist auch die Voranmeldung
  unverzüglich zu berichtigen ([§ 18b Satz 5 UStG][ustg18b]).
- In Buchfink: Die Berichtigung rechnet den Zeitraum vollständig neu und
  verknüpft sie mit der übermittelten Meldung. `erklaerung` zeigt deshalb auch
  unveränderte Zeilen. Leg dem Menschen nur die Zeilen vor, die sich ändern
  oder storniert werden; nur diese übernimmt er in die berichtigte ZM.

[ustg18a]: https://www.gesetze-im-internet.de/ustg_1980/__18a.html
[ustg18b]: https://www.gesetze-im-internet.de/ustg_1980/__18b.html

BZSt-Seiten:

- Elektronische Abgabe: https://www.bzst.de/DE/Unternehmen/Umsatzsteuer/ZusammenfassendeMeldung/ElektronischeAbgabe/elektronischeabgabe_zm_node.html
- Berichtigung einer ZM: https://www.bzst.de/DE/Unternehmen/Umsatzsteuer/ZusammenfassendeMeldung/BerichtigungZM/berichtigung_node.html
- Fristen: https://www.bzst.de/DE/Unternehmen/Umsatzsteuer/ZusammenfassendeMeldung/Fristen/fristen_node.html
