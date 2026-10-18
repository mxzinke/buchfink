package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/vatid"
)

// -------------------------------------------------------------------------
// Folgebefund zur unbestätigten USt-IdNr.
// -------------------------------------------------------------------------

// euServiceRecord legt eine ausgestellte Leistung an einen Unternehmer im
// übrigen Gemeinschaftsgebiet an.
func (e *testEnv) euServiceRecord(t *testing.T, customer *domain.Contact, number, overrideReason string) *domain.Invoice {
	t.Helper()
	invoice := &domain.Invoice{
		FiscalYear: 2026, InvoiceNumber: number, Date: "2026-04-02",
		ServiceDateFrom: "2026-04-01", ServiceDateTo: "2026-04-01",
		ContactID: customer.ID, ContactName: customer.Name,
		TaxTreatment:        domain.TaxTreatmentReverseChargeSupply,
		VatIDOverrideReason: overrideReason,
		Status:              domain.InvoiceStatusIssued, Currency: "EUR",
		NetAmount: 500_000, GrossAmount: 500_000,
	}
	if err := repository.NewInvoiceRepository(e.db).Save(context.Background(), invoice); err != nil {
		t.Fatalf("Rechnung %s: %v", number, err)
	}
	return invoice
}

// Der Prüflauf meldet die Rechnung, die ohne bestätigte USt-IdNr. ausgestellt
// wurde — der zugesagte Folgebefund zur Übersteuerung bei Offline-BZSt. Eine
// Rechnung ohne Übersteuerung ging mit einer Bestätigung hinaus und bleibt
// ohne Befund.
func TestCheckRunReportsUnconfirmedVatIDs(t *testing.T) {
	env := newTestEnv(t)
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euServiceRecord(t, customer, "RE-2026-0001", "Bundeszentralamt nicht erreichbar")
	env.euServiceRecord(t, customer, "RE-2026-0002", "")

	checks := env.checks(t)
	checks.SetVatIDStatusSource(env.vatIDs(t, "https://example.invalid/evatr"))

	run := runChecks(t, checks, "2026-12-31")
	findings := findingsFor(run, domain.CheckRuleVatIDUnconfirmed)
	if len(findings) != 1 {
		t.Fatalf("erwartet einen Befund %q, erhalten %+v", domain.CheckRuleVatIDUnconfirmed, findings)
	}
	unconfirmed := findings[0]
	if unconfirmed.ObjectName != "RE-2026-0001" {
		t.Errorf("Bezugsobjekt %q — der Befund muss die übersteuerte Rechnung benennen", unconfirmed.ObjectName)
	}
	if unconfirmed.Severity != domain.CheckWarning {
		t.Errorf("Schwere %q — die Abfrage darf nachgeholt werden", unconfirmed.Severity)
	}
	if !strings.Contains(unconfirmed.Message, "Bundeszentralamt nicht erreichbar") {
		t.Errorf("der Folgebefund muss den festgehaltenen Grund nennen: %s", unconfirmed.Message)
	}
	if !strings.Contains(unconfirmed.Reference, "§ 3a Abs. 2") {
		t.Errorf("Fundstelle %q — erwartet § 3a Abs. 2 UStG", unconfirmed.Reference)
	}
}

// -------------------------------------------------------------------------
// Der Rechnungsweg gegen das Bundeszentralamt
// -------------------------------------------------------------------------

