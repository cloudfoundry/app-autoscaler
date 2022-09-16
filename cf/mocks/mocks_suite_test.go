package mocks_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCfMocks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CF mocks Suite")
}
