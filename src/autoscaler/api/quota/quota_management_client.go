package quota

import (
	"autoscaler/api/config"
	"autoscaler/cf"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"code.cloudfoundry.org/lager"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type QuotaManagementClient struct {
	client   *http.Client
	cfClient cf.CFClient
	conf     *config.QuotaManagementConfig
	logger   lager.Logger
}

func NewQuotaManagementClient(config *config.QuotaManagementConfig, logger lager.Logger, cfClient cf.CFClient) *QuotaManagementClient {
	return &QuotaManagementClient{
		cfClient: cfClient,
		conf:     config,
		logger:   logger.Session("quota-management-client"),
	}
}

// Ask the quota manager for instance quota
func (qmc *QuotaManagementClient) GetQuota(orgGUID, serviceName, planName string) (int, error) {
	if qmc.conf == nil {
		qmc.logger.Info("quota-management-not-configured-allowing-all")
		return -1, nil // quota management disabled
	}

	if qmc.client == nil {
		qmc.logger.Info("creating-client")
		// Create http.DefaultTransport with or without SSL validation
		tr := &http.Transport{}
		var ok bool
		if tr, ok = http.DefaultTransport.(*http.Transport); !ok {
			return 0,
				fmt.Errorf("http.DefaultTransport: %T\n", http.DefaultTransport)
		}
		tr.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: qmc.conf.SkipSSLValidation,
		}
		hc := &http.Client{
			Transport: tr,
		}

		if qmc.conf.ClientID != "" && qmc.conf.Secret != "" && qmc.conf.TokenURL != "" {
			ctx := context.WithValue(context.TODO(), oauth2.HTTPClient, hc)
			conf := &clientcredentials.Config{
				ClientID:     qmc.conf.ClientID,
				ClientSecret: qmc.conf.Secret,
				TokenURL:     qmc.conf.TokenURL,
			}
			qmc.logger.Info("creating-oauth-client", lager.Data{"client_id": conf.ClientID, "token_url": conf.TokenURL})
			qmc.client = conf.Client(ctx)
		} else {
			// plain http client for tests
			qmc.client = hc
		}
	}

	quotaUrl := fmt.Sprintf("%s/api/v2.0/orgs/%s/services/%s/plan/%s",
		qmc.conf.API, orgGUID, serviceName, planName)
	req, err := http.NewRequest(http.MethodGet, quotaUrl, nil)
	if err != nil {
		return 0, fmt.Errorf("quota-management-client: creating GET request to %s failed: %w", quotaUrl, err)
	}
	res, err := qmc.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("quota-management-client: request %#v failed with %w", req, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("quota-management-client: GET %s returned %#v", quotaUrl, res.Status)
	}

	response, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, fmt.Errorf("quota-management-client: failed to read response: %w", err)
	}

	quotaResponse := &struct {
		Quota int `json:"quota"`
	}{}
	err = json.Unmarshal(response, quotaResponse)
	if err != nil {
		return 0, fmt.Errorf("quota-management-client: error while parsing '%s': %w", response, err)
	}
	return quotaResponse.Quota, nil
}

func (qmc *QuotaManagementClient) GetServiceInstancesInOrg(orgGUID, servicePlanGuid string) (int, error) {
	return qmc.cfClient.GetServiceInstancesInOrg(orgGUID, servicePlanGuid)
}
