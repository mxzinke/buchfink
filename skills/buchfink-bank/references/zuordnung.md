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

`bankumsaetze` liefert je Umsatz die besten drei `vorschlaege` mit `art`
(`offener_posten`, `sammelzahlung`, `regel`, `kreditkarte`), `punkte`,
`gruende`, `text`, `kontakt_id`, `betrag`, `posten [{buchung_id, beleg_nr,
offen, faellig}]`, `gegenkonto`, bei Kreditkarte `stichtag`, und `einstufung`.
Alle Vorschläge liefert der Aufruf mit `bankumsatz_id`. Ein Vorschlag ist ein
Hinweis, keine Entscheidung:

- Stimmen Betrag, Kontakt und Rechnungsnummer, ist der Vorschlag meist richtig.
- Liegen zwei Vorschläge nahe beieinander, entscheidet der Mensch.
- Passt die IBAN der Gegenpartei nicht zum Kontakt, legst du den Umsatz vor.

Offene Posten siehst du mit `offene_posten` (`richtung: forderung` oder
`verbindlichkeit`), auch je `kontakt_id`.

## 2. Routine

Routine ist nur der Ausgleich genau eines offenen Postens:
`bankumsatz_zuordnen` mit `bankumsatz_id` und `buchung_id`. Die Bedingungen
stehen in buchfink-grundlagen (`references/arbeitsweise.md`, Abschnitt 4):
ein offener Posten in der Richtung des Umsatzes, centgenau, seine Beleg-,
Buchungs- oder Rechnungsnummer im Verwendungszweck, Auszug geprüft. Sonst
meldet das Werkzeug `freigabe_noetig` mit Gründen wie `auszug_ungeprueft`,
`kein_exakter_treffer` oder `mehrere_treffer`.

Vorschläge der Art `regel` stammen aus früheren Buchungen desselben Umsatzes
gegen ein Konto (`internal/service/bank_service.go`), Vorschläge der Art
`kreditkarte` aus dem Saldo von 3610. Beide sind Rückfrage.

## 3. Aufteilung schreiben

`bankumsatz_buchen` nimmt genau eines von `aufteilung`,
`kreditkartenabrechnung {stichtag}` oder `ignorieren {grund}`, dazu
`bankumsatz_id` und `beschreibung`. Die `aufteilung` hat 1 bis 20 Zeilen:

- Zeile mit `buchung_id` und `ausgleich`: gleicht einen offenen Posten aus;
  `ausgleich` ist der Betrag, um den der Posten sinkt (bei Skonto der volle
  Betrag). Optional `differenz {art, betrag}` mit `skonto`, `gebuehr`,
  `rundung` oder `waehrung`.
- Zeile mit `konto` und `betrag`: bucht gegen ein Sachkonto; bei einem
  Personenkonto mit `kontakt_id`.
- Zeile mit `darlehen_id` und `betrag`: Auszahlung oder Tilgung eines
  Gesellschafterdarlehens, das `darlehen_erfassen` angelegt hat.
- `text` je Zeile für den Prüfer.

Die Zeilen ergeben zusammen den Umsatz. Buchfink zieht beim Ausgleich Skonto,
Rundung und Kursdifferenz ab und schlägt eine Gebühr auf, bevor es mit dem
Umsatz vergleicht (`internal/service/payment_service.go`, `cashAmount`).
`betrag` ist bei Konto- und Darlehenszeilen positiv, auch bei einer Abbuchung
(`"4.90"` für eine Gebühr von −4,90 €). Negativ ist er nur als Abzug innerhalb
der Aufteilung, etwa die Kapitalertragsteuer einer Zinsgutschrift. Die Zeilen
tragen keine Umsatzsteuer; Vorgänge mit Steuer buchst du über ihren Beleg
(buchfink-belege).

Beispiel Skonto auf eine Eingangsrechnung über 1.190,00 €, gezahlt 1.166,20 €:

```
aufteilung: [
  {buchung_id: 812, ausgleich: "1190.00",
   differenz: {art: skonto, betrag: "23.80"}}
]
```

Buchfink bucht daraus Verbindlichkeit 1.190,00 an Bank 1.166,20, erhaltene
Skonti 20,00 und Vorsteuerkorrektur 3,80 (`internal/service/payment_service.go`).

Schreib `text` so, dass ein Prüfer ohne Chat versteht, was geschah
(„Zinsen Q3 2026 Tagesgeld, Abrechnung vom 30.09.2026“).

## 4. Teilzahlung, Sammelzahlung, Überzahlung

- **Teilzahlung:** eine Zeile mit `buchung_id` und dem gezahlten Betrag als
  `ausgleich`. Der Rest bleibt offen. Frag, ob eine Ratenvereinbarung besteht oder ob der Rest strittig ist;
  Mahnung und Ausfall regelt buchfink-rechnungen.
- **Sammelzahlung:** je Rechnung eine Zeile mit `buchung_id`. Fehlt im
  Verwendungszweck die Liste, frag nach dem Zahlungsavis.
