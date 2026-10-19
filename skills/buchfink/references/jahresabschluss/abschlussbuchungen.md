# Abschlussbuchungen, Inventar und Abstimmung

Rechtsstand: 17. Oktober 2026. Wortlaut des HGB am 17.10.2026 gelesen.
Werkzeugstand: Buchfink 0.3.0.

## Inhalt

1. Inventar und Abstimmung
2. Abschlussbuchung mit `abschlussbuchung`
3. Umbuchung mit `buchung_erfassen`
4. Liste Abschlussposten
5. Reihenfolge

## 1. Inventar und Abstimmung

Zum Schluss jedes Geschäftsjahres ist ein Inventar aufzustellen: Forderungen,
Schulden, Geld und sonstige Vermögensgegenstände mit ihrem Wert
([§ 240 Abs. 1, 2 HGB][hgb240]), innerhalb der einem ordnungsmäßigen
Geschäftsgang entsprechenden Zeit. Der Abschluss muss sämtliche
Vermögensgegenstände, Schulden und Rechnungsabgrenzungsposten enthalten
([§ 246 Abs. 1 HGB][hgb246]). Buchfink führt kein Vorratsvermögen und keine
Kasse; bei der Zielgruppe besteht das Inventar aus den folgenden Nachweisen.

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

Ablauf:

1. `auswertung` (`art: susa`, `geschaeftsjahr`): Salden aller bebuchten Konten.
2. Ordne jedem Bestandskonto mit Saldo einen Nachweis aus der Tabelle zu.
3. Liste dem Menschen auf, was fehlt (z. B. Saldenbestätigung der Bank).
4. Differenzen klärst du nach der Referenz des Kontos, etwa
   [bank/zuordnung.md](../bank/zuordnung.md) für Bankkonten.

