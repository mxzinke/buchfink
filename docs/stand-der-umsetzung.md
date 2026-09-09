# Buchfink – Stand der Umsetzung

Status: laufend gepflegt
Letzte Aktualisierung: 2026-09-08

Dieses Dokument beschreibt, was Buchfink heute tut, wo eine Funktion
eingeschränkt ist und was noch fehlt. Es ist die erzählende Gegenprobe zum
[Anforderungskatalog](anforderungskatalog.md): dort steht jedes Kriterium mit
Norm, Status und Fundstelle, hier steht dasselbe in Vorgangssprache. Warum
Buchfink so geschnitten ist, steht in [docs/architektur.md](architektur.md),
Abschnitt 2.

Der Katalog zählt 349 Akzeptanzkriterien: 242 erfüllt, 38 teilweise, 14 fehlend,
55 außerhalb des Funktionsumfangs. Acht Umsetzungswellen sind gebaut. Welle 8
hat die laufende Buchhaltung abgeschlossen: Belege, Journal, Konten,
Festschreibung, Protokoll, Aufbewahrung, Rechnungen und Umsatzsteuer haben
kein fehlendes Kriterium mehr. Offen bleibt der Jahresabschluss als Dokument
mit Feststellung, Offenlegung und Prüfung — bewusst zurückgestellt, bis ein
Geschäftsjahr laufend geführt worden ist.

## 1. Was funktioniert

