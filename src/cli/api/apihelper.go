package api

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"autoscaler/models"
	e "cli/errors"
	"cli/ui"
	. "cli/util/http"

	"code.cloudfoundry.org/cli/cf/trace"
)

const (
	HealthPath  = "/health"
	PolicyPath  = "/v1/apps/{appId}/policy"
	MetricPath  = "/v1/apps/{appId}/metric_histories/{metric_type}"
	HistoryPath = "/v1/apps/{appId}/scaling_histories"
)

type APIHelper struct {
	Endpoint *APIEndpoint
	Client   *CFClient
	Logger   trace.Printer
}

func NewAPIHelper(endpoint *APIEndpoint, cfclient *CFClient, traceEnabled string) *APIHelper {

	return &APIHelper{
		Endpoint: endpoint,
		Client:   cfclient,
		Logger:   trace.NewLogger(os.Stdout, false, traceEnabled, ""),
	}
}

func newHTTPClient(skipSSLValidation bool, logger trace.Printer) *http.Client {
	return &http.Client{
		Transport: makeTransport(skipSSLValidation, logger),
		Timeout:   30 * time.Second,
	}
}

func makeTransport(skipSSLValidation bool, logger trace.Printer) http.RoundTripper {
	return NewTraceLoggingTransport(&http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableCompression:  true,
		DisableKeepAlives:   true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipSSLValidation,
		},
	}, logger)
}

func (helper *APIHelper) DoRequest(req *http.Request, host string) (*http.Response, error) {

	client := newHTTPClient(helper.Endpoint.SkipSSLValidation || helper.Client.IsSSLDisabled, helper.Logger)
	resp, err := client.Do(req)
	if err != nil {
		var innerErr error
		switch typedErr := err.(type) {
		case *url.Error:
			innerErr = typedErr.Err
		}

		if innerErr != nil {
			switch typedInnerErr := innerErr.(type) {
			case x509.UnknownAuthorityError, x509.HostnameError, x509.CertificateInvalidError:
				return nil, e.NewAccessError(fmt.Sprintf(ui.InvalidSSLCerts, host))
			default:
				return nil, typedInnerErr
			}
		}
	}

	return resp, nil

}

func (helper *APIHelper) CheckHealth() error {
	baseURL := helper.Endpoint.URL
	requestURL := fmt.Sprintf("%s%s", baseURL, HealthPath)
	req, err := http.NewRequest("GET", requestURL, nil)

	resp, err := helper.DoRequest(req, baseURL)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {

		var errorMsg string
		_ = json.NewDecoder(resp.Body).Decode(&errorMsg)

		if errorMsg == "" {
			errorMsg = fmt.Sprintf(ui.InvalidAPIEndpoint, baseURL)
		}

		return e.NewAccessError(errorMsg)
	}

	return nil

}

func (helper *APIHelper) GetPolicy() (string, error) {

	err := helper.CheckHealth()
	if err != nil {
		return "", err
	}

	baseURL := helper.Endpoint.URL
	requestURL := fmt.Sprintf("%s%s", baseURL, strings.Replace(PolicyPath, "{appId}", helper.Client.AppId, -1))
	req, err := http.NewRequest("GET", requestURL, nil)
	req.Header.Add("Authorization", helper.Client.AuthToken)

	resp, err := helper.DoRequest(req, baseURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var errorMsg string
		switch resp.StatusCode {
		case 401:
			errorMsg = fmt.Sprintf(ui.Unauthorized, baseURL, helper.Client.CCAPIEndpoint)
		case 404:
			errorMsg = fmt.Sprintf(ui.PolicyNotFound, helper.Client.AppName)
		default:
			_ = json.NewDecoder(resp.Body).Decode(&errorMsg)
		}
		return "", e.NewAccessError(errorMsg)
	}

	var policy *models.ScalingPolicy
	err = json.NewDecoder(resp.Body).Decode(&policy)
	if err != nil {
		return "", err
	}

	var policyJSON bytes.Buffer
	enc := json.NewEncoder(&policyJSON)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "\t")
	enc.Encode(policy)

	return policyJSON.String(), nil

}
