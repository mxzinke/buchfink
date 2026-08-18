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
			DataDir:        filepath.Join(home, ".buchfink", "data"),
			CertPath:       filepath.Join(home, ".buchfink", "certs", "buchfink-cert.pem"),
			HasPassword:    false,
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

	return &cfg, nil
}

func (r *appConfigRepositoryJSON) Save(cfg *domain.AppConfig) error {
	dir := filepath.Dir(r.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	return os.WriteFile(r.configPath, data, 0644)
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
			if strings.HasPrefix(name, "buchfink_") && strings.HasSuffix(name, ".sqlite") {
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
