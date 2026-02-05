package startup

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
)

// CreateAndLoginCFClient creates a CF client and logs in
func CreateAndLoginCFClient(cfConfig *cf.Config, logger lager.Logger, clock clock.Clock) cf.CFClient {
	cfClient, err := cf.NewCFClient(cfConfig, logger.Session("cf"), clock)
	ExitOnError(err, logger, "failed to create cloud foundry client", lager.Data{"API": cfConfig.API})
	err = cfClient.Login()
	ExitOnError(err, logger, "failed to login cloud foundry", lager.Data{"API": cfConfig.API})
	return cfClient
}
