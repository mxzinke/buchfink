# Dividenden und andere Bezüge

Rechtsstand: 17. Oktober 2026. Wortlaut von EStG, KStG, SolZG, GmbHG und HGB
am 17.10.2026 gelesen. Steuerbescheinigung: BMF-Schreiben vom 16.05.2025,
IV C 1 - S 2401/00008/014/051, gelesen über einen
[nichtamtlichen Spiegel][bmf2025].

Ausschüttungen von Investmentfonds regelt
[investmentfonds.md](investmentfonds.md).

## Inhalt

1. Anspruch und Zeitpunkt
2. Steuerabzug
3. Buchung
4. Steuerbescheinigung und Anrechnung
5. Sonderfälle
6. Umsatzsteuer
7. Für die Erklärung

## 1. Anspruch und Zeitpunkt

- Der Anspruch entsteht mit dem Gewinnverwendungsbeschluss der Tochter
  ([§ 29 Abs. 1][gmbhg29], [§ 46 Nr. 1 GmbHG][gmbhg46]); erst dann ist der
  Ertrag realisiert ([§ 252 Abs. 1 Nr. 4 HGB][hgb252]). Erfasse die Dividende
  mit dem Beschluss und lege ihn mit `beleg_ablegen` ab.
- Für den Steuerabzug fließt die Dividende am Auszahlungstag laut Beschluss
  zu, ohne Bestimmung am Tag nach dem Beschluss; dann entsteht die
  Kapitalertragsteuer ([§ 44 Abs. 1 Satz 2, Abs. 2 Satz 1, 2 EStG][estg44]).

Wertungsfrage **phasengleiche Vereinnahmung** (im Jahr, für das die Tochter
ausschüttet): steuerlich nur, wenn am Stichtag der ausschüttungsfähige Gewinn
und der endgültige Ausschüttungswille objektiv nachprüfbar feststanden
([BFH vom 20.12.2000, I R 50/95][bfh150-95], nach GrS 2/99, BStBl II 2000,
632). Buchfink führt eine Einheitsbilanz. Empfiehl, mit dem Beschluss zu
buchen; Abweichungen klärt der Mensch mit dem Steuerberater.

## 2. Steuerabzug

