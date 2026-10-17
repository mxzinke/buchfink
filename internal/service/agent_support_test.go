package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func agentContext(t *testing.T) context.Context {
	t.Helper()
	channel, err := actor.Agent("claude-code")
	if err != nil {
		t.Fatal(err)
	}
	return actor.WithChannel(context.Background(), channel)
}

var appContext = actor.WithChannel(context.Background(), actor.ChannelUI)

// Ein Agent schlägt die Standard-Buchungsgruppe vor; bestätigen kann nur der
// Mensch, und kein Speichern der Stammdaten ändert sie nebenbei.
func TestDefaultPostingGroupIsConfirmedOnlyInTheApp(t *testing.T) {
	env := newTestEnv(t)
	agent := agentContext(t)
	vendor := domain.Contact{Type: domain.ContactTypeVendor, Name: "Netzwerk GmbH",
		DefaultPostingGroup: "software", DefaultPostingGroupConfirmedAt: "2026-01-01T00:00:00Z",
		ProposedPostingGroup: "software", ProposedPostingGroupNote: "drei Rechnungen"}
	if err := env.contacts.SaveContact(agent, &vendor); err != nil {
		t.Fatal(err)
	}
	if vendor.DefaultPostingGroup != "" || vendor.DefaultPostingGroupConfirmedAt != "" {
		t.Errorf("ein Agent legt keine bestätigte Gruppe an: %+v", vendor)
	}
	if vendor.ProposedPostingGroup != "software" || vendor.ProposedPostingGroupBy != "agent:claude-code" {
		t.Errorf("Vorschlag beim Anlegen: %+v", vendor)
	}

	if _, err := env.contacts.ConfirmDefaultPostingGroup(agent, vendor.ID, "software"); !errors.Is(err, actor.ErrAppOnly) {
		t.Fatalf("Bestätigen durch den Agenten: %v", err)
	}
	if _, err := env.contacts.ProposeDefaultPostingGroup(agent, vendor.ID, "erloese", ""); err == nil {
		t.Error("eine Erlösgruppe ist keine Standardgruppe eines Lieferanten")
	}
	confirmed, err := env.contacts.ConfirmDefaultPostingGroup(appContext, vendor.ID, "software")
	if err != nil {
		t.Fatal(err)
	}
	if confirmed.DefaultPostingGroup != "software" || confirmed.DefaultPostingGroupConfirmedAt == "" ||
		confirmed.ProposedPostingGroup != "" {
		t.Errorf("bestätigt: %+v", confirmed)
	}

	// Das Formular kennt die Felder nicht und schickt sie leer.
	form := *confirmed
	form.DefaultPostingGroup, form.DefaultPostingGroupConfirmedAt = "", ""
	form.City = "Hamburg"
	if err := env.contacts.SaveContact(appContext, &form); err != nil {
		t.Fatal(err)
	}
	stored, _ := env.contactRepo.FindByID(context.Background(), vendor.ID)
	if stored.DefaultPostingGroup != "software" || stored.City != "Hamburg" {
		t.Errorf("Speichern der Stammdaten hat die Gruppe geändert: %+v", stored)
	}
	// Und ein Agent ändert sie über das Speichern ebenso wenig.
	form.DefaultPostingGroup = "werbung"
	if err := env.contacts.SaveContact(agent, &form); err != nil {
		t.Fatal(err)
	}
	stored, _ = env.contactRepo.FindByID(context.Background(), vendor.ID)
	if stored.DefaultPostingGroup != "software" {
		t.Errorf("ein Agent hat die bestätigte Gruppe überschrieben: %s", stored.DefaultPostingGroup)
	}
}

