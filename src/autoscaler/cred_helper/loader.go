package cred_helper

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

type PluginManager struct {
	client *plugin.Client
}

func (p *PluginManager) Kill() {
	if p.client != nil {
		p.client.Kill()
	}
}

func (p *PluginManager) LoadCredentialPlugin(dbConfigs map[db.Name]db.DatabaseConfig, loggingConfig helpers.LoggingConfig, credHelperPath string) (Credentials, error) {
	if credHelperPath == "" {
		return nil, errors.New("credHelperPath is not configured")
	}

	// Create an hclog.Logger
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "Plugin",
		Output: os.Stdout,
		Level:  hclog.Debug,
	})

	p.client = plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         pluginMap,
		Cmd:             exec.Command(credHelperPath),
		Logger:          logger,
	})

	// Connect via RPC
	rpcClient, err := p.client.Client()
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
