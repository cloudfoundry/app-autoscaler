package server

import (
	"autoscaler/db"
	"autoscaler/metricsforwarder/forwarder"
	"autoscaler/models"
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"
	cache "github.com/patrickmn/go-cache"
	"golang.org/x/crypto/bcrypt"
)

type CustomMetricsHandler struct {
	metricForwarder    forwarder.MetricForwarder
	policyDB           db.PolicyDB
	credentialCache    cache.Cache
	allowedMetricCache cache.Cache
	cacheTTL           time.Duration
	logger             lager.Logger
}

func NewCustomMetricsHandler(logger lager.Logger, metricForwarder forwarder.MetricForwarder, policyDB db.PolicyDB, credentialCache cache.Cache, allowedMetricCache cache.Cache, cacheTTL time.Duration) *CustomMetricsHandler {
	return &CustomMetricsHandler{
		metricForwarder:    metricForwarder,
		policyDB:           policyDB,
		credentialCache:    credentialCache,
		allowedMetricCache: allowedMetricCache,
		cacheTTL:           cacheTTL,
		logger:             logger,
	}
}

func (mh *CustomMetricsHandler) PublishMetrics(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	w.Header().Set("Content-Type", "application/json")

	username, password, authOK := r.BasicAuth()

	if authOK == false {
		mh.logger.Info("error-processing-authorizaion-header")
		handlers.WriteJSONResponse(w, http.StatusUnauthorized, models.ErrorResponse{
			Code:    "Authorization-Failure-Error",
			Message: "Incorrect credentials. Basic authorization is not used properly"})
		return
	}

	var isValid bool

	appID := vars["appid"]
	res, found := mh.credentialCache.Get(appID)
	if found {
		// Credentials found in cache
		credentials := res.(*models.Credential)
		isValid = mh.validateCredentials(username, credentials.Username, password, credentials.Password)
	}

	// Credentials not found in cache or
	// stale cache entry with invalid credential found in cache
	// search in the database and update the cache
	if !found || !isValid {
		credentials, err := mh.policyDB.GetCredential(appID)
		if err != nil {
			if err == sql.ErrNoRows {
				mh.logger.Error("no-credential-found-in-db", err, lager.Data{"appID": appID})
				handlers.WriteJSONResponse(w, http.StatusUnauthorized, models.ErrorResponse{
					Code:    "Authorization-Failure-Error",
					Message: "Incorrect credentials. Basic authorization credential does not match"})
				return
			}
			mh.logger.Error("error-during-getting-credentials-from-policyDB", err, lager.Data{"appid": appID})
			handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
				Code:    "Interal-Server-Error",
				Message: "Error getting binding crededntials from policyDB"})
			return
		}
		// update the cache
		mh.credentialCache.Set(appID, credentials, mh.cacheTTL)

		isValid = mh.validateCredentials(username, credentials.Username, password, credentials.Password)
		// If Credentials in DB is not valid
		if !isValid {
			mh.logger.Error("error-validating-authorizaion-header", err)
			handlers.WriteJSONResponse(w, http.StatusUnauthorized, models.ErrorResponse{
				Code:    "Authorization-Failure-Error",
				Message: "Incorrect credentials. Basic authorization credential does not match"})
			return
		}
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		mh.logger.Error("error-reading-request-body", err, lager.Data{"body": r.Body})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error reading custom metrics request body"})
		return
	}
	var metricsConsumer *models.MetricsConsumer
	err = json.Unmarshal(body, &metricsConsumer)
	if err != nil {
		mh.logger.Error("error-unmarshaling-metrics", err, lager.Data{"body": r.Body})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad-Request",
			Message: "Error unmarshaling custom metrics request body"})
		return
	}
	err = mh.validateCustomMetricTypes(appID, metricsConsumer)
	if err != nil {
		mh.logger.Error("failed-validating-metrictypes", err, lager.Data{"metrics": metricsConsumer})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad-Request",
			Message: err.Error()})
		return
	}

	metrics := mh.getMetrics(appID, metricsConsumer)

	if len(metrics) <= 0 {
		mh.logger.Debug("failed-parsing-custom-metrics-request-body", lager.Data{"metrics": metrics})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad-Request",
			Message: "Error parsing request body"})
		return
	}

	mh.logger.Debug("custom-metrics-parsed-successfully", lager.Data{"metrics": metrics})
	for _, metric := range metrics {
		mh.metricForwarder.EmitMetric(metric)
	}
	w.WriteHeader(http.StatusOK)
}

func (mh *CustomMetricsHandler) validateCredentials(username string, usernameHash string, password string, passwordHash string) bool {
	usernameAuthErr := bcrypt.CompareHashAndPassword([]byte(usernameHash), []byte(username))
	passwordAuthErr := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if usernameAuthErr == nil && passwordAuthErr == nil { // password matching successfull
		return true
	}
	mh.logger.Debug("failed-to-authorize-credentials")
	return false
}

func (mh *CustomMetricsHandler) validateCustomMetricTypes(appGUID string, metricsConsumer *models.MetricsConsumer) error {
	standardMetricsTypes := make(map[string]struct{})
	standardMetricsTypes[models.MetricNameMemoryUsed] = struct{}{}
	standardMetricsTypes[models.MetricNameMemoryUtil] = struct{}{}
	standardMetricsTypes[models.MetricNameThroughput] = struct{}{}
	standardMetricsTypes[models.MetricNameResponseTime] = struct{}{}
	standardMetricsTypes[models.MetricNameCPUUtil] = struct{}{}

	allowedMetricTypeSet := make(map[string]struct{})
	res, found := mh.allowedMetricCache.Get(appGUID)
	if found {
		// AllowedMetrics found in cache
		allowedMetricTypeSet = res.(map[string]struct{})
	} else {
		//  AllowedMetrics not found in cache, find AllowedMetrics from Database
		scalingPolicy, err := mh.policyDB.GetAppPolicy(appGUID)
		if err != nil {
			mh.logger.Error("error-getting-policy", err, lager.Data{"appId": appGUID})
			return errors.New("not able to get policy details")
		}
		if err == nil && scalingPolicy == nil {
			mh.logger.Debug("no-policy-found", lager.Data{"appId": appGUID})
			return errors.New("no policy defined")
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
			return errors.New("Custom Metric: " + metric.Name + " matches with standard metrics name")
		}
		// check if any of the custom metrics not defined during policy binding
		_, ok := allowedMetricTypeSet[metric.Name]
		if !ok { // If any of the custom metrics is not defined during policy binding, it should fail
			mh.logger.Info("unmatched-custom-metric-type", lager.Data{"metric": metric.Name})
			return errors.New("Custom Metric: " + metric.Name + " does not match with metrics defined in policy")
		}
	}
	return nil
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