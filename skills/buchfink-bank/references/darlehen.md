# Bankdarlehen und Kontokorrent

Rechtsstand: 17. Oktober 2026. Wortlaut von HGB, EStG, KStG, GewStG und UStG am
17.10.2026 gelesen. Gesellschafterdarlehen (Vertrag mit `darlehen_erfassen`,
Bankumsatz mit `darlehen_id`) regelt buchfink-gesellschafter, Darlehen an
Tochtergesellschaften buchfink-beteiligungen.

## Inhalt

1. Belege
2. Konten
3. Auszahlung
4. Rate: Tilgung und Zinsen
5. Disagio und Bearbeitungsentgelt
6. Kontokorrent
7. Steuerliche Folgen, die du nur meldest

## 1. Belege

Lege Darlehensvertrag und Tilgungsplan mit `beleg_ablegen` ab, bevor die
Auszahlung gebucht wird. Der Vertrag ist Ursprungsbeleg für alle Folgebuchungen
([GoBD Rz. 81][gobd]). Die jährliche Zins- und Saldenbestätigung der Bank legst
du ebenfalls ab.

## 2. Konten

Aus dem SKR04:

| Konto | Inhalt |
|---|---|
| 3150 | Verbindlichkeiten gegenüber Kreditinstituten |
| 3151 | dto. Restlaufzeit bis 1 Jahr |
| 3160 | dto. Restlaufzeit 1 bis 5 Jahre |
| 3170 | dto. Restlaufzeit größer 5 Jahre |
| 7310 | Zinsaufwendungen für kurzfristige Verbindlichkeiten |
| 7318 | Zinsen auf Kontokorrentkonten |
| 7320 | Zinsaufwendungen für langfristige Verbindlichkeiten |
| 1940 | Damnum/Disagio |
| 7323 | Abschreibungen auf ein Agio oder Disagio/Damnum zur Finanzierung |

- Je Darlehen ein Konto der passenden Restlaufzeit. Hat die GmbH mehrere
  Darlehen, frag den Menschen, wie sie getrennt werden sollen (Text, Konto).
- In der Bilanz ist bei jedem Posten zu vermerken, was bis zu einem Jahr und
  was länger läuft ([§ 268 Abs. 5 Satz 1 HGB][hgb268]). Die Umgliederung zum
  Stichtag macht buchfink-jahresabschluss.
- Darlehen werden mit dem Erfüllungsbetrag angesetzt
  ([§ 253 Abs. 1 Satz 2 HGB][hgb253]), also mit dem Betrag, der zurückzuzahlen
  ist.

## 3. Auszahlung

Bankumsatz: Eingang des Auszahlungsbetrags. Werkzeug: `bankumsatz_buchen`
mit einer Zeile `konto` (Darlehenskonto) über den Nennbetrag. Behält die Bank
Disagio oder Entgelte ein, kommt je Abzug eine weitere Zeile mit negativem
`betrag` dazu (Abschnitt 5).

## 4. Rate: Tilgung und Zinsen

- Teile jede Rate nach Tilgungsplan oder Kontoauszug in Tilgung (Darlehenskonto)
  und Zinsen (7320 bzw. 7310). Die Tilgung ist kein Aufwand.
- Fehlt die Aufteilung, frag nach dem Tilgungsplan. Rechne sie nicht selbst
  aus, außer der Mensch bittet darum und bestätigt das Ergebnis.
- Kreditzinsen sind umsatzsteuerfrei ([§ 4 Nr. 8 Buchst. a UStG][ustg4]); die
  Zinszeile trägt keine Vorsteuer.
- Raten sind Rückfrage (`bankumsatz_buchen` mit je einer Zeile für Tilgung
  und Zinsen). Lege mehrere Raten gesammelt vor.

## 5. Disagio und Bearbeitungsentgelt

- Ist der Rückzahlungsbetrag höher als der Auszahlungsbetrag, darf der
  Unterschied als aktiver Rechnungsabgrenzungsposten angesetzt und über die
  Laufzeit planmäßig abgeschrieben werden ([§ 250 Abs. 3 HGB][hgb250]).
- Buchfink führt eine Einheitsbilanz (buchfink-grundlagen, Grenzen). Vorschlag:
  aktivieren (1940) und über die Laufzeit bzw. Zinsbindung auf 7323 auflösen.
  Ob die Steuerbilanz das verlangt, bestätigt der Steuerberater; die Frage
  ist hier nicht am Wortlaut geprüft. Wertungsfrage, Rückfrage.
- Bearbeitungsentgelte der Bank: Frag, ob sie laufzeitabhängig sind. Die
  Einordnung (Aufwand oder Abgrenzung) ist eine Wertungsfrage.
- Die jährliche Auflösung ist eine Abschlussbuchung (`abschlussbuchung` mit
  `art: abgrenzung` und `abgrenzung.art: disagio`; buchfink-jahresabschluss).

## 6. Kontokorrent

- Ein Bankkonto mit Sollsaldo (Dispositionskredit) bleibt auf seinem
  Bankkonto. Zinsen dafür buchst du auf 7318.
- Ein dauerhaft negativer Saldo ist ein Warnzeichen für die Liquidität. Melde ihn
  dem Menschen (buchfink-grundlagen, Grenzen).

## 7. Steuerliche Folgen, die du nur meldest

- **Gewerbesteuer:** Ein Viertel der Summe aus Entgelten für Schulden und
  weiteren Finanzierungsanteilen wird hinzugerechnet, soweit die Summe
  200.000 € übersteigt ([§ 8 Nr. 1 GewStG][gewstg8]). Stand: Oktober 2026.
  Rechnung in buchfink-steuererklaerungen. Buche Zinsen deshalb getrennt von
  Gebühren.
- **Zinsschranke:** Sie greift erst, wenn die Nettozinsaufwendungen 3 Mio. €
  erreichen ([§ 4h Abs. 2 Satz 1 Buchst. a EStG][estg4h],
  [§ 8a KStG][kstg8a]). Stand: Oktober 2026. Für die Zielgruppe von Buchfink
  selten; erreicht der Zinsaufwand diese Größenordnung, empfiehl einen
  Steuerberater.
- **Zinsen zum Stichtag:** Zinsen, die wirtschaftlich in das alte Jahr gehören,
  aber erst im neuen belastet werden, grenzt buchfink-jahresabschluss ab
  ([§ 252 Abs. 1 Nr. 5 HGB][hgb252]).

[gobd]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf
[hgb250]: https://www.gesetze-im-internet.de/hgb/__250.html
[hgb252]: https://www.gesetze-im-internet.de/hgb/__252.html
[hgb253]: https://www.gesetze-im-internet.de/hgb/__253.html
[hgb268]: https://www.gesetze-im-internet.de/hgb/__268.html
[estg4h]: https://www.gesetze-im-internet.de/estg/__4h.html
[kstg8a]: https://www.gesetze-im-internet.de/kstg_1977/__8a.html
[gewstg8]: https://www.gesetze-im-internet.de/gewstg/__8.html
[ustg4]: https://www.gesetze-im-internet.de/ustg_1980/__4.html
