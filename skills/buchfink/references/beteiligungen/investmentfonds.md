# Investmentfonds: Teilfreistellung und Vorabpauschale

Rechtsstand: 17. Oktober 2026. Wortlaut von InvStG und EStG am 17.10.2026 auf
gesetze-im-internet.de gelesen. Basiszins aus den BMF-Schreiben vom
05.01.2024, IV C 1 - S 1980-1/19/10038 :008, BStBl I S. 154
([NWB-Wiedergabe][bmf2024basis]), vom 10.01.2025, IV C 1 - S
1980/00230/009/002 ([BMF][bmf2025basis]), und vom 13.01.2026, IV C 1 - S
1980/00230/012/001 ([BMF][bmf2026basis]), am 19.10.2026 gelesen.
Steuerbescheinigung: BMF-Schreiben vom
16.05.2025, IV C 1 - S 2401/00008/014/051 ([Spiegel][bmf2025]).

## Inhalt

1. Was ein Investmentfonds ist
2. Ausweis und Buchung
3. Steuerliche Grundregeln
4. Teilfreistellung
5. Vorabpauschale
6. Verkauf und Verlust
7. Kapitalertragsteuer und Bescheinigung
8. Was Buchfink rechnet und was nicht
9. Beispiel
10. Grenzen

## 1. Was ein Investmentfonds ist

- Investmentfonds sind Investmentvermögen nach dem Kapitalanlagegesetzbuch
  ([§ 1 Abs. 1, 2 InvStG][invstg1]); dazu gehören ETF und Publikumsfonds.
  Keine Investmentfonds sind unter anderem REIT-Aktiengesellschaften und
  Investmentvermögen in Form einer Personengesellschaft, soweit sie keine OGAW
  oder Altersvorsorgevermögenfonds sind ([§ 1 Abs. 3 InvStG][invstg1]).
- Die Fondsart folgt aus den Anlagebedingungen:

| Fondsart | Anlagebedingungen verlangen fortlaufend | Fundstelle |
|---|---|---|
| Aktienfonds | mehr als 50 % des Aktivvermögens in Kapitalbeteiligungen | [§ 2 Abs. 6 InvStG][invstg2] |
| Mischfonds | mindestens 25 % in Kapitalbeteiligungen | [§ 2 Abs. 7 InvStG][invstg2] |
| Immobilienfonds | mehr als 50 % in Immobilien und Immobilien-Gesellschaften | [§ 2 Abs. 9 Satz 1 InvStG][invstg2] |
| Auslands-Immobilienfonds | mehr als 50 % in ausländischen Immobilien | [§ 2 Abs. 9 Satz 2 InvStG][invstg2] |
| sonstiger Investmentfonds | keine dieser Quoten | – |

- Lege die Anlagebedingungen oder den Verkaufsprospekt als Beleg ab.
  Wertungsfrage, wenn sie keine Quote nennen.

## 2. Ausweis und Buchung

- Daueranlage: 0900 Wertpapiere des Anlagevermögens; kurzfristige Anlage: 1510
  Sonstige Wertpapiere.
- Anlagegut: Fondsart, Stückzahl und ISIN bildet Buchfink noch nicht ab,
  weder als Werkzeug noch in der App; halte sie in der Beteiligungsliste fest.
  Ein Fonds ist nie „Anteil an einer Kapitalgesellschaft“.
  Buchfink leitet die Anlegerstellung aus der Rechtsform ab; für GmbH und UG
  gilt die Stellung als Anleger, der dem KStG unterliegt.
- Ausschüttung: `bankumsatz_buchen`, Ertrag brutto auf 7010
  (Anlagevermögen) bzw. 7100 (Umlaufvermögen), Kapitalertragsteuer auf 7630,
  Solidaritätszuschlag auf 7633. Nie auf die §-8b-Konten. Steht der Fonds als
  Anlagegut in der Kartei, buche mit `kapitalertrag` (`anlage_id`, `brutto`,
  `kapitalertragsteuer`, `solidaritaetszuschlag`); erst dann rechnet die
  Erklärung für die Ausschüttung die Teilfreistellung.
