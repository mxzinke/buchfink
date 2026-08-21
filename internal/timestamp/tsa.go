// Package timestamp obtains and verifies RFC-3161 trusted timestamps over the
// GoBD hash chain head. Only a SHA-256 hash is ever sent to the Time-Stamping
// Authority (TSA) — never accounting data — so no confidential content leaves
// the machine. Timestamps are an optional, additional proof of existence; GoBD
// does not require them.
package timestamp

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	_ "crypto/sha256" // register SHA-256 for crypto.SHA256.New()
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	rfc3161 "github.com/digitorus/timestamp"
)

// DefaultTSA is the default free, public Time-Stamping Authority (no account
// required). FreeTSA (https://freetsa.org/tsr) is a drop-in alternative.
const DefaultTSA = "http://timestamp.digicert.com"

const (
	contentTypeQuery = "application/timestamp-query"
	contentTypeReply = "application/timestamp-reply"
	httpTimeout      = 20 * time.Second
)

// Result carries the outcome of a timestamp request or verification.
type Result struct {
	Token   []byte    // RFC-3161 TimeStampToken (self-contained, re-verifiable offline)
	GenTime time.Time // trusted time asserted by the TSA
	TSAName string    // human-readable TSA identity (from the signing certificate)
}

// RequestToken asks the TSA to timestamp the given hex-encoded SHA-256 hash and
// returns the resulting token plus the trusted time.
func RequestToken(ctx context.Context, tsaURL, sha256Hex string) (*Result, error) {
	if tsaURL == "" {
		tsaURL = DefaultTSA
	}
	digest, err := hex.DecodeString(sha256Hex)
	if err != nil || len(digest) != 32 {
		return nil, fmt.Errorf("invalid sha-256 hex hash")
	}

	nonce, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	req := rfc3161.Request{
		HashAlgorithm: crypto.SHA256,
		HashedMessage: digest,
		Certificates:  true, // include the TSA cert chain so the token verifies offline later
		Nonce:         nonce,
	}
	reqDER, err := req.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal tsa request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, tsaURL, bytes.NewReader(reqDER))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", contentTypeQuery)
	httpReq.Header.Set("Accept", contentTypeReply)

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("tsa request failed (offline?): %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tsa returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read tsa response: %w", err)
	}

	ts, err := rfc3161.ParseResponse(body)
	if err != nil {
		return nil, fmt.Errorf("parse tsa response: %w", err)
	}
	if !bytes.Equal(ts.HashedMessage, digest) {
		return nil, fmt.Errorf("tsa timestamped a different hash than requested")
	}

	return &Result{
		Token:   ts.RawToken,
		GenTime: ts.Time,
		TSAName: tsaName(ts),
	}, nil
}

// VerifyToken re-parses a stored token offline and confirms it covers the given
// hex-encoded SHA-256 hash. It returns the trusted time and TSA identity.
func VerifyToken(token []byte, sha256Hex string) (*Result, error) {
	digest, err := hex.DecodeString(sha256Hex)
	if err != nil {
		return nil, fmt.Errorf("invalid sha-256 hex hash")
	}
	ts, err := rfc3161.Parse(token)
	if err != nil {
		return nil, fmt.Errorf("token is invalid or its signature does not verify: %w", err)
	}
	if !bytes.Equal(ts.HashedMessage, digest) {
		return nil, fmt.Errorf("timestamp does not cover this hash (data changed)")
	}
	return &Result{Token: token, GenTime: ts.Time, TSAName: tsaName(ts)}, nil
}

func tsaName(ts *rfc3161.Timestamp) string {
	if len(ts.Certificates) > 0 {
		if cn := ts.Certificates[0].Subject.CommonName; cn != "" {
			return cn
		}
		if orgs := ts.Certificates[0].Subject.Organization; len(orgs) > 0 {
			return orgs[0]
		}
	}
	return "Unbekannte TSA"
}
