package server

import (
	"fmt"
	"metrics-collector/cf"
	"metrics-collector/config"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pivotal-golang/lager"
)

const PATH_MEMORY_METRIC = "/v1/apps/{appid}/metrics/memory"
const ROUTE_NAME_MEMORY_METRIC = "memory-metric"

type Server struct {
	config   config.ServerConfig
	logger   lager.Logger
	handler  http.Handler
	listener net.Listener
}

type VarsFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string)

func (vh VarsFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vh(w, r, vars)
}

func NewServer(logger lager.Logger, conf config.ServerConfig, cfc cf.CfClient, consumer NoaaConsumer) *Server {
	mmh := NewMemoryMetricHandler(logger, cfc, consumer)

	r := mux.NewRouter()
	r.Methods("GET").Path(PATH_MEMORY_METRIC).Handler(VarsFunc(mmh.GetMemoryMetric)).Name(ROUTE_NAME_MEMORY_METRIC)

	return &Server{
		config:  conf,
		logger:  logger,
		handler: r,
	}
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("0.0.0.0:%d", s.config.Port)
	s.logger.Info("start-server", lager.Data{"addr": addr})

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		s.logger.Error("start-listener", err)
		return err
	}
	s.listener = listener

	http.Serve(listener, s.handler)
	return nil
}

func (s *Server) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}
}
