package operator_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

const (
	TestRefreshInterval = 12 * time.Hour
)

func TestOperators(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Operators Test Suite")
}
