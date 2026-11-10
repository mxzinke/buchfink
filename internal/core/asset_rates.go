package core

import "context"

// PrefetchAssetRate holt den Stichtagskurs vor dem Schreibvorgang aus der Währung der Kartei.
func (t *Tenant) PrefetchAssetRate(ctx context.Context, assetID uint, date string) error {
	var currency string
	err := t.Read(ctx, func(ctx context.Context, s *Services) error {
		detail, err := s.Assets.Get(ctx, s.Journal.FiscalYearOf(ctx, date), assetID)
		if err != nil {
			return err
		}
		currency = detail.Asset.Currency
		return nil
	})
	if err != nil {
		return err
	}
	return t.PrefetchRate(ctx, currency, date)
}
