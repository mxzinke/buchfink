# I. Betriebsprüfung und Verfahrensdokumentation

[Anforderungskatalog](README.md) · [Legende](README.md#legende) ·
[Architektur, Prüferpaket](../entwicklung/architektur.md#65-prüferpaket)

## PRF-01 Datenzugriff Z1, Z2, Z3

MUSS · § 147 Abs. 6 AO, GoBD Rz 158 bis 177

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Nur-Lese-Zugriff auf Buchungen, Belege, Stammdaten, Auswertungen und Protokolle | erfüllt | Prüfermodus: der Kern weist jeden Schreibvorgang ab, auch von verbundenen Apps (`core.Tenant.EnableExaminer`) |
| Zugriff auf Jahre und Mandanten eingrenzbar und protokolliert | teilweise | Ein- und Ausschalten mit Datum und Grund im Änderungsprotokoll; der Modus gilt für den ganzen Mandanten; die Überlassung ist auf ein Geschäftsjahr begrenzt |
| Auswertungen frei filter-, sortier- und summierbar | teilweise | Journalfilter nach Konto, Gegenkonto, Betrag, Steuerschlüssel, Bearbeiter, Erfassungsweg, Beleg, Zeitraum und Text (`accounting.FilterJournal`); freie Summierung nur über den Export |
| Vollständiger Export eines Jahres, einlesbar in Prüfsoftware | erfüllt | `service.ExportService.ExportArchive` (Z3), `service.ExportService.ExportAuditPackage` mit Belegdateien |
| Bereitstellung über eine Datenaustauschplattform | nicht im Umfang | Buchfink überträgt nichts selbst; der Export ist ein Ordner |

## PRF-02 Format der Datenüberlassung

MUSS · § 147 Abs. 6 AO, Beschreibungsstandard für die Datenüberlassung

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Datendateien und XML-Beschreibung mit Feldnamen, Typen und Bedeutung | erfüllt | `index.xml` (Paket `export`), Feldbeschreibung aus den Tabellendefinitionen |
| CSV mit definiertem Trennzeichen und XLSX | teilweise | CSV nach RFC 4180, Festlegungen in `index.xml`; kein XLSX |
| Umfang: Journal, Konten, Salden, Partner, offene Posten, Anlagen, Steuerschlüssel, Belege, Protokoll | erfüllt | Tabellen in `service/export_tables.go` |
| Testeinlesen in eine Prüfsoftware | offen | geprüft gegen die mitgelieferte DTD; ein Einlesen in IDEA oder vergleichbare Software ist nicht belegt |

## PRF-03 Verfahrensdokumentation

MUSS · GoBD Rz 151 bis 155

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Allgemeine Beschreibung, Anwender-, technische und Betriebsdokumentation | erfüllt | `procdoc.Render` |
| Muster für die unternehmensindividuellen Teile | erfüllt | `domain.OrganisationTexts`, vorbelegt, Pflege in der App oder durch einen Agenten mit Rückfrage (`verfahrensdoku_text`); jede Änderung im Änderungsprotokoll |
| Versioniert, Fassungen über die Frist erhalten | erfüllt | `service.ProcDocService.Generate` nummeriert und legt jede Fassung im Belegspeicher ab |
| Beschreibung des internen Kontrollsystems | erfüllt | eigener Abschnitt aus dem Regelkatalog der Prüfregeln |

## PRF-04 Beweiskraft und Schnittstellen

MUSS · §§ 158, 162 AO

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Digitale Lohnschnittstelle | nicht im Umfang | Buchfink rechnet keine Löhne ab; das Lohnjournal kommt als Summenbuchung |
| DSFinV-K-Exporte | nicht im Umfang | Keine Kasse |
| Schnittstellenversion mit dem Export protokolliert | erfüllt | Fassung des Beschreibungsstandards und Programmfassung in `export.json` und `index.xml` |

## PRF-05 Zugriff bei Cloud- und Drittbetrieb

MUSS* · § 147 Abs. 6 AO, § 146 Abs. 2a und 2b AO

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Betreibervertrag, Datenzugriff, Exit-Klausel | nicht im Umfang | Buchfink läuft lokal ohne Betreiber |
| Speicherort dokumentiert | erfüllt | Abschnitt der Verfahrensdokumentation; Hinweis bei Cloud-Ordnern (`domain.CloudFolderWarning`) |

## PRF-06 Künftige Buchführungsschnittstelle

TERMIN · § 147b AO, Entwurf einer Buchführungsdatenschnittstellenverordnung

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Exportschicht trennt Datenmodell und Format | erfüllt | `export.Dataset`, `export.Table`, `export.Field` kennen kein Format |
| Datenmodell hält die Felder des Entwurfs vor | teilweise | feldreiches Journal; ein Abgleich mit dem Entwurf steht aus |
| Prüfpunkt für die Verkündung | teilweise | steht ohne Datum in der [Roadmap](../projekt/roadmap.md#prüfpunkte-mit-datum) |
