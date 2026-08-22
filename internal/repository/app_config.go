// SPDX-FileCopyrightText: 2026 Maximilian Pfennig
// SPDX-License-Identifier: EUPL-1.2

package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

type appConfigRepositoryJSON struct {
	configPath string
}

func NewAppConfigRepository(baseDir string) domain.AppConfigRepository {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		baseDir = filepath.Join(home, ".buchfink")
	}
	return &appConfigRepositoryJSON{
		configPath: filepath.Join(baseDir, "config.json"),
	}
}

func (r *appConfigRepositoryJSON) Load() (*domain.AppConfig, error) {
	data, err := os.ReadFile(r.configPath)
	if os.IsNotExist(err) {
		home, _ := os.UserHomeDir()
		currentYear := time.Now().Year()
		return &domain.AppConfig{
			Tenants:        []domain.TenantConfig{},
			ActiveTenantID: "",
			DataDir:        filepath.Join(home, ".buchfink", "data"),
			IsConfigured:   false,
			LastFiscalYear: currentYear,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	var cfg domain.AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("could not unmarshal config file: %w", err)
	}

	// Auto-migrate legacy single-tenant configuration to Tenants list
	if cfg.IsConfigured && len(cfg.Tenants) == 0 {
		tenantID := "default"
		cfg.Tenants = []domain.TenantConfig{
			{
				ID:        tenantID,
				Name:      "Hauptmandant",
				DataDir:   cfg.DataDir,
				CreatedAt: time.Now().Format(time.RFC3339),
			},
		}
		cfg.ActiveTenantID = tenantID
		_ = r.Save(&cfg)
	}

	return &cfg, nil
}

func (r *appConfigRepositoryJSON) Save(cfg *domain.AppConfig) error {
	dir := filepath.Dir(r.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}

	// Sync active tenant fields with top-level fields for convenience
	if cfg.ActiveTenantID != "" {
		for _, t := range cfg.Tenants {
			if t.ID == cfg.ActiveTenantID {
				cfg.DataDir = t.DataDir
				break
			}
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	// 0600: the config lists tenant paths; keep it owner-only.
	return os.WriteFile(r.configPath, data, 0600)
}

// DiscoverAvailableFiscalYears scans the data directory for existing SQLite files and ensures the current year is included.
func DiscoverAvailableFiscalYears(dataDir string) []int {
	currentYear := time.Now().Year()
	yearsMap := map[int]bool{
		currentYear: true,
	}

	entries, err := os.ReadDir(dataDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.HasPrefix(name, "buchfink_") && strings.HasSuffix(name, ".sqlite") && name != "buchfink.sqlite" {
				// e.g. "buchfink_2026.sqlite"
				part := strings.TrimPrefix(name, "buchfink_")
				part = strings.TrimSuffix(part, ".sqlite")
				if y, err := strconv.Atoi(part); err == nil && y >= 2000 && y <= 2100 {
					yearsMap[y] = true
				}
			}
		}
	}

	var years []int
	for y := range yearsMap {
		years = append(years, y)
	}
	sort.Ints(years)
	return years
}
