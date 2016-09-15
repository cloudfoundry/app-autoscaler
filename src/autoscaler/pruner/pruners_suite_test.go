package pruner_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

// make sure TestRefreshInterval is equal TestIntervalInHours
const (
	TestIntervalInHours int           = 12
	TestRefreshInterval time.Duration = 12 * time.Hour
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pruners Suite")
}
