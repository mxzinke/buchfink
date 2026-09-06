package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

// --- Vorher/Nachher des Änderungsprotokolls -------------------------------

func TestChangedFieldsKeepsOnlyWhatChanged(t *testing.T) {
	before := &Contact{
		ID: 7, Type: ContactTypeCustomer, Name: "Meier GmbH",
		LedgerAccount: "10001", Street: "Hauptstraße 1", City: "München",
		PaymentTermsDays: 14,
	}
	after := *before
	after.Street = "Nebenstraße 5"
	after.PaymentTermsDays = 30

	beforeJSON, afterJSON := ChangedFields(before, &after)
	if beforeJSON == "" || afterJSON == "" {
		t.Fatal("eine Änderung muss Vorher und Nachher liefern")
	}

	var beforeMap, afterMap map[string]any
	if err := json.Unmarshal([]byte(beforeJSON), &beforeMap); err != nil {
		t.Fatalf("Vorher ist kein JSON-Objekt: %v", err)
	}
	if err := json.Unmarshal([]byte(afterJSON), &afterMap); err != nil {
		t.Fatalf("Nachher ist kein JSON-Objekt: %v", err)
	}

	// Genau die beiden geänderten Felder — nicht der ganze Datensatz. Sonst
	// verbärge sich die eine Änderung zwischen dreißig unveränderten Feldern.
	wantKeys := map[string]bool{"street": true, "paymentTermsDays": true}
	if len(afterMap) != len(wantKeys) {
		t.Errorf("Nachher trägt %d Felder (%v), erwartet %d", len(afterMap), afterMap, len(wantKeys))
	}
	for key := range wantKeys {
		if _, ok := afterMap[key]; !ok {
			t.Errorf("das geänderte Feld %q fehlt im Nachher", key)
		}
	}
	if _, ok := afterMap["name"]; ok {
		t.Error("der unveränderte Name gehört nicht ins Protokoll")
	}
	if beforeMap["street"] != "Hauptstraße 1" {
		t.Errorf("Vorher.street = %v, erwartet die alte Anschrift", beforeMap["street"])
	}
	if afterMap["street"] != "Nebenstraße 5" {
		t.Errorf("Nachher.street = %v, erwartet die neue Anschrift", afterMap["street"])
	}
}

func TestChangedFieldsOnCreateHasNoBefore(t *testing.T) {
	after := &Contact{ID: 1, Type: ContactTypeVendor, Name: "Neu GmbH", LedgerAccount: "70001"}

	beforeJSON, afterJSON := ChangedFields(nil, after)
	if beforeJSON != "" {
		t.Errorf("ein neu angelegter Datensatz hat kein Vorher, bekommen %q", beforeJSON)
	}
	if !strings.Contains(afterJSON, "Neu GmbH") {
		t.Errorf("das Nachher muss den angelegten Datensatz tragen, bekommen %q", afterJSON)
	}
}

func TestChangedFieldsIsEmptyWithoutChange(t *testing.T) {
	c := &Contact{ID: 3, Name: "Unverändert GmbH", LedgerAccount: "10002"}
	same := *c

	beforeJSON, afterJSON := ChangedFields(c, &same)
	if beforeJSON != "" || afterJSON != "" {
		t.Errorf("ohne Änderung darf nichts protokolliert werden, bekommen %q / %q", beforeJSON, afterJSON)
	}
}

func TestChangedFieldsIgnoresNamedFields(t *testing.T) {
	// Der Saldo des Personenkontos wird beim Lesen gerechnet und nicht
	// gespeichert. Stünde er im Protokoll, meldete jedes Speichern eine
	// Änderung, die niemand vorgenommen hat.
	before := &Contact{ID: 4, Name: "Meier GmbH", OpenAmount: 10000}
	after := *before
	after.OpenAmount = 25000

	beforeJSON, afterJSON := ChangedFields(before, &after, "openAmount")
	if beforeJSON != "" || afterJSON != "" {
		t.Errorf("ausgenommene Felder gehören nicht ins Protokoll, bekommen %q / %q", beforeJSON, afterJSON)
	}
}

func TestChangedFieldsWorksOnSettings(t *testing.T) {
	before := &CompanySettings{CompanyName: "Alt GmbH", TaxationType: "SOLL", VatPeriod: "quarter"}
	after := &CompanySettings{CompanyName: "Alt GmbH", TaxationType: "IST", VatPeriod: "quarter"}

	beforeJSON, afterJSON := ChangedFields(before, after)
	if !strings.Contains(beforeJSON, "SOLL") || !strings.Contains(afterJSON, "IST") {
		t.Errorf("die Umstellung der Besteuerungsart muss im Protokoll stehen: %q → %q", beforeJSON, afterJSON)
	}
	if strings.Contains(afterJSON, "companyName") {
		t.Error("der unveränderte Firmenname gehört nicht ins Protokoll")
	}
}

// --- Belegkopfdaten -------------------------------------------------------

