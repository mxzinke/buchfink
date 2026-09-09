# Buchfink – Gründung einer Kapitalgesellschaft

Gesetzliche Grundlage: [Anforderungskatalog](anforderungskatalog.md), GOB-01,
GOB-06, JAB-07, UST-03, QUE-02

Status: umgesetzt
Letzte Aktualisierung: 2026-09-08
Voraussetzung: [Beleg- & Buchungsflow](anforderung-beleg-buchungsflow.md)

> Kontonummern sind gegen `internal/accounting/skr04_2026.json` (DATEV SKR04 2026,
> Art.-Nr. 11175) geprüft. Alle Paragrafenangaben sind am **01.09.2026** gegen den
> Gesetzestext auf gesetze-im-internet.de verifiziert. Eröffnungsbilanz, Offenlegung
> und Voranmeldungszeitraum stehen im Anforderungskatalog unter GOB-01, GOB-06,
> JAB-07 und UST-03; das Gründungsrecht führt [Abschnitt 9](#9-fundstellen).

## 1. Worum es geht

Eine GmbH entsteht in zwei Schritten – nicht beim Notar und nicht mit der ersten
Rechnung. Beim Notar wird der Gesellschaftsvertrag beurkundet (§ 2 GmbHG),
und von da an existiert die **Vorgesellschaft**. Als juristische Person entsteht
die GmbH erst mit der Eintragung ins Handelsregister (§ 11 Abs. 1 GmbHG).
Dazwischen liegen Wochen bis Monate, und in dieser Zeit ist das Unternehmen
bereits buchführungspflichtig, zahlt Notar- und Gerichtsgebühren, mietet an und
kauft ein.

Zwei Haftungen gehören zu dieser Phase, und beide überraschen Gründer
regelmäßig:

**Handelndenhaftung.** „Ist vor der Eintragung im Namen der Gesellschaft
gehandelt worden, so haften die Handelnden persönlich und solidarisch"
(§ 11 Abs. 2 GmbHG). Sie endet mit der Eintragung. Buchfink kann sie nicht
abwenden, aber es kann sagen, wie lange sie noch läuft.

**Unterbilanzhaftung.** Bleibt das Reinvermögen der Gesellschaft am Tag der
Eintragung hinter dem Stammkapital zurück, schulden die Gesellschafter die
Differenz, anteilig nach ihren Geschäftsanteilen. Das ist Richterrecht, kein
Paragraf. Verschärft wird es durch § 248 Abs. 1 Nr. 1 HGB: Aufwendungen für die
Gründung eines Unternehmens dürfen nicht aktiviert werden. Die Notarrechnung ist
also sofort Aufwand und mindert das Reinvermögen in voller Höhe.

Das Rechenbeispiel, an dem die ganze Funktion hängt:

| Vorgang | Wirkung auf das Reinvermögen |
|---|---|
| Stammkapital 25.000 € gezeichnet, 12.500 € eingezahlt | 25.000 € (Bank plus offene Einlageforderung) |
| Notar- und Gerichtskosten 3.000 €, aus der Bank bezahlt | 22.000 € |
| **Unterbilanz am Eintragungstag** | **3.000 €**, davon 1.800 € und 1.200 € bei 60/40 |

Die 3.000 € stehen seit dem Tag der Notarrechnung im Journal. Buchfink hat sie
bisher nur nicht zusammengezählt.

## 2. Für wen der Gründungsweg gilt

Nur für Kapitalgesellschaften: GmbH, UG (haftungsbeschränkt), AG. Bei einer
Personengesellschaft gibt es keine Vorgesellschaft, keine Handelndenhaftung des
§ 11 Abs. 2 GmbHG und keine Unterbilanzhaftung. Der Katalog steht in
`internal/accounting/gruendung.go`; `FoundationRulesFor` ist die einzige Stelle,
an der entschieden wird, ob eine Rechtsform dazugehört. Die Oberfläche fragt
dort, statt Rechtsformnamen zu vergleichen.

Die drei Phasen:

| Phase | Beginn | Ende | Was gilt |
|---|---|---|---|
| Vorgründung | Entschluss | Beurkundung | GbR oder OHG, **nicht** identisch mit der späteren GmbH. Verbindlichkeiten gehen nicht automatisch über. |
| Vorgesellschaft | Beurkundung | Eintragung | Buchführungspflicht, Handelndenhaftung, die Unterbilanz wächst mit jeder Buchung. |
| Eingetragene Gesellschaft | Eintragung | — | Haftungsbeschränkung greift, die Unterbilanz steht fest. |

Buchfink bildet die Phasen zwei und drei ab. Die Vorgründung wird benannt und
nicht verwaltet: vor der Beurkundung gibt es keinen Mandanten und nichts zu
buchen.

## 3. Kapitalaufbringung

| Rechtsform | Mindestkapital | Vor der Anmeldung zu leisten | Fundstelle |
|---|---|---|---|
| GmbH | 25.000 € | Auf jeden Geschäftsanteil ein Viertel des Nennbetrags; zusammen mindestens die Hälfte des **Mindest**stammkapitals, also 12.500 € | § 5 Abs. 1, § 7 Abs. 2 GmbHG |
| UG (haftungsbeschränkt) | 1 € | Das Stammkapital in voller Höhe, Sacheinlagen ausgeschlossen | § 5a Abs. 1 und 2 GmbHG |
| AG | 50.000 € | Bei Bareinlagen mindestens ein Viertel des geringsten Ausgabebetrags | § 7, § 36a Abs. 1 AktG |

Die Untergrenze der GmbH ist der Punkt, an dem eine Umsetzung leicht danebengreift.
§ 7 Abs. 2 Satz 2 GmbHG nennt die Hälfte des Mindeststammkapitals nach § 5 Abs. 1,
nicht die Hälfte des vereinbarten Stammkapitals. Wer eine GmbH mit 100.000 €
gründet, schuldet vor der Anmeldung 25.000 € (ein Viertel je Anteil), nicht
50.000 €. `FoundationRules.RequiredPaidIn` rechnet deshalb beides und nimmt den
höheren Wert.

Sacheinlagen zählen mit ihrem vollen Nennbetrag, weil sie vor der Anmeldung
vollständig zu bewirken sind (§ 7 Abs. 3 GmbHG) und § 7 Abs. 2 Satz 2 sie mit dem
Gesamtnennbetrag in die Rechnung nimmt.

## 4. Die Unterbilanzrechnung

```
Reinvermögen = Aktiva + noch ausstehende Einlagen − Schulden
Unterdeckung = Stammkapital − Reinvermögen        (nie negativ)
gedeckt      = min(Gründungsaufwand laut Satzung, Unterdeckung)
Haftung      = Unterdeckung − gedeckt
je Gesellschafter = Haftung × Geschäftsanteil ÷ Stammkapital
```

Drei Entscheidungen darin sind erklärungsbedürftig.

**Die ausstehende Einlage zählt zum Reinvermögen.** Konto 1298 steht als
Forderung auf der Aktivseite, Konto 2910 als offene Absetzung vom gezeichneten
Kapital auf der Passivseite (§ 272 Abs. 1 Satz 3 HGB). Der Sache nach sind beide
Ansprüche gegen den Gesellschafter, und beide gehören ins Reinvermögen. Ließe man
sie weg, zeigte die Rechnung eine Unterbilanz in Höhe der noch nicht gezahlten
Einlage. Das wäre falsch: Die Einlage wird geschuldet, aber als Einlage, nicht
als Vorbelastungshaftung. Beide Ansprüche stehen nebeneinander.

**Der Stichtag ist der Tag der Eintragung.** Fehlt er noch, rechnet Buchfink auf
heute und schreibt „vorläufig" daneben. Eingefroren wird die Zahl nicht: Die
Buchungen bis zum Eintragungstag hängen in der Hash-Chain, und was sich nicht
mehr ändern kann, muss auch nicht zweimal gespeichert werden. Eine rückdatierte
Buchung könnte das Ergebnis noch verschieben; davor schützt die Festschreibung
des Zeitraums, nicht dieser Dienst.

**Die Satzungsklausel deckt die Unterdeckung, nicht bestimmte Buchungen.**
Buchfink unterscheidet nicht, ob die Unterdeckung aus dem Gründungsaufwand oder
aus einem Anlaufverlust stammt. Diese Zuordnung kann nur der Gründer treffen.
Unterdeckung, gedeckter Anteil und verbleibende Haftung stehen deshalb als drei
Zahlen nebeneinander, statt zu einer verschmolzen zu werden.

Gerechnet wird über alle Geschäftsjahre bis zum Stichtag, nicht über das gerade
aktive. Beurkundung im November, Eintragung im Februar ist der Regelfall — das
Handelsregister braucht Wochen. Auf das aktive Jahr eingeschränkt zählte die
Rechnung im Februar nur die Buchungen des neuen Jahres, und weil die Zeichnung
des Stammkapitals im alten steht, wies sie näherungsweise das volle Stammkapital
als Unterbilanz aus.

Der fehlende Saldenvortrag (siehe [stand-der-umsetzung.md](stand-der-umsetzung.md),
Abschnitt 3) stört dabei nicht: Bis zur Eintragung sind Bewegung und Bestand
dasselbe, denn vor der Beurkundung gab es keine Buchung.

## 5. Buchungen

Zwei Sätze, mehr ist es nicht.

| Schritt | Buchung |
|---|---|
| Zeichnung, am Tag der Beurkundung | SOLL **1298** Ausstehende Einlagen, eingefordert 25.000,00 · HABEN **2900** Gezeichnetes Kapital 25.000,00 |
| Einzahlung je Gesellschafter | SOLL **1800** Bank 7.500,00 · HABEN **1298** Ausstehende Einlagen, eingefordert 7.500,00 |

Gebucht wird über `JournalService.Post`, also denselben Weg wie jede andere
Buchung. Hash-Chain, Belegnummernkreis und Audit-Log greifen unverändert. Die
Quelle ist `EntrySourceOpening`, die bisher definiert war und von keiner Stelle
erzeugt wurde.

Buchfink bucht das volle Stammkapital als eingefordert. Ein nicht eingeforderter
Teil (Konto 2910) ist eine Entscheidung der Gesellschafter und folgt nicht aus den
erfassten Daten; wer ihn braucht, bucht ihn von Hand. Die Unterbilanzrechnung
liest das Konto trotzdem.

Eine Sacheinlage wird nicht automatisch gebucht. Sie gehört auf das Konto des
eingebrachten Gegenstands, und welches das ist, weiß nur der Gründer. Die
Vorschau nennt sie und sagt, warum sie fehlt.

Gebucht wird auf Freigabe, mit Vorschau, wie beim Abschreibungslauf. Buchungen,
die eine Anwendung von sich aus schreibt, sind in einer GoBD-Buchhaltung die
schlechtere Hälfte der Bequemlichkeit.

Der Gründungsaufwand selbst wird über **6825** Rechts- und Beratungskosten oder
**6827** Abschluss- und Prüfungskosten gebucht, als gewöhnlicher Beleg. Ein eigener
Erfassungsweg wäre nur eine zweite Art, dasselbe zu tun.

## 6. Fristen aus der Gründung

Jede Pflicht hängt an dem Ereignis, das sie auslöst. Ein einheitlicher Anker
wäre für die eine Hälfte zu früh und für die andere zu spät.

| Pflicht | Anker | Frist | Fundstelle |
|---|---|---|---|
| Anmeldung zum Handelsregister | Beurkundung | sobald die Mindesteinlage geleistet ist | §§ 7, 8 GmbHG |
| Fragebogen zur steuerlichen Erfassung | Beurkundung | einen Monat | § 138 Abs. 1b und Abs. 4 AO |
| Eröffnungsbilanz auf den Beurkundungstag | Beurkundung | im ordnungsmäßigen Geschäftsgang | § 242 Abs. 1 HGB, Katalog GOB-01 und GOB-06 |
| Gewerbeanmeldung bei der Gemeinde | Eintragung | einen Monat | § 14 GewO, § 138 Abs. 1 AO |
| Wirtschaftlich Berechtigte melden | Eintragung | unverzüglich | § 20 Abs. 1 GwG |
| Gesetzliche Rücklage, nur UG | Abschlussstichtag | mit dem Jahresabschluss | § 5a Abs. 3 GmbHG |
| Ersten Jahresabschluss offenlegen | Abschlussstichtag | zwölf Monate | § 325 Abs. 1a HGB, Katalog JAB-07 |

Zwei Anker sind erklärungsbedürftig, weil man beide für die Eintragung halten
könnte.

**Der Fragebogen hängt an der Beurkundung.** Die Vorgesellschaft ist mit der
späteren GmbH dasselbe Rechtssubjekt: sie ist bereits buchführungspflichtig und
bereits Körperschaftsteuersubjekt. Anzuzeigen ist nach § 138 Abs. 1b AO die
Aufnahme der Tätigkeit, und die beginnt beim Notar. Die Frist an die Eintragung
zu hängen wäre auch praktisch verkehrt: aus dem Fragebogen folgt die
Steuernummer, und ohne sie gibt es keine Rechnung mit Steuerausweis — der
Gründer wartete auf das Register, um überhaupt abrechnen zu können.

**Die Gewerbeanmeldung hängt an der Eintragung.** Das ist eine Wertung und keine
Ableitung: § 14 GewO stellt auf die Aufnahme des Betriebs ab, die schon vorher
liegen kann. Das Gewerbeamt führt die Gesellschaft aber unter ihrer
Registernummer und verlangt den Registerauszug; vor der Eintragung ist die
Anmeldung nicht zu erledigen. Buchfink setzt die Frist deshalb dorthin, wo sie
erfüllbar wird, und sagt das am Vorgang.

Wo das Gesetz „unverzüglich" sagt, steht der Wortlaut in der Liste, kein Datum.
Eine erfundene Tagesfrist wäre bequemer und falsch. Bei der
Eröffnungsbilanz nennt § 242 Abs. 1 HGB ebenfalls keine Frist; der angezeigte
Termin ist als Richtwert gekennzeichnet und stammt aus § 264 Abs. 1 Satz 4 HGB.

Was die Eintragung voraussetzt, steht schon vorher in der Liste — als wartender
Posten ohne Datum. Der Gründer soll sehen, was auf ihn zukommt, ohne dafür
überfällig zu sein. Bis Welle 9 fiel eine Pflicht ohne Tagesdatum ganz aus der
Fristenliste heraus, und das traf genau die beiden, die keins haben: die
Anmeldung zum Handelsregister und die Meldung an das Transparenzregister.

Erledigt wird eine Gründungspflicht mit ihrem Datum in der Datenbank, nicht mit
einem Haken im `localStorage`. Dass der Fragebogen am 12. Oktober übermittelt
wurde, ist eine Tatsache über das Unternehmen. Für die übrigen Steuertermine
bleibt der Haken, was er ist: eine Merkhilfe.

### Der Voranmeldungszeitraum bei Neugründung

Hier stand in der Oberfläche fünf Jahre lang das Falsche. Der
Einrichtungsassistent riet „monatlich gilt bei Neugründung". § 18 Abs. 2 Satz 4
UStG verlangt das auch, aber Satz 6 setzt die Pflicht für die
Besteuerungszeiträume **2021 bis 2026** aus. In dieser Zeit gilt der Regelfall
des Satzes 2, also das Kalendervierteljahr. Ab dem Besteuerungszeitraum 2027 lebt
die monatliche Pflicht wieder auf.

`accounting.RecommendedVatPeriod` beantwortet das aus dem Gründungsjahr,
`VatPeriodReason` liefert die Begründung dazu. Beides steht an einer Stelle, weil
die Regel ein Stichjahr hat: Ein fest getippter Hinweis war seit 2021 falsch und
wäre es ab 2027 wieder.

## 6a. Das erste Geschäftsjahr

Aus der Beurkundung folgt der Beginn des ersten Geschäftsjahres: eine
Gesellschaft, die im März entstanden ist, hat kein Geschäftsjahr, das im Januar
begonnen hätte — der Zeitraum davor gehörte zu einem Unternehmen, das es noch
nicht gab. Das Gründungsjahr ist deshalb ein Rumpfgeschäftsjahr (§ 8b EStDV,
Katalog GOB-06).

`ClosingService.derive` konnte das schon immer, kam im Einrichtungsassistenten
aber zu spät: der Mandant entsteht mit `CreateTenant`, und dabei legt
`EnsureFiscalYears` das laufende Jahr an — die Gründung wird erst danach erfasst.
Das Geschäftsjahr stand dann als volles Kalenderjahr in der Datenbank, und zwar
dauerhaft, denn vorhandene Einträge rührt der Lauf nicht mehr an.

`AlignFoundingYear` zieht den Beginn nach, sobald die Gründung gespeichert wird,
und nur solange nichts daran hängt:

| Bedingung | Warum |
|---|---|
| Der Abschluss ist nicht festgestellt | Der Zeitraum ist Teil des festgestellten Abschlusses |
| Kein Zeitraum ist festgeschrieben | Ein festgeschriebener Zeitraum behält seinen Beginn |
| Keine Buchung liegt vor dem Beurkundungstag | Sie läge danach außerhalb ihres Geschäftsjahres |

Scheitert eine Bedingung, bleibt das Jahr, wie es ist, und der Grund steht im
Änderungsprotokoll. Die Gründung ist trotzdem gespeichert: sie ist die Tatsache,
das Geschäftsjahr die Folge daraus.

Beim Start wird nichts angeglichen. Ein bestehender Mandant, der sein
Geschäftsjahr bereits geführt hat, wird nicht umgeschrieben — eine Migration, die
den Zeitraum einer laufenden Buchführung verschiebt, wäre ein stiller Eingriff in
den Abschluss.

## 6b. Der Zustand in der Oberfläche

Die Vorgesellschaft ist die Lage des ganzen Unternehmens und nicht die einer
Ansicht. Sie steht deshalb als Hinweisstreifen über der Aufgabenliste — ein Satz
mit dem Beurkundungsdatum, dem Zusatz „i. G." und der Handelndenhaftung — und
verschwindet mit der Eintragung von selbst.

Die Erklärung dazu liegt in `frontend/src/components/GruendungHelp.tsx` und wird
von beiden Stellen benutzt, dem Streifen und dem Gründungsabschnitt der
Fristenseite. Die drei Stufen folgen dem Entwurfskonzept §15.2: ein Satz auf der
Fläche, ein bis drei Sätze im Popover, die Rechnung mit ihrem Beispiel im Dialog
hinter „Mehr dazu".

Die anteilige Haftung je Gesellschafter wird gerechnet und nicht ausgewiesen. Sie
ist eine Aussage über Personen, sie hängt an einer Zahl, die bis zur Eintragung
vorläufig ist, und für den nächsten Schritt — Einlage leisten, anmelden,
eintragen — ändert sie nichts. Die Gesellschafterzeile zeigt stattdessen die
Kapitalaufbringung. `UnterbilanzShare` bleibt in der Schnittstelle; ausgeblendet
ist die Darstellung, nicht die Rechnung.

## 6c. Der Firmenzusatz „i. G."

Bis zur Eintragung ist die Gesellschaft noch keine juristische Person, die
Haftungsbeschränkung greift nicht, und wer mit ihr abschließt, soll das am Namen
erkennen. Sie führt deshalb den Zusatz „i. G.".

Er wird abgeleitet und nirgends gespeichert: `CompanySettings.InGruendung` setzt
die Stammdaten-Abfrage aus der Gründungszeile, `CompanySettings.FirmName()` hängt
den Zusatz an. Mit der Eintragung fällt er von selbst weg. Ein gespeichertes
Kennzeichen ginge irgendwann mit dem Eintragungsdatum auseinander — derselbe
Grund, aus dem `Foundation.Stage()` abgeleitet ist.

Der Zustand wird beim Lesen der Unternehmensdaten gesetzt und nicht von jedem
Ausgabeweg einzeln erfragt: den Firmennamen brauchen Rechnung, E-Rechnung,
E-Bilanz, Abschlusskopf, Mahnschreiben und Eigenbeleg, und jedem von ihnen die
Gründung durchzureichen hieße, sieben Dienste um eine Abhängigkeit zu erweitern,
die sie nur weitergeben.

| Weg | Feld |
|---|---|
| Rechnungs-PDF | Dokumententitel und Absenderzeile |
| ZUGFeRD und XRechnung | BT-27 Verkäufername, BT-85 Kontoinhaber |
| E-Bilanz | `de-gcd:genInfo.company.id.name` |
| Bilanz und GuV | Firma im Kopf, § 264 Abs. 1a Nr. 1 HGB |
| Mahnschreiben | Briefkopf und Kontoinhaber |
| Eigenbeleg | Aussteller |
| Z3-Export | Datenlieferant |
| Verfahrensdokumentation | Unternehmen |

Roh bleibt der Name, wo er Eingabe oder Schlüssel ist: im Eingabefeld der
Einstellungen, im Mandantennamen der Anwendungskonfiguration und im
Änderungsprotokoll. Der Mandantenname liegt in einer JSON-Datei — ein dort
eingebrannter Zusatz verschwände mit der Eintragung nicht mehr.

Im Bilanzkopf wird die Firma an einer Stelle zusammengesetzt und als eigenes Feld
neben dem erfassten Namen geführt. Sonst hängte `headerTitle` die Rechtsform
hinter den Zusatz: „Muster Ventures i. G. GmbH" ist keine Firma.

Bereits erzeugte Rechnungen und Belege bleiben, wie sie abgelegt sind. Neu
gerechnet wird nur die Vorschau, und dort ist der aktuelle Stand der richtige.

## 6d. Der geführte Weg

Die Pflichten stehen nicht nur als Liste, sondern als Weg: nummeriert in der
Reihenfolge des Tuns, mit Fortschritt und dem Satz, was als Nächstes ansteht.
Er folgt dem geführten Weg des Jahresabschlusses (architektur.md, Abschnitt 6.3):
die Schritte kommen aus dem Backend, gezählt wird dort, und jeder Schritt öffnet
seine Arbeit da, wo sie wohnt.

Was ihn davon unterscheidet, ist der Anlass. Wer zum ersten Mal gründet, weiß
nicht, wohin er sich wenden soll. Jede Pflicht trägt deshalb drei Angaben, die
in keinem Paragrafen stehen:

| Feld | Inhalt |
|---|---|
| `Where` | Der Ort: beim Notar, über Mein ELSTER, beim Gewerbeamt, auf transparenzregister.de |
| `Todo` | Die Handgriffe, einzeln und in ihrer Reihenfolge |
| `Provides` | Was Buchfink dazu beisteuert — leer, wo es nichts beisteuern kann |

Die Seite hat keinen Eintrag in der Navigation. Der Weg ist eine Phase und kein
Ort; erreicht wird er über den Hinweisstreifen der Startseite, über den
Gründungsabschnitt der Fristenseite und aus dem Einrichtungsassistenten. Wer dort
eine Gründung erfasst hat, landet auf dem Weg statt in der Aufgabenliste — die
ist am ersten Tag leer.

`FoundationGuide` trägt den Stand: `Total`, `Done`, `Open` und `Waiting`.
Wartende Schritte zählen nicht als offen, denn zu tun ist an ihnen gerade nichts.

## 6e. Die Eröffnungsbilanz

Sie steht auf den Tag der Beurkundung (§ 242 Abs. 1 HGB) und nicht auf den der
Eintragung: mit ihm beginnt das Handelsgewerbe. Sie ist der eine
Gründungsvorgang, den Buchfink vollständig aus den eigenen Zahlen kann — an
diesem Tag stehen in den Büchern die Zeichnung des Stammkapitals und, soweit
schon geflossen, die Einlage.

`StatementService.StatementAt` baut die Gliederung dafür auf einen Stichtag
statt auf ein Geschäftsjahr: über alle Jahre bis zu diesem Tag, weil eine
Beurkundung im November und ein aktives Geschäftsjahr im Januar auseinander
liegen können, und ohne Vorjahresspalte, weil es vor dem ersten Tag des
Unternehmens kein Vorjahr gibt.

Drei Ausgaben:

- **Zahlenwerk** in der Ansicht, mit Aktiva, Passiva und der Frage, ob beides
  übereinstimmt. Tut es das nicht, fehlt eine Buchung — die Ansicht sagt welche
  und führt dorthin.
- **PDF**, abgelegt in der Dokumentenablage unter seiner Prüfsumme. Abgelegt und
  nicht nur heruntergeladen: die Eröffnungsbilanz ist eine Unterlage nach § 147
  Abs. 1 Nr. 1 AO und wird zehn Jahre gebraucht.
- **E-Bilanz** als XBRL-Instanz. Die Eröffnungsbilanz ist eine Bilanz im Sinne
  des § 5b Abs. 1 EStG und damit elektronisch zu übermitteln; die Instanz nennt
  die Bilanzart, hat keine Gewinn- und Verlustrechnung und kein Vorjahr, und ihr
  Berichtszeitraum ist der eine Tag, an dem das Handelsgewerbe beginnt. Der
  Elementname der Bilanzart trägt denselben Vorbehalt wie die übrigen: vor der
  Übermittlung gegen die amtliche Taxonomie auf esteuer.de abgleichen.

Eine Bilanz, die nicht aufgeht, wird nicht abgelegt. Nach Registergericht und
-nummer wird vor der Eintragung nicht gefragt: sie können dann noch nicht
vorliegen.

## 6f. Das Datenblatt zum Fragebogen

Der Fragebogen wird über Mein ELSTER übermittelt; ERiC bleibt außerhalb des
Funktionsumfangs. Buchfink stellt zusammen, was es kennt — Firma, Sitz,
Rechtsform, Beurkundung, Stammkapital, Gesellschafter, Bankverbindung,
Voranmeldungszeitraum — und nennt ausdrücklich, was der Fragebogen außerdem
verlangt und in keinem Konto steht: voraussichtliche Umsätze, Betriebseröffnung,
Beschäftigte, Kleinunternehmerregelung, Empfangsvollmacht, Lastschriftmandat.

Ohne diese Liste läse sich das Datenblatt wie eine vollständige Antwort. Die
Steuernummer steht nicht darin: sie ist das Ergebnis des Fragebogens und nicht
seine Angabe.

## 6g. Die Dokumentenablage

Sie steht neben dem Beleg und neben dem Anlagendokument, weil sie eine dritte
Sache ist. Ein Beleg gehört zu einer Buchung und einem Geschäftsjahr, ein
Anlagendokument zu einem Wirtschaftsgut. Der Gesellschaftsvertrag gehört zu
keinem von beidem — er gehört zum Unternehmen und gilt, solange es das
Unternehmen gibt.

Der Ablageweg ist derselbe wie überall: die Datei liegt unter ihrem eigenen
SHA256, unverschlüsselt, nur Pfad und Dateiname sind in der Datenbank
verschlüsselt, und herausgegeben wird sie erst, nachdem die Prüfsumme stimmt.
Aufbewahrt wird zehn Jahre (§ 147 Abs. 1 Nr. 1 AO, Organisationsunterlagen).

Ein Dokument kann über `DutyKey` an einer Gründungspflicht hängen, ohne ihr zu
gehören: der Registerauszug erscheint als Nachweis am Schritt und bleibt in der
Ablage, wenn die Gründung längst vorbei ist. Die Ablage ist freiwillig — ein
Schritt gilt mit seinem Datum als erledigt und nicht erst mit einer Datei.

Gezeigt wird sie unter „Unterlagen" in der Verwaltung: eine Liste aus Art,
Bezeichnung, Datum, Größe und Aufbewahrungsfrist, das jüngste Dokument zuerst.
Bewusst eine Liste und keine Verwaltung — was fehlt, fehlt, weil es bei zwei
Dutzend Unterlagen nichts zu suchen gibt. Was Buchfink selbst erzeugt hat, ist
gekennzeichnet: eine Eröffnungsbilanz lässt sich jederzeit neu herstellen, ein
Registerauszug nicht, und wer beide für gleich unersetzlich hält, sichert das
Falsche.

Damit ist die offene Entscheidung „Ablage der Gründungsurkunden" aus Abschnitt 8
erledigt. Die Ablage ist von Anfang an auf den Mandanten gehoben und nicht auf
die Gründung: sie nimmt später den Mietvertrag und den Versicherungsschein
genauso auf.

## 7. Datenmodell

```
Foundation                 höchstens eine je Mandant
├── notarizedOn            Beurkundung, setzt Rumpfgeschäftsjahr und Fristen
├── registeredOn           leer = Vorgesellschaft; die Phase folgt daraus
├── registerCourt/-Number  Amtsgericht, HRB
├── shareCapital           Stammkapital laut Satzung
├── foundationCostCap      Gründungsaufwand laut Satzung
└── shareholders[]         Name (verschlüsselt), Anteil, geleistet, Bar/Sache

FoundationTask             erledigte Pflicht: Schlüssel, Datum, Notiz

FoundationDuty             abgeleitet, nicht gespeichert
├── anchor                 beurkundung | eintragung | abschlussstichtag
└── isPending              das auslösende Ereignis steht noch aus
```

Die Phase ist abgeleitet, nicht gespeichert. Ein Status, den man unabhängig vom
Datum setzen kann, geht irgendwann mit ihm auseinander, und dann steht in der
Oberfläche etwas anderes als in der Rechnung.

Der Gesellschaftername ist personenbezogen und liegt verschlüsselt, wie jedes
andere Namensfeld (`serializer:encrypted`).

Die Fundstellen im Code:

| Was | Wo |
|---|---|
| Typen und Repository-Schnittstelle | `internal/domain/foundation.go` |
| Regeln je Rechtsform, Fristen, § 18 UStG | `internal/accounting/gruendung.go` |
| Persistenz | `internal/repository/foundation_gorm.go` |
| Unterbilanz, Anmeldungsbefund, Buchungen, Eintragung | `internal/service/foundation_service.go` |
| Rumpfjahr aus der Beurkundung | `internal/service/closing_service.go`, `AlignFoundingYear` |
| Firmenzusatz „i. G." | `internal/domain/settings.go`, `FirmName`; `internal/repository/settings_gorm.go` |
| Bridge | `internal/wailsbridge/app_service.go`, Abschnitt „Gründung" |
| Erfassung | `frontend/src/components/SetupAssistantScreen.tsx` |
| Laufende Begleitung | `frontend/src/components/FoundationSection.tsx`, `frontend/src/pages/DeadlinesPage.tsx` |
| Hinweis und Erklärung | `frontend/src/components/GruendungHelp.tsx`, `frontend/src/pages/TasksPage.tsx` |
| Der geführte Weg | `frontend/src/pages/GruendungPage.tsx` |
| Eröffnungsbilanz | `internal/service/foundation_opening.go`, `internal/service/statement_service.go` (`StatementAt`) |
| E-Bilanz der Eröffnungsbilanz | `internal/service/ebilanz_service.go` (`ExportOpeningXBRL`), `internal/ebilanz/ebilanz.go` |
| Datenblatt zum Fragebogen | `internal/service/foundation_fragebogen.go` |
| Dokumentenablage | `internal/domain/document.go`, `internal/service/document_service.go` |

## 8. Offene Entscheidungen

**GmbH & Co. KG.** Zwei Gesellschaften, zwei Eintragungen, Kapitalkonten der
Kommanditisten. Die Komplementär-GmbH durchläuft den hier beschriebenen Weg, die
KG einen anderen. Solange die Kapitalkonten der Gesellschafter fehlen (siehe
[stand-der-umsetzung.md](stand-der-umsetzung.md), Abschnitt 4), wäre der zweite
Teil ohnehin nicht darstellbar.

**Bewertung von Sacheinlagen.** Buchfink erfasst, dass eine vorliegt und mit
welchem Nennbetrag. Der Sachgründungsbericht nach § 5 Abs. 4 GmbHG, die
Werthaltigkeitsprüfung und die Differenzhaftung des § 9 GmbHG sind keine
Software-Aufgabe.

**Nicht eingeforderte Einlagen.** Konto 2910 wird gelesen, aber nicht angeboten.
Ob und in welcher Höhe eingefordert wird, entscheiden die Gesellschafter; es aus
dem gesetzlichen Mindestbetrag zu erraten wäre eine Festlegung, die niemand
getroffen hat.

**Die UG-Rücklage nach § 5a Abs. 3 GmbHG.** Sie hängt am Jahresüberschuss und
damit an den Abschlussbuchungen, die es noch nicht gibt. Buchfink führt die
Pflicht als Frist und bucht sie nicht.

**Übermittlung des Fragebogens.** ERiC bleibt out-of-scope (README, „Scope &
Entscheidungen"). Buchfink führt die Frist und den Nachweis; übermittelt wird
über Mein ELSTER.

**Was die Dokumentenablage noch nicht kann.** Die Ansicht „Unterlagen" steht
(Abschnitt 6g) und zeigt die Ablage als Liste. Sie hat keine Suche, keine Ordner
und keine Versionen — bei zwei Dutzend Unterlagen gibt es nichts zu suchen. Wenn
dort einmal jeder Miet- und Versicherungsvertrag liegt, wird sich das ändern.

**Die Zuordnung der Unterdeckung.** Ob eine Unterdeckung aus dem Gründungsaufwand
oder aus einem Anlaufverlust stammt, entscheidet heute der Nutzer im Kopf. Eine
Kennzeichnung am Beleg wäre möglich und ist bewusst nicht gebaut: sie verlangte
eine Angabe bei jeder Buchung der Gründungsphase, und die meisten Gründer haben
gar keine Satzungsklausel.

**Die Haftung je Gesellschafter.** Sie wird gerechnet und vorerst nicht
ausgewiesen (Abschnitt 6b). Wieder einzublenden ist sie eine Frage von wenigen
Zeilen — die Zahlen stehen in der Schnittstelle.

**Das Geschäftsjahr eines Bestandsmandanten.** Wer vor dieser Welle gegründet
hat, führt sein Gründungsjahr weiterhin als volles Kalenderjahr. Beim Start wird
nichts umgeschrieben (Abschnitt 6a); den Zeitraum von Hand richtigzustellen gibt
es noch keinen Weg.

## 9. Fundstellen

Eröffnungsbilanz und Rumpfgeschäftsjahr stehen im Anforderungskatalog unter GOB-01
und GOB-06, die Offenlegung unter JAB-07, der Voranmeldungszeitraum unter UST-03,
die Verschlüsselung personenbezogener Felder unter QUE-02. Das Gründungsrecht selbst
kennt der Katalog nicht; dafür gilt, geprüft am 01.09.2026 gegen den Wortlaut auf
gesetze-im-internet.de:

- § 5 GmbHG – Stammkapital, Geschäftsanteil, Sachgründungsbericht
- § 5a GmbHG – Unternehmergesellschaft: Volleinzahlung, Sacheinlageverbot, gesetzliche Rücklage
- § 7 GmbHG – Anmeldung, Viertelregel, Untergrenze, Sacheinlagen
- § 11 GmbHG – Vorgesellschaft und Handelndenhaftung
- § 7 AktG, § 36a AktG – Grundkapital und Leistung der Einlagen
- § 248 Abs. 1 Nr. 1 HGB – Aktivierungsverbot für Gründungsaufwand
- § 272 Abs. 1 HGB – Ausweis von gezeichnetem Kapital und ausstehenden Einlagen
- § 138 AO – Anzeige der Erwerbstätigkeit, Monatsfrist in Absatz 4
- § 18 Abs. 2 UStG – Voranmeldungszeitraum, Aussetzung 2021 bis 2026 in Satz 6
- § 20 GwG – Mitteilung an das Transparenzregister
- § 14 GewO – Gewerbeanmeldung
- § 5b Abs. 1 EStG – elektronische Übermittlung der Bilanz; die Eröffnungsbilanz ist eine
- § 147 Abs. 1 Nr. 1 AO – Aufbewahrung der Organisationsunterlagen, zehn Jahre

Die Unterbilanzhaftung (Vorbelastungshaftung) ist Richterrecht des BGH und steht
in keinem Paragrafen. Sie ist ständige Rechtsprechung; die Rechnung folgt der
herrschenden Auffassung, wonach auf den Tag der Eintragung abzustellen ist und
die Gesellschafter anteilig nach ihren Geschäftsanteilen haften.
