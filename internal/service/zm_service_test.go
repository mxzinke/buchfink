package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func (e *testEnv) zmReturns(t *testing.T) *ZMService {
	t.Helper()
	return NewZMService(
		e.journalRepo,
		e.contactRepo,
		repository.NewSettingsRepository(e.db),
		repository.NewFestschreibungRepository(e.db),
		repository.NewZMReturnRepository(e.db),
		repository.NewVatReturnRepository(e.db),
		repository.NewAuditRepository(e.db),
		e.fiscalYear,
	)
}

// euInvoice stellt eine Rechnung mit einem Steuerfall ohne Steuerausweis aus.
func (e *testEnv) euInvoice(t *testing.T, customerID uint, date string, net domain.Cents, treatment domain.TaxTreatment) *domain.Invoice {
	t.Helper()
	inv := &domain.Invoice{
		ContactID: customerID, Date: date,
		ServiceDateFrom: date, ServiceDateTo: date,
		TaxTreatment: treatment,
		Items: []domain.InvoiceItem{{
			Description: "Leistung", QuantityMilli: 1000, UnitPrice: net, TaxRate: domain.TaxRateNone,
		}},
	}
	if err := e.invoices(t).Issue(context.Background(), inv); err != nil {
		t.Fatalf("Rechnung vom %s: %v", date, err)
	}
	return inv
}

// euService stellt eine sonstige Leistung an einen Unternehmer im übrigen
// Gemeinschaftsgebiet aus — den Umsatz, den die Zusammenfassende Meldung
// erfasst.
func (e *testEnv) euService(t *testing.T, customerID uint, date string, net domain.Cents) *domain.Invoice {
	t.Helper()
	return e.euInvoice(t, customerID, date, net, domain.TaxTreatmentReverseChargeSupply)
}

// Die Meldung entsteht aus den Buchungen der sonstigen Leistungen ins übrige
// Gemeinschaftsgebiet und stimmt sich gegen die Kennziffer 21 der Voranmeldung
// ab — beide Meldungen beschreiben denselben Umsatz.
func TestZMReconcilesAgainstTheVatReturn(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")

	env.euService(t, customer.ID, "2026-02-10", 500000)
	env.euService(t, customer.ID, "2026-03-05", 100000)

	// Die Voranmeldung desselben Quartals wird gespeichert, damit sich die
	// Meldung dagegen abstimmen kann.
	if _, err := env.vatReturns(t).Save(ctx, "2026-Q1"); err != nil {
		t.Fatalf("Voranmeldung speichern: %v", err)
	}

	zm, err := env.zmReturns(t).Draft(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("ZM-Entwurf: %v", err)
	}
	if len(zm.Lines) != 1 {
		t.Fatalf("erwartet eine Meldezeile, erhalten %d: %+v", len(zm.Lines), zm.Lines)
	}
	if zm.Lines[0].Kind != domain.ZMKindService {
		t.Errorf("Meldeart %q, erwartet %q", zm.Lines[0].Kind, domain.ZMKindService)
	}
	if zm.TotalServices != 600000 {
		t.Errorf("Summe = %s, erwartet 6.000,00", zm.TotalServices)
	}

	rec := zm.Reconciliation
	if rec == nil {
		t.Fatal("die Abstimmung fehlt")
	}
	if rec.ServicesVat != 600000 {
		t.Errorf("Kz 21 der Voranmeldung = %s, erwartet 6.000,00", rec.ServicesVat)
	}
	if rec.ServicesDifference() != 0 {
		t.Errorf("Abweichung = %s, erwartet null", rec.ServicesDifference())
	}
}