| Größe | Regel | Fundstelle |
|---|---|---|
| Kapitalertragsteuer mit Solidaritätszuschlag | Abzug auf Gewinnanteile; Satz und Bemessung in [bank/zinsen-kapest.md](../bank/zinsen-kapest.md#2-kapitalertragsteuer-und-solidaritätszuschlag) | [§ 43 Abs. 1 Satz 1 Nr. 1, 1a EStG][estg43] |
| Steuerfreiheit nach § 8b KStG | ändert am Abzug nichts | [§ 43 Abs. 1 Satz 3 EStG][estg43] |
| Wer einbehält | die ausschüttende GmbH; bei Aktien in Sammelverwahrung die auszahlende Bank | [§ 44 Abs. 1 Satz 3 EStG][estg44] |

## 3. Buchung

**Fall A: Beschluss und Zahlung im selben Geschäftsjahr.** Steht der Anteil
in der Kartei (`anlagen`, `klasse: finanzanlage`), buche am Bankumsatz mit
`bankumsatz_buchen` und `kapitalertrag`; Buchfink wählt das Dividendenkonto
aus dem Anlagekonto.

| Zeile | Konto (SKR04) | bei 100.000,00 € brutto |
|---|---|---|
| Ertrag brutto | Dividendenkonto nach Tabelle unten | 100.000,00 |
| Kapitalertragsteuer | 7630 | 25.000,00 (Abzug) |
| Solidaritätszuschlag | 7633 | 1.375,00 (Abzug) |
| Summe = Bankumsatz | – | 73.625,00 |

Ohne Anteil in der Kartei buchst du dieselben Zeilen mit `aufteilung` und
`konto`: Ertrag positiv, Abzüge negativ
([bank/zinsen-kapest.md](../bank/zinsen-kapest.md)).

**Fall B: Beschluss vor dem Stichtag, Zahlung danach.**

1. Zum Beschluss: `buchung_erfassen` mit dem Beschluss als Beleg, Soll 1260
   Forderungen gegen verbundene Unternehmen (Beteiligung ohne Mehrheit:
   1280), Haben Dividendenkonto, brutto.
2. Zur Zahlung: `bankumsatz_buchen` mit `aufteilung`, 1260 bzw. 1280 brutto,
   7630 und 7633 als Abzug.

Angerechnet wird die Kapitalertragsteuer im Jahr, in dem die Bezüge erfasst
sind ([§ 36 Abs. 2 Nr. 2 EStG][estg36]), auch wenn du sie erst mit der
Zahlung buchst. Nenne dem Menschen beide Jahre; die Zuordnung bestätigt der
Steuerberater.

**Dividendenkonto:**

| Anteil auf | Dividende auf |
|---|---|
| 0804 (verbundenes Unternehmen) | 7006 |
| 0820, 0850 (Beteiligung) | 7005 |
| 0900, 0910 (Wertpapiere des Anlagevermögens) | 7014 |
| 1500 (verbundenes Unternehmen, Umlaufvermögen) | 7104 |
| 1510 (Umlaufvermögen) | 7103 |

Steht eine Dividende auf einem anderen Konto, etwa aus einem Import, schlag
die Umbuchung auf das Konto der Tabelle vor (`buchung_erfassen`), ebenso
Kapitalertragsteuer und Solidaritätszuschlag getrennt auf 7630 und 7633.

## 4. Steuerbescheinigung und Anrechnung

- Die Tochter bescheinigt auf Verlangen nach der Zahlung nach Muster II
  ([gesellschafter/steuerbescheinigung.md](../gesellschafter/steuerbescheinigung.md));
  für Aktien im Depot die Bank ([§ 45a Abs. 2 Satz 1 Nr. 2 EStG][estg45a]),
  für Körperschaften nach Muster III ([BMF 16.05.2025, Rn. 57][bmf2025]).
- Angerechnet wird auch die Steuer auf Bezüge, die nach § 8b KStG außer Ansatz
  bleiben ([§ 36 Abs. 2 Nr. 2 Buchst. b EStG][estg36]). Voraussetzungen und
  Abgleich von 7630 und 7633:
  [steuererklaerungen/kst.md, Abschnitt 9](../steuererklaerungen/kst.md#9-steuer-rundung-anrechnung-vorauszahlungen).

## 5. Sonderfälle

- **Börsennotierte Aktien, Dividenden über 20.000 € im Jahr, weniger als ein
  Jahr gehalten:** Drei Fünftel der Kapitalertragsteuer können nicht
  anrechenbar sein ([§ 36a EStG][estg36a]; Solidaritätszuschlag bleibt
  anrechenbar, [§ 1 Abs. 2 SolZG][solzg1]). Steuerberater.
- **Dauerhafte Überzahlung** (Holding mit überwiegend steuerfreien
  Dividenden aus GmbH-Anteilen): Auf Antrag bescheinigt das Finanzamt, dass
  der Abzug unterbleibt ([§ 44a Abs. 5 Satz 1, 4 EStG][estg44a]); den Antrag
  stellt der Mensch.
- **Einlagenrückgewähr** aus dem steuerlichen Einlagekonto der Tochter:
  keine Einnahme ([§ 20 Abs. 1 Nr. 1 Satz 3 EStG][estg20]), kein Bezug nach
  § 8b KStG, von der Tochter bescheinigt ([§ 27 Abs. 3 KStG][kstg27]). Nie auf
  ein Dividendenkonto; gebucht wird, wenn der Steuerberater Ertrag oder
  Minderung der Anschaffungskosten geklärt hat.
- **Verdeckte Gewinnausschüttung, Liquidation oder Kapitalherabsetzung der
  Tochter:**
  [steuerfreistellung.md, Abschnitte 2 und 4](steuerfreistellung.md#2-dividenden-und-andere-bezüge);
  Steuerberater.
- **Ausländische Dividenden:** Eine inländische Bank behält bei der GmbH
  keine Kapitalertragsteuer ein ([§ 43 Abs. 2 Satz 3 Nr. 1 EStG][estg43] mit
  Abs. 1 Satz 1 Nr. 6). Ausländische Quellensteuer auf 7638 und 7639.
  Anrechnung, Erstattung und § 9 Nr. 7 GewStG: Steuerberater.

## 6. Umsatzsteuer

Dividenden sind nicht steuerbar
([holding-umsatzsteuer.md, Abschnitt 1](holding-umsatzsteuer.md#1-unternehmerin-oder-nicht));
die Zeilen haben keinen Steuerfall.

## 7. Für die Erklärung

Halte je Dividende in der Beteiligungsliste fest: Datum von Beschluss und
Zahlung, Gesellschaft, Quote zu Beginn des Kalenderjahres und des
Erhebungszeitraums, Konto, Bruttobetrag, Kapitalertragsteuer,
Solidaritätszuschlag, `beleg_id` der Bescheinigung
([ablauf.md, Abschnitt 2](ablauf.md#2-ablauf-für-jeden-vorgang), Schritt 7).

[bmf2025]: https://www.steuerschroeder.de/blog/wp-content/uploads/2025/05/2025-05-16-kapitalertragSt-steuerbescheinigung.pdf
[bfh150-95]: https://datenbank.nwb.de/Dokument/97545/
[estg20]: https://www.gesetze-im-internet.de/estg/__20.html
[estg36]: https://www.gesetze-im-internet.de/estg/__36.html
[estg36a]: https://www.gesetze-im-internet.de/estg/__36a.html
[estg43]: https://www.gesetze-im-internet.de/estg/__43.html
[estg44]: https://www.gesetze-im-internet.de/estg/__44.html
[estg44a]: https://www.gesetze-im-internet.de/estg/__44a.html
[estg45a]: https://www.gesetze-im-internet.de/estg/__45a.html
[gmbhg29]: https://www.gesetze-im-internet.de/gmbhg/__29.html
[gmbhg46]: https://www.gesetze-im-internet.de/gmbhg/__46.html
[hgb252]: https://www.gesetze-im-internet.de/hgb/__252.html
[kstg27]: https://www.gesetze-im-internet.de/kstg_1977/__27.html
[solzg1]: https://www.gesetze-im-internet.de/solzg_1995/__1.html
