package ebilanz

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestMappingReportWithoutFindingsReturnsEmptyLists(t *testing.T) {
	for _, stmt := range []*domain.Statement{nil, {}} {
		report, err := BuildMappingReport(2026, stmt, nil)
		if err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(report)
		if err != nil {
			t.Fatal(err)
		}
		for _, field := range []string{"rows", "blocking", "fallbacks"} {
			if !strings.Contains(string(data), `"`+field+`":[]`) {
				t.Errorf("%s must be an empty list: %s", field, data)
			}
		}
	}
}
