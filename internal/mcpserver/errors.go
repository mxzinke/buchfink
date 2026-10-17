package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/core/policy"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"
)

// Fehlercodes der Werkzeuge. Sie sind stabil; Agenten verzweigen daran.
const (
	CodeNoTenant         = "kein_mandant_offen"
	CodeTenantNotGranted = "mandant_nicht_freigegeben"
	CodeTenantSwitched   = "mandant_gewechselt"
	CodeMissingScope     = "berechtigung_fehlt"
	CodeExaminer         = "pruefermodus_aktiv"
	CodeAppOnly          = "nur_in_app"
	CodeApprovalNeeded   = "freigabe_noetig"
	CodePathNotAllowed   = "pfad_nicht_erlaubt"
	CodeFileInvalid      = "datei_ungueltig"
	CodeBusy             = "beschaeftigt"
	CodeKeyReused        = "schluessel_wiederverwendet"
	CodeInvalidInput     = "ungueltige_eingabe"
	CodeNotFound         = "nicht_gefunden"
	CodeRule             = "fachliche_regel"
	CodeInternal         = "intern"
)

// ToolError ist der einheitliche Fehler eines Werkzeugs. Werkzeug nennt das
// Werkzeug, mit dem es weitergeht, oder „nur_app". Gruende erklären
// freigabe_noetig.
type ToolError struct {
	Code             string          `json:"code"`
	Nachricht        string          `json:"nachricht"`
	NaechsterSchritt string          `json:"naechster_schritt,omitempty"`
	Werkzeug         string          `json:"werkzeug,omitempty"`
	Gruende          []policy.Reason `json:"gruende,omitempty"`
	Wiederholbar     bool            `json:"wiederholbar,omitempty"`
}

func (e *ToolError) Error() string { return e.Code + ": " + e.Nachricht }

// toolError ordnet einen Fehler aus Kern und Diensten einem Code zu. Was
// keinem bekannten Fehler entspricht, ist eine fachliche Regel: die Dienste
// formulieren ihre Meldungen für den Menschen und nennen den Ausweg meist
// selbst.
func toolError(err error) *ToolError {
	var te *ToolError
	var fe *policy.FileError
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
	case errors.As(err, &fe) && fe.NotAllowed:
		return &ToolError{Code: CodePathNotAllowed, Nachricht: err.Error(),
			NaechsterSchritt: "Den Menschen bitten, die Datei in einen Belegordner zu legen (status nennt sie) oder ins Buchfink-Fenster zu ziehen.",
			Werkzeug:         "status"}
	case errors.As(err, &fe):
		return &ToolError{Code: CodeFileInvalid, Nachricht: err.Error(),
			NaechsterSchritt: "Die Originaldatei (PDF, Bild oder XML bis 50 MB) anfordern."}
	case errors.Is(err, gorm.ErrRecordNotFound):
		return &ToolError{Code: CodeNotFound, Nachricht: "Eine der angegebenen Kennungen gibt es nicht: " + err.Error(),
			NaechsterSchritt: "Die Kennungen mit dem passenden Such-Werkzeug neu ermitteln."}
	case errors.Is(err, context.DeadlineExceeded):
		return &ToolError{Code: CodeBusy, Nachricht: "Buchfink ist gerade mit einem anderen Vorgang beschäftigt.",
			NaechsterSchritt: "In einigen Sekunden mit demselben Idempotenzschlüssel erneut senden.", Wiederholbar: true}
	case errors.Is(err, core.ErrOffline), errors.Is(err, core.ErrNetworkInWrite):
		return &ToolError{Code: CodeRule, Nachricht: "Für den Vorgang braucht Buchfink das Netz (etwa einen Wechselkurs); es ist nicht erreichbar: " + err.Error(),
			NaechsterSchritt: "Später erneut senden.", Wiederholbar: true}
	case errors.Is(err, core.ErrRatesInExaminerMode):
		return &ToolError{Code: CodeExaminer, Nachricht: err.Error(),
			NaechsterSchritt: "Bis zum Ende des Prüfermodus nur lesen.", Werkzeug: "nur_app"}
	case errors.Is(err, domain.ErrInvalidAmount), errors.Is(err, domain.ErrFiscalYearRequired),
		errors.Is(err, service.ErrDeclarationKind):
		return &ToolError{Code: CodeInvalidInput, Nachricht: err.Error()}
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
		Nachricht:        "Diese App hat im geöffneten Mandanten das Recht " + scope + " nicht (Anmeldung oder Zulassung).",
		NaechsterSchritt: "Die Anmeldung im Client mit dem Scope " + scope + " wiederholen und in Buchfink für diesen Mandanten zustimmen."}
}

// errApproval ist die Antwort eines direkten Werkzeugs auf einen Vorgang, den
// Buchfink nicht als Routine einstuft. Es ist nichts geschrieben.
func errApproval(v policy.Verdict, confirmTool string) *ToolError {
	return &ToolError{Code: CodeApprovalNeeded,
		Nachricht:        "Kein Routinefall; es wurde nichts geschrieben.",
		NaechsterSchritt: "Dem Menschen Vorgang und Gründe vorlegen; nach seinem Ja dieselben Angaben mit " + confirmTool + " senden.",
		Werkzeug:         confirmTool, Gruende: v.Reasons}
}

// searchTool nennt das Werkzeug, mit dem sich eine Kennung neu ermitteln lässt.
var searchTool = map[string]string{
	"beleg_id": "belege_suchen", "kontakt_id": "kontakte_suchen", "buchung_id": "buchungen_suchen",
	"bankumsatz_id": "bankumsaetze", "anlage_id": "anlagen",
}

// notFound benennt bei einer unbekannten Kennung das Feld; andere Fehler
// gehen unverändert durch.
func notFound(field string, id uint, err error) error {
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	tool := searchTool[field]
	return &ToolError{Code: CodeNotFound, Nachricht: fmt.Sprintf("%s %d gibt es nicht", field, id),
		NaechsterSchritt: "Die Kennung mit " + tool + " neu ermitteln.", Werkzeug: tool}
}

// errorResult ist das Ergebnis eines gescheiterten Aufrufs. Der Fehler steht
// als JSON im Text; structuredContent bleibt leer, weil er sonst gegen das
// Ausgabeschema des Werkzeugs geprüft würde.
func errorResult(te *ToolError) *mcp.CallToolResult {
	body, _ := json.Marshal(map[string]*ToolError{"fehler": te})
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: string(body)}}}
}
