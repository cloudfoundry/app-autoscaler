package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"testing"
)

func TestAppAutoScaler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "App-AutoScaler Suite")
}

var validPluginPath string

var _ = SynchronizedBeforeSuite(func() []byte {
	path, buildErr := Build("cli")
	Expect(buildErr).NotTo(HaveOccurred())
	return []byte(path)
}, func(data []byte) {
	validPluginPath = string(data)
})

// gexec.Build leaves a compiled binary behind in /tmp.
var _ = SynchronizedAfterSuite(func() {}, func() {
	CleanupBuildArtifacts()
})
