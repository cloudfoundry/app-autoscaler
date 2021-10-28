package auth

import (
	"autoscaler/db"
	"autoscaler/metricsforwarder/server/common"
	"autoscaler/models"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"
	"github.com/patrickmn/go-cache"
)

var ErrorAuthNotFound = errors.New("authentication method not found")
var ErrCAFileEmpty = errors.New("CA file empty")

type Auth struct {
	logger          lager.Logger
	credentialCache cache.Cache
	policyDB        db.PolicyDB
	cacheTTL        time.Duration
	certVerifyOpts  *x509.VerifyOptions
}

func New(logger lager.Logger, policyDB db.PolicyDB, credentialCache cache.Cache, cacheTTL time.Duration, metricsForwarderMtlsCACert string) (*Auth, error) {
	opts, err := loadMtlsAuth(metricsForwarderMtlsCACert)
	if err != nil {
		return nil, fmt.Errorf("error loading certs from %s: %w", metricsForwarderMtlsCACert, err)
	}

	return &Auth{
		logger:          logger,
		credentialCache: credentialCache,
		policyDB:        policyDB,
		cacheTTL:        cacheTTL,
		certVerifyOpts:  opts,
	}, nil
}

func loadMtlsAuth(metricsForwarderMtlsCACert string) (*x509.VerifyOptions, error) {
	if metricsForwarderMtlsCACert == "" {
		return nil, nil
	}
	caCerts, err := loadCACert(metricsForwarderMtlsCACert)
	if err != nil {
		return nil, fmt.Errorf("loading cert failed: %w", err)
	}

	if len(caCerts) == 0 {
		return nil, ErrCAFileEmpty
	}

	roots := x509.NewCertPool()
	for _, cert := range caCerts {
		roots.AddCert(cert)
	}

	opts := &x509.VerifyOptions{Roots: roots}
	return opts, nil
}

func loadCACert(filename string) ([]*x509.Certificate, error) {
	restOfFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("could not load mtls cert %s: %w", filename, err)
	}

	var block *pem.Block
	var caCerts []*x509.Certificate
	for {
		block, restOfFile = pem.Decode(restOfFile)
		if block == nil {
			break
		}

		ca, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse local mtls cert: %w", err)
		}
		caCerts = append(caCerts, ca)
	}

	return caCerts, nil
}

func (a *Auth) Authenticate(next http.Handler) http.Handler {
	return common.VarsFunc(a.AuthenticateHandler(next))
}

func (a *Auth) AuthenticateHandler(next http.Handler) func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
		err := a.CheckAuth(r, vars["appid"])
		if err != nil {
			a.logger.Info("Authentication Failed:", lager.Data{"error": err.Error()})
			if errors.Is(err, ErrorAppIDWrong) {
				handlers.WriteJSONResponse(w, http.StatusForbidden, models.ErrorResponse{
					Code:    http.StatusText(http.StatusForbidden),
					Message: "Unauthorized"})
			} else {
				handlers.WriteJSONResponse(w, http.StatusUnauthorized, models.ErrorResponse{
					Code:    http.StatusText(http.StatusUnauthorized),
					Message: "Unauthorized"})
			}
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (a *Auth) CheckAuth(r *http.Request, appID string) error {
	var errAuth error
	isMtlsConfigured := a.isMtlsConfigured()
	if isMtlsConfigured {
		errAuth = a.XFCCAuth(r, appID)
	}
	if errors.Is(errAuth, ErrorMTLSHeaderNotFound) || !isMtlsConfigured {
		a.logger.Info("Trying basic auth", lager.Data{"error": fmt.Sprintf("%+v", errAuth), "isMtlsConfigured": isMtlsConfigured})
		errAuth = a.BasicAuth(r, appID)
	}
	return errAuth
}

func (a *Auth) isMtlsConfigured() bool {
	return a.certVerifyOpts != nil
}
