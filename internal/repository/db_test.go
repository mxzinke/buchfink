package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

func TestTenantDBSetsBusyTimeout(t *testing.T) {
	db, err := InitTenantDB(t.TempDir())
	if err != nil {
		t.Fatalf("Mandantendatenbank: %v", err)
	}
	t.Cleanup(func() { _ = CloseDB(db) })

	var timeout int64
	if err := db.Raw("PRAGMA busy_timeout").Scan(&timeout).Error; err != nil {
		t.Fatalf("busy_timeout lesen: %v", err)
	}
	if timeout != BusyTimeout.Milliseconds() {
		t.Errorf("busy_timeout = %d ms, erwartet %d ms", timeout, BusyTimeout.Milliseconds())
	}
}

// Belegt, dass der Treiber `_txlock=immediate` auswertet: eine Transaktion, die
// bisher nur gelesen hat, hält die Schreibsperre schon, und ein zweiter
// Schreiber kommt erst nach ihrem Ende zum Zug.
func TestTenantDBBeginsTransactionsImmediate(t *testing.T) {
	db, err := InitTenantDB(t.TempDir())
	if err != nil {
		t.Fatalf("Mandantendatenbank: %v", err)
	}
	t.Cleanup(func() { _ = CloseDB(db) })
	ctx := context.Background()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	other, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatalf("zweite Verbindung: %v", err)
	}
	defer other.Close()
	if _, err := other.ExecContext(ctx, "PRAGMA busy_timeout = 50"); err != nil {
		t.Fatal(err)
	}

	write := func() error {
		_, err := other.ExecContext(ctx,
			"INSERT INTO setting_items (key, value) VALUES ('txlock_probe', '1') "+
				"ON CONFLICT(key) DO UPDATE SET value = excluded.value")
		return err
	}

	inTx := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- db.Transaction(func(tx *gorm.DB) error {
			var n int64
			if err := tx.Model(&domain.SettingItem{}).Count(&n).Error; err != nil {
				return err
			}
			close(inTx)
			<-release
			return nil
		})
	}()

	<-inTx
	err = write()
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "locked") {
		close(release)
		<-done
		t.Fatalf("eine lesende Transaktion muss die Schreibsperre halten (BEGIN IMMEDIATE); Schreiben ergab %v", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("Transaktion: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		err = write()
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "locked") || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("nach dem Ende der Transaktion muss der zweite Schreiber zum Zug kommen: %v", err)
	}
}
