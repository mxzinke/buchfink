# Rechnungsabgrenzung

[Fachkonzepte](README.md) · [Jahresabschluss](jahresabschluss.md)

Anforderungen: BEW-08, GOB-06. Code: `service.AccrualService`,
`internal/accounting/accrual.go`.

Aufwand und Ertrag gehören in das Jahr, zu dem sie wirtschaftlich gehören
([§ 252 Abs. 1 Nr. 5 HGB][hgb252]). Die Versicherungsprämie, die im Dezember für
zwölf Monate gezahlt wird, ist zu elf Zwölfteln Aufwand des nächsten Jahres.

| Fall | Konto |
|---|---|
| Ausgabe vor dem Stichtag, Aufwand danach | **1900** Aktive Rechnungsabgrenzung |
| Einnahme vor dem Stichtag, Ertrag danach | **3900** Passive Rechnungsabgrenzung |

Beispiel: Versicherung 1.200 € für zwölf Monate ab 1. Dezember.

| Schritt | Buchung |
|---|---|
| Zahlung im Dezember | **6400** Versicherungen 1.200,00 an **1800** Bank 1.200,00 |
| Abgrenzung zum 31.12. | **1900** 1.100,00 an **6400** 1.100,00 |
| Auflösung im Folgejahr | **6400** 1.100,00 an **1900** 1.100,00 |

## Was abzugrenzen ist

Nur Aufwand oder Ertrag „für eine bestimmte Zeit“ nach dem Stichtag
([§ 250 HGB][hgb250]): Miete, Versicherung, Wartung, Lizenz, Abonnement.
Anzahlungen führt Buchfink auf eigenen Konten, siehe
[Anzahlungen](anzahlungen.md). Vorschuss und Kaution gehören zu den sonstigen
Vermögensgegenständen (**1300**) oder sonstigen Verbindlichkeiten (**3500**).
Eine Rechnung, deren Leistung vor dem Stichtag liegt und die erst danach bezahlt
wird, steht als Forderung oder Verbindlichkeit in der Bilanz.

## Wie Buchfink abgrenzt

1. Zum Bilanzstichtag sucht Buchfink die Buchungen des Jahres, deren
   Leistungszeitraum über den Stichtag hinausreicht. Der Leistungszeitraum
   steht an jeder Buchung.
2. Es rechnet den abzugrenzenden Anteil und schlägt die Buchung vor.
   Unterhalb einer einstellbaren Schwelle schlägt es nichts vor.
3. Die Abgrenzung wird mit Eigenbeleg gebucht und gehört zur Abschlussperiode.
4. Die Auflösung ist an die Abgrenzung gekoppelt und läuft ins Folgejahr,
   einmal zum Jahresbeginn oder monatlich.

Das Verfahren, monatsgenau oder taggenau, ist eine Einstellung des Mandanten,
weil [§ 252 Abs. 1 Nr. 6 HGB][hgb252] die Stetigkeit der Bewertungsmethoden
verlangt. Monatsgenau zählt der angefangene Anfangsmonat voll, ein
Zwölfmonatsvertrag bleibt zwölf Einheiten. Gerechnet wird in ganzen Cent.

Handelsrechtlich gibt es keine Bagatellgrenze. Steuerlich darf der Ansatz bei
Beträgen bis 800 € unterbleiben ([§ 5 Abs. 5 Satz 2 EStG][estg5]); das ist ein
Wahlrecht.

[hgb250]: https://www.gesetze-im-internet.de/hgb/__250.html
[hgb252]: https://www.gesetze-im-internet.de/hgb/__252.html
[estg5]: https://www.gesetze-im-internet.de/estg/__5.html
