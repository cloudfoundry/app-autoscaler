package api

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"autoscaler/models"
	"cli/ui"
	. "cli/util/http"
	cjson "cli/util/json"

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
				return nil, fmt.Errorf(ui.InvalidSSLCerts, host)
			default:
				return nil, typedInnerErr
			}
		}
	}

	return resp, nil

}

func parseErrResponse(raw []byte) (string, error) {

	var f interface{}
	err := json.Unmarshal(raw, &f)
	if err != nil {
		return "", err
	}

	m := f.(map[string]interface{})

	retMsg := ""
	for k, v := range m {
		if k == "error" {
			switch vv := v.(type) {
			case map[string]interface{}:
				for ik, iv := range vv {
					if ik == "message" {
						retMsg = fmt.Sprintf("%v", iv)
					}
				}
			case []interface{}:
				for _, entry := range vv {
					mentry := entry.(map[string]interface{})
					for ik, iv := range mentry {
						if ik == "stack" {
							retMsg = fmt.Sprintf("%v", iv)
							break
						}
					}
				}
			default:
			}
			if retMsg == "" {
				retMsg = fmt.Sprintf("%v", v)
			}

		}
	}

	return retMsg, nil
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

		return errors.New(errorMsg)
	}

	return nil

}

func (helper *APIHelper) GetPolicy() ([]byte, error) {

	err := helper.CheckHealth()
	if err != nil {
		return nil, err
	}

	baseURL := helper.Endpoint.URL
	requestURL := fmt.Sprintf("%s%s", baseURL, strings.Replace(PolicyPath, "{appId}", helper.Client.AppId, -1))
	req, err := http.NewRequest("GET", requestURL, nil)
	req.Header.Add("Authorization", helper.Client.AuthToken)

	resp, err := helper.DoRequest(req, baseURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var errorMsg string
		switch resp.StatusCode {
		case 401:
			errorMsg = fmt.Sprintf(ui.Unauthorized, baseURL, helper.Client.CCAPIEndpoint)
		case 404:
			errorMsg = fmt.Sprintf(ui.PolicyNotFound, helper.Client.AppName)
		default:
			errorMsg, err = parseErrResponse(raw)
			if err != nil {
				return nil, err
			}
		}
		return nil, errors.New(errorMsg)
	}

	var policy *models.ScalingPolicy
	err = json.Unmarshal(raw, &policy)
	if err != nil {
		return nil, err
	}

	prettyPolicy, err := cjson.MarshalWithoutHTMLEscape(policy)
	if err != nil {
		return nil, err
	}

	return prettyPolicy, nil

}

func (helper *APIHelper) CreatePolicy(data interface{}) error {

	err := helper.CheckHealth()
	if err != nil {
		return err
	}

	baseURL := helper.Endpoint.URL
	requestURL := fmt.Sprintf("%s%s", baseURL, strings.Replace(PolicyPath, "{appId}", helper.Client.AppId, -1))

	var body io.Reader
	if data != nil {
		jsonByte, e := json.Marshal(data)
		if e != nil {
			return fmt.Errorf(ui.InvalidPolicy, e)
		}
		body = bytes.NewBuffer(jsonByte)
	}

	req, err := http.NewRequest("PUT", requestURL, body)
	req.Header.Add("Authorization", helper.Client.AuthToken)
	req.Header.Add("Content-Type", "application/json")

	resp, err := helper.DoRequest(req, baseURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {

		var errorMsg string
		switch resp.StatusCode {
		case 401:
			errorMsg = fmt.Sprintf(ui.Unauthorized, baseURL, helper.Client.CCAPIEndpoint)
		case 400:
			errorMsg, err = parseErrResponse(raw)
			if err != nil {
				return err
			}
			errorMsg = fmt.Sprintf(ui.InvalidPolicy, errorMsg)

		default:
			errorMsg, err = parseErrResponse(raw)
			if err != nil {
				return err
			}
		}
		return errors.New(errorMsg)
	}

	return nil
}

func (helper *APIHelper) DeletePolicy() error {

	err := helper.CheckHealth()
	if err != nil {
		return err
	}

	baseURL := helper.Endpoint.URL
	requestURL := fmt.Sprintf("%s%s", baseURL, strings.Replace(PolicyPath, "{appId}", helper.Client.AppId, -1))

	req, err := http.NewRequest("DELETE", requestURL, nil)
	req.Header.Add("Authorization", helper.Client.AuthToken)

	resp, err := helper.DoRequest(req, baseURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var errorMsg string
		switch resp.StatusCode {
		case 401:
			errorMsg = fmt.Sprintf(ui.Unauthorized, baseURL, helper.Client.CCAPIEndpoint)
		case 404:
			errorMsg = fmt.Sprintf(ui.PolicyNotFound, helper.Client.AppName)
		default:
			errorMsg, err = parseErrResponse(raw)
			if err != nil {
				return err
			}
		}
		return errors.New(errorMsg)
	}

	return nil

}
