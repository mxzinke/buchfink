package oauth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Store hält Apps, Zulassungen, Codes und Tokens in einer eigenen
// SQLite-Datei neben der App-Konfiguration. Sie liegt bewusst nicht in der
// Mandantendatenbank: eine Sicherung trägt keine Tokens weiter, und ein
// Mandantenwechsel macht keine Anmeldung ungültig. Codes und Tokens stehen
// nur als SHA-256 darin.
type Store struct {
	db *gorm.DB
	// mu ordnet die Schreiber; die Datei hat genau eine Verbindung.
	mu sync.Mutex
}

// Client ist eine per DCR registrierte App.
type Client struct {
	ID   string `gorm:"primaryKey;size:64"`
	Name string `gorm:"size:100;not null"`
	// Slug ist der Kurzname im Erfassungsweg agent:<slug>. Er entsteht bei der
	// ersten Zustimmung und ändert sich danach nie, weil er in die
	// Hash-Ketten der Buchungen eingeht.
	Slug            *string `gorm:"uniqueIndex;size:40"`
	RedirectURIs    string  `gorm:"type:text;not null"`
	SoftwareID      string  `gorm:"size:200"`
	SoftwareVersion string  `gorm:"size:100"`
	CreatedAt       time.Time
	FirstApprovedAt *time.Time
	LastUsedAt      *time.Time
}

// Grant ist die Zustimmung zu einer App für einen Mandanten.
type Grant struct {
	ID         uint   `gorm:"primaryKey"`
	ClientID   string `gorm:"index;size:64;not null"`
	TenantID   string `gorm:"index;size:100;not null"`
	TenantName string `gorm:"size:200"`
	Scopes     string `gorm:"size:200"`
	GrantedAt  time.Time
	GrantedBy  string `gorm:"size:200"`
	RevokedAt  *time.Time
}

// Family ist eine Tokenkette: alles, was aus einem Code entsteht. Wird ein
// Code oder ein Refresh-Token ein zweites Mal eingelöst, fällt die ganze
// Kette.
type Family struct {
	ID           string `gorm:"primaryKey;size:64"`
	ClientID     string `gorm:"index;size:64;not null"`
	Scopes       string `gorm:"size:200"`
	Resource     string `gorm:"size:200"`
	CreatedAt    time.Time
	ExpiresAt    time.Time
	RevokedAt    *time.Time
	RevokeReason string `gorm:"size:200"`
}

// Code ist ein Autorisierungscode, einmal einlösbar.
type Code struct {
	Hash        string `gorm:"primaryKey;size:64"`
	ClientID    string `gorm:"size:64;not null"`
	RedirectURI string `gorm:"type:text;not null"`
	Challenge   string `gorm:"size:128;not null"`
	Scopes      string `gorm:"size:200"`
	Resource    string `gorm:"size:200"`
	FamilyID    string `gorm:"size:64;not null"`
	ExpiresAt   time.Time
	UsedAt      *time.Time
}

// Token ist ein Access- oder Refresh-Token.
type Token struct {
	Hash      string `gorm:"primaryKey;size:64"`
	Kind      string `gorm:"size:10;not null"`
	ClientID  string `gorm:"index;size:64;not null"`
	FamilyID  string `gorm:"index;size:64;not null"`
	Scopes    string `gorm:"size:200"`
	Resource  string `gorm:"size:200"`
	IssuedAt  time.Time
	ExpiresAt time.Time
	RotatedAt *time.Time
	RevokedAt *time.Time
}

const (
	kindAccess  = "access"
	kindRefresh = "refresh"
)

// OpenStore öffnet die Datei und legt fehlende Tabellen an.
func OpenStore(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("der Ordner für die verbundenen Apps ließ sich nicht anlegen: %w", err)
	}
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)&_pragma=secure_delete(ON)&_txlock=immediate", path)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:  logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("die Datei der verbundenen Apps ließ sich nicht öffnen: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&Client{}, &Grant{}, &Family{}, &Code{}, &Token{}); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("die Datei der verbundenen Apps ließ sich nicht einrichten: %w", err)
	}
	// Tokens und Zulassungen sind Zugangsdaten; nur der Besitzer liest sie.
	_ = os.Chmod(path, 0o600)
	return &Store{db: db}, nil
}

// Close schließt die Datei.
func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// write führt fn als eine Transaktion unter der Schreibsperre aus.
func (s *Store) write(ctx context.Context, fn func(tx *gorm.DB) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.WithContext(ctx).Transaction(fn)
}

// CreateClient legt eine registrierte App an.
func (s *Store) CreateClient(ctx context.Context, c *Client) error {
	return s.write(ctx, func(tx *gorm.DB) error { return tx.Create(c).Error })
}

