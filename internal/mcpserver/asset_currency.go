package mcpserver

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

type fremdwaehrungsanlageIn struct {
	Art         string `json:"art" jsonschema:"zugang oder bewertung"`
	AnlageID    uint   `json:"anlage_id,omitempty"`
	Datum       Datum  `json:"datum"`
	Kurs        int64  `json:"kurs_millionstel,omitempty" jsonschema:"Fremdwährung je Euro × 1000000; ohne Angabe Kursabruf"`
	BuchungID   uint   `json:"buchung_id,omitempty"`
	Konto       Konto  `json:"konto,omitempty"`
	Bezeichnung string `json:"bezeichnung,omitempty"`
	Waehrung    string `json:"waehrung,omitempty"`
	Fremdbetrag Betrag `json:"fremdbetrag,omitempty" jsonschema:"Anschaffungskosten in Originalwährung"`
	Stueckzahl  int64  `json:"stueckzahl,omitempty" jsonschema:"Zehntausendstel Stück"`
	Kursquelle  string `json:"kursquelle,omitempty"`
	FaelligAm   Datum  `json:"faellig_am,omitempty"`
}

func prepareForeignAsset(ctx context.Context, t *core.Tenant, in *fremdwaehrungsanlageIn) error {
	if in == nil || in.Kurs != 0 {
		return nil
	}
	if in.Art == "zugang" {
		return t.PrefetchRate(ctx, in.Waehrung, string(in.Datum))
	}
	return t.PrefetchAssetRate(ctx, in.AnlageID, string(in.Datum))
}

func runForeignAsset(ctx context.Context, s core.AgentServices, year int, in abschlussbuchungIn, out *abschlussbuchungOut) error {
	req := in.Fremdwaehrungsanlage
	if req == nil {
		return fmt.Errorf("fremdwaehrungsanlage fehlt")
	}
	date, err := req.Datum.check("datum")
	if err != nil {
		return err
	}
	switch req.Art {
	case "zugang":
		cost, err := req.Fremdbetrag.cents("fremdbetrag")
		if err != nil {
			return err
		}
		asset, err := s.Assets.RecordForeignAcquisition(ctx, service.ForeignAssetAcquisitionRequest{
			Preview: in.NurVorschau, EntryID: req.BuchungID, Account: string(req.Konto), Name: req.Bezeichnung, Date: date,
			MaturityDate: string(req.FaelligAm), Quantity: domain.Units(req.Stueckzahl),
			Currency: req.Waehrung, ForeignCost: cost, RatePerEuro: req.Kurs, RateSource: req.Kursquelle,
		})
		if err == nil {
			out.AnlageID = &asset.ID
			out.Betrag = optBetrag(asset.AcquisitionCost)
		}
		return err
	case "bewertung":
		request := service.CurrencyValuationRequest{AssetID: req.AnlageID, Date: date, RatePerEuro: req.Kurs}
		if in.NurVorschau {
			value, err := s.Assets.ValuateCurrency(ctx, request)
			if err == nil {
				out.Betrag = optBetrag(value.ProposedAmount)
				out.Erlaeuterung = value.Explanation
			}
			return err
		}
		entry, err := s.Assets.BookCurrencyValuation(ctx, request)
		if err == nil {
			addEntry(out, entry)
		}
		return err
	default:
		return fmt.Errorf("unbekannter Fremdwährungsvorgang")
	}
}
