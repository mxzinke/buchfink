# E-Rechnung auf der Ausgangsseite

Rechtsstand: 17. Oktober 2026. Verwaltungsauffassung:
[BMF 15.10.2024][bmf2024], geändert durch BMF 15.10.2025
([Kanzleispiegel][bmf2025]), mit UStAE.

Empfang: [belege/e-rechnung-empfang.md](../belege/e-rechnung-empfang.md).

## Inhalt

1. Begriffe
2. Wann eine E-Rechnung Pflicht ist
3. Übergang bis Ende 2027
4. Formate und was Buchfink erzeugt
5. Inhalt des strukturierten Teils
6. Zustimmung bei PDF und Papier
7. Versand und Empfangsprobleme
8. Berichtigung und Aufbewahrung

## 1. Begriffe

- **E-Rechnung:** in einem strukturierten elektronischen Format ausgestellt,
  übermittelt und empfangen, elektronisch verarbeitbar
  ([§ 14 Abs. 1 Satz 3 UStG][ustg14]); Format nach EN 16931 oder vereinbart
  und in sie überführbar (Satz 6).
- **Sonstige Rechnung:** Papier oder anderes elektronisches Format, etwa PDF
  ohne Datensatz ([§ 14 Abs. 1 Satz 4 UStG][ustg14],
  [UStAE 14.1 Abs. 2][bmf2025]), auch eine Datei mit Formatfehlern
  ([BMF 15.10.2025, Rn. 6a][bmf2025]).

## 2. Wann eine E-Rechnung Pflicht ist

Pflicht bei Leistung an ein Unternehmen für dessen Unternehmen, wenn beide im
Inland ansässig sind (Sitz, Geschäftsleitung oder beteiligte Betriebsstätte;
[§ 14 Abs. 2 Satz 2 Nr. 1, Satz 3 UStG][ustg14]).

Keine Pflicht:

| Fall | Grundlage |
|---|---|
| Kunde ist Verbraucher | [§ 14 Abs. 2 Satz 2 Nr. 1 UStG][ustg14] |
| Kunde oder Unternehmen nicht im Inland ansässig | [UStAE 14.1 Abs. 6 Satz 3][bmf2025] |
| Steuerfreie Leistung nach § 4 Nr. 8 bis 29 UStG | [§ 14 Abs. 2 Satz 2 UStG][ustg14] |
| Kleinbetragsrechnung bis 250 € | [§ 33 Satz 4 UStDV][ustdv33] |

Die Pflicht gilt auch für Anzahlungsrechnungen ([§ 14 Abs. 5 Satz 1 UStG][ustg14],
[UStAE 14.1 Abs. 4 Satz 1][bmf2025]), Gutschriften und Rechnungen mit
Steuerschuldnerschaft des Empfängers zwischen Inländern
([UStAE 14.1 Abs. 4 Satz 8][bmf2025], [BMF 15.10.2025, Rn. 17][bmf2025]).
Unerkannte Unternehmereigenschaft ohne Fahrlässigkeit beanstandet die
Verwaltung nicht; eine USt-IdNr. ist ein Indiz
([UStAE 14.1 Abs. 6 Satz 1, 2][bmf2025]).

## 3. Übergang bis Ende 2027

Statt der E-Rechnung ist eine sonstige Rechnung zulässig
([§ 27 Abs. 38 UStG][ustg27]):

| Leistung ausgeführt | Sonstige Rechnung zulässig | Bedingung |
|---|---|---|
| 01.01.2025 bis 31.12.2026 | Papier; anderes elektronisches Format mit Zustimmung | Rechnung bis 31.12.2026 übermittelt (Nr. 1) |
| 01.01.2027 bis 31.12.2027 | Papier; anderes elektronisches Format mit Zustimmung | Gesamtumsatz (§ 19 Abs. 2 UStG) im Vorjahr höchstens 800.000 €; Rechnung bis 31.12.2027 übermittelt (Nr. 2) |
| 01.01.2027 bis 31.12.2027 | EDI-Format mit Zustimmung | Rechnung bis 31.12.2027 (Nr. 3) |
| ab 01.01.2028 | – | E-Rechnung Pflicht |

