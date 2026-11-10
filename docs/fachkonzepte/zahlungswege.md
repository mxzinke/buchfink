# Zahlungswege

[Fachkonzepte](README.md) · [Belege und Buchungen](beleg-buchungsflow.md) ·
[Buchungsbeispiele](buchungsbeispiele.md)

Eine GmbH bezahlt über ihre Bankkonten, die Firmenkreditkarte oder durch einen
Gesellschafter oder Geschäftsführer, der auslegt. Eine Kasse führt Buchfink
nicht: Buchungen auf ein Kassenkonto ohne Kassenbuch verstießen gegen
[§ 146 Abs. 1 AO][ao146]. Zahlungsmittelkonten sind die Bankkonten 1800 bis 1850
und Geldtransit 1460 (`domain.LiquidAccounts`).

## Bankkonto

Kontoauszüge kommen als CAMT.053. Der Auszug wird als Beleg abgelegt, jeder
Umsatz ist danach einzeln zu buchen und steht in genau einer gültigen Buchung.
Wird sie storniert, ist der Umsatz wieder offen. Buchfink nimmt nur
Euro-Konten und weist Sammeleinträge mit mehreren Einzeltransaktionen ab.
Nennt die Bank zu einem Umsatz keine Wertstellung, gilt der Buchungstag.

Zu einem Umsatz schlägt Buchfink offene Posten vor, mit Punktzahl und
Begründung (`service/bank_suggest.go`). Gebucht wird erst, wenn jemand den
Vorschlag annimmt. Ein Konto aus dem Verwendungszweck rät Buchfink nicht.

**Aufteilung.** Ein Umsatz lässt sich auf mehrere Gegenkonten aufteilen
(`service/settlement_split.go`): offene Posten, ein Gesellschafterdarlehen oder
beliebige Konten. Typische Fälle sind eine Zinsgutschrift mit
Kapitalertragsteuer, eine Darlehensrate aus Tilgung und Zins und eine
Steuerlastschrift über mehrere Steuerarten. Die Summe muss dem Umsatz
entsprechen. Der Kontoauszug belegt die Zahlung; den Grund belegt ein
abgelegter Beleg, den die Aufteilung nennt, etwa ein Kaufvertrag. Er ist danach
mit der Buchung versiegelt. Nach einem Storno belegt er die erneute Buchung
desselben Umsatzes.

## Firmenkreditkarte

Ein mit der Karte bezahlter Beleg gilt am Buchungstag als bezahlt und steht bis
zur Abbuchung der Monatsabrechnung auf 3610 Kreditkartenabrechnung. Die
Abbuchung vom Bankkonto gleicht das Konto aus. Ein Beleg in Fremdwährung steht zum
EZB-Kurs auf dem Konto; die Abweichung zum Kartenkurs bucht Buchfink beim
Ausgleich der Abrechnung als Kursdifferenz auf 6880 oder 4840. Nennt der Beleg
den belasteten Euro-Betrag, entsteht keine. Für die Umsatzsteuer ist der
Beleg mit dem Kartenumsatz bezahlt, was bei Zahlung vor der Leistung die
Mindest-Istversteuerung auslöst.

## Auslage

Bezahlt ein Gesellschafter oder Geschäftsführer eine Ausgabe privat, bleibt der
Lieferant Aussteller des Belegs; Gläubiger der Erstattung ist die Person, die
gezahlt hat. Die Erstattungsschuld steht auf 3640 (Gesellschafter) oder 3500
(Geschäftsführer, der kein Gesellschafter ist) mit der Person als Kontakt. Wer
den Beleg ausgestellt hat, kann nicht zugleich die Erstattung bekommen. Die
Auslage eines Arbeitnehmers gehört in die Lohnabrechnung.

## Gesellschafterdarlehen

Ein Darlehensvertrag zwischen GmbH und Gesellschafter wird in beide Richtungen
geführt (`service.ShareholderService`). Der Vertrag steuert Zinsbuchungen und
den Fremdvergleich. Erfassen und berichtigen kann ihn der Mensch in der App
oder ein Agent nach Zustimmung im Chat (`darlehen_erfassen`); eine
Berichtigung braucht einen Grund und steht im Änderungsprotokoll.

Den Fremdvergleich des Zinses belegt ein Vergleichszins mit Nachweis oder ein
abgelegter Vermerk zur Zinsentscheidung. Fehlt beides, weist Buchfink darauf
hin und nennt die Abweichung, die schadet: Zahlt die Gesellschaft den Zins, ist
es der überhöhte, zahlt ihn der Gesellschafter, der zu niedrige. Liegt ein
Vergleichszins vor, meldet Buchfink nur die schädliche Abweichung.

| Richtung | Konto nach Restlaufzeit | Zinsen |
|---|---|---|
| Gesellschafter an GmbH | 3641, 3642, 3643 | Aufwand auf 7316, bei mehr als 25 % Beteiligung 7317 |
| GmbH an Gesellschafter | 1308, 1309 | Ertrag auf 7110 |

Der Bestand steht im Journal: Auszahlung und Tilgung buchen über die
Aufteilung eines Bankumsatzes. Unübliche Konditionen, etwa ein zinsloses
Darlehen an den Gesellschafter, sperren nichts; sie erscheinen als Befund im
Prüflauf, weil sie eine verdeckte Gewinnausschüttung sein können. Im
Jahresabschluss werden Ausleihungen, Forderungen und Verbindlichkeiten gegenüber
Gesellschaftern gesondert ausgewiesen ([§ 42 Abs. 3 GmbHG][gmbhg42]).

