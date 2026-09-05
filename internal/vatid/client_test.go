package vatid

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Ergebniscode entscheidet über alles Weitere, und die Grenzen sind eng.
//
// In der REST-Fassung bestätigen evatr-0000 und evatr-2008 die Nummer,
// evatr-2001, -2002 und -2006 weisen sie zurück, und alles Übrige — Serverfehler,
// eine ungültige eigene Nummer, eine überschrittene Abfragezahl — ist ein
// Ausfall der Übermittlung. Einen Ausfall als „ungültig" zu lesen hieße, aus
// einem Serverproblem eine Aussage über den Geschäftspartner zu gewinnen und die
// Rechnung an ihm scheitern zu lassen; ein „ungültig" als Ausfall zu lesen
// hieße, eine steuerfreie Lieferung an eine Nummer zu stellen, die es nicht
// gibt. Die Zahlen der abgeschalteten XML-RPC-Fassung werden weiter gelesen.
func TestStatusForResultCodes(t *testing.T) {
	tests := []struct {
		code string
		want domain.VatIDCheckStatus
	}{
		{"200", domain.VatIDValid},
		{"201", domain.VatIDInvalid},
		{"217", domain.VatIDInvalid},
		{"223", domain.VatIDInvalid},
		{"224", domain.VatIDUnavailable},
		{"999", domain.VatIDUnavailable},
		{"evatr-0000", domain.VatIDValid},
		{"evatr-2008", domain.VatIDValid},
		{"evatr-2001", domain.VatIDInvalid},
		{"evatr-2002", domain.VatIDInvalid},
		{"evatr-2006", domain.VatIDInvalid},
		{"evatr-0012", domain.VatIDInvalid},
		// Die eigene Nummer ist falsch, das Amt überlastet, die Abfragezahl
		// erschöpft: über den Geschäftspartner sagt das nichts.
		{"evatr-0004", domain.VatIDUnavailable},
		{"evatr-0008", domain.VatIDUnavailable},
		{"evatr-1004", domain.VatIDUnavailable},
		{"evatr-2005", domain.VatIDUnavailable},
		{"unbekannt", domain.VatIDUnavailable},
	}
	for _, tc := range tests {
		if got := statusFor(tc.code); got != tc.want {
			t.Errorf("Code %q ergibt %q — erwartet %q", tc.code, got, tc.want)
		}
	}
}

// Die Antwort der REST-Fassung, so wie sie das Bundeszentralamt schickt:
// `status`, `id`, `anfrageZeitpunkt` und die Vergleichsergebnisse
// `ergFirmenname`, `ergStrasse`, `ergPlz`, `ergOrt`.
func TestCheckReadsTheRestResponse(t *testing.T) {
	var got Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Methode %q — die Abfrage ist ein POST", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"evatr-0000","id":"f1c2-4711",` +
			`"anfrageZeitpunkt":"2026-03-10T09:12:00Z",` +
			`"ergFirmenname":"A","ergOrt":"A","ergPlz":"B","ergStrasse":"D"}`))
	}))
	defer server.Close()

	result, err := New(server.URL).Check(context.Background(), Request{
		OwnVatID: "DE123456789", VatID: "ATU12345678",
		CompanyName: "Musterhaus GmbH", City: "Wien", PostalCode: "1010", Street: "Ringstraße 1",
	})
	if err != nil {
		t.Fatalf("Abfrage: %v", err)
	}
	// Die Feldnamen der Anfrage sind die der Schnittstelle; ein Tippfehler darin
	// beantwortet das Amt mit „Pflichtfeld nicht besetzt".
	if got.OwnVatID != "DE123456789" || got.VatID != "ATU12345678" || got.City != "Wien" {
		t.Errorf("übermittelte Anfrage %+v — erwartet anfragendeUstid/angefragteUstid/ort", got)
	}
	if result.Status != domain.VatIDValid || result.Code != "evatr-0000" {
		t.Errorf("Ergebnis %q (Code %q) — erwartet gültig", result.Status, result.Code)
	}
	if result.RequestID != "f1c2-4711" {
		t.Errorf("Abfrage-Nr. %q — sie ist der Nachweis", result.RequestID)
	}
	if result.Name != domain.VatIDFieldMatch || result.City != domain.VatIDFieldMatch ||
		result.PostalCode != domain.VatIDFieldMismatch || result.Street != domain.VatIDFieldUnknown {
		t.Errorf("Feldergebnisse %q/%q/%q/%q — erwartet A/A/B/D",
			result.Name, result.City, result.PostalCode, result.Street)
	}
}

// Die REST-Fassung antwortet auf „diese Nummer ist nicht vergeben" mit HTTP 404
// und einem Rumpf, der das Ergebnis nennt. Wer am Statuscode abbricht, macht aus
// einer Auskunft einen Ausfall — und lässt die steuerfreie Lieferung mit einer
// Übersteuerung durch.
func TestCheckReadsAnInvalidResultBehindAnErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":"evatr-2001","anfrageZeitpunkt":"2026-03-10T09:12:00Z"}`))
	}))
	defer server.Close()

	result, err := New(server.URL).Check(context.Background(), Request{
		OwnVatID: "DE123456789", VatID: "ATU99999999",
	})
	if err != nil {
		t.Fatalf("eine beantwortete Anfrage ist kein Fehler: %v", err)
	}
	if result.Status != domain.VatIDInvalid {
		t.Errorf("Ergebnis %q — „nicht vergeben\" ist ungültig und nicht „nicht erreichbar\"",
			result.Status)
	}
	if result.Code != "evatr-2001" {
		t.Errorf("Code %q — er ist der Nachweis", result.Code)
	}
}

