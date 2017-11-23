package commons

import (
	uuid "github.com/nu7hatch/gouuid"
        "code.cloudfoundry.org/lager"
)

func GenerateGUID(logger lager.Logger) (string, error) {
	guid, err := uuid.NewV4()
	if err != nil {
		logger.Fatal("Couldn't generate uuid", err)
		return "", err
	}
	return guid.String(), nil
}
