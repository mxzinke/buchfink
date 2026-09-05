package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func (e *testEnv) vatReturns(t *testing.T) *VatReturnService {
	t.Helper()
	return NewVatReturnService(
		e.journalRepo,
		e.receiptRepo,
		e.contactRepo,
		repository.NewSettingsRepository(e.db),
		repository.NewFestschreibungRepository(e.db),
		repository.NewVatReturnRepository(e.db),
		repository.NewAuditRepository(e.db),
		e.fiscalYear,
	)
}

// setSettings ändert die Unternehmensdaten gezielt und lässt den Rest stehen.
func (e *testEnv) setSettings(t *testing.T, change func(*domain.CompanySettings)) {
	t.Helper()
	repo := repository.NewSettingsRepository(e.db)
	cfg, err := repo.GetCompanySettings(context.Background())
	if err != nil {
		t.Fatalf("Unternehmensdaten lesen: %v", err)
	}
	change(cfg)
	if err := repo.UpdateCompanySettings(context.Background(), cfg); err != nil {
		t.Fatalf("Unternehmensdaten schreiben: %v", err)
	}
}

// commitUntil schreibt den Zeitraum bis zum Stichtag fest.
func (e *testEnv) commitUntil(t *testing.T, cutoff string) {
	t.Helper()
	repo := repository.NewFestschreibungRepository(e.db)
	err := repo.Create(context.Background(), &domain.Festschreibung{
		FiscalYear: e.fiscalYear, PeriodType: "quarter", PeriodLabel: "Test",
		CutoffDate: cutoff, ChainHead: domain.GenesisHash, CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Festschreibung bis %s: %v", cutoff, err)
	}
}

// domesticInvoice stellt eine Inlandsrechnung mit 19 % aus.
func (e *testEnv) domesticInvoice(t *testing.T, customerID uint, date string, net domain.Cents) *domain.Invoice {
	t.Helper()
	inv := &domain.Invoice{
		ContactID: customerID, Date: date,
		ServiceDateFrom: date, ServiceDateTo: date,
		TaxTreatment: domain.TaxTreatmentDomestic,
		Items: []domain.InvoiceItem{{
			Description: "Leistung", QuantityMilli: 1000, UnitPrice: net, TaxRate: domain.TaxRateStandard,
		}},
	}
	if err := e.invoices(t).Issue(context.Background(), inv); err != nil {
		t.Fatalf("Rechnung vom %s: %v", date, err)
	}
	return inv
}

// Das Kennziffernblatt entsteht aus dem Journal und ordnet den Umsatz dem
// Zeitraum der Leistung zu.
func TestVatReturnDraftFromJournal(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-02-10", 200000)

	ret, err := env.vatReturns(t).Draft(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Entwurf: %v", err)
	}
	if ret.Base("81") != 200000 || ret.Tax("81") != 38000 {
		t.Errorf("Kz 81 = %s / %s, erwartet 2.000,00 / 380,00", ret.Base("81"), ret.Tax("81"))
	}
	if ret.Payable != 38000 {
		t.Errorf("Zahllast = %s, erwartet 380,00", ret.Payable)
	}
	if len(ret.EntryIDs("81")) != 1 {
		t.Errorf("Drill-down zu Kz 81 hat %d Buchungen, erwartet 1", len(ret.EntryIDs("81")))
	}
	if ret.ProgramVersion == "" {
		t.Error("die Programmversion gehört an die Anmeldung, sonst ist sie später nicht nachvollziehbar")
	}
	// Ohne Dauerfristverlängerung ist die Anmeldung für Q1 am 10. April fällig.
	if ret.DueDate != "2026-04-10" {
		t.Errorf("Fälligkeit = %s, erwartet 2026-04-10", ret.DueDate)
	}
}

// Bestätigt wird nur, was festgeschrieben ist: eine übermittelte Anmeldung muss
// sich auf einen unveränderlichen Stand stützen.
func TestVatReturnConfirmationRequiresCommittedPeriod(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-02-10", 200000)

	svc := env.vatReturns(t)
	saved, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("speichern: %v", err)
	}

	_, err = svc.ConfirmSubmitted(ctx, saved.ID, "2026-04-09", "1234567890", "")
	if err == nil {
		t.Fatal("ohne Festschreibung darf keine Übermittlung bestätigt werden")
	}
	if !strings.Contains(err.Error(), "festgeschrieben") {
		t.Errorf("die Meldung sollte die fehlende Festschreibung benennen: %v", err)
	}

	env.commitUntil(t, "2026-03-31")
	if _, err := svc.ConfirmSubmitted(ctx, saved.ID, "2026-04-09", "1234567890", "Test"); err != nil {
		t.Fatalf("nach der Festschreibung muss die Bestätigung möglich sein: %v", err)
	}
}

// Ohne Transferticket gibt es keine Bestätigung: es ist der einzige Nachweis,
// dass die Anmeldung angekommen ist.
func TestVatReturnConfirmationRequiresTransferTicket(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-02-10", 200000)
	env.commitUntil(t, "2026-03-31")

	svc := env.vatReturns(t)
	saved, _ := svc.Save(ctx, "2026-Q1")

	if _, err := svc.ConfirmSubmitted(ctx, saved.ID, "2026-04-09", "   ", ""); err == nil {
		t.Fatal("ohne Transferticket darf nicht bestätigt werden")
	}
	if _, err := svc.ConfirmSubmitted(ctx, saved.ID, "", "1234567890", ""); err == nil {
		t.Fatal("ohne Datum darf nicht bestätigt werden")
	}
}

