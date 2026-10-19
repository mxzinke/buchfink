# Ablauf: Lohn

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Buchfink rechnet keine Löhne ab. Abrechnung, Lohnsteuer-Anmeldung,
Beitragsnachweise und Meldungen an die Sozialversicherung macht das Lohnbüro
oder eine Lohnsoftware. Buchfink bucht die Monatssummen aus deren Lohnjournal
und hält die Verbindlichkeiten bis zur Zahlung nach. Die Pflichten bleiben
beim Arbeitgeber: Er behält die Lohnsteuer ein ([§ 38 Abs. 3 EStG][estg38]),
meldet und führt sie ab ([§ 41a EStG][estg41a]) und zahlt den
Gesamtsozialversicherungsbeitrag ([§ 28e Abs. 1 SGB IV][sgb4_28e]).

Werkzeuge: `eingang`, `beleg_ablegen`, `lohnjournal_buchen` (Rückfrage),
`auswertung`, `buchungen_suchen`, `buchung_lesen`, `bankumsaetze`.

Abgrenzung: Zahlungen zuordnen regelt [bank/zuordnung.md](../bank/zuordnung.md).
Urlaubsrückstellung und andere Personalrückstellungen regelt
[jahresabschluss/rueckstellungen.md](../jahresabschluss/rueckstellungen.md).
Angemessenheit und vGA beim Geschäftsführergehalt regelt
[gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md).

## Inhalt

