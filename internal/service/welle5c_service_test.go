package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// -------------------------------------------------------------------------
// Verzeichnis der Vorsteuerberichtigung (§ 15a UStG)
// -------------------------------------------------------------------------

func (e *testEnv) inputTax(t *testing.T) *InputTaxService {
	t.Helper()
	return NewInputTaxService(
		repository.NewInputTaxCorrectionRepository(e.db),
		e.journal, e.journalRepo,
		repository.NewSettingsRepository(e.db),
		nil,
		repository.NewAuditRepository(e.db),
		e.fiscalYear,
	)
}

// Der Pkw aus dem Lehrbuch: 47.600 € brutto, 7.600 € Vorsteuer, im Zugangsjahr
// zu 100 % abziehbar verwendet, im dritten Jahr nur noch zu 60 %. Berichtigt
// werden 608 €, und sie landen in der Kennziffer 64 der Voranmeldung.
func TestInputTaxCorrectionRegisterBooksIntoCode64(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.inputTax(t)

	correction, err := svc.Register(ctx, RegisterInputTaxRequest{
		Label: "Pkw AN-2026-0001", Account: "0520",
		AcquisitionDate: "2026-01-15",
		NetAmount:       4_000_000, InputTaxAmount: 760_000,
	})
	if err != nil {
		t.Fatalf("Aufnahme ins Verzeichnis: %v", err)
	}
	if correction.CorrectionPeriodYears != 5 {
		t.Errorf("Berichtigungszeitraum %d Jahre — ein Pkw ist beweglich, also fünf",
			correction.CorrectionPeriodYears)
	}
	if correction.OriginalPermille != 1000 {
		t.Errorf("ursprünglicher Anteil %d — ohne Angabe ist die Verwendung voll abziehbar",
			correction.OriginalPermille)
	}

	// Das Zugangsjahr gehört in den Zeitraum — er läuft ab der erstmaligen
	// Verwendung (§ 15a Abs. 1 UStG). Zu berichtigen ist darin nur, was von der
	// beim Abzug beabsichtigten Verwendung abweicht; hier nichts.
	first, err := svc.Year(ctx, 2026)
	if err != nil {
		t.Fatalf("Verzeichnis 2026: %v", err)
	}
	if !first.Rows[0].InPeriod {
		t.Error("das Jahr der erstmaligen Verwendung gehört in den Berichtigungszeitraum")
	}
	if first.Rows[0].Assessment.Required {
		t.Errorf("ohne geänderte Verwendung ist nichts zu berichtigen: %s",
			first.Rows[0].Assessment.Reason)
	}
	// Zugang am 15.01.2026: der Zeitraum endet am 14.01.2031, und weil das vor
	// dem 16. liegt, bleibt der Januar 2031 nach § 45 UStDV unberücksichtigt.
	if correction.PeriodEnd != "2030-12-31" {
		t.Errorf("Ende des Zeitraums %s — erwartet 2030-12-31", correction.PeriodEnd)
	}
	if correction.LastFiscalYear != 2030 {
		t.Errorf("letztes Berichtigungsjahr %d — erwartet 2030", correction.LastFiscalYear)
	}

	// Ohne bestätigten Anteil wird nicht gebucht.
	if _, err := svc.BookYear(ctx, 2028); err == nil {
		t.Fatal("ein ungeprüfter Verwendungsanteil darf nicht gebucht werden")
	}

	view, err := svc.SaveUsage(ctx, SaveInputTaxUsageRequest{
		CorrectionID: correction.ID, FiscalYear: 2028, Permille: 600,
		Reason: "Kfz ab 2028 zu 40 % für steuerfreie Vermietung eingesetzt",
	})
	if err != nil {
		t.Fatalf("Verwendungsanteil: %v", err)
	}
	if view.Unconfirmed != 0 {
		t.Errorf("%d Anteile offen — nach der Bestätigung darf keiner mehr offen sein",
			view.Unconfirmed)
	}
	if view.TotalAmount != -60_800 {
		t.Errorf("Berichtigung %s € — erwartet -608,00 €", view.TotalAmount)
	}

	booked, err := svc.BookYear(ctx, 2028)
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	if !booked.Rows[0].Booked {
		t.Fatal("nach dem Lauf muss die Berichtigung als gebucht vermerkt sein")
	}

	entries, err := env.journalRepo.FindAll(ctx, 2028)
	if err != nil {
		t.Fatalf("Journal: %v", err)
	}
	var entry *domain.JournalEntry
	for i := range entries {
		if entries[i].DocumentNumber == "15A-2028" {
			entry = &entries[i]
		}
	}
	if entry == nil {
		t.Fatal("die Berichtigungsbuchung steht nicht im Journal")
	}
	if entry.BookingDate != "2028-12-31" {
		t.Errorf("Buchungsdatum %s — berichtigt wird zum Ende des Wirtschaftsjahres",
			entry.BookingDate)
	}

	var taxLine *domain.JournalLine
	for i := range entry.Lines {
		if entry.Lines[i].TaxKey == accounting.TaxKeyInputTaxCorrection {
			taxLine = &entry.Lines[i]
		}
	}
	if taxLine == nil {
		t.Fatalf("keine Zeile mit dem Steuerschlüssel %s", accounting.TaxKeyInputTaxCorrection)
	}
	if taxLine.Account != accounting.InputTaxCorrectionRepayableMovable {
		t.Errorf("Konto %s — die zurückzuzahlende Vorsteuer eines beweglichen Wirtschaftsguts "+
			"gehört auf %s", taxLine.Account, accounting.InputTaxCorrectionRepayableMovable)
	}
	if taxLine.Side != domain.SideCredit || taxLine.Amount != 60_800 {
		t.Errorf("Steuerzeile %s %s € — erwartet Haben 608,00 €", taxLine.Side, taxLine.Amount)
	}

	period, err := accounting.VatPeriodOf("2028-12-31", "quarter")
	if err != nil {
		t.Fatalf("Zeitraum: %v", err)
	}
	ret := accounting.BuildVatReturn(period, accounting.VatReturnSource{
		Entries: []domain.JournalEntry{*entry},
	})
	if got := ret.Tax(accounting.VatCodeInputTaxCorrection); got != -60_800 {
		t.Errorf("Kennziffer 64 meldet %s € — erwartet -608,00 €", got)
	}
	if got := ret.Tax(accounting.VatCodeInputTax); got != 0 {
		t.Errorf("Kennziffer 66 meldet %s € — die Berichtigung gehört nicht dorthin", got)
	}
}

