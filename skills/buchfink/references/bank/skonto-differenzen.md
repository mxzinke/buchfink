# Skonto und andere Zahlungsdifferenzen

Rechtsstand: 17. Oktober 2026. Wortlaut von UStG und HGB am 17.10.2026 gelesen.

## Inhalt

1. Skonto
2. Gebühr
3. Rundung
4. Kursdifferenz
5. Kürzung durch den Kunden
6. Kleine Differenzen ausbuchen

## 1. Skonto

Skonto mindert die Bemessungsgrundlage: Der Leistende berichtigt seine Steuer,
der Empfänger seine Vorsteuer ([§ 17 Abs. 1 Satz 1, 2 UStG][ustg17]), mit der
Zahlung als Zeitpunkt der Änderung
([monatsabschluss/berichtigung.md, Abschnitt 1](../monatsabschluss/berichtigung.md#1-berichtigung-oder-laufender-zeitraum)).
Bei § 13b und innergemeinschaftlichem Erwerb werden beide Steuerzeilen
berichtigt (Satz 5). Eine Rechnungsberichtigung ist nicht nötig
([belege/vorsteuer.md](../belege/vorsteuer.md#7-minderung-des-entgelts)).

`differenz {art: skonto, betrag}` nimmt das Skonto brutto (Beispiel in
[zuordnung.md](zuordnung.md#3-aufteilung-schreiben)). Buchfink bucht:

| Steuerfall des Postens | Aufteilung | Konto |
|---|---|---|
| Inland | Netto und Steuer nach dem Satz der Rechnung | erhalten 5736 (19 %), 5731 (7 %); gewährt 4736 (19 %), 4731 (7 %) |
| § 13b, innergemeinschaftlicher Erwerb | ganzes Skonto ist Netto; beide Steuerzeilen berichtigt | 5730 bzw. 4734 |
| steuerfrei, nicht steuerbar, Nullsteuersatz | keine Steuer | 5730 bzw. 4734 |

Auf eine Anlagenrechnung mindert Skonto die Anschaffungskosten
([§ 255 Abs. 1 Satz 3 HGB][hgb255]); Buchfink bucht es auf das Anlagekonto.

Vor dem Vorschlag prüfen:

- Steht ein Skonto mit Satz und Frist auf der Rechnung, und wurde laut
  Bankdatum innerhalb der Frist gezahlt?
- Ausgangsseite, Frist abgelaufen: Der Rest ist offene Forderung; ob die GmbH
  ihn verlangt, entscheidet der Mensch
  ([rechnungen/mahnen-verzug.md](../rechnungen/mahnen-verzug.md)).

## 2. Gebühr

Belastet die Bank Rechnungsbetrag plus Gebühr oder zieht eine Bank vom Eingang
eine Gebühr ab: `ausgleich` ist der volle Rechnungsbetrag,
`differenz {art: gebuehr}` die Gebühr; Buchfink bucht sie auf 6855.
Kontoführung und Überweisungsentgelt sind steuerfrei
([§ 4 Nr. 8 Buchst. d UStG][ustg4]).

## 3. Rundung

`differenz {art: rundung}` für Cent-Beträge ohne wirtschaftlichen Grund, auf
6300 (Aufwand) oder 4830 (Ertrag). Eine bewusste Kürzung ist keine Rundung
(Abschnitt 5).

## 4. Kursdifferenz

Bei einer in Fremdwährung gebuchten Rechnung bucht `differenz {art: waehrung}`
die realisierte Differenz auf 6880 bzw. 4840, gesondert ausgewiesen nach
§ 277 Abs. 5 Satz 2 HGB. Beleg in Fremdwährung:
[belege/fremdwaehrung-auslagen.md](../belege/fremdwaehrung-auslagen.md#1-beleg-in-fremdwährung);
Stichtag: [jahresabschluss/forderungen.md](../jahresabschluss/forderungen.md).

## 5. Kürzung durch den Kunden

Zahlt ein Kunde bewusst weniger (Mängelrüge, Abzug ohne Vereinbarung), ist das
Teilzahlung (Rest bleibt offen, Rückfrage;
[zuordnung.md](zuordnung.md#4-teilzahlung-sammelzahlung-überzahlung)) oder
vereinbarte Minderung mit Berichtigung der Umsatzsteuer
([§ 17 Abs. 1 Satz 1 UStG][ustg17]; Weg in
[rechnungen/storno-berichtigung.md](../rechnungen/storno-berichtigung.md)).

## 6. Kleine Differenzen ausbuchen

Eine gesetzliche Bagatellgrenze gibt es nicht; jeder Vorgang ist richtig
abzubilden ([GoBD Rz. 44][gobd]). Ob eine Differenz verfolgt oder ausgebucht
wird, entscheidet der Mensch je Fall (Wertungsfrage). Nenne Betrag, Grund und
Folge; eine pauschale Grenze schlägst du nicht vor.

[gobd]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf
[ustg4]: https://www.gesetze-im-internet.de/ustg_1980/__4.html
[ustg17]: https://www.gesetze-im-internet.de/ustg_1980/__17.html
[hgb255]: https://www.gesetze-im-internet.de/hgb/__255.html
