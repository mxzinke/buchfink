# Prüflauf und Festschreibung

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Die Festschreibung sichert die Unveränderbarkeit
([grundlagen/gobd.md](../grundlagen/gobd.md#7-unveränderbarkeit-storno-festschreibung));
Fälligkeit:
[grundlagen/pflichten.md#monatsfestschreibung](../grundlagen/pflichten.md#monatsfestschreibung).

## Inhalt

1. Befunde des Prüflaufs
2. Befunde übergehen
3. Was nach der Festschreibung gilt

## 1. Befunde des Prüflaufs

`auswertung` (`art: pruefbericht`) liefert je Befund `regel`, `schwere`,
`text` und `norm`. Ein blockierender Befund lässt die Festschreibung nur mit
Begründung des Menschen zu.

| Regel | Vorgehen |
|---|---|
| `entry_without_receipt` | Beleg ablegen und verknüpfen oder Eigenbeleg ([grundlagen/gobd.md](../grundlagen/gobd.md#8-belegwesen-und-eigenbeleg)) |
| `receipt_unbooked`, `receipt_overdue` | buchen ([belege/ablauf.md](../belege/ablauf.md)) oder verwerfen |
| `bank_unmatched` | zuordnen oder mit Grund ignorieren ([bank/zuordnung.md](../bank/zuordnung.md)) |
| `interim_balance` | Gegenbuchung finden ([bank/kreditkarte-geldtransit.md](../bank/kreditkarte-geldtransit.md)) |
| `account_unmapped` | Konto prüfen; Fehlbuchung stornieren |
| `depreciation_missing` | [jahresabschluss/abschreibungen.md](../jahresabschluss/abschreibungen.md) |
| `duplicate_receipt`, `duplicate_payment` | prüfen, gegebenenfalls verwerfen oder stornieren |
| `number_gap` | Ursache nennen |
| `vat_return_missing` | [ablauf.md](ablauf.md#2-ablauf), Schritt 5 |
| `commit_overdue`, `period_not_committed` | Festschreibung anstoßen |
| `vat_id_unconfirmed` | [rechnungen/ausland.md](../rechnungen/ausland.md) |
| `service_proof_missing` | Vermerk beim Menschen anfordern |
| `provision_discount`, `closing_step_skipped`, `size_class_change` | [jahresabschluss/ablauf.md](../jahresabschluss/ablauf.md) |

Andere Regeln: nach `text` und `norm`. Fertig mit einem Befund, wenn ein neuer
Prüflauf ihn nicht mehr zeigt oder der Mensch ihn übergehen will.

## 2. Befunde übergehen

- Nur der Mensch übergeht einen blockierenden Befund, in der App bei der
  Festschreibung.
- Schlag eine Begründung vor, die ein Prüfer ohne Rückfrage versteht: was der
  Befund ist, warum er die Zahlen nicht berührt, wann er erledigt wird.
- Wertungsfrage: vertretbar, wenn der Befund die Umsatzsteuer des Zeitraums
  nicht berührt und die Klärung außerhalb von Buchfink liegt (etwa eine
  Rücklastschrift, auf die der Kunde antworten muss). Berührt er die Steuer,
  wird erst behoben.

## 3. Was nach der Festschreibung gilt

- Korrektur per Storno mit heutigem Datum
  ([grundlagen/arbeitsweise.md](../grundlagen/arbeitsweise.md#11-eine-buchung-korrigieren)).
- Abschlussbuchungen bleiben in der Abschlussperiode möglich, bis das Jahr
  festgeschrieben ist
  ([jahresabschluss/abschlussbuchungen.md](../jahresabschluss/abschlussbuchungen.md)).
