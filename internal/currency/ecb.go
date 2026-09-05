// Package currency holt die Referenzkurse der Europäischen Zentralbank.
//
// Der Dienst ist der einzige Ort, an dem Buchfink einen Kurs von außen bezieht,
// und er hat genau eine Regel: er rät nie. Die frühere Fassung lieferte bei
// einem Netzfehler still 1,0 zurück — ein Dollarbetrag wurde damit als
// Eurobetrag gebucht, in richtiger Größenordnung und mit falschem Wert, und
// nichts an der Buchung sah danach aus. Ein Fehler, den der Anwender sieht, ist
// jeder stillen Zahl vorzuziehen.
package currency

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// DefaultEndpoint ist die Adresse des Kursdienstes.
//
// frankfurter.app gibt die Referenzkurse der EZB unverändert weiter und braucht
// keinen Schlüssel. Die Adresse ist eine Einstellung, damit ein Wechsel des
// Anbieters keine Programmänderung ist.
const DefaultEndpoint = "https://api.frankfurter.app"

// Timeout ist die Frist einer Kursabfrage.
const Timeout = 10 * time.Second

// fxResponse ist die Antwort des Kursdienstes. Date ist der Tag, für den der
// Kurs tatsächlich festgestellt wurde — an einem Wochenende ist das der
// vorangegangene Handelstag.
type fxResponse struct {
	Amount float64            `json:"amount"`
	Base   string             `json:"base"`
	Date   string             `json:"date"`
	Rates  map[string]float64 `json:"rates"`
}

// Service holt Tageskurse.
type Service struct {
	endpoint string
	http     *http.Client
}

// New erzeugt den Dienst. Ein leerer Endpunkt bedeutet die Voreinstellung.
func New(endpoint string) *Service {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = DefaultEndpoint
	}
	return &Service{
		endpoint: strings.TrimRight(endpoint, "/"),
		http:     &http.Client{Timeout: Timeout},
	}
}

// Endpoint liefert die Adresse, die dieser Dienst abfragt.
func (s *Service) Endpoint() string { return s.endpoint }

// RateAt liefert den EZB-Referenzkurs eines Tages.
//
// Zurück kommen der Kurs in Millionsteln, der Tag, für den er festgestellt
// wurde, und die Quelle. Bei jedem Fehler kommt ein Fehler — kein Kurs, kein
// Rückfallwert, keine zwischengespeicherte Zahl von gestern.
func (s *Service) RateAt(ctx context.Context, currency, date string) (int64, string, string, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" || currency == "EUR" {
		return 0, "", "", fmt.Errorf(
			"der Euro ist die Buchwährung; für ihn wird kein Kurs geholt")
	}
	if len(date) != 10 {
		return 0, "", "", fmt.Errorf("das Kursdatum fehlt oder ist unvollständig (erwartet JJJJ-MM-TT)")
	}

	target := fmt.Sprintf("%s/%s?from=EUR&to=%s", s.endpoint, url.PathEscape(date), url.QueryEscape(currency))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return 0, "", "", fmt.Errorf("die Kursabfrage ließ sich nicht aufbauen: %w", err)
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return 0, "", "", fmt.Errorf(
			"der Kursdienst ist nicht erreichbar (%s). Trage den Kurs mit seiner Quelle von Hand ein — "+
				"Buchfink rät keinen: %w", s.endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, "", "", fmt.Errorf("die Antwort des Kursdienstes ließ sich nicht lesen: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, "", "", fmt.Errorf(
			"der Kursdienst hat mit HTTP %d geantwortet: %s",
			resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var fx fxResponse
	if err := json.Unmarshal(body, &fx); err != nil {
		return 0, "", "", fmt.Errorf(
			"die Antwort des Kursdienstes ist nicht lesbar. Stimmt die eingestellte Adresse (%s)? %w",
			s.endpoint, err)
	}
	rate, ok := fx.Rates[currency]
	if !ok || rate <= 0 {
		return 0, "", "", fmt.Errorf(
			"der Kursdienst kennt für den %s keinen Kurs von EUR nach %s", date, currency)
	}

	actual := fx.Date
	if actual == "" {
		actual = date
	}
	source := fmt.Sprintf("EZB-Referenzkurs vom %s (%s)", actual, s.endpoint)
	// Millionstel, kaufmännisch gerundet: der Kurs wird als ganze Zahl geführt,
	// damit jede spätere Umrechnung dieselbe Zahl liefert.
	micros := int64(rate*float64(domain.RateScale) + 0.5)
	if micros <= 0 {
		return 0, "", "", fmt.Errorf("der gelieferte Kurs %v ergibt keinen brauchbaren Wert", rate)
	}
	return micros, actual, source, nil
}
