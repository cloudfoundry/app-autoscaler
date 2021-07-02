// +build tools

package main

import (
	_ "github.com/onsi/ginkgo/ginkgo"
	_ "github.com/square/certstrap"
	_ "github.com/maxbrunsfeld/counterfeiter/v6"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
)
