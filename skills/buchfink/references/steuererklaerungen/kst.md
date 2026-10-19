# Körperschaftsteuer und steuerliches Einlagekonto

Rechtsstand: 17. Oktober 2026. Wortlaut von KStG, EStG, EStDV, SolZG und AO am
17.10.2026 auf gesetze-im-internet.de gelesen. Konten nach dem SKR04 in
Buchfink.

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
11. Formulare in Mein ELSTER
12. Was Buchfink rechnet

## 1. Steuerpflicht und Zeitraum

- Eine GmbH oder UG mit Sitz oder Geschäftsleitung im Inland ist mit allen
  Einkünften unbeschränkt steuerpflichtig ([§ 1 Abs. 1 Nr. 1, Abs. 2 KStG][kstg1]).
  Alle Einkünfte gelten als Einkünfte aus Gewerbebetrieb
  ([§ 8 Abs. 2 KStG][kstg8]).
- Die Steuerpflicht beginnt mit der Vorgesellschaft, also mit der Beurkundung
  ([H 1.1 KStH „Beginn der Steuerpflicht“][ksth11];
  [gruendung/ablauf.md](../gruendung/ablauf.md)).
- Die Körperschaftsteuer ist eine Jahressteuer für das Kalenderjahr; besteht
  die Steuerpflicht nicht das ganze Jahr, gilt ihr Zeitraum
  ([§ 7 Abs. 3 KStG][kstg7]). Der Gewinn wird nach dem Wirtschaftsjahr
  ermittelt und gilt im Kalenderjahr seines Endes als bezogen
  ([§ 7 Abs. 4 Satz 1, 2 KStG][kstg7]).
- Zuständig ist das Finanzamt am Ort der Geschäftsleitung
  ([§ 20 Abs. 1 AO][ao20]).
- Die GmbH bekommt keinen Freibetrag: § 24 KStG schließt Körperschaften aus,
  deren Ausschüttungen beim Empfänger Einnahmen nach § 20 Abs. 1 Nr. 1 EStG
  sind ([§ 24 Satz 2 Nr. 1 KStG][kstg24]).

## 2. Überleitung zum zu versteuernden Einkommen

Was als Einkommen gilt, bestimmen EStG und KStG
([§ 8 Abs. 1 Satz 1 KStG][kstg8]). Buchfink rechnet die Überleitung in dieser
Reihenfolge (Anlagen GK und ZVE). Prüfe jede Zeile gegen ihre `herkunft`: Konto,
Beleg oder Bescheid.

| Nr. | Posten | Quelle in Buchfink | Grundlage |
|---|---|---|---|
| 1 | Jahresüberschuss oder -fehlbetrag | `auswertung` `art: guv` | – |
| 2 | + gebuchte Körperschaftsteuer und Solidaritätszuschlag | 7600, 7603, 7608, 7609 | [§ 10 Nr. 2 KStG][kstg10] |
| 3 | + gebuchte Gewerbesteuer | 7610, 7641 (Soll) | [§ 4 Abs. 5b EStG][estg4] |
| 4 | − gebuchte Erstattungen dieser Steuern für Vorjahre | 7604, 7607, 7641 (Haben), 7643 | siehe unten |
| 5 | + übrige nicht abziehbare Aufwendungen | Abschnitt 3 | [§ 4 Abs. 5, 5b, 6 EStG][estg4], [§ 10 KStG][kstg10] |
| 6 | + verdeckte Gewinnausschüttungen | Abschnitt 4 | [§ 8 Abs. 3 Satz 2 KStG][kstg8] |
| 7 | − verdeckte Einlagen, soweit sie den Gewinn erhöht haben | Abschnitt 4 | [§ 8 Abs. 3 Satz 3 KStG][kstg8] |
| 8 | − steuerfreie Beteiligungserträge, + 5 % nicht abziehbare Ausgaben | Abschnitt 5 | [§ 8b Abs. 1, 2, 5 KStG][kstg8b] |
| 9 | ± Abweichungen der Steuerbilanz | Abschnitt 7 | [§ 5b Abs. 1 Satz 2 EStG][estg5b] |
| 10 | + gebuchte Spenden, − abziehbare Spenden | Abschnitt 6 | [§ 9 Abs. 1 Satz 1 Nr. 2 KStG][kstg9] |
| 11 | = Einkommen vor Verlustabzug | – | – |
| 12 | − Verlustrücktrag und Verlustvortrag | Bescheid | [§ 10d EStG][estg10d] |
| 13 | = zu versteuerndes Einkommen | – | [§ 7 Abs. 1, 2 KStG][kstg7] |

