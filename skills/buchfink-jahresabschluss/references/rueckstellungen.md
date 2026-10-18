# Rückstellungen

Rechtsstand: 17. Oktober 2026. Wortlaut von §§ 249, 253, 277, 285, 288 HGB,
§ 5 Abs. 4a und § 6 Abs. 1 Nr. 3a EStG und der Rückstellungsabzinsungsverordnung
(RückAbzinsV) am 17.10.2026 gelesen. Steuerrückstellungen: SKILL.md, Schritt 10.

## Inhalt

1. Wann eine Rückstellung zu bilden ist
2. Typische Fälle und Konten
3. Höhe
4. Abzinsung
5. Buchen
6. Verbrauch und Auflösung
7. Anhang

## 1. Wann eine Rückstellung zu bilden ist

Pflicht ([§ 249 Abs. 1 HGB][hgb249]):

- ungewisse Verbindlichkeiten;
- drohende Verluste aus schwebenden Geschäften;
- im Geschäftsjahr unterlassene Instandhaltung, die im folgenden Geschäftsjahr
  innerhalb von drei Monaten nachgeholt wird;
- Gewährleistungen ohne rechtliche Verpflichtung.

Für andere Zwecke sind Rückstellungen verboten. Aufgelöst wird nur, soweit der
Grund entfallen ist ([§ 249 Abs. 2 HGB][hgb249]).

Prüffragen für eine ungewisse Verbindlichkeit; alle müssen mit Ja beantwortet
sein:

1. Besteht eine Pflicht gegenüber einem Dritten, aus Vertrag oder Gesetz?
2. Hat sie ihre Ursache im abgelaufenen Jahr?
3. Ist ernsthaft damit zu rechnen, dass die GmbH in Anspruch genommen wird?
4. Ist Höhe oder Zeitpunkt ungewiss? Steht beides fest, ist es eine
   Verbindlichkeit (SKILL.md, Schritt 5).

Wertungsfrage: jede der vier Antworten. Leg Tatsachen und Belege vor; der Mensch
entscheidet.

## 2. Typische Fälle und Konten

Konten aus dem SKR04-Katalog von Buchfink:

| Fall | Rückstellung | Aufwand | Hinweis |
|---|---|---|---|
| Kosten des Jahresabschlusses, der Offenlegung und der Steuererklärungen für das abgelaufene Jahr | 3095 | 6827 | gesetzliche Pflicht ([§ 242][hgb242], [§ 325 HGB][hgb325]); Wertungsfrage, ob interne Kosten zählen |
| Aufbewahrung der Unterlagen des Jahres | 3096 | nach Kostenart | Wertungsfrage: Berechnung der künftigen Kosten mit dem Menschen |
| Nicht genommener Urlaub, Überstunden, Boni | 3079 Urlaub, 3074 Personal | 6076 Urlaub, sonst Personalaufwand | Daten vom Lohnbüro (buchfink-lohn) |
| Rechnung erwartet, Höhe unsicher | 3070 | Aufwandskonto der Leistung | netto, ohne Vorsteuer |
| Gewährleistung, Nacharbeit | 3090 | 6790 | Erfahrungswerte vergangener Jahre |
| Prozess, Schadensersatz, Vertragsstrafe | 3070 | 6825 Rechts- und Beratungskosten, sonst 6300 | Anwaltsschreiben als Beleg |
| Drohender Verlust aus einem Festpreisauftrag | 3092 | 6300 | steuerlich verboten (Abschnitt 4) |
| Unterlassene Instandhaltung, Nachholung bis Ende März | 3075 | Instandhaltung | nur bei Nachholung in drei Monaten |

Pensionsrückstellungen bildet Buchfink nicht ohne Gutachten. Bei einer
Pensionszusage: Steuerberater (buchfink-grundlagen, Grenzen).

## 3. Höhe

- Anzusetzen ist der nach vernünftiger kaufmännischer Beurteilung notwendige
  Erfüllungsbetrag ([§ 253 Abs. 1 Satz 2 HGB][hgb253]). Handelsrechtlich zählen
  Preise und Kosten im Zeitpunkt der Erfüllung.
- Steuerlich gilt höchstens ([§ 6 Abs. 1 Nr. 3a EStG][estg6]):
  - gleichartige Verpflichtungen nach Erfahrung der Vergangenheit (Buchst. a);
  - Sachleistungen mit Einzelkosten und angemessenen Gemeinkosten (Buchst. b);
  - künftige Vorteile mindern (Buchst. c);
  - Wertverhältnisse am Stichtag, ohne künftige Preis- und Kostensteigerungen
    (Buchst. f).
- Weichen beide Werte ab, bucht Buchfink den handelsrechtlichen. Den
  steuerlichen trägst du in die Liste Abschlussposten ein.
- Rechenweg und Quellen (Angebot, Vorjahreswert, Stundenliste) gehören in den
  Eigenbeleg.

## 4. Abzinsung

Handelsrecht ([§ 253 Abs. 2 Satz 1, 4 HGB][hgb253], [§ 1, § 7 RückAbzinsV][rueckabzinsv]):

