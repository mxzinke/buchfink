# E-Bilanz

Rechtsstand: 17. Oktober 2026.

Frist, Nachweis und Folgen:
[grundlagen/pflichten.md#ebilanz](../grundlagen/pflichten.md#ebilanz).

## Inhalt

1. Pflicht und Inhalt
2. Taxonomie und Mindestumfang
3. Was Buchfink liefert
4. Prüfpunkte
5. Übermittlungsweg
6. Eröffnungsbilanz

## 1. Pflicht und Inhalt

Bilanz und GuV gehen nach amtlichem Datensatz elektronisch an das Finanzamt
([§ 5b Abs. 1 Satz 1 EStG][estg5b]), für Wirtschaftsjahre ab 2025 mit
unverdichteten Kontennachweisen, ab 2028 auch mit Anlagenspiegel,
Anlagenverzeichnis, Anhang, Lage- und Prüfungsbericht
([§ 52 Abs. 11 EStG][estg52]).

- Den Anlagenspiegel verlangt die Taxonomie schon jetzt als Mussfeld
  ([BMF 10.06.2025][bmf69]).
- Steuerlich abweichende Ansätze: Überleitungsrechnung oder Steuerbilanz
  ([§ 5b Abs. 1 Satz 2, 3 EStG][estg5b]).

## 2. Taxonomie und Mindestumfang

| Wirtschaftsjahr beginnt | Taxonomie | Fundstelle |
|---|---|---|
| 2025 | 6.8; 6.9 wird nicht beanstandet | [BMF 27.05.2024][bmf68], [BMF 10.06.2025][bmf69] |
| 2026 | 6.9; 6.10 wird nicht beanstandet | [BMF 10.06.2025][bmf69]; Schreiben vom 08.06.2026 laut Sekundärquelle |
| 2027 | 6.10 | BMF vom 08.06.2026, IV C 6 - S 2133-b/00067/002/023, laut Sekundärquelle |

Mussfelder füllt Buchfink mit Wert oder ohne Wert (NIL)
([BMF 27.05.2024][bmf68]).

## 3. Was Buchfink liefert

`ebilanz_exportieren` schreibt die XBRL-Instanz mit Kontennachweis für den
Import in einen Online-Dienst (Abschnitt 5); `taxonomie` wählt eine andere
zulässige Fassung aus Abschnitt 2. Die Summen- und Saldenliste
(`format: susa_csv`) ist der Rückfallweg, wenn ein Dienst die XBRL-Datei nicht
vollständig übernimmt.

## 4. Prüfpunkte

1. Der Abschluss ist aufgestellt; nach jeder späteren Buchung neu exportieren.
2. Jeder blockierende Befund ist geklärt: falsche Zuordnung über die Buchung,
   fehlende Stammdaten mit `stammdaten_vorschlagen`. Warnungen legst du vor.
3. Abweichungen der Steuerbilanz
   ([kst.md](kst.md#7-abweichungen-der-steuerbilanz)) sind entschieden.
4. Die Sendung enthält Kontennachweise und Anlagenspiegel
   ([BMF 10.06.2025][bmf69]). Fehlen die Kontennachweise, beanstandet die
   Verwaltung das bei Taxonomie 6.9 nicht, wenn sie auf anderem Weg eingehen
   und die dafür vorgesehene Erläuterungsposition die Gründe nennt (ebenda).

Fertig, wenn kein blockierender Befund bleibt, der Mensch die Warnungen kennt
und die Sendung Punkt 4 erfüllt.

## 5. Übermittlungsweg

Mein ELSTER hat kein Formular für die E-Bilanz
([Formularliste][elsterformulare], geprüft am 17.10.2026). Regelweg ist ein
Online-Dienst, der die XBRL-Datei importiert und über ELSTER sendet. Beide
Dienste erzeugen die Instanz neu und dokumentieren nicht, ob alles übernommen
wird; daher der Abgleich in Schritt 3.

| Dienst | Import | Kosten netto (Oktober 2026) | Hinweise |
|---|---|---|---|
| [eBilanz+][ebilanzplus] | XBRL ohne alle Berichtsteile, etwa Anlagenspiegel ([Hilfe][ebilanzplusxbrl]); Saldenliste ([Import][ebilanzplusimport]) | 35 € je Übermittlung, Eröffnungsbilanz und Korrektur frei ([Kosten][ebilanzpluskosten]) | Zertifikatsdatei oder Personalausweis |
| [eBilanz-Online][ebilanzonline] | XBRL ([Hilfe][ebilanzonlinexbrl]) | 39 € je Übertragung ([Preise][ebilanzonlinepreis]) | Auftragsverarbeitungsvertrag vorab; Zertifikatsdatei |

Ablauf mit dem Menschen:

1. `ebilanz_exportieren` (`jahr`); Abschnitt 4.
2. Im Dienst Unternehmen mit 13-stelliger Steuernummer und Vorgang für das
   Wirtschaftsjahr anlegen, XBRL-Datei importieren.
3. Abgleich: Bilanz und GuV mit `auswertung` (`art: bilanz`, `guv`),
   Kontennachweis mit `auswertung` (`art: susa`), Stammdaten mit `status`.
   Fehlendes (etwa Anlagenspiegel aus `anlagen`) ergänzt der Mensch im Dienst.
4. Der Mensch validiert, bis kein Fehler bleibt, und sendet mit dem
   ELSTER-Zertifikat.
5. Er speichert das Protokoll mit Transferticket in den Eingang und trägt beides
   in Buchfink ein; du legst das Protokoll mit `beleg_ablegen` ab.

Weitere Wege: Saldenliste (`format: susa_csv`, danach Abgleich wie Schritt 3);
Übermittlung durch einen Steuerberater; Antrag auf Verzicht bei unbilliger
Härte ([§ 5b Abs. 2 EStG][estg5b], [§ 150 Abs. 8 AO][ao150]), dann Abschriften
nach [§ 60 Abs. 1 EStDV][estdv60] und [§ 84 Abs. 3f EStDV][estdv84].

Wertungsfrage: Welcher Dienst? Nenne Kosten, Datenschutz (Auftragsverarbeitung,
Serverstandort) und Aufwand; der Mensch entscheidet. Ob ein Dienst die
Taxonomie annimmt, zeigt seine Validierung.

## 6. Eröffnungsbilanz

Bei Eröffnung des Betriebs auch für die Eröffnungsbilanz
([§ 5b Abs. 1 Satz 4 EStG][estg5b]), mit der Taxonomie, die zum Stichtag der
Aufstellung gilt ([BMF 10.06.2025][bmf69]). `fristen` setzt die Frist der
E-Bilanz des ersten Geschäftsjahres an
([ebilanz-eroeffnung](../grundlagen/pflichten.md#ebilanz-eroeffnung)).
`ebilanz_exportieren` mit `bilanzart: eroeffnungsbilanz` und dem
Gründungsjahr als `jahr` schreibt die Datei auf den Beurkundungstag, ohne GuV;
weiter wie Abschnitt 5. Zahlen:
[gruendung/eroeffnungsbilanz.md](../gruendung/eroeffnungsbilanz.md).

[ao150]: https://www.gesetze-im-internet.de/ao_1977/__150.html
[estdv60]: https://www.gesetze-im-internet.de/estdv_1955/__60.html
[estdv84]: https://www.gesetze-im-internet.de/estdv_1955/__84.html
[estg5b]: https://www.gesetze-im-internet.de/estg/__5b.html
[estg52]: https://www.gesetze-im-internet.de/estg/__52.html
[bmf68]: https://betriebs-berater.ruw.de/news/media/2/BMF-Schreiben-16701.pdf
[bmf69]: https://betriebs-berater.ruw.de/news/media/2/BMF-Schreiben-17367.pdf
[elsterformulare]: https://www.elster.de/eportal/formulare-leistungen/alleformulare
[ebilanzplus]: https://ebilanzplus.de
[ebilanzplusxbrl]: https://ebilanzplus.de/hilfe/xbrl
[ebilanzplusimport]: https://ebilanzplus.de/hilfe/import
[ebilanzpluskosten]: https://ebilanzplus.de/kosten
[ebilanzonline]: https://www.ebilanzonline.de
[ebilanzonlinexbrl]: https://www.ebilanzonline.de/wissen/faq/importmoeglichkeiten/xbrl-datei/
[ebilanzonlinepreis]: https://www.ebilanzonline.de/preismodell/
