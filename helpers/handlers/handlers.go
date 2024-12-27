package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/lager/v3"
)

var handlersLogger = helpers.InitLoggerFromConfig(&helpers.LoggingConfig{Level: "error"}, "helpers.handlers")

func WriteJSONResponse(w http.ResponseWriter, statusCode int, jsonObj interface{}) {
	logger := handlersLogger.Session("WriteJSONResponse", lager.Data{"json": jsonObj, "statusCode": statusCode})

	jsonBytes := new(bytes.Buffer)
	jsonEncoder := json.NewEncoder(jsonBytes)
	jsonEncoder.SetEscapeHTML(false)
	err := jsonEncoder.Encode(jsonObj)

	// json.Encoder adds a newline in the end as it is typically used to encode multiple objects
	// however we only use it to be able to turn off escaping HTML characters
	result := bytes.TrimRight(jsonBytes.Bytes(), "\n")

	if err != nil {
		logger.Error("marshall-json-response", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(result)))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, err = w.Write(result)
	if err != nil {
		logger.Error("write-json-response", err)
	}
}
