# Gründungsweg und Behördenwege

Rechtsstand: 19. Oktober 2026. Wortlaut von GmbHG, HGB, AO, EStDV, GewO, GwG,
SGB IV, SGB VII und IHKG am 17.10.2026 gelesen, § 5b EStG am 19.10.2026.

## Inhalt

1. Reihenfolge der Gründung
2. Gesellschaftsvertrag
3. Anmeldung und Eintragung
4. Behördenwege mit Fristen
5. Nach der Eintragung

## 1. Reihenfolge der Gründung

1. **Beurkundung.** Der Gesellschaftsvertrag braucht notarielle Form; alle
   Gesellschafter unterschreiben ([§ 2 Abs. 1 GmbHG][gmbhg2]). Vertreter
   brauchen eine notariell errichtete oder beglaubigte Vollmacht
   ([§ 2 Abs. 2 GmbHG][gmbhg2]). Die Beurkundung ist auch per
   Videokommunikation möglich ([§ 2 Abs. 3 GmbHG][gmbhg2]).
2. **Vereinfachtes Verfahren.** Mit höchstens drei Gesellschaftern und einem
   Geschäftsführer kann das Musterprotokoll genutzt werden; abweichende
   Bestimmungen sind dann nicht erlaubt, und das Protokoll gilt zugleich als
   Gesellschafterliste ([§ 2 Abs. 1a GmbHG][gmbhg2]).
3. **Geschäftsführer.** Die GmbH braucht mindestens einen; er muss eine
   natürliche, unbeschränkt geschäftsfähige Person sein, bestellt im Vertrag
   oder durch Beschluss ([§ 6 Abs. 1–3 GmbHG][gmbhg6]).
4. **Konto und Einlage.** Der Mensch eröffnet ein Konto der GmbH i. G. Die
   Gesellschafter zahlen mindestens die Beträge nach der Referenz zur
   Kapitalaufbringung ein (SKILL.md, Schritt 5).
5. **Anmeldung.** Die Geschäftsführer melden die GmbH beim Registergericht des
   Sitzes an ([§ 7 Abs. 1 GmbHG][gmbhg7]), elektronisch in öffentlich
   beglaubigter Form, also über den Notar ([§ 12 Abs. 1 HGB][hgb12]).
6. **Prüfung und Eintragung.** Das Gericht lehnt ab, wenn die GmbH nicht
   ordnungsgemäß errichtet und angemeldet ist oder Sacheinlagen nicht
   unwesentlich überbewertet sind ([§ 9c Abs. 1 GmbHG][gmbhg9c]). Eingetragen
   werden unter anderem Firma, Sitz, Geschäftsanschrift, Gegenstand,
   Stammkapital, Tag des Vertrags und Geschäftsführer
   ([§ 10 Abs. 1 GmbHG][gmbhg10]).

Der Notar schickt dem Finanzamt binnen zwei Wochen eine beglaubigte Abschrift
der Gründungsurkunde ([§ 54 Abs. 1, 2 EStDV][estdv54]). Das ersetzt den
Fragebogen zur steuerlichen Erfassung nicht.

## 2. Gesellschaftsvertrag

| Pflichtinhalt | Grundlage |
|---|---|
| Firma und Sitz | [§ 3 Abs. 1 Nr. 1 GmbHG][gmbhg3] |
| Gegenstand des Unternehmens | [§ 3 Abs. 1 Nr. 2 GmbHG][gmbhg3] |
| Betrag des Stammkapitals | [§ 3 Abs. 1 Nr. 3 GmbHG][gmbhg3] |
| Zahl und Nennbeträge der Geschäftsanteile je Gesellschafter | [§ 3 Abs. 1 Nr. 4 GmbHG][gmbhg3] |

- Die Firma enthält „Gesellschaft mit beschränkter Haftung“ oder eine
  verständliche Abkürzung ([§ 4 GmbHG][gmbhg4]). Bei einem Stammkapital unter
  25.000 € lautet der Zusatz „Unternehmergesellschaft (haftungsbeschränkt)“
  oder „UG (haftungsbeschränkt)“ ([§ 5a Abs. 1 GmbHG][gmbhg5a]).
- Nennbeträge lauten auf volle Euro; ihre Summe ist das Stammkapital
  ([§ 5 Abs. 2, 3 GmbHG][gmbhg5]).
