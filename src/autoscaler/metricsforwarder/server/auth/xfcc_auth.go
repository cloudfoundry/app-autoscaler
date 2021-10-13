package auth

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

var ErrorMTLSHeaderNotFound = errors.New("mTLS authentication method not found")
var ErrorDecodingFailed = errors.New("certificate decoding in XFCC header failed")
var ErrorNoAppIDFound = errors.New("certificate does not contain an app id")
var ErrorAppIDWrong = errors.New("app id in certificate is not valid")

func (a *Auth) XFCCAuth(r *http.Request, appID string) error {
	xfccHeader := r.Header.Get("X-Forwarded-Client-Cert")
	if xfccHeader == "" {
		return ErrorMTLSHeaderNotFound
	}
	if xfccHeader[0] == '"' {
		xfccHeader = xfccHeader[1 : len(xfccHeader)-1]
	}

	data, err := base64.StdEncoding.DecodeString(xfccHeader)
	if err != nil {
		return fmt.Errorf("base64 parsing failed: %w", err)
	}

	cert, err := x509.ParseCertificate(data)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	var certAppId string
	for _, ou := range cert.Subject.OrganizationalUnit {
		if strings.Contains(ou, "app:") {
			certAppId = strings.Split(ou, ":")[1]
			break
		}
	}

	if len(certAppId) == 0 {
		return ErrorNoAppIDFound
	}

	if appID != certAppId {
		return ErrorAppIDWrong
	}

	//TODO consider caching this in mem
	certificates, err := a.LoadCACerts()
	if err != nil {
		return fmt.Errorf("loading cert failed: %w", err)
	}

	if len(certificates) == 0 {
		return fmt.Errorf("did not find any certifcate authorities")
	}

	_, err = a.IsCertificateAuthorityValid(cert, certificates)
	if err != nil {
		return fmt.Errorf("cert CA check failed: %w", err)
	}
	return nil
}

func (a *Auth) IsCertificateAuthorityValid(cert *x509.Certificate, certificates []*x509.Certificate) (bool, error) {
	roots := x509.NewCertPool()
	for _, ca := range certificates {
		roots.AddCert(ca)
	}
	opts := x509.VerifyOptions{
		Roots: roots,
	}
	opts.Roots = roots
	_, err := cert.Verify(opts)
	return true, err
}

func (a *Auth) LoadCACerts() ([]*x509.Certificate, error) {
	file, err := ioutil.ReadFile(a.metricsForwarderMtlsCACert)
	if err != nil {
		return nil, fmt.Errorf("could not load mtls cert %s: %w", a.metricsForwarderMtlsCACert, err)
	}

	rest := file
	var certificates []*x509.Certificate

	for len(rest) > 0 {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			return nil, fmt.Errorf("failed to decode local mtls cert: %w", ErrorDecodingFailed)
		}

		ca, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse local mtls cert: %w", err)
		}

		certificates = append(certificates, ca)
	}

	return certificates, nil
}
