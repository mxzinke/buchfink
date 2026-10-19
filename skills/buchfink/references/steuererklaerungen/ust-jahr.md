# Umsatzsteuer-Jahreserklärung

Rechtsstand: 17. Oktober 2026. Wortlaut von UStG, UStDV und AO am 17.10.2026
gelesen. UStAE nicht amtlich gelesen, nur über Verlagsdatenbank und
Kanzleispiegel ([Quellenverzeichnis][quellen]).

Frist, Weg in Mein ELSTER, Nachweis und Folgen einer Versäumnis stehen in
[grundlagen/pflichten.md#ust-erklaerung](../grundlagen/pflichten.md#ust-erklaerung).

## Inhalt

1. Pflicht, Zeitraum, Frist
2. Abgleich mit den Voranmeldungen
3. Abschlusszahlung und Erstattung
4. Abgabe in Mein ELSTER
5. Fehler nach der Abgabe

## 1. Pflicht, Zeitraum, Frist

- Die GmbH übermittelt für das Kalenderjahr eine Steuererklärung, in der sie die
  Steuer oder den Überschuss selbst berechnet (Steueranmeldung,
  [§ 18 Abs. 3 Satz 1 UStG][ustg18]).
- Besteuerungszeitraum ist das Kalenderjahr, auch bei abweichendem
  Wirtschaftsjahr ([§ 16 Abs. 1 Satz 2 UStG][ustg16]). War die GmbH nur einen
  Teil des Jahres tätig, tritt dieser Teil an die Stelle des Kalenderjahres
  ([§ 16 Abs. 3 UStG][ustg16]).
- Den Fristtag nennt `fristen` (Art `ust-erklaerung`). Endet die Tätigkeit im
  Laufe des Jahres, gilt die Monatsfrist nach Ablauf des kürzeren Zeitraums
  ([grundlagen/pflichten.md#ust-erklaerung](../grundlagen/pflichten.md#ust-erklaerung)).
- Beginnt die Tätigkeit im Laufe des Jahres (Gründungsjahr), lässt die
  Verwaltung „abweichend von § 18 Abs. 3 Satz 2 UStG“ die Regelfrist gelten
  ([UStAE 18.1 Abs. 4 Satz 2][ustae181]). Der Gesetzeswortlaut erfasst auch
  diesen Fall. Sag das dem Menschen und empfiehl, die Erklärung für das
  Gründungsjahr früh abzugeben; im Zweifel klärt er die Frist mit dem
  Finanzamt.

## 2. Abgleich mit den Voranmeldungen

1. Hole die Jahreswerte: `erklaerung` (`art: ust_jahr`, `zeitraum: <Jahr>`).
2. Hole jede Voranmeldung des Jahres: `erklaerung` (`art: ustva`) je Monat oder
   Quartal. Prüfe, dass jede den Status `uebermittelt` hat.
3. Vergleiche je Kennzahl die Summe der Voranmeldungen mit dem Jahreswert.
   Jede Differenz braucht einen Grund. Typische Gründe:
   - Buchung in einen schon übermittelten Zeitraum, die keine berichtigte
     Voranmeldung ausgelöst hat
     ([monatsabschluss/berichtigung.md](../monatsabschluss/berichtigung.md)).
   - Berichtigung des Vorsteuerabzugs nach § 15a UStG bis 6.000 € je
     Wirtschaftsgut und Jahr: sie gehört erst in die Jahreserklärung
     ([§ 44 Abs. 3 Satz 1 UStDV][ustdv44]). Stand: Oktober 2026.
   - Rundung: In den Formularen stehen Bemessungsgrundlagen in vollen Euro.
4. Die Sondervorauszahlung aus der Dauerfristverlängerung ist schon in der
   letzten Voranmeldung des Jahres angerechnet
   ([monatsabschluss/dauerfristverlaengerung.md](../monatsabschluss/dauerfristverlaengerung.md#4-anrechnung-im-dezember)).
   Rechne sie nicht ein zweites Mal an.
5. Gleiche die Zahllast laut Jahreserklärung mit den Konten ab: Summe der
   gezahlten Vorauszahlungen auf 3820 und offene Umsatzsteuer (`auswertung`,
   `art: susa`).

Fertig, wenn jede Kennzahl erklärt ist und die Differenz zwischen
Jahressteuer und Summe der Vorauszahlungen als Abschlusszahlung oder
Erstattung feststeht.

## 3. Abschlusszahlung und Erstattung

- Wann eine Nachzahlung aus der Erklärung fällig ist, steht in
  [grundlagen/pflichten.md#ust-erklaerung](../grundlagen/pflichten.md#ust-erklaerung).
  Setzt das Finanzamt abweichend fest, ist der Unterschied einen Monat nach
  Bekanntgabe des Bescheids fällig ([§ 18 Abs. 4 Satz 2 UStG][ustg18]).
- Die Erklärung wirkt wie eine Festsetzung unter Vorbehalt der Nachprüfung.
  Ergibt sie eine Erstattung, gilt das erst mit Zustimmung des Finanzamts
  ([§ 168 Satz 1, 2 AO][ao168]).
- Zahlung und Erstattung buchst du nach
  [bank/steuerzahlungen.md](../bank/steuerzahlungen.md#2-konten-je-steuerart)
  (Vorjahr: 3841, Erstattung: 1420).

## 4. Abgabe in Mein ELSTER

- Formular „Umsatzsteuererklärung“ des Jahres; Ablauf der Eingabe in
  [abgabe.md](abgabe.md#4-eingabe-in-mein-elster). `erklaerung`
  (`art: ust_jahr`) zeigt Zeilen und Kennzahlen formularnah mit Herkunft und
  die Summe der Vorauszahlungen.
- Gleiche nach der Eingabe vor allem die von Mein ELSTER berechnete
  verbleibende Zahllast oder Erstattung mit `erklaerung` ab.

## 5. Fehler nach der Abgabe

- Für eine unrichtige Erklärung gilt die Anzeige- und Berichtigungspflicht
  aus [monatsabschluss/berichtigung.md](../monatsabschluss/berichtigung.md#2-pflicht-und-recht-zur-berichtigung).
- Korrigiere die Bücher per Storno
  ([grundlagen/arbeitsweise.md](../grundlagen/arbeitsweise.md#11-eine-buchung-korrigieren))
  und lege eine berichtigte Jahreserklärung vor. Ob zusätzlich Voranmeldungen
  zu berichtigen sind, klärt
  [monatsabschluss/berichtigung.md](../monatsabschluss/berichtigung.md#4-nach-der-jahreserklärung).

[quellen]: https://github.com/mxzinke/buchfink/blob/main/skills/QUELLEN.md
[ao168]: https://www.gesetze-im-internet.de/ao_1977/__168.html
[ustg16]: https://www.gesetze-im-internet.de/ustg_1980/__16.html
[ustg18]: https://www.gesetze-im-internet.de/ustg_1980/__18.html
[ustdv44]: https://www.gesetze-im-internet.de/ustdv_1980/__44.html
[ustae181]: https://datenbank.nwb.de/Dokument/378652_18___1/
