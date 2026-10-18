# UG-Rücklage, Ergebnisverwendung, Vorträge

Rechtsstand: 17. Oktober 2026. Wortlaut von §§ 5, 5a, 29, 42a, 46, 48 GmbHG und
§§ 268, 270, 275, 285, 288 HGB am 17.10.2026 gelesen. Buchfink-Verhalten aus
`internal/service/ug_reserve.go`, `closing_service.go` und
`internal/wailsbridge/festschreibung_service.go`. Die Ausschüttung selbst
(Kapitalertragsteuer, Auszahlung, Bescheinigung) regelt buchfink-gesellschafter.

## Inhalt

1. Gesetzliche Rücklage der UG
2. Rücklage und Festschreibung in Buchfink
3. Ergebnisverwendung
4. Buchen im Folgejahr
5. Verlust

## 1. Gesetzliche Rücklage der UG

- Eine UG (Stammkapital unter dem Mindeststammkapital von 25.000 €,
  [§ 5 Abs. 1 GmbHG][gmbhg5], [§ 5a Abs. 1 GmbHG][gmbhg5a]) bildet in der Bilanz
  des Jahresabschlusses eine gesetzliche Rücklage. In sie geht ein Viertel des
  um einen Verlustvortrag aus dem Vorjahr geminderten Jahresüberschusses
  ([§ 5a Abs. 3 Satz 1 GmbHG][gmbhg5a]). Eine Obergrenze nennt der Wortlaut
  nicht.
- Verwenden darf die UG die Rücklage nur für eine Kapitalerhöhung aus
  Gesellschaftsmitteln, zum Ausgleich eines Jahresfehlbetrags, soweit kein
  Gewinnvortrag ihn deckt, und zum Ausgleich eines Verlustvortrags, soweit kein
  Jahresüberschuss ihn deckt (§ 5a Abs. 3 Satz 2). Ausschütten darf sie sie
  nicht.
- Erreicht oder übersteigt das Stammkapital nach einer Erhöhung 25.000 €, gilt
  die Pflicht nicht mehr ([§ 5a Abs. 5 GmbHG][gmbhg5a]).
- Rechnung: Jahresüberschuss laut GuV vor der Einstellung, minus
  Verlustvortrag aus dem Vorjahr, mal ein Viertel. Buchfink rundet auf den
  Cent auf. Bei Jahresfehlbetrag oder wenn der Verlustvortrag den Überschuss
  aufzehrt: keine Einstellung.
- Buchung: Soll 7765 Einstellungen in die gesetzliche Rücklage, Haben 2930
  Gesetzliche Rücklage, Datum Stichtag; `abschlussbuchung` mit
  `art: ug_ruecklage` rechnet und bucht sie. Die Einstellung steht
  in der GuV nach dem Jahresüberschuss ([§ 275 Abs. 4 HGB][hgb275]).
- Weil die Rücklage schon bei der Aufstellung gebildet wird, ist die Bilanz
  unter teilweiser Ergebnisverwendung aufgestellt: An die Stelle von
  Jahresüberschuss und Gewinn- oder Verlustvortrag tritt der Bilanzgewinn oder
  Bilanzverlust ([§ 268 Abs. 1 HGB][hgb268], [§ 270 Abs. 2 HGB][hgb270]). Prüfe
  in `auswertung` (`art: bilanz`), ob Buchfink den Posten so zeigt; sonst
  Hinweis an den Menschen.

## 2. Rücklage und Festschreibung in Buchfink

- Buchfink prüft die Rücklage, bevor ein Zeitraum bis zum Jahresende
  festgeschrieben wird, und vor Aufstellen und Feststellen. Fehlt sie oder
  stimmt ihr Betrag nicht, lehnt Buchfink ab.
- Folge: Bei einer UG verlangt schon die Festschreibung des Dezembers die
  Rücklage, obwohl das Ergebnis erst nach allen Abschlussbuchungen feststeht.
  Weg, bis Buchfink die Prüfung an die Jahresfestschreibung bindet:
  1. Vor der Dezember-Festschreibung die Rücklage aus dem vorläufigen Ergebnis
     buchen (`abschlussbuchung`, `art: ug_ruecklage`, Rückfrage).
  2. Nach der Steuerrückstellung mit `nur_vorschau: true` neu rechnen; die
     Vorschau nennt, was einzustellen ist und was schon gebucht ist. Bleibt
     eine Differenz, buche sie nach Rückfrage.
  3. Vor dem Aufstellen mit `auswertung` prüfen, dass kein Befund zur Rücklage
     bleibt.

## 3. Ergebnisverwendung

