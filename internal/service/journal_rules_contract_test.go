package service

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die Tabelle hält Meldungen und ihre Reihenfolge für Vorprüfung und Schreiben
// fest. Generalumkehr prüft die internen Pfade mit postMode: öffentlich darf sie
// nur Reverse erzeugen; Post und ValidatePostable müssen sie zurückweisen.
func TestJournalRuleContract(t *testing.T) {
	const (
		accrual       = "Für diesen Mandanten ist die Istversteuerung hinterlegt. Buchfink unterstützt derzeit nur die Sollversteuerung (§ 16 Abs. 1 Satz 1 UStG): bei Istversteuerung entsteht die Steuer erst mit der Vereinnahmung des Entgelts (§ 13 Abs. 1 Nr. 1 Buchst. b UStG), und die Buchungen sähen anders aus. Stornos bestehender Buchungen bleiben möglich"
		entertainment = "zu einer Bewirtung gehören Ort, Tag, Teilnehmer und Anlass (§ 4 Abs. 5 Satz 1 Nr. 2 EStG). Es fehlt: Anlass"
		unknown       = "Zeile 1: Konto 9999999 ist im SKR04 nicht vorhanden"
		taxAccount    = "Zeile 1: Konto 1406 ist ein Steuerkonto und darf nur über die Steuerautomatik bebucht werden"
		collective    = "Zeile 1: Konto 1200 ist das Sammelkonto für die Bilanz und wird nicht direkt bebucht. Der offene Posten gehört auf das Personenkonto des Kunden – die Bilanzposition verdichtet sich daraus"
		unlawful      = "Zeile 1 weist 19,00 € Umsatzsteuer aus, der Steuerfall \"not_taxable\" lässt aber keine entstehen. Ein solcher Ausweis wird nach § 14c UStG trotzdem geschuldet – er gehört auf den Steuerschlüssel UST14C (Kennziffer 69), oder der Umsatz ist steuerpflichtig im Inland"
		unknownKey    = "Zeile 1: unbekannter Steuerschlüssel \"UNBEKANNT\"; er käme in keiner Kennziffer der Voranmeldung an"
		noTreatment   = "der Buchung fehlt der Steuerfall. Er sagt, warum Steuer entsteht oder nicht — beim Inlandsumsatz, beim innergemeinschaftlichen Erwerb, bei § 13b UStG oder bei einem steuerfreien Vorgang. Ohne ihn läuft die Buchung an der Voranmeldung vorbei"
		noTaxLine     = "die Buchung hat den Steuerfall „steuerpflichtiger Inlandsumsatz\", aber keine Steuerzeile. Ein steuerpflichtiger Umsatz löst Umsatzsteuer oder Vorsteuer aus; fällt keine an, ist der Steuerfall ein anderer — steuerfrei, nicht steuerbar oder Nullsteuersatz nach § 12 Abs. 3 UStG"
		noManualBase  = "Zeile 1 hat den Steuerschlüssel VST15A, aber keine Bemessungsgrundlage. Die Voranmeldung meldet den Umsatz und die Steuer daraus — ohne die Bemessungsgrundlage fehlte der Umsatz"
		adopted       = "Der Jahresabschluss 2026 ist am 2026-02-01 festgestellt. Eine Buchung zum 2026-03-01 ist deshalb nicht mehr möglich. Muss der Abschluss geändert werden, setzen Sie die Feststellung unter Abschlüsse › Jahr mit Angabe des Grundes zurück"
		committed     = "Der Zeitraum bis 2026-03-31 ist festgeschrieben. Eine Buchung zum 2026-03-01 ist nicht mehr möglich – bitte über eine Stornierung korrigieren"
		header        = "Beleg REGEL-01 hat kein Belegdatum. Ohne es lässt sich nicht beurteilen, ob er zeitgerecht erfasst wurde (§ 146 Abs. 1 AO)"
	)
	type change func(*testing.T, *testEnv, *domain.JournalEntry)
	compose := func(changes ...change) change {
		return func(t *testing.T, env *testEnv, e *domain.JournalEntry) {
			for _, apply := range changes {
				apply(t, env, e)
			}
		}
	}
	cashTax := func(t *testing.T, env *testEnv, _ *domain.JournalEntry) {
		env.setSettings(t, func(c *domain.CompanySettings) { c.TaxationType = "IST" })
	}
	missingText := func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) { e.Description = "" }
	badEntertainment := func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) {
		e.Entertainment = entertainmentRecord()
		e.Entertainment.Occasion = ""
	}
	account := func(number string) change {
		return func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) { e.Lines[0].Account = number }
	}
	source := func(s domain.EntrySource, reference string) change {
		return func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) { e.Source, e.DocumentNumber = s, reference }
	}
	tax := func(key, number string) change {
		return func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) {
			e.Lines[0].Account, e.Lines[0].TaxKey, e.Lines[0].TaxBase = number, key, 10000
		}
	}
	badKey := tax("UNBEKANNT", "1406")
	badDisclosure := tax("UST19", "3806")
	missingTreatment := func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) { e.TaxTreatment = "" }
	lock := func(t *testing.T, env *testEnv, _ *domain.JournalEntry) { env.commitUntil(t, "2026-03-31") }
	adopt := func(t *testing.T, env *testEnv, _ *domain.JournalEntry) {
		years := repository.NewFiscalYearRepository(env.db)
		fy, err := years.FindByYear(context.Background(), 2026)
		if err != nil {
			t.Fatal(err)
		}
		if fy == nil {
			fy = &domain.FiscalYear{Year: 2026, StartDate: "2026-01-01", EndDate: "2026-12-31"}
		}
		fy.Status, fy.AdoptedOn = domain.FiscalYearAdopted, "2026-02-01"
		if err := years.Save(context.Background(), fy); err != nil {
			t.Fatal(err)
		}
	}
	receipt := func(complete bool, status domain.ReceiptStatus) change {
		return func(t *testing.T, env *testEnv, e *domain.JournalEntry) {
			r := &domain.Receipt{ReceiptNumber: "REGEL-01", FiscalYear: 2026,
				Direction: domain.DirectionIncoming, Kind: domain.ReceiptKindInvoice,
				Status: status, IssuerName: "Lieferant", GrossAmount: 1900}
			if complete {
				r.DocumentDate = "2026-03-01"
			}
			if err := env.receiptRepo.Create(context.Background(), r, accounting.ReceiptHash); err != nil {
				t.Fatal(err)
			}
			e.ReceiptID = &r.ID
		}
	}
	badHeader := receipt(false, domain.ReceiptStatusFiled)
	partner := func(foreign bool) change {
		return func(t *testing.T, env *testEnv, e *domain.JournalEntry) {
			own := env.customer(t, "Kunde Alpha", "DE", "")
			e.Lines[0].Account, e.Lines[0].ContactID = own.LedgerAccount, &own.ID
			if foreign {
				other := env.customer(t, "Kunde Beta", "DE", "")
				e.Lines[0].ContactID = &other.ID
			}
		}
	}
	cases := []struct {
		name         string
		change       change
		want         string
		reversalWant string
	}{
		{"gültig/Sollversteuerung", nil, "", ""},
		{"Istversteuerung", cashTax, accrual, ""},
		{"Pflichtfeld", missingText, "Buchungstext fehlt", "Buchungstext fehlt"},
		{"Ausgleich", func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) { e.Lines[1].Amount = 1800 },
			"Buchung ist nicht ausgeglichen: Soll 19,00 €, Haben 18,00 € (Differenz 1,00 €)",
			"Buchung ist nicht ausgeglichen: Soll -19,00 €, Haben -18,00 € (Differenz -1,00 €)"},
		{"Bewirtung/unvollständig", badEntertainment, entertainment, entertainment},
		{"Bewirtung/vollständig", func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) { e.Entertainment = entertainmentRecord() }, "", ""},
		{"Kontenplan/unbekannt", account("9999999"), unknown, unknown},
		{"Kontenplan/reserviert", account("0055"), "Zeile 1: Konto 0055 ist im SKR04 als reserviert gekennzeichnet und darf nicht bebucht werden", "Zeile 1: Konto 0055 ist im SKR04 als reserviert gekennzeichnet und darf nicht bebucht werden"},
		{"Rechtsform/GmbH-Konto", account("2920"), "", ""},
		{"Steuerkonto/ohne-Schlüssel", account("1406"), taxAccount, taxAccount},
		{"Steuerkonto/mit-Schlüssel", tax("VST19", "1406"), "", ""},
		{"Eröffnung/Steuerkonto", compose(account("1406"), source(domain.EntrySourceOpening, "SV 2026")), "", ""},
		{"USTV/Abschluss", compose(account("1406"), source(domain.EntrySourceClosing, "USTV 2026")), "", ""},
		{"USTV/anderes-Jahr", compose(account("1406"), source(domain.EntrySourceClosing, "USTV 2025")), "", ""},
		{"USTV/Handbuchung", compose(account("1406"), source(domain.EntrySourceManual, "USTV 2026")), taxAccount, taxAccount},
		{"USTV/andere-Referenz", compose(account("1406"), source(domain.EntrySourceClosing, "USTV 02026")), taxAccount, taxAccount},
		{"USTV/anderes-Steuerkonto", compose(account("1396"), source(domain.EntrySourceClosing, "USTV 2026")),
			"Zeile 1: Konto 1396 ist ein Steuerkonto und darf nur über die Steuerautomatik bebucht werden", "Zeile 1: Konto 1396 ist ein Steuerkonto und darf nur über die Steuerautomatik bebucht werden"},
		{"Sammelkonto", account("1200"), collective, collective},
		{"Personenkonto/eigener-Kontakt", partner(false), "", ""},
		{"Personenkonto/fremder-Kontakt", partner(true), "Zeile 1: Personenkonto 10000 gehört zu Kunde Alpha, die Buchung verweist aber auf einen anderen Geschäftspartner", "Zeile 1: Personenkonto 10000 gehört zu Kunde Alpha, die Buchung verweist aber auf einen anderen Geschäftspartner"},
		{"Personenkonto/unbekannt", account("10000"), "Zeile 1: Personenkonto 10000 gehört zu keinem angelegten Geschäftspartner", "Zeile 1: Personenkonto 10000 gehört zu keinem angelegten Geschäftspartner"},
		{"Steuerausweis/unzulässig", badDisclosure, unlawful, ""},
		{"Steuerausweis/14c", tax("UST14C", "3851"), "", ""},
		{"Steuerschlüssel/unbekannt", badKey, unknownKey, ""},
		{"Steuerschlüssel/falsches-Konto", tax("VST19", "6815"), "Zeile 1: der Steuerschlüssel VST19 gehört auf Konto 1406, nicht auf 6815", ""},
		{"Steuerschlüssel/ohne-Grundlage", compose(tax("VST19", "1406"), func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) { e.Lines[0].TaxBase = 0 }), "Zeile 1: der Steuerschlüssel VST19 braucht die Bemessungsgrundlage; ohne sie fehlt der Umsatz in der Voranmeldung", ""},
		{"Handbuchung/ohne-Steuerfall", missingTreatment, noTreatment, ""},
		{"Handbuchung/Inland-ohne-Steuerzeile", func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) { e.TaxTreatment = domain.TaxTreatmentDomestic }, noTaxLine, ""},
		{"Handbuchung/Inland-mit-Steuerzeile", compose(tax("VST19", "1406"), func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) { e.TaxTreatment = domain.TaxTreatmentDomestic }), "", ""},
		{"Handbuchung/15a-ohne-Grundlage", compose(tax("VST15A", "1396"), func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) { e.Lines[0].TaxBase = 0 }), noManualBase, ""},
		{"Automatik/ohne-Steuerfall", compose(missingTreatment, source(domain.EntrySourceOpening, "SV 2026")), "", ""},
		{"Feststellung", adopt, adopted, adopted},
		{"Festschreibung", lock, committed, committed},
		{"Festschreibung/späterer-Tag", compose(lock, func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) { e.BookingDate = "2026-04-01" }), "", ""},
		{"Belegkopf/unvollständig", badHeader, header, ""},
		{"Belegkopf/vollständig", receipt(true, domain.ReceiptStatusFiled), "", ""},
		{"Belegkopf/versiegelt", receipt(false, domain.ReceiptStatusSealed), "", ""},
		{"Reihenfolge/Ist-vor-Pflichtfeld", compose(cashTax, missingText), accrual, "Buchungstext fehlt"},
		{"Reihenfolge/Pflichtfeld-vor-Bewirtung", compose(missingText, badEntertainment), "Buchungstext fehlt", "Buchungstext fehlt"},
		{"Reihenfolge/Bewirtung-vor-Konto", compose(badEntertainment, account("9999999")), entertainment, entertainment},
		{"Reihenfolge/Sammelkonto-vor-folgendem-Konto", compose(account("1200"), func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) { e.Lines[1].Account = "9999999" }), collective, collective},
		{"Reihenfolge/Konto-vor-folgendem-Sammelkonto", compose(account("9999999"), func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) { e.Lines[1].Account = "1200" }), unknown, unknown},
		{"Reihenfolge/Konto-vor-Steuerausweis", compose(badDisclosure, func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) { e.Lines[1].Account = "9999999" }), "Zeile 2: Konto 9999999 ist im SKR04 nicht vorhanden", "Zeile 2: Konto 9999999 ist im SKR04 nicht vorhanden"},
		{"Reihenfolge/Steuerausweis-vor-Katalog", compose(badDisclosure, func(_ *testing.T, _ *testEnv, e *domain.JournalEntry) { e.Lines[0].TaxBase = 0 }), unlawful, ""},
		{"Reihenfolge/Katalog-vor-Handbuchung", compose(badKey, missingTreatment), unknownKey, ""},
		{"Reihenfolge/Handbuchung-vor-Feststellung", compose(missingTreatment, adopt), noTreatment, adopted},
		{"Reihenfolge/Feststellung-vor-Festschreibung", compose(adopt, lock), adopted, adopted},
		{"Reihenfolge/Festschreibung-vor-Belegkopf", compose(lock, badHeader), committed, committed},
	}
	for _, form := range []string{domain.LegalFormGmbH, domain.LegalFormUG} {
		want := fmt.Sprintf("Zeile 1: Konto 2100 (Privatentnahmen allgemein) gehört zu Einzelunternehmen und Personengesellschaften; eine %s hat keine Privat- oder Kapitalkonten ihrer Gesellschafter. Entnimmt ein Gesellschafter Geld, ist das eine Forderung gegen ihn (1307 Forderungen gegen GmbH-Gesellschafter); legt er Geld ein oder gibt er ein Darlehen, eine Verbindlichkeit ihm gegenüber (3640 Verbindlichkeiten gegenüber GmbH-Gesellschaftern) oder eine Kapitalrücklage (2920). Fließt ihm ohne angemessene Gegenleistung ein Vorteil zu, kann das eine verdeckte Gewinnausschüttung sein (§ 8 Abs. 3 Satz 2 KStG)", form)
		cases = append(cases, struct {
			name         string
			change       change
			want         string
			reversalWant string
		}{"Rechtsform/Privatkonto/" + form, compose(account("2100"), func(t *testing.T, env *testEnv, _ *domain.JournalEntry) {
			env.setSettings(t, func(c *domain.CompanySettings) { c.LegalForm = form })
		}), want, want})
	}
	for _, tc := range cases {
		for _, reversal := range []bool{false, true} {
			name := tc.name
			if reversal {
				name += "/Generalumkehr"
			}
			t.Run(name, func(t *testing.T) {
				ctx := context.Background()
				env := newTestEnv(t)
				env.journal.SetFiscalYearRepo(repository.NewFiscalYearRepository(env.db))
				env.journal.SetFestschreibungRepo(repository.NewFestschreibungRepository(env.db))
				e := simpleEntry("6815", "1800", 1900)
				mode := postMode{}
				want := tc.want
				if reversal {
					original, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 1900))
					if err != nil {
						t.Fatal(err)
					}
					e.Kind, e.ReversalOfID, mode.reversal = domain.EntryKindReversal, &original.ID, true
					want = tc.reversalWant
				}
				if tc.change != nil {
					tc.change(t, env, e)
				}
				if reversal {
					for i := range e.Lines {
						e.Lines[i].Amount = -e.Lines[i].Amount
						e.Lines[i].TaxBase = -e.Lines[i].TaxBase
					}
				}
				before := *e
				before.Lines = append([]domain.JournalLine(nil), e.Lines...)
				if e.Entertainment != nil {
					d := *e.Entertainment
					before.Entertainment = &d
				}
				var err error
				if reversal {
					err = env.journal.validatePostable(ctx, e, mode)
				} else {
					err = env.journal.ValidatePostable(ctx, e)
				}
				assertJournalRuleError(t, "ValidatePostable", err, want)
				if !reflect.DeepEqual(*e, before) {
					t.Fatal("Vorprüfung verändert die Eingabe")
				}
				entries, err := env.journalRepo.FindAll(ctx, 2026)
				if err != nil {
					t.Fatal(err)
				}
				count := 0
				if reversal {
					count = 1
				}
				if len(entries) != count {
					t.Fatal("Vorprüfung schreibt ins Journal")
				}
				if reversal {
					_, err = env.journal.post(ctx, e, mode)
				} else {
					_, err = env.journal.Post(ctx, e)
				}
				assertJournalRuleError(t, "Post", err, want)
				entries, err = env.journalRepo.FindAll(ctx, 2026)
				if err != nil {
					t.Fatal(err)
				}
				if want == "" {
					count++
				}
				if len(entries) != count {
					t.Fatalf("Journal hat %d statt %d Buchungen", len(entries), count)
				}
			})
		}
	}
}

func assertJournalRuleError(t *testing.T, path string, err error, want string) {
	t.Helper()
	got := ""
	if err != nil {
		got = err.Error()
	}
	if got != want {
		t.Fatalf("%s: Meldung = %q, erwartet %q", path, got, want)
	}
}

func TestJournalRulePublicPathsRefuseReversal(t *testing.T) {
	env := newTestEnv(t)
	e := simpleEntry("6815", "1800", -1900)
	id := uint(1)
	e.Kind, e.ReversalOfID = domain.EntryKindReversal, &id
	const want = "eine Generalumkehr entsteht nur über den Storno einer vorhandenen Buchung, nicht als eigene Buchung"
	assertJournalRuleError(t, "ValidatePostable", env.journal.ValidatePostable(context.Background(), e), want)
	_, err := env.journal.Post(context.Background(), e)
	assertJournalRuleError(t, "Post", err, want)
}
