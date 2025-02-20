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

var (
	ErrorWrongSpace       = errors.New("space guid is wrong")
	ErrorWrongOrg         = errors.New("org guid is wrong")
	ErrXFCCHeaderNotFound = errors.New("xfcc header not found")
)

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

func NewXfccAuthMiddleware(logger lager.Logger, xfccAuth models.XFCCAuth) XFCCAuthMiddleware {
	return &xfccAuthMiddleware{
		logger:   logger,
		xfccAuth: &xfccAuth,
	}
}

func (m *xfccAuthMiddleware) XFCCAuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := m.checkAuth(r); err != nil {
			m.logger.Error("xfcc-auth-error", err)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func CheckAuth(r *http.Request, org, space string) error {
	xfccHeader := r.Header.Get("X-Forwarded-Client-Cert")
	if xfccHeader == "" {
		return ErrXFCCHeaderNotFound
	}

	attrs := parseXFCCHeader(xfccHeader)

	data, err := base64.StdEncoding.DecodeString(attrs["Cert"])
	if err != nil {
		return fmt.Errorf("base64 parsing failed: %w", err)
	}

	cert, err := x509.ParseCertificate(data)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	if getSpaceGuid(cert) != space {
		return ErrorWrongSpace
	}

	if getOrgGuid(cert) != org {
		return ErrorWrongOrg
	}

	return nil
}

func (m *xfccAuthMiddleware) checkAuth(r *http.Request) error {
	return CheckAuth(r, m.xfccAuth.ValidOrgGuid, m.xfccAuth.ValidSpaceGuid)
}

func parseXFCCHeader(xfccHeader string) map[string]string {
	attrs := make(map[string]string)
	for _, v := range strings.Split(xfccHeader, ";") {
		attr := strings.SplitN(v, "=", 2)
		if len(attr) == 2 {
			attrs[attr[0]] = attr[1]
		}
	}
	return attrs
}

func getSpaceGuid(cert *x509.Certificate) string {
	return getGuidFromCert(cert, "space:")
}

func getOrgGuid(cert *x509.Certificate) string {
	return getGuidFromCert(cert, "org:")
}

func getGuidFromCert(cert *x509.Certificate, prefix string) string {
	for _, ou := range cert.Subject.OrganizationalUnit {
		if strings.Contains(ou, prefix) {
			kv := mapFrom(ou)
			return kv[strings.TrimSuffix(prefix, ":")]
		}
	}
	return ""
}

func mapFrom(input string) map[string]string {
	result := make(map[string]string)
	r := regexp.MustCompile(`(\w+):((\w+-)*\w+)`)
	matches := r.FindAllStringSubmatch(input, -1)

	for _, match := range matches {
		result[match[1]] = match[2]
	}

	return result
}
