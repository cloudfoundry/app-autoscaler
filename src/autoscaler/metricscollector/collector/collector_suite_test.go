package collector_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
	"time"
)

// make sure TestPollInterval is less than TestRefreshInterval
const (
	TestPollInterval    time.Duration = 1 * time.Second
	TestRefreshInterval time.Duration = 3 * time.Second
	TestRetryTimes      uint          = 3
)

func TestCollector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Collector Suite")
}
