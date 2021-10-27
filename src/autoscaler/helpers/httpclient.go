package helpers

import (
	"autoscaler/models"
	"crypto/tls"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/cfhttp"
)

type AutoClients interface {
	Api() *http.Client
	Broker() *http.Client
	Plain() *http.Client
	MetricServer() *http.Client
	EventGenerator() *http.Client
}

var _ AutoClients = &clientsHolder{}
var testCertDir = "../../../../../test-certs"

type clientsHolder struct {
	api            *http.Client
	broker         *http.Client
	plain          *http.Client
	metricServer   *http.Client
	eventGenerator *http.Client
}

var autoClients AutoClients = &clientsHolder{}

func Clients() AutoClients {
	return autoClients
}

func (c clientsHolder) Api() *http.Client {
	if c.api == nil {
		tlsConfig, err := cfhttp.NewTLSConfig(
			filepath.Join(testCertDir, "api.crt"),
			filepath.Join(testCertDir, "api.key"),
			filepath.Join(testCertDir, "autoscaler-ca.crt"))
		if err != nil {
			panic(err)
		}
		c.api = &http.Client{Transport: NewTransport(tlsConfig)}
	}
	return c.api
}

func (c clientsHolder) Broker() *http.Client {
	if c.broker == nil {
		tlsConfig, err := cfhttp.NewTLSConfig(
			filepath.Join(testCertDir, "servicebroker.crt"),
			filepath.Join(testCertDir, "servicebroker.key"),
			filepath.Join(testCertDir, "autoscaler-ca.crt"))
		if err != nil {
			panic(err)
		}
		c.broker = &http.Client{Transport: NewTransport(tlsConfig)}
	}
	return c.broker
}

func (c clientsHolder) Plain() *http.Client {
	if c.plain == nil {
		c.plain = &http.Client{Transport: NewTransport(nil)}
	}
	return c.plain
}

func (c clientsHolder) MetricServer() *http.Client {
	if c.metricServer == nil {
		tlsConfig, err := cfhttp.NewTLSConfig(
			filepath.Join(testCertDir, "metricserver.crt"),
			filepath.Join(testCertDir, "metricserver.key"),
			filepath.Join(testCertDir, "autoscaler-ca.crt"))
		if err != nil {
			panic(err)
		}
		c.metricServer = &http.Client{Transport: NewTransport(tlsConfig)}
	}
	return c.metricServer
}

func (c clientsHolder) EventGenerator() *http.Client {
	if c.eventGenerator == nil {
		tlsConfig, err := cfhttp.NewTLSConfig(
			filepath.Join(testCertDir, "eventgenerator.crt"),
			filepath.Join(testCertDir, "eventgenerator.key"),
			filepath.Join(testCertDir, "autoscaler-ca.crt"))
		if err != nil {
			panic(err)
		}
		c.eventGenerator = &http.Client{Transport: NewTransport(tlsConfig)}
	}
	return c.eventGenerator
}

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

func NewTransport(tlsConfig *tls.Config) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig:       tlsConfig,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100000,
		MaxIdleConnsPerHost:   100000,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}