// Nach der Bestätigung ist die Anmeldung unveränderlich — sie ist das
// Übermittlungsprotokoll.
func TestSubmittedVatReturnIsImmutable(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-02-10", 200000)
	env.commitUntil(t, "2026-03-31")

	svc := env.vatReturns(t)
	saved, _ := svc.Save(ctx, "2026-Q1")
	if _, err := svc.ConfirmSubmitted(ctx, saved.ID, "2026-04-09", "TT-1", ""); err != nil {
		t.Fatalf("bestätigen: %v", err)
	}

	if _, err := svc.ConfirmSubmitted(ctx, saved.ID, "2026-04-20", "TT-2", ""); err == nil {
		t.Fatal("eine bestätigte Anmeldung darf nicht ein zweites Mal bestätigt werden")
	}

	// Ein erneutes Speichern legt *keinen* zweiten Entwurf an: er wäre eine
	// zweite Erstanmeldung desselben Zeitraums, ohne Kennziffer 10 und ohne
	// Bezug auf die erste. Geändert wird über die Berichtigung.
	if _, err := svc.Save(ctx, "2026-Q1"); err == nil {
		t.Fatal("nach der Bestätigung darf kein zweiter Entwurf desselben Zeitraums entstehen")
	} else if !strings.Contains(err.Error(), "berichtigte") {
		t.Errorf("die Meldung sollte auf die Berichtigung verweisen: %v", err)
	}

	stored, err := repository.NewVatReturnRepository(env.db).FindByID(ctx, saved.ID)
	if err != nil {
		t.Fatalf("laden: %v", err)
	}
	if stored.Status != domain.VatReturnSubmitted || stored.TransferTicket != "TT-1" {
		t.Errorf("die bestätigte Anmeldung steht jetzt auf %q mit Ticket %q", stored.Status, stored.TransferTicket)
	}
}

// Die Bestätigung der Übermittlung ist ein Statuswechsel und keine Ausgabe.
//
// Sie steht deshalb unter UPDATE, die Datei-Ausgabe unter EXPORT. Stünden beide
// unter derselben Aktion, wären sie im Protokoll nur am Text zu unterscheiden —
// und wer den Weg einer Anmeldung sucht, filtert nach der Aktion.
func TestConfirmationIsLoggedAsUpdateNotExport(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-02-10", 200000)
	env.commitUntil(t, "2026-03-31")

	svc := env.vatReturns(t)
	saved, _ := svc.Save(ctx, "2026-Q1")
	if _, err := svc.ConfirmSubmitted(ctx, saved.ID, "2026-04-09", "TT-1", ""); err != nil {
		t.Fatalf("bestätigen: %v", err)
	}
	if _, err := svc.ExportCSV(ctx, saved.ID); err != nil {
		t.Fatalf("Datei-Ausgabe: %v", err)
	}

	entries, err := repository.NewAuditRepository(env.db).FindAll(ctx, 100)
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	var confirmAction, exportAction domain.AuditAction
	for _, e := range entries {
		if e.EntityType != "VAT_RETURN" {
			continue
		}
		switch {
		case strings.Contains(e.Details, "übermittelt"):
			confirmAction = e.Action
		case strings.Contains(e.Details, "als Datei ausgegeben"):
			exportAction = e.Action
		}
	}
	if confirmAction != domain.AuditActionUpdate {
		t.Errorf("die Bestätigung steht unter %q, erwartet %q", confirmAction, domain.AuditActionUpdate)
	}
	if exportAction != domain.AuditActionExport {
		t.Errorf("die Datei-Ausgabe steht unter %q, erwartet %q", exportAction, domain.AuditActionExport)
	}
}

// Die Dauerfristverlängerung verschiebt jede Fälligkeit um einen Monat, und die
// Sondervorauszahlung wird beim Monatszahler nur in der letzten Voranmeldung des
// Jahres angerechnet (§ 48 Abs. 4 UStDV).
func TestPermanentExtensionShiftsDueDateAndCreditsPrepayment(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.setSettings(t, func(c *domain.CompanySettings) {
		c.VatPeriod = "month"
		c.PermanentExtension = true
		c.SpecialPrepayment = 100000 // 1.000,00 €
	})
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-02-10", 200000)
	env.domesticInvoice(t, customer.ID, "2026-12-10", 500000)

	svc := env.vatReturns(t)

	first, err := svc.Draft(ctx, "2026-02")
	if err != nil {
		t.Fatalf("Februar: %v", err)
	}
	if first.DueDate != "2026-04-10" {
		t.Errorf("Februar mit Dauerfrist fällig am %s, erwartet 2026-04-10", first.DueDate)
	}
	if first.Tax("39") != 0 {
		t.Errorf("Kz 39 im Februar = %s, erwartet null — angerechnet wird erst im letzten Zeitraum", first.Tax("39"))
	}

	last, err := svc.Draft(ctx, "2026-12")
	if err != nil {
		t.Fatalf("Dezember: %v", err)
	}
	if last.Tax("39") != 100000 {
		t.Errorf("Kz 39 im Dezember = %s, erwartet 1.000,00", last.Tax("39"))
	}
	// 5.000,00 netto × 19 % = 950,00 Steuer, abzüglich 1.000,00 Sondervorauszahlung.
	if last.Payable != -5000 {
		t.Errorf("Zahllast Dezember = %s, erwartet −50,00 (950,00 − 1.000,00)", last.Payable)
	}
}

// Der Vierteljahreszahler bekommt die Dauerfristverlängerung ohne
// Sondervorauszahlung (§ 46 UStDV): die Fälligkeit verschiebt sich, Kennziffer 39
// bleibt leer.
//
// § 47 Abs. 1 UStDV knüpft die Sondervorauszahlung an die monatliche Abgabe.
// Rechnete Buchfink sie auch hier an, minderte das Blatt die Zahllast um einen
// Betrag, den niemand vorausgezahlt hat — und das Finanzamt forderte die
// Differenz nach.
func TestQuarterlyFilerGetsExtensionWithoutPrepaymentCredit(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.setSettings(t, func(c *domain.CompanySettings) {
		c.VatPeriod = "quarter"
		c.PermanentExtension = true
		c.SpecialPrepayment = 100000 // erfasst, aber ohne Wirkung
	})
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-11-10", 500000)

	last, err := env.vatReturns(t).Draft(ctx, "2026-Q4")
	if err != nil {
		t.Fatalf("Q4: %v", err)
	}
	if last.DueDate != "2027-02-10" {
		t.Errorf("Q4 mit Dauerfrist fällig am %s, erwartet 2027-02-10", last.DueDate)
	}
	if last.Tax("39") != 0 {
		t.Errorf("Kz 39 = %s, erwartet null — der Vierteljahreszahler leistet keine Sondervorauszahlung", last.Tax("39"))
	}
	// Volle 950,00 Steuer, ohne Anrechnung.
	if last.Payable != 95000 {
		t.Errorf("Zahllast Q4 = %s, erwartet 950,00 ohne Anrechnung", last.Payable)
	}
}

