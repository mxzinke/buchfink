package bank

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/buchfink/buchfink/internal/models"
)

// Minimal CAMT.053 XML Structs
type Document struct {
	XMLName xml.Name `xml:"Document"`
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
				CdtDbtInd string `xml:"CdtDbtInd"` // "CRDT" (Credit/Plus) or "DBIT" (Debit/Minus)
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

// ParseCAMT053 parses an ISO 20022 CAMT.053 XML file content into BankTransaction models.
func ParseCAMT053(r io.Reader) ([]models.BankTransaction, error) {
	var doc Document
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("failed to parse CAMT.053 XML: %w", err)
	}

	stmt := doc.BkToCstmrStmt.Stmt
	iban := stmt.Acct.Id.IBAN

	var transactions []models.BankTransaction
	for _, ntry := range stmt.Ntry {
		val, _ := strconv.ParseFloat(ntry.Amt.Value, 64)
		if ntry.CdtDbtInd == "DBIT" {
			val = -val
		}

		tx := models.BankTransaction{
			AccountIBAN:    iban,
			BookingDate:    ntry.BookgDt.Dt,
			ValueDate:      ntry.ValDt.Dt,
			Amount:         val,
			Currency:       ntry.Amt.Ccy,
			RemittanceInfo: ntry.NtryDtls.TxDtls.RmtInf.Ustrd,
			EndToEndID:     ntry.NtryDtls.TxDtls.Refs.EndToEndId,
			MatchStatus:    "unmatched",
		}

		if val > 0 {
			tx.CounterpartyName = ntry.NtryDtls.TxDtls.RltdPties.Dbtr.Nm
			tx.CounterpartyIBAN = ntry.NtryDtls.TxDtls.RltdPties.DbtrAcct.Id.IBAN
			// Heuristic suggestions
			tx.SuggestedAccount = "4400" // Erlöse 19%
		} else {
			tx.CounterpartyName = ntry.NtryDtls.TxDtls.RltdPties.Cdtr.Nm
			tx.CounterpartyIBAN = ntry.NtryDtls.TxDtls.RltdPties.CdtrAcct.Id.IBAN
			// Heuristic suggestions based on text
			lowerRem := strings.ToLower(tx.RemittanceInfo + " " + tx.CounterpartyName)
			if strings.Contains(lowerRem, "telekom") || strings.Contains(lowerRem, "vodafone") || strings.Contains(lowerRem, "hosting") {
				tx.SuggestedAccount = "6800" // IT & Kommunikation
			} else if strings.Contains(lowerRem, "miete") || strings.Contains(lowerRem, "immobilie") {
				tx.SuggestedAccount = "6500" // Miete
			} else if strings.Contains(lowerRem, "steuer") || strings.Contains(lowerRem, "finanzamt") {
				tx.SuggestedAccount = "3820" // USt Vorauszahlung
			} else if strings.Contains(lowerRem, "software") || strings.Contains(lowerRem, "adobe") || strings.Contains(lowerRem, "github") {
				tx.SuggestedAccount = "6800" // Software & SaaS
			} else {
				tx.SuggestedAccount = "3300" // Verbindlichkeiten aus LuL
			}
		}

		transactions = append(transactions, tx)
	}

	return transactions, nil
}

