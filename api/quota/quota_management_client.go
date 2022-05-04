package quota

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"

	"code.cloudfoundry.org/lager"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type Client struct {
	client   *http.Client
	cfClient cf.CFClient
	conf     *config.Config
	logger   lager.Logger
}

func NewClient(config *config.Config, logger lager.Logger) *Client {
	qmc := &Client{conf: config, logger: logger.Session("quota-management-client")}

	if config.QuotaManagement != nil {
		qmc.logger.Info("creating-client")
		hc := &http.Client{
			Timeout:   15 * time.Second,
			Transport: newTransport(),
		}
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, hc)
		conf := &clientcredentials.Config{ClientID: config.QuotaManagement.ClientID, ClientSecret: config.QuotaManagement.Secret, TokenURL: config.QuotaManagement.TokenURL}
		qmc.logger.Info("creating-oauth-client", lager.Data{"client_id": conf.ClientID, "token_url": conf.TokenURL})
		qmc.client = conf.Client(ctx)
	}
	return qmc
}

func newTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// GetQuota Ask the quota manager for instance quota
func (qmc *Client) GetQuota(orgGUID, serviceName, planName string) (int, error) {
	if qmc.conf.QuotaManagement == nil {
		qmc.logger.Info("quota-management-not-configured-allowing-all")
		return -1, nil // quota management disabled
	}

	quotaUrl := fmt.Sprintf("%s/api/v2.0/orgs/%s/services/%s/plan/%s", qmc.conf.QuotaManagement.API, orgGUID, serviceName, planName)

	req, err := http.NewRequest(http.MethodGet, quotaUrl, nil)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return 0, fmt.Errorf("quota-management-client: creating GET request to %s failed: %w", quotaUrl, err)
	}
	res, err := qmc.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("quota-management-client: request %#v failed with %w", req, err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("quota-management-client: GET %s returned %#v", quotaUrl, res.Status)
	}
	var quotaResponse struct {
		Quota int `json:"quota"`
	}
	err = json.NewDecoder(res.Body).Decode(&quotaResponse)
	if err != nil {
		return 0, fmt.Errorf("quota-management-client: failed to read response: %w", err)
	}
	return quotaResponse.Quota, nil
}

func (qmc *Client) SetClient(client *http.Client) {
	qmc.client = client
}

func (qmc *Client) GetServiceInstancesInOrg(orgGUID, servicePlanGuid string) (int, error) {
	return qmc.cfClient.GetServiceInstancesInOrg(orgGUID, servicePlanGuid)
}
