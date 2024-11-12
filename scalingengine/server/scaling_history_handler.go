package server

import (
	"context"
	"errors"
	"math"
	"net/url"
	"strconv"
	"strings"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"github.com/ogen-go/ogen/ogenerrors"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/apis/scalinghistory"
	"code.cloudfoundry.org/lager/v3"

	"net/http"
)

type SecuritySource struct{}

type ScalingHistoryHandler struct {
	logger          lager.Logger
	scalingEngineDB db.ScalingEngineDB
}

func NewScalingHistoryHandler(logger lager.Logger, scalingEngineDB db.ScalingEngineDB) (*ScalingHistoryHandler, error) {
	newHandler := &ScalingHistoryHandler{
		logger:          logger.Session("scaling-history-handler"),
		scalingEngineDB: scalingEngineDB,
	}

	return newHandler, nil
}

func (h *ScalingHistoryHandler) NewError(_ context.Context, err error) *scalinghistory.ErrorStatusCode {
	result := &scalinghistory.ErrorStatusCode{}

	if errors.Is(err, ogenerrors.ErrSecurityRequirementIsNotSatisfied) {
		result.SetStatusCode(http.StatusUnauthorized)
		result.SetResponse(scalinghistory.ErrorResponse{
			Code:    scalinghistory.NewOptString(http.StatusText(http.StatusUnauthorized)),
			Message: scalinghistory.NewOptString("missing authentication"),
		})
	} else {
		result.SetStatusCode(http.StatusInternalServerError)
		result.SetResponse(scalinghistory.ErrorResponse{
			Code:    scalinghistory.NewOptString(http.StatusText(http.StatusInternalServerError)),
			Message: scalinghistory.NewOptString(err.Error()),
		})
	}
	return result
}

func (h *ScalingHistoryHandler) V1AppsGUIDScalingHistoriesGet(ctx context.Context, params scalinghistory.V1AppsGUIDScalingHistoriesGetParams) (*scalinghistory.History, error) {
	appId := params.GUID
	// actually not necessary if a default is provided in the schema, however this is not exposed yet:
	// https://github.com/ogen-go/ogen/issues/966
	startTime := params.StartTime.Or(0)
	endTime := params.EndTime.Or(-1)

	//nolint:staticcheck // For backwards-compatibility with our CF CLI plugin we want to honor the deprecated parameter if is used
	if !params.OrderDirection.IsSet() && params.Order.IsSet() {
		// client is using deprecated order parameter
		if params.Order.Value == scalinghistory.V1AppsGUIDScalingHistoriesGetOrderAsc { //
			params.OrderDirection.SetTo(scalinghistory.V1AppsGUIDScalingHistoriesGetOrderDirectionAsc)
		} else {
			params.OrderDirection.SetTo(scalinghistory.V1AppsGUIDScalingHistoriesGetOrderDirectionDesc)
		}
	}

	orderDirection := params.OrderDirection.Or(scalinghistory.V1AppsGUIDScalingHistoriesGetOrderDirectionDesc)
	dbOrder := db.DESC
	if orderDirection == scalinghistory.V1AppsGUIDScalingHistoriesGetOrderDirectionAsc {
		dbOrder = db.ASC
	}
	includeAll := false
	page := params.Page.Or(1)
	resultsPerPage := params.ResultsPerPage.Or(50)

	parameters := url.Values{}
	parameters.Add("start-time", strconv.Itoa(startTime))
	parameters.Add("end-time", strconv.Itoa(endTime))
	parameters.Add("order-direction", string(orderDirection))
	parameters.Add("results-per-page", strconv.Itoa(resultsPerPage))

	logger := h.logger.Session("get-scaling-histories", helpers.AddTraceID(ctx, lager.Data{"parameters": parameters, "app-guid": appId}))
	logger.Info("start")
	defer logger.Info("end")

	count, err := h.scalingEngineDB.CountScalingHistories(ctx, string(appId), int64(startTime), int64(endTime), includeAll)
	if err != nil {
		logger.Error("failed-to-count-histories", err)
		return nil, errors.New("error counting scaling histories in database")
	}
	totalPages := int(math.Ceil(float64(count) / float64(resultsPerPage)))
	logger.Debug("count-results", lager.Data{"count": count, "totalPages": totalPages})

	histories, err := h.scalingEngineDB.RetrieveScalingHistories(ctx, string(appId), int64(startTime), int64(endTime), dbOrder, includeAll, page, resultsPerPage)
	if err != nil {
		logger.Error("failed-to-retrieve-histories", err)
		return nil, errors.New("error getting scaling histories from database")
	}

	resources := make([]scalinghistory.HistoryEntry, len(histories))

	for i, item := range histories {
		entry := scalinghistory.HistoryEntry{
			AppID:        scalinghistory.NewOptGUID(scalinghistory.GUID(item.AppId)),
			Status:       scalinghistory.NewOptHistoryEntryStatus(scalinghistory.HistoryEntryStatus(item.Status)),
			Timestamp:    scalinghistory.NewOptInt(int(item.Timestamp)),
			ScalingType:  scalinghistory.NewOptHistoryEntryScalingType(scalinghistory.HistoryEntryScalingType(item.ScalingType)),
			OldInstances: scalinghistory.NewOptInt64(int64(item.OldInstances)),
			NewInstances: scalinghistory.NewOptInt64(int64(item.NewInstances)),
			Reason:       scalinghistory.NewOptString(item.Reason),
			Message:      scalinghistory.NewOptString(item.Message),
		}

		switch item.Status {
		case models.ScalingStatusSucceeded:
			entry.SetOneOf(scalinghistory.NewHistorySuccessEntryHistoryEntrySum(scalinghistory.HistorySuccessEntry{}))
		case models.ScalingStatusIgnored:
			entry.SetOneOf(scalinghistory.NewHistoryIgnoreEntryHistoryEntrySum(scalinghistory.HistoryIgnoreEntry{IgnoreReason: scalinghistory.NewOptString(item.Message)}))
		case models.ScalingStatusFailed:
			entry.SetOneOf(scalinghistory.NewHistoryErrorEntryHistoryEntrySum(scalinghistory.HistoryErrorEntry{Error: scalinghistory.NewOptString(item.Error)}))
		}

		resources[i] = entry
	}

	prevURL := scalinghistory.OptURI{}
	if page > 1 {
		prevURL, err = getPageURL(appId, page-1, parameters)
		if err != nil {
			return nil, err
		}
	}

	nextURL := scalinghistory.OptURI{}
	if page < totalPages {
		nextURL, err = getPageURL(appId, page+1, parameters)
		if err != nil {
			return nil, err
		}
	}

	result := &scalinghistory.History{
		TotalResults: scalinghistory.NewOptInt64(int64(count)),
		TotalPages:   scalinghistory.NewOptInt64(int64(totalPages)),
		Page:         scalinghistory.NewOptInt64(int64(page)),
		PrevURL:      prevURL,
		NextURL:      nextURL,
		Resources:    resources,
	}

	return result, nil
}

func getPageURL(appId scalinghistory.GUID, page int, parameters url.Values) (scalinghistory.OptURI, error) {
	scalingHistoryURL, err := url.Parse(routes.ScalingHistoriesPath)
	if err != nil {
		return scalinghistory.OptURI{}, err
	}
	scalingHistoryURL.Path = strings.Replace(scalingHistoryURL.Path, "{guid}", string(appId), 1)

	pageURL := *scalingHistoryURL
	parameters.Set("page", strconv.Itoa(page))
	pageURL.RawQuery = parameters.Encode()
	return scalinghistory.NewOptURI(pageURL), nil
}
