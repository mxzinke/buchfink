# Abstimmung der Lohnkonten

Rechtsstand: 17. Oktober 2026. Konten nach SKR04-Bezeichnung; maßgeblich sind
die Konten, die `lohnjournal_buchen` bebucht.

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
| 3759, 3790 | null, falls die Antwort sie nutzt |

Zeitversatz ist normal:

- Die Lohnsteuer eines Monats wird bis zum 10. Tag nach Ablauf des
  Anmeldungszeitraums angemeldet und gezahlt ([§ 41a Abs. 1 Satz 1 EStG][estg41a]).
  Am Monatsende steht sie deshalb noch auf 3730.
- Sozialversicherungsbeiträge sind am drittletzten Bankarbeitstag des
  Beschäftigungsmonats in voraussichtlicher Höhe fällig, ein Rest im Folgemonat;
  statt der voraussichtlichen Höhe darf der Arbeitgeber den Betrag des Vormonats
  zahlen ([§ 23 Abs. 1 Satz 2, 3 SGB IV][sgb4_23]). Die Zahlung geht oft vor der
  Lohnbuchung ab; 3740 steht dann bis zur Buchung im Soll.

## 2. Abgleich mit Anmeldung und Beitragsnachweis

- **Lohnsteuer-Anmeldung:** Summe aus Lohnsteuer, pauschaler Lohnsteuer (außer
  Minijob), Solidaritätszuschlag und Kirchensteuer des Anmeldungszeitraums
  gleich Haben auf 3730 für diesen Zeitraum. Lege die Anmeldung als Beleg ab.
  Anmeldungszeitraum: Monat; Quartal bei Vorjahreslohnsteuer über 1.080 € bis
  5.000 €; Jahr bis 1.080 € ([§ 41a Abs. 2 EStG][estg41a]).
- **Beitragsnachweise:** Summe aller Nachweise der Krankenkassen und der
  Minijob-Zentrale gleich Haben auf 3740 für den Monat. Der Nachweis geht zwei
  Arbeitstage vor Fälligkeit an die Einzugsstelle
  ([§ 28f Abs. 3 Satz 1 SGB IV][sgb4_28f]); fehlt er, kann sie schätzen.
- **Bank:** Jede Zahlung an Finanzamt, Krankenkasse und Minijob-Zentrale ist
  einem Monat zugeordnet. Prüfe offene Bankumsätze mit `bankumsaetze`.

## 3. Differenzen und ihre Ursachen

| Befund | Häufige Ursache | Vorgehen |
|---|---|---|
| 3720 Rest nach Auszahlung | Sonderzeile (Vorschuss, Pfändung, Sachbezug) nicht umgebucht | SKILL.md, Abschnitt 2 |
| 3720 negativ | Auszahlung doppelt zugeordnet oder Vorschuss auf 3720 gebucht | Bankzuordnung prüfen, Storno nach Rückfrage |
| 3730 weicht von der Anmeldung ab | Korrektur eines Vormonats im aktuellen Journal; Anmeldung berichtigt | Lohnbüro fragen, Beleg ablegen |
| 3740 bleibt im Soll | Beitrag gezahlt, Journal noch nicht gebucht | Lohnjournal buchen |
| 3740 Rest im Haben | Restbeitrag fällig im Folgemonat, Lastschrift fehlt | Fälligkeit prüfen, Mensch informieren |
| Säumniszuschlag im Kontoauszug | Zahlung zu spät ([§ 240 AO][ao240]) | Buchung regelt buchfink-bank; Ursache mit dem Menschen klären |

Korrigierst du eine Lohnbuchung, gilt der Storno-Ablauf aus
buchfink-grundlagen. Ändert sich dadurch eine übermittelte Anmeldung, berichtigt
das Lohnbüro sie.

## 4. Jahresende

1. Lohnjournal Dezember gebucht, alle Zahlungen bis Jahresende zugeordnet.
2. Jahressumme der Lohnjournale gleich Jahressumme der Aufwandskonten, die
   die Lohnbuchungen bebucht haben (`auswertung`, `art: susa`). Weicht
   sie ab, finde den Monat.
3. 3730 zum 31.12. gleich der Lohnsteuer-Anmeldung Dezember (bzw. des
   letzten Quartals oder des Jahres), fällig am 10. Januar.
4. 3740 zum 31.12. gleich dem erklärten Restbeitrag.
5. Personalrückstellungen (Urlaub, Überstunden, Berufsgenossenschaft) und
   Abgrenzungen regelt buchfink-jahresabschluss.

## 5. Haftung und Aufbewahrung

- Der Arbeitgeber haftet für die Lohnsteuer, die er einzubehalten und
  abzuführen hat ([§ 42d Abs. 1 Nr. 1 EStG][estg42d]); der Geschäftsführer bei
  vorsätzlicher oder grob fahrlässiger Pflichtverletzung
  ([§ 34 Abs. 1][ao34], [§ 69 AO][ao69]).
- Wer als Arbeitgeber Arbeitnehmerbeiträge zur Sozialversicherung der
  Einzugsstelle vorenthält, macht sich strafbar ([§ 266a Abs. 1 StGB][stgb266a]).
- Das Lohnjournal ist Buchungsbeleg; Fristen für Buchungsbelege:
  buchfink-grundlagen. Lohnkonten: bis zum Ablauf des sechsten Kalenderjahres
  nach der letzten Lohnzahlung ([§ 41 Abs. 1 Satz 9 EStG][estg41]).
  Entgeltunterlagen und Beitragsabrechnungen: bis zum Ablauf des
  Kalenderjahres nach der letzten Prüfung der Rentenversicherung
  ([§ 28f Abs. 1 Satz 1, 4 SGB IV][sgb4_28f]).
- Bei einer Außenprüfung umfasst der Datenzugriff auch die Lohnbuchhaltung
  ([GoBD Rz. 159][gobd]). Die Daten liegen beim Lohnbüro; der Mensch sorgt dafür,
  dass sie verfügbar sind.

[estg41]: https://www.gesetze-im-internet.de/estg/__41.html
[estg41a]: https://www.gesetze-im-internet.de/estg/__41a.html
[estg42d]: https://www.gesetze-im-internet.de/estg/__42d.html
[sgb4_23]: https://www.gesetze-im-internet.de/sgb_4/__23.html
[sgb4_28f]: https://www.gesetze-im-internet.de/sgb_4/__28f.html
[ao34]: https://www.gesetze-im-internet.de/ao_1977/__34.html
[ao69]: https://www.gesetze-im-internet.de/ao_1977/__69.html
[ao240]: https://www.gesetze-im-internet.de/ao_1977/__240.html
[stgb266a]: https://www.gesetze-im-internet.de/stgb/__266a.html
[gobd]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf
