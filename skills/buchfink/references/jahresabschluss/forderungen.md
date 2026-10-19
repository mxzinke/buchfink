# Forderungen, Verbindlichkeiten und Fremdwährung zum Stichtag

Rechtsstand: 17. Oktober 2026. Wortlaut von §§ 252, 253, 256a, 268, 274a, 277
HGB, § 6 EStG, § 17 UStG und § 42 GmbHG am 17.10.2026 gelesen. Den Ausfall
einer Forderung im laufenden Jahr regelt buchfink-rechnungen.

## Inhalt

1. Abstimmen
2. Einzelwertberichtigung
3. Pauschalwertberichtigung
4. Fremdwährung
5. Ausweis und Restlaufzeiten
6. Folgejahr

## 1. Abstimmen

1. `offene_posten` (`richtung: forderung`, `stichtag`) und
   (`richtung: verbindlichkeit`, `stichtag`).
2. Summe je Liste gleich Saldo des Sammelkontos 1200 bzw. 3300 laut
   `auswertung` (`art: susa`).
3. Überfällige Posten (`offene_posten`, `nur_ueberfaellig: true`), Gutschriften
   ohne Gegenposten und Posten mit Minusbetrag legst du dem Menschen vor.

Fertig, wenn beide Listen mit den Konten übereinstimmen und jeder auffällige
Posten erklärt ist.

## 2. Einzelwertberichtigung

- Forderungen gehören zum Umlaufvermögen. Ist ihr beizulegender Wert am
  Stichtag niedriger, ist auf diesen Wert abzuschreiben
  ([§ 253 Abs. 4 Satz 2 HGB][hgb253]). Bewertet wird jede Forderung einzeln
  ([§ 252 Abs. 1 Nr. 3 HGB][hgb252]).
- Risiken, die am Stichtag bestanden und bis zur Aufstellung bekannt werden,
  zählen mit ([§ 252 Abs. 1 Nr. 4 HGB][hgb252]). Zahlt der Kunde im Januar,
  bestand am Stichtag kein Ausfallrisiko in dieser Höhe.
- Steuerlich darf der niedrigere Teilwert nur bei voraussichtlich dauernder
  Wertminderung angesetzt werden ([§ 6 Abs. 1 Nr. 2 Satz 2 EStG][estg6]).
- Bemessung: der Nettobetrag. Die Umsatzsteuer wird erst berichtigt, wenn das
  Entgelt uneinbringlich ist ([§ 17 Abs. 2 Nr. 1 UStG][ustg17]); bis dahin ist
  sie nicht gefährdet, weil die GmbH sie dann zurückbekommt.
- Indizien: erfolglose Mahnungen, Mahnbescheid ohne Zahlung, Kunde bestreitet
  die Forderung, bekannte Zahlungsschwierigkeiten, Insolvenzantrag.
  Steht der Ausfall fest (Insolvenzeröffnung, fruchtlose Vollstreckung), wird
  die Forderung mit Umsatzsteuerkorrektur ausgebucht (buchfink-rechnungen).
- Buchung: Soll 6923 Einstellung in die Einzelwertberichtigung, Haben 1246
  (Restlaufzeit bis ein Jahr) oder 1247 (über ein Jahr). Die Forderung bleibt
  auf dem Personenkonto des Kunden. Werkzeug: `abschlussbuchung` mit
  `art: wertberichtigung` und je Forderung `einzeln` (`buchung_id`,
  `prozent`, `grund`).

Wertungsfrage: der erwartete Ausfall in Prozent je Forderung. Leg Tatsachen,
Alter, Mahnstand und Vorschlag vor; der Mensch entscheidet.

## 3. Pauschalwertberichtigung

- Für das allgemeine Ausfallrisiko der übrigen Forderungen kann eine
  Pauschalwertberichtigung angesetzt werden; Grundlage sind § 253 Abs. 4
  Satz 2 und § 252 Abs. 1 Nr. 4 HGB.
- Maßstab sind die tatsächlichen Ausfälle der Vergangenheit. Rechne die Quote
  aus den Forderungsverlusten der letzten Jahre (`auswertung`, Konten 6930 bis 6938)
  im Verhältnis zu den Umsätzen.
- Bemessung: Nettobetrag der Forderungen ohne die einzeln berichtigten.
- Buchung: Soll 6920 Einstellung in die Pauschalwertberichtigung, Haben 1248
  bzw. 1249. Werkzeug: `abschlussbuchung` mit `art: wertberichtigung` und
  `pauschal_promille`.
- Steuerlich gilt der Teilwert mit voraussichtlich dauernder Wertminderung
  (§ 6 Abs. 1 Nr. 2 Satz 2 EStG); die Quote muss sich belegen lassen.

