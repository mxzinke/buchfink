# Pflichtenkalender

Rechtsstand: 17. Oktober 2026. Wege in Mein ELSTER, im BZSt-Online-Portal, auf
der Publikations-Plattform und im Transparenzregister nach deren Seiten vom
selben Tag.

Jeder Abschnitt ergänzt `fristen` um den **Weg**, den **Nachweis**, die Folgen
**bei Versäumnis** und Wertungsfragen. Den Nachweis erfasst oder bestätigt der
Mensch in der App.

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

1. **Wählen.** Überfällig vor offen, dann nach Fälligkeit. Lies den Abschnitt
   unter `anker` und seine Vertiefung.
2. **Vorbereiten** nach der Vertiefung: Zahlen, Entwürfe und Erinnerungen,
   bei `zustaendig: steuerberater` die Unterlagen für den Berater. Fertig, wenn jede Zahl geprüft und vom Menschen bestätigt ist und
   die Exportdateien im Exportordner liegen.
3. **Übergeben.** Melde `pflicht_vorbereitet` und nenne dem Menschen in einer
   Nachricht Pflicht und Zeitraum, Fälligkeit, Weg, Werte oder Datei und den
   Nachweis.
4. **Nachhalten.** Steht die Pflicht in der nächsten Sitzung noch auf
   vorbereitet, frag nach.

Dein Teil ist fertig, wenn die Pflicht auf vorbereitet steht und der Mensch
Weg, Fälligkeit und Nachweis kennt.

- **Überfällig:** Frag, ob der Mensch die Pflicht außerhalb von Buchfink
  erfüllt hat; dann bestätigt er sie mit Nachweis. Sonst nenne die Folgen und
  den kürzesten Weg.
- **Entfällt** setzt nur der Mensch, mit Grund.
- **Anlässe** (Gesellschafterwechsel, Änderung der wirtschaftlich Berechtigten,
  Vorauszahlungsbescheid) kennt `fristen` erst, wenn du sie nach Rückfrage mit
  `anlass_erfassen` erfasst hast; Bescheid oder Vertrag legst du vorher ab.
