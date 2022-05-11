package envelopeprocessor_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEnvelopeprocessor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Envelopeprocessor Suite")
}
