// +build tools

package main

import (
	_ "github.com/golangci/golangci-lint"
	_ "github.com/maxbrunsfeld/counterfeiter/v6"
	_ "github.com/onsi/ginkgo/ginkgo"
	_ "github.com/square/certstrap"
)
