# Buchfink – E-Rechnung (Empfang und Ausstellung)

Gesetzliche Grundlage: [Anforderungskatalog](anforderungskatalog.md), RECH-05,
RECH-06, RECH-07, RECH-10, UST-07, ARC-01, ARC-07

Status: Empfang umgesetzt; Validierung siehe Abschnitt 8
Letzte Aktualisierung: 2026-08-22
Voraussetzung: [Beleg- & Buchungsflow](anforderung-beleg-buchungsflow.md)

> Dieses Dokument ist vollständig gegen Primärquellen geprüft (Stand 22.08.2026):
> UStG über gesetze-im-internet.de, das BMF-Schreiben vom 15.10.2025 zur
> obligatorischen E-Rechnung und die GoBD in der Fassung vom 14.07.2025. Die
> gesetzlichen Anforderungen samt Fundstellen stehen im Anforderungskatalog unter
> RECH-05, RECH-06, RECH-07 und ARC-01/ARC-07.

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

Wen die Pflicht zur **Ausstellung** trifft, steht im Anforderungskatalog unter
RECH-06; welche Rechnungen sonstige Rechnungen bleiben dürfen – Kleinbeträge,
Fahrausweise, Kleinunternehmer –, unter RECH-05.

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

## 3. Die Übergangsfristen – nur für das Ausstellen

Die Übergangsregelungen des § 27 Abs. 38 UStG stehen mit ihren Fristen und der
Umsatzgrenze im Anforderungskatalog unter RECH-06 und im dortigen Terminplan.

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
| **Validierung gegen EN 16931** | vollständig in Go (`internal/einvoice`), 223 von 223 Regeln, Regelliste und Version am Beleg |
| **UBL lesen (Peppol, XRechnung-UBL)** | vorhanden (`internal/einvoice`), dieselbe Prüfung wie für CII |
| **Rechnungstyp auswerten (BT-3)** | im Modul vorhanden; die Verbuchung von Gutschrift und Korrektur steht noch aus |
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

