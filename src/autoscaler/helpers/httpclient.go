package helpers

import (
	"autoscaler/models"
	"net"
	"net/http"
	"time"

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
		client.Transport.(*http.Transport).DialContext = (&net.Dialer{
			Timeout: 30 * time.Second,
		}).DialContext
	}

	return client, nil
}