## Lohnjournal

Löhne rechnet ein externes Lohnbüro ab. Buchfink bucht die Monatssummen aus
dessen Lohnjournal, das als Beleg abgelegt wird (`service.PayrollService`):
Gehälter, Geschäftsführergehälter, Minijobs und Arbeitgeberanteile im Soll,
Nettolöhne, Lohnsteuer und Sozialversicherung im Haben. Vor dem Buchen prüft
es, dass Brutto abzüglich der Abzüge den Nettolohn ergibt. Ein Monat wird
einmal gebucht; ein berichtigtes Journal ersetzt die Buchung durch Storno und
neue Buchung. Die Abstimmung zum Stichtag zeigt, ob auf den Lohnkonten mehr
offen steht als die Beträge des laufenden Monats.

Lohnkonten nach § 41 EStG, Lohnsteuer-Anmeldung und Beitragsnachweise führt
das Lohnbüro. Die Pflichten bleiben beim Arbeitgeber.

[ao146]: https://www.gesetze-im-internet.de/ao_1977/__146.html
[gmbhg42]: https://www.gesetze-im-internet.de/gmbhg/__42.html

## Uneinbringliche Forderungen

Die Ausbuchung ist ein eigener Vorgang ohne Bankumsatz. Der Bruttobetrag
mindert den offenen Posten; bei Teilbeträgen bleibt die Restforderung stehen.
Der Nettobetrag geht auf das Forderungsverlustkonto des Steuersatzes. Eine
zugeordnete Einzelwertberichtigung wird bis zum Nettoausfall aufgelöst,
bei Vollausbuchung vollständig, über 1246/1247 an 4923. Ihre Zuordnung steht
an der Buchungszeile und bleibt bei einer Generalumkehr erhalten.

Nachweis und Grund gehören zur Buchung. Die Steuerberichtigung erfolgt zum
Eintritt der Uneinbringlichkeit, nicht zum Rechnungsdatum
([§ 17 Abs. 2 Nr. 1 i. V. m. Abs. 1 UStG](https://www.gesetze-im-internet.de/ustg_1980/__17.html),
Wortlaut gelesen am 10.11.2026). Ohne eigene Umsatzsteuer gibt es keine
Steuerkorrektur. Eine festgeschriebene Periode wird nicht umgangen.

Spätere Vereinnahmung verlangt nach § 17 Abs. 2 Nr. 1 Satz 2 UStG eine erneute
Berichtigung. Dafür fehlt ein geführter Ablauf; der ursprüngliche Ausfall darf
nicht allein wegen einer späteren Zahlung storniert werden. Nicht zugeordnete
EWB-Altbestände werden nicht automatisch auf einzelne Forderungen verteilt.

## Abschlagsrechnungen

Offene Abschläge sind Merkposten ohne Forderungsbuchung. Bilanz, Bewertung
und Buchführungsprüfungen lesen weiterhin nur die gebuchten Posten. App und Werkzeuge
reichen ihre Rechnungskennung an den Zahlungsdienst weiter. Die Vorbereitung
von `SettleAdvance` und der Bankaufteilung verwendet dieselben Prüfungen und
Buchungszeilen. Eine Aufteilung erzeugt eine gemeinsame Buchung für den ganzen
Bankumsatz. Jede Vereinnahmung und Rückzahlung wird mit Betrag, Steuer und
Buchung am Abschlag gespeichert; ein Stichtag berücksichtigt nur die bis dahin
wirksamen Zahlungen und Stornos.

Die Steuer entsteht auf den vereinnahmten Teilbetrag im Zahlungsmonat
([§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG](https://www.gesetze-im-internet.de/ustg_1980/__13.html)).
Buchfink verteilt die Steuer kumuliert auf Teilzahlungen, damit die vollständige
Vereinnahmung centgenau den Rechnungsbetrag ergibt. Die Schlussrechnung setzt
auch einen teilweise vereinnahmten Abschlag nur mit seinem vereinnahmten Betrag
und der zugehörigen Steuer ab
([§ 14 Abs. 5 Satz 2 UStG](https://www.gesetze-im-internet.de/ustg_1980/__14.html)).
Mit der Schlussrechnung entfällt der offene Rest des Abschlags als eigener Posten.

Eine Rückzahlung setzt die Bestätigung voraus, dass die Anzahlung wegen einer
nicht ausgeführten Leistung zurückgewährt wird. Die Steuerberichtigung erfolgt
im Rückzahlungszeitraum
([§ 17 Abs. 2 Nr. 2 in Verbindung mit Abs. 1 Satz 8 UStG](https://www.gesetze-im-internet.de/ustg_1980/__17.html),
[BFH vom 02.09.2010, V R 34/09, Leitsatz 1 und Rz. 19–21](https://www.bundesfinanzhof.de/de/entscheidung/entscheidungen-online/detail/pdf/STRE201010349?type=1646225765)).
Bankabgang und Begründung werden zusammen gebucht. Die Rechnung wird dadurch
nicht storniert: Ihr Betrag steht wieder offen, bis sie bezahlt oder storniert
wird. Eine Schlussrechnung setzt zurückgezahlte Beträge nicht ab. Nach der
Schlussrechnung ist zunächst deren Berichtigung erforderlich.
