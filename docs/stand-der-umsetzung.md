# Stand der Umsetzung

Abgleich mit dem Repository: 12. September 2026, nach Bug Hunt und Nachbesserungen.

Diese Seite beschreibt die implementierten Funktionen und bekannte Grenzen.
Sie ist keine Freigabe für den produktiven Einsatz. Ziele stehen getrennt auf
der [Roadmap](roadmap.md), einzelne fachliche Kriterien im
[Anforderungskatalog](anforderungskatalog.md).

## Zielgruppe und Umfang

Buchfink ist eine lokale Desktop-Anwendung für doppelte Buchführung im SKR04.
Der Schwerpunkt liegt auf kleinen Kapitalgesellschaften, insbesondere GmbH und
UG. Auch eine Holding kann darunterfallen; ein Konzernabschluss wird nicht
unterstützt. Für die Bedienung setzen wir keine Buchhaltungs-, Rechts- oder
Programmierausbildung voraus. Die Installation der Vorschau erfordert derzeit
noch Entwicklungswerkzeuge.

Andere Rechtsformen sind teilweise auswählbar. Kapitalkonten je Gesellschafter,
Entnahmen und Einlagen für Personengesellschaften sind nicht vollständig
abgebildet. Keine EÜR, Lohnabrechnung, Kassenführung, Lagerverwaltung oder
Konzernrechnungslegung. Die eigene Kleinunternehmerregelung und Istversteuerung
werden nicht unterstützt; ein Kleinunternehmer als Lieferant ist erfassbar.

## Implementierte Funktionen

| Vorgang | Aktuelles Verhalten | Code |
|---|---|---|
| Belege und Unterlagen ablegen | Originaldateien bleiben erhalten. E-Rechnungen werden ausgelesen und geprüft; andere Belege brauchen manuell erfasste Angaben. | `internal/service/receipt_service.go`, `internal/service/document_service.go`, `internal/einvoice/` |
| Buchen und korrigieren | Der Buchungskern prüft ausgeglichene Buchungen. Belegbuchung und Handbuchung sind vorhanden; eine Handbuchung verlangt einen Beleg oder Eigenbeleg. Korrekturen erfolgen durch Storno und Neubuchung. | `internal/service/journal_service.go`, `manual_entry.go`, `self_issued_receipt.go` |
| Bank und offene Rechnungen | CAMT.053-Import mit Originalnachweis und mehreren Euro-Bankkonten, Kontoeinrichtung beim ersten Import, Zuordnungsvorschläge, Teilzahlungen, Sammelzahlungen, Skonto und Zahlungsausfälle. CAMT-Sammeleinträge mit mehreren Einzeltransaktionen werden abgewiesen. Skonto korrigiert auch die Beträge der Voranmeldung. | `internal/bank/`, `internal/service/bank_service.go`, `payment_service.go`, `internal/accounting/ustva.go` |
| Rechnungen schreiben | Rechnungsnummer und Buchung entstehen zusammen. ZUGFeRD und XRechnung im CII-Format, Berichtigungen und Stornorechnungen sind vorhanden. | `internal/service/invoice_service.go`, `invoice_correction.go`, `internal/invoice/` |
| Anzahlungen | Rechnungsverbund mit Abschlägen, Zahlungseingängen und Schlussrechnung; geleistete Anzahlungen auf der Eingangsseite. | `internal/service/advance_service.go`, `posting_service.go` |
| Mahnen | Vorschläge aus überfälligen Posten, Mahnstufen und Berechnung von Zinsen und Pauschalen; Schreiben als Dokument. | `internal/service/dunning_service.go`, `internal/accounting/default_interest.go` |
| Umsatzsteuer | Voranmeldung mit Rückverfolgung zur Buchung, Berichtigung, Dauerfristverlängerung und Zusammenfassende Meldung. Übermittlungsdaten werden nach einer externen Abgabe von Hand erfasst. | `internal/service/vat_return_service.go`, `internal/accounting/ustva.go`, `zm.go` |
| Steuerliche Nachweise | Verzeichnis für Vorsteuerberichtigungen, Bestätigung von Umsatzsteuer-IDs und Liefernachweise. | `internal/service/input_tax_service.go`, `supply_evidence_service.go`, `internal/vatid/` |
| Anlagen | Anlagenkartei, Abschreibungen, Bewegungen, Abgänge und Anlagenspiegel. Steuerliche Sonderabschreibungen werden neben dem Handelswert geführt. | `internal/service/asset_service.go`, `internal/accounting/afa.go`, `afa_rules.json` |
| Abschluss vorbereiten | Rechnungsabgrenzung, Rückstellungen, Inventurwert, Umsatzsteuer-Verrechnung, Steuerrückstellung, gesetzliche UG-Rücklage im Abschlussjahr und anschließende Ergebnisverwendung. | `internal/service/accrual_service.go`, `provision_service.go`, `appropriation_service.go`, `closing_steps_service.go` |
| Auswerten | Journal, Konten, Bilanz und Gewinn- und Verlustrechnung mit Vorjahr. Anhangtexte, Rückstellungsspiegel und Überleitung werden zusammengestellt und als Teil der Ausgaben berücksichtigt. | `internal/service/statement_service.go`, `statement_export.go`, `internal/accounting/statement.go` |
| E-Bilanz exportieren | Vorläufige XBRL-Datei aus der Bilanzgliederung. Die Taxonomie-Zuordnung ist ungeprüft; fehlende Zuordnungen nach teilweiser Ergebnisverwendung verhindern den Export. | `internal/ebilanz/ebilanz.go`, `taxonomy_6.9.json` |
| Geschäftsjahr wechseln | Jahresanlage, Saldenvortrag mit offenen Posten, Übernahme von Anhangtexten als Vorlage. | `internal/service/closing_service.go` |
| Gründung begleiten | Gründungsangaben, Kapitalaufbringung, Aufgaben und Nachweise, Eröffnungsbilanz. | `internal/service/foundation_opening.go`, `frontend/src/pages/GruendungPage.tsx` |
| Änderungen nachvollziehen | Verkettete Prüfwerte für Journal und Änderungsprotokoll, Prüfläufe und Festschreibung mit externem Zeitstempel. | `internal/accounting/journalhash.go`, `audithash.go`, `internal/service/check_service.go`, `internal/timestamp/` |
| Sichern und herausgeben | Sicherung, Wiederherstellung, Belegarchiv, Datenexport für Prüfungen und Verfahrensdokumentation. | `internal/service/backup_service.go`, `export_service.go`, `internal/export/`, `internal/procdoc/` |
| Hilfe lesen | Kurzer Tooltip am Fragezeichen. „Mehr erfahren“ öffnet einen Detaildialog; erkannte Gesetzesverweise sind verlinkt. | `frontend/src/components/ui/Help.tsx`, `LegalText.tsx` |

