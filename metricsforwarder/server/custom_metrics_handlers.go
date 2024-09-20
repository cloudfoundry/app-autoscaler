package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/forwarder"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"errors"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/handlers"
	"code.cloudfoundry.org/lager/v3"
	"github.com/patrickmn/go-cache"
)

var (
	ErrorReadingBody       = errors.New("error reading custom metrics request body")
	ErrorUnmarshallingBody = errors.New("error unmarshalling custom metrics request body")
	ErrorParsingBody       = errors.New("error parsing request body")
)

type CustomMetricsHandler struct {
	metricForwarder    forwarder.MetricForwarder
	policyDB           db.PolicyDB
	bindingDB          db.BindingDB
	allowedMetricCache cache.Cache
	cacheTTL           time.Duration
	logger             lager.Logger
}

func NewCustomMetricsHandler(logger lager.Logger, metricForwarder forwarder.MetricForwarder, policyDB db.PolicyDB, bindingDB db.BindingDB, allowedMetricCache cache.Cache) *CustomMetricsHandler {
	return &CustomMetricsHandler{
		metricForwarder:    metricForwarder,
		policyDB:           policyDB,
		bindingDB:          bindingDB,
		allowedMetricCache: allowedMetricCache,
		logger:             logger,
	}
}

func (mh *CustomMetricsHandler) VerifyCredentialsAndPublishMetrics(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appID := vars["appid"]

	err := mh.PublishMetrics(w, r, appID)
	if err != nil {
		if errors.Is(err, ErrorReadingBody) {
			handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
				Code:    "Internal-Server-Error",
				Message: "error reading custom metrics request body"})
		} else if errors.Is(err, ErrorUnmarshallingBody) {
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad-Request",
				Message: "Error unmarshaling custom metrics request body"})
		} else if errors.Is(err, ErrorNoPolicy) {
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad-Request",
				Message: "no policy defined"})
		} else if errors.Is(err, ErrorStdMetricExists) {
			var metricError *Error
			if errors.As(err, &metricError) {
				handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
					Code:    "Bad-Request",
					Message: fmt.Sprintf("Custom Metric: %s matches with standard metrics name", metricError.GetMetricName()),
				})
			}
		} else if errors.Is(err, ErrorMetricNotInPolicy) {
			var metricError *Error
			if errors.As(err, &metricError) {
				handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
					Code:    "Bad-Request",
					Message: fmt.Sprintf("Custom Metric: %s does not match with metrics defined in policy", metricError.GetMetricName()),
				})
			}
		} else if errors.Is(err, ErrorParsingBody) {
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad-Request",
				Message: "Error parsing request body"})
		} else {
			handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
				Code:    "Unexpected Error",
				Message: err.Error()})
		}
		return
	}
}

