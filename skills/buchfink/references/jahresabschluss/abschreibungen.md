# Abschreibungen, Wertaufholung, Abgang

Rechtsstand: 17. Oktober 2026. Wortlaut von HGB und EStG am 17.10.2026
gelesen.

Ob ein Zugang Aufwand, geringwertiges Wirtschaftsgut, Sammelposten oder Anlage
ist, regelt [belege/anlagen-gwg.md](../belege/anlagen-gwg.md). Finanzanlagen
bewertet [beteiligungen/bewertung-veraeusserung.md](../beteiligungen/bewertung-veraeusserung.md).

## Inhalt

1. Planmäßige Abschreibung
2. Abschreibungslauf in Buchfink
3. Außerplanmäßige Abschreibung
4. Wertaufholung
5. Abgang
6. Anlagenspiegel
7. Steuerliche Abweichungen

## 1. Planmäßige Abschreibung

- Anlagen mit begrenzter Nutzung werden planmäßig über die voraussichtliche
  Nutzungsdauer abgeschrieben ([§ 253 Abs. 3 Satz 1, 2 HGB][hgb253]).
- Steuerliche Abschreibung (linear, degressiv), geringwertige Wirtschaftsgüter
  und Sammelposten:
  [belege/anlagen-gwg.md, Abschnitte 3 und 4](../belege/anlagen-gwg.md#3-wertgrenzen).
- Anlagen im Bau und geleistete Anzahlungen schreibt Buchfink nicht ab.

Wertungsfrage: die Nutzungsdauer. Buchfink schlägt Werte vor; die Regeln zur
Wahl stehen in [belege/anlagen-gwg.md](../belege/anlagen-gwg.md#4-nutzungsdauer-und-methode).
Eine Änderung bei einem vorhandenen Anlagegut weicht von der
Bewertungsmethode ab und braucht einen Grund
([Stetigkeit](ablauf.md#4-entscheidungsregeln)). Ob sie im Anhang zu
erläutern ist ([§ 284 Abs. 2 Nr. 2 HGB][hgb284]), entscheidet der Mensch.

## 2. Abschreibungslauf in Buchfink

1. `anlagen` (`geschaeftsjahr`). Prüfe je Zugang des Jahres: Datum,
   Anschaffungskosten netto, Nutzungsdauer, Methode. Aktivierungskandidaten
   (Buchungen auf Anlagekonten ohne Anlage) aktivierst du nach Rückfrage mit
   `abschlussbuchung` und `art: anlage_aktivieren`.
2. Frag den Menschen nach Anlagen, die verkauft, verschrottet, gestohlen oder
   nicht mehr auffindbar sind (Abschnitt 5).
3. `abschlussbuchung` mit `art: afa_lauf`, erst mit `nur_vorschau: true`;
   die Vorschau nennt je Anlage Betrag und Konto. Nach dem Ja bucht Buchfink
   aus der Kartei, auf Wunsch nur für `anlage_ids`.
4. `auswertung` (`art: pruefbericht`): kein Befund `depreciation_missing`.

Fertig, wenn der Prüflauf frei ist und das Anlagenverzeichnis zu den Salden
der Anlagenkonten (`auswertung`, `art: susa`) passt.

## 3. Außerplanmäßige Abschreibung

- Handelsrecht: Bei voraussichtlich dauernder Wertminderung ist auf den
  niedrigeren Stichtagswert abzuschreiben, auch bei nicht abnutzbaren Anlagen
  ([§ 253 Abs. 3 Satz 5 HGB][hgb253]). Bei Finanzanlagen darf auch bei
  vorübergehender Minderung abgeschrieben werden (Satz 6;
  [beteiligungen/bewertung-veraeusserung.md](../beteiligungen/bewertung-veraeusserung.md)).
- Steuerrecht: Der niedrigere Teilwert kann bei voraussichtlich dauernder
  Wertminderung angesetzt werden ([§ 6 Abs. 1 Nr. 1 Satz 2 EStG][estg6]). Teilwert
  ist der Betrag, den ein Erwerber des ganzen Betriebs für das Wirtschaftsgut
  ansetzen würde (Satz 3).
- Typische Fälle: Gerät defekt und nicht reparabel, Software nicht mehr
  nutzbar, Gegenstand nur noch eingeschränkt verwendbar.
- Buchfink verlangt einen Grund. Formuliere ihn mit Tatsache und Datum
  (z. B. „Notebook INV-0042 am 12.11.2026 irreparabel beschädigt,
  Kostenvoranschlag liegt als Beleg vor“).

Wertungsfrage: ob die Minderung dauernd ist. Leg die Tatsachen vor; der Mensch
entscheidet. Bei großen Beträgen Steuerberater.

## 4. Wertaufholung

- Fällt der Grund einer außerplanmäßigen Abschreibung weg, ist der niedrigere
  Wert nicht beizubehalten; ausgenommen der entgeltlich erworbene Geschäfts- oder
  Firmenwert ([§ 253 Abs. 5 HGB][hgb253]).
- Steuerlich bleibt der niedrigere Teilwert nur, wenn er jedes Jahr nachgewiesen
  wird ([§ 6 Abs. 1 Nr. 1 Satz 4 EStG][estg6]).
- Buchfink führt einen Bericht der Anlagen mit früherer außerplanmäßiger
  Abschreibung. Frag den Menschen je Anlage, ob der Grund fortbesteht. Die
  Zuschreibung, höchstens bis zu den fortgeführten Anschaffungskosten, bildet
  Buchfink noch nicht ab.

## 5. Abgang

- Verkauf: Die Rechnung an den Käufer stellst du als Ausgangsrechnung aus
  ([rechnungen/ablauf.md](../rechnungen/ablauf.md); Umsatzsteuer). Den Abgang
  aus der Kartei bildet Buchfink noch nicht ab; gebucht würde der Restbuchwert
  auf SKR04 4855 bei Buchgewinn, 6895 bei Buchverlust.
- Verschrottung, Diebstahl: Abgang ohne Erlös. Übernimmt ein Gesellschafter
  den Gegenstand ohne angemessenes Entgelt, prüfe eine verdeckte
  Gewinnausschüttung ([gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md)).
- Sammelposten: Ein Abgang mindert ihn nicht
  ([belege/anlagen-gwg.md, Abschnitt 3](../belege/anlagen-gwg.md#3-wertgrenzen));
  Buchfink lehnt einen solchen Abgang ab.

## 6. Anlagenspiegel

- Der Anhang stellt die Entwicklung des Anlagevermögens dar: Anschaffungskosten,
  Zugänge, Abgänge, Umbuchungen, Zuschreibungen, Abschreibungen
  ([§ 284 Abs. 3 HGB][hgb284]).
- Kleine Gesellschaften brauchen diese Angabe nicht
  ([§ 288 Abs. 1 Nr. 1 HGB][hgb288]); Kleinstgesellschaften ohne Anhang auch
  nicht. Buchfink erzeugt ihn trotzdem; er dient dir zur Abstimmung.

## 7. Steuerliche Abweichungen

- Die Sonderabschreibung nach [§ 7g Abs. 5 EStG][estg7g] (bis zu insgesamt
  40 % der Anschaffungs- oder Herstellungskosten, verteilt auf das Jahr der
  Anschaffung und die vier Folgejahre, neben der linearen oder degressiven
  Abschreibung; Stand: Oktober 2026) bucht Buchfink nicht in den
  handelsrechtlichen Abschluss. Sie steht als steuerlicher Wert an der Anlage, im
  Verzeichnis nach [§ 5 Abs. 1 Satz 2, 3 EStG][estg5] und in der Überleitung
  ([steuererklaerungen/kst.md](../steuererklaerungen/kst.md)). Voraussetzungen
  nach § 7g Abs. 6 EStG: Der Gewinn im Wirtschaftsjahr vor der Anschaffung
  überschreitet 200.000 € nicht
  (Abs. 6 Nr. 1 mit Abs. 1 Satz 2 Nr. 1), und das Wirtschaftsgut wird im Jahr
  der Anschaffung und im Folgejahr vermietet oder fast ausschließlich
  betrieblich in einer inländischen Betriebsstätte genutzt (Abs. 6 Nr. 2). Ob
  sie vorliegen, klärt der Mensch.
- Eine handelsrechtliche außerplanmäßige Abschreibung, die steuerlich nicht
  angesetzt wird, kommt in die
  [Liste Abschlussposten](abschlussbuchungen.md#4-liste-abschlussposten).

[hgb253]: https://www.gesetze-im-internet.de/hgb/__253.html
[hgb284]: https://www.gesetze-im-internet.de/hgb/__284.html
[hgb288]: https://www.gesetze-im-internet.de/hgb/__288.html
[estg5]: https://www.gesetze-im-internet.de/estg/__5.html
[estg6]: https://www.gesetze-im-internet.de/estg/__6.html
[estg7g]: https://www.gesetze-im-internet.de/estg/__7g.html
