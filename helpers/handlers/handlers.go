package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/lager/v3"
)

var handlersLogger = helpers.InitLoggerFromConfig(&helpers.LoggingConfig{Level: "error"}, "helpers.handlers")

func WriteJSONResponse(w http.ResponseWriter, statusCode int, jsonObj interface{}) {
	logger := handlersLogger.Session("WriteJSONResponse", lager.Data{"json": jsonObj, "statusCode": statusCode})
	jsonBytes, err := json.Marshal(jsonObj)
	if err != nil {
		logger.Error("marshall-json-response", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(jsonBytes)))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, err = w.Write(jsonBytes)
	if err != nil {
		logger.Error("write-json-response", err)
	}
}