// Die Sondervorauszahlung wird aus den übermittelten Voranmeldungen des
// Vorjahres vorgeschlagen: ein Elftel der Summe der Vorauszahlungen
// (§ 47 Abs. 1 UStDV). Die im Vorjahr angerechnete Sondervorauszahlung
// (Kennziffer 39) gehört wieder hinzu — sonst schrumpfte der Vorschlag Jahr für
// Jahr, ohne dass die Umsätze es täten.
func TestSpecialPrepaymentSuggestionFromPriorYearReturns(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.setSettings(t, func(c *domain.CompanySettings) {
		c.VatPeriod = "month"
		c.PermanentExtension = true
		c.SpecialPrepayment = 100000 // 1.000,00 € im Vorjahr angerechnet
	})
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-02-10", 1000000) // 10.000,00 netto → 1.900,00 Steuer
	env.domesticInvoice(t, customer.ID, "2026-12-10", 1000000)

	svc := env.vatReturns(t)
	env.commitUntil(t, "2026-12-31")
	for _, c := range []struct{ key, date, ticket string }{
		{"2026-02", "2026-04-10", "TT-FEB"},
		{"2026-12", "2027-02-10", "TT-DEZ"},
	} {
		saved, err := svc.Save(ctx, c.key)
		if err != nil {
			t.Fatalf("%s speichern: %v", c.key, err)
		}
		if _, err := svc.ConfirmSubmitted(ctx, saved.ID, c.date, c.ticket, ""); err != nil {
			t.Fatalf("%s bestätigen: %v", c.key, err)
		}
	}

	// Der Dezember weist eine Zahllast von 900,00 aus (1.900,00 − 1.000,00
	// angerechnet); als Vorauszahlung zählen die vollen 1.900,00.
	suggestion, err := svc.SuggestedSpecialPrepayment(ctx, 2027)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	if suggestion.PrepaymentSum != 380000 {
		t.Errorf("Summe der Vorauszahlungen = %s €, erwartet 3.800,00 (1.900,00 + 1.900,00)", suggestion.PrepaymentSum)
	}
	// 3.800,00 / 11 = 345,4545… → auf volle Euro abgerundet 345,00.
	if suggestion.Amount != 34500 {
		t.Errorf("Vorschlag = %s €, erwartet 345,00", suggestion.Amount)
	}
	if len(suggestion.Periods) != 2 || suggestion.Periods[0].PeriodKey != "2026-02" {
		t.Errorf("der Vorschlag muss die berücksichtigten Anmeldungen nennen: %+v", suggestion.Periods)
	}
	if suggestion.Periods[1].Prepayment != 190000 {
		t.Errorf("Vorauszahlung Dezember = %s €, erwartet 1.900,00 vor Anrechnung", suggestion.Periods[1].Prepayment)
	}
	if !suggestion.Applicable {
		t.Error("bei monatlicher Voranmeldung gibt es eine Sondervorauszahlung")
	}
	if suggestion.Complete {
		t.Error("für zehn Monate liegt keine Anmeldung vor — der Vorschlag ist unvollständig")
	}
	if !strings.Contains(suggestion.Note, "47 Abs. 2") {
		t.Errorf("der Hinweis sollte auf die Hochrechnung nach § 47 Abs. 2 UStDV zeigen: %q", suggestion.Note)
	}
	if suggestion.Account != domain.AccountSondervorauszahlung {
		t.Errorf("Konto = %q, erwartet %q", suggestion.Account, domain.AccountSondervorauszahlung)
	}
}

// Ohne übermittelte Anmeldungen des Vorjahres gibt es keinen Vorschlag, sondern
// den Hinweis auf die Schätzung nach § 47 Abs. 3 UStDV.
func TestSpecialPrepaymentSuggestionWithoutPriorYearReturns(t *testing.T) {
	env := newTestEnv(t)
	env.setSettings(t, func(c *domain.CompanySettings) { c.VatPeriod = "month" })
	suggestion, err := env.vatReturns(t).SuggestedSpecialPrepayment(context.Background(), 2026)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	if suggestion.Amount != 0 || suggestion.Complete {
		t.Errorf("ohne Anmeldungen des Vorjahres gibt es keinen Vorschlag: %+v", suggestion)
	}
	if !strings.Contains(suggestion.Note, "47 Abs. 3") {
		t.Errorf("der Hinweis sollte die Schätzung nach § 47 Abs. 3 UStDV nennen: %q", suggestion.Note)
	}
}

