package mcpserver

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

type anlageAbgangIn struct {
	BewegungID   uint `json:"bewegung_id,omitempty"`
	PoolZugangID uint `json:"pool_zugang_id,omitempty"`
	KeyedWriteArgs
	AnlageID     uint         `json:"anlage_id"`
	Datum        Datum        `json:"datum,omitempty"`
	Art          string       `json:"art" jsonschema:"verkauf, verschrottung oder storno"`
	Erloes       Betrag       `json:"erloes,omitempty" jsonschema:"Nettoerlös"`
	Stueckzahl   domain.Units `json:"stueckzahl,omitempty" jsonschema:"Zehntausendstel Stück; ohne Angabe Vollabgang"`
	KontaktID    uint         `json:"kontakt_id,omitempty"`
	Bankkonto    string       `json:"bankkonto,omitempty"`
	BelegID      *uint        `json:"beleg_id,omitempty"`
	BankumsatzID uint         `json:"bankumsatz_id,omitempty"`
	Grund        string       `json:"grund"`
	NurVorschau  bool         `json:"nur_vorschau,omitempty"`
}

func (in anlageAbgangIn) preview() bool { return in.NurVorschau }

type anlageAbgangOut struct {
	Wiederholung
	Vorschau *service.DisposalPreview `json:"vorschau,omitempty"`
	Ergebnis *service.DisposalResult  `json:"ergebnis,omitempty"`
}

func (out anlageAbgangOut) Ref() string {
	if out.Ergebnis == nil {
		return ""
	}
	return fmt.Sprintf("anlage:%d", out.Ergebnis.Asset.ID)
}

func registerAssetDisposal(tb *Toolbox) {
	AddWrite(tb, Spec{Name: "anlage_abgehen", Title: "Anlage abgehen lassen",
		Description: "Verkauf oder Verschrottung mit zeitanteiliger AfA. Erst Vorschau vorlegen, nach dem Ja buchen."},
		func(ctx context.Context, c *Call, s core.AgentServices, in anlageAbgangIn) (anlageAbgangOut, error) {
			out := anlageAbgangOut{}
			if in.Art == "storno" {
				if in.NurVorschau {
					return out, fmt.Errorf("Storno mit Grund ohne nur_vorschau aufrufen")
				}
				asset, err := s.Assets.ReverseDisposal(ctx, in.AnlageID, in.BewegungID, in.Grund)
				if err == nil {
					out.Ergebnis = &service.DisposalResult{Asset: *asset, Message: "Abgang zurückgenommen"}
				}
				return out, err
			}
			date, err := in.Datum.check("datum")
			if err != nil {
				return out, err
			}
			proceeds, err := in.Erloes.cents("erloes")
			if err != nil {
				return out, err
			}
			kind := domain.DisposalSale
			switch in.Art {
			case "verkauf":
			case "verschrottung":
				kind = domain.DisposalScrapped
			default:
				return out, fmt.Errorf("art: verkauf, verschrottung oder storno angeben")
			}
			req := service.DisposalRequest{PoolItemMovementID: in.PoolZugangID, ReceiptID: in.BelegID, BankTxID: in.BankumsatzID, AssetID: in.AnlageID, Date: date, Kind: kind,
				Proceeds: proceeds, Quantity: in.Stueckzahl, ContactID: in.KontaktID, Note: in.Grund}
			if in.Bankkonto != "" {
				req.Settlement, req.PaymentAccount = service.SettlementPaid, in.Bankkonto
			}
			if in.NurVorschau {
				out.Vorschau, err = s.Assets.PreviewDisposal(ctx, req)
			} else {
				out.Ergebnis, err = s.Assets.Dispose(ctx, req)
			}
			return out, err
		})
}
