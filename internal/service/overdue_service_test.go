package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Überfällige Kundenforderungen: Verzugsbeginn, Zinsen und Pauschale.

// stubRenderer setzt jedes Dokument in dieselben Bytes.
//
// Der echte Setzer startet einen Compiler; ihn im Test zu benutzen prüfte
// Typst und nicht die Ablage des erzeugten Dokuments.
type stubRenderer struct {
	documents int
	lastText  string
}

func (r *stubRenderer) RenderDocumentPDF(_ context.Context, template, _ string) ([]byte, error) {
	r.documents++
	r.lastText = template
	return []byte("%PDF-1.7\n% Testdokument\n"), nil
}

// openReceivable bucht eine Ausgangsrechnung auf das Personenkonto und liefert
// die Buchung.
func (e *testEnv) openReceivable(
	t *testing.T, customer *domain.Contact, gross domain.Cents, documentDate, dueDate, number string,
) *domain.JournalEntry {
	t.Helper()
	contactID := customer.ID
	entry, err := e.journal.Post(context.Background(), &domain.JournalEntry{
		FiscalYear: e.fiscalYear, BookingDate: documentDate, DocumentDate: documentDate,
		ServiceDateFrom: documentDate, ServiceDateTo: documentDate,
		Description: "Ausgangsrechnung " + number, Source: domain.EntrySourceManual, TaxTreatment: domain.TaxTreatmentNotTaxable,
		DocumentNumber: number, DueDate: dueDate, ContactID: &contactID,
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: customer.LedgerAccount, Amount: gross},
			{Position: 2, Side: domain.SideCredit, Account: "4400", Amount: gross},
		},
	})
	if err != nil {
		t.Fatalf("Ausgangsrechnung %s buchen: %v", number, err)
	}
	return entry
}

// overdue baut die Verzugsrechnung über der Testumgebung.
func (e *testEnv) overdue(t *testing.T) *OverdueService {
	t.Helper()
	svc := NewOverdueService(e.payments(t), e.contactRepo)
	// Wie in der Anwendung: die Rechnungen haben das Kennzeichen des
	// Verzugshinweises an einen Verbraucher.
	svc.SetInvoiceSource(repository.NewInvoiceRepository(e.db))
	return svc
}

