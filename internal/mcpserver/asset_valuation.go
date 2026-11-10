package mcpserver

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

type anlagenbewertungIn struct {
	AnlageID  uint   `json:"anlage_id"`
	Art       string `json:"art" jsonschema:"abschreibung, zuschreibung oder beibehalten"`
	Datum     Datum  `json:"datum"`
	Betrag    Betrag `json:"betrag,omitempty"`
	Dauerhaft bool   `json:"dauerhaft,omitempty"`
	Grund     string `json:"grund"`
	Nachweis  string `json:"nachweis"`
	BelegID   *uint  `json:"beleg_id,omitempty"`
}

func runAssetValuation(ctx context.Context, s core.AgentServices, year int, in abschlussbuchungIn, out *abschlussbuchungOut) error {
	req := in.Anlagenbewertung
	if req == nil {
		return fmt.Errorf("anlagenbewertung fehlt")
	}
	date, err := req.Datum.check("datum")
	if err != nil {
		return err
	}
	amount, err := req.Betrag.cents("betrag")
	if err != nil {
		return err
	}
	var entry *domain.JournalEntry
	switch req.Art {
	case "abschreibung":
		entry, err = s.Assets.BookImpairment(ctx, service.ImpairmentRequest{
			AssetID: req.AnlageID, Date: date, Amount: amount, Permanent: req.Dauerhaft,
			Reason: req.Grund, Evidence: req.Nachweis, ReceiptID: req.BelegID, Preview: in.NurVorschau,
		})
	case "zuschreibung":
		entry, err = s.Assets.BookWriteUp(ctx, service.WriteUpRequest{
			AssetID: req.AnlageID, Date: date, Amount: amount, Reason: req.Grund,
			Evidence: req.Nachweis, ReceiptID: req.BelegID, Preview: in.NurVorschau,
		})
	case "beibehalten":
		if in.NurVorschau {
			out.Erlaeuterung = "Grund und Nachweis für die Beibehaltung prüfen"
			return nil
		}
		_, err = s.Assets.ConfirmImpairmentPersists(ctx, req.AnlageID, year, req.Grund, req.Nachweis)
	default:
		return fmt.Errorf("unbekannte Anlagenbewertung")
	}
	if err != nil {
		return err
	}
	if in.NurVorschau && entry != nil {
		out.Zeilen = zeilenOf(entry.Lines)
	} else {
		addEntry(out, entry)
	}
	out.Betrag = optBetrag(amount)
	return nil
}
