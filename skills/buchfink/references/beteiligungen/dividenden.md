# Dividenden und andere Bezüge

Rechtsstand: 17. Oktober 2026. Wortlaut von EStG, KStG, SolZG, GmbHG und HGB
am 17.10.2026 gelesen. Verwaltungsauffassung zur Steuerbescheinigung:
BMF-Schreiben vom 16.05.2025, IV C 1 - S 2401/00008/014/051, gelesen über einen
[nichtamtlichen Spiegel][bmf2025]; das BMF-Original war nicht abrufbar.
Sätze: Stand Oktober 2026; sie ändern sich nur durch Gesetz.

Dividenden aus Investmentfonds sind Investmenterträge; sie regelt
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

- Die Gesellschafter der Tochter stellen deren Jahresabschluss fest und
  beschließen über die Verwendung des Ergebnisses
  ([§ 46 Nr. 1 GmbHG][gmbhg46]); erst dann besteht der Anspruch auf den
  verteilbaren Gewinn ([§ 29 Abs. 1 GmbHG][gmbhg29]). Lass dir den Beschluss
  geben und lege ihn mit `beleg_ablegen` ab.
- Gewinne sind nur zu berücksichtigen, wenn sie am Stichtag realisiert sind
  ([§ 252 Abs. 1 Nr. 4 HGB][hgb252]). Erfasse die Dividende deshalb mit dem
  Gewinnverwendungsbeschluss der Tochter.
- Für den Steuerabzug fließt die Dividende an dem Tag zu, den der Beschluss
  als Auszahlungstag bestimmt; fehlt er, am Tag nach dem Beschluss
  ([§ 44 Abs. 2 Satz 1, 2 EStG][estg44]). Mit dem Zufluss entsteht die
  Kapitalertragsteuer ([§ 44 Abs. 1 Satz 2 EStG][estg44]).

Wertungsfrage: **phasengleiche Vereinnahmung.** Soll die Holding die Dividende
der Tochter schon im Jahr erfassen, für das die Tochter ausschüttet, obwohl
der Beschluss erst danach fällt? Steuerlich darf ein beherrschender
Gesellschafter den Anspruch vor dem Beschluss nur aktivieren, wenn am
Stichtag der ausschüttungsfähige Gewinn bekannt war und objektiv nachprüfbar
feststeht, dass die Gesellschafter endgültig zu einer bestimmten Ausschüttung
entschlossen waren; das liegt regelmäßig nicht vor
([BFH vom 20.12.2000, I R 50/95][bfh150-95], zum Beschluss des Großen Senats
vom 07.08.2000, GrS 2/99, BStBl II 2000, 632). Buchfink führt eine
Einheitsbilanz. Eine phasengleiche Erfassung nur in der Handelsbilanz kann es
nicht abbilden. Empfiehl, mit dem Beschluss zu buchen; Abweichungen klärt der
Mensch mit dem Steuerberater.

## 2. Steuerabzug

