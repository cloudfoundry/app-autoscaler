package forwarder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3"
)

type GatewayEmitter struct {
	logger lager.Logger
	client *http.Client
	url    string
}

func NewGatewayEmitter(logger lager.Logger, gatewayURL string, tlsCerts models.TLSCerts) (MetricForwarder, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	tlsConfig, err := tlsCerts.CreateClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config for gateway: %w", err)
	}
	if tlsConfig != nil {
		transport.TLSClientConfig = tlsConfig
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	return &GatewayEmitter{
		logger: logger,
		client: client,
		url:    gatewayURL + "/v1/envelopes",
	}, nil
}

func (e *GatewayEmitter) EmitMetric(metric *models.CustomMetric) {
	body, err := json.Marshal([]*models.CustomMetric{metric})
	if err != nil {
		e.logger.Error("failed-to-marshal-metric", err)
		return
	}

	resp, err := e.client.Post(e.url, "application/json", bytes.NewReader(body))
	if err != nil {
		e.logger.Error("failed-to-send-metric-to-gateway", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		e.logger.Error("gateway-returned-error", fmt.Errorf("status %d", resp.StatusCode), lager.Data{"url": e.url})
	}
}
