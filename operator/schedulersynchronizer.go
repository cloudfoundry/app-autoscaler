package operator

import (
	"fmt"
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
)

type ScheduleSynchronizer struct {
	client *http.Client
	url    string
	clock  clock.Clock
	logger lager.Logger
}

func NewScheduleSynchronizer(client *http.Client, url string, clock clock.Clock, logger lager.Logger) *ScheduleSynchronizer {
	return &ScheduleSynchronizer{
		client: client,
		url:    url,
		clock:  clock,
		logger: logger,
	}
}

func (s ScheduleSynchronizer) Operate() {
	syncURL := s.url + routes.SyncActiveSchedulesPath
	s.logger.Debug(fmt.Sprintf("Sync schedules of %s", syncURL))

	req, err := http.NewRequest("PUT", syncURL, nil)
	if err != nil {
		s.logger.Error("failed-to-create-sync-scheduler-request", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Error("failed-to-send-sync-scheduler-request", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
}
