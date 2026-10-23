# Änderungshistorie

Diese Datei ist die Versionshistorie des Programms (UNV-06). Sie ist in das
Programm eingebettet, wird in der Prüfübersicht unter „Systemhistorie" gezeigt
und liegt dem Prüferpaket bei. Zu jeder Fassung steht hier, was sie für die
Buchführung ändert.

Das Format ist Teil der Schnittstelle: `internal/changelog` liest die
Überschrift als `## <Fassung> — <Datum>`, den Absatz darunter als
Zusammenfassung und jeden Strichpunkt als Änderung. Der Text einer Fassung ist
zugleich der Text ihrer Veröffentlichung.

## v0.3.0 — 2026-10-17

Ab dieser Fassung führt ein KI-Agent die Bücher über MCP; die App zeigt den
Stand und nimmt die Freigaben an. Claude Code, Claude Desktop, OpenCode, Codex
und Pi verbinden sich mit der laufenden App. Buchfink führt nur noch GmbH und
UG. Mandanten aus v0.1 und v0.2 öffnet diese Fassung nicht, auch nicht aus
einer Sicherung; ein Unternehmen wird neu angelegt.

- Der MCP-Server läuft mit der App auf 127.0.0.1, Port 7465. Apps melden sich
  über den OAuth-Server von Buchfink an und werden im Fenster von Buchfink je
  Mandant zugelassen, mit den Rechten Lesen oder Buchen. Zulassung und Entzug
  stehen im Änderungsprotokoll. Für Claude Desktop gibt es die Brücke
  buchfink-mcp als Erweiterung (.mcpb).
- App und Agent schreiben über denselben Kern. Jeder Schreibvorgang läuft mit
  seinem Protokolleintrag in einer Transaktion; mit einem Idempotenzschlüssel
  bucht eine Wiederholung nicht doppelt. Jede Buchung speichert den
  Erfassungsweg: App, Hintergrundlauf oder verbundene App. Festschreibung,
  Übermittlungsvermerk, Aufstellung, Feststellung, Offenlegung,
  Pflichtbestätigungen, Unternehmensdaten und Prüfermodus nimmt Buchfink nur in
  der App an. Was sich per Storno korrigieren lässt, bucht ein verbundener
  Agent nach Zustimmung im Chat auf jedem Konto, auch Darlehensverträge mit
  Gesellschaftern, die Ergebnisverwendung nach der Feststellung und die
  Organisationstexte der Verfahrensdokumentation.
- Belege, E-Rechnungen und Kontoauszüge kommen in einen Eingang je
  Unternehmen, voreingestellt im Datenordner, auf Wunsch auch in einem
  Cloud-Ordner. Dateien lassen sich ins Fenster ziehen. Der Agent legt sie ab,
  wenn der Mensch es sagt; danach verschiebt Buchfink sie in den Unterordner
  Übernommen. Liegt der Datenordner in einem synchronisierten Ordner, warnt die
  Einrichtung.
- Der Pflichtenkalender führt 21 Pflichtarten einer GmbH mit Frist und
  Nachweis. Die Fristen richten sich nach Bundesland, Dauerfristverlängerung
  und steuerlicher Beratung.
- KSt 1, GewSt 1 A und USt 2 A stellt Buchfink formularnah als Eingabehilfe für
  Mein ELSTER zusammen, mit Herkunft je Wert und Export als JSON. Der
  Zeilenkatalog gilt für das Vordruckjahr 2025. Die Zeilen von USt 2 A sind am
  amtlichen Vordruck geprüft, die von KSt und GewSt nicht.
- Die E-Bilanz entsteht als XBRL nach der amtlichen Taxonomie 6.9 oder 6.10,
  mit Kontennachweis und Überleitung. Übermittelt wird sie über einen Dienst
  wie ebilanzplus.de oder ebilanzonline.de. Ersatzweise schreibt Buchfink eine
  Summen- und Saldenliste, die eBilanz+ einliest.
- Neue Buchungswege: Bankumsatz aufteilen, Firmenkreditkarte, Auslagen,
  Gesellschafterdarlehen mit Zinsen, Lohnjournal als Summenbuchung. Bei Zahlung
  vor der Leistung rechnet Buchfink die Umsatzsteuer dem Zeitraum der Zahlung
  zu (Mindest-Istversteuerung). Die Voranmeldung folgt dem Vordruck 2026.
- Neue Abschlussbausteine: Wertberichtigung auf Forderungen, unfertige
  Leistungen und eine Steuerrückstellung mit dem Körperschaftsteuersatz des
  Jahres. Abschlussbuchungen gehören zu einer Abschlussperiode zwischen
  Dezember- und Jahresfestschreibung.
