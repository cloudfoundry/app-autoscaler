package auth

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/blend/go-sdk/envoyutil"
)

var ErrorMTLSHeaderNotFound = errors.New("mTLS authentication method not found")
var ErrorDecodingFailed = errors.New("certificate decoding in XFCC header failed")
var ErrorLoadingCAFailed = errors.New("loading ca cert failed")
var ErrorNoAppIDFound = errors.New("certificate does not contain an app id")
var ErrorAppIDWrong = errors.New("app id in certificate is not valid")
var ErrorExpired = errors.New("certificate expired")
var ErrorUnknownCA = errors.New("unknown certificate authority(CA)")
var ErrorParsingXFCCHeaderFailed = errors.New("certificate parsing in XFCC header failed")
var ErrorMalformedCertificate = errors.New("malformed certificate")
var ErrorCertificateExpired = x509.CertificateInvalidError{}

func (a *Auth) XFCCAuth(w http.ResponseWriter, r *http.Request, appID string) error {
	if r.Header.Get("X-Forwarded-Client-Cert") == "" {
		return ErrorMTLSHeaderNotFound
	}

	w.Header().Set("Content-Type", "application/json")

	XFCCCert, err := envoyutil.ParseXFCC(r.Header.Get("X-Forwarded-Client-Cert"))

	if err != nil {
		return ErrorMalformedCertificate
	}

	block, _ := pem.Decode([]byte(XFCCCert[0].Cert))
	if block == nil {
		return ErrorDecodingFailed
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return ErrorParsingXFCCHeaderFailed
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

	ca, err := a.LoadCACert()
	if err != nil {
		return ErrorLoadingCAFailed
	}

	_, err = a.IsCertificateAuthorityValid(cert, ca)
	if err != nil {
		if errors.As(err, &ErrorCertificateExpired) {
			return ErrorExpired
		}
		return ErrorUnknownCA
	}
	return nil
}

func (a *Auth) IsCertificateAuthorityValid(cert, ca *x509.Certificate) (bool, error) {
	roots := x509.NewCertPool()
	roots.AddCert(ca)
	opts := x509.VerifyOptions{
		Roots: roots,
	}
	opts.Roots = roots
	_, err := cert.Verify(opts)
	return true, err
}

func (a *Auth) LoadCACert() (*x509.Certificate, error) {
	file, err := ioutil.ReadFile(a.metricsForwarderMtlsCACert)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(file)
	if block == nil {
		return nil, errors.New("unable to decode pem")
	}

	ca, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	return ca, nil
}