Zeile 4: Eine Erstattung von Steuern, die beim Abzug das Einkommen nicht
gemindert haben, erhöht es auch nicht. Buchfink saldiert die Erstattungskonten
in den Zeilen der Steuern. Die SKR04-Bezeichnungen „nach § 4 Abs. 5b EStG“
(7641, 7643) kennzeichnen die Gewerbesteuer. Ist der Betrag wesentlich, lass
die Behandlung vom Steuerberater bestätigen.

Erstattungszinsen nach § 233a AO bucht das SKR04 je nach Behandlung auf 7105,
7106 oder 7107. Die Hinweise zu § 10 KStG verweisen dazu auf die Rechtsprechung
des BFH ([H 10.1 KStH „Erstattungs- und Nachzahlungszinsen“][ksth101]).
Wertungsfrage: Welche Behandlung gilt? Lege sie dem Menschen mit Empfehlung
Steuerberater vor.

## 3. Nicht abziehbare Aufwendungen

| Posten | Grundlage | Konto (SKR04) |
|---|---|---|
| Geschenke über der Freigrenze | [§ 4 Abs. 5 Satz 1 Nr. 1 EStG][estg4] (Regeln in [belege/bewirtung-geschenke.md](../belege/bewirtung-geschenke.md)) | 6620, 6621, Pauschsteuer 6622 |
| 30 % der Bewirtungskosten | [§ 4 Abs. 5 Satz 1 Nr. 2 EStG][estg4] | 6644 |
| Gästehaus, Jagd, Jacht und Ähnliches | [§ 4 Abs. 5 Satz 1 Nr. 3, 4 EStG][estg4] | 6645 |
| Geldbußen, Ordnungs- und Verwarnungsgelder | [§ 4 Abs. 5 Satz 1 Nr. 8 EStG][estg4] | Konto aus der Buchung ablesen |
| Gewerbesteuer und ihre Nebenleistungen | [§ 4 Abs. 5b EStG][estg4] | 7610, 7641, 7302 |
| Spenden an politische Parteien | [§ 4 Abs. 6 EStG][estg4] | 6394 |
| Körperschaftsteuer, Solidaritätszuschlag, Nebenleistungen darauf; Vorsteuer auf Aufwendungen nach § 4 Abs. 5 Satz 1 Nr. 1–4, 7 oder Abs. 7 EStG | [§ 10 Nr. 2 KStG][kstg10] | 7600–7609, 7304, 7308, 6437 |
| Geldstrafen | [§ 10 Nr. 3 KStG][kstg10] | Konto aus der Buchung ablesen |
| Hälfte der Aufsichtsratsvergütung | [§ 10 Nr. 4 KStG][kstg10] | 6875 |

Buchfink rechnet die Konten der Buchungsgruppen mit nicht abziehbarem Teil
und die Steuerkonten in eigenen Zeilen hinzu. Geldbußen, Geldstrafen und nicht
abziehbare Vorsteuer auf anderen Konten gehören in die Zeile der weiteren
nicht abziehbaren Aufwendungen; Buchfink schlägt dort 0 vor und nennt im
Hinweis, wenn auf den Vorsteuerkonten 6860 ff. gebucht ist. Ob eine einzelne
Ausgabe hierher gehört, ist oft eine Wertungsfrage.

## 4. Verdeckte Gewinnausschüttung und verdeckte Einlage

- Eine verdeckte Gewinnausschüttung mindert das Einkommen nicht
  ([§ 8 Abs. 3 Satz 2 KStG][kstg8]). Sie gehört in Zeile 6, wenn sie den
  Jahresüberschuss gemindert hat. Ist auf typischen Konten für Leistungen an
  Gesellschafter gebucht, meldet Buchfink `vga_pruefen`.