// Dem Vierteljahreszahler wird nichts vorgeschlagen: er schuldet keine
// Sondervorauszahlung (§ 47 Abs. 1 UStDV), und ein Vorschlag wäre die
// Aufforderung, etwas anzumelden und zu zahlen, das es für ihn nicht gibt. Der
// Hinweis sagt, warum — ein Vorschlag von null Euro allein sähe aus wie ein
// Rechenfehler.
func TestNoSpecialPrepaymentSuggestionForQuarterlyFiler(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.setSettings(t, func(c *domain.CompanySettings) {
		c.VatPeriod = "quarter"
		c.PermanentExtension = true
	})
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-02-10", 1000000)

	svc := env.vatReturns(t)
	env.commitUntil(t, "2026-12-31")
	saved, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Q1 speichern: %v", err)
	}
	if _, err := svc.ConfirmSubmitted(ctx, saved.ID, "2026-05-11", "TT-Q1", ""); err != nil {
		t.Fatalf("Q1 bestätigen: %v", err)
	}

	suggestion, err := svc.SuggestedSpecialPrepayment(ctx, 2027)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	if suggestion.Applicable {
		t.Error("der Vierteljahreszahler schuldet keine Sondervorauszahlung")
	}
	if suggestion.Amount != 0 || len(suggestion.Periods) != 0 {
		t.Errorf("kein Vorschlag trotz übermittelter Anmeldung erwartet: %+v", suggestion)
	}
	if !strings.Contains(suggestion.Note, "46 UStDV") {
		t.Errorf("der Hinweis sollte die Dauerfristverlängerung ohne Sondervorauszahlung nennen: %q", suggestion.Note)
	}
}

// Eine Buchung, deren Zeitraum bereits übermittelt ist, erscheint als Nachtrag;
// die Berichtigung meldet den Zeitraum vollständig neu.
func TestLateEntryLeadsToCorrection(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.setSettings(t, func(c *domain.CompanySettings) { c.VatPeriod = "month" })
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-01-20", 100000)

	svc := env.vatReturns(t)
	env.commitUntil(t, "2026-01-31")
	january, err := svc.Save(ctx, "2026-01")
	if err != nil {
		t.Fatalf("Januar speichern: %v", err)
	}
	if _, err := svc.ConfirmSubmitted(ctx, january.ID, "2026-02-10", "TT-JAN", ""); err != nil {
		t.Fatalf("Januar bestätigen: %v", err)
	}

	// Eine im Februar erfasste Rechnung mit Januar-Leistung: sie gehört in den
	// übermittelten Januar und ist deshalb ein Nachtrag.
	late := &domain.Invoice{
		ContactID: customer.ID, Date: "2026-02-05",
		ServiceDateFrom: "2026-01-28", ServiceDateTo: "2026-01-28",
		TaxTreatment: domain.TaxTreatmentDomestic,
		Items: []domain.InvoiceItem{{
			Description: "Nachgereichte Leistung", QuantityMilli: 1000, UnitPrice: 50000, TaxRate: domain.TaxRateStandard,
		}},
	}
	if err := env.invoices(t).Issue(ctx, late); err != nil {
		t.Fatalf("Nachzügler: %v", err)
	}

	february, err := svc.Draft(ctx, "2026-02")
	if err != nil {
		t.Fatalf("Februar: %v", err)
	}
	if february.Base("81") != 0 {
		t.Errorf("Kz 81 im Februar = %s, erwartet null — der Umsatz gehört in den Januar", february.Base("81"))
	}
	if len(february.LateEntries) != 1 || february.LateEntries[0].PeriodKey != "2026-01" {
		t.Fatalf("erwartet einen Nachtrag für 2026-01, erhalten %+v", february.LateEntries)
	}

	correction, err := svc.CreateCorrection(ctx, "2026-01")
	if err != nil {
		t.Fatalf("Berichtigung: %v", err)
	}
	if !correction.IsCorrection || correction.CorrectsID == nil || *correction.CorrectsID != january.ID {
		t.Errorf("die Berichtigung muss auf die ursprüngliche Anmeldung %d verweisen: %+v", january.ID, correction)
	}
	if correction.Base("81") != 150000 {
		t.Errorf("Kz 81 der Berichtigung = %s, erwartet 1.500,00 (1.000,00 + 500,00)", correction.Base("81"))
	}
	if len(correction.LateEntries) != 0 {
		t.Errorf("die Berichtigung führt keine Nachträge mehr, erhalten %d", len(correction.LateEntries))
	}
}

// Nach dem Anlegen der Berichtigung trägt auch das neu gerechnete Blatt die
// Kennziffer 10.
//
// Die Ansicht zeigt immer den Entwurf aus Draft. Übernähme er die Berichtigung
// nicht, wäre sie auf dem Blatt von einer Erstanmeldung nicht zu unterscheiden —
// Kennziffer 10 stünde allein in der Ausgabedatei, und das Merkmal, an dem das
// Finanzamt die ersetzende Anmeldung erkennt, fehlte genau dort, wo der Anwender
// es abliest.
func TestDraftCarriesCorrectionMark(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.setSettings(t, func(c *domain.CompanySettings) { c.VatPeriod = "month" })
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-01-20", 100000)

	svc := env.vatReturns(t)
	env.commitUntil(t, "2026-01-31")
	january, err := svc.Save(ctx, "2026-01")
	if err != nil {
		t.Fatalf("Januar speichern: %v", err)
	}
	if _, err := svc.ConfirmSubmitted(ctx, january.ID, "2026-02-10", "TT-JAN", ""); err != nil {
		t.Fatalf("Januar bestätigen: %v", err)
	}

	// Vor der Berichtigung ist das Blatt eine Erstanmeldung.
	before, err := svc.Draft(ctx, "2026-01")
	if err != nil {
		t.Fatalf("Januar-Entwurf: %v", err)
	}
	if before.IsCorrection {
		t.Error("ohne angelegte Berichtigung trägt das Blatt keine Kennziffer 10")
	}

	correction, err := svc.CreateCorrection(ctx, "2026-01")
	if err != nil {
		t.Fatalf("Berichtigung: %v", err)
	}

	after, err := svc.Draft(ctx, "2026-01")
	if err != nil {
		t.Fatalf("Januar-Entwurf nach der Berichtigung: %v", err)
	}
	if !after.IsCorrection {
		t.Error("nach der Berichtigung trägt das Blatt die Kennziffer 10")
	}
	if after.CorrectsID == nil || *after.CorrectsID != january.ID {
		t.Errorf("das Blatt muss auf die berichtigte Anmeldung %d verweisen, erhalten %+v", january.ID, after.CorrectsID)
	}
	if correction.CorrectsID == nil || *after.CorrectsID != *correction.CorrectsID {
		t.Errorf("Entwurf und gespeicherte Berichtigung müssen auf dieselbe Anmeldung verweisen")
	}
}

