# Anforderungskatalog

[Dokumentation](../README.md) · [Fachkonzepte](../fachkonzepte/README.md) ·
[Umsetzungsstand](../projekt/umsetzungsstand.md) · [Roadmap](../projekt/roadmap.md)

Der Katalog verbindet die gesetzlichen Anforderungen an eine Buchführung mit
Prüfkriterien und mit der Stelle im Code, an der Buchfink sie erfüllt. Jedes
Kriterium ist so formuliert, dass es sich an der laufenden Software prüfen
lässt. Bewertet ist der Stand von Version 0.3. Die IDs sind stabil; Code und
Tests verweisen auf sie.

Der Katalog ist eine Prüfhilfe. Er sagt nicht, dass eine Buchführung mit
Buchfink ordnungsgemäß ist; das hängt auch davon ab, wie das Unternehmen
arbeitet.

## Geltungsbereich

| Merkmal | Annahme |
|---|---|
| Rechtsform | GmbH und UG (haftungsbeschränkt) mit Sitz in Deutschland |
| Buchführung | doppelt, Bilanz nach §§ 238 ff. HGB, SKR04 |
| Größe | Kleinst- und kleine Kapitalgesellschaft nach §§ 267, 267a HGB |
| Geschäft | Dienstleistung, Software, Agentur, Projektgeschäft, Holding |
| Besteuerung | Sollversteuerung, keine Kleinunternehmerregelung |

Was Buchfink bewusst nicht abbildet, steht in [Umfang](../nutzung/umfang.md).
Im Katalog haben diese Kriterien den Stand „nicht im Umfang“.

## Legende