Forderungen und Verbindlichkeiten stehen unverrechnet nebeneinander
([forderungen.md](forderungen.md#5-ausweis-und-restlaufzeiten)).

Fertig, wenn jeder Saldo belegt ist oder der Mensch die offene Lücke kennt.

## 2. Abschlussbuchung mit `abschlussbuchung`

Jede Abschlussbuchung ist eine Rückfrage. `abschluss_status` zeigt, welcher
Schritt offen ist. Prüfe vor jeder Abschlussbuchung mit `buchungen_suchen`
(`konto`, `text`, Zeitraum), ob es sie schon gibt. Rechne jeden Schritt zuerst
mit `nur_vorschau: true`, lege Buchungssatz und `erlaeuterung` der Vorschau im
[Rückfrage-Format](../../SKILL.md#3-den-menschen-fragen) vor und buche nach dem
Ja mit demselben `idempotenz_schluessel` (`abschluss-<jahr>-<posten>`, z. B.
`abschluss-2026-rst-abschlusskosten`). Ändern heißt stornieren und neu buchen
([Buchung korrigieren](../grundlagen/arbeitsweise.md#11-eine-buchung-korrigieren)).

| `art` | Angaben im gleichnamigen Feld | Mehr in |
|---|---|---|
| `afa_lauf` | `buchungsdatum` (Vorgabe Bilanzstichtag), `anlage_ids` | [abschreibungen.md](abschreibungen.md) |
| `anlage_aktivieren` | `anlage`: `buchung_id` aus `anlagen`, `bezeichnung`, `klasse`, `methode`, `nutzungsdauer_monate`, `nutzungsdauer_grund`, `in_betrieb` | [belege/anlagen-gwg.md](../belege/anlagen-gwg.md) |
| `abgrenzung` | `art` (`aktiv`, `passiv`, `disagio`), `text`, `gesamtbetrag`, `von`, `bis`, `konto`, `quell_buchung_id`, `abzugrenzen` | [abgrenzung.md](abgrenzung.md) |
| `rueckstellung` | `art`, `text`, `betrag` (Erfüllungsbetrag), `erwartet_am`, `grund`, `aufwandskonto` | [rueckstellungen.md](rueckstellungen.md) |
| `wertberichtigung` | `einzeln` (`buchung_id`, `prozent`, `grund`), `pauschal_promille`, `grund` | [forderungen.md](forderungen.md) |
| `unfertige_leistungen` | `projekte` (`name`, `kontakt_id`, `einzelkosten`, `gemeinkosten`, `auftragswert`, `restkosten`, `anzahlungen`, `notiz`), `anzahlungen_absetzen`, `grund` | [unfertige-leistungen.md](unfertige-leistungen.md) |
| `fremdwaehrung` | keine | [forderungen.md](forderungen.md) |
| `ust_verrechnung` | keine | [steuern-im-abschluss.md](steuern-im-abschluss.md) |
| `steuerrueckstellung` | `hebesatz`, `verlustvortrag_kst`, `verlustvortrag_gewst`, `korrekturen`, `hinzurechnungen`, `kuerzungen`, `kst_betrag`, `gewst_betrag`, `grund` | [steuern-im-abschluss.md](steuern-im-abschluss.md) |
| `ug_ruecklage` | keine | [ergebnisverwendung-ug.md](ergebnisverwendung-ug.md) |

Die Feldnamen nennt das Schema des Werkzeugs; es geht vor. `abschlussbuchung`
führt Buchfinks Karteien mit: Anlagenverzeichnis, Rückstellungsspiegel,
Auflösungsplan der Abgrenzung.

## 3. Umbuchung mit `buchung_erfassen`

Was kein Schritt von `abschlussbuchung` ist (etwa die Umbuchung auf ein
Restlaufzeitkonto), buchst du nach Rückfrage mit `buchung_erfassen`:

| Feld | Inhalt |
|---|---|
| `buchungsdatum` | Abschlussstichtag |
| `abschlussperiode` | `true` |
| `beschreibung` | Was, warum, Grundlage, z. B. „Umbuchung Darlehen auf Restlaufzeit bis 1 Jahr, § 268 Abs. 5 HGB“ |
| `zeilen` | Konten, Seite, Betrag; keine Steuerschlüssel bei Abschlussbuchungen |
| `steuerfall` | `not_taxable`; Buchfink verlangt an jeder Handbuchung einen Steuerfall |
| `eigenbeleg` | `grund` mit Rechenweg und Quelle der Zahlen (Angebot, Vertrag, Vorjahr, Liste), `brutto` |
| `beleg_id` | statt Eigenbeleg, wenn ein Fremdbeleg trägt (Kostenvoranschlag, Bescheid) |
| `idempotenz_schluessel` | Pflicht, wie oben |

Regeln:

- Abschlussbuchungen tragen keine Umsatzsteuer. Bringt ein Posten
  Umsatzsteuer mit (vergessene Rechnung), ist das ein Eingangsbeleg
  ([belege/ablauf.md](../belege/ablauf.md)) oder eine Ausgangsrechnung
  ([rechnungen/ablauf.md](../rechnungen/ablauf.md)) und betrifft die
  Voranmeldung.
- Die Umsatzsteuer- und Vorsteuerkonten (1400, 1401, 1402, 1404, 1406, 1407,
  1408, 3800, 3801, 3802, 3804, 3806, 3835, 3837, 3851) und die Konten der
  Vorsteuerberichtigung bebucht nur die Steuerautomatik; `buchung_erfassen`
  lehnt Zeilen darauf ohne Steuerschlüssel ab. Die Verrechnung bucht
  `abschlussbuchung` mit `art: ust_verrechnung`.
- Nimm keine Erlöskonten, die Buchfink einem Steuerfall zuordnet (z. B. 4336,
  4338). Sie zählen auch ohne Steuerschlüssel in die Voranmeldung.
- Nutze nur Konten, die `konten` (`art: suche`) findet. Die Kontonummern in den
  references stammen aus dem SKR04-Katalog von Buchfink.

## 4. Liste Abschlussposten

Führe im Chat eine Tabelle und gib sie dem Menschen am Ende. Sie hält fest,
was die Karteien nicht zeigen: Umbuchungen mit `buchung_erfassen` und
steuerliche Abweichungen.

| Posten | Konto | Betrag | Grundlage | Folgejahr | Steuerlich anders? |
|---|---|---|---|---|---|
| z. B. Aktiver RAP Versicherung | 1900 | 600,00 | § 250 Abs. 1 HGB | am 1.1. auf Aufwand auflösen | nein |
| z. B. Rückstellung Prozesskosten | 3070 | 12.000,00 | § 249 Abs. 1 HGB | Verbrauch oder Auflösung | Abzinsung prüfen |

Die Spalte „Folgejahr“ ist deine Aufgabenliste nach dem Saldenvortrag. Die
Spalte „Steuerlich anders?“ geht in die Erklärungen
([steuererklaerungen/kst.md](../steuererklaerungen/kst.md),
[steuererklaerungen/gewst.md](../steuererklaerungen/gewst.md)).

## 5. Reihenfolge

Die Reihenfolge folgt den Abhängigkeiten: Erst stehen die Bestände
(Abstimmung, Anlagen), dann die Periodenabgrenzung (Abgrenzung,
Rückstellungen), dann die Bewertung (Forderungen, unfertige Leistungen,
Fremdwährung), dann die Umsatzsteuer-Verrechnung. Danach steht das Ergebnis vor
Steuern, aus dem die Steuerrückstellung folgt. Die UG-Rücklage setzt den
Jahresüberschuss nach Steuern voraus.

[hgb240]: https://www.gesetze-im-internet.de/hgb/__240.html
[hgb246]: https://www.gesetze-im-internet.de/hgb/__246.html
