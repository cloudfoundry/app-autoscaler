package auth_test

import (
	"encoding/base64"
	"encoding/pem"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

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
		xfccAuth := models.XFCCAuth{
			ValidOrgGuid:   orgGuid,
			ValidSpaceGuid: spaceGuid,
		}
		xm := auth.NewXfccAuthMiddleware(logger, xfccAuth)

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
			xfccClientCert, err = testhelpers.GenerateClientCert(orgGuid, spaceGuid)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 200", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})

	When("xfcc cert does not match org guid", func() {
		BeforeEach(func() {
			xfccClientCert, err = testhelpers.GenerateClientCert("wrong-org-guid", spaceGuid)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 401", func() {
			Eventually(buffer).Should(gbytes.Say(auth.ErrorWrongOrg.Error()))
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
		})

	})

	When("xfcc cert does not match space guid", func() {
		BeforeEach(func() {
			xfccClientCert, err = testhelpers.GenerateClientCert(orgGuid, "wrong-space-guid")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 401", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			Eventually(buffer).Should(gbytes.Say(auth.ErrorWrongSpace.Error()))
		})
	})
})
