# Steuerliche Erfassung

Rechtsstand: 17. Oktober 2026. Wortlaut von UStG und UStDV am 17.10.2026
gelesen. ELSTER-Hinweise aus dem Merkblatt für Existenzgründer der
Finanzverwaltung NRW.

## Inhalt

1. Fragebogen zur steuerlichen Erfassung
2. Angaben, die Buchfink später braucht
3. Voranmeldungszeitraum
4. Was du zum Fragebogen beisteuerst

## 1. Fragebogen zur steuerlichen Erfassung

- Frist, Formular in Mein ELSTER, Nachweis und Folgen:
  [pflichten.md#fragebogen-steuerliche-erfassung](../grundlagen/pflichten.md#fragebogen-steuerliche-erfassung).
  Die Anzeige an die Gemeinde:
  [gruendungsweg.md, Abschnitt 4](gruendungsweg.md#4-behördenwege-mit-fristen).
- Ohne ELSTER-Zertifikat kann sich die Gesellschaft nur mit einer E-Mail-Adresse
  registrieren; damit lassen sich ausschließlich Fragebögen übermitteln. Nach
  Erteilung der Steuernummer wird das Konto in ein vollwertiges umgewandelt. Die
  Registrierung kann bis zu zwei Wochen dauern
  ([Merkblatt NRW][nrwmerkblatt]).
- Der Fragebogen führt zur Steuernummer. Den Fragebogen füllt und sendet der
  Mensch; du bereitest die Angaben vor (Abschnitt 4).

## 2. Angaben, die Buchfink später braucht

Der Fragebogen legt fest, wie das Finanzamt die GmbH führt. Die Antworten
müssen zu den Einstellungen in Buchfink passen; diese ändert nur der Mensch in
der App.

| Angabe | Regel | Buchfink |
|---|---|---|
| Wirtschaftsjahr | Maßgebend ist der Zeitraum, für den die GmbH regelmäßig Abschlüsse macht, meist im Vertrag festgelegt; Wahl bei der Gründung und spätere Umstellung: [rumpfjahr.md, Abschnitt 1](rumpfjahr.md#1-was-ein-rumpfgeschäftsjahr-ist) | `status.mandant.gj_beginn_monat` |
| Besteuerung nach vereinbarten oder vereinnahmten Entgelten | Regel sind vereinbarte Entgelte ([§ 16 Abs. 1 Satz 1 UStG][ustg16]); vereinnahmte auf Antrag, unter anderem bei Gesamtumsatz im Vorjahr bis 800.000 € ([§ 20 Satz 1 Nr. 1 UStG][ustg20]) | Buchfink bildet nur vereinbarte Entgelte ab ([grundlagen/grenzen.md](../grundlagen/grenzen.md#1-was-buchfink-nicht-abbildet)) |
| Kleinunternehmer | Umsätze sind steuerfrei, wenn der Gesamtumsatz im Vorjahr 25.000 € nicht überschritten hat und im laufenden Jahr 100.000 € nicht überschreitet ([§ 19 Abs. 1 Satz 1 UStG][ustg19]); Verzicht unwiderruflich, bindet mindestens fünf Kalenderjahre ([§ 19 Abs. 3 UStG][ustg19]) | Buchfink setzt voraus, dass die GmbH Umsatzsteuer ausweist ([grundlagen/grenzen.md](../grundlagen/grenzen.md#1-was-buchfink-nicht-abbildet)) |
| Voranmeldungszeitraum | Abschnitt 3 | `status.mandant.ust_zeitraum` |
| Dauerfristverlängerung | auf Antrag; Wirkung, Antrag und Sondervorauszahlung: [monatsabschluss/dauerfristverlaengerung.md](../monatsabschluss/dauerfristverlaengerung.md) | `status.mandant.dauerfrist` |
| USt-IdNr. | erteilt das Bundeszentralamt für Steuern auf Antrag; der Antrag nennt Name, Anschrift und Steuernummer ([§ 27a Abs. 1 Satz 1, 4, 5 UStG][ustg27a]) | Pflichtangabe bei EU-Geschäften ([rechnungen/ausland.md](../rechnungen/ausland.md#2-unternehmen-im-übrigen-eu-gebiet)) |
| Gewerbesteuer-Hebesatz | bestimmt die Gemeinde ([§ 16 Abs. 1 GewStG][gewstg16]) | Einstellungen |

Stand: Oktober 2026.

Wertungsfrage, Kleinunternehmer: Liegt der erwartete Umsatz einer neuen GmbH
unter den Grenzen des § 19 Abs. 1 UStG, kann sie ohne Verzicht unter die
Regelung fallen. Wie die Grenzen im Gründungsjahr anzuwenden sind, klärt der
Mensch mit Finanzamt oder Steuerberater. Für die Arbeit mit Buchfink braucht es
den Verzicht nach § 19 Abs. 3 UStG; nenne dem Menschen die Bindung von fünf
Jahren.

Ist-Versteuerung: Beantragt der Mensch die Besteuerung nach vereinnahmten
Entgelten, sag ihm, dass Buchfink sie nicht abbildet.

## 3. Voranmeldungszeitraum

Die Regeln des [§ 18 Abs. 2 UStG][ustg18] stehen in
[monatsabschluss/ustva.md, Abschnitt 2](../monatsabschluss/ustva.md#2-voranmeldungszeitraum).
Folgen für eine Gründung, Stand Oktober 2026:

- Gründung 2026: Ausschlaggebend ist die voraussichtliche Steuer 2026. Über
  9.000 € monatlich, sonst vierteljährlich.
- Gründung ab 2027: monatlich im Gründungsjahr und im Folgejahr.
- Wertungsfrage: Gilt für eine Gründung 2026 im Jahr 2027 Satz 4 (monatlich als
  „folgendes Kalenderjahr“) oder die Umrechnung nach Satz 6? Maßgebend ist, was
  das Finanzamt mitteilt; der Mensch stellt es in der App ein.
- Buchfink schlägt im Einrichtungsassistenten für Gründungen 2021 bis 2026
  pauschal das Vierteljahr vor (`accounting.RecommendedVatPeriod`). Das
  übersieht den Fall einer voraussichtlichen Steuer über 9.000 €; prüfe es.

## 4. Was du zum Fragebogen beisteuerst

Stelle dem Menschen aus Buchfink zusammen:

- Firma, Sitz, Rechtsform, Geschäftsjahr, Bankverbindung (`status`).
- Beurkundungstag, Stammkapital, Gesellschafter mit Anteilen (Unterlagen aus
  [ablauf.md, Abschnitt 2](ablauf.md#2-ablauf), Schritt 3).
- Eröffnungsbilanz: Salden der Gründungskonten zum Beurkundungstag
  (`auswertung` mit `art: kontoblatt` und `bis`).

Der Fragebogen verlangt außerdem Angaben, die nicht in den Büchern stehen und
die der Mensch schätzt oder entscheidet, etwa voraussichtliche Umsätze und
Gewinne, Beginn der Tätigkeit, Beschäftigte, Kleinunternehmerregelung, Art der
Steuerberechnung und Dauerfristverlängerung (Liste ohne Abgleich mit dem
aktuellen Formular; prüfe die Felder dort). Die
Schätzungen beeinflussen Voranmeldungszeitraum und Vorauszahlungen; frag nach
Zahlen und nenne die Folgen.

[gewstg16]: https://www.gesetze-im-internet.de/gewstg/__16.html
[ustg16]: https://www.gesetze-im-internet.de/ustg_1980/__16.html
[ustg18]: https://www.gesetze-im-internet.de/ustg_1980/__18.html
[ustg19]: https://www.gesetze-im-internet.de/ustg_1980/__19.html
[ustg20]: https://www.gesetze-im-internet.de/ustg_1980/__20.html
[ustg27a]: https://www.gesetze-im-internet.de/ustg_1980/__27a.html
[nrwmerkblatt]: https://www.finanzverwaltung.nrw.de/sites/default/files/asset/document/merkblatt_fuer_existenzgruenderinnen_und_existenzgruender.pdf
