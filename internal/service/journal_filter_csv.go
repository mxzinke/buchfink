package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// JournalFilterCSV schreibt die gefilterte Journalmenge als CSV (PRF-01 K3).
//
// Semikolon als Trennzeichen und UTF-8 ohne BOM — dieselbe Form wie die
// Datenüberlassung und die Voranmeldung. Die BOM bleibt weg, weil sie keine
// Kodierungsangabe ist, sondern drei unsichtbare Bytes vor dem ersten
// Spaltennamen (siehe internal/export/csv.go).
//
// Die Summenzeile steht mit in der Datei. Sie ist der Grund, aus dem gefiltert
// wurde — wer die Zahl in der Ansicht sieht und in der Datei nicht wiederfindet,
// rechnet sie von Hand nach.
func JournalFilterCSV(result *accounting.JournalFilterResult) string {
	var b strings.Builder
	w := csv.NewWriter(&b)
	w.Comma = ';'
	_ = w.Write([]string{
		"Buchungsnummer", "Buchungsdatum", "Belegdatum", "Belegnummer", "Buchungstext",
		"Position", "Konto", "Kontobezeichnung", "Soll", "Haben",
		"Steuerschlüssel", "Bemessungsgrundlage", "Zeilentext", "Bearbeiter", "Beleg",
	})
	if result != nil {
		for _, row := range result.Rows {
			debit, credit := "", ""
			if row.Side == domain.SideDebit {
				debit = row.Amount.String()
			} else {
				credit = row.Amount.String()
			}
			receipt := "nein"
			if row.ReceiptID != nil && *row.ReceiptID != 0 {
				receipt = "ja"
			}
			taxBase := ""
			if row.TaxBase != 0 {
				taxBase = row.TaxBase.String()
			}
			_ = w.Write([]string{
				row.EntryNumber, row.BookingDate, row.DocumentDate, row.DocumentNumber,
				row.Description, strconv.Itoa(row.Position), row.Account, row.AccountName,
				debit, credit, row.TaxKey, taxBase, row.Text, row.Actor, receipt,
			})
		}
		_ = w.Write([]string{
			"Summe", "", "", "", "", "", "", "",
			result.TotalDebit.String(), result.TotalCredit.String(), "", "", "", "", "",
		})
	}
	w.Flush()
	return b.String()
}

// FilterEntriesCSV gibt die gefilterte Journalmenge als CSV heraus und
// protokolliert die Herausgabe (QUE-02 K2).
//
// Protokolliert, weil die Zeilen personenbezogene Daten enthalten: die
// Bearbeiterkennung jeder Buchung und die Buchungstexte, in denen Namen stehen.
// Wer eine solche Menge aus dem Programm herausträgt, führt eine
// Datenüberlassung durch — dieselbe Sache, die der Z3-Export und das
// Prüferpaket protokollieren, nur über einen anderen Weg. Die
// Verfahrensdokumentation sagt zu, dass jede von ihnen im Protokoll steht; ein
// stiller Weg hinaus machte diese Zusage unwahr.
//
// Der Filter steht im Protokolleintrag: eine Herausgabe ohne die Menge, die
// herausgegeben wurde, ließe sich später nicht beurteilen.
func (s *AccountingService) FilterEntriesCSV(
	ctx context.Context, filter accounting.JournalFilter,
) (string, error) {
	result, err := s.FilterEntries(ctx, filter)
	if err != nil {
		return "", err
	}
	if s.auditRepo != nil {
		rows := 0
		if result != nil {
			rows = len(result.Rows)
		}
		_ = s.auditRepo.Log(ctx, domain.AuditActionExport, "JOURNAL_FILTER",
			JournalFilterLabel(filter),
			fmt.Sprintf("Gefiltertes Journal als CSV herausgegeben: %d Zeilen, Filter %s",
				rows, JournalFilterLabel(filter)))
	}
	return JournalFilterCSV(result), nil
}

// JournalFilterLabel beschreibt den Filter in einem Satzteil für das Protokoll.
func JournalFilterLabel(filter accounting.JournalFilter) string {
	if filter.IsEmpty() {
		return "ohne Einschränkung"
	}
	parts := make([]string, 0, 9)
	add := func(label, value string) {
		if strings.TrimSpace(value) != "" {
			parts = append(parts, label+" "+value)
		}
	}
	add("von", filter.From)
	add("bis", filter.To)
	add("Konto", filter.Account)
	add("Gegenkonto", filter.CounterAccount)
	if filter.AmountFrom != nil {
		add("Betrag ab", filter.AmountFrom.String())
	}
	if filter.AmountTo != nil {
		add("Betrag bis", filter.AmountTo.String())
	}
	add("Steuerschlüssel", filter.TaxKey)
	add("Bearbeiter", filter.Actor)
	add("Suchtext", strings.TrimSpace(filter.Text))
	if filter.HasReceipt != nil {
		if *filter.HasReceipt {
			parts = append(parts, "nur mit Beleg")
		} else {
			parts = append(parts, "nur ohne Beleg")
		}
	}
	return strings.Join(parts, ", ")
}
