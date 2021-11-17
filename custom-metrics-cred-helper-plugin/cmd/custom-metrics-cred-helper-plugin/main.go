package main

import (
	plugin2 "code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper/plugin"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/custom-metrics-cred-helper-plugin/custom_metrics"

	"github.com/hashicorp/go-plugin"
)

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugin2.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"credHelper": &plugin2.CredentialsPlugin{Impl: &custom_metrics.Credentials{}},
		},

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
