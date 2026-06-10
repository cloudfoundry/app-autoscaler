package main

import (
	"fmt"
	"testing"
	"time"
)

func TestCurlerCurl(t *testing.T) {
	curler := Curler{
		NumAllowedErrors: 10,
		NumRequests: 3000,
		Timeout: 10*time.Second,
		//Gap: 2*time.Second,
		Url: "https://api.autoscaler.app-runtime-interfaces.ci.cloudfoundry.org/v2/info",
		SkipSslValidation: true,
	}

	curler.Start()

	fmt.Printf("Errors %d/%d\n", curler.NumActualErrors, curler.NumAllowedErrors)
}