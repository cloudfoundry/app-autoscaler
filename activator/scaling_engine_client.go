package activator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/lager/v3"
)

// ScalingEngineClient wakes an app by asking the scaling engine to scale it up.
type ScalingEngineClient interface {
	ScaleUp(ctx context.Context, appID string) error
}

type scalingEngineClient struct {
	logger     lager.Logger
	httpClient *http.Client
	engineURL  string
}

func NewScalingEngineClient(logger lager.Logger, httpClient *http.Client, engineURL string) ScalingEngineClient {
	return &scalingEngineClient{
		logger:     logger.Session("scaling-engine-client"),
		httpClient: httpClient,
		engineURL:  engineURL,
	}
}

// ScaleUp posts a +1 trigger to the scaling engine. From a parked (zero)
// app this yields one instance; the engine clamps to the policy's bounds.
// The scaling engine has no absolute "scale to N", so "+1" is the wake step.
func (c *scalingEngineClient) ScaleUp(ctx context.Context, appID string) error {
	trigger := &models.Trigger{
		AppId:          appID,
		MetricType:     "scale-from-zero",
		Adjustment:     "+1",
		BypassCooldown: true,
	}
	body, err := json.Marshal(trigger)
	if err != nil {
		return fmt.Errorf("failed to marshal wake trigger for %s: %w", appID, err)
	}

	path, err := routes.NewRouter().CreateScalingEngineRoutes().Get(routes.ScaleRouteName).URLPath("appid", appID)
	if err != nil {
		return fmt.Errorf("failed to build scale url for %s: %w", appID, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.engineURL+path.Path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build scale request for %s: %w", appID, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed calling scaling engine for %s: %w", appID, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("scaling engine returned status %d for %s", resp.StatusCode, appID)
	}
	return nil
}
