package testhelpers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"
)

func GenerateClientCertWithPrivateKeyExpiring(orgGUID, spaceGUID string, privateKey *rsa.PrivateKey, notAfter time.Time) ([]byte, error) {
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		NotBefore:    time.Now(),
		NotAfter:     notAfter,
		Subject: pkix.Name{
			Organization:       []string{"My Organization"},
			OrganizationalUnit: []string{fmt.Sprintf("OU=space:%s+OU=organization:%s", spaceGUID, orgGUID)},
			CommonName:         "localhost",
		},
		DNSNames:    []string{"localhost"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}

	if privateKey == nil {
		return nil, fmt.Errorf("private key is nil")
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	return certPEM, nil
}

func GenerateClientCertWithPrivateKey(orgGUID, spaceGUID string, privateKey *rsa.PrivateKey) ([]byte, error) {
	notAfter := time.Now().AddDate(1, 0, 0)
	return GenerateClientCertWithPrivateKeyExpiring(orgGUID, spaceGUID, privateKey, notAfter)
}

func GenerateClientCert(orgGUID, spaceGUID string) ([]byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	return GenerateClientCertWithPrivateKey(orgGUID, spaceGUID, privateKey)
}

func GenerateClientKeyWithPrivateKey(privateKey *rsa.PrivateKey) []byte {
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	return pem.EncodeToMemory(pemBlock)
}

func SetXFCCCertHeader(req *http.Request, orgGuid, spaceGuid string) error {
	xfccClientCert, err := GenerateClientCert(orgGuid, spaceGuid)
	if err != nil {
		return err
	}

	cert := auth.NewCert(string(xfccClientCert))

	req.Header.Add("X-Forwarded-Client-Cert", cert.GetXFCCHeader())
	return nil
}

func GenerateClientCertWithCA(orgGUID, spaceGUID string, caCertPath, caKeyPath string) ([]byte, []byte, error) {
	caCert, caKey, err := loadCA(caCertPath, caKeyPath)
	if err != nil {
		return nil, nil, err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		Subject: pkix.Name{
			Organization:       []string{"My Organization"},
			OrganizationalUnit: []string{fmt.Sprintf("OU=space:%s+OU=organization:%s", spaceGUID, orgGUID)},
			CommonName:         "localhost",
		},
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, caCert, &privateKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	caCertPEM, err := os.ReadFile(caCertPath) // Read CA cert directly
	if err != nil {
		return nil, nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	fullCertChain := append(certPEM, caCertPEM...)
	return fullCertChain, keyPEM, nil
}

func loadCA(certPath, keyPath string) (*x509.Certificate, *rsa.PrivateKey, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil, err
	}

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, err
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, nil, fmt.Errorf("failed to decode CA certificate")
	}
	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, err
	}

	block, _ = pem.Decode(keyPEM)
	if block == nil {
		return nil, nil, fmt.Errorf("failed to decode CA key")
	}
	caKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, err
	}

	return caCert, caKey, nil
}
