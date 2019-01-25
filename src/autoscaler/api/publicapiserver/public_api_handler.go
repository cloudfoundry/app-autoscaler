package publicapiserver

import (
	"autoscaler/api/config"
	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/models"
	"autoscaler/routes"
	"io/ioutil"
	"net/http"
	"os"

	"code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"
)

type PublicApiHandler struct {
	logger                 lager.Logger
	conf                   *config.Config
	policydb               db.PolicyDB
	scalingEngineClient    *http.Client
	metricsCollectorClient *http.Client
	eventGeneratorClient   *http.Client
}

func NewPublicApiHandler(logger lager.Logger, conf *config.Config, policydb db.PolicyDB) *PublicApiHandler {
	seClient, err := helpers.CreateHTTPClient(&conf.ScalingEngine.TLSClientCerts)
	if err != nil {
		logger.Error("failed to create http client for ScalingEngine", err, lager.Data{"scalingengine": conf.ScalingEngine.TLSClientCerts})
		os.Exit(1)
	}
	mcClient, err := helpers.CreateHTTPClient(&conf.MetricsCollector.TLSClientCerts)
	if err != nil {
		logger.Error("failed to create http client for MetricsCollector", err, lager.Data{"metricscollector": conf.MetricsCollector.TLSClientCerts})
		os.Exit(1)
	}
	egClient, err := helpers.CreateHTTPClient(&conf.EventGenerator.TLSClientCerts)
	if err != nil {
		logger.Error("failed to create http client for EventGenerator", err, lager.Data{"eventgenerator": conf.EventGenerator.TLSClientCerts})
		os.Exit(1)
	}
	return &PublicApiHandler{
		logger:                 logger,
		conf:                   conf,
		policydb:               policydb,
		scalingEngineClient:    seClient,
		metricsCollectorClient: mcClient,
		eventGeneratorClient:   egClient,
	}
}

func (h *PublicApiHandler) GetScalingHistories(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]

	h.logger.Info("Get ScalingHistories", lager.Data{"appId": appId})

	parameters, err := parseParameter(r, vars)
	if err != nil {
		h.logger.Error("Bad Request", err, lager.Data{"appId": appId})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: err.Error(),
		})
		return
	}

	path, _ := routes.ScalingEngineRoutes().Get(routes.GetScalingHistoriesRouteName).URLPath("appid", appId)

	url := h.conf.ScalingEngine.ScalingEngineUrl + path.RequestURI() + "?" + parameters.Encode()

	resp, err := h.scalingEngineClient.Get(url)
	if err != nil {
		h.logger.Error("Failed to retrieve scaling history from scaling engine", err, lager.Data{"url": url})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error retrieving scaling history from scaling engine"})
		return
	}
	defer resp.Body.Close()

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("Error occured during parsing scaling histories result", err, lager.Data{"url": url})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error parsing scaling history from scaling engine"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		h.logger.Error("Error occured during getting scaling histories", nil, lager.Data{"statusCode": resp.StatusCode, "body": string(responseData)})
		handlers.WriteJSONResponse(w, resp.StatusCode, models.ErrorResponse{
			Code:    string(resp.StatusCode),
			Message: string(responseData)})
		return
	}

	paginatedResponse, err := paginateResource(responseData, parameters, r)
	if err != nil {
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: err.Error()})
	}

	handlers.WriteJSONResponse(w, resp.StatusCode, paginatedResponse)
}

