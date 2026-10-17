package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/declaration"
	"github.com/buchfink/buchfink/internal/domain"
)

// DeclarationService stellt die Jahressteuererklärungen zusammen und hält
// Eingaben, Exporte und Übermittlungen fest. Gerechnet wird bei jedem Abruf
// aus den Büchern; gespeichert werden nur die Eingaben und die Stände, die
// hinausgegangen sind.
type DeclarationService struct {
	repo  domain.DeclarationRepository
	audit domain.AuditRepository
	src   DeclarationSources
	now   func() time.Time
}

// DeclarationSources sind die Dienste und Ablagen, aus denen die Erklärungen
// rechnen.
type DeclarationSources struct {
	Journal        domain.JournalRepository
	Settings       domain.SettingsRepository
	FiscalYears    domain.FiscalYearRepository
	Appropriations domain.AppropriationRepository
	Assets         domain.AssetRepository
	InputTax       domain.InputTaxCorrectionRepository
	VatReturns     *VatReturnService
	ZM             *ZMService
	Statement      *StatementService
	TaxRegister    *TaxRegisterService
}

// NewDeclarationService verdrahtet die Erklärungen.
func NewDeclarationService(repo domain.DeclarationRepository, audit domain.AuditRepository, src DeclarationSources) *DeclarationService {
	return &DeclarationService{repo: repo, audit: audit, src: src, now: time.Now}
}

// SetClock ersetzt die Uhr, für Tests.
func (s *DeclarationService) SetClock(now func() time.Time) { s.now = now }

// ErrDeclarationKind: die Art ist keine der gerechneten.
var ErrDeclarationKind = errors.New("unbekannte Erklärung (erwartet ust_jahr, kst oder gewst)")

// Build rechnet eine Erklärung aus dem heutigen Stand der Bücher.
func (s *DeclarationService) Build(ctx context.Context, kind declaration.Kind, year int) (*declaration.Declaration, error) {
	if !kind.Valid() {
		return nil, ErrDeclarationKind
	}
	if err := requireYear(year); err != nil {
		return nil, err
	}
	common, err := s.common(ctx, kind, year)
	if err != nil {
		return nil, err
	}
	switch kind {
	case declaration.KindVatYear:
		data, err := s.vatYearData(ctx, year, common.Settings)
		if err != nil {
			return nil, err
		}
		return declaration.BuildVatYear(common, data)
	case declaration.KindCorporateTax:
		data, err := s.corporateData(ctx, common)
		if err != nil {
			return nil, err
		}
		return declaration.BuildCorporateTax(common, data)
	default:
		data, err := s.corporateData(ctx, common)
		if err != nil {
			return nil, err
		}
		kstInputs, err := s.inputs(ctx, declaration.KindCorporateTax, year)
		if err != nil {
			return nil, err
		}
		return declaration.BuildTradeTax(declaration.TradeCommon{Common: common, CorporateInputs: kstInputs}, data)
	}
}

// InputRequest setzt eine Angabe zu einer Zeile.
type InputRequest struct {
	Kind      declaration.Kind        `json:"kind"`
	Year      int                     `json:"year"`
	Key       string                  `json:"key"`
	Value     declaration.Value       `json:"value"`
	Source    declaration.InputSource `json:"source"`
	ReceiptID *uint                   `json:"receiptId,omitempty"`
	Note      string                  `json:"note,omitempty"`
}

