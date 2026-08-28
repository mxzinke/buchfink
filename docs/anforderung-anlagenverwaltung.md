# Buchfink – Anlagenverwaltung

Status: Anforderung, noch nicht implementiert
Letzte Aktualisierung: 2026-08-22
Voraussetzung: [Beleg- & Buchungsflow](anforderung-beleg-buchungsflow.md)

> Kontonummern sind gegen `internal/accounting/skr04_2026.json` (DATEV SKR04 2026,
> Art.-Nr. 11175) geprüft. Alle Paragrafenangaben sind am **22.08.2026** gegen den
> Gesetzestext auf gesetze-im-internet.de verifiziert; die Fundstellen stehen in
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

Daraus folgt ein Punkt, der mit dem bestehenden Zahlungsflow kollidiert: **Skonto auf
eine Anlage mindert die Anschaffungskosten, nicht den Aufwand.** Die vorhandene
Skonto-Logik bucht auf 5736/4736 und korrigiert die Steuer – für eine Anlage müsste
sie stattdessen das Anlagekonto mindern und die AfA-Bemessungsgrundlage anpassen.
Dasselbe gilt für Rabatte und für nachträgliche Anschaffungskosten.

Ebenso zu erfassen: Fracht, Montage, Überführung sind Nebenkosten und gehören auf das
Anlagekonto; Finanzierungskosten dagegen nicht.

Zugangsbuchung (Beispiel Pkw auf Ziel):

| Buchung |
|---|
| SOLL **0520** Pkw + SOLL **1406** Vorsteuer · HABEN **Kreditorenkonto** |

Anzahlungen auf noch nicht gelieferte Anlagen und Anlagen im Bau laufen über **0700**
Geleistete Anzahlungen und Anlagen im Bau und werden bei Fertigstellung umgebucht.

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

## 9. Offene Entscheidungen

- **Wertgrenzen und AfA-Sätze:** Quelle und Versionierung. Vorschlag: Stammdaten je
  Geschäftsjahr mit vorbelegten Werten, überschreibbar.
- **Skonto und Rabatt auf Anlagen:** die bestehende Skonto-Logik mindert Aufwand und
  Steuer. Für Anlagen müsste sie die Anschaffungskosten mindern und die AfA neu
  berechnen. Wie weit soll das automatisch laufen?
- **Nutzungsdauer-Katalog:** ausliefern oder leer starten?
- **Wechsel degressiv → linear:** automatisch zum optimalen Zeitpunkt vorschlagen oder
  nur zulassen?
- **Anlagenkartei über Geschäftsjahre:** eigene Tabelle außerhalb der
  Geschäftsjahres-Logik, oder Fortschreibung beim Jahreswechsel?
- **Finanzanlagen:** Wertpapiere und Beteiligungen unterliegen anderen
  Bewertungsregeln als Sachanlagen. In v1 aufnehmen oder zurückstellen?

## 10. Abhängigkeiten

- Die **Passiva-Verwaltung** ([anforderung-passiva-verwaltung.md](anforderung-passiva-verwaltung.md))
  beschreibt dieselbe Mechanik für Darlehen, Rückstellungen und Sonderposten: Stammsatz,
  Plan, Bewegungen mit Journalverweis. Das Datenmodell aus Abschnitt 8 sollte für beide
  Bilanzseiten tragen – und ein Verzeichniseintrag muss auf einen anderen verweisen
  können, weil ein Investitionszuschuss (Sonderposten **2998**) an der AfA des
  Anlageguts hängt.
- Der **Festschreibungs-Workflow** muss vor der Jahressperre die AfA-Prüfung aufrufen.
- Der **Zahlungsflow** braucht eine Sonderbehandlung für Skonto auf Anlagen.
- Die **E-Bilanz** braucht den Anlagenspiegel als Kontennachweis.

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
| Sonderabschreibung bis 40 %, verteilbar auf fünf Jahre | § 7g Abs. 5 EStG | [estg/__7g.html](https://www.gesetze-im-internet.de/estg/__7g.html) |
| Gewinngrenze 200.000 € im Vorjahr | § 7g Abs. 6 Nr. 1 i. V. m. Abs. 1 Satz 2 Nr. 1 Buchst. b EStG | dito |
| Außerplanmäßige Abschreibung bei voraussichtlich dauernder Wertminderung | § 253 Abs. 3 Satz 5 HGB | [hgb/__253.html](https://www.gesetze-im-internet.de/hgb/__253.html) |
| Anlagenspiegel als Anhangbestandteil | § 284 Abs. 3 HGB | [hgb/__284.html](https://www.gesetze-im-internet.de/hgb/__284.html) |
| Befreiung kleiner Kapitalgesellschaften vom Anlagenspiegel | § 288 Abs. 1 Nr. 1 HGB | [hgb/__288.html](https://www.gesetze-im-internet.de/hgb/__288.html) |

**Korrektur gegenüber einer früheren Fassung dieses Dokuments:** die Befreiung
kleiner Kapitalgesellschaften vom Anlagenspiegel wurde dort auf § 274a HGB
gestützt. Das ist falsch – § 274a HGB befreit von § 268 Abs. 4 Satz 2, § 268
Abs. 5 Satz 3, § 268 Abs. 6 und § 274 HGB, nicht von § 284 Abs. 3 HGB. Die
richtige Fundstelle ist § 288 Abs. 1 Nr. 1 HGB.

**Nicht aus dem Gesetz, sondern Verwaltungsanweisung:** die AfA-Tabellen (BMF).
Sie binden die Finanzverwaltung, nicht den Steuerpflichtigen; eine abweichende,
begründete Nutzungsdauer ist zulässig. Deshalb muss der Katalog in Buchfink
überschreibbar bleiben.
