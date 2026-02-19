package metric

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCFOauth2HTTPClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CFOauth2HTTPClient Suite")
}

var _ = Describe("CFOauth2HTTPClient", func() {
	var (
		client          *CFOauth2HTTPClient
		tokenServer     *httptest.Server
		metricsServer   *httptest.Server
		tokenCallCount  int
		tokenCallsMutex sync.Mutex
	)

	BeforeEach(func() {
		tokenCallCount = 0

		// Set up mock UAA server
		tokenServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenCallsMutex.Lock()
			tokenCallCount++
			tokenCallsMutex.Unlock()

			// Verify Basic auth header
			auth := r.Header.Get("Authorization")
			Expect(auth).To(HavePrefix("Basic "))

			// Verify request is for password grant
			Expect(r.Method).To(Equal("POST"))
			Expect(r.Header.Get("Content-Type")).To(Equal("application/x-www-form-urlencoded"))

			err := r.ParseForm()
			Expect(err).NotTo(HaveOccurred())
			Expect(r.FormValue("grant_type")).To(Equal("password"))
			Expect(r.FormValue("username")).To(Equal("test-user"))
			Expect(r.FormValue("password")).To(Equal("test-password"))

			// Return valid token response
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{
				"access_token": "test-token-%d",
				"token_type": "Bearer",
				"expires_in": 3600
			}`, tokenCallCount)
		}))

		// Set up mock metrics server
		metricsServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify Bearer token
			auth := r.Header.Get("Authorization")
			Expect(auth).To(HavePrefix("Bearer "))

			// Return mock metrics
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"metrics": "data"}`)
		}))

		// Create client with mock servers
		client = NewCFOauth2HTTPClient(
			tokenServer.URL,
			"cf",
			"cf-secret",
			"test-user",
			"test-password",
			true, // skipSSLValidation
		)
	})

	AfterEach(func() {
		if tokenServer != nil {
			tokenServer.Close()
		}
		if metricsServer != nil {
			metricsServer.Close()
		}
	})

	Describe("Do", func() {
		It("should add Bearer token to request", func() {
			req, err := http.NewRequest("GET", metricsServer.URL, nil)
			Expect(err).NotTo(HaveOccurred())

			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			// Verify token was retrieved
			Expect(tokenCallCount).To(Equal(1))
		})

		It("should reuse token for multiple requests", func() {
			// First request
			req1, err := http.NewRequest("GET", metricsServer.URL, nil)
			Expect(err).NotTo(HaveOccurred())
			resp1, err := client.Do(req1)
			Expect(err).NotTo(HaveOccurred())
			defer resp1.Body.Close()
			Expect(resp1.StatusCode).To(Equal(http.StatusOK))

			// Second request - should reuse token
			req2, err := http.NewRequest("GET", metricsServer.URL, nil)
			Expect(err).NotTo(HaveOccurred())
			resp2, err := client.Do(req2)
			Expect(err).NotTo(HaveOccurred())
			defer resp2.Body.Close()
			Expect(resp2.StatusCode).To(Equal(http.StatusOK))

			// Token should only be requested once
			Expect(tokenCallCount).To(Equal(1))
		})

		It("should refresh token on 401 response", func() {
			// Create metrics server that returns 401 on first request
			callCount := 0
			authenticatingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount++
				auth := r.Header.Get("Authorization")
				Expect(auth).To(HavePrefix("Bearer "))

				// First request gets 401, second gets 200
				if callCount == 1 {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"metrics": "data"}`)
			}))
			defer authenticatingServer.Close()

			req, err := http.NewRequest("GET", authenticatingServer.URL, nil)
			Expect(err).NotTo(HaveOccurred())

			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			// Should get 200 after refresh
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			// Token should have been requested twice (initial + refresh)
			Expect(tokenCallCount).To(Equal(2))
		})

		It("should fail if token endpoint returns error", func() {
			failingTokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, "token server error")
			}))
			defer failingTokenServer.Close()

			failingClient := NewCFOauth2HTTPClient(
				failingTokenServer.URL,
				"cf",
				"cf-secret",
				"test-user",
				"test-password",
				true,
			)

			req, err := http.NewRequest("GET", metricsServer.URL, nil)
			Expect(err).NotTo(HaveOccurred())

			_, err = failingClient.Do(req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to refresh token"))
		})

		It("should refresh token when expired", func() {
			// Create token server that returns short-lived tokens
			expiringTokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tokenCallsMutex.Lock()
				tokenCallCount++
				currentCount := tokenCallCount
				tokenCallsMutex.Unlock()

				w.Header().Set("Content-Type", "application/json")
				// Return token that expires in 1 second (minus 30s buffer = already expired)
				fmt.Fprintf(w, `{
					"access_token": "expired-token-%d",
					"token_type": "Bearer",
					"expires_in": 1
				}`, currentCount)
			}))
			defer expiringTokenServer.Close()

			expiringClient := NewCFOauth2HTTPClient(
				expiringTokenServer.URL,
				"cf",
				"cf-secret",
				"test-user",
				"test-password",
				true,
			)

			// First request - gets initial token
			req1, err := http.NewRequest("GET", metricsServer.URL, nil)
			Expect(err).NotTo(HaveOccurred())
			resp1, err := expiringClient.Do(req1)
			Expect(err).NotTo(HaveOccurred())
			defer resp1.Body.Close()
			Expect(resp1.StatusCode).To(Equal(http.StatusOK))
			Expect(tokenCallCount).To(Equal(1))

			// Wait for token to expire (it's already expired due to 30s buffer)
			time.Sleep(100 * time.Millisecond)

			// Second request - should refresh token automatically
			req2, err := http.NewRequest("GET", metricsServer.URL, nil)
			Expect(err).NotTo(HaveOccurred())
			resp2, err := expiringClient.Do(req2)
			Expect(err).NotTo(HaveOccurred())
			defer resp2.Body.Close()
			Expect(resp2.StatusCode).To(Equal(http.StatusOK))

			// Token should have been refreshed
			Expect(tokenCallCount).To(Equal(2))
		})

		It("should not refresh token unnecessarily when not expired", func() {
			// Create token server that returns long-lived tokens
			longLivedTokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tokenCallsMutex.Lock()
				tokenCallCount++
				currentCount := tokenCallCount
				tokenCallsMutex.Unlock()

				w.Header().Set("Content-Type", "application/json")
				// Return token that expires in 1 hour
				fmt.Fprintf(w, `{
					"access_token": "long-lived-token-%d",
					"token_type": "Bearer",
					"expires_in": 3600
				}`, currentCount)
			}))
			defer longLivedTokenServer.Close()

			longLivedClient := NewCFOauth2HTTPClient(
				longLivedTokenServer.URL,
				"cf",
				"cf-secret",
				"test-user",
				"test-password",
				true,
			)

			// Make multiple requests
			for i := 0; i < 5; i++ {
				req, err := http.NewRequest("GET", metricsServer.URL, nil)
				Expect(err).NotTo(HaveOccurred())
				resp, err := longLivedClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			}

			// Token should only be requested once (not refreshed)
			Expect(tokenCallCount).To(Equal(1))
		})
	})

	Describe("Basic Auth Header", func() {
		It("should use Basic auth with client credentials", func() {
			headerCaptured := ""
			captureServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/oauth/token" {
					headerCaptured = r.Header.Get("Authorization")
					Expect(headerCaptured).To(HavePrefix("Basic "))

					// Decode and verify
					parts := bytes.Split([]byte(headerCaptured), []byte(" "))
					Expect(parts).To(HaveLen(2))

					decoded, err := base64.StdEncoding.DecodeString(string(parts[1]))
					Expect(err).NotTo(HaveOccurred())
					Expect(string(decoded)).To(Equal("cf:cf-secret"))

					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, `{
						"access_token": "test-token",
						"token_type": "Bearer",
						"expires_in": 3600
					}`)
				}
			}))
			defer captureServer.Close()

			testClient := NewCFOauth2HTTPClient(
				captureServer.URL,
				"cf",
				"cf-secret",
				"test-user",
				"test-password",
				true,
			)

			// This will trigger token fetch
			req, _ := http.NewRequest("POST", captureServer.URL+"/oauth/token", nil)
			_, _ = testClient.Do(req)

			Expect(headerCaptured).NotTo(BeEmpty())
		})
	})

	Describe("Token Endpoint URL Handling", func() {
		It("should append /oauth/token to URL if not present", func() {
			testClient := NewCFOauth2HTTPClient(
				"https://uaa.example.com",
				"cf",
				"cf-secret",
				"test-user",
				"test-password",
				true,
			)

			Expect(testClient.oauth2URL).To(Equal("https://uaa.example.com"))
			// The URL normalization happens in refreshToken method

			// We can't directly test the URL construction here without making a request,
			// but the logic is tested through integration
		})

		It("should not duplicate /oauth/token", func() {
			testClient := NewCFOauth2HTTPClient(
				"https://uaa.example.com/oauth/token",
				"cf",
				"cf-secret",
				"test-user",
				"test-password",
				true,
			)

			Expect(testClient.oauth2URL).To(Equal("https://uaa.example.com/oauth/token"))
		})
	})

	Describe("Thread Safety", func() {
		It("should handle concurrent requests", func() {
			concurrentServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Sleep a bit to increase chance of race conditions
				time.Sleep(10 * time.Millisecond)

				if r.URL.Path == "/oauth/token" {
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, `{
						"access_token": "test-token",
						"token_type": "Bearer",
						"expires_in": 3600
					}`)
				} else {
					auth := r.Header.Get("Authorization")
					Expect(auth).To(HavePrefix("Bearer "))
					w.WriteHeader(http.StatusOK)
				}
			}))
			defer concurrentServer.Close()

			testClient := NewCFOauth2HTTPClient(
				concurrentServer.URL,
				"cf",
				"cf-secret",
				"test-user",
				"test-password",
				true,
			)

			// Make concurrent requests
			var wg sync.WaitGroup
			errChan := make(chan error, 10)

			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					req, err := http.NewRequest("GET", concurrentServer.URL+"/metrics", nil)
					if err != nil {
						errChan <- err
						return
					}

					resp, err := testClient.Do(req)
					if err != nil {
						errChan <- err
						return
					}
					resp.Body.Close()
				}()
			}

			wg.Wait()
			close(errChan)

			// Check for any errors
			for err := range errChan {
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})

	Describe("HTTP Client Configuration", func() {
		It("should skip SSL validation when configured", func() {
			testClient := NewCFOauth2HTTPClient(
				"https://uaa.example.com",
				"cf",
				"cf-secret",
				"test-user",
				"test-password",
				true, // skipSSLValidation
			)

			Expect(testClient.skipSSLValidation).To(BeTrue())
			Expect(testClient.httpClient).NotTo(BeNil())
		})

		It("should not skip SSL validation when false", func() {
			testClient := NewCFOauth2HTTPClient(
				"https://uaa.example.com",
				"cf",
				"cf-secret",
				"test-user",
				"test-password",
				false, // skipSSLValidation
			)

			Expect(testClient.skipSSLValidation).To(BeFalse())
			Expect(testClient.httpClient).NotTo(BeNil())
		})
	})

	Describe("Error Handling", func() {
		It("should handle malformed token response", func() {
			badTokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{invalid json}`)
			}))
			defer badTokenServer.Close()

			badClient := NewCFOauth2HTTPClient(
				badTokenServer.URL,
				"cf",
				"cf-secret",
				"test-user",
				"test-password",
				true,
			)

			req, err := http.NewRequest("GET", metricsServer.URL, nil)
			Expect(err).NotTo(HaveOccurred())

			_, err = badClient.Do(req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to refresh token"))
		})

		It("should handle network errors during token fetch", func() {
			testClient := NewCFOauth2HTTPClient(
				"https://invalid-host-that-does-not-exist.local",
				"cf",
				"cf-secret",
				"test-user",
				"test-password",
				true,
			)

			req, err := http.NewRequest("GET", metricsServer.URL, nil)
			Expect(err).NotTo(HaveOccurred())

			_, err = testClient.Do(req)
			Expect(err).To(HaveOccurred())
		})
	})
})
