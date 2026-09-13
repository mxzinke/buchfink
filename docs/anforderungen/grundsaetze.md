# A. Buchführungspflicht und Grundsätze

[Anforderungskatalog](README.md) · [Legende](README.md#legende) · [Fachkonzepte](../fachkonzepte/README.md)

### GOB-01 Doppelte Buchführung und Bilanzierung `MUSS`

**Norm:** §§ 238, 242 HGB, § 6 HGB in Verbindung mit § 13 Abs. 3 GmbHG und § 3 Abs. 1 AktG, § 140 AO

**Bedeutung:** Kapitalgesellschaften sind Formkaufleute und damit unabhängig von Umsatz und Gewinn buchführungs- und bilanzierungspflichtig. Die Erleichterung des § 241a HGB gilt nur für Einzelkaufleute und greift hier nicht. Aus dem Handelsrecht folgt über § 140 AO die gleiche Pflicht für das Steuerrecht.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Jeder Geschäftsvorfall wird als Buchungssatz mit mindestens einer Soll- und einer Habenposition erfasst, Sollsumme gleich Habensumme | ✅ | internal/domain/journal.go:215-278, :208 (`IsBalanced`, ohne Toleranz) | – |
| Ein Buchungssatz ohne Ausgleich lässt sich nicht speichern | ✅ | internal/service/journal_service.go:96, internal/domain/journal.go:266; `Post` ist der einzige Schreibweg | – |
| Summen- und Saldenliste zu jedem Stichtag mit übereinstimmenden Summen | ✅ | internal/service/accounting_service.go:326-330 (`GetSuSaOverviewAt`) baut die Liste aus den Verkehrszahlen bis zum Stichtag (:76-120, internal/repository/journal_gorm.go:274-293 `AccountTurnoversUntil`), statt das Jahr zu summieren und danach Zeilen auszublenden; die Differenz weist :389-400 exakt aus, die Bestandskonten haben seit dem Saldenvortrag (internal/service/closing_service.go:995-1100) ihren Eröffnungswert. Der Stichtag steht als Feld an der Liste (frontend/src/pages/AccountsPage.tsx:171-177, :366) | – |
| Bilanz und GuV aus den Kontensalden abgeleitet, ohne Nacherfassung | ✅ | internal/accounting/statement.go:291-358 (`BuildStatement`) gliedert allein die Kontensalden und weist eine Bilanz ab, die nicht aufgeht (:354-356); die Eröffnungswerte kommen als gebuchter Saldenvortrag aus dem Vorjahr (internal/service/closing_service.go:1107-1175), nicht aus einer Nacherfassung | – |

**Stand.** Die Mechanik des Buchungssatzes ist streng und lückenlos abgesichert, und der Jahreswechsel führt sie fort: der Saldenvortrag bringt die Bestandskonten ins Folgejahr, die Bilanz ist ab dem zweiten Geschäftsjahr vollständig. Seit Welle 2 entsteht sie im Backend, aus denselben Salden, nicht mehr in der Ansicht. Mit Welle 4 kommt der Stichtag dazu: die Summen- und Saldenliste lässt sich auf jeden Tag innerhalb des Jahres ziehen, und die Grenze liegt in der Abfrage. Alle vier Kriterien sind erfüllt.

### GOB-02 Nachvollziehbarkeit für einen sachverständigen Dritten `MUSS`

**Norm:** § 238 Abs. 1 S. 2 HGB, § 145 Abs. 1 AO, GoBD Rz 30 ff.

**Bedeutung:** Die Buchführung muss einem sachverständigen Dritten in angemessener Zeit einen Überblick über Geschäftsvorfälle und Lage des Unternehmens geben. Das ist der Maßstab, an dem ein Betriebsprüfer die Software misst. Er kennt das Produkt nicht und muss trotzdem jeden Betrag bis zum Beleg zurückverfolgen können.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Drill-down von jedem Wert in Bilanz oder GuV in höchstens vier Schritten zum Buchungssatz und zum Beleg | ✅ | frontend/src/components/StatementView.tsx:116, :450-471 verlinkt jedes Konto unter einer Gliederungsposition ins Kontoblatt, frontend/src/pages/AccountsPage.tsx:105-113 öffnet es, :531-544 führt von der Kontoblattzeile zur Buchung, und seit Welle 7 hat die Journalzeile selbst den Weg zum Beleg (frontend/src/pages/JournalPage.tsx:489-491, :605-618): vier Klicks von der Bilanzposition bis zur Belegdatei. Der Knopf steht in der Zeile und nicht im aufgeklappten Buchungssatz, weil das Aufklappen ein fünfter wäre. Der Weg ist als Prüfszenario aufgeschrieben (docs/entwicklung/pruefszenarien.md) und wird gemessen: scripts/site-screenshots/four-clicks.mjs fährt ihn an der echten Oberfläche ab und schlägt über vier Klicks fehl (Taskfile.yml:81-88) | – |
| Drill-up vom Beleg zu den Buchungen und zu den Konten | ✅ | internal/domain/receipt.go:152, internal/repository/receipt_gorm.go:56 bilden beide Richtungen im Datenmodell ab, internal/service/audit_trail_service.go:97-184 setzt aus Beleg, Buchung, Zahlung und Bankumsatz den Prüfpfad zusammen (frontend/src/pages/ReceiptsPage.tsx:965-1163, als Ansicht, CSV und PDF). Seit Welle 8 führt der Beleg auch dorthin, wo die Buchung steht: internal/wailsbridge/welle8_service.go:89-95 (`GetEntriesForReceipt`) sucht seine Buchungen im laufenden und im vorigen Geschäftsjahr, frontend/src/pages/ReceiptsPage.tsx:1169-1250 zeigt sie unter „Buchungen zu diesem Beleg", öffnet die Buchung im Journal (:1240-1247) und jedes angesprochene Konto im Kontoblatt (:1204-1227) | – |
| Bezeichnungen im Klartext oder als exportierbares Schlüsselverzeichnis | ✅ | internal/domain/tax.go:154, internal/accounting/chart.go:49 lösen alles im Klartext auf; seit Welle 4 geht dasselbe Verzeichnis auch als Datei hinaus und liegt jeder Datenüberlassung bei (internal/service/export_tables.go:530-714, internal/service/export_service.go:229-247, siehe GOB-04) | – |
| Testlauf mit einer fachkundigen Person ohne Produktkenntnis | 🟡 | docs/entwicklung/pruefszenarien.md schreibt den Beispielmandanten, die Aufgabe und die vier erwarteten Klicks auf, scripts/site-screenshots/four-clicks.mjs fährt sie an der echten Oberfläche mit den Beispieldaten der Bridge ab und zählt sie (Taskfile.yml:81-88). Gemessen wird damit der Weg; ein Durchlauf mit einer fachkundigen Person ohne Produktkenntnis ist im Repository nicht belegt | Politur |

**Stand.** Der Weg führt von der Bilanzposition über das Konto und das Kontoblatt bis zur Belegdatei — vier Klicks, als Prüfszenario aufgeschrieben und von einem Playwright-Lauf an der laufenden Oberfläche gezählt. Welle 8 baut die Gegenrichtung zu Ende: der Beleg listet seine Buchungen, öffnet jede davon im Journal und jedes angesprochene Konto im Kontoblatt. Wer von der Ablage ausgeht — und das ist der übliche Weg eines Prüfers —, kommt damit ins Journal und weiter in die Konten. Offen bleibt der Durchlauf mit einer fachkundigen Person ohne Produktkenntnis: gezählt werden die Klicks, das Verständnis prüft erst ein Mensch. Politur.

### GOB-03 Vollständig, richtig, zeitgerecht, geordnet `MUSS`

**Norm:** § 239 Abs. 2 HGB, § 146 Abs. 1 S. 1 AO, GoBD Rz 36 ff.

**Bedeutung:** Die vier Ordnungsmerkmale sind der Kern der GoB. Die Software muss Verstöße erkennbar machen, statt sie zu verdecken.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Lücken in der Nummerierung von Belegen und Buchungen im Prüfbericht | ✅ | internal/service/check_service.go:783-819 (`checkNumberGaps`, Regel `number_gap` in internal/domain/check.go:35) vergleicht je Nummernkreis den Zähler gegen die vorhandenen Nummern und nennt jede Lücke mit Kreis und Nummer; internal/repository/numberrange_gorm.go:47-70 vergibt weiterhin in der Transaktion, internal/accounting/journalhash.go:131-145 meldet entfernte Buchungen | – |
| Nicht kontierte Belege in eigener Liste, blockieren den Periodenabschluss | ✅ | internal/service/check_service.go:619-698 (`receipt_unbooked`) führt jeden abgelegten, nicht gebuchten Beleg auf und macht ihn blockierend, sobald sein Eingang in den festzuschreibenden Zeitraum fällt; internal/wailsbridge/festschreibung_service.go:41-43, :93-103 lässt die Festschreibung nur durch, wenn der Lauf frei ist oder eine Begründung vorliegt | – |
| Plausibilitätsprüfung vor dem Periodenabschluss | ✅ | internal/service/check_service.go:185-269 rechnet vierzehn Regeln über Buchungen, Belege, Bank, Nummernkreise, Voranmeldung, Belegnachweis, Leistungsnachweis und Festschreibungsstand (internal/domain/check.go:27-69), im Jahreslauf vier weitere; :359-371 (`EnsureCommittable`) entscheidet daraus über die Festschreibung. Seit Welle 7 hat der Bericht seinen Platz im Ablauf: er ist der erste Schritt des Monatsabschlusses und steht vor dem Festschreiben (internal/service/month_close_service.go:132-228, frontend/src/components/MonthCloseDialog.tsx:258-345), aufgerufen von der Aufgabenliste und von der Fristenseite (frontend/src/pages/TasksPage.tsx:466-480, frontend/src/pages/DeadlinesPage.tsx:715-723) | – |
| Reproduzierbare Sortierung von Journal und Kontenblatt | ✅ | internal/repository/journal_gorm.go:32, :70-78; jede Leseabfrage ordnet nach `id asc`, Anzeige als stabile Sortierung | – |

**Stand.** Seit Welle 3 steht die Kontrolle neben der Ordnung: vor jeder Festschreibung läuft ein Prüfbericht aus vierzehn Regeln, blockierende Befunde halten sie auf, und übergehen lässt sich nur mit einer Begründung, die am Lauf und im Protokoll steht. Welle 7 gibt dem Bericht seinen Platz im Monat — der Monatsabschluss beginnt mit ihm, schreibt danach fest und bestätigt zuletzt die Voranmeldung —, und dieselben Befunde stehen auf der Aufgabenliste, jeder mit dem Sprung an die Stelle, an der er zu klären ist. Alle vier Kriterien sind erfüllt.

### GOB-04 Sprache, Abkürzungen, Währung `MUSS`

**Norm:** § 239 Abs. 1 HGB, § 244 HGB

**Bedeutung:** Handelsbücher sind in einer lebenden Sprache zu führen. Wer Abkürzungen, Ziffern oder Symbole verwendet, muss deren Bedeutung eindeutig festlegen. Der Jahresabschluss ist zwingend in deutscher Sprache und in Euro aufzustellen.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Schlüsselverzeichnis aller Codes als Datei exportierbar, Teil der Verfahrensdokumentation | ✅ | internal/service/export_tables.go:530-714 (`keyDirectoryTable`) führt jeden Code aus Quelle, Buchungsart, Steuerfall, Belegart, Belegstatus, Prüfregeln und Steuerschlüssel mit Bedeutung; internal/service/export_service.go:229-247 schreibt ihn als CSV an einen gewählten Ort (Bridge: internal/wailsbridge/export_service.go:78-90), er liegt jeder Datenüberlassung als `schluesselverzeichnis.csv` bei und steht im Klartext in der Feldbeschreibung (internal/export/fielddoc.go:12, :165-174). Die Oberfläche zeigt dieselbe Tabelle (frontend/src/pages/TaxAuditPage.tsx:170-226, hinter „Schlüsselverzeichnis" bei der Datenüberlassung) | – |
| Bilanz, GuV und Anhang in deutscher Sprache und in Euro | ✅ | `internal/service/statement_service.go`, `notesFor`, stellt Freitexte, Rückstellungsspiegel und Überleitung zusammen. `internal/service/statement_export.go`, `ExportCSV`, `statementTypst` und `writeTypstNotes`, geben die Daten mit deutschen Bezeichnungen und Euro-Beträgen aus. Fehlende Quellen und der noch nicht angepasste Anhangumfang bleiben Einschränkungen unter JAB-03. | – |
| Buchungswährung des Hauptbuchs ist Euro, Fremdwährung zusätzlich | ✅ | internal/domain/journal.go:197 führt die Währung am Buchungskopf, :112-125 seit Welle 5c den Betrag der Zeile in dieser Währung (`ForeignAmount`), :154-157 Kurs, Kursquelle und Kursdatum. Gebucht wird in Euro, die Fremdwährung steht daneben: die Umrechnung lässt sich nicht zurückrechnen, und wer später fragt, worauf die Rechnung des Lieferanten lautete, bekommt die Zahl, die auf ihr stand. internal/accounting/journalhash.go:136-137 nimmt den Betrag in die kanonische Form auf, wo er belegt ist — eine Buchung in Euro hat das Feld nicht, und die Ketten bestehender Buchhaltungen bleiben unverändert | – |

**Stand.** Bilanz und Gewinn- und Verlustrechnung verlassen das Programm seit Welle 2 als PDF und als CSV, deutsch und in Euro. Das Schlüsselverzeichnis geht seit Welle 4 als Datei hinaus und liegt jeder Datenüberlassung bei — die Bedeutung jeder Abkürzung steht damit dort, wo die Abkürzung ankommt. Die Fremdwährung führt die Buchungszeile seit Welle 5c mit ihrem eigenen Betrag neben dem Eurowert, und dieser Betrag geht in die Hash-Kanonisierung ein, wo er belegt ist. Der Anhang fehlt weiterhin. Welle 2.

### GOB-05 Einzelaufzeichnung `MUSS`

**Norm:** § 146 Abs. 1 S. 1 AO, GoBD Rz 39 ff.

**Bedeutung:** Jeder Geschäftsvorfall wird einzeln aufgezeichnet. Verdichtete Sammelbuchungen sind nur zulässig, wenn die Einzelpositionen jederzeit aus dem System heraus nachweisbar bleiben.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Sammelbuchungen verweisen auf eine gespeicherte Einzelpostenliste mit gleicher Aufbewahrungsfrist | ✅ | internal/domain/payment.go:48-65; die Allokationen der Sammelzahlung stehen in der Datenbank und gehen als eigene Tabelle in die Überlassung (internal/service/export_tables.go:229-260). Ihre Frist ist die der Buchung, zu der sie gehören: das Löschkonzept führt Journal und Hauptbuch mit zehn Jahren (internal/accounting/retention.go:168-175), und aus der Datenbank verschwinden sie erst mit dem abgelaufenen Geschäftsjahr (internal/repository/welle6_gorm.go:299-303, internal/service/retention_service.go:367-429). Kassenbuch, Lohnjournal und Stapelerfassung sind außerhalb des Funktionsumfangs | – |
| Einzelpostenliste mit einem Klick erreichbar und maschinell auswertbar exportierbar | ✅ | frontend/src/pages/JournalPage.tsx:773-855 klappt die Einzelposten einer Sammelzahlung an der Buchung auf, mit Belegverweis und Betrag je Posten (Bridge: internal/wailsbridge/export_service.go:265-273, internal/service/payment_service.go:138-173); dieselben Zeilen gehen als Tabelle `zahlungszuordnungen` in die Datenüberlassung (internal/service/export_tables.go:229-260) | – |
| Je Geschäftsvorfall mindestens Datum, Betrag, Steuerbetrag und -satz, Leistungsgegenstand, Partner, Belegverweis | ✅ | internal/domain/journal.go:110-145, :79-91 erfassen alles auf dem Belegweg; seit Welle 8 gilt dasselbe für die Handbuchung: internal/service/manual_entry.go:38-117 (`PostManualEntry`) verlangt den Beleg oder den Eigenbeleg und setzt den Verweis in derselben Transaktion, :174-231 (`ValidateManualTaxLines`) verlangt den Steuerfall am Kopf und je Steuerzeile Steuerschlüssel und Bemessungsgrundlage. Die beiden Ausnahmen haben ihren Grund: die Generalumkehr übernimmt die Zeilen der Ursprungsbuchung (:179-181), und der Ausschluss nach § 15 Abs. 1a UStG erklärt die fehlende Steuerzeile (:190-195, :224-231) | – |

**Stand.** Der Belegweg zeichnet vollständig einzeln auf, und mit Welle 8 zeichnet der Handweg genauso auf: die Buchung verlangt ihren Beleg, den Steuerfall am Kopf und je Steuerzeile Steuerschlüssel und Bemessungsgrundlage. Die einzige Sammelbuchung, die Buchfink erzeugt, gibt ihre Einzelposten seit Welle 4 auch heraus — an der Buchung aufklappbar und als eigene Tabelle der Überlassung —, und mit Welle 6 gilt für diese Tabelle eine Frist: die Zuordnungen teilen die zehn Jahre der Buchung, zu der sie gehören. Alle drei Kriterien sind erfüllt.

### GOB-06 Geschäftsjahr und Periodenabgrenzung `MUSS`

**Norm:** § 240 Abs. 2 S. 2 HGB, § 252 Abs. 1 Nr. 5 HGB, § 4a EStG

**Bedeutung:** Das Geschäftsjahr darf zwölf Monate nicht überschreiten. Aufwendungen und Erträge gehören in die Periode ihrer wirtschaftlichen Verursachung, unabhängig vom Zahlungszeitpunkt.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Geschäftsjahr frei definierbar, höchstens zwölf Monate, Rumpfgeschäftsjahre möglich | ✅ | internal/domain/fiscalyear.go:81-120 führt Beginn und Ende je Jahr, :161-197 weist mehr als zwölf Monate nach § 240 Abs. 2 Satz 2 HGB ab, :124-130 kennzeichnet das Rumpfjahr; internal/service/closing_service.go:359-376 (`derive`) setzt den Beginn des Gründungsjahres auf die Beurkundung, und `AlignFoundingYear` (:1623-1729) zieht ihn nach, wenn die Gründung erst nach dem Anlegen des Jahres erfasst wird — der Weg des Einrichtungsassistenten. Angeglichen wird nur, solange der Abschluss nicht festgestellt, kein Zeitraum festgeschrieben und keine Buchung vor dem Beurkundungstag ist | – |
| Buchungsdatum getrennt vom Belegdatum, Periodenzuordnung nach Buchungsdatum | ✅ | internal/domain/journal.go:110-114, internal/service/journal_service.go:257; vier getrennte Datumsfelder | – |
| Buchung in ein abgeschlossenes Jahr nur nach dokumentierter Wiedereröffnung, protokolliert | ✅ | internal/service/journal_service.go:489-505 (`ensureYearNotAdopted`) weist jede Buchung in ein festgestelltes Jahr ab und nennt den Weg zurück; internal/service/closing_service.go:443-474 (`ReopenFiscalYear`) verlangt einen Grund und schreibt ihn ins Protokoll. Berechtigte Rollen entfallen im Einzelplatzbetrieb | – |
| Abgrenzungsbuchungen mit automatischer Auflösung im Folgejahr | ✅ | internal/domain/accrual.go:151-184 führt den Abgrenzungsposten mit Zeitraum, Verfahren und Auflösungsplan, internal/accounting/accrual.go:157-218 (`AccrualReleasePlanFor`) legt den Plan bei der Bildung an, internal/service/accrual_service.go:374-439 (`Book`) bucht sie auf 1900 bzw. 3900; die Auflösung stößt der Saldenvortrag an (internal/service/closing_service.go:1157-1165, internal/service/accrual_service.go:561-625 `ReleaseInto`) und die Vortragsvorschau kündigt sie vorher an (internal/service/closing_service.go:612-616, :646-652) | – |

**Stand.** Das Geschäftsjahr ist eine eigene Entität mit Zeitraum, Rumpfjahrkennzeichen und Abschlussstand; die Periodenzuordnung folgt weiterhin dem Buchungsdatum, und die Wiedereröffnung ist ein dokumentierter Vorgang statt einer Lücke. Mit Welle 5 kommt die Periodenabgrenzung dazu: der Abgrenzungsposten hat seinen Zeitraum, sein Verteilungsverfahren und seinen Auflösungsplan, und die Auflösung bucht der Saldenvortrag von selbst ins Folgejahr. Alle vier Kriterien sind erfüllt.

---
