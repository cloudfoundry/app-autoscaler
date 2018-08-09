package server

import (
	"autoscaler/scalingengine/schedule"
	"net/http"

	"code.cloudfoundry.org/lager"
)

type SyncHandler struct {
	logger      lager.Logger
	sychronizer schedule.ActiveScheduleSychronizer
}

func NewSyncHandler(logger lager.Logger, sychronizer schedule.ActiveScheduleSychronizer) *SyncHandler {
	return &SyncHandler{
		logger:      logger,
		sychronizer: sychronizer,
	}
}

func (s *SyncHandler) Sync(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	go s.sychronizer.Sync()
}