// Ein Serverfehler ohne lesbaren Rumpf bleibt ein Ausfall.
func TestCheckTreatsAServerErrorAsNoAnswer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>Gateway Timeout</html>"))
	}))
	defer server.Close()

	if _, err := New(server.URL).Check(context.Background(), Request{
		OwnVatID: "DE123456789", VatID: "ATU12345678",
	}); err == nil {
		t.Error("eine Antwort ohne Ergebniscode ist keine Auskunft")
	}
}

// Die qualifizierte Anfrage braucht beide Nummern; ohne die eigene beantwortet
// das Bundeszentralamt sie nicht.
func TestRequestValidation(t *testing.T) {
	if err := (Request{VatID: "ATU12345678"}).Validate(); err == nil {
		t.Error("ohne die eigene USt-IdNr. ist die Anfrage keine")
	}
	if err := (Request{OwnVatID: "DE123456789"}).Validate(); err == nil {
		t.Error("ohne die zu prüfende Nummer ist die Anfrage keine")
	}
	full := Request{
		OwnVatID: "DE123456789", VatID: "ATU12345678",
		CompanyName: "Kunde GmbH", City: "Wien",
	}
	if err := full.Validate(); err != nil {
		t.Errorf("die vollständige Anfrage ist gültig: %v", err)
	}
	if !full.Qualified() {
		t.Error("mit Firmenname und Ort ist die Anfrage die qualifizierte")
	}
	if (Request{OwnVatID: "DE1", VatID: "AT1"}).Qualified() {
		t.Error("ohne Adressangaben ist es die einfache Anfrage")
	}
}

// Die Feldergebnisse der qualifizierten Anfrage werden übernommen, wie sie
// kommen — der Unterschied zwischen B („stimmt nicht überein") und D („vom
// Mitgliedstaat nicht mitgeteilt") entscheidet darüber, ob eine Rechnung ein
// Problem hat.
func TestCheckReadsFieldResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errorCode":"200","anfrageId":"2026-4711",` +
			`"ergName":"A","ergOrt":"B","ergPlz":"C","ergStr":"D"}`))
	}))
	defer server.Close()

	result, err := New(server.URL).Check(context.Background(), Request{
		OwnVatID: "DE123456789", VatID: "ATU12345678",
		CompanyName: "Kunde GmbH", City: "Wien",
	})
	if err != nil {
		t.Fatalf("Abfrage: %v", err)
	}
	if result.Status != domain.VatIDValid {
		t.Errorf("Ergebnis %q — erwartet gültig", result.Status)
	}
	if result.RequestID != "2026-4711" {
		t.Errorf("Abfrage-Nr. %q — sie ist der Nachweis", result.RequestID)
	}
	if result.Name != domain.VatIDFieldMatch || result.City != domain.VatIDFieldMismatch ||
		result.PostalCode != domain.VatIDFieldNotAsked || result.Street != domain.VatIDFieldUnknown {
		t.Errorf("Feldergebnisse %q/%q/%q/%q — erwartet A/B/C/D",
			result.Name, result.City, result.PostalCode, result.Street)
	}
	if result.Raw == "" {
		t.Error("die Antwort wird aufgehoben, wie sie kam")
	}
}

// Eine Antwort ohne Ergebniscode ist kein Nachweis und wird nicht aufgehoben.
func TestCheckRefusesAResponseWithoutACode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"statusMeldung":"irgendetwas"}`))
	}))
	defer server.Close()

	if _, err := New(server.URL).Check(context.Background(), Request{
		OwnVatID: "DE123456789", VatID: "ATU12345678",
	}); err == nil {
		t.Error("ohne Ergebniscode ist die Antwort kein Nachweis")
	}
}

// Ein leerer Endpunkt bedeutet die Voreinstellung — die Adresse ist eine
// Einstellung, damit ein Wechsel keine Programmversion verlangt.
//
// Die Voreinstellung muss die vollständige Adresse der REST-Schnittstelle sein,
// Pfad eingeschlossen. Sie stand eine Zeit lang als Host der REST-Fassung mit
// dem Pfad der alten XML-RPC-Fassung; damit scheiterte jede Abfrage, und eine
// steuerfreie ig. Lieferung ließ sich produktiv nur mit Übersteuerung ausstellen.
func TestDefaultEndpoint(t *testing.T) {
	if got := New("").Endpoint(); got != DefaultEndpoint {
		t.Errorf("Endpunkt %q — erwartet die Voreinstellung %q", got, DefaultEndpoint)
	}
	if !strings.HasPrefix(DefaultEndpoint, "https://") {
		t.Errorf("Voreinstellung %q — die Abfrage geht verschlüsselt", DefaultEndpoint)
	}
	if strings.HasSuffix(DefaultEndpoint, "/evatrRPC") {
		t.Errorf("Voreinstellung %q — das ist der Pfad der abgeschalteten XML-RPC-Fassung",
			DefaultEndpoint)
	}
	if !strings.Contains(DefaultEndpoint, "/v1/abfrage") {
		t.Errorf("Voreinstellung %q — die REST-Abfrage liegt unter /v1/abfrage", DefaultEndpoint)
	}
	if got := New("https://example.test/evatr").Endpoint(); got != "https://example.test/evatr" {
		t.Errorf("Endpunkt %q — die Einstellung muss durchschlagen", got)
	}
}
