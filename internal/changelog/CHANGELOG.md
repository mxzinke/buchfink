# Änderungshistorie

Diese Datei ist die Versionshistorie des Programms (UNV-06). Sie ist in das
Programm eingebettet, wird in der Prüfübersicht unter „Systemhistorie" gezeigt
und liegt dem Prüferpaket bei. Zu jeder Fassung steht hier, was sie für die
Buchführung ändert.

Das Format ist Teil der Schnittstelle: `internal/changelog` liest die
Überschrift als `## <Fassung> — <Datum>`, den Absatz darunter als
Zusammenfassung und jeden Strichpunkt als Änderung. Der Text einer Fassung ist
zugleich der Text ihrer Veröffentlichung.

## v0.1 — 2026-09-07

Erste Fassung. Buchfink führt die doppelte Buchführung einer Kapitalgesellschaft
vom Beleg bis zur E-Bilanz, auf dem eigenen Rechner und ohne Konto bei
irgendwem. Diese Fassung ist erprobbar, aber unerprobt: sie hatte noch keinen
Anwender und keinen Betriebsprüfer.

- Doppelte Buchführung auf dem SKR04: Journal mit Soll und Haben, den
  Buchungssatz und die Steuer rechnet Buchfink aus dem Beleg, der Rechnung oder
  der zugeordneten Zahlung. Korrigiert wird durch Storno mit Verweis auf die
  Neubuchung, nie durch Überschreiben.
- Geschäftsjahr als Entität mit Rumpfjahr, Abschlussstand und Saldenvortrag: der
  Vortrag bringt Bestandskonten und offene Posten ins Folgejahr und weist eine
  Differenz aus, statt sie zu verstecken.
- Belege: unveränderliche Ablage mit Prüfsumme, Kopfdaten am Beleg, Eigenbeleg
  mit erzeugtem PDF, Belegpflicht auch bei Handbuchungen und ein einstellbares
  Belegnummernformat.
- Ausgangsrechnungen mit eigenem Layout, ausgegeben als ZUGFeRD-konformes
  PDF/A-3 oder als XRechnung im CII-Profil. Nummer, Datensatz und Buchung
  entstehen in einer Transaktion; die Pflichtangaben des § 14 Abs. 4 UStG werden
  vor der Nummernvergabe geprüft. Dazu Storno- und Korrekturbeleg,
  Kleinbetragsrechnung und Anzahlungen als Rechnungsverbund mit Schlussrechnung.
- Eingehende E-Rechnungen: ZUGFeRD, Factur-X und XRechnung werden erkannt, CII
  und UBL gelesen und gegen das Regelwerk der Norm geprüft. Aus dem Datensatz
  entsteht ein Buchungsvorschlag, der Beleg bleibt, wie er ankam.
- Bankauszüge im Format CAMT.053. Zu jedem Umsatz schlägt Buchfink den offenen
  Posten vor — nach Betrag, Verwendungszweck, Rechnungsnummer und Datumsnähe,
  mit Sammelzahlung und gelernten Regeln. Gebucht wird nach Bestätigung.
- Offene Posten zum Stichtag mit Altersstruktur, Zahlungsausgleich mit
  Teilzahlung und Skonto, Ausbuchung uneinbringlicher Posten und ein Mahnwesen
  mit Stufen, taggenauen Verzugszinsen nach § 288 BGB und abgelegtem
  Mahnschreiben.
- Anlagevermögen mit Verzeichnis und Kartei: Wertgrenzen für geringwertige
  Wirtschaftsgüter und Sammelposten, lineare und degressive Abschreibung mit
  automatischem Übergang, Sonderabschreibung nach § 7g EStG, außerplanmäßige
  Abschreibung und Zuschreibung, Abgang mit Buchgewinn oder -verlust und der
  Anlagenspiegel nach § 284 Abs. 3 HGB. Die Sätze liegen als datierte Ressource
  neben dem Code.