- Eine verdeckte Einlage erhöht das Einkommen nicht
  ([§ 8 Abs. 3 Satz 3 KStG][kstg8]). Sie gehört in Zeile 7, wenn sie als
  Ertrag gebucht ist (etwa ein Forderungsverzicht eines Gesellschafters).
- Wann ein Vorgang vGA oder verdeckte Einlage ist, regelt
  [gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md). Beide
  Zeilen verantwortet der Mensch: Du trägst
  nur Betrag und Fundstelle ein, die er bestätigt hat. Jede vGA ist eine
  Wertungsfrage.

## 5. Beteiligungserträge

Dividenden und Gewinne aus der Veräußerung von Anteilen an Kapitalgesellschaften
bleiben unter Bedingungen außer Ansatz; 5 % der Bezüge gelten als nicht
abziehbare Ausgaben ([§ 8b Abs. 1, 2, 5 KStG][kstg8b]). Dividenden aus einer
Beteiligung, die zu Beginn des Kalenderjahres unmittelbar unter 10 % des
Stammkapitals lag, sind steuerpflichtig ([§ 8b Abs. 4 Satz 1 KStG][kstg8b]).
Für Veräußerungsgewinne gilt diese Ausnahme nicht. Buchfink rechnet § 8b aus
den Konten für Beteiligungserträge und der Anlagenkartei; fehlt einem Anteil
die Quote, bleibt die Zeile der steuerpflichtigen Bezüge aus Streubesitz
offen. Die Regeln stehen in
[beteiligungen/steuerfreistellung.md](../beteiligungen/steuerfreistellung.md),
die Gewerbesteuer in
[beteiligungen/gewst-holding.md](../beteiligungen/gewst-holding.md).

## 6. Spenden

- Abziehbar sind Zuwendungen zur Förderung steuerbegünstigter Zwecke nach
  §§ 52 bis 54 AO bis zu 20 % des Einkommens oder 4 Promille der Summe aus
  Umsätzen und Löhnen und Gehältern des Kalenderjahres, vorbehaltlich § 8
  Abs. 3 KStG ([§ 9 Abs. 1 Satz 1 Nr. 2 KStG][kstg9]). Stand: Oktober 2026.
- Einkommen im Sinne des Höchstbetrags ist das Einkommen vor Spenden und vor
  Verlustabzug ([§ 9 Abs. 2 Satz 1 KStG][kstg9]). Was darüber liegt, wird
  vorgetragen ([§ 9 Abs. 1 Satz 9 KStG][kstg9]).
- Voraussetzung ist eine Zuwendungsbestätigung nach amtlichem Muster
  ([§ 50 Abs. 1 EStDV][estdv50]). Lege sie als Beleg ab.
- Konten: 6391 bis 6393 und 6395 bis 6398 je nach Zweck, 6390 für nicht
  abziehbare Zuwendungen, 6394 für Parteispenden (Abschnitt 3).
- Buchfink rechnet Höchstbetrag, Abzug und Vortrag (Anlage Z) aus den
  Spendenkonten. Prüfe, dass zu jeder Spende eine Zuwendungsbestätigung
  abgelegt ist.
- Wertungsfrage: Steht hinter der Spende ein Interesse des Gesellschafters
  (Verein des Gesellschafters, Familie), droht eine vGA.

## 7. Abweichungen der Steuerbilanz

Weichen Ansätze der Handelsbilanz von steuerlichen Vorschriften ab, sind sie
durch Zusätze oder Anmerkungen anzupassen oder eine Steuerbilanz zu übermitteln
([§ 5b Abs. 1 Satz 2, 3 EStG][estg5b], [§ 60 Abs. 2 EStDV][estdv60]).
Beispiel: Rückstellungen für drohende Verluste aus schwebenden Geschäften sind
steuerlich verboten ([§ 5 Abs. 4a Satz 1 EStG][estg5]). Bewertungsfragen
stehen in [jahresabschluss/ablauf.md](../jahresabschluss/ablauf.md). Jede
Abweichung ist eine Wertungsfrage für den Menschen und den Steuerberater.

## 8. Verlustabzug

