# GoBD und Ordnungsvorschriften

Rechtsstand: 17. Oktober 2026.

Die GoBD sind eine Verwaltungsanweisung des BMF. Sie binden die
Finanzverwaltung und zeigen, was eine Betriebsprüfung erwartet. Die
Pflichten selbst stehen im Gesetz: §§ 238, 239, 257 HGB, über
[§ 140 AO][ao140] auch für die Steuer, und §§ 145 bis 147 AO.

Fassung: BMF-Schreiben vom 28.11.2019 (IV A 4 - S 0316/19/10003 :001, BStBl I
S. 1269), geändert am 11.03.2024 (IV D 2 - S 0316/21/10001 :002, BStBl I S. 374)
und am 14.07.2025 (IV D 2 - S 0316/00128/005/088, anzuwenden ab 14.07.2025).
Volltexte: [GoBD 2019][gobd], [Änderung 2024][gobd2024],
[Änderung 2025][gobd2025]. Die Änderung 2024 betrifft u. a. Rz. 11, 15, 39,
43, 94, 159 und 167 bis 169; die Änderung 2025 Rz. 76, 118, 119, 121, 125,
127, 131, 133, 166, 175 und fügt Rz. 185 an. Randziffern unten nach der geltenden Fassung.

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

- Die Geschäftsführer sorgen für die ordnungsmäßige Buchführung
  ([§ 41 GmbHG][gmbhg41]).
- Für die Ordnungsmäßigkeit elektronischer Bücher ist allein der
  Steuerpflichtige verantwortlich, auch wenn er Aufgaben an Dritte gibt
  ([GoBD Rz. 21][gobd]).
- Folge für dich: Du arbeitest zu, der Mensch verantwortet. Formuliere
  Ergebnisse als Vorschlag mit Begründung.

## 2. Nachvollziehbarkeit

- Ein sachverständiger Dritter muss sich in angemessener Zeit einen Überblick
  verschaffen können; jeder Vorgang muss sich von der Entstehung bis zur
  Abwicklung verfolgen lassen ([§ 238 Abs. 1 Satz 2 und 3 HGB][hgb238],
  [§ 145 Abs. 1 AO][ao145], [GoBD Rz. 30–32][gobd]).
- Die Prüfung läuft vom Beleg über Journal und Konten bis zur Erklärung und
  zurück ([GoBD Rz. 33][gobd]).
- Abkürzungen und Schlüssel brauchen eine eindeutige Bedeutung
  ([§ 239 Abs. 1 HGB][hgb239], [§ 146 Abs. 3 AO][ao146]). Buchfink führt dafür
  ein Schlüsselverzeichnis.
- Folge für dich: Schreib Buchungstexte, die ohne den Chat verständlich sind
  („Toner und Papier Okt. 2026, Rechnung 4711“), und verknüpfe jede Buchung mit
  ihrem Beleg.

## 3. Vollständigkeit

- Jeder Geschäftsvorfall wird einzeln, vollzählig und lückenlos aufgezeichnet,
  mit Inhalt und Vertragspartner ([§ 146 Abs. 1 Satz 1 AO][ao146],
  [§ 239 Abs. 2 HGB][hgb239], [GoBD Rz. 36–37][gobd]).
- Ein Vorgang darf nicht doppelt erfasst werden ([GoBD Rz. 41][gobd]).
- Verdichtete Buchungen sind zulässig, wenn sie sich in Einzelposten aufgliedern
  lassen; die Einzelaufzeichnungen aus Vor- und Nebensystemen bleiben erhalten
  ([GoBD Rz. 42, 99][gobd]). Das betrifft z. B. die Lohnbuchung aus Summen;
  das Lohnjournal des Lohnbüros ist dann der Beleg mit den Einzelposten.
- Vorgänge dürfen nicht unterdrückt werden ([GoBD Rz. 43][gobd]).
- Buchfink: lückenlose Nummernkreise, Lückenprüfung im Prüflauf,
  Dublettenprüfung beim Ablegen und Buchen.

