# Abschlussbuchungen, Inventar und Abstimmung

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

## Inhalt

1. Inventar und Abstimmung
2. Abschlussbuchung mit `abschlussbuchung`
3. Umbuchung mit `buchung_erfassen`
4. Liste Abschlussposten

## 1. Inventar und Abstimmung

Zum Stichtag ist ein Inventar aufzustellen ([§ 240 Abs. 1, 2 HGB][hgb240]); der
Abschluss enthält sämtliche Vermögensgegenstände, Schulden und
Rechnungsabgrenzungsposten ([§ 246 Abs. 1 HGB][hgb246]). Buchfink führt weder
Vorräte noch Kasse; das Inventar besteht aus diesen Nachweisen:

| Bestand | Nachweis | Werkzeug |
|---|---|---|
| Bankkonten | Kontoauszug zum Stichtag | `auswertung` (`art: kontoblatt`, `konto`, `bis`) |
| Forderungen, Verbindlichkeiten aus Lieferungen und Leistungen | Liste der offenen Posten zum Stichtag | `offene_posten` (`stichtag`) |
| Darlehen | Saldenbestätigung oder Jahreskontoauszug der Bank | `auswertung` (`art: kontoblatt`) |
| Kreditkarte 3610 | Abrechnung zum Stichtag | `auswertung` (`art: kontoblatt`) |
| Umsatzsteuer, Lohnsteuer, Sozialversicherung | Anmeldungen, Beitragsnachweise | `auswertung`, [lohn/abstimmung.md](../lohn/abstimmung.md) |
| Gesellschafterkonten | Vertrag, Bestätigung des Gesellschafters | [gesellschafter/ausweis.md](../gesellschafter/ausweis.md) |
| Anlagen | Anlagenverzeichnis und Abgleich mit dem, was vorhanden ist | `anlagen` |
| Durchgangskonten 1460, 1370 | müssen null sein | `auswertung` (`art: kontoblatt`) |

Ordne jedem Bestandskonto mit Saldo aus `auswertung` (`art: susa`) einen
Nachweis zu und liste dem Menschen auf, was fehlt. Differenzen klärst du nach
der Referenz des Kontos, etwa [bank/zuordnung.md](../bank/zuordnung.md).

Fertig, wenn jeder Saldo belegt ist oder der Mensch die offene Lücke kennt.

## 2. Abschlussbuchung mit `abschlussbuchung`

Prüfe vor jeder Abschlussbuchung mit `buchungen_suchen`, ob es sie schon gibt.
Vorschau und Buchung haben denselben `idempotenz_schluessel`, etwa
`abschluss-2026-rst-abschlusskosten`. `abschlussbuchung` führt
Anlagenverzeichnis, Rückstellungsspiegel und Auflösungsplan der Abgrenzung
mit.

Die Fachregeln je `art` stehen in der Referenz des Ablaufschritts
([ablauf.md](ablauf.md#3-ablauf)), `anlage_aktivieren` in
[belege/anlagen-gwg.md](../belege/anlagen-gwg.md).

## 3. Umbuchung mit `buchung_erfassen`

Was kein Schritt von `abschlussbuchung` ist (etwa die Umbuchung auf ein
Restlaufzeitkonto), buchst du nach Rückfrage mit `buchung_erfassen`: Stichtag
als `buchungsdatum`, `abschlussperiode: true`, `steuerfall: not_taxable`. Die
`beschreibung` nennt was, warum und die Grundlage („Umbuchung Darlehen auf
Restlaufzeit bis 1 Jahr, § 268 Abs. 5 HGB“). Ohne Fremdbeleg nennt der
`eigenbeleg` im `grund` Rechenweg und Quelle der Zahlen.

- Abschlussbuchungen haben keine Umsatzsteuer. Ein Posten mit Umsatzsteuer
  ist ein [Eingangsbeleg](../belege/ablauf.md) oder eine
  [Ausgangsrechnung](../rechnungen/ablauf.md).
- Erlöskonten, die Buchfink einem Steuerfall zuordnet (z. B. 4336, 4338),
  zählen auch ohne Steuerschlüssel in die Voranmeldung; nimm sie hier nicht.

## 4. Liste Abschlussposten

Die Tabelle führst du im Chat und gibst sie dem Menschen am Ende. Sie hält
fest, was die Karteien nicht zeigen: Umbuchungen mit `buchung_erfassen` und
steuerliche Abweichungen.

| Posten | Konto | Betrag | Grundlage | Folgejahr | Steuerlich anders? |
|---|---|---|---|---|---|
| z. B. Aktiver RAP Versicherung | 1900 | 600,00 | § 250 Abs. 1 HGB | am 1.1. auf Aufwand auflösen | nein |
| z. B. Rückstellung Prozesskosten | 3070 | 12.000,00 | § 249 Abs. 1 HGB | Verbrauch oder Auflösung | Abzinsung prüfen |

„Folgejahr“ ist deine Aufgabenliste nach dem Saldenvortrag; „Steuerlich
anders?“ geht in die Erklärungen
([steuererklaerungen/kst.md](../steuererklaerungen/kst.md),
[steuererklaerungen/gewst.md](../steuererklaerungen/gewst.md)).

[hgb240]: https://www.gesetze-im-internet.de/hgb/__240.html
[hgb246]: https://www.gesetze-im-internet.de/hgb/__246.html
