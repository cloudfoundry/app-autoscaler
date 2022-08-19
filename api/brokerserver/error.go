package brokerserver

import (
	"fmt"
	"net/http"

	"code.cloudfoundry.org/lager"
	"golang.org/x/xerrors"
)

var _ error = &BrokerError{}
var _ xerrors.Wrapper = &BrokerError{}

type BrokerError struct {
	Status  int
	Message string
	Err     error
	Data    lager.Data
}

func (b BrokerError) sendResponse(w http.ResponseWriter, logger lager.Logger) {
	logger.Error(b.Message, b.Err, b.Data)
	writeErrorResponse(w, b.Status, b.Message)
}

func (b BrokerError) Error() string {
	wrapped := ""
	if b.Err != nil {
		wrapped = ": " + b.Err.Error()
	}
	message := b.Message
	if message == "" {
		message = "uninitialised"
	}
	return fmt.Sprintf("%s, statusCode(%d)%s", message, b.Status, wrapped)
}

func (b BrokerError) Unwrap() error {
	return b.Err
}
