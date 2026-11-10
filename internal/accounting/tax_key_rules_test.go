package accounting

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestEveryCatalogTaxKeyPassesOnItsAccount(t *testing.T) {
	for _, info := range TaxKeyCatalog() {
		line := domain.JournalLine{Side: info.Side, Account: info.Account, TaxKey: info.Key, Amount: 1900}
		if info.Rate != domain.TaxRateNone && info.Key != TaxKeyUnlawful && info.Key != TaxKeyInputTaxCorrection {
			line.TaxBase = 10000
			line.Amount = info.Rate.Tax(10000)
		}
		if err := CheckTaxLine(line); err != nil {
			t.Errorf("%s auf %s: %v", info.Key, info.Account, err)
		}
		if info.Rate != domain.TaxRateNone && info.Key != TaxKeyUnlawful && info.Key != TaxKeyInputTaxCorrection {
			line.TaxKey = TaxKeySkontoPrefix + info.Key
			line.Side = info.Side.Opposite()
			if err := CheckTaxLine(line); err != nil {
				t.Errorf("Skontokorrektur %s: %v", line.TaxKey, err)
			}
		}
	}
}

func TestCheckTaxLineRejectsWhatWouldFalsifyTheVatReturn(t *testing.T) {
	cases := []struct {
		name string
		line domain.JournalLine
		want string
	}{
		{"unbekannter Schlüssel",
			domain.JournalLine{Account: domain.AccountUmsatzsteuer19, TaxKey: "UST16", Amount: 1600, TaxBase: 10000},
			"unbekannter Steuerschlüssel"},
		{"Umsatzsteuer auf dem Vorsteuerkonto",
			domain.JournalLine{Account: domain.AccountVorsteuer19, TaxKey: "UST19", Amount: 1900, TaxBase: 10000},
			"gehört auf Konto 3806"},
		{"Satz passt nicht zur Grundlage",
			domain.JournalLine{Account: domain.AccountVorsteuer19, TaxKey: "VST19", Amount: 700, TaxBase: 10000},
			"passen nicht"},
		{"Bruttobetrag als Grundlage",
			domain.JournalLine{Account: domain.AccountUmsatzsteuer19, TaxKey: "UST19", Amount: 1900, TaxBase: 11900},
			"passen nicht"},
		{"ohne Bemessungsgrundlage",
			domain.JournalLine{Account: domain.AccountUmsatzsteuer7, TaxKey: "UST7", Amount: 700},
			"Bemessungsgrundlage"},
		{"verschiedene Vorzeichen",
			domain.JournalLine{Account: domain.AccountUmsatzsteuer19, TaxKey: "UST19", Amount: -1900, TaxBase: 10000},
			"Vorzeichen"},
		{"§ 14c auf einem anderen Konto",
			domain.JournalLine{Account: domain.AccountUmsatzsteuer19, TaxKey: TaxKeyUnlawful, Amount: 1900},
			"gehört auf Konto 3851"},
	}
	for _, c := range cases {
		err := CheckTaxLine(c.line)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: erwartet Fehler mit %q, erhalten %v", c.name, c.want, err)
		}
	}
}

func TestCheckTaxLineAllowsRoundingAndCorrections(t *testing.T) {
	ok := []domain.JournalLine{
		// Rundung je Position auf einer fremden Rechnung.
		{Side: domain.SideDebit, Account: domain.AccountVorsteuer19, TaxKey: "VST19", Amount: 1903, TaxBase: 10000},
		// Rechnungskorrektur auf der Gegenseite, gleicher Schlüssel.
		{Side: domain.SideDebit, Account: domain.AccountUmsatzsteuer19, TaxKey: "UST19", Amount: 1900, TaxBase: 10000},
		// Generalumkehr: beide negativ.
		{Side: domain.SideCredit, Account: domain.AccountUmsatzsteuer19, TaxKey: "UST19", Amount: -1900, TaxBase: -10000},
		{Side: domain.SideCredit, Account: domain.AccountUmsatzsteuer14c, TaxKey: TaxKeyUnlawful, Amount: 1900},
		{Side: domain.SideCredit, Account: InputTaxCorrectionRepayableImmovable, TaxKey: TaxKeyInputTaxCorrection, Amount: 12345},
		{Account: "4400", Amount: 10000},
	}
	for _, l := range ok {
		if err := CheckTaxLine(l); err != nil {
			t.Errorf("%+v: %v", l, err)
		}
	}
}