// SetInput legt eine Angabe zu einer Zeile ab, die eine Eingabe zulässt. In
// der App gesetzt gilt sie sofort. Ein Agent setzt mit Rückfrage beim
// Menschen: bei Zeilen, die ihm zustehen, gilt sie; bei Zeilen, die dem
// Menschen vorbehalten sind, bleibt sie ein Vorschlag, bis der Mensch sie in
// der App bestätigt. Was der Mensch gesetzt oder bestätigt hat, ändert nur er.
func (s *DeclarationService) SetInput(ctx context.Context, req InputRequest) (*declaration.Input, error) {
	line, err := inputLine(req.Kind, req.Year, req.Key)
	if err != nil {
		return nil, err
	}
	if err := checkInputValue(line, req.Value); err != nil {
		return nil, err
	}
	if req.Source == "" {
		req.Source = declaration.InputFromPerson
	}
	if !req.Source.Valid() {
		return nil, fmt.Errorf("unbekannte Quelle %q (erwartet bescheid, beschluss, beleg, mensch oder buecher)", req.Source)
	}
	if len(req.Note) > 500 {
		return nil, fmt.Errorf("die Notiz ist länger als 500 Zeichen")
	}
	value, err := json.Marshal(req.Value)
	if err != nil {
		return nil, err
	}
	channel := actor.RecordedChannel(ctx)
	confirmed := channel == actor.ChannelUI || line.Owner != declaration.OwnerHuman

	existing, err := s.repo.FindInput(ctx, string(req.Kind), req.Year, req.Key)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Confirmed && (line.Owner == declaration.OwnerHuman || !actor.Channel(existing.Channel).IsAgent()) {
		if err := requireHuman(ctx, "Das Ändern einer bestätigten Angabe des Menschen"); err != nil {
			return nil, err
		}
	}
	var before *domain.DeclarationInput
	rec := &domain.DeclarationInput{Kind: string(req.Kind), Year: req.Year, Key: req.Key}
	if existing != nil {
		prev := *existing
		before = &prev
		rec = existing
	}
	now := s.now().UTC()
	rec.ValueJSON = string(value)
	rec.Source = string(req.Source)
	rec.ReceiptID = req.ReceiptID
	rec.Note = strings.TrimSpace(req.Note)
	rec.Channel = string(channel)
	rec.Actor = actor.Actor()
	rec.EnteredAt = now
	rec.Confirmed = confirmed
	rec.ConfirmedBy, rec.ConfirmedAt = "", nil
	if confirmed {
		rec.ConfirmedBy = confirmer(channel)
		rec.ConfirmedAt = &now
	}
	if err := s.repo.SaveInput(ctx, rec); err != nil {
		return nil, fmt.Errorf("die Angabe ließ sich nicht speichern: %w", err)
	}
	state := "gesetzt"
	if !confirmed {
		state = "vorgeschlagen, Bestätigung in der App ausstehend"
	}
	if err := logAuditChange(ctx, s.audit, domain.AuditActionUpdate, "DECLARATION_INPUT", inputRef(rec),
		fmt.Sprintf("%s %d, %s: %s (%s)", req.Kind.Label(), req.Year, line.Label, req.Value.Canonical(), state),
		before, rec); err != nil {
		return nil, err
	}
	out := toInput(*rec)
	return &out, nil
}

// ConfirmInput bestätigt den Vorschlag eines Agenten. Nur in der App.
func (s *DeclarationService) ConfirmInput(ctx context.Context, kind declaration.Kind, year int, key string) (*declaration.Input, error) {
	if err := requireHuman(ctx, "Die Bestätigung einer Angabe zur Steuererklärung"); err != nil {
		return nil, err
	}
	rec, err := s.repo.FindInput(ctx, string(kind), year, key)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("zur Zeile %s gibt es keine Angabe, die sich bestätigen ließe", key)
	}
	if rec.Confirmed {
		out := toInput(*rec)
		return &out, nil
	}
	before := *rec
	now := s.now().UTC()
	rec.Confirmed = true
	rec.ConfirmedBy = confirmer(actor.RecordedChannel(ctx))
	rec.ConfirmedAt = &now
	if err := s.repo.SaveInput(ctx, rec); err != nil {
		return nil, err
	}
	if err := logAuditChange(ctx, s.audit, domain.AuditActionUpdate, "DECLARATION_INPUT", inputRef(rec),
		fmt.Sprintf("%s %d, %s: Vorschlag bestätigt", kind.Label(), year, key), &before, rec); err != nil {
		return nil, err
	}
	out := toInput(*rec)
	return &out, nil
}

