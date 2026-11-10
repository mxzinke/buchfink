package mcpserver

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/service"
)

type anlageFertigIn struct {
	KeyedWriteArgs
	AnlageID           uint       `json:"anlage_id"`
	Datum              Datum      `json:"datum"`
	Konto              Konto      `json:"konto"`
	Abschreibungskonto Konto      `json:"abschreibungskonto"`
	Methode            AfaMethode `json:"methode"`
	Nutzungsdauer      int        `json:"nutzungsdauer_monate"`
	Grund              string     `json:"grund"`
}
type anlageFertigOut struct {
	Wiederholung
	AnlageID uint `json:"anlage_id"`
}

func (out anlageFertigOut) Ref() string { return fmt.Sprintf("anlage:%d", out.AnlageID) }
func registerAssetTransfer(tb *Toolbox) {
	AddWrite(tb, Spec{Name: "anlage_fertigstellen", Title: "Anlage fertigstellen",
		Description: "Anlage im Bau auf das endgültige Konto umbuchen; AfA beginnt mit der Fertigstellung."},
		func(ctx context.Context, c *Call, s core.AgentServices, in anlageFertigIn) (anlageFertigOut, error) {
			out := anlageFertigOut{}
			date, err := in.Datum.check("datum")
			if err != nil {
				return out, err
			}
			method, err := pick("methode", in.Methode, depreciationMethods)
			if err != nil {
				return out, err
			}
			asset, err := s.Assets.Transfer(ctx, service.TransferRequest{
				AssetID: in.AnlageID, Date: date, Account: string(in.Konto),
				DepreciationAccount: string(in.Abschreibungskonto), Method: method,
				UsefulLifeMonths: in.Nutzungsdauer, Note: in.Grund,
			})
			if err == nil {
				out.AnlageID = asset.ID
			}
			return out, err
		})
}
