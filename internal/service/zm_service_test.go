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

// Die Meldung entsteht aus den Buchungen mit den beiden meldepflichtigen
// Steuerfällen und stimmt sich gegen die Kennziffern 41 und 21 der
// Voranmeldung ab — beide Meldungen beschreiben denselben Umsatz.
func TestZMReconcilesAgainstTheVatReturn(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")

	env.euInvoice(t, customer.ID, "2026-02-10", 500000, domain.TaxTreatmentIntraCommunitySupply)
	env.euInvoice(t, customer.ID, "2026-03-05", 100000, domain.TaxTreatmentReverseChargeSupply)

	// Die Voranmeldung desselben Quartals wird gespeichert, damit sich die
	// Meldung dagegen abstimmen kann.
	if _, err := env.vatReturns(t).Save(ctx, "2026-Q1"); err != nil {
		t.Fatalf("Voranmeldung speichern: %v", err)
	}

	zm, err := env.zmReturns(t).Draft(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("ZM-Entwurf: %v", err)
	}
	if len(zm.Lines) != 2 {
		t.Fatalf("erwartet zwei Meldezeilen, erhalten %d: %+v", len(zm.Lines), zm.Lines)
	}
	if zm.TotalSupplies != 500000 || zm.TotalServices != 100000 {
		t.Errorf("Summen = %s Lieferungen / %s Leistungen, erwartet 5.000,00 / 1.000,00",
			zm.TotalSupplies, zm.TotalServices)
	}

	rec := zm.Reconciliation
	if rec == nil {
		t.Fatal("die Abstimmung fehlt")
	}
	if rec.SuppliesVat != 500000 || rec.ServicesVat != 100000 {
		t.Errorf("Kennziffern der Voranmeldung: Kz 41 = %s, Kz 21 = %s, erwartet 5.000,00 / 1.000,00",
			rec.SuppliesVat, rec.ServicesVat)
	}
	if rec.SuppliesDifference() != 0 || rec.ServicesDifference() != 0 {
		t.Errorf("Abweichung = %s / %s, erwartet null in beiden Richtungen",
			rec.SuppliesDifference(), rec.ServicesDifference())
	}
}

// Der Meldezeitraum folgt der Leistung, das Geschäftsjahr dem Buchungsdatum:
// die im Januar erfasste Dezemberlieferung gehört in die Meldung des vierten
// Quartals und nicht in die des ersten.
func TestZMReadsAcrossFiscalYears(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")

	late := &domain.JournalEntry{
		BookingDate: "2027-01-08", DocumentDate: "2027-01-08",
		ServiceDateFrom: "2026-12-18", ServiceDateTo: "2026-12-18",
		Description:  "Dezemberlieferung, im Januar erfasst",
		Source:       domain.EntrySourceManual,
		TaxTreatment: domain.TaxTreatmentIntraCommunitySupply,
		ContactID:    &customer.ID,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountKasse, Amount: 200000},
			{Side: domain.SideCredit, Account: "4125", Amount: 200000},
		},
	}
	if _, err := env.journal.Post(ctx, late); err != nil {
		t.Fatalf("Buchung im Folgejahr: %v", err)
	}

	zm, err := env.zmReturns(t).Draft(ctx, "2026-Q4")
	if err != nil {
		t.Fatalf("ZM-Entwurf: %v", err)
	}
	if zm.TotalSupplies != 200000 {
		t.Errorf("Summe der Lieferungen im vierten Quartal = %s, erwartet 2.000,00", zm.TotalSupplies)
	}
}

// Eine Abweichung gegen die Voranmeldung wird angezeigt und nicht ausgeglichen:
// wenn beide auseinandergehen, ist eine von beiden falsch, und welche, weiß nur
// der Anwender.
func TestZMShowsDifferenceAgainstTheVatReturn(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euInvoice(t, customer.ID, "2026-02-10", 500000, domain.TaxTreatmentIntraCommunitySupply)

	if _, err := env.vatReturns(t).Save(ctx, "2026-Q1"); err != nil {
		t.Fatalf("Voranmeldung speichern: %v", err)
	}
	// Nach der Anmeldung kommt eine weitere Lieferung hinzu; die gespeicherte
	// Anmeldung kennt sie nicht.
	env.euInvoice(t, customer.ID, "2026-03-20", 200000, domain.TaxTreatmentIntraCommunitySupply)

	zm, err := env.zmReturns(t).Draft(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("ZM-Entwurf: %v", err)
	}
	if zm.Reconciliation.SuppliesDifference() != 200000 {
		t.Errorf("Abweichung = %s, erwartet 2.000,00", zm.Reconciliation.SuppliesDifference())
	}
}

