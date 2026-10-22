# Bankumsätze zuordnen

Rechtsstand: 17. Oktober 2026. Werkzeugfelder nach Buchfink 0.3.0; die
Schemas des Servers gehen vor.

## Inhalt

1. Vorschläge lesen
2. Routine
3. Aufteilung schreiben
4. Teilzahlung, Sammelzahlung, Überzahlung
5. Zahlung ohne Rechnung
6. Rücklastschrift und Rücküberweisung
7. Umsatz ohne Beleg oder Verwendungszweck
8. Ignorieren
9. Offene Posten über den Jahreswechsel

## 1. Vorschläge lesen

Ein Vorschlag von `bankumsaetze` ist ein Hinweis. Stimmen Betrag, Kontakt und
Rechnungsnummer, ist er meist richtig; liegen zwei Vorschläge nahe beieinander,
entscheidet der Mensch.

## 2. Routine

Routine ist nur der Ausgleich genau eines offenen Postens mit
`bankumsatz_zuordnen`; was im Einzelfall fehlt, nennen die `gruende` der
Antwort. Vorschläge der Art `regel` (aus früheren Buchungen desselben Umsatzes)
und `kreditkarte` (aus dem Saldo von 3610) sind Rückfrage.

## 3. Aufteilung schreiben

Die Zeilen von `aufteilung` ergeben zusammen den Umsatz. Beim Ausgleich eines
Postens zieht Buchfink Skonto, Rundung und Kursdifferenz vom `ausgleich` ab;
eine Gebühr rechnet es beim Zahlungsausgang hinzu und beim Eingang ab. Die
Zeilen enthalten keine Umsatzsteuer; Vorgänge mit Steuer buchst du über ihren
Beleg ([belege/ablauf.md](../belege/ablauf.md)).

Skonto auf eine Eingangsrechnung über 1.190,00 €, gezahlt 1.166,20 €:

```
aufteilung: [
  {buchung_id: 812, ausgleich: "1190.00",
   differenz: {art: skonto, betrag: "23.80"}}
]
```

Buchfink bucht daraus Verbindlichkeit 1.190,00 an Bank 1.166,20, erhaltene
Skonti 20,00 und Vorsteuerkorrektur 3,80.

Schreib `text` je Zeile so, dass ein Prüfer ihn ohne Chat versteht („Zinsen Q3
2026 Tagesgeld, Abrechnung vom 30.09.2026“).

## 4. Teilzahlung, Sammelzahlung, Überzahlung

- **Teilzahlung:** Zeile mit `buchung_id` und dem gezahlten Betrag als
  `ausgleich`; der Rest bleibt offen. Frag, ob eine Ratenvereinbarung besteht
  oder der Rest strittig ist
  ([rechnungen/mahnen-verzug.md](../rechnungen/mahnen-verzug.md),
  [rechnungen/forderungsausfall.md](../rechnungen/forderungsausfall.md)).
- **Sammelzahlung:** je Rechnung eine Zeile mit `buchung_id`; fehlt die Liste
  im Verwendungszweck, frag nach dem Zahlungsavis.
- **Überzahlung, Doppelzahlung:** Ein Posten nimmt höchstens seinen offenen
  Betrag auf. Gleiche ihn aus und leg den Rest vor; der Mensch entscheidet:
  erstatten oder verrechnen. Danach bucht eine zweite Zeile den Rest auf das
  Personenkonto des Kunden (`konto` aus `kontakte_suchen`, `kontakt_id`), nie
  auf Ertrag. Prüf mit `offene_posten`, ob das Guthaben beim Kunden erscheint.

## 5. Zahlung ohne Rechnung

- **Eingang:** Kann eine Anzahlung sein; dann entsteht die Umsatzsteuer mit
  der Vereinnahmung
  ([rechnungen/anzahlungen.md](../rechnungen/anzahlungen.md#1-wann-die-steuer-entsteht)).
  Vorlegen.
- **Ausgang:** Rechnung anfordern und zuerst buchen
  ([belege/ablauf.md](../belege/ablauf.md)); Anzahlung an einen Lieferanten
  mit `anzahlung_fuer`, Vorsteuer nach
  [belege/vorsteuer.md](../belege/vorsteuer.md#2-zeitpunkt).
- **Erstattung eines Lieferanten:** gleicht den offenen Posten der gebuchten
  Gutschrift aus; fehlt sie, buche erst die Gutschrift als Beleg.

## 6. Rücklastschrift und Rücküberweisung

- Die Rücklastschrift ist ein eigener Umsatz; die ursprüngliche Zahlung bleibt
  gebucht ([grundlagen/gobd.md](../grundlagen/gobd.md#3-vollständigkeit)).
  Einen Posten öffnest du mit der Rücklastschrift wieder, nie durch Storno
  der Zahlungsbuchung.
- Die Forderung lebt wieder auf, meist mit einer Gebühr der Bank; ob der Kunde
  sie trägt, regelt der Vertrag. Frag den Menschen.
- Nach dem Ja `bankumsatz_buchen`: eine Zeile auf das Personenkonto des Kunden
  (`konto`, `kontakt_id`), die Gebühr als eigene Zeile `konto: 6855`. Prüf mit
  `offene_posten`, ob die Forderung wieder offen ist.

## 7. Umsatz ohne Beleg oder Verwendungszweck

- Für Kontoführung, Überweisungsentgelte, Zinsen und Überträge ist der
  Kontoauszug der Beleg; Buchfink legt ihn beim Import ab
  ([grundlagen/gobd.md](../grundlagen/gobd.md#8-belegwesen-und-eigenbeleg)).
- Für alles andere braucht es einen Beleg; gibt es keinen, erfasst der Mensch
  einen Eigenbeleg (ebenda).
- Ist der Zweck unklar, bleibt der Umsatz offen; die Kontierung entscheidet
  der Mensch. Sieht er privat aus:
  [gesellschafter/ablauf.md](../gesellschafter/ablauf.md#1-vorgang-einordnen).

## 8. Ignorieren

`ignorieren` nur, wenn der Geschäftsvorfall schon vollständig im Journal steht,
etwa eine per `buchung_erfassen` gebuchte Kreditkartenabrechnung oder ein auf
beiden Konten von Hand gebuchter Übertrag. Nenne im Grund die Buchungsnummer
([grundlagen/gobd.md](../grundlagen/gobd.md#3-vollständigkeit)). Unklare
Umsätze bleiben offen; der Prüflauf meldet sie.

## 9. Offene Posten über den Jahreswechsel

Aufwand und Ertrag gehören in das Jahr der Leistung
([jahresabschluss/abgrenzung.md](../jahresabschluss/abgrenzung.md#1-grundsatz)).
Die Zahlung im neuen Jahr gleicht mit ihrem Bankdatum den Posten des Vorjahres
aus (`offene_posten` zeigt alle Jahre); das Jahr des Postens bleibt.