// Client liefert eine App oder nil.
func (s *Store) Client(ctx context.Context, id string) (*Client, error) {
	var c Client
	err := s.db.WithContext(ctx).Where("id = ?", id).Take(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// approve hält die Zustimmung fest: Kurzname beim ersten Mal, Zulassung für
// den Mandanten (falls einer offen ist), neue Tokenkette und Code. Alles in
// einer Transaktion.
func (s *Store) approve(ctx context.Context, a approval) (*Client, error) {
	var client Client
	err := s.write(ctx, func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", a.ClientID).Take(&client).Error; err != nil {
			return fmt.Errorf("die App ist nicht mehr registriert: %w", err)
		}
		if client.Slug == nil {
			slug, err := uniqueSlug(tx, client.Name)
			if err != nil {
				return err
			}
			client.Slug = &slug
			at := a.At
			client.FirstApprovedAt = &at
			if err := tx.Save(&client).Error; err != nil {
				return err
			}
		}
		if a.TenantID != "" {
			if err := tx.Model(&Grant{}).
				Where("client_id = ? AND tenant_id = ? AND revoked_at IS NULL", a.ClientID, a.TenantID).
				Update("revoked_at", a.At).Error; err != nil {
				return err
			}
			if err := tx.Create(&Grant{
				ClientID: a.ClientID, TenantID: a.TenantID, TenantName: a.TenantName,
				Scopes: a.Code.Scopes, GrantedAt: a.At, GrantedBy: a.GrantedBy,
			}).Error; err != nil {
				return err
			}
		}
		if err := tx.Create(&a.Family).Error; err != nil {
			return err
		}
		return tx.Create(&a.Code).Error
	})
	if err != nil {
		return nil, err
	}
	return &client, nil
}

type approval struct {
	ClientID   string
	TenantID   string
	TenantName string
	GrantedBy  string
	At         time.Time
	Family     Family
	Code       Code
}

// uniqueSlug bildet aus dem Namen einen freien Kurznamen.
func uniqueSlug(tx *gorm.DB, name string) (string, error) {
	base := slugify(name)
	for i := 1; i < 1000; i++ {
		candidate := base
		if i > 1 {
			suffix := fmt.Sprintf("-%d", i)
			if len(candidate)+len(suffix) > maxSlugLen {
				candidate = strings.TrimRight(candidate[:maxSlugLen-len(suffix)], "-")
			}
			candidate += suffix
		}
		var n int64
		if err := tx.Model(&Client{}).Where("slug = ?", candidate).Count(&n).Error; err != nil {
			return "", err
		}
		if n == 0 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("für %q ließ sich kein freier Kurzname finden", name)
}

// redeemCode löst einen Code ein. check prüft Client, Rücksprung, PKCE und
// Ressource; scheitert sie, bleibt der Code unberührt. Wurde der Code schon
// eingelöst, fällt die Kette, die aus ihm entstand (RFC 6749 §4.1.2); der
// Widerruf wird gespeichert, der Aufrufer bekommt trotzdem einen Fehler.
func (s *Store) redeemCode(ctx context.Context, hash string, now time.Time, check func(*Code) error,
	issue func(tx *gorm.DB, family *Family) error) error {
	var refused error
	err := s.write(ctx, func(tx *gorm.DB) error {
		var code Code
		if err := tx.Where("hash = ?", hash).Take(&code).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errInvalidGrant("der Code ist unbekannt")
			}
			return err
		}
		if code.UsedAt != nil {
			refused = errInvalidGrant("der Code wurde schon eingelöst; die daraus entstandenen Tokens sind widerrufen")
			return revokeFamily(tx, code.FamilyID, now, "Code ein zweites Mal eingelöst")
		}
		if !now.Before(code.ExpiresAt) {
			return errInvalidGrant("der Code ist abgelaufen")
		}
		if err := check(&code); err != nil {
			return err
		}
		res := tx.Model(&Code{}).Where("hash = ? AND used_at IS NULL", hash).Update("used_at", now)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			refused = errInvalidGrant("der Code wurde schon eingelöst")
			return revokeFamily(tx, code.FamilyID, now, "Code ein zweites Mal eingelöst")
		}
		var family Family
		if err := tx.Where("id = ?", code.FamilyID).Take(&family).Error; err != nil {
			return err
		}
		if family.RevokedAt != nil {
			return errInvalidGrant("die Anmeldung wurde widerrufen")
		}
		return issue(tx, &family)
	})
	if err != nil {
		return err
	}
	return refused
}

