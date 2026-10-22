# Bescheide, Einspruch, Vorauszahlungen

Rechtsstand: 17. Oktober 2026.

## Inhalt

1. Eingang und Bekanntgabe
2. Abgleich mit der Erklärung
3. Gewerbesteuer: zwei Bescheide
4. Einspruch
5. Vorauszahlungsbescheide
6. Buchung

## 1. Eingang und Bekanntgabe

| Weg | gilt als bekannt gegeben | Grundlage |
|---|---|---|
| Post im Inland | am vierten Tag nach Aufgabe zur Post | [§ 122 Abs. 2 Nr. 1 AO][ao122] |
| elektronisch übermittelt | am vierten Tag nach der Absendung | [§ 122 Abs. 2a AO][ao122] |
| zum Abruf bereitgestellt (ELSTER-Postfach) | am vierten Tag nach der Bereitstellung | [§ 122a Abs. 4 Satz 1 AO][ao122a] |

Ab diesem Tag laufen Einspruchs- und Zahlungsfristen; bei späterem Zugang gilt
der spätere Tag.

Leg jeden Bescheid am Eingangstag mit `beleg_ablegen` ab und notiere das Datum
von Aufgabe, Bereitstellung oder Absendung. Bescheide im ELSTER-Postfach
speichert der Mensch als PDF in den Eingang.

## 2. Abgleich mit der Erklärung

Gleiche jeden Bescheid Zeile für Zeile mit der freigegebenen Erklärung ab.

| Prüfpunkt | Worauf achten |
|---|---|
| Bemessungsgrundlage | Einkommen, Gewerbeertrag, Umsätze wie erklärt? Erläuterungen lesen |
| festgesetzte Steuer | Satz und Rundung |
| Anrechnung | Vorauszahlungen und Kapitalertragsteuer vollständig? |
| Nebenleistungen | Verspätungszuschlag ([§ 152 AO][ao152]), Zinsen ([§ 233a AO][ao233a]) |
| Vorbehalt der Nachprüfung | Festsetzung bleibt änderbar, Änderung jederzeit beantragbar ([§ 164 Abs. 1, 2 AO][ao164]) |
| Vorläufigkeit | Umfang und Grund müssen angegeben sein ([§ 165 Abs. 1 Satz 3 AO][ao165]) |
| Feststellungen | Verlustvortrag ([§ 10d Abs. 4 EStG][estg10d]), Gewerbeverlust ([§ 10a Satz 6 GewStG][gewstg10a]), Einlagekonto ([§ 27 Abs. 2 Satz 1 KStG][kstg27]): für das Folgejahr notieren |
| Zahlung | Betrag, Fälligkeit, Konto von Finanzamt oder Gemeinde |

Fertig, wenn jede Abweichung einen Grund hat oder als Frage beim Menschen
liegt.