Die Aufbewahrungsfristen und die Anforderungen an die Aufbewahrung von
E-Rechnungen stehen im Anforderungskatalog unter ARC-01 und ARC-07.

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

  Geprüft werden **alle 223 Geschäftsregeln**, die EN 16931 für das semantische
  Modell und seine Codelisten definiert. Vier davon — BR-CO-05 bis BR-CO-08 —
  verlangen, dass ein Grundschlüssel und ein freier Begründungstext dasselbe
  bedeuten; das ist maschinell nicht entscheidbar, und der Referenzprüfer der
  Norm führt sie selbst als `true()`. Buchfink hält es genauso und schreibt
  dazu, warum.

  **Der Aufbau folgt dem der Norm.** Deren Geschäftsregeln stehen nicht an einer
  Syntax, sondern in einem abstrakten Regelsatz über dem semantischen Modell;
  CII und UBL liefern nur die Bindungen. `internal/einvoice` ist genauso gebaut:
  in der Mitte das Modell mit allen Geschäftsbegriffen, daneben zwei Leser und
  ein Schreiber. Dieselbe Rechnung wird damit gleich beurteilt, egal in welcher
  Schreibweise sie ankommt — nachgewiesen über alle offiziellen UBL-Beispiele,
  die durch das Modell nach CII geschrieben dieselbe Beurteilung bekommen.

  **Wie belegt wird, dass die Prüfungen greifen.** Beispielrechnungen können das
  nicht: ein gültiges Dokument löst keine Regel aus, es zeigt also nur, dass
  eine Prüfung nicht zu streng ist. Das Artefakt bringt aber eine zweite
  Sammlung mit — 277 Dateien, eine je Geschäftsregel, jede mit der Erwartung, ob
  die Regel anschlagen muss oder schweigen. **191 der 223 Regeln sind darüber
  bestätigt**, die übrigen 32 bringt die Suite nicht mit und haben eigene Tests.
  `task test:en16931` lässt beides laufen.

  Dabei kamen achtzehn Fehler heraus, elf allein aus der Regelsuite. Vier davon
  sind Stellen, an denen die **beiden Syntaxbindungen der Norm einander
  widersprechen**: BT-8 wird in CII und UBL verschieden verschlüsselt, BR-AF-05
  verlangt in CII einen Satz größer null und in UBL null oder größer, BR-AF-09
  und BR-AG-09 sind in CII `true()` und in UBL vollwertig geprüft, und die
  Codelisten weichen an vier Werten ab. Solche Stellen findet nur, wer beide
  Syntaxen baut.

  Der Prüfumfang kennt trotzdem **keinen Wert für „vollständig geprüft"**. Das
  ist Absicht: XRechnung (BR-DE-*) und Peppol legen eigene Regeln über
  EN 16931, und die prüft das Modul nicht. Dass ein Dokument eine XRechnung zu
  sein behauptet, sagt es — damit ein Aufrufer weiß, dass Bestehen hier nicht
  die ganze Geschichte ist. `RulesChecked()`, `RulesUnchecked()` und `Rule()`
  machen den Umfang abfragbar statt behauptbar; zwei Tests halten ihn ehrlich,
  einer gegen erfundene Regelnamen, einer gegen zugesagte Regeln ohne Prüfung.

  Ergebnis, Zeitpunkt, Regelwerk und dessen Version stehen am Beleg, damit ein
  späterer Prüflauf vergleichbar bleibt; die Prüfung fasst keine Datei an und
  lässt den Beleg-Hash deshalb unberührt. Jeder Befund nennt Regel, Ort und die
  betroffenen Geschäftsbegriffe, sodass eine Oberfläche auf das Feld zeigen kann
  statt eine Regelnummer zum Nachschlagen hinzulegen.

  Buchfink hält auch die **eigenen** Rechnungen gegen dieses Regelwerk: eine
  Ausgangsrechnung, die es verletzt, wird gar nicht erst erzeugt. Eine Rechnung
  ohne vollständige Empfängeranschrift ist nach § 14 Abs. 4 Nr. 1 UStG keine
  ordnungsmäßige Rechnung – der Empfänger verlöre den Vorsteuerabzug und merkte
  es erst bei der Betriebsprüfung.

  Die Beispiel- und Regeldateien liegen unter `internal/einvoice/testdata/`.
  Das Artefakt steht unter EUPL-1.2 — seit dem Lizenzwechsel dieselbe Lizenz wie
  Buchfink. Die Prüfungen laufen damit ohne Netz und ohne Vorbereitung; Herkunft
  und Lizenz stehen im README daneben.
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

## 9. Anmerkungen zu den Fundstellen

Die Normen, das BMF-Schreiben vom 15.10.2025 und die GoBD in der Fassung vom
14.07.2025 stehen mit Fundstellen im Anforderungskatalog unter RECH-05, RECH-06,
RECH-07, RECH-10, UST-07 sowie ARC-01, ARC-03 und ARC-07. Zwei Punkte trägt der
Katalog nicht:

Die Abschnittsnummern beziehen sich auf den UStAE in der durch das
BMF-Schreiben vom 15.10.2025 geänderten Fassung. Achtung bei der Nachprüfung: die
E-Rechnungs-Regelungen stehen in **Abschnitt 14.1** (Rechnung), nicht in
Abschnitt 14.4 (Echtheit und Unversehrtheit) – eine Verwechslung, die naheliegt,
weil Abschnitt 14.4 ebenfalls geändert wurde.

### Nicht geprüft

Die technischen Spezifikationen selbst – EN 16931, XRechnung (KoSIT) und ZUGFeRD
(FeRD) – sind für dieses Konzept nicht ausgewertet worden. Vor der Umsetzung des
Einlesens sind sie die maßgebliche Quelle für Feldbezeichner; die oben genannte
`BT-72` stammt aus dem allgemeinen Sprachgebrauch der Norm und ist **nicht
verifiziert**.
