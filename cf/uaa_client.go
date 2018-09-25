package cf

import (
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-incubator/uaago"
)

type UaaClient interface {
	RefreshAuthToken() (string, error)
}

type uaaClient struct {
	logger      lager.Logger
	conf        *CFConfig
	uaaClient   *uaago.Client
	UAAEndpoint string
}

func NewUaaClient(conf *CFConfig, logger lager.Logger, uaaEndpoint string) UaaClient {
	uc := &uaaClient{}
	uc.logger = logger
	uc.conf = conf
	uc.UAAEndpoint = uaaEndpoint

	uaaClient, err := uaago.NewClient(uaaEndpoint)
	if err != nil {
		uc.logger.Error("failed-to-create-uaago-client", err)
		os.Exit(1)
	}
	uc.uaaClient = uaaClient

	return uc
}

func (uc *uaaClient) RefreshAuthToken() (string, error) {
	token, err := uc.uaaClient.GetAuthToken(uc.conf.ClientID, uc.conf.Secret, uc.conf.SkipSSLValidation)
	if err != nil {
		return "", err
	}
	return token, nil
}
