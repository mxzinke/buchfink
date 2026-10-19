# E-Rechnung auf der Ausgangsseite

Rechtsstand: 17. Oktober 2026. Verwaltungsauffassung:
[BMF-Schreiben vom 15.10.2024][bmf2024] (III C 2 - S 7287-a/23/10001 :007,
BStBl I S. 1320, Spiegel des Bayerischen Landesamts für Steuern), geändert und
ergänzt durch BMF-Schreiben vom 15.10.2025 (III C 2 - S 7287-a/00019/007/243),
das zugleich den UStAE anpasst. Das Schreiben von 2025 wurde über einen
[nichtamtlichen Spiegel][bmf2025] gelesen; das BMF-Original war nicht
abrufbar.

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
  ([§ 14 Abs. 1 Satz 3 UStG][ustg14]). Das Format entspricht der Norm EN 16931
  oder ist zwischen den Parteien vereinbart und in sie überführbar
  ([§ 14 Abs. 1 Satz 6 UStG][ustg14]).
- **Sonstige Rechnung:** Papier oder ein anderes elektronisches Format, etwa ein
  PDF ohne eingebettete Daten ([§ 14 Abs. 1 Satz 4 UStG][ustg14],
  [UStAE 14.1 Abs. 2][bmf2025]).
- Eine Datei mit Formatfehlern gilt als sonstige Rechnung
  ([BMF 15.10.2025, Rn. 6a][bmf2025]).

## 2. Wann eine E-Rechnung Pflicht ist

Pflicht bei einer Leistung an ein anderes Unternehmen für dessen Unternehmen,
wenn beide im Inland ansässig sind ([§ 14 Abs. 2 Satz 2 Nr. 1 UStG][ustg14]).
Ansässig heißt: Sitz, Geschäftsleitung oder beteiligte Betriebsstätte im Inland
([§ 14 Abs. 2 Satz 3 UStG][ustg14]).

Keine Pflicht:

| Fall | Grundlage |
|---|---|
| Kunde ist Verbraucher | [§ 14 Abs. 2 Satz 2 Nr. 1 UStG][ustg14] (nur Unternehmer erfasst) |
| Kunde oder Unternehmen nicht im Inland ansässig | [UStAE 14.1 Abs. 6 Satz 3][bmf2025] |
| Steuerfreie Leistung nach § 4 Nr. 8 bis 29 UStG | [§ 14 Abs. 2 Satz 2 UStG][ustg14] |
| Kleinbetragsrechnung bis 250 € | [§ 33 Satz 4 UStDV][ustdv33] |

Die Pflicht gilt auch für Anzahlungsrechnungen, weil § 14 Abs. 1 bis 4 UStG für
sie sinngemäß gilt ([§ 14 Abs. 5 Satz 1 UStG][ustg14]). Sie gilt ebenso für
Gutschriften und für Rechnungen mit Steuerschuldnerschaft des
Leistungsempfängers zwischen Inländern ([UStAE 14.1 Abs. 4 Satz 8][bmf2025],
[BMF 15.10.2025, Rn. 17][bmf2025]).

Hat das Unternehmen ohne Fahrlässigkeit nicht erkannt, dass der Kunde
Unternehmer ist, beanstandet die Verwaltung die fehlende E-Rechnung nicht. Eine
USt-IdNr. kann ein Indiz für die Unternehmereigenschaft sein
([UStAE 14.1 Abs. 6 Satz 1, 2][bmf2025]).

## 3. Übergang bis Ende 2027

Nach [§ 27 Abs. 38 UStG][ustg27] darf statt der E-Rechnung eine sonstige
Rechnung hinausgehen:

| Leistung ausgeführt | Sonstige Rechnung zulässig | Bedingung |
|---|---|---|
| 01.01.2025 bis 31.12.2026 | Papier; anderes elektronisches Format mit Zustimmung | Rechnung bis 31.12.2026 übermittelt (Nr. 1) |
| 01.01.2027 bis 31.12.2027 | Papier; anderes elektronisches Format mit Zustimmung | Gesamtumsatz (§ 19 Abs. 2 UStG) des Unternehmens im Vorjahr höchstens 800.000 €; Rechnung bis 31.12.2027 übermittelt (Nr. 2) |
| 01.01.2027 bis 31.12.2027 | EDI-Format mit Zustimmung | Rechnung bis 31.12.2027 (Nr. 3) |
| ab 01.01.2028 | – | E-Rechnung Pflicht |

Maßgeblich sind der Tag der Leistung und der Tag der Übermittlung. Eine
Leistung aus 2026, die erst nach dem 31.12.2026 abgerechnet wird, fällt unter
keine Übergangsregel: Nr. 2 und 3 gelten nur für Umsätze ab 2027. Sie braucht
eine E-Rechnung, auch beim kleinen Unternehmen. Das betrifft vor allem die
Abrechnung des Dezembers 2026 im Januar 2027. Für den Empfang von E-Rechnungen
gibt es keinen Übergang ([BMF 15.10.2024, Rn. 62][bmf2024]). Stand: Oktober
2026; Fundstelle ist der Gesetzestext.

Regel für dich: Stell jede Rechnung an ein inländisches Unternehmen als
E-Rechnung aus. Ein reines PDF nur, wenn der Mensch es ausdrücklich will, die
Übergangsregel greift und der Kunde zustimmt.

