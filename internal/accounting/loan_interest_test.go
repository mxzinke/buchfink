package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// 50.000 € zu 3,5 % über das Kalenderjahr 2026: 1.750,00 €.
func TestLoanInterestFullYear(t *testing.T) {
	got, err := LoanInterest([]LoanMovement{{Date: "2025-12-01", Amount: 5_000_000}},
		350, domain.DayCountAct365, "2026-01-01", "2026-12-31")
	if err != nil {
		t.Fatal(err)
	}
	if got.Amount != 175_000 {
		t.Errorf("Zinsen = %s €, erwartet 1.750,00", got.Amount)
	}
	if len(got.Segments) != 1 || got.Segments[0].Days != 365 {
		t.Errorf("Abschnitte = %+v, erwartet einer über 365 Tage", got.Segments)
	}
}

// Auszahlung am 15.03.2026, 20.000 € zu 4 %: act/365 292 Tage = 640,00 €, die
// deutsche Zinsmethode 286 Tage = 635,56 €.
func TestLoanInterestFromDisbursementDay(t *testing.T) {
	movements := []LoanMovement{{Date: "2026-03-15", Amount: 2_000_000}}
	act, err := LoanInterest(movements, 400, domain.DayCountAct365, "2026-01-01", "2026-12-31")
	if err != nil {
		t.Fatal(err)
	}
	if act.Amount != 64_000 {
		t.Errorf("act/365: %s €, erwartet 640,00", act.Amount)
	}
	german, err := LoanInterest(movements, 400, domain.DayCount30E360, "2026-01-01", "2026-12-31")
	if err != nil {
		t.Fatal(err)
	}
	if german.Amount != 63_556 {
		t.Errorf("30/360: %s €, erwartet 635,56", german.Amount)
	}
}

// Tilgung am 01.07.2026 von 10.000 € auf 30.000 € zu 5 % (act/360): bis
// 30.06. 181 Tage auf 30.000 €, ab 01.07. 184 Tage auf 20.000 €.
// 30.000 × 5 % × 181/360 = 754,17; 20.000 × 5 % × 184/360 = 511,11.
func TestLoanInterestFollowsRepayment(t *testing.T) {
	got, err := LoanInterest([]LoanMovement{
		{Date: "2025-06-01", Amount: 3_000_000},
		{Date: "2026-07-01", Amount: -1_000_000},
	}, 500, domain.DayCountAct360, "2026-01-01", "2026-12-31")
	if err != nil {
		t.Fatal(err)
	}
	if got.Amount != 126_528 {
		t.Errorf("Zinsen = %s €, erwartet 1.265,28", got.Amount)
	}
	if len(got.Segments) != 2 || got.Segments[0].Days != 181 || got.Segments[1].Days != 184 ||
		got.Segments[1].Balance != 2_000_000 {
		t.Errorf("Abschnitte = %+v", got.Segments)
	}
}

// Ein zinsloses Darlehen hat null Zinsen, ein Zeitraum vor der Auszahlung auch.
func TestLoanInterestZero(t *testing.T) {
	got, err := LoanInterest([]LoanMovement{{Date: "2026-06-01", Amount: 1_000_000}},
		0, domain.DayCountAct365, "2026-01-01", "2026-12-31")
	if err != nil || got.Amount != 0 {
		t.Errorf("zinslos: %s €, %v", got.Amount, err)
	}
	got, err = LoanInterest([]LoanMovement{{Date: "2026-06-01", Amount: 1_000_000}},
		500, domain.DayCountAct365, "2026-01-01", "2026-05-31")
	if err != nil || got.Amount != 0 {
		t.Errorf("vor der Auszahlung: %s €, %v", got.Amount, err)
	}
	if _, err := LoanInterest(nil, 500, domain.DayCountAct365, "2026-02-01", "2026-01-01"); err == nil {
		t.Error("ein Zinsende vor dem Beginn muss abgewiesen werden")
	}
}
