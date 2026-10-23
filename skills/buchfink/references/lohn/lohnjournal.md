# Lohnjournal: Felder, Konten, Sonderzeilen

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

## Inhalt

1. Die Summenfelder
2. Wohin die Beträge gehören
3. Gegenkonten der Zahlungen
4. Sonderzeilen im Journal
5. Was Buchfink nicht abbildet

## 1. Die Summenfelder

| Feld | Inhalt |
|---|---|
| `gehaelter` | Arbeitslohn der Arbeitnehmer ohne Geschäftsführer und Minijobs |
| `gf_gehaelter_gesellschafter` | Gehalt der Geschäftsführer, die Gesellschafter sind |
| `gf_gehaelter` | Gehalt der übrigen Geschäftsführer |
| `minijob_loehne` | Löhne für Minijobs |
| `an_sozialversicherung` | Arbeitnehmeranteil Kranken-, Pflege-, Renten-, Arbeitslosenversicherung |
| `ag_sozialversicherung` | Arbeitgeberanteil derselben Zweige |
| `lohnsteuer`, `kirchensteuer`, `soli` | einbehaltene Lohnsteuer, Kirchensteuer, Solidaritätszuschlag |
| `netto` | Brutto minus gesetzliche Abzüge (Nettoverdienst) |
| `umlagen` | U1 und U2 ([§ 7 AAG][aag7]), Insolvenzgeldumlage ([§ 358 SGB III][sgb3_358]) |
| `minijob_abgaben` | Pauschalbeiträge des Arbeitgebers für Minijobs |
| `minijob_pauschsteuer` | einheitliche Pauschsteuer für Minijobs ([§ 40a Abs. 2 EStG][estg40a]) |
| `pauschale_lohnsteuer` | pauschale Lohnsteuer, die der Arbeitgeber übernimmt und schuldet ([§ 40 Abs. 3 EStG][estg40]) |
| `pauschale_lohnsteuer_gf` | dieselbe für Gesellschafter-Geschäftsführer |

- Netto-Probe: `gehaelter + gf_gehaelter_gesellschafter + gf_gehaelter +
  minijob_loehne − an_sozialversicherung − lohnsteuer − kirchensteuer − soli
  = netto`. Arbeitgeberanteil, Umlagen und Pauschalsteuern sind Aufwand
  außerhalb der Probe.
- In `netto` steht der Nettoverdienst; die Differenz zur Auszahlung klärst du
  gesondert (Abschnitt 4).

## 2. Wohin die Beträge gehören

Die Konten wählt Buchfink; die Antwort nennt sie. Frag beim ersten Lohnmonat
eines Mandanten, wie das Journal gegliedert ist, und prüfe, ob jede Summe auf
dem passenden Konto steht. Verbindlichkeiten: 3720 Netto (3790 mit
`netto_auf_verrechnungskonto`), 3730 Lohnsteuer, Kirchensteuer,
Solidaritätszuschlag und pauschale Lohnsteuer, 3740 Sozialversicherung,
Umlagen und Minijob-Zentrale samt Pauschsteuer ([§ 40a Abs. 6 EStG][estg40a]).

## 3. Gegenkonten der Zahlungen

| Zahlung | Empfänger | Gegenkonto |
|---|---|---|
| Auszahlung an Beschäftigte | Beschäftigte | 3720 |
| Lohnsteuer, Kirchensteuer, Solidaritätszuschlag, pauschale Lohnsteuer (außer Minijob) | Betriebsstättenfinanzamt ([§ 41a Abs. 1 EStG][estg41a]) | 3730 |
| Gesamtsozialversicherungsbeitrag und Umlagen | Krankenkassen als Einzugsstellen ([§ 28e Abs. 1 SGB IV][sgb4_28e]) | 3740 |
| Minijob-Abgaben und einheitliche Pauschsteuer | Minijob-Zentrale (Deutsche Rentenversicherung Knappschaft-Bahn-See, [§ 40a Abs. 6 Satz 1 EStG][estg40a]) | 3740 |
| Erstattung nach dem Aufwendungsausgleichsgesetz | von der Krankenkasse | 4972 Erstattungen Aufwendungsausgleichsgesetz |

