package server

import (
	"cf"
	"db"
	"models"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"

	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const TokenTypeBearer = "bearer"

type ScalingHandler struct {
	cfClient  cf.CfClient
	logger    lager.Logger
	policyDB  db.PolicyDB
	historyDB db.HistoryDB
	hClock    clock.Clock
}

func NewScalingHandler(logger lager.Logger, cfc cf.CfClient, policyDB db.PolicyDB, historyDB db.HistoryDB, hClock clock.Clock) *ScalingHandler {
	return &ScalingHandler{
		cfClient:  cfc,
		logger:    logger,
		policyDB:  policyDB,
		historyDB: historyDB,
		hClock:    hClock,
	}
}

func (h *ScalingHandler) HandleScale(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appid"]
	logger := h.logger.WithData(lager.Data{"appId": appId})

	trigger := &models.Trigger{}
	err := json.NewDecoder(r.Body).Decode(trigger)
	if err != nil {
		logger.Error("handle-scale-unmarshal-trigger", err)
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad-Request",
			Message: "Incorrect trigger in request body"})
		return
	}

	logger.Debug("handle-scale", lager.Data{"trigger": trigger})

	var newInstances int
	newInstances, err = h.Scale(appId, trigger)
	if err != nil {
		logger.Error("handle-scale-perform-scaling-action", err, lager.Data{"trigger": trigger})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Internal-server-error",
			Message: "Error taking scaling action"})
		return
	}
	handlers.WriteJSONResponse(w, http.StatusOK, models.AppEntity{Instances: newInstances})

}

func (h *ScalingHandler) Scale(appId string, trigger *models.Trigger) (int, error) {
	logger := h.logger.WithData(lager.Data{"appId": appId})

	history := &models.AppScalingHistory{
		AppId:        appId,
		Timestamp:    h.hClock.Now().UnixNano(),
		ScalingType:  models.ScalingTypeDynamic,
		OldInstances: -1,
		NewInstances: -1,
		Reason:       getScalingReason(trigger),
	}

	defer h.historyDB.SaveScalingHistory(history)

	policy, err := h.policyDB.GetAppPolicy(appId)
	if err != nil {
		logger.Error("scale-get-app-policy", err)
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to get scaling policy"
		return -1, err
	}

	instances, err := h.cfClient.GetAppInstances(appId)
	if err != nil {
		logger.Error("scale-get-app-instances", err)
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to get app instances"
		return -1, err
	}
	history.OldInstances = instances

	var newInstances int
	newInstances, err = h.ComputeNewInstances(instances, trigger.Adjustment)
	if err != nil {
		logger.Error("scale-compute-new-instance", err, lager.Data{"instances": instances, "adjustment": trigger.Adjustment})
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to compute new app instances"
		return -1, err
	}

	if newInstances < policy.InstanceMin {
		newInstances = policy.InstanceMin
		history.Message = fmt.Sprintf("limited by min instances %d", policy.InstanceMin)
	} else if newInstances > policy.InstanceMax {
		newInstances = policy.InstanceMax
		history.Message = fmt.Sprintf("limited by max instances %d", policy.InstanceMax)
	}
	history.NewInstances = newInstances

	if newInstances == instances {
		history.Status = models.ScalingStatusIgnored
		return newInstances, nil
	}

	err = h.cfClient.SetAppInstances(appId, newInstances)
	if err != nil {
		logger.Error("scale-set-app-instances", err, lager.Data{"newInstances": newInstances})
		history.Status = models.ScalingStatusFailed
		history.Error = "failed to set app instances"
		return -1, err
	}

	history.Status = models.ScalingStatusSucceeded
	return newInstances, nil
}

func (h *ScalingHandler) ComputeNewInstances(currentInstances int, adjustment string) (int, error) {
	var newInstances int
	if strings.HasSuffix(adjustment, "%") {
		percentage, err := strconv.ParseFloat(strings.TrimSuffix(adjustment, "%"), 32)
		if err != nil {
			h.logger.Error("compute-new-instance-get-percentage", err, lager.Data{"adjustment": adjustment})
			return -1, err
		}
		newInstances = int(float64(currentInstances)*(1+percentage/100) + 0.5)
	} else {
		step, err := strconv.ParseInt(adjustment, 10, 32)
		if err != nil {
			h.logger.Error("compute-new-instance-get-step", err, lager.Data{"adjustment": adjustment})
			return -1, err
		}
		newInstances = int(step) + currentInstances
	}

	return newInstances, nil
}

func (h *ScalingHandler) GetScalingHistories(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appid"]
	logger := h.logger.WithData(lager.Data{"appId": appId})

	startParam := r.URL.Query()["start"]
	endParam := r.URL.Query()["end"]
	logger.Debug("get-scaling-histories", lager.Data{"start": startParam, "end": endParam})

	var err error
	start := int64(0)
	end := int64(-1)

	if len(startParam) == 1 {
		start, err = strconv.ParseInt(startParam[0], 10, 64)
		if err != nil {
			logger.Error("get-scaling-histories-parse-start-time", err, lager.Data{"start": startParam})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad-Request",
				Message: "Error parsing start time"})
			return
		}
	} else if len(startParam) > 1 {
		logger.Error("get-scaling-histories-get-start-time", err, lager.Data{"start": startParam})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad-Request",
			Message: "Incorrect start parameter in query string"})
		return
	}

	if len(endParam) == 1 {
		end, err = strconv.ParseInt(endParam[0], 10, 64)
		if err != nil {
			logger.Error("get-scaling-histories-parse-end-time", err, lager.Data{"end": endParam})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad-Request",
				Message: "Error parsing end time"})
			return
		}
	} else if len(endParam) > 1 {
		logger.Error("get-scaling-histories-get-end-time", err, lager.Data{"end": endParam})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad-Request",
			Message: "Incorrect end parameter in query string"})
		return
	}

	var histories []*models.AppScalingHistory

	histories, err = h.historyDB.RetrieveScalingHistories(appId, start, end)
	if err != nil {
		logger.Error("get-scaling-history-retrieve-histories", err, lager.Data{"start": start, "end": end})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error getting scaling histories from database"})
		return
	}

	var body []byte
	body, err = json.Marshal(histories)
	if err != nil {
		logger.Error("get-scaling-history-marshal", err, lager.Data{"histories": histories})

		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error getting scaling histories from database"})
		return
	}
	w.Write(body)
}

func getScalingReason(trigger *models.Trigger) string {
	return fmt.Sprintf("%s instance(s) because %s %s %d for %d seconds",
		trigger.Adjustment,
		trigger.MetricType,
		trigger.Operator,
		trigger.Threshold,
		trigger.BreachDurationSeconds)
}
