# Prüfmaterial für die E-Rechnungs-Module

Diese Dateien stammen nicht von Buchfink. Sie liegen hier, damit die
Prüfungen ohne Netz und ohne Vorbereitung laufen — der Beleg dafür, dass die
Umsetzung tut, was sie behauptet, gehört ins Repository und nicht in eine
Anleitung.

## `en16931/`

Aus **ConnectingEurope/eInvoicing-EN16931**, dem Validierungsartefakt der
Europäischen Kommission. Lizenz: **EUPL-1.2**, dieselbe wie Buchfink; der
Lizenztext liegt als `en16931/LICENSE.txt` daneben.

| Verzeichnis | Herkunft im Artefakt | Wofür |
|---|---|---|
| `cii-examples/` | `cii/examples/` | 15 gültige CII-Rechnungen |
| `ubl-examples/` | `ubl/examples/` | 18 gültige UBL-Rechnungen |
| `rules-invoice/` | `test/Invoice-unit-UBL/` | eine Datei je Geschäftsregel, mit Erwartung |
| `rules-creditnote/` | `test/CreditNote-unit-UBL/` | dasselbe für Gutschriften |

Nicht übernommen wurde `ubl/examples/FT G2G_TD01 con Allegato, Bonifico e
Split Payment.xml`: eine einzige Datei von 3,3 MB, deren eingebetteter Anhang
den Korpus verzehnfacht hätte. Sie prüft nichts, was die übrigen nicht auch
prüfen.

**Warum beide Sammlungen.** Die Beispiele sind gültige Rechnungen: sie zeigen,
dass eine Prüfung nicht zu streng ist. Dass sie überhaupt greift, können sie
nicht zeigen — ein gültiges Dokument löst keine Regel aus. Dafür sind die
Regeldateien da, die je Regel sagen, ob sie anschlagen muss oder schweigen.

## `../xrechnung/testdata/kosit/`

Aus **itplr-kosit/validator-configuration-xrechnung**. Lizenz: **Apache-2.0**,
Lizenztext liegt daneben. Es sind die Testinstanzen, für die KoSIT in
`assertions.xml` zusichert, ob sie gültig sind — eine ja, acht nein. Damit
lässt sich das eigene Urteil gegen ein fremdes stellen, und zwar in beide
Richtungen.
