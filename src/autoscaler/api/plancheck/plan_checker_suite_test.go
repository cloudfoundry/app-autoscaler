package plancheck_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestQuota(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plan Checker Suite")
}
