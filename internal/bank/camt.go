// SPDX-FileCopyrightText: 2026 Maximilian Pfennig
// SPDX-License-Identifier: EUPL-1.2

package bank

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/buchfink/buchfink/internal/domain"
)

// Minimal CAMT.053 XML structures.
type Document struct {
	XMLName       xml.Name `xml:"Document"`
	BkToCstmrStmt struct {
		Stmt struct {
			Acct struct {
				Id struct {
					IBAN string `xml:"IBAN"`
				} `xml:"Id"`
			} `xml:"Acct"`
			Ntry []struct {
				Amt struct {
					Value string `xml:",chardata"`
					Ccy   string `xml:"Ccy,attr"`
				} `xml:"Amt"`
				CdtDbtInd string `xml:"CdtDbtInd"` // "CRDT" (Eingang) oder "DBIT" (Ausgang)
				Sts       string `xml:"Sts"`
				BookgDt   struct {
					Dt string `xml:"Dt"`
				} `xml:"BookgDt"`
				ValDt struct {
					Dt string `xml:"Dt"`
				} `xml:"ValDt"`
				NtryDtls struct {
					TxDtls struct {
						Refs struct {
							EndToEndId string `xml:"EndToEndId"`
						} `xml:"Refs"`
						RltdPties struct {
							Dbtr struct {
								Nm string `xml:"Nm"`
							} `xml:"Dbtr"`
							Cdtr struct {
								Nm string `xml:"Nm"`
							} `xml:"Cdtr"`
							DbtrAcct struct {
								Id struct {
									IBAN string `xml:"IBAN"`
								} `xml:"Id"`
							} `xml:"DbtrAcct"`
							CdtrAcct struct {
								Id struct {
									IBAN string `xml:"IBAN"`
								} `xml:"Id"`
							} `xml:"CdtrAcct"`
						} `xml:"RltdPties"`
						RmtInf struct {
							Ustrd string `xml:"Ustrd"`
						} `xml:"RmtInf"`
					} `xml:"TxDtls"`
				} `xml:"NtryDtls"`
			} `xml:"Ntry"`
		} `xml:"Stmt"`
	} `xml:"BkToCstmrStmt"`
}

// ParseCAMT053 reads an ISO 20022 CAMT.053 statement into bank transactions.
//
// The parser records what the bank reported and nothing more. It deliberately
// suggests no accounts: guessing an expense account from the payment reference
// would put an unverifiable heuristic in front of the booking rules, and the
// account it picks is exactly the decision the user has to make and answer for.
func ParseCAMT053(r io.Reader) ([]domain.BankTransaction, error) {
	var doc Document
	if err := xml.NewDecoder(r).Decode(&doc); err != nil {
		return nil, fmt.Errorf("CAMT.053-Datei konnte nicht gelesen werden: %w", err)
	}

	stmt := doc.BkToCstmrStmt.Stmt
	iban := stmt.Acct.Id.IBAN

	transactions := make([]domain.BankTransaction, 0, len(stmt.Ntry))
	for i, ntry := range stmt.Ntry {
		amount, err := domain.ParseCents(ntry.Amt.Value)
		if err != nil {
			return nil, fmt.Errorf("Umsatz %d: Betrag %q konnte nicht gelesen werden: %w", i+1, ntry.Amt.Value, err)
		}
		// CAMT reports the amount as a positive number plus a direction marker.
		if ntry.CdtDbtInd == "DBIT" {
			amount = -amount
		}

		tx := domain.BankTransaction{
			AccountIBAN:    iban,
			BookingDate:    ntry.BookgDt.Dt,
			ValueDate:      ntry.ValDt.Dt,
			Amount:         amount,
			Currency:       ntry.Amt.Ccy,
			RemittanceInfo: ntry.NtryDtls.TxDtls.RmtInf.Ustrd,
			EndToEndID:     ntry.NtryDtls.TxDtls.Refs.EndToEndId,
			MatchStatus:    domain.MatchStatusUnmatched,
		}

		if amount > 0 {
			tx.CounterpartyName = ntry.NtryDtls.TxDtls.RltdPties.Dbtr.Nm
			tx.CounterpartyIBAN = ntry.NtryDtls.TxDtls.RltdPties.DbtrAcct.Id.IBAN
		} else {
			tx.CounterpartyName = ntry.NtryDtls.TxDtls.RltdPties.Cdtr.Nm
			tx.CounterpartyIBAN = ntry.NtryDtls.TxDtls.RltdPties.CdtrAcct.Id.IBAN
		}

		transactions = append(transactions, tx)
	}

	return transactions, nil
}
