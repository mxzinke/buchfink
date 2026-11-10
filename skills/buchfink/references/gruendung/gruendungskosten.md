# Gründungskosten und Vorgründungskosten

Rechtsstand: 17. Oktober 2026. Prüfstand der Quellen:
[Quellenverzeichnis][quellen].

## Inhalt

1. Bilanz: kein Aktivposten
2. Wer die Gründungskosten trägt
3. Buchen
4. Kosten vor der Beurkundung

## 1. Bilanz: kein Aktivposten

Aufwendungen für die Gründung und die Beschaffung des Eigenkapitals sind nicht
aktivierbar ([§ 248 Abs. 1 Nr. 1, 2 HGB][hgb248]). Sie sind Aufwand des
ersten Geschäftsjahres und mindern das Reinvermögen sofort (Unterbilanz:
[kapitalaufbringung.md, Abschnitt 6](kapitalaufbringung.md#6-unterbilanz-am-eintragungstag)).

## 2. Wer die Gründungskosten trägt

- Die GmbH trägt Gründungsaufwand nur, soweit der Gesellschaftsvertrag ihn
  festsetzt ([§ 26 Abs. 2 AktG][aktg26] entsprechend: BGH vom 20.02.1989,
  II ZB 10/88; [BFH vom 11.10.1989, I R 12/87, BStBl II 1990, 89][bfh1287]).
- Zahlt die GmbH Gründungskosten, die laut Vertrag die Gesellschafter tragen,
  ist das keine Betriebsausgabe, sondern verdeckte Gewinnausschüttung
  ([BFH I R 12/87][bfh1287]; [H 8.5 KStH „Gründungskosten“][ksth85] mit
  BMF-Schreiben vom 25.06.1991, IV B 7 - S 2741 - 4/91, BStBl I S. 661).
- Musterprotokoll: Die Gesellschaft trägt die Gründungskosten bis 300 €,
  höchstens bis zu ihrem Stammkapital; den Rest trägt der Gesellschafter
  ([Anlage 1 GmbHG][gmbhganl1], Nr. 5).

Vorgehen:

1. Lies die Klausel im abgelegten Gesellschaftsvertrag oder frag den Menschen
   danach. Fertig, wenn Betrag oder „keine Klausel“ feststeht; den Betrag
   schlägst du als `gruendungsaufwand_bis` vor
   ([ablauf.md, Abschnitt 2](ablauf.md#2-ablauf), Schritt 3).
2. Summiere die Gründungsrechnungen (`buchungen_suchen` auf 6825 und 6430 im
   Gründungszeitraum, dazu offene Belege).
3. Liegt die Summe über der Klausel oder fehlt die Klausel: Wertungsfrage mit
   den Optionen (a) der Gesellschafter erstattet den Überschuss, Forderung auf
   1307 ([gesellschafter/verrechnungskonto.md](../gesellschafter/verrechnungskonto.md));
   (b) die GmbH trägt ihn mit dem Risiko einer verdeckten Gewinnausschüttung
   ([gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md));
   (c) Rücksprache mit Notar oder Steuerberater. Der Mensch entscheidet.

## 3. Buchen

Notar und Steuerberatung zur Gründung gehören in die Gruppe `beratung`
(6825 Rechts- und Beratungskosten), Gebühren von Registergericht und Behörden
in `gebuehren` (6430 Sonstige Abgaben, nicht steuerbar), nicht unter die
Beiträge. Den Beleg buchst du
nach [belege/ablauf.md](../belege/ablauf.md).

- Vorsteuer setzt eine Rechnung an die GmbH voraus
  ([belege/pflichtangaben.md, Abschnitt 2](../belege/pflichtangaben.md#2-die-zehn-pflichtangaben));
  ist ein Gesellschafter Rechnungsempfänger, frag nach. Ob der GmbH die
  Vorsteuer aus Gründungsrechnungen zusteht, ist im Zweifel eine Wertungsfrage
  für den Steuerberater.
- Zahlt ein Gesellschafter eine Gründungsrechnung privat, buche sie als
  Auslage
  ([belege/fremdwaehrung-auslagen.md](../belege/fremdwaehrung-auslagen.md)).

## 4. Kosten vor der Beurkundung

- Die Vorgründungsgesellschaft (Entschluss bis Beurkundung) ist weder mit der
  Vorgesellschaft noch mit der GmbH identisch und in der Regel nicht
  körperschaftsteuerpflichtig
  ([H 1.1 KStH „Vorgründungsgesellschaft“][ksth11]). Ihre Verträge und
  Schulden gehen nicht von selbst auf die GmbH über; Rechnungen aus dieser
  Zeit (Beratung, Domain, Logo, Miete) buchst du erst nach Klärung.
- Wertungsfrage: Übernimmt die GmbH Kosten oder Verträge aus der
  Vorgründungszeit, braucht es eine Vereinbarung; Vorsteuer, Betriebsausgabe
  und mögliche vGA prüft ein Steuerberater. Lege Belege, Beträge und Datum
  vor und halte die Entscheidung im Eigenbeleg oder in der
  Buchungsbeschreibung fest.

[quellen]: https://github.com/mxzinke/buchfink/blob/master/skills/QUELLEN.md
[aktg26]: https://www.gesetze-im-internet.de/aktg/__26.html
[bfh1287]: https://www.haufe.de/id/entscheidung/bfh-urteil-vom-11101989-i-r-1287-HI62976.html
[gmbhganl1]: https://www.gesetze-im-internet.de/normengrafiken/bgbl1_2021/j3364_0010.pdf
[hgb248]: https://www.gesetze-im-internet.de/hgb/__248.html
[ksth11]: https://www.steuerschroeder.de/steuergesetze/ksth/1.1
[ksth85]: https://www.steuerschroeder.de/steuergesetze/ksth/8.5
