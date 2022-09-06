package helpers

import (
	"net"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/cfhttp"
)

func CreateHTTPClient(tlsCerts *models.TLSCerts) (*http.Client, error) {
	if tlsCerts.CertFile == "" || tlsCerts.KeyFile == "" {
		tlsCerts = nil
	}
	//nolint:staticcheck //TODO https://github.com/cloudfoundry/app-autoscaler-release/issues/549
	client := cfhttp.NewClient()

	transport := client.Transport.(*http.Transport)
	transport.DialContext = (&net.Dialer{
		Timeout: 30 * time.Second,
	}).DialContext
	transport.IdleConnTimeout = 5 * time.Second
	transport.MaxIdleConnsPerHost = 200

	if tlsCerts != nil {
		//nolint:staticcheck  // SA1019 TODO: https://github.com/cloudfoundry/app-autoscaler-release/issues/548
		tlsConfig, err := cfhttp.NewTLSConfig(tlsCerts.CertFile, tlsCerts.KeyFile, tlsCerts.CACertFile)
		if err != nil {
			return nil, err
		}
		transport.TLSClientConfig = tlsConfig
	}

	return client, nil
}