// Fehlt die USt-IdNr., ist die Meldung nicht bestätigbar: ohne sie nimmt das
// BZSt-Portal sie nicht an, und die Steuerbefreiung der Lieferung ist nicht
// belegt.
func TestZMWithMissingVatIDCannotBeConfirmed(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euInvoice(t, customer.ID, "2026-02-10", 500000, domain.TaxTreatmentIntraCommunitySupply)

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

// Die Datei trägt die vier Spalten des BZSt-Online-Portals, die Beträge in
// vollen Euro.
func TestZMCSVFormat(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euInvoice(t, customer.ID, "2026-02-10", 500049, domain.TaxTreatmentIntraCommunitySupply)
	env.euInvoice(t, customer.ID, "2026-03-05", 100000, domain.TaxTreatmentReverseChargeSupply)

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
	if !strings.Contains(csv, "FR;12345678901;5000;L\n") {
		t.Errorf("die Lieferzeile fehlt oder ist falsch gerundet (5.000,49 € → 5000):\n%s", csv)
	}
	if !strings.Contains(csv, "FR;12345678901;1000;S\n") {
		t.Errorf("die Leistungszeile fehlt:\n%s", csv)
	}
}

// Bei Griechenland weicht das Präfix der USt-IdNr. („EL") vom Länderkennzeichen
// des Kontakts („GR") ab. Die Datei muss „EL;123456789" tragen: nähme sie das
// Kontaktland, bliebe das Präfix in der Nummer stehen („GR;EL123456789") und das
// BZSt-Portal wiese die Zeile zurück.
func TestZMCSVUsesTheVatIDPrefixForGreece(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Hellenic AE", "GR", "EL123456789")
	env.euInvoice(t, customer.ID, "2026-02-10", 500000, domain.TaxTreatmentIntraCommunitySupply)

	svc := env.zmReturns(t)
	saved, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("speichern: %v", err)
	}
	csv, err := svc.ExportCSV(ctx, saved.ID)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !strings.Contains(csv, "EL;123456789;5000;L\n") {
		t.Errorf("erwartet die Zeile \"EL;123456789;5000;L\":\n%s", csv)
	}
}

// Eine bestätigte Meldung ist unveränderlich; geändert wird sie über eine
// berichtigte Meldung.
func TestZMSubmissionIsImmutableAndCorrectable(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euInvoice(t, customer.ID, "2026-02-10", 500000, domain.TaxTreatmentIntraCommunitySupply)

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

	env.euInvoice(t, customer.ID, "2026-03-20", 200000, domain.TaxTreatmentIntraCommunitySupply)
	correction, err := svc.CreateCorrection(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Berichtigung: %v", err)
	}
	if !correction.IsCorrection || correction.CorrectsID == nil || *correction.CorrectsID != saved.ID {
		t.Errorf("die Berichtigung muss auf die ursprüngliche Meldung verweisen: %+v", correction)
	}
	if correction.TotalSupplies != 700000 {
		t.Errorf("Summe der Berichtigung = %s, erwartet 7.000,00", correction.TotalSupplies)
	}
}

// Die Länge des Meldezeitraums folgt aus den Umsätzen: über 50.000 Euro
// Lieferungen im Quartal wird monatlich gemeldet (§ 18a Abs. 1 Satz 2 UStG).
func TestZMPeriodsFollowTheThreshold(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euInvoice(t, customer.ID, "2026-02-10", 4_000_000, domain.TaxTreatmentIntraCommunitySupply)

	svc := env.zmReturns(t)
	periods, err := svc.Periods(ctx, 2026)
	if err != nil {
		t.Fatalf("Zeiträume: %v", err)
	}
	if len(periods) != 4 {
		t.Fatalf("unter der Grenze erwartet vier Quartale, erhalten %d", len(periods))
	}
	if periods[0].Total != 4_000_000 {
		t.Errorf("Summe Q1 = %s, erwartet 40.000,00", periods[0].Total)
	}
	if periods[0].DueDate != "2026-04-27" {
		t.Errorf("Fälligkeit Q1 = %s, erwartet 2026-04-27 (der 25. April 2026 ist ein Samstag)", periods[0].DueDate)
	}

	env.euInvoice(t, customer.ID, "2026-02-20", 2_000_000, domain.TaxTreatmentIntraCommunitySupply)
	periods, err = svc.Periods(ctx, 2026)
	if err != nil {
		t.Fatalf("Zeiträume: %v", err)
	}
	if len(periods) != 12 {
		t.Fatalf("über der Grenze erwartet zwölf Monate, erhalten %d", len(periods))
	}
}