// Der Meldezeitraum folgt der Leistung, das Geschäftsjahr dem Buchungsdatum:
// die im Januar erfasste Dezemberleistung gehört in die Meldung des vierten
// Quartals und nicht in die des ersten.
func TestZMReadsAcrossFiscalYears(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")

	late := &domain.JournalEntry{
		BookingDate: "2027-01-08", DocumentDate: "2027-01-08",
		ServiceDateFrom: "2026-12-18", ServiceDateTo: "2026-12-18",
		Description:  "Dezemberleistung, im Januar erfasst",
		Source:       domain.EntrySourceManual,
		TaxTreatment: domain.TaxTreatmentReverseChargeSupply,
		ContactID:    &customer.ID,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 200000},
			{Side: domain.SideCredit, Account: "4337", Amount: 200000},
		},
	}
	if _, err := env.journal.Post(ctx, late); err != nil {
		t.Fatalf("Buchung im Folgejahr: %v", err)
	}

	zm, err := env.zmReturns(t).Draft(ctx, "2026-Q4")
	if err != nil {
		t.Fatalf("ZM-Entwurf: %v", err)
	}
	if zm.TotalServices != 200000 {
		t.Errorf("Summe im vierten Quartal = %s, erwartet 2.000,00", zm.TotalServices)
	}
}

// Eine Abweichung gegen die Voranmeldung wird angezeigt und nicht ausgeglichen:
// wenn beide auseinandergehen, ist eine von beiden falsch, und welche, weiß nur
// der Anwender.
func TestZMShowsDifferenceAgainstTheVatReturn(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euService(t, customer.ID, "2026-02-10", 500000)

	if _, err := env.vatReturns(t).Save(ctx, "2026-Q1"); err != nil {
		t.Fatalf("Voranmeldung speichern: %v", err)
	}
	// Nach der Anmeldung kommt eine weitere Leistung hinzu; die gespeicherte
	// Anmeldung kennt sie nicht.
	env.euService(t, customer.ID, "2026-03-20", 200000)

	zm, err := env.zmReturns(t).Draft(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("ZM-Entwurf: %v", err)
	}
	if zm.Reconciliation.ServicesDifference() != 200000 {
		t.Errorf("Abweichung = %s, erwartet 2.000,00", zm.Reconciliation.ServicesDifference())
	}
}

// Fehlt die USt-IdNr., ist die Meldung nicht bestätigbar: ohne sie nimmt das
// BZSt-Portal sie nicht an.
func TestZMWithMissingVatIDCannotBeConfirmed(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euService(t, customer.ID, "2026-02-10", 500000)

	svc := env.zmReturns(t)
	saved, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("speichern: %v", err)
	}
	env.commitUntil(t, "2026-03-31")

	// Die USt-IdNr. wird nachträglich aus den Stammdaten entfernt.
	customer.VatID = ""
	if err := env.contacts.SaveContact(ctx, customer); err != nil {
		t.Fatalf("Stammdaten ändern: %v", err)
	}

	draft, err := svc.Draft(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Entwurf: %v", err)
	}
	if len(draft.Findings) != 1 {
		t.Fatalf("erwartet einen Befund zur fehlenden USt-IdNr., erhalten %v", draft.Findings)
	}

	if _, err := svc.ConfirmSubmitted(ctx, saved.ID, "2026-04-24", "TT-ZM", ""); err == nil {
		t.Fatal("eine Meldung mit fehlender USt-IdNr. darf nicht bestätigt werden")
	} else if !strings.Contains(err.Error(), "USt-IdNr") {
		t.Errorf("die Meldung sollte die fehlende USt-IdNr. benennen: %v", err)
	}
}

// Die Datei hat die vier Spalten des BZSt-Online-Portals, die Beträge in
// vollen Euro.
func TestZMCSVFormat(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	french := env.customer(t, "Client SARL", "FR", "FR12345678901")
	italian := env.customer(t, "Cliente SRL", "IT", "IT00743110157")
	env.euService(t, french.ID, "2026-02-10", 500049)
	env.euService(t, italian.ID, "2026-03-05", 100000)

	svc := env.zmReturns(t)
	saved, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("speichern: %v", err)
	}
	csv, err := svc.ExportCSV(ctx, saved.ID)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(csv), "\n")
	if lines[0] != "Laenderkennzeichen;USt-IdNr.;Betrag;Art" {
		t.Errorf("Kopfzeile = %q", lines[0])
	}
	if len(lines) != 3 {
		t.Fatalf("erwartet Kopfzeile und zwei Meldezeilen, erhalten %d:\n%s", len(lines), csv)
	}
	if !strings.Contains(csv, "FR;12345678901;5000;S\n") {
		t.Errorf("die französische Zeile fehlt oder ist falsch gerundet (5.000,49 € → 5000):\n%s", csv)
	}
	if !strings.Contains(csv, "IT;00743110157;1000;S\n") {
		t.Errorf("die italienische Zeile fehlt:\n%s", csv)
	}
}

