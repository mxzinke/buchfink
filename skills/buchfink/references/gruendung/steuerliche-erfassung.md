# Steuerliche Erfassung

Rechtsstand: 17. Oktober 2026.

## Inhalt

1. Fragebogen zur steuerlichen Erfassung
2. Angaben, die Buchfink später braucht
3. Voranmeldungszeitraum
4. Was du zum Fragebogen beisteuerst

## 1. Fragebogen zur steuerlichen Erfassung

- Frist: ein Monat ab der Beurkundung ([§ 137 Abs. 2 AO][ao137],
  [§ 138 Abs. 4 AO][ao138]). Formular, Nachweis und Folgen:
  [pflichten.md#fragebogen-steuerliche-erfassung](../grundlagen/pflichten.md#fragebogen-steuerliche-erfassung);
  Anzeige an die Gemeinde:
  [gruendungsweg.md, Abschnitt 4](gruendungsweg.md#4-behördenwege-mit-fristen).
- Für den Fragebogen genügt eine ELSTER-Registrierung nur mit E-Mail-Adresse;
  nach Erteilung der Steuernummer wird das Konto vollwertig. Die
  Registrierung kann bis zu zwei Wochen dauern
  ([Merkblatt NRW][nrwmerkblatt]).
- Den Fragebogen füllt und sendet der Mensch; du bereitest die Angaben vor
  (Abschnitt 4).

## 2. Angaben, die Buchfink später braucht

Die Antworten im Fragebogen müssen zu den Einstellungen in Buchfink passen.
Steuernummer, Finanzamt, Voranmeldungszeitraum und Besteuerung aus dem
Bescheid schlägst du mit `stammdaten_vorschlagen` vor; übernehmen kann sie nur
der Mensch in der App.

| Angabe | Regel | Buchfink |
|---|---|---|
| Wirtschaftsjahr | Zeitraum der regelmäßigen Abschlüsse, meist im Vertrag ([rumpfjahr.md, Abschnitt 1](rumpfjahr.md#1-was-ein-rumpfgeschäftsjahr-ist)) | `status.mandant.gj_beginn_monat` |
| Vereinbarte oder vereinnahmte Entgelte | Regel: vereinbarte ([§ 16 Abs. 1 Satz 1 UStG][ustg16]); vereinnahmte auf Antrag, unter anderem bei Vorjahresumsatz bis 800.000 € ([§ 20 Satz 1 Nr. 1 UStG][ustg20]) | nur vereinbarte Entgelte ([grundlagen/grenzen.md](../grundlagen/grenzen.md#1-was-buchfink-nicht-abbildet)) |
| Kleinunternehmer | steuerfrei bei Gesamtumsatz im Vorjahr bis 25.000 € und im laufenden Jahr bis 100.000 € ([§ 19 Abs. 1 Satz 1 UStG][ustg19]); Verzicht bindet mindestens fünf Kalenderjahre ([§ 19 Abs. 3 UStG][ustg19]) | setzt Umsatzsteuerausweis voraus ([grundlagen/grenzen.md](../grundlagen/grenzen.md#1-was-buchfink-nicht-abbildet)) |
| Voranmeldungszeitraum | Abschnitt 3 | `status.mandant.ust_zeitraum` |
| Dauerfristverlängerung | auf Antrag ([monatsabschluss/dauerfristverlaengerung.md](../monatsabschluss/dauerfristverlaengerung.md)) | `status.mandant.dauerfrist` |
| USt-IdNr. | Antrag beim Bundeszentralamt für Steuern mit Name, Anschrift und Steuernummer ([§ 27a Abs. 1 Satz 1, 4, 5 UStG][ustg27a]) | Pflichtangabe bei EU-Geschäften ([rechnungen/ausland.md](../rechnungen/ausland.md#2-unternehmen-im-übrigen-eu-gebiet)) |
| Gewerbesteuer-Hebesatz | bestimmt die Gemeinde ([§ 16 Abs. 1 GewStG][gewstg16]) | Einstellungen |

Wertungsfrage, Kleinunternehmer: Unter den Grenzen braucht Buchfink den
Verzicht; nenne die Bindung. Wie die Grenzen im Gründungsjahr gelten, klärt
der Mensch mit Finanzamt oder Steuerberater.

## 3. Voranmeldungszeitraum

Regeln des [§ 18 Abs. 2 UStG][ustg18]:
[monatsabschluss/ustva.md, Abschnitt 2](../monatsabschluss/ustva.md#2-voranmeldungszeitraum).
Für eine Gründung:

- Gründung 2026: maßgebend ist die voraussichtliche Steuer 2026, über
  9.000 € monatlich, sonst vierteljährlich; keine Befreiung im Gründungs- und
  Folgejahr ([UStAE 18.7 Abs. 5 Satz 3][usth187]). Frag nach der Schätzung im
  Fragebogen.
- Gründung ab 2027: monatlich im Gründungs- und im Folgejahr.
- Gründung 2026, Jahr 2027: Satz 6 gilt nur für 2021 bis 2026; für 2027 gilt
  nach dem Wortlaut wieder Satz 4, also monatlich, und so schlägt es die App
  vor. Maßgebend ist die Mitteilung des Finanzamts; du schlägst sie mit
  `stammdaten_vorschlagen` vor.

## 4. Was du zum Fragebogen beisteuerst

- Firma, Sitz, Rechtsform, Geschäftsjahr, Bankverbindung (`status`).
- Beurkundungstag, Stammkapital, Gesellschafter mit Anteilen (Unterlagen aus
  [ablauf.md, Abschnitt 2](ablauf.md#2-ablauf), Schritt 3).
- Eröffnungsbilanz: Salden der Gründungskonten zum Beurkundungstag
  (`auswertung` mit `art: kontoblatt` und `bis`).

Schätzungen und Entscheidungen außerhalb der Bücher (Umsätze, Gewinne,
Beginn der Tätigkeit, Beschäftigte, Wahlrechte aus Abschnitt 2) liefert der
Mensch. Sie bestimmen Voranmeldungszeitraum und Vorauszahlungen; frag nach
Zahlen und nenne die Folgen.

[ao137]: https://www.gesetze-im-internet.de/ao_1977/__137.html
[ao138]: https://www.gesetze-im-internet.de/ao_1977/__138.html
[gewstg16]: https://www.gesetze-im-internet.de/gewstg/__16.html
[ustg16]: https://www.gesetze-im-internet.de/ustg_1980/__16.html
[ustg18]: https://www.gesetze-im-internet.de/ustg_1980/__18.html
[ustg19]: https://www.gesetze-im-internet.de/ustg_1980/__19.html
[ustg20]: https://www.gesetze-im-internet.de/ustg_1980/__20.html
[ustg27a]: https://www.gesetze-im-internet.de/ustg_1980/__27a.html
[nrwmerkblatt]: https://www.finanzverwaltung.nrw.de/sites/default/files/asset/document/merkblatt_fuer_existenzgruenderinnen_und_existenzgruender.pdf
[usth187]: https://usth.bundesfinanzministerium.de/usth/2023/A-Umsatzsteuergesetz/V-Besteuerung/Paragraf-18/ae-18-7.html