- Vorabpauschale: keine Buchung. Handelsrechtlich fließt nichts zu; sie wirkt
  nur in der Erklärung (Abschnitt 5).
- Belastet die Bank Anfang Januar Kapitalertragsteuer auf die Vorabpauschale:
  `bankumsatz_buchen` mit 7630 und 7633.

## 3. Steuerliche Grundregeln

- Investmenterträge sind Ausschüttungen, Vorabpauschalen und Gewinne aus der
  Veräußerung ([§ 16 Abs. 1 InvStG][invstg16]). Ausschüttungen umfassen den
  Steuerabzug ([§ 2 Abs. 11 InvStG][invstg2]).
- § 8b KStG gilt für Investmenterträge nicht ([§ 16 Abs. 3 InvStG][invstg16]).
- Betriebsausgaben, Betriebsvermögensminderungen und Veräußerungskosten im
  Zusammenhang mit Aktien-, Misch- oder Immobilienfonds sind in dem Umfang
  nicht abziehbar, in dem die Erträge freigestellt sind
  ([§ 21 InvStG][invstg21]).

## 4. Teilfreistellung

| Fondsart | Körperschaftsteuer (GmbH, UG) | Gewerbesteuer | Fundstelle |
|---|---|---|---|
| Aktienfonds | 80 % | 40 % | [§ 20 Abs. 1 Satz 3, Abs. 5 InvStG][invstg20] |
| Mischfonds | 40 % | 20 % | [§ 20 Abs. 2, 5 InvStG][invstg20] |
| Immobilienfonds | 60 % | 30 % | [§ 20 Abs. 3 Satz 1, Abs. 5 InvStG][invstg20] |
| Auslands-Immobilienfonds | 80 % | 40 % | [§ 20 Abs. 3 Satz 2, Abs. 5 InvStG][invstg20] |
| sonstiger Investmentfonds | 0 % | 0 % | – |

Stand: Oktober 2026. Die Immobilienteilfreistellung schließt die
Aktienteilfreistellung aus ([§ 20 Abs. 3 Satz 3 InvStG][invstg20]).
Freigestellt ist ein Teil „der Erträge“, also aller Investmenterträge nach
§ 16 Abs. 1: Ausschüttung, Vorabpauschale und Veräußerungsgewinn.

- Erreicht der Fonds die Quote nach seinen Bedingungen nicht, tatsächlich
  aber durchgehend, kann die GmbH die Teilfreistellung in der Veranlagung
  beantragen ([§ 20 Abs. 4 InvStG][invstg20]). Steuerberater.
- Ändert sich der Satz oder fällt die Voraussetzung weg, gilt der Anteil als
  veräußert und neu angeschafft ([§ 22 Abs. 1 InvStG][invstg22]).
  Steuerberater.

## 5. Vorabpauschale

**Rechnung** ([§ 18 InvStG][invstg18]):

1. Basisertrag = Rücknahmepreis zu Beginn des Kalenderjahres × 70 % des
   Basiszinses (Abs. 1 Satz 2). Ohne Rücknahmepreis gilt der Börsen- oder
   Marktpreis (Satz 4).
2. Der Basisertrag ist begrenzt auf den Mehrbetrag zwischen erstem und letztem
   Rücknahmepreis des Jahres zuzüglich der Ausschüttungen (Satz 3).
3. Vorabpauschale = Basisertrag minus Ausschüttungen des Jahres, nicht unter
   null (Satz 1).
4. Im Erwerbsjahr ein Zwölftel weniger je vollem Monat vor dem Erwerbsmonat
   (Abs. 2).
5. Sie gilt am ersten Werktag des folgenden Kalenderjahres als zugeflossen
   (Abs. 3).

**Basiszins** (Abs. 4: Bundesbank errechnet, BMF veröffentlicht):

| Jahr | Basiszins | gilt als zugeflossen am | Quelle |
|---|---|---|---|
| 2024 | 2,29 % | 02.01.2025 | [BMF vom 05.01.2024][bmf2024basis] |
| 2025 | 2,53 % | 02.01.2026 | [BMF vom 10.01.2025][bmf2025basis] |
| 2026 | 3,20 % | 04.01.2027 | [BMF vom 13.01.2026][bmf2026basis] |

