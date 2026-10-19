# E-Bilanz

Rechtsstand: 17. Oktober 2026. Wortlaut von § 5b EStG, § 52 Abs. 11 EStG,
§ 60 EStDV und § 150 AO am 17.10.2026 gelesen. BMF-Schreiben zu den
Taxonomien 6.8 und 6.9 über einen Spiegel gelesen; das Schreiben zur
Taxonomie 6.10 nur über Sekundärquellen. Seiten von eBilanz+ und
eBilanz-Online am 17.10.2026 gelesen.

Frist, Nachweis und Folgen einer Versäumnis stehen in
[grundlagen/pflichten.md#ebilanz](../grundlagen/pflichten.md#ebilanz).

## Inhalt

1. Pflicht und Inhalt
2. Taxonomie und Mindestumfang
3. Was Buchfink liefert
4. Prüfpunkte
5. Übermittlungsweg
6. Eröffnungsbilanz

## 1. Pflicht und Inhalt

Wer den Gewinn durch Bilanz ermittelt, übermittelt den Inhalt der Bilanz und
der GuV nach amtlichem Datensatz elektronisch
([§ 5b Abs. 1 Satz 1 EStG][estg5b]). Was dazugehört, hängt vom Beginn des
Wirtschaftsjahres ab ([§ 52 Abs. 11 EStG][estg52]):

| Bestandteil | Pflicht für Wirtschaftsjahre, die beginnen |
|---|---|
| Bilanz und GuV | nach dem 31.12.2010 |
| unverdichtete Kontennachweise mit Kontensalden | nach dem 31.12.2024 |
| Anlagenspiegel und Anlagenverzeichnis, Anhang, Lagebericht, Prüfungsbericht | nach dem 31.12.2027 |

Stand: Oktober 2026. Den Anlagenspiegel verlangt die Taxonomie nach dem
BMF-Schreiben vom 10.06.2025 schon seit Wirtschaftsjahren ab 2017 als Mussfeld
([BMF 10.06.2025][bmf69]).

- Ansätze, die steuerlichen Vorschriften nicht entsprechen, werden durch
  Zusätze oder Anmerkungen angepasst (Überleitungsrechnung); alternativ wird
  eine Steuerbilanz übermittelt ([§ 5b Abs. 1 Satz 2, 3 EStG][estg5b]).
- Ein unverdichteter Kontennachweis umfasst mindestens alle Sachkonten mit
  Saldo am Ende des Wirtschaftsjahres, je werthaltiger Position mit
  Positionsname, Kontonummer, Kontobezeichnung und Saldo. Personenkonten und
  Journal werden nicht übermittelt ([BMF 10.06.2025][bmf69]).
- Für die Taxonomie 6.9 beanstandet die Verwaltung nicht, wenn die
  Kontennachweise wegen Softwareanpassungen auf anderem Weg eingereicht und die
  Gründe in der dafür vorgesehenen Position erläutert werden
  ([BMF 10.06.2025][bmf69]).

## 2. Taxonomie und Mindestumfang

| Wirtschaftsjahr beginnt | Taxonomie | Fundstelle |
|---|---|---|
| 2025 | 6.8; 6.9 wird nicht beanstandet | [BMF 27.05.2024][bmf68], [BMF 10.06.2025][bmf69] |
| 2026 | 6.9; 6.10 wird nicht beanstandet | [BMF 10.06.2025][bmf69]; Schreiben vom 08.06.2026 laut Sekundärquelle |
| 2027 | 6.10 | BMF vom 08.06.2026, IV C 6 - S 2133-b/00067/002/023, laut Sekundärquelle |

Stand: Oktober 2026. Die Taxonomie wechselt jährlich; die gültige Fassung
veröffentlicht die Verwaltung auf esteuer.de.

Mindestumfang: Als „Mussfeld“ gekennzeichnete Positionen sind zu befüllen. Ab
Taxonomie 6.9 gilt das nur für die Mussfelder der jeweiligen Rechtsform. Lässt
sich ein Mussfeld aus der Buchführung nicht füllen, wird es ohne Wert (NIL)
übermittelt ([BMF 27.05.2024][bmf68]).

## 3. Was Buchfink liefert

- `ebilanz_exportieren` (`jahr`, `format: xbrl` als Vorgabe) prüft die
  E-Bilanz und schreibt eine XBRL-Instanz nach der amtlichen Taxonomie des
  Jahres (Abschnitt 2) mit Kontennachweis in den Exportordner. `taxonomie`
  wählt eine andere zulässige Fassung (etwa `6.10`),
  `kleinstkapitalgesellschaft` die GuV nach § 275 Abs. 5 HGB. Diese Datei
  importiert der Mensch in einen Online-Dienst (Abschnitt 5).
- `ebilanz_exportieren` mit `format: susa_csv` schreibt die Summen- und
  Saldenliste mit den Elementnamen der Taxonomie als CSV. Das ist der
  Rückfallweg, wenn ein Dienst die XBRL-Datei nicht vollständig übernimmt.
- Vor dem Schreiben prüft `ebilanz_exportieren` die Zuordnung jedes Kontos,
  die Rechenregeln der Taxonomie, Bilanzsumme und Jahresüberschuss, die
  Mussfelder der Rechtsform und die Stammdaten. Blockierende `befunde`
  verhindern den Export.

## 4. Prüfpunkte

1. Der Abschluss ist aufgestellt. Ändert sich danach eine Buchung, exportiere
   neu.
2. Jeder blockierende Befund aus `ebilanz_exportieren` ist geklärt: eine
   falsche Zuordnung über die Buchung, fehlende Stammdaten (Steuernummer,
   Anschrift) über den Menschen in der App. Warnungen legst du ihm vor.
3. Abweichungen der Steuerbilanz aus der Körperschaftsteuer
   ([kst.md](kst.md#7-abweichungen-der-steuerbilanz)) sind als Wertungsfrage
   entschieden.
4. Die Sendung im Online-Dienst (Abschnitt 5) enthält Kontennachweise und
   Anlagenspiegel. Der Anlagenspiegel ist Mussfeld und zu übermitteln
   ([BMF vom 10.06.2025][bmf69]). Fehlen die Kontennachweise in der Sendung,
   gilt für Taxonomie 6.9 eine Nichtbeanstandung: Kontennachweise auf anderem
   Weg beim Finanzamt einreichen und die Gründe in der Position „Erläuterung,
   warum eine Übermittlung der Kontennachweise noch nicht möglich ist“
   darlegen ([BMF vom 10.06.2025][bmf69]).

Fertig, wenn kein blockierender Befund bleibt, der Mensch die Warnungen kennt
und die Sendung Punkt 4 erfüllt.

## 5. Übermittlungsweg

Die Formularliste von Mein ELSTER führt kein Formular für Bilanz oder
E-Bilanz ([Formularliste][elsterformulare], geprüft am 17.10.2026). Buchfink
übermittelt nicht selbst. Der Regelweg ist ein Online-Dienst, der die
XBRL-Datei importiert und mit ELSTER-Anbindung sendet. Beide Dienste lesen
die Werte in ihren eigenen Editor ein und erzeugen die Instanz beim Senden
neu; ob Kontennachweis, Stammdaten und Überleitung dabei vollständig
übernommen werden, dokumentiert keiner von beiden.

| Dienst | Import | Kosten (netto, Stand Oktober 2026) | Hinweise |
|---|---|---|---|
| [eBilanz+][ebilanzplus] (ebilanzplus GmbH) | „XBRL-Datei“ in fünf Schritten ([Hilfe][ebilanzplusxbrl]); nicht alle Berichtsteile, etwa der Anlagenspiegel, kommen über den Import. Alternativ Summen- und Saldenliste mit Taxonomie-Layout ([Import][ebilanzplusimport]) | 35 € je Übermittlung, Eröffnungsbilanz und Korrektur kostenlos ([Kosten][ebilanzpluskosten]) | Validierung kostenlos und beliebig oft; Senden mit ELSTER-Zertifikat als Datei oder Personalausweis; Übertragungsprotokoll mit Transferticket |
| [eBilanz-Online][ebilanzonline] (Bundesanzeiger Verlag) | XBRL-Import zum Weiterbearbeiten ([Hilfe][ebilanzonlinexbrl]); Einzelheiten im Benutzerhandbuch nach der Anmeldung | 39 € je erfolgreicher Übertragung, Staffelpreise ([Preise][ebilanzonlinepreis]) | Vereinbarung zur Auftragsverarbeitung vor der Nutzung; Senden mit Zertifikat als Datei; Übertragungsprotokoll |

Ablauf mit dem Menschen:

1. `ebilanz_exportieren` (`jahr`); Abschnitt 4.
2. Im Dienst: Unternehmen mit 13-stelliger Steuernummer anlegen, Vorgang für
   das Wirtschaftsjahr anlegen, die XBRL-Datei importieren.
3. Nach dem Import gleicht ihr die Werte mit dem Buchfink-Bericht ab: Bilanz
   und GuV mit `auswertung` (`art: bilanz`, `guv`), den Kontennachweis mit
   den Salden aus `auswertung` (`art: susa`), die Stammdaten mit `status`.
   Was fehlt (etwa der Anlagenspiegel aus `anlagen`), ergänzt der Mensch im
   Dienst.
4. Der Mensch validiert im Dienst, bis kein Fehler bleibt, und sendet mit dem
   ELSTER-Zertifikat.
5. Er lädt das Übertragungsprotokoll mit Transferticket herunter, speichert es
   in den Eingang und trägt Protokoll und Transferticket in Buchfink ein. Du
   legst das Protokoll auf seine Anweisung mit `beleg_ablegen` ab.

Weitere Wege:

1. **Rückfallweg Summen- und Saldenliste.** Übernimmt der Dienst die
   XBRL-Datei nicht vollständig, schreibt `ebilanz_exportieren` mit
   `format: susa_csv` die Salden mit den Elementnamen der Taxonomie; der Dienst
   liest sie als Saldenliste ein. Prüfe danach wie in Schritt 3.
2. Ein Steuerberater übermittelt.
3. Antrag auf Verzicht wegen unbilliger Härte
   ([§ 5b Abs. 2 EStG][estg5b]). Das Finanzamt muss zustimmen, wenn die
   elektronische Übermittlung wirtschaftlich oder persönlich unzumutbar ist,
   insbesondere wenn sie nur mit nicht unerheblichem finanziellem Aufwand
   möglich wäre oder der Steuerpflichtige sie nach seinen Kenntnissen und
   Fähigkeiten nicht oder nur eingeschränkt nutzen kann
   ([§ 150 Abs. 8 AO][ao150]). Dann gehen Bilanz und GuV mit
   Kontennachweisen und der Anlagenspiegel als Abschrift mit der Erklärung,
   die Kontennachweise unverdichtet, bei Eröffnung des Betriebs auch die
   Eröffnungsbilanz ([§ 60 Abs. 1 EStDV][estdv60]); das Anlagenverzeichnis erst für
   Wirtschaftsjahre, die nach dem 31.12.2027 beginnen
   ([§ 84 Abs. 3f EStDV][estdv84]).

Wertungsfrage: Welcher Dienst passt? Nenne Kosten, Datenschutz
(Auftragsverarbeitung, Serverstandort laut den Bedingungen des Dienstes) und
Aufwand; der Mensch entscheidet. Ob ein Dienst die Taxonomie des
Wirtschaftsjahres annimmt, zeigt seine Validierung.

## 6. Eröffnungsbilanz

Bei Eröffnung des Betriebs gilt die Pflicht auch für die Eröffnungsbilanz
([§ 5b Abs. 1 Satz 4 EStG][estg5b]). Die Taxonomie eines Jahres gilt für
Eröffnungsbilanzen, die nach dem jeweiligen Stichtag aufzustellen sind
([BMF 10.06.2025][bmf69]). Einen eigenen Abgabetermin nennt § 5b nicht;
`fristen` setzt die Frist der E-Bilanz des ersten Geschäftsjahres an
([ebilanz-eroeffnung](../grundlagen/pflichten.md#ebilanz-eroeffnung)).
`ebilanz_exportieren` mit `bilanzart: eroeffnungsbilanz` und dem Geschäftsjahr
der Gründung als `jahr` schreibt die XBRL-Datei auf den Beurkundungstag, ohne
GuV; Import und Senden wie in Abschnitt 5. Die Zahlen prüft
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
