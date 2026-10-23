# Investmentfonds: Teilfreistellung und Vorabpauschale

Rechtsstand: 17. Oktober 2026. Wortlaut von InvStG und EStG am 17.10.2026 auf
gesetze-im-internet.de gelesen. Basiszins aus den BMF-Schreiben in
Abschnitt 5, am 19.10.2026 gelesen. Steuerbescheinigung: BMF-Schreiben vom
16.05.2025, IV C 1 - S 2401/00008/014/051 ([Spiegel][bmf2025]).

## Inhalt

1. Was ein Investmentfonds ist
2. Ausweis und Buchung
3. Steuerliche Grundregeln
4. Teilfreistellung
5. Vorabpauschale
6. Verkauf und Verlust
7. Kapitalertragsteuer und Bescheinigung
8. Beispiel
9. Grenzen

## 1. Was ein Investmentfonds ist

Investmentfonds sind Investmentvermögen nach dem Kapitalanlagegesetzbuch,
etwa ETF und Publikumsfonds; ausgenommen sind unter anderem
REIT-Aktiengesellschaften ([§ 1 Abs. 1 bis 3 InvStG][invstg1]). Die Fondsart
folgt aus den Anlagebedingungen:

| Fondsart | Anlagebedingungen verlangen fortlaufend | Fundstelle |
|---|---|---|
| Aktienfonds | mehr als 50 % des Aktivvermögens in Kapitalbeteiligungen | [§ 2 Abs. 6 InvStG][invstg2] |
| Mischfonds | mindestens 25 % in Kapitalbeteiligungen | [§ 2 Abs. 7 InvStG][invstg2] |
| Immobilienfonds | mehr als 50 % in Immobilien und Immobilien-Gesellschaften | [§ 2 Abs. 9 Satz 1 InvStG][invstg2] |
| Auslands-Immobilienfonds | mehr als 50 % in ausländischen Immobilien | [§ 2 Abs. 9 Satz 2 InvStG][invstg2] |
| sonstiger Investmentfonds | keine dieser Quoten | – |

Lege Anlagebedingungen oder Verkaufsprospekt als Beleg ab. Nennen sie keine
Quote: Wertungsfrage.

## 2. Ausweis und Buchung

- Daueranlage 0900 Wertpapiere des Anlagevermögens, kurzfristige Anlage 1510
  Sonstige Wertpapiere. Die Fondsart nimmst du beim Aktivieren auf
  (`abschlussbuchung`, `art: anlage_aktivieren`, `fondsart`). Ein Fonds ist
  nie „Anteil an einer Kapitalgesellschaft“.
- Ausschüttung: `bankumsatz_buchen`, Ertrag brutto auf 7010
  (Anlagevermögen) bzw. 7100 (Umlaufvermögen), 7630 und 7633 als Abzug; nie
  auf §-8b-Konten. Steht der Fonds mit Fondsart in der Kartei, buche mit
  `kapitalertrag`; nur dann rechnet die Erklärung die Teilfreistellung der
  Ausschüttung.
- Vorabpauschale: keine Buchung; sie wirkt nur in der Erklärung
  (Abschnitt 5). Die Kapitalertragsteuer darauf, Anfang Januar von der Bank
  belastet: `bankumsatz_buchen` mit 7630 und 7633.

## 3. Steuerliche Grundregeln

- Investmenterträge sind Ausschüttungen (einschließlich Steuerabzug),
  Vorabpauschalen und Veräußerungsgewinne ([§ 16 Abs. 1 InvStG][invstg16];
  [§ 2 Abs. 11 InvStG][invstg2]).
- § 8b KStG gilt für sie nicht ([§ 16 Abs. 3 InvStG][invstg16]).
- Betriebsausgaben, Vermögensminderungen und Veräußerungskosten bei Aktien-,
  Misch- und Immobilienfonds sind im Umfang der Teilfreistellung nicht
  abziehbar ([§ 21 InvStG][invstg21]).

## 4. Teilfreistellung

| Fondsart | Körperschaftsteuer (GmbH, UG) | Gewerbesteuer | Fundstelle |
|---|---|---|---|
| Aktienfonds | 80 % | 40 % | [§ 20 Abs. 1 Satz 3, Abs. 5 InvStG][invstg20] |
| Mischfonds | 40 % | 20 % | [§ 20 Abs. 2, 5 InvStG][invstg20] |
| Immobilienfonds | 60 % | 30 % | [§ 20 Abs. 3 Satz 1, Abs. 5 InvStG][invstg20] |
| Auslands-Immobilienfonds | 80 % | 40 % | [§ 20 Abs. 3 Satz 2, Abs. 5 InvStG][invstg20] |
| sonstiger Investmentfonds | 0 % | 0 % | – |

- Freigestellt sind alle Investmenterträge (Abschnitt 3), auch die
  Vorabpauschale. Die Immobilienteilfreistellung schließt die
  Aktienteilfreistellung aus ([§ 20 Abs. 3 Satz 3 InvStG][invstg20]).
- Erreicht der Fonds die Quote tatsächlich durchgehend, nach seinen
  Bedingungen aber nicht, kann die GmbH die Teilfreistellung in der
  Veranlagung beantragen ([§ 20 Abs. 4 InvStG][invstg20]); Steuerberater.
- Ändert sich der Satz oder entfällt die Voraussetzung, gilt der Anteil als
  veräußert und neu angeschafft ([§ 22 Abs. 1 InvStG][invstg22]);
  Steuerberater.

## 5. Vorabpauschale

**Rechnung** ([§ 18 InvStG][invstg18]):

