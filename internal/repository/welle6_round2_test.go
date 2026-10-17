package repository

import (
	"context"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// --- Protokollkette und Transaktionen -------------------------------------

// Ein Protokolleintrag innerhalb einer laufenden Transaktion darf nicht auf eine
// Sperre warten, die ihrerseits auf diese Transaktion wartet.
//
// Der Fall: ein Weg außerhalb einer Transaktion hält die Anhänge-Sperre und
// wartet auf die Schreibsperre der Datei, die eine laufende Buchungstransaktion
// hält; die schreibt in sich noch ihren Protokolleintrag und wartet dafür auf
// die Anhänge-Sperre. Beide kämen erst über den busy_timeout wieder frei — nach
// fünf Sekunden, und einer der beiden Einträge ginge verloren, weil fast jede
// Aufrufstelle den Fehler des Protokollierens verwirft.
func TestAuditLogInsideATransactionDoesNotWaitForTheOutsideWriter(t *testing.T) {
	// Eine Datei und kein Speicher: die Schreibsperre von SQLite ist der
	// Gegenstand des Tests, und im Speicher gibt es sie so nicht.
	db, err := InitTenantDB(t.TempDir())
	if err != nil {
		t.Fatalf("Testdatenbank anlegen: %v", err)
	}
	audit := NewAuditRepository(db)
	runner := NewTxRunner(db)

	holdsWriteLock := make(chan struct{})
	inner := make(chan error, 1)
	outer := make(chan error, 1)

	go func() {
		inner <- runner.RunInTx(context.Background(), func(ctx context.Context) error {
			// Erst schreiben: damit hält die Transaktion die Schreibsperre.
			if err := audit.Log(ctx, domain.AuditActionCreate, "TEST", "1", "erste Schreiboperation"); err != nil {
				return err
			}
			close(holdsWriteLock)
			// Der Schreiber draußen soll in dieser Zeit die Sperre nehmen und
			// an der Schreibsperre hängen bleiben.
			time.Sleep(300 * time.Millisecond)
			return audit.Log(ctx, domain.AuditActionUpdate, "TEST", "1", "Eintrag aus der Transaktion")
		})
	}()

	<-holdsWriteLock
	go func() {
		outer <- audit.Log(context.Background(), domain.AuditActionCreate, "TEST", "2", "Eintrag von draußen")
	}()

	deadline := time.After(3 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case err := <-inner:
			if err != nil {
				t.Fatalf("der Eintrag aus der Transaktion ist gescheitert: %v", err)
			}
		case err := <-outer:
			if err != nil {
				t.Fatalf("der Eintrag von draußen ist gescheitert: %v", err)
			}
		case <-deadline:
			t.Fatal("die beiden Schreiber blockieren sich gegenseitig: nach drei Sekunden ist erst einer durch")
		}
	}

	entries, err := audit.FindAllAscending(context.Background())
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	written := 0
	for i := range entries {
		if entries[i].EntityType == "TEST" {
			written++
		}
	}
	if written != 3 {
		t.Errorf("%d der drei Protokolleinträge sind angekommen — einer ist verloren gegangen", written)
	}
	result := accounting.NewAuditChain().Verify(entries)
	if !result.IsValid {
		t.Errorf("die Protokollkette ist gebrochen: %s", result.Message)
	}
}