// Bei Griechenland weicht das Präfix der USt-IdNr. („EL") vom Länderkennzeichen
// des Kontakts („GR") ab. Die Datei muss „EL;123456789" enthalten: nähme sie das
// Kontaktland, bliebe das Präfix in der Nummer stehen („GR;EL123456789") und das
// BZSt-Portal wiese die Zeile zurück.
func TestZMCSVUsesTheVatIDPrefixForGreece(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Hellenic AE", "GR", "EL123456789")
	env.euService(t, customer.ID, "2026-02-10", 500000)

	svc := env.zmReturns(t)
	saved, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("speichern: %v", err)
	}
	csv, err := svc.ExportCSV(ctx, saved.ID)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !strings.Contains(csv, "EL;123456789;5000;S\n") {
		t.Errorf("erwartet die Zeile \"EL;123456789;5000;S\":\n%s", csv)
	}
}

// Eine bestätigte Meldung ist unveränderlich; geändert wird sie über eine
// berichtigte Meldung.
func TestZMSubmissionIsImmutableAndCorrectable(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euService(t, customer.ID, "2026-02-10", 500000)

	svc := env.zmReturns(t)
	saved, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("speichern: %v", err)
	}
	env.commitUntil(t, "2026-03-31")
	if _, err := svc.ConfirmSubmitted(ctx, saved.ID, "2026-04-24", "TT-ZM", ""); err != nil {
		t.Fatalf("bestätigen: %v", err)
	}
	if _, err := svc.ConfirmSubmitted(ctx, saved.ID, "2026-04-25", "TT-ZM-2", ""); err == nil {
		t.Fatal("eine bestätigte Meldung darf nicht ein zweites Mal bestätigt werden")
	}

	env.euService(t, customer.ID, "2026-03-20", 200000)
	correction, err := svc.CreateCorrection(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Berichtigung: %v", err)
	}
	if !correction.IsCorrection || correction.CorrectsID == nil || *correction.CorrectsID != saved.ID {
		t.Errorf("die Berichtigung muss auf die ursprüngliche Meldung verweisen: %+v", correction)
	}
	if correction.TotalServices != 700000 {
		t.Errorf("Summe der Berichtigung = %s, erwartet 7.000,00", correction.TotalServices)
	}
}

// Sonstige Leistungen werden vierteljährlich gemeldet, unabhängig von ihrer
// Höhe (§ 18a Abs. 1 Satz 1 UStG).
func TestZMPeriodsAreQuarters(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euService(t, customer.ID, "2026-02-10", 9_000_000)

	periods, err := env.zmReturns(t).Periods(ctx, 2026)
	if err != nil {
		t.Fatalf("Zeiträume: %v", err)
	}
	if len(periods) != 4 {
		t.Fatalf("erwartet vier Quartale, erhalten %d", len(periods))
	}
	if periods[0].Total != 9_000_000 {
		t.Errorf("Summe Q1 = %s, erwartet 90.000,00", periods[0].Total)
	}
	if periods[0].DueDate != "2026-04-27" {
		t.Errorf("Fälligkeit Q1 = %s, erwartet 2026-04-27 (der 25. April 2026 ist ein Samstag)", periods[0].DueDate)
	}
}

// Ein erneutes Speichern schreibt denselben Entwurf fort, statt einen zweiten
// anzulegen — und ersetzt seine Zeilen, statt sie zu ergänzen. Eine Zeile, die
// aus dem Journal verschwunden ist, darf nicht als Rest stehen bleiben.
func TestZMSaveReplacesTheOpenDraft(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euService(t, customer.ID, "2026-02-10", 500000)

	svc := env.zmReturns(t)
	first, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("erstes Speichern: %v", err)
	}

	env.euService(t, customer.ID, "2026-03-20", 200000)
	second, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("zweites Speichern: %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("das zweite Speichern legte die Meldung %d neu an statt %d fortzuschreiben", second.ID, first.ID)
	}

	stored, err := repository.NewZMReturnRepository(env.db).FindByID(ctx, first.ID)
	if err != nil {
		t.Fatalf("laden: %v", err)
	}
	if len(stored.Lines) != 1 {
		t.Fatalf("die gespeicherte Meldung hat %d Zeilen, erwartet 1: %+v", len(stored.Lines), stored.Lines)
	}
	if stored.Lines[0].Amount != 700000 {
		t.Errorf("Betrag = %s, erwartet 7.000,00", stored.Lines[0].Amount)
	}
	if len(stored.Lines[0].EntryIDs) != 2 {
		t.Errorf("die Zeile führt %d Buchungen, erwartet 2 — ohne sie gibt es keinen Drill-down", len(stored.Lines[0].EntryIDs))
	}
}