- **Falsche Fälligkeit** kommt aus falschen Stammdaten (Bundesland,
  Voranmeldungszeitraum, Dauerfristverlängerung, Gründung, Eintragung,
  steuerliche Beratung); schlag die Korrektur vor
  ([SKILL.md, Abschnitt 4](../../SKILL.md#4-nur-der-mensch-in-der-app)).
- **Steuerberater:** Eine längere Frist haben nur Körperschaftsteuer-,
  Gewerbesteuer- und Umsatzsteuererklärung und E-Bilanzen, und nur, wenn der
  Berater mit der Erstellung beauftragt ist ([§ 149 Abs. 3 AO][ao149]); ist er
  nur eingetragen, bitte den Menschen, das Stammdatum zu korrigieren. Dem
  Berater stellst du Saldenliste, Bilanz, GuV, Anlagenspiegel und offene Fragen
  zusammen; Nachweis ist das Transferticket, das er nennt.

## Fristende, Zahlung, Verlängerung

- **Steuerliche Fristen**, auch Zahlungstermine, enden an einem Samstag,
  Sonntag oder Feiertag mit Ablauf des nächsten Werktags
  ([§ 108 Abs. 3 AO][ao108], [AEAO zu § 108 Nr. 1][aeao108]; Gewerbesteuer
  über [§ 1 Abs. 2 Nr. 3 AO][ao1]). `fristen` verschiebt selbst und nennt das
  gesetzliche Ende in `gesetzliche_frist`.
- **Fristen aus HGB, GmbHG, GwG und GewO** verschiebt `fristen` nicht, § 108
  Abs. 3 AO gilt dort nicht: Plane mit dem Kalenderdatum.
- **Unverzüglich** heißt ohne schuldhaftes Zögern
  ([§ 121 Abs. 1 Satz 1 BGB][bgb121]); `fristen` setzt den Tag des Anlasses.
- **Zahlung:** Eine Überweisung gilt am Tag der Gutschrift beim Finanzamt als
  entrichtet, eine Lastschrift am Fälligkeitstag
  ([§ 224 Abs. 2 Nr. 2, 3 AO][ao224]).
- **Säumniszuschlag:** 1 % je angefangenem Monat auf den auf 50 € abgerundeten
  Rückstand; bei bis zu drei Tagen Säumnis keiner, außer bei Bar- oder
  Scheckzahlung ([§ 240 Abs. 1, 3 AO][ao240]).
- **Verspätungszuschlag:**
  [steuererklaerungen/fristen-zuschlaege.md](../steuererklaerungen/fristen-zuschlaege.md#4-verspätungszuschlag).
- **Verlängerung** von Erklärungsfristen auf Antrag, auch rückwirkend
  ([§ 109 Abs. 1 AO][ao109]); im Berater-Fall über den letzten Februartag und
  nach einer Vorabanforderung über deren Termin hinaus nur ohne Verschulden
  ([§ 109 Abs. 2 Satz 1 Nr. 1, 2 AO][ao109]). Mein ELSTER: „Anträge und
  Nachrichten“ → „Antrag auf Fristverlängerung“.
- **Mein ELSTER:** Jedes Formular steht unter „Formulare und Leistungen“ →
  „Alle Formulare“, nach Steuerart ([Formularliste][elsterformulare]). Eingabe,
  Senden, Protokoll:
  [steuererklaerungen/abgabe.md](../steuererklaerungen/abgabe.md#4-eingabe-in-mein-elster).

<a id="ustva"></a>
## Umsatzsteuer-Voranmeldung

- **Frist:** Übermittlung und Zahlung am 10. Tag nach Ablauf des
  Voranmeldungszeitraums ([§ 18 Abs. 1 Satz 1, 4 UStG][ustg18]), mit
  Dauerfristverlängerung einen Monat später ([§ 46 Satz 1 UStDV][ustdv46]).
  Zeitraum Monat oder Vierteljahr ([§ 18 Abs. 2, 2a UStG][ustg18]); schlag ihn
  mit `stammdaten_vorschlagen` (`ust_zeitraum`) vor.
- **Weg:**
  1. `erklaerung` (`art: ustva`, `zeitraum`) prüfen und dem Menschen die
     Kennzahlen als Tabelle vorlegen.
  2. Mein ELSTER → „Formulare und Leistungen“ → „Alle Formulare“ →
     „Umsatzsteuer“ → „Umsatzsteuer-Voranmeldung“, Jahr wählen.
  3. Der Mensch trägt Zeitraum und Kennzahlen ein, lässt prüfen, vergleicht mit
     der Tabelle und sendet.
  4. Zahllast bis zur Fälligkeit zahlen oder per Lastschrift einziehen lassen.
- **Nachweis:** Transferticket an der Voranmeldung. Buchfink nimmt es erst nach
  der Festschreibung des Zeitraums an; den letzten Monat schreibt der Mensch
  deshalb spätestens am Fälligkeitstag fest
  ([monatsfestschreibung](#monatsfestschreibung)).
- **Bei Versäumnis:** Verspätungszuschlag ([§ 152 Abs. 8 Nr. 1 AO][ao152]);
  Säumniszuschlag ([§ 240 AO][ao240]); Geldbuße bis 30.000 € bei vorsätzlich
  nicht oder nicht rechtzeitig entrichteter Vorauszahlung
  ([§ 26a Abs. 1, 3 UStG][ustg26a]).
- **Vertiefung:** [monatsabschluss/ustva.md](../monatsabschluss/ustva.md)

<a id="sondervorauszahlung"></a>
## Sondervorauszahlung

- **Voraussetzung:** monatliche Voranmeldung mit Dauerfristverlängerung
  ([§ 47 Abs. 1 UStDV][ustdv47]). Den Betrag rechnest du aus den
  Voranmeldungen des Vorjahres (`erklaerung`) nach
  [monatsabschluss/dauerfristverlaengerung.md, Abschnitt 3](../monatsabschluss/dauerfristverlaengerung.md#3-sondervorauszahlung).
- **Frist:** beim ersten Antrag bis zum Abgabetag der ersten Voranmeldung, für
  die die Dauerfristverlängerung gelten soll ([§ 48 Abs. 1 UStDV][ustdv48]),
  danach jährlich bis zum 10. Februar ([§ 48 Abs. 2 UStDV][ustdv48]).
  Berechnen, anmelden und zahlen am selben Tag.
- **Weg:** Betrag mit Herkunft vorlegen. Mein ELSTER, unter „Umsatzsteuer“ das
  Formular „Dauerfristverlängerung/Sondervorauszahlung (monatlich)“; der Mensch
  sendet und zahlt. Dauerfristverlängerung und Betrag schlägst du mit
  `stammdaten_vorschlagen` vor (`dauerfrist`, `sondervorauszahlung`).
- **Nachweis:** Transferticket und Zahlung auf 3830; die Zahlung allein belegt
  die Anmeldung nicht ([§ 48 Abs. 2 UStDV][ustdv48]). Buchfink setzt die
  Pflicht schon mit der Buchung auf 3830 auf erledigt; frag dann nach dem
  Transferticket.
- **Bei Versäumnis:** Verspätungszuschlag, nie zwingend
  ([§ 152 Abs. 3 Nr. 4, Abs. 8 Nr. 5 AO][ao152]); Festsetzung durch das
  Finanzamt ([§ 48 Abs. 3 UStDV][ustdv48]); Säumniszuschlag ([§ 240 AO][ao240]).
- **Vertiefung:** [monatsabschluss/dauerfristverlaengerung.md](../monatsabschluss/dauerfristverlaengerung.md)

<a id="zm"></a>
## Zusammenfassende Meldung

- **Frist:** ohne Dauerfristverlängerung ([BZSt, Fristen][bzstzmfrist]).
- **Weg:** `erklaerung` (`art: zm`) prüfen (Summe gleich Kennzahl 21 der
  Voranmeldungen des Zeitraums) und die Zeilen als Tabelle vorlegen; eine
  CSV-Datei erzeugt Buchfink nicht. Mein ELSTER, unter „Umsatzsteuer“ das
  Formular „Zusammenfassende Meldung“, oder das BZSt-Online-Portal mit dem
  Fachverfahren ZM ([BZSt, Elektronische Abgabe][bzstzm]). Der Mensch
  vergleicht mit deiner Tabelle und sendet.
- **Nachweis:** Transferticket an der ZM.
- **Bei Versäumnis:** kein Verspätungszuschlag ([§ 18a Abs. 11 UStG][ustg18a]);
  Geldbuße bis 5.000 € bei vorsätzlich oder leichtfertig fehlender, falscher,
  verspäteter oder nicht binnen eines Monats berichtigter Meldung
  ([§ 26a Abs. 2 Nr. 5, Abs. 3 UStG][ustg26a], [§ 18a Abs. 10 UStG][ustg18a]).
- **Vertiefung:** [monatsabschluss/zm.md](../monatsabschluss/zm.md)

<a id="kst-vorauszahlung"></a>
## Körperschaftsteuer-Vorauszahlung

- **Im Kalender** erst mit erfasstem Vorauszahlungsbescheid oder gebuchten
  Vorauszahlungen im Vorjahr oder Jahr; die Höhe steht im Bescheid. Bei
  abweichendem Wirtschaftsjahr gehören sie in das Wirtschaftsjahr, das im
  Veranlagungszeitraum endet ([§ 31 Abs. 2 KStG][kstg31]).
- **Weg:** Überweisung an die Finanzkasse mit Steuernummer und Angaben aus dem
  Bescheid oder SEPA-Lastschrift. Anpassung: Mein ELSTER, „Antrag auf
  Anpassung von Vorauszahlungen“
  ([steuererklaerungen/bescheide.md](../steuererklaerungen/bescheide.md)).
- **Nachweis:** Zahlung auf 7600 oder 7608 um den Termin, vom Bankkonto oder
  vom Menschen gebucht (deine Handbuchung zählt nicht), oder Bestätigung des
  Menschen. Zuordnung:
  [bank/steuerzahlungen.md](../bank/steuerzahlungen.md#2-konten-je-steuerart).
- **Bei Versäumnis:** Säumniszuschlag ([§ 240 AO][ao240]).
- **Vertiefung:** [steuererklaerungen/bescheide.md](../steuererklaerungen/bescheide.md)

<a id="gewst-vorauszahlung"></a>
## Gewerbesteuer-Vorauszahlung

- **Im Kalender** wie die [kst-vorauszahlung](#kst-vorauszahlung). Je Termin
  ein Viertel der Steuer der letzten Veranlagung oder laut Anpassungsbescheid;
  eine nachträgliche Erhöhung ist binnen eines Monats nach Bekanntgabe zu
  zahlen; festgesetzt wird erst ab 50 €
  ([§ 19 Abs. 1–3, 5 Satz 2 GewStG][gewstg19]). Bei abweichendem
  Wirtschaftsjahr gilt das Wirtschaftsjahr, das im Erhebungszeitraum endet
  ([§ 19 Abs. 1 Satz 2, 3 GewStG][gewstg19]).
- **Weg:** Überweisung an die Gemeindekasse mit dem Kassenzeichen oder
  SEPA-Lastschrift bei der Gemeinde.
- **Nachweis:** wie bei der [kst-vorauszahlung](#kst-vorauszahlung), auf 7610.
- **Bei Versäumnis:** Säumniszuschlag ([§ 240 AO][ao240] über
  [§ 1 Abs. 2 Nr. 5 AO][ao1]).
- **Vertiefung:** [steuererklaerungen/gewst.md](../steuererklaerungen/gewst.md),
  [bank/steuerzahlungen.md](../bank/steuerzahlungen.md)

<a id="lohnsteuer-anmeldung"></a>
## Lohnsteuer-Anmeldung

- **Zeitraum:** Monat; Quartal bei abzuführender Lohnsteuer des Vorjahres über
  1.080 € bis 5.000 €; Kalenderjahr bis 1.080 €; Teiljahr hochrechnen
  ([§ 41a Abs. 2 EStG][estg41a]). Stammt der Betrag in `hinweise` aus Konto
  3730, enthält er womöglich Kirchensteuer und Solidaritätszuschlag: Prüfe ihn
  an den Anmeldungen des Vorjahres (Protokolle des Lohnbüros) und nenne dem
  Menschen bei Abweichung Zeitraum und Fälligkeiten.
- **Entfällt**, wenn die GmbH keine Arbeitnehmer mehr beschäftigt und das dem
  Finanzamt mitteilt ([§ 41a Abs. 1 Satz 4 EStG][estg41a]).
- **Weg:** Meist übermittelt das Lohnbüro aus der Lohnsoftware; sonst Mein
  ELSTER, unter „Lohnsteuer Arbeitgeber“ das Formular „Lohnsteuer-Anmeldung“.
  Zahlung bis zur Fälligkeit.
- **Nachweis:** Übermittlungsprotokoll mit Transferticket und die gebuchte
  Zahlung; verlangt sind Anmeldung und Abführung
  ([§ 41a Abs. 1 Satz 1 Nr. 1, 2 EStG][estg41a]). Ein gebuchtes Lohnjournal
  meldet den Zeitraum nur als vorbereitet. Bitte das Lohnbüro je Zeitraum um
  das Protokoll und lege es ab.
- **Bei Versäumnis:** Verspätungszuschlag ([§ 152 Abs. 8 Nr. 1, 2 AO][ao152]);
  Säumniszuschlag ([§ 240 AO][ao240]); Geldbuße bis 25.000 € bei vorsätzlich
  oder leichtfertig nicht rechtzeitig abgeführten Steuerabzugsbeträgen
  ([§ 380 AO][ao380]); Haftung des Geschäftsführers bei grober Fahrlässigkeit
  ([§ 69 AO][ao69]).
- **Vertiefung:** [lohn/ablauf.md](../lohn/ablauf.md), Abschnitt 4;
  [lohn/abstimmung.md](../lohn/abstimmung.md)

<a id="monatsfestschreibung"></a>
## Monatsfestschreibung

- **Frist:** gesetzlich keine
  ([gobd.md, Abschnitte 5 und 7](gobd.md#7-unveränderbarkeit-storno-festschreibung)).
  Den letzten Monat eines Voranmeldungszeitraums schreibt der Mensch
  spätestens am Fälligkeitstag der Voranmeldung fest ([ustva](#ustva));
  `fristen` zeigt für diesen Monat das spätere Datum.
- **Weg:** Du klärst die Befunde aus `auswertung` (`art: pruefbericht`); der
  Mensch schreibt fest und übergeht blockierende Befunde nur mit Begründung.
- **Nachweis:** die Festschreibung.
- **Bei Versäumnis:** Die Voranmeldung bleibt unbestätigt und wird nach ihrer
  Fälligkeit überfällig.
- **Vertiefung:** [monatsabschluss/festschreibung.md](../monatsabschluss/festschreibung.md)

<a id="bilanzstichtag"></a>
## Bilanzstichtag: Inventar und Abschlussbuchungen

- **Weg:** Inventar zum Schluss des Geschäftsjahres
  ([§ 240 Abs. 2 HGB][hgb240]) aus Saldenliste und Nachweisen des Menschen
  (etwa Saldenbestätigungen); Abschlussbuchungen nach Rückfrage
  ([jahresabschluss/ablauf.md](../jahresabschluss/ablauf.md)); danach schreibt
  der Mensch das Geschäftsjahr fest.
- **Nachweis:** die Festschreibung des Geschäftsjahres.
- **Bei Versäumnis:** Wer Bilanz oder Inventar nicht rechtzeitig aufstellt, ist
  strafbar, wenn die Gesellschaft ihre Zahlungen einstellt oder ein
  Insolvenzverfahren eröffnet oder mangels Masse abgelehnt wird
  ([§ 283b Abs. 1 Nr. 3 Buchst. b, Abs. 3 StGB][stgb283b],
  [§ 283 Abs. 6 StGB][stgb283]).
- **Vertiefung:** [jahresabschluss/abschlussbuchungen.md](../jahresabschluss/abschlussbuchungen.md)

<a id="aufstellung"></a>
## Aufstellung des Jahresabschlusses

- **Weg:**
  1. Du übergibst Bilanz, GuV, Anhangentwurf und die Liste der
     Abschlussposten.
  2. Die Geschäftsführung unterzeichnet mit Datum ([§ 245 HGB][hgb245]).
  3. Der Mensch speichert den unterschriebenen Abschluss in den Eingang, du
     legst ihn auf seine Anweisung ab, und er setzt in der App „aufgestellt“.
- **Nachweis:** der Stand „aufgestellt“.
- **Bei Versäumnis:** Strafbarkeit wie beim [bilanzstichtag](#bilanzstichtag);
  das Ordnungsgeldverfahren zur Offenlegung wartet nicht auf die Aufstellung
  ([§ 335 Abs. 1 Satz 3 HGB][hgb335]).
- **Vertiefung:** [jahresabschluss/aufstellen-feststellen-offenlegen.md](../jahresabschluss/aufstellen-feststellen-offenlegen.md)

<a id="feststellung"></a>
## Feststellung des Jahresabschlusses

- **Frist:** Der Gesellschaftsvertrag kann sie nicht verlängern
  ([§ 42a Abs. 2 GmbHG][gmbhg42a]).
- **Weg:**
  1. Die Geschäftsführung legt den aufgestellten Abschluss unverzüglich den
     Gesellschaftern vor; du entwirfst den Beschluss (Datum, Bilanzsumme,
     Ergebnis, Ergebnisverwendung).
  2. Die Gesellschafter beschließen ([§ 46 Nr. 1 GmbHG][gmbhg46]), der
     Alleingesellschafter mit unterschriebener Niederschrift
     ([§ 48 GmbHG][gmbhg48]).
  3. Der Mensch schreibt das Jahr fest und setzt „festgestellt“ mit Datum und
     Bezug auf den Beschluss; du legst den Beschluss ab.
- **Nachweis:** der Stand „festgestellt“.
- **Bei Versäumnis:** Ohne Feststellung fehlt der Gegenstand der Offenlegung
  ([§ 325 Abs. 1 Satz 1 Nr. 1 HGB][hgb325]); das Ordnungsgeld droht trotzdem
  ([§ 335 Abs. 1 Satz 3 HGB][hgb335]).
- **Vertiefung:** [jahresabschluss/aufstellen-feststellen-offenlegen.md](../jahresabschluss/aufstellen-feststellen-offenlegen.md)

<a id="ergebnisverwendung"></a>
## Beschluss über die Ergebnisverwendung

- **Weg:** Beschluss mit der Feststellung, bei der UG über das Ergebnis nach
  Einstellung in die gesetzliche Rücklage ([§ 5a Abs. 3 GmbHG][gmbhg5a]). Lege
  ihn ab und buche ihn nach Rückfrage mit `ergebnisverwendung_buchen`. Bei
  einer Ausschüttung bestimmt der Beschluss den Auszahlungstag; daraus folgt
  die Pflicht [kapitalertragsteuer](#kapitalertragsteuer).
- **Nachweis:** der mit `ergebnisverwendung_buchen` erfasste Beschluss.
- **Bei Versäumnis:** Über die Verwendung bestimmen allein die Gesellschafter
  ([§ 46 Nr. 1 GmbHG][gmbhg46]); ohne Beschluss keine Ausschüttung.
- **Vertiefung:** [jahresabschluss/ergebnisverwendung-ug.md](../jahresabschluss/ergebnisverwendung-ug.md),
  [gesellschafter/ausschuettung.md](../gesellschafter/ausschuettung.md)

<a id="offenlegung"></a>
## Offenlegung im Unternehmensregister

- **Umfang:** der festgestellte Abschluss in deutscher Sprache, auch eine
  spätere Änderung ([§ 325 Abs. 1b HGB][hgb325]); je Größenklasse:
  [jahresabschluss/aufstellen-feststellen-offenlegen.md](../jahresabschluss/aufstellen-feststellen-offenlegen.md#4-offenlegen).
- **Weg** ([Publikations-Plattform][pp], [So geht's][ppveroeff]):
  1. Einmal registrieren und vor der ersten Übermittlung je übermittelnder
     Person elektronisch identifizieren (Video-Verfahren, Online-Ausweis oder
     Steuerberaterplattform; Gebühr nach JVKostG;
     [Registrieren/Identifizieren][ppident]).
  2. „Unternehmensregister“ → „Veröffentlichen“ oder, nur Kleinst,
     „Hinterlegen“.
  3. Datei hochladen (XML; PDF oder Office gegen Konvertierungsentgelt) oder,
     nur kleine Gesellschaften, das Formular ausfüllen. Die Bilanz liefert
     `auswertung` (`art: bilanz`); eine Datei für die Plattform erzeugt
     Buchfink nicht.
  4. Unter „Meine Daten“ den Auftrag verfolgen ([Meine Daten][ppdaten]).
- **Nachweis:** Beleg der Veröffentlichung oder Hinterlegung aus „Meine Daten“;
  der Mensch setzt damit „offengelegt“, du legst den Beleg ab.
- **Bei Versäumnis:** Das Bundesamt für Justiz droht ein Ordnungsgeld von
  2.500 € bis 25.000 € mit sechs Wochen Frist an, wiederholt bis zur
  Offenlegung, dazu Verfahrenskosten; nach Fristablauf erfüllt, sinkt es auf
  500 € (hinterlegende Kleinstgesellschaft), 1.000 € (kleine) oder 2.500 €
  ([§ 335 Abs. 1, 3, 4 HGB][hgb335]). Unverschuldet versäumt: Wiedereinsetzung
  binnen zwei Wochen nach Wegfall des Hindernisses, Offenlegung binnen sechs
  Wochen ([§ 335 Abs. 5 HGB][hgb335]). Leg eine Androhung sofort ab und nenne
  dem Menschen die Sechswochenfrist.
- **Vertiefung:** [jahresabschluss/aufstellen-feststellen-offenlegen.md](../jahresabschluss/aufstellen-feststellen-offenlegen.md),
  [jahresabschluss/groessenklassen-anhang.md](../jahresabschluss/groessenklassen-anhang.md)

<a id="ebilanz"></a>
## E-Bilanz

- **Weg:**
  1. Abschluss aufgestellt; Bilanz und GuV mit `auswertung` geprüft.
  2. `ebilanz_exportieren` schreibt die geprüfte XBRL-Datei in den
     Exportordner.
  3. Der Mensch importiert sie in einen Online-Dienst mit ELSTER-Anbindung,
     gleicht mit dir ab, validiert, sendet und trägt Protokoll und
     Transferticket in Buchfink ein.
  Prüfpunkte und Rückfallweg (`format: susa_csv`):
  [steuererklaerungen/ebilanz.md, Abschnitte 4 und 5](../steuererklaerungen/ebilanz.md#4-prüfpunkte).
- **Weitere Wege:** Steuerberater; bei unbilliger Härte Verzicht auf Antrag
  ([§ 5b Abs. 2 EStG][estg5b] i. V. m. [§ 150 Abs. 8 AO][ao150]), dann auf
  Papier ([§ 60 EStDV][estdv60];
  [steuererklaerungen/ebilanz.md, Abschnitt 5](../steuererklaerungen/ebilanz.md#5-übermittlungsweg)).
- **Nachweis:** Transferticket.
- **Bei Versäumnis:** Die Körperschaftsteuererklärung ist unvollständig; das
  Finanzamt kann die Übermittlung mit Zwangsgeld bis 25.000 € je Festsetzung
  durchsetzen ([§§ 328, 329 AO][ao328]).
- **Vertiefung:** [steuererklaerungen/ebilanz.md](../steuererklaerungen/ebilanz.md)

<a id="kst-erklaerung"></a>
## Körperschaftsteuererklärung

- **Einlagekonto:** Mit der Erklärung geht die Feststellungserklärung zum
  steuerlichen Einlagekonto ([§ 27 Abs. 2 Satz 4 KStG][kstg27]). Bei
  abweichendem Wirtschaftsjahr kann ihre Frist früher enden
  ([§ 149 Abs. 2 Satz 1 AO][ao149]): Plane mit dem früheren Datum oder lass den
  Menschen beim Finanzamt nachfragen.
- **Weg:** Mein ELSTER, unter „Körperschaftsteuer“ das Formular
  „Körperschaftsteuererklärung (KSt 1)“ mit Anlagen. Der Mensch überträgt die
  Werte aus `erklaerung` (`art: kst`; offene Angaben über
  `erklaerung_eingabe`), lässt prüfen und sendet.
- **Nachweis:** Transferticket.
- **Bei Versäumnis:** Verspätungszuschlag, auch auf die
  Feststellungserklärungen (Einlagekonto, Verlustvortrag):
  [steuererklaerungen/fristen-zuschlaege.md, Abschnitt 4](../steuererklaerungen/fristen-zuschlaege.md#4-verspätungszuschlag).
  Das Finanzamt kann schätzen, die Abgabepflicht bleibt ([§ 162 AO][ao162],
  [§ 149 Abs. 1 Satz 4 AO][ao149]), und die Abgabe mit Zwangsgeld durchsetzen
  ([§§ 328, 329 AO][ao328]).
- **Vertiefung:** [steuererklaerungen/kst.md](../steuererklaerungen/kst.md),
  [steuererklaerungen/abgabe.md](../steuererklaerungen/abgabe.md)

<a id="gewst-erklaerung"></a>
## Gewerbesteuererklärung

- **Weg:** Mein ELSTER, unter „Gewerbesteuer“ das Formular
  „Gewerbesteuererklärung (GewSt 1 A)“ an das Finanzamt, Werte aus
  `erklaerung` (`art: gewst`). Bei Betriebsstätten in mehreren Gemeinden
  zusätzlich die „Erklärung für die Zerlegung des Gewerbesteuermessbetrages
  (GewSt 1 D)“ ([§ 14a Satz 1 GewStG][gewstg14a]).
- **Nachweis:** Transferticket.
- **Bei Versäumnis:** Verspätungszuschlag, Schätzung und Zwangsgeld wie bei der
  [kst-erklaerung](#kst-erklaerung).
- **Vertiefung:** [steuererklaerungen/gewst.md](../steuererklaerungen/gewst.md)

<a id="ust-erklaerung"></a>
## Umsatzsteuererklärung

- **Frist:** Endet die Tätigkeit im Jahr, ein Monat nach Ablauf des kürzeren
  Besteuerungszeitraums, ohne Berater-Frist
  ([§ 18 Abs. 3 Satz 2 UStG][ustg18], [§ 149 Abs. 5 AO][ao149]). Eine
  Abschlusszahlung ist einen Monat nach Eingang der Erklärung fällig
  ([§ 18 Abs. 4 Satz 1 UStG][ustg18]).
- **Weg:** Mein ELSTER, unter „Umsatzsteuer“ das Formular
  „Umsatzsteuererklärung“, Werte aus `erklaerung` (`art: ust_jahr`).
- **Nachweis:** Transferticket.
- **Bei Versäumnis:** Verspätungszuschlag, Schätzung und Zwangsgeld wie bei der
  [kst-erklaerung](#kst-erklaerung); Säumniszuschlag auf die Abschlusszahlung
  ([§ 240 AO][ao240]).
- **Vertiefung:** [steuererklaerungen/ust-jahr.md](../steuererklaerungen/ust-jahr.md)

<a id="transparenzregister"></a>
## Mitteilung an das Transparenzregister

- **Anlass:** Eintragung der GmbH ins Handelsregister, ab der sie
  mitteilungspflichtig ist ([§ 20 Abs. 1 Satz 1 GwG][gwg20],
  [§ 11 Abs. 1 GmbHG][gmbhg11]), und jede Änderung der wirtschaftlich
  Berechtigten oder ihrer Angaben ([§ 19 Abs. 1 GwG][gwg19]). Eine Änderung
  erfasst du nach Rückfrage mit `anlass_erfassen`; ändert ein
  Gesellschafterwechsel die Berechtigten nicht, setzt der Mensch die Pflicht
  mit Grund auf entfallen.
- **Wirtschaftlich berechtigt** ist jede natürliche Person, die unmittelbar
  oder mittelbar mehr als 25 % der Kapitalanteile hält, mehr als 25 % der
  Stimmrechte kontrolliert oder auf vergleichbare Weise Kontrolle ausübt;
  mittelbar über eine Vereinigung, die sie nach § 290 Abs. 2–4 HGB beherrscht
  ([§ 3 Abs. 2 Satz 1–4 GwG][gwg3]). Prüfe die Kontrolle je Ebene der
  Holdingkette, ohne Quoten zu multiplizieren. Lässt sich niemand ermitteln,
  gilt der gesetzliche Vertreter als wirtschaftlich Berechtigter
  ([§ 3 Abs. 2 Satz 5 GwG][gwg3]). Mittelbare Beteiligungen kennt Buchfink
  nicht; frag danach.
- **Weg** ([Hilfe-Center][trhilfe]; ohne Browser als [JSON][trhilfeapi]):
  transparenzregister.de → „Nutzerkonto anlegen“ (Aktivierungslink binnen
  24 Stunden bestätigen, „Eintragungen vornehmen“ wählen); GmbH suchen, wirtschaftlich Berechtigte angeben, Berechtigung
  versichern, übermitteln; bei einer Änderung „Erste Mitteilung oder
  Folgemitteilung“ mit Gültigkeitsdatum. Eine schon eingetragene GmbH i. G.
  meldet den Wegfall von „in Gründung“ als Änderung. Die Eingangsmitteilung
  (PDF) sofort herunterladen; sie steht nur 14 Tage im Konto.
- **Nachweis:** Vorgangsnummer der Eingangsmitteilung; du legst das PDF ab. Die
  Eintragung belegt erst ein Registerauszug.
- **Bei Versäumnis:** Geldbuße bis 150.000 € bei Vorsatz, bis 100.000 € bei
  Leichtfertigkeit, bei schwerwiegenden, wiederholten oder systematischen
  Verstößen bis 1 Mio. € oder das Zweifache des Vorteils
  ([§ 56 Abs. 1 Satz 1 Nr. 55, Satz 2, Abs. 3 GwG][gwg56]); Veröffentlichung
  mit Namen ([§ 57 Abs. 1 GwG][gwg57]).
- **Gebühr:** jährlich, seit 2024 19,80 € ([§ 24 Abs. 1 GwG][gwg24],
  [TrGebV, Anlage Nr. 1][trgebv]).
- **Vertiefung:** Reihenfolge in der Gründung:
  [gruendung/gruendungsweg.md](../gruendung/gruendungsweg.md#4-behördenwege-mit-fristen)

<a id="gesellschafterliste"></a>
## Neue Gesellschafterliste

- **Anlass:** jede Veränderung in den Personen der Gesellschafter oder im
  Umfang ihrer Beteiligung ([§ 40 Abs. 1 GmbHG][gmbhg40]). Du erfasst sie nach
  Rückfrage als `gesellschafterwechsel` mit `anlass_erfassen`; daraus folgen
  diese Pflicht und [transparenzregister](#transparenzregister).
- **Weg:** Hat ein Notar an der Veränderung mitgewirkt, reicht er die Liste ein
  und schickt der GmbH eine Abschrift ([§ 40 Abs. 2 GmbHG][gmbhg40]). Sonst
  unterschreiben die Geschäftsführer die Liste oder signieren sie qualifiziert
  elektronisch ([§ 40 Abs. 1 GmbHG][gmbhg40]) und reichen sie elektronisch
  beim Handelsregister ein ([§ 12 Abs. 2 HGB][hgb12]); den technischen Weg
  klärt der Mensch mit dem Gericht oder lässt einen Notar einreichen.
- **Nachweis:** Datum der Einreichung und Abschrift oder Bestätigung des
  Notars; du legst die Abschrift ab.
- **Bei Versäumnis:** Gegenüber der GmbH gilt als Gesellschafter nur, wer in
  der aufgenommenen Liste steht ([§ 16 Abs. 1 GmbHG][gmbhg16]). Die
  Geschäftsführer haften den Betroffenen und den Gläubigern als
  Gesamtschuldner ([§ 40 Abs. 3 GmbHG][gmbhg40]).
- **Vertiefung:** [gruendung/gruendungsweg.md](../gruendung/gruendungsweg.md#5-nach-der-eintragung)

<a id="fragebogen-steuerliche-erfassung"></a>
## Fragebogen zur steuerlichen Erfassung

- **Weg:** Mein ELSTER, unter „Fragebogen zur steuerlichen Erfassung“ das
  Formular „Fragebogen zur steuerlichen Erfassung: Gründung einer
  Kapitalgesellschaft beziehungsweise Genossenschaft“
  ([Formularliste][elsterformulare]); dafür genügt eine Registrierung mit
  E-Mail-Adresse. Firma, Sitz, Geschäftsjahr, Bankverbindung und
  Eröffnungsbilanz kommen aus Buchfink
  ([gruendung/eroeffnungsbilanz.md](../gruendung/eroeffnungsbilanz.md)). Die
  Steuernummer aus dem Antwortschreiben schlägst du mit
  `stammdaten_vorschlagen` vor.
- **Nachweis:** Transferticket.
- **Bei Versäumnis:** Zwangsgeld bis 25.000 € je Festsetzung
  ([§§ 328, 329 AO][ao328]). Ohne Steuernummer oder USt-IdNr. fehlt jeder
  Ausgangsrechnung eine Pflichtangabe ([§ 14 Abs. 4 Satz 1 Nr. 2 UStG][ustg14]).
- **Vertiefung:** [gruendung/steuerliche-erfassung.md](../gruendung/steuerliche-erfassung.md)

<a id="gewerbeanmeldung"></a>
## Gewerbeanzeige

- **Anlässe:** Beginn, Verlegung, Wechsel oder Ausdehnung des Gegenstands,
  Namensänderung, Aufgabe ([§ 14 Abs. 1 Satz 2 GewO][gewo14]). `fristen` führt
  nur den Beginn; an die übrigen erinnerst du.
- **Wertungsfrage:** rein freiberuflicher Unternehmensgegenstand. Der Mensch
  klärt mit dem Gewerbeamt, ob eine Anzeige nötig ist, und setzt die Pflicht
  sonst mit Grund auf entfallen.
- **Weg:** Vordruck GewA 1 ([§ 1 Nr. 1][gewanzv1], [§ 2 GewAnzV][gewanzv2])
  über den Online-Dienst der Gemeinde oder des Landes, sonst beim Gewerbeamt.
  Binnen einer Woche nach Beginn erstattet, gilt sie auch als Meldung an die
  Berufsgenossenschaft ([§ 192 Abs. 1 Satz 2 SGB VII][sgb7_192]).
- **Nachweis:** Empfangsbescheinigung, binnen drei Tagen
  ([§ 15 Abs. 1 GewO][gewo15]); du legst sie ab.
- **Bei Versäumnis:** Geldbuße bis 1.000 €, auch für die übrigen Anlässe
  ([§ 146 Abs. 2 Nr. 2 Buchst. b, Abs. 3 GewO][gewo146]).
- **Vertiefung:** [gruendung/gruendungsweg.md](../gruendung/gruendungsweg.md),
  Abschnitt 4

<a id="eroeffnungsbilanz"></a>
## Aufstellung der Eröffnungsbilanz

- **Im Kalender** nur bei einer Gründung in Buchfink, nicht für einen
  Umsteiger.
- **Weg:**
  1. Du prüfst die Bilanz zum Beurkundungstag (Gründungsdatum der
     Unternehmensdaten) nach
     [gruendung/eroeffnungsbilanz.md](../gruendung/eroeffnungsbilanz.md) und
     meldest `pflicht_vorbereitet`.
  2. Der Mensch speichert sie in der App als PDF (Unternehmen, Stammdaten,
     Block „Gesellschafter & Kapital“, „Als PDF speichern“ an der Pflicht).
     Die App gibt es erst aus, wenn die Bilanz aufgeht, das Kapital der
     Gründung entspricht und die Geschäftsführung eingetragen ist; einen
     Befund klärst du mit dem Menschen.
  3. Die Geschäftsführer unterzeichnen mit Datum
     ([§ 245 HGB][hgb245] über [§ 242 Abs. 1 Satz 2 HGB][hgb242]). Die
     unterzeichnete Fassung legst du auf Anweisung aus dem Eingang ab; das
     Original bewahrt der Mensch auf ([§ 257 Abs. 3 HGB][hgb257]).
  4. Der Mensch bestätigt die Pflicht an derselben Stelle in der App.
- **Nachweis:** die Bestätigung; die unterzeichnete Fassung liegt im Archiv.
- **Bei Versäumnis:** Strafbarkeit wie beim [bilanzstichtag](#bilanzstichtag);
  die Eröffnungsbilanz ist eine solche Bilanz
  ([§ 242 Abs. 1 Satz 1 HGB][hgb242]). Ohne sie fehlt auch die
  [ebilanz-eroeffnung](#ebilanz-eroeffnung).
- **Vertiefung:** [gruendung/eroeffnungsbilanz.md](../gruendung/eroeffnungsbilanz.md)

<a id="ebilanz-eroeffnung"></a>
## E-Bilanz der Eröffnungsbilanz

- **Im Kalender** wie die [eroeffnungsbilanz](#eroeffnungsbilanz); früher zu
  übermitteln ist möglich.
- **Weg:** Geht die Eröffnungsbilanz auf, schreibt `ebilanz_exportieren`
  (`jahr`: Geschäftsjahr der Gründung, `bilanzart: eroeffnungsbilanz`) die
  geprüfte XBRL-Datei ohne GuV in den Exportordner. Der Mensch sendet sie wie
  bei der [ebilanz](#ebilanz) und trägt das Transferticket ein (Abschlüsse,
  Erklärungen, Zeile „E-Bilanz der Eröffnungsbilanz“).
- **Nachweis:** Transferticket.
- **Bei Versäumnis:** Zwangsgeld bis 25.000 € je Festsetzung
  ([§§ 328, 329 AO][ao328]).
- **Vertiefung:** [steuererklaerungen/ebilanz.md](../steuererklaerungen/ebilanz.md),
  Abschnitt 6

<a id="kapitalertragsteuer"></a>
## Kapitalertragsteuer auf eine Ausschüttung

- **Frist:** Einbehalt, Anmeldung und Abführung bei Zufluss
  ([§ 44 Abs. 1 Satz 5 Halbsatz 2 EStG][estg44], [§ 45a Abs. 1 EStG][estg45a]):
  am Auszahlungstag aus dem Beschluss, ohne ihn am Tag nach der
  Beschlussfassung, bei einer Fälligkeit aus Gesetz oder zugelassener
  Satzungsbestimmung am Fälligkeitstag
  ([§ 44 Abs. 2 Satz 1, 2 EStG][estg44]). `fristen` rechnet immer mit dem Tag
  nach dem Beschluss: Frag, ob Beschluss oder Gesellschaftsvertrag einen Tag
  bestimmen, und nenne dem Menschen das richtige Datum. Schlag als
  Auszahlungstag einen Werktag vor.
- **Kirchensteuer:** Anlass- und Regelabfrage beim BZSt (1. September bis
  31. Oktober, [§ 51a Abs. 2c Satz 1 Nr. 3 EStG][estg51a]) führt `fristen`
  nicht ([gesellschafter/ausschuettung.md](../gesellschafter/ausschuettung.md)).
- **Weg:**
  1. Beträge je Gesellschafter aus `hinweise` vorlegen: Ausschüttung,
     Kapitalertragsteuer, Solidaritätszuschlag, gegebenenfalls Kirchensteuer
     (Konten 3519 und 3760).
  2. Mein ELSTER, unter „Kapitalertragsteuer“ das Formular
     „Kapitalertragsteuer-Anmeldung nach Einkommensteuergesetz“; auch ohne
     Steuerabzug anmelden, mit Grund ([§ 45a Abs. 1 Satz 2, 3 EStG][estg45a]).
  3. Den abgerundeten Betrag aus `hinweise` am Zuflusstag an das Finanzamt der
     GmbH zahlen.
  4. Auf Verlangen Steuerbescheinigung ausstellen
     ([§ 45a Abs. 2 EStG][estg45a]).
- **Nachweis:** Transferticket.
- **Bei Versäumnis:** Die GmbH haftet für die Steuer
  ([§ 44 Abs. 5 EStG][estg44]), der Geschäftsführer bei grober Fahrlässigkeit
  ([§ 69 AO][ao69]); Säumnis- und Verspätungszuschlag
  ([§ 240 AO][ao240], [§ 152 Abs. 1 AO][ao152]); Geldbuße bis 25.000 €
  ([§ 380 AO][ao380]).
- **Vertiefung:** [gesellschafter/ausschuettung.md](../gesellschafter/ausschuettung.md),
  [gesellschafter/steuerbescheinigung.md](../gesellschafter/steuerbescheinigung.md)

[ao1]: https://www.gesetze-im-internet.de/ao_1977/__1.html
[ao69]: https://www.gesetze-im-internet.de/ao_1977/__69.html
[ao108]: https://www.gesetze-im-internet.de/ao_1977/__108.html
[ao109]: https://www.gesetze-im-internet.de/ao_1977/__109.html
[ao149]: https://www.gesetze-im-internet.de/ao_1977/__149.html
[ao150]: https://www.gesetze-im-internet.de/ao_1977/__150.html
[ao152]: https://www.gesetze-im-internet.de/ao_1977/__152.html
[ao162]: https://www.gesetze-im-internet.de/ao_1977/__162.html
[ao224]: https://www.gesetze-im-internet.de/ao_1977/__224.html
[ao240]: https://www.gesetze-im-internet.de/ao_1977/__240.html
[ao328]: https://www.gesetze-im-internet.de/ao_1977/__328.html
[ao380]: https://www.gesetze-im-internet.de/ao_1977/__380.html
[aeao108]: https://www.steuerschroeder.de/steuergesetze/aeao/108
[bgb121]: https://www.gesetze-im-internet.de/bgb/__121.html
[estdv60]: https://www.gesetze-im-internet.de/estdv_1955/__60.html
[estg5b]: https://www.gesetze-im-internet.de/estg/__5b.html
[estg41a]: https://www.gesetze-im-internet.de/estg/__41a.html
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
[gmbhg11]: https://www.gesetze-im-internet.de/gmbhg/__11.html
[gmbhg16]: https://www.gesetze-im-internet.de/gmbhg/__16.html
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
[hgb240]: https://www.gesetze-im-internet.de/hgb/__240.html
[hgb242]: https://www.gesetze-im-internet.de/hgb/__242.html
[hgb245]: https://www.gesetze-im-internet.de/hgb/__245.html
[hgb257]: https://www.gesetze-im-internet.de/hgb/__257.html
[hgb325]: https://www.gesetze-im-internet.de/hgb/__325.html
[hgb335]: https://www.gesetze-im-internet.de/hgb/__335.html
[kstg27]: https://www.gesetze-im-internet.de/kstg_1977/__27.html
[kstg31]: https://www.gesetze-im-internet.de/kstg_1977/__31.html
[sgb7_192]: https://www.gesetze-im-internet.de/sgb_7/__192.html
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
[elsterformulare]: https://www.elster.de/eportal/formulare-leistungen/alleformulare
[bzstzm]: https://www.bzst.de/DE/Unternehmen/Umsatzsteuer/ZusammenfassendeMeldung/ElektronischeAbgabe/elektronischeabgabe_zm_node.html
[bzstzmfrist]: https://www.bzst.de/DE/Unternehmen/Umsatzsteuer/ZusammenfassendeMeldung/Fristen/fristen_node.html
[pp]: https://publikations-plattform.de
[ppveroeff]: https://publikations-plattform.de/order/de/wissenswertes/sogehts/javeroeffentlichung
[ppident]: https://publikations-plattform.de/order/de/wissenswertes/sogehts/registrierenidentifizeren
[ppdaten]: https://publikations-plattform.de/order/de/wissenswertes/sogehts/meinedaten
[trhilfe]: https://www.transparenzregister.de/hilfe-center
[trhilfeapi]: https://www.transparenzregister.de/connector/api/help-center
