# Lohnjournal: Felder, Konten, Sonderzeilen

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0. Die Konten, auf
die `lohnjournal_buchen` bucht, legt Buchfink fest; maßgeblich sind die Zeilen
in der Antwort. Die Tabellen unten zeigen, wohin die Beträge nach den
Kontenbezeichnungen des SKR04 gehören, damit du die Antwort prüfen kannst.

## Inhalt

1. Die Summenfelder
2. Wohin die Beträge gehören
3. Gegenkonten der Zahlungen
4. Sonderzeilen im Journal
5. Was Buchfink nicht abbildet

## 1. Die Summenfelder

| Feld | Inhalt | Im Journal meist |
|---|---|---|
| `gehaelter` | Arbeitslohn der Arbeitnehmer im Monat (6020) | Gesamtbrutto ohne Geschäftsführer und Minijobs |
| `gf_gehaelter_gesellschafter` | Gehalt der Geschäftsführer, die Gesellschafter sind (6024) | Geschäftsführer |
| `gf_gehaelter` | Gehalt der übrigen Geschäftsführer (6027) | Geschäftsführer |
| `minijob_loehne` | Löhne für Minijobs (6035) | Minijob |
| `an_sozialversicherung` | Arbeitnehmeranteil Kranken-, Pflege-, Renten-, Arbeitslosenversicherung | SV-Anteil AN |
| `ag_sozialversicherung` | Arbeitgeberanteil derselben Zweige | SV-Anteil AG |
| `lohnsteuer` | einbehaltene Lohnsteuer | LSt |
| `kirchensteuer` | einbehaltene Kirchensteuer | KiSt |
| `soli` | einbehaltener Solidaritätszuschlag | SolZ |
| `netto` | Brutto minus gesetzliche Abzüge (Nettoverdienst); Pflichtfeld | Netto |
| `umlagen` | U1 und U2 nach dem Aufwendungsausgleichsgesetz ([§ 7 AAG][aag7]), Insolvenzgeldumlage ([§ 358 SGB III][sgb3_358]) | Umlagen |
| `minijob_abgaben` | Pauschalbeiträge des Arbeitgebers für Minijobs | Abgaben Minijob-Zentrale |
| `minijob_pauschsteuer` | einheitliche Pauschsteuer für Minijobs ([§ 40a Abs. 2 EStG][estg40a]) | Pauschsteuer Minijob |
| `pauschale_lohnsteuer` | pauschale Lohnsteuer, die der Arbeitgeber übernimmt und schuldet ([§ 40 Abs. 3 EStG][estg40]) | Pauschale Steuern |
| `pauschale_lohnsteuer_gf` | dieselbe für Gesellschafter-Geschäftsführer | Pauschale Steuern |
| `netto_auf_verrechnungskonto` | `true` bucht das Netto auf 3790 Lohn- und Gehaltsverrechnungskonto statt 3720 | – |

Dazu `monat` (Abrechnungsmonat `JJJJ-MM`), `beleg_id`, auf Wunsch
`buchungsdatum` (Vorgabe letzter Tag des Monats) und bei der Dezemberbuchung
im Abschluss `abschlussperiode`. Die Feldnamen nennt das Schema des Werkzeugs.

- Netto-Probe: `gehaelter + gf_gehaelter_gesellschafter + gf_gehaelter +
  minijob_loehne − an_sozialversicherung − lohnsteuer − kirchensteuer − soli
  = netto`.
- Der Auszahlungsbetrag kann vom Netto abweichen (Abschnitt 4). Trag in `netto`
  den Nettoverdienst ein und kläre die Differenz zur Auszahlung gesondert.

## 2. Wohin die Beträge gehören

| Betrag | Soll (SKR04) | Haben (SKR04) |
|---|---|---|
| Brutto | 6020 Gehälter; 6024 Geschäftsführergehälter der GmbH-Gesellschafter; 6027 Geschäftsführergehälter; 6035 Löhne für Minijobs | – |
| Netto | – | 3720 Verbindlichkeiten aus Lohn und Gehalt |
| Lohnsteuer, Kirchensteuer, Solidaritätszuschlag | – | 3730 Verbindlichkeiten aus Lohn- und Kirchensteuer |
| Arbeitnehmeranteil SV | – | 3740 Verbindlichkeiten im Rahmen der sozialen Sicherheit |
| Arbeitgeberanteil SV | 6110 Gesetzliche soziale Aufwendungen; Minijobs 6171 Soziale Abgaben für Minijobber | 3740 |
| Umlagen | Aufwand der sozialen Abgaben (6110 oder 6170 Sonstige soziale Abgaben) | 3740 |
| Pauschalsteuer | 6036 Pauschale Steuer für Minijobber; 6037 für Gesellschafter-Geschäftsführer; 6039 für Arbeitnehmer; 6040 für Aushilfen | 3730; bei Minijobs 3740, weil sie an die Minijob-Zentrale geht ([§ 40a Abs. 6 EStG][estg40a]) |

Geschäftsführergehälter (6024, 6027) und Minijobs (6035) bucht Buchfink über
ihre eigenen Felder getrennt. Frag beim ersten Lohnmonat eines Mandanten, wie
das Journal gegliedert ist, und vergleiche mit der Antwort.

