package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"

	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

var _ = Describe("XfccAuthMiddleware", func() {
	var (
		server *httptest.Server
		resp   *http.Response

		buffer *gbytes.Buffer

		err            error
		xfccClientCert []byte

		orgGuid   string
		spaceGuid string
	)

	AfterEach(func() {
		server.Close()
	})

	JustBeforeEach(func() {
		logger := lagertest.NewTestLogger("xfcc-auth-test")
		buffer = logger.Buffer()
		xm := auth.NewXfccAuthMiddleware(logger, orgGuid, spaceGuid)

		server = httptest.NewServer(xm.XFCCAuthenticationMiddleware(handler))

		req, err := http.NewRequest("GET", server.URL+"/some-protected-endpoint", nil)

		if len(xfccClientCert) > 0 {
			block, _ := pem.Decode(xfccClientCert)
			Expect(err).NotTo(HaveOccurred())
			Expect(block).ShouldNot(BeNil())

			req.Header.Add("X-Forwarded-Client-Cert", base64.StdEncoding.EncodeToString(block.Bytes))
		}
		Expect(err).NotTo(HaveOccurred())

		resp, err = http.DefaultClient.Do(req)
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		orgGuid = "org-guid"
		spaceGuid = "space-guid"
	})

	When("xfcc header is not set", func() {
		BeforeEach(func() {
			xfccClientCert = []byte{}
		})

		It("should return 401", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			Eventually(buffer).Should(gbytes.Say(auth.ErrXFCCHeaderNotFound.Error()))
		})
	})

	When("xfcc cert matches org and space guids", func() {
		BeforeEach(func() {
			xfccClientCert, err = generateClientCert(orgGuid, spaceGuid)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 200", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})

	When("xfcc cert does not match org guid", func() {
		BeforeEach(func() {
			xfccClientCert, err = generateClientCert("wrong-org-guid", spaceGuid)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 401", func() {
			Eventually(buffer).Should(gbytes.Say(auth.ErrorWrongOrg.Error()))
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
		})

	})

	When("xfcc cert does not match space guid", func() {
		BeforeEach(func() {
			xfccClientCert, err = generateClientCert(orgGuid, "wrong-space-guid")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 401", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			Eventually(buffer).Should(gbytes.Say(auth.ErrorWrongSpace.Error()))
		})
	})
})

// generateClientCert generates a client certificate with the specified spaceGUID and orgGUID
// included in the organizational unit string.
func generateClientCert(orgGUID, spaceGUID string) ([]byte, error) {
	// Generate a random serial number for the certificate
	//
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// Create a new X.509 certificate template
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{"My Organization"},
			OrganizationalUnit: []string{fmt.Sprintf("space:%s org:%s", spaceGUID, orgGUID)},
		},
	}
	// Generate the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, err
	}

	// Encode the certificate to PEM format
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	return certPEM, nil
}
