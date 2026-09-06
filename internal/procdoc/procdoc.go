// Package procdoc erzeugt die Verfahrensdokumentation aus dem System.
//
// Sie muss nach GoBD Rz. 151 vorliegen, und sie ist der Teil, den kleine
// Unternehmen am zuverlässigsten nicht haben: ein Dokument, das jemand einmal
// schreibt und danach nie wieder anfasst, beschreibt nach dem ersten Update ein
// Programm, das es nicht mehr gibt. Deshalb wird sie erzeugt und nicht
// geschrieben — die statischen Textbausteine liegen als Vorlage in diesem
// Paket, die veränderlichen Angaben kommen aus der Datenbank und aus dem Code,
// der tatsächlich rechnet.
//
// Was das Paket nicht kann, sagt es selbst: die unternehmensindividuellen
// Teile — wer scannt, wer prüft, wer vertritt — kann kein Programm wissen. Sie
// kommen als Freitext aus den Einstellungen und sind mit Mustern vorbelegt.
package procdoc

import (
	"fmt"
	"strings"
	"text/template"
	"time"
)

// Input sind die veränderlichen Angaben der Verfahrensdokumentation.
//
// Ein einfaches Datenpaket und kein Zugriff auf Repositories: so lässt sich das
// Ergebnis prüfen, ohne eine Datenbank aufzusetzen, und die Frage „steht das
// wirklich drin" ist eine Prüfung und keine Sichtkontrolle.
type Input struct {
	// Teil 1: das Unternehmen.
	CompanyName    string
	LegalForm      string
	Street         string
	ZipCity        string
	Country        string
	TaxNumber      string
	VatID          string
	TaxOffice      string
	FiscalYear     int
	FiscalYearFrom string
	FiscalYearTo   string
	DataDir        string
	// CloudWarning ist belegt, wenn der Datenordner in einem
	// Synchronisationsordner liegt.
	CloudWarning string
	SKR          string
	// TaxCases sind die abgedeckten Steuerfälle, ExcludedCases die
	// ausdrücklich nicht abgedeckten. Beide gehören hinein: eine
	// Verfahrensdokumentation, die nur sagt, was geht, verschweigt die Grenze
	// des Verfahrens.
	TaxCases      []string
	ExcludedCases []string

	// Teil 2: die Anwenderdokumentation.
	NumberRanges  []NumberRange
	CaptureDays   int
	TaxationType  string
	VatPeriod     string
	Organisation  Organisation
	LegalFormNote string

	// Teil 3: die technische Systemdokumentation.
	AppVersion    string
	RuleVersion   string
	TSAName       string
	ExportFormats []string
	RetentionRows []RetentionRow

	// Teil 4: der Betrieb.
	BackupDir      string
	BackupRhythm   string
	BackupRuns     []BackupRun
	Migrations     []Migration
	ChangelogTable string
	CheckRules     []CheckRule

	// Kopf.
	Version          string
	CreatedAt        time.Time
	Actor            string
	SystemChangeDate string
}

// NumberRange ist ein Nummernkreis mit seiner Systematik.
type NumberRange struct {
	Name   string
	Format string
	Next   string
	Scope  string
}

// Organisation sind die unternehmensindividuellen Freitexte.
type Organisation struct {
	Responsibilities string
	ReceiptFlow      string
	Scanning         string
	Approval         string
	Substitution     string
	Backup           string
	Notes            string
}

// RetentionRow ist eine Zeile des Löschkonzepts.
type RetentionRow struct {
	Category   string
	Class      string
	Years      int
	LegalBasis string
	Note       string
}

// BackupRun ist ein Sicherungslauf in Kurzform.
type BackupRun struct {
	At      string
	Result  string
	Target  string
	SizeKiB int64
}

// Migration ist ein Eintrag des Schemaprotokolls.
type Migration struct {
	At          string
	AppVersion  string
	FromVersion int
	ToVersion   int
	Result      string
}

// CheckRule ist eine Regel des internen Kontrollsystems.
type CheckRule struct {
	Key      string
	Severity string
	Purpose  string
}

// FileName ist der Dateiname der erzeugten Fassung.
func FileName(company, version string) string {
	slug := slugify(company)
	if slug == "" {
		slug = "mandant"
	}
	return fmt.Sprintf("Verfahrensdokumentation-%s-%s.md", slug, version)
}

func slugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case r == 'ä' || r == 'Ä':
			b.WriteString("ae")
			prevDash = false
		case r == 'ö' || r == 'Ö':
			b.WriteString("oe")
			prevDash = false
		case r == 'ü' || r == 'Ü':
			b.WriteString("ue")
			prevDash = false
		case r == 'ß':
			b.WriteString("ss")
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// Render erzeugt die Verfahrensdokumentation als Markdown.
func Render(in Input) (string, error) {
	tmpl, err := template.New("procdoc").Funcs(template.FuncMap{
		"orNone": func(s string) string {
			if strings.TrimSpace(s) == "" {
				return "— nicht erfasst —"
			}
			return s
		},
		"date": func(t time.Time) string { return t.UTC().Format("02.01.2006 15:04 UTC") },
	}).Parse(documentTemplate)
	if err != nil {
		return "", fmt.Errorf("die Vorlage der Verfahrensdokumentation ist fehlerhaft: %w", err)
	}
	var out strings.Builder
	if err := tmpl.Execute(&out, in); err != nil {
		return "", fmt.Errorf("die Verfahrensdokumentation konnte nicht erzeugt werden: %w", err)
	}
	return out.String(), nil
}

// Die vier Überschriften, an denen sich die Vollständigkeit prüfen lässt. Sie
// stehen als Konstanten, weil der Test sie sonst als Zeichenkette wiederholen
// müsste — und dann prüfte er die Kopie und nicht das Dokument.
const (
	HeadingGeneral    = "## 1. Allgemeine Beschreibung"
	HeadingUser       = "## 2. Anwenderdokumentation"
	HeadingTechnical  = "## 3. Technische Systemdokumentation"
	HeadingOperations = "## 4. Betriebsdokumentation"
	HeadingControls   = "## 5. Internes Kontrollsystem"
)