## 4. Richtigkeit

- Vorgänge werden den tatsächlichen Verhältnissen und den Rechtsvorschriften
  entsprechend durch Belege abgebildet und zutreffend kontiert
  ([GoBD Rz. 44][gobd]).
- Folge für dich: Ein Betrag wird nie „passend gemacht“. Differenzen zwischen
  Beleg und Zahlung (Skonto, Gebühr, Teilzahlung) bekommen eine eigene,
  begründete Zeile.

## 5. Zeitgerechte Erfassung

- Jeder Vorgang wird zeitnah, möglichst unmittelbar nach Entstehung erfasst;
  Belege nur zu sammeln und lange danach zu buchen widerspricht der Buchführung
  ([GoBD Rz. 45–46][gobd]).
- Unbare Vorgänge: Erfassung binnen zehn Tagen ist unbedenklich
  ([GoBD Rz. 47][gobd]).
- Kasseneinnahmen und -ausgaben täglich ([§ 146 Abs. 1 Satz 2 AO][ao146],
  [GoBD Rz. 48][gobd]). Buchfink führt keine Kasse.
- Periodenweises Buchen ist zulässig, wenn die Vorgänge vorher zeitnah
  festgehalten, gegen Verlust gesichert und zugeordnet sind
  ([GoBD Rz. 50][gobd]). Liegt zwischen Erfassung und Buchung mehr als der
  Folgemonat, müssen die Vorgänge vorher vollständig in Grundaufzeichnungen
  stehen ([GoBD Rz. 52, 87][gobd]).
- Jeder Vorgang gehört in die Periode, in der er angefallen ist
  ([GoBD Rz. 51][gobd]).
- In Buchfink: `beleg_ablegen` sichert den Beleg aus dem Eingang mit Nummer und
  Hashwert; `kontoauszug_importieren` übernimmt die Bankumsätze unverändert.
  Dateien, die lange im Eingang liegen, meldet Buchfink als Hinweis. Der
  Prüflauf meldet abgelegte, nicht gebuchte Belege und hält damit die
  Festschreibung auf.
- Arbeitstakt, der diese Fristen einhält: Eingang am Eingangstag übernehmen,
  Kontoauszug mindestens alle zehn Tage importieren, alles bis Ende des
  Folgemonats buchen, danach den Monat festschreiben lassen.

## 6. Ordnung

- Buchungen sind systematisch, übersichtlich und eindeutig
  ([GoBD Rz. 53–54][gobd]).
- Steuerpflichtige, steuerfreie und nicht steuerbare Umsätze werden getrennt
  oder gekennzeichnet gebucht ([GoBD Rz. 55][gobd], [§ 22 UStG][ustg22]).
- Folge für dich: Wähle Steuerfall und Satz nach dem Beleg. „Ohne Steuer“ ist
  kein Sammelbecken: nicht steuerbar, steuerfrei, Reverse Charge und
  innergemeinschaftlicher Erwerb sind verschiedene Steuerfälle.

## 7. Unveränderbarkeit, Storno, Festschreibung

- Eine Buchung darf nicht so verändert werden, dass der ursprüngliche Inhalt
  nicht mehr feststellbar ist oder unklar bleibt, wann geändert wurde
  ([§ 146 Abs. 4 AO][ao146], [§ 239 Abs. 3 HGB][hgb239],
  [GoBD Rz. 58–59, 107–108][gobd]).
- Fehler werden durch Storno und Neubuchung korrigiert; weitere Änderungen
  einer Buchung sind nicht nötig ([GoBD Rz. 93][gobd]). Der Storno muss auf die
  Ursprungsbuchung rückbeziehbar sein ([GoBD Rz. 64][gobd]).
