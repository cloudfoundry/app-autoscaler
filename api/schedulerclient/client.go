package schedulerclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/lager/v3"
)

type Client struct {
	httpClient   *http.Client
	logger       lager.Logger
	schedulerUrl string
}

func New(conf *config.Config, logger lager.Logger) *Client {
	logger = logger.Session("schedulerclient")

	client, err := helpers.CreateHTTPSClient(&conf.Scheduler.TLSClientCerts, helpers.DefaultClientConfig(), logger)
	if err != nil {
		logger.Error("Failed to create http client for Scheduler", err, lager.Data{"scheduler": conf.Scheduler.TLSClientCerts})
		os.Exit(1)
	}

	return &Client{
		httpClient:   client,
		logger:       logger,
		schedulerUrl: conf.Scheduler.SchedulerURL,
	}
}

func (s *Client) CreateOrUpdateSchedule(ctx context.Context, appId string, policy *models.ScalingPolicy, policyGuid string) error {
	if policy.Schedules.IsEmpty() {
		return nil
	}
	logger := s.logger.Session("CreateOrUpdateSchedule", lager.Data{"appId": appId})

	req, err := s.putScheduleReq(ctx, logger, appId, policy, policyGuid)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		logger.Error("Failed to do request to scheduler", err, lager.Data{"policy": policy})
		return err
	}

	defer func() { _ = resp.Body.Close() }()

	status := resp.StatusCode
	if unsuccessful(status) {
		responseData, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Error("Failed to read response body", err)
			return err
		}
		return fmt.Errorf("unable to creation/update schedule: %s", string(responseData))
	}
	logger.Info("Successfully created/updated schedules", lager.Data{"policy": policy})
	return nil
}

func (s *Client) putScheduleReq(ctx context.Context, logger lager.Logger, appId string, policy *models.ScalingPolicy, policyGuid string) (*http.Request, error) {
	var url string
	path, _ := routes.SchedulerRoutes().Get(routes.UpdateScheduleRouteName).URLPath("appId", appId)
	parameters := path.Query()
	parameters.Add("guid", policyGuid)

	url = s.schedulerUrl + path.RequestURI() + "?" + parameters.Encode()
	policyBytes, err := json.Marshal(policy)
	if err != nil {
		s.logger.Error("Failed to marshal policy", err, lager.Data{"policy": policy})
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(policyBytes))
	if err != nil {
		logger.Error("Failed to create request to scheduler", err, lager.Data{"policy": policy})
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return req, nil
}

func unsuccessful(status int) bool {
	return status != http.StatusOK && status != http.StatusNoContent
}

func (s *Client) DeleteSchedule(ctx context.Context, appId string) error {
	var url string
	path, err := routes.SchedulerRoutes().Get(routes.DeleteScheduleRouteName).URLPath("appId", appId)
	if err != nil {
		return fmt.Errorf("deleteScheduleRouteName could not make url: %w", err)
	}

	url = s.schedulerUrl + path.RequestURI()

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		s.logger.Error("Failed to create request to scheduler", err, lager.Data{"appId": appId})
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("Failed to do request to scheduler", err, lager.Data{"appId": appId})
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		s.logger.Info("Successfully deleted schedules", lager.Data{"appId": appId})
		return nil
	}

	if resp.StatusCode == http.StatusNotFound {
		s.logger.Info("Schedule not found", lager.Data{"appId": appId})
		return nil
	}

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("Failed to read response body", err, lager.Data{"appId": appId})
		return fmt.Errorf("failed reading response: %w", err)
	}

	return fmt.Errorf("error occurred deleting schedule for app %s response:'%s'", appId, string(responseData))
}
