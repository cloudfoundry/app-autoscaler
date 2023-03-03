package helpers

import (
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/lager/v3"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/cfhttp/v2"
)

func DefaultClientConfig() cf.ClientConfig {
	return cf.ClientConfig{
		MaxIdleConnsPerHost:     200,
		IdleConnectionTimeoutMs: 5 * 1000,
	}
}
func CreateHTTPClient(tlsCerts *models.TLSCerts, config cf.ClientConfig, logger lager.Logger) (*http.Client, error) {
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
