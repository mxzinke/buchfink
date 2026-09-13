# J. Querschnitt

[Anforderungskatalog](README.md) · [Legende](README.md#legende) · [Fachkonzepte](../fachkonzepte/README.md)

### QUE-01 Löschpflicht und Aufbewahrungspflicht `MUSS`

**Norm:** Art. 5 Abs. 1 lit. e, Art. 17 Abs. 3 lit. b DSGVO, § 257 HGB, § 147 AO

**Bedeutung:** Die Löschpflicht der DSGVO entfällt, soweit die Verarbeitung zur Erfüllung einer rechtlichen Verpflichtung erforderlich ist. Handels- und steuerrechtliche Aufbewahrungsfristen gehen dem Löschanspruch also vor. Nach Fristablauf kehrt sich das Verhältnis um: dann ist zu löschen.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Löschantrag geprüft und, soweit Aufbewahrungspflichten greifen, mit Normverweis zurückgestellt statt abgelehnt | ✅ | internal/accounting/retention.go:246-259 (`BlockedContactAnswer`) formuliert die Antwort an die betroffene Person mit § 257 HGB, § 147 AO, Art. 17 Abs. 3 Buchst. b und Art. 18 DSGVO und sagt die Löschung nach Fristablauf zu; internal/service/contact_service.go:226-251 sperrt den Geschäftspartner dazu und protokolliert den Vorgang mit Grund, internal/wailsbridge/nachweise_service.go:335-358 gibt Sperre und Antworttext an die Oberfläche (frontend/src/pages/ContactsPage.tsx:254-310) | – |
| Betroffene Datensätze bis zum Fristablauf gesperrt | ✅ | internal/domain/contact.go:108-110 führt Sperre, Sperrdatum und Grund am Geschäftspartner; internal/service/contact_service.go:274-295 (`ensureNotBlocked`) weist jeden Schreibweg ab, der ihn noch verwenden würde, :265-272 nimmt ihn aus jeder Auswahl. In Buchungen, Rechnungen und Exporten bleibt er sichtbar, weil die Aufbewahrungspflicht dem Löschanspruch vorgeht, und die Liste kennzeichnet ihn als gesperrt (frontend/src/pages/ContactsPage.tsx:222-240) | – |
| Nach Fristablauf löscht ein Verfahren oder legt einer dokumentierten Entscheidung vor, Löschung protokolliert | ✅ | internal/service/retention_service.go:272-288 führt die Geschäftsjahre, deren Frist abgelaufen ist und für die keine Aussetzung gilt; gelöscht wird nur auf ausdrückliche Anweisung: :367-429 (`ArchiveAndDelete`) verlangt die Jahreszahl als ausgeschriebene Bestätigung und einen erstellten Archivexport, löscht das Jahr in einer Transaktion (internal/repository/welle6_gorm.go:223-330) und schreibt danach ins Protokoll, was verschwunden ist — je Objektart gezählt, mit den entfernten Dateien und dem Archivpfad | – |
| Löschkonzept ordnet jeder Datenkategorie eine Frist und eine Rechtsgrundlage zu | ✅ | internal/accounting/retention.go:164-238 (`DeletionConcept`) führt neun Datenkategorien mit Klasse, Frist, Rechtsgrundlage und dem, was nach Fristablauf geschieht — vom Journal über die Fassungen der Verfahrensdokumentation bis zu den Stammdaten der Geschäftspartner. Dieselbe Tabelle ist Abschnitt 3.8 der Verfahrensdokumentation (internal/procdoc/procdoc.go:424-441) und die Regel, nach der die Löschfunktion entscheidet; sie steht in der Oberfläche (frontend/src/pages/BackupPage.tsx:869-907) | – |

**Stand.** Mit Welle 6 hat die Löschseite ihre Antwort: das Löschkonzept ordnet jeder Datenkategorie ihre Frist und ihre Rechtsgrundlage zu, ein Löschverlangen wird mit dem Normverweis zurückgestellt und der Geschäftspartner stattdessen gesperrt — in keiner Auswahl mehr wählbar, in Buchungen und Exporten weiter sichtbar —, und nach Fristablauf steht die Löschung des Geschäftsjahres als Vorgang bereit: Archivexport davor, ausgeschriebene Bestätigung, Protokoll danach. Alle vier Kriterien sind erfüllt.

### QUE-02 Technische Maßnahmen und Auftragsverarbeitung `MUSS`

**Norm:** Art. 28, Art. 32 DSGVO

**Bedeutung:** Buchhaltungsdaten enthalten personenbezogene Daten von Mitarbeitern, Kunden und Lieferanten. Bei Cloud-Betrieb ist ein Auftragsverarbeitungsvertrag erforderlich.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Übertragung und Speicherung verschlüsselt, die Verfahren dokumentiert | 🟡 | internal/repository/encryption.go:36-58 verschlüsselt feldweise mit AES-256-GCM, Schlüssel im Betriebssystem-Schlüsselbund. Das Verfahren steht seit Welle 8 vollständig in der Verfahrensdokumentation: internal/procdoc/procdoc.go:556-575 beschreibt als Maßnahme nach Art. 32 DSGVO die Verschlüsselung mit einem Schlüssel je Mandant, die Verwahrung im Schlüsselbund, den Wiederherstellungsschlüssel, den Zugriffsschutz des Betriebssystems, die Protokollierung, die Hash-Ketten und die Sicherung. Verschlüsselt ist, was `serializer:encrypted` hat — Vorher- und Nachherwerte des Protokolls (internal/domain/audit.go:46-47), die personenbezogenen Kopfdaten des Belegs (internal/domain/receipt.go:209-214), der Leistungsnachweis (:282); Kontonummern, Beträge und Datumsangaben stehen weiter im Klartext | Politur |
| Zugriffe auf personenbezogene Daten protokolliert und auswertbar | ✅ | internal/domain/audit.go:11-18 erfasst die Schreibvorgänge; daneben steht jeder Vorgang, bei dem personenbezogene Daten das Programm verlassen oder geöffnet werden — Journalexport und Datenüberlassung (internal/service/export_service.go:254, :1083), Sicherungslauf, Datenübernahme und der Prüfermodus mit Grund und Frist (internal/wailsbridge/readonly.go:342, :375). Welle 8 schließt die beiden offenen Wege: die Herausgabe einer einzelnen Belegdatei wird beim Schreiben protokolliert und nicht erst im Dialog (internal/wailsbridge/export_service.go:220-245), ebenso die CSV-Ausgabe des gefilterten Journals (internal/service/journal_filter_csv.go:77-95). Ausgewertet wird das über internal/domain/audit.go:75-83 (`AuditFilter.Access`), das die Ausgaben und das Ein- und Ausschalten des Prüfermodus in einer Abfrage zusammenfasst, weil „Zugriffe" eine Kategorie ist und keine Aktion; das Änderungsprotokoll der Prüfübersicht führt sie als eigenen Filter (frontend/src/pages/AuditPage.tsx:1045-1157). Welche Vorgänge als Zugriff gelten, sagt die Verfahrensdokumentation (internal/procdoc/procdoc.go:565-570) | – |
| Auftragsverarbeitungsvertrag mit Unterauftragnehmern und Speicherorten | ⛔ | Keine Auftragsverarbeitung bei lokalem Betrieb. Einzige externe Verbindung ist der Zeitstempeldienst nach RFC 3161, an den nur ein Hash geht (internal/wailsbridge/festschreibung_service.go:57-64) | – |
| Verzeichnis von Verarbeitungstätigkeiten für die Buchhaltung erstellbar | ✅ | internal/procdoc/procdoc.go:204-212, :523-575 setzt das Verzeichnis nach Art. 30 DSGVO als eigenen Abschnitt der Verfahrensdokumentation: den Verantwortlichen aus den Unternehmensangaben, acht Verarbeitungstätigkeiten von der Finanzbuchhaltung bis zum Zugriffsprotokoll mit Zweck, Rechtsgrundlage, betroffenen Personen, Datenkategorien, Empfängern — Finanzverwaltung, Steuerberater, Bundeszentralamt, Zeitstempeldienst — und Löschfrist, die Fristen aus den Aufbewahrungsklassen samt dem Verhältnis zu Art. 17 DSGVO und die technischen und organisatorischen Maßnahmen nach Art. 32 DSGVO. Es entsteht aus dem laufenden System und wird mit der Verfahrensdokumentation als Fassung im Belegspeicher abgelegt (internal/service/procdoc_service.go) | – |

**Stand.** Die Vertraulichkeit ist der am besten ausgebaute Querschnittsteil, und mit Welle 8 hat sie ihre Rechenschaft: die Verfahrensdokumentation enthält das Verzeichnis von Verarbeitungstätigkeiten nach Art. 30 DSGVO — acht Tätigkeiten mit Zweck, Rechtsgrundlage, Datenkategorien, Empfängern und Löschfrist —, dazu die Löschfristen aus den Aufbewahrungsklassen und die Maßnahmen nach Art. 32 DSGVO. Jeder Vorgang, bei dem personenbezogene Daten das Programm verlassen, steht im Protokoll: Datenüberlassung, Prüferpaket, Prüfermodus, die Herausgabe einzelner Belegdateien und die CSV-Ausgabe des gefilterten Journals, auswertbar über den Filter „Zugriffe". Offen bleibt die Verschlüsselung über die personenbezogenen Felder hinaus — Kontonummern, Beträge und Datumsangaben stehen im Klartext. Politur.

### QUE-03 Mandanten- und Buchungskreistrennung `MUSS`

**Norm:** § 146 Abs. 1 AO, GoBD Rz 36 ff.

**Bedeutung:** Buchführungen verschiedener Unternehmen dürfen nicht vermischt werden. Auch bei getrennten Datenbanken pro Mandant muss die Trennung im Betrieb belegbar sein.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Ein Buchungsvorgang wirkt nur innerhalb eines Mandanten, mandantenübergreifende Buchungen ausgeschlossen | ✅ | internal/repository/db.go:24 führt eine Datenbank je Mandant, internal/wailsbridge/app_service.go:135-262 verdrahtet alle Repositories auf genau diese Verbindung; es ist stets nur ein Mandant offen | – |
| Auswertungen und Exporte mandantenscharf | ✅ | internal/wailsbridge/app_service.go:181-193; jede Auswertung liest über die Verbindung des aktiven Mandanten, eine mandantenübergreifende Auswertung gibt es nicht | – |
| Berechtigungen je Mandant vergeben | 🟡 | internal/security/keyring.go:67 hält je Mandant einen eigenen Umschlagschlüssel, ein fehlendes Geheimnis sperrt den Mandanten; das ist mandantenscharfer Zugriffsschutz auf Betriebssystemebene, aber keine Berechtigungsvergabe, weil es keine Benutzer gibt (siehe UNV-04) | – |
| Nummernkreise für Belege und Buchungen je Mandant unabhängig | ✅ | internal/domain/numberrange.go:36-43; die Tabelle liegt in der Mandantendatenbank, zwei Mandanten teilen keinen Zähler | – |

**Stand.** Strukturell ist die Trennung solide und die stärkste Querschnittsanforderung im Bestand. Was fehlt, ist die Berechtigungsvergabe, und die entfällt mit der Entscheidung für den Einzelplatzbetrieb.

### QUE-04 Zeit und Zeitstempel `MUSS`

**Norm:** § 146 Abs. 1 AO, GoBD Rz 45 ff., 107 ff.

**Bedeutung:** Fast jede Ordnungsanforderung hängt an einem Zeitpunkt. Eine manipulierbare oder unklare Systemzeit entwertet die Protokolle.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Alle Zeitstempel in UTC gespeichert und mit Zeitzone ausgegeben | ✅ | internal/service/journal_service.go:111 und internal/repository/journal_gorm.go:277-280 sichern den Buchungszeitstempel doppelt in UTC; seit Welle 6 gilt dasselbe für Änderungsprotokoll (internal/repository/audit_gorm.go:104), Festschreibung (internal/wailsbridge/festschreibung_service.go:70), Schema- und Migrationsprotokoll (internal/repository/migrations.go:63), Aussetzungen, Sperren, Belegänderungen und die Verfahrensdokumentation. Die Anzeige nennt die Zone dazu: frontend/src/utils/formatters.ts:185-194 und :204-213 setzen `timeZoneName` an jedem Zeitpunkt, Datumsfelder bleiben ohne | – |
| Systemzeit aus synchronisierter Quelle, Abweichung über einer Toleranz protokolliert | ✅ | internal/service/time_drift.go:15-59 hält die Systemzeit gegen die beglaubigte Zeit des Zeitstempeldienstes — die einzige Zeitangabe im Programm, die nicht von der Uhr dieses Rechners stammt — und beschreibt jede Abweichung über fünf Minuten nach Richtung und Größe; internal/wailsbridge/festschreibung_service.go:85-93 setzt den Vergleich bei jeder Festschreibung an, hält den Hinweis an ihr fest (internal/domain/festschreibung.go:30-39) und schreibt ihn mit beiden Zeiten als eigenen Protokolleintrag (:117-128) | – |
| Anwender können den Erfassungszeitstempel nicht setzen, Beleg- und Buchungsdatum sind getrennt | ✅ | internal/service/journal_service.go:111 überschreibt `CreatedAt` bedingungslos; vier getrennte Datumsfelder, drei davon Pflicht | – |
| Zeitstempel in Protokollen so geschützt wie die Buchungen | ✅ | internal/repository/audit_gorm.go:96-172 setzt `PreviousHash` und `EntryHash` an jedem Eintrag, gelesen und geschrieben in derselben Transaktion; der Zeitpunkt geht in die kanonische Form ein (internal/accounting/audithash.go:32-45), ein nachträglich verstellter Zeitstempel bricht die Kette und wird mit erwartetem und tatsächlichem Wert gemeldet (:65-125) — in der Oberfläche und im Prüferpaket (internal/service/export_service.go:867-899) | – |

**Stand.** Der Buchungszeitstempel war vorbildlich, seit Welle 6 ist es die ganze Zeitführung: jeder Zeitpunkt geht in UTC auf die Platte, die Anzeige nennt die Zone dazu, das Protokoll ist verkettet wie das Journal, und bei jeder Festschreibung wird die Uhr des Rechners gegen die beglaubigte Zeit gehalten — weicht sie um mehr als fünf Minuten ab, steht das mit beiden Zeiten im Protokoll und als Hinweis an der Festschreibung. Alle vier Kriterien sind erfüllt.

### QUE-05 Zahlungsziele, Verzug und Mahnwesen `SOLL`

**Norm:** §§ 271a, 286, 288 BGB

**Bedeutung:** Verzug tritt spätestens 30 Tage nach Fälligkeit und Zugang der Rechnung ein. Der Verzugszins beträgt im Geschäftsverkehr neun Prozentpunkte über dem Basiszinssatz, gegenüber Verbrauchern fünf Prozentpunkte. Hinzu kommt im B2B eine Pauschale von 40 Euro. Zahlungsfristen über 60 Tage sind im Geschäftsverkehr nur unter engen Voraussetzungen wirksam.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Basiszinssatz als pflegbare Zeitreihe mit Gültigkeit ab 1. Januar und 1. Juli, keine Hardcodierung | ✅ | internal/accounting/tax_params.go:260-292 führt den Satz nach § 247 BGB als datierte Tabelle: Stichtag, Wert in Hundertsteln eines Prozentpunktes, Quelle der Bekanntgabe und das Kennzeichen `Provisional` für einen fortgeschriebenen Wert. internal/accounting/default_interest.go:57-110 löst den Tag auf und legt die nachgetragenen Sätze über die hinterlegten; nachgetragen werden sie über internal/service/dunning_service.go:758-812 (`BaseRates`, `SaveBaseRate`) in internal/domain/dunning.go:181-219, und die Einstellungen zeigen die Reihe mit dem Vermerk zum fortgeschriebenen Wert (frontend/src/pages/SettingsPage.tsx:1094-1160). Vor dem ersten Eintrag liefert die Auflösung einen Fehler, damit kein Zins aus einem Satz von null entsteht | – |
| Verzugszinsen taggenau, mit unterschiedlichen Aufschlägen für Verbraucher und Unternehmer | ✅ | internal/accounting/default_interest.go:154-223 (`DefaultInterest`) rechnet abschnittsweise über die Halbjahre des Basiszinssatzes, taggenau über ein Jahr von 365 Tagen (:43-50) und je Abschnitt auf Cent gerundet, damit die Summe der im Schreiben ausgewiesenen Zeilen die Endsumme ergibt; die Zuschläge des § 288 BGB stehen als fünf Prozentpunkte gegenüber einem Verbraucher und neun im Geschäftsverkehr daneben (:19-29), der Verzug beginnt dreißig Tage nach Fälligkeit (:36-41, :111-124, § 286 Abs. 3 BGB), und welche Seite gilt, entscheidet internal/domain/contact.go:135-142 (`IsConsumer`) | – |
| Pauschale von 40 Euro bei Unternehmerforderungen ansetzbar | ✅ | internal/accounting/default_interest.go:30-34 (`DefaultInterestLumpSum`, § 288 Abs. 5 BGB); internal/service/dunning_service.go:146-318 setzt sie je Forderung einmal an — :177 liest, für welche Posten sie schon berechnet wurde, :270-291 legt sie an den Posten — und lässt sie gegenüber einem Verbraucher weg; das Mahnschreiben weist sie als eigene Zeile aus (:618-689) | – |
| Zahlungsziele über 60 Tage lösen einen Hinweis auf § 271a BGB aus | ✅ | internal/domain/invoice.go:167-186 (`LongPaymentTermDays`, `PaymentTermNotice`) nennt die Grenze und die Voraussetzung, unter der eine längere Frist wirksam ist; internal/wailsbridge/welle7_service.go:245-254 gibt den Satz an die Oberfläche, und der Rechnungsdialog zeigt ihn unter dem Feld, sobald die Zahl dort steht — beim Tippen und nicht erst beim Ausstellen | – |
| Mahnstufen, Fristen und Gebühren konfigurierbar, jede Mahnung mit Datum und Inhalt archiviert | ✅ | internal/domain/dunning.go:21-80 führt die Stufen mit Tagen nach Fälligkeit und Gebühr, voreingestellt Zahlungserinnerung nach sieben, erste Mahnung nach einundzwanzig und zweite nach fünfunddreißig Tagen (:39-52); gespeichert werden sie in den Einstellungen (internal/domain/settings.go:95-97) und dort gepflegt. internal/service/dunning_service.go:402-425 hält die Stufenfolge und den Abstand zwischen zwei Schreiben ein, :471-551 legt das Schreiben als PDF im Belegspeicher unter `dokumente/mahnungen/` ab (:25-27), hält Datum, Stufe, Posten, Zinsen, Gebühr und Pauschale als `domain.DunningNotice` fest (internal/domain/dunning.go:88-160) und schreibt den Vorgang ins Änderungsprotokoll; der Verlauf je Kunde kommt über :749-754 zurück | – |

**Stand.** Mit Welle 7 rechnet Buchfink den Verzug: der Basiszinssatz steht als datierte Reihe nach § 247 BGB im Programm und lässt sich nach jeder Bekanntgabe der Bundesbank nachtragen, die Zinsen laufen taggenau über die Halbjahre mit neun Prozentpunkten im Geschäftsverkehr und fünf gegenüber Verbrauchern, und die Pauschale von 40 Euro fällt je Forderung einmal an. Der Mahnlauf legt je Kunde einen Vorschlag mit Posten, Stufe, Zinsen und Gebühr vor, das Schreiben entsteht als PDF, liegt im Belegspeicher unter `dokumente/mahnungen/` und steht mit Datum, Stufe und Beträgen im Verlauf des Kunden; Gebühr und Zinsen bleiben ungebucht, weil sie erst mit der Zahlung Ertrag sind. Im Rechnungsdialog meldet ein Zahlungsziel über sechzig Tagen den Vorbehalt des § 271a BGB, solange die Zahl noch zu ändern ist. Alle fünf Kriterien sind erfüllt.

### QUE-06 Auskunft und Rechenschaft `SOLL`

**Norm:** § 259 BGB, § 51a GmbHG, § 42a GmbHG

**Bedeutung:** Gesellschafter einer GmbH haben ein Auskunfts- und Einsichtsrecht in die Bücher. Wer rechenschaftspflichtig ist, schuldet eine geordnete Zusammenstellung der Einnahmen und Ausgaben mit Belegen.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Lesender Zugang für Gesellschafter oder Beiräte, Umfang einstellbar, Zugriffe protokolliert | ✅ | Derselbe Prüfermodus wie in PRF-01 und JAB-08: internal/wailsbridge/readonly.go:37-189 lässt Lesen und Auswerten zu, :210-220 weist jede Änderung ab, befristet und mit protokolliertem Grund (:227-289). Je Person einstellbar ist der Umfang nicht — im Einzelplatzbetrieb gibt es keine Person, an der er hinge (docs/entwicklung/architektur.md Abschnitt 2); protokolliert wird der Modus, nicht der einzelne Lesezugriff (QUE-02) | – |
| Geordnete Zusammenstellung der Einnahmen und Ausgaben mit Belegverweisen als Bericht und als Datei | ✅ | frontend/src/pages/ReportsPage.tsx:179-267 zeigt sie am Bildschirm samt Belegnummer und Belegvorschau; als Datei geht dieselbe Aufstellung über den Journalexport eines beliebigen Zeitraums hinaus, der Belegnummer und Beleg-Prüfsumme je Buchung führt (internal/service/export_service.go:190-224, internal/service/export_tables.go:74-96), im Archivexport zusammen mit den Belegdateien selbst (internal/service/export_service.go:700-727) | – |
| Jahresabschluss steht den Gesellschaftern in der Frist des § 42a GmbHG zur Verfügung | ❌ | Es entsteht kein Jahresabschluss und keine zugehörige Frist (siehe JAB-04) | 1 |

**Stand.** Der Prüfermodus und die Dateiausgabe aus Welle 4 decken die ersten beiden Punkte mit ab: wer Einsicht verlangt, bekommt einen lesenden Zugang und eine geordnete Zusammenstellung mit Belegverweisen, ohne dass jemand daneben sitzen muss. Die Frist des § 42a GmbHG setzt weiterhin das Abschlussobjekt aus Welle 1 voraus.

---
