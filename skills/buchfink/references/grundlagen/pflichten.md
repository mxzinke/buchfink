# Pflichtenkalender

Rechtsstand: 17. Oktober 2026. Wortlaut von AO, EGAO, UStG, UStDV, EStG, EStDV,
KStG, SolZG, GewStG, HGB, GmbHG, GwG, TrGebV, GewO, GewAnzV, SGB VII, StGB,
OWiG und BGB am 17.10.2026 gelesen, für die Eröffnungsbilanz § 5b EStG,
§§ 242, 243, 257 HGB und § 283b StGB am 19.10.2026. GoBD: BMF-Schreiben vom 28.11.2019 in der
Fassung vom 14.07.2025 ([Text 2019][gobd], Änderungen [2024][gobd2024] und
[2025][gobd2025]); die hier zitierten Randziffern sind unverändert. Wege nach
den Seiten von ELSTER, BZSt, Publikations-Plattform des Unternehmensregisters,
Transparenzregister, eBilanz+ und eBilanz-Online vom selben Tag.

Jeder Abschnitt einer Pflicht beginnt mit einem Anker `<a id="<anker>"></a>`.
`fristen` nennt ihn je Pflicht im Feld `anker` und im `verweis`
(`buchfink/references/grundlagen/pflichten.md#<anker>`). Meist ist der Anker
die Art der Pflicht. Bei fünf Arten weicht er ab: `festschreibung` hat den Anker
`monatsfestschreibung`, `abschlussbuchungen` `bilanzstichtag`, `e-bilanz`
`ebilanz`, `e-bilanz-eroeffnung` `ebilanz-eroeffnung` und
`steuerliche-erfassung` `fragebogen-steuerliche-erfassung`.
Suche im Text nach `id="<anker>"`.

## Inhalt

