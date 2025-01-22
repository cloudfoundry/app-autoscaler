package helpers_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"log"
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	"code.cloudfoundry.org/lager/v3/lagertest"
	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("HTTPClient", func() {
	var (
		fakeServer *ghttp.Server
		client     *http.Client
		logger     *lagertest.TestLogger
		err        error
	)

	BeforeEach(func() {
		fakeServer = ghttp.NewServer()
		fakeServer.RouteToHandler("GET", "/", ghttp.RespondWith(http.StatusOK, "successful"))
	})

	Describe("CreateHTTPSClient", func() {
		var (
			cfInstanceCertFile    string
			cfInstanceKeyFile     string
			cfInstanceCertContent []byte
			cfInstanceKeyContent  []byte
			notAfter              time.Time
			certTmpDir            string
			privateKey            *rsa.PrivateKey
		)

		JustBeforeEach(func() {
			privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
			Expect(err).ToNot(HaveOccurred())

			cfInstanceCertContent, err = testhelpers.GenerateClientCertWithPrivateKeyExpiring("org", "space", privateKey, notAfter)

			certTmpDir, err = os.MkdirTemp("", "cert-tests-*")
			Expect(err).NotTo(HaveOccurred())

			cfInstanceKeyContent = testhelpers.GenerateClientKeyWithPrivateKey(privateKey)

			cfInstanceCertFile, err = configutil.MaterializeContentInFile(certTmpDir, "eventgenerator.crt", string(cfInstanceCertContent))
			Expect(err).NotTo(HaveOccurred())

			cfInstanceKeyFile, err = configutil.MaterializeContentInFile(certTmpDir, "eventgenerator.key", string(cfInstanceKeyContent))
			Expect(err).NotTo(HaveOccurred())

			logger = lagertest.NewTestLogger("http-client-test")

			Expect(fileExists(cfInstanceCertFile)).To(BeTrue())
			Expect(fileExists(cfInstanceKeyFile)).To(BeTrue())

			tlsCerts := &models.TLSCerts{
				KeyFile:    cfInstanceKeyFile,
				CertFile:   cfInstanceCertFile,
				CACertFile: cfInstanceCertFile,
			}

			client, err = helpers.CreateHTTPSClient(tlsCerts, helpers.DefaultClientConfig(), logger)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			err = os.Remove(cfInstanceCertFile)
			Expect(err).ToNot(HaveOccurred())

			err = os.Remove(cfInstanceKeyFile)
			Expect(err).ToNot(HaveOccurred())
		})

		When("No cert is provided", func() {
			BeforeEach(func() {
				notAfter = time.Now().Add(7 * time.Minute)
			})

			It("should process request ok", func() {
				client.Transport.(*helpers.TLSReloadTransport).Base.(*retryablehttp.RoundTripper).Client.HTTPClient.Transport.(*http.Transport).TLSClientConfig = nil
				resp, err := client.Get(fakeServer.URL())
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})
		})

		When("Cert cert is not within 5m of expiration", func() {
			BeforeEach(func() {
				notAfter = time.Now().Add(7 * time.Minute)
			})

			It("should reload the cert", func() {
				Expect(client).ToNot(BeNil())
				resp, err := client.Get(fakeServer.URL())
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(logger).To(gbytes.Say("cert-not-expiring"))
			})
		})

		When("Cert cert is within 5m of expiration", func() {
			var cfInstanceCertFileToRotateContent []byte

			BeforeEach(func() {
				notAfter = time.Now().Add(3 * time.Minute)
			})

			It("should reload the cert", func() {
				cfInstanceCertFileToRotateContent, err = testhelpers.GenerateClientCertWithPrivateKey("org", "space", privateKey)
				Expect(err).ToNot(HaveOccurred())

				oldCertExpiration := getCertExpirationFromClient(client)

				By("Rotating with unexpired one")
				_, err = configutil.MaterializeContentInFile(certTmpDir, "eventgenerator.crt", string(cfInstanceCertFileToRotateContent))
				Expect(err).NotTo(HaveOccurred())

				Expect(getCertFromClient(client)).To(Equal(string(cfInstanceCertContent)))
				resp, err := client.Get(fakeServer.URL())
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(logger).To(gbytes.Say("reloading-cert"))
				newCertExpiration := getCertExpirationFromClient(client)
				Expect(newCertExpiration).To(BeTemporally(">", oldCertExpiration))
				Expect(getCertFromClient(client)).To(Equal(string(cfInstanceCertFileToRotateContent)))
			})
		})
	})
})

func getCertExpirationFromClient(client *http.Client) time.Time {
	GinkgoHelper()
	return client.Transport.(*helpers.TLSReloadTransport).GetCertExpiration()
}

func getCertFromClient(client *http.Client) string {
	GinkgoHelper()
	cert := client.Transport.(*helpers.TLSReloadTransport).Base.(*retryablehttp.RoundTripper).Client.HTTPClient.Transport.(*http.Transport).TLSClientConfig.Certificates[0]
	return getPEM(cert)
}

func getPEM(cert tls.Certificate) string {
	result := ""

	for _, certBytes := range cert.Certificate {
		parsedCert, err := x509.ParseCertificate(certBytes)
		if err != nil {
			log.Printf("Failed to parse certificate: %v", err)
			continue
		}

		// Encode to PEM format
		pemBlock := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: parsedCert.Raw,
		}
		result += string(pem.EncodeToMemory(pemBlock))
	}

	return result
}
func fileExists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}
