package repository

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// OpenTenantReader öffnet die Mandantendatei ein zweites Mal, nur zum Lesen.
//
// Jede Verbindung läuft mit `query_only`: ein Lesevorgang, der doch schreiben
// will, scheitert, statt an der Schreibsperre des Kerns vorbei zu schreiben.
// Die Transaktion beginnt aufgeschoben und nimmt deshalb keine Schreibsperre.
// Das Schema legt InitTenantDB an; der Leser setzt es voraus.
func OpenTenantReader(dataDir string) (*gorm.DB, error) {
	dbPath := filepath.Join(dataDir, "buchfink.sqlite")
	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(%d)&_pragma=query_only(1)&_txlock=deferred",
		dbPath, BusyTimeout.Milliseconds())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:  logger.Default.LogMode(logger.Warn),
		NowFunc: nowUTC,
		Plugins: map[string]gorm.Plugin{"buchfink:utc": utcTimes{}},
	})
	if err != nil {
		return nil, fmt.Errorf("die Mandantendatei %s ließ sich nicht zum Lesen öffnen: %w", dbPath, err)
	}
	return db, nil
}

type readOnlyKey struct{}

// RunReadOnly führt fn in einer lesenden Transaktion auf db aus. Alle
// Repository-Aufrufe in fn sehen denselben Stand der Datei. Läuft schon eine
// Transaktion, gehört fn zu ihr.
//
// Zwei Schreibwege bleiben im Lesevorgang offen (siehe writer): der Eintrag
// ins Änderungsprotokoll, weil ein Zugriff auf personenbezogene Daten auch
// beim Lesen protokolliert wird (QUE-02), und ein vom Kursdienst geholter
// Referenzkurs, der kein Buchungsinhalt ist und beim nächsten Mal derselbe
// sein muss.
func RunReadOnly(ctx context.Context, db *gorm.DB, fn func(ctx context.Context) error) error {
	if InTx(ctx) {
		return fn(ctx)
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ctx := context.WithValue(ctx, txContextKey{}, tx)
		return fn(context.WithValue(ctx, readOnlyKey{}, true))
	})
}

// InTx meldet, ob der Kontext eine Transaktion trägt.
func InTx(ctx context.Context) bool {
	_, ok := ctx.Value(txContextKey{}).(*gorm.DB)
	return ok
}

// IsReadOnly meldet, ob der Kontext in einer lesenden Transaktion läuft.
func IsReadOnly(ctx context.Context) bool {
	readOnly, _ := ctx.Value(readOnlyKey{}).(bool)
	return readOnly
}

// writer ist die Verbindung eines der beiden Schreibwege, die auch aus einem
// Lesevorgang heraus schreiben: dort über die eigene Verbindung und eine
// eigene kurze Transaktion, sonst in der laufenden.
func writer(ctx context.Context, db *gorm.DB) *gorm.DB {
	if IsReadOnly(ctx) {
		return db.WithContext(ctx)
	}
	return dbFrom(ctx, db)
}
