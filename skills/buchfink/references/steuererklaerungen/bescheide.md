# Bescheide, Einspruch, Vorauszahlungen

Rechtsstand: 17. Oktober 2026. Wortlaut von AO, EStG, KStG und GewStG am
17.10.2026 gelesen.

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

Stand: Oktober 2026. Ab diesem Tag laufen Einspruchs- und Zahlungsfristen.
Geht ein Bescheid später zu, gilt der spätere Tag; im Zweifel muss die Behörde
den Zugang nachweisen.

- Lege jeden Bescheid am Tag des Eingangs mit `beleg_ablegen` ab und notiere
  Datum der Aufgabe, Bereitstellung oder Absendung.
- Bescheide im ELSTER-Postfach lädt der Mensch als PDF herunter und speichert
  sie in den Eingang; du legst sie dann ab.

## 2. Abgleich mit der Erklärung

Gleiche jeden Bescheid Zeile für Zeile mit der freigegebenen Erklärung ab.

| Prüfpunkt | Worauf achten |
|---|---|
| Bemessungsgrundlage | Einkommen, Gewerbeertrag, Umsätze wie erklärt? Erläuterungen im Bescheid lesen |
| festgesetzte Steuer | Satz und Rundung |
| Anrechnung | Vorauszahlungen und Kapitalertragsteuer vollständig? |
| Nebenleistungen | Verspätungszuschlag ([§ 152 AO][ao152]), Zinsen ([§ 233a AO][ao233a]) |
| Vorbehalt der Nachprüfung | Festsetzung bleibt änderbar; Änderung kann jederzeit beantragt werden ([§ 164 Abs. 1, 2 AO][ao164]) |
| Vorläufigkeit | Umfang und Grund müssen angegeben sein ([§ 165 Abs. 1 Satz 3 AO][ao165]) |
| Feststellungen | Verlustvortrag ([§ 10d Abs. 4 EStG][estg10d]), Gewerbeverlust ([§ 10a Satz 6 GewStG][gewstg10a]), Einlagekonto ([§ 27 Abs. 2 Satz 1 KStG][kstg27]): Werte für das Folgejahr notieren |
| Zahlung | Betrag, Fälligkeit, Konto des Finanzamts oder der Gemeinde |

Fertig, wenn jede Abweichung einen Grund hat oder als offene Frage beim
Menschen liegt.

Eine Umsatzsteuer-Jahreserklärung wirkt ohne Bescheid wie eine Festsetzung
unter Vorbehalt der Nachprüfung; ergibt sie eine Erstattung, erst mit
Zustimmung des Finanzamts ([§ 168 AO][ao168]). Ein Bescheid kommt nur, wenn
das Finanzamt abweicht.

## 3. Gewerbesteuer: zwei Bescheide

- Das Finanzamt setzt den Messbetrag fest; der Messbescheid bindet die
  Gemeinde ([§ 184 Abs. 1 Satz 4 AO][ao184] mit
  [§ 182 Abs. 1 AO][ao182]).
- Die Gemeinde setzt mit ihrem Hebesatz die Steuer fest
  ([§ 16 Abs. 1 GewStG][gewstg16]).
- Einwände gegen den Gewerbeertrag richten sich gegen den Messbescheid des
  Finanzamts; im Bescheid der Gemeinde lassen sie sich nicht angreifen
  ([§ 351 Abs. 2 AO][ao351]). Den Rechtsbehelf gegen den Bescheid der Gemeinde
  nennt dessen Rechtsbehelfsbelehrung.

## 4. Einspruch

- Frist: ein Monat nach Bekanntgabe; bei einer Steueranmeldung ein Monat nach
  ihrem Eingang beim Finanzamt, bei Zustimmung nach § 168 Satz 2 AO ein Monat
  nach Bekanntwerden der Zustimmung ([§ 355 Abs. 1 AO][ao355]).
- Der Einspruch hält die Zahlung nicht auf ([§ 361 Abs. 1 Satz 1 AO][ao361]).
  Auf Antrag soll die Vollziehung ausgesetzt werden, wenn ernstliche Zweifel an
  der Rechtmäßigkeit bestehen ([§ 361 Abs. 2 Satz 2 AO][ao361]).
- Wertungsfrage: Einspruch oder nicht. Nenne Abweichung, Betrag, Fundstelle und
  Frist. Der Mensch entscheidet und legt ihn ein; du entwirfst den Text.
  Bei größeren Beträgen oder Rechtsfragen empfiehl einen Steuerberater.
- Ist die Frist verstrichen, ändert das Finanzamt nur noch nach den
  Änderungsvorschriften, etwa bei neuen Tatsachen ([§ 173 Abs. 1 AO][ao173]).
  Das prüft ein Steuerberater.

## 5. Vorauszahlungsbescheide

| Steuer | Termine | Höhe und Anpassung | Grundlage |
|---|---|---|---|
| Körperschaftsteuer, Solidaritätszuschlag | 10.03., 10.06., 10.09., 10.12. | grundsätzlich nach der letzten Veranlagung; Anpassung bis zum Ablauf des 15. Monats nach dem Jahr | [§ 37 Abs. 1, 3 EStG][estg37], [§ 31 Abs. 1 KStG][kstg31] |
| Gewerbesteuer | 15.02., 15.05., 15.08., 15.11. | je ein Viertel der letzten Veranlagung; Anpassung bis zum Ende des 15. Monats nach dem Erhebungszeitraum | [§ 19 Abs. 1–3 GewStG][gewstg19] |

Stand: Oktober 2026.

- `fristen` führt die Termine als Pflichten `kst-vorauszahlung` und
  `gewst-vorauszahlung`, sobald der Vorauszahlungsbescheid als Anlass erfasst
  ist oder Vorauszahlungen gebucht sind. Die Höhe steht nur im Bescheid. Geht
  ein neuer Bescheid ein, lege ihn ab und erfasse ihn nach Rückfrage mit
  `anlass_erfassen` (`art: vorauszahlungsbescheid-kst` oder
  `vorauszahlungsbescheid-gewst`, `datum` des Bescheids).
- Weicht der erwartete Gewinn deutlich ab, kann der Mensch beim Finanzamt
  oder der Gemeinde eine Anpassung anregen ([§ 37 Abs. 3 Satz 3 EStG][estg37],
  [§ 19 Abs. 3 GewStG][gewstg19]). Liegt der Gewinn deutlich über den
  Vorauszahlungen, weise auf die spätere Nachzahlung und auf Zinsen nach
  [§ 233a AO][ao233a] hin.

## 6. Buchung

Zahlung, Erstattung und die Differenz zur gebuchten Rückstellung bucht
buchfink-bank (Konten je Steuerart, Vorjahre, Nebenleistungen). Ist der
Abschluss des Jahres noch offen und weicht der Bescheid von der Rückstellung
ab, sag es buchfink-jahresabschluss.

[ao122]: https://www.gesetze-im-internet.de/ao_1977/__122.html
[ao122a]: https://www.gesetze-im-internet.de/ao_1977/__122a.html
[ao152]: https://www.gesetze-im-internet.de/ao_1977/__152.html
[ao164]: https://www.gesetze-im-internet.de/ao_1977/__164.html
[ao165]: https://www.gesetze-im-internet.de/ao_1977/__165.html
[ao168]: https://www.gesetze-im-internet.de/ao_1977/__168.html
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
