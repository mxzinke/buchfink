package service

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var updateGolden = flag.Bool("update", false, "Golden-Dateien mit dem aktuellen Ergebnis überschreiben")

func assertGolden(t *testing.T, name string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	// UseNumber erhält auch große Centbeträge; JSON-Objektschlüssel werden sortiert.
	var canonical any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&canonical); err != nil {
		t.Fatal(err)
	}
	got, err := json.MarshalIndent(canonical, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	path := filepath.Join("testdata", "golden", name+".json")
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v; Golden-Datei mit -run TestGolden -update erzeugen", t.Name(), err)
	}
	if bytes.Equal(want, got) {
		return
	}
	oldLines, newLines := strings.Split(string(want), "\n"), strings.Split(string(got), "\n")
	for i := 0; i < max(len(oldLines), len(newLines)); i++ {
		oldLine, newLine := "<Dateiende>", "<Dateiende>"
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}
		if oldLine != newLine {
			start := max(0, i-3)
			t.Fatalf("%s: %s, erste Abweichung in Zeile %d (- erwartet, + erhalten):\n%s\n- %s\n+ %s", t.Name(), path, i+1, strings.Join(newLines[start:i], "\n"), oldLine, newLine)
		}
	}
}
