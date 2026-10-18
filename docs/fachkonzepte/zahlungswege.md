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

Zu einem Umsatz schlägt Buchfink offene Posten vor, mit Punktzahl und
Begründung (`service/bank_suggest.go`). Gebucht wird erst, wenn jemand den
Vorschlag annimmt. Ein Konto aus dem Verwendungszweck rät Buchfink nicht.

**Aufteilung.** Ein Umsatz lässt sich auf mehrere Gegenkonten aufteilen
(`service/settlement_split.go`): offene Posten, ein Gesellschafterdarlehen oder
beliebige Konten. Typische Fälle sind eine Zinsgutschrift mit
Kapitalertragsteuer, eine Darlehensrate aus Tilgung und Zins und eine
Steuerlastschrift über mehrere Steuerarten. Die Summe muss dem Umsatz
entsprechen.

## Firmenkreditkarte

Ein mit der Karte bezahlter Beleg gilt am Buchungstag als bezahlt und steht bis
zur Abbuchung der Monatsabrechnung auf 3610 Kreditkartenabrechnung. Die
Abbuchung vom Bankkonto gleicht das Konto aus. Für die Umsatzsteuer ist der
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
