package server

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/appstate"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/appstateobserver/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"
	"code.cloudfoundry.org/lager/v3"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tedsuo/ifrit"
)

type Server struct {
	logger lager.Logger
	conf   *config.Config
	source appstate.Source
}

func NewServer(logger lager.Logger, conf *config.Config, source appstate.Source) *Server {
	return &Server{logger: logger, conf: conf, source: source}
}

func (s *Server) Router(am auth.XFCCAuthMiddleware) http.Handler {
	router := mux.NewRouter()
	router.Use(am.XFCCAuthenticationMiddleware)
	router.HandleFunc("/v1/changes", func(w http.ResponseWriter, r *http.Request) {
		after, err := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
		if err != nil || after < 0 {
			http.Error(w, "invalid cursor", http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		changes, err := s.source.Changes(ctx, after)
		if err != nil {
			s.logger.Error("read-changes", err)
			http.Error(w, "observer unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(changes)
	}).Methods(http.MethodGet)
	router.HandleFunc("/v1/apps/{appid}/invalidate", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		if err := s.source.Invalidate(ctx, mux.Vars(r)["appid"]); err != nil {
			s.logger.Error("invalidate", err)
			http.Error(w, "observer unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}).Methods(http.MethodPost)
	return router
}

func (s *Server) CreateCFServer(am auth.XFCCAuthMiddleware) (ifrit.Runner, error) {
	return helpers.NewHTTPServer(s.logger, s.conf.CFServer, s.Router(am))
}

func (s *Server) CreateHealthServer() (ifrit.Runner, error) {
	lag := func() float64 {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		reader, ok := s.source.(interface {
			ReaderCheckedAt(context.Context) (time.Time, error)
		})
		if !ok {
			return math.Inf(1)
		}
		checkedAt, err := reader.ReaderCheckedAt(ctx)
		if err != nil {
			return math.Inf(1)
		}
		return max(0, time.Since(checkedAt).Seconds())
	}
	// Reader health is independent of the HTTP server's liveness.
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "autoscaler_app_state_reader_lag_seconds", Help: "Age of the last completely consumed CF audit window."}, lag))
	check := func() healthendpoint.ReadinessCheck {
		status := "UP"
		if lag() > max(30, (3*s.conf.PollInterval).Seconds()) {
			status = "DOWN"
		}
		return healthendpoint.ReadinessCheck{Name: "cf-audit-journal", Type: "event-reader", Status: status}
	}
	router, err := healthendpoint.NewHealthRouter(s.conf.Health, []healthendpoint.Checker{check}, s.logger, registry, time.Now)
	if err != nil {
		return nil, err
	}
	return helpers.NewHTTPServer(s.logger, s.conf.Health.ServerConfig, router)
}
