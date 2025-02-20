package models

import (
	"crypto/tls"

	"code.cloudfoundry.org/tlsconfig"
)

type BasicAuth struct {
	Username     string `yaml:"username" json:"username"`
	UsernameHash string `yaml:"username_hash" json:"usernameHash"`
	Password     string `yaml:"password" json:"password"`
	PasswordHash string `yaml:"password_hash" json:"passwordHash"`
}

type TLSCerts struct {
	KeyFile    string `yaml:"key_file" json:"key_file"`
	CertFile   string `yaml:"cert_file" json:"cert_file"`
	CACertFile string `yaml:"ca_file" json:"ca_file"`
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
