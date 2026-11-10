package mcpserver

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

type ausbuchenIn struct {
	KeyedWriteArgs
	BuchungID   uint   `json:"buchung_id"`
	BelegID     uint   `json:"beleg_id" jsonschema:"Nachweis der Uneinbringlichkeit"`
	Datum       Datum  `json:"datum" jsonschema:"Tag des Eintritts der Uneinbringlichkeit"`
	Betrag      Betrag `json:"betrag,omitempty" jsonschema:"Bruttoteilbetrag; ohne Angabe ganz ausbuchen"`
	Grund       string `json:"grund"`
	NurVorschau bool   `json:"nur_vorschau,omitempty"`
}

func (in ausbuchenIn) preview() bool { return in.NurVorschau }

type ausbuchenOut struct {
	Wiederholung
	Buchung  buchungKurz `json:"buchung"`
	Vorschau bool        `json:"vorschau"`
}

func (out ausbuchenOut) Ref() string { return fmt.Sprintf("buchung:%d", out.Buchung.ID) }
func registerForderung(tb *Toolbox) {
	AddWrite(tb, Spec{Name: "forderung_ausbuchen", Title: "Forderung ausbuchen", Description: "Uneinbringliche Forderung ganz oder teilweise ausbuchen, mit Nachweis und Steuerberichtigung. Erst nur_vorschau vorlegen, nach dem Ja buchen. Keine Rechnungskorrektur."}, func(ctx context.Context, c *Call, s core.AgentServices, in ausbuchenIn) (ausbuchenOut, error) {
		out := ausbuchenOut{Vorschau: in.NurVorschau}
		date, err := in.Datum.check("datum")
		if err != nil {
			return out, err
		}
		amount, err := in.Betrag.cents("betrag")
		if err != nil {
			return out, err
		}
		req := service.WriteOffRequest{OpenItemEntryID: in.BuchungID, ReceiptID: in.BelegID, Date: date, Amount: amount, Reason: in.Grund}
		var entry *domain.JournalEntry
		if in.NurVorschau {
			entry, err = s.Payments.PreviewWriteOff(ctx, req)
		} else {
			entry, err = s.Payments.WriteOffOpenItem(ctx, req)
		}
		if err != nil {
			return out, err
		}
		out.Buchung = buchungKurzOf(entry)
		return out, nil
	})
}