| Modul | Was funktioniert | Fundstelle |
|---|---|---|
| A. Buchführungspflicht und Grundsätze | Jeder Buchungssatz gleicht sich ohne Toleranz aus, `Post` ist der einzige Schreibweg ins Journal, der Saldenvortrag bringt die Bestandskonten ins Folgejahr, Bilanz und GuV entstehen allein aus Kontensalden, das Gründungsjahr beginnt mit der Beurkundung und ist ein Rumpfgeschäftsjahr. | `internal/domain/journal.go:325`, `internal/service/journal_service.go:97`, `internal/service/closing_service.go:1129`, `:1641`, `internal/accounting/statement.go:291` |
| B. Beleg, Journal, Konten | Beleg unter seinem SHA256 abgelegt, Kopfdaten als Pflicht vor dem Buchen, Nummernkreise ohne Doppelvergabe in der Transaktion, Storno als einzige Korrektur, offene Posten mit Stichtag, Prüfbericht aus vierzehn Regeln vor jeder Festschreibung, Handbuchung nur mit Beleg oder Eigenbeleg, eigene Konten mit HGB-Position. | `internal/service/receipt_service.go:124`, `:285`, `internal/repository/numberrange_gorm.go:49`, `internal/service/journal_service.go:258`, `internal/service/payment_service.go:185`, `internal/service/check_service.go:167`, `internal/service/manual_entry.go:38`, `internal/service/self_issued_receipt.go:68`, `internal/service/account_service.go:84` |
| C. Unveränderbarkeit und Protokollierung | Hashkette über das Journal, zweite Kette über das Änderungsprotokoll mit Vorher und Nachher, Bearbeiterkennung und Programmfassung an jeder Buchung, Festschreibung mit RFC-3161-Zeitstempel, „Monat festschreiben" als Aufgabe ab dem 10. des Folgemonats, Steuersätze als datierte Tabelle. | `internal/accounting/journalhash.go:198`, `internal/accounting/audithash.go:65`, `internal/actor/actor.go:38`, `internal/buildinfo/buildinfo.go:29`, `internal/timestamp/tsa.go`, `internal/service/task_service.go:446`, `internal/accounting/tax_params.go:326` |
| D. Aufbewahrung und Archivierung | Fristenklasse aus der Belegart mit Fristbeginn und frühestem Löschdatum, Aussetzung je Geschäftsjahr, Archivexport mit Index, Sicherung und Wiederherstellung als Vorgang. | `internal/accounting/retention.go:121`, `internal/domain/retention.go:158`, `internal/service/retention_service.go:220`, `internal/service/export_service.go:145`, `internal/service/backup_service.go:419` |
| E. Ausgangsrechnungen und E-Rechnung | Rechnungsnummer, Datensatz und Buchung in einer Transaktion, Pflichtangaben vor der Nummernvergabe geprüft, ZUGFeRD und XRechnung im CII-Profil, Storno und Berichtigung als eigene Dokumente, Anzahlungen als Rechnungsverbund, Empfang mit Regelwerksprüfung und Beanstandungen nach Fehlerklassen, Abgleich des eingebetteten XML gegen den Datensatz vor der Ablage. | `internal/service/invoice_service.go:114`, `internal/service/invoice_correction.go:117`, `internal/service/advance_service.go:137`, `internal/einvoice/validate.go:134`, `internal/service/receipt_findings.go:170`, `internal/invoice/hybrid_check.go:33` |
| F. Umsatzsteuer, Aufzeichnung und Meldewesen | Voranmeldung mit allen Kennziffern des Vordrucks USt 1 A aus den Steuerzeilen, Zusammenfassende Meldung, Verzeichnis nach § 15a UStG, Bestätigung der USt-IdNr. beim Bundeszentralamt, Belegnachweis je Lieferung, Voranmeldungszeitraum aus der Steuer des Vorjahres, geteilter Vorsteuerabzug auch beim Erwerb und beim Fall des § 13b UStG. | `internal/accounting/ustva.go:161`, `internal/service/vat_return_service.go:114`, `internal/accounting/zm.go:172`, `internal/service/input_tax_service.go:14`, `internal/vatid/client.go:141`, `internal/service/supply_evidence_service.go:115`, `internal/service/vat_period_proposal.go:58`, `internal/service/posting_service.go:1449` |
| G. Bewertung, Anlagen, Fremdwährung | AfA-Sätze und Wertgrenzen als datierte Ressource, Anlagenspiegel aus der jahresübergreifenden Kartei, Rückstellungen mit Abzinsung, Rechnungsabgrenzung mit monatlicher Auflösung, Inventurwert als Bestandsveränderung, Kurse mit EZB-Abruf. | `internal/accounting/afa_rules.json`, `internal/accounting/afa.go:50`, `internal/service/asset_service.go:2724`, `internal/accounting/provision.go:20`, `internal/service/accrual_service.go:567`, `internal/accounting/inventory.go:43`, `internal/currency/ecb.go:69` |
| H. Jahresabschluss, E-Bilanz, Offenlegung | Bilanz und GuV nach §§ 266, 275 HGB mit Vorjahresspalte und Ausgabe als Datei, Größenklasse aus zwei Stichtagen, E-Bilanz aus derselben Gliederung, geführter Abschlussweg mit Fortschritt aus dem Backend, Ergebnisverwendung mit Beschluss, Gründungsweg von der Beurkundung bis zur Offenlegung mit Eröffnungsbilanz als PDF und E-Bilanz. | `internal/accounting/statement.go:291`, `internal/service/statement_export.go:20`, `internal/accounting/groessenklasse.go:89`, `internal/ebilanz/ebilanz.go:112`, `internal/service/closing_steps_service.go:43`, `internal/service/appropriation_service.go:93`, `internal/service/foundation_opening.go` |
| I. Betriebsprüfung und Verfahrensdokumentation | Z3-Export nach dem Beschreibungsstandard mit `index.xml` und Feldbeschreibung, Prüferpaket in einem Ordner, Journalfilter mit Summenzeile und protokollierter CSV-Ausgabe, schreibgeschützter Prüfermodus mit Frist und Grund, Verfahrensdokumentation aus dem laufenden System. | `internal/service/export_service.go:136`, `:151`, `internal/export/gdpdu.go:52`, `internal/export/fielddoc.go`, `internal/wailsbridge/readonly.go:323`, `internal/procdoc/procdoc.go:175`, `internal/accounting/journal_filter.go:95` |
| J. Querschnitt | Unterlagen des Unternehmens unter ihrer Prüfsumme abgelegt, Feldverschlüsselung mit AES-256-GCM und Schlüssel je Mandant im Schlüsselbund, Mahnwesen mit Basiszinssatz, Verzugszinsen und Pauschale, Prüfpfad vom Beleg zu Buchung, Zahlung und Bankumsatz, Verzeichnis nach Art. 30 DSGVO in der Verfahrensdokumentation, Protokollfilter „Zugriffe" über jede Herausgabe. | `internal/service/document_service.go`, `internal/repository/encryption.go:81`, `internal/security/keyring.go:34`, `internal/service/dunning_service.go:146`, `internal/accounting/default_interest.go:63`, `internal/service/audit_trail_service.go:97`, `internal/procdoc/procdoc.go:523`, `internal/domain/audit.go:75` |

Die Bedienung liegt über diesen Funktionen: Aufgabenliste als Startseite,
Monatsabschluss in drei Schritten, Jahresabschluss als geführter Weg
(`internal/service/task_service.go`, `internal/service/month_close_service.go`,
`internal/service/closing_steps_service.go`; docs/architektur.md Abschnitt 6).