Stand: Oktober 2026. Den Basiszins für 2027 veröffentlicht das BMF im Januar
2027; trag ihn erst danach ein.

**Steuer.** Die Vorabpauschale ist ein Investmentertrag; die Teilfreistellung
gilt für sie. Für die GmbH mit einem Aktienfonds sind 20 % der
Vorabpauschale bei der Körperschaftsteuer und 60 % bei der Gewerbesteuer
anzusetzen. In der Erklärung steht sie in der Zeile `kst.gk.inv_vorabpauschale`
und die Teilfreistellung in `kst.gk.inv_teilfreistellung`.

Wertungsfrage: **In welchem Jahr setzt die GmbH die Vorabpauschale an?** Das
Gesetz fingiert den Zufluss am ersten Werktag des Folgejahres. Ob die
bilanzierende GmbH sie im Jahr der Entstehung oder im Jahr des fiktiven
Zuflusses erfasst, klärt der Mensch mit dem Steuerberater; danach gilt die
Wahl jedes Jahr gleich.

**In Buchfink:** Die Vorabpauschale am Anlagegut festzuhalten bildet Buchfink
noch nicht ab, weder als Werkzeug noch in der App. Nimm den Betrag aus der
Bescheinigung der Bank (Abschnitt 7) und trag ihn nach Rückfrage mit
`erklaerung_eingabe` in die Zeile `kst.gk.inv_vorabpauschale` ein; die
Teilfreistellung (`kst.gk.inv_teilfreistellung`) ebenso.

## 6. Verkauf und Verlust

- Den Abgang aus der Kartei bildet Buchfink noch nicht ab; die Konten wären
  4851 und 4857 bei Buchgewinn, 6891 und 6897 bei Buchverlust.
- Steuerpflichtig ist der Gewinn abzüglich der über die Besitzzeit
  angesetzten Vorabpauschalen, davon der nicht freigestellte Teil.
- [§ 19 Abs. 1 Satz 3 InvStG][invstg19] ordnet den Abzug der Vorabpauschalen
  ausdrücklich für Anteile an, die nicht zu einem Betriebsvermögen gehören.
  Ob er bei der GmbH ebenso gilt, weil sie dort schon versteuert wurden, lass
  vom Steuerberater bestätigen.
