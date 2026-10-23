# GoBD und Ordnungsvorschriften

Rechtsstand: 17. Oktober 2026. GoBD: BMF-Schreiben vom 28.11.2019
([Text][gobd]), geändert am 11.03.2024 ([Änderung][gobd2024]) und am
14.07.2025 ([Änderung][gobd2025]); Randziffern nach der geltenden Fassung.

Die Pflichten stehen in §§ 238, 239, 257 HGB, über [§ 140 AO][ao140] auch
steuerlich, und in §§ 145 bis 147 AO; die GoBD zeigen, wie die
Finanzverwaltung sie in der Betriebsprüfung anwendet.

## Inhalt

1. Verantwortung
2. Nachvollziehbarkeit
3. Vollständigkeit
4. Richtigkeit
5. Zeitgerechte Erfassung
6. Ordnung
7. Unveränderbarkeit, Storno, Festschreibung
8. Belegwesen und Eigenbeleg
9. Internes Kontrollsystem und Datensicherheit
10. Verfahrensdokumentation
11. Folgen von Verstößen

## 1. Verantwortung

Die Geschäftsführer sorgen für die ordnungsmäßige Buchführung
([§ 41 GmbHG][gmbhg41]), auch wenn sie Aufgaben an Dritte geben
([GoBD Rz. 21][gobd]).

## 2. Nachvollziehbarkeit

- Ein sachverständiger Dritter muss sich in angemessener Zeit einen Überblick
  verschaffen und jeden Vorgang vom Beleg bis zur Erklärung und zurück
  verfolgen können ([§ 238 Abs. 1 Satz 2, 3 HGB][hgb238],
  [§ 145 Abs. 1 AO][ao145], [GoBD Rz. 30–33][gobd]).
- Folge für dich: Schreib Buchungstexte, die ohne den Chat verständlich sind
  („Toner und Papier Okt. 2026, Rechnung 4711“), und verknüpfe jede Buchung mit
  ihrem Beleg.

## 3. Vollständigkeit

- Jeder Geschäftsvorfall wird einzeln, vollzählig, lückenlos und genau einmal
  aufgezeichnet, mit Inhalt und Vertragspartner
  ([§ 146 Abs. 1 Satz 1 AO][ao146], [§ 239 Abs. 2 HGB][hgb239],
  [GoBD Rz. 36–37, 41, 43][gobd]).
- Verdichtete Buchungen sind zulässig, wenn sie sich in Einzelposten
  aufgliedern lassen und die Einzelaufzeichnungen erhalten bleiben
  ([GoBD Rz. 42, 99][gobd]); bei der Lohnbuchung aus Summen ist das Lohnjournal
  der Beleg.
- Buchfink: lückenlose Nummernkreise, Lückenprüfung im Prüflauf,
  Dublettenprüfung beim Ablegen und Buchen.

## 4. Richtigkeit

- Vorgänge werden den tatsächlichen Verhältnissen und den Rechtsvorschriften
  entsprechend belegt und kontiert ([GoBD Rz. 44][gobd]).
- Folge für dich: Kein Betrag wird „passend gemacht“. Eine Differenz zwischen
  Beleg und Zahlung (Skonto, Gebühr, Teilzahlung) bekommt eine eigene,
  begründete Zeile.

## 5. Zeitgerechte Erfassung

- Jeder Vorgang wird zeitnah erfasst ([GoBD Rz. 45–46][gobd]), unbare binnen
  zehn Tagen gelten als unbedenklich ([GoBD Rz. 47][gobd]).
- Periodenweises Buchen ist zulässig, wenn die Vorgänge vorher zeitnah
  festgehalten, gesichert und zugeordnet sind ([GoBD Rz. 50][gobd]); liegt mehr
  als der Folgemonat dazwischen, stehen sie vorher vollständig in
  Grundaufzeichnungen ([GoBD Rz. 52, 87][gobd]).
- Jeder Vorgang gehört in die Periode, in der er angefallen ist
  ([GoBD Rz. 51][gobd]).
- In Buchfink sichert `beleg_ablegen` den Beleg mit Nummer und Hashwert,
  `kontoauszug_importieren` übernimmt die Bankumsätze unverändert. Der Prüflauf
  meldet abgelegte, ungebuchte Belege und hält die Festschreibung auf.