// Die Bagatellgrenzen greifen auch im Verzeichnis: ein Wirtschaftsgut mit 900 €
// Vorsteuer wird nie berichtigt, so groß die Änderung auch ist.
func TestInputTaxCorrectionRegisterRespectsTheMinorLimit(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.inputTax(t)

	correction, err := svc.Register(ctx, RegisterInputTaxRequest{
		Label: "Kopierer", Account: "0650",
		AcquisitionDate: "2026-03-01", NetAmount: 473_700, InputTaxAmount: 90_000,
	})
	if err != nil {
		t.Fatalf("Aufnahme: %v", err)
	}
	if _, err := svc.SaveUsage(ctx, SaveInputTaxUsageRequest{
		CorrectionID: correction.ID, FiscalYear: 2027, Permille: 0,
	}); err != nil {
		t.Fatalf("Verwendungsanteil: %v", err)
	}
	view, err := svc.Year(ctx, 2027)
	if err != nil {
		t.Fatalf("Verzeichnis: %v", err)
	}
	if view.Rows[0].Assessment.Required {
		t.Errorf("bei 900 € Vorsteuer entfällt die Berichtigung: %s", view.Rows[0].Assessment.Reason)
	}
	if view.TotalAmount != 0 {
		t.Errorf("Summe %s € — erwartet null", view.TotalAmount)
	}
	booked, err := svc.BookYear(ctx, 2027)
	if err != nil {
		t.Fatalf("Lauf: %v", err)
	}
	if !strings.Contains(booked.Note, "nichts zu berichtigen") {
		t.Errorf("der Lauf muss sagen, dass nichts zu berichtigen ist: %s", booked.Note)
	}
}

// -------------------------------------------------------------------------
// Bestätigung der USt-IdNr. beim Bundeszentralamt
// -------------------------------------------------------------------------

