package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

type adapterSettings struct {
	domain.SettingsRepository
	company *domain.CompanySettings
	err     error
}

func (r adapterSettings) GetCompanySettings(context.Context) (*domain.CompanySettings, error) {
	return r.company, r.err
}

func TestReceiptPeriodDifferenceUsesTenantPeriod(t *testing.T) {
	for _, tc := range []struct{ kind, document, received, docPeriod, receivedPeriod string }{
		{"month", "2026-01-31", "2026-02-01", "2026-01", "2026-02"},
		{"quarter", "2026-01-31", "2026-02-01", "", ""},
		{"quarter", "2026-03-31", "2026-04-01", "2026-Q1", "2026-Q2"},
		{"year", "2026-03-31", "2026-04-01", "", ""},
		{"year", "2025-12-31", "2026-01-01", "2025", "2026"},
		{"month", "2026-03-31", "2026-03-01", "", ""},
		{"month", "", "2026-04-01", "", ""},
		{"month", "2026-03-31", "ungueltig", "", ""},
	} {
		t.Run(tc.kind+tc.document+tc.received, func(t *testing.T) {
			s := &ReceiptService{settingsRepo: adapterSettings{company: &domain.CompanySettings{VatPeriod: tc.kind}}}
			got := s.InputTaxPeriodDifference(context.Background(), &domain.Receipt{DocumentDate: tc.document, ReceivedAt: tc.received})
			var want *InputTaxPeriodDifference
			if tc.docPeriod != "" {
				want = &InputTaxPeriodDifference{Document: tc.docPeriod, Received: tc.receivedPeriod}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("Zeiträume: %+v, erwartet %+v", got, want)
			}
		})
	}
}

func TestPeriodHintsIgnoreUnavailableSettings(t *testing.T) {
	for _, repo := range []domain.SettingsRepository{nil, adapterSettings{}, adapterSettings{err: errors.New("nicht lesbar")}} {
		receipts := &ReceiptService{settingsRepo: repo}
		returns := &VatReturnService{settingsRepo: repo}
		if got := receipts.InputTaxPeriodDifference(context.Background(), &domain.Receipt{DocumentDate: "2026-03-31", ReceivedAt: "2026-04-01"}); got != nil {
			t.Fatalf("Hinweis ohne Einstellungen: %+v", got)
		}
		if got := returns.PeriodMismatch(context.Background(), "2026-03"); got != "" {
			t.Fatalf("Zeitraum ohne Einstellungen: %s", got)
		}
	}
}

func TestPeriodMismatchUsesTenantPeriod(t *testing.T) {
	for _, kind := range []string{"month", "quarter", "year", "ungueltig"} {
		s := &VatReturnService{settingsRepo: adapterSettings{company: &domain.CompanySettings{VatPeriod: kind}}}
		for period, periodKind := range map[string]string{"2026-03": "month", "2026-Q1": "quarter", "2026": "year", "ungueltig": ""} {
			want := domain.VatPeriodType("")
			if periodKind != "" && kind != "ungueltig" && periodKind != kind {
				want = domain.VatPeriodType(kind)
			}
			if got := s.PeriodMismatch(context.Background(), period); got != want {
				t.Fatalf("%s/%s: %s, erwartet %s", kind, period, got, want)
			}
		}
	}
}
