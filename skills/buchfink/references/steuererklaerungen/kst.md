# Körperschaftsteuer und steuerliches Einlagekonto

Rechtsstand: 17. Oktober 2026. Konten nach dem SKR04 in Buchfink.

## Inhalt

1. Steuerpflicht und Zeitraum
2. Überleitung zum zu versteuernden Einkommen
3. Nicht abziehbare Aufwendungen
4. Verdeckte Gewinnausschüttung und verdeckte Einlage
5. Beteiligungserträge
6. Spenden
7. Abweichungen der Steuerbilanz
8. Verlustabzug
9. Steuer, Rundung, Anrechnung, Vorauszahlungen
10. Steuerliches Einlagekonto
11. Was Buchfink nicht aus den Büchern kennt

## 1. Steuerpflicht und Zeitraum

- Unbeschränkt steuerpflichtig mit Sitz oder Geschäftsleitung im Inland
  ([§ 1 Abs. 1 Nr. 1, Abs. 2 KStG][kstg1]), ab der Vorgesellschaft, also der
  Beurkundung ([H 1.1 KStH][ksth11];
  [gruendung/ablauf.md](../gruendung/ablauf.md)).
- Jahressteuer für das Kalenderjahr, im Gründungs- oder Schlussjahr für den
  Zeitraum der Steuerpflicht ([§ 7 Abs. 3 KStG][kstg7]). Der Gewinn eines
  abweichenden Wirtschaftsjahres gilt im Kalenderjahr seines Endes als
  bezogen ([§ 7 Abs. 4 Satz 1, 2 KStG][kstg7]).
- Kein Freibetrag ([§ 24 Satz 2 Nr. 1 KStG][kstg24]).

## 2. Überleitung zum zu versteuernden Einkommen

`erklaerung` rechnet die Überleitung (Anlagen GK und ZVE) in dieser
Reihenfolge ([§ 8 Abs. 1 Satz 1 KStG][kstg8]). Prüfe jede Zeile gegen ihre
`herkunft`.

| Nr. | Posten | Quelle in Buchfink, Grundlage |
|---|---|---|
| 1 | Jahresüberschuss oder -fehlbetrag | `auswertung` `art: guv` |
| 2 | + gebuchte Körperschaftsteuer und Solidaritätszuschlag | 7600, 7603, 7608, 7609; [§ 10 Nr. 2 KStG][kstg10] |
| 3 | + gebuchte Gewerbesteuer | 7610, 7641 (Soll); [§ 4 Abs. 5b EStG][estg4] |
| 4 | − gebuchte Erstattungen dieser Steuern für Vorjahre | 7604, 7607, 7641 (Haben), 7643; siehe unten |
| 5 | + übrige nicht abziehbare Aufwendungen | Abschnitt 3 |
| 6 | + verdeckte Gewinnausschüttungen | Abschnitt 4 |
| 7 | − verdeckte Einlagen, soweit sie den Gewinn erhöht haben | Abschnitt 4 |
| 8 | − steuerfreie Beteiligungserträge, + 5 % nicht abziehbare Ausgaben | Abschnitt 5 ([§ 8b Abs. 1, 2, 5 KStG][kstg8b]) |
| 9 | ± Abweichungen der Steuerbilanz | Abschnitt 7 |
| 10 | + gebuchte Spenden, − abziehbare Spenden | Abschnitt 6 |
| 11 | = Einkommen vor Verlustabzug | – |
| 12 | − Verlustrücktrag und Verlustvortrag | Bescheid, Abschnitt 8 |
| 13 | = zu versteuerndes Einkommen | [§ 7 Abs. 1, 2 KStG][kstg7] |

Zeile 4: Die Erstattung einer nicht abziehbaren Steuer erhöht das Einkommen
nicht; bei wesentlichem Betrag lass das den Steuerberater bestätigen.

