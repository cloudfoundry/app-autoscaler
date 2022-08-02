package cf_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCfClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cf Suite")
}
