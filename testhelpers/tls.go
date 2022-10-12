package testhelpers

import (
	"crypto/tls"
	"path/filepath"

	"code.cloudfoundry.org/tlsconfig"
)

func ServerTlsConfig(serverName string) *tls.Config {
	return ServerTlsConfigFiles(serverName+".crt", serverName+".key")
}

func ServerTlsConfigFiles(certFile, keyFile string) *tls.Config {
	certFolder := testCertFolder()
	config, err := tlsconfig.
		Build(
			tlsconfig.WithIdentityFromFile(
				filepath.Join(certFolder, certFile),
				filepath.Join(certFolder, keyFile),
			),
		).Server(tlsconfig.WithClientAuthenticationFromFile("autoscaler-ca.crt"))
	FailOnError("Creating server tls config failed", err)
	return config
}
