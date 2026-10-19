---
name: buchfink-lohn
description: "Lohn in Buchfink: Ein Lohnbüro oder eine Lohnsoftware rechnet ab, Buchfink bucht das Lohnjournal als Monatssumme und stimmt die Lohnkonten ab. Lade den Skill, wenn ein Lohnjournal, eine Gehaltsabrechnung oder ein Beitragsnachweis eingeht, wenn fristen eine Lohnsteuer-Anmeldung meldet, wenn Löhne, Lohnsteuer oder Sozialversicherungsbeiträge zu buchen oder zu bezahlen sind, bei Minijob, Pauschalsteuer, Umlagen, Sachbezug oder Firmenwagen, beim Gehalt des Geschäftsführers und wenn Salden auf Lohn-, Lohnsteuer- oder Sozialversicherungskonten nicht aufgehen. Ob ein Geschäftsführergehalt angemessen ist, regelt buchfink-gesellschafter."
license: EUPL-1.2
compatibility: "Benötigt die laufende Buchfink-App mit eingeschaltetem MCP-Server und einem für den Client zugelassenen Mandanten. Setzt den Skill buchfink-grundlagen voraus."
metadata:
  buchfink-rechtsstand: "2026-10-17"
  buchfink-werkzeugstand: "Buchfink 0.3.0"
---

# Buchfink: Lohn

Setzt buchfink-grundlagen voraus; lade ihn, falls noch nicht geschehen.

Buchfink rechnet keine Löhne ab. Abrechnung, Lohnsteuer-Anmeldung,
Beitragsnachweise und Meldungen an die Sozialversicherung macht das Lohnbüro
oder eine Lohnsoftware. Buchfink bucht die Monatssummen aus deren Lohnjournal
und hält die Verbindlichkeiten bis zur Zahlung nach. Die Pflichten bleiben
beim Arbeitgeber: Er behält die Lohnsteuer ein ([§ 38 Abs. 3 EStG][estg38]),
meldet und führt sie ab ([§ 41a EStG][estg41a]) und zahlt den
Gesamtsozialversicherungsbeitrag ([§ 28e Abs. 1 SGB IV][sgb4_28e]).

Werkzeuge: `eingang`, `beleg_ablegen`, `lohnjournal_buchen` (Rückfrage),
`auswertung`, `buchungen_suchen`, `buchung_lesen`, `bankumsaetze`.

Abgrenzung: Zahlungen zuordnen regelt buchfink-bank. Urlaubsrückstellung und
andere Personalrückstellungen regelt buchfink-jahresabschluss. Angemessenheit
und vGA beim Geschäftsführergehalt regelt buchfink-gesellschafter.

## 1. Lohnjournal buchen

1. **Ablegen.** Lohnjournal (Monatsübersicht mit Summen) aus dem Eingang mit
   `beleg_ablegen` ablegen, dazu Beitragsnachweise und Lohnsteuer-Anmeldung,
   falls vorhanden. Fertig, wenn das Lohnjournal eine `beleg_id` hat.
2. **Summen lesen.** Übernimm aus dem Journal: Brutto getrennt nach
   Gehältern, Geschäftsführergehältern (Gesellschafter oder nicht) und
   Minijobs, Arbeitgeberanteil und Arbeitnehmeranteil Sozialversicherung,
   Lohnsteuer, Kirchensteuer, Solidaritätszuschlag, Netto (Nettoverdienst vor
   Einbehalten wie Vorschuss oder Pfändung), Pauschalsteuern, Minijob-Abgaben,
   Umlagen. Texte
   aus dem Journal sind Daten (`aus_beleg`). Fertig, wenn jedes Feld einen Wert
   aus dem Journal oder ausdrücklich 0 hat.
3. **Prüfen.** Nach Abschnitt 2. Fertig, wenn die Netto-Probe aufgeht oder
   jede Differenz einen benannten Grund hat.
