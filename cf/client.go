package cf

import (
	"context"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3"
	"github.com/hashicorp/go-retryablehttp"
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
	retryClient.RetryWaitMax = time.Duration(config.MaxRetryWaitMs) * time.Millisecond
	retryClient.Logger = LeveledLoggerAdapter{logger.Session("retryablehttp")}
	retryClient.HTTPClient = client
	retryClient.ErrorHandler = func(resp *http.Response, err error, _ int) (*http.Response, error) {
		return resp, err
	}
	return retryClient.StandardClient()
}