// DeleteInput nimmt eine Angabe zurück; die Zeile rechnet dann wieder aus den
// Büchern oder bleibt offen. Eine bestätigte Angabe zu einer Zeile, die dem
// Menschen vorbehalten ist, nimmt nur der Mensch zurück.
func (s *DeclarationService) DeleteInput(ctx context.Context, kind declaration.Kind, year int, key string) error {
	line, err := inputLine(kind, year, key)
	if err != nil {
		return err
	}
	rec, err := s.repo.FindInput(ctx, string(kind), year, key)
	if err != nil {
		return err
	}
	if rec == nil {
		return nil
	}
	if rec.Confirmed && line.Owner == declaration.OwnerHuman {
		if err := requireHuman(ctx, "Das Zurücknehmen einer bestätigten Angabe des Menschen"); err != nil {
			return err
		}
	}
	if err := s.repo.DeleteInput(ctx, rec.ID); err != nil {
		return err
	}
	return logAuditChange(ctx, s.audit, domain.AuditActionUpdate, "DECLARATION_INPUT", inputRef(rec),
		fmt.Sprintf("%s %d, %s: Angabe zurückgenommen", kind.Label(), year, line.Label), rec, nil)
}

// Inputs liefert die Angaben zu einer Erklärung.
func (s *DeclarationService) Inputs(ctx context.Context, kind declaration.Kind, year int) ([]declaration.Input, error) {
	return s.inputs(ctx, kind, year)
}

// Records liefert die Exporte und Übermittlungen einer Erklärung.
func (s *DeclarationService) Records(ctx context.Context, kind declaration.Kind, year int) ([]domain.DeclarationRecord, error) {
	return s.repo.FindRecords(ctx, string(kind), year)
}

// RecordExport hält einen Export fest. Die Erklärung muss dem heutigen Stand
// entsprechen: der Kern schreibt die Datei aus einem Lesevorgang und hält sie
// hier fest, und dazwischen kann sich etwas geändert haben.
func (s *DeclarationService) RecordExport(ctx context.Context, d *declaration.Declaration, file, sha string) (*domain.DeclarationRecord, error) {
	current, err := s.Build(ctx, d.Kind, d.Period.Year)
	if err != nil {
		return nil, err
	}
	if current.FiguresHash != d.FiguresHash {
		return nil, fmt.Errorf("die Zahlen haben sich während des Exports geändert; bitte erneut exportieren")
	}
	snapshot, err := declaration.ExportJSON(d)
	if err != nil {
		return nil, err
	}
	rec := &domain.DeclarationRecord{
		Kind: string(d.Kind), Year: d.Period.Year, Status: string(declaration.StatusExported),
		FiguresHash: d.FiguresHash, FormYear: d.FormYear, CatalogVersion: d.Basis.CatalogVersion,
		ProgramVersion: d.Basis.ProgramVersion, JournalHash: d.Basis.JournalHash,
		SnapshotJSON: string(snapshot), ExportFile: file, ExportSHA256: sha,
		Channel: string(actor.RecordedChannel(ctx)), Actor: actor.Actor(), CreatedAt: s.now().UTC(),
	}
	if err := s.repo.CreateRecord(ctx, rec); err != nil {
		return nil, err
	}
	return rec, logAudit(ctx, s.audit, domain.AuditActionExport, "DECLARATION", fmt.Sprintf("%d", rec.ID),
		fmt.Sprintf("%s exportiert nach %s (Zahlen-Hash %s, SHA-256 %s, %d blockierende Befunde)",
			d.Title, file, d.FiguresHash, sha, len(d.Blocking())))
}

// SubmissionRequest erfasst die Übermittlung einer Erklärung in Mein ELSTER.
type SubmissionRequest struct {
	Kind           declaration.Kind `json:"kind"`
	Year           int              `json:"year"`
	SubmittedAt    string           `json:"submittedAt"`
	TransferTicket string           `json:"transferTicket"`
	// FiguresHash ist der Zahlen-Hash der übermittelten Zahlen: der eines
	// Exports oder der des heutigen Stands.
	FiguresHash string `json:"figuresHash"`
	// OverrideReason begründet die Übermittlung trotz blockierender Befunde.
	OverrideReason string `json:"overrideReason,omitempty"`
}