## 6. Ordnung

- Buchungen sind systematisch, übersichtlich und eindeutig; steuerpflichtige,
  steuerfreie und nicht steuerbare Umsätze werden getrennt oder gekennzeichnet
  ([GoBD Rz. 53–55][gobd], [§ 22 UStG][ustg22]).
- Folge für dich: Wähle Steuerfall und Satz nach dem Beleg. Nicht steuerbar,
  steuerfrei, Reverse Charge und innergemeinschaftlicher Erwerb sind
  verschiedene Steuerfälle.

## 7. Unveränderbarkeit, Storno, Festschreibung

- Keine Änderung darf den ursprünglichen Inhalt oder den Zeitpunkt der
  Änderung unkenntlich machen ([§ 146 Abs. 4 AO][ao146],
  [§ 239 Abs. 3 HGB][hgb239], [GoBD Rz. 58–59, 107–108][gobd]). Einen Fehler
  korrigierst du durch Storno mit Bezug auf die Ursprungsbuchung und
  Neubuchung ([GoBD Rz. 64, 93][gobd]).
- Vorerfassungen dürfen nicht bis zum Jahresabschluss änderbar bleiben
  ([GoBD Rz. 109][gobd]). Die Festschreibung sichert die Unveränderbarkeit;
  die Grundaufzeichnung zeigt, ob und wann festgeschrieben wurde
  ([GoBD Rz. 85, 110][gobd]).
- Stammdatenänderungen, die Buchungen beeinflussen, werden protokolliert oder
  historisiert ([GoBD Rz. 111][gobd]).
