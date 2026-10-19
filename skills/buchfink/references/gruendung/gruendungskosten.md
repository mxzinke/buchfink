# Gründungskosten und Vorgründungskosten

Rechtsstand: 17. Oktober 2026. Wortlaut von HGB, AktG, GmbHG (Anlage 1) und UStG
am 17.10.2026 gelesen; KStH über einen Kanzleispiegel; das BMF-Schreiben vom
25.06.1991 nur über Sekundärquellen ([Quellenverzeichnis][quellen]).

## Inhalt

1. Bilanz: kein Aktivposten
2. Wer die Gründungskosten trägt
3. Buchen
4. Kosten vor der Beurkundung

## 1. Bilanz: kein Aktivposten

Aufwendungen für die Gründung und für die Beschaffung des Eigenkapitals dürfen
nicht aktiviert werden ([§ 248 Abs. 1 Nr. 1, 2 HGB][hgb248]). Sie sind Aufwand
des ersten Geschäftsjahres und mindern das Reinvermögen sofort (Unterbilanz:
[kapitalaufbringung.md, Abschnitt 6](kapitalaufbringung.md#6-unterbilanz-am-eintragungstag)).

## 2. Wer die Gründungskosten trägt

- Für die AG verlangt das Gesetz, den Gründungsaufwand zu Lasten der
  Gesellschaft in der Satzung gesondert festzusetzen; ohne Festsetzung sind die
  Verträge der Gesellschaft gegenüber unwirksam
  ([§ 26 Abs. 2, 3 AktG][aktg26]). Das GmbHG hat keine eigene Vorschrift. Für
  die GmbH gilt § 26 Abs. 2 AktG nach der Rechtsprechung entsprechend (BGH vom
  20.02.1989, II ZB 10/88, GmbHR 1989, 250; übernommen von
  [BFH vom 11.10.1989, I R 12/87, BStBl II 1990, 89][bfh1287]). Die GmbH trägt
  Gründungsaufwand also nur, soweit der Gesellschaftsvertrag ihn festsetzt.
- Steuerlich: Zahlt die GmbH Gründungskosten, die nach dem Vertrag die
  Gesellschafter tragen, sind sie keine Betriebsausgabe; der BFH nahm eine
  Ausschüttung an ([BFH I R 12/87][bfh1287]). Die Finanzverwaltung verweist auf
  das BMF-Schreiben vom 25.06.1991, IV B 7 - S 2741 - 4/91, BStBl I S. 661
  ([H 8.5 KStH „Gründungskosten“][ksth85]); nach Sekundärquellen sieht es darin
  eine verdeckte Gewinnausschüttung. Den Wortlaut des Schreibens haben wir
  nicht gelesen.
- Das Musterprotokoll legt fest: Die Gesellschaft trägt die Gründungskosten bis
  300 €, höchstens bis zum Betrag ihres Stammkapitals; darüber hinaus trägt sie
  der Gesellschafter ([Anlage 1 GmbHG][gmbhganl1], Nr. 5). Stand: Oktober 2026.
- Buchfink speichert den Betrag aus dem Vertrag als „Gründungsaufwand laut
  Satzung“; der Mensch erfasst ihn im Einrichtungsassistenten.

Vorgehen:

1. Lies die Klausel im abgelegten Gesellschaftsvertrag oder frag den Menschen
   danach. Fertig, wenn Betrag oder „keine Klausel“ feststeht.
2. Summiere die Gründungsrechnungen (`buchungen_suchen` auf 6825 und 6420 im
   Gründungszeitraum, dazu offene Belege).
3. Liegt die Summe über dem Betrag laut Vertrag oder fehlt die Klausel:
   Wertungsfrage. Optionen: (a) der Gesellschafter erstattet den Überschuss,
   Forderung auf 1307
   ([gesellschafter/verrechnungskonto.md](../gesellschafter/verrechnungskonto.md));
   (b) die GmbH trägt ihn, mit dem Risiko einer verdeckten
   Gewinnausschüttung ([gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md));
   (c) Rücksprache mit Notar oder Steuerberater. Der Mensch entscheidet.

## 3. Buchen

| Kosten | Buchungsgruppe | Konto | Steuer |
|---|---|---|---|
| Notar (Beurkundung, Anmeldung, Beglaubigung) | `beratung` | 6825 Rechts- und Beratungskosten | Regelsatz laut Rechnung |
| Steuerberatung zur Gründung | `beratung` | 6825 | Regelsatz laut Rechnung |
| Registergericht, Behördengebühren | `beitraege` | 6420 Beiträge | meist nicht steuerbar |

- Den Beleg prüfst und buchst du nach [belege/ablauf.md](../belege/ablauf.md).
  Gründungsbelege sind meist keine Routine, weil der Kontakt neu ist und noch
  keine bestätigte Buchungsgruppe hat; meldet Buchfink `freigabe_noetig`,
  buche nach Rückfrage mit `beleg_buchen_mit_rueckfrage`.
- Vorsteuer setzt eine Rechnung an die GmbH voraus
  ([belege/pflichtangaben.md, Abschnitt 2](../belege/pflichtangaben.md#2-die-zehn-pflichtangaben)).
  Ist ein Gesellschafter persönlich Rechnungsempfänger, frag nach. Wertungsfrage: Ob der GmbH die Vorsteuer aus
  Gründungsrechnungen zusteht, beurteilt im Zweifel ein Steuerberater.
- Zahlt ein Gesellschafter eine Gründungsrechnung privat, buche sie als
  Auslage (`bezahlung: auslage`,
  [belege/fremdwaehrung-auslagen.md](../belege/fremdwaehrung-auslagen.md)).

## 4. Kosten vor der Beurkundung

- Die Vorgründungsgesellschaft besteht vom Entschluss bis zur Beurkundung. Sie
  ist weder mit der Vorgesellschaft noch mit der späteren GmbH identisch und in
  der Regel nicht körperschaftsteuerpflichtig
  ([H 1.1 KStH „Vorgründungsgesellschaft“][ksth11]).
- Ihre Verträge und Schulden gehen nicht von selbst auf die GmbH über. Rechnungen
  aus dieser Zeit (Beratung, Domain, Logo, Miete) buchst du deshalb nicht ohne
  Klärung in die GmbH.
- Wertungsfrage: Übernimmt die GmbH Kosten oder Verträge aus der
  Vorgründungszeit, braucht es dafür eine Vereinbarung; steuerlich (Vorsteuer,
  Betriebsausgabe, mögliche vGA) prüft das ein Steuerberater. Lege dem
  Menschen Belege, Beträge und Datum vor und notiere seine Entscheidung im
  Eigenbeleg oder in der Buchungsbeschreibung.

[quellen]: https://github.com/mxzinke/buchfink/blob/main/skills/QUELLEN.md
[aktg26]: https://www.gesetze-im-internet.de/aktg/__26.html
[bfh1287]: https://www.haufe.de/id/entscheidung/bfh-urteil-vom-11101989-i-r-1287-HI62976.html
[gmbhganl1]: https://www.gesetze-im-internet.de/normengrafiken/bgbl1_2021/j3364_0010.pdf
[hgb248]: https://www.gesetze-im-internet.de/hgb/__248.html
[ksth11]: https://www.steuerschroeder.de/steuergesetze/ksth/1.1
[ksth85]: https://www.steuerschroeder.de/steuergesetze/ksth/8.5
