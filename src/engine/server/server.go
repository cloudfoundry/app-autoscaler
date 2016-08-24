package server

import (
	"cf"
	"db"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
)

const (
	PathScalingActions = "/v1/scaling_actions"
)

func NewServer(logger lager.Logger, cfc cf.CfClient, database db.PolicyDB) ifrit.Runner {
	handler := NewScalingHandler(logger, cfc, database)

	r := mux.NewRouter()
	r.Methods("POST").Path(PathScalingActions).HandlerFunc(handler.HandleScale)

	// todo: make port configurable
	return http_server.New("0.0.0.0:8080", r)
}
