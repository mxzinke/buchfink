# Grenzen von Buchfink, Warnzeichen, Steuerberater

Rechtsstand: 17. Oktober 2026. Funktionsumfang von Buchfink 0.3.0. Lehnt ein
Werkzeug einen Fall mit `fachliche_regel` ab, gilt die Meldung des Servers.

## Inhalt

1. Was Buchfink nicht abbildet
2. Warnzeichen, die du sofort meldest
3. Wann du einen Steuerberater empfiehlst
4. Wie du eine Grenze ansprichst

## 1. Was Buchfink nicht abbildet

Was hier nicht buchbar ist, sprichst du nach Abschnitt 4 an.

| Fall | Was du tust |
|---|---|
| Andere Rechtsform als GmbH oder UG, Einnahmenüberschussrechnung; Kasse, Bargeschäfte mit Kassensystem, Warenlager; Konzernabschluss ([§ 290 HGB][hgb290]), IFRS, kapitalmarktorientierte Gesellschaft | Nicht abgebildet. |
| Differenzbesteuerung ([§ 25a UStG][ustg25a]), Reiseleistungen ([§ 25 UStG][ustg25]), Dreiecksgeschäft ([§ 25b UStG][ustg25b]), Verzicht auf Steuerbefreiungen ([§ 9 UStG][ustg9]), Organschaft ([§§ 14][kstg14], [17 KStG][kstg17], [§ 2 Abs. 2 Nr. 2 UStG][ustg2]) | Nicht abgebildet. Steuerberater. |
| Eigene Kleinunternehmerregelung ([§ 19 UStG][ustg19]) | Nicht abgebildet; Buchfink setzt Ausweis und Anmeldung der Umsatzsteuer voraus. Rechnungen von Kleinunternehmern lassen sich erfassen. |
| Istversteuerung ([§ 20 UStG][ustg20]) | Buchfink lehnt Buchungen ab; der Mensch klärt den Fall mit dem Steuerberater. |
| OSS und IOSS ([§§ 18i][ustg18i], [18j][ustg18j], [18k UStG][ustg18k]) | Meldung außerhalb von Buchfink. |
| Bauleistungen als Leistungsempfänger nach [§ 13b Abs. 2 Nr. 4 UStG][ustg13b] | Buchbar als `reverse_charge`. Ob die GmbH selbst nachhaltig Bauleistungen erbringt (Abs. 5 Satz 2), prüft Buchfink nicht; die Bauabzugsteuer fehlt. Steuerberater. |
| Einfuhrumsatzsteuer, unentgeltliche Wertabgaben | Nur teilweise abgebildet; vor dem Buchen nachfragen. |
| Gutschrift im Sinne von [§ 14 Abs. 2 Satz 5 UStG][ustg14] ausstellen | Nicht abgebildet; empfangene Gutschriften lassen sich ablegen. |
| Reine Holding ohne eigene Umsätze | Buchfink behandelt sie als Unternehmer mit Vorsteuerabzug ([§ 2 Abs. 1 UStG][ustg2]); ob das zutrifft, ist eine Wertungsfrage ([beteiligungen/holding-umsatzsteuer.md](../beteiligungen/holding-umsatzsteuer.md)). |
| Lohnabrechnung, Reisekostenabrechnung | Extern. Buchfink bucht die Summen aus dem Lohnjournal ([lohn/ablauf.md](../lohn/ablauf.md)). |
| Bankkonto in Fremdwährung, CAMT-Sammeleinträge mit mehreren Einzelumsätzen | Der Import lehnt sie ab. |
| Mittelgroße oder große Gesellschaft ([§ 267 HGB][hgb267]): Abschlussprüfung ([§ 316 HGB][hgb316]), latente Steuern, Lagebericht | Steuerberater und Wirtschaftsprüfer. |
| Eigene Steuerbilanz mit abweichenden Werten | Buchfink führt eine Einheitsbilanz; abweichende Steuerwerte nur bei Sonderabschreibungen nach § 7g EStG. |
| Ersetzendes Scannen | Papierbelege aufheben lassen ([aufbewahrung.md, Abschnitt 4](aufbewahrung.md#4-papierbelege-und-scannen)). |
| Versand von Rechnungen (E-Mail, Peppol, Portal) | Außerhalb von Buchfink. |

## 2. Warnzeichen, die du sofort meldest

Melde diese Funde unabhängig von deinem Auftrag, mit Zahlen und Fundstelle.
Die Bewertung trifft der Mensch mit Berater.

- **Hälfte des Stammkapitals verloren.** Zeigt eine Bilanz, auch eine
  unterjährige, ein Eigenkapital unter der Hälfte des Stammkapitals, beruft
  die Geschäftsführung einer GmbH unverzüglich die Gesellschafterversammlung
  ein ([§ 49 Abs. 3 GmbHG][gmbhg49]); bei der UG gilt das bei drohender
  Zahlungsunfähigkeit ([§ 5a Abs. 4 GmbHG][gmbhg5a]).
- **Negatives Eigenkapital oder Zahlungsschwierigkeiten.** Bei
  Zahlungsunfähigkeit oder Überschuldung beantragt die Geschäftsführung ohne
  schuldhaftes Zögern Insolvenz, spätestens drei Wochen nach Eintritt der
  Zahlungsunfähigkeit, sechs Wochen nach Eintritt der Überschuldung
  ([§ 15a Abs. 1 InsO][inso15a]). Negatives Eigenkapital ist noch keine
  Überschuldung; diese hängt auch von der Fortführungsprognose ab
  ([§ 19 Abs. 2 InsO][inso19]). Rate zu sofortiger rechtlicher Beratung.
- **Fehler in einer übermittelten Erklärung**, der zu einer zu niedrigen
  Steuer geführt haben kann:
  [monatsabschluss/berichtigung.md, Abschnitt 2](../monatsabschluss/berichtigung.md#2-pflicht-und-recht-zur-berichtigung).
- **Vorgänge, die nach Steuerverkürzung aussehen** (private Ausgaben als
  Betriebsausgabe, fehlende Erlöse, unrichtige Belege): Buche sie nicht und
  sprich den Menschen an. Berichtigung oder Selbstanzeige
  ([§ 371 AO][ao371]) klärt er mit einem Steuerberater.
- **Mehr als 50 % der Anteile innerhalb von fünf Jahren übertragen.** Nicht
  genutzte Verluste können wegfallen
  ([steuererklaerungen/kst.md, Abschnitt 8](../steuererklaerungen/kst.md#8-verlustabzug)).

## 3. Wann du einen Steuerberater empfiehlst

- Eine Betriebsprüfung ist angekündigt.
- Umwandlung, Verschmelzung, Kapitalerhöhung oder -herabsetzung, Kauf oder
  Verkauf von Anteilen.
- Auslandssachverhalte über Reverse Charge und innergemeinschaftlichen Erwerb
  hinaus, etwa eine Betriebsstätte oder Leistungen zwischen nahestehenden
  Unternehmen im Ausland ([§ 1 AStG][astg1]).
- Einspruch gegen einen Steuerbescheid.
- Eine Wertungsfrage mit großem Betrag oder Wirkung über mehrere Jahre.
- Der Mensch will eine verbindliche Auskunft.

## 4. Wie du eine Grenze ansprichst

```
Buchfink bildet <Fall> nicht ab (<Grund, Fundstelle>).
Folge: <was ohne Lösung passiert, z. B. Vorsteuer fehlt in der UStVA>.
Vorschlag: Beleg ist abgelegt und bleibt ungebucht; <externer Weg oder Steuerberater>.
Soll ich eine Aufgabe dafür notieren oder den Beleg mit Hinweis liegen lassen?
```

[ustg2]: https://www.gesetze-im-internet.de/ustg_1980/__2.html
[ustg9]: https://www.gesetze-im-internet.de/ustg_1980/__9.html
[ustg13b]: https://www.gesetze-im-internet.de/ustg_1980/__13b.html
[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg18i]: https://www.gesetze-im-internet.de/ustg_1980/__18i.html
[ustg18j]: https://www.gesetze-im-internet.de/ustg_1980/__18j.html
[ustg18k]: https://www.gesetze-im-internet.de/ustg_1980/__18k.html
[ustg19]: https://www.gesetze-im-internet.de/ustg_1980/__19.html
[ustg20]: https://www.gesetze-im-internet.de/ustg_1980/__20.html
[ustg25]: https://www.gesetze-im-internet.de/ustg_1980/__25.html
[ustg25a]: https://www.gesetze-im-internet.de/ustg_1980/__25a.html
[ustg25b]: https://www.gesetze-im-internet.de/ustg_1980/__25b.html
[kstg14]: https://www.gesetze-im-internet.de/kstg_1977/__14.html
[kstg17]: https://www.gesetze-im-internet.de/kstg_1977/__17.html
[hgb267]: https://www.gesetze-im-internet.de/hgb/__267.html
[hgb290]: https://www.gesetze-im-internet.de/hgb/__290.html
[hgb316]: https://www.gesetze-im-internet.de/hgb/__316.html
[gmbhg5a]: https://www.gesetze-im-internet.de/gmbhg/__5a.html
[gmbhg49]: https://www.gesetze-im-internet.de/gmbhg/__49.html
[inso15a]: https://www.gesetze-im-internet.de/inso/__15a.html
[inso19]: https://www.gesetze-im-internet.de/inso/__19.html
[ao371]: https://www.gesetze-im-internet.de/ao_1977/__371.html
[astg1]: https://www.gesetze-im-internet.de/astg/__1.html