- Sacheinlagen müssen mit Gegenstand und Nennbetrag im Vertrag stehen
  ([§ 5 Abs. 4 Satz 1 GmbHG][gmbhg5]).
- Für Buchfink wichtig: Klausel zum Gründungsaufwand (Referenz zu den
  Gründungskosten) und ein abweichendes Geschäftsjahr, falls vereinbart.

## 3. Anmeldung und Eintragung

Der Anmeldung liegen bei: Gesellschaftsvertrag, Legitimation der
Geschäftsführer, Gesellschafterliste, bei Sacheinlagen die Verträge, der
Sachgründungsbericht und Wertnachweise ([§ 8 Abs. 1 GmbHG][gmbhg8]). Die
Geschäftsführer versichern, dass die Einlagen bewirkt sind und endgültig zu
ihrer freien Verfügung stehen; bei erheblichen Zweifeln kann das Gericht
Einzahlungsbelege verlangen ([§ 8 Abs. 2 GmbHG][gmbhg8]). Anzugeben sind eine
inländische Geschäftsanschrift und die Vertretungsbefugnis
([§ 8 Abs. 4 GmbHG][gmbhg8]).

Lege Anmeldung, Gesellschafterliste und Einzahlungsbeleg mit `beleg_ablegen`
ab. Die Anmeldung erstellt der Notar; du prüfst nur, ob die eingetragenen
Beträge zu den Buchungen passen.

## 4. Behördenwege mit Fristen

`fristen` führt diese Wege als Pflichten. Frist, Weg, Nachweis und Folgen
stehen in buchfink-grundlagen, references/pflichten.md, im Abschnitt der Art:

| Pflicht | Art in `fristen` | Hinweis für die Gründung |
|---|---|---|
| Fragebogen zur steuerlichen Erfassung ([§ 137 AO][ao137], [§ 138 Abs. 1b, 4 AO][ao138]) | `fragebogen-steuerliche-erfassung` | an das Finanzamt am Ort der Geschäftsleitung ([§ 20 Abs. 1 AO][ao20]); Inhalt: Referenz zur steuerlichen Erfassung |
| Gewerbeanzeige ([§ 14 Abs. 1 Satz 1 GewO][gewo14]) | `gewerbeanmeldung` | die Gemeinde unterrichtet das Finanzamt ([§ 138 Abs. 1 Satz 1 AO][ao138]); Wertungsfrage bei rein freiberuflichem Gegenstand |
| Mitteilung an das Transparenzregister ([§ 20 Abs. 1 GwG][gwg20]) | `transparenzregister` | Pflicht der juristischen Person; die GmbH besteht als solche erst mit der Eintragung ([§ 11 Abs. 1 GmbHG][gmbhg11]); Berechtigte nach [§ 3 Abs. 2 GwG][gwg3], unten |
| Aufstellung der Eröffnungsbilanz ([§ 242 Abs. 1 HGB][hgb242]) | `eroeffnungsbilanz` | auf den Beurkundungstag; Prüfen und Unterzeichnen: [eroeffnungsbilanz.md](eroeffnungsbilanz.md) |
| E-Bilanz der Eröffnungsbilanz ([§ 5b Abs. 1 Satz 4 EStG][estg5b]) | `ebilanz-eroeffnung` | `ebilanz_exportieren` mit `bilanzart: eroeffnungsbilanz`; Übermittlung über einen Online-Dienst wie die E-Bilanz des Jahres |

Diese Wege führt `fristen` nicht:

| Pflicht | Stelle | Frist | Grundlage | Hinweis |
|---|---|---|---|---|
| Anzeige an die Gemeinde (Realsteuern) | Gemeinde | ein Monat | [§ 137 Abs. 1, 2 AO][ao137] | frag beim Gewerbeamt, ob die Gewerbeanzeige genügt |
| Meldung an den Unfallversicherungsträger (Berufsgenossenschaft) | zuständige Berufsgenossenschaft | binnen einer Woche nach Beginn des Unternehmens | [§ 192 Abs. 1 Satz 1 SGB VII][sgb7192] | gilt als erfüllt, wenn die Gewerbeanzeige binnen einer Woche erstattet wurde (Satz 2) |
| Betriebsnummer | Bundesagentur für Arbeit | vor der ersten Meldung zur Sozialversicherung | [§ 18i Abs. 1 SGB IV][sgb418i] | nur mit Beschäftigten; Lohn: buchfink-lohn |
| Kammerzugehörigkeit | Industrie- und Handelskammer | – | [§ 2 Abs. 1 IHKG][ihkg2] | entsteht mit Veranlagung zur Gewerbesteuer und Betriebsstätte im Bezirk; Beitragsbescheid als Beleg (Gruppe `beitraege`) |

