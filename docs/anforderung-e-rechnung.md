# Buchfink – E-Rechnung (Empfang und Ausstellung)

Status: Empfang umgesetzt; Validierung siehe Abschnitt 8
Letzte Aktualisierung: 2026-08-22
Voraussetzung: [Beleg- & Buchungsflow](anforderung-beleg-buchungsflow.md)

> Dieses Dokument ist vollständig gegen Primärquellen geprüft (Stand 22.08.2026):
> UStG über gesetze-im-internet.de, das BMF-Schreiben vom 15.10.2025 zur
> obligatorischen E-Rechnung und die GoBD in der Fassung vom 14.07.2025. Die
> Fundstellen stehen in [Abschnitt 9](#9-quellen).

## 1. Warum dieses Dokument anders ist als die anderen

Anlagen, Anzahlungen, Abgrenzung und DATEV-Export beschreiben Funktionen, die
Buchfink noch nicht kann und die niemand vermisst, solange sie fehlen. Die
E-Rechnung ist ein anderer Fall: **die Pflicht gilt bereits, und die für Buchfink
kritische Hälfte davon hat nie eine Übergangsfrist gehabt.**

Seit dem 01.01.2025 muss zwischen inländischen Unternehmern über eine Leistung an
einen anderen Unternehmer für dessen Unternehmen grundsätzlich per E-Rechnung
abgerechnet werden (§ 14 Abs. 2 Satz 2 Nr. 1 UStG). Für das **Ausstellen** gibt es
Übergangsregelungen bis Ende 2026 bzw. Ende 2027 (§ 27 Abs. 38 UStG). Für das
**Empfangen** gibt es keine.

Der Grund steht in § 14 Abs. 1 Satz 5 UStG: die Übermittlung einer elektronischen
Rechnung bedarf der Zustimmung des Empfängers nur, „soweit keine Verpflichtung
nach Absatz 2 Satz 2 Nummer 1 besteht". Wo die Pflicht greift, darf der Aussteller
also einfach senden. Die Finanzverwaltung zieht daraus die ausdrückliche
Konsequenz:

> „Inländische Unternehmer müssen die technischen Voraussetzungen zum Empfang
> einer E-Rechnung schaffen. Dies gilt auch, wenn der Rechnungsempfänger der
> Sonderregelung nach § 19 UStG unterliegt. […] Ist ein Unternehmer technisch
> nicht in der Lage, eine E-Rechnung empfangen zu können, bzw. verweigert er die
> Annahme, hat er kein Anrecht auf eine alternative Ausstellung einer sonstigen
> Rechnung durch den Rechnungsaussteller."
>
> — UStAE Abschnitt 14.1 Abs. 5 i. d. F. des BMF-Schreibens vom 15.10.2025

**Buchfink konnte lange keine E-Rechnung empfangen.** Ein Beleg war ein Dateipfad
am Journaleintrag; ein strukturierter Rechnungsdatensatz existierte im Modell
nicht. Das war nicht eine Lücke unter mehreren, sondern die einzige, die eine
laufende Rechtspflicht des Nutzers unerfüllt ließ. Sie ist geschlossen: der Beleg
trägt den strukturierten Teil als eigene Datei, das XML wird aus dem Hybrid-PDF
gezogen, und der Buchungsvorschlag entsteht daraus.

## 2. Wer betroffen ist und wer nicht

Die Pflicht zur **Ausstellung** einer E-Rechnung setzt drei Dinge zusammen
voraus (§ 14 Abs. 2 Satz 2 Nr. 1 UStG):

| Voraussetzung | Anmerkung |
|---|---|
| Leistung an einen **anderen Unternehmer für dessen Unternehmen** | B2C ist nie betroffen |
| **Beide im Inland ansässig** (Sitz, Geschäftsleitung oder beteiligte Betriebsstätte, § 14 Abs. 2 Satz 3 UStG) | ist einer im Ausland ansässig, entfällt die Pflicht |
| Der Umsatz ist **nicht nach § 4 Nr. 8 bis 29 UStG steuerfrei** | Vermietung, Heilbehandlung, Finanzdienstleistungen etc. fallen heraus |

Drei Punkte daran werden regelmäßig falsch erinnert:

- **§ 13b-Umsätze sind erfasst.** Die Pflicht besteht auch, wenn der
  Leistungsempfänger nach § 13b Abs. 5 UStG die Steuer schuldet (UStAE 14.1
  Abs. 4 Satz 8). Buchfinks Reverse-Charge-Fall ist also kein Ausweg.
- **Steuerfreie Umsätze nach § 4 Nr. 1 bis 7 sind erfasst** – die
  innergemeinschaftliche Lieferung an die inländische Betriebsstätte eines anderen
  inländischen Unternehmers zum Beispiel (UStAE 14.1 Abs. 4 Satz 9). Nur § 4
  Nr. 8 bis 29 nimmt die Pflicht heraus.
- **Kleinunternehmer sind von der Ausstellung befreit, vom Empfang nicht.**
  Rechnungen eines Kleinunternehmers nach § 34a UStDV dürfen immer als sonstige
  Rechnung ausgestellt werden; die Empfangspflicht gilt ausdrücklich „auch, wenn
  der Rechnungsempfänger der Sonderregelung nach § 19 UStG unterliegt".

Immer als sonstige Rechnung zulässig bleiben außerdem Kleinbetragsrechnungen
(§ 33 UStDV) und Fahrausweise (§ 34 UStDV).

## 3. Die Übergangsfristen – nur für das Ausstellen

§ 27 Abs. 38 UStG kennt drei Nummern:

| Nr. | Was erlaubt bleibt | Bis wann | Bedingung |
|---|---|---|---|
| 1 | Papier oder – mit Zustimmung des Empfängers – ein nicht konformes elektronisches Format | **31.12.2026** | für Umsätze nach dem 31.12.2024 und vor dem 01.01.2027 |
| 2 | dasselbe | **31.12.2027** | für Umsätze in 2027, wenn der **Gesamtumsatz des Ausstellers im Vorjahr höchstens 800.000 €** betrug |
| 3 | abweichende EDI-Formate nach der Empfehlung 94/820/EG | **31.12.2027** | – |

Ab 2028 gibt es keine Übergangsregelung mehr.

Für Buchfink heißt das: die **Ausgangsseite** hat noch Zeit – bei einem
Vorjahresumsatz über 800.000 € allerdings nur bis Ende 2026. Die **Eingangsseite**
hat keine. Diese Asymmetrie bestimmt die Reihenfolge der Umsetzung.

## 4. Was Buchfink heute hat und was fehlt

| Baustein | Stand |
|---|---|
| ZUGFeRD-XML-Erzeugung für Ausgangsrechnungen | vorhanden (`internal/invoice/zugferd.go`), Profil `urn:cen.eu:en16931:2017` |
| **Hybrides PDF/A-3b mit eingebettetem XML** | vorhanden (`internal/invoice/render.go`), Typst als WebAssembly im Prozess |
| Mehrdateiliger Beleg mit Rollen (PDF + XML) | vorhanden (`internal/domain/receipt.go`), Beleg-Hash über die geordnete Dateiliste |
| **Einlesen einer empfangenen E-Rechnung** | vorhanden: XML aus dem Hybrid-PDF (`pdfattach.go`), CII-Parser (`cii.go`), Buchungsvorschlag (`einvoice_service.go`) |
| **Ablehnung der Profile MINIMUM und BASIC WL** | vorhanden |
| **Validierung gegen EN 16931** | Teilprüfung in Go (`internal/invoice/en16931.go`), 170 von 223 Regeln, Regelliste und Version am Beleg |
| **XRechnung (reines XML ohne PDF)** | ablegbar und auslesbar; buchbar erst mit erzeugter Darstellung |
| Unveränderbare Aufbewahrung über die Hash-Chain | vorhanden, über den Beleg-Hash |

Das gewählte Profil ist bereits das richtige: zulässig sind ZUGFeRD ab Version
2.0.1 **außer den Profilen MINIMUM und BASIC-WL** (UStAE 14.1 Abs. 14 Satz 4).
MINIMUM und BASIC-WL enthalten keine vollständige Rechnung und sind deshalb keine
E-Rechnung im Sinne des Gesetzes. Ein Test, der genau das festhält, wäre billig
und würde einen späteren Profilwechsel absichern.

## 5. Der Vorsteuerabzug hängt am strukturierten Teil

Das ist der Punkt mit den härtesten Folgen für die Architektur, und er ist neu:

> „Ein Vorsteuerabzug ist auch in diesen Fällen nur aus dem strukturierten
> Rechnungsteil möglich."
>
> — UStAE Abschnitt 14c.1 Abs. 4a Satz 4 i. d. F. vom 15.10.2025

Und schärfer noch für den Fall, dass jemand trotz Pflicht auf Papier abrechnet:

> „Sofern für einen Umsatz nach § 14 Abs. 2 Satz 2 in Verbindung mit § 27 Abs. 38
> UStG eine Verpflichtung zur Ausstellung einer E-Rechnung besteht, erfüllt nur
> eine solche dem Grunde nach die Anforderungen der §§ 14, 14a UStG. Wird in einem
> solchen Fall stattdessen eine sonstige Rechnung ausgestellt, handelt es sich
> nicht um eine ordnungsmäßige Rechnung […], so dass diese Rechnung dem Grunde
> nach nicht zum Vorsteuerabzug berechtigt."
>
> — UStAE Abschnitt 15.2a Abs. 1 Sätze 3 und 4 i. d. F. vom 15.10.2025

Daraus folgen zwei Anforderungen, die sich nicht nachrüsten lassen, ohne den
Belegfluss anzufassen:

1. **Die Buchung muss aus dem XML entstehen, nicht aus dem PDF.** Ein
   OCR-Vorschlag aus dem Bildteil ist ab dem Moment, in dem eine E-Rechnung
   vorliegt, die schlechtere Quelle – und rechtlich die falsche. Wo ein
   strukturierter Teil da ist, hat er Vorrang.
2. **Weicht der Bildteil inhaltlich ab, ist das kein Darstellungsproblem,
   sondern potenziell eine zweite Rechnung** mit § 14c-Folgen (UStAE 14c.1
   Abs. 4a Sätze 2 und 3). Geringfügige technische Abweichungen, verkürzte
   Leistungsbeschreibungen und Rundungsdifferenzen bleiben unbeanstandet.
   Buchfink sollte die Abweichung erkennen und anzeigen, aber sie **nicht selbst
   bewerten** – das ist eine Rechtsfrage, keine Rechenfrage.

## 6. Aufbewahrung – hier wird es für Buchfink einfacher, nicht schwerer

Die GoBD in der Fassung vom 14.07.2025 sind gerade wegen der E-Rechnung geändert
worden. Die Änderungen gehen in eine für Buchfink günstige Richtung:

| Randziffer | Inhalt |
|---|---|
| **119**, **131** | Bei E-Rechnungen genügt die Aufbewahrung des **strukturierten Teils**. Der menschenlesbare Teil ist nur aufzubewahren, wenn er zusätzliche oder abweichende steuerlich bedeutsame Informationen enthält (z. B. Buchungsvermerke, qualifizierte elektronische Signaturen). |
| **125**, Beispiel 10 | Entscheidend ist, dass der strukturierte Datenteil vorhanden ist; er **darf nicht durch eine Formatumwandlung** (z. B. in TIFF) gelöscht werden. |
| **118** | Bei empfangenen strukturierten Datensätzen bedarf es **abweichend von § 147 Abs. 2 Nr. 1 AO keiner bildlichen, sondern nur einer inhaltlichen Übereinstimmung.** |
| **76** Abs. 2 | Bei Einsatz eines Fakturierungsprogramms muss **keine bildhafte Kopie der Ausgangsrechnung** aufbewahrt werden, wenn jederzeit ein inhaltlich identisches Mehrstück erstellt werden kann. |
| **127** | Sonstige strukturierte Dateien – ausdrücklich E-Rechnungen – sind maschinell auswertbar vorzuhalten. |

Rz. 76 Abs. 2 ist für Buchfink direkt relevant: das PDF der eigenen
Ausgangsrechnung muss nicht dauerhaft gespeichert werden, solange sich aus den
Daten jederzeit ein identisches Mehrstück erzeugen lässt. Ob man diese
Erleichterung nutzen *will*, ist eine andere Frage – ein archiviertes PDF ist für
den Nutzer greifbarer als eine Rendering-Zusage.

Die Aufbewahrungsfrist beträgt acht Jahre (§ 14b Abs. 1 UStG), beginnend mit dem
Schluss des Kalenderjahres, in dem die Rechnung ausgestellt wurde.

**Die Kopplung an die Hash-Chain ist damit klar:** was gesichert werden muss, ist
der strukturierte Datensatz in seiner empfangenen Form. Er gehört in denselben
Unveränderbarkeitsnachweis wie die Buchung – und zwar über seinen Hash, nicht über
seinen Pfad, konsistent zu der Entscheidung, `DocumentPath` bewusst aus der
Kanonisierung herauszulassen.

Genau das leistet der **Beleg-Hash** aus Abschnitt 15 des Hauptkonzepts: er läuft
über die geordnete Liste aller Belegdateien, deckt damit Original *und*
strukturierten Teil ab und wandert als ein Wert in die Buchung. Ein
nachträglich ausgetauschtes XML fällt auf, ohne dass die Buchung n Hashes tragen
muss.

## 7. Was daraus für das Datenmodell folgt

**Das Belegmodell ist nicht Teil dieses Dokuments, sondern des Hauptkonzepts.**
E-Rechnung ist kein Anbau neben dem Belegflow, sondern der Grund, warum ein Beleg
dort aus mehreren Dateien besteht: Rollen `original`, `structured`, `rendering`
und `attachment`, Hash je Datei, Beleg-Hash über die geordnete Liste, mit der
Buchung versiegelt. Siehe
[anforderung-beleg-buchungsflow.md](anforderung-beleg-buchungsflow.md),
Abschnitt 15.

Was dieses Dokument darüber hinaus verlangt, sind zwei Angaben am Beleg, die nur
im E-Rechnungsfall entstehen:

- das **Validierungsergebnis** gegen EN 16931 – mit Zeitpunkt, Regelversion und
  dem erkannten Format samt Profil, damit ein späterer Prüflauf reproduzierbar
  bleibt,
- das Ergebnis des **Abgleichs zwischen Bild- und strukturiertem Teil** bei
  Hybridbelegen (siehe Abschnitt 5).

Der Buchungsvorschlag entsteht aus der Datei mit der Rolle `structured`:
Lieferant über die USt-IdNr. oder Steuernummer, Beträge und Steuersätze aus den
XML-Feldern, Leistungsdatum aus dem entsprechenden Feld der Norm. Das ist ein
deutlich besserer Ausgangspunkt als OCR – die Daten sind exakt statt geraten.

**Der Steuerkategoriecode wird dabei gedreht.** Er steht aus Sicht des Ausstellers
im Dokument: „K" heißt beim Lieferanten steuerfreie innergemeinschaftliche
Lieferung und beim Empfänger innergemeinschaftlicher Erwerb, „AE" heißt
Steuerschuld beim Leistungsempfänger – also bei uns. Wer den Code eins zu eins
übernimmt, bucht den halben Vorgang und meldet die Erwerbsteuer nicht.

Was Buchfink nicht ehrlich abbilden kann, wird benannt statt geraten: „G" steht
für eine Ausfuhr des Lieferanten, beim Empfänger also eine Einfuhr mit
Einfuhrumsatzsteuer aus dem Zollbescheid – die steht nicht in dieser Rechnung.
Ebenso bleibt die **Buchungsgruppe offen**: welches Aufwandskonto zutrifft, sagt
keine Rechnung.

Für die **Ausgangsseite** gilt dasselbe Modell in die andere Richtung: eine
selbst erzeugte ZUGFeRD-Rechnung ist ein Beleg mit dem hybriden PDF als
`original`, eine XRechnung einer mit dem XML als `original` und einer erzeugten
Darstellung als `rendering`. Damit braucht der Ausgangsfall keine eigene Ablage.

**Der Empfangsweg selbst gehört ausdrücklich nicht in v1.** Ein eigenes
Postfach ist nicht erforderlich (UStAE 14.1 Abs. 5 Satz 3); es genügt, wenn der
Nutzer die empfangene Datei ablegt. Was Buchfink braucht, ist das **Einlesen und
Verstehen** einer E-Rechnung, nicht ein Mailserver.

## 8. Offene Entscheidungen

- **Reihenfolge:** Empfang zuerst, Ausstellung danach – das folgt aus den Fristen.
  Zu entscheiden ist, ob der Empfang vor die übrigen offenen Punkte gezogen wird.
  *Vorschlag: ja, weil er als einziger eine bereits laufende Pflicht betrifft.*
- **Validierung:** entschieden und umgesetzt als eigener Regelprüfer in Go
  (`internal/invoice/en16931.go`). Eine native Go-Bibliothek gibt es nicht, und
  der Referenzweg – der KoSIT-Validator mit Schematron über Saxon-XSLT – setzt
  eine Java-Laufzeit voraus, die eine lokale Go-Desktop-App nicht mitbringen soll.

  Geprüft wird eine **belegte Teilmenge, und zwar eine gemessene**: 170 der 223
  Geschäftsregeln, die das Validierungsartefakt der EU-Kommission für CII
  definiert – rund drei Viertel. Die Zahl steht nicht in diesem Dokument, weil
  sie jemand geschätzt hätte: `internal/invoice/en16931_rules.go` führt die
  Regelkennungen der Norm als Inventar, und `TestCoverageIsMeasured` rechnet die
  Abdeckung bei jedem Testlauf aus. Die Liste der geprüften Regeln ist über
  `GetEInvoiceRules()` abrufbar und Teil jedes Ergebnisses.

  Abgedeckt sind die Pflichtangaben (BR-01 bis BR-16), die Positionsregeln
  (BR-21 bis BR-26), Nachlässe und Zuschläge (BR-31 bis BR-44), die
  Steueraufschlüsselung (BR-45 bis BR-48), die Rechenregeln (BR-CO-04, -09 bis
  -19, -26), die Nachkommastellen (BR-DEC, bis auf eine Regel vollständig), die
  Codelisten für Währung, Land und Steuerkategorie (BR-CL-03, -04, -05, -14,
  -18) und **alle neun Steuerkategorie-Familien vollständig** – BR-S, BR-Z,
  BR-E, BR-AE, BR-IC, BR-G, BR-O, BR-AF, BR-AG mit je zehn Regeln, dazu
  BR-IC-11/-12 und BR-O-11 bis -14.

  Ungeprüft bleiben Regeln zu Feldern, die Buchfink nicht liest: Zahlungsmittel,
  Anlagen, Rechnungsbezüge, Kennungsschemata nach ISO 6523 und die zugehörigen
  Codelisten. `EN16931UncheckedRules()` nennt sie einzeln, damit sich für eine
  bestimmte Regel nachsehen lässt, ob sie geprüft wurde – statt es aus einer
  Prozentzahl zu erraten.

  **Wie die Teilmenge belastbar wird.** Testdateien machen eine Prüfung nicht
  vollständig – sie decken ab, was sie enthalten, und über beliebige Eingaben
  sagen sie nichts. Sie sind trotzdem das schärfste Werkzeug, das hier zur
  Verfügung steht, weil die 15 offiziellen CII-Beispiele des Artefakts *bekannt
  gültig* sind: jeder Fund darin ist ein Fehler im Prüfer, nicht in der
  Rechnung. Sechs Fehler kamen so heraus, darunter zwei, die korrekte Rechnungen
  abgewiesen hätten (BR-08/BR-10 verlangten Straße und Ort, wo die Norm nur nach
  der Adressgruppe fragt; BR-CO-17 verglich ohne die zulässige Toleranz von
  einer Währungseinheit), einer im eigenen Erzeuger (das Bestimmungsland BT-80
  fehlte bei innergemeinschaftlicher Lieferung) und einer, den nur eine echte
  Datei zeigt: eine dänische Rechnung führt den Steuerbetrag zweimal, in
  Rechnungs- und in Abrechnungswährung, und Buchfink las den falschen.

  Zwei Fehlerarten finden Beispieldateien allerdings **nicht**, weil ein gültiges
  Dokument keine Regel auslöst: eine Regel, die unter falscher Kennung gemeldet
  wird, und eine Regel, die in der Liste steht, aber nichts prüft. Beides ist
  vorgekommen – die innergemeinschaftlichen Regeln liefen unter dem erfundenen
  Namen BR-K statt BR-IC, drei BR-CL-Kennungen zeigten auf andere Felder als
  gemeint, und BR-AG-05 stand in der Liste, ohne dass die Kategorie „IPSI"
  überhaupt eine Satzbedingung hinterlegt hatte. Dagegen laufen zwei eigene
  Tests: einer vergleicht jede zugesagte Kennung mit dem Regelinventar der Norm,
  der andere liest den Quelltext der Prüfung und meldet jede Regel, die zugesagt,
  aber nirgends gemeldet wird – und umgekehrt.

  Die Beispieldateien liegen **nicht** im Repository: das Artefakt steht unter
  EUPL-1.2, Buchfink unter MIT. Übernommen wurden nur die Regelkennungen, das
  sind Tatsachen über die Norm. `task test:en16931` holt die Dateien bei Bedarf
  und lässt den Abgleich laufen; ohne sie überspringt sich der Test.

  Der Prüfumfang kennt **keinen Wert für „vollständig geprüft"** – nur
  `partial`. Das ist Absicht: Vollständigkeit zu behaupten wäre der einzige
  Fehler, der schlimmer wäre als die Lücke selbst, weil sich jemand darauf
  verließe. Ergebnis, Zeitpunkt, Regelwerk und dessen Version stehen am Beleg,
  damit ein späterer Prüflauf vergleichbar bleibt; die Prüfung fasst keine Datei
  an und lässt den Beleg-Hash deshalb unberührt.

  Buchfink hält auch die **eigenen** Rechnungen gegen dieses Regelwerk: eine
  Ausgangsrechnung, die es verletzt, wird gar nicht erst erzeugt. Eine Rechnung
  ohne vollständige Empfängeranschrift ist nach § 14 Abs. 4 Nr. 1 UStG keine
  ordnungsmäßige Rechnung – der Empfänger verlöre den Vorsteuerabzug und merkte
  es erst bei der Betriebsprüfung.
- **Umgang mit abweichendem Bildteil:** anzeigen und blockieren, oder anzeigen und
  weiterbuchen lassen? *Vorschlag: anzeigen, Buchung aus dem XML, Hinweis auf die
  mögliche § 14c-Relevanz – keine automatische Bewertung.*
- **Papierrechnung im Pflichtfall:** entschieden und umgesetzt
  (`internal/service/einvoice_notice.go`). Buchfink weist in der
  Buchungsvorschau darauf hin, wenn ein inländischer, unternehmerischer
  Lieferant ohne Kleinunternehmerstatus über der Kleinbetragsgrenze des
  § 33 UStDV eine sonstige Rechnung stellt und der Beleg keinen strukturierten
  Teil trägt. Der Text wechselt mit dem Belegdatum über die Fristen des
  § 27 Abs. 38 UStG; dazu kommt ein zweiter, der sich an den Lieferanten
  weitergeben lässt. **Blockiert wird nie** – ob aus der sonstigen Rechnung ein
  Vorsteuerabzug folgt, ist eine Rechtsfrage, und den Vorjahresumsatz des
  Ausstellers kennt Buchfink nicht.
- **Aufbewahrung des eigenen PDF:** Rz. 76 Abs. 2 erlaubt den Verzicht. Nutzen
  oder trotzdem archivieren?
- **Ausstellungsseite:** XRechnung zusätzlich zu ZUGFeRD anbieten? Beide sind
  zulässig; XRechnung ist bei öffentlichen Auftraggebern verbreiteter.

## 9. Quellen

Stand der Prüfung: 22.08.2026.

### Gesetz

| Aussage | Fundstelle | Link |
|---|---|---|
| Definition E-Rechnung (strukturiertes Format, elektronische Verarbeitung) | § 14 Abs. 1 Satz 3 UStG | [ustg_1980/__14.html](https://www.gesetze-im-internet.de/ustg_1980/__14.html) |
| Definition sonstige Rechnung (anderes elektronisches Format oder Papier) | § 14 Abs. 1 Satz 4 UStG | dito |
| Zustimmung des Empfängers nur, soweit keine Pflicht nach Abs. 2 Satz 2 Nr. 1 besteht | § 14 Abs. 1 Satz 5 UStG | dito |
| Zulässige Formate: EN 16931 oder vereinbart | § 14 Abs. 1 Satz 6 UStG | dito |
| Ausstellungspflicht B2B im Inland, Frist sechs Monate, Ausnahme § 4 Nr. 8 bis 29 | § 14 Abs. 2 Satz 2 Nr. 1 UStG | dito |
| Ansässigkeitsbegriff | § 14 Abs. 2 Satz 3 UStG | dito |
| Übergangsregelungen (2026 / 2027 / EDI, Grenze 800.000 €) | § 27 Abs. 38 Nrn. 1 bis 3 UStG | [ustg_1980/__27.html](https://www.gesetze-im-internet.de/ustg_1980/__27.html) |
| Aufbewahrungsfrist acht Jahre | § 14b Abs. 1 UStG | [ustg_1980/__14b.html](https://www.gesetze-im-internet.de/ustg_1980/__14b.html) |
| Steuerschuld bei zu hohem Steuerausweis | § 14c Abs. 1 UStG | [ustg_1980/__14c.html](https://www.gesetze-im-internet.de/ustg_1980/__14c.html) |
| Kleinbetragsrechnungen / Fahrausweise / Kleinunternehmerrechnungen | §§ 33, 34, 34a UStDV | [ustdv_1980](https://www.gesetze-im-internet.de/ustdv_1980/) |

### Verwaltungsanweisungen

**BMF-Schreiben vom 15.10.2025**, „Einführung der obligatorischen elektronischen
Rechnung bei Umsätzen zwischen inländischen Unternehmern ab dem 1. Januar 2025",
Änderung des Umsatzsteuer-Anwendungserlasses (33 Seiten). Es ersetzt das
BMF-Schreiben vom 15.10.2024 (BStBl I S. 1320), dessen Rn. 62 bis 65 zu den
Übergangsregelungen weiter zu beachten sind.

<https://www.bundesfinanzministerium.de/Content/DE/Downloads/BMF_Schreiben/Steuerarten/Umsatzsteuer/Umsatzsteuer-Anwendungserlass/2025-10-15-einfuehrung-obligatorische-e-rechnung.html>

Die in diesem Dokument zitierten Stellen des neu gefassten UStAE:

| Stelle | Inhalt |
|---|---|
| Abschnitt 14.1 Abs. 4 Sätze 4 bis 10 | keine Zustimmung nötig; Übermittlungswege; § 13b und § 4 Nr. 1 bis 7 erfasst |
| Abschnitt 14.1 Abs. 5 | **Empfangspflicht**, auch für Kleinunternehmer; kein eigenes Postfach nötig; kein Anspruch auf eine sonstige Rechnung |
| Abschnitt 14.1 Abs. 6 Sätze 3 bis 5 | Auslandsbezug; §§ 33, 34, 34a UStDV |
| Abschnitt 14.1 Abs. 13 | XRechnung als Umsetzung der EN 16931 |
| Abschnitt 14.1 Abs. 14 | hybride Formate; ZUGFeRD ab 2.0.1 außer MINIMUM und BASIC-WL |
| Abschnitt 14.1 Abs. 15 | vereinbarte Formate, Interoperabilität, Informationsverlust |
| Abschnitt 14c.1 Abs. 4a | Bildteil als inhaltlich identisches Mehrstück; abweichender Bildteil und § 14c; **Vorsteuerabzug nur aus dem strukturierten Teil** |
| Abschnitt 15.2a Abs. 1 Sätze 3 und 4 | sonstige Rechnung im Pflichtfall berechtigt dem Grunde nach nicht zum Vorsteuerabzug |

Die Abschnittsnummern beziehen sich auf den UStAE in der durch das
BMF-Schreiben vom 15.10.2025 geänderten Fassung. Achtung bei der Nachprüfung: die
E-Rechnungs-Regelungen stehen in **Abschnitt 14.1** (Rechnung), nicht in
Abschnitt 14.4 (Echtheit und Unversehrtheit) – eine Verwechslung, die naheliegt,
weil Abschnitt 14.4 ebenfalls geändert wurde.

**GoBD in der Fassung vom 14.07.2025** (BMF, GZ IV D 2 - S 0316/00128/005/088,
BStBl I S. 1502), 2. Änderung, ausdrücklich begründet mit der Einführung der
obligatorischen E-Rechnung. Sie ändert die Randziffern 76, 118, 119, 121, 125,
127, 131, 133, 166 und 175 und ist mit Wirkung vom 14.07.2025 anzuwenden (neue
Rz. 185). Ausgangsfassung: BMF-Schreiben vom 28.11.2019 (BStBl I S. 1269),
geändert am 11.03.2024 (BStBl I S. 374).

<https://www.bundesfinanzministerium.de/Content/DE/Downloads/BMF_Schreiben/Weitere_Steuerthemen/Abgabenordnung/2025-07-14-GoBD-2-aenderung.html>

### Nicht geprüft

Die technischen Spezifikationen selbst – EN 16931, XRechnung (KoSIT) und ZUGFeRD
(FeRD) – sind für dieses Konzept nicht ausgewertet worden. Vor der Umsetzung des
Einlesens sind sie die maßgebliche Quelle für Feldbezeichner; die oben genannte
`BT-72` stammt aus dem allgemeinen Sprachgebrauch der Norm und ist **nicht
verifiziert**.