// Taggenau: 10.000 € über 45 Tage bei 1,27 % Basiszins und neun
// Prozentpunkten sind 126,62 € Zinsen; dazu die Pauschale von 40 €.
//
// Die Fälligkeit ist der 31.01.2026, der Verzug beginnt am 03.03.2026
// (dreißig Tage danach plus einen), und der Stichtag 17.04.2026 liegt 45 Tage
// später.
func TestOverdueClaimComputesInterestAndLumpSum(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Saumseligkeit GmbH", "DE", "")
	env.openReceivable(t, customer, 1_000_000, "2026-01-17", "2026-01-31", "RE-2026-0001")

	claims, err := env.overdue(t).Claims(ctx, "2026-04-17")
	if err != nil {
		t.Fatalf("Forderungen: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("erwartet eine Forderung, erhalten %d", len(claims))
	}
	c := claims[0]
	if len(c.Items) != 1 {
		t.Fatalf("erwartet einen Posten, erhalten %d", len(c.Items))
	}
	item := c.Items[0]
	if item.DefaultFrom != "2026-03-03" {
		t.Errorf("Verzugsbeginn = %s, erwartet 2026-03-03", item.DefaultFrom)
	}
	if item.DaysOverdue != 76 {
		t.Errorf("Tage seit Fälligkeit = %d, erwartet 76", item.DaysOverdue)
	}
	if item.InterestDays != 45 {
		t.Errorf("verzinste Tage = %d, erwartet 45", item.InterestDays)
	}
	if item.Interest != 12662 {
		t.Errorf("Zinsen = %s €, erwartet 126,62", item.Interest)
	}
	if c.LumpSum != 4000 {
		t.Errorf("Pauschale = %s €, erwartet 40,00 (§ 288 Abs. 5 BGB)", c.LumpSum)
	}
	if c.Total != 1_000_000+12662+4000 {
		t.Errorf("Gesamtbetrag = %s €, erwartet Hauptforderung, Zinsen und Pauschale", c.Total)
	}
	if !strings.Contains(c.Note, "nicht gebucht") {
		t.Errorf("der Hinweis muss sagen, dass Zinsen und Pauschale nicht gebucht werden: %q", c.Note)
	}
}

// Reicht der Verzug über die hinterlegte Tabelle des Basiszinssatzes hinaus,
// rechnet Buchfink vorläufig mit dem letzten Satz weiter.
func TestOverdueInterestContinuesProvisionally(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Langzahler GmbH", "DE", "")
	env.openReceivable(t, customer, 1_000_000, "2026-01-17", "2026-01-31", "RE-2026-0009")

	covered := accounting.BaseRatesCoveredUntil(accounting.DefaultBaseRates())
	end, err := time.Parse("2006-01-02", covered)
	if err != nil {
		t.Fatalf("Ende der Tabelle %q: %v", covered, err)
	}
	cutoff := end.AddDate(0, 3, 0).Format("2006-01-02")

	claims, err := env.overdue(t).Claims(ctx, cutoff)
	if err != nil {
		t.Fatalf("Forderungen: %v", err)
	}
	if len(claims) != 1 || len(claims[0].Items) != 1 {
		t.Fatalf("erwartet eine Forderung mit einem Posten, erhalten %+v", claims)
	}
	item := claims[0].Items[0]
	if item.Interest <= 0 {
		t.Error("bis zum Ende der Tabelle sind Zinsen zu rechnen")
	}
	if !strings.Contains(item.Note, "vorläufig") {
		t.Errorf("der Hinweis muss sagen, dass ab dem Ende der Tabelle vorläufig gerechnet ist: %q", item.Note)
	}
	if item.LumpSum != 4000 {
		t.Errorf("Pauschale = %s €, erwartet 40,00", item.LumpSum)
	}
}

// Gegenüber einem Verbraucher gilt der niedrigere Satz, und die Pauschale
// entfällt (§ 288 Abs. 1 und 5 BGB).
func TestOverdueConsumerHasNoLumpSumAndTheLowerRate(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Erika Mustermann", "DE", "")
	customer.IsPrivate = true
	if err := env.contacts.SaveContact(ctx, customer); err != nil {
		t.Fatalf("Kunde speichern: %v", err)
	}
	env.openReceivable(t, customer, 1_000_000, "2026-01-17", "2026-01-31", "RE-2026-0002")

	claims, err := env.overdue(t).Claims(ctx, "2026-04-17")
	if err != nil {
		t.Fatalf("Forderungen: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("erwartet eine Forderung, erhalten %d", len(claims))
	}
	c := claims[0]
	if !c.IsConsumer {
		t.Error("die Forderung muss den Kunden als Verbraucher führen")
	}
	if c.LumpSum != 0 {
		t.Errorf("Pauschale = %s €, erwartet 0 gegenüber einem Verbraucher", c.LumpSum)
	}
	if c.Items[0].Interest != 7730 {
		t.Errorf("Zinsen = %s €, erwartet 77,30 (fünf Prozentpunkte)", c.Items[0].Interest)
	}
	// Der Verzug ohne Mahnung tritt gegenüber einem Verbraucher nur ein, wenn
	// die Rechnung darauf hingewiesen hat (§ 286 Abs. 3 Satz 1 Halbsatz 2 BGB).
	if !strings.Contains(c.Note, "§ 286 Abs. 3 Satz 1 Halbsatz 2 BGB") {
		t.Errorf("die Forderung verschweigt die Voraussetzung des Verzugseintritts: %q", c.Note)
	}
	if !strings.Contains(c.Note, "Teilzahlung") {
		t.Errorf("die Forderung nennt die Vereinfachung bei Teilzahlungen nicht: %q", c.Note)
	}
}

// Ein Posten, dessen Fälligkeit nicht verstrichen ist, ist nicht überfällig.
func TestOverdueSkipsItemsNotYetDue(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Pünktlich GmbH", "DE", "")
	env.openReceivable(t, customer, 100_000, "2026-01-17", "2026-01-31", "RE-2026-0003")

	claims, err := env.overdue(t).Claims(ctx, "2026-01-31")
	if err != nil {
		t.Fatalf("Forderungen: %v", err)
	}
	if len(claims) != 0 {
		t.Errorf("erwartet keine Forderung am Tag der Fälligkeit, erhalten %d", len(claims))
	}
}

// Gegenüber einem Verbraucher entsteht die Pauschale auch nach Verzugseintritt
// nicht (§ 288 Abs. 5 Satz 1 BGB).
func TestOverdueLumpSumStaysAwayFromConsumersAfterDefault(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Hans Beispiel", "DE", "")
	customer.IsPrivate = true
	if err := env.contacts.SaveContact(ctx, customer); err != nil {
		t.Fatalf("Kunde speichern: %v", err)
	}
	env.openReceivable(t, customer, 1_000_000, "2026-01-17", "2026-01-31", "RE-2026-0008")

	claims, err := env.overdue(t).Claims(ctx, "2026-03-07")
	if err != nil {
		t.Fatalf("Forderungen: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("erwartet eine Forderung, erhalten %d", len(claims))
	}
	if claims[0].Items[0].InterestDays == 0 {
		t.Fatal("der Verzug muss zum 07.03. eingetreten sein, sonst prüft der Test nichts")
	}
	if claims[0].LumpSum != 0 {
		t.Errorf("Pauschale = %s €, erwartet 0 gegenüber einem Verbraucher", claims[0].LumpSum)
	}
}

// errStubContactMissing ist der Lesefehler, den der Kontaktstub liefert.
var errStubContactMissing = errors.New("die Kontaktkartei antwortet nicht")

// stubLiveOpenItems ist eine offene-Posten-Liste, die der Test selbst setzt.
//
// Sie erlaubt einen Fälligkeitstag vor der Basiszinstabelle — über den Weg der
// gebuchten Rechnung ginge das nicht, weil das Journal ein Datum außerhalb des
// Geschäftsjahres abweist.
type stubLiveOpenItems []domain.OpenItem

func (s stubLiveOpenItems) OpenItems(context.Context) ([]domain.OpenItem, error) {
	return []domain.OpenItem(s), nil
}

// stubContacts liefert zu jeder Kennung denselben Fehler.
type stubContacts struct{ err error }

func (s *stubContacts) FindAll(context.Context) ([]domain.Contact, error) { return nil, s.err }
func (s *stubContacts) FindByID(context.Context, uint) (*domain.Contact, error) {
	return nil, s.err
}
func (s *stubContacts) FindByLedgerAccount(context.Context, string) (*domain.Contact, error) {
	return nil, s.err
}
func (s *stubContacts) Save(context.Context, *domain.Contact) error { return s.err }
func (s *stubContacts) Delete(context.Context, uint) error          { return s.err }
func (s *stubContacts) Count(context.Context) (int64, error)        { return 0, s.err }

// Ein fehlender Basiszinssatz bleibt nicht stumm.
//
// Ohne Satz für den Zeitraum sind die Zinsen null — die Forderung fiele zu
// niedrig aus, und niemand sähe, warum. Der Grund gehört an den Posten und in
// den Hinweis.
func TestOverdueNamesTheMissingBaseRate(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := NewOverdueService(
		stubLiveOpenItems{{
			EntryID: 4711, EntryNumber: "2015-0001", ContactID: 1, ContactName: "Altfall GmbH",
			ContactType: domain.ContactTypeCustomer, DocumentNumber: "RE-2015-0001",
			DocumentDate: "2015-01-02", DueDate: "2015-01-31",
			GrossAmount: 1_000_000, OpenAmount: 1_000_000,
		}},
		env.contactRepo,
	)

	claims, err := svc.Claims(ctx, "2015-06-01")
	if err != nil {
		t.Fatalf("Forderungen: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("erwartet eine Forderung, erhalten %d", len(claims))
	}
	c := claims[0]
	if c.Interest != 0 || c.Items[0].Note == "" {
		t.Fatalf("erwartet null Zinsen mit Begründung, erhalten %s € und %q",
			c.Interest, c.Items[0].Note)
	}
	if !strings.Contains(c.Items[0].Note, "Basiszinssatz") {
		t.Errorf("der Hinweis nennt den fehlenden Basiszinssatz nicht: %q", c.Items[0].Note)
	}
	if !strings.Contains(c.Note, "Basiszinssatz") {
		t.Errorf("die Forderung enthält den Hinweis nicht: %q", c.Note)
	}
	// Keine Pauschale: die Stammdaten dieses Kunden fehlen, und er wird
	// deshalb vorsichtshalber als Verbraucher gerechnet.
	if c.LumpSum != 0 {
		t.Errorf("Pauschale = %s €, erwartet 0 gegenüber einem Verbraucher", c.LumpSum)
	}
}

// Die Pauschale richtet sich nach dem Verzug, nicht nach dem Gelingen der
// Zinsrechnung (§ 288 Abs. 5 BGB).
func TestOverdueLumpSumSurvivesTheMissingBaseRate(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Altfall GmbH", "DE", "")
	svc := NewOverdueService(
		stubLiveOpenItems{{
			EntryID: 4711, EntryNumber: "2015-0001",
			ContactID: customer.ID, ContactName: customer.Name,
			ContactType: domain.ContactTypeCustomer, DocumentNumber: "RE-2015-0001",
			DocumentDate: "2015-01-02", DueDate: "2015-01-31",
			GrossAmount: 1_000_000, OpenAmount: 1_000_000,
		}},
		env.contactRepo,
	)

	claims, err := svc.Claims(ctx, "2015-06-01")
	if err != nil {
		t.Fatalf("Forderungen: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("erwartet eine Forderung, erhalten %d", len(claims))
	}
	c := claims[0]
	if c.Interest != 0 {
		t.Fatalf("erwartet null Zinsen ohne Basiszinssatz, erhalten %s €", c.Interest)
	}
	if c.LumpSum != 4000 {
		t.Errorf("Pauschale = %s €, erwartet 40,00 trotz fehlender Zinsrechnung", c.LumpSum)
	}
	if c.Total != 1_000_000+4000 {
		t.Errorf("Gesamtbetrag = %s €, erwartet Hauptforderung und Pauschale", c.Total)
	}
	note := c.Items[0].Note
	if !strings.Contains(note, "Basiszinssatz") || !strings.Contains(note, "Pauschale") {
		t.Errorf("der Hinweis nennt nicht beides — fehlende Zinsen und angesetzte Pauschale: %q", note)
	}
}

// Vor dem Verzugsbeginn gibt es weder Zinsen noch Pauschale: der Verzug tritt
// erst dreißig Tage nach Fälligkeit ein (§ 286 Abs. 3 BGB).
func TestOverdueLumpSumWaitsForTheDefaultToBegin(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Fruehmahnung GmbH", "DE", "")
	env.openReceivable(t, customer, 1_000_000, "2026-01-17", "2026-01-31", "RE-2026-0007")

	claims, err := env.overdue(t).Claims(ctx, "2026-02-10")
	if err != nil {
		t.Fatalf("Forderungen: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("erwartet eine Forderung, erhalten %d", len(claims))
	}
	c := claims[0]
	if c.LumpSum != 0 {
		t.Errorf("Pauschale = %s €, erwartet 0 vor dem Verzugsbeginn", c.LumpSum)
	}
	if c.Interest != 0 {
		t.Errorf("Zinsen = %s €, erwartet 0 vor dem Verzugsbeginn", c.Interest)
	}
}

// Ein Kunde, dessen Stammdaten sich nicht lesen lassen, fällt nicht still aus
// der Liste.
func TestOverdueKeepsCustomerWithUnreadableContact(t *testing.T) {
	ctx := context.Background()
	svc := NewOverdueService(
		stubLiveOpenItems{{
			EntryID: 815, EntryNumber: "2026-0001", ContactID: 99, ContactName: "Verschollen GmbH",
			ContactType: domain.ContactTypeCustomer, DocumentNumber: "RE-2026-0011",
			DocumentDate: "2026-01-17", DueDate: "2026-01-31",
			GrossAmount: 1_000_000, OpenAmount: 1_000_000,
		}},
		&stubContacts{err: errStubContactMissing},
	)

	claims, err := svc.Claims(ctx, "2026-04-17")
	if err != nil {
		t.Fatalf("Forderungen: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("der Kunde darf nicht stillschweigend entfallen, erhalten %d Forderungen", len(claims))
	}
	c := claims[0]
	if !strings.Contains(c.Note, "Stammdaten") {
		t.Errorf("die Forderung nennt die nicht lesbaren Stammdaten nicht: %q", c.Note)
	}
	// Vorsichtshalber der Verbrauchersatz: fünf Prozentpunkte, keine Pauschale.
	if !c.IsConsumer || c.LumpSum != 0 {
		t.Errorf("ohne Stammdaten ist mit dem niedrigeren Satz zu rechnen: %+v", c)
	}
	if c.Items[0].Interest != 7730 {
		t.Errorf("Zinsen = %s €, erwartet 77,30 (fünf Prozentpunkte)", c.Items[0].Interest)
	}
}
