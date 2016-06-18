package server

import (
	"fmt"
	"github.com/pivotal-golang/lager"
	"metrics-collector/cf"
	"metrics-collector/config"
	"net"
	"net/http"
	"os"
)

type Server struct {
	config   config.ServerConfig
	cfClient cf.CfClient
	logger   lager.Logger
	listener net.Listener
}

func NewServer(conf config.ServerConfig, cfc cf.CfClient, logger lager.Logger) *Server {
	return &Server{
		config:   conf,
		cfClient: cfc,
		logger:   logger,
	}
}

func (s *Server) Start() {
	addr := fmt.Sprintf("0.0.0.0:%d", s.config.Port)
	s.logger.Info("start-server", lager.Data{"addr": addr})

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		s.logger.Error("start-listener", err)
		os.Exit(1)
	}
	s.listener = listener

	handler := NewHandler(s.cfClient, s.logger)
	http.Handle("/", handler)
	http.Serve(listener, nil)
}

func (s *Server) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}
}
