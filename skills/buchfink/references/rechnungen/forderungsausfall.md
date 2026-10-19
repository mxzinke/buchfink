# Forderungsausfall im laufenden Jahr

Rechtsstand: 17. Oktober 2026.

Die Bewertung zweifelhafter Forderungen zum Abschlussstichtag (Einzel- und
Pauschalwertberichtigung) regelt
[jahresabschluss/forderungen.md](../jahresabschluss/forderungen.md).

## Inhalt

1. Zweifelhaft oder uneinbringlich
2. Was die Ausbuchung bewirkt
3. Ablauf
4. Teilausfall und Vergleich
5. Geld kommt doch noch

## 1. Zweifelhaft oder uneinbringlich

- **Uneinbringlich** ist das Entgelt insbesondere, wenn der Kunde
  zahlungsunfähig ist, wenn ihm die Einrede des Einforderungsverzichts zusteht
  oder wenn bei objektiver Betrachtung die Forderung auf absehbare Zeit
  rechtlich oder tatsächlich nicht durchsetzbar ist. Bestreitet der Kunde die
  Forderung substantiiert, kommt eine Berichtigung ebenfalls in Betracht
  ([UStAE 17.1 Abs. 5][ustae171]).
- **Zweifelhaft** ist eine Forderung, deren Ausfall möglich, aber offen ist
  (Kunde zahlt schleppend, Mahnverfahren läuft). Sie bleibt im Jahr stehen; zum
  Stichtag prüfst du eine Wertberichtigung nach
  [jahresabschluss/forderungen.md](../jahresabschluss/forderungen.md).
- Wertungsfrage: die Einordnung. Typische Belege für Uneinbringlichkeit:
  Eröffnung des Insolvenzverfahrens über den Kunden, Abweisung mangels Masse,
  fruchtlose Vollstreckung, Vergleich mit Verzicht. Der Mensch entscheidet und
  legt den Beleg ab.

## 2. Was die Ausbuchung bewirkt

- Der Nettobetrag wird Aufwand: SKR04 6936 Forderungsverluste 19 % USt,
  6931 bei 7 %, 6930 ohne Steuerausweis (übliche Höhe).
- Die Umsatzsteuer wird berichtigt ([§ 17 Abs. 2 Nr. 1 Satz 1 i. V. m.
  Abs. 1 UStG][ustg17]), und zwar im Voranmeldungszeitraum, in dem die
  Uneinbringlichkeit eintritt ([§ 17 Abs. 1 Satz 8 UStG][ustg17]).
- Bei steuerfreien und nicht steuerbaren Umsätzen (z. B. EU-Kunde mit
  Steuerschuldnerschaft) gibt es keine deutsche Steuer zu berichtigen; der ganze
  Betrag ist Aufwand.
- Buchfink bildet die Ausbuchung noch nicht ab, weder als Werkzeug noch in der
  App.

## 3. Ablauf

1. Posten finden: `offene_posten` (`richtung: forderung`, `kontakt_id`).
   Fertig, wenn Nummer, offener Betrag, Steuersatz und Steuerfall feststehen
   (`buchungen_suchen` mit `beleg_id` oder Text).
2. Beleg für die Uneinbringlichkeit ablegen (`beleg_ablegen`): Bekanntmachung,
   Schreiben des Insolvenzverwalters, Vollstreckungsprotokoll, Vergleich.
3. Vorlage an den Menschen: Posten, Betrag netto und Steuer, Grund, Beleg,
   Zeitraum der Steuerberichtigung, Wertungsfrage nach Abschnitt 1.
4. Sag dem Menschen, dass Buchfink die Ausbuchung noch nicht bucht. Zum
   Stichtag lässt sich eine Einzelwertberichtigung buchen
   ([jahresabschluss/forderungen.md](../jahresabschluss/forderungen.md)); die
   Steuerberichtigung nach § 17 fehlt dann weiter.
5. Läuft ein Insolvenzverfahren, erinnere den Menschen, die Forderung beim
   Insolvenzverwalter anzumelden. Das geschieht außerhalb von Buchfink.

## 4. Teilausfall und Vergleich

- Zahlt der Kunde einen Teil und ist der Rest uneinbringlich, wird nur der Rest
  ausgebucht; die Steuer wird anteilig berichtigt
  ([§ 17 Abs. 2 Nr. 1 UStG][ustg17]).
- Kommt eine Teilzahlung über die Bank, ordnest du sie nach
  [bank/zuordnung.md](../bank/zuordnung.md) zu. Den Rest auszubuchen bildet
  Buchfink noch nicht ab (Abschnitt 3).
- Ein Nachlass wegen Mängeln mindert das Entgelt
  ([§ 17 Abs. 1 UStG][ustg17]). Weg: Differenz am Zahlungseingang
  ([bank/skonto-differenzen.md](../bank/skonto-differenzen.md)) oder
  Rechnungskorrektur ([storno-berichtigung.md](storno-berichtigung.md)).

## 5. Geld kommt doch noch

- Wird ein ausgebuchtes Entgelt später gezahlt, sind Steuer und Vorsteuer
  erneut zu berichtigen ([§ 17 Abs. 2 Nr. 1 Satz 2 UStG][ustg17]).
- Konto (SKR04): 4925 Erträge aus abgeschriebenen Forderungen, mit
  Umsatzsteuer zum ursprünglichen Satz. Lege die Aufteilung dem Menschen vor;
  am Bankumsatz buchst du sie nach [bank/zuordnung.md](../bank/zuordnung.md)
  mit `bankumsatz_buchen`.

[ustg17]: https://www.gesetze-im-internet.de/ustg_1980/__17.html
[ustae171]: https://www.steuerschroeder.de/steuergesetze/ustae/17.1
