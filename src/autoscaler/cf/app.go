package cf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"

	"code.cloudfoundry.org/lager"

	"autoscaler/models"
)

const (
	TokenTypeBearer = "Bearer"
	PathApp         = "/v2/apps"
	CFAppNotFound   = "CF-AppNotFound"
)

func (c *cfClient) GetApp(appID string) (*models.AppEntity, error) {
	url := c.conf.API + path.Join(PathApp, appID, "summary")
	c.logger.Debug("get-app-instances", lager.Data{"url": url})

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.logger.Error("get-app-instances-new-request", err)
		return nil, err
	}
	req.Header.Set("Authorization", TokenTypeBearer+" "+c.GetTokens().AccessToken)

	var resp *http.Response
	resp, err = c.httpClient.Do(req)

	if err != nil {
		c.logger.Error("get-app-instances-do-request", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == 404 {
			respBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				c.logger.Error("failed-to-read-response-body-while-getting-app-summary", err, lager.Data{"appID": appID})
				return nil, err
			}
			var bodydata map[string]interface{}
			err = json.Unmarshal([]byte(respBody), &bodydata)
			if err != nil {
				err = fmt.Errorf("%s", string(respBody))
				c.logger.Error("failed-to-get-application-summary", err, lager.Data{"appID": appID})
				return nil, err
			}
			errorDescription := bodydata["description"].(string)
			errorCode := bodydata["error_code"].(string)
			code := bodydata["code"].(float64)

			if errorCode == CFAppNotFound && code == 100004 {
				// Application does not exists
				err = models.NewAppNotFoundErr(errorDescription)
			} else {
				err = fmt.Errorf("failed getting application summary: [%d] %s: %s", resp.StatusCode, errorCode, errorDescription)
			}
			c.logger.Error("get-app-summary-response", err, lager.Data{"appID": appID, "statusCode": resp.StatusCode, "description": errorDescription, "errorCode": errorCode})
			return nil, err
		}
		// For Non 404 Error type
		err = fmt.Errorf("failed getting application summary: %s [%d] %s", url, resp.StatusCode, resp.Status)
		c.logger.Error("get-app-instances-response", err)
		return nil, err
	}

	appEntity := &models.AppEntity{}
	err = json.NewDecoder(resp.Body).Decode(appEntity)
	if err != nil {
		c.logger.Error("get-app-instances-decode", err)
		return nil, err
	}
	return appEntity, nil
}

func (c *cfClient) SetAppInstances(appID string, num int) error {
	url := c.conf.API + path.Join(PathApp, appID)
	c.logger.Debug("set-app-instances", lager.Data{"url": url})

	appEntity := models.AppEntity{
		Instances: num,
	}
	body, err := json.Marshal(appEntity)
	if err != nil {
		c.logger.Error("set-app-instances-marshal", err, lager.Data{"appID": appID, "appEntity": appEntity})
		return err
	}

	var req *http.Request
	req, err = http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		c.logger.Error("set-app-instances-new-request", err)
		return err
	}
	req.Header.Set("Authorization", TokenTypeBearer+" "+c.GetTokens().AccessToken)
	req.Header.Set("Content-Type", "application/json")

	var resp *http.Response
	resp, err = c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("set-app-instances-do-request", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			c.logger.Error("failed-to-read-response-body-while-setting-app-instance", err, lager.Data{"appID": appID})
			return err
		}
		var bodydata map[string]interface{}
		err = json.Unmarshal([]byte(respBody), &bodydata)
		if err != nil {
			err = fmt.Errorf("%s", string(respBody))
			c.logger.Error("faileded-to-set-application-instances", err, lager.Data{"appID": appID})
			return err
		}
		errorDescription := bodydata["description"].(string)
		errorCode := bodydata["error_code"].(string)
		err = fmt.Errorf("failed setting application instances: [%d] %s: %s", resp.StatusCode, errorCode, errorDescription)
		c.logger.Error("set-app-instances-response", err, lager.Data{"appID": appID, "statusCode": resp.StatusCode, "description": errorDescription, "errorCode": errorCode})
		return err
	}
	return nil
}
