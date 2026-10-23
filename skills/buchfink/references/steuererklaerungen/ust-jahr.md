# Umsatzsteuer-Jahreserklärung

Rechtsstand: 17. Oktober 2026.

Frist, Weg, Nachweis und Folgen:
[grundlagen/pflichten.md#ust-erklaerung](../grundlagen/pflichten.md#ust-erklaerung).

## Inhalt

1. Pflicht, Zeitraum, Frist
2. Abgleich mit den Voranmeldungen
3. Abschlusszahlung und Erstattung
4. Abgabe in Mein ELSTER
5. Fehler nach der Abgabe

## 1. Pflicht, Zeitraum, Frist

- Steueranmeldung für das Kalenderjahr mit selbst berechneter Steuer
  ([§ 18 Abs. 3 Satz 1 UStG][ustg18]), auch bei abweichendem Wirtschaftsjahr;
  bei Tätigkeit nur in einem Teil des Jahres für diesen Teil
  ([§ 16 Abs. 1 Satz 2, Abs. 3 UStG][ustg16]).
- Fristtag: `fristen` (Art `ust-erklaerung`). Endet die Tätigkeit im Jahr, gilt
  die Monatsfrist nach Ablauf des kürzeren Zeitraums
  ([grundlagen/pflichten.md#ust-erklaerung](../grundlagen/pflichten.md#ust-erklaerung)).
- Gründungsjahr: Die Verwaltung lässt die Regelfrist gelten
  ([UStAE 18.1 Abs. 4 Satz 2][ustae181]), der Wortlaut von § 18 Abs. 3 Satz 2
  UStG erfasst auch diesen Fall. Sag das dem Menschen und empfiehl frühe
  Abgabe; im Zweifel klärt er die Frist mit dem Finanzamt.

## 2. Abgleich mit den Voranmeldungen

1. `erklaerung` (`art: ust_jahr`, `zeitraum: <Jahr>`).
2. Jede Voranmeldung des Jahres (`art: ustva`) hat den Status `uebermittelt`.
3. Je Kennzahl Summe der Voranmeldungen gegen Jahreswert; jede Differenz
   braucht einen Grund, typisch:
   - Buchung in einen übermittelten Zeitraum ohne berichtigte Voranmeldung
     ([monatsabschluss/berichtigung.md](../monatsabschluss/berichtigung.md));
   - Vorsteuerberichtigung nach § 15a UStG bis 6.000 € je Wirtschaftsgut und
     Jahr, erst in der Jahreserklärung ([§ 44 Abs. 3 Satz 1 UStDV][ustdv44]);
   - Rundung auf volle Euro bei Bemessungsgrundlagen.
4. Die Sondervorauszahlung ist in der letzten Voranmeldung schon angerechnet
   ([monatsabschluss/dauerfristverlaengerung.md](../monatsabschluss/dauerfristverlaengerung.md#4-anrechnung-im-dezember));
   nicht doppelt anrechnen.
5. Zahllast gegen die Konten: gezahlte Vorauszahlungen auf 3820 und offene
   Umsatzsteuer (`auswertung`, `art: susa`).

Fertig, wenn jede Kennzahl erklärt ist und Abschlusszahlung oder Erstattung
feststeht.

## 3. Abschlusszahlung und Erstattung

- Fälligkeit der Nachzahlung:
  [grundlagen/pflichten.md#ust-erklaerung](../grundlagen/pflichten.md#ust-erklaerung);
  bei abweichender Festsetzung ist der Unterschied einen Monat nach
  Bekanntgabe fällig ([§ 18 Abs. 4 Satz 2 UStG][ustg18]).
- Die Erklärung wirkt wie eine Festsetzung unter Vorbehalt der Nachprüfung;
  eine Erstattung erst mit Zustimmung des Finanzamts
  ([§ 168 Satz 1, 2 AO][ao168]).
- Buchung: [bank/steuerzahlungen.md](../bank/steuerzahlungen.md#2-konten-je-steuerart)
  (Vorjahr 3841, Erstattung 1420).

## 4. Abgabe in Mein ELSTER

Nach [abgabe.md](abgabe.md#4-eingabe-in-mein-elster); gleiche danach die von
Mein ELSTER berechnete Zahllast oder Erstattung mit `erklaerung` ab.

## 5. Fehler nach der Abgabe

- Anzeige- und Berichtigungspflicht:
  [monatsabschluss/berichtigung.md](../monatsabschluss/berichtigung.md#2-pflicht-und-recht-zur-berichtigung).
- Bücher per Storno korrigieren
  ([grundlagen/arbeitsweise.md](../grundlagen/arbeitsweise.md#11-eine-buchung-korrigieren)),
  berichtigte Jahreserklärung vorlegen; Voranmeldungen:
  [monatsabschluss/berichtigung.md](../monatsabschluss/berichtigung.md#4-nach-der-jahreserklärung).

[ao168]: https://www.gesetze-im-internet.de/ao_1977/__168.html
[ustg16]: https://www.gesetze-im-internet.de/ustg_1980/__16.html
[ustg18]: https://www.gesetze-im-internet.de/ustg_1980/__18.html
[ustdv44]: https://www.gesetze-im-internet.de/ustdv_1980/__44.html
[ustae181]: https://datenbank.nwb.de/Dokument/378652_18___1/
