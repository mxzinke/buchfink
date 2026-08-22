# Buchfink – Beleg- & Buchungsflow

Status: Konzept, Buchungskern implementiert
Letzte Aktualisierung: 2026-08-22
Kontenrahmen: DATEV SKR04 2026 (Art.-Nr. 11175)

> Alle Kontonummern in diesem Dokument sind gegen `internal/accounting/skr04_2026.json`
> geprüft, das aus `assets/DATEV-SKR04-BilrUg-2026.pdf` extrahiert wurde. Der Test
> `TestPostingGroupAccountsExistInSKR04` prüft bei jedem Build, dass jede Kontierung
> im Code auf ein existierendes, bebuchbares SKR04-Konto zeigt.
>
> **Alle Paragrafenangaben sind am 22.08.2026 gegen den Gesetzestext geprüft**;
> das Quellenverzeichnis mit Fundstellen und den dabei gefundenen Korrekturen
> steht in [Abschnitt 17](#17-quellen).
>
> **Vorsicht bei SKR03-Nummern.** Eine frühere Fassung dieses Dokuments enthielt
> durchgehend SKR03-Konten. Die Nummern kollidieren: 1600 ist im SKR04 die *Kasse*
> und im SKR03 *Verbindlichkeiten aus LuL*, 8400 gibt es im SKR04 gar nicht, 4930
> sind *Erträge aus der Auflösung von Rückstellungen* statt Bürobedarf. Der
> Buchungskern lehnt Konten der Klasse 8 daher mit einem expliziten Hinweis ab.

## 1. Leitgedanke

Ein Abstraktionslayer, der dem Nutzer verständlich ist: er denkt in **Belegen** und
**Rechnungen**, die Software übersetzt über Backend-Logik und saubere Auswahl in die
korrekten **SOLL/HABEN-Buchungen**. Beide Seiten sind sauber getrennt, aber
deterministisch verbunden.

**Kein „vorbereiten, nicht buchen".** Jeder erfasste Beleg wird sofort gebucht. Ob
dabei Geld fließt, entscheidet die Kontenseite: ist noch nicht bezahlt, entsteht eine
Verbindlichkeit bzw. Forderung auf dem Personenkonto des Geschäftspartners. Die
Zahlung ist ein späterer, separater Geschäftsvorfall.

Das ist keine Bequemlichkeitsentscheidung, sondern folgt aus der GoBD: unbare
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
einer GmbH offen. Buchfink fragt die Versteuerungsart im Setup ab und weist
Istversteuerung in v1 ausdrücklich ab, statt sie stillschweigend falsch zu
behandeln.

**E-Rechnung ist keine Scope-Grenze, sondern eine offene Pflicht.** Der Empfang
strukturierter Eingangsrechnungen fehlt, ist aber seit dem 01.01.2025 verpflichtend
und von den Übergangsregelungen des § 27 Abs. 38 UStG nicht erfasst – die gelten
nur für das Ausstellen. Das ist damit kein „in v1 außen vor", sondern eine Lücke
mit Frist; siehe [anforderung-e-rechnung.md](anforderung-e-rechnung.md).

**Steuern und Auswertungen bleiben außen vor.** USt-Voranmeldung, USt-Erklärung, ZM
und der Jahresabschluss (Saldenvortrag, GuV-Abrechnung, Bilanzierung) sind nicht Teil
des Beleg- und Zahlungsflows. Sie brauchen eigene Eingabemasken und eigene Logik. Der
Buchungskern liefert die Grundlage dafür – jede Buchung trägt Steuerschlüssel,
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

Die Stammdaten müssen den Steuerfall tragen. Ohne USt-IdNr. des Empfängers lehnt
Buchfink eine innergemeinschaftliche Lieferung ab (§ 6a Abs. 1 Nr. 4 UStG); ein
deutscher Kunde kann keine bekommen, ein EU-Kunde keine Ausfuhrlieferung.

## 6. Kontierung: fachliche Gruppe → SKR04

Der Nutzer wählt eine Gruppe, das Backend mappt deterministisch. Keine Lernfunktion,
keine Heuristik. Das Konto hängt an Gruppe **plus Steuerfall plus Steuersatz**:

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

Der Gegenbetrag ergibt sich aus dem Ausgleich der übrigen Zeilen. Das trägt jeden
Steuerfall ohne Sonderfall: bei einer Inlandsrechnung sind es netto plus Vorsteuer,
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

### 7.3 Zahlungsdifferenzen

Der Zahlbetrag stimmt aus mehreren Gründen nicht mit dem Belegbetrag überein. Ohne
saubere Behandlung bleiben offene Posten mit drei Cent ewig stehen, und irgendwann
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

1200 (Forderungen aus LuL) und 3300 (Verbindlichkeiten aus LuL) sind keine
Buchungsziele, sondern **Bilanzpositionen**. Das ist keine Designentscheidung, sondern
§ 266 HGB: die Bilanz zeigt eine Zeile „Forderungen aus Lieferungen und Leistungen",
nicht vierhundert Kundenzeilen. Der Betrag dieser Zeile entsteht durch Verdichtung der
Personenkonten.

Der Buchungskern weist eine Zeile auf 1200 oder 3300 deshalb ab, so wie er auch
Steuerkonten abweist. Wäre beides erlaubt, gäbe es zwei Wahrheiten für dieselbe Zahl:
eine direkt gebuchte Forderung stünde in der Bilanz, aber in keiner OPOS-Liste, und die
Differenz fiele erst auf, wenn jemand nachrechnet, warum das Kundenkonto nicht zur
Bilanzposition passt.

In der Kontenübersicht ist die Verdichtung sichtbar gemacht – 3300 trägt den Hinweis,
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

Alle Buchungssätze sind als Test hinterlegt
(`internal/service/posting_service_test.go`, `payment_service_test.go`).

### 10.1 Lieferantenrechnung auf Ziel, Inland

Dienstleistung 1.000 € netto, 19 % USt.

| Schritt | Buchung |
|---|---|
| Beleg erfassen | SOLL **5906** Fremdleistungen 1.000,00 + SOLL **1406** Vorsteuer 190,00 · HABEN **Kreditorenkonto** 1.190,00 |
| Zahlung | SOLL **Kreditorenkonto** 1.190,00 · HABEN **1800** Bank 1.190,00 |

### 10.2 Ausgangsrechnung auf Ziel, Inland

Beratungsleistung 2.000 € netto, 19 % USt.

| Schritt | Buchung |
|---|---|
| Rechnung ausstellen | SOLL **Debitorenkonto** 2.380,00 · HABEN **4400** Erlöse 19 % 2.000,00 + HABEN **3806** Umsatzsteuer 380,00 |
| Zahlungseingang | SOLL **1800** Bank 2.380,00 · HABEN **Debitorenkonto** 2.380,00 |

### 10.3 Reverse Charge, § 13b UStG

Cloud-Leistung eines irischen Anbieters, 1.000 € netto.

| Schritt | Buchung |
|---|---|
| Beleg erfassen | SOLL **5909** Fremdleistungen (§ 13b) 1.000,00 + SOLL **1407** Vorsteuer § 13b 190,00 · HABEN **3837** Umsatzsteuer § 13b 190,00 + HABEN **Kreditorenkonto** 1.000,00 |

An den Lieferanten gehen nur 1.000 €. Die Steuer wird geschuldet und zugleich als
Vorsteuer abgezogen; ergebniswirksam ist der Vorgang neutral, für die Voranmeldung
aber nicht.

### 10.4 Innergemeinschaftlicher Erwerb

Warenkauf aus den Niederlanden, 2.000 € netto.

| Schritt | Buchung |
|---|---|
| Beleg erfassen | SOLL **5400** Wareneingang 19 % VSt 2.000,00 + SOLL **1404** Vorsteuer i.g. Erwerb 380,00 · HABEN **3804** USt i.g. Erwerb 380,00 + HABEN **Kreditorenkonto** 2.000,00 |

### 10.5 Beleg mit zwei Steuersätzen

Hotelrechnung: Übernachtung 200 € netto zu 7 %, Frühstück 50 € netto zu 19 %.

| Buchung |
|---|
| SOLL **6650** Reisekosten 200,00 + SOLL **6650** Reisekosten 50,00 + SOLL **1401** Vorsteuer 7 % 14,00 + SOLL **1406** Vorsteuer 19 % 9,50 · HABEN **Kreditorenkonto** 273,50 |

### 10.6 Barzahlung

Quittung Büromaterial, 50 € netto, sofort bar bezahlt.

| Buchung |
|---|
| SOLL **6815** Bürobedarf 50,00 + SOLL **1406** Vorsteuer 9,50 · HABEN **1600** Kasse 59,50 |

Bar heißt **Kasse**. Wer stattdessen 1800 Bank bucht, hat einen Kassenbestand, den es
nicht gibt, und einen Bankbestand, der nicht stimmt.

### 10.7 Zahlung mit Skonto

Rechnung über 1.190,00 € brutto, 2 % Skonto bei Zahlung binnen 10 Tagen.

| Buchung |
|---|
| SOLL **Kreditorenkonto** 1.190,00 · HABEN **5736** Erhaltene Skonti 20,00 + HABEN **1406** Vorsteuer 3,80 + HABEN **1800** Bank 1.166,20 |

Die Vorsteuer steht danach bei 186,20 € statt 190,00 € – das ist die Korrektur nach
§ 17 UStG.

### 10.8 Zahlung mit Bankgebühr

Auslandsüberweisung, die Bank bucht 5,00 € Entgelt zusätzlich ab.

| Buchung |
|---|
| SOLL **Kreditorenkonto** 1.190,00 + SOLL **6855** Nebenkosten des Geldverkehrs 5,00 · HABEN **1800** Bank 1.195,00 |

### 10.9 Storno per Generalumkehr

Beleg aus 10.1 doppelt erfasst.

| Buchung |
|---|
| SOLL **5906** −1.000,00 + SOLL **1406** −190,00 · HABEN **Kreditorenkonto** −1.190,00 |

Gleiche Konten, gleiche Seiten, negierte Beträge. Die Verkehrszahlen von 5906 stehen
danach wieder auf null.

### 10.10 Rückstellung zum Bilanzstichtag

Gewährleistungsrückstellung 5.000 €, § 249 HGB.

| Buchung |
|---|
| SOLL **6790** Aufwand für Gewährleistung 5.000,00 · HABEN **3090** Rückstellungen für Gewährleistungen 5.000,00 |

Kein Bankumsatz, keine Zahlungszuordnung. Abschlussbuchung, üblicherweise im Q1 des
Folgejahres gebucht.

### 10.11 Darlehen

| Schritt | Buchung |
|---|---|
| Auszahlung 50.000 € | SOLL **1800** Bank 50.000,00 · HABEN **3160** Verb. ggü. Kreditinstituten (1–5 J.) 50.000,00 |
| Rate 1.000 € (800 Tilgung, 200 Zins) | SOLL **3160** 800,00 + SOLL **7320** Zinsaufwendungen langfristig 200,00 · HABEN **1800** Bank 1.000,00 |

### 10.12 Zusammenfassung der Nutzer-Aktionen

Alle Vorfälle laufen über drei Aktionen:

1. **Etwas erfassen** – Beleg, Rechnung, Rückstellung, Anlage → sofortige Buchung
2. **Zahlung zuordnen** – Ausgleich offener Posten inklusive Differenzen
3. **Freigeben** – für periodische Vorfälle wie AfA und Abschlussbuchungen

Der Nutzer wählt nie direkt eine SOLL/HABEN-Kombination. Die Kontenwahl ergibt sich
deterministisch aus fachlicher Gruppe, Steuerfall, Steuersatz und Zahlungsweg.

## 11. Anlagenverwaltung

> Überblick. Die Ausarbeitung steht in
> [anforderung-anlagenverwaltung.md](anforderung-anlagenverwaltung.md).

### 11.1 Die erste Frage ist nicht die AfA-Methode

Bei jeder Anschaffung steht zuerst eine andere Entscheidung an:

| Fall | Behandlung | Konten |
|---|---|---|
| **GWG, Sofortabschreibung** | selbständig nutzbar bis zur Wertgrenze § 6 Abs. 2 EStG | 0670 Zugang, 6260 Sofortabschreibung |
| **Sammelposten** | Poolabschreibung § 6 Abs. 2a EStG über fünf Jahre | 0675 Zugang, 6264 Abschreibung |
| **Anlagegut** | Aktivierung mit planmäßiger AfA | Anlagekonto, 6220 / 6222 |

Die Wertgrenzen gehören in die Stammdaten, nicht in den Code – sie ändern sich.

### 11.2 Anlagegut

- **Erfassen:** Bezeichnung, Anschaffungskosten, Anschaffungsdatum, Nutzungsdauer,
  AfA-Methode.
- **Zugang:** SOLL Anlagekonto · HABEN Kreditorenkonto oder Zahlungsmittel.
  Beispiel Pkw: SOLL **0520** Pkw · HABEN **1800** Bank.
- **AfA:** SOLL **6222** Abschreibungen auf Fahrzeuge · HABEN **0520** Pkw. Für übrige
  Sachanlagen 6220.
- **Abgang:** Restbuchwert ausbuchen, Gewinn oder Verlust erfassen.

### 11.3 AfA-Methoden

Linear als Standard. Degressiv nach dem Investitionssofortprogramm 2025 (das Dreifache
der linearen AfA, höchstens 30 %, für Anschaffungen vom 01.07.2025 bis 31.12.2027).
Zusätzlich Sonderabschreibung nach § 7g Abs. 5 EStG (bis 40 % für KMU,
Fünf-Jahres-Begünstigungszeitraum). Übergang von degressiv auf linear ist zulässig.

### 11.4 Kopplung an die Festschreibung

AfA und Rückstellungen sind Abschlussbuchungen zum Bilanzstichtag (§ 253 Abs. 3 HGB,
§ 7 EStG, § 249 HGB), keine laufenden Geschäftsvorfälle. Sie werden nicht im
Hintergrund gebucht. Stattdessen ist die **jährliche** Festschreibung der Auslöser:

1. Vor der Jahres-Festschreibung prüft Buchfink, ob für alle Anlagegüter die fällige
   AfA gebucht ist.
2. Fehlende Buchungen werden angezeigt und lassen sich mit Vorschau und Freigabe
   erzeugen.
3. Erst danach kann das Jahr festgeschrieben werden.

Bei monatlicher oder quartalsweiser Festschreibung wird die AfA nicht geprüft – sie ist
eine Jahresendbuchung. Bei unterjähriger Anschaffung wird zeitanteilig monatsgenau
gerechnet.

## 12. Anzahlungen

> Überblick. Die Ausarbeitung steht in
> [anforderung-anzahlungen.md](anforderung-anzahlungen.md).

Bei Anzahlungen entsteht die Umsatzsteuer mit der Vereinnahmung – auch bei
Sollversteuerung (§ 13 Abs. 1 Nr. 1 lit. a Satz 4 UStG). Das ist kein Detail, sondern
ein eigener Buchungsweg:

- Erhaltene Anzahlung: HABEN **3272** Erhaltene, versteuerte Anzahlungen 19 % USt,
  Steuer sofort abführen.
- Geleistete Anzahlung: SOLL **1180** Geleistete Anzahlungen auf Vorräte (bzw. das
  passende Anzahlungskonto der jeweiligen Bilanzposition).
- Die Schlussrechnung setzt die Anzahlungen ab; nur die Differenz wird zum offenen
  Posten.

Der Rechnungsverbund fasst Abschläge und Schlussrechnung als eigener Entity zusammen
und stellt den Gesamtfortschritt dar. Ohne die Anzahlungskonten und die Verrechnung in
der Schlussrechnung wäre er nur eine UI-Gruppierung.

## 13. Nicht abziehbare Betriebsausgaben

Einige Aufwendungen sind handelsrechtlich Aufwand, steuerlich aber nur teilweise
abziehbar. Sie brauchen getrennte Konten, sonst ist die Steuerbilanz falsch:

| Fall | Konten |
|---|---|
| **Bewirtung** – 70 % abziehbar, 30 % nicht (§ 4 Abs. 5 Satz 1 Nr. 2 EStG); die Vorsteuer bleibt trotzdem voll abziehbar, § 15 Abs. 1a Satz 2 UStG nimmt Bewirtungsaufwendungen vom Vorsteuerausschluss ausdrücklich aus | 6640 abziehbar, 6644 nicht abzugsfähig |
| **Geschenke** – abziehbar, solange die Zuwendungen an einen Empfänger im Wirtschaftsjahr **50 €** nicht übersteigen (§ 4 Abs. 5 Satz 1 Nr. 1 Satz 2 EStG); darüber weder Aufwand noch Vorsteuer (§ 15 Abs. 1a Satz 1 UStG) | 6610 abzugsfähig, 6620 nicht abzugsfähig |

Beide Grenzen gehören wie die AfA-Wertgrenzen in die versionierten Stammdaten: die
Geschenkegrenze lag bis einschließlich der vor dem 01.01.2024 beginnenden
Wirtschaftsjahre bei 35 €. Ein fest verdrahteter Wert bucht ein nachbearbeitetes
Altjahr still falsch.

Die Bewirtung hat zusätzlich eine **Aufzeichnungspflicht**, die keine Buchung ist,
aber am Beleg hängt: Ort, Tag, Teilnehmer und Anlass der Bewirtung sowie die Höhe
der Aufwendungen sind schriftlich festzuhalten; bei einer Gaststätte genügen Anlass
und Teilnehmer, die Rechnung ist beizufügen (§ 4 Abs. 5 Satz 1 Nr. 2 Sätze 2 und 3
EStG). Das sind Pflichtfelder am Beleg, sobald auf 6640 gebucht wird – ohne sie ist
der Abzug auch für die 70 % verloren.

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

- **Vorschau:** PDF für Belege und Rechnungen, Bildvorschau für Scans und Fotos.
- **Ablage:** nach Jahr und Art sortiert (`belege/<jahr>/eingang/…`, `…/ausgang/…`),
  Originaldatei unverändert, Hash-gesichert.
- Der Beleg-Hash ist Teil der Buchung und damit der Hash-Chain; der Dateipfad nicht,
  damit ein verschobener Datenordner die Kette nicht bricht.

## 16. Entscheidungen

| Thema | Entscheidung |
|---|---|
| **Buchungsmodell** | Kopf + n Zeilen mit harter Invariante Summe Soll = Summe Haben |
| **Beträge** | ganzzahlige Cent, Rundung einmal je Steuersatzgruppe |
| **Storno** | Generalumkehr, nicht Seitentausch |
| **Personenkonten** | echte DATEV-Nummernkreise 10000–69999 / 70000–99999; 1200 und 3300 sind Bilanzpositionen und keine Buchungsziele |
| **Reverse Charge** | in v1 enthalten, kein Randfall |
| **Kontierung** | deterministisch, keine Lernfunktion, Regelwerk versioniert |
| **Versteuerung** | nur Sollversteuerung; Istversteuerung wird abgewiesen |
| **CAMT-Import** | schlägt keine Konten vor |
| **Warenbestand** | out of scope für v1 |
| **Steuern & Abschluss** | eigene Masken, nicht Teil des Belegflows |

### Offene Punkte

Jeder hat ein eigenes Anforderungsdokument, damit er für sich angegangen werden kann:

| Thema | Dokument | Was noch fehlt |
|---|---|---|
| **E-Rechnung** | [anforderung-e-rechnung.md](anforderung-e-rechnung.md) | **Kein Konzeptpunkt, sondern geltendes Recht.** Der Empfang einer E-Rechnung ist seit dem 01.01.2025 Pflicht und hat nie eine Übergangsfrist gehabt; Buchfink kann ihn nicht. Der Vorsteuerabzug hängt am strukturierten Teil. |
| **Rechnungsabgrenzung** | [anforderung-rechnungsabgrenzung.md](anforderung-rechnungsabgrenzung.md) | Der Leistungszeitraum wird erfasst, die Abgrenzungsbuchung auf 1900 / 3900 fehlt. Kleinster Punkt, hängt im Kern an einer Entscheidung. |
| **Anzahlungen & Rechnungsverbund** | [anforderung-anzahlungen.md](anforderung-anzahlungen.md) | Die Steuer entsteht mit der Vereinnahmung, nicht mit der Rechnung. Die Schlussrechnung muss die Anzahlungen absetzen, sonst greift § 14c Abs. 1 UStG. |
| **Anlagenverwaltung** | [anforderung-anlagenverwaltung.md](anforderung-anlagenverwaltung.md) | Größter Block: GWG und Sammelposten, AfA-Methoden, Abgang, Anlagenspiegel. |
| **DATEV-Export** | [anforderung-datev-export.md](anforderung-datev-export.md) | Keine buchhalterische Entscheidung offen, aber eine architektonische: das n-Zeilen-Modell trifft auf DATEVs Konto/Gegenkonto. |

Die E-Rechnung steht bewusst oben. Die anderen vier beschreiben Funktionen, die
fehlen; sie beschreibt eine Pflicht, die läuft.

Die Abschnitte 11 (Anlagenverwaltung) und 12 (Anzahlungen) in diesem Dokument bleiben
als Überblick stehen; die Ausarbeitung steht in den verlinkten Dokumenten.

### Bekannte Lücke im Steuermodell: der Nullsteuersatz

`TaxRate` ist in Basispunkten geführt, und `0` bedeutet dort „keine Steuer".
Damit lässt sich der **Nullsteuersatz des § 12 Abs. 3 UStG** nicht abbilden:
„Die Steuer ermäßigt sich auf 0 Prozent" für die Lieferung von Solarmodulen an
den Betreiber einer Photovoltaikanlage, für deren wesentliche Komponenten und
Speicher, für den innergemeinschaftlichen Erwerb und die Einfuhr solcher
Gegenstände sowie für die Installation (Nrn. 1 bis 4).

Fachlich ist das etwas anderes als eine Steuerbefreiung: der Umsatz ist
**steuerpflichtig zum Satz null**, der Vorsteuerabzug des Leistenden bleibt
erhalten, und der SKR04 hat dafür ein eigenes Erlöskonto (**4290** Erlöse 0 %
USt), das nicht mit den Konten für steuerfreie Umsätze zusammenfällt.

Solange `0` doppelt belegt ist, würde der Fall stillschweigend als steuerfrei
gebucht und auf dem falschen Konto landen. Die Lösung ist klein, aber sie greift
in den Typ ein: entweder ein eigener `TaxTreatment` für den Nullsteuersatz oder
ein separates „Steuersatz gesetzt / nicht gesetzt" neben dem Wert. Bis dahin gilt
der Fall ausdrücklich als **nicht unterstützt** – lieber eine bekannte Lücke als
eine unbemerkt falsche Buchung.

## 17. Quellen

Stand der Prüfung: **22.08.2026**. Alle Paragrafenangaben in diesem Dokument sind
gegen den Volltext auf [gesetze-im-internet.de](https://www.gesetze-im-internet.de)
geprüft, die GoBD gegen das BMF-Schreiben im Original.

### Umsatzsteuer

| Aussage | Fundstelle | Link |
|---|---|---|
| Sollversteuerung: Berechnung nach vereinbarten Entgelten | § 16 Abs. 1 Satz 1 UStG | [ustg_1980/__16.html](https://www.gesetze-im-internet.de/ustg_1980/__16.html) |
| Entstehung der Steuer bei Sollversteuerung mit Ablauf des Voranmeldungszeitraums der Leistung | § 13 Abs. 1 Nr. 1 Buchst. a Satz 1 UStG | [ustg_1980/__13.html](https://www.gesetze-im-internet.de/ustg_1980/__13.html) |
| Istversteuerung: Entstehung mit Vereinnahmung | § 13 Abs. 1 Nr. 1 Buchst. b UStG | dito |
| Gestattung der Istversteuerung, Gesamtumsatz höchstens 800.000 € im Vorjahr, ohne Rechtsformbindung | § 20 Satz 1 Nr. 1 UStG | [ustg_1980/__20.html](https://www.gesetze-im-internet.de/ustg_1980/__20.html) |
| Rechnungsnummer einmalig und fortlaufend | § 14 Abs. 4 Nr. 4 UStG | [ustg_1980/__14.html](https://www.gesetze-im-internet.de/ustg_1980/__14.html) |
| Leistungszeitpunkt als Pflichtangabe | § 14 Abs. 4 Nr. 6 UStG | dito |
| E-Rechnungspflicht im B2B-Inland | § 14 Abs. 2 Satz 2 Nr. 1 UStG | dito |
| Übergangsregelungen zur E-Rechnung (nur Ausstellung) | § 27 Abs. 38 UStG | [ustg_1980/__27.html](https://www.gesetze-im-internet.de/ustg_1980/__27.html) |
| Steuerschuld bei zu hohem Steuerausweis | § 14c Abs. 1 UStG | [ustg_1980/__14c.html](https://www.gesetze-im-internet.de/ustg_1980/__14c.html) |
| Steuerbefreiungen (i. g. Lieferung, Ausfuhr u. a.) | § 4 UStG | [ustg_1980/__4.html](https://www.gesetze-im-internet.de/ustg_1980/__4.html) |
| Nullsteuersatz Photovoltaik | § 12 Abs. 3 UStG | [ustg_1980/__12.html](https://www.gesetze-im-internet.de/ustg_1980/__12.html) |
| Reverse Charge, Steuerschuldnerschaft des Leistungsempfängers | § 13b UStG | [ustg_1980/__13b.html](https://www.gesetze-im-internet.de/ustg_1980/__13b.html) |
| Änderung der Bemessungsgrundlage (Skonto) | § 17 UStG | [ustg_1980/__17.html](https://www.gesetze-im-internet.de/ustg_1980/__17.html) |
| Vorsteuerausschluss, Ausnahme für Bewirtung | § 15 Abs. 1a UStG | [ustg_1980/__15.html](https://www.gesetze-im-internet.de/ustg_1980/__15.html) |

Zum § 13b-Fall ist eine Präzisierung nötig, die im Code umgesetzt ist: die
Steuerschuldnerschaft folgt in den beiden für Buchfink relevanten Fällen aus
verschiedenen Merkmalen. Bei einer sonstigen Leistung eines im übrigen
Gemeinschaftsgebiet ansässigen Unternehmers (§ 13b Abs. 1 UStG) belegt die
USt-IdNr. des Lieferanten dessen Unternehmereigenschaft im Ausland. Bei einer
inländischen Bauleistung (§ 13b Abs. 2 Nr. 4 UStG) hängt die Steuerschuld dagegen
am **Leistungsempfänger**, der selbst nachhaltig Bauleistungen erbringen muss
(§ 13b Abs. 5 UStG). Eine fehlende USt-IdNr. des inländischen Lieferanten darf den
Fall deshalb nicht blockieren – sie ist für ihn kein Tatbestandsmerkmal.

### Einkommensteuer

| Aussage | Fundstelle | Link |
|---|---|---|
| Bewirtung: 70 % abziehbar, Aufzeichnungspflichten | § 4 Abs. 5 Satz 1 Nr. 2 EStG | [estg/__4.html](https://www.gesetze-im-internet.de/estg/__4.html) |
| Geschenke: Grenze 50 € je Empfänger und Wirtschaftsjahr | § 4 Abs. 5 Satz 1 Nr. 1 Satz 2 EStG | dito |
| GWG, Sammelposten | § 6 Abs. 2, Abs. 2a EStG | [estg/__6.html](https://www.gesetze-im-internet.de/estg/__6.html) |
| AfA | § 7 EStG | [estg/__7.html](https://www.gesetze-im-internet.de/estg/__7.html) |
| Sonderabschreibung | § 7g Abs. 5 EStG | [estg/__7g.html](https://www.gesetze-im-internet.de/estg/__7g.html) |

### Handelsrecht

| Aussage | Fundstelle | Link |
|---|---|---|
| Rückstellungen | § 249 HGB | [hgb/__249.html](https://www.gesetze-im-internet.de/hgb/__249.html) |
| Rechnungsabgrenzungsposten | § 250 HGB | [hgb/__250.html](https://www.gesetze-im-internet.de/hgb/__250.html) |
| Außerplanmäßige Abschreibung | § 253 Abs. 3 HGB | [hgb/__253.html](https://www.gesetze-im-internet.de/hgb/__253.html) |
| Währungsumrechnung | § 256a HGB | [hgb/__256a.html](https://www.gesetze-im-internet.de/hgb/__256a.html) |
| Bilanzgliederung | § 266 HGB | [hgb/__266.html](https://www.gesetze-im-internet.de/hgb/__266.html) |
| Gezeichnetes Kapital, offener Abzug nicht eingeforderter Einlagen | § 272 Abs. 1 HGB, insbes. Satz 3 | [hgb/__272.html](https://www.gesetze-im-internet.de/hgb/__272.html) |
| Einforderung der Einlagen durch Gesellschafterbeschluss | § 46 Nr. 2 GmbHG | [gmbhg/__46.html](https://www.gesetze-im-internet.de/gmbhg/__46.html) |

**Korrektur gegenüber einer früheren Fassung:** die Einforderung ausstehender
Einlagen war dort auf § 19 Abs. 2 GmbHG gestützt. Das ist falsch – § 19 Abs. 2
GmbHG regelt das Verbot, den Gesellschafter von der Einlagepflicht zu befreien.
Die Zuständigkeit der Gesellschafter für die Einforderung steht in § 46 Nr. 2
GmbHG.

### GoBD

**GoBD in der Fassung vom 14.07.2025**, BMF-Schreiben GZ IV D 2 -
S 0316/00128/005/088, BStBl I S. 1502 (2. Änderung). Ausgangsfassung: BMF-Schreiben
vom 28.11.2019, BStBl I S. 1269, geändert am 11.03.2024, BStBl I S. 374.

<https://www.bundesfinanzministerium.de/Content/DE/Downloads/BMF_Schreiben/Weitere_Steuerthemen/Abgabenordnung/2025-07-14-GoBD-2-aenderung.html>

Die 2. Änderung ist ausdrücklich mit der Einführung der obligatorischen E-Rechnung
begründet und betrifft die Aufbewahrung strukturierter Datensätze. Was daraus für
Buchfink folgt, steht in [anforderung-e-rechnung.md](anforderung-e-rechnung.md),
Abschnitt 6.

### Kontenrahmen

DATEV **SKR04** 2026, Art.-Nr. 11175, extrahiert aus
`assets/DATEV-SKR04-BilrUg-2026.pdf` nach
`internal/accounting/skr04_2026.json`. Der Test
`TestConceptDocumentsUseRealSKR04Accounts` prüft bei jedem Build, dass jede in den
Konzeptdokumenten hervorgehobene Kontonummer im Katalog existiert.

### Was nicht geprüft ist

- Das **DATEV-Austauschformat** – siehe die Vorbemerkung in
  [anforderung-datev-export.md](anforderung-datev-export.md).
- Die **technischen Normen** EN 16931, XRechnung und ZUGFeRD.
- Die amtlichen **AfA-Tabellen**; sie sind Verwaltungsanweisung, nicht Gesetz.