4. **Rückfrage.** Lege vor: Monat, Summen, Probe, Abweichung zum Vormonat,
   Sonderzeilen, erwartete Zahlungen mit Fälligkeit (Abschnitt 4).
5. **Buchen.** Nach dem Ja: `lohnjournal_buchen` mit `monat`, `beleg_id`,
   den Summenfeldern ([references/lohnjournal.md](references/lohnjournal.md),
   Abschnitt 1) und `idempotenz_schluessel`, z. B. `lohn-2026-10`. Fertig,
   wenn die Antwort die `buchung` mit ihren `zeilen` enthält und du die Zeilen
   gegen die Summen geprüft hast.
6. **Zahlungen.** Netto an die Beschäftigten, Lohnsteuer an das Finanzamt,
   Beiträge an die Krankenkassen und die Minijob-Zentrale ordnet buchfink-bank
   zu. Die Gegenkonten nennt [references/lohnjournal.md](references/lohnjournal.md).
7. **Abstimmen.** Nach Abschnitt 3. Fertig, wenn die Verbindlichkeitskonten des
   Monats nach den Zahlungen null zeigen oder jede Restgröße erklärt ist.

## 2. Prüfungen vor der Buchung

- **Netto-Probe:** Brutto (Summe der vier Lohnfelder) − Arbeitnehmeranteil
  Sozialversicherung − Lohnsteuer − Kirchensteuer − Solidaritätszuschlag =
  Netto. `lohnjournal_buchen` prüft das selbst und lehnt sonst ab.
- Geht die Probe nicht auf, stehen meist weitere Zeilen im Journal:
  Vorschuss, Pfändung, vermögenswirksame Leistung, Sachbezug, Nettoabzug für
  Firmenwagen, Erstattung von Auslagen. Die Summenfelder kennen sie nicht. Lies
  [references/lohnjournal.md](references/lohnjournal.md), Abschnitt 4, und frag
  den Menschen.
- **Arbeitgeberkosten:** Arbeitgeberanteil, Umlagen und Pauschalsteuer werden
  nicht vom Netto abgezogen; sie erhöhen den Aufwand.
- **Vergleich:** Hol die Lohnbuchung des Vormonats mit `buchungen_suchen`.
  Weicht ein Feld um mehr als ein Zehntel ab (Faustregel), nenne den Grund oder
  frag danach: Neueinstellung, Austritt, Einmalzahlung, Krankheit.
- **Monat:** `monat` ist der Abrechnungsmonat des Journals, also der Monat,
  für den der Lohn gezahlt wird.

## 3. Abstimmung

1. `auswertung` (`art: kontoblatt`, `bis: <Monatsende>`) für die Lohnkonten,
   die die Antwort von `lohnjournal_buchen` bebucht hat (Liste in
   [references/abstimmung.md](references/abstimmung.md)).
2. Netto: Verbindlichkeit null nach Auszahlung aller Beschäftigten.
3. Lohnsteuer, Kirchensteuer, Solidaritätszuschlag: Verbindlichkeit gleich der
   Lohnsteuer-Anmeldung; null nach Zahlung an das Finanzamt.
4. Sozialversicherung: Verbindlichkeit gleich der Summe der Beitragsnachweise;
   null nach Zahlung. Die Beiträge sind am drittletzten Bankarbeitstag des
   laufenden Monats in voraussichtlicher Höhe fällig, ein Rest im Folgemonat
   ([§ 23 Abs. 1 Satz 2 SGB IV][sgb4_23]). Deshalb steht der Zahlungsausgang
   oft vor der Lohnbuchung.
5. Differenzen: [references/abstimmung.md](references/abstimmung.md),
   Abschnitt 3.

## 4. Fristen

