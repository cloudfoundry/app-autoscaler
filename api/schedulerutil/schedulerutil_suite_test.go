package schedulerutil_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestSchedulerutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Schedulerutil Suite")
}
