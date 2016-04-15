package api

import (
	"acceptance/helpers"
	"fmt"
)

func GetMetrics(APIUrl string, appGUID string) (int, []byte, error) {
	metricsURL := fmt.Sprintf("%s/v1/apps/%s/metrics", APIUrl, appGUID)
	return helpers.Curl("-H", "Accept: application/json", "-H", "Authorization: "+helpers.OauthToken(), metricsURL)
}
