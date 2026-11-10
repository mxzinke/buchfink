# Abstimmung der Lohnkonten

Rechtsstand: 17. Oktober 2026. Maßgeblich sind die Konten, die
`lohnjournal_buchen` bebucht.

## Inhalt

1. Monatliche Abstimmung
2. Abgleich mit Anmeldung und Beitragsnachweis
3. Differenzen und ihre Ursachen
4. Jahresende
5. Haftung und Aufbewahrung

## 1. Monatliche Abstimmung

Prüfe nach dem Buchen und nach den Zahlungen mit `auswertung`
(`art: kontoblatt`, `konto`, `bis` Monatsende):

| Konto | Erwarteter Stand nach allen Zahlungen des Monats |
|---|---|
| 3720 Verbindlichkeiten aus Lohn und Gehalt | null |
| 3725 Verbindlichkeiten für Einbehaltungen von Arbeitnehmern | null, sobald Pfändungen und Sparbeträge weitergeleitet sind |
| 3730 Verbindlichkeiten aus Lohn- und Kirchensteuer | Betrag der noch nicht fälligen Lohnsteuer-Anmeldung, sonst null |
| 3740 Verbindlichkeiten im Rahmen der sozialen Sicherheit | null oder ein erklärter Restbeitrag für den Folgemonat |
| 1340 Forderungen gegen Personal aus Lohn- und Gehaltsabrechnung | offene Vorschüsse |
| 3790 Lohn- und Gehaltsverrechnungskonto | null, falls die Lohnbuchung es nutzt |

Zeitversatz ist normal: Die Lohnsteuer wird nach Ablauf des
Anmeldungszeitraums gezahlt
([grundlagen/pflichten.md#lohnsteuer-anmeldung](../grundlagen/pflichten.md#lohnsteuer-anmeldung))
und steht am Monatsende noch auf 3730. Sozialversicherungsbeiträge sind im
Beschäftigungsmonat fällig ([ablauf.md](ablauf.md#3-fristen)); geht die
Zahlung vor der Lohnbuchung ab, steht 3740 bis dahin im Soll.

## 2. Abgleich mit Anmeldung und Beitragsnachweis

- **Lohnsteuer-Anmeldung:** Lohnsteuer, pauschale Lohnsteuer (außer
  Minijob), Solidaritätszuschlag und Kirchensteuer des Anmeldungszeitraums
  ergeben das Haben auf 3730 für diesen Zeitraum. Lege die Anmeldung als Beleg
  ab.
- **Beitragsnachweise:** Die Nachweise der Krankenkassen und der
  Minijob-Zentrale ergeben das Haben auf 3740 für den Monat. Fehlt ein
  Nachweis, kann die Einzugsstelle schätzen ([§ 28f Abs. 3 SGB IV][sgb4_28f]).
- **Bank:** Jede Zahlung an Finanzamt, Krankenkasse und Minijob-Zentrale ist
  einem Monat zugeordnet; offene Bankumsätze zeigt `bankumsaetze`.

## 3. Differenzen und ihre Ursachen

| Befund | Häufige Ursache | Vorgehen |
|---|---|---|
| 3720 Rest nach Auszahlung | Sonderzeile (Vorschuss, Pfändung, Sachbezug) nicht umgebucht | [lohnjournal.md](lohnjournal.md#4-sonderzeilen-im-journal) |
| 3720 negativ | Auszahlung doppelt zugeordnet oder Vorschuss auf 3720 gebucht | Bankzuordnung prüfen, Storno nach Rückfrage |
| 3730 weicht von der Anmeldung ab | Korrektur eines Vormonats im aktuellen Journal; Anmeldung berichtigt | Lohnbüro fragen, Beleg ablegen |
| 3740 bleibt im Soll | Beitrag gezahlt, Journal noch nicht gebucht | Lohnjournal buchen |
| 3740 Rest im Haben | Restbeitrag fällig im Folgemonat, Lastschrift fehlt | Fälligkeit prüfen, Mensch informieren |
| Säumniszuschlag im Kontoauszug | Zahlung zu spät ([§ 240 AO][ao240]) | Buchung nach [bank/steuerzahlungen.md](../bank/steuerzahlungen.md#4-säumniszuschläge-zinsen-verspätungszuschläge); Ursache mit dem Menschen klären |

Korrigierst du eine Lohnbuchung, gilt
[ablauf.md, Abschnitt 2](ablauf.md#2-prüfungen-vor-der-buchung).

## 4. Jahresende

1. Lohnjournal Dezember gebucht, alle Zahlungen bis Jahresende zugeordnet.
2. Jahressumme der Lohnjournale gleich Jahressumme der bebuchten
   Aufwandskonten (`auswertung`, `art: susa`); sonst finde den Monat.
3. 3730 zum 31.12. gleich der letzten Lohnsteuer-Anmeldung des Jahres,
   fällig am 10. Januar.
4. 3740 zum 31.12. gleich dem erklärten Restbeitrag.
5. Personalrückstellungen (Urlaub, Überstunden, Berufsgenossenschaft):
   [jahresabschluss/rueckstellungen.md](../jahresabschluss/rueckstellungen.md);
   Abgrenzungen:
   [jahresabschluss/abgrenzung.md](../jahresabschluss/abgrenzung.md).

## 5. Haftung und Aufbewahrung

- Der Arbeitgeber haftet für die Lohnsteuer, die er einzubehalten und
  abzuführen hat ([§ 42d Abs. 1 Nr. 1 EStG][estg42d]); der Geschäftsführer bei
  vorsätzlicher oder grob fahrlässiger Pflichtverletzung
  ([§ 34 Abs. 1][ao34], [§ 69 AO][ao69]).
- Vorenthaltene Arbeitnehmerbeiträge zur Sozialversicherung sind strafbar
  ([§ 266a Abs. 1 StGB][stgb266a]).
- Aufbewahrung: Lohnjournal als Buchungsbeleg nach
  [grundlagen/aufbewahrung.md](../grundlagen/aufbewahrung.md); Lohnkonten bis
  zum Ablauf des sechsten Kalenderjahres nach der letzten Lohnzahlung
  ([§ 41 Abs. 1 Satz 9 EStG][estg41]); Entgeltunterlagen und
  Beitragsabrechnungen bis zum Ablauf des Kalenderjahres nach der letzten
  Prüfung der Rentenversicherung ([§ 28f Abs. 1 Satz 1, 4 SGB IV][sgb4_28f]).
- Der Datenzugriff einer Außenprüfung umfasst die Lohnbuchhaltung
  ([GoBD Rz. 159][gobd]); der Mensch sorgt dafür, dass das Lohnbüro die Daten
  bereitstellt.

[estg41]: https://www.gesetze-im-internet.de/estg/__41.html
[estg42d]: https://www.gesetze-im-internet.de/estg/__42d.html
[sgb4_28f]: https://www.gesetze-im-internet.de/sgb_4/__28f.html
[ao34]: https://www.gesetze-im-internet.de/ao_1977/__34.html
[ao69]: https://www.gesetze-im-internet.de/ao_1977/__69.html
[ao240]: https://www.gesetze-im-internet.de/ao_1977/__240.html
[stgb266a]: https://www.gesetze-im-internet.de/stgb/__266a.html
[gobd]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf
