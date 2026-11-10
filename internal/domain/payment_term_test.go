package domain

import (
	"strings"
	"testing"
)

// Ein Zahlungsziel über sechzig Tagen löst den Hinweis auf § 271a BGB aus —
// und darunter keinen.
func TestPaymentTermNoticeOnlyAboveSixtyDays(t *testing.T) {
	if notice := PaymentTermNotice(60); notice != "" {
		t.Errorf("sechzig Tage sind unauffällig: %q", notice)
	}
	notice := PaymentTermNotice(90)
	if notice == "" {
		t.Fatal("neunzig Tage müssen den Hinweis auslösen")
	}
	if !strings.Contains(notice, "271a") {
		t.Errorf("der Hinweis nennt die Norm nicht: %q", notice)
	}
	if got := (PaymentTerms{DueDays: 90}).LongTermNotice(); got != notice {
		t.Errorf("die Bedingungen liefern einen anderen Hinweis: %q", got)
	}
}
