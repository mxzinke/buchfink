# Anzahlungen und Rechnungsverbund

[Fachkonzepte](README.md) · [Belege und Buchungen](beleg-buchungsflow.md)

Anforderungen: UST-02, RECH-02, RECH-03, RECH-10, BEL-07. Code:
`internal/service/advance_service.go`.

## Warum das kein gewöhnlicher Rechnungsfall ist

Bei einer Anzahlung entsteht die Umsatzsteuer mit der Vereinnahmung, auch bei
Sollversteuerung (§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG). Die
Schlussrechnung muss die Teilentgelte samt Steuer absetzen, über die
Anzahlungsrechnungen ausgestellt und die vereinnahmt wurden
([§ 14 Abs. 5 Satz 2 UStG][ustg14]). Wer das vergisst, weist die Steuer zweimal
aus und schuldet sie nach § 14c UStG.

Buchfink hält deshalb Abschläge und Schlussrechnung in einem
**Rechnungsverbund** zusammen. Die Schlussrechnung ist nicht erstellbar, ohne
dass der Verbund alle berechneten und vereinnahmten Anzahlungen absetzt. Welche
das sind, entscheidet der Verbund, nicht die Auswahl des Menschen.

## Erhaltene Anzahlungen

| Schritt | Buchung |
|---|---|
| Abschlagsrechnung ausstellen | keine; Nummer und Dokument wie jede Rechnung |
| Zahlung geht ein | **1800** Bank an **3272** Erhaltene, versteuerte Anzahlungen 19 % + **3806** Umsatzsteuer |
| Schlussrechnung | Debitor (Rest) + **3272** + **3806** an **4400** Erlöse (gesamt) + **3806** Umsatzsteuer (gesamt) |
| Rückzahlung | **3272** + **3806** an **1800** Bank, im Zeitraum der Rückzahlung |

Den offenen Posten eines Abschlags rechnet `service.InvoiceService.OpenAdvanceItems`,
die Rückzahlung bucht `service.InvoiceService.RefundAdvance`; weder App noch MCP
erreichen sie. Ein gestellter Abschlag fehlt deshalb in `offene_posten` und in
den Bankvorschlägen. Den Zahlungseingang bucht die App
(`service.InvoiceService.SettleAdvance`) nur aus einem solchen Vorschlag, und
solange ein Abschlag nicht vereinnahmt ist, lehnt Buchfink die Schlussrechnung
ab.

Die Anzahlungsrechnung ist eine vollwertige Rechnung nach § 14 UStG mit Nummer
aus demselben Kreis. Statt des Leistungszeitpunkts nennt sie den Zeitpunkt der
Vereinnahmung, wenn er feststeht. Der Zahlungseingang muss den Abschlag ganz
decken; eine Teilzahlung zählt nicht als Vereinnahmung des ganzen Abschlags.

Für 7 % gibt es **3260**. Erhaltene Anzahlungen stehen unter den
Verbindlichkeiten; offen von den Vorräten absetzen (**1190**) lässt sich nur,
was auf Vorräte angezahlt wurde ([§ 268 Abs. 5 Satz 2 HGB][hgb268]).

## Regeln des Verbunds

- Die Abschläge zusammen übersteigen den Gesamtbetrag nicht.
- Ein Verbund mit Schlussrechnung ist abgeschlossen.
- Eine stornierte Abschlagsrechnung fällt aus der Verrechnung.
- Eine bezahlte Abschlagsrechnung lässt sich erst nach der Rückzahlung
  stornieren, weil die Steuer bis dahin geschuldet bleibt (§ 17 Abs. 2 Nr. 2
  UStG).
- Das Storno einer Schlussrechnung öffnet den Verbund wieder, in derselben
  Transaktion.

Die Regeln zu Storno und Rückzahlung stehen im Dienst. Storno
(`service.InvoiceService.CorrectInvoice`) und Rückzahlung haben keinen
Erfassungsweg in App oder MCP.

## Geleistete Anzahlungen

Auf der Eingangsseite ist die Vorsteuer mit der Zahlung abziehbar, sobald eine
Rechnung vorliegt. Das Anzahlungskonto richtet sich nach dem, wofür angezahlt
wird:

| Zweck | Konto |
|---|---|
| immaterielle Vermögensgegenstände | **0170** |
| Sachanlagen und Anlagen im Bau | **0700** |
| Vorräte | **1180** |

Eingehende E-Rechnungen vom Typ Anzahlungsrechnung erkennt Buchfink; einen
Buchungsvorschlag aus dem Datensatz macht es dafür nicht, weil Zeitpunkt und
Konto andere sind als bei einer gewöhnlichen Rechnung.

[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[hgb268]: https://www.gesetze-im-internet.de/hgb/__268.html
