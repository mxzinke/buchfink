# Gewinnausschüttung und Kapitalertragsteuer

Rechtsstand: 17. Oktober 2026. Sätze: Stand Oktober 2026, Fundstellen in der
Tabelle; sie ändern sich nur durch Gesetz.

## Inhalt

1. Voraussetzungen
2. Grenzen der Ausschüttung
3. Steuerabzug
4. Zufluss, Anmeldung, Abführung
5. Buchungen in Buchfink
6. Was Buchfink nicht rechnet
7. Vorabausschüttung

## 1. Voraussetzungen

- Die Gesellschafter stellen den Jahresabschluss fest und beschließen über die
  Verwendung des Ergebnisses ([§ 46 Nr. 1 GmbHG][gmbhg46]). Anspruch besteht auf
  den Jahresüberschuss zuzüglich Gewinnvortrag und abzüglich Verlustvortrag,
  soweit Gesetz, Satzung oder Beschluss ihn nicht binden
  ([§ 29 Abs. 1 GmbHG][gmbhg29]).
- Verteilt wird nach Geschäftsanteilen, wenn die Satzung nichts anderes
  bestimmt ([§ 29 Abs. 3 GmbHG][gmbhg29]).
- Beschluss in der Versammlung oder schriftlich in Textform mit Zustimmung
  aller ([§ 48 Abs. 1, 2 GmbHG][gmbhg48]). Hält ein Gesellschafter alle Anteile,
  nimmt er unverzüglich eine Niederschrift auf und unterschreibt sie
  ([§ 48 Abs. 3 GmbHG][gmbhg48]).
- Im Beschluss steht der Auszahlungstag. Er bestimmt den Zufluss (Abschnitt 4).
- Lege Feststellungs- und Verwendungsbeschluss mit `beleg_ablegen` ab.

## 2. Grenzen der Ausschüttung

- **Kapitalerhaltung:** Das zur Erhaltung des Stammkapitals erforderliche
  Vermögen darf nicht ausgezahlt werden ([§ 30 Abs. 1 GmbHG][gmbhg30]).
  Prüfhilfe: Bilanz zum letzten Stichtag und aktuell mit `auswertung`
  (`art: bilanz`). Bleibt das Eigenkapital nach der Auszahlung unter dem
  Stammkapital, halte an und empfiehl rechtliche Beratung. Verbotene Zahlungen
  sind zu erstatten ([§ 31 GmbHG][gmbhg31]).
- **UG:** Ein Viertel des um einen Verlustvortrag geminderten
  Jahresüberschusses gehört in die gesetzliche Rücklage und ist nicht
  ausschüttbar ([§ 5a Abs. 3 GmbHG][gmbhg5a]). Das bucht
  buchfink-jahresabschluss.
- **Zahlungsfähigkeit:** Zahlungen an Gesellschafter, die zur
  Zahlungsunfähigkeit führen mussten, machen den Geschäftsführer
  ersatzpflichtig ([§ 15b Abs. 5 InsO][inso15b]). Prüfe mit `offene_posten`
  und den Bankständen, ob nach der Auszahlung alle fälligen Verbindlichkeiten
  bezahlt werden können.

## 3. Steuerabzug

| Größe | Wert | Grundlage |
|---|---|---|
| Kapitalertragsteuer | 25 % des Kapitalertrags | [§ 43a Abs. 1 Satz 1 Nr. 1 EStG][estg43a] |
| Bemessung | volle Ausschüttung ohne Abzug | [§ 43a Abs. 2 Satz 1 EStG][estg43a] |
| Solidaritätszuschlag | 5,5 % der Kapitalertragsteuer | [§ 3 Abs. 1 Nr. 5][solzg3], [§ 4 Satz 1 SolZG][solzg4] |
| Kirchensteuer | Satz nach Landesrecht; mindert die Kapitalertragsteuer um 25 % der Kirchensteuer | [§ 43a Abs. 1 Satz 2 EStG][estg43a], [§ 51a Abs. 2c EStG][estg51a] |

Beispiel ohne Kirchensteuer: Ausschüttung 10.000,00 € → Kapitalertragsteuer
2.500,00 €, Solidaritätszuschlag 137,50 €, Auszahlung 7.362,50 €.

