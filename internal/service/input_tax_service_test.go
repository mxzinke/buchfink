package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die Berichtigung eines abweichenden Wirtschaftsjahres wird zu seinem letzten
// Tag gebucht und nicht zum ersten des folgenden — sonst fiele sie in die
// Voranmeldung des nächsten Jahres.
func TestInputTaxCorrectionBooksOnTheLastDayOfTheFiscalYear(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	settings := repository.NewSettingsRepository(env.db)
	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Einstellungen: %v", err)
	}
	cfg.FiscalYearStartMonth = 7
	if err := settings.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatalf("Einstellungen setzen: %v", err)
	}

	svc := env.inputTax(t)
	date, err := svc.bookingDate(ctx, 2026)
	if err != nil {
		t.Fatalf("Buchungsdatum: %v", err)
	}
	if date != "2027-06-30" {
		t.Errorf("Buchungsdatum %q — ein Wirtschaftsjahr, das im Juli beginnt, endet am 30.06.", date)
	}
}

// -------------------------------------------------------------------------
// § 15a UStG: nach dem Storno wird erneut berichtigt
// -------------------------------------------------------------------------

// Der Storno ist der Weg zurück, den jede Sperre dieser Bausteine empfiehlt.
// Solange die Buchungskennung am Verwendungsanteil stehen blieb, führte er in
// die Sackgasse: das Verzeichnis meldete weiter „gebucht", der Jahreslauf fand
// nichts zu tun, und die Berichtigung dieses Jahres kam nie mehr in die
// Kennziffer 64.
func TestInputTaxCorrectionCanBeBookedAgainAfterAReversal(t *testing.T) {
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
	if _, err := svc.SaveUsage(ctx, SaveInputTaxUsageRequest{
		CorrectionID: correction.ID, FiscalYear: 2028, Permille: 600,
		Reason: "Kfz ab 2028 zu 40 % für steuerfreie Vermietung eingesetzt",
	}); err != nil {
		t.Fatalf("Verwendungsanteil: %v", err)
	}
	if _, err := svc.BookYear(ctx, 2028); err != nil {
		t.Fatalf("erste Buchung: %v", err)
	}

	first := env.closingEntry(t, 2028, inputTaxCorrectionDocument(2028))
	if _, err := env.journal.Reverse(ctx, first.ID,
		"Verwendungsanteil beruhte auf einem falschen Fahrtenbuch"); err != nil {
		t.Fatalf("Generalumkehr: %v", err)
	}

	// Nach dem Storno ist das Jahr wieder offen — und zwar in allen drei
	// Antworten des Dienstes.
	view, err := svc.Year(ctx, 2028)
	if err != nil {
		t.Fatalf("Verzeichnis: %v", err)
	}
	if view.Rows[0].Booked {
		t.Error("eine per Generalumkehr zurückgenommene Berichtigung ist keine gebuchte")
	}
	if view.Rows[0].EntryNumber != "" {
		t.Errorf("Buchungsnummer %q — die Buchung steht nicht mehr", view.Rows[0].EntryNumber)
	}
	if view.TotalAmount != -60_800 {
		t.Errorf("zu buchen %s € — nach dem Storno steht die Berichtigung wieder an",
			view.TotalAmount)
	}

	// Der Anteil lässt sich wieder ändern: das war der Rat der Meldung.
	if _, err := svc.SaveUsage(ctx, SaveInputTaxUsageRequest{
		CorrectionID: correction.ID, FiscalYear: 2028, Permille: 500,
		Reason: "Fahrtenbuch berichtigt: nur noch 50 % abziehbare Verwendung",
	}); err != nil {
		t.Fatalf("Anteil nach dem Storno ändern: %v", err)
	}

	booked, err := svc.BookYear(ctx, 2028)
	if err != nil {
		t.Fatalf("zweite Buchung: %v", err)
	}
	if !booked.Rows[0].Booked {
		t.Fatal("nach dem zweiten Lauf muss die Berichtigung wieder als gebucht gelten")
	}
	// 7.600 € × 50 Prozentpunkte / 5 Jahre = 760 € zurückzuzahlen.
	if booked.Rows[0].Assessment.Amount != -76_000 {
		t.Errorf("Berichtigung %s € — erwartet -760,00 € aus dem geänderten Anteil",
			booked.Rows[0].Assessment.Amount)
	}

	second := env.standingClosingEntry(t, 2028, inputTaxCorrectionDocument(2028))
	if second.ID == first.ID {
		t.Fatal("die zweite Berichtigung muss eine neue Buchung sein")
	}
	var taxLine *domain.JournalLine
	for i := range second.Lines {
		if second.Lines[i].TaxKey == accounting.TaxKeyInputTaxCorrection {
			taxLine = &second.Lines[i]
		}
	}
	if taxLine == nil || taxLine.Amount != 76_000 {
		t.Fatalf("die neue Buchung hat keine Zeile mit %s über 760,00 €: %+v",
			accounting.TaxKeyInputTaxCorrection, second.Lines)
	}
}

