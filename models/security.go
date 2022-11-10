package models

import (
	"crypto/tls"

	"code.cloudfoundry.org/tlsconfig"
)

type TLSCerts struct {
	KeyFile    string `yaml:"key_file" json:"keyFile"`
	CertFile   string `yaml:"cert_file" json:"certFile"`
	CACertFile string `yaml:"ca_file" json:"caCertFile"`
}

func (t *TLSCerts) CreateClientConfig() (*tls.Config, error) {
	if t != nil && t.CertFile != "" && t.KeyFile != "" {
		client := tlsconfig.Build(tlsconfig.WithIdentityFromFile(t.CertFile, t.KeyFile))
		if t.CACertFile != "" {
			return client.Client(tlsconfig.WithAuthorityFromFile(t.CACertFile))
		}
		return client.Client()
	}
	return nil, nil
}

func (t *TLSCerts) CreateServerConfig() (*tls.Config, error) {
	if t != nil && t.CertFile != "" && t.KeyFile != "" {
		build := tlsconfig.Build(tlsconfig.WithIdentityFromFile(t.CertFile, t.KeyFile))
		if t.CACertFile != "" {
			return build.Server(tlsconfig.WithClientAuthenticationFromFile(t.CACertFile))
		}
		return build.Server()
	}
	return nil, nil
}
