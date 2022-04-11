package aggregator_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

const (
	testAggregatorExecuteInterval = 1 * time.Millisecond
	testPolicyPollerInterval      = 10 * time.Millisecond
	testSaveInterval              = 1 * time.Millisecond
)

func TestPolicyPoller(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Aggregator Suite")
}
