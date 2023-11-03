package server

import (
	"errors"
	"fmt"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/handlers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine"
	"code.cloudfoundry.org/lager/v3"

	"encoding/json"
	"net/http"
)

type ScalingHandler struct {
	logger          lager.Logger
	scalingEngineDB db.ScalingEngineDB
	scalingEngine   scalingengine.ScalingEngine
}

func NewScalingHandler(logger lager.Logger, scalingEngineDB db.ScalingEngineDB, scalingEngine scalingengine.ScalingEngine) *ScalingHandler {
	return &ScalingHandler{
		logger:          logger.Session("scaling-handler"),
		scalingEngineDB: scalingEngineDB,
		scalingEngine:   scalingEngine,
	}
}

func (h *ScalingHandler) Scale(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appid"]
	logger := h.logger.Session("scale", lager.Data{"appId": appId})

	trigger := &models.Trigger{}
	err := json.NewDecoder(r.Body).Decode(trigger)
	if err != nil {
		logger.Error("failed-to-decode", err)
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad-Request",
			Message: "Incorrect trigger in request body"})
		return
	}

	logger.Debug("handling", lager.Data{"trigger": trigger})

	result, err := h.scalingEngine.Scale(appId, trigger)

	if err != nil {
		logger.Error("failed-to-scale", err, lager.Data{"trigger": trigger})

		var cfApiClientErrTypeProxy *cf.CfError
		if errors.As(err, &cfApiClientErrTypeProxy) {
			errorDescription, err := json.Marshal(cfApiClientErrTypeProxy)
			if err != nil {
				logger.Error("failed-to-serialize cf-api-error", err)
			}

			handlers.WriteJSONResponse(w, cfApiClientErrTypeProxy.StatusCode, models.ErrorResponse{
				Code:    "Error on request to the cloud-controller via a cf-client",
				Message: fmt.Sprintf("Error taking scaling action:\n%s", string(errorDescription))})
		} else {
			handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
				Code:    "Internal-server-error",
				Message: "Error taking scaling action"})
		}

		return
	}

	handlers.WriteJSONResponse(w, http.StatusOK, result)
}

func (h *ScalingHandler) StartActiveSchedule(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appid"]
	scheduleId := vars["scheduleid"]

	logger := h.logger.Session("start-active-schedule", lager.Data{"appid": appId, "scheduleid": scheduleId})

	activeSchedule := &models.ActiveSchedule{}
	err := json.NewDecoder(r.Body).Decode(activeSchedule)
	if err != nil {
		logger.Error("failed-to-decode", err)
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad-Request",
			Message: "Incorrect active schedule in request body",
		})
		return
	}

	activeSchedule.ScheduleId = scheduleId
	logger.Info("handling", lager.Data{"activeSchedule": activeSchedule})

	err = h.scalingEngine.SetActiveSchedule(appId, activeSchedule)
	if err != nil {
		h.logger.Error("failed-to-set-active-schedule", err, lager.Data{"activeSchedule": activeSchedule})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Internal-Server-Error",
			Message: "Error setting active schedule",
		})
	}
}

func (h *ScalingHandler) RemoveActiveSchedule(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appid"]
	scheduleId := vars["scheduleid"]

	logger := h.logger.Session("remove-active-schedule", lager.Data{"appid": appId, "scheduleid": scheduleId})
	logger.Info("handle-active-schedule-end")

	err := h.scalingEngine.RemoveActiveSchedule(appId, scheduleId)

	if err != nil {
		logger.Error("failed-to-remove-active-schedule", err)
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Internal-Server-Error",
			Message: "Error removing active schedule"})
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ScalingHandler) GetActiveSchedule(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appid"]

	logger := h.logger.Session("get-active-schedule", lager.Data{"appid": appId})
	logger.Info("handle-active-schedule-get")

	activeSchedule, err := h.scalingEngineDB.GetActiveSchedule(appId)
	if err != nil {
		logger.Error("failed-to-get-active-schedule", err, lager.Data{"appid": appId})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Internal-Server-Error",
			Message: "Error getting active schedule from database"})
		return
	}

	if activeSchedule == nil {
		handlers.WriteJSONResponse(w, http.StatusNotFound, models.ErrorResponse{
			Code:    "Not-Found",
			Message: "Active schedule not found",
		})
		return
	}

	var body []byte
	body, err = json.Marshal(activeSchedule)
	if err != nil {
		logger.Error("failed-to-marshal", err, lager.Data{"activeSchedule": activeSchedule})

		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Internal-Server-Error",
			Message: "Error getting active schedule from database"})
		return
	}

	_, err = w.Write(body)
	if err != nil {
		logger.Error("failed-to-write-body", err)
	}
}
