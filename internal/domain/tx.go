package domain

import "context"

// TxRunner runs several repository calls inside one database transaction.
//
// It exists for the one place where two records must appear together or not at
// all: the outgoing invoice. Its number, the invoice itself and its booking are
// one act — the number is consecutive only if a failed booking gives it back
// (§ 14 Abs. 4 Nr. 4 UStG, GoBD Rz. 42). Everything else in Buchfink writes one
// record at a time and does not need this.
//
// The interface says nothing about how the transaction is carried; the
// repository layer decides that. What a service sees is: inside fn, every
// repository call belongs to the same transaction, and returning an error
// undoes all of them.
type TxRunner interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
