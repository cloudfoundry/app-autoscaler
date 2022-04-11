package policyvalidator_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPolicyvalidator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PolicyValidator Suite")
}
