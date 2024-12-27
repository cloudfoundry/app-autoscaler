package auth

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3"
)

var ErrorWrongSpace = errors.New("space guid is wrong")
var ErrorWrongOrg = errors.New("org guid is wrong")
var ErrXFCCHeaderNotFound = errors.New("xfcc header not found")

type XFCCAuthMiddleware interface {
	XFCCAuthenticationMiddleware(next http.Handler) http.Handler
}

type Cert struct {
	FullChainPem string
	Sha256       [32]byte
	Base64       string
}

func NewCert(fullChainPem string) *Cert {
	block, _ := pem.Decode([]byte(fullChainPem))
	if block == nil {
		return nil
	}
	return &Cert{
		FullChainPem: fullChainPem,
		Sha256:       sha256.Sum256(block.Bytes),
		Base64:       base64.StdEncoding.EncodeToString(block.Bytes),
	}
}

func (c *Cert) GetXFCCHeader() string {
	return fmt.Sprintf("Hash=%x;Cert=%s", c.Sha256, c.Base64)
}

type xfccAuthMiddleware struct {
	logger   lager.Logger
	xfccAuth *models.XFCCAuth
}

func (m *xfccAuthMiddleware) checkAuth(r *http.Request) error {
	xfccHeader := r.Header.Get("X-Forwarded-Client-Cert")
	if xfccHeader == "" {
		return ErrXFCCHeaderNotFound
	}

	attrs := make(map[string]string)
	for _, v := range strings.Split(xfccHeader, ";") {
		attr := strings.SplitN(v, "=", 2)
		attrs[attr[0]] = attr[1]
	}

	data, err := base64.StdEncoding.DecodeString(attrs["Cert"])
	if err != nil {
		return fmt.Errorf("base64 parsing failed: %w", err)
	}

	cert, err := x509.ParseCertificate(data)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	if m.getSpaceGuid(cert) != m.xfccAuth.ValidSpaceGuid {
		return ErrorWrongSpace
	}

	if m.getOrgGuid(cert) != m.xfccAuth.ValidOrgGuid {
		return ErrorWrongOrg
	}

	return nil
}

func (m *xfccAuthMiddleware) XFCCAuthenticationMiddleware(next http.Handler) http.Handler {
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

func NewXfccAuthMiddleware(logger lager.Logger, xfccAuth models.XFCCAuth) XFCCAuthMiddleware {
	return &xfccAuthMiddleware{
		logger:   logger,
		xfccAuth: &xfccAuth,
	}
}

func (m *xfccAuthMiddleware) getSpaceGuid(cert *x509.Certificate) string {
	var certSpaceGuid string
	for _, ou := range cert.Subject.OrganizationalUnit {
		if strings.Contains(ou, "space:") {
			kv := m.mapFrom(ou)
			certSpaceGuid = kv["space"]
			break
		}
	}
	return certSpaceGuid
}

func (m *xfccAuthMiddleware) mapFrom(input string) map[string]string {
	result := make(map[string]string)

	r := regexp.MustCompile(`(\w+):((\w+-)*\w+)`)
	matches := r.FindAllStringSubmatch(input, -1)

	for _, match := range matches {
		result[match[1]] = match[2]
	}

	m.logger.Debug("parseCertOrganizationalUnit", lager.Data{"input": input, "result": result})
	return result
}

func (m *xfccAuthMiddleware) getOrgGuid(cert *x509.Certificate) string {
	var certOrgGuid string
	for _, ou := range cert.Subject.OrganizationalUnit {
		if strings.Contains(ou, "org:") {
			kv := m.mapFrom(ou)
			certOrgGuid = kv["org"]
			break
		}
	}
	return certOrgGuid
}
