package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Das Mahnwesen: Verzugsbeginn, Zinsen, Pauschale, Stufenfolge und das
// abgelegte Schreiben.

// stubRenderer setzt jedes Dokument in dieselben Bytes.
//
// Der echte Setzer startet einen Compiler; ihn im Test zu benutzen prüfte
// Typst und nicht das Mahnwesen. Was hier geprüft wird, ist die Ablage: dass
// das erzeugte Dokument im Belegspeicher landet und am Schreiben vermerkt ist.
type stubRenderer struct {
	documents int
	lastText  string
}

func (r *stubRenderer) RenderDocumentPDF(_ context.Context, template, _ string) ([]byte, error) {
	r.documents++
	r.lastText = template
	return []byte("%PDF-1.7\n% Mahnschreiben\n"), nil
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
		Description: "Ausgangsrechnung " + number, Source: domain.EntrySourceManual,
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

// dunning baut den Mahnlauf über der Testumgebung.
func (e *testEnv) dunning(t *testing.T, renderer DocumentRenderer) *DunningService {
	t.Helper()
	svc := NewDunningService(
		e.payments(t),
		e.contactRepo,
		repository.NewDunningRepository(e.db),
		repository.NewBaseRateRepository(e.db),
		repository.NewSettingsRepository(e.db),
		repository.NewAuditRepository(e.db),
		e.store,
		e.fiscalYear,
	)
	if renderer != nil {
		svc.SetRenderer(renderer)
	}
	// Wie in der Anwendung: die Rechnungen tragen das Kennzeichen des
	// Verzugshinweises an einen Verbraucher.
	svc.SetInvoiceSource(repository.NewInvoiceRepository(e.db))
	return svc
}

// Der Vorschlag rechnet taggenau: 10.000 € über 45 Tage bei 1,27 % Basiszins
// und neun Prozentpunkten sind 126,62 € Zinsen; dazu die Pauschale von 40 €.
//
// Die Fälligkeit ist der 31.01.2026, der Verzug beginnt am 03.03.2026
// (dreißig Tage danach plus einen), und der Stichtag 17.04.2026 liegt 45 Tage
// später.
func TestDunningProposalComputesInterestAndLumpSum(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Saumseligkeit GmbH", "DE", "")
	env.openReceivable(t, customer, 1_000_000, "2026-01-17", "2026-01-31", "RE-2026-0001")

	proposals, err := env.dunning(t, nil).Proposals(ctx, "2026-04-17")
	if err != nil {
		t.Fatalf("Mahnvorschläge: %v", err)
	}
	if len(proposals) != 1 {
		t.Fatalf("erwartet einen Vorschlag, erhalten %d", len(proposals))
	}
	p := proposals[0]
	if len(p.Items) != 1 {
		t.Fatalf("erwartet einen Posten, erhalten %d", len(p.Items))
	}
	item := p.Items[0]
	if item.DefaultFrom != "2026-03-03" {
		t.Errorf("Verzugsbeginn = %s, erwartet 2026-03-03", item.DefaultFrom)
	}
	if item.InterestDays != 45 {
		t.Errorf("verzinste Tage = %d, erwartet 45", item.InterestDays)
	}
	if item.Interest != 12662 {
		t.Errorf("Zinsen = %s €, erwartet 126,62", item.Interest)
	}
	if p.LumpSum != 4000 {
		t.Errorf("Pauschale = %s €, erwartet 40,00 (§ 288 Abs. 5 BGB)", p.LumpSum)
	}
	if p.Level != 1 || p.LevelLabel != "Zahlungserinnerung" {
		t.Errorf("Stufe = %d (%q), erwartet 1 (Zahlungserinnerung)", p.Level, p.LevelLabel)
	}
	if p.Fee != 0 {
		t.Errorf("Gebühr = %s €, erwartet 0 in der Zahlungserinnerung", p.Fee)
	}
	if p.Total != 1_000_000+12662+4000 {
		t.Errorf("Gesamtbetrag = %s €, erwartet Hauptforderung, Zinsen und Pauschale", p.Total)
	}
	if !strings.Contains(p.Note, "nicht gebucht") {
		t.Errorf("der Hinweis muss sagen, dass Zinsen und Gebühr nicht gebucht werden: %q", p.Note)
	}
}

// Gegenüber einem Verbraucher gilt der niedrigere Satz, und die Pauschale
// entfällt (§ 288 Abs. 1 und 5 BGB).
func TestDunningConsumerHasNoLumpSumAndTheLowerRate(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Erika Mustermann", "DE", "")
	customer.IsPrivate = true
	if err := env.contacts.SaveContact(ctx, customer); err != nil {
		t.Fatalf("Kunde speichern: %v", err)
	}
	env.openReceivable(t, customer, 1_000_000, "2026-01-17", "2026-01-31", "RE-2026-0002")

	proposals, err := env.dunning(t, nil).Proposals(ctx, "2026-04-17")
	if err != nil {
		t.Fatalf("Mahnvorschläge: %v", err)
	}
	if len(proposals) != 1 {
		t.Fatalf("erwartet einen Vorschlag, erhalten %d", len(proposals))
	}
	p := proposals[0]
	if !p.IsConsumer {
		t.Error("der Vorschlag muss den Kunden als Verbraucher führen")
	}
	if p.LumpSum != 0 {
		t.Errorf("Pauschale = %s €, erwartet 0 gegenüber einem Verbraucher", p.LumpSum)
	}
	if p.Items[0].Interest != 7730 {
		t.Errorf("Zinsen = %s €, erwartet 77,30 (fünf Prozentpunkte)", p.Items[0].Interest)
	}
	// Der Verzug ohne Mahnung tritt gegenüber einem Verbraucher nur ein, wenn
	// die Rechnung darauf hingewiesen hat (§ 286 Abs. 3 Satz 1 Halbsatz 2 BGB).
	// Ob sie das getan hat, steht auf dem Formular und nicht in den Daten —
	// verschwiegen forderte das Schreiben Zinsen, die noch nicht laufen.
	if !strings.Contains(p.Note, "§ 286 Abs. 3 Satz 1 Halbsatz 2 BGB") {
		t.Errorf("der Vorschlag verschweigt die Voraussetzung des Verzugseintritts: %q", p.Note)
	}
	// Und die Vereinfachung bei Teilzahlungen steht darunter.
	if !strings.Contains(p.Note, "Teilzahlung") {
		t.Errorf("der Vorschlag nennt die Vereinfachung bei Teilzahlungen nicht: %q", p.Note)
	}
}

// Vor Ablauf der ersten Stufe gibt es nichts zu mahnen.
func TestDunningWaitsForTheFirstLevel(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Pünktlich GmbH", "DE", "")
	env.openReceivable(t, customer, 100_000, "2026-01-17", "2026-01-31", "RE-2026-0003")

	// Drei Tage nach Fälligkeit: die Zahlungserinnerung kommt nach sieben.
	proposals, err := env.dunning(t, nil).Proposals(ctx, "2026-02-03")
	if err != nil {
		t.Fatalf("Mahnvorschläge: %v", err)
	}
	if len(proposals) != 0 {
		t.Errorf("erwartet keinen Vorschlag, erhalten %d", len(proposals))
	}
}

// Die Stufenfolge wird eingehalten: nach der Zahlungserinnerung kommt die erste
// Mahnung, und zwar erst nach dem Abstand, den die Stufen vorsehen.
func TestDunningLevelsFollowInOrder(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	renderer := &stubRenderer{}
	svc := env.dunning(t, renderer)
	customer := env.customer(t, "Langsam GmbH", "DE", "")
	env.openReceivable(t, customer, 500_000, "2026-01-17", "2026-01-31", "RE-2026-0004")

	first, err := svc.Create(ctx, DunningRequest{ContactID: customer.ID, NoticeDate: "2026-04-17"})
	if err != nil {
		t.Fatalf("erstes Schreiben: %v", err)
	}
	if first.Level != 1 {
		t.Errorf("erste Stufe = %d, erwartet 1", first.Level)
	}

	// Am selben Tag noch einmal: der Posten steht schon auf Stufe 1, und der
	// Abstand zur nächsten ist nicht verstrichen.
	same, err := svc.Proposals(ctx, "2026-04-17")
	if err != nil {
		t.Fatalf("Mahnvorschläge: %v", err)
	}
	if len(same) != 0 {
		t.Errorf("am selben Tag darf keine zweite Stufe vorgeschlagen werden, erhalten %d", len(same))
	}

	// Vierzehn Tage später — der Abstand zwischen Stufe 1 (7 Tage) und Stufe 2
	// (21 Tage) — steht die erste Mahnung an.
	later, err := svc.Proposals(ctx, "2026-05-01")
	if err != nil {
		t.Fatalf("Mahnvorschläge: %v", err)
	}
	if len(later) != 1 {
		t.Fatalf("erwartet einen Vorschlag, erhalten %d", len(later))
	}
	if later[0].Level != 2 || later[0].LevelLabel != "1. Mahnung" {
		t.Errorf("Stufe = %d (%q), erwartet 2 (1. Mahnung)", later[0].Level, later[0].LevelLabel)
	}
	if later[0].Fee != 500 {
		t.Errorf("Gebühr = %s €, erwartet 5,00 in der ersten Mahnung", later[0].Fee)
	}
	// Die Pauschale fällt je Forderung einmal an und nicht je Schreiben.
	if later[0].LumpSum != 0 {
		t.Errorf("Pauschale = %s €, erwartet 0 im zweiten Schreiben zur selben Forderung",
			later[0].LumpSum)
	}
}

// Das Schreiben wird als Dokument abgelegt und ist am Datensatz vermerkt.
func TestDunningNoticeIsFiledAsDocument(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	renderer := &stubRenderer{}
	svc := env.dunning(t, renderer)
	customer := env.customer(t, "Zahlungsunwillig GmbH", "DE", "")
	env.openReceivable(t, customer, 238_000, "2026-01-17", "2026-01-31", "RE-2026-0005")

	notice, err := svc.Create(ctx, DunningRequest{ContactID: customer.ID, NoticeDate: "2026-04-17"})
	if err != nil {
		t.Fatalf("Mahnschreiben: %v", err)
	}
	if renderer.documents != 1 {
		t.Fatalf("erwartet ein gesetztes Dokument, erhalten %d", renderer.documents)
	}
	if notice.DocumentPath == "" || notice.DocumentSHA256 == "" {
		t.Fatalf("das Schreiben ist nicht abgelegt: %+v", notice)
	}
	if !strings.Contains(notice.DocumentPath, filepath.Join("dokumente", DunningCategory)) {
		t.Errorf("Ablagepfad = %q, erwartet den Zweig dokumente/%s", notice.DocumentPath, DunningCategory)
	}
	if _, err := os.Stat(filepath.Join(env.dataDir, notice.DocumentPath)); err != nil {
		t.Errorf("die abgelegte Datei fehlt: %v", err)
	}
	// Der Text nennt Posten, Beträge und Frist.
	for _, want := range []string{"RE-2026-0005", "Zahlungserinnerung", "17.04.2026", "01.05.2026"} {
		if !strings.Contains(renderer.lastText, want) {
			t.Errorf("das Schreiben nennt %q nicht", want)
		}
	}
	if notice.DueDate != "2026-05-01" {
		t.Errorf("Zahlungsfrist = %s, erwartet vierzehn Tage nach dem Schreiben", notice.DueDate)
	}

	// Der Verlauf je Kunde liest sich zurück.
	notices, err := svc.Notices(ctx, customer.ID)
	if err != nil {
		t.Fatalf("Mahnverlauf: %v", err)
	}
	if len(notices) != 1 || len(notices[0].Items) != 1 {
		t.Fatalf("Verlauf: %d Schreiben mit %d Posten — erwartet 1 mit 1",
			len(notices), len(notices[0].Items))
	}
	if notices[0].Items[0].Level != 1 {
		t.Errorf("die Stufe je Posten fehlt im Verlauf: %+v", notices[0].Items[0])
	}
}

// Ein Kunde ohne fälligen Posten bekommt kein Schreiben — auch nicht auf
// ausdrückliche Anforderung.
func TestDunningRefusesWithoutAnythingToDun(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.dunning(t, &stubRenderer{})
	customer := env.customer(t, "Zahlt sofort GmbH", "DE", "")

	if _, err := svc.Create(ctx, DunningRequest{ContactID: customer.ID, NoticeDate: "2026-04-17"}); err == nil {
		t.Error("ohne fälligen Posten darf kein Mahnschreiben entstehen")
	}
	if _, err := svc.CreateMany(ctx, DunningRunRequest{}); err == nil {
		t.Error("ein Mahnlauf ohne Auswahl muss abgewiesen werden")
	}
}

// Der nachgetragene Basiszinssatz wirkt auf die Zinsrechnung.
func TestSavedBaseRateChangesTheInterest(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.dunning(t, nil)
	customer := env.customer(t, "Zins GmbH", "DE", "")
	env.openReceivable(t, customer, 1_000_000, "2026-01-17", "2026-01-31", "RE-2026-0006")

	rates, err := svc.SaveBaseRate(ctx, "2026-03-01", 227)
	if err != nil {
		t.Fatalf("Basiszinssatz speichern: %v", err)
	}
	var found bool
	for _, rate := range rates {
		if rate.ValidFrom == "2026-03-01" && rate.BasisPoints == 227 {
			found = true
			if rate.Percent() != "2,27 %" {
				t.Errorf("Anzeige = %q, erwartet \"2,27 %%\"", rate.Percent())
			}
		}
	}
	if !found {
		t.Fatalf("der gespeicherte Satz fehlt in der Tabelle: %+v", rates)
	}

	proposals, err := svc.Proposals(ctx, "2026-04-17")
	if err != nil {
		t.Fatalf("Mahnvorschläge: %v", err)
	}
	// 10.000 × (2,27 % + 9) × 45/365 = 138,9452… → 138,95
	if got := proposals[0].Items[0].Interest; got != 13895 {
		t.Errorf("Zinsen = %s €, erwartet 138,95 mit dem nachgetragenen Satz", got)
	}

	// Ein unplausibler Satz wird abgewiesen: ein Tippfehler im Zins ist eine
	// unberechtigte Forderung.
	if _, err := svc.SaveBaseRate(ctx, "2026-09-01", 15200); err == nil {
		t.Error("ein Satz von 152 Prozentpunkten muss abgewiesen werden")
	}
	if _, err := svc.SaveBaseRate(ctx, "01.09.2026", 100); err == nil {
		t.Error("ein unlesbarer Stichtag muss abgewiesen werden")
	}
}

// Die Pauschale des § 288 Abs. 5 BGB wird auch mit der voreingestellten
// Stufenfolge angesetzt — und zwar einmal.
//
// Der Regelablauf ist der Fall, der zählt: Zahlungserinnerung am siebten Tag,
// erste Mahnung am 21., zweite am 35. Tag nach Fälligkeit. Verzug tritt erst
// dreißig Tage nach Fälligkeit ein (§ 286 Abs. 3 BGB), also während der zweiten
// Mahnung. Hinge die Pauschale an der ersten Mahnstufe, bekäme ein
// Unternehmer-Kunde sie in keinem einzigen Schreiben.
func TestDunningLumpSumAppearsOnceUnderTheDefaultLevels(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.dunning(t, &stubRenderer{})
	customer := env.customer(t, "Regelablauf GmbH", "DE", "")
	env.openReceivable(t, customer, 1_000_000, "2026-01-17", "2026-01-31", "RE-2026-0007")

	// Fälligkeit 31.01.2026: Tag 7 ist der 07.02., Tag 21 der 21.02., Tag 35
	// der 07.03. Verzug ab dem 03.03.
	runs := []struct {
		date      string
		level     int
		lumpSum   domain.Cents
		hasInter  bool
		levelName string
	}{
		{date: "2026-02-07", level: 1, lumpSum: 0, hasInter: false, levelName: "Zahlungserinnerung"},
		{date: "2026-02-21", level: 2, lumpSum: 0, hasInter: false, levelName: "1. Mahnung"},
		{date: "2026-03-07", level: 3, lumpSum: 4000, hasInter: true, levelName: "2. Mahnung"},
	}
	for _, run := range runs {
		proposals, err := svc.Proposals(ctx, run.date)
		if err != nil {
			t.Fatalf("Mahnvorschläge zum %s: %v", run.date, err)
		}
		if len(proposals) != 1 {
			t.Fatalf("zum %s erwartet ein Vorschlag, erhalten %d", run.date, len(proposals))
		}
		p := proposals[0]
		if p.Level != run.level || p.LevelLabel != run.levelName {
			t.Errorf("zum %s Stufe = %d (%q), erwartet %d (%q)",
				run.date, p.Level, p.LevelLabel, run.level, run.levelName)
		}
		if p.LumpSum != run.lumpSum {
			t.Errorf("zum %s Pauschale = %s €, erwartet %s €", run.date, p.LumpSum, run.lumpSum)
		}
		if got := p.Items[0].InterestDays > 0; got != run.hasInter {
			t.Errorf("zum %s Verzugstage = %d, erwartet %v", run.date, p.Items[0].InterestDays, run.hasInter)
		}

		notice, err := svc.Create(ctx, DunningRequest{ContactID: customer.ID, NoticeDate: run.date})
		if err != nil {
			t.Fatalf("Schreiben zum %s: %v", run.date, err)
		}
		if notice.LumpSumAmount != run.lumpSum {
			t.Errorf("zum %s trägt das Schreiben %s € Pauschale, erwartet %s €",
				run.date, notice.LumpSumAmount, run.lumpSum)
		}
		if notice.Items[0].LumpSumAmount != run.lumpSum {
			t.Errorf("zum %s trägt der Posten %s € Pauschale, erwartet %s €",
				run.date, notice.Items[0].LumpSumAmount, run.lumpSum)
		}
	}

	// Nach dem dritten Schreiben ist die Pauschale verbraucht: ein weiterer Lauf
	// derselben Forderung setzt sie nicht erneut an.
	charged, err := repository.NewDunningRepository(env.db).LumpSumChargedByOpenItem(ctx)
	if err != nil {
		t.Fatalf("Pauschalenstand: %v", err)
	}
	if len(charged) != 1 {
		t.Errorf("erwartet genau einen Posten mit angesetzter Pauschale, erhalten %d", len(charged))
	}
}

// Gegenüber einem Verbraucher entsteht die Pauschale auch nach Verzugseintritt
// nicht (§ 288 Abs. 5 Satz 1 BGB).
func TestDunningLumpSumStaysAwayFromConsumersAfterDefault(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Hans Beispiel", "DE", "")
	customer.IsPrivate = true
	if err := env.contacts.SaveContact(ctx, customer); err != nil {
		t.Fatalf("Kunde speichern: %v", err)
	}
	env.openReceivable(t, customer, 1_000_000, "2026-01-17", "2026-01-31", "RE-2026-0008")

	proposals, err := env.dunning(t, nil).Proposals(ctx, "2026-03-07")
	if err != nil {
		t.Fatalf("Mahnvorschläge: %v", err)
	}
	if len(proposals) != 1 {
		t.Fatalf("erwartet einen Vorschlag, erhalten %d", len(proposals))
	}
	if proposals[0].Items[0].InterestDays == 0 {
		t.Fatal("der Verzug muss zum 07.03. eingetreten sein, sonst prüft der Test nichts")
	}
	if proposals[0].LumpSum != 0 {
		t.Errorf("Pauschale = %s €, erwartet 0 gegenüber einem Verbraucher", proposals[0].LumpSum)
	}
}

// Das Mahnschreiben nennt die Bankverbindung, auf die es verweist.
//
// „Ausgleich auf das unten genannte Konto" ohne IBAN ist eine Vorlage, die der
// Empfänger nicht ausführen kann. Fehlt die IBAN in den Einstellungen, muss auch
// der Verweis fehlen — ein Schreiben darf nicht auf etwas zeigen, das nicht
// darin steht.
func TestDunningNoticeCarriesTheBankDetails(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	settings := repository.NewSettingsRepository(env.db)
	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Unternehmensdaten lesen: %v", err)
	}
	cfg.IBAN = "DE02120300000000202051"
	cfg.BIC = "BYLADEM1001"
	cfg.BankName = "Musterbank München"
	if err := settings.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatalf("Bankverbindung speichern: %v", err)
	}

	renderer := &stubRenderer{}
	svc := env.dunning(t, renderer)
	customer := env.customer(t, "Bankverbindung GmbH", "DE", "")
	env.openReceivable(t, customer, 238_000, "2026-01-17", "2026-01-31", "RE-2026-0009")

	if _, err := svc.Create(ctx, DunningRequest{ContactID: customer.ID, NoticeDate: "2026-04-17"}); err != nil {
		t.Fatalf("Mahnschreiben: %v", err)
	}
	for _, want := range []string{
		"DE02120300000000202051", "BYLADEM1001", "Musterbank München",
		"unten genannte Konto", "Verwendungszweck: RE-2026-0009",
	} {
		if !strings.Contains(renderer.lastText, want) {
			t.Errorf("das Schreiben nennt %q nicht:\n%s", want, renderer.lastText)
		}
	}
}