Dateinamen ohne Verzeichnis in einer Tabellenzelle beziehen sich auf das zuletzt
genannte Verzeichnis derselben Zelle.

## Bekannte Einschränkungen

### Jahresabschluss und Einreichung

- Alle Elemente der E-Bilanz-Ressource sind mit `verified: false` gekennzeichnet.
  Eine erzeugte Datei ist kein Nachweis für ein amtlich gültiges Format.
  Eine erprobte Einreichung ist nicht belegt.
- Buchfink übermittelt keine Meldungen selbst an die Finanzverwaltung.
  Das Kennziffernblatt für die Umsatzsteuer kann in Mein ELSTER übertragen werden.
  Ein beliebiger XBRL-Upload für die E-Bilanz in Mein ELSTER wird nicht zugesichert.
- Anhang und Ersatzangaben entstehen aus Freitexten und berechneten Tabellen.
  Kleinstgesellschaften erhalten die abgefragten Ersatzangaben unter der Bilanz;
  fehlende Pflichtabschnitte verhindern die Aufstellung. Ob alle Sachverhalte
  vollständig und richtig erklärt sind, wird nicht automatisch festgestellt.
  Weitere rechtsform- und größenabhängige Anhangpflichten bleiben unvollständig.
- Die Größenklasse wird berechnet. Die Befreiung von latenten Steuern wird noch
  nicht vollständig über diese Klasse gesteuert.
- Unterzeichneter Abschluss, Feststellungsbeschluss und Prüfungsvermerk sind noch
  nicht als vollständiges Abschlussdokument mit Nachweisen eingebunden.
  Die allgemeine Dokumentablage und ein Beschluss zur Ergebnisverwendung ersetzen
  diesen Ablauf nicht.
- Offenlegung, Hinterlegung und zugehörige Einreichungsnachweise sind nicht
  vollständig implementiert. Ein fertiger Datensatz für das Unternehmensregister fehlt.

