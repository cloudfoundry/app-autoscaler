package forwarder_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestForwarder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Forwarder Suite")
}
