package operator

import (
	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/models"

	"code.cloudfoundry.org/lager"
)

type ApplicationSynchronizer struct {
	cfClient cf.CFClient
	policyDb db.PolicyDB
	logger   lager.Logger
}

func NewApplicationSynchronizer(cfClient cf.CFClient, policyDb db.PolicyDB, logger lager.Logger) *ApplicationSynchronizer {
	return &ApplicationSynchronizer{
		policyDb: policyDb,
		cfClient: cfClient,
		logger:   logger,
	}
}

func (as ApplicationSynchronizer) Operate() {
	as.logger.Debug("deleting non-existent application details")
	// Get all the application details from policyDB
	appIds, err := as.policyDb.GetAppIds()
	if err != nil {
		as.logger.Error("failed-to-get-apps", err)
		return
	}
	// For each app check if they really exist or not via CC api call
	for appID := range appIds {
		_, err = as.cfClient.GetApp(appID)
		if err != nil {
			as.logger.Error("failed-to-get-app-info", err)
			_, ok := err.(*models.AppNotFoundErr)
			if ok {
				// Application does not exist, lets clean up app details from policyDB
				err = as.policyDb.DeletePolicy(appID)
				if err != nil {
					as.logger.Error("failed-to-prune-non-existent-application-details", err)
					return
				}
				as.logger.Info("successfully-pruned-non-existent-applcation", lager.Data{"appid": appID})
			}
		}
	}
}
