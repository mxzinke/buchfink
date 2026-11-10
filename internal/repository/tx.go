package repository

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

// Eine Transaktion über mehrere Repositories. Sie reist im Kontext und nicht in
// der Signatur: jede Methode, die `dbFrom` benutzt, läuft in ihr, ohne dass es
// von jeder Methode eine Fassung mit und eine ohne Transaktion gibt.

type txContextKey struct{}

// NewTxRunner returns a runner over one database handle.
func NewTxRunner(db *gorm.DB) domain.TxRunner { return &gormTxRunner{db: db} }

type gormTxRunner struct {
	db *gorm.DB
}

// RunInTx runs fn inside one database transaction.
//
// A context that already carries a transaction runs fn as-is rather than
// opening a nested one: the outer boundary is the one the caller reasoned
// about, and a nested savepoint that rolls back on its own would leave half of
// what the caller meant to be atomic.
func (r *gormTxRunner) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txContextKey{}).(*gorm.DB); ok {
		return fn(ctx)
	}
	return dbFrom(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txContextKey{}, tx))
	})
}

// dbFrom hands back the handle a repository call has to run on: the ambient
// transaction if there is one, otherwise the repository's own connection.
func dbFrom(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txContextKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return db.WithContext(ctx)
}
