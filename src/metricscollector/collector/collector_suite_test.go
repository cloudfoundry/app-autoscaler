package collector_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

//go:generate counterfeiter -o ./fakes/fake_cf_client.go ../cf CfClient
//go:generate counterfeiter -o ./fakes/fake_noaa_consumer.go . NoaaConsumer
//go:generate counterfeiter -o ./fakes/fake_DB.go ../db DB

// make sure TestPollInterval is less than TestRefreshInterval
const (
	TestPollInterval    = 1
	TestRefreshInterval = 3
)

func TestCollector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Collector Suite")
}