func bookableReceipt() *Receipt {
	return &Receipt{
		ReceiptNumber: "ER-2026-0001",
		FiscalYear:    2026,
		Direction:     DirectionIncoming,
		Status:        ReceiptStatusFiled,
		Kind:          ReceiptKindInvoice,
		DocumentDate:  "2026-03-01",
		IssuerName:    "Lieferant GmbH",
		GrossAmount:   11900,
		TaxAmount:     1900,
		Currency:      "EUR",
		Files: []ReceiptFile{{
			Position: 1, Role: ReceiptRoleOriginal, FileName: "rechnung.pdf",
			MimeType: "application/pdf", Size: 1024,
			SHA256:     strings.Repeat("a", 64),
			StoredPath: "belege/2026/eingang/aaa.pdf",
		}},
	}
}

func TestValidateBookableDemandsHeaderFields(t *testing.T) {
	receipt := bookableReceipt()
	if err := receipt.ValidateBookable(); err != nil {
		t.Fatalf("ein vollständiger Beleg muss buchbar sein: %v", err)
	}

	withoutDate := bookableReceipt()
	withoutDate.DocumentDate = ""
	if err := withoutDate.ValidateBookable(); err == nil {
		t.Error("ohne Belegdatum darf nicht gebucht werden — die zeitgerechte Erfassung wäre nicht zu beurteilen")
	}

	withoutIssuer := bookableReceipt()
	withoutIssuer.IssuerName = ""
	if err := withoutIssuer.ValidateBookable(); err == nil {
		t.Error("ohne Aussteller darf nicht gebucht werden (§ 14 Abs. 4 Nr. 1 UStG)")
	}

	withoutAmount := bookableReceipt()
	withoutAmount.GrossAmount = 0
	if err := withoutAmount.ValidateBookable(); err == nil {
		t.Error("ohne Betrag darf nicht gebucht werden")
	}
}

func TestValidateBookableHasOtherRulesForStatementsAndSelfIssued(t *testing.T) {
	// Ein Kontoauszug hat keinen Aussteller im Sinne einer Rechnung und keinen
	// einzelnen Betrag: er braucht nur seine Einordnung in der Zeit.
	statement := bookableReceipt()
	statement.Kind = ReceiptKindStatement
	statement.IssuerName = ""
	statement.GrossAmount = 0
	if err := statement.ValidateBookable(); err != nil {
		t.Errorf("ein Kontoauszug braucht nur das Belegdatum: %v", err)
	}

	// Der Eigenbeleg hat keinen fremden Aussteller, aber einen Anlass.
	selfIssued := bookableReceipt()
	selfIssued.Kind = ReceiptKindSelfIssued
	selfIssued.IssuerName = ""
	if err := selfIssued.ValidateBookable(); err == nil {
		t.Error("ein Eigenbeleg ohne Betreff nennt seinen Anlass nicht und darf nicht gebucht werden")
	}
	selfIssued.Subject = "Trinkgeld Taxifahrt Kundentermin"
	if err := selfIssued.ValidateBookable(); err != nil {
		t.Errorf("ein Eigenbeleg mit Betreff und Betrag ist buchbar: %v", err)
	}
}

func TestOtherDocumentsNeedDateAndSubjectInsteadOfIssuerAndAmount(t *testing.T) {
	// Die Schlussbilanz des Altsystems, die Inventurliste, der
	// Gesellschafterbeschluss: Dokumente, die eine Buchung tragen, aber keine
	// Rechnung sind. § 14 Abs. 4 Nr. 1 UStG gilt für sie nicht — sie haben
	// keinen leistenden Unternehmer und oft keinen einzelnen Betrag.
	for _, kind := range []ReceiptKind{ReceiptKindOther, ReceiptKindLetter} {
		document := bookableReceipt()
		document.Kind = kind
		document.IssuerName = ""
		document.GrossAmount = 0

		if err := document.ValidateBookable(); err == nil {
			t.Errorf("%s: ohne Betreff ist nicht zu erkennen, was das Dokument belegt", kind.Label())
		} else if strings.Contains(err.Error(), "§ 14 Abs. 4 Nr. 1 UStG") {
			t.Errorf("%s: die Rechnungspflichtangabe gilt hier nicht: %v", kind.Label(), err)
		}

		document.Subject = "Schlussbilanz zum 31.12.2025"
		if err := document.ValidateBookable(); err != nil {
			t.Errorf("%s mit Belegdatum und Betreff ist buchbar: %v", kind.Label(), err)
		}
	}

	// Die Rechnung bleibt, was sie war: ohne Aussteller und Betrag nicht
	// buchbar.
	invoice := bookableReceipt()
	invoice.IssuerName = ""
	if err := invoice.ValidateBookable(); err == nil {
		t.Error("eine Rechnung ohne Aussteller darf nicht gebucht werden (§ 14 Abs. 4 Nr. 1 UStG)")
	}
}

func TestHasHeaderIsTheCanonicalSwitch(t *testing.T) {
	legacy := bookableReceipt()
	legacy.DocumentDate = ""
	if legacy.HasHeader() {
		t.Error("ein Beleg ohne Belegdatum stammt aus der Zeit vor den Kopfdaten")
	}
	if !bookableReceipt().HasHeader() {
		t.Error("ein Beleg mit Belegdatum trägt Kopfdaten")
	}
}

