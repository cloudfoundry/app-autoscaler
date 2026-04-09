package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsgateway/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsgateway/emitter"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tedsuo/ifrit"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

type Server struct {
	logger  lager.Logger
	conf    *config.Config
	emitter emitter.Emitter
}

func NewServer(logger lager.Logger, conf *config.Config, emitter emitter.Emitter) *Server {
	return &Server{
		logger:  logger,
		conf:    conf,
		emitter: emitter,
	}
}

func (s *Server) CreateCFServer(am auth.XFCCAuthMiddleware) (ifrit.Runner, error) {
	router := s.createRouter(am)
	return helpers.NewHTTPServer(s.logger, s.conf.CFServer, router)
}

func (s *Server) createRouter(am auth.XFCCAuthMiddleware) *mux.Router {
	router := mux.NewRouter()

	v1 := router.PathPrefix("/v1").Subrouter()
	v1.Use(otelmux.Middleware("metricsgateway"))
	if am != nil {
		v1.Use(am.XFCCAuthenticationMiddleware)
	}
	v1.HandleFunc("/envelopes", s.handleEnvelopes).Methods(http.MethodPost)

	return router
}

func (s *Server) CreateTestRouter() *mux.Router {
	return s.createRouter(nil)
}

func (s *Server) CreateHealthServer() (ifrit.Runner, error) {
	promRegistry := prometheus.NewRegistry()
	healthendpoint.RegisterCollectors(promRegistry, []prometheus.Collector{}, true, s.logger.Session("metricsgateway-prometheus"))

	healthRouter, err := healthendpoint.NewHealthRouter(s.conf.Health, []healthendpoint.Checker{}, s.logger.Session("health-server"), promRegistry, time.Now)
	if err != nil {
		return nil, fmt.Errorf("failed to create health router: %w", err)
	}

	return helpers.NewHTTPServer(s.logger, s.conf.Health.ServerConfig, healthRouter)
}

func (s *Server) handleEnvelopes(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("failed-to-read-request-body", err)
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var metrics []*models.CustomMetric
	if err := json.Unmarshal(body, &metrics); err != nil {
		s.logger.Error("failed-to-unmarshal-metrics", err)
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	if len(metrics) == 0 {
		http.Error(w, "empty metrics array", http.StatusBadRequest)
		return
	}

	var emitErrors int
	for _, metric := range metrics {
		if err := s.emitter.EmitMetric(metric); err != nil {
			emitErrors++
		}
	}

	if emitErrors > 0 {
		s.logger.Error("partial-emit-failure", fmt.Errorf("failed to emit %d of %d metrics", emitErrors, len(metrics)))
		http.Error(w, fmt.Sprintf("failed to emit %d of %d metrics", emitErrors, len(metrics)), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