// rotateRefresh löst ein Refresh-Token ein und ersetzt es. Ein schon
// ersetztes Token ist ein Zeichen für Diebstahl: die ganze Kette fällt.
func (s *Store) rotateRefresh(ctx context.Context, hash string, now time.Time, check func(*Token, *Family) error,
	issue func(tx *gorm.DB, family *Family, old *Token) error) error {
	var refused error
	err := s.write(ctx, func(tx *gorm.DB) error {
		var token Token
		if err := tx.Where("hash = ? AND kind = ?", hash, kindRefresh).Take(&token).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errInvalidGrant("das Refresh-Token ist unbekannt")
			}
			return err
		}
		var family Family
		if err := tx.Where("id = ?", token.FamilyID).Take(&family).Error; err != nil {
			return err
		}
		if token.RevokedAt != nil || family.RevokedAt != nil {
			return errInvalidGrant("die Anmeldung wurde widerrufen")
		}
		if token.RotatedAt != nil {
			refused = errInvalidGrant("das Refresh-Token wurde schon eingelöst; die Anmeldung ist widerrufen")
			return revokeFamily(tx, token.FamilyID, now, "Refresh-Token ein zweites Mal eingelöst")
		}
		if !now.Before(token.ExpiresAt) || !now.Before(family.ExpiresAt) {
			return errInvalidGrant("das Refresh-Token ist abgelaufen")
		}
		if err := check(&token, &family); err != nil {
			return err
		}
		res := tx.Model(&Token{}).Where("hash = ? AND rotated_at IS NULL", hash).Update("rotated_at", now)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			refused = errInvalidGrant("das Refresh-Token wurde schon eingelöst")
			return revokeFamily(tx, token.FamilyID, now, "Refresh-Token ein zweites Mal eingelöst")
		}
		return issue(tx, &family, &token)
	})
	if err != nil {
		return err
	}
	return refused
}

func revokeFamily(tx *gorm.DB, familyID string, now time.Time, reason string) error {
	if err := tx.Model(&Family{}).Where("id = ? AND revoked_at IS NULL", familyID).
		Updates(map[string]any{"revoked_at": now, "revoke_reason": reason}).Error; err != nil {
		return err
	}
	return tx.Model(&Token{}).Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", now).Error
}

// accessToken liefert ein gültiges Access-Token samt App oder nil.
func (s *Store) accessToken(ctx context.Context, hash string, now time.Time) (*Token, *Client, error) {
	var token Token
	err := s.db.WithContext(ctx).Where("hash = ? AND kind = ?", hash, kindAccess).Take(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if token.RevokedAt != nil || !now.Before(token.ExpiresAt) {
		return nil, nil, nil
	}
	var family Family
	if err := s.db.WithContext(ctx).Where("id = ?", token.FamilyID).Take(&family).Error; err != nil {
		return nil, nil, nil
	}
	if family.RevokedAt != nil {
		return nil, nil, nil
	}
	var client Client
	if err := s.db.WithContext(ctx).Where("id = ?", token.ClientID).Take(&client).Error; err != nil {
		return nil, nil, nil
	}
	return &token, &client, nil
}

// revokeToken widerruft ein Token (RFC 7009). Ein Refresh-Token nimmt seine
// Kette mit, ein Access-Token nur sich selbst. Ein unbekanntes Token oder
// eines einer anderen App ist kein Fehler.
func (s *Store) revokeToken(ctx context.Context, hash, clientID string, now time.Time) error {
	return s.write(ctx, func(tx *gorm.DB) error {
		var token Token
		err := tx.Where("hash = ?", hash).Take(&token).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if clientID != "" && token.ClientID != clientID {
			return nil
		}
		if token.Kind == kindRefresh {
			return revokeFamily(tx, token.FamilyID, now, "vom Client widerrufen")
		}
		return tx.Model(&Token{}).Where("hash = ?", hash).Update("revoked_at", now).Error
	})
}

// RevokeClient entzieht einer App den Zugriff: alle Zulassungen und alle
// Tokenketten. Die Registrierung bleibt; eine neue Anmeldung fragt wieder in
// der App nach.
func (s *Store) RevokeClient(ctx context.Context, clientID string, now time.Time) error {
	return s.write(ctx, func(tx *gorm.DB) error {
		var n int64
		if err := tx.Model(&Client{}).Where("id = ?", clientID).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("die App %s ist nicht registriert", clientID)
		}
		if err := tx.Model(&Grant{}).Where("client_id = ? AND revoked_at IS NULL", clientID).
			Update("revoked_at", now).Error; err != nil {
			return err
		}
		var families []string
		if err := tx.Model(&Family{}).Where("client_id = ? AND revoked_at IS NULL", clientID).
			Pluck("id", &families).Error; err != nil {
			return err
		}
		for _, id := range families {
			if err := revokeFamily(tx, id, now, "Zugriff in der App entzogen"); err != nil {
				return err
			}
		}
		return nil
	})
}

