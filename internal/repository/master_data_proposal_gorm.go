package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type masterDataProposalRepositoryGorm struct {
	db *gorm.DB
}

// NewMasterDataProposalRepository speichert die Vorschläge für die Stammdaten.
func NewMasterDataProposalRepository(db *gorm.DB) domain.MasterDataProposalRepository {
	return &masterDataProposalRepositoryGorm{db: db}
}

func (r *masterDataProposalRepositoryGorm) Create(ctx context.Context, p *domain.MasterDataProposal) error {
	p.CreatedAt = time.Now().UTC()
	return dbFrom(ctx, r.db).Create(p).Error
}

func (r *masterDataProposalRepositoryGorm) Find(ctx context.Context, id uint) (*domain.MasterDataProposal, error) {
	var p domain.MasterDataProposal
	err := dbFrom(ctx, r.db).First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("Vorschlag %d nicht gefunden", id)
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *masterDataProposalRepositoryGorm) FindByStatus(
	ctx context.Context, status domain.MasterDataProposalStatus,
) ([]domain.MasterDataProposal, error) {
	out := []domain.MasterDataProposal{}
	if err := dbFrom(ctx, r.db).Where("status = ?", status).Order("id").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *masterDataProposalRepositoryGorm) FindRecent(ctx context.Context, limit int) ([]domain.MasterDataProposal, error) {
	out := []domain.MasterDataProposal{}
	if err := dbFrom(ctx, r.db).Order("id DESC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *masterDataProposalRepositoryGorm) Save(ctx context.Context, p *domain.MasterDataProposal) error {
	return dbFrom(ctx, r.db).Save(p).Error
}
