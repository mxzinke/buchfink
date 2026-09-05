package repository

import (
	"context"
	"errors"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type assetRepositoryGorm struct {
	db *gorm.DB
}

// NewAssetRepository creates a GORM-backed AssetRepository.
func NewAssetRepository(db *gorm.DB) domain.AssetRepository {
	return &assetRepositoryGorm{db: db}
}

// Every read preloads the movements. An Anlagegut without them has no
// Anschaffungskosten, no kumulierte AfA and no Buchwert — all three are sums
// over the movements — so a finder that skips the preload would answer every
// question with zero.
func (r *assetRepositoryGorm) preloaded(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Preload("Movements", func(db *gorm.DB) *gorm.DB {
			return db.Order("asset_movements.date asc, asset_movements.id asc")
		}).
		Preload("Documents", func(db *gorm.DB) *gorm.DB {
			return db.Order("asset_documents.document_date desc, asset_documents.id desc")
		}).
		Order("inventory_number asc")
}

func (r *assetRepositoryGorm) FindAll(ctx context.Context) ([]domain.FixedAsset, error) {
	var assets []domain.FixedAsset
	err := r.preloaded(ctx).Find(&assets).Error
	return assets, err
}

func (r *assetRepositoryGorm) FindByClass(ctx context.Context, class domain.AssetClass) ([]domain.FixedAsset, error) {
	var assets []domain.FixedAsset
	err := r.preloaded(ctx).Where("class = ?", class).Find(&assets).Error
	return assets, err
}

func (r *assetRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.FixedAsset, error) {
	var asset domain.FixedAsset
	if err := r.preloaded(ctx).First(&asset, id).Error; err != nil {
		return nil, err
	}
	return &asset, nil
}

// FindPool returns the Sammelposten of a fiscal year, or nil if none was formed.
// § 6 Abs. 2a EStG knows exactly one pool per Wirtschaftsjahr, which is why this
// looks up by year rather than returning a list.
func (r *assetRepositoryGorm) FindPool(ctx context.Context, fiscalYear int) (*domain.FixedAsset, error) {
	var asset domain.FixedAsset
	err := r.preloaded(ctx).
		Where("method = ? AND pool_year = ?", domain.DepreciationPool, fiscalYear).
		First(&asset).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

// Save writes the master data of an asset. It deliberately does not write the
// movements: those are appended one by one alongside their booking, and a full
// Save with an association would silently replace them.
func (r *assetRepositoryGorm) Save(ctx context.Context, asset *domain.FixedAsset) error {
	return r.db.WithContext(ctx).Omit("Movements").Save(asset).Error
}

func (r *assetRepositoryGorm) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.FixedAsset{}, id).Error
}

// AddMovement appends a movement — or writes back one that was corrected before
// any booking hung on it, which is why this is a Save and not a Create.
func (r *assetRepositoryGorm) AddMovement(ctx context.Context, movement *domain.AssetMovement) error {
	return r.db.WithContext(ctx).Save(movement).Error
}

// DeleteMovement removes a movement. Only movements Buchfink generated itself
// and that carry no booking are ever deleted this way — the Sofortabzug of a
// GWG, when its Anschaffungskosten or seine Methode korrigiert werden.
func (r *assetRepositoryGorm) DeleteMovement(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.AssetMovement{}, id).Error
}

func (r *assetRepositoryGorm) FindMovements(ctx context.Context, assetID uint) ([]domain.AssetMovement, error) {
	var movements []domain.AssetMovement
	err := r.db.WithContext(ctx).
		Where("asset_id = ?", assetID).
		Order("date asc, id asc").
		Find(&movements).Error
	return movements, err
}

func (r *assetRepositoryGorm) LinkedEntryIDs(ctx context.Context) (map[uint]bool, error) {
	linked := map[uint]bool{}

	var assetEntries []uint
	if err := r.db.WithContext(ctx).Model(&domain.FixedAsset{}).
		Where("acquisition_entry_id IS NOT NULL").
		Pluck("acquisition_entry_id", &assetEntries).Error; err != nil {
		return nil, err
	}
	for _, id := range assetEntries {
		linked[id] = true
	}

	var movementEntries []uint
	if err := r.db.WithContext(ctx).Model(&domain.AssetMovement{}).
		Where("journal_entry_id IS NOT NULL").
		Pluck("journal_entry_id", &movementEntries).Error; err != nil {
		return nil, err
	}
	for _, id := range movementEntries {
		linked[id] = true
	}
	return linked, nil
}

// FindByAcquisitionEntry returns the Anlagegut a booking is the Zugang of.
//
// Der Zahlungsflow fragt danach, bevor er ein Skonto bucht: auf eine
// Anlagenrechnung mindert es die Anschaffungskosten und nicht den Aufwand
// (§ 255 Abs. 1 Satz 3 HGB). Ohne Treffer bleibt es beim gewöhnlichen Skonto.
func (r *assetRepositoryGorm) FindByAcquisitionEntry(ctx context.Context, entryID uint) (*domain.FixedAsset, error) {
	if entryID == 0 {
		return nil, nil
	}
	var asset domain.FixedAsset
	err := r.preloaded(ctx).Where("acquisition_entry_id = ?", entryID).First(&asset).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

// AddDocument stores a document of an Anlagegut.
func (r *assetRepositoryGorm) AddDocument(ctx context.Context, document *domain.AssetDocument) error {
	return r.db.WithContext(ctx).Save(document).Error
}

// FindDocument returns one document.
func (r *assetRepositoryGorm) FindDocument(ctx context.Context, id uint) (*domain.AssetDocument, error) {
	var document domain.AssetDocument
	if err := r.db.WithContext(ctx).First(&document, id).Error; err != nil {
		return nil, err
	}
	return &document, nil
}

// FindAllDocuments liefert alle Dokumente aller Anlagegüter.
func (r *assetRepositoryGorm) FindAllDocuments(ctx context.Context) ([]domain.AssetDocument, error) {
	documents := make([]domain.AssetDocument, 0)
	err := r.db.WithContext(ctx).Order("id asc").Find(&documents).Error
	return documents, err
}

// DeleteDocument removes the record. Die Datei auf der Platte räumt der Dienst
// ab, und nur, wenn kein anderes Dokument mehr auf sie zeigt.
func (r *assetRepositoryGorm) DeleteDocument(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.AssetDocument{}, id).Error
}

// CountDocumentsBySHA reports how many documents share one file.
func (r *assetRepositoryGorm) CountDocumentsBySHA(ctx context.Context, sha256 string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.AssetDocument{}).
		Where("sha256 = ?", sha256).Count(&count).Error
	return count, err
}

func (r *assetRepositoryGorm) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.FixedAsset{}).Count(&count).Error
	return count, err
}