- **Überzahlung und Doppelzahlung:** Ein Posten nimmt höchstens seinen offenen
  Betrag auf. Gleiche den Posten aus und leg den Rest vor. Der Mensch
  entscheidet: erstatten oder mit der nächsten Rechnung verrechnen. Buche den
  Rest nie als Ertrag. Nach seiner Entscheidung bucht eine zweite Zeile den
  Rest auf das Personenkonto des Kunden (`konto` aus `kontakte_suchen`,
  `kontakt_id`). Prüf danach mit `offene_posten`, ob das Guthaben beim Kunden
  erscheint; sonst frag den Menschen.

## 5. Zahlung ohne Rechnung

- **Eingang ohne Rechnung:** Kann eine Anzahlung sein. Bei Anzahlungen entsteht
  die Umsatzsteuer schon mit der Vereinnahmung
  ([§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG][ustg13]). Leg den Fall vor;
  Abschläge und Anzahlungen regelt buchfink-rechnungen.
- **Ausgang ohne Rechnung:** Fordere die Rechnung an und buche sie zuerst
  (buchfink-belege). Ist es eine Anzahlung an einen Lieferanten, gilt dort
  `anzahlung_fuer`. Vorsteuer daraus erst mit Rechnung und Zahlung
  ([§ 15 Abs. 1 Satz 1 Nr. 1 Satz 3 UStG][ustg15]).
- **Erstattung eines Lieferanten:** Ist eine Gutschrift gebucht, gleicht der
  Eingang deren offenen Posten aus. Sonst erst die Gutschrift als Beleg
  (buchfink-belege).

## 6. Rücklastschrift und Rücküberweisung

- Eine Rücklastschrift ist ein eigener Umsatz. Die ursprüngliche Zahlung bleibt
  gebucht; der Kontoauszug zeigt beide ([GoBD Rz. 36][gobd], vollständig und
  einzeln).
- Die Forderung lebt wieder auf, meist zuzüglich einer Rücklastschriftgebühr
  der Bank (6855 Nebenkosten des Geldverkehrs). Ob der Kunde die Gebühr tragen muss, ist
  eine Frage des Vertrags; frag den Menschen.
- Die Rücklastschrift buchst du nach dem Ja mit `bankumsatz_buchen`: eine
  Zeile auf das Personenkonto des Kunden (`konto`, `kontakt_id`), die Gebühr
  als eigene Zeile `konto: 6855`. Prüf danach mit `offene_posten`, ob die
  Forderung wieder offen ist; sonst frag den Menschen.

## 7. Umsatz ohne Beleg oder Verwendungszweck

- Für Kontoführung, Überweisungsentgelte, Zinsen und Überträge ist der
  Kontoauszug der Beleg. Buchfink legt ihn beim Import als Beleg ab; eine
  strukturierte Kontoauszugsdatei erfüllt die Belegfunktion
  ([GoBD Rz. 76][gobd]).
- Für alles andere braucht es einen Beleg ([GoBD Rz. 61][gobd]). Fehlt er, frag
  danach. Gibt es keinen, erfasst der Mensch einen Eigenbeleg
  (buchfink-grundlagen).
- Ist der Zweck unklar, bleibt der Umsatz offen. Rate kein Konto; die
  Entscheidung über die Kontierung trägt der Mensch.
- Ein Umsatz, der wie eine private Ausgabe aussieht, gehört zu
  buchfink-gesellschafter.

## 8. Ignorieren

`ignorieren {grund}` setzt den Umsatz auf „ignoriert“. Zulässig nur, wenn der
Geschäftsvorfall schon vollständig im Journal steht, etwa:

- die Kreditkartenabrechnung wurde ohne Bankumsatz mit `buchung_erfassen`
  gebucht und der Umsatz käme doppelt,
- ein Übertrag wurde auf beiden Konten von Hand gebucht.

Nenne im Grund die Buchungsnummer. Jeder Vorgang wird genau einmal erfasst
([§ 146 Abs. 1 AO][ao146], [GoBD Rz. 36, 41][gobd]).

## 9. Offene Posten über den Jahreswechsel

- Aufwand und Ertrag gehören in das Jahr der Leistung, unabhängig von der
  Zahlung ([§ 252 Abs. 1 Nr. 5 HGB][hgb252]). Die Dezemberrechnung, die im
  Januar bezahlt wird, gleicht im Januar den offenen Posten des Vorjahres aus.
- `offene_posten` zeigt Posten aller Jahre. Buche die Zahlung mit ihrem
  Bankdatum; das Jahr des Postens ändert sich dadurch nicht.

[gobd]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf
[ao146]: https://www.gesetze-im-internet.de/ao_1977/__146.html
[hgb252]: https://www.gesetze-im-internet.de/hgb/__252.html
[ustg13]: https://www.gesetze-im-internet.de/ustg_1980/__13.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
