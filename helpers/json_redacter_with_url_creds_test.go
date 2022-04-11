package helpers_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JSON Redacter With URL Cred", func() {
	var (
		resp         []byte
		err          error
		jsonRedacter *helpers.JSONRedacterWithURLCred
	)

	BeforeEach(func() {
		jsonRedacter, err = helpers.NewJSONRedacterWithURLCred([]string{"[Pp]wd", "[Pp]ass"}, []string{`AKIA[A-Z0-9]{16}`})
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when called with normal (non-secret) json", func() {
		BeforeEach(func() {
			resp = jsonRedacter.Redact([]byte(`{"foo":"bar","url":"https://api.bosh-lite.com"}`))
		})
		It("should return the same text", func() {
			Expect(resp).To(Equal([]byte(`{"foo":"bar","url":"https://api.bosh-lite.com"}`)))
		})
	})

	Context("when called with postgres db url with credentials", func() {
		BeforeEach(func() {
			resp = jsonRedacter.Redact([]byte(`{"dbURL":"postgresql://username:password@hostname:5432/dbname?sslmode=disabled","password":"secret!"}`))
		})
		It("Should only redact the password", func() {
			Expect(resp).To(Equal([]byte(`{"dbURL":"postgresql://username:*REDACTED*@hostname:5432/dbname?sslmode=disabled","password":"*REDACTED*"}`)))
		})
	})

	Context("when called with secrets inside the json", func() {
		BeforeEach(func() {
			resp = jsonRedacter.Redact([]byte(`{"foo":"fooval","password":"secret!","something":"AKIA1234567890123456"}`))
		})

		It("should redact the secrets", func() {
			Expect(resp).To(Equal([]byte(`{"foo":"fooval","password":"*REDACTED*","something":"*REDACTED*"}`)))
		})
	})

	Context("when a password flag is specified", func() {
		BeforeEach(func() {
			resp = jsonRedacter.Redact([]byte(`{"abcPwd":"abcd","password":"secret!","userpass":"fooval"}`))
		})

		It("should redact the secrets", func() {
			Expect(resp).To(Equal([]byte(`{"abcPwd":"*REDACTED*","password":"*REDACTED*","userpass":"*REDACTED*"}`)))
		})
	})

	Context("when called with an array of objects with a secret", func() {
		BeforeEach(func() {
			resp = jsonRedacter.Redact([]byte(`[{"dbURL":"postgresql://username:password@hostname:5432/dbname?sslmode=disabled","foo":"fooval","password":"secret!"}]`))
		})

		It("should redact the secrets", func() {
			Expect(resp).To(Equal([]byte(`[{"dbURL":"postgresql://username:*REDACTED*@hostname:5432/dbname?sslmode=disabled","foo":"fooval","password":"*REDACTED*"}]`)))
		})
	})
})