Stand: Oktober 2026.

Wirtschaftlich berechtigt ist jede natürliche Person, die unmittelbar oder
mittelbar mehr als 25 % der Kapitalanteile hält oder der Stimmrechte
kontrolliert oder auf vergleichbare Weise Kontrolle ausübt
([§ 3 Abs. 2 Satz 1 GwG][gwg3]). Lässt sich keine solche Person ermitteln,
gilt der gesetzliche Vertreter als wirtschaftlich Berechtigter
([§ 3 Abs. 2 Satz 5 GwG][gwg3]).

## 5. Nach der Eintragung

- Geschäftsbriefe nennen Rechtsform, Sitz, Registergericht, Registernummer
  und alle Geschäftsführer ([§ 35a Abs. 1 GmbHG][gmbhg35a]). Prüfe
  Rechnungsvorlagen darauf (buchfink-rechnungen).
- Ändern sich Gesellschafter oder Beteiligungen, reichen die Geschäftsführer
  unverzüglich eine neue Gesellschafterliste ein ([§ 40 Abs. 1 GmbHG][gmbhg40]).
  `fristen` führt das als Pflicht `gesellschafterliste`, sobald der Anlass
  `gesellschafterwechsel` erfasst ist; das tust du nach Rückfrage mit
  `anlass_erfassen`.
- Registerdaten trägt der Mensch in der App ein; danach entfällt der Zusatz
  „i. G.“ in der Firma.

[ao20]: https://www.gesetze-im-internet.de/ao_1977/__20.html
[ao137]: https://www.gesetze-im-internet.de/ao_1977/__137.html
[ao138]: https://www.gesetze-im-internet.de/ao_1977/__138.html
[estdv54]: https://www.gesetze-im-internet.de/estdv_1955/__54.html
[estg5b]: https://www.gesetze-im-internet.de/estg/__5b.html
[gewo14]: https://www.gesetze-im-internet.de/gewo/__14.html
[gmbhg2]: https://www.gesetze-im-internet.de/gmbhg/__2.html
[gmbhg3]: https://www.gesetze-im-internet.de/gmbhg/__3.html
[gmbhg4]: https://www.gesetze-im-internet.de/gmbhg/__4.html
[gmbhg5]: https://www.gesetze-im-internet.de/gmbhg/__5.html
[gmbhg5a]: https://www.gesetze-im-internet.de/gmbhg/__5a.html
[gmbhg6]: https://www.gesetze-im-internet.de/gmbhg/__6.html
[gmbhg7]: https://www.gesetze-im-internet.de/gmbhg/__7.html
[gmbhg8]: https://www.gesetze-im-internet.de/gmbhg/__8.html
[gmbhg9c]: https://www.gesetze-im-internet.de/gmbhg/__9c.html
[gmbhg10]: https://www.gesetze-im-internet.de/gmbhg/__10.html
[gmbhg11]: https://www.gesetze-im-internet.de/gmbhg/__11.html
[gmbhg35a]: https://www.gesetze-im-internet.de/gmbhg/__35a.html
[gmbhg40]: https://www.gesetze-im-internet.de/gmbhg/__40.html
[gwg3]: https://www.gesetze-im-internet.de/gwg_2017/__3.html
[gwg20]: https://www.gesetze-im-internet.de/gwg_2017/__20.html
[hgb12]: https://www.gesetze-im-internet.de/hgb/__12.html
[hgb242]: https://www.gesetze-im-internet.de/hgb/__242.html
[ihkg2]: https://www.gesetze-im-internet.de/ihkg/__2.html
[sgb418i]: https://www.gesetze-im-internet.de/sgb_4/__18i.html
[sgb7192]: https://www.gesetze-im-internet.de/sgb_7/__192.html