- Unzulässig ist es, Vorerfassungen bis zum Jahresabschluss offen und änderbar
  zu halten ([GoBD Rz. 109][gobd]). Festschreibung ist eines der Mittel, die
  Unveränderbarkeit zu sichern ([GoBD Rz. 110][gobd]); die Grundaufzeichnung
  muss erkennen lassen, ob und wann festgeschrieben wurde
  ([GoBD Rz. 85][gobd]).
- Stammdatenänderungen, die Buchungen beeinflussen, müssen protokolliert oder
  historisiert sein ([GoBD Rz. 111][gobd]).
- Buchfink: Das Journal ist nur fortschreibbar und durch eine Hashkette
  gesichert; Änderungen am Protokoll ebenso. Ein Storno ist eine
  Generalumkehr mit dem Datum seiner Erstellung und Verweis auf die
  Ursprungsbuchung. Die Festschreibung eines Zeitraums verankert den
  Kettenstand mit dem Zeitstempel eines externen Dienstes (RFC 3161); danach nimmt der Zeitraum
  keine Buchung mehr an.
- Die GoBD nennen keinen festen Takt für die Festschreibung. Buchfink empfiehlt,
  jeden Monat nach Prüflauf und Voranmeldung festzuschreiben. Die
  Festschreibung löst nur der Mensch in der App aus.

## 8. Belegwesen und Eigenbeleg

- Jeder Vorgang ist zu belegen. Fehlt ein Fremdbeleg, ist ein Eigenbeleg zu
  erstellen ([GoBD Rz. 61][gobd], [§ 238 Abs. 1 Satz 3 HGB][hgb238]).
- Mindestinhalt eines Belegs: eindeutige Belegnummer, Aussteller und Empfänger
  (soweit üblich), Betrag oder Mengen- und Wertangaben, Währung und Kurs bei
  Fremdwährung, hinreichende Erläuterung, Belegdatum
  ([GoBD Rz. 77][gobd]). Für die Umsatzsteuer kommen die Rechnungsangaben
  nach §§ 14, 14a UStG und § 33 UStDV hinzu ([GoBD Rz. 78][gobd]); siehe
  buchfink-belege.
- Belege werden zeitnah gegen Verlust gesichert, z. B. durch laufende
  Nummerierung ([GoBD Rz. 67–69][gobd]); Beleg und Buchung sind über eindeutige
  Merkmale verknüpft ([GoBD Rz. 71–72][gobd]).
- Strukturierte Daten können die Belegfunktion tragen, etwa der Kontoauszug als
  CAMT-Datei oder der strukturierte Teil einer E-Rechnung; ein inhaltsgleiches
  PDF ist dann nicht zusätzlich aufzubewahren ([GoBD Rz. 76][gobd], geändert
  durch [Änderung 2025][gobd2025]).
- Bei Dauersachverhalten ist der Ursprungsbeleg die Grundlage der
  Folgebuchungen; bei Abschreibungen der Anschaffungsbeleg
  ([GoBD Rz. 81][gobd]).
- Eigenbeleg in Buchfink: `buchung_erfassen` mit `eigenbeleg` und Grund. Nutze
  ihn nur, wenn kein Fremdbeleg zu beschaffen ist (z. B. verlorene Quittung,
  interne Umbuchung). Ein Eigenbeleg ersetzt keine Rechnung für den
  Vorsteuerabzug ([§ 15 Abs. 1 Satz 1 Nr. 1 Satz 2 UStG][ustg15]). Ein Eigenbeleg
  muss wahr sein: unrichtige Belege auszustellen oder Vorgänge unrichtig zu
  verbuchen ist ordnungswidrig, wenn es eine Steuerverkürzung ermöglicht
  ([§ 379 Abs. 1 Satz 1 Nr. 1 und 3 AO][ao379]).

## 9. Internes Kontrollsystem und Datensicherheit

- Der Steuerpflichtige richtet Kontrollen ein, übt sie aus und protokolliert
  sie, z. B. Zugriffsrechte, Funktionstrennung, Erfassungs- und
  Plausibilitätskontrollen ([GoBD Rz. 100][gobd]). Die Beschreibung gehört in die
  Verfahrensdokumentation ([GoBD Rz. 102][gobd]).