- Restlaufzeit über einem Jahr: abzinsen mit dem durchschnittlichen
  Marktzinssatz der vergangenen sieben Geschäftsjahre für diese Restlaufzeit
  (Pensionen: zehn Jahre).
- Die Deutsche Bundesbank ermittelt die Sätze und veröffentlicht sie monatlich
  für Restlaufzeiten von einem bis 50 Jahren:
  [Abzinsungszinssätze, 7-Jahresdurchschnitt][bbk7]. Nimm den Satz des Monats,
  in dem der Stichtag liegt, für die auf ganze Jahre bezogene Restlaufzeit.
  Werte ändern sich jeden Monat; lies den Satz an der Quelle.
- Restlaufzeit bis ein Jahr: keine Abzinsung.

Steuerrecht ([§ 6 Abs. 1 Nr. 3a Buchst. e EStG][estg6]):

- 5,5 % für Verpflichtungen mit einer Laufzeit am Stichtag von mindestens zwölf
  Monaten. Ausgenommen sind verzinsliche Verpflichtungen und solche aus
  Anzahlungen oder Vorausleistungen. Bei Sachleistungen zählt die Zeit bis zum
  Beginn der Erfüllung. Stand: Oktober 2026.

Drohverlustrückstellungen sind steuerlich nicht zulässig
([§ 5 Abs. 4a Satz 1 EStG][estg5]). Rückstellungen für künftige
Anschaffungs- oder Herstellungskosten ebenfalls nicht (§ 5 Abs. 4b Satz 1
EStG). Beides gehört in die Liste Abschlussposten.

## 5. Buchen

Rückfrage, dann `abschlussbuchung` mit `art: rueckstellung` und
`rueckstellung` (`art`, `text`, `betrag` als Erfüllungsbetrag, `erwartet_am`,
`grund`, `aufwandskonto`). Rechne erst mit `nur_vorschau: true`; die Vorschau
zeigt Zeilen, Betrag und Befunde, etwa zur Abzinsung.

- Bildung: Soll Aufwandskonto, Haben Rückstellungskonto, Betrag nach
  Abzinsung.
- Aufzinsung in den Folgejahren: Soll 7362 Zinsaufwendungen aus der Abzinsung
  von Rückstellungen, Haben Rückstellungskonto. Ein Ertrag aus der Abzinsung
  geht auf 7142. Beides steht gesondert unter den Zinsen
  ([§ 277 Abs. 5 Satz 1 HGB][hgb277]).
- `grund`: Rechtsgrund, Prüffragen, Berechnung, Restlaufzeit, Zinssatz mit
  Monat der Veröffentlichung.

`abschlussbuchung` führt die Rückstellung in Buchfinks Kartei
(Rückstellungsspiegel). Eine Rückstellung, die nur mit `buchung_erfassen`
gebucht ist, fehlt dort; trag sie in die Liste Abschlussposten ein.

## 6. Verbrauch und Auflösung

- Kommt im Folgejahr die Rechnung, bucht buchfink-belege sie. Führt Buchfink
  die Rückstellung in seiner Kartei, verbraucht `rueckstellung_id` sie dabei.
  Sonst löst du die Rückstellung mit `buchung_erfassen` gegen dasselbe
  Aufwandskonto auf, bis zur Höhe der Rechnung.
- Bleibt ein Rest, weil weniger anfiel: Soll Rückstellung, Haben 4930 Erträge
  aus der Auflösung von Rückstellungen. Fiel mehr an, ist der Rest Aufwand des
  neuen Jahres.
- Aufgelöst wird nur, wenn der Grund entfallen ist
  ([§ 249 Abs. 2 Satz 2 HGB][hgb249]). Prüfe zu jedem Stichtag jede bestehende
  Rückstellung (`auswertung`, `art: kontoblatt`, Konten 3070 bis 3096).

## 7. Anhang

Sonstige Rückstellungen von nicht unerheblichem Umfang sind zu erläutern
([§ 285 Nr. 12 HGB][hgb285]); kleine Gesellschaften sind davon befreit
([§ 288 Abs. 1 Nr. 1 HGB][hgb288]).

[hgb242]: https://www.gesetze-im-internet.de/hgb/__242.html
[hgb249]: https://www.gesetze-im-internet.de/hgb/__249.html
[hgb253]: https://www.gesetze-im-internet.de/hgb/__253.html
[hgb277]: https://www.gesetze-im-internet.de/hgb/__277.html
[hgb285]: https://www.gesetze-im-internet.de/hgb/__285.html
[hgb288]: https://www.gesetze-im-internet.de/hgb/__288.html
[hgb325]: https://www.gesetze-im-internet.de/hgb/__325.html
[estg5]: https://www.gesetze-im-internet.de/estg/__5.html
[estg6]: https://www.gesetze-im-internet.de/estg/__6.html
[rueckabzinsv]: https://www.gesetze-im-internet.de/r_ckabzinsv/__1.html
[bbk7]: https://www.bundesbank.de/de/statistiken/geld-und-kapitalmaerkte/zinssaetze-und-renditen/abzinsungszinssaetze-gemaess-253-abs-2-hgb-7-jahresdurchschnitt-650652
