# Dividenden und andere Bezüge

Rechtsstand: 17. Oktober 2026. Wortlaut von EStG, KStG, SolZG, GmbHG und HGB
am 17.10.2026 gelesen. Verwaltungsauffassung zur Steuerbescheinigung:
BMF-Schreiben vom 16.05.2025, IV C 1 - S 2401/00008/014/051, gelesen über einen
[nichtamtlichen Spiegel][bmf2025]; das BMF-Original war nicht abrufbar.
Sätze: Stand Oktober 2026; sie ändern sich nur durch Gesetz.

Dividenden aus Investmentfonds sind Investmenterträge; sie regelt die
Referenz zu Investmentfonds (Liste in SKILL.md).

## Inhalt

1. Anspruch und Zeitpunkt
2. Steuerabzug
3. Buchung
4. Steuerbescheinigung und Anrechnung
5. Sonderfälle
6. Umsatzsteuer
7. Beteiligungsposten

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
| Kapitalertragsteuer | 25 % der Dividende | [§ 43 Abs. 1 Satz 1 Nr. 1, 1a][estg43], [§ 43a Abs. 1 Satz 1 Nr. 1 EStG][estg43a] |
| Bemessung | volle Dividende ohne Abzug | [§ 43a Abs. 2 Satz 1 EStG][estg43a] |
| Steuerfreiheit nach § 8b KStG | ändert am Abzug nichts | [§ 43 Abs. 1 Satz 3 EStG][estg43] |
| Solidaritätszuschlag | 5,5 % der Kapitalertragsteuer; Bruchteile eines Cents bleiben außer Ansatz | [§ 3 Abs. 1 Nr. 5][solzg3], [§ 4 Satz 1, 3 SolZG][solzg4] |
| Wer einbehält | die ausschüttende GmbH; bei Aktien in Sammelverwahrung die auszahlende Bank | [§ 44 Abs. 1 Satz 3 EStG][estg44] |

Beispiel: Die Tochter schüttet 100.000,00 € aus. Kapitalertragsteuer
25.000,00 €, Solidaritätszuschlag 1.375,00 €, Gutschrift 73.625,00 €.

## 3. Buchung

**Fall A: Beschluss und Zahlung im selben Geschäftsjahr.** Buche am
Bankumsatz mit `bankumsatz_buchen`, drei Zeilen `konto`:

| Zeile | Konto (SKR04) | Betrag im Beispiel |
|---|---|---|
| Ertrag brutto | Dividendenkonto nach Tabelle unten | 100.000,00 |
| Kapitalertragsteuer | 7630 Kapitalertragsteuer 25 % | 25.000,00 (Abzug) |
| Solidaritätszuschlag | 7633 Anrechenbarer Solidaritätszuschlag auf Kapitalertragsteuer 25 % | 1.375,00 (Abzug) |
| Summe = Bankumsatz | – | 73.625,00 |

Der Ertrag hat einen positiven `betrag`, die Abzugszeilen einen negativen,
wie bei Zinsen mit Kapitalertragsteuer (buchfink-bank).

**Fall B: Beschluss vor dem Stichtag, Zahlung danach.**

1. Zum Beschluss: `buchung_erfassen` mit dem Beschluss als Beleg, Soll 1260
   Forderungen gegen verbundene Unternehmen (bei Beteiligung ohne Mehrheit
   1280), Haben Dividendenkonto, je mit dem Bruttobetrag.
2. Zur Zahlung: `bankumsatz_buchen` mit 1260 bzw. 1280 über den
   Bruttobetrag, 7630 und 7633 als Abzug.

Die Kapitalertragsteuer wird nach dem Wortlaut auf die Steuer des Jahres
angerechnet, in dem die Bezüge erfasst sind
([§ 36 Abs. 2 Nr. 2 EStG][estg36]), auch wenn du sie erst mit der Zahlung
buchst. Vermerke im Beteiligungsposten beide Jahre und lass die Zuordnung vom
Steuerberater bestätigen.

**Dividendenkonto:**

| Anteil auf | Dividende auf |
|---|---|
| 0804 (verbundenes Unternehmen) | 7006 Erträge aus Anteilen an Kapitalgesellschaften (verbundene Unternehmen) § 3 Nr. 40 EStG bzw. § 8b Abs. 1 KStG |
| 0820, 0850 (Beteiligung) | 7005 Erträge aus Anteilen an Kapitalgesellschaften (Beteiligung) § 3 Nr. 40 EStG bzw. § 8b Abs. 1 KStG |
| 0900, 0910 (Wertpapiere des Anlagevermögens) | 7014 Erträge aus Anteilen an Kapitalgesellschaften (Finanzanlagevermögen) § 3 Nr. 40 EStG bzw. § 8b Abs. 1 und 4 KStG |
| 1500 (verbundenes Unternehmen, Umlaufvermögen) | 7104 |
| 1510 (Umlaufvermögen) | 7103 Erträge aus Anteilen an Kapitalgesellschaften (Umlaufvermögen) § 3 Nr. 40 EStG bzw. § 8b Abs. 1 und 4 KStG |

**Weg über die App:** „Ertrag buchen“ in der Anlagenkartei verknüpft den
Ertrag mit dem Anlagegut, wählt aber 7000, 7010 oder 7011 und bucht
Kapitalertragsteuer und Solidaritätszuschlag als eine Zeile auf 7630. Nutzt
der Mensch diesen Weg, trennst du die Beträge im Beteiligungsposten nach der
Bescheinigung.

## 4. Steuerbescheinigung und Anrechnung

