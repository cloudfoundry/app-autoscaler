package collector_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

const (
	TestCollectInterval time.Duration = 1 * time.Second
	TestRefreshInterval time.Duration = 2 * time.Second
	TestSaveInterval    time.Duration = 2 * time.Second
)

func TestCollector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Collector Suite")
}
