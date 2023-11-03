package publicapiserver

import (
	"context"
	"fmt"
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/apis/scalinghistory"
	"code.cloudfoundry.org/lager/v3"
)

var (
	_ = scalinghistory.SecurityHandler(&ScalingHistoryHandler{})
	_ = scalinghistory.SecuritySource(&ScalingHistoryHandler{})
)

type ScalingHistoryHandler struct {
	logger              lager.Logger
	conf                *config.Config
	scalingEngineClient *http.Client
	client              *scalinghistory.Client
}

func NewScalingHistoryHandler(logger lager.Logger, conf *config.Config) (*ScalingHistoryHandler, error) {
	seClient, err := helpers.CreateHTTPClient(&conf.ScalingEngine.TLSClientCerts, helpers.DefaultClientConfig(), logger.Session("scaling_client"))
	if err != nil {
		return nil, fmt.Errorf("error creating scaling history HTTP client: %w", err)
	}

	newHandler := &ScalingHistoryHandler{
		logger:              logger.Session("scaling-history-handler"),
		conf:                conf,
		scalingEngineClient: seClient,
	}

	if client, err := scalinghistory.NewClient(conf.ScalingEngine.ScalingEngineUrl, newHandler, scalinghistory.WithClient(seClient)); err != nil {
		return nil, fmt.Errorf("error creating ogen scaling history client: %w", err)
	} else {
		newHandler.client = client
	}

	return newHandler, nil
}

func (h *ScalingHistoryHandler) NewError(_ context.Context, _ error) *scalinghistory.ErrorResponseStatusCode {
	result := &scalinghistory.ErrorResponseStatusCode{}
	result.SetStatusCode(http.StatusInternalServerError)
	result.SetResponse(scalinghistory.ErrorResponse{
		Code:    scalinghistory.NewOptString(http.StatusText(http.StatusInternalServerError)),
		Message: scalinghistory.NewOptString("Error retrieving scaling history from scaling engine"),
	})
	return result
}

func (h *ScalingHistoryHandler) HandleBearerAuth(ctx context.Context, operationName string, t scalinghistory.BearerAuth) (context.Context, error) {
	// This handler is a no-op, as this handler shall only be available used behind our own auth middleware.
	// Having this handler is required by the interface `SecurityHandler` in “oas_security_gen”.
	return ctx, nil
}

func (h *ScalingHistoryHandler) V1AppsGUIDScalingHistoriesGet(ctx context.Context, params scalinghistory.V1AppsGUIDScalingHistoriesGetParams) (*scalinghistory.History, error) {
	logger := h.logger.Session("get-scaling-histories", helpers.AddTraceID(ctx, lager.Data{"app_guid": params.GUID}))
	logger.Info("start")
	defer logger.Info("end")

	result, err := h.client.V1AppsGUIDScalingHistoriesGet(ctx, params)
	if err != nil {
		logger.Error("get", err)
	}
	return result, err
}

func (h *ScalingHistoryHandler) BearerAuth(_ context.Context, _ string) (scalinghistory.BearerAuth, error) {
	// We are calling the scalingengine server authenticated via mTLS, so no bearer token is necessary.
	// Having this function is required by the interface `SecuritySource`in “oas_security_gen”.
	return scalinghistory.BearerAuth{Token: "none"}, nil
}