// bzstServer ist ein Bundeszentralamt aus Papier: es antwortet mit dem
// übergebenen Ergebniscode und zählt die Aufrufe.
func bzstServer(t *testing.T, code string, fields map[string]string) (*httptest.Server, *int) {
	t.Helper()
	calls := 0
	body := `{"errorCode":"` + code + `","statusMeldung":"Antwort des Testservers",` +
		`"anfrageId":"2026-0001"`
	for key, value := range fields {
		body += `,"` + key + `":"` + value + `"`
	}
	body += "}"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server, &calls
}

// vatIDs baut den Dienst mit der Adresse aus den Einstellungen. Der Weg über
// die Einstellung und nicht über einen ausgetauschten Client ist Absicht: die
// Adresse des Bundeszentralamts ist einstellbar, damit ein Wechsel keine
// Programmversion verlangt — und genau dieser Weg wird hier mitgeprüft.
func (e *testEnv) vatIDs(t *testing.T, endpoint string) *VatIDService {
	t.Helper()
	settings := repository.NewSettingsRepository(e.db)
	if err := settings.Set(context.Background(), SettingVatIDEndpoint, endpoint); err != nil {
		t.Fatalf("Endpunkt setzen: %v", err)
	}
	return NewVatIDService(
		repository.NewVatIDCheckRepository(e.db), e.contactRepo,
		settings, repository.NewAuditRepository(e.db),
	)
}

// Ergebniscode 200: die Nummer ist bestätigt, das Ergebnis wird aufgehoben, und
// die nächste Frage innerhalb von 90 Tagen kommt ohne Netzaufruf aus.
func TestVatIDConfirmationIsStoredAndCachedForNinetyDays(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, calls := bzstServer(t, "200", map[string]string{
		"ergName": "A", "ergOrt": "A", "ergPlz": "A", "ergStr": "A",
	})
	svc := env.vatIDs(t, server.URL)
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")

	check, err := svc.Check(ctx, customer.ID)
	if err != nil {
		t.Fatalf("Bestätigungsanfrage: %v", err)
	}
	if check.Status != domain.VatIDValid {
		t.Fatalf("Ergebnis %q — erwartet gültig", check.Status)
	}
	if check.RequestID != "2026-0001" {
		t.Errorf("Abfrage-Nr. %q — sie ist der Nachweis gegenüber dem Finanzamt", check.RequestID)
	}
	if check.RawResponse == "" {
		t.Error("die Antwort wird aufgehoben, wie sie kam (GoBD Rz. 130)")
	}
	if check.Endpoint != server.URL {
		t.Errorf("Endpunkt %q — die befragte Stelle gehört ans Ergebnis", check.Endpoint)
	}

	// Innerhalb der Frist genügt die gespeicherte Bestätigung.
	if err := svc.EnsureConfirmed(ctx, customer, ""); err != nil {
		t.Fatalf("die gespeicherte Bestätigung muss genügen: %v", err)
	}
	if *calls != 1 {
		t.Errorf("%d Aufrufe — innerhalb von 90 Tagen wird nicht erneut gefragt", *calls)
	}

	// Nach 91 Tagen wird erneut gefragt.
	svc.SetClock(func() time.Time { return time.Now().AddDate(0, 0, 91) })
	if err := svc.EnsureConfirmed(ctx, customer, ""); err != nil {
		t.Fatalf("die erneute Abfrage muss durchgehen: %v", err)
	}
	if *calls != 2 {
		t.Errorf("%d Aufrufe — nach Ablauf der Frist wird erneut gefragt", *calls)
	}
}

// Ergebniscode 201: die Nummer ist ungültig. Eine steuerfreie Lieferung an
// diesen Empfänger wird abgelehnt, und die Ablehnung nennt das Ergebnis.
func TestVatIDRejectionBlocksTheExemptSupply(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, _ := bzstServer(t, "201", nil)
	svc := env.vatIDs(t, server.URL)
	customer := env.customer(t, "Client SARL", "FR", "FR99999999999")

	err := svc.EnsureConfirmed(ctx, customer, "")
	if err == nil {
		t.Fatal("eine nicht bestätigte USt-IdNr. trägt keine steuerfreie Lieferung")
	}
	if !strings.Contains(err.Error(), "§ 6a Abs. 1") {
		t.Errorf("die Meldung muss die Vorschrift nennen: %v", err)
	}
	if !strings.Contains(err.Error(), "201") {
		t.Errorf("die Meldung muss den Ergebniscode nennen: %v", err)
	}

	// Auch das negative Ergebnis wird aufgehoben: es ist der Nachweis, dass
	// geprüft wurde.
	checks, err := svc.Checks(ctx, customer.ID)
	if err != nil {
		t.Fatalf("Verlauf: %v", err)
	}
	if len(checks) != 1 || checks[0].Status != domain.VatIDInvalid {
		t.Errorf("Verlauf %+v — erwartet genau ein ungültiges Ergebnis", checks)
	}
}

