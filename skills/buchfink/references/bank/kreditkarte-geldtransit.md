# Kreditkarte und Geldtransit

Rechtsstand: 17. Oktober 2026. Wortlaut von UStG und UStDV am 17.10.2026
gelesen. Werkzeugstand: Buchfink 0.3.0.

## Inhalt

1. Wie Buchfink die Firmenkreditkarte führt
2. Kartenbeleg
3. Monatsabrechnung ausgleichen
4. Differenzen zur Abrechnung
5. Abrechnung ohne Bankumsatz
6. Debitkarte
7. Übertrag zwischen eigenen Konten

## 1. Wie Buchfink die Firmenkreditkarte führt

Jede Kartenzahlung ist ein Beleg mit `bezahlung: kreditkarte`, gebucht gegen
3610 Kreditkartenabrechnung statt gegen den Lieferanten. Die Lastschrift der
Monatsabrechnung auf dem Bankkonto gleicht 3610 aus. Lege die Monatsabrechnung
als Beleg ab; sie belegt Gebühren, die nur dort stehen.

## 2. Kartenbeleg

- Kartenbeleg und Abrechnungszeile weisen nur die Zahlung nach; Vorsteuer
  gibt nur die Rechnung des Händlers
  ([§ 15 Abs. 1 Satz 1 Nr. 1 Satz 2 UStG][ustg15]), bis 250 € auch ein
  Kassenbon nach [§ 33 UStDV][ustdv33]. Buche sie nach
  [belege/ablauf.md](../belege/ablauf.md) mit `bezahlung: kreditkarte`.
- Fehlt sie, frag danach (Kundenkonto des Anbieters, E-Mail); sonst erfasst
  der Mensch einen Eigenbeleg ohne Vorsteuer
  ([grundlagen/gobd.md](../grundlagen/gobd.md#8-belegwesen-und-eigenbeleg)).
- Fremdwährung: Rechnung nach
  [belege/fremdwaehrung-auslagen.md](../belege/fremdwaehrung-auslagen.md#1-beleg-in-fremdwährung);
  den abweichenden Euro-Betrag der Abrechnung klärst du beim Ausgleich
  (Abschnitt 4).

## 3. Monatsabrechnung ausgleichen

1. Lies Abrechnungsstichtag und Summe.
2. Vergleiche jede Zeile mit den Buchungen auf 3610 bis zum Stichtag
   (`auswertung`, `art: kontoblatt`, `konto: 3610`, `bis: <Stichtag>`).
3. Fehlt eine Buchung, buche erst den Beleg (Abschnitt 2); fehlt der Beleg,
   frag danach.
4. Stimmt der Saldo von 3610 zum Stichtag mit der Lastschrift, leg den
   Ausgleich vor und buche nach dem Ja mit `bankumsatz_buchen` und
   `kreditkartenabrechnung`. Sonst kläre erst die Differenz (Abschnitt 4).

Fertig, wenn 3610 nach der Lastschrift nur noch Zahlungen nach dem Stichtag
enthält.

## 4. Differenzen zur Abrechnung

| Ursache | Vorgehen |
|---|---|
| Jahresgebühr, Auslandseinsatzentgelt, Zinsen der Kartenbank | `buchung_erfassen` (Rückfrage) gegen 3610, Beleg ist die Abrechnung; Gebühren 6855, Zinsen 7310 |
| Kartenzahlung in Fremdwährung, Euro-Betrag weicht ab | als Kursdifferenz vorlegen ([skonto-differenzen.md](skonto-differenzen.md#4-kursdifferenz)) |
| Erstattung eines Händlers auf die Karte | Gutschrift als Beleg mit Bezahlung `kreditkarte` |
| Zahlung fällt in die nächste Abrechnung | keine Differenz; Stichtag beachten |
| Zahlung ohne Beleg und erkennbaren Zweck | Mensch fragen; private Nutzung nach [gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md) |

Entgelte im Zahlungsverkehr sind steuerfrei ([§ 4 Nr. 8 Buchst. d UStG][ustg4]);
weist die Abrechnung Umsatzsteuer aus, übernimm sie.

## 5. Abrechnung ohne Bankumsatz

Zahlt ein Konto, das Buchfink nicht führt, oder der Gesellschafter die
Abrechnung, buchst du den Ausgleich von 3610 mit `buchung_erfassen`
(Rückfrage); Zahlungen des Gesellschafters über das
[Verrechnungskonto](../gesellschafter/verrechnungskonto.md).

## 6. Debitkarte

Eine Debitkarte (Girocard, Debit-Visa) belastet das Bankkonto je Zahlung: Beleg
mit Bezahlung `offen` buchen, Bankumsatz zuordnen; 3610 bleibt unberührt. Frag,
welche Karte welche ist.

## 7. Übertrag zwischen eigenen Konten

- Ein Übertrag (Bank an Tagesgeld, Bank an Bank) ist erfolgsneutral: beide
  Seiten gegen 1460 Geldtransit.
- 1460 steht am Monatsende auf null. Ein Rest ist Geld unterwegs (Abgang am
  31., Zugang am 1.) oder ein Fehler; kläre ihn beim Abstimmen
  ([ablauf.md](ablauf.md#1-ablauf), Schritt 6).

[ustg4]: https://www.gesetze-im-internet.de/ustg_1980/__4.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
[ustdv33]: https://www.gesetze-im-internet.de/ustdv_1980/__33.html
