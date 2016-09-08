package cf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path"

	"code.cloudfoundry.org/lager"

	"models"
)

const (
	TokenTypeBearer = "bearer"
	PathApp         = "/v2/apps"
)

func (c *cfClient) GetAppInstances(appId string) (int, error) {
	url := c.conf.Api + path.Join(PathApp, appId)
	c.logger.Debug("get-app-instances", lager.Data{"url": url})

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.logger.Error("get-app-instances-new-request", err)
		return -1, err
	}
	req.Header.Set("Authorization", TokenTypeBearer+" "+c.GetTokensWithRefresh().AccessToken)

	var resp *http.Response
	resp, err = c.httpClient.Do(req)

	if err != nil {
		c.logger.Error("get-app-instances-do-request", err)
		return -1, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("failed getting application summary: %s [%d] %s", url, resp.StatusCode, resp.Status)
		c.logger.Error("get-app-instances-response", err)
		return -1, err
	}

	appInfo := &models.AppInfo{}
	err = json.NewDecoder(resp.Body).Decode(appInfo)
	if err != nil {
		c.logger.Error("get-app-instances-decode", err)
		return -1, err
	}
	return appInfo.Entity.Instances, nil
}

func (c *cfClient) SetAppInstances(appId string, num int) error {
	url := c.conf.Api + path.Join(PathApp, appId)
	c.logger.Debug("set-app-instances", lager.Data{"url": url})

	entity := models.AppEntity{
		Instances: num,
	}
	body, err := json.Marshal(entity)
	if err != nil {
		c.logger.Error("set-app-instances-marshal", err, lager.Data{"appid": appId, "entity": entity})
		return err
	}

	var req *http.Request
	req, err = http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		c.logger.Error("set-app-instances-new-request", err)
		return err
	}
	req.Header.Set("Authorization", TokenTypeBearer+" "+c.GetTokensWithRefresh().AccessToken)
	req.Header.Set("Content-Type", "application/json")

	var resp *http.Response
	resp, err = c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("set-app-instances-do-request", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		err = fmt.Errorf("failed setting application instances: %s [%d] %s", url, resp.StatusCode, resp.Status)
		c.logger.Error("set-app-instances-response", err)
		return err
	}

	return nil
}