func (mh *CustomMetricsHandler) PublishMetrics(w http.ResponseWriter, r *http.Request, appID string) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		mh.logger.Error("error-reading-request-body", err, lager.Data{"body": r.Body})
		return ErrorReadingBody
	}
	var metricsConsumer *models.MetricsConsumer
	err = json.Unmarshal(body, &metricsConsumer)
	if err != nil {
		mh.logger.Error("error-unmarshalling-metrics", err, lager.Data{"body": r.Body})
		return ErrorUnmarshallingBody
	}
	err = mh.validateCustomMetricTypes(appID, metricsConsumer)
	if err != nil {
		mh.logger.Error("failed-validating-metric-types", err, lager.Data{"metrics": metricsConsumer})
		return fmt.Errorf("metric validation Failed %w", err)
	}

	metrics := mh.getMetrics(appID, metricsConsumer)

	if len(metrics) <= 0 {
		mh.logger.Debug("failed-parsing-custom-metrics-request-body", lager.Data{"metrics": metrics})
		return ErrorParsingBody
	}

	mh.logger.Debug("custom-metrics-parsed-successfully", lager.Data{"metrics": metrics})
	for _, metric := range metrics {
		mh.metricForwarder.EmitMetric(metric)
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

func (mh *CustomMetricsHandler) validateCustomMetricTypes(appGUID string, metricsConsumer *models.MetricsConsumer) error {
	standardMetricsTypes := make(map[string]struct{})
	standardMetricsTypes[models.MetricNameMemoryUsed] = struct{}{}
	standardMetricsTypes[models.MetricNameMemoryUtil] = struct{}{}
	standardMetricsTypes[models.MetricNameThroughput] = struct{}{}
	standardMetricsTypes[models.MetricNameResponseTime] = struct{}{}
	standardMetricsTypes[models.MetricNameCPU] = struct{}{}
	standardMetricsTypes[models.MetricNameCPUUtil] = struct{}{}
	standardMetricsTypes[models.MetricNameDisk] = struct{}{}
	standardMetricsTypes[models.MetricNameDiskUtil] = struct{}{}

	allowedMetricTypeSet := make(map[string]struct{})
	res, found := mh.allowedMetricCache.Get(appGUID)
	if found {
		// AllowedMetrics found in cache
		allowedMetricTypeSet = res.(map[string]struct{})
	} else {
		// allow app with strategy as bound_app to submit metrics without policy
		isAppWithBoundStrategy, err := mh.isAppWithBoundStrategy(appGUID)
		if err != nil {
			mh.logger.Error("error-finding-app-submission-strategy", err, lager.Data{"appId": appGUID})
			return err
		}
		if isAppWithBoundStrategy {
			mh.logger.Info("app-with-bound-strategy-found", lager.Data{"appId": appGUID})
			return nil
		}

		scalingPolicy, err := mh.policyDB.GetAppPolicy(context.TODO(), appGUID)
		if err != nil {
			mh.logger.Error("error-getting-policy", err, lager.Data{"appId": appGUID})
			return errors.New("not able to get policy details")
		}
		if err == nil && scalingPolicy == nil {
			mh.logger.Debug("no-policy-found", lager.Data{"appId": appGUID})
			return ErrorNoPolicy
		}
		for _, metrictype := range scalingPolicy.ScalingRules {
			allowedMetricTypeSet[metrictype.MetricType] = struct{}{}
		}
		//update the cache
		mh.allowedMetricCache.Set(appGUID, allowedMetricTypeSet, mh.cacheTTL)
	}
	mh.logger.Debug("allowed-metrics-types", lager.Data{"metrics": allowedMetricTypeSet})
	for _, metric := range metricsConsumer.CustomMetrics {
		// check if the custom metric name matches with any standard metrics name
		_, stdMetricsExists := standardMetricsTypes[metric.Name]
		if stdMetricsExists { //it should fail
			mh.logger.Info("custom-metric-name-matches-with-standard-metrics-name", lager.Data{"metric": metric.Name})
			return &Error{metric: metric.Name, err: ErrorStdMetricExists}
		}
		// check if any of the custom metrics not defined during policy binding
		_, ok := allowedMetricTypeSet[metric.Name]
		if !ok { // If any of the custom metrics is not defined during policy binding, it should fail
			mh.logger.Info("unmatched-custom-metric-type", lager.Data{"metric": metric.Name})
			return &Error{metric: metric.Name, err: ErrorMetricNotInPolicy}
		}
	}
	return nil
}

func (mh *CustomMetricsHandler) isAppWithBoundStrategy(appGUID string) (bool, error) {
	// allow app with submission_strategy as bound_app to submit custom metrics even without policy
	submissionStrategy, err := mh.bindingDB.GetCustomMetricStrategyByAppId(context.TODO(), appGUID)
	if err != nil {
		mh.logger.Error("error-getting-custom-metrics-strategy", err, lager.Data{"appId": appGUID})
		return false, err
	}
	if submissionStrategy == "bound_app" {
		mh.logger.Info("bounded-metrics-submission-strategy", lager.Data{"appId": appGUID, "submission_strategy": submissionStrategy})
		return true, nil
	}
	return false, nil
}

func (mh *CustomMetricsHandler) getMetrics(appID string, metricsConsumer *models.MetricsConsumer) []*models.CustomMetric {
	var metrics []*models.CustomMetric
	for _, metric := range metricsConsumer.CustomMetrics {
		metrics = append(metrics, &models.CustomMetric{
			AppGUID:       appID,
			InstanceIndex: metricsConsumer.InstanceIndex,
			Name:          metric.Name,
			Value:         metric.Value,
			Unit:          metric.Unit,
		})
	}
	return metrics
}