Wertungsfrage: ob überhaupt und mit welcher Quote. Eine Quote ohne Beleg aus
eigenen Ausfällen schlägst du nicht vor. Bei wenigen Kunden oder ohne
Ausfallgeschichte ist die Einzelbewertung der bessere Weg.

## 4. Fremdwährung

- Forderungen und Verbindlichkeiten in fremder Währung werden zum
  Devisenkassamittelkurs am Stichtag umgerechnet ([§ 256a Satz 1 HGB][hgb256a]).
- Restlaufzeit bis ein Jahr: auch ein Kursgewinn über die Anschaffungskosten
  hinaus wird gebucht (§ 256a Satz 2 HGB). Restlaufzeit über ein Jahr: nur der
  Verlust ([§ 252 Abs. 1 Nr. 4 HGB][hgb252], [§ 253 Abs. 1 Satz 1 HGB][hgb253]).
- Konten: Soll 6880 Aufwendungen aus der Währungsumrechnung bzw. Haben 4840
  Erträge aus der Währungsumrechnung; Ausweis unter den sonstigen betrieblichen
  Posten ([§ 277 Abs. 5 Satz 2 HGB][hgb277]). Gegenkonto ist das Personenkonto
  des Kunden oder Lieferanten.
- Buchfink bewertet alle offenen Fremdwährungsposten mit `abschlussbuchung`
  und `art: fremdwaehrung` zum Kurs des Stichtags; die Vorschau nennt Kurs und
  Betrag. Die Auflösung kommt mit dem Saldenvortrag.
- Steuerlich bleibt es bei den Anschaffungskosten als Obergrenze
  ([§ 6 Abs. 1 Nr. 2 Satz 1, Nr. 3 EStG][estg6]); ein gebuchter Kursgewinn über
  die Anschaffungskosten hinaus kommt in die Liste Abschlussposten.
- Im Folgejahr löst du die Bewertung zum 1.1. auf (Gegenbuchung); der
  Zahlungsausgleich bucht dann die realisierte Differenz (buchfink-bank).

## 5. Ausweis und Restlaufzeiten

- Forderungen mit Restlaufzeit über einem Jahr und Verbindlichkeiten mit
  Restlaufzeit bis zu einem Jahr und über einem Jahr sind je Posten zu
  vermerken ([§ 268 Abs. 4 Satz 1, Abs. 5 Satz 1 HGB][hgb268]). Buchfink rechnet
  sie aus den offenen Posten; Darlehen ordnest du nach buchfink-bank zu.
- Erst nach dem Stichtag rechtlich entstehende Beträge unter den sonstigen
  Vermögensgegenständen oder Verbindlichkeiten sind bei größerem Umfang im
  Anhang zu erläutern (§ 268 Abs. 4 Satz 2, Abs. 5 Satz 3); kleine
  Gesellschaften sind davon befreit ([§ 274a Nr. 1, 2 HGB][hgb274a]).
- Forderungen und Verbindlichkeiten gegenüber Gesellschaftern sind gesondert
  auszuweisen oder im Anhang anzugeben ([§ 42 Abs. 3 GmbHG][gmbhg42]). Das gilt
  auch für Rechnungen an einen Gesellschafter als Kunden. Konten:
  buchfink-gesellschafter.
- Forderungen und Verbindlichkeiten gegenüber demselben Partner werden nicht
  verrechnet ([§ 246 Abs. 2 Satz 1 HGB][hgb246]).

## 6. Folgejahr

- Einzelwertberichtigung: Zahlt der Kunde oder fällt die Forderung aus,
  entfällt der Grund; die Berichtigung wird aufgelöst (Soll 1246, Haben 4923
  Erträge aus der Herabsetzung der Einzelwertberichtigung) und der Ausfall nach
  buchfink-rechnungen gebucht.
- Pauschalwertberichtigung: zum nächsten Stichtag neu berechnen; die
  Differenz geht auf 6920 oder 4920.
- Fremdwährung: Auflösung nach Abschnitt 4.

[hgb246]: https://www.gesetze-im-internet.de/hgb/__246.html
[hgb252]: https://www.gesetze-im-internet.de/hgb/__252.html
[hgb253]: https://www.gesetze-im-internet.de/hgb/__253.html
[hgb256a]: https://www.gesetze-im-internet.de/hgb/__256a.html
[hgb268]: https://www.gesetze-im-internet.de/hgb/__268.html
[hgb274a]: https://www.gesetze-im-internet.de/hgb/__274a.html
[hgb277]: https://www.gesetze-im-internet.de/hgb/__277.html
[estg6]: https://www.gesetze-im-internet.de/estg/__6.html
[ustg17]: https://www.gesetze-im-internet.de/ustg_1980/__17.html
[gmbhg42]: https://www.gesetze-im-internet.de/gmbhg/__42.html
