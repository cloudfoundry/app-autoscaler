package cf

import (
	"context"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3"
	"github.com/hashicorp/go-retryablehttp"
)

const (
	GrantTypeClientCredentials = "client_credentials"
	GrantTypePassword          = "password"
)

type (
	Guid = models.GUID

	IntrospectionResponse struct {
		Active   bool     `json:"active"`
		Email    string   `json:"email"`
		ClientId string   `json:"client_id"`
		Scopes   []string `json:"scope"`
	}

	// CFClient is the main CF client interface.
	// All methods accept a context.Context parameter for cancellation and timeout control.
	CFClient interface {
		Login(ctx context.Context) error
		IsUserAdmin(ctx context.Context, userToken string) (bool, error)
		IsUserSpaceDeveloper(ctx context.Context, userToken string, appId Guid) (bool, error)
		IsTokenAuthorized(ctx context.Context, token, clientId string) (bool, error)
		GetEndpoints(ctx context.Context) (Endpoints, error)
		GetApp(ctx context.Context, appId Guid) (*App, error)
		GetAppProcesses(ctx context.Context, appId Guid, processTypes ...string) (Processes, error)
		GetAppAndProcesses(ctx context.Context, appId Guid) (*AppAndProcesses, error)
		ScaleAppWebProcess(ctx context.Context, appId Guid, numberOfProcesses int) error
		HasActiveDeployment(ctx context.Context, appId Guid) (bool, error)
		CreateRollingMemoryDeployment(ctx context.Context, appId Guid, memoryMB int, correlationID string) (*Deployment, error)
		GetDeployment(ctx context.Context, guid string) (*Deployment, error)
		FindDeployment(ctx context.Context, appId Guid, correlationID string) (*Deployment, error)
		GetServiceInstance(ctx context.Context, serviceInstanceGuid string) (*ServiceInstance, error)
		GetServicePlan(ctx context.Context, servicePlanGuid string) (*ServicePlan, error)
	}
)

func NewCFClient(conf *Config, logger lager.Logger, opts ...WrapperOption) (CFClient, error) {
	return NewCFClientWrapper(conf, logger, opts...)
}

func RetryClient(config ClientConfig, client *http.Client, logger lager.Logger) *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = config.MaxRetries
	if config.MaxRetryWaitMs != 0 {
		retryClient.RetryWaitMax = time.Duration(config.MaxRetryWaitMs) * time.Millisecond
	}
	retryClient.Logger = LeveledLoggerAdapter{logger.Session("retryablehttp")}
	retryClient.HTTPClient = client
	retryClient.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if noRetry, _ := ctx.Value(deploymentPostContextKey{}).(bool); noRetry {
			return false, nil
		}
		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}
	retryClient.ErrorHandler = func(resp *http.Response, err error, _ int) (*http.Response, error) {
		return resp, err
	}
	standardClient := retryClient.StandardClient()
	standardClient.Transport = deploymentPostRetryGuard{transport: standardClient.Transport}
	return standardClient
}

// The retry policy receives no request on transport errors. Mark deployment
// creation before entering retryablehttp so even a lost response cannot resend it.
type deploymentPostContextKey struct{}

type deploymentPostRetryGuard struct {
	transport http.RoundTripper
}

func (t deploymentPostRetryGuard) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodPost && req.URL.Path == "/v3/deployments" {
		req = req.WithContext(context.WithValue(req.Context(), deploymentPostContextKey{}, true))
	}
	return t.transport.RoundTrip(req)
}