- Buchfink-Kontrollen, an denen du beteiligt bist: Direktregeln mit Grenze,
  Rückfrage beim Menschen, Prüflauf vor der Festschreibung, Festschreibung nur
  durch den Menschen, Erfassungsweg an jeder Buchung.
- Das System ist gegen Verlust und unberechtigte Änderung zu sichern; können
  Daten deshalb nicht vorgelegt werden, ist die Buchführung formell nicht
  ordnungsmäßig ([GoBD Rz. 103–104][gobd]). Buchfink sichert automatisch, wenn
  in der App ein Sicherungsziel eingerichtet ist.

## 10. Verfahrensdokumentation

- Für jedes DV-System ist eine verständliche Verfahrensdokumentation nötig:
  allgemeine Beschreibung, Anwender-, System- und Betriebsdokumentation
  ([GoBD Rz. 151–153][gobd]). Sie muss zur eingesetzten Programmversion passen
  und Änderungen historisch nachweisen ([GoBD Rz. 154][gobd]).
- Eine fehlende oder ungenügende Verfahrensdokumentation ist nur dann ein
  formeller Mangel mit sachlichem Gewicht, wenn sie die Nachvollziehbarkeit
  beeinträchtigt ([GoBD Rz. 155][gobd]). Darauf verlässt sich niemand.
- Buchfink erzeugt die technischen Teile und listet die verbundenen Apps.
  Die organisatorischen Texte entwirfst du auf Bitte mit dem Menschen: wer
  Belege in den Eingang legt, wie Papier gescannt und aufbewahrt wird, welche
  Vorgänge du buchst und wer freigibt. Ein Werkzeug für die
  Organisationstexte gibt es noch nicht; der Mensch überträgt deinen Entwurf
  in der App.

## 11. Folgen von Verstößen

- Eine ordnungsmäßige Buchführung wird der Besteuerung zugrunde gelegt
  ([§ 158 Abs. 1 AO][ao158]). Besteht Anlass, ihre sachliche Richtigkeit zu
  beanstanden, entfällt das ([§ 158 Abs. 2 Nr. 1 AO][ao158]); dann kann das
  Finanzamt schätzen ([§ 162 Abs. 2 AO][ao162]).
- Unrichtige Belege, unrichtiges Verbuchen und fehlende Aufbewahrung können
  eine Ordnungswidrigkeit sein ([§ 379 AO][ao379]).
- Erkennt das Unternehmen nachträglich, dass eine abgegebene Erklärung
  unrichtig ist und Steuer verkürzt sein kann, muss es das unverzüglich anzeigen
  und richtigstellen ([§ 153 Abs. 1 AO][ao153]). Melde solche Funde sofort dem
  Menschen.

[gobd]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf
[gobd2024]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2024-03-11-aenderung-gobd.pdf
[gobd2025]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2025-07-14-GoBD-2-aenderung.pdf
[gmbhg41]: https://www.gesetze-im-internet.de/gmbhg/__41.html
[hgb238]: https://www.gesetze-im-internet.de/hgb/__238.html
[hgb239]: https://www.gesetze-im-internet.de/hgb/__239.html
[ao140]: https://www.gesetze-im-internet.de/ao_1977/__140.html
[ao145]: https://www.gesetze-im-internet.de/ao_1977/__145.html
[ao146]: https://www.gesetze-im-internet.de/ao_1977/__146.html
[ao153]: https://www.gesetze-im-internet.de/ao_1977/__153.html
[ao158]: https://www.gesetze-im-internet.de/ao_1977/__158.html
[ao162]: https://www.gesetze-im-internet.de/ao_1977/__162.html
[ao379]: https://www.gesetze-im-internet.de/ao_1977/__379.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
[ustg22]: https://www.gesetze-im-internet.de/ustg_1980/__22.html