func (h *PublicApiHandler) GetAggregatedMetricsHistories(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]
	metricType := vars["metricType"]

	h.logger.Info("Get AggregatedMetricHistories", lager.Data{"appId": appId, "metricType": metricType})

	parameters, err := parseParameter(r, vars)
	if err != nil {
		h.logger.Error("Bad Request", err, lager.Data{"appId": appId})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: err.Error(),
		})
		return
	}
	if metricType == "" {
		h.logger.Error("Bad Request", nil, lager.Data{"appId": appId})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: "metrictype is required",
		})
		return
	}

	path, _ := routes.EventGeneratorRoutes().Get(routes.GetAggregatedMetricHistoriesRouteName).URLPath("appid", appId, "metrictype", metricType)

	url := h.conf.EventGenerator.EventGeneratorUrl + path.RequestURI() + "?" + parameters.Encode()

	resp, err := h.metricsCollectorClient.Get(url)
	if err != nil {
		h.logger.Error("Failed to retrieve metrics history from eventgenerator", err, lager.Data{"url": url})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error retrieving metrics history from eventgenerator"})
		return
	}
	defer resp.Body.Close()

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("Error occured during parsing metrics histories result", err, lager.Data{"url": url})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error parsing metric history from eventgenerator"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		h.logger.Error("Error occured during getting metric histories", nil, lager.Data{"statusCode": resp.StatusCode, "body": string(responseData)})
		handlers.WriteJSONResponse(w, resp.StatusCode, models.ErrorResponse{
			Code:    string(resp.StatusCode),
			Message: string(responseData)})
		return
	}

	paginatedResponse, err := paginateResource(responseData, parameters, r)
	if err != nil {
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: err.Error()})
	}

	handlers.WriteJSONResponse(w, resp.StatusCode, paginatedResponse)
}

func (h *PublicApiHandler) GetInstanceMetricsHistories(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	appId := vars["appId"]

	metricType := vars["metricType"]
	instanceIndex := r.URL.Query().Get("instance-index")

	h.logger.Info("Get ScalingHistories", lager.Data{"appId": appId, "metricType": metricType, "instanceIndex": instanceIndex})

	parameters, err := parseParameter(r, vars)
	if err != nil {
		h.logger.Error("Bad Request", err, lager.Data{"appId": appId})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: err.Error(),
		})
		return
	}
	if metricType == "" {
		h.logger.Error("Bad Request", nil, lager.Data{"appId": appId})
		handlers.WriteJSONResponse(w, http.StatusBadRequest, models.ErrorResponse{
			Code:    "Bad Request",
			Message: "metrictype is required",
		})
		return
	}
	if instanceIndex != "" {
		parameters.Add("instance-index", instanceIndex)
	}

	path, _ := routes.MetricsCollectorRoutes().Get(routes.GetMetricHistoriesRouteName).URLPath("appid", appId, "metrictype", metricType)

	url := h.conf.MetricsCollector.MetricsCollectorUrl + path.RequestURI() + "?" + parameters.Encode()

	resp, err := h.metricsCollectorClient.Get(url)
	if err != nil {
		h.logger.Error("Failed to retrieve metrics history from metricscollector", err, lager.Data{"url": url})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error retrieving metrics history from metricscollector"})
		return
	}
	defer resp.Body.Close()

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("Error occured during parsing metrics histories result", err, lager.Data{"url": url})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Error parsing metric history from metricscollector"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		h.logger.Error("Error occured during getting metric histories", nil, lager.Data{"statusCode": resp.StatusCode, "body": string(responseData)})
		handlers.WriteJSONResponse(w, resp.StatusCode, models.ErrorResponse{
			Code:    string(resp.StatusCode),
			Message: string(responseData)})
		return
	}

	paginatedResponse, err := paginateResource(responseData, parameters, r)
	if err != nil {
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: err.Error()})
	}

	handlers.WriteJSONResponse(w, resp.StatusCode, paginatedResponse)
}

func (h *PublicApiHandler) GetApiInfo(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	info, err := ioutil.ReadFile(h.conf.InfoFilePath)
	if err != nil {
		h.logger.Error("Failed to info file", err, lager.Data{"info-file-path": h.conf.InfoFilePath})
		handlers.WriteJSONResponse(w, http.StatusInternalServerError, models.ErrorResponse{
			Code:    "Interal-Server-Error",
			Message: "Failed to load info"})
		return
	}
	w.Write([]byte(info))
}

func (h *PublicApiHandler) GetHealth(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	w.Write([]byte(`{"alive":"true"}`))
}
