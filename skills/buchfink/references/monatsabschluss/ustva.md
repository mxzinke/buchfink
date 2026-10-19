# Umsatzsteuer-Voranmeldung

Rechtsstand: 17. Oktober 2026. Wortlaut von UStG, UStDV und AO am
17.10.2026 gelesen. Kennziffern nach dem Vordruck USt 1 A 2026, gelesen in einer
[Wiedergabe des Vordrucks][ust1a2026] (Formular-ID 2026USt1A561, Stand
Dezember 2025), und wie Buchfink sie führt.

Frist, Weg in Mein ELSTER, Nachweis und Folgen einer Versäumnis stehen in
[grundlagen/pflichten.md#ustva](../grundlagen/pflichten.md#ustva).

## Inhalt

1. Pflicht und Form
2. Voranmeldungszeitraum
3. Verspätungszuschlag
4. Kennziffern
5. Abgleich vor der Freigabe
6. Eingabe in Mein ELSTER

## 1. Pflicht und Form

- Die GmbH übermittelt nach Ablauf jedes Voranmeldungszeitraums eine
  Voranmeldung nach amtlich vorgeschriebenem Datensatz über die amtlich
  bestimmte Schnittstelle und berechnet die Vorauszahlung selbst
  ([§ 18 Abs. 1 Satz 1 UStG][ustg18]). Papier nur auf Antrag bei unbilliger
  Härte (Satz 2).
- Die Voranmeldung ist eine Steueranmeldung. Sie steht einer Festsetzung unter
  Vorbehalt der Nachprüfung gleich; ergibt sie einen Überschuss oder eine
  niedrigere Steuer, gilt das erst mit Zustimmung des Finanzamts
  ([§ 168 AO][ao168]).
- Buchfink führt nur die Sollversteuerung. Die Steuer entsteht mit Ablauf des
  Zeitraums, in dem die Leistung ausgeführt wurde, unabhängig von Rechnung und
  Zahlung ([§ 13 Abs. 1 Nr. 1 Buchst. a Satz 1 UStG][ustg13]). Eine fehlende
  Rechnung verschiebt die Steuer nicht. Für eine vor der Leistung
  vereinnahmte Anzahlung gilt
  [rechnungen/anzahlungen.md, Abschnitt 1](../rechnungen/anzahlungen.md#1-wann-die-steuer-entsteht).
- Bemessungsgrundlagen innergemeinschaftlicher Leistungen nach § 3a Abs. 2 UStG
  sind in der Voranmeldung gesondert zu erklären, im Zeitraum der Ausführung
  ([§ 18b Satz 1 Nr. 2, Satz 3 UStG][ustg18b]).

## 2. Voranmeldungszeitraum

| Lage | Zeitraum | Fundstelle |
|---|---|---|
| Regelfall | Kalendervierteljahr | [§ 18 Abs. 2 Satz 1 UStG][ustg18] |
| Steuer des Vorjahres über 9.000 € | Kalendermonat | § 18 Abs. 2 Satz 2 |
| Steuer des Vorjahres nicht über 2.000 € | Finanzamt kann von der Abgabe befreien | § 18 Abs. 2 Satz 3 |
| Überschuss des Vorjahres über 9.000 € | Monat wählbar; Voranmeldung Januar bis 10. Februar abgeben; bindet für das Jahr | [§ 18 Abs. 2a UStG][ustg18] |
| Tätigkeit beginnt (Neugründung) | Kalendermonat im laufenden und folgenden Jahr | § 18 Abs. 2 Satz 4 |
| Neugründung in den Besteuerungszeiträumen 2021 bis 2026 | abweichend von Satz 4 zählt die voraussichtliche Steuer des laufenden Jahres, bei Beginn im Vorjahr die auf ein Jahr hochgerechnete Steuer | § 18 Abs. 2 Satz 6 |
| Vorratsgesellschaft nimmt Tätigkeit auf, Übernahme eines Firmenmantels | Kalendermonat im laufenden und folgenden Jahr | § 18 Abs. 2 Satz 5 |

Stand: Oktober 2026. Die Grenzen ändern sich durch Gesetz. Die Sonderregel für
Neugründer gilt laut Wortlaut bis einschließlich 2026; prüfe für 2027 den
Wortlaut von § 18 Abs. 2 UStG.

In Buchfink: Buchfink schlägt den Zeitraum aus der Summe der Kennziffer 83
der übermittelten Anmeldungen des Vorjahres vor und nennt fehlende Zeiträume.
Liegt das Gründungsdatum der Unternehmensdaten im Jahr oder im Vorjahr, gilt
die Regel für Neugründungen: ab 2027 monatlich, bis 2026 im Folgejahr nach der
Steuer des Gründungsjahres, umgerechnet mit vollen Monaten ab dem
Gründungsmonat. Die voraussichtliche Steuer des Gründungsjahres kennt
Buchfink nicht; dann bleibt der Vorschlag beim eingestellten Zeitraum und nennt
die Regel. Eine Vorratsgesellschaft oder einen Firmenmantel erkennt Buchfink
nicht. Der Mensch stellt den Zeitraum in der App ein. Eine Befreiung oder eine
abweichende Festlegung des Finanzamts gibt er dort ebenfalls ein.

## 3. Verspätungszuschlag

Den Fristtag des einzelnen Zeitraums nennt `fristen` (Art `ustva`); Frist und
Folgen einer Versäumnis stehen in
[grundlagen/pflichten.md#ustva](../grundlagen/pflichten.md#ustva). Für
monatliche und vierteljährliche Anmeldungen gilt der Satz von 0,25 % je Monat
aus
[steuererklaerungen/fristen-zuschlaege.md](../steuererklaerungen/fristen-zuschlaege.md#4-verspätungszuschlag)
nicht; maßgeblich sind Dauer und Häufigkeit der Verspätung und die Höhe der
Steuer ([§ 152 Abs. 8 AO][ao152]). Stand: Oktober 2026.

## 4. Kennziffern

Die Kennziffern, die bei der Zielgruppe vorkommen (Vordruck USt 1 A 2026):

| Kz | Inhalt |
|---|---|
| 81 | Steuerpflichtige Umsätze 19 % (Bemessungsgrundlage) |
| 86 | Steuerpflichtige Umsätze 7 % |
| 21 | Nicht steuerbare sonstige Leistungen im übrigen Gemeinschaftsgebiet (§ 18b Satz 1 Nr. 2 UStG) |
| 45 | Übrige nicht steuerbare Umsätze (Leistungsort nicht im Inland) |
| 48 | Steuerfreie Umsätze ohne Vorsteuerabzug |
| 89, 93 | Innergemeinschaftliche Erwerbe 19 %, 7 % |
| 46 / 47 | Sonstige Leistungen nach § 3a Abs. 2 UStG eines im übrigen Gemeinschaftsgebiet ansässigen Unternehmers ([§ 13b Abs. 1 UStG][ustg13b]), Bemessungsgrundlage / Steuer (Zeile 30) |
| 84 / 85 | Andere Leistungen ([§ 13b Abs. 2 Nr. 1, 2, 4 bis 12 UStG][ustg13b]), u. a. Werklieferungen und sonstige Leistungen eines Unternehmers aus dem Drittland, etwa ein Software-Abo eines US-Anbieters (Zeile 32) |
| 66 | Vorsteuer aus Rechnungen anderer Unternehmer |
| 61 | Vorsteuer aus innergemeinschaftlichem Erwerb |
| 67 | Vorsteuer aus Leistungen nach § 13b UStG |
| 64 | Berichtigung des Vorsteuerabzugs (§ 15a UStG) |
| 69 | Unrichtig oder unberechtigt ausgewiesene Steuer (§ 14c UStG) |
| 39 | Anrechnung der Sondervorauszahlung (nur in der letzten Voranmeldung des Jahres) |
| 83 | Verbleibende Vorauszahlung; Überschuss mit Minus |
| 10 | Kennzeichen „berichtigte Anmeldung“ |

Die übrigen Kennziffern des Vordrucks (Lieferungen, Land- und Forstwirtschaft,
Dreiecksgeschäft, Fahrzeuge) betreffen die Zielgruppe nicht. Ist eine davon
belegt, frag nach dem Vorgang.

Ob eine § 13b-Steuer unter 46/47 oder 84/85 steht, folgt aus dem Steuerfall
der Buchung: `reverse_charge_eu` für die sonstige Leistung nach § 3a Abs. 2
UStG eines Unternehmers aus einem anderen EU-Staat (46/47), `reverse_charge`
für Drittland, Werklieferung und Bauleistung (84/85). Wählt eine Buchung
`reverse_charge` für einen EU-Lieferanten, warnt Buchfink mit
`reverse_charge_eu_supplier`; kläre jede solche Warnung, bevor der Zeitraum
abgegeben wird.

Ab dem Kalenderjahr 2026 gilt der Vordruck nach dem BMF-Schreiben vom
29.12.2025 (III C 3 – S 7344/00039/007/036; Wortlaut des Schreibens nicht
gelesen). Im Vordruck steht in Zeile 55 die Kennzahl 500 „Ergänzende Angaben
zur Steueranmeldung“ mit vier Schlüsseln: 1 steuererhebliche Sachverhalte
konnten nicht erklärt werden, 2 bewusst abweichende Rechtsauffassung,
3 Sachverhalte sollen personell vertieft geprüft werden, 4 mehrere Gründe. Die
Erläuterung geht als gesonderte Anlage mit dieser Überschrift ein und wird
gesondert geprüft ([Vordruck USt 1 A 2026][ust1a2026]). Nach Sekundärquellen
ersetzt Kennzahl 500 das frühere Freitextfeld Kennzahl 23. Setze Kennzahl 500
nur nach Rückfrage.

Bemessungsgrundlagen führt Buchfink in vollen Euro (Cent abgeschnitten),
Steuerbeträge mit Cent. Übertrage beides unverändert.

## 5. Abgleich vor der Freigabe

`erklaerung` liefert je Kennzahl `kz`, `bemessungsgrundlage` und `steuer`,
dazu `zahllast`, `befunde` und `zahlen_hash`. Die Konten hinter einer Kennzahl
zeigen `auswertung` (`art: kontoblatt`) und `buchungen_suchen` (Konto,
Zeitraum); die App zeigt die Herkunft in der formularnahen Ansicht. Prüfe:

1. **Befunde.** Jeder Eintrag in `befunde` ist geklärt oder dem Menschen
   vorgelegt.
2. **Rechnerische Steuer.** Bei 81 und 86 ist die gebuchte Steuer gleich Satz
   mal ungerundeter Bemessungsgrundlage, bis auf Rundungscent. Größere
   Abweichungen: Buchung mit falschem Steuerbetrag suchen
   (`buchungen_suchen` mit Konto und Zeitraum).
3. **Herkunft.** Kennziffer 21 stammt aus Erlösen mit Steuerschuld des
   Empfängers im EU-Ausland (SKR04 4336), Kennziffer 66 aus den
   Vorsteuerkonten (1401, 1406). Taucht ein Konto auf, das nicht passt, such die
   Buchung.
4. **Paare.** 47 und 85 haben ihr Gegenstück in 67, die Steuer aus 89 und 93
   in 61, sofern der Vorsteuerabzug voll ist. Ein Unterschied ist nur bei teilweisem
   Vorsteuerabzug richtig (Vorsteueranteil am Beleg).
5. **Nachträge.** Meldet `erklaerung` Buchungen zu einem schon übermittelten
   Zeitraum, gehören sie nicht in den laufenden. Behandle sie als Berichtigung
   des alten Zeitraums ([berichtigung.md](berichtigung.md)).
6. **Vergleich mit dem Vorzeitraum.** Starke Ausschläge bei 81 oder 66 nennst
   du dem Menschen mit den größten Einzelbuchungen.
7. **Zahllast.** Kennziffer 83 gleich geschuldete Steuer minus Vorsteuer minus
   Kennziffer 39.

Fertig, wenn alle sieben Punkte erledigt sind und der Mensch die Zahlen
bestätigt hat.

## 6. Eingabe in Mein ELSTER

Den Ablauf hält [ablauf.md](ablauf.md#3-in-mein-elster-übertragen). Für die
Felder gilt:

- Formular „Umsatzsteuer-Voranmeldung“ des Jahres, in dem der Zeitraum liegt.
  Der Vordruck ändert sich jährlich; Buchfink zeigt die Kennzahlen des Jahres.
- Bemessungsgrundlagen in vollen Euro, Steuerbeträge mit Cent, so wie
  Buchfink sie liefert (Abschnitt 4).
- Kennzahl 39 nur in der letzten Voranmeldung des Jahres; Kennzeichen
  „berichtigte Anmeldung“ (Kennzahl 10) nur bei einer Berichtigung;
  Kennzahl 500 nur nach Rückfrage (Abschnitt 4).

[ustg13]: https://www.gesetze-im-internet.de/ustg_1980/__13.html
[ustg13b]: https://www.gesetze-im-internet.de/ustg_1980/__13b.html
[ustg18]: https://www.gesetze-im-internet.de/ustg_1980/__18.html
[ustg18b]: https://www.gesetze-im-internet.de/ustg_1980/__18b.html
[ao152]: https://www.gesetze-im-internet.de/ao_1977/__152.html
[ao168]: https://www.gesetze-im-internet.de/ao_1977/__168.html
[ust1a2026]: https://www.buhl.de/steuer/wp-content/uploads/sites/31/2026/03/formular-dauerfristverlaengerung-2026.pdf
