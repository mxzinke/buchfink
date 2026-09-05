package repository

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

// Eine Transaktion über mehrere Repositories.
//
// Die Ausgangsrechnung braucht sie: Nummer vergeben, Rechnung speichern und
// Buchung anhängen müssen zusammen gelingen oder zusammen ausbleiben. Vorher
// vergab `Issue` die Nummer in einer eigenen Transaktion und buchte danach —
// scheiterte die Buchung, war die Nummer verbraucht und keine Rechnung trug
// sie. Genau die Lücke, die § 14 Abs. 4 Nr. 4 UStG und die GoBD nicht
// vorsehen.
//
// Die Transaktion reist im Kontext und nicht in der Signatur. Der Weg über die
// Signatur hieße, jede Methode jedes beteiligten Repositories zu verdoppeln —
// eine mit und eine ohne tx —, und die Verdopplung wäre die Stelle, an der die
// beiden Fassungen auseinanderlaufen. Im Kontext greift sie überall, wo eine
// Methode `dbFrom` benutzt, und nirgends sonst.

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