// Der Voranmeldungszeitraum ist ein Kalenderzeitraum, das Geschäftsjahr einer
// Buchung folgt dem Buchungsdatum. Beide fallen auseinander — die im Januar
// erfasste Dezemberleistung ebenso wie jeder Kalendermonat, der bei abweichendem
// Wirtschaftsjahr im anderen Geschäftsjahr liegt. Gelesen wird deshalb über die
// angrenzenden Geschäftsjahre, ausgewählt wird über das Datum.
func TestVatReturnReadsAcrossFiscalYears(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.setSettings(t, func(c *domain.CompanySettings) { c.VatPeriod = "month" })

	// Im Januar 2027 erfasst (Geschäftsjahr 2027), Leistung im Dezember 2026.
	late := &domain.JournalEntry{
		BookingDate: "2027-01-05", DocumentDate: "2027-01-05",
		ServiceDateFrom: "2026-12-20", ServiceDateTo: "2026-12-20",
		Description: "Dezemberleistung, im Januar erfasst",
		Source:      domain.EntrySourceManual, TaxTreatment: domain.TaxTreatmentDomestic,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountKasse, Amount: 119000},
			{Side: domain.SideCredit, Account: "4400", Amount: 100000},
			{Side: domain.SideCredit, Account: domain.AccountUmsatzsteuer19, Amount: 19000, TaxKey: "UST19", TaxBase: 100000},
		},
	}
	if _, err := env.journal.Post(ctx, late); err != nil {
		t.Fatalf("Buchung im Folgejahr: %v", err)
	}

	// Im Dezember 2025 erfasst (Geschäftsjahr 2025), Leistung im Januar 2026.
	early := &domain.JournalEntry{
		BookingDate: "2025-12-28", DocumentDate: "2025-12-28",
		ServiceDateFrom: "2026-01-15", ServiceDateTo: "2026-01-15",
		Description: "Januarleistung, im Dezember erfasst",
		Source:      domain.EntrySourceManual, TaxTreatment: domain.TaxTreatmentDomestic,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountKasse, Amount: 59500},
			{Side: domain.SideCredit, Account: "4400", Amount: 50000},
			{Side: domain.SideCredit, Account: domain.AccountUmsatzsteuer19, Amount: 9500, TaxKey: "UST19", TaxBase: 50000},
		},
	}
	if _, err := env.journal.Post(ctx, early); err != nil {
		t.Fatalf("Buchung im Vorjahr: %v", err)
	}

	svc := env.vatReturns(t)
	december, err := svc.Draft(ctx, "2026-12")
	if err != nil {
		t.Fatalf("Dezember: %v", err)
	}
	if december.Base("81") != 100000 {
		t.Errorf("Kz 81 im Dezember 2026 = %s, erwartet 1.000,00 — die Leistung wurde im Dezember ausgeführt", december.Base("81"))
	}

	january, err := svc.Draft(ctx, "2026-01")
	if err != nil {
		t.Fatalf("Januar: %v", err)
	}
	if january.Base("81") != 50000 {
		t.Errorf("Kz 81 im Januar 2026 = %s, erwartet 500,00 — die Leistung wurde im Januar ausgeführt", january.Base("81"))
	}
}

// Ohne übermittelte Anmeldung gibt es nichts zu berichtigen.
func TestCorrectionNeedsASubmittedReturn(t *testing.T) {
	env := newTestEnv(t)
	if _, err := env.vatReturns(t).CreateCorrection(context.Background(), "2026-Q1"); err == nil {
		t.Fatal("eine Berichtigung ohne ursprüngliche Anmeldung muss abgewiesen werden")
	}
}

// Die Datei trägt Kennziffer und Wert — nur die Zeilen mit Inhalt, weil eine
// getippte Null in ELSTER eine Angabe ist und keine Auslassung.
func TestVatReturnCSVCarriesCodeAndValue(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-02-10", 200000)

	svc := env.vatReturns(t)
	saved, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("speichern: %v", err)
	}
	csv, err := svc.ExportCSV(ctx, saved.ID)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if !strings.HasPrefix(csv, "Kennziffer;Wert\n") {
		t.Errorf("die Datei braucht eine Kopfzeile, beginnt aber mit %q", strings.SplitN(csv, "\n", 2)[0])
	}
	for _, want := range []string{"81;2000\n", "81;380.00\n", "83;380.00\n"} {
		if !strings.Contains(csv, want) {
			t.Errorf("die Datei sollte %q enthalten:\n%s", want, csv)
		}
	}
	if strings.Contains(csv, "44;") {
		t.Errorf("leere Kennziffern gehören nicht in die Datei:\n%s", csv)
	}
}

// Die Zeitraumliste sagt, was fällig, festgeschrieben und übermittelt ist. Ohne
// sie müsste die Oberfläche selbst rechnen.
func TestVatPeriodsCarryDueDateAndStatus(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-02-10", 200000)
	env.commitUntil(t, "2026-03-31")

	svc := env.vatReturns(t)
	saved, _ := svc.Save(ctx, "2026-Q1")
	if _, err := svc.ConfirmSubmitted(ctx, saved.ID, "2026-04-09", "TT-1", ""); err != nil {
		t.Fatalf("bestätigen: %v", err)
	}

	periods, err := svc.Periods(ctx, 2026)
	if err != nil {
		t.Fatalf("Zeiträume: %v", err)
	}
	if len(periods) != 4 {
		t.Fatalf("erwartet vier Quartale, erhalten %d", len(periods))
	}
	if periods[0].Status != domain.VatReturnSubmitted || !periods[0].Committed {
		t.Errorf("Q1 = Status %q, festgeschrieben %v; erwartet übermittelt und festgeschrieben",
			periods[0].Status, periods[0].Committed)
	}
	if periods[1].Committed {
		t.Error("Q2 ist nicht festgeschrieben")
	}
	if periods[1].DueDate != accounting.VatDueDate(periods[1].VatPeriod, false) {
		t.Errorf("Fälligkeit Q2 = %s", periods[1].DueDate)
	}
}