- Jahresabschluss als geführter Weg: Rechnungsabgrenzung, Rückstellungen mit
  Abzinsung, Inventurwert der Vorräte, Umsatzsteuer-Verrechnung,
  Steuerrückstellung, Abschluss der Erfolgskonten, Ergebnisverwendung mit
  Beschluss, Verzeichnis der steuerlichen Wahlrechte und Überleitungsrechnung
  zur Steuerbilanz.
- Auswertungen aus denselben Kontensalden: Kontenblatt, Summen- und Saldenliste
  zu jedem Stichtag, Journal mit Volltextsuche über Jahresgrenzen, Bilanz und
  Gewinn- und Verlustrechnung nach §§ 266, 275 HGB mit Vorjahresspalte,
  Größenklasse und Anhang — als PDF und CSV. Die E-Bilanz entsteht als XBRL aus
  derselben Gliederung.
- Umsatzsteuer: Voranmeldung mit allen Kennziffern des Vordrucks USt 1 A und
  Drill-down bis zur Buchung, Berichtigung, Dauerfristverlängerung,
  Zusammenfassende Meldung nach § 18a UStG und ein Übermittlungsprotokoll, das
  nach der Erfassung unveränderlich ist.
- Steuerliche Nebenpflichten: Vorsteuerabzug an der geprüften Rechnung,
  Aufteilungsmaßstab nach § 15 Abs. 4 UStG, Verzeichnis der Vorsteuerberichtigung
  nach § 15a UStG, qualifizierte Bestätigung der USt-IdNr. beim Bundeszentralamt
  vor der steuerfreien Lieferung, Belegnachweis nach §§ 17a, 17b UStDV.
- Unveränderbarkeit: jede Buchung hängt an der Hash-Kette ihrer Vorgängerin, das
  Änderungsprotokoll führt Vorher und Nachher in einer eigenen Kette, und die
  Festschreibung eines Zeitraums wird von einem unabhängigen Zeitstempeldienst
  nach RFC 3161 beglaubigt. Vor jeder Festschreibung läuft ein Prüfbericht;
  blockierende Befunde halten sie auf.
- Aufbewahrung und Betriebsprüfung: Fristen je Belegart mit frühestem
  Löschdatum und Aussetzung, Löschung eines abgelaufenen Jahres erst nach
  Archivexport, Datenüberlassung als Z3-Export nach dem Beschreibungsstandard,
  Prüferpaket in einem Ordner, schreibgeschützter Prüfermodus und eine
  Verfahrensdokumentation, die aus dem laufenden System entsteht.
- Sicherung als offene ZIP-Datei mit Datenbank, Belegen, Dokumenten und
  Schlüsseldatei — täglich und beim Beenden, mit Wiederherstellungstest und
  anschließender Integritätsprüfung.
- Gründung einer Kapitalgesellschaft im Einrichtungsassistenten: Stammkapital
  und Gesellschafter, Prüfung der Kapitalaufbringung vor der Anmeldung zum
  Handelsregister, laufende Rechnung der Unterbilanzhaftung, Gründungsbuchungen
  als Vorschlag und die Fristen von der Gewerbeanmeldung bis zum
  Transparenzregister.
- Betrieb: die Daten liegen als SQLite-Datei je Mandant auf dem eigenen Rechner,
  personenbezogene und geschäftliche Felder mit AES-256-GCM verschlüsselt und
  dem Schlüssel im Schlüsselbund des Betriebssystems. Die Startseite ist eine
  Aufgabenliste, die sagt, was heute zu tun ist.
- Nicht enthalten: keine Einnahmen-Überschuss-Rechnung, keine Übermittlung ans
  Finanzamt (kein ERiC), kein DATEV-Buchungsstapel, kein Lohn, keine Kasse, kein
  Lager, kein Rollenmodell. Die Elementnamen der E-Bilanz-Taxonomie sind vor der
  ersten Übermittlung gegen die amtliche Fassung abzugleichen.
