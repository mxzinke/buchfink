# Abschreibungen, Wertaufholung, Abgang

Rechtsstand: 17. Oktober 2026.

Zugang als Aufwand, geringwertiges Wirtschaftsgut, Sammelposten oder Anlage:
[belege/anlagen-gwg.md](../belege/anlagen-gwg.md). Finanzanlagen:
[beteiligungen/bewertung-veraeusserung.md](../beteiligungen/bewertung-veraeusserung.md).

## Inhalt

1. Planmäßige Abschreibung
2. Abschreibungslauf in Buchfink
3. Außerplanmäßige Abschreibung
4. Wertaufholung
5. Abgang
6. Anlagenspiegel
7. Steuerliche Abweichungen

## 1. Planmäßige Abschreibung

- Anlagen mit begrenzter Nutzung schreibst du planmäßig über die
  voraussichtliche Nutzungsdauer ab ([§ 253 Abs. 3 Satz 1, 2 HGB][hgb253]).
  Steuerliche Methoden, geringwertige Wirtschaftsgüter und Sammelposten:
  [belege/anlagen-gwg.md, Abschnitte 3 und 4](../belege/anlagen-gwg.md#3-wertgrenzen).
- Anlagen im Bau und Anzahlungen schreibt Buchfink nicht ab und bucht sie bei
  Fertigstellung nicht um.
- Wertungsfrage: die Nutzungsdauer
  ([Regeln](../belege/anlagen-gwg.md#4-nutzungsdauer-und-methode)). Eine
  Änderung bei einem vorhandenen Anlagegut bricht die
  [Stetigkeit](ablauf.md#4-entscheidungsregeln) und braucht einen Grund; ob sie
  in den Anhang gehört ([§ 284 Abs. 2 Nr. 2 HGB][hgb284]), entscheidet der
  Mensch.

## 2. Abschreibungslauf in Buchfink

1. `anlagen` (`geschaeftsjahr`): je Zugang Datum, Anschaffungskosten netto,
   Nutzungsdauer und Methode prüfen. Aktivierungskandidaten aktivierst du nach
   Rückfrage mit `abschlussbuchung` (`art: anlage_aktivieren`).
2. Frag nach verkauften, verschrotteten, gestohlenen oder verschwundenen
   Anlagen (Abschnitt 5).
3. `abschlussbuchung` mit `art: afa_lauf`, Vorschau vorlegen, nach dem Ja
   buchen.

Fertig, wenn `auswertung` (`art: pruefbericht`) kein `depreciation_missing`
meldet und das Anlagenverzeichnis zu den Salden der Anlagenkonten
(`art: susa`) passt.

## 3. Außerplanmäßige Abschreibung

- Handelsrecht: Bei voraussichtlich dauernder Wertminderung ist auf den
  niedrigeren Stichtagswert abzuschreiben, auch bei nicht abnutzbaren Anlagen
  ([§ 253 Abs. 3 Satz 5 HGB][hgb253]); Finanzanlagen auch bei vorübergehender
  Minderung (Satz 6).
- Steuerrecht: Der niedrigere Teilwert darf bei voraussichtlich dauernder
  Wertminderung angesetzt werden ([§ 6 Abs. 1 Nr. 1 Satz 2, 3 EStG][estg6]).
- Buchfink bildet sie nicht ab. Leg dem Menschen Betrag, Stichtag und Grund mit
  Tatsache und Datum vor
  ([grundlagen/arbeitsweise.md, Abschnitt 9](../grundlagen/arbeitsweise.md#9-vorgänge-ohne-werkzeug)).
  Ob die Minderung dauernd ist, entscheidet er; bei großen Beträgen
  Steuerberater.

## 4. Wertaufholung

- Fällt der Grund weg, darf der niedrigere Wert nicht bleiben; ausgenommen der
  entgeltlich erworbene Geschäfts- oder Firmenwert
  ([§ 253 Abs. 5 HGB][hgb253]). Obergrenze sind die fortgeführten
  Anschaffungskosten ([§ 253 Abs. 1 Satz 1 HGB][hgb253]).
- Steuerlich bleibt der niedrigere Teilwert nur bei jährlichem Nachweis
  ([§ 6 Abs. 1 Nr. 1 Satz 4 EStG][estg6]).
- Frag den Menschen je Anlage mit früherer außerplanmäßiger Abschreibung
  (Bericht in der App), ob der Grund fortbesteht. Die Zuschreibung bildet
  Buchfink nicht ab.

## 5. Abgang

- Verkauf: Ausgangsrechnung an den Käufer
  ([rechnungen/ablauf.md](../rechnungen/ablauf.md)). Den Abgang aus der Kartei
  bildet Buchfink nicht ab; der Restbuchwert gehört auf 4855 bei Buchgewinn,
  6895 bei Buchverlust.
- Verschrottung, Diebstahl: Abgang ohne Erlös. Übernimmt ein Gesellschafter
  den Gegenstand ohne angemessenes Entgelt, prüfe eine verdeckte
  Gewinnausschüttung ([gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md)).
- Ein Abgang mindert den Sammelposten nicht
  ([belege/anlagen-gwg.md, Abschnitt 3](../belege/anlagen-gwg.md#3-wertgrenzen)).

## 6. Anlagenspiegel

Die Entwicklung des Anlagevermögens im Anhang ([§ 284 Abs. 3 HGB][hgb284])
brauchen kleine und Kleinstgesellschaften nicht
([§ 288 Abs. 1 Nr. 1 HGB][hgb288]). Buchfink erzeugt den Spiegel trotzdem;
nutze ihn zur Abstimmung.

## 7. Steuerliche Abweichungen

- Sonderabschreibung nach [§ 7g Abs. 5 EStG][estg7g]: bis zu 40 % der
  Anschaffungs- oder Herstellungskosten, verteilt auf Anschaffungsjahr und vier
  Folgejahre, neben der regulären Abschreibung. Voraussetzungen (Abs. 6):
  Gewinn im Vorjahr der Anschaffung höchstens 200.000 €; im Anschaffungs- und
  Folgejahr vermietet oder fast ausschließlich betrieblich in einer
  inländischen Betriebsstätte genutzt; der Mensch klärt sie. Buchfink führt sie
  nur als steuerlichen Wert an der Anlage, im Verzeichnis nach
  [§ 5 Abs. 1 Satz 2, 3 EStG][estg5] und in der
  [Überleitung](../steuererklaerungen/kst.md).
- Eine handelsrechtliche außerplanmäßige Abschreibung ohne steuerlichen Ansatz
  kommt in die [Liste Abschlussposten](abschlussbuchungen.md#4-liste-abschlussposten).

[hgb253]: https://www.gesetze-im-internet.de/hgb/__253.html
[hgb284]: https://www.gesetze-im-internet.de/hgb/__284.html
[hgb288]: https://www.gesetze-im-internet.de/hgb/__288.html
[estg5]: https://www.gesetze-im-internet.de/estg/__5.html
[estg6]: https://www.gesetze-im-internet.de/estg/__6.html
[estg7g]: https://www.gesetze-im-internet.de/estg/__7g.html
