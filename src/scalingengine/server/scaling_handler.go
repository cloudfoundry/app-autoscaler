package server

import (
	"cf"
	"db"
	"models"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"

	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

const TokenTypeBearer = "bearer"

type ScalingHandler struct {
	cfClient cf.CfClient
	logger   lager.Logger
	database db.PolicyDB
}

func NewScalingHandler(logger lager.Logger, cfc cf.CfClient, database db.PolicyDB) *ScalingHandler {
	return &ScalingHandler{
		cfClient: cfc,
		logger:   logger,
		database: database,
	}
}

func (h *ScalingHandler) HandleScale(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appid"]
	trigger := &models.Trigger{}
	err := json.NewDecoder(r.Body).Decode(trigger)
	if err != nil {
		h.logger.Error("handle-scale-unmarshal-trigger", err)
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad-Request",
			Message: "Incorrect trigger in request body"})
		return
	}

	h.logger.Debug("handle-scale", lager.Data{"appid": appId, "trigger": trigger})

	var newInstances int
	newInstances, err = h.Scale(appId, trigger)
	if err != nil {
		h.logger.Error("handle-scale-perform-scaling-action", err, lager.Data{"appid": appId, "trigger": trigger})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Internal-server-error",
			Message: "Error taking scaling action"})
		return
	}
	handlers.WriteJSONResponse(w, http.StatusOK, models.AppEntity{Instances: newInstances})

}

func (h *ScalingHandler) Scale(appId string, trigger *models.Trigger) (int, error) {
	logger := h.logger.WithData(lager.Data{"appId": appId})

	policy, err := h.database.GetAppPolicy(appId)
	if err != nil {
		logger.Error("scale-get-app-policy", err)
		return -1, err
	}

	instances, err := h.cfClient.GetAppInstances(appId)
	if err != nil {
		logger.Error("scale-get-app-instances", err)
		return -1, err
	}

	var newInstances int
	newInstances, err = h.ComputeNewInstances(instances, trigger.Adjustment, policy.InstanceMin, policy.InstanceMax)
	if err != nil {
		logger.Error("scale-compute-new-instance", err, lager.Data{"instances": instances, "adjustment": trigger.Adjustment, "instanceMin": policy.InstanceMin, "InstanceMax": policy.InstanceMax})
		return -1, err
	}

	logger.Info("Scale", lager.Data{"trigger": trigger, "instanceMin": policy.InstanceMin, "InstanceMax": policy.InstanceMax, "currentInstances": instances, "newInstances": newInstances})
	if newInstances == instances {
		return newInstances, nil
	}

	err = h.cfClient.SetAppInstances(appId, newInstances)
	if err != nil {
		logger.Error("scale-set-app-instances", err, lager.Data{"newInstances": newInstances})
		return -1, err
	}
	return newInstances, nil
}

func (h *ScalingHandler) ComputeNewInstances(currentInstances int, adjustment string, instanceMin int, instanceMax int) (int, error) {
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

	if newInstances < instanceMin {
		newInstances = instanceMin
	} else if newInstances > instanceMax {
		newInstances = instanceMax
	}

	return newInstances, nil
}