- Ein Verlust ist bei einem Aktienfonds für die GmbH nur zu 20 % abziehbar
  ([§ 21 InvStG][invstg21] mit [§ 20 Abs. 1 Satz 3 InvStG][invstg20]); ebenso
  eine Teilwertabschreibung. Die übrigen 80 % rechnet Buchfink nicht hinzu;
  leg den Betrag dem Menschen vor. Zeile 48 der Gewerbesteuererklärung nimmt
  die gesamten Teilfreistellungen „nach den §§ 20 und 21 InvStG“ auf
  ([gewst-holding.md, Abschnitt 6](gewst-holding.md#6-investmentfonds)). Ob
  § 21 damit bei der Gewerbesteuer nur zur Hälfte wirkt, sagt das Gesetz nicht
  ausdrücklich; Steuerberater.

## 7. Kapitalertragsteuer und Bescheinigung

- Auf Ausschüttungen und Vorabpauschalen behält die verwahrende Bank 25 %
  Kapitalertragsteuer ein ([§ 43 Abs. 1 Satz 1 Nr. 5][estg43],
  [§ 43a Abs. 1 Satz 1 Nr. 1][estg43a], [§ 44 Abs. 1 Satz 3 EStG][estg44]),
  dazu 5,5 % Solidaritätszuschlag. Stand: Oktober 2026.
- Beim Abzug berücksichtigt sie nur die Teilfreistellung für Privatanleger
  (30 % bei Aktienfonds); die 80 % der GmbH wirken erst in der Veranlagung
  ([§ 43a Abs. 2 Satz 1 EStG][estg43a]; [BMF 16.05.2025, Rn. 59][bmf2025]).
- Gewinne aus dem Verkauf:
  [bewertung-veraeusserung.md, Abschnitt 7](bewertung-veraeusserung.md#7-verkauf-steuer).
- Die Bank bescheinigt Körperschaften nach Muster III
  ([BMF 16.05.2025, Rn. 57][bmf2025]). Der nachrichtliche Teil gliedert die
  Investmenterträge vor Teilfreistellung und die als zugeflossen geltenden
  Vorabpauschalen auf (Rn. 59). Daraus nimmst du die Werte für die
  Erklärung.

## 8. Was Buchfink rechnet und was nicht

| Buchfink rechnet | Buchfink rechnet nicht |
|---|---|
| Teilfreistellung je Fondsart aus den Daten der Kartei; Gewerbesteuer nach [gewst-holding.md, Abschnitt 6](gewst-holding.md#6-investmentfonds) | Fondsart, Ertrag, Vorabpauschale und Abgang in der Kartei erfassen: noch nicht abgebildet; ohne diese Daten bleiben die Zeilen offen |
| | Vorabpauschale; den Betrag nennt die Bescheinigung der Bank |
| | anteiliges Abzugsverbot nach § 21 für Verluste und Abschreibungen |

## 9. Beispiel

Rechnerisches Beispiel ohne Beratung. Aktien-ETF im Anlagevermögen,
gehalten seit Jahren, Bestandswert 100.000,00 € am Jahresanfang 2025 und
112.000,00 € am Jahresende, keine Ausschüttung.

| Schritt | Betrag |
|---|---|
| Basisertrag 100.000,00 € × 70 % × 2,53 % | 1.771,00 € |
| Begrenzung auf Wertzuwachs 12.000,00 € | greift nicht |
| Vorabpauschale 2025 | 1.771,00 € |
| steuerpflichtig Körperschaftsteuer (20 %) | 354,20 € |
| steuerpflichtig Gewerbesteuer (60 %), also zusätzlich | 708,40 € |

Verkauf 2026 mit Buchgewinn 15.000,00 €: Gewinn nach Abzug der
Vorabpauschale 13.229,00 €; steuerpflichtig bei der Körperschaftsteuer 20 %,
also 2.645,80 €. Korrektur: − 15.000,00 € + 2.645,80 €. Bei der
Gewerbesteuer zusätzlich 40 % von 13.229,00 €, also 5.291,60 €.

## 10. Grenzen

Spezial-Investmentfonds, Anteile, die vor 2018 erworben wurden (Übergang nach
§ 56 InvStG), Dach-Investmentfonds mit eigenen Quotenregeln und ausländische
Fonds mit Steuerfragen nach Doppelbesteuerungsabkommen: Steuerberater.

[bmf2025]: https://www.steuerschroeder.de/blog/wp-content/uploads/2025/05/2025-05-16-kapitalertragSt-steuerbescheinigung.pdf
[bmf2024basis]: https://datenbank.nwb.de/Dokument/1033913/
[bmf2025basis]: https://www.bundesfinanzministerium.de/Content/DE/Downloads/BMF_Schreiben/Steuerarten/Investmentsteuer/2025-01-10-basiszins-vorabpauschale-zum-2-1-2025.pdf
[bmf2026basis]: https://www.bundesfinanzministerium.de/Content/DE/Downloads/BMF_Schreiben/Steuerarten/Investmentsteuer/2026-01-13-basiszins-berechnung-vorabpauschale.pdf
[estg43]: https://www.gesetze-im-internet.de/estg/__43.html
[estg43a]: https://www.gesetze-im-internet.de/estg/__43a.html
[estg44]: https://www.gesetze-im-internet.de/estg/__44.html
[invstg1]: https://www.gesetze-im-internet.de/invstg_2018/__1.html
[invstg2]: https://www.gesetze-im-internet.de/invstg_2018/__2.html
[invstg16]: https://www.gesetze-im-internet.de/invstg_2018/__16.html
[invstg18]: https://www.gesetze-im-internet.de/invstg_2018/__18.html
[invstg19]: https://www.gesetze-im-internet.de/invstg_2018/__19.html
[invstg20]: https://www.gesetze-im-internet.de/invstg_2018/__20.html
[invstg21]: https://www.gesetze-im-internet.de/invstg_2018/__21.html
[invstg22]: https://www.gesetze-im-internet.de/invstg_2018/__22.html
