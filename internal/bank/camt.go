package bank

import (
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

type camtDate struct {
	Date     string `xml:"Dt"`
	DateTime string `xml:"DtTm"`
}
type camtParty struct {
	Name  string `xml:"Nm"`
	Party struct {
		Name string `xml:"Nm"`
	} `xml:"Pty"`
}

func (p camtParty) name() string {
	if p.Name != "" {
		return p.Name
	}
	return p.Party.Name
}

type camtAccount struct {
	ID struct {
		IBAN string `xml:"IBAN"`
	} `xml:"Id"`
}
type camtDetail struct {
	Refs struct {
		EndToEndID         string `xml:"EndToEndId"`
		AccountServicerRef string `xml:"AcctSvcrRef"`
		TransactionID      string `xml:"TxId"`
	} `xml:"Refs"`
	Parties struct {
		Debtor          camtParty   `xml:"Dbtr"`
		Creditor        camtParty   `xml:"Cdtr"`
		DebtorAccount   camtAccount `xml:"DbtrAcct"`
		CreditorAccount camtAccount `xml:"CdtrAcct"`
	} `xml:"RltdPties"`
	Remittance struct {
		Lines []string `xml:"Ustrd"`
	} `xml:"RmtInf"`
}
type camtEntry struct {
	Amount struct {
		Value    string `xml:",chardata"`
		Currency string `xml:"Ccy,attr"`
	} `xml:"Amt"`
	Direction string `xml:"CdtDbtInd"`
	Status    struct {
		Text string `xml:",chardata"`
		Code string `xml:"Cd"`
	} `xml:"Sts"`
	Booking        camtDate `xml:"BookgDt"`
	Value          camtDate `xml:"ValDt"`
	Reference      string   `xml:"AcctSvcrRef"`
	EntryReference string   `xml:"NtryRef"`
	Details        []struct {
		Transactions []camtDetail `xml:"TxDtls"`
	} `xml:"NtryDtls"`
}
type Document struct {
	XMLName    xml.Name `xml:"Document"`
	Statements struct {
		Items []struct {
			Account camtAccount `xml:"Acct"`
			Entries []camtEntry `xml:"Ntry"`
		} `xml:"Stmt"`
	} `xml:"BkToCstmrStmt"`
}

var camtAmount = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,2})?$`)
var camtCurrency = regexp.MustCompile(`^[A-Z]{3}$`)

func readCAMTDate(d camtDate) (string, error) {
	if d.Date != "" {
		if _, err := time.Parse("2006-01-02", d.Date); err != nil {
			return "", err
		}
		return d.Date, nil
	}
	if d.DateTime != "" {
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05"} {
			if v, err := time.Parse(layout, d.DateTime); err == nil {
				return v.Format("2006-01-02"), nil
			}
		}
	}
	return "", fmt.Errorf("gültiges Datum fehlt")
}

// ParseCAMT053 preserves each statement's account. Unsupported collective
// bookings are rejected in full instead of silently taking their last detail.
func ParseCAMT053(r io.Reader) ([]domain.BankTransaction, error) {
	var doc Document
	dec := xml.NewDecoder(r)
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("CAMT.053-Datei konnte nicht gelesen werden: %w", err)
	}
	if doc.XMLName.Space != "" && !strings.Contains(doc.XMLName.Space, ":camt.053.") {
		return nil, fmt.Errorf("bitte einen Kontoauszug im Format CAMT.053 wählen")
	}
	if len(doc.Statements.Items) == 0 {
		return nil, fmt.Errorf("die Datei enthält keinen CAMT.053-Kontoauszug")
	}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("ungültiges XML nach dem Kontoauszug: %w", err)
		}
		if data, ok := tok.(xml.CharData); ok && strings.TrimSpace(string(data)) == "" {
			continue
		}
		if _, ok := tok.(xml.Comment); ok {
			continue
		}
		return nil, fmt.Errorf("unerwarteter Inhalt nach dem Kontoauszug")
	}
	transactions := make([]domain.BankTransaction, 0)
	for s, stmt := range doc.Statements.Items {
		iban := strings.ToUpper(strings.Join(strings.Fields(stmt.Account.ID.IBAN), ""))
		if iban == "" {
			return nil, fmt.Errorf("Auszug %d: IBAN des Geschäftskontos fehlt", s+1)
		}
		for i, e := range stmt.Entries {
			prefix := fmt.Sprintf("Auszug %d, Umsatz %d", s+1, i+1)
			if !camtAmount.MatchString(e.Amount.Value) {
				return nil, fmt.Errorf("%s: Betrag muss positiv und mit höchstens zwei Nachkommastellen angegeben sein", prefix)
			}
			amount, err := domain.ParseCents(e.Amount.Value)
			if err != nil || amount <= 0 {
				return nil, fmt.Errorf("%s: ungültiger Betrag %q", prefix, e.Amount.Value)
			}
			if e.Direction != "CRDT" && e.Direction != "DBIT" {
				return nil, fmt.Errorf("%s: Zahlungsrichtung muss CRDT oder DBIT sein", prefix)
			}
			if !camtCurrency.MatchString(e.Amount.Currency) {
				return nil, fmt.Errorf("%s: gültige Währung fehlt", prefix)
			}
			status := strings.TrimSpace(e.Status.Text)
			if e.Status.Code != "" {
				status = e.Status.Code
			}
			if status != "" && status != "BOOK" {
				return nil, fmt.Errorf("%s: nur gebuchte Umsätze werden übernommen (Status %s)", prefix, status)
			}
			booking, err := readCAMTDate(e.Booking)
			if err != nil {
				return nil, fmt.Errorf("%s: ungültiges Buchungsdatum", prefix)
			}
			value, err := readCAMTDate(e.Value)
			if err != nil {
				return nil, fmt.Errorf("%s: ungültiges Wertstellungsdatum", prefix)
			}
			var details []camtDetail
			for _, d := range e.Details {
				details = append(details, d.Transactions...)
			}
			if len(details) > 1 {
				return nil, fmt.Errorf("%s: Sammelbuchung mit %d Einzelzahlungen. Bitte bei der Bank einen Auszug mit einzeln ausgewiesenen Buchungen exportieren; es wurden keine Umsätze übernommen", prefix, len(details))
			}
			var detail camtDetail
			if len(details) == 1 {
				detail = details[0]
			}
			ref := strings.TrimSpace(detail.Refs.AccountServicerRef)
			if ref == "" {
				ref = strings.TrimSpace(e.Reference)
			}
			if ref == "" {
				ref = strings.TrimSpace(e.EntryReference)
			}
			if ref == "" {
				ref = strings.TrimSpace(detail.Refs.TransactionID)
			}
			if e.Direction == "DBIT" {
				amount = -amount
			}
			tx := domain.BankTransaction{AccountIBAN: iban, BookingDate: booking, ValueDate: value, Amount: amount, Currency: e.Amount.Currency, RemittanceInfo: strings.Join(detail.Remittance.Lines, "\n"), EndToEndID: strings.TrimSpace(detail.Refs.EndToEndID), BankReference: ref, MatchStatus: domain.MatchStatusUnmatched}
			if amount > 0 {
				tx.CounterpartyName = detail.Parties.Debtor.name()
				tx.CounterpartyIBAN = detail.Parties.DebtorAccount.ID.IBAN
			} else {
				tx.CounterpartyName = detail.Parties.Creditor.name()
				tx.CounterpartyIBAN = detail.Parties.CreditorAccount.ID.IBAN
			}
			transactions = append(transactions, tx)
		}
	}
	return transactions, nil
}