// Ein erneutes Speichern schreibt denselben Entwurf fort, statt einen zweiten
// anzulegen — und ersetzt seine Zeilen, statt sie zu ergänzen. Eine Zeile, die
// aus dem Journal verschwunden ist, darf nicht als Rest stehen bleiben.
func TestZMSaveReplacesTheOpenDraft(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")
	env.euInvoice(t, customer.ID, "2026-02-10", 500000, domain.TaxTreatmentIntraCommunitySupply)

	svc := env.zmReturns(t)
	first, err := svc.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("erstes Speichern: %v", err)
	}

	env.euInvoice(t, customer.ID, "2026-03-20", 200000, domain.TaxTreatmentIntraCommunitySupply)
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
	env.euInvoice(t, customer.ID, "2026-02-10", 500000, domain.TaxTreatmentIntraCommunitySupply)

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
	env.euInvoice(t, customer.ID, "2026-02-10", 500000, domain.TaxTreatmentIntraCommunitySupply)

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
	late := env.euInvoice(t, customer.ID, "2026-03-25", 100000, domain.TaxTreatmentIntraCommunitySupply)

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
	if nachtrag.PeriodKey != "2026-Q1" || nachtrag.Amount != 100000 || nachtrag.Kind != domain.ZMKindSupply {
		t.Errorf("Nachtrag = %+v, erwartet 1.000,00 € Lieferung im Zeitraum 2026-Q1", nachtrag)
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
	if correction.TotalSupplies != 600000 {
		t.Errorf("Summe der Berichtigung = %s, erwartet 6.000,00", correction.TotalSupplies)
	}
	if len(correction.LateEntries) != 0 {
		t.Errorf("die Berichtigung meldet den Zeitraum vollständig neu und führt keine Nachträge: %+v", correction.LateEntries)
	}
}

// Wer monatlich meldet und vierteljährlich voranmeldet, hat für einen ZM-Monat
// keine Anmeldung. Abgestimmt wird dann auf der Ebene des Quartals — sonst
// stünde in genau der Lage, für die es die Monatsmeldung gibt, immer die volle
// Summe als Abweichung.
func TestZMMonthlyReconcilesAgainstTheQuarterlyVatReturn(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")

	// Über der Grenze des § 18a Abs. 1 Satz 2 UStG: monatliche Meldung.
	env.euInvoice(t, customer.ID, "2026-01-15", 4_000_000, domain.TaxTreatmentIntraCommunitySupply)
	env.euInvoice(t, customer.ID, "2026-02-20", 2_000_000, domain.TaxTreatmentIntraCommunitySupply)

	// Die Voranmeldung bleibt vierteljährlich.
	if _, err := env.vatReturns(t).Save(ctx, "2026-Q1"); err != nil {
		t.Fatalf("Voranmeldung speichern: %v", err)
	}

	zm, err := env.zmReturns(t).Draft(ctx, "2026-01")
	if err != nil {
		t.Fatalf("ZM-Entwurf Januar: %v", err)
	}
	rec := zm.Reconciliation
	if rec == nil || rec.VatReturnsFound != 1 {
		t.Fatalf("die Abstimmung fand %+v, erwartet die Quartalsanmeldung", rec)
	}
	if rec.ScopeKey != "2026-Q1" {
		t.Errorf("abgestimmt wurde über %q, erwartet 2026-Q1", rec.ScopeKey)
	}
	if rec.SuppliesZM != 6_000_000 || rec.SuppliesVat != 6_000_000 {
		t.Errorf("Summen = ZM %s / Kz 41 %s, erwartet je 60.000,00", rec.SuppliesZM, rec.SuppliesVat)
	}
	if rec.SuppliesDifference() != 0 {
		t.Errorf("Abweichung = %s, erwartet null", rec.SuppliesDifference())
	}
}
