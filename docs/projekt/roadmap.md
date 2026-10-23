# Roadmap

[Dokumentation](../README.md) · [Umsetzungsstand](umsetzungsstand.md)

Ziel: Eine GmbH oder UG führt ihre Bücher mit einem KI-Agenten vom Beleg bis zu
Jahresabschluss und Erklärungen, mit oder ohne Steuerberater. Buchfink rechnet
nach festen Regeln, der Mensch gibt frei, was sich nicht rückgängig machen
lässt. Was in Version 0.3 gebaut ist, steht im
[Umsetzungsstand](umsetzungsstand.md). Termine gibt es nicht.

## Bis Version 1

- **Pilotbetrieb.** Ein Monat mit Festschreibung und ein Jahresabschluss mit
  Erklärungen bei einer echten GmbH. Was dabei auffällt, geht in Regeln, Skills
  und Werkzeuge.
- **Clients Ende zu Ende prüfen.** Anmeldung, Zustimmung und Arbeit mit Claude
  Code, Claude Desktop, OpenCode, Codex und Pi unter macOS, Windows und Linux,
  einschließlich der Plugin-Installation unter Windows.
- **Signierte Releases.** Notarisierte Builds für macOS und signierte für
  Windows, damit die Warnungen beim ersten Start entfallen.
- **Vordrucke je Jahr.** Zeilenkataloge der Jahreserklärungen für jedes neue
  Vordruckjahr, die Zeilen von KSt und GewSt am amtlichen Vordruck prüfen.
- **E-Bilanz erproben.** Eine von Buchfink erzeugte XBRL-Datei über
  ebilanzplus.de oder ebilanzonline.de übermitteln, die Saldenliste in
  eBilanz+ einlesen und die Annahme belegen.
- **Werkzeuge schärfen.** Welche Vorgänge der Agent direkt buchen darf und
  welche er erfragt, nach den Erfahrungen aus dem Pilotbetrieb nachstellen.

## Danach, ohne Zusage

- Istversteuerung nach § 20 UStG
- Datensatz für die Offenlegung im Unternehmensregister
- Zertifikatskette der Zeitstempel gegen eine Vertrauensliste prüfen
- weitere Steuerfälle, etwa OSS

## Nicht geplant

- eigene Übermittlung an die Finanzverwaltung
- Verschlüsselung in Buchfink; den Schutz am Rechner übernimmt das
  Betriebssystem
- Kassenbuch, Warenwirtschaft, Lohnabrechnung
- andere Rechtsformen und die Einnahmen-Überschuss-Rechnung

## Prüfpunkte mit Datum

Diese Termine stehen von außen fest und sind zu ihrem Zeitpunkt nachzuziehen.

| Wann | Was | Wo im Code |
|---|---|---|
| jährlich | neuer Vordruck USt 1 A und neue Vordrucke der Jahreserklärungen | `accounting/ustva.go`, `internal/declaration/catalog/` |
| jährlich | neue E-Bilanz-Taxonomie nach BMF-Schreiben | `internal/ebilanz/taxonomy/`, `scripts/gen_ebilanz_taxonomy.py` |
| jährlich | Jahresstand des SKR04 | `internal/accounting/skr04_2026.json` |
| 1. Januar und 1. Juli | Basiszinssatz nach § 247 BGB | `accounting/tax_params.go` |
| 1. Januar 2027 und 2028 | Sendepflicht für E-Rechnungen ([§ 27 Abs. 38 UStG][ustg27]) | Rechnungsausgabe |
| Wirtschaftsjahre ab 2028 | Anlagenspiegel und Anlagenverzeichnis in der E-Bilanz (§ 5b Abs. 1 EStG) | `internal/ebilanz` |
| offen | Buchführungsdatenschnittstelle nach [§ 147b AO][ao147b], Verordnung noch nicht erlassen | `internal/export` |

[ustg27]: https://www.gesetze-im-internet.de/ustg_1980/__27.html
[ao147b]: https://www.gesetze-im-internet.de/ao_1977/__147b.html
