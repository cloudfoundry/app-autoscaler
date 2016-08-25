package aggregator_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

const (
	TestPolicyPollerInterval = 1
	TestMetricPollerCount    = 1
)

func TestPolicyPoller(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Aggregator Suite")
}
