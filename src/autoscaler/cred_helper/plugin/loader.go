package plugin

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/custom-metrics-cred-helper-plugin/custom_metrics"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db/sqldb"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/storedprocedure"
	"github.com/hashicorp/go-hclog"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"

	"github.com/hashicorp/go-plugin"
)

type CredHelperOptions string

const (
	CRED_HELPER_DEFAULT          = "default"
	CRED_HELPER_STORED_PROCEDURE = "stored_procedure"
)

type PluginManager struct {
	client *plugin.Client
}

func (p *PluginManager) Kill() {
	if p.client != nil {
		p.client.Kill()
	}
}

func (p *PluginManager) LoadCredentialPlugin(dbConfigs map[db.Name]db.DatabaseConfig, loggingConfig helpers.LoggingConfig, credHelperPlugin string, storedProcedureConfig *models.StoredProcedureConfig) (cred_helper.Credentials, error) {
	switch credHelperPlugin {
	case CRED_HELPER_DEFAULT:
		logger := helpers.InitLoggerFromConfig(&loggingConfig, "default_cred_helper")
		policyDB, err := sqldb.NewPolicySQLDB(dbConfigs[db.PolicyDb], logger.Session("policy-db"))
		if err != nil {
			return nil, err
		}
		return custom_metrics.NewWithPolicyDb(policyDB, custom_metrics.MaxRetry), nil
	case CRED_HELPER_STORED_PROCEDURE:
		logger := helpers.InitLoggerFromConfig(&loggingConfig, "stored_procedure_cred_helper")
		spDb, err := sqldb.NewStoredProcedureSQLDb(*storedProcedureConfig, dbConfigs[db.StoredProcedureDb], logger.Session("stored-procedure-db"))
		if err != nil {
			return nil, err
		}
		return storedprocedure.NewWithStoredProcedureDb(spDb, custom_metrics.MaxRetry, logger), nil
	default:
		if credHelperPlugin == "" {
			return nil, errors.New("credHelperPlugin is not configured")
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
			Cmd:             exec.Command(credHelperPlugin),
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
		credentials := raw.(cred_helper.Credentials)
		err = credentials.InitializeConfig(dbConfigs, loggingConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize plugin %w", err)
		}
		return credentials, nil
	}
	return nil, errors.New("unable to determine credentials implementation")
}

// pluginMap is the map of plugins we can dispense.
var pluginMap = map[string]plugin.Plugin{
	"credHelper": &CredentialsPlugin{},
}
