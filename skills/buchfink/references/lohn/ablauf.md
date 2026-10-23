# Ablauf: Lohn

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Buchfink rechnet keine Löhne ab; Abrechnung, Lohnsteuer-Anmeldung,
Beitragsnachweise und Meldungen macht das Lohnbüro oder eine Lohnsoftware.
Buchfink bucht die Monatssummen aus deren Lohnjournal und hält die
Verbindlichkeiten bis zur Zahlung nach; die Pflichten aus
[§ 41a EStG][estg41a] und [§ 28e Abs. 1 SGB IV][sgb4_28e] bleiben beim
Arbeitgeber.

Löhne buchst du nur über das Lohnjournal; ein zusätzlich mit
`buchung_erfassen` gebuchtes Gehalt stünde doppelt im Aufwand.

## Inhalt

1. Lohnjournal buchen
2. Prüfungen vor der Buchung
3. Fristen
4. Sonderfälle
5. Nur der Mensch

## 1. Lohnjournal buchen

1. **Ablegen.** Das vollständige Lohnjournal, nicht nur die Summenseite, mit
   `beleg_ablegen`, dazu Beitragsnachweise und Lohnsteuer-Anmeldung, falls
   vorhanden. Das Journal ist der Beleg der Summenbuchung
   ([grundlagen/gobd.md](../grundlagen/gobd.md#3-vollständigkeit)). Fertig,
   wenn das Lohnjournal eine `beleg_id` hat.
2. **Summen lesen** in die Felder nach
   [lohnjournal.md, Abschnitt 1](lohnjournal.md#1-die-summenfelder). Fertig,
   wenn jedes Feld einen Wert aus dem Journal oder ausdrücklich 0 hat.
3. **Prüfen** nach Abschnitt 2. Fertig, wenn die Netto-Probe aufgeht oder
   jede Differenz einen benannten Grund hat.
4. **Rückfrage.** Lege vor: Monat, Summen, Probe, Abweichung zum Vormonat,
   Sonderzeilen, erwartete Zahlungen mit Fälligkeit (Abschnitt 3).
5. **Buchen** nach dem Ja mit `lohnjournal_buchen`. Fertig, wenn die Antwort
   die `buchung` enthält und du ihre Zeilen gegen die Summen geprüft hast.
6. **Zahlungen** an Beschäftigte, Finanzamt, Krankenkassen und
   Minijob-Zentrale ordnest du nach [bank/zuordnung.md](../bank/zuordnung.md)
   zu, Gegenkonten nach
   [lohnjournal.md, Abschnitt 3](lohnjournal.md#3-gegenkonten-der-zahlungen).
7. **Abstimmen** nach [abstimmung.md](abstimmung.md). Fertig, wenn die
   Verbindlichkeitskonten des Monats nach den Zahlungen null zeigen oder jede
   Restgröße erklärt ist.

## 2. Prüfungen vor der Buchung

- **Netto-Probe** nach
  [lohnjournal.md, Abschnitt 1](lohnjournal.md#1-die-summenfelder). Geht sie
  nicht auf, stehen meist Sonderzeilen im Journal
  ([lohnjournal.md, Abschnitt 4](lohnjournal.md#4-sonderzeilen-im-journal));
  frag den Menschen.
- **Vergleich** mit der Lohnbuchung des Vormonats (`buchungen_suchen`): Weicht
  ein Feld um mehr als ein Zehntel ab, nenne den Grund oder frag danach
  (Neueinstellung, Austritt, Einmalzahlung, Krankheit).
- **Berichtigtes Journal** für einen gebuchten Monat: Storno und Neubuchung
  nach Rückfrage
  ([grundlagen/arbeitsweise.md](../grundlagen/arbeitsweise.md#11-eine-buchung-korrigieren)).
  Korrigiert das Lohnbüro im Folgemonat, buchst du nur den Folgemonat. Eine
  übermittelte Anmeldung berichtigt das Lohnbüro.

## 3. Fristen

Lohnsteuer-Anmeldung:
[grundlagen/pflichten.md#lohnsteuer-anmeldung](../grundlagen/pflichten.md#lohnsteuer-anmeldung).
Die Sozialversicherung führt `fristen` nicht:

| Was | Frist | Grundlage |
|---|---|---|
| Sozialversicherungsbeiträge | drittletzter Bankarbeitstag des Beschäftigungsmonats in voraussichtlicher Höhe (oder Betrag des Vormonats), Rest zum drittletzten Bankarbeitstag des Folgemonats | [§ 23 Abs. 1 Satz 2, 3 SGB IV][sgb4_23] |
| Beitragsnachweis an die Krankenkasse | zwei Arbeitstage vor Fälligkeit | [§ 28f Abs. 3 Satz 1 SGB IV][sgb4_28f] |

Melde fehlende Zahlungen sofort dem Menschen; Haftung:
[abstimmung.md, Abschnitt 5](abstimmung.md#5-haftung-und-aufbewahrung).

## 4. Sonderfälle

- **Minijob:** Pauschalabgaben und die einheitliche Pauschsteuer von 2 % gehen
  an die Minijob-Zentrale ([§ 40a Abs. 2, 6 EStG][estg40a]). Die
  Geringfügigkeitsgrenze folgt dem Mindestlohn
  ([§ 8 Abs. 1a SGB IV][sgb4_8]): 2026 13,90 € × 130 ÷ 3, aufgerundet 603 €
  im Monat ([MiLoV5][milov5]; 2027 mit 14,60 €: 633 €); maßgeblich ist die
  Bekanntgabe des BMAS. Überschreitet ein Minijob die Grenze, frag das
  Lohnbüro.
- **Geschäftsführer:** Lohn wie jeder andere. Für einen geschäftsführenden
  Gesellschafter beantragt die Einzugsstelle die Statusfeststellung bei der
  Deutschen Rentenversicherung ([§ 7a Abs. 1 Satz 2 SGB IV][sgb4_7a]); du
  übernimmst, was das Journal ausweist. Zahlungen über den Arbeitsvertrag
  hinaus prüfst du nach
  [gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md).
- **Sachbezug, Firmenwagen:**
  [lohnjournal.md, Abschnitt 4](lohnjournal.md#4-sonderzeilen-im-journal).
- **Erstattungen der Krankenkasse** (U1, U2) sind Ertrag ([§ 1 AAG][aag1]);
  Gegenkonto in
  [lohnjournal.md, Abschnitt 3](lohnjournal.md#3-gegenkonten-der-zahlungen).
- **Beitrag zur Berufsgenossenschaft:** jährlicher Bescheid, Beleg nach
  [belege/ablauf.md](../belege/ablauf.md), Konto 6120.

## 5. Nur der Mensch

- Lohnbüro beauftragen, Abrechnungsdaten liefern, Lohnsteuer-Anmeldung und
  Beitragsnachweise freigeben; ohne Lohnbüro die Lohnsteuer-Anmeldung in Mein
  ELSTER senden.
- Arbeitsverträge, Gehaltsänderungen, Statusfeststellung.
- Dafür sorgen, dass das Lohnbüro Lohnkonten und Entgeltunterlagen für die
  Fristen aus
  [abstimmung.md, Abschnitt 5](abstimmung.md#5-haftung-und-aufbewahrung)
  aufbewahrt oder übergibt.

[estg40a]: https://www.gesetze-im-internet.de/estg/__40a.html
[estg41a]: https://www.gesetze-im-internet.de/estg/__41a.html
[sgb4_7a]: https://www.gesetze-im-internet.de/sgb_4/__7a.html
[sgb4_8]: https://www.gesetze-im-internet.de/sgb_4/__8.html
[sgb4_23]: https://www.gesetze-im-internet.de/sgb_4/__23.html
[sgb4_28e]: https://www.gesetze-im-internet.de/sgb_4/__28e.html
[sgb4_28f]: https://www.gesetze-im-internet.de/sgb_4/__28f.html
[aag1]: https://www.gesetze-im-internet.de/aufag/__1.html
[milov5]: https://www.gesetze-im-internet.de/milov5/__1.html
