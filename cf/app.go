package cf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"time"

	"code.cloudfoundry.org/lager"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

const (
	TokenTypeBearer = "Bearer"
	PathApp         = "/v2/apps"
	CFAppNotFound   = "CF-AppNotFound"
)

type usage struct {
	Guid      string    `json:"guid"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	State     struct {
		Current  string `json:"current"`
		Previous string `json:"previous"`
	} `json:"state"`
	App struct {
		Guid string `json:"guid"`
		Name string `json:"name"`
	} `json:"app"`
	Process struct {
		Guid string `json:"guid"`
		Type string `json:"type"`
	} `json:"process"`
	Space struct {
		Guid string `json:"guid"`
		Name string `json:"name"`
	} `json:"space"`
	Organization struct {
		Guid string `json:"guid"`
	} `json:"organization"`
	Buildpack struct {
		Guid string `json:"guid"`
		Name string `json:"name"`
	} `json:"buildpack"`
	Task struct {
		Guid string `json:"guid"`
		Name string `json:"name"`
	} `json:"task"`
	MemoryInMbPerInstance struct {
		Current  int `json:"current"`
		Previous int `json:"previous"`
	} `json:"memory_in_mb_per_instance"`
	InstanceCount struct {
		Current  int `json:"current"`
		Previous int `json:"previous"`
	} `json:"instance_count"`
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
}

/*GetApp
 * Get the usage information for a specific app
 * from the v3 api https://v3-apidocs.cloudfoundry.org/version/3.122.0/index.html#app-usage-events
 */
func (c *cfClient) GetApp(appID string) (*models.AppEntity, error) {
	url := fmt.Sprintf("%s%s/%s", c.conf.API, "/v3/app_usage_events", appID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("app usage request failed for app %s :%w", appID, err)
	}
	tokens, err := c.GetTokens()
	if err != nil {
		return nil, fmt.Errorf("failed to get token %s: %w", appID, err)
	}
	req.Header.Set("Authorization", TokenTypeBearer+" "+tokens.AccessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get app %s: %w", appID, err)
	}
	defer func() { _ = resp.Body.Close() }()

	statusCode := resp.StatusCode
	if statusCode != http.StatusOK {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response[%d] for %s : %w", statusCode, appID, err)
		}
		return nil, fmt.Errorf("failed getting application usage events: %w", models.NewCfError(appID, statusCode, respBody))
	}

	usage := &usage{}
	err = json.NewDecoder(resp.Body).Decode(usage)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal app_usage_events response:%w", err)
	}
	return &models.AppEntity{Instances: usage.InstanceCount.Current, State: &usage.State.Current}, nil
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
	tokens, err := c.GetTokens()
	if err != nil {
		c.logger.Error("set-app-instances-get-tokens", err)
		return err
	}
	req.Header.Set("Authorization", TokenTypeBearer+" "+tokens.AccessToken)
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
		err = json.Unmarshal(respBody, &bodydata)
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
