package cf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

const (
	TokenTypeBearer = "Bearer"
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

	AppAndProcesses struct {
		App       *App
		Processes Processes
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
)

func (p Processes) GetInstances() int {
	instances := 0
	for _, process := range p {
		instances += process.Instances
	}
	return instances
}

func (c *Client) GetAppAndProcesses(appID string) (*AppAndProcesses, error) {
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
	return &AppAndProcesses{App: app, Processes: processes}, nil
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
	url := fmt.Sprintf("%s/v3/apps/%s/processes?per_page=%d", c.conf.API, appID, c.conf.PerPage)
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

/* getProcesses
 * processResponse https://v3-apidocs.cloudfoundry.org/version/3.122.0/index.html#list-processes
 */
func (c *Client) getProcesses(appID string, url string) (*Pagination, Processes, error) {
	resp, err := c.get(url)
	if err != nil {
		return nil, nil, fmt.Errorf("failed getting processes for app '%s': %w", appID, err)
	}
	defer func() { _ = resp.Body.Close() }()

	processResp := struct {
		Pagination Pagination `json:"pagination"`
		Resources  Processes  `json:"resources"`
	}{}

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
	return c.sendRequest(req)
}

func (c *Client) post(url string, bodyStuct any) (*http.Response, error) {
	body, err := json.Marshal(bodyStuct)
	if err != nil {
		return nil, fmt.Errorf("failed post: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("post %s failed: %w", url, err)
	}
	req.Header.Add("Content-Type", "application/json")
	return c.sendRequest(req)
}

func (c *Client) sendRequest(req *http.Request) (*http.Response, error) {
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
	if isError(statusCode) {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response[%d]: %w", statusCode, err)
		}
		return nil, fmt.Errorf("%s request failed: %w", req.Method, models.NewCfError(req.RequestURI, req.RequestURI, statusCode, respBody))
	}
	return resp, nil
}

func isError(statusCode int) bool {
	return statusCode >= 300
}

func (c *Client) ScaleAppWebProcess(appID string, num int) error {
	url := fmt.Sprintf("%s/v3/apps/%s/processes/web/actions/scale", c.conf.API, appID)
	type scaleApp struct {
		Instances int `json:"instances"`
	}
	_, err := c.post(url, scaleApp{Instances: num})
	if err != nil {
		return fmt.Errorf("failed scaling app '%s' to %d: %w", appID, num, err)
	}
	return err
}