// -------------------------------------------------------------------------
// Die Berichtigung eines Jahres wird nicht zweimal gebucht
// -------------------------------------------------------------------------

// Bricht ein Lauf zwischen Buchung und Vermerk ab, steht die Berichtigung im
// Journal und gilt im Verzeichnis als offen. Der nächste Lauf darf sie dann
// nicht ein zweites Mal buchen — sie stünde doppelt in Kennziffer 64.
func TestInputTaxCorrectionIsNotBookedTwiceWhenTheNoteIsMissing(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	repo := repository.NewInputTaxCorrectionRepository(env.db)
	svc := env.inputTax(t)

	correction, err := svc.Register(ctx, RegisterInputTaxRequest{
		Label: "Pkw AN-2026-0001", Account: "0520",
		AcquisitionDate: "2026-01-15",
		NetAmount:       4_000_000, InputTaxAmount: 760_000,
	})
	if err != nil {
		t.Fatalf("Aufnahme ins Verzeichnis: %v", err)
	}
	if _, err := svc.SaveUsage(ctx, SaveInputTaxUsageRequest{
		CorrectionID: correction.ID, FiscalYear: 2028, Permille: 600,
		Reason: "Kfz ab 2028 zu 40 % für steuerfreie Vermietung eingesetzt",
	}); err != nil {
		t.Fatalf("Verwendungsanteil: %v", err)
	}
	if _, err := svc.BookYear(ctx, 2028); err != nil {
		t.Fatalf("erste Buchung: %v", err)
	}

	// Der Vermerk fällt weg — der Zustand, den ein Abbruch zwischen Buchung und
	// Vermerk hinterlässt.
	if err := repo.SaveUsage(ctx, &domain.InputTaxUsage{
		CorrectionID: correction.ID, FiscalYear: 2028,
		Permille: 600, Confirmed: true, Amount: -60_800,
		Reason: "Vermerk fehlt",
	}); err != nil {
		t.Fatalf("Vermerk zurücksetzen: %v", err)
	}
	view, err := svc.Year(ctx, 2028)
	if err != nil {
		t.Fatalf("Verzeichnis: %v", err)
	}
	if view.Rows[0].Booked {
		t.Fatal("der Testaufbau verlangt einen Eintrag ohne Vermerk")
	}

	if _, err := svc.BookYear(ctx, 2028); err == nil {
		t.Fatal("die Berichtigung darf nicht ein zweites Mal gebucht werden")
	} else if !strings.Contains(err.Error(), "bereits gebucht") {
		t.Errorf("die Meldung muss die stehende Buchung nennen: %v", err)
	}
}

// Die Runde-4-Nachbesserungen: der ursprüngliche Verwendungsanteil von null.

// Ein ursprünglicher Anteil von 0 ‰ bleibt null.
//
// Er ist der klassische Fall der Berichtigung nach oben: angeschafft ohne
// Vorsteuerabzug, weil das Wirtschaftsgut für steuerfreie Umsätze verwendet
// wurde, später für abzugsberechtigende Umsätze eingesetzt (§ 15a Abs. 1 UStG).
// Solange die Angabe eine Zahl mit `omitempty` war, wurde daraus stillschweigend
// volle Verwendung — und damit aus einer Berichtigung zugunsten des Anwenders
// eine zu seinen Lasten.
func TestOriginalPermilleOfZeroSurvives(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.inputTax(t)

	none := 0
	correction, err := svc.Register(ctx, RegisterInputTaxRequest{
		Label: "Halle AN-2026-0002", Account: "0240",
		AcquisitionDate:  "2026-01-15",
		NetAmount:        20_000_000,
		InputTaxAmount:   3_800_000,
		OriginalPermille: &none,
		Note:             "Anschaffung für steuerfreie Vermietung, ohne Vorsteuerabzug",
	})
	if err != nil {
		t.Fatalf("Aufnahme ins Verzeichnis: %v", err)
	}
	if correction.OriginalPermille != 0 {
		t.Fatalf("ursprünglicher Anteil %d ‰ — erwartet 0 ‰",
			correction.OriginalPermille)
	}

	// Ohne Angabe bleibt es beim Regelfall: volle Verwendung.
	silent, err := svc.Register(ctx, RegisterInputTaxRequest{
		Label: "Pkw AN-2026-0003", Account: "0520",
		AcquisitionDate: "2026-02-01",
		NetAmount:       4_000_000, InputTaxAmount: 760_000,
	})
	if err != nil {
		t.Fatalf("Aufnahme ohne Anteil: %v", err)
	}
	if silent.OriginalPermille != 1000 {
		t.Errorf("ohne Angabe %d ‰ — erwartet 1000 ‰", silent.OriginalPermille)
	}

	// Und die Erhöhung wird gerechnet: von 0 auf 1000 ‰ im Berichtigungszeitraum
	// eines Grundstücks sind 3.800.000 / 10 = 380.000 Cent zugunsten des
	// Anwenders.
	view, err := svc.SaveUsage(ctx, SaveInputTaxUsageRequest{
		CorrectionID: correction.ID, FiscalYear: 2028, Permille: 1000,
		Reason: "Halle ab 2028 vollständig für steuerpflichtige Umsätze genutzt",
	})
	if err != nil {
		t.Fatalf("Verwendungsanteil speichern: %v", err)
	}
	var found bool
	for _, row := range view.Rows {
		if row.Correction.ID != correction.ID {
			continue
		}
		found = true
		if row.Assessment.Amount != domain.Cents(380_000) {
			t.Errorf("Berichtigung %s — erwartet 3.800,00 € zugunsten des Anwenders",
				row.Assessment.Amount)
		}
	}
	if !found {
		t.Fatalf("die Halle steht nicht im Verzeichnis des Jahres (%d Zeilen)", len(view.Rows))
	}
}