## 4. Formate und was Buchfink erzeugt

- Immer zulässig sind Formate nach EN 16931, z. B. XRechnung und ZUGFeRD ab
  Version 2.0.1; die ZUGFeRD-Profile MINIMUM und BASIC-WL genügen nicht
  ([UStAE 14.1 Abs. 14][bmf2025]).
- Buchfink erzeugt hybride ZUGFeRD-Rechnungen im Profil EN 16931 (PDF mit
  eingebettetem XML) oder XRechnung im CII-Format, je nach Einstellung am
  Kontakt. Es prüft die Datei vor dem Ausstellen gegen die Regeln; scheitert die
  Prüfung, stellt es nicht aus.
- Behörden und manche Unternehmen verlangen XRechnung mit Leitweg-ID. Ob ein
  Kontakt sie hat, siehst du in `kontakte_suchen` nicht vollständig; frag den
  Menschen. Das Profil am Kontakt ändert er in der App.
- Bei einem hybriden Format führt der strukturierte Teil. Weichen PDF und XML
  voneinander ab, gelten die Daten des XML ([UStAE 14.4 Abs. 3 i. d. F. vom 15.10.2025][bmf2025]).

## 5. Inhalt des strukturierten Teils

- Alle Pflichtangaben nach §§ 14, 14a UStG stehen im strukturierten Teil. Ein
  Verweis auf einen Anhang mit den Angaben genügt nicht; ein Link auf ein
  externes Ziel auch nicht ([BMF 15.10.2025, Rn. 35][bmf2025]).
- Ergänzende Unterlagen (z. B. Stundennachweise als PDF) dürfen als Anhang in
  der E-Rechnung stehen ([BMF 15.10.2025, Rn. 35][bmf2025]). Frag den
  Menschen, wie er solche Nachweise mitschickt.
- Die Leistungsbeschreibung im strukturierten Teil muss die Leistung eindeutig
  erkennen lassen. Schreib deshalb präzise Positionstexte („Entwicklung
  Kundenportal, Sprint 14, 01.–14.10.2026, 62 Std.“).

## 6. Zustimmung bei PDF und Papier

- Papier ist zulässig, wenn keine E-Rechnungspflicht besteht oder der Übergang
  greift ([UStAE 14.1 Abs. 7 Satz 1][bmf2025]).
- Jede elektronische Übermittlung außerhalb der Pflicht braucht die Zustimmung
  des Empfängers ([§ 14 Abs. 1 Satz 5 UStG][ustg14]). Sie ist formlos, kann in
  AGB stehen, nachträglich erklärt werden oder sich aus gelebter Praxis ergeben
  ([UStAE 14.1 Abs. 7 Satz 4 bis 8][bmf2025]).
- An ein inländisches Unternehmen darf eine E-Rechnung immer ohne Zustimmung
  hinausgehen ([UStAE 14.1 Abs. 4 Satz 4][bmf2025]).

## 7. Versand und Empfangsprobleme

- Übermittlungswege: E-Mail, Portal, EDI, Schnittstelle, gemeinsamer
  Speicherort. Format und Weg vereinbaren die Parteien
  ([UStAE 14.1 Abs. 4 Satz 6, 7][bmf2025]).
- Kann oder will ein inländischer Kunde keine E-Rechnung annehmen, hat er keinen
  Anspruch auf eine sonstige Rechnung. Das Unternehmen hat seine Pflicht erfüllt,
  wenn es eine E-Rechnung ausgestellt und sich nachweislich um die Übermittlung
  bemüht hat, z. B. mit Sendeprotokoll ([UStAE 14.1 Abs. 5 Satz 4, 5][bmf2025]).
  Rate dem Menschen, das Protokoll als Beleg abzulegen.
- Mehrfach versandte identische E-Rechnungen lösen keine zusätzliche Steuer aus
  ([UStAE 14c.1 Abs. 4 Satz 7 i. d. F. vom 15.10.2025][bmf2025]). Eine
  inhaltlich abweichende Zweitfassung (anderer Betrag, andere Leistung) schon:
  schick nie eine nachträglich geänderte Datei unter derselben Nummer.

## 8. Berichtigung und Aufbewahrung

- Eine E-Rechnung wird als E-Rechnung berichtigt; Angaben in anderer Form
  reichen nicht. Besteht wegen des Übergangs keine Pflicht, darf die
  Berichtigung auch als sonstige Rechnung erfolgen
  ([UStAE 14.11 Abs. 1 Satz 7 bis 9 i. d. F. vom 15.10.2025][bmf2025]).
- Aufbewahrt wird mindestens der strukturierte Teil in seiner ursprünglichen
  Form ([BMF 15.10.2025, Rn. 60][bmf2025]). Buchfink legt die ausgestellte Datei
  unverändert ab. Fristen: buchfink-grundlagen.

[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg27]: https://www.gesetze-im-internet.de/ustg_1980/__27.html
[ustdv33]: https://www.gesetze-im-internet.de/ustdv_1980/__33.html
[bmf2024]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/E-Rechnung/2024-10-15-einfuehrung-e-rechnung.pdf
[bmf2025]: https://www.steuerschroeder.de/blog/wp-content/uploads/2025/10/2025-10-15-einfuehrung-obligatorische-e-rechnung.pdf