// Ist das Bundeszentralamt nicht erreichbar, ist das kein negatives Ergebnis.
// Die Rechnung geht dann nur mit einem festgehaltenen Grund hinaus.
func TestVatIDUnavailableNeedsAnOverride(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	// Ein Server, der sofort geschlossen wird: die Verbindung scheitert.
	server, _ := bzstServer(t, "200", nil)
	url := server.URL
	server.Close()

	svc := env.vatIDs(t, url)
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")

	err := svc.EnsureConfirmed(ctx, customer, "")
	if err == nil {
		t.Fatal("ohne Bestätigung und ohne Grund geht die Rechnung nicht hinaus")
	}
	if strings.Contains(err.Error(), "nicht bestätigt") {
		t.Errorf("eine ausgebliebene Antwort ist kein negatives Ergebnis: %v", err)
	}

	if err := svc.EnsureConfirmed(ctx, customer,
		"Bundeszentralamt nicht erreichbar, Nummer aus dem Vorjahr bestätigt"); err != nil {
		t.Fatalf("mit Grund muss die Rechnung hinausgehen: %v", err)
	}
}

// -------------------------------------------------------------------------
// Belegnachweis der innergemeinschaftlichen Lieferung
// -------------------------------------------------------------------------

func (e *testEnv) supplyEvidence(t *testing.T) *SupplyEvidenceService {
	t.Helper()
	return NewSupplyEvidenceService(
		repository.NewSupplyEvidenceRepository(e.db),
		repository.NewInvoiceRepository(e.db),
		repository.NewAuditRepository(e.db),
		e.fiscalYear,
	)
}

// Der Bericht führt die steuerfreie Lieferung ohne vollständigen Nachweis auf —
// und nimmt sie heraus, sobald er geführt ist.
func TestSupplyEvidenceReportListsIncompleteSupplies(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.supplyEvidence(t)
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")

	invoiceRepo := repository.NewInvoiceRepository(env.db)
	invoice := &domain.Invoice{
		FiscalYear: 2026, InvoiceNumber: "RE-2026-0001", Date: "2026-04-02",
		ServiceDateFrom: "2026-04-01", ServiceDateTo: "2026-04-01",
		ContactID: customer.ID, ContactName: customer.Name,
		TaxTreatment: domain.TaxTreatmentIntraCommunitySupply,
		Status:       domain.InvoiceStatusIssued, Currency: "EUR",
		NetAmount: 500_000, GrossAmount: 500_000,
	}
	if err := invoiceRepo.Save(ctx, invoice); err != nil {
		t.Fatalf("Rechnung: %v", err)
	}

	report, err := svc.Report(ctx, 2026)
	if err != nil {
		t.Fatalf("Bericht: %v", err)
	}
	if report.Incomplete != 1 || len(report.Rows) != 1 {
		t.Fatalf("%d von %d unvollständig — erwartet eine Lieferung ohne Nachweis",
			report.Incomplete, len(report.Rows))
	}
	if !strings.Contains(report.Note, "§ 17a") {
		t.Errorf("der Bericht muss den Fristhinweis tragen: %s", report.Note)
	}

	// Zwei Belege unabhängiger Aussteller: die Vermutung greift.
	for _, item := range []SupplyEvidenceRequest{
		{InvoiceID: invoice.ID, Kind: string(accounting.EvidenceCMR),
			Issuer: "Spedition Nord", Independent: true, Date: "2026-04-03"},
		{InvoiceID: invoice.ID, Kind: string(accounting.EvidenceInsurance),
			Issuer: "Transportversicherung AG", Independent: true, Date: "2026-04-03"},
	} {
		if _, err := svc.Add(ctx, item); err != nil {
			t.Fatalf("Nachweisbeleg: %v", err)
		}
	}

	view, err := svc.View(ctx, invoice.ID, accounting.TransportBySupplier)
	if err != nil {
		t.Fatalf("Nachweisstand: %v", err)
	}
	if !view.Status.Fulfilled {
		t.Errorf("mit zwei unabhängigen Belegen greift die Vermutung: %s", view.Status.Reason)
	}
	if len(view.Items) != 2 {
		t.Errorf("%d Belege abgelegt — erwartet zwei", len(view.Items))
	}

	after, err := svc.Report(ctx, 2026)
	if err != nil {
		t.Fatalf("Bericht: %v", err)
	}
	if after.Incomplete != 0 {
		t.Errorf("%d unvollständig — nach dem Nachweis darf keine offen sein", after.Incomplete)
	}

	// Ein Beleg ohne Aussteller trägt die Vermutung nicht und wird deshalb gar
	// nicht erst angenommen.
	if _, err := svc.Add(ctx, SupplyEvidenceRequest{
		InvoiceID: invoice.ID, Kind: string(accounting.EvidenceCMR), Date: "2026-04-04",
	}); err == nil {
		t.Error("ein Nachweisbeleg ohne Aussteller lässt sich nicht auf Unabhängigkeit prüfen")
	}
}