| Verbindlichkeit | Bedeutung |
|---|---|
| MUSS | gesetzliche Pflicht |
| MUSS* | Pflicht, sobald der Sachverhalt vorliegt |
| SOLL | Verwaltungsauffassung oder Prüfungserwartung |
| TERMIN | Pflicht mit künftigem Stichtag, siehe [Terminplan](#terminplan) |

| Stand | Bedeutung |
|---|---|
| erfüllt | Die Fundstelle nennt Paket, Funktion oder Test, an dem es nachzuprüfen ist. |
| teilweise | Die Spalte nennt den vorhandenen und den fehlenden Teil. |
| offen | Fehlt. |
| nicht im Umfang | Bewusst nicht gebaut, mit Grund. |

Fundstellen sind Paket, Typ oder Funktion, etwa
`service.JournalService.Post` für die Methode `Post` des Typs `JournalService`
in `internal/service`. Prüfregeln stehen mit ihrem Code, etwa
`entry_without_receipt`; definiert sind sie in `internal/domain`, gerechnet in
`service.CheckService`.

## Module

| Modul | Kernnormen | Anforderungen |
|---|---|---|
| [A. Buchführungspflicht und Grundsätze](grundsaetze.md) | §§ 238, 239, 244 HGB, §§ 140, 145, 146 AO | GOB-01 bis GOB-06 |
| [B. Beleg, Journal, Konten](beleg-journal-konten.md) | § 238 HGB, § 146 AO, GoBD Rz 61 bis 99 | BEL-01 bis BEL-09 |
| [C. Unveränderbarkeit und Protokollierung](unveraenderbarkeit.md) | § 239 Abs. 3 HGB, § 146 Abs. 4 AO, GoBD Rz 100 bis 112 | UNV-01 bis UNV-06 |
| [D. Aufbewahrung und Archivierung](aufbewahrung.md) | § 257 HGB, § 147 AO, § 14b UStG | ARC-01 bis ARC-08 |
| [E. Ausgangsrechnungen und E-Rechnung](rechnungen.md) | §§ 14, 14a, 14c UStG, §§ 33, 34 UStDV | RECH-01 bis RECH-10 |
| [F. Umsatzsteuer und Meldungen](umsatzsteuer.md) | §§ 13b, 15, 15a, 18, 18a, 22 UStG | UST-01 bis UST-09 |
| [G. Bewertung, Anlagen, Fremdwährung](bewertung.md) | §§ 252 bis 256a HGB, §§ 5 bis 7g EStG | BEW-01 bis BEW-13 |
| [H. Jahresabschluss, E-Bilanz, Offenlegung](jahresabschluss.md) | §§ 242 bis 289, 325 HGB, § 5b EStG, § 42a GmbHG | JAB-01 bis JAB-09 |
| [I. Betriebsprüfung und Verfahrensdokumentation](betriebspruefung.md) | §§ 147, 147b, 158 AO, GoBD Rz 151 bis 177 | PRF-01 bis PRF-06 |
| [J. Querschnitt](querschnitt.md) | DSGVO, BGB, GmbHG | QUE-01 bis QUE-06 |

Die GoBD sind eine Verwaltungsanweisung, kein Gesetz. Die Randziffern beziehen
sich auf das BMF-Schreiben vom 28.11.2019 in der Fassung vom 11.03.2024 und vom
14.07.2025.

## Offene Kriterien

| ID | Kriterium |
|---|---|
| BEW-01 | Bericht über geänderte Bewertungsmethoden |
| BEW-02 | Pflicht- und Wahlbestandteile der Herstellungskosten getrennt |
| BEW-03, JAB-05 | Anlagenspiegel und Anlagenverzeichnis in der E-Bilanz |
| JAB-04 | unterzeichneten Abschluss unveränderlich ablegen |
| JAB-07 | Datensatz für das Unternehmensregister, Hinterlegung für Kleinstgesellschaften |
| JAB-08 | Saldenbestätigungen, Prüfungsbericht am Abschluss |
| PRF-02 | Testeinlesen in eine Prüfsoftware |

## Was hier nicht gefordert ist

Diese Punkte stehen oft in Anforderungslisten, gelten für eine
Finanzbuchhaltung im beschriebenen Geltungsbereich aber nicht.

- **Technische Sicherheitseinrichtung nach KassenSichV.** Buchhaltungsprogramme
  sind ausgenommen (§ 1 Abs. 1 Nr. 3 KassenSichV); Buchfink hat keine Kasse.
- **Qualifizierte Signatur auf Rechnungen.** Ein innerbetriebliches
  Kontrollverfahren genügt (§ 14 Abs. 3 UStG, RECH-08).
- **XBRL bei der Offenlegung.** Pflicht nur für Inlandsemittenten
  (§ 328 Abs. 1 S. 4 HGB). XBRL braucht die E-Bilanz.
- **Nachhaltigkeitsbericht.** Nach der Richtlinie (EU) 2026/470 erst ab 1.000
  Beschäftigten und 450 Mio. Euro Umsatz.
- **Geldwäscherecht nach § 2 GwG.** Eine gewöhnliche GmbH ist keine
  Verpflichtete. Die Mitteilung an das Transparenzregister (§ 20 GwG) führt
  Buchfink als Pflicht im Pflichtenkalender.
- **Buchführungsgrenzen nach § 141 AO.** Kapitalgesellschaften buchen als
  Formkaufleute immer.

## Terminplan

| Datum | Pflicht | Anforderung | Norm |
|---|---|---|---|
| 31.12.2026 | Ende der Übergangsfrist für Papier- und sonstige Rechnungen | RECH-06 | § 27 Abs. 38 Nr. 1 UStG |
| 01.01.2027 | Sendepflicht für E-Rechnungen bei Vorjahresumsatz über 800.000 Euro | RECH-06 | § 27 Abs. 38 Nr. 2 UStG |
| 01.01.2027 | monatliche Voranmeldung für Neugründer wieder Pflicht | UST-03 | § 18 Abs. 2 S. 4 UStG |
| WJ ab 01.01.2027 | Taxonomie 6.10 | JAB-05 | BMF-Schreiben vom 08.06.2026 |
| 01.01.2028 | Sendepflicht für alle inländischen B2B-Umsätze | RECH-06 | § 27 Abs. 38 UStG |
| WJ ab 01.01.2028 | Anlagenspiegel und Anlagenverzeichnis in der E-Bilanz | BEW-03, JAB-05 | § 5b Abs. 1 EStG |
| 01.07.2030 | ViDA: E-Rechnung und Meldung für innergemeinschaftliche B2B-Umsätze | RECH-06, UST-04 | Richtlinie (EU) 2025/516 |
| offen | Buchführungsdatenschnittstelle, Verordnung nicht erlassen | PRF-06 | § 147b AO |

Die ViDA-Termine stammen aus der Richtlinie; das deutsche Umsetzungsgesetz
steht aus.

## Quellen

**Gesetze und Verordnungen:** HGB, EGHGB, AO, EGAO, EStG, EStDV, UStG, UStDV,
KassenSichV, BGB, GmbHG, GwG, DSGVO, Richtlinie 2014/55/EU, Richtlinie (EU)
2025/516, Richtlinie (EU) 2026/470, alle nach gesetze-im-internet.de und
EUR-Lex.

**Verwaltungsanweisungen:**

- GoBD, BMF-Schreiben vom 28.11.2019, geändert am 11.03.2024 und 14.07.2025
- E-Rechnung, BMF-Schreiben vom 15.10.2024, ergänzt am 15.10.2025 (Format-,
  Geschäftsregel- und Inhaltsfehler, Rn. 35a)
- Bewirtungsaufwendungen, BMF-Schreiben vom 19.11.2025, für Bewirtungen ab
  1.1.2025
- E-Bilanz-Taxonomien 6.9 (BMF-Schreiben vom 10.06.2025) und 6.10
  (BMF-Schreiben vom 08.06.2026)
- AfA-Tabelle AV vom 15.12.2000; Nutzungsdauer von Hardware und Software,
  BMF-Schreiben vom 22.02.2022
- Entwurf der Buchführungsdatenschnittstellenverordnung (DSFinVBV), Fassung 2026

Der Katalog ersetzt keine steuerliche oder rechtliche Beratung. Vor dem
Einsatz sollten eine Steuerberaterin oder ein Wirtschaftsprüfer die Module C,
D, E und I gegenlesen; dort liegen die Beanstandungsrisiken einer
Betriebsprüfung.
