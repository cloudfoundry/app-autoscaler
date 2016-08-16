package server

import (
	"cf"
	"db"
	"metricscollector/metrics"
	"metricscollector/noaa"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"

	"encoding/json"
	"net/http"
	"strconv"
)

const TokenTypeBearer = "bearer"

type MemoryMetricHandler struct {
	cfClient     cf.CfClient
	logger       lager.Logger
	noaaConsumer noaa.NoaaConsumer
	database     db.MetricsDB
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewMemoryMetricHandler(logger lager.Logger, cfc cf.CfClient, consumer noaa.NoaaConsumer, database db.MetricsDB) *MemoryMetricHandler {
	return &MemoryMetricHandler{
		cfClient:     cfc,
		noaaConsumer: consumer,
		logger:       logger,
		database:     database,
	}
}

func (h *MemoryMetricHandler) GetMemoryMetric(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appid"]

	w.Header().Set("Content-Type", "application/json")

	containerMetrics, err := h.noaaConsumer.ContainerMetrics(appId, TokenTypeBearer+" "+h.cfClient.GetTokens().AccessToken)
	if err != nil {
		h.logger.Error("Get-memory-metric-from-noaa", err, lager.Data{"appId": appId})

		handlers.WriteJSONResponse(w, http.StatusInternalServerError, ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error getting memory metrics from doppler"})
		return
	}
	h.logger.Debug("Get-memory-metric-from-noaa", lager.Data{"appId": appId, "container-metrics": containerMetrics})

	metric := metrics.GetMemoryMetricFromContainerMetrics(appId, containerMetrics)
	var body []byte
	body, err = json.Marshal(metric)
	if err != nil {
		h.logger.Error("get-memory-metrics-marshal", err, lager.Data{"appId": appId, "metric": metric})

		handlers.WriteJSONResponse(w, http.StatusInternalServerError, ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error getting memory metrics from doppler"})
		return
	}

	w.Write(body)
}

func (h *MemoryMetricHandler) GetMemoryMetricHistory(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appid"]
	startParam := r.URL.Query()["start"]
	endParam := r.URL.Query()["end"]
	h.logger.Debug("get-memeory-metric-history", lager.Data{"appId": appId, "start": startParam, "end": endParam})

	var err error
	start := int64(0)
	end := int64(-1)

	if len(startParam) == 1 {
		start, err = strconv.ParseInt(startParam[0], 10, 64)
		if err != nil {
			h.logger.Error("get-memory-metric-history-parse-start-time", err, lager.Data{"start": startParam})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, ErrorResponse{
				Code:    "Bad-Request",
				Message: "Error parsing start time"})
			return
		}
	} else if len(startParam) > 1 {
		h.logger.Error("get-memory-metric-history-get-start-time", err, lager.Data{"start": startParam})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, ErrorResponse{
			Code:    "Bad-Request",
			Message: "Incorrect start parameter in query string"})
		return
	}

	if len(endParam) == 1 {
		end, err = strconv.ParseInt(endParam[0], 10, 64)
		if err != nil {
			h.logger.Error("get-memory-metric-history-parse-end-time", err, lager.Data{"end": endParam})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, ErrorResponse{
				Code:    "Bad-Request",
				Message: "Error parsing end time"})
			return
		}
	} else if len(endParam) > 1 {
		h.logger.Error("get-memory-metric-history-get-end-time", err, lager.Data{"end": endParam})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, ErrorResponse{
			Code:    "Bad-Request",
			Message: "Incorrect end parameter in query string"})
		return
	}

	var mtrcs []*metrics.Metric

	mtrcs, err = h.database.RetrieveMetrics(appId, metrics.MetricNameMemory, start, end)
	if err != nil {
		h.logger.Error("get-memmory-history-retrieve-metrics", err, lager.Data{"appId": appId, "start": start, "end": end})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error getting memory metrics history from database"})
		return
	}

	var body []byte
	body, err = json.Marshal(mtrcs)
	if err != nil {
		h.logger.Error("get-memory-metric-history-marshal", err, lager.Data{"appId": appId, "metrics": mtrcs})

		handlers.WriteJSONResponse(w, http.StatusInternalServerError, ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error getting memory metrics history from database"})
		return
	}
	w.Write(body)
}
