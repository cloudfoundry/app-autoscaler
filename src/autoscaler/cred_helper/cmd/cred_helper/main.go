package main

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper/internal"

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
