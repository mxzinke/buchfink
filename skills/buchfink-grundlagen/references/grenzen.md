# Grenzen von Buchfink, Warnzeichen, Steuerberater

Rechtsstand: 17. Oktober 2026. Funktionsumfang von Buchfink 0.3.0. Lehnt ein
Werkzeug einen Fall mit `fachliche_regel` ab, gilt die Meldung des Servers.

## Inhalt

1. Was Buchfink nicht abbildet
2. Warnzeichen, die du sofort meldest
3. Wann du einen Steuerberater empfiehlst
4. Wie du eine Grenze ansprichst

## 1. Was Buchfink nicht abbildet

| Fall | Was du tust |
|---|---|
| Andere Rechtsform als GmbH oder UG, Einnahmenüberschussrechnung | Buchfink ist dafür nicht gebaut. Nicht buchen, den Menschen informieren. |
| Eigene Kleinunternehmerregelung ([§ 19 UStG][ustg19]) | Buchfink setzt voraus, dass das Unternehmen Umsatzsteuer ausweist und anmeldet. Ein Lieferant, der Kleinunternehmer ist, lässt sich erfassen. |
| Istversteuerung ([§ 20 UStG][ustg20]) | Buchfink lehnt Buchungen ab. Mandant mit Istversteuerung: Mensch und Steuerberater klären. |
| OSS und IOSS ([§§ 18i][ustg18i], [18j][ustg18j], [18k UStG][ustg18k]) | Meldung außerhalb von Buchfink. |
| Differenzbesteuerung ([§ 25a UStG][ustg25a]), Reiseleistungen ([§ 25 UStG][ustg25]), Dreiecksgeschäft ([§ 25b UStG][ustg25b]) | Nicht abgebildet. Steuerberater. |
| Bauleistungen als Leistungsempfänger nach [§ 13b Abs. 2 Nr. 4 UStG][ustg13b] | Nicht abgebildet. Steuerberater. |
| Verzicht auf Steuerbefreiungen ([§ 9 UStG][ustg9]) | Nicht abgebildet. |
| Einfuhrumsatzsteuer, unentgeltliche Wertabgaben | Nicht vollständig abgebildet. Vor dem Buchen nachfragen. |
| Gutschrift im Sinne von [§ 14 Abs. 2 Satz 5 UStG][ustg14] ausstellen (Abrechnung durch den Leistungsempfänger) | Buchfink stellt keine aus. Empfangene Gutschriften lassen sich ablegen. |
| Organschaft ([§§ 14][kstg14], [17 KStG][kstg17], [§ 2 Abs. 2 Nr. 2 UStG][ustg2]) | Nicht abgebildet. Steuerberater. |
| Reine Holding ohne eigene Umsätze | Ob sie Unternehmer im Sinne des [§ 2 Abs. 1 UStG][ustg2] ist und Vorsteuer abziehen darf, ist eine Wertungsfrage. Buchfink geht von einem Unternehmer aus. Nachfragen, siehe buchfink-beteiligungen. |
| Kasse, Bargeschäfte mit Kassensystem, Warenlager | Nicht im Funktionsumfang. |
| Lohnabrechnung, Reisekostenabrechnung | Extern. Buchfink bucht die Summen aus dem Lohnjournal (buchfink-lohn). |
| Bankkonto in Fremdwährung, CAMT-Sammeleinträge mit mehreren Einzelumsätzen | Der Import lehnt sie ab. Mensch informieren. |
| Konzernabschluss ([§ 290 HGB][hgb290]), IFRS, kapitalmarktorientierte Gesellschaft | Nicht abgebildet. |
| Mittelgroße oder große Gesellschaft ([§ 267 HGB][hgb267]): Abschlussprüfung ([§ 316 HGB][hgb316]), latente Steuern, Lagebericht | Steuerberater und Wirtschaftsprüfer. |
| Eigene Steuerbilanz mit abweichenden Werten | Buchfink führt eine Einheitsbilanz; abweichende Steuerwerte nur bei Sonderabschreibungen nach § 7g EStG. |
| Ersetzendes Scannen | Kein dokumentiertes Verfahren in Buchfink ([GoBD Rz. 136][gobd]). Papierbelege aufheben lassen. |
| Übermittlung an das Finanzamt | Buchfink übermittelt nicht selbst. Es zeigt die Erklärungen formularnah; der Mensch überträgt sie in Mein ELSTER und sendet (buchfink-monatsabschluss, buchfink-steuererklaerungen). Für die E-Bilanz erzeugt Buchfink eine XBRL-Datei für den Import in einen Online-Dienst wie eBilanz+ oder eBilanz-Online; Weg, Kontennachweise, Anlagenspiegel und Härtefall: buchfink-steuererklaerungen/references/ebilanz.md. |
| Versand von Rechnungen (E-Mail, Peppol, Portal) | Außerhalb von Buchfink. |

