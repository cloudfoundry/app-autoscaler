package testhelpers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

// XFCCPKI models the platform root -> instance intermediate -> caller chain.
type XFCCPKI struct {
	Root   *x509.Certificate
	Issuer *x509.Certificate
	Leaf   *x509.Certificate
	Key    *rsa.PrivateKey
	Config models.XFCCAuth
}

func GenerateXFCCPKI(dir string) (*XFCCPKI, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	pki := &XFCCPKI{Key: key}
	root := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "app-root"},
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
	}
	pki.Root, err = pki.Sign(root, root)
	if err != nil {
		return nil, err
	}
	issuer := *root
	issuer.SerialNumber = big.NewInt(2)
	issuer.Subject.CommonName = "instance-issuer"
	issuer.MaxPathLenZero = true
	pki.Issuer, err = pki.Sign(&issuer, pki.Root)
	if err != nil {
		return nil, err
	}
	leaf := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject: pkix.Name{
			OrganizationalUnit: []string{"organization:validorg", "space:validspace", "app:an-app-id"},
		},
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		NotBefore:   root.NotBefore,
		NotAfter:    root.NotAfter,
	}
	pki.Leaf, err = pki.Sign(leaf, pki.Issuer)
	if err != nil {
		return nil, err
	}
	pki.Config = models.XFCCAuth{
		ValidOrgGuid:     "validorg",
		ValidSpaceGuid:   "validspace",
		SystemCertPath:   filepath.Join(dir, "roots"),
		InstanceCertPath: filepath.Join(dir, "instance.crt"),
	}
	if err := os.MkdirAll(pki.Config.SystemCertPath, 0700); err != nil {
		return nil, err
	}
	if err := WriteXFCCCertificates(filepath.Join(pki.Config.SystemCertPath, "root.pem"), pki.Root); err != nil {
		return nil, err
	}
	if err := WriteXFCCCertificates(pki.Config.InstanceCertPath, pki.Leaf, pki.Issuer); err != nil {
		return nil, err
	}
	return pki, nil
}

func (p *XFCCPKI) Sign(template, parent *x509.Certificate) (*x509.Certificate, error) {
	der, err := x509.CreateCertificate(rand.Reader, template, parent, &p.Key.PublicKey, p.Key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(der)
}

func WriteXFCCCertificates(path string, certs ...*x509.Certificate) error {
	var data []byte
	for _, cert := range certs {
		data = append(data, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})...)
	}
	return os.WriteFile(path, data, 0600)
}