- Negative Einkünfte werden bis 1 Mio. € in die zwei vorangegangenen
  Veranlagungszeiträume zurückgetragen; der Rücktrag kann auf Antrag
  unterbleiben ([§ 10d Abs. 1 Satz 1, 2, 6 EStG][estg10d]).
- Vorgetragen werden sie bis 1 Mio. € unbeschränkt, darüber zu 70 % des
  übersteigenden Gesamtbetrags der Einkünfte
  ([§ 10d Abs. 2 Satz 1 EStG][estg10d]). Die 70 % gelten seit dem
  Veranlagungszeitraum 2024 ([§ 52 Abs. 18b Satz 3 EStG][estg52]); laut
  Sekundärquellen sinkt der Satz ab 2028 wieder auf 60 %. Stand: Oktober 2026;
  prüfe den Wortlaut für das bearbeitete Jahr.
- Der verbleibende Verlustvortrag wird gesondert festgestellt
  ([§ 10d Abs. 4 Satz 1 EStG][estg10d]). Übernimm ihn nur aus diesem Bescheid;
  den Abzug mit Mindestbesteuerung rechnet Buchfink.
- Über den Rücktrag entscheidet der Mensch. Ohne Eintrag in der Zeile zum
  Verlustrücktrag nimmt das Finanzamt ihn in größtmöglicher Höhe vor.
- Werden innerhalb von fünf Jahren mehr als 50 % der Anteile an einen
  Erwerber übertragen (schädlicher Beteiligungserwerb), gehen nicht genutzte
  Verluste unter ([§ 8c Abs. 1 Satz 1 KStG][kstg8c]). Wertungsfrage;
  Ausnahmen prüft ein Steuerberater.

## 9. Steuer, Rundung, Anrechnung, Vorauszahlungen

