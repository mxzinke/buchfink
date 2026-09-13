# Buchfink – Beleg- & Buchungsflow

[Fachkonzepte](README.md) · [Dokumentation](../README.md)

Gesetzliche Grundlage: [Anforderungskatalog](../anforderungen/README.md), GOB-01 bis
GOB-06, BEL-01 bis BEL-09, UNV-01, UNV-02, RECH-02, RECH-03, RECH-06, RECH-07,
UST-01, UST-02, UST-05, BEW-08, BEW-12

Status: Fachkonzept; Belegweg und Handbuchung implementiert
Letzte Aktualisierung: 2026-08-22 (Belegkern und Steuerfall umgesetzt)
Kontenrahmen: DATEV SKR04 2026 (Art.-Nr. 11175)

> Alle Kontonummern in diesem Dokument sind gegen `internal/accounting/skr04_2026.json`
> geprüft, das aus `assets/DATEV-SKR04-BilrUg-2026.pdf` extrahiert wurde. Der Test
> `TestPostingGroupAccountsExistInSKR04` prüft bei jedem Build, dass jede Kontierung
> im Code auf ein existierendes, bebuchbares SKR04-Konto zeigt.
>
> **Alle Paragrafenangaben sind am 22.08.2026 gegen den Gesetzestext geprüft**; die
> Fundstellen stehen im Anforderungskatalog, die dabei gefundenen Korrekturen in
> [Abschnitt 17](#17-fundstellen).
>
> **Vorsicht bei SKR03-Nummern.** Eine frühere Fassung dieses Dokuments enthielt
> durchgehend SKR03-Konten. Die Nummern kollidieren: 1600 ist im SKR04 die *Kasse*
> und im SKR03 *Verbindlichkeiten aus LuL*, 8400 gibt es im SKR04 gar nicht, 4930
> sind *Erträge aus der Auflösung von Rückstellungen* statt Bürobedarf. Der
> Buchungskern lehnt Konten der Klasse 8 daher mit einem expliziten Hinweis ab.

> Abgleich vom 11. September 2026: Die frühere Umsetzungseinschätzung ist
> überholt. Maßgebliche Implementierung: `internal/service/posting_service.go`, `manual_entry.go` und `self_issued_receipt.go`.
> Dieses Dokument bewahrt die fachliche Entwurfsgrundlage. Den aktuellen Umfang
> und verbleibende Grenzen beschreibt der [Umsetzungsstand](../projekt/umsetzungsstand.md).

## 1. Leitgedanke

Ein Abstraktionslayer, der dem Nutzer verständlich ist: er denkt in **Belegen** und
**Rechnungen**, die Software übersetzt über Backend-Logik in die
korrekten **SOLL/HABEN-Buchungen**. Beide Seiten sind klar getrennt, aber
deterministisch verbunden.

**Kein „vorbereiten, nicht buchen".** Jeder erfasste Beleg wird sofort gebucht. Ob
dabei Geld fließt, entscheidet die Kontenseite: ist noch nicht bezahlt, entsteht eine
Verbindlichkeit bzw. Forderung auf dem Personenkonto des Geschäftspartners. Die
Zahlung ist ein späterer, separater Geschäftsvorfall.

Das folgt aus der GoBD, keine Bequemlichkeitsentscheidung: unbare
Geschäftsvorfälle sind zeitnah festzuhalten, bare Kassenvorgänge täglich. Ein
Datensatz, der schon erfasst ist, darf nicht mehr editierbar sein. Konsequenz für die
UI: es gibt keinen gebuchten Beleg, den man nachträglich ändern kann. Eine Korrektur
ist immer Storno plus Neuerfassung.

## 2. Scope-Grenzen

**Nur Sollversteuerung.** Der gesamte Flow „Rechnung erfassen → sofort buchen →
Zahlung später" setzt die Berechnung der Steuer nach vereinbarten Entgelten
(§ 16 Abs. 1 Satz 1 UStG) voraus. Bei Istversteuerung entsteht die Steuer erst mit
Ablauf des Voranmeldungszeitraums, in dem das Entgelt vereinnahmt wurde
(§ 13 Abs. 1 Nr. 1 Buchst. b UStG), und die Buchungen sähen anders aus. Die
Gestattung ist an keine Rechtsform gebunden: § 20 Satz 1 Nr. 1 UStG stellt allein
auf einen Gesamtumsatz von höchstens 800.000 € im Vorjahr ab, steht also auch
einer GmbH offen. Buchfink fragt die Versteuerungsart im Setup ab, und der
Buchungskern weist jede Buchung ab, solange sie nicht auf Sollversteuerung steht –
statt sie stillschweigend falsch zu behandeln. Die Einstellung war lange nur ein
Feld, das niemand prüfte; das ist der gefährlichere Zustand von beiden.

**E-Rechnung ist eine Pflicht, die seit dem 01.01.2025 gilt, keine
Scope-Grenze** und von den Übergangsregelungen des § 27 Abs. 38 UStG nicht
erfasst ist – die gelten nur für das Ausstellen. Der Empfang strukturierter
Eingangsrechnungen ist inzwischen umgesetzt: ZUGFeRD, Factur-X und XRechnung
werden erkannt, CII und UBL gelesen und gegen das Regelwerk geprüft
(`internal/einvoice/`). Ausgestellt wird bisher nur als ZUGFeRD-PDF, die reine
XRechnung fehlt; siehe [E-Rechnung](e-rechnung.md).

**Steuern und Auswertungen bleiben außen vor.** USt-Voranmeldung, USt-Erklärung, ZM
und der Jahresabschluss (Saldenvortrag, GuV-Abrechnung, Bilanzierung) sind nicht Teil
des Beleg- und Zahlungsflows. Sie brauchen eigene Eingabemasken und eigene Logik. Der
Buchungskern liefert die Grundlage dafür – jede Buchung hat Steuerschlüssel,
Bemessungsgrundlage und Steuerfall –, die Auswertung selbst wird später ergänzt.

**Weiter außen vor in v1:** Warenbestand und Inventur, Lohnbuchhaltung, EÜR,
Rechnungsabgrenzung als automatische Buchung (der Leistungszeitraum wird aber ab Tag 1
erfasst, siehe 4).

## 3. Was eine Buchung vollständig macht

Eine Buchung lässt sich nicht aus „Kategorie plus Betrag" ableiten. Der vollständige
Input sind acht Angaben:

| # | Angabe | Warum sie nötig ist |
|---|---|---|
| 1 | **Richtung** (Eingang / Ausgang) | bestimmt Aufwands- oder Ertragsseite |
| 2 | **Geschäftspartner** | Personenkonto, offener Posten, USt-IdNr. für EU-Fälle |
| 3 | **Belegdatum** | Entstehung der Umsatzsteuer, Vorsteuerabzug |
| 4 | **Leistungsdatum / -zeitraum** | Periodenabgrenzung; der Leistungs*zeitpunkt* ist Pflichtangabe nach § 14 Abs. 4 Nr. 6 UStG |
| 5 | **Nettobeträge je Steuersatz** | ein Beleg kann 19 % und 7 % enthalten |
| 6 | **Steuerfall** | entscheidet über Vorsteuer- und Umsatzsteuerkonto |
| 7 | **Fachliche Gruppe** | Aufwands- bzw. Ertragskonto |
| 8 | **Zahlungszeitpunkt + Zahlungsmittel** | Personenkonto oder Zahlungsmittelkonto |

Punkt 6 ist die Angabe, die am leichtesten vergessen wird und ohne die das Mapping das
Vorsteuerkonto nicht bestimmen kann. Punkt 8 sind zwei unabhängige Dimensionen: *wann*
gezahlt wird (sofort oder auf Ziel) und *womit* (Kasse, Bankkonto, Kreditkarte).
„Bar" heißt Kasse 1600, nicht Bank 1800.

## 4. Die vier Daten

Das System führt vier Datumsfelder getrennt. Ihre Verwechslung ist die häufigste
Ursache für falsche Perioden und falsche Voranmeldungen.

| Feld | Bedeutung | Wofür maßgeblich |
|---|---|---|
| **Belegdatum** | Rechnungsdatum | Entstehung der USt, Vorsteuerabzug |
| **Leistungsdatum / -zeitraum** | wann geleistet wurde | Periodenabgrenzung; § 14 Abs. 4 Nr. 6 UStG verlangt den Zeitpunkt, der Zeitraum ist Buchfinks eigene Anforderung für die Abgrenzung |
| **Buchungsdatum** | Zuordnung zur Periode | Geschäftsjahr, Festschreibung |
| **Valuta** | Wertstellung der Bank | nur bei Zahlungsbuchungen |

Rechnung im Dezember, Leistung im Januar: das ist ein Fall für die
Rechnungsabgrenzung (1900 aktiv / 3900 passiv, § 250 HGB). Buchfink bucht die
Abgrenzung in v1 nicht automatisch, erfasst den Leistungszeitraum aber vollständig –
sonst wäre der Fall später nicht mehr rekonstruierbar.

## 5. Steuerfälle

| Steuerfall | Richtung | Buchung |
|---|---|---|
| **Inland, steuerpflichtig** | beide | eine Steuerzeile: Vorsteuer 1406/1401 bzw. Umsatzsteuer 3806/3801 |
| **§ 13b UStG (Reverse Charge)** | Eingang | **zwei** Steuerzeilen: Vorsteuer 1407 *und* Umsatzsteuer 3837 |
| **Innergemeinschaftlicher Erwerb** | Eingang | **zwei** Steuerzeilen: Vorsteuer 1404 *und* Umsatzsteuer 3804 |
| **Innergem. Lieferung** (§ 4 Nr. 1b) | Ausgang | keine Steuerzeile, Erlöse auf 4125 |
| **Ausfuhr Drittland** (§ 4 Nr. 1a) | Ausgang | keine Steuerzeile, Erlöse auf 4120 |
| **§ 13b beim Empfänger** | Ausgang | keine Steuerzeile, Erlöse auf 4337 |
| **Steuerfrei** (§ 4 UStG) | beide | keine Steuerzeile, Erlöse auf 4150 |
| **Nicht steuerbar** | beide | keine Steuerzeile |

**Reverse Charge ist kein Randfall.** Eine GmbH, die bei AWS, Google, Stripe oder
Hetzner einkauft, hat ihn ständig. Der Vorgang erzeugt vier Zeilen, und an den
Lieferanten geht nur der Nettobetrag – die Steuer schuldet man selbst und zieht sie im
selben Atemzug als Vorsteuer ab.

Die Stammdaten müssen den Steuerfall abbilden. Ohne USt-IdNr. des Empfängers lehnt
Buchfink eine innergemeinschaftliche Lieferung ab (§ 6a Abs. 1 Nr. 4 UStG); ein
deutscher Kunde kann keine bekommen, ein EU-Kunde keine Ausfuhrlieferung.

## 6. Kontierung: fachliche Gruppe → SKR04

Der Nutzer wählt eine Gruppe, das Backend mappt deterministisch. Keine Lernfunktion,
keine Heuristik. Das Konto richtet sich nach Gruppe **plus Steuerfall plus Steuersatz**:

```
Gruppe „Fremdleistungen"
  + Inland, 19 %          →  5906  Fremdleistungen 19 % Vorsteuer
  + Inland, 7 %           →  5908  Fremdleistungen 7 % Vorsteuer
  + § 13b UStG            →  5909  Fremdleistungen ohne Vorsteuer (§ 13b)

Gruppe „Erlöse"
  + Inland, 19 %          →  4400  Erlöse 19 % USt
  + Inland, 7 %           →  4300  Erlöse 7 % USt
  + innergem. Lieferung   →  4125  Steuerfreie innergem. Lieferungen § 4 Nr. 1b
  + Ausfuhr               →  4120  Steuerfreie Umsätze § 4 Nr. 1a
  + § 13b beim Empfänger  →  4337  Erlöse, Steuerschuld beim Leistungsempfänger
```

Jede Buchung speichert die **Version des Regelwerks** mit. Ändert sich das Mapping
später, bleiben Altbuchungen erklärbar – das verlangt die Verfahrensdokumentation.

Steuerkonten (1400er, 3800er) dürfen ausschließlich über die Steuerautomatik bebucht
werden. Eine handgeschriebene Zeile auf 1406 würde die Voranmeldung vom Journal
entkoppeln, und der Buchungskern weist sie ab.

### Bebuchbarkeit

Der SKR04-Katalog enthält 1.855 Einträge, davon 243 **Bereichskonten** wie
`4400-4409 Erlöse 19 % USt`. Ein Bereich ist eine Kurzschreibweise für zehn nutzbare
Konten, kein Konto: gebucht wird auf 4400 oder 4407, nie auf die Zeichenkette
„4400-4409". Ebenfalls gesperrt sind reservierte Konten und die gesamte
**Kontenklasse 8**, die im SKR04 für künftige DATEV-Verwendung freigehalten wird.

## 7. Der Flow

```
Beleg erfassen ──► sofort buchen ──► bezahlt?
                                      │
                                      ├─ nein → offener Posten auf dem Personenkonto
                                      │          └─► Zahlung zuordnen ──► OP schließt
                                      └─ ja   → direkt gegen Kasse / Bank / Karte
```

### 7.1 Erfassen und buchen

| Fall | Buchung |
|---|---|
| Eingangsbeleg, auf Ziel | SOLL Aufwand + SOLL Vorsteuer · HABEN Kreditorenkonto |
| Eingangsbeleg, sofort bezahlt | SOLL Aufwand + SOLL Vorsteuer · HABEN Kasse/Bank/Karte |
| Ausgangsrechnung | SOLL Debitorenkonto · HABEN Erlös + HABEN Umsatzsteuer |

Der Gegenbetrag ergibt sich aus dem Ausgleich der übrigen Zeilen. Das deckt jeden
Steuerfall ohne Sonderfall ab: bei einer Inlandsrechnung sind es netto plus Vorsteuer,
bei Reverse Charge nur netto, weil sich Vorsteuer- und Umsatzsteuerzeile aufheben.

### 7.2 Zahlung zuordnen

Zuordnung ist ein eigener Datensatz, keine Markierung am Bankumsatz. Damit sind alle
drei Fälle abgedeckt:

- **Eine Zahlung → ein Beleg:** klassischer Ausgleich.
- **Mehrere Zahlungen → ein Beleg:** Teilzahlungen und Raten. Der Beleg bleibt offen,
  bis die Summe erreicht ist.
- **Eine Zahlung → mehrere Belege:** Sammelüberweisung. Der Betrag wird aufgeteilt.

Der Status ergibt sich aus dem Saldo und wird nicht gespeichert: `bezahlt`,
`teilbezahlt`, `offen`.

Ist die Zahlung ein Bankumsatz, muss die Summe der Zuordnungen exakt dem Kontoauszug
entsprechen. Ein vertipptes Skonto wird so zur Fehlermeldung statt zur stillen
Falschbuchung.

**Offene Posten kennen keinen Jahreswechsel.** Die Rechnung vom 20. Dezember wird im
Januar bezahlt — das ist der Normalfall, nicht die Ausnahme, und beide Hälften des
Vorgangs liegen planmäßig in verschiedenen Wirtschaftsjahren. § 252 Abs. 1 Nr. 5 HGB
verlangt genau das: Aufwendungen und Erträge sind „unabhängig von den Zeitpunkten der
entsprechenden Zahlungen" zu erfassen. Die Forderung und der Ertrag entstehen mit der
Leistung, die Zahlung wird gebucht, wann sie fließt.

Das gilt **unabhängig von der Besteuerungsart**. Die Istversteuerung nach § 20 UStG
verschiebt allein den Zeitpunkt, zu dem die *Umsatzsteuer* entsteht
(§ 13 Abs. 1 Nr. 1 Buchst. b UStG); bei einem bilanzierenden Unternehmen bleiben
Forderung und Ertrag im Jahr der Leistung, und die Steuer wartet solange auf einem
Konto „Umsatzsteuer nicht fällig". Ein **Rechnungsabgrenzungsposten** entsteht dabei
nicht: § 250 HGB setzt eine Ausgabe oder Einnahme **vor** dem Stichtag voraus, die
Aufwand oder Ertrag für eine Zeit **danach** ist — also den umgekehrten Fall. Hier ist
die Leistung vor dem Stichtag und die Zahlung danach, und das ist eine Forderung.

Daraus folgt für die Liste der offenen Posten:

- Sie zeigt die Posten **aller Jahre bis zum eingestellten**, nicht nur die des
  laufenden. Sonst ließe sich die Dezemberrechnung im Januar in keiner Auswahl mehr
  finden und nie mehr ausgleichen.
- Der ausgeglichene Betrag zählt über die **ganze Historie**, nicht je Jahr. Zählte er
  je Jahr, stünde die im Januar bezahlte Rechnung im Blick auf das Vorjahr weiter offen
  — und der Zuordnungsdialog ließe sie ein zweites Mal bezahlen.
- Zuordnungen, deren **Zahlungsbuchung storniert** wurde, zählen nicht mit. Die
  Generalumkehr lässt die Zuordnungszeilen stehen; ohne diese Regel bliebe der Posten
  ausgeglichen, ohne dass jemand gezahlt hätte.

**Was das ausdrücklich nicht ist:** eine Stichtagsbetrachtung. „Welche Posten waren am
31.12. offen" ist eine andere Frage, und sie braucht eine Datumsgrenze statt einer
Jahreszahl. Die Bilanz ist davon unberührt — die Position *Forderungen aus Lieferungen
und Leistungen* kommt aus den Salden der Personenkonten des Jahres, nicht aus dieser
Liste. Eine stichtagsbezogene OP-Liste gibt es noch nicht; sie gehört zum
Jahresabschluss und ist dort vermerkt.

### 7.3 Zahlungsdifferenzen

Der Zahlbetrag stimmt aus mehreren Gründen nicht mit dem Belegbetrag überein. Ohne
klare Behandlung bleiben offene Posten mit drei Cent ewig stehen, und irgendwann
räumt jemand sie mit einer Falschbuchung weg.

| Differenz | Behandlung |
|---|---|
| **Skonto** | mindert Entgelt **und** Steuer (§ 17 UStG): 5736/5731/5730 erhalten, 4736/4731/4734 gewährt, dazu die Steuerkorrektur |
| **Bankgebühr** | eigener Aufwand auf 6855 Nebenkosten des Geldverkehrs |
| **Rundungsdifferenz** | Ausbuchung über 4830 bzw. 6300 |
| **Kursdifferenz** | realisierter Kursgewinn/-verlust beim Ausgleich, § 256a HGB |
| **Überzahlung** | der Saldo des Personenkontos dreht sich; das ist ein Habensaldo beim Debitor, keine negative Forderung |

Beispiel Skonto: 2 % auf 1.190,00 € brutto sind 23,80 € – 20,00 € netto und 3,80 €
Steuer. Nur den Nettoteil zu buchen ließe die Vorsteuer um 3,80 € zu hoch stehen.

Die Steuerkorrektur folgt dem **Steuerfall der ursprünglichen Buchung**, nicht dem
Steuersatz allein. Das ist der Unterschied zwischen richtig und plausibel, kein
Detail:

- Nur beim **steuerpflichtigen Inlandsumsatz** steckt die Steuer im offenen Betrag.
  Dort wird das Skonto in Entgelt und Steuer zerlegt (§ 17 Abs. 1 Satz 1 und 2 UStG).
- Bei **§ 13b** und beim **innergemeinschaftlichen Erwerb** ist die Rechnung netto
  ausgestellt. Das ganze Skonto ist Bemessungsgrundlage, und zu berichtigen sind
  **beide** Steuerzeilen – die geschuldete Steuer und die abgezogene Vorsteuer
  (§ 17 Abs. 1 Satz 5 UStG). Dass sie sich im Ergebnis ausgleichen, ist ausdrücklich
  kein Grund, sie wegzulassen: UStAE 17.1 Abs. 3 verlangt die Berichtigung „auch dann,
  wenn sich die Berichtigung der Steuer und die Berichtigung des Vorsteuerabzugs im
  Ergebnis ausgleichen". Ohne sie stehen zwei Kennzahlen der Voranmeldung zu hoch.
- Bei **steuerfreien, nicht steuerbaren und nullbesteuerten** Umsätzen gibt es keine
  Steuer zu berichtigen; das Skonto läuft vollständig über 5730 bzw. 4734.

Die Steuerzeilen der Korrektur kommen deshalb aus derselben Steuerautomatik wie die
der ursprünglichen Buchung, nicht aus einer zweiten Tabelle von Kontonummern im
Zahlungsflow. Berichtigt wird nach § 17 Abs. 1 Satz 8 UStG im Zeitraum der Änderung,
also mit der Zahlung – nie rückwirkend an der Rechnung.

### 7.4 Bankumsatz ohne Beleg

Nicht jeder Umsatz hat einen Beleg: Zinsen, Kontoführung, Privatentnahmen,
Umbuchungen zwischen eigenen Konten. Diese werden direkt gegen ein Konto gebucht. Die
Bankseite kommt aus dem Kontoauszug, die Richtung ist damit nicht vertippbar.

Der CAMT-Import schlägt **kein Konto vor**. Aus dem Verwendungszweck ein Aufwandskonto
zu raten wäre eine unprüfbare Heuristik vor den Buchungsregeln – und genau die
Entscheidung, die der Nutzer treffen und verantworten muss.

## 8. Personenkonten und offene Posten

**Es wird nie auf ein Sammelkonto gebucht.** Jeder Geschäftspartner bekommt ein echtes
Personenkonto aus den DATEV-Bereichen, und der offene Posten entsteht dort:

| Bereich | Bedeutung | Saldenvortrag |
|---|---|---|
| 10000–69999 | Debitoren (Kunden) | 9008 |
| 70000–99999 | Kreditoren (Lieferanten) | 9009 |

1200 (Forderungen aus LuL) und 3300 (Verbindlichkeiten aus LuL) sind
**Bilanzpositionen**, keine Buchungsziele. Das folgt aus § 266 HGB, nicht aus
einer Designentscheidung: die Bilanz zeigt eine Zeile „Forderungen aus Lieferungen und Leistungen",
nicht vierhundert Kundenzeilen. Der Betrag dieser Zeile entsteht durch Verdichtung der
Personenkonten.

Der Buchungskern weist eine Zeile auf 1200 oder 3300 deshalb ab, so wie er auch
Steuerkonten abweist. Wäre beides erlaubt, gäbe es zwei Wahrheiten für dieselbe Zahl:
eine direkt gebuchte Forderung stünde in der Bilanz, aber in keiner OPOS-Liste, und die
Differenz fiele erst auf, wenn jemand nachrechnet, warum das Kundenkonto nicht zur
Bilanzposition passt.

In der Kontenübersicht ist die Verdichtung sichtbar gemacht – 3300 zeigt den Hinweis,
aus wie vielen Personenkonten der Betrag stammt. Nummern werden nie wiederverwendet;
eine alte Buchung muss zuordenbar bleiben. Echte Nummernkreise sind außerdem
Voraussetzung für einen späteren DATEV-Export an den Steuerberater.

## 9. GoBD: Unveränderbarkeit

### Hash-Chain

Jede Buchung enthält den SHA256-Hash der vorangehenden. Der Hash deckt **alle**
buchungsrelevanten Felder ab, einschließlich Buchungstext, Steuerbeträgen und aller
Buchungszeilen. Die Serialisierung ist längenpräfigiert, damit kein Feldinhalt eine
Feldgrenze vortäuschen kann. Nicht abgedeckt ist der Dateipfad des Belegs – ein
verschobener Datenordner darf die Kette nicht brechen; der Dateiinhalt hängt über
seinen eigenen Hash daran.

### Nummernkreise

Buchungsnummern, Eingangsbelege und Ausgangsrechnungen haben je einen eigenen Zähler
pro Geschäftsjahr. Nummernvergabe, Kettenkopf und Insert laufen in einer Transaktion:
eine gescheiterte Buchung verbraucht keine Nummer und hinterlässt keine Lücke.
Rechnungsnummern sind nach § 14 Abs. 4 Nr. 4 UStG einmalig und fortlaufend.

### Korrektur: Generalumkehr statt Seitentausch

Ein Storno per Seitentausch – Soll und Haben vertauscht – ergibt zwar einen Saldo von
null, bläht aber die Verkehrszahlen auf: ein korrigierter Aufwand von 1.000 € steht
danach mit 1.000 € Soll *und* 1.000 € Haben im Konto. Die Summen- und Saldenliste
zeigt Umsätze, die es nie gab, und die aus Umsätzen abgeleiteten Steuerkennzahlen
werden falsch.

Buchfink storniert daher per **Generalumkehr**: gleiche Konten, gleiche Seiten,
negativer Betrag mit Kennzeichen. Die Verkehrszahlen gehen auf null zurück. Das ist
auch das Verfahren, das DATEV als „GU" führt.

Die Stornobuchung wird auf den Korrekturtag datiert, nie zurück in die
Ursprungsperiode. Eine Buchung kann genau einmal storniert werden, und eine
Generalumkehr lässt sich nicht ihrerseits stornieren.

### Festschreibung

Vor dem Stichtag einer festgeschriebenen Periode sind keine neuen Buchungen mehr
möglich; Korrekturen laufen über die Generalumkehr in der offenen Periode. Jede
Festschreibung verankert den Kettenkopf zusätzlich mit einem RFC-3161-Zeitstempel.

### Exakte Beträge

Alle Beträge sind ganzzahlige Cent. Damit ist „Summe Soll = Summe Haben" eine exakte
Prüfung statt eines Toleranzvergleichs. Gerundet wird an genau einer Stelle:
kaufmännisch, **einmal je Steuersatzgruppe**. Positionsweise Rundung mit anschließender
Summierung ergäbe einen Gesamtbetrag, der ein bis zwei Cent neben der Steuer auf die
Rechnungssumme liegt – und genau diese Differenz hinterlässt später den offenen
Posten, der nie zugeht.

## 10. Durchgespielte Geschäftsvorfälle

Siehe [Buchungsbeispiele](buchungsbeispiele.md).

## 11. Anlagenverwaltung

> Überblick. Die Ausarbeitung steht in
> [Anlagenverwaltung](anlagenverwaltung.md), die
> gesetzlichen Anforderungen im Anforderungskatalog unter BEW-03, BEW-04 und
> BEW-05.

Vor der Abschreibung steht die Entscheidung zwischen Sofortabzug (0670 · 6260),
Sammelposten (0675 · 6264) und Aktivierung (Anlagekonto · 6220 oder 6222). Für den
Buchungsflow zählt daran vor allem die Kopplung an die Festschreibung: AfA und
Rückstellungen sind Abschlussbuchungen zum Bilanzstichtag, keine laufenden
Geschäftsvorfälle, und werden nicht im Hintergrund gebucht. Auslöser ist die
**jährliche** Festschreibung – sie prüft vor dem Sperren, ob für alle Anlagegüter die
fällige AfA gebucht ist, zeigt Fehlendes an und lässt es mit Vorschau und Freigabe
erzeugen. Bei monatlicher oder quartalsweiser Festschreibung wird nicht geprüft.

## 12. Anzahlungen

> Überblick. Die Ausarbeitung steht in
> [Anzahlungen](anzahlungen.md), die gesetzlichen
> Anforderungen im Anforderungskatalog unter UST-02 und RECH-10.

Bei Anzahlungen entsteht die Umsatzsteuer mit der Vereinnahmung – auch bei
Sollversteuerung. Das ist ein eigener Buchungsweg, kein Detail: die erhaltene
Anzahlung wird über **3272** Erhaltene, versteuerte Anzahlungen 19 % USt gebucht,
die geleistete über das Anzahlungskonto der jeweiligen Bilanzposition (etwa **1180**
Geleistete Anzahlungen auf Vorräte), und die Schlussrechnung setzt die Anzahlungen
ab; nur die Differenz wird zum offenen Posten.

Der Rechnungsverbund fasst Abschläge und Schlussrechnung als eigener Entity zusammen
und stellt den Gesamtfortschritt dar. Ohne die Anzahlungskonten und die Verrechnung in
der Schlussrechnung wäre er nur eine UI-Gruppierung.

## 13. Nicht abziehbare Betriebsausgaben

Einige Aufwendungen sind handelsrechtlich Aufwand, steuerlich aber nur teilweise
abziehbar. Sie brauchen getrennte Konten, sonst ist die Steuerbilanz falsch:

| Fall | Konten |
|---|---|
| **Bewirtung** – 70 % abziehbar, 30 % nicht (§ 4 Abs. 5 Satz 1 Nr. 2 EStG); die Vorsteuer bleibt trotzdem voll abziehbar, § 15 Abs. 1a Satz 2 UStG nimmt Bewirtungsaufwendungen vom Vorsteuerausschluss ausdrücklich aus | **6640** abziehbar, **6644** nicht abzugsfähig |
| **Geschenke** – abziehbar, solange die Zuwendungen an einen Empfänger im Wirtschaftsjahr **50 €** nicht übersteigen (§ 4 Abs. 5 Satz 1 Nr. 1 Satz 2 EStG); darüber weder Aufwand noch Vorsteuer (§ 15 Abs. 1a Satz 1 UStG) | 6610 abzugsfähig, 6620 nicht abzugsfähig |

Der abziehbare Anteil wird gebucht, nicht nur berechnet: eine Bewirtungsposition
erzeugt zwei Aufwandszeilen, den abziehbaren Teil auf 6640 und den Rest als
Differenz auf 6644 – nie zweimal gerundet, sonst summierten sich die beiden an
einem Cent vorbei. Die Bemessungsgrundlage der Vorsteuerzeile bleibt der volle
Nettobetrag.

Beide Grenzen sind wie die AfA-Wertgrenzen nach Gültigkeitszeitraum geschlüsselt:
die Geschenkegrenze lag bis einschließlich der vor dem 01.01.2024 beginnenden
Wirtschaftsjahre bei 35 €, die Kleinbetragsgrenze des § 33 UStDV bis 2016 bei
150 €. Ein fest verdrahteter Wert bucht ein nachbearbeitetes Altjahr still falsch.
Sie stehen dabei in einer datierten Tabelle im Code
(`internal/accounting/tax_params.go`), die `PostingRuleVersion` mitabdeckt,
**nicht** in editierbaren Stammdaten: diese Werte ändert der Gesetzgeber, nicht
der Nutzer, und editierbar zu machen, was nicht zur Wahl steht, lädt zum
Falschbuchen ein.

Die Bewirtung hat zusätzlich eine **Aufzeichnungspflicht**, die keine Buchung ist:
Ort, Tag, Teilnehmer und Anlass der Bewirtung sowie die Höhe der Aufwendungen sind
schriftlich festzuhalten; bei einer Gaststätte genügen Anlass und Teilnehmer, die
Rechnung ist beizufügen (§ 4 Abs. 5 Satz 1 Nr. 2 Sätze 2 und 3 EStG). Ohne sie ist
der Abzug auch für die 70 % verloren.

Die Angaben stehen an der **Buchung**, nicht am Beleg. Das ist eine bewusste
Abweichung von der naheliegenden Ablage: der Beleg-Hash deckt ausschließlich die
Dateiliste ab (siehe [Belegmodell](belege.md)), eine Teilnehmerliste am Beleg wäre also von
keiner Prüfsumme gedeckt und nachträglich änderbar. Eine Aufzeichnung, von der der
Betriebsausgabenabzug abhängt, gehört unter die Hash-Chain.

## 14. Eröffnungsbilanz & Stammkapital

Die Gründung einer Kapitalgesellschaft erfordert besondere Buchungssätze, die Buchfink
als geführten Workflow anbietet. Rechtsgrundlagen: § 272 Abs. 1 HGB (offener Abzug
nicht eingeforderter Einlagen vom gezeichneten Kapital) und § 46 Nr. 2 GmbHG – die
Einforderung der Einlagen unterliegt der Bestimmung der Gesellschafter.

Das Eröffnungsbilanzkonto ist im SKR04 **9000 Saldenvorträge, Sachkonten**; für
Personenkonten gibt es 9008 (Debitoren) und 9009 (Kreditoren). Nach allen
Eröffnungsbuchungen muss es den Saldo null aufweisen.

### 14.1 Volleinzahlung

25.000 € Stammkapital, voll eingezahlt.

| # | Buchungssatz |
|---|---|
| 1 | **9000** Saldenvorträge an **2900** Gezeichnetes Kapital · 25.000,00 |
| 2 | **1800** Bank an **9000** Saldenvorträge · 25.000,00 |

### 14.2 Teileinzahlung

25.000 € Stammkapital, 12.500 € eingezahlt. Der Rest ist nicht eingeforderte
ausstehende Einlage und nach § 272 Abs. 1 Satz 3 HGB offen vom gezeichneten Kapital
abzusetzen.

| # | Buchungssatz |
|---|---|
| 1 | **9000** an **2900** Gezeichnetes Kapital · 25.000,00 |
| 2 | **1800** Bank an **9000** · 12.500,00 |
| 3 | **2910** Ausstehende Einlagen, nicht eingefordert an **9000** · 12.500,00 |

Bilanzausweis: Gezeichnetes Kapital 25.000 €, abzüglich nicht eingeforderter Einlagen
12.500 €, eingefordertes Kapital 12.500 €.

### 14.3 Einfordern und Einzahlung

| # | Buchungssatz |
|---|---|
| Einfordern (Gesellschafterbeschluss) | **1298** Ausstehende Einlagen, eingefordert an **2910** · 12.500,00 |
| Einzahlung | **1800** Bank an **1298** · 12.500,00 |

Der Betrag wechselt von 2910 (Passivseite, Kapitalkorrektur) auf 1298 (Aktivseite,
Forderung). Das gezeichnete Kapital bleibt unberührt.

### 14.4 Kontenübersicht

| Bedeutung | SKR04 | Bilanzseite |
|---|---|---|
| Gezeichnetes Kapital | 2900 | Passiva (Eigenkapital) |
| Ausstehende Einlagen, **nicht** eingefordert | 2910 | Passiva (offener Abzug vom Kapital) |
| Ausstehende Einlagen, eingefordert | 1298 | Aktiva (Forderungen) |
| Saldenvorträge Sachkonten | 9000 | Hilfskonto (Saldo null) |

Der Wizard ist nur verfügbar, solange keine Buchungen vorliegen. Zum Abschluss wird die
Eröffnungsbilanz als PDF erzeugt – wie die Ausgangsrechnungen über Typst.

## 15. Beleg- & Dateiverwaltung

Siehe [Beleg- und Dateiverwaltung](belege.md).

## 16. Entscheidungen

| Thema | Entscheidung |
|---|---|
| **Buchungsmodell** | Kopf + n Zeilen mit harter Invariante Summe Soll = Summe Haben |
| **Beträge** | ganzzahlige Cent, Rundung einmal je Steuersatzgruppe |
| **Storno** | Generalumkehr, nicht Seitentausch |
| **Personenkonten** | echte DATEV-Nummernkreise 10000–69999 / 70000–99999; 1200 und 3300 sind Bilanzpositionen und keine Buchungsziele |
| **Reverse Charge** | in v1 enthalten, kein Randfall |
| **Kontierung** | deterministisch, keine Lernfunktion, Regelwerk versioniert |
| **Beleg** | eigene Entität mit 1..n Dateien je Rolle; Beleg-Hash über die geordnete Dateiliste, mit der Buchung versiegelt; verbindlich, nicht optional |
| **E-Rechnung** | Teil des Belegflows, kein Zusatzmodul; gebucht wird immer aus dem strukturierten Teil |
| **Rechnungs-PDF** | Typst als WebAssembly im Prozess; PDF/A-3b mit eingebettetem XML in einem Schritt |
| **Versteuerung** | nur Sollversteuerung; Istversteuerung wird im Buchungskern abgewiesen |
| **Kleinunternehmer** | § 19 UStG wird für den eigenen Mandanten nicht unterstützt (Zielgruppe sind bilanzierende Kapitalgesellschaften); als Eigenschaft eines Lieferanten bleibt der Fall relevant |
| **Fachlogik** | steht im Backend; die Oberfläche sammelt ein, zeigt an und rechnet nichts nach |
| **Nullsteuersatz** | eigener Steuerfall (§ 12 Abs. 3 UStG), Erlöse auf 4290; der Steuerfall steht an der Buchung |
| **CAMT-Import** | schlägt keine Konten vor |
| **Warenbestand** | out of scope für v1 |
| **Steuern & Abschluss** | eigene Masken, nicht Teil des Belegflows |

### Offene Punkte

Jeder hat ein eigenes Anforderungsdokument, damit er für sich angegangen werden kann:

| Thema | Dokument | Was noch fehlt |
|---|---|---|
| **E-Rechnung** | [E-Rechnung](e-rechnung.md) | **Kein Konzeptpunkt, sondern geltendes Recht.** Der Empfang einer E-Rechnung ist seit dem 01.01.2025 Pflicht und hat nie eine Übergangsfrist gehabt; Buchfink kann ihn nicht. Der Vorsteuerabzug hängt am strukturierten Teil. Das **Belegmodell** dafür steht in [Belegmodell](belege.md) – E-Rechnung ist Teil des Belegflows, kein Anbau. Offen bleibt das Einlesen und die Validierung. |
| **Rechnungsabgrenzung** | [Rechnungsabgrenzung](rechnungsabgrenzung.md) | Der Leistungszeitraum wird erfasst, die Abgrenzungsbuchung auf 1900 / 3900 fehlt. Kleinster Punkt, hängt im Kern an einer Entscheidung. |
| **Anzahlungen & Rechnungsverbund** | [Anzahlungen](anzahlungen.md) | Die Steuer entsteht mit der Vereinnahmung, nicht mit der Rechnung. Die Schlussrechnung muss die Anzahlungen absetzen, sonst greift § 14c Abs. 1 UStG. |
| **Anlagenverwaltung** | [Anlagenverwaltung](anlagenverwaltung.md) | Größter Block: GWG und Sammelposten, AfA-Methoden, Abgang, Anlagenspiegel. |
| **DATEV-Export** | [DATEV-Export](datev-export.md) | Keine buchhalterische Entscheidung offen, aber eine architektonische: das n-Zeilen-Modell trifft auf DATEVs Konto/Gegenkonto. |

Die E-Rechnung steht bewusst oben. Die anderen vier beschreiben Funktionen, die
fehlen; sie beschreibt eine Pflicht, die läuft.

Die Abschnitte 11 (Anlagenverwaltung) und 12 (Anzahlungen) in diesem Dokument bleiben
als Überblick stehen; die Ausarbeitung steht in den verlinkten Dokumenten.

### Der Nullsteuersatz

Der **Nullsteuersatz des § 12 Abs. 3 UStG** – „Die Steuer ermäßigt sich auf
0 Prozent" für die Lieferung von Solarmodulen an den Betreiber einer
Photovoltaikanlage, für deren wesentliche Komponenten und Speicher, für den
innergemeinschaftlichen Erwerb und die Einfuhr solcher Gegenstände sowie für die
Installation (Nrn. 1 bis 4) – ist etwas anderes als eine Steuerbefreiung: der
Umsatz ist **steuerpflichtig zum Satz null**, der Vorsteuerabzug des Leistenden
bleibt erhalten, und der SKR04 hat dafür ein eigenes Erlöskonto (**4290** Erlöse
0 % USt), das nicht mit den Konten für steuerfreie Umsätze zusammenfällt.

Der Fall ist als **eigener Steuerfall** umgesetzt, nicht als Steuersatz. Das war
die Alternative zu einem Eingriff in `TaxRate`, wo `0` weiterhin „kein Steuersatz"
bedeutet – die Doppelbelegung, an der der Fall früher scheiterte, löst sich damit
auf, statt sie mit einem Sentinel in die Steuerarithmetik einzubringen.

Zwei Konsequenzen folgen daraus:

- **Ein steuerpflichtiger Inlandsumsatz ohne Steuersatz wird abgewiesen.** Er hat
  19 % oder 7 %; fällt keine Steuer an, ist zu sagen warum. Die Buchungsgruppen,
  die bisher einen Satz von null als Abkürzung für „hier gibt es keine Vorsteuer"
  benutzten – Miete, Versicherungen, Beiträge, Nebenkosten des Geldverkehrs,
  Gehälter –, schlagen jetzt den zutreffenden Steuerfall vor.
- **Der Steuerfall wird an der Buchung gespeichert** und ist von der Hash-Chain
  gedeckt. Auf der Ausgangsseite ließe er sich aus dem Erlöskonto erschließen, auf
  der Eingangsseite nicht: ein nullbesteuerter Einkauf bucht auf dasselbe
  Aufwandskonto wie ein steuerfreier und hat wie dieser keine Steuerzeile. Ohne das
  Feld wären die beiden nach dem Buchen nicht mehr unterscheidbar – und sie sind
  nicht dasselbe.

## 17. Fundstellen

Die gesetzlichen Grundlagen dieses Dokuments stehen mit Fundstellen im
Anforderungskatalog: die Buchführungsgrundsätze unter GOB-01 bis GOB-06, Beleg,
Journal und offene Posten unter BEL-01 bis BEL-09, Unveränderbarkeit und
Festschreibung unter UNV-01 und UNV-02, Rechnung und E-Rechnung unter RECH-02,
RECH-03, RECH-06 und RECH-07, die Umsatzsteuer unter UST-01, UST-02 und UST-05,
Rechnungsabgrenzung und nicht abziehbare Betriebsausgaben unter BEW-08 und BEW-12.
Was im Katalog fehlt, steht hier:

Zum § 13b-Fall ist eine Präzisierung nötig, die im Code umgesetzt ist: die
Steuerschuldnerschaft folgt in den beiden für Buchfink relevanten Fällen aus
verschiedenen Merkmalen. Bei einer sonstigen Leistung eines im übrigen
Gemeinschaftsgebiet ansässigen Unternehmers (§ 13b Abs. 1 UStG) belegt die
USt-IdNr. des Lieferanten dessen Unternehmereigenschaft im Ausland. Bei einer
inländischen Bauleistung (§ 13b Abs. 2 Nr. 4 UStG) hängt die Steuerschuld dagegen
am **Leistungsempfänger**, der selbst nachhaltig Bauleistungen erbringen muss
(§ 13b Abs. 5 UStG). Eine fehlende USt-IdNr. des inländischen Lieferanten darf den
Fall deshalb nicht blockieren – sie ist für ihn kein Tatbestandsmerkmal.

**Korrektur gegenüber einer früheren Fassung:** die Einforderung ausstehender
Einlagen war dort auf § 19 Abs. 2 GmbHG gestützt. Das ist falsch – § 19 Abs. 2
GmbHG regelt das Verbot, den Gesellschafter von der Einlagepflicht zu befreien.
Die Zuständigkeit der Gesellschafter für die Einforderung steht in § 46 Nr. 2
GmbHG.

### Kontenrahmen

DATEV **SKR04** 2026, Art.-Nr. 11175, extrahiert aus
`assets/DATEV-SKR04-BilrUg-2026.pdf` nach
`internal/accounting/skr04_2026.json`. Der Test
`TestConceptDocumentsUseRealSKR04Accounts` prüft bei jedem Build, dass jede in den
Konzeptdokumenten hervorgehobene Kontonummer im Katalog existiert.

### Was nicht geprüft ist

- Das **DATEV-Austauschformat** – siehe die Vorbemerkung in
  [DATEV-Export](datev-export.md).
- Die **technischen Normen** EN 16931, XRechnung und ZUGFeRD.
- Die amtlichen **AfA-Tabellen**; sie sind Verwaltungsanweisung, nicht Gesetz.
