package auth_test

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("XFCC chain verification", func() {
	var (
		pki      *testhelpers.XFCCPKI
		verifier *auth.ChainVerifier
		logger   *lagertest.TestLogger
		now      time.Time
	)

	BeforeEach(func() {
		var err error
		pki, err = testhelpers.GenerateXFCCPKI(GinkgoT().TempDir())
		Expect(err).NotTo(HaveOccurred())
		logger = lagertest.NewTestLogger("xfcc-verification")
		now = time.Now()
	})

	JustBeforeEach(func() {
		var err error
		verifier, err = auth.NewChainVerifier(logger, pki.Config)
		Expect(err).NotTo(HaveOccurred())
		verifier.SetClockForTest(func() time.Time { return now })
	})

	It("verifies a leaf using the container root and mounted intermediate", func() {
		Expect(verifier.Verify(pki.Leaf)).To(Succeed())
	})

	It("rejects an intermediate with incompatible client-authentication purpose", func() {
		template := *pki.Issuer
		template.Extensions = nil
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		issuer, err := pki.Sign(&template, pki.Root)
		Expect(err).NotTo(HaveOccurred())
		Expect(testhelpers.WriteXFCCCertificates(pki.Config.InstanceCertPath, pki.Leaf, issuer)).To(Succeed())
		Expect(verifier.Verify(pki.Leaf)).NotTo(Succeed())
	})

	It("enforces the intermediate's path-length constraint", func() {
		template := *pki.Issuer
		template.Subject.CommonName = "additional-issuer"
		template.RawSubject = nil
		template.Extensions = nil
		extraIssuer, err := pki.Sign(&template, pki.Issuer)
		Expect(err).NotTo(HaveOccurred())
		leaf, err := pki.Sign(pki.Leaf, extraIssuer)
		Expect(err).NotTo(HaveOccurred())
		Expect(testhelpers.WriteXFCCCertificates(pki.Config.InstanceCertPath, leaf, extraIssuer, pki.Issuer)).To(Succeed())
		Expect(verifier.Verify(leaf)).NotTo(Succeed())
	})

	DescribeTable("checks client certificate policy", func(change func(*x509.Certificate), valid bool) {
		template := *pki.Leaf
		template.Extensions = nil
		change(&template)
		leaf, err := pki.Sign(&template, pki.Issuer)
		Expect(err).NotTo(HaveOccurred())
		if valid {
			Expect(verifier.Verify(leaf)).To(Succeed())
		} else {
			Expect(verifier.Verify(leaf)).To(MatchError(ContainSubstring(auth.ErrChainVerificationFailed.Error())))
		}
	},
		Entry("accepts absent EKU", func(cert *x509.Certificate) {
			cert.ExtKeyUsage = nil
		}, true),
		Entry("accepts any-purpose EKU", func(cert *x509.Certificate) {
			cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageAny}
		}, true),
		Entry("rejects server-only EKU", func(cert *x509.Certificate) {
			cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		}, false),
		Entry("rejects expired certificates", func(cert *x509.Certificate) {
			cert.NotAfter = time.Now().Add(-time.Minute)
		}, false),
		Entry("rejects certificates not yet valid", func(cert *x509.Certificate) {
			cert.NotBefore = time.Now().Add(time.Hour)
		}, false),
		Entry("rejects CA certificates as callers", func(cert *x509.Certificate) {
			cert.IsCA = true
			cert.BasicConstraintsValid = true
		}, false),
		Entry("rejects unsupported critical extensions", func(cert *x509.Certificate) {
			cert.ExtraExtensions = []pkix.Extension{
				{Id: []int{1, 2, 3, 4}, Critical: true, Value: []byte{5, 0}},
			}
		}, false),
	)

	Context("when the mounted chain is replaced before expiry", func() {
		It("refreshes the chain without rereading immutable roots", func() {
			Expect(verifier.Verify(pki.Leaf)).To(Succeed())
			Expect(os.Remove(filepath.Join(pki.Config.SystemCertPath, "root.pem"))).To(Succeed())
			other, err := testhelpers.GenerateXFCCPKI(GinkgoT().TempDir())
			Expect(err).NotTo(HaveOccurred())
			Expect(testhelpers.WriteXFCCCertificates(pki.Config.InstanceCertPath, other.Leaf, other.Issuer)).To(Succeed())

			Expect(verifier.Verify(pki.Leaf)).To(Succeed())
			now = now.Add(30 * time.Second)
			Expect(verifier.Verify(pki.Leaf)).NotTo(Succeed())
		})
	})

	Context("when a refresh fails", func() {
		It("fails closed and throttles retries until the next interval", func() {
			Expect(verifier.Verify(pki.Leaf)).To(Succeed())
			Expect(os.Remove(pki.Config.InstanceCertPath)).To(Succeed())
			Expect(verifier.Verify(pki.Leaf)).To(Succeed())
			now = now.Add(30 * time.Second)
			Expect(verifier.Verify(pki.Leaf)).NotTo(Succeed())

			Expect(testhelpers.WriteXFCCCertificates(pki.Config.InstanceCertPath, pki.Leaf, pki.Issuer)).To(Succeed())
			Expect(verifier.Verify(pki.Leaf)).NotTo(Succeed())
			now = now.Add(30 * time.Second)
			Expect(verifier.Verify(pki.Leaf)).To(Succeed())
		})
	})

	It("safely publishes the cache to concurrent requests", func() {
		var requests sync.WaitGroup
		for range 20 {
			requests.Go(func() {
				defer GinkgoRecover()
				Expect(verifier.Verify(pki.Leaf)).To(Succeed())
			})
		}
		requests.Wait()
	})

	DescribeTable("rejects incomplete or non-CA instance chains", func(shape string) {
		switch shape {
		case "empty":
			Expect(testhelpers.WriteXFCCCertificates(pki.Config.InstanceCertPath)).To(Succeed())
		case "leaf-only":
			Expect(testhelpers.WriteXFCCCertificates(pki.Config.InstanceCertPath, pki.Leaf)).To(Succeed())
		case "non-CA":
			Expect(testhelpers.WriteXFCCCertificates(pki.Config.InstanceCertPath, pki.Leaf, pki.Leaf)).To(Succeed())
		}
		Expect(verifier.Verify(pki.Leaf)).NotTo(Succeed())
	},
		Entry("empty file", "empty"),
		Entry("missing intermediate", "leaf-only"),
		Entry("non-CA intermediate", "non-CA"),
	)

	Context("when roots are unavailable", func() {
		BeforeEach(func() {
			pki.Config.SystemCertPath = filepath.Join(GinkgoT().TempDir(), "missing")
		})

		It("rejects an otherwise valid leaf", func() {
			Expect(verifier.Verify(pki.Leaf)).NotTo(Succeed())
		})

		Context("in shadow mode", func() {
			BeforeEach(func() {
				pki.Config.VerificationMode = "shadow"
			})

			It("continues and throttles warnings", func() {
				Expect(verifier.Verify(pki.Leaf)).To(Succeed())
				Expect(verifier.Verify(pki.Leaf)).To(Succeed())
				Expect(logger.Logs()).To(HaveLen(1))
				now = now.Add(30 * time.Second)
				Expect(verifier.Verify(pki.Leaf)).To(Succeed())
				Expect(logger.Logs()).To(HaveLen(2))
			})
		})

		Context("in off mode", func() {
			BeforeEach(func() {
				pki.Config.VerificationMode = "off"
			})

			It("does not read or require certificates", func() {
				Expect(verifier.Verify(nil)).To(Succeed())
			})
		})
	})

	Context("when root files contain multiple PEM blocks", func() {
		BeforeEach(func() {
			Expect(os.Mkdir(filepath.Join(pki.Config.SystemCertPath, "subdir"), 0700)).To(Succeed())
			Expect(testhelpers.WriteXFCCCertificates(filepath.Join(pki.Config.SystemCertPath, "roots.pem"), pki.Root, pki.Leaf)).To(Succeed())
			otherBlock := pem.EncodeToMemory(&pem.Block{Type: "OTHER", Bytes: []byte("not a certificate")})
			Expect(os.WriteFile(filepath.Join(pki.Config.SystemCertPath, "other.pem"), otherBlock, 0600)).To(Succeed())
		})

		It("ignores directories, non-certificate blocks and non-CA certificates", func() {
			Expect(verifier.Verify(pki.Leaf)).To(Succeed())
		})
	})

	DescribeTable("fails closed with invalid root files", func(shape string) {
		rootFile := filepath.Join(pki.Config.SystemCertPath, "root.pem")
		Expect(os.Remove(rootFile)).To(Succeed())
		switch shape {
		case "non-CA":
			Expect(testhelpers.WriteXFCCCertificates(rootFile, pki.Leaf)).To(Succeed())
		case "bad DER":
			badPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("bad DER")})
			Expect(os.WriteFile(rootFile, badPEM, 0600)).To(Succeed())
		case "unreadable":
			Expect(os.Symlink(filepath.Join(pki.Config.SystemCertPath, "missing"), rootFile)).To(Succeed())
		}
		fresh, err := auth.NewChainVerifier(logger, pki.Config)
		Expect(err).NotTo(HaveOccurred())
		Expect(fresh.Verify(pki.Leaf)).NotTo(Succeed())
	},
		Entry("no roots", "empty"),
		Entry("only a caller leaf", "non-CA"),
		Entry("malformed certificate", "bad DER"),
		Entry("unreadable file", "unreadable"),
	)

	It("uses environment paths when no explicit paths are configured", func() {
		GinkgoT().Setenv("CF_SYSTEM_CERT_PATH", pki.Config.SystemCertPath)
		GinkgoT().Setenv("CF_INSTANCE_CERT", pki.Config.InstanceCertPath)
		fresh, err := auth.NewChainVerifier(logger, models.XFCCAuth{})
		Expect(err).NotTo(HaveOccurred())
		Expect(fresh.Verify(pki.Leaf)).To(Succeed())
	})

	It("falls back to the standard container paths when the environment is empty", func() {
		GinkgoT().Setenv("CF_SYSTEM_CERT_PATH", "")
		GinkgoT().Setenv("CF_INSTANCE_CERT", "")
		_, err := auth.NewChainVerifier(logger, models.XFCCAuth{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("rejects a non-PEM value in the certificate header helper", func() {
		Expect(auth.NewCert("not PEM")).To(BeNil())
	})

	DescribeTable("rejects malformed headers before verifying chains", func(header string) {
		middleware, err := auth.NewXfccAuthMiddleware(logger, pki.Config)
		Expect(err).NotTo(HaveOccurred())
		request := httptest.NewRequest("GET", "/protected", nil)
		request.Header.Set("X-Forwarded-Client-Cert", header)
		response := httptest.NewRecorder()
		middleware.XFCCAuthenticationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Fail("Malformed XFCC reached the protected handler")
		})).ServeHTTP(response, request)
		Expect(response.Code).To(Equal(http.StatusUnauthorized))
		Expect(auth.CheckAuth(request, "validorg", "validspace")).NotTo(Succeed())
	},
		Entry("missing XFCC", ""),
		Entry("invalid base64", "@@@"),
		Entry("invalid DER", base64.StdEncoding.EncodeToString([]byte("not a certificate"))),
	)

	It("preserves the identity-only CheckAuth compatibility helper", func() {
		request := httptest.NewRequest("GET", "/protected", nil)
		request.Header.Set("X-Forwarded-Client-Cert", base64.StdEncoding.EncodeToString(pki.Leaf.Raw))
		Expect(auth.CheckAuth(request, "validorg", "validspace")).To(Succeed())
		template := *pki.Leaf
		template.Subject.OrganizationalUnit = nil
		template.RawSubject = nil
		leaf, err := pki.Sign(&template, pki.Issuer)
		Expect(err).NotTo(HaveOccurred())
		request.Header.Set("X-Forwarded-Client-Cert", base64.StdEncoding.EncodeToString(leaf.Raw))
		Expect(auth.CheckAuth(request, "validorg", "")).To(MatchError(auth.ErrorWrongOrg))
	})

	It("rejects unknown verification modes at construction", func() {
		_, err := auth.NewXfccAuthMiddleware(logger, models.XFCCAuth{VerificationMode: "invalid"})
		Expect(err).To(MatchError(ContainSubstring("invalid XFCC verification mode")))
	})

	DescribeTable("applies verification before the middleware identity checks", func(mode string, trusted bool, expectedStatus int) {
		config := pki.Config
		config.VerificationMode = mode
		middleware, err := auth.NewXfccAuthMiddleware(logger, config)
		Expect(err).NotTo(HaveOccurred())
		leaf := pki.Leaf
		if !trusted {
			foreign, err := testhelpers.GenerateXFCCPKI(GinkgoT().TempDir())
			Expect(err).NotTo(HaveOccurred())
			leaf = foreign.Leaf
		}
		request := httptest.NewRequest("GET", "/protected", nil)
		request.Header.Set("X-Forwarded-Client-Cert", base64.StdEncoding.EncodeToString(leaf.Raw))
		response := httptest.NewRecorder()
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		middleware.XFCCAuthenticationMiddleware(next).ServeHTTP(response, request)
		Expect(response.Code).To(Equal(expectedStatus))
	},
		Entry("default mode accepts trusted callers", "", true, http.StatusNoContent),
		Entry("default mode rejects foreign callers", "", false, http.StatusUnauthorized),
		Entry("enforce accepts trusted callers", "enforce", true, http.StatusNoContent),
		Entry("enforce rejects foreign callers", "enforce", false, http.StatusUnauthorized),
		Entry("shadow preserves otherwise valid identity", "shadow", false, http.StatusNoContent),
		Entry("off preserves otherwise valid identity", "off", false, http.StatusNoContent),
	)
})