## 2. Wo eine Funktion eingeschränkt ist

38 Kriterien sind teilweise erfüllt. Jede Zeile nennt, was fehlt.

**A. Buchführungspflicht und Grundsätze**

- Testlauf mit einer fachkundigen Person: gemessen wird der Klickweg, nicht das Verständnis.
- Anhang in deutscher Sprache: Bilanz und GuV gehen als PDF und CSV hinaus, ein Anhang aus den Daten entsteht nicht.
- Zeitraum eines bestehenden Geschäftsjahres: der Beginn des Gründungsjahres zieht auf die Beurkundung nach, sobald die Gründung erfasst wird. Wer vor dieser Welle gegründet hat, führt weiterhin das volle Kalenderjahr — von Hand richtigstellen lässt sich der Zeitraum nicht.

**D. Aufbewahrung und Archivierung**

- Fristenlogik: die Fristen liegen als Ressource mit Fassung, Quelle und Gültigkeitsbeginn neben dem Code, eingebettet in das Programm und nicht als Einstellung.
- Aufbewahrungs-Hold: er gilt je Geschäftsjahr, nach Steuerart lässt er sich nicht schneiden.
- Lesbarmachung: der Export erklärt sich selbst, ein Einlesen auf einem fremden System ist nicht belegt.
- Abstimmung beim Systemwechsel: die übernommene Datei wird gegen sich selbst gerechnet, Zahlen aus dem Altsystem liest Buchfink nicht.
- Sicherungen: Zielordner und Läufe sind dokumentiert, wie lange eine Sicherung aufzubewahren ist, sagt das Löschkonzept nicht.

**E. Ausgangsrechnungen und E-Rechnung**

- Leistungsdatum: Pflichtfeld in der Rechnung, ein leeres Feld wird still auf das Rechnungsdatum gesetzt.
- Zielprofil je Empfänger: drei Profile sind wählbar, die UBL-Ausprägung der XRechnung fehlt.

**F. Umsatzsteuer, Aufzeichnung und Meldewesen**

- Aufzeichnung je Zeitraum: der Vordruck ist vollständig, die Umsatzsteuerkorrektur eines gewährten Skontos erreicht ihn nicht.
- Amtlicher Datensatz: das Kennziffernblatt geht als CSV hinaus, ein amtlich erzeugter Datensatz entsteht ohne ERiC nicht.

**G. Bewertung, Anlagen, Fremdwährung**

- Bewertungsmethoden je Bilanzposition: geführt wird die Methode je Anlagegut, die Angabe nach § 284 Abs. 2 Nr. 1 HGB ist Anhangtext von Hand.
- Wertansätze: das Anlagevermögen wird einzeln bewertet, Sammelbewertungen betreffen Vorräte und fehlen mit ihnen.
- Parallele Wertansätze: die Sonderabschreibung nach § 7g EStG steht neben dem Handelswert, abweichende Anschaffungskosten kennt der Datensatz nicht.
- Anlagenspiegel: alle Spalten des laufenden Jahres, ein vollständiger Vorjahresspiegel fehlt.
- Anlagenspiegel in der Taxonomie: der Block steht in der Instanz, die Elementnamen sind selbst gebildet.
- Abschreibungsmethoden: sechs Methoden sind je Konto hinterlegt, die Leistungsabschreibung ist außerhalb des Umfangs.
- Datierte Regelsätze: Sätze und Fenster liegen als Ressource neben dem Code, die Datei ist eingebettet und reist mit der Auslieferung.
- Methodenwechsel: der Übergang von degressiv auf linear läuft automatisch und steht in der Planzeile, nicht im Stammsatz.
- Unterschiedliche Nutzungsdauern: die Differenz aus § 7g EStG ist auswertbar, ein zweiter Bewertungskreis entsteht nicht.
- AfA-Tabellenwerte: neun von dreiundvierzig Konten haben einen Vorschlag mit Begründungspflicht, die übrigen keinen.
- Wertgrenzen: datiert und an einer Stelle, parametrisierbar ausdrücklich nicht.
- Verzeichnis der Wahlrechte: die Überleitung geht in die Instanz, ihre Elementnamen sind ungeprüft.

**H. Jahresabschluss, E-Bilanz, Offenlegung**

