package server

import (
	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/metricscollector/collector"
	"autoscaler/models"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"

	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type MetricHandler struct {
	nodeIndex int
	nodeAdds  []string
	logger    lager.Logger
	queryFunc collector.MetricQueryFunc
}

func NewMetricHandler(logger lager.Logger, nodeIndex int, nodeAdds []string, query collector.MetricQueryFunc) *MetricHandler {
	return &MetricHandler{
		nodeIndex: nodeIndex,
		nodeAdds:  nodeAdds,
		logger:    logger,
		queryFunc: query,
	}
}

func (h *MetricHandler) GetMetricHistories(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appid"]
	if r.URL.Query()["referer"] == nil {
		shardID := helpers.FNVHash(appId) % uint32(len(h.nodeAdds))
		if shardID != uint32(h.nodeIndex) {
			params := r.URL.Query()
			params.Add("referer", h.nodeAdds[h.nodeIndex])
			newURL := "https://" + h.nodeAdds[shardID] + r.URL.Path + "?" + params.Encode()
			h.logger.Debug("get-metric-histories-redirect", lager.Data{"appId": appId, "newURL": newURL})
			http.Redirect(w, r, newURL, http.StatusFound)
			return
		}
	}
	metricType := vars["metrictype"]
	instanceIndexParam := r.URL.Query()["instanceindex"]
	startParam := r.URL.Query()["start"]
	endParam := r.URL.Query()["end"]
	orderParam := r.URL.Query()["order"]

	h.logger.Debug("get-metric-histories", lager.Data{"appId": appId, "metrictype": metricType, "instanceIndex": instanceIndexParam, "start": startParam, "end": endParam, "order": orderParam})

	var err error
	start := int64(0)
	end := int64(-1)
	order := db.ASC
	instanceIndex := int64(-1)

	if len(instanceIndexParam) == 1 {
		instanceIndex, err = strconv.ParseInt(instanceIndexParam[0], 10, 64)
		if err != nil {
			h.logger.Error("get-metric-histories-parse-instance-index", err, lager.Data{"instanceIndex": instanceIndexParam})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad-Request",
				Message: "Error parsing instanceIndex"})
			return
		}
		if instanceIndex < 0 {
			h.logger.Error("get-metric-histories-parse-instance-index", err, lager.Data{"instanceIndex": instanceIndexParam})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad-Request",
				Message: "InstanceIndex must be greater than or equal to 0"})
			return
		}
	} else if len(instanceIndexParam) > 1 {
		h.logger.Error("get-metric-histories-get-instance-index", err, lager.Data{"instanceIndex": instanceIndexParam})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad-Request",
			Message: "Incorrect instanceIndex parameter in query string"})
		return
	}

	if len(startParam) == 1 {
		start, err = strconv.ParseInt(startParam[0], 10, 64)
		if err != nil {
			h.logger.Error("get-metric-histories-parse-start-time", err, lager.Data{"start": startParam})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad-Request",
				Message: "Error parsing start time"})
			return
		}
	} else if len(startParam) > 1 {
		h.logger.Error("get-metric-histories-get-start-time", err, lager.Data{"start": startParam})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad-Request",
			Message: "Incorrect start parameter in query string"})
		return
	}

	if len(endParam) == 1 {
		end, err = strconv.ParseInt(endParam[0], 10, 64)
		if err != nil {
			h.logger.Error("get-metric-histories-parse-end-time", err, lager.Data{"end": endParam})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad-Request",
				Message: "Error parsing end time"})
			return
		}
	} else if len(endParam) > 1 {
		h.logger.Error("get-metric-histories-get-end-time", err, lager.Data{"end": endParam})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad-Request",
			Message: "Incorrect end parameter in query string"})
		return
	}

	if len(orderParam) == 1 {
		orderStr := strings.ToUpper(orderParam[0])
		if orderStr == db.DESCSTR {
			order = db.DESC
		} else if orderStr == db.ASCSTR {
			order = db.ASC
		} else {
			h.logger.Error("get-metric-histories-parse-order", err, lager.Data{"order": orderParam})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad-Request",
				Message: fmt.Sprintf("Incorrect order parameter in query string, the value can only be %s or %s", db.ASCSTR, db.DESCSTR),
			})
			return
		}
	} else if len(orderParam) > 1 {
		h.logger.Error("get-metric-histories-parse-order", err, lager.Data{"order": orderParam})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad-Request",
			Message: "Incorrect order parameter in query string"})
		return
	}

	mtrcs, err := h.queryFunc(appId, int(instanceIndex), metricType, start, end, order)
	if err != nil {
		h.logger.Error("get-metric-histories-retrieve-metrics", err, lager.Data{"appId": appId, "metrictype": metricType, "instanceIndex": instanceIndex, "start": start, "end": end, "order": order})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error getting instance metric histories"})
		return
	}

	var body []byte
	body, err = json.Marshal(mtrcs)
	if err != nil {
		h.logger.Error("get-metric-histories-marshal", err, lager.Data{"appId": appId, "metrictype": metricType, "metrics": mtrcs})

		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error marshal metric histories"})
		return
	}
	w.Write(body)
}