// Das Ausstellen einer Leistung an einen Unternehmer im übrigen
// Gemeinschaftsgebiet geht durch die Bestätigungsabfrage: bestätigt sie, wird
// die Nummer vergeben; weist sie zurück, nicht; antwortet niemand, nur mit
// festgehaltenem Grund. Geprüft wird hier der Weg über Issue und nicht
// EnsureConfirmed allein — dazwischen liegen prepareForIssue und die Frage, ob
// der Empfänger überhaupt im übrigen Gemeinschaftsgebiet sitzt.
func TestIssuingAnEUServiceGoesThroughTheConfirmation(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Der Ergebniscode wird je Aufruf umgestellt; jede Rechnung geht an einen
	// eigenen Empfänger, damit die Frist von 90 Tagen die zweite Abfrage nicht
	// aus dem Zwischenspeicher beantwortet.
	status := "evatr-0000"
	code := http.StatusOK
	hang := false
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hang {
			<-release
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = fmt.Fprintf(w,
			`{"status":%q,"id":"2026-0001","ergFirmenname":"A","ergOrt":"A"}`, status)
	}))
	defer func() {
		close(release)
		server.Close()
	}()

	vatIDs := env.vatIDs(t, server.URL)
	// Der Ausfall wird über einen Server geprüft, der nicht antwortet. Mit der
	// Voreinstellung von zehn Sekunden wartete der Test genau so lange real;
	// geprüft wird aber die Behandlung der Frist und nicht ihre Länge.
	vatIDs.SetClientFactory(func() VatIDClient {
		client := vatid.New(server.URL)
		client.SetTimeout(200 * time.Millisecond)
		return client
	})
	invoices := env.invoicesWired(t)
	invoices.SetVatIDConfirmer(vatIDs)

	issue := func(customer *domain.Contact, date, reason string) (*domain.Invoice, error) {
		inv := env.simpleInvoice(customer.ID, date, 500_000)
		inv.TaxTreatment = domain.TaxTreatmentReverseChargeSupply
		inv.Items[0].TaxRate = domain.TaxRateNone
		inv.VatIDOverrideReason = reason
		// Wie im Kern: gefragt wird vor dem Ausstellen, nicht darin.
		if id, err := invoices.VatIDToConfirm(ctx, inv); err != nil {
			return inv, err
		} else if id != 0 {
			_, _ = vatIDs.Check(ctx, id)
		}
		return inv, invoices.Issue(ctx, inv)
	}

	confirmed, err := issue(env.customer(t, "Client SARL", "FR", "FR12345678901"), "2026-03-01", "")
	if err != nil {
		t.Fatalf("mit bestätigter USt-IdNr. muss Buchfink die Rechnung ausstellen: %v", err)
	}
	if confirmed.InvoiceNumber == "" {
		t.Error("die bestätigte Leistung bekommt ihre Nummer")
	}

	// Das Amt weist die Nummer zurück: keine Nummer, keine Rechnung.
	status, code = "evatr-2001", http.StatusNotFound
	rejected, err := issue(env.customer(t, "Client Nord SARL", "FR", "FR99999999999"), "2026-03-02", "")
	if err == nil {
		t.Fatal("eine zurückgewiesene USt-IdNr. ermöglicht keine Leistung mit Steuerschuld des Empfängers")
	}
	if !strings.Contains(err.Error(), "evatr-2001") {
		t.Errorf("die Meldung muss den Ergebniscode nennen: %v", err)
	}
	if rejected.InvoiceNumber != "" {
		t.Errorf("Nummer %q — die abgelehnte Rechnung darf keine verbrauchen", rejected.InvoiceNumber)
	}

	// Und ein Amt, das nicht antwortet, ist kein negatives Ergebnis: Buchfink
	// stellt die Rechnung nur mit einem festgehaltenen Grund aus.
	hang = true
	blocked := env.customer(t, "Client Süd SARL", "FR", "FR11111111111")
	if _, err := issue(blocked, "2026-03-03", ""); err == nil {
		t.Fatal("ohne Antwort und ohne Grund stellt Buchfink die Rechnung nicht aus")
	} else if strings.Contains(err.Error(), "nicht bestätigt") {
		t.Errorf("eine ausgebliebene Antwort ist kein negatives Ergebnis: %v", err)
	}
	withReason, err := issue(blocked, "2026-03-03",
		"Bundeszentralamt antwortet nicht, Nummer aus dem Vorjahr bestätigt")
	if err != nil {
		t.Fatalf("mit festgehaltenem Grund muss Buchfink die Rechnung ausstellen: %v", err)
	}
	if withReason.InvoiceNumber == "" {
		t.Error("die übersteuerte Leistung bekommt ihre Nummer")
	}

	// Ein Empfänger im Drittland wird nicht abgefragt: die Bestätigung kennt
	// nur Nummern des übrigen Gemeinschaftsgebiets.
	hang = false
	swiss := env.customer(t, "Kunde AG", "CH", "CHE123456789")
	if _, err := issue(swiss, "2026-03-04", ""); err != nil {
		t.Errorf("eine Leistung ins Drittland braucht keine Bestätigung: %v", err)
	}
}

// -------------------------------------------------------------------------
// Der Folgebefund endet mit der nachgeholten Bestätigung
// -------------------------------------------------------------------------