- Größenklassenwechsel: der abweichende Stichtag wird benannt, eine Ankündigung des bevorstehenden Wechsels fehlt.
- Anhangangaben: drei kommen aus den Daten, Haftungsverhältnisse, finanzielle Verpflichtungen und Beteiligungsliste sind Freitext.
- Anhangumfang: er folgt der Größenklasse nicht, dieselben Abschnitte stehen in jeder Klasse.
- Kleinstkapitalgesellschaft: unter der Bilanz stehen bisher nur die Restlaufzeiten.
- XBRL nach gültiger Taxonomie: die Instanz entsteht aus derselben Gliederung wie die Bilanz, kein Elementname ist gegen die amtliche Fassung geprüft (`internal/ebilanz/taxonomy_6.9.json`, durchgehend `verified: false`).
- Kontennachweise: unverdichtet je Konto in der Instanz, die Hüllelemente sind frei gebildet.
- Anlagenspiegel ab 2028: er geht mit, das Anlagenverzeichnis nicht.
- Überleitung in der E-Bilanz: der Block steht in der Instanz, die Elementnamen sind ungeprüft.
- Offenlegungsumfang: er wird aus der Größenklasse gesetzt und angezeigt, ein darauf beschränkter Datensatz entsteht nicht.

**I. Betriebsprüfung und Verfahrensdokumentation**

- Eingrenzung des Zugriffs: die Überlassung ist auf ein Geschäftsjahr begrenzt und protokolliert, Mandanten- und Jahreswechsel sind auch im Prüfermodus offen.
- Freie Auswertung: das Journal filtert nach Konto, Gegenkonto, Betragsbereich, Steuerschlüssel, Bearbeiter, Belegkennzeichen, Zeitraum und Volltext, summiert die gefilterte Menge und gibt sie als CSV heraus; die Sortierung bleibt die reproduzierbare Journalordnung, eine freie Sortierwahl gibt es nicht.
- Zielformate: CSV nach RFC 4180 mit benanntem Trennzeichen, XLSX ist mit Begründung weggelassen.
- Datenmodell für § 147b AO: feldreich bis zur Bearbeiterkennung, es fehlen Kostenstelle und feldbezogene Änderungshistorie der Buchung.

**J. Querschnitt**

- Verschlüsselung: personenbezogene Felder liegen verschlüsselt und das Verfahren steht in der Verfahrensdokumentation, Kontonummern, Beträge und Datumsangaben stehen im Klartext.
- Berechtigungen je Mandant: der Schlüssel je Mandant sperrt den Zugriff, eine Vergabe gibt es ohne Benutzer nicht.

## 3. Was fehlt

Vierzehn Kriterien sind offen. Keines davon gehört zur laufenden Buchhaltung.

**(a) Laufende Buchhaltung.** Kein Kriterium der Module B, C, E, F und J ist
mehr als fehlend eingestuft. Belege, Journal, Konten, Festschreibung, Protokoll,
Aufbewahrung, Ein- und Ausgangsrechnungen, Voranmeldung und Zusammenfassende
Meldung decken den Alltag vollständig ab. Übrig sind Ränder, die als teilweise
erfüllt in Abschnitt 2 stehen und keinen Vorgang blockieren: die
Aufbewahrungsfristen liegen als eingebettete Ressource statt als Einstellung,
das Journal kennt keine freie Sortierwahl, ein leeres Leistungsdatum wird still
auf das Rechnungsdatum gesetzt, die Umsatzsteuerkorrektur eines gewährten
Skontos erreicht den Vordruck nicht, die Verschlüsselung reicht über die
personenbezogenen Felder nicht hinaus, und der Durchlauf mit einer fachkundigen
Person ohne Produktkenntnis steht aus — er gehört zur Erprobung und nicht zum
Code.

**(b) Jahresabschluss und Prüfung.** Diese vierzehn Kriterien sind bewusst
zurückgestellt, bis ein Geschäftsjahr laufend geführt worden ist. Neun von
ihnen richten sich nach demselben Objekt, das Buchfink noch nicht führt: dem
Jahresabschluss als Dokument.

