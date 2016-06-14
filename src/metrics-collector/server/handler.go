package server

import (
	"crypto/tls"
	"encoding/json"
	"metrics-collector/cf"
	"metrics-collector/metrics"
	. "metrics-collector/util"
	"net/http"

	"github.com/cloudfoundry/noaa/consumer"
	"github.com/gorilla/mux"
)

const PATH_MEMORY_METRIC = "/v1/apps/{appid}/metrics/memory"
const ROUTE_NAME_MEMORY_METRIC = "memory-metric"
const TOKEN_TYPE_BEARER = "bearer"

type Handler struct {
	noaa     *consumer.Consumer
	cfClient cf.CfClient
	router   *mux.Router
}

func NewHandler(cfc cf.CfClient) *Handler {

	var handler = Handler{}

	handler.cfClient = cfc

	dopplerUrl := cfc.GetEndpoints().DopplerEndpoint
	Logger.Info("create-noaa-client", map[string]interface{}{"dopplerUrl": dopplerUrl})
	handler.noaa = consumer.New(dopplerUrl, &tls.Config{InsecureSkipVerify: true}, nil)

	r := mux.NewRouter()
	r.Methods("GET").Path(PATH_MEMORY_METRIC).HandlerFunc(handler.GetMemoryMetric).Name(ROUTE_NAME_MEMORY_METRIC)
	handler.router = r
	return &handler
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *Handler) GetMemoryMetric(w http.ResponseWriter, r *http.Request) {
	appId := mux.Vars(r)["appid"]
	containerMetrics, err := h.noaa.ContainerMetrics(appId, TOKEN_TYPE_BEARER+" "+h.cfClient.GetTokens().AccessToken)

	if err == nil {
		Logger.Debug("get-container-metrics-from-doppler", map[string]interface{}{"container-metrics": containerMetrics})

		metric := metrics.GetMemoryMetricFromContainerMetrics(appId, containerMetrics)
		var body []byte
		body, err = json.Marshal(metric)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write(body)
			return
		}
	}
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "application/json")
	w.Write(cf.CreateJsonErrorResponse("Error-Get-Metrics-From-Doppler", err.Error()))
	Logger.Error("failed-to-get-memory-metric", err)
}
