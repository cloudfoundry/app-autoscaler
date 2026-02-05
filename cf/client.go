package cf

import (
	"context"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
	"github.com/hashicorp/go-retryablehttp"
)

const (
	PathCFAuth                     = "/oauth/token"
	PathIntrospectToken            = "/introspect"
	GrantTypeClientCredentials     = "client_credentials"
	TimeToRefreshBeforeTokenExpire = 10 * time.Minute
	defaultPerPage                 = 100
)

type (
	Guid   = models.GUID
	Tokens struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}

	TokensInfo struct {
		Tokens
		grantTime time.Time
	}

	IntrospectionResponse struct {
		Active   bool     `json:"active"`
		Email    string   `json:"email"`
		ClientId string   `json:"client_id"`
		Scopes   []string `json:"scope"`
	}

	AuthClient interface {
		Login() error
		InvalidateToken()
		RefreshAuthToken() (Tokens, error)
		GetTokens() (Tokens, error)
		IsUserAdmin(userToken string) (bool, error)
		IsUserSpaceDeveloper(userToken string, appId Guid) (bool, error)
		IsTokenAuthorized(token, clientId string) (bool, error)
	}

	AuthContextClient interface {
		Login(ctx context.Context) error
		InvalidateToken()
		RefreshAuthToken(ctx context.Context) (Tokens, error)
		GetTokens(ctx context.Context) (Tokens, error)
		IsUserAdmin(ctx context.Context, userToken string) (bool, error)
		IsUserSpaceDeveloper(ctx context.Context, userToken string, appId Guid) (bool, error)
		IsTokenAuthorized(ctx context.Context, token, clientId string) (bool, error)
	}

	CFClient interface {
		AuthClient
		ApiClient
		GetCtxClient() ContextClient
	}

	ContextClient interface {
		AuthContextClient
		ApiContextClient
	}

	ApiClient interface {
		GetEndpoints() (Endpoints, error)
		GetApp(appId Guid) (*App, error)
		GetAppProcesses(appId Guid, processTypes ...string) (Processes, error)
		GetAppAndProcesses(appId Guid) (*AppAndProcesses, error)
		ScaleAppWebProcess(appId Guid, numberOfProcesses int) error
		GetServiceInstance(serviceInstanceGuid string) (*ServiceInstance, error)
		GetServicePlan(servicePlanGuid string) (*ServicePlan, error)
	}

	ApiContextClient interface {
		GetEndpoints(ctx context.Context) (Endpoints, error)
		GetApp(ctx context.Context, appId Guid) (*App, error)
		GetAppProcesses(ctx context.Context, appId Guid, processTypes ...string) (Processes, error)
		GetAppAndProcesses(ctx context.Context, appId Guid) (*AppAndProcesses, error)
		ScaleAppWebProcess(ctx context.Context, appId Guid, numberOfProcesses int) error
		GetServiceInstance(ctx context.Context, serviceInstanceGuid string) (*ServiceInstance, error)
		GetServicePlan(ctx context.Context, servicePlanGuid string) (*ServicePlan, error)
	}
)

func (t TokensInfo) isTokenExpired(now func() time.Time) bool {
	return now().Sub(t.grantTime) > (time.Duration(t.ExpiresIn)*time.Second - TimeToRefreshBeforeTokenExpire)
}

// NewCFClient creates a new CF client using go-cfclient/v3
func NewCFClient(conf *Config, logger lager.Logger, clk clock.Clock) (CFClient, error) {
	return NewCFClientWrapper(conf, logger, clk)
}

// RetryClient creates a retryable HTTP client
func RetryClient(config ClientConfig, client *http.Client, logger lager.Logger) *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 0
	if config.MaxRetries != 0 {
		retryClient.RetryMax = config.MaxRetries
	}
	if config.MaxRetryWaitMs != 0 {
		retryClient.RetryWaitMax = time.Duration(config.MaxRetryWaitMs) * time.Millisecond
	}
	retryClient.Logger = LeveledLoggerAdapter{logger.Session("retryablehttp")}
	retryClient.HTTPClient = client
	retryClient.ErrorHandler = func(resp *http.Response, err error, numTries int) (*http.Response, error) {
		return resp, err
	}
	return retryClient.StandardClient()
}