// Bestätigt wird nur, was festgeschrieben ist — dieselbe Bedingung wie bei der
// Voranmeldung (§ 18a UStG kennt keine eigene, aber eine Meldung, deren
// Zeitraum sich noch ändern kann, entfernt sich nach der Bestätigung
// unbemerkt vom Journal).
func TestZMConfirmationRequiresCommittedPeriod(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euService(t, customer.ID, "2026-02-10", 500000)

	svc := env.zmReturns(t)
	saved, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("speichern: %v", err)
	}

	_, err = svc.ConfirmSubmitted(ctx, saved.ID, "2026-04-24", "TT-ZM", "")
	if err == nil {
		t.Fatal("ohne Festschreibung darf keine Übermittlung bestätigt werden")
	}
	if !strings.Contains(err.Error(), "festgeschrieben") {
		t.Errorf("die Meldung sollte die fehlende Festschreibung benennen: %v", err)
	}

	// Die Oberfläche sperrt die Bestätigung an derselben Bedingung. Ohne diese
	// Auskunft erführe der Anwender die fehlende Festschreibung erst, nachdem er
	// Datum und Transferticket eingetippt hat.
	periods, err := svc.Periods(ctx, 2026)
	if err != nil {
		t.Fatalf("Meldezeiträume: %v", err)
	}
	if q1 := zmPeriod(t, periods, "2026-Q1"); q1.Committed {
		t.Error("ohne Festschreibung meldet der Zeitraum committed=false")
	}

	env.commitUntil(t, "2026-03-31")
	if _, err := svc.ConfirmSubmitted(ctx, saved.ID, "2026-04-24", "TT-ZM", ""); err != nil {
		t.Fatalf("nach der Festschreibung muss die Bestätigung durchgehen: %v", err)
	}

	periods, err = svc.Periods(ctx, 2026)
	if err != nil {
		t.Fatalf("Meldezeiträume: %v", err)
	}
	if q1 := zmPeriod(t, periods, "2026-Q1"); !q1.Committed {
		t.Error("nach der Festschreibung meldet der Zeitraum committed=true")
	}
	if q2 := zmPeriod(t, periods, "2026-Q2"); q2.Committed {
		t.Error("das zweite Quartal ist nicht festgeschrieben")
	}
}

// zmPeriod sucht einen Meldezeitraum heraus.
func zmPeriod(t *testing.T, periods []ZMPeriodStatus, key string) ZMPeriodStatus {
	t.Helper()
	for _, p := range periods {
		if p.Key == key {
			return p
		}
	}
	t.Fatalf("Meldezeitraum %s nicht gefunden", key)
	return ZMPeriodStatus{}
}

