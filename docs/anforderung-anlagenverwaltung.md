# Buchfink – Anlagenverwaltung

Status: umgesetzt
Letzte Aktualisierung: 2026-08-28
Voraussetzung: [Beleg- & Buchungsflow](anforderung-beleg-buchungsflow.md)

> **Umsetzung.** Das Anlagenverzeichnis liegt unter „Anlagevermögen" in der
> Seitenspalte. Der Kern steht in `internal/domain/asset.go` (Anlagegut und
> Bewegung), `internal/accounting/afa.go` (Wertgrenzen, Zeitfenster der
> degressiven AfA, Abschreibungsplan), `internal/accounting/asset_accounts.go`
> (Kontenkatalog und die Ableitung der Abgangskonten) und
> `internal/service/asset_service.go` (Kartei, Abschreibungslauf, Abgang,
> Anlagenspiegel). Die Oberfläche ist `frontend/src/pages/AssetsPage.tsx`.

> Kontonummern sind gegen `internal/accounting/skr04_2026.json` (DATEV SKR04 2026,
> Art.-Nr. 11175) geprüft. Alle Paragrafenangaben sind gegen den Gesetzestext auf
> gesetze-im-internet.de verifiziert — der ursprüngliche Bestand am **22.08.2026**,
> die Vorschriften zur Sonderabschreibung, zur Fremdwährung und zum
> Erhaltungsaufwand am **28.08.2026**; die Fundstellen stehen in
> [Abschnitt 11](#11-quellen). Die AfA-Regeln ändern sich häufig – vor der
> Umsetzung erneut prüfen, insbesondere die befristete degressive AfA.

## 1. Die erste Frage ist nicht die AfA-Methode

Bei jeder Anschaffung steht eine Entscheidung vor der Abschreibung, und sie
entscheidet über den ganzen weiteren Verlauf:

| Fall | Voraussetzung | Behandlung | Konten |
|---|---|---|---|
| **Sofortabzug** | selbständig nutzbar, AK bis 800 € (§ 6 Abs. 2 Satz 1 EStG) | voller Aufwand im Anschaffungsjahr | **0670** GWG · **6260** Sofortabschreibungen GWG |
| **Sammelposten** | AK über 250 € bis 1.000 € (§ 6 Abs. 2a Satz 1 EStG) | Pool, im Jahr der Bildung und den folgenden vier Jahren mit je einem Fünftel aufzulösen | **0675** Wirtschaftsgüter (Sammelposten) · **6264** Abschreibungen auf den Sammelposten |
| **Aktivierung** | alles darüber, und alles nicht selbständig Nutzbare | planmäßige AfA über die Nutzungsdauer | Anlagekonto · **6220** / **6222** |

Zwei Fallen stecken darin. **„Selbständig nutzbar"** ist die eigentliche Hürde, nicht
der Betrag: ein Monitor für 300 € ist ohne Rechner nicht nutzbar und damit kein GWG.
Und das **Sammelposten-Wahlrecht gilt einheitlich für alle Wirtschaftsgüter eines
Jahres** – wer einmal poolt, poolt für dieses Jahr durchgehend. Beides muss die
Software abfragen bzw. festhalten, nicht raten.

Ab 250 € ist ein GWG zusätzlich in ein besonderes, laufend zu führendes Verzeichnis
aufzunehmen (§ 6 Abs. 2 Satz 4 EStG) – es sei denn, die Angaben sind aus der Buchführung
ersichtlich (Satz 5). Buchfink führt die Anlagenkartei ohnehin, erfüllt das also.

**Die Wertgrenzen gehören trotzdem in die Stammdaten, versioniert je Geschäftsjahr.**
Sie haben sich in den letzten Jahren mehrfach geändert; ein fest verdrahteter Wert
produziert still falsche Buchungen, sobald ein altes Jahr nachbearbeitet wird.

## 2. Zugang

**Anschaffungskosten nach § 255 Abs. 1 HGB** sind der Anschaffungspreis zuzüglich
Anschaffungsnebenkosten und abzüglich Anschaffungspreisminderungen.

Daraus folgt ein Punkt, den der Zahlungsflow kennen muss: **Skonto auf eine Anlage
mindert die Anschaffungskosten, nicht den Aufwand.** Auf 5736 gebucht wäre es ein Ertrag
des Zahlungsjahres, und die AfA liefe weiter von einem Wert, den das Wirtschaftsgut nie
gekostet hat – ein Fehler, der mit jedem Jahr des Plans wächst.

Der Zahlungsflow fragt deshalb die Kartei, ob die bezahlte Rechnung die Zugangsbuchung
eines Anlageguts war. Ist sie es, geht das Skonto im Haben auf das Anlagekonto statt auf
das Skontokonto, und die Kartei nimmt die Minderung mit Verweis auf die Zahlungsbuchung
auf. Die Steuerkorrektur nach § 17 Abs. 1 UStG bleibt unverändert: sie hängt am Umsatz und
nicht daran, was mit dem Entgelt im Anlagevermögen geschieht. Nur auf der Eingangsseite –
ein gewährtes Skonto mindert den eigenen Erlös.

Ebenso zu erfassen: Fracht, Montage, Überführung sind Nebenkosten und gehören auf das
Anlagekonto; Finanzierungskosten dagegen nicht.

**Erweiterungen kommen später wieder.** Was ein Anlagegut erweitert oder über seinen
ursprünglichen Zustand hinaus wesentlich verbessert, sind nachträgliche Herstellungskosten
(§ 255 Abs. 2 Satz 1 HGB); was es nur im Zustand hält, ist Erhaltungsaufwand und geht
sofort in die GuV. Die Abgrenzung ist eine Einschätzung, keine Rechnung — Buchfink fragt
sie, statt sie zu raten.

Für die AfA gilt: der Betrag wirkt **ab seinem eigenen Jahr**, behandelt als wäre er zu
dessen Beginn angefallen (R 7.4 Abs. 9 EStR). Der Restbuchwert samt Erweiterung verteilt
sich auf die Restnutzungsdauer; verlängert die Erweiterung die Nutzungsdauer, wächst die
Restnutzungsdauer mit. Den Plan von vorn zu rechnen wäre der naheliegende Fehler: er
behauptete rückwirkend, in längst festgeschriebenen Jahren sei zu wenig abgeschrieben
worden.

Zugangsbuchung (Beispiel Pkw auf Ziel):

| Buchung |
|---|
| SOLL **0520** Pkw + SOLL **1406** Vorsteuer · HABEN **Kreditorenkonto** |

Anzahlungen auf noch nicht gelieferte Anlagen und Anlagen im Bau laufen über **0700**
Geleistete Anzahlungen und Anlagen im Bau und werden bei Fertigstellung umgebucht.

Bei der Umbuchung geschehen zwei Dinge, und das zweite wird gern übersehen: das Konto
wechselt, **und die Abschreibung beginnt** — ab der Betriebsbereitschaft, nicht rückwirkend
zur ersten Anzahlung. Buchfink führt dafür ein eigenes Datum am Anlagegut. Im
Anlagenspiegel steht die Umbuchung in einer eigenen Spalte: bei der abgebenden Position
negativ, bei der aufnehmenden positiv, über alle Positionen zusammen null.

## 3. Planmäßige Abschreibung

| Methode | Grundlage | Anmerkung |
|---|---|---|
| **Linear** | § 7 Abs. 1 EStG | Standard; gleichmäßig über die betriebsgewöhnliche Nutzungsdauer |
| **Degressiv** | § 7 Abs. 2 EStG | höchstens das Dreifache des linearen Prozentsatzes und höchstens 30 %; nur für bewegliche Wirtschaftsgüter, die nach dem 30.06.2025 und vor dem 01.01.2028 angeschafft wurden |
| **Sonderabschreibung** | § 7g Abs. 5 EStG | bis zu 40 % der Anschaffungskosten, verteilbar auf das Anschaffungsjahr und die vier Folgejahre; Gewinn des Vorjahres höchstens 200.000 € (§ 7g Abs. 6 i. V. m. Abs. 1 Satz 2 Nr. 1 EStG) |

**Zeitanteilig, monatsgenau** ab dem Anschaffungsmonat (§ 7 Abs. 1 Satz 4 EStG). Eine
im September angeschaffte Anlage wird im ersten Jahr mit vier Zwölfteln abgeschrieben.

Der **Übergang von degressiv auf linear** ist zulässig (§ 7 Abs. 3 EStG) und lohnt sich
ab dem Jahr, in dem die lineare Restwert-AfA höher wäre. Die Software sollte den
optimalen Wechselzeitpunkt errechnen und vorschlagen, nicht den Nutzer rechnen lassen.

**Die Sonderabschreibung tritt neben die planmäßige AfA**, sie ersetzt sie nicht: § 7g
Abs. 5 EStG lässt sie „neben den Absetzungen für Abnutzung nach § 7 Absatz 1 oder Absatz 2"
zu, also neben der linearen wie neben der degressiven. Die allgemeine Regel des
§ 7a Abs. 4 EStG, die nur § 7 Abs. 1 oder 4 nennt, wird davon verdrängt – und weil der
SKR04 die Sonderabschreibung getrennt ausweist, läuft sie auf ein eigenes
Aufwandskonto: **6242** für Fahrzeuge, **6241** für alles andere. Im Anschaffungsjahr wird
sie *nicht* zeitanteilig gekürzt; § 7 Abs. 1 Satz 4 EStG betrifft die Absetzung für
Abnutzung. Mit dem Ende des Begünstigungszeitraums – dem Anschaffungsjahr und den vier
folgenden – verteilt § 7a Abs. 9 EStG den Restwert auf die Restnutzungsdauer; ohne diese
Umstellung stünde das Wirtschaftsgut Jahre vor seinem Ende bei null.

Zwei Voraussetzungen kennt keine Software: der Gewinn des Vorjahres von höchstens
200.000 € und die fast ausschließlich betriebliche Nutzung (§ 7g Abs. 6 EStG). Buchfink
fragt sie und hält die Begründung fest, wie bei der außerplanmäßigen Abschreibung.
Begünstigt sind außerdem nur **bewegliche** Wirtschaftsgüter – ein Gebäude ist eine
Sachanlage wie eine Maschine und bekommt sie trotzdem nicht. Woran das erkennbar ist,
weiß allein der Kontenkatalog.

**Nutzungsdauer** kommt aus den amtlichen AfA-Tabellen. Ein hinterlegter Katalog der
gängigen Fälle wäre nützlich; er ist aber kein Gesetz, sondern eine Verwaltungsanweisung
und muss überschreibbar bleiben.

Buchung:

| Buchung |
|---|
| SOLL **6220** Abschreibungen auf Sachanlagen · HABEN Anlagekonto |
| SOLL **6222** Abschreibungen auf Fahrzeuge · HABEN **0520** Pkw |

Für Gebäude gilt **6221**, für den Sammelposten **6264**, für GWG im Zugangsjahr
**6260**.

## 4. Außerplanmäßige Abschreibung

Bei voraussichtlich dauernder Wertminderung ist auf den niedrigeren beizulegenden Wert
abzuschreiben (§ 253 Abs. 3 Satz 5 HGB). Konto **6230** Außerplanmäßige Abschreibungen
auf Sachanlagen.

Das ist ein Ermessensvorgang und keine Rechnung: die Software kann ihn erfassen und
dokumentieren, aber nicht auslösen. Der Grund gehört zwingend an die Buchung, weil ihn
sonst niemand mehr rekonstruieren kann.

## 5. Abgang

Der Abgang ist der Fall, in dem die meisten Fehler passieren, weil drei Dinge
gleichzeitig geschehen: der Restbuchwert verschwindet, ein Erlös entsteht, und die
Differenz ist Gewinn oder Verlust.

Ablauf:

1. **AfA bis zum Abgangsmonat** nachholen – zeitanteilig bis einschließlich des Monats
   des Abgangs.
2. **Restbuchwert ausbuchen.**
3. **Erlös erfassen** mit Umsatzsteuer.
4. Die Differenz ist der Buchgewinn oder -verlust.

**Der SKR04 wählt das Erlöskonto nach dem Ergebnis, nicht nach dem Vorgang:**

| Fall | Konto |
|---|---|
| Verkauf mit **Buchgewinn**, 19 % USt | **4845** Erlöse aus Verkäufen Sachanlagevermögen 19 % USt (bei Buchgewinn) |
| Verkauf mit **Buchverlust**, 19 % USt | **6885** Erlöse aus Verkäufen Sachanlagevermögen 19 % USt (bei Buchverlust) |
| innergemeinschaftliche Lieferung | **4848** bzw. **6888** |
| Ausfuhr | **4844** bzw. **6884** |

Das ist eine Besonderheit, die man kennen muss: dasselbe Geschäft landet in der GuV
einmal unter den Erträgen und einmal unter den Aufwendungen, je nachdem, ob der
Verkaufspreis über oder unter dem Restbuchwert lag. Die Software muss das Ergebnis
kennen, **bevor** sie das Erlöskonto wählt – die Kontierung hängt hier also vom
Rechenergebnis ab und nicht nur von der Nutzereingabe.

Verschrottung ohne Erlös: Restbuchwert direkt in den Aufwand.

**Teilabgang.** Bei Finanzanlagen ist er der Normalfall: eine Tranche von Anteilen wird
verkauft, eine Ausleihung wird getilgt. Dann gehen nur ein Teil der Anschaffungskosten und
der entsprechende Anteil einer früheren außerplanmäßigen Abschreibung hinaus; der Rest
bleibt im Bestand. Bei Sach- und immateriellen Anlagen ist der Teilabgang nicht vorgesehen
— dort liefe ein Abschreibungsplan, der aufzuteilen wäre, und ein halber Pkw geht nicht ab.

Wo Stücke geführt werden, ist die **Stückzahl die Vorgabe und der Betrag das Ergebnis**:
verkauft wird eine Tranche von 40 Anteilen, nicht ein Betrag von 4.000 €. Den Anteil der
Anschaffungskosten daraus zu rechnen ist genau die Arbeit, die dem Nutzer sonst bliebe —
und die er dann rundet. Die Stückzahl steht an jeder Bewegung, sonst ergäbe sich der
Bestand nach dem ersten Teilabgang nicht mehr.

## 6. Kopplung an die Festschreibung

AfA ist eine Abschlussbuchung zum Bilanzstichtag, kein laufender Geschäftsvorfall. Sie
wird nicht im Hintergrund gebucht. Auslöser ist die **jährliche** Festschreibung:

1. Vor der Jahres-Festschreibung prüft Buchfink, ob für jedes Anlagegut die fällige AfA
   gebucht ist.
2. Fehlende Buchungen werden mit Vorschau angezeigt und auf Freigabe erzeugt.
3. Erst danach kann das Jahr festgeschrieben werden.

Bei monatlicher oder quartalsweiser Festschreibung wird nicht geprüft – die AfA ist
dann nicht fällig.

## 7. Anlagenspiegel

Der Anlagenspiegel stellt für jede Position die Entwicklung dar: Anschaffungskosten am
Jahresanfang, Zugänge, Abgänge, Umbuchungen, kumulierte Abschreibungen, Buchwerte zum
Anfang und Ende. Für Kapitalgesellschaften ist er Bestandteil des Anhangs (§ 284 Abs. 3
HGB). Kleine Kapitalgesellschaften sind davon befreit – die Erleichterung steht in
**§ 288 Abs. 1 Nr. 1 HGB**, nicht in § 274a HGB, der nur § 268 Abs. 4/5/6 und § 274
betrifft.

Er ist keine zusätzliche Buchung, sondern eine Auswertung – aber eine, die nur
funktioniert, wenn Zugänge, Abgänge und kumulierte AfA je Anlagegut über die Jahre
erhalten bleiben. Das ist eine Anforderung an das Datenmodell, nicht an den Report:
**die Anlagenkartei muss jahresübergreifend geführt werden**, während das Journal pro
Geschäftsjahr organisiert ist.

## 8. Datenmodell (Skizze)

Je Anlagegut: Bezeichnung, Inventarnummer, Anlagekonto, Anschaffungsdatum,
Anschaffungskosten, Nutzungsdauer, AfA-Methode, Sonderabschreibungen, Abgangsdatum,
Abgangsart. Dazu eine Historie der AfA-Buchungen mit Verweis auf den Journaleintrag.

Der Bezug zwischen Anlagegut und Journal muss in beide Richtungen tragen: vom
Anlagegut zu seinen Buchungen und von der Buchung zurück zum Anlagegut.

## 9. Entscheidungen

Entschieden und umgesetzt:

- **Wertgrenzen und AfA-Sätze:** datierte Tabelle im Code
  (`accounting.AfAParametersFor`), nicht editierbare Stammdaten — wie bei den
  Steuerparametern. Eine Grenze ist keine Wahl des Nutzers, und ein fest
  verdrahteter Wert würde ein nachbearbeitetes altes Jahr still falsch rechnen.
  Die Grenzen binden auch die Eingabe: ein Sofortabzug über der GWG-Grenze und
  ein Sammelposten außerhalb des Korridors werden abgelehnt, beim Sammelposten
  je einzelnem Wirtschaftsgut und nicht auf die Summe des Postens. Je
  Wirtschaftsjahr entsteht genau ein Sammelposten; weitere Güter des Jahres
  kommen als Zugang hinein.
- **Nutzungsdauer-Katalog:** ausgeliefert, aber nur für die Fälle, die eindeutig
  sind (Pkw, Lkw, Büromöbel, Ladeneinbauten, Geschäfts- oder Firmenwert). Jeder
  Vorschlag ist überschreibbar; die AfA-Tabellen binden die Finanzverwaltung,
  nicht den Steuerpflichtigen.
- **Wechsel degressiv → linear:** Buchfink rechnet den Übergang selbst und weist
  ihn im Plan als eigene Zeile aus. Der Nutzer rechnet nichts nach.
- **Anlagenkartei über Geschäftsjahre:** eigene Tabellen (`fixed_assets`,
  `asset_movements`) außerhalb der Geschäftsjahres-Logik. Der Anlagenspiegel
  verlangt genau das.
- **Finanzanlagen:** in v1 aufgenommen, mit eigenen Regeln — keine planmäßige
  AfA, gemildertes Niederstwertprinzip (§ 253 Abs. 3 Satz 6 HGB), Wertaufholung
  nach § 253 Abs. 5 Satz 1 HGB, Teilabgang für Tranchen und Tilgungen und eigene
  Abgangskonten für Anteile, die § 8b Abs. 2 KStG bzw. § 3 Nr. 40 EStG
  unterliegen.
- **Anlagen im Bau:** die Fertigstellung ist eine eigene Aktion. Sie bucht um,
  setzt das Datum der Betriebsbereitschaft und startet damit die AfA.
- **Sonderabschreibung nach § 7g Abs. 5 EStG:** umgesetzt, mit eigenem
  Aufwandskonto, ohne Zeitanteil im Anschaffungsjahr, neben der linearen wie
  neben der degressiven AfA und mit der Restwertverteilung des § 7a Abs. 9 EStG
  nach dem Begünstigungszeitraum — sie geht dort auch der degressiven vor. Wie
  der Betrag über bis zu fünf Jahre verteilt wird, entscheidet der
  Steuerpflichtige; ist sie einmal gebucht, lässt sie sich nicht mehr
  umverteilen — das änderte ein abgeschlossenes Jahr.
- **Skonto und Rabatt auf Anlagen:** der Zahlungsflow erkennt die
  Anlagenrechnung und bucht die Minderung auf das Anlagekonto
  (§ 255 Abs. 1 Satz 3 HGB). Die Kartei bekommt sie mit Verweis auf die
  Zahlungsbuchung; die Steuerkorrektur nach § 17 Abs. 1 UStG bleibt unverändert.

Ebenfalls umgesetzt, mit den Grenzen, die dabei bewusst gezogen wurden:

- **Erhaltungsaufwand und laufende Erträge:** beide werden gebucht und mit dem
  Anlagegut verknüpft, ohne seinen Buchwert anzurühren — genau das unterscheidet
  den Erhaltungsaufwand von den nachträglichen Herstellungskosten und die
  Dividende vom Rückfluss der Anschaffungskosten. Ihre Bewegungen tragen null in
  beiden Wertspalten und erscheinen deshalb nicht im Anlagenspiegel. Das
  Aufwandskonto folgt aus dem Anlagekonto, das Ertragskonto aus der Art der
  Finanzanlage; die einbehaltene Kapitalertragsteuer mindert den Zufluss und
  nicht den Ertrag.
- **Fremdwährung:** § 256a HGB ist als Bewertung abgebildet, nicht als eigene
  Buchung. Der Anschaffungskurs folgt aus Fremdbetrag und
  Euro-Anschaffungskosten, der Stichtagskurs kommt vom Nutzer, und aus beiden
  entsteht ein Vorschlag. Nach oben deckelt ihn das Anschaffungskostenprinzip
  (§ 253 Abs. 1 Satz 1 HGB) — die Ausnahme des § 256a Satz 2 HGB für eine
  Restlaufzeit bis zu einem Jahr passt auf ein Anlagegut nicht, das dauernd dem
  Geschäftsbetrieb dienen soll. Gebucht wird über die Wege, die ihre Grenzen
  ohnehin prüfen: die außerplanmäßige Abschreibung und die Zuschreibung.
- **Stückzahlen bei Wertpapieren:** in Zehntausendsteln geführt, weil
  Fondsanteile in Bruchteilen gehalten werden. Der Teilabgang rechnet damit; wo
  keine Stückzahl geführt wird, bleibt der Betrag die Vorgabe.
- **Degressive AfA in älteren Zeiträumen:** die früheren Fassungen des
  § 7 Abs. 2 EStG stehen als eigene Zeitfenster in derselben Tabelle — 2009/2010
  und 2020 bis 2022 mit dem Zweieinhalbfachen und höchstens 25 %, das zweite bis
  vierte Quartal 2024 mit dem Zweifachen und höchstens 20 %, ab dem 01.07.2025
  mit dem Dreifachen und höchstens 30 %. Zwischen ihnen liegen Lücken, und die
  sind gewollt: 2023, das erste Quartal 2024 und das erste Halbjahr 2025 kannten
  keine degressive AfA. Der Faktor war einmal ein halber und steht deshalb in
  Promille.
- **Wertgrenzen vor 2018:** von 2010 bis 2017 endete der Sofortabzug bei 410 €
  und der Sammelposten begann bei 150 €. Ein Altbestand bleibt damit erklärbar;
  für Anschaffungen davor lehnt Buchfink die Einordnung ab, statt sie zu raten.

Bewusst nicht abgebildet:

- **Die Anrechnung der Kapitalertragsteuer.** Der einbehaltene Betrag wird
  erfasst und gebucht; die Anrechnung selbst ist Sache der Steuererklärung und
  nicht dieser Kartei.
- **Erhaltungsaufwand als Rückstellung** für unterlassene Instandhaltung
  (§ 249 Abs. 1 Satz 2 Nr. 1 HGB). Das ist ein Vorgang des Jahresabschlusses und
  gehört nicht an das einzelne Anlagegut.

## 10. Abhängigkeiten

- Der **Festschreibungs-Workflow** ruft vor der Jahressperre die AfA-Prüfung auf
  (`ensureDepreciationBooked` in `internal/wailsbridge/festschreibung_service.go`).
  Monats- und Quartalsfestschreibungen prüfen nicht — dort ist die AfA nicht fällig.
- Der **Zahlungsflow** fragt die Kartei über eine Schnittstelle mit zwei Methoden,
  bevor er ein Skonto bucht (`AssetRegister` in `internal/service/payment_service.go`).
  Er soll Anlagegüter nicht verwalten können, sondern nur erkennen, dass eine
  Rechnung eine war. Ist die Kartei nicht angeschlossen, bucht das Skonto wie
  bisher — eine fehlende Verdrahtung darf keine Zahlung scheitern lassen.
- Die **E-Bilanz** übernimmt den Anlagenspiegel als Kontennachweis
  (`SetAnlagenspiegelSource` in `internal/service/ebilanz_service.go`). Die
  Bilanz zeigt einen Buchwert; erst der Spiegel zeigt, woraus er entstanden ist.
  Scheitert seine Auswertung, entsteht die Instanz ohne den Block. Die
  Elementnamen folgen der vereinfachten Form, in der diese Datei schon den
  Kontennachweis führt, und sind vor der Übermittlung gegen die amtliche
  Taxonomie zu prüfen; die Zahlen darin stammen aus der Buchführung.

## 11. Quellen

Stand der Prüfung: 22.08.2026, Volltexte über gesetze-im-internet.de.

| Aussage im Dokument | Fundstelle | Link |
|---|---|---|
| GWG-Grenze 800 € (netto), Sofortabzug | § 6 Abs. 2 Satz 1 EStG | [estg/__6.html](https://www.gesetze-im-internet.de/estg/__6.html) |
| Verzeichnispflicht ab 250 €, Entbehrlichkeit bei Ersichtlichkeit aus der Buchführung | § 6 Abs. 2 Sätze 4 und 5 EStG | dito |
| Sammelposten 250 € bis 1.000 €, Auflösung über fünf Jahre, Wahlrecht einheitlich je Wirtschaftsjahr | § 6 Abs. 2a Sätze 1 bis 5 EStG | dito |
| Anschaffungskosten = Anschaffungspreis + Nebenkosten − Minderungen | § 255 Abs. 1 HGB | [hgb/__255.html](https://www.gesetze-im-internet.de/hgb/__255.html) |
| Lineare AfA über die betriebsgewöhnliche Nutzungsdauer | § 7 Abs. 1 Sätze 1 und 2 EStG | [estg/__7.html](https://www.gesetze-im-internet.de/estg/__7.html) |
| Zeitanteilige AfA ab dem Anschaffungsmonat (pro rata temporis) | § 7 Abs. 1 Satz 4 EStG | dito |
| Degressive AfA: höchstens das Dreifache des linearen Satzes, höchstens 30 %, Anschaffung nach dem 30.06.2025 und vor dem 01.01.2028 | § 7 Abs. 2 Sätze 1 und 2 EStG | dito |
| Übergang degressiv → linear zulässig | § 7 Abs. 3 EStG | dito |
| Sonderabschreibung bis 40 %, im Jahr der Anschaffung und den vier folgenden, **neben** der AfA nach § 7 Abs. 1 **oder Abs. 2** | § 7g Abs. 5 EStG | [estg/__7g.html](https://www.gesetze-im-internet.de/estg/__7g.html) |
| Nur abnutzbare **bewegliche** Wirtschaftsgüter des Anlagevermögens | § 7g Abs. 5 EStG | dito |
| Gewinngrenze des Vorjahres und fast ausschließlich betriebliche Nutzung im Jahr der Anschaffung und im folgenden | § 7g Abs. 6 Nr. 1 und 2 EStG | dito |
| Allgemeine Regel: neben Sonderabschreibungen AfA nach § 7 Abs. 1 oder 4 — von § 7g Abs. 5 EStG verdrängt | § 7a Abs. 4 EStG | [estg/__7a.html](https://www.gesetze-im-internet.de/estg/__7a.html) |
| Nach Ablauf des Begünstigungszeitraums AfA „nach dem Restwert und der Restnutzungsdauer" | § 7a Abs. 9 EStG | dito |
| Herstellungskosten sind Aufwendungen für die Erweiterung oder eine über den ursprünglichen Zustand hinausgehende wesentliche Verbesserung | § 255 Abs. 2 Satz 1 HGB | [hgb/__255.html](https://www.gesetze-im-internet.de/hgb/__255.html) |
| Anschaffungspreisminderungen sind abzusetzen (Skonto auf eine Anlage) | § 255 Abs. 1 Satz 3 HGB | dito |
| Vermögensgegenstände höchstens mit den Anschaffungskosten, vermindert um Abschreibungen | § 253 Abs. 1 Satz 1 HGB | [hgb/__253.html](https://www.gesetze-im-internet.de/hgb/__253.html) |
| Fremdwährungsposten zum Devisenkassamittelkurs am Abschlussstichtag; die Ausnahme gilt nur bei einer Restlaufzeit von höchstens einem Jahr | § 256a HGB | [hgb/__256a.html](https://www.gesetze-im-internet.de/hgb/__256a.html) |
| Berichtigung der Bemessungsgrundlage bei Skonto | § 17 Abs. 1 UStG | [ustg_1980/__17.html](https://www.gesetze-im-internet.de/ustg_1980/__17.html) |
| Außerplanmäßige Abschreibung bei voraussichtlich dauernder Wertminderung | § 253 Abs. 3 Satz 5 HGB | [hgb/__253.html](https://www.gesetze-im-internet.de/hgb/__253.html) |
| Anlagenspiegel als Anhangbestandteil | § 284 Abs. 3 HGB | [hgb/__284.html](https://www.gesetze-im-internet.de/hgb/__284.html) |
| Befreiung kleiner Kapitalgesellschaften vom Anlagenspiegel | § 288 Abs. 1 Nr. 1 HGB | [hgb/__288.html](https://www.gesetze-im-internet.de/hgb/__288.html) |

**Korrektur gegenüber einer früheren Fassung dieses Dokuments:** die Befreiung
kleiner Kapitalgesellschaften vom Anlagenspiegel wurde dort auf § 274a HGB
gestützt. Das ist falsch – § 274a HGB befreit von § 268 Abs. 4 Satz 2, § 268
Abs. 5 Satz 3, § 268 Abs. 6 und § 274 HGB, nicht von § 284 Abs. 3 HGB. Die
richtige Fundstelle ist § 288 Abs. 1 Nr. 1 HGB.

**Nicht am Volltext prüfbar:** die früheren Fassungen des § 7 Abs. 2 EStG.
gesetze-im-internet.de führt nur die geltende Fassung; die aufgehobenen Zeiträume der
degressiven AfA (2009/2010, 2020 bis 2022, das zweite bis vierte Quartal 2024) stehen
in `internal/accounting/afa.go` mit dem Änderungsgesetz, auf das sie zurückgehen. Nur
das laufende Fenster — nach dem 30.06.2025 und vor dem 01.01.2028, höchstens das
Dreifache des linearen Satzes und höchstens 30 % — ist am Volltext verifiziert. Wer ein
Altjahr nachbearbeitet, prüft den Satz an der damaligen Fassung nach.

**Nicht aus dem Gesetz, sondern Verwaltungsanweisung:** die AfA-Tabellen (BMF).
Sie binden die Finanzverwaltung, nicht den Steuerpflichtigen; eine abweichende,
begründete Nutzungsdauer ist zulässig. Deshalb muss der Katalog in Buchfink
überschreibbar bleiben.