Umsatzsteuer: Ein Bescheid kommt nur bei Abweichung
([ust-jahr.md](ust-jahr.md#3-abschlusszahlung-und-erstattung)).

## 3. Gewerbesteuer: zwei Bescheide

- Der Messbescheid des Finanzamts bindet die Gemeinde
  ([§ 184 Abs. 1 Satz 4 AO][ao184] mit [§ 182 Abs. 1 AO][ao182]); die Gemeinde
  setzt mit ihrem Hebesatz die Steuer fest ([§ 16 Abs. 1 GewStG][gewstg16]).
- Einwände gegen den Gewerbeertrag richten sich gegen den Messbescheid
  ([§ 351 Abs. 2 AO][ao351]); den Rechtsbehelf gegen den Bescheid der Gemeinde
  nennt dessen Belehrung.

## 4. Einspruch

- Ein Monat nach Bekanntgabe; bei einer Steueranmeldung nach Eingang, bei
  Zustimmung nach § 168 Satz 2 AO nach Bekanntwerden der Zustimmung
  ([§ 355 Abs. 1 AO][ao355]).
- Der Einspruch hält die Zahlung nicht auf ([§ 361 Abs. 1 Satz 1 AO][ao361]);
  Aussetzung der Vollziehung auf Antrag bei ernstlichen Zweifeln
  ([§ 361 Abs. 2 Satz 2 AO][ao361]).
- Wertungsfrage: Nenne Abweichung, Betrag, Fundstelle und Frist; der Mensch
  entscheidet und legt ein, du entwirfst den Text. Bei größeren Beträgen oder
  Rechtsfragen Steuerberater.
- Nach Fristablauf nur Änderung nach den Änderungsvorschriften, etwa neue
  Tatsachen ([§ 173 Abs. 1 AO][ao173]); das prüft ein Steuerberater.

## 5. Vorauszahlungsbescheide

| Steuer | Höhe und Anpassung | Grundlage |
|---|---|---|
| Körperschaftsteuer, Solidaritätszuschlag | nach der letzten Veranlagung; Anpassung bis zum Ablauf des 15. Monats nach dem Jahr | [§ 37 Abs. 1, 3 EStG][estg37], [§ 31 Abs. 1 KStG][kstg31] |
| Gewerbesteuer | je ein Viertel der letzten Veranlagung; Anpassung bis zum Ende des 15. Monats nach dem Erhebungszeitraum | [§ 19 Abs. 1–3 GewStG][gewstg19] |

Termine: [kst-vorauszahlung](../grundlagen/pflichten.md#kst-vorauszahlung),
[gewst-vorauszahlung](../grundlagen/pflichten.md#gewst-vorauszahlung).

- Die Höhe steht nur im Bescheid. Einen neuen legst du ab und erfasst ihn nach
  Rückfrage als Anlass
  ([grundlagen/pflichten.md](../grundlagen/pflichten.md#arbeiten-mit-fristen)).
- Weicht der erwartete Gewinn deutlich ab, kann der Mensch eine Anpassung
  anregen ([§ 37 Abs. 3 Satz 3 EStG][estg37], [§ 19 Abs. 3 GewStG][gewstg19]);
  bei deutlich höherem Gewinn weise auf Nachzahlung und Zinsen
  ([§ 233a AO][ao233a]) hin.

## 6. Buchung

Zahlung, Erstattung und Differenz zur Rückstellung:
[bank/steuerzahlungen.md](../bank/steuerzahlungen.md). Ist der Abschluss noch
offen und weicht der Bescheid von der Rückstellung ab:
[jahresabschluss/steuern-im-abschluss.md](../jahresabschluss/steuern-im-abschluss.md).

[ao122]: https://www.gesetze-im-internet.de/ao_1977/__122.html
[ao122a]: https://www.gesetze-im-internet.de/ao_1977/__122a.html
[ao152]: https://www.gesetze-im-internet.de/ao_1977/__152.html
[ao164]: https://www.gesetze-im-internet.de/ao_1977/__164.html
[ao165]: https://www.gesetze-im-internet.de/ao_1977/__165.html
[ao173]: https://www.gesetze-im-internet.de/ao_1977/__173.html
[ao182]: https://www.gesetze-im-internet.de/ao_1977/__182.html
[ao184]: https://www.gesetze-im-internet.de/ao_1977/__184.html
[ao233a]: https://www.gesetze-im-internet.de/ao_1977/__233a.html
[ao351]: https://www.gesetze-im-internet.de/ao_1977/__351.html
[ao355]: https://www.gesetze-im-internet.de/ao_1977/__355.html
[ao361]: https://www.gesetze-im-internet.de/ao_1977/__361.html
[estg10d]: https://www.gesetze-im-internet.de/estg/__10d.html
[estg37]: https://www.gesetze-im-internet.de/estg/__37.html
[gewstg10a]: https://www.gesetze-im-internet.de/gewstg/__10a.html
[gewstg16]: https://www.gesetze-im-internet.de/gewstg/__16.html
[gewstg19]: https://www.gesetze-im-internet.de/gewstg/__19.html
[kstg27]: https://www.gesetze-im-internet.de/kstg_1977/__27.html
[kstg31]: https://www.gesetze-im-internet.de/kstg_1977/__31.html
