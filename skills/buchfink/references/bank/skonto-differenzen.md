# Skonto und andere Zahlungsdifferenzen

Rechtsstand: 17. Oktober 2026. Wortlaut von UStG, HGB und GewStG am 17.10.2026
gelesen.

## Inhalt

1. Skonto
2. Gebühr
3. Rundung
4. Kursdifferenz
5. Kürzung durch den Kunden
6. Kleine Differenzen ausbuchen

## 1. Skonto

**Umsatzsteuer.** Skonto mindert die Bemessungsgrundlage. Der Leistende
berichtigt seine Steuer, der Empfänger seine Vorsteuer
([§ 17 Abs. 1 Satz 1 und 2 UStG][ustg17]), und zwar im Zeitraum, in dem die
Minderung eintritt, also mit der Zahlung (Satz 8). Bei § 13b und beim
innergemeinschaftlichen Erwerb gilt das sinngemäß (Satz 5): Beide Steuerzeilen
werden berichtigt. Eine Rechnungsberichtigung ist dafür nicht nötig
([belege/vorsteuer.md](../belege/vorsteuer.md#7-minderung-des-entgelts)).

**Was Buchfink bucht.** Mit `differenz {art: skonto, betrag}` (Bruttobetrag des
Skontos):

| Steuerfall des Postens | Aufteilung des Skontos | Konto |
|---|---|---|
| Inland | Netto und Steuer nach dem Satz der Rechnung | erhalten 5736 (19 %), 5731 (7 %); gewährt 4736 (19 %), 4731 (7 %) |
| § 13b, innergemeinschaftlicher Erwerb | ganzes Skonto ist Netto; beide Steuerzeilen werden berichtigt | 5730 bzw. 4734 |
| steuerfrei, nicht steuerbar, Nullsteuersatz | keine Steuer | 5730 bzw. 4734 |

Beispiel: 2 % Skonto auf 1.190,00 € brutto sind 23,80 €, davon 20,00 € Netto
und 3,80 € Steuer.

**Anlage.** Auf eine Anlagenrechnung mindert das Skonto die Anschaffungskosten
([§ 255 Abs. 1 Satz 3 HGB][hgb255]). Buchfink bucht es dann auf das
Anlagekonto und schreibt das Anlagenverzeichnis fort.

**Prüfen vor dem Vorschlag.**

- Steht ein Skonto auf der Rechnung (Prozentsatz, Frist)? Sonst ist es keine
  vereinbarte Minderung.
- Wurde innerhalb der Frist gezahlt (Bankdatum)?
- Ausgangsseite: Hat der Kunde Skonto gezogen, obwohl die Frist abgelaufen war,
  ist der Rest eine offene Forderung. Ob die GmbH ihn verlangt, entscheidet der
  Mensch (Wertungsfrage); Mahnung in
  [rechnungen/mahnen-verzug.md](../rechnungen/mahnen-verzug.md).
- Ausgangsseite, Gewerbesteuer: Skonti, die nicht dem gewöhnlichen
  Geschäftsverkehr entsprechen, gelten als Entgelt für Schulden
  ([§ 8 Nr. 1 Buchst. a Satz 2 GewStG][gewstg8]). Bei ungewöhnlich hohen
  gewährten Skonti weise darauf hin.

## 2. Gebühr

- **Zahlungsausgang:** Die Bank belastet Rechnungsbetrag plus Gebühr in einem
  Umsatz. `differenz {art: gebuehr}` bucht die Gebühr auf 6855 Nebenkosten des
  Geldverkehrs.
- **Zahlungseingang:** Eine Bank in der Kette hat vom Betrag eine Gebühr
  abgezogen. `differenz {art: gebuehr}` ist im Code für den Ausgang gebaut und
  würde hier falsch buchen. Schreib zwei Zeilen: `buchung_id` mit dem vollen
  Betrag als `ausgleich` und `konto: 6855` mit der Gebühr als negativem
  `betrag`.
- Kontoführung und Überweisungsentgelt sind steuerfrei
  ([§ 4 Nr. 8 Buchst. d UStG][ustg4]); die Zeilen einer Aufteilung tragen
  keine Umsatzsteuer.

## 3. Rundung

`differenz {art: rundung}` für Cent-Beträge ohne wirtschaftlichen Grund. Buchfink
bucht auf 6300 (Aufwand) oder 4830 (Ertrag). Rundung ist nie der Weg für eine
bewusste Kürzung (Abschnitt 5).

## 4. Kursdifferenz

Wurde eine Rechnung in Fremdwährung gebucht, weicht der Euro-Betrag der Zahlung
vom gebuchten Betrag ab. `differenz {art: waehrung}` bucht die realisierte
Differenz; Buchfink nimmt dafür 6300 bzw. 4830. Den Beleg in Fremdwährung
regelt [belege/fremdwaehrung-auslagen.md](../belege/fremdwaehrung-auslagen.md#1-beleg-in-fremdwährung),
die Bewertung offener Posten zum Stichtag
[jahresabschluss/forderungen.md](../jahresabschluss/forderungen.md).

## 5. Kürzung durch den Kunden

Zahlt ein Kunde bewusst weniger (Mängelrüge, Abzug ohne Vereinbarung), ist das
Teilzahlung oder Minderung des Entgelts.

- Teilzahlung: Rest bleibt offen, Rückfrage
  ([zuordnung.md](zuordnung.md#4-teilzahlung-sammelzahlung-überzahlung)).
- Vereinbarte Minderung: Sie ändert die Bemessungsgrundlage; Umsatzsteuer
  berichtigen ([§ 17 Abs. 1 Satz 1 UStG][ustg17]). Den Weg (Gutschrift,
  Rechnungskorrektur) regelt
  [rechnungen/storno-berichtigung.md](../rechnungen/storno-berichtigung.md).

## 6. Kleine Differenzen ausbuchen

Es gibt keine gesetzliche Grenze, bis zu der Differenzen ausgebucht werden
dürfen. Jeder Vorgang ist richtig abzubilden ([GoBD Rz. 44][gobd]).
Wertungsfrage: Der Mensch entscheidet je Fall, ob eine Differenz verfolgt oder
ausgebucht wird. Nenne Betrag, Grund und Folge; schlage keine pauschale Grenze
vor.

[gobd]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf
[ustg4]: https://www.gesetze-im-internet.de/ustg_1980/__4.html
[ustg17]: https://www.gesetze-im-internet.de/ustg_1980/__17.html
[hgb255]: https://www.gesetze-im-internet.de/hgb/__255.html
[gewstg8]: https://www.gesetze-im-internet.de/gewstg/__8.html