func TestAgentSettingsAreChangedOnlyInTheApp(t *testing.T) {
	env := newTestEnv(t)
	svc := NewSettingsService(repository.NewSettingsRepository(env.db), repository.NewAuditRepository(env.db))
	got, err := svc.AgentSettings(context.Background())
	if err != nil || got.DirectMaxGross != 0 || got.ReceiptFolders != nil || got.ExportFolder != "" {
		t.Fatalf("ohne Einstellung gilt die Voreinstellung: %+v %v", got, err)
	}
	if _, err := svc.SaveAgentSettings(agentContext(t), domain.AgentSettings{DirectMaxGross: 99999999}); !errors.Is(err, actor.ErrAppOnly) {
		t.Fatalf("ein Agent ändert die Direktgrenze: %v", err)
	}
	if _, err := svc.SaveAgentSettings(appContext, domain.AgentSettings{ReceiptFolders: []string{"relativ"}}); err == nil {
		t.Error("ein relativer Belegordner wird abgewiesen")
	}
	saved, err := svc.SaveAgentSettings(appContext, domain.AgentSettings{
		DirectMaxGross: 50000, DirectMinBookings: 5, ReceiptFolders: []string{}, ExportFolder: "/tmp/export",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.DirectMaxGross != 50000 || saved.DirectMinBookings != 5 || saved.ReceiptFolders == nil ||
		len(saved.ReceiptFolders) != 0 || saved.ExportFolder != "/tmp/export" {
		t.Errorf("gespeichert: %+v", saved)
	}
	back, err := svc.SaveAgentSettings(appContext, domain.AgentSettings{})
	if err != nil || back.DirectMaxGross != 0 || back.ReceiptFolders != nil || back.ExportFolder != "" {
		t.Errorf("zurück zur Voreinstellung: %+v %v", back, err)
	}
}

type fixedOpenItems []domain.OpenItem

func (f fixedOpenItems) OpenItems(context.Context) ([]domain.OpenItem, error) { return f, nil }

// Eine Mahnung nach Fälligkeit setzt den Verbraucher in Verzug, auch wenn
// die Rechnung keinen Hinweis auf die dreißig Tage trug.
func TestReminderStartsTheDefault(t *testing.T) {
	env := newTestEnv(t)
	consumer := domain.Contact{Type: domain.ContactTypeCustomer, Name: "Erika Muster", IsPrivate: true}
	if err := env.contacts.SaveContact(appContext, &consumer); err != nil {
		t.Fatal(err)
	}
	today := time.Now().Format("2006-01-02")
	due := time.Now().AddDate(0, 0, -20).Format("2006-01-02")
	items := fixedOpenItems{{
		EntryID: 41, DocumentNumber: "RE-2026-0041", ContactID: consumer.ID, ContactName: consumer.Name,
		ContactType: domain.ContactTypeCustomer, DueDate: due, GrossAmount: 119000, OpenAmount: 119000,
	}, {
		EntryID: 42, DocumentNumber: "ER-2026-0042", ContactType: domain.ContactTypeVendor, DueDate: due, OpenAmount: 5000,
	}}
	overdue := NewOverdueService(items, env.contactRepo)
	overdue.SetInvoiceSource(repository.NewInvoiceRepository(env.db))
	overdue.SetDunning(repository.NewDunningRepository(env.db), repository.NewAuditRepository(env.db))

	// Eine Rechnung, die Buchfink nicht kennt, gilt als mit Hinweis; für den
	// Test zählt nur, dass der Vermerk an der Zeile steht.
	if _, err := overdue.RecordReminder(agentContext(t), ReminderRequest{EntryID: 42, Date: today}); err == nil {
		t.Error("eine Verbindlichkeit wird nicht gemahnt")
	}
	if _, err := overdue.RecordReminder(agentContext(t), ReminderRequest{EntryID: 41,
		Date: time.Now().AddDate(0, 0, -30).Format("2006-01-02")}); err == nil {
		t.Error("eine Mahnung vor Fälligkeit setzt nicht in Verzug")
	}
	reminded := time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	note, err := overdue.RecordReminder(agentContext(t), ReminderRequest{EntryID: 41, Date: reminded, Level: 1, Note: "per E-Mail"})
	if err != nil {
		t.Fatal(err)
	}
	if note.Channel != "agent:claude-code" {
		t.Errorf("Erfassungsweg: %s", note.Channel)
	}
	claims, err := overdue.Claims(context.Background(), today)
	if err != nil || len(claims) != 1 || len(claims[0].Items) != 1 {
		t.Fatalf("Forderungen: %+v %v", claims, err)
	}
	row := claims[0].Items[0]
	wantFrom := time.Now().AddDate(0, 0, -9).Format("2006-01-02")
	if row.LastReminder != reminded || row.ReminderLevel != 1 || row.DefaultFrom != wantFrom || row.Interest <= 0 {
		t.Errorf("Verzug nach Mahnung: %+v", row)
	}
}