// Ohne hinterlegte IBAN verweist das Schreiben auf kein Konto.
func TestDunningNoticeWithoutIBANDoesNotPromiseAnAccount(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	renderer := &stubRenderer{}
	svc := env.dunning(t, renderer)
	customer := env.customer(t, "Ohne Konto GmbH", "DE", "")
	env.openReceivable(t, customer, 238_000, "2026-01-17", "2026-01-31", "RE-2026-0010")

	if _, err := svc.Create(ctx, DunningRequest{ContactID: customer.ID, NoticeDate: "2026-04-17"}); err != nil {
		t.Fatalf("Mahnschreiben: %v", err)
	}
	if strings.Contains(renderer.lastText, "unten genannte Konto") {
		t.Errorf("ohne IBAN darf das Schreiben nicht auf ein Konto verweisen:\n%s", renderer.lastText)
	}
	if !strings.Contains(renderer.lastText, "Wir bitten um Ausgleich bis zum 01.05.2026.") {
		t.Errorf("die Zahlungsaufforderung fehlt oder nennt die Frist nicht:\n%s", renderer.lastText)
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
// Ohne Satz für den Zeitraum sind die Zinsen null und mit ihnen die Pauschale —
// die Forderung ginge zu niedrig heraus, und niemand sähe, warum. Der Grund
// gehört deshalb an den Posten und in den Vorschlag.
func TestDunningProposalNamesTheMissingBaseRate(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := NewDunningService(
		stubLiveOpenItems{{
			EntryID: 4711, EntryNumber: "2015-0001", ContactID: 1, ContactName: "Altfall GmbH",
			ContactType: domain.ContactTypeCustomer, DocumentNumber: "RE-2015-0001",
			DocumentDate: "2015-01-02", DueDate: "2015-01-31",
			GrossAmount: 1_000_000, OpenAmount: 1_000_000,
		}},
		env.contactRepo,
		repository.NewDunningRepository(env.db),
		repository.NewBaseRateRepository(env.db),
		repository.NewSettingsRepository(env.db),
		repository.NewAuditRepository(env.db),
		env.store, env.fiscalYear,
	)

	proposals, err := svc.Proposals(ctx, "2015-06-01")
	if err != nil {
		t.Fatalf("Mahnvorschläge: %v", err)
	}
	if len(proposals) != 1 {
		t.Fatalf("erwartet einen Vorschlag, erhalten %d", len(proposals))
	}
	p := proposals[0]
	if p.Interest != 0 || p.Items[0].Note == "" {
		t.Fatalf("erwartet null Zinsen mit Begründung, erhalten %s € und %q",
			p.Interest, p.Items[0].Note)
	}
	if !strings.Contains(p.Items[0].Note, "Basiszinssatz") {
		t.Errorf("der Hinweis nennt den fehlenden Basiszinssatz nicht: %q", p.Items[0].Note)
	}
	if !strings.Contains(p.Note, "Basiszinssatz") {
		t.Errorf("der Vorschlag trägt den Hinweis nicht: %q", p.Note)
	}
	// Keine Pauschale — nicht wegen der ausgefallenen Zinsen, sondern weil die
	// Stammdaten dieses Kunden fehlen und der Lauf ihn deshalb vorsichtshalber
	// als Verbraucher rechnet. Dass sie am Verzug hängt und nicht an der
	// Zinsrechnung, prüft der nächste Test.
	if p.LumpSum != 0 {
		t.Errorf("Pauschale = %s €, erwartet 0 gegenüber einem Verbraucher", p.LumpSum)
	}
}

// Die Pauschale hängt am Verzug, nicht am Gelingen der Zinsrechnung.
//
// Fehlt der Basiszinssatz eines Halbjahres, fallen die Zinsen aus — der Verzug
// ist trotzdem eingetreten, und § 288 Abs. 5 BGB knüpft die Pauschale allein
// daran. Sie mit den Zinsen wegfallen zu lassen, hieße: eine Forderung geht um
// 40 € zu niedrig heraus, und der Hinweis spräche nur von Zinsen.
func TestDunningLumpSumSurvivesTheMissingBaseRate(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Altfall GmbH", "DE", "")
	svc := NewDunningService(
		stubLiveOpenItems{{
			EntryID: 4711, EntryNumber: "2015-0001",
			ContactID: customer.ID, ContactName: customer.Name,
			ContactType: domain.ContactTypeCustomer, DocumentNumber: "RE-2015-0001",
			DocumentDate: "2015-01-02", DueDate: "2015-01-31",
			GrossAmount: 1_000_000, OpenAmount: 1_000_000,
		}},
		env.contactRepo,
		repository.NewDunningRepository(env.db),
		repository.NewBaseRateRepository(env.db),
		repository.NewSettingsRepository(env.db),
		repository.NewAuditRepository(env.db),
		env.store, env.fiscalYear,
	)

	proposals, err := svc.Proposals(ctx, "2015-06-01")
	if err != nil {
		t.Fatalf("Mahnvorschläge: %v", err)
	}
	if len(proposals) != 1 {
		t.Fatalf("erwartet einen Vorschlag, erhalten %d", len(proposals))
	}
	p := proposals[0]
	if p.Interest != 0 {
		t.Fatalf("erwartet null Zinsen ohne Basiszinssatz, erhalten %s €", p.Interest)
	}
	if p.LumpSum != 4000 {
		t.Errorf("Pauschale = %s €, erwartet 40,00 trotz fehlender Zinsrechnung", p.LumpSum)
	}
	if p.Total != 1_000_000+4000 {
		t.Errorf("Gesamtbetrag = %s €, erwartet Hauptforderung und Pauschale", p.Total)
	}
	note := p.Items[0].Note
	if !strings.Contains(note, "Basiszinssatz") || !strings.Contains(note, "Pauschale") {
		t.Errorf("der Hinweis nennt nicht beides — fehlende Zinsen und angesetzte Pauschale: %q", note)
	}
}

// Vor dem Verzugsbeginn gibt es keine Pauschale.
//
// Die Zahlungserinnerung geht sieben Tage nach Fälligkeit heraus, der Verzug
// tritt erst nach dreißig Tagen ein (§ 286 Abs. 3 BGB). Zwischen beiden Tagen
// steht ein Vorschlag ohne Zinsen und ohne Pauschale.
func TestDunningLumpSumWaitsForTheDefaultToBegin(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Fruehmahnung GmbH", "DE", "")
	env.openReceivable(t, customer, 1_000_000, "2026-01-17", "2026-01-31", "RE-2026-0007")

	proposals, err := env.dunning(t, nil).Proposals(ctx, "2026-02-10")
	if err != nil {
		t.Fatalf("Mahnvorschläge: %v", err)
	}
	if len(proposals) != 1 {
		t.Fatalf("erwartet einen Vorschlag, erhalten %d", len(proposals))
	}
	p := proposals[0]
	if p.LumpSum != 0 {
		t.Errorf("Pauschale = %s €, erwartet 0 vor dem Verzugsbeginn", p.LumpSum)
	}
	if p.Interest != 0 {
		t.Errorf("Zinsen = %s €, erwartet 0 vor dem Verzugsbeginn", p.Interest)
	}
}

// Ein Kunde, dessen Stammdaten sich nicht lesen lassen, fällt nicht still aus
// dem Mahnlauf.
func TestDunningKeepsCustomerWithUnreadableContact(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := NewDunningService(
		stubLiveOpenItems{{
			EntryID: 815, EntryNumber: "2026-0001", ContactID: 99, ContactName: "Verschollen GmbH",
			ContactType: domain.ContactTypeCustomer, DocumentNumber: "RE-2026-0011",
			DocumentDate: "2026-01-17", DueDate: "2026-01-31",
			GrossAmount: 1_000_000, OpenAmount: 1_000_000,
		}},
		&stubContacts{err: errStubContactMissing},
		repository.NewDunningRepository(env.db),
		repository.NewBaseRateRepository(env.db),
		repository.NewSettingsRepository(env.db),
		repository.NewAuditRepository(env.db),
		env.store, env.fiscalYear,
	)

	proposals, err := svc.Proposals(ctx, "2026-04-17")
	if err != nil {
		t.Fatalf("Mahnvorschläge: %v", err)
	}
	if len(proposals) != 1 {
		t.Fatalf("der Kunde darf nicht stillschweigend entfallen, erhalten %d Vorschläge", len(proposals))
	}
	p := proposals[0]
	if !strings.Contains(p.Note, "Stammdaten") {
		t.Errorf("der Vorschlag nennt die nicht lesbaren Stammdaten nicht: %q", p.Note)
	}
	// Vorsichtshalber der Verbrauchersatz: fünf Prozentpunkte, keine Pauschale.
	if !p.IsConsumer || p.LumpSum != 0 {
		t.Errorf("ohne Stammdaten ist mit dem niedrigeren Satz zu rechnen: %+v", p)
	}
	if p.Items[0].Interest != 7730 {
		t.Errorf("Zinsen = %s €, erwartet 77,30 (fünf Prozentpunkte)", p.Items[0].Interest)
	}
}

// failingDunningRepo lässt jedes Anlegen scheitern und liest sonst wie das
// Original.
type failingDunningRepo struct{ domain.DunningRepository }

func (r failingDunningRepo) Create(context.Context, *domain.DunningNotice) error {
	return errors.New("die Datenbank ist nicht erreichbar")
}

// Scheitert der Datensatz, bleibt keine Datei im Belegspeicher zurück.
//
// Das PDF wird vor dem Datensatz abgelegt, weil Pfad und Prüfsumme in ihn
// gehören. Bliebe es nach einem Fehler liegen, stünde unter dokumente/mahnungen/
// ein Schreiben ohne Empfänger, ohne Betrag und ohne Protokolleintrag — ein
// Fundstück, das im Prüfermodus niemand mehr erklären kann.
func TestDunningRemovesTheDocumentWhenTheRecordFails(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Fehlschlag GmbH", "DE", "")
	env.openReceivable(t, customer, 238_000, "2026-01-17", "2026-01-31", "RE-2026-0009")

	svc := NewDunningService(
		env.payments(t), env.contactRepo,
		failingDunningRepo{repository.NewDunningRepository(env.db)},
		repository.NewBaseRateRepository(env.db),
		repository.NewSettingsRepository(env.db),
		repository.NewAuditRepository(env.db),
		env.store, env.fiscalYear,
	)
	svc.SetRenderer(&stubRenderer{})

	if _, err := svc.Create(ctx, DunningRequest{
		ContactID: customer.ID, NoticeDate: "2026-04-17",
	}); err == nil {
		t.Fatal("ein Schreiben ohne Datensatz darf nicht als erzeugt gelten")
	}

	dir := filepath.Join(env.dataDir, "dokumente", DunningCategory)
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("Ablageordner lesen: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("im Ordner %s liegen %d Dateien ohne Datensatz: %v",
			dir, len(entries), entries)
	}
}
