package policy

import (
	"slices"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// routine ist eine E-Rechnung, die alle Regeln erfüllt.
func routine() ReceiptFacts {
	net := map[domain.TaxRate]domain.Cents{domain.TaxRateStandard: 10000}
	return ReceiptFacts{
		Open: true, Displayable: true, Incoming: true, Invoice: true, Structured: true, Checked: true,
		ProposalContactID: 7, ProposalMatchedBy: "ust_id", ProposalTreatment: domain.TaxTreatmentDomestic, ProposalCurrency: "EUR",
		ProposalNet: net, ProposalGross: 11900,
		ContactID: 7, ConfirmedGroup: "software", BookingsWithGroup: 3,
		Treatment: domain.TaxTreatmentDomestic, Currency: "EUR",
		Net: map[domain.TaxRate]domain.Cents{domain.TaxRateStandard: 10000}, Gross: 11900,
		Groups: []string{"software"}, Settlement: "open",
		BookingDate: "2026-05-04", DocumentDate: "2026-05-04", Today: "2026-05-10",
	}
}

func TestReceiptRules(t *testing.T) {
	limits := LimitsFrom(nil)
	if limits.MaxGross != 100000 || limits.MinBookings != 3 {
		t.Fatalf("Voreinstellung: %+v", limits)
	}
	if v := ClassifyReceipt(routine(), limits); !v.Direct() || len(v.Reasons) != 0 {
		t.Fatalf("Routine wird %v: %+v", v.Level, v.Reasons)
	}
	cases := []struct {
		name   string
		change func(*ReceiptFacts)
		reason string
	}{
		{"gebucht", func(f *ReceiptFacts) { f.Open = false }, "beleg_nicht_offen"},
		{"ohne Darstellung", func(f *ReceiptFacts) { f.Displayable = false }, "keine_darstellung"},
		{"Foto", func(f *ReceiptFacts) { f.Structured = false }, "keine_e_rechnung"},
		{"ungeprüft", func(f *ReceiptFacts) { f.Checked = false }, "e_rechnung_ungeprueft"},
		{"Befund", func(f *ReceiptFacts) { f.Findings = 1 }, "e_rechnung_mit_befund"},
		{"kein Vorschlag", func(f *ReceiptFacts) { f.ProposalError = "Gutschrift" }, "kein_vorschlag"},
		{"anderer Aussteller", func(f *ReceiptFacts) { f.ProposalContactID = 8 }, "kontakt_nicht_erkannt"},
		{"nur der Name", func(f *ReceiptFacts) { f.ProposalMatchedBy = "name" }, "kontakt_nicht_erkannt"},
		{"Kopfdaten", func(f *ReceiptFacts) { f.HeaderMismatch = []string{"belegdatum"} }, "kopf_weicht_ab"},
		{"Stammdaten", func(f *ReceiptFacts) { f.MasterDataMismatch = []string{"anschrift"} }, "stammdaten_weichen_ab"},
		{"anderes Konto", func(f *ReceiptFacts) { f.IBANMismatch = true }, "iban_weicht_ab"},
		{"gesperrt", func(f *ReceiptFacts) { f.ContactBlocked = true }, "kontakt_gesperrt"},
		{"nur Vorschlag", func(f *ReceiptFacts) { f.ConfirmedGroup = "" }, "keine_bestaetigte_standardgruppe"},
		{"andere Gruppe", func(f *ReceiptFacts) { f.Groups = []string{"software", "werbung"} }, "gruppe_weicht_ab"},
		{"zwei Buchungen", func(f *ReceiptFacts) { f.BookingsWithGroup = 2 }, "zu_wenige_buchungen"},
		{"Reverse Charge", func(f *ReceiptFacts) { f.Treatment = domain.TaxTreatmentReverseChargeEU }, "steuerfall_nicht_inland"},
		{"Dollar", func(f *ReceiptFacts) { f.Currency = "USD" }, "fremdwaehrung"},
		{"ein Cent", func(f *ReceiptFacts) { f.Gross = 11901 }, "betraege_weichen_ab"},
		{"anderer Satz", func(f *ReceiptFacts) {
			f.Net = map[domain.TaxRate]domain.Cents{domain.TaxRateReduced: 10000}
		}, "betraege_weichen_ab"},
		{"an der Grenze", func(f *ReceiptFacts) {
			f.Gross, f.ProposalGross = 100000, 100000
		}, "ueber_direktgrenze"},
		{"Bewirtung", func(f *ReceiptFacts) { f.Special = []string{"bewirtung"} }, "sonderangaben"},
		{"Auslage", func(f *ReceiptFacts) { f.Settlement = "shareholder_expense" }, "bezahlart"},
		{"Zukunft", func(f *ReceiptFacts) { f.DocumentDate = "2026-06-01" }, "datum_in_zukunft"},
		{"Dublette", func(f *ReceiptFacts) { f.DuplicateOf = "ER-2026-0003" }, "dublettenverdacht"},
	}
	for _, tc := range cases {
		f := routine()
		tc.change(&f)
		v := ClassifyReceipt(f, limits)
		if v.Direct() || !slices.Contains(v.Keys(), tc.reason) {
			t.Errorf("%s: %v %v, erwartet %s", tc.name, v.Level, v.Keys(), tc.reason)
		}
	}
	// Direkt geht nur, was unter der Grenze liegt.
	f := routine()
	f.Gross, f.ProposalGross = 99999, 99999
	if v := ClassifyReceipt(f, limits); !v.Direct() {
		t.Errorf("ein Cent unter der Grenze: %v", v.Keys())
	}
	// Eingestellte Schwellen gelten.
	strict := LimitsFrom(&domain.AgentSettings{DirectMaxGross: 5000, DirectMinBookings: 5})
	if v := ClassifyReceipt(routine(), strict); !slices.Contains(v.Keys(), "ueber_direktgrenze") ||
		!slices.Contains(v.Keys(), "zu_wenige_buchungen") {
		t.Errorf("eingestellte Schwellen: %v", v.Keys())
	}
}

func TestBankRules(t *testing.T) {
	base := BankFacts{Open: true, Currency: "EUR", Trusted: true, Requested: 4, Candidates: []BankCandidate{{EntryID: 4, Reference: "RE-2026-0004"}}}
	if v := ClassifyBankMatch(base); !v.Direct() {
		t.Fatalf("exakter Treffer: %v", v.Keys())
	}
	cases := []struct {
		name   string
		change func(*BankFacts)
		reason string
	}{
		{"zugeordnet", func(f *BankFacts) { f.Open = false }, "umsatz_nicht_offen"},
		{"Dollar", func(f *BankFacts) { f.Currency = "USD" }, "fremdwaehrung"},
		{"ungeprüfter Auszug", func(f *BankFacts) { f.Trusted = false }, "auszug_ungeprueft"},
		{"ohne Nummer", func(f *BankFacts) { f.Candidates[0].Reference = "" }, "kein_exakter_treffer"},
		{"kein Betrag", func(f *BankFacts) { f.Candidates = nil }, "kein_exakter_treffer"},
		{"zwei", func(f *BankFacts) {
			f.Candidates = append(f.Candidates, BankCandidate{EntryID: 5, Reference: "RE-2026-0005"})
		}, "mehrere_treffer"},
		{"anderer", func(f *BankFacts) { f.Requested = 9 }, "anderer_posten"},
	}
	for _, tc := range cases {
		f := base
		f.Candidates = append([]BankCandidate{}, base.Candidates...)
		tc.change(&f)
		if v := ClassifyBankMatch(f); v.Direct() || !slices.Contains(v.Keys(), tc.reason) {
			t.Errorf("%s: %v, erwartet %s", tc.name, v.Keys(), tc.reason)
		}
	}
	// Ein zweiter Posten mit gleichem Betrag, aber ohne Nummer, stört nicht.
	f := base
	f.Candidates = []BankCandidate{{EntryID: 4, Reference: "RE-2026-0004"}, {EntryID: 5}}
	if v := ClassifyBankMatch(f); !v.Direct() {
		t.Errorf("gleicher Betrag ohne Nummer: %v", v.Keys())
	}
}

func TestDuplicateContact(t *testing.T) {
	existing := []domain.Contact{
		{ID: 1, Type: domain.ContactTypeVendor, Name: "Netzwerk GmbH", PostalCode: "20095", VatID: "DE555666777"},
		{ID: 2, Type: domain.ContactTypeVendor, Name: "Papier AG", IBAN: "DE02 1203 0000 0000 2020 51"},
		{ID: 3, Type: domain.ContactTypeCustomer, Name: "Kunde KG", PostalCode: "80331"},
	}
	cases := []struct {
		c     domain.Contact
		id    uint
		match string
	}{
		{domain.Contact{Type: domain.ContactTypeVendor, Name: "Anders", VatID: "de 555666777"}, 1, "ust_id"},
		{domain.Contact{Type: domain.ContactTypeVendor, Name: "Anders", IBAN: "DE02120300000000202051"}, 2, "iban"},
		{domain.Contact{Type: domain.ContactTypeVendor, Name: "netzwerk gmbh", PostalCode: "20095"}, 1, "name_plz"},
		{domain.Contact{Type: domain.ContactTypeVendor, Name: "Netzwerk GmbH", PostalCode: "10115"}, 0, ""},
		{domain.Contact{Type: domain.ContactTypeVendor, Name: "Kunde KG", PostalCode: "80331"}, 0, ""},
	}
	for _, tc := range cases {
		found, match := DuplicateContact(existing, tc.c)
		id := uint(0)
		if found != nil {
			id = found.ID
		}
		if id != tc.id || match != tc.match {
			t.Errorf("%+v: %d %q, erwartet %d %q", tc.c, id, match, tc.id, tc.match)
		}
	}
}