// MarkSubmitted hält die Übermittlung mit Transferticket und Datum fest. Nur in
// der App: das Transferticket ist der Nachweis, den der Mensch aus Mein ELSTER
// mitbringt.
func (s *DeclarationService) MarkSubmitted(ctx context.Context, req SubmissionRequest) (*domain.DeclarationRecord, error) {
	if err := requireHuman(ctx, "Die Bestätigung der Übermittlung einer Steuererklärung"); err != nil {
		return nil, err
	}
	if !req.Kind.Valid() {
		return nil, ErrDeclarationKind
	}
	ticket := strings.TrimSpace(req.TransferTicket)
	if ticket == "" || len(ticket) > 100 {
		return nil, fmt.Errorf("das Transferticket aus dem Übermittlungsprotokoll fehlt oder ist länger als 100 Zeichen")
	}
	if err := domain.CheckDate("Übermittlungsdatum", req.SubmittedAt); err != nil {
		return nil, err
	}
	if req.SubmittedAt > s.now().Format(domain.DateLayout) {
		return nil, fmt.Errorf("das Übermittlungsdatum %s liegt in der Zukunft", req.SubmittedAt)
	}
	current, err := s.Build(ctx, req.Kind, req.Year)
	if err != nil {
		return nil, err
	}
	if req.SubmittedAt < current.Period.From {
		return nil, fmt.Errorf("das Übermittlungsdatum %s liegt vor dem Beginn des Zeitraums %s", req.SubmittedAt, current.Period.From)
	}
	snapshot, file, err := s.submittedState(ctx, current, req.FiguresHash)
	if err != nil {
		return nil, err
	}
	var parsed declaration.Declaration
	if err := json.Unmarshal([]byte(snapshot), &parsed); err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(req.OverrideReason)
	if blocking := parsed.Blocking(); len(blocking) > 0 && reason == "" {
		texts := make([]string, 0, len(blocking))
		for _, f := range blocking {
			texts = append(texts, f.Text)
		}
		return nil, fmt.Errorf("die übermittelten Zahlen haben blockierende Befunde; mit Begründung trotzdem erfassen: %s",
			strings.Join(texts, " | "))
	}
	rec := &domain.DeclarationRecord{
		Kind: string(req.Kind), Year: req.Year, Status: string(declaration.StatusSubmitted),
		FiguresHash: req.FiguresHash, FormYear: parsed.FormYear, CatalogVersion: parsed.Basis.CatalogVersion,
		ProgramVersion: parsed.Basis.ProgramVersion, JournalHash: parsed.Basis.JournalHash,
		SnapshotJSON: snapshot, ExportFile: file,
		SubmittedAt: req.SubmittedAt, TransferTicket: ticket, OverrideReason: reason,
		Channel: string(actor.RecordedChannel(ctx)), Actor: actor.Actor(), CreatedAt: s.now().UTC(),
	}
	if err := s.repo.CreateRecord(ctx, rec); err != nil {
		return nil, err
	}
	details := fmt.Sprintf("%s am %s übermittelt (Transferticket %s, Zahlen-Hash %s)",
		current.Title, req.SubmittedAt, ticket, req.FiguresHash)
	if reason != "" {
		details += ". Blockierende Befunde übergangen mit der Begründung: " + reason
	}
	if req.FiguresHash != current.FiguresHash {
		details += ". Die Bücher sind seit dem Export geändert"
	}
	if err := logAudit(ctx, s.audit, domain.AuditActionUpdate, "DECLARATION", fmt.Sprintf("%d", rec.ID), details); err != nil {
		return nil, err
	}
	return rec, nil
}

// submittedState sucht die Zahlen, die übermittelt wurden: den heutigen Stand
// oder einen Export mit diesem Hash.
func (s *DeclarationService) submittedState(ctx context.Context, current *declaration.Declaration, hash string) (string, string, error) {
	if strings.TrimSpace(hash) == "" {
		return "", "", fmt.Errorf("der Zahlen-Hash der übermittelten Erklärung fehlt")
	}
	if hash == current.FiguresHash {
		data, err := declaration.ExportJSON(current)
		return string(data), "", err
	}
	records, err := s.repo.FindRecords(ctx, string(current.Kind), current.Period.Year)
	if err != nil {
		return "", "", err
	}
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].FiguresHash == hash {
			return records[i].SnapshotJSON, records[i].ExportFile, nil
		}
	}
	return "", "", fmt.Errorf("der Zahlen-Hash %s passt weder zum heutigen Stand (%s) noch zu einem Export; "+
		"die übermittelten Zahlen lassen sich nicht zuordnen", hash, current.FiguresHash)
}

