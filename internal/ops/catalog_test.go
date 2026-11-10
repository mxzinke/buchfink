package ops

import "testing"

func TestCatalogDefinitionsAreComplete(t *testing.T) {
	bridges := map[string]string{}
	for _, op := range All() {
		if op.Tier != AppOnly {
			if op.Enforcement != "" || len(op.Guards) != 0 || len(op.Bridge) != 0 {
				t.Errorf("%s ist zugleich MCP-Vorgang und der App vorbehalten", op.Name)
			}
			continue
		}
		if op.Description == "" || len(op.Guards) == 0 || (op.Enforcement != Channel && op.Enforcement != Structural) {
			t.Errorf("AppOnly-Definition unvollständig: %+v", op)
		}
		for _, method := range op.Bridge {
			if previous := bridges[method]; previous != "" {
				t.Errorf("%s gehört zu %s und %s", method, previous, op.Name)
			}
			bridges[method] = op.Name
		}
	}
}
