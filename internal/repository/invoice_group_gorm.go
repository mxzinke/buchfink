package repository

import (
	"context"
	"errors"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type invoiceGroupRepositoryGorm struct {
	db *gorm.DB
}

// NewInvoiceGroupRepository creates a GORM-backed InvoiceGroupRepository.
func NewInvoiceGroupRepository(db *gorm.DB) domain.InvoiceGroupRepository {
	return &invoiceGroupRepositoryGorm{db: db}
}

// FindAll liefert die Verbünde eines Geschäftsjahres — und jeden offenen dazu,
// gleich aus welchem Jahr er stammt.
//
// Ein Vorgang hält sich nicht an den Jahreswechsel: die Abschläge fallen ins
// eine Jahr, die Schlussrechnung ins nächste. Wäre allein das Jahr maßgeblich,
// verschwände der Verbund am 1. Januar aus der Ansicht, obwohl seine Abschläge
// offen sind und die Schlussrechnung noch fehlt — und mit ihm der einzige Weg,
// die Anzahlungen abzusetzen (§ 14 Abs. 5 Satz 2 UStG).
func (r *invoiceGroupRepositoryGorm) FindAll(ctx context.Context, fiscalYear int) ([]domain.InvoiceGroup, error) {
	groups := make([]domain.InvoiceGroup, 0)
	q := dbFrom(ctx, r.db).Order("id desc")
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ? OR closed = ?", fiscalYear, false)
	}
	if err := q.Find(&groups).Error; err != nil {
		return nil, err
	}
	// Die Abschläge werden nachgeladen und nicht per Preload verknüpft: sie
	// hängen an der Rechnung und nicht am Verbund allein, und ein
	// Fremdschlüssel-Preload verlöre die Reihenfolge, in der sie gestellt
	// wurden.
	for i := range groups {
		advances, err := r.advancesOf(ctx, groups[i].ID)
		if err != nil {
			return nil, err
		}
		groups[i].Advances = advances
		// Der Fortschritt wird hier gefüllt und nicht von der Oberfläche
		// nachgerechnet: zwei Rechenwege für dieselben Summen laufen
		// auseinander, sobald einer von beiden eine Regel dazubekommt (etwa
		// dass ein stornierter Abschlag herausfällt).
		groups[i].Progress = groups[i].ComputeProgress()
	}
	return groups, nil
}

func (r *invoiceGroupRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.InvoiceGroup, error) {
	var group domain.InvoiceGroup
	if err := dbFrom(ctx, r.db).First(&group, id).Error; err != nil {
		return nil, mapError(err)
	}
	advances, err := r.advancesOf(ctx, group.ID)
	if err != nil {
		return nil, err
	}
	group.Advances = advances
	group.EnsureLists()
	group.Progress = group.ComputeProgress()
	return &group, nil
}

func (r *invoiceGroupRepositoryGorm) advancesOf(ctx context.Context, groupID uint) ([]domain.AdvanceItem, error) {
	advances := make([]domain.AdvanceItem, 0)
	err := dbFrom(ctx, r.db).Where("group_id = ?", groupID).
		Order("invoice_date asc, id asc").Find(&advances).Error
	if err != nil {
		return nil, err
	}
	for i := range advances {
		if err := r.refreshAdvance(ctx, &advances[i]); err != nil {
			return nil, err
		}
	}
	return advances, nil
}

func (r *invoiceGroupRepositoryGorm) Save(ctx context.Context, group *domain.InvoiceGroup) error {
	// Advances sind kein gemapptes Feld (gorm:"-"); gespeichert wird nur der
	// Verbund selbst.
	return dbFrom(ctx, r.db).Save(group).Error
}

func (r *invoiceGroupRepositoryGorm) SaveAdvance(ctx context.Context, advance *domain.AdvanceItem) error {
	return dbFrom(ctx, r.db).Save(advance).Error
}

func (r *invoiceGroupRepositoryGorm) FindAdvanceByInvoice(ctx context.Context, invoiceID uint) (*domain.AdvanceItem, error) {
	var advance domain.AdvanceItem
	err := dbFrom(ctx, r.db).Where("invoice_id = ?", invoiceID).First(&advance).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := r.refreshAdvance(ctx, &advance); err != nil {
		return nil, err
	}
	return &advance, nil
}

// FindOpenAdvances liefert die ausgestellten, weder vereinnahmten noch
// stornierten Abschläge — die zweite Quelle der Liste offener Posten.
func (r *invoiceGroupRepositoryGorm) FindOpenAdvances(ctx context.Context) ([]domain.AdvanceItem, error) {
	var all []domain.AdvanceItem
	if err := dbFrom(ctx, r.db).Where("cancelled = ?", false).Order("invoice_date asc, id asc").Find(&all).Error; err != nil {
		return nil, err
	}
	out := []domain.AdvanceItem{}
	for i := range all {
		a := &all[i]
		if err := r.refreshAdvance(ctx, a); err != nil {
			return nil, err
		}
		gross, _ := a.Received()
		if gross < a.GrossAmount {
			out = append(out, *a)
		}
	}
	return out, nil
}

func (r *invoiceGroupRepositoryGorm) AdvancePayments(ctx context.Context, invoiceID uint) ([]domain.AdvancePayment, error) {
	var out []domain.AdvancePayment
	err := dbFrom(ctx, r.db).Table("advance_payments AS p").Select("p.*, COALESCE(reversal.booking_date, '') AS reversed_at").Joins("LEFT JOIN journal_entries reversal ON reversal.reversal_of_id = p.entry_id").Where("p.invoice_id = ?", invoiceID).Order("p.date, p.id").Scan(&out).Error
	return out, err
}

func (r *invoiceGroupRepositoryGorm) SaveAdvancePayment(ctx context.Context, payment *domain.AdvancePayment) error {
	return dbFrom(ctx, r.db).Create(payment).Error
}

func (r *invoiceGroupRepositoryGorm) refreshAdvance(ctx context.Context, a *domain.AdvanceItem) error {
	payments, err := r.AdvancePayments(ctx, a.InvoiceID)
	if err != nil || len(payments) == 0 {
		return err
	}
	a.ReceivedGross, a.ReceivedTax, a.SettledAt, a.SettlementEntryID = 0, 0, "", nil
	for _, p := range payments {
		if p.ReversedAt != "" {
			continue
		}
		a.ReceivedGross += p.Gross
		a.ReceivedTax += p.Tax
		a.SettledAt = p.Date
		id := p.EntryID
		a.SettlementEntryID = &id
	}
	if a.ReceivedGross == 0 {
		a.SettledAt, a.SettlementEntryID = "", nil
	}
	return nil
}
