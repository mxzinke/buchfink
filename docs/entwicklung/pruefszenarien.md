# Prüfszenarien

[Dokumentation](../README.md) · [Entwicklung](README.md)

Diese Seite bündelt wiederholbare Prüfabläufe und deren Aussagegrenzen.
Anleitungen für die echte Anwendung mit isolierten Beispieldaten stehen unter
[scripts/bug-hunt](../../scripts/bug-hunt/README.md). Prüfergebnisse und Aufnahmen
liegen lokal unter `.cache/`; sie werden nicht mit dem Quellcode veröffentlicht.

## Vom Abschluss zum Beleg

Ein sachverständiger Dritter muss sich in Buchfink ohne Erklärung zurechtfinden
(GOB-02, GoBD Rz. 145). Das Prüfszenario stellt diese Behauptung auf die
Probe: ein Beispielmandant, eine Aufgabe, eine Klickgrenze. Es läuft als
automatischer Test und lässt sich von Hand nachvollziehen.

### Der Beispielmandant

Die Beispieldaten des Screenshot-Werkzeugs (`scripts/site-screenshots/mock-bridge.ts`)
bilden einen kleinen Mandanten mit einem Geschäftsjahr, Bankkonto 1800,
Ausgangs- und Eingangsrechnungen, Bankumsätzen, Belegen und einer Bilanz. Die
Zahlen sind gerechnet, nicht gewürfelt: Soll gleich Haben, Aktiva gleich
Passiva, die Zahllast aus Umsatzsteuer und Vorsteuer. Wer eine Zahl ändert,
zieht die verbundenen mit.

### Die Aufgabe

Von der Bilanzposition, unter der das Bankkonto steht, zum Beleg der Buchung
B-2026-0052 in höchstens vier Klicks. Der Weg zur Bilanz selbst zählt nicht;
das Szenario beginnt dort, wo ein Prüfer sitzt, wenn er eine Position
hinterfragt.

| Klick | Wo | Was passiert |
|---|---|---|
| 1 | Bilanz, Position „Umlaufvermögen" | Die Position klappt auf und zeigt ihre Konten. |
| 2 | Konto 1800 | Das Kontoblatt öffnet sich mit allen Buchungen des Jahres. |
| 3 | Zeile B-2026-0052 | Die Buchung öffnet sich im Journal mit Soll und Haben. |
| 4 | „Beleg zur Buchung anzeigen" | Der versiegelte Beleg BE-2026-0229 erscheint mit seiner Datei. |

Jeder Schritt führt in die Richtung Beleg, keiner in ein Menü. Die Rückrichtung
funktioniert genauso: vom Beleg zur Buchung, von der Buchung ins Kontoblatt.

### Der Test

```bash
npm --prefix frontend install
npm --prefix scripts/site-screenshots install
task check:clicks          # oder: node scripts/site-screenshots/four-clicks.mjs
```

Der Test startet den Vite-Server mit der echten Oberfläche und der
Beispiel-Bridge, klickt den Weg und zählt. Er schlägt fehl, wenn ein Schritt
nicht klickbar ist oder mehr als vier Klicks nötig sind. Ändert sich die
Oberfläche so, dass der Weg länger wird, ist das ein Befund und keine
Anpassung der Grenze.

### Erwartetes Ergebnis

```
  1. Bilanzposition „Umlaufvermögen" aufklappen
  2. Konto 1800 öffnen
  3. Buchung B-2026-0052 öffnen
  4. Beleg zur Buchung B-2026-0052 öffnen

Bilanz → Konto → Buchung → Beleg in 4 Klicks (erlaubt: 4).
```

## Einrichtung, laufende Buchhaltung und Jahresabschluss

Für diese Abläufe werden neue Testunternehmen in getrennten Ordnern angelegt.
Die echte Oberfläche verwendet Wails-Bridge, SQLite, Typst und den
Betriebssystem-Schlüsselbund. Es werden keine Rechnungen versendet und keine
Meldungen eingereicht. Eine Festschreibung fordert einen echten Zeitstempel
für einen Prüfwert an; externe Offenlegungsreferenzen müssen als Softwaretest
gekennzeichnet sein.

