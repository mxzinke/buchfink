# Buchungsbeispiele

[Fachkonzepte](README.md) · [Belege und Buchungen](beleg-buchungsflow.md) ·
[Zahlungswege](zahlungswege.md)

Die Beispiele zeigen, welche Buchungssätze Buchfink aus einem Vorgang erzeugt.
Die Tests dazu stehen vor allem in `internal/service/posting_service_test.go`,
`payment_service_test.go`, `payment_ways_test.go` und `bank_split_test.go`.

## Lieferantenrechnung auf Ziel

Dienstleistung 1.000 € netto, 19 % USt.

| Schritt | Buchung |
|---|---|
| Beleg buchen | **5906** Fremdleistungen 1.000,00 + **1406** Vorsteuer 190,00 an Kreditor 1.190,00 |
| Zahlung | Kreditor 1.190,00 an **1800** Bank 1.190,00 |

## Ausgangsrechnung auf Ziel

Beratung 2.000 € netto, 19 % USt.

| Schritt | Buchung |
|---|---|
| Rechnung ausstellen | Debitor 2.380,00 an **4400** Erlöse 19 % 2.000,00 + **3806** Umsatzsteuer 380,00 |
| Zahlungseingang | **1800** Bank 2.380,00 an Debitor 2.380,00 |

## Cloud-Leistung aus der EU, § 13b Abs. 1 UStG

Rechnung eines irischen Anbieters, 1.000 € netto.

| Buchung |
|---|
| **5909** Fremdleistungen § 13b 1.000,00 + **1407** Vorsteuer § 13b 190,00 an **3837** Umsatzsteuer § 13b 190,00 + Kreditor 1.000,00 |

An den Anbieter gehen 1.000 €. Die Steuer schuldet die GmbH und zieht sie
zugleich als Vorsteuer ab. Im Ergebnis neutral, in der Voranmeldung stehen
Kz 46, 47 und 67.

## Innergemeinschaftlicher Erwerb

Laptop von einem Händler in den Niederlanden, 2.000 € netto.

| Buchung |
|---|
| Aufwands- oder Anlagekonto 2.000,00 + **1404** Vorsteuer i. g. Erwerb 380,00 an **3804** Umsatzsteuer i. g. Erwerb 380,00 + Kreditor 2.000,00 |

## Beleg mit zwei Steuersätzen

Hotel: Übernachtung 200 € netto zu 7 %, Frühstück 50 € netto zu 19 %.

| Buchung |
|---|
| **6650** Reisekosten 200,00 + **6650** Reisekosten 50,00 + **1401** Vorsteuer 7 % 14,00 + **1406** Vorsteuer 19 % 9,50 an Kreditor 273,50 |

## Firmenkreditkarte

Büromaterial 50 € netto, mit der Firmenkarte bezahlt.

| Schritt | Buchung |
|---|---|
| Beleg buchen | **6815** Bürobedarf 50,00 + **1406** Vorsteuer 9,50 an **3610** Kreditkartenabrechnung 59,50 |
| Abbuchung der Monatsabrechnung | **3610** 59,50 an **1800** Bank 59,50 |

## Auslage eines Gesellschafters

Der Gesellschafter bezahlt Software für 100 € netto privat. Aussteller des
Belegs bleibt der Lieferant.

| Schritt | Buchung |
|---|---|
| Beleg buchen | Aufwand 100,00 + **1406** Vorsteuer 19,00 an **3640** Verbindlichkeiten gegenüber Gesellschaftern 119,00 |
| Erstattung | **3640** 119,00 an **1800** Bank 119,00 |

## Zahlung mit Skonto

Lieferantenrechnung 1.190,00 € brutto, 2 % Skonto.

| Buchung |
|---|
| Kreditor 1.190,00 an **5736** Erhaltene Skonti 20,00 + **1406** Vorsteuer 3,80 + **1800** Bank 1.166,20 |

Die Vorsteuer sinkt um 3,80 € auf 186,20 € (§ 17 UStG).

## Zinsgutschrift mit Kapitalertragsteuer

Die Bank schreibt 73,63 € gut: 100,00 € Zinsen abzüglich 25,00 €
Kapitalertragsteuer und 1,37 € Solidaritätszuschlag. Ein Umsatz, drei Zeilen.

| Buchung |
|---|
| **1800** Bank 73,63 + **7630** Kapitalertragsteuer 25,00 + **7633** Solidaritätszuschlag 1,37 an **7110** Sonstige Zinserträge 100,00 |

## Gesellschafterdarlehen

Der Gesellschafter leiht der GmbH 20.000 € für drei Jahre zu 4 %.

| Schritt | Buchung |
|---|---|
| Auszahlung | **1800** Bank 20.000,00 an **3642** Verbindlichkeiten gegenüber Gesellschaftern, Restlaufzeit 1 bis 5 Jahre 20.000,00 |
| Zinsen eines Jahres, überwiesen | **7316** Zinsen für Gesellschafterdarlehen 800,00 an **1800** Bank 800,00 |

Das Konto folgt aus Richtung und Laufzeit, das Zinskonto aus der Beteiligung:
über 25 % am Stammkapital auf **7317**.

## Lohnjournal

Summen eines Monats aus dem Lohnbüro: Gehälter 4.000 €, Arbeitgeberanteil 840 €,
Arbeitnehmeranteil 800 €, Lohnsteuer 500 €, Nettolöhne 2.700 €.

| Buchung |
|---|
| **6020** Gehälter 4.000,00 + **6110** Gesetzliche soziale Aufwendungen 840,00 an **3720** Nettolöhne 2.700,00 + **3730** Lohnsteuer 500,00 + **3740** Sozialversicherung 1.640,00 |

## Storno per Generalumkehr

Die Lieferantenrechnung oben wurde doppelt gebucht.

| Buchung |
|---|
| **5906** −1.000,00 + **1406** −190,00 an Kreditor −1.190,00 |

Gleiche Konten, gleiche Seiten, negative Beträge. Die Verkehrszahlen von 5906
stehen wieder auf null.

## Rückstellung zum Bilanzstichtag

| Buchung |
|---|
| **6790** Aufwand für Gewährleistung 5.000,00 an **3090** Rückstellungen für Gewährleistungen 5.000,00 |

Eine Abschlussbuchung zum Bilanzstichtag; sie gehört zur Abschlussperiode.
