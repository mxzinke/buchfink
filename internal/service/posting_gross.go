package service

import (
	"fmt"
	"sort"

	"github.com/buchfink/buchfink/internal/domain"
)

// grossPositions setzt die Nettobeträge eines Belegs ein, der nur Bruttobeträge
// je Steuersatz nennt: Kleinbetragsrechnung und Fahrausweis. Die Steuer wird
// aus dem Bruttobetrag herausgerechnet, einmal je Steuersatz
// (§ 35 Abs. 1 und 2 UStDV). Zurück kommt die Steuer je Satz; sie ersetzt den
// aus dem Nettobetrag gerechneten Betrag, der um einen Cent abweichen kann.
func grossPositions(req *ReceiptRequest) (map[domain.TaxRate]domain.Cents, error) {
	gross := 0
	for _, p := range req.Positions {
		if p.Gross != 0 {
			gross++
		}
	}
	if gross == 0 {
		return nil, nil
	}
	if gross != len(req.Positions) {
		return nil, fmt.Errorf("die Positionen eines Belegs sind alle netto oder alle brutto")
	}
	if req.TaxTreatment != domain.TaxTreatmentDomestic {
		return nil, fmt.Errorf("Bruttobeträge gibt es nur bei einem Inlandsbeleg mit ausgewiesener Steuer; " +
			"ohne Steuer auf dem Beleg ist der Betrag netto")
	}
	if code := req.Currency; code != "" && code != "EUR" {
		return nil, fmt.Errorf("Bruttobeträge gibt es nur bei einem Beleg in Euro")
	}

	// Vorschau und Buchung rechnen mit derselben Anfrage; ihre Positionen
	// bleiben unverändert.
	req.Positions = append([]ReceiptPosition(nil), req.Positions...)
	byRate := map[domain.TaxRate][]int{}
	for i, p := range req.Positions {
		if p.Gross < 0 {
			return nil, fmt.Errorf("Position %d: der Bruttobetrag muss größer als null sein", i+1)
		}
		if p.Net != 0 {
			return nil, fmt.Errorf("Position %d: netto oder brutto, nicht beides", i+1)
		}
		byRate[p.TaxRate] = append(byRate[p.TaxRate], i)
	}
	rates := make([]domain.TaxRate, 0, len(byRate))
	for r := range byRate {
		rates = append(rates, r)
	}
	sort.Slice(rates, func(i, j int) bool { return rates[i] < rates[j] })

	tax := make(map[domain.TaxRate]domain.Cents, len(rates))
	for _, rate := range rates {
		var total, assigned domain.Cents
		for _, i := range byRate[rate] {
			total += req.Positions[i].Gross
		}
		net := rate.NetFromGross(total)
		for _, i := range byRate[rate] {
			p := &req.Positions[i]
			p.Net = rate.NetFromGross(p.Gross)
			assigned += p.Net
		}
		// Die Rundung je Position gleicht die letzte aus, damit der Satz als
		// Ganzes so gerechnet ist wie auf dem Beleg.
		last := &req.Positions[byRate[rate][len(byRate[rate])-1]]
		last.Net += net - assigned
		tax[rate] = total - net
	}
	return tax, nil
}
