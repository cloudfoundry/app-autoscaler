package helpers

import (
	"code.cloudfoundry.org/lager"
	uuid "github.com/nu7hatch/gouuid"
)

func GenerateGUID(logger lager.Logger) (string, error) {
	guid, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	return guid.String(), nil
}
