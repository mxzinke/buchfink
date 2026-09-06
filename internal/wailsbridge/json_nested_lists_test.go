package wailsbridge

import (
	"reflect"
	"strings"
	"testing"
)

// Auch die Listen *in* einer Antwort sind leer und nicht nil.
//
// Der Nachbar dieses Tests (json_lists_test.go) prüft die Rückgabe selbst: eine
// Bridge-Methode, die eine Liste liefert, liefert keine nil. Die zweite Hälfte
// des Problems steckt eine Ebene tiefer — der Anlagenspiegel ohne
// Anlagevermögen, die Startseite ohne Zahlungsverkehr, die Einstellungen ohne
// eingerichteten Dienst. Die Maske liest `spiegel.rows.map(…)` genauso ohne
// Umweg wie die äußere Liste, und `null.map` nimmt im Render den ganzen Baum
// mit.
//
// Geprüft wird am Lauf und nicht am Quelltext: ob ein Feld belegt wird, steht
// nicht in der Deklaration. Aufgerufen werden dafür die lesenden Methoden ohne
// Argumente auf einer frisch eingerichteten Bridge — also genau in der Lage, in
// der am meisten leer ist.
func TestNoBridgeAnswerCarriesANilListInside(t *testing.T) {
	b := testBridge(t)
	bridge := reflect.ValueOf(b)

	checked := 0
	for i := 0; i < bridge.NumMethod(); i++ {
		method := bridge.Type().Method(i)
		// Ohne Argumente, damit der Aufruf ohne erfundene Eingabe auskommt, und
		// nur lesend: eine Sicherung oder ein Export gehört nicht in einen Test
		// über die Form der Antwort.
		if method.Type.NumIn() != 1 || method.Type.NumOut() == 0 {
			continue
		}
		if !strings.HasPrefix(method.Name, "Get") && !strings.HasPrefix(method.Name, "Verify") {
			continue
		}
		checked++
		results := bridge.Method(i).Call(nil)
		var nils []string
		collectNilLists(results[0], method.Name, &nils, 0)
		for _, path := range nils {
			t.Errorf("%s liefert %s als nil — erwartet eine leere Liste (siehe emptyList und die "+
				"EnsureLists-Methoden der Antworttypen)", method.Name, path)
		}
	}
	if checked < 40 {
		t.Fatalf("nur %d lesende Bridge-Methoden aufgerufen — der Test findet sie offenbar nicht mehr",
			checked)
	}
}

// collectNilLists sammelt die Pfade der nicht belegten Listen einer Antwort.
//
// Die Tiefe ist begrenzt und die Zahl der geprüften Elemente einer Liste
// ebenfalls: geprüft wird die Form der Antwort, und dafür genügt der Anfang
// jeder Liste. Ein Zyklus über Zeiger liefe sonst bis zum Stapelende.
func collectNilLists(v reflect.Value, path string, out *[]string, depth int) {
	if !v.IsValid() || depth > 8 {
		return
	}
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if !v.IsNil() {
			collectNilLists(v.Elem(), path, out, depth+1)
		}
	case reflect.Slice:
		if v.IsNil() {
			*out = append(*out, path)
			return
		}
		for i := 0; i < v.Len() && i < 3; i++ {
			collectNilLists(v.Index(i), path+"[]", out, depth+1)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			if !field.IsExported() {
				continue
			}
			collectNilLists(v.Field(i), path+"."+field.Name, out, depth+1)
		}
	}
}
