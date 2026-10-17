package mcpserver

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Fehlercodes der Werkzeuge. Sie sind stabil; Agenten verzweigen daran.
const (
	CodeNoTenant         = "kein_mandant_offen"
	CodeTenantNotGranted = "mandant_nicht_freigegeben"
	CodeTenantSwitched   = "mandant_gewechselt"
	CodeMissingScope     = "berechtigung_fehlt"
	CodeExaminer         = "pruefermodus_aktiv"
	CodeAppOnly          = "nur_in_app"
	CodeBusy             = "beschaeftigt"
	CodeKeyReused        = "schluessel_wiederverwendet"
	CodeInvalidInput     = "ungueltige_eingabe"
	CodeNotFound         = "nicht_gefunden"
	CodeRule             = "fachliche_regel"
	CodeInternal         = "intern"
)

// ToolError ist der einheitliche Fehler eines Werkzeugs. Werkzeug nennt das
// Werkzeug, mit dem es weitergeht, oder „nur_app".
type ToolError struct {
	Code             string `json:"code"`
	Nachricht        string `json:"nachricht"`
	NaechsterSchritt string `json:"naechster_schritt,omitempty"`
	Werkzeug         string `json:"werkzeug,omitempty"`
	Wiederholbar     bool   `json:"wiederholbar,omitempty"`
}

func (e *ToolError) Error() string { return e.Code + ": " + e.Nachricht }

// toolError ordnet einen Fehler aus Kern und Diensten einem Code zu. Was
// keinem bekannten Fehler entspricht, ist eine fachliche Regel: die Dienste
// formulieren ihre Meldungen für den Menschen und nennen den Ausweg meist
// selbst.
func toolError(err error) *ToolError {
	var te *ToolError
	switch {
	case errors.As(err, &te):
		return te
	case errors.Is(err, core.ErrNoTenant):
		return errNoTenant()
	case errors.Is(err, core.ErrTenantClosing):
		return &ToolError{Code: CodeTenantSwitched, Nachricht: err.Error(),
			NaechsterSchritt: "Mit status prüfen, welcher Mandant jetzt geöffnet ist.", Werkzeug: "status", Wiederholbar: true}
	case errors.Is(err, core.ErrExaminerActive):
		return &ToolError{Code: CodeExaminer, Nachricht: err.Error(),
			NaechsterSchritt: "Bis zum Ende des Prüfermodus nur lesen. Beenden lässt er sich nur in der App.", Werkzeug: "nur_app"}
	case errors.Is(err, core.ErrAppOnly):
		return &ToolError{Code: CodeAppOnly, Nachricht: err.Error(),
			NaechsterSchritt: "Den Menschen bitten, das in Buchfink selbst zu erledigen.", Werkzeug: "nur_app"}
	case errors.Is(err, context.DeadlineExceeded):
		return &ToolError{Code: CodeBusy, Nachricht: "Buchfink ist gerade mit einem anderen Vorgang beschäftigt.",
			NaechsterSchritt: "In einigen Sekunden mit demselben Idempotenzschlüssel erneut senden.", Wiederholbar: true}
	case errors.Is(err, domain.ErrIdempotencyKeyReused):
		return &ToolError{Code: CodeKeyReused, Nachricht: err.Error(),
			NaechsterSchritt: "Für einen neuen Vorgang einen neuen Idempotenzschlüssel wählen."}
	default:
		return &ToolError{Code: CodeRule, Nachricht: err.Error()}
	}
}

func errNoTenant() *ToolError {
	return &ToolError{Code: CodeNoTenant, Nachricht: "In Buchfink ist kein Mandant geöffnet.",
		NaechsterSchritt: "Den Menschen bitten, in Buchfink einen Mandanten zu öffnen; danach status aufrufen.", Werkzeug: "status"}
}

func errNotGranted() *ToolError {
	return &ToolError{Code: CodeTenantNotGranted,
		Nachricht:        "Diese App ist für den geöffneten Mandanten nicht freigegeben.",
		NaechsterSchritt: "Die Anmeldung im Client wiederholen und in Buchfink für diesen Mandanten zustimmen.", Werkzeug: "status"}
}

func errSwitched(expected, open string) *ToolError {
	return &ToolError{Code: CodeTenantSwitched,
		Nachricht:        "In Buchfink ist inzwischen ein anderer Mandant geöffnet (" + open + " statt " + expected + "). Es wurde nichts geschrieben.",
		NaechsterSchritt: "Mit status den geöffneten Mandanten prüfen und den Menschen fragen, ob dort gebucht werden soll.", Werkzeug: "status"}
}

func errScope(scope string) *ToolError {
	return &ToolError{Code: CodeMissingScope,
		Nachricht:        "Die Anmeldung dieser App erlaubt das nicht (fehlender Scope " + scope + ").",
		NaechsterSchritt: "Die Anmeldung im Client mit dem Scope " + scope + " wiederholen und in Buchfink zustimmen."}
}

// errorResult ist das Ergebnis eines gescheiterten Aufrufs. Der Fehler steht
// als JSON im Text; structuredContent bleibt leer, weil er sonst gegen das
// Ausgabeschema des Werkzeugs geprüft würde.
func errorResult(te *ToolError) *mcp.CallToolResult {
	body, _ := json.Marshal(map[string]*ToolError{"fehler": te})
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: string(body)}}}
}
