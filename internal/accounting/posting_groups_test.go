// SPDX-FileCopyrightText: 2026 Maximilian Pfennig
// SPDX-License-Identifier: EUPL-1.2

package accounting

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// chartForTest builds the resolver from the shipped DATEV SKR04 2026 catalog.
func chartForTest(t *testing.T) *Chart {
	t.Helper()
	accounts := DefaultSKR04Accounts()
	if len(accounts) < 1000 {
		t.Fatalf("SKR04-Katalog wurde nicht geladen, nur %d Konten", len(accounts))
	}
	return NewChart(accounts)
}

// TestPostingGroupAccountsExistInSKR04 is the guard that keeps invented or
// SKR03 account numbers out of the mapping. Every account any group can resolve
// to — for every Steuerfall and every rate — has to exist in the DATEV catalog
// and be bookable.
func TestPostingGroupAccountsExistInSKR04(t *testing.T) {
	chart := chartForTest(t)

	treatments := []domain.TaxTreatment{
		domain.TaxTreatmentDomestic,
		domain.TaxTreatmentReverseCharge,
		domain.TaxTreatmentIntraCommunityAcquisition,
		domain.TaxTreatmentIntraCommunitySupply,
		domain.TaxTreatmentReverseChargeSupply,
		domain.TaxTreatmentExport,
		domain.TaxTreatmentExempt,
		domain.TaxTreatmentNotTaxable,
	}

	for _, group := range PostingGroups("") {
		for _, treatment := range treatments {
			for _, rate := range domain.ValidTaxRates() {
				account := group.ResolveAccount(treatment, rate)

				if err := chart.EnsurePostable(account); err != nil {
					t.Errorf("Gruppe %q (Steuerfall %s, Satz %s) → Konto %s: %v",
						group.Key, treatment, rate.Label(), account, err)
					continue
				}

				resolved, _ := chart.Lookup(account)
				wantType := domain.AccountTypeExpense
				if group.Direction == domain.DirectionOutgoing {
					wantType = domain.AccountTypeRevenue
				}
				if resolved.Type != wantType {
					t.Errorf("Gruppe %q (Steuerfall %s) → Konto %s %q ist %s, erwartet %s",
						group.Key, treatment, account, resolved.Name, resolved.Type, wantType)
				}
			}
		}
	}
}

// TestStandardAccountsMatchDATEV pins the account constants against the shipped
// catalog by name. A wrong constant would otherwise stay invisible: 1600 exists
// in both SKR03 and SKR04, it just means something entirely different.
func TestStandardAccountsMatchDATEV(t *testing.T) {
	chart := chartForTest(t)

	want := map[string]string{
		domain.AccountForderungenLuL:            "Forderungen aus Lieferungen und Leistungen",
		domain.AccountKasse:                     "Kasse",
		domain.AccountBank:                      "Bank",
		domain.AccountGeldtransit:               "Geldtransit",
		domain.AccountVerbindlichkeitenLuL:      "Verbindlichkeiten aus Lieferungen und Leistungen",
		domain.AccountVorsteuer19:               "Abziehbare Vorsteuer 19 %",
		domain.AccountVorsteuer7:                "Abziehbare Vorsteuer 7 %",
		domain.AccountVorsteuer13b19:            "Abziehbare Vorsteuer nach § 13b UStG 19 %",
		domain.AccountVorsteuerIG19:             "Abziehbare Vorsteuer aus innergemeinschaftlichem Erwerb 19 %",
		domain.AccountUmsatzsteuer19:            "Umsatzsteuer 19 %",
		domain.AccountUmsatzsteuer7:             "Umsatzsteuer 7 %",
		domain.AccountUmsatzsteuer13b19:         "Umsatzsteuer nach § 13b UStG 19 %",
		domain.AccountUmsatzsteuerIG19:          "Umsatzsteuer aus innergemeinschaftlichem Erwerb 19 %",
		domain.AccountAktiveRAP:                 "Aktive Rechnungsabgrenzung",
		domain.AccountPassiveRAP:                "Passive Rechnungsabgrenzung",
		domain.AccountGezeichnetesKapital:       "Gezeichnetes Kapital",
		domain.AccountSaldenvortraegeSachkonten: "Saldenvorträge, Sachkonten",
		domain.AccountNebenkostenGeld:           "Nebenkosten des Geldverkehrs",
	}

	for number, name := range want {
		acc, ok := chart.Lookup(number)
		if !ok {
			t.Errorf("Konto %s fehlt im SKR04-Katalog", number)
			continue
		}
		if acc.Name != name {
			t.Errorf("Konto %s heißt im DATEV-Katalog %q, die Konstante behauptet %q", number, acc.Name, name)
		}
	}
}