## 2. Warnzeichen, die du sofort meldest

Melde diese Funde unabhängig von deinem Auftrag, mit Zahlen und Fundstelle.
Die Bewertung trifft der Mensch mit Berater.

- **Hälfte des Stammkapitals verloren.** Zeigt eine Bilanz, auch eine
  unterjährige, dass die Hälfte des Stammkapitals verloren ist (Eigenkapital
  unter der Hälfte des Stammkapitals), muss die Geschäftsführung einer GmbH die
  Gesellschafterversammlung unverzüglich einberufen
  ([§ 49 Abs. 3 GmbHG][gmbhg49]). Bei der UG gilt das bei drohender
  Zahlungsunfähigkeit ([§ 5a Abs. 4 GmbHG][gmbhg5a]).
- **Negatives Eigenkapital oder Zahlungsschwierigkeiten.** Bei
  Zahlungsunfähigkeit oder Überschuldung muss die Geschäftsführung ohne
  schuldhaftes Zögern Insolvenz beantragen, spätestens drei Wochen nach Eintritt
  der Zahlungsunfähigkeit und sechs Wochen nach Eintritt der Überschuldung
  ([§ 15a Abs. 1 InsO][inso15a]). Bilanziell negatives Eigenkapital ist nicht
  dasselbe wie Überschuldung; diese hängt auch von der Fortführungsprognose ab
  ([§ 19 Abs. 2 InsO][inso19]). Rate zu sofortiger rechtlicher Beratung.
- **Fehler in einer übermittelten Erklärung.** Kann er zu einer zu niedrigen
  Steuer geführt haben, muss das Unternehmen ihn unverzüglich anzeigen und
  richtigstellen ([§ 153 Abs. 1 AO][ao153]).
- **Vorgänge, die nach Steuerverkürzung aussehen** (private Ausgaben als
  Betriebsausgabe, fehlende Erlöse, unrichtige Belege). Buche sie nicht und
  sprich den Menschen an. Ob eine Berichtigung oder eine Selbstanzeige
  ([§ 371 AO][ao371]) in Frage kommt, klärt er mit einem Steuerberater.
- **Übertragung von mehr als 50 % der Anteile innerhalb von fünf Jahren** an
  einen Erwerber oder ihm nahestehende Personen. Nicht genutzte Verluste können
  wegfallen ([§ 8c Abs. 1 KStG][kstg8c]). Hinweis an den Menschen.

## 3. Wann du einen Steuerberater empfiehlst

- Eine Betriebsprüfung ist angekündigt.
- Umwandlung, Verschmelzung, Kapitalerhöhung oder -herabsetzung, Verkauf oder
  Kauf von Anteilen.
- Auslandssachverhalte über Reverse Charge und innergemeinschaftlichen Erwerb
  hinaus, etwa eine Betriebsstätte oder Leistungen zwischen nahestehenden
  Unternehmen im Ausland ([§ 1 AStG][astg1]).
- Einspruch gegen einen Steuerbescheid.
- Eine Wertungsfrage mit großem Betrag oder mit Wirkung über mehrere Jahre.
- Der Mensch ist unsicher und will eine verbindliche Auskunft.

## 4. Wie du eine Grenze ansprichst

```
Buchfink bildet <Fall> nicht ab (<Grund, Fundstelle>).
Folge: <was ohne Lösung passiert, z. B. Vorsteuer fehlt in der UStVA>.
Vorschlag: Beleg ist abgelegt und bleibt ungebucht; <externer Weg oder Steuerberater>.
Soll ich eine Aufgabe dafür notieren oder den Beleg mit Hinweis liegen lassen?
```

[gobd]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf
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
[kstg8c]: https://www.gesetze-im-internet.de/kstg_1977/__8c.html
[kstg14]: https://www.gesetze-im-internet.de/kstg_1977/__14.html
[kstg17]: https://www.gesetze-im-internet.de/kstg_1977/__17.html
[hgb267]: https://www.gesetze-im-internet.de/hgb/__267.html
[hgb290]: https://www.gesetze-im-internet.de/hgb/__290.html
[hgb316]: https://www.gesetze-im-internet.de/hgb/__316.html
[gmbhg5a]: https://www.gesetze-im-internet.de/gmbhg/__5a.html
[gmbhg49]: https://www.gesetze-im-internet.de/gmbhg/__49.html
[inso15a]: https://www.gesetze-im-internet.de/inso/__15a.html
[inso19]: https://www.gesetze-im-internet.de/inso/__19.html
[ao153]: https://www.gesetze-im-internet.de/ao_1977/__153.html
[ao371]: https://www.gesetze-im-internet.de/ao_1977/__371.html
[astg1]: https://www.gesetze-im-internet.de/astg/__1.html
