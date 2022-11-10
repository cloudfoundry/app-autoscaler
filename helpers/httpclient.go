package helpers

import (
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/cfhttp/v2"
)

func CreateHTTPClient(tlsCerts *models.TLSCerts) (*http.Client, error) {
	tlsConfig, err := tlsCerts.CreateClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create tls config: %w", err)
	}
	return cfhttp.NewClient(
		cfhttp.WithTLSConfig(tlsConfig),
		cfhttp.WithDialTimeout(30*time.Second),
		cfhttp.WithIdleConnTimeout(5*time.Second),
		cfhttp.WithMaxIdleConnsPerHost(200),
	), nil
}
