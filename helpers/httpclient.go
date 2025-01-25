package helpers

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/lager/v3"
	"github.com/hashicorp/go-retryablehttp"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/cfhttp/v2"
)

type TLSReloadTransport struct {
	Base           http.RoundTripper
	logger         lager.Logger
	tlsCerts       *models.TLSCerts
	certExpiration time.Time

	HTTPClient *http.Client // Internal HTTP client.

}

func (t *TLSReloadTransport) GetCertExpiration() time.Time {
	if t.certExpiration.IsZero() {
		x509Cert, _ := x509.ParseCertificate(t.tlsClientConfig().Certificates[0].Certificate[0])
		t.certExpiration = x509Cert.NotAfter
	}
	return t.certExpiration
}

func (t *TLSReloadTransport) tlsClientConfig() *tls.Config {
	return t.HTTPClient.Transport.(*http.Transport).TLSClientConfig
}

func (t *TLSReloadTransport) setTLSClientConfig(tlsConfig *tls.Config) {
	t.HTTPClient.Transport.(*http.Transport).TLSClientConfig = tlsConfig
}

func (t *TLSReloadTransport) reloadCert() {
	tlsConfig, _ := t.tlsCerts.CreateClientConfig()
	t.setTLSClientConfig(tlsConfig)
	x509Cert, _ := x509.ParseCertificate(t.tlsClientConfig().Certificates[0].Certificate[0])
	t.certExpiration = x509Cert.NotAfter
}

func (t *TLSReloadTransport) certificateExpiringWithin(dur time.Duration) bool {
	return time.Until(t.GetCertExpiration()) < dur
}

func (t *TLSReloadTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// skips if no tls config to reload
	if t.tlsClientConfig() == nil {
		return t.Base.RoundTrip(req)
	}

	// Checks for cert validity within 5m timeframe. See https://docs.cloudfoundry.org/devguide/deploy-apps/instance-identity.html
	if t.certificateExpiringWithin(5 * time.Minute) {
		t.logger.Debug("reloading-cert")
		t.reloadCert()
	} else {
		t.logger.Debug("cert-not-expiring")
	}

	return t.Base.RoundTrip(req)
}

func DefaultClientConfig() cf.ClientConfig {
	return cf.ClientConfig{
		MaxIdleConnsPerHost:     200,
		IdleConnectionTimeoutMs: 5 * 1000,
	}
}

func CreateHTTPSClient(tlsCerts *models.TLSCerts, config cf.ClientConfig, logger lager.Logger) (*http.Client, error) {
	tlsConfig, err := tlsCerts.CreateClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create tls config: %w", err)
	}

	client := cfhttp.NewClient(
		cfhttp.WithTLSConfig(tlsConfig),
		cfhttp.WithDialTimeout(30*time.Second),
		cfhttp.WithIdleConnTimeout(time.Duration(config.IdleConnectionTimeoutMs)*time.Millisecond),
		cfhttp.WithMaxIdleConnsPerHost(config.MaxIdleConnsPerHost),
	)

	retryClient := cf.RetryClient(config, client, logger)

	retryClient.Transport = &TLSReloadTransport{
		Base:     retryClient.Transport,
		logger:   logger,
		tlsCerts: tlsCerts,

		// Send wrapped HTTPClient referente to access tls configuration inside RoundTrip
		// and to abract the TLSReloadTransport from the retryablehttp
		HTTPClient: retryClient.Transport.(*retryablehttp.RoundTripper).Client.HTTPClient,
	}

	return retryClient, nil
}
