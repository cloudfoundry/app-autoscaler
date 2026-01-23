package app

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	api "code.cloudfoundry.org/app-autoscaler-release/src/acceptance/assets/app/go_app/internal/custommetrics"
	"github.com/cloudfoundry-community/go-cfenv"
	json "github.com/json-iterator/go"
	"github.com/mitchellh/mapstructure"
)

//counterfeiter:generate . CustomMetricClient
type CustomMetricClient interface {
	PostCustomMetric(ctx context.Context, logger *slog.Logger, appConfig *cfenv.App, metricsValue float64, metricName string, useMtls bool) error
}

type CustomMetricAPIClient struct{}

var _ CustomMetricClient = &CustomMetricAPIClient{}

var CfenvCurrent = cfenv.Current

func CustomMetricsTests(logger *slog.Logger, mux *http.ServeMux, customMetricTest CustomMetricClient) {
	mux.HandleFunc("GET /custom-metrics/mtls/{name}/{value}", handleCustomMetricsEndpoint(logger, customMetricTest, true))
	mux.HandleFunc("GET /custom-metrics/{name}/{value}", handleCustomMetricsEndpoint(logger, customMetricTest, false))
}

func handleCustomMetricsEndpoint(logger *slog.Logger, customMetricTest CustomMetricClient, useMtls bool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			metricName  string
			metricValue uint64
		)
		var err error

		if metricName = r.PathValue("name"); metricName == "" {
			err = fmt.Errorf("empty metric name")
			logger.Error("Errorf in custom metrics", "error", err)
			Errorf(logger, w, http.StatusBadRequest, "empty metric name")
			return
		}

		if metricValue, err = strconv.ParseUint(r.PathValue("value"), 10, 64); err != nil {
			logger.Error("Invalid metric value", "error", err)
			Errorf(logger, w, http.StatusBadRequest, "invalid metric value: %s", err.Error())
			return
		}

		// required if producer app is sending metric for appToScaleGuid
		appToScaleGuid := r.URL.Query().Get("appToScaleGuid")
		appEnv := &cfenv.App{AppID: appToScaleGuid}

		err = customMetricTest.PostCustomMetric(r.Context(), logger, appEnv, float64(metricValue), metricName, useMtls)
		if err != nil {
			logger.Error("Errorf posting custom metric", "error", err)
			Errorf(logger, w, http.StatusInternalServerError, "error posting custom metric")
			return
		}

		if err := writeJSON(w, http.StatusOK, JSONResponse{
			"mtls": useMtls,
		}); err != nil {
			logger.Error("Failed to write JSON response", slog.Any("error", err))
		}
	}
}

func (*CustomMetricAPIClient) PostCustomMetric(ctx context.Context, logger *slog.Logger, appConfig *cfenv.App, metricValue float64, metricName string, useMtls bool) error {
	currentApp, err := CfenvCurrent()
	if err != nil {
		return fmt.Errorf("cloud foundry environment not found %w", err)
	}
	if appConfig != nil && appConfig.AppID != "" {
		logger.Info("producer-app-relationship-found", "appToScaleGuid", appConfig.AppID)
		//assuming the producer app has the same autoscaler service credentials as appToScale
		appConfig.Services = currentApp.Services
	} else { // metric producer =  appToScale (default case)
		appConfig = currentApp
	}
	appId := api.GUID(appConfig.AppID)
	autoscalerCredentials, err := getAutoscalerCredentials(appConfig)
	if err != nil {
		return err
	}

	var (
		autoscalerApiServerURL string
		client                 *http.Client
	)
	if useMtls {
		autoscalerApiServerURL = autoscalerCredentials.MtlsURL
		if autoscalerApiServerURL == "" {
			// fallback for the "build in" case
			autoscalerApiServerURL = autoscalerCredentials.URL
		}
		client, err = getCFInstanceIdentityCertificateClient()
	} else {
		autoscalerApiServerURL = autoscalerCredentials.URL
		client, err = getBasicAuthClient(autoscalerCredentials)
	}
	if err != nil {
		return err
	}

	apiClient, err := api.NewClient(autoscalerApiServerURL, autoscalerCredentials, api.WithClient(client))
	if err != nil {
		return err
	}

	metrics := createSingletonMetric(metricName, metricValue)
	logger.Info("sending metric to autoscaler for app", "appId", appId, "metricName", metricName, "metricValue", metricValue)
	params := api.V1AppsAppGuidMetricsPostParams{AppGuid: appId}

	return apiClient.V1AppsAppGuidMetricsPost(ctx, metrics, params)
}

func getBasicAuthClient(autoscalerCredentials api.CustomMetricsCredentials) (*http.Client, error) {
	return api.NewBasicAuthTransport(autoscalerCredentials).Client(), nil
}

func getAutoscalerCredentials(appConfig *cfenv.App) (api.CustomMetricsCredentials, error) {
	result := api.CustomMetricsCredentials{}

	customMetricEnv := os.Getenv("AUTO_SCALER_CUSTOM_METRIC_ENV")

	if customMetricEnv != "" {
		err := json.Unmarshal([]byte(customMetricEnv), &result)
		if err != nil {
			return result, err
		}
	} else {
		autoscalers, err := appConfig.Services.WithTag("app-autoscaler")
		if err != nil {
			return result, err
		}
		autoscalerCredentials := autoscalers[0].Credentials["custom_metrics"]

		err = mapstructure.Decode(autoscalerCredentials, &result)

		if err != nil {
			return result, err
		}
	}

	return result, nil
}

func createSingletonMetric(metricName string, metricValue float64) *api.Metrics {
	metric := api.Metric{
		Name:  metricName,
		Value: metricValue,
	}

	metrics := &api.Metrics{
		InstanceIndex: 0,
		Metrics:       []api.Metric{metric},
	}
	return metrics
}

func getCFInstanceIdentityCertificateClient() (*http.Client, error) {
	cfInstanceKey := os.Getenv("CF_INSTANCE_KEY")
	cfInstanceCert := os.Getenv("CF_INSTANCE_CERT")

	cert, err := tls.LoadX509KeyPair(cfInstanceCert, cfInstanceKey)
	if err != nil {
		return nil, fmt.Errorf("error loading CF instance identity x509 keypair: %w", err)
	}

	caCertBytes, err := os.ReadFile(cfInstanceCert)
	if err != nil {
		return nil, fmt.Errorf("error reading CF CA: %w", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCertBytes)

	//#nosec G402 -- test app that shall run on dev foundations without proper certs
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
		RootCAs:            caCertPool,
	}

	t := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	client := &http.Client{Transport: t}
	return client, nil
}