// § 14c UStG: eine Rechnung mit Steuerausweis zu einem Steuerfall ohne
// Steuerpflicht wird nicht ausgestellt.
func TestInvoiceWithUnlawfulTaxIsRefused(t *testing.T) {
	inv := &domain.Invoice{
		ContactID: 1, Date: "2026-03-01",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-01",
		TaxTreatment: domain.TaxTreatmentIntraCommunitySupply,
		NetAmount:    100000, TaxAmount: 19000, GrossAmount: 119000,
		Items: []domain.InvoiceItem{{
			Description: "Ware", QuantityMilli: 1000, UnitPrice: 100000, TaxRate: domain.TaxRateStandard,
		}},
	}
	err := inv.Validate()
	if err == nil {
		t.Fatal("ein Steuerausweis bei steuerfreiem Umsatz muss abgewiesen werden (§ 14c UStG)")
	}
	if !strings.Contains(err.Error(), "14c") {
		t.Errorf("die Meldung sollte § 14c UStG benennen: %v", err)
	}

	// Derselbe Betrag bei einem steuerpflichtigen Inlandsumsatz ist in Ordnung.
	inv.TaxTreatment = domain.TaxTreatmentDomestic
	if err := inv.Validate(); err != nil {
		t.Errorf("ein Inlandsumsatz darf Steuer ausweisen: %v", err)
	}
}

// Dieselbe Regel greift beim Ausstellen, und zwar als Abweisung: Buchfink nahm
// den Steuersatz früher stillschweigend aus der Position: heraus kam eine
// Rechnung ohne Steuer, obwohl der Anwender 19 % erfasst hatte. Welche der
// beiden Angaben stimmt — Steuerfall oder Satz —, weiß nur er.
func TestIssueRefusesUnlawfulTax(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")

	inv := &domain.Invoice{
		ContactID: customer.ID, Date: "2026-03-15",
		ServiceDateFrom: "2026-03-15", ServiceDateTo: "2026-03-15",
		TaxTreatment: domain.TaxTreatmentIntraCommunitySupply,
		Items: []domain.InvoiceItem{{
			Description: "Warenlieferung", QuantityMilli: 1000, UnitPrice: 500000, TaxRate: domain.TaxRateStandard,
		}},
	}
	err := env.invoices(t).Issue(ctx, inv)
	if err == nil {
		t.Fatal("eine Rechnung mit Steuersatz zu einem steuerfreien Steuerfall darf nicht ausgestellt werden")
	}
	if !strings.Contains(err.Error(), "14c") {
		t.Errorf("die Meldung sollte § 14c UStG benennen: %v", err)
	}
	if inv.Status == domain.InvoiceStatusIssued || inv.JournalEntryID != nil {
		t.Error("die abgewiesene Rechnung wurde trotzdem ausgestellt oder gebucht")
	}

	// Derselbe Fall ohne Steuersatz an der Position ist in Ordnung.
	inv.Items[0].TaxRate = domain.TaxRateNone
	if err := env.invoices(t).Issue(ctx, inv); err != nil {
		t.Fatalf("ohne Steuersatz muss die Lieferung ausstellbar sein: %v", err)
	}
	if inv.TaxAmount != 0 {
		t.Errorf("ausgewiesene Steuer = %s, erwartet null", inv.TaxAmount)
	}

	// Und die Vorschau weist denselben Fehler aus, damit er in der Maske
	// auffällt und nicht erst am Knopf „Ausstellen".
	draft := &domain.Invoice{
		ContactID: customer.ID, Date: "2026-03-16",
		ServiceDateFrom: "2026-03-16", ServiceDateTo: "2026-03-16",
		TaxTreatment: domain.TaxTreatmentIntraCommunitySupply,
		Items: []domain.InvoiceItem{{
			Description: "Warenlieferung", QuantityMilli: 1000, UnitPrice: 100000, TaxRate: domain.TaxRateStandard,
		}},
	}
	if _, err := env.invoices(t).Preview(ctx, draft); err == nil {
		t.Error("die Vorschau muss denselben Steuerausweis abweisen wie das Ausstellen")
	}
}

// Für Beträge, die trotzdem geschuldet werden — eine Rechnung außerhalb von
// Buchfink —, gibt es den Handbuchungsweg auf das Konto 3851. Er landet in
// Kennziffer 69.
func TestManualUnlawfulTaxLandsInCode69(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	entry := &domain.JournalEntry{
		BookingDate: "2026-02-20", DocumentDate: "2026-02-20",
		ServiceDateFrom: "2026-02-20", ServiceDateTo: "2026-02-20",
		Description: "Unrichtig ausgewiesene Steuer nach § 14c UStG",
		Source:      domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountKasse, Amount: 19000},
			{
				Side: domain.SideCredit, Account: domain.AccountUmsatzsteuer14c, Amount: 19000,
				TaxKey: accounting.TaxKeyUnlawful,
			},
		},
	}
	if _, err := env.journal.Post(ctx, entry); err != nil {
		t.Fatalf("§ 14c-Buchung: %v", err)
	}

	ret, err := env.vatReturns(t).Draft(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Entwurf: %v", err)
	}
	if ret.Tax("69") != 19000 {
		t.Errorf("Kz 69 = %s, erwartet 190,00", ret.Tax("69"))
	}
	if ret.Payable != 19000 {
		t.Errorf("Zahllast = %s, erwartet 190,00 — der Betrag wird geschuldet", ret.Payable)
	}
}

