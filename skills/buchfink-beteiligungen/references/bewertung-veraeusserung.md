# Bewertung und Veräußerung von Finanzanlagen

Rechtsstand: 17. Oktober 2026. Wortlaut von HGB, EStG, KStG, GmbHG und UStG am
17.10.2026 auf gesetze-im-internet.de gelesen. Konten und Abläufe aus
`internal/accounting/asset_accounts.go` und `internal/service/asset_service.go`.

Planmäßige Abschreibung, Anlagenspiegel und den Abschreibungslauf regelt
buchfink-jahresabschluss. Hier geht es um Anteile, Wertpapiere und
Ausleihungen.

## Inhalt

1. Bewertung zum Stichtag nach HGB
2. Teilwert nach EStG
3. Folgen bei Anteilen an Kapitalgesellschaften
4. Darlehen an die Tochter
5. Wertänderung in Buchfink
6. Verkauf: Ablauf
7. Verkauf: Steuer
8. Teilverkauf, Tilgung, Fonds

## 1. Bewertung zum Stichtag nach HGB

- Anlagevermögen ist bei voraussichtlich dauernder Wertminderung auf den
  niedrigeren Stichtagswert abzuschreiben
  ([§ 253 Abs. 3 Satz 5 HGB][hgb253]). Bei Finanzanlagen darf auch bei nicht
  dauernder Minderung abgeschrieben werden (Satz 6).
- Wertpapiere des Umlaufvermögens sind auf den niedrigeren Börsen- oder
  Marktpreis am Stichtag abzuschreiben, ob dauernd oder nicht
  ([§ 253 Abs. 4 HGB][hgb253]).
- Entfällt der Grund, ist zuzuschreiben, höchstens auf die Anschaffungskosten
  ([§ 253 Abs. 5 Satz 1, Abs. 1 Satz 1 HGB][hgb253]).
- Vorhersehbare Risiken bis zum Stichtag zählen, auch wenn sie erst bis zur
  Aufstellung bekannt werden ([§ 252 Abs. 1 Nr. 4 HGB][hgb252]).

Prüfe zum Jahresende jede Finanzanlage: Kurs am Stichtag (Depotauszug),
letzter Jahresabschluss und laufende Zahlen der Tochter, Ausfälle bei
Darlehen. Frag nach früheren Abschreibungen und ob ihr Grund noch besteht.

## 2. Teilwert nach EStG

- Beteiligungen und andere nicht abnutzbare Wirtschaftsgüter stehen mit den
  Anschaffungskosten in der Steuerbilanz; der niedrigere Teilwert kann bei
  voraussichtlich dauernder Wertminderung angesetzt werden
  ([§ 6 Abs. 1 Nr. 2 Satz 1, 2 EStG][estg6]).
- Teilwert ist der Betrag, den ein Erwerber des ganzen Betriebs für das
  einzelne Wirtschaftsgut ansetzen würde
  ([§ 6 Abs. 1 Nr. 1 Satz 3 EStG][estg6]).
- In den Folgejahren gilt wieder der höhere Wert, wenn die GmbH den
  niedrigeren Teilwert nicht erneut nachweist
  ([§ 6 Abs. 1 Nr. 2 Satz 3 mit Nr. 1 Satz 4 EStG][estg6]).
- Der Teilwert einer Beteiligung entspricht beim Erwerb den
  Anschaffungskosten; danach zählen Ertragslage und Ertragsaussichten,
  Vermögenswert und funktionale Bedeutung des Unternehmens
  ([H 6.7 EStH „Teilwertvermutungen“][esth67], mit BFH vom 06.11.2003,
  BStBl 2004 II S. 416; Kanzleispiegel).
- Buchfink führt eine Einheitsbilanz. Schreibt die GmbH nach § 253 Abs. 3
  Satz 6 HGB bei nur vorübergehender Minderung ab, weicht die Steuerbilanz ab.
  Wertungsfrage; bei Wertpapieren ohne § 8b-Schutz (Anleihen, Fonds) wirkt das
  auf die Steuer. Lege es dem Menschen mit Empfehlung Steuerberater vor.

## 3. Folgen bei Anteilen an Kapitalgesellschaften

- Teilwertabschreibung und andere Gewinnminderungen beim Anteil mindern das
  Einkommen nicht ([§ 8b Abs. 3 Satz 3 KStG][kstg8b]). Rechne sie hinzu.
