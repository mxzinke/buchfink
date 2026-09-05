// Package vatid fragt die Gültigkeit einer ausländischen USt-IdNr. beim
// Bundeszentralamt für Steuern ab (§ 18e UStG).
//
// Buchfink ist local-first: die Buchführung läuft ohne Netz. Diese Abfrage ist
// die eine Ausnahme, und sie ist es aus einem Grund — § 6a Abs. 1 Satz 1 Nr. 4
// UStG macht die gültige USt-IdNr. des Abnehmers zur materiellen Voraussetzung
// der Steuerbefreiung, und ohne Bestätigung trägt der Lieferer das Risiko. Der
// Aufruf blockiert deshalb das Ausstellen einer steuerfreien ig. Lieferung, aber
// nicht Buchfink: ist das Amt nicht erreichbar, sagt der Dienst genau das, und
// der Anwender entscheidet mit einer dokumentierten Übersteuerung.
package vatid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// DefaultEndpoint ist die Adresse der REST-Schnittstelle des Bundeszentralamts.
//
// Der Pfad gehört dazu und ist keine Nachlässigkeit: die Voreinstellung stand
// hier eine Zeit lang als Mischung aus dem Host der REST-Schnittstelle und dem
// Pfad der alten XML-RPC-Fassung, und mit ihr scheiterte jede Abfrage — eine
// steuerfreie ig. Lieferung wäre produktiv nur noch mit Übersteuerung
// ausstellbar gewesen.
//
// Stand: die XML-RPC-Schnittstelle unter https://evatr.bff-online.de/evatrRPC
// ist seit dem 30.11.2025 abgeschaltet; an ihre Stelle ist die REST-Fassung
// getreten (OpenAPI unter https://api.evatr.vies.bzst.de/api-docs, Server
// .../app, Abfrage POST /v1/abfrage). Die Adresse bleibt eine Einstellung, damit
// die nächste Umstellung keine neue Programmversion verlangt; welche Stelle
// geantwortet hat, steht an jedem Ergebnis.
const DefaultEndpoint = "https://api.evatr.vies.bzst.de/app/v1/abfrage"

// Timeout ist die Frist, nach der die Abfrage abgebrochen wird.
//
// Zehn Sekunden sind großzügig für einen Dienst, der eine Zeile zurückgibt, und
// kurz genug, dass niemand glaubt, das Programm hänge. Ohne Frist wäre das
// Ausstellen einer Rechnung von der Laune eines fremden Servers abhängig.
const Timeout = 10 * time.Second

// Request sind die Angaben der qualifizierten Bestätigungsanfrage.
//
// Die einfache Anfrage prüft nur, ob die Nummer gültig ist. Die qualifizierte
// prüft zusätzlich, ob Name, Ort, PLZ und Straße zu ihr gehören — und nur sie
// ist der Nachweis, den § 6a Abs. 4 UStG dem Lieferer abverlangt, der auf die
// Angaben des Abnehmers vertraut hat.
type Request struct {
	OwnVatID    string `json:"anfragendeUstid"`
	VatID       string `json:"angefragteUstid"`
	CompanyName string `json:"firmenname,omitempty"`
	City        string `json:"ort,omitempty"`
	PostalCode  string `json:"plz,omitempty"`
	Street      string `json:"strasse,omitempty"`
}

// Validate weist eine Anfrage zurück, die keine sein kann.
func (r Request) Validate() error {
	if strings.TrimSpace(r.OwnVatID) == "" {
		return fmt.Errorf(
			"für die Bestätigungsanfrage fehlt die eigene USt-IdNr. Sie steht in den " +
				"Unternehmensdaten und ist Teil der Anfrage — ohne sie beantwortet das " +
				"Bundeszentralamt sie nicht")
	}
	if strings.TrimSpace(r.VatID) == "" {
		return fmt.Errorf("die zu prüfende USt-IdNr. fehlt")
	}
	return nil
}