1. Lohnjournal buchen
2. Prüfungen vor der Buchung
3. Abstimmung
4. Fristen
5. Sonderfälle
6. Nur der Mensch
7. Stolperfallen

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
   den Summenfeldern ([lohnjournal.md](lohnjournal.md#1-die-summenfelder))
   und `idempotenz_schluessel`, z. B. `lohn-2026-10`. Fertig,
   wenn die Antwort die `buchung` mit ihren `zeilen` enthält und du die Zeilen
   gegen die Summen geprüft hast.
6. **Zahlungen.** Netto an die Beschäftigten, Lohnsteuer an das Finanzamt,
   Beiträge an die Krankenkassen und die Minijob-Zentrale ordnest du nach
   [bank/zuordnung.md](../bank/zuordnung.md) zu. Die Gegenkonten nennt
   [lohnjournal.md](lohnjournal.md#3-gegenkonten-der-zahlungen).
7. **Abstimmen.** Nach Abschnitt 3. Fertig, wenn die Verbindlichkeitskonten des
   Monats nach den Zahlungen null zeigen oder jede Restgröße erklärt ist.

## 2. Prüfungen vor der Buchung

- **Netto-Probe** nach [lohnjournal.md](lohnjournal.md#1-die-summenfelder).
  `lohnjournal_buchen` prüft sie selbst und lehnt sonst ab.
- Geht die Probe nicht auf, stehen meist weitere Zeilen im Journal:
  Vorschuss, Pfändung, vermögenswirksame Leistung, Sachbezug, Nettoabzug für
  Firmenwagen, Erstattung von Auslagen. Die Summenfelder kennen sie nicht. Lies
  [lohnjournal.md](lohnjournal.md#4-sonderzeilen-im-journal) und frag
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
   die die Antwort von `lohnjournal_buchen` bebucht hat. Konten und erwartete
   Stände: [abstimmung.md](abstimmung.md#1-monatliche-abstimmung).
2. Gleiche Lohnsteuer mit der Anmeldung und Sozialversicherung mit den
   Beitragsnachweisen ab
   ([abstimmung.md](abstimmung.md#2-abgleich-mit-anmeldung-und-beitragsnachweis)).
   Weil die Beiträge vor Monatsende fällig sind (Abschnitt 4), steht der
   Zahlungsausgang oft vor der Lohnbuchung.
3. Differenzen:
   [abstimmung.md](abstimmung.md#3-differenzen-und-ihre-ursachen).

## 4. Fristen

Die Lohnsteuer-Anmeldung führt `fristen` als Pflicht `lohnsteuer-anmeldung`.
Anmeldungszeitraum, Frist, Weg ohne Lohnbüro, Nachweis und Folgen:
[grundlagen/pflichten.md#lohnsteuer-anmeldung](../grundlagen/pflichten.md#lohnsteuer-anmeldung).

Die Sozialversicherung führt `fristen` nicht:

| Was | Frist | Grundlage |
|---|---|---|
| Sozialversicherungsbeiträge | drittletzter Bankarbeitstag des Beschäftigungsmonats (voraussichtliche Höhe; statt dieser darf der Arbeitgeber den Betrag des Vormonats zahlen), Rest zum drittletzten Bankarbeitstag des Folgemonats | [§ 23 Abs. 1 Satz 2, 3 SGB IV][sgb4_23] |
| Beitragsnachweis an die Krankenkasse | zwei Arbeitstage vor Fälligkeit | [§ 28f Abs. 3 Satz 1 SGB IV][sgb4_28f] |

Stand Oktober 2026.

Melde fehlende Zahlungen sofort dem Menschen. Die Folgen stehen in
[grundlagen/pflichten.md#lohnsteuer-anmeldung](../grundlagen/pflichten.md#lohnsteuer-anmeldung)
(Lohnsteuer) und
[abstimmung.md](abstimmung.md#5-haftung-und-aufbewahrung)
(Sozialversicherung, Haftung).

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
- **Sachbezug, Firmenwagen:** Die Summenfelder bilden sie nicht ab; Regeln und
  Umbuchung in [lohnjournal.md](lohnjournal.md#4-sonderzeilen-im-journal).
- **Erstattungen der Krankenkasse** nach dem Aufwendungsausgleichsgesetz
  (U1, U2) sind Ertrag ([§ 1 AAG][aag1]); Gegenkonto in
  [lohnjournal.md](lohnjournal.md#3-gegenkonten-der-zahlungen), Buchung am
  Bankumsatz nach [bank/zuordnung.md](../bank/zuordnung.md).
- **Beitrag zur Berufsgenossenschaft:** jährlicher Bescheid, Beleg nach
  [belege/ablauf.md](../belege/ablauf.md), Konto 6120.

## 6. Nur der Mensch

Zu den Vorbehalten in
[SKILL.md](../../SKILL.md#4-nur-der-mensch-in-der-app) gehören hier:

- Lohnbüro beauftragen, Abrechnungsdaten liefern, Lohnsteuer-Anmeldung und
  Beitragsnachweise freigeben. Buchfink übermittelt sie nicht; ohne Lohnbüro
  sendet der Mensch die Lohnsteuer-Anmeldung in Mein ELSTER.
- Arbeitsverträge, Gehaltsänderungen, Statusfeststellung.
- Lohnkonten und Entgeltunterlagen führt das Lohnbüro. Bitte den Menschen,
  dafür zu sorgen, dass das Lohnbüro sie für die Fristen aus
  [abstimmung.md](abstimmung.md#5-haftung-und-aufbewahrung) aufbewahrt oder
  übergibt.

## 7. Stolperfallen

- Buche Löhne ausschließlich über das Lohnjournal. Ein Gehalt, das zusätzlich
  mit `buchung_erfassen` gebucht wird, steht doppelt im Aufwand.
- Das Lohnjournal ist der Beleg der Summenbuchung
  ([grundlagen/gobd.md](../grundlagen/gobd.md#3-vollständigkeit)). Lege jedes
  Journal vollständig ab, nicht nur die Summenseite.
- Ein korrigiertes Journal für einen gebuchten Monat: Storno der Lohnbuchung
  mit Neubuchung nach Rückfrage
  ([grundlagen/arbeitsweise.md](../grundlagen/arbeitsweise.md#11-eine-buchung-korrigieren)).
  Bildet das Lohnbüro die Korrektur im Folgemonat ab, buchst du nur den
  Folgemonat. Ändert sich dadurch eine übermittelte Anmeldung, berichtigt das
  Lohnbüro sie.
- Schreib Buchungstexte ohne Namen und Einzelbeträge von Beschäftigten, etwa
  „Löhne und Gehälter 10/2026 lt. Lohnjournal“. Die Einzelheiten stehen im
  abgelegten Journal.
- Gibt der Mensch eine Gehaltszahlung an einen Gesellschafter frei, die über den
  Arbeitsvertrag hinausgeht, prüfe sie nach
  [gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md).

[estg38]: https://www.gesetze-im-internet.de/estg/__38.html
[estg40a]: https://www.gesetze-im-internet.de/estg/__40a.html
[estg41a]: https://www.gesetze-im-internet.de/estg/__41a.html
[sgb4_7a]: https://www.gesetze-im-internet.de/sgb_4/__7a.html
[sgb4_8]: https://www.gesetze-im-internet.de/sgb_4/__8.html
[sgb4_23]: https://www.gesetze-im-internet.de/sgb_4/__23.html
[sgb4_28e]: https://www.gesetze-im-internet.de/sgb_4/__28e.html
[sgb4_28f]: https://www.gesetze-im-internet.de/sgb_4/__28f.html
[aag1]: https://www.gesetze-im-internet.de/aufag/__1.html
[milov5]: https://www.gesetze-im-internet.de/milov5/__1.html
