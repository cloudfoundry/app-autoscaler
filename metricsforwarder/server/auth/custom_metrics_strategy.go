package auth

import (
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/lager/v3"
)

type MetricsSubmissionStrategy interface {
	validate(appId string, submitterAppIdFromCert string, logger lager.Logger, bindingDB db.BindingDB, r *http.Request) error
}

var _ MetricsSubmissionStrategy = &DefaultMetricsSubmissionStrategy{}

type DefaultMetricsSubmissionStrategy struct{}

func (d *DefaultMetricsSubmissionStrategy) validate(appId string, submitterAppIdFromCert string, _ lager.Logger, _ db.BindingDB, _ *http.Request) error {
	// check if appID is same as AppIdFromCert
	if appId != submitterAppIdFromCert {
		return ErrorAppIDWrong
	}
	return nil
}

type BoundedMetricsSubmissionStrategy struct{}

func (c *BoundedMetricsSubmissionStrategy) validate(appToScaleID string, submitterAppIdFromCert string, logger lager.Logger, bindingDB db.BindingDB, r *http.Request) error {
	if appToScaleID != submitterAppIdFromCert {
		return c.verifyMetricSubmissionStrategy(r, logger, bindingDB, submitterAppIdFromCert, appToScaleID)
	}
	return nil
}

func (c *BoundedMetricsSubmissionStrategy) verifyMetricSubmissionStrategy(r *http.Request, logger lager.Logger, bindingDB db.BindingDB, submitterAppIDFromCert string, appToScaleID string) error {
	isAppBound, err := bindingDB.IsAppBoundToSameAutoscaler(r.Context(), submitterAppIDFromCert, appToScaleID)
	if err != nil {
		logger.Error("error-checking-app-bound-to-same-service", err, lager.Data{"metric-submitter-app-id": submitterAppIDFromCert})
		return err
	}
	if !isAppBound {
		logger.Info("app-not-bound-to-same-service", lager.Data{"app-id": submitterAppIDFromCert})
		return ErrorAppNotBound
	}
	return nil
}