// inputLine prüft, ob eine Zeile eine Eingabe zulässt.
func inputLine(kind declaration.Kind, year int, key string) (declaration.CatalogLine, error) {
	if !kind.Valid() {
		return declaration.CatalogLine{}, ErrDeclarationKind
	}
	if err := requireYear(year); err != nil {
		return declaration.CatalogLine{}, err
	}
	cat, _, err := declaration.CatalogFor(kind, year)
	if err != nil {
		return declaration.CatalogLine{}, err
	}
	line, ok := cat.Line(key)
	if !ok {
		return declaration.CatalogLine{}, fmt.Errorf("die %s kennt keine Zeile %q", kind.Label(), key)
	}
	if !line.Input {
		return declaration.CatalogLine{}, fmt.Errorf("die Zeile %q rechnet aus den Büchern und nimmt keine Eingabe an", line.Label)
	}
	return line, nil
}

// checkInputValue prüft, ob der Wert zur Einheit der Zeile passt.
func checkInputValue(line declaration.CatalogLine, v declaration.Value) error {
	set := 0
	for _, present := range []bool{v.Cents != nil, v.Text != nil, v.Bool != nil, v.Date != nil} {
		if present {
			set++
		}
	}
	if set != 1 {
		return fmt.Errorf("zur Zeile %q gehört genau ein Wert", line.Label)
	}
	switch line.Unit {
	case declaration.UnitEuro, declaration.UnitCents:
		if v.Cents == nil {
			return fmt.Errorf("die Zeile %q erwartet einen Betrag", line.Label)
		}
		if line.Unit == declaration.UnitEuro && *v.Cents%100 != 0 {
			return fmt.Errorf("die Zeile %q erwartet volle Euro", line.Label)
		}
	case declaration.UnitYesNo:
		if v.Bool == nil {
			return fmt.Errorf("die Zeile %q erwartet ja oder nein", line.Label)
		}
	case declaration.UnitDate:
		if v.Date == nil {
			return fmt.Errorf("die Zeile %q erwartet ein Datum", line.Label)
		}
		return domain.CheckDate(line.Label, *v.Date)
	default:
		if v.Text == nil || strings.TrimSpace(*v.Text) == "" {
			return fmt.Errorf("die Zeile %q erwartet einen Text", line.Label)
		}
		if len(*v.Text) > 500 {
			return fmt.Errorf("der Text zur Zeile %q ist länger als 500 Zeichen", line.Label)
		}
	}
	return nil
}

func (s *DeclarationService) inputs(ctx context.Context, kind declaration.Kind, year int) ([]declaration.Input, error) {
	stored, err := s.repo.FindInputs(ctx, string(kind), year)
	if err != nil {
		return nil, err
	}
	out := make([]declaration.Input, 0, len(stored))
	for _, rec := range stored {
		out = append(out, toInput(rec))
	}
	return out, nil
}

func toInput(rec domain.DeclarationInput) declaration.Input {
	var v declaration.Value
	_ = json.Unmarshal([]byte(rec.ValueJSON), &v)
	return declaration.Input{
		Key: rec.Key, Value: v, Source: declaration.InputSource(rec.Source), ReceiptID: rec.ReceiptID,
		Note: rec.Note, Channel: rec.Channel, EnteredBy: rec.Actor, EnteredAt: rec.EnteredAt,
		Confirmed: rec.Confirmed, ConfirmedBy: rec.ConfirmedBy, ConfirmedAt: rec.ConfirmedAt,
	}
}

func inputRef(rec *domain.DeclarationInput) string {
	return fmt.Sprintf("%s/%d/%s", rec.Kind, rec.Year, rec.Key)
}

// confirmer nennt, wer eine Angabe bestätigt hat: in der App der Bearbeiter,
// bei einem Agenten der Agent mit Rückfrage im Client.
func confirmer(channel actor.Channel) string {
	if channel.IsAgent() {
		return fmt.Sprintf("%s (%s, mit Rückfrage im Client)", actor.Actor(), channel)
	}
	return actor.Actor()
}