| Schritt | Regel | Grundlage |
|---|---|---|
| Körperschaftsteuer | 15 % bis 2027, 14 % 2028, 13 % 2029, 12 % 2030, 11 % 2031, 10 % ab 2032 | [§ 23 Abs. 1 KStG][kstg23] |
| Rundung | einzelne Körperschaftsteuerbeträge zugunsten der GmbH auf volle Euro | [§ 31 Abs. 1 Satz 2 KStG][kstg31] |
| Solidaritätszuschlag | 5,5 % der festgesetzten Körperschaftsteuer; die Freigrenze gilt nur für Einkommensteuerpflichtige; Bruchteile eines Cents bleiben außer Ansatz | [§ 3 Abs. 1 Nr. 1, Abs. 3 SolZG][solzg3], [§ 4 Satz 1, 3 SolZG][solzg4] |
| Anrechnung | Vorauszahlungen und Kapitalertragsteuer, Letztere nur mit Steuerbescheinigung | [§ 36 Abs. 2 Nr. 1, 2 EStG][estg36] über [§ 31 Abs. 1 Satz 1 KStG][kstg31] |
| Vorauszahlungen | Termine in [grundlagen/pflichten.md#kst-vorauszahlung](../grundlagen/pflichten.md#kst-vorauszahlung); Höhe nach Bescheid ([bescheide.md](bescheide.md#5-vorauszahlungsbescheide)) | – |

Stand: Oktober 2026. Kapitalertragsteuer und Solidaritätszuschlag auf
Zinsen und Dividenden stehen auf 7630 und 7633; gleiche die Summe mit den
Steuerbescheinigungen ab ([bank/zinsen-kapest.md](../bank/zinsen-kapest.md),
[beteiligungen/dividenden.md](../beteiligungen/dividenden.md)).

## 10. Steuerliches Einlagekonto

- Die GmbH weist Einlagen, die nicht ins Stammkapital gehen, am Schluss jedes
  Wirtschaftsjahres auf dem steuerlichen Einlagekonto aus und schreibt es fort
  ([§ 27 Abs. 1 Satz 1, 2 KStG][kstg27]).
- Der Bestand wird gesondert festgestellt; die Erklärung ist jedes Jahr
  abzugeben ([§ 27 Abs. 2 Satz 1, 4 KStG][kstg27]), auch bei Bestand null.
- Leistungen an Gesellschafter mindern das Konto nur, soweit sie den
  ausschüttbaren Gewinn übersteigen (Einlagenrückgewähr,
  [§ 27 Abs. 1 Satz 3, 5 KStG][kstg27]). Buchfink bildet die Einlagenrückgewähr
  nicht ab; bei Verdacht Steuerberater.
- Zugänge: Kapitalrücklage aus Zuzahlungen (2928), verdeckte Einlagen,
  Aufgeld bei der Gründung. Die Liste führt
  [gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md).

## 11. Formulare in Mein ELSTER

Die Formularliste von Mein ELSTER führt die „Körperschaftsteuererklärung
(KSt 1)“ ([Formularliste][elsterformulare], geprüft am 17.10.2026). Nach
einem Leitfaden zum Vordruck 2025 ist KSt 1 der Mantelbogen, der auch die
Erklärung zum steuerlichen Einlagekonto aufnimmt; der Gewinn steht im Vordruck
GK, das Einkommen in der Anlage ZVE, dazu nach Bedarf die Anlagen Verluste, Z
(Zuwendungsvortrag), Zinsschranke und AE ([Haufe, Leitfaden 2025][haufekst1];
Sekundärquelle). Prüfe die Anlagen im Formular des jeweiligen Jahres.

Die Erklärung ist elektronisch zu übermitteln
([§ 31 Abs. 1a Satz 1 KStG][kstg31]). Den Ablauf der Abgabe beschreibt
[abgabe.md](abgabe.md#4-eingabe-in-mein-elster).

## 12. Was Buchfink rechnet

`erklaerung` (`art: kst`) rechnet die Anlagen GK, ZVE, Z, Verluste und WA und
die Erklärung zum Einlagekonto aus Büchern und Anlagenkartei, mit dem
Steuersatz des Jahres. Jede Zeile nennt `schluessel`, `wert`, `status`,
`zustaendig` und `herkunft`. Was nicht in den Büchern steht, bleibt `offen`
oder `vorgeschlagen`:

| Angabe | `schluessel` | Quelle | `zustaendig` |
|---|---|---|---|
| Verlustvortrag zum Schluss des Vorjahres | `kst.v.z14` | Feststellungsbescheid | mensch |
| Verlustrücktrag | `kst.v.z20` | Entscheidung des Menschen | mensch |
| Einlagekonto zum Schluss des Vorjahres | `kst.f.bestand_vorjahr` | Feststellungsbescheid | mensch |
| Verdeckte Gewinnausschüttung, verdeckte Einlage | `kst.gk.vga`, `kst.gk.einlage` | Abschnitt 4 | mensch |
| Schädlicher Beteiligungserwerb | `kst.wa.z11e` | Gesellschafterliste | mensch |
| Steuerpflichtige Bezüge aus Streubesitz | `kst.gk.8b_streubesitz` | Quote laut Anlagenkartei | agent_rueckfrage |
| Weitere nicht abziehbare Aufwendungen | `kst.gk.nab_weitere` | Abschnitt 3 | agent_rueckfrage |
| Abweichungen der Steuerbilanz | `kst.gk.stb` | Abschnitt 7 | agent_rueckfrage |

Den Vorschlag zur Steuerrückstellung (`abschlussbuchung`,
`art: steuerrueckstellung`) rechnet Buchfink nach denselben Regeln; seine
Eingaben und Konten stehen in
[jahresabschluss/steuern-im-abschluss.md](../jahresabschluss/steuern-im-abschluss.md).
Eine Überzahlung bucht er als Erstattungsanspruch, außer du gibst mit
`kst_betrag` oder `gewst_betrag` einen abweichenden Betrag vor. Bei einem
Verlust setzt Buchfink keine Steuer an.

[ao20]: https://www.gesetze-im-internet.de/ao_1977/__20.html
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
[solzg3]: https://www.gesetze-im-internet.de/solzg_1995/__3.html
[solzg4]: https://www.gesetze-im-internet.de/solzg_1995/__4.html
[ksth11]: https://www.steuerschroeder.de/steuergesetze/ksth/1.1
[ksth101]: https://www.steuerschroeder.de/steuergesetze/ksth/10.1
[elsterformulare]: https://www.elster.de/eportal/formulare-leistungen/alleformulare
[haufekst1]: https://www.haufe.de/id/beitrag/leitfaden-2025-vordruck-kst-1-HI16941392.html
