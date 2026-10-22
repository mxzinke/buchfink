# Umsatzsteuer-Voranmeldung

Rechtsstand: 17. Oktober 2026. Kennziffern nach Vordruck USt 1 A 2026
([Wiedergabe][ust1a2026], Stand Dezember 2025).

Frist, Weg, Nachweis und Folgen:
[grundlagen/pflichten.md#ustva](../grundlagen/pflichten.md#ustva).

## Inhalt

1. Pflicht und Form
2. Voranmeldungszeitraum
3. Verspätungszuschlag
4. Kennziffern
5. Abgleich vor der Freigabe
6. Eingabe in Mein ELSTER

## 1. Pflicht und Form

- Nach jedem Voranmeldungszeitraum elektronische Voranmeldung mit selbst
  berechneter Vorauszahlung ([§ 18 Abs. 1 Satz 1 UStG][ustg18]).
- Sie steht einer Festsetzung unter Vorbehalt der Nachprüfung gleich; ein
  Überschuss oder eine niedrigere Steuer wirkt erst mit Zustimmung des
  Finanzamts ([§ 168 AO][ao168]).
- Buchfink führt nur die Sollversteuerung: Die Steuer entsteht mit Ablauf des
  Zeitraums der Leistung, unabhängig von Rechnung und Zahlung
  ([§ 13 Abs. 1 Nr. 1 Buchst. a Satz 1 UStG][ustg13]). Anzahlungen:
  [rechnungen/anzahlungen.md, Abschnitt 1](../rechnungen/anzahlungen.md#1-wann-die-steuer-entsteht).

## 2. Voranmeldungszeitraum

| Lage | Zeitraum | Fundstelle |
|---|---|---|
| Regelfall | Kalendervierteljahr | [§ 18 Abs. 2 Satz 1 UStG][ustg18] |
| Steuer des Vorjahres über 9.000 € | Kalendermonat | § 18 Abs. 2 Satz 2 |
| Steuer des Vorjahres nicht über 2.000 € | Finanzamt kann befreien | § 18 Abs. 2 Satz 3 |
| Überschuss des Vorjahres über 9.000 € | Monat wählbar durch Voranmeldung Januar bis 10. Februar; bindet für das Jahr | [§ 18 Abs. 2a UStG][ustg18] |
| Neugründung | Kalendermonat im laufenden und folgenden Jahr | § 18 Abs. 2 Satz 4 |
| Neugründung 2021 bis 2026 | statt Satz 4 die voraussichtliche Steuer des laufenden Jahres, bei Beginn im Vorjahr die hochgerechnete | § 18 Abs. 2 Satz 6 |
| Vorratsgesellschaft nimmt Tätigkeit auf, Firmenmantel übernommen | Kalendermonat im laufenden und folgenden Jahr | § 18 Abs. 2 Satz 5 |

Die App schlägt einen Zeitraum aus Kennziffer 83 der Vorjahresanmeldungen vor;
Gründungsjahr, Vorratsgesellschaft und Firmenmantel kennt sie nicht. Den
Zeitraum, auch eine Befreiung (`keiner`) oder eine Festlegung des Finanzamts,
schlägst du mit `stammdaten_vorschlagen` (`ust_zeitraum`) vor.

## 3. Verspätungszuschlag

Der Satz von 0,25 % je Monat
([steuererklaerungen/fristen-zuschlaege.md](../steuererklaerungen/fristen-zuschlaege.md#4-verspätungszuschlag))
gilt nicht; maßgeblich sind Dauer, Häufigkeit und Steuerhöhe
([§ 152 Abs. 8 AO][ao152]).

## 4. Kennziffern

| Kz | Inhalt |
|---|---|
| 81 | Steuerpflichtige Umsätze 19 % (Bemessungsgrundlage) |
| 86 | Steuerpflichtige Umsätze 7 % |
| 21 | Nicht steuerbare sonstige Leistungen im übrigen Gemeinschaftsgebiet (§ 18b Satz 1 Nr. 2 UStG) |
| 45 | Übrige nicht steuerbare Umsätze (Leistungsort nicht im Inland) |
| 48 | Steuerfreie Umsätze ohne Vorsteuerabzug |
| 89, 93 | Innergemeinschaftliche Erwerbe 19 %, 7 % |
| 46 / 47 | Sonstige Leistungen nach § 3a Abs. 2 UStG eines EU-Unternehmers ([§ 13b Abs. 1 UStG][ustg13b]), Bemessungsgrundlage / Steuer |
| 84 / 85 | Andere Leistungen ([§ 13b Abs. 2 Nr. 1, 2, 4 bis 12 UStG][ustg13b]), u. a. Werklieferungen und Leistungen aus dem Drittland (Software-Abo eines US-Anbieters) |
| 66 | Vorsteuer aus Rechnungen anderer Unternehmer |
| 61 | Vorsteuer aus innergemeinschaftlichem Erwerb |
| 67 | Vorsteuer aus Leistungen nach § 13b UStG |
| 64 | Berichtigung des Vorsteuerabzugs (§ 15a UStG) |
| 69 | Unrichtig oder unberechtigt ausgewiesene Steuer (§ 14c UStG) |
| 39 | Anrechnung der Sondervorauszahlung (nur letzte Voranmeldung des Jahres) |
| 83 | Verbleibende Vorauszahlung; Überschuss mit Minus |
| 10 | Kennzeichen „berichtigte Anmeldung“ |

Eine andere belegte Kennziffer (Lieferungen, Land- und Forstwirtschaft,
Dreiecksgeschäft, Fahrzeuge): frag nach dem Vorgang.

46/47 oder 84/85 folgt aus dem Steuerfall: `reverse_charge_eu` (sonstige
Leistung eines EU-Unternehmers) → 46/47, `reverse_charge` (Drittland,
Werklieferung, Bauleistung) → 84/85. Steht ein EU-Lieferant unter 84/85
(Warnung `reverse_charge_eu_supplier`), kläre vor der Abgabe, ob 46/47
richtig ist.

Kennzahl 500 „Ergänzende Angaben zur Steueranmeldung“ nur nach Rückfrage;
Schlüssel 1 bis 4 und gesonderte Anlage nach dem
[Vordruck USt 1 A 2026][ust1a2026].

## 5. Abgleich vor der Freigabe

Konten hinter einer Kennzahl: `auswertung` (`art: kontoblatt`),
`buchungen_suchen` (Konto, Zeitraum).

1. **Befunde.** Jeder Eintrag in `befunde` ist geklärt oder vorgelegt.
2. **Rechnerische Steuer.** Bei 81 und 86 gleich Satz mal ungerundeter
   Bemessungsgrundlage, bis auf Rundungscent; sonst Buchung mit falschem
   Steuerbetrag suchen.
3. **Herkunft.** Kz 21 aus SKR04 4336, Kz 66 aus 1401, 1406. Ein unpassendes
   Konto: Buchung suchen.
4. **Paare.** 47 und 85 gleich 67, Steuer aus 89 und 93 gleich 61; Unterschied
   nur bei teilweisem Vorsteuerabzug.
5. **Nachträge.** Buchungen zu einem übermittelten Zeitraum sind Berichtigung
   des alten Zeitraums ([berichtigung.md](berichtigung.md)).
6. **Vorzeitraum.** Starke Ausschläge bei 81 oder 66 mit den größten
   Einzelbuchungen nennen.
7. **Zahllast.** Kz 83 gleich Steuer minus Vorsteuer minus Kz 39.

Fertig, wenn alle sieben Punkte erledigt sind und der Mensch die Zahlen
bestätigt hat.

## 6. Eingabe in Mein ELSTER

- Formular des Jahres, in dem der Zeitraum liegt.
- Werte aus `erklaerung` unverändert: Bemessungsgrundlagen in vollen Euro
  (Buchfink schneidet die Cent ab), Steuerbeträge mit Cent.
- Kennzahl 10 nur bei Berichtigung, Kennzahl 500 nur nach Rückfrage.

[ustg13]: https://www.gesetze-im-internet.de/ustg_1980/__13.html
[ustg13b]: https://www.gesetze-im-internet.de/ustg_1980/__13b.html
[ustg18]: https://www.gesetze-im-internet.de/ustg_1980/__18.html
[ao152]: https://www.gesetze-im-internet.de/ao_1977/__152.html
[ao168]: https://www.gesetze-im-internet.de/ao_1977/__168.html
[ust1a2026]: https://www.buhl.de/steuer/wp-content/uploads/sites/31/2026/03/formular-dauerfristverlaengerung-2026.pdf
