package cf

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
	"golang.org/x/oauth2"
)

const DeploymentCorrelationLabel = "app-autoscaler.cloudfoundry.org/scaling-correlation-id"

// A label value must fit CAPI's label syntax and cannot inject selector clauses.
var deploymentCorrelationPattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9_.-]{0,61}[A-Za-z0-9])?$`)

func (w *CFClientWrapper) HasActiveDeployment(ctx context.Context, appId Guid) (bool, error) {
	opts := client.NewDeploymentListOptions()
	opts.AppGUIDs = client.Filter{Values: []string{string(appId)}}
	opts.StatusValues = client.Filter{Values: []string{"ACTIVE"}}
	deployments, err := w.cfClient.Deployments.ListAll(ctx, opts)
	if err != nil {
		return false, fmt.Errorf("failed checking active deployments for app '%s': %w", appId, MapCFClientError(err))
	}
	return len(deployments) > 0, nil
}

func (w *CFClientWrapper) CreateRollingMemoryDeployment(ctx context.Context, appId Guid, memoryMB int, correlationID string) (*Deployment, error) {
	if appId == "" || memoryMB <= 0 || !deploymentCorrelationPattern.MatchString(correlationID) {
		return nil, fmt.Errorf("deployment requires an app GUID, positive memory and a valid correlation label value")
	}

	maxInFlight := 1
	deployment := resource.NewDeploymentCreate(string(appId))
	deployment.Strategy = "rolling"
	deployment.Options = &resource.DeploymentOptions{
		MaxInFlight: &maxInFlight,
		MemoryInMB:  &memoryMB,
	}
	deployment.Metadata = &resource.Metadata{Labels: map[string]*string{DeploymentCorrelationLabel: &correlationID}}
	body, err := json.Marshal(deployment)
	if err != nil {
		return nil, fmt.Errorf("failed encoding deployment for app '%s': %w", appId, err)
	}
	ctx = context.WithValue(ctx, deploymentPostContextKey{}, true)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.cfClient.ApiURL("/v3/deployments"), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed creating deployment request for app '%s': %w", appId, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", GetUserAgent())

	// Use the existing CF client's authentication configuration and HTTP transport,
	// but not its auth transport, which automatically replays requests after 401.
	// A dedicated token source also keeps token acquisition failures distinct from
	// responses to the deployment POST. No new dependencies or credentials needed.
	source, err := w.cfClient.CreateOAuth2TokenSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed authenticating deployment for app '%s': %w", appId, err)
	}
	httpClient := *w.cfClient.HTTPClient()
	httpClient.Transport = &oauth2.Transport{Base: httpClient.Transport, Source: source}
	httpClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	// Prevent even net/http from replaying the body on a reused-connection error.
	req.GetBody = nil
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed requesting deployment for app '%s': %w", appId, err)
	}
	defer resp.Body.Close()

	// go-cfclient's error decoder drops the HTTP status for CAPI JSON errors.
	// Preserve the actual response status even if the error body is malformed.
	if resp.StatusCode != http.StatusCreated {
		responseBody, readErr := io.ReadAll(resp.Body)
		cfErr := &CfError{StatusCode: resp.StatusCode, ResourceId: string(appId), Url: req.URL.String()}
		if readErr == nil {
			_ = json.Unmarshal(responseBody, cfErr)
		}
		if !cfErr.IsValid() {
			detail := truncateString(string(responseBody), 512)
			if readErr != nil {
				detail = fmt.Sprintf("failed reading response: %s", readErr)
			}
			cfErr.Errors = ErrorItems{{Title: http.StatusText(resp.StatusCode), Detail: detail}}
		}
		return nil, fmt.Errorf("failed requesting deployment for app '%s': %w", appId, cfErr)
	}
	var result resource.Deployment
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed decoding deployment for app '%s': %w", appId, err)
	}
	if result.GUID == "" {
		return nil, fmt.Errorf("deployment response for app '%s' has no GUID", appId)
	}
	return mapResourceDeployment(&result), nil
}

func (w *CFClientWrapper) GetDeployment(ctx context.Context, guid string) (*Deployment, error) {
	deployment, err := w.cfClient.Deployments.Get(ctx, guid)
	if err != nil {
		return nil, fmt.Errorf("failed getting deployment '%s': %w", guid, MapCFClientError(err))
	}
	return mapResourceDeployment(deployment), nil
}

// FindDeployment includes finalized deployments so a lost POST response can be
// reconciled even after the rollout has completed. No match is (nil, nil).
func (w *CFClientWrapper) FindDeployment(ctx context.Context, appId Guid, correlationID string) (*Deployment, error) {
	if appId == "" || !deploymentCorrelationPattern.MatchString(correlationID) {
		return nil, fmt.Errorf("deployment lookup requires an app GUID and a valid correlation label value")
	}
	opts := client.NewDeploymentListOptions()
	opts.AppGUIDs = client.Filter{Values: []string{string(appId)}}
	opts.LabelSel = client.LabelSelector{}
	opts.LabelSel.EqualTo(DeploymentCorrelationLabel, correlationID)
	deployments, err := w.cfClient.Deployments.ListAll(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed finding deployment for app '%s', correlation '%s': %w", appId, correlationID, MapCFClientError(err))
	}
	if len(deployments) > 1 {
		return nil, fmt.Errorf("multiple deployments for app '%s', correlation '%s'", appId, correlationID)
	}
	if len(deployments) == 0 {
		return nil, nil
	}
	return mapResourceDeployment(deployments[0]), nil
}

func mapResourceDeployment(deployment *resource.Deployment) *Deployment {
	return &Deployment{
		GUID:         deployment.GUID,
		StatusValue:  deployment.Status.Value,
		StatusReason: deployment.Status.Reason,
	}
}