Der SKR04 führt außerdem 3759 Voraussichtliche Beitragsschuld gegenüber den
Sozialversicherungsträgern und 3790 Lohn- und Gehaltsverrechnungskonto. Nutzt
die Antwort sie, gleiche sie in der Abstimmung mit aus.

## 3. Gegenkonten der Zahlungen

| Zahlung | Empfänger | Gegenkonto |
|---|---|---|
| Auszahlung an Beschäftigte | Beschäftigte | 3720 |
| Lohnsteuer, Kirchensteuer, Solidaritätszuschlag, pauschale Lohnsteuer (außer Minijob) | Betriebsstättenfinanzamt ([§ 41a Abs. 1 EStG][estg41a]) | 3730 |
| Gesamtsozialversicherungsbeitrag und Umlagen | Krankenkassen als Einzugsstellen ([§ 28e Abs. 1 SGB IV][sgb4_28e]) | 3740 |
| Minijob-Abgaben und einheitliche Pauschsteuer | Minijob-Zentrale (Deutsche Rentenversicherung Knappschaft-Bahn-See, [§ 40a Abs. 6 Satz 1 EStG][estg40a]) | 3740 |
| Erstattung nach dem Aufwendungsausgleichsgesetz | von der Krankenkasse | 4972 Erstattungen Aufwendungsausgleichsgesetz |

Die Zuordnung am Bankumsatz macht buchfink-bank. Beitragslastschriften der
Krankenkassen wiederholen sich monatlich; `bankumsaetze` schlägt sie als
Vorschlag der Art `regel` vor. Sie bleiben Rückfrage.

## 4. Sonderzeilen im Journal

Jede dieser Zeilen lässt die Auszahlung vom Netto abweichen. Lege sie dem
Menschen vor; die Umbuchung nach dem Lohnjournal erfasst du mit
`buchung_erfassen` (Rückfrage), Beleg ist das Journal.

| Zeile | Umbuchung nach dem Journal | Hinweis |
|---|---|---|
| Vorschuss, früher ausgezahlt | 3720 an 1340 Forderungen gegen Personal aus Lohn- und Gehaltsabrechnung | Den Vorschuss selbst bucht buchfink-bank auf 1340 |
| Pfändung, Abtretung | 3720 an 3725 Verbindlichkeiten für Einbehaltungen von Arbeitnehmern | Zahlung an den Gläubiger gegen 3725 |
| Sparbetrag vermögenswirksame Leistung | 3720 an 3725 | Zahlung an das Anlageinstitut gegen 3725; der Arbeitgeberzuschuss steckt im Brutto (6080 Vermögenswirksame Leistungen) |
| Sachbezug, Firmenwagen (Nettoabzug) | 3720 an Ertrag für Sachbezüge (z. B. 4947 Verrechnete sonstige Sachbezüge aus Fahrzeug-Gestellung 19 % USt, 4949 ohne Umsatzsteuer) | Wertungsfrage, siehe unten |
| Steuerfreie Erstattung (Reisekosten) über die Abrechnung | Aufwand (z. B. 6650 Reisekosten Arbeitnehmer) an 3720 | Belege der Reise bucht buchfink-belege; frag, ob sie schon gebucht sind |

Sachbezug und Firmenwagen:

- Der geldwerte Vorteil ist Arbeitslohn ([§ 8 Abs. 2 EStG][estg8]); das
  Lohnbüro rechnet ihn ins steuerpflichtige Brutto und zieht ihn vom Netto
  wieder ab.
- Überlässt die GmbH einem Beschäftigten, auch einem
  Gesellschafter-Geschäftsführer mit Arbeitsvertrag, ein Fahrzeug zur privaten
  Nutzung, ist das in der Regel eine entgeltliche sonstige Leistung, die
  Umsatzsteuer auslöst ([UStAE 15.23 Abs. 8 Satz 1, 4][ustae1523]).
  Bemessungsgrundlage und Satz klärt der Mensch mit Lohnbüro oder
  Steuerberater; Buchfink rechnet sie nicht.
- Ohne Regelung im Anstellungsvertrag ist die private Nutzung durch einen
  beherrschenden Gesellschafter vGA-verdächtig; lade buchfink-gesellschafter.

## 5. Was Buchfink nicht abbildet

- Lohnabrechnung, Lohnsteuer-Anmeldung, Beitragsnachweise, Meldungen zur
  Sozialversicherung, elektronische Lohnsteuerbescheinigung.
- Kurzarbeitergeld, Altersteilzeit, betriebliche Altersversorgung mit
  Pauschalsteuer (6147), Entgeltumwandlung: Rückfrage, im Zweifel Grenze.
- Personalrückstellungen (Urlaub, Überstunden) zum Stichtag:
  buchfink-jahresabschluss.

[estg8]: https://www.gesetze-im-internet.de/estg/__8.html
[estg40]: https://www.gesetze-im-internet.de/estg/__40.html
[estg40a]: https://www.gesetze-im-internet.de/estg/__40a.html
[estg41a]: https://www.gesetze-im-internet.de/estg/__41a.html
[sgb4_28e]: https://www.gesetze-im-internet.de/sgb_4/__28e.html
[sgb3_358]: https://www.gesetze-im-internet.de/sgb_3/__358.html
[aag7]: https://www.gesetze-im-internet.de/aufag/__7.html
[ustae1523]: https://www.steuerschroeder.de/steuergesetze/ustae/15.23
