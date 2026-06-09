package tlscert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writePEMFile(t *testing.T, path string, certs []*x509.Certificate) {
	t.Helper()

	var data []byte
	for _, cert := range certs {
		data = append(data, pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})...)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write pem file: %v", err)
	}
}

func mustGenerateKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return key
}

func TestScanFullchainAndExcludeCA(t *testing.T) {
	dir := t.TempDir()
	privateKey := mustGenerateKey(t)

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(20),
		Subject: pkix.Name{
			CommonName: "Test CA",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(48 * time.Hour),
		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(10),
		Subject: pkix.Name{
			CommonName: "example.com",
		},
		DNSNames:              []string{"example.com", "www.example.com"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		BasicConstraintsValid: true,
		IsCA:                  false,
		KeyUsage:              x509.KeyUsageDigitalSignature,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("create ca: %v", err)
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, caTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("create leaf: %v", err)
	}

	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("parse ca: %v", err)
	}
	leafCert, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatalf("parse leaf: %v", err)
	}

	fullchainPath := filepath.Join(dir, "fullchain.pem")
	writePEMFile(t, fullchainPath, []*x509.Certificate{leafCert, caCert})

	allCerts, err := Scan(ScanOptions{
		Files: []string{fullchainPath},
	})
	if err != nil {
		t.Fatalf("scan all certs: %v", err)
	}
	if len(allCerts) != 2 {
		t.Fatalf("expected 2 certificates, got %d", len(allCerts))
	}

	leafOnly, err := Scan(ScanOptions{
		Files:     []string{fullchainPath},
		ExcludeCA: true,
	})
	if err != nil {
		t.Fatalf("scan leaf only: %v", err)
	}
	if len(leafOnly) != 1 {
		t.Fatalf("expected 1 certificate with excludeCA, got %d", len(leafOnly))
	}
	if leafOnly[0].CommonName != "example.com" {
		t.Fatalf("unexpected common name: %s", leafOnly[0].CommonName)
	}
	if leafOnly[0].DNSNames != "example.com,www.example.com" {
		t.Fatalf("unexpected dns names: %s", leafOnly[0].DNSNames)
	}
	if leafOnly[0].Issuer != "CN=Test CA" {
		t.Fatalf("unexpected issuer: %s", leafOnly[0].Issuer)
	}
}

func TestIssuerNameIncludesCountryOrganizationAndCommonName(t *testing.T) {
	cert := &x509.Certificate{
		Issuer: pkix.Name{
			Country:      []string{"US"},
			Organization: []string{"Let's Encrypt"},
			CommonName:   "R3",
		},
	}
	if got := issuerName(cert); got != "C=US,O=Let's Encrypt,CN=R3" {
		t.Fatalf("unexpected issuer: %s", got)
	}
}
