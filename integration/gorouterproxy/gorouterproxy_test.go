package main_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Gorouterproxy", func() {
	var (
		session    *gexec.Session
		testserver *httptest.Server
		proxyPort  string
		orgGUID    string
		spaceGUID  string
	)

	BeforeEach(func() {
		orgGUID = "valid-org"
		spaceGUID = "valid-space"
		proxyPort = fmt.Sprintf("%d", 8888+GinkgoParallelProcess())

		testserver = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/valid-path" {
				http.NotFound(w, r)
				return
			}

			if r.Header.Get("X-Forwarded-Client-Cert") == "" {
				http.Error(w, "No xfcc header", http.StatusForbidden)
				return
			}

			err := auth.CheckAuth(r, orgGUID, spaceGUID)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "Hello, client")
		}))

		_, port, err := net.SplitHostPort(testserver.URL[len("http://"):])
		Expect(err).ShouldNot(HaveOccurred())

		testCertDir := "../../../../test-certs"
		cmd := exec.Command(cmdPath,
			"--port", proxyPort,
			"--forwardTo", port,
			"--certFile", filepath.Join(testCertDir, "gorouter.crt"),
			"--keyFile", filepath.Join(testCertDir, "gorouter.key"))
		session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ShouldNot(HaveOccurred())

	})

	AfterEach(func() {
		session.Kill().Wait()
		testserver.Close()
	})

	It("proxy request to test server and turns tls creds into xfcc header", func() {
		Eventually(session.Out, 20*time.Second).Should(gbytes.Say("gorouter-proxy.started"))

		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		Expect(err).ToNot(HaveOccurred())
		key := testhelpers.GenerateClientKeyWithPrivateKey(privateKey)

		cert, err := testhelpers.GenerateClientCertWithPrivateKey(orgGUID, spaceGUID, privateKey)
		Expect(err).ToNot(HaveOccurred())

		tlsCert, err := tls.X509KeyPair(cert, key)
		Expect(err).ToNot(HaveOccurred())

		c := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					Certificates: []tls.Certificate{tlsCert},
					//nolint:gosec // #nosec G402 -- due to https://github.com/securego/gosec/issues/1105
					InsecureSkipVerify: true,
				},
			},
		}

		Expect(proxyPort).ToNot(BeEmpty())
		resp, err := c.Get(fmt.Sprintf("https://127.0.0.1:%s/valid-path", proxyPort))
		Expect(err).ShouldNot(HaveOccurred())

		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		body, err := io.ReadAll(resp.Body)
		Expect(err).ShouldNot(HaveOccurred())

		Expect(string(body)).To(ContainSubstring("Hello, client"))
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})
})