- Buchfink: Journal und Protokoll sind nur fortschreibbar und hashverkettet;
  ein Storno kehrt die Ursprungsbuchung mit Verweis um
  ([arbeitsweise.md, Abschnitt 11](arbeitsweise.md#11-eine-buchung-korrigieren)).
  Einen Takt der Festschreibung nennen die GoBD nicht; Buchfink sieht sie
  monatlich nach Prüflauf und Voranmeldung vor
  ([monatsabschluss/ablauf.md](../monatsabschluss/ablauf.md)). Auslösen kann
  sie nur der Mensch; danach nimmt der Zeitraum keine Buchung mehr an.

## 8. Belegwesen und Eigenbeleg

- Jeder Vorgang ist zu belegen; fehlt ein Fremdbeleg, ist ein Eigenbeleg zu
  erstellen ([GoBD Rz. 61][gobd], [§ 238 Abs. 1 Satz 3 HGB][hgb238]).
- Mindestinhalt: eindeutige Belegnummer, Aussteller und Empfänger (soweit
  üblich), Betrag oder Menge und Wert, Währung und Kurs bei Fremdwährung,
  Erläuterung, Belegdatum ([GoBD Rz. 77][gobd]); für die Umsatzsteuer die
  Rechnungsangaben nach §§ 14, 14a UStG und § 33 UStDV ([GoBD Rz. 78][gobd],
  [belege/pflichtangaben.md](../belege/pflichtangaben.md)).
- Belege werden zeitnah gesichert, etwa durch laufende Nummerierung, und über
  eindeutige Merkmale mit der Buchung verknüpft
  ([GoBD Rz. 67–69, 71–72][gobd]).
- Strukturierte Daten können Beleg sein (CAMT-Kontoauszug, strukturierter Teil
  einer E-Rechnung); ein inhaltsgleiches PDF ist dann nicht zusätzlich
  aufzubewahren ([GoBD Rz. 76][gobd] in der Fassung der
  [Änderung 2025][gobd2025]).
- Bei Dauersachverhalten ist der Ursprungsbeleg Grundlage der Folgebuchungen,
  bei Abschreibungen der Anschaffungsbeleg ([GoBD Rz. 81][gobd]).
- Eigenbeleg in Buchfink: `buchung_erfassen` mit `eigenbeleg` und Grund, nur
  wenn kein Fremdbeleg zu beschaffen ist (verlorene Quittung, interne
  Umbuchung). Er ersetzt keine Rechnung für den Vorsteuerabzug
  ([§ 15 Abs. 1 Satz 1 Nr. 1 Satz 2 UStG][ustg15]). Ein unrichtiger Beleg oder
  eine unrichtige Buchung ist ordnungswidrig, wenn sie eine Steuerverkürzung
  ermöglicht ([§ 379 Abs. 1 Satz 1 Nr. 1, 3 AO][ao379]).

## 9. Internes Kontrollsystem und Datensicherheit

- Das Unternehmen richtet Kontrollen ein (Zugriffsrechte, Funktionstrennung,
  Erfassungs- und Plausibilitätskontrollen), übt sie aus, protokolliert sie
  und beschreibt sie in der Verfahrensdokumentation
  ([GoBD Rz. 100, 102][gobd]).
- Buchfink-Kontrollen, an denen du beteiligt bist: Direktregeln mit Grenze,
  Rückfrage beim Menschen, Prüflauf vor der Festschreibung, Festschreibung nur
  durch den Menschen, Erfassungsweg an jeder Buchung.
- Können Daten mangels Sicherung nicht vorgelegt werden, ist die Buchführung
  formell nicht ordnungsmäßig ([GoBD Rz. 103–104][gobd]). Buchfink sichert,
  sobald in der App ein Sicherungsziel eingerichtet ist.

## 10. Verfahrensdokumentation

- Jedes DV-System braucht eine Verfahrensdokumentation (allgemeine
  Beschreibung, Anwender-, System-, Betriebsdokumentation), passend zur
  Programmversion und mit Änderungshistorie ([GoBD Rz. 151–154][gobd]). Fehlt
  sie oder ist sie ungenügend, ist das nur bei beeinträchtigter
  Nachvollziehbarkeit ein formeller Mangel mit sachlichem Gewicht
  ([GoBD Rz. 155][gobd]).
- Buchfink erzeugt die technischen Teile und listet die verbundenen Apps. Die
  organisatorischen Texte entwirfst du auf Bitte mit dem Menschen: wer Belege
  in den Eingang legt, wie Papier gescannt und aufbewahrt wird, welche Vorgänge
  du buchst und wer freigibt. Lies die bestehenden Texte mit
  `verfahrensdoku_text` ohne `texte`; nach dem Ja des Menschen zum Wortlaut
  schreibst du sie mit demselben Werkzeug.

## 11. Folgen von Verstößen

- Besteht Anlass, die sachliche Richtigkeit der Buchführung zu beanstanden,
  legt das Finanzamt sie nicht mehr zugrunde und kann schätzen
  ([§ 158 AO][ao158], [§ 162 Abs. 2 AO][ao162]).
- Unrichtige Belege, unrichtiges Verbuchen und fehlende Aufbewahrung können
  eine Ordnungswidrigkeit sein ([§ 379 AO][ao379]).
- Eine abgegebene Erklärung, die sich als unrichtig erweist, ist zu berichtigen
  ([monatsabschluss/berichtigung.md, Abschnitt 2](../monatsabschluss/berichtigung.md#2-pflicht-und-recht-zur-berichtigung));
  melde den Fund sofort ([Warnzeichen](grenzen.md#2-warnzeichen-die-du-sofort-meldest)).

[gobd]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf
[gobd2024]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2024-03-11-aenderung-gobd.pdf
[gobd2025]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2025-07-14-GoBD-2-aenderung.pdf
[gmbhg41]: https://www.gesetze-im-internet.de/gmbhg/__41.html
[hgb238]: https://www.gesetze-im-internet.de/hgb/__238.html
[hgb239]: https://www.gesetze-im-internet.de/hgb/__239.html
[ao140]: https://www.gesetze-im-internet.de/ao_1977/__140.html
[ao145]: https://www.gesetze-im-internet.de/ao_1977/__145.html
[ao146]: https://www.gesetze-im-internet.de/ao_1977/__146.html
[ao158]: https://www.gesetze-im-internet.de/ao_1977/__158.html
[ao162]: https://www.gesetze-im-internet.de/ao_1977/__162.html
[ao379]: https://www.gesetze-im-internet.de/ao_1977/__379.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
[ustg22]: https://www.gesetze-im-internet.de/ustg_1980/__22.html
