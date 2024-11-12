package server

import (
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
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
	logger          lager.Logger
	conf            *config.Config
	policyDB        db.PolicyDB
	scalingEngineDB db.ScalingEngineDB
	schedulerDB     db.SchedulerDB
	scalingEngine   scalingengine.ScalingEngine
	synchronizer    schedule.ActiveScheduleSychronizer
}

func NewServer(logger lager.Logger, conf *config.Config, policyDB db.PolicyDB, scalingEngineDB db.ScalingEngineDB, schedulerDB db.SchedulerDB, scalingEngine scalingengine.ScalingEngine, synchronizer schedule.ActiveScheduleSychronizer) *Server {
	return &Server{
		logger:          logger,
		conf:            conf,
		policyDB:        policyDB,
		scalingEngineDB: scalingEngineDB,
		schedulerDB:     schedulerDB,
		scalingEngine:   scalingEngine,
		synchronizer:    synchronizer,
	}
}

func (s *Server) GetHealthServer() (ifrit.Runner, error) {
	httpStatusCollector := healthendpoint.NewHTTPStatusCollector("autoscaler", "scalingengine")
	healthRouter, err := createHealthRouter(s.logger, s.conf, s.policyDB, s.scalingEngineDB, s.schedulerDB, httpStatusCollector)
	if err != nil {
		return nil, fmt.Errorf("failed to create health router: %w", err)
	}

	return helpers.NewHTTPServer(s.logger, s.conf.Health.ServerConfig, healthRouter)
}

func (s *Server) GetMtlsServer() (ifrit.Runner, error) {
	httpStatusCollector := healthendpoint.NewHTTPStatusCollector("autoscaler", "scalingengine")
	scalingEngineRouter, err := createScalingEngineRouter(s.logger, s.scalingEngineDB, s.scalingEngine, s.synchronizer, httpStatusCollector, s.conf.Server)
	if err != nil {
		return nil, fmt.Errorf("failed to create scaling engine router: %w", err)
	}

	//	mainRouter := setupMainRouter(scalingEngineRouter, healthRouter)

	return helpers.NewHTTPServer(s.logger, s.conf.Server, scalingEngineRouter)
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

func createHealthRouter(logger lager.Logger, conf *config.Config, policyDB db.PolicyDB, scalingEngineDB db.ScalingEngineDB, schedulerDB db.SchedulerDB, httpStatusCollector healthendpoint.HTTPStatusCollector) (*mux.Router, error) {
	checkers := []healthendpoint.Checker{}
	gatherer := createPrometheusRegistry(policyDB, scalingEngineDB, schedulerDB, httpStatusCollector, logger)
	healthRouter, err := healthendpoint.NewHealthRouter(conf.Health, checkers, logger.Session("health-server"), gatherer, time.Now)
	if err != nil {
		return nil, fmt.Errorf("failed to create health router: %w", err)
	}
	return healthRouter, nil
}

func createScalingEngineRouter(logger lager.Logger, scalingEngineDB db.ScalingEngineDB, scalingEngine scalingengine.ScalingEngine, synchronizer schedule.ActiveScheduleSychronizer, httpStatusCollector healthendpoint.HTTPStatusCollector, serverConfig helpers.ServerConfig) (*mux.Router, error) {
	httpStatusCollectMiddleware := healthendpoint.NewHTTPStatusCollectMiddleware(httpStatusCollector)

	se := NewScalingHandler(logger, scalingEngineDB, scalingEngine)
	syncHandler := NewSyncHandler(logger, synchronizer)

	r := routes.ScalingEngineRoutes()
	r.Use(otelmux.Middleware("scalingengine"))

	r.Use(httpStatusCollectMiddleware.Collect)
	r.Get(routes.ScaleRouteName).Handler(VarsFunc(se.Scale))

	scalingHistoryHandler, err := newScalingHistoryHandler(logger, scalingEngineDB)
	if err != nil {
		return nil, err
	}
	r.Get(routes.GetScalingHistoriesRouteName).Handler(scalingHistoryHandler)

	r.Get(routes.SetActiveScheduleRouteName).Handler(VarsFunc(se.StartActiveSchedule))
	r.Get(routes.DeleteActiveScheduleRouteName).Handler(VarsFunc(se.RemoveActiveSchedule))
	r.Get(routes.GetActiveSchedulesRouteName).Handler(VarsFunc(se.GetActiveSchedule))

	r.Get(routes.SyncActiveSchedulesRouteName).Handler(VarsFunc(syncHandler.Sync))
	return r, nil
}

//  func setupMainRouter(r *mux.Router, healthRouter *mux.Router) *mux.Router {
//  	mainRouter := mux.NewRouter()
//  	mainRouter.PathPrefix("/v1").Handler(r)
//  	mainRouter.PathPrefix("/health").Handler(healthRouter)
//  	mainRouter.PathPrefix("/").Handler(healthRouter)
//  	return mainRouter
//  }

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
