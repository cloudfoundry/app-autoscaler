package fips_test

import (
	"testing"

	"acceptance/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	cfg *config.Config
)

const componentName = "FIPS Mode Suite"

func TestAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, componentName)
}