1. [Arbeiten mit `fristen`](#arbeiten-mit-fristen)
2. [Fristende, Zahlung, Verlängerung](#fristende-zahlung-verlängerung)
3. Laufende Pflichten: [ustva](#ustva), [sondervorauszahlung](#sondervorauszahlung),
   [zm](#zm), [kst-vorauszahlung](#kst-vorauszahlung),
   [gewst-vorauszahlung](#gewst-vorauszahlung),
   [lohnsteuer-anmeldung](#lohnsteuer-anmeldung),
   [monatsfestschreibung](#monatsfestschreibung)
4. Jahresabschluss: [bilanzstichtag](#bilanzstichtag), [aufstellung](#aufstellung),
   [feststellung](#feststellung), [ergebnisverwendung](#ergebnisverwendung),
   [offenlegung](#offenlegung)
5. Jahreserklärungen: [ebilanz](#ebilanz), [kst-erklaerung](#kst-erklaerung),
   [gewst-erklaerung](#gewst-erklaerung), [ust-erklaerung](#ust-erklaerung)
6. Anlässe und Gründung: [transparenzregister](#transparenzregister),
   [gesellschafterliste](#gesellschafterliste),
   [fragebogen-steuerliche-erfassung](#fragebogen-steuerliche-erfassung),
   [gewerbeanmeldung](#gewerbeanmeldung), [eroeffnungsbilanz](#eroeffnungsbilanz),
   [ebilanz-eroeffnung](#ebilanz-eroeffnung), [kapitalertragsteuer](#kapitalertragsteuer)

## Arbeiten mit `fristen`

`fristen` liefert jede Pflicht der Gesellschaft mit `schluessel` (etwa
`ustva.2026-Q1`), Art, Zeitraum, Fälligkeit, Status, Kanal, Zuständigkeit und
Anker des Abschnitts hier. Buchfink
berechnet die Fälligkeit aus Stammdaten und Buchführung. Den Stand liest es aus
den Daten (Transferticket, Festschreibung, Abschlussstand, gebuchte Zahlung).
Was Buchfink nicht sieht, bestätigt der Mensch in der App mit Nachweis.

| Status | Bedeutung | Dein Vorgehen |
|---|---|---|
| offen | Frist läuft | vorbereiten, soweit die Zuständigkeit es vorsieht |
| vorbereitet | Unterlagen fertig, Bestätigung fehlt | den Menschen an Abgabe und Bestätigung erinnern |
| ueberfaellig | Fälligkeit verstrichen, kein Nachweis | sofort ansprechen (unten) |
| erledigt | Nachweis liegt vor | nichts |
| entfaellt | der Mensch hat sie mit Grund gekennzeichnet | nichts |

Zuständigkeit (`zustaendig`):

- **`agent_bereitet_vor`:** Du stellst Zahlen und Dateien zusammen. Der Mensch
  gibt ab und bestätigt.
- **`mensch`:** Er handelt selbst (Zahlung, Beschluss, Register, Behörde). Du
  lieferst Zahlen, Entwürfe und Erinnerungen.
- **`steuerberater`:** Der Berater erstellt und übermittelt. Du stellst ihm die
  Unterlagen zusammen.

Ablauf:

1. **Abrufen.** Rufe `fristen` zu Sitzungsbeginn nach `status` und `aufgaben`
   auf und jedes Mal, wenn der Mensch nach Terminen fragt.
2. **Wählen.** Überfällig vor offen, innerhalb davon nach Fälligkeit.
3. **Nachschlagen.** Lies den Abschnitt aus dem Verweis und die dort unter
   „Vertiefung“ genannte Referenz.
4. **Vorbereiten.** Nach der Referenz. Fertig, wenn jede Zahl geprüft ist,
   der Mensch die Zahlen bestätigt hat und Exportdateien im Exportordner liegen.
5. **Vorbereitet melden.** `pflicht_vorbereitet` mit dem `schluessel` aus
   `fristen` und einer `notiz`: was vorbereitet ist, wo die Datei liegt.
   „Erledigt“ meldest du nie; diesen Stand setzen die Daten oder der Mensch.
6. **Übergeben.** Nenne dem Menschen in einer Nachricht: Pflicht und Zeitraum,
   Fälligkeit, Weg (Kanal, Formular), Werte oder Datei, den Nachweis, den er
   danach in der App erfasst.
7. **Nachhalten.** In der nächsten Sitzung zeigt `fristen` den Stand. Steht die
   Pflicht noch auf vorbereitet, frag nach.

Dein Teil einer Pflicht ist fertig, wenn sie auf „vorbereitet“ steht und der
Mensch Weg, Fälligkeit und Nachweis kennt.

- **Überfällig:** Frag zuerst, ob der Mensch die Pflicht außerhalb von Buchfink
  erfüllt hat. Dann bittest du ihn, sie mit Nachweis zu bestätigen. Sonst
  nennst du die Folgen aus dem Abschnitt und den kürzesten Weg.
- **Entfällt:** setzt nur der Mensch in der App, mit Grund. Typisch:
  Gewerbeanzeige bei rein freiberuflichem Gegenstand; Transparenzregister nach
  einem Gesellschafterwechsel, der die wirtschaftlich Berechtigten nicht ändert.
- **Anlässe:** Gesellschafterwechsel, Änderung der wirtschaftlich Berechtigten,
  Vorauszahlungsbescheid. Buchfink bildet die Pflichten daraus erst, wenn der
  Anlass erfasst ist. Lege Bescheid oder Vertrag mit `beleg_ablegen` ab und
  erfasse den Anlass nach Rückfrage mit `anlass_erfassen` (`art`:
  `gesellschafterwechsel`, `wirtschaftlich-berechtigte`,
  `vorauszahlungsbescheid-kst`, `vorauszahlungsbescheid-gewst`; `datum`;
  `notiz` mit Fundstelle des Nachweises).
- **Stammdaten** (Bundesland, Voranmeldungszeitraum, Dauerfristverlängerung,
  Gründung, Eintragung, Gesellschafter, steuerliche Beratung) ändert nur der
  Mensch in der App. Ist eines falsch, ist die Fälligkeit falsch: Bitte ihn um
  die Korrektur.

Mit und ohne Steuerberater:

- Das Stammdatum „steuerliche Beratung“ sagt, ob und ab welchem Jahr ein Berater
  welche Jahreserklärungen erstellt. Davon hängen Fälligkeit und Zuständigkeit
  von Körperschaftsteuer-, Gewerbesteuer- und Umsatzsteuererklärung und
  E-Bilanz ab.
- Die längere Frist gilt nur, wenn der Berater mit der Erstellung beauftragt ist
  ([§ 149 Abs. 3 AO][ao149]). Ist der Berater eingetragen, aber nicht
  beauftragt, bitte den Menschen, das Stammdatum zu korrigieren.
- Voranmeldung, ZM, Lohnsteuer-Anmeldung, Vorauszahlungen und alle Pflichten
  aus HGB, GmbHG, GwG und GewO haben mit und ohne Berater dieselbe Frist.
- Mit Berater stellst du ihm Saldenliste, Bilanz, GuV, Anlagenspiegel und die
  offenen Fragen zusammen. Erledigt bestätigt der Mensch mit dem Transferticket,
  das der Berater ihm nennt.

## Fristende, Zahlung, Verlängerung

- **Steuerliche Fristen:** Fällt das Ende auf Samstag, Sonntag oder einen
  gesetzlichen Feiertag, endet die Frist mit Ablauf des nächsten Werktags
  ([§ 108 Abs. 3 AO][ao108]). Das gilt auch für Zahlungstermine:
  „Fälligkeitstermine geben das Ende einer Frist an“
  ([AEAO zu § 108 Nr. 1][aeao108]; Spiegel einer Steuerkanzlei, die amtliche
  Fassung des BMF war am 17.10.2026 nicht abrufbar). Für die Gewerbesteuer gilt
  § 108 über [§ 1 Abs. 2 Nr. 3 AO][ao1]. Buchfink nimmt die Feiertage des
  Bundeslands aus den Stammdaten und nennt gesetzliches und verschobenes Datum.
  Ausnahme: Für KSt- und GewSt-Vorauszahlungen zeigt `fristen` nur das
  gesetzliche Datum. Fällt es auf Samstag, Sonntag oder Feiertag, nenne dem
  Menschen den nächsten Werktag als Fristende.
- **Fristen aus HGB, GmbHG, GwG und GewO** (Aufstellung, Feststellung,
  Ergebnisverwendung, Offenlegung, Eröffnungsbilanz, Transparenzregister,
  Gesellschafterliste, Gewerbeanzeige): § 108 Abs. 3 AO gilt dort nicht. Ob [§ 193 BGB][bgb193]
  das Ende auf den nächsten Werktag verschiebt, ist für diese Pflichten nicht
  geklärt. Buchfink rechnet vorsichtshalber ohne Verschiebung; plane mit dem
  Kalenderdatum.
- **Unverzüglich** heißt ohne schuldhaftes Zögern
  ([§ 121 Abs. 1 Satz 1 BGB][bgb121]). Buchfink setzt die Fälligkeit auf den Tag
  des Anlasses.
- **Zahlung:** Eine Überweisung gilt am Tag der Gutschrift beim Finanzamt als
  entrichtet, eine Lastschrift aus SEPA-Mandat am Fälligkeitstag
  ([§ 224 Abs. 2 Nr. 2, 3 AO][ao224]).
- **Säumniszuschlag:** 1 % je angefangenem Monat auf den auf 50 € abgerundeten
  Rückstand. Bei einer Säumnis bis zu drei Tagen wird er nicht erhoben, außer
  bei Zahlung mit Bargeld oder Scheck ([§ 240 Abs. 1, 3 AO][ao240]).
- **Verspätungszuschlag:** Regeln und Höhe stehen in
  [steuererklaerungen/fristen-zuschlaege.md](../steuererklaerungen/fristen-zuschlaege.md#4-verspätungszuschlag),
  Einzelheiten je Pflicht im Abschnitt.
- **Verlängerung:** Erklärungsfristen verlängert das Finanzamt auf Antrag, auch
  rückwirkend ([§ 109 Abs. 1 AO][ao109]). Im Berater-Fall geht das über den
  letzten Februartag hinaus nur ohne Verschulden, nach einer Vorabanforderung
  ebenso über den angeordneten Termin hinaus
  ([§ 109 Abs. 2 Satz 1 Nr. 1, 2 AO][ao109]).
  Antrag in Mein ELSTER: „Antrag auf Fristverlängerung“ im Bereich „Anträge und
  Nachrichten“.
- **Mein ELSTER:** Jedes Formular steht unter „Formulare und Leistungen“ →
  „Alle Formulare“, gruppiert nach Steuerart ([Formularliste][elsterformulare]).
  Organisationszertifikat, Eingabe, Prüfen, Senden und Ablage des
  Übermittlungsprotokolls mit dem Transferticket:
  [steuererklaerungen/abgabe.md](../steuererklaerungen/abgabe.md#4-eingabe-in-mein-elster).

<a id="ustva"></a>
## Umsatzsteuer-Voranmeldung

- **Grundlage:** [§ 18 Abs. 1, 2, 2a UStG][ustg18], [§ 46 UStDV][ustdv46].
- **Frist:** Übermittlung und Zahlung am 10. Tag nach Ablauf des
  Voranmeldungszeitraums ([§ 18 Abs. 1 Satz 1, 4 UStG][ustg18]); mit
  Dauerfristverlängerung einen Monat später ([§ 46 Satz 1 UStDV][ustdv46]).
  § 108 Abs. 3 AO gilt. Mit Berater gleich.
- **Zeitraum:** Vierteljahr; Monat bei Vorjahressteuer über 9.000 €; Wahl des
  Monats bei einem Vorjahresüberschuss über 9.000 €; Sonderregeln für
  Neugründungen ([§ 18 Abs. 2, 2a UStG][ustg18]; Vertiefung). Den Zeitraum
  stellt der Mensch in der App ein.
- **Zuständig:** du bereitest vor; der Mensch übermittelt. Kanal: ELSTER.
- **Weg:**
  1. `erklaerung` (`art: ustva`, `zeitraum`) prüfen und dem Menschen die
     Kennzahlen als Tabelle vorlegen.
  2. Mein ELSTER → „Formulare und Leistungen“ → „Alle Formulare“ →
     „Umsatzsteuer“ → „Umsatzsteuer-Voranmeldung“, Jahr wählen.
  3. Der Mensch trägt Zeitraum und Kennzahlen ein, lässt prüfen, vergleicht mit
     der Tabelle und sendet.
  4. Zahllast bis zur Fälligkeit zahlen oder per Lastschrift einziehen lassen.
- **Buchfink liefert:** `erklaerung` mit Kennzahl, Wert, Herkunft, Zahllast und
  Befunden; dieselbe Ansicht formularnah in der App.
- **Nachweis:** Transferticket. Der Mensch erfasst es in der App an der
  Voranmeldung; Buchfink nimmt es erst nach der Festschreibung des Zeitraums
  an. Danach steht die Pflicht aus den Daten auf erledigt. Den letzten Monat
  des Zeitraums muss der Mensch deshalb spätestens am Fälligkeitstag
  festschreiben ([monatsfestschreibung](#monatsfestschreibung)).
- **Bei Versäumnis:** Verspätungszuschlag nach Dauer, Häufigkeit und Höhe
  ([§ 152 Abs. 8 Nr. 1 AO][ao152]); Säumniszuschlag ([§ 240 AO][ao240]);
  Geldbuße bis 30.000 €, wenn die Vorauszahlung vorsätzlich nicht oder nicht
  rechtzeitig entrichtet wird ([§ 26a Abs. 1, 3 UStG][ustg26a]); Abs. 1 nennt
  keine Leichtfertigkeit, geahndet wird also nur Vorsatz
  ([§ 10 OWiG][owig10] über [§ 377 Abs. 2 AO][ao377]).
- **Vertiefung:** [monatsabschluss/ustva.md](../monatsabschluss/ustva.md)

<a id="sondervorauszahlung"></a>
## Sondervorauszahlung

- **Grundlage:** [§§ 47, 48 UStDV][ustdv47]. Nur bei monatlicher Voranmeldung mit
  Dauerfristverlängerung ([§ 47 Abs. 1 UStDV][ustdv47]).
- **Höhe:** nach [§ 47 UStDV][ustdv47]; Regelfall, Teiljahr und
  Gründungsjahr in
  [monatsabschluss/dauerfristverlaengerung.md, Abschnitt 3](../monatsabschluss/dauerfristverlaengerung.md#3-sondervorauszahlung).
- **Frist:** Mit dem ersten Antrag auf Dauerfristverlängerung, also bis zum
  Abgabetag der ersten Voranmeldung, für die sie gelten soll
  ([§ 48 Abs. 1 UStDV][ustdv48]). In den Folgejahren bis zum gesetzlichen
  Abgabetag der ersten Voranmeldung, dem 10. Februar
  ([§ 48 Abs. 2 UStDV][ustdv48], [§ 18 Abs. 1 Satz 1 UStG][ustg18]). Berechnen,
  anmelden und zahlen am selben Tag. § 108 Abs. 3 AO gilt. Mit Berater gleich.
- **Zuständig:** du bereitest vor; der Mensch übermittelt. Kanal: ELSTER.
- **Weg:**
  1. Betrag und die Zeiträume, aus denen er stammt, dem Menschen vorlegen.
  2. Mein ELSTER → „Formulare und Leistungen“ → „Alle Formulare“ →
     „Umsatzsteuer“ → „Dauerfristverlängerung/Sondervorauszahlung (monatlich)“.
  3. Der Mensch sendet und zahlt.
  4. Er trägt Dauerfristverlängerung und Betrag in der App ein (Stammdaten).
- **Buchfink liefert:** die Voranmeldungen des Vorjahres über `erklaerung`,
  aus denen du den Betrag rechnest
  ([monatsabschluss/dauerfristverlaengerung.md](../monatsabschluss/dauerfristverlaengerung.md#3-sondervorauszahlung));
  Konto 3830 über `konten`. Einen Vorschlag des Betrags gibt kein Werkzeug.
- **Nachweis:** Transferticket der Anmeldung, vom Menschen in der App
  bestätigt, und die Zahlung auf 3830. Die Sondervorauszahlung ist zu
  berechnen, anzumelden und zu entrichten ([§ 48 Abs. 2 UStDV][ustdv48]); die
  Zahlung allein belegt die Anmeldung nicht. Buchfink führt die Pflicht heute
  schon mit der Buchung auf 3830 als erledigt. Frag dann nach dem
  Transferticket und bitte den Menschen, es zu bestätigen.
- **Bei Versäumnis:** Verspätungszuschlag nach Dauer, Häufigkeit und Höhe, nie
  zwingend ([§ 152 Abs. 3 Nr. 4, Abs. 8 Nr. 5 AO][ao152]); das Finanzamt kann
  die Sondervorauszahlung festsetzen ([§ 48 Abs. 3 UStDV][ustdv48]);
  Säumniszuschlag ([§ 240 AO][ao240]).
- **Vertiefung:** [monatsabschluss/dauerfristverlaengerung.md](../monatsabschluss/dauerfristverlaengerung.md)

<a id="zm"></a>
## Zusammenfassende Meldung

- **Grundlage:** [§ 18a Abs. 2, 3, 10, 11 UStG][ustg18a]. Pflicht für jedes
  Quartal, in dem die GmbH sonstige Leistungen nach § 3a Abs. 2 UStG an
  Unternehmer im übrigen Gemeinschaftsgebiet ausgeführt hat (Steuerschuld des
  Empfängers). Buchfink bildet die Pflicht nur für solche Quartale.
- **Frist:** 25. Tag nach Ablauf des Quartals; bei angezeigter Monatsmeldung
  25. Tag nach Ablauf des Monats ([§ 18a Abs. 2, 3 UStG][ustg18a]). Die
  Dauerfristverlängerung gilt nicht ([BZSt, Fristen][bzstzmfrist]). § 108
  Abs. 3 AO gilt. Mit Berater gleich.
- **Zuständig:** du bereitest vor; der Mensch übermittelt. Kanal: BZSt.
- **Weg:**
  1. `erklaerung` (`art: zm`) prüfen; Summe gleich Kennzahl 21 der
     Voranmeldungen des Quartals.
  2. Du legst die Zeilen als Tabelle vor; eine CSV-Datei der ZM bildet
     Buchfink noch nicht ab.
  3. Mein ELSTER → „Formulare und Leistungen“ → „Alle Formulare“ →
     „Umsatzsteuer“ → „Zusammenfassende Meldung“, Zeilen eintragen. Alternativ
     das BZSt-Online-Portal (BOP) mit Anmeldung zum Fachverfahren ZM
     ([BZSt, Elektronische Abgabe][bzstzm]).
  4. Der Mensch vergleicht die Zeilen mit deiner Tabelle und sendet.
- **Buchfink liefert:** `erklaerung` je Empfänger mit USt-IdNr. und Summe.
- **Nachweis:** Transferticket, vom Menschen in der App an der ZM erfasst; die
  Pflicht steht dann aus den Daten auf erledigt.
- **Bei Versäumnis:** kein Verspätungszuschlag ([§ 18a Abs. 11 UStG][ustg18a]).
  Geldbuße bis 5.000 € bei vorsätzlich oder leichtfertig fehlender, falscher,
  verspäteter oder nicht binnen eines Monats berichtigter Meldung
  ([§ 26a Abs. 2 Nr. 5, Abs. 3 UStG][ustg26a], [§ 18a Abs. 10 UStG][ustg18a]).
- **Vertiefung:** [monatsabschluss/zm.md](../monatsabschluss/zm.md)

<a id="kst-vorauszahlung"></a>
## Körperschaftsteuer-Vorauszahlung

- **Grundlage:** [§ 37 Abs. 1, 3 EStG][estg37] über
  [§ 31 Abs. 1, 2 KStG][kstg31]; Solidaritätszuschlag
  [§ 3 Abs. 1 Nr. 2 SolZG][solzg3].
- **Frist:** 10. März, 10. Juni, 10. September, 10. Dezember; Höhe laut
  Vorauszahlungsbescheid. Bei abweichendem Wirtschaftsjahr während des
  Wirtschaftsjahres, das im Veranlagungszeitraum endet
  ([§ 31 Abs. 2 KStG][kstg31]). § 108 Abs. 3 AO gilt; `fristen` zeigt nur das
  gesetzliche Datum ([Fristende](#fristende-zahlung-verlängerung)). Mit Berater
  gleich.
- **Wann im Kalender:** sobald ein Vorauszahlungsbescheid als Anlass erfasst ist
  oder Vorauszahlungen im Vorjahr oder im Jahr gebucht sind.
- **Zuständig:** Mensch (Zahlung). Kanal: intern.
- **Weg:** Überweisung an die Finanzkasse mit Steuernummer und Angaben aus dem
  Bescheid, oder SEPA-Lastschriftmandat beim Finanzamt. Anpassung beantragen:
  Mein ELSTER, „Antrag auf Anpassung von Vorauszahlungen“
  ([steuererklaerungen/bescheide.md](../steuererklaerungen/bescheide.md)).
- **Buchfink liefert:** Termine aus `fristen`; Konten 7600 und 7608 über
  `konten`; Zuordnung der Zahlung nach
  [bank/steuerzahlungen.md](../bank/steuerzahlungen.md#2-konten-je-steuerart).
- **Nachweis:** Buchung der Zahlung auf 7600 oder 7608 um den Termin (aus den
  Daten) oder Bestätigung durch den Menschen. Als Zahlung zählt nur eine, die
  vom Bankkonto kommt oder die der Mensch gebucht hat; deine Handbuchung belegt
  keine.
- **Bei Versäumnis:** Säumniszuschlag ([§ 240 AO][ao240]).
- **Vertiefung:** [steuererklaerungen/bescheide.md](../steuererklaerungen/bescheide.md),
  [bank/steuerzahlungen.md](../bank/steuerzahlungen.md)

<a id="gewst-vorauszahlung"></a>
## Gewerbesteuer-Vorauszahlung

- **Grundlage:** [§ 19 GewStG][gewstg19].
- **Frist:** 15. Februar, 15. Mai, 15. August, 15. November; je ein Viertel der
  Steuer der letzten Veranlagung oder laut Anpassungsbescheid der Gemeinde. Eine
  nachträgliche Erhöhung ist binnen eines Monats nach Bekanntgabe zu zahlen
  ([§ 19 Abs. 1–3 GewStG][gewstg19]). Bei abweichendem Wirtschaftsjahr während
  des Wirtschaftsjahres, das im Erhebungszeitraum endet
  ([§ 19 Abs. 1 Satz 2, 3 GewStG][gewstg19]). Festgesetzt wird eine
  Vorauszahlung erst ab 50 € ([§ 19 Abs. 5 Satz 2 GewStG][gewstg19]). § 108
  Abs. 3 AO gilt über [§ 1 Abs. 2 Nr. 3 AO][ao1]; `fristen` zeigt nur das
  gesetzliche Datum. Mit Berater gleich.
- **Wann im Kalender:** wie bei der Körperschaftsteuer-Vorauszahlung.
- **Zuständig:** Mensch (Zahlung an die Gemeinde). Kanal: intern.
- **Weg:** Überweisung an die Gemeindekasse mit dem Kassenzeichen aus dem
  Bescheid, oder SEPA-Lastschriftmandat bei der Gemeinde.
- **Buchfink liefert:** Termine aus `fristen`; Konto 7610 über `konten`.
- **Nachweis:** Buchung der Zahlung auf 7610 um den Termin (aus den Daten) oder
  Bestätigung durch den Menschen. Deine Handbuchung belegt keine Zahlung.
- **Bei Versäumnis:** Säumniszuschlag ([§ 240 AO][ao240] über
  [§ 1 Abs. 2 Nr. 5 AO][ao1]).
- **Vertiefung:** [steuererklaerungen/gewst.md](../steuererklaerungen/gewst.md),
  [bank/steuerzahlungen.md](../bank/steuerzahlungen.md)

<a id="lohnsteuer-anmeldung"></a>
## Lohnsteuer-Anmeldung

- **Grundlage:** [§ 41a Abs. 1, 2 EStG][estg41a].
- **Frist:** Anmeldung und Abführung spätestens am 10. Tag nach Ablauf des
  Anmeldungszeitraums. Zeitraum: Monat; Quartal bei Lohnsteuer des Vorjahres
  über 1.080 € bis 5.000 €; Kalenderjahr bis 1.080 €; Teiljahr hochrechnen
  ([§ 41a Abs. 2 EStG][estg41a]). § 108 Abs. 3 AO gilt. Mit Berater gleich.
- **Wann im Kalender:** sobald ein Lohnjournal gebucht ist. Maßgebend für den
  Zeitraum ist nur die abzuführende Lohnsteuer des Vorjahres
  ([§ 41a Abs. 2 Satz 2 EStG][estg41a]). Buchfink schätzt sie aus Konto 3730,
  das auch Kirchensteuer und Solidaritätszuschlag sammelt. Die Schätzung kann
  zu hoch sein und einen zu kurzen Zeitraum ergeben. Prüfe die Summe der
  Lohnsteuer aus den Anmeldungen des Vorjahres (Protokolle des Lohnbüros);
  weicht der Zeitraum ab, nenne dem Menschen den richtigen und die
  Fälligkeiten danach.
- **Entfällt:** Beschäftigt die GmbH keine Arbeitnehmer mehr und teilt sie das
  dem Finanzamt mit, entfallen weitere Anmeldungen
  ([§ 41a Abs. 1 Satz 4 EStG][estg41a]). Der Mensch kennzeichnet die Pflicht
  dann mit diesem Grund.
- **Zuständig:** Mensch; meist übermittelt das Lohnbüro aus der Lohnsoftware.
  Kanal: Lohnanbieter.
- **Weg:** Lohnbüro übermittelt; ohne Lohnbüro der Mensch in Mein ELSTER →
  „Formulare und Leistungen“ → „Alle Formulare“ → „Lohnsteuer Arbeitgeber“ →
  „Lohnsteuer-Anmeldung“. Zahlung bis zur Fälligkeit.
- **Buchfink liefert:** Zeitraum und Fälligkeit aus `fristen`; Lohnkonten über
  `auswertung` (`art: kontoblatt`); Abstimmung nach
  [lohn/abstimmung.md](../lohn/abstimmung.md).
- **Nachweis:** Übermittlungsprotokoll mit Transferticket, vom Menschen in der
  App bestätigt, und die gebuchte Zahlung zum Fälligkeitstag. Verlangt sind
  Anmeldung und Abführung ([§ 41a Abs. 1 Satz 1 Nr. 1, 2 EStG][estg41a]); ein
  gebuchtes Lohnjournal meldet den Zeitraum nur als vorbereitet. Prüfe je
  Zeitraum Protokoll und Zahlung, bitte das Lohnbüro um das Protokoll und lege
  es aus dem Eingang mit `beleg_ablegen` ab.
- **Bei Versäumnis:** Verspätungszuschlag nach Dauer, Häufigkeit und Höhe
  ([§ 152 Abs. 8 Nr. 1, 2 AO][ao152]); Säumniszuschlag ([§ 240 AO][ao240]);
  Geldbuße bis 25.000 €, wenn Steuerabzugsbeträge vorsätzlich oder leichtfertig
  nicht rechtzeitig abgeführt werden ([§ 380 AO][ao380]); Haftung des
  Geschäftsführers bei grober Fahrlässigkeit ([§ 69 AO][ao69]).
- **Vertiefung:** [lohn/ablauf.md](../lohn/ablauf.md),
  Abschnitt 4; [lohn/abstimmung.md](../lohn/abstimmung.md)

<a id="monatsfestschreibung"></a>
## Monatsfestschreibung

- **Grundlage:** Unveränderbarkeit ([§ 146 Abs. 4 AO][ao146],
  [§ 239 Abs. 3 HGB][hgb239]); Vorerfassungen bis zum Jahresabschluss offen zu
  halten ist unzulässig, Festschreibung ist ein Mittel der Unveränderbarkeit
  ([GoBD Rz. 107–110][gobd]); Buchung bis zum Ablauf des Folgemonats
  ([GoBD Rz. 52, 87][gobd]).
- **Frist:** Gesetzlich keine. Buchfink setzt das Ende des Folgemonats, mit
  Dauerfristverlängerung einen Monat später, zuzüglich der Nachfrist aus den
  Einstellungen. Ab dem 10. des Folgemonats steht der Monat als offene Aufgabe in
  der Liste. Keine Verschiebung nach § 108 AO. Den letzten Monat eines
  Voranmeldungszeitraums muss der Mensch spätestens am Fälligkeitstag der
  Voranmeldung festschreiben: Sie beruht auf festgeschriebenen Zahlen, und
  Buchfink nimmt ihr Transferticket erst nach der Festschreibung an. `fristen`
  zeigt für diesen Monat noch das spätere Datum; plane nach [ustva](#ustva).
- **Zuständig:** Mensch (nur in der App). Kanal: intern.
- **Weg:**
  1. Du klärst die Befunde aus `auswertung` (`art: pruefbericht`).
  2. Der Mensch schreibt den Monat in der App fest; blockierende Befunde
     übergeht er nur mit Begründung.
- **Buchfink liefert:** Prüfbericht mit Befunden und Schwere.
- **Nachweis:** die Festschreibung selbst (aus den Daten).
- **Bei Versäumnis:** Buchfink nimmt das Transferticket der Voranmeldung erst
  nach der Festschreibung an; die Voranmeldung bleibt bis dahin unbestätigt und
  steht nach ihrer Fälligkeit auf überfällig.
- **Vertiefung:** [monatsabschluss/festschreibung.md](../monatsabschluss/festschreibung.md)

<a id="bilanzstichtag"></a>
## Bilanzstichtag: Inventar und Abschlussbuchungen

- **Grundlage:** Inventar zum Schluss jedes Geschäftsjahres, innerhalb der einem
  ordnungsmäßigen Geschäftsgang entsprechenden Zeit
  ([§ 240 Abs. 2 HGB][hgb240]); Bilanz und GuV zum Schluss jedes
  Geschäftsjahres ([§ 242 Abs. 1, 2 HGB][hgb242]).
- **Frist:** Buchfink setzt sie gleich der Aufstellungsfrist
  ([aufstellung](#aufstellung)). Keine Verschiebung nach § 108 AO.
- **Zuständig:** du bereitest vor (Abschlussbuchungen nach Ja); der Mensch
  liefert Nachweise wie Saldenbestätigungen. Kanal: intern.
- **Weg:** Inventar aus Saldenliste und Nachweisen, Abschlussbuchungen
  (Abschreibung, Abgrenzung, Rückstellungen, Umsatzsteuer-Verrechnung,
  Steuerrückstellung) nach
  [jahresabschluss/ablauf.md](../jahresabschluss/ablauf.md); danach schreibt der
  Mensch das Geschäftsjahr fest.
- **Buchfink liefert:** `auswertung` (`art: susa`, `bilanz`, `guv`),
  `offene_posten`, `anlagen`, `abschluss_status`; die Abschlussbuchungen über
  `abschlussbuchung`.
- **Nachweis:** Festschreibung des Geschäftsjahres oder Stand „aufgestellt“
  (aus den Daten).
- **Bei Versäumnis:** Wer Bilanz oder Inventar nicht in der vorgeschriebenen
  Zeit aufstellt, macht sich strafbar, wenn die Gesellschaft ihre Zahlungen
  einstellt oder ein Insolvenzverfahren eröffnet oder mangels Masse abgelehnt
  wird ([§ 283b Abs. 1 Nr. 3 Buchst. b, Abs. 3 StGB][stgb283b],
  [§ 283 Abs. 6 StGB][stgb283]).
- **Vertiefung:** [jahresabschluss/abschlussbuchungen.md](../jahresabschluss/abschlussbuchungen.md)

<a id="aufstellung"></a>
## Aufstellung des Jahresabschlusses

- **Grundlage:** [§ 264 Abs. 1 Satz 1, 3, 4 HGB][hgb264]; Unterzeichnung
  [§ 245 HGB][hgb245].
- **Frist:** drei Monate nach Ende des Geschäftsjahres. Kleine und
  Kleinstgesellschaften: später, wenn das einem ordnungsgemäßen Geschäftsgang
  entspricht, spätestens nach sechs Monaten ([§ 264 Abs. 1 Satz 4 HGB][hgb264],
  [§ 267a Abs. 2 HGB][hgb267a]). Buchfink wählt die Frist nach der Größenklasse
  des Jahres. Keine Verschiebung nach § 108 AO. Mit Berater gleich.
- **Zuständig:** Mensch (Geschäftsführung). Kanal: intern.
- **Weg:**
  1. Du übergibst Bilanz, GuV, Anhangentwurf und die Liste der
     Abschlussposten.
  2. Die Geschäftsführung unterzeichnet mit Datum.
  3. Der Mensch speichert den unterschriebenen Abschluss in den Eingang (du legst
     ihn auf seine Anweisung mit `beleg_ablegen` ab) und setzt in der App
     „aufgestellt“.
- **Buchfink liefert:** `auswertung` (`art: bilanz`, `guv`), `anlagen`;
  Größenklasse und Frist in `fristen`.
- **Nachweis:** Stand „aufgestellt“ mit Datum (aus den Daten).
- **Bei Versäumnis:** Strafbarkeit wie beim [bilanzstichtag](#bilanzstichtag).
  Das Ordnungsgeldverfahren zur Offenlegung wartet nicht auf die Aufstellung
  ([§ 335 Abs. 1 Satz 3 HGB][hgb335]).
- **Vertiefung:** [jahresabschluss/aufstellen-feststellen-offenlegen.md](../jahresabschluss/aufstellen-feststellen-offenlegen.md)

<a id="feststellung"></a>
## Feststellung des Jahresabschlusses

- **Grundlage:** [§ 42a Abs. 1, 2 GmbHG][gmbhg42a], [§ 46 Nr. 1 GmbHG][gmbhg46],
  Beschlussform [§ 48 GmbHG][gmbhg48].
- **Frist:** Beschluss der Gesellschafter bis zum Ablauf von acht Monaten nach
  Ende des Geschäftsjahres, bei kleinen Gesellschaften von elf Monaten; der
  Gesellschaftsvertrag kann die Frist nicht verlängern
  ([§ 42a Abs. 2 GmbHG][gmbhg42a]). Keine Verschiebung nach § 108 AO. Mit
  Berater gleich.
- **Zuständig:** Mensch (Gesellschafterversammlung). Kanal: intern.
- **Weg:**
  1. Die Geschäftsführung legt den aufgestellten Abschluss unverzüglich den
     Gesellschaftern vor.
  2. Du entwirfst den Beschluss (Datum, Bilanzsumme, Ergebnis,
     Ergebnisverwendung).
  3. Die Gesellschafter beschließen; beim Alleingesellschafter eine
     unterschriebene Niederschrift.
  4. Der Mensch schreibt das Jahr fest und setzt „festgestellt“ mit Datum und
     Bezug auf den Beschluss. Du legst den Beschluss mit `beleg_ablegen` ab.
- **Buchfink liefert:** Zahlen für den Beschluss aus `auswertung`.
- **Nachweis:** Beschluss, Stand „festgestellt“ (aus den Daten).
- **Bei Versäumnis:** Ohne Feststellung fehlt der Gegenstand der Offenlegung
  ([§ 325 Abs. 1 Satz 1 Nr. 1 HGB][hgb325]); das Ordnungsgeld droht trotzdem
  ([§ 335 Abs. 1 Satz 3 HGB][hgb335]).
- **Vertiefung:** [jahresabschluss/aufstellen-feststellen-offenlegen.md](../jahresabschluss/aufstellen-feststellen-offenlegen.md)

<a id="ergebnisverwendung"></a>
## Beschluss über die Ergebnisverwendung

- **Grundlage:** [§ 42a Abs. 2 GmbHG][gmbhg42a], [§ 46 Nr. 1 GmbHG][gmbhg46],
  [§ 29 GmbHG][gmbhg29]; UG: gesetzliche Rücklage nach
  [§ 5a Abs. 3 GmbHG][gmbhg5a]
  ([jahresabschluss/ergebnisverwendung-ug.md](../jahresabschluss/ergebnisverwendung-ug.md#1-gesetzliche-rücklage-der-ug)).
- **Frist:** wie die Feststellung. Keine Verschiebung nach § 108 AO.
- **Zuständig:** Mensch (die Gesellschafter beschließen); du buchst den
  Beschluss. Kanal: intern.
- **Weg:** Beschluss mit der Feststellung. Lege ihn mit `beleg_ablegen` ab und
  buche ihn nach Rückfrage mit `ergebnisverwendung_buchen`. Bei einer
  Ausschüttung bestimmt der Beschluss den Auszahlungstag; daraus folgt die
  Pflicht [kapitalertragsteuer](#kapitalertragsteuer).
- **Buchfink liefert:** Jahresergebnis, Gewinn- oder Verlustvortrag, bei der UG
  die Rücklage.
- **Nachweis:** gebuchter Beschluss (aus den Daten).
- **Bei Versäumnis:** Über die Verwendung bestimmen allein die Gesellschafter
  ([§ 46 Nr. 1 GmbHG][gmbhg46]); ohne Beschluss keine Ausschüttung.
- **Vertiefung:** [jahresabschluss/ergebnisverwendung-ug.md](../jahresabschluss/ergebnisverwendung-ug.md),
  [gesellschafter/ausschuettung.md](../gesellschafter/ausschuettung.md)

<a id="offenlegung"></a>
## Offenlegung im Unternehmensregister

- **Grundlage:** [§ 325 Abs. 1, 1a, 1b HGB][hgb325], [§ 326 HGB][hgb326].
- **Frist:** spätestens ein Jahr nach dem Abschlussstichtag
  ([§ 325 Abs. 1a Satz 1 HGB][hgb325]). Eine spätere Änderung des Abschlusses
  ist ebenfalls offenzulegen ([§ 325 Abs. 1b HGB][hgb325]). Keine Verschiebung
  nach § 108 AO. Mit Berater gleich.
- **Umfang:** festgestellter Abschluss in deutscher Sprache; was je
  Größenklasse offenzulegen ist und wer hinterlegen darf, steht in
  [jahresabschluss/aufstellen-feststellen-offenlegen.md](../jahresabschluss/aufstellen-feststellen-offenlegen.md#4-offenlegen).
- **Zuständig:** Mensch (Geschäftsführung). Kanal: Unternehmensregister.
- **Weg** ([Publikations-Plattform][pp], [So geht's][ppveroeff]):
  1. Einmal registrieren auf publikations-plattform.de.
  2. Einmal elektronisch identifizieren, vor der ersten Übermittlung, je Person,
     die übermittelt: automatisches oder begleitetes Video-Verfahren,
     Online-Ausweis oder Steuerberaterplattform; Gebühr nach JVKostG
     ([Registrieren/Identifizieren][ppident]). Ohne Identifizierung keine
     Offenlegung.
  3. Prozesseinstieg „Unternehmensregister“ → „Veröffentlichen“ oder, nur
     Kleinst, „Hinterlegen“.
  4. Unterlagen übermitteln: Datei hochladen (XML; PDF oder Office kostet ein
     zusätzliches Konvertierungsentgelt) oder, nur für kleine Gesellschaften,
     Formular ausfüllen: Bilanzwerte in vorbelegte Felder, Anhang in ein
     Textfeld.
  5. Im Bereich „Meine Daten“ den Stand des Auftrags verfolgen; nach
     Veröffentlichung oder Hinterlegung steht dort der Beleg
     ([Meine Daten][ppdaten]).
  Geschäftsjahre, die vor dem 01.01.2022 begannen, gehen an den Bundesanzeiger.
- **Buchfink liefert:** Bilanz aus `auswertung` (`art: bilanz`), Größenklasse
  und Umfang in `fristen`. Eine Datei für die Plattform erzeugt Buchfink nicht.
- **Nachweis:** Beleg der Veröffentlichung oder Hinterlegung aus „Meine Daten“
  mit Datum. Der Mensch setzt damit in der App „offengelegt“; die Pflicht steht
  dann aus den Daten auf erledigt. Du legst den Beleg mit `beleg_ablegen` ab.
- **Bei Versäumnis:** Ordnungsgeldverfahren des Bundesamts für Justiz gegen
  Geschäftsführer oder Gesellschaft; Androhung mit Frist von sechs Wochen, die
  Pflicht zu erfüllen oder Einspruch einzulegen, und Kosten des Verfahrens; Ordnungsgeld 2.500 € bis 25.000 €; bei Erfüllung nach
  den sechs Wochen herabgesetzt auf 500 € (hinterlegende Kleinstgesellschaft),
  1.000 € (kleine Gesellschaft), sonst 2.500 €, bei nur geringfügiger
  Überschreitung weniger ([§ 335 Abs. 1, 3, 4 HGB][hgb335]). Mit der
  Festsetzung wiederholt das Bundesamt die Androhung mit neuem Ordnungsgeld;
  das Verfahren läuft weiter, bis offengelegt ist
  ([§ 335 Abs. 4 Satz 1 HGB][hgb335]). War die Frist unverschuldet versäumt:
  Wiedereinsetzung binnen zwei Wochen nach Wegfall des Hindernisses schriftlich
  beim Bundesamt beantragen und die Offenlegung binnen sechs Wochen nachholen
  ([§ 335 Abs. 5 HGB][hgb335]). Liegt eine Androhung vor, lege sie ab und nenne
  dem Menschen sofort die Sechswochenfrist.
- **Vertiefung:** [jahresabschluss/aufstellen-feststellen-offenlegen.md](../jahresabschluss/aufstellen-feststellen-offenlegen.md),
  [jahresabschluss/groessenklassen-anhang.md](../jahresabschluss/groessenklassen-anhang.md)

<a id="ebilanz"></a>
## E-Bilanz

- **Grundlage:** [§ 5b Abs. 1, 2 EStG][estg5b]; Härtefall
  [§ 150 Abs. 8 AO][ao150]; Papierweg [§ 60 EStDV][estdv60].
- **Frist:** mit der Körperschaftsteuererklärung des Jahres, in dem das
  Wirtschaftsjahr endet ([kst-erklaerung](#kst-erklaerung)), mit Berater also
  die Berater-Frist der Körperschaftsteuer. § 108 Abs. 3 AO gilt.
- **Zuständig:** du bereitest vor; der Mensch übermittelt oder der Berater.
  Kanal: Online-Dienst mit ELSTER-Anbindung; Mein ELSTER hat kein Formular für
  die E-Bilanz
  ([steuererklaerungen/ebilanz.md, Abschnitt 5](../steuererklaerungen/ebilanz.md#5-übermittlungsweg)).
- **Weg:**
  1. Abschluss aufgestellt; Bilanz und GuV mit `auswertung` geprüft.
  2. `ebilanz_exportieren` (`jahr`, `format: xbrl` als Vorgabe) schreibt die
     geprüfte XBRL-Datei in den Exportordner.
  3. Der Mensch importiert sie im Online-Dienst, gleicht mit dir ab, validiert
     und sendet; danach trägt er Protokoll und Transferticket in Buchfink ein.
  Prüfpunkte vor dem Senden (Kontennachweise, Anlagenspiegel), Ablauf im
  Dienst und Rückfallweg (`format: susa_csv`):
  [steuererklaerungen/ebilanz.md, Abschnitte 4 und 5](../steuererklaerungen/ebilanz.md#4-prüfpunkte).
- **Weitere Wege:** Steuerberater; Antrag auf Verzicht wegen unbilliger Härte
  ([§ 5b Abs. 2 EStG][estg5b] i. V. m. [§ 150 Abs. 8 AO][ao150]); Voraussetzungen
  und Abschriften in
  [steuererklaerungen/ebilanz.md, Abschnitt 5](../steuererklaerungen/ebilanz.md#5-übermittlungsweg).
- **Buchfink liefert:** XBRL-Datei nach der amtlichen Taxonomie mit
  Kontennachweis, vor dem Export geprüft (`befunde`); als Rückfallweg die
  Summen- und Saldenliste als CSV.
- **Nachweis:** Transferticket; der Mensch bestätigt in der App.
- **Bei Versäumnis:** Ohne E-Bilanz ist die Körperschaftsteuererklärung
  unvollständig. Das Finanzamt kann die Übermittlung mit Zwangsgeld
  durchsetzen, je Festsetzung bis 25.000 € ([§§ 328, 329 AO][ao328]).
- **Vertiefung:** [steuererklaerungen/ebilanz.md](../steuererklaerungen/ebilanz.md)

<a id="kst-erklaerung"></a>
## Körperschaftsteuererklärung

- **Grundlage:** [§ 31 Abs. 1a KStG][kstg31], [§ 149 Abs. 2–4 AO][ao149]. Mit
  ihr geht die Erklärung zur gesonderten Feststellung des steuerlichen
  Einlagekontos auf den Schluss des Wirtschaftsjahres
  ([§ 27 Abs. 2 Satz 4 KStG][kstg27]). Bei abweichendem Wirtschaftsjahr endet
  ihre Frist nach dem Wortlaut sieben Monate nach diesem Schluss
  ([§ 149 Abs. 2 Satz 1 AO][ao149]), also vor der Frist der
  Körperschaftsteuererklärung. Nicht geklärt; plane vorsichtshalber mit dem
  früheren Datum oder lass den Menschen beim Finanzamt nachfragen.
- **Frist:** ohne Berater sieben Monate nach Ablauf des Kalenderjahres
  ([§ 149 Abs. 2 AO][ao149]). Mit beauftragtem Berater letzter Februartag des
  zweiten Folgejahres ([§ 149 Abs. 3 Nr. 2 AO][ao149]); fordert das Finanzamt
  vorab an, vier Monate nach Bekanntgabe, nie kürzer als ohne Berater
  ([§ 149 Abs. 4 AO][ao149]). Bei abweichendem Wirtschaftsjahr zählt das
  Kalenderjahr, in dem es endet ([§ 7 Abs. 4 Satz 2 KStG][kstg7]). Für 2020 bis
  2024 gelten Übergangsfristen ([Art. 97 § 36 Abs. 3 EGAO][egao36]); Buchfink
  rechnet sie ein. § 108 Abs. 3 AO gilt.
- **Zuständig:** du bereitest vor, oder Steuerberater. Kanal: ELSTER.
- **Weg:** Mein ELSTER → „Formulare und Leistungen“ → „Alle Formulare“ →
  „Körperschaftsteuer“ → „Körperschaftsteuererklärung (KSt 1)“ mit Anlagen.
  Der Mensch überträgt die Werte nach deiner Tabelle, lässt prüfen und sendet.
- **Buchfink liefert:** `erklaerung` (`art: kst`) formularnah mit Zeilen und
  Herkunft; offene Angaben über `erklaerung_eingabe`; `erklaerung_exportieren`
  (`art: kst`, `jahr`) als JSON zur Ablage.
- **Nachweis:** Transferticket; der Mensch bestätigt in der App.
- **Bei Versäumnis:** Verspätungszuschlag; wann er zwingend ist, Ausnahmen und
  Höhe:
  [steuererklaerungen/fristen-zuschlaege.md, Abschnitt 4](../steuererklaerungen/fristen-zuschlaege.md#4-verspätungszuschlag),
  auch für die Feststellungserklärungen (Einlagekonto, Verlustvortrag). Das
  Finanzamt kann
  schätzen; die Pflicht zur Abgabe bleibt ([§ 162 AO][ao162],
  [§ 149 Abs. 1 Satz 4 AO][ao149]). Es kann die Abgabe mit Zwangsgeld
  durchsetzen ([§§ 328, 329 AO][ao328]).
- **Vertiefung:** [steuererklaerungen/kst.md](../steuererklaerungen/kst.md),
  [steuererklaerungen/fristen-zuschlaege.md](../steuererklaerungen/fristen-zuschlaege.md),
  [steuererklaerungen/abgabe.md](../steuererklaerungen/abgabe.md)

<a id="gewst-erklaerung"></a>
## Gewerbesteuererklärung

- **Grundlage:** [§ 14a GewStG][gewstg14a], [§ 149 Abs. 2, 3 Nr. 3 AO][ao149].
- **Frist:** wie die Körperschaftsteuererklärung. § 108 Abs. 3 AO gilt.
- **Zuständig:** du bereitest vor, oder Steuerberater. Kanal: ELSTER (an das
  Finanzamt; die Gemeinde setzt die Steuer fest).
- **Weg:** Mein ELSTER → „Formulare und Leistungen“ → „Alle Formulare“ →
  „Gewerbesteuer“ → „Gewerbesteuererklärung (GewSt 1 A)“. Bei Betriebsstätten
  in mehreren Gemeinden zusätzlich „Erklärung für die Zerlegung des
  Gewerbesteuermessbetrages (GewSt 1 D)“ ([§ 14a Satz 1 GewStG][gewstg14a]).
- **Buchfink liefert:** `erklaerung` (`art: gewst`) formularnah mit Zeilen;
  `erklaerung_exportieren` (`art: gewst`) als JSON zur Ablage.
- **Nachweis:** Transferticket; der Mensch bestätigt in der App.
- **Bei Versäumnis:** Verspätungszuschlag nach
  [steuererklaerungen/fristen-zuschlaege.md, Abschnitt 4](../steuererklaerungen/fristen-zuschlaege.md#4-verspätungszuschlag).
  Schätzung und Zwangsgeld wie bei der Körperschaftsteuererklärung.
- **Vertiefung:** [steuererklaerungen/gewst.md](../steuererklaerungen/gewst.md)

<a id="ust-erklaerung"></a>
## Umsatzsteuererklärung

- **Grundlage:** [§ 18 Abs. 3, 4 UStG][ustg18],
  [§ 149 Abs. 2, Abs. 3 Nr. 4, Abs. 5 AO][ao149].
- **Frist:** wie die Körperschaftsteuererklärung. Endet die Tätigkeit im Jahr:
  ein Monat nach Ablauf des kürzeren Besteuerungszeitraums, ohne Berater-Frist
  ([§ 18 Abs. 3 Satz 2 UStG][ustg18], [§ 149 Abs. 5 AO][ao149]). Eine
  Abschlusszahlung ist einen Monat nach Eingang der Erklärung fällig
  ([§ 18 Abs. 4 Satz 1 UStG][ustg18]). § 108 Abs. 3 AO gilt.
- **Zuständig:** du bereitest vor, oder Steuerberater. Kanal: ELSTER.
- **Weg:** Mein ELSTER → „Formulare und Leistungen“ → „Alle Formulare“ →
  „Umsatzsteuer“ → „Umsatzsteuererklärung“.
- **Buchfink liefert:** `erklaerung` (`art: ust_jahr`) formularnah mit Zeilen
  und Summe der Vorauszahlungen; `erklaerung_exportieren` (`art: ust_jahr`) als
  JSON zur Ablage.
- **Nachweis:** Transferticket; der Mensch bestätigt in der App.
- **Bei Versäumnis:** Verspätungszuschlag nach
  [steuererklaerungen/fristen-zuschlaege.md, Abschnitt 4](../steuererklaerungen/fristen-zuschlaege.md#4-verspätungszuschlag);
  Schätzung und Zwangsgeld wie bei der Körperschaftsteuererklärung;
  Säumniszuschlag auf die Abschlusszahlung ([§ 240 AO][ao240]).
- **Vertiefung:** [steuererklaerungen/ust-jahr.md](../steuererklaerungen/ust-jahr.md)

<a id="transparenzregister"></a>
## Mitteilung an das Transparenzregister

- **Grundlage:** [§ 20 Abs. 1 GwG][gwg20]; wirtschaftlich Berechtigte
  [§ 3 Abs. 2 GwG][gwg3]; Angaben [§ 19 Abs. 1 GwG][gwg19]: Vor- und Nachname,
  Geburtsdatum, Wohnort, Art und Umfang des wirtschaftlichen Interesses, alle
  Staatsangehörigkeiten.
- **Anlass:** Eintragung der GmbH ins Handelsregister und jede Änderung der
  wirtschaftlich Berechtigten oder ihrer Angaben. Mitzuteilen hat die
  juristische Person ([§ 20 Abs. 1 Satz 1 GwG][gwg20]); als solche besteht die
  GmbH erst mit der Eintragung ([§ 11 Abs. 1 GmbHG][gmbhg11]).
- **Wirtschaftlich berechtigt** ist jede natürliche Person, die unmittelbar
  oder mittelbar mehr als 25 % der Kapitalanteile hält, mehr als 25 % der
  Stimmrechte kontrolliert oder auf vergleichbare Weise Kontrolle ausübt
  ([§ 3 Abs. 2 Satz 1 GwG][gwg3]). Mittelbar berechtigt ist insbesondere, wer
  eine Vereinigung kontrolliert, die solche Anteile hält; Kontrolle heißt
  beherrschender Einfluss nach § 290 Abs. 2–4 HGB
  ([§ 3 Abs. 2 Satz 2–4 GwG][gwg3]). Prüfe die Kontrolle je Ebene der
  Holdingkette; multipliziere keine Quoten. Lässt sich niemand ermitteln,
  gilt der gesetzliche Vertreter als wirtschaftlich Berechtigter
  ([§ 3 Abs. 2 Satz 5 GwG][gwg3]).
- **Frist:** unverzüglich; die Angaben sind aktuell zu halten
  ([§ 20 Abs. 1 Satz 1 GwG][gwg20]). Keine Verschiebung nach § 108 AO.
- **Zuständig:** Mensch (Geschäftsführung). Kanal: Transparenzregister.
- **Weg** ([Hilfe-Center des Registers][trhilfe]; die Seite braucht
  JavaScript, ohne Browser liest du denselben Inhalt als [JSON][trhilfeapi]):
  1. transparenzregister.de → „Nutzerkonto anlegen“ → „Neues Nutzerkonto
     erstellen“ mit E-Mail-Adresse und Passwort; Aktivierungslink binnen
     24 Stunden bestätigen.
  2. Bei den weiteren Daten die Option „Eintragungen vornehmen“ wählen.
  3. Eintragungsstrecke: GmbH suchen, wirtschaftlich Berechtigte angeben,
     Berechtigung zur Mitteilung versichern, übermitteln. Bei einer Änderung
     „Erste Mitteilung oder Folgemitteilung“ mit Gültigkeitsdatum wählen.
  4. Eingangsmitteilung (PDF mit 12-stelliger Vorgangsnummer) sofort
     herunterladen; sie steht nur 14 Tage im Konto.
  Eine GmbH i. G., die sich schon eingetragen hat, meldet den Wegfall von
  „in Gründung“ als Änderungsmitteilung.
- **Buchfink liefert:** Gesellschafter mit Anteilen aus den Stammdaten der
  App. Mittelbare Beteiligungen kennt Buchfink nicht; frag danach. Eine
  Änderung der wirtschaftlich Berechtigten erfasst du nach Rückfrage mit
  `anlass_erfassen` (`art: wirtschaftlich-berechtigte`).
- **Nachweis:** Vorgangsnummer der Eingangsmitteilung; der Mensch bestätigt in
  der App, du legst das PDF mit `beleg_ablegen` ab. Die Eingangsmitteilung
  belegt den Eingang; die Eintragung belegt erst ein Registerauszug.
- **Bei Versäumnis:** Geldbuße bis 150.000 € bei Vorsatz, bei Leichtfertigkeit
  bis 100.000 € ([§ 56 Abs. 1 Satz 1 Nr. 55, Satz 2 GwG][gwg56]); bei
  schwerwiegendem, wiederholtem oder systematischem Verstoß bis 1 Mio. € oder
  das Zweifache des Vorteils ([§ 56 Abs. 3 GwG][gwg56]). Zuständig:
  Bundesverwaltungsamt ([§ 56 Abs. 5 Satz 2 GwG][gwg56]). Unanfechtbare
  Bußgeldentscheidungen macht es mit den Namen der Verantwortlichen im Internet
  bekannt ([§ 57 Abs. 1 GwG][gwg57]).
- **Gebühr:** Für die Führung des Registers erhebt die registerführende Stelle
  eine Jahresgebühr ([§ 24 Abs. 1 GwG][gwg24]), ab dem Gebührenjahr 2024
  19,80 € ([TrGebV, Anlage Nr. 1][trgebv]).
- **Vertiefung:** Reihenfolge in der Gründung:
  [gruendung/gruendungsweg.md](../gruendung/gruendungsweg.md#4-behördenwege-mit-fristen)

<a id="gesellschafterliste"></a>
## Neue Gesellschafterliste

- **Grundlage:** [§ 40 Abs. 1–3 GmbHG][gmbhg40], [§ 16 Abs. 1 GmbHG][gmbhg16],
  Einreichung [§ 12 Abs. 2 HGB][hgb12].
- **Anlass:** jede Veränderung in den Personen der Gesellschafter oder im
  Umfang ihrer Beteiligung.
- **Frist:** unverzüglich nach Wirksamwerden der Veränderung
  ([§ 40 Abs. 1 Satz 1 GmbHG][gmbhg40]). Keine Verschiebung nach § 108 AO.
- **Zuständig:** Mensch. Hat ein Notar an der Veränderung mitgewirkt (etwa bei
  der Abtretung), reicht er die Liste anstelle der Geschäftsführer ein
  ([§ 40 Abs. 2 GmbHG][gmbhg40]). Kanal: Notar.
- **Weg:** Notar reicht ein und schickt der GmbH eine Abschrift. Ohne Notar
  unterschreiben die Geschäftsführer die Liste oder versehen sie mit
  qualifizierter elektronischer Signatur ([§ 40 Abs. 1 GmbHG][gmbhg40]); die
  Einreichung beim Handelsregister ist elektronisch
  ([§ 12 Abs. 2 HGB][hgb12]). Den technischen Weg zum Registergericht nennt
  keine hier geprüfte Quelle: Der Mensch klärt ihn mit dem Gericht, im Zweifel
  reicht ein Notar die Liste gegen Gebühr ein. Danach prüfen:
  [transparenzregister](#transparenzregister).
- **Buchfink liefert:** aus dem Anlass `gesellschafterwechsel`, den du nach
  Rückfrage mit `anlass_erfassen` erfasst, die Pflichten Gesellschafterliste
  und Transparenzregister. Die Gesellschafter in den Stammdaten ändert nur der
  Mensch.
- **Nachweis:** Datum der Einreichung und Abschrift oder Bestätigung des Notars;
  der Mensch bestätigt in der App, du legst die Abschrift ab.
- **Bei Versäumnis:** Gegenüber der GmbH gilt als Gesellschafter nur, wer in
  der aufgenommenen Liste steht ([§ 16 Abs. 1 GmbHG][gmbhg16]). Die
  Geschäftsführer haften denjenigen, deren Beteiligung sich geändert hat, und
  den Gläubigern der Gesellschaft für den Schaden als Gesamtschuldner
  ([§ 40 Abs. 3 GmbHG][gmbhg40]).
- **Vertiefung:** Nach der Eintragung:
  [gruendung/gruendungsweg.md](../gruendung/gruendungsweg.md#5-nach-der-eintragung)

<a id="fragebogen-steuerliche-erfassung"></a>
## Fragebogen zur steuerlichen Erfassung

- **Grundlage:** [§ 137 AO][ao137], [§ 138 Abs. 1b, 4 AO][ao138].
- **Frist:** ein Monat nach der Gründung ([§ 137 Abs. 2 AO][ao137],
  [§ 138 Abs. 4 AO][ao138]). Buchfink rechnet ab dem Gründungstag aus den
  Stammdaten (Beurkundung). § 108 Abs. 3 AO gilt. Mit Berater gleich.
- **Zuständig:** du bereitest vor; der Mensch übermittelt. Kanal: ELSTER.
- **Weg:** Mein ELSTER → „Formulare und Leistungen“ → „Alle Formulare“ →
  „Fragebogen zur steuerlichen Erfassung“ → „Fragebogen zur steuerlichen
  Erfassung: Gründung einer Kapitalgesellschaft beziehungsweise
  Genossenschaft“ ([Formularliste][elsterformulare]). Ohne Zertifikat
  registriert sich die GmbH nur mit E-Mail-Adresse; das reicht für Fragebögen
  ([gruendung/steuerliche-erfassung.md](../gruendung/steuerliche-erfassung.md)).
  Die Steuernummer aus dem Antwortschreiben trägt der
  Mensch in der App ein.
- **Buchfink liefert:** Firma, Sitz, Geschäftsjahr und Bankverbindung aus den
  Stammdaten; die Eröffnungsbilanz aus den Gründungsbuchungen
  ([gruendung/eroeffnungsbilanz.md](../gruendung/eroeffnungsbilanz.md)).
- **Nachweis:** Transferticket; der Mensch bestätigt in der App.
- **Bei Versäumnis:** Das Finanzamt kann die Angaben mit Zwangsgeld bis
  25.000 € je Festsetzung durchsetzen ([§§ 328, 329 AO][ao328]). Ohne
  Steuernummer oder USt-IdNr. fehlt jeder Ausgangsrechnung eine Pflichtangabe
  ([§ 14 Abs. 4 Satz 1 Nr. 2 UStG][ustg14]).
- **Vertiefung:** [gruendung/steuerliche-erfassung.md](../gruendung/steuerliche-erfassung.md)

<a id="gewerbeanmeldung"></a>
## Gewerbeanzeige

- **Grundlage:** [§ 14 Abs. 1 GewO][gewo14], Vordruck GewA 1 nach
  [§ 1 Nr. 1 GewAnzV][gewanzv1], elektronisch nach [§ 2 GewAnzV][gewanzv2].
- **Frist:** gleichzeitig mit dem Beginn des Betriebs
  ([§ 14 Abs. 1 Satz 1 GewO][gewo14]). Buchfink setzt den Gründungstag an.
  Keine Verschiebung nach § 108 AO.
- **Weitere Anlässe:** Anzuzeigen sind auch Verlegung, Wechsel oder
  Ausdehnung des Gegenstands, Namensänderung und Aufgabe
  ([§ 14 Abs. 1 Satz 2 GewO][gewo14]). Buchfink bildet nur die Anzeige zum
  Beginn; bei diesen Anlässen erinnere den Menschen selbst.
- **Zuständig:** Mensch (Geschäftsführung). Kanal: Gewerbeamt.
- **Weg:** Online-Dienst der Gemeinde oder des Landes, sonst beim Gewerbeamt.
  Die Gemeinde unterrichtet das Finanzamt ([§ 138 Abs. 1 Satz 1 AO][ao138]).
  Die Behörde bescheinigt den Empfang binnen drei Tagen
  ([§ 15 Abs. 1 GewO][gewo15]). Die Meldung an die Berufsgenossenschaft gilt
  als erfüllt, wenn die Gewerbeanzeige binnen einer Woche nach Beginn erstattet
  ist ([§ 192 Abs. 1 Satz 2 SGB VII][sgb7_192];
  [gruendung/gruendungsweg.md](../gruendung/gruendungsweg.md)).
- **Wertungsfrage:** rein freiberuflicher Unternehmensgegenstand. Der Mensch
  klärt mit dem Gewerbeamt, ob eine Anzeige nötig ist, und kennzeichnet die
  Pflicht sonst mit Grund als entfallen.
- **Buchfink liefert:** Firma, Sitz und Geschäftsführer aus den Stammdaten.
- **Nachweis:** Empfangsbescheinigung; der Mensch bestätigt in der App, du
  legst sie ab.
- **Bei Versäumnis:** Geldbuße bis 1.000 €, auch für die weiteren Anlässe
  ([§ 146 Abs. 2 Nr. 2 Buchst. b, Abs. 3 GewO][gewo146]).
- **Vertiefung:** [gruendung/gruendungsweg.md](../gruendung/gruendungsweg.md),
  Abschnitt 4

<a id="eroeffnungsbilanz"></a>
## Aufstellung der Eröffnungsbilanz

- **Grundlage:** [§ 242 Abs. 1 HGB][hgb242]; nach Satz 2 gelten die
  Vorschriften über die Bilanz des Jahresabschlusses entsprechend, darunter
  [§ 243 Abs. 3 HGB][hgb243] und [§ 245 HGB][hgb245]. Aufbewahrung
  [§ 147 Abs. 1 Nr. 1, Abs. 3 AO][ao147], [§ 257 HGB][hgb257].
- **Frist:** in der einem ordnungsmäßigen Geschäftsgang entsprechenden Zeit
  ([§ 243 Abs. 3 HGB][hgb243]); eine feste Frist nennt das Gesetz nicht.
  Buchfink setzt in Anlehnung an [§ 264 Abs. 1 Satz 3 HGB][hgb264] drei Monate
  nach der Beurkundung an und sagt das im Hinweis der Pflicht. Keine
  Verschiebung nach § 108 AO. Mit Berater gleich. `fristen` führt die Pflicht
  nur für eine Gesellschaft, die in Buchfink gegründet wurde
  (`eroeffnungsbilanz.gruendung`), nicht für einen Umsteiger.
- **Zuständig:** Mensch (Geschäftsführung); du prüfst die Zahlen. Kanal:
  intern.
- **Weg:**
  1. Du prüfst die Bilanz zum Beurkundungstag nach
     [gruendung/eroeffnungsbilanz.md](../gruendung/eroeffnungsbilanz.md) und
     meldest `pflicht_vorbereitet`.
  2. Der Mensch speichert sie in der App als PDF (Unternehmen, Stammdaten,
     Block „Gesellschafter & Kapital“, „Als PDF speichern“ an der Pflicht).
     Meldet Buchfink einen Befund statt der Datei, klärst du ihn mit ihm.
  3. Die Geschäftsführer unterzeichnen sie mit Datum
     ([§ 245 HGB][hgb245]).
  4. Der Mensch speichert die unterzeichnete Fassung in den Eingang, du legst
     sie auf seine Anweisung mit `beleg_ablegen` ab. Das Original bewahrt er
     auf ([§ 257 Abs. 3 HGB][hgb257]).
  5. Er bestätigt die Pflicht in der App, an derselben Stelle.
- **Buchfink liefert:** Salden der Gründungskonten zum Beurkundungstag über
  `auswertung`. Das PDF zum Unterzeichnen gibt nur die App aus. Stichtag ist
  das Gründungsdatum der Unternehmensdaten, wie bei
  [ebilanz-eroeffnung](#ebilanz-eroeffnung). Das PDF hat je Geschäftsführer
  aus den Unternehmensdaten eine Zeile für Ort, Datum und Unterschrift.
  Buchfink setzt es nicht, solange die Bilanz nicht aufgeht, das Kapital von
  der Gründung abweicht oder die Geschäftsführung in den Unternehmensdaten
  fehlt.
- **Nachweis:** Bestätigung in der App; die unterzeichnete Fassung liegt als
  Beleg im Archiv. Du meldest höchstens „vorbereitet“.
- **Bei Versäumnis:** Strafbar ist, wer die Bilanz seines Vermögens nicht in
  der vorgeschriebenen Zeit aufstellt, wenn die Gesellschaft ihre Zahlungen
  einstellt oder ein Insolvenzverfahren eröffnet oder mangels Masse abgelehnt
  wird ([§ 283b Abs. 1 Nr. 3 Buchst. b, Abs. 3 StGB][stgb283b],
  [§ 283 Abs. 6 StGB][stgb283]). Die Eröffnungsbilanz ist eine solche Bilanz
  ([§ 242 Abs. 1 Satz 1 HGB][hgb242]). Ohne sie fehlt auch die E-Bilanz der
  Eröffnungsbilanz ([ebilanz-eroeffnung](#ebilanz-eroeffnung)).
- **Vertiefung:** [gruendung/eroeffnungsbilanz.md](../gruendung/eroeffnungsbilanz.md)

<a id="ebilanz-eroeffnung"></a>
## E-Bilanz der Eröffnungsbilanz

- **Grundlage:** [§ 5b Abs. 1 Satz 4 EStG][estg5b]: bei Eröffnung des
  Betriebs gelten die Sätze 1 bis 3 entsprechend.
- **Frist:** Eine eigene Frist nennt das Gesetz nicht. Buchfink setzt die
  Frist der E-Bilanz des ersten Geschäftsjahres an ([ebilanz](#ebilanz)), also
  die der Körperschaftsteuererklärung des Kalenderjahres, in dem dieses
  Geschäftsjahr endet, mit Berater die Berater-Frist. Das ist die späteste
  vertretbare Annahme; früher zu übermitteln ist möglich. § 108 Abs. 3 AO
  gilt. `fristen` führt die Pflicht wie die Eröffnungsbilanz nur für eine
  Gründung in Buchfink (`e-bilanz-eroeffnung.gruendung`).
- **Zuständig:** du bereitest vor; der Mensch übermittelt oder der Berater.
  Kanal: Online-Dienst mit ELSTER-Anbindung wie bei der [ebilanz](#ebilanz);
  Mein ELSTER hat kein Formular.
- **Weg:**
  1. Die Gründungsbuchungen stehen; die Eröffnungsbilanz geht auf
     ([eroeffnungsbilanz](#eroeffnungsbilanz)).
  2. `ebilanz_exportieren` (`jahr`: Geschäftsjahr der Gründung,
     `bilanzart: eroeffnungsbilanz`): Buchfink prüft und schreibt die
     XBRL-Datei mit der Bilanzart Eröffnungsbilanz, ohne GuV, in den
     Exportordner.
  3. Der Mensch importiert sie im Online-Dienst, validiert und sendet wie bei
     der [ebilanz](#ebilanz).
  4. Er trägt das Transferticket in Buchfink ein (Abschlüsse, Erklärungen,
     Zeile „E-Bilanz der Eröffnungsbilanz“).
- **Buchfink liefert:** XBRL-Datei der Eröffnungsbilanz auf den
  Beurkundungstag, vor dem Export geprüft (`befunde`).
- **Nachweis:** Transferticket; der Mensch bestätigt in der App.
- **Bei Versäumnis:** Das Finanzamt kann die Übermittlung mit Zwangsgeld
  durchsetzen, je Festsetzung bis 25.000 € ([§§ 328, 329 AO][ao328]).
- **Vertiefung:** [steuererklaerungen/ebilanz.md](../steuererklaerungen/ebilanz.md),
  Abschnitt 6

<a id="kapitalertragsteuer"></a>
## Kapitalertragsteuer auf eine Ausschüttung

- **Grundlage:** [§ 43 Abs. 1 Satz 1 Nr. 1 EStG][estg43],
  [§ 44 Abs. 1, 2, 5 EStG][estg44], [§ 45a Abs. 1 EStG][estg45a].
- **Frist:** Einbehalt, Anmeldung und Abführung im Zeitpunkt des Zuflusses
  ([§ 44 Abs. 1 Satz 5 Halbsatz 2 EStG][estg44], [§ 45a Abs. 1 EStG][estg45a]).
  Zufluss am Tag, den der Beschluss als Auszahlungstag bestimmt; ohne
  Bestimmung am Tag nach der Beschlussfassung
  ([§ 44 Abs. 2 Satz 1, 2 Halbsatz 1 EStG][estg44]). Ist durch Gesetz eine
  abweichende Fälligkeit bestimmt oder lässt das Gesetz eine Bestimmung durch
  die Satzung zu, gilt ohne Auszahlungstag im Beschluss der Fälligkeitstag
  ([§ 44 Abs. 2 Satz 2 Halbsatz 2 EStG][estg44]); frag, ob der
  Gesellschaftsvertrag die Fälligkeit regelt. Buchfink rechnet mit dem Tag
  nach dem Beschluss. Nennt der Beschluss einen anderen Auszahlungstag oder
  gilt ein Fälligkeitstag, nenne dem Menschen das richtige Datum. Schlag für
  den Beschluss einen Werktag als Auszahlungstag vor. Mit Berater gleich.
- **Kirchensteuer:** Die Anlass- und die jährliche Regelabfrage beim BZSt
  (1. September bis 31. Oktober, [§ 51a Abs. 2c Satz 1 Nr. 3 EStG][estg51a])
  führt `fristen` nicht. Hinweis und offene Frage:
  [gesellschafter/ausschuettung.md](../gesellschafter/ausschuettung.md).
- **Zuständig:** du bereitest vor; der Mensch übermittelt und zahlt. Kanal:
  ELSTER.
- **Weg:**
  1. Beträge je Gesellschafter vorlegen: Ausschüttung, Kapitalertragsteuer,
     Solidaritätszuschlag, gegebenenfalls Kirchensteuer.
  2. Mein ELSTER → „Formulare und Leistungen“ → „Alle Formulare“ →
     „Kapitalertragsteuer“ → „Kapitalertragsteuer-Anmeldung nach
     Einkommensteuergesetz“. Auch ohne Steuerabzug anmelden und den Grund
     angeben ([§ 45a Abs. 1 Satz 2, 3 EStG][estg45a]).
  3. Zahlung am Zuflusstag an das Finanzamt der GmbH; die abzuführende Steuer
     auf volle Euro abrunden ([§ 44 Abs. 1 Satz 6 EStG][estg44]). Den Betrag
     nennen die Hinweise der Pflicht.
  4. Auf Verlangen Steuerbescheinigung ausstellen
     ([§ 45a Abs. 2 EStG][estg45a]).
- **Buchfink liefert:** Beträge aus dem erfassten Ergebnisverwendungsbeschluss
  (Hinweise der Pflicht in `fristen`); Konten 3519 und 3760 über `konten`.
- **Nachweis:** Transferticket; der Mensch bestätigt in der App.
- **Bei Versäumnis:** Die GmbH haftet für die Steuer
  ([§ 44 Abs. 5 EStG][estg44]), der Geschäftsführer bei grober Fahrlässigkeit
  ([§ 69 AO][ao69]); Säumniszuschlag ([§ 240 AO][ao240]); Verspätungszuschlag
  ([§ 152 Abs. 1 AO][ao152]); Geldbuße bis 25.000 € ([§ 380 AO][ao380]).
- **Vertiefung:** [gesellschafter/ausschuettung.md](../gesellschafter/ausschuettung.md),
  [gesellschafter/steuerbescheinigung.md](../gesellschafter/steuerbescheinigung.md)

[ao1]: https://www.gesetze-im-internet.de/ao_1977/__1.html
[ao69]: https://www.gesetze-im-internet.de/ao_1977/__69.html
[ao108]: https://www.gesetze-im-internet.de/ao_1977/__108.html
[ao109]: https://www.gesetze-im-internet.de/ao_1977/__109.html
[ao137]: https://www.gesetze-im-internet.de/ao_1977/__137.html
[ao138]: https://www.gesetze-im-internet.de/ao_1977/__138.html
[ao146]: https://www.gesetze-im-internet.de/ao_1977/__146.html
[ao147]: https://www.gesetze-im-internet.de/ao_1977/__147.html
[ao149]: https://www.gesetze-im-internet.de/ao_1977/__149.html
[ao150]: https://www.gesetze-im-internet.de/ao_1977/__150.html
[ao152]: https://www.gesetze-im-internet.de/ao_1977/__152.html
[ao162]: https://www.gesetze-im-internet.de/ao_1977/__162.html
[ao224]: https://www.gesetze-im-internet.de/ao_1977/__224.html
[ao240]: https://www.gesetze-im-internet.de/ao_1977/__240.html
[ao328]: https://www.gesetze-im-internet.de/ao_1977/__328.html
[ao377]: https://www.gesetze-im-internet.de/ao_1977/__377.html
[ao380]: https://www.gesetze-im-internet.de/ao_1977/__380.html
[aeao108]: https://www.steuerschroeder.de/steuergesetze/aeao/108
[bgb121]: https://www.gesetze-im-internet.de/bgb/__121.html
[bgb193]: https://www.gesetze-im-internet.de/bgb/__193.html
[egao36]: https://www.gesetze-im-internet.de/aoeg_1977/art_97__36.html
[estdv60]: https://www.gesetze-im-internet.de/estdv_1955/__60.html
[estg5b]: https://www.gesetze-im-internet.de/estg/__5b.html
[estg37]: https://www.gesetze-im-internet.de/estg/__37.html
[estg41a]: https://www.gesetze-im-internet.de/estg/__41a.html
[estg43]: https://www.gesetze-im-internet.de/estg/__43.html
[estg44]: https://www.gesetze-im-internet.de/estg/__44.html
[estg45a]: https://www.gesetze-im-internet.de/estg/__45a.html
[estg51a]: https://www.gesetze-im-internet.de/estg/__51a.html
[gewanzv1]: https://www.gesetze-im-internet.de/gewanzv_2014/__1.html
[gewanzv2]: https://www.gesetze-im-internet.de/gewanzv_2014/__2.html
[gewo14]: https://www.gesetze-im-internet.de/gewo/__14.html
[gewo15]: https://www.gesetze-im-internet.de/gewo/__15.html
[gewo146]: https://www.gesetze-im-internet.de/gewo/__146.html
[gewstg14a]: https://www.gesetze-im-internet.de/gewstg/__14a.html
[gewstg19]: https://www.gesetze-im-internet.de/gewstg/__19.html
[gmbhg5a]: https://www.gesetze-im-internet.de/gmbhg/__5a.html
[gmbhg16]: https://www.gesetze-im-internet.de/gmbhg/__16.html
[gmbhg29]: https://www.gesetze-im-internet.de/gmbhg/__29.html
[gmbhg11]: https://www.gesetze-im-internet.de/gmbhg/__11.html
[gmbhg40]: https://www.gesetze-im-internet.de/gmbhg/__40.html
[gmbhg42a]: https://www.gesetze-im-internet.de/gmbhg/__42a.html
[gmbhg46]: https://www.gesetze-im-internet.de/gmbhg/__46.html
[gmbhg48]: https://www.gesetze-im-internet.de/gmbhg/__48.html
[gwg3]: https://www.gesetze-im-internet.de/gwg_2017/__3.html
[gwg19]: https://www.gesetze-im-internet.de/gwg_2017/__19.html
[gwg20]: https://www.gesetze-im-internet.de/gwg_2017/__20.html
[gwg24]: https://www.gesetze-im-internet.de/gwg_2017/__24.html
[gwg56]: https://www.gesetze-im-internet.de/gwg_2017/__56.html
[gwg57]: https://www.gesetze-im-internet.de/gwg_2017/__57.html
[hgb12]: https://www.gesetze-im-internet.de/hgb/__12.html
[hgb239]: https://www.gesetze-im-internet.de/hgb/__239.html
[hgb240]: https://www.gesetze-im-internet.de/hgb/__240.html
[hgb242]: https://www.gesetze-im-internet.de/hgb/__242.html
[hgb243]: https://www.gesetze-im-internet.de/hgb/__243.html
[hgb245]: https://www.gesetze-im-internet.de/hgb/__245.html
[hgb257]: https://www.gesetze-im-internet.de/hgb/__257.html
[hgb264]: https://www.gesetze-im-internet.de/hgb/__264.html
[hgb267a]: https://www.gesetze-im-internet.de/hgb/__267a.html
[hgb325]: https://www.gesetze-im-internet.de/hgb/__325.html
[hgb326]: https://www.gesetze-im-internet.de/hgb/__326.html
[hgb335]: https://www.gesetze-im-internet.de/hgb/__335.html
[kstg7]: https://www.gesetze-im-internet.de/kstg_1977/__7.html
[kstg27]: https://www.gesetze-im-internet.de/kstg_1977/__27.html
[kstg31]: https://www.gesetze-im-internet.de/kstg_1977/__31.html
[owig10]: https://www.gesetze-im-internet.de/owig_1968/__10.html
[sgb7_192]: https://www.gesetze-im-internet.de/sgb_7/__192.html
[solzg3]: https://www.gesetze-im-internet.de/solzg_1995/__3.html
[stgb283]: https://www.gesetze-im-internet.de/stgb/__283.html
[stgb283b]: https://www.gesetze-im-internet.de/stgb/__283b.html
[trgebv]: https://www.gesetze-im-internet.de/trgebv_2020/anlage.html
[ustdv46]: https://www.gesetze-im-internet.de/ustdv_1980/__46.html
[ustdv47]: https://www.gesetze-im-internet.de/ustdv_1980/__47.html
[ustdv48]: https://www.gesetze-im-internet.de/ustdv_1980/__48.html
[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg18]: https://www.gesetze-im-internet.de/ustg_1980/__18.html
[ustg18a]: https://www.gesetze-im-internet.de/ustg_1980/__18a.html
[ustg26a]: https://www.gesetze-im-internet.de/ustg_1980/__26a.html
[gobd]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf
[gobd2024]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2024-03-11-aenderung-gobd.pdf
[gobd2025]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2025-07-14-GoBD-2-aenderung.pdf
[elsterformulare]: https://www.elster.de/eportal/formulare-leistungen/alleformulare
[bzstzm]: https://www.bzst.de/DE/Unternehmen/Umsatzsteuer/ZusammenfassendeMeldung/ElektronischeAbgabe/elektronischeabgabe_zm_node.html
[bzstzmfrist]: https://www.bzst.de/DE/Unternehmen/Umsatzsteuer/ZusammenfassendeMeldung/Fristen/fristen_node.html
[pp]: https://publikations-plattform.de
[ppveroeff]: https://publikations-plattform.de/order/de/wissenswertes/sogehts/javeroeffentlichung
[ppident]: https://publikations-plattform.de/order/de/wissenswertes/sogehts/registrierenidentifizeren
[ppdaten]: https://publikations-plattform.de/order/de/wissenswertes/sogehts/meinedaten
[trhilfe]: https://www.transparenzregister.de/hilfe-center
[trhilfeapi]: https://www.transparenzregister.de/connector/api/help-center
