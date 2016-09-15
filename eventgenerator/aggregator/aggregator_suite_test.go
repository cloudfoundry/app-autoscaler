package aggregator_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

const (
	testPolicyPollerInterval = 10 * time.Millisecond
)

func TestPolicyPoller(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Aggregator Suite")
}

//go:generate counterfeiter -o ./fakes/fake_policy_db.go ../../db PolicyDB
//go:generate counterfeiter -o ./fakes/fake_app_metric_db.go ../../db  AppMetricDB
