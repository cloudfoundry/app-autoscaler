package quota

import (
	"autoscaler/api/config"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	"code.cloudfoundry.org/lager"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type QuotaManagementClient struct {
	client *http.Client
	conf   *config.QuotaManagementConfig
	logger lager.Logger
}

func NewQuotaManagementClient(config *config.QuotaManagementConfig, logger lager.Logger) *QuotaManagementClient {
	return &QuotaManagementClient{
		conf:   config,
		logger: logger.Session("quota-management-client"),
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
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := qmc.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("GET %s returned %#v", quotaUrl, res.Status)
	}
	var quotaResponse struct {
		Quota int `json:"quota"`
	}
	err = json.NewDecoder(res.Body).Decode(&quotaResponse)
	if err != nil {
		return 0, err
	}
	return quotaResponse.Quota, nil
}