- Die Gesellschafter beschließen über die Feststellung und die Verwendung des
  Ergebnisses ([§ 46 Nr. 1 GmbHG][gmbhg46]), spätestens acht bzw. bei kleinen
  Gesellschaften elf Monate nach Ende des Geschäftsjahres
  ([§ 42a Abs. 2 GmbHG][gmbhg42a]).
- Anspruch haben sie auf den Jahresüberschuss zuzüglich Gewinnvortrag und
  abzüglich Verlustvortrag, bei teilweiser Ergebnisverwendung auf den
  Bilanzgewinn, soweit Gesetz, Satzung oder Beschluss nichts anderes bestimmen
  ([§ 29 Abs. 1 GmbHG][gmbhg29]).
- Der Beschluss kann Beträge in Gewinnrücklagen einstellen oder als Gewinn
  vortragen, wenn die Satzung nichts anderes bestimmt (§ 29 Abs. 2). Verteilt
  wird nach Geschäftsanteilen, sofern die Satzung keinen anderen Maßstab nennt
  (§ 29 Abs. 3).
- Form: Versammlung, Textform mit Zustimmung aller oder bei einem
  Alleingesellschafter eine unverzüglich unterschriebene Niederschrift
  ([§ 48 GmbHG][gmbhg48]). Lies die Satzung, wenn der Mensch sie ablegt.
- Anhang: Vorschlag oder Beschluss über die Ergebnisverwendung
  ([§ 285 Nr. 34 HGB][hgb285]); kleine Gesellschaften sind davon befreit
  ([§ 288 Abs. 1 Nr. 1 HGB][hgb288]).

Wertungsfrage: Ausschüttung, Rücklage oder Vortrag. Die Entscheidung treffen
die Gesellschafter; Grenzen der Ausschüttung (Kapitalerhaltung,
Zahlungsfähigkeit, UG-Rücklage) prüft buchfink-gesellschafter.

## 4. Buchen im Folgejahr

Der Saldenvortrag (nur App) bringt das Jahresergebnis unverwendet auf 2970
Gewinnvortrag vor Verwendung bzw. 2978 Verlustvortrag vor Verwendung. Der
Beschluss verteilt es an seinem Datum:

| Beschluss | Buchung |
|---|---|
| Einstellung in andere Gewinnrücklagen | Soll 2970, Haben 2960 Andere Gewinnrücklagen |
| Ausschüttung | Soll 2970, Haben 3519 (Nettobetrag) und 3760 (einbehaltene Steuern); Beträge nach buchfink-gesellschafter |
| Vortrag auf neue Rechnung | keine Buchung, bleibt auf 2970 |

- Buche den Beschluss am Beschlusstag nach Rückfrage mit
  `ergebnisverwendung_buchen`; Beleg ist der abgelegte Beschluss. Das Werkzeug
  setzt voraus, dass die Gesellschafter beschlossen haben.
- Fertig, wenn die Summe der Buchungen dem Beschluss entspricht und der Rest
  auf 2970 dem beschlossenen Vortrag.

## 5. Verlust

- Ein Jahresfehlbetrag bleibt als Verlustvortrag auf 2978, bis ein späterer
  Gewinn oder ein Beschluss ihn ausgleicht.
- Ein Ausgleich aus Rücklagen ist eine Entnahme aus Gewinnrücklagen in der GuV
  nach dem Jahresüberschuss (SKR04 7735 gesetzliche Rücklage, 7750 andere
  Gewinnrücklagen). Wertungsfrage; leg ihn nur auf Beschluss und nach Rückfrage
  vor.
- Übersteigen die Verluste das Eigenkapital, steht auf der Aktivseite ein nicht
  durch Eigenkapital gedeckter Fehlbetrag ([§ 268 Abs. 3 HGB][hgb268]).
  Warnzeichen und Pflichten (Einberufung, Insolvenzantrag): buchfink-grundlagen,
  Grenzen. Melde sie sofort.

[gmbhg5]: https://www.gesetze-im-internet.de/gmbhg/__5.html
[gmbhg5a]: https://www.gesetze-im-internet.de/gmbhg/__5a.html
[gmbhg29]: https://www.gesetze-im-internet.de/gmbhg/__29.html
[gmbhg42a]: https://www.gesetze-im-internet.de/gmbhg/__42a.html
[gmbhg46]: https://www.gesetze-im-internet.de/gmbhg/__46.html
[gmbhg48]: https://www.gesetze-im-internet.de/gmbhg/__48.html
[hgb268]: https://www.gesetze-im-internet.de/hgb/__268.html
[hgb270]: https://www.gesetze-im-internet.de/hgb/__270.html
[hgb275]: https://www.gesetze-im-internet.de/hgb/__275.html
[hgb285]: https://www.gesetze-im-internet.de/hgb/__285.html
[hgb288]: https://www.gesetze-im-internet.de/hgb/__288.html
