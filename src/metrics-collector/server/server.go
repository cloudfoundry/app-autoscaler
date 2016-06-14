package server

import (
	"fmt"
	"metrics-collector/cf"
	"metrics-collector/config"
	. "metrics-collector/util"
	"net"
	"net/http"
	"os"
)

type Server struct {
	conf     config.ServerConfig
	listener net.Listener
	cfClient cf.CfClient
}

func NewServer(c config.ServerConfig, cfc cf.CfClient) *Server {
	return &Server{
		conf:     c,
		cfClient: cfc,
	}
}

func (s *Server) Start() {
	addr := fmt.Sprintf("0.0.0.0:%d", s.conf.Port)
	Logger.Info("Starting server at " + addr)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		Logger.Error("Failed-to-start-listener", err)
		os.Exit(1)
	}
	s.listener = listener

	handler := NewHandler(s.cfClient)
	http.Handle("/", handler)
	http.Serve(listener, nil)
}

func (s *Server) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}
}