Maßgeblich sind Leistungstag und Übermittlungstag. Eine Leistung aus 2026, die
nach dem 31.12.2026 abgerechnet wird, braucht eine E-Rechnung, auch beim
kleinen Unternehmen (Nr. 2 und 3 gelten erst für Umsätze ab 2027); typisch ist
die Dezember-Rechnung im Januar 2027.

Regel für dich: Stell jede Rechnung an ein inländisches Unternehmen als
E-Rechnung aus. Ein reines PDF nur, wenn der Mensch es ausdrücklich will, die
Übergangsregel greift und der Kunde zustimmt.

## 4. Formate und was Buchfink erzeugt

- Zulässig sind Formate nach EN 16931, z. B. XRechnung und ZUGFeRD ab 2.0.1
  ohne die Profile MINIMUM und BASIC-WL ([UStAE 14.1 Abs. 14][bmf2025]).
- Buchfink erzeugt ZUGFeRD im Profil EN 16931 (PDF mit XML) oder, nach
  Einstellung am Kontakt, XRechnung (CII).
- XRechnung mit Leitweg-ID (Behörden, manche Unternehmen): neuer Kontakt mit
  `leitweg_id`. Ob ein vorhandener Kontakt sie hat, zeigt `kontakte_suchen`
  nicht vollständig; frag den Menschen, er ändert das Profil in der App.
- Bei hybriden Formaten gilt bei Abweichungen das XML
  ([UStAE 14.4 Abs. 3 i. d. F. vom 15.10.2025][bmf2025]).

## 5. Inhalt des strukturierten Teils

- Alle Pflichtangaben nach §§ 14, 14a UStG stehen im strukturierten Teil; ein
  Verweis auf einen Anhang oder Link genügt nicht
  ([BMF 15.10.2025, Rn. 35][bmf2025]).
- Ergänzende Unterlagen (Stundennachweise) dürfen als Anhang mit
  ([BMF 15.10.2025, Rn. 35][bmf2025]); frag den Menschen, wie er sie mitschickt.
- Schreib Positionstexte, die die Leistung eindeutig erkennen lassen
  („Entwicklung Kundenportal, Sprint 14, 01.–14.10.2026, 62 Std.“).

## 6. Zustimmung bei PDF und Papier

- Elektronische Übermittlung außerhalb der Pflicht braucht die Zustimmung des
  Empfängers ([§ 14 Abs. 1 Satz 5 UStG][ustg14]); formlos, auch in AGB,
  nachträglich oder durch gelebte Praxis
  ([UStAE 14.1 Abs. 7 Satz 4 bis 8][bmf2025]).
- An ein inländisches Unternehmen geht eine E-Rechnung immer ohne Zustimmung
  ([UStAE 14.1 Abs. 4 Satz 4][bmf2025]).

## 7. Versand und Empfangsprobleme

- Nimmt ein inländischer Kunde keine E-Rechnung an, bleibt es bei der
  E-Rechnung; die Pflicht ist mit nachweisbarem Übermittlungsversuch erfüllt
  ([UStAE 14.1 Abs. 5 Satz 4, 5][bmf2025]). Rate, das Sendeprotokoll
  abzulegen.
- Identische Mehrfachsendungen lösen keine Steuer aus
  ([UStAE 14c.1 Abs. 4 Satz 7 i. d. F. vom 15.10.2025][bmf2025]), eine
  abweichende Zweitfassung schon: schick nie eine geänderte Datei unter
  derselben Nummer.

## 8. Berichtigung und Aufbewahrung

- Eine E-Rechnung wird als E-Rechnung berichtigt; greift der Übergang, auch als
  sonstige Rechnung
  ([UStAE 14.11 Abs. 1 Satz 7 bis 9 i. d. F. vom 15.10.2025][bmf2025]).
- Aufbewahrt wird mindestens der strukturierte Teil im Original
  ([BMF 15.10.2025, Rn. 60][bmf2025]); Fristen:
  [grundlagen/aufbewahrung.md](../grundlagen/aufbewahrung.md).

[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg27]: https://www.gesetze-im-internet.de/ustg_1980/__27.html
[ustdv33]: https://www.gesetze-im-internet.de/ustdv_1980/__33.html
[bmf2024]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/E-Rechnung/2024-10-15-einfuehrung-e-rechnung.pdf
[bmf2025]: https://www.steuerschroeder.de/blog/wp-content/uploads/2025/10/2025-10-15-einfuehrung-obligatorische-e-rechnung.pdf
