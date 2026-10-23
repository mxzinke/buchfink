package service

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestEntryEffectsRunInOrderBeforeAudit(t *testing.T) {
	e := newTestEnv(t)
	ctx := context.Background()
	audit := repository.NewAuditRepository(e.db)
	var calls []string
	var original *domain.JournalEntry
	var auditCount int64
	checkWritten := func(ctx context.Context, entry *domain.JournalEntry) {
		t.Helper()
		if !repository.InTx(ctx) {
			t.Fatal("Effekt läuft außerhalb der Journaltransaktion")
		}
		stored, err := e.journalRepo.FindByID(ctx, entry.ID)
		if err != nil || stored.EntryHash != entry.EntryHash || stored.EntryNumber == "" {
			t.Fatalf("Effekt sieht keine geschriebene Buchung: %+v, %v", stored, err)
		}
		if got, err := audit.Count(ctx); err != nil || got != auditCount {
			t.Fatalf("Journalprotokoll wurde vor den Effekten geschrieben: %d, %v; erwartet %d", got, err, auditCount)
		}
	}
	for _, name := range []string{"erster", "zweiter"} {
		e.journal.AddEntryEffects(domain.EntryEffect{
			Name: name,
			BeforeAppend: func(ctx context.Context, entry *domain.JournalEntry) error {
				if !repository.InTx(ctx) || entry.ID != 0 {
					t.Fatal("Vorprüfung muss vor Append in der Transaktion laufen")
				}
				calls = append(calls, "before:"+name)
				return nil
			},
			AfterPost: func(ctx context.Context, entry *domain.JournalEntry) error {
				checkWritten(ctx, entry)
				calls = append(calls, "post:"+name)
				return nil
			},
			AfterReverse: func(ctx context.Context, from, reversal *domain.JournalEntry) error {
				checkWritten(ctx, reversal)
				if from == nil || from.ID != original.ID || from.EntryHash != original.EntryHash ||
					reversal.ReversalOfID == nil || *reversal.ReversalOfID != from.ID {
					t.Fatalf("Original und Umkehr fehlen: %+v, %+v", from, reversal)
				}
				calls = append(calls, "reverse:"+name)
				return nil
			},
		})
	}
	if err := e.journal.ValidatePostable(ctx, simpleEntry("6815", "1800", 10000)); err != nil {
		t.Fatal(err)
	}
	invalid := simpleEntry("9999999", "1800", 10000)
	if _, err := e.journal.Post(ctx, invalid); err == nil {
		t.Fatal("ungültige Buchung angenommen")
	}
	if len(calls) != 0 {
		t.Fatalf("Effekte liefen bei Vorschau oder vor den Regeln: %v", calls)
	}
	var err error
	auditCount, err = audit.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	original, err = e.journal.Post(ctx, simpleEntry("6815", "1800", 10000))
	if err != nil {
		t.Fatal(err)
	}
	auditCount, err = audit.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.journal.Reverse(ctx, original.ID, "Korrektur"); err != nil {
		t.Fatal(err)
	}
	want := []string{"before:erster", "before:zweiter", "post:erster", "post:zweiter", "reverse:erster", "reverse:zweiter"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("Effektreihenfolge: %v; erwartet %v", calls, want)
	}
}

func TestEntryEffectFailureRollsBackBankJournalAndAudit(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(map[bool]string{false: "Buchung", true: "Storno"}[reverse], func(t *testing.T) {
			e := newTestEnv(t)
			ctx := context.Background()
			e.paymentsWithBank(t)
			id := e.bankLine(t, "2026-03-01", -1500, "Hausbank", "Kontoführung")
			entry := simpleEntry("6855", "1800", 1500)
			entry.BankTxID = &id
			if reverse {
				if _, err := e.journal.Post(ctx, entry); err != nil {
					t.Fatal(err)
				}
			}
			audit := repository.NewAuditRepository(e.db)
			before, err := audit.Count(ctx)
			if err != nil {
				t.Fatal(err)
			}
			failure := errors.New("Folgeschritt fehlgeschlagen")
			fail := func(ctx context.Context, _ *domain.JournalEntry) error {
				want := domain.MatchStatusMatched
				if reverse {
					want = domain.MatchStatusUnmatched
				}
				tx, err := repository.NewBankRepository(e.db).FindByID(ctx, id)
				if err != nil || tx.MatchStatus != want {
					t.Fatalf("Bankeffekt fehlt innerhalb der Transaktion: %+v, %v", tx, err)
				}
				return failure
			}
			e.journal.AddEntryEffects(domain.EntryEffect{
				Name: "fehler", AfterPost: fail,
				AfterReverse: func(ctx context.Context, _, reversal *domain.JournalEntry) error { return fail(ctx, reversal) },
			}, domain.EntryEffect{
				Name: "danach", AfterPost: func(context.Context, *domain.JournalEntry) error {
					t.Fatal("Effekt nach dem ersten Fehler ausgeführt")
					return nil
				}, AfterReverse: func(context.Context, *domain.JournalEntry, *domain.JournalEntry) error {
					t.Fatal("Stornoeffekt nach dem ersten Fehler ausgeführt")
					return nil
				},
			})
			if reverse {
				_, err = e.journal.Reverse(ctx, entry.ID, "Korrektur")
			} else {
				_, err = e.journal.Post(ctx, entry)
			}
			if !errors.Is(err, failure) || err.Error() != failure.Error() {
				t.Fatalf("Effektfehler verändert: %v", err)
			}
			wantCount, wantStatus := 0, domain.MatchStatusUnmatched
			if reverse {
				wantCount, wantStatus = 1, domain.MatchStatusMatched
			}
			if got := e.entryCount(t); got != wantCount {
				t.Fatalf("Journal nach Fehler: %d, erwartet %d", got, wantCount)
			}
			e.assertMatch(t, id, wantStatus)
			if after, err := audit.Count(ctx); err != nil || after != before {
				t.Fatalf("Protokoll nach Fehler: %d, %v; erwartet %d", after, err, before)
			}
		})
	}
}

