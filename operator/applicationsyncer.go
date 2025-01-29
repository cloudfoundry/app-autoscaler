package operator

import (
	"context"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/lager/v3"
)

type Operator interface {
	Operate(ctx context.Context)
}

var _ Operator = &ApplicationSynchronizer{}

type ApplicationSynchronizer struct {
	cfClient cf.ContextClient
	policyDb db.PolicyDB
	logger   lager.Logger
}

func NewApplicationSynchronizer(cfClient cf.ContextClient, policyDb db.PolicyDB, logger lager.Logger) *ApplicationSynchronizer {
	return &ApplicationSynchronizer{
		policyDb: policyDb,
		cfClient: cfClient,
		logger:   logger.Session("application-synchronizer"),
	}
}

func (as ApplicationSynchronizer) Operate(ctx context.Context) {
	logger := as.logger.Session("syncing-apps")
	logger.Info("starting")
	defer logger.Info("completed")

	// Get all the application details from policyDB
	appIds, err := as.policyDb.GetAppIds(ctx)
	if err != nil {
		as.logger.Error("failed-to-get-apps", err)
		return
	}
	// For each app check if they really exist or not via CC api call
	for appID := range appIds {
		_, err = as.cfClient.GetApp(ctx, cf.Guid(appID))
		if err != nil {
			as.logger.Error("failed-to-get-app-info", err)
			if cf.IsNotFound(err) {
				// Application does not exist, lets clean up app details from policyDB
				err = as.policyDb.DeletePolicy(ctx, appID)
				if err != nil {
					as.logger.Error("failed-to-prune-non-existent-application-details", err)
					continue
				}
				as.logger.Info("successfully-pruned-non-existent-application", lager.Data{"appid": appID})
			}
		}
	}
}
