package main

import (
	"autoscaler/api/cred_helper"
	"autoscaler/custom-metrics-cred-helper-plugin/internal"

	"github.com/hashicorp/go-plugin"
)

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: cred_helper.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"credHelper": &cred_helper.CredentialsPlugin{Impl: &internal.Credentials{}},
		},

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
