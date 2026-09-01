package activator_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestActivator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Activator Suite")
}
