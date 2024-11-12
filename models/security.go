package models

import (
	"crypto/tls"

	"code.cloudfoundry.org/tlsconfig"
)

type BasicAuth struct {
	Username     string `yaml:"username"`
	UsernameHash string `yaml:"username_hash"`
	Password     string `yaml:"password"`
	PasswordHash string `yaml:"password_hash"`
}

type TLSCerts struct {
	KeyFile    string `yaml:"key_file" json:"keyFile"`
	CertFile   string `yaml:"cert_file" json:"certFile"`
	CACertFile string `yaml:"ca_file" json:"caCertFile"`
}

func (t *TLSCerts) CreateClientConfig() (*tls.Config, error) {
	if t != nil && t.CertFile != "" && t.KeyFile != "" {
		clientTls := tlsconfig.Build(
			tlsconfig.WithInternalServiceDefaults(),
			tlsconfig.WithIdentityFromFile(t.CertFile, t.KeyFile))
		if t.CACertFile != "" {
			return clientTls.Client(tlsconfig.WithAuthorityFromFile(t.CACertFile))
		}
		return clientTls.Client()
	}
	return nil, nil
}

func (t *TLSCerts) CreateServerConfig() (*tls.Config, error) {
	if t != nil && t.CertFile != "" && t.KeyFile != "" {
		serverTls := tlsconfig.Build(
			tlsconfig.WithInternalServiceDefaults(),
			tlsconfig.WithIdentityFromFile(t.CertFile, t.KeyFile))
		if t.CACertFile != "" {
			return serverTls.Server(tlsconfig.WithClientAuthenticationFromFile(t.CACertFile))
		}
		return serverTls.Server()
	}
	return nil, nil
}