| Was fehlt | Katalog |
|---|---|
| Feststellungsbeschluss als Dokument am Geschäftsjahr. Der Beschluss über die Ergebnisverwendung lässt sich ablegen, der Feststellungsbeschluss nicht. | JAB-04 |
| Unterzeichneter Jahresabschluss als unveränderliches, archiviertes Dokument. Es entsteht kein Abschlussdokument, mit dem sich die übrigen Nachweise verknüpfen ließen. | JAB-04 |
| Datensatz für die Einreichung beim Unternehmensregister. Der offenzulegende Umfang wird aus der Größenklasse gesetzt, ein darauf beschränkter Datensatz entsteht nicht. | JAB-07 |
| Hinterlegung nach § 326 Abs. 2 HGB als Wahl für Kleinstgesellschaften. Die Norm steht bisher nur im Beschreibungstext. | JAB-07 |
| Einreichungsnachweis der Offenlegung, archiviert am Abschluss. | JAB-07 |
| Saldenbestätigungslauf für Debitoren und Kreditoren mit Anschreiben und festgehaltener Rückmeldung. Der Weg zum Anschreiben besteht für das Mahnwesen. | JAB-08 |
| Prüfungsvermerk und Prüfungsbericht als Dokument am Abschluss. Ohne Abschlussobjekt gibt es nichts, womit sie zu verknüpfen wären. | JAB-08 |
| Übermittlungsprotokoll der E-Bilanz. Das Muster steht an der Voranmeldung, für die E-Bilanz gibt es kein solches Objekt. | JAB-05 |
| Bericht über alle im Geschäftsjahr geänderten Bewertungsmethoden. Geführt wird die Methode je Anlagegut, ein Bericht über die Änderungen entsteht daraus nicht. | BEW-01 |
| Getrennte Erfassung der Pflicht- und Wahlbestandteile der Herstellungskosten mit gespeicherter Wahlrechtsausübung. Geführt wird ein Gesamtbetrag. | BEW-02 |
| Steuerung der Befreiung von den latenten Steuern über die Größenklasse. Die Klasse wird geführt, die Befreiung nach § 274a Nr. 4 HGB ist nicht daran gebunden. | BEW-11 |
| Testeinlesen der Datenüberlassung in eine Prüfsoftware. Der Export ist gegen die amtliche Grammatik gebaut und getestet, ein Lauf in IDEA oder ACL hat nicht stattgefunden. | PRF-02 |
| Prüfpunkt mit Datum für die Verkündung der Verordnung nach § 147b AO. Mit dieser Fassung in docs/architektur.md Abschnitt 4 ergänzt. | PRF-06 |
| Bereitstellung des Jahresabschlusses an die Gesellschafter in der Frist des § 42a GmbHG. Ohne Abschlussdokument gibt es keine Frist. | QUE-06 |

## 4. Bewusst außerhalb des Umfangs

55 Kriterien sind ausgelassen. Jede Auslassung folgt einer Grundentscheidung
aus [docs/architektur.md](architektur.md), Abschnitt 2.

**Einzelplatz, ein Bearbeiter.** Kein Rollenmodell, keine Benutzerkonten, keine
Funktionstrennung im System (UNV-04, QUE-03). An ihre Stelle treten die
Bearbeiterkennung an jeder Buchung und der schreibgeschützte Prüfermodus, wo
der Katalog ein Benutzerkonto für Dritte verlangt (JAB-08, PRF-01, QUE-06).
Dazu gehört der Local-First-Betrieb: kein Cloud-Betrieb, keine
Auftragsverarbeitung, keine Verlagerung nach § 146 Abs. 2a AO (ARC-06, PRF-05,
QUE-02).

**Keine ERiC-Anbindung.** Buchfink übermittelt nichts selbst. Voranmeldung,
Zusammenfassende Meldung und E-Bilanz entstehen als Kennziffernblatt und als
Datei für Mein ELSTER oder den Steuerberater; das Übermittlungsprotokoll wird
danach von Hand erfasst und ist dann unveränderlich (UST-03, JAB-05).

**Geschlossene Steuerfall-Liste.** Ausgeschlossen sind Kleinunternehmer
(UST-09, RECH-05), Differenzbesteuerung, Reiseleistungen und Dreiecksgeschäft
(RECH-04), OSS und IOSS (UST-08), Konsignationslager (UST-01), Bauleistungen
nach § 13b Abs. 2 Nr. 4 UStG (UST-05) und die Option nach § 9 UStG. Ebenfalls
ausgeschlossen: die Istversteuerung (UST-02), die Abrechnungsgutschrift
(RECH-02), der Versandweg über Peppol oder EDI (RECH-06) und mehrsprachige
Rechnungshinweise (RECH-04). Die Oberfläche sagt bei einem ausgeschlossenen
Fall, dass Buchfink ihn nicht abbildet.

