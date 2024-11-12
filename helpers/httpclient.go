package helpers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/lager/v3"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/cfhttp/v2"
)

type TransportWithBasicAuth struct {
	Username string
	Password string
	Base     http.RoundTripper
}

func (t *TransportWithBasicAuth) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

func (t *TransportWithBasicAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	credentials := t.Username + ":" + t.Password
	basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(credentials))
	req.Header.Add("Authorization", basicAuth)
	return t.base().RoundTrip(req)
}

func DefaultClientConfig() cf.ClientConfig {
	return cf.ClientConfig{
		MaxIdleConnsPerHost:     200,
		IdleConnectionTimeoutMs: 5 * 1000,
	}
}

func CreateHTTPClient(ba *models.BasicAuth, config cf.ClientConfig, logger lager.Logger) (*http.Client, error) {
	client := cfhttp.NewClient(
		cfhttp.WithDialTimeout(30*time.Second),
		cfhttp.WithIdleConnTimeout(time.Duration(config.IdleConnectionTimeoutMs)*time.Millisecond),
		cfhttp.WithMaxIdleConnsPerHost(config.MaxIdleConnsPerHost),
	)

	client = cf.RetryClient(config, client, logger)
	client.Transport = &TransportWithBasicAuth{
		Username: ba.Username,
		Password: ba.Password,
	}

	return client, nil
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

	return cf.RetryClient(config, client, logger), nil
}