func TestLetterIsNotBooked(t *testing.T) {
	// Ein Handelsbrief belegt eine Abrede, keinen Geschäftsvorfall. Meldete der
	// Prüflauf ihn als „abgelegt, aber nicht gebucht", stünde er dort für immer.
	if ReceiptKindLetter.RequiresBooking() {
		t.Error("ein Handelsbrief trägt keine Buchung")
	}
	if !ReceiptKindInvoice.RequiresBooking() {
		t.Error("eine Rechnung ist zu buchen")
	}
	if RetentionKindOf(ReceiptKindLetter) != RetentionKindReceiptLetter {
		t.Error("der Handelsbrief muss seine eigene Aufbewahrungsart haben — sechs statt acht Jahre")
	}
}

// --- Hinweise -------------------------------------------------------------

func TestCloudFolderWarningRecognisesSyncFolders(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/home/anna/OneDrive/Buchfink/data", true},
		{"/Users/anna/Dropbox/buchhaltung", true},
		{`C:\Users\Anna\Google Drive\Buchfink`, true},
		{"/Users/anna/Library/Mobile Documents/com~apple~CloudDocs/iCloud/buchfink", true},
		{"/home/anna/Nextcloud/daten", true},
		{"/home/anna/.buchfink/data", false},
		{"/var/lib/buchfink", false},
	}
	for _, tc := range cases {
		got := CloudFolderWarning(tc.path) != ""
		if got != tc.want {
			t.Errorf("CloudFolderWarning(%q) = %v, erwartet %v", tc.path, got, tc.want)
		}
	}
	if warning := CloudFolderWarning("/home/anna/Dropbox/x"); !strings.Contains(warning, "§ 146") {
		t.Errorf("der Hinweis muss die Norm nennen: %q", warning)
	}
}

func TestLegalFormLimitationNoteOnlyWhereWithdrawalsExist(t *testing.T) {
	for _, form := range []string{"GbR", "OHG", "KG", "Eingetragener Kaufmann (e. K.)", "Einzelunternehmen"} {
		note := LegalFormLimitationNote(form)
		if note == "" {
			t.Errorf("%s kennt Entnahmen — der Hinweis fehlt", form)
			continue
		}
		if !strings.Contains(note, "§ 4 Abs. 4a EStG") {
			t.Errorf("%s: der Hinweis muss § 4 Abs. 4a EStG nennen: %q", form, note)
		}
	}
	for _, form := range []string{"GmbH", "AG", "UG (haftungsbeschränkt)", "eG"} {
		if LegalFormLimitationNote(form) != "" {
			t.Errorf("%s kennt keine Entnahmen — der Hinweis gehört dort nicht hin", form)
		}
	}
}

func TestTaxCaseHintsNameTheGaps(t *testing.T) {
	hints := TaxCaseHints()
	if len(hints) == 0 {
		t.Fatal("die Grenzen des Funktionsumfangs müssen benannt sein")
	}
	joined := strings.Join(hints, " ")
	for _, needle := range []string{"OSS", "IOSS", "§ 19 UStG"} {
		if !strings.Contains(joined, needle) {
			t.Errorf("die Hinweise nennen %q nicht", needle)
		}
	}
}

// --- Aussetzung der Aufbewahrungsfrist ------------------------------------

func TestRetentionHoldNeedsADescribedReason(t *testing.T) {
	hold := &RetentionHold{FiscalYear: 2025, Reason: RetentionHoldAudit}
	if err := hold.Validate(); err != nil {
		t.Errorf("die Außenprüfung ist ein benannter Grund: %v", err)
	}

	other := &RetentionHold{FiscalYear: 2025, Reason: RetentionHoldOther}
	if err := other.Validate(); err == nil {
		t.Error("ein sonstiger Grund ist zu beschreiben — sonst lässt sich später nicht beurteilen, ob er noch gilt")
	}
	other.Description = "Betriebsprüfung des Zolls angekündigt"
	if err := other.Validate(); err != nil {
		t.Errorf("ein beschriebener sonstiger Grund ist zulässig: %v", err)
	}

	unknown := &RetentionHold{FiscalYear: 2025, Reason: "irgendwas"}
	if err := unknown.Validate(); err == nil {
		t.Error("ein unbekannter Grund darf keine Frist aussetzen")
	}
}

func TestMigrationCountsBalance(t *testing.T) {
	// Eine Buchung ist immer ausgeglichen, also ist es auch die Summe aller.
	// Stimmen Soll und Haben nach einer Übernahme nicht überein, fehlt etwas.
	balanced := MigrationCounts{DebitTotal: 100000, CreditTotal: 100000}
	if !balanced.IsBalanced() {
		t.Error("gleiche Summen müssen als ausgeglichen gelten")
	}
	broken := MigrationCounts{DebitTotal: 100000, CreditTotal: 99900}
	if broken.IsBalanced() {
		t.Error("ungleiche Summen bedeuten eine unvollständige Übernahme")
	}
}