func TestJournalAuditFailureRollsBackBankEffect(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(map[bool]string{false: "Buchung", true: "Storno"}[reverse], func(t *testing.T) {
			e := newTestEnv(t)
			ctx := context.Background()
			e.paymentsWithBank(t)
			id := e.bankLine(t, "2026-03-01", -1500, "Hausbank", "Kontoführung")
			entry := simpleEntry("6855", "1800", 1500)
			entry.BankTxID = &id
			if reverse {
				if _, err := e.journal.Post(ctx, entry); err != nil {
					t.Fatal(err)
				}
			}
			e.journal.auditRepo = failingAudit{AuditRepository: repository.NewAuditRepository(e.db), trigger: "JOURNAL"}
			var err error
			wantCount, wantStatus := 0, domain.MatchStatusUnmatched
			if reverse {
				_, err = e.journal.Reverse(ctx, entry.ID, "Korrektur")
				wantCount, wantStatus = 1, domain.MatchStatusMatched
			} else {
				_, err = e.journal.Post(ctx, entry)
			}
			if err == nil || e.entryCount(t) != wantCount {
				t.Fatalf("Protokollfehler muss Buchung zurückrollen: %v", err)
			}
			e.assertMatch(t, id, wantStatus)
		})
	}
}

func TestBankEffectPreservesClaimErrorOrder(t *testing.T) {
	for _, name := range []string{"Journalregel", "ignoriert", "doppelt", "Betrag"} {
		t.Run(name, func(t *testing.T) {
			e := newTestEnv(t)
			ctx := context.Background()
			e.paymentsWithBank(t)
			id := e.bankLine(t, "2026-03-01", -1500, "Hausbank", "Kontoführung")
			entry := simpleEntry("6855", domain.AccountBank, 1400)
			entry.BankTxID = &id
			want := fmt.Sprintf("die Buchung bewegt auf dem Bankkonto 1800 -14,00 €, der Bankumsatz %d lautet über -15,00 €. Der Betrag kommt aus dem Kontoauszug und wird nicht angepasst", id)
			switch name {
			case "Journalregel":
				entry.Description = ""
				want = "Buchungstext fehlt"
			case "ignoriert":
				if err := repository.NewBankRepository(e.db).SetMatchStatus(ctx, id, domain.MatchStatusIgnored); err != nil {
					t.Fatal(err)
				}
				want = fmt.Sprintf("Bankumsatz %d ist als nicht zu buchen markiert; eine Buchung dazu stünde neben dieser Entscheidung", id)
			case "doppelt":
				first := simpleEntry("6855", domain.AccountBank, 1500)
				first.BankTxID = &id
				if _, err := e.journal.Post(ctx, first); err != nil {
					t.Fatal(err)
				}
				want = fmt.Sprintf("Bankumsatz %d vom 01.03.2026 über -15,00 € ist bereits mit Buchung %s gebucht. Ein Umsatz wird genau einmal gebucht; eine falsche Buchung wird storniert und neu gebucht", id, first.EntryNumber)
			}
			if name != "Journalregel" {
				if err := e.journal.ValidatePostable(ctx, entry); err != nil {
					t.Fatalf("Die Vorschau beansprucht keinen Bankumsatz: %v", err)
				}
			}
			if _, err := e.journal.Post(ctx, entry); err == nil || err.Error() != want {
				t.Fatalf("Fehler: %v; erwartet %q", err, want)
			}
		})
	}
}

func TestBankEffectAppliesToEveryEntrySource(t *testing.T) {
	for _, source := range []domain.EntrySource{
		domain.EntrySourceManual, domain.EntrySourceReceipt, domain.EntrySourceInvoice,
		domain.EntrySourceAdvance, domain.EntrySourcePayment, domain.EntrySourceClosing,
		domain.EntrySourceOpening, domain.EntrySourceDepreciation,
	} {
		t.Run(string(source), func(t *testing.T) {
			e := newTestEnv(t)
			ctx := context.Background()
			e.paymentsWithBank(t)
			id := e.bankLine(t, "2026-03-01", -1500, "Hausbank", "Kontoführung")
			entry := simpleEntry("6855", domain.AccountBank, 1500)
			entry.Source, entry.BankTxID = source, &id
			if _, err := e.journal.Post(ctx, entry); err != nil {
				t.Fatal(err)
			}
			e.assertMatch(t, id, domain.MatchStatusMatched)
			if _, err := e.journal.Reverse(ctx, entry.ID, "Korrektur"); err != nil {
				t.Fatal(err)
			}
			e.assertMatch(t, id, domain.MatchStatusUnmatched)
		})
	}
}

func TestEntryEffectRegistrationRejectsAmbiguity(t *testing.T) {
	noop := func(context.Context, *domain.JournalEntry) error { return nil }
	for _, tc := range []struct {
		name    string
		effects []domain.EntryEffect
	}{
		{"ohne Namen", []domain.EntryEffect{{AfterPost: noop}}},
		{"ohne Phase", []domain.EntryEffect{{Name: "leer"}}},
		{"doppelter Name", []domain.EntryEffect{{Name: "gleich", AfterPost: noop}, {Name: "gleich", BeforeAppend: noop}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("mehrdeutige oder leere Registrierung angenommen")
				}
			}()
			new(JournalService).AddEntryEffects(tc.effects...)
		})
	}
}
