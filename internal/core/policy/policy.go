// Package policy entscheidet, was ein verbundener Agent ohne Rückfrage
// schreiben darf, und prüft die Dateien, die er übergibt. Die Regeln sind
// rein: Fakten hinein, Urteil heraus. Die Fakten sammelt facts.go aus den
// Diensten des Mandanten, damit Werkzeug und Regel nicht auseinanderlaufen.
package policy

import (
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// Level ist die Freigabestufe eines Schreibvorgangs.
type Level string

const (
	// Direct: Buchfink stuft den Vorgang als Routine ein; der Agent schreibt
	// ohne Rückfrage.
	Direct Level = "direkt"
	// Confirm: der Agent holt vorher im Chat das Ja des Menschen ein.
	Confirm Level = "rueckfrage"
)

// Reason ist ein Grund gegen die direkte Stufe. Key ist stabil; Agenten
// verzweigen daran.
type Reason struct {
	Key  string `json:"schluessel"`
	Text string `json:"text"`
}

// Verdict ist das Urteil über einen Vorgang.
type Verdict struct {
	Level   Level    `json:"stufe"`
	Reasons []Reason `json:"gruende"`
}

// Direct meldet, ob der Vorgang ohne Rückfrage laufen darf.
func (v Verdict) Direct() bool { return v.Level == Direct }

// Keys sind die Schlüssel der Gründe.
func (v Verdict) Keys() []string {
	out := make([]string, 0, len(v.Reasons))
	for _, r := range v.Reasons {
		out = append(out, r.Key)
	}
	return out
}

func verdict(reasons []Reason) Verdict {
	if len(reasons) == 0 {
		return Verdict{Level: Direct, Reasons: []Reason{}}
	}
	return Verdict{Level: Confirm, Reasons: reasons}
}

// Limits sind die Schwellen der direkten Stufe.
type Limits struct {
	// MaxGross ist die Direktgrenze brutto je Beleg.
	MaxGross domain.Cents `json:"grenze_brutto"`
	// MinBookings ist die Zahl früherer Buchungen eines Lieferanten mit seiner
	// Standard-Buchungsgruppe.
	MinBookings int `json:"mindestbuchungen"`
}

// LimitsFrom setzt fehlende Schwellen auf die Voreinstellung.
func LimitsFrom(s *domain.AgentSettings) Limits {
	l := Limits{MaxGross: domain.DefaultDirectMaxGross, MinBookings: domain.DefaultDirectMinBookings}
	if s != nil && s.DirectMaxGross > 0 {
		l.MaxGross = s.DirectMaxGross
	}
	if s != nil && s.DirectMinBookings > 0 {
		l.MinBookings = s.DirectMinBookings
	}
	return l
}

// ReceiptFacts ist, was über eine beabsichtigte Belegbuchung bekannt ist.
type ReceiptFacts struct {
	// Der Beleg.
	Open       bool
	Incoming   bool
	Invoice    bool
	Structured bool
	Checked    bool
	Findings   int
	// Der Vorschlag aus dem strukturierten Teil; ProposalError ist gesetzt,
	// wenn sich keiner bilden ließ.
	ProposalError     string
	ProposalContactID uint
	ProposalTreatment domain.TaxTreatment
	ProposalCurrency  string
	ProposalNet       map[domain.TaxRate]domain.Cents
	ProposalGross     domain.Cents

	// Der Lieferant.
	ContactID         uint
	ContactBlocked    bool
	ConfirmedGroup    string
	BookingsWithGroup int

	// Die Anfrage.
	Treatment    domain.TaxTreatment
	Currency     string
	Net          map[domain.TaxRate]domain.Cents
	Gross        domain.Cents
	Groups       []string
	Special      []string
	Settlement   string
	BookingDate  string
	DocumentDate string
	Today        string

	// DuplicateOf nennt einen anderen Beleg mit Dublettenverdacht.
	DuplicateOf string
}

// ClassifyReceipt entscheidet, ob ein Eingangsbeleg ohne Rückfrage gebucht
// werden darf. Direkt nur, wenn alles zutrifft: vom Menschen bestätigte
// Standard-Buchungsgruppe des Lieferanten mit genug früheren Buchungen,
// E-Rechnung ohne Befund, Beträge centgenau aus der E-Rechnung, Inland in
// Euro, Brutto bis zur Direktgrenze, keine Sonderangaben, kein
// Dublettenverdacht.
func ClassifyReceipt(f ReceiptFacts, l Limits) Verdict {
	var r []Reason
	add := func(key, text string, args ...any) { r = append(r, Reason{Key: key, Text: fmt.Sprintf(text, args...)}) }

	if !f.Open {
		add("beleg_nicht_offen", "Der Beleg ist nicht offen (gebucht oder verworfen).")
	}
	if !f.Incoming || !f.Invoice {
		add("keine_eingangsrechnung", "Der Beleg ist keine Eingangsrechnung.")
	}
	switch {
	case !f.Structured:
		add("keine_e_rechnung", "Der Beleg hat keinen strukturierten Rechnungsdatensatz; Fotos und PDF ohne E-Rechnung bucht Buchfink nie direkt.")
	case !f.Checked:
		add("e_rechnung_ungeprueft", "Die E-Rechnung ist noch nicht geprüft.")
	case f.Findings > 0:
		add("e_rechnung_mit_befund", "Die Prüfung der E-Rechnung hat %d Befund(e).", f.Findings)
	}
	if f.Structured && f.ProposalError != "" {
		add("kein_vorschlag", "Aus der E-Rechnung lässt sich kein Buchungsvorschlag bilden: %s", f.ProposalError)
	}
	if f.Structured && f.ProposalError == "" && (f.ProposalContactID == 0 || f.ProposalContactID != f.ContactID) {
		add("kontakt_nicht_erkannt", "Der Aussteller der E-Rechnung ist nicht eindeutig dieser Kontakt.")
	}
	if f.ContactBlocked {
		add("kontakt_gesperrt", "Der Kontakt ist gesperrt.")
	}
	switch {
	case f.ConfirmedGroup == "":
		add("keine_bestaetigte_standardgruppe", "Der Lieferant hat keine vom Menschen bestätigte Standard-Buchungsgruppe.")
	default:
		for _, g := range f.Groups {
			if g != f.ConfirmedGroup {
				add("gruppe_weicht_ab", "Die Positionen nutzen nicht nur die bestätigte Standard-Buchungsgruppe %s.", f.ConfirmedGroup)
				break
			}
		}
		if len(f.Groups) == 0 {
			add("gruppe_weicht_ab", "Die Positionen nennen keine Buchungsgruppe.")
		}
		if f.BookingsWithGroup < l.MinBookings {
			add("zu_wenige_buchungen", "Der Lieferant hat erst %d Buchung(en) mit seiner Standardgruppe, nötig sind %d.",
				f.BookingsWithGroup, l.MinBookings)
		}
	}
	if f.Treatment != domain.TaxTreatmentDomestic || (f.Structured && f.ProposalError == "" && f.ProposalTreatment != domain.TaxTreatmentDomestic) {
		add("steuerfall_nicht_inland", "Der Steuerfall ist nicht Inland.")
	}
	if !isEuro(f.Currency) || (f.Structured && f.ProposalError == "" && !isEuro(f.ProposalCurrency)) {
		add("fremdwaehrung", "Der Beleg lautet nicht auf Euro.")
	}
	if f.Structured && f.ProposalError == "" && (!sameAmounts(f.Net, f.ProposalNet) || f.Gross != f.ProposalGross) {
		add("betraege_weichen_ab", "Netto je Steuersatz oder Brutto weichen von der E-Rechnung ab (Brutto %s € statt %s €).",
			f.Gross, f.ProposalGross)
	}
	if f.Gross > l.MaxGross {
		add("ueber_direktgrenze", "Brutto %s € liegt über der Direktgrenze von %s €.", f.Gross, l.MaxGross)
	}
	if len(f.Special) > 0 {
		add("sonderangaben", "Die Buchung trägt Sonderangaben: %v.", f.Special)
	}
	if f.Settlement != "open" && f.Settlement != "credit_card" {
		add("bezahlart", "Direkt gebucht wird nur offen oder mit der Firmenkreditkarte.")
	}
	if (f.BookingDate != "" && f.BookingDate > f.Today) || (f.DocumentDate != "" && f.DocumentDate > f.Today) {
		add("datum_in_zukunft", "Beleg- oder Buchungsdatum liegt in der Zukunft.")
	}
	if f.DuplicateOf != "" {
		add("dublettenverdacht", "Beleg %s hat denselben Aussteller und dieselbe Rechnungsnummer oder denselben Betrag am selben Tag.", f.DuplicateOf)
	}
	return verdict(r)
}

func isEuro(code string) bool { return code == "" || code == "EUR" }

func sameAmounts(a, b map[domain.TaxRate]domain.Cents) bool {
	clean := func(m map[domain.TaxRate]domain.Cents) map[domain.TaxRate]domain.Cents {
		out := map[domain.TaxRate]domain.Cents{}
		for k, v := range m {
			if v != 0 {
				out[k] = v
			}
		}
		return out
	}
	a, b = clean(a), clean(b)
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// BankCandidate ist ein offener Posten mit genau dem Betrag eines
// Bankumsatzes in derselben Richtung. Reference ist die Nummer, die der
// Verwendungszweck nennt, leer ohne Treffer.
type BankCandidate struct {
	EntryID   uint
	Reference string
}

// BankFacts ist, was über eine beabsichtigte Zuordnung bekannt ist.
type BankFacts struct {
	Open       bool
	Currency   string
	Requested  uint
	Candidates []BankCandidate
}

// ClassifyBankMatch entscheidet, ob ein Bankumsatz ohne Rückfrage einem
// offenen Posten zugeordnet werden darf: nur bei genau einem offenen Posten,
// der im Betrag centgenau übereinstimmt und dessen Nummer im
// Verwendungszweck steht, und nur diesem.
func ClassifyBankMatch(f BankFacts) Verdict {
	var r []Reason
	add := func(key, text string, args ...any) { r = append(r, Reason{Key: key, Text: fmt.Sprintf(text, args...)}) }
	if !f.Open {
		add("umsatz_nicht_offen", "Der Bankumsatz ist schon zugeordnet oder ignoriert.")
	}
	if !isEuro(f.Currency) {
		add("fremdwaehrung", "Der Bankumsatz lautet nicht auf Euro.")
	}
	var matches []BankCandidate
	for _, c := range f.Candidates {
		if c.Reference != "" {
			matches = append(matches, c)
		}
	}
	switch {
	case len(matches) == 0:
		add("kein_exakter_treffer", "Kein offener Posten stimmt im Betrag centgenau überein und steht mit seiner Nummer im Verwendungszweck.")
	case len(matches) > 1:
		add("mehrere_treffer", "%d offene Posten passen nach Betrag und Nummer.", len(matches))
	case matches[0].EntryID != f.Requested:
		add("anderer_posten", "Der exakte Treffer ist Buchung %d, nicht die gewählte.", matches[0].EntryID)
	}
	return verdict(r)
}
