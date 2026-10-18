# Kreditkartenabrechnung und Geldtransit

Rechtsstand: 17. Oktober 2026. Werkzeuge nach Buchfink 0.3.0. Den einzelnen
Kartenbeleg bucht buchfink-belege.

## Inhalt

1. Wie Buchfink die Firmenkreditkarte führt
2. Monatsabrechnung ausgleichen
3. Differenzen zur Abrechnung
4. Abrechnung ohne Bankumsatz
5. Debitkarte
6. Übertrag zwischen eigenen Konten

## 1. Wie Buchfink die Firmenkreditkarte führt

- Jede Kartenzahlung ist ein Beleg mit `bezahlung: kreditkarte`. Buchfink
  bucht ihn gegen 3610 Kreditkartenabrechnung.
- Die Monatsabrechnung der Kartenbank listet die Zahlungen und belastet den
  Saldo per Lastschrift auf dem Bankkonto. Diese Lastschrift gleicht 3610 aus.
- Lege die Monatsabrechnung als Beleg ab. Sie ist Beleg für Gebühren, die nur
  dort stehen.

## 2. Monatsabrechnung ausgleichen

1. Lies den Abrechnungsstichtag und die Summe der Abrechnung.
2. Vergleiche jede Zeile der Abrechnung mit den Buchungen auf 3610 bis zum
   Stichtag (`auswertung` mit `art: kontoblatt`, `konto: 3610`,
   `bis: <Stichtag>`).
3. Fehlt zu einer Zeile die Buchung, buche erst den Beleg (buchfink-belege).
   Fehlt der Beleg, frag danach.
4. Stimmt der Saldo von 3610 zum Stichtag mit der Lastschrift überein, lege den
   Ausgleich vor und buche ihn nach dem Ja mit `bankumsatz_buchen` und
   `kreditkartenabrechnung {stichtag}`. `bankumsaetze` zeigt den Fall als
   Vorschlag der Art `kreditkarte` mit Stichtag.
5. Stimmt er nicht, kläre die Differenz (Abschnitt 3) und buche erst die
   fehlenden Posten.

Fertig, wenn 3610 nach der Lastschrift nur noch Zahlungen nach dem Stichtag
enthält.

## 3. Differenzen zur Abrechnung

| Ursache | Vorgehen |
|---|---|
| Jahresgebühr, Auslandseinsatzentgelt, Zinsen der Kartenbank | Buchung mit `buchung_erfassen` (Rückfrage) gegen 3610, Beleg ist die abgelegte Abrechnung; Gebühren auf 6855 Nebenkosten des Geldverkehrs, Zinsen 7310 |
| Kartenzahlung in Fremdwährung, Euro-Betrag weicht vom gebuchten Betrag ab | Differenz als Kursdifferenz vorlegen; Grundlagen in buchfink-belege |
| Erstattung oder Gutschrift eines Händlers auf die Karte | Gutschrift als Beleg buchen (buchfink-belege), Bezahlung `kreditkarte` |
| Zahlung liegt zeitlich in der nächsten Abrechnung | keine Differenz; Stichtag beachten |
| Zahlung ohne Beleg und ohne erkennbaren Zweck | Mensch fragen; private Nutzung prüft buchfink-gesellschafter |

Ob Entgelte der Kartenbank steuerfrei sind, ergibt sich aus der Abrechnung;
Entgelte im Zahlungsverkehr sind nach [§ 4 Nr. 8 Buchst. d UStG][ustg4]
steuerfrei. Weist die Abrechnung Umsatzsteuer aus, übernimm sie.

## 4. Abrechnung ohne Bankumsatz

Wird die Abrechnung von einem Konto bezahlt, das Buchfink nicht führt, oder
vom Gesellschafter, gibt es keinen Bankumsatz zum Zuordnen. Dann buchst du den
Ausgleich von 3610 mit `buchung_erfassen` (Rückfrage). Zahlungen des
Gesellschafters regelt buchfink-gesellschafter.

## 5. Debitkarte

Eine Debitkarte (Girocard, Debit-Visa) belastet das Bankkonto direkt; jede
Zahlung ist ein eigener Bankumsatz. Buche den Beleg mit Bezahlung `offen` und
ordne den Bankumsatz zu. Das Konto 3610 bleibt dafür unberührt. Frag den
Menschen, welche Karte welche ist.

## 6. Übertrag zwischen eigenen Konten

- Ein Übertrag (Bank an Tagesgeld, Bank an Bank) ist kein Aufwand und kein
  Ertrag. Buche beide Seiten gegen 1460 Geldtransit: Abgang auf dem einen Konto
  an 1460, Zugang auf dem anderen von 1460.
- Ist das Zielkonto nicht in Buchfink eingerichtet, kann der Zugang nicht
  importiert werden; das Konto richtet der Mensch in der App ein.
- Wiederkehrende Überträge schlägt `bankumsaetze` als Vorschlag der Art
  `regel` vor; sie bleiben Rückfrage.
- 1460 steht am Monatsende auf null. Ein Rest ist Geld unterwegs (Abgang am
  31., Zugang am 1.) oder ein Fehler; kläre ihn in Schritt 7 des Ablaufs.

[ustg4]: https://www.gesetze-im-internet.de/ustg_1980/__4.html
