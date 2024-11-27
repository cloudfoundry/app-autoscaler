package testhelpers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
)

// generateClientCert generates a client certificate with the specified spaceGUID and orgGUID
// included in the organizational unit string.
func GenerateClientCert(orgGUID, spaceGUID string) ([]byte, error) {
	// Generate a random serial number for the certificate
	//
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// Create a new X.509 certificate template
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{"My Organization"},
			OrganizationalUnit: []string{fmt.Sprintf("space:%s org:%s", spaceGUID, orgGUID)},
		},
	}
	// Generate the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, err
	}

	// Encode the certificate to PEM format
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	return certPEM, nil
}
