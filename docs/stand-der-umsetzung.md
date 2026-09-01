# Buchfink – Stand der Umsetzung

Status: laufend gepflegt
Letzte Aktualisierung: 2026-09-01

Dieses Dokument beschreibt, was Buchfink heute tut, wo eine Funktion an einer
Grenze endet und was noch fehlt. Es ist die Gegenprobe zum README: dort steht,
wofür Buchfink gedacht ist, hier steht, was davon im Code angekommen ist. Jede
Angabe unten nennt die Stelle, an der sie nachzulesen ist.

Wer eine Funktion vermisst, findet sie entweder in Abschnitt 4 oder gar nicht –
und wenn sie dort fehlt, ist sie noch nie bedacht worden.

## 1. Was trägt

| Bereich | Stand | Wo |
|---|---|---|
| Buchungskern mit Steuerautomatik | Jede Zeile trägt Steuerschlüssel, Bemessungsgrundlage und Steuerfall. Konten der Klasse 8 werden abgelehnt. | `internal/service/posting_service.go`, `internal/accounting/tax_skr04.go` |
| Hash-Chain über das Journal | SHA256 je Buchung über die Vorgängerbuchung, in der Oberfläche prüfbar. | `internal/accounting/journalhash.go`, `internal/service/journal_service.go:218` |
| Storno als einzige Korrektur | Generalumkehr mit negativen Beträgen auf derselben Seite, kein Seitentausch. | `internal/service/journal_service.go:133` |
| Belegablage | Ablage unter dem eigenen SHA256, Deduplizierung, lückenlose Belegnummern. | `internal/receiptstore/store.go`, `internal/domain/numberrange.go` |
| E-Rechnungs-Empfang | ZUGFeRD, Factur-X, XRechnung, CII und UBL werden erkannt, gelesen und gegen das Regelwerk geprüft. Der ausgereifteste Teil des Backends. | `internal/einvoice/` |
| Rechnungsausstellung | ZUGFeRD-PDF/A-3 über Typst. | `internal/invoice/`, `internal/einvoice/zugferd/` |
| Offene Posten und Zahlungsausgleich | Personenkonten, Teilzahlungen, Skonto, Differenzgründe. | `internal/service/payment_service.go` |
| Anlagenbuchhaltung | Wertgrenzen, lineare und degressive AfA, § 7g, § 7a Abs. 9, außerplanmäßige Abschreibung, Zuschreibung, Abgang, Anlagenspiegel, Darlehen, Investmentanteile. | `internal/service/asset_service.go`, `internal/accounting/afa.go` |
| Festschreibung | Zeitraum-Festschreibung mit RFC-3161-Zeitstempel über den Kettenkopf, Nachholung bei fehlendem Netz. | `internal/domain/festschreibung.go`, `internal/timestamp/tsa.go` |
| Audit-Log | Änderungsprotokoll über Buchungen, Belege und Stammdaten. | `internal/service/audit_service.go` |
| Verschlüsselung at rest | 31 Datenbankfelder mit personenbezogenem oder geschäftlichem Inhalt, AES-256-GCM, Schlüssel im Betriebssystem-Schlüsselbund, Wiederherstellungsdatei. | `internal/repository/encryption.go`, `internal/security/` |
| Mandanten | Mehrere Unternehmen nebeneinander, je eigener Datenordner und eigener Schlüssel. | `internal/domain/app_config.go` |
| Gründung einer Kapitalgesellschaft | Erfassung im Einrichtungsassistenten, Kapitalaufbringung nach § 7 Abs. 2 GmbHG bzw. § 5a Abs. 2 GmbHG und § 36a AktG, Unterbilanzrechnung auf den Eintragungstag, Gründungsbuchungen und die Fristen der Gründung. | `internal/accounting/gruendung.go`, `internal/service/foundation_service.go` |

## 2. Wo eine Funktion an ihrer Grenze endet

**Umsatzsteuer ist eine Auswertung, keine Voranmeldung.** Der `VatService`
aggregiert die Steuerzeilen des Journals nach Zeitraum. Die Oberfläche zeigt
daraus vier Kennziffern des amtlichen Vordrucks: 81, 86, 66 und 83
(`frontend/src/pages/ReportsPage.tsx:346`). Alles andere fehlt, und eine
Feinheit ist dabei falsch: die abziehbare Vorsteuer läuft vollständig in
Kennziffer 66, auch soweit sie aus § 13b UStG oder aus innergemeinschaftlichem
Erwerb stammt. Dort gehören die Kennziffern 67 und 61 hin. Der Service sagt
seine Grenze im Kommentar selbst (`internal/service/vat_service.go:18`).

**Bilanz und GuV sind Kontensummen, keine Gliederung.** Beide entstehen im
Frontend durch Filtern nach Kontenklasse und Bilanzseite
(`frontend/src/pages/ReportsPage.tsx:93`). Es gibt keine Gliederung nach § 266
und § 275 HGB, keine Größenklassen nach § 267 HGB, keine Vorjahresspalte,
keinen Bilanzgewinn und keine Ausgabe als Datei oder Druck.

