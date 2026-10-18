# Prüflauf und Festschreibung

Rechtsstand: 17. Oktober 2026. Grundsätze zur Unveränderbarkeit stehen in
buchfink-grundlagen (GoBD); hier steht, was Buchfink vor und nach der
Festschreibung prüft.

## Inhalt

1. Wozu
2. Befunde des Prüflaufs
3. Befunde übergehen
4. Was nach der Festschreibung gilt
5. Takt

## 1. Wozu

Eine Buchung darf nicht so verändert werden, dass der ursprüngliche Inhalt
nicht mehr feststellbar ist ([§ 146 Abs. 4 AO][ao146]). Vorerfassungen bis zum
Jahresabschluss offen zu halten ist unzulässig; Festschreibung ist eines der
Mittel, die Unveränderbarkeit zu sichern ([GoBD Rz. 109–110][gobd]). Buchfink
schreibt nur fest, was der Prüflauf freigibt oder der Mensch mit Begründung
übergeht.

## 2. Befunde des Prüflaufs

`auswertung` (`art: pruefbericht`) liefert Befunde mit Schwere und Objekt.
Blockierend heißt: Die Festschreibung geht nur mit Begründung des Menschen.

| Regel | Schwere | Bedeutung | Vorgehen |
|---|---|---|---|
| `entry_without_receipt` | blockierend | Buchung ohne Beleg | Beleg ablegen und verknüpfen oder Eigenbeleg (grundlagen) |
| `receipt_unbooked` | blockierend bis Stichtag, sonst Hinweis | Beleg abgelegt, nicht gebucht | buchen (belege) oder verwerfen; bei Anzahlungsfällen nur Hinweis |
| `bank_unmatched` | blockierend | Bankumsatz offen | zuordnen oder mit Grund ignorieren (bank) |
| `interim_balance` | blockierend | Saldo auf 1460 Geldtransit oder 1370 Durchlaufende Posten | Gegenbuchung finden (bank) |
| `account_unmapped` | blockierend | Konto mit Saldo ohne Bilanz- oder GuV-Posten | Konto prüfen; Fehlbuchung stornieren |
| `depreciation_missing` | blockierend | Abschreibung des Jahres fehlt (Jahresabschluss) | buchfink-jahresabschluss |
| `receipt_overdue` | Hinweis | Beleg liegt länger ungebucht | buchen |
| `duplicate_receipt`, `duplicate_payment` | Hinweis | Dublettenverdacht | prüfen, ggf. verwerfen oder stornieren |
| `number_gap` | Hinweis | Lücke im Nummernkreis | Ursache nennen |
| `vat_return_missing` | Hinweis | Voranmeldung fehlt | Ablauf in SKILL.md |
| `commit_overdue`, `period_not_committed` | Hinweis | Monat nicht festgeschrieben, ab zwei Monaten nach Monatsende als Rückstand | Festschreibung anstoßen |
| `vat_id_unconfirmed` | Hinweis | USt-IdNr. des EU-Kunden unbestätigt | buchfink-rechnungen |
| `service_proof_missing` | Hinweis | Leistungsnachweis fehlt über der Grenze | Vermerk beim Menschen anfordern |
| `provision_discount`, `closing_step_skipped`, `size_class_change` | Hinweis | Jahresabschluss | buchfink-jahresabschluss |

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
  (grundlagen).
- Die Festschreibung verankert den Stand der Hashkette mit einem Zeitstempel.
- Nach der Festschreibung des letzten Monats bleiben Abschlussbuchungen in der
  Abschlussperiode möglich, bis das Jahr festgeschrieben ist
  (buchfink-jahresabschluss).
- Erst danach nimmt Buchfink das Transferticket der Voranmeldung an.

## 5. Takt

Die GoBD nennen keinen festen Takt. Buchfink führt jeden Monat als Pflicht
`monatsfestschreibung` in `fristen`; Fälligkeit und Nachfrist:
buchfink-grundlagen, references/pflichten.md, Abschnitt `monatsfestschreibung`. Bei vierteljährlicher
Voranmeldung schreibt der Mensch trotzdem jeden Monat fest; die Voranmeldung
folgt nach dem dritten Monat.

[ao146]: https://www.gesetze-im-internet.de/ao_1977/__146.html
[gobd]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf
