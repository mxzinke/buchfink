package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Die Skonto-Fassade und der Zahlungsdienst müssen Konten, Steuerzeilen und Rundung gleich bestimmen.
func TestTaxCaseFacadeMatchesSkonto(t *testing.T) {
	env := newTestEnv(t)
	payments := env.payments(t)
	for _, direction := range []domain.Direction{domain.DirectionIncoming, domain.DirectionOutgoing} {
		for _, info := range domain.TaxTreatments(direction) {
			c := accounting.TaxCaseFor(info.Treatment)
			for _, rate := range domain.ValidTaxRates() {
				for _, discount := range []domain.Cents{0, 1, 1190, -1190} {
					t.Run(fmt.Sprintf("%s/%s/%d/%d", direction, info.Treatment, rate, discount), func(t *testing.T) {
						item := domain.OpenItem{TaxTreatment: info.Treatment, TaxRate: rate, ContactType: domain.ContactTypeVendor}
						side := domain.SideCredit
						if direction == domain.DirectionOutgoing {
							item.ContactType, side = domain.ContactTypeCustomer, domain.SideDebit
						}
						lines, _, serviceErr := payments.skontoLines(context.Background(), AllocationRequest{DifferenceAmount: discount}, item, side)
						got, err := c.Skonto(direction, rate, discount)
						if (err != nil) != (serviceErr != nil) {
							t.Fatalf("Fehler Fassade=%v, Dienst=%v", err, serviceErr)
						}
						if err != nil {
							return
						}
						if len(lines) != len(got.Legs)+1 || lines[0].Account != got.Account || lines[0].Amount != got.Net {
							t.Fatalf("Skonto Fassade=%+v, Dienst=%+v", got, lines)
						}
						for i, leg := range got.Legs {
							line := lines[i+1]
							if line.Account != leg.Account || line.TaxKey != leg.Key || line.Amount != leg.Amount || line.TaxBase != leg.Base || line.Side != leg.Side {
								t.Errorf("Steuerzeile Fassade=%+v, Dienst=%+v", leg, line)
							}
						}
					})
				}
			}
		}
	}
}