### Laufende Buchhaltung

- Die unterstützten Steuerfälle sind begrenzt. Unter anderem fehlen eigene
  Kleinunternehmerbesteuerung, Istversteuerung, OSS/IOSS, Differenzbesteuerung
  und weitere Sonderfälle. Ein vorhandenes Feld im Meldeformular bedeutet
  nicht, dass alle zugehörigen Geschäftsvorgänge erfasst werden können.
- Unentgeltliche Wertabgaben, Einfuhrumsatzsteuer und die Option zur
  Umsatzsteuerpflicht sind im Katalog weiterhin nicht vollständig abgedeckt.
  Die frühere Aussage, Skontokorrekturen fehlten ebenfalls, ist überholt.
- Ein leeres Leistungsdatum wird beim Ausstellen einer Rechnung auf das
  Rechnungsdatum gesetzt, siehe `InvoiceService.Issue` in `invoice_service.go`.
- XRechnungen werden im CII-Format ausgestellt. UBL wird beim Empfang gelesen,
  aber nicht als Ausgangsformat erzeugt.
- Vorräte werden als bereits bewerteter Inventurwert übernommen. Buchfink
  ermittelt diesen Wert nicht durch eine Lagerverwaltung.

### Speicherung und Prüfung

- Es gibt keine Benutzerverwaltung oder Rollenverteilung. Der Prüfermodus
  sperrt Änderungen; er ersetzt keine Mehrbenutzerberechtigungen.
- Ausgewählte Datenbankfelder sind verschlüsselt, einschließlich Kontakt- und
  Zahlungspartnernamen, Positionsbeschreibungen und Protokolltexten mit
  Bestandsmigration. Die Datenbank als Ganzes, Metadaten und Originaldateien sind
  es nicht. Einzelheiten stehen im [Sicherheitskonzept](security-concept.md).
- Sicherungen enthalten Datenbank, Belege, Dokumente und die Schlüsseldatei.
  Sie benötigen ein gewähltes Sicherungsziel. Automatische Sicherungen laufen
  beim Start nach 24 Stunden und beim Beenden zusätzlich bei Änderungen. Die Wiederherstellungsdatei ist
  für den Verlust des Betriebssystem-Schlüssels getrennt aufzubewahren.
- Aufbewahrungsregeln sind eingebettete Ressourcen. Geänderte Regeln brauchen
  eine neue Programmfassung. Eine Aufbewahrungssperre gilt je Geschäftsjahr.
- Der Datenexport für Betriebsprüfungen ist vorhanden. Ein erfolgreiches
  Einlesen in eine externe Prüfsoftware ist nicht belegt.
- DATEV-Export und lokaler MCP-Server sind nicht implementiert.

## Was die Prüfungen aussagen

Code und automatisierte Tests belegen einzelne Abläufe. Sie belegen weder die
vollständige Gesetzeskonformität noch, dass Menschen ohne Vorkenntnisse alle
Arbeitsschritte verstehen. Dafür bleiben fachliche Prüfung und Erprobung mit
der Zielgruppe nötig. Die [Dokumentationsprüfung](dokumentationspruefung-2026-09-11.md)
nennt die überprüften Widersprüche und die Grenzen dieses Abgleichs.

Der [Bug-Hunt-Bericht vom 12. September 2026](bug-hunt-2026-09-12.md) dokumentiert
die durchgespielten Abläufe, Videos, die Checkliste der Korrekturen und verbleibende
Grenzen. Die reproduzierten Fehler bei Bankimport, Rechnungsempfänger,
Eröffnungsbilanz, UG-Rücklage, Unternehmensdokumenten und Einrichtung eines
Vorjahres sind behoben. Die Abnahmeproben sind reguläre Regressionstests.

Die erzeugten Rechnungsbeispiele wurden zusätzlich mit CII-Schema,
EN-16931-Regeln, KoSIT-XRechnung-Konfiguration und veraPDF geprüft. Erfolgreiche
Beispiele ersetzen keine Zertifizierung aller unterstützten Rechnungsfälle.
Die Prüfung einer alten Testbuchhaltung belegt die Verschlüsselungsmigration
bei erhaltener Buchungs- und Protokollkette. Eine neue Sicherung wurde in einen
separaten Ordner wiederhergestellt und erneut geprüft.
