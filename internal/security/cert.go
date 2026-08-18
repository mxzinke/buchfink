package security

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// GenerateCertificate creates a self-signed GoBD signing certificate and private key.
func GenerateCertificate(certDir, companyName, password string) (certPath, keyPath string, err error) {
	// Resolve to an absolute path so the persisted cert/key locations do not
	// depend on the process working directory (e.g. relative paths in dev mode).
	absCertDir, err := filepath.Abs(certDir)
	if err != nil {
		return "", "", fmt.Errorf("could not resolve certificate dir: %w", err)
	}
	certDir = absCertDir

	if err := os.MkdirAll(certDir, 0700); err != nil {
		return "", "", fmt.Errorf("could not create certificate dir: %w", err)
	}

	certPath = filepath.Join(certDir, "buchfink-cert.pem")
	keyPath = filepath.Join(certDir, "buchfink-key.pem")

	// Generate Ed25519 Keypair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate ed25519 key: %w", err)
	}

	// Create X.509 Certificate Template
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate serial number: %w", err)
	}

	if companyName == "" {
		companyName = "Buchfink Local Accounting"
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{companyName},
			CommonName:   fmt.Sprintf("%s - Buchfink GoBD Identity", companyName),
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour), // 10 years valid
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, pubKey, privKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create certificate: %w", err)
	}

	// Write Certificate PEM
	certOut, err := os.OpenFile(certPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", "", fmt.Errorf("failed to open cert.pem for writing: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return "", "", fmt.Errorf("failed to write cert pem: %w", err)
	}

	// Write Private Key PEM (with optional password encryption or PKCS#8)
	privBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	keyBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privBytes,
	}

	if password != "" {
		// Encrypt PEM if password provided
		keyBlock, err = x509.EncryptPEMBlock(rand.Reader, "ENCRYPTED PRIVATE KEY", privBytes, []byte(password), x509.PEMCipherAES256) //nolint:staticcheck
		if err != nil {
			return "", "", fmt.Errorf("failed to encrypt key: %w", err)
		}
	}

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", "", fmt.Errorf("failed to open key.pem for writing: %w", err)
	}
	defer keyOut.Close()

	if err := pem.Encode(keyOut, keyBlock); err != nil {
		return "", "", fmt.Errorf("failed to write key pem: %w", err)
	}

	return certPath, keyPath, nil
}
