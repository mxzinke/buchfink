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

func (r *invoiceGroupRepositoryGorm) FindAll(ctx context.Context, fiscalYear int) ([]domain.InvoiceGroup, error) {
	groups := make([]domain.InvoiceGroup, 0)
	q := dbFrom(ctx, r.db).Order("id desc")
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
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
		return nil, err
	}
	advances, err := r.advancesOf(ctx, group.ID)
	if err != nil {
		return nil, err
	}
	group.Advances = advances
	group.Progress = group.ComputeProgress()
	return &group, nil
}

func (r *invoiceGroupRepositoryGorm) advancesOf(ctx context.Context, groupID uint) ([]domain.AdvanceItem, error) {
	advances := make([]domain.AdvanceItem, 0)
	err := dbFrom(ctx, r.db).Where("group_id = ?", groupID).
		Order("invoice_date asc, id asc").Find(&advances).Error
	return advances, err
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
	return &advance, nil
}

// FindOpenAdvances liefert die ausgestellten, weder vereinnahmten noch
// stornierten Abschläge — die zweite Quelle der Liste offener Posten.
func (r *invoiceGroupRepositoryGorm) FindOpenAdvances(ctx context.Context) ([]domain.AdvanceItem, error) {
	advances := make([]domain.AdvanceItem, 0)
	err := dbFrom(ctx, r.db).
		Where("cancelled = ?", false).
		Where("settled_at IS NULL OR settled_at = ''").
		Order("invoice_date asc, id asc").Find(&advances).Error
	return advances, err
}