- Die ausschüttende GmbH bescheinigt auf Verlangen
  ([§ 45a Abs. 2 Satz 1 Nr. 1 EStG][estg45a]) nach Muster II
  ([BMF 16.05.2025, Rn. 43][bmf2025]); erst nach der Zahlung, weil der
  Zahlungstag zu bescheinigen ist (Rn. 47). Für Aktien im Depot bescheinigt
  die Bank ([§ 45a Abs. 2 Satz 1 Nr. 2 EStG][estg45a]); für Körperschaften
  nach Muster III (Rn. 57).
- Für Zuflüsse ab dem 01.01.2026 gelten die Muster des Schreibens vom
  16.05.2025, davor die vom 23.05.2022 (Rn. 71).
- Angerechnet wird die einbehaltene Steuer auch, soweit die Bezüge nach § 8b
  KStG außer Ansatz bleiben ([§ 36 Abs. 2 Nr. 2 Buchst. b EStG][estg36]), und
  nur mit Bescheinigung ([§ 36 Abs. 2 Nr. 2 Satz 2 EStG][estg36]); bei der GmbH
  über [§ 31 Abs. 1 KStG][kstg31], der Solidaritätszuschlag entsprechend
  ([§ 1 Abs. 2 SolZG][solzg1]).
- Lege jede Bescheinigung mit `beleg_ablegen` ab. Gleiche am Jahresende die
  Summe auf 7630 und 7633 mit allen Bescheinigungen ab (Dividenden hier,
  Zinsen in buchfink-bank). Fertig, wenn jede Abweichung erklärt ist.

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
  ([§ 20 Abs. 1 Nr. 1 Satz 3 EStG][estg20]); die Tochter bescheinigt den
  Betrag ([§ 27 Abs. 3 KStG][kstg27]). Buche nicht auf ein Dividendenkonto,
  bevor der Steuerberater die Buchung (Ertrag oder Minderung der
  Anschaffungskosten) und die Folgen bei Übersteigen des Buchwerts geklärt hat.
- **Verdeckte Gewinnausschüttung der Tochter an die Holding:** Sie ist ein
  sonstiger Bezug ([§ 20 Abs. 1 Nr. 1 Satz 2 EStG][estg20]). Bei der Holding
  bleibt sie nur außer Ansatz, soweit sie das Einkommen der Tochter nicht
  gemindert hat ([§ 8b Abs. 1 Satz 2 KStG][kstg8b]). Steuerberater.
- **Liquidation oder Kapitalherabsetzung der Tochter:** Bezüge nach
  [§ 20 Abs. 1 Nr. 2 EStG][estg20] fallen unter [§ 8b Abs. 1][kstg8b], Gewinne
  aus Auflösung oder Herabsetzung unter § 8b Abs. 2 Satz 3 KStG.
  Steuerberater.
- **Ausländische Dividenden:** Eine inländische Bank behält bei einer
  unbeschränkt steuerpflichtigen GmbH auf ausländische Dividenden keine
  Kapitalertragsteuer ein ([§ 43 Abs. 2 Satz 3 Nr. 1 EStG][estg43] mit Abs. 1
  Satz 1 Nr. 6). Ausländische Quellensteuer führt das SKR04 auf 7638 und 7639.
  Anrechnung oder Erstattung und die Gewerbesteuer (§ 9 Nr. 7 GewStG):
  Steuerberater.

## 6. Umsatzsteuer

Dividenden sind kein Entgelt für eine Leistung und damit nicht steuerbar
([UStAE 2.3 Abs. 2 Satz 3][ustae23]). Die Zeilen tragen keinen Steuerfall; in
der Voranmeldung erscheint nichts.

## 7. Beteiligungsposten

Je Dividende: Datum des Beschlusses und der Zahlung, Gesellschaft, Quote zu
Beginn des Kalenderjahres und des Erhebungszeitraums, Konto, Bruttobetrag,
Kapitalertragsteuer, Solidaritätszuschlag, `beleg_id` der Bescheinigung.
Die Korrekturen dazu berechnest du mit den Referenzen zu § 8b KStG und zur
Gewerbesteuer (Liste in SKILL.md).

[bmf2025]: https://www.steuerschroeder.de/blog/wp-content/uploads/2025/05/2025-05-16-kapitalertragSt-steuerbescheinigung.pdf
[bfh150-95]: https://datenbank.nwb.de/Dokument/97545/
[ustae23]: https://datenbank.nwb.de/Dokument/378652_2___3/
[estg20]: https://www.gesetze-im-internet.de/estg/__20.html
[estg36]: https://www.gesetze-im-internet.de/estg/__36.html
[estg36a]: https://www.gesetze-im-internet.de/estg/__36a.html
[estg43]: https://www.gesetze-im-internet.de/estg/__43.html
[estg43a]: https://www.gesetze-im-internet.de/estg/__43a.html
[estg44]: https://www.gesetze-im-internet.de/estg/__44.html
[estg44a]: https://www.gesetze-im-internet.de/estg/__44a.html
[estg45a]: https://www.gesetze-im-internet.de/estg/__45a.html
[gmbhg29]: https://www.gesetze-im-internet.de/gmbhg/__29.html
[gmbhg46]: https://www.gesetze-im-internet.de/gmbhg/__46.html
[hgb252]: https://www.gesetze-im-internet.de/hgb/__252.html
[kstg8b]: https://www.gesetze-im-internet.de/kstg_1977/__8b.html
[kstg27]: https://www.gesetze-im-internet.de/kstg_1977/__27.html
[kstg31]: https://www.gesetze-im-internet.de/kstg_1977/__31.html
[solzg1]: https://www.gesetze-im-internet.de/solzg_1995/__1.html
[solzg3]: https://www.gesetze-im-internet.de/solzg_1995/__3.html
[solzg4]: https://www.gesetze-im-internet.de/solzg_1995/__4.html