Erstattungszinsen nach § 233a AO: Die Behandlung folgt der BFH-Rechtsprechung
([H 10.1 KStH][ksth101]); Konten je Behandlung in
[bank/steuerzahlungen.md, Abschnitt 4](../bank/steuerzahlungen.md#4-säumniszuschläge-zinsen-verspätungszuschläge).
Wertungsfrage; leg sie mit Empfehlung Steuerberater vor.

## 3. Nicht abziehbare Aufwendungen

| Posten | Grundlage | Konto (SKR04) |
|---|---|---|
| Geschenke über der Freigrenze | [§ 4 Abs. 5 Satz 1 Nr. 1 EStG][estg4] ([belege/bewirtung-geschenke.md](../belege/bewirtung-geschenke.md)) | 6620, 6621, Pauschsteuer 6622 |
| 30 % der Bewirtungskosten | [§ 4 Abs. 5 Satz 1 Nr. 2 EStG][estg4] ([belege/bewirtung-geschenke.md](../belege/bewirtung-geschenke.md)) | 6644 |
| Gästehaus, Jagd, Jacht und Ähnliches | [§ 4 Abs. 5 Satz 1 Nr. 3, 4 EStG][estg4] | 6645 |
| Geldbußen, Ordnungs- und Verwarnungsgelder | [§ 4 Abs. 5 Satz 1 Nr. 8 EStG][estg4] | aus der Buchung |
| Gewerbesteuer und ihre Nebenleistungen | [§ 4 Abs. 5b EStG][estg4] | 7610, 7641, 7302 |
| Spenden an politische Parteien | [§ 4 Abs. 6 EStG][estg4] | 6394 |
| Körperschaftsteuer, Solidaritätszuschlag, Nebenleistungen darauf; Vorsteuer auf Aufwendungen nach § 4 Abs. 5 Satz 1 Nr. 1–4, 7 oder Abs. 7 EStG | [§ 10 Nr. 2 KStG][kstg10] | 7600–7609, 7304, 7308, 6437 |
| Geldstrafen | [§ 10 Nr. 3 KStG][kstg10] | aus der Buchung |
| Hälfte der Aufsichtsratsvergütung | [§ 10 Nr. 4 KStG][kstg10] | 6875 |

Geldbußen, Geldstrafen und nicht abziehbare Vorsteuer auf anderen Konten
suchst du mit `buchungen_suchen` und trägst sie in `kst.gk.nab_weitere` ein.
Die Zuordnung einer einzelnen Ausgabe ist oft eine Wertungsfrage.

## 4. Verdeckte Gewinnausschüttung und verdeckte Einlage

- Die vGA mindert das Einkommen nicht ([§ 8 Abs. 3 Satz 2 KStG][kstg8]):
  Zeile 6, wenn sie den Jahresüberschuss gemindert hat.
- Die verdeckte Einlage erhöht es nicht ([§ 8 Abs. 3 Satz 3 KStG][kstg8]):
  Zeile 7, wenn sie als Ertrag gebucht ist (etwa Forderungsverzicht eines
  Gesellschafters).
- Erkennen: [gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md).
  Jede vGA ist eine Wertungsfrage; du trägst nur bestätigten Betrag mit
  Fundstelle ein.

## 5. Beteiligungserträge

§ 8b KStG: [beteiligungen/steuerfreistellung.md](../beteiligungen/steuerfreistellung.md);
Gewerbesteuer: [beteiligungen/gewst-holding.md](../beteiligungen/gewst-holding.md).

## 6. Spenden

- Abziehbar für steuerbegünstigte Zwecke nach §§ 52 bis 54 AO bis 20 % des
  Einkommens oder 4 Promille der Summe aus Umsätzen, Löhnen und Gehältern
  ([§ 9 Abs. 1 Satz 1 Nr. 2 KStG][kstg9]); Einkommen vor Spenden und
  Verlustabzug ([§ 9 Abs. 2 Satz 1 KStG][kstg9]); der Rest wird vorgetragen
  ([§ 9 Abs. 1 Satz 9 KStG][kstg9]).
- Nur mit Zuwendungsbestätigung nach amtlichem Muster
  ([§ 50 Abs. 1 EStDV][estdv50]); prüfe, dass zu jeder Spende eine abgelegt
  ist.
- Konten: 6391 bis 6393 und 6395 bis 6398 nach Zweck, 6390 nicht abziehbar,
  6394 Parteispenden.
- Wertungsfrage: Steht ein Interesse des Gesellschafters dahinter (sein Verein,
  Familie), droht eine vGA.

## 7. Abweichungen der Steuerbilanz

Abweichende Ansätze passt die E-Bilanz durch Überleitung an, oder sie enthält
eine Steuerbilanz ([§ 5b Abs. 1 Satz 2, 3 EStG][estg5b],
[§ 60 Abs. 2 EStDV][estdv60]). Typisch: Drohverlustrückstellungen sind
steuerlich verboten ([§ 5 Abs. 4a Satz 1 EStG][estg5]). Bewertung:
[jahresabschluss/ablauf.md](../jahresabschluss/ablauf.md). Jede Abweichung ist
eine Wertungsfrage für Mensch und Steuerberater.

## 8. Verlustabzug

- Rücktrag bis 1 Mio. € in die zwei Vorjahre; auf Antrag unterbleibt er
  ([§ 10d Abs. 1 Satz 1, 2, 6 EStG][estg10d]). Darüber entscheidet der
  Mensch.
- Vortrag bis 1 Mio. € unbeschränkt, darüber zu 70 % des übersteigenden
  Gesamtbetrags der Einkünfte ([§ 10d Abs. 2 Satz 1 EStG][estg10d]; ab VZ
  2024, [§ 52 Abs. 18b Satz 3 EStG][estg52]; ab 2028 laut Sekundärquellen
  wieder 60 %).
- Der Verlustvortrag wird gesondert festgestellt
  ([§ 10d Abs. 4 Satz 1 EStG][estg10d]); übernimm ihn nur aus dem Bescheid.
- Schädlicher Beteiligungserwerb: Mehr als 50 % der Anteile binnen fünf Jahren
  an einen Erwerber oder Nahestehende, dann gehen nicht genutzte Verluste unter
  ([§ 8c Abs. 1 Satz 1 KStG][kstg8c]). Wertungsfrage; Ausnahmen prüft ein
  Steuerberater.

## 9. Steuer, Rundung, Anrechnung, Vorauszahlungen

| Schritt | Regel | Grundlage |
|---|---|---|
| Körperschaftsteuer | 15 % bis 2027, 14 % 2028, 13 % 2029, 12 % 2030, 11 % 2031, 10 % ab 2032 | [§ 23 Abs. 1 KStG][kstg23] |
| Rundung | Körperschaftsteuerbeträge zugunsten der GmbH auf volle Euro | [§ 31 Abs. 1 Satz 2 KStG][kstg31] |
| Solidaritätszuschlag | 5,5 % der festgesetzten Körperschaftsteuer, ohne Freigrenze; Bruchteile eines Cents entfallen | [§ 3 Abs. 1 Nr. 1, Abs. 3 SolZG][solzg3], [§ 4 Satz 1, 3 SolZG][solzg4] |
| Anrechnung | Vorauszahlungen; Kapitalertragsteuer und Solidaritätszuschlag darauf nur mit Steuerbescheinigung | [§ 36 Abs. 2 Nr. 1, 2 EStG][estg36] über [§ 31 Abs. 1 Satz 1 KStG][kstg31]; [§ 1 Abs. 2 SolZG][solzg1] |
| Vorauszahlungen | Termine in [grundlagen/pflichten.md#kst-vorauszahlung](../grundlagen/pflichten.md#kst-vorauszahlung); Höhe nach Bescheid ([bescheide.md](bescheide.md#5-vorauszahlungsbescheide)) | – |

Kapitalertragsteuer und Solidaritätszuschlag stehen auf 7630 und 7633; gleiche
die Summe mit allen Steuerbescheinigungen ab
([bank/zinsen-kapest.md](../bank/zinsen-kapest.md),
[beteiligungen/dividenden.md](../beteiligungen/dividenden.md)). Fertig, wenn
jede Abweichung erklärt ist.

## 10. Steuerliches Einlagekonto

- Einlagen außerhalb des Stammkapitals stehen am Ende jedes Wirtschaftsjahres
  auf dem steuerlichen Einlagekonto ([§ 27 Abs. 1 Satz 1, 2 KStG][kstg27]).
- Gesonderte Feststellung; Erklärung jedes Jahr, auch bei Bestand null
  ([§ 27 Abs. 2 Satz 1, 4 KStG][kstg27]).
- Zugänge: Kapitalrücklage aus Zuzahlungen (2928), verdeckte Einlagen, Aufgeld
  bei der Gründung; Liste in
  [gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md).
- Einlagenrückgewähr (Leistungen über den ausschüttbaren Gewinn hinaus,
  [§ 27 Abs. 1 Satz 3, 5 KStG][kstg27]) bildet Buchfink nicht ab; bei Verdacht
  Steuerberater.

## 11. Was Buchfink nicht aus den Büchern kennt

Diese Zeilen zeigt `erklaerung` (`art: kst`, `nur_offen: true`) mit Quelle und
Zuständigkeit ([ablauf.md](ablauf.md#2-ablauf), Schritt 4); die Fachfragen
stehen in den Abschnitten 3, 4, 7, 8 und 10.

[estdv50]: https://www.gesetze-im-internet.de/estdv_1955/__50.html
[estdv60]: https://www.gesetze-im-internet.de/estdv_1955/__60.html
[estg4]: https://www.gesetze-im-internet.de/estg/__4.html
[estg5]: https://www.gesetze-im-internet.de/estg/__5.html
[estg5b]: https://www.gesetze-im-internet.de/estg/__5b.html
[estg10d]: https://www.gesetze-im-internet.de/estg/__10d.html
[estg36]: https://www.gesetze-im-internet.de/estg/__36.html
[estg52]: https://www.gesetze-im-internet.de/estg/__52.html
[kstg1]: https://www.gesetze-im-internet.de/kstg_1977/__1.html
[kstg7]: https://www.gesetze-im-internet.de/kstg_1977/__7.html
[kstg8]: https://www.gesetze-im-internet.de/kstg_1977/__8.html
[kstg8b]: https://www.gesetze-im-internet.de/kstg_1977/__8b.html
[kstg8c]: https://www.gesetze-im-internet.de/kstg_1977/__8c.html
[kstg9]: https://www.gesetze-im-internet.de/kstg_1977/__9.html
[kstg10]: https://www.gesetze-im-internet.de/kstg_1977/__10.html
[kstg23]: https://www.gesetze-im-internet.de/kstg_1977/__23.html
[kstg24]: https://www.gesetze-im-internet.de/kstg_1977/__24.html
[kstg27]: https://www.gesetze-im-internet.de/kstg_1977/__27.html
[kstg31]: https://www.gesetze-im-internet.de/kstg_1977/__31.html
[solzg1]: https://www.gesetze-im-internet.de/solzg_1995/__1.html
[solzg3]: https://www.gesetze-im-internet.de/solzg_1995/__3.html
[solzg4]: https://www.gesetze-im-internet.de/solzg_1995/__4.html
[ksth11]: https://www.steuerschroeder.de/steuergesetze/ksth/1.1
[ksth101]: https://www.steuerschroeder.de/steuergesetze/ksth/10.1