// Ohne Steuerschlüssel bleibt das § 14c-Konto gesperrt: es ist ein Steuerkonto
// und gehört der Steuerautomatik.
func TestUnlawfulTaxAccountNeedsItsTaxKey(t *testing.T) {
	env := newTestEnv(t)
	entry := &domain.JournalEntry{
		BookingDate: "2026-02-20", DocumentDate: "2026-02-20",
		ServiceDateFrom: "2026-02-20", ServiceDateTo: "2026-02-20",
		Description: "Buchung ohne Steuerschlüssel", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountKasse, Amount: 19000},
			{Side: domain.SideCredit, Account: domain.AccountUmsatzsteuer14c, Amount: 19000},
		},
	}
	if _, err := env.journal.Post(context.Background(), entry); err == nil {
		t.Fatal("das § 14c-Konto darf nicht ohne Steuerschlüssel bebucht werden")
	}
}

// Ein erneutes Speichern schreibt den offenen Entwurf fort, statt einen zweiten
// anzulegen: ein Zeitraum hat eine Anmeldung, nicht eine je Klick.
func TestVatReturnSaveUpdatesTheOpenDraft(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-02-10", 100000)

	svc := env.vatReturns(t)
	first, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("erstes Speichern: %v", err)
	}

	env.domesticInvoice(t, customer.ID, "2026-03-05", 200000)
	second, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("zweites Speichern: %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("das zweite Speichern legte die Anmeldung %d neu an statt %d fortzuschreiben", second.ID, first.ID)
	}

	stored, err := repository.NewVatReturnRepository(env.db).FindByID(ctx, first.ID)
	if err != nil {
		t.Fatalf("laden: %v", err)
	}
	if stored.Base("81") != 300000 {
		t.Errorf("Kz 81 der gespeicherten Anmeldung = %s, erwartet 3.000,00", stored.Base("81"))
	}
	if len(stored.EntryIDs("81")) != 2 {
		t.Errorf("der Drill-down führt %d Buchungen, erwartet 2", len(stored.EntryIDs("81")))
	}
	if stored.Payable != 57000 {
		t.Errorf("Zahllast = %s, erwartet 570,00", stored.Payable)
	}
}

// Für einen übermittelten Zeitraum gibt es keine zweite Erstanmeldung — auch
// nicht über einen Entwurf, der neben der übermittelten Anmeldung steht.
//
// Bestätigt würde sonst ein zweites Original desselben Zeitraums, ohne
// Kennziffer 10 und ohne Bezug auf das erste: das Finanzamt hätte zwei
// Erklärungen, von denen keine die andere ersetzt.
func TestSecondOriginalVatReturnCannotBeConfirmed(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-02-10", 200000)
	env.commitUntil(t, "2026-03-31")

	svc := env.vatReturns(t)
	first, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("speichern: %v", err)
	}
	if _, err := svc.ConfirmSubmitted(ctx, first.ID, "2026-04-09", "TT-1", ""); err != nil {
		t.Fatalf("bestätigen: %v", err)
	}

	// Ein Entwurf, der vor der Bestätigung entstanden ist und liegen blieb.
	repo := repository.NewVatReturnRepository(env.db)
	stale := &domain.VatReturn{
		FiscalYear: 2026, PeriodType: domain.VatPeriodQuarter, PeriodKey: "2026-Q1",
		PeriodFrom: "2026-01-01", PeriodTo: "2026-03-31", Status: domain.VatReturnDraft,
		Figures: first.Figures, Payable: first.Payable,
	}
	if err := repo.Create(ctx, stale); err != nil {
		t.Fatalf("zweiten Entwurf anlegen: %v", err)
	}

	_, err = svc.ConfirmSubmitted(ctx, stale.ID, "2026-04-20", "TT-2", "")
	if err == nil {
		t.Fatal("eine zweite Erstanmeldung desselben Zeitraums darf nicht bestätigt werden")
	}
	if !strings.Contains(err.Error(), "berichtigte") {
		t.Errorf("die Meldung sollte auf die Berichtigung verweisen: %v", err)
	}

	// Als Berichtigung geht derselbe Zeitraum durch — mit Kennziffer 10.
	correction, err := svc.CreateCorrection(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Berichtigung: %v", err)
	}
	if _, err := svc.ConfirmSubmitted(ctx, correction.ID, "2026-04-20", "TT-2", ""); err != nil {
		t.Fatalf("die Berichtigung muss bestätigt werden können: %v", err)
	}
}

// Bestätigt wird der Stand, der übermittelt wurde. Ein Entwurf, der vor
// weiteren Buchungen gespeichert wurde, ist keiner mehr.
func TestStaleVatReturnDraftCannotBeConfirmed(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-02-10", 200000)

	svc := env.vatReturns(t)
	saved, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("speichern: %v", err)
	}
	env.commitUntil(t, "2026-03-31")

	// Nach dem Speichern kommt eine Buchung des Zeitraums hinzu.
	env.domesticInvoice(t, customer.ID, "2026-03-01", 100000)

	_, err = svc.ConfirmSubmitted(ctx, saved.ID, "2026-04-09", "TT-1", "")
	if err == nil {
		t.Fatal("ein veralteter Entwurf darf nicht als übermittelt bestätigt werden")
	}
	if !strings.Contains(err.Error(), "Journal") {
		t.Errorf("die Meldung sollte die Abweichung zum Journal benennen: %v", err)
	}

	// Neu gespeichert stimmt er wieder und lässt sich bestätigen.
	again, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("erneut speichern: %v", err)
	}
	if again.ID != saved.ID {
		t.Errorf("das erneute Speichern legte den Entwurf %d neu an statt %d fortzuschreiben", again.ID, saved.ID)
	}
	if _, err := svc.ConfirmSubmitted(ctx, again.ID, "2026-04-09", "TT-1", ""); err != nil {
		t.Fatalf("der neu gespeicherte Entwurf muss bestätigt werden können: %v", err)
	}
}