// Der Berichtigungszeitraum des § 15a UStG läuft ab der erstmaligen Verwendung
// und nicht ab dem Jahresbeginn. Bei einem Zugang mitten im Jahr reicht er
// deshalb ins sechste Kalenderjahr, und Anfangs- wie Schlussjahr gehen nur mit
// ihren Monaten ein (§ 45 UStDV).
func TestInputTaxCorrectionPeriodRunsFromFirstUse(t *testing.T) {
	// Pkw, 47.600 € brutto, 7.600 € Vorsteuer, Zugang zur Jahresmitte. Der
	// Zeitraum endet am 30.06.2031; auf 2031 entfallen sechs Monate.
	t.Run("Schlussjahr anteilig", func(t *testing.T) {
		env := newTestEnv(t)
		ctx := context.Background()
		svc := env.inputTax(t)

		correction, err := svc.Register(ctx, RegisterInputTaxRequest{
			Label: "Pkw AN-2026-0002", Account: "0520",
			AcquisitionDate: "2026-07-01",
			NetAmount:       4_000_000, InputTaxAmount: 760_000,
		})
		if err != nil {
			t.Fatalf("Aufnahme ins Verzeichnis: %v", err)
		}
		if correction.PeriodEnd != "2031-06-30" {
			t.Errorf("Ende des Zeitraums %s — erwartet 2031-06-30", correction.PeriodEnd)
		}
		if correction.LastFiscalYear != 2031 {
			t.Errorf("letztes Berichtigungsjahr %d — erwartet 2031: der Zeitraum reicht ins "+
				"sechste Kalenderjahr", correction.LastFiscalYear)
		}

		view, err := svc.SaveUsage(ctx, SaveInputTaxUsageRequest{
			CorrectionID: correction.ID, FiscalYear: 2031, Permille: 600,
			Reason: "Pkw ab 2031 zu 40 % für steuerfreie Vermietung eingesetzt",
		})
		if err != nil {
			t.Fatalf("Verwendungsanteil 2031: %v", err)
		}
		if len(view.Rows) != 1 {
			t.Fatalf("%d Zeilen im Verzeichnis — erwartet eine", len(view.Rows))
		}
		if !view.Rows[0].InPeriod {
			t.Fatalf("2031 liegt im Zeitraum: %s", view.Rows[0].Assessment.Reason)
		}
		if view.Rows[0].MonthsInYear != 6 {
			t.Errorf("%d Monate in 2031 — erwartet sechs (Januar bis Juni)",
				view.Rows[0].MonthsInYear)
		}
		// 7.600 / 5 × 6/12 × 40 % = 304,00 €.
		if view.TotalAmount != -30_400 {
			t.Errorf("Berichtigung 2031: %s € — erwartet -304,00 €", view.TotalAmount)
		}

		// Erst das Jahr danach liegt außerhalb, und der Grund nennt den Zeitraum
		// mit seinen Daten.
		after, err := svc.Year(ctx, 2032)
		if err != nil {
			t.Fatalf("Verzeichnis 2032: %v", err)
		}
		if after.Rows[0].InPeriod {
			t.Error("2032 liegt nach dem Berichtigungszeitraum")
		}
		if !strings.Contains(after.Rows[0].Assessment.Reason, "30.06.2031") {
			t.Errorf("der Grund muss das Ende des Zeitraums nennen: %s",
				after.Rows[0].Assessment.Reason)
		}
	})

	// Zugang am 20.12.: der Zeitraum endet am 19.12.2031, und weil das nach dem
	// 15. liegt, zählt der Dezember 2031 nach § 45 UStDV voll. 2031 geht
	// deshalb mit zwölf Monaten ein — und war vorher gar nicht berichtigbar.
	//
	// Dieselbe Rundung am Anfang: die Verwendung beginnt nach dem 15., also
	// bleibt der Dezember 2026 unberücksichtigt. Sonst trüge der Zeitraum 61
	// Monate, und die Summe der Jahresanteile überstiege die Vorsteuer.
	t.Run("Zugang im Dezember", func(t *testing.T) {
		env := newTestEnv(t)
		ctx := context.Background()
		svc := env.inputTax(t)

		correction, err := svc.Register(ctx, RegisterInputTaxRequest{
			Label: "Pkw AN-2026-0003", Account: "0520",
			AcquisitionDate: "2026-12-20",
			NetAmount:       4_000_000, InputTaxAmount: 760_000,
		})
		if err != nil {
			t.Fatalf("Aufnahme ins Verzeichnis: %v", err)
		}
		if correction.PeriodEnd != "2031-12-31" {
			t.Errorf("Ende des Zeitraums %s — erwartet 2031-12-31", correction.PeriodEnd)
		}

		view, err := svc.SaveUsage(ctx, SaveInputTaxUsageRequest{
			CorrectionID: correction.ID, FiscalYear: 2031, Permille: 600,
		})
		if err != nil {
			t.Fatalf("Verwendungsanteil 2031: %v", err)
		}
		if view.Rows[0].MonthsInYear != 12 {
			t.Errorf("%d Monate in 2031 — erwartet zwölf", view.Rows[0].MonthsInYear)
		}
		if view.TotalAmount != -60_800 {
			t.Errorf("Berichtigung 2031: %s € — erwartet -608,00 €", view.TotalAmount)
		}

		// Das Zugangsjahr hat keinen Monat: der angefangene Dezember 2026
		// bleibt nach § 45 UStDV außer Betracht, weil die Verwendung nach dem
		// 15. begann.
		first, err := svc.Year(ctx, 2026)
		if err != nil {
			t.Fatalf("Verzeichnis 2026: %v", err)
		}
		if first.Rows[0].MonthsInYear != 0 {
			t.Errorf("%d Monate in 2026 — erwartet keinen: der Zeitraum beginnt am 20.12. und der "+
				"angefangene Monat bleibt unberücksichtigt", first.Rows[0].MonthsInYear)
		}
	})

	// Weicht die Verwendung schon im Jahr der erstmaligen Verwendung von der
	// beim Abzug beabsichtigten ab, ist auch das ein Fall des § 15a UStG
	// (UStAE 15a.2 Abs. 2) — anteilig für die Monate der Verwendung.
	t.Run("Änderung im Zugangsjahr", func(t *testing.T) {
		env := newTestEnv(t)
		ctx := context.Background()
		svc := env.inputTax(t)

		correction, err := svc.Register(ctx, RegisterInputTaxRequest{
			Label: "Pkw AN-2026-0004", Account: "0520",
			AcquisitionDate: "2026-07-01",
			NetAmount:       4_000_000, InputTaxAmount: 760_000,
		})
		if err != nil {
			t.Fatalf("Aufnahme ins Verzeichnis: %v", err)
		}

		view, err := svc.SaveUsage(ctx, SaveInputTaxUsageRequest{
			CorrectionID: correction.ID, FiscalYear: 2026, Permille: 600,
			Reason: "Pkw von Beginn an zu 40 % für steuerfreie Vermietung eingesetzt",
		})
		if err != nil {
			t.Fatalf("Verwendungsanteil im Zugangsjahr: %v", err)
		}
		if !view.Rows[0].InPeriod {
			t.Fatalf("das Zugangsjahr gehört in den Zeitraum: %s", view.Rows[0].Assessment.Reason)
		}
		if view.TotalAmount != -30_400 {
			t.Errorf("Berichtigung 2026: %s € — erwartet -304,00 € (sechs Monate)", view.TotalAmount)
		}

		booked, err := svc.BookYear(ctx, 2026)
		if err != nil {
			t.Fatalf("Buchung im Zugangsjahr: %v", err)
		}
		if !booked.Rows[0].Booked {
			t.Error("die Berichtigung des Zugangsjahres muss buchbar sein")
		}
	})
}

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
