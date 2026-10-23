# Fehler in einem übermittelten Zeitraum

Rechtsstand: 17. Oktober 2026.

## Inhalt

1. Berichtigung oder laufender Zeitraum
2. Pflicht und Recht zur Berichtigung
3. Ablauf in Buchfink
4. Nach der Jahreserklärung

## 1. Berichtigung oder laufender Zeitraum

| Vorgang | Zeitraum | Fundstelle |
|---|---|---|
| Entgelt ändert sich später (Skonto, Rabatt, Gutschrift, Ausfall, Rückgängigmachung) | Zeitraum der Änderung; der alte bleibt | [§ 17 Abs. 1 Satz 8 UStG][ustg17], [§ 18 Abs. 1 Satz 3 UStG][ustg18] |
| Fehler der ursprünglichen Erfassung: falscher Satz oder Steuerfall, vergessene Rechnung, falscher Monat, Vorsteuer ohne ordnungsgemäße Rechnung | ursprünglicher Zeitraum wird berichtigt | [§ 153 AO][ao153], [§ 164 Abs. 2 AO][ao164] |
| Falsche Kennziffer 21 | Voranmeldung unverzüglich, ZM binnen eines Monats berichtigen | [zm.md](zm.md#5-berichtigung) |

Wertungsfrage: Fehler oder spätere Änderung (etwa bei berichtigten
Eingangsrechnungen)? Leg beide Lesarten mit Folge vor; im Zweifel
Steuerberater.

## 2. Pflicht und Recht zur Berichtigung

- Erkennt die GmbH vor Ablauf der Festsetzungsfrist eine unrichtige Erklärung,
  die zu einer Steuerverkürzung führen kann, muss sie unverzüglich anzeigen und
  richtigstellen ([§ 153 Abs. 1 AO][ao153]).
- War die Steuer zu hoch, darf sie die Änderung beantragen
  ([§ 164 Abs. 2 AO][ao164]); die Herabsetzung wirkt erst mit Zustimmung des
  Finanzamts ([ustva.md](ustva.md#1-pflicht-und-form)).
- Absicht oder großer Fehler:
  [grundlagen/grenzen.md](../grundlagen/grenzen.md#2-warnzeichen-die-du-sofort-meldest).

## 3. Ablauf in Buchfink

1. **Fehler belegen.** Buchung finden (`buchungen_suchen`), Zeitraum,
   Kennziffer und Betrag nennen.
2. **Korrigieren.** Storno und Neubuchung nach
   [grundlagen/arbeitsweise.md](../grundlagen/arbeitsweise.md#11-eine-buchung-korrigieren)
   (Rückfrage). Die Steuer des Stornos zählt als Nachtrag im ursprünglichen
   Zeitraum; der Storno einer Ausgangsrechnung im Zeitraum des Stornos
   ([§ 17 Abs. 1 Satz 8 UStG][ustg17]), ohne Berichtigung.
3. **Auswirkung zeigen.** `erklaerung` (`art: ustva`, alter `zeitraum`): alt,
   neu, Differenz.
4. **Entscheidung.** Der Mensch entscheidet; bei Steuerverkürzung ist es keine
   Wahl (§ 153 AO).
5. **Berichtigte Anmeldung.** Der Mensch legt sie in der App an; Buchfink
   rechnet den Zeitraum neu und setzt Kennziffer 10. Weiter wie
   [ablauf.md](ablauf.md#2-ablauf), Schritte 8 bis 10.
6. **ZM**, wenn Kennziffer 21 betroffen ist ([zm.md](zm.md#5-berichtigung)).
7. **Zahlung** nach
   [bank/steuerzahlungen.md](../bank/steuerzahlungen.md#2-konten-je-steuerart);
   den Termin legt der Mensch fest.

Fertig, wenn der Mensch entschieden hat und, falls berichtigt wird, das
Transferticket der berichtigten Anmeldung in Buchfink steht.

## 4. Nach der Jahreserklärung

Ist die Jahreserklärung übermittelt, betrifft der Fehler auch sie
([§ 18 Abs. 3 UStG][ustg18], [§ 153 AO][ao153];
[steuererklaerungen/ust-jahr.md](../steuererklaerungen/ust-jahr.md#5-fehler-nach-der-abgabe)).
Ob die Voranmeldung daneben berichtigt wird, klärt der Mensch mit dem
Finanzamt.

[ustg17]: https://www.gesetze-im-internet.de/ustg_1980/__17.html
[ustg18]: https://www.gesetze-im-internet.de/ustg_1980/__18.html
[ao153]: https://www.gesetze-im-internet.de/ao_1977/__153.html
[ao164]: https://www.gesetze-im-internet.de/ao_1977/__164.html
