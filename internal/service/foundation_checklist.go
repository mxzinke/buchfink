package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

var foundationEmployeeDuties = map[string]bool{
	"betriebsnummer": true, "sozialversicherung": true, "lohnabrechnung": true, "arbeitsschutz": true,
}

func foundationMasterDataMissing(settings *domain.CompanySettings) []string {
	fields := []struct{ label, value string }{
		{"Firma", settings.CompanyName}, {"Straße und Hausnummer", settings.Street},
		{"Postleitzahl", settings.PostalCode}, {"Ort", settings.City},
	}
	missing := []string{}
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" {
			missing = append(missing, field.label)
		}
	}
	if len(settings.Shareholders) == 0 {
		missing = append(missing, "Gesellschafter")
	}
	for _, shareholder := range settings.Shareholders {
		if strings.TrimSpace(shareholder.Name) == "" || shareholder.ShareCapital <= 0 {
			missing = append(missing, "Gesellschafter mit Namen und Geschäftsanteilen")
			break
		}
	}
	return missing
}

func applyFoundationMasterData(duties []domain.FoundationDuty, settings *domain.CompanySettings) {
	for i := range duties {
		duty := &duties[i]
		if duty.Key != "stammdaten" {
			continue
		}
		duty.MissingFields = foundationMasterDataMissing(settings)
		if len(duty.MissingFields) > 0 {
			duty.IsDone, duty.IsNotApplicable = false, false
			duty.DoneOn = ""
		}
	}
}

// AttachDutyProof wird zusammen mit der Dokumentablage in einer Transaktion ausgeführt.
func (s *FoundationService) AttachDutyProof(ctx context.Context, req DocumentRequest) (*domain.Document, error) {
	if s.documents == nil {
		return nil, fmt.Errorf("die Dokumentenablage ist nicht verfügbar")
	}
	state, err := s.GetState(ctx)
	if err != nil {
		return nil, err
	}
	req.DutyKey = strings.TrimSpace(req.DutyKey)
	for _, duty := range state.Duties {
		if duty.Key != req.DutyKey {
			continue
		}
		if !duty.AcceptsProof {
			return nil, fmt.Errorf("für diese Aufgabe ist keine Nachweisablage vorgesehen")
		}
		if duty.ExcludedBy != "" {
			return nil, fmt.Errorf("%s", duty.ExcludedBy)
		}
		if duty.IsPending {
			return nil, fmt.Errorf("%s", duty.WaitingFor)
		}
		doc, err := s.documents.Attach(ctx, req)
		if err != nil {
			return nil, err
		}
		if !duty.IsDone || duty.IsNotApplicable {
			if err := s.foundationRepo.CompleteTask(ctx, &domain.FoundationTask{
				FoundationID: state.Foundation.ID, Key: duty.Key, Status: "done", DoneOn: todayLocal(),
			}); err != nil {
				return nil, err
			}
		}
		return doc, nil
	}
	return nil, fmt.Errorf("unbekannte Gründungsaufgabe")
}

func applyFoundationDependencies(duties []domain.FoundationDuty, employees string) {
	dependencies := map[string][]string{
		"fragebogen":         {"stammdaten"},
		"ust_id":             {"fragebogen"},
		"eroeffnungsbilanz":  {"stammdaten"},
		"geschaeftsbriefe":   {"stammdaten"},
		"betriebsnummer":     {"unfallversicherung"},
		"sozialversicherung": {"betriebsnummer"},
		"lohnabrechnung":     {"fragebogen"},
	}
	byKey := make(map[string]*domain.FoundationDuty, len(duties))
	for i := range duties {
		byKey[duties[i].Key] = &duties[i]
	}
	for i := range duties {
		duty := &duties[i]
		duty.DependsOn = dependencies[duty.Key]
		if duty.IsDone {
			continue
		}
		if foundationEmployeeDuties[duty.Key] {
			if employees == "none" {
				duty.IsNotApplicable, duty.IsDone, duty.IsPending = true, true, false
				duty.ExcludedBy = "Keine Beschäftigten"
				continue
			}
			if employees != "yes" {
				duty.IsPending = true
				duty.WaitingFor = "Beschäftigte in den Stammdaten angeben"
				continue
			}
		}
		if duty.IsPending {
			duty.WaitingFor = "Nach der Handelsregistereintragung"
			continue
		}
		for _, key := range duty.DependsOn {
			prerequisite := byKey[key]
			if prerequisite != nil && (!prerequisite.IsDone || prerequisite.IsNotApplicable) {
				duty.IsPending = true
				duty.WaitingFor = "Zuerst: " + prerequisite.Title
				break
			}
		}
	}
}

func (s *FoundationService) SetDutyStatus(ctx context.Context, key, status string) error {
	if status != "open" && status != "done" && status != "skipped" {
		return fmt.Errorf("ungültiger Aufgabenstatus")
	}
	state, err := s.GetState(ctx)
	if err != nil {
		return err
	}
	if state.Foundation == nil {
		return fmt.Errorf("für diesen Mandanten ist keine Gründung erfasst")
	}
	var duty *domain.FoundationDuty
	for i := range state.Duties {
		if state.Duties[i].Key == key {
			duty = &state.Duties[i]
			break
		}
	}
	if duty == nil {
		return fmt.Errorf("unbekannte Gründungsaufgabe")
	}
	if duty.ExcludedBy != "" {
		return fmt.Errorf("ändern Sie zuerst die Angabe zur Beschäftigung")
	}
	if status == "done" && len(duty.MissingFields) > 0 {
		return fmt.Errorf("bitte zuerst die Stammdaten ergänzen: %s", strings.Join(duty.MissingFields, ", "))
	}
	if status == "done" && duty.IsPending {
		return fmt.Errorf("%s", duty.WaitingFor)
	}
	if status == "skipped" && duty.Condition == "" {
		return fmt.Errorf("diese Aufgabe kann nicht als nicht zutreffend markiert werden")
	}
	if status == "open" {
		return s.foundationRepo.ClearTask(ctx, state.Foundation.ID, key)
	}
	return s.foundationRepo.CompleteTask(ctx, &domain.FoundationTask{
		FoundationID: state.Foundation.ID, Key: key, Status: status, DoneOn: todayLocal(),
	})
}
