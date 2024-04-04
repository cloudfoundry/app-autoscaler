package cf

import (
	"context"
	"fmt"
	"sync"
	"time"
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
		Metadata      `json:"metadata"`
	}

	AppAndProcesses struct {
		App       *App
		Processes Processes
	}

	Relationships struct {
		Space *Space `json:"space"`
	}
	SpaceData struct {
		Guid SpaceId `json:"guid"`
	}
	Space struct {
		Data SpaceData `json:"data"`
	}

	//Process information for an App from cf for full version look at https://v3-apidocs.cloudfoundry.org/version/3.122.0/index.html#processes
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
)

func (p Processes) GetInstances() int {
	instances := 0
	for _, process := range p {
		instances += process.Instances
	}
	return instances
}

/*GetAppAndProcesses
 * A utility function that gets the app and processes for the app in one call in parallel
 */
func (c *Client) GetAppAndProcesses(appID Guid) (*AppAndProcesses, error) {
	return c.CtxClient.GetAppAndProcesses(context.Background(), appID)
}

func (c *CtxClient) GetAppAndProcesses(ctx context.Context, appID Guid) (*AppAndProcesses, error) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	var app *App
	var processes Processes
	var errApp, errProc error
	go func() {
		defer wg.Done()
		app, errApp = c.GetApp(ctx, appID)
	}()
	go func() {
		defer wg.Done()
		processes, errProc = c.GetAppProcesses(ctx, appID, ProcessTypeWeb)
	}()
	wg.Wait()
	if errApp != nil {
		return nil, fmt.Errorf("get state&instances failed: %w", errApp)
	}
	if errProc != nil {
		return nil, fmt.Errorf("get state&instances failed: %w", errProc)
	}
	return &AppAndProcesses{App: app, Processes: processes}, nil
}

/*GetApp
 * Get the information for a specific app
 * from the v3 api https://v3-apidocs.cloudfoundry.org/version/3.122.0/index.html#apps
 * using GET /v3/apps/:guid/processes/:type
 */
func (c *Client) GetApp(appID Guid) (*App, error) {
	return c.CtxClient.GetApp(context.Background(), appID)
}

func (c *CtxClient) GetApp(ctx context.Context, appID Guid) (*App, error) {
	url := fmt.Sprintf("/v3/apps/%s", appID)
	resp, err := ResourceRetriever[*App]{AuthenticatedClient{c}}.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed getting app '%s': %w", appID, err)
	}
	return resp, nil
}

/*ScaleAppWebProcess
 * Scale the given application Web processes to the given amount
 * https://v3-apidocs.cloudfoundry.org/version/3.122.0/index.html#scale-a-process
 */
func (c *Client) ScaleAppWebProcess(appID Guid, num int) error {
	return c.CtxClient.ScaleAppWebProcess(context.Background(), appID, num)
}

func (c *CtxClient) ScaleAppWebProcess(ctx context.Context, appID Guid, num int) error {
	url := fmt.Sprintf("/v3/apps/%s/processes/web/actions/scale", appID)
	type scaleApp struct {
		Instances int `json:"instances"`
	}
	_, err := ResourceRetriever[Process]{AuthenticatedClient{c}}.Post(ctx, url, scaleApp{Instances: num})
	if err != nil {
		return fmt.Errorf("failed scaling app '%s' to %d: %w", appID, num, err)
	}
	return err
}
