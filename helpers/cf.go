package helpers

import (
	"encoding/json"
	"fmt"
)

func GetDbURLFromVcap(vcapServices string, dbType string) (string, error) {
	var vcap map[string][]struct {
		Credentials map[string]interface{} `json:"credentials"`
	}

	err := json.Unmarshal([]byte(vcapServices), &vcap)
	if err != nil {
		return "", err
	}

	creds := vcap[dbType][0].Credentials
	if creds == nil {
		return "", fmt.Errorf("credentials not found for %s", dbType)
	}

	dbURL := creds["uri"].(string)
	return dbURL, nil
}
