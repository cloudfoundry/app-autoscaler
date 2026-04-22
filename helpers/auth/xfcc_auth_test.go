package auth_test

import (
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

func doRequest(server *httptest.Server, xfccClientCert []byte) *http.Response {
	server.Start()
	req, err := http.NewRequest("GET", server.URL+"/some-protected-endpoint", nil)
	Expect(err).NotTo(HaveOccurred())

	if len(xfccClientCert) > 0 {
		cert := auth.NewCert(string(xfccClientCert))
		req.Header.Add("X-Forwarded-Client-Cert", cert.GetXFCCHeader())
	}

	resp, err := server.Client().Do(req)
	Expect(err).NotTo(HaveOccurred())
	return resp
}

var _ = Describe("XfccAuthMiddleware", func() {
	var (
		resp *http.Response

		buffer *gbytes.Buffer

		err            error
		xfccClientCert []byte

		xm auth.XFCCAuthMiddleware

		expectedOrgGuid   = "validorg"
		expectedSpaceGuid = "validspace"
		server            *httptest.Server
	)

	BeforeEach(func() {
		logger := lagertest.NewTestLogger("xfcc-auth-test")
		buffer = logger.Buffer()
		xm = auth.NewXfccAuthMiddleware(logger, models.XFCCAuth{ValidOrgGuid: expectedOrgGuid, ValidSpaceGuid: expectedSpaceGuid})

		server = httptest.NewUnstartedServer(xm.XFCCAuthenticationMiddleware(handler))
	})

	JustBeforeEach(func() {
		resp = doRequest(server, xfccClientCert)
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
			xfccClientCert, err = testhelpers.GenerateClientCert(expectedOrgGuid, expectedSpaceGuid)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 200", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})

	When("xfcc cert does not match org guid", func() {
		BeforeEach(func() {
			xfccClientCert, err = testhelpers.GenerateClientCert("wrong-org-guid", expectedSpaceGuid)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 401", func() {
			Eventually(buffer).Should(gbytes.Say(auth.ErrorWrongOrg.Error()))
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
		})
	})

	When("xfcc cert does not match space guid", func() {
		BeforeEach(func() {
			xfccClientCert, err = testhelpers.GenerateClientCert(expectedOrgGuid, "wrong-space-guid")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 401", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			Eventually(buffer).Should(gbytes.Say(auth.ErrorWrongSpace.Error()))
		})
	})
})

var _ = Describe("MultiOrgXfccAuthMiddleware", func() {
	var (
		resp *http.Response

		buffer *gbytes.Buffer

		err            error
		xfccClientCert []byte

		xm auth.XFCCAuthMiddleware

		validOrgGuids     = []string{"org-1", "org-2", "org-3"}
		expectedSpaceGuid = "validspace"
		server            *httptest.Server
	)

	BeforeEach(func() {
		logger := lagertest.NewTestLogger("multi-org-xfcc-auth-test")
		buffer = logger.Buffer()
		xm = auth.NewMultiOrgXfccAuthMiddleware(logger, validOrgGuids, expectedSpaceGuid)

		server = httptest.NewUnstartedServer(xm.XFCCAuthenticationMiddleware(handler))
	})

	JustBeforeEach(func() {
		resp = doRequest(server, xfccClientCert)
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

	When("xfcc cert matches first org in list", func() {
		BeforeEach(func() {
			xfccClientCert, err = testhelpers.GenerateClientCert("org-1", expectedSpaceGuid)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 200", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})

	When("xfcc cert matches second org in list", func() {
		BeforeEach(func() {
			xfccClientCert, err = testhelpers.GenerateClientCert("org-2", expectedSpaceGuid)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 200", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})

	When("xfcc cert matches last org in list", func() {
		BeforeEach(func() {
			xfccClientCert, err = testhelpers.GenerateClientCert("org-3", expectedSpaceGuid)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 200", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})

	When("xfcc cert org does not match any in list", func() {
		BeforeEach(func() {
			xfccClientCert, err = testhelpers.GenerateClientCert("unknown-org", expectedSpaceGuid)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 401", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			Eventually(buffer).Should(gbytes.Say(auth.ErrorWrongOrg.Error()))
		})
	})

	When("xfcc cert does not match space guid", func() {
		BeforeEach(func() {
			xfccClientCert, err = testhelpers.GenerateClientCert("org-1", "wrong-space")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return 401", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			Eventually(buffer).Should(gbytes.Say(auth.ErrorWrongSpace.Error()))
		})
	})
})
