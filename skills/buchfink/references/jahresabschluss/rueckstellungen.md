# Rückstellungen

Rechtsstand: 17. Oktober 2026.

Steuerrückstellungen: [steuern-im-abschluss.md](steuern-im-abschluss.md).

## Inhalt

1. Wann eine Rückstellung zu bilden ist
2. Typische Fälle und Konten
3. Höhe
4. Abzinsung
5. Buchen
6. Verbrauch und Auflösung
7. Anhang

## 1. Wann eine Rückstellung zu bilden ist

Pflicht ([§ 249 Abs. 1 HGB][hgb249]) für ungewisse Verbindlichkeiten, drohende
Verluste aus schwebenden Geschäften, im Geschäftsjahr unterlassene
Instandhaltung, die im Folgejahr innerhalb von drei Monaten nachgeholt wird,
und Gewährleistungen ohne rechtliche Verpflichtung. Für andere Zwecke sind
Rückstellungen verboten; aufgelöst wird nur, soweit der Grund entfallen ist
([§ 249 Abs. 2 HGB][hgb249]).

Ungewisse Verbindlichkeit, wenn alle vier Prüffragen mit Ja beantwortet sind:

1. Pflicht gegenüber einem Dritten aus Vertrag oder Gesetz?
2. Ursache im abgelaufenen Jahr?
3. Inanspruchnahme ernsthaft zu erwarten?
4. Höhe oder Zeitpunkt ungewiss? Steht beides fest, ist es eine
   Verbindlichkeit ([abgrenzung.md](abgrenzung.md#5-ausstehende-rechnungen-und-erträge)).

Jede Antwort ist Wertungsfrage: Leg Tatsachen und Belege vor, der Mensch
entscheidet.

## 2. Typische Fälle und Konten

| Fall | Rückstellung | Aufwand | Hinweis |
|---|---|---|---|
| Abschluss, Offenlegung, Steuererklärungen des Jahres | 3095 | 6827 | [§ 242][hgb242], [§ 325 HGB][hgb325]; ob interne Kosten zählen, ist Wertungsfrage |
| Aufbewahrung der Unterlagen des Jahres | 3096 | nach Kostenart | Kosten mit dem Menschen rechnen |
| Urlaub, Überstunden, Boni | 3079 Urlaub, 3074 Personal | 6076 Urlaub, sonst Personalaufwand | Daten vom [Lohnbüro](../lohn/ablauf.md) |
| Rechnung erwartet, Höhe unsicher | 3070 | Konto der Leistung | netto |
| Gewährleistung, Nacharbeit | 3090 | 6790 | Erfahrungswerte |
| Prozess, Schadensersatz, Vertragsstrafe | 3070 | 6825, sonst 6300 | Anwaltsschreiben |
| Drohverlust aus Festpreisauftrag | 3092 | 6300 | steuerlich verboten |
| Unterlassene Instandhaltung | 3075 | Instandhaltung | Nachholung bis Ende März |

Pensionszusage: [Steuerberater](../grundlagen/grenzen.md#3-wann-du-einen-steuerberater-empfiehlst).

## 3. Höhe

- Handelsrecht: der nach vernünftiger kaufmännischer Beurteilung notwendige
  Erfüllungsbetrag, mit Preisen und Kosten im Zeitpunkt der Erfüllung
  ([§ 253 Abs. 1 Satz 2 HGB][hgb253]).
- Buchfink bucht den handelsrechtlichen Wert; einen abweichenden steuerlichen
  Wert ([rueckstellungen-steuerlich.md](rueckstellungen-steuerlich.md#höhe))
  trägst du in die
  [Liste Abschlussposten](abschlussbuchungen.md#4-liste-abschlussposten) ein.
  Rechenweg und Quellen gehören in den Eigenbeleg.

## 4. Abzinsung

- Handelsrecht: Restlaufzeit über einem Jahr mit dem Durchschnittszins der
  vergangenen sieben Geschäftsjahre abzinsen
  ([§ 253 Abs. 2 Satz 1, 4 HGB][hgb253], [§ 1, § 7 RückAbzinsV][rueckabzinsv]);
  Satz der [Bundesbank][bbk7] für den Stichtagsmonat und die Restlaufzeit in
  ganzen Jahren.
- Steuerlich 5,5 % und steuerlich unzulässige Rückstellungen (Drohverlust):
  [rueckstellungen-steuerlich.md](rueckstellungen-steuerlich.md).

## 5. Buchen

Rückfrage, dann `abschlussbuchung` mit `art: rueckstellung`: Soll
Aufwandskonto, Haben Rückstellungskonto. `grund`: Rechtsgrund, Prüffragen,
Berechnung, Restlaufzeit, Zinssatz mit Monat der Veröffentlichung.

- Fehlt der Bundesbank-Satz (Befund der Vorschau), bucht Buchfink den
  Erfüllungsbetrag. Sag das dem Menschen und trag den abgezinsten Betrag mit
  Satz und Quelle in die
  [Liste Abschlussposten](abschlussbuchungen.md#4-liste-abschlussposten) ein.
- Aufzinsung in Folgejahren: Soll 7362, Haben Rückstellungskonto; Ertrag aus
  Abzinsung auf 7142; beides gesondert unter den Zinsen
  ([§ 277 Abs. 5 Satz 1 HGB][hgb277]). Für Rückstellungen aus Buchfinks Kartei
  bietet `abschlussbuchung` die Bewegungen `zufuehrung`, `aufloesung` und
  `aufzinsung` mit `id`, `datum` und `grund`. Zuerst `nur_vorschau: true`.
  Die automatische Aufzinsung verwendet den gespeicherten Satz; geänderte
  Marktzinsen müssen gesondert geprüft werden.
- Eine nur mit `buchung_erfassen` gebuchte Rückstellung fehlt im
  Rückstellungsspiegel; trag sie in die Liste Abschlussposten ein.

Ein Satz zur Bildung oder Zuführung wird in `zinssatz.rate` mit `month`,
`years`, `average: 7` und `rateMicros` eingegeben (15000 = 1,5 %).
`zinssatz.source` nennt die HTTPS-Fundstelle der Bundesbank.
Monat und Laufzeit müssen zur Bewertung passen.

## 6. Verbrauch und Auflösung

Beim Buchen einer Eingangsrechnung `rueckstellung_id` mitgeben: gedeckter
Nettobetrag gegen die Rückstellung, Mehrbetrag gegen Aufwand.
`abschluss_status` liefert Kennungen und Spiegel einschließlich Vorjahresbestand.
Auflösung nur mit dokumentiertem Wegfall des Grundes ([§ 249 Abs. 2 HGB][hgb249]).
Steuerliche Ausnahmen von der Abzinsung bleiben gesondert zu prüfen.

## 7. Anhang

Kleine Gesellschaften erläutern sonstige Rückstellungen nicht
([§ 285 Nr. 12][hgb285], [§ 288 Abs. 1 Nr. 1 HGB][hgb288]).

[hgb242]: https://www.gesetze-im-internet.de/hgb/__242.html
[hgb249]: https://www.gesetze-im-internet.de/hgb/__249.html
[hgb253]: https://www.gesetze-im-internet.de/hgb/__253.html
[hgb277]: https://www.gesetze-im-internet.de/hgb/__277.html
[hgb285]: https://www.gesetze-im-internet.de/hgb/__285.html
[hgb288]: https://www.gesetze-im-internet.de/hgb/__288.html
[hgb325]: https://www.gesetze-im-internet.de/hgb/__325.html
[rueckabzinsv]: https://www.gesetze-im-internet.de/r_ckabzinsv/__1.html
[bbk7]: https://www.bundesbank.de/de/statistiken/geld-und-kapitalmaerkte/zinssaetze-und-renditen/abzinsungszinssaetze-gemaess-253-abs-2-hgb-7-jahresdurchschnitt-650652
