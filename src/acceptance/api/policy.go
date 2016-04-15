package api

import (
	"acceptance/helpers"
	"fmt"
)

type Policy struct {
	args []string
}

func NewPolicy(apiURL string, appGUID string) Policy {
	u := fmt.Sprintf("%s/v1/apps/%s/policy", apiURL, appGUID)

	return Policy{
		args: []string{"-H", "Accept: application/json", "-H", "Content-Type: application/json", "-H", "Authorization: " + helpers.OauthToken(), u},
	}
}

func (p Policy) Get() (int, error) {
	statusCode, _, err := helpers.Curl(p.args...)
	return statusCode, err
}

func (p Policy) UpdateWithText(policy string) (int, error) {
	statusCode, _, err := helpers.Curl(append([]string{"-X", "PUT", "--data-binary", policy}, p.args...)...)
	return statusCode, err
}

func (p Policy) Update(policyFile string) (int, error) {
	statusCode, _, err := helpers.Curl(append([]string{"-X", "PUT", "--data-binary", "@" + policyFile}, p.args...)...)
	return statusCode, err
}

func (p Policy) Delete() (int, error) {
	statusCode, _, err := helpers.Curl(append([]string{"-X", "DELETE"}, p.args...)...)
	return statusCode, err
}
