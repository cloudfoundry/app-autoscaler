package server

import (
	"autoscaler/db"
	"autoscaler/metricsforwarder/forwarder"
	"autoscaler/models"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"
)

type CustomMetricsHandler struct {
	metricForwarder forwarder.MetricForwarder
	policyDB        db.PolicyDB
	logger          lager.Logger
}

func NewCustomMetricsHandler(logger lager.Logger, metricForwarder forwarder.MetricForwarder, policyDB db.PolicyDB) *CustomMetricsHandler {
	return &CustomMetricsHandler{
		metricForwarder: metricForwarder,
		policyDB:        policyDB,
		logger:          logger,
	}
}

func (mh *CustomMetricsHandler) PublishMetrics(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	auth := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(auth) != 2 || auth[0] != "Basic" {
		http.Error(w, "Authorization failed", http.StatusUnauthorized)
		handlers.WriteJSONResponse(w, http.StatusUnauthorized, models.ErrorResponse{
			Code:    "Authorization-Failure-Error",
			Message: "Error varifying user credentials. Basic authorization is not used properly"})
		return
	}
	payload, err := base64.StdEncoding.DecodeString(auth[1]) // Decoding the username:password

	if err != nil {
		mh.logger.Error("error-decoding-authorizaion-header", err, lager.Data{"authheader": auth})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error decoding credentials"})
		return
	}

	appID := vars["appid"]

	pair := strings.SplitN(string(payload), ":", 2)

	if len(pair) != 2 || !mh.policyDB.ValidateCustomMetricsCreds(appID, pair[0], pair[1]) {
		mh.logger.Error("error-validating-authorizaion-header", err, lager.Data{"authheader": auth})
		http.Error(w, "Authorization failed", http.StatusUnauthorized)
		handlers.WriteJSONResponse(w, http.StatusUnauthorized, models.ErrorResponse{
			Code:    "Authorization-Failure-Error",
			Message: "Basic authorization credential does not match"})
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		mh.logger.Error("error-reading-request-body", err, lager.Data{"body": r.Body})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error reading custom metrics request body"})
		return
	}
	var metricsConsumer *models.MetricsConsumer
	err = json.Unmarshal(body, &metricsConsumer)
	if err != nil {
		mh.logger.Error("error-unmarshaling-metrics", err, lager.Data{"body": r.Body})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error unmarshaling custom metrics request body"})
		return
	}
	isValidRequest, err := mh.policyDB.ValidateCustomMetricTypes(appID, metricsConsumer)
	if !isValidRequest && err != nil {
		mh.logger.Error("failed-validating-metrictypes", err, lager.Data{"metrics": metricsConsumer})
		http.Error(w, err.Error(), http.StatusBadRequest)
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad-Request",
			Message: "Error validating custom metrics type"})
		return
	}

	metrics := mh.parseMetrics(metricsConsumer)

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

func (mh *CustomMetricsHandler) parseMetrics(metricsConsumer *models.MetricsConsumer) []*models.CustomMetric {
	var metrics []*models.CustomMetric
	for _, metric := range metricsConsumer.CustomMetrics {
		metrics = append(metrics, &models.CustomMetric{
			AppGUID:       metricsConsumer.AppGUID,
			InstanceIndex: metricsConsumer.InstanceIndex,
			Name:          metric.Name,
			Type:          metric.Type,
			Value:         metric.Value,
			Unit:          metric.Unit,
		})
	}
	return metrics
}