const documentTemplate = `# Verfahrensdokumentation

**Unternehmen:** {{orNone .CompanyName}}
**Fassung:** {{.Version}}
**Erzeugt am:** {{date .CreatedAt}} von {{orNone .Actor}}
**Programmfassung:** buchfink {{.AppVersion}}, Regelstand {{.RuleVersion}}

Diese Verfahrensdokumentation ist aus dem laufenden System erzeugt. Die
Angaben zu Unternehmen, Nummernkreisen, Fristen und Prüfregeln stammen aus der
Datenbank und aus dem Programmcode, der tatsächlich rechnet; die
unternehmensindividuellen Abschnitte stammen aus den Einstellungen. Jede
erzeugte Fassung wird im Belegspeicher aufbewahrt, damit zu jedem
Geschäftsjahr die Dokumentation vorliegt, die damals gegolten hat
(GoBD Rz. 151).

` + HeadingGeneral + `

### 1.1 Das Unternehmen

| Angabe | Wert |
| --- | --- |
| Firma | {{orNone .CompanyName}} |
| Rechtsform | {{orNone .LegalForm}} |
| Anschrift | {{orNone .Street}}, {{orNone .ZipCity}}, {{orNone .Country}} |
| Steuernummer | {{orNone .TaxNumber}} |
| USt-IdNr. | {{orNone .VatID}} |
| Finanzamt | {{orNone .TaxOffice}} |
| Geschäftsjahr | {{.FiscalYear}} ({{.FiscalYearFrom}} bis {{.FiscalYearTo}}) |
| Kontenrahmen | {{orNone .SKR}} |
| Besteuerungsart | {{orNone .TaxationType}} |
| Voranmeldungszeitraum | {{orNone .VatPeriod}} |
{{if .LegalFormNote}}
{{.LegalFormNote}}
{{end}}
### 1.2 Sachlicher und zeitlicher Geltungsbereich

Die Dokumentation beschreibt die Buchführung des genannten Unternehmens mit
Buchfink ab dem Geschäftsjahr {{.FiscalYear}}. Sie gilt, bis eine neue Fassung
erzeugt wird; die abgelösten Fassungen bleiben im Belegspeicher.
{{if .SystemChangeDate}}
Die Daten wurden zum {{.SystemChangeDate}} aus einem Altsystem übernommen. Nach
§ 147 Abs. 6 Satz 6 AO ist das Altsystem noch fünf Jahre nach der Umstellung für
den Datenzugriff verfügbar zu halten.
{{end}}
### 1.3 Speicherort und Betriebsform

Die Daten liegen ausschließlich auf dem Rechner der Anwenderin bzw. des
Anwenders im Inland, im Ordner ` + "`{{orNone .DataDir}}`" + `. Es findet keine
Verlagerung der elektronischen Buchführung ins Ausland statt (§ 146 Abs. 2, 2a
AO). Es gibt keine Serverkomponente und keine Mandantenfähigkeit im Netz.
{{if .CloudWarning}}
**Hinweis:** {{.CloudWarning}}
{{end}}
Buchfink läuft im Einzelplatzbetrieb. Eine Benutzerverwaltung mit Anmeldung gibt
es nicht; an ihre Stelle tritt die Bearbeiterkennung aus Benutzerkonto und
Rechnername, die an jeder Buchung, jeder Festschreibung und jedem
Protokolleintrag festgehalten wird. Der Zugang zum Rechner ist die
Zugangskontrolle.

### 1.4 Abgedeckte und nicht abgedeckte Steuerfälle

Abgedeckt sind:
{{range .TaxCases}}
- {{.}}
{{- end}}

Nicht abgedeckt sind:
{{range .ExcludedCases}}
- {{.}}
{{- end}}

` + HeadingUser + `

### 2.1 Belegfluss

Eingang → Ablage → Erfassung der Kopfdaten → Buchung → Zahlung → Festschreibung.

1. **Eingang.** Ein Beleg kommt als Datei (E-Rechnung, PDF), per E-Mail oder als
   Papier ins Haus. Der Eingangsweg wird am Beleg festgehalten.
2. **Ablage.** Der Beleg wird unverändert in der empfangenen Form abgelegt und
   bekommt eine Belegnummer aus dem Nummernkreis. Die Originaldatei lässt sich
   danach nicht mehr entfernen (GoBD Rz. 131).
3. **Kopfdaten.** Belegdatum, Aussteller, Betrag und Steuerbetrag werden
   erfasst; bei einer E-Rechnung kommen sie aus dem strukturierten Teil.
4. **Buchung.** Die Buchung erfolgt innerhalb von {{.CaptureDays}} Tagen
   (GoBD Rz. 47). Ohne vollständige Kopfdaten ist ein Beleg nicht buchbar.
5. **Zahlung.** Der Zahlungsausgleich erfolgt über den Bankimport oder von Hand
   gegen den offenen Posten.
6. **Festschreibung.** Nach dem Prüflauf wird die Periode festgeschrieben. Ab
   dann nimmt sie keine Buchung mehr auf; Korrekturen laufen über die
   Generalumkehr mit dem Datum der Korrektur.

### 2.2 Ausgangsrechnungen

Ausgangsrechnungen werden in Buchfink erstellt, als E-Rechnung (ZUGFeRD oder
XRechnung) ausgegeben und im selben Zug gebucht und abgelegt. Die
Rechnungsnummer stammt aus dem Nummernkreis und ist fortlaufend und einmalig
(§ 14 Abs. 4 Nr. 4 UStG).

### 2.3 Bankimport

Kontoumsätze werden als CAMT.053 oder MT940 eingelesen. Der eingelesene
Kontoauszug wird als Beleg abgelegt; gebucht werden die einzelnen Umsätze aus
ihm, jeder gegen seinen offenen Posten oder gegen ein Sachkonto.

### 2.4 Monats- und Jahresabschluss

Zum Monatsende laufen Kontenabstimmung, Umsatzsteuer-Voranmeldung und die
Festschreibung des Monats. Zum Jahresende kommen die Abschlussbuchungen hinzu:
Abschreibungen, Abgrenzungen, Rückstellungen, Inventur und die
Ergebnisverwendung; danach die Feststellung und die Festschreibung des Jahres.

### 2.5 Eigenbelege

Wo kein Fremdbeleg vorliegt, wird ein Eigenbeleg erstellt. Er trägt Datum,
Betrag, Anlass und die Unterschrift bzw. die Bearbeiterkennung dessen, der ihn
ausgestellt hat.

### 2.6 Ersetzendes Scannen

Ersetzendes Scannen findet nicht statt. Papierbelege werden eingescannt und
zusätzlich im Original bis zum Ablauf der Aufbewahrungsfrist aufbewahrt. Eine
Vernichtung der Papieroriginale ist nicht vorgesehen.

### 2.7 Nummernkreise

| Kreis | Systematik | Nächste Nummer | Geltung |
| --- | --- | --- | --- |
{{- range .NumberRanges}}
| {{.Name}} | ` + "`{{.Format}}`" + ` | {{.Next}} | {{.Scope}} |
{{- end}}

Jeder Nummernkreis wird innerhalb derselben Transaktion vergeben, die den
Datensatz schreibt. Eine zurückgerollte Buchung verbraucht keine Nummer. Eine
dennoch entstandene Lücke wird als Lückenvermerk festgehalten und begründet.

### 2.8 Korrekturen

Eine gebuchte Buchung wird nicht geändert. Korrigiert wird durch Generalumkehr:
dieselben Konten auf denselben Seiten mit negativen Beträgen, datiert auf den
Tag der Korrektur. Die richtige Buchung wird neu erfasst und verweist auf die
stornierte; beide Richtungen sind im Journal sichtbar (GoBD Rz. 58).

### 2.9 Organisation im Unternehmen

**Zuständigkeiten:** {{orNone .Organisation.Responsibilities}}

**Belegfluss im Haus:** {{orNone .Organisation.ReceiptFlow}}

**Scannen:** {{orNone .Organisation.Scanning}}

**Freigabe und Festschreibung:** {{orNone .Organisation.Approval}}

**Vertretung:** {{orNone .Organisation.Substitution}}

**Sicherung:** {{orNone .Organisation.Backup}}
{{if .Organisation.Notes}}
**Weiteres:** {{.Organisation.Notes}}
{{end}}
` + HeadingTechnical + `

### 3.1 Datenmodell in Kurzform

Konto, Geschäftspartner mit Personenkonto, Beleg mit seinen Dateien, Buchung mit
ihren Zeilen, Zahlung mit ihren Zuordnungen, Anlagegut mit seinen Bewegungen,
Festschreibung, Änderungsprotokoll. Eine Buchung besteht immer aus Kopf und
mindestens zwei Zeilen und ist in sich ausgeglichen.

### 3.2 Unveränderbarkeit: Hash-Kette und Kanonisierung

Jede Buchung trägt den Hash ihres Vorgängers und einen eigenen Hash über ihren
Inhalt (SHA-256). Die Kette beginnt je Geschäftsjahr beim Genesis-Hash. Gehasht
wird eine kanonische Form: jedes Feld als Name, Bytelänge und Wert, in fester
Reihenfolge — die Längenangabe macht die Form fälschungssicher, weil kein Wert
so gebaut werden kann, dass er wie eine Feldgrenze aussieht.

Gedeckt sind alle Felder mit buchhalterischer Bedeutung, einschließlich aller
Zeilen, der Programmfassung, des Regelstands und der Bearbeiterkennung. Nicht
gedeckt sind reine Fundstellen (Belegkennung, Verweis auf eine korrigierte
Buchung) und der Festschreibungszeitpunkt: er wird nach dem Schreiben gesetzt
und ändert die Buchung nicht.

Das Änderungsprotokoll ist auf dieselbe Weise verkettet. Ein entfernter
Protokolleintrag bricht die Kette.

### 3.3 Beleg-Hash

Über die geordnete Dateiliste eines Belegs — Rolle, Dateiname und Prüfsumme
jeder Datei — und über die Kopfdaten wird ein Beleg-Hash gebildet. Er reist in
die Buchung und wird damit von der Journalkette mitgedeckt. Eine ausgetauschte
oder entfernte Datei ändert ihn.

### 3.4 Verschlüsselung

Personenbezogene und geschäftsgeheime Felder liegen feldweise verschlüsselt in
der Datenbank (AES-256-GCM). Der Schlüssel liegt im Schlüsselbund des
Betriebssystems; für den Verlustfall gibt es einen Wiederherstellungsschlüssel.

### 3.5 Zeitstempel

Zu jeder Festschreibung wird der Kettenkopf mit einem qualifizierten Zeitstempel
nach RFC 3161 beglaubigt ({{orNone .TSAName}}). Nur der Hash verlässt den
Rechner. Ist der Dienst nicht erreichbar, steht die Festschreibung trotzdem und
der Zeitstempel wird nachgeholt. Die beglaubigte Zeit wird mit der Systemzeit
verglichen; eine Abweichung über fünf Minuten wird protokolliert.

### 3.6 Festschreibung

Die Festschreibung sperrt einen Zeitraum bis zu einem Stichtag. Danach nimmt er
keine Buchung mehr auf. Jede erfasste Buchung bis zum Stichtag bekommt den
Festschreibungszeitpunkt und den Verweis auf die Festschreibung.

### 3.7 Exporte

{{range .ExportFormats}}
- {{.}}
{{- end}}

### 3.8 Aufbewahrungsfristen und Löschkonzept

| Datenkategorie | Klasse | Jahre | Rechtsgrundlage |
| --- | --- | --- | --- |
{{- range .RetentionRows}}
| {{.Category}} | {{.Class}} | {{.Years}} | {{.LegalBasis}} |
{{- end}}

Die Frist beginnt mit dem Schluss des Kalenderjahres, in dem die letzte
Eintragung gemacht oder der Beleg entstanden ist (§ 257 Abs. 5 HGB, § 147 Abs. 4
AO). Gelöscht wird nicht selbsttätig: die Löschung eines Geschäftsjahres ist ein
ausdrücklicher Vorgang mit Bestätigung, vorherigem Archivexport und Protokoll.
Solange eine Aussetzung der Frist eingetragen ist — Außenprüfung,
Rechtsbehelfsverfahren oder ein sonstiger Grund (§ 147 Abs. 3 Satz 5 AO) —, ist
sie gesperrt.

` + HeadingOperations + `

### 4.1 Sicherung

Zielordner: ` + "`{{orNone .BackupDir}}`" + `. Rhythmus: {{orNone .BackupRhythm}}.
Die Sicherung schreibt eine in sich stimmige Kopie der Datenbank und die
Belegdateien; jeder Lauf wird protokolliert.

{{if .BackupRuns}}
| Zeitpunkt | Ergebnis | Ziel | Größe (KiB) |
| --- | --- | --- | --- |
{{- range .BackupRuns}}
| {{.At}} | {{.Result}} | {{.Target}} | {{.SizeKiB}} |
{{- end}}
{{else}}
Es sind noch keine Sicherungsläufe protokolliert.
{{end}}

### 4.2 Wiederherstellung

Eine Sicherung wird schreibgeschützt geöffnet, ihre Hash-Ketten werden geprüft
und die Zählungen ausgewiesen, bevor sie übernommen wird. Die Übernahme wird als
Datenübernahme protokolliert.

### 4.3 Integritätsprüfung

Die Prüfung rechnet die Hash-Kette jedes Geschäftsjahres und die Kette des
Änderungsprotokolls nach und meldet jeden Bruch mit erwartetem und tatsächlichem
Hash. Sie läuft auf Anforderung, vor jeder Sicherung und als Teil des
Prüferpakets.

### 4.4 Änderungshistorie des Programms

{{.ChangelogTable}}

### 4.5 Migrationsprotokoll

{{if .Migrations}}
| Zeitpunkt | Programmfassung | Von | Nach | Ergebnis |
| --- | --- | --- | --- | --- |
{{- range .Migrations}}
| {{.At}} | {{.AppVersion}} | {{.FromVersion}} | {{.ToVersion}} | {{.Result}} |
{{- end}}
{{else}}
Es sind noch keine Schemaänderungen protokolliert.
{{end}}

` + HeadingControls + `

Das interne Kontrollsystem (GoBD Rz. 100 ff.) besteht aus den harten Regeln des
Buchungskerns und den Prüfläufen vor der Festschreibung.

### 5.1 Harte Regeln des Buchungskerns

- Eine Buchung hat mindestens eine Soll- und eine Haben-Zeile und ist exakt
  ausgeglichen; eine unausgeglichene Buchung wird abgewiesen.
- Jedes Konto muss im Kontenplan existieren und bebuchbar sein; die
  Steuerkonten sind der Steuerautomatik vorbehalten.
- Die Buchungsnummern sind lückenlos und werden in derselben Transaktion
  vergeben wie die Buchung.
- In eine festgeschriebene Periode wird nicht mehr gebucht.
- In ein festgestelltes Geschäftsjahr wird nicht mehr gebucht.
- Ein Beleg ohne ansehbare Darstellung und ohne vollständige Kopfdaten ist nicht
  buchbar.
- Eine Generalumkehr trägt negative Beträge auf denselben Seiten und verweist
  auf die ursprüngliche Buchung.

### 5.2 Prüfregeln vor der Festschreibung

| Regel | Gewicht | Zweck |
| --- | --- | --- |
{{- range .CheckRules}}
| ` + "`{{.Key}}`" + ` | {{.Severity}} | {{.Purpose}} |
{{- end}}

Ein blockierender Befund verhindert die Festschreibung. Er lässt sich mit einer
Begründung übergehen; die Begründung steht am Prüflauf und im
Änderungsprotokoll.
`
