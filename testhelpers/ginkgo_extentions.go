package testhelpers

import (
	"fmt"

	"github.com/onsi/ginkgo/v2"
)

func FailOnError(message string, err error) {
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("%s: %s", message, err.Error()), 1)
	}
}