**Keine Kasse, kein Lager, kein Lohn.** Kein Kassenbuch, kein Vorratsmodul,
keine Lohnabrechnung (BEW-09, PRF-04). Der Vorratsbestand wird zum Stichtag als
Inventurwert erfasst, der Lohn kommt als Sammelbuchung aus dem Lohnjournal.
Ohne Kassenfunktion greift die KassenSichV nicht. Hierher gehört auch das
ersetzende Scannen: Buchfink erklärt es nicht zum unterstützten Verfahren, der
Papierbeleg bleibt aufzubewahren (BEL-08).

**Kapitalgesellschaften zuerst.** Kapitalkonten der Gesellschafter, Entnahmen
und der Schuldzinsenabzug nach § 4 Abs. 4a EStG sind nicht abgebildet (BEW-13).
KG, OHG und e.K. bleiben wählbar und zeigen den Hinweis in der Oberfläche. Dazu
gehören das Umsatzkostenverfahren (JAB-01) und das Merkmal der
Kapitalmarktorientierung (JAB-02), die beide nicht setzbar sind.

**SKR04 als Einheitsbilanz.** Ein Kontenrahmen, ein Wertansatz (BEW-02, BEW-03,
BEW-04, BEW-07, JAB-06). Abweichende steuerliche Werte entstehen allein aus der
Sonderabschreibung nach § 7g Abs. 5 EStG und werden am Anlagegut mitgeführt;
daraus entstehen das Verzeichnis nach § 5 Abs. 1 S. 2 EStG und die
Überleitungsrechnung. Latente Steuern entfallen für kleine Kapitalgesellschaften
nach § 274a Nr. 4 HGB (BEW-11), die Leistungsabschreibung nach § 7 Abs. 1 S. 6
EStG ist nicht abgebildet (BEW-04).

**Buchführungsdatenschnittstelle nach § 147b AO.** Die Verordnung ist noch nicht
erlassen; der Diskussionsentwurf 2026 legt xBRL-CSV 1.0 fest. Solange sie
aussteht, entsteht kein Export in diesem Format (PRF-06). Die Exportschicht ist
formatunabhängig geschnitten, damit die Schnittstelle ein weiteres Formatmodul
wird.

## 5. Wie es weitergeht

Die laufende Buchhaltung ist gebaut. Was bleibt, sind die 38 teilweise
erfüllten Kriterien und die 14 offenen, jedes mit seinem Grund im Katalog. Neun
der offenen Punkte richten sich nach demselben Objekt — dem Jahresabschluss als Dokument
mit Feststellung, Offenlegung und Nachweisen. Wer dort ansetzt, schließt sie
zusammen. Sie stehen bewusst hinten an: erst soll ein Geschäftsjahr laufend
geführt worden sein.

Danach folgt die Erprobung mit echten Anwendern. Es gibt noch keine. Bis eine
Gründerin ein Geschäftsjahr durchgeführt hat, sind die gemessenen Klickwege und
die Prüfläufe Aussagen über den Code und nicht über die Bedienung.

Prüfpunkte mit Datum, jeder mit dem Ereignis, das ihn auslöst:

| Wann | Was zu prüfen ist |
|---|---|
| bei Verkündung, offen | Verordnung nach § 147b AO (DSFinVBV). Der Diskussionsentwurf 2026 legt xBRL-CSV 1.0 fest. Mit der Verkündung wird der Export ein weiteres Formatmodul (PRF-06). |
| 01.01.2027 und 01.01.2028 | Übergangsfrist der E-Rechnung auf der Ausstellerseite (§ 27 Abs. 38 UStG). Ab 2027 gilt die Sendepflicht bei einem Vorjahresumsatz über 800.000 Euro, ab 2028 für alle inländischen B2B-Umsätze (RECH-06). |
| 01.01. und 01.07. jedes Jahres | Basiszinssatz nach § 247 BGB. Die Bundesbank gibt ihn halbjährlich bekannt; er wird als datierte Zeile in den Einstellungen nachgetragen (QUE-05). |
| jährlich, nach dem BMF-Schreiben | Stand der E-Bilanz-Taxonomie. Taxonomie 6.9 gilt für Wirtschaftsjahre ab 2026, 6.10 ab 2027. Die Elementnamen in `internal/ebilanz/taxonomy_6.9.json` sind durchgehend mit `verified: false` markiert und vor der ersten Übermittlung gegen die amtliche Fassung abzugleichen (JAB-05). |
| jährlich | Jahresstand des SKR04. Der Kontenrahmen liegt als `internal/accounting/skr04_2026.json` bei; eine neue Fassung ist eine neue Datei (BEL-06). |