- Der Abzug gilt auch, wenn der Gesellschafter eine Kapitalgesellschaft ist
  ([§ 43 Abs. 1 Satz 3 EStG][estg43]). Sie rechnet die Steuer später an.
- Die GmbH behält für Rechnung des Gesellschafters ein
  ([§ 44 Abs. 1 Satz 3 EStG][estg44]) und haftet für die Steuer
  ([§ 44 Abs. 5 Satz 1 EStG][estg44]); der Geschäftsführer haftet bei grober
  Fahrlässigkeit ([§ 69 AO][ao69]).
- Kirchensteuer: Die GmbH muss für Gesellschafter, die natürliche Personen
  sind, beim BZSt abfragen, ob sie kirchensteuerpflichtig sind: bei Begründung
  der Beteiligung (Anlassabfrage) und jährlich vom 1. September bis
  31. Oktober zum Stichtag 31. August (Regelabfrage)
  ([§ 51a Abs. 2c Satz 1 Nr. 3 EStG][estg51a]). Weise den Menschen im
  September darauf hin. Ob eine GmbH ohne geplante Ausschüttung abfragen muss,
  klärt er mit dem Steuerberater oder anhand der Hinweise des BZSt.
- Freistellung: Mit einer Nichtveranlagungs-Bescheinigung des Gesellschafters
  unterbleibt der Abzug ([§ 44a Abs. 1 Satz 4, Abs. 2 Satz 1 Nr. 2 EStG][estg44a]).
  Ein Freistellungsauftrag reicht bei Gewinnanteilen aus GmbH-Anteilen nicht;
  § 44a Abs. 1 Satz 1 EStG erfasst bei Gewinnanteilen nur Genussrechte und
  Anteile von Arbeitnehmern ([§ 44a Abs. 1 Satz 1 Nr. 1, 2 EStG][estg44a]).
- Ausländische Gesellschafter, EU-Muttergesellschaften: Grenze,
  Steuerberater.

## 4. Zufluss, Anmeldung, Abführung

- **Zufluss** am Tag, den der Beschluss als Auszahlungstag bestimmt; fehlt er,
  am Tag nach der Beschlussfassung ([§ 44 Abs. 2 Satz 1, 2 EStG][estg44]).
- **Abführung** bei Gewinnausschüttungen im Zeitpunkt des Zuflusses
  ([§ 44 Abs. 1 Satz 5 Halbsatz 2 EStG][estg44]). Die Frist „bis zum 10. des
  Folgemonats“ aus § 44 Abs. 1 Satz 5 Halbsatz 1 gilt für andere
  Kapitalerträge.
- **Anmeldung** innerhalb derselben Frist, elektronisch nach amtlichem
  Datensatz ([§ 45a Abs. 1 Satz 1 EStG][estg45a]). In Mein ELSTER heißt das
  Formular „Kapitalertragsteuer-Anmeldung nach Einkommensteuergesetz“
  ([ELSTER-Formularliste][elster]). `fristen` führt sie als Pflicht
  `kapitalertragsteuer` mit dem Zuflusstag als Fälligkeit.
  Unterbleibt der Abzug ganz oder teilweise, ist trotzdem anzumelden und der
  Grund anzugeben ([§ 45a Abs. 1 Satz 2, 3 EStG][estg45a]).
- **Rundung:** Die zu einem Zeitpunkt abzuführende Kapitalertragsteuer wird auf
  volle Euro abgerundet ([§ 44 Abs. 1 Satz 6 EStG][estg44]). Buchfink bucht auf
  den Cent; lege eine Differenz dem Menschen vor.
- **Säumnis:** 1 % des abgerundeten Rückstands je angefangenem Monat; bis drei
  Tage Verspätung wird nicht erhoben ([§ 240 Abs. 1, 3 AO][ao240]).

Bereite für den Menschen vor: Beschlussdatum, Zuflusstag, je Gesellschafter
Bruttobetrag, Kapitalertragsteuer, Solidaritätszuschlag, Kirchensteuer, Summen
für die Anmeldung, Zahlungsempfänger Finanzamt.

