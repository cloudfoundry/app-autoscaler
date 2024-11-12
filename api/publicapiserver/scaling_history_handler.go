package publicapiserver

import (
	"context"
	"fmt"
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/apis/scalinghistory"
	internalscalingenginehistory "code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/apis/scalinghistory"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/lager/v3"
)

var (
	_ = scalinghistory.SecurityHandler(&SecuritySource{})
	_ = scalinghistory.SecuritySource(&SecuritySource{})
)

type SecuritySource struct{}

func (h *SecuritySource) BearerAuth(_ context.Context, _ string) (scalinghistory.BearerAuth, error) {
	// We are calling the scalingengine server authenticated via mTLS, so no bearer token is necessary.
	// Having this function is required by the interface `SecuritySource`in “oas_security_gen”.
	return scalinghistory.BearerAuth{Token: "none"}, nil
}

func (h SecuritySource) HandleBearerAuth(ctx context.Context, operationName string, t scalinghistory.BearerAuth) (context.Context, error) {
	// this handler is a no-op, as this handler shall only be available used behind our own auth middleware.
	// having this handler is required by the interface `securityhandler` in “oas_security_gen”.
	return ctx, nil
}

type ScalingHistoryHandler struct {
	logger lager.Logger
	conf   *config.Config
	client *internalscalingenginehistory.Client
}

func NewScalingHistoryHandler(logger lager.Logger, conf *config.Config) (*ScalingHistoryHandler, error) {
	seClient, err := helpers.CreateHTTPSClient(&conf.ScalingEngine.TLSClientCerts, helpers.DefaultClientConfig(), logger.Session("scaling_client"))
	if err != nil {
		return nil, fmt.Errorf("error creating scaling history HTTP client: %w", err)
	}
	newHandler := &ScalingHistoryHandler{
		logger: logger.Session("scaling-history-handler"),
		conf:   conf,
	}

	if client, err := internalscalingenginehistory.NewClient(conf.ScalingEngine.ScalingEngineUrl, internalscalingenginehistory.WithClient(seClient)); err != nil {
		return nil, fmt.Errorf("error creating ogen scaling history client: %w", err)
	} else {
		newHandler.client = client
	}

	return newHandler, nil
}

func (h *ScalingHistoryHandler) NewError(_ context.Context, _ error) *scalinghistory.ErrorStatusCode {
	result := &scalinghistory.ErrorStatusCode{}
	result.SetStatusCode(http.StatusInternalServerError)
	result.SetResponse(scalinghistory.ErrorResponse{
		Code:    scalinghistory.NewOptString(http.StatusText(http.StatusInternalServerError)),
		Message: scalinghistory.NewOptString("Error retrieving scaling history from scaling engine"),
	})
	return result
}

func (h *ScalingHistoryHandler) V1AppsGUIDScalingHistoriesGet(ctx context.Context, params scalinghistory.V1AppsGUIDScalingHistoriesGetParams) (*scalinghistory.History, error) {
	result := &scalinghistory.History{}
	logger := h.logger.Session("get-scaling-histories", helpers.AddTraceID(ctx, lager.Data{"app_guid": params.GUID}))
	logger.Info("start")
	defer logger.Info("end")

	internalParams := internalscalingenginehistory.V1AppsGUIDScalingHistoriesGetParams{
		GUID:      internalscalingenginehistory.GUID(params.GUID),
		StartTime: internalscalingenginehistory.OptInt(params.StartTime),
		EndTime:   internalscalingenginehistory.OptInt(params.EndTime),
		OrderDirection: internalscalingenginehistory.OptV1AppsGUIDScalingHistoriesGetOrderDirection{
			Value: internalscalingenginehistory.V1AppsGUIDScalingHistoriesGetOrderDirection(params.OrderDirection.Value),
			Set:   params.OrderDirection.Set,
		},
		Page:           internalscalingenginehistory.OptInt(params.Page),
		ResultsPerPage: internalscalingenginehistory.OptInt(params.ResultsPerPage),
	}
	internalResult, err := h.client.V1AppsGUIDScalingHistoriesGet(ctx, internalParams)
	if err != nil {
		logger.Error("get", err)
		return nil, err
	}
	jsonResult, err := internalResult.MarshalJSON()
	if err != nil {
		logger.Error("marshal", err)
		return nil, err
	}

	err = result.UnmarshalJSON(jsonResult)
	if err != nil {
		logger.Error("unmarshal", err)
		return nil, err
	}

	return result, err
}
