package cf_test

import (
	"os"
	"runtime"
	"strings"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("User Agent", func() {
	var (
		originalVersion string
		originalRepo    string
	)

	BeforeEach(func() {
		// Store original environment variables
		originalVersion = os.Getenv("DT_RELEASE_BUILD_VERSION")
		originalRepo = os.Getenv("GO_INSTALL_PACKAGE_SPEC")
	})

	AfterEach(func() {
		// Restore original environment variables
		if originalVersion != "" {
			os.Setenv("DT_RELEASE_BUILD_VERSION", originalVersion)
		} else {
			os.Unsetenv("DT_RELEASE_BUILD_VERSION")
		}
		if originalRepo != "" {
			os.Setenv("GO_INSTALL_PACKAGE_SPEC", originalRepo)
		} else {
			os.Unsetenv("GO_INSTALL_PACKAGE_SPEC")
		}
	})

	Context("GetUserAgent", func() {
		It("returns a valid user agent string", func() {
			userAgent := cf.GetUserAgent()

			Expect(userAgent).To(ContainSubstring("app-autoscaler/"))
			Expect(userAgent).To(ContainSubstring("Go/" + runtime.Version()))
			Expect(userAgent).To(ContainSubstring(runtime.GOOS + "/" + runtime.GOARCH))
		})

		It("has the expected format", func() {
			userAgent := cf.GetUserAgent()

			// Expected format: app-autoscaler/{version} ({repoURL}; {version}) Go/{goVersion} {os}/{arch}
			parts := strings.Split(userAgent, " ")
			Expect(len(parts)).To(BeNumerically(">=", 3))

			// Check product/version part
			productPart := parts[0]
			Expect(productPart).To(HavePrefix("app-autoscaler/"))

			// Check system info part is wrapped in parentheses
			systemInfoStart := strings.Index(userAgent, "(")
			systemInfoEnd := strings.Index(userAgent, ")")
			Expect(systemInfoStart).To(BeNumerically(">", 0))
			Expect(systemInfoEnd).To(BeNumerically(">", systemInfoStart))

			systemInfo := userAgent[systemInfoStart+1 : systemInfoEnd]
			Expect(systemInfo).To(ContainSubstring(";"))
		})

		It("uses environment variable for version when set", func() {
			os.Setenv("DT_RELEASE_BUILD_VERSION", "1.2.3")
			os.Setenv("GO_INSTALL_PACKAGE_SPEC", "github.com/example/repo")

			userAgent := cf.GetUserAgent()

			Expect(userAgent).To(ContainSubstring("app-autoscaler/1.2.3"))
			Expect(userAgent).To(ContainSubstring("github.com/example/repo; 1.2.3"))
		})

		It("uses default values when environment variables are not set", func() {
			os.Unsetenv("DT_RELEASE_BUILD_VERSION")
			os.Unsetenv("GO_INSTALL_PACKAGE_SPEC")

			userAgent := cf.GetUserAgent()

			Expect(userAgent).To(ContainSubstring("app-autoscaler/unknown"))
			Expect(userAgent).To(ContainSubstring("somehow related to github.com/cloudfoundry/app-autoscaler; unknown"))
		})

		It("includes platform information", func() {
			userAgent := cf.GetUserAgent()

			// Should include Go version and platform info
			Expect(userAgent).To(ContainSubstring("Go/"))
			Expect(userAgent).To(MatchRegexp(`\w+/\w+$`)) // os/arch at the end
		})

		Context("with different environment variable combinations", func() {
			It("uses custom version but default repo", func() {
				os.Setenv("DT_RELEASE_BUILD_VERSION", "2.0.0")
				os.Unsetenv("GO_INSTALL_PACKAGE_SPEC")

				userAgent := cf.GetUserAgent()

				Expect(userAgent).To(ContainSubstring("app-autoscaler/2.0.0"))
				Expect(userAgent).To(ContainSubstring("somehow related to github.com/cloudfoundry/app-autoscaler; 2.0.0"))
			})

			It("uses default version but custom repo", func() {
				os.Unsetenv("DT_RELEASE_BUILD_VERSION")
				os.Setenv("GO_INSTALL_PACKAGE_SPEC", "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/cmd/api")

				userAgent := cf.GetUserAgent()

				Expect(userAgent).To(ContainSubstring("app-autoscaler/unknown"))
				Expect(userAgent).To(ContainSubstring("code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/cmd/api; unknown"))
			})
		})
	})
})