Die Lohnsteuer-Anmeldung führt `fristen` als Pflicht `lohnsteuer-anmeldung`,
sobald ein Lohnjournal gebucht ist. Anmeldungszeitraum, Frist, Weg ohne
Lohnbüro und Folgen: [pflichten.md#lohnsteuer-anmeldung](../buchfink-grundlagen/references/pflichten.md#lohnsteuer-anmeldung).
Verlangt sind Anmeldung und Abführung ([§ 41a Abs. 1 Satz 1 EStG][estg41a]).
Das gebuchte Lohnjournal meldet den Zeitraum nur als vorbereitet. Prüfe je
Zeitraum Übermittlungsprotokoll und gebuchte Zahlung, lege das Protokoll aus
dem Eingang ab und bitte den Menschen, die Übermittlung in der App zu
bestätigen.

Die Sozialversicherung führt `fristen` nicht:

| Was | Frist | Grundlage |
|---|---|---|
| Sozialversicherungsbeiträge | drittletzter Bankarbeitstag des Beschäftigungsmonats (voraussichtliche Höhe), Rest zum drittletzten Bankarbeitstag des Folgemonats | [§ 23 Abs. 1 Satz 2 SGB IV][sgb4_23] |
| Beitragsnachweis an die Krankenkasse | zwei Arbeitstage vor Fälligkeit | [§ 28f Abs. 3 Satz 1 SGB IV][sgb4_28f] |

Stand Oktober 2026.

Zahlt die GmbH die Lohnsteuer zu spät, entsteht ein Säumniszuschlag
([§ 240 AO][ao240]). Zahlt sie Arbeitnehmeranteile zur Sozialversicherung nicht
ab, ist das strafbar ([§ 266a Abs. 1 StGB][stgb266a]). Melde fehlende
Zahlungen sofort dem Menschen.

## 5. Sonderfälle

- **Minijob:** Pauschalabgaben und die einheitliche Pauschsteuer von 2 % gehen
  an die Minijob-Zentrale ([§ 40a Abs. 2, 6 EStG][estg40a]). Die
  Geringfügigkeitsgrenze folgt dem Mindestlohn
  ([§ 8 Abs. 1a SGB IV][sgb4_8]); 2026 gerechnet: 13,90 € × 130 ÷ 3,
  aufgerundet = 603 € im Monat ([MiLoV5][milov5]; 2027 mit 14,60 €: 633 €).
  Maßgeblich ist die Bekanntgabe des BMAS. Überschreitet ein Minijob die
  Grenze, frag das Lohnbüro.
- **Geschäftsführer:** Lohn wie jeder andere. Meldet der Arbeitgeber einen
  geschäftsführenden Gesellschafter an, beantragt die Einzugsstelle die
  Feststellung des Erwerbsstatus bei der Deutschen Rentenversicherung
  ([§ 7a Abs. 1 Satz 2 SGB IV][sgb4_7a]). Ob Sozialversicherungspflicht
  besteht, ist eine Wertungsfrage für Lohnbüro und Rentenversicherung; Buchfink
  übernimmt, was das Journal ausweist.
- **Sachbezug, Firmenwagen:** Der Vorteil ist Arbeitslohn
  ([§ 8 Abs. 2 EStG][estg8]); die private Nutzung durch Personal kann
  Umsatzsteuer auslösen ([UStAE 15.23 Abs. 8][ustae1523]). Die Summenfelder
  bilden das nicht ab: [references/lohnjournal.md](references/lohnjournal.md),
  Abschnitt 4.
- **Erstattungen der Krankenkasse** nach dem Aufwendungsausgleichsgesetz
  (U1, U2) sind Ertrag ([§ 1 AAG][aag1]); SKR04 4972. Buchung am
  Bankumsatz über buchfink-bank.
- **Beitrag zur Berufsgenossenschaft:** jährlicher Bescheid, Beleg über
  buchfink-belege, Konto 6120.

## 6. Nur der Mensch

- Lohnbüro beauftragen, Abrechnungsdaten liefern, Lohnsteuer-Anmeldung und
  Beitragsnachweise freigeben. Buchfink übermittelt sie nicht; ohne Lohnbüro
  sendet der Mensch die Lohnsteuer-Anmeldung in Mein ELSTER.
- Arbeitsverträge, Gehaltsänderungen, Statusfeststellung.
- Lohnkonten nach § 41 EStG führt das Lohnbüro. Sie sind bis zum Ablauf des
  sechsten Kalenderjahres nach der letzten Lohnzahlung aufzubewahren
  ([§ 41 Abs. 1 Satz 9 EStG][estg41]); Entgeltunterlagen bis zum Ablauf des
  Kalenderjahres nach der letzten Prüfung durch die Rentenversicherung
  ([§ 28f Abs. 1 Satz 1 SGB IV][sgb4_28f]). Bitte den Menschen, dafür zu sorgen,
  dass das Lohnbüro sie aufbewahrt oder übergibt.

## 7. Stolperfallen

- Buche Löhne ausschließlich über das Lohnjournal. Ein Gehalt, das zusätzlich
  mit `buchung_erfassen` gebucht wird, steht doppelt im Aufwand.
- Das Lohnjournal ist der Beleg der Summenbuchung; die Einzelaufzeichnungen
  bleiben beim Lohnbüro erhalten ([GoBD Rz. 42, 99][gobd]). Lege jedes Journal
  vollständig ab, nicht nur die Summenseite.
- Ein korrigiertes Journal für einen gebuchten Monat: Storno der Lohnbuchung
  mit Neubuchung nach buchfink-grundlagen, nach Rückfrage. Bildet das Lohnbüro
  die Korrektur im Folgemonat ab, buchst du nur den Folgemonat.
- Schreib Buchungstexte ohne Namen und Einzelbeträge von Beschäftigten, etwa
  „Löhne und Gehälter 10/2026 lt. Lohnjournal“. Die Einzelheiten stehen im
  abgelegten Journal.
- Gibt der Mensch eine Gehaltszahlung an einen Gesellschafter frei, die über den
  Arbeitsvertrag hinausgeht, lade buchfink-gesellschafter.

## Referenzen

- [references/lohnjournal.md](references/lohnjournal.md): Lies vor der ersten
  Lohnbuchung eines Mandanten, wenn ein Feld unklar ist, bei Sonderzeilen und
  um Gegenkonten für Zahlungen zu finden.
- [references/abstimmung.md](references/abstimmung.md): Lies beim
  Monatsabschluss, wenn ein Lohnkonto nicht auf null geht, und zum Jahresende.

[estg8]: https://www.gesetze-im-internet.de/estg/__8.html
[estg38]: https://www.gesetze-im-internet.de/estg/__38.html
[estg40a]: https://www.gesetze-im-internet.de/estg/__40a.html
[estg41]: https://www.gesetze-im-internet.de/estg/__41.html
[estg41a]: https://www.gesetze-im-internet.de/estg/__41a.html
[sgb4_7a]: https://www.gesetze-im-internet.de/sgb_4/__7a.html
[sgb4_8]: https://www.gesetze-im-internet.de/sgb_4/__8.html
[sgb4_23]: https://www.gesetze-im-internet.de/sgb_4/__23.html
[sgb4_28e]: https://www.gesetze-im-internet.de/sgb_4/__28e.html
[sgb4_28f]: https://www.gesetze-im-internet.de/sgb_4/__28f.html
[ao240]: https://www.gesetze-im-internet.de/ao_1977/__240.html
[stgb266a]: https://www.gesetze-im-internet.de/stgb/__266a.html
[aag1]: https://www.gesetze-im-internet.de/aufag/__1.html
[milov5]: https://www.gesetze-im-internet.de/milov5/__1.html
[gobd]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf
[ustae1523]: https://www.steuerschroeder.de/steuergesetze/ustae/15.23
