# Forderungsausfall im laufenden Jahr

Rechtsstand: 17. Oktober 2026.

Wertberichtigung zum Stichtag:
[jahresabschluss/forderungen.md](../jahresabschluss/forderungen.md).

## Inhalt

1. Zweifelhaft oder uneinbringlich
2. Was die Ausbuchung bewirkt
3. Ablauf
4. Teilausfall und Vergleich
5. Geld kommt doch noch

## 1. Zweifelhaft oder uneinbringlich

- **Uneinbringlich:** Kunde zahlungsunfähig, Einrede des
  Einforderungsverzichts, oder die Forderung ist auf absehbare Zeit nicht
  durchsetzbar; auch substantiiertes Bestreiten ([UStAE 17.1 Abs. 5][ustae171]).
  Typische Belege: Insolvenzeröffnung, Abweisung mangels Masse, fruchtlose
  Vollstreckung, Vergleich mit Verzicht.
- **Zweifelhaft:** Ausfall möglich, aber offen (schleppende Zahlung,
  Mahnverfahren). Bleibt im Jahr stehen; zum Stichtag
  [jahresabschluss/forderungen.md](../jahresabschluss/forderungen.md).
- Wertungsfrage: die Einordnung. Der Mensch entscheidet und legt den Beleg ab.

## 2. Was die Ausbuchung bewirkt

- Netto wird Aufwand: SKR04 6936 Forderungsverluste 19 % USt, 6931 bei 7 %,
  6930 ohne Steuerausweis.
- Die Umsatzsteuer wird berichtigt ([§ 17 Abs. 2 Nr. 1 Satz 1 i. V. m.
  Abs. 1 UStG][ustg17]), im Voranmeldungszeitraum des Ausfalls
  ([§ 17 Abs. 1 Satz 8 UStG][ustg17]).
- Steuerfreie und nicht steuerbare Umsätze: nichts zu berichtigen, der ganze
  Betrag ist Aufwand.

## 3. Ablauf

1. Posten finden: `offene_posten` (`richtung: forderung`, `kontakt_id`).
   Fertig, wenn Nummer, offener Betrag, Steuersatz und Steuerfall feststehen
   (`buchungen_suchen` mit `beleg_id` oder Text).
2. Beleg der Uneinbringlichkeit mit `beleg_ablegen` ablegen.
3. Vorlage an den Menschen: Posten, netto und Steuer, Grund, Beleg, Zeitraum der
   Steuerberichtigung, Wertungsfrage nach Abschnitt 1.
4. Sag ihm, dass Buchfink die Ausbuchung noch nicht abbildet. Zum Stichtag
   lässt sich eine Einzelwertberichtigung buchen
   ([jahresabschluss/forderungen.md](../jahresabschluss/forderungen.md)); die
   Steuerberichtigung nach § 17 fehlt dann weiter.
5. Bei Insolvenz: Forderung beim Insolvenzverwalter anmelden (außerhalb von
   Buchfink).

## 4. Teilausfall und Vergleich

- Teilzahlung und Rest uneinbringlich: nur der Rest wird ausgebucht, die Steuer
  anteilig berichtigt ([§ 17 Abs. 2 Nr. 1 UStG][ustg17]). Die Teilzahlung
  ordnest du nach [bank/zuordnung.md](../bank/zuordnung.md) zu.
- Nachlass wegen Mängeln mindert das Entgelt ([§ 17 Abs. 1 UStG][ustg17]):
  Differenz am Zahlungseingang
  ([bank/skonto-differenzen.md](../bank/skonto-differenzen.md)) oder
  Rechnungskorrektur ([storno-berichtigung.md](storno-berichtigung.md)).

## 5. Geld kommt doch noch

- Steuer erneut berichtigen ([§ 17 Abs. 2 Nr. 1 Satz 2 UStG][ustg17]).
- SKR04 4925 Erträge aus abgeschriebenen Forderungen, mit Umsatzsteuer zum
  ursprünglichen Satz. Leg die Aufteilung vor und buche sie am Bankumsatz mit
  `bankumsatz_buchen` ([bank/zuordnung.md](../bank/zuordnung.md)).

[ustg17]: https://www.gesetze-im-internet.de/ustg_1980/__17.html
[ustae171]: https://www.steuerschroeder.de/steuergesetze/ustae/17.1
