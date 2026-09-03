package activator_test

import (
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/activator"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SelfBackendFromEnv", func() {
	var saved map[string]string

	setEnv := func(kv map[string]string) {
		for k, v := range kv {
			saved[k] = os.Getenv(k)
			_ = os.Setenv(k, v)
		}
	}

	BeforeEach(func() { saved = map[string]string{} })
	AfterEach(func() {
		for k, v := range saved {
			if v == "" {
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, v)
			}
		}
	})

	It("derives the host/tls_port/SAN from CF instance env", func() {
		setEnv(map[string]string{
			"CF_INSTANCE_IP":    "10.0.1.24",
			"CF_INSTANCE_GUID":  "abc-guid",
			"CF_INSTANCE_PORTS": `[{"internal":8080,"external_tls_proxy":61032,"internal_tls_proxy":61001},{"internal":2222,"external_tls_proxy":61033}]`,
		})

		b, err := activator.SelfBackendFromEnv()
		Expect(err).NotTo(HaveOccurred())
		Expect(b.Host).To(Equal("10.0.1.24"))
		Expect(b.TLSPort).To(Equal(61032)) // the web (8080) external_tls_proxy, not the ssh (2222) one
		Expect(b.ServerCertDomainSAN).To(Equal("abc-guid"))
		Expect(b.InstanceGUID).To(Equal("abc-guid"))
	})

	It("errors when CF instance env is absent", func() {
		setEnv(map[string]string{
			"CF_INSTANCE_IP":    "",
			"CF_INSTANCE_GUID":  "",
			"CF_INSTANCE_PORTS": "",
		})
		_, err := activator.SelfBackendFromEnv()
		Expect(err).To(HaveOccurred())
	})
})
