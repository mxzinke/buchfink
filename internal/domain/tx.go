package domain

import "context"

// TxRunner runs several repository calls inside one database transaction.
//
// Jeder Vorgang, der mehr als einen Datensatz schreibt — Buchung, Vermerke,
// Siegel, Protokolleintrag —, läuft darin ganz oder gar nicht. Die
// Transaktion reist im Kontext; innerhalb von fn gehört jeder
// Repository-Aufruf zu ihr, und ein Fehler nimmt alle zurück.
type TxRunner interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
