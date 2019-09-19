package sbss_cred_helper

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCustomMetricsCredHelper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CustomMetricsCredHelper Suite")
}
