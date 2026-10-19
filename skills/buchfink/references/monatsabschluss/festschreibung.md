# Prüflauf und Festschreibung

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Grundsätze zur Unveränderbarkeit und Festschreibung stehen in
[grundlagen/gobd.md](../grundlagen/gobd.md#7-unveränderbarkeit-storno-festschreibung);
hier steht, was Buchfink vor und nach der Festschreibung prüft.

## Inhalt

1. Wozu
2. Befunde des Prüflaufs
3. Befunde übergehen
4. Was nach der Festschreibung gilt
5. Takt

## 1. Wozu

Die Festschreibung sichert die Unveränderbarkeit
([grundlagen/gobd.md](../grundlagen/gobd.md#7-unveränderbarkeit-storno-festschreibung)).
Buchfink schreibt nur fest, was der Prüflauf freigibt oder der Mensch mit
Begründung übergeht.

## 2. Befunde des Prüflaufs

`auswertung` (`art: pruefbericht`) liefert Befunde mit Schwere und Objekt.
Blockierend heißt: Die Festschreibung geht nur mit Begründung des Menschen.

| Regel | Schwere | Bedeutung | Vorgehen |
|---|---|---|---|
| `entry_without_receipt` | blockierend | Buchung ohne Beleg | Beleg ablegen und verknüpfen oder Eigenbeleg ([grundlagen/gobd.md](../grundlagen/gobd.md#8-belegwesen-und-eigenbeleg)) |
| `receipt_unbooked` | blockierend bis Stichtag, sonst Hinweis | Beleg abgelegt, nicht gebucht | buchen ([belege/ablauf.md](../belege/ablauf.md)) oder verwerfen; bei Anzahlungsfällen nur Hinweis |
| `bank_unmatched` | blockierend | Bankumsatz offen | zuordnen oder mit Grund ignorieren ([bank/zuordnung.md](../bank/zuordnung.md)) |
| `interim_balance` | blockierend | Saldo auf 1460 Geldtransit oder 1370 Durchlaufende Posten | Gegenbuchung finden ([bank/kreditkarte-geldtransit.md](../bank/kreditkarte-geldtransit.md)) |
| `account_unmapped` | blockierend | Konto mit Saldo ohne Bilanz- oder GuV-Posten | Konto prüfen; Fehlbuchung stornieren |
| `depreciation_missing` | blockierend | Abschreibung des Jahres fehlt (Jahresabschluss) | [jahresabschluss/abschreibungen.md](../jahresabschluss/abschreibungen.md) |
| `receipt_overdue` | Hinweis | Beleg liegt länger ungebucht | buchen |
| `duplicate_receipt`, `duplicate_payment` | Hinweis | Dublettenverdacht | prüfen, ggf. verwerfen oder stornieren |
| `number_gap` | Hinweis | Lücke im Nummernkreis | Ursache nennen |
| `vat_return_missing` | Hinweis | Voranmeldung fehlt | [ablauf.md](ablauf.md#2-ablauf), Schritt 5 |
| `commit_overdue`, `period_not_committed` | Hinweis | Monat nicht festgeschrieben, ab zwei Monaten nach Monatsende als Rückstand | Festschreibung anstoßen |
| `vat_id_unconfirmed` | Hinweis | USt-IdNr. des EU-Kunden unbestätigt | [rechnungen/ausland.md](../rechnungen/ausland.md) |
| `service_proof_missing` | Hinweis | Leistungsnachweis fehlt über der Grenze | Vermerk beim Menschen anfordern |
| `provision_discount`, `closing_step_skipped`, `size_class_change` | Hinweis | Jahresabschluss | [jahresabschluss/ablauf.md](../jahresabschluss/ablauf.md) |

Neue Regeln können hinzukommen. Unbekannte Regeln liest du an `meldung` und
Fundstelle ab.

Fertig mit einem Befund, wenn die Ursache behoben ist (neuer Prüflauf zeigt
ihn nicht mehr) oder der Mensch entschieden hat, ihn zu übergehen.

## 3. Befunde übergehen

- Nur der Mensch übergeht einen blockierenden Befund, in der App bei der
  Festschreibung. Buchfink speichert die Begründung am Prüflauf, damit ein
  Prüfer sie findet.
- Schlag eine Begründung vor, die der Prüfer ohne Rückfrage versteht: was der
  Befund ist, warum er die Zahlen nicht berührt, wann er erledigt wird.
- Wertungsfrage: Übergehen ist vertretbar, wenn der Befund die Umsatzsteuer
  des Zeitraums nicht berührt und die Klärung außerhalb von Buchfink liegt
  (z. B. eine Rücklastschrift, bei der der Kunde noch antworten muss). Berührt
  er die Steuer, wird erst behoben.

## 4. Was nach der Festschreibung gilt

- Buchfink nimmt keine Buchung mit Datum bis `mandant.festgeschrieben_bis`
  an (`fachliche_regel`). Korrigiert wird per Storno mit heutigem Datum
  ([grundlagen/arbeitsweise.md](../grundlagen/arbeitsweise.md#11-eine-buchung-korrigieren)).
- Nach der Festschreibung des letzten Monats bleiben Abschlussbuchungen in der
  Abschlussperiode möglich, bis das Jahr festgeschrieben ist
  ([jahresabschluss/abschlussbuchungen.md](../jahresabschluss/abschlussbuchungen.md)).
- Das Transferticket einer Voranmeldung nimmt Buchfink erst nach der
  Festschreibung ihres Zeitraums an ([ablauf.md](ablauf.md)).

## 5. Takt

Buchfink führt jeden Monat als Pflicht `monatsfestschreibung` in `fristen`;
Fälligkeit und Nachfrist:
[grundlagen/pflichten.md#monatsfestschreibung](../grundlagen/pflichten.md#monatsfestschreibung).
Auch bei vierteljährlicher Voranmeldung schreibt der Mensch jeden Monat fest
([ablauf.md](ablauf.md#2-ablauf), Schritt 1).