## 5. Buchungen in Buchfink

| Schritt | Buchung (SKR04) | Wer |
|---|---|---|
| Ergebnisverwendung beschlossen | 2970 Gewinnvortrag vor Verwendung an 3519 Verbindlichkeiten gegenüber Gesellschaftern für offene Ausschüttungen (netto) und 3760 Verbindlichkeiten aus Einbehaltungen (KapESt, SolZ) | du nach dem Ja mit `ergebnisverwendung_buchen` (buchfink-jahresabschluss) |
| Auszahlung an Gesellschafter | 3519 an Bank | buchfink-bank |
| Zahlung an das Finanzamt | 3760 an Bank | buchfink-bank |

Fertig, wenn 3519 und 3760 für diese Ausschüttung auf null stehen
(`auswertung`, `art: kontoblatt`) und die Anmeldung mit Übermittlungsprotokoll als
Beleg abgelegt ist.

## 6. Was Buchfink nicht rechnet

- Kirchensteuer und die Minderung der Kapitalertragsteuer bei
  Kirchensteuerpflicht.
- Abstandnahme bei Nichtveranlagungs-Bescheinigung.
- **Einlagenrückgewähr:** Übersteigt eine Leistung den ausschüttbaren Gewinn,
  mindert sie das steuerliche Einlagekonto ([§ 27 Abs. 1 Satz 3, 5 KStG][kstg27]).
  Insoweit ist sie beim Gesellschafter keine Einnahme
  ([§ 20 Abs. 1 Nr. 1 Satz 3 EStG][estg20]) und braucht eine Bescheinigung nach
  [§ 27 Abs. 3 KStG][kstg27].

In diesen Fällen stimmen die Beträge auf 3519 und 3760 nicht. Lege die
Abweichung vor; die Korrektur bucht nach Klärung mit dem Steuerberater
`buchung_erfassen` (Rückfrage) mit dem Beschluss als Beleg.

## 7. Vorabausschüttung

Eine Ausschüttung vor Feststellung des Abschlusses regelt das GmbHG nicht
ausdrücklich; sie braucht einen Beschluss der Gesellschafter und unterliegt
§ 30 GmbHG und dem Kapitalertragsteuerabzug wie jede Ausschüttung. Buchfink hat
dafür keinen eigenen Ablauf (der SKR04 führt 7790 Vorabausschüttung). Grenze:
Steuerberater.

[gmbhg5a]: https://www.gesetze-im-internet.de/gmbhg/__5a.html
[gmbhg29]: https://www.gesetze-im-internet.de/gmbhg/__29.html
[gmbhg30]: https://www.gesetze-im-internet.de/gmbhg/__30.html
[gmbhg31]: https://www.gesetze-im-internet.de/gmbhg/__31.html
[gmbhg46]: https://www.gesetze-im-internet.de/gmbhg/__46.html
[gmbhg48]: https://www.gesetze-im-internet.de/gmbhg/__48.html
[estg20]: https://www.gesetze-im-internet.de/estg/__20.html
[estg43]: https://www.gesetze-im-internet.de/estg/__43.html
[estg43a]: https://www.gesetze-im-internet.de/estg/__43a.html
[estg44]: https://www.gesetze-im-internet.de/estg/__44.html
[estg44a]: https://www.gesetze-im-internet.de/estg/__44a.html
[estg45a]: https://www.gesetze-im-internet.de/estg/__45a.html
[estg51a]: https://www.gesetze-im-internet.de/estg/__51a.html
[solzg3]: https://www.gesetze-im-internet.de/solzg_1995/__3.html
[solzg4]: https://www.gesetze-im-internet.de/solzg_1995/__4.html
[kstg27]: https://www.gesetze-im-internet.de/kstg_1977/__27.html
[inso15b]: https://www.gesetze-im-internet.de/inso/__15b.html
[ao69]: https://www.gesetze-im-internet.de/ao_1977/__69.html
[ao240]: https://www.gesetze-im-internet.de/ao_1977/__240.html
[elster]: https://www.elster.de/eportal/formulare-leistungen/alleformulare
