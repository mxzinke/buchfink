# UG-Rücklage, Ergebnisverwendung, Vorträge

Rechtsstand: 17. Oktober 2026.

Die Ausschüttung selbst (Kapitalertragsteuer, Auszahlung, Bescheinigung):
[gesellschafter/ausschuettung.md](../gesellschafter/ausschuettung.md).

## Inhalt

1. Gesetzliche Rücklage der UG
2. Rücklage und Festschreibung in Buchfink
3. Ergebnisverwendung
4. Buchen im Folgejahr
5. Verlust

## 1. Gesetzliche Rücklage der UG

- Die UG (Stammkapital unter 25.000 €, [§ 5 Abs. 1][gmbhg5],
  [§ 5a Abs. 1 GmbHG][gmbhg5a]) stellt ein Viertel des um den Verlustvortrag
  aus dem Vorjahr geminderten Jahresüberschusses in eine gesetzliche Rücklage
  ein, ohne Obergrenze ([§ 5a Abs. 3 Satz 1 GmbHG][gmbhg5a]). Bei
  Jahresfehlbetrag oder aufgezehrtem Überschuss entfällt die Einstellung.
- Verwenden darf sie die Rücklage nur für eine Kapitalerhöhung aus
  Gesellschaftsmitteln oder zum Ausgleich eines ungedeckten Jahresfehlbetrags
  oder Verlustvortrags (§ 5a Abs. 3 Satz 2), nie für eine Ausschüttung. Ab
  25.000 € Stammkapital entfällt die Pflicht (§ 5a Abs. 5).
- `abschlussbuchung` mit `art: ug_ruecklage` bucht sie zum Stichtag: 7765 an
  2930, in der GuV nach dem Jahresüberschuss ([§ 275 Abs. 4 HGB][hgb275]).
- Die Bilanz ist damit unter teilweiser Ergebnisverwendung aufgestellt und
  zeigt Bilanzgewinn oder Bilanzverlust ([§ 268 Abs. 1 HGB][hgb268],
  [§ 270 Abs. 2 HGB][hgb270]); prüfe das in `auswertung` (`art: bilanz`).

## 2. Rücklage und Festschreibung in Buchfink

Buchfink verlangt die Rücklage der UG schon für die Festschreibung des
Dezembers und wieder vor Aufstellen und Feststellen:

1. Vor der Dezember-Festschreibung die Rücklage aus dem vorläufigen Ergebnis
   buchen (Rückfrage).
2. Nach der Steuerrückstellung mit `nur_vorschau: true` neu rechnen; eine
   Differenz nach Rückfrage buchen.
3. Vor dem Aufstellen mit `auswertung` prüfen, dass kein Befund zur Rücklage
   bleibt.

## 3. Ergebnisverwendung

- Die Gesellschafter beschließen sie mit der Feststellung
  ([aufstellen-feststellen-offenlegen.md](aufstellen-feststellen-offenlegen.md#3-feststellen)).
- Anspruch: Jahresüberschuss plus Gewinnvortrag minus Verlustvortrag, bei
  teilweiser Ergebnisverwendung der Bilanzgewinn ([§ 29 Abs. 1 GmbHG][gmbhg29]).
  Der Beschluss kann Beträge in Gewinnrücklagen einstellen oder vortragen
  (Abs. 2); verteilt wird nach Geschäftsanteilen (Abs. 3). Satzung und
  Beschluss gehen vor; lies die Satzung, wenn sie abgelegt ist.
- Kleine Gesellschaften machen keine Anhangangabe dazu
  ([§ 285 Nr. 34][hgb285], [§ 288 Abs. 1 Nr. 1 HGB][hgb288]).
- Ausschüttung, Rücklage oder Vortrag entscheiden die Gesellschafter; Grenzen
  der Ausschüttung: [gesellschafter/ausschuettung.md](../gesellschafter/ausschuettung.md).

## 4. Buchen im Folgejahr

Der Saldenvortrag bringt das Ergebnis auf 2970 bzw. 2978 (Gewinn- bzw.
Verlustvortrag vor Verwendung). Buche den abgelegten Beschluss zu seinem Datum
mit `ergebnisverwendung_buchen`.

| Beschluss | Buchung |
|---|---|
| Einstellung in andere Gewinnrücklagen | Soll 2970, Haben 2960 Andere Gewinnrücklagen |
| Ausschüttung | nach [gesellschafter/ausschuettung.md, Abschnitt 5](../gesellschafter/ausschuettung.md#5-buchungen-in-buchfink) |
| Vortrag auf neue Rechnung | keine Buchung, bleibt auf 2970 |

Fertig, wenn die Buchungen dem Beschluss entsprechen und der Rest auf 2970 dem
beschlossenen Vortrag.

## 5. Verlust

- Ein Jahresfehlbetrag bleibt auf 2978, bis ein Gewinn oder Beschluss ihn
  ausgleicht. Ausgleich aus Rücklagen: Entnahme in der GuV nach dem
  Jahresüberschuss (7735 gesetzliche Rücklage, 7750 andere Gewinnrücklagen),
  nur auf Beschluss und nach Rückfrage.
- Übersteigen die Verluste das Eigenkapital, steht auf der Aktivseite ein nicht
  durch Eigenkapital gedeckter Fehlbetrag ([§ 268 Abs. 3 HGB][hgb268]). Melde
  ihn sofort als
  [Warnzeichen](../grundlagen/grenzen.md#2-warnzeichen-die-du-sofort-meldest).

[gmbhg5]: https://www.gesetze-im-internet.de/gmbhg/__5.html
[gmbhg5a]: https://www.gesetze-im-internet.de/gmbhg/__5a.html
[gmbhg29]: https://www.gesetze-im-internet.de/gmbhg/__29.html
[hgb268]: https://www.gesetze-im-internet.de/hgb/__268.html
[hgb270]: https://www.gesetze-im-internet.de/hgb/__270.html
[hgb275]: https://www.gesetze-im-internet.de/hgb/__275.html
[hgb285]: https://www.gesetze-im-internet.de/hgb/__285.html
[hgb288]: https://www.gesetze-im-internet.de/hgb/__288.html
