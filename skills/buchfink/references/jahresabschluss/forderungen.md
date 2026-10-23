# Forderungen, Verbindlichkeiten und Fremdwährung zum Stichtag

Rechtsstand: 17. Oktober 2026.

Ausfall im laufenden Jahr:
[rechnungen/forderungsausfall.md](../rechnungen/forderungsausfall.md).

## Inhalt

1. Abstimmen
2. Einzelwertberichtigung
3. Pauschalwertberichtigung
4. Fremdwährung
5. Ausweis und Restlaufzeiten
6. Folgejahr

## 1. Abstimmen

1. `offene_posten` mit `stichtag` für `richtung: forderung` und
   `richtung: verbindlichkeit`; die Summe je Liste gleich dem Saldo von 1200
   bzw. 3300 (`auswertung`, `art: susa`).
2. Überfällige Posten (`nur_ueberfaellig: true`), Gutschriften ohne
   Gegenposten und Minusbeträge legst du dem Menschen vor.

Fertig, wenn beide Listen mit den Konten übereinstimmen und jeder auffällige
Posten erklärt ist.

## 2. Einzelwertberichtigung

- Ist der beizulegende Wert einer Forderung am Stichtag niedriger, ist auf ihn
  abzuschreiben ([§ 253 Abs. 4 Satz 2 HGB][hgb253]), je Forderung einzeln
  ([§ 252 Abs. 1 Nr. 3 HGB][hgb252]). Steuerlich nur bei voraussichtlich
  dauernder Wertminderung ([§ 6 Abs. 1 Nr. 2 Satz 2 EStG][estg6]).
- [Wertaufhellung](ablauf.md#4-entscheidungsregeln): Zahlt der Kunde im
  Januar, bestand am Stichtag kein Ausfallrisiko in dieser Höhe.
- Bemessung netto; die Umsatzsteuer wird erst bei Uneinbringlichkeit
  berichtigt.
- Indizien: erfolglose Mahnungen, Mahnbescheid ohne Zahlung, bestrittene
  Forderung, Zahlungsschwierigkeiten, Insolvenzantrag. Steht der Ausfall fest
  (Insolvenzeröffnung, fruchtlose Vollstreckung), buche ihn nach
  [rechnungen/forderungsausfall.md](../rechnungen/forderungsausfall.md).
- Buchung: Soll 6923, Haben 1246 (Restlaufzeit bis ein Jahr) oder 1247 (über
  ein Jahr); die Forderung bleibt auf dem Personenkonto. `abschlussbuchung` mit
  `art: wertberichtigung`, je Forderung `einzeln`.
- Wertungsfrage: der erwartete Ausfall je Forderung. Leg Tatsachen, Alter,
  Mahnstand und Vorschlag vor; der Mensch entscheidet.

## 3. Pauschalwertberichtigung

- Für das allgemeine Ausfallrisiko der übrigen Forderungen ist eine
  Pauschalwertberichtigung zulässig (§ 253 Abs. 4 Satz 2, § 252 Abs. 1 Nr. 4
  HGB); steuerlich nur mit belegbarer Quote (§ 6 Abs. 1 Nr. 2 Satz 2 EStG).
- Quote: eigene Forderungsverluste der letzten Jahre (Konten 6930 bis 6938)
  im Verhältnis zu den Umsätzen, angewandt auf die Nettoforderungen ohne die
  einzeln berichtigten. Ohne eigene Ausfallgeschichte schlägst du keine Quote
  vor; dann bewerte einzeln. Ob und mit welcher Quote, entscheidet der Mensch.
- Buchung: Soll 6920, Haben 1248 bzw. 1249; `abschlussbuchung` mit
  `art: wertberichtigung` und `pauschal_promille`.

## 4. Fremdwährung

- Umrechnung zum Devisenkassamittelkurs am Stichtag
  ([§ 256a Satz 1 HGB][hgb256a]). Restlaufzeit bis ein Jahr: auch ein
  Kursgewinn über die Anschaffungskosten hinaus (Satz 2); über ein Jahr: nur
  der Verlust ([§ 252 Abs. 1 Nr. 4][hgb252], [§ 253 Abs. 1 Satz 1 HGB][hgb253]).
- `abschlussbuchung` mit `art: fremdwaehrung` bewertet alle offenen
  Fremdwährungsposten: Soll 6880 bzw. Haben 4840 gegen das Personenkonto;
  Ausweis unter den sonstigen betrieblichen Posten
  ([§ 277 Abs. 5 Satz 2 HGB][hgb277]).
- Steuerlich bleiben die Anschaffungskosten Obergrenze
  ([§ 6 Abs. 1 Nr. 2 Satz 1, Nr. 3 EStG][estg6]); ein Kursgewinn darüber kommt
  in die [Liste Abschlussposten](abschlussbuchungen.md#4-liste-abschlussposten).
- Der Saldenvortrag löst die Bewertung zum 1.1. auf; die realisierte
  Differenz bucht der [Zahlungsausgleich](../bank/skonto-differenzen.md).

## 5. Ausweis und Restlaufzeiten

- Zu vermerken sind je Posten Forderungen mit Restlaufzeit über einem Jahr
  und Verbindlichkeiten mit Restlaufzeit bis zu und über einem Jahr
  ([§ 268 Abs. 4 Satz 1, Abs. 5 Satz 1 HGB][hgb268]). Buchfink rechnet sie aus
  den offenen Posten; Darlehen ordnest du nach
  [bank/darlehen.md](../bank/darlehen.md) zu.
- Kleine Gesellschaften erläutern antizipative Posten nicht (§ 268 Abs. 4
  Satz 2, Abs. 5 Satz 3, [§ 274a Nr. 1, 2 HGB][hgb274a]).
- Posten gegenüber Gesellschaftern, auch als Kunden:
  [gesellschafter/ausweis.md](../gesellschafter/ausweis.md#1-regel).
- Forderungen und Verbindlichkeiten gegenüber demselben Partner werden nicht
  verrechnet ([§ 246 Abs. 2 Satz 1 HGB][hgb246]).

## 6. Folgejahr

- Einzelwertberichtigung: Zahlt der Kunde oder fällt die Forderung aus, löst
  du sie auf (Soll 1246, Haben 4923) und buchst den Ausfall nach
  [rechnungen/forderungsausfall.md](../rechnungen/forderungsausfall.md).
- Pauschalwertberichtigung: zum nächsten Stichtag neu berechnen; die Differenz
  geht auf 6920 oder 4920.

[hgb246]: https://www.gesetze-im-internet.de/hgb/__246.html
[hgb252]: https://www.gesetze-im-internet.de/hgb/__252.html
[hgb253]: https://www.gesetze-im-internet.de/hgb/__253.html
[hgb256a]: https://www.gesetze-im-internet.de/hgb/__256a.html
[hgb268]: https://www.gesetze-im-internet.de/hgb/__268.html
[hgb274a]: https://www.gesetze-im-internet.de/hgb/__274a.html
[hgb277]: https://www.gesetze-im-internet.de/hgb/__277.html
[estg6]: https://www.gesetze-im-internet.de/estg/__6.html
