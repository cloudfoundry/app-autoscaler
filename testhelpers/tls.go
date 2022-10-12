package testhelpers

import (
	"crypto/tls"
	"path/filepath"

	"code.cloudfoundry.org/tlsconfig"
)

const testCertDir = "../../../test-certs"

func ServerTlsConfig(serverName string) *tls.Config {
	return ServerTlsConfigFiles(serverName+".crt", serverName+".key")
}

func ServerTlsConfigFiles(certFile, keyFile string) *tls.Config {
	config, err := tlsconfig.Build(
		tlsconfig.WithIdentityFromFile(
			filepath.Join(testCertDir, certFile),
			filepath.Join(testCertDir, keyFile),
		),
	).
		Server(tlsconfig.WithClientAuthenticationFromFile("autoscaler-ca.crt"))
	FailOnError("Creating server tls config failed", err)
	return config
}