**Die E-Bilanz ist ein Gerüst, kein fertiger Export.** Die Zuordnungstabelle
umfasst rund fünfzig SKR04-Konten, alles andere landet auf
`de-gaap-ci:bs.other` (`internal/ebilanz/ebilanz.go:196`). Aus dem GAAP-Modul
werden drei Werte geschrieben: `is.netSales`, `is.operatingExpenses` und
`is.netIncome`. Eine Bilanz steht nicht in der Instanz. Die Taxonomie ist auf
`2023-04-14` festverdrahtet, die Datei entsteht per `fmt.Sprintf` ohne Prüfung
gegen Schema oder Mussfelder. Der Kontennachweis und der Anlagenspiegel sind
dagegen echt und vollständig. Vor einer Übermittlung ist die Instanz von Hand
zu prüfen.

**Der Bankabgleich schlägt nichts vor.** Importiert wird CAMT.053 aus einer
Datei. Einen Abgleich nach Betrag, Verwendungszweck, Rechnungsnummer oder
Datumsnähe gibt es nicht: die Oberfläche filtert die offenen Posten nach dem
Vorzeichen des Umsatzes und zeigt den Rest als Liste
(`frontend/src/pages/BankImportPage.tsx:242`). Die Zuordnung trifft der Nutzer.

**Fremdwährung ist zur Hälfte verdrahtet.** Kurs, Kursquelle und Kursdatum
hängen an der Buchung und gehen in den Hash ein
(`internal/accounting/journalhash.go:58`), und die Bewertung von Finanzanlagen
nach § 256a HGB rechnet mit ihnen. Der EZB-Abruf existiert als Service
(`internal/currency/ecb.go`), wird aber von keiner Bridge-Methode aufgerufen:
Kurse kommen heute nur von Hand herein.

**Die Fristenseite ist zur Hälfte eine Merkliste.** Die Steuertermine berechnet
das Frontend aus dem Voranmeldungszeitraum, der Haken liegt im `localStorage` des
Browsers (`frontend/src/pages/DeadlinesPage.tsx`), nicht in der Datenbank und
nicht im Audit-Log. Für die Gründungspflichten gilt das nicht mehr: Sie werden
mit ihrem Datum als `FoundationTask` gespeichert. Die Festschreibung auf
derselben Seite ist ebenfalls echt.

## 3. Was fehlt und den Jahreslauf blockiert

**Jahreswechsel, Saldenvortrag, Eröffnungsbilanz.** Das ist die größte Lücke.
Kontensalden entstehen ausschließlich aus den Buchungen des aktiven
Geschäftsjahres: `AccountTurnovers` filtert auf `e.fiscal_year`
(`internal/repository/journal_gorm.go:247`), und `GetAccounts` faltet nur
dieses Ergebnis in den Kontenplan (`internal/service/accounting_service.go:59`).
Ein Bestandskonto zeigt im zweiten Jahr deshalb nur die Bewegung dieses Jahres,
nicht seinen Bestand. Die Bausteine liegen bereit und sind unbenutzt:

- Die Vortragskonten 9000, 9008 und 9009 stehen als Konstanten
  (`internal/domain/skr04_accounts.go`) und werden nie bebucht.
- `CreateFiscalYear` legt kein Jahr an, sondern schaltet nur den Jahresfilter um
  (`internal/wailsbridge/app_service.go`). Der Name führt in die Irre.
- Ein Erfassungsweg für Eröffnungswerte beim Einrichten eines Mandanten fehlt
  für den Umsteiger mit laufender Buchhaltung weiterhin. Der Gründungsfall ist
  seit dem Gründungsweg abgedeckt: `EntrySourceOpening` wird dort erzeugt
  (`internal/service/foundation_service.go`), und die Kapitalkonten 2900 und 1298
  werden bebucht.

**Abschlussbuchungen.** `EntrySourceClosing` setzt allein der
Abschreibungslauf (`internal/service/asset_service.go:1156`). Es fehlen der
Abschluss der Erfolgskonten auf das Eigenkapital, die Verrechnung von
Umsatzsteuer und Vorsteuer auf die Zahllast, die Steuerrückstellung und die
Ergebnisverwendung.

**Rechnungsabgrenzung und Rückstellungen.** ARAP und PRAP sind beschrieben und
nicht gebaut ([anforderung-rechnungsabgrenzung.md](anforderung-rechnungsabgrenzung.md)).
Rückstellungen kommen im Code nur als Kontobezeichnung vor, es gibt keinen Weg
für Bildung, Auflösung oder Verbrauch.

