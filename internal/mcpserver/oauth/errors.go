package oauth

import (
	"encoding/json"
	"net/http"
)

// oauthError ist eine Fehlerantwort nach RFC 6749 §5.2 bzw. RFC 7591 §3.2.2.
type oauthError struct {
	Code        string
	Description string
	Status      int
}

func (e *oauthError) Error() string { return e.Code + ": " + e.Description }

func errInvalidGrant(desc string) *oauthError {
	return &oauthError{Code: "invalid_grant", Description: desc, Status: http.StatusBadRequest}
}

func errInvalidRequest(desc string) *oauthError {
	return &oauthError{Code: "invalid_request", Description: desc, Status: http.StatusBadRequest}
}

func errInvalidClient(desc string) *oauthError {
	return &oauthError{Code: "invalid_client", Description: desc, Status: http.StatusUnauthorized}
}

func errInvalidScope(desc string) *oauthError {
	return &oauthError{Code: "invalid_scope", Description: desc, Status: http.StatusBadRequest}
}

func errInvalidTarget(desc string) *oauthError {
	return &oauthError{Code: "invalid_target", Description: desc, Status: http.StatusBadRequest}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeOAuthError(w http.ResponseWriter, err *oauthError) {
	writeJSON(w, err.Status, map[string]string{"error": err.Code, "error_description": err.Description})
}