// ActiveGrant liefert die gültige Zulassung einer App für einen Mandanten oder nil.
func (s *Store) ActiveGrant(ctx context.Context, clientID, tenantID string) (*Grant, error) {
	var g Grant
	err := s.db.WithContext(ctx).
		Where("client_id = ? AND tenant_id = ? AND revoked_at IS NULL", clientID, tenantID).
		Order("granted_at DESC").Take(&g).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// touch merkt den letzten Zugriff einer App.
func (s *Store) touch(ctx context.Context, clientID string, at time.Time) error {
	return s.write(ctx, func(tx *gorm.DB) error {
		return tx.Model(&Client{}).Where("id = ?", clientID).Update("last_used_at", at).Error
	})
}

// ConnectedApp ist eine zugelassene App, wie die Einstellungen sie zeigen.
type ConnectedApp struct {
	ClientID   string
	Name       string
	Channel    string
	ApprovedAt time.Time
	LastUsedAt *time.Time
	Tenants    []GrantedTenant
	// Revoked heißt: keine gültige Zulassung mehr.
	Revoked bool
}

// GrantedTenant ist ein Mandant, für den eine App zugelassen ist.
type GrantedTenant struct {
	ID        string
	Name      string
	Scopes    []string
	GrantedAt time.Time
}

// ConnectedApps listet die Apps, denen je zugestimmt wurde, die jüngsten zuerst.
func (s *Store) ConnectedApps(ctx context.Context) ([]ConnectedApp, error) {
	var clients []Client
	if err := s.db.WithContext(ctx).Where("first_approved_at IS NOT NULL").Find(&clients).Error; err != nil {
		return nil, err
	}
	var grants []Grant
	if err := s.db.WithContext(ctx).Where("revoked_at IS NULL").Order("granted_at").Find(&grants).Error; err != nil {
		return nil, err
	}
	byClient := map[string][]GrantedTenant{}
	for _, g := range grants {
		byClient[g.ClientID] = append(byClient[g.ClientID], GrantedTenant{
			ID: g.TenantID, Name: g.TenantName, Scopes: strings.Fields(g.Scopes), GrantedAt: g.GrantedAt,
		})
	}
	var active []string
	if err := s.db.WithContext(ctx).Model(&Family{}).Where("revoked_at IS NULL").
		Distinct().Pluck("client_id", &active).Error; err != nil {
		return nil, err
	}
	hasFamily := map[string]bool{}
	for _, id := range active {
		hasFamily[id] = true
	}
	apps := make([]ConnectedApp, 0, len(clients))
	for _, c := range clients {
		app := ConnectedApp{
			ClientID: c.ID, Name: c.Name, ApprovedAt: *c.FirstApprovedAt, LastUsedAt: c.LastUsedAt,
			Tenants: byClient[c.ID],
		}
		if c.Slug != nil {
			app.Channel = "agent:" + *c.Slug
		}
		app.Revoked = len(app.Tenants) == 0 && !hasFamily[c.ID]
		apps = append(apps, app)
	}
	sort.SliceStable(apps, func(i, j int) bool { return apps[i].ApprovedAt.After(apps[j].ApprovedAt) })
	return apps, nil
}

// CountConnected zählt die Apps mit mindestens einer gültigen Zulassung.
func (s *Store) CountConnected(ctx context.Context) (int, error) {
	var n int64
	err := s.db.WithContext(ctx).Model(&Grant{}).Where("revoked_at IS NULL").
		Distinct("client_id").Count(&n).Error
	return int(n), err
}

// prune räumt Abgelaufenes ab: Codes und Tokens einen Tag nach Ablauf,
// Registrierungen ohne Zustimmung nach 30 Tagen. Eine App, die ihre
// Registrierung länger aufhebt, ohne je zugelassen worden zu sein, muss sich
// danach neu registrieren.
func (s *Store) prune(ctx context.Context, now time.Time) error {
	cutoff := now.Add(-24 * time.Hour)
	return s.write(ctx, func(tx *gorm.DB) error {
		if err := tx.Where("expires_at < ?", cutoff).Delete(&Code{}).Error; err != nil {
			return err
		}
		if err := tx.Where("expires_at < ?", cutoff).Delete(&Token{}).Error; err != nil {
			return err
		}
		return tx.Where("first_approved_at IS NULL AND created_at < ?", now.AddDate(0, 0, -30)).Delete(&Client{}).Error
	})
}