1. Basisertrag = Rücknahmepreis zu Beginn des Kalenderjahres × 70 % des
   Basiszinses (Abs. 1 Satz 2); ohne Rücknahmepreis Börsen- oder Marktpreis
   (Satz 4).
2. Höchstens der Wertzuwachs zwischen erstem und letztem Rücknahmepreis des
   Jahres zuzüglich der Ausschüttungen (Satz 3).
3. Vorabpauschale = Basisertrag − Ausschüttungen des Jahres, nicht unter null
   (Satz 1).
4. Im Erwerbsjahr ein Zwölftel weniger je vollem Monat vor dem Erwerbsmonat
   (Abs. 2).
5. Zufluss am ersten Werktag des folgenden Kalenderjahres (Abs. 3).

**Basiszins** (Abs. 4):

| Jahr | Basiszins | gilt als zugeflossen am | Quelle |
|---|---|---|---|
| 2024 | 2,29 % | 02.01.2025 | [BMF vom 05.01.2024][bmf2024basis] |
| 2025 | 2,53 % | 02.01.2026 | [BMF vom 10.01.2025][bmf2025basis] |
| 2026 | 3,20 % | 04.01.2027 | [BMF vom 13.01.2026][bmf2026basis] |

Den Basiszins für 2027 veröffentlicht das BMF im Januar 2027; trag ihn erst
danach ein.

Wertungsfrage: **In welchem Jahr setzt die GmbH die Vorabpauschale an,** im
Jahr der Entstehung oder des fiktiven Zuflusses? Der Mensch klärt das mit dem
Steuerberater; danach gilt die Wahl jedes Jahr gleich.

**In Buchfink:** Nach dem Jahresende `abschlussbuchung`
(`art: vorabpauschale`) mit den Rücknahmepreisen der Bank und einem Basiszins
außerhalb der Tabelle; gleiche das Ergebnis mit der Bescheinigung der Bank ab
(Abschnitt 7). Die Erklärung setzt die Vorabpauschale im Jahr des fiktiven
Zuflusses an (`kst.gk.inv_vorabpauschale`, `kst.gk.inv_teilfreistellung`);
ohne Fondsart in der Kartei trägst du beide Zeilen nach Rückfrage mit
`erklaerung_eingabe` ein.

## 6. Verkauf und Verlust

- Abgang und Konten:
  [bewertung-veraeusserung.md, Abschnitt 6](bewertung-veraeusserung.md#6-verkauf-ablauf).
- Steuerpflichtig ist der nicht freigestellte Teil des Gewinns nach Abzug der
  angesetzten Vorabpauschalen. [§ 19 Abs. 1 Satz 3 InvStG][invstg19] nennt
  den Abzug ausdrücklich nur für Privatvermögen; dass er bei der GmbH ebenso
  gilt, lass vom Steuerberater bestätigen.
- Ein Verlust oder eine Teilwertabschreibung ist bei einem Aktienfonds nur zu
  20 % abziehbar ([§ 21 InvStG][invstg21] mit
  [§ 20 Abs. 1 Satz 3 InvStG][invstg20]). Die übrigen 80 % rechnet Buchfink
  nicht hinzu; leg den Betrag dem Menschen vor. Wirkung bei der
  Gewerbesteuer ([gewst-holding.md, Abschnitt 6](gewst-holding.md#6-investmentfonds)):
  Steuerberater.

## 7. Kapitalertragsteuer und Bescheinigung

- Auf Ausschüttungen und Vorabpauschalen behält die verwahrende Bank 25 %
  Kapitalertragsteuer und 5,5 % Solidaritätszuschlag ein
  ([§ 43 Abs. 1 Satz 1 Nr. 5][estg43], [§ 43a Abs. 1 Satz 1 Nr. 1][estg43a],
  [§ 44 Abs. 1 Satz 3 EStG][estg44]).
- Sie berücksichtigt nur die Teilfreistellung für Privatanleger (30 % bei
  Aktienfonds); die 80 % der GmbH wirken erst in der Veranlagung
  ([§ 43a Abs. 2 Satz 1 EStG][estg43a]; [BMF 16.05.2025, Rn. 59][bmf2025]).
- Auf Verkaufsgewinne behält sie nichts ein
  ([bewertung-veraeusserung.md, Abschnitt 7](bewertung-veraeusserung.md#7-verkauf-steuer)).
- Die Bank bescheinigt Körperschaften nach Muster III
  ([BMF 16.05.2025, Rn. 57][bmf2025]); der nachrichtliche Teil nennt
  Investmenterträge vor Teilfreistellung und Vorabpauschalen (Rn. 59). Daraus
  nimmst du die Werte für die Erklärung.

## 8. Beispiel

Aktien-ETF im Anlagevermögen, seit Jahren gehalten, Wert 100.000,00 € am
Jahresanfang 2025, 112.000,00 € am Jahresende, keine Ausschüttung.

| Schritt | Betrag |
|---|---|
| Basisertrag 100.000,00 € × 70 % × 2,53 % (Wertzuwachs 12.000,00 € begrenzt nicht) | 1.771,00 € |
| steuerpflichtig Körperschaftsteuer (20 %) | 354,20 € |
| steuerpflichtig Gewerbesteuer (60 %) | 1.062,60 € |

## 9. Grenzen

Spezial-Investmentfonds, vor 2018 erworbene Anteile (§ 56 InvStG),
Dach-Investmentfonds und ausländische Fonds mit Fragen nach
Doppelbesteuerungsabkommen: Steuerberater.

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
