package collector

import (
	"fmt"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsserver/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

// ServerConfig a type to wrap both the config.ServerConfig and config.Config data.
type ServerConfig struct {
	Port      int             `yaml:"port"`
	TLS       models.TLSCerts `yaml:"tls"`
	NodeAddrs []string        `yaml:"node_addrs"`
	NodeIndex int             `yaml:"node_index"`
}

func FromConfig(config *config.Config) ServerConfig {
	return ServerConfig{
		Port:      config.Server.Port,
		TLS:       config.Server.TLS,
		NodeAddrs: withPort(config.NodeAddrs, config.Server.Port),
		NodeIndex: config.NodeIndex,
	}
}

func withPort(addrs []string, port int) []string {
	serverNodeAddrs := make([]string, len(addrs))
	for i, n := range addrs {
		serverNodeAddrs[i] = fmt.Sprintf("%s:%d", n, port)
	}
	return serverNodeAddrs
}
