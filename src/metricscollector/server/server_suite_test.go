package server_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

//go:generate counterfeiter -o ./fakes/fake_cf_client.go ../cf CfClient
//go:generate counterfeiter -o ./fakes/fake_noaa_consumer.go . NoaaConsumer
func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}
