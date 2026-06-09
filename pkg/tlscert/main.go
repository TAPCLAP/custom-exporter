package tlscert

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Certificate struct {
	File           string
	CommonName     string
	Issuer         string
	DNSNames       string
	ExpirationTime time.Time
}

type ScanOptions struct {
	Directories []string
	Files       []string
	ExcludeCA   bool
}

func Scan(opts ScanOptions) ([]Certificate, error) {
	filePaths, err := collectFilePaths(opts.Directories, opts.Files)
	if err != nil {
		return nil, err
	}

	var certificates []Certificate
	for _, filePath := range filePaths {
		certs, err := parseCertificateFile(filePath, opts.ExcludeCA)
		if err != nil {
			fmt.Printf("Error parsing certificates in %s: %v\n", filePath, err)
			continue
		}
		certificates = append(certificates, certs...)
	}

	return certificates, nil
}

func collectFilePaths(directories, files []string) ([]string, error) {
	seen := make(map[string]struct{})
	var filePaths []string

	addFile := func(path string) {
		if _, exists := seen[path]; exists {
			return
		}
		seen[path] = struct{}{}
		filePaths = append(filePaths, path)
	}

	for _, filePath := range files {
		info, err := os.Stat(filePath)
		if err != nil {
			return nil, fmt.Errorf("error stating file %s: %w", filePath, err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("%s is a directory, use directories option instead", filePath)
		}
		addFile(filePath)
	}

	for _, dirPath := range directories {
		info, err := os.Stat(dirPath)
		if err != nil {
			return nil, fmt.Errorf("error stating directory %s: %w", dirPath, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", dirPath)
		}

		err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			addFile(path)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("error walking directory %s: %w", dirPath, err)
		}
	}

	sort.Strings(filePaths)
	return filePaths, nil
}

func parseCertificateFile(filePath string, excludeCA bool) ([]Certificate, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	var certificates []Certificate
	rest := content
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("error parsing certificate: %w", err)
		}

		if excludeCA && cert.IsCA {
			continue
		}

		certificates = append(certificates, Certificate{
			File:           filePath,
			CommonName:     cert.Subject.CommonName,
			Issuer:         issuerName(cert),
			DNSNames:       strings.Join(cert.DNSNames, ","),
			ExpirationTime: cert.NotAfter,
		})
	}

	return certificates, nil
}

func issuerName(cert *x509.Certificate) string {
	return formatDN(cert.Issuer)
}

func formatDN(name pkix.Name) string {
	var parts []string
	if len(name.Country) > 0 && name.Country[0] != "" {
		parts = append(parts, "C="+name.Country[0])
	}
	if len(name.Organization) > 0 && name.Organization[0] != "" {
		parts = append(parts, "O="+name.Organization[0])
	}
	if name.CommonName != "" {
		parts = append(parts, "CN="+name.CommonName)
	}
	if len(parts) == 0 {
		return name.String()
	}
	return strings.Join(parts, ",")
}
