# D. Aufbewahrung und Archivierung

[Anforderungskatalog](README.md) · [Legende](README.md#legende) · [Fachkonzepte](../fachkonzepte/README.md)

### ARC-01 Aufbewahrungsfristen `MUSS`

**Norm:** § 257 Abs. 4 HGB, § 147 Abs. 3 AO, § 14b Abs. 1 UStG, Art. 95 EGHGB, Art. 97 § 19a EGAO

**Bedeutung:** Seit dem 1. Januar 2025 gelten drei Fristen nebeneinander. Die Verkürzung für Buchungsbelege von zehn auf acht Jahre durch das Vierte Bürokratieentlastungsgesetz wirkt auch auf Fristen zurück, die am 31.12.2024 noch liefen.

| Unterlage | Frist | Norm |
|---|---|---|
| Handelsbücher, Inventare, Jahresabschlüsse, Lageberichte, Arbeitsanweisungen, Organisationsunterlagen | 10 Jahre | § 257 Abs. 4 HGB, § 147 Abs. 3 S. 1 AO |
| Buchungsbelege | 8 Jahre | § 257 Abs. 4 HGB, § 147 Abs. 3 S. 1 AO |
| Rechnungen (umsatzsteuerlich) | 8 Jahre | § 14b Abs. 1 S. 1 UStG |
| Empfangene und Kopien abgesandter Handels- und Geschäftsbriefe, sonstige Unterlagen | 6 Jahre | § 257 Abs. 4 HGB, § 147 Abs. 3 S. 1 AO |
| Aufzeichnungen zu OSS, IOSS und § 21a UStG | 10 Jahre | § 22 Abs. 1 S. 4 UStG |

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Jedes archivierte Objekt hat eine Fristenklasse und ein daraus berechnetes frühestes Löschdatum | ✅ | internal/accounting/retention.go:121-145 (`RetentionFor`) liefert zu jeder Objektart und ihrem Entstehungsjahr Klasse, Fristende und frühestes Löschdatum; internal/service/receipt_service.go:261-284 (`applyRetention`) schreibt beides beim Ablegen an den Beleg (internal/domain/receipt.go:216-225), internal/service/asset_document_service.go:145-152 an jedes Anlagendokument (internal/domain/asset_document.go:123-137). Gespeichert und nicht bei jedem Lesen gerechnet, weil die damals geltende Frist eine Tatsache über den Beleg ist. Für Journal, Festschreibungen, Abschlüsse und Meldungen führt die Fristenübersicht die Klasse je Geschäftsjahr (internal/service/retention_service.go:148-201); Belegdetail (frontend/src/pages/ReceiptsPage.tsx:904-935) und Datenüberlassung (internal/service/export_tables.go:832-833) zeigen Klasse und Fristende | – |
| Fristenklasse aus der Belegart abgeleitet, überschreibbar mit Protokollierung | ✅ | internal/domain/retention.go:76-90 (`RetentionKindOf`) leitet die Klasse aus der Belegart ab und internal/accounting/retention.go:45-74 ordnet sie ihrer Frist zu: Rechnung, Kontoauszug und Eigenbeleg acht Jahre, Handelsbrief und sonstiges Dokument sechs, Anlagendokument als Organisationsunterlage zehn. Seit Welle 8 lässt sie sich am einzelnen Beleg heraufsetzen: internal/service/self_issued_receipt.go:262-314 (`OverrideRetention`) verlangt einen Grund, vergleicht die Fristenden statt der Klassennamen — sonst nähme eine zweite Überschreibung die erste zurück — und weist jede Verkürzung ab, weil die gesetzliche Frist die Untergrenze ist; der Vorgang steht mit Vorher, Nachher und Grund im Änderungsprotokoll (:304-312). Wer die Belegart in den Kopfdaten ändert, ändert die Frist weiterhin mit, ebenfalls protokolliert (internal/service/receipt_service.go:286-317) | – |
| Löschung vor Fristablauf technisch ausgeschlossen | ✅ | internal/service/retention_service.go:295-314 (`EnsureDeletable`) weist ein Geschäftsjahr ab, dessen Frist noch läuft, und nennt den Tag, an dem sie endet; dieselbe Prüfung steht vor der Löschung (:367-388), die zusätzlich die ausgeschriebene Jahreszahl als Bestätigung und einen erstellten Archivexport verlangt. Einen anderen Löschweg gibt es nicht — `Discard` hält den Beleg mit Begründung sichtbar (internal/service/receipt_service.go:473-487) | – |
| Fristenlogik konfigurierbar ohne Codeänderung | 🟡 | Die Fristen stehen seit Welle 8 in internal/accounting/retention_rules.json, nicht mehr als Go-Literale: je Klasse Dauer, Norm und Vorgängerfrist, dazu Fassung, Quelle und der Gültigkeitsbeginn 1.1.2025, an dem das Vierte Bürokratieentlastungsgesetz die Belegfrist auf acht Jahre verkürzt hat (internal/accounting/retention_rules.go:30-68). internal/accounting/retention.go:94-145 rechnet daraus je Entstehungsjahr, sodass die Verkürzung auf laufende Fristen wirkt und abgelaufene unberührt lässt, und die Einstellungsseite zeigt die Tabelle mit ihrem Rechtsstand (internal/wailsbridge/welle8_service.go:78-83, frontend/src/pages/SettingsPage.tsx:866-880). Die Ressource ist eingebettet (internal/accounting/retention_rules.go:26): eine geänderte Frist reist mit der nächsten Auslieferung, eine Einstellung dafür gibt es nicht — dieselbe Entscheidung wie bei internal/accounting/afa_rules.json | Politur |

**Stand.** Mit Welle 6 ist die Aufbewahrung eine geführte Größe: jede Objektart hat ihre Klasse, jede Klasse ihre datierte Frist, und Beleg wie Anlagendokument haben das Fristende, das beim Ablegen galt. Welle 8 nimmt die Zahlen aus dem Code in eine Ressource mit Fassung, Quelle und Gültigkeitsbeginn und macht die Klasse am einzelnen Beleg heraufsetzbar: nur nach oben, mit Grund, mit Vorher und Nachher im Protokoll, gemessen an den Fristenden statt an den Klassennamen. Offen bleibt, dass die Ressource eingebettet ist — eine geänderte Frist reist mit der nächsten Auslieferung, so wie die Abschreibungsregeln. Politur.

### ARC-02 Fristbeginn und Ablaufhemmung `MUSS`

**Norm:** § 257 Abs. 5 HGB, § 147 Abs. 3 S. 5 AO, §§ 169, 170 AO

**Bedeutung:** Die Frist beginnt mit dem Schluss des Kalenderjahres, in dem die letzte Eintragung gemacht, das Inventar aufgestellt oder der Beleg entstanden ist. Sie endet nicht, solange die Festsetzungsfrist für die betroffene Steuer läuft. Die Regelfristen sind Mindestfristen. Eine laufende Außenprüfung oder ein Rechtsbehelf verlängert sie faktisch.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Fristbeginn auf den 31.12. des Entstehungsjahres normiert | ✅ | internal/accounting/retention.go:121-145; die Frist beginnt mit dem Schluss des Entstehungsjahres, endet am 31.12. des Jahres Entstehungsjahr zuzüglich Frist, und gelöscht werden darf ab dem 1.1. danach (§ 257 Abs. 5 HGB, § 147 Abs. 4 AO). Entstehungsjahr ist bei Buchung, Festschreibung und Abschluss das Geschäftsjahr, beim Beleg das Jahr seiner Ablage (internal/service/receipt_service.go:261-284) | – |
| Aufbewahrungs-Hold je Geschäftsjahr, Steuerart und Belegmenge | 🟡 | internal/domain/retention.go:158-203 (`RetentionHold`) setzt die Frist eines Geschäftsjahres aus, mit Grund (Außenprüfung, Rechtsbehelf, sonstiges), Beschreibung, Zeitpunkt, Bearbeiter und der Menge der betroffenen Objekte im Zeitpunkt des Setzens; internal/service/retention_service.go:220-248 zählt sie dafür, internal/service/retention_service.go:181-201 nimmt das ausgesetzte Jahr aus den abgelaufenen heraus. Nach Steuerart lässt sich die Aussetzung weiterhin nicht schneiden, sie gilt für das ganze Jahr | Politur |
| Setzen und Aufheben eines Holds mit Grund, Zeitpunkt und Benutzer protokolliert | ✅ | internal/service/retention_service.go:220-263; beide Vorgänge verlangen einen Grund, halten Zeitpunkt, Bearbeiterkennung und Programmfassung am Hold fest (internal/domain/retention.go:158-203) und gehen ins Änderungsprotokoll — das Setzen mit Vorher und Nachher (:242), das Aufheben mit seinem eigenen Grund (:251-263) | – |
| Bericht über aktive Holds und betroffene Datenmengen | ✅ | internal/service/retention_service.go:266-288 liefert die Aussetzungen, die geltenden zuerst, jede mit den Zählungen ihres Jahres (internal/domain/retention.go:205-232: Buchungen, Zeilen, Belege, Belegdateien, Festschreibungen, Prüfläufe, Meldungen, Rechnungen, Bankumsätze); internal/wailsbridge/nachweise_service.go:115-166 stellt Setzen, Aufheben, Liste und die abgelaufenen Jahrgänge bereit, frontend/src/pages/BackupPage.tsx:481-1061 zeigt sie nebeneinander | – |

**Stand.** Der Fristbeginn liegt seit Welle 6 auf dem Schluss des Entstehungsjahres, und die Ablaufhemmung ist ein eigener Vorgang: wer eine Außenprüfung oder einen Rechtsbehelf hat, setzt die Frist des betroffenen Jahres mit Grund aus, und solange sie ausgesetzt ist, führt die Übersicht das Jahr nicht als abgelaufen und die Löschung weist es ab. Setzen und Aufheben stehen mit Bearbeiter, Zeitpunkt und betroffener Datenmenge im Protokoll. Feiner als das Geschäftsjahr, nach Steuerart, lässt sich eine Aussetzung nicht schneiden. Politur.

### ARC-03 Originalformat und maschinelle Auswertbarkeit `MUSS`

**Norm:** § 147 Abs. 2 AO, GoBD Rz 131 bis 135

**Bedeutung:** Elektronisch eingegangene Unterlagen sind in dem Format aufzubewahren, in dem sie eingegangen sind. Eine Umwandlung darf die maschinelle Auswertbarkeit nicht einschränken. Wer eine strukturierte Datei in ein Bild umwandelt und nur dieses aufbewahrt, verletzt die Pflicht.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Eingehende Dateien im Originalformat, zusätzlich zu jeder erzeugten Ansicht | ✅ | internal/domain/receipt.go:22-37, :251-268 erzwingen genau eine Datei in der Rolle `original`; internal/receiptstore/store.go:110-170 schreibt die Bytes unverändert | – |
| Strukturierte Formate bleiben strukturiert erhalten | ✅ | internal/service/einvoice_service.go:90-171 legt das eingebettete XML als eigene Datei ab; internal/service/bank_service.go:64-113 (`ImportCAMT053File`) legt die CAMT.053-Datei vor dem Parsen als Beleg der Art Kontoauszug ab und hängt jeden Umsatz daran (:127), sodass sie auch dann im Archiv liegt, wenn das Parsen scheitert. Ein zweiter Import derselben Datei erzeugt über die Prüfsumme keinen zweiten Beleg (:70-84) | – |
| Ursprünglicher Dateiname, Format, Zeitpunkt, Quelle und Prüfsumme gespeichert | ✅ | internal/domain/receipt.go:78-103, :125-128 | – |
| Konvertate gekennzeichnet und mit dem Original verknüpft | ✅ | internal/domain/receipt.go:94-99, :184-189; `Derived` markiert jede abgeleitete Datei, `DisplayFile` bevorzugt das Original | – |

**Stand.** Seit Welle 4 gilt das auch für den Bankimport: der Kontoauszug ist eine aufbewahrungspflichtige Unterlage und wird zuerst abgelegt und dann gelesen, nicht umgekehrt. Alle vier Kriterien sind erfüllt.

### ARC-04 Lesbarmachung `MUSS`

**Norm:** § 239 Abs. 4 HGB, § 147 Abs. 5 AO

**Bedeutung:** Wer Unterlagen elektronisch aufbewahrt, muss sie auf Verlangen innerhalb angemessener Frist lesbar machen und auf eigene Kosten die dafür nötigen Hilfsmittel bereitstellen. Das gilt für die gesamte Aufbewahrungsdauer, auch nach einem Systemwechsel.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Jede archivierte Unterlage am Bildschirm anzeigbar und als Datei ausgebbar | ✅ | internal/wailsbridge/app_service.go:1451-1470 löst die Anzeige mit Prüfsummenwarnung; internal/wailsbridge/export_service.go:171-200 (`SaveReceiptFileAs`) gibt jede Belegdatei unter ihrem Originalnamen heraus — als Kopie, der Beleg bleibt im Archiv — und verweigert die Ausgabe, wenn die Prüfsumme nicht mehr stimmt (frontend/src/pages/ReceiptsPage.tsx:509-527) | – |
| Vollständiger Archivexport eines Geschäftsjahres in offenem Format mit Index und Feldbeschreibung | ✅ | internal/service/export_service.go:145-181 (`ExportArchive`) schreibt achtzehn Tabellen als CSV, dazu index.xml, die amtliche Grammatik und die Feldbeschreibung (internal/export/writer.go:146-163), und legt die Belegdateien unter Belegnummer und Originalnamen sowie die Anlagendokumente daneben (:700-757); jede geschriebene Datei steht mit Prüfsumme in export.json (internal/export/writer.go:234-252) | – |
| Export ohne die Anwendung lesbar, belegt durch einen Test auf fremdem System | 🟡 | Der Export ist klarschriftlich und erklärt sich selbst: CSV nach RFC 4180 mit benannten Trennzeichen (internal/export/csv.go:11-25, :27-41), index.xml gegen die mitgelieferte amtliche Grammatik (internal/export/gdpdu.go:17-52) und die Feldbeschreibung mit jeder Spalte im Klartext (internal/export/fielddoc.go:97-175) — die Verschlüsselung der Datenbank steht der Lesbarkeit damit nicht mehr im Weg. Ein Einlesen auf einem fremden System ist weiterhin nicht belegt (siehe PRF-02) | Politur |

**Stand.** Seit Welle 4 gibt Buchfink heraus, was es anzeigt: die einzelne Belegdatei unter ihrem Originalnamen, das Geschäftsjahr als Archivexport mit Index, Feldbeschreibung und Belegdateien. Derselbe Export erfüllt PRF-01, PRF-02 und UNV-01. Mit Welle 7 kommt der Prüfpfad als achtzehnte Tabelle dazu: eine Zeile je Beleg über Buchung, Zahlung und Bankumsatz. Offen bleibt der Nachweis, dass ein fremdes System die Überlassung liest; er entsteht mit dem Testeinlesen aus PRF-02 und ist Politur.

### ARC-05 Systemwechsel und Auslagerung `MUSS`

**Norm:** § 147 Abs. 6 S. 5 AO, GoBD Rz 142 bis 144

**Bedeutung:** Nach einem Systemwechsel oder einer Datenauslagerung genügt es erst nach Ablauf des fünften Kalenderjahres, das auf die Umstellung folgt, nur noch einen maschinell auswertbaren Datenträger vorzuhalten. Bis dahin ist das Altsystem vorzuhalten, wenn die Daten nicht qualitativ und quantitativ gleichwertig migriert wurden.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Migrationsprotokoll mit Quelle, Ziel, Umfang je Objektart, Zeitpunkt und Abweichungen | ✅ | internal/wailsbridge/app_service.go:973-1027 (`recordMigration`) zählt nach dem Öffnen einer übernommenen Datei die Objekte je Art und die Soll- und Habensummen (internal/repository/migrations.go:174-225), rechnet die Hash-Kette nach und legt einen `MigrationRecord` mit Quelle, Ziel, Zeitpunkt, Programmfassung und Bearbeiter ab (internal/domain/migration.go:76-108); alle drei Wege — Übernahme, Öffnen einer vorhandenen Datei und Wiederherstellung — gehen durch diese eine Stelle (:961-968), damit derselbe Vorgang nicht zweimal gezählt wird. Er steht zusätzlich im Änderungsprotokoll (:1015-1026) und in der Oberfläche (frontend/src/pages/AuditPage.tsx:1187-1358) | – |
| Abstimmung von Salden, Journalsummen und Belegzahlen vor und nach der Migration | 🟡 | internal/wailsbridge/app_service.go:985-1026 zählt Buchungen, Zeilen, Belege, Geschäftspartner, Konten, Rechnungen, Anlagegüter und Protokolleinträge, stellt Soll- und Habensumme gegeneinander und schreibt ausdrücklich ins Protokoll, wenn sie nicht übereinstimmen; die Kette wird sofort geprüft, damit eine schon beim Ankommen gebrochene Buchhaltung das sagt. Gegengeprüft wird die übernommene Datei gegen sich selbst — Zahlen aus dem Altsystem liest Buchfink nicht, die Gegenprobe bleibt die Eröffnungsbilanz gegen die Schlussbilanz des Altsystems (internal/service/opening_balance.go:104-143) | Politur |
| Migrierte Datensätze gekennzeichnet, mit Verweis auf die Herkunft im Altsystem | ✅ | internal/domain/journal.go:246-253 führt `LegacyRef` an der Buchung; die Eröffnungsbilanz des Umsteigers schreibt das Altsystem in jede Buchung und in die Personenkonten die Kennung des übernommenen Postens (internal/service/opening_balance.go:343-424), alle mit der Herkunft Eröffnung und gegen den Beleg mit der Schlussbilanz des Altsystems, der Pflicht ist (:38-52, :159-169). Die Kennung geht in die kanonische Form (internal/accounting/journalhash.go:85-89) und als eigene Spalte in die Datenüberlassung (internal/service/export_tables.go:98) | – |
| Zeitpunkt der Umstellung hinterlegt, Fünfjahresfrist berechenbar | ✅ | internal/wailsbridge/nachweise_service.go:265-288 (`SetSystemChangeDate`) hält den Umstellungszeitpunkt als Einstellung fest, internal/service/retention_service.go:204-217 rechnet daraus die Fünfjahresfrist des § 147 Abs. 6 Satz 6 AO und sagt, bis wann das Altsystem für den Datenzugriff verfügbar zu halten ist — und ab wann nicht mehr; der Satz steht bei den Aufbewahrungsfristen (frontend/src/pages/BackupPage.tsx:481-1061), gepflegt wird der Zeitpunkt in den Einstellungen (frontend/src/pages/SettingsPage.tsx:1171-1206) | – |

**Stand.** Der Umstieg von einem Altsystem ist der wahrscheinlichste Einstieg in Buchfink und seit Welle 6 der belegte Vorgang, der er sein muss: jede Übernahme zählt, was angekommen ist, rechnet die Kette nach und legt beides als Protokolleintrag ab; die Eröffnungsbilanz des Umsteigers bucht Anfangsbestände und offene Posten gegen die Schlussbilanz des Altsystems als Beleg und hat deren Kennung an jeder Buchung; der Umstellungszeitpunkt steht in den Einstellungen, die Fünfjahresfrist auf der Fristenseite. Was Buchfink nicht leisten kann, ist die Gegenprobe gegen die Zahlen des Altsystems selbst; sie bliebe auch als Politur eine Eingabe von Hand.

### ARC-06 Speicherort und Verlagerung ins Ausland `MUSS`

**Norm:** § 146 Abs. 2 bis 2c AO

**Bedeutung:** Elektronische Bücher sind grundsätzlich im Inland zu führen. Eine Verlagerung in einen EU-Mitgliedstaat ist der Finanzbehörde anzuzeigen und setzt den vollständigen Datenzugriff voraus. Eine Verlagerung in einen Drittstaat bedarf der Bewilligung. Verstöße können ein Verzögerungsgeld von 2.500 bis 250.000 Euro auslösen. Für Cloud-Betrieb ist das eine Architekturentscheidung und keine Betriebsdetailfrage.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Speicherort dokumentiert und für Anwender einsehbar | ✅ | frontend/src/pages/SettingsPage.tsx:945-959 zeigt den Datenordner schreibgeschützt an; Land und Anbieter entfallen, der Ort ist der Rechner der Anwenderin. Die Verfahrensdokumentation beschreibt ihn als eigenen Abschnitt (internal/procdoc/procdoc.go:250-264), und liegt er in einem bekannten Synchronisationsordner, steht seit Welle 6 der Hinweis auf § 146 Abs. 2 und 2a AO daneben — in den Einstellungen und beim Einrichten (internal/domain/datadir.go:14-53, internal/wailsbridge/nachweise_service.go:307-331) | – |
| Verlagerung löst einen Hinweis auf die Anzeige- oder Bewilligungspflicht aus und wird protokolliert | ⛔ | Local-First: es gibt keinen Umzugsweg, und der Knopf, der einen versprach, ist mit Welle 4 aus den Einstellungen verschwunden. An seiner Stelle steht seit Welle 6 der Hinweis auf § 146 Abs. 2 und 2a AO, sobald der Datenordner in einem OneDrive-, Dropbox-, Google-Drive-, iCloud- oder Nextcloud-Ordner liegt (internal/domain/datadir.go:14-53). Er hält nichts an, weil von hier aus nicht feststellbar ist, wo ein Synchronisationsdienst die Daten tatsächlich ablegt, und benennt zugleich, dass ein solcher Ordner kein Sicherungsziel ist | – |
| Backups und Replikate unterliegen derselben Ortsbindung, Betriebsvertrag benennt die Regionen | ⛔ | Kein Betreiber und kein Betriebsvertrag; die Ortsbindung folgt aus dem lokalen Betrieb und wird in der Verfahrensdokumentation als Inland dokumentiert | – |
| Datenzugriff nach § 147 Abs. 6 AO aus dem Inland heraus möglich | ⛔ | Die Daten liegen als Datei auf dem Rechner des Unternehmens (internal/repository/db.go:19-46); ein Auslandsbezug entsteht nicht | – |

**Stand.** Die Ortsfrage stellt sich für eine lokale Anwendung kaum, und der Speicherort ist sichtbar — in den Einstellungen und als Abschnitt der Verfahrensdokumentation. Mit Welle 6 wird die Frage auch gestellt, wo sie sich stellt: wer den Datenordner in einen Cloud-Ordner legt, liest den Hinweis auf die Bewilligungspflicht der Verlagerung. Ein Umzugsweg bleibt außerhalb des Funktionsumfangs.

### ARC-07 Aufbewahrung von E-Rechnungen `MUSS`

**Norm:** § 14b Abs. 1 UStG, § 14 Abs. 3 UStG, BMF-Schreiben vom 15.10.2025 Rn. 60

**Bedeutung:** Bei einer E-Rechnung ist zumindest der strukturierte Teil in seiner ursprünglichen Form aufzubewahren. Echtheit der Herkunft, Unversehrtheit des Inhalts und Lesbarkeit müssen über die gesamte Frist gewährleistet bleiben.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Strukturiertes XML byteidentisch, Hybridformat vollständig einschließlich eingebettetem XML | ✅ | internal/receiptstore/store.go:110-170, internal/service/einvoice_service.go:111-155; bei einem Hybrid bleibt das PDF `original`, das herausgelöste XML kommt als `structured` hinzu | – |
| Prüfsumme bei Eingang gebildet und bei jedem Abruf verifiziert | ✅ | internal/service/receipt_service.go:259-285; jeder Abrufweg rechnet SHA-256 über die gelesenen Bytes neu und liefert `Intact` | – |
| Validierungsberichte mit der Rechnung gespeichert | ✅ | internal/domain/receipt.go:134-149, internal/service/einvoice_service.go:150-172; Zeitpunkt, Regelwerk, Version, Abdeckung und Befunde als JSON | – |
| Bildansicht ergänzt das Original, ersetzt es nicht | ✅ | internal/domain/receipt.go:30-33, :266-268; die Buchung liest immer den strukturierten Teil | – |

**Stand.** Der einzige Punkt des Moduls, der vollständig erfüllt ist. Der Validierungsbericht am Beleg (internal/domain/receipt.go:134) ist zugleich der Nachweis für den Vertrauensschutz nach dem BMF-Schreiben vom 15.10.2025 Rn. 35a. Seit Welle 4 wird die Unversehrtheit nicht mehr nur beim einzelnen Abruf geprüft: der Belegprüflauf rechnet die Prüfsumme jeder Beleg- und Anlagendatei über alle Geschäftsjahre neu (internal/service/file_check.go:22-44, frontend/src/pages/AuditPage.tsx:289-366) und findet den stillen Plattenfehler, bevor der Prüfer den Beleg sehen will.

### ARC-08 Verfügbarkeit und Wiederherstellbarkeit `MUSS`

**Norm:** § 239 Abs. 4 HGB, § 147 Abs. 5 AO, GoBD Rz 103 ff., Art. 32 Abs. 1 lit. b und c DSGVO

**Bedeutung:** Aufbewahrung ohne belegte Wiederherstellbarkeit ist keine Aufbewahrung. Der Nachweis liegt beim Unternehmen, nicht beim Prüfer.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Automatisierte Sicherungen nach dokumentiertem Plan, Ergebnis je Lauf protokolliert | ✅ | internal/service/backup_service.go:147-170 schreibt Datenbank (`VACUUM INTO`), Schlüsseldatei, Belege und Dokumente als ZIP mit Prüfsumme je Datei (:173-318) und hält jeden Lauf mit Zeitpunkt, Umfang, Ergebnis und Programmfassung fest, auch den gescheiterten; internal/wailsbridge/backup_service.go:305-330 löst ihn beim Start und beim Beenden aus, sobald die letzte gelungene Sicherung älter als 24 Stunden ist (internal/service/backup_service.go:44, :134-140) | – |
| Mindestens jährlich vollständiger Wiederherstellungstest, dokumentiert | ✅ | internal/service/backup_service.go:326-402 (`VerifyBackup`) entpackt die Sicherung in einen Temporärordner, prüft die Prüfsummen aus backup.json, öffnet die Datenbank mit dem Schlüssel der Sicherung, rechnet die Hash-Chain über alle Geschäftsjahre nach, prüft die Belegdateien und räumt den Ordner wieder ab; der Lauf steht als eigene Art im Protokoll (internal/domain/backup.go:21-24) und ist über frontend/src/pages/BackupPage.tsx:385 jederzeit auszulösen | – |
| Nach der Wiederherstellung bestätigt die Integritätsprüfung den Bestand | ✅ | internal/wailsbridge/backup_service.go:222-245 ruft nach jeder Wiederherstellung zwingend Hash-Chain-Prüfung und Belegprüflauf über den wiederhergestellten Mandanten und schreibt das Ergebnis ins Änderungsprotokoll; wo nicht geprüft werden kann — verschlüsselt ohne Schlüssel, Mandant nicht offen —, sagt die Meldung das, statt Unversehrtheit zu behaupten (:191-220). Wiederhergestellt wird nur in einen leeren Ordner (internal/service/backup_service.go:443-461) | – |
| Aufbewahrungsdauer der Sicherungen deckt die Fristen ab oder das Archiv ist getrennt | 🟡 | internal/wailsbridge/backup_service.go:43-57 weist einen Sicherungsordner im Datenordner ab, das Archiv liegt also getrennt, und jeder Lauf legt eine neue Datei mit Zeitstempel an, ohne eine ältere zu löschen (internal/service/backup_service.go:205-215); die Fristen der Daten selbst führt Buchfink seit Welle 6 (ARC-01), und die Betriebsdokumentation beschreibt Zielordner, Rhythmus und die letzten Läufe (internal/procdoc/procdoc.go:442-457). Wie lange eine Sicherung aufzubewahren ist und wann eine alte weichen darf, sagt das Löschkonzept weiterhin nicht (internal/accounting/retention.go:164-238) | Politur |

**Stand.** Die Lücke mit dem größten Schadenspotenzial ist mit Welle 4 geschlossen: die Sicherung läuft von selbst, sobald die letzte einen Tag alt ist, sie liegt außerhalb des Datenordners, sie lässt sich versuchsweise zurückspielen, und nach einer echten Wiederherstellung prüft Buchfink Kette und Belegdateien, bevor jemand weiterbucht. Seit Welle 6 wird die Wiederherstellung außerdem gezählt und als Datenübernahme protokolliert (ARC-05), und jeder Lauf hat seine Bearbeiterkennung. Was fehlt, ist die Aufbewahrungsdauer der Sicherungsdateien selbst — für die Daten stehen die Fristen, für ihre Kopien sagt das Löschkonzept nichts. Politur.

---