| Ablauf | Beispieldaten und erwartetes Ergebnis | Automatisierte Prüfung |
|---|---|---|
| Einrichtung | Leuchtspur Design UG beginnt 2025. Das gewählte Jahr bleibt aktiv. Schlüsselsicherung wird angeboten; Abbruch zählt nicht als Sicherung. Ein belegter Datenordner wird abgewiesen. | `internal/wailsbridge/recovery_onboarding_test.go`, `tenant_directory_test.go`, `bughunt_year_test.go` |
| Bankimport | Zwei Euro-Konten und drei Umsätze in einer CAMT-Datei. Neue Konten werden beim Import eingerichtet. Die Originaldatei bleibt als Nachweis erhalten; ein identischer Wiederholungsimport erzeugt keine zusätzlichen Umsätze. | `internal/service/bank_accounts_test.go`, `internal/bank/bughunt_test.go`, `internal/repository/bughunt_bank_test.go` |
| Teilzahlung und Skonto | Auf eine Rechnung über 1.190 € werden 500 € zugeordnet; 690 € bleiben offen. Im Skontoszenario gleichen weitere 666,20 € Zahlung und 23,80 € Skonto den Posten aus; die Umsatzsteuer beträgt danach 186,20 €. | `internal/service/payment_service_test.go`, `internal/accounting/ustva_test.go` |
| Rechnungsdokumente | Vollständigen Firmenempfänger, Verkäuferdaten und Ansprechpartner in PDF/XML prüfen. Berichtigung und Storno erhalten ihre Bezüge. ZUGFeRD und XRechnung mit unabhängigen Werkzeugen validieren. | `internal/invoice/business_letters_test.go`, `internal/invoice/cii_content_test.go` |
| Jahresabschluss | Nordlicht Nachprüfung UG: Rumpfjahr 2025, 5.000 € Kapital, 10.000 € Nettoumsatz, Schreibtisch und Hosting. Nach 46,16 € Abschreibung, 900 € Abgrenzung und Steuerrückstellungen ergeben sich 6.782,12 € Jahresüberschuss, 1.695,53 € UG-Rücklage, 5.086,59 € Bilanzgewinn und 16.097,84 € Bilanzsumme. Feststellung, Nachweisabfrage und Vortrag nach 2026 prüfen. | [Szenarioskript](../../scripts/bug-hunt/followup-year-end.mjs), `internal/service/ug_reserve_test.go`, `statement_header_lists_test.go` |
| Export und Sicherung | Unternehmensdokumente im Prüferpaket nachweisen. CSV-Spalten, XML-Index, Größen und SHA-256 prüfen. Sicherung in einen neuen Ordner wiederherstellen; Buchungen, Dokumente, Abschlussstand und Hashketten vergleichen. Beschädigte Dateien müssen erkannt werden. | `internal/service/audit_followup_test.go`, `internal/wailsbridge/recovery_onboarding_test.go` |
| Verschlüsselungsmigration | Alten Datenbestand auf Schema 10 migrieren. Namen, Positionstexte und Protokollbeschreibungen sind anschließend in den geprüften SQLite-, WAL- und SHM-Dateien nicht mehr im Klartext auffindbar; Buchungen und Hashketten bleiben gültig. | `internal/repository/encryption_migration_test.go` |
| Nummernkreise | Unter Einstellungen → Rechnungsstellung aufklappen. Jahreswechsel lädt den passenden Bericht; ungespeicherte Formate bleiben erhalten. Lückenbegründung, Schreibsperre im abgeschlossenen Jahr und erneutes Laden nach einem Fehler prüfen. | Browserprüfung; Ladefehler und Lücken können durch kontrollierte Antworten simuliert werden. |

Die Skripte enthalten feste Testnamen und Pfade. Vor einem neuen Lauf müssen
sie geprüft und gegebenenfalls angepasst werden. `year-end.mjs` dokumentiert
den ursprünglichen Fehlerlauf; für den korrigierten Abschluss ist
`followup-year-end.mjs` maßgeblich. Die Gewerbeanmeldung war nicht Bestandteil
der fachlichen Nachprüfung.

## Bisherige Nachweise und ihre Grenzen

Der Lauf vom September 2026 prüfte 20 Hauptansichten ohne JavaScript-Ausnahme.
Die neuen Onboarding-, Bank- und Abschlussabläufe sind in den Videos 05 bis 07
unter `.cache/bughunt-2026-09-12/videos/` aufgezeichnet. Die Videos 01 bis 04
zeigen frühere Zustände, teilweise vor der Fehlerbehebung. Die zugehörigen
Exporte, Prüfprotokolle und historischen Berichte liegen im selben lokalen
Ergebnisordner. Die beiden früher versionierten Arbeitsberichte sind außerdem
im Git-Verlauf bis einschließlich Commit `0ac5d20` nachlesbar.

Die erzeugten Rechnungsbeispiele bestanden CII-Schema und EN 16931 v1.3.16,
die XRechnung zusätzlich KoSIT Validator 1.6.3 mit Konfiguration 3.0.2 vom
31. August 2026. veraPDF 1.30.2 bestätigte PDF/A-3b für das ZUGFeRD-Beispiel.
Der XML-Index des Prüferpakets bestand die Prüfung mit der mitgelieferten
deterministischen DTD; CSV-Spalten und Manifestdateien wurden abgeglichen.
Das belegt die geprüften Beispiele, keine Zertifizierung aller Exportfälle.

Im Wails-Serverbetrieb fehlen native Dateiauswahldialoge. Bei den entsprechenden
Walkthroughs wurde nur die Dateiauswahl durch einen festen Testpfad ersetzt;
Dateien und Buchungen entstanden über das Backend. Kontrollierte Antworten für
den Nummernkreis-Dialog prüfen dessen Bedienung und API-Aufrufe, nicht die
Persistenz. Diese Unterschiede müssen auch bei künftigen Aufnahmen erkennbar
bleiben.

Ein Import in IDEA, eine ERiC-Abnahme, die Annahme beim Unternehmensregister
und eine Wiederherstellung auf einem zweiten physischen Rechner sind nicht
belegt. Auch vollständige Steuerfallabdeckung und Verständlichkeit für die
Zielgruppe erfordern weitere Prüfung. Bekannte Produktgrenzen stehen im
[Umsetzungsstand](../projekt/umsetzungsstand.md), ausstehende Arbeiten in der
[Roadmap](../projekt/roadmap.md).
