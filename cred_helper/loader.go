package cred_helper

import (
	"autoscaler/db"
	"autoscaler/helpers"
	"fmt"
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

func LoadCredentialPlugin(dbConfigs map[db.Name]db.DatabaseConfig, loggingConfig helpers.LoggingConfig) (Credentials, error) {
	// Create an hclog.Logger
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "Plugin",
		Output: os.Stdout,
		Level:  hclog.Debug,
	})

	// We're a host! Start by launching the plugin process.
	// #nosec G101
	credHelper := os.Getenv("CRED_HELPER")
	if credHelper == "" {
		// #nosec G101
		credHelper = "../../../build/custom-metrics-cred-helper-plugin"
	}
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         pluginMap,
		Cmd:             exec.Command(credHelper),
		Logger:          logger,
	})
	// FIXME why is this call closing the rpc session
	//defer client.Kill()

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		logger.Error("failed to create rpcClient", err)
		return nil, fmt.Errorf("failed to create rpcClient %w", err)
	}
	// Request the plugin
	raw, err := rpcClient.Dispense("credHelper")
	if err != nil {
		return nil, fmt.Errorf("failed to dispense plugin %w", err)
	}
	// We should have a customMetricsCredHelper now! This feels like a normal interface
	// implementation but is in fact over an RPC connection.
	credentials := raw.(Credentials)
	err = credentials.InitializeConfig(dbConfigs, loggingConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize plugin %w", err)
	}
	return credentials, nil
}

// pluginMap is the map of plugins we can dispense.
var pluginMap = map[string]plugin.Plugin{
	"credHelper": &CredentialsPlugin{},
}