**Anzahlungen.** Beschrieben und nicht gebaut
([anforderung-anzahlungen.md](anforderung-anzahlungen.md)). Ohne die Verrechnung
berechneter Anzahlungen in der Schlussrechnung droht die doppelt ausgewiesene
und nach § 14c Abs. 1 UStG geschuldete Steuer.

**DATEV-Export** ([anforderung-datev-export.md](anforderung-datev-export.md))
und **Datenträgerüberlassung nach Z3** (GoBD-Datenzugriff, Beschreibungsstandard
mit `index.xml`). Der Z3-Export ist bisher nirgends beschrieben und wird in
jeder Betriebsprüfung verlangt.

**Datensicherung.** Es gibt keinen Sicherungs- oder Rückspielweg für Datenbank
und Belegordner. Bei einer Anwendung, die alles auf einem Rechner hält, ist das
das größte Betriebsrisiko.

## 4. Was sonst noch fehlt

Steuerliche Meldungen: Umsatzsteuer-Voranmeldung als vollständiger Vordruck mit
Übermittlungsdatei, Dauerfristverlängerung mit Sondervorauszahlung,
Zusammenfassende Meldung nach § 18a UStG, Umsatzsteuer-Jahreserklärung,
qualifizierte Bestätigungsabfrage der USt-IdNr. beim BZSt, Einfuhrumsatzsteuer
aus dem Zollbescheid (`internal/domain/einvoice.go:194`).

Zahlungsverkehr und offene Posten: Mahnwesen mit Stufen, Fristen, Gebühren und
Verzugszinsen, stichtagsbezogene OP-Liste
(`internal/service/payment_service.go:107`), SEPA-Überweisungsträger nach
pain.001, Lastschriftmandate (`internal/domain/contact.go:66`),
Zahlungsvorschlag nach Fälligkeit und Skontofrist.

Bank: CAMT.052 und CAMT.054, MT940, CSV
(`internal/domain/bank.go:44`), Abruf über FinTS oder EBICS, Kreditkarten- und
Zahlungsdienstleisterkonten.

Belege und Buchungen: Kassenbuch mit Kassenbericht und täglicher Aufzeichnung,
wiederkehrende Buchungen und Buchungsvorlagen, Stapelerfassung, Kostenstellen.

Ausgangsrechnungen: XRechnung als reines XML ohne PDF
(`internal/domain/invoice.go:91`), Gutschrift und Stornorechnung als eigene
Buchungswege (`internal/domain/einvoice.go:96`), Abschlags- und Teilrechnungen,
Zahlungsbedingungen mit Skonto in der Rechnung selbst
(`internal/domain/invoice.go:90`), Steuerfall je Rechnungsposition statt je
Rechnung (`internal/service/einvoice_service.go:339`), Versand per E-Mail oder
Peppol, Angebot und Lieferschein.

Rechtsform und Eigenkapital: Kapitalkonten der Gesellschafter, Privatentnahmen
und Privateinlagen, Gesellschafterverrechnungskonten. Der Gründungsweg kennt die
Gesellschafter einer Kapitalgesellschaft und ihre Einlagen, führt für sie aber
kein laufendes Kapitalkonto. Das trifft die
Personenhandelsgesellschaften, die das README ausdrücklich als Zielgruppe
nennt. Dazu die Rückstellungsrechnung für Körperschaft- und Gewerbesteuer.

Lohn: die Lohnbuchhaltung selbst bleibt außen vor, der Import eines
Lohnjournals mit den Verbindlichkeiten gegenüber Personal und
Sozialversicherungsträgern fehlt aber und wird gebraucht, sobald ein Unternehmen
Angestellte hat.

Weiteres: Verfahrensdokumentation. Das README hat sie lange als Funktion
geführt, erzeugt oder verknüpft wird nichts. Wechselkurse ohne Netz
(`internal/domain/currency.go:16`). Eine durchgehende Prüfung im
Dauerbetrieb: es gibt kein `.github/`, die Tests laufen nur von Hand über
`task test`.

## 5. Vorgeschlagene Reihenfolge

1. Jahreswechsel mit Saldenvortrag und die Abschlussbuchungen. Ohne beides
   endet die Buchhaltung nach einem Geschäftsjahr.
2. Datensicherung. Wenig Aufwand, größter Schadensfall.
3. Umsatzsteuer-Voranmeldung als vollständiger Vordruck. Die Daten hängen
   bereits an den Buchungszeilen, es fehlen Formular und Ausgabe.
4. Bilanz und GuV nach HGB-Gliederung im Backend, mit Ausgabe als Datei. Das
   trägt die E-Bilanz gleich mit, weil erst dann eine gegliederte Struktur
   existiert, auf die das XBRL-Mapping zeigen kann.
5. Abgleichvorschläge im Bankimport. Daran hängt das Versprechen „Buchung folgt
   Bankumsatz".
6. Danach Rechnungsabgrenzung und Rückstellungen, Anzahlungen, DATEV- und
   Z3-Export, Mahnwesen.
