# Buchungsbeispiele

[Fachkonzepte](README.md) · [Beleg- und Buchungsflow](beleg-buchungsflow.md)

## 10. Durchgespielte Geschäftsvorfälle

Alle Buchungssätze sind als Test hinterlegt
(`internal/service/posting_service_test.go`, `payment_service_test.go`).

### 10.1 Lieferantenrechnung auf Ziel, Inland

Dienstleistung 1.000 € netto, 19 % USt.

| Schritt | Buchung |
|---|---|
| Beleg erfassen | SOLL **5906** Fremdleistungen 1.000,00 + SOLL **1406** Vorsteuer 190,00 · HABEN **Kreditorenkonto** 1.190,00 |
| Zahlung | SOLL **Kreditorenkonto** 1.190,00 · HABEN **1800** Bank 1.190,00 |

### 10.2 Ausgangsrechnung auf Ziel, Inland

Beratungsleistung 2.000 € netto, 19 % USt.

| Schritt | Buchung |
|---|---|
| Rechnung ausstellen | SOLL **Debitorenkonto** 2.380,00 · HABEN **4400** Erlöse 19 % 2.000,00 + HABEN **3806** Umsatzsteuer 380,00 |
| Zahlungseingang | SOLL **1800** Bank 2.380,00 · HABEN **Debitorenkonto** 2.380,00 |

### 10.3 Reverse Charge, § 13b UStG

Cloud-Leistung eines irischen Anbieters, 1.000 € netto.

| Schritt | Buchung |
|---|---|
| Beleg erfassen | SOLL **5909** Fremdleistungen (§ 13b) 1.000,00 + SOLL **1407** Vorsteuer § 13b 190,00 · HABEN **3837** Umsatzsteuer § 13b 190,00 + HABEN **Kreditorenkonto** 1.000,00 |

An den Lieferanten gehen nur 1.000 €. Die Steuer wird geschuldet und zugleich als
Vorsteuer abgezogen; ergebniswirksam ist der Vorgang neutral, für die Voranmeldung
aber nicht.

### 10.4 Innergemeinschaftlicher Erwerb

Warenkauf aus den Niederlanden, 2.000 € netto.

| Schritt | Buchung |
|---|---|
| Beleg erfassen | SOLL **5400** Wareneingang 19 % VSt 2.000,00 + SOLL **1404** Vorsteuer i.g. Erwerb 380,00 · HABEN **3804** USt i.g. Erwerb 380,00 + HABEN **Kreditorenkonto** 2.000,00 |

### 10.5 Beleg mit zwei Steuersätzen

Hotelrechnung: Übernachtung 200 € netto zu 7 %, Frühstück 50 € netto zu 19 %.

| Buchung |
|---|
| SOLL **6650** Reisekosten 200,00 + SOLL **6650** Reisekosten 50,00 + SOLL **1401** Vorsteuer 7 % 14,00 + SOLL **1406** Vorsteuer 19 % 9,50 · HABEN **Kreditorenkonto** 273,50 |

### 10.6 Barzahlung

Quittung Büromaterial, 50 € netto, sofort bar bezahlt.

| Buchung |
|---|
| SOLL **6815** Bürobedarf 50,00 + SOLL **1406** Vorsteuer 9,50 · HABEN **1600** Kasse 59,50 |

Bar heißt **Kasse**. Wer stattdessen 1800 Bank bucht, hat einen Kassenbestand, den es
nicht gibt, und einen Bankbestand, der nicht stimmt.

### 10.7 Zahlung mit Skonto

Rechnung über 1.190,00 € brutto, 2 % Skonto bei Zahlung binnen 10 Tagen.

| Buchung |
|---|
| SOLL **Kreditorenkonto** 1.190,00 · HABEN **5736** Erhaltene Skonti 20,00 + HABEN **1406** Vorsteuer 3,80 + HABEN **1800** Bank 1.166,20 |

Die Vorsteuer steht danach bei 186,20 € statt 190,00 € – das ist die Korrektur nach
§ 17 UStG.

### 10.8 Zahlung mit Bankgebühr

Auslandsüberweisung, die Bank bucht 5,00 € Entgelt zusätzlich ab.

| Buchung |
|---|
| SOLL **Kreditorenkonto** 1.190,00 + SOLL **6855** Nebenkosten des Geldverkehrs 5,00 · HABEN **1800** Bank 1.195,00 |

### 10.9 Storno per Generalumkehr

Beleg aus 10.1 doppelt erfasst.

| Buchung |
|---|
| SOLL **5906** −1.000,00 + SOLL **1406** −190,00 · HABEN **Kreditorenkonto** −1.190,00 |

Gleiche Konten, gleiche Seiten, negierte Beträge. Die Verkehrszahlen von 5906 stehen
danach wieder auf null.

### 10.10 Rückstellung zum Bilanzstichtag

Gewährleistungsrückstellung 5.000 €, § 249 HGB.

| Buchung |
|---|
| SOLL **6790** Aufwand für Gewährleistung 5.000,00 · HABEN **3090** Rückstellungen für Gewährleistungen 5.000,00 |

Kein Bankumsatz, keine Zahlungszuordnung. Abschlussbuchung, üblicherweise im Q1 des
Folgejahres gebucht.

### 10.11 Darlehen

| Schritt | Buchung |
|---|---|
| Auszahlung 50.000 € | SOLL **1800** Bank 50.000,00 · HABEN **3160** Verb. ggü. Kreditinstituten (1–5 J.) 50.000,00 |
| Rate 1.000 € (800 Tilgung, 200 Zins) | SOLL **3160** 800,00 + SOLL **7320** Zinsaufwendungen langfristig 200,00 · HABEN **1800** Bank 1.000,00 |

### 10.12 Zusammenfassung der Nutzer-Aktionen

Alle Vorfälle laufen über drei Aktionen:

1. **Etwas erfassen** – Beleg, Rechnung, Rückstellung, Anlage → sofortige Buchung
2. **Zahlung zuordnen** – Ausgleich offener Posten inklusive Differenzen
3. **Freigeben** – für periodische Vorfälle wie AfA und Abschlussbuchungen

Der Nutzer wählt nie direkt eine SOLL/HABEN-Kombination. Die Kontenwahl ergibt sich
deterministisch aus fachlicher Gruppe, Steuerfall, Steuersatz und Zahlungsweg.
