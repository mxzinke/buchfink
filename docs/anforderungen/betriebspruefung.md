# I. Betriebsprüfung und Verfahrensdokumentation

[Anforderungskatalog](README.md) · [Legende](README.md#legende) · [Fachkonzepte](../fachkonzepte/README.md)

### PRF-01 Datenzugriff Z1, Z2, Z3 `MUSS`

**Norm:** § 147 Abs. 6 AO, GoBD Rz 158 bis 177

**Bedeutung:** Die Finanzbehörde hat drei gleichrangige Zugriffsarten zur Wahl. Sie entscheidet, welche sie nutzt, und kann alle drei kombinieren. Die Software muss alle drei bedienen können.

| Art | Bezeichnung | Was die Software leisten muss |
|---|---|---|
| Z1 | Unmittelbarer Zugriff | Nur-Lese-Zugang für den Prüfer am System, mit den Auswertungsmöglichkeiten, die dem Unternehmen zur Verfügung stehen |
| Z2 | Mittelbarer Zugriff | Auswertung durch das Unternehmen nach Vorgaben des Prüfers |
| Z3 | Datenüberlassung | Übertragung der Daten in maschinell auswertbarem Format, seit dem Änderungsschreiben vom 11.03.2024 auch über eine Datenaustauschplattform nach § 87a Abs. 1 AO |

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Prüferprofil mit Nur-Lese-Zugriff auf Buchungen, Belege, Stammdaten, Auswertungen und Änderungsprotokolle | ✅ | Ein Modus, kein Rollenmodell — im Einzelplatzbetrieb ohne Benutzerkonten ist das die Form, in der Z1 überhaupt herstellbar ist (docs/entwicklung/architektur.md Abschnitt 2): internal/wailsbridge/readonly.go:37-189 lässt Lesen, Auswerten, Prüfen und Ausgeben zu, :210-220 weist alles Schreibende an der Bridge ab und nicht erst in der Oberfläche. Der Modus gilt befristet und mit protokolliertem Grund (:227-260), die Oberfläche nennt beides über jeder Ansicht (frontend/src/pages/TaxAuditPage.tsx:369-437) | – |
| Zugriff auf Geschäftsjahre und Mandanten eingrenzbar und protokolliert | 🟡 | Ein- und Ausschalten des Prüfermodus stehen mit Datum und Grund im Änderungsprotokoll (internal/wailsbridge/readonly.go:255-258, :285-288), und die Überlassung ist auf ein Geschäftsjahr begrenzt und wird mit Umfang protokolliert (internal/service/export_service.go:902-912). Der Zugriff selbst bleibt unbegrenzt: internal/wailsbridge/app_service.go:727 (`SwitchTenant`) und :1373 (`SetFiscalYear`) sind auch im Prüfermodus offen und schreiben nichts ins Protokoll; Welle 7 hat daran nichts geändert | Politur |
| Auswertungen nach freien Kriterien filter-, sortier- und summierbar | 🟡 | Seit Welle 8 filtert das Journal nach Konto, Gegenkonto, Betrag von/bis, Steuerschlüssel, Bearbeiter, Beleg vorhanden ja/nein, Zeitraum und Volltext: internal/accounting/journal_filter.go:30-52 führt den Filter — Zeiger bei Betrag und Belegkennzeichen, damit „nicht gesetzt" und „null" bzw. „nein" auseinandergehen —, :95-146 (`FilterJournal`) wendet ihn zeilenweise an und rechnet Soll, Haben, Saldo, Zeilen- und Buchungszahl der gefilterten Menge (:77-94). Die Ausgabe geht als CSV derselben Menge hinaus und wird als Zugriff protokolliert (internal/service/journal_filter_csv.go:24-96, internal/wailsbridge/welle8_service.go:221-275), die Oberfläche zeigt Filter und Summenzeile (frontend/src/components/JournalFilterView.tsx:270-370). Die Sortierung bleibt die reproduzierbare Journalordnung nach Datum, Buchungsnummer und Position (internal/accounting/journal_filter.go:132-140); eine freie Sortierwahl gibt es nicht — sie geschieht in der Prüfsoftware am Z3-Export | Politur |
| Vollständiger Datenexport eines Geschäftsjahres ohne Nachbearbeitung, einlesbar in Prüfsoftware | ✅ | internal/service/export_service.go:107-181; ein Aufruf schreibt Tabellen, index.xml, Grammatik, Feldbeschreibung und Metadatei in den gewählten Ordner (internal/export/writer.go:146-163), das Prüferpaket zusätzlich Belegdateien, Integritätsnachweis und Verfahrensdokumentation (:700-757, :761-808, :877-888). Nachzubearbeiten ist nichts; das Testeinlesen in eine Prüfsoftware steht aus (PRF-02) | – |
| Bereitstellung über eine Datenaustauschplattform möglich | ⛔ | Buchfink überträgt nichts selbst: Local-First, kein Fernzugriff, keine Anbindung an einen Dienst (docs/entwicklung/architektur.md Abschnitt 2, ebenso PRF-05). Was der Plattformweg braucht, ist eine Datei, und die entsteht seit Welle 4 — die Überlassung liegt als Ordner mit CSV, index.xml und Prüfsummen je Datei (internal/export/writer.go:234-252) und lässt sich unverändert hochladen oder auf einem Datenträger übergeben | – |

**Stand.** Mit Welle 4 sind alle drei Zugriffsarten bedienbar: Z1 als Prüfermodus mit befristetem, protokolliertem Nur-Lese-Zugang, Z2 über die Auswertungen des Programms, Z3 als vollständige Datenüberlassung nach dem Beschreibungsstandard. Welle 8 rüstet die zweite Zugriffsart aus: das Journal filtert nach Konto, Gegenkonto, Betragsbereich, Steuerschlüssel, Bearbeiter, Belegkennzeichen und Zeitraum, zeigt zur gefilterten Menge ihre Summenzeile und gibt genau diese Menge als CSV heraus, protokolliert als Zugriff. Zwei Ränder bleiben: die Sortierung ist die reproduzierbare Journalordnung und keine freie Wahl, und der Wechsel zwischen Geschäftsjahren und Mandanten bleibt auch im Prüfermodus offen und unprotokolliert. Politur.

### PRF-02 Format der Datenüberlassung `MUSS`

**Norm:** § 147 Abs. 6 AO, Beschreibungsstandard für die Datenüberlassung (Anlage zum GoBD-Änderungsschreiben vom 11.03.2024)

**Bedeutung:** Für die Finanzbuchhaltung existiert derzeit keine verbindliche gesetzliche Schnittstelle. Faktischer Standard ist der Beschreibungsstandard für die Datenüberlassung mit einer XML-Strukturbeschreibung und den zugehörigen Datendateien. Für Besteuerungszeiträume ab dem 01.01.2025 werden EBCDIC, Lotus 123, ASCII-Druckdateien und die AS400-Konvertierung nicht mehr unterstützt.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Export erzeugt Datendateien plus eine XML-Strukturbeschreibung mit Feldnamen, Typen, Längen und fachlicher Bedeutung | ✅ | internal/export/gdpdu.go:52-79 schreibt index.xml nach dem Beschreibungsstandard: je Tabelle Dateiname, Bezeichnung, Zeitraum und Formatfestlegungen, je Spalte Name, Erläuterung und Typ (alphanumerisch, numerisch mit Nachkommastellen, Datum mit Format). Die amtliche Grammatik liegt eingebettet bei und wird mitgeschrieben (:17-44, internal/export/gdpdu-01-09-2004.dtd), die fachliche Bedeutung jeder Spalte steht zusätzlich im Klartext in feldbeschreibung.md (internal/export/fielddoc.go:97-175, gespeist aus der Erläuterung, die jedes Feld in internal/service/export_tables.go mitführt) | – |
| Zielformate mindestens CSV oder ASCII mit definiertem Trennzeichen und XLSX, keine aufgegebenen Formate | 🟡 | internal/export/csv.go:11-25, :27-41 schreibt UTF-8 ohne BOM nach RFC 4180: Semikolon als Trennzeichen, CR LF, doppeltes Anführungszeichen, Punkt als Dezimaltrennzeichen — dieselben Festlegungen stehen maschinenlesbar in index.xml und im Klartext in der Feldbeschreibung (internal/export/fielddoc.go:112-126). Aufgegebene Formate kommen nicht vor. XLSX ist bewusst weggelassen, und der Grund steht in der Überlassung selbst (internal/export/fielddoc.go:141-145): es wäre eine zweite Fassung derselben Daten in einem Format, das Beträge und führende Nullen beim Öffnen verändert | – |
| Umfang mindestens Journal, Kontenbeschreibung, Kontensalden, Debitoren- und Kreditorenstammdaten, offene Posten, Anlagenstammdaten und -bewegungen, Steuerschlüsselverzeichnis, Änderungsprotokoll, Belegverzeichnis | ✅ | internal/service/export_tables.go:21-40 nennt die achtzehn Tabellen, gebaut in internal/service/export_service.go:697-720: journal (:68-165), konten mit Gliederungs- und Taxonomieposition (:245-295), salden (:314-348), kontakte (:349-379), offene_posten (:380-415), anlagen (:416-460) und anlagen_bewegungen (:461-500), steuerschluessel (:501-529), schluesselverzeichnis (:530-714), aenderungsprotokoll (:716-737), belege (:738-790) — dazu dokumente, voranmeldungen, festschreibungen, pruefläufe, zahlungszuordnungen, bewirtungen und seit Welle 7 pruefpfad (:1030-1102) | – |
| Testeinlesen in eine Prüfsoftware belegt die Verwendbarkeit | ❌ | Der Export ist gegen die mitgelieferte amtliche Grammatik gebaut und in internal/export/export_test.go sowie internal/service/export_service_test.go geprüft; ein Einlesen in IDEA oder ACL hat weiterhin nicht stattgefunden und lässt sich im Code auch nicht belegen | Politur |

**Stand.** Mit Welle 4 gebaut, und der Hebel hat sich gezeigt: derselbe Export erfüllt ARC-04, UNV-01, JAB-08, QUE-06 und liefert später die Grundlage für den DATEV-Export. Mit Welle 7 kommt der Prüfpfad als achtzehnte Tabelle hinzu. Zwei Punkte bleiben, und keiner davon ist Programmierarbeit: XLSX wird bewusst nicht angeboten, und ob eine Prüfsoftware die Überlassung tatsächlich einliest, ist erst belegt, wenn es jemand versucht hat. Der zweite ist Politur.

### PRF-03 Verfahrensdokumentation `MUSS`

**Norm:** GoBD Rz 151 bis 155

**Bedeutung:** Für jedes eingesetzte DV-System ist eine Verfahrensdokumentation zu führen. Sie besteht aus einer allgemeinen Beschreibung, einer Anwenderdokumentation, einer technischen Systemdokumentation und einer Betriebsdokumentation. Eine fehlende Verfahrensdokumentation ist nach Rz 155 nur dann ein formeller Mangel, wenn die Nachvollziehbarkeit tatsächlich beeinträchtigt ist. Auf diese Einschränkung sollte man sich nicht verlassen.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Herstellerdokumentation mit allen vier Bestandteilen und dem Zusammenhang zu den gesetzlichen Anforderungen | ✅ | internal/procdoc/procdoc.go:198-204 legt die Abschnitte fest, :206-520 setzt sie: allgemeine Beschreibung mit Unternehmen, Geltungsbereich, Speicherort und Steuerfällen (:223-278), Anwenderdokumentation mit Belegfluss, Ausgangsrechnung, Bankimport, Abschluss, Eigenbelegen, Nummernkreisen und Korrekturen (:279-366), technische Systemdokumentation mit Datenmodell, Kanonisierung, Beleg-Hash, Verschlüsselung, Zeitstempel, Festschreibung, Exporten und Fristen (:367-441), Betriebsdokumentation mit Sicherung, Wiederherstellung, Integritätsprüfung, Änderungshistorie und Migrationsprotokoll (:442-491). Jeder Abschnitt nennt die Norm, auf der er beruht; die veränderlichen Angaben kommen aus der Datenbank und aus dem Code, der tatsächlich rechnet (internal/service/procdoc_service.go:266-338) | – |
| Muster für die unternehmensindividuellen Teile | ✅ | internal/domain/procdoc.go:71-86 führt Zuständigkeiten, Belegfluss im Haus, Scannen, Freigabe und Festschreibung, Vertretung, Sicherung und Weiteres als Freitexte, :104-132 gibt die Muster vor, mit denen sie vorbelegt sind; internal/service/procdoc_service.go:90-153 liest und speichert sie mit Vorher und Nachher im Protokoll, gepflegt werden sie unter „Betriebsprüfung", wo die Fassung entsteht, die sie aufnimmt (frontend/src/pages/TaxAuditPage.tsx:627-720). Das Formular gruppiert die sieben Freitexte nach den drei Fragen, die an die Organisation gestellt werden — wer arbeitet damit, wie kommen die Belege herein, wie sind die Daten gesichert —, und sagt an jedem Abschnitt, ob er beschrieben ist oder noch im Muster steht: dafür liefert internal/wailsbridge/nachweise_service.go:324-336 (`GetOrganisationTextDefaults`) die Muster getrennt, weil `GetOrganisationTexts` beides ununterscheidbar mischt. Ein unverändertes Muster ist kein Fehler — für ein Ein-Personen-Unternehmen ist es oft die Wahrheit —, aber es soll niemandem als eigene Beschreibung durchgehen und sind aus den Einstellungen verlinkt (frontend/src/pages/SettingsPage.tsx:1189-1204). In der erzeugten Fassung stehen sie als Abschnitt 2.9 (internal/procdoc/procdoc.go:349-366) | – |
| Versioniert, Historie über die gesamte Aufbewahrungsfrist nachvollziehbar | ✅ | internal/service/procdoc_service.go:154-223 (`Generate`) nummeriert jede Fassung (:339-347) und legt sie im Belegspeicher unter `dokumente/verfahrensdokumentation/` ab — dort, wo Sicherung, Archivexport und Prüferpaket sie mitnehmen —, mit Fassung, Zeitpunkt, Bearbeiter, Programmfassung, Regelstand, Dateiname und Prüfsumme (internal/domain/procdoc.go:15-52); das Löschkonzept führt die Fassungen zehn Jahre (internal/accounting/retention.go:191-198), und das Prüferpaket legt alle bei und nicht nur die jüngste, weil zu jedem Geschäftsjahr die damals geltende gehört (internal/service/export_service.go:983-1030). Einzeln herausgeben lässt sich jede Fassung seit Welle 9 (internal/service/procdoc_service.go:276-309 `File`, internal/wailsbridge/nachweise_service.go:250-297 `SaveProcedureDocumentationAs`): die Prüfsumme wird vor dem Schreiben verglichen, damit keine Fassung hinausgeht, die nicht mehr die ist, deren Erzeugung im Protokoll steht, und die Herausgabe selbst wird als Zugriff festgehalten | – |
| Beschreibung des internen Kontrollsystems enthalten | ✅ | internal/procdoc/procdoc.go:203, :492-518 setzt das interne Kontrollsystem als eigenen Abschnitt: die harten Regeln des Buchungskerns und die Prüfregeln vor der Festschreibung mit Regel, Gegenstand und Wirkung, gespeist aus dem Regelkatalog des Prüfdienstes (internal/service/procdoc_service.go:436-457) | – |

**Stand.** Seit Welle 6 erzeugt Buchfink die Verfahrensdokumentation aus dem laufenden System: Unternehmensdaten, Kontenrahmen, Nummernkreise mit ihrer Systematik, Regelstand, Programmfassung, Fristen und Prüfregeln kommen aus der Datenbank und aus dem Code, die Textbausteine aus dem Programm, die unternehmensindividuellen Teile aus Freitextfeldern mit Muster. Jede Fassung wird nummeriert, im Belegspeicher abgelegt und liegt jedem Prüferpaket bei — alle Fassungen, weil zu jedem Geschäftsjahr die damals geltende gehört. Sie enthält zugleich die Scope-Entscheidungen dieses Katalogs: Einzelplatzbetrieb, Speicherort Inland, kein ersetzendes Scannen, geschlossene Liste der Steuerfälle. Alle vier Kriterien sind erfüllt.

### PRF-04 Beweiskraft und Schnittstellenkonformität `MUSS`

**Norm:** § 158 AO, § 162 AO

**Bedeutung:** § 158 Abs. 2 AO entzieht der Buchführung die Beweiskraft, wenn die Daten nicht nach den Vorgaben der einheitlichen digitalen Schnittstellen bereitgestellt werden. Die Folge ist die Schätzung nach § 162 AO. Für die Finanzbuchhaltung greift diese Variante derzeit noch nicht, weil die Verordnung nach § 147b AO fehlt. Für die Digitale Lohnschnittstelle und die DSFinV-K greift sie bereits.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Digitale Lohnschnittstelle in der jeweils aktuellen Version bedienbar, wenn Lohndaten verarbeitet werden | ⛔ | Buchfink verarbeitet keine Lohndaten; der Lohn kommt als Sammelbuchung aus dem Lohnjournal des Lohnbüros herein, die Schnittstelle trifft den Lohnabrechner | – |
| DSFinV-K-Exporte einlesbar, Sammelbuchung auf die Einzelvorgänge zurückführbar | ⛔ | Kein Kassensystem und kein Aufzeichnungssystem nach § 146a AO | – |
| Schnittstellenversionen konfigurierbar und mit dem Export protokolliert | ✅ | internal/export/gdpdu.go:33 führt die Fassung des Beschreibungsstandards, internal/export/writer.go:61-70 schreibt sie zusammen mit der Programmfassung in export.json, internal/export/gdpdu.go:57, :66-68 nennt beide in index.xml, und internal/service/export_service.go:902-912 hält Art, Umfang und Fassung jedes Exports im Änderungsprotokoll fest | – |

**Stand.** Die beiden Sachverhalte, an die § 158 Abs. 2 AO heute anknüpft, liegen außerhalb des Funktionsumfangs. Seit Welle 4 nennt jede Datenüberlassung ihre Fassung: Beschreibungsstandard und Programmversion stehen in index.xml, in export.json und im Änderungsprotokoll. Wählbar ist die Fassung nicht, und die Taxonomieversion der E-Bilanz bleibt hartcodiert — das steht in JAB-05.

### PRF-05 Zugriff bei Cloud- und Drittbetrieb `MUSS*`

**Norm:** § 147 Abs. 6 AO in der Fassung des DAC7-Umsetzungsgesetzes, § 146 Abs. 2a und 2b AO

**Bedeutung:** Seit 2023 trifft die Pflicht, Einsicht, Auswertung und Übertragung zu ermöglichen, ausdrücklich auch Dritte, bei denen die Daten liegen, also Rechenzentren und Cloud-Anbieter. Der Betriebsvertrag muss das abbilden.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Vertrag mit dem Betreiber verpflichtet diesen zur Mitwirkung nach § 147 Abs. 6 AO | ⛔ | Local-First: es gibt keinen Betreiber, die Daten liegen im Datenordner auf dem Rechner der Anwenderin (internal/wailsbridge/app_service.go:426-435) | – |
| Datenzugriff unabhängig vom Speicherort aus dem Inland | ⛔ | Lokale Datei, kein Fernzugriff | – |
| Exit-Klausel und hinterlegter Vollexport bei Kündigung oder Insolvenz des Anbieters | ⛔ | Kein Anbieter, der ausfallen könnte. Das reale Gegenstück ist der fehlende Sicherungsweg, und der steht in ARC-08 | – |
| Speicherort dokumentiert | ⛔ | internal/domain/app_config.go:6-10 führt den Pfad je Mandant, frontend/src/pages/SettingsPage.tsx:945-959 zeigt ihn an. Die ausgewiesene Dokumentation für den Prüfer, die auch Belegordner und Schlüsselort nennt, steht seit Welle 6 als Abschnitt 1.3 der erzeugten Verfahrensdokumentation (internal/procdoc/procdoc.go:250-264) | – |

**Stand.** Der Sachverhalt tritt bei einer lokalen Einzelplatzanwendung nicht ein. Offen bleibt allein die Dokumentation des Speicherorts, und die gehört in die Verfahrensdokumentation.

### PRF-06 Künftige Buchführungsschnittstelle `TERMIN`

**Norm:** § 147b AO, Entwurf einer Buchführungsdatenschnittstellenverordnung (DSFinVBV)

**Bedeutung:** § 147b AO ermächtigt seit 2023 zu einer Rechtsverordnung über eine einheitliche digitale Schnittstelle für die Buchführung. Die Verordnung ist noch nicht erlassen; zu dem Entwurf, der im Amtsdeutsch Buchführungsdatenschnittstellenverordnung (DSFinVBV) heißt, liegt 2026 ein weiterer Diskussionsentwurf vor, zu dem die Bundessteuerberaterkammer am 09.03.2026 Stellung genommen hat. Als Format sieht der Entwurf xBRL-CSV in der Version 1.0 vor, also CSV-Dateien plus Metadaten als JSON. Nach dem Entwurf gälte die Verordnung erst für Wirtschaftsjahre, die nach einem Stichtag rund drei Jahre nach Verkündung beginnen. Für die Architektur bedeutet das: die Exportschicht sollte formatunabhängig gebaut sein.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Exportschicht trennt Datenmodell und Ausgabeformat | ✅ | internal/export/dataset.go:1-14 benennt die Trennung und setzt sie durch: `Dataset`, `Table` und `Field` (:45-78) kennen kein Format, die Erzeuger kennen keine Buchhaltung (internal/export/csv.go:34, internal/export/gdpdu.go:52, internal/export/fielddoc.go:97), und die Auswahl entsteht ausschließlich in internal/service/export_tables.go. Ein Erzeuger für xBRL-CSV tritt daneben, ohne die Auswahl anzufassen; die E-Bilanz bleibt der Altfall (internal/ebilanz/ebilanz.go:199-279) | – |
| Internes Datenmodell hält alle Felder vor, die der Entwurf verlangt | 🟡 | internal/domain/journal.go:104-253 ist feldreich: vier getrennte Daten, Steuerschlüssel und Bemessungsgrundlage je Zeile, Währung mit Kurs und Quelle, Belegverweis, Regelversion, Herkunft, und seit Welle 6 Bearbeiterkennung, Programmfassung, Festschreibungszeitpunkt und Herkunftskennung aus einem Altsystem. Es fehlen eine Kostenstelle und eine feldbezogene Änderungshistorie der Buchung — die gibt es für Stammdaten und Belegkopfdaten (UNV-03), an der Buchung ist eine Änderung überhaupt nicht vorgesehen | Politur |
| Entwicklungsplan enthält einen Prüfpunkt für den Zeitpunkt der Verkündung | ❌ | docs/entwicklung/architektur.md:112-114 nennt die Schnittstelle nach § 147b AO als künftiges Formatmodul, aber keinen Termin, zu dem der Stand der Verordnung zu prüfen wäre; der Terminplan dieses Katalogs führt sie weiterhin ohne Datum als „offen" | Politur |

**Stand.** Die Architekturfolge ist mit Welle 4 gezogen: der Z3-Export ist so geschnitten, dass die Datenauswahl formatfrei bleibt, und xBRL-CSV wäre ein Erzeuger neben CSV und index.xml, kein Umbau. Offen bleiben zwei Felder im Datenmodell und der Prüfpunkt für den Tag, an dem die Verordnung verkündet wird. Beides ist Politur.

---