// Wer die Rechnung mit einem Grund ausgestellt und die Abfrage danach nachgeholt
// hat, hat den Befund erledigt. Vorher meldete der Prüflauf weiter, es liege
// keine gültige Bestätigung vor — und hängte im selben Satz an, die Nummer sei
// bestätigt worden.
func TestConfirmedVatIDEndsTheFollowUpFinding(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euServiceRecord(t, customer, "RE-2026-0001", "Bundeszentralamt nicht erreichbar")

	checks := env.checks(t)
	checks.SetVatIDStatusSource(env.vatIDs(t, "https://example.invalid/evatr"))

	// Ohne Bestätigung steht der Folgebefund.
	run := runChecks(t, checks, "2026-12-31")
	if findingByRule(run, domain.CheckRuleVatIDUnconfirmed) == nil {
		t.Fatalf("ohne Bestätigung muss der Folgebefund stehen: %+v", run.Findings)
	}

	// Die Abfrage wird nachgeholt.
	if err := repository.NewVatIDCheckRepository(env.db).Save(ctx, &domain.VatIDCheck{
		ContactID: customer.ID, VatID: "FR12345678901",
		CheckedAt: time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		Status:    domain.VatIDValid, ResultCode: "200", RequestID: "BZST-4711",
	}); err != nil {
		t.Fatalf("Bestätigung speichern: %v", err)
	}

	after := runChecks(t, checks, "2026-12-31")
	if found := findingByRule(after, domain.CheckRuleVatIDUnconfirmed); found != nil {
		t.Errorf("nach der nachgeholten Bestätigung darf der Befund nicht mehr stehen: %s",
			found.Message)
	}
}

// findingByRule sucht den ersten Befund einer Regel.
func findingByRule(run *domain.CheckRun, rule string) *domain.CheckFinding {
	for i := range run.Findings {
		if run.Findings[i].Rule == rule {
			return &run.Findings[i]
		}
	}
	return nil
}

// -------------------------------------------------------------------------
// USt-IdNr.: die eigene Nummer ist kein Netzproblem
// -------------------------------------------------------------------------

