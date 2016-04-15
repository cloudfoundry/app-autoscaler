package api

import (
	"acceptance/helpers"
	"fmt"
)

type HistoryData struct {
	AppID     string
	Status    string
	StartTime int64
	EndTime   int64
	// Tigger
	TimeZone        string
	ErrorCode       string
	ScheduleType    string
	InstancesBefore int
	InstancesAfter  int
}

type HistoryResponse struct {
	Data      []HistoryData
	Timestamp int
}

func GetHistory(APIUrl string, appGUID string) (int, []byte, error) {
	historyURL := fmt.Sprintf("%s/v1/apps/%s/scalinghistory", APIUrl, appGUID)
	return helpers.Curl("-H", "Accept: application/json", "-H", "Authorization: "+helpers.OauthToken(), historyURL)
}
