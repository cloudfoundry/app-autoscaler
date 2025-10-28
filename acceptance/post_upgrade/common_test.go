package post_upgrade_test

import (
	"acceptance/helpers"
	"strings"
)

func GetAppInfo(org, space, appType string) (fullAppName string, appGuid string) {
	apps := helpers.GetApps(cfg, org, space, "autoscaler-")
	for _, app := range apps {
		if strings.Contains(app, appType) {
			appGuid, _ := helpers.GetAppGuid(cfg, app)
			return app, appGuid
		}
	}
	return "", ""
}