- Eine spätere Wertaufholung bleibt außer Ansatz; 5 % davon gelten als nicht
  abziehbar ([§ 8b Abs. 2 Satz 3, Abs. 3 Satz 1 KStG][kstg8b]).
- Bei der Gewerbesteuer wirkt das über den Gewinn mit
  ([§ 7 Satz 1 GewStG][gewstg7]).
- Die Frage „dauernd oder vorübergehend“ entscheidet deshalb über die
  Handelsbilanz, nicht über die Körperschaftsteuer.

## 4. Darlehen an die Tochter

- Ausleihungen sind Finanzanlagen; es gelten Abschnitt 1 und 2.
- Ist die GmbH zu mehr als einem Viertel an der Tochter beteiligt, mindern
  Teilwertabschreibung, Ausfall oder Inanspruchnahme aus einer Sicherheit das
  Einkommen nicht, außer ein fremder Dritter hätte das Darlehen ebenso gewährt
  oder stehen lassen ([§ 8b Abs. 3 Satz 4, 7 KStG][kstg8b]).
- Wertaufholung: außer Ansatz, soweit die Abschreibung nicht abgezogen wurde,
  ohne 5-%-Zuschlag ([§ 8b Abs. 3 Satz 9 KStG][kstg8b]).
- Wertungsfrage: Fremdüblichkeit. Sammle Vertrag, Zins, Sicherheiten,
  Zahlungsverlauf und die Lage der Tochter bei Hingabe.

## 5. Wertänderung in Buchfink

Außerplanmäßige Abschreibung und Zuschreibung eines Anlageguts erfasst der
Mensch in der Anlagenkartei der App („Außerplanmäßig abschreiben“,
„Zuschreiben“); ein Werkzeug dafür gibt es nicht. Du lieferst Betrag, Stichtag und
Grund mit Tatsache und Datum, etwa „Kurs am 31.12.2026 38,20 € gegenüber
Anschaffung 52,00 €, Depotauszug B-2027-0003“. Buchfink verlangt den Grund.

| Anlagegut | Abschreibung | Zuschreibung |
|---|---|---|
| gekennzeichnet als „Anteil an einer Kapitalgesellschaft“ | 7204 Abschreibungen auf Finanzanlagen § 3 Nr. 40 EStG bzw. § 8b Abs. 3 KStG (dauerhaft) | 4913 |
| andere Finanzanlage, dauernde Minderung | 7200 Abschreibungen auf Finanzanlagen (dauerhaft) | 4912 Erträge aus Zuschreibungen des Finanzanlagevermögens |
| andere Finanzanlage, vorübergehende Minderung | 7201 Abschreibungen auf Finanzanlagen (nicht dauerhaft) | 4912 |

- Buchfink bucht beim gekennzeichneten Anteil auch eine vorübergehende
  Minderung auf 7204. Halte im Grund fest, ob die Minderung dauernd ist.
- Wertpapiere des Umlaufvermögens (1510) gehören nicht in die Anlagenkartei.
  Buche ihre Abschreibung mit `buchung_erfassen` (`abschlussperiode: true`):
  Soll 7214 bei Aktien, 7210 bei anderen Papieren, Haben 1510; Zuschreibung
  auf 4916 bzw. 4915. Eigenbeleg mit Kursnachweis.
- Notiert ein Papier in Fremdwährung, rechnet „Währung bewerten“ in der App
  den Euro-Wert zum Stichtag (6880, 4840); Regeln in buchfink-jahresabschluss.
- Vermerke jede Wertänderung als Beteiligungsposten.

## 6. Verkauf: Ablauf

1. **Belege:** notarieller Vertrag bei GmbH-Anteilen
   ([§ 15 Abs. 3, 4 GmbHG][gmbhg15]), Verkaufsabrechnung der Bank bei
   Wertpapieren; Rechnungen von Notar und Beratern über buchfink-belege.
   Fertig, wenn alle Belege abgelegt sind.
2. **Rechnung vorlegen:** Erlös, Veräußerungskosten, Buchwert laut
   Anlagenspiegel, Buchgewinn oder -verlust, steuerliche Folge
   (Abschnitt 7). Fertig mit dem Ja des Menschen.
3. **Abgang (nur App):** Der Mensch erfasst in der Anlagenkartei „Verkauf“
   mit Datum, Erlös, bei Teilverkauf der Stückzahl, und wählt, ob der Erlös
   auf ein Zahlungsmittelkonto oder als offener Posten an den Käufer geht.
   Buchfink wählt die Konten nach Ergebnis und Kennzeichen:

| Anlagegut | Buchgewinn | Buchverlust |
|---|---|---|
| gekennzeichneter Anteil | 4852 Erlöse, 4858 Restbuchwert | 6892 Erlöse, 6898 Restbuchwert |
| andere Finanzanlage, auch Fonds | 4851 Erlöse, 4857 Restbuchwert | 6891 Erlöse, 6897 Restbuchwert |

4. **Zahlung:** Bleibt der Erlös offener Posten, gleiche den Bankumsatz nach
   buchfink-bank aus. Fertig, wenn Anlagekonto und offener Posten null zeigen.
5. **Beteiligungsposten** vormerken und die Beteiligungsliste schließen.

## 7. Verkauf: Steuer

- Der Gewinn bleibt bei der Körperschaftsteuer außer Ansatz, 5 % gelten als
  nicht abziehbar, unabhängig von der Quote
  ([§ 8b Abs. 2, 3 Satz 1, Abs. 4 Satz 1 KStG][kstg8b]); ein Verlust ist nicht
  abziehbar ([§ 8b Abs. 3 Satz 3 KStG][kstg8b]). Veräußerungskosten mindern
  den Gewinn ([§ 8b Abs. 2 Satz 2 KStG][kstg8b]).
- Gewerbesteuer: folgt dem Gewinn ([§ 7 Satz 1 GewStG][gewstg7]); § 8 Nr. 5
  GewStG erfasst nur Gewinnanteile.
- Kapitalertragsteuer: Eine inländische Bank behält auf Gewinne aus dem
  Verkauf von Aktien und Fondsanteilen bei einer unbeschränkt
  steuerpflichtigen GmbH nichts ein
  ([§ 43 Abs. 2 Satz 3 Nr. 1 EStG][estg43] mit Abs. 1 Satz 1 Nr. 9).
- Umsatzsteuer: Der Verkauf von Anteilen ist nicht steuerbar oder nach
  [§ 4 Nr. 8 Buchst. e, f UStG][ustg4] steuerfrei; Vorsteuer aus den Kosten
  des Verkaufs ist in beiden Fällen nicht abziehbar
  ([UStAE 15.22 Abs. 2][ustae1522]), außer beim Verzicht auf die
  Steuerfreiheit nach § 9 Abs. 1 UStG, den Buchfink nicht abbildet. Im Abgang
  wählt der Mensch deshalb einen Steuerfall ohne Umsatzsteuer.
- Ausländische Gesellschaft: Mitteilung an das Finanzamt
  ([§ 138 Abs. 2 Satz 1 Nr. 3 AO][ao138]).

## 8. Teilverkauf, Tilgung, Fonds

- **Teilverkauf:** Ist die Stückzahl geführt, rechnet Buchfink den Anteil der
  Anschaffungskosten aus der verkauften Stückzahl; sonst nennt der Mensch den
  abgehenden Betrag.
- **Tilgung einer Ausleihung:** Abgang „Tilgung“ in der Kartei; gebucht wird
  Geld an Ausleihung, ohne Erlöskonto.
- **Fondsanteile:** nie als Anteil an einer Kapitalgesellschaft gekennzeichnet.
  Beim Abgang zeigt Buchfink die Nebenrechnung mit Teilfreistellung und
  abgezogenen Vorabpauschalen (Referenz zu Investmentfonds, Liste in
  SKILL.md).

[ao138]: https://www.gesetze-im-internet.de/ao_1977/__138.html
[esth67]: https://www.steuerschroeder.de/steuergesetze/esth/6.7
[estg6]: https://www.gesetze-im-internet.de/estg/__6.html
[estg43]: https://www.gesetze-im-internet.de/estg/__43.html
[gewstg7]: https://www.gesetze-im-internet.de/gewstg/__7.html
[gmbhg15]: https://www.gesetze-im-internet.de/gmbhg/__15.html
[hgb252]: https://www.gesetze-im-internet.de/hgb/__252.html
[hgb253]: https://www.gesetze-im-internet.de/hgb/__253.html
[kstg8b]: https://www.gesetze-im-internet.de/kstg_1977/__8b.html
[ustae1522]: https://datenbank.nwb.de/Dokument/378652_15___22/
[ustg4]: https://www.gesetze-im-internet.de/ustg_1980/__4.html
