package auth

import (
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"code.cloudfoundry.org/lager/v3"
)

var ErrorWrongSpace = errors.New("space guid is wrong")
var ErrorWrongOrg = errors.New("org guid is wrong")
var ErrXFCCHeaderNotFound = errors.New("xfcc header not found")

type XFCCAuthMiddleware struct {
	logger    lager.Logger
	spaceGuid string
	orgGuid   string
}

func (m *XFCCAuthMiddleware) checkAuth(r *http.Request) error {
	xfccHeader := r.Header.Get("X-Forwarded-Client-Cert")
	if xfccHeader == "" {
		return ErrXFCCHeaderNotFound
	}

	data, err := base64.StdEncoding.DecodeString(removeQuotes(xfccHeader))
	if err != nil {
		return fmt.Errorf("base64 parsing failed: %w", err)
	}

	cert, err := x509.ParseCertificate(data)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	if getSpaceGuid(cert) != m.spaceGuid {
		return ErrorWrongSpace
	}

	if getOrgGuid(cert) != m.orgGuid {
		return ErrorWrongOrg
	}

	return nil
}

func (m *XFCCAuthMiddleware) XFCCAuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := m.checkAuth(r)

		if err != nil {
			m.logger.Error("xfcc-auth-error", err)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func NewXfccAuthMiddleware(logger lager.Logger, orgGuid, spaceGuid string) *XFCCAuthMiddleware {
	return &XFCCAuthMiddleware{
		logger:    logger,
		orgGuid:   orgGuid,
		spaceGuid: spaceGuid,
	}
}

func getSpaceGuid(cert *x509.Certificate) string {
	var certSpaceGuid string
	for _, ou := range cert.Subject.OrganizationalUnit {
		if strings.Contains(ou, "space:") {
			kv := mapFrom(ou)
			certSpaceGuid = kv["space"]
			break
		}
	}
	return certSpaceGuid
}

func mapFrom(input string) map[string]string {
	result := make(map[string]string)

	r := regexp.MustCompile(`(\w+):(\w+-\w+)`)
	matches := r.FindAllStringSubmatch(input, -1)

	for _, match := range matches {
		result[match[1]] = match[2]
	}
	return result
}

func getOrgGuid(cert *x509.Certificate) string {
	var certOrgGuid string
	for _, ou := range cert.Subject.OrganizationalUnit {
		// capture from string k:v with regex
		if strings.Contains(ou, "org:") {
			kv := mapFrom(ou)
			certOrgGuid = kv["org"]
			break
		}
	}
	return certOrgGuid
}

func removeQuotes(xfccHeader string) string {
	if xfccHeader[0] == '"' {
		xfccHeader = xfccHeader[1 : len(xfccHeader)-1]
	}
	return xfccHeader
}
