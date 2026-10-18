# Steuerfälle auf der Eingangsseite

Rechtsstand: 17. Oktober 2026. Wortlaut von UStG und UStDV am 17.10.2026
gelesen (UStG zuletzt geändert durch Art. 5 G v. 29.06.2026).

## Inhalt

1. Steuerfälle in Buchfink
2. Leistungen aus dem Ausland (§ 13b UStG)
3. Waren aus einem anderen EU-Staat (innergemeinschaftlicher Erwerb)
4. Waren aus dem Drittland
5. Steuerfrei und nicht steuerbar
6. Lieferant ist Kleinunternehmer
7. Ausländische Umsatzsteuer auf der Rechnung
8. Steuersatz

## 1. Steuerfälle in Buchfink

Die gültigen Werte zeigt das Schema; `konten` mit `art: steuerfaelle` erklärt
sie. Für Eingangsbelege gibt es:

| Steuerfall | Steuerzeilen, die Buchfink bucht | Typischer Beleg |
|---|---|---|
| Inland, steuerpflichtig | Vorsteuer 19 % (1406) oder 7 % (1401) | deutscher Lieferant mit Steuerausweis |
| § 13b Abs. 1 UStG (`reverse_charge_eu`) | Vorsteuer (1407) und geschuldete Steuer (3837) in gleicher Höhe; Kennzahl 46/47 | Software-Abo, Werbung, Beratung aus einem anderen EU-Staat |
| § 13b Abs. 2 UStG (`reverse_charge`) | wie oben; Kennzahl 84/85 | Leistung aus dem Drittland, Werklieferung, inländische Bauleistung an einen Bauleister |
| Innergemeinschaftlicher Erwerb | Vorsteuer (1404) und Erwerbsteuer (3804) | Ware aus einem anderen EU-Staat |
| Nullsteuersatz | keine | Photovoltaik nach § 12 Abs. 3 UStG |
| Steuerfrei | keine | Versicherung, Kontoführung, Miete ohne Option |
| Nicht steuerbar | keine | Kammerbeitrag, Behördengebühr, echter Schadenersatz |

Kontonummern nach dem SKR04 in Buchfink.
„Ohne Steuer“ ist kein Steuerfall: Wähle immer den Grund
([GoBD Rz. 55][gobd], getrennte Aufzeichnung nach [§ 22 UStG][ustg22]).

## 2. Leistungen aus dem Ausland (§ 13b UStG)

**Wann die GmbH die Steuer schuldet.** Für Werklieferungen und sonstige
Leistungen eines im Ausland ansässigen Unternehmers schuldet der
Leistungsempfänger die Steuer, wenn er Unternehmer oder juristische Person ist
([§ 13b Abs. 1, Abs. 2 Nr. 1, Abs. 5 Satz 1 UStG][ustg13b]). Im Ausland ansässig
ist, wer im Inland weder Sitz, Geschäftsleitung, Betriebsstätte noch Wohnsitz
hat ([§ 13b Abs. 7 UStG][ustg13b]).

**Ort der Leistung.** Eine sonstige Leistung an ein Unternehmen wird dort
ausgeführt, wo der Empfänger sein Unternehmen betreibt
([§ 3a Abs. 2 UStG][ustg3a]). Ausnahmen mit Ort im Ausland, also ausländischer
Steuer statt § 13b:

- Leistungen im Zusammenhang mit einem Grundstück, etwa Hotelübernachtung
  ([§ 3a Abs. 3 Nr. 1 UStG][ustg3a]).
- Kurzfristige Vermietung eines Beförderungsmittels, etwa Mietwagen, am Ort der
  Übergabe ([§ 3a Abs. 3 Nr. 2 UStG][ustg3a]).

Diese Belege führen nicht zu § 13b; siehe Abschnitt 7.

**Kein § 13b** bei grenzüberschreitender Personenbeförderung im Luftverkehr,
Eintritt zu Messen und Kongressen im Inland und Restauration an Bord
([§ 13b Abs. 6 UStG][ustg13b]).

**Zeitpunkt der Steuer.**

| Fall | Steuer entsteht | Fundstelle |
|---|---|---|
| Sonstige Leistung eines Unternehmers aus einem anderen EU-Staat | mit Ablauf des Voranmeldungszeitraums der Leistung | [§ 13b Abs. 1 UStG][ustg13b] |
| Leistung aus dem Drittland, Werklieferung | mit Ausstellung der Rechnung, spätestens mit Ablauf des Folgemonats der Leistung | [§ 13b Abs. 2 Nr. 1 UStG][ustg13b] |
| Dauerleistung über mehr als ein Jahr | spätestens mit Ablauf jedes Kalenderjahres | [§ 13b Abs. 3 UStG][ustg13b] |
| Zahlung vor der Leistung | mit Ablauf des Voranmeldungszeitraums der Zahlung | [§ 13b Abs. 4 Satz 2 UStG][ustg13b] |

