package collector_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
	"time"
)

// make sure TestPollInterval is less than TestRefreshInterval
const (
	TestRefreshInterval time.Duration = 3 * time.Second
	TestCollectInterval time.Duration = 1 * time.Second
	TestSaveInterval    time.Duration = 1 * time.Second
)

func TestCollector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Collector Suite")
}
