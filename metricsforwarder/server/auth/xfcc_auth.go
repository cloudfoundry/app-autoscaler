package auth

import (
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var ErrorMTLSHeaderNotFound = errors.New("mTLS authentication method not found")
var ErrorNoAppIDFound = errors.New("certificate does not contain an app id")
var ErrorAppIDWrong = errors.New("app id in certificate is not valid")

func (a *Auth) XFCCAuth(r *http.Request, appID string) error {
	xfccHeader := r.Header.Get("X-Forwarded-Client-Cert")
	if xfccHeader == "" {
		return ErrorMTLSHeaderNotFound
	}

	data, err := base64.StdEncoding.DecodeString(removeQuotes(xfccHeader))
	if err != nil {
		return fmt.Errorf("base64 parsing failed: %w", err)
	}

	cert, err := x509.ParseCertificate(data)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	certAppId := getAppId(cert)

	if len(certAppId) == 0 {
		return ErrorNoAppIDFound
	}

	if appID != certAppId {
		return ErrorAppIDWrong
	}

	_, err = cert.Verify(*a.certVerifyOpts)
	if err != nil {
		return fmt.Errorf("cert CA check failed: %w", err)
	}

	return nil
}

func getAppId(cert *x509.Certificate) string {
	var certAppId string
	for _, ou := range cert.Subject.OrganizationalUnit {
		if strings.Contains(ou, "app:") {
			certAppId = strings.Split(ou, ":")[1]
			break
		}
	}
	return certAppId
}

func removeQuotes(xfccHeader string) string {
	if xfccHeader[0] == '"' {
		xfccHeader = xfccHeader[1 : len(xfccHeader)-1]
	}
	return xfccHeader
}