// -------------------------------------------------------------------------
// Freistellungsbescheinigung nach § 48b EStG
// -------------------------------------------------------------------------

// Die Bescheinigung wird mit ihrer Frist überwacht: 30 Tage vorher steht sie in
// der Warnliste, danach als abgelaufen. Wer sie erst am Ablauftag erfährt, hat
// bei der nächsten Zahlung schon einbehalten müssen.
func TestExemptionCertificateWarnsBeforeItExpires(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	valid := env.vendor(t, "Bau Weit GmbH", "DE", "")
	valid.ExemptionCertificateNumber = "FS-2026-001"
	valid.ExemptionCertificateValidUntil = "2027-06-30"
	if err := env.contacts.SaveContact(ctx, valid); err != nil {
		t.Fatalf("Lieferant: %v", err)
	}
	soon := env.vendor(t, "Bau Bald GmbH", "DE", "")
	soon.ExemptionCertificateNumber = "FS-2026-002"
	soon.ExemptionCertificateValidUntil = "2026-04-10"
	if err := env.contacts.SaveContact(ctx, soon); err != nil {
		t.Fatalf("Lieferant: %v", err)
	}
	expired := env.vendor(t, "Bau Vorbei GmbH", "DE", "")
	expired.ExemptionCertificateNumber = "FS-2025-003"
	expired.ExemptionCertificateValidUntil = "2026-02-01"
	if err := env.contacts.SaveContact(ctx, expired); err != nil {
		t.Fatalf("Lieferant: %v", err)
	}

	warnings, err := env.contacts.ExemptionCertificateWarnings(ctx, "2026-03-20")
	if err != nil {
		t.Fatalf("Warnliste: %v", err)
	}
	if len(warnings) != 2 {
		t.Fatalf("%d Warnungen — erwartet die abgelaufene und die bald ablaufende: %+v",
			len(warnings), warnings)
	}
	if warnings[0].State != "expired" || warnings[0].ContactID != expired.ID {
		t.Errorf("erste Warnung %+v — die abgelaufene steht vorn", warnings[0])
	}
	if warnings[1].State != "expiring" || warnings[1].ContactID != soon.ID {
		t.Errorf("zweite Warnung %+v — erwartet die in 21 Tagen ablaufende", warnings[1])
	}
	for _, w := range warnings {
		if !strings.Contains(w.Note, "§ 48 EStG") {
			t.Errorf("die Warnung muss die Folge nennen: %s", w.Note)
		}
	}

	// Ein Kontakt ohne Bescheinigung taucht nicht auf: das Feld wird nur geführt,
	// wo es eine gibt.
	if got := valid.ExemptionCertificateState("2026-03-20"); got != "valid" {
		t.Errorf("Zustand %q — die Bescheinigung läuft noch über ein Jahr", got)
	}
	plain := env.vendor(t, "Ohne Bescheinigung GmbH", "DE", "")
	if got := plain.ExemptionCertificateState("2026-03-20"); got != "" {
		t.Errorf("Zustand %q — ohne erfasste Bescheinigung gibt es keinen", got)
	}
}