// TestSKR03NumbersAreNotUsed keeps the numbers from the original concept draft
// out of the code. They are either absent from SKR04 or mean something else.
func TestSKR03NumbersAreNotUsed(t *testing.T) {
	chart := chartForTest(t)

	// number → what SKR04 actually says, empty if the account does not exist.
	skr03 := map[string]string{
		"1576": "",                                             // dort: abziehbare Vorsteuer 19 % — in SKR04 nicht vorhanden
		"4930": "Erträge aus der Auflösung von Rückstellungen", // dort: Bürobedarf
		"0820": "Beteiligungen",                                // dort: Fahrzeuge
		"2120": "Privatentnahmen allgemein",                    // dort: Zinsaufwand
		"0630": "Betriebsausstattung",                          // dort: Darlehen
	}

	for number, expected := range skr03 {
		acc, ok := chart.Lookup(number)
		if expected == "" {
			if ok {
				t.Errorf("Konto %s sollte im SKR04 nicht existieren, wurde aber als %q gefunden", number, acc.Name)
			}
			continue
		}
		if !ok || acc.Name != expected {
			t.Errorf("Konto %s: erwartet %q, gefunden %q", number, expected, acc.Name)
		}
	}
	skr03["8400"] = "" // Klasse 8 – im SKR04 kein Erlöskonto

	// 8400 is the SKR03 revenue account. In SKR04 it falls into the reserved
	// class 8, and booking to it has to be refused with an explanation rather
	// than silently accepted.
	err := chart.EnsurePostable("8400")
	if err == nil {
		t.Error("auf Konto 8400 (SKR03-Erlöskonto, im SKR04 Klasse 8) darf nicht gebucht werden")
	} else if !strings.Contains(err.Error(), "4400") {
		t.Errorf("die Fehlermeldung zu 8400 sollte auf das richtige SKR04-Erlöskonto verweisen, lautet aber: %v", err)
	}

	// None of them may appear as a target of the mapping.
	for _, group := range PostingGroups("") {
		for _, rate := range domain.ValidTaxRates() {
			account := group.ResolveAccount(domain.TaxTreatmentDomestic, rate)
			if _, isSKR03 := skr03[account]; isSKR03 {
				t.Errorf("Gruppe %q bucht auf %s – das ist eine SKR03-Nummer", group.Key, account)
			}
		}
	}
}

// TestRangeAccountsAreNotPostable makes sure the catalog's grouping notation is
// not mistaken for an account. A company books to 4400, never to "4400-4409".
func TestRangeAccountsAreNotPostable(t *testing.T) {
	chart := chartForTest(t)

	if err := chart.EnsurePostable("4400-4409"); err == nil {
		t.Error("auf das Bereichskonto 4400-4409 darf nicht gebucht werden")
	}
	if err := chart.EnsurePostable("4400"); err != nil {
		t.Errorf("Konto 4400 liegt im Bereich 4400-4409 und muss bebuchbar sein: %v", err)
	}
	if err := chart.EnsurePostable("4409"); err != nil {
		t.Errorf("Konto 4409 liegt im Bereich 4400-4409 und muss bebuchbar sein: %v", err)
	}

	// A reserved range stays closed.
	if err := chart.EnsurePostable("0055"); err == nil {
		t.Error("Konto 0055 liegt in einem reservierten Bereich und darf nicht bebuchbar sein")
	}
}

func TestChartLookupInheritsRangeClassification(t *testing.T) {
	chart := chartForTest(t)

	acc, ok := chart.Lookup("4405")
	if !ok {
		t.Fatal("Konto 4405 wurde nicht gefunden")
	}
	if acc.Number != "4405" {
		t.Errorf("die Kontonummer muss erhalten bleiben, ist aber %q", acc.Number)
	}
	if acc.Type != domain.AccountTypeRevenue {
		t.Errorf("4405 muss ein Ertragskonto sein, ist aber %s", acc.Type)
	}
	if acc.IsRange {
		t.Error("ein aus einem Bereich abgeleitetes Konto darf selbst kein Bereichskonto sein")
	}
}
