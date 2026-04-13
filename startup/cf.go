package startup

import (
	"context"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/lager/v3"
)

// CreateAndLoginCFClient creates a CF client and logs in
func CreateAndLoginCFClient(cfConfig *cf.Config, logger lager.Logger) cf.CFClient {
	cfClient, err := cf.NewCFClient(cfConfig, logger.Session("cf"))
	ExitOnError(err, logger, "failed to create cloud foundry client", lager.Data{"API": cfConfig.API})
	err = cfClient.Login(context.Background())
	ExitOnError(err, logger, "failed to login cloud foundry", lager.Data{"API": cfConfig.API})
	return cfClient
}
