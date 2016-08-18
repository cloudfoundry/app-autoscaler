package aggregator_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

const (
	TestPolicyPollerInterval = 1
)

func TestAggregator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PolicyPoller Suite")
}