// Wer vom Voranmeldungsverfahren befreit ist (§ 18 Abs. 2 Satz 3 UStG), gibt
// nur die Jahreserklärung ab. Ein Termin „Voranmeldung Jahr" wäre eine Pflicht,
// die es nicht gibt.
func TestYearlyVatPeriodHasNoAdvanceReturnDeadlines(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.setSettings(t, func(c *domain.CompanySettings) { c.VatPeriod = "year" })

	deadlines, err := env.deadlines(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	annual := 0
	for _, d := range deadlines {
		if strings.HasPrefix(d.Key, DeadlineKeyVatReturn+".") {
			t.Errorf("Termin %q (%s) verlangt eine Voranmeldung, die es beim Jahreszeitraum nicht gibt", d.Title, d.DueDate)
		}
		if strings.HasPrefix(d.Key, DeadlineKeyAnnualVat+".") {
			annual++
		}
	}
	if annual != 1 {
		t.Errorf("die Jahreserklärung steht %d-mal in der Liste, erwartet einmal", annual)
	}
}

// § 14c gilt auch auf dem Handbuchungsweg: eine Umsatzsteuerzeile zu einem
// Steuerfall ohne Steuerpflicht wird nicht gebucht. Sonst stünde derselbe
// Umsatz in Kennziffer 41 und in 81 — und in der Zusammenfassenden Meldung als
// steuerfrei, was die Voranmeldung als steuerpflichtig führt.
func TestManualEntryWithUnlawfulTaxIsRefused(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	entry := &domain.JournalEntry{
		BookingDate: "2026-02-20", DocumentDate: "2026-02-20",
		ServiceDateFrom: "2026-02-20", ServiceDateTo: "2026-02-20",
		Description:  "Innergemeinschaftliche Lieferung mit Steuerausweis",
		Source:       domain.EntrySourceManual,
		TaxTreatment: domain.TaxTreatmentIntraCommunitySupply,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountKasse, Amount: 119000},
			{Side: domain.SideCredit, Account: "4125", Amount: 100000},
			{
				Side: domain.SideCredit, Account: domain.AccountUmsatzsteuer19, Amount: 19000,
				TaxKey: "UST19",
			},
		},
	}
	_, err := env.journal.Post(ctx, entry)
	if err == nil {
		t.Fatal("eine Umsatzsteuerzeile zu einem steuerfreien Steuerfall darf nicht gebucht werden")
	}
	if !strings.Contains(err.Error(), "14c") {
		t.Errorf("die Meldung sollte § 14c UStG benennen: %v", err)
	}

	// Auch ohne gesetzten Steuerfall: das Erlöskonto 4125 ist die
	// innergemeinschaftliche Lieferung, und eine Handbuchung kommt ohne
	// Buchungsgruppe zustande. Ohne die Ableitung stünde derselbe Umsatz in
	// Kennziffer 41 und in 81.
	withoutTreatment := *entry
	withoutTreatment.TaxTreatment = ""
	withoutTreatment.Lines = append([]domain.JournalLine(nil), entry.Lines...)
	_, err = env.journal.Post(ctx, &withoutTreatment)
	if err == nil {
		t.Fatal("die Umsatzsteuerzeile zum Erlöskonto der ig. Lieferung darf auch ohne gesetzten Steuerfall nicht gebucht werden")
	}
	if !strings.Contains(err.Error(), "14c") {
		t.Errorf("die Meldung sollte § 14c UStG benennen: %v", err)
	}

	// Gegenprobe: enthält dieselbe Buchung daneben einen steuerpflichtigen
	// Inlandsumsatz (4400), gehört der Steuerausweis zu ihm. Die Ableitung darf
	// die richtige Buchung nicht treffen.
	mixed := *entry
	mixed.TaxTreatment = ""
	mixed.Lines = []domain.JournalLine{
		{Side: domain.SideDebit, Account: domain.AccountKasse, Amount: 219000},
		{Side: domain.SideCredit, Account: "4400", Amount: 100000},
		{Side: domain.SideCredit, Account: "4125", Amount: 100000},
		{
			Side: domain.SideCredit, Account: domain.AccountUmsatzsteuer19, Amount: 19000,
			TaxKey: "UST19",
		},
	}
	if _, err := env.journal.Post(ctx, &mixed); err != nil {
		t.Fatalf("der steuerpflichtige Inlandsumsatz muss buchbar bleiben: %v", err)
	}

	// Der Weg für den Betrag, der trotzdem geschuldet wird, bleibt offen: der
	// Steuerschlüssel UST14C auf dem eigenen Konto (Kennziffer 69).
	entry.TaxTreatment = ""
	entry.Lines = []domain.JournalLine{
		{Side: domain.SideDebit, Account: domain.AccountKasse, Amount: 19000},
		{
			Side: domain.SideCredit, Account: domain.AccountUmsatzsteuer14c, Amount: 19000,
			TaxKey: accounting.TaxKeyUnlawful,
		},
	}
	if _, err := env.journal.Post(ctx, entry); err != nil {
		t.Fatalf("die § 14c-Buchung muss möglich bleiben: %v", err)
	}
}

// Die Programmversion an der Anmeldung kommt aus dem Bau und trägt den
// Regelstand mit. Eine feste Bezeichnung änderte sich nie und beantwortete die
// Frage nicht, welcher Stand eine alte Anmeldung gerechnet hat.
func TestVatReturnCarriesProgramAndRuleVersion(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-02-10", 200000)

	ret, err := env.vatReturns(t).Draft(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Entwurf: %v", err)
	}
	if !strings.Contains(ret.ProgramVersion, buildinfo.Version) {
		t.Errorf("die Programmversion %q nennt die Fassung %q nicht", ret.ProgramVersion, buildinfo.Version)
	}
	if !strings.Contains(ret.ProgramVersion, accounting.PostingRuleVersion) {
		t.Errorf("die Programmversion %q nennt den Regelstand %q nicht",
			ret.ProgramVersion, accounting.PostingRuleVersion)
	}
	if len(ret.ProgramVersion) > 40 {
		t.Errorf("die Programmversion %q passt mit %d Zeichen nicht in die Spalte (40)",
			ret.ProgramVersion, len(ret.ProgramVersion))
	}
}