// Fehlt die eigene USt-IdNr., kommt die Bestätigungsanfrage nach § 18e UStG
// gar nicht zustande. Dieser Fall darf nicht wie ein nicht erreichbares
// Bundeszentralamt behandelt werden: er ließe sich sonst mit einem Grund
// übersteuern, obwohl der Mangel in den eigenen Stammdaten liegt und ohne Netz
// zu beheben ist.
func TestMissingOwnVatIDIsNotAnOverridableOutage(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, calls := bzstServer(t, "200", map[string]string{
		"ergName": "A", "ergOrt": "A", "ergPlz": "A", "ergStr": "A",
	})
	svc := env.vatIDs(t, server.URL)
	env.setSettings(t, func(c *domain.CompanySettings) { c.VatID = "" })
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")

	err := svc.EnsureConfirmed(ctx, customer, "")
	if err == nil {
		t.Fatal("ohne eigene USt-IdNr. gibt es keine Bestätigung")
	}
	if !strings.Contains(err.Error(), "Stammdaten") {
		t.Errorf("die Meldung muss auf die Stammdaten verweisen: %v", err)
	}

	// Und ein Grund hilft hier nicht: die Abfrage ist nicht fehlgeschlagen,
	// sondern noch nicht möglich.
	if err := svc.EnsureConfirmed(ctx, customer,
		"Bundeszentralamt nicht erreichbar"); err == nil {
		t.Error("der fehlende Stammdatensatz ist nicht übersteuerbar")
	}
	if *calls != 0 {
		t.Errorf("%d Aufrufe — eine Anfrage ohne eigene USt-IdNr. wird gar nicht erst gestellt",
			*calls)
	}

	// Mit der eigenen Nummer läuft die Abfrage wieder.
	env.setSettings(t, func(c *domain.CompanySettings) { c.VatID = "DE123456789" })
	if needed, err := svc.NeedsCheck(ctx, customer); err != nil || !needed {
		t.Fatalf("mit eigener USt-IdNr. ist die Anfrage fällig: %v, %v", needed, err)
	}
	if _, err := svc.Check(ctx, customer.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.EnsureConfirmed(ctx, customer, ""); err != nil {
		t.Errorf("mit eigener USt-IdNr. muss die Bestätigung durchgehen: %v", err)
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

// vatIDs baut den Dienst gegen eine Testadresse statt gegen das
// Bundeszentralamt.
func (e *testEnv) vatIDs(t *testing.T, endpoint string) *VatIDService {
	t.Helper()
	svc := NewVatIDService(
		repository.NewVatIDCheckRepository(e.db), e.contactRepo,
		repository.NewSettingsRepository(e.db), repository.NewAuditRepository(e.db),
	)
	svc.SetClientFactory(func() VatIDClient { return vatid.New(endpoint) })
	return svc
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

	if needed, err := svc.NeedsCheck(ctx, customer); err != nil || needed {
		t.Errorf("innerhalb der Frist ist keine neue Anfrage nötig: %v, %v", needed, err)
	}

	// Nach 91 Tagen genügt die alte Bestätigung nicht mehr; die Prüfung vor
	// dem Ausstellen fragt nicht selbst, sondern verlangt eine neue Anfrage.
	later := func() time.Time { return time.Now().AddDate(0, 0, 91) }
	svc.SetClock(later)
	svc.SetClientFactory(func() VatIDClient {
		client := vatid.New(server.URL)
		client.SetClock(later)
		return client
	})
	if needed, err := svc.NeedsCheck(ctx, customer); err != nil || !needed {
		t.Fatalf("nach Ablauf der Frist ist eine neue Anfrage nötig: %v, %v", needed, err)
	}
	if err := svc.EnsureConfirmed(ctx, customer, ""); err == nil {
		t.Fatal("eine abgelaufene Bestätigung darf nicht genügen")
	}
	if *calls != 1 {
		t.Errorf("%d Aufrufe — die Prüfung vor dem Ausstellen ruft das Netz nicht", *calls)
	}
	if _, err := svc.Check(ctx, customer.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.EnsureConfirmed(ctx, customer, ""); err != nil {
		t.Fatalf("nach der neuen Anfrage muss die Bestätigung genügen: %v", err)
	}
	if *calls != 2 {
		t.Errorf("%d Aufrufe — nach Ablauf der Frist wird erneut gefragt", *calls)
	}
}

// Ergebniscode 201: die Nummer ist ungültig. Eine Leistung mit Steuerschuld
// dieses Empfängers wird abgelehnt, und die Ablehnung nennt das Ergebnis.
func TestVatIDRejectionBlocksTheEUService(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	server, _ := bzstServer(t, "201", nil)
	svc := env.vatIDs(t, server.URL)
	customer := env.customer(t, "Client SARL", "FR", "FR99999999999")
	if _, err := svc.Check(ctx, customer.ID); err != nil {
		t.Fatalf("eine negative Antwort ist eine Antwort: %v", err)
	}

	err := svc.EnsureConfirmed(ctx, customer, "")
	if err == nil {
		t.Fatal("eine nicht bestätigte USt-IdNr. ermöglicht keine Leistung mit Steuerschuld des Empfängers")
	}
	if !strings.Contains(err.Error(), "§ 3a Abs. 2") {
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
// Buchfink stellt die Rechnung dann nur mit einem festgehaltenen Grund aus.
func TestVatIDUnavailableNeedsAnOverride(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	// Ein Server, der sofort geschlossen wird: die Verbindung scheitert.
	server, _ := bzstServer(t, "200", nil)
	url := server.URL
	server.Close()

	svc := env.vatIDs(t, url)
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	if _, err := svc.Check(ctx, customer.ID); err == nil {
		t.Fatal("ein nicht erreichbares Amt liefert kein Ergebnis")
	}

	err := svc.EnsureConfirmed(ctx, customer, "")
	if err == nil {
		t.Fatal("ohne Bestätigung und ohne Grund stellt Buchfink die Rechnung nicht aus")
	}
	if strings.Contains(err.Error(), "nicht bestätigt") {
		t.Errorf("eine ausgebliebene Antwort ist kein negatives Ergebnis: %v", err)
	}

	if err := svc.EnsureConfirmed(ctx, customer,
		"Bundeszentralamt nicht erreichbar, Nummer aus dem Vorjahr bestätigt"); err != nil {
		t.Fatalf("mit Grund muss Buchfink die Rechnung ausstellen: %v", err)
	}
}
