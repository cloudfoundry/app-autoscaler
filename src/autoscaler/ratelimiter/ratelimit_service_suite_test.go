package ratelimiter_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRatelimitService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ratelimiter Suite")
}
