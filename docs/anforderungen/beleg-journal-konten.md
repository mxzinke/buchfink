# B. Beleg, Journal, Konten

[Anforderungskatalog](README.md) · [Legende](README.md#legende) · [Fachkonzepte](../fachkonzepte/README.md)

### BEL-01 Keine Buchung ohne Beleg `MUSS`

**Norm:** § 238 Abs. 1 S. 3 HGB, § 145 Abs. 1 AO, GoBD Rz 61 ff.

**Bedeutung:** Der Beleg verbindet den Geschäftsvorfall mit der Buchung. Fehlt er, fehlt der Nachweis, und die Buchung ist formal angreifbar. Bei fehlendem Fremdbeleg tritt ein Eigenbeleg an seine Stelle.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Jede Buchung referenziert einen Beleg oder einen Eigenbeleg | ✅ | internal/service/posting_service.go:237 erzwingt die Referenz auf dem Belegweg, internal/service/manual_entry.go:48-61 seit Welle 8 auch auf dem Handweg: ohne abgelegten Beleg und ohne die Angaben eines Eigenbelegs entsteht keine Buchung, und beides zugleich wird abgewiesen. Die Ausgangsrechnung hat den Verweis seit Welle 5b nicht an der Buchung — das Dokument entsteht hinter der Transaktion aus Nummer und Buchung, ein Nachtrag bräche die Hashkette —, der Beleg wird stattdessen auf die Buchung versiegelt (internal/service/invoice_service.go:686-734). AfA, Zahlung, Saldenvortrag und Abschlussbuchung bleiben ohne eigenen Beleg: sie sind Folgebuchungen, ihr Nachweis ist die Buchung, auf die sie sich beziehen, und der Grund steht an der Regel (internal/service/check_service.go:410-421) | – |
| Eigenbelege gekennzeichnet, mit Aussteller, Datum, Betrag, Grund und erfassender Person | ✅ | internal/service/self_issued_receipt.go:68-131 (`CreateSelfIssued`) verlangt Grund, Datum und Betrag, nimmt den Aussteller aus den Unternehmensangaben (:164-181) und die erfassende Person aus internal/actor; :209-234 setzt daraus das Dokument mit dem Satz, dass kein Fremdbeleg vorliegt, :113-129 legt es als Original unter der Belegart `self_issued` ab und lässt es die Kopfdatenprüfung durchlaufen. Scheitert die Buchung danach, wird die Datei wieder eingesammelt (:145-162). Die Oberfläche legt den Eigenbeleg für sich an oder im selben Zug mit der Handbuchung (frontend/src/components/LedgerForms.tsx:135-260) | – |
| Buchungen ohne Belegzuordnung im Prüfbericht, verhindern die Festschreibung | ✅ | internal/service/check_service.go:307-343 (`entry_without_receipt`) meldet jede Buchung ohne Beleg als blockierenden Befund — der Nachweis zählt in beide Richtungen (:289-306), die Generalumkehr erbt ihn von der Ursprungsbuchung —, internal/wailsbridge/festschreibung_service.go:41-43, :93-103 hängt die Festschreibung daran | – |
| Belegverweis bidirektional | ✅ | internal/domain/journal.go:129-131, internal/domain/receipt.go:152 bilden beide Richtungen im Datenmodell ab; seit Welle 8 auch die Oberfläche: die Journalzeile führt zum Beleg (frontend/src/pages/JournalPage.tsx:489-491, :605-618), der Beleg über internal/wailsbridge/welle8_service.go:89-95 (`GetEntriesForReceipt`) zu seinen Buchungen und von dort ins Journal und ins Kontoblatt (frontend/src/pages/ReceiptsPage.tsx:1169-1250) | – |

**Stand.** Mit Welle 8 entsteht keine Buchung mehr ohne Beleg. Die Handbuchung verlangt entweder den abgelegten Beleg oder die Angaben eines Eigenbelegs, der in derselben Transaktion als PDF entsteht, abgelegt und auf die Buchung versiegelt wird; scheitert die Buchung danach, sammelt Buchfink die Datei wieder ein. Der Eigenbeleg hat Aussteller, Datum, Betrag, Grund und die erfassende Person und sagt auf dem Dokument, dass kein Fremdbeleg vorliegt. Der Verweis gilt in beide Richtungen und ist in der Oberfläche begehbar. AfA, Zahlung, Saldenvortrag und Abschlussbuchung bleiben Folgebuchungen ohne eigenen Beleg — ihr Nachweis ist die Buchung, auf die sie sich beziehen, und der Prüfbericht verlangt für sie keinen. Alle vier Kriterien sind erfüllt.

### BEL-02 Belegangaben und eindeutige Belegnummer `MUSS`

**Norm:** GoBD Rz 64, 71 bis 77

**Bedeutung:** Die GoBD zählen die Mindestangaben eines Belegs auf. Die eindeutige Belegnummer ist das technische Bindeglied zwischen Papier- oder Dateiablage und Buchungssatz.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Je Beleg Nummer, Belegdatum, Aussteller oder Empfänger, Betrag mit Währung, Steuersatz und Steuerbetrag, Leistungsbeschreibung, Erfassungszeitpunkt | ✅ | internal/domain/receipt.go:196-225 führt neben Nummer, Richtung, Belegart, Eingangsdatum, Eingangsweg und Erfassungszeitpunkt seit Welle 6 auch Belegdatum, Aussteller, Brutto- und Steuerbetrag, Währung und Betreff am Beleg selbst; bei einer E-Rechnung kommen sie aus dem strukturierten Teil (internal/service/einvoice_service.go:190-212), sonst aus dem Ablagedialog (frontend/src/pages/ReceiptsPage.tsx:496-640). Der Steuersatz steht als Steuerschlüssel an der Steuerzeile der Buchung (internal/domain/journal.go:84-96), weil ein Beleg mehrere Sätze hat. Die Kopfdaten gehen in den Beleg-Hash und damit in jede Buchung, die auf ihn zeigt (internal/accounting/receipthash.go:40-61), und stehen in der Datenüberlassung (internal/service/export_tables.go:826-834) | – |
| Belegnummer systemseitig vergeben, eindeutig, nicht nachträglich änderbar | ✅ | internal/repository/receipt_gorm.go:90-113 vergibt die Nummer in derselben Transaktion, die den Beleg schreibt, `uniqueIndex` auf der Nummer, kein Änderungsweg | – |
| Fehlende Pflichtangaben blockieren die Freigabe und werden benannt | ✅ | internal/domain/receipt.go:429-473 (`ValidateHeader`) nennt jede fehlende Angabe einzeln und je Belegart: das Belegdatum immer, Aussteller und Betrag bei Rechnungen, den Betreff bei Eigenbeleg, Handelsbrief und sonstigem Dokument; :396-414 (`ValidateBookable`) hängt das Buchen daran, die Prüfung je Datei steht daneben (:319-392) | – |
| Belegnummernkreis je Geschäftsjahr und Belegart konfigurierbar, ohne Doppelvergabe | ✅ | internal/domain/numberrange.go:21-26 trennt je Jahr und Art, internal/repository/numberrange_gorm.go:22-30 vergibt ohne Doppelvergabe. Die Systematik ist seit Welle 8 auch für den Belegkreis einstellbar: internal/domain/numberrange.go:60-104 kennt `{JAHR}` und `{NR:n}` mit derselben Prüfung und derselben Rückleseheuristik wie der Rechnungskreis, internal/domain/settings.go:43-45 hält die Einstellung, internal/repository/receipt_gorm.go:345-364 liest sie in derselben Transaktion, in der die Nummer vergeben wird — ein zwischendurch geändertes Format ergäbe sonst eine Nummer nach der alten Systematik mit dem Zähler der neuen —, frontend/src/pages/SettingsPage.tsx:616-625 stellt sie ein. Bestehende Nummern bleiben gültig | – |

**Stand.** Die Nummernvergabe ist vorbildlich, und seit Welle 6 hat der Beleg seine Kopfdaten selbst: Belegdatum, Aussteller, Betrag, Steuer, Währung und Betreff kommen bei einer E-Rechnung aus dem strukturierten Teil und sonst aus dem Ablagedialog — dort freiwillig, vor dem Buchen Pflicht, mit einer eigenen Meldung je fehlender Angabe. Welle 8 macht die Systematik des Belegkreises einstellbar, wie sie es beim Rechnungskreis seit Welle 5b ist: das Format steht in den Einstellungen, gelesen wird es in derselben Transaktion, die die Nummer vergibt, und der Lückenbericht liest auch Nummern aus einem früher eingestellten Format zurück. Bestehende Nummern bleiben gültig. Alle vier Kriterien sind erfüllt.

### BEL-03 Belegsicherung `MUSS`

**Norm:** GoBD Rz 67 bis 70

**Bedeutung:** Der Beleg ist gegen Verlust zu sichern, sobald er im Unternehmen eingeht. Zwischen Eingang und Sicherung darf kein Zeitraum liegen, in dem ein Beleg spurlos verschwinden kann.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Eingehender Beleg unveränderlich gespeichert, mit Zeitstempel, Quelle und Prüfsumme | ✅ | internal/receiptstore/store.go:104-120 legt die Datei unter ihrem SHA-256 ab, internal/domain/receipt.go:92, :127 führen `CreatedAt`, `ReceivedAt`, `ReceivedVia` | – |
| Löschen ausgeschlossen, fehlerhafte Belege storniert und mit Vermerk sichtbar | ✅ | internal/service/receipt_service.go:473-487 (`Discard`) verlangt eine Begründung und hält den Beleg sichtbar; die empfangene Originaldatei lässt sich seit Welle 6 nicht mehr entfernen (:230-244, mit dem Grund aus GoBD Rz. 131), und wird eine andere Datei eines noch offenen Belegs entfernt, stehen ihr Name, ihre Rolle und ihre Prüfsumme im Änderungsprotokoll (:252-257) | – |
| Import erst mit persistierter Prüfsumme, keine halb gespeicherten Zustände | ✅ | internal/service/receipt_service.go:98-153, internal/receiptstore/store.go:58-62; Dateien gehen atomar auf die Platte, dann läuft die Transaktion | – |

**Stand.** Die Belegsicherung ist der solideste Teil des Moduls, und mit Welle 6 ist auch die letzte Lücke zu: die empfangene Originaldatei lässt sich aus einem offenen Beleg nicht mehr herausnehmen, und das Entfernen einer Darstellung oder eines Anhangs hinterlässt Name und Prüfsumme im Protokoll statt einer Zahl. Alle drei Kriterien sind erfüllt.

### BEL-04 Zeitgerechte Erfassung und Festschreibungsfristen `MUSS`

**Norm:** § 146 Abs. 1 AO, GoBD Rz 45 bis 52

**Bedeutung:** Die Finanzverwaltung nennt konkrete Zeiträume. Unbare Geschäftsvorfälle sind innerhalb von zehn Tagen zu erfassen. Kasseneinnahmen und Kassenausgaben sind täglich festzuhalten. Waren- und Kostenrechnungen müssen innerhalb von acht Tagen kontokorrentmäßig erfasst werden. Die endgültige Verbuchung ist bis zum Ablauf des Folgemonats unbedenklich, wenn die Grundaufzeichnung vorher erfolgt ist.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Bericht über Belege, deren Eingang mehr als zehn Tage zurückliegt und die nicht erfasst sind | ✅ | internal/service/check_service.go:445-505 (`receipt_overdue`) misst den Belegeingang gegen die Frist aus internal/domain/settings.go:60-63 (`ReceiptCaptureDays`, ohne Einstellung zehn Tage nach GoBD Rz. 47) und nennt jeden liegen gebliebenen Beleg mit Nummer und Eingangstag | – |
| Festschreibung des Vormonats erzwingen oder erinnern, Schwellenwert konfigurierbar | ✅ | internal/service/deadline_service.go:285-317 führt je Monat einen Festschreibungstermin zum Ende des Folgemonats zuzüglich der Nachfrist aus internal/domain/settings.go:64-67 (`CommitGraceDays`) und gilt als erledigt, sobald der Monat festgeschrieben ist; internal/service/check_service.go:700-731 (`commit_overdue`) mahnt denselben Tag im Prüfbericht an, internal/service/task_service.go:344-390 stellt den Termin und :227-268 den Befund als Aufgabe auf die Startseite | – |
| Abstände zwischen Belegdatum, Erfassung und Festschreibung je Buchung gespeichert und auswertbar | ✅ | internal/domain/journal.go:218-228 führt `CommittedAt` und `FestschreibungID` an der Buchung, internal/repository/journal_gorm.go:286-306 (`MarkCommitted`) stempelt beide bei der Festschreibung über eine Spaltenauswahl an jede Buchung bis zum Stichtag, ohne den Eigenhash zu berühren (internal/wailsbridge/festschreibung_service.go:99-107). Die Abstände gehen als Spalten in den Journalexport (internal/service/export_tables.go:96-107) und als Kennzahlen in jeden Prüflauf: Median, Maximum und Zahl der Buchungen über der Frist (internal/service/check_service.go:279-337, internal/domain/check.go:108-141) | – |

**Stand.** Die zehn Tage sind seit Welle 3 eine Zahl: sie stehen als `receipt_capture_days` in den Einstellungen, laufen im Prüfbericht gegen jeden liegen gebliebenen Beleg und haben in der Festschreibungsfrist des Vormonats ein Gegenstück. Mit Welle 6 kommt der dritte Abstand dazu — die Festschreibung stempelt ihren Zeitpunkt an jede Buchung, die sie erfasst, und Journalexport wie Prüflauf weisen Belegdatum, Erfassung und Festschreibung in Tagen aus, im Prüflauf als Median und Maximum über den Zeitraum. Alle drei Kriterien sind erfüllt.

### BEL-05 Journalfunktion `MUSS`

**Norm:** § 239 Abs. 2 HGB, § 146 Abs. 1 AO, GoBD Rz 94 bis 99

**Bedeutung:** Das Journal bildet die zeitliche Ordnung ab. Es ist die Grundlage der Progressiv- und Retrogradprüfung, mit der ein Prüfer vom Beleg zum Abschluss und zurück arbeitet.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Jede Buchung genau einmal, chronologisch, mit fortlaufender lückenloser Journalnummer | ✅ | internal/repository/journal_gorm.go:203-231 vergibt Nummer und Kettenkopf in einer Transaktion, `uniqueIndex` auf `EntryNumber` | – |
| Journalnummer bei der Festschreibung vergeben und danach unveränderlich | ✅ | internal/accounting/journalhash.go:32, internal/repository/journal_gorm.go:209; die Nummer entsteht bereits bei der Erfassung und geht in den Hash ein, strenger als gefordert | – |
| Journal für jeden Zeitraum als Bericht und als maschinell auswertbare Datei | ✅ | frontend/src/pages/JournalPage.tsx:285-315 grenzt das Journal auf ein Datumsfenster ein, über Jahresgrenzen hinweg, und gibt genau dieses Fenster als CSV aus; internal/service/export_service.go:190-224 schreibt dafür dieselbe Tabelle wie der Z3-Export (internal/service/export_tables.go:68-165, einschließlich der Hashwerte) und protokolliert den Lauf, die Buchungen liefert internal/repository/journal_gorm.go:100-115 | – |
| Stornierungen als eigene Journalzeilen, nicht als Löschung | ✅ | internal/service/journal_service.go:133-215; die Generalumkehr ist eine neue Buchung mit eigener Nummer | – |

**Stand.** Das Journal selbst erfüllt die Anforderung strenger als verlangt, und seit Welle 4 verlässt es auch das Programm: jeder Zeitraum geht als CSV hinaus, mit denselben Spalten wie die Datenüberlassung nach Z3 und mit den Hashwerten, an denen sich die Kette von außen nachrechnen lässt (UNV-01). Alle vier Kriterien sind erfüllt.

### BEL-06 Kontenfunktion und Kontenrahmen `MUSS`

**Norm:** § 238 HGB, GoBD Rz 96 bis 99

**Bedeutung:** Die sachliche Ordnung überführt die Journalzeilen in Konten. Der Kontenrahmen selbst ist gesetzlich nicht vorgeschrieben. Vorgeschrieben ist, dass sich aus den Konten die Gliederungen nach §§ 266 und 275 HGB und die Positionen der E-Bilanz-Taxonomie ableiten lassen.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Standardkontenrahmen als Vorlage, je Mandant erweiterbar | ✅ | internal/accounting/skr04.go, internal/accounting/skr04_2026.json liefern SKR04 mit 1.855 Konten; SKR03 entfällt nach der Entscheidung für einen Kontenrahmen (docs/entwicklung/architektur.md Abschnitt 2). Eigene Konten legt seit Welle 8 internal/service/account_service.go:84-138 (`CreateCustom`) an: die Nummer muss frei sein, und die Gliederungsposition ist Pflicht und muss in Bilanz und GuV erscheinen (:99-104) — ein Konto ohne sie erschiene weder im Abschluss noch in der E-Bilanz. :149-182 (`SetBlocked`) sperrt statt zu löschen, damit keine Buchung auf eine Nummer ohne Bezeichnung zeigt, :50-55 verwirft den zwischengespeicherten Kontenplan, damit die Änderung sofort gilt. Bridge internal/wailsbridge/welle8_service.go:148-201, Oberfläche frontend/src/pages/AccountsPage.tsx:343-450, :601-620 | – |
| Je Konto Zuordnung zu einer Position nach §§ 266, 275 HGB und zur E-Bilanz-Taxonomie | ✅ | internal/accounting/statement_mapping.go:27-234 führt alle 206 SKR04-Positionen (internal/domain/account.go:30-34) auf eine Gliederungsposition nach §§ 266, 275 HGB; internal/accounting/statement_test.go:70-73 prüft die Tabelle gegen den Katalog. Darauf baut internal/ebilanz/taxonomy.go:68-80 mit der Ressource internal/ebilanz/taxonomy_6.9.json das Taxonomie-Element auf, lückenlos geprüft von internal/ebilanz/ebilanz.go:379-394 (`StatementCoverage`); internal/accounting/statement.go:710-721 trifft für die E-Bilanz dieselbe Entscheidung wie für die Bilanz | – |
| Konten ohne Zuordnung werden vor dem Jahresabschluss gemeldet | ✅ | internal/ebilanz/mapping.go:52-107 stellt vor jedem Export jedes Konto mit Saldo gegen Gliederung und Taxonomie, :111-128 (`BlockingError`) benennt die ungeklärten einzeln und internal/ebilanz/ebilanz.go:118-124 bricht ab, bevor eine Datei entsteht; in der Bilanz stehen dieselben Konten unter „Nicht zugeordnet" (internal/accounting/statement.go:434-448, :553-572), gezeigt in frontend/src/components/StatementView.tsx:208-215 und frontend/src/pages/EBilanzPage.tsx:116-170 | – |
| Summe der Kontensalden gleich Summe der Journalbuchungen, automatische Abstimmung | ✅ | internal/service/accounting_service.go:294-341, :93-119; Salden und Summen entstehen aus denselben Verkehrszahlen, `Difference` weist jede Abweichung aus | – |

**Stand.** Der Kontenrahmen ist vollständig und richtig der HGB-Gliederung zugeordnet, und die E-Bilanz beruht seit Welle 2 auf denselben 206 Positionen; ein Konto ohne Zuordnung blockiert den Export namentlich. Mit Welle 8 kommt der Weg zu eigenen Konten dazu: eine freie Nummer, eine Gliederungsposition aus der Liste als Pflicht, ein Änderungsprotokoll und ein Kontenplan, der nach jeder Änderung neu gelesen wird. Wer ein Konto nicht mehr braucht, sperrt es; gelöscht wird keines, damit keine Buchung auf eine Nummer ohne Bezeichnung zeigt. Alle vier Kriterien sind erfüllt.

### BEL-07 Kontokorrent und offene Posten `MUSS`

**Norm:** § 238 HGB, § 240 HGB, GoBD Rz 49

**Bedeutung:** Forderungen und Verbindlichkeiten sind personenbezogen zu führen. Ohne offene Posten lassen sich Bilanzausweis, Wertberichtigung und Fälligkeitsgliederung nach § 268 Abs. 4 und 5 HGB nicht belegen.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Personenkonten rollen auf Sammelkonten auf, Saldo stimmt immer überein | ✅ | internal/service/accounting_service.go:87-119 verdichtet an einer Stelle auf 1200/3300, internal/service/journal_service.go:340-350 weist die direkte Buchung auf das Sammelkonto ab | – |
| Offene-Posten-Liste je Debitor und Kreditor zu jedem Stichtag, mit Fälligkeit und Altersstruktur | ✅ | internal/service/payment_service.go:185-198 (`OpenItemsAt`) liest die zum Stichtag offenen Buchungen und die bis dahin gebuchten Ausgleiche, beide Grenzen auf dem Buchungsdatum; die ausgestellte Abschlagsrechnung ist seit Welle 5b eine zweite Quelle derselben Liste (internal/service/advance_service.go:602-637). Die Altersstruktur kam mit Welle 6: internal/accounting/aging.go:53-101 (`AgeOpenItems`) verteilt dieselbe Stichtagsliste auf die Bänder nicht fällig, 1 bis 30, 31 bis 60, 61 bis 90 und über 90 Tage (internal/domain/aging.go:6-21), gemessen gegen den Stichtag und nicht gegen heute (:103-127); frontend/src/pages/AccountsPage.tsx:646-738 zeigt sie unter der Liste. Seit Welle 7 ist dieselbe Liste die Quelle des Zuordnungsvorschlags zum Bankumsatz (internal/service/bank_suggest.go:98-186) | – |
| Restlaufzeiten für §§ 268 Abs. 4, 5 und 285 Nr. 1 HGB auswertbar | ✅ | internal/accounting/aging.go:129-140 ordnet jeden offenen Posten nach seiner Fälligkeit einem Restlaufzeitband zu — bis ein Jahr, über eins bis fünf, über fünf (internal/domain/aging.go:26-41) —, gerechnet vom Stichtag aus (:178-196); die Bänder stehen mit Betrag und Postenzahl je Seite neben der Altersstruktur (internal/domain/aging.go:43-60, frontend/src/pages/AccountsPage.tsx:646-738) und kommen über internal/wailsbridge/nachweise_service.go:442-462 in die Oberfläche | – |
| Teilzahlungen, Skonti, Gutschriften und Ausbuchungen je Posten dokumentiert | ✅ | internal/service/payment_service.go:490-529 hält Teilzahlung, Skonto mit Steuerkorrektur nach § 17 Abs. 1 UStG (:549-624) sowie Rundungs- und Kursdifferenzen je Allokation fest. Die Ausbuchung des uneinbringlichen Postens kam mit Welle 5b dazu: internal/service/writeoff.go:39-179 verlangt einen Grund, bucht den Forderungsverlust und berichtigt die Steuer nach § 17 Abs. 2 Nr. 1 UStG (:104-127), schließt den Posten über eine eigene Differenzart (:161-168, internal/domain/payment.go:27) und protokolliert den Vorgang. Die Gutschrift an den Kunden ist die Stornorechnung oder die Berichtigung (RECH-09); sie nimmt den Posten über die Generalumkehr zurück | – |

**Stand.** Das Kontokorrent stimmt rechnerisch immer und kennt seit Welle 4 den Stichtag: die Liste zeigt, welche Posten am Bilanzstichtag offen waren, und nicht, was heute davon übrig ist. Welle 5b hat die beiden Wege ergänzt, auf denen ein Posten ohne Zahlung verschwindet — die Ausbuchung mit Grund und Steuerkorrektur und die Stornorechnung —, und die Abschlagsrechnung als zweite Quelle offener Posten hinzugefügt. Mit Welle 6 wird dieselbe Liste ausgewertet: die Altersstruktur beantwortet die Frage der Wertberichtigung, die Restlaufzeitbänder die Angaben nach §§ 268 Abs. 4 und 5 HGB unter der Bilanz. Mit Welle 7 wird sie vorgeschlagen: internal/accounting/bank_match.go:20-80 bewertet den genauen Betrag, die Rechnungsnummer im Verwendungszweck, die Namensähnlichkeit und die Datumsnähe mit Punkten und mit Gründen, internal/service/bank_suggest.go:193-246 erkennt die Sammelzahlung, deren Posten desselben Geschäftspartners zusammen den Betrag treffen, und :250-336 legt für den wiederkehrenden Umsatz ohne Posten die gelernte Buchungsgruppe vor; gebucht wird erst auf Bestätigung, und die gelernten Regeln stehen in den Einstellungen und lassen sich löschen (internal/service/bank_suggest.go:350-418). Alle vier Kriterien sind erfüllt.

### BEL-08 Ersetzendes Scannen `MUSS*`

**Norm:** GoBD Rz 130, 136 bis 141

**Bedeutung:** Papierbelege dürfen digitalisiert und anschließend vernichtet werden, wenn das Verfahren dokumentiert ist und die bildliche wie inhaltliche Übereinstimmung sichergestellt ist. Ohne Organisationsanweisung ist das Verfahren angreifbar.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Scanprotokoll mit Zeitpunkt, Person, Gerät und Ergebnis der Qualitätssicherung | ⛔ | Buchfink erklärt das ersetzende Scannen nicht zum unterstützten Verfahren. Der Herkunftswert `scan` (internal/domain/receipt.go:165) benennt nur, woher der Beleg kam; das Papier ist weiter aufzubewahren | – |
| Digitalisat bildlich vollständig und unverändert gespeichert | ⛔ | Wie oben. Unveränderlich ist die Datei ohnehin (internal/receiptstore/store.go), eine Vollständigkeitsprüfung über Seiten und Rückseiten wäre Teil des nicht unterstützten Verfahrens | – |
| Fehlerprotokoll exportierbar | ⛔ | Wie oben | – |
| Organisationsanweisung mit der Verfahrensdokumentation verknüpft | ⛔ | Wie oben; die Verfahrensdokumentation hält den Ausschluss fest (siehe PRF-03) | – |
| Nachträgliche Bildbearbeitung des Digitalisats ausgeschlossen | ✅ | internal/service/receipt_service.go:162-166, internal/domain/receipt.go:55; nach dem Buchen ist die Dateiliste versiegelt, die Inhalte sind inhaltsadressiert und werden bei jedem Lesen gegen ihre Prüfsumme gehalten | – |

**Stand.** Das `MUSS*` löst nicht aus, weil Buchfink das ersetzende Scannen nicht anbietet. Technisch wäre die Unveränderbarkeit da; was fehlt, sind Scanprotokoll und Organisationsanweisung, und deshalb bleibt der Papierbeleg aufbewahrungspflichtig. Das ist eine Entscheidung, keine Lücke.

### BEL-09 Storno und Korrektur `MUSS`

**Norm:** § 239 Abs. 3 HGB, GoBD Rz 58 bis 59

**Bedeutung:** Eine festgeschriebene Buchung wird nicht korrigiert, sondern storniert und neu gebucht. Der ursprüngliche Inhalt muss feststellbar bleiben.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Korrektur erzeugt Stornopaar plus Neubuchung, die Ursprungsbuchung bleibt im Journal | ✅ | internal/service/journal_service.go:210-301, :257-274; die Generalumkehr ist auf den Korrekturtag datiert, einen Änderungsweg gibt es überhaupt nicht | – |
| Storno und Neubuchung mit der Ursprungsbuchung verknüpft, in beide Richtungen navigierbar | ✅ | internal/domain/journal.go:154 verknüpft Storno und Ursprung über `ReversalOfID`, :230-244 die Neubuchung über `CorrectsEntryID` mit der Buchung, die sie ersetzt; internal/service/journal_correction.go:44-100 (`CorrectEntry`) setzt den Verweis, prüft die Neubuchung vor dem Storno und führt beides in einem Vorgang, :109-111 (`CorrectionOf`) liefert die Gegenrichtung. Das Journal zeigt beide Richtungen (frontend/src/pages/JournalPage.tsx:229-240, :469, :1319), bei einer Rechnung ebenso Stornodokument und berichtigte Rechnung (internal/domain/invoice.go:306-312, internal/service/invoice_correction.go:170-171) | – |
| Stornogrund ist Pflichtfeld | ✅ | internal/service/journal_service.go:232-233; ohne Grund bricht `ReverseOn` ab, der Grund geht in den Hash ein. Dasselbe verlangt das Storno einer Rechnung, bevor überhaupt eine Nummer vergeben wird (internal/service/invoice_correction.go:117-196) | – |
| Änderungen vor der Festschreibung mit Vorher- und Nachherwert, Zeitpunkt und Benutzer protokolliert | ✅ | internal/domain/audit.go:33-56 speichert je Protokolleintrag die geänderten Felder in beiden Ständen, den Zeitpunkt in UTC und die Bearbeiterkennung, internal/repository/audit_gorm.go:30-41 schreibt sie über `LogChange`. Änderungen an einer Buchung sind weiterhin überhaupt nicht möglich, insoweit ist das Kriterium strenger erfüllt; geändert werden können die Stammdaten und die Belegkopfdaten, und genau die stehen mit Vorher und Nachher im Protokoll (internal/service/receipt_service.go:305-316) | – |

**Stand.** Storno und Korrektur sind streng gebaut. Auf der Rechnungsseite schließt Welle 5b die Verkettung: Stornodokument und Ursprungsrechnung zeigen aufeinander, und die berichtigte Rechnung hat Nummer und Datum der Rechnung, an deren Stelle sie tritt. Im Journal schließt Welle 6 sie: „Stornieren und neu buchen" ist ein Vorgang, die Neubuchung vermerkt die Buchung, die sie ersetzt, und beide Richtungen stehen im Journal. Alle vier Kriterien sind erfüllt.

---