// Qualified meldet, ob die Anfrage die qualifizierte ist — also die, die auch
// die Adressangaben prüft.
func (r Request) Qualified() bool {
	return strings.TrimSpace(r.CompanyName) != "" && strings.TrimSpace(r.City) != ""
}

// Result ist die Antwort des Bundeszentralamts, in die Form gebracht, in der
// Buchfink sie aufhebt.
type Result struct {
	Status     domain.VatIDCheckStatus
	Code       string
	Text       string
	RequestID  string
	CheckedAt  time.Time
	Name       domain.VatIDFieldResult
	City       domain.VatIDFieldResult
	PostalCode domain.VatIDFieldResult
	Street     domain.VatIDFieldResult
	// Raw ist die Antwort, wie sie kam.
	Raw string
}

// response ist die Antwort der Schnittstelle.
//
// Die Felder der REST-Fassung stehen zuerst: `status` mit einer Kennung der Form
// „evatr-0000", `id` als technische Kennung der Anfrage, `anfrageZeitpunkt` und
// die Vergleichsergebnisse `ergFirmenname`, `ergStrasse`, `ergPlz`, `ergOrt`.
// Daneben werden die Namen der abgeschalteten XML-RPC-Fassung weiter gelesen —
// das kostet ein paar Zeilen, und es hält Buchfink lesefähig, wo ein Anwender
// die Adresse auf einen Zwischendienst gestellt hat, der noch das alte Format
// liefert. Eine Antwort, die Buchfink nicht versteht, ist „nicht erreichbar",
// und das kostet den Anwender im Zweifel seine Steuerbefreiung.
type response struct {
	Status     string      `json:"status"`
	ErrorCode  json.Number `json:"errorCode"`
	StatusText string      `json:"statusMeldung"`
	Message    string      `json:"errorMeldung"`

	RequestID    string `json:"id"`
	RequestIDAlt string `json:"anfrageId"`

	CheckedAt string `json:"anfrageZeitpunkt"`
	ValidFrom string `json:"gueltigAb"`
	ValidTo   string `json:"gueltigBis"`

	Name       string `json:"ergFirmenname"`
	City       string `json:"ergOrt"`
	PostalCode string `json:"ergPlz"`
	Street     string `json:"ergStrasse"`

	NameAlt   string `json:"ergName"`
	StreetAlt string `json:"ergStr"`
}

// Client fragt beim Bundeszentralamt an.
type Client struct {
	endpoint string
	http     *http.Client
	now      func() time.Time
}

// New erzeugt einen Client. Ein leerer Endpunkt bedeutet die Voreinstellung.
func New(endpoint string) *Client {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = DefaultEndpoint
	}
	return &Client{
		endpoint: endpoint,
		http:     &http.Client{Timeout: Timeout},
		now:      time.Now,
	}
}

// Endpoint liefert die Adresse, an die dieser Client fragt.
func (c *Client) Endpoint() string { return c.endpoint }

// SetClock ersetzt die Uhr. Nur für Tests: ein Ergebnis trägt seinen Zeitpunkt,
// und die Frist von 90 Tagen hängt an ihm.
func (c *Client) SetClock(now func() time.Time) { c.now = now }

// SetTimeout ändert die Frist, nach der die Abfrage abgebrochen wird.
//
// Für Tests: der Ausfall des Bundeszentralamts wird über einen Server geprüft,
// der nicht antwortet, und mit der Voreinstellung von zehn Sekunden wartete
// jeder solche Durchlauf zehn Sekunden real. Eine Frist ist nichts, was eine
// Prüfung in Echtzeit erleben muss.
func (c *Client) SetTimeout(d time.Duration) {
	if d <= 0 {
		return
	}
	c.http.Timeout = d
}