Buchfink ordnet die Steuer nach Abs. 1 dem Leistungszeitraum zu, die nach
Abs. 2 dem Rechnungsdatum, spätestens dem Folgemonat der Leistung. Ohne
Leistungszeitraum entscheidet das Belegdatum; trag ihn deshalb immer ein.

**Vorsteuer.** Die Steuer nach § 13b ist als Vorsteuer abziehbar, wenn die
Leistung für das Unternehmen ausgeführt ist
([§ 15 Abs. 1 Satz 1 Nr. 4 UStG][ustg15]). Eine Rechnung nach § 14 ist dafür
nicht Voraussetzung; als Beleg brauchst du sie trotzdem.

**Rechnung.** Der Lieferant soll „Steuerschuldnerschaft des
Leistungsempfängers“ angeben ([§ 14a Abs. 5 UStG][ustg14a]). Fehlt der Hinweis,
ändert das an § 13b nichts.

**Was Buchfink nicht abbildet.**

- Ob die GmbH bei einer Bauleistung eines inländischen Unternehmers
  ([§ 13b Abs. 2 Nr. 4 UStG][ustg13b]) die Steuer schuldet: nur, wenn sie
  selbst nachhaltig Bauleistungen erbringt ([§ 13b Abs. 5 Satz 2 UStG][ustg13b]).
  Dann buchst du `reverse_charge`; Buchfink prüft die Voraussetzung nicht.
  Die Bauabzugsteuer ([§ 48 EStG][estg48]) bildet Buchfink nicht ab.
- Mobilfunkgeräte, Tablets, Spielekonsolen, Schaltkreise und Metalle: § 13b
  erst ab 5.000 € Entgelt je wirtschaftlichem Vorgang
  ([§ 13b Abs. 2 Nr. 10, 11 UStG][ustg13b]). Buchfink prüft die Grenze nicht.
  Stand: Oktober 2026.
- Gebäudereinigung und Telekommunikation zwischen Inländern (Nr. 8, 12) setzen
  beim Empfänger eine Bescheinigung voraus
  ([§ 13b Abs. 5 Satz 5, 6 UStG][ustg13b]); für die Zielgruppe von Buchfink
  meist nicht einschlägig. Weist ein inländischer Lieferant solche Leistungen
  ohne Steuer aus, frag nach.

**Kontakt.** Für einen EU-Lieferanten verlangt Buchfink die USt-IdNr. am
Kontakt. Für Drittlandslieferanten nicht.

## 3. Waren aus einem anderen EU-Staat (innergemeinschaftlicher Erwerb)

- Voraussetzungen: Ware gelangt aus einem anderen Mitgliedstaat ins Inland, die
  GmbH erwirbt für ihr Unternehmen, der Lieferant ist Unternehmer und nicht
  Kleinunternehmer in seinem Staat ([§ 1a Abs. 1 UStG][ustg1a]).
- Ort ist das Ende der Beförderung. Verwendet die GmbH die USt-IdNr. eines
  anderen Mitgliedstaats, gilt der Erwerb zunächst dort als bewirkt
  ([§ 3d UStG][ustg3d]).
- Die Erwerbsteuer entsteht mit Ausstellung der Rechnung, spätestens mit Ablauf
  des Folgemonats ([§ 13 Abs. 1 Nr. 6 UStG][ustg13]); abziehbar als Vorsteuer
  ([§ 15 Abs. 1 Satz 1 Nr. 3 UStG][ustg15]).
- Weist der EU-Lieferant seine eigene Steuer aus, hat er die USt-IdNr. der GmbH
  nicht verwendet. Lass die Rechnung berichtigen; siehe Abschnitt 7.

## 4. Waren aus dem Drittland

Die Einfuhr unterliegt der Einfuhrumsatzsteuer; abziehbar ist die entstandene
Einfuhrumsatzsteuer ([§ 15 Abs. 1 Satz 1 Nr. 2 UStG][ustg15]). Buchfink bildet
sie nicht vollständig ab (buchfink-grundlagen, Grenzen). Lege Rechnung und
Zollbeleg ab, buche nicht und gib den Fall an den Menschen.

## 5. Steuerfrei und nicht steuerbar

Häufige steuerfreie Eingangsleistungen ([§ 4 UStG][ustg4]):

| Leistung | Befreiung | Buchungsgruppe |
|---|---|---|
| Kredit, Zinsen | § 4 Nr. 8 Buchst. a | über Bank (buchfink-bank) |
| Kontoführung, Zahlungs- und Überweisungsverkehr | § 4 Nr. 8 Buchst. d | `geldverkehr` |
| Versicherung | § 4 Nr. 10 Buchst. a | `versicherungen` |
| Miete von Räumen ohne Option | § 4 Nr. 12 Buchst. a | `miete` |

- Verwahrung und Verwaltung von Wertpapieren sind von Buchst. e ausgenommen,
  also steuerpflichtig.
- Weist ein Vermieter Umsatzsteuer aus, hat er zur Steuerpflicht optiert; dann
  ist der Fall „Inland“.