// Ein Umsatz in einem bereits übermittelten Meldezeitraum erscheint als
// Nachtrag und nicht stillschweigend im laufenden Zeitraum: § 18a Abs. 10 UStG
// verlangt die Berichtigung der ursprünglichen Meldung.
func TestZMLateEntryAppearsAsNachtrag(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euService(t, customer.ID, "2026-02-10", 500000)

	svc := env.zmReturns(t)
	saved, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("speichern: %v", err)
	}
	env.commitUntil(t, "2026-03-31")
	if _, err := svc.ConfirmSubmitted(ctx, saved.ID, "2026-04-24", "TT-ZM", ""); err != nil {
		t.Fatalf("bestätigen: %v", err)
	}

	// Ein Nachzügler, der in das übermittelte Quartal gehört.
	late := env.euService(t, customer.ID, "2026-03-25", 100000)

	second, err := svc.Draft(ctx, "2026-Q2")
	if err != nil {
		t.Fatalf("ZM-Entwurf Q2: %v", err)
	}
	if len(second.Lines) != 0 {
		t.Errorf("der Nachtrag gehört nicht in die Meldung des laufenden Zeitraums: %+v", second.Lines)
	}
	if len(second.LateEntries) != 1 {
		t.Fatalf("erwartet einen Nachtrag, erhalten %+v", second.LateEntries)
	}
	nachtrag := second.LateEntries[0]
	if nachtrag.PeriodKey != "2026-Q1" || nachtrag.Amount != 100000 {
		t.Errorf("Nachtrag = %+v, erwartet 1.000,00 € im Zeitraum 2026-Q1", nachtrag)
	}
	if nachtrag.EntryID != *late.JournalEntryID {
		t.Errorf("der Nachtrag verweist auf die Buchung %d, erwartet %d", nachtrag.EntryID, *late.JournalEntryID)
	}

	// Die Berichtigung des Quartals nimmt ihn auf und führt ihn dort nicht mehr
	// als Nachtrag.
	correction, err := svc.CreateCorrection(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Berichtigung: %v", err)
	}
	if correction.TotalServices != 600000 {
		t.Errorf("Summe der Berichtigung = %s, erwartet 6.000,00", correction.TotalServices)
	}
	if len(correction.LateEntries) != 0 {
		t.Errorf("die Berichtigung meldet den Zeitraum vollständig neu und führt keine Nachträge: %+v", correction.LateEntries)
	}
}

// Wer monatlich voranmeldet, hat im ZM-Quartal drei Anmeldungen; abgestimmt
// wird gegen ihre Summe.
func TestZMReconcilesAgainstMonthlyVatReturns(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euService(t, customer.ID, "2026-01-15", 400000)
	env.euService(t, customer.ID, "2026-03-20", 200000)

	for _, key := range []string{"2026-01", "2026-02", "2026-03"} {
		if _, err := env.vatReturns(t).Save(ctx, key); err != nil {
			t.Fatalf("Voranmeldung %s speichern: %v", key, err)
		}
	}

	zm, err := env.zmReturns(t).Draft(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("ZM-Entwurf: %v", err)
	}
	rec := zm.Reconciliation
	if rec == nil || rec.VatReturnsFound != 3 {
		t.Fatalf("die Abstimmung fand %+v, erwartet drei Monatsanmeldungen", rec)
	}
	if rec.ServicesZM != 600000 || rec.ServicesVat != 600000 {
		t.Errorf("Summen = ZM %s / Kz 21 %s, erwartet je 6.000,00", rec.ServicesZM, rec.ServicesVat)
	}
}

// Wer jährlich anmeldet, hat für ein ZM-Quartal keine Anmeldung. Abgestimmt
// wird dann auf der Ebene des Jahres — sonst stünde jedes Quartal mit seiner
// vollen Summe als Abweichung da.
func TestZMReconcilesAgainstTheAnnualVatReturn(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euService(t, customer.ID, "2026-02-10", 400000)
	env.euService(t, customer.ID, "2026-08-20", 200000)

	if _, err := env.vatReturns(t).Save(ctx, "2026"); err != nil {
		t.Fatalf("Jahresanmeldung speichern: %v", err)
	}

	zm, err := env.zmReturns(t).Draft(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("ZM-Entwurf: %v", err)
	}
	rec := zm.Reconciliation
	if rec == nil || rec.VatReturnsFound != 1 {
		t.Fatalf("die Abstimmung fand %+v, erwartet die Jahresanmeldung", rec)
	}
	if rec.ScopeKey != "2026" {
		t.Errorf("abgestimmt wurde über %q, erwartet 2026", rec.ScopeKey)
	}
	if rec.ServicesZM != 600000 || rec.ServicesVat != 600000 {
		t.Errorf("Summen = ZM %s / Kz 21 %s, erwartet je 6.000,00", rec.ServicesZM, rec.ServicesVat)
	}
	if rec.ServicesDifference() != 0 {
		t.Errorf("Abweichung = %s, erwartet null", rec.ServicesDifference())
	}
}