// Check führt die Bestätigungsanfrage aus.
//
// Ein Fehler heißt „keine Auskunft" und nie „ungültig". Die beiden zu
// verwechseln wäre der teuerste Fehler dieser Schnittstelle: eine Rechnung
// abzulehnen, weil das Amt gerade nicht antwortet, ist so falsch wie eine
// steuerfreie Lieferung an eine Nummer, die es nicht gibt.
func (c *Client) Check(ctx context.Context, req Request) (*Result, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("die Anfrage ließ sich nicht zusammenstellen: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("die Anfrage an %s ließ sich nicht aufbauen: %w", c.endpoint, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf(
			"das Bundeszentralamt für Steuern ist nicht erreichbar (%s): %w", c.endpoint, err)
	}
	defer resp.Body.Close()

	// Begrenzt gelesen: die Antwort ist eine Zeile, und ein Server, der
	// stattdessen einen Datenstrom schickt, soll nicht den Arbeitsspeicher
	// füllen.
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("die Antwort des Bundeszentralamts ließ sich nicht lesen: %w", err)
	}
	// Der HTTP-Status ist nicht das Ergebnis.
	//
	// Die REST-Fassung antwortet auf „die angefragte USt-IdNr. ist nicht
	// vergeben" (evatr-2001) mit HTTP 404 und auf eine syntaktisch falsche Nummer
	// mit HTTP 400 — beides sind Auskünfte über den Geschäftspartner und keine
	// Ausfälle. Wer sie am Statuscode abweist, verwandelt ein „ungültig" in ein
	// „nicht erreichbar" und lässt die steuerfreie Lieferung mit einer
	// Übersteuerung durch. Maßgeblich ist deshalb das Feld `status` im Rumpf;
	// erst wo keines kommt, entscheidet der HTTP-Status.
	var parsed response
	if err := json.Unmarshal(raw, &parsed); err != nil || resultCode(parsed) == "" {
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf(
				"das Bundeszentralamt hat mit HTTP %d geantwortet: %s",
				resp.StatusCode, strings.TrimSpace(string(raw)))
		}
		if err != nil {
			return nil, fmt.Errorf(
				"die Antwort des Bundeszentralamts ist nicht lesbar. Stimmt die eingestellte Adresse "+
					"(%s)? %w", c.endpoint, err)
		}
	}
	return c.toResult(parsed, string(raw))
}

// resultCode liest den Ergebniscode aus der Antwort — die Kennung der
// REST-Fassung, sonst die Zahl der alten.
func resultCode(parsed response) string {
	if code := strings.TrimSpace(parsed.Status); code != "" {
		return code
	}
	return strings.TrimSpace(parsed.ErrorCode.String())
}

func (c *Client) toResult(parsed response, raw string) (*Result, error) {
	code := resultCode(parsed)
	if code == "" {
		return nil, fmt.Errorf(
			"die Antwort des Bundeszentralamts nennt keinen Ergebniscode. Ohne ihn ist sie kein " +
				"Nachweis und wird nicht aufgehoben")
	}

	out := &Result{
		Code:       code,
		Text:       firstNonEmpty(parsed.StatusText, parsed.Message),
		RequestID:  firstNonEmpty(parsed.RequestID, parsed.RequestIDAlt),
		CheckedAt:  c.now().UTC(),
		Name:       fieldResult(firstNonEmpty(parsed.Name, parsed.NameAlt)),
		City:       fieldResult(parsed.City),
		PostalCode: fieldResult(parsed.PostalCode),
		Street:     fieldResult(firstNonEmpty(parsed.Street, parsed.StreetAlt)),
		Raw:        raw,
	}
	// Der Gültigkeitszeitraum gehört in den Wortlaut: die Codes evatr-2002 und
	// evatr-2006 sagen „nicht zum Anfragezeitpunkt gültig" und nennen die Grenzen
	// in eigenen Feldern. Ohne sie stünde am Kontakt ein „ungültig", dem niemand
	// ansieht, dass die Nummer nächsten Monat gilt.
	if parsed.ValidFrom != "" || parsed.ValidTo != "" {
		out.Text = strings.TrimSpace(fmt.Sprintf(
			"%s (gültig von %s bis %s)", out.Text,
			orDash(parsed.ValidFrom), orDash(parsed.ValidTo)))
	}
	out.Status = statusFor(code)
	if out.Status == domain.VatIDUnavailable {
		return out, fmt.Errorf(
			"das Bundeszentralamt hat die Anfrage nicht beantwortet (Code %s%s)",
			code, suffix(out.Text))
	}
	return out, nil
}

