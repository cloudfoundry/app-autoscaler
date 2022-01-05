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
	if tlsCerts != nil {
		//nolint:staticcheck  // SA1019 TODO: https://github.com/cloudfoundry/app-autoscaler-release/issues/548
		tlsConfig, err := cfhttp.NewTLSConfig(tlsCerts.CertFile, tlsCerts.KeyFile, tlsCerts.CACertFile)
		if err != nil {
			return nil, err
		}
		client.Transport.(*http.Transport).TLSClientConfig = tlsConfig
		client.Transport.(*http.Transport).DialContext = (&net.Dialer{
			Timeout: 30 * time.Second,
		}).DialContext
	}

	return client, nil
}