- Nicht steuerbar ist ein Vorgang ohne Leistungsaustausch. Ob ein Beitrag oder
  eine Zahlung Entgelt für eine Leistung ist, ist eine Wertungsfrage, wenn der
  Beleg es offen lässt.

## 6. Lieferant ist Kleinunternehmer

- Umsätze eines Kleinunternehmers sind steuerfrei, wenn der Gesamtumsatz im
  Vorjahr 25.000 € nicht überschritten hat und im laufenden Jahr 100.000 € nicht
  überschreitet ([§ 19 Abs. 1 UStG][ustg19]). Stand: Oktober 2026. Die Werte
  prüft der Lieferant; du erkennst den Fall am Hinweis auf § 19 UStG
  ([§ 34a UStDV][ustdv34a]).
- Buche als steuerfrei, ohne Vorsteuer.
- Weist ein Kleinunternehmer dennoch Steuer aus, schuldet er sie
  ([§ 14c Abs. 2 UStG][ustg14c]). Für die GmbH ist sie keine abziehbare
  Vorsteuer. Vorlegen.

## 7. Ausländische Umsatzsteuer auf der Rechnung

- Abziehbar ist nur die deutsche Steuer, die für die Leistung gesetzlich
  geschuldet ist ([§ 15 Abs. 1 Satz 1 Nr. 1 UStG][ustg15]).
- Fall a, Ort im Ausland (Hotel, Mietwagen im Ausland): Die
  ausländische Steuer ist richtig. Erstattung nur über das
  Vorsteuer-Vergütungsverfahren des anderen Staates; der Antrag läuft über das
  BZSt ([§ 18g UStG][ustg18g]) und außerhalb von Buchfink. Frag den Menschen,
  ob er einen Antrag stellt. Bis dahin ist die ausländische Steuer Teil des
  Aufwands (Wertungsfrage, Steuerfall „nicht steuerbar“).
- Fall b, Ort im Inland, Lieferant hat trotzdem seine Steuer berechnet: Die
  USt-IdNr. der GmbH fehlte. Lass die Rechnung ohne ausländische Steuer und mit
  USt-IdNr. neu ausstellen und buche dann nach § 13b oder als Erwerb.

## 8. Steuersatz

- Regelsatz 19 % ([§ 12 Abs. 1 UStG][ustg12]); ermäßigt 7 % in den Fällen des
  [§ 12 Abs. 2 UStG][ustg12]. Häufig auf Eingangsbelegen:
  - Personenbeförderung im Schienenbahnverkehr; mit Taxi, Bus im Linienverkehr
    und ähnlichen innerhalb einer Gemeinde oder bis 50 km (Nr. 10).
  - Übernachtung im Hotel; nicht für Leistungen, die nicht unmittelbar der
    Vermietung dienen, auch wenn sie im Preis enthalten sind (Nr. 11).
  - Restaurant- und Verpflegungsdienstleistungen, ohne Getränke (Nr. 15, nach
    dem Wortlaut am 17.10.2026 ohne Befristung).
- Übernimm die Sätze der Rechnung je Position. Passt ein Satz nicht zum Gesetz,
  lass die Rechnung berichtigen; abziehbar ist nur die gesetzlich geschuldete
  Steuer ([§ 15 Abs. 1 Satz 1 Nr. 1 UStG][ustg15]).
- Maßgeblich ist der Tag der Leistung ([§ 27 Abs. 1 Satz 1 UStG][ustg27]).
  Buchfink führt die Sätze datiert, etwa 16 % und 5 % für Leistungen vom
  01.07. bis 31.12.2020.

[estg48]: https://www.gesetze-im-internet.de/estg/__48.html
[gobd]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf
[ustg1a]: https://www.gesetze-im-internet.de/ustg_1980/__1a.html
[ustg3a]: https://www.gesetze-im-internet.de/ustg_1980/__3a.html
[ustg3d]: https://www.gesetze-im-internet.de/ustg_1980/__3d.html
[ustg4]: https://www.gesetze-im-internet.de/ustg_1980/__4.html
[ustg12]: https://www.gesetze-im-internet.de/ustg_1980/__12.html
[ustg13]: https://www.gesetze-im-internet.de/ustg_1980/__13.html
[ustg13b]: https://www.gesetze-im-internet.de/ustg_1980/__13b.html
[ustg14a]: https://www.gesetze-im-internet.de/ustg_1980/__14a.html
[ustg14c]: https://www.gesetze-im-internet.de/ustg_1980/__14c.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
[ustg18g]: https://www.gesetze-im-internet.de/ustg_1980/__18g.html
[ustg19]: https://www.gesetze-im-internet.de/ustg_1980/__19.html
[ustg22]: https://www.gesetze-im-internet.de/ustg_1980/__22.html
[ustg27]: https://www.gesetze-im-internet.de/ustg_1980/__27.html
[ustdv34a]: https://www.gesetze-im-internet.de/ustdv_1980/__34a.html
