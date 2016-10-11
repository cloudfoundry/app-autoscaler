package server

import (
	"autoscaler/db"
	"autoscaler/models"
	"autoscaler/scalingengine"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"

	"encoding/json"
	"net/http"
	"strconv"
)

const TokenTypeBearer = "bearer"

type ScalingHandler struct {
	logger        lager.Logger
	historyDB     db.HistoryDB
	appLock       *StripedLock
	scalingEngine scalingengine.ScalingEngine
}

func NewScalingHandler(logger lager.Logger, historyDB db.HistoryDB, scalingEngine scalingengine.ScalingEngine) *ScalingHandler {
	return &ScalingHandler{
		logger:        logger.Session("scaling-handler"),
		historyDB:     historyDB,
		appLock:       NewStripedLock(32),
		scalingEngine: scalingEngine,
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

	var newInstances int

	h.appLock.GetLock(appId).Lock()
	newInstances, err = h.scalingEngine.Scale(appId, trigger)
	h.appLock.GetLock(appId).Unlock()

	if err != nil {
		logger.Error("failed-to-scale", err, lager.Data{"trigger": trigger})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Internal-server-error",
			Message: "Error taking scaling action"})
		return
	}

	handlers.WriteJSONResponse(w, http.StatusOK, models.AppEntity{Instances: newInstances})
}

func (h *ScalingHandler) GetScalingHistories(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appid"]
	logger := h.logger.Session("get-scaling-histories", lager.Data{"appId": appId})

	startParam := r.URL.Query()["start"]
	endParam := r.URL.Query()["end"]
	logger.Debug("handling", lager.Data{"start": startParam, "end": endParam})

	var err error
	start := int64(0)
	end := int64(-1)

	if len(startParam) == 1 {
		start, err = strconv.ParseInt(startParam[0], 10, 64)
		if err != nil {
			logger.Error("failed-to-parse-start-time", err, lager.Data{"start": startParam})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad-Request",
				Message: "Error parsing start time"})
			return
		}
	} else if len(startParam) > 1 {
		logger.Error("failed-to-get-start-time", err, lager.Data{"start": startParam})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad-Request",
			Message: "Incorrect start parameter in query string"})
		return
	}

	if len(endParam) == 1 {
		end, err = strconv.ParseInt(endParam[0], 10, 64)
		if err != nil {
			logger.Error("failed-to-parse-end-time", err, lager.Data{"end": endParam})
			handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
				Code:    "Bad-Request",
				Message: "Error parsing end time"})
			return
		}
	} else if len(endParam) > 1 {
		logger.Error("failed-to-get-end-time", err, lager.Data{"end": endParam})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad-Request",
			Message: "Incorrect end parameter in query string"})
		return
	}

	var histories []*models.AppScalingHistory

	histories, err = h.historyDB.RetrieveScalingHistories(appId, start, end)
	if err != nil {
		logger.Error("failed-to-retrieve-histories", err, lager.Data{"start": start, "end": end})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error getting scaling histories from database"})
		return
	}

	var body []byte
	body, err = json.Marshal(histories)
	if err != nil {
		logger.Error("failed-to-marshal", err, lager.Data{"histories": histories})

		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error getting scaling histories from database"})
		return
	}

	w.Write(body)
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

	h.appLock.GetLock(appId).Lock()
	err = h.scalingEngine.SetActiveSchedule(appId, activeSchedule)
	h.appLock.GetLock(appId).Unlock()

	if err != nil {
		h.logger.Error("failed-to-set-active-schedule", err, lager.Data{"activeSchedule": activeSchedule})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error setting active schedule",
		})
	}
}

func (h *ScalingHandler) RemoveActiveSchedule(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appid"]
	scheduleId := vars["scheduleid"]

	logger := h.logger.Session("remove-active-schedule", lager.Data{"appid": appId, "scheduleid": scheduleId})
	logger.Info("handle-active-schedule-end")

	h.appLock.GetLock(appId).Lock()
	err := h.scalingEngine.RemoveActiveSchedule(appId, scheduleId)
	h.appLock.GetLock(appId).Unlock()

	if err != nil {
		logger.Error("failed-to-remove-active-schedule", err)
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error removing active schedule",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
