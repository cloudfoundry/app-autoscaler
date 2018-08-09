package helpers

import (
	"autoscaler/models"
	"net/http"

	"code.cloudfoundry.org/cfhttp"
)

func CreateHTTPClient(tlsCerts *models.TLSCerts) (*http.Client, error) {

	if tlsCerts.CertFile == "" || tlsCerts.KeyFile == "" {
		tlsCerts = nil
	}
	client := cfhttp.NewClient()
	if tlsCerts != nil {
		tlsConfig, err := cfhttp.NewTLSConfig(tlsCerts.CertFile, tlsCerts.KeyFile, tlsCerts.CACertFile)
		if err != nil {
			return nil, err
		}
		client.Transport.(*http.Transport).TLSClientConfig = tlsConfig
	}

	return client, nil
}
