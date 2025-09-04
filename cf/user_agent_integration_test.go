package cf_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/v3/lagertest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("User Agent Integration", func() {
	var (
		client      *cf.Client
		server      *httptest.Server
		logger      *lagertest.TestLogger
		clock       *fakeclock.FakeClock
		conf        *cf.Config
		userAgentCh chan string
	)

	BeforeEach(func() {
		userAgentCh = make(chan string, 10)

		// Mock server to capture User-Agent headers
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userAgent := r.Header.Get("User-Agent")
			select {
			case userAgentCh <- userAgent:
			default:
			}

			// Mock different endpoints
			switch r.URL.Path {
			case "/oauth/token":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"access_token":"test-token","expires_in":3600}`))
			case "/introspect":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"active":true,"client_id":"test-client","scope":["read","write"]}`))
			case "/userinfo":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"user_id":"test-user-id"}`))
			case "/":
				// CF API root endpoint for getting endpoints
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				endpointsJSON := fmt.Sprintf(`{
					"links": {
						"cloud_controller_v3": {"href": "%s/v3"},
						"uaa": {"href": "%s"}
					}
				}`, server.URL, server.URL)
				_, _ = w.Write([]byte(endpointsJSON))
			case "/v3/apps/test-app-id":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"guid":"test-app-id","name":"test-app","relationships":{"space":{"data":{"guid":"test-space-id"}}}}`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		logger = lagertest.NewTestLogger("test")
		clock = fakeclock.NewFakeClock(time.Now())

		conf = &cf.Config{
			API:      server.URL,
			ClientID: "test-client",
			Secret:   "test-secret",
			ClientConfig: cf.ClientConfig{
				SkipSSLValidation:       true,
				MaxIdleConnsPerHost:     10,
				IdleConnectionTimeoutMs: 1000,
			},
		}

		client = cf.NewCFClient(conf, logger, clock)
	})

	AfterEach(func() {
		server.Close()
		close(userAgentCh)
	})

	Context("HTTP requests include custom user agent", func() {
		It("sets user agent for authentication requests", func() {
			err := client.Login()
			Expect(err).NotTo(HaveOccurred())

			Eventually(userAgentCh).Should(Receive(ContainSubstring("app-autoscaler/")))
		})

		It("sets user agent for token introspection requests", func() {
			_, err := client.IsTokenAuthorized("test-token", "test-client")
			Expect(err).NotTo(HaveOccurred())

			Eventually(userAgentCh).Should(Receive(ContainSubstring("app-autoscaler/")))
		})

		It("sets user agent for user info requests", func() {
			_, err := client.IsUserAdmin("test-token")
			Expect(err).NotTo(HaveOccurred())

			Eventually(userAgentCh).Should(Receive(ContainSubstring("app-autoscaler/")))
		})

		It("sets user agent for API requests through retriever", func() {
			_, err := client.GetEndpoints()
			Expect(err).NotTo(HaveOccurred())

			Eventually(userAgentCh).Should(Receive(ContainSubstring("app-autoscaler/")))
		})

		It("includes system information in user agent", func() {
			_ = client.Login()
			_, err := client.GetEndpoints()
			Expect(err).NotTo(HaveOccurred())

			var userAgent string
			Eventually(userAgentCh).Should(Receive(&userAgent))

			// Verify format: app-autoscaler/{version} ({repoURL}; {version}) Go/{goVersion} {os}/{arch}
			Expect(userAgent).To(ContainSubstring("app-autoscaler/"))
			Expect(userAgent).To(ContainSubstring("somehow related to github.com/cloudfoundry/app-autoscaler; unknown")) // Default values
			Expect(userAgent).To(ContainSubstring("Go/"))
			Expect(userAgent).To(MatchRegexp(`\w+/\w+$`)) // os/arch at the end
		})

		It("maintains user agent across multiple requests", func() {
			// Make multiple requests
			_ = client.Login()
			_, _ = client.GetEndpoints()
			_, _ = client.IsTokenAuthorized("test-token", "test-client")

			// Collect all user agents
			var userAgents []string
			for len(userAgents) < 3 {
				select {
				case ua := <-userAgentCh:
					if ua != "" {
						userAgents = append(userAgents, ua)
					}
				case <-time.After(100 * time.Millisecond):
					break
				}
			}

			// All user agents should be the same and contain app-autoscaler
			for _, ua := range userAgents {
				Expect(ua).To(ContainSubstring("app-autoscaler/"))
			}

			// Verify they are consistent
			if len(userAgents) > 1 {
				for i := 1; i < len(userAgents); i++ {
					Expect(userAgents[i]).To(Equal(userAgents[0]))
				}
			}
		})
	})
})
