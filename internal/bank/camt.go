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

// GenerateSampleCAMT053 creates sample transactions for demo/testing purposes.
func GenerateSampleCAMT053() []models.BankTransaction {
	return []models.BankTransaction{
		{
			AccountIBAN:      "DE89370400440532013000",
			BookingDate:      "2024-04-15",
			ValueDate:        "2024-04-15",
			Amount:           2856.00,
			Currency:         "EUR",
			CounterpartyName: "Acme Corp GmbH",
			CounterpartyIBAN: "DE12500105170648489890",
			RemittanceInfo:   "Rechnung RE-2024-042 Webentwicklung",
			EndToEndID:       "E2E-20240415-001",
			MatchStatus:      "unmatched",
			SuggestedAccount: "4400",
			SuggestedContact: "Acme Corp GmbH",
		},
		{
			AccountIBAN:      "DE89370400440532013000",
			BookingDate:      "2024-04-12",
			ValueDate:        "2024-04-12",
			Amount:           -89.25,
			Currency:         "EUR",
			CounterpartyName: "Hetzner Online GmbH",
			CounterpartyIBAN: "DE45700202700015762901",
			RemittanceInfo:   "Server Hosting Invoice 2024-4412",
			EndToEndID:       "E2E-20240412-002",
			MatchStatus:      "unmatched",
			SuggestedAccount: "6800",
			SuggestedContact: "Hetzner Online GmbH",
		},
		{
			AccountIBAN:      "DE89370400440532013000",
			BookingDate:      "2024-04-10",
			ValueDate:        "2024-04-10",
			Amount:           -650.00,
			Currency:         "EUR",
			CounterpartyName: "Immobilienverwaltung Schmidt",
			CounterpartyIBAN: "DE33200411550123456789",
			RemittanceInfo:   "Büromiete April 2024",
			EndToEndID:       "E2E-20240410-003",
			MatchStatus:      "unmatched",
			SuggestedAccount: "6500",
			SuggestedContact: "Immobilienverwaltung Schmidt",
		},
		{
			AccountIBAN:      "DE89370400440532013000",
			BookingDate:      "2024-04-05",
			ValueDate:        "2024-04-05",
			Amount:           -12.90,
			Currency:         "EUR",
			CounterpartyName: "Commerzbank AG",
			CounterpartyIBAN: "DE89370400440532013000",
			RemittanceInfo:   "Kontoführungsentgelt März 2024",
			EndToEndID:       "E2E-20240405-004",
			MatchStatus:      "unmatched",
			SuggestedAccount: "6870",
			SuggestedContact: "Commerzbank AG",
		},
	}
}