- Entfallen sind Verschlüsselung, Warenlieferungen ins Ausland samt
  Belegnachweis, Vorräte, Kasse, Mahnschreiben, Dokumentenablage, die
  Gründungsoberfläche und andere Rechtsformen; die buchhalterischen
  Gründungsschritte bleiben. Den Schutz am Rechner übernimmt die
  Festplattenverschlüsselung des Betriebssystems.
- Die Veröffentlichung enthält die App für macOS, Windows und Linux, die Brücke
  buchfink-mcp, die Erweiterung für Claude Desktop, die Skill-Pakete und
  Prüfsummen. Nichts davon ist signiert. Buchfink übermittelt nichts selbst an
  die Finanzverwaltung. Diese Vorschau ist keine Freigabe für den produktiven
  Einsatz.

## v0.2 — 2026-09-13

Diese Vorabversion verbessert Einrichtung, Bankimport und Jahresabschluss.
Sie enthält die Korrekturen aus den Prüfungen mit kleinen
Dienstleistungsunternehmen. Die Veröffentlichung enthält den Quellcode;
zum Starten werden weiterhin Entwicklungswerkzeuge benötigt.

- Ein eigener Gründungsweg führt durch Aufgaben und Nachweise.
  Unternehmensunterlagen lassen sich ablegen und im Prüferpaket ausgeben.
  Eröffnungsbilanzen werden auf Vollständigkeit und Übereinstimmung mit den
  Gründungsbuchungen geprüft.
- Das Onboarding fordert zur Sicherung des Wiederherstellungsschlüssels auf.
  Ein abgebrochener Export zählt nicht als Sicherung; erneuter Export macht
  ältere Wiederherstellungsdateien nicht ungültig. Belegte Datenordner werden
  vor der Schlüsselanlage abgewiesen, und das gewählte Geschäftsjahr bleibt aktiv.
- Bankkonten werden beim ersten CAMT-Import eingerichtet. Mehrere Euro-Konten
  werden getrennt zugeordnet. Originaldateien werden als Nachweis abgelegt,
  Wiederholungsimporte erkannt und ungültige Beträge oder Buchungsdaten
  abgewiesen. Nicht unterstützte Sammeleinträge erzeugen eine Fehlermeldung.
- Betragseingaben erkennen ungültige Tausendergruppen und Überläufe.
  Teilzahlungen schlagen höchstens den verfügbaren Zahlungsbetrag vor.
  Kontoauszüge erhalten keine Aufforderung zur Prüfung von Rechnungsangaben.
- Rechnungen enthalten die vollständige Empfängerfirma sowie ergänzte
  Verkäufer- und Registerangaben. XRechnungen erhalten die fehlende
  Geschäftsprozesskennung. Die geprüften ZUGFeRD- und XRechnungsbeispiele
  bestanden unabhängige Formatprüfungen.
- Die gesetzliche UG-Rücklage wird im Abschlussjahr berücksichtigt.
  Bilanzgewinn und Vortrag verwenden das verbleibende Ergebnis.
  Kleinstgesellschaften werden vor der Aufstellung nach Ersatzangaben gefragt;
  ein Offenlegungsvermerk verlangt einen Nachweisbezug.
- Kontakt- und Zahlungspartnernamen, Rechnungspositionen, Kontobezeichnungen
  und Protokolltexte werden verschlüsselt. Die Migration auf Datenbankschema 10
  erhält Buchungen und Hashketten. Sicherung und Wiederherstellung wurden mit
  einem migrierten Datenbestand geprüft. Vor dem Upgrade eine Sicherung und
  den Wiederherstellungsschlüssel getrennt aufbewahren.
- Leere Auswertungen führen nicht mehr zu den reproduzierten Abstürzen in
  Bilanz und E-Bilanz. Unvollständige E-Bilanz-Zuordnungen nach teilweiser
  Ergebnisverwendung verhindern den Export.
- Kurze Tooltips und Detaildialoge mit Quellenlinks erklären die Bedienung.
  Nummernformate und Lückenprüfung liegen standardmäßig eingeklappt unter
  Einstellungen → Rechnungsstellung → Nummernkreise.
- Dokumentation wird nach Thema gepflegt. Umsetzungsstand, bekannte Grenzen,
  Prüfszenarien und Roadmap ersetzen separate Berichte je Arbeitsschritt.
- Die E-Bilanz-Zuordnung bleibt ungeprüft; eine amtliche Übermittlung und
  vollständige Abschlussunterlagen sind weiterhin nicht abgedeckt.
  Originaldateien und Sicherungsarchive sind nicht verschlüsselt.
  Diese Vorabversion ist keine Freigabe für den produktiven Einsatz.

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