// Die Ergebniscodes der REST-Fassung, in drei Gruppen.
//
// Die Einteilung ist die Entscheidung dieser Datei und nicht die des Amtes: das
// Amt kennt „Ergebnis", „Hinweis" und „Fehler", Buchfink braucht „gültig",
// „ungültig" und „keine Auskunft". Der Unterschied zwischen den letzten beiden
// ist der teuerste dieser Schnittstelle — eine Rechnung abzulehnen, weil ein
// fremder Server nicht antwortet, ist so falsch wie eine steuerfreie Lieferung
// an eine Nummer, die es nicht gibt.
var (
	// evatrValid: die angefragte Nummer ist zum Anfragezeitpunkt gültig.
	// evatr-2008 ist es ebenfalls, nur mit einer Besonderheit bei der
	// qualifizierten Abfrage; evatr-0003 bestätigt die Gültigkeit, sagt aber,
	// dass für die qualifizierte Anfrage Angaben fehlten.
	evatrValid = map[string]bool{
		"evatr-0000": true, "evatr-0003": true, "evatr-2008": true,
	}
	// evatrInvalid: eine Auskunft über die angefragte Nummer, und zwar eine
	// negative — nicht vergeben, (noch/nicht mehr) nicht gültig, syntaktisch
	// falsch oder mit einem Länderkennzeichen, das es nicht gibt.
	evatrInvalid = map[string]bool{
		"evatr-2001": true, "evatr-2002": true, "evatr-2006": true,
		"evatr-0005": true, "evatr-0012": true, "evatr-2003": true,
	}
)

// statusFor ordnet den Ergebniscode ein.
//
// Die alte XML-RPC-Fassung antwortete mit Zahlen: 200 ist die einzige, die
// „gültig" bedeutet, 201 bis 223 weisen zurück. Sie wird weiter gelesen, weil
// die Adresse einstellbar ist und ein Zwischendienst noch so antworten kann.
//
// Alles, was in keine der beiden Gruppen fällt — Serverfehler, überschrittene
// Abfragezahl, eine ungültige eigene USt-IdNr. —, ist keine Aussage über den
// Geschäftspartner und deshalb „keine Auskunft".
func statusFor(code string) domain.VatIDCheckStatus {
	if n, err := strconv.Atoi(code); err == nil {
		switch {
		case n == 200:
			return domain.VatIDValid
		case n >= 201 && n <= 223:
			return domain.VatIDInvalid
		default:
			return domain.VatIDUnavailable
		}
	}
	normalized := strings.ToLower(strings.TrimSpace(code))
	switch {
	case evatrValid[normalized]:
		return domain.VatIDValid
	case evatrInvalid[normalized]:
		return domain.VatIDInvalid
	default:
		return domain.VatIDUnavailable
	}
}

// orDash macht aus einer fehlenden Angabe einen Strich statt einer Lücke.
func orDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return strings.TrimSpace(value)
}

func fieldResult(value string) domain.VatIDFieldResult {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "A":
		return domain.VatIDFieldMatch
	case "B":
		return domain.VatIDFieldMismatch
	case "C":
		return domain.VatIDFieldNotAsked
	case "D":
		return domain.VatIDFieldUnknown
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func suffix(text string) string {
	if text == "" {
		return ""
	}
	return ": " + text
}
