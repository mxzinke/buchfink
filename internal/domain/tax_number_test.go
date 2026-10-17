package domain

import "testing"

// Die Beispiele je Land aus dem Handbuch zu myebilanz 2.3.1, Abschnitt 15.1.
func TestElsterTaxNumberPerState(t *testing.T) {
	cases := []struct{ state, local, want string }{
		{"BW", "93815/08152", "2893081508152"},
		{"BY", "181/815/08155", "9181081508155"},
		{"BE", "21/815/08150", "1121081508150"},
		{"BB", "048/815/08155", "3048081508155"},
		{"HB", "75 815 08152", "2475081508152"},
		{"HH", "02/815/08156", "2202081508156"},
		{"HE", "013 815 08153", "2613081508153"},
		{"MV", "079/815/08151", "4079081508151"},
		{"NI", "24/815/08151", "2324081508151"},
		{"NW", "133/8150/8159", "5133081508159"},
		{"RP", "22/815/0815/4", "2722081508154"},
		{"SL", "010/815/08182", "1010081508182"},
		{"SN", "201/123/12340", "3201012312340"},
		{"ST", "101/815/08154", "3101081508154"},
		{"SH", "29 815 08158", "2129081508158"},
		{"TH", "151/815/08156", "4151081508156"},
	}
	if len(cases) != len(taxNumberSchemes) {
		t.Fatalf("%d Beispiele für %d Länder", len(cases), len(taxNumberSchemes))
	}
	for _, c := range cases {
		got, err := ElsterTaxNumber(c.local, c.state)
		if err != nil {
			t.Errorf("%s %s: %v", c.state, c.local, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s %s = %s, erwartet %s", c.state, c.local, got, c.want)
		}
		// Die Nummer im Bundesschema bleibt, wie sie ist.
		again, err := ElsterTaxNumber(got, c.state)
		if err != nil || again != got {
			t.Errorf("%s: Bundesschema %s nicht erkannt (%s, %v)", c.state, got, again, err)
		}
	}
}

// Das Beispiel aus dem Handbuch: Finanzamt Nürnberg-Nord, BF4 9238.
func TestTaxOfficeNumber(t *testing.T) {
	elster, err := ElsterTaxNumber("238/123/45678", "BY")
	if err != nil || elster != "9238012345678" {
		t.Fatalf("Steuernummer = %s, %v", elster, err)
	}
	if got := TaxOfficeNumber(elster); got != "9238" {
		t.Errorf("Bundesfinanzamtsnummer = %s, erwartet 9238", got)
	}
}

func TestElsterTaxNumberRejectsWhatDoesNotFit(t *testing.T) {
	for _, c := range []struct{ state, local string }{
		{"BY", "21/815/08150"},  // zehn Ziffern, Bayern hat elf
		{"SN", "101/815/08154"}, // Sachsen beginnt mit 2
		{"BW", "9181081508155"}, // Bundesschema eines anderen Landes
		{"XX", "181/815/08155"}, // kein Land
		{"HE", "113 815 08153"}, // Hessen beginnt mit 0
	} {
		if got, err := ElsterTaxNumber(c.local, c.state); err == nil {
			t.Errorf("%s %s ergab %s statt eines Fehlers", c.state, c.local, got)
		}
	}
	// Hessen ohne führende Null.
	if got, err := ElsterTaxNumber("13 815 08153", "HE"); err != nil || got != "2613081508153" {
		t.Errorf("Hessen ohne führende 0 = %s, %v", got, err)
	}
	if got, err := ElsterTaxNumber("", "BY"); got != "" || err != nil {
		t.Errorf("leere Steuernummer = %q, %v", got, err)
	}
	if got, err := ElsterTaxNumber("181/815/08155", ""); got != "" || err != nil {
		t.Errorf("ohne Land = %q, %v", got, err)
	}
}