| Größe | Regel | Fundstelle |
|---|---|---|
| Kapitalertragsteuer mit Solidaritätszuschlag | Abzug auf Gewinnanteile; Satz und Bemessung in [bank/zinsen-kapest.md](../bank/zinsen-kapest.md#2-kapitalertragsteuer-und-solidaritätszuschlag) | [§ 43 Abs. 1 Satz 1 Nr. 1, 1a EStG][estg43] |
| Steuerfreiheit nach § 8b KStG | ändert am Abzug nichts | [§ 43 Abs. 1 Satz 3 EStG][estg43] |
| Wer einbehält | die ausschüttende GmbH; bei Aktien in Sammelverwahrung die auszahlende Bank | [§ 44 Abs. 1 Satz 3 EStG][estg44] |

Beispiel: Die Tochter schüttet 100.000,00 € aus. Kapitalertragsteuer
25.000,00 €, Solidaritätszuschlag 1.375,00 €, Gutschrift 73.625,00 €.

## 3. Buchung

**Fall A: Beschluss und Zahlung im selben Geschäftsjahr.** Steht der Anteil
in der Kartei (`anlagen`, `klasse: finanzanlage`), buche am Bankumsatz mit
`bankumsatz_buchen` und `kapitalertrag`: `anlage_id`, `brutto`,
`kapitalertragsteuer`, `solidaritaetszuschlag`. Buchfink bucht den Ertrag
brutto auf das Dividendenkonto nach der Tabelle unten, die Abzüge getrennt auf
7630 und 7633, und verknüpft den Ertrag mit dem Anteil. Brutto abzüglich der
Abzüge muss der Bankumsatz sein. Aus der Verknüpfung schlägt die
Körperschaftsteuererklärung den Streubesitz nach der Quote des Anteils vor.

| Zeile | Konto (SKR04) | Betrag im Beispiel |
|---|---|---|
| Ertrag brutto | Dividendenkonto nach Tabelle unten | 100.000,00 |
| Kapitalertragsteuer | 7630 Kapitalertragsteuer 25 % | 25.000,00 (Abzug) |
| Solidaritätszuschlag | 7633 Anrechenbarer Solidaritätszuschlag auf Kapitalertragsteuer 25 % | 1.375,00 (Abzug) |
| Summe = Bankumsatz | – | 73.625,00 |

Ohne Anteil in der Kartei buchst du dieselben drei Zeilen mit `aufteilung`
und `konto`: der Ertrag mit positivem `betrag`, die Abzüge mit negativem, wie
bei Zinsen mit Kapitalertragsteuer
([bank/zinsen-kapest.md](../bank/zinsen-kapest.md)). Der Streubesitz bleibt
dann in der Erklärung offen.

**Fall B: Beschluss vor dem Stichtag, Zahlung danach.**

1. Zum Beschluss: `buchung_erfassen` mit dem Beschluss als Beleg, Soll 1260
   Forderungen gegen verbundene Unternehmen (bei Beteiligung ohne Mehrheit
   1280), Haben Dividendenkonto, je mit dem Bruttobetrag.
2. Zur Zahlung: `bankumsatz_buchen` mit `aufteilung`, 1260 bzw. 1280 über
   den Bruttobetrag, 7630 und 7633 als Abzug. Diesen Weg verknüpft Buchfink
   nicht mit dem Anteil; der Streubesitz bleibt in der Erklärung offen.

Die Kapitalertragsteuer wird nach dem Wortlaut auf die Steuer des Jahres
angerechnet, in dem die Bezüge erfasst sind
([§ 36 Abs. 2 Nr. 2 EStG][estg36]), auch wenn du sie erst mit der Zahlung
buchst. Nenne dem Menschen beide Jahre und lass die Zuordnung vom
Steuerberater bestätigen.

**Dividendenkonto:**

| Anteil auf | Dividende auf |
|---|---|
| 0804 (verbundenes Unternehmen) | 7006 Erträge aus Anteilen an Kapitalgesellschaften (verbundene Unternehmen) § 3 Nr. 40 EStG bzw. § 8b Abs. 1 KStG |
| 0820, 0850 (Beteiligung) | 7005 Erträge aus Anteilen an Kapitalgesellschaften (Beteiligung) § 3 Nr. 40 EStG bzw. § 8b Abs. 1 KStG |
| 0900, 0910 (Wertpapiere des Anlagevermögens) | 7014 Erträge aus Anteilen an Kapitalgesellschaften (Finanzanlagevermögen) § 3 Nr. 40 EStG bzw. § 8b Abs. 1 und 4 KStG |
| 1500 (verbundenes Unternehmen, Umlaufvermögen) | 7104 |
| 1510 (Umlaufvermögen) | 7103 Erträge aus Anteilen an Kapitalgesellschaften (Umlaufvermögen) § 3 Nr. 40 EStG bzw. § 8b Abs. 1 und 4 KStG |

Steht eine Dividende auf einem anderen Konto, etwa aus einem älteren Import,
schlägst du die Umbuchung auf das Konto der Tabelle vor (`buchung_erfassen`),
ebenso Kapitalertragsteuer und Solidaritätszuschlag getrennt auf 7630 und 7633.

## 4. Steuerbescheinigung und Anrechnung

- Die ausschüttende Tochter bescheinigt nach denselben Regeln, nach denen die
  GmbH ihren eigenen Gesellschaftern bescheinigt: auf Verlangen, nach der
  Zahlung, nach Muster II
  ([gesellschafter/steuerbescheinigung.md](../gesellschafter/steuerbescheinigung.md)).
  Für Aktien im Depot bescheinigt die Bank
  ([§ 45a Abs. 2 Satz 1 Nr. 2 EStG][estg45a]); für Körperschaften nach
  Muster III ([BMF 16.05.2025, Rn. 57][bmf2025]).
- Angerechnet wird die einbehaltene Steuer auch, soweit die Bezüge nach § 8b
  KStG außer Ansatz bleiben ([§ 36 Abs. 2 Nr. 2 Buchst. b EStG][estg36]).
  Voraussetzungen der Anrechnung und Abgleich von 7630 und 7633 am
  Jahresende:
  [steuererklaerungen/kst.md, Abschnitt 9](../steuererklaerungen/kst.md#9-steuer-rundung-anrechnung-vorauszahlungen).
- Lege jede Bescheinigung mit `beleg_ablegen` ab.

## 5. Sonderfälle

- **Börsennotierte Aktien in Sammelverwahrung:** Die volle Anrechnung setzt
  voraus, dass die GmbH die Aktien 45 Tage um den Fälligkeitstag hält und
  mindestens 70 % des Wertänderungsrisikos trägt; sonst sind drei Fünftel der
  Kapitalertragsteuer nicht anzurechnen
  ([§ 36a Abs. 1 bis 3 EStG][estg36a]). Der Solidaritätszuschlag bleibt
  anrechenbar; [§ 1 Abs. 2 SolZG][solzg1] nimmt § 36a aus. Das gilt nicht, wenn diese
  Kapitalerträge im Jahr 20.000 € nicht übersteigen oder die Aktien bei
  Zufluss seit einem Jahr gehalten werden ([§ 36a Abs. 5 EStG][estg36a]).
  Stand: Oktober 2026. Liegt keiner dieser Fälle vor: Steuerberater.
- **Dauerhafte Überzahlung:** Wäre die Kapitalertragsteuer wegen der Art der
  Geschäfte auf Dauer höher als die festzusetzende Körperschaftsteuer, kann
  das Finanzamt bescheinigen, dass der Abzug unterbleibt
  ([§ 44a Abs. 5 Satz 1, 4 EStG][estg44a]). Das trifft eine Holding mit
  überwiegend steuerfreien Dividenden. Die Vorschrift nennt Erträge nach § 43
  Abs. 1 Satz 1 Nr. 1 (GmbH-Anteile), nicht Nr. 1a (Aktien in
  Sammelverwahrung). Den Antrag stellt der Mensch, am besten mit Steuerberater.
- **Einlagenrückgewähr:** Zahlt die Tochter aus dem steuerlichen
  Einlagekonto, sind das keine Einnahmen
  ([§ 20 Abs. 1 Nr. 1 Satz 3 EStG][estg20]) und keine Bezüge nach § 8b KStG;
  die Tochter bescheinigt den Betrag ([§ 27 Abs. 3 KStG][kstg27]). Buche nicht
  auf ein Dividendenkonto, bevor der Steuerberater die Buchung (Ertrag oder
  Minderung der Anschaffungskosten) und die Folgen bei Übersteigen des
  Buchwerts geklärt hat.
- **Verdeckte Gewinnausschüttung der Tochter an die Holding:**
  [steuerfreistellung.md, Abschnitt 2](steuerfreistellung.md#2-dividenden-und-andere-bezüge).
  Steuerberater.
- **Liquidation oder Kapitalherabsetzung der Tochter:**
  [steuerfreistellung.md, Abschnitte 2 und 4](steuerfreistellung.md#4-gewinne-aus-der-veräußerung).
  Steuerberater.
- **Ausländische Dividenden:** Eine inländische Bank behält bei einer
  unbeschränkt steuerpflichtigen GmbH auf ausländische Dividenden keine
  Kapitalertragsteuer ein ([§ 43 Abs. 2 Satz 3 Nr. 1 EStG][estg43] mit Abs. 1
  Satz 1 Nr. 6). Ausländische Quellensteuer führt das SKR04 auf 7638 und 7639.
  Anrechnung oder Erstattung und die Gewerbesteuer (§ 9 Nr. 7 GewStG):
  Steuerberater.

## 6. Umsatzsteuer

Dividenden sind nicht steuerbar
([holding-umsatzsteuer.md, Abschnitt 1](holding-umsatzsteuer.md#1-unternehmerin-oder-nicht)).
Die Zeilen tragen keinen Steuerfall; in der Voranmeldung erscheint nichts.

## 7. Für die Erklärung

Halte je Dividende in der Beteiligungsliste fest: Datum des Beschlusses und
der Zahlung, Gesellschaft, Quote zu Beginn des Kalenderjahres und des
Erhebungszeitraums, Konto, Bruttobetrag, Kapitalertragsteuer,
Solidaritätszuschlag, `beleg_id` der Bescheinigung. Damit prüfst du zum
Jahresende die Zeilen, die Buchfink aus Konten und Kartei rechnet
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