Die Zuordnung am Bankumsatz regelt [bank/zuordnung.md](../bank/zuordnung.md).

## 4. Sonderzeilen im Journal

Diese Zeilen lassen die Auszahlung vom Netto abweichen. Lege sie vor und
erfasse die Umbuchung mit `buchung_erfassen` (Rückfrage), Beleg ist das
Journal. Der Buchungstext nennt keine Namen und Einzelbeträge von
Beschäftigten.

| Zeile | Umbuchung nach dem Journal | Hinweis |
|---|---|---|
| Vorschuss, früher ausgezahlt | 3720 an 1340 Forderungen gegen Personal aus Lohn- und Gehaltsabrechnung | Den Vorschuss selbst buchst du am Bankumsatz auf 1340 ([bank/zuordnung.md](../bank/zuordnung.md)) |
| Pfändung, Abtretung | 3720 an 3725 Verbindlichkeiten für Einbehaltungen von Arbeitnehmern | Zahlung an den Gläubiger gegen 3725 |
| Sparbetrag vermögenswirksame Leistung | 3720 an 3725 | Zahlung an das Anlageinstitut gegen 3725; der Arbeitgeberzuschuss steckt im Brutto (6080 Vermögenswirksame Leistungen) |
| Sachbezug, Firmenwagen (Nettoabzug) | 3720 an Ertrag für Sachbezüge (z. B. 4947 Verrechnete sonstige Sachbezüge aus Fahrzeug-Gestellung 19 % USt, 4949 ohne Umsatzsteuer) | Wertungsfrage, siehe unten |
| Steuerfreie Erstattung (Reisekosten) über die Abrechnung | Aufwand (z. B. 6650 Reisekosten Arbeitnehmer) an 3720 | Belege der Reise nach [belege/ablauf.md](../belege/ablauf.md); frag, ob sie schon gebucht sind |

Sachbezug und Firmenwagen:

- Der geldwerte Vorteil ist Arbeitslohn ([§ 8 Abs. 2 EStG][estg8]); das
  Lohnbüro rechnet ihn ins Brutto und zieht ihn vom Netto ab.
- Die Überlassung eines Fahrzeugs zur Privatnutzung an Beschäftigte, auch an
  den Gesellschafter-Geschäftsführer mit Arbeitsvertrag, ist in der Regel eine
  umsatzsteuerpflichtige entgeltliche sonstige Leistung
  ([UStAE 15.23 Abs. 8 Satz 1, 4][ustae1523]). Bemessungsgrundlage und Satz
  klärt der Mensch mit Lohnbüro oder Steuerberater; Buchfink rechnet sie
  nicht (Wertungsfrage).
- Ohne Regelung im Anstellungsvertrag ist die Privatnutzung durch einen
  beherrschenden Gesellschafter vGA-verdächtig
  ([gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md)).

## 5. Was Buchfink nicht abbildet

- Lohnabrechnung, Lohnsteuer-Anmeldung, Beitragsnachweise, Meldungen zur
  Sozialversicherung, elektronische Lohnsteuerbescheinigung.
- Kurzarbeitergeld, Altersteilzeit, betriebliche Altersversorgung mit
  Pauschalsteuer (6147), Entgeltumwandlung: Rückfrage, im Zweifel Grenze.
- Personalrückstellungen (Urlaub, Überstunden) zum Stichtag:
  [jahresabschluss/rueckstellungen.md](../jahresabschluss/rueckstellungen.md).

[estg8]: https://www.gesetze-im-internet.de/estg/__8.html
[estg40]: https://www.gesetze-im-internet.de/estg/__40.html
[estg40a]: https://www.gesetze-im-internet.de/estg/__40a.html
[estg41a]: https://www.gesetze-im-internet.de/estg/__41a.html
[sgb4_28e]: https://www.gesetze-im-internet.de/sgb_4/__28e.html
[sgb3_358]: https://www.gesetze-im-internet.de/sgb_3/__358.html
[aag7]: https://www.gesetze-im-internet.de/aufag/__7.html
[ustae1523]: https://www.steuerschroeder.de/steuergesetze/ustae/15.23
