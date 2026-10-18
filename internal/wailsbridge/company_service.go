package wailsbridge

import (
	"context"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Darlehensverträge mit Gesellschaftern, die Steuernummer im Bundesschema und
// die Anleitungen zum Verbinden einer KI-App.

// GetShareholderLoans liefert die Darlehensverträge mit Gesellschaftern und
// ihrem gebuchten Stand von heute.
func (b *BuchfinkBridge) GetShareholderLoans() ([]service.LoanView, error) {
	today := time.Now().Format(domain.DateLayout)
	return readList(b, func(ctx context.Context, s *core.Services) ([]service.LoanView, error) {
		return s.Shareholders.Loans(ctx, today)
	})
}

// RegisterShareholderLoan erfasst einen Darlehensvertrag. Er steuert
// Zinsbuchungen und Fremdvergleich.
func (b *BuchfinkBridge) RegisterShareholderLoan(req service.LoanRequest) (*service.LoanView, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*service.LoanView, error) {
		return s.Shareholders.RegisterLoan(ctx, req)
	})
}

// CorrectShareholderLoan berichtigt einen Darlehensvertrag mit Grund.
func (b *BuchfinkBridge) CorrectShareholderLoan(id uint, req service.LoanRequest, reason string) (*service.LoanView, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*service.LoanView, error) {
		return s.Shareholders.CorrectLoan(ctx, id, req, reason)
	})
}

// FormatElsterTaxNumber rechnet eine Steuernummer in das 13-stellige
// Bundesschema um, das ELSTER erwartet. Die Stammdaten zeigen es schon beim
// Eintippen; gespeichert wird es nicht.
func (b *BuchfinkBridge) FormatElsterTaxNumber(taxNumber, federalState string) (string, error) {
	return domain.ElsterTaxNumber(taxNumber, federalState)
}

// agentGuideBase ist der Ordner der Anleitungen je KI-App im Repository.
const agentGuideBase = "https://github.com/mxzinke/buchfink/blob/main/docs/nutzung/ki-agenten/"

// agentGuides sind die Anleitungen, die die App öffnen darf. Eine feste
// Liste: die Oberfläche nennt nur den Schlüssel, nie eine Adresse.
var agentGuides = map[string]string{
	"":               "README.md",
	"claude-code":    "claude-code.md",
	"claude-desktop": "claude-desktop.md",
	"opencode":       "opencode.md",
	"codex":          "codex.md",
	"pi":             "pi.md",
}

// agentGuideURL ist die Adresse der Anleitung zu einer KI-App.
func agentGuideURL(client string) (string, error) {
	file, ok := agentGuides[client]
	if !ok {
		return "", fmt.Errorf("zu %q gibt es keine Anleitung", client)
	}
	return agentGuideBase + file, nil
}

// OpenAgentGuide öffnet die Anleitung zu einer KI-App im Browser.
func (b *BuchfinkBridge) OpenAgentGuide(client string) error {
	url, err := agentGuideURL(client)
	if err != nil {
		return err
	}
	app := application.Get()
	if app == nil || app.Browser == nil {
		return fmt.Errorf("der Browser lässt sich von hier aus nicht öffnen")
	}
	return app.Browser.OpenURL(url)
}
