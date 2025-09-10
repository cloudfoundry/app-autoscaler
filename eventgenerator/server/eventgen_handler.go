package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/handlers"
	"code.cloudfoundry.org/lager/v3"
)

type EventGenHandler struct {
	logger         lager.Logger
	queryAppMetric aggregator.QueryAppMetricsFunc
}

func NewEventGenHandler(logger lager.Logger, queryAppMetric aggregator.QueryAppMetricsFunc) *EventGenHandler {
	return &EventGenHandler{
		logger:         logger,
		queryAppMetric: queryAppMetric,
	}
}

func (h *EventGenHandler) GetAggregatedMetricHistories(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appID := vars["appid"]
	metricType := vars["metrictype"]
	startParam := r.URL.Query()["start"]
	endParam := r.URL.Query()["end"]
	orderParam := r.URL.Query()["order"]

	h.logger.Debug("get-aggregated-metric-histories", lager.Data{"appid": appID, "metrictype": metricType, "start": startParam, "end": endParam, "order": orderParam})

	var err error
	start := int64(0)
	end := int64(-1)
	order := db.ASC

	if len(startParam) == 1 {
		start, err = strconv.ParseInt(startParam[0], 10, 64)
		if err != nil {
			h.logger.Error("get-aggregated-metric-histories-parse-start-time", err, lager.Data{"start": startParam})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad-Request",
				Message: "Error parsing start time"})
			return
		}
	} else if len(startParam) > 1 {
		h.logger.Error("get-aggregated-metric-histories-get-start-time", err, lager.Data{"start": startParam})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad-Request",
			Message: "Incorrect start parameter in query string"})
		return
	}

	if len(endParam) == 1 {
		end, err = strconv.ParseInt(endParam[0], 10, 64)
		if err != nil {
			h.logger.Error("get-aggregated-metric-histories-parse-end-time", err, lager.Data{"end": endParam})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad-Request",
				Message: "Error parsing end time"})
			return
		}
	} else if len(endParam) > 1 {
		h.logger.Error("get-aggregated-metric-histories-get-end-time", err, lager.Data{"end": endParam})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad-Request",
			Message: "Incorrect end parameter in query string"})
		return
	}

	if len(orderParam) == 1 {
		orderStr := strings.ToUpper(orderParam[0])
		switch orderStr {
		case db.DESCSTR:
			order = db.DESC
		case db.ASCSTR:
			order = db.ASC
		default:
			h.logger.Error("get-aggregated-metric-histories-parse-order", err, lager.Data{"order": orderParam})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad-Request",
				Message: fmt.Sprintf("Incorrect order parameter in query string, the value can only be %s or %s", db.ASCSTR, db.DESCSTR),
			})
			return
		}
	} else if len(orderParam) > 1 {
		h.logger.Error("get-aggregated-metric-histories-parse-order", err, lager.Data{"order": orderParam})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad-Request",
			Message: "Incorrect order parameter in query string"})
		return
	}

	var mtrcs []*models.AppMetric

	mtrcs, err = h.queryAppMetric(appID, metricType, start, end, order)
	if err != nil {
		h.logger.Error("get-aggregated-metric-histories-retrieve-metrics", err, lager.Data{"appid": appID, "metrictype": metricType, "start": start, "end": end, "order": order})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Internal-Server-Error",
			Message: "Error getting aggregated metric histories"})
		return
	}

	var body []byte
	body, err = json.Marshal(mtrcs)
	if err != nil {
		h.logger.Error("get-aggregated-metric-histories-marshal", err, lager.Data{"appid": appID, "metrictype": metricType, "metrics": mtrcs})

		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Internal-Server-Error",
			Message: "Error marshaling aggregated metric histories"})
		return
	}
	_, err = w.Write(body)
	if err != nil {
		h.logger.Error("unable to write body", err)
	}
}
