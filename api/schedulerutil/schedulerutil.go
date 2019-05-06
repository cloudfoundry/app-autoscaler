package schedulerutil

import (
	"autoscaler/api/config"
	"autoscaler/helpers"
	"autoscaler/routes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"code.cloudfoundry.org/lager"
)

type SchedulerUtil struct {
	httpClient   *http.Client
	logger       lager.Logger
	schedulerUrl string
}

func NewSchedulerUtil(conf *config.Config, logger lager.Logger) *SchedulerUtil {
	client, err := helpers.CreateHTTPClient(&conf.Scheduler.TLSClientCerts)
	if err != nil {
		logger.Error("failed to create http client for Scheduler", err, lager.Data{"scheduler": conf.Scheduler.TLSClientCerts})
		os.Exit(1)
	}

	schedulerUtil := &SchedulerUtil{
		httpClient:   client,
		logger:       logger,
		schedulerUrl: conf.Scheduler.SchedulerURL,
	}
	return schedulerUtil
}

func (su *SchedulerUtil) CreateOrUpdateSchedule(appId string, policyJSONStr string, policyGuid string) error {
	var url string
	path, _ := routes.SchedulerRoutes().Get(routes.UpdateScheduleRouteName).URLPath("appId", appId)
	parameters := path.Query()
	parameters.Add("guid", policyGuid)

	url = su.schedulerUrl + path.RequestURI() + "?" + parameters.Encode()

	req, err := http.NewRequest("PUT", url, strings.NewReader(policyJSONStr))
	if err != nil {
		su.logger.Error("failed to create request to scheduler", err, lager.Data{"appId": appId, "policy": policyJSONStr})
		return err
	}

	resp, err := su.httpClient.Do(req)
	if err != nil {
		su.logger.Error("failed to do request to scheduler", err, lager.Data{"appId": appId, "policy": policyJSONStr})
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		su.logger.Info("successfully created/updated schedules", lager.Data{"appId": appId, "policy": policyJSONStr})
		return nil
	}

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		su.logger.Error("failed to read response body", err, lager.Data{"appId": appId})
		return err
	}

	if resp.StatusCode == http.StatusBadRequest {
		su.logger.Error("Failed to create schedules due to validation errors in schedule", nil, lager.Data{"appId": appId, "responsebody": string(responseData)})
		return fmt.Errorf("Failed to create schedules due to validation errors in schedule : " + string(responseData))
	}

	return fmt.Errorf("Error occurred in scheduler module during creation/update : " + string(responseData))
}

func (su *SchedulerUtil) DeleteSchedule(appId string) error {
	var url string
	path, _ := routes.SchedulerRoutes().Get(routes.DeleteScheduleRouteName).URLPath("appId", appId)

	url = su.schedulerUrl + path.RequestURI()

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		su.logger.Error("failed to create request to scheduler", err, lager.Data{"appId": appId})
		return err
	}

	resp, err := su.httpClient.Do(req)
	if err != nil {
		su.logger.Error("failed to do request to scheduler", err, lager.Data{"appId": appId})
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		su.logger.Info("successfully deleted schedules", lager.Data{"appId": appId})
		return nil
	}

	if resp.StatusCode == http.StatusNotFound {
		su.logger.Info("schedule not found", lager.Data{"appId": appId})
		return nil
	}

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		su.logger.Error("failed to read response body", err, lager.Data{"appId": appId})
		return err
	}

	return fmt.Errorf("Error occurred in scheduler module during deletion : " + string(responseData))
}
