// SPDX-FileCopyrightText: 2026 Maximilian P.
// SPDX-License-Identifier: EUPL-1.2

package timestamp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

// TestRequestAndVerifyRoundTrip hits the real DigiCert TSA. It is skipped in
// -short mode and tolerates network failures (CI/offline) so it never flakes the
// suite, but proves the end-to-end request → store → offline-verify path when
// the network is available.
func TestRequestAndVerifyRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network TSA test in short mode")
	}
	sum := sha256.Sum256([]byte("buchfink chain head 2026"))
	hashHex := hex.EncodeToString(sum[:])

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	res, err := RequestToken(ctx, DefaultTSA, hashHex)
	if err != nil {
		t.Skipf("TSA unreachable, skipping: %v", err)
	}
	if len(res.Token) == 0 || res.GenTime.IsZero() {
		t.Fatalf("empty token or time: %+v", res)
	}
	t.Logf("timestamp from %q at %s", res.TSAName, res.GenTime.Format(time.RFC3339))

	// Offline verification against the correct hash must succeed.
	v, err := VerifyToken(res.Token, hashHex)
	if err != nil {
		t.Fatalf("VerifyToken (correct hash): %v", err)
	}
	if !v.GenTime.Equal(res.GenTime) {
		t.Fatalf("verified time %s != issued time %s", v.GenTime, res.GenTime)
	}

	// Verification against a different hash must fail (tamper detection).
	other := sha256.Sum256([]byte("tampered"))
	if _, err := VerifyToken(res.Token, hex.EncodeToString(other[:])); err == nil {
		t.Fatal("expected verification failure for a different hash")
	}
}

func TestVerifyTokenGarbage(t *testing.T) {
	if _, err := VerifyToken([]byte("not-a-token"), "00"); err == nil {
		t.Fatal("expected error parsing garbage token")
	}
}
