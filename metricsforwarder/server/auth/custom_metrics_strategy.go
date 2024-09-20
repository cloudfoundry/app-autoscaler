package auth

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/lager/v3"
	"net/http"
)

type MetricsSubmissionStrategy interface {
	validate(appId string, submitterAppIdFromCert string, logger lager.Logger, bindingDB db.BindingDB, r *http.Request) error
}

var _ MetricsSubmissionStrategy = &DefaultMetricsSubmissionStrategy{}

type DefaultMetricsSubmissionStrategy struct{}

func (d *DefaultMetricsSubmissionStrategy) validate(appId string, submitterAppIdFromCert string, logger lager.Logger, bindingDB db.BindingDB, r *http.Request) error {
	// check if appID is same as AppIdFromCert
	if appId != submitterAppIdFromCert {
		return ErrorAppIDWrong
	}
	return nil
}

type BoundedMetricsSubmissionStrategy struct{}

func (c *BoundedMetricsSubmissionStrategy) validate(appId string, submitterAppIdFromCert string, logger lager.Logger, bindingDB db.BindingDB, r *http.Request) error {
	if appId != submitterAppIdFromCert {
		c.verifyMetricSubmissionStrategy(r, logger, bindingDB, submitterAppIdFromCert, appId)
	}
	return nil
}

func (c *BoundedMetricsSubmissionStrategy) verifyMetricSubmissionStrategy(r *http.Request, logger lager.Logger, bindingDB db.BindingDB, submitterAppCert string, appID string) (bool, error) {

	logger.Info("custom-metrics-submission-strategy-found", lager.Data{"appID": appID, "submitterAppCertID": submitterAppCert})
	// check if the app is bound to same autoscaler instance by check the binding id from the bindingdb
	// if the app is bound to the same autoscaler instance, then allow the request to the next handler i.e publish custom metrics
	isAppBound, err := bindingDB.IsAppBoundToSameAutoscaler(r.Context(), submitterAppCert, appID)
	if err != nil {
		logger.Error("error-checking-app-bound-to-same-service", err, lager.Data{"metric-submitter-app-id": submitterAppCert})
		return false, err
	}
	if isAppBound == false {
		logger.Info("app-not-bound-to-same-service", lager.Data{"app-id": submitterAppCert})
		return false, ErrorAppNotBound
	}
	return true, nil
}
