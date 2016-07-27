package collector_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
	"time"
)

//go:generate counterfeiter -o ./fakes/fake_cf_client.go ../cf CfClient
//go:generate counterfeiter -o ./fakes/fake_noaa_consumer.go . NoaaConsumer
//go:generate counterfeiter -o ./fakes/fake_DB.go ../db DB
//go:generate counterfeiter -o ./fakes/fake_app_poller.go . AppPoller

// make sure TestPollInterval is less than TestRefreshInterval
const (
	TestPollInterval    time.Duration = 1 * time.Second
	TestRefreshInterval time.Duration = 3 * time.Second
)

func TestCollector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Collector Suite")
}
