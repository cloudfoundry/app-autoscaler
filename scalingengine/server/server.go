package server

import (
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/apis/scalinghistory"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/schedule"

	"code.cloudfoundry.org/lager/v3"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tedsuo/ifrit"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

	"fmt"
	"net/http"
)

type VarsFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string)

func (vh VarsFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vh(w, r, vars)
}

type Server struct {
	logger              lager.Logger
	conf                *config.Config
	policyDB            db.PolicyDB
	scalingEngineDB     db.ScalingEngineDB
	schedulerDB         db.SchedulerDB
	scalingEngine       scalingengine.ScalingEngine
	synchronizer        schedule.ActiveScheduleSychronizer
	httpStatusCollector healthendpoint.HTTPStatusCollector

	autoscalerRouter *routes.Router
	healthRouter     *mux.Router
}

func NewServer(logger lager.Logger, conf *config.Config, policyDB db.PolicyDB, scalingEngineDB db.ScalingEngineDB, schedulerDB db.SchedulerDB, scalingEngine scalingengine.ScalingEngine, synchronizer schedule.ActiveScheduleSychronizer) *Server {
	return &Server{
		logger:              logger,
		conf:                conf,
		policyDB:            policyDB,
		scalingEngineDB:     scalingEngineDB,
		schedulerDB:         schedulerDB,
		scalingEngine:       scalingEngine,
		synchronizer:        synchronizer,
		httpStatusCollector: healthendpoint.NewHTTPStatusCollector("autoscaler", "scalingengine"),

		autoscalerRouter: routes.NewRouter(),
	}
}

func (s *Server) CreateHealthServer() (ifrit.Runner, error) {
	if err := s.createHealthRouter(); err != nil {
		return nil, fmt.Errorf("failed to create health router: %w", err)
	}

	return helpers.NewHTTPServer(s.logger, s.conf.Health.ServerConfig, s.healthRouter)
}

func (s *Server) CreateCFServer(am auth.XFCCAuthMiddleware) (ifrit.Runner, error) {
	scalingEngine, err := s.createScalingEngineRoutes()
	if err != nil {
		return nil, fmt.Errorf("failed to create scaling engine routes: %w", err)
	}

	scalingEngine.Use(am.XFCCAuthenticationMiddleware)

	if err := s.createHealthRouter(); err != nil {
		return nil, fmt.Errorf("failed to create health router: %w", err)
	}

	s.autoscalerRouter.GetRouter().PathPrefix("/v1").Handler(scalingEngine)
	s.autoscalerRouter.GetRouter().PathPrefix("/health").Handler(s.healthRouter)

	return helpers.NewHTTPServer(s.logger, s.conf.CFServer, s.autoscalerRouter.GetRouter())
}

func (s *Server) CreateMtlsServer() (ifrit.Runner, error) {
	r, err := s.createScalingEngineRoutes()
	if err != nil {
		return nil, fmt.Errorf("failed to create scaling engine routes: %w", err)
	}

	return helpers.NewHTTPServer(s.logger, s.conf.Server, r)
}

func createPrometheusRegistry(policyDB db.PolicyDB, scalingEngineDB db.ScalingEngineDB, schedulerDB db.SchedulerDB, httpStatusCollector healthendpoint.HTTPStatusCollector, logger lager.Logger) *prometheus.Registry {
	promRegistry := prometheus.NewRegistry()
	//validate that db are not nil

	if policyDB == nil || scalingEngineDB == nil || schedulerDB == nil {
		logger.Error("failed-to-create-prometheus-registry", fmt.Errorf("db is nil: have policyDB: %t, have scalingEngineDB: %t, have schedulerDB: %t", policyDB != nil, scalingEngineDB != nil, schedulerDB != nil))
		return promRegistry
	}

	healthendpoint.RegisterCollectors(promRegistry, []prometheus.Collector{
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "scalingengine", "policyDB", policyDB),
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "scalingengine", "scalingengineDB", scalingEngineDB),
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "scalingengine", "schedulerDB", schedulerDB),
		httpStatusCollector,
	}, true, logger.Session("scalingengine-prometheus"))
	return promRegistry
}

func (s *Server) createHealthRouter() error {
	checkers := []healthendpoint.Checker{}
	gatherer := createPrometheusRegistry(s.policyDB, s.scalingEngineDB, s.schedulerDB, s.httpStatusCollector, s.logger)
	healthRouter, err := healthendpoint.NewHealthRouter(s.conf.Health, checkers, s.logger.Session("health-server"), gatherer, time.Now)
	if err != nil {
		return err
	}
	s.healthRouter = healthRouter
	return nil
}

func Liveness(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) createScalingEngineRoutes() (*mux.Router, error) {
	se := NewScalingHandler(s.logger, s.scalingEngineDB, s.scalingEngine)
	syncHandler := NewSyncHandler(s.logger, s.synchronizer)

	scalingHistoryHandler, err := newScalingHistoryHandler(s.logger, s.scalingEngineDB)
	if err != nil {
		return nil, fmt.Errorf("failed to create scaling history handler: %w", err)
	}

	httpStatusCollectMiddleware := healthendpoint.NewHTTPStatusCollectMiddleware(s.httpStatusCollector)

	autoscalerRouter := routes.NewRouter()
	r := autoscalerRouter.CreateScalingEngineRoutes()

	r.Use(otelmux.Middleware("scalingengine"))

	r.Use(httpStatusCollectMiddleware.Collect)
	r.Get(routes.LivenessRouteName).Handler(VarsFunc(Liveness))
	r.Get(routes.ScaleRouteName).Handler(VarsFunc(se.Scale))

	r.Get(routes.GetScalingHistoriesRouteName).Handler(scalingHistoryHandler)

	r.Get(routes.SetActiveScheduleRouteName).Handler(VarsFunc(se.StartActiveSchedule))
	r.Get(routes.DeleteActiveScheduleRouteName).Handler(VarsFunc(se.RemoveActiveSchedule))
	r.Get(routes.GetActiveSchedulesRouteName).Handler(VarsFunc(se.GetActiveSchedule))

	r.Get(routes.SyncActiveSchedulesRouteName).Handler(VarsFunc(syncHandler.Sync))
	return r, nil
}

func newScalingHistoryHandler(logger lager.Logger, scalingEngineDB db.ScalingEngineDB) (http.Handler, error) {
	scalingHistoryHandler, err := NewScalingHistoryHandler(logger, scalingEngineDB)
	if err != nil {
		return nil, fmt.Errorf("error creating scaling history handler: %w", err)
	}
	server, err := scalinghistory.NewServer(scalingHistoryHandler)
	if err != nil {
		return nil, fmt.Errorf("error creating ogen scaling history server: %w", err)
	}
	return server, err
}
