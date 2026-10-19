# Kreditkarte und Geldtransit

Rechtsstand: 17. Oktober 2026. Wortlaut von UStG und UStDV am 17.10.2026
gelesen. Werkzeugstand: Buchfink 0.3.0.

Den Kartenbeleg und die Monatsabrechnung der Firmenkreditkarte regelt diese
Datei gemeinsam; Belege in Fremdwährung stehen in
[belege/fremdwaehrung-auslagen.md](../belege/fremdwaehrung-auslagen.md).

## Inhalt

1. Wie Buchfink die Firmenkreditkarte führt
2. Kartenbeleg
3. Monatsabrechnung ausgleichen
4. Differenzen zur Abrechnung
5. Abrechnung ohne Bankumsatz
6. Debitkarte
7. Übertrag zwischen eigenen Konten

## 1. Wie Buchfink die Firmenkreditkarte führt

- Jede Kartenzahlung ist ein Beleg mit `bezahlung: kreditkarte`. Buchfink
  bucht ihn gegen 3610 Kreditkartenabrechnung statt gegen den Lieferanten.
- Die Monatsabrechnung der Kartenbank listet die Zahlungen und belastet den
  Saldo per Lastschrift auf dem Bankkonto. Diese Lastschrift gleicht 3610 aus.
- Lege die Monatsabrechnung als Beleg ab. Sie ist Beleg für Gebühren, die nur
  dort stehen.

## 2. Kartenbeleg

**Was ein Beleg ist.** Der Kartenbeleg (Slip) und die Zeile der Monatsabrechnung
weisen nur die Zahlung nach. Für die Vorsteuer braucht die GmbH die Rechnung des
Händlers ([§ 15 Abs. 1 Satz 1 Nr. 1 Satz 2 UStG][ustg15]). Bis 250 € genügt
eine Kleinbetragsrechnung, etwa ein Kassenbon mit den Angaben nach
[§ 33 UStDV][ustdv33].

**Buchen.** Lege die Rechnung ab und buche sie nach
[belege/ablauf.md](../belege/ablauf.md) mit `bezahlung: kreditkarte`.
Kreditkarte ist mit `beleg_buchen` als Routine möglich, wenn alle Bedingungen
für „direkt“ erfüllt sind
([grundlagen/arbeitsweise.md](../grundlagen/arbeitsweise.md#4-wann-der-server-direkt-zulässt)).

**Ohne Rechnung.** Frag den Menschen nach der Rechnung (Kundenkonto des
Anbieters, E-Mail). Gibt es keine, erfasst er einen Eigenbeleg mit Grund
([grundlagen/gobd.md](../grundlagen/gobd.md#8-belegwesen-und-eigenbeleg));
Vorsteuer entfällt.

**Fremdwährung auf der Karte.** Die Rechnung in Fremdwährung buchst du nach
[belege/fremdwaehrung-auslagen.md](../belege/fremdwaehrung-auslagen.md#1-beleg-in-fremdwährung).
Der Betrag, den die Kartenabrechnung in Euro belastet, weicht vom Kurs ab; die
Differenz klärst du beim Ausgleich der Abrechnung (Abschnitt 4).

## 3. Monatsabrechnung ausgleichen

1. Lies den Abrechnungsstichtag und die Summe der Abrechnung.
2. Vergleiche jede Zeile der Abrechnung mit den Buchungen auf 3610 bis zum
   Stichtag (`auswertung` mit `art: kontoblatt`, `konto: 3610`,
   `bis: <Stichtag>`).
3. Fehlt zu einer Zeile die Buchung, buche erst den Beleg (Abschnitt 2).
   Fehlt der Beleg, frag danach.
4. Stimmt der Saldo von 3610 zum Stichtag mit der Lastschrift überein, lege den
   Ausgleich vor und buche ihn nach dem Ja mit `bankumsatz_buchen` und
   `kreditkartenabrechnung {stichtag}`. `bankumsaetze` zeigt den Fall als
   Vorschlag der Art `kreditkarte` mit Stichtag.
5. Stimmt er nicht, kläre die Differenz (Abschnitt 4) und buche erst die
   fehlenden Posten.

Fertig, wenn 3610 nach der Lastschrift nur noch Zahlungen nach dem Stichtag
enthält.

## 4. Differenzen zur Abrechnung

| Ursache | Vorgehen |
|---|---|
| Jahresgebühr, Auslandseinsatzentgelt, Zinsen der Kartenbank | Buchung mit `buchung_erfassen` (Rückfrage) gegen 3610, Beleg ist die abgelegte Abrechnung; Gebühren auf 6855 Nebenkosten des Geldverkehrs, Zinsen 7310 |
| Kartenzahlung in Fremdwährung, Euro-Betrag weicht vom gebuchten Betrag ab | Differenz als Kursdifferenz vorlegen ([skonto-differenzen.md](skonto-differenzen.md#4-kursdifferenz)) |
| Erstattung oder Gutschrift eines Händlers auf die Karte | Gutschrift als Beleg buchen ([belege/ablauf.md](../belege/ablauf.md)), Bezahlung `kreditkarte` |
| Zahlung liegt zeitlich in der nächsten Abrechnung | keine Differenz; Stichtag beachten |
| Zahlung ohne Beleg und ohne erkennbaren Zweck | Mensch fragen; private Nutzung prüfst du nach [gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md) |

Ob Entgelte der Kartenbank steuerfrei sind, ergibt sich aus der Abrechnung;
Entgelte im Zahlungsverkehr sind nach [§ 4 Nr. 8 Buchst. d UStG][ustg4]
steuerfrei. Weist die Abrechnung Umsatzsteuer aus, übernimm sie.

## 5. Abrechnung ohne Bankumsatz

Wird die Abrechnung von einem Konto bezahlt, das Buchfink nicht führt, oder
vom Gesellschafter, gibt es keinen Bankumsatz zum Zuordnen. Dann buchst du den
Ausgleich von 3610 mit `buchung_erfassen` (Rückfrage). Zahlungen des
Gesellschafters laufen über das
[Verrechnungskonto](../gesellschafter/verrechnungskonto.md).

## 6. Debitkarte

Eine Debitkarte (Girocard, Debit-Visa) belastet das Bankkonto direkt; jede
Zahlung ist ein eigener Bankumsatz. Buche den Beleg mit Bezahlung `offen` und
ordne den Bankumsatz zu. Das Konto 3610 bleibt dafür unberührt. Frag den
Menschen, welche Karte welche ist.

## 7. Übertrag zwischen eigenen Konten

- Ein Übertrag (Bank an Tagesgeld, Bank an Bank) ist kein Aufwand und kein
  Ertrag. Buche beide Seiten gegen 1460 Geldtransit: Abgang auf dem einen Konto
  an 1460, Zugang auf dem anderen von 1460.
- Ist das Zielkonto nicht in Buchfink eingerichtet, kann der Zugang nicht
  importiert werden; das Konto richtet der Mensch in der App ein.
- Wiederkehrende Überträge schlägt `bankumsaetze` als Vorschlag der Art
  `regel` vor; sie bleiben Rückfrage.
- 1460 steht am Monatsende auf null. Ein Rest ist Geld unterwegs (Abgang am
  31., Zugang am 1.) oder ein Fehler; kläre ihn beim Abstimmen
  ([ablauf.md](ablauf.md#1-ablauf), Schritt 7).

[ustg4]: https://www.gesetze-im-internet.de/ustg_1980/__4.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
[ustdv33]: https://www.gesetze-im-internet.de/ustdv_1980/__33.html
