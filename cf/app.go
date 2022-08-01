package cf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"sync"
	"time"

	"code.cloudfoundry.org/lager"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

const (
	TokenTypeBearer = "Bearer"
	PathApp         = "/v2/apps"
	CFAppNotFound   = "CF-AppNotFound"
)

type (
	//App the app information from cf for full version look at https://v3-apidocs.cloudfoundry.org/version/3.122.0/index.html#apps
	App struct {
		Guid          string        `json:"guid"`
		Name          string        `json:"name"`
		State         string        `json:"state"`
		CreatedAt     time.Time     `json:"created_at"`
		UpdatedAt     time.Time     `json:"updated_at"`
		Relationships Relationships `json:"relationships"`
	}
	Relationships struct {
		Space *Space `json:"space"`
	}
	SpaceData struct {
		Guid string `json:"guid"`
	}
	Space struct {
		Data SpaceData `json:"data"`
	}

	//Processes the processes information for an App from cf for full version look at https://v3-apidocs.cloudfoundry.org/version/3.122.0/index.html#processes
	Process struct {
		Guid       string    `json:"guid"`
		Type       string    `json:"type"`
		Instances  int       `json:"instances"`
		MemoryInMb int       `json:"memory_in_mb"`
		DiskInMb   int       `json:"disk_in_mb"`
		CreatedAt  time.Time `json:"created_at"`
		UpdatedAt  time.Time `json:"updated_at"`
	}

	Processes []Process

	Pagination struct {
		TotalResults int  `json:"total_results"`
		TotalPages   int  `json:"total_pages"`
		First        Href `json:"first"`
		Last         Href `json:"last"`
		Next         Href `json:"next"`
		Previous     Href `json:"previous"`
	}
	Href struct {
		Url string `json:"href"`
	}

	//processResponse https://v3-apidocs.cloudfoundry.org/version/3.122.0/index.html#list-processes
	processesResponse struct {
		Pagination Pagination `json:"pagination"`
		Resources  Processes  `json:"resources"`
	}
)

func (p Processes) GetInstances() int {
	instances := 0
	for _, process := range p {
		instances += process.Instances
	}
	return instances
}

func (c *Client) GetStateAndInstances(appID string) (*models.AppEntity, error) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	var app *App
	var processes Processes
	var errApp, errProc error
	go func() {
		app, errApp = c.GetApp(appID)
		wg.Done()
	}()
	go func() {
		processes, errProc = c.GetAppProcesses(appID)
		wg.Done()
	}()
	wg.Wait()
	if errApp != nil {
		return nil, fmt.Errorf("get state&instances getApp failed: %w", errApp)
	}
	if errProc != nil {
		return nil, fmt.Errorf("get state&instances GetAppProcesses failed: %w", errProc)
	}
	return &models.AppEntity{State: &app.State, Instances: processes.GetInstances()}, nil
}

/*GetApp
 * Get the information for a specific app
 * from the v3 api https://v3-apidocs.cloudfoundry.org/version/3.122.0/index.html#apps
 */
func (c *Client) GetApp(appID string) (*App, error) {
	url := fmt.Sprintf("%s/v3/apps/%s", c.conf.API, appID)

	resp, err := c.get(url)
	if err != nil {
		return nil, fmt.Errorf("failed getting app '%s': %w", appID, err)
	}
	defer func() { _ = resp.Body.Close() }()

	app := &App{}
	err = json.NewDecoder(resp.Body).Decode(app)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshalling app information for '%s': %w", appID, err)
	}
	return app, nil
}

/*GetAppProcesses
 * Get the processes information for a specific app
 * from the v3 api https://v3-apidocs.cloudfoundry.org/version/3.122.0/index.html#apps
 */
func (c *Client) GetAppProcesses(appID string) (Processes, error) {
	pageNumber := 1
	url := fmt.Sprintf("%s/v3/apps/%s/processes?per_page=100", c.conf.API, appID)
	var processes Processes
	for url != "" {
		pagination, pageProcesses, err := c.getProcesses(appID, url)
		if err != nil {
			return nil, fmt.Errorf("failed getting processes page %d: %w", pageNumber, err)
		}
		processes = append(processes, pageProcesses...)
		url = pagination.Next.Url
		pageNumber++
	}

	return processes, nil
}

func (c *Client) getProcesses(appID string, url string) (*Pagination, Processes, error) {
	resp, err := c.get(url)
	if err != nil {
		return nil, nil, fmt.Errorf("failed getting processes for app '%s': %w", appID, err)
	}
	defer func() { _ = resp.Body.Close() }()

	processResp := processesResponse{}
	err = json.NewDecoder(resp.Body).Decode(&processResp)
	if err != nil {
		return nil, nil, fmt.Errorf("failed unmarshalling processes information for '%s': %w", appID, err)
	}
	return &processResp.Pagination, processResp.Resources, nil
}

func (c *Client) get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	tokens, err := c.GetTokens()
	if err != nil {
		return nil, fmt.Errorf("get token failed: %w", err)
	}
	req.Header.Set("Authorization", TokenTypeBearer+" "+tokens.AccessToken)

	resp, err := c.retryClient.Do(req)
	if err != nil {
		return nil, err
	}

	statusCode := resp.StatusCode
	if statusCode != http.StatusOK {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response[%d]: %w", statusCode, err)
		}
		return nil, fmt.Errorf("get request failed: %w", models.NewCfError(url, statusCode, respBody))
	}
	return resp, nil
}

func (c *Client) SetAppInstances(appID string, num int) error {
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
